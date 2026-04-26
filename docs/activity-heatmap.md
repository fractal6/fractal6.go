# Activity Tracking

Daily activity counters per user and per organisation, designed for GitHub-style contribution heatmaps.

## Schema

```graphql
type Activity @auth(
  query: <<is-root>>,
  add: <<is-root>>,
  update: <<is-root>>,
  delete: <<is-root>>
) {
  id: ID!
  activityid: String! @id          # "u#<username>#YYYY-MM-DD" or "o#<rootnameid>#YYYY-MM-DD"
  ownerid: String! @search(by: [hash])  # "u#<username>" or "o#<rootnameid>"
  date: DateTime! @search(by: [day])
  count: Int!
}
```

### Fields

| Field | Description |
|-------|-------------|
| `activityid` | Composite unique key for DQL upsert. Encodes owner + day (e.g. `u#alice#2026-02-09`). |
| `ownerid` | Owner identifier for range queries. Encodes owner only (e.g. `u#alice` or `o#myorg`). |
| `date` | Day of the activity (ISO 8601). Indexed by day for `between` filters. |
| `count` | Number of tracked events on this day. |

### Tracked events (noise filter)

Not every `TensionEvent` bumps the counter. The set of events that *do* count
lives in `graph/activity.go::trackedEvents` — that's the single source of truth.
Events absent from the set are treated as noise and dropped at `trackActivity`.

### Access

Activity nodes are only accessible via `@meta` computed fields on `User` and `Node`:

```graphql
# On User type
activity(from: String, to: String): [Activity!] @meta(f:"getUserActivity", k:["username"])

# On Node type (root nodes)
activity(from: String, to: String): [Activity!] @meta(f:"getNodeActivity", k:["rootnameid"])
```

Direct GraphQL mutations are restricted to root (`<<is-root>>`). All writes go through DQL upserts which bypass GraphQL auth.

## DQL Templates

### `upsertActivity` (mutation)

Conditional upsert: increments `count` if the activity node exists, creates it with `count=1` otherwise. Single Dgraph transaction, no race conditions.

### `getUserActivity` / `getNodeActivity` (queries)

Query activity nodes by `ownerid` with optional `from`/`to` date range filter. Returns up to 366 entries ordered by date descending.

## Async Pattern

Activity tracking hooks into the `ProcessEvent` pipeline in `graph/tension_op.go`:

1. `ProcessEvent` calls `leaveTrace(uctx, tension, eventType)` as a **goroutine**
2. `leaveTrace` updates node timestamps then calls `trackActivity`
3. `trackActivity` short-circuits if the event is not in `trackedEvents`
4. Otherwise it issues two `upsertActivity` DQL mutations: one for the user, one for the org
5. A `recover()` in `leaveTrace` prevents goroutine panics from crashing the server

Activity updates are fire-and-forget and do not block the GraphQL response.

## Query Examples

### User activity (last year)

```graphql
query {
  getUser(username: "alice") {
    activity(from: "2025-02-09T00:00:00Z", to: "2026-02-09T00:00:00Z") {
      date
      count
    }
  }
}
```

### Organisation activity (all time)

```graphql
query {
  getNode(nameid: "myorg#") {
    activity {
      date
      count
    }
  }
}
```

## Integration Tests

See `db/integration_mutation_test.go` — `TestUpsertActivity_Integration` covers:
- Creating an activity node via upsert
- Querying it back via `getUserActivity`
- Incrementing via second upsert
- Cleanup

`graph/integration_card_event_test.go::TestPushProjectAdded_TracksActivity`
covers the full pipeline: emitting a project event bumps both the user and the
org daily counter.
