# frontend-runtime Specification

> **Domain**: frontend-runtime
> **Change**: cachicamas-frontend-dockerize
> **Type**: New capability (full spec — no existing behavior, except for the underlying Qwik app which already exists)
> **Created**: 2026-07-03
> **Persistence**: hybrid (this file + engram `sdd/cachicamas-frontend-dockerize/spec/frontend-runtime`)

## Purpose

Defines the runtime contract for the cachicamas frontend served from Docker. The frontend MUST be packaged as a static single-page application (Qwik City with the static adapter), built inside a multi-stage Dockerfile, and served by an nginx container on port 80 inside the `cachicamas_network`. nginx MUST apply SPA fallback for client-side routing, MUST proxy `/api/*` to the Go binary inside the compose network, and MUST gzip and cache the static assets per Qwik conventions.

## Requirements

### Requirement: Multi-stage Docker build with pinned base images

The `frontend/Dockerfile` MUST build the Qwik app in a `node:20-alpine` builder stage and copy the resulting static artifacts into a `nginx:1.27-alpine` runner stage. The final image MUST be reproducible and MUST NOT contain Node.js, pnpm, or source files in the runner stage.

#### Scenario: Final image contains the built static assets

- GIVEN a `frontend/Dockerfile` with the multi-stage build described in `design.md`
- WHEN `docker compose build frontend` is run
- THEN the resulting image MUST contain `/usr/share/nginx/html/index.html`
- AND the image MUST contain the Qwik client chunks under `/usr/share/nginx/html/build/`
- AND the image MUST contain `/etc/nginx/conf.d/default.conf`

#### Scenario: Final image does not contain Node or pnpm

- GIVEN the same Dockerfile
- WHEN the build completes
- THEN `docker run --rm <image> which node` MUST exit non-zero (no Node binary in the runner)
- AND `docker run --rm <image> which pnpm` MUST exit non-zero (no pnpm in the runner)
- AND the runner image MUST be based on `nginx:1.27-alpine` (verifiable via `docker inspect` → `Config.Image`)

#### Scenario: Final image size is under 50 MB

- GIVEN the multi-stage build
- WHEN `docker images cachicamas/frontend:local` is run
- THEN the reported virtual size of the `frontend` service image MUST be less than 50 MB

### Requirement: Static prerender of static routes

The Qwik static adapter MUST prerender the routes that have no dynamic data at build time, producing one HTML file per route that the browser can load directly.

#### Scenario: Static routes are prerendered as HTML

- GIVEN the Qwik static adapter is registered in `frontend/vite.config.ts`
- WHEN `pnpm build` runs in the builder stage
- THEN the build output (`dist/`) MUST contain `index.html` (for `/`)
- AND the output MUST contain `organizations/new/index.html` (for `/organizations/new`)
- AND the output MUST contain `organizations/index.html` (for `/organizations`)

#### Scenario: Dynamic routes are not prerendered

- GIVEN `/organizations/{id}` has no `onStaticGenerate` hook (or the hook returns an empty list)
- WHEN `pnpm build` runs
- THEN the output MUST NOT contain a `organizations/[id]/index.html` file
- AND the route MUST rely on SPA fallback at runtime (see next requirement)

### Requirement: SPA fallback via nginx

nginx MUST serve `index.html` for any request that does not match a static file in `/usr/share/nginx/html/`. This enables client-side routing for dynamic routes that cannot be prerendered.

#### Scenario: Unknown route returns the SPA shell

- GIVEN the frontend container is running and healthy
- WHEN `curl -fsS http://localhost:3015/organizations/123` is executed
- THEN the response MUST have HTTP status 200
- AND the response body MUST contain the brand `<h1>` element from the landing page
- AND the response MUST be HTML (Content-Type starting with `text/html`)

#### Scenario: Known static asset is served directly

- GIVEN a Qwik client chunk exists at `dist/build/q-abc123.js`
- WHEN `curl -fsS http://localhost:3015/build/q-abc123.js` is executed
- THEN the response MUST have HTTP status 200
- AND the response body MUST be the JavaScript source
- AND the `Cache-Control` header MUST include `max-age=31536000` (1 year) and `immutable`

#### Scenario: Index shell is not cached aggressively

- GIVEN a request to `http://localhost:3015/` (or any SPA fallback path)
- WHEN the response is returned
- THEN the `Cache-Control` header MUST include `no-cache`
- AND the response MUST NOT include `immutable`

### Requirement: Reverse proxy for /api/* to the Go binary

nginx MUST forward requests matching `/api/*` to the `database_administrator` service inside the compose network, preserving the original HTTP method, query string, and request body. The Go binary MUST see the path WITHOUT the `/api` prefix (i.e., `/api/organizations` → `/organizations` on the Go side).

#### Scenario: Browser call to /api/organizations reaches the Go binary

- GIVEN the frontend container is running and the Go binary is healthy
- WHEN `curl -fsS -X POST -H "Content-Type: application/x-www-form-urlencoded" -d "full_name=Test&identification=test" http://localhost:3015/api/organizations` is executed
- THEN the response MUST be the same as a direct `curl -X POST ... http://localhost:8080/organizations` to the Go binary
- AND the response MUST have HTTP status 201 (created)

#### Scenario: Browser call to /api/organizations/123 reaches the Go binary

- GIVEN a previously created organization with id 1
- WHEN `curl -fsS http://localhost:3015/api/organizations/1` is executed
- THEN the response MUST be the JSON representation of the organization
- AND the response MUST be equivalent to a direct call to `http://localhost:8080/organizations/1`

#### Scenario: Go binary sees paths without the /api prefix

- GIVEN the proxy configuration strips the `/api` prefix
- WHEN the Go binary logs the request (via OTel spans or access log)
- THEN the path recorded MUST be `/organizations` (NOT `/api/organizations`)
- AND the host recorded MUST be `database_administrator:8080` (the internal compose DNS)

### Requirement: Gzip compression for text assets

nginx MUST compress text-based responses with gzip, and MUST advertise this capability via the `Vary: Accept-Encoding` response header.

#### Scenario: Compressed text response

- GIVEN a request to a text asset (e.g., the JavaScript chunks under `/build/`)
- WHEN the request is sent with `Accept-Encoding: gzip`
- THEN the response MUST have `Content-Encoding: gzip`
- AND the response MUST have `Vary: Accept-Encoding`
- AND the decompressed body MUST equal the original source

#### Scenario: Pre-compressed content is not double-compressed

- GIVEN Qwik pre-compresses assets to `.gz` during build
- WHEN nginx serves the pre-compressed file
- THEN the response MUST use the pre-compressed body (no on-the-fly recompression)
- AND the response MUST have `Content-Encoding: gzip`

### Requirement: Cache headers for Qwik client chunks

nginx MUST serve Qwik client chunks (under `/build/`) with a long-lived `Cache-Control: public, max-age=31536000, immutable` header, since Qwik content-hashes chunk filenames (e.g., `q-abc123.js`) and a new filename is generated whenever the content changes.

#### Scenario: Client chunk cache header

- GIVEN a Qwik client chunk at `/build/q-abc123.js`
- WHEN the chunk is requested
- THEN the `Cache-Control` header MUST be `public, max-age=31536000, immutable`

#### Scenario: Index shell is never cached aggressively

- GIVEN a request to `/index.html` (the SPA shell)
- WHEN the request is returned
- THEN the `Cache-Control` header MUST include `no-cache` or `no-store`
- AND the response MUST NOT be cached for longer than 5 minutes

### Requirement: Healthcheck on the frontend container

The `frontend` service in `docker-compose.yaml` MUST have a `healthcheck` that verifies nginx is serving content. The healthcheck MUST use `wget --spider` (busybox in `nginx:1.27-alpine`) and MUST NOT require a shell, curl, or a third-party tool.

#### Scenario: Healthcheck reports healthy when nginx is up

- GIVEN the frontend container is running with `wget --spider` healthcheck
- WHEN `docker compose ps frontend` is executed
- THEN the `Status` column MUST include `(healthy)` within 30 seconds of startup

#### Scenario: Healthcheck reports unhealthy when nginx is stopped

- GIVEN the frontend container is running and healthy
- WHEN `docker compose exec frontend nginx -s stop` is executed
- THEN within `interval * retries` seconds, `docker compose ps frontend` MUST report `(unhealthy)` or `Exit`

#### Scenario: Healthcheck uses wget, not curl

- GIVEN the `nginx:1.27-alpine` base image
- WHEN the healthcheck is executed
- THEN the test command MUST be `["CMD", "wget", "--spider", "-q", "http://127.0.0.1/"]` (busybox wget)
- AND the image MUST NOT be rebuilt to include `curl` just to support the healthcheck

## Result

- **Status**: ok
- **Executive summary**: 6 capabilities covered (multi-stage build, prerender, SPA fallback, /api reverse proxy, gzip, cache headers, healthcheck). 14 independently-verifiable scenarios.
- **Artifacts**: `openspec/changes/cachicamas-frontend-dockerize/specs/frontend-runtime/spec.md` (this file), engram topic `sdd/cachicamas-frontend-dockerize/spec/frontend-runtime`.
- **Next recommended**: `sdd-design` for the implementation details of these requirements.
- **Skill resolution**: `paths-injected` (spec was written inline by the parent orchestrator after the sdd-* agent runtime was found to not expose file tools; the explore + proposal artifacts are the inputs and were honored).
