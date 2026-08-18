# `backend/agent`

The cachicamas agent stack: a self-contained Go module that owns the model
adapter, the portable agent runtime, and the archetype layer — whose first
occupant is the coding archetype. It is a **library plus a CLI**, not a service —
it serves no HTTP traffic, owns no database, and is deployed by being run, not by
being containerised.

This module is deliberately isolated. It imports nothing from
`backend/database_administrator` or `backend/workspace_syncer`, in any package,
including `cmd/`. Because these are separate Go modules, a cycle is a build error,
so the compiler enforces half of that rule for free; the guards described below
cover the other half.

## Contents — three layers plus a composition root

Dependencies flow in exactly one direction, and only downward:

```
src/cmd/cachicamas   this archetype's composition root (package main) — the ONLY
        |            place policy meets mechanism, and the only place allowed to
        |            install the OTel SDK
        v
src/coding           Layer 3, the archetype layer — occupied by the coding
        |            archetype, the FIRST archetype on this stack: slash commands, session
        |            persistence, skills, prompts, tools, permission policy
        v
src/agent            Layer 2 — the portable agent runtime: the stateless loop and
        |            the stateful harness, connected by an event stream
        v
src/ai               Layer 1 — the model adapter: provider-neutral request and
                     stream contracts, plus one adapter per vendor
```

**`src/coding` is one archetype, not the whole of Layer 3.** Layers 1 and 2 are
each the whole of their layer; Layer 3 is a position, and the coding archetype is
the first occupant to stand on it. A second archetype would add a sibling
`src/<archetype>/` with its own `src/cmd/<archetype>/` root, attach to the *same*
`src/agent`, and change nothing below it — which is what "portable" in Layer 2's
name is claiming.

Future occupants of Layer 3 are archetypes — implementations of one specialist
agent each — that own business systems over MCP
([ADR 0009](../../docs/adr/0009-redefine-cachicamas-as-a-multiplayer-agentic-system.md)).
The first planned example is the Database Administrator archetype, which will
front `backend/database_administrator` through an MCP client.

**Only `src/ai` exists today.** Layers 2 and 3 and the composition root are planned
in [doc 0003](../../docs/architecture/milestones/0003-cachicamas-agent-layer-2-task-graph.md)
and [doc 0004](../../docs/architecture/milestones/0004-cachicamas-coding-layer-3-task-graph.md);
their directories are **deliberately absent** rather than empty. An empty
`src/agent/` would make the forward guard's forbidden-prefix list untestable — a
prefix that also matches a real package of this module can no longer be shown to be
forbidden.

`src/agenttest/` is the external-consumer proof: a package outside `src/ai` that
imports it, so anything `src/ai` fails to export is caught here rather than by
Layer 2 months later. **It must remain a direct sibling of `src/ai/`.** A later
milestone's signature guard resolves `../ai/provider.go` from `runtime.Caller(0)`;
any other layout breaks it silently. See ADR 0005 § D2 and Guard C.

## The dependency rule

Governed by [ADR 0005 § D1](../../docs/adr/0005-promote-agent-stack-to-own-module.md#d1--dependency-rule-v2)
and [§ D3](../../docs/adr/0005-promote-agent-stack-to-own-module.md#d3--observability-boundary).

| Package | May import | Must not import |
| --- | --- | --- |
| `src/ai` (Layer 1) | Go stdlib; `net/http`; vendor model SDKs *(ADR-permitted in principle; **none admitted today** — AI-24 chose raw `net/http`, and the deny-by-default allowlist admits no SDK until one is deliberately added)*; OpenTelemetry **API** | the sibling layers; any package of another backend module; OTel **SDK**, exporters, `otelslog` |
| `src/agent` (Layer 2) | Layer 1; Go stdlib; OTel API | Layer 3; `cmd`; another module; OTel SDK; the environment; the filesystem; `net/http` |
| `src/coding` (Layer 3) | Layers 1–2; Go stdlib; OTel API; `net/http`; third-party TUI/TOML/tokenizer deps | `cmd`; **any** Go package of another backend module — it may call them over HTTP, never import them; OTel SDK |
| `src/cmd/cachicamas` | everything above, plus the OTel SDK and exporters | — it is `package main`, so nothing *can* import it |

On OpenTelemetry the rule is the standard library-instrumentation split: **API yes,
SDK no.** The API modules are dependency-light and are no-ops until a provider is
registered, so an unwired Layer 1 costs nothing at runtime. The SDK and its
exporters are process-lifecycle concerns and belong to a composition root.

**The module's dependency set is exactly the OpenTelemetry API** — 2 direct
`require`s (`go.opentelemetry.io/otel`, `go.opentelemetry.io/otel/trace`) plus 1
indirect (`github.com/cespare/xxhash/v2`), added by AI-37 under ADR 0005 § D3.
The two milestones this paragraph once said *may* add a dependency have both
resolved: AI-24 selected raw `net/http` (standard library, zero requires) and
AI-37 added the OTel API. That the set grows no further is a checked property,
not an accident: the forward guard is deny-by-default AND pins the exact
require-set and package closure by set equality
(`TestLayer1_DependencySet_ExactRequiresAndClosure`), so a dependency nobody
thought to forbid by name still fails.

## Layout

```
backend/agent/
├── go.mod              2 requires (OTel API) + 1 indirect; go.sum committed
├── Makefile            copied from database_administrator; see the note below
├── .golangci.yml       byte-identical to database_administrator's
├── README.md
├── .gitignore
└── src/
    ├── ai/             Layer 1
    └── agenttest/      external-consumer proof — direct sibling of ai/, always
```

A repo-root `go.work` lists all three backend modules, for editor and future-CI
ergonomics. No module gains a `replace` directive: nothing imports this module yet,
and a `replace` with no requirer is dead weight that also disguises a real build
cost (ADR 0005 § Migration).

## Running the checks

**There is no CI.** `.github/workflows/` does not exist, so every guard in this
module runs only when a human runs `make test` **in this directory**. Three modules
now means three places to run it, and "recorded green" means output pasted into a
pull request, not a check mark from a runner. This is recorded as a known
consequence in ADR 0005 § Enforcement.

```bash
cd backend/agent
make test    # go test -race -v ./...   <- the evidence gate for every milestone
make lint    # go vet, then golangci-lint v2.9.0
make fmt
make build
```

`make test` and `make lint` are load-bearing: docs 0002, 0003 and 0004 all close
their leaves on recorded green output from them. Their recipes are pinned in the
`Makefile` so that a later divergence from `database_administrator`'s copy is
visible rather than silent.

The build config is a **copy** of `database_administrator`'s rather than something
shared. That is deliberate and ADR 0005 records the drift risk it creates: a shared
config would be a fourth thing to own and would couple two modules that must not be
coupled.

> **This module is not hexagonal.** `database_administrator` and `workspace_syncer`
> are; this one is *layered*. Applying `domain`/`application`/`infrastructure`/`interfaces`
> rules here produces nonsense. See `openspec/AGENTS.md` § 2.

## Where the plans live

- [Architecture reference (v2)](../../docs/architecture/0001-cachicamas-agent-stack-v2.md) — the seams, the gaps, the review checklist.
- [Layer 1 task graph](../../docs/architecture/milestones/0002-cachicamas-ai-layer-1-task-graph.md) — AI-00 … AI-40.
- [ADR 0004](../../docs/adr/0004-adopt-tau-3-layer-agentic-architecture.md) · [ADR 0005](../../docs/adr/0005-promote-agent-stack-to-own-module.md) · [ADR 0006](../../docs/adr/0006-resolve-skill-and-prompt-source-of-truth.md)
