# Exploration — `cachicamas-agent-history` (AG-12: history and the pairing invariant)

> Phase: `sdd-explore` · Change: `cachicamas-agent-history` · Milestone: **AG-12** in
> [doc 0003](../../../docs/architecture/milestones/0003-cachicamas-agent-layer-2-task-graph.md#ag-12--implement-history-and-the-pairing-invariant)
> Engram topic: `sdd/cachicamas-agent-history/explore`

## Charter (verbatim source: `docs/architecture/milestones/0003-cachicamas-agent-layer-2-task-graph.md:1227-1292`)

- **Goal:** the harness's transcript store — append-only within a run, validated at the boundary, with orphan synthesis for interruption.
- **Deliverable:** History with the pairing invariant; orphan-synthesis behavior; read access for the loop and (eventually) Layer 3 persistence.
- **Acceptance:** a transcript that would orphan a call cannot be committed; interruption synthesizes results for orphaned calls before the next turn; the enforcement lives at the boundary, not at call sites (v2 § 4.2's exact phrasing, as architecture).
- **Depends on:** AG-03, AI-40 (frozen Layer 1 message contracts). **Blocks:** AG-13, AG-17.
- **Out of scope:** persistence (Layer 3); compaction's interaction (AG-18.2 re-proves the invariant post-compaction).
- Two leaves: **AG-12.1** (append + boundary validation, `[leaf]`) and **AG-12.2** (orphan synthesis, `[leaf]`, depends on AG-12.1).

Vocabulary already frozen for this milestone (`openspec/changes/archive/2026-08-11-cachicamas-agent-contract-vocabulary/decision.md`):

- `VL2-HAR-01` **history** — "The harness's append-only-within-a-run, boundary-validated transcript store. It rejects any commit that would orphan a tool call … through exactly one commit path, with no privileged bypass for internal callers. Seeded construction … validates by the same rule."
- `VL2-COR-10` **transcript** — the harness's ordered history of messages across a run (distinct from a Layer 1 `ai.Message`).
- `VL2-COR-11` **pairing invariant** — every tool call has a matching result, enforced at the history boundary, not patched at each call site.

"The C1 lesson applied to history" traces to `docs/architecture/0001-cachicamas-agent-stack-v2.md:376` — defect C1: the exported text value type satisfied the part interface directly, so its zero value was a valid part that passed message validation and bypassed every construction rule. Fixed by AI-06 with unexported fields plus one package-owned validating constructor and no back door. That is the precedent AG-12 must reapply for `History`.

## Current state — package layout and conventions (`backend/agent/src/agent/`)

| File | Shape that matters to AG-12 |
| --- | --- |
| `event.go` | `Event` — opaque struct, unexported fields, read-only accessors, `CheckEmit` boundary validator using `ai.FirstFailure`/`ai.Invalid`. The "one door in" shape. |
| `loop.go` | `Turn(...)` — the stateless loop skeleton; on `FinishReasonToolCalls` calls `sched.Schedule(...)`. AG-12 does **not** wire into it (that is AG-13's edge). |
| `scheduler.go` | `Scheduler.Schedule(...)` dispatches `ai.ToolCall`s and produces `[]Result`. `Result.callID` is unexported, set only through the scheduler's own reconstruction from `ai.ToolCall.ID()`. |
| `failure.go` | `Failure` — thin wrap of `*ai.Failure`, constructed only via `NewFailure`, `nil` rejected. Wraps upstream provider/runtime failures, **not** caller-contract violations. |
| `doc.go` + `doc_contract_guard_test.go` | Machine-checked layer contract, rows `L2C-01`…`L2C-06`, each landed in the same PR as its `expectedLayer2ContractRows` entry (closed-amendment rule `R-AGP-002`). |

No `history`/`transcript` file exists in `backend/agent/src/agent/` yet — this is greenfield.

## Frozen Layer 1 contracts AG-12 stands on (`backend/agent/src/ai/`)

- `ToolCall` (`tool_call.go`) — opaque `{id, name, arguments}`; `ID()`, `Name()`, `Arguments()`; `NewToolCall` validates id/name non-empty and arguments well-formed JSON.
- `ToolResult` (`tool_result.go`) — opaque `{callID, content, failed}`; `NewToolResult` / `NewToolFailure` funnel through one `newToolResult` validator (single rule: `callID != ""`). `CallID()` is the identity paired against `ToolCall.ID()` by plain `==`.
- `Message` (`message.go`) — opaque struct with `id MessageID`, `role`, `content []Part`, `cacheBoundary`; `Content()` returns a cloned slice (no aliasing).
- `ai.ToolResult`'s own doc comment defers the transcript-level invariant explicitly: "the invariant that every call in a transcript has a matching result is V-OUT-02 … Layer 2's job." AG-12 is the first and correct owner.
- `backend/agent/src/handoff/` does not exist in this repo — not applicable.

## Typed-error idiom already established

Two coexisting typed-failure surfaces in `agent`:

1. **Caller-contract violations** (construction/validation time) — reuse Layer 1's `ai.Violation` / `ai.Invalid(rule, at...)` / `ai.At(name)` / `ai.FirstFailure(...)`. Already called directly from `agent` package code (`agent/run_events.go`'s `NewRunEnd` uses `ai.Invalid(ai.ErrNotInVocabulary, …)`, `ai.Invalid(ai.ErrEmpty, …)`, `ai.Invalid(ai.ErrMisplaced, …)`). This is the natural vocabulary for both of AG-12.1's typed rejections.
2. **Runtime/provider failures** — `agent.Failure`. Not applicable to history's pairing rejections.

`ai`'s rule-class vocabulary (`ai/validation.go:58-124`) is documented as closed but extensible "by appending a class in the pull request that needs it, never by a milestone defining a sentinel of its own." Closest existing fits: `ErrUnresolvedReference` covers the orphaned-result case ("a value naming something the request does not declare") well; **no class cleanly covers "a call has no result once the turn closes."**

## The seam AG-12 must create

1. A `History` type holding an ordered sequence of Layer 1 values across a run, exposing read-only views with stable entry identity.
2. **Exactly one commit path** — every public route that could append or mutate routes through one internal validating function; no privileged bypass for internal callers (independently testable: an orphaning sequence constructed through *any* public route is rejected).
3. A **seeded constructor** that re-runs the same validation over a pre-existing transcript. Named in the milestone as "the seam session resume and next-run model switching stand on, frozen at the handoff" — its signature matters beyond AG-12.
4. **Orphan synthesis** — given a transcript with N orphaned calls, synthesize a result for each, typed distinguishably as an interruption artifact; idempotent and total (closes exactly N on the first pass, changes nothing on the second).

## Candidate approaches

### 1. Opaque `History` struct, unexported storage, one private validating commit path *(recommended)*

- **Pros:** reuses the idiom this codebase already trusts four times (`ai.Part`, `agent.Event`, `agent.Result`, `agent.Failure`); exactly one unexported mutation point to audit; matches `VL2-HAR-01`'s "exactly one commit path" wording almost verbatim.
- **Cons:** none significant.
- **Effort:** Low–Medium.

### 2. `History` as an interface with a single implementation

- **Pros:** leaves an extension seam for AG-13 / Layer 3.
- **Cons:** persistence is explicitly out of scope; no concrete boundary type in `agent/` from AG-04 to AG-11 is an interface until a second implementation actually existed. Speculative generality.
- **Effort:** Medium.

### 3. Orphan synthesis as a method vs. a free function

A method (`h.SynthesizeOrphans()`) that itself routes through the same internal append primitive keeps the mutation path singular, so the "any public route" scenario covers it for free. A free function would need its own route back through the validating path or become a second unaudited bypass.

### 4. Where the typed pairing rejection lives

| Option | Assessment |
| --- | --- |
| Reuse existing `ai` machinery and classes | Lowest risk; matches the `NewRunEnd` precedent. Fits the orphaned-result case via `ErrUnresolvedReference`. |
| Add a new `ai` rule class | Technically permitted by the closed-set doc comment, but touches a Layer 1 file this milestone treats as frozen input, adding Layer 1 review surface to a Layer 2 milestone. |
| `agent`-local typed violation | Keeps the change fully Layer 2-scoped, consistent with `VL2-COR-11` naming the pairing invariant a Layer 2 concept; cost is a second small typed-error shape. |

A genuine open decision for `sdd-design`; the "frozen Layer 1 contracts" framing leans against touching `ai/validation.go`.

## Test conventions observed

- Table-driven tests with `t.Run(tt.name, …)`.
- `backend/agent/src/agenttest/` holds cross-package, vendor-neutral kit code. No history helper exists today; AG-12 can test `History` from an external test package, which the "through any public route" scenario in fact requires.
- Strict-TDD RED evidence convention (per AG-04's `apply-progress.md`): each RED captured with the exact failing test name, file, and terminal output before the GREEN commit.
- Doc-contract guard tests are bidirectional (`doc.go` rows pinned against `expectedLayer2ContractRows`), plus a scratch-edit RED proving the guard fires.

## Doc-contract and spec-ID machinery

- `doc.go` rows `L2C-01` imports · `L2C-02` no I/O · `L2C-03` event-stream-only · `L2C-04` stream-membership criterion (AG-04) · `L2C-05` message/tool reconstructability (AG-05) · `L2C-06` protocol-family constructibility (AG-06). AG-12 introduces a comparably new package-wide guarantee — an `L2C-07` row is a plausible but undecided candidate.
- `openspec/specs/` holds one `spec.md` per capability with a reserved ID prefix. `AEV`, `AGE`, `AGP`, `ATT`, `TLS`, `APE`, `PRH`, `LSK`, `AMT` are taken. AG-12 must mint an uncollided prefix; `HIS` or `HAR` are natural given `VL2-HAR-01`.

## Open questions for `sdd-propose` / `sdd-design`

1. Does AG-12 add an `L2C-07` doc-contract row, following the AG-04 precedent, or express the guarantee only in the spec (as AG-05/AG-06 did)?
2. Does the "call has no result once the turn closes" rejection reuse an existing `ai` rule class, add a new one, or become an `agent`-local typed violation?
3. What is the exact seeded-construction signature, given it is frozen at the handoff for AG-13 and Layer 3 session resume?
4. How does a synthesized result stay "distinguishable from a real result" when `ai.ToolResult` has no interruption-marker field and is frozen?
5. Does `History` wire into `Turn`/`Schedule` in this milestone? The charter's placement ("beside wave 2", only in-document edge is the scaffold) says no — that wiring is AG-13's.

## Risks

- Touching `ai/validation.go` for a new rule class is architecturally sensitive given this milestone's own "frozen Layer 1 contracts" framing.
- The interruption-artifact distinguishability requirement (AG-12.2) may force a Layer 2 wrapper, in tension with AG-12.1's "history exposes … Layer 1 values". These two scenarios must be reconciled explicitly in design, not implicitly.
- Spec-ID prefix collision — a documented prior pitfall in this repo's parallel-wave SDD work.
- The seeded-construction signature and read-only view shape are consumed by AG-13; drift there ripples forward without a redesign window.

## Recommendation

Approach 1 — opaque `History` struct, unexported storage, one private validating commit path, `SynthesizeOrphans` as a method routed through the same path. Defer the `ai` rule-class question and the `L2C-07` doc-row question to `sdd-design` as two named open decisions.

**Ready for proposal:** yes.
