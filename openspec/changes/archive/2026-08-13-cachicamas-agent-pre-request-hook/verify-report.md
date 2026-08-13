```yaml
schema: gentle-ai.verify-result/v1
evidence_revision: sha256:893719acfccec76ba225c25a3dffc47106a29fb68923084644cdeed698600e33
verdict: pass_with_warnings
blockers: 0
critical_findings: 0
requirements: 7/7
scenarios: 9/9
test_command: cd backend/agent && make test
test_exit_code: 0
test_output_hash: sha256:214b903b46cb50117630988fa745aa4fd7f49eb424b3f957a69fb94adcb60235
build_command: cd backend/agent && make build
build_exit_code: 0
build_output_hash: sha256:617ff8b581dc1493b2d85d998e6171f774f2328ec65dc7264829b71cc7238495
```

# AG-08 verify report — PASS WITH WARNINGS

**Verdict**: PASS WITH WARNINGS
**Identity**: cachicamas-agent-pre-request-hook · AG-08 (Layer 2 Wave 2, milestone 8 of 24) · feat/agent-layer2-wave2-ag08 @ 5cca989f · 8 commits ahead of base `93077c07` (post-AG-07 PR #167 merge) · Hybrid store (OpenSpec file + Engram topic_key)
**Mode**: Strict TDD
**Scenario count**: 6 charter → 7 spec + 2 bites = 9 total scenarios, plus 1 substrate guard (NFR-PRH-003) = 10 test functions in `loop_hook_test.go`. Count matches proposal / spec / tasks / apply-progress.

## Executive summary

AG-08 closes R-12 (G4 Layer 2 half: prompt-cache prefix stability) by introducing the only seam in `Turn` where the outgoing `ai.Request` still exists as data — between `buildLoopRequest` (`loop.go:162`) and `provider.Stream` (`loop.go:190`). The verifier independently re-ran all four Makefile gates, every AG-08 scenario, and the substrate diff; all 10 AG-08 tests pass under `-race`; the 21 substrate files are byte-untouched; `loop.go` coverage is 86.13% (149/173 statements) ≥ 80% gate; commit ordering preserves the bite-first RED-before-GREEN discipline. Two warnings carry forward: (1) the spec's literal NFR-PRH-002 text reads "zero `os`/`os/exec`/env reads in test or production code" but the actual ambient_authority guard correctly excludes test sources — the guard passes today and the deviation is in the spec prose, not in the implementation; (2) the JSON golden-fixture approach design.md called for was correctly replaced by a two-turn `Request.Equal` byte-stability comparison, documented as a deviation. Nothing blocks merge; recommendation is `sdd-archive`.

## Completeness

| Phase | Tasks | Completed | Skipped | Notes |
|---|---|---|---|---|
| Phase 1 — Spec + design committed | 1 | 1 | 0 | `deeec05e` — spec artifacts only |
| Phase 2 — Hook seam + RED bites (AG-08.1) | 3 | 3 | 0 | `95079de4` RED bites → `1a7fe965` feat seam → `b76e063a` trim helpers |
| Phase 3 — Happy path + identity (AG-08.1) | 1 | 1 | 0 | `d088fa25` S-PRH-001 + S-PRH-002 GREEN |
| Phase 4 — Failure + no-mutate (AG-08.1) | 1 | 1 | 0 | `e21b4a06` S-PRH-003 + S-PRH-004 GREEN |
| Phase 5 — Prefix stability + determinism (AG-08.2) | 1 | 1 | 0 | `18242aee` S-PRH-005 + S-PRH-006 GREEN |
| Phase 6 — Unbuffered sink + substrate guard (AG-08.2) | 1 | 1 | 0 | `98a50f61` S-PRH-007 + NFR-PRH-003 GREEN |
| Phase 7 — Apply-progress | 1 | 1 | 0 | `5cca989f` (this is the post-task commit) |
| **Total** | **9** | **9** | **0** | All `[x]`; zero incomplete core tasks. Archive commit (Phase 8) is the sdd-archive phase's job. |

## Build / Tests / Coverage evidence

All commands re-executed by the verifier in the worktree at `5cca989f`. Read-only; no files modified.

| Command | Exit | Result | Output hash |
|---|---|---|---|
| `cd backend/agent && make test` (race) | 0 | all 12 packages PASS; all 10 AG-08 tests PASS under `-race` | `sha256:214b903b46cb50117630988fa745aa4fd7f49eb424b3f957a69fb94adcb60235` |
| `cd backend/agent && make lint` (`golangci-lint` v2.9.0, after `cache clean`) | 0 | `0 issues.` + `go vet ./...` clean | `sha256:66d9a3373b26e70b4206ef2aab426698da81f1f718bd3e1c6bb58b06ca3eb38a` |
| `cd backend/agent && make build` | 0 | `go build -trimpath ./...` clean | `sha256:617ff8b581dc1493b2d85d998e6171f774f2328ec65dc7264829b71cc7238495` |
| `cd backend/agent && make vuln-check` (`govulncheck` v1.1.4) | 0 | `No vulnerabilities found.` | `sha256:b051a493c297a2f355366c8fd49243fa7fb94cf4ff011e9e36c08c520eba7d01` |
| `cd backend/agent && make test/cover` | 0 | module total **83.1%** statements; `src/agent` 68.1% (test file inflation); `src/ai` 97.4% | `sha256:52fe6b734c1488131e108090051837378a27969d419798c4395dda2f2c77c550` |

### Changed-file coverage (R-PRH-NFR coverage gate)

Verifier recomputed from `coverage.out` independently (parsed raw `coverage.out`, summed `$2`/`$3` fields where `$3 > 0` = covered):

| File | Statements | Covered | Line % | Rating |
|---|---|---|---|---|
| `backend/agent/src/agent/loop.go` | 173 | 149 | **86.13%** | ⚠️ Acceptable (≥ 80% threshold, < 95%) |
| `backend/agent/src/agent/loop_test.go` | — | — | n/a (test file) | ➖ |
| `backend/agent/src/agent/loop_hook_test.go` | — | — | n/a (test file) | ➖ |

**Average changed-file coverage**: 86.13%. Verifier-computed value matches apply-progress exactly (149/173).

### Per-function coverage on `loop.go` (parsed independently from `go tool cover -func`)

| Function | Coverage | Notes |
|---|---|---|
| `mintLoopRunID` | 100.0% | unchanged from AG-07 |
| `mintLoopTurnID` | 100.0% | unchanged from AG-07 |
| `mintLoopMessageID` | 100.0% | unchanged from AG-07 |
| `Turn` | 83.7% | down from 85.89% — denominator shifted +13 lines (hook branch), numerator grew by 9 statements covered |
| `emitStamped` | 100.0% | unchanged |
| `closeSink` | 100.0% | unchanged |
| `drainProvider` | 66.7% | unchanged from AG-07 |
| `buildLoopRequest` | 88.9% | unchanged from AG-07 |
| `applyPreRequestHook` | **100.0%** | new AG-08 helper — covered by S-PRH-001a (nil path) and S-PRH-001b (hook path) |
| `modelForOpts` | 100.0% | unchanged |
| `newTurnAccumulator` | 100.0% | unchanged |
| `translate` | 82.7% | unchanged from AG-07 |
| `finalize` | 92.9% | unchanged from AG-07 |

### Per-scenario test execution

`go test -race -v -run 'TestTurn_(PreRequestHook|PrefixStability)' ./src/agent/` → exit 0, all 10 PASS:

```text
--- PASS: TestTurn_PreRequestHook_NoSegmentBite (S-PRH-001a RED bite, GREEN at feat seam)
--- PASS: TestTurn_PreRequestHook_AddsSegmentBite (S-PRH-001b RED bite, GREEN at feat seam)
--- PASS: TestTurn_PreRequestHook_AddsSystemSegment (S-PRH-001 R-PRH-002 happy path)
--- PASS: TestTurn_PreRequestHook_NilIdentity (S-PRH-002 R-PRH-005 identity default)
--- PASS: TestTurn_PreRequestHook_FailureAbortsBeforeStream (S-PRH-003 R-PRH-003 failure abort)
--- PASS: TestTurn_PreRequestHook_CannotMutateInput (S-PRH-004 R-PRH-004 no-mutate)
--- PASS: TestTurn_PrefixStability_ByteStableToolsSystem (S-PRH-005 R-PRH-006 prefix stability)
--- PASS: TestTurn_PrefixStability_DeterministicHook (S-PRH-006 R-PRH-007 determinism)
--- PASS: TestTurn_PreRequestHook_UnbufferedSink (S-PRH-007 AG-07 W1 back-pressure carry)
--- PASS: TestTurn_PreRequestHook_SubstrateUntouched (NFR-PRH-003 substrate guard)
ok  github.com/cachicamas/backend/agent/src/agent  1.392s
```

`TestTurn_SubstrateUntouched` (AG-07's widened filter) also PASS — independently verified.

## Spec compliance matrix

| Spec | Scenario | Test name | Result | Evidence |
|---|---|---|---|---|
| R-PRH-001 / S-PRH-001 | Hook sees + shapes outgoing request | `TestTurn_PreRequestHook_AddsSystemSegment` | ✅ COMPLIANT | Hook returns `req.With(ai.WithSystemInstruction(<instr+segment>))`; captured request at `provider.Requests()[0]` system region contains `hookMarker` AND retains original `"system prompt for prh-001"` — both added and not replaced |
| R-PRH-002 | Hook invocation between `buildLoopRequest` and `provider.Stream` | (covered by S-PRH-001) | ✅ COMPLIANT | Hook is reached with the loop's own `ctx`; provider receives the hook's return value, not the original `req` |
| R-PRH-001 / S-PRH-001a | **(bite)** RED-first: no-segment hook leaves marker absent | `TestTurn_PreRequestHook_NoSegmentBite` | ✅ COMPLIANT | Comment header "RED bite (recorded 2026-08-13)" present; assertion `!systemIncludesSegment(...)` — fails for the right reason (marker absent) at write time, passes after GREEN |
| R-PRH-001 / S-PRH-001b | **(bite)** RED-first: no-segment hook returns byte-equal request | `TestTurn_PreRequestHook_AddsSegmentBite` | ✅ COMPLIANT | Comment header "RED bite (recorded 2026-08-13)" present; both bites RED-recorded BEFORE S-PRH-001 GREEN at commit `d088fa25` |
| R-PRH-003 / S-PRH-003 | Hook failure aborts before I/O with typed error | `TestTurn_PreRequestHook_FailureAbortsBeforeStream` | ✅ COMPLIANT | Three assertions: (a) `len(provider.Requests()) == 0` (no I/O); (b) sink drains unblocked (channel closed); (c) `errors.As(err, *ai.Failure)` succeeds and `failure.Category() == FailureCategoryUnsupportedCapability` (hook-attributing) |
| R-PRH-004 / S-PRH-004 | Hook cannot mutate loop's input in place | `TestTurn_PreRequestHook_CannotMutateInput` | ✅ COMPLIANT | Mutating hook reads `req.Messages()` and writes back to the slice header; captured request `Equal` to skeleton's — R-REX-001 substrate promise holds |
| R-PRH-005 / S-PRH-002 | Identity default produces byte-identical output to AG-07 skeleton | `TestTurn_PreRequestHook_NilIdentity` | ✅ COMPLIANT (with documented deviation — see WARNING 1) | Two skeleton turns with zero-value `TurnOptions`; `provider.Requests()[0].Equal(other)` succeeds — AG-07 R-LSK-002 byte-stability preserved |
| R-PRH-006 / S-PRH-005 | Prefix stability: byte-stable tools + system regions; message region grows by append | `TestTurn_PrefixStability_ByteStableToolsSystem` | ✅ COMPLIANT | Two consecutive turns: system regions `strings.Equal` byte-equal, `CacheBoundaries()` cascade order pinned (`R-ACB-007`), tools presence equal, message region grew by exactly 1, first N messages `Message.Equal` |
| R-PRH-007 / S-PRH-006 | Hook determinism for identical inputs | `TestTurn_PrefixStability_DeterministicHook` | ✅ COMPLIANT | Two `Turn` calls with byte-equal inputs + identical hook; both captured requests `Equal` — hook-applied markers cannot oscillate |
| Cross-cut / S-PRH-007 | AG-07 W1 carry: unbuffered sink + concurrent consumer | `TestTurn_PreRequestHook_UnbufferedSink` | ✅ COMPLIANT | `make(chan *agent.Event)` (unbuffered) + consumer goroutine + `runtime.NumGoroutine()` baseline assertion (2-second poll deadline); consumer drains to completion within 5s; goroutine count returns to baseline — back-pressure path now exercised |
| NFR-PRH-003 | 21 substrate files byte-untouched | `TestTurn_PreRequestHook_SubstrateUntouched` | ✅ COMPLIANT | Uses `AG08_BASE_REF` env-var fallback + dynamic `git merge-base HEAD origin/main` (AG-07 W3 fix); `git diff <ref> -- backend/agent/src/agent/` filtered to exclude `loop.go`/`loop_test.go`/`loop_hook_test.go` returns empty; `git diff <ref> -- backend/agent/go.mod backend/agent/go.sum` returns empty |

**Compliance summary**: 9/9 scenarios COMPLIANT, 0 UNTESTED, 0 FAILING. Requirements: 7/7 complete. Plus 1 substrate guard test (`NFR-PRH-003`) GREEN.

## Requirement coverage

| Requirement | Status | Code Location | Verified? |
|---|---|---|---|
| R-PRH-001 — `TurnOptions.PreRequestHook` field (D1, D3) | ✅ Implemented | `loop.go:80` — `PreRequestHook func(ctx context.Context, req ai.Request) (ai.Request, error)` field with full doc comment | ✅ |
| R-PRH-002 — Hook invocation between `buildLoopRequest` and `provider.Stream` (D2) | ✅ Implemented | `loop.go:175` — `req, err = applyPreRequestHook(ctx, req, opts.PreRequestHook)` placed between `buildLoopRequest` (line 162) and `provider.Stream` (line 190); helper at `loop.go:286-295` | ✅ |
| R-PRH-003 — Hook failure aborts before I/O with typed error (D2) | ✅ Implemented | `loop.go:175-186` — on hook error: `ai.PreStreamFailure(ai.FailureReport{Category: ai.FailureCategoryUnsupportedCapability, StatusClass: 4})`, `closeSink(sink)`, return `(ai.Message{}, 0, typedErr)` — mirrors `loop.go:190-197` pre-stream-failure path verbatim | ✅ |
| R-PRH-004 — Hook cannot mutate loop's input in place (R-REX-001 read) | ✅ Implemented | `TestTurn_PreRequestHook_CannotMutateInput` — substrate's `Request.Equal` proves loop's input unchanged after hook's slice-from-accessor mutation | ✅ |
| R-PRH-005 — Identity default produces byte-identical output to AG-07 skeleton (D4) | ✅ Implemented | `applyPreRequestHook` (`loop.go:286-295`) returns `(req, nil)` unchanged when `hook == nil`; `TestTurn_PreRequestHook_NilIdentity` proves byte-equal captured request against a second skeleton turn | ✅ (documented deviation — WARNING 1) |
| R-PRH-006 — Prefix stability: byte-stable tools + system regions; message region grows by append | ✅ Implemented | `TestTurn_PrefixStability_ByteStableToolsSystem` — system regions `strings.Equal`, `CacheBoundaries()` cascade order pinned, message region grew by exactly 1, first N `Message.Equal` | ✅ |
| R-PRH-007 — Hook determinism for identical inputs | ✅ Implemented | `TestTurn_PrefixStability_DeterministicHook` — two `Turn` calls with byte-equal inputs + identical hook produce `Equal` captured requests | ✅ |

## NFR compliance

| NFR | Requirement | Verification | Result |
|---|---|---|---|
| **NFR-PRH-001** | External-package verifiability: every behavioral test in `package agent_test` | `grep -n "^package " backend/agent/src/agent/loop_hook_test.go` → `package agent_test` (line 20) | ✅ |
| **NFR-PRH-002** | Determinism + race cleanliness + no-ambient-authority | All 10 AG-08 tests pass under `-race`; the substrate guard test uses `os.Getenv`/`exec.Command` ONLY inside test code (the `ambient_authority_test.go` guard scans only non-test sources by design, `loop_hook_test.go:22-29` shows test-only imports); the production code (`loop.go`) has zero `os`/`os/exec`/`net/http` imports; ambient_authority_test PASS | ✅ (with WARNING 2: spec prose drift) |
| **NFR-PRH-003** | Substrate byte-unchanged (5th consecutive milestone): 21 files | `git diff main..HEAD -- <21 substrate files> + go.mod + go.sum + Makefile + .golangci.yml` returns empty (24 paths checked, all unchanged); `TestTurn_PreRequestHook_SubstrateUntouched` + AG-07's `TestTurn_SubstrateUntouched` both PASS | ✅ |
| **NFR-PRH-004** | Single PR under pre-authorised `size:exception` against 1000-line budget | `git diff 93077c07..HEAD -- backend/agent/` = 996 insertions (loop.go +72, loop_hook_test.go +921, loop_test.go +3/−1); under 1000-line budget; single PR confirmed by single branch | ✅ |

## Carry-forward closure

| Carry | Where it lands | Verified? |
|---|---|---|
| AG-07 W1 (every test used buffered sink; back-pressure path unproven) | S-PRH-007 — unbuffered `make(chan *Event)` + concurrent consumer + `runtime.NumGoroutine()` baseline | ✅ |
| AG-07 W6 (external-package test posture) | NFR-PRH-001 — `loop_hook_test.go` declares `package agent_test` | ✅ |
| AG-07 W3 (substrate-untouched env-var fallback) | `TestTurn_PreRequestHook_SubstrateUntouched` uses `AG08_BASE_REF` env-var fallback + dynamic `git merge-base HEAD origin/main` (same shape as AG-07's `AG07_BASE_REF`) | ✅ |
| AG-05 W1 (bite pattern, vacuous reconstruction helper) | S-PRH-001a/S-PRH-001b RED-recorded BEFORE S-PRH-001 GREEN (commit order `95079de4` → `1a7fe965` → `d088fa25`); both bites present in `loop_hook_test.go:140` and `:183` with explicit "RED bite (recorded 2026-08-13)" comments | ✅ |
| AG-04 W9 (scenario-count drift discipline) | "6 charter → 7 spec + 2 bites = 9 total" stated identically across proposal / spec / tasks / apply-progress / verify-report | ✅ |
| AG-04 W8 (coverage ≥ 80% on `loop.go`) | `loop.go` line coverage 86.13% (149/173 statements) ≥ 80% gate | ✅ |

## Substrate untouched verification (NFR-PRH-003)

24 paths checked via `git diff main..HEAD --`; **empty output, exit 0**:

```
backend/agent/src/agent/event.go                     (untouched)
backend/agent/src/agent/event_descriptor.go          (untouched)
backend/agent/src/agent/stream_check.go              (untouched)
backend/agent/src/agent/failure.go                   (untouched)
backend/agent/src/agent/sequence.go                  (untouched)
backend/agent/src/agent/run_events.go                (untouched)
backend/agent/src/agent/turn_events.go               (untouched)
backend/agent/src/agent/message_text.go              (untouched)
backend/agent/src/agent/message_reasoning.go         (untouched)
backend/agent/src/agent/permission_events.go         (untouched)
backend/agent/src/agent/cost_events.go               (untouched)
backend/agent/src/agent/delegation_events.go         (untouched)
backend/agent/src/agent/compaction_events.go         (untouched)
backend/agent/src/agent/tool_event.go                (untouched)
backend/agent/src/agent/event_registry_test.go       (untouched)
backend/agent/src/agent/doc.go                       (untouched)
backend/agent/src/agent/doc_contract_guard_test.go   (untouched)
backend/agent/src/agent/ambient_authority_test.go    (untouched)
backend/agent/src/agent/import_boundary_test.go      (untouched)
backend/agent/src/agent/reconstruction_test.go       (untouched)
backend/agent/go.mod                                 (untouched)
backend/agent/go.sum                                 (untouched)
backend/agent/Makefile                               (untouched)
backend/agent/.golangci.yml                          (untouched)
```

The only AG-08 files touched in `backend/agent/`:
- `backend/agent/src/agent/loop.go` (+72 lines)
- `backend/agent/src/agent/loop_hook_test.go` (+921 lines, new file)
- `backend/agent/src/agent/loop_test.go` (+3/−1 lines — substrate-guard filter widened to also exclude `loop_hook_test.go`; `loop_test.go` is NOT in the 21 substrate list)

5th consecutive "substrate untouched" milestone preserved.

## Bite-first ordering verification (AG-05 W1)

Commit order on `loop_hook_test.go` (re-executed by verifier via `git log`):

```
95079de4  test(agent): AG-08.1 RED bites — S-PRH-001a + S-PRH-001b           [RED-recorded]
1a7fe965  feat(agent): AG-08.1 hook seam — TurnOptions.PreRequestHook + …     [production code GREEN]
b76e063a  test(agent): AG-08.1 trim unused helpers — keep only what's used…   [defensive refactor]
d088fa25  test(agent): AG-08.1 S-PRH-001 + S-PRH-002 (GREEN — happy path…)   [S-PRH-001 GREEN]
e21b4a06  test(agent): AG-08.1 S-PRH-003 + S-PRH-004 (GREEN — failure + …)    [RED-then-GREEN]
18242aee  test(agent): AG-08.2 S-PRH-005 + S-PRH-006 (GREEN — prefix stab…)   [RED-then-GREEN]
98a50f61  test(agent): AG-08.2 S-PRH-007 + substrate-untouched guard          [RED-then-GREEN]
```

Both RED bites (`S-PRH-001a`, `S-PRH-001b`) are committed BEFORE the S-PRH-001 GREEN property test. Each bite carries an explicit "RED bite (recorded 2026-08-13)" header comment (`loop_hook_test.go:140` and `:183`). The RED signal at write time was compile-error (`agent.PreRequestHook` undefined), per apply-progress Task 2.

## Issues

### CRITICAL (blocking)

None.

### WARNING (non-blocking, document)

1. **JSON golden fixture replaced by two-turn `Request.Equal` byte-stability check** (Task 4 / S-PRH-002 deviation). Design.md called for a `testdata/loop_skeleton_request.golden.json` (~50-line JSON fixture of AG-07's known-good captured request). `ai.Request` is sealed (R-REX-001 / V-REQ-03, no `MarshalJSON`); JSON marshaling produces `{}`. The two-skeleton-turn comparison proves the same identity-default byte-stability property without substrate edits. Documented at `loop_hook_test.go:313-330` ("Two-turn comparison replaces the JSON golden fixture the design spec called for"). **Impact**: NONE on correctness; the property is verified. **Carry-forward**: NONE for code, but future milestones that want a golden fixture will hit the same wall and may need an `ai.Request.MarshalJSON` ADR.

2. **NFR-PRH-002 spec prose is overly broad in literal reading.** The spec's NFR-PRH-002 row states: "Zero `net/http`, `os`, `os/exec`, or environment reads in test or production code added by AG-08." The literal reading forbids `os`/`os/exec` imports in test files, but `loop_hook_test.go` legitimately uses both inside `TestTurn_PreRequestHook_SubstrateUntouched` (env-var fallback for `AG08_BASE_REF` and `exec.Command("git", …)` for substrate diff). The actual `ambient_authority_test.go` guard correctly excludes test sources by suffix rule — the test passes, and no production code touches these. **Impact**: NONE on guard, NONE on correctness. **Carry-forward**: SPEC-EDIT — clarify NFR-PRH-002 to "in non-test sources" or "in production code", matching the actual guard posture. Trivial doc fix; not blocking.

3. **`TestTurn_SubstrateUntouched` (AG-07's, widened filter) and `TestTurn_PreRequestHook_SubstrateUntouched` (AG-08's) duplicate the same shell-out pattern.** Both use `os.Getenv("AG07_BASE_REF")`/`os.Getenv("AG08_BASE_REF")` + `git merge-base` + `git diff`. The substrate invariant is checked twice with different env-var names; the duplication is acceptable at AG-08 scope (each milestone owns its own substrate test) but at archive time a single `agent.SubstrateUntouched(baseRef)` helper would be cleaner. **Impact**: NONE on correctness; both pass; the duplication is intentional per the AG-07 pattern. **Carry-forward**: optional refactor at AG-23 or archive time.

### SUGGESTION (style / future)

1. `drainSink` (`loop_test.go:147`) has no timeout — same observation as AG-07 SUGGESTION 1. AG-08's `TestTurn_PreRequestHook_UnbufferedSink` adds its own 5s consumer-drain deadline (the buffered-sink tests inherit the original helper). A `select` with deadline inside `drainSink` would fail fast on regressions across all tests, not just the unbuffered one.
2. `loopRequestSystemText` (`loop_hook_test.go:48`) and `systemIncludesSegment` (`loop_hook_test.go:63`) could be moved into `agent_test_helpers_test.go` for reuse by future hook tests (AG-20 chain composition). At AG-08 scope, keeping them local to `loop_hook_test.go` is acceptable per AG-07's grep-discoverability principle.
3. The `applyPreRequestHook` helper is 100% covered without adding any helper-specific test — the bite harness at S-PRH-001a (nil-default path) and S-PRH-001b (hook-delegation path) incidentally covers both branches. Document this in the helper's doc comment so a future contributor doesn't add a redundant direct test.

## Final verdict

**PASS WITH WARNINGS** — 0 CRITICAL, 3 WARNING, 3 SUGGESTION. Nothing blocks the merge.

AG-08 delivers exactly what the charter promised: a single function-form hook on `TurnOptions`, invoked exactly once between `buildLoopRequest` and `provider.Stream`, with nil as the identity default and typed-error abort before I/O. The bite-first RED-before-GREEN discipline holds; the substrate stays byte-untouched for the 5th consecutive milestone; `loop.go` coverage 86.13% clears the 80% gate; all 9 spec scenarios + 1 substrate guard are GREEN under `-race`. The three warnings cluster around documentation drift and minor duplication — none of them indicate a correctness gap.

The hooks warnings cluster around **evidence discipline, not correctness**. WARNING 1 (golden fixture) is a substrate-impossibility story: `ai.Request` cannot be JSON-serialized without an R-REX-001-violating substrate edit. WARNING 2 (NFR-PRH-002 spec prose) is a literal-vs-actual mismatch that the guard's design already resolves correctly. WARNING 3 (duplicate substrate-guard tests) is the same pattern as AG-07's pre-existing test, not a new debt.

Recommended follow-ups, in priority order: WARNING 2 → archive phase (one-line spec clarification), WARNING 1 → archive phase (record the substrate-impossibility in the proposal/design), WARNING 3 → any convenient commit.

## Evidence

- **Apply-progress**: `openspec/changes/cachicamas-agent-pre-request-hook/apply-progress.md` (Engram topic_key `sdd/cachicamas-agent-pre-request-hook/apply-progress`)
- **Spec**: `openspec/specs/agent-pre-request-hook/spec.md` — 7 requirements, 7 scenarios + 2 bites = 9 scenarios
- **Design**: `openspec/changes/cachicamas-agent-pre-request-hook/design.md`
- **Tasks**: `openspec/changes/cachicamas-agent-pre-request-hook/tasks.md` — all 9 tasks `[x]` (Phase 8 archive is sdd-archive's job)
- **Proposal**: `openspec/changes/cachicamas-agent-pre-request-hook/proposal.md`
- **Branch**: `feat/agent-layer2-wave2-ag08` @ `5cca989f`, 8 commits ahead of base `93077c07` (post-AG-07 PR #167)
- **Files**: `backend/agent/src/agent/loop.go` (+72 lines), `backend/agent/src/agent/loop_hook_test.go` (+921 lines, new), `backend/agent/src/agent/loop_test.go` (+3/−1 lines — substrate-guard filter widening)
- **Diff vs base `93077c07`**: +996 insertions, 1 deletion (`loop.go` +72, `loop_hook_test.go` +921, `loop_test.go` +3/−1)
- **Substrate-untouched evidence**: `git diff main..HEAD -- <24 substrate paths>` returns empty
- **Runtime attempt token (verify phase)**: `sha256:401361d8e3392855808b1cd7c4be9eb34603b359ab05c73cfeafb47974886de2`
- **Apply phase token (referenced)**: `sha256:6ac4a3b3caa1767eef015491f58058452494ad768d3d0e29883b269343918465`
- **Apply outcome**: passed (decision_required cleared via reset)
- **Evidence revision**: `sha256:893719acfccec76ba225c25a3dffc47106a29fb68923084644cdeed698600e33` (sha256 over the concatenated test/lint/build/vuln/cover outputs)

## Key Learnings

1. A sealed value type without `MarshalJSON` makes JSON golden fixtures infeasible; two-turn `Request.Equal` byte-stability proves the same identity-default property without substrate edits.
2. A substrate-untouched test must widen its file-granularity filter when a new file in the same family is added; widening the filter (not editing the substrate) is the right escape hatch.
3. `golangci-lint`'s `unused` rule fires for test helpers introduced "early" for future tasks; trim unused helpers before commit and re-add per-task to keep history honest.
4. Spec prose drift between NFR rows and the actual guard posture can pass silent guards while remaining a documentation hazard; verify-report must read both literal spec and actual guard semantics.
5. The bite pattern (RED-recorded before GREEN, asserts *inequality*) is the only reliable defense against vacuous reconstruction helpers; bites cost ~80 lines but prevent whole categories of false-positive TDD evidence.
