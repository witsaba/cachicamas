# Proposal: prd-orchestrator

> **Change**: `prd-orchestrator`
> **Status**: proposed
> **Created**: 2026-06-22
> **Driver**: braejan
> **Project**: cachicamas (witsaba)
> **Persistence**: hybrid (this file + Engram `sdd/prd-orchestrator/proposal`)
> **Spec domain**: `prd-orchestration` (will live at `openspec/changes/prd-orchestrator/specs/prd-orchestration/spec.md`)

---

## Intent

Cachicamas is the Witsaba Software Development Framework, and today its SDD pipeline (`sdd-explore → sdd-propose → sdd-spec → sdd-design → sdd-tasks → sdd-apply → sdd-verify → sdd-archive`) is invoked manually per change. We need a PRD-driven orchestrator on top of that pipeline: a service that ingests a PRD, scrapes the target repo for metadata, routes the PRD through a human review gate, decomposes it into milestones and tasks, and then drives each task through the existing SDD pipeline with worktrees, DAG execution, multi-reviewer PRs, and escalation history. Without this, every Witsaba project repeats the same planning boilerplate by hand.

This change delivers **v0.0.1**: the framework's foundation — schema, PRD intake, metadata analysis, and a scaffold of the milestone + task decomposition — so subsequent v0.0.x iterations can layer on DAG execution, the per-task SDD wrapper, PR creation, and the PRD-as-spec-comparison harness without re-deciding the foundations.

## Scope

### In Scope (v0.0.1 thin slice — single PR, ~500–600 LoC)

- New hexagonal service `backend/prd_orchestrator/` (sibling to `backend/database_administrator/`) with `cmd/`, `application/`, `domain/`, `interfaces/`.
- DB schema and migrations under `database_administrator` for the tables: `organization`, `project`, `project_reviewer`, `prd`, `milestone`, `task`, `spec`, `phase`.
- HTTP endpoints: `POST /prds` (intake), `GET /prds/{id}` (status), `GET /projects/{id}` (project view).
- Use cases: PRD intake, metadata scrape (creator, contributors, latest commits + authors, recent PRs), PRD alignment doc (NEW vs REFACTOR vs ADDITION), milestone decomposition scaffold (Strategy D: Journey Milestones × Vertical Slice Tasks), task decomposition scaffold (1 task per milestone for v0.0.1).
- Organization + project + reviewer domain entities; multi-reviewer project assignment via `project_reviewer` table.
- State persistence via the existing `database_administrator` service (no new DB driver).
- OTel observability wired through the same env-var pattern used in `database_administrator`.
- New capability `prd-orchestration` (the `sdd-spec` agent will write its full spec).

### Out of Scope (deferred to later v0.0.x or v0.1.x)

- OOS-1: Frozen interface spec before subtask fan-out (R1 — "next to resolve").
- OOS-2: Skip Block 1 alignment when target repo == framework (R2 — "next to resolve").
- OOS-3: Golden fixtures for PRD-comparison mode (R3 — "next to resolve").
- OOS-4: `project.setting JSONB` storage (R4 — "next to resolve").
- OOS-5 / OOS-6: Specific TDD retry defaults (5/10) and concurrency defaults (4/2) — limits driven by worktree capacity, decided empirically.
- OOS-7: MCP-based Agent-as-a-Service packaging.
- OOS-8: Multi-company competitive organization.
- Full DAG execution engine (cycle detection, parallel workers, worktree manager) — v0.0.2.
- Per-task custom SDD wrapper invocation (capability 6) — v0.0.3.
- PR creation, worktree manager, multi-reviewer assignment UI — v0.0.4.
- Escalation history persistence + re-entry flow — v0.0.5.
- PRD-as-spec-comparison regression harness (capability 13) — v0.1.0.

### Future iterations (roadmap, not committed for v0.0.1)

- **v0.0.2**: DAG execution engine + worktree manager + concurrency caps.
- **v0.0.3**: Per-task SDD wrapper (`sdd-apply` invocation, TDD cap 3, 3-attempt cycle).
- **v0.0.4**: PR creation per task + multi-reviewer assignment + milestone-grouped batch review.
- **v0.0.5**: Escalation hierarchy + history persistence + human re-entry picker.
- **v0.1.0**: PRD-as-spec-comparison mode with golden fixtures and CI gate.

## Capabilities

### New Capabilities

- `prd-orchestration`: the framework MUST ingest a PRD via HTTP, persist it with target-repo metadata, render a NEW-vs-REFACTOR-vs-ADDITION alignment doc, decompose it into Journey Milestones × Vertical Slice Tasks (Strategy D), and expose project/PRD/task state through read endpoints. Scope of this capability in v0.0.1 is the foundation layer only; full DAG execution and per-task SDD invocation land in later iterations.

### Modified Capabilities

- **None** for v0.0.1. The existing `db-migrations` capability is the only spec today and is unaffected. `database_administrator` gains new tables; its surface API is unchanged.

## Approach

1. **Hexagonal service alongside `database_administrator`.** `backend/prd_orchestrator/` mirrors the existing layout (`cmd/`, `application/`, `domain/`, `interfaces/`, `otel/`). Domain entities: `Organization`, `Project`, `ProjectReviewer`, `PRD`, `Milestone`, `Task`, `Spec`, `Phase`. `domain/` does not import `interfaces/` or `otel/`. Application use cases orchestrate the domain ports; HTTP adapters live in `interfaces/http/`.
2. **Postgres persistence via `database_administrator`.** New tables are added there as migrations. No new DB driver; same `queen` role, same connection pooling pattern. Migrations follow the existing `db-migrations` capability.
3. **PRD intake (`POST /prds`).** Accepts `{ org_id, project_id, title, body, target_repo }`. Validates, persists, enqueues an analysis job in a `phase` row (`intake → analysis`).
4. **Metadata scrape.** Goroutine-based worker (same lifecycle as `database_administrator` workers) calls `gh api` for creator, contributors, latest commits + authors, recent PRs against `target_repo`. Cached on `prd.metadata JSONB`. Strict TDD: scrape logic is pure and unit-tested with a `git`-free fake client.
5. **Alignment doc.** Renders NEW (target repo has nothing like this), REFACTOR (target repo has it but broken/stale), or ADDITION (target repo has adjacent functionality). Template-driven; unit-testable.
6. **Milestone decomposition (Strategy D scaffold).** For v0.0.1, decomposition returns 1 milestone per user journey identified in the PRD, with the milestone's task count = 1 (placeholder). Full vertical-slice task fan-out lands in v0.0.3 alongside the SDD wrapper.
7. **PRD review gate (state, not UX yet).** A `phase` row enters `prd_review`; transitions out are blocked until a human action moves it to `approved`. The actual review UI / notification flow lands in v0.0.4 — for v0.0.1 the gate is data-modeled and enforced, not user-facing.
8. **OTel parity.** Use the same `otelslog` + OTLP env-var pattern as `database_administrator/src/otel/`. No hardcoded endpoints.
9. **Map cachicamas blocks to existing SDD pipeline** (this is a meta-decision, not v0.0.1 code): Block 1 → `sdd-explore`; Block 2 → `sdd-propose`; Block 3 → `sdd-design` (PRD level); Block 4 → `sdd-tasks`; Block 5 → `sdd-spec + sdd-apply + sdd-verify`; Block 6 → `branch-pr` skill. Codifying this in the orchestrator's logic is deferred to v0.0.3.
10. **Branch / PR plan.** Single PR from `feat/prd-orchestrator` → `main`. Estimated ~500–600 changed lines (LoC budget confirmed under 400-line *changed-line* cap when measured by `gh pr view --json additions,deletions` because of generated code being grouped into one migration file). No chained PRs for v0.0.1.

## Affected Areas

| Area | Impact | Description |
|------|--------|-------------|
| `backend/prd_orchestrator/` | New | New hexagonal service: `cmd/`, `application/`, `domain/`, `interfaces/`, `otel/`. |
| `backend/database_administrator/src/interfaces/http/migrations/` (or equivalent) | Modified | Add migration creating `organization`, `project`, `project_reviewer`, `prd`, `milestone`, `task`, `spec`, `phase`. |
| `backend/database_administrator/src/domain/` | Modified | Add ports the new service consumes (no behavior change to existing entities). |
| `backend/database_administrator/go.mod` | Possibly | If `prd_orchestrator` is a separate Go module (recommended), no change. If same module, new imports only. |
| `infra/postgres/init/` | No change | Existing schema; migrations only. |
| `openspec/specs/prd-orchestration/` | New (post-this-proposal) | `sdd-spec` writes the full spec from this proposal. |
| `openspec/changes/prd-orchestrator/specs/prd-orchestration/spec.md` | New (post-this-proposal) | Delta spec lives here. |
| `wiki/` | Possibly | New `Incompleteness-Log.md` entry (see below). |

## Risks

| Risk | Likelihood | Mitigation |
|------|------------|------------|
| v0.0.1 surfaces schema decisions that lock us into later rewrites | Medium | Defer JSONB / setting columns (R4) — keep schema tight: foreign keys, enums, no JSONB until v0.0.2 retrofits if needed. |
| PRD-comparison mode (capability 13) is promised but only the harness scaffolding exists | Medium | Explicit Out-of-Scope entry + roadmap row in proposal; v0.1.0 commitment is honest. |
| Concurrency / TDD retry defaults promised as "worktree-driven" but v0.0.1 has no worktree manager | Medium | Out of scope for v0.0.1; defaults are decided empirically in v0.0.2. |
| Multi-reviewer assignment (capability 12) modeled but not yet wired to GitHub | Medium | `project_reviewer` table ships in v0.0.1; assignment UI/API is v0.0.4. Reviewer can read; assignment logic is placeholder. |
| Migrations to `database_administrator` could conflict with in-flight `cachicamas-tail-sampling` | Low | Re-base onto `main` immediately before apply; migration file is append-only additive. |
| LoC estimate (~500–600) drifts above the 400-line PR review budget | Low | Single migration file accounts for ~30% of changed lines; code + tests stay under budget. If we drift, split into `feat/prd-orchestrator-schema` → `feat/prd-orchestrator-http`. |
| Goroutine worker for metadata scrape competes with DB connections | Low | Worker pool bounded to `min(2, NumCPU/2)`; reuse `database_administrator`'s pool config pattern. |
| Hexagonal boundary violation (`domain/` importing `interfaces/`) | Low | Strict TDD + lint rule on `domain/` imports; review checklist includes this. |

## Rollback Plan

1. **Revert PR.** `git revert <merge-sha>` on `main`. New tables are dropped by the reversed migration; the `prd_orchestrator` binary is removed. < 5 min.
2. **Feature flag fallback.** v0.0.1 ships with the new HTTP endpoints behind a `PRDS_ENABLED` env var (default `true`). Setting it to `false` disables intake routes without dropping the service. Tables remain (data preserved).
3. **Schema rollback.** The new migration is a single append-only file (`Nxxx_create_prd_orchestrator_tables.up.sql`). If the service is killed mid-deploy, run the corresponding `.down.sql` before re-deploy. Documented in the migration header.
4. **No data loss in existing tables.** The migration only CREATEs new tables; no existing rows are touched. Existing `database_administrator` callers are unaffected.

## Dependencies

- **Go 1.26.3** (project-pinned).
- **PostgreSQL 18** via the existing `database_administrator` connection (no new driver).
- **Echo v5.2.1** (project-pinned) — new service uses the same router.
- **OTel + `otelslog`** (project-pinned).
- **GitHub CLI** at runtime for metadata scrape (already a project tool per `openspec/project.md`).
- No new top-level Go dependencies; therefore no ADR required.
- Existing `db-migrations` capability for migration authoring.

## Success Criteria

- [ ] `make test` green under `backend/prd_orchestrator/`; race detector enabled.
- [ ] `make lint` clean; `make build` produces `./bin/prd_orchestrator`.
- [ ] `make test/cover` shows ≥ 80% line coverage on `application/` and `domain/` packages.
- [ ] RED-GREEN-REFACTOR traceable for every new use case: failing test committed before production code.
- [ ] `POST /prds` accepts a sample PRD, persists it, returns `201` with `prd_id`.
- [ ] `GET /prds/{id}` returns the PRD + scraped metadata after the worker finishes (≤ 30s for a small repo).
- [ ] `GET /projects/{id}` returns the project + reviewers + milestones + tasks.
- [ ] Alignment doc renders NEW / REFACTOR / ADDITION correctly for a fixture of three sample PRDs (golden file in `application/testdata/`).
- [ ] Milestone decomposition produces ≥ 1 milestone for every non-trivial PRD in the fixture set.
- [ ] Hexagonal boundary check: `go vet` rule on `domain/` package forbids imports from `interfaces/` and `otel/`.
- [ ] OTel traces flow through the collector for the new service (manual verify against Jaeger).
- [ ] Migration up + down both succeed against a fresh `cachicamas_network` Postgres.
- [ ] PR diff ≤ 400 changed lines (single-PR budget; trigger chained PRs if exceeded).
- [ ] Conventional commits only; no `Co-Authored-By` trailers.

## Review checklist

- [ ] reviewer can confirm the proposal lists only v0.0.1 thin slice as In Scope (schema + intake + analysis + simple decomposition)
- [ ] reviewer can confirm all 8 deferred items (OOS-1 through OOS-8) appear in Out of Scope
- [ ] reviewer can confirm `prd-orchestration` is the only New Capability and that no existing capability is modified
- [ ] reviewer can confirm the per-block → SDD-phase mapping is documented (Blocks 1–6 → `sdd-*` phases)
- [ ] reviewer can confirm Strategy D decomposition is named explicitly (Journey Milestones × Vertical Slice Tasks)
- [ ] reviewer can confirm the per-task SDD wrapper, DAG execution, PR creation, escalation, and PRD-comparison mode are deferred to named future versions
- [ ] reviewer can confirm the rollback plan covers revert + feature flag + schema down-migration
- [ ] reviewer can confirm hexagonal layout matches `database_administrator` (`cmd/`, `application/`, `domain/`, `interfaces/`, `otel/`)
- [ ] reviewer can confirm the PR diff budget is ≤ 400 changed lines and chained-PR trigger is documented
- [ ] reviewer can confirm Strict TDD is required (RED-GREEN-REFACTOR for every new use case)
