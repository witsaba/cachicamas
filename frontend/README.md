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
