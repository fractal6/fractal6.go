# Integration Tests for db/ Package

Integration tests verify that DQL queries and mutations work correctly against a real Dgraph instance running in Docker.

## Prerequisites

- Docker and Docker Compose (v2)
- Go 1.22+

## Quick Start

Run the full cycle (start containers, run tests, tear down):

```bash
make test-integration-clean
```

## Targets

| Target | Description |
|--------|-------------|
| `make test-integration-up` | Start Dgraph containers (zero + alpha) |
| `make test-integration` | Start containers + run integration tests |
| `make test-integration-down` | Stop and remove containers |
| `make test-integration-clean` | Full cycle: start, test, tear down |

## Iterative Development

Keep containers running between test runs to speed up iteration:

```bash
make test-integration-up
# edit code...
go test -tags integration -v -count=1 -timeout 120s ./db/...
# edit more...
go test -tags integration -v -count=1 -timeout 120s ./db/...
# done
make test-integration-down
```

## Checking Compilation Without Containers

To verify integration test files compile without needing running Dgraph containers:

```bash
go vet -tags integration ./db/...
```

This catches syntax errors, type mismatches, and import issues without executing any tests.

## Architecture

### Docker Setup

`docker-compose.test.yml` runs Dgraph zero + alpha using `dgraph/dgraph:v22.0.2` (matching the project's `DGRAPH_RELEASE`).

Ports are offset by +100 to avoid collision with dev instance:

| Service | Internal Port | Host Port |
|---------|--------------|-----------|
| zero gRPC | 5180 | 5180 |
| zero HTTP | 6180 | 6180 |
| alpha HTTP | 8180 | 8180 |
| alpha gRPC | 9180 | 9180 |

No volumes are mounted - data is ephemeral and destroyed on `docker compose down`.

### Test Data

`TestMain` in `db/integration_test.go` handles setup:

1. Overrides the global `DB` singleton with test-instance addresses
2. Waits for Dgraph alpha to be healthy (polls `/health`)
3. Loads `schema/dgraph_schema.graphql` via the admin endpoint
4. Seeds test data via gRPC N-Quads mutation

Seeded entities:
- 1 root Node (Circle): `test-org#`
- 1 User: `testuser`
- 1 Member role node: `test-org#@testuser`
- 1 Coordinator role node: `test-org#:coordo`
- 1 Tension (Open, Operational)
- 1 Event (Created)

### Build Tag

All integration test files use `//go:build integration`. This means:
- `go test ./...` (and `make test`) skips them entirely
- Only `go test -tags integration ./db/...` runs them

### DQL Bypasses @auth

DQL queries go through gRPC and skip the Dgraph GraphQL auth layer. No JWT keys are needed for integration tests.

### Test Categories

- `integration_query_test.go` - Read-only tests (CountHas, Exists, GetFieldByEq, IsChild, GetChildren, HasCoordos, QueryDql, Meta)
- `integration_mutation_test.go` - Write tests (SetFieldByEq, UpgradeMember, Gamma, markAllAsRead). Mutation tests restore original values after modifying data.
