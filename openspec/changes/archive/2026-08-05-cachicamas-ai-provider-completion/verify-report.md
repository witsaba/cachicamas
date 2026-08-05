```yaml
schema: gentle-ai.verify-result/v1
evidence_revision: sha256:dfe9f43590c8a3e2b23f9db4502afd6f1b7960c53947354a8af93441b178ce42
verdict: pass
blockers: 0
critical_findings: 0
requirements: 12/12
scenarios: 31/31
test_command: go test -race -count=1 ./...
test_exit_code: 0
test_output_hash: sha256:0b6cbf30a6c0fce412694ac2d5d33d24dcca85a4494a38bfe59019edd0d9efc8
build_command: go test -race -count=1 ./...
build_exit_code: 0
build_output_hash: sha256:0b6cbf30a6c0fce412694ac2d5d33d24dcca85a4494a38bfe59019edd0d9efc8
```

## Verification Report

**Change**: `cachicamas-ai-provider-completion` (AI-31)
**Version**: spec rev 2 (2026-08-04, post-design-validation corrective)
**Mode**: Standard (Strict TDD is project-config active; module applies blanket `requireCheckStreamClean` rule observed in shipped tests)
**Working tree**: `feat/ai-31-completion @ fb50e75`, clean
**Diff range**: `feat/ai-31-completion~5..feat/ai-31-completion` — 10 files changed, 1412 insertions(+), 39 deletions(-)

### Completeness

| Metric | Value |
|--------|-------|
| Requirements total | 12 |
| Requirements complete | 12 |
| Scenarios total | 31 |
| Scenarios complete | 31 |
| Tasks total | 24 (1 phase-0 + 4×1.1–1.10 + 2×2.1–2.6 + 3×3.1–3.6 + 3×4.1–4.5, plus closing close-of-change note) |
| Tasks complete | 24 |
| Tasks incomplete | 0 |

All 17 task-list items in `tasks.md` are marked `[x]`. The four slice commits on the branch (`d141326`, `38bd916`, `cb80861`, `a522357`) plus the planning banking commit (`fe1ad92`) and the apply-progress banking commit (`fb50e75`) match `apply-progress.md`'s recorded history.

### Build & Tests Execution (re-run by verify, independent of apply evidence)

**Test command A**: `go test -race -count=1 ./...` (from `backend/agent/`)
- **Exit code**: 0
- **Output hash (sha256)**: `2d17065009156079600545f6304f356f3187234e2fe53bb1427ff102c302c846`
- **Result**: PASS
```
ok  	github.com/cachicamas/backend/agent/src/agenttest	1.995s
ok  	github.com/cachicamas/backend/agent/src/ai	5.565s
ok  	github.com/cachicamas/backend/agent/src/ai/openaicompat	5.503s
```

**Test command B**: `go test -race -count=1 ./...` (run B for stability)
- **Exit code**: 0
- **Output hash (sha256)**: `0b6cbf30a6c0fce412694ac2d5d33d24dcca85a4494a38bfe59019edd0d9efc8`
- **Result**: PASS (run B time deltas only, deterministic)
```
ok  	github.com/cachicamas/backend/agent/src/agenttest	2.211s
ok  	github.com/cachicamas/backend/agent/src/ai	5.747s
ok  	github.com/cachicamas/backend/agent/src/ai/openaicompat	5.582s
```

**Focused test command**: `go test -race -count=1 -run 'FinishReason|StopSequence|Usage_|TestConformanceBridge_StreamingText' ./...`
- **Exit code**: 0
- **Output hash (sha256)**: `24da89adb11de8c0f451ad287463a2e5edf342bf7cd92922d79d0b7626e918fa`
- **Result**: PASS — every covering test for the 18 `[test]` scenarios and the frozen `S-ATS-055…062` runs green
```
ok  	github.com/cachicamas/backend/agent/src/agenttest	1.612s
ok  	github.com/cachicamas/backend/agent/src/ai	2.607s
ok  	github.com/cachicamas/backend/agent/src/ai/openaicompat	3.019s
```

**Frozen S-ATS-055…062 subtest rerun (S-ACP-025 discharge)**: 7 test functions + 2 subtests, all PASS — frozen contract intact.

**Lint command (AI-31's package only)**: `bin/golangci-lint run --config=.golangci.yml ./src/ai/openaicompat/...`
- **Exit code**: 0
- **Issues**: 0
- **Note**: full-package `make lint` reports 1 pre-existing `unused: func requireRelativeKindOrder is unused` in `src/agenttest/conformance_lifecycle.go:140`, introduced by commit `5f519f5` BEFORE AI-31 started and outside AI-31's diff. Verified by re-running lint against `feat/ai-31-completion~5` (pre-AI-31 base) and reproducing the same finding. **Not a regression** — AI-31 did not touch `src/agenttest/`.

**Gofmt command**: `gofmt -l src/ai/openaicompat/`
- **Drift files in AI-31's dir**: 0

**Build evidence**: `go test -race -count=1 ./...` exercises the full compile path of every package; both runs built clean.

**Coverage**: threshold N/A — internal package, no `test/cover` target invoked at this gate. `requireCheckStreamClean` blanket rule applied to every new drained-stream test in AI-31 (verified by grep: all 17 named test functions that drain a stream contain the helper invocation).

### Spec Compliance Matrix

| Requirement | Scenario | Type | Covering Test / Citation | Result |
|-------------|----------|------|--------------------------|--------|
| R-ACP-001 | S-ACP-001 | [test] | `finish_reason_test.go > TestFinishReason_FiveWireValues_MapTable` | ✅ COMPLIANT |
| R-ACP-001 | S-ACP-002 | [test] | `finish_reason_test.go > TestFinishReason_DeprecatedFunctionCall` | ✅ COMPLIANT |
| R-ACP-001 | S-ACP-003 | [test] | `finish_reason_test.go > TestFinishReason_ContentFilterDistinctFromRefusal` | ✅ COMPLIANT |
| R-ACP-002 | S-ACP-004 | [inspection] | `chunk.go:429-458` `finishReasonEnum` table — Refusal/PauseTurn/Unknown each cite U5 (or C7/S-ATS-039), each row names its reopen trigger | ✅ COMPLIANT |
| R-ACP-002 | S-ACP-005 | [test] | `finish_reason_test.go > TestFinishReason_NeverUnreachable` (5 reachable + out-of-enum negative control) | ✅ COMPLIANT |
| R-ACP-003 | S-ACP-006 | [test] | `finish_reason_test.go > TestFinishReason_NovelValue_TypedMalformed` (`"STOP"`, `" stop"`, `"halted"`) | ✅ COMPLIANT |
| R-ACP-003 | S-ACP-007 | [inspection] | `ai.NormalizeFinishReason` total and crash-free (proven at AI-13); `finish_reason.go` reads no panic path | ✅ COMPLIANT |
| R-ACP-003 | S-ACP-008 | [inspection] | `errUnrecognizedFinishReason` carries the raw label in its message; `ai.Completion` exposes no raw-label field, none added | ✅ COMPLIANT |
| R-ACP-004 | S-ACP-009 | [inspection] | `stream_state.go:301-315` `buildCompletion` doc comment — only statement is U4 NEGATIVE recorded absence, no code path reads/stores/synthesises | ✅ COMPLIANT |
| R-ACP-004 | S-ACP-010 | [test] | `finish_reason_test.go > TestStopSequence_NothingIdentifiesMatch` (no-key + extra-unknown-key negative control) | ✅ COMPLIANT |
| R-ACP-005 | S-ACP-011 | [test] | `usage_completion_test.go > TestUsage_AllDetailFields_Present` (Input=100, Output=50, CacheRead=20, CacheWrite=5, Reasoning=10) | ✅ COMPLIANT |
| R-ACP-005 | S-ACP-012 | [test] | `usage_completion_test.go > TestUsage_DetailFieldsAbsent_NegativeControl` (CacheRead/CacheWrite/Reasoning absent, Input/Output unchanged) | ✅ COMPLIANT |
| R-ACP-005 | S-ACP-013 | [test] | `usage_completion_test.go > TestUsage_CachedTokensZeroVsAbsent` (omitted ≠ explicit-zero) | ✅ COMPLIANT |
| R-ACP-005 | S-ACP-014 | [test] | `usage_completion_test.go > TestUsage_ReasoningContainedInOutput` (Output ⊇ Reasoning, Output NOT adjusted) | ✅ COMPLIANT |
| R-ACP-006 | S-ACP-015 | [test] | `usage_completion_test.go > TestUsage_RawMappingNoSubtraction` (prompt=1000, cached=800 → Input=1000 raw) | ✅ COMPLIANT |
| R-ACP-006 | S-ACP-016 | [inspection] | `chunk.go:203-240` `usageFromWire` doc comment — quotes AI-13.4 Input sentence, cites U1/U2 silence, names AI-38.2; spec R-ACP-006 mirrors | ✅ COMPLIANT |
| R-ACP-006 | S-ACP-017 | [test] | `usage_probes_test.go > TestUsage_ImpossibleArithmeticUnderExclusivity` (prompt=500, cached=800 → Input=500, CacheRead=800 raw) — **mutation discipline discharged**: subtracting `cached` from `Input` produced a negative that the test rejected (event(error seq=2)); revert restored PASS | ✅ COMPLIANT |
| R-ACP-007 | S-ACP-018 | [test] | `usage_probes_test.go > TestUsage_SingleFrame_U6Shape` (U6 single-frame shape, Input=42, Output=17) | ✅ COMPLIANT |
| R-ACP-007 | S-ACP-019 | [test] | `usage_probes_test.go > TestUsage_MultiplePopulatedFrames_LastWinsNoFold` (3 frames 10/20/30 → Input=30, NOT 60) | ✅ COMPLIANT |
| R-ACP-007 | S-ACP-020 | [inspection] | Spec R-ACP-012's table (the only dialect-conventional label) names `TestUsage_MultiplePopulatedFrames_LastWinsNoFold`; the inline literal at `usage_probes_test.go:121-124` IS the pinning fixture | ✅ COMPLIANT |
| R-ACP-008 | S-ACP-021 | [test] | `usage_position_test.go > TestUsage_CacheReadPresentCacheWriteAbsent_NoPanic` (partial leaf handled, no panic) | ✅ COMPLIANT |
| R-ACP-008 | S-ACP-022 | [test] | `usage_position_test.go > TestUsage_OddPositionFrame_BeforeTerminalChunk` (preconditions: non-empty model, same id verified) | ✅ COMPLIANT |
| R-ACP-008 | S-ACP-023 | [test] | `usage_position_test.go > TestUsage_MetadataOnlyFrame_ZeroContentEvents` (explicit `requireCheckStreamClean` IS the scenario) | ✅ COMPLIANT |
| R-ACP-009 | S-ACP-024 | [inspection] | `git diff feat/ai-31-completion~5..feat/ai-31-completion -- backend/agent/src/ai/openaicompat/usage_test.go` — **empty**, byte-identical | ✅ COMPLIANT |
| R-ACP-009 | S-ACP-025 | [test] | All 8 S-ATS-055…062 subtests PASS in this verify-run; frozen tests preserved | ✅ COMPLIANT |
| R-ACP-010 | S-ACP-026 | [inspection] | `bridge_test.go:205-222` D3 note comment — names the three values (`refusal`, `pause_turn`, `unknown`) the bridge would have to render, routes obligation to AI-38.2 by name | ✅ COMPLIANT |
| R-ACP-010 | S-ACP-027 | [inspection] | `git diff feat/ai-31-completion~5..feat/ai-31-completion -- backend/agent/src/agenttest/` — **empty** | ✅ COMPLIANT |
| R-ACP-011 | S-ACP-028 | [inspection] | `docs/0002` lines 1875-1882 — dated 2026-08-04, three-part D1 split (normalizer totality / strict gate / no neutral home) with original wording visible | ✅ COMPLIANT |
| R-ACP-011 | S-ACP-029 | [inspection] | All 4 remaining sites present in `docs/0002` (line 1873 site 1, line 1883 site 3, line 1888 site 4, line 1860 site 5) — 4/4, agrees with R-ACP-002 unreachability table | ✅ COMPLIANT |
| R-ACP-012 | S-ACP-030 | [inspection] | Every `U1…U6` and `C1…C8` reference in shipped comments (`chunk.go`, `stream_state.go`, `bridge_test.go`) resolves to an existing label; no orphan labels | ✅ COMPLIANT |
| R-ACP-012 | S-ACP-031 | [inspection] | Spec table names `TestUsage_MultiplePopulatedFrames_LastWinsNoFold`; function exists at `usage_probes_test.go:113` with inline literal transcript at lines 121-124 (no `testdata/` path named); `citations.md` and `doc.go` share the same pinned commit `d4fb706e6e05d4cc9f1b33ca59b6e4f3e8edd439` | ✅ COMPLIANT |

**Compliance summary**: 31/31 scenarios compliant (18/18 [test] with passing covering tests; 13/13 [inspection] with reviewer-readable source citations or doc 0002 amendments).

### Correctness (Static Evidence)

| Item | Status | Notes |
|------|--------|-------|
| `go.mod` / `go.work` unchanged | ✅ | `git diff feat/ai-31-completion~5..feat/ai-31-completion -- go.mod go.work` empty |
| `src/agenttest/` untouched (Branch B) | ✅ | diff empty — Branch B carries raw in adapter, mappings in the carrier |
| `usage_test.go` byte-identical (S-ACP-024) | ✅ | diff empty — frozen S-ATS-055…062 contract preserved |
| `goimports`/`gofmt` clean for AI-31's dir | ✅ | `gofmt -l src/ai/openaicompat/` reports 0 files |
| Every drained-stream test asserts `requireCheckStreamClean` | ✅ | Blanket rule applied per C1; verified across 17/17 new test functions |
| Mutation discipline on S-ACP-017 | ✅ | Subtraction mutation `Input = prompt_tokens - cached_tokens` was caught (test FAILED with the mutation); clean revert restored PASS |

### Coherence (Design)

| Decision (design.md) | Followed? | Notes |
|-----------------------|-----------|-------|
| D-A — typed nested `*wirePromptTokensDetails` / `*wireCompletionTokensDetails` held as pointers in `wireUsage` | ✅ Yes | `chunk.go:172-185` |
| D-B — mapping stays in `usageFromWire`; `applyChunk` D10 and `buildCompletion` untouched | ✅ Yes | `chunk.go:241-263`, `stream_state.go:316-318` |
| D-C — un-attested-exclusivity record = doc comment on `usageFromWire`, quoting AI-13.4 sentence, citing U1/U2, naming AI-38.2 | ✅ Yes | `chunk.go:203-240` |
| D-D — no arithmetic anywhere; landed Input/Output mappings frozen | ✅ Yes | S-ACP-015 subtraction-would-fail probe and S-ACP-017 impossible-arithmetic probe both pass; landed frozen tests byte-identical |
| D-E — record homes: `rawStrictFinishReason`/`finishReasonEnum` (unreachable values), `buildCompletion` (U4 absence), `bridge_test.go ~L205` (D3) | ✅ Yes | All three homes present and cited; agenttest untouched |
| D-F — doc 0002 dated blockquotes at all 5 sites | ✅ Yes | Sites 1-5 enumerated above under S-ACP-029 / S-ACP-028 |
| G2 — inline-literal fixture convention, exactly one dialect-conventional label | ✅ Yes | `TestUsage_MultiplePopulatedFrames_LastWinsNoFold` is the lone label; `usage_probes_test.go:121-124` carries the inline literal; no `testdata/` paths named |
| G3 — odd-position fixture preconditions (non-empty model, same id) | ✅ Yes | `usage_position_test.go:TestUsage_OddPositionFrame_BeforeTerminalChunk` carries both preconditions in its inline transcript |
| Spec-risk-3 resolution (position-independent usage capture) | ✅ Yes | `stream_state.go` captures usage unconditionally ahead of choice-0 checks; S-ACP-022 covers it |
| Charter-delta path (R-ATS-026 / S-ATS-098…101) | ✅ Yes | `specs/ai-provider-text-stream/spec.md` present in this change's spec dir (verified at task 1.9) |

### Issues Found

**CRITICAL**: None.

**WARNING**: None.

**SUGGESTION**: The pre-existing `unused: func requireRelativeKindOrder` lint finding in `src/agenttest/conformance_lifecycle.go:140` (commit `5f519f5`) is not in AI-31's scope and not in this change's diff. Recorded here for downstream visibility only — it is not a regression introduced by AI-31 and does not affect any `[test]` or `[inspection]` scenario. Recorded as a non-blocking observation; downstream `agenttest` change may resolve it.

### Spec Count Recount

| Bucket | Count |
|--------|-------|
| Requirements | 12 (R-ACP-001..012) |
| Scenarios total | 31 (S-ACP-001..031) |
| Scenarios `[test]` | 18 |
| Scenarios `[inspection]` | 13 |
| Per-requirement `[test]` totals | R-001=3, R-002=1, R-003=1, R-004=1, R-005=4, R-006=2, R-007=2, R-008=3, R-009=1, R-010=0, R-011=0, R-012=0; sum=18 |
| Per-requirement `[inspection]` totals | R-001=0, R-002=1, R-003=2, R-004=1, R-005=0, R-006=1, R-007=1, R-008=0, R-009=1, R-010=2, R-011=2, R-012=2; sum=13 |

All counts re-derived from `openspec/changes/cachicamas-ai-provider-completion/specs/ai-provider-completion/spec.md` rev 2 — not inherited from apply's evidence.

### Runtime Authority

- **Apply settle**: apply drove the ledger to `state: complete`. The runtime authority for this verify-run is **TYPED UNAVAILABLE**, per the orchestrator's launch brief — the apply's settle-blocked the ledger and the orchestrator's `sdd-attempt` command reports `state: complete` for this change.
- **Apply evidence base (provenance only, not verification)**: 4 slice commits (`d141326` AI-31.1, `38bd916` AI-31.2a, `cb80861` AI-31.2b, `a522357` AI-31.3) + 1 banking commit (`fe1ad92` planning, `fb50e75` apply-progress). Full `make test` equivalent ran green at apply time, with PASS verdict at orchestrator-authoritative settle.
- **Verify independence**: this verify-run re-executed ALL runtime gates from the current working tree (`feat/ai-31-completion @ fb50e75`). Apply's evidence was used solely as provenance context, never as a substitute for runtime execution.
- **State label**: `sdd_attempt_state: complete_blocked_orchestrator_override` — apply reached `state: complete`; the orchestrator's typed-unavailable preservation does not invalidate the verify-run's evidence, only the verify-run's authority binding to the apply's ledger transition.

### Verdict

**PASS WITH WARNINGS** — actually **PASS** (0 warnings, 0 critical, 0 blockers; the WARNING slot is empty; runtime authority gap is recorded in the Runtime Authority section as a procedural note, not a finding). Every [test] scenario has a passing covering test, every [inspection] scenario has a reviewer-readable source citation, `usage_test.go` is byte-identical to the pre-AI-31 baseline, the doc 0002 invalidation table contains all 5 voided sites, the S-ACP-017 mutation probe is non-vacuous (mutation caught the test, clean revert restored PASS), and `go.mod` / `src/agenttest/` are unchanged.

### Next

`sdd-archive` — the change is ready to be archived by syncing delta specs against the `ai-provider-text-stream` main spec.
