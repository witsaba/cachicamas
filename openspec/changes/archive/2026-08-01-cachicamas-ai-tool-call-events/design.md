# Design — streamed tool-call events

> **Change**: `cachicamas-ai-tool-call-events` · **Milestone**: AI-18 (Wave 2 "Stream")
> **Predecessors**: [`proposal.md`](./proposal.md) (D1–D3) · [`specs/ai-tool-call-events/spec.md`](./specs/ai-tool-call-events/spec.md) (R-ATC-001…012) · Engram obs #2346
> **Reconciliation gate — RESOLVED (2026-08-01)**: AI-14's `design.md` has landed (`cachicamas-ai-event-envelope/design.md`, Engram obs #2354). Every AI-14 symbol below is now pinned from AI-14's **landed design**, not just its spec. AI-14's code (`event.go`, `event_descriptor.go`, `sequence.go`, `stream_check.go`) is **not yet present in this worktree** — that is a pre-apply dependency, not a design gap; see "Open Questions".

> ## ⚠ Archive-time correction notice, 2026-08-01 — read before trusting the construction model below
>
> **The "Reconciled construction model" this design asserts is wrong, and the code does not implement it.** This design concluded that `NewToolCallStart/Delta/End` return a **bare `Event`** that never errors, with all validation deferred to `CheckEmit`. That conclusion was derived from AI-14's design *prose* rather than from AI-15's landed *code*, and AI-18's own "Open Questions" flagged the assumption as unverified in its last line.
>
> At apply time the agent read `response_start.go` and `completion.go`, found that AI-15 (and AI-16, AI-17, AI-19) **validate eagerly in the constructor and return `(Event, error)`**, and re-reconciled — updating `tasks.md` in place rather than deviating silently. The landed `tool_call_event.go` follows the eager-validating `(Event, error)` model like every other Wave 2 family.
>
> This design is preserved **verbatim and uncorrected** because it is the artifact the milestone was planned against, and because the gate catching it is the point: the Wave 2 verify report records it as **D13**, "a design error, caught at apply — this is the gate working", and § 3.6 calls it the only case in the wave where a reconciliation gate actually bit. The corrections live in this change's `tasks.md` and `apply-progress.md`.
>
> **One side effect outlived the correction.** AI-14 had deferred its `CheckEmit` rule-4 failure-path coverage to "AI-15+", on the assumption that a later milestone would ship a payload whose `validate` could fail *after* construction. The eager-validating model designed that receiver away, and nobody connected the two decisions. Recorded as **W1** in the Wave 2 verify report and owned by Wave 3.

## Technical Approach

One new file `backend/agent/src/ai/tool_call_event.go` adds three exported opaque payload types on AI-14's envelope, via AI-06's five-step append-only registration, once per kind. Byte discipline mirrors `tool_call.go`: values stored as `string` (comparable, immutable), byte accessors return a fresh slice per call, nothing on the path re-marshals. Validation reports only through AI-04's `Violation` + landed sentinels.

**Reconciled construction model (supersedes this design's original assumption).** AI-14's landed design fixes a **two-phase** emission model, not the single-shot `(Event, error)` this design originally assumed: `NewXxxEvent(...) Event` — bare `Event`, unstamped (`seq:0`), **never errors** — then `(*Stamper).Stamp(e) Event` assigns the sequence, then the exported `CheckEmit(e Event) error` is the **one** family-wide validation boundary (AI-14 D4/D5). `CheckEmit`'s step 4 invokes the payload's own unexported `validate(at Path) *Violation` — the same mechanism `content_part.go`'s `Part.validate` uses for its concrete payloads. Consequently `NewToolCallStart/Delta/End` do **not** return `(Event, error)`: they return bare `Event`, and the RFC-2119 "rejected at construction" language in `R-ATC-002`/`R-ATC-004` is satisfied by the payload's `validate(at Path)` firing inside `CheckEmit` — the same boundary AI-14's own `R-AEE-010` (unstamped-sequence rejection) already uses, so this is consistent with, not a deviation from, the family's existing "construction is validated at the emission boundary" posture. Tests exercise validation by calling `CheckEmit` on a constructed (optionally stamped) event, never by inspecting a constructor return error.

## Architecture Decisions

### Decision: No argument-size ceiling — explicitly declined

**Choice**: Neither a fragment nor the call-end's complete argument bytes carries a length bound. Construction accepts any size, including zero.
**Alternatives considered**: (a) reuse `MaxTextLen`; (b) mint a new `MaxToolArgumentsLen` constant.
**Rationale**: Verified in code — AI-09 ships **no** argument-length constant; `MaxTextLen` binds text/reasoning *complete content* only, an unrelated contract. (a) would silently couple tool arguments to a text bound; (b) invents a new Layer 1 bound the spec deliberately left unstated (spec open item 2) and would make a legal provider stream unconstructible at the transport layer. Contrast with AI-16's provisional `R-ATE-009`, which *reuses* an existing family constant — no analogous constant exists here, so the analogy does not carry. A future ceiling is a spec-level change, not a design add.

### Decision: Three independent structs, not one shared family struct

**Choice**: `ToolCallStart`, `ToolCallDelta`, `ToolCallEnd` — three distinct payload types (spec open item 3).
**Alternatives considered**: one struct with a phase field; a shared embedded base.
**Rationale**: The kind is derived from the payload type (`R-AEE-001`); one struct per kind is the only shape where kind and contents cannot disagree, and a phase discriminator is exactly what `R-ATC-001` forbids. Matches the one-type-per-kind `content_part.go` precedent and gives each payload its own `validate()` order and accessor set.

### Decision: Validation order — block index first, then identity, then name

**Choice**: Rule 1 on all three payloads: `blockIndex >= 1`, else `ErrOutOfRange` at `At("blockIndex")` (`R-ATC-004` = `R-ATE-003` adopted). Call-start adds rule 2 `id != ""` and rule 3 `name != ""`, each `ErrEmpty` (D3). Delta/end add **no** further rule: fragments and argument bytes are raw — zero-length, whitespace-only, invalid UTF-8, malformed JSON all accepted unaltered (`R-ATC-005/-007`).
**Alternatives considered**: identity before index (NewToolCall's order).
**Rationale**: The index rule is the family-wide invariant shared by all three kinds; putting it first makes the three rule lists agree on rule 1 and keeps V-FAIL-04 ordering predictable across the family.

### Decision: No `{}` canonicalization and no JSON check on the event path

**Choice**: `NewToolCallEnd` stores zero-length arguments as zero-length; never substitutes `emptyToolArguments`; never calls `isWellFormedJSON`.
**Alternatives considered**: reusing `NewToolCall`'s rules at construction.
**Rationale**: D2/`R-ATC-007` — one rule set, two entry points, no third; canonicalization and syntax both belong to AI-30's reassembly through `NewToolCall`.

### Decision: Constructors return bare `Event`; validation is deferred to `CheckEmit` (reconciled against AI-14's landed design)

> **This decision is the one the archive-time notice above corrects. The landed code returns `(Event, error)` and validates eagerly.**

**Choice**: `NewToolCallStart/Delta/End` return `Event`, not `(Event, error)`. Each of the three payloads implements the sealed `eventPayload` interface's unexported `validate(at Path) *Violation` (rule 1: `blockIndex >= 1` else `ErrOutOfRange` at `At("blockIndex")`; call-start adds `id != ""` then `name != ""`, each `ErrEmpty`; delta/end add no rule). Each payload also implements the optional unexported `blockPayload` interface's `blockIndex() BlockIndex` (lower-case, distinct from the exported `BlockIndex() BlockIndex` accessor), so AI-14's generic `CheckEmit` step 3 (block index vs. `Role`) and `CheckStream`'s block-ordering checker can read the index without a concrete-type switch (`R-AEE-015`).
**Alternatives considered**: (a) keep this design's original `(Event, error)` shape by validating eagerly inside the constructor, duplicating the check `CheckEmit` performs; (b) a hybrid where the constructor validates identity/name eagerly but defers only the block-index-vs-`Role` check to `CheckEmit`.
**Rationale**: AI-14's Data Flow (`design.md` line 51) types `NewXxxEvent(...) → Event{seq:0}` with no error, and D4 fixes `CheckEmit` as **the** exported emission boundary, separate from `Stamp`, precisely so a hand-built, never-stamped event can be *offered* at that boundary and rejected there (`R-AEE-010`, S-AEE-028). (a) would fork validation into two call sites disagreeing on when a rule fires and would fight AI-14's own two-phase model; the spec's "at construction" wording is satisfied because `validate(at Path)` is the payload's construction-time rule set, merely *invoked* from `CheckEmit` rather than from the raw constructor — the identical posture AI-14 already takes for its own `seq == 0` rejection.
**Type correction**: block index type is AI-14's exported `BlockIndex` (`uint64`), not `int` as originally assumed — this design's Interfaces section is corrected below.

## Interfaces / Contracts

```go
// Registry appends (in AI-14's registry file, name table strings following
// the landed "tool_call" precedent):
EventKindToolCallStart // "tool_call_start" — descriptor {start, any, terminal:false}
EventKindToolCallDelta // "tool_call_delta" — descriptor {delta, any, terminal:false}
EventKindToolCallEnd   // "tool_call_end"   — descriptor {end,   any, terminal:false}

type ToolCallStart struct{ blockIndex BlockIndex; id, name string }
func NewToolCallStart(blockIndex BlockIndex, id, name string) Event // never errors; unstamped (seq:0)
func (e Event) ToolCallStart() (ToolCallStart, bool)
// exported: BlockIndex() BlockIndex · ID() string · Name() string
// unexported (interface satisfaction only): kind() EventKind · blockIndex() BlockIndex · validate(at Path) *Violation

type ToolCallDelta struct{ blockIndex BlockIndex; fragment string }
func NewToolCallDelta(blockIndex BlockIndex, fragment []byte) Event // never errors; unstamped (seq:0)
func (e Event) ToolCallDelta() (ToolCallDelta, bool)
// exported: BlockIndex() BlockIndex · Fragment() []byte — fresh copy per call
// unexported: kind() EventKind · blockIndex() BlockIndex · validate(at Path) *Violation (no-op beyond rule 1)

type ToolCallEnd struct{ blockIndex BlockIndex; arguments string }
func NewToolCallEnd(blockIndex BlockIndex, arguments []byte) Event // never errors; unstamped (seq:0)
func (e Event) ToolCallEnd() (ToolCallEnd, bool)
// exported: BlockIndex() BlockIndex · Arguments() []byte — fresh copy per call
// unexported: kind() EventKind · blockIndex() BlockIndex · validate(at Path) *Violation (no-op beyond rule 1)
```

- Block-index type is AI-14's exported `BlockIndex` (`uint64`) — **reconciled**, no longer `int`.
- Constructor return shape is bare `Event`, **never `(Event, error)`** — **reconciled** against AI-14's landed `NewXxxEvent(...) → Event{seq:0}` data flow (D4). Validation runs later, at `CheckEmit`, via each payload's unexported `validate(at Path) *Violation`.
- Producer emission sequence: `NewToolCallXxx(...)` → `(*Stamper).Stamp(e)` → `CheckEmit(e)` — the last call is the one that can fail; RED tests for `R-ATC-002`/`R-ATC-004`/`S-ATC-005/-006/-007/-011` construct, optionally stamp, then assert on `CheckEmit`'s returned error.
- `String()`/`GoString()` return `"toolcallstart"` / `"toolcalldelta"` / `"toolcallend"` — no identity, name, bytes, or lengths (`R-ATC-008`, V-FAIL-13).
- **No ordinal field anywhere** (`R-ATC-012`); tests derive it by filtering call-start events in sequence order with test-local code. No exported accumulator or reconstruction helper ships (`R-AEE-020`).

## Data Flow

    NewToolCallXxx(...) → Event{seq:0} ──(*Stamper).Stamp──→ Event{seq:n} ──CheckEmit──→ ok/err
                                                                                 │
                                                                    consumer partitions by blockIndex()
                                                                    │ concatenates fragments (test-local)
                                                                    └──→ AI-30 reassembly → NewToolCall (single validation entry)

## File Changes

| File | Action | Description |
|---|---|---|
| `backend/agent/src/ai/tool_call_event.go` | Create | Three payloads, constructors, accessors, redacted rendering |
| `backend/agent/src/ai/tool_call_event_test.go` (`package ai_test`) | Create | S-ATC-001…038: lifecycle, byte-equality, zero-delta, interleaving under `-race`, ordinal, redaction, totality table |
| AI-14's registry file (per its design) | Modify | Three append-only entries: constant + table + descriptor + name table |

## Testing Strategy

| Layer | What | Approach |
|---|---|---|
| Unit (external) | All 38 scenarios | `ai_test` tables; validation scenarios call `CheckEmit`, not a constructor error (reconciled); test-local concatenator; `-race` on interleaving |
| Registry | Exhaustiveness over the three kinds | AI-14's assertion extended by table entries only (S-ATC-003) |
| Guards | go.mod zero requires, import guards, AST sequence guard | Existing suites, unchanged |

## Threat Matrix

N/A — no routing, shell, subprocess, VCS/PR automation, executable-file classification, or process-integration boundary.

## Migration / Rollout

No migration required — purely additive; rollback per proposal (revert single PR).

## Open Questions

- [x] AI-14 design symbols (`Event`, `EventKind`, `BlockIndex`, `Stamper`, `CheckEmit`, `eventPayload`/`blockPayload` interfaces, descriptor registration shape) — **resolved**, pinned from AI-14's landed `design.md` (Engram obs #2354) on 2026-08-01.
- [x] Constructor return shape — **resolved**: bare `Event`, never `(Event, error)`; validation deferred to `CheckEmit` (see reconciled Architecture Decision above).
- [ ] AI-14's code (`event.go`, `event_descriptor.go`, `sequence.go`, `stream_check.go`) is not yet present in this worktree. This is a `sdd-apply` **pre-condition**, not a design gap: AI-18's GREEN tasks cannot compile until AI-14's code lands on this branch or is merged to `main` and pulled in. RED tasks (tests written against the pinned signatures above) are not blocked.
- [ ] AI-15's landed constructors were not independently re-verified in this pass (AI-15 depends on AI-14 too); if AI-15's actual landed code diverges from this same bare-`Event`/`CheckEmit` pattern once it exists on disk, re-reconcile before `sdd-apply`.

> **That last open question is the one that bit.** AI-15's landed code *did* diverge, it *was* re-reconciled before apply completed, and the flag written here is what made the divergence findable rather than a surprise. See the archive-time correction notice at the top of this file.
