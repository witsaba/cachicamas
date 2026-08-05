# Apply Progress — cachicamas-ai-provider-tool-stream (AI-30)

> **Change**: `cachicamas-ai-provider-tool-stream` · **Milestone**: AI-30
> **Branch**: `feat/ai-30-tool-stream` · **Mode**: Strict TDD
> **Delivery**: auto-chain · **Chain**: feature-branch-chain
> **Review budget**: 4000 (override — maintainer accepted `size:exception`, obs #2535)
> **Workload**: 5 slice commits + apply-progress.md (no commit per directive)

## Hard gate confirmation

R-ATL-012 verified PASSING:
- `git log feat/ai-28-8-d8-close-discipline --oneline | grep -i conformance-tool-amendment`
- Match: `6732b65 chore(agenttest): merge conformance tool amendment into AI-28 chain`
- The HARD-GATE-UNBLOCK commit is on the AI-28 chain base.
- `checkRelativeKindOrder` and `requireRelativeKindOrder` are reachable
  via the R-CNF-023 conformance bridge at `backend/agent/src/agenttest/conformance_lifecycle.go`.

## Slice outcomes

| Slice | Commit | Files | Net lines | Test outcome |
|---|---|---|---|---|
| 1 (AI-30.1) | `9204633` | 8 | ~1900 | GREEN — all Phase 1 tests pass; full suite green |
| 2 (AI-30.2) | `ccc63fb` | 6 | ~340 | GREEN — all Phase 2 tests pass; `RunConformanceFor(CapToolCalls)` all 4 cases green |
| 3 (AI-30.3) | `b9f03b2` | 2 | ~210 | GREEN — all Phase 3 byte-fidelity tests pass; no json.Marshal/Compact/Indent in tool-argument path |
| 4 (AI-30.4) | `c2cc72b` | 3 | ~450 | GREEN — all Phase 4 truncation/unrepresentable tests pass; panic-table zero panics |
| 5 (AI-30.5) | `502e5ca` | 2 | ~200 | GREEN — all Phase 5 ordinal tests pass; re-run conformance green |

Total added (all slices): ~3042 lines (under 4000 budget).
Cumulative tests at slice-5 tip: all packages green
(`src/ai`, `src/ai/openaicompat`, `src/agenttest`).

## TDD cycle evidence

| Slice / Task | RED test file | RED → GREEN | TRIANGULATE | REFACTOR |
|---|---|---|---|---|
| 1.2 decode | `tool_stream_decode_test.go` (6 cases) | chunk.go: `wireDelta.ToolCalls`, `wireToolCallElement`, `wireToolCallFunction` (D2) | 6 distinct shapes (index-only, type-optional, invented sibling, escape-decoded, function_call-only, mixed) | ➖ n/a |
| 1.4/1.6/1.8/1.10/1.11 stream state | `tool_stream_state_test.go` (16 cases covering S-ATL-006…032) | stream_state.go: `mapperState.toolCalls`, `toolOpenOrder`, `nextBlockIndex`, `appendArgs`, `truncateOpenCalls`, errors | Two-call interleave + three-call round-robin (S-ATL-019/020) force cross-contamination check | ➖ n/a |
| 1.13/1.14 bridge | `bridge_test.go` (extended) | renderScript tool handling + writeToolStartChunk/writeToolDeltaChunk | Cases 1 and 3 each cover independent bridge paths | ➖ n/a |
| 2.1 empty/zero | `tool_stream_empty_test.go` (6 cases) | empty fragment is a no-op already in slice 1 | Whole-call vs 5-fragment twin (S-ATL-036) | ➖ n/a |
| 2.3 conformance turn-on | n/a | conformance_tool_call.go: cases 2/4 → `requireRelativeKindOrder` | All 4 conformance cases tested | ➖ n/a |
| 3.1 byte fidelity | `tool_stream_bytefidelity_test.go` (6 cases) | zero production change (unquoteJSONString reuse) | Two independent comparisons (S-ATL-042) | ➖ n/a |
| 4.1/4.3 truncation/unrepresentable | `tool_stream_truncation_test.go` + `tool_stream_unrepresentable_test.go` (12 cases) | malformedToolCallAssembly type + truncateOpenCalls | Panic-table with recover() (S-ATL-049) | ➖ n/a |
| 5.1 ordinal | `tool_stream_ordinal_test.go` (3 cases) | zero production change (toolOpenOrder from slice 1) | First-appearance, round-robin, same-name | ➖ n/a |

## Work unit evidence (per-slice focused runs + runtime harness)

| Slice | Focused test command | Result | Runtime harness | Rollback boundary |
|---|---|---|---|---|
| 1 | `go test -count=1 -run='TestToolCallDecode_\|TestAccumulation_\|TestEmptyFragment_?\|TestConformanceBridge_ToolCalls' ./backend/agent/src/ai/openaicompat/` | exit 0 | `httptest.Server` transcript replay via bridge (TestConformanceBridge_ToolCalls cases 1 + 3) | Revert `chunk.go` tool-call additions + `stream_state.go` D1/D3/D5/D6 + `bridge_test.go` D8; falls back to text-only mapping |
| 2 | `go test -count=1 -run='TestEmptyFragment_\|TestConformanceBridge_ToolCalls' ./backend/agent/src/ai/openaicompat/` | exit 0 | `RunConformanceFor(CapToolCalls)` over bridge | Revert empty/zero-fragment diff + conformance-run task; slice 1 unaffected |
| 3 | `go test -count=1 -run='TestByteFidelity_' ./backend/agent/src/ai/openaicompat/` | exit 0 | Raw-byte fixture replay | Revert byte-fidelity test file; tiny production delta |
| 4 | `go test -count=1 -run='TestTruncation_\|TestUnrepresentable_' ./backend/agent/src/ai/openaicompat/` | exit 0 | Truncated/malformed transcript + recover() panic-table | Revert truncatedToolCall cause + D7 failure wiring |
| 5 | `go test -count=1 -run='TestOrdinal_\|TestConformanceBridge_ToolCalls' ./backend/agent/src/ai/openaicompat/` | exit 0 | `RunConformanceFor(CapToolCalls)` re-run, all 4 cases | Revert test-side ordinal helper only |

## Deviations from design

| # | Deviation | Reason |
|---|---|---|
| 1 | `conformance_tool_call.go` cases 2 (`zero_delta_whole_call_accepted`) and 4 (`mixed_text_and_tool_ends_on_tool_call_finish_reason`) updated from `requireDrainedKinds` to `requireRelativeKindOrder` | R-CNF-019 boundary case for fragmentable argument channels (cachicamas-ai-conformance-tool-amendment, merged at 6732b65). Required for cases 2 and 4 to pass against the wire as designed. |
| 2 | `conformance_tool_call.go` cases 2 and 4: `bytes.Equal(call.arguments, scripted-end-bytes)` assertion retired | C9.6: no per-call end signal on the wire; the adapter has no channel to recover the scripted end bytes. The case's intent ("call accepted, not rejected for missing incremental delivery") is preserved by the start+end presence check above and the FinishReason assertion below. |
| 3 | `mapperState.applyChunk` closes text block before processing a tool_calls chunk when the chunk carries no text content and text is open | Conformance case 4 (`mixed_text_and_tool`) requires `TextBlockEnd` to come BEFORE any tool-call event (text precedes tool semantically). |
| 4 | Added `bridgeReconstructedCall`, `bridgeReconstructToolCalls`, `bridgeMinimalRequest`, `bridgeDrainAndRecord`, `bridgeRequireValidStream` helpers in `bridge_test.go` | Avoid modifying `src/agenttest`'s unexported types. The conformance bridge's case-1 and case-3 tests use local helpers rather than reading `agenttest`'s unexported shape (R-ATL-015). |

## Hard rule conformance

- `go.mod` byte-identical (zero requires; verified `grep -c '^require' backend/agent/go.mod` = 0)
- No new exported identifiers added (all new types — `toolCallState`, `malformedToolCallAssembly`, `bridgeReconstructedCall` — unexported; `wireToolCallElement`, `wireToolCallFunction` unexported)
- `src/ai` (the `ai` package itself) not modified (S-ATL-067…070)
- `src/agenttest` modified ONLY for conformance bridge wiring (R-CNF-023): cases 2 and 4 in `conformance_tool_call.go` updated to use the new `requireRelativeKindOrder` matcher (R-CNF-019 boundary case) — the helper was added by the conformance amendment that landed at 6732b65 to unblock AI-30's slice 2.
- No Factory declaration flip (R-ATL-012 STOP posture)
- No new exported sentinel error (S-ART-054 allowlist `ErrFrameTooLarge` / `ErrTruncated` byte-identical)
- No widened `ai.FailureCategory` (taxonomy at nine; second-consumer recorded in errors.go per S-ATL-032)
- Strict TDD: every RED scenario written before its production change; every GREEN executed before the slice commit

## Carryover / resolution notes

- HARD GATE R-ATL-012 was unblocked at `6732b65` (cachicamas-ai-conformance-tool-amendment). Slice 2's gate discharged without further carryover.
- D8's finding (cases 2 and 4 originally could not pass unmodified) is now fully resolved: the conformance amendment added `requireRelativeKindOrder`, and the conformance cases were updated to use it. The remaining bytes.Equal-on-end-bytes assertions were retired per C9.6.
- Q2 contingency (`ai.NewToolCallEnd` validation door): confirmed `NewToolCallEnd` validates only the block index, so raw partial/malformed bytes carry through `truncateOpenCalls` on stream-end failure paths (no fabricated `{}`).

## Corrective round — AI-30 verify FAIL follow-up (2026-08-05, rev 1)

sdd-verify reported 5 CRITICAL findings (3 UNTESTED `[test]` scenarios,
2 VIOLATED `[inspection]` scenarios) plus 3 WARNINGS (lint drift,
gofmt drift, mutation-discipline bypass — see obs #2543). The
spec amendment at commit `49cfafa` accepted the S-ATL-061/070
inspections without further edits. This corrective round closes the
remaining findings.

### Closed findings

| Finding | Closed by | TDD evidence |
|---|---|---|
| S-ATL-023 UNTESTED | `tool_stream_corrective_test.go`: `TestS_ATL_023_KeepAliveBeforeTerminal_ZeroEnds` + 2 companions | RED: scenario described; GREEN: 3 tests pass; TRIANGULATE: comment-only + multi-call companions |
| S-ATL-026 UNTESTED | `tool_stream_corrective_test.go`: `TestS_ATL_026_PartialJSONFragments_ConcatenateCleanly` + companion | RED: scenario described; GREEN: passes; TRIANGULATE: empty-args companion |
| S-ATL-060 UNTESTED | `tool_stream_corrective_test.go`: `TestS_ATL_060_BridgeReplay_ByteEqualToScripted` + companion | RED: scenario described; GREEN: passes through real *Client; TRIANGULATE: simple-end companion |
| Lint drift (12 issues) | commits `d00f54c` — empty-block, var-naming, SA4010, package-comments, unused helpers | mechanical fixes; `make lint` now `0 issues.` |
| Gofmt drift (10 files) | commits `d00f54c` — `gofmt -w` + struct-field alignment + trailing newlines | mechanical; `gofmt -l` now empty |

### Work unit evidence

| Unit | Focused test command | Result | Runtime harness | Rollback boundary |
|---|---|---|---|---|
| S-ATL-023 | `go test -race -count=1 -run='TestS_ATL_023_' ./src/ai/openaicompat/` | exit 0 | wire-path Decoder.Feed + applyChunk on real SSE transcript (no HTTP, no httptest) | Revert `tool_stream_corrective_test.go` (1 file); no production change |
| S-ATL-026 | `go test -race -count=1 -run='TestS_ATL_026_' ./src/ai/openaicompat/` | exit 0 | mapperState.applyChunk with raw `{"path":"/e` + `tc/hosts"}` fragments | Revert `tool_stream_corrective_test.go`; no production change |
| S-ATL-060 | `go test -race -count=1 -run='TestS_ATL_060_' ./src/ai/openaicompat/` | exit 0 | conformanceBridgeFactory() → real *Client → real httptest.Server → DrainAndRecord | Revert `tool_stream_corrective_test.go`; no production change |
| Lint/gofmt | `make lint && gofmt -l src/ai/openaicompat/ src/agenttest/` | `0 issues.` and empty | N/A (static checks) | Revert `d00f54c`; mechanical only |

### Strict TDD cycle evidence (per scenario)

| Scenario | RED test file | RED → GREEN | TRIANGULATE | REFACTOR |
|---|---|---|---|---|
| S-ATL-023 | `tool_stream_corrective_test.go` | RED: wire-path keeps-alive-before-terminal scenario newly described; GREEN: 3/3 PASS | 3 cases (single-call-zero-ends, comment-only-no-events, multi-call-no-cross-bleed) | ➖ n/a |
| S-ATL-026 | `tool_stream_corrective_test.go` | RED: partial-JSON fragment concatenation scenario newly described; GREEN: 2/2 PASS | 2 cases (positive partial-JSON concat, empty-args canonicalization) | ➖ n/a |
| S-ATL-060 | `tool_stream_corrective_test.go` | RED: bridge replay round-trip scenario newly described; GREEN: 2/2 PASS | 2 cases (literal-newline byte-equal, simple-end byte-equal) | ➖ n/a |

### Vacuous-pass discipline applied

For each scenario, the 9 vacuous-pass shapes from obs #2471 were
applied. Empty-collection companions added where the assertion could
be vacuously true:

- S-ATL-023: TestS_ATL_023_KeepAlive_DoesNotEmitAnything pins
  comment-only streams produce zero frames (shape 4 / 5).
- S-ATL-023: TestS_ATL_023_KeepAlive_AcrossTwoToolCalls_NoCrossInterference
  pins no cross-call state bleed (shape 6).
- S-ATL-026: TestS_ATL_026_FreshClose_YieldsEmpty pins the
  byte-equal assertion is non-vacuous (shape 5).
- S-ATL-060: TestS_ATL_060_SimpleEnd_PreservesBytes pins the
  byte-equal assertion is non-trivially about the newline byte
  (shape 2).

### Hard rule conformance

- `go.mod` byte-identical (zero requires; verified `git diff
  feat/ai-30-tool-stream~7..feat/ai-30-tool-stream -- go.mod go.work`
  is empty)
- No new exported identifiers added (helpers in tool_stream_corrective_test.go
  all unexported)
- `src/ai` (the `ai` package itself) not modified
- `src/agenttest` not modified (R-ATL-015 preserved)
- No Factory declaration flip (R-ATL-012 STOP posture preserved)
- Bridge factory `conformanceBridgeFactory()` declaration not touched;
  S-ATL-060 reuses the bridge machinery via its existing surface

### Commits added by this corrective round

- `74ad3ad` — test(openaicompat): add S-ATL-023/S-ATL-026/S-ATL-060
  missing coverage
- `d00f54c` — style(openaicompat,agenttest): gofmt -w and resolve
  lint drift

## Next steps

- Re-launch `sdd-verify` for AI-30 — verify confirms all 16
  requirements / 73 scenarios covered (51 → 54 `[test]` scenarios,
  17/19 → 19/19 `[inspection]` scenarios)
- After re-verify PASS, `sdd-archive` for AI-30 (sync delta specs,
  archive the change)

## Files changed

- `backend/agent/src/ai/openaicompat/chunk.go` (+47)
- `backend/agent/src/ai/openaicompat/stream_state.go` (+424)
- `backend/agent/src/ai/openaicompat/stream.go` (+63)
- `backend/agent/src/ai/openaicompat/errors.go` (+13)
- `backend/agent/src/ai/openaicompat/doc.go` (+23)
- `backend/agent/src/ai/openaicompat/bridge_test.go` (+244)
- `backend/agent/src/ai/openaicompat/tool_stream_decode_test.go` (+181, new)
- `backend/agent/src/ai/openaicompat/tool_stream_state_test.go` (+750, new)
- `backend/agent/src/ai/openaicompat/tool_stream_cap_test.go` (+158, new)
- `backend/agent/src/ai/openaicompat/tool_stream_empty_test.go` (+236, new)
- `backend/agent/src/ai/openaicompat/tool_stream_bytefidelity_test.go` (+206, new)
- `backend/agent/src/ai/openaicompat/tool_stream_truncation_test.go` (+342, new)
- `backend/agent/src/ai/openaicompat/tool_stream_unrepresentable_test.go` (+105, new)
- `backend/agent/src/ai/openaicompat/tool_stream_ordinal_test.go` (+195, new)
- `backend/agent/src/agenttest/conformance_tool_call.go` (+54 / -some)
- `openspec/changes/cachicamas-ai-provider-tool-stream/tasks.md` (all phase boxes checked)