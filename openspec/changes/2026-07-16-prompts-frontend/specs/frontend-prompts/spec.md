# SDD Spec — 2026-07-16-prompts-frontend

**Change:** Frontend Prompts UI — "Agentic Control Center"  
**Date:** 2026-07-16  
**Status:** SPEC  
**Parent proposal:** `openspec/changes/2026-07-16-prompts-frontend/proposal.md`

---

## Conventions

- Keywords: RFC 2119 (MUST, SHALL, SHOULD, MAY)
- Scenarios: Given/When/Then format
- Each scenario is independently verifiable
- UI state machine: `loading` → `loaded` | `error` | `empty`

---

## 1. Page Load

### S-PS-001: Authenticated user loads `/settings/prompts`

**Given** an authenticated, onboarded user  
**When** the user navigates to `/settings/prompts`  
**Then** the page shows a loading state while fetching the prompt list from `GET /prompts?deleted=false`  
**And** the URL remains `/settings/prompts`

### S-PS-002: Prompts exist — list renders

**Given** the backend returns at least one prompt  
**When** `GET /prompts?deleted=false` returns HTTP 200 with a non-empty array  
**Then** the sidebar shows a list of `PromptListItem` components  
**And** the first prompt is selected by default  
**And** the editor panel shows the selected prompt's body

### S-PS-003: No prompts exist — empty state

**Given** the backend returns an empty array  
**When** `GET /prompts?deleted=false` returns HTTP 200 with `[]`  
**Then** the sidebar shows an empty state  
**And** the editor panel shows a "Create your first prompt" empty state  
**And** no prompt is selected in the sidebar

### S-PS-004: Backend returns error on load

**Given** the backend is unavailable  
**When** `GET /prompts?deleted=false` throws an "offline" error  
**Then** the page shows an error alert with the message "Couldn't reach the backend"  
**And** a retry button is shown

### S-PS-005: Prompt is soft-deleted (HTTP 410 → treated as not_found)

**Given** a prompt exists but was soft-deleted  
**When** `GET /prompts/:slug` returns HTTP 410  
**Then** the UI treats it as `not_found`  
**And** the prompt is removed from the list  
**And** if it was selected, the editor shows the empty state

---

## 2. Prompt List (Sidebar)

### S-PL-001: Sidebar shows prompt metadata

**Given** a list of prompts is loaded  
**When** a `PromptListItem` renders  
**Then** it shows the prompt's `slug`  
**And** the current revision number (e.g. "v3")  
**And** the `updated_at` timestamp in compact form (YYYY-MM-DD HH:MM)

### S-PL-002: Sidebar selected state

**Given** the user clicks a prompt in the sidebar  
**When** the click event fires  
**Then** that prompt becomes selected (highlighted background)  
**And** the editor panel loads the selected prompt's data

### S-PS-006: User clicks "+ New Prompt"

**Given** the user is on `/settings/prompts`  
**When** the user clicks "+ New Prompt"  
**Then** the editor panel switches to create mode  
**And** the slug, description, and body fields are empty  
**And** the preview pane is empty  
**And** no sidebar item is selected

---

## 3. Prompt Editor

### S-PE-001: Edit mode — body textarea

**Given** a prompt is selected in the sidebar  
**When** the editor renders in edit mode  
**Then** the slug field is read-only (pre-filled, not editable)  
**And** the description field is a text input  
**And** the body field is a `MarkdownTextarea`  
**And** the preview pane shows the rendered markdown

### S-PE-002: Markdown preview renders correctly

**Given** the body contains markdown syntax  
**When** the user types or loads a prompt body  
**Then** the preview pane renders the markdown as HTML  
**And** headings, lists, code blocks, bold, italic are styled appropriately  
**And** the preview is updated within 300ms of typing (debounced)

### S-PE-003: Save creates new revision

**Given** the user is editing a prompt in edit mode  
**When** the user clicks "Save as v{N}"  
**And** the body has changed  
**Then** `PUT /prompts/:slug` is called with the new body  
**And** on success, the revision number increments  
**And** the activity feed shows a new "edited" event  
**And** a success toast or indicator is shown

### S-PE-004: Save with unchanged body

**Given** the user is in edit mode with no changes  
**When** the user clicks "Save as v{N}"  
**Then** no API call is made  
**And** an inline hint says "No changes to save"

### S-PE-005: Cancel discards changes

**Given** the user has unsaved changes in the editor  
**When** the user clicks "Cancel"  
**Then** the editor reverts to the last saved state  
**And** the preview shows the saved body

### S-PE-006: Validation error on empty body

**Given** the user clears the body field  
**When** the user clicks "Save"  
**Then** a validation error is shown: "Body is required"  
**And** the save is not submitted

### S-PE-007: Validation error on slug conflict

**Given** the user creates a new prompt  
**When** the user enters a slug that already exists  
**And** the user clicks "Save"  
**Then** `POST /prompts` returns 409 Conflict  
**And** a conflict error is shown: "Slug already in use"

---

## 4. Create Prompt

### S-PC-001: Create form renders

**Given** the user is in create mode  
**When** the editor renders  
**Then** the slug field is editable  
**And** the description field is editable  
**And** the body field is a `MarkdownTextarea`  
**And** the preview pane is empty  
**And** the save button says "Create prompt"

### S-PC-002: Create success

**Given** the user fills slug, description, and body  
**When** the user clicks "Create prompt"  
**And** `POST /prompts` returns HTTP 201  
**Then** the new prompt appears in the sidebar  
**And** the editor switches to edit mode for the new prompt  
**And** the activity feed shows a "created" event

### S-PC-003: Slug validation (regex)

**Given** the user enters a slug  
**When** the slug does not match the backend regex (2-100 chars, lowercase letters, numbers, hyphens)  
**Then** a validation error is shown: "Slug must be 2-100 characters, lowercase letters, numbers, and hyphens only"  
**And** the save is not submitted

---

## 5. Version History (Collapsed by Default)

### S-PH-001: History button is visible

**Given** a prompt is selected in edit mode  
**When** the editor renders  
**Then** a "History" button is visible below the editor  
**And** it shows the number of revisions (e.g. "History (3)")

### S-PH-002: History expands on click

**Given** the user clicks "History"  
**When** the history panel expands  
**Then** it shows a list of all revisions, newest first  
**And** each revision shows: version number, `created_at`, first 50 chars of body  
**And** a "Compare" or expand toggle is shown

### S-PH-003: Diff renders on expand

**Given** the user expands a revision in the history panel  
**When** the diff block renders  
**Then** it shows a side-by-side diff between this revision and the previous one  
**And** removed lines are shown with red background  
**And** added lines are shown with green background

### S-PH-004: History collapses on click

**Given** the history panel is expanded  
**When** the user clicks "Collapse"  
**Then** the history panel hides  
**And** only the "History (N)" button remains visible

---

## 6. Restore Revision

### S-PR-001: Restore button on historical revision

**Given** a historical revision is visible in the history panel  
**When** the revision is not the current one  
**Then** a "Restore" button is shown on that revision

### S-PR-002: Restore confirmation dialog

**Given** the user clicks "Restore" on a historical revision  
**When** the click fires  
**Then** a confirmation dialog appears  
**And** it says: "Restore to v{N}? This will create a new revision with the v{N} content."  
**And** the current content will be preserved in history

### S-PR-003: Restore confirmed

**Given** the user confirms the restore dialog  
**When** `POST /prompts/:slug/revisions/:n/restore` returns HTTP 200  
**Then** the editor's body is updated to the restored content  
**And** the revision number increments  
**And** the activity feed shows a "restored" event  
**And** the history panel refreshes

### S-PR-004: Restore on current revision — no-op

**Given** the user clicks "Restore" on the current revision  
**When** the click fires  
**Then** the button is disabled or shows "Already current"  
**And** no API call is made

---

## 7. Delete Prompt

### S-PD-001: Delete button visible

**Given** a prompt is selected in edit mode  
**When** the editor renders  
**Then** a "Delete prompt" button is visible (destructive styling)

### S-PD-002: Delete confirmation dialog

**Given** the user clicks "Delete prompt"  
**When** the click fires  
**Then** a confirmation dialog appears  
**And** it says: "Delete 'slug'? This cannot be undone."  
**And** the prompt name is shown

### S-PD-003: Delete confirmed

**Given** the user confirms the delete dialog  
**When** `DELETE /prompts/:slug` returns HTTP 200  
**Then** the prompt is removed from the sidebar  
**And** the editor shows the empty state  
**And** no API calls for this prompt succeed anymore (410 Gone)

### S-PD-004: Delete cancelled

**Given** the user cancels the delete dialog  
**When** the cancel button is clicked  
**Then** no API call is made  
**And** the dialog closes

---

## 8. Activity Feed

### S-PA-001: Activity feed shows events

**Given** a prompt is selected in edit mode  
**When** the activity feed renders  
**Then** it shows events for the selected prompt: created, edited, restored, deleted  
**And** events are sorted newest first  
**And** each event shows: icon, description, timestamp

### S-PA-002: Event icons

**Given** an activity event  
**When** the event type is "created"  
**Then** a "+" icon is shown  
**And** the text says: "v{N} created"

**When** the event type is "edited"  
**Then** a pencil icon is shown  
**And** the text says: "v{N} saved"

**When** the event type is "restored"  
**Then** a restore icon is shown  
**And** the text says: "v{N} restored to current"

**When** the event type is "deleted"  
**Then** a trash icon is shown  
**And** the text says: "Prompt deleted"

### S-PA-003: Empty activity feed

**Given** the prompt has no events beyond creation  
**When** the activity feed renders  
**Then** it shows at least the creation event

---

## 9. Error States

### S-PE-008: Offline error on save

**Given** the network is offline  
**When** the user clicks "Save"  
**Then** an error alert is shown: "Couldn't reach the backend. Check your connection."  
**And** the user's changes are preserved in the form

### S-PE-009: Server error on save

**Given** the backend returns HTTP 500  
**When** the user clicks "Save"  
**Then** an error alert is shown with the backend message  
**And** the user's changes are preserved in the form

---

## 10. Navigation

### S-PN-001: Back to settings

**Given** the user is on `/settings/prompts`  
**When** the user clicks "Back to settings"  
**Then** the user navigates to `/settings`

### S-PN-002: Unsaved changes — navigation guard

**Given** the user has unsaved changes in the editor  
**When** the user navigates away (via sidebar, back link, or URL change)  
**Then** the user is NOT prompted to confirm (Qwik SPA navigation — no page reload risk)  
**And** changes are lost (acceptable for v1 — no guard needed for SPA navigation)

---

## 11. Accessibility

### S-PA-011: Skip link

**Given** the user focuses the skip link  
**When** the user activates it  
**Then** focus moves to `<main>`

### S-PA-012: ARIA labels

**Given** interactive elements are rendered  
**When** the page renders  
**Then** all buttons have accessible labels  
**And** the history panel has `role="region"` with `aria-label="Version history"`  
**And** the activity feed has `role="feed"` or `role="list"`

### S-PA-013: Keyboard navigation

**Given** the user is on the page  
**When** the user tabs through the interface  
**Then** focus order is logical: sidebar items → editor fields → buttons  
**And** the History button is keyboard accessible
