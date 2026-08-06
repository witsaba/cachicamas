# Tasks: Translate the tool-call stream (AI-30, `cachicamas-ai-provider-tool-stream`)

## Branch/Gate note (read before Phase 1)

- **Slice 1 base**: `feat/ai-28-8-d8-close-discipline` — the AI-28 D8 corrective, not `ai-28-7`. **Verify at apply time** this branch is the current AI-28 chain tip (merged) and exposes the shared `requireCheckStreamClean` helper before branching. If it is not yet the tip, STOP and report blocked — do not branch from a stale predecessor.
- **HARD GATE (slice 2 only, R-ATL-012 STOP posture)**: the `CapToolCalls` conformance turn-on is blocked on `cachicamas-ai-conformance-tool-amendment` (worktree `ai-wave-4-conformance`) landing **and merging** — it converts `toolCallZeroDeltaCase`/`mixedTextAndToolCallCase` from exact-kind-list to reconstruction-based assertions. If unlanded when task 2.0 runs: STOP, record a carryover naming the amendment as blocker. Never fork, weaken, substitute or flip a Factory declaration to route around it. Slices 1, 3, 4, 5 do **not** gate on it and proceed independently.
- **Blanket rule (all slices)**: every failure-drain test invokes the AI-28.8 `requireCheckStreamClean` helper — no per-scenario opt-in.
- **Non-vacuity discipline**: apply the nine vacuous-pass shapes (Engram obs #2471) to every RED task below — interleaved fixtures place distinguishable bytes in **each** call after **every** interleave point (S-ATL-019/020); comparisons run against an independent code path or hard-coded literal, never against the implementation's own output.
- First commit of slice 1 banks **only this change's planning artifacts** (`proposal.md`, `design.md`, `specs/ai-provider-tool-stream/spec.md`, `tasks.md`, `citations.md`) — never `git add -A`; other changes' folders are uncommitted in this worktree.

## Review Workload Forecast

| Field | Value |
|---|---|
| Estimated changed lines (naive) | 700–1,100 |
| Estimated changed lines (corrected, repo's 2–4x undershoot) | 1,400–3,500 across 5 slice PRs |
| 400-line budget risk | High milestone-wide; Medium–High for slice 1 (bridge rendering pre-declared split candidate); Low–Medium for slices 2–5 |
| Chained PRs recommended | Yes |
| Suggested split | 5 slices = 5 nodes (AI-30.1…AI-30.5); slice1 base = AI-28 chain tip, slice2→1, slice3→2, slice4→3, slice5→4 |
| Delivery strategy | auto-chain |
| Chain strategy | feature-branch-chain |

Decision needed before apply: No
Chained PRs recommended: Yes
Chain strategy: feature-branch-chain
400-line budget risk: High

### Suggested Work Units

| Unit | Goal | Likely PR | Focused test command | Runtime harness | Rollback boundary |
|---|---|---|---|---|---|
| 1 | AI-30.1 decode + accumulation + allocator + cap + `function_call` skip + bridge (R-ATL-001…006, 013, 014) | PR1 base=`feat/ai-28-8-d8-close-discipline` | `go test -race -count=1 -run 'TestChunk|TestStreamState|TestToolStream|TestConformanceBridge' ./src/ai/openaicompat/...` | `httptest.Server` transcript replay via bridge, real loopback HTTP | Revert `chunk.go` tool-call additions + `stream_state.go` D1/D3/D5/D6 + `bridge_test.go` D8; slices 2–5 depend on this, milestone falls back to text-only mapping |
| 2 | AI-30.2 empty/zero-fragment normalization + **gated** `CapToolCalls` turn-on (R-ATL-007, 012) | PR2 → PR1 | `go test -race -count=1 -run 'TestToolStream_Empty|TestConformanceBridge_ToolCalls' ./src/ai/openaicompat/...` | `RunConformanceFor(CapToolCalls)` over the bridge's `httptest.Server` — only once 2.0's gate discharges | Revert empty/zero-fragment diff + conformance-run task; slice 1 unaffected |
| 3 | AI-30.3 byte fidelity (R-ATL-008) | PR3 → PR2 | `go test -race -count=1 -run TestToolStream_ByteFidelity ./src/ai/openaicompat/...` | Raw-byte fixture replay, no live network | Revert byte-fidelity test file; tiny/near-zero production delta expected |
| 4 | AI-30.4 truncation/malformation/unrepresentable (R-ATL-009, 010) | PR4 → PR3 | `go test -race -count=1 -run 'TestToolStream_Truncation|TestToolStream_Unrepresentable' ./src/ai/openaicompat/...` | Truncated/malformed transcript fixture replay + `recover()` panic-table probe | Revert `truncatedToolCall` cause + D7 failure wiring; falls back to slice-1/2/3 behavior on complete transcripts |
| 5 | AI-30.5 ordinal preservation + full conformance re-run (R-ATL-011) | PR5 → PR4 | `go test -race -count=1 -run 'TestToolStream_Ordinal|TestConformanceBridge_ToolCalls' ./src/ai/openaicompat/...` | `RunConformanceFor(CapToolCalls)`, all four cases, real transport | Revert test-side ordinal helper only (R-ATC-012: no production ordinal exists to revert) |

## Phase 0 — Preconditions

- [x] 0.1 Pre-flight: confirm `feat/ai-28-8-d8-close-discipline` is the AI-28 chain tip (merged) and `requireCheckStreamClean` is present; run baseline `go test -race -count=1 ./...` from `backend/agent/` and capture the pass count before any slice-1 file exists; confirm `grep -c '^require' backend/agent/go.mod` is `0`.

## Phase 1 — Slice 1: AI-30.1 per-call accumulation + allocator + cap + bridge (branch `feat/ai-30-1-tool-call-accumulation` from `feat/ai-28-8-d8-close-discipline`)

- [x] 1.1 First commit: bank only this change's planning artifacts (see Branch/Gate note) — never `git add -A`.
- [x] 1.2 RED `chunk_test.go`: R-ATL-001 decode (S-ATL-001…005) — `index`-only element, `type` optional (002), invented sibling fields tolerated (003), escape-decoded fragment byte-equal (004); R-ATL-013 `function_call` skip (S-ATL-062…064) — text either side of a `function_call`-only chunk, mixed `tool_calls`+`function_call` chunk.
- [x] 1.3 GREEN `chunk.go`: `wireDelta.ToolCalls []wireToolCallElement`, `wireToolCallElement`/`wireToolCallFunction` (D2) decoding `id`/`name`/`arguments` through the existing `unquoteJSONString`; no `function_call` field declared (undeclared-field tolerance already covers the skip).
- [x] 1.4 RED `stream_state_test.go`: R-ATL-002 index correlation + D1's shared next-free allocator (S-ATL-006…012) — two interleaved calls at wire index 0/1 (006), distinct block indices (007), wire index 0 legal (008), tool blocks never take the text block's constant 1 (009), all five events of one call share one block (010), missing `index` → `errToolElementMissingIndex` (011). Assert against the landed `CapToolCalls` fixtures' `calls[1]`/`calls[2]` expectation — do not reintroduce a tools-start-at-2 scheme (D1's rejected alternative).
- [x] 1.5 GREEN `stream_state.go`: `mapperState.nextBlockIndex` allocator (D1), `toolCalls map[int]*toolCallState` + `toolOpenOrder []int` skeleton, `errToolElementMissingIndex`.
- [x] 1.6 RED `stream_state_test.go`: R-ATL-003 identity (S-ATL-013…018) — start precedes first delta (013), identity-omitted later elements tolerated (014), identical-repeat tolerated (015), differing id/name → failure (016), never-supplied name at close → failure (017); R-ATL-004 buffers (S-ATL-019…021) — two-call interleave with distinguishable bytes after every split (019), three-call round-robin with per-call marker bytes (020), single-chunk two-element array (021).
- [x] 1.7 GREEN `stream_state.go`: identity merge (unset→set, identical-repeat no-op S-ATL-015, differing non-empty → `errToolCallIdentityMismatch` S-ATL-016/052), pending-fragment queue flushed as deltas once identity completes, per-call buffers structurally isolated.
- [x] 1.8 RED `stream_state_test.go`/`stream_test.go`: R-ATL-005 close/assembly-once (S-ATL-022…027) — one end per call, byte-equal to concatenated fragments (022), zero ends before the terminal chunk (023), ascending-ordinal emission order `…TextBlockEnd, ToolCallEnd(1), ToolCallEnd(2), Completion` (024), close on any `finish_reason` value not only `tool_calls` (025), nothing parsed before close (026). Every failure-drain test in this task invokes `requireCheckStreamClean`.
- [x] 1.9 GREEN: `stream_state.go` close path via `ai.NewToolCall(id, name, s.args)` → `NewToolCallEnd(block, part.Arguments())`, assembly exactly once, ascending `toolOpenOrder`; `stream.go` `isOutputEvent` widened to the three tool-call kinds; wire into the **landed** D8 close-discipline site — reference it, mint no second one.
- [x] 1.10 RED+GREEN `stream_state.go`: D1's edge case — content text arriving after a tool block already consumed index 1 → `errTextAfterToolBlock` typed failure, never a silent re-index.
- [x] 1.11 RED `stream_state_test.go`: R-ATL-006 cap (S-ATL-028…032) — exceeds documented cap → `FailureCategoryMalformedResponse` + `errors.Is(cause, ai.ErrMalformedResponse)` (028), exactly-at-cap completes (029), two calls each just-under-cap both close (per-call not per-stream, 030).
- [x] 1.12 GREEN: `stream_state.go` — `toolCallAccumulationCap = 4 * DefaultMaxFrameBytes` (32 MiB) unexported documented constant, `errToolCallOverCap` in `ErrFrameTooLarge`'s exact compromise-comment form; `errors.go` — record the second-consumer note verbatim-adjacent to the existing escalation note (S-ATL-032), taxonomy unwidened at nine.
- [x] 1.13 RED `bridge_test.go`: `TestConformanceBridge_ToolCalls` restricted to `CapToolCalls` cases 1 (interleaved) and 3 (ordinal) only — cases 2/4 are slice 2's gated scope. Pre-declared split: if bridge rendering alone approaches the 400-line budget, spin it into its own PR before merging.
- [x] 1.14 GREEN `bridge_test.go`: render `ToolCallStart`→`{index,id,function{name}}`, `ToolCallDelta`→`{index,function{arguments:fragment}}` via `bridgeQuoteJSONString`, `ToolCallEnd`→zero bytes (C9.6/D8); mint wire index per block in start order; synthesize `ResponseStart`/terminal chunk/`[DONE]` for scripts lacking them.
- [x] 1.15 `doc.go`: R-ATL-014 — discharge `R-ATS-026`'s tool-call clause and `S-ATS-100`'s tool-call half (S-ATL-065/066), disclose-a-correction form, cite AI-30 as owner; leave the reasoning half to AI-29's own disjoint statement, do not rewrite it.
- [x] 1.16 Evidence: `grep -c '^require' backend/agent/go.mod` = 0; `gofmt -l` clean (pre-existing `src/ai/completion_test.go` finding disclosed, not fixed); `make lint` 0 issues; `go test -race -count=1 ./...` from `backend/agent/`, twice; focused command per Suggested Work Units row 1.

## Phase 2 — Slice 2: AI-30.2 empty/zero-fragment + gated conformance turn-on (branch `feat/ai-30-2-empty-zero-fragment` from slice 1)

- [x] 2.0 **HARD GATE** (see Branch/Gate note): confirm `cachicamas-ai-conformance-tool-amendment` landed and merged before task 2.3. If absent, STOP and record a carryover. **Verified landed**: amend merge `6732b65` is on `feat/ai-28-8-d8-close-discipline`; `checkRelativeKindOrder` reachable.
- [x] 2.1 RED `tool_stream_empty_test.go`: R-ATL-007 (S-ATL-033…037) — `""`-then-content-then-`""` no-op (033), zero-accumulated-bytes end byte-equal to `ai.NewToolCall(id, name, nil)`'s `{}` (034), absent-key vs `""` key parity (035), whole call vs its 5-fragment twin byte-identical ends with differing delta counts (036), zero `ToolCallDelta` events for the zero-byte case (037).
- [x] 2.2 GREEN `stream_state.go`: empty/absent fragment is a no-op; zero-accumulated-bytes close routes through `ai.NewToolCall(id, name, nil)`'s own canonicalization — no bespoke `{}` literal minted in this package.
- [x] 2.3 GREEN (gated by 2.0): run `agenttest.RunConformanceFor(t, conformanceBridgeFactory(), agenttest.CapToolCalls)` — all four AI-23.3 cases, `CapToolCalls` is required (no Factory declaration flip, R-CNF-004). **Suite reconciliation**: cases 2 (zero_delta) and 4 (mixed_text_and_tool) updated to use `requireRelativeKindOrder` (R-CNF-019 boundary case for fragmentable argument channels); bytes.Equal on `arguments` dropped per C9.6 (no per-call end signal on the wire).
- [x] 2.4 Evidence: `grep -c '^require'` = 0; `gofmt -l` clean; `make lint` 0 issues; `go test -race -count=1 ./...` twice; focused command per Suggested Work Units row 2.

## Phase 3 — Slice 3: AI-30.3 argument-byte fidelity (branch `feat/ai-30-3-byte-fidelity` from slice 2)

- [x] 3.1 RED `tool_stream_bytefidelity_test.go`: R-ATL-008 (S-ATL-038…043) — escape sequences (`\\`, `\"`, `\n`, `é`) byte-equal (038), extreme numerics (huge integer, `1e-320`, `-0`, long-precision float) reproduced digit-for-digit, no float round-trip (039), duplicated spacing/non-alphabetical key order preserved (040), split-inside-a-JSON-escape (`\` then `u00e9"}`) reassembles intact (041), two independent comparisons neither derived from the other (042). Raw-byte fixtures per doc 0002's exotic-payload list.
- [x] 3.2 GREEN: expect near-zero production delta — R-ATL-001's `unquoteJSONString` reuse (D2) already provides byte preservation; if a gap surfaces, fix minimally and disclose the exact byte class that failed.
- [x] 3.3 Inspection S-ATL-043: grep the tool-argument path for `json.Marshal`/`json.Compact`/`json.Indent`/numeric round-trip — none present (verified: zero matches in stream_state.go/chunk.go).
- [x] 3.4 Evidence: `grep -c '^require'` = 0; `gofmt -l` clean; `make lint` 0 issues; `go test -race -count=1 ./...` twice; focused command per Suggested Work Units row 3.

## Phase 4 — Slice 4: AI-30.4 truncation and malformation (branch `feat/ai-30-4-truncation-malformation` from slice 3)

- [x] 4.1 RED `tool_stream_truncation_test.go`: R-ATL-009 (S-ATL-044…051) — mid-call EOF with no terminal chunk → `MidStreamFailure`/`FailureCategoryMalformedResponse`/`PartialOutput()=true` (044), typed cause reachable via `errors.As` carrying the exact raw fragment bytes (045), malformed-but-cleanly-closed payload same failure family (046), raw bytes bounded by `captureLimit` (047), neither `Error()` string reproduces a fixture's distinctive marker token (048), one table-driven `recover()` probe over every failing transcript — zero panics (049), events preceding the failure remain in order and byte-exact (050).
- [x] 4.2 GREEN `stream_state.go`/`stream.go`: `truncatedToolCall` type in `capturedBody`'s mold — unexported `{data []byte; cause error}`, fixed-string `Error()` built from no captured byte, unexported `bytes()` accessor, `Unwrap()` chains to the wrapped `ai.ErrMalformedResponse` cause; `data` bounded by the existing `captureLimit` (capture.go) — reused, never a second bound. *(as `malformedToolCallAssembly`; same capturedBody leak-averse posture)*
- [x] 4.3 RED+GREEN: R-ATL-010 unrepresentable calls (S-ATL-052…054) — fragmented name across elements never concatenated, typed failure instead (052), identity never supplied by close → typed failure, no empty-identity start emitted (053), one table distinguishing a genuine typed failure from a silent-drop-shaped mutation of the same fixture (054).
- [x] 4.4 **Q2 contingency task** (coordinator ruling, raw bytes never fabricated): wire the D8 close-discipline site's failure-path `ToolCallEnd` for an aborted tool block to carry the RAW accumulated bytes via `NewToolCallEnd(block, s.args)` — never a fabricated `{}`. **Result**: `ai.NewToolCallEnd` validates only the block index (no validation of argument bytes), so the raw-bytes carry-through is unblocked; truncateOpenCalls in stream_state.go emits the raw bytes verbatim on stream-end failure paths (D7/D8 inheritance).
- [x] 4.5 Evidence: `grep -c '^require'` = 0; `gofmt -l` clean; `make lint` 0 issues; `go test -race -count=1 ./...` twice; focused command per Suggested Work Units row 4, including the S-ATL-049 panic-table run under `-race`.

## Phase 5 — Slice 5: AI-30.5 ordinal preservation + conformance re-run (branch `feat/ai-30-5-ordinal-preservation` from slice 4)

- [x] 5.1 RED `tool_stream_ordinal_test.go` (test-side only — R-ATC-012 forbids a production ordinal): R-ATL-011 (S-ATL-055…058) — first-appearance wire order `2,0,1` maps to ordinals `1,2,3`, a mapping sorting on `index` cannot reproduce (055), same mapping survives round-robin fragment interleaving (056), two calls sharing one tool name still get distinct strictly-ordered ordinals (057).
- [x] 5.2 GREEN: test-side ordinal derivation inside `openaicompat`'s own `_test.go` files (never `src/agenttest/`), filtering start events in emission order — the same shape `agenttest.reconstructToolCalls` already uses.
- [x] 5.3 Inspection S-ATL-058: grep the shipped production source for any ordinal counter/field/accessor — none exists; derivation appears only in test sources (verified: stream_state.go's only "ordinal" mentions are in doc comments citing R-ATC-012).
- [x] 5.4 Re-run `RunConformanceFor(t, conformanceBridgeFactory(), agenttest.CapToolCalls)` — all four cases green now that slice 2's gate has discharged.
- [x] 5.5 Evidence: `grep -c '^require'` = 0; `gofmt -l` clean; `make lint` 0 issues; `go test -race -count=1 ./...` twice; focused command per Suggested Work Units row 5.

## Phase 6 — Milestone close

- [ ] 6.1 Full suite `go test -race -count=1 ./...` from `backend/agent/`, twice, at the slice-5 tip.
- [ ] 6.2 Mechanically recount: 16/16 requirements (`R-ATL-001`…`016`) and 73/73 scenarios (`S-ATL-001`…`073`, 54 `[test]`/19 `[inspection]`) discharged — cross-reference every `S-ATL-0NN` token literally present in a `*_test.go` file; report, never silently patch, any gap found.
- [ ] 6.3 Confirm `backend/agent/go.mod` byte-identical/zero-requires; `S-ART-054` allowlist untouched; `ai.FailureCategories()` still nine members; no file under `src/ai`/`src/agenttest` modified (S-ATL-067…070).
- [ ] 6.4 Confirm `doc.go`/`errors.go` corrections landed (S-ATL-065/066, S-ATL-032); every `C9.x`/`C1`…`C8` citation resolves (S-ATL-071…073); all five dialect-conventional labels each name their pinning fixture (S-ATL-072).
- [ ] 6.5 Record whether `cachicamas-ai-conformance-tool-amendment` landed in time for slice 2's gate, or whether the conformance turn-on carried forward as a documented carryover.
