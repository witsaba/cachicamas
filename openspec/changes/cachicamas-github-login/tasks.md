# Tasks: cachicamas-github-login

> **Change**: `cachicamas-github-login`
> **Status**: task-ized
> **Depends on**: `proposal.md`, `specs/{frontend-auth,backend-auth-middleware,identity-schema}/spec.md`, `design.md`
> **Forecast**: ~1,400 changed lines, **3 chained PRs** recommended. See Review Workload Forecast below.

---

## Review Workload Forecast

| Field | Value |
| ------- | ------- |
| Estimated changed lines | **~1,400** (rough: 250 backend / 150 migration / 200 frontend auth core / 200 e2e specs / 100 docs+ADRs / 50 README + config) |
| 400-line budget risk | **High** |
| Chained PRs recommended | **Yes** |
| Suggested split | PR-1 (Schema + identity domain + ADRs) → PR-2 (Frontend Auth.js + e2e) → PR-3 (Backend JWE verifier + protected endpoint) |
| Delivery strategy | **ask-on-risk** — pause to confirm the chain strategy before `sdd-apply` |
| Chain strategy | **feature-branch-chain** — three PRs, each from a feature branch into `main` sequentially |

```text
Decision needed before apply: Yes
Chained PRs recommended: Yes
Chain strategy: feature-branch-chain
400-line budget risk: High
```

### Why three PRs

- **PR-1 alone is ~500 lines** (mostly SQL + Go domain + integration tests + two ADRs), already at the budget edge. Splitting it would force the ADRs into a separate "docs-only" PR that doesn't compile in isolation.
- **PR-2 alone is ~500 lines** (Auth.js + sign-in UI + vitest + Playwright specs).
- **PR-3 alone is ~400 lines** (auth_middleware + tests + fixture + demo endpoint).

Each PR is independently reviewable, runs the existing test gate (`make test`, `pnpm test:ci`, `pnpm test:e2e` for the relevant subset), and can be merged without breaking `main`:

- PR-1 keeps `identity.user` empty; the API only adds nullable columns and tables.
- PR-2's Auth.js wiring persists users but has no verifier on the Go side — for the
  time PR-2 is open, the `/api/*` reverse proxy behaves as before.
- PR-3 introduces the verifier. Until PR-3 lands, PR-2's cookies are still
  readable by the Qwik Node server but ignored by the Go service.

Merge order: PR-1 → PR-2 → PR-3.

---

## Dependency graph

```text
PR-1 (Schema + identity domain + ADRs)
├── T1.1  migrate: up (sql)
├── T1.2  migrate: down (sql)
├── T1.3  docs/adr/0001
├── T1.4  docs/adr/0002
├── T1.5  domain/identity.go + test
├── T1.6  application/identity_service.go + test
└── T1.7  infrastructure/postgres/identity_repository.go + integration test

PR-2 (Frontend Auth.js integration + e2e)
├── T2.1  frontend/package.json: +@auth/qwik, +@panva/hkdf
├── T2.2  frontend/vite.config.ts: +optimizeDeps.include
├── T2.3  frontend/src/routes/plugin@auth.ts
├── T2.4  frontend/src/lib/sign-in-callback.ts
├── T2.5  frontend/src/components/sign-in-button/*
├── T2.6  frontend/src/routes/profile/index.tsx + index.spec.tsx
├── T2.7  frontend/src/routes/index.tsx: +<SignInButton/>
├── T2.8  .env.example: +5 AUTH_* keys (frontend block)
├── T2.9  docker-compose.yaml: +5 env vars (frontend service)
├── T2.10 frontend/Dockerfile: +COPY --from=builder for @auth/qwik + @auth/core
├── T2.11 frontend/e2e/specs (5 files)
└── T2.12 frontend/README.md: +"Auth.js / GitHub login" section

PR-3 (Backend JWE verifier + protected endpoint)
├── T3.1  backend/go.mod: +lestrrat-go/jwx/v2, +golang.org/x/crypto
├── T3.2  backend/src/interfaces/http/auth_middleware.go + test
├── T3.3  backend/scripts/regenerate_authjs_testdata.sh
├── T3.4  backend/src/interfaces/http/testdata/authjs_session_token.jwe (fixture)
├── T3.5  backend/src/cmd/server/main.go: +load AUTH_SECRET, +wire middleware
├── T3.6  backend/README.md: +callback URL table
└── T3.7  compose stack end-to-end smoke (PR-3 acceptance)
```

---

## PR-1: Schema + identity domain + ADRs

> ~500 lines, single PR. Branch `feat/cachicamas-github-login-pr1-schema-identity`. Target: `main`.

### T1.1 [RED] Write failing migration Up

- File: `backend/database_administrator/src/migration/sql/20260703120000_github_login.sql`
- Action: create the `-- +goose Up` section with `CREATE TABLE identity.user`, `CREATE TABLE identity.account`, `ALTER TABLE organization ADD COLUMN owner_user_id`, and `ALTER TABLE ... OWNER TO queen` for each new table.
- Verify RED: `cd backend/database_administrator && go test ./src/migration/... -run TestRunner_Applies20260703120000` fails because the file does not exist.

### T1.2 [RED] Write failing migration Down

- File: same.
- Action: create the `-- +goose Down` section that drops the FK column and the two tables in the opposite order of creation.
- Verify RED: same test now asserts a "missing file" error; it still fails.

### T1.3 [GREEN] Apply the migration on a fresh stack

- Action: bring up the dev compose (`docker compose -f docker-compose.yaml up -d --build`); confirm `psql -U queen -d cachicamas_pg -c "\dt identity.*"` lists both tables; confirm `\d organization` shows `owner_user_id`.
- Verify GREEN: `docker compose exec postgres psql -U queen -d cachicamas_pg -c "\d identity.user"` returns the locked column set from R-IS-001.

### T1.4 [GREEN] Verify migration rollback

- Action: `goose -dir backend/database_administrator/src/migration/sql postgres "$DATABASE_URL" down` to roll back the migration; re-up to confirm idempotency.
- Verify GREEN: `S-IS-061` (rolling down then up leaves no stale rows in `public.schema_migrations`).

### T1.5 [RED] Add ADR 0001 — accept @auth/qwik

- File: `docs/adr/0001-accept-authjs-qwik.md`
- Verify RED: `git grep "0001-accept-authjs-qwik"` returns no results.

### T1.6 [GREEN] Commit ADR 0001

- Action: commit the ADR file from T1.5.
- Verify GREEN: `git grep -l "0001-accept-authjs-qwik" docs/adr/` returns the new file. No code change required.

### T1.7 Repeat T1.5/T1.6 for ADR 0002 (`docs/adr/0002-promote-lestrrat-jwx-for-jwe.md`)

### T1.8 [RED] `domain/identity.go` skeleton

- File: `backend/database_administrator/src/domain/identity.go`
- Action: declare `Identity`, `IdentityRepository`, `ErrIdentityNotFound` plus the field list from spec R-BAM-010.
- Verify RED: the file compiles; `go test ./src/domain/...` fails because tests reference `Identity.Email` etc. that aren't asserted yet.

### T1.9 [GREEN] `domain/identity_test.go` struct assertions

- File: `backend/database_administrator/src/domain/identity_test.go`
- Action: assert the locked field list and the `IdentityRepository.LookupByEmail` signature.
- Verify GREEN: `go test ./src/domain/... -run TestIdentityStructFields` passes.

### T1.10 [GREEN] `application/identity_service.go` thin wrapper

- File: `backend/database_administrator/src/application/identity_service.go`
- Action: expose a `type IdentityService struct { repo IdentityRepository }` with `LookupByEmail(ctx, email) (*Identity, error)`.
- Test: `application/identity_service_test.go` uses a fake repo to assert pass-through and error mapping.
- Verify GREEN: `go test ./src/application/... -run TestIdentityService_LookupByEmail` passes.

### T1.11 [RED] `infrastructure/postgres/identity_repository.go` skeleton

- File: `backend/database_administrator/src/infrastructure/postgres/identity_repository.go`
- Action: declare `type IdentityRepository struct { db *pgxpool.Pool }` and a `LookupByEmail` implementation. The implementation will initially `panic("not implemented")`.
- Verify RED: `go test ./src/infrastructure/postgres/... -run TestIdentityRepository_LookupByEmail_Hit` fails on the panic.

### T1.12 [GREEN] Real SELECT against identity.user

- Action: implement `SELECT id, email, name, image_url FROM identity.user WHERE lower(email) = lower($1)`.
- Verify GREEN: integration test with a real `cachicamas_pg` shows the row.

### T1.13 [TRIANGULATE] Negative test cases

- File: `infrastructure/postgres/identity_repository_test.go`
- Add: `TestIdentityRepository_LookupByEmail_Miss` (returns `ErrIdentityNotFound`), `TestIdentityRepository_LookupByEmail_UpperCase` (case-insensitive lookup works).
- Verify: `go test -race -v ./src/infrastructure/postgres/... -run TestIdentityRepository` all pass.

### T1.14 [REFACTOR] Lint and review

- Run `cd backend/database_administrator && make fmt && make lint && make test`.
- Target: zero new issues; no style drift in the new files.

### T1.15 [VERIFY] S-IS-010..S-IS-080 walkthrough

- Walk through every spec scenario in `identity-schema/spec.md`:
  - S-IS-010 (`\d identity.user`)
  - S-IS-020 (citext extension)
  - S-IS-030 (`\d identity.account`)
  - S-IS-031..S-IS-033 (uniqueness, multi-account, cascade delete)
  - S-IS-040..S-IS-043 (organization.owner_user_id FK behaviors)
  - S-IS-060 (filename pattern)
  - S-IS-061 (down then up)
  - S-IS-070 (fresh-stack apply)
  - S-IS-080 (ownership = queen)
- Record results in `openspec/changes/cachicamas-github-login/verify-report.md` (filled in `sdd-verify`).

### PR-1 PR-ready gates

- `cd backend/database_administrator && make test` — green.
- `cd backend/database_administrator && make lint` — green.
- `cd backend/database_administrator && make build` — green.
- `docker compose -f docker-compose.yaml build` — green.
- Identity tests pass against a real `cachicamas_pg`.
- PR title: `feat(identity): github_login schema + identity domain + ADRs #N`.
- PR body uses the `branch-pr` skill template (linked from parent).

---

## PR-2: Frontend Auth.js integration + e2e

> ~500 lines, single PR. Branch `feat/cachicamas-github-login-pr2-frontend-auth`. Target: `main`. **Depends on PR-1 merged**.

### T2.1 [CONFIG] Add `@auth/qwik` and `@panva/hkdf`

- File: `frontend/package.json`
- Action: add dependencies at pinned exact versions (`"@auth/qwik": "0.9.2"`, `"@panva/hkdf": "1.x.y"` resolved via `npm view`).
- Verify: `pnpm install --frozen-lockfile` updates the lockfile; `pnpm test:ci` still green (no behavior change yet).

### T2.2 [CONFIG] Optimize Vite for @auth/qwik

- File: `frontend/vite.config.ts`
- Action: add `"@auth/qwik"`, `"@auth/core"`, `"@panva/hkdf"` to `optimizeDeps.include`.
- Verify: `pnpm dev` starts; no Vite warnings about the missing deps.

### T2.3 [RED] `routes/plugin@auth.ts` exports

- File: `frontend/src/routes/plugin@auth.ts`
- Action: invoke `QwikAuth$(({ env }) => ({ providers: [GitHub], trustHost: true }))`; export `{ onRequest, useSession, useSignIn, useSignOut }`.
- Verify RED: `pnpm test:ci` runs `plugin@auth.test.ts` (asserts the four exports exist).

### T2.4 [GREEN] `plugin@auth.ts` test

- File: `frontend/src/routes/plugin@auth.test.ts`
- Action: assert the four exports exist and `QwikAuth$` was called with `GitHub` as the only provider.
- Verify GREEN: `pnpm test:ci -t "plugin@auth"` passes.

### T2.5 [RED] `lib/sign-in-callback.ts` skeleton

- File: `frontend/src/lib/sign-in-callback.ts`
- Action: declare the `signIn` callback shape (params + return) but throw "not implemented".
- Verify RED: `pnpm test:ci -t "sign-in-callback"` panics.

### T2.6 [GREEN] Implement the auto-link-on-email-match logic

- Action: in `lib/sign-in-callback.ts`, write the callback that:
  1. `SELECT id, email, name, image_url FROM identity.user WHERE lower(email) = lower($1)` via `postgres` client (passed in by Auth.js's adapter); create on miss.
  2. `INSERT INTO identity.account ...` (handle `ON CONFLICT (provider, provider_account_id) DO NOTHING`).
- Verify GREEN: integration spec against `cachicamas_pg` shows the rows.

### T2.7 [TRIANGULATE] Negative case: same email via a different GitHub account_id

- Action: insert a `identity.user` row with email `linktest@example.com`; run the callback with `account.providerAccountId = 'NEW_ID'`; assert that the existing `identity.user` row is reused and a new `identity.account` row is created.
- Verify: spec passes.

### T2.8 [RED] `components/sign-in-button/sign-in-button.tsx` skeleton

- File: `frontend/src/components/sign-in-button/sign-in-button.tsx`
- Action: render `<Form action={signIn}>` with hidden `providerId` and visible text.
- Verify RED: `vitest run components/sign-in-button/sign-in-button.spec.tsx` — required text not present.

### T2.9 [GREEN] Sign-in button renders + click triggers action

- Verify: vitest asserts the text is present and a `<form action={signIn}>` is in the DOM.

### T2.10 [RED] `routes/profile/index.tsx` skeleton

- File: `frontend/src/routes/profile/index.tsx`
- Action: render `useSession().value?.user?.name`.
- Verify RED: `pnpm test:ci -t "profile"` asserts the name is absent.

### T2.11 [GREEN] `routes/profile/index.tsx` server-known profile

- Action: add a `routeLoader$` that hits `GET /api/whoami` (dev-only demo endpoint from PR-3 stub or mock) and uses the returned JSON.
- Wait — PR-3 hasn't landed yet. **Adaptation for PR-2 in isolation:** the profile route reads from `useSession()` only (no extra roundtrip). PR-3 will refactor this to read from the Go demo endpoint.
- Verify GREEN: vitest spec uses an injected `useSession` mock and asserts the rendered name.

### T2.12 [RED] `routes/index.tsx` landing CTA

- File: `frontend/src/routes/index.tsx`
- Action: import `<SignInButton/>` and place it in the hero CTA section.
- Verify RED: vitest asserts the landing page does NOT yet contain the sign-in button.

### T2.13 [GREEN] Landing renders the button

- Verify GREEN: vitest assert passes.

### T2.14 [CONFIG] `.env.example` — add the 5 AUTH_* keys

- File: `.env.example`
- Action: append the block from design §5.1.

### T2.15 [CONFIG] `docker-compose.yaml` — frontend env

- File: `docker-compose.yaml`
- Action: append the 5 `AUTH_*` env vars to the `cachicamas-frontend` service.

### T2.16 [INFRA] `Dockerfile` audit + COPY additions

- File: `frontend/Dockerfile`
- Action: run `pnpm build` once; inspect the produced `frontend/server/`; identify any `@auth/*` or `@panva/hkdf` modules that were dynamic-imported and not bundled by Vite; add `COPY --from=builder /app/node_modules/<pkg> ./node_modules/<pkg>` lines (mirroring the existing `undici` pattern).
- Verify: `docker compose -f docker-compose.yaml build frontend` succeeds; container healthcheck OK.

### T2.17 [RED] Playwright `sign-in-landing.spec.ts`

- File: `frontend/e2e/sign-in-landing.spec.ts`
- Action: render `/` while signed out and assert the button is visible.
- Verify RED: dev server up, `pnpm test:e2e -- --grep "sign-in landing"` fails because the button isn't there.

### T2.18 [GREEN] Button is visible

- Verify: same spec passes (T2.13 already added the button).

### T2.19 [RED] Playwright `github-sign-in.spec.ts`

- File: `frontend/e2e/github-sign-in.spec.ts`
- Action: complete the full OAuth roundtrip (using the GitHub test account fixture) → land on `/profile` with the right name visible + identity rows in Postgres.
- **For CI determinism**, this spec points at the `mocks/github-oauth` compose service (added in T2.20).

### T2.20 [TEST-INFRA] GitHub OAuth simulator compose service

- File: `docker-compose.yaml` + `scripts/mocks-github-oauth/`
- Action: add a tiny Node-side mock that exposes `https://mocks-github-oauth/...` URLs with shape compatible to `github.com/login/oauth/...`. The Qwik Node server's `AUTH_GITHUB_ID`/`AUTH_GITHUB_SECRET`/`AUTH_URL` env vars point at it.
- Verify: the spec in T2.19 passes without requiring a live GitHub account.

### T2.21..T2.23 [RED/GREEN/TRIANGULATE] Three more Playwright specs

- `frontend/e2e/sign-in-cookie-attrs.spec.ts` (cookie HttpOnly/SameSite/Path/prefix).
- `frontend/e2e/sign-in-denied.spec.ts` (`error=access_denied` path).
- `frontend/e2e/sign-out.spec.ts` (cookie cleared; `/profile` → `/auth/signin`).

### T2.24 [DOCS] `frontend/README.md` — Auth.js section

- Action: add an "Authentication" section listing:
  - How to create the GitHub OAuth App and the exact callback URLs per env.
  - How to generate `AUTH_SECRET` (`openssl rand -base64 32`).
  - List of new env vars.
  - How to run the Playwright suite locally.

### PR-2 PR-ready gates

- `pnpm test:ci` — green.
- `pnpm test:e2e` — green (against the dev compose + mocks service).
- `pnpm build` — green (used by the Dockerfile).
- `docker compose -f docker-compose.yaml build frontend` — green.
- `docker compose up -d` → sign-in flow works end-to-end in dev.
- Existing create-org Playwright spec still green.
- PR title: `feat(frontend): github login via @auth/qwik + e2e #N`.

---

## PR-3: Backend JWE verifier + protected endpoint

> ~400 lines, single PR. Branch `feat/cachicamas-github-login-pr3-backend-verifier`. Target: `main`. **Depends on PR-2 merged**.

### T3.1 [CONFIG] `go.mod`: + `lestrrat-go/jwx/v2`, + `golang.org/x/crypto`

- File: `backend/database_administrator/go.mod`
- Action: add the two deps; pin to current minor.
- Verify: `go mod tidy` succeeds; `make build` still green.

### T3.2 [RED] `interfaces/http/auth_middleware.go` skeleton

- File: `backend/database_administrator/src/interfaces/http/auth_middleware.go`
- Action: declare `IdentityMiddlewareConfig`, the factory function, and the middleware function body that returns 501 Not Implemented.
- Verify RED: `go test ./src/interfaces/http/... -run TestIdentityFromCookie_ValidCookie_PopulatesIdentity` fails with 501.

### T3.3 [TDD] Define the fixture JWE

- File: `backend/database_administrator/scripts/regenerate_authjs_testdata.sh`
- Action: a small Node script that uses `@auth/core/jwt`'s `encode` to produce a JWE for a known payload (`{email: "test@example.com", name: "Test User"}`) using the fixture `AUTH_SECRET`.
- Run: outputs `testdata/authjs_session_token.jwe` and a `testdata/authjs_session_token.json` (plaintext).

### T3.4 [TDD] Commit the committed fixture

- File: `backend/database_administrator/src/interfaces/http/testdata/authjs_session_token.jwe` (and the `.json` plaintext)
- Verify: committed in git; `git log -p -- testdata/` shows the regenerated file.

### T3.5 [GREEN] Implement the middleware — happy path

- Action: read `authjs.session-token` cookie → derive 64-byte key via HKDF-SHA256 (per design §3) → `jwe.Decrypt(..., jwa.DIRECT, key)` → decode the JSON → resolve user via `IdentityRepository.LookupByEmail` → `c.Set("identity", &identity)` → `next(c)`.
- Verify GREEN: `TestIdentityFromCookie_ValidCookie_PopulatesIdentity` passes.

### T3.6 [TDD] Negative path tests (TRIANGULATE)

- `TestIdentityFromCookie_TamperedCookie_Returns401` — fixture's AUTH_SECRET swapped.
- `TestIdentityFromCookie_MissingCookie_Returns401`.
- `TestIdentityFromCookie_UnknownEmail_Returns401` — cookie decrypts but email has no row.
- `TestIdentityFromCookie_DecryptionShape` — asserts the JSON shape `{sub, email, name, picture, iat, exp, jti}`.
- `TestIdentityFromCookie_NoSetCookieOnSuccess` — middleware does not set Set-Cookie.
- `TestIdentityFromCookie_LogLineOmitsRawEmail` — span + log line use only `auth.email_hash`.

### T3.7 [REFACTOR] Lint and review

- `cd backend/database_administrator && make fmt && make lint && make test -race`.
- Target: zero new issues.

### T3.8 [CONFIG] `cmd/server/main.go` — load `AUTH_SECRET`, validate, wire middleware

- File: `backend/database_administrator/src/cmd/server/main.go`
- Action:
  - Read `AUTH_SECRET` env; fail-fast (exit 1 + slog.Error) if empty or shorter than 32 bytes (`TestMain_RejectsEmptyAuthSecret`, `TestMain_RejectsShortAuthSecret`).
  - Construct `IdentityRepository` from the pgx pool; construct `IdentityFromCookie`; mount on `/api/v1/protected/*` route group ONLY when `SERVICE_ENV=development`.
  - Add a `/api/v1/protected/whoami` stub that returns `c.Get("identity")` for dev smoke.

### T3.9 [DOCS] `backend/database_administrator/README.md` — auth section

- Action: add a "Auth: JWE cookie verification" section listing:
  - Why we share `AUTH_SECRET` with the frontend.
  - The exact HKDF parameters (alg=sha256, salt=cookieName, info literal, len=64).
  - Callback URL table per env (which GitHub OAuth App callback URL).
  - Operator docs for rotating `AUTH_SECRET` (deferred change for v1.0; out of scope for this PR).

### T3.10 [VERIFY] Full spec scenario walk-through

- Walk through `backend-auth-middleware/spec.md` end-to-end against a real Qwik frontend running in compose. Re-run with a tampered cookie → 401. Re-run with no cookie → 401. Confirm OTel spans emitted. Confirm CORS unchanged.
- Record results in `verify-report.md`.

### T3.11 [VERIFY] identity-schema scenarios still green

- Re-run the PR-1 scenario walk-through against the new code path. The repository changes in PR-3 should NOT alter the migration outcomes; this is a regression guard.

### PR-3 PR-ready gates

- `cd backend/database_administrator && make test` — green (race + verbose).
- `cd backend/database_administrator && make lint` — green.
- `cd backend/database_administrator && make build` — green.
- `docker compose -f docker-compose.yaml build` — green.
- `frontend` pnpm test:ci + pnpm test:e2e — green (against the now-full stack).
- PR title: `feat(backend): verify authjs JWE cookie on /api/* + protected demo endpoint #N`.

---

## Cross-cutting tasks (run alongside any PR)

### X.1 Frontend lint + format

- `pnpm lint` and `pnpm fmt.check` pass at every step.
- Run: in every commit that touches `frontend/`.

### X.2 Backend lint + format

- `make lint && make fmt` pass at every step.
- Run: in every commit that touches `backend/database_administrator/src/`.

### X.3 Commit-message discipline

- Conventional commits, no `Co-Authored-By` trailer (per `openspec/AGENTS.md`).
- Footer references the change name when meaningful: `Refs: cachicamas-github-login`.

---

## Rollout per PR

| Step | PR-1 | PR-2 | PR-3 |
| --- | --- | --- | --- |
| Branch | `feat/cachicamas-github-login-pr1-schema-identity` | `feat/cachicamas-github-login-pr2-frontend-auth` | `feat/cachicamas-github-login-pr3-backend-verifier` |
| Rebase on | `main` | PR-1 merged | PR-2 merged |
| Merge target | `main` | `main` | `main` |
| Pre-merge smoke | `make test && docker compose build` | `pnpm test:ci && pnpm test:e2e (against mocks)` | full compose + sign-in + `/api/v1/protected/whoami` returns 200 |

---

## Out-of-scope follow-up changes (NOT in this tasks.md)

- Add Google / Microsoft provider (extensible via `plugin@auth.ts`).
- DB-backed sessions via Auth.js DB adapter.
- `AUTH_SECRET` rotation recipe.
- `identity.audit` table for sign-in events.
- CSP / HSTS headers.
- Email verification before linking.
- Rate-limit on `/auth/*` endpoints.

---

## Review checklist for `sdd-apply` gate

- [ ] User confirms the three-PR chain strategy before PR-1 starts.
- [ ] Each PR opens with one of the three commits in `feat/...` branches.
- [ ] Each PR runs the existing `make test` / `pnpm test:ci` / `pnpm test:e2e` gate and surfaces the diff in the PR body.
- [ ] `verify-report.md` is filled out incrementally per PR (after PR-3 merge).
- [ ] After PR-3 merges, `sdd-archive` runs and the folder moves under `openspec/changes/archive/2026-07-03-cachicamas-github-login/`.
- [ ] The two ADRs (`0001`, `0002`) are committed in PR-1, not later.
