# Explore — `home-page-placeholder`

## Change at a glance

Auth-aware conditional render in `frontend/src/routes/index.tsx`:

- Anonymous visitors (`useSession().value === null`) keep seeing the existing landing (hero + 4 features + CLI block + footer).
- Authenticated visitors (`useSession().value?.user !== null`) see a new minimal Home Page (heading + paragraph).
- Single route, no new files required for the implementation itself.

## Codebase map

### `frontend/src/routes/index.tsx` (existing landing)

- Already calls `useSession()` and currently `void session.value;` to keep the route auth-co-located with `routes/layout.tsx`.
- The existing comment block explicitly forbids a body SignInButton: the layout's header is the single identity affordance.
- Exports: `default component$` and `head: DocumentHead`.

### `frontend/src/routes/layout.tsx` (auth-aware shell)

- Renders the header chrome and a `<Slot />` for the route.
- Re-validates session on `popstate` (UAT-8 revision 4) — relevant only for production navigation; vitest's `createDOM()` does not exercise this.

### `frontend/src/routes/plugin@auth.ts` (Auth.js plugin)

- Exports `useSession`, `useSignIn`, `useSignOut`, `onRequest` from `QwikAuth$`.
- `useSession()` returns a Qwik `Signal<Session | null>`. The shape is `{ value: Session | null }` at the consumer side; `session.value?.user` is the canonical "is the user signed in?" check (already used in `routes/layout.tsx`).

### `frontend/src/routes/index.spec.tsx` (existing tests)

- Mocks `~/routes/plugin@auth` so `useSession()` returns `{ value: null }` — i.e. the anonymous case.
- Asserts the landing surface: brand `<h1>`, primary CTA → `/organizations/new`, secondary CTA → `#interface`, 4 feature cards, CLI block, 3 section labels, footer, no `<img>/<picture>/<svg>`, no body SignInButton.
- Critical: every existing test renders against the anonymous session, so the auth-aware branch must keep the anonymous render identical. Tests must keep passing without modification.

### Test pattern notes

- Vitest with `@builder.io/qwik/testing`'s `createDOM()` + `render(<Component />)`.
- `vi.mock("~/routes/plugin@auth", ...)` factory pattern; the file header warns about mock-state leakage between `vi.doMock`/`vi.resetModules` tests.
- Tests use `data-*` attributes (`data-cta`, `data-section`, `data-feature`, `data-surface`, `data-footer`) as stable selectors — the new Home Page should follow the same convention so existing harnesses stay deterministic.

## Implementation shape (recommended)

Single file: `frontend/src/routes/index.tsx`.

```tsx
const session = useSession();
const isAuthenticated =
  session.value?.user !== null && session.value?.user !== undefined;

if (isAuthenticated) {
  return <HomePagePlaceholder />; // minimal heading + paragraph
}
return <LandingPage />; // existing JSX, untouched
```

A small inline `HomePagePlaceholder` component or a private `component$` helper inside the file is enough. A separate component file is not justified for this scope.

### Test strategy

- Keep `index.spec.tsx` as the landing suite (anonymous mock stays `{ value: null }`).
- Add a sibling or extended test in the same file for the authenticated case:
  - Use `vi.doMock` + `vi.resetModules` to swap `useSession` to `{ value: { user: { name: "Test" } } }`.
  - Place this test LAST per the file's own comment about mock-state leakage.
  - Assert: brand heading is present, no `data-cta="get-started"`, no `data-surface="cli"`, the new `[data-testid="home-placeholder"]` or equivalent marker exists.

### Edge cases / pitfalls

- **SSR vs client**: `useSession()` works in both; no `useVisibleTask$` needed.
- **`head` metadata**: The authed Home Page should set its own `head` (title + description). Decide whether the current landing's `head` stays as the anonymous default.
- **Layout popstate reload** (UAT-8 r4): not exercised in vitest; production-safe because the auth-aware branch reads `session.value` afresh on each render.
- **No body SignInButton**: this lock from UAT-1 must hold for the new Home Page too — the header is the single identity affordance.

## Scope confirmation

- No new routes.
- No new components outside `index.tsx`.
- No backend changes.
- No database / migration changes.
- Diff estimated ≤ 60 lines (test additions ≈ 30, source change ≈ 30).
- Single PR; well under the 500-line review budget.
