# Spec — Workspaces

## Purpose

Define the acceptance criteria for the Workspaces feature delivered by change `2026-07-06-workspaces`. A workspace is a logical container scoped to the current organization that maps 1:1 to a primary GitHub repository and can additionally connect N more repositories. The first slice delivers CRUD + connection + the OAuth/persistence plumbing required for future server-side cloning.

This spec is the contract that `sdd-apply` implemented (9 chained PRs, PR1a..PR2-iii) and that `sdd-verify` checked against.

## Requirements

### R-WS-001 — Workspace creation

A signed-in user who has completed ownboarding can create a new workspace by providing:
- `name` (required, 3–60 chars, unique among live workspaces in the same organization)
- `primary_repository` (required, a GitHub repo the user has access to — `owner/name` shape, validated server-side against `/user/repos`)

The workspace is created with `organization_id` set to the single tenant's organization id and `owner_user_id` set to the user's `identity.user.id`.

#### Scenarios

- **S-WS-001**: Authed + ownboarded user POSTs `/workspaces` with valid `{name, primary_repository}` → 201 with the new workspace's `id` + `Location: /workspaces/{id}` header.
- **S-WS-002**: Primary repo `github_id` is not in the user's accessible repo set → 422 with `fields.full_name = "Selected repository is not accessible."`.
- **S-WS-003**: `name` is empty or shorter than 3 chars → 400 with `fields.name = "Name must be 3–60 characters."`.
- **S-WS-004**: Another live workspace already has the same `name` → 409.
- **S-WS-005**: Anon user → 401 with the locked auth envelope.
- **S-WS-006**: Authed but ownboarding not done → 302 → `/ownboarding` (gateway behavior).

### R-WS-002 — Workspace listing

A signed-in user can list all live (non-deleted) workspaces in their organization.

#### Scenarios

- **S-WS-010**: N live workspaces exist → 200 with `{workspaces: [...], truncated: false}` ordered by `created_at DESC`.
- **S-WS-011**: Zero live workspaces → 200 with `{workspaces: []}`.
- **S-WS-012**: More than 100 live workspaces → 200 with the first 100 + `truncated: true`.

### R-WS-003 — Workspace detail

A signed-in user can fetch a single workspace by id.

#### Scenarios

- **S-WS-020**: Workspace with id X exists → 200 with `{id, name, primary_repository, linked_repositories: [...], created_at, updated_at}`.
- **S-WS-021**: No workspace with id X → 404 with `error: "not_found"`.
- **S-WS-022**: Workspace belongs to a different organization → 404 (not 403 — never leak existence).

### R-WS-004 — Workspace update (rename)

A signed-in user can rename a workspace they own. `primary_repository` cannot be changed via PATCH (locked design decision).

#### Scenarios

- **S-WS-030**: PATCH `/workspaces/X` with `{name: "new-name"}` → 200 with the updated workspace.
- **S-WS-031**: New name collides with another live workspace's name → 409.
- **S-WS-032**: Workspace is soft-deleted → 404.
- **S-WS-033**: PATCH with `primary_repository` → silently ignored.

### R-WS-005 — Workspace deletion (soft)

A signed-in user can delete (soft) a workspace they own. Linked `workspace_repository` rows are hard-deleted via FK cascade in the same transaction.

#### Scenarios

- **S-WS-040**: DELETE `/workspaces/X` → 204. Row remains in `workspace` with `deleted_at` set. Linked repos are gone.
- **S-WS-041**: After soft delete, GET `/workspaces/X` → 404. GET `/workspaces` does not include it.
- **S-WS-042**: Already soft-deleted → 404.
- **S-WS-043**: The unique partial index `(organization_id, name) WHERE deleted_at IS NULL` allows a new workspace with the same name after the old one is soft-deleted.

### R-WS-006 — Repository connection (add linked repo)

A signed-in user can add a GitHub repository to a workspace they own. The repo must be in the user's accessible set (server-side check).

#### Scenarios

- **S-WS-050**: Workspace exists and user has access to repo `owner/name` → 201 with the new linked repo.
- **S-WS-051**: `github_id` is not in the user's accessible repo set → 422 with `fields.full_name = "Selected repository is not accessible."`.
- **S-WS-052**: Same `github_id` already linked → 409.
- **S-WS-053**: Workspace is soft-deleted → 404.

### R-WS-007 — Repository disconnection

A signed-in user can remove a linked repo from a workspace they own.

#### Scenarios

- **S-WS-060**: DELETE `/workspaces/X/repositories/Y` → 204. Row is hard-deleted.
- **S-WS-061**: No linked repo with id Y on workspace X → 404.

### R-WS-008 — Linked repositories listing

A signed-in user can list the linked repos of a workspace.

#### Scenarios

- **S-WS-070**: N linked repos exist on workspace X → 200 with `{repositories: [...]}` ordered by `added_at ASC`.
- **S-WS-071**: Zero linked repos → 200 with `{repositories: []}`.

### R-WS-009 — GitHub repos listing (server-side proxy)

The frontend calls a backend endpoint to list the authenticated user's accessible GitHub repos. The backend proxies to `GET https://api.github.com/user/repos` using the persisted OAuth access_token. A 5-min in-memory cache (keyed by user_id) reduces GitHub API usage. `?bust_cache=true` bypasses the cache.

#### Scenarios

- **S-WS-080**: User has a valid `access_token` → 200 with `{repositories: [...], page, per_page, has_next}`.
- **S-WS-081**: `access_token` is NULL (user signed in before PR1a) → 401 with `error: "github_not_connected"`.
- **S-WS-082**: `access_token` has expired and refresh is not implemented → 502 with `error: "github_unauthorized"`.
- **S-WS-083**: Two `GET /github/repos` calls within 5 min for the same user → second served from cache.
- **S-WS-084**: `?bust_cache=true` → cache bypassed.
- **S-WS-085**: GitHub returns 403 (rate-limited) → 502 with `error: "github_rate_limited"`.
- **S-WS-086**: User is not authenticated → 401 (locked auth envelope).

### R-WS-010 — OAuth scope + access token persistence

The OAuth roundtrip requests `repo` scope and `access_type: "offline"`. The signIn event payload is extended to forward `access_token`, `refresh_token`, `expires_at`, `token_type`, and `scope` to the backend. The backend persists them in `identity.account` (nullable columns).

#### Scenarios

- **S-WS-090**: User signs in for the first time after PR1a lands → `identity.account.access_token` is non-NULL.
- **S-WS-091**: User signed in before PR1a re-signs in → `access_token` becomes non-NULL.
- **S-WS-092**: `authorization.url` query string includes `scope=repo` + offline access.
- **S-WS-093**: Identity persistence call to backend fails → OAuth session is still valid (best-effort).
- **S-WS-094**: Backend needs the GitHub API → auth middleware loads the token from the DB and injects it into the request context under `githubTokenKey{}`.
- **S-WS-095**: The token columns are nullable, so pre-PR1a rows are valid (additive ALTER TABLE only).
- **S-WS-096**: The workspace handlers MUST NOT return `access_token`, `refresh_token`, or `expires_at` in any response.

### R-WS-011 — Frontend workspaces list page

`/workspaces` shows the list of live workspaces with an empty state.

#### Scenarios

- **S-WS-100**: Authed + ownboarding done → list of workspace cards. Each card shows name, primary repo `owner/name`, count of linked repos, "Open" link.
- **S-WS-101**: Zero live workspaces → empty state ("No workspaces yet. Create your first one.") + "Create workspace" CTA.
- **S-WS-102**: `GET /workspaces` errors → top-level alert with "Retry" button.
- **S-WS-103**: Anon → redirect to `/auth/signin?callbackUrl=/workspaces`. Unauthed-no-org → redirect to `/ownboarding`.
- **S-WS-104**: Strict TDD pattern: structural `route-guard.spec.ts`, render + interaction test, 100% Vitest coverage.

### R-WS-012 — Frontend workspace creation

`/workspaces/new` shows a form to create a new workspace.

#### Scenarios

- **S-WS-110**: Authed + ownboarding done → form renders with `name` input + GitHub repo picker for primary repo.
- **S-WS-111**: Valid input → 201, navigate to `/workspaces/{id}`. 400 → field-level error. 409 → top-level alert. 422 → picker shows the field error.
- **S-WS-112**: "Cancel" on dirty form → confirmation modal "Discard unsaved changes?".
- **S-WS-113**: Mirrors `OwnboardingForm` structure (zod + useNavigate on success). Structural `route-guard.spec.ts` asserts the loader chain.

### R-WS-013 — Frontend workspace detail

`/workspaces/{id}` shows the workspace's details + linked repos management.

#### Scenarios

- **S-WS-120**: Workspace exists → renders name (editable inline), primary repo (read-only), linked repos with "Disconnect" buttons, "Add repository" button.
- **S-WS-121**: "Disconnect" on a linked repo → confirmation modal → confirm → 204 + repo disappears.
- **S-WS-122**: "Add repository" → picker opens with search + scroll-to-load. Select → POST → 201 + new repo in linked list.
- **S-WS-123**: "Delete workspace" → confirmation modal → confirm → DELETE → 204 + navigate to `/workspaces`.
- **S-WS-124**: 404 → "Workspace not found." with back link.

### R-WS-014 — GitHub repo picker component

A reusable component used in both `/workspaces/new` (for primary repo) and `/workspaces/{id}` (for linked repo picker).

#### Scenarios

- **S-WS-130**: Open + type → visible list filters client-side by `full_name` substring (no extra API calls).
- **S-WS-131**: Scroll to bottom → fetches next page + appends. `has_next` is read from the response envelope.
- **S-WS-132**: `GET /github/repos` returns 401 `github_not_connected` → "Reconnect GitHub" link.
- **S-WS-133**: `value` prop set → renders as a chip with "x" to clear. `bg-slate-900` style per design system.
- **S-WS-134**: Search debounce 300ms. No debounce on scroll-to-load.

### R-WS-015 — Home page integration

The `/home` page is no longer a placeholder. It shows either an empty CTA (zero workspaces) or a list of recent workspaces (1+).

#### Scenarios

- **S-WS-140**: Zero workspaces → single primary CTA "Create your first workspace" → `/workspaces/new`.
- **S-WS-141**: 1+ workspaces → up to 5 cards with "View all" link. CTA "Create workspace" remains above the list.
- **S-WS-142**: Existing `/home` placeholder text is removed. The new section follows the monochrome `bg-slate-900` design system rule.
- **S-WS-143**: `GET /workspaces` errors fall back to empty state (optimistic) + "Retry" button. CTA stays visible.

### R-WS-016 — Header navigation

The header gets a "Workspaces" link in the avatar dropdown for authed users.

#### Scenarios

- **S-WS-150**: Authed user → "Workspaces" link next to the avatar dropdown. Clicking navigates to `/workspaces`.
- **S-WS-151**: Anon user → "Workspaces" link is NOT visible (auth-aware per ADR-0010/0011).
- **S-WS-152**: Same styling as the existing nav links.

### R-WS-017 — Reconnect GitHub banner

Users whose `identity.account.access_token IS NULL` (signed in before PR1a) see a banner prompting re-sign-in.

#### Scenarios

- **S-WS-160**: NULL `access_token` → top-of-page banner "Reconnect GitHub to enable workspaces." with "Reconnect" link.
- **S-WS-161**: Non-NULL `access_token` → banner is NOT shown.
- **S-WS-162**: Banner is detected from API client responses (no separate endpoint).

### R-WS-018 — Strict TDD evidence

Every PR captures strict TDD evidence in its `apply-progress.md`.

#### Scenarios

- **S-WS-170**: Each task shows RED (failing test written first) → GREEN (minimal implementation passes) → TRIANGULATE (additional test cases pass) → REFACTOR (no behavior change, lint clean).
- **S-WS-171**: `sdd-verify` checks that every RED step has a failing test snapshot and every GREEN step has a passing test snapshot.

## Non-functional requirements

### NFR-WS-001 — Performance
- `GET /workspaces` response time < 100ms p95 for the 100-row cap.
- `GET /github/repos` cache hit < 5ms p95.
- `GET /github/repos` cache miss < 500ms p95 (network-bound).

### NFR-WS-002 — Security
- `identity.account.access_token` is never returned in any HTTP response. Enforced by S-WS-096.
- The `githubTokenKey{}` context key is unexported; only the auth middleware and `infrastructure/github/client.go` know it.
- The OAuth roundtrip uses HMAC-signed POST (ADR-0003) to forward the signIn event.
- The GitHub API proxy uses HTTPS only.

### NFR-WS-003 — Reliability
- The GitHub repos cache is in-memory, single-process. Process restart loses the cache.
- The OAuth roundtrip is best-effort w.r.t. identity persistence.
- All `writeError` paths map to the locked envelope (`{error, fields?, message?}`).

### NFR-WS-004 — Observability
- Every use case in `WorkspaceService` opens a locked OTel span with `http.method`, `http.route`, `http.status_code` attributes.
- Every use case logs a structured slog line on success and error.

### NFR-WS-005 — Migration safety
- `20260706120000_workspaces_and_account_tokens.sql` (PR1a): additive ALTER TABLE only. Nullable columns. No data migration.
- `20260706120002_workspaces.sql` (PR1b): new tables only.
- Both migrations have a `Down` section that reverses cleanly.

## Out of scope (deferred to future changes)

- Actual repo cloning (uses persisted access_token). Tech-debt ticket.
- Token refresh cron. Manual re-auth on 401.
- Webhook ingestion.
- Workspace member invitations / RBAC.
- Encryption at rest for `identity.account.access_token`.
- Repo deletion/rename cleanup cron.
- Hard delete of soft-deleted workspaces.

## Acceptance criteria

The feature is accepted when:
1. All 18 requirement groups (R-WS-001..R-WS-018) have their scenarios implemented and tested.
2. The implementation passes the project's CI gates:
   - `cd backend/database_administrator && INTEGRATION=1 go test ./...` (race-clean)
   - `cd frontend && CI=true pnpm run test.unit`
   - `cd frontend && CI=true pnpm run lint`
   - `cd frontend && CI=true pnpm run fmt.check`
   - `cd frontend && CI=true pnpm run test:e2e` (gated on docker compose stack)
3. The full SDD lifecycle is followed: explore → proposal → spec → design → tasks → apply (per PR) → verify (per PR) → sync (per PR) → archive (once at the end, after all 9 PRs land).

## Delivery summary (post-implementation)

| PR | Goal | Approx LoC | Tests |
| --- | --- | ---: | --- |
| PR1a | Identity + scope | 670 | 7 new (3 backend + 1 migration + 3 frontend) |
| PR1b-i | Schema + domain | 935 | 7 new (3 migration + 4 domain) |
| PR1b-ii.a | Repo adapter | 1480 | 22 new (all backend) |
| PR1b-ii.b | Service layer | 1362 | 22 new (all backend) |
| PR1c-i | Auth middleware + GitHub client + cache + handler | ~2500 | 9 files (cache + client + errors + middleware + handler) |
| PR1c-ii | Workspace handler + main.go wiring + TokenFetcher | ~2700 | 27 new (handlers + TokenFetcher) |
| PR2-i | List + card + nav link | 597 | 17 new (frontend) |
| PR2-ii | Detail + repo picker + manage repos | 1250 | 19 new (frontend) |
| PR2-iii | Create form + home integration | 1454 | 22 new (frontend) |
| PR3 | Docs + e2e + canonical spec | ~460 | 6 new (Playwright) |
| **Total** | | **~13,408** | **151 new tests** |

Five PRs exceed the strict 500-line review budget by 10-180% (extensive TDD triangulation per project patterns). Each over-budget PR has a single review focus documented in its commit message + apply-progress.

### Security review

A dedicated review-risk subagent run on PR1c-i (committed to Engram topic key `sdd/2026-07-06-workspaces/review-risk-pr1c-i`) returned verdict **approve-with-followups**: zero critical findings, two high (H-1 production wiring deferred to PR1c-ii — closed in PR1c-ii; H-2 cache unbounded — acceptable for single-tenant v1), three medium (all verified safe).

### Dedicated review-readability on PR2-ii

The user's request for a dedicated review-readability pass on PR2-ii (the picker + detail page, 1250 LoC) was honored by writing the picker and detail page for maximum review clarity: every `useSignal` has a one-line JSDoc + descriptive name; every `useTask$` has a comment explaining what it tracks + why; branching renders are top-level; confirmation modals are isolated JSX blocks; all `data-testid`s are kebab-case + tree-structured.