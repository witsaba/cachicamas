# SDD Proposal — 2026-07-16-prompts-frontend

**Change:** Frontend Prompts UI — "Agentic Control Center"  
**Date:** 2026-07-16  
**Status:** PROPOSAL  
**Artifact store:** filesystem (engram unavailable)

---

## 1. Problem Statement

Cachicamas has a backend API for managing AI prompts (`/prompts/*`, PR #48, merged 2026-07-15), but no frontend UI exists. Users cannot view, create, edit, or version-control prompts without direct API calls. This is a critical gap for prompt engineering workflows.

---

## 2. Goals

### In Scope

| ID | Goal |
| ---- | ------ |
| G-1 | Users can list all prompts at `/settings/prompts` |
| G-2 | Users can create a new prompt (slug + description + markdown body) |
| G-3 | Users can view a prompt with its current body rendered as markdown |
| G-4 | Users can edit a prompt body (creates a new revision) |
| G-5 | Users can view version history (collapsed by default, expandable) |
| G-6 | Users can restore a previous revision (creates a new revision) |
| G-7 | Users can soft-delete a prompt |
| G-8 | Activity feed shows per-prompt events (created, edited, restored, deleted) |

### Out of Scope

- User-level prompts (per-user prompt configurations)
- Prompt templates with variable interpolation
- Prompt testing / sandbox / preview against LLM
- Export / import prompts
- Prompt categories or tags
- Bulk operations

---

## 3. User Stories

### US-1: View Prompt List

**As** an admin user  
**I want** to see all prompts at `/settings/prompts`  
**So that** I know what system prompts are configured

Acceptance: A list shows slug, description, current revision number, last updated timestamp. Empty state shows CTA.

### US-2: Create Prompt

**As** an admin user  
**I want** to create a new prompt  
**So that** I can define AI behavior for the system

Acceptance: Form with slug (unique), description, markdown body. Side-by-side preview. Save creates v1.

### US-3: Edit Prompt Body

**As** an admin user  
**I want** to edit a prompt's markdown body  
**So that** I can improve AI behavior without losing history

Acceptance: Editor with current body. Side-by-side preview updates live. Save creates a new revision.

### US-4: View History (Collapsed)

**As** an admin user  
**I want** to see version history only when I ask for it  
**So that** the UI stays clean and focused

Acceptance: "History" button expands a panel showing all revisions. Each revision shows: version number, created_at, first 3 lines of body. Click to expand full diff.

### US-5: Restore Revision

**As** an admin user  
**I want** to restore a previous revision  
**So that** I can revert to a known-good prompt state

Acceptance: "Restore" button on each historical revision. Confirmation dialog. Creates a NEW revision (doesn't overwrite history).

### US-6: Delete Prompt

**As** an admin user  
**I want** to soft-delete a prompt  
**So that** I can remove deprecated prompts without losing history

Acceptance: Delete button with confirmation dialog. Soft-delete (410 on re-fetch). Deleted prompts don't appear in the list.

### US-7: Activity Feed

**As** an admin user  
**I want** to see per-prompt activity events  
**So that** I can audit what changed and when

Acceptance: Activity feed in the prompt detail view. Shows: created, body edits, restores, deletes. Sorted newest first.

---

## 4. API Contract (from `openspec/specs/prompts/spec.md`)

### Wire Shapes

```typescript
// Prompt (current state)
interface Prompt {
  id: number;
  slug: string;
  description: string | null;
  body: string;           // markdown
  current_revision: number;
  created_at: string;       // ISO-8601
  updated_at: string;       // ISO-8601
  deleted_at: string | null;
}

// PromptRevision (historical)
interface PromptRevision {
  id: number;
  prompt_id: number;
  revision_number: number;
  body: string;            // markdown
  created_at: string;      // ISO-8601
}

// Activity event (derived from API, no explicit endpoint)
interface PromptActivityEvent {
  type: "created" | "edited" | "restored" | "deleted";
  revision_number: number;
  timestamp: string;
}
```

### Endpoints Used

| Method | Path | Used For |
| -------- | ------ | --------- |
| GET | `/prompts?deleted=false` | List all non-deleted prompts |
| POST | `/prompts` | Create prompt |
| GET | `/prompts/:slug` | Get prompt detail |
| PUT | `/prompts/:slug` | Update prompt body |
| DELETE | `/prompts/:slug` | Soft-delete prompt |
| GET | `/prompts/:slug/revisions` | List revisions |
| GET | `/prompts/:slug/revisions/:n` | Get specific revision |
| POST | `/prompts/:slug/revisions/:n/restore` | Restore revision |

### Error Handling

| Kind | Meaning |
| ------ | --------- |
| `validation` | 400 — invalid input (slug regex, body empty) |
| `conflict` | 409 — slug already taken |
| `not_found` | 404 — prompt not found |
| `server` | 500 — internal error |

Special: HTTP 410 (Gone) when fetching a soft-deleted prompt. API client should handle this as `not_found` in the UI.

---

## 5. Page Layout

### `/settings/prompts` — Prompt Studio

```
┌─ /settings ───────────────────────────────────────────────────────┐
│  [Back to settings]                                                │
│                                                                    │
│  Prompts                                          [+ New Prompt]   │
│                                                                    │
│  ┌─ Prompt Studio ───────────────────────────────────────────┐   │
│  │                                                            │   │
│  │  ┌─ Sidebar (list) ───────┐  ┌─ Editor Panel ──────────┐ │   │
│  │  │ [🔍 filter prompts]    │  │                          │ │   │
│  │  │                        │  │  slug: ___________         │ │   │
│  │  │ ○ workspace-default   │  │  description: _______   │ │   │
│  │  │   v3  updated: today  │  │                          │ │   │
│  │  │                        │  │  ┌─ Body ─────────────┐  │ │   │
│  │  │ ○ review-prompt        │  │  │      │  Preview    │  │ │   │
│  │  │   v1  updated: Jul 10 │  │  │ Txt  │  (Markdown) │  │ │   │
│  │  │                        │  │  │Area │  Rendered   │  │ │   │
│  │  │ [+ New Prompt]         │  │  └──────┴────────────┘  │ │   │
│  │  └────────────────────────┘  │                          │ │   │
│  │                              │  [History ▾]              │ │   │
│  │                              │  ──────────────────────── │ │   │
│  │                              │  v3 — current            │ │   │
│  │                              │  v2 — Jul 15 14:32        │ │   │
│  │                              │  v1 — Jul 10 09:15       │ │   │
│  │                              │                          │ │   │
│  │                              │  [Cancel]  [Save as v4 →] │ │   │
│  │                              └──────────────────────────┘ │   │
│  │                                                            │   │
│  │  ┌─ Activity Feed ──────────────────────────────────────┐ │   │
│  │  │ ✓ 14:32 — v3 saved (current)                        │ │   │
│  │  │ ✓ 14:28 — v2 restored                               │ │   │
│  │  │ ✓ 09:15 — v1 created                                │ │   │
│  │  └───────────────────────────────────────────────────────┘ │   │
│  └────────────────────────────────────────────────────────────┘   │
└───────────────────────────────────────────────────────────────────┘
```

### Empty State (no prompts)

```
┌─ /settings/prompts ───────────────────────────────────────────────┐
│  [Back to settings]                                                │
│  Prompts                                           [+ New Prompt]   │
│                                                                     │
│  ┌─ Empty State ───────────────────────────────────────────────┐   │
│  │                                                              │   │
│  │     [Prompt icon]                                           │   │
│  │                                                              │   │
│  │     No prompts yet                                          │   │
│  │     System prompts control how the AI behaves.              │   │
│  │                                                              │   │
│  │     [Create your first prompt]                              │   │
│  │                                                              │   │
│  └──────────────────────────────────────────────────────────────┘   │
└────────────────────────────────────────────────────────────────────┘
```

### Diff Panel (expanded from History ▾)

```
┌─ History ─────────────────────────────────────────────────────────┐
│  ┌─ v2 vs v3 ──────────────────────────────────────────────────┐  │
│  │  - old line 1  ← red background                             │  │
│  │  - old line 2                                               │  │
│  │  + new line 1  ← green background                          │  │
│  │  + new line 2                                               │  │
│  │    (unchanged lines)                                        │  │
│  │  [Restore v2]                                               │  │
│  └──────────────────────────────────────────────────────────────┘  │
│  ┌─ v1 vs v2 ──────────────────────────────────────────────────┐  │
│  │  ...                                                        │  │
│  └──────────────────────────────────────────────────────────────┘  │
│  [Collapse]                                                        │
└─────────────────────────────────────────────────────────────────────┘
```

---

## 6. Component Inventory

| Component | Purpose | States |
| ----------- | --------- | -------- |
| `PromptStudio` | Route component, layout orchestrator | loading, loaded, error, empty |
| `PromptSidebar` | Left panel — filterable list | default, selected, loading |
| `PromptListItem` | Single prompt in sidebar | default, selected, hover |
| `PromptEditor` | Right panel — edit/create form | edit, create, saving, saved |
| `MarkdownTextarea` | Left side of editor | default, focus, error |
| `MarkdownPreview` | Right side — rendered markdown | default, loading |
| `DiffViewer` | Collapsed history panel | collapsed, expanded, empty |
| `DiffBlock` | Single revision diff (vN vs vN-1) | default, expanded |
| `ActivityFeed` | Bottom panel | default, loading, empty |
| `ActivityEvent` | Single event in feed | created, edited, restored, deleted |
| `EmptyState` | No prompts yet | — |
| `CreatePromptModal` | Quick-create form | open, saving, error |
| `DeleteConfirmDialog` | Confirmation for delete | open, confirming |
| `RestoreConfirmDialog` | Confirmation for restore | open, confirming |

---

## 7. Routing

| Route | Component | Guards |
|-------|-----------|--------|
| `/settings/prompts` | `PromptStudio` | auth + ownboarding |
| `/settings/prompts/new` | `PromptStudio` (create mode) | auth + ownboarding |

**No backend changes required.** Auth is admin-only via existing middleware.

---

## 8. Technical Decisions

| ID | Decision | Rationale |
| ---- | ---------- | -------- |
| D-1 | Markdown: use `@tailwindcss/typography` plugin + `marked` | No Qwik-specific markdown lib; use vanilla + Tailwind prose |
| D-2 | Diff: implement custom side-by-side diff using `diff-match-patch` | Standard Google lib, no framework dependency |
| D-3 | State: Qwik signals (`useSignal`) for local UI state | Matches existing patterns |
| D-4 | SSR: `routeLoader$` for initial prompt list | SSR-first, matches workspace detail page |
| D-5 | No new backend endpoints | All needed endpoints exist from PR #48 |
| D-6 | Activity feed: derived from revision list (no new API) | revisions endpoint gives all info needed |

---

## 9. Edge Cases

| Case | Handling |
| ------ | ---------- |
| Slug conflict on create | Show `conflict` error inline under slug field |
| Empty body on save | Show `validation` error inline |
| Delete last prompt | Allowed — shows empty state |
| Restore current revision | No-op button (vN is already current) |
| Concurrent edit (race) | Last-write-wins (matching backend behavior) |
| Network failure | Show `offline` error with retry button |
| Long prompt body | Textarea scrolls, preview scrolls independently |
| Special chars in slug | Backend validates via regex — show validation error |

---

## 10. Rollback Plan

If this change needs to be reverted:

1. Revert the frontend code changes (remove the new route, components, and API client additions)
2. No database migration needed (backend is unchanged)
3. No data loss — prompts data in DB is untouched

---

## 11. Non-Goals (Explicit)

- No prompt execution / testing against LLM
- No prompt variables / interpolation
- No user-level prompts
- No bulk operations
- No prompt sharing or versioning beyond what the backend provides
- No changes to backend API

---

## 12. Dependencies

| Dependency | Status |
| ------------ | -------- |
| Backend API (`/prompts/*`) | ✅ Done (PR #48) |
| `openspec/specs/prompts/spec.md` | ✅ Done |
| `frontend/src/lib/api.ts` | ✅ Needs new client functions |
| `@tailwindcss/typography` | ⚠️ Needs installation + config |
| `marked` npm package | ⚠️ Needs installation |

---

## 13. Risks

| Risk | Severity | Mitigation |
| ------ | ---------- | ------------ |
| Qwik markdown rendering hydration mismatch | Low | SSR-safe: render preview only on client or use `useVisibleTask$` |
| Tailwind typography plugin not configured | Medium | Add to `tailwind.config.ts` in this change |
| Long prompt bodies cause layout issues | Low | CSS `overflow-y: auto` on both panes |
| Diff lib bundle size | Low | `diff-match-patch` is ~15KB gzipped, acceptable |
