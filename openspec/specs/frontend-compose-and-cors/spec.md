# frontend-compose-and-cors Specification

> **Domain**: frontend-compose-and-cors
> **Change**: cachicamas-frontend-dockerize
> **Type**: New capability (compose override + CORS contract)
> **Created**: 2026-07-03
> **Persistence**: hybrid (this file + engram `sdd/cachicamas-frontend-dockerize/spec/frontend-compose-and-cors`)

## Purpose

Defines the contract for (a) the docker-compose override file that produces a VPS-deployable profile, and (b) the CORS contract that the Go binary exposes when running in that profile. The override file MUST isolate all internal services (postgres, jaeger, otel-collector, database_administrator) from the host's public network interface, leaving only the `frontend` service publishing a port. The CORS contract MUST be configurable via the `CORS_ALLOW_ORIGINS` environment variable so that operators can opt in to cross-origin requests when the deployment topology requires it (e.g., a separate API gateway or a developer using `curl` from a different origin).

## Requirements

### Requirement: Compose override file isolates internal services from the host

The `docker-compose.vps.yaml` file MUST override the base `docker-compose.yaml` such that, when both files are used together, the `postgres`, `jaeger`, `otel-collector`, and `database_administrator` services do NOT publish any host port. Only the `frontend` service MUST publish a port (default: `${FRONTEND_PORT:-3015}:80`).

#### Scenario: Override file composes with the base file

- GIVEN `docker-compose.yaml` (the base file) and `docker-compose.vps.yaml` (the override file)
- WHEN `docker compose -f docker-compose.yaml -f docker-compose.vps.yaml config` is executed
- THEN the command MUST exit with code 0
- AND the rendered config MUST contain the `frontend` service with the port mapping
- AND the rendered config MUST NOT contain a port mapping for `postgres`
- AND the rendered config MUST NOT contain a port mapping for `jaeger`
- AND the rendered config MUST NOT contain a port mapping for `otel-collector`
- AND the rendered config MUST NOT contain a port mapping for `database_administrator`

#### Scenario: Internal services are reachable from frontend on the compose network

- GIVEN the override-composed stack is up
- WHEN `docker compose -f docker-compose.yaml -f docker-compose.vps.yaml exec frontend wget -q -O- http://database_administrator:8080/health` is executed
- THEN the response MUST be `{"status":"ok"}` (the Go binary's health endpoint)
- AND `wget` MUST succeed without any port mapping (DNS-based service discovery on `cachicamas_network`)

#### Scenario: Base file alone still works for local dev

- GIVEN only `docker-compose.yaml` is used (no override)
- WHEN `docker compose up -d --build` is executed
- THEN the `frontend` service MUST publish its port
- AND the `database_administrator` service MUST publish its port (8080:8080, as today)
- AND the `postgres` service MUST publish its port (5432:5432, as today)
- AND the `jaeger` service MUST publish its port (16686:16686, as today)
- AND the dev loop on the host is unchanged

### Requirement: The frontend is the only service publishing a port to the host

When the override file is applied, ONLY the `frontend` service SHALL have a `ports:` mapping in the rendered compose config.

#### Scenario: All non-frontend services have no `ports:` section

- GIVEN the override-composed config is rendered
- WHEN the rendered YAML is inspected
- THEN for each service in `[postgres, jaeger, otel-collector, database_administrator]`, the service MUST have either no `ports:` key or an empty `ports:` list

#### Scenario: Only the frontend service has a `ports:` section

- GIVEN the same rendered config
- WHEN the `frontend` service is inspected
- THEN its `ports:` list MUST contain exactly one entry: `${FRONTEND_PORT:-3015}:80`
- AND the entry MUST be a string (not a complex mapping), compatible with compose v2

### Requirement: The override file sets the production service environment

The override file MUST set `SERVICE_ENV=production` on the `database_administrator` service so that the Go binary's CORS default (which is "production = off") is honored. Operators MUST be able to opt back in to CORS via `CORS_ALLOW_ORIGINS` without modifying code.

#### Scenario: SERVICE_ENV=production in the override

- GIVEN the override file is applied
- WHEN the rendered config is inspected
- THEN the `database_administrator` service's `environment` MUST include `SERVICE_ENV: production`

#### Scenario: CORS_ALLOW_ORIGINS in the override

- GIVEN the override file is applied
- WHEN the rendered config is inspected
- THEN the `database_administrator` service's `environment` MUST include `CORS_ALLOW_ORIGINS: ${CORS_ALLOW_ORIGINS:-http://localhost:3015}`

### Requirement: CORS middleware respects the CORS_ALLOW_ORIGINS environment variable

When the Go binary is started with `CORS_ALLOW_ORIGINS` set to a comma-separated list of origins, requests whose `Origin` header matches one of those origins MUST receive the corresponding `Access-Control-Allow-Origin` response header. Requests whose `Origin` is not in the list MUST NOT receive the header (and the browser will block them client-side, as is correct CORS behavior).

#### Scenario: Allowlisted origin receives CORS headers

- GIVEN `CORS_ALLOW_ORIGINS=http://localhost:3015` is set in the Go binary's environment
- WHEN a request is sent with `Origin: http://localhost:3015` and `Access-Control-Request-Method: POST`
- THEN the response MUST include `Access-Control-Allow-Origin: http://localhost:3015`
- AND the response MUST include `Access-Control-Allow-Methods: GET, POST, OPTIONS`
- AND the response MUST include `Access-Control-Allow-Headers: Content-Type`
- AND the response MUST include `Vary: Origin`

#### Scenario: Non-allowlisted origin does not receive CORS headers

- GIVEN the same `CORS_ALLOW_ORIGINS` setting
- WHEN a request is sent with `Origin: http://evil.example.com`
- THEN the response MUST NOT include `Access-Control-Allow-Origin`
- AND the browser will block the response client-side (this is the correct CORS behavior)

#### Scenario: Preflight OPTIONS request returns 204

- GIVEN the same setting
- WHEN a preflight `OPTIONS` request is sent with `Origin: http://localhost:3015` and `Access-Control-Request-Method: POST`
- THEN the response MUST have HTTP status 204 (No Content)
- AND the response MUST include `Access-Control-Allow-Origin: http://localhost:3015`
- AND the response MUST NOT invoke the downstream handler (verified by the absence of the actual handler's response body)

#### Scenario: CORS_ALLOW_ORIGINS unset means CORS is off (production default)

- GIVEN `CORS_ALLOW_ORIGINS` is NOT set
- AND `SERVICE_ENV=production` is set
- WHEN a request is sent with any `Origin` header
- THEN the response MUST NOT include `Access-Control-Allow-Origin`
- AND the browser will block any cross-origin request

#### Scenario: Multiple origins (comma-separated)

- GIVEN `CORS_ALLOW_ORIGINS=http://localhost:3015,https://cachicamas.example.com` is set
- WHEN a request is sent with `Origin: https://cachicamas.example.com`
- THEN the response MUST include `Access-Control-Allow-Origin: https://cachicamas.example.com`
- AND a request with `Origin: http://localhost:3015` MUST also receive the matching header

### Requirement: CORS middleware is a passthrough for same-origin requests

The CORS middleware MUST NOT add `Access-Control-Allow-Origin` to responses when the request does not include an `Origin` header (i.e., same-origin or non-browser requests). This avoids polluting the response headers with CORS noise for non-CORS callers.

#### Scenario: Request without Origin header is untouched

- GIVEN `CORS_ALLOW_ORIGINS=http://localhost:3015` is set
- WHEN a request is sent WITHOUT an `Origin` header
- THEN the response MUST NOT include `Access-Control-Allow-Origin`
- AND the downstream handler MUST be invoked normally

## Result

- **Status**: ok
- **Executive summary**: 5 capabilities (compose override, frontend-only exposure, production service env, CORS contract, same-origin passthrough). 13 independently-verifiable scenarios.
- **Artifacts**: `openspec/changes/cachicamas-frontend-dockerize/specs/frontend-compose-and-cors/spec.md` (this file), engram topic `sdd/cachicamas-frontend-dockerize/spec/frontend-compose-and-cors`.
- **Next recommended**: `sdd-design` for the compose override file structure and the CORS configuration diff.
- **Skill resolution**: `paths-injected`.
