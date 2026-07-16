# SDD Tasks — 2026-07-16-prompts-frontend

**Change:** Frontend Prompts UI — "Agentic Control Center"  
**Date:** 2026-07-16  
**Status:** TASKS  
**Parent design:** `openspec/changes/2026-07-16-prompts-frontend/design.md`

---

## 1. Forecast

| Metric | Estimate |
| -------- | ---------- |
| Total files | ~16 new files |
| Total lines | ~1,400 lines |
| PR count | 1 PR (within 450-line budget) |
| Test functions | ~50 |
| New npm packages | 3 (`marked`, `diff-match-patch`, `@tailwindcss/typography`) |

---

## 2. PR Scope

Single PR: **frontend prompts UI v1**

This is a frontend-only change. All backend API endpoints already exist from PR #48.

---

## 3. Task Breakdown

### Phase 1: Infrastructure

- [ ] **T-1.1** Install npm packages  

  ```bash
  npm install marked diff-match-patch
  npm install -D @tailwindcss/typography @types/diff-match-patch
  ```

- [ ] **T-1.2** Update `tailwind.config.ts` — add `@tailwindcss/typography` plugin

- [ ] **T-1.3** Create `frontend/src/lib/markdown.ts` — `renderMarkdown()` function  
  - RED: test `renderMarkdown("# Hello")` returns `<h1>Hello</h1>`
  - GREEN: implement with `marked`
  - REFACTOR: add SSR safety note

- [ ] **T-1.4** Create `frontend/src/lib/diff.ts` — `computeLineDiff()` function  
  - RED: test diff produces correct delete/insert/equal lines
  - GREEN: implement with `diff-match-patch`
  - REFACTOR: extract type definitions

- [ ] **T-1.5** Create `frontend/src/lib/prompts-api.ts` — API client functions  
  - RED: test `listPrompts()`, `getPrompt()`, `createPrompt()`, `updatePrompt()`, `deletePrompt()`, `listRevisions()`, `getRevision()`, `restoreRevision()`
  - GREEN: implement using `serverAwareFetch()` pattern from `api.ts`
  - REFACTOR: extract wire types to top of file

---

### Phase 2: Classes (Pure Logic)

- [ ] **T-2.1** Create `frontend/src/components/prompts/prompt-list-item/classes.ts`  
  - Logic: `listItemClasses(selected: boolean)` → `{ container, label, meta }`
  - RED: 3 unit tests (default, selected, hover)
  - GREEN: implement Tailwind classes

- [ ] **T-2.2** Create `frontend/src/components/prompts/prompt-list-item/classes.spec.ts`

- [ ] **T-2.3** Create `frontend/src/components/prompts/diff-viewer/classes.ts`  
  - Logic: `diffLineClasses(type: "delete" | "insert" | "equal")` → CSS class strings
  - RED: 3 unit tests
  - GREEN: implement Tailwind classes (red-50, green-50, white)

- [ ] **T-2.4** Create `frontend/src/components/prompts/diff-viewer/classes.spec.ts`

- [ ] **T-2.5** Create `frontend/src/components/prompts/activity-feed/classes.ts`  
  - Logic: `activityEventClasses(type: EventType)` → `{ icon, text }`
  - RED: 4 unit tests (created, edited, restored, deleted)

- [ ] **T-2.6** Create `frontend/src/components/prompts/activity-feed/classes.spec.ts`

---

### Phase 3: Core Components

- [ ] **T-3.1** Create `frontend/src/components/prompts/empty-state/empty-state.tsx`  
  - Props: `onCreate$` handler
  - RED: test renders with CTA button
  - GREEN: implement with prompt icon (SVG), heading, subtext, Button

- [ ] **T-3.2** Create `frontend/src/components/prompts/prompt-list-item/prompt-list-item.tsx`  
  - Props: `prompt`, `selected`, `onClick$`
  - RED: test renders slug, revision, timestamp
  - GREEN: implement with `listItemClasses()`

- [ ] **T-3.3** Create `frontend/src/components/prompts/prompt-sidebar/prompt-sidebar.tsx`  
  - Props: `prompts[]`, `selectedSlug`, `onSelect$`, `onNewPrompt$`
  - RED: test renders list, filter input, new prompt button
  - GREEN: implement with `PromptListItem`

- [ ] **T-3.4** Create `frontend/src/components/prompts/markdown-textarea/markdown-textarea.tsx`  
  - Props: `value`, `onInput$`, `placeholder`
  - RED: test textarea renders with value
  - GREEN: implement with `w-full h-full resize-none` + border

- [ ] **T-3.5** Create `frontend/src/components/prompts/markdown-preview/markdown-preview.tsx`  
  - Props: `body: string`
  - RED: test renders with `dangerouslySetInnerHTML`
  - GREEN: implement with `renderMarkdown()` + `prose prose-sm`

- [ ] **T-3.6** Create `frontend/src/components/prompts/diff-viewer/diff-block.tsx`  
  - Props: `oldRevision`, `newRevision`, `onRestore$`
  - RED: test renders diff lines with correct classes
  - GREEN: implement with `computeLineDiff()` + `diffLineClasses()`

- [ ] **T-3.7** Create `frontend/src/components/prompts/diff-viewer/diff-viewer.tsx`  
  - Props: `revisions[]`, `currentRevision`, `promptSlug`
  - RED: test collapsible panel, expand/collapse toggle
  - GREEN: implement with expand/collapse signal

- [ ] **T-3.8** Create `frontend/src/components/prompts/activity-feed/activity-event.tsx`  
  - Props: `event: PromptActivityEvent`
  - RED: test renders correct icon and text for each event type
  - GREEN: implement with `activityEventClasses()`

- [ ] **T-3.9** Create `frontend/src/components/prompts/activity-feed/activity-feed.tsx`  
  - Props: `events[]`
  - RED: test renders list of events, empty state
  - GREEN: implement with `ActivityEvent`

- [ ] **T-3.10** Create `frontend/src/components/prompts/delete-confirm-dialog/delete-confirm-dialog.tsx`  
  - Props: `promptSlug`, `onConfirm$`, `onCancel$`
  - RED: test renders confirmation text
  - GREEN: implement with `role="dialog"`, `aria-modal`, Button destructive

- [ ] **T-3.11** Create `frontend/src/components/prompts/restore-confirm-dialog/restore-confirm-dialog.tsx`  
  - Props: `revisionNumber`, `onConfirm$`, `onCancel$`
  - RED: test renders confirmation text
  - GREEN: implement with `role="dialog"`

- [ ] **T-3.12** Create `frontend/src/components/prompts/prompt-editor/prompt-editor.tsx`  
  - Props: `prompt`, `mode: "edit" | "create"`, `onSave$`, `onCancel$`, `onDelete$`
  - RED: test renders editor layout, split pane, save/cancel
  - GREEN: implement with `MarkdownTextarea` + `MarkdownPreview` + `DiffViewer` + `ActivityFeed`

---

### Phase 4: Route and Orchestration

- [ ] **T-4.1** Create `frontend/src/routes/settings/prompts/index.tsx`  
  - `onRequest`: cookie capture, auth/ownboarding guards
  - `routeLoader$`: SSR fetch of prompt list
  - State signals: `selectedSlug`, `mode`, `prompts`, `editingPrompt`
  - Render: `PromptSidebar` + `PromptEditor` + `EmptyState`
  - RED: test loading, empty, loaded, error states
  - GREEN: implement full orchestrator

- [ ] **T-4.2** Create `frontend/src/routes/settings/prompts/index.spec.tsx`  
  - Integration tests for route states

- [ ] **T-4.3** Update `frontend/src/routes/settings/` — add `settings` parent route if needed  
  - Check if `/settings` layout exists; if not, create minimal layout

---

## 4. Cross-Cutting Tasks

- [ ] **T-5.1** Run `make lint` — fix any lint errors introduced
- [ ] **T-5.2** Run `make test` — ensure all tests pass
- [ ] **T-5.3** Manual browser test: verify `/settings/prompts` loads, create/edit/delete flows work
- [ ] **T-5.4** Check pi-lens diagnostics for new files

---

## 5. PR Checklist (Reviewer)

### Before opening PR

- [ ] All 50+ tests pass (`make test`)
- [ ] `make lint` clean (0 issues in new files)
- [ ] `make build` succeeds
- [ ] No console errors in browser

### PR Description

- [ ] Summary: what changed and why
- [ ] Screenshots of the UI (before/after or new UI)
- [ ] Test evidence: `make test` output
- [ ] Link to `openspec/specs/prompts/spec.md`
- [ ] Note: backend unchanged, frontend-only

---

## 6. Implementation Order (for reference)

```
T-1.1 → T-1.2 → T-1.3 → T-1.4 → T-1.5
  → T-2.1 → T-2.2 → T-2.3 → T-2.4 → T-2.5 → T-2.6
  → T-3.1 → T-3.2 → T-3.3 → T-3.4 → T-3.5 → T-3.6 → T-3.7 → T-3.8 → T-3.9 → T-3.10 → T-3.11 → T-3.12
  → T-4.1 → T-4.2 → T-4.3
  → T-5.1 → T-5.2 → T-5.3 → T-5.4
```

---

## 7. Review Workload Assessment

| Phase | Files | Lines | Risk |
| ------- | ------- | ------- | ------ |
| Infrastructure (T-1) | 4 | ~200 | Low |
| Classes (T-2) | 6 | ~150 | Low |
| Components (T-3) | 12 | ~600 | Medium |
| Route (T-4) | 2 | ~350 | Medium |
| **Total** | **~24** | **~1,300** | **Within budget** |

**Risk: Low-Medium.** Components are well-isolated (each has its own test file). The route is the highest-risk component but follows existing patterns exactly.
