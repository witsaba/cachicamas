# Design — Home Page

## Overview

Implement an authed-only `/home` route following the existing protected-route pattern (`/profile`, `/organizations/[id]`), plus two single-line `redirectTo` updates so the auth roundtrips land where they should.

The implementation reuses the project's existing primitives — `requireSession`, `SignInRequiredCard`, `SignInButton`, `useSession`, `useSignIn` — and introduces no new shared abstractions.

## Tradeoffs

### T1 — Inline JSX vs. extracted `HomeView` component

**Considered**

- **A. Inline JSX in `routes/home/index.tsx`** (recommended).
- **B. Extract `<HomeView>` to `components/home-view/home-view.tsx`** (matches the `ProfileView` split).

**Decision**: **A** — inline JSX.

**Why**: The authenticated render is a single `<h1>` + one `<p>` placeholder. Extracting a component for two elements adds a file, an import boundary, and a separate spec for marginal gain. The route's spec can drive the render directly via `vi.mock("~/routes/plugin@auth")` — same approach already used by `routes/index.spec.tsx`. The cost of extracting only pays off when the view grows beyond a few elements.

**When to revisit**: if a future iteration adds widgets, cards, or per-section specs, split out `<HomeView>` at that point.

### T2 — Route loader: `routeLoader$` vs. client-only guard

**Considered**

- **A. `routeLoader$` + `requireSession()`** (recommended — matches `/profile`).
- **B. Client-side `useTask$` guard that redirects via `useNavigate()`**.

**Decision**: **A**.

**Why**: `routeLoader$` runs in the request context (cookie parse, JWT verify), so anonymous visitors never receive the Home Page HTML — they get the redirect response from the server. The client-only alternative flashes the page before redirecting and exposes the auth check to client-side manipulation. The pattern is already established in the codebase; using it keeps `/home` consistent with `/profile` and `/organizations/[id]`. The unit-test limitation (createDOM can't drive `routeLoader$`) is mitigated by the structural `route-guard.spec.ts`.

### T3 — Personalised greeting: render name claim vs. generic greeting

**Considered**

- **A. Render `session.value?.user?.name` with `?? "there"` fallback** (recommended).
- **B. Always render a static "Welcome to cachicamas" headline.**

**Decision**: **A**.

**Why**: The user explicitly chose the personalised path in the proposal question round. The fallback to `"there"` (not the empty string) keeps the comma grammar correct: `"Welcome, there"` rather than `"Welcome, "`. The spec covers all four name-claim shapes (Alice, empty, null, unicode).

### T4 — Header `SignInButton` `redirectTo` change: prop vs. default

**Considered**

- **A. Pass `redirectTo="/home"` explicitly at the layout's call site** (recommended).
- **B. Change the `SignInButton` default to `/home`.**

**Decision**: **A**.

**Why**: `SignInRequiredCard` already passes a per-call `redirectTo`; changing the default would shadow those calls and make the `SignInRequiredCard` semantics ambiguous. Keeping the default at `/profile` preserves backward compatibility for the `SignInRequiredCard` consumers, while the layout's explicit prop makes the intent obvious at the call site.

### T5 — Sign-out destination: `/auth/signin` vs. `/`

**Considered**

- **A. `redirectTo="/auth/signin"`** (per user direction).
- **B. `redirectTo="/"` (current behaviour).**

**Decision**: **A**.

**Why**: User explicitly chose `/auth/signin` in the proposal question round. Side effect: signed-out users land on the Auth.js sign-in surface instead of the public landing. This is intentional — the user wants the explicit sign-in step rather than the implicit return to marketing. Documented in the proposal under "Decisions locked this round".

### T6 — Test mock placement: `vi.mock` factory vs. `vi.doMock` per test

**Considered**

- **A. `vi.mock` factory at the top + `vi.doMock` per test for the authed case** (recommended).
- **B. Separate spec files for anon and authed cases.**

**Decision**: **A**.

**Why**: The file-header comment in `routes/index.spec.tsx` warns about `vi.doMock`/`vi.resetModules` test ordering (mock-state leakage). One file keeps the spec discoverable and matches the project's existing structure. Place the `vi.doMock` test LAST so leakage cannot break earlier assertions. The factory at the top covers the anon case by default; the per-test override swaps in a user.

### T7 — `route-guard.spec.ts` coverage: structural vs. behavioural

**Considered**

- **A. Structural assertions via `readFileSync` + regex** (recommended — R-PR-003 pattern).
- **B. Mock `routeLoader$` directly and assert guard dispatch behaviour.**

**Decision**: **A**.

**Why**: The existing `routes/profile/route-guard.spec.ts` uses `readFileSync` and asserts that the source contains `requireSession`, `useSession()`, `kind === "anon"`, `<SignInRequiredCard`, etc. This is the project's established pattern. The behavioural coverage lives in `index.spec.tsx` via `vi.mock`. No need to invent a new approach.

## Affected files and change shape

### `frontend/src/routes/home/index.tsx` (new)

```tsx
import { component$ } from "@builder.io/qwik";
import { routeLoader$, type DocumentHead } from "@builder.io/qwik-city";
import { SignInRequiredCard } from "~/components/sign-in-required-card/sign-in-required-card";
import { requireSession } from "~/lib/require-session";
import { useSession, useSignIn } from "~/routes/plugin@auth";

export const useHomeSession = routeLoader$(({ cookie, env }) => {
  // Session resolution happens via the Auth.js plugin's request context.
  // We do not read `cookie`/`env` directly here; useSession() inside the
  // component$ does that. The loader exists to allow SSR-time auth checks
  // (mirrors routes/profile/index.tsx).
  void cookie;
  void env;
  return null;
});

export default component$(() => {
  const sessionSig = useSession();
  const signInAction = useSignIn();
  const guard = requireSession(sessionSig.value, "/home");
  if (guard.kind === "anon") {
    return (
      <SignInRequiredCard
        signIn={signInAction}
        description="Sign in to view your home."
        redirectTo={guard.pathname}
      />
    );
  }
  const name = guard.session?.user?.name ?? "";
  const heading = name.length > 0 ? `Welcome, ${name}` : "Welcome";
  return (
    <main class="mx-auto max-w-3xl px-4 py-16">
      <h1 class="text-3xl font-bold text-slate-900" data-testid="home-heading">
        {heading}
      </h1>
      <p class="mt-3 text-slate-700" data-testid="home-paragraph">
        This is your cachicamas home. New sections and shortcuts will appear
        here as the app grows.
      </p>
    </main>
  );
});

export const head: DocumentHead = {
  title: "Home — Cachicamas",
  meta: [
    {
      name: "description",
      content: "Your cachicamas home, signed in via GitHub.",
    },
  ],
};
```

### `frontend/src/routes/home/index.spec.tsx` (new)

Mocks `~/routes/plugin@auth` with `useSession: () => ({ value: null })` at the top (anon default). One test for anon render. One test for the heading shape. One test for UX-4 (no `<img>`/`<picture>`/`<svg>`). One test for the authed case using `vi.doMock` + `vi.resetModules`, placed LAST.

### `frontend/src/routes/home/route-guard.spec.ts` (new)

Structural assertions per `routes/profile/route-guard.spec.ts`:

- imports `requireSession` from `~/lib/require-session`
- imports `SignInRequiredCard` from `~/components/sign-in-required-card/sign-in-required-card`
- calls `useSession()` and `requireSession(..., "/home")`
- branches on `kind === "anon"` and renders `<SignInRequiredCard`

### `frontend/src/routes/layout.tsx` (modified)

Single-line change: pass `redirectTo="/home"` to the header `SignInButton`:

```tsx
<SignInButton signIn={signIn} redirectTo="/home" />
```

### `frontend/src/routes/layout.spec.tsx` (modified)

Add or update one assertion to verify the header `SignInButton` carries `redirectTo="/home"`. Use the same DOM query pattern the spec already uses for the sign-in button form.

### `frontend/src/components/avatar-dropdown/avatar-dropdown.tsx` (modified)

Single-line change in the sign-out `<Form>`: hidden `redirectTo` value `/` → `/auth/signin`:

```tsx
<input type="hidden" name="redirectTo" value="/auth/signin" />
```

### `frontend/src/components/avatar-dropdown/avatar-dropdown.spec.tsx` (modified)

Add or update one assertion to verify the sign-out form's hidden `redirectTo` input has value `/auth/signin`.

## Failure modes and rollback

| Failure | Detection | Mitigation |
| --- | --- | --- |
| Anonymous flash on `/home` (SSR sends HTML then JS redirects) | Vitest `route-guard.spec.ts` would still pass; only Playwright catches it. | Use `routeLoader$` + `requireSession()` (not client-side guard). |
| Mock state leakage between tests | `routes/home/index.spec.tsx` | Place `vi.doMock` tests LAST; add a comment explaining the order. |
| Layout/avatar dropdown redirectTo breaks an existing test | `cd frontend && pnpm run vitest` | Update the affected assertion explicitly in the same change. |
| Name claim is `undefined` instead of `null` (Auth.js shape drift) | Spec fails on S-HP-001 | Optional chaining + `?? ""` covers both shapes; if Auth.js changes again, extend the fallback. |

Rollback is the diff inverse — see `proposal.md` "Rollback" section. No migrations.

## Dependencies on other code

- `lib/require-session.ts` — existing, used as-is.
- `components/sign-in-required-card/sign-in-required-card.tsx` — existing, used as-is.
- `components/sign-in-button/sign-in-button.tsx` — existing, only the layout call site passes the new prop.
- `routes/plugin@auth.ts` — existing, only consumed (not modified).

No new dependencies, no `package.json` changes.

## Strict TDD plan (RED → GREEN → TRIANGULATE → REFACTOR)

Apply phase must record evidence for:

1. RED — write `routes/home/index.spec.tsx` and `route-guard.spec.ts` with the assertions from the spec; run vitest, capture the failure output.
2. GREEN — implement `routes/home/index.tsx` to satisfy all assertions; run vitest, capture the pass output.
3. TRIANGULATE — add the updated layout + avatar-dropdown spec assertions (RED), then apply the single-line edits (GREEN), capture evidence.
4. REFACTOR — clean up any duplication introduced in steps 2 and 3, re-run vitest to confirm green.

The `apply-progress.md` artifact must reference these test outputs (paths to captured output files or pasted excerpts).
