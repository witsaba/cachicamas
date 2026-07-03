# frontend-e2e-and-client-data Specification

> **Domain**: frontend-e2e-and-client-data
> **Change**: cachicamas-frontend-dockerize
> **Type**: Modified capability (existing behavior changes from server-side to client-side data loading; e2e runner gains an env-driven mode)
> **Created**: 2026-07-03
> **Persistence**: hybrid (this file + engram `sdd/cachicamas-frontend-dockerize/spec/frontend-e2e-and-client-data`)

## Purpose

Defines two related contracts that change as a consequence of moving the frontend to Static SSG:

1. **Client-side data loading for the Organizations list and readback routes**: the existing `routeLoader$` server-side fetchers MUST be replaced with client-side fetchers (`useVisibleTask$` + `useSignal`) because the static adapter does not run loaders at runtime — it captures their output at build time. The functional outcome (the list is shown, the readback shows the org) MUST be preserved, including the "offline → empty + error banner" behavior and the "load + render" sequence.
2. **Dual-mode e2e runner**: Playwright's `baseURL` and `webServer` block MUST be env-driven via `E2E_BASE_URL`, so that the same spec runs both against the dev server (current behavior, for local dev) and against the dockerized stack (for CI and post-deploy validation).

## Requirements

### Requirement: Organizations list loads its data in the browser

The `/organizations` route MUST fetch the list of organizations from the Go binary on the client, after the page is hydrated. The fetched data MUST be displayed in the `<OrganizationList>` component. On fetch failure, the page MUST render an empty list and a top-level alert with the offline or error message, matching the existing contract (the e2e spec relies on this).

#### Scenario: Successful fetch renders the list

- GIVEN the user navigates to `/organizations`
- AND the Go binary returns 200 with an array of organizations
- WHEN the page hydrates
- THEN the page MUST issue a `GET /api/organizations` request to the same origin
- AND after the fetch completes, the page MUST render one `<article data-organization>` per organization

#### Scenario: Empty list renders the empty state

- GIVEN the user navigates to `/organizations`
- AND the Go binary returns 200 with `[]`
- WHEN the page hydrates
- THEN the page MUST render the empty state component (the existing "no organizations" copy)
- AND no error banner MUST be shown

#### Scenario: Offline / network failure renders an error banner

- GIVEN the user navigates to `/organizations`
- AND the Go binary is unreachable (connection refused, DNS failure, or timeout)
- WHEN the fetch fails
- THEN the page MUST render the empty list (NOT crash, NOT show a stale list)
- AND the page MUST render the alert with `data-organization-list-error` and a message that includes the API base URL

#### Scenario: HTTP 5xx renders an error banner

- GIVEN the user navigates to `/organizations`
- AND the Go binary returns 500
- WHEN the response is received
- THEN the page MUST render the empty list
- AND the page MUST render the alert with the server's error message

#### Scenario: routeLoader$ is not used

- GIVEN the refactor described in `design.md` is applied
- WHEN the source of `frontend/src/routes/organizations/index.tsx` is inspected
- THEN the file MUST NOT import `routeLoader$` from `@builder.io/qwik-city`
- AND the file MUST NOT export a `useXxxLoader` value

### Requirement: Organizations readback loads its data in the browser

The `/organizations/{id}` route MUST fetch the single organization from the Go binary on the client, using the `id` from the URL. On success, the page MUST render the `<OrganizationReadback>` component. On failure (404 or offline), the page MUST render an error alert with a "Back to organizations" link, matching the existing contract.

#### Scenario: Successful fetch renders the readback

- GIVEN the user navigates to `/organizations/123`
- AND the Go binary returns 200 with the organization
- WHEN the page hydrates
- THEN the page MUST issue a `GET /api/organizations/123` request
- AND the page MUST render the readback component with the org's data

#### Scenario: 404 renders the not-found alert

- GIVEN the user navigates to `/organizations/999999`
- AND the Go binary returns 404
- WHEN the response is received
- THEN the page MUST render the alert with `data-organization-error`
- AND the page MUST render the "Back to organizations" link
- AND the readback component MUST NOT be rendered

#### Scenario: Offline / network failure renders an alert

- GIVEN the user navigates to `/organizations/123`
- AND the Go binary is unreachable
- WHEN the fetch fails
- THEN the page MUST render the alert with the offline message
- AND the page MUST render the "Back to organizations" link

#### Scenario: Invalid id in the URL

- GIVEN the user navigates to `/organizations/abc` (non-numeric id)
- WHEN the page hydrates
- THEN the page MUST treat the id as invalid and render the "Organization not found." alert
- AND no fetch to the Go binary MUST be issued (the validation happens client-side, before the fetch)

#### Scenario: routeLoader$ is not used

- GIVEN the refactor is applied
- WHEN the source of `frontend/src/routes/organizations/[id]/index.tsx` is inspected
- THEN the file MUST NOT import `routeLoader$` from `@builder.io/qwik-city`
- AND the file MUST NOT export a `useXxxLoader` value

### Requirement: Form submit on /organizations/new still works

The `/organizations/new` route MUST continue to use the existing client-side submit action (a `$()` QRL wrapping `createOrganization` from `~/lib/api`). The form MUST submit a `POST /api/organizations` request and MUST navigate to `/organizations/{id}` on success. **No refactor is required for this route** because the action already runs in the browser.

#### Scenario: Successful submit navigates to the readback

- GIVEN the user is on `/organizations/new`
- AND fills in the required fields
- AND clicks the submit button
- WHEN the Go binary returns 201 with the new organization
- THEN the browser MUST navigate to `/organizations/{id}` (the new org's id)
- AND the readback page MUST render the new org's data (per the previous requirement)

#### Scenario: 409 conflict renders the inline error

- GIVEN the user is on `/organizations/new`
- AND enters a `full_name` whose auto-derived `identification` is already taken
- WHEN the Go binary returns 409
- THEN the form MUST render the inline slug-taken message below the `identification` input
- AND the form MUST NOT navigate

### Requirement: Playwright e2e runner supports two modes

The `frontend/playwright.config.ts` MUST read `E2E_BASE_URL` from the environment. When unset, the runner MUST use the existing local dev behavior (baseURL `http://localhost:5173`, `webServer` block starting `pnpm dev`). When set, the runner MUST use the supplied URL as `baseURL` and MUST NOT start a `webServer` (the caller is responsible for ensuring the stack is already running).

#### Scenario: Default mode starts the dev server

- GIVEN `E2E_BASE_URL` is unset
- WHEN `pnpm test:e2e` is executed
- THEN the runner MUST set `use.baseURL` to `http://localhost:5173`
- AND the runner MUST configure `webServer` with `command: "pnpm dev"` and `url: "http://localhost:5173"`
- AND the spec MUST pass on a fresh checkout (no other process required)

#### Scenario: Container mode uses the supplied baseURL

- GIVEN `E2E_BASE_URL=http://localhost:3015` is set
- AND the docker-compose stack is up (frontend + Go + Postgres)
- WHEN `pnpm test:e2e` is executed
- THEN the runner MUST set `use.baseURL` to `http://localhost:3015`
- AND the runner MUST NOT configure a `webServer` (i.e., `webServer` is `undefined` or absent)
- AND the spec MUST pass without Playwright starting any process

#### Scenario: VPS mode uses the public domain

- GIVEN `E2E_BASE_URL=https://cachicamas.example.com` is set
- AND the VPS stack is up and reachable
- WHEN `pnpm test:e2e` is executed
- THEN the runner MUST set `use.baseURL` to `https://cachicamas.example.com`
- AND the spec MUST pass against the production URL

#### Scenario: Misuse is caught with a clear error

- GIVEN `E2E_BASE_URL` is set to an unreachable URL (e.g., `http://localhost:9999`)
- AND no stack is running on that port
- WHEN `pnpm test:e2e` is executed
- THEN the spec MUST fail with a clear connection error (NOT a Playwright webServer timeout)
- AND the error message MUST be actionable (e.g., "Could not connect to <http://localhost:9999>. Is docker compose up?")

### Requirement: e2e spec waits for client-side data on navigation to readback

The `frontend/e2e/create-organization.spec.ts` MUST wait for the client-side data fetch to complete after navigating to `/organizations/{id}` (the readback page), so that the assertions on the readback content are deterministic.

#### Scenario: Readback content is asserted after the client fetch

- GIVEN the e2e spec has just submitted the create form
- AND the browser has navigated to `/organizations/{id}`
- WHEN the test asserts the readback content
- THEN the test MUST wait for the client-side data fetch to complete (via `page.waitForLoadState("networkidle")` or an equivalent `expect(...).toBeVisible({ timeout })` pattern)
- AND the assertion MUST NOT race the client fetch (no flaky fails on a fast machine or a slow CI)

## Result

- **Status**: ok
- **Executive summary**: 6 capabilities (org list, org readback, form submit unchanged, e2e dual mode, e2e error handling, e2e waitForLoadState). 18 independently-verifiable scenarios.
- **Artifacts**: `openspec/changes/cachicamas-frontend-dockerize/specs/frontend-e2e-and-client-data/spec.md` (this file), engram topic `sdd/cachicamas-frontend-dockerize/spec/frontend-e2e-and-client-data`.
- **Next recommended**: `sdd-design` for the refactor diff and the playwright config patch.
- **Skill resolution**: `paths-injected`.
