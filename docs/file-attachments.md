# File Attachments

S3-compatible object storage (Garage in production, MinIO compatible) for files
attached to comments. Bytes are served via a single REST proxy that re-uses
the existing GBAC auth for the parent comment.

## Where things live

| Concern | File |
|---|---|
| Schema (`File` type, `Comment.files`) | `schema/graphql/fractal6.graphql` |
| S3 client wrapper | `internal/storage/s3.go` |
| DQL templates (auth, add, delete, list) | `db/dql_templates.go`, `db/dql_mutations.go` |
| DB helpers (`GetFileAuth`, `AddFileToComment`, `CleanupCommentFiles`, …) | `db/files.go` |
| HTTP handlers (`/file/*`) | `web/handlers/files.go` |
| Route wiring | `cmd/server.go` |
| Comment-delete GC | `graph/tension_op.go::RemoveComment` |
| Storage config | `[storage]` block in `config.toml` |

## Auth model

There is **one** auth path for both first-class attachments and markdown-embedded
images. Read access matches the parent comment's tension visibility (re-using
`auth.IsNodeVisible`); write/delete is restricted to the comment author (same
rule as `RemoveComment`).

```
GET /file/<id>
   ↓
db.GetFileAuth(id)              # File → Comment → Tension → receiver(nameid, visibility)
   ↓
auth.IsNodeVisible(uctx, ...)   # 403 on fail
   ↓
storage.PresignGet(key, ttl)    # short-lived presigned URL
   ↓
302 redirect (Cache-Control: private, no-store)
```

Bytes never travel through Fractale; the storage backend serves them directly
from the presigned URL. The TTL (default 10 min, see `presign_ttl_sec`) bounds
the leak window of any captured URL — see the design discussion in
`docs/file-attachments.md` (this file) for why this is the recommended posture
over either pure proxying or returning presigned URLs to the client directly.

## REST surface

| Method | Path | Auth | Notes |
|---|---|---|---|
| `GET` | `/file/<id>` | Comment read auth | 302 to presigned URL; **404** on miss *or* unauthorised (no existence leak) |
| `POST` | `/file/upload` | Comment author | multipart, fields: `comment_id`, `file` |
| `DELETE` | `/file/<id>` | Comment author | 404 on miss/not-yours; S3 first, then DB on success |

Handlers are constructed via `FileGetHandler(cli)` / `FileUploadHandler(cli)` /
`FileDeleteHandler(cli)` taking an injected `*storage.Client`. When the
`[storage]` section is unset, `cmd/server.go` passes `nil` and the handlers
return **503** instead of panicking. This also lets tests inject a fake.

The same client is also registered as a process-wide handle via
`storage.SetGlobal(cli)` in `cmd/server.go`, so non-handler callers (the
comment-delete GC in `graph.RemoveComment`) can read it via `storage.Global()`
without growing a parameter chain. `storage.Global()` returns `nil` when storage
is unset; callers MUST nil-check (this is the fast path through
`db.CleanupCommentFiles(cid, nil)`).

`POST /file/upload` returns:

```json
{ "id": "0x...", "url": "/file/0x...", "filename": "...", "contentType": "...", "size": ... }
```

Use `url` directly — for both `<a href="...">` attachments and markdown
`![alt](/file/0x...)`. The URL is stable for the lifetime of the file; presign
expiry is invisible to clients (re-issued on every request).

## Storage layout

A single bucket holds all file kinds, namespaced by key prefix:

| Prefix | Purpose |
|---|---|
| `comments/<cid>/<rand>-<safe-name>` | comment attachments (wired now) |
| `avatars/<username>/<rand>-<safe-name>` | user avatars (future) |
| `orgas/<rootnameid>/<rand>-<safe-name>` | organisation assets (future) |

`<rand>` is 16 hex chars (~64 bits) to prevent key guessing and collisions.
`<safe-name>` is the original filename with control chars + path separators
stripped, capped at 120 chars. The original filename is also stored in the
`File.filename` predicate and surfaced as `Content-Disposition` on download.

## Schema (GraphQL)

```graphql
type Comment implements Post @auth(update: <<update-comment>>) @hook_ {
  message: String! @search(by: [fulltext]) @x_alter
  reactions: [Reaction!] @hasInverse(field: comment)
  files: [File!] @hasInverse(field: comment) @x_ro
}

type File @auth(
  add: <<is-root>>,
  update: <<is-root>>,
  delete: <<is-root>>
) {
  id: ID!
  createdBy: User!
  createdAt: DateTime! @search
  comment: Comment!
  filename: String!
  contentType: String!
  size: Int!
  storageKey: String! @id
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
bucket             = "fractale-attachments"
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
doesn't generate a reverse predicate. The auth templates (`getCommentAuth`,
`getFileAuth`) therefore start their walk at the parent `Tension` and filter
with `uid_in(Tension.comments, <comment uid>)` rather than `~Tension.comments`.
The same pattern is reused in `getCommentFiles`. Don't switch back to a
reverse-edge walk without first declaring the predicate as `@reverse` in the
underlying DQL schema (which the GraphQL admin won't do for you).

## Operational notes

- **Comment delete** triggers `db.GetDB().CleanupCommentFiles(cid)` before the
  DQL `deleteComment` runs. Per-file failures are logged but do not block the
  delete; orphan objects can be GC'd by listing keys under `comments/<cid>/`.
- **No revocation**: a presigned URL captured by a user remains valid until
  TTL expires. Rotating credentials invalidates *all* outstanding URLs (last
  resort).
- **No audit of byte access**: the storage backend sees only the presigned
  request, not the originating user identity. If audit is needed, switch to
  proxy-streaming in `FileGet` (replace the `http.Redirect` with `cli.GetObject`
  + `io.Copy`).
- **Filename safety**: see `safeFilename` in `web/handlers/files.go` for the
  exact sanitisation rules.
