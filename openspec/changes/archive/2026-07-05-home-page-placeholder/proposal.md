# Proposal — `home-page-placeholder`

## Intent (problem statement)

After a signed-in user finishes the GitHub OAuth roundtrip, the Auth.js flow currently lands them back on the public landing (`/`) or `/profile`, with no in-app "home" surface to return to. The landing page is a marketing surface, not an app entry point. We need a stable, authed-only "Home" route that:

- Serves as the signed-in user's app entry point after sign-in.
- Greets them by name (uses the OAuth `name` claim from `useSession()`).
- Is reachable directly via `/home`.
- Stays minimal on purpose — no dashboard widgets, no data fetches, no DB calls.

The landing page (`/`) is intentionally NOT modified: it stays as the public, anonymous-friendly marketing surface for both anon and authed users.

## Scope

### In scope

- New authed-only route at `/home` rendered through Qwik City's `routeLoader$` + `requireSession()` guard (the existing pattern used by `/profile` and `/organizations/[id]`).
- New Home Page UI: heading "Welcome, <user.name>" + a single short paragraph placeholder. Personalised via `session.value?.user?.name`, falling back to a generic greeting when the name is missing.
- A new auth-guard test asserting `requireSession` and `SignInRequiredCard` are imported (R-PR-003 pattern).
- A new Home Page spec asserting the authenticated render and the anon card render via `vi.doMock` + `vi.resetModules`.
- Update `routes/layout.tsx` so the header `SignInButton` uses `redirectTo="/home"` (currently defaults to `/profile`).
- Update `components/avatar-dropdown/avatar-dropdown.tsx` so the sign-out form's hidden `redirectTo` is `/auth/signin` (currently `/`).
- Mirror those changes in their existing specs so the redirect assertions stay green.

### Out of scope (non-goals)

- No dashboard widgets, no org/project/recent-activity lists. The Home Page is a placeholder by intent (per user direction: "stable minimal UX", not a seed for future expansion).
- No new auth providers, no session shape changes.
- No backend changes, no DB changes, no migrations.
- No changes to the landing page (`routes/index.tsx`).
- No changes to `/profile` or `/organizations/*`.
- No i18n / locale negotiation.

## Affected areas

| Area | File | Change |
| --- | --- | --- |
| New route | `frontend/src/routes/home/index.tsx` | Authed-only Home Page |
| New route spec | `frontend/src/routes/home/index.spec.tsx` | Vitest coverage for anon + authed renders |
| New route guard test | `frontend/src/routes/home/route-guard.spec.ts` | Asserts `requireSession` + `SignInRequiredCard` wiring (R-PR-003) |
| Layout | `frontend/src/routes/layout.tsx` | Header `SignInButton` `redirectTo="/home"` |
| Layout spec | `frontend/src/routes/layout.spec.tsx` | Update redirect assertions if any |
| Avatar dropdown | `frontend/src/components/avatar-dropdown/avatar-dropdown.tsx` | Sign-out `redirectTo="/auth/signin"` |
| Avatar dropdown spec | `frontend/src/components/avatar-dropdown/avatar-dropdown.spec.tsx` | Update redirect assertions if any |

## Decisions locked this round

These came out of the proposal question round and are the contract for the spec phase.

1. **Trigger flow**: Sign-in lands the user on `/home` (via `redirectTo="/home"` on every SignInButton that does not pass its own `redirectTo`).
2. **Audience**: Authenticated only. Anonymous requests to `/home` get `SignInRequiredCard` with `redirectTo="/home"` so the roundtrip works.
3. **Content tone**: Personalised greeting. Heading reads "Welcome, <name>" using `session.value?.user?.name` (fallback: "Welcome").
4. **Future shape**: Stable minimal UX. Not a dashboard seed; not a placeholder for future cards. Future evolution is separate.
5. **Sign-out destination**: `/auth/signin` (per user direction — explicit over the implicit landing page).

## Risks

| Risk | Likelihood | Impact | Mitigation |
| --- | --- | --- | --- |
| Changing header SignInButton `redirectTo` breaks an existing test assertion | Low | Low | Layout spec already covers the redirect; update assertions explicitly. |
| Sign-out `redirectTo="/auth/signin"` removes the landing page from the post-signout roundtrip | Low | Low | Intentional per user direction. Spec asserts the new value. |
| New `/home` route conflicts with a future file/route name | Low | Low | None — `/home` is the canonical name; reserve it in the change. |
| Personalised greeting assumes `session.value?.user?.name` is always present | Low | Low | Fallback to generic greeting; covered by spec. |
| `routeLoader$` cannot be unit-tested with `createDOM()` (the existing limitation) | High | Low | Follow the R-PR-003 pattern: a `route-guard.spec.ts` that asserts the imports only; the rendering spec mocks `useSession` and tests the JSX branch. |

## Rollback

- All changes are additive (one new route + spec files) plus two single-line config edits (`redirectTo` values). Reverting the diff restores prior behaviour:
  - Remove `frontend/src/routes/home/` and its tests.
  - Revert `routes/layout.tsx` header `SignInButton` `redirectTo` to default (`/profile`).
  - Revert `avatar-dropdown.tsx` sign-out `redirectTo` to `/`.
- No migrations, no DB changes, no breaking API changes — rollback is a plain revert.

## Success criteria

- A signed-in user visiting `/home` sees a heading containing their name (or "Welcome" fallback) and a short paragraph placeholder. No decorative imagery, no `<img>`/`<picture>`/`<svg>` carrying meaning (UX-4).
- An anonymous user visiting `/home` sees the existing `SignInRequiredCard` with `redirectTo="/home"`. Clicking sign-in lands them on `/home`.
- After a successful sign-in from the header CTA (anon on any page), the user lands on `/home`.
- After signing out from the avatar dropdown, the user lands on `/auth/signin`.
- Existing tests stay green: `routes/index.spec.tsx`, `routes/layout.spec.tsx`, `avatar-dropdown.spec.tsx`, `require-session.spec.ts`, `plugin@auth.test.ts`, and the route-guard specs for `/profile` and `/organizations/[id]`.
- New tests pass under `cd frontend && pnpm run vitest`: `routes/home/index.spec.tsx` and `routes/home/route-guard.spec.ts`.
- `cd frontend && pnpm run lint` and `cd frontend && pnpm run fmt.check` pass.
- Diff ≤ ~150 changed lines (source + tests), single PR, well under the 500-line review budget.
