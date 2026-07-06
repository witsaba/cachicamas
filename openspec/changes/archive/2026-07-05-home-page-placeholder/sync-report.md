# Sync Report — `home-page-placeholder`

## Status

**PASS** — verified delta spec promoted into the OpenSpec canonical `specs/` tree.

## What was synced

The delta spec at `openspec/changes/home-page-placeholder/specs/home-page/spec.md` was promoted to canonical:

- **From**: `openspec/changes/home-page-placeholder/specs/home-page/spec.md` (the change-local delta, R-HP-001..R-HP-008)
- **To**: `openspec/specs/home-page/spec.md` (the canonical spec now visible to all future changes)

The promoted file is byte-identical to the verified delta — no edits during sync.

## Why this sync

`sdd-verify` reported **PASS_WITH_NOTES** with verdict **READY FOR SYNC**. All 8 spec requirements satisfied with concrete test evidence, strict TDD gate green (215/215 tests, lint clean, fmt clean), and no spec-to-implementation deviations.

The canonical `openspec/specs/` was previously empty for this area. Promoting the delta makes the home-page spec discoverable alongside the other canonical specs.

## Affected files

| Action | Path |
|---|---|
| Created | `openspec/specs/home-page/spec.md` |
| Read (no edit) | `openspec/changes/home-page-placeholder/specs/home-page/spec.md` |

## Notes inherited from verify

These notes propagate to the canonical spec; no spec-content changes were needed.

- D-1: `Array.from(...)` wrap on `querySelectorAll("svg")` in `routes/home/index.spec.tsx` — behaviour-preserving.
- D-2: `useHomeSession` routeLoader is a no-op (`void cookie; void env; return null;`) — required by the design for SSR parity with `/profile`.
- D-3: `openspec/archive/*` and `openspec/specs/*` deletions in `git status` are pre-existing working-tree state — out of scope.

## Residual risks (inherited from verify)

- R-1: Anonymous SSR flash on `/home` — Low/Low. Server-side guard via `routeLoader$` mitigates; Playwright e2e would close it (out of scope for this slice).
- R-2: `createDOM()` `querySelectorAll` non-iterable — Low/Low. Already mitigated by `Array.from`.
- R-3: Personalised greeting relies on the OAuth `name` claim — Low/Low. Fallback `"Welcome"` covers missing/empty/null shapes.
- R-4: `useHomeSession` routeLoader no-op — Low/Low. Required for SSR declaration parity; no runtime cost.

## Strict TDD evidence (last run before sync)

| Step | Command | Exit | Summary |
|---|---|---|---|
| 1 | `cd frontend && pnpm run test:ci` | 0 | 26 test files / 215 tests passing |
| 2 | `pnpm --dir frontend run lint` | 0 | ESLint clean |
| 3 | `pnpm --dir frontend run fmt.check` | 0 | "All matched files use Prettier code style!" |

## Next

`sdd-archive` — move `openspec/changes/home-page-placeholder/` to `openspec/changes/archive/2026-07-05-home-page-placeholder/`. The change is verified and synced; archive marks it as done.