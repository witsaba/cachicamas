# database_administrator

The Go service that owns the `database_administrator` schema and exposes
the first cachicamas HTTP API: `/health` (liveness) and `/organizations`
(CRUD over the `organization` table).

## Layout

```
src/
  cmd/server/         composition root (main.go)
  domain/            pure business types — no framework, no DB
  application/       use cases — OTel + slog wrappers around the ports
  infrastructure/    adapters (Postgres / pgx)
    postgres/        organization repository (pgx stdlib)
  interfaces/http/   Echo transport adapters (handlers, routes)
  migration/         goose runner + SQL files
  otel/              tracing/logging setup
```

The package boundaries are hexagonal: `domain` imports nothing from the
outer layers, `application` depends on `domain` only, and
`infrastructure/postgres` is the only file that imports `jackc/pgx` for
organization data access (mirroring the existing `migration/postgres`
rule).

## Run

```bash
# 1. Boot Postgres + Jaeger (project root).
docker compose up -d postgres

# 2. Run the binary — migrations apply on first boot, then Echo binds.
make run
```

The server listens on `:8080` by default. Override with `SERVICE_PORT=9090
make run`.

## Tests

```bash
make test                  # unit tests with race detector
INTEGRATION=1 make test    # also run the pgx integration tests
make test/integration      # boot compose Postgres, run integration tests, stop Postgres
make lint                  # golangci-lint + go vet (requires `make tools` first)
```

## API

### `GET /health`

Liveness probe. Returns `200 {"status":"ok"}`. Emits an OTel span and a
slog line per request. Supports a dev-only fail-injection via
`?fail=true` when `SERVICE_ENV=development`.

### `POST /organizations`

Creates a new organization. Accepts BOTH `Content-Type: application/json`
AND `Content-Type: application/x-www-form-urlencoded` (the Qwik
frontend posts form-encoded; programmatic clients post JSON).

```bash
# JSON
curl -sS -X POST -H 'Content-Type: application/json' \
  -d '{"full_name":"Acme","identification":"acme"}' \
  localhost:8080/organizations
```

```bash
# form-encoded
curl -sS -X POST -H 'Content-Type: application/x-www-form-urlencoded' \
  --data 'full_name=Acme&identification=acme' \
  localhost:8080/organizations
```

**Request fields** (JSON keys shown; form fields are identical):

| Field            | Type   | Required | Rule |
|------------------|--------|----------|------|
| `full_name`      | string | yes      | 3-120 chars after trim |
| `identification` | string | yes      | matches `^[a-z0-9][a-z0-9-]{1,58}[a-z0-9]$` |
| `shortname`      | string | no       | at most 40 chars |
| `email`          | string | no       | RFC 5322 valid |
| `phone`          | string | no       | E.164 `^\+[1-9]\d{1,14}$` |

**Responses:**

| Status | Body |
| -------- | ------ |
| 201 Created | `OrganizationResponse` (see below); `Location: /organizations/{id}` header. |
| 400 Bad Request | `{"error":"validation","fields":{...}}` — fields with the failing rule and the locked message string. |
| 409 Conflict | `{"error":"conflict","message":"This slug is already taken. Try another."}` — `identification` already exists. |
| 500 Internal Server Error | `{"error":"server","message":"Something went wrong. Please try again."}` — unexpected error. |

### `GET /organizations`

REMOVED 2026-07-06 ownboarding. The list endpoint was deleted as
part of the single-tenant model. Use `GET /setup-state` to detect
whether an organization exists at all (the only check the
ownboarding flow needs).

### `GET /organizations/:id`

REMOVED 2026-07-06 ownboarding. The get-by-id endpoint was deleted.
There is no longer a UI surface that needs to look up an organization
by id — the unique org is created via `POST /organizations` (the
ownboarding submit) and consumed via the `/home` route (which gates
on `/setup-state`).

### Workspaces API (2026-07-06)

The workspaces surface exposes 8 endpoints + a GitHub repos proxy. All
endpoints are mounted on the same Echo instance as the organization +
setup-state endpoints. Auth is required for every endpoint; the
`requireOwnboarding` gate (frontend-side) ensures the user has completed
ownboarding before any workspace handler runs.

#### `GET /workspaces`

List live (non-soft-deleted) workspaces for the current organization,
ordered by `created_at DESC`. Capped at 100.

```bash
curl -sS -b cookies.txt localhost:8080/workspaces
```

**Responses:**

| Status | Body |
|--------|------|
| 200 OK | `{"workspaces":[{...}], "truncated": false}` |
| 401 | auth envelope (`error: "github_not_connected"` if no access_token) |

#### `POST /workspaces`

Create a workspace. The primary repo must be in the user's accessible
GitHub repo set (server-side check via `GitHubAccessor.IsRepoAccessible`).

```bash
curl -sS -X POST -H 'Content-Type: application/json' -b cookies.txt \
  -d '{"name":"My Workspace","primary_repository":{"github_id":12345,"full_name":"octocat/hello-world","owner":"octocat","name":"hello-world"}}' \
  localhost:8080/workspaces
```

**Responses:**

| Status | Body |
|--------|------|
| 201 Created | `WorkspaceResponse` + `Location: /workspaces/{id}` |
| 400 / 422 | validation envelope |
| 409 Conflict | duplicate name |
| 502 Bad Gateway | `error: "github_unreachable"` if the GitHub check fails |

#### `GET /workspaces/:id`

Fetch a workspace detail + linked repos.

#### `PATCH /workspaces/:id`

Rename a workspace. `primary_repository` is silently ignored (locked
design decision: primary repo is the workspace's identity, cannot be
renamed via PATCH).

#### `DELETE /workspaces/:id`

Soft-delete the workspace. Linked repos are hard-deleted via FK cascade
in the same transaction. Returns 204.

#### `POST /workspaces/:id/repositories`

Connect a GitHub repo to a workspace.

```bash
curl -sS -X POST -H 'Content-Type: application/json' -b cookies.txt \
  -d '{"github_id":67890,"github_full_name":"octocat/spoon-knife","github_owner":"octocat","github_name":"spoon-knife"}' \
  localhost:8080/workspaces/42/repositories
```

#### `DELETE /workspaces/:id/repositories/:repoId`

Disconnect a repo. Hard delete. Returns 204.

#### `GET /workspaces/:id/repositories`

List linked repos for a workspace, ordered by `added_at ASC`.

#### `GET /github/repos`

Server-side proxy for the authenticated user's GitHub repos. Backed by
a 5-min in-memory cache keyed by user_id. The frontend uses this for
the `GitHubRepoPicker` component.

```bash
curl -sS -b cookies.txt 'localhost:8080/github/repos?page=1&per_page=100&bust_cache=false'
```

**Query params:**
- `page` (1-indexed, default 1)
- `per_page` (max 100, default 30)
- `bust_cache=true` bypasses the 5-min in-memory cache

**Responses:**

| Status | Body |
|--------|------|
| 200 OK | `{"repositories":[{...}], "page":1, "per_page":30, "has_next":false}` |
| 401 Unauthorized | `error: "github_not_connected"` (no access_token) |
| 502 Bad Gateway | `error: "github_unauthorized"`, `"github_rate_limited"`, or `"github_unreachable"` |

### `GET /setup-state`

Returns the install-level "is there at least one organization?" boolean.
The ownboarding gate (frontend `requireOwnboarding` helper) reads
this to decide whether the user lands on `/home` or `/ownboarding`
after authentication.

```bash
curl -sS localhost:8080/setup-state
```

**Responses:**

| Status | Body |
|--------|------|
| 200 OK | `{"hasOrganization": true}` or `{"hasOrganization": false}`. |
| 500 Internal Server Error | `{"error":"server","message":"Something went wrong. Please try again."}` — DB query failed. |

**Implementation note**: uses `SELECT EXISTS (SELECT 1 FROM organizations)`
under the hood — the cheapest possible Postgres query (short-circuits at
the first row, no row materialization). See
`src/application/organization_service.go` (GetSetupState) and
`src/infrastructure/postgres/organization_repo.go` (HasOrganization).

## Wire shape

```json
{
  "id": 42,
  "full_name": "Acme Industrial S.A.",
  "identification": "acme-industrial",
  "shortname": "Acme",
  "is_active": true,
  "email": "hello@acme.example",
  "phone": "+14155552671",
  "created_at": "2026-06-22T18:00:00Z",
  "updated_at": "2026-06-22T18:00:00Z"
}
```

`id` is `BIGSERIAL` end-to-end (Go `int64`, DDL `BIGSERIAL`, wire
`integer`). `is_active` defaults to `true`. `created_at` / `updated_at`
are set by Postgres `DEFAULT now()` and returned via `INSERT ... RETURNING *`.

## Error envelope

Every error response carries one of the four locked envelope shapes.
The user-facing message strings are pinned to the spec vocabulary; a
typo in the strings fails the test suite.

```json
// 400 — validation
{"error":"validation","fields":{"full_name":"Name is required."}}

// 404 — not found
{"error":"not_found","message":"Organization not found."}

// 409 — conflict
{"error":"conflict","message":"This slug is already taken. Try another."}

// 500 — server
{"error":"server","message":"Something went wrong. Please try again."}
```

## Observability

Every endpoint emits an OTel span under the same tracer provider as
`/health` (Jaeger query:
`service.name = "database_administrator" AND name = "organization.*"`):

| Span name             | HTTP route              | Always-emitted attributes                                |
|-----------------------|-------------------------|----------------------------------------------------------|
| `organization.create` | `POST /organizations`   | `http.method=POST`, `http.route=/organizations`, `http.status_code=201` (+ `organization.id` on success) |
| `organization.setup_state` | `GET /setup-state` | `http.method=GET`, `http.route=/setup-state`, `http.status_code=200` (+ `has_organization` on success) |
| `workspace.create` | `POST /workspaces` | `http.method=POST`, `http.route=/workspaces`, `http.status_code=201` (+ `workspace.id` on success) |
| `workspace.list` | `GET /workspaces` | `http.method=GET`, `http.route=/workspaces`, `http.status_code=200` |
| `workspace.get` | `GET /workspaces/:id` | `http.method=GET`, `http.route=/workspaces/:id`, `http.status_code=200` |
| `workspace.update` | `PATCH /workspaces/:id` | `http.method=PATCH`, `http.route=/workspaces/:id`, `http.status_code=200` |
| `workspace.delete` | `DELETE /workspaces/:id` | `http.method=DELETE`, `http.route=/workspaces/:id`, `http.status_code=204` |
| `workspace.add_repo` | `POST /workspaces/:id/repositories` | `http.method=POST`, `http.route=/workspaces/:id/repositories`, `http.status_code=201` |
| `workspace.remove_repo` | `DELETE /workspaces/:id/repositories/:repoId` | `http.method=DELETE`, `http.route=/workspaces/:id/repositories/:repoId`, `http.status_code=204` |
| `workspace.list_repos` | `GET /workspaces/:id/repositories` | `http.method=GET`, `http.route=/workspaces/:id/repositories`, `http.status_code=200` |

Validation failures do NOT emit a span — they short-circuit before
`tracer.Start`.

## Database access rules

- `src/infrastructure/postgres/organization_repo.go` is the only file
  that imports `jackc/pgx` for organization data access.
- All queries are parameterised (`$1, $2, ...`); no string
  interpolation into SQL.
- The repository shares the `*sql.DB` the migration runner opened
  (`migrationpg.Open`); no second pool.

## Configuration

The server reads the following environment variables:

| Var | Default | Purpose |
| ----- | --------- | --------- |
| `DATABASE_URL` | — | Postgres DSN; takes precedence over the `POSTGRES_*` family. |
| `POSTGRES_HOST` | — | Postgres host. |
| `POSTGRES_PORT` | `5432` | Postgres port. |
| `POSTGRES_DB` | — | Database name. |
| `POSTGRES_USER` | — | Database user. |
| `POSTGRES_PASSWORD` | — | Database password. |
| `MIGRATION_TABLE` | `schema_migrations` | goose bookkeeping table. |
| `MIGRATION_TIMEOUT` | `30s` | Bound on the migration runner. |
| `SERVICE_PORT` | `8080` | HTTP listener port. |
| `SERVICE_ENV` | — | Set to `development` to enable the dev-only fail injection on `/health`. |
| `OTEL_EXPORTER_OTLP_ENDPOINT` | — | OTLP/gRPC collector endpoint (traces + logs). |
| `GITHUB_REPOS_CACHE_TTL` | `5m` | TTL for the `/github/repos` in-memory cache. Override for tests. |
