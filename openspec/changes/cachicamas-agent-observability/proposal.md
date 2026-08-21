# Proposal: AG-22 — Add the observability boundary

Milestone AG-22 (`docs/architecture/milestones/0003-cachicamas-agent-layer-2-task-graph.md:2043-2094`) · Closes R-19 · Governed by ADR 0005 § D3 (`docs/adr/0005-promote-agent-stack-to-own-module.md:230-262`) · Depends on AG-21 (merged, PR #183, `7dac9ec9`).

## Intent

Layer 2 runs, turns, tool executions and compactions are observable today only through the in-process event stream. Nothing is exportable, so an operator cannot see a run's shape outside the process. AG-22 closes R-19 by emitting spans through the **OTel API only**.

**Why the vocabulary lands first.** § D3's attribute allowlist is explicitly scoped — "**Attribute allowlist** for Layer 1 spans" (`0005:246`) — a twelve-key list ending in `error.type` (`:246-250`). Layer 2 has **no** decided allowlist. Instrumenting before deciding one would mint keys ad hoc, which is exactly what the milestone forbids: attributes must be "recorded as an extension of ADR 0005 § D3's table … not ad hoc" (`0003:2067`). AG-22.1 is therefore a `[decision]` node gating AG-22.2 (`0003:2093`).

## Scope

### In scope

1. **AG-22.1 — the Layer 2 attribute vocabulary**, recorded as an **extension of § D3's table** (appended to ADR 0005, not a sibling ADR, not a replacement — the milestone's own wording). It must carry: span names per lifecycle (run / turn / tool-execution / compaction); attribute keys per span kind, following the OTel GenAI semantic conventions where they exist and inventing nothing gratuitous where they do not; the § D3 denylist restated, not weakened; and the named "deliberately not recorded" list with reasons.
2. **AG-22.2 — the instrumentation**, at seams verified in this worktree: `Harness.Run` (`harness.go:408`), `Turn` (`loop.go:261`), `Scheduler.executeCall` (`scheduler.go:401`) with `runToolWithWindDown` (`:594`), `runCompaction` (`compaction.go:258`) and `Harness.Compact` (`:413`). Spans start and end at the same call sites the corresponding Start/End events are already constructed, never by duplicating control flow.
3. **The AG-03.2 guard amendment.** `import_boundary_test.go`'s `allowedProductionPrefixes` (`:127-130`) today holds exactly two entries and **zero** OTel entries. The first production OTel import fails that guard. The amendment lands in the **same commit** as that import, each entry annotated with its § D3 clause per `S-AGP-024`.
4. **A new Layer-2 observability capability spec**, modelled on Layer 1's `openspec/specs/ai-observability-boundary/spec.md`.

### Explicitly out of scope

| Excluded | Reason |
|---|---|
| Exporters, the OTel **SDK**, dashboards, metrics | Composition-root concerns (`0003:2053`); § D3 denies `otel/sdk`, `otel/exporters`, `otelslog` below `cmd/` (`0005:242-243`), and `import_boundary_test.go:118-120` already enforces it |
| Per-delta events (text/reasoning deltas, tool progress) | Per-delta payloads are raw model/tool output — denylisted **and** unbounded in cardinality |
| Hook timings | Named out of scope by AG-22.1 item 2; a concrete telemetry hook is Layer 3's (`agent-hook-taxonomy/spec.md:328`) |
| Permission argument content | Named out of scope by AG-22.1 item 2; permission arguments are tool-argument text — denylisted |
| A new `go.mod` require | `backend/agent/go.mod:5-8` already requires `go.opentelemetry.io/otel v1.44.0` and `.../otel/trace v1.44.0`. **Design must still measure the real package closure** at the version `go.work` resolves before relying on any allowlist entry — an ADR authorises a path, not its closure |

Each excluded item must be named **with its reason** inside the vocabulary decision itself, not only here.

### The denylist — absolute, restated verbatim, never weakened

> **Attribute denylist, absolute:** any prompt, completion, reasoning, tool-argument or tool-result text; any HTTP header; any credential; any raw provider response body. (`0005:252-255`)

This applies to Layer 2 spans without exception or carve-out. Concretely it forbids `ToolStart.Arguments()`, `ToolEndSuccess.Result()`, `ToolEndResultFailure.Result()` and any `Failure` projection beyond its closed category vocabulary from ever reaching an attribute.

## Two open questions this change must decide (design owns the mechanism)

1. **Detached tool-call span lifecycle.** `runToolWithWindDown` (`scheduler.go:594`) can report `detached=true` when the wind-down bound expires with the tool's goroutine still running. AG-21 hardened this package against goroutine leaks; a span never ended on that path is a leak of the same shape. This path **must get an explicit, decided outcome** — every tool-execution span ends exactly once on every exit including the detached one — not an implicit one.
2. **Delegation-tree nesting proof surface.** No production delegating tool ships; AG-19 kept every subagent concept in test fixtures. "Delegation trees preserved" is therefore provable **only** through the existing fixtures (`nested_run_test.go`, `siblings_test.go`, `delegating_tool_test.go`). The proposal states this honestly rather than implying a production surface exists; the fixtures may need extending.

## Approach

- Mirror AI-37's Layer 1 shape: a constants file of literal attribute keys, presence-typed optional attributes, one finalizer per span family, and failure recorded only as a closed category name — never `err.Error()`, never a response body.
- Thread spans along the **existing** `ctx` chain (`Harness.Run` → `Turn` → `Schedule` → `executeCall` → `tool.Run`). No `context.WithoutCancel` and no ctx re-rooting exists in `src/agent`, so standard `trace.ContextWithSpan` propagation gives child-harness nesting without bespoke plumbing.
- Reuse the test substrate verbatim: `src/agenttest/tracetest` (hand-rolled API-level recorder — the OTel **SDK is structurally forbidden even in `_test.go`** because the guards scan `go list -deps -test`, so this is the only admissible double; **this fork is settled, not open**) and `src/agenttest/sweep` (`Entry`/`Scan`/`SelfTest`) for the absence proof, with `SelfTest` as the mandatory positive control before any clean scan.

## Capabilities

### New capabilities

- `agent-observability-boundary` — Layer 2's span vocabulary, the four instrumented lifecycles, the absolute denylist and the no-tracer parity guarantee. Proposed requirement prefix **`AGO`** (verified free: zero `R-AGO-`/`S-AGO-` hits across `openspec/` including `changes/` and `changes/archive/`; `AOB` belongs to Layer 1 and must not be reused).

### Modified capabilities

- `agent-package-scaffold` — `R-AGP-003` already permits "such observability API paths as ADR 0005 § D3 authorises for Layer 2 **and** design explicitly selects" (`spec.md:70`), and `S-AGP-024` (`:84`) requires every entry to carry its ADR clause. This is a requirement AG-22 **satisfies**, not a contradiction; the delta adds scenarios pinning the new entries and re-proving the guard **bites** on a scratch SDK import.

### Checked and found NOT falsified (confirm at spec time, do not assume)

| Spec | Line | Text | Verdict |
|---|---|---|---|
| `agent-module-scaffold` | `:124-128`, `:135` | Third-party group admits OTel **API** paths with § D3 annotation; `S-AGM-043` requires no API path in the forbidden list | Layer 1's guard, unchanged by AG-22; **no go.mod amendment** |
| `agent-v1-scope` | `:343` (`S-AGS-066`) | "…declares **no** concrete hook, no cache-breakpoint placement, no compaction policy and **no telemetry sink**" | AG-22 ships **no sink** (no provider registration, no exporter) and no hook — but this is an absolute quantifier over the package surface and MUST be re-read at spec time |
| `agent-hook-taxonomy` | `:328` | "Any concrete hook — cache breakpoints, compaction policy, telemetry → **Layer 3**" | AG-22 instruments the runtime directly; it ships no telemetry **hook** |
| `agent-concurrency-hardening` | `:181` | "Telemetry or spans over the hardened assembly → **AG-22**" | Pre-planned handoff; no contradiction |

## Acceptance criteria (verbatim from AG-22.2, `0003:2075-2091`)

```gherkin
Scenario: spans nest correctly with vocabulary attributes only
  Given an in-memory tracer over a run using tools, delegation and compaction
  When the spans are inspected
  Then run, turn, tool-execution and compaction spans nest correctly with delegation trees preserved
  And every attribute is drawn from the decided vocabulary with values equal to the corresponding events'

Scenario: the denylist is proven by absence
  Given a full-featured run that used tools, reasoning, permission and compaction
  When all recorded telemetry is scanned
  Then no prompt, completion, reasoning, tool argument or result text, header, or credential appears anywhere

Scenario: no tracer, no difference
  Given the same run with no tracer configured
  When its event sequence is compared to the traced run's
  Then behavior is identical and nothing panics
```

Plus AG-22.1's closing checklist (`0003:2066-2068`): the four span families' names and attributes decided and recorded as a § D3 extension with the denylist restated; and the not-recorded list stated with reasons.

## Affected areas

| Area | Impact | Description |
|---|---|---|
| `docs/adr/0005-promote-agent-stack-to-own-module.md` | Modified | § D3 extension: Layer 2 attribute allowlist + not-recorded list |
| `backend/agent/src/agent/` (new trace file) | New | Attribute-key constants, span helpers, one finalizer per family |
| `harness.go`, `loop.go`, `scheduler.go`, `compaction.go` | Modified | Span start/end co-located with existing event constructors |
| `backend/agent/src/agent/import_boundary_test.go` | Modified | `allowedProductionPrefixes` gains its § D3-annotated OTel entries |
| `backend/agent/src/agent/*_test.go` (new) | New | Nesting proof, absence proof, no-tracer parity proof |
| `backend/agent/src/agenttest/tracetest`, `.../sweep` | Reused | Extension only if a needed capability is missing |
| `docs/architecture/milestones/0003-…` | Modified | AG-22 back-annotation |

## Risks

| Risk | Likelihood | Mitigation |
|---|---|---|
| Detached tool-call span leak (open question 1) | Med | Design decides the detached exit explicitly; `tracetest.Provider.AssertAllEndedOnce()` proves every span ended exactly once |
| AG-03.2 guard fails on first OTel import | High (expected) | Guard amendment ships in the same commit, never as a follow-up |
| Absence assertion passes vacuously | Med | `sweep.SelfTest` positive control, non-empty-corpus and event-coverage guards, plus a deliberate defeat test that must FAIL before it is reverted |
| Allowlist entry admitted beyond its measured closure | Med | Fresh `go list -deps` measurement before/after, diffed; no speculative grant |
| Delegation nesting has no production surface | High (accepted) | Proven through AG-19's existing fixtures; stated as such, never implied otherwise |
| Size forecast exceeds budget | Med | See below; slice at the AG-22.1/AG-22.2 boundary if it does |

## Size forecast (non-`openspec/` lines only)

| Area | Est. lines |
|---|---|
| ADR § D3 extension + doc 0003 back-annotation | 50–80 |
| Production Go (constants, helpers, 4 seams, tracer plumbing) | 220–360 |
| Guard amendment | 10–20 |
| Tests (nesting, absence ≈ AI-37's 282-line precedent, no-tracer parity) | 450–650 |
| **Total** | **≈ 730–1110** |

`400-line budget risk: High` against the default; **Medium** against AG-22's granted 1000. The forecast's upper half exceeds 1000, so a budget extension or an AG-22.1 / AG-22.2 slice may be needed — `sdd-tasks` must re-forecast with real file plans before apply.

## Rollback plan

Single-PR revert of the merge commit. The change is purely additive: no `go.mod` entry, no exported-surface removal, no event-stream change. Reverting restores `allowedProductionPrefixes` to its two-entry form and removes every OTel import from `src/agent` production code; ADR 0005 § D3's Layer 1 allowlist and Layer 1's own instrumentation are untouched on either side of the revert.

## Dependencies

- AG-21 — merged (PR #183, `7dac9ec9`).
- `go.opentelemetry.io/otel v1.44.0` and `.../otel/trace v1.44.0` — already present in `backend/agent/go.mod:5-8`; closure to be measured, not assumed.

## Success criteria

- [ ] The Layer 2 attribute vocabulary is recorded as an extension of ADR 0005 § D3's table, with the denylist restated verbatim and unweakened.
- [ ] Per-delta events, hook timings and permission argument content are each named as not recorded, each with its reason.
- [ ] Run, turn, tool-execution and compaction spans nest correctly with delegation trees preserved, attributes drawn only from the decided vocabulary and equal to the corresponding events' values.
- [ ] A full-featured run's telemetry is scanned and contains no denylisted vector — with the scan's own needles proven to bite first.
- [ ] Every span ends exactly once on every exit path, including the detached tool call.
- [ ] With no tracer configured, the event sequence is byte-identical to the traced run's and nothing panics.
- [ ] The AG-03.2 guard passes with § D3-annotated OTel entries and is re-proven to bite on a scratch SDK import.
- [ ] `backend/agent/go.mod` gains no `require` entry.
