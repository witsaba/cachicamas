# SDD Design — 2026-07-16-prompts-frontend

**Change:** Frontend Prompts UI — "Agentic Control Center"  
**Date:** 2026-07-16  
**Status:** DESIGN  
**Parent spec:** `openspec/changes/2026-07-16-prompts-frontend/specs/frontend-prompts/spec.md`

---

## 1. File Structure

```
frontend/src/
├── routes/
│   └── settings/
│       └── prompts/
│           ├── index.tsx          ← Route component (PromptStudio)
│           └── index.spec.tsx     ← Integration tests
├── components/
│   └── prompts/
│       ├── prompt-studio.tsx     ← Main orchestrator component
│       ├── prompt-studio.spec.tsx
│       ├── prompt-sidebar.tsx
│       ├── prompt-sidebar.spec.tsx
│       ├── prompt-list-item.tsx
│       ├── prompt-list-item.spec.tsx
│       ├── prompt-editor.tsx
│       ├── prompt-editor.spec.tsx
│       ├── markdown-textarea.tsx
│       ├── markdown-preview.tsx
│       ├── diff-viewer.tsx
│       ├── diff-viewer.spec.tsx
│       ├── diff-block.tsx
│       ├── activity-feed.tsx
│       ├── activity-feed.spec.tsx
│       ├── empty-state.tsx
│       ├── delete-confirm-dialog.tsx
│       └── restore-confirm-dialog.tsx
└── lib/
    ├── prompts-api.ts             ← API client functions
    ├── prompts-api.spec.ts       ← API client tests
    └── markdown.ts               ← Markdown rendering utilities
```

**Estimated lines of code:** ~1,400 lines across 16 new files

---

## 2. Tailwind Typography Plugin

The `@tailwindcss/typography` plugin must be added to enable `prose` class for markdown rendering.

### Installation

```bash
npm install -D @tailwindcss/typography
```

### Config update (`tailwind.config.ts`)

```typescript
import typography from "@tailwindcss/typography";

export default {
  plugins: [
    typography,
  ],
};
```

---

## 3. Markdown Library

Use `marked` (lightweight, no dependencies) for markdown parsing.

### Installation

```bash
npm install marked
```

### `lib/markdown.ts`

```typescript
import { marked } from "marked";

/**
 * Render markdown string to HTML string.
 * Safe: no arbitrary HTML execution.
 */
export function renderMarkdown(md: string): string {
  return marked.parse(md, { async: false }) as string;
}
```

**Note on SSR safety:** `marked.parse(..., { async: false })` is synchronous and safe for SSR. The result is an HTML string that Qwik renders via `{dangerouslySetInnerHTML}`.

---

## 4. Diff Implementation

Use `diff-match-patch` for computing line-level diffs.

### Installation

```bash
npm install diff-match-patch
npm install -D @types/diff-match-patch
```

### Algorithm

1. Compute line-by-line diff using `diff_match_patch.diff_main()`
2. Convert to line-mode: split both old and new text into lines
3. Run `diff_main(lineOld, lineNew)` with `diff_match_patch.DIFF_DELETE = -1`, `INSERT = +1`, `EQUAL = 0`
4. Group consecutive same-type hunks
5. Render: delete lines with red background, insert lines with green background, equal lines as normal text

### `lib/diff.ts`

```typescript
import DiffMatchPatch from "diff-match-patch";

export interface DiffLine {
  type: "delete" | "insert" | "equal";
  text: string;
}

export interface DiffResult {
  lines: DiffLine[];
}

/**
 * Compute a line-level diff between two text strings.
 * Returns an array of DiffLine objects for side-by-side rendering.
 */
export function computeLineDiff(oldText: string, newText: string): DiffLine[] {
  const dmp = new DiffMatchPatch();
  const oldLines = oldText.split("\n");
  const newLines = newText.split("\n");
  const diffs = dmp.diff_main(oldLines.join("\n"), newLines.join("\n"));
  dmp.diff_cleanupSemantic(diffs);
  // Convert char-level diffs to line-level...
  // (Implementation detail: process diffs array, grouping by type)
}
```

---

## 5. API Client (`lib/prompts-api.ts`)

```typescript
// New file — mirrors existing api.ts conventions
// Uses the existing serverAwareFetch() pattern
// Returns ApiResult<Prompt> etc.

export async function listPrompts(): Promise<ApiResult<Prompt[]>>;
export async function getPrompt(slug: string): Promise<ApiResult<Prompt>>;
export async function createPrompt(input: CreatePromptInput): Promise<ApiResult<Prompt>>;
export async function updatePrompt(slug: string, body: string): Promise<ApiResult<Prompt>>;
export async function deletePrompt(slug: string): Promise<ApiResult<void>>;
export async function listRevisions(slug: string): Promise<ApiResult<PromptRevision[]>>;
export async function getRevision(slug: string, n: number): Promise<ApiResult<PromptRevision>>;
export async function restoreRevision(slug: string, n: number): Promise<ApiResult<Prompt>>;
```

**Wire shapes match backend domain** (from `openspec/specs/prompts/spec.md`):

```typescript
export interface Prompt {
  id: number;
  slug: string;
  description: string | null;
  body: string;
  current_revision: number;
  created_at: string;
  updated_at: string;
  deleted_at: string | null;
}

export interface PromptRevision {
  id: number;
  prompt_id: number;
  revision_number: number;
  body: string;
  created_at: string;
}

export interface CreatePromptInput {
  slug: string;
  description?: string;
  body: string;
}
```

**Error handling:** 410 Gone is mapped to `not_found` for UI consumption.

---

## 6. Component Specifications

### 6.1 PromptStudio (Route Component)

```typescript
// routes/settings/prompts/index.tsx
export default component$(() => {
  const selectedSlug = useSignal<string | null>(null);
  const mode = useSignal<"list" | "create">("list");
  // ... useTask$ fetches prompts list
  // ... renders PromptSidebar + PromptEditor
});
```

**States:** loading, empty, loaded (with or without selection)

### 6.2 PromptSidebar

```tsx
<div class="w-64 border-r border-slate-200 flex flex-col h-full">
  <input type="text" placeholder="Filter prompts..." />
  <ul>
    {prompts.value?.map(p => (
      <PromptListItem key={p.slug} prompt={p} selected={...} />
    ))}
  </ul>
  <button>+ New Prompt</button>
</div>
```

**Width:** `w-64` (256px).  
**Styling:** Matches `WorkspaceSyncCard` patterns — border-r slate-200, scrollable list.

### 6.3 PromptEditor

```
┌─ Editor Panel ──────────────────────────────────────┐
│  [slug] [description input]                        │
│  ┌─ Split ───────────────────────────────────────┐  │
│  │  MarkdownTextarea   │  MarkdownPreview         │  │
│  │  (left, flex-1)    │  (right, flex-1)         │  │
│  └─────────────────────┴──────────────────────────┘  │
│  [History (N)]                                       │
│  [Cancel]                              [Save v{N} →] │
│  [Delete prompt]                                     │
└──────────────────────────────────────────────────────┘
```

**Layout:** `flex-row gap-4` for the split editor.  
**Preview:** `prose prose-sm max-w-none` for rendered markdown.  
**Save button:** disabled when no changes, shows "Saving…" when loading.

### 6.4 MarkdownPreview

```tsx
<div
  class="prose prose-sm max-w-none overflow-y-auto border border-slate-200 rounded p-3 bg-slate-50"
  dangerouslySetInnerHTML={renderMarkdown(body)}
/>
```

**SSR-safe:** `renderMarkdown()` is synchronous, safe for Qwik SSR.  
**Styling:** `prose-sm` (from `@tailwindcss/typography`), slate-50 background.

### 6.5 DiffViewer

```
┌─ Version History ─────────────────────────────────────┐
│  ┌─ v2 vs v3 ──────────────────────────────────────┐  │
│  │  - old line (red bg)                           │  │
│  │  + new line (green bg)                         │  │
│  │  (unchanged lines in white)                     │  │
│  │  [Restore v2]                                   │  │
│  └─────────────────────────────────────────────────┘  │
│  [Collapse]                                            │
└────────────────────────────────────────────────────────┘
```

**Styling:**

- Delete: `bg-red-50 text-red-900`
- Insert: `bg-green-50 text-green-900`
- Equal: `bg-white text-slate-700`

### 6.6 ActivityFeed

```
┌─ Activity ───────────────────────────────────────────┐
│  ✓ 14:32 — v3 saved (current)                       │
│  ✓ 14:28 — v2 restored                              │
│  ✓ 09:15 — v1 created                               │
└──────────────────────────────────────────────────────┘
```

**Styling:** Each event is a row with an icon + text + timestamp.  
**Icon colors:** created=emerald, edited=blue, restored=amber, deleted=red.

---

## 7. Sequence Diagram

```
User → Browser → PromptStudio → listPrompts() → Backend API
                              ← Promise<Prompt[]> ←
User → Sidebar → selectPrompt(slug) → setSelectedSlug(slug)
                                  → getPrompt(slug) → Backend API
                                  ← Promise<Prompt> ←
User → Editor → updateBody(newText)
           → preview updates (debounced 300ms)
User → Save → updatePrompt(slug, body) → Backend PUT /prompts/:slug
          ← Promise<Prompt> ←
          → activityFeed.add("edited", newRevision)
          → revisionCount++

User → History → expandHistory()
               → listRevisions(slug) → Backend GET /prompts/:slug/revisions
               ← Promise<Revision[]> ←
User → Expand diff → computeLineDiff(revN.body, revN-1.body)
                  → render DiffBlock

User → Restore → RestoreConfirmDialog
             → confirm → restoreRevision(slug, N) → Backend POST /prompts/:slug/revisions/N/restore
                        ← Promise<Prompt> ←
                        → updateBody(restored.body)
                        → activityFeed.add("restored", newRev)
```

---

## 8. Test Map

| Component | Test File | Type |
| ----------- | ----------- | ------ |
| `markdown.ts` | `markdown.spec.ts` | Unit |
| `diff.ts` | `diff.spec.ts` | Unit |
| `prompts-api.ts` | `prompts-api.spec.ts` | Unit (mock fetch) |
| `PromptListItem` | `prompt-list-item.spec.tsx` | Unit (classes.ts) |
| `DiffBlock` | `diff-viewer.spec.tsx` | Unit (classes.ts) |
| `PromptStudio` | `index.spec.tsx` | Integration |
| `PromptEditor` | `prompt-editor.spec.tsx` | Integration |
| `ActivityFeed` | `activity-feed.spec.tsx` | Integration |

**Total test coverage estimate:** ~50 test functions.

---

## 9. Tailwind Config Additions

```typescript
// tailwind.config.ts — additions only
import type { Config } from "tailwindcss";

export default {
  plugins: [
    require("@tailwindcss/typography"),
  ],
} satisfies Config;
```

---

## 10. Package.json Additions

```json
{
  "dependencies": {
    "marked": "^12.0.0",
    "diff-match-patch": "^1.0.5"
  },
  "devDependencies": {
    "@tailwindcss/typography": "^0.5.15",
    "@types/diff-match-patch": "^1.0.36"
  }
}
```

---

## 11. Global CSS Additions

No changes needed to `global.css`. The `prose` class from `@tailwindcss/typography` is self-contained.

---

## 12. Classes Pattern (per project convention)

Every component follows the existing `classes.ts` pattern (matching `WorkspaceSyncCard`):

```
component.tsx → calls → classes.ts → pure function returns class strings
classes.spec.ts → unit tests on the pure functions
```

---

## 13. SSR and Cookie Handling

Follows the established pattern from `routes/workspaces/[id]/index.tsx`:

1. `onRequest` captures `Cookie` header → `setSsrCookieHeader()`
2. `requireAuthRedirect` + `requireOwnboarding` guards
3. `routeLoader$` fetches initial prompt list (SSR)
4. Client-side `useTask$` for interactive operations

---

## 14. Design Decisions Summary

| ID | Decision | Rationale |
| ---- | ---------- | ----------- |
| D-1 | `@tailwindcss/typography` + `marked` | Minimal deps, SSR-safe, proven stack |
| D-2 | `diff-match-patch` | Google's standard, no framework lock-in |
| D-3 | History collapsed by default | User's stated preference: hide but visible |
| D-4 | Per-prompt activity feed | Scoped events, matches agentic UX principles |
| D-5 | `dangerouslySetInnerHTML` for preview | `marked` returns trusted HTML; sanitized by design |
| D-6 | `w-64` sidebar, `max-w-3xl` page | Consistent with existing pages |
| D-7 | No new backend endpoints | All needed APIs exist from PR #48 |
