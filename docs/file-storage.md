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

Deletion is uploader-only. `File.tension` is denormalised alongside `File.comment` so
read auth is a single DQL hop (contract comments carry the contract's tension).
Avatars are replace-on-upload; the previous S3 object is GC'd asynchronously.

Mutations on `File` are root-only in the GraphQL schema — all writes go through the
REST endpoints, which own the auth checks and persist via internal DQL. `queryFile` /
`getFile` are deliberately left open: `storageKey` is useless without going through
`/file/<id>`, which is where the auth check lives.

## REST surface

`POST /file/upload` (multipart, exactly one anchor), `GET /file/<id>` (302 to a
presigned URL), `DELETE /file/<id>`. Misses and unauthorised reads both return 404
so existence never leaks. Handlers take an injected `*storage.Client`; when
`[storage]` is unset they return 503 instead of panicking.

`initStorage` also registers the client process-wide (`storage.SetGlobal`) for
non-handler callers (cascade deletes, email attachment builder). **Both binaries must
call it** — api server for `/file/*`, notifier for email attachments;
`storage.Global()` is nil when `[storage]` is unset and every caller nil-checks.

### Upload validation

Client-supplied metadata is not trusted: Content-Type is re-sniffed server-side,
inline rendering is an explicit allowlist (SVG and HTML forced to
`Content-Disposition: attachment`), size is capped, and every client-supplied uid goes
through `db.ValidateUids` before reaching a DQL `uid()` root — an unvalidated id would
let a caller widen a query or a delete. See `web/handlers/files.go`.

## Storage layout

One bucket, namespaced by key prefix: comment attachments under
`orgas/<rootnameid>/tensions/<tid>/<cid>/`, org avatars under `orgas/<rootnameid>/`,
user avatars under `users/<username>/`. Attachments live under their owning org so
per-org bucket policies can be expressed as a prefix filter. Keys carry a 16-hex-char
random prefix against guessing and collisions.

## Inline screenshot pasting

On paste the frontend embeds `![alt](<unique-filename>)` in the message and uploads the
bytes *after* the carrier comment exists. The upload handler finds the bare filename
token in `Comment.message` (code regions masked), rewrites it to `/file/<id>` and flips
`File.embedded=true`.

The rewrite is a read-modify-write: concurrent uploads to the same comment race on
`Comment.message` (last write wins). Only the inline URL substitution can be lost; the
UI renders orphaned `embedded` files as regular attachment chips.

## Email notifications

A mail client's anonymous fetch can't satisfy `/file/<id>` for Private/Secret orgs, so
notification emails ship the comment's files in the Postal payload: embedded images
become RFC 2392 inline attachments (`cid:` rewrite, allowlisted in the bluemonday
policy), everything else a plain attachment plus a footer link. Caps live under
`notify.*` in `config.toml`; anything dropped falls back to an absolute `/file/<id>`
URL. Attachments are fetched once per notification and shared across recipients.

### Upload settle poll

The notifier can fire while browser uploads are still in flight. The author declares
*how many* attachments are coming in `Comment.expected_attachments` (never which — an
integer carries no authority). `getLastCommentSettled` (`graph/notifications.go`)
re-polls until that many `File` rows are anchored and no bare `![](paste.png)` token
remains, bounded by `notify.upload_poll_*`. Declaring nothing settles on the first
read, no delay. Past the budget the email degrades and renders a hint line from
`EventNotif.MissingAttachments`.

### Inbound email replies

`POST /notifications` turns an email reply into a tension or contract Comment,
`POST /mailing` an email into a new Tension (`web/handlers/mailer.go`); both then run
`processInboundAttachments` on the exact comment uid returned by the insert.
`POST /mailing` reuses `graph.CreateTensionHook` with the attachment step as its
`attach` callback, so inbound tensions leave the same trace as app ones; contract
replies are gated by `graph.CanCommentContract`.

MUAs re-attach the quoted notification's inline images, so `dropKnownAttachments`
fingerprints `(safe filename, size)` against the tension's existing files first —
otherwise a re-sent image wins the pairing and lands in the wrong comment. Remaining
`cid:` refs resolve in three passes (quoted-back CIDs, filename, document order);
leftovers become plain paperclips. See `web/handlers/cid.go`.

Inbound processing is sequential and finishes before publication, so the notifier never
polls for it. Attachment failures are logged and skipped — the comment always commits.
Authorship trusts `From:`, gated by Postal's webhook signature. `POST /file/upload`
anchors on tension comments only; contract comment attachments are disabled in the app.

## Deployment and operations

Self-hosted Garage: Ansible role at `contrib/ansible/roles/garage/`. Dev fixtures
(keys, ports) live in `docker-compose.dev.yml` and `contrib/garage/garage.toml`;
inspect a running instance with the `garage` CLI (admin plane) or `mc` / `aws`.

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
  (`presign_ttl_sec`, 2h by default); only credential rotation kills outstanding URLs,
  and it kills all of them.
- **No byte-access audit**: the backend sees the presigned request, not the user. Swap
  the redirect in `FileGet` for a proxy stream if that changes.
- **Inbound dedup depth**: the fingerprint set covers the 20 newest comments only; an
  image quoted from further back mispairs.
- **Cascade-delete GC** is fire-and-forget (`deleteStorageKeysAsync`), driven by the
  `Delete*Deep` wrappers; with storage unset the `File` nodes still drop and the
  objects are left orphaned.
