# Tasks: workspaces-simplify (1:1)

> Mechanical breakdown of the proposal. Each task is sized as a single commit (≤ 500 lines per the project's review budget).

---

## 1. Backend — drop schema + service for linked repos

### T-SIMP-001 [migration] Add `20260708HHMMSS_drop_workspace_repository.sql`

Drops the `workspace_repository` table and its indexes. Forward-only per the project's migration policy. Idempotent: re-running is a no-op (DROP TABLE IF EXISTS).

Affected files:

- `backend/database_administrator/src/migration/sql/20260708HHMMSS_drop_workspace_repository.sql`

### T-SIMP-002 [domain] Strip `WorkspaceRepository` from `domain/workspace.go`

Remove the `WorkspaceRepository` struct and the related fields on the `Workspace` repo interface. `Workspace` domain itself stays; the 4 `primary_repo_*` columns on the `workspace` table get renamed to `repo_*`.

Affected files:

- `backend/database_administrator/src/domain/workspace.go`

### T-SIMP-003 [service] Strip `ListRepositories`, `AddRepoToWorkspace`, `RemoveRepoFromWorkspace` use cases

The `WorkspaceService` loses the 3 use cases that touch linked repos. Existing use cases stay.

Affected files:

- `backend/database_administrator/src/application/workspace_service.go`
- `backend/database_administrator/src/application/workspace_service_test.go`

### T-SIMP-004 [repo] Strip linked repo queries from `workspace_repo.go` + drop the dedicated `workspace_repository_repository.go`

`workspace_repo.go` keeps the workspace-row CRUD. The separate `workspace_repository_repository.go` goes away (no other callsite reads it).

Affected files:

- `backend/database_administrator/src/infrastructure/postgres/workspace_repo.go`
- `backend/database_administrator/src/infrastructure/postgres/workspace_repository_repository.go` (DELETED)
- `backend/database_administrator/src/infrastructure/postgres/workspace_repo_test.go`

### T-SIMP-005 [http] Strip 3 handlers + rename wire field

`workspace_handler.go`:

- Drop `AddRepo`, `RemoveRepo`, `ListRepos` handlers
- Drop the `/workspaces/:id/repositories` route group (only the `POST /workspaces` GET/PATCH/DELETE remain)
- Rename `primary_repository` → `repository` in `createWorkspaceRequest` + `workspaceResponse` + `workspaceSummaryResponse` + `workspaceDetailResponse`
- Drop the `linked_repositories` field from `workspaceDetailResponse`
- Drop the `linked_repos_count` from `workspaceSummaryResponse`
- Rename DB columns in `toWorkspaceResponse` etc.

Affected files:

- `backend/database_administrator/src/interfaces/http/workspace_handler.go`
- `backend/database_administrator/src/interfaces/http/workspace_handler_test.go`
- `backend/database_administrator/src/interfaces/http/workspaces_auth_chain_test.go`

---

## 2. Frontend — drop API client + UI slice

### T-SIMP-101 [api] Strip `addRepoToWorkspace` / `removeRepoFromWorkspace` / `listLinkedRepos`

`lib/api.ts`:

- Drop the 3 functions
- Drop the `LinkedRepository` and `AddRepoInput` interfaces
- Rename `PrimaryRepository` → `Repository`
- Drop `linked_repositories` from `WorkspaceDetail`
- Drop `linked_repos_count` from `WorkspaceSummary`
- Update `createWorkspace`'s input + wire body to use `repository` instead of `primary_repository`
- Update related tests in `api.spec.ts`

Affected files:

- `frontend/src/lib/api.ts`
- `frontend/src/lib/api.spec.ts`

### T-SIMP-102 [detail page] Drop Linked repositories + Add repository sections

`routes/workspaces/[id]/index.tsx`:

- Drop the entire "Linked repositories" section + the empty-state message
- Drop the "Add repository" toggle button
- Drop the picker QRL + the `onSelectRepoForAdd` handler
- Drop `addRepoToWorkspace` / `removeRepoFromWorkspace` imports + state
- Rename `primaryRepo` references to `repo`
- Update the page header to render `ws.repository` instead of `ws.primary_repository`

Affected files:

- `frontend/src/routes/workspaces/[id]/index.tsx`
- `frontend/src/routes/workspaces/[id]/index.spec.tsx`

### T-SIMP-103 [workspace card] Drop linked-repos badge

`components/workspace-card/workspace-card.tsx`:

- Drop the `linked_repos_count` badge

Affected files:

- `frontend/src/components/workspace-card/workspace-card.tsx`
- (No spec changes — the card currently renders `linked_repos_count` only; removing the field from the workspace API will force a TS error here, which we fix by removing the badge.)

### T-SIMP-104 [create form] Copy update

`components/workspace-form/workspace-form.tsx` + `routes/workspaces/new/index.tsx`:

- Change the form intro copy from "Pick a name and a primary GitHub repository. You can connect more repositories to the workspace after it's created." to "Pick a name and the GitHub repository for this workspace."
- `WorkspaceForm`'s action type stays the same (discriminated union); the inline `primary_repository` field names in the FormData are renamed to `repository`

Affected files:

- `frontend/src/components/workspace-form/workspace-form.tsx`
- `frontend/src/routes/workspaces/new/index.tsx`

### T-SIMP-105 [picker] Stays scoped to create form

`components/github-repo-picker/` itself is unchanged. It is now used only by the create form (`routes/workspaces/new/index.tsx`), not by the workspace detail page.

Affected files: none (no change).

---

## 3. Verify

### T-SIMP-201 [tests] Backend + frontend suites green

- `cd backend/database_administrator && go test -race -count=1 ./src/interfaces/http/ ./src/domain/ ./src/application/ ./src/infrastructure/github/` → all PASS
- `cd frontend && CI=true pnpm test.unit` → 339+ / 339+ PASS
- `cd frontend && CI=true pnpm lint` → 0 errors

### T-SIMP-202 [smoke] End-to-end through Docker

Manual via `docker compose down -v && docker compose up -d --build`:

- Sign in (already covered by previous fixes)
- Create a workspace → POST returns flat shape, UI navigates to /workspaces/:id
- /workspaces/:id renders name + repository (no linked-repos section visible)
- Delete workspace → /workspaces list updates

---

## 4. Toolchain

No toolchain changes. Both backend and frontend keep their existing build/test/lint pipelines. The spec archive document is updated to reflect the 1:1 contract; the canonical `openspec/specs/workspaces/spec.md` will be replaced with the 1:1 version in a follow-up that runs `sdd-verify` against the new shape.
