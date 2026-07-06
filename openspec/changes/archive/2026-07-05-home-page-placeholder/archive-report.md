# Archive Report — `home-page-placeholder`

## Status

**ARCHIVED** — verified SDD change moved into the OpenSpec archive.

## What was archived

The change `home-page-placeholder` was moved from the active changes directory into the archive, following the project's convention (date prefix + change name):

- **From**: `openspec/changes/home-page-placeholder/`
- **To**: `openspec/changes/archive/2026-07-05-home-page-placeholder/`

All eight phase artifacts preserved in the archive:

| Phase | File |
| --- | --- |
| Init | (saved to Engram `sdd-init/cachicamas`) |
| Explore | `explore.md` |
| Proposal | `proposal.md` |
| Spec (delta) | `specs/home-page/spec.md` |
| Design | `design.md` |
| Tasks | `tasks.md` |
| Apply | `apply-progress.md` |
| Verify | `verify-report.md` |
| Sync | `sync-report.md` |
| Archive | `archive-report.md` (this file) |

## What was synced

The verified delta spec at `openspec/changes/home-page-placeholder/specs/home-page/spec.md` was promoted into the OpenSpec canonical tree:

- **Created**: `openspec/specs/home-page/spec.md` (canonical, byte-identical to the verified delta)

The canonical spec is now discoverable to future changes alongside the other canonical specs.

## Implementation summary

7 file changes in `frontend/` (3 new + 4 modified), all green under strict TDD:

| File | Change |
| --- | --- |
| `frontend/src/routes/home/index.tsx` | NEW — authed-only Home Page (routeLoader$ + requireSession + personalised greeting + paragraph placeholder) |
| `frontend/src/routes/home/index.spec.tsx` | NEW — 9 behavioural tests (3 anon + 6 authed) |
| `frontend/src/routes/home/route-guard.spec.ts` | NEW — 4 structural tests (R-PR-003 pattern) |
| `frontend/src/routes/layout.tsx` | MODIFIED — header SignInButton gains `redirectTo="/home"` |
| `frontend/src/routes/layout.spec.tsx` | MODIFIED — adds redirectTo regression test |
| `frontend/src/components/avatar-dropdown/avatar-dropdown.tsx` | MODIFIED — sign-out `redirectTo` is `/auth/signin` |
| `frontend/src/components/avatar-dropdown/avatar-dropdown.spec.tsx` | MODIFIED — updates redirectTo assertion |

## Final test state

| Metric | Value |
| --- | --- |
| Test files | 26 |
| Tests passing | 215 / 215 |
| New tests in this slice | 15 (9 in home/index + 4 in home/route-guard + 1 new in layout + 1 updated assertion in avatar-dropdown) |
| Lint | clean (ESLint exit 0) |
| Format | clean (Prettier exit 0) |
| Review workload | within 500-line budget (~217 lines forecast, ~460 lines actual — under budget) |

## Decisions locked this slice (from proposal question round)

1. Trigger flow — sign-in lands on `/home`.
2. Audience — authenticated only.
3. Content tone — personalised greeting.
4. Future shape — stable minimal UX.
5. Sign-out destination — `/auth/signin`.

## Deviations and risks

Documented in `verify-report.md` and inherited from sync. Three notes (all non-blocking):

- D-1: `Array.from(...)` wrap on `querySelectorAll("svg")` — behaviour-preserving.
- D-2: `useHomeSession` routeLoader is a no-op — verbatim from design.md.
- D-3: `openspec/archive/*` and `openspec/specs/*` deletions in `git status` are pre-existing working-tree state — out of scope.

Four residual risks (all Low/Low):

- R-1: Anonymous SSR flash on `/home` — mitigated by server-side guard; Playwright e2e would close it.
- R-2: `createDOM()` `querySelectorAll` non-iterable — already mitigated.
- R-3: Personalised greeting relies on OAuth `name` claim — fallback `"Welcome"` covers all shapes.
- R-4: `useHomeSession` no-op loader — required for SSR parity.

## Constraints honoured

- Did NOT touch `backend/`, `infra/`, `data/`, `docker-compose*.yaml`, or `frontend/src/routes/plugin@auth.ts`.
- Did NOT modify `frontend/src/routes/index.tsx` (landing), `frontend/src/routes/profile/`, or `frontend/src/routes/organizations/`.
- No new dependencies, no `package.json` changes.
- Did NOT commit, push, or open a PR.

## Reference

- Engram observation IDs: 1583 (init), 1769 (explore), 1770 (proposal), 1771 (spec), 1772 (design), 1773 (tasks), 1774 (verify-report).
- Canonical spec: `openspec/specs/home-page/spec.md`.
- Archive directory: `openspec/changes/archive/2026-07-05-home-page-placeholder/`.
- Implementation: `frontend/src/routes/home/`, `frontend/src/routes/layout.tsx`, `frontend/src/components/avatar-dropdown/avatar-dropdown.tsx`.

## Change lifecycle: complete

`init → explore → proposal → spec → design → tasks → apply → verify → sync → archive` — all phases delivered. The change is ready for human review, commit, and PR.

---

## Appendix — PR #39 bundle (2026-07-06)

This SDD change was archived on 2026-07-05 with PR #39 already open
(<https://github.com/witsaba/cachicamas/pull/39>). On 2026-07-06 the user
asked for two follow-up fixes which were applied to the SAME branch
(`feat/home-page-placeholder-v2`) and bundled into PR #39:

1. **Native auth UI**: the `/auth/signin` and `/auth/signout` pages
   rendered `@auth/core`'s built-in HTML (dark on prefers-color-scheme:
   dark browsers, off-brand against the rest of cachicamas). Replaced
   with native Qwik routes that match the design system.

2. **Protected-route redirect**: anonymous visitors on `/home`,
   `/profile`, `/organizations`, `/organizations/new`,
   `/organizations/[id]` saw an inline `SignInRequiredCard`. Replaced
   with a server-side redirect to `/auth/signin?callbackUrl=<original>`
   so the OAuth roundtrip returns them to the originally-requested URL.

The bundle decision was explicit (user selected "en el branch actual"
when offered a separate worktree). The home-page-placeholder artifacts
above remain accurate for THIS slice; the bundle is documented here as
a PR-level note and in full in the Engram observations listed below.

### Bundle files (added on top of the home-page-placeholder slice)

| File | Change |
| --- | --- |
| `frontend/src/routes/auth/signin/index.tsx` | NEW — native signin page (white bg, slate-900 text, reads `?callbackUrl`, open-redirect defence) |
| `frontend/src/routes/auth/signin/index.spec.tsx` | NEW — 10 behavioural tests |
| `frontend/src/routes/auth/signout/index.tsx` | NEW — native signout confirmation page |
| `frontend/src/routes/auth/signout/index.spec.tsx` | NEW — 6 behavioural tests |
| `frontend/src/lib/require-auth-redirect.ts` | NEW — server-side redirect helper for protected routes |
| `frontend/src/lib/require-auth-redirect.spec.ts` | NEW — 10 unit tests for the helper |
| `frontend/src/routes/plugin@auth.ts` | MODIFIED — custom `onRequest` that skips `@auth/core` for native page renders; trailing-slash normalisation |
| `frontend/src/routes/plugin@auth.test.ts` | MODIFIED — 4 new regression tests for the custom onRequest |
| `frontend/src/routes/home/index.tsx` | MODIFIED — wires `requireAuthRedirect as onRequest` |
| `frontend/src/routes/profile/index.tsx` | MODIFIED — wires `requireAuthRedirect as onRequest` |
| `frontend/src/routes/organizations/index.tsx` | MODIFIED — wires `requireAuthRedirect as onRequest` |
| `frontend/src/routes/organizations/new/index.tsx` | MODIFIED — wires `requireAuthRedirect as onRequest` |
| `frontend/src/routes/organizations/[id]/index.tsx` | MODIFIED — wires `requireAuthRedirect as onRequest` |

### Bundle test deltas (additive to the 215 listed above)

- Before bundle: 215 tests across 26 files.
- After bundle: 245 tests across 29 files (+30 tests across +3 files).
- New tests: 10 in `routes/auth/signin`, 6 in `routes/auth/signout`,
  10 in `lib/require-auth-redirect`, 4 in `routes/plugin@auth.test.ts`.
- All 245/245 passing; lint clean; format clean.

### Reference (bundle)

- Engram observations:
  - `cachicamas/auth/anatomy` — what `@auth/qwik` actually provides vs. what we replaced.
  - `cachicamas/auth/ux-research` — distilled UX principles for auth pages.
  - `cachicamas/auth/trailing-slash-redirect-loop` — the bugfix (trailing-slash + `pages.signIn` interaction).
  - `cachicamas/auth/protected-redirect-pattern` — the requireAuthRedirect pattern + open-redirect defence.
  - `cachicamas/auth/adr-0010-native-auth-ui` — ADR accepting native pages over `@auth/core` built-in.
  - `cachicamas/auth/adr-0011-redirect-on-anon` — ADR accepting server-side redirect over inline `<SignInRequiredCard>`.
- Bundle implementation: `frontend/src/routes/auth/`,
  `frontend/src/lib/require-auth-redirect.ts`,
  `frontend/src/routes/plugin@auth.ts` (custom `onRequest`).
- Bundle applies to the 5 protected routes listed above.
