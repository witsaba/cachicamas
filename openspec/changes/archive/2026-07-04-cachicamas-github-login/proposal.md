# Proposal: cachicamas-github-login

> **Change name**: `cachicamas-github-login`
> **Status**: proposed
> **Phase**: SDD proposal (post-explore; pre-proposal interactive question round completed)
> **Persisted**: 2026-07-03 to Engram topic `sdd/cachicamas-github-login/proposal`
> **Companion artifact**: `openspec/changes/cachicamas-github-login/explore.md` (parent)

---

## 1. Intent

Add **GitHub OAuth login** to the cachicamas frontend and a **server-side session
verifier** to the backend, such that the human (braejan) — and any future GitHub
user — can sign in once, land on an authenticated home view, and have their
identity model persisted server-side. The slice introduces the foundational
`identity.*` Postgres tables (`identity.user`, `identity.account`) and an
optional `organization.owner_user_id` FK. Sessions remain a stateless signed
JWE cookie; the backend's Go service can verify that same cookie so future
protected `/api/*` endpoints have a tested, audited verifier ready.

### Business problem

Today cachicamas has **zero authentication**. The `/organizations/new` form writes
to Postgres over the `/api/*` reverse proxy with no notion of *who* is creating
the row. Every dev environment is shared-state; staging has no concept of
"identity". As the product grows (multi-user, multi-tenant), this becomes a
blocking gap: we cannot ship a protected endpoint, cannot say who created an
organization, cannot put team invitations behind a permission check.

### Target users and situations

- **Primary user today**: braejan (Witsaba). Wants to log in via GitHub and see
  his own profile / be recognized on protected pages.
- **Primary user tomorrow**: any GitHub user the OAuth App is published to. Can
  sign in, has a row in `identity.user`, sees their profile, can be credited
  as the owner of organizations they create.
- **Trigger moment**: a user lands on `/`, clicks "Sign in with GitHub",
  completes the OAuth roundtrip on GitHub.com, and is redirected back to the
  app authenticated. Same UX pattern as the modern web (Linear, Vercel, etc.).

### Current-state gap

| Surface | Today | After this slice |
| --- | --- | --- |
| `frontend/src/routes/` | No `/auth/*`, no `plugin@auth.ts` | New `routes/plugin@auth.ts`; built-in `/auth/signin`, `/auth/callback/github`, `/auth/signout`, `/auth/session`, `/auth/csrf`, `/auth/providers` |
| Frontend `package.json` | No auth deps | New `@auth/qwik@0.9.2` (pinned exact) |
| `backend/database_administrator/go.mod` | No auth deps | New `github.com/lestrrat-go/jwx/v2` (pinned to current minor) |
| `interfaces/http/` | `health_handler.go`, `cors.go`, `organization_handler.go` | + `auth_middleware.go` + `auth_middleware_test.go` (covers JWE verification) |
| `domain/` | `organization.go`, `health.go`, `migration.go` | + `identity.go` (port for user/account lookup), `identity_test.go` |
| `application/` | `organization_service.go` | + `identity_service.go` (sign-in, sign-out, get-by-id) |
| `infrastructure/postgres/` | `organization_repository.go` | + `identity_repository.go` |
| `migration/sql/` | `20260621120000…20260622120002_*` | + `20260703NNNN_github_login.sql` (creates `identity.user`, `identity.account`; adds `organization.owner_user_id`) |
| `infra/postgres/init/01-init.sql` | `identity` schema exists, empty | unchanged (schema is provisioned; tables come from goose migration) |
| Go HTTP layer | No auth middleware | `auth_middleware.go` reads Auth.js JWE cookie, verifies with `AUTH_SECRET`, populates `c.Set("identity", *Identity)` |
| Profile page | none | New `frontend/src/routes/profile/index.tsx` (uses `useSession`) |
| Sign-in UI | none | New `frontend/src/components/sign-in-button/sign-in-button.tsx` + landing CTA on `/` |
| `.env.example` | No `AUTH_*` vars | Add `AUTH_GITHUB_ID`, `AUTH_GITHUB_SECRET`, `AUTH_SECRET`, `AUTH_TRUST_HOST=true`, `AUTH_URL`, comments |
| `docker-compose.yaml` | `frontend` env: ORIGIN, PUBLIC_API_BASE_URL, SERVER_API_BASE_URL, API_TARGET, PORT, HOST | + `AUTH_GITHUB_ID`, `AUTH_GITHUB_SECRET`, `AUTH_SECRET`, `AUTH_TRUST_HOST=true`, `AUTH_URL` (compose override) |

## 2. Scope

### In scope (this slice)

- `@auth/qwik` integration in Qwik Node SSR; GitHub provider; PKCE enabled.
- JWE cookie session (`AUTH_SECRET`, 32 bytes base64). Cookie attrs:
  `HttpOnly`, `Secure`, `SameSite=Lax`, `Path=/`. Scope is single-origin via
  the Qwik `/api/*` reverse proxy.
- New Postgres tables (under existing `identity` schema):
  - `identity.user` (`id BIGSERIAL`, `email CITEXT UNIQUE`, `name TEXT`,
    `image_url TEXT`, `created_at TIMESTAMPTZ`, `updated_at TIMESTAMPTZ`).
  - `identity.account` (`id BIGSERIAL`, `user_id BIGINT FK`, `provider TEXT`,
    `provider_account_id TEXT`, `UNIQUE(provider, provider_account_id)`).
- Migration `20260703NNNN_github_login.sql` adds both tables and
  `organization.owner_user_id BIGINT NULL FK → identity.user(id)`.
- Auth.js `signIn` callback that auto-links on email match (looked up in
  `identity.user` by lowercased email).
- Go-side JWE verifier (`interfaces/http/auth_middleware.go`) plus its TDD test
  pair.
- `/profile` route + sign-in/out buttons in the landing page.
- `docker-compose.yaml` + `.env.example` updated; documentation in
  `frontend/README.md` and `backend/database_administrator/README.md`.

### Out of scope (deferred but related)

- Other OAuth providers (Google, Microsoft, generic OIDC). Reusable hook points
  are in place via `useSignIn({ providerId: 'github' })`; adding more providers
  is a follow-up change.
- 2FA / step-up auth / WebAuthn.
- SCIM, team invitations, "join org" UX.
- Server-side session revocation table — `identity.session` is intentionally not
  added (you chose stateless cookie for this slice).
- Audit log table (`identity.audit`) — sign-in events will be slog-logged for
  observability, but no DB audit row.
- Rate-limiting / abuse defense on `/auth/*` endpoints.
- E-mail verification before linking — GitHub's email is trusted as-is.
- Renaming the existing `cachicamas_pg` `postgres` SUPERUSER or adding a
  least-privilege `cachicamas_app` role (separate change already noted).
- CSP / HSTS headers — recommended as a follow-up change, since they are
  product-wide and not specific to auth.

## 3. Affected areas

- **Frontend**: `package.json`, `src/entry.express.tsx` (env reads),
  `src/routes/plugin@auth.ts` (new), `src/routes/profile/index.tsx` (new),
  `src/components/sign-in-button/*` (new), `src/routes/index.tsx` (CTA wired).
  Tests: `vitest` for components + `playwright` for sign-in → profile round-trip.
- **Frontend infra**: `Dockerfile` (may need `COPY` for unaudited `@auth/*`
  dynamic imports — see Gotcha #7 in explore.md), `docker-compose.yaml`,
  `.env.example`.
- **Backend**: `go.mod` (new `lestrrat-go/jwx/v2`), `src/cmd/server/main.go`
  (env reads, register middleware), `src/interfaces/http/{auth_middleware.go,
  auth_middleware_test.go, router wiring}`, `src/domain/identity.go` (User,
  Account, Identity port), `src/domain/identity_test.go`,
  `src/application/identity_service.go`, `src/infrastructure/postgres/
  identity_repository.go`, `src/migration/sql/20260703NNNN_github_login.sql`,
  `src/migration/sql/<up+down>`, `README.md`.
- **Postgres**: new tables in `identity` schema; new FK column on
  `organization`. Existing rows (orgs created before this slice) keep
  `owner_user_id = NULL` (we do NOT backfill).
- **No change**: `infra/`, `docker-compose.vps.yaml` (other than env updates
  via override), `openspec/specs/db-migrations/*` (no new migration paradigm),
  `docker-compose.yaml` beyond the `frontend` service's env block.
- **Not affected**: existing `/api/organizations` calls; the reverse proxy
  contract; health checks; CORS config (no credentials enabled by design).

## 4. Architecture (one-liner)

GitHub OAuth roundtrip happens in `@auth/qwik` on Qwik Node SSR; session is a
signed JWE cookie (`AUTH_SECRET`, PKCE on); backend Go service can verify the
same cookie via `lestrrat-go/jwx/v2`; user/account rows persist in
`identity.user` and `identity.account`; `organization.owner_user_id` is an
optional FK.

## 5. Proposal question round — answered summary

| # | Question | Answer |
| --- | --- | --- |
| 1 | MVP scope | Frontend + Go JWE verifier (parent picks: "evaluate by security, choose the most secure" → Option B). |
| 2 | Identity persistence | Stateless JWE cookie only (no `identity.session` table). |
| 3 | Account linking | Auto-link on email match (Auth.js default), via `signIn` callback in `routes/plugin@auth.ts`. |
| 4 | Org ↔ user link | Add nullable `organization.owner_user_id` FK now. |
| 5 | Role model | Not introduced in this slice. (Q deferred; user did not need a second round.) |
| 6 | Other providers | Out of scope this slice (follow-up). |

Open assumptions from the architecture section that were NOT in the question
round but are part of the design:

- PKCE **on**, despite Auth.js being a confidential client (forward-compat).
- Cookie attrs: `HttpOnly`, `Secure`, `SameSite=Lax`, `Path=/`.
- Session lifetime: default Auth.js `jwt` strategy (30 days rolling); revisit in
  a follow-up if revocation is needed.
- No `identity.audit` table; sign-in events are `slog` + `otelslog` only.
- Go-side JWE verifier MUST match Auth.js's exact key derivation (HS256
  symmetric). Implementation reference: Auth.js encodes the JWE with `deriveKey`
  from `AUTH_SECRET` and an HKDF/SHA-256 step. The Go verifier uses
  `lestrrat-go/jwx/v2/jwe` with the corresponding A256GCM-HS256 envelope.
  (Decided in `sdd-design`.)
- Auth.js version pinned to **exactly `0.9.2`** (no caret) to lock the pre-1.0
  surface; upgrade is a future change.

## 6. Risks

| Risk | Severity | Likelihood | Mitigation |
| --- | --- | --- | --- |
| `@auth/qwik` pre-1.0 has bugs that bite login | Med | Med | Pin exact version; `pnpm audit` in CI; test in dev before merging. |
| JWE envelope mismatch between Auth.js and Go verifier | High | Med | Implementation parity is checked by a TDD test that signs a cookie in a fixture using Auth.js's own encoder and reads it via the Go verifier (cross-tooling round-trip). |
| GitHub OAuth App callback URL drift between envs | Med | Med | `.env.example` is the only source; docker-compose env from host; `frontend/README.md` lists the exact URLs to register on the GitHub side; lint script greps the value. |
| `AUTH_SECRET` leaked to git | High | Low | `.env.example` uses placeholders only; `.gitignore` covers `.env`; secret-handling rule in README; CI scans for the prefix. |
| Cookie scope confusion (e.g., wrong `Domain`) | Low | Low | Single-origin by construction; no domain attr set. |
| `identity.user` schema fails on `email CITEXT UNIQUE` because an existing email has odd Unicode normalization | Low | Low | Verified during apply. |
| Auto-link via email conflicts with privacy expectations | Low | Low | Documented as the chosen policy. |
| New deps require ADRs which were not pre-filed | Med | Low | Two ADRs drafted during design (see §7). |

## 7. ADR pre-staging (to be filed in design, not proposal)

- `adr-authjs-qwik` — accept `@auth/qwik@0.9.2` as a direct frontend prod dep
  despite pre-1.0 status. Drivers: velocity; existing pre-1.0 alternatives
  (`@builder.io/qwik-auth`) are deprecated. Consequences: track upstream
  releases; upgrade is its own change with a regression test gate.
- `adr-jwx-for-jwe` — promote `lestrrat-go/jwx/v2/jwe` to verify Auth.js JWE
  cookies on the Go service. Drivers: avoids re-implementing JWE key
  derivation; strong security posture; small dep footprint. Consequences: must
  cross-check key derivation against Auth.js's JS encoder at design time.

## 8. Rollback plan

The slice is designed to be **safely reversible in two stages**:

### Stage 1 — disable login in dev (no DB changes)

1. Revert `docker-compose.yaml` `frontend` `environment:` to remove the
   `AUTH_*` vars.
2. Comment out the `QwikAuth$` export in `routes/plugin@auth.ts`.
3. Revert profile route additions; revert landing-page CTA.
4. Re-run `docker compose up -d --build frontend` and verify the landing page
   still loads without sign-in.

At this point: identity is gone, no DB effect, GitHub OAuth App registration
remains (we can either keep or remove it; recommended: keep until you confirm
rollback completed).

### Stage 2 — drop new schema (if needed)

1. Run a goose down-migration:
   `ALTER TABLE organization DROP COLUMN owner_user_id;`
   `DROP TABLE identity.account;`
   `DROP TABLE identity.user;`
2. Revert the migration file (`20260703NNNN_github_login.sql`'s Down section).
3. Remove `letestrat-go/jwx/v2` from `go.mod` (`go mod tidy`).
4. Uninstall `@auth/qwik` and `qwik add auth` artifacts; remove `routes/plugin@auth.ts`.
5. Optionally remove the GitHub OAuth App registration.

**Stage 2 is destructive** but bounded: zero existing rows in `identity.*`, and
back-filling `organization.owner_user_id` from NULL to a real FK is not required
for any existing data (no orgs have a known owner to migrate). After Stage 2,
the repo is byte-identical to pre-this-slice plus the empty `identity` schema
it had before.

## 9. Success criteria

The slice is **accepted** when:

1. The frontend dev mode shows a "Sign in with GitHub" button on `/`.
2. Clicking it redirects to GitHub, completes the roundtrip, and lands the
   user on `/profile` with their GitHub name + email rendered from
   `useSession()`.
3. A row exists in `identity.user` and `identity.account` for that GitHub
   user.
4. Restarting the dev container and reopening `/profile` while the cookie is
   valid still renders the session (refresh-token / rolling-cookie works).
5. Going to `/auth/signout` clears the cookie; `/profile` now redirects to
   `/auth/signin`.
6. Go side: a TDD test signs a cookie with the Auth.js encoder
   (`@auth/core/jwt` shape), passes the cookie in a `Cookie:` header to a
   test handler behind the new middleware, and the middleware successfully
   sets `c.Get("identity")` to the expected user record. Negative test: a
   tampered cookie yields HTTP 401.
7. Vitest suite (`pnpm test:ci`) is green. Playwright e2e (`pnpm test:e2e`)
   is green for the existing create-org flow plus a new
   `sign-in-then-create-org` flow.
8. `make test` (Go, `-race -v`) is green. `make lint` is green.
9. `docker compose -f docker-compose.yaml build` succeeds; the
   `cachicamas-frontend` container starts and reports its healthcheck OK.
10. `.env.example` documents every new env var with a comment block,
    placeholder values, and a note that `AUTH_SECRET` must be
    `openssl rand -base64 32`.
11. The two ADRs (`adr-authjs-qwik`, `adr-jwx-for-jwe`) are committed under
    `docs/adr/` with status `Accepted`.

## 10. Open items for `sdd-design`

- Decide exact callback URL convention (one URL per env or dynamic via
  `AUTH_URL`).
- Final cookie scope and `Domain` policy.
- Decide whether `/profile` lives behind a `routeLoader$` server-side guard or
  uses `onRequest`.
- Decide on the Adapter choice if/when we ever add a DB session (out of scope
  this slice — pre-staged only).
- Confirm Auth.js JWE envelope structure for the Go verifier to match exactly
  (`a256GCMKW` + `A256GCM` content encryption + HS256-derived KEK).
- Confirm `frontend/Dockerfile` `undici`-style copy step for any unaudited
  `@auth/*` deps.
