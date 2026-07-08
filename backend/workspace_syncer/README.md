# workspace_syncer

The Go service that owns git, the local filesystem layout, and the actual
`git clone` / `git worktree add` invocations for the cachicamas Workspaces
sync feature. It is the **data-plane** complement to `database_administrator`
(control-plane).

## What it does

1. Receives a `POST /internal/clone-and-validate` request from
   `database_administrator` carrying a `job_id`, `workspace_id`, the
   GitHub repo's `owner`/`repo`, the `default_branch`, and the user's
   OAuth `access_token`.
2. Verifies the token's permissions against
   `GET https://api.github.com/repos/{owner}/{repo}` (checks
   `permissions.push === true` — the precondition for both `git
   worktree` operations and PR creation).
3. Clones the repository into the canonical filesystem path
   `/data/workspaces/{workspace_id}/{owner}/{repo}.git/` as a **bare
   mirror** using `git clone --bare` with a 90s `context.WithTimeout`.
4. Runs a worktree probe: `git -C <path> worktree add /tmp/probe HEAD`.
   A non-zero exit code is treated as a `WORKTREE_PROBE_FAILED`.
5. Posts the outcome back to
   `database_administrator`'s `POST /internal/sync-callback` with
   `status = "done"` + `commit_sha_after` on success, or
   `status = "failed"` + `error_code` + `error_message` on any
   failure.
6. On startup, runs a sweep that removes `/data/workspaces/{id}/...`
   directories whose `id` no longer has a live `sync_job` (defends
   against orphans from soft-deleted workspaces).

## What it does NOT do

- **No Postgres access.** The service has zero `database/sql` / pgx
  imports. The `sync_job` table is owned by `database_administrator`.
- **No public HTTP surface.** The Echo instance only listens on the
  docker network. No route is exposed to the public internet.
- **No `go-git`.** The service shells out to the system `git` binary
  via `os/exec`. Rationale: `go-git` does not have full worktree
  support; the system `git` is the canonical implementation.

## Configuration

The service reads the following environment variables:

| Var | Default | Purpose |
| --- | --- | --- |
| `INTERNAL_SERVICE_TOKEN` | (required) | Bearer token presented by `database_administrator` on every request. The service fails to boot with an empty value. |
| `WORKSPACE_SYNCER_PORT` | `8081` | HTTP listener port. |
| `WORKSPACE_SYNCER_DATA_DIR` | `/data/workspaces` | Filesystem path for cloned repos. |
| `DATABASE_ADMINISTRATOR_URL` | `http://database_administrator:8080` | URL of the control plane (for the sync callback and the live-workspaces sweep lookup). |
| `GIT_TIMEOUT_SECONDS` | `90` | Per-clone timeout in seconds. |
| `OTEL_EXPORTER_OTLP_ENDPOINT` | (unset) | OTLP/gRPC collector endpoint (traces). Empty falls back to a no-op tracer. |
| `SERVICE_ENV` | — | Set to `development` to enable the dev-only fail injection (future; not in v1). |

## Endpoints

### `POST /internal/clone-and-validate`

Authenticated with `Authorization: Bearer <INTERNAL_SERVICE_TOKEN>`.

Request:

```json
{
  "job_id": 42,
  "workspace_id": 7,
  "owner": "octocat",
  "repo": "hello-world",
  "default_branch": "main",
  "oauth_token": "gho_..."
}
```

Response on acceptance (the work runs in a goroutine; the 202 just
means "we accepted and started"):

```http
HTTP 202 Accepted
{ "job_id": 42, "status": "running" }
```

Failure envelope (synchronous rejection only — async failures go via the
callback):

```json
{ "error": "validation", "fields": { "owner": "owner is required" } }
```

## Build & run

```bash
# Boot Postgres + Jaeger (project root).
docker compose up -d postgres

# Run the binary.
make run
```

## Tests

```bash
make test                  # unit tests with race detector
make test/integration      # boots compose Postgres, runs integration tests
make lint                  # golangci-lint + go vet (requires `make tools` first)
```

## Architecture invariants (enforced by the test suite)

- **No Postgres access.** The compile-time check in `src/cmd/server/main.go`
  uses `domain.Validator` (a non-DB type) to confirm the dependency
  direction.
- **No public HTTP surface.** The `http.handler` test boots the Echo
  instance on `127.0.0.1:0` and asserts the route is not exposed via
  any reverse proxy.
- **Token redaction.** A custom `slog` handler in `src/otel/logging.go`
  redacts any field matching `oauth_token`, `authorization`, or
  `access_token` (case-insensitive) from every log line.
- **Filesystem path safety.** `src/infrastructure/git/layout.go`'s
  `WorkspacePath` rejects any `workspace_id` that is not a positive
  int64 and any `owner`/`repo` that does not match
  `^[a-zA-Z0-9._-]+$`. Defense-in-depth against path traversal.
- **Shell injection safety.** `src/infrastructure/git/runner.go`'s
  `Clone` always passes args as a `[]string` to `exec.Command`; no
  `sh -c` with interpolated input.

## References

- Proposal: `openspec/changes/2026-07-08-workspace-sync-clone/proposal.md`
- Spec (workspaces delta): `openspec/changes/2026-07-08-workspace-sync-clone/specs/workspaces/spec.md`
- Spec (this service): `openspec/changes/2026-07-08-workspace-sync-clone/specs/workspace-syncer/spec.md`
- Design: `openspec/changes/2026-07-08-workspace-sync-clone/design.md`
- ADRs (persisted in Engram):
  - `adr/workspace-syncer-git-impl`
  - `adr/workspace-syncer-internal-auth`
