# Design — translate the tool-call stream

> **Change**: `cachicamas-ai-provider-tool-stream` · **Milestone**: AI-30 · **Phase**: design · **Date**: 2026-08-04
> **Spec**: [`specs/ai-provider-tool-stream/spec.md`](specs/ai-provider-tool-stream/spec.md) (R-ATL-001…016, S-ATL-001…073) · **Proposal**: [`proposal.md`](proposal.md)
> **Base**: `feat/ai-28-8-d8-close-discipline` once the AI-28 D8 corrective lands (supersedes the proposal's `ai-28-7` base — see Delivery).

## Technical Approach

`mapperState` (stream_state.go) grows a per-call accumulation map and a shared block-index allocator; `wireDelta` (chunk.go) gains a byte-preserving `tool_calls` element decode; `stream.go` widens `isOutputEvent` and inherits the D8 close-discipline site; all failures are new unexported causes wrapping `ai.ErrMalformedResponse`. Assembly happens exactly once, at close, through `ai.NewToolCall` — the door `tool_call_event.go` (R-ATC-007) already names for AI-30. The test-only bridge learns to render tool events to SSE. No `src/ai`/`src/agenttest` edits, no new exported identifier, `go.mod` untouched (R-ATL-015).

## Architecture Decisions

### D1 — Block-index allocation: one shared next-free counter starting at 1

**Choice**: `mapperState.nextBlockIndex` starts at 1 and is consumed by *whichever block opens next*: the text block may mint only while the counter is still 1 (landed `textBlockIndex = 1`, R-ATS-008 rule 1); each new tool call takes the counter value and increments. Stream-wide unique, stable per call, ≥ 1, never equal to the wire `index` (R-ATL-002).
**Rejected**: reserving 1 for text and starting tools at 2 (the coordinator's "e.g."). **Verified against the landed suite**: `conformance_tool_call.go`'s text-free cases assert `calls[1]`/`calls[2]` (interleaved lines 116–122, zero-delta line 166, ordinal lines 200–201), so a tools-start-at-2 allocator fails three of the four `CapToolCalls` cases unmodified → R-ATL-012 STOP. The shared counter satisfies both those cases and the mixed case (text 1, tool 2) and S-ATL-009 (text present ⇒ no tool at 1).
**Edge**: a content string arriving after a tool block consumed index 1 (no text block yet) is unrepresentable under the landed text constant → typed failure `errTextAfterToolBlock`, never a silent re-index (R-ATL-010's posture). **Amendment hook**: if AI-29 v2 ever mints reasoning blocks, they join this same allocator — one line here.

### D2 — Wire decode: `wireToolCallElement`, byte-preserving throughout

`wireDelta` gains `ToolCalls []wireToolCallElement \`json:"tool_calls"\`` (and nothing for `function_call` — undeclared fields are already ignored by `encoding/json`, which *is* R-ATL-013's tolerate-and-skip; S-ATL-062/063 pin it, disposition comment cites C9.5 + C7 and names the exclusive-emission reopen trigger, S-ATL-064).

```go
type wireToolCallElement struct {
    Index    *int                  `json:"index"`    // pointer: absent key = C9.1 required-list violation (S-ATL-011)
    ID       json.RawMessage       `json:"id"`
    Function *wireToolCallFunction `json:"function"`
}
type wireToolCallFunction struct {
    Name      json.RawMessage `json:"name"`
    Arguments json.RawMessage `json:"arguments"`
}
```

`Arguments`, `ID` and `Name` all decode through the existing `unquoteJSONString` (chunk.go D2) — arguments because R-ATL-001 mandates it, id/name because R-ATL-003's "byte-exact" would otherwise be repaired to U+FFFD by `encoding/json`'s own string decode. No second unquoter (S-ATL-005). `type` is not decoded at all: single-member enum carrying no mapping decision (S-ATL-002; the landed "no field this milestone does not need" posture). Absent key and JSON null both read as "no value", `contentText`'s trichotomy restated.

### D3 — Per-call accumulation in `mapperState`

```go
type toolCallState struct {
    block    ai.BlockIndex
    id, name string        // "" = not yet supplied (absent and empty treated alike, R-ATL-003)
    args     []byte        // accumulated fragment bytes, capped (D5)
    pending  [][]byte      // fragments received before identity completed — flushed as deltas at start
    started  bool          // ToolCallStart emitted
}
// mapperState additions:
toolCalls      map[int]*toolCallState // keyed by wire index (dialect label 1, C9.2)
toolOpenOrder  []int                  // wire indexes in first-seen order = ascending ordinal (R-ATL-005)
nextBlockIndex ai.BlockIndex          // D1
```

Per element, in array order within chunk order: (1) `Index == nil` → `errToolElementMissingIndex`; (2) lookup/create state (create mints block, appends to `toolOpenOrder`); (3) identity merge — unset→set, identical repeat→no-op (S-ATL-015), differing non-empty→`errToolCallIdentityMismatch` (S-ATL-016/052; name "fragments" are just differing values, never concatenated, R-ATL-010); (4) once id *and* name are set and `!started`: emit `ToolCallStart`, flush `pending` as one `ToolCallDelta` per queued fragment (boundaries preserved, so S-ATL-036's 1-vs-5 counts hold); (5) non-empty fragment → cap check, append, emit delta (or queue if `!started` — start always precedes any delta, R-ATL-003); (6) empty/absent fragment → no-op (S-ATL-033/035). Buffers are per-state — cross-contamination is structurally impossible (R-ATL-004).

### D4 — Close, assembly-once, terminal window

**Close fires only at choice 0's terminal chunk** (any C2 enum member, S-ATL-025). Order inside `applyChunk` on the terminal chunk: process this chunk's own tool elements → text handling → `TextBlockEnd` (if open) → for each open call in `toolOpenOrder`: identity incomplete → `errToolCallMissingIdentity` (S-ATL-017/053); else assemble **exactly once** via `part, err := ai.NewToolCall(id, name, s.args)` — one validation door (well-formed JSON) plus zero-byte canonicalization to `{}` (R-ATL-007, R-ATC-007), no re-marshal anywhere (`NewToolCall` stores supplied bytes verbatim, V-REQ-17; S-ATL-043's forbidden-call list holds) — then `NewToolCallEnd(block, part.Arguments())`. `NewToolCall` error → malformed-assembly failure (D6). Emitted order matches S-ATL-024: `… TextBlockEnd, ToolCallEnd(ordinal 1), ToolCallEnd(ordinal 2)`, Completion still deferred to the sentinel (landed D4/D9).
**Terminal window** (terminalSeen set by a *prior* chunk — the landed split): a `tool_calls` element there is the row-1 analog → `errToolDeltaAfterClose` variant `errToolDeltaAfterCloseToolCalls` (own cause for debuggability), checked inside the existing `if s.terminalSeen` block beside the content check. The terminal chunk itself may legally carry tool elements (processed before close, same never-trips-on-the-setting-chunk discipline).
**Stream end with a call open and no terminal chunk** (sentinel-early, transport EOF, decoder truncation): R-ATL-009 truncation — the D8 close-discipline site closes open blocks, then emits the typed failure (D6/D7). Nothing parses fragments before close (S-ATL-026); ordinal remains stream-derived only, no production counter (R-ATL-011, S-ATL-058).

### D5 — The cap: `toolCallAccumulationCap = 4 * DefaultMaxFrameBytes` (32 MiB)

**Home**: stream_state.go, beside the map it bounds; documented unexported constant (`captureLimit`'s mold, S-ATL-031). **Rationale**: per-call arguments span many frames (C9.6 — no per-call end), so the self-consistent bound is a multiple of the decoder's own frame cap (`DefaultMaxFrameBytes`, 8 MiB); 4× guarantees no payload the frame cap admits in one frame can ever trip the per-call cap, admits legitimately multi-frame arguments, and still hard-bounds runaway accumulation. Relation strict: `len(args)+len(frag) > cap` fails, exactly-at-cap completes (S-ATL-029, `ErrFrameTooLarge`'s recorded reading). Per call, not per stream (S-ATL-030); the per-stream total remains unbounded in call count — recorded, matching the spec's own scope. Cause `errToolCallOverCap` keeps `ErrFrameTooLarge`'s exact compromise-comment form; the **second-consumer note** is recorded verbatim-adjacent to errors.go's existing escalation note, taxonomy untouched at nine (S-ATL-032).

### D6 — Failure wiring

| Cause | Kind | Fires |
| --- | --- | --- |
| `errToolElementMissingIndex` | var, `%w` ErrMalformedResponse | element with no `index` key (S-ATL-011) |
| `errToolCallIdentityMismatch` | var | differing non-empty id/name repeat (S-ATL-016/052) |
| `errToolCallMissingIdentity` | var | call closes with id or name never supplied (S-ATL-017/053) |
| `errToolDeltaAfterCloseToolCalls` | var | tool element in the post-terminal window |
| `errToolCallOverCap` | var, compromise comment | cap exceeded (S-ATL-028) |
| `errTextAfterToolBlock` | var | D1's unrepresentable edge |
| `truncatedToolCall` | **type**, `capturedBody`'s mold | truncation mid-call and malformed assembly (S-ATL-044…048) |

`truncatedToolCall`: unexported struct `{data []byte; cause error}`; `Error()` is a **fixed string built from no captured byte**; bytes reachable only via unexported `bytes()`; `Unwrap()` returns the wrapped `ai.ErrMalformedResponse`-chained cause so `errors.Is` holds and `errors.As` reaches it (S-ATL-045); `data` bounded by **`captureLimit`** — the existing 8 KiB bound reused, not a second one (S-ATL-047/051). This is the coordinator-confirmed `capturedBody` leak-averse posture over `nonStreamContentType`'s rendering one (R-ATL-009 disposition 3). `emitFailure` needs no change: `Category()` recognises none of these, and its fallback already assigns `ai.FailureCategoryMalformedResponse`. `isOutputEvent` widens to the three tool kinds, so `PartialOutput()` reports true once any tool event was emitted (S-ATL-044).

### D7 — D8 close discipline inherited by tool blocks

Every pre-terminal failure path closes **all** open blocks — text and tool, ascending open order — at the same single D8 close-discipline site the AI-28.8 corrective establishes (reference that landed site at apply time; do not mint a second one). A failure-path `ToolCallEnd` carries the **raw accumulated bytes as-is** (`NewToolCallEnd` validates only the block index by design): no fabricated `{}`, no byte loss; the terminal ErrorEvent remains the authoritative signal. See Open Questions Q2.

### D8 — Bridge rendering, and the conformance reconciliation this design surfaces

Rendering (test-only, `bridge_test.go`): `ToolCallStart` → chunk with element `{index, id, function{name}}`, wire index minted per block in start order (0,1,2…) — this is how identity-less neutral events get a wire `index`; `ToolCallDelta` → element `{index, function{arguments: fragment}}` via `bridgeQuoteJSONString`; `ToolCallEnd` → **zero bytes** (C9.6 — no wire end signal exists; encoding one would invent wire, the TextBlockStart/End precedent). Scripts lacking `ResponseStart`/`Completion` (cases 1 and 3 carry neither) get synthesized identity constants, a terminal chunk (`finish_reason:"tool_calls"`) and `[DONE]` — otherwise replay cannot construct a valid stream at all. **No Factory flip exists**: `CapToolCalls` is a *required* capability — `Factory` declares only the three optional ones and `declaredAbsentSkipReason` never skips required capabilities (R-CNF-004) — so `RunConformanceFor(t, factory, CapToolCalls)` simply runs the four cases; the brief's "ToolCalls:true declaration" is N/A, verified.

**Finding (blocking for S-ATL-059, not for the mapping):** two of the four cases **cannot pass unmodified against this wire**. `toolCallZeroDeltaCase` asserts drained kinds exactly `[ResponseStart, ToolCallStart, ToolCallEnd, Completion]` (`requireDrainedKinds` is exact count + per-index kind) while its end carries `{"q":"weather"}`; the wire's only argument channel is `function.arguments` fragments, and S-ATL-036 pins fragment→`ToolCallDelta` — so any faithful replay emits ≥ 1 delta and fails on count. `mixedTextAndToolCallCase` (exact 7 kinds, no delta) fails identically. The suite's exact-kind lists make delta count observable and mandate zero — contradicting `ai`'s own R-ATC-010 ("fragment count unobservable in the normalized result"). Cases 1 (interleaved) and 3 (ordinal) pass under this design (reconstruction-based assertions; D1's allocator supplies blocks 1/2). Per R-ATL-012's own escape clause: the node's conformance gate **STOPS** on those two cases and the reconciliation is raised as its own change against `ai-provider-conformance-suite` (recommended fix there: tolerate `ToolCallDelta` events between start and end, or convert both cases to reconstruction-based assertions like case 1). S-ATL-060's byte-fidelity round-trip is unaffected and stays in this change.

## Data Flow

    SSE frame ─ decoder ─ decodeChunk ─ applyChunk ─┬─ text path (landed)
                                                    └─ tool elements → toolCalls[wireIdx] buffers
                                                         │ start/delta events as identity/fragments arrive
                terminal chunk ──── close-all: NewToolCall(id,name,args) → ToolCallEnd, ascending ordinal
                sentinel ────────── buildCompletion (landed)
                any failure ─────── D8 site: close open blocks → emitFailure (D7 causes)

## File Changes

| File | Action | What |
| --- | --- | --- |
| `openaicompat/chunk.go` | Modify | `wireDelta.ToolCalls`, `wireToolCallElement`/`wireToolCallFunction`, value helpers (D2) |
| `openaicompat/stream_state.go` | Modify | D1 allocator, D3 map, D4 close, D5 cap, D6 causes |
| `openaicompat/stream.go` | Modify | `isOutputEvent` widened; stream-end open-call truncation check at the D8 site |
| `openaicompat/errors.go` | Modify | second-consumer note only (S-ATL-032) |
| `openaicompat/doc.go` | Modify | R-ATS-026 tool-call discharge, disclose-a-correction form (S-ATL-065) |
| `openaicompat/bridge_test.go` | Modify | D8 rendering, synthesis, `TestConformanceBridge_ToolCalls` |
| `openaicompat/tool_stream_*_test.go` | Create | one proof file per node + fixtures (five dialect labels named at their fixtures) |
| `src/ai`, `src/agenttest`, `go.mod` | Untouched | R-ATL-015 (S-ATL-067…070) |

## Testing Strategy

| Layer | What | How |
| --- | --- | --- |
| Unit/integration ([test] ×54) | all S-ATL [test] scenarios | real `httptest.Server` transcripts through `Stream` (landed `sseServer`/`drainAll` kit); strict TDD, RED first |
| Blanket rule 1 | **every** failure-drain test | runs the AI-28.8 shared `requireCheckStreamClean` helper — no per-scenario opt-in |
| Blanket rule 2 | S-ATL-049 panic table | one runner + `recover()` probe over every failing transcript |
| Acceptance | `RunConformanceFor(CapToolCalls)` | cases 1/3 green in this change; 2/4 gated on the D8 reconciliation change |
| Inspection (×19) | per spec list | reviewer checklist in tasks.md evidence log |

## Threat Matrix

N/A — no routing, shell, subprocess, VCS/PR automation, executable-file classification, or process-integration boundary.

## Delivery (auto-chain · feature-branch-chain · 5000 budget)

**Rebase note**: slice 1 targets `feat/ai-28-8-d8-close-discipline` once the corrective lands, not `ai-28-7`.

| Slice | Scope | First RED tests | Note |
| --- | --- | --- | --- |
| 1 | AI-30.1 decode+accumulation+cap, `function_call` skip, doc.go/errors.go notes, bridge rendering | S-ATL-011, S-ATL-006, S-ATL-013, S-ATL-028, S-ATL-062 | `R-CNF-023` **verified landed** (`agenttest.RunConformanceFor`, used by bridge_test.go:198) — no stop. Pre-declared split: bridge rendering → own PR if over budget |
| 2 | AI-30.2 empty/zero-fragment | S-ATL-033, S-ATL-034, S-ATL-036 | CapToolCalls turn-on **blocked** by D8's suite reconciliation (cases 2/4) |
| 3 | AI-30.3 byte fidelity | S-ATL-038, S-ATL-039, S-ATL-041 | fixture-heavy, tiny production delta |
| 4 | AI-30.4 truncation/malformation/unrepresentable | S-ATL-044, S-ATL-046, S-ATL-052, S-ATL-054 | `truncatedToolCall` lands here |
| 5 | AI-30.5 ordinal + conformance re-run | S-ATL-055, S-ATL-057 | test-side only (S-ATL-058) |

Requirement homes: R-ATL-001→chunk.go(s1) · 002/004/005/006→stream_state.go(s1) · 003→stream_state.go(s1) · 007→close path(s2) · 008→(s3) · 009/010→(s4) · 011→test-side(s5) · 012→bridge(s1)+gate(s2) · 013→(s1) · 014→doc.go(s1) · 015/016→inspection, all slices. **Homeless: zero.**

## Guard Sweep

- `S-ART-054` allowlist (`ErrFrameTooLarge`/`ErrTruncated`): untouched — every new cause unexported (S-ATL-068).
- `R-ATS-026`/`S-ATS-100`: no executable guard exists (verified — no landed test asserts tool-call absence; the "skip" was structural: `wireDelta` never declared the field). Discharge is documentary only (R-ATL-014); `tolerance_test.go` S-ATS-067 (refusal) untouched.
- `S-ATS-042` no-agenttest-edits: inherited; bridge stays in `openaicompat` `_test.go` (S-ATL-061).
- Tool event kinds already registered `BlockRoleStart/Delta/End`, `CardinalityAny` (event.go:195–206) — `CheckStream`/`CheckEmit` need no change.
- `hasValidIndex` (R-ATS-020 row 4) stays scoped to content-carrying items — its row text says so; extending it to tool elements would be uncited invention.
- AI-28.8's `requireCheckStreamClean`: consumed by every new failure test (Testing, rule 1).

## Open Questions

- [ ] **Q1 (spec-level, blocks S-ATL-059 only)**: D8's finding — `toolCallZeroDeltaCase` and `mixedTextAndToolCallCase` cannot pass unmodified (exact-kind asserts vs. the wire's fragment-only argument channel; they also contradict R-ATC-010). Recommended reading: R-ATL-012's STOP clause is the sanctioned path — raise the suite reconciliation as its own change; spec's S-ATL-059 should be re-scoped to the two passable cases until it lands. Coordinator decision requested.
- [ ] **Q2**: failure-path hygiene `ToolCallEnd` (D7) carries raw accumulated bytes — recommended (no fabrication, `NewToolCallEnd` validates nothing by design), but it grazes R-ATL-007's "every call this adapter emits carries decodable argument bytes" rationale on failure streams. Alternative: empty-bytes ends. Sign-off requested with the D8 corrective's landed shape in view.
- [ ] **Q3 (resolved, recorded)**: S-ATL-016 repeat-same vs repeat-different — **not ambiguous**: R-ATL-003's text plus S-ATL-015 pin repeat-identical as tolerated; S-ATL-016 fires only on differing non-empty values. No spec change needed.
