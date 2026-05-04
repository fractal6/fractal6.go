# Integration Tests

Integration tests verify that DQL queries, mutations, and HTTP handlers work correctly against real Dgraph and Redis instances running in Docker.

## Prerequisites

- Docker and Docker Compose (v2)
- Go 1.22+
- RSA key pair (`public.pem` / `private.pem`) in the project root (see `make certs`)

## Quick Start

Run the full cycle (start containers, seed data, run tests, tear down):

```bash
make test-integration-clean
```

## Targets

| Target | Description |
|--------|-------------|
| `make test` | Run vet + unit tests |
| `make test-unit` | Unit tests only |
| `make test-external` | Tests calling external services (Matrix, etc.) |
| `make test-integration-up` | Start Dgraph + Redis containers |
| `make test-integration-setup` | Start containers + seed test data |
| `make test-integration` | Start containers, seed data, run all integration tests |
| `make test-integration-down` | Stop and remove containers |
| `make test-integration-clean` | Full cycle: start, seed, test, tear down |
| `make test-all` | Unit tests + full integration cycle |

## Iterative Development

Keep containers running between test runs to speed up iteration:

```bash
make test-integration-setup     # Start containers + seed data (once)
# edit code...
REDIS_ADDR=localhost:6479 \
DGRAPH_PUBLIC_KEY="$(cat public.pem)" \
DGRAPH_PRIVATE_KEY="$(cat private.pem)" \
go test -tags integration -v -count=1 -timeout 120s ./db/... ./web/handlers/...
# edit more, re-run tests...
make test-integration-down      # Tear down when done
```

To re-seed data after schema or seed changes:

```bash
go run ./cmd/testsetup
```

## Checking Compilation Without Containers

To verify integration test files compile without needing running Dgraph containers:

```bash
go vet -tags integration ./db/... ./web/handlers/...
```

This catches syntax errors, type mismatches, and import issues without executing any tests.

## Architecture

### Docker Setup

`docker-compose.test.yml` runs Dgraph zero + alpha using `dgraph/dgraph:v22.0.2` (matching the project's `DGRAPH_RELEASE`) and Redis 7 Alpine.

Ports are offset by +100 to avoid collision with dev instances:

| Service | Internal Port | Host Port |
|---------|--------------|-----------|
| zero gRPC | 5080 | 5180 |
| zero HTTP | 6080 | 6180 |
| alpha HTTP | 8080 | 8180 |
| alpha gRPC | 9080 | 9180 |
| Redis | 6379 | 6479 |
| MinIO S3 | 9000 | 9100 |
| MinIO console | 9101 | 9101 |

No volumes are mounted - data is ephemeral and destroyed on `docker compose down`.

The `REDIS_ADDR` environment variable controls which Redis the session cache connects to (default: `localhost:6379`). The Makefile test target sets `REDIS_ADDR=localhost:6479` to use the Docker Redis.

### Test Data Setup

Test data is seeded once by `cmd/testsetup` before any test packages run. The program is split into two files plus an embedded data file:

| File | Role |
|------|------|
| `cmd/testsetup/main.go` | Orchestration: wait for Dgraph, drop, load schema, poll until predicates are visible, then call seed |
| `cmd/testsetup/seed.go` | Substitutes password placeholders into the embedded N-Quads and runs the gRPC mutation |
| `cmd/testsetup/seed.nq` | Static N-Quads dataset (embedded via `//go:embed`) — edit this to change seed data |

The setup is resilient to Dgraph startup races: schema upload retries on transient `Server not ready` errors, and a schema-predicate poll runs before seeding (so mutations don't race ahead of schema propagation).

Each test package's `TestMain` only verifies that the data exists (no setup logic).

**Shared seed data** (see `seed.nq` for the full picture):
- `test-org` (Public): general-purpose org with users `testuser` / `testuser2`, Owner + Coordinator roles, one tension with blob/event/comment/label, plus tension and project templates (recursive + non-recursive)
- `sec-org` (Private): security/visibility org with `private-circle` (Private) and `secret-circle` (Secret) sub-circles, plus three Projects scoped to each visibility level

**Nameid convention:**
- Root org: `"test-org"` (no trailing `#`)
- Roles at root: `"test-org##@testuser"` (double `##` separator, empty middle part)
- Sub-circle roles: `"org#circle#@user"` (three parts separated by `#`)

### Build Tags

All integration test files use `//go:build integration`. This means:
- `go test ./...` (and `make test`) skips them entirely
- Only `go test -tags integration` runs them

Tests that call external services (e.g. Matrix webhook) use `//go:build external` and are excluded from both unit and integration runs. Run them with `make test-external` or `go test -tags external -v ./internal/tools/...`.

### DQL Bypasses @auth

DQL queries go through gRPC and skip the Dgraph GraphQL auth layer. No JWT keys are needed for seeding test data.

### Shared Test Constants

`internal/testutil/config.go` exports connection addresses, usernames, and passwords used by all integration test packages and `cmd/testsetup`. This prevents drift when ports or credentials change.

### Test Packages

#### `db/` — DQL Query & Mutation Tests
- `integration_test.go` - TestMain: verify seed data exists
- `integration_query_test.go` - Read-only tests (CountHas, Exists, GetFieldByEq, IsChild, GetChildren, HasCoordos, GetUserRoles, QueryDql, Meta). All tests use `t.Parallel()` and related scenarios are grouped as subtests (e.g. `TestExists_Integration/found`, `TestExists_Integration/not_found`).
- `integration_mutation_test.go` - Write tests (SetFieldByEq, UpgradeMember, Gamma, markAllAsRead, upsertActivity). Mutation tests restore original values after modifying data.

#### `web/handlers/` — HTTP Handler Tests

Uses a real chi router with JWT middleware and tests handlers end-to-end.

- `integration_test.go` - TestMain: verify seed data, start mock email server, build test router. Also provides shared helpers: `doRequest()`, `loginAs()`, `requireStatus()`, `requireJWTCookie()`.
- `integration_auth_test.go` - Auth handler tests (Login, Logout, Signup, SignupValidate, TokenAck, UpdatePassword)
- `integration_org_test.go` - Org handler tests (CreateOrga, SetUserCanJoin, SetGuestCanCreateTension)
- `integration_files_test.go` - `/file/*` end-to-end tests across the three anchor kinds: comment attachments (auth + MIME-sniff + cross-tension rejection), user/org avatars (replace-on-upload + visibility GET), inline-screenshot rewrite (`embedded` flag + code-block masking + parallel uploads), DELETE 404 leak guard, comment-delete S3 GC. Uses MinIO from compose. Companion unit tests for the markdown-rewrite helper live in `files_rewrite_test.go` (no `integration` tag — runs under plain `go test`).

**Dependencies mocked:**
- Email API: A local `httptest.Server` accepts all POST requests (configured via `email.SetTestConfig`)
- Redis: Uses the Docker Redis on port 6479 (configured via `REDIS_ADDR` env var)
- S3: Uses the Docker MinIO on port 9100; the bucket (`fractale-test`) is bootstrapped by `cmd/testsetup` via `storage.EnsureBucket`. Test setup constants live in `internal/testutil/config.go`.

### Adding New Tests

1. Add test functions to existing `integration_*_test.go` files (or create new ones with `//go:build integration`)
2. If new seed data is needed, edit `cmd/testsetup/seed.nq` (or `seed.go` if Go-side substitution is required)
3. Mutation tests should restore original values to avoid interfering with other tests

## Pitfalls & Lessons Learned

### Nameid format — no trailing `#` on root orgs

Root org nameids are bare strings like `"test-org"`, **not** `"test-org#"`. The `#` is only a separator in child nameids. Getting this wrong causes `Nid2pid` mismatches:

```
Nid2pid("test-org##@testuser") → "test-org"   (correct)
```

If the org nameid were `"test-org#"`, the `UserHasCoordoRole` check would fail because `"test-org" != "test-org#"`. This is because `Nid2pid` splits by `#` and when `parts[1] == ""`, it returns `parts[0]` directly (stripping the trailing separator).

### Mutation tests must restore exact original values

`UpgradeMember` sets both `Node.role_type` and `Node.name` to the given role type string. If a test changes a role from `Owner` to `Guest` for verification, it must restore to `Owner` (the exact original), not `Member`. Otherwise subsequent tests that check for Owner/Coordinator roles will fail.

### Parallel test packages share Dgraph and Redis

Both `db/` and `web/handlers/` test packages run in parallel by default and share the same Dgraph and Redis instances. This means:
- Mutation tests in `db/` can temporarily corrupt data seen by `web/handlers/` tests
- Redis role cache (12-second TTL in `MaybeRefresh`) may serve stale data if roles were modified by another package mid-test
- Keep mutation windows short and always restore data immediately

### JWT keys and env var fallback

The `db.init()` loads JWT signing keys from file paths in viper config. When those files don't exist (common in test CWD), it falls back to `DGRAPH_PUBLIC_KEY`/`DGRAPH_PRIVATE_KEY` env vars. The Makefile passes these via `$(cat public.pem)` / `$(cat private.pem)`.

### Email mock is required for Signup

The `Signup` handler calls `email.SendVerificationEmail()` which **panics** on error. The test `TestMain` starts a mock HTTP server and configures it via `email.SetTestConfig()` before any handler tests run.
