# Tasks: AI-31 — ai-provider-completion

## Review Workload Forecast

| Field | Value |
|-------|-------|
| Estimated changed lines | ~150-250/slice (600-900 total) |
| 400-line budget risk | Low (configured PR budget = 5000/PR; well under both) |
| Chained PRs recommended | Yes |
| Suggested split | PR1 (AI-31.1) → PR2 (AI-31.2a) → PR3 (AI-31.2b) → PR4 (AI-31.3) |
| Delivery strategy | auto-chain |
| Chain strategy | feature-branch-chain |

Decision needed before apply: No
Chained PRs recommended: Yes
Chain strategy: feature-branch-chain
400-line budget risk: Low

### Suggested Work Units

| Unit | Goal | PR | Focused test command | Runtime harness | Rollback boundary |
|---|---|---|---|---|---|
| 1 | Finish-reason coverage + all records + 5 doc 0002 sites | PR1, base=tracker | `go test -race -count=1 ./src/ai/openaicompat/... -run FinishReason\|StopSequence` | `go test -race -count=1 ./...` (backend/agent) | revert `finish_reason_test.go`, comment additions, doc 0002 blockquotes |
| 2 | Detail-struct decode, +3 raw mappings | PR2, base=PR1 | `go test -race -count=1 ./src/ai/openaicompat/... -run TestUsage_AllDetailFields\|TestUsage_DetailFieldsAbsent\|TestUsage_CachedTokensZero\|TestUsage_Reasoning` | `go test -race -count=1 ./...` | revert struct+mapping diff in `chunk.go`, new `usage_test.go` cases |
| 3 | Impossible-arithmetic probe + multi-frame pin | PR3, base=PR2 | `go test -race -count=1 ./src/ai/openaicompat/... -run TestUsage_Impossible\|TestUsage_SingleFrame\|TestUsage_MultiplePopulatedFrames` | `go test -race -count=1 ./...` | revert new `usage_test.go` cases only (no production diff expected) |
| 4 | Position/tolerance coverage | PR4, base=PR3 | `go test -race -count=1 ./src/ai/openaicompat/... -run TestUsage_CacheReadPresent\|TestUsage_OddPosition\|TestUsage_MetadataOnly` | `go test -race -count=1 ./...` | revert new `usage_test.go` cases only (coverage-only expected) |

## Phase 0: Branch setup

- [x] 0.1 Verify chain tip ancestry includes `feat/ai-28-8-d8-close-discipline` and `requireCheckStreamClean` exists; branch slice 1 from the current tip (record actual base).

## Slice 1 (PR1, AI-31.1)

- [x] 1.1 RED `openaicompat/finish_reason_test.go`: `TestFinishReason_FiveWireValues_MapTable` (S-ACP-001, row count=5), `TestFinishReason_DeprecatedFunctionCall` + `TestFinishReason_ContentFilterDistinctFromRefusal` (S-ACP-002/003).
- [x] 1.2 RED same file: `TestFinishReason_NeverUnreachable` (S-ACP-005, incl. out-of-enum negative control).
- [x] 1.3 RED `TestFinishReason_NovelValue_TypedMalformed` (S-ACP-006: `"STOP"`, `" stop"`, `"halted"`).
- [x] 1.4 RED `TestStopSequence_NothingIdentifiesMatch` (S-ACP-010, extra-`stop_sequence`-key negative control).
- [x] 1.5 GREEN: confirm 1.1–1.4 pass unmodified (finish mapping already landed, coverage-only); STOP+report if a code change is needed.
- [x] 1.6 Inspection S-ACP-004/007/008: confirm `ai.NormalizeFinishReason` totality/no-panic and the malformed diagnostic chain carries the raw label; `ai.Completion` unchanged.
- [x] 1.7 Records: comment at `rawStrictFinishReason`/`finishReasonEnum` (unreachable table, S-ACP-004); `buildCompletion` U4 comment (S-ACP-009); `bridge_test.go` D3 comment at `RunConformanceFor` ~L198 (S-ACP-026/027, `agenttest` untouched).
- [x] 1.8 Doc 0002: 5 dated blockquotes below original wording — item 1 vacuity, item 2 D1 split, item 3 unexercisable, AI-31.1 Note vacuity, charter Acceptance unsatisfiable-and-unviolated (S-ACP-028/029; cite U5/D2, agree with R-ACP-002 table).
- [x] 1.9 Confirm `specs/ai-provider-text-stream/spec.md` R-ATS-026 delta already authored (S-ATS-098…101); no action if present.
- [x] 1.10 Gate: `go.mod` 0 requires, gofmt (ignore pre-existing `src/ai/completion_test.go`), lint, `go test -race -count=1 ./...` ×1; every new drained-stream test asserts `requireCheckStreamClean`.

## Slice 2 (PR2, AI-31.2a)

- [x] 2.1 RED `usage_test.go`: `TestUsage_AllDetailFields_Present` (S-ACP-011), `TestUsage_DetailFieldsAbsent_NegativeControl` (S-ACP-012).
- [x] 2.2 RED `TestUsage_CachedTokensZeroVsAbsent` (S-ACP-013), `TestUsage_ReasoningContainedInOutput` (S-ACP-014).
- [x] 2.3 GREEN `chunk.go`: add `wirePromptTokensDetails{CachedTokens,CacheWriteTokens *int64}` / `wireCompletionTokensDetails{ReasoningTokens *int64}`; `usageFromWire` +3 raw mappings only, no arithmetic; Input/Output rows untouched.
- [x] 2.4 Add D-C doc comment on `usageFromWire` (quote AI-13.4 Input sentence, cite U1/U2, name AI-38.2) — S-ACP-016.
- [x] 2.5 Frozen gate: diff `usage_test.go` shows S-ATS-055…062 byte-identical (S-ACP-024); confirm they pass (S-ACP-025).
- [x] 2.6 Gate: `go.mod` 0 requires, lint, gofmt, suite ×1; every new test asserts `requireCheckStreamClean`.

## Slice 3 (PR3, AI-31.2b)

- [ ] 3.1 RED `TestUsage_ImpossibleArithmeticUnderExclusivity` (S-ACP-017: `prompt_tokens:500`/`cached_tokens:800` → `Input=500`, `CacheRead=800` raw, no error/clamp).
- [ ] 3.2 RED `TestUsage_SingleFrame_U6Shape` (S-ACP-018).
- [ ] 3.3 RED `TestUsage_MultiplePopulatedFrames_LastWinsNoFold` (S-ACP-019: inline 3-frame `10/20/30` → `Input=30`; the ONE dialect-conventional label pin).
- [ ] 3.4 GREEN: confirm D10 last-wins wholesale-overwrite needs no change; STOP+report if a new mechanism is needed.
- [ ] 3.5 Inspection S-ACP-016/020/030/031: label table has exactly one row naming `TestUsage_MultiplePopulatedFrames_LastWinsNoFold`.
- [ ] 3.6 Gate: `go.mod` 0 requires, lint, gofmt, suite ×1; `requireCheckStreamClean` on every new test.

## Slice 4 (PR4, AI-31.3)

- [ ] 4.1 RED `TestUsage_CacheReadPresentCacheWriteAbsent_NoPanic` (S-ACP-021).
- [ ] 4.2 RED `TestUsage_OddPositionFrame_BeforeTerminalChunk` (S-ACP-022; precondition: non-empty `model` + same `id` on the odd-position frame).
- [ ] 4.3 RED `TestUsage_MetadataOnlyFrame_ZeroContentEvents` (S-ACP-023; the explicit `requireCheckStreamClean` assertion IS the scenario).
- [ ] 4.4 GREEN: expect coverage-only (spec-risk-3 resolved); an unexpected RED needing a code change is new information — STOP+report.
- [ ] 4.5 Gate: `go.mod` 0 requires, lint, gofmt, full suite ×2 (close-of-change).

Note: first commit banks THIS change's planning artifacts only (`tasks.md` + Engram twin) — never `git add -A`.
