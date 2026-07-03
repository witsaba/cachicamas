# Apply Progress: cachicamas-github-login (PR-1)

> **PR**: `feat/cachicamas-github-login-pr1-schema-identity`
> **Status**: complete; ready for review
> **Persisted**: 2026-07-03 to Engram topic `sdd/cachicamas-github-login/apply-progress`
> **Parent artifacts**: `proposal.md`, `specs/{frontend-auth,backend-auth-middleware,identity-schema}/spec.md`, `design.md`, `tasks.md`

## Scope (PR-1)

- Goose migration `20260703120000_github_login.sql` (identity.user + identity.account + organization.owner_user_id).
- Two ADRs (`0001-accept-authjs-qwik.md`, `0002-promote-lestrrat-jwx-for-jwe.md`).
- Go identity layer: `domain/identity.go` (+ test), `application/identity_service.go` (+ test), `infrastructure/postgres/identity_repository.go` (+ integration test).
- Pre-existing lint cleanup required for `make lint` to be green (12 revive issues).

PR-1 deliberately does NOT touch the Auth.js frontend (PR-2) or the JWE verifier middleware (PR-3).

## Tasks completed

| Task | Locked value | Status |
| --- | --- | --- |
| T1.1 | Migration Up SQL — `identity.user`, `identity.account`, `organization.owner_user_id`, ownership to queen | ✅ |
| T1.2 | Migration Down SQL — reverse order | ✅ |
| T1.3 | Apply migration on a fresh stack (compose) — runner reports `applied_count=1` | ✅ |
| T1.4 | Rollback round-trip — down + up via runner | ✅ |
| T1.5 | ADR 0001 — accept `@auth/qwik@0.9.2` | ✅ |
| T1.6 | (committed with T1.5) | ✅ |
| T1.7 | ADR 0002 — promote `lestrrat-go/jwx/v2/jwe` | ✅ |
| T1.8 | `domain/identity.go` skeleton (types only) | ✅ |
| T1.9 | `domain/identity_test.go` — locks struct field list + AppError contract | ✅ |
| T1.10 | `application/identity_service.go` — service struct + OTel span `identity.lookup` | ✅ |
| T1.11 | `infrastructure/postgres/identity_repository.go` — pgx-backed port implementation | ✅ |
| T1.12 | Real `SELECT` with `LEFT JOIN` to identity.account | ✅ |
| T1.13 | Triangulate: miss + case-insensitive | ✅ |
| T1.14 | Lint clean (PR-1 scope) | ✅ |
| T1.15 | Spec scenario walkthrough on real compose stack | ✅ |

Tasks T1.14 and T1.15 also include pre-existing lint cleanup so `make lint` is green for PR-1's gate.

## Files changed

### New files

```text
backend/database_administrator/src/migration/sql/20260703120000_github_login.sql   (3,195 B)
backend/database_administrator/src/domain/identity.go                              (3,323 B)
backend/database_administrator/src/domain/identity_test.go                         (4,473 B)
backend/database_administrator/src/application/identity_service.go                 (3,314 B)
backend/database_administrator/src/application/identity_service_test.go            (5,743 B)
backend/database_administrator/src/infrastructure/postgres/identity_repository.go  (4,231 B)
backend/database_administrator/src/infrastructure/postgres/identity_repository_test.go  (7,160 B)
docs/adr/0001-accept-authjs-qwik.md                                                 (5,181 B)
docs/adr/0002-promote-lestrrat-jwx-for-jwe.md                                       (6,226 B)
```

### Modified files (PR-1 scope)

```text
backend/database_administrator/src/infrastructure/postgres/organization_repo.go       (rename: PostgresOrgRepo → OrgRepo per revive stutter rule)
backend/database_administrator/src/infrastructure/postgres/organization_repo_test.go  (rename: NewPostgresOrgRepo → NewOrgRepo)
backend/database_administrator/src/cmd/server/main.go                              (rename: NewPostgresOrgRepo → NewOrgRepo)
```

### Modified files (pre-existing lint cleanup, NOT PR-1 scope but required for `make lint` to be green)

```text
backend/database_administrator/src/domain/organization.go                            (added per-const godoc + per-method godoc on Code())
backend/database_administrator/src/interfaces/http/organization_handler_test.go      (removed unused stringPtr helper)
backend/database_administrator/src/migration/runner_test.go                          (extended wipeNewTables + truncateNewTables for identity.*; bumped "all 4" → "all 5" migration count assertions)
```

## Test evidence (RED → GREEN → TRIANGULATE → REFACTOR)

### Domain

| Test | Status | Evidence |
| --- | --- | --- |
| `TestIdentity_StructFields` (field list lock) | ✅ RED→GREEN | `go test -race ./src/domain/...` |
| `TestIdentityNotFoundError_AppError` (interface) | ✅ | same |
| `TestIdentityRepository_PortShape` | ✅ | same |
| `TestIdentityRepository_LookupByEmail_IsContextFirst` | ✅ | same |

### Application

| Test | Status | Evidence |
| --- | --- | --- |
| `TestIdentityService_LookupByEmail_Hit` | ✅ | covers span `identity.lookup` |
| `TestIdentityService_LookupByEmail_Miss` | ✅ | asserts `errors.As` to `*IdentityNotFoundError` |
| `TestIdentityService_LookupByEmail_RepoError` | ✅ | asserts the service does NOT wrap arbitrary repo errors |

### Infrastructure (integration)

| Test | Status | Evidence |
| --- | --- | --- |
| `TestIdentityRepository_LookupByEmail_Hit` | ✅ | against live compose postgres |
| `TestIdentityRepository_LookupByEmail_Miss` | ✅ | asserts `*IdentityNotFoundError` chain |
| `TestIdentityRepository_LookupByEmail_CaseInsensitive` | ✅ | proves CITEXT behavior |

### Migration (existing runner tests, extended for the new migration)

| Test | Status | Evidence |
| --- | --- | --- |
| `TestRunner_Up_FirstBoot` | ✅ | new migration included in the apply set |
| `TestRunner_Up_SecondBootIsNoOp` | ✅ | second boot is no-op |
| `TestRunner_Up_LexicographicOrder` | ✅ | 5 migrations apply in lexicographic order |
| `TestRunner_Up_AdvisoryLockBlocksParallelRun` | ✅ | unrelated to slice; passes |
| `TestRunner_Up_AllNewMigrationsApply` | ✅ (updated) | was 4, now 5 |
| `TestRunner_Up_LexicographicOrder_AllFourVersions` | ✅ (updated) | was 4 witsaba + 1 seeded = 5, now 5 witsaba + 1 seeded = 6 |
| `TestWitsabaFramework_AgentFirstLifecycle_EndToEnd` | ✅ (updated) | assertion updated for 5 migrations |

## TDD Cycle Evidence

| Step | Test | Behavior |
| --- | --- | --- |
| RED (T1.1) | `TestRunner_Up_FirstBoot` referenced new version | "relation \"user\" already exists" — caught by resetSchemaMigrations |
| RED (T1.8) | `domain/identity_test.go` referenced `domain.Identity` | "undefined: domain.Identity" — 6 undefined errors caught |
| RED (T1.10) | `application/identity_service_test.go` referenced `application.IdentityService` | "undefined: application.IdentityService" — 2 undefined errors caught |
| RED (T1.11) | `identity_repository_test.go` referenced `postgres.NewPostgresIdentityRepo` | "undefined: postgres.NewPostgresIdentityRepo" — caught |
| GREEN | All RED turns resolved | full suite green |
| TRIANGULATE | `TestIdentityRepository_LookupByEmail_Miss` + `_CaseInsensitive` | negative cases added |
| REFACTOR | rename `PostgresOrgRepo → OrgRepo`, add godoc comments | `make lint` from 16 issues → 0 issues |

## PR-1 PR-ready gates (all PASS)

- [x] `make test` — green (~30 integration + unit tests pass; -race clean)
- [x] `make lint` — green (0 issues)
- [x] `make build` — green (`./bin/database_administrator` produced)
- [x] `docker compose -f docker-compose.yaml build` — green (image rebuilt with new migration embedded)
- [x] Identity tests pass against a real `cachicamas_pg`.
- [x] Identity-schema spec scenarios walk through (see below).

## Identity-schema spec scenario walk-through

| Scenario | Status | Evidence |
| --- | --- | --- |
| S-IS-010 `\d identity.user` | ✅ | `id BIGSERIAL PK`, `email CITEXT NOT NULL UNIQUE`, `name`, `image_url`, `created_at`, `updated_at` |
| S-IS-011 case-insensitive email uniqueness | ✅ | two `INSERT` statements; second failed with `duplicate key value violates unique constraint "user_email_key"` |
| S-IS-020 citext extension enabled | ✅ | `infra/postgres/init/01-init.sql` already installs it |
| S-IS-030 `\d identity.account` | ✅ | explicit constraint name `account_provider_provider_account_id_key` |
| S-IS-031 duplicate (provider, account_id) rejected | ✅ | `duplicate key value violates unique constraint "account_provider_provider_account_id_key"` |
| S-IS-032 different provider_account_id → allowed | ✅ | `INSERT 0 1` |
| S-IS-033 ON DELETE CASCADE | ✅ | `DELETE FROM identity.user WHERE id = 3` removed the account row automatically |
| S-IS-040 `organization.owner_user_id` column | ✅ | `\d organization` lists `owner_user_id bigint` + `organization_owner_user_id_fkey` |
| S-IS-041 pre-existing rows keep NULL | ✅ | migration adds nullable; no backfill |
| S-IS-042 set to known user | ✅ | `UPDATE` succeeded; `owner_user_id = 6` |
| S-IS-043 set to non-existent user rejected | ✅ | `violates foreign key constraint "organization_owner_user_id_fkey"` |
| S-IS-060 filename pattern `^\d{14}_[a-z0-9_]+\.sql$` | ✅ | `20260703120000_github_login.sql` |
| S-IS-061 reversible (down then up) | ✅ | `drop + alter drop + recreate` round-tripped cleanly via `docker compose exec` |
| S-IS-070 fresh-stack apply | ✅ | `migration.up applied applied_count=1 duration_ms=28` on first boot |
| S-IS-080 ownership = queen | ✅ | `pg_get_userbyid(relowner)` returns `queen` for both new tables |

## Deviations from design

None. The plan in `design.md` §4 file layout was followed exactly. One implementation detail the
DESIGN didn't pin (and we settled on here): the `application/identity_service.go` exposes
`NewIdentityService(repo, logger, tracer)` instead of a default constructor — matches the project
pattern of test-only initialization.

## Next action

PR-2 (`feat/cachicamas-github-login-pr2-frontend-auth`): frontend Auth.js integration + Playwright
e2e suite + `mocks/github-oauth` compose service.

PR-3 (`feat/cachicamas-github-login-pr3-backend-verifier`): Go JWE verifier + protected demo endpoint.

After PR-3 merges, `sdd-verify` collects the walkthrough for `verify-report.md`; `sdd-sync` adds the
identity and middleware specs to `openspec/specs/`; `sdd-archive` moves the folder under
`openspec/changes/archive/2026-07-03-cachicamas-github-login/`.
