# File Storage

S3-compatible object storage (Garage in production, MinIO compatible) for the
three asset kinds Fractale persists outside Dgraph: comment attachments,
user avatars, and org (root Node) avatars. Bytes are served via a single REST
proxy that re-authorises against the file's anchor on every read.

## Where things live

| Concern | File |
|---|---|
| Schema (`File`, anchor edges) | `schema/graphql/fractal6.graphql` |
| S3 client wrapper | `internal/storage/s3.go` |
| DQL templates (auth, add, replace, delete, list) | `db/dql_templates.go`, `db/dql_mutations.go` |
| DB helpers (`GetFileAuth`, `AddCommentFile`, `Replace*Avatar`, …) | `db/files.go` |
| HTTP handlers (`/file/*`) + markdown rewrite | `web/handlers/files.go` |
| Route wiring | `cmd/server.go` |
| Cascade-delete wrappers (`Delete{Comment,Tension,Contract,User}Deep`) | `db/dql_mutations.go` |
| Storage config | `[storage]` block in `config.toml` |

## Anchors

Every `File` row has exactly one anchor. The handler picks the auth path from
the populated triple in the form body:

| Form fields | Kind | GET auth | DB helper |
|---|---|---|---|
| `tid` + `cid` | `KindComment` | parent tension's receiver visibility | `AddCommentFile` |
| `userid` | `KindUser`  | public | `ReplaceUserAvatar` |
| `orgaid` | `KindNode` | node visibility | `ReplaceNodeAvatar` |

`File.tension` is denormalised alongside `File.comment` so the GET-time auth
fetch is a single DQL hop. There is no `Tension.files` edge.

User and org avatars are *replace-on-upload*: the helper drops the previous
`File` row + reverse edge, returns the previous `storageKey`, and the handler
fires a goroutine to GC the old S3 object.

## Auth model

Read access depends on the anchor kind (table above). Write access:

| Anchor | POST /file/upload | DELETE /file/<id> |
|---|---|---|
| comment (`tid`+`cid`) | EMAP[`CommentPushed`].Check on tension AND `cid` belongs to `tid` AND comment author == caller | uploader-only (`File.createdBy.username == uctx.Username`) |
| user (`userid`) | `userid == uctx.Username` | uploader-only (always self for avatars) |
| node (`orgaid`) | `auth.UserHasCoordoRole(uctx, orgaid)` | uploader-only |

```
GET /file/<id>
   ↓
db.GetFileAuth(id)              # one DQL hop, three branches projected; Kind picked Go-side
   ↓
isFileVisible(uctx, fa)         # public (KindUser) | IsNodeVisible(receiver) | IsNodeVisible(node)
   ↓
storage.PresignGet(key, ttl)    # short-lived presigned URL
   ↓
302 redirect (Cache-Control: private, no-store)
```

Bytes never travel through Fractale; the storage backend serves them directly
from the presigned URL. The TTL (default 10 min, see `presign_ttl_sec`) bounds
the leak window of any captured URL — see the design discussion in
`docs/file-storage.md` (this file) for why this is the recommended posture
over either pure proxying or returning presigned URLs to the client directly.

## REST surface

| Method | Path | Auth | Notes |
|---|---|---|---|
| `GET` | `/file/<id>` | per-anchor (table above) | 302 to presigned URL; **404** on miss *or* unauthorised (no existence leak) |
| `POST` | `/file/upload` | per-anchor (table above) | multipart; pass exactly one anchor: (`tid`+`cid`) \| `userid` \| `orgaid`. `file` part required. |
| `DELETE` | `/file/<id>` | uploader-only | 404 on miss/not-yours; DB row dropped first, then async S3 GC |

Handlers are constructed via `FileGetHandler(cli)` / `FileUploadHandler(cli)` /
`FileDeleteHandler(cli)` taking an injected `*storage.Client`. When the
`[storage]` section is unset, `cmd/server.go` passes `nil` and the handlers
return **503** instead of panicking. This also lets tests inject a fake.

The same client is also registered as a process-wide handle via
`storage.SetGlobal(cli)` in `cmd/server.go`, so non-handler callers (the
cascade-delete wrappers in `db/dql_mutations.go`) can read it via
`storage.Global()` without growing a parameter chain. `storage.Global()`
returns `nil` when storage is unset; the async GC then short-circuits and
the DQL templates still drop the `File` nodes from Dgraph.

`POST /file/upload` returns:

```json
{ "id": "0x...", "url": "/file/0x...", "filename": "...", "contentType": "...", "size": ..., "embedded": false }
```

`embedded` is set on the comment-attachment path when the upload's filename
appeared inline as `![alt](filename)` in the parent comment. The handler then
rewrites the comment to reference `/file/<id>` and flips `File.embedded=true`
in a single guarded upsert. See "Inline screenshot pasting" below.

Use `url` directly — for both `<a href="...">` attachments and markdown
`![alt](/file/0x...)`. The URL is stable for the lifetime of the file; presign
expiry is invisible to clients (re-issued on every request).

## Upload validation

The upload handler does not trust client-supplied metadata:

- **Server-side MIME sniff.** The client-supplied `Content-Type` header is
  discarded. The handler reads the first 512 bytes and feeds them to
  `http.DetectContentType`; the result is what gets persisted to
  `File.contentType` and what `/file/<id>` later serves. Uploading an HTML
  document labelled `image/png` will land as `text/html` — frontends should
  not be surprised.
- **Inline vs attachment is an allowlist, not the sniffed type.** Only an
  explicit set of types render in-page (`image/png|jpeg|gif|webp|avif|bmp|x-icon`,
  `video/mp4|webm|ogg`, `audio/mpeg|ogg|wav`, `application/pdf`, `text/plain`).
  Everything else — notably **`image/svg+xml` and `text/html`** — is forced
  to `Content-Disposition: attachment` to neutralise the SVG-with-`<script>`
  and HTML-masquerade-as-image XSS vectors. The allowlist lives in
  `inlineSafeContentTypes` in `web/handlers/files.go`.
- **`X-Content-Type-Options: nosniff`** is set on every GET response, so
  browsers don't second-guess the Content-Type the storage backend echoes
  back. Defence in depth on top of the allowlist.
- **Size cap.** `storage.max_upload_bytes` (default **10 MiB**) is enforced
  via `http.MaxBytesReader` *and* `ParseMultipartForm`. Going over → 400
  with `"upload too large or malformed: ..."`. To raise (e.g. 100 MiB), set
  `max_upload_bytes = 104857600` in `config.toml` — and bump the matching
  body limit on any reverse proxy in front (`client_max_body_size` for nginx).
- **Uid validation.** Every client-supplied id — the `tid`/`cid` form fields
  and the `<id>` in `/file/<id>` — goes through `db.ValidateUids` before it
  reaches a DQL `uid(...)` root: malformed id → 400 on upload, 404 on
  GET/DELETE. DQL `uid()` accepts comma-separated lists, so an unvalidated
  id lets a caller widen a query (or a delete) to nodes it never owned.
  The inbound-email path applies the same rule to the tension/contract uid
  parsed from the `References` header (`parseEmailReferences` in
  `web/handlers/mailer.go`), which is sender-controlled.

## Inline screenshot pasting

Frontend convention: when the user pastes a screenshot during compose, the
client embeds `![alt](filename)` *with a unique filename per paste* (e.g.
`paste-<timestamp>-<counter>.png`) and holds the bytes in browser memory.
On publish the client does:

1. The GraphQL mutation that creates the carrier (comment / tension+initial
   comment / `updateComment`).
2. `POST /file/upload?tid=<tid>&cid=<cid>` once per pasted file, *after* the
   carrier exists server-side.

The upload handler reads the comment message, finds the bare-token
`![alt](<filename>)` reference, replaces it with `![alt](/file/<id>)`, and
flips `File.embedded=true`. The rewrite is a plain read-modify-write —
**concurrent uploads to the same comment race on `Comment.message` (last
write wins).** The File rows themselves are independent and both persist;
only the inline URL substitution can be lost. The UI is lenient about
`embedded=true` files whose URL is no longer in the message — they render
as regular attachment chips.

Code regions are masked before matching: filenames mentioned inside fenced
` ``` ` / `~~~` blocks or inline backticks are not rewritten.

**Filename collisions are the frontend's responsibility.** Two pasted files
with the same filename inside one comment cannot both be embedded — the
first match wins, the second is left as a placeholder. To avoid concurrent-
upload races, frontends should serialise (or low-concurrency batch) the
per-file `POST /file/upload` calls for the same parent comment.

### Order of operations from the frontend

| Scenario | Sequence |
|---|---|
| Edit existing comment with new pastes | `updateComment(cid, message=…filenames…)` → for each file `POST /file/upload?tid=&cid=` |
| New comment on existing tension | `addComment(tension=tid, message=…filenames…)` → use returned `cid` → for each file `POST /file/upload?tid=&cid=` |
| New tension with screenshots in body | `addTension(initial body …filenames…)` → use returned `tid` and initial `cid` → for each file `POST /file/upload?tid=&cid=` |
| User avatar | `POST /file/upload?userid=` |
| Org avatar | `POST /file/upload?orgaid=` |

## Storage layout

A single bucket holds all file kinds, namespaced by key prefix:

| Prefix | Purpose |
|---|---|
| `orgas/<rootnameid>/tensions/<tid>/<cid>/<rand>-<safe-name>` | comment attachments |
| `orgas/<rootnameid>/<rand>-<safe-name>` | org (root Node) avatars |
| `users/<username>/<rand>-<safe-name>` | user avatars |

Comment attachments live under their owning org so per-org bucket policies
(lifecycle, KMS, export, isolation) can be expressed as a key-prefix filter.
Avatar keys for an org never collide with its tension subtree because avatars'
second segment is always `<rand16hex>-<name>`, never the literal `tensions`.

`<rand>` is 16 hex chars (~64 bits) to prevent key guessing and collisions.
`<safe-name>` is the original filename with control chars + path separators
stripped, capped at 120 chars. The original filename is also stored in the
`File.filename` predicate and surfaced as `Content-Disposition` on download.

## Schema (GraphQL)

```graphql
type File @auth(
  add: <<is-root>>,
  update: <<is-root>>,
  delete: <<is-root>>
) {
  id: ID!
  createdBy: User!
  createdAt: DateTime! @search
  filename: String!
  contentType: String!
  size: Int!
  storageKey: String! @id

  # Anchor — exactly one of the three groups below is populated per row.
  comment: Comment           # comment attachment
  tension: Tension           # denormalised parent tension uid (co-set with comment)
  user:    User              # user avatar
  node:    Node              # node (root org) avatar

  # True when referenced inline in comment.message.
  embedded: Boolean
}

type Comment { ...
  files: [File!] @hasInverse(field: comment) @x_ro
}
type User { ...
  avatar: File @x_ro @hasInverse(field: user)
}
type Node { ...
  avatar: File @x_ro @hasInverse(field: node)
}
```

Mutations are root-only — clients **never** create/edit/delete `File` via GraphQL. All writes go through the REST endpoints, which run the auth checks and persist via internal DQL.

`queryFile` / `getFile` are intentionally left open for query: the `storageKey` field is useless without going through `/file/<id>` (which performs the auth check before issuing a presigned URL).

## Configuration

`config.toml` `[storage]` block:

```toml
[storage]
endpoint           = "garage.fractale.co"   # host[:port], no scheme
region             = "garage"               # Garage default
bucket             = "fractale-storage"
access_key         = "..."
secret_key         = "..."
use_ssl            = true                   # Garage on a separate host → require TLS
public_url_prefix  = ""                     # public-facing proxy/CDN, e.g. "https://files.fractale.co"
max_upload_bytes   = 10485760               # 10 MiB
presign_ttl_sec    = 600                    # 10 min
```

Leave `endpoint` empty in dev environments to disable upload features cleanly (handlers return 503 instead of crashing).

`endpoint` is the data-plane address the backend uses (may be private, e.g. `127.0.0.1:3900`). When it is not reachable by browsers, set `public_url_prefix` to the public proxy in front of Garage (`scheme://host[:port]`, no path): presigned URLs are then *signed with that host*, since SigV4 covers the `Host` header. The proxy must forward the Host header unchanged, otherwise Garage answers `SignatureDoesNotMatch`.

## Deployment

For self-hosted Garage on a separate VM, see the standalone Ansible role at `contrib/ansible/roles/garage/`. 
It installs the binary, renders the TOML config, sets up systemd, and (optionally) bootstraps the bucket and access keys.

## Inspecting Garage

Three complementary ways to poke a running instance. The dev fixtures below are the access key / secret / admin token baked into `docker-compose.dev.yml` and `contrib/garage/garage.toml` — do not reuse outside dev.

### 1. `garage` CLI (admin/control plane)

Same binary as the server, run inside the container. Talks to the admin API on `3903` with the `admin_token`. Covers cluster status, buckets, keys, layout
— **not** object listing or fetching.

```sh
docker compose -f docker-compose.dev.yml exec garage-dev /garage status
docker compose -f docker-compose.dev.yml exec garage-dev /garage bucket list
docker compose -f docker-compose.dev.yml exec garage-dev /garage bucket info fractale-storage
docker compose -f docker-compose.dev.yml exec garage-dev /garage key list
docker compose -f docker-compose.dev.yml exec garage-dev /garage key info GKdeadbeefdeadbeefdeadbeefdeadbeef --show-secret
```

### 2. MinIO Client `mc` (S3 data plane)

Best tool for browsing/listing/fetching actual objects — i.e. for verifying that an upload landed at the expected key prefix.

```sh
mc alias set garage-dev http://127.0.0.1:3900 \
  GKdeadbeefdeadbeefdeadbeefdeadbeef \
  deadbeefdeadbeefdeadbeefdeadbeefdeadbeefdeadbeefdeadbeefdeadbeef \
  --api S3v4

mc ls   garage-dev/fractale-storage
mc tree garage-dev/fractale-storage
mc ls   --recursive garage-dev/fractale-storage/orgas/
mc cp   ./local.png garage-dev/fractale-storage/scratch/test.png
mc rm   garage-dev/fractale-storage/scratch/test.png
```

Equivalent with the `aws` CLI:

```sh
AWS_ACCESS_KEY_ID=GKdeadbeefdeadbeefdeadbeefdeadbeef \
AWS_SECRET_ACCESS_KEY=deadbeefdeadbeefdeadbeefdeadbeefdeadbeefdeadbeefdeadbeefdeadbeef \
aws --endpoint-url http://127.0.0.1:3900 s3 ls s3://fractale-storage/ --recursive
```

### 3. Web UI

Garage has no first-party web UI. The community option is [`khairul169/garage-webui`](https://github.com/khairul169/garage-webui),
which speaks Garage's admin API — admin-plane only (buckets, keys, layout), no object browsing. For object browsing through a GUI, point Cyberduck or similar at `http://127.0.0.1:3900` with an S3 profile and the dev credentials.

## DQL note: no reverse edge on `Tension.comments`

`Tension.comments` has no `@hasInverse` in the GraphQL schema, so Dgraph
doesn't generate a reverse predicate. `getCommentMessage` therefore walks
forward from `tid` and filters its `Tension.comments` edge by `cid` rather
than via a reverse-edge query. `getFileAuth_v2` sidesteps the issue entirely
by reading `File.tension` (denormalised at insert time).

Note: `db.Meta()` runs every response through `tools.CleanDqlMap`, which strips
`Type.` prefixes (`File.storageKey` → `storageKey`, `Post.createdBy` →
`createdBy`, `uid` → `id`). The decoders in `db/files.go` look up the cleaned
keys; a previous version used the raw keys and silently returned empty values.

## Email notifications: inline CID + plain attachments

Tension and contract notification emails ship the comment's attached files in
the Postal `attachments` array, so recipients see them locally instead of via
the auth-gated `/file/<id>` proxy (which a mail client's anonymous fetch
can't satisfy for Private/Secret orgs).

| Bucket | Criteria | Wire-up |
|---|---|---|
| **A — Inline (CID)** | `File.embedded == true` AND ContentType ∈ {`image/png`, `image/jpeg`, `image/gif`, `image/webp`} AND size under per-file cap | `attachments[]` entry with `content_id = <fid>@<DOMAIN>`; the HTML body's `<img src="/file/<id>">` is rewritten to `<img src="cid:<fid>@<DOMAIN>">` |
| **B — Plain attachment** | everything else on the comment (non-embedded, oversized inline, non-image, SVG) | `attachments[]` entry without `content_id` (paperclip in the mail UI); also appears in a `Attachments:` footer block as `<a href="https://<DOMAIN>/file/<id>">name (size)</a>` |

Both buckets share the per-email caps:

| `config.toml` | Default | Behaviour at cap |
|---|---|---|
| `notify.attachment_per_file_bytes` | 10 MiB | file dropped from Postal payload; inline → absolute-URL `<img>`; plain → footer link only |
| `notify.attachment_total_bytes`    | 100 MiB | next file dropped (Bucket B first, then A); spillover stays as footer links |
| `notify.attachment_max_count`      | 20      | hard cap on number of files attached to one email |

Files that didn't make it into the inline-CID set are absolutised
(`<img src="https://<DOMAIN>/file/<id>">`) — Public-org images may still load
in the recipient's client; Private/Secret-org images will 404. The HTML is
re-sanitised with bluemonday after the rewrites; `cid:` is in the URL-scheme
allowlist (`web/email/main.go`).

Where it lives:
- `web/email/attachments.go` — partition, fetch, rewrite, footer
- `web/email/main.go` — `SendEventNotificationEmail` / `SendContractNotificationEmail` integration
- `db/dql_templates.go::getLastCommentFiles` / `getLastContractCommentFiles` — file projection
- `db/files.go::GetLastCommentFiles` / `GetLastContractCommentFiles` — Go decoders
- `internal/storage/s3.go::GetObject` — byte fetch (base64-encoded into the Postal payload)

### Upload settle poll

The notifier daemon is a separate cobra subcommand (`cmd/notifier.go`) from the
api server. The frontend `POST /file/upload`s **after** the GraphQL mutation
that creates the carrier comment, so a notification can fire while uploads
are still in flight — at which point `Comment.files` is empty and inline
pastes still hold their bare paste filenames.

The message itself is the ground truth: a successful inline upload rewrites
`![](paste.png)` to `![](/file/<id>)` (`embedIfReferenced`). So before
reading the comment for events containing `Created` / `CommentPushed`,
`PushEventNotifications` polls (`getLastCommentSettled` in
`graph/notifications.go`):

1. Sleep `notify.upload_poll_interval_sec` (default 5s). This first sleep is
   also the window for plain (non-inline) attachments, which leave no token
   and therefore can't be detected by the poll.
2. Fetch the author's last comment and count remaining bare tokens via
   `tools.CountInlineImageCandidates`.
3. None left → proceed. Some remain → sleep and re-fetch, up to
   `notify.upload_poll_attempts` (default 10, ~55s worst case), then send
   as-is.

No cross-process coordination: the api process plays no role in the wait.
Degradation past the budget has the same shape as a slow upload without any
poll — broken inline `<img>` / missing attachment in the email — while the
web comment renders correctly once the upload lands.

### Inbound email replies

`POST /notifications` (`web/handlers/mailer.go::Notifications`) is the
Postal-driven webhook that turns an email reply into a new `Comment` on the
referenced tension. Postal ships any attached files in the same payload as

```json
{ "filename": "...", "content_type": "...", "size": ..., "data": "<base64>" }
```

with **no Content-ID and no Content-Disposition**. `processInboundAttachments`
persists those bytes under the comment's S3 prefix
(`orgas/<root>/tensions/<tid>/<cid>/`) and rewrites the message so it lines
up with the outbound flow.

Three matching paths, in order:

| Case | Match | Action |
|---|---|---|
| **Quoted-back** | `cid:<fid>@<server.domain>` (the original outbound CID surviving the reply quote) | rewrite to `/file/<fid>`; do not consume an attachment |
| **Filename heuristic** | inbound `Filename` equals the cid token, its local-part, or `<local-part>.<ext>` | write the file, rewrite the ref to `/file/<newFid>`, flip `embedded=true` |
| **Document-order fallback** | unmatched cid refs paired with remaining attachments in order | as above |

Unmatched cid refs are dropped from the message. Attachments that don't
resolve to a cid ref are persisted anyway and surface as plain paperclips
(non-`embedded`). All decisions roll up into a single `EmbedCommentMessage`
upsert that rewrites `Comment.message` and flips `embedded=true` on the
inline-matched fids in one shot.

**No upload gate** here — writes are sequential inside the handler, so
`PublishTensionEvent` runs only after every file is committed and the
notifier daemon's `getLastCommentFiles` projection sees them on the first
read.

**Best-effort, never aborts the comment.** Storage unset, bad base64,
oversize, S3 or DB failure — each attachment is logged-and-skipped; the
comment itself stays committed.

**Trust on `From:` for authorship.** Unchanged from the text-only flow:
Postal's webhook signature gates the request; `GetUctx("email", From)`
resolves to the account. A spoofed `From:` that resolves to a real user is
the same risk surface the comment write already carries.

Caps (`templates/config.toml`):

| Key | Default | Effect |
|---|---|---|
| `notify.inbound_attachment_per_file_bytes` | falls back to `storage.max_upload_bytes` (10 MiB) | oversize attachments dropped before the S3 PUT |
| `notify.inbound_attachment_max_count` | 20 | extras past this index silently truncated |

**Contract replies are not yet supported.** The `isCid != ""` branch in
`Notifications` still writes a text-only comment; attachments on a contract
reply are ignored. Follow-up PR.

## Operational notes

- **Cascade-delete S3 GC** is shared by `DeleteCommentDeep`, `DeleteTensionDeep`,
  and `DeleteUser` (all in `db/dql_mutations.go`). Each template projects an
  `all` block of `File.storageKey` rows; the wrapper feeds them through
  `collectStorageKeys` + `deleteStorageKeysAsync` (a 2-min, fire-and-forget
  goroutine using `storage.Global()`). When `storage.Global()` is `nil`
  (no `[storage]` block), the GC is a no-op and the DQL still drops the
  `File` nodes — orphan objects can be swept out-of-band by listing keys
  under `orgas/<rootnameid>/tensions/<tid>/<cid>/` (or the relevant prefix).
- **Crash-orphan leak (known, unhandled)**: if the process dies between S3 `Put` and the DB row insert (upload) the bytes orphan with no GC. Deemed marginal — bounded leak, no broken links/auth impact. No backstop; sweep via key listing if it ever matters.
- **No revocation**: a presigned URL captured by a user remains valid until TTL expires. Rotating credentials invalidates *all* outstanding URLs (last resort).
- **No audit of byte access**: the storage backend sees only the presigned request, not the originating user identity. If audit is needed, switch to proxy-streaming in `FileGet` (replace the `http.Redirect` with `cli.GetObject` `io.Copy`).
- **Filename safety**: see `safeFilename` in `web/handlers/files.go` for the exact sanitisation rules.
