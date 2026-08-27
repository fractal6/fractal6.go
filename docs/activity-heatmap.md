# Activity Tracking

Daily activity counters per user and per organisation, feeding GitHub-style
contribution heatmaps.

## Model

An `Activity` node holds a count for one owner on one day, keyed by
`activityid` (`u#<username>#YYYY-MM-DD` or `o#<rootnameid>#YYYY-MM-DD`) for upserts,
and by `ownerid` (`u#<username>` / `o#<rootnameid>`) for range queries. Declared in
`schema/graphql/fractal6.graphql`.

Direct mutations are root-only. Reads go exclusively through the `activity` `@meta`
computed field on `User` and `Node`, backed by the `getUserActivity` /
`getNodeActivity` DQL templates (optional `from`/`to` range, capped at 366 entries).

## Write path

Writes go through the `upsertActivity` DQL mutation — a conditional upsert
(increment, else create with `count=1`) in a single Dgraph transaction, so
concurrent events cannot race.

It hangs off the tension event pipeline: `ProcessEvent` fires `leaveTrace` as a
goroutine, which bumps node timestamps and calls `trackActivity`. The counter is only
bumped for events in `graph/activity.go::trackedEvents` — the noise filter and single
source of truth. Two upserts follow, one for the user and one for the org.

Fire-and-forget: it never blocks the GraphQL response, and a `recover()` in
`leaveTrace` keeps a panicking goroutine from taking down the server.

## Tests

`db/integration_mutation_test.go::TestUpsertActivity_Integration` covers the upsert
and query cycle; `graph/integration_card_event_test.go::TestPushProjectAdded_TracksActivity`
covers the full pipeline.
