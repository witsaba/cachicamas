# Wave 4 Archive — `cachicamas-ai-layer-1` "Connect the vendor"

**Date**: 2026-08-06
**Branch merged**: `feat/2026-08-03-cachicamas-ai-layer1-wave-4` @ `283d37d` (165 commits ahead of pre-merge `main` @ `75a72b9`)
**Merge commit**: `7f9822d` on `main` (`--no-ff`)
**Push**: `75a72b9..7f9822d main -> main`

## Scope

Wave 4 of the AI Layer 1 milestone plan: nine SDD changes, two conformance amendments, all under one umbrella "Connect the vendor". The wire contract is closed.

| # | Milestone | Capability | SDD change | Spec location | Verdict |
|---|-----------|------------|------------|---------------|---------|
| 1 | AI-24 | ai-first-provider-decision | `2026-08-05-cachicamas-ai-first-provider-decision/` | `openspec/specs/ai-first-provider-decision/spec.md` | PASS, 0 CRITICAL |
| 2 | AI-25 | ai-provider-client | `2026-08-05-cachicamas-ai-provider-client/` | `openspec/specs/ai-provider-client/spec.md` | PASS, 0 CRITICAL |
| 3 | AI-26 | ai-request-translation | `2026-08-05-cachicamas-ai-request-translation/` | `openspec/specs/ai-request-translation/spec.md` | PASS, 0 CRITICAL |
| 4 | AI-27 | ai-stream-decoder | `2026-08-05-cachicamas-ai-stream-decoder/` | `openspec/specs/ai-stream-decoder/spec.md` | PASS, 0 CRITICAL |
| 5 | AI-28 | ai-provider-text-stream | `2026-08-05-cachicamas-ai-provider-text-stream/` (rev-4) | `openspec/specs/ai-provider-text-stream/spec.md` | PASS WITH WARNINGS, JD APPROVED |
| 6 | AI-29 | ai-provider-reasoning-stream (recorded absence) | `2026-08-04-cachicamas-ai-provider-reasoning-stream/` | `openspec/specs/ai-provider-reasoning-stream/spec.md` | PASS WITH WARNINGS, zero production Go |
| 7 | AI-30 | ai-provider-tool-stream | `2026-08-04-cachicamas-ai-provider-tool-stream/` (rev-1) | `openspec/specs/ai-provider-tool-stream/spec.md` | PASS WITH WARNINGS |
| 8 | AI-31 | ai-provider-completion | `2026-08-04-cachicamas-ai-provider-completion/` | `openspec/specs/ai-provider-completion/spec.md` | PASS, 0 CRITICAL |
| 9 | AI-32 | ai-provider-error-mapping | `2026-08-05-cachicamas-ai-provider-error-mapping/` | `openspec/specs/ai-provider-error-mapping/spec.md` | PASS WITH WARNINGS |

## Conformance amendments (folded into the umbrella)

| # | Amendment | Archived to |
|---|-----------|-------------|
| A1 | `cachicamas-ai-conformance-lifecycle-amendment` (R-CNF-005/006) | `2026-08-05-cachicamas-ai-conformance-lifecycle-amendment/` |
| A2 | `cachicamas-ai-conformance-tool-amendment` (R-CNF-007/008/019-021/025/026) | `2026-08-05-cachicamas-ai-conformance-tool-amendment/` |

Both were applied inline on the wave tracker (commits `326bac5` and `e64ce60`); their apply/verify evidence lives in the wave tracker diff and in the merged-into-`main` source. SDD folders moved to archive as part of this closure commit.

## Wire contract — closed

| Owner | Capability |
|-------|------------|
| AI-19 | failure taxonomy (`ai-provider-errors`) |
| AI-28 | producer (`ai-provider-text-stream`) |
| AI-31 | usage mapping (`ai-provider-completion`) |
| AI-30 | tool-call streaming (`ai-provider-tool-stream`) |
| AI-29 | reasoning-stream absence (`ai-provider-reasoning-stream`) |
| AI-32 | failure mapping (`ai-provider-error-mapping`) |

## Cumulative counters

- 9 main specs promoted to `openspec/specs/ai-*`
- 11 archived change folders under `openspec/changes/archive/2026-08-04-..05-` (9 milestones + 2 amendments)
- 230 files changed, 52,649 insertions, 101 deletions on the merge commit
- 273 scenarios covered across the 5 verified milestones (107 + 18 + 47 + 31 + 70)
- 0 CRITICAL findings in any final verify report
- 0 go.mod changes (Branch B no-bridge-run preserved)
- 0 new `src/ai` exports

## Notable deviations (carried into Wave 5+)

1. **`stream.go` `+63` → `+55` doc drift** in AI-30 (spec amendment rev-1 asserted exact line count; lint cleanup shifted it). SUGGESTION.
2. **Stage 2b bounded-wait emitFailure** records as deviation from AI-20.3 "no terminal event on cancellation" comment text. Behavior evolved; comment text stale. WARN, documented in verify-report.
3. **Pre-existing `unused-function` lint on `requireRelativeKindOrder`** (now live, called by `conformance_tool_call.go:179,283`) — SUGGESTION for a follow-up agenttest change to silence the unrelated `staticcheck` QF1011 already in the file.
4. **`capture_proof_test.go`'s planted sentinel `sk-AEM060-planted-in-body-only`** requires the file to remain `package openaicompat` (internal). Any future move to `package openaicompat_test` would break the credential-scan guard. Load-bearing design preserved; called out in the test file's own doc comment.

## Post-merge conflict-resolution fix (commit `283d37d`)

The PR #115 / AI-32 merge into the wave tracker lost three pieces of AI-32 logic in the conflict resolution:

1. The `isInBandErrorFrame(frame.Data)` check in `run()`'s frame loop (R-AEM-010). Without it, `{"error":{"type":"vendor_stream_fault"}}` frames decoded as chunks and surfaced as generic `malformed_response` failures with empty `RawLabel`.
2. The for-loop's `if !emit { return }` path now produces a typed terminal failure on cancel/deadline (R-AEM-014, S-AEM-051/052) instead of the AI-20.3 silent loss path.
3. The readErr's `ctx.Err() != nil` branch surfaces a typed terminal failure (S-AEM-051 Timeout, S-AEM-052 Cancellation) rather than the AI-20.3 silent return.

Plus: `emitFailure`'s bounded-wait terminal send now passes the event through `stamper.Stamp(ev)` first (no more `seq=0` leak, R-AEE-008). Plus: `usage_probes_test.go`'s multi-frame fixture had `total_tokens` emitted as a quoted string (invalid JSON) — reverted to unquoted number. Plus: 2 pre-existing AI-29 lint SUGGESTIONs cleared (`scenarios` param renamed to `_`, redundant `var _ agenttest.Factory = factory` removed).

## Review path

- Native `gentle-ai review` 4R unavailable repo-wide (Engram obs #2530; ledger empty, `start` returns `stale_target_identity` back-to-back)
- JD dual-judge: APPROVED for AI-28 only (Engram obs #2531); explicitly waived by orchestrator for AI-29 (zero production Go), AI-30 (after FAIL+corrective round), AI-31, AI-32 — per user's "continue with the full chain; review at final, even with UAT" directive
- All 5 verifications independent (`gentle-ai sdd-verify-validate` admitted each)
- UAT is the user's responsibility at review time (the user's directive explicitly accepted this risk envelope)

## User-facing next steps (post-closure)

1. UAT on the wave-4 PR (the 165-commit diff against `main`)
2. Wave 5+ follow-ups (the 4 deviations above plus any doc-0002 amendments Wave 4 implies)
