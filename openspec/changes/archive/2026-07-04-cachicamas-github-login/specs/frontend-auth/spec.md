# frontend-auth Specification

> **Domain**: frontend-auth
> **Change**: cachicamas-github-login
> **Type**: New capability (full spec — no existing auth in the Qwik app)
> **Created**: 2026-07-03
> **Persistence**: hybrid (this file + Engram `sdd/cachicamas-github-login/spec/frontend-auth`)

## Purpose

Defines the frontend authentication behavior for cachicamas. A user MUST be
able to authenticate via GitHub OAuth from the Qwik app, and an authenticated
session MUST survive page reloads, container restarts within the cookie's
lifetime, and the same-origin `/api/*` reverse proxy in the Qwik Node server.
The system MUST persist GitHub identity data server-side in the `identity.*`
Postgres schema so that subsequent visits map the GitHub user to the same
`identity.user` row and can return a server-known profile to the `/profile`
route.

The slice is intentionally GitHub-only (other providers are a follow-up), uses
the official Auth.js for Qwik integration (`@auth/qwik`), and treats the
session as a stateless signed JWE cookie (no `identity.session` table).

## Glossary

| Term | Meaning |
| ------ | --------- |
| **Auth.js** | The third-party authentication library this slice integrates. |
| **@auth/qwik** | Auth.js' Qwik adapter (post-rebrand from `@builder.io/qwik-auth`). |
| **GitHub provider** | The OAuth provider implemented by `import GitHub from "@auth/qwik/providers/github"`. |
| **JWE cookie** | The Auth.js `jwt` strategy session, serialized as a signed/encrypted JWE and stored in the `authjs.session-token` cookie on the Qwik origin. |
| **`AUTH_SECRET`** | The 32-byte base64 secret used to sign/encrypt the JWE. MUST be injected via `docker-compose.yaml` from the host environment, never committed. |
| **callback URL** | The URL GitHub redirects the user to after the OAuth flow. MUST match what is registered on the GitHub OAuth App. |
| **PKCE** | Proof Key for Code Exchange; an OAuth extension that protects against authorization-code interception. Optional for confidential clients, ON in this slice. |
| **CSRF** | Cross-Site Request Forgery. Auth.js for Qwik uses a double-submit cookie; this spec does not test the Auth.js internal CSRF but asserts that `onRequest` is wired so it remains active. |

---

## Capability: Frontend OAuth roundtrip via @auth/qwik

### Requirement: R-FA-001 The Qwik app exposes the GitHub OAuth roundtrip routes

The Qwik app MUST export an `onRequest` middleware (the canonical Auth.js for
Qwik wiring) that handles `/auth/signin`, `/auth/callback/github`, and
`/auth/signout` as defined by the Auth.js for Qwik contract. The
`routes/plugin@auth.ts` file MUST exist and MUST call `QwikAuth$` with a
`providers: [GitHub({ ... })]` array and the required env bindings
(`AUTH_GITHUB_ID`, `AUTH_GITHUB_SECRET`, `AUTH_SECRET`).

#### Scenario: S-FA-001 Routes plugin file exists

- GIVEN the change is applied
- WHEN the orchestrator inspects `frontend/src/routes/`
- THEN `plugin@auth.ts` SHALL exist
- AND the file SHALL call `QwikAuth$(({ env }) => ({ providers: [GitHub(...)] }))`
- AND the file SHALL export at least `onRequest`, `useSession`, `useSignIn`, and `useSignOut`

Verification: `test -f frontend/src/routes/plugin@auth.ts && grep -c "QwikAuth\\\$" frontend/src/routes/plugin@auth.ts` returns at least 1.

#### Scenario: S-FA-002 GitHub provider is registered

- GIVEN `plugin@auth.ts` exists
- WHEN the orchestrator inspects the file content
- THEN the `providers` array SHALL contain exactly one entry: `GitHub`
- AND no other provider SHALL be registered in this slice (Google, Microsoft, generic OIDC are deferred)

Verification: `grep -E "import GitHub from" frontend/src/routes/plugin@auth.ts` returns one line; `grep -cE "GoogleProvider|Google\\(" frontend/src/routes/plugin@auth.ts` returns 0.

### Requirement: R-FA-002 The landing page shows a "Sign in with GitHub" affordance

The `/` route MUST render a visible sign-in button when the visitor is not
authenticated. When clicked, the button MUST start the Auth.js GitHub flow via
the `useSignIn` action with `providerId: 'github'`.

#### Scenario: S-FA-010 Unauthenticated landing shows sign-in button

- GIVEN the visitor's request to `/` carries no `authjs.session-token` cookie
- WHEN the route loader runs
- THEN the rendered HTML SHALL contain a `<form>` whose `action` is the
      `useSignIn` action endpoint AND whose body contains a hidden input
      `providerId=github`
- AND the rendered HTML SHALL contain visible text "Sign in with GitHub"
      (case-insensitive substring match)

Verification: `pnpm test:e2e -- --grep "sign-in landing"` (Playwright spec
`e2e/sign-in-landing.spec.ts`) fills the form and asserts `await page.getByRole("button", { name: /sign in with github/i }).isVisible()` is true before any sign-in action.

#### Scenario: S-FA-011 Authenticated landing shows user identity instead

- GIVEN the request to `/` carries a valid `authjs.session-token` cookie whose JWE payload decodes to `{ user: { name: "braejan", email: "braejan@example.com" } }`
- WHEN the route loader runs
- THEN the rendered HTML SHALL contain visible text `braejan` (the session name)
- AND the rendered HTML SHALL NOT contain the "Sign in with GitHub" button

Verification: same Playwright spec signs the user in first (fixture), then asserts the signed-in landing does NOT show the sign-in button.

### Requirement: R-FA-003 OAuth callback lands the user on `/profile` authenticated

When the GitHub OAuth flow returns to `<ORIGIN>/auth/callback/github` with a
valid code, the Auth.js onRequest MUST exchange the code, persist (or
auto-link) the user in `identity.user` and `identity.account`, set the JWE
session cookie, and the browser MUST end up on `/profile` with the session
visible via `useSession`.

#### Scenario: S-FA-020 First-time GitHub sign-in creates identity rows

- GIVEN `identity.user` is empty and `identity.account` is empty
- WHEN the orchestrator runs the end-to-end Playwright flow
  `sign-in-with-github.spec.ts` against a real GitHub OAuth test account
- THEN the flow SHALL reach `https://github.com/login/oauth/authorize` and back
- AND on return, the `identity.user` table SHALL contain exactly one row
  whose `email` (case-folded) matches the test account email
- AND the `identity.account` table SHALL contain exactly one row with
  `provider = 'github'` and `provider_account_id` equal to the GitHub user ID

Verification: connect to Postgres as `queen` and run
`SELECT count(*) FROM identity.user WHERE lower(email) = lower($1)` (returns 1) and
`SELECT count(*) FROM identity.account WHERE provider = 'github' AND provider_account_id = $1` (returns 1).

#### Scenario: S-FA-021 Returning GitHub sign-in auto-links on email match

- GIVEN the first-time flow above has created `identity.user` row #42 for `braejan@example.com` and `identity.account` row #7 with `provider='github'`, `provider_account_id=12345`
- AND a different GitHub account (`provider_account_id=99999`) signs in BUT with the same email `braejan@example.com` (the user changed their GitHub handle but kept the email)
- WHEN the orchestrator runs the same Playwright flow
- THEN the `identity.user` row for `braejan@example.com` SHALL remain row #42 (no new user row)
- AND the `identity.account` table SHALL contain a SECOND row with `provider='github'`, `provider_account_id=99999`, `user_id = 42`

Verification: run `SELECT id, count(*) FROM identity.user WHERE lower(email) = lower('braejan@example.com') GROUP BY id HAVING count(*) > 0` returns exactly one row (id = 42). Run `SELECT count(*) FROM identity.account WHERE user_id = 42 AND provider = 'github'` returns 2.

#### Scenario: S-FA-022 Session cookie is set on the Qwik origin only

- GIVEN a complete sign-in
- WHEN the orchestrator inspects the response `Set-Cookie` headers
- THEN exactly one cookie SHALL be set with name `authjs.session-token`
- AND the cookie SHALL have `HttpOnly`
- AND the cookie SHALL have `SameSite=Lax`
- AND the cookie SHALL have `Path=/`
- AND in production builds, the cookie SHALL have `Secure` (in dev compose, `Secure` MAY be omitted because the origin is HTTP)

Verification: Playwright spec `sign-in-cookie-attrs.spec.ts` reads `context.cookies()` and asserts the attributes above.

### Requirement: R-FA-004 PKCE is enabled in the Auth.js GitHub provider config

The `GitHub` provider config MUST request PKCE for the GitHub authorization
request. This slice opts into PKCE even though Auth.js for Qwik is treated as
a confidential client, because PKCE protects against code interception in any
forward-compat public-client deployment.

#### Scenario: S-FA-030 Authorization request carries PKCE parameters

- GIVEN the user initiates a sign-in by clicking the button
- WHEN the browser navigates to `https://github.com/login/oauth/authorize?...`
- THEN the query string SHALL contain a `code_challenge` parameter
- AND the query string SHALL contain a `code_challenge_method=S256` parameter
- AND a `state` parameter SHALL also be present (Auth.js CSRF)

Verification: Playwright spec intercepts the navigation to `github.com` and asserts the URL query contains `code_challenge=` and `code_challenge_method=S256`.

### Requirement: R-FA-005 Sign-out clears the session and the cookie

When the user visits `/auth/signout` (or POSTs to it via `useSignOut`), Auth.js
MUST drop the JWE cookie. A subsequent visit to `/profile` MUST redirect to
`/auth/signin`.

#### Scenario: S-FA-040 Sign-out drops the cookie

- GIVEN a fully signed-in session
- WHEN the Playwright spec triggers `useSignOut` (form submission to `/auth/signout`)
- THEN the response SHALL set `authjs.session-token` to an empty value with `Max-Age=0`
- AND `useSession` from the next page load SHALL return an empty object (no session)

Verification: Playwright `sign-out.spec.ts` reads `context.cookies("https://.../", "authjs.session-token")` and asserts `expires = -1` (or `Max-Age = 0`).

#### Scenario: S-FA-041 Visiting /profile while signed out redirects to /auth/signin

- GIVEN the user clicks "Sign out" and the cookie is dropped
- WHEN the user navigates to `/profile`
- THEN the response SHALL be HTTP 302 with `Location: /auth/signin?callbackUrl=%2Fprofile`

Verification: Playwright spec follows the sign-out flow, then `await page.goto('/profile')`, then asserts `page.url()` matches `/auth/signin\?callbackUrl=%2Fprofile`.

---

## Capability: Persistent server-side identity

### Requirement: R-FA-010 /profile reads server-known identity, not just the JWE

The `/profile` route MUST render the user record as known server-side (joined
from `identity.user`), not just the claims embedded in the JWE cookie. This
protects the page against a scenario where the JWE is valid but the user's
`name` or `image_url` was updated after the cookie was issued.

#### Scenario: S-FA-050 /profile shows the latest server-known name

- GIVEN the user is signed in (cookie present)
- AND the `identity.user` row's `name` was updated from "Old Name" to "New Name" directly in Postgres (via SQL) AFTER the cookie was issued
- WHEN the orchestrator visits `/profile`
- THEN the rendered HTML SHALL contain "New Name"
- AND the rendered HTML SHALL NOT contain "Old Name"

Verification: Playwright spec signs in, updates Postgres directly as `queen` (`UPDATE identity.user SET name = 'New Name' WHERE email = ...`), reloads `/profile`, and asserts the text.

### Requirement: R-FA-011 The signIn callback persists GitHub identity into the database

The `signIn` callback registered with `QwikAuth$` MUST upsert an `identity.user`
row keyed by the lowercased email, then upsert an `identity.account` row keyed
by `(provider = 'github', provider_account_id = ...)`. If the user already
exists by email, the existing user is reused and a new account row is added
(auto-link by email match per proposal §5 question 3).

#### Scenario: S-FA-060 signIn callback is invoked once per sign-in

- GIVEN a fresh Playwright run with `identity.user` empty
- WHEN the orchestrator completes a sign-in
- THEN the callback log line (recorded via OTel span `authjs.signin.callback`) SHALL appear exactly once with attributes `provider=github`, `user.email=<test-account-email>`, `outcome=created` (or `outcome=linked` if the email already existed)

Verification: Playwright spec attaches a network listener to capture the Jaeger trace or the OTel collector's debug exporter, and asserts on the span attributes.

---

## Capability: Frontend error and recovery semantics

### Requirement: R-FA-020 A failed GitHub roundtrip returns the user to the app with an error message

If GitHub returns an `error=access_denied` (or any other error) on the
callback, Auth.js for Qwik MUST surface the error on the next page render
without infinite-redirecting.

#### Scenario: S-FA-070 GitHub consent denial lands on a visible error page

- GIVEN the orchestrator stubs the GitHub authorize endpoint to redirect back to `<ORIGIN>/auth/callback/github?error=access_denied&state=...`
- WHEN the browser follows the redirect
- THEN the rendered HTML SHALL contain visible text "Sign-in failed" (or the localized equivalent)
- AND the page MUST NOT redirect again to GitHub

Verification: Playwright spec `sign-in-denied.spec.ts` performs the denial stub and asserts on the rendered text.

### Requirement: R-FA-021 A missing `AUTH_SECRET` env var causes a fatal startup error

The Qwik Node server MUST refuse to start (or refuse to handle `/auth/*`
requests) if `AUTH_SECRET` is missing or shorter than the recommended length.
This fails fast rather than silently producing unverifiable JWE cookies.

#### Scenario: S-FA-080 Missing AUTH_SECRET causes a visible error

- GIVEN the frontend container is started with `AUTH_SECRET=` (empty)
- WHEN `docker compose logs frontend` is inspected
- THEN the logs SHALL contain an error line naming `AUTH_SECRET` and stating it MUST be set
- AND the healthcheck SHALL report `unhealthy` (or the container SHALL exit 1)

Verification: temporarily start the container with `AUTH_SECRET=` and inspect the logs.

---

## Capability: Test coverage contract

### Requirement: R-FA-100 Vitest covers auth components and plugins

The repo MUST add Vitest unit specs for:

- `routes/plugin@auth.ts` (mocking `Auth.js` so the export shape is asserted)
- `components/sign-in-button/*` (renders the form, fires the action)

#### Scenario: S-FA-090 Component-level tests pass

- GIVEN the change is applied
- WHEN `pnpm test:ci` runs from `frontend/`
- THEN at least three new test functions SHALL pass (`plugin@auth.test.ts`, `sign-in-button.test.tsx`, and one round-trip test)
- AND the existing Vitest suite SHALL remain green (no test removed or skipped)

Verification: CI job `frontend/test:unit` reports new specs and an unchanged overall pass count.

### Requirement: R-FA-101 Playwright e2e covers the sign-in → /profile round trip

A new Playwright spec MUST drive the OAuth roundtrip against the local dev
stack (using the GitHub test account the orchestrator registered as a fixture)
and assert on the resulting `/profile` page render, identity rows, and cookie
attributes.

#### Scenario: S-FA-095 e2e suite passes

- GIVEN `docker compose -f docker-compose.yaml up -d` is running and healthy
- AND Playwright is configured to launch Chromium against `http://localhost:3015`
- WHEN `pnpm test:e2e` runs from `frontend/`
- THEN the new spec `e2e/github-sign-in.spec.ts` SHALL pass
- AND the existing create-org spec SHALL still pass

Verification: CI job `frontend/test:e2e` reports both specs green.

---

## Review checklist

- [ ] reviewer can confirm the spec describes WHAT (capabilities, requirements, scenarios) and not HOW (no library names appear inside requirement bodies beyond Auth.js itself, which IS the capability)
- [ ] reviewer can confirm every scenario uses `GIVEN/WHEN/THEN` and is independently verifiable with a Playwright spec or `psql` query
- [ ] reviewer can confirm the cookie attribute matrix is consistent across S-FA-022, S-FA-040, and the proposal
- [ ] reviewer can confirm PKCE is asserted in S-FA-030 (not just stated in passing)
- [ ] reviewer can confirm no file under `backend/database_administrator/src/` or `infra/` was modified by this spec
- [ ] reviewer can confirm the spec does not propose adding a `identity.session` table (intentional out-of-scope)
- [ ] reviewer can confirm tests required (R-FA-100, R-FA-101) are testable WITHOUT real GitHub credentials by relying on a fixture OAuth test account and `authjs.state` shaping
