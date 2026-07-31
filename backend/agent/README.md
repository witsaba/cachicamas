# `backend/agent`

The cachicamas agent stack: a self-contained Go module that turns a language model
into a coding agent. It owns the model adapter, the portable agent brain, and the
coding application that drives them. It is a **library plus a CLI**, not a service —
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
src/cmd/cachicamas   composition root (package main) — the ONLY place policy meets
        |            mechanism, and the only place allowed to install the OTel SDK
        v
src/coding           Layer 3 — the coding application: slash commands, session
        |            persistence, skills, prompts, tools, permission policy
        v
src/agent            Layer 2 — the portable brain: the stateless loop and the
        |            stateful harness, connected by an event stream
        v
src/ai               Layer 1 — the model adapter: provider-neutral request and
                     stream contracts, plus one adapter per vendor
```

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
| `src/ai` (Layer 1) | Go stdlib; `net/http`; vendor model SDKs; OpenTelemetry **API** | the sibling layers; any package of another backend module; OTel **SDK**, exporters, `otelslog` |
| `src/agent` (Layer 2) | Layer 1; Go stdlib; OTel API | Layer 3; `cmd`; another module; OTel SDK; the environment; the filesystem; `net/http` |
| `src/coding` (Layer 3) | Layers 1–2; Go stdlib; OTel API; `net/http`; third-party TUI/TOML/tokenizer deps | `cmd`; **any** Go package of another backend module — it may call them over HTTP, never import them; OTel SDK |
| `src/cmd/cachicamas` | everything above, plus the OTel SDK and exporters | — it is `package main`, so nothing *can* import it |

On OpenTelemetry the rule is the standard library-instrumentation split: **API yes,
SDK no.** The API modules are dependency-light and are no-ops until a provider is
registered, so an unwired Layer 1 costs nothing at runtime. The SDK and its
exporters are process-lifecycle concerns and belong to a composition root.

**The module currently has zero dependencies**, and `go.mod` carries no `require`
at all. That is a checked property, not an accident: the forward guard is
deny-by-default, so a dependency that nobody thought to forbid by name still fails.
Two milestones may change it — one selecting a transport, one adding the OTel API —
and each needs its own ADR.

## Layout

```
backend/agent/
├── go.mod              zero requires; no go.sum
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
