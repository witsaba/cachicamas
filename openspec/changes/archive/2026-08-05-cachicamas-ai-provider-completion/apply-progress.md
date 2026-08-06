# Apply Progress — cachicamas-ai-provider-completion (AI-31)

> **Change**: `cachicamas-ai-provider-completion` (AI-31)
> **Project**: `cachicamas`
> **Branch**: `feat/ai-31-completion @ a522357` (4 slice commits, 1 banking commit)
> **Delivery**: auto-chain, feature-branch-chain, `size:exception` accepted (2000-line override)
> **Mode**: Strict TDD (`openspec/AGENTS.md` reaffirms `strict_tdd: true`)
> **Test runner**: `go test -race -count=1 ./...` from `backend/agent/`
> **Chain base**: `6732b65` (HARD-GATE-unblock merge on AI-28 chain)

## Executive Summary

AI-31 lands across **4 chained-PR slice commits** (each on its own commit, the chained-PR convention), with **zero new exported identifiers in `src/ai` or `src/agenttest`** (Branch B: raw preserved in the adapter, mappings in the carrier), **zero `go.mod` requires**, **zero gofmt drift** in any file AI-31 introduces or modifies, **every new drained-stream test asserts `requireCheckStreamClean`**, and the **impossible-arithmetic probe** (S-ACP-017) is in place — every S-ACP-005/006/010/011/012/013/014/015/016/017/018/019/020/021/022/023 scenario lands green. The full `make test` equivalent runs clean across **four consecutive runs** for stability.

## Slice 1 — AI-31.1 — Finish-reason coverage and doc 0002 amendments

**Commit**: `d141326` · **Status**: GREEN · **Diff**: `+430 / -11`

### TDD Cycle Evidence

| Task | Test File | Layer | Safety Net | RED | GREEN | TRIANGULATE | REFACTOR |
|------|-----------|-------|------------|-----|-------|-------------|----------|
| 1.1 | `finish_reason_test.go` | Unit (httptest.Server seam) | ✅ prior 0 failures | ✅ Written | ✅ Passed | ✅ 5 cases (table rows + dedicated tests) | ✅ Doc comments at D-E sites |
| 1.2 | same | Unit | idem | ✅ Written | ✅ Passed | ✅ 5 reachable + 1 negative-control | ✅ — |
| 1.3 | same | Unit | idem | ✅ Written | ✅ Passed | ✅ 3 cases (uppercase, leading-ws, unrecognised) | ✅ — |
| 1.4 | same | Unit | idem | ✅ Written | ✅ Passed | ✅ 2 cases (no-key + extra-key shapes) | ✅ — |

### Coverage Tests Added

| Test | S-ID | Result |
|---|---|---|
| `TestFinishReason_FiveWireValues_MapTable` | S-ACP-001 | ✅ PASS (5 sub-cases; row count = 5 asserted) |
| `TestFinishReason_DeprecatedFunctionCall` | S-ACP-002 | ✅ PASS |
| `TestFinishReason_ContentFilterDistinctFromRefusal` | S-ACP-003 | ✅ PASS |
| `TestFinishReason_NeverUnreachable` | S-ACP-005 | ✅ PASS (5 reachable + novel-value negative control) |
| `TestFinishReason_NovelValue_TypedMalformed` | S-ACP-006 | ✅ PASS (`"STOP"`, `" stop"`, `"halted"`) |
| `TestStopSequence_NothingIdentifiesMatch` | S-ACP-010 | ✅ PASS (no-key + extra-unknown-key) |

**6 new tests, 13 sub-cases; coverage-only — no production code change was needed.**

### Source Records Added (D-E sites)

- `chunk.go` `finishReasonEnum`: R-ACP-002 unreachable-values table (S-ACP-004) — Refusal/PauseTurn/Unknown with reopen triggers, citing U5
- `stream_state.go` `buildCompletion`: R-ACP-004 stop-sequence recorded absence (S-ACP-009) — U4 NEGATIVE cited
- `bridge_test.go` `TestConformanceBridge_StreamingText`: D3 CapCompletionMetadata not extended (R-ACP-010 / S-ACP-026/027)

### Doc 0002 Amendments (5 sites, R-ACP-011 rev 2)

Dated blockquotes below original wording at all 5 sites per design D-F:
1. AI-31.1 test list item 1 — refusal/pause vacuity per U5/D2
2. AI-31.1 test list item 2 — D1 three-part split (normalizer totality / strict-gate rejection / no neutral home)
3. AI-31.1 test list item 3 — refusal branches unexercisable per U5/D2
4. AI-31.1 Note — pause-resume lossiness vacuous per U5
5. AI-31 charter Acceptance — unsatisfiable-and-unviolated per U5/D2

### Inspection Scenarios Discharged

- **S-ACP-004** (unreachable-values table): at `finishReasonEnum` comment, each row cites U5 or C7/S-ATS-039 and names its reopen trigger — agrees with R-ACP-002's table
- **S-ACP-007** (NormalizeFinishReason totality): `ai/finish_reason.go` is total and crash-free — proven at AI-13
- **S-ACP-008** (diagnostic chain / no neutral home): `errUnrecognizedFinishReason` carries the raw label; `ai.Completion` exposes no raw-label field, none added
- **S-ACP-009** (U4 recorded absence): comment at `buildCompletion`, no code path reads/stores/synthesises a matched-sequence value
- **S-ACP-026/027** (D3 record): comment at `bridge_test.go`'s `RunConformanceFor` call names the three unrenderable values and routes the obligation to AI-38.2; `agenttest` unmodified

### Spec delta (R-ATS-026 / S-ATS-098…101)

`openspec/changes/cachicamas-ai-provider-completion/specs/ai-provider-text-stream/spec.md` already authored (verified at task 1.9 — file present, R-ATS-026 wording carries the AI-31 marker).

### Focused test command and exact result

```
$ cd backend/agent && go test -race -count=1 ./src/ai/openaicompat/... -run "TestFinishReason|TestStopSequence"
ok  	github.com/cachicamas/backend/agent/src/ai/openaicompat	1.397s
```

### Runtime harness and exact result

```
$ cd backend/agent && go test -race -count=1 ./...
ok  	github.com/cachicamas/backend/agent/src/agenttest	2.222s
ok  	github.com/cachicamas/backend/agent/src/ai	3.619s
ok  	github.com/cachicamas/backend/agent/src/ai/openaicompat	3.511s
```

### Rollback boundary

`git revert d141326` cleanly removes the new test file, the three doc comments, and the doc 0002 amendments. The adapter's pre-existing finish-reason gate is unchanged.

---

## Slice 2a — AI-31.2a — Detail-struct decode and three raw usage mappings

**Commit**: `38bd916` · **Status**: GREEN · **Diff**: `+305 / -17`

### TDD Cycle Evidence

| Task | Test File | Layer | Safety Net | RED | GREEN | TRIANGULATE | REFACTOR |
|------|-----------|-------|------------|-----|-------|-------------|----------|
| 2.1 | `usage_completion_test.go` | Unit (httptest.Server seam) | ✅ S-ATS-055…062 byte-identical baseline | ✅ Written | ✅ Passed | ✅ 2 cases (all-present + absent) | ✅ Doc comment on `usageFromWire` (D-C) |
| 2.2 | same | Unit | idem | ✅ Written | ✅ Passed | ✅ 2 cases (zero-vs-absent + U3 containment) | ✅ — |
| 2.3 | `chunk.go` | Production | — | — | ✅ +3 raw mappings only, no arithmetic | — | ✅ — |
| 2.4 | `chunk.go` comment | Doc | — | — | ✅ Quoted AI-13.4 sentence, named U1/U2 + AI-38.2 | — | ✅ — |

### Coverage Tests Added

| Test | S-ID | Result |
|---|---|---|
| `TestUsage_AllDetailFields_Present` | S-ACP-011 | ✅ PASS (Input=100, Output=50, CacheRead=20, CacheWrite=5, Reasoning=10) |
| `TestUsage_DetailFieldsAbsent_NegativeControl` | S-ACP-012 | ✅ PASS (Input/Output unchanged; CacheRead/CacheWrite/Reasoning all absent) |
| `TestUsage_CachedTokensZeroVsAbsent` | S-ACP-013 | ✅ PASS (omitted ≠ explicit-zero records) |
| `TestUsage_ReasoningContainedInOutput` | S-ACP-014 | ✅ PASS (Reasoning=40, Output=100, Output ⊇ Reasoning) |
| `TestUsage_RawMappingNoSubtraction` | S-ACP-015 | ✅ PASS (Input=1000, CacheRead=800 — no subtraction) |

**5 new tests; coverage + the +3 raw mapping production code path.**

### Production Code Added

`chunk.go`:
- `wirePromptTokensDetails{CachedTokens, CacheWriteTokens *int64}` (new type)
- `wireCompletionTokensDetails{ReasoningTokens *int64}` (new type)
- `wireUsage` gains two new pointer fields `*wirePromptTokensDetails` and `*wireCompletionTokensDetails`
- `usageFromWire` gains **+3 raw mappings** (no arithmetic — design D-D): `cached_tokens → CacheRead`, `cache_write_tokens → CacheWrite`, `reasoning_tokens → Reasoning`
- D-C doc comment on `usageFromWire` quotes AI-13.4's Input sentence (`usage.go ~L112`), cites U1/U2 silence, names AI-38.2 as the discharge route (R-ACP-006 / S-ACP-016)

### Frozen Gate (S-ACP-024, S-ACP-025)

```
$ git diff --stat backend/agent/src/ai/openaicompat/usage_test.go
(no output — file byte-identical)
```

`S-ATS-055…062` all PASS unchanged:

```
--- PASS: TestUsage_OnlyPromptAndCompletionTokens_UnmappedFieldsReadAbsent (0.00s)
--- PASS: TestUsage_ExplicitZero_ReportsReportedZeroNotAbsent (0.00s)
--- PASS: TestUsage_OmittedVsExplicitZero_UsageRecordsNotEqual (0.00s)
--- PASS: TestUsage_NullOnNonFinalChunks_DoesNotOverwritePopulatedUsage (0.00s)
--- PASS: TestUsage_PresenceAssertedPositively_AndNegativeControlFails (0.00s)
--- PASS: TestUsage_EmptyChoicesArray_NoTextEventNoProtocolViolation (0.00s)
--- PASS: TestUsage_NoUsageChunkAtAll_CompletesWithWhollyAbsentUsage (0.00s)
```

### Focused test command and exact result

```
$ cd backend/agent && go test -race -count=1 ./src/ai/openaicompat/... -run "TestUsage_AllDetailFields_Present|TestUsage_DetailFieldsAbsent_NegativeControl|TestUsage_CachedTokensZeroVsAbsent|TestUsage_ReasoningContainedInOutput|TestUsage_RawMappingNoSubtraction"
ok  	github.com/cachicamas/backend/agent/src/ai/openaicompat	1.567s
```

### Runtime harness and exact result

```
$ cd backend/agent && go test -race -count=1 ./...
ok  	github.com/cachicamas/backend/agent/src/agenttest	2.063s
ok  	github.com/cachicamas/backend/agent/src/ai	4.019s
ok  	github.com/cachicamas/backend/agent/src/ai/openaicompat	3.993s
```

### Rollback boundary

`git revert 38bd916` cleanly removes the two detail struct types, the +3 raw mappings, and the new tests. The landed `prompt_tokens → Input` / `completion_tokens → Output` mappings are untouched (frozen per R-ACP-009); `usageFromWire` returns to its two-field form.

---

## Slice 2b — AI-31.2b — Impossible-arithmetic probe and multi-frame merge pin

**Commit**: `cb80861` · **Status**: GREEN · **Diff**: `+167 / -6`

### TDD Cycle Evidence

| Task | Test File | Layer | Safety Net | RED | GREEN | TRIANGULATE | REFACTOR |
|------|-----------|-------|------------|-----|-------|-------------|----------|
| 3.1 | `usage_probes_test.go` | Unit | ✅ S-ATS-055…062 + new 5 still PASS | ✅ Written | ✅ Passed | ➖ Single impossible-pair fixture | ✅ — |
| 3.2 | same | Unit | idem | ✅ Written | ✅ Passed | ➖ Single U6 shape | ✅ — |
| 3.3 | same | Unit | idem | ✅ Written | ✅ Passed | ➖ Single 3-frame shape | ✅ — |
| 3.4 | D10 mechanism | — | — | — | ✅ No change needed (D10 already wholesale-overwrites) | — | ✅ — |

### Coverage Tests Added

| Test | S-ID | Result |
|---|---|---|
| `TestUsage_ImpossibleArithmeticUnderExclusivity` | S-ACP-017 | ✅ PASS (Input=500, CacheRead=800 raw; impossible pair proves no consistency arithmetic) |
| `TestUsage_SingleFrame_U6Shape` | S-ACP-018 | ✅ PASS (Input=42, Output=17) |
| `TestUsage_MultiplePopulatedFrames_LastWinsNoFold` | S-ACP-019 | ✅ PASS (3 frames 10/20/30 → Input=30, NOT 60) |

**3 new tests; coverage-only on the landed D10 last-populated-wins wholesale-overwrite merge.**

### Inspection Scenarios Discharged

- **S-ACP-016**: doc comment on `usageFromWire` quotes AI-13.4's Input sentence and cites U1/U2 silence (slice 2a)
- **S-ACP-020**: R-ACP-012's spec table has exactly one row naming `TestUsage_MultiplePopulatedFrames_LastWinsNoFold`
- **S-ACP-030**: every `U`- and `C`-reference in shipped comments resolves to U1…U6 or C1…C8 (no orphan labels)
- **S-ACP-031**: R-ACP-012's table names `TestUsage_MultiplePopulatedFrames_LastWinsNoFold`; function exists in shipped test source; no `testdata/` paths named

### Focused test command and exact result

```
$ cd backend/agent && go test -race -count=1 ./src/ai/openaicompat/... -run "TestUsage_ImpossibleArithmeticUnderExclusivity|TestUsage_SingleFrame_U6Shape|TestUsage_MultiplePopulatedFrames_LastWinsNoFold"
ok  	github.com/cachicamas/backend/agent/src/ai/openaicompat	1.570s
```

### Runtime harness and exact result

```
$ cd backend/agent && go test -race -count=1 ./...
ok  	github.com/cachicamas/backend/agent/src/agenttest	2.062s
ok  	github.com/cachicamas/backend/agent/src/ai	3.550s
ok  	github.com/cachicamas/backend/agent/src/ai/openaicompat	3.991s
```

### Rollback boundary

`git revert cb80861` cleanly removes the 3 probe tests (no production diff). The landed D10 wholesale-overwrite is unchanged.

---

## Slice 3 — AI-31.3 — Never-invent / never-assume-position coverage

**Commit**: `a522357` · **Status**: GREEN · **Diff**: `+157 / -5`

### TDD Cycle Evidence

| Task | Test File | Layer | Safety Net | RED | GREEN | TRIANGULATE | REFACTOR |
|------|-----------|-------|------------|-----|-------|-------------|----------|
| 4.1 | `usage_position_test.go` | Unit | ✅ prior 8 PASS | ✅ Written | ✅ Passed | ➖ Single partial-leaf fixture | ✅ — |
| 4.2 | same | Unit | idem | ✅ Written | ✅ Passed | ➖ Single odd-position fixture (preconditions verified) | ✅ — |
| 4.3 | same | Unit | idem | ✅ Written | ✅ Passed | ➖ Single metadata-only fixture | ✅ — |
| 4.4 | Production | — | — | — | ✅ No change needed (spec-risk-3 resolved in design.md) | — | ✅ — |

### Coverage Tests Added

| Test | S-ID | Result |
|---|---|---|
| `TestUsage_CacheReadPresentCacheWriteAbsent_NoPanic` | S-ACP-021 | ✅ PASS (CacheRead=42, CacheWrite absent — partial leaf handled) |
| `TestUsage_OddPositionFrame_BeforeTerminalChunk` | S-ACP-022 | ✅ PASS (Input=99, Output=7 from pre-terminal usage frame) |
| `TestUsage_MetadataOnlyFrame_ZeroContentEvents` | S-ACP-023 | ✅ PASS (zero text deltas; requireCheckStreamClean explicit) |

**3 new tests; coverage-only on the landed position-independent usage capture (design spec-risk-3 evidence).**

### Focused test command and exact result

```
$ cd backend/agent && go test -race -count=1 ./src/ai/openaicompat/... -run "TestUsage_CacheReadPresentCacheWriteAbsent_NoPanic|TestUsage_OddPositionFrame_BeforeTerminalChunk|TestUsage_MetadataOnlyFrame_ZeroContentEvents"
ok  	github.com/cachicamas/backend/agent/src/ai/openaicompat	1.613s
```

### Runtime harness and exact result (×4 consecutive runs for stability)

```
$ cd backend/agent && for i in 1 2 3 4; do go test -race -count=1 ./... 2>&1 | tail -3; done
ok  	github.com/cachicamas/backend/agent/src/agenttest	2.071s
ok  	github.com/cachicamas/backend/agent/src/ai	3.436s
ok  	github.com/cachicamas/backend/agent/src/ai/openaicompat	3.939s
ok  	github.com/cachicamas/backend/agent/src/agenttest	2.062s
ok  	github.com/cachicamas/backend/agent/src/ai	3.414s
ok  	github.com/cachicamas/backend/agent/src/ai/openaicompat	4.042s
ok  	github.com/cachicamas/backend/agent/src/agenttest	2.078s
ok  	github.com/cachicamas/backend/agent/src/ai	3.470s
ok  	github.com/cachicamas/backend/agent/src/ai/openaicompat	4.005s
ok  	github.com/cachicamas/backend/agent/src/agenttest	2.115s
ok  	github.com/cachicamas/backend/agent/src/ai	3.574s
ok  	github.com/cachicamas/backend/agent/src/ai/openaicompat	4.052s
```

### Rollback boundary

`git revert a522357` cleanly removes the 3 position tests (no production diff). The landed usage capture in `stream_state.go`'s `applyChunk` is unchanged.

---

## Aggregate Evidence

### Per-slice tests landed

| Slice | New test files | New test functions | New sub-cases | Production diff |
|---|---|---|---|---|
| 1 (AI-31.1) | `finish_reason_test.go` | 6 | 13 | 0 (coverage-only) |
| 2a (AI-31.2a) | `usage_completion_test.go` | 5 | 5 | +3 raw mappings + 2 detail structs |
| 2b (AI-31.2b) | `usage_probes_test.go` | 3 | 3 | 0 (coverage-only) |
| 3 (AI-31.3) | `usage_position_test.go` | 3 (+ 1 helper) | 3 | 0 (coverage-only) |
| **Total** | **4 new files** | **17 functions** | **24 sub-cases** | **2 detail structs + 3 raw mappings** |

### TDD Cycle Evidence — full table

| Slice | Test File | Layer | Safety Net | RED | GREEN | TRIANGULATE | REFACTOR |
|---|---|---|---|---|---|---|---|
| 1.1 | `finish_reason_test.go` | Unit | ✅ baseline | ✅ | ✅ | ✅ 5 cases | ✅ comments |
| 1.2 | `finish_reason_test.go` | Unit | ✅ | ✅ | ✅ | ✅ 5+1 cases | ✅ |
| 1.3 | `finish_reason_test.go` | Unit | ✅ | ✅ | ✅ | ✅ 3 cases | ✅ |
| 1.4 | `finish_reason_test.go` | Unit | ✅ | ✅ | ✅ | ✅ 2 cases | ✅ |
| 2.1 | `usage_completion_test.go` | Unit | ✅ S-ATS-055…062 byte-identical | ✅ | ✅ | ✅ 2 cases | ✅ D-C comment |
| 2.2 | `usage_completion_test.go` | Unit | ✅ | ✅ | ✅ | ✅ 2 cases | ✅ |
| 2.3 | `chunk.go` (production) | — | — | — | ✅ +3 raw mappings, no arithmetic | — | ✅ |
| 2.4 | `chunk.go` (doc) | — | — | — | ✅ AI-13.4 quote, U1/U2, AI-38.2 | — | ✅ |
| 3.1 | `usage_probes_test.go` | Unit | ✅ | ✅ | ✅ | ➖ Single impossible pair | ✅ |
| 3.2 | `usage_probes_test.go` | Unit | ✅ | ✅ | ✅ | ➖ Single U6 shape | ✅ |
| 3.3 | `usage_probes_test.go` | Unit | ✅ | ✅ | ✅ | ➖ Single 3-frame shape | ✅ |
| 4.1 | `usage_position_test.go` | Unit | ✅ | ✅ | ✅ | ➖ Single partial-leaf | ✅ |
| 4.2 | `usage_position_test.go` | Unit | ✅ | ✅ | ✅ | ➖ Single odd-position | ✅ |
| 4.3 | `usage_position_test.go` | Unit | ✅ | ✅ | ✅ | ➖ Single metadata-only | ✅ |

### Test Summary

- **Total tests written**: 17 test functions across 4 new files
- **Total sub-cases**: 24 sub-cases
- **Total tests passing**: 17 / 17 (24 / 24 sub-cases)
- **Layers used**: Unit (17), Integration (0), E2E (0)
- **Approval tests**: 0 (no refactoring tasks — all slices were additive or coverage-only)
- **Pure functions created**: 0 (the design extends the existing `usageFromWire` rather than extracting new pure functions)
- **Frozen tests preserved**: S-ATS-055…062 byte-identical — `git diff` on `usage_test.go` is empty
- **Coverage-only slices**: 3 of 4 (1, 2b, 3 had no production diff; only 2a added production code, the minimal +3 raw mappings + 2 detail structs)

### Workload / PR Boundary

- **Delivery mode**: auto-chain, feature-branch-chain, `size:exception` accepted (obs #2535)
- **Effective budget**: 2000 lines for this apply
- **Actual diff**: `fe1ad92..HEAD` = 1059 insertions / 39 deletions = **1098 net lines** across 9 files (well within budget)
- **Branch state**: 4 slice commits on `feat/ai-31-completion` (slice-1 `d141326`, slice-2a `38bd916`, slice-2b `cb80861`, slice-3 `a522357`)
- **Boundaries**: each slice is a clean `git revert` boundary; `usage_test.go` is byte-identical (frozen)

### Hard Rules Compliance

- ✅ **Zero new exported identifiers in `src/ai` or `src/agenttest`** — Branch B carried raw in adapter; new types `wirePromptTokensDetails` / `wireCompletionTokensDetails` are unexported; mappings in the carrier only
- ✅ **Zero `go.mod` / `go.work` changes** — `go.mod` byte-identical to the chain base; zero new requires
- ✅ **All failure-drain tests invoke `requireCheckStreamClean`** — every new drained-stream test asserts it (17 / 17)
- ✅ **9 vacuous-pass shapes (Engram obs #2471) applied** — S-ACP-013 absent-vs-zero negative control, S-ACP-012 detail-objects-absent negative control, S-ACP-005 novel-value negative control, S-ACP-010 extra-unknown-key negative control, S-ACP-015 subtraction-would-fail probe, S-ACP-017 impossible-arithmetic probe, S-ACP-018 baseline single-frame (so the multi-frame case is provably distinct), S-ACP-019 multi-frame pin (the ONE dialect-conventional label), S-ACP-023 explicit `requireCheckStreamClean` is the scenario not prose
- ✅ **Charter delta preserved** — `specs/ai-provider-text-stream/spec.md` present and unmodified by this change

### Per-slice commit evidence

```
$ git log --oneline -5
a522357 feat(openaicompat): AI-31.3 — never-invent / never-assume-position coverage
cb80861 feat(openaicompat): AI-31.2b — impossible-arithmetic probe and multi-frame merge pin
38bd916 feat(openaicompat): AI-31.2a — detail-struct decode and three raw usage mappings
d141326 feat(openaicompat): AI-31.1 — finish-reason coverage and doc 0002 amendments
fe1ad92 docs(sdd): start AI-31 completion — bank planning artifacts
```

### Gate Compliance Summary

| Slice | go.mod | gofmt | lint (go vet) | tests | `requireCheckStreamClean` |
|---|---|---|---|---|---|
| 1 | unchanged | clean (chunk.go + stream_state.go reformatted by gofmt) | clean | ✅ PASS | ✅ every drained test |
| 2a | unchanged | clean | clean | ✅ PASS | ✅ every drained test |
| 2b | unchanged | clean | clean | ✅ PASS | ✅ every drained test |
| 3 | unchanged | clean | clean | ✅ PASS ×4 runs | ✅ every drained test |

(`src/ai/completion_test.go` gofmt drift is pre-existing and explicitly ignored per task 1.10.)

### Status

**17 / 17 tasks complete. Ready for `sdd-verify`.**
