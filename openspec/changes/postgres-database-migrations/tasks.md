# Tasks — postgres-database-migrations

> **Status**: PR-A applied, PR-B/C/D pending
> **Date**: 2026-06-21
> **Driver**: braejan
> **Project**: cachicamas (witsaba)
> **Base branch**: `feat/add-postgres-database-migrations` (= `main` @ `7939de0`)
> **Persistence**: hybrid (this file + Engram `sdd/postgres-database-migrations/tasks`)
> **Depends on**:
> - `sdd/postgres-database-migrations/proposal` (Engram id 1586)
> - `sdd/postgres-database-migrations/spec` (Engram id 1587)
> - `sdd/postgres-database-migrations/design` (Engram id 1590)
> - `decision/postgres-database-migrations-user-decisions-on-explore-open-questions` (Engram id 1585)
> - `adr/pgx-v5-stdlib-adapter` (Engram id 1588) — driver
> - `adr/goose-v3-embed-fs` (Engram id 1589) — migration library

## PR progress (apply phase)

- [x] **PR-A — Foundation** (Phases 1-3): pgx dep + driver tests + driver impl + Makefile `test/integration` target. **PR #4** at https://github.com/witsaba/cachicamas/pull/4, opened 2026-06-21. **6 work-unit commits** (see apply-progress Engram id 1593). Gates: `make test` PASS, `make test/integration` PASS, `make lint` PASS, `make build` PASS.
- [x] **PR-B — Runner** (Phases 4-5): goose dep + embed.FS + hello-world SQL + runner + domain port + application service. **PR #5** at https://github.com/witsaba/cachicamas/pull/5, opened 2026-06-21. **9 work-unit commits**. Gates: `make test` PASS, `make test/integration` PASS, `make lint` PASS, `make build` PASS. Hexagonal rule preserved.
- [ ] **PR-C — Wire + infra** (Phases 6-7): `cmd/server/main.go` pre-Echo hook + `01-init.sql` two-line change. Scoped infra exception in effect (PR-C only).
- [ ] **PR-D — Docs + verify + archive** (Phases 8-10): `migration/README.md` + scenario walk-through + sdd-archive.

## Hard constraints (from proposal + design)

- **Strict TDD is ON** — every behavior task is `— RED` (test first) then `— GREEN` (impl).
- **Work-unit commits** — each task or logical group lands as ONE commit (or a small number of reviewable commits). NO giant batched commits.
- **Hexagonal boundaries** — only `migration/runner.go` imports `goose`; only `migration/postgres/driver.go` imports `pgx`. `domain/` imports nothing from the migration slice; `application/` imports `domain` only.
- **Scoped infra exception** — ONLY `infra/postgres/init/01-init.sql` may be touched, only the two lines specified (owner change + timezone). Do NOT widen.
- **Conventional commits only** — no `Co-Authored-By` trailer.
- **No other change folder** — `cachicamas-tail-sampling/`, `cachicamas-deep-healthcheck/` are OFF LIMITS.

---

## Phase 0 — Pre-flight

- [ ] 0.1 Verify dev environment: `go version` (= 1.26.x), `docker --version`, `docker compose version`, `curl http://localhost:4318` (OTel) or note absence
- [ ] 0.2 Confirm branch: `git branch --show-current` reports `feat/add-postgres-database-migrations` (create it from `main` if missing: `git switch -c feat/add-postgres-database-migrations main`)
- [ ] 0.3 Read end-to-end: `proposal.md` (277 lines), `specs/db-migrations/spec.md` (410 lines), `design.md` (392 lines), `explore.md` (331 lines)
- [ ] 0.4 Read `openspec/AGENTS.md` and `openspec/config.yaml` rules one more time
- [ ] 0.5 Resolve `delivery_strategy` with the user BEFORE Phase 1 if the forecast below is High (already resolved as `ask-on-risk`; see "Review workload forecast" at the end)

---

## Phase 1 — Dependencies (ADR-grounded)

> **Goal**: bring in the two new top-level deps with their ADRs cited. No behavior yet.

- [x] 1.1 Add `github.com/jackc/pgx/v5` to `backend/database_administrator/go.mod` — ADR-001 (`adr/pgx-v5-stdlib-adapter` / Engram id 1588). Commit: `feat(deps): add jackc/pgx/v5 postgres driver` — **DONE in PR-A** (commit `f4955d3`, dep added as `github.com/jackc/pgx/v5 v5.10.0`)
- [ ] 1.2 Add `github.com/pressly/goose/v3 v3.27.1` (pinned exactly) to `backend/database_administrator/go.mod` — ADR-002 (`adr/goose-v3-embed-fs` / Engram id 1589). Commit: `feat(deps): add pressly/goose v3.27.1 migration library` — **PR-B**
- [x] 1.3 Run `go mod tidy` from `backend/database_administrator/`; commit `go.mod` + `go.sum` together if anything changed. Commit: `chore(deps): go mod tidy after pgx and goose additions` — **DONE in PR-A** (rolled into `f4955d3`)

**Verification**: `cd backend/database_administrator && go build ./...` compiles (no calls yet).

---

## Phase 2 — Test infrastructure (TDD: RED)

> **Goal**: failing tests for the driver and runner exist. No production code yet. Both new test files fail with the expected reason.

- [x] 2.1 Write `backend/database_administrator/src/migration/postgres/driver_test.go` — table-driven cases covering (a) `DATABASE_URL` set → parses; (b) `DATABASE_URL` unset + `POSTGRES_*` set → assembles DSN; (c) neither set → returns error; (d) `DATABASE_URL` malformed → returns error. Each case asserts the resulting `*sql.DB` and `Ping()` behavior. Mark as `— RED` — **DONE in PR-A** (commit `c262b06`; env hardening in `590fdd3`; errcheck in `e7fbd51`). Implemented as `TestLoadConfigFromEnv` (6 subtests) + `TestApplyPoolSettings` + 2 integration tests for `Open` (`TestOpen_Ping`, `TestOpen_ConnectError`).
- [ ] 2.2 Write `backend/database_administrator/src/migration/runner_test.go` — integration test gated on `os.Getenv("INTEGRATION") == "1"`; covers (a) first boot applies the hello-world migration and inserts one row in `public.schema_migrations`; (b) second boot applies zero migrations; (c) lexicographic order tolerates non-monotonic timestamps (drop a synthetic older file in dev build); (d) advisory lock prevents two parallel runs from double-applying (table-driven via two `runner.Up` calls in goroutines). Mark as `— RED` — **PR-B**
- [x] 2.3 Add `make test/integration` target in `backend/database_administrator/Makefile` that boots compose Postgres (`docker compose up -d postgres`), waits for `pg_isready -h localhost -p 5432 -U cachicamas`, then runs `INTEGRATION=1 go test -race -v ./src/migration/...`. Commit: `test(migration): add integration target and RED tests for driver and runner` — **DONE in PR-A** (commit `7b45ee7`). **Note**: the target runs `INTEGRATION=1 go test -race -v ./...` rather than `./src/migration/...` to keep the integration gate simple and catch any future integration test in any package. If the reviewer wants tighter scope, easy to tighten in PR-B.
- [x] 2.4 Run `make test` and `make test/integration` from `backend/database_administrator/` — confirm both fail with the expected reason (driver_test: "undefined: NewDriver" or similar; runner_test: skipped under unit, fails under INTEGRATION with "undefined: NewGooseRunner"). Commit message captures the RED state in the body. — **DONE in PR-A**: RED state was "undefined: LoadConfigFromEnv / Config / Open / applyPoolSettings" (compile error in `c262b06`). GREEN state at the tip of PR-A: `make test` PASS (6 unit subtests + apply-pool test, 2 integration tests SKIP), `make test/integration` PASS (boots compose, runs 2 integration tests, stops compose).

**TDD discipline**: do NOT write `driver.go` or `runner.go` before step 2.1 and 2.2 fail to compile/run. The test file's failure to compile IS the red step for the driver; the runner_test gating on INTEGRATION is the red step for the runner.

---

## Phase 3 — Driver (TDD: GREEN)

> **Goal**: the driver tests go green. Pure adapter, no domain coupling.

- [x] 3.1 Implement `backend/database_administrator/src/migration/postgres/driver.go` — exports `NewDriver(cfg Config) (*sql.DB, error)`. Reads `DATABASE_URL` first, falls back to `POSTGRES_HOST`/`POSTGRES_PORT`/`POSTGRES_DB`/`POSTGRES_USER`/`POSTGRES_PASSWORD`. Calls `db.Ping()` for fail-fast. Registers pgx stdlib adapter. Mark as `— GREEN` (driver_test passes). Commit: `feat(migration): postgres driver factory with DATABASE_URL precedence` — **DONE in PR-A** (commit `8af91f2`). Exports: `Config`, `LoadConfigFromEnv`, `Open`, `applyPoolSettings` (unexported helper). Uses `db.PingContext` under a `ConnectTimeout`-bounded context for the fail-fast guarantee.
- [x] 3.2 Run `make test` — confirm `driver_test` passes; no other unit tests regressed — **DONE in PR-A**: `make test` PASS.
- [x] 3.3 Run `make lint` — confirm zero issues — **DONE in PR-A**: `make lint` PASS, 0 issues.

---

## Phase 4 — Runner (TDD: GREEN)

> **Goal**: the runner tests go green. Hexagonal: this is the ONLY file that imports `goose`.

- [x] 4.1 Implement `backend/database_administrator/src/migration/embed.go` — `//go:embed sql/*.sql` → exported `MigrationsFS embed.FS`. One tiny file, one variable. Commit: `feat(migration): embed SQL migrations via embed.FS` — **DONE in PR-B** (commit `f3a2b32`)
- [x] 4.2 Create `backend/database_administrator/src/migration/sql/20260621120000_hello_world.sql` with body `SELECT 1;` and a header comment explaining the no-op. Commit: `feat(migration): add hello-world no-op migration` — **DONE in PR-B** (commit `7c30d70`, combined with 4.3 — goose v3.27.1 idiom is a SINGLE file with both `-- +goose Up` and `-- +goose Down` blocks; the legacy v2 `XXX.sql` + `XXX.down.sql` pair is rejected as a duplicate version)
- [x] 4.3 Create `backend/database_administrator/src/migration/sql/20260621120000_hello_world.down.sql` with body `SELECT 1;` (no-op down for symmetry, keeps `goose down` working locally). Commit (combined with 4.2 OR separate): `feat(migration): add hello-world no-op down` — **DONE in PR-B** (commit `7c30d70`, combined with 4.2 — see 4.2 note about the v3 idiom)
- [x] 4.4 Implement `backend/database_administrator/src/migration/runner.go` — `GooseRunner` struct with `NewGooseRunner(db *sql.DB, tableName string, logger *slog.Logger) *GooseRunner`. `Up(ctx) ([]domain.Version, error)` calls `goose.SetBaseFS(MigrationsFS)`, `goose.SetDialect("postgres")`, `goose.SetTableName(r.tableName)`, `goose.WithSessionLocker(lock.NewPostgresSessionLocker())`, then `goose.UpContext(ctx, r.db, "sql")`. `Status(ctx) ([]domain.Version, error)` returns goose's current versions. Wraps every `Up` call in OTel span `migration.up` and `slog.Info`. Imports `github.com/pressly/goose/v3` and `github.com/pressly/goose/v3/lock` ONLY in this file. Mark as `— GREEN` (runner_test passes). Commit: `feat(migration): goose runner with advisory lock and OTel span` — **DONE in PR-B** (commit `f096076`, RED in `3d82107`, lint cleanup in `2f49936`). Implementation uses `goose.NewProvider` with `fs.Sub(MigrationsFS, "sql")` per the v3 idiom; the legacy `goose.SetBaseFS` + `goose.UpContext` path does NOT honour `WithSessionLocker` (it's a `ProviderOption`, not an `OptionsFunc`).
- [x] 4.5 Run `make test`, `make test/integration`, `make lint` — all green — **DONE in PR-B**

---

## Phase 5 — Domain port + Application use case

> **Goal**: hexagonal port exists; application service wraps the port with OTel/slog. No goose/pgx import here.

- [x] 5.1 Implement `backend/database_administrator/src/domain/migration.go` — `Version struct { ID int64; Description string; AppliedAt time.Time }` and `Runner interface { Up(ctx context.Context) ([]Version, error); Status(ctx context.Context) ([]Version, error) }`. **No imports from migration package.** Commit: `feat(domain): migration Runner port and Version struct` — **DONE in PR-B** (commit `dcea916`)
- [x] 5.2 Write `backend/database_administrator/src/application/migration_service_test.go` — unit test with a fake `domain.Runner` (no live DB). Asserts that `Up` calls the runner, wraps the call in an OTel span, logs the result, and returns any error. Mark as `— RED` — **DONE in PR-B** (commit `a9977c5`)
- [x] 5.3 Implement `backend/database_administrator/src/application/migration_service.go` — `MigrationService` struct holds `runner domain.Runner`, `logger *slog.Logger`, `tracer trace.Tracer`. `Up(ctx)` wraps `runner.Up(ctx)` in OTel span `migration.up` with attrs `db.system=postgresql`, `migration.dir`, `migration.applied_count`, `migration.duration_ms` (plus `migration.error` and `migration.error.kind` on failure). `slog.Info` on success, `slog.Error` on failure. `Status(ctx)` delegates. **No goose/pgx imports.** Mark as `— GREEN`. Commit: `feat(application): migration service with OTel and slog instrumentation` — **DONE in PR-B** (commit `ef049f1`)
- [x] 5.4 Run `make test`, `make lint` — all green — **DONE in PR-B**

---

## Phase 6 — Wire into `cmd/server/main.go`

> **Goal**: the composition root calls the runner BEFORE Echo binds. Fail-fast on error.

- [ ] 6.1 Modify `backend/database_administrator/src/cmd/server/main.go` — after `otel.Init(ctx)` and logger setup, build the driver (`postgres.NewDriver(...)`), build the runner (`migration.NewGooseRunner(db, "schema_migrations", logger)`), build the service (`application.NewMigrationService(runner, logger, otel.Tracer("database_administrator"))`), call `migrationService.Up(ctx)` under a bounded context (`MIGRATION_TIMEOUT`, default 30s). On error: `logger.Error("migration.up failed; exiting", "error", err)` then `os.Exit(1)`. Existing Echo wiring follows. Commit: `feat(server): run migrations before Echo binds`
- [ ] 6.2 Run `make test`, `make build`, `make lint` — all green
- [ ] 6.3 Boot the stack and verify: `docker compose up -d --build database_administrator` then `docker compose logs database_administrator | grep migration.up` — confirm one INFO line with `applied_count=1 duration_ms=...` and no errors

---

## Phase 7 — `infra/postgres/init/01-init.sql` (SCOPED EXCEPTION IN EFFECT)

> **Scope**: ONLY this file. Only the two lines specified. Do NOT widen the exception.

- [ ] 7.1 Read `infra/postgres/init/01-init.sql` end-to-end; confirm line numbers (queen role ≈ line 75; `CREATE TABLE public.schema_migrations` ≈ line 99)
- [ ] 7.2 Insert TWO lines AFTER the queen role creation (line 75) and BEFORE `CREATE TABLE public.schema_migrations` (line 99), in this order:
  ```sql
  ALTER DATABASE current_database() OWNER TO queen;
  ALTER DATABASE current_database() SET timezone = 'UTC';
  ```
  Add a short comment block above them explaining the ordering rationale. Commit: `feat(infra): transfer cachicamas_pg ownership to queen and pin timezone to UTC`
- [ ] 7.3 `git diff infra/postgres/init/01-init.sql` — confirm the ONLY change to this file is the two new lines + the comment block above them. NO other infra files in the diff
- [ ] 7.4 Wipe the volume and re-test: `docker compose down -v && docker compose up -d --build`. Then verify:
  - `docker compose exec postgres psql -U cachicamas -d cachicamas_pg -c "SHOW timezone"` returns `UTC`
  - `docker compose exec postgres psql -U queen -d cachicamas_pg -c "SHOW timezone"` returns `UTC` (no explicit SET)
  - `docker compose logs database_administrator | grep migration.up` shows one INFO line, `applied_count=1`
  - `docker compose exec postgres psql -U queen -d cachicamas_pg -c "SELECT count(*) FROM public.schema_migrations WHERE version = '20260621120000'"` returns `1`
- [ ] 7.5 Restart (no volume wipe): `docker compose stop database_administrator && docker compose start database_administrator`. Confirm logs show `migration.applied_count=0` (idempotency)

---

## Phase 8 — Migration README + docs

> **Goal**: operator-facing documentation. How to add a migration, how to recover from a half-applied one.

- [ ] 8.1 Write `backend/database_administrator/src/migration/README.md` covering:
  - **Naming**: timestamp-prefixed `YYYYMMDDHHMMSS_<context>_<description>.sql`; regex `^\d{14}_[a-z0-9_]+\.sql$`
  - **Add a migration**: `goose create <name> sql` from `backend/database_administrator/`; commit the resulting `*.sql` AND optional `*.down.sql`
  - **Up vs down**: schema-only → write `.down.sql`; data moves → comment `// TODO: data migration, down is destructive` in the up file
  - **Recovery from a half-applied migration** (per design §15.2 and spec R-DBMIG-060): goose does NOT use a `dirty` column. Verify the transaction rolled back via `SELECT * FROM public.schema_migrations` — if no row for the version, the next boot re-applies it. If a row exists but the schema is incomplete, manually `DELETE FROM public.schema_migrations WHERE version = '...'` after fixing the schema
  - **Advisory lock**: `pg_try_advisory_lock(42)`, non-blocking; second replica logs `waiting for lock` and retries
  - **Operational boundaries**: `MIGRATION_TIMEOUT` default 30s; large migrations must use online ops (`CREATE INDEX CONCURRENTLY`, `ADD COLUMN ... DEFAULT NULL` then backfill)
  - **Why one global dir, schema-qualified names**: design §8 rationale
  Commit: `docs(migration): operator README with naming, recovery, and lock semantics`
- [ ] 8.2 Run `make lint`; if markdown lint is configured, ensure clean

---

## Phase 9 — Verify

- [ ] 9.1 Run all checks from a clean tree: `make test`, `make test/integration`, `make lint`, `make build` — all green
- [ ] 9.2 Walk the spec scenarios R-DBMIG-001 through R-DBMIG-070 against the running stack; mark each PASS in the spec's review checklist
- [ ] 9.3 `git log --oneline feat/add-postgres-database-migrations ^main` — confirm work-unit commits (one per logical unit: deps, driver-test, driver-impl, embed+sql, runner-impl, port, service, main wire, init.sql, docs), NOT one giant commit
- [ ] 9.4 Hexagonal check: `grep -RE "pressly/goose" backend/database_administrator/src/` returns ONLY files under `src/migration/`; `grep -RE "jackc/pgx" backend/database_administrator/src/` returns ONLY `src/migration/postgres/driver.go` and `src/migration/runner.go` (the runner uses stdlib registration)

---

## Phase 10 — PR + archive

- [ ] 10.1 Open PR from `feat/add-postgres-database-migrations` to `main` with:
  - Summary linking to the proposal, spec, design, and tasks
  - **Chain Context** section (only if delivery_strategy resolved to chaining — see "Review workload forecast")
  - Verification commands run locally
  - Review checklist (copy from spec.md §Review checklist)
- [ ] 10.2 After merge, run `sdd-archive` to move `openspec/changes/postgres-database-migrations/` into `openspec/changes/archive/2026-06-21-postgres-database-migrations/` and copy the spec into `openspec/specs/db-migrations/spec.md`
- [ ] 10.3 Confirm the spec is reachable at `openspec/specs/db-migrations/spec.md` (the main specs directory)

---

## Review workload forecast (MANDATORY per `openspec/config.yaml` rules)

| Task group | Files touched | Estimated LoC | Risk |
|---|---|---|---|
| Phase 1 (deps) | `backend/database_administrator/go.mod`, `go.sum` | ~10 | Low |
| Phase 2 (test infra) | NEW: `migration/postgres/driver_test.go`, `migration/runner_test.go`; MODIFIED: `backend/database_administrator/Makefile` | ~180 | Low |
| Phase 3 (driver) | NEW: `migration/postgres/driver.go` | ~60 | Low |
| Phase 4 (runner) | NEW: `migration/embed.go`, `migration/runner.go`, `migration/sql/20260621120000_hello_world.sql`, `migration/sql/20260621120000_hello_world.down.sql` | ~150 | Medium |
| Phase 5 (port + service) | NEW: `domain/migration.go`, `application/migration_service.go`, `application/migration_service_test.go` | ~110 | Low |
| Phase 6 (main.go) | MODIFIED: `backend/database_administrator/src/cmd/server/main.go` | ~30 | Low |
| Phase 7 (01-init.sql) | MODIFIED: `infra/postgres/init/01-init.sql` (+2 SQL lines + comment block) | ~15 | Low |
| Phase 8 (README) | NEW: `migration/README.md` | ~80 | Low |
| Phase 9 (verify only) | — | 0 | — |
| Phase 10 (PR + archive) | meta | 0 | — |
| **Total** | — | **~635** | **High** |

## Budget verdict

- **400-line PR review budget: HIGH risk** (~635 LoC, ~59% over budget)
- **Chained PRs recommended: YES**
- **Decision needed before apply: YES** — must pause for user input

### Proposed chained split

- **PR-A — Foundation** (Phases 1-3): deps + driver tests + driver impl + Makefile integration target — ~250 LoC, scope: bring pgx + goose in, prove the DSN factory works. Verifies driver_test goes RED → GREEN.
- **PR-B — Runner** (Phases 4-5): embed.FS + hello-world SQL + runner + domain port + application service + service tests — ~260 LoC, scope: goose.Up wrapper behind the domain port with OTel/slog. Verifies runner_test (unit + integration) goes RED → GREEN.
- **PR-C — Wire + infra** (Phases 6-7): main.go pre-Echo hook + 01-init.sql two-line change — ~45 LoC, scope: the container boots, runs migrations, crashes on error, applies UTC.
- **PR-D — Docs + verify + archive** (Phases 8-10): README + scenario walk-through + PR + sdd-archive — ~80 LoC.

PR-D is small enough that it could be merged into PR-C; the split keeps review focus sharp either way.

### Chain strategies (chain_strategy NOT cached yet)

- **stacked-to-main**: each PR (A, B, C, D) merges to main in order. Fast iteration, fix on the go. Best for speed-first teams and independent slices.
- **feature-branch-chain**: a tracker branch `feat/postgres-database-migrations-chain` accumulates the integration. PR-A targets the tracker; PR-B targets PR-A's branch; PR-C targets PR-B's; PR-D targets PR-C's. Only the tracker merges to main. Best for rollback control and coordinated releases.

---

## Pause-for-input block (DECISION NEEDED)

The forecast is HIGH risk (~635 LoC, ~59% over the 400-line PR budget). Per the cached `delivery_strategy: ask-on-risk` and `openspec/config.yaml` rules, I MUST stop and ask the user before `sdd-apply` launches.

### What the orchestrator should ask the user

> The total forecast for `postgres-database-migrations` is **~635 LoC**, which is **~59% over the 400-line PR review budget**. Per your cached `delivery_strategy: ask-on-risk`, I'm pausing before `sdd-apply`. Please pick one:
>
> **(a) Split into chained PRs** (recommended). I propose 4 PRs:
> - PR-A — Foundation (deps + driver, ~250 LoC)
> - PR-B — Runner (embed + SQL + runner + port + service, ~260 LoC)
> - PR-C — Wire + infra (main.go + 01-init.sql, ~45 LoC)
> - PR-D — Docs + verify + archive (~80 LoC; can be folded into PR-C if you prefer 3 PRs)
>
> If you pick (a), which chain strategy?
> - **`stacked-to-main`**: each PR merges to main in order. Fast iteration.
> - **`feature-branch-chain`**: PR-A targets a tracker branch; PR-B targets PR-A's branch; PR-C targets PR-B's; PR-D targets PR-C's. Only the tracker merges to main. Best for rollback control.
>
> **(b) Proceed as a single PR with `size:exception`**. The whole change lands in one PR (~635 LoC) with an explicit size-exception note in the PR body. You accept the higher review cognitive load.
>
> **(c) Reduce scope**. Drop one phase to bring the total under 400 LoC. The most defensible drop is Phase 8 (README) — but operator recovery is a spec requirement, so dropping it shifts burden to memory. Phase 7 (the 01-init.sql change) is the OTHER side of the scoped exception and can't be split out without splitting the change.
>
> **My recommendation**: (a) with **`stacked-to-main`** — four small PRs, each independently green, each independently revertable, each inside the 400-line budget.

---

## Review checklist

- [ ] reviewer can confirm tasks follow hierarchical numbering (`0.1`, `1.1`, ..., `10.3`) and group by phase
- [ ] reviewer can confirm every behavior task carries `— RED` then `— GREEN` markers (TDD discipline)
- [ ] reviewer can confirm each task is small enough to land in a single PR OR is one of the proposed chained PRs
- [ ] reviewer can confirm the Review Workload Forecast is present and the Budget Verdict is correct (~635 LoC, High risk)
- [ ] reviewer can confirm the proposed PR split aligns with the chained-pr skill (independent slices, each green before next)
- [ ] reviewer can confirm the scoped infra exception is NOT widened (only `01-init.sql`, only the 2 lines)
- [ ] reviewer can confirm no other change folder is referenced (`cachicamas-tail-sampling`, `cachicamas-deep-healthcheck`)
- [ ] reviewer can confirm `openspec/specs/` is NOT touched here (it stays empty until archive)
- [ ] reviewer can confirm work-unit commits are required (no giant batched commits)
- [ ] reviewer can confirm the hex rule is preserved: only `migration/runner.go` imports goose, only `migration/postgres/driver.go` imports pgx, domain imports nothing from migration, application imports domain only
- [ ] reviewer can confirm `delivery_strategy: ask-on-risk` triggered the pause-for-input block because forecast > 400 LoC
- [ ] reviewer can confirm the Pause-for-input block proposes 4 chained PRs and asks the user to choose a chain strategy