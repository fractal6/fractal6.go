# Text search

## Tension search

All tension search paths match the same two fulltext indexes:
`anyoftext(Tension.title, q) OR anyoftext(Post.message, q)`.

- GraphQL `queryTension` (filters built by the frontend).
- REST `/q/tensions*` — `db/tensionQuery.go` (`Pattern`).
- Journal `Node.events_history(query:)` — `getNodeHistory` template in `db/dql_templates.go`.

`Tension.title` is stored and indexed directly. `Post.message` on a Tension is a
**denormalized search blob**: label names + the first comment body, synthesized by
`SyncTensionSearchMessage` (`graph/tension_search.go`) from the `getTensionSearchData`
DQL query.

### Index sync triggers

- Tension creation: seeded unconditionally (`addTensionHook`).
- `updateTension` events listed in `searchSyncEvents` (label add/remove, close/reopen):
  post-mutation via `HistoryNeedsSearchSync`.
- First-comment edit (`updateComment`, i.e. editing the tension body): `updateCommentHook`
  resolves the parent tension through the `~Tension.comments` reverse edge
  (`GetCommentTension`) and resyncs only when the edited comment is the first one.
- First-comment deletion is NOT synced; the index refreshes on the next label change.

All syncs run async (`goSyncSearch`), fired after the mutation is persisted.

### Reverse edge

`Comment.tension: Tension @dgraph(pred: "~Tension.comments")` in the schema makes Dgraph
maintain a storage-level `@reverse` index on `Tension.comments`. It is maintained on every
write path (GraphQL, DQL, live loader) and backfilled automatically when the schema is
pushed. Contract comments have no such edge and are skipped.

### Escaping

User patterns spliced into DQL string literals are escaped with `QuoteString`
(`db/tensionQuery.go`, `graph/meta_directive.go` for `@meta` field args). GraphQL paths
are escaped by the GraphQL layer itself.

## Project search

`Project.name` is fulltext-indexed and queried via GraphQL only (`anyoftext` on `name`).
The name is stored as-is, so there is no denormalization and no sync code.
`Project.description` is not indexed (not searched by the frontend either).
