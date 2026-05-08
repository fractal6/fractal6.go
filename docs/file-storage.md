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
| `DELETE` | `/file/<id>` | uploader-only | 404 on miss/not-yours; S3 first, then DB on success |

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

Mutations are root-only — clients **never** create/edit/delete `File` via
GraphQL. All writes go through the REST endpoints, which run the auth checks
and persist via internal DQL.

`queryFile` / `getFile` are intentionally left open for query: the `storageKey`
field is useless without going through `/file/<id>` (which performs the auth
check before issuing a presigned URL).

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
public_url_prefix  = ""                     # optional CDN/proxy host rewrite
max_upload_bytes   = 10485760               # 10 MiB
presign_ttl_sec    = 600                    # 10 min
```

Leave `endpoint` empty in dev environments to disable upload features cleanly
(handlers return 503 instead of crashing).

## Deployment

For self-hosted Garage on a separate VM, see the standalone Ansible role at
`contrib/ansible/roles/garage/`. It installs the binary, renders the TOML config,
sets up systemd, and (optionally) bootstraps the bucket and access keys.

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

## Operational notes

- **Cascade-delete S3 GC** is shared by `DeleteCommentDeep`, `DeleteTensionDeep`,
  and `DeleteUser` (all in `db/dql_mutations.go`). Each template projects an
  `all` block of `File.storageKey` rows; the wrapper feeds them through
  `collectStorageKeys` + `deleteStorageKeysAsync` (a 2-min, fire-and-forget
  goroutine using `storage.Global()`). When `storage.Global()` is `nil`
  (no `[storage]` block), the GC is a no-op and the DQL still drops the
  `File` nodes — orphan objects can be swept out-of-band by listing keys
  under `orgas/<rootnameid>/tensions/<tid>/<cid>/` (or the relevant prefix).
- **No revocation**: a presigned URL captured by a user remains valid until
  TTL expires. Rotating credentials invalidates *all* outstanding URLs (last
  resort).
- **No audit of byte access**: the storage backend sees only the presigned
  request, not the originating user identity. If audit is needed, switch to
  proxy-streaming in `FileGet` (replace the `http.Redirect` with `cli.GetObject`
  + `io.Copy`).
- **Filename safety**: see `safeFilename` in `web/handlers/files.go` for the
  exact sanitisation rules.
