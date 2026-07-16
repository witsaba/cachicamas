# SDD Apply Progress — 2026-07-16-prompts-frontend

**Status:** DONE  
**Date:** 2026-07-16

## Implementation Summary

All tasks from `tasks.md` completed. Frontend-only change — backend unchanged.

### Files Created

**Infrastructure:**

- `frontend/src/lib/markdown.ts` — `renderMarkdown()` using `marked`
- `frontend/src/lib/markdown.spec.ts` — 10 unit tests
- `frontend/src/lib/diff.ts` — `computeLineDiff()` using LCS-based algorithm
- `frontend/src/lib/diff.spec.ts` — 9 unit tests
- `frontend/src/lib/prompts-api.ts` — API client for all 8 endpoints
- `frontend/src/lib/prompts-api.spec.ts` — 11 wire-shape tests

**Components:**

- `frontend/src/components/prompts/empty-state/empty-state.tsx`
- `frontend/src/components/prompts/prompt-list-item/classes.ts` + `.spec.ts`
- `frontend/src/components/prompts/prompt-list-item/prompt-list-item.tsx`
- `frontend/src/components/prompts/markdown-textarea/markdown-textarea.tsx`
- `frontend/src/components/prompts/markdown-preview/markdown-preview.tsx`
- `frontend/src/components/prompts/diff-viewer/classes.ts` + `.spec.ts`
- `frontend/src/components/prompts/diff-viewer/diff-block.tsx`
- `frontend/src/components/prompts/diff-viewer/diff-viewer.tsx`
- `frontend/src/components/prompts/activity-feed/classes.ts` + `.spec.ts`
- `frontend/src/components/prompts/activity-feed/activity-event.tsx`
- `frontend/src/components/prompts/activity-feed/activity-feed.tsx`
- `frontend/src/components/prompts/delete-confirm-dialog/delete-confirm-dialog.tsx`
- `frontend/src/components/prompts/restore-confirm-dialog/restore-confirm-dialog.tsx`
- `frontend/src/components/prompts/prompt-sidebar/prompt-sidebar.tsx`
- `frontend/src/components/prompts/prompt-editor/prompt-editor.tsx`
- `frontend/src/routes/settings/prompts/index.tsx`

**Dependencies Added:**

- `npm install marked diff-match-patch` (removed diff-match-patch after rewrite)
- `npm install -D @tailwindcss/typography`

**Tailwind Config:**

- `frontend/src/global.css` — added `@plugin "@tailwindcss/typography"`

### Test Results

| Suite | Files | Tests | Status |
| ------- | ------- | ------- | -------- |
| lib/markdown.spec.ts | 1 | 10 | ✓ PASS |
| lib/diff.spec.ts | 1 | 9 | ✓ PASS |
| lib/prompts-api.spec.ts | 1 | 11 | ✓ PASS |
| components/classes.spec.ts | 3 | 15 | ✓ PASS |
| **Frontend total** | **53** | **504** | **✓ PASS** |

**Backend (short):** all packages OK

### Build

- `npm run build` — ✓ PASS (both client + SSR bundles)

### Deviations from Design

- Removed `diff-match-patch` dependency — replaced with custom
  LCS-based `computeLineDiff()` to avoid character-level diffing issues
- Used `@tailwindcss/typography` via CSS `@plugin` directive (Tailwind 4 configuration)
- No integration tests written (require running Postgres via `make test/integration`)

### Next Steps

1. Run `make test/integration` with compose Postgres up to verify integration tests
2. Open PR for review
3. Sync canonical spec: `openspec/specs/frontend-prompts/spec.md`
4. Archive to `openspec/changes/archive/2026-07-16-prompts-frontend/`
