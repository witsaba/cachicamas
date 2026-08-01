# Qwik City App ⚡️

- [Qwik Docs](https://qwik.dev/)
- [Discord](https://qwik.dev/chat)
- [Qwik GitHub](https://github.com/QwikDev/qwik)
- [@QwikDev](https://twitter.com/QwikDev)
- [Vite](https://vitejs.dev/)

---

## Project Structure

This project is using Qwik with [QwikCity](https://qwik.dev/qwikcity/overview/). QwikCity is just an extra set of tools on top of Qwik to make it easier to build a full site, including directory-based routing, layouts, and more.

Inside your project, you'll see the following directory structure:

```
├── public/
│   └── ...
└── src/
    ├── components/
    │   └── ...
    └── routes/
        └── ...
```

- `src/routes`: Provides the directory-based routing, which can include a hierarchy of `layout.tsx` layout files, and an `index.tsx` file as the page. Additionally, `index.ts` files are endpoints. Please see the [routing docs](https://qwik.dev/qwikcity/routing/overview/) for more info.

- `src/components`: Recommended directory for components.

- `public`: Any static assets, like images, can be placed in the public directory. Please see the [Vite public directory](https://vitejs.dev/guide/assets.html#the-public-directory) for more info.

## Add Integrations and deployment

Use the `pnpm qwik add` command to add additional integrations. Some examples of integrations includes: Cloudflare, Netlify or Express Server, and the [Static Site Generator (SSG)](https://qwik.dev/qwikcity/guides/static-site-generation/).

```shell
pnpm qwik add # or `pnpm qwik add`
```

## Development

Development mode uses [Vite's development server](https://vitejs.dev/). The `dev` command will server-side render (SSR) the output during development.

```shell
npm start # or `pnpm start`
```

> Note: during dev mode, Vite may request a significant number of `.js` files. This does not represent a Qwik production build.

## Preview

The preview command will create a production build of the client modules, a production build of `src/entry.preview.tsx`, and run a local server. The preview server is only for convenience to preview a production build locally and should not be used as a production server.

```shell
pnpm preview # or `pnpm preview`
```

## Production

The production build will generate client and server modules by running both client and server build commands. The build command will use Typescript to run a type check on the source code.

```shell
pnpm build # or `pnpm build`
```

## First-run onboarding (Organizations)

The frontend ships the very first UI surface: a four-route
aphantasia-friendly Organizations flow.

| Route             | Purpose                                                                                                                                                  |
| ----------------- | -------------------------------------------------------------------------------------------------------------------------------------------------------- |
| `/`               | Brand mark + tagline + a single "Get started" CTA to `/ownboarding`.                                                                                     |
| `/ownboarding`    | First-run setup form. Collects `full_name` + `identification` to create the unique organization. On success, redirects to `/home`.                       |
| `/workspaces`     | Workspaces list. Authed + ownboarded. Empty CTA when zero workspaces; list of cards when 1+. Each card shows name, primary repo, and linked-repo count.  |
| `/workspaces/new` | Workspace creation form. Authed + ownboarded. Single field `name` + GitHub repo picker for primary repo.                                                 |
| `/workspaces/:id` | Workspace detail. Authed + ownboarded. Shows workspace name + primary repo + linked repos. "Add repository" / "Disconnect" / "Delete workspace" buttons. |

### Aphantasia-friendly layout

- Text-first, single column. No hero images, no carousels, no decorative icons.
- Single-column form, capped at 42 rem wide (Tailwind `max-w-2xl`).
- Labels are questions or verbs. Required fields have no `placeholder`.
- Progressive disclosure: the "Review" field group (short name, email, phone) renders only when the required fields are filled and the user clicks "Add optional details" (or blurs any optional field).
- Submitting a 409 renders the locked message ("This slug is already taken. Try another.") inline below the slug input. Navigation does not happen on 409.

### Auto-derivation rule

When the user types in the `full_name` field, the `identification` (slug) field is auto-derived after a 200 ms debounce:

1. Lowercase the input.
2. Replace any character that is NOT `[a-z0-9-]` with `-`.
3. Collapse runs of `-` into a single `-`.
4. Strip leading and trailing `-`.
5. Truncate to 60 chars; if truncation lands on `-`, strip the trailing `-`.

Once the user manually edits the slug field, auto-derivation stops until the field is cleared.

## Workspaces (2026-07-06)

After organization ownboarding, the user can create **workspaces**: logical
containers that map 1:1 to a primary GitHub repository and optionally
connect N additional repos. Each workspace lives behind the
`requireAuthRedirect` + `requireOwnboarding` gate chain (same pattern as
the ownboarding flow).

| Route             | Purpose                                                                                                                                                  |
| ----------------- | -------------------------------------------------------------------------------------------------------------------------------------------------------- |
| `/workspaces`     | Workspaces list. Authed + ownboarded. Empty CTA when zero workspaces; list of cards when 1+. Each card shows name, primary repo, and linked-repo count.  |
| `/workspaces/new` | Workspace creation form. Authed + ownboarded. Single field `name` + GitHub repo picker for primary repo.                                                 |
| `/workspaces/:id` | Workspace detail. Authed + ownboarded. Shows workspace name + primary repo + linked repos. "Add repository" / "Disconnect" / "Delete workspace" buttons. |

The Workspaces link is in the avatar dropdown (auth-aware per existing
ADR-0010/0011 patterns).

### Workspace env vars

| Variable                                | Default | Purpose                                                                                              |
| --------------------------------------- | ------- | ---------------------------------------------------------------------------------------------------- |
| `PUBLIC_GITHUB_REPO_PICKER_DEBOUNCE_MS` | `300`   | Debounce on the GitHub repo picker's search input (used in `/workspaces/new` and `/workspaces/:id`). |

### Out-of-scope follow-up

UX-10 (AI pre-fill from PRD ideas) is a follow-up. A literal
`TODO(organizations-first-front): UX-10` comment is present
in the source; the grep test
`src/__tests__/ux-todo.spec.ts` enforces its presence so a
future PR cannot ship without addressing it.

### Tests

The frontend uses Vitest with `createDOM()` from
`@builder.io/qwik/testing`. Run:

```shell
pnpm test:ci
```

The runner discovers every `*.spec.{ts,tsx}` file under
`src/`. The full PR 2 surface (including the 4 routes, the
form, the empty-state component, and the Zod schema parity
test) is locked by 56 tests.

#### End-to-end tests

For the wire contract between Qwik and the database_administrator
Go binary we use [Playwright](https://playwright.dev). The e2e
specs live under `frontend/e2e/` and drive a real Chromium
browser against the full local stack:

```shell
pnpm test:e2e
```

### Workspace env vars

| Variable                                | Default | Purpose                                                                                              |
| --------------------------------------- | ------- | ---------------------------------------------------------------------------------------------------- |
| `PUBLIC_GITHUB_REPO_PICKER_DEBOUNCE_MS` | `300`   | Debounce on the GitHub repo picker's search input (used in `/workspaces/new` and `/workspaces/:id`). |

Pre-requisites:

- `docker compose up` (Postgres + database_administrator healthy)
- `pnpm dev` running in another terminal (Playwright's
  `webServer.reuseExistingServer: true` reuses it locally; in CI
  it starts the dev server itself)

The first time, install the Chromium browser binary:

```shell
pnpm exec playwright install chromium
```

The current spec (`e2e/create-organization.spec.ts`) is a
regression guard for the CORS bug fixed in commit `81f16fb` —
if `Access-Control-Allow-Origin` is ever missing on the Go
bin's responses again, this test fails immediately.

#### Dual mode (dev / container / VPS)

The Playwright runner is env-driven via `E2E_BASE_URL`. The same
spec runs against three different targets without any code change:

```bash
# Dev local (default). Playwright starts `pnpm dev` on :5173 and
# the spec runs against the Vite dev server. Same as before.
pnpm test:e2e

# Dockerized stack (CI / post-deploy validation). Playwright does
# NOT start a webServer — it trusts that the compose is up. The
# frontend is the dockerized nginx serving the Qwik SSG build.
docker compose -f ../docker-compose.yaml -f ../docker-compose.vps.yaml up -d
E2E_BASE_URL=http://localhost:3015 pnpm test:e2e

# VPS production (or any reachable URL). The spec validates the
# real public surface end-to-end.
E2E_BASE_URL=https://cachicamas.example.com pnpm test:e2e
```

Pre-requisites by mode:

| Mode             | Pre-requisite                                                                                     |
| ---------------- | ------------------------------------------------------------------------------------------------- |
| dev (no env var) | `docker compose up` (Postgres + Go bin) + Playwright auto-starts `pnpm dev`                       |
| container        | `docker compose -f docker-compose.yaml -f docker-compose.vps.yaml up -d` (all 5 services healthy) |
| VPS              | Reachable URL; the stack is assumed to be running on the target host                              |

## Authentication (cachicamas-github-login)

The frontend authenticates users via GitHub OAuth using
[Auth.js for Qwik](https://qwik.authjs.dev) (`@auth/qwik@0.9.2`).
The integration lives in `src/routes/plugin@auth.ts` and persists
the GitHub identity to the local `identity.user` + `identity.account`
Postgres tables via `src/lib/sign-in-callback.ts` on each successful
sign-in. PR-3 adds the Go-side JWE cookie verifier middleware.

### Prerequisites (one-time)

1. **Create a GitHub OAuth App** at
   <https://github.com/settings/developers> → "New OAuth App". Use the
   callback URL that matches your `AUTH_URL` environment variable plus
   `/auth/callback/github`. For local dev: `http://localhost:3015/auth/callback/github`.
2. **Generate `AUTH_SECRET`** — a 32-byte base64 string used to sign
   and encrypt the JWE session cookie. The same secret is consumed by
   the Go verifier middleware (PR-3) for cookie decryption.

   ```bash
   openssl rand -base64 32
   ```

3. **Populate `.env`** from `.env.example` and set the five auth vars:

   ```bash
   AUTH_GITHUB_ID=...           # GitHub OAuth App client_id
   AUTH_GITHUB_SECRET=...       # GitHub OAuth App client_secret
   AUTH_SECRET=...              # 32-byte base64 (from openssl rand)
   AUTH_TRUST_HOST=true         # required for non-Vercel/Netlify
   AUTH_URL=http://localhost:3015
   ```

### Production (real GitHub)

When `AUTH_GITHUB_BASE_URL` is **unset** (the default), the OAuth
flow points at the canonical `https://github.com/login/oauth/...`
URLs. No additional configuration is required.

### Tests (mocks-github-oauth)

For the Playwright e2e suite, point the OAuth URLs at the in-process
`mocks-github-oauth` compose service:

```bash
AUTH_GITHUB_BASE_URL=http://mocks-github-oauth:3016
AUTH_GITHUB_API_BASE_URL=http://mocks-github-oauth:3016
```

The simulator (`scripts/mocks-github-oauth/server.mjs`) exposes
`/login/oauth/authorize`, `/login/oauth/access_token`, `/user`, and
`/user/emails` endpoints that return a canned test user
(`octocat@example.com`). See `frontend/e2e/sign-in-landing.spec.ts`
and `frontend/e2e/github-sign-in.spec.ts` for the running tests.

### Architecture notes

- The session is a **stateless signed JWE cookie** — there is no
  `identity.session` table. The Go verifier (PR-3) decrypts the
  same cookie via `lestrrat-go/jwx/v2`, using HKDF-SHA256 with the
  locked envelope contract documented in `docs/adr/0002-promote-lestrrat-jwx-for-jwe.md`.

#### Identity persistence (via backend callback)

The Qwik frontend does NOT talk to Postgres directly. On each
successful GitHub sign-in, the `events.signIn` callback forwards
the Auth.js event to the database_administrator Go service via an
HMAC-signed POST:

```
POST /api/v1/identity/signin-callback
Headers:
  Content-Type: application/json
  X-Cachicamas-Timestamp: <unix_ms>
  X-Cachicamas-Signature: base64(HMAC_SHA256(IDENTITY_CALLBACK_SECRET,
                                              timestamp + "." + canonical_json))
Body:
  { "user": { "id": "...", "email": "...", "name": "...", "image": null },
    "account": { "provider": "github", "providerAccountId": "...",
                  "accessToken": "...", "refreshToken": null, ... } }
```

**Canonical JSON** (locked, cross-tooling with the Go side):
keys sorted lexicographically; no whitespace; no padding.
The algorithm is implemented in `src/lib/identity-callback-client.ts`
(TS) and `backend/database_administrator/src/interfaces/http/identity_handler.go`
(Go). A shared test oracle pins a known input → known output vector
on both sides so future drift is caught immediately.

**Anti-replay**: 5-minute window. The backend rejects timestamps
outside `|now - ts| <= 5min` with 401, and uses `crypto/subtle`
constant-time compare for the signature itself.

**Threat model**: HMAC + timestamp is sufficient because the trust
boundary is the compose internal network. If the network becomes
hostile (e.g., multi-tenant, public-facing), switch to mTLS / SPIFFE
— see ADR 0003 §"Threat model".

**Env vars**:

- `IDENTITY_CALLBACK_SECRET` — required in BOTH the `frontend` and
  `database_administrator` compose env blocks. Same value, same
  rotation cadence. Generate with `openssl rand -base64 32`.
  Different from `AUTH_SECRET` (different purpose + cadence).
- `SERVER_API_BASE_URL` — compose direct-call path
  (`http://database_administrator:8080`).
- `ORIGIN` — dev reverse-proxy fallback
  (`http://localhost:3015`).

**Failure mode**: the callback is best-effort. A successful GitHub
OAuth roundtrip is NEVER blocked by an identity persistence
failure; errors are logged + swallowed in `plugin@auth.ts`. This
posture was approved by the inline 4R review of the previous
(PR #29) slice as R4-1 [LOW] and is preserved here.

The exact pin of `@auth/qwik@0.9.2` is enforced per
`docs/adr/0001-accept-authjs-qwik.md` (pre-1.0 mitigation).
ADR 0003 documents the HMAC wire protocol in detail.

## Vulnerability scanning

`frontend/` ships three opt-in `package.json` scripts wrapping pnpm's
built-in `audit` command. None of them run as part of `verify`, `build`,
`test:ci`, or the Docker build — dependency risk is visible on demand,
not gating the default loop, and this change introduces no CI workflow.

```bash
pnpm vuln-check         # THE GATE — full tree incl. devDependencies; exits non-zero on any high/critical finding
pnpm vuln-check:prod    # informational — production dependencies only (shippable-runtime view)
pnpm vuln-check:ci      # alias of vuln-check, reserved for future CI wiring
```

`pnpm audit` is **detection-only** (it never installs, upgrades,
downgrades, or otherwise mutates `package.json` / `pnpm-lock.yaml`) and,
unlike the backend's `govulncheck`, it is **not reachability-aware**: it
flags every advisory matching an installed version range regardless of
whether the vulnerable code path is ever called from this app.

### Current baseline

Scanned with `pnpm@11.8.0 audit` on 2026-08-01. The full tree reports
**19 findings** (1 low, 5 moderate, 12 high, 1 critical). `pnpm
vuln-check` (`--audit-level=high`) enforces only the high/critical tier
— 13 findings across the 11 advisories below — and exits non-zero by
design until each is remediated or superseded. `pnpm vuln-check:prod`
(production dependencies only: `@auth/core`, `@auth/qwik`,
`@panva/hkdf`, `marked`, `postgres`, `zod`) is **clean (exit 0)** —
every finding below is rooted in `devDependencies` (eslint, vite,
vitest, and qwik-city's SVG/image build pipeline), not the shipped
runtime bundle. The 1 low + 5 moderate findings sit below the gate's
threshold and are not itemized here; they do not affect the gate's exit
status.

| #   | ID                                                                                                                                                  | Severity | Package           | Found in                              | Fixed in | Status                                                                                                                           |
| --- | --------------------------------------------------------------------------------------------------------------------------------------------------- | -------- | ----------------- | ------------------------------------- | -------- | -------------------------------------------------------------------------------------------------------------------------------- |
| 1   | [GHSA-5xrq-8626-4rwp](https://github.com/advisories/GHSA-5xrq-8626-4rwp)                                                                            | Critical | `vitest`          | 0.34.6                                | 3.2.6    | Accepted debt — `0.34 → 3.2.6` is a multi-major jump deferred to its own change; dev-only (`vitest`/`@vitest/ui`), never shipped |
| 2   | [GHSA-v2wj-q39q-566r](https://github.com/advisories/GHSA-v2wj-q39q-566r)                                                                            | High     | `vite`            | 7.3.1                                 | 7.3.2    | Accepted debt — dev-only build tooling                                                                                           |
| 3   | [GHSA-p9ff-h696-f583](https://github.com/advisories/GHSA-p9ff-h696-f583)                                                                            | High     | `vite`            | 7.3.1                                 | 7.3.2    | Accepted debt — dev-only build tooling                                                                                           |
| 4   | [GHSA-fx2h-pf6j-xcff](https://github.com/advisories/GHSA-fx2h-pf6j-xcff)                                                                            | High     | `vite`            | 5.4.21 (nested under `vitest@0.34.6`) | 6.4.3    | Accepted debt — resolves once the `vitest` major bump (row 1) lands                                                              |
| 5   | [GHSA-fx2h-pf6j-xcff](https://github.com/advisories/GHSA-fx2h-pf6j-xcff)                                                                            | High     | `vite`            | 7.3.1                                 | 7.3.5    | Accepted debt — dev-only build tooling                                                                                           |
| 6   | [GHSA-3jxr-9vmj-r5cp](https://github.com/advisories/GHSA-3jxr-9vmj-r5cp) + [GHSA-mh99-v99m-4gvg](https://github.com/advisories/GHSA-mh99-v99m-4gvg) | High     | `brace-expansion` | 1.1.15                                | 1.1.17   | Accepted debt — transitive via `eslint`/`stylus`/`glob`/`minimatch`, dev-only                                                    |
| 7   | [GHSA-3jxr-9vmj-r5cp](https://github.com/advisories/GHSA-3jxr-9vmj-r5cp) + [GHSA-mh99-v99m-4gvg](https://github.com/advisories/GHSA-mh99-v99m-4gvg) | High     | `brace-expansion` | 2.1.1                                 | 2.1.3    | Accepted debt — transitive via `typescript-eslint`, dev-only                                                                     |
| 8   | [GHSA-mh99-v99m-4gvg](https://github.com/advisories/GHSA-mh99-v99m-4gvg)                                                                            | High     | `brace-expansion` | 5.0.7                                 | 5.0.8    | Accepted debt — transitive via `eslint-plugin-qwik`, dev-only                                                                    |
| 9   | [GHSA-2p49-hgcm-8545](https://github.com/advisories/GHSA-2p49-hgcm-8545)                                                                            | High     | `svgo`            | 3.3.3                                 | 3.3.4    | Accepted debt — transitive via `@builder.io/qwik-city`, dev-only                                                                 |
| 10  | [GHSA-f88m-g3jw-g9cj](https://github.com/advisories/GHSA-f88m-g3jw-g9cj)                                                                            | High     | `sharp`           | 0.34.5                                | 0.35.0   | Accepted debt — transitive via `qwik-city`'s `vite-imagetools`, dev-only                                                         |
| 11  | [GHSA-r28c-9q8g-f849](https://github.com/advisories/GHSA-r28c-9q8g-f849)                                                                            | High     | `postcss`         | 8.5.16                                | 8.5.18   | Accepted debt — transitive via `vite`, dev-only                                                                                  |

The production auth-bypass finding that originally motivated this gate
(`@auth/core` GHSA-7rqj-j65f-68wh) is **not** in the table above — it
was remediated before this snapshot was taken (see Remediation history).
The gate is wired to fail on any high/critical finding so this baseline
is enforced rather than silently carried.

### Remediation history

**2026-08-01** — `@auth/core` bumped from `0.41.2` to `0.41.3`, closing
**GHSA-7rqj-j65f-68wh** (homoglyph email-normalization auth bypass) on
the application's only login path. `@auth/qwik@0.9.2`
(`docs/adr/0001-accept-authjs-qwik.md`) pins `@auth/core@0.41.2` as a
regular (non-peer) dependency, so the direct bump alone would have left
a second vulnerable copy resolved under `@auth/qwik`;
`frontend/pnpm-workspace.yaml` now carries
`overrides: { "@auth/core": "0.41.3" }` to force a single tree-wide
resolution (confirmed via `pnpm why @auth/core`). The `@auth/qwik@0.9.2`
pin itself is unchanged. Verification: `pnpm install` resolved cleanly;
`pnpm lint`, `pnpm build.types`, and `pnpm test:ci` (647 tests) passed.
The mocks-backed OAuth e2e specs (`github-sign-in`, `sign-in-denied`,
`sign-in-cookie-attrs`, `sign-out`) self-skip without a configured
`mocks-github-oauth` stack / `AUTH_GITHUB_BASE_URL` and were not
exercised in this environment; `sign-in-landing` ran against a local
dev server with no regression attributable to the bump (its one
failure is a pre-existing, unrelated button-copy assertion that
reproduces identically against `@auth/core@0.41.2` on the base tree).
`pnpm vuln-check:prod` is clean (exit 0) as a result of this bump.
