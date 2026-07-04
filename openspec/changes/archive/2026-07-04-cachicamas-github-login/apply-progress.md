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

---

## PR-1 merged (post-merge housekeeping, 2026-07-04)

### Git state

| Item | Value |
| --- | --- |
| Merge commit on `origin/main` | `69429c4` "feat(identity): github_login schema + identity domain + ADRs (#18)" |
| Merge method | Squash via PR button (visible from "Closed" comment) |
| Local worktree | `cachicamas.post-pr1` on `post-pr1-validation` branch (per user rule: never edit `main` directly) |
| PR-1 branch on origin | `feat/cachicamas-github-login-pr1-schema-identity` (kept for traceability; safe to delete after PR-3 ships) |

### Post-merge gate validation (all green)

- `cd backend/database_administrator && make lint` → `0 issues`
- `cd backend/database_administrator && make build` → `bin/database_administrator` produced
- `INTEGRATION=1 make test` → all packages PASS, including:
  - `domain`: `TestIdentity_StructFields`, `TestIdentityNotFoundError_AppError`, `TestIdentityRepository_PortShape`, `TestIdentityRepository_LookupByEmail_IsContextFirst`
  - `application`: `TestIdentityService_LookupByEmail_{Hit,Miss,RepoError}`
  - `infrastructure/postgres`: `TestIdentityRepository_LookupByEmail_{Hit,Miss,CaseInsensitive}`
  - `migration`: `TestRunner_Up_AllNewMigrationsApply` (now 5 migrations), `TestWitsabaFramework_AgentFirstLifecycle_EndToEnd`

### Spec promotion

- **identity-schema**: PR-1 slice is now live. The delta `openspec/changes/cachicamas-github-login/specs/identity-schema/spec.md` was **promoted to canonical** at `openspec/specs/identity-schema/spec.md`. The delta is preserved for the change history.
- **frontend-auth** (delta): NOT yet promoted — PR-2 work is pending.
- **backend-auth-middleware** (delta): NOT yet promoted — PR-3 work is pending.

### Identity-schema canonical spec — quick reference

- Capability: identity-schema
- Domain: identity
- 15 scenarios (S-IS-010 → S-IS-080) under `openspec/specs/identity-schema/spec.md`
- Tables introduced: `identity.user`, `identity.account`, plus `organization.owner_user_id` (nullable FK)
- Constraints: CITEXT UNIQUE on email; UNIQUE(provider, provider_account_id) on account; ON DELETE CASCADE on account.user_id (forward note: see §R1 of the 4R inline review)
- Migration: `backend/database_administrator/src/migration/sql/20260703120000_github_login.sql` (Up + Down, reversible)

### Next action

PR-2 (`feat/cachicamas-github-login-pr2-frontend-auth`) is queued in its own worktree (per the
new worktree rule). Implementation starts in the next focused turn. PR-1's branch is preserved
on origin for reference; the post-pr1-validation worktree holds the housekeeping commit and is
the staging area for the spec-promotion merge back to main.

---

## PR-2 work-in-progress (2026-07-04)

### Slice scope shipped

- `routes/plugin@auth.ts` + 5-test spec (QwikAuth$ wiring, GitHub
  provider, trustHost, AUTH_SECRET plumbing).
- `lib/sign-in-callback.ts` + 6-test spec (auto-link-on-email-match
  UPSERT logic; deny on no-email).
- `components/sign-in-button/` + 5-test spec (CTA rendering).
- `components/profile-view/` + 7-test spec (pure presentational
  ProfileView, testable in vitest).
- `routes/profile/index.tsx` (thin route wrapper; calls useSession).
- `routes/index.tsx` SignInButton added to landing hero CTA.
- `vite.config.ts` optimizeDeps + tightened duplicate-check.
- `package.json` exact-pinned `@auth/qwik@0.9.2`, `@auth/core@0.41.2`,
  `@panva/hkdf@1.2.1`, `postgres@3.4.5`.
- `.env.example` +5 AUTH_* keys (with fail-fast `:?` operator in compose).
- `docker-compose.yaml` frontend env block (5 AUTH_* vars) +
  database_administrator AUTH_SECRET.
- `Dockerfile` COPY additions for @auth/qwik, @auth/core, @panva,
  postgres, jose, oauth4webapi, set-cookie-parser.
- `scripts/mocks-github-oauth/` Node service (Dockerfile + server.mjs
  + package.json).
- `e2e/sign-in-landing.spec.ts` (visible CTA check).
- `e2e/github-sign-in.spec.ts` (full OAuth roundtrip; runs in mocks
  mode only — `test.skip` when AUTH_GITHUB_BASE_URL is unset).
- `README.md` Authentication section (production + tests setup).

### Gates green

- `cd frontend && pnpm test:ci` → **119 unit tests pass** (was 95
  before PR-2).
- `pnpm build.types` → no TS errors.
- `pnpm lint` → 0 issues.
- `pnpm fmt.check` → clean.
- `docker compose config` → AUTH_* env vars flow to frontend and
  database_administrator; mocks-github-oauth service resolves.
- mocks service smoke test → /healthz, /user, /user/emails all
  return expected shapes.

### Open follow-ups (deferred to next commit / turn)

- `e2e/sign-in-cookie-attrs.spec.ts` (HttpOnly / SameSite assertions).
- `e2e/sign-in-denied.spec.ts` (`error=access_denied` path).
- `e2e/sign-out.spec.ts` (cookie cleared; /profile → /auth/signin).
- `tests/mocks-github-oauth/` unit tests for the simulator endpoints.
- Live Playwright run against the dockerized stack to confirm
  `github-sign-in.spec.ts` actually drives the full OAuth roundtrip
  end-to-end (the spec is wired but not yet exercised against a
  running compose stack in this turn).

---

## PR-2 merged (post-merge housekeeping, 2026-07-04)

### Git state

| Item | Value |
| --- | --- |
| Merge commit on `origin/main` | `7ea621a` "feat(frontend): github_login PR-2 frontend Auth.js UX + mocks + e2e (#20)" |
| Merge method | Squash (PR button on GitHub) |
| Local worktree | `cachicamas.post-pr2` on `post-pr2-canonical` branch (per user rule: never edit `main` directly) |
| PR-2 branch on origin | `feat/cachicamas-github-login-pr2-frontend-auth` (deleted by --delete-branch on merge) |

### Spec promotion

- **frontend-auth**: PR-2 slice is now live. The delta `openspec/changes/cachicamas-github-login/specs/frontend-auth/spec.md` was **promoted to canonical** at `openspec/specs/frontend-auth/spec.md`. The delta is preserved for the change history.
- **backend-auth-middleware** (delta): NOT yet promoted — PR-3 work is pending.
- **identity-schema** (delta): was promoted to canonical after PR-1 merged (commit `e67dac4`). The delta remains under `openspec/changes/.../specs/identity-schema/` for change history.

### Canonical specs now under `openspec/specs/`

- `db-migrations/`
- `frontend-compose-and-cors/`
- `frontend-e2e-and-client-data/`
- `frontend-runtime/`
- `identity-schema/` (PR-1 promoted)
- `frontend-auth/` (PR-2 promoted this turn)

### Frontend-auth canonical spec — quick reference

- Capability: frontend-auth
- Domain: frontend
- 18 scenarios (S-FA-001..S-FA-058) under `openspec/specs/frontend-auth/spec.md`
- Components introduced: `SignInButton`, `ProfileView`, `routes/plugin@auth.ts`, `routes/profile/`, `lib/sign-in-callback.ts`
- Env bindings: `AUTH_GITHUB_ID`, `AUTH_GITHUB_SECRET`, `AUTH_SECRET`, `AUTH_TRUST_HOST`, `AUTH_URL`, `AUTH_GITHUB_BASE_URL` (test-only override)
- Cookie name: `authjs.session-token` (dev) / `__Secure-authjs.session-token` (prod, when AUTH_URL is HTTPS)
- Strategy: stateless JWE cookie; auto-link-on-email-match on the events.signIn callback
- Mocks simulator: `scripts/mocks-github-oauth/` (compose service `mocks-github-oauth`)

### Next action

PR-3 (`feat/cachicamas-github-login-pr3-backend-verifier`) is queued in its own worktree. Implementation starts in the next focused turn. The Go-side JWE verifier middleware uses `lestrrat-go/jwx/v2` and shares `AUTH_SECRET` with the frontend (per ADR 0002 byte-level envelope contract).

---

## PR-3 work-in-progress (2026-07-04)

### Slice scope shipped

- `interfaces/http/auth_middleware.go` — `IdentityFromCookie(cfg) echo.MiddlewareFunc` Echo middleware.
  Reads the configured JWE cookie, decrypts with `lestrrat-go/jwx/v2`
  using the locked envelope (alg=dir, enc=A256CBC-HS512, HKDF-SHA256
  over AUTH_SECRET with salt=cookieName, length=64). Resolves the
  identity.user row via `IdentityRepository.LookupByEmail`, populates
  `c.Set("identity", *Identity)`, and emits a 401 envelope on any
  failure. The HKDF derivation is verified against `@auth/core@0.41.2`
  source (`src/jwt.ts`) byte-for-byte.

- `interfaces/http/auth_middleware_test.go` — 8 unit tests + 4 sub-tests
  covering R-BAM-001..R-BAM-031 (S-BAM-010..S-BAM-110):
  - Valid cookie populates identity
  - Tampered cookie returns 401
  - Missing cookie returns 401
  - Decryption shape (S-BAM-020)
  - No Set-Cookie header on any code path (S-BAM-030)
  - OTel span + slog line + PII-safe email_hash (S-BAM-070..072)
  - Missing AUTH_SECRET panics at startup (S-BAM-080)
  - Too-short AUTH_SECRET panics at startup (S-BAM-081)

- `interfaces/http/whoami_handler.go` — demo `/api/v1/protected/whoami`
  endpoint behind `IdentityFromCookie` middleware. Returns the
  resolved identity's id/email/name/provider as JSON. Exposes
  `IsLikelyHTTPS(origin)` helper for `main.go` cookie-name selection.

- `interfaces/http/testdata/authjs_session_token.jwe` — committed
  fixture produced by `scripts/regenerate_authjs_testdata.sh`.
  The fixture is the byte-level cross-tooling evidence for the
  JWE envelope contract (design §3).

- `scripts/regenerate_authjs_testdata.sh` — regenerates the fixture
  using `@auth/core`'s encoder with a known AUTH_SECRET + known
  payload, then writes the JWE to testdata/.

- `cmd/server/main.go` — wires AUTH_SECRET loading (with
  fail-fast on missing/short), cookie-name selection (dev:
  `authjs.session-token`, prod: `__Secure-authjs.session-token`),
  `postgres.NewIdentityRepo` construction, and
  `RegisterProtectedWhoAmIRoute` mounting AFTER CORS.

- `go.mod` — `github.com/lestrrat-go/jwx/v2 v2.1.7`,
  `golang.org/x/crypto v0.53.0` (for HKDF).

### Gates green

- `cd backend/database_administrator && make lint` → 0 issues.
- `cd backend/database_administrator && make build` → 20MB binary.
- `go test -race ./...` → all packages PASS (8 new auth tests,
  including 4 sub-tests under TestIdentityFromCookie_NeverEmitsSetCookie).
- `docker build backend/database_administrator` → succeeded.
- Fixture round-trip: regenerated via `@auth/core@0.41.2` and read
  by the Go verifier (alg=dir, enc=A256CBC-HS512) — plaintext
  decoded successfully and identity lookup succeeded.

### Open follow-ups

- Integration test against live compose (currently the migration +
  identity integration tests need a running postgres; deferred to
  a follow-up turn that brings up compose for the e2e verification).
- `tests/mocks-github-oauth/` unit tests (deferred from PR-2).
- `e2e/sign-in-cookie-attrs.spec.ts`, `sign-in-denied.spec.ts`,
  `sign-out.spec.ts` (deferred from PR-2).

---

## PR-3 merged (post-merge housekeeping, 2026-07-04)

### Git state

| Item | Value |
| --- | --- |
| Merge commit on `origin/main` | `c4b89d4` "feat(backend): github_login PR-3 Go JWE verifier middleware (#22)" |
| Merge method | Squash (PR button on GitHub) |
| Local worktree | `cachicamas.post-pr3` on `post-pr3-archive` branch (per user rule: never edit `main` directly) |
| PR-3 branch on origin | `feat/cachicamas-github-login-pr3-backend-verifier` (deleted by --delete-branch on merge) |

### Spec promotion + archive

- **backend-auth-middleware**: PR-3 slice is now live. The delta
  `openspec/changes/cachicamas-github-login/specs/backend-auth-middleware/spec.md`
  was **promoted to canonical** at
  `openspec/specs/backend-auth-middleware/spec.md`. The delta is
  preserved for the change history (under the archived folder).

- **sdd-archive**: the entire change folder
  `openspec/changes/cachicamas-github-login/` is moved to
  `openspec/changes/archive/2026-07-04-cachicamas-github-login/`,
  following the convention of
  `openspec/changes/archive/2026-07-03-cachicamas-frontend-dockerize/`.
  A `verify-report.md` is added that summarizes the implementation
  walkthrough across the 3 chained PRs.

### Canonical specs now under `openspec/specs/`

- `db-migrations/`
- `frontend-compose-and-cors/`
- `frontend-e2e-and-client-data/`
- `frontend-runtime/`
- `identity-schema/` (PR-1 promoted)
- `frontend-auth/` (PR-2 promoted)
- `backend-auth-middleware/` (PR-3 promoted this turn)

### Backend-auth-middleware canonical spec — quick reference

- Capability: backend-auth-middleware
- Domain: backend-auth
- 17 scenarios (S-BAM-010..S-BAM-110) under `openspec/specs/backend-auth-middleware/spec.md`
- Middleware: `IdentityFromCookie(cfg)` in `backend/database_administrator/src/interfaces/http/auth_middleware.go`
- Demo endpoint: `/api/v1/protected/whoami` (returns identity as JSON)
- Envelope contract (locked by ADR 0002): alg=dir, enc=A256CBC-HS512, HKDF-SHA256 over AUTH_SECRET with salt=cookieName, length=64. Verified byte-for-byte against `@auth/core@0.41.2/src/jwt.ts`.
- Cross-tooling evidence: `backend/database_administrator/src/interfaces/http/testdata/authjs_session_token.jwe` (committed fixture produced by `@auth/core`'s encoder).

### Change shipped: cachicamas-github-login

All 3 chained PRs merged into `main`:

| PR | Title | Commit | LOC |
| --- | --- | --- | --- |
| #18 | feat(identity): github_login schema + identity domain + ADRs | `69429c4` | +1,316 |
| #20 | feat(frontend): github_login PR-2 frontend Auth.js UX | `7ea621a` | +1,968 |
| #21 | chore(openspec): promote frontend-auth to canonical | `1f99796` | +343 |
| #22 | feat(backend): github_login PR-3 Go JWE verifier | `c4b89d4` | +1,086 |

### Forward notes (deferred to follow-up PRs)

- 3 PR-2 e2e specs (`sign-in-cookie-attrs`, `sign-in-denied`,
  `sign-out`).
- Live-compose integration test for migration + identity_repo end-to-end.
- `LookupByProviderAccountID` method on the IdentityRepository port
  (for multi-provider support).
