# File Storage

S3-compatible object storage (Garage in production, MinIO in tests) for the three
asset kinds Fractale persists outside Dgraph: comment attachments, user avatars and
org (root Node) avatars. Bytes never transit through Fractale — a REST proxy
re-authorises on every read and redirects to a short-lived presigned URL.

## Where things live

| Concern | File |
|---|---|
| Schema (`File`, anchor edges) | `schema/graphql/fractal6.graphql` |
| S3 client wrapper | `internal/storage/s3.go` |
| DQL templates / mutations | `db/dql_templates.go`, `db/dql_mutations.go` |
| DB helpers and decoders | `db/files.go` |
| HTTP handlers (`/file/*`) + markdown rewrite | `web/handlers/files.go` |
| Inbound `cid:` matching | `web/handlers/cid.go` |
| Email attachments (partition, fetch, rewrite) | `web/email/attachments.go` |
| Route wiring | `cmd/server.go` |
| Storage init + process-wide global | `cmd/health.go::initStorage` |
| Config | `[storage]` / `[notify]` blocks in `config.toml` |

## Anchors

Every `File` row has exactly one anchor, picked from the populated form fields:

| Form fields | Kind | Read auth | Write auth |
|---|---|---|---|
| `tid` + `cid` | comment attachment | parent tension's receiver visibility | comment author, via EMAP `CommentPushed` |
| `userid` | user avatar | public | self |
| `orgaid` | org avatar | node visibility | coordinator |

Deletion is uploader-only for all kinds. `File.tension` is denormalised alongside
`File.comment` so read auth is a single DQL hop. Avatars are replace-on-upload: the
previous row is dropped and its S3 object GC'd asynchronously.

Mutations on `File` are root-only in the GraphQL schema — all writes go through the
REST endpoints, which own the auth checks and persist via internal DQL. `queryFile` /
`getFile` are deliberately left open: `storageKey` is useless without going through
`/file/<id>`, which is where the auth check lives.

## REST surface

`POST /file/upload` (multipart, exactly one anchor), `GET /file/<id>` (302 to a
presigned URL), `DELETE /file/<id>`. Misses and unauthorised reads both return 404
so existence never leaks. Handlers take an injected `*storage.Client`; when
`[storage]` is unset they return 503 instead of panicking.

`initStorage` also registers the client process-wide (`storage.SetGlobal`) so
non-handler callers — the cascade-delete wrappers in `db/dql_mutations.go` and the
email attachment builder — reach it without a parameter chain. **Both binaries must
call it**: the api server for `/file/*`, the notifier daemon for email attachments.
`storage.Global()` returns nil when `[storage]` is unset and every caller nil-checks.

### Upload validation

Client-supplied metadata is not trusted: the Content-Type is re-sniffed server-side,
inline rendering is an explicit allowlist (SVG and HTML are forced to
`Content-Disposition: attachment`), size is capped, and every client-supplied uid
goes through `db.ValidateUids` before reaching a DQL `uid()` root — an unvalidated id
would let a caller widen a query or a delete. See `web/handlers/files.go`.

## Storage layout

One bucket, namespaced by key prefix: comment attachments under
`orgas/<rootnameid>/tensions/<tid>/<cid>/`, org avatars under `orgas/<rootnameid>/`,
user avatars under `users/<username>/`. Attachments live under their owning org so
per-org bucket policies can be expressed as a prefix filter. Keys carry a 16-hex-char
random prefix against guessing and collisions.

## Inline screenshot pasting

On paste the frontend embeds `![alt](<unique-filename>)` in the message and uploads
the bytes *after* the carrier comment exists. The upload handler finds the bare
filename token in `Comment.message`, rewrites it to `/file/<id>` and flips
`File.embedded=true`. Code regions (fences, inline backticks) are masked before
matching.

The rewrite is a read-modify-write: concurrent uploads to the same comment race on
`Comment.message` (last write wins). Both `File` rows persist; only the inline URL
substitution can be lost, and the UI renders orphaned `embedded` files as regular
attachment chips. Frontends should serialise per-comment uploads and use unique
filenames.

## Email notifications

Notification emails ship the comment's files in the Postal payload, since a mail
client's anonymous fetch can't satisfy `/file/<id>` for Private/Secret orgs. Embedded
images within the caps become RFC 2392 inline attachments (`cid:` src rewrite,
allowlisted in the bluemonday policy); everything else ships as a plain attachment
plus a footer link. Per-file, total and count caps live under `notify.*` in
`config.toml`; anything dropped falls back to an absolute `/file/<id>` URL.

Attachments are fetched once per notification and shared across recipients, so S3
traffic doesn't scale with recipient count.

### Upload settle poll

The notifier daemon can fire while uploads are still in flight. The message is the
ground truth — a successful inline upload rewrites the bare token — so
`getLastCommentSettled` (`graph/notifications.go`) re-polls the comment until no bare
`![](paste.png)` tokens remain, or the attempt budget runs out. No cross-process
coordination; past the budget the email degrades exactly like a slow upload with no
poll at all.

### Inbound email replies

`POST /notifications` (`web/handlers/mailer.go`) turns an email reply into a Comment.
Postal ships attachments with no Content-ID, so `processInboundAttachments` resolves
`cid:` references in three passes: quoted-back outbound CIDs (rewrite only), filename
heuristic, then document-order pairing. Unmatched refs are dropped; unmatched
attachments are persisted as plain paperclips. All decisions roll up into a single
`EmbedCommentMessage` upsert.

Writes are sequential here, so no settle poll is needed. Every attachment failure is
logged and skipped — the comment itself always commits. Authorship trusts `From:`,
gated by Postal's webhook signature. Contract replies are text-only for now.

## Deployment and operations

Self-hosted Garage: Ansible role at `contrib/ansible/roles/garage/`. Dev fixtures
(keys, ports) live in `docker-compose.dev.yml` and `contrib/garage/garage.toml`;
inspect a running instance with the `garage` CLI (admin plane) or `mc` / `aws`
(object listing).

Two config traps, both in the `[storage]` block:

- `endpoint` is the data-plane address *the backend* uses and may be private. When
  browsers can't reach it, set `public_url_prefix` to the public proxy — presigned
  URLs are then signed with that host, since SigV4 covers the `Host` header. The proxy
  must forward `Host` unchanged or Garage answers `SignatureDoesNotMatch`.
- Raising `max_upload_bytes` also requires raising the body limit of any reverse proxy
  in front (`client_max_body_size` on nginx).

Known ceilings:

- **Crash-orphan leak**: a death between the S3 `Put` and the DB insert orphans the
  bytes with no GC. Bounded, no broken links; sweep by key prefix if it ever matters.
- **No revocation**: a captured presigned URL stays valid until its TTL expires
  (`presign_ttl_sec`, 2h by default — a wide window; rotating the credentials is the
  only way to kill outstanding URLs, and it kills all of them).
- **No byte-access audit**: the backend sees the presigned request, not the user. Swap
  the redirect in `FileGet` for a proxy stream if that changes.
- **Cascade-delete GC** is fire-and-forget (`deleteStorageKeysAsync`), driven by the
  `Delete*Deep` wrappers; with storage unset the DQL still drops the `File` nodes and
  the objects are left orphaned.
