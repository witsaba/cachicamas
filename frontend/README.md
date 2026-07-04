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

| Route                 | Purpose                                                                                                                              |
| --------------------- | ------------------------------------------------------------------------------------------------------------------------------------ |
| `/`                   | Brand mark + tagline + a single "Get started" CTA to `/organizations`.                                                               |
| `/organizations`      | List of organizations. Empty state is an instruction (no image, no illustration).                                                    |
| `/organizations/new`  | Create-organization form. Auto-derives the slug from the full name; user can override; the review field group renders progressively. |
| `/organizations/{id}` | Read-back of a single organization. The URL is the breadcrumb.                                                                       |

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
- The `signIn` callback (in `lib/sign-in-callback.ts`) implements
  **auto-link-on-email-match**: if a row with the same email already
  exists, a new `identity.account` row is attached to it instead of
  creating a duplicate `identity.user`.
- The exact pin of `@auth/qwik@0.9.2` is enforced per
  `docs/adr/0001-accept-authjs-qwik.md` (pre-1.0 mitigation).
