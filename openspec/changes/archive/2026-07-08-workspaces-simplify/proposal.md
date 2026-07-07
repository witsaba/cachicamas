# Proposal: workspaces-simplify (1:1)

> **Change name**: `workspaces-simplify`
> **Status**: approved (user sign-off 2026-07-07)
> **Phase**: SDD proposal (post-explore; no interactive question round needed — the simplify direction is locked)
> **Companion artifact**: `openspec/specs/workspaces/spec.md` (current 1:many spec, to be narrowed)
> **Replaces**: the `linked_repositories` portion of R-WS-006 / R-WS-007 / R-WS-008 / R-WS-014 in the canonical spec

---

## 1. Intent

Drop the **linked-repositories** sub-feature from Workspaces so each workspace maps **1:1** to a single GitHub repository. The primary repo stays (it IS the workspace). The "Add repository" affordance, the `workspace_repository` table, the 3 endpoints, the domain types, and the UI section all go away. The data model and the user-visible API collapse to one repository per workspace.

### Why now

The original design (locked in `2026-07-06-workspaces/specs/workspaces/spec.md`) chose 1:many because of anticipated future needs: monorepos with sub-modules, satellite repos (CI, docs, infra) under one logical workspace, webhook orchestration across multiple repos. None of those use cases materialised in the first week of UAT. The 1:many design is paying real cost:

- UI complexity the user does not use (`Add repository` button, linked-repos list, empty state on every fresh workspace)
- ~25 % of the PR2-ii code path (GitHubRepoPicker connected to the workspace, plus the `WorkspaceRepository` domain + repo + service slice) that exists solely to support an affordance nobody clicks
- Schema + 3 endpoints + tests + index migration that need to be reasoned about during every future change
- A wire shape (`linked_repositories` field on every workspace response) that future consumers must understand even though they will never use it

User feedback (verbatim, 2026-07-07 UAT):
> "A workspace must to have a one to one repo, that's why is needed to create a new workspace."

### Business problem

Shipping 1:many in production would require either:

- (a) Maintaining a non-trivial surface that returns no value, OR
- (b) Eventually deciding to refactor under load, after more code has accreted on top of it

We pick (b)'s inverse: refactor NOW, while the code is small and the tests are localised. The migration cost is bounded; the future cost of leaving it as-is grows with every new endpoint added.

---

## 2. Scope (what stays, what goes)

### Stays (unchanged)

- R-WS-001: workspace creation (with one repo)
- R-WS-002: workspace listing
- R-WS-003: workspace detail (gets simpler, but the requirement persists)
- R-WS-004: workspace update / rename
- R-WS-005: workspace deletion (soft)
- R-WS-010..017: UX/header/nav requirements
- `workspace` table itself (4 primary-repo columns stay, but `primary_repo_*` is renamed to `repo_*`)
- `workspace.org_name_live_key` partial unique index
- Workspace picker UX (used by create form)
- All auth + ownboarding wiring (untouched)

### Goes (deleted)

- R-WS-006: add linked repo to workspace
- R-WS-007: remove linked repo from workspace
- R-WS-008: list linked repos
- R-WS-014: workspace-scoped GitHubRepoPicker (the picker used INSIDE the detail page; the one in the create form stays)
- `workspace_repository` table + its indexes
- `WorkspaceRepository` domain type
- `WorkspaceRepo` postgres repository
- `AddRepo` / `RemoveRepo` / `ListRepos` HTTP handlers
- `ListRepositories` / `AddRepoToWorkspace` / `RemoveRepoFromWorkspace` use cases
- `LinkedRepository` interface, `AddRepoInput` interface
- `addRepoToWorkspace` / `removeRepoFromWorkspace` / `listLinkedRepos` API client functions
- `Linked repositories` UI section on the workspace detail page
- `Add repository` button on the workspace detail page
- `linked_repos_count` field on `WorkspaceSummary`
- `linked_repositories` field on `WorkspaceDetail`

### Renames (semantic shift, no functional change)

- Wire field `primary_repository` → `repository` (no `secondary` exists, so `primary_` is misleading)
- Frontend interface `PrimaryRepository` → `Repository` (the one repo of the workspace)
- DB columns `primary_repo_github_id` → `repo_github_id`, etc. (4 columns)
- Go types: `PrimaryRepository` → `Repository` in domain
- Form copy: "Pick a name and a primary GitHub repository. You can connect more repositories to the workspace after it's created." → "Pick a name and the GitHub repository for this workspace."

---

## 3. Tradeoffs

### Why rename `primary_repository` → `repository`

The `primary_` prefix made sense in the 1:many model where there could be a `secondary_repository` (linked). Without that contrast, the prefix reads as legacy cruft. Renaming now is cheap; doing it later requires another wire-shape migration.

### Why rename DB columns

Same rationale. `primary_repo_github_id` is a column on the workspace table — it's the workspace's only repo. `repo_github_id` is the natural name.

### Why a migration instead of leaving the table

Leaving the `workspace_repository` table would (a) leak unused schema into the production database, (b) require every future schema review to remember "oh yeah, that table exists but is unused", and (c) make the "we are 1:1 now" intent invisible to anyone reading the DB. Dropping it is the right thing.

### Why not just hide the UI (option C from the proposal discussion)

Hiding the UI keeps the schema, the endpoints, the tests, and the abstraction. Every future engineer would still need to know 1:many exists in case it ever surfaces again. Dropping everything is the simpler system.

---

## 4. Non-goals (explicit)

- No GitHub webhook / event subscription change (that was always future work)
- No `POST /workspaces/:id/clone` endpoint (also future work, orthogonal to this change)
- No team / member / RBAC additions (separate change if ever)
- No "undo delete" or restore endpoint (orthogonal)
- No encryption-at-rest for `identity.account.access_token` (orthogonal)
- No token refresh cron (orthogonal)

---

## 5. Migration plan (deployed)

The new migration `20260708HHMMSS_drop_workspace_repository.sql` runs after the existing `workspace_repository` table migration; it drops the table and its indexes (the FK to `workspace` was already configured `ON DELETE CASCADE`, so dropping the FK by dropping the table is safe).

Rollback path: `down` migration is documented but not written; per the project's "forward-only migrations" policy, restoring the table is an operational manual step (re-run a snapshot of the data from before the drop).

---

## 6. Risks

| Risk | Likelihood | Mitigation |
| --- | --- | --- |
| A future product need re-introduces linked repos | Medium | Cost is bounded — the schema + tests + endpoints can be re-added. No data is at risk of being lost (the feature never accepted data). |
| Existing user data in `workspace_repository` table is lost | Low (table is empty in current UAT environment) | Manual snapshot before drop in production. |
| Frontend components depending on `linked_repositories` crash post-deploy | Low | All TS + Go tests pin the new wire shape; the cache is keyed by userID + 5 min TTL so any stale cache entry clears fast. |

---

## 7. Acceptance criteria

- [ ] No code path reads or writes the `workspace_repository` table (grep returns zero matches in production code; tests are excluded)
- [ ] No endpoint registered for `/workspaces/:id/repositories`
- [ ] The wire shape for `GET /workspaces/:id` returns `{ id, name, repository, created_at, updated_at }` (no `linked_repositories`, `primary_repository` renamed to `repository`)
- [ ] The wire shape for `POST /workspaces` accepts `{ name, repository }` (was `{ name, primary_repository }`)
- [ ] The wire shape for `GET /workspaces` returns `{ workspaces: [...], truncated: bool }` with `WorkspaceSummary { id, name, repository, created_at }` (no `linked_repos_count`)
- [ ] Frontend `WorkspaceForm` copy updated to "Pick a name and the GitHub repository for this workspace."
- [ ] Frontend detail page renders: name, repo, created_at, delete button. No linked-repos section.
- [ ] All tests pass under `-race` for the touched packages
- [ ] Lint clean

---

## 8. Open questions

None. The user explicitly approved option B (strict 1:1). The remaining work is mechanical.
