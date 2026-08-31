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

### Tests

The frontend uses Vitest with `createDOM()` from
`@builder.io/qwik/testing`. Run:

```shell
pnpm test:ci
```

The runner discovers every `*.spec.{ts,tsx}` file under
`src/`. The trimmed marketing surface (landing + chrome +
hero-proof + pricing) is locked by 179 tests.

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
(production dependencies only: `isomorphic-dompurify`,
`marked`, `zod`) is **clean (exit 0)** —
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
was closed by removal on 2026-08-31 (see Remediation history).
The gate is wired to fail on any high/critical finding so this baseline
is enforced rather than silently carried.

### Remediation history

**2026-08-31** — `@auth/core`, `@auth/qwik`, `@panva/hkdf`, `postgres`,
`@playwright/test` and the auth-shaped mocks service are all **removed**
from the frontend tree by `clean-frontend-keep-only-landing-page`. The
GHSA-7rqj-j65f-68wh finding is closed by removal, not by version
bump; the `overrides["@auth/core"]` block in `pnpm-workspace.yaml` is
gone and `pnpm install` no longer fetches any `@auth/*` package.
