# cachicamas

<div align="center">
  <picture>
    <source
      type="image/png"
      srcset="docs/assets/cachicamas-logo@2x.png 2x, docs/assets/cachicamas-logo@1x.png 1x"
    />
    <img
      src="docs/assets/cachicamas-logo.png"
      alt="cachicamas — Witsaba Software Development Framework"
      width="280"
      height="280"
      loading="eager"
      decoding="async"
      style="max-width: 280px; width: 100%; height: auto; display: block; margin: 0 auto;"
    />
  </picture>
</div>

> **Witsaba's Software Development Framework** — an agent-first, PRD-driven orchestrator that wraps the existing SDD pipeline so any team inside the org can ship a project from intake to merged PR without leaving the framework. v0.0.1 is a thin slice: schema + PRD intake + metadata analysis + simple milestone decomposition. The framework is the product; the SDD pipeline is its execution engine.

---

## Agent-First Documentation

This README follows an **Agent-First** pattern so an agent (or a human in a hurry) can find what it needs in three passes, top to bottom:

1. **Resumed Table of Contents** — a one-screen summary table. Read this first; if your question is answered here, stop.
2. **Table of Contents** — the full structured index with anchors. Read this to jump to a specific section.
3. **Content** — the actual sections, each self-contained and reviewable.

The pattern is **portable**: section [11. The Agent-First Doc Pattern](#11-the-agent-first-doc-pattern) explains how to reuse it in any doc and how to adapt it to code (file-level summary → symbol index → implementation).

---

## Resumed Table of Contents

| # | Topic | TL;DR |
|---|-------|-------|
| 1 | [What is cachicamas?](#1-what-is-cachicamas) | Witsaba's SDLC framework, v0.0.1, wraps `/sdd-*` as its per-task engine. |
| 2 | [Why does it exist?](#2-why-does-it-exist) | To enable a competitive engineering org with many internal companies, all running the same playbook. |
| 3 | [v0.0.1 Scope](#3-v001-scope) | Thin slice: schema + PRD intake + metadata analysis + 1:1 milestone→task decomposition. |
| 4 | [Architecture](#4-architecture) | Hexagonal Go services (`database_administrator`, `prd_orchestrator`) + Qwik frontend on Postgres 18. |
| 5 | [Repository Layout](#5-repository-layout) | `backend/`, `frontend/`, `openspec/`, `infra/`, `scripts/`, `wiki/`, `docs/`, `spikes/`, `.worktrees/`. |
| 6 | [Tech Stack](#6-tech-stack) | Go 1.26.3, Echo v5.2.1, Postgres 18, OpenTelemetry → Jaeger v2, docker-compose v2, Qwik 1.20.0, pnpm 11. |
| 7 | [SDD Pipeline Mapping](#7-sdd-pipeline-mapping) | cachicamas blocks 1–6 map to `sdd-explore`, `sdd-propose`, `sdd-design`, `sdd-tasks`, `sdd-spec`+`sdd-apply`+`sdd-verify`, `branch-pr`. |
| 8 | [Conventions](#8-conventions) | Conventional commits (no Co-Authored-By), hex layout, tools to `./bin/`, `slog`+`otelslog`, triple-pinned images, ADR for new top-level deps. |
| 9 | [Testing & TDD](#9-testing--tdd) | Strict TDD enabled (`make test` = `go test ./... -race -v`); tests next to code; coverage threshold 0 (not enforced yet). |
| 10 | [Quick Start](#10-quick-start) | `make test`, `make lint`, `make build`, `docker compose up`; Qwik dev with `pnpm dev`. |
| 11 | [The Agent-First Doc Pattern](#11-the-agent-first-doc-pattern) | Resumed TOC → TOC → Content. Reuse on any doc; adapt to code (file summary → symbol list → impl). |
| 12 | [Review Checklist](#12-review-checklist) | Mandatory reviewer checklist, item by item. |
| 13 | [Next Step](#13-next-step) | Read `openspec/project.md`, then `wiki/Incompleteness-Log.md`, then start an SDD cycle. |

---

## Table of Contents

1. [What is cachicamas?](#1-what-is-cachicamas)
2. [Why does it exist?](#2-why-does-it-exist)
3. [v0.0.1 Scope](#3-v001-scope)
4. [Architecture](#4-architecture)
5. [Repository Layout](#5-repository-layout)
6. [Tech Stack](#6-tech-stack)
7. [SDD Pipeline Mapping](#7-sdd-pipeline-mapping)
8. [Conventions](#8-conventions)
9. [Testing & TDD](#9-testing--tdd)
10. [Quick Start](#10-quick-start)
11. [The Agent-First Doc Pattern](#11-the-agent-first-doc-pattern)
12. [Review Checklist](#12-review-checklist)
13. [Next Step](#13-next-step)

---

## 1. What is cachicamas?

**cachicamas** is Witsaba's Software Development Framework. It is a **PRD-driven orchestrator** that wraps the existing `/sdd-*` pipeline (Spec-Driven Development) so any team inside the Witsaba org can run a project from PRD intake to merged pull request through a single, predictable engine.

It is NOT a replacement for the SDD pipeline. The SDD pipeline (explore → propose → spec → design → tasks → apply → verify → archive) is the **execution engine** cachicamas drives. cachicamas adds the **missing upper layer**: organization → project → PRD → milestone → task → spec → phase hierarchy, intake, analysis, and decomposition.

| Identity | Value |
|----------|-------|
| Project | `cachicamas` |
| Repo | [`witsaba/cachicamas`](https://github.com/witsaba/cachicamas) |
| Primary branch | `main` |
| Owner | braejan (founder, Witsaba) |
| Current version | v0.0.1 (thin slice) |
| License | See `LICENSE` |

## 2. Why does it exist?

Witsaba is building a **competitive engineering organization** with multiple internal companies. Each company has its own product roadmap, but they all share the same engineering process. cachicamas is the **shared process made software**: every PRD lands in the same database, gets analyzed the same way, decomposes into the same shape, and flows through the same review gates. The output is consistent quality and visible state across every project — no matter which team owns it.

The framework exists to make the *process* the *product*. Without it, every team re-invents intake, decomposition, and review — and the org loses the leverage of shared learnings.

## 3. v0.0.1 Scope

v0.0.1 is intentionally a **thin slice**. We want the smallest useful product that exercises every part of the engine end-to-end.

**In scope (v0.0.1):**

- Schema for the full hierarchy (`organization`, `project`, `requirement`, `requirement_spike`, `milestone`, `task`, `spec`, `spec_phase`) — even when some tables are not yet read by the orchestrator.
- PRD intake: capture, persist, version.
- Metadata analysis: read the repo, extract facts (modules, package paths, linter, test command).
- Simple milestone decomposition (Strategy D: Journey Milestones × Vertical Slice Tasks) — collapsed to 1:1 (one task per milestone) for v0.0.1; the full slice fan-out is a follow-up.

**Out of scope (deferred to v0.0.2 → v0.1.0):**

| Item | Why deferred |
|------|--------------|
| R1 | Frozen interface spec — needs more usages before freezing |
| R2 | Skip self-analysis when repo == framework — needs framework-comparison logic |
| R3 | Golden fixtures for PRD-comparison mode — needs reference PRDs |
| R4 | `project.setting JSONB` storage — needs concrete settings list |
| R5 | Specific TDD retry defaults — needs empirical data |
| R6 | Specific concurrency defaults — needs observed worktree capacity |
| MCP-based Agent-as-a-Service packaging | Needs MCP infrastructure decisions |
| Multi-company competitive org | Needs org modeling decisions |

These live in `openspec/changes/prd-orchestrator/proposal.md` → "Out of Scope". Revisit when each iteration starts.

## 4. Architecture

The framework is built as **hexagonal Go services** sharing a Postgres database. Each service owns its own HTTP layer, application layer, domain layer, and adapters — and communicates only through Postgres (no service-to-service HTTP).

| Service | Purpose | Status |
|---------|---------|--------|
| `database_administrator` | Migration runner + observability scaffolding. Owns all schema migrations under `src/migration/sql/`. | Live on `main` |
| `prd_orchestrator` | The framework. PRD intake, analysis, decomposition. v0.0.1 thin slice. | In development (PR #11 open, chained-PR strategy) |
| `frontend` (Qwik 1.20.0) | Operator UI for the orchestrator. | Scaffolded on `feat/qwik-frontend` worktree branch |

**Hexagonal layout** (under `backend/<service>/src/`):

```
cmd/server/         → entrypoint (main.go)
application/        → use cases (services)
domain/             → entities and contracts (no I/O)
interfaces/         → adapters (HTTP handlers, repos, scrapers)
otel/               → observability wiring (logging, tracing)
migration/sql/      → goose-style .sql migrations (in database_administrator only)
```

The orchestrator reuses the migration runner from `database_administrator` via `go.mod replace ../database_administrator` — it does NOT call it over HTTP.

**Hierarchy:**

```
organization
  └── project (1+ reviewers)
        └── PRD (requirement)
              ├── requirement_spike (0..*)
              ├── milestone (1..*)
              │     └── task (1..*; v0.0.1 = 1:1 with milestone)
              ├── spec (0..*; created but unused in v0.0.1, active from v0.0.3)
              │     └── spec_phase (1..*)
              └── spec_phase references the task via task_id
```

**Branch strategy:** parent worktree branch per major change (e.g., `feat/prd-orchestrator`) + child branches per PR (e.g., `feat/prd-orchestrator-schema`).

## 5. Repository Layout

| Path | Contents |
|------|----------|
| `backend/` | Go services. `database_administrator/` (live) and `prd_orchestrator/` (in flight). |
| `frontend/` | Qwik 1.20.0 app. Scaffolded on the `feat/qwik-frontend` worktree branch. |
| `openspec/` | OpenSpec artifacts. `project.md` (bootstrap), `AGENTS.md`, `config.yaml`, `changes/<change>/`, `specs/`. |
| `infra/` | Infrastructure configs. Postgres init scripts under `infra/postgres/init/`. |
| `scripts/` | Utility shell scripts. |
| `wiki/` | Living documentation. Currently houses `Incompleteness-Log.md`. |
| `docs/` | Project-level docs. `assets/` for images (this README's logo lives here). |
| `spikes/` | Exploratory / throwaway work. Not promoted to `openspec/changes/`. |
| `.worktrees/` | Local git worktrees (untracked). Created per multi-PR change. |
| `.atl/` | Agent Teams Lite config (skill registry). |
| `.claude/` | Claude Code local config. |
| `docker-compose.yaml` | Single-node local dev stack on `cachicamas_network`. |
| `Makefile` (per service) | `make test`, `make lint`, `make build`, `make fmt`, `make vet`, `make test/cover`. |

## 6. Tech Stack

| Layer | Tool | Version |
|-------|------|---------|
| Language (backend) | Go | 1.26.3 |
| HTTP framework | `github.com/labstack/echo/v5` | v5.2.1 |
| Database | PostgreSQL (alpine) | 18-alpine3.24 |
| Telemetry | OpenTelemetry (`otel`, `otelslog`, `sdk/log`, `sdk/trace`) | OTLP/gRPC exporters |
| Tracing backend | Jaeger | v2.19.0 |
| Collector | OTel Collector (contrib) | 0.137.0 |
| Orchestration | docker-compose | v2 |
| Frontend | Qwik (`@cachicamas/frontend`) | 1.20.0 |
| Frontend package manager | pnpm | 11.x (note: Corepack not installed; use `npm i -g pnpm`) |
| Frontend build | Vite | 7.x |
| Linter | `golangci-lint` | v2.9.0 |
| Migrations | goose (via `database_administrator`) | embedded |

**Database user:** `queen` (NOSUPERUSER, CREATEROLE, CREATEDB, REPLICATION). Provisioned once via `infra/postgres/init/01-init.sql` (mounted at `/docker-entrypoint-initdb.d`, first-boot only).

**Pinning discipline:** every base image is triple-pinned (`image:tag@digest`).

## 7. SDD Pipeline Mapping

cachicamas runs **on top of** the `/sdd-*` pipeline. Each cachicamas block delegates to one or more SDD phases. The orchestrator owns the *state machine*; the SDD sub-agents own the *artifact production*.

| cachicamas block | SDD phase(s) | Output |
|------------------|--------------|--------|
| 1. Intake | `sdd-explore` | Exploration report |
| 2. Proposal | `sdd-propose` | `proposal.md` |
| 3. Design (PRD level) | `sdd-design` | `design.md` |
| 4. Task decomposition | `sdd-tasks` | `tasks.md` |
| 5. Execution | `sdd-spec` + `sdd-apply` + `sdd-verify` | `spec.md`, code, `verify-report.md` |
| 6. Delivery | `branch-pr` (or `chained-pr`) | Merged PR |

**Concurrency** is driven by **worktree capacity**, not fixed numbers. v0.0.1 reads `project.setting->>'max_concurrent_tasks'` (JSONB) — default is unlimited. The orchestrator spawns SDD sub-agents inside worktrees and respects the cap.

## 8. Conventions

These are the standing rules. They are not aspirational — they are enforced.

- **Conventional commits.** Required. No exceptions.
- **No `Co-Authored-By` trailer.** No AI attribution in commit messages. (Repo-wide rule.)
- **Hexagonal layout.** Every Go service follows `cmd / application / domain / interfaces / otel`.
- **Tools to `./bin/`.** `LOCALBIN` keeps `$GOPATH/bin` clean.
- **Logging: `slog` + `otelslog`.** OTLP via env vars. No hardcoded endpoints.
- **Triple-pinned base images.** `image:tag@digest`.
- **New top-level dependencies require an ADR.** Add it to `openspec/changes/<change>/design.md` before adding the dep.
- **Tests next to the code they exercise.** No separate `tests/` directory per service.
- **Worktree discipline.** Major changes get a parent worktree branch + child branches per PR. Sub-agents MUST receive explicit absolute worktree paths.
- **No auto-merge.** The user reviews and merges. The orchestrator never merges on its own.

## 9. Testing & TDD

Strict TDD is **enabled**. This is non-negotiable for every change.

| Capability | Value |
|------------|-------|
| Test runner | `go test ./...` (Makefile target: `make test` adds `-race -v`) |
| Coverage command | `make test/cover` (writes `coverage.out`, atomic mode) |
| Linter command | `make lint` (auto-installs `golangci-lint` if missing) |
| Vet command | `make vet` |
| Format command | `make fmt` (`gofmt` + `goimports`) |
| Build command | `make build` |
| Coverage threshold | 0 (no enforcement at project level yet) |
| Strict TDD | **enabled** (RED → GREEN → REFACTOR; tests first) |

`openspec/config.yaml` declares `apply.tdd: true`. Integration tests against the compose stack grow as services stabilize.

**TDD flow for a new task:**

1. **RED** — write the failing test in `interfaces/http/<thing>_handler_test.go` (or whichever layer owns the behavior).
2. **GREEN** — write the smallest code that makes the test pass.
3. **REFACTOR** — clean up without breaking the test.

Boundary tests (e.g., `tests/domain_imports_test.go`) ensure the hexagonal layers stay clean — domain must not import interfaces or adapters. These are the LAST file written in a separate commit so early commits don't fail the boundary check.

## 10. Quick Start

```bash
# 1. Clone
git clone https://github.com/witsaba/cachicamas.git
cd cachicamas

# 2. Bring up the dev stack (Postgres + OTel Collector + Jaeger)
docker compose up -d

# 3. Run backend tests (strict TDD gate)
cd backend/database_administrator
make test      # go test ./... -race -v
make lint      # golangci-lint run
make build     # binary to ./bin/database_administrator

# 4. (Optional) Run the frontend dev server
cd ../../frontend
pnpm dev       # http://localhost:5173/

# 5. (Optional) Open Jaeger for traces
open http://localhost:16686
```

If `pnpm` is missing: `npm i -g pnpm` (Corepack is not installed on this machine — known issue, recorded in engram).

## 11. The Agent-First Doc Pattern

The pattern this README follows is **portable**. Three passes, top to bottom, so an agent (or a hurried human) can find what it needs at the level of detail it needs.

### The three passes

| Pass | What | Purpose | Cost to read |
|------|------|---------|--------------|
| **Pass 1: Resumed TOC** | A small table with one row per topic and a one-line answer. | Scan everything at a glance. Decide whether to keep reading. | One screen. |
| **Pass 2: Full TOC** | The complete structured index with anchors. | Jump straight to a specific section. | One screen. |
| **Pass 3: Content** | The detailed sections, each self-contained. | Read the depth you actually need. | Variable. |

### Adapting to any document

| Doc type | Pass 1 (Resumed TOC) | Pass 2 (TOC) | Pass 3 (Content) |
|----------|----------------------|--------------|------------------|
| README | Topic table with one-line answers | Anchor-linked headings | Detailed sections |
| Architecture doc | Decision summary table | Section index | Trade-off analysis |
| Onboarding guide | What you'll learn table | Step-by-step index | Step instructions |
| ADR | Status table (Proposed/Accepted/Superseded) | ADR index | Full ADR |
| PR description | Impact summary | Change list | Detailed diff context |
| API reference | Resource table | Endpoint index | Endpoint reference |

### Adapting to code (slight modification)

For code, the same three passes apply — the "resumed TOC" becomes a **module/file summary**, the "full TOC" becomes a **symbol index**, and "content" is the **implementation**.

| Pass | In a Go file | In a package | In a service |
|------|--------------|--------------|--------------|
| **Pass 1: Resumed view** | One-line `// Package ...` doc comment above the package + a 3–5 row table at the top of `main.go` listing each top-level symbol with its purpose | Package-level `doc.go` with a symbol summary table | Service `README.md` (this very pattern) + `application/services.go` index |
| **Pass 2: Symbol index** | Comment header listing exported symbols (`// Exported: NewX, DoY, ZType`) | `package_symbols.go` (or generated) listing every symbol with a one-line summary | Service `Makefile` targets + `cmd/server/main.go` wiring |
| **Pass 3: Implementation** | The actual functions and types | The actual `.go` files | The actual hex layout |

**Rule of thumb:** if an agent can answer 80% of "what is this?" by reading pass 1 alone, the doc is well-shaped. Pass 2 exists so it doesn't have to grep. Pass 3 is the detail it reads only when pass 1 + 2 say "yes, this is what I need."

## 12. Review Checklist

A reviewer can confirm each item below without re-reading the whole README. This is the self-grade.

- [ ] reviewer can confirm the logo renders at `docs/assets/cachicamas-logo.png`
- [ ] reviewer can confirm cachicamas is identified as v0.0.1 thin slice, not a finished framework
- [ ] reviewer can confirm the SDD pipeline mapping (blocks 1–6 → `sdd-*` phases) is accurate
- [ ] reviewer can confirm the v0.0.1 out-of-scope items match `openspec/changes/prd-orchestrator/proposal.md`
- [ ] reviewer can confirm the tech stack table matches `openspec/project.md`
- [ ] reviewer can confirm the hexagonal layout matches `backend/database_administrator/src/`
- [ ] reviewer can confirm "no Co-Authored-By" is stated and the rule is honored
- [ ] reviewer can confirm Strict TDD is documented as enabled
- [ ] reviewer can confirm the Resumed TOC fits on one screen and answers "what is this repo?" in one line
- [ ] reviewer can confirm the Agent-First pattern section (11) is reusable on docs and adaptable to code
- [ ] reviewer can confirm there is a wiki `Incompleteness-Log.md` and this README does not contradict it
- [ ] reviewer can confirm the Quick Start commands match the actual Makefile targets

## 13. Next Step

1. Read [`openspec/project.md`](openspec/project.md) for the bootstrap artifact (stack, conventions, testing discipline).
2. Read [`wiki/Incompleteness-Log.md`](wiki/Incompleteness-Log.md) to see what the wiki is honest about not yet knowing.
3. Pick an active change under `openspec/changes/` and run its next phase (`/sdd-continue` or `/sdd-ff`).
4. Or start a new change with `/sdd-new <name>` — the orchestrator will route you to `sdd-explore` first.

If you are an agent: before changing code, search engram (`mem_search` with project `cachicamas`) for prior context on the topic. Save every decision, bug fix, and convention via `mem_save` — the next session depends on it.
