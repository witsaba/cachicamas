# Spec — Home Page

## Purpose

Define the acceptance criteria for the new authed-only `/home` route and the post-signin / post-signout redirects that point to it.

This spec is the contract that `sdd-apply` implements and `sdd-verify` checks against. It is intentionally narrow — no dashboard widgets, no DB queries, no new auth providers.

## Requirements

### R-HP-001 — Home Page renders a personalised greeting for authenticated users

The route at `/home` (implemented by `frontend/src/routes/home/index.tsx`) renders a single `<h1>` heading that includes the authenticated user's `name` claim.

**Scenarios**

- S-HP-001 — `session.value.user.name === "Alice"` → heading reads `"Welcome, Alice"`.
- S-HP-002 — `session.value.user.name === ""` (empty string) → heading reads `"Welcome"` (no trailing comma, no name token).
- S-HP-003 — `session.value.user.name === null` (claim missing) → heading reads `"Welcome"`.
- S-HP-004 — `session.value.user.name === "María José"` (unicode) → heading renders the full name without truncation; no HTML escaping regressions.

### R-HP-002 — Home Page contains a single short paragraph placeholder

The `/home` route renders exactly one `<p>` placeholder paragraph under the heading.

**Scenarios**

- S-HP-010 — Authenticated render → exactly one `<p>` element directly under the heading container.
- S-HP-011 — Paragraph text is non-empty and ≤ 200 characters (placeholder copy).
- S-HP-012 — No `<img>`, `<picture>`, or `<svg>` element on the authenticated render (UX-4).

### R-HP-003 — Anonymous request to `/home` renders SignInRequiredCard

The `/home` route uses the same `requireSession()` + `SignInRequiredCard` pattern as `/profile` and `/organizations/[id]`.

**Scenarios**

- S-HP-020 — `session.value === null` → renders the SignInRequiredCard.
- S-HP-021 — The card's `description` text references "home" (not "profile" or "organizations").
- S-HP-022 — The card's `redirectTo` is `"/home"`.
- S-HP-023 — The embedded SignInButton submits to the Auth.js sign-in flow with `providerId="github"`.

### R-HP-004 — Header SignInButton redirects to `/home` after sign-in

The `<SignInButton>` rendered by `frontend/src/routes/layout.tsx` passes `redirectTo="/home"` so anonymous visitors on any route land on `/home` after a successful sign-in.

**Scenarios**

- S-HP-030 — The layout source contains `<SignInButton signIn={signIn} redirectTo="/home" />`.
- S-HP-031 — A regression test in `routes/layout.spec.tsx` asserts the hidden `redirectTo` input value is `"/home"`.
- S-HP-032 — Existing layout tests (header chrome, skip link, auth-aware avatar dropdown) keep passing.

### R-HP-005 — Avatar dropdown sign-out redirects to `/auth/signin`

The sign-out form inside `frontend/src/components/avatar-dropdown/avatar-dropdown.tsx` has a hidden `redirectTo` field set to `"/auth/signin"`.

**Scenarios**

- S-HP-040 — The avatar dropdown source contains `<input type="hidden" name="redirectTo" value="/auth/signin" />` inside the sign-out form.
- S-HP-041 — A regression test in `avatar-dropdown.spec.tsx` asserts the hidden input's value is `"/auth/signin"`.
- S-HP-042 — Other avatar dropdown tests (avatar image, panel open/close, profile/orgs links) keep passing.

### R-HP-006 — Landing page (`/`) is unchanged

The route at `/` continues to render the existing landing surface (hero, 4 feature cards, CLI block, footer). Authenticated users visiting `/` see the same landing, not the Home Page.

**Scenarios**

- S-HP-050 — `routes/index.spec.tsx` continues to pass without modification.
- S-HP-051 — No file in `frontend/src/routes/` other than `home/index.tsx` is added or renamed in this change.

### R-HP-007 — No backend changes

This change does not touch the Go service, the database, the OAuth provider config, or the auth middleware.

**Scenarios**

- S-HP-060 — No file under `backend/` is modified.
- S-HP-061 — No file under `infra/`, `data/`, or `docker-compose*.yaml` is modified.
- S-HP-062 — `frontend/src/routes/plugin@auth.ts` is not modified (the OAuth callback URL handling stays as-is).

### R-HP-008 — Test coverage

The change ships with vitest coverage for every behavioral requirement above and does not regress existing coverage.

**Scenarios**

- S-HP-070 — `frontend/src/routes/home/index.spec.tsx` covers R-HP-001 (all four sub-scenarios for the name claim), R-HP-002 (paragraph shape + UX-4 imagery check), and R-HP-003 (anon card render).
- S-HP-071 — `frontend/src/routes/home/route-guard.spec.ts` follows the R-PR-003 pattern: structural assertions on `index.tsx` for `requireSession` and `SignInRequiredCard` imports + the `kind === "anon"` branch.
- S-HP-072 — `frontend/src/routes/layout.spec.tsx` is updated to assert the header `SignInButton` carries `redirectTo="/home"`.
- S-HP-073 — `frontend/src/components/avatar-dropdown/avatar-dropdown.spec.tsx` is updated to assert the sign-out form's `redirectTo` is `"/auth/signin"`.
- S-HP-074 — `cd frontend && pnpm run vitest` runs the full vitest suite green.
- S-HP-075 — `cd frontend && pnpm run lint` and `cd frontend && pnpm run fmt.check` pass.

## Strict TDD posture

`openspec/config.yaml` declares `strict_tdd: true` and `apply.test_command: "cd frontend && pnpm run vitest"`. The apply phase MUST record RED → GREEN → TRIANGULATE → REFACTOR evidence in `apply-progress.md` for at least the following tests:

- `routes/home/index.spec.tsx` — at least 4 tests (R-HP-001, R-HP-002, R-HP-003, plus the UX-4 no-imagery check).
- `routes/home/route-guard.spec.ts` — at least 3 tests (imports, anon branch, auth branch).
- The updated layout + avatar-dropdown specs (one regression test each).

## Out of scope (per proposal)

- No dashboard widgets.
- No org / project / recent-activity lists.
- No new auth providers, no session shape changes.
- No `/profile` or `/organizations/*` changes.
- No i18n / locale handling.
