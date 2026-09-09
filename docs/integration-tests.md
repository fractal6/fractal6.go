# Integration Tests

Integration tests exercise DQL queries, mutations and HTTP handlers against real
Dgraph, Redis and MinIO instances in Docker (`docker-compose.test.yml`). Ports are
offset by +100 from the dev instances; no volumes are mounted, so data is ephemeral.

Needs Docker Compose v2 and an RSA key pair in the project root (`make certs`).

## Running

`make test-integration-clean` is the canonical entry point: start containers, seed,
run, tear down. See the `test-*` targets in the `Makefile` for the individual steps —
`test-integration-setup` keeps containers up for iterative runs, and
`go vet -tags integration ./...` type-checks the test files without any container.

Calling `go test -tags integration` by hand needs the env the Makefile normally
injects: `REDIS_ADDR=localhost:6479` (the test Redis, not the dev one) plus
`DGRAPH_PUBLIC_KEY` / `DGRAPH_PRIVATE_KEY` from the PEM files. Re-seed with
`go run ./cmd/testsetup`.

Files are tagged `//go:build integration`, so plain `go test ./...` skips them. Tests
hitting external services (Matrix) use `//go:build external` and are excluded from
both.

The GQL request builder (`db/gql_test.go`) and the Dgraph bridges
(`graph/dgraph_resolver_test.go`) are covered container-free: they impersonate the
Dgraph endpoint with `internal/testutil` (`FakeGqlServer` / `CheckRequest`), so that
layer needs no Docker — only a local Redis (the `graph` package `init()` exits without
one).

## Test data

`cmd/testsetup` seeds once before any package runs; each `TestMain` only verifies the
data exists.

| File | Role |
|---|---|
| `cmd/testsetup/main.go` | Wait for Dgraph, drop, load schema, poll predicates, bootstrap the S3 bucket |
| `cmd/testsetup/seed.go` | Substitute password placeholders, run the gRPC mutation |
| `cmd/testsetup/seed.nq` | Static N-Quads dataset (`//go:embed`) — edit this to change seed data |

Setup is resilient to Dgraph startup races (schema upload retries, predicate poll
before seeding). Shared connection addresses, usernames and passwords live in
`internal/testutil/config.go`.

Two orgs are seeded: `test-org` (Public, general purpose) and `sec-org` (Private,
with Private and Secret sub-circles for visibility tests). See `seed.nq` for the full
picture.

DQL goes through gRPC and bypasses the Dgraph GraphQL auth layer, so seeding needs no
JWT keys.

## Test packages

- `db/` — DQL read tests (`integration_query_test.go`) and write tests
  (`integration_mutation_test.go`).
- `web/handlers/` — end-to-end through a real chi router with JWT middleware. Shared
  helpers (`doRequest`, `loginAs`, …) live in `integration_test.go`; auth, org, import
  and `/file/*` suites sit alongside.

Email is mocked with a local `httptest.Server` (`email.SetTestConfig`); Redis and S3
are the real Docker containers.

## Pitfalls

**Nameid format.** Root orgs are bare (`"test-org"`, no trailing `#`). `Nid2pid`
splits on `#` and returns `parts[0]` when `parts[1]` is empty, so a trailing separator
silently breaks every `UserHasCoordoRole` check.

**Mutation tests must restore the exact original value.** `UpgradeMember` writes both
`Node.role_type` and `Node.name`; restoring to a plausible-but-different role breaks
later tests that expect Owner/Coordinator.

**Packages share Dgraph and Redis.** `db/` and `web/handlers/` run in parallel against
the same instances, and the Redis role cache has a 12s TTL. Keep mutation windows
short, restore immediately, and never seed roles into the shared orgs — create a
throwaway org and delete it in `t.Cleanup`.

**JWT keys.** `db.init()` loads signing keys from the viper config paths and falls
back to `DGRAPH_PUBLIC_KEY` / `DGRAPH_PRIVATE_KEY` when they are missing (usual case
in the test CWD).

**External-service tests** (Matrix) are tagged `external` and run only via
`make test-external`.

**Signup needs the email mock.** `Signup` panics if `SendVerificationEmail` fails.
