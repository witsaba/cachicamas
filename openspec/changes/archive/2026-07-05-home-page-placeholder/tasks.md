# Tasks — Home Page

Implementation is broken into 10 ordered tasks. The order follows the Strict TDD posture in the design: RED → GREEN → TRIANGULATE → REFACTOR.

Each task lists the spec requirement(s) it satisfies, the files it touches, and a line-budget estimate. The total diff is well under the 500-line review budget, so a single PR is appropriate (no chained-PR decision needed).

## Task ordering

```
RED           1 ──┐
RED           2 ──┤
RED           3 ──┤
RED           4 ──┤
GREEN         5 ──┤
GREEN         6 ──┤
GREEN         7 ──┤
TRIANGULATE   8 ──┤
REFACTOR      9 ──┤
VERIFY       10 ──┘
```

---

## T-HP-01 — RED: write `routes/home/route-guard.spec.ts` (structural) ✅

**Spec**: R-HP-008 (S-HP-071)

Create the structural spec that asserts the route's wiring (R-PR-003 pattern).

**Files**

- New: `frontend/src/routes/home/route-guard.spec.ts`

**Acceptance**

- Imports `describe`/`it`/`expect` from vitest and `readFileSync`/`fileURLToPath` from node.
- Computes the route source path via `fileURLToPath(import.meta.url).replace(...)`.
- Asserts: imports `requireSession`, imports `SignInRequiredCard`, calls `useSession()`, calls `requireSession(..., "/home")`, branches on `kind === "anon"`, renders `<SignInRequiredCard` with `redirectTo={guard.pathname}`.
- Asserts: auth branch renders an `<h1>` with `data-testid="home-heading"`.
- Test fails because `routes/home/index.tsx` does not exist yet.

**Lines**: ~35 (test file)

## T-HP-02 — RED: write `routes/home/index.spec.tsx` (anon default) ✅

**Spec**: R-HP-001 (S-HP-001, S-HP-002, S-HP-003), R-HP-002 (S-HP-010, S-HP-011, S-HP-012), R-HP-003 (S-HP-020, S-HP-021, S-HP-022), R-HP-008 (S-HP-070)

Create the behavioural spec with the `vi.mock` factory for the anon case.

**Files**

- New: `frontend/src/routes/home/index.spec.tsx`

**Acceptance**

- `vi.mock("~/routes/plugin@auth")` factory with `useSession: () => ({ value: null })`.
- Test 1 — anon render → `<SignInRequiredCard>` is present, with `data-testid="sign-in-required-card"`, and the embedded form's hidden `redirectTo` is `/home`.
- Test 2 — anon card description contains "home" (not "profile", not "organizations").
- Test 3 — UX-4 check: no `<img>`, `<picture>`, or `<svg>` on the anon render.
- Tests fail because `routes/home/index.tsx` does not exist yet.

**Lines**: ~55 (test file)

## T-HP-03 — RED: extend `routes/home/index.spec.tsx` (authed case) ✅

**Spec**: R-HP-001 (S-HP-001, S-HP-002, S-HP-003, S-HP-004), R-HP-002 (S-HP-010, S-HP-012)

Append the authed-case tests. Per file-header convention, these MUST be placed LAST so `vi.doMock` leakage cannot break earlier assertions.

**Files**

- Modified: `frontend/src/routes/home/index.spec.tsx`

**Acceptance**

- Test 4 — `vi.doMock` + `vi.resetModules`, re-import `Index`, mock returns `{ value: { user: { name: "Alice" } } }` → heading reads `"Welcome, Alice"`.
- Test 5 — same pattern with `{ value: { user: { name: "" } } }` → heading reads `"Welcome"` (no trailing comma).
- Test 6 — same pattern with `{ value: { user: { name: null } } }` → heading reads `"Welcome"`.
- Test 7 — same pattern with `{ value: { user: { name: "María José" } } }` → heading renders the full unicode name.
- Test 8 — authed render → exactly one `<p>` element with `data-testid="home-paragraph"`, text ≤ 200 chars.
- Test 9 — authed render → UX-4 check (no `<img>`/`<picture>`/`<svg>`).
- Tests fail because the authed branch does not exist yet.

**Lines**: ~50 (additions to existing file)

## T-HP-04 — RED: extend `routes/layout.spec.tsx` (header redirectTo) ✅

**Spec**: R-HP-004 (S-HP-030, S-HP-031), R-HP-008 (S-HP-072)

Add the assertion for the header `SignInButton` carrying `redirectTo="/home"`.

**Files**

- Modified: `frontend/src/routes/layout.spec.tsx`

**Acceptance**

- One new test (or extended existing) that locates the header `SignInButton` form via DOM query and asserts the hidden `redirectTo` input value is `"/home"`.
- Test fails because the layout still uses the default (`/profile`).

**Lines**: ~10 (additions)

## T-HP-05 — RED: extend `avatar-dropdown.spec.tsx` (sign-out redirectTo) ✅

**Spec**: R-HP-005 (S-HP-040, S-HP-041), R-HP-008 (S-HP-073)

Add the assertion for the sign-out form's hidden `redirectTo` input value.

**Files**

- Modified: `frontend/src/components/avatar-dropdown/avatar-dropdown.spec.tsx`

**Acceptance**

- One new test (or extended existing) that locates the sign-out form's hidden `redirectTo` input via DOM query and asserts the value is `"/auth/signin"`.
- Test fails because the value is still `"/"`.

**Lines**: ~10 (additions)

## T-HP-06 — GREEN: implement `routes/home/index.tsx` ✅

**Spec**: R-HP-001, R-HP-002, R-HP-003, R-HP-006, R-HP-007

Implement the route so that T-HP-01..T-HP-03 all turn green.

**Files**

- New: `frontend/src/routes/home/index.tsx`

**Acceptance**

- Uses `routeLoader$` + `requireSession()` + `SignInRequiredCard` for the anon branch.
- Auth branch renders `<main>` with `<h1 data-testid="home-heading">` containing `Welcome, <name>` or `Welcome`.
- Renders exactly one `<p data-testid="home-paragraph">` placeholder.
- No `<img>`, `<picture>`, `<svg>`.
- Exports `head: DocumentHead` with title `"Home — Cachicamas"`.
- Imports `useSession` and `useSignIn` from `~/routes/plugin@auth`.

**Lines**: ~45 (route file)

## T-HP-07 — GREEN: pass `redirectTo="/home"` in `routes/layout.tsx` ✅

**Spec**: R-HP-004 (S-HP-030)

Single-line edit to the layout's `SignInButton` call site.

**Files**

- Modified: `frontend/src/routes/layout.tsx`

**Acceptance**

- The `<SignInButton>` JSX gains `redirectTo="/home"`.
- T-HP-04 turns green.
- All other layout tests stay green.

**Lines**: 1

## T-HP-08 — GREEN: change sign-out `redirectTo` in `avatar-dropdown.tsx` ✅

**Spec**: R-HP-005 (S-HP-040)

Single-line edit in the sign-out form's hidden input.

**Files**

- Modified: `frontend/src/components/avatar-dropdown/avatar-dropdown.tsx`

**Acceptance**

- Hidden `<input name="redirectTo" value="/auth/signin" />` inside the sign-out `<Form>`.
- T-HP-05 turns green.
- All other avatar-dropdown tests stay green.

**Lines**: 1

## T-HP-09 — REFACTOR: clean up and document ✅

**Spec**: R-HP-008

Light pass over the new and modified files for consistency.

**Files**

- `frontend/src/routes/home/index.tsx`
- `frontend/src/routes/home/index.spec.tsx`
- `frontend/src/routes/home/route-guard.spec.ts`
- `frontend/src/routes/layout.tsx`
- `frontend/src/routes/layout.spec.tsx`
- `frontend/src/components/avatar-dropdown/avatar-dropdown.tsx`
- `frontend/src/components/avatar-dropdown/avatar-dropdown.spec.tsx`

**Acceptance**

- Add JSDoc headers on `index.tsx` and `index.spec.tsx` referencing the spec requirements (mirror the comment style on `routes/profile/index.tsx`).
- Confirm the file header on `index.spec.tsx` warns about `vi.doMock`/`vi.resetModules` test ordering, the same way `routes/index.spec.tsx` does.
- Confirm no dead code or duplicated logic.

**Lines**: ~10 (comment additions)

## T-HP-10 — VERIFY: full vitest + lint + fmt check ✅

**Spec**: R-HP-008 (S-HP-074, S-HP-075)

Run the full quality gate and capture evidence.

**Commands**

```bash
cd frontend && pnpm run vitest
cd frontend && pnpm run lint
cd frontend && pnpm run fmt.check
```

**Acceptance**

- `pnpm run vitest` — all suites green; new tests included; no skipped tests.
- `pnpm run lint` — no warnings.
- `pnpm run fmt.check` — all files formatted.
- Capture the output (last ~40 lines each) into `apply-progress.md` as evidence.

**Lines**: 0 (verification only)

---

## Review workload forecast

| Section | Source lines | Test lines | Total |
| --- | --- | --- | --- |
| New `routes/home/index.tsx` | 45 | — | 45 |
| New `routes/home/index.spec.tsx` | — | 105 | 105 |
| New `routes/home/route-guard.spec.ts` | — | 35 | 35 |
| Modified `routes/layout.tsx` | 1 | — | 1 |
| Modified `routes/layout.spec.tsx` | — | 10 | 10 |
| Modified `avatar-dropdown.tsx` | 1 | — | 1 |
| Modified `avatar-dropdown.spec.tsx` | — | 10 | 10 |
| Comments + headers (refactor) | 10 | — | 10 |
| **Total** | **57** | **160** | **217** |

217 changed lines is well under the 500-line review budget. **Single PR. No chained-PR decision needed.**

## Dependency graph

```
T-HP-01 ──→ T-HP-06 ──→ T-HP-09 ──→ T-HP-10
T-HP-02 ──→ T-HP-06 ──→ T-HP-09 ──→ T-HP-10
T-HP-03 ──→ T-HP-06 ──→ T-HP-09 ──→ T-HP-10
T-HP-04 ──→ T-HP-07 ──→ T-HP-09 ──→ T-HP-10
T-HP-05 ──→ T-HP-08 ──→ T-HP-09 ──→ T-HP-10
```

Apply phase executes in the order T-HP-01 → T-HP-10. No parallel writers.
