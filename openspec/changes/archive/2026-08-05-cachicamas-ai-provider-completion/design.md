# Design — cachicamas-ai-provider-completion (AI-31)

> **Change**: `cachicamas-ai-provider-completion` · **Phase**: design · **Date**: 2026-08-04
> Binds spec `specs/ai-provider-completion/spec.md` (R-ACP-001…012), the `ai-provider-text-stream` delta
> (R-ATS-026/S-ATS-101), `citations.md` U1–U6, and rulings D1/D2/D3.

> **Corrective re-run (design-validation gate, 2026-08-04).** Rev 2, against spec rev 2. Applied: C1
> (blanket `CheckStream` rule), C3 (+3 new mappings, not five), G2 (inline-literal fixture convention,
> one-label table), G3 (odd-position fixture preconditions + delta-less path analysis), G5 (charter-delta
> path), R-ACP-011's five amendment sites, S-ACP-017's impossible-arithmetic probe, branch-point note.
> Validator-confirmed clean and unchanged: coverage-only claim, spec-risk-3 verdict (extended to mid-stream
> frames), S-ACP-015 distinguishability, D-E record homes, all 12 requirement homes, citations.

## Technical Approach

Extend, record, prove — no new mechanism. Three surfaces:

1. **Decode** (`chunk.go`): `wireUsage` gains two typed nested detail structs; `usageFromWire` gains exactly
   **+3 new mappings** (`cached_tokens→CacheRead`, `cache_write_tokens→CacheWrite`,
   `reasoning_tokens→Reasoning`), all raw. The landed `prompt_tokens→Input` / `completion_tokens→Output`
   mappings are a **separate, frozen fact** (R-ACP-009, byte-identical tests) — this change touches neither.
   The landed `*int64` pointer-nil-on-absence discipline extends unchanged.
2. **Finish mapping is already complete in landed code**: `rawStrictFinishReason` gates the five enum members
   byte-exactly, then delegates to `ai.NormalizeFinishReason`, whose table already maps all five rows of
   R-ACP-001 (incl. `function_call → ToolCalls`, `content_filter → ContentFilter`). AI-31.1's mapping rows are
   **coverage work** (new `finish_reason_test.go` table test, row count asserted = 5), not new code.
3. **Records**: source doc comments + doc 0002 dated amendments carry the unreachability, stop-sequence-absence,
   cache-exclusivity and D3 obligations (sites in Decisions).

## Spec risk 3 — S-ACP-022 vs landed ordering: RESOLVED, no conflict

Evidence (`stream_state.go`, `applyChunk`): usage capture (`if chunk.Usage != nil { s.usage = usageFromWire(...) }`)
runs **unconditionally, ahead of all choice-0 checks** — the doc comment states this by design. A usage-bearing
frame BEFORE choice 0's terminal chunk passes `hasRequiredFields`/identity, is captured, then (empty `choices`)
returns at `choice0()`'s `!hasChoice` branch — it never reaches the R-ATS-020 row-1/2 window block, which fires
only when `terminalSeen` is already true AND the chunk carries choice-0 content or a non-null `finish_reason`.
`buildCompletion` reads `s.usage` at the sentinel regardless of when it was set. S-ACP-022 is therefore a
coverage test over landed behaviour, not a code change.

## Architecture Decisions

| # | Decision | Alternatives rejected | Rationale |
|---|---|---|---|
| D-A | Typed nested structs `wirePromptTokensDetails{CachedTokens, CacheWriteTokens *int64}` / `wireCompletionTokensDetails{ReasoningTokens *int64}`, held as pointers in `wireUsage` | `json.RawMessage` | Fields are plain JSON integers; chunk.go's own D2 reasons RawMessage exists only for byte-preservation of strings. Pointer parent + pointer leaf gives absent-object and absent-key vs explicit-0 discrimination via the exact landed mechanism (S-ACP-013, S-ACP-021) |
| D-B | Mapping stays in `usageFromWire` (chunk.go); `applyChunk`'s D10 wholesale-overwrite and `buildCompletion` untouched | Mapping at `buildCompletion` | Merge site unchanged means S-ACP-018/019 (last-populated-wins) hold by construction; `total_tokens` stays undecoded |
| D-C | Un-attested-exclusivity record = doc comment **on `usageFromWire`** quoting AI-13.4's Input sentence (`usage.go:112-113`), citing U1/U2 silence, naming AI-38.2 | Comment on `mapperState.usage` | S-ACP-016 names "the shipped usage-mapping source"; `usageFromWire` IS the mapping site |
| D-D | No arithmetic anywhere on the path: the **three new** mappings (`cached_tokens→CacheRead`, `cache_write_tokens→CacheWrite`, `reasoning_tokens→Reasoning`) map raw; the **landed, frozen** `prompt_tokens→Input`/`completion_tokens→Output` mappings stay untouched (R-ACP-009); U3 containment asserted in test only | Subtracting `cached_tokens`; clamping/reconciling | Branch B ruling; S-ACP-015 fails a subtracting implementation; S-ACP-017's impossible-arithmetic probe (`prompt_tokens: 500`, `cached_tokens: 800` → `Input=500`, `CacheRead=800`, no error/clamp) proves **no consistency arithmetic is enforced anywhere on the path** — a path enforcing any would have to fail, clamp or adjust there |
| D-E | Record homes: unreachable-values table (U5/C7/S-ATS-039 + reopen triggers) → comment at `rawStrictFinishReason`/`finishReasonEnum`; U4 stop-sequence recorded absence → `buildCompletion` doc comment; D3 CapCompletionMetadata note (three unrenderable values, AI-38.2 route) → `bridge_test.go` comment at the `RunConformanceFor(..., CapStreamingText)` call (~line 198) | agenttest edits | `agenttest` is read-only (S-ACP-027); `bridge_test.go` is openaicompat-owned and is where the run-set decision lives |
| D-F | Doc 0002 amendments as dated blockquotes **below** original wording, at **all five sites** of R-ACP-011 rev 2's table: (1) item 1 vacuity note, (2) item 2 D1 three-part split, (3) item 3 unexercisable-refusal-branches note, (4) the AI-31.1 Note's pause-resume vacuity, (5) charter Acceptance line unsatisfiable-and-unviolated note — each citing U5/D2 and agreeing with R-ACP-002's unreachability table (S-ACP-029) | Rewriting lines; the prior two-site plan | S-ACP-028 requires original visible; R-ACP-011 rev 2 enumerates five sites and forbids silent rewrite |

## Data Flow

    SSE frame → decodeChunk → wireChunk{Usage *wireUsage(+2 detail structs)}
      → applyChunk: usage captured unconditionally (position-independent), D10 overwrite
      → sentinel → buildCompletion → ai.NewCompletion(finishReason, usage)

## File Changes

| File | Action | Slice |
|---|---|---|
| `backend/agent/src/ai/openaicompat/chunk.go` | Modify: detail structs, `usageFromWire` **+3 new mappings** (Input/Output rows landed and frozen), D-C/D-E comments | 2 (comments D-E finish: 1) |
| `backend/agent/src/ai/openaicompat/finish_reason_test.go` | Create: 5-row table (S-ACP-001…003, 005), novel-value tests (S-ACP-006), stop-seq negative control (S-ACP-010) | 1 |
| `backend/agent/src/ai/openaicompat/stream_state.go` | Modify: `buildCompletion` U4 comment only | 1 |
| `backend/agent/src/ai/openaicompat/bridge_test.go` | Modify: D3 note comment only | 1 |
| `backend/agent/src/ai/openaicompat/usage_test.go` | Append new tests only; S-ATS-055…062 byte-identical (S-ACP-024) | 2–4 |
| `docs/architecture/milestones/0002-…-task-graph.md` | Dated amendments (R-ACP-011) | 1 |
| `openspec/changes/cachicamas-ai-provider-completion/specs/ai-provider-text-stream/spec.md` | Charter delta (R-ATS-026/S-ATS-101) — already authored, lives in THIS change's spec dir, reconciled against the in-flight `cachicamas-ai-provider-text-stream` change at archive | — |

## Delivery Slices (auto-chain, feature-branch-chain, budget 5000)

Slice 0 (citations U1–U6) is **done**. Frozen-test gate (`S-ATS-055…062` + full `go test -race`) at every slice.

| Slice | Node | RED tests first | Content |
|---|---|---|---|
| 1 | AI-31.1 | `TestFinishReason_FiveWireValues_MapTable` (count=5), `TestFinishReason_NovelValue_TypedMalformed`, `TestStopSequence_NothingIdentifiesMatch` | Coverage + all D-E/D-F records |
| 2 | AI-31.2a | S-ACP-011…015 tests | Detail-struct decode, raw mapping, D-C comment |
| 3 | AI-31.2b | S-ACP-017 (impossible-arithmetic probe: `prompt_tokens: 500`/`cached_tokens: 800` → raw `500`/`800`, no error, no clamp — proves no consistency arithmetic on the path), S-ACP-018/019 (multi-frame pin) | **One** dialect-conventional label — multi-frame usage — pinned per the spec's definitional inline-literal convention by its named test function, planned `TestUsage_MultiplePopulatedFrames_LastWinsNoFold` (S-ACP-019); no `testdata/` paths. S-ACP-017's probe is synthetic, makes no wire-shape claim, needs no label (R-ACP-012 rev 2) |
| 4 | AI-31.3 | S-ACP-021…023 | Tolerance/position coverage (code change expected: none — verify). S-ACP-023's "the stream check reports no protocol violation" IS an explicit `ai.CheckStream` assertion via the shared helper, not prose |

**Fixture-authoring preconditions (G3, validator-proved) for every odd-position usage frame**: it MUST carry a
non-empty `model` (else `hasRequiredFields` fails → `errMissingRequiredField`) and the SAME `id` as the
established identity (else `errSecondResponseStart`) — S-ACP-022's pre-terminal frame is only row-rule-safe
when both hold. **Delta-less non-empty-choices path (analysed)**: a chunk whose choice 0 exists but carries no
content string and a null `finish_reason` falls through every gate to the zero-event path — correct, now
recorded rather than incidental.

**Branch point (at-apply-time verification)**: AI-31's apply branches from the chain tip **current at apply
time** — after AI-30's slices, with the D8 corrective `feat/ai-28-8-d8-close-discipline` and possibly
`feat/ai-30-*` landing first. `sdd-apply` MUST resolve and record the actual tip then; this design's landed-code
reads were taken at `feat/ai-28-7-pre-decode-checks` and the frozen-test gate re-verifies them at the real base.

## Testing Strategy

Unit only, `httptest.Server` seam per landed `usage_test.go`; internal test package as landed. Inspection
scenarios (13) discharge at review against the comment/doc sites named in D-E/D-F.

**Blanket rule (C1, D8-class)**: EVERY new drained-stream test in this change asserts that
`ai.CheckStream(events)` reports no violation, via the shared stream-check helper the AI-28.8 corrective lands
(`requireCheckStreamClean` by role) — no new test hand-rolls the check or omits it. S-ACP-023's row makes this
assertion explicit; the rule extends it to every drained transcript, so a fixture that accidentally violates
stream protocol fails loudly instead of passing on its usage assertion alone.

## Threat Matrix

N/A — no routing, shell, subprocess, VCS/PR automation, executable-file classification, or process-integration boundary.

## Migration / Rollout

No migration. Rollback = revert per slice; `usageFromWire` returns to two-field form (proposal's plan).

## Open Questions

- [ ] Slice 4 may be pure coverage (landed code already position-independent, per spec-risk-3 evidence); if any
      test REDs unexpectedly, that is new information, not a mechanism change.

> **Resolved (spec rev 2, 2026-08-04)**: the R-ACP-012 third-label reading is settled — the clause is now
> explicitly IF/THEN conditional, the label table is definitional and holds exactly ONE label (multi-frame
> usage), and the fixture convention is inline literal named by test function. No longer open.
