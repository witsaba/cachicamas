# SDD Explore — `cachicamas-github-login`

> Persisted 2026-07-03 in the parent session (the `sdd-explore` subagent was launched
> without the executor tool set; explore was performed inline). Both copies
> (this file + Engram topic `sdd/cachicamas-github-login/explore`) carry the same
> information; this markdown is the source of truth for the parent.

## Scope

Map the current auth/account surface (there is none today), document GitHub
OAuth integration options for the Qwik 1.20 + Qwik City Node-SSR frontend
talking to the Go 1.26 + Echo v5 + Postgres 18 backend over docker-compose,
and produce a 3–5 option shortlist the user can choose between. No proposal,
no implementation, no decision.

## Confirmed web findings (verified against primary sources)

| Claim | Source | Status |
| --- | --- | --- |
| `@auth/qwik` is the current package name (post-rebrand from `@builder.io/qwik-auth`) | <https://qwik.dev/docs/integrations/authjs/> | confirmed |
| Install via `pnpm run qwik add auth` scaffolds `routes/plugin@auth.ts` | qwik.dev integration page | confirmed |
| GitHub provider is built-in: `import GitHub from "@auth/qwik/providers/github"` | qwik.dev + authjs.dev | confirmed |
| Env vars: `AUTH_GITHUB_ID`, `AUTH_GITHUB_SECRET`, `AUTH_SECRET` (32-byte base64) | qwik.dev + authjs.dev | confirmed |
| Server-side helpers: `QwikAuth$(({ env }) => ({ ... }))` returns `{ onRequest, useSession, useSignIn, useSignOut }` | qwik.dev | confirmed |
| Default session storage is a signed JWE cookie (default Auth.js behavior) | authjs.dev + qwik.dev | confirmed |
| Auth.js for Qwik is **still pre-1.0** ("could have bugs") | qwik.dev docs | confirmed |
| GitHub OAuth Web App flow supports **PKCE** as of 2025-07-14 | <https://github.blog/changelog/2025-07-14-pkce-support-for-oauth-and-github-app-authentication/> | confirmed |
| GitHub OAuth Apps: confidential server-side clients do NOT require PKCE; public SPAs SHOULD use PKCE | <https://docs.github.com/en/apps/oauth-apps/building-oauth-apps/authorizing-oauth-apps> | confirmed |

Latest `@auth/qwik` npm version on registry was `0.9.2` (Oct 26 2025); `@auth/qwik` has
**430 files, ~87KB unpacked, ~757 weekly downloads** — small but real. Pre-1.0 caveat
must appear in the proposal.

The official Auth.js docs page `authjs.dev/reference/qwik` confirms the **same API surface**
as the Qwik integration page (`useSession`/`useSignIn`/`useSignOut`; `/auth/callback/:provider`
route convention; double-submit CSRF cookie). No DB adapter is mentioned on either page as
required — DB-backed sessions are an optional `@auth/core/adapter` (Auth.js core) but the
adapter wiring for Qwik is not officially documented (would need an ADR).

## Project map (verified)

### Frontend (`frontend/`)

- **Runtime**: `src/entry.express.tsx` is a **plain Node http server** (no Express, no nginx)
  built from `vite build -c adapters/node-server/vite.config.ts`. Reads `./server/entry.express.js`
  in Docker. Reverse proxies `/api/*` to `API_TARGET` (defaults to `http://database_administrator:8080`
  in compose, overridable via env). Serves Qwik client chunks from `dist/` with content-hashed
  filenames.
- **`frontend/Dockerfile`** uses node:22-alpine in a single Node process; no nginx, no Express.
  Sets `API_TARGET=http://database_administrator:8080`, `ORIGIN=http://localhost:3015`,
  `PUBLIC_API_BASE_URL=/api`, `SERVER_API_BASE_URL=http://database_administrator:8080`.
- **`frontend/package.json`** (head):
  - `@builder.io/qwik ^1.20.0`, `@builder.io/qwik-city ^1.20.0`
  - `zod 3.25.48` — only prod dep today
  - `vitest ^0.34.6`, `@playwright/test ^1.61.1`, `eslint`, `prettier`, `tailwindcss`
  - **No auth-related dependency.**
- **`frontend/src/routes/`**:
  - `index.tsx`, `index.spec.tsx`
  - `organizations/{index.tsx,index.spec.tsx,[id]/...,new/...}`
  - **No `/auth/*`, no `plugin@auth.ts`, no `[[...auth]]/` folder.** ✓ confirms zero auth surface.
- **`frontend/.gitignore`** covers `node_modules/`, `dist/`, `server/`, `.env*`.
- **`frontend/adapters/`** confirmed as empty entry — only `node-server/vite.config.ts` is the
  build adapter.

### Backend (`backend/database_administrator/`)

- **Module**: `github.com/cachicamas/backend/database_administrator`, Go 1.26.3.
- **Layout** (hexagonal):
  - `src/cmd/`, `src/application/`, `src/domain/`, `src/infrastructure/`, `src/interfaces/`,
    `src/migration/`, `src/otel/`, `src/tools/`.
  - `src/domain/{health.go,migration.go,organization.go,organization_test.go}`.
  - `src/interfaces/http/{cors.go,health_handler.go,health_handler_test.go,organization_handler.go,organization_handler_test.go}`.
  - `src/migration/sql/` holds goose migrations:
    - `20260621120000_hello_world.sql`
    - `20260622120000_orgs_and_projects.sql`
    - `20260622120001_requirements_and_milestones.sql`
    - `20260622120002_tasks_and_specs.sql`
- **`go.mod`**: Echo v5.2.1, pgx/v5, otel, otelslog, go-retry, goose. **No auth deps today.**
- **`src/interfaces/http/cors.go`**: dev-only CORS allowlist (no `Access-Control-Allow-Credentials`
  ever). Production assumes same-origin via the Qwik `/api/*` proxy.
- **`Organization` domain entity**: `id, shortname, full_name, identification, is_active, email,
  phone, created_at, updated_at`. **No `owner_id`/`created_by` column today** — orgs are
  not yet user-owned. This is a clean baseline for first-slice auth.

### Postgres (`infra/postgres/init/01-init.sql`)

- Three schemas already provisioned at first boot:
  - `catalog` — product catalog
  - **`identity` — "Users, roles, sessions, audit log"** ← natural home for the new auth tables
  - `observability` — operational state
- `queen` role (NOSUPERUSER, CREATEROLE/CREATEDB/REPLICATION) owns all three schemas.
  Least-privilege app role (`cachicamas_app`) is documented as a future migration.
- `public.schema_migrations` is goose-v3-shaped (`id, version_id, is_applied, tstamp`).

### Frontend ↔ backend wiring (`docker-compose.yaml`)

- Two services confirmed for this change: `database_administrator` (Go) and `cachicamas-frontend`.
- The Qwik Node server fronts the Go binary on `/api/*`, so the **only origin the browser
  ever talks to is the Qwik host** (`http://localhost:3015` in dev, `http://<vps>` in prod).
- `docker-compose.yaml` has only two `environment:` blocks (one per service); new env vars
  for the frontend would be the place to inject `AUTH_GITHUB_ID`, `AUTH_GITHUB_SECRET`,
  `AUTH_SECRET`, `AUTH_TRUST_HOST=true`, `ORIGIN`, possibly `AUTH_URL`.

## Stack-specific gotchas (verified)

1. **Frontend runtime is Node SSR, not nginx**, so any "redirect to GitHub" page must
   originate from `entry.express.tsx`. The single `createServer` handler routes:
   `/api/*` → Go proxy; static; Qwik router. `routes/plugin@auth.ts` (the file `qwik add auth`
   creates) returns an `onRequest` middleware that handles `/auth/signin|/auth/callback/:provider|/auth/signout|/auth/session|/auth/csrf|/auth/providers` — Qwik will dispatch these AFTER the `/api/*` proxy
   branch, so we do NOT conflict with `/api/*`. This is clean.
2. **`AUTH_TRUST_HOST=true` is required** behind any non-Vercel/Netlify deploy (qwik.dev
   docs call this out). In compose/VPS we are behind a single Node process but origin
   detection still benefits from the env.
3. **`ORIGIN` env var** is already wired in `entry.express.tsx` (Qwik reads it for canonical
   tags and CSRF). Auth.js will need the same `ORIGIN` to build correct callback URLs.
4. **`AUTH_SECRET`** must be a 32-byte base64 random string (`openssl rand -base64 32`).
   Must be injected via `docker-compose.yaml` from the host environment (never committed).
5. **Go dev-mode CORS does NOT allow credentials** (`src/interfaces/http/cors.go` comment
   is explicit: "We deliberately do NOT enable Access-Control-Allow-Credentials"). For any
   option where the Go service is later asked to read the session cookie set by Auth.js,
   the cookie must be `SameSite=Lax` (browser default) on the Qwik origin. The
   reverse-proxy in `entry.express.tsx` sets `host` header to the target, so the browser
   sees only the Qwik origin → cookie scoping is single-origin by construction.
6. **Postgres schema `identity` is provisioned but empty** (`infra/postgres/init/01-init.sql`
   creates the schema, no tables). New tables `user`, `account`, `session`, `identity_audit`
   (or the Auth.js Adapter equivalent) belong in `identity.*`. We must use goose migrations
   (consistent with the rest of the backend), not `01-init.sql` (that file is one-shot for
   first boot).
7. **The Qwik Node SSR build does NOT bundle all dynamic deps** — the Dockerfile comment
   explicitly notes that dynamically-imported `undici` is copied into the runner image.
   `@auth/qwik` brings `@auth/core` and friends; we will need to check whether any of
   those are dynamic imports and add them to the `COPY --from=builder /app/node_modules/<pkg>`
   pattern. (Needs to be verified during `sdd-apply` by inspecting `frontend/server/` output.)
8. **No `user` concept on `organization` rows today**; this is intentional. Choosing how
   to link users ↔ orgs is a deferred decision we should NOT solve in the first slice
   (see Open questions below).
9. **Auth.js is pre-1.0** (qwik.dev is explicit about this). Production should pin the
   exact version in `package.json` (no `^`) and document the upgrade policy.
10. **GitHub OAuth App needs to be created manually** (Settings → Developer settings →
    OAuth Apps → New OAuth App). Callback URL = `http://localhost:3015/auth/callback/github`
    in dev; the VPS callback URL must be registered before deploy. Both must be added to the
    `.env.example` as documented defaults that the human fills in.
11. **Pre-1.0 caveat for `@auth/qwik`**: 757 weekly npm downloads, 430 files, ~87KB. Real
    users exist but the surface is small. A defensive `pnpm audit` is part of the apply plan.

## Options shortlist (NOT a recommendation)

> Each option carries a rollback path and an ADR-when-needed marker. Rollback plans
> are required by `openspec/config.yaml` for proposal phase.

### Option A — Frontend-only via `@auth/qwik`, JWE cookie session, Go stays auth-unaware

- **Roundtrip location**: Qwik Node SSR (`routes/plugin@auth.ts` + `entry.express.tsx`).
- **Session location**: signed JWE cookie on the Qwik origin; nothing persisted server-side.
- **New `frontend/package.json` deps**: `@auth/qwik@0.9.2` (ADR needed — first top-level
  prod dep that has auth semantics).
- **New `backend/go.mod` deps**: none.
- **New Postgres tables**: none.
- **GitHub OAuth App registration**: callback = `<ORIGIN>/auth/callback/github` for each env.
- **Estimated changed lines**: ~80–150 frontend + ~30 lines `docker-compose.yaml`/`.env.example`
  - a few lines README. Total well under the 400-line PR budget.
- **Reversibility**: remove `routes/plugin@auth.ts`, uninstall `@auth/qwik`, delete env vars.
- **Trade-offs**:
  - ✅ Fastest path; minimal surface; no new Go deps; no DB migration.
  - ✅ Existing dev CORS settings remain correct (no credentials).
  - ⚠️ Org-owned-by-user UX decisions deferred indefinitely (no `user` table).
  - ⚠️ Any later "Qwik calls protected `/api/*` endpoint" works by sending the same JWE
    cookie, but the Go service currently has no way to **verify** that cookie; we'd need a
    follow-up change to add JWE verification on the Go side or accept the change as
    "frontend-only auth, backend stays open". This is the biggest go/no-go question.
  - ⚠️ `@auth/qwik` pre-1.0 risk: pin exact version.

### Option B — Frontend via `@auth/qwik` + Go verifies the same JWE cookie

- **Roundtrip location**: Qwik (same as A).
- **Session location**: JWE cookie on Qwik origin; **Go verifies** the same cookie for
  future protected endpoints using the shared `AUTH_SECRET`.
- **New `frontend/package.json` deps**: `@auth/qwik@0.9.2` (ADR).
- **New `backend/go.mod` deps** (each is a new top-level dep → ADR):
  - `github.com/lestrrat-go/jwx/v2` (for JWE verification using HS256 / A256GCM).
- **New Postgres tables**: none.
- **GitHub OAuth App registration**: same as A.
- **Estimated changed lines**: A's ~120 lines + a new `interfaces/http/auth_middleware.go`
  (~80 LOC + tests) + a small domain `Identity` port (~30 LOC) + ADR (~50 LOC). Roughly
  **280–380 lines** total. Likely just under the 400-line PR budget; might trigger chained PRs.
- **Reversibility**: drop the middleware and the JWE verification helper; keep frontend.
- **Trade-offs**:
  - ✅ Sets up a clean foundation for protected `/api/*` calls later.
  - ✅ Qwik still owns UX, Go doesn't need its own OAuth client.
  - ⚠️ New top-level Go dep (`jwx`); ADR required.
  - ⚠️ Cookie format drift between Auth.js and our JWE verifier is a real risk (Auth.js uses
    a specific JWE envelope; we need to match the exact key derivation). Strong TDD coverage
    required.

### Option C — Backend passthrough via Go (`golang.org/x/oauth2` + `golang.org/x/oauth2/github`)

- **Roundtrip location**: Go service (`/auth/github/start`, `/auth/github/callback`,
  `/auth/session`, `/auth/signout`).
- **Session location**: signed cookie (custom format) issued by Go on success. Session token
  may be random or derived from a Postgres-backed session store.
- **New `frontend/package.json` deps**: none (Qwik gets a `<form action="/auth/github">` or
  a plain `<a href>`).
- **New `backend/go.mod` deps** (each → ADR):
  - `golang.org/x/oauth2` (technically already in `x/net` indirect, but direct use requires promotion to direct → ADR).
  - `github.com/google/uuid` (already indirect; promotion → ADR).
  - If DB-backed sessions: `github.com/gorilla/sessions` OR `github.com/alexedwards/scs/v2`
    with a Postgres store (`github.com/antonlindstrom/pgstore` or `github.com/oxipipp/pgdbstore`).
- **New Postgres tables (identity schema)**: `identity.session` (or user table) + Auth.js's
  preferred shape if we ever revisit Auth.js.
- **GitHub OAuth App registration**: callback = `<api-origin>/auth/github/callback`. Since
  the Go service is only reachable via the Qwik `/api/*` proxy, the callback must resolve
  through that proxy. Two viable sub-shapes:
  - **C1.** Frontend proxies: `http(s)://<host>/api/auth/github/callback` ← Go sees
    `<req.Host>` and reconstructs the public callback.
  - **C2.** Frontend has no proxy entry; callback URL = `http(s)://<api-host>`; we
    would need to expose Go separately or use a different routing layer.
- **Estimated changed lines**: ~100 Go cmd/routes + ~80 application service + ~60 domain
  - ~80 interfaces/http + migrations + tests + ADR. Roughly **350–500 lines**. Likely
  chained PRs.
- **Reversibility**: drop the new routes, delete the session table migration, remove deps.
- **Trade-offs**:
  - ✅ Full control over session lifecycle in Postgres; matches the existing
    `identity` schema intent.
  - ✅ No pre-1.0 frontend dependency.
  - ⚠️ Coupling between GitHub OAuth callback URL and the Qwik proxy composition; the
    callback URL becomes a deployment-environment variable that must agree across compose,
    GitHub OAuth App registration, and the Qwik static `ORIGIN` env.
  - ⚠️ Larger code surface; needs `application/`+`interfaces/http/`+`infrastructure/` layers
    - tests + ADR.

### Option D — Backend passthrough via `dghubble/gologin/v2` + `gorilla/sessions` (or `scs/v2`)

Same shape as C with a different library. `gologin/v2` handles the GitHub-specific OAuth
glue (`github.Handler`), saving ~50 LOC vs hand-rolling. Still requires a session store and
a migration. Trade-offs are essentially those of C with one fewer ADR.

### Option E — Hybrid: Auth.js on Qwik for UX, callback reverse-proxied to Go for session storage

- **Roundtrip location**: Auth.js issues the redirect; the `/auth/callback/github` handler
  *forwards* the `code` to Go (`POST /api/auth/exchange`) which exchanges it, persists
  an `identity.session` row, returns a session cookie scoped to Go's domain.
- **Session location**: a Postgres `identity.session` row; cookie set by Go with a token
  that resolves server-side.
- **New `frontend/package.json` deps**: `@auth/qwik@0.9.2` (ADR).
- **New `backend/go.mod` deps**: `golang.org/x/oauth2`, jwx or scs (ADRs).
- **New Postgres tables (identity schema)**: `identity.session`, possibly `identity.user`.
- **Estimated changed lines**: ~500–700 lines spread across the two services; chained PRs certain.
- **Reversibility**: drop the proxy glue, fall back to Option A.
- **Trade-offs**:
  - ✅ Best UX (Auth.js components + sessions in DB).
  - ⚠️ Highest complexity; the proxy-shuffle is brittle (the Qwik SSR hook must forward
    the right headers/body to Go and apply Go's response cookies).
  - ⚠️ Two ADRs minimum.

## ADR pre-staging (to be filed during `sdd-design`, not now)

| Slug | Trigger | Decision drivers |
| --- | --- | --- |
| `adr-authjs-qwik` | `@auth/qwik` accepted as a direct frontend prod dep | pre-1.0 risk vs. velocity; cookie-only session vs. DB session; QR vs. App vs. token |
| `adr-jwx-for-jwe` (only if B) | promote `lestrrat-go/jwx/v2` to verify Auth.js JWE in Go | key derivation parity with Auth.js; threat model; secret rotation |
| `adr-golang-x-oauth2-promote` (C/D/E) | promote `golang.org/x/oauth2` and `github.com/google/uuid` from indirect to direct | cleaner ADR; needed for direct OAuth client |
| `adr-gologin-vs-x-oauth2` (D only) | pick gologin vs hand-rolled handler | LOC saved vs. dep footprint; gologin v2 maintenance status |
| `adr-session-store-pg` (C/D/E) | pick `gorilla/sessions` vs `scs/v2` with a Postgres store | ergonomics; CSRF protection defaults; license |

## Open questions for the user

### Product / business (run in the pre-proposal interactive round)

1. **MVP scope cut**: is this slice "login on the frontend only" (Option A, Go stays
   open for now), or "login on the frontend AND a protected `/api/*` route on the Go
   side" (Option B+)? The choice changes 2/3 of the implementation surface.
2. **Identity persistence**: keep the default JWE cookie session (Stateless, ≤A/B) or
   persist users/sessions in Postgres `identity.*` (C/D/E)? The `identity` schema is
   provisioned but empty; persisting aligns with the schema intent.
3. **Account linking policy**: a returning GitHub user with a previously-seen email in
   our DB — should we auto-link on email match (Auth.js default behavior), force a
   separate "link account" step, or treat GitHub identity as authoritative (every
   login = a fresh `user` row in `identity.user`)?
4. **Relationship to `organization`**: today orgs have no `owner_id`. Should the
   first-slice of auth add an optional `organization.owner_user_id` (FK to
   `identity.user`) so orgs are owned-by-user from day one, OR keep orgs ownerless and
   defer the `owner_user_id` to a future change?
5. **Role model**: do you need any authz notion now (e.g. `user` vs `admin`), or just
   authenticated/unauthenticated for this slice?
6. **Out of scope for this slice**: other providers (Google, Microsoft), 2FA, SCIM,
   team invitations? Confirm we ship GitHub-only first.

### Architecture (decided in `sdd-design`, not the proposal round; mentioned for awareness)

- Cookie scope: `Secure` + `SameSite=Lax` + `HttpOnly` is the starting assumption.
- PKCE: not required for confidential Auth.js client, but decide whether to opt in
  anyway for forward-compat with public clients.
- Cookie domain: single-origin via the Qwik reverse proxy; no cross-origin cookies.
- Session lifetime + refresh token policy.
- Audit logging: do we write `identity.audit` rows on sign-in/sign-out?
- Logout semantics: client-side cookie drop only, or also invalidate a DB session row?

## Next recommended

```text
Run the sdd-proposal interactive question round with the user, covering the 6 product
questions above (one round, then summarize assumptions and offer a second round).
Then delegate sdd-proposal with the user's answers as inputs.
```

## Status

`status: explored`
