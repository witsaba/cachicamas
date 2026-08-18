# cachicamas

<div align="center">
  <picture>
    <source
      type="image/png"
      srcset="docs/assets/cachicamas-logo@2x.png 2x, docs/assets/cachicamas-logo@1x.png 1x"
    />
    <img
      src="docs/assets/cachicamas-logo.png"
      alt="cachicamas — a multiplayer agentic system for building and running a company"
      width="280"
      height="280"
      loading="eager"
      decoding="async"
      style="display:block; margin:0 auto; width:280px; height:280px; max-width:80vw; aspect-ratio:1 / 1; border-radius:50%; object-fit:cover;"
    />
  </picture>
</div>

> **cachicamas is a multiplayer agentic system for building and running a company.** Everything a company needs — database administration, finance, marketing, ticketing, software development — exists as cooperating specialist agents that employees talk and work with. It is usable by any company; [Witsaba](https://witsaba.com/) is its first user, not its boundary. Identity ratified in [ADR 0009](docs/adr/0009-redefine-cachicamas-as-a-multiplayer-agentic-system.md).

---

## Agent-First Documentation

This README follows an **Agent-First** pattern so an agent (or a human in a hurry) can find what it needs in three passes, top to bottom:

1. **Resumed Table of Contents** — a one-screen summary table. Read this first; if your question is answered here, stop.
2. **Table of Contents** — the full structured index with anchors. Read this to jump to a specific section.
3. **Content** — the actual sections, each self-contained and reviewable.

The pattern is **portable**: section [12. The Agent-First Doc Pattern](#12-the-agent-first-doc-pattern) explains how to reuse it in any doc and how to adapt it to code (file-level summary → symbol index → implementation).

---

## Resumed Table of Contents

| # | Topic | TL;DR |
| --- | ------- | ------- |
| 1 | [What is cachicamas?](#1-what-is-cachicamas) | A multiplayer agentic system for building and running a company ([ADR 0009](docs/adr/0009-redefine-cachicamas-as-a-multiplayer-agentic-system.md)). |
| 2 | [Why does it exist?](#2-why-does-it-exist) | Every company function as cooperating specialist agents; any company can run it — Witsaba is the first user. |
| 3 | [The Agent Stack](#3-the-agent-stack) | Layer 1 model adapter (complete, 42/42) · Layer 2 portable agent runtime (14/24) · Layer 3 archetype layer (not started, 0/25). One MCP server + one owning archetype per business system. |
| 4 | [Architecture](#4-architecture) | Hexagonal Go services (`database_administrator`, `workspace_syncer`) + the layered `agent` module + Qwik frontend on Postgres 18. |
| 5 | [Repository Layout](#5-repository-layout) | `backend/`, `frontend/`, `docs/`, `openspec/`, `infra/`, `scripts/`, `spikes/`, `.worktrees/`. |
| 6 | [Tech Stack](#6-tech-stack) | Go 1.26.3, Echo v5.2.1, Postgres 18, OpenTelemetry → Jaeger v2, docker-compose v2, Qwik 1.20.0, pnpm 11. |
| 7 | [SDD: the Engineering Process](#7-sdd-the-engineering-process) | SDD is how cachicamas is built (`openspec/`, `/sdd-*` skills). It is not the product. |
| 8 | [Conventions](#8-conventions) | Conventional commits (no Co-Authored-By), hex layout, tools to `./bin/`, `slog`+`otelslog`, triple-pinned images, ADR for new top-level deps. |
| 9 | [Testing & TDD](#9-testing--tdd) | Strict TDD enabled (`make test` = `go test ./... -race -v`); tests next to code; coverage threshold 0 (not enforced yet). |
| 10 | [Quick Start](#10-quick-start) | `make test`, `make lint`, `make build`, `docker compose up`; Qwik dev with `pnpm dev`. |
| 11 | [Where This Is Going](#11-where-this-is-going) | Finish Layer 2 → ship the coding archetype → the DBA archetype over MCP → further archetypes (tickets, finance, marketing). |
| 12 | [The Agent-First Doc Pattern](#12-the-agent-first-doc-pattern) | Resumed TOC → TOC → Content. Reuse on any doc; adapt to code (file summary → symbol list → impl). |
| 13 | [Review Checklist](#13-review-checklist) | Mandatory reviewer checklist, item by item. |
| 14 | [Next Step](#14-next-step) | Read `openspec/project.md`, the v2 architecture reference, ADR 0009, then the milestone docs. |

---

## Table of Contents

1. [What is cachicamas?](#1-what-is-cachicamas)
2. [Why does it exist?](#2-why-does-it-exist)
3. [The Agent Stack](#3-the-agent-stack)
4. [Architecture](#4-architecture)
5. [Repository Layout](#5-repository-layout)
6. [Tech Stack](#6-tech-stack)
7. [SDD: the Engineering Process](#7-sdd-the-engineering-process)
8. [Conventions](#8-conventions)
9. [Testing & TDD](#9-testing--tdd)
10. [Quick Start](#10-quick-start)
11. [Where This Is Going](#11-where-this-is-going)
12. [The Agent-First Doc Pattern](#12-the-agent-first-doc-pattern)
13. [Review Checklist](#13-review-checklist)
14. [Next Step](#14-next-step)

---

## 1. What is cachicamas?

**cachicamas** is a **multiplayer agentic system for building and running a company**. Every function a company needs to operate — database administration, finance, marketing, ticketing, software development — exists as a cooperating specialist agent that employees talk and work with. The identity is ratified in [ADR 0009](docs/adr/0009-redefine-cachicamas-as-a-multiplayer-agentic-system.md).

It is built on a three-layer agent stack (section [3](#3-the-agent-stack)): a vendor-portable model adapter, a portable agent runtime that can run *any* agent, and an **archetype layer** where each specialist agent lives. The **coding archetype** — an agent that codes, comparable to Pi or Opencode — is the first archetype; a **Database Administrator archetype** is planned next.

cachicamas is usable by **any company**. [Witsaba](https://witsaba.com/) is its first user, not its boundary.

| Identity | Value |
| ---------- | ------- |
| Project | `cachicamas` |
| Repo | [`witsaba/cachicamas`](https://github.com/witsaba/cachicamas) |
| Primary branch | `main` |
| Owner | braejan (founder, Witsaba) |
| Status | Pre-release — Layer 1 complete, Layer 2 in progress (see [section 3](#3-the-agent-stack)) |
| License | See `LICENSE` |

## 2. Why does it exist?

Running a company means running many systems — databases, tickets, books, campaigns, codebases — and today every one of them demands its own specialist attention. cachicamas makes each of those functions a **specialist agent** that any employee can talk to and work with, and makes the agents **cooperate**: the ticket agent can ask the DBA agent for a schema change; the coding agent can pick up the work the ticket describes.

The leverage comes from the stack's shape. The runtime (Layer 2) is pure mechanism and cannot tell which agent it is running, so every new specialist — an archetype — is an *additive* change: new policy, tools, resources, and frontend standing on an unchanged runtime. One company function at a time, the system grows without rewrites.

Witsaba runs cachicamas first and feeds what it learns back into the system, but nothing in the design is Witsaba-specific.

## 3. The Agent Stack

The stack lives in the `backend/agent` Go module ([ADR 0005](docs/adr/0005-promote-agent-stack-to-own-module.md)). The current vocabulary of record is [the v2 architecture reference](docs/architecture/0001-cachicamas-agent-stack-v2.md), as amended by [ADR 0009](docs/adr/0009-redefine-cachicamas-as-a-multiplayer-agentic-system.md).

| Layer | Package | What it is | Status |
| --- | --- | --- | --- |
| **Layer 1 — the model adapter** | `src/ai/` | LLM provider connector; one adapter per vendor, vendor-portable contract. | **Complete — 42/42 milestones** ([doc 0002](docs/architecture/milestones/0002-cachicamas-ai-layer-1-task-graph.md)) |
| **Layer 2 — the portable agent runtime** | `src/agent/` | Runs *any* agent; pure mechanism, no judgement. Loop + harness. | **In progress — 14/24 milestones** ([doc 0003](docs/architecture/milestones/0003-cachicamas-agent-layer-2-task-graph.md)) |
| **Layer 3 — the archetype layer** | `src/coding/` today | Where specialist agents live. An **archetype** is the implementation of one specialist agent: its policy, tools, resources, persistence, and frontend. | **Not started — 0/25 milestones** ([doc 0004](docs/architecture/milestones/0004-cachicamas-coding-layer-3-task-graph.md)) |

Three rules of the stack, all decided and cited:

- **The coding archetype is the first occupant of Layer 3, not its definition** ([ADR 0009 § D2](docs/adr/0009-redefine-cachicamas-as-a-multiplayer-agentic-system.md)). Layer 3 is a position in the stack; a DBA archetype, a ticket archetype, or a finance archetype occupies the same position with different policy, tools, and frontend on an unchanged runtime.
- **Layer 3 is not the top of the stack** ([ADR 0009 § D3](docs/adr/0009-redefine-cachicamas-as-a-multiplayer-agentic-system.md)). Higher layers may be added — cross-archetype coordination is the obvious future occupant of a layer above 3.
- **Each business system runs its own MCP server and has exactly one owning archetype** ([ADR 0009 § D4](docs/adr/0009-redefine-cachicamas-as-a-multiplayer-agentic-system.md)). The Database Administrator archetype will consume the existing `backend/database_administrator` service through an MCP client; a future ticket system runs its own MCP server owned by a ticket archetype. This rides on [ADR 0005 § D1](docs/adr/0005-promote-agent-stack-to-own-module.md#d1--dependency-rule-v2): Layer 3 reaches other modules over the network only, never by import. Each business system owns its own tables; any archetype asks the DBA archetype for database work ([ADR 0009 § D6](docs/adr/0009-redefine-cachicamas-as-a-multiplayer-agentic-system.md)).

## 4. Architecture

The business systems are **hexagonal Go services** sharing a Postgres database. Each service owns its own HTTP layer, application services, domain layer, and adapters. Most inter-service coupling is through Postgres; `database_administrator → workspace_syncer` is the one direct HTTP call (`src/infrastructure/workspacesyncer/client.go`).

The agent stack is the exception to "hexagonal": `backend/agent` is a **layered** module (see [ADR 0005](docs/adr/0005-promote-agent-stack-to-own-module.md)), not a hexagonal service, and hexagonal review rules do not apply to it.

| Service | Purpose | Status |
| --------- | --------- | -------- |
| `database_administrator` | The backend API. Identity/OAuth callback, organizations, workspaces, sync jobs + SSE, GitHub adapter, prompts, skills. Also owns all schema migrations under `src/migration/sql/`. Will be fronted by the Database Administrator archetype over MCP ([ADR 0009 § D5](docs/adr/0009-redefine-cachicamas-as-a-multiplayer-agentic-system.md)). | Live on `main` |
| `workspace_syncer` | Git clone + validate worker. `POST /internal/clone-and-validate`, HMAC callback to `database_administrator`. | Live on `main` |
| `agent` | The layered agent stack — Layer 1 model adapter (`src/ai/`), Layer 2 portable agent runtime (`src/agent/`), Layer 3 archetype layer (`src/coding/` — the coding archetype, its first occupant), CLI composition root (`src/cmd/cachicamas/`). See [ADR 0004](docs/adr/0004-adopt-tau-3-layer-agentic-architecture.md) as amended by [ADR 0005](docs/adr/0005-promote-agent-stack-to-own-module.md) and [ADR 0009](docs/adr/0009-redefine-cachicamas-as-a-multiplayer-agentic-system.md), with the architecture in [the v2 reference](docs/architecture/0001-cachicamas-agent-stack-v2.md). | Layer 1 complete (42/42) · Layer 2 in progress (14/24) · Layer 3 not started |
| `frontend` (Qwik 1.20.0) | Operator UI. Auth.js GitHub login, workspaces, Prompt Studio, Skill Studio. | Live on `main` |

**Hexagonal layout** (under `backend/<service>/src/`):

```
cmd/server/         → entrypoint (main.go)
application/        → use cases (services)
domain/             → entities and contracts (no I/O)
interfaces/         → adapters (HTTP handlers, repos, scrapers)
otel/               → observability wiring (logging, tracing)
migration/sql/      → goose-style .sql migrations (in database_administrator only)
```

**Cross-module imports have a real cost.** Each service's Docker build context is its own directory (`docker-compose.yaml`), and each `Dockerfile` copies only that module's `go.mod`, `go.sum` and `src/`. The first cross-module import therefore also requires moving the compose build context up to `./backend` and rewriting every `COPY` path. `backend/agent` does not import, and is not imported by, any other module — see [ADR 0005 § D1](docs/adr/0005-promote-agent-stack-to-own-module.md#d1--dependency-rule-v2). Archetypes reach business systems over the network (MCP), never by import.

**Branch strategy:** parent worktree branch per major change (e.g., `feat/agent-layer2-wave3`) + child branches per PR.

## 5. Repository Layout

| Path | Contents |
| ------ | ---------- |
| `backend/` | Go modules. `database_administrator/` (API + migrations), `workspace_syncer/` (git worker), `agent/` (the layered agent stack). Each is a separate Go module with its own `Makefile` and `.golangci.yml`; `make test` runs per module. |
| `frontend/` | Qwik 1.20.0 operator UI. |
| `docs/adr/` | Architecture Decision Records, with the external source material they cite under `docs/adr/references/`. |
| `docs/architecture/` | cachicamas-authored architecture documents and the milestone task graphs under `milestones/`. |
| `docs/prd/` | Product requirements documents. PRD 0001 (the delivery loop) is accepted and awaits re-scope under [ADR 0009 § D7](docs/adr/0009-redefine-cachicamas-as-a-multiplayer-agentic-system.md). |
| `docs/assets/` | Images (this README's logo lives here). |
| `openspec/` | OpenSpec artifacts. `project.md` (bootstrap), `AGENTS.md`, `config.yaml`, `changes/<change>/`, `specs/` (the promoted capability register). |
| `infra/` | Infrastructure configs. Postgres init scripts under `infra/postgres/init/`. |
| `scripts/` | Utility shell scripts. |
| `spikes/` | Exploratory / throwaway work. Not promoted to `openspec/changes/`. |
| `.worktrees/` | Local git worktrees (untracked). Created per multi-PR change. |
| `.atl/` | Agent Teams Lite config (skill registry). |
| `.claude/` | Claude Code local config. |
| `docker-compose.yaml` | Single-node local dev stack on `cachicamas_network`. |
| `Makefile` (per module) | `make test`, `make lint`, `make build`, `make fmt`, `make vet`, `make test/cover`. |

## 6. Tech Stack

| Layer | Tool | Version |
| ------- | ------ | --------- |
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

## 7. SDD: the Engineering Process

**SDD (Spec-Driven Development) is the engineering process cachicamas is built with. It is not the product** ([ADR 0009 § D1](docs/adr/0009-redefine-cachicamas-as-a-multiplayer-agentic-system.md)).

Every change flows through the `/sdd-*` skill pipeline (explore → propose → spec → design → tasks → apply → verify → archive), leaves its artifacts under `openspec/changes/<change>/`, and promotes its requirements into `openspec/specs/` — the living capability register — on archive. Plans are milestone task graphs under `docs/architecture/milestones/`, governed by the DAG convention of [ADR 0007](docs/adr/0007-adopt-dag-convention-for-task-graphs.md); the delivery loop of [ADR 0008](docs/adr/0008-adopt-the-cachicamas-delivery-loop.md) is the machinery being built to run them.

Start with [`openspec/project.md`](openspec/project.md) and `openspec/AGENTS.md` for the rules the pipeline enforces.

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
| ------------ | ------- |
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

Boundary tests (e.g., `src/domain/imports_test.go`, and the agent module's forward/reverse import guards) keep the layer rules mechanical — domain must not import interfaces or adapters, and no hexagonal service imports the agent module. These are the LAST file written in a separate commit so early commits don't fail the boundary check.

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

## 10.1 Deploy to VPS

The compose stack supports a **VPS profile** that exposes only the frontend (port 3015) to the host. All other services (Postgres, Go binary, Jaeger, OTel collector) stay on the private `cachicamas_network`. nginx inside the frontend container reverse-proxies `/api/*` to the Go binary, so the browser only sees one origin.

```bash
# Dev local (all services published, full debug surface):
docker compose up -d --build

# VPS (only the frontend in :3015; rest in the private network):
docker compose -f docker-compose.yaml -f docker-compose.vps.yaml up -d --build
```

For VPS production, adjust `CORS_ALLOW_ORIGINS` in `.env` to the real public domain (`https://cachicamas.example.com`). The frontend stays accessible at `http://<host>:3015/`. The browser talks to the Go binary via the internal nginx reverse-proxy (`/api/*`); no cross-origin from the browser's perspective, so CORS is a non-issue in the normal flow.

## 11. Where This Is Going

The roadmap, in order — each step is a milestone document, planned or to be written:

1. **Finish Layer 2** — the portable agent runtime, 14/24 milestones shipped ([doc 0003](docs/architecture/milestones/0003-cachicamas-agent-layer-2-task-graph.md)).
2. **Ship the coding archetype** — the first occupant of Layer 3, 25 milestones planned ([doc 0004](docs/architecture/milestones/0004-cachicamas-coding-layer-3-task-graph.md)).
3. **The Database Administrator archetype** — fronts `backend/database_administrator` through an MCP client; the first business system integrated under the one-MCP-server-per-system pattern ([ADR 0009 § D4–D5](docs/adr/0009-redefine-cachicamas-as-a-multiplayer-agentic-system.md)). Milestone doc to be written when planning starts.
4. **Further archetypes** — tickets, finance, marketing — each owning its business system's MCP server, each an additive change on the unchanged runtime, each with its own milestone doc when planned.

In parallel: [PRD 0001](docs/prd/0001-cachicamas-delivery-loop.md) (the delivery loop) is re-scoped under the new identity before it generates milestone doc 0005 ([ADR 0009 § D7](docs/adr/0009-redefine-cachicamas-as-a-multiplayer-agentic-system.md)).

## 12. The Agent-First Doc Pattern

The pattern this README follows is **portable**. Three passes, top to bottom, so an agent (or a hurried human) can find what it needs at the level of detail it needs.

### The three passes

| Pass | What | Purpose | Cost to read |
| ------ | ------ | --------- | -------------- |
| **Pass 1: Resumed TOC** | A small table with one row per topic and a one-line answer. | Scan everything at a glance. Decide whether to keep reading. | One screen. |
| **Pass 2: Full TOC** | The complete structured index with anchors. | Jump straight to a specific section. | One screen. |
| **Pass 3: Content** | The detailed sections, each self-contained. | Read the depth you actually need. | Variable. |

### Adapting to any document

| Doc type | Pass 1 (Resumed TOC) | Pass 2 (TOC) | Pass 3 (Content) |
| ---------- | ---------------------- | -------------- | ------------------ |
| README | Topic table with one-line answers | Anchor-linked headings | Detailed sections |
| Architecture doc | Decision summary table | Section index | Trade-off analysis |
| Onboarding guide | What you'll learn table | Step-by-step index | Step instructions |
| ADR | Status table (Proposed/Accepted/Superseded) | ADR index | Full ADR |
| PR description | Impact summary | Change list | Detailed diff context |
| API reference | Resource table | Endpoint index | Endpoint reference |

### Adapting to code (slight modification)

For code, the same three passes apply — the "resumed TOC" becomes a **module/file summary**, the "full TOC" becomes a **symbol index**, and "content" is the **implementation**.

| Pass | In a Go file | In a package | In a service |
| ------ | -------------- | -------------- | -------------- |
| **Pass 1: Resumed view** | One-line `// Package ...` doc comment above the package + a 3–5 row table at the top of `main.go` listing each top-level symbol with its purpose | Package-level `doc.go` with a symbol summary table | Service `README.md` (this very pattern) + `application/services.go` index |
| **Pass 2: Symbol index** | Comment header listing exported symbols (`// Exported: NewX, DoY, ZType`) | `package_symbols.go` (or generated) listing every symbol with a one-line summary | Service `Makefile` targets + `cmd/server/main.go` wiring |
| **Pass 3: Implementation** | The actual functions and types | The actual `.go` files | The actual hex layout |

**Rule of thumb:** if an agent can answer 80% of "what is this?" by reading pass 1 alone, the doc is well-shaped. Pass 2 exists so it doesn't have to grep. Pass 3 is the detail it reads only when pass 1 + 2 say "yes, this is what I need."

## 13. Review Checklist

A reviewer can confirm each item below without re-reading the whole README. This is the self-grade.

- [ ] reviewer can confirm the logo renders at `docs/assets/cachicamas-logo.png` and its alt text names the multiplayer agentic system
- [ ] reviewer can confirm cachicamas is identified as a multiplayer agentic system for building and running a company, citing [ADR 0009](docs/adr/0009-redefine-cachicamas-as-a-multiplayer-agentic-system.md)
- [ ] reviewer can confirm Layer 3 is named the archetype layer, and the coding archetype is stated as its first occupant, not its definition
- [ ] reviewer can confirm the layer statuses match the milestone docs — Layer 1 complete 42/42 (doc 0002), Layer 2 in progress 14/24 (doc 0003), Layer 3 not started 0/25 (doc 0004)
- [ ] reviewer can confirm the MCP pattern is stated (one MCP server + one owning archetype per business system) and does not contradict [ADR 0005 § D1](docs/adr/0005-promote-agent-stack-to-own-module.md#d1--dependency-rule-v2) (network only, never import)
- [ ] reviewer can confirm SDD is described as the engineering process, explicitly not the product
- [ ] reviewer can confirm no references to the removed wiki directory or to the archived prd-orchestrator change remain anywhere in this README
- [ ] reviewer can confirm the tech stack table matches `openspec/project.md`
- [ ] reviewer can confirm the hexagonal layout matches `backend/database_administrator/src/`
- [ ] reviewer can confirm "no Co-Authored-By" is stated and the rule is honored
- [ ] reviewer can confirm Strict TDD is documented as enabled
- [ ] reviewer can confirm the Resumed TOC fits on one screen and answers "what is this repo?" in one line
- [ ] reviewer can confirm the Agent-First pattern section (12) is reusable on docs and adaptable to code
- [ ] reviewer can confirm the Quick Start commands match the actual Makefile targets

## 14. Next Step

1. Read [`openspec/project.md`](openspec/project.md) for the bootstrap artifact (stack, conventions, testing discipline).
2. Read [the v2 architecture reference](docs/architecture/0001-cachicamas-agent-stack-v2.md) for the agent stack, then [ADR 0009](docs/adr/0009-redefine-cachicamas-as-a-multiplayer-agentic-system.md) for the identity and vocabulary that amend it.
3. Read the milestone docs for the current frontier: [doc 0002](docs/architecture/milestones/0002-cachicamas-ai-layer-1-task-graph.md) (Layer 1, complete), [doc 0003](docs/architecture/milestones/0003-cachicamas-agent-layer-2-task-graph.md) (Layer 2, active), [doc 0004](docs/architecture/milestones/0004-cachicamas-coding-layer-3-task-graph.md) (the coding archetype, next).
4. Pick an active change under `openspec/changes/` and run its next phase (`/sdd-continue` or `/sdd-ff`), or start a new change with `/sdd-new <name>`.

If you are an agent: before changing code, search engram (`mem_search` with project `cachicamas`) for prior context on the topic. Save every decision, bug fix, and convention via `mem_save` — the next session depends on it.
