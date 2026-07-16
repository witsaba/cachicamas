# SDD Explore — 2026-07-16-prompts-frontend

**Change:** Frontend Prompts UI — "Agentic Control Center" pattern  
**Date:** 2026-07-16  
**Status:** EXPLORE

---

## 1. What Exists

### Backend (PR #48, merged 2026-07-15)

| Aspect | Detail |
| -------- | -------- |
| Tables | `prompt` + `prompt_revision` |
| API base | `/prompts/*` |
| Canonical spec | `openspec/specs/prompts/spec.md` |
| Code | `backend/database_administrator/src/interfaces/http/prompt_handler.go` |
| Auth | Admin-only, no extra header |

**API surface:**

```
GET    /prompts                    → list (limit, offset, deleted bool)
POST   /prompts                    → create { slug, description, body }
GET    /prompts/:slug              → get current revision
PUT    /prompts/:slug              → update (creates new revision)
DELETE /prompts/:slug              → soft-delete (410 on re-fetch)
GET    /prompts/:slug/revisions    → list all revisions
GET    /prompts/:slug/revisions/:n → get specific revision
POST   /prompts/:slug/revisions/:n/restore → restore as new revision
```

**Wire shapes (from backend domain):**

- `Prompt` — id, slug, description, body (markdown), current_revision, created_at, updated_at, deleted_at
- `PromptRevision` — id, prompt_id, revision_number, body, created_at

### Frontend

**Gap:** No UI exists yet. Routes, components, and API client functions are all missing.

**Existing patterns to follow:**

- Qwik SSR-first, max-width `max-w-3xl` for content pages
- Tailwind design system (Button with 4 variants, status pills with color states)
- `WorkspaceSyncCard` as the closest analog: activity panel + polling + status
- API client pattern: discriminated `ApiResult<T>` with `ok: true | false`
- SSR cookie forwarding via `setSsrCookieHeader`
- `routeLoader$` + `useTask$` for data fetching
- Confirmation dialogs for destructive actions

---

## 2. Web Research — Sources

| # | Source | Key Insight |
| --- | -------- | ------------- |
| 1 | Zylos AI — Agentic UX Patterns 2026 | Activity panel must be SEPARATE from conversation. Progressive disclosure: collapsed = step-level, expanded = tool-level. Plan-and-execute previews. |
| 2 | Cloudscape Design Patterns | Service dashboard patterns: summary card → detail expansion → action panel |
| 3 | Zypsy — Prompt Management UI | Semantic versioning for prompts. Human-readable diffs. Prompt ops as control plane for AI safety |
| 4 | LangChain PromptHub | Prompt versioning UI: list → preview → diff → restore |
| 5 | Qwik Markdown editors | No existing Qwik markdown component — need custom implementation or a vanilla-JS library |
| 6 | Tailwind typography plugin | `@tailwindcss/typography` for markdown rendering (prose) |
| 7 | diff-match-patch (Google) | Standard library for computing human-readable text diffs |
| 8 | Shiki / highlight.js | Syntax highlighting for markdown code blocks |
| 9 | GitHub Copilot prompt studio | Split-pane: sidebar list + editor + diff viewer |
| 10 | Linear / Vercel settings | Global settings page pattern: section nav + content panel |

---

## 3. Open Questions (must be answered before proposal)

### Q1 — Routing URL

Where should the page live?

| Option | URL | Notes |
| -------- | ----- | ------- |
| A | `/settings/prompts` | Under system settings (best for global/admin config) |
| B | `/prompts` | Top-level (clean, but risks collision with future user-level prompts) |
| C | `/admin/prompts` | Explicitly admin-scoped |

**Recommendation: Option A** (`/settings/prompts`). Mirrors `/settings` conventions from Linear/Vercel. The "Settings" umbrella is the right mental model for system-level config.

### Q2 — Editor UX

What level of markdown editing support?

| Option | Detail |
| -------- | -------- |
| A | Plain textarea (no preview, no syntax highlighting) |
| B | Textarea + side-by-side rendered preview |
| C | Rich markdown editor (toolbar, live preview, syntax highlighting) |

**Recommendation: Option B.** Option A is too primitive for prompt engineering. Option C adds too much complexity for v1. Side-by-side preview is the standard (GitHub, Notion, Linear).

### Q3 — Diff Viewer

How to show version comparison?

| Option | Detail |
| -------- | -------- |
| A | No diff — just version list + restore button |
| B | Unified diff (side-by-side text diff) |
| C | Inline diff with color highlights |

**Recommendation: Option B.** Unified side-by-side diff is the minimum viable "transparency" pattern from agentic UX research. GitHub-style red/green line diff.

### Q4 — Activity Feed Scope

What goes in the activity feed?

| Option | Detail |
| -------- | -------- |
| A | All prompt events across all prompts (audit log) |
| B | Per-prompt events only |
| C | No activity feed (skip for v1) |

**Recommendation: Option B.** Per-prompt activity feed gives the user context without overwhelming them. A global audit log can be added later.

### Q5 — Empty State

What to show when no prompts exist?

| Option | Detail |
| -------- | -------- |
| A | Empty state with "Create your first prompt" CTA |
| B | Redirect to a "create prompt" flow automatically |
| C | Placeholder prompts seeded on first load |

**Recommendation: Option A.** Explicit empty state with CTA. User should control when prompts are created.

---

## 4. Constraints

- **Tech:** Qwik + Tailwind CSS, no new top-level deps without ADR
- **Max-width:** `max-w-3xl` for content (matches existing pages)
- **API client:** Must use the existing `ApiResult<T>` pattern from `lib/api.ts`
- **SSR:** Must use `routeLoader$` + `useTask$` pattern, cookie forwarding
- **TDD:** Strict TDD per `openspec/config.yaml` — `go test ./...` not applicable (this is frontend), but unit tests for classes.ts + integration tests for the route
- **Design system:** Reuse `Button`, status pills, confirmation dialog patterns
- **No backend changes** — this is a frontend-only change

---

## 5. Next

Move to **proposal** phase with Q1-Q5 answered (or defaulted to recommendations).
