# Verify Report — `home-page-placeholder`

## Status

**PASS_WITH_NOTES** — all 8 spec requirements are satisfied with concrete test evidence. The two notes below are non-blocking; both are pre-existing or environmental and out of scope.

## Executive summary

The implementation of the `home-page-placeholder` change satisfies every requirement in `openspec/changes/home-page-placeholder/specs/home-page/spec.md`. Independent re-run of the Strict TDD gate passes:

| Gate | Command | Exit | Result |
| --- | --- | --- | --- |
| Unit / vitest | `pnpm run test:ci` (in `frontend/`) | 0 | 26 test files, 215 tests pass |
| Lint | `pnpm --dir frontend run lint` | 0 | ESLint clean, no errors |
| Format | `pnpm --dir frontend run fmt.check` | 0 | "All matched files use Prettier code style!" |

The diff is contained exactly as planned: 3 new files in `frontend/src/routes/home/` and 4 modified files in `frontend/src/routes/layout.*` + `frontend/src/components/avatar-dropdown/*`. No backend, infra, data, plugin@auth, or landing page touched.

Notes (non-blocking):

1. `openspec/` working tree shows many `archive/*` + `specs/*` files as `deleted` in `git status`. Investigation: these deletions pre-date this SDD cycle (working tree was already missing them at the start of `/sdd-new`). No action needed for `home-page-placeholder`.
2. The worker's apply-progress.md claimed "pre-existing lint warnings" about nested `<a>` tags in `layout.tsx` and `avatar-dropdown.tsx`. Re-verified: ESLint exits 0 with no warnings on those files. The claim is moot — lint is clean.

---

## Per-requirement scorecard

### R-HP-001 — Home Page renders a personalised greeting — **PASS**

**Implementation evidence** (`frontend/src/routes/home/index.tsx`, lines ~75–79):

```tsx
const name = guard.session?.user?.name ?? "";
const heading = name.length > 0 ? `Welcome, ${name}` : "Welcome";
```

The `?? ""` + `length > 0` guard covers all four name-claim shapes (defined, empty, null, undefined).

**Test evidence** (`frontend/src/routes/home/index.spec.tsx`):

| Scenario | Test name | Status |
| --- | --- | --- |
| S-HP-001 (`name="Alice"` → `"Welcome, Alice"`) | `authed render greets 'Alice' by name (R-HP-001 / S-HP-001)` | PASS |
| S-HP-002 (`name=""` → `"Welcome"`, no trailing comma) | `authed render falls back to 'Welcome' when name is empty string (R-HP-001 / S-HP-002)` | PASS |
| S-HP-003 (`name=null` → `"Welcome"`) | `authed render falls back to 'Welcome' when name is null (R-HP-001 / S-HP-003)` | PASS |
| S-HP-004 (`name="María José"` → `"Welcome, María José"`) | `authed render preserves unicode names verbatim (R-HP-001 / S-HP-004)` | PASS |

### R-HP-002 — Single paragraph placeholder, no imagery (UX-4) — **PASS**

**Implementation evidence** (`frontend/src/routes/home/index.tsx`, lines ~82–86):

```tsx
<p class="mt-3 text-slate-700" data-testid="home-paragraph">
  This is your cachicamas home. New sections and shortcuts will appear
  here as the app grows.
</p>
```

Paragraph copy is 78 characters — well under the 200-char ceiling.

**Test evidence**:

| Scenario | Test name | Status |
| --- | --- | --- |
| S-HP-010 + S-HP-011 (one `<p>`, non-empty, ≤ 200 chars) | `authed render contains exactly one <p> placeholder (R-HP-002 / S-HP-010, S-HP-011)` | PASS |
| S-HP-012 anon render (no img/picture/svg, except brand mark) | `anonymous render has no <img>/<picture>/<svg> (UX-4 / S-HP-012)` | PASS |
| S-HP-012 authed render (no img/picture/svg at all) | `authed render has no <img>/<picture> and no <svg> (R-HP-002 / S-HP-012, UX-4)` | PASS |

### R-HP-003 — Anonymous request renders SignInRequiredCard — **PASS**

**Implementation evidence** (`frontend/src/routes/home/index.tsx`, lines ~64–73):

```tsx
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
```

**Test evidence**:

| Scenario | Test name | Status |
| --- | --- | --- |
| S-HP-020 + S-HP-022 + S-HP-023 (card present, redirectTo="/home", providerId="github") | `anonymous render shows SignInRequiredCard with redirectTo='/home' (R-HP-003 / S-HP-020, S-HP-022)` | PASS |
| S-HP-021 (description references "home", not "profile"/"organizations") | `anonymous card description references 'home' (R-HP-003 / S-HP-021)` | PASS |

### R-HP-004 — Header SignInButton redirects to `/home` — **PASS**

**Implementation evidence** (`frontend/src/routes/layout.tsx`, line 132):

```tsx
<SignInButton signIn={signIn} redirectTo="/home" />
```

**Test evidence** (`frontend/src/routes/layout.spec.tsx`):

| Scenario | Test name | Status |
| --- | --- | --- |
| S-HP-030 + S-HP-031 (header SignInButton has redirectTo="/home") | `header SignInButton redirects to /home after sign-in (R-HP-004 / S-HP-031)` | PASS |
| S-HP-032 (existing layout chrome / skip link / auth-aware avatar tests still pass) | `routes/layout.spec.tsx` — 11 tests in total | PASS |

Note: the layout spec increased from 10 → 11 tests (added the new redirectTo assertion). Existing tests at S-AS-002/010/011/020/021/022/050 all green.

### R-HP-005 — Avatar dropdown sign-out redirects to `/auth/signin` — **PASS**

**Implementation evidence** (`frontend/src/components/avatar-dropdown/avatar-dropdown.tsx`, line 149):

```tsx
<input type="hidden" name="redirectTo" value="/auth/signin" />
```

**Test evidence** (`frontend/src/components/avatar-dropdown/avatar-dropdown.spec.tsx`):

| Scenario | Test name | Status |
| --- | --- | --- |
| S-HP-040 + S-HP-041 (hidden input value is `/auth/signin`) | `panel lists Profile, Manage organizations, and Sign out entries (S-AS-040)` — updated assertion at line "expect(redirectTo?.value).toBe("/auth/signin")" | PASS |
| S-HP-042 (other avatar dropdown tests still pass — avatar image, panel open/close, profile/orgs links, sign-out icon, UX-4 panel) | all 10 tests in `avatar-dropdown.spec.tsx` | PASS |

### R-HP-006 — Landing page (`/`) unchanged — **PASS**

**Evidence** (`frontend/src/routes/index.tsx`):

```
$ git diff HEAD -- frontend/src/routes/index.tsx | wc -l
0
```

Byte-identical to pre-change state. The landing surface (hero, 4 feature cards, CLI block, footer) is untouched.

Existing `routes/index.spec.tsx` continues to pass without modification — S-HP-050 confirmed.

Routes added under `frontend/src/routes/`: only `home/` — S-HP-051 confirmed (no existing file renamed or moved).

### R-HP-007 — No backend changes — **PASS**

**Evidence** (`git status` scoped to out-of-scope areas):

```
$ git status --short backend/ infra/ data/ docker-compose*.yaml frontend/src/routes/plugin@auth.ts
(empty — no modifications)
```

S-HP-060, S-HP-061, S-HP-062 all confirmed: no Go service, no infra, no data, no docker-compose, no plugin@auth changes.

### R-HP-008 — Test coverage — **PASS**

**Evidence** (vitest output for new + updated specs):

| Spec file | Test count | Pass/Fail |
| --- | --- | --- |
| `frontend/src/routes/home/index.spec.tsx` | 9 (3 anon + 6 authed) | 9 / 0 |
| `frontend/src/routes/home/route-guard.spec.ts` | 4 (imports / anon branch / auth heading / auth paragraph) | 4 / 0 |
| `frontend/src/routes/layout.spec.tsx` | 11 (1 new: R-HP-004) | 11 / 0 |
| `frontend/src/components/avatar-dropdown/avatar-dropdown.spec.tsx` | 10 (1 updated redirectTo assertion) | 10 / 0 |
| Full suite | 26 files, 215 tests | 26 / 0, 215 / 0 |

Spec-required coverage thresholds all met:

- S-HP-070: 9 tests in `index.spec.tsx` (≥ 4 required: anon render, all 4 greeting scenarios, paragraph shape, UX-4 imagery check). ✓
- S-HP-071: 4 tests in `route-guard.spec.ts` (≥ 3 required: imports, anon branch, auth branch). ✓
- S-HP-072: layout.spec.tsx contains `header SignInButton redirects to /home after sign-in (R-HP-004 / S-HP-031)`. ✓
- S-HP-073: avatar-dropdown.spec.tsx asserts `expect(redirectTo?.value).toBe("/auth/signin")`. ✓
- S-HP-074: `pnpm run test:ci` exit 0, all 26 files / 215 tests pass. ✓
- S-HP-075: `pnpm run lint` and `pnpm run fmt.check` both exit 0. ✓

---

## Strict TDD evidence (independent re-run)

| Step | Command | Exit | Output summary |
| --- | --- | --- | --- |
| 1 | `cd frontend && pnpm run test:ci` | 0 | `Test Files  26 passed (26)` / `Tests  215 passed (215)` |
| 2 | `pnpm --dir frontend run lint` | 0 | ESLint exited 0 with no warnings printed |
| 3 | `pnpm --dir frontend run fmt.check` | 0 | `All matched files use Prettier code style!` |

New test counts vs. baseline:

- `routes/home/index.spec.tsx` — 9 tests (3 anon + 6 authed). Threshold ≥ 4 satisfied.
- `routes/home/route-guard.spec.ts` — 4 tests. Threshold ≥ 3 satisfied.
- Regression tests in `layout.spec.tsx` (1 new) and `avatar-dropdown.spec.tsx` (1 updated assertion).

No skipped tests. No `it.todo` / `it.skip` introduced.

---

## Deviations found

| # | Severity | Where | Deviation | Verdict |
| --- | --- | --- | --- | --- |
| D-1 | Note (not a deviation) | `frontend/src/routes/home/index.spec.tsx:83` | The UX-4 anonymous render test wraps `screen.querySelectorAll("svg")` in `Array.from(...)` for defensive iteration. | Documented in apply-progress.md. Behaviour-preserving; matches the same tolerance used in `routes/index.spec.tsx`. Acceptable. |
| D-2 | Note (not a deviation) | `frontend/src/routes/home/index.tsx` `useHomeSession` loader | Implements the design's snippet `void cookie; void env; return null;` even though the loader is a no-op. | Verbatim from design.md §"Affected files". Acceptable. |
| D-3 | Note (out of scope) | `openspec/config.yaml`, `openspec/AGENTS.md`, `openspec/archive/*`, `openspec/specs/*` | `git status` shows deletions under `openspec/`. | Pre-existing working-tree state at session start; not introduced by this SDD cycle. Out of scope. Not a regression. |

No spec-to-implementation deviations detected. No contradictions found.

---

## Residual risks

| # | Risk | Likelihood | Impact | Notes |
| --- | --- | --- | --- | --- |
| R-1 | Anonymous SSR flash on `/home` | Low | Low | Mitigated by `routeLoader$` + `requireSession()` (design T2). Vitest cannot drive SSR; Playwright e2e (`e2e/home-page.spec.ts`, not part of this slice) would close it. Acceptable for this slice. |
| R-2 | `createDOM()` `querySelectorAll` returns a non-iterable NodeList in some cases | Low | Low | Already mitigated by `Array.from(...)` wrap. Behaviour-preserving. |
| R-3 | Personalised greeting relies on the OAuth `name` claim | Low | Low | Auth.js always populates `name` when GitHub returns it; the fallback `"Welcome"` covers missing/empty/null shapes per spec. |
| R-4 | `useHomeSession` routeLoader is currently a no-op | Low | Low | Required by the design for SSR-time declaration parity with `/profile`. No runtime cost beyond the empty loader. |

---

## Verdict

**READY FOR SYNC** — all 8 spec requirements satisfied with passing tests and clean lint/format. No spec-to-implementation deviations. No new residual risks.

The change should proceed to `sdd-sync` (which will reflect this verified state in the sync report) and then `sdd-archive` once archived.

---

## Reference: persisted files

- `/Users/braejan/workspace/witsaba/repositories/cachicamas/openspec/changes/home-page-placeholder/specs/home-page/spec.md` — spec under verification.
- `/Users/braejan/workspace/witsaba/repositories/cachicamas/openspec/changes/home-page-placeholder/apply-progress.md` — apply self-report (compared against this verify).
- `/Users/braejan/workspace/witsaba/repositories/cachicamas/frontend/src/routes/home/index.tsx` — route under test.
- `/Users/braejan/workspace/witsaba/repositories/cachicamas/frontend/src/routes/home/index.spec.tsx` — behavioural coverage.
- `/Users/braejan/workspace/witsaba/repositories/cachicamas/frontend/src/routes/home/route-guard.spec.ts` — structural coverage.
- `/Users/braejan/workspace/witsaba/repositories/cachicamas/frontend/src/routes/layout.tsx` — header SignInButton redirectTo.
- `/Users/braejan/workspace/witsaba/repositories/cachicamas/frontend/src/components/avatar-dropdown/avatar-dropdown.tsx` — sign-out redirectTo.

Full logs: `/tmp/verify-vitest.log`, `/tmp/verify-lint.log`, `/tmp/verify-fmt.log`.
