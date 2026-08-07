# Apply Progress: Discharge the Wave-2 carryovers (AI-41)

> **Change**: `cachicamas-ai-wave2-carryovers` · **Milestone**: AI-41 · **Wave**: 5 — Harden
> **Branch**: `feat/ai-41-wave2-carryovers` (base `origin/main` @ `f2e460d`) · **Worktree**: `cachicamas-worktrees/ai-41-wave2-carryovers`
> **Mode**: Strict TDD (RED → GREEN → REFACTOR per leaf)
> **Scope executed**: WU-1 through WU-5 (Phase 1–3 of `tasks.md`). Phase 4 (A.1–A.5, archive-phase spec/doc promotion) intentionally NOT executed — reserved for `sdd-archive`, after merge.

## Status

18/18 apply-phase tasks (WU-1 … WU-5) complete. Phase 4 (5 archive tasks) untouched, as scoped. Ready for `sdd-verify`.

## Commits

| SHA | Message | Files |
| --- | --- | --- |
| `ca2ede7` | `feat(ai): prove CheckEmit rule 4 surfaces the payload's own violation` | `export_test.go`, `event_test.go` |
| `b82cfdf` | `feat(ai): redact *Failure's Go-syntax formatting via GoString` | `provider_failure.go`, `provider_failure_test.go` |

Both on `feat/ai-41-wave2-carryovers`. Not pushed. No PR opened.

## TDD Cycle Evidence

| Leaf | RED (observed, verbatim) | GREEN (observed) | REFACTOR |
| --- | --- | --- | --- |
| **AI-41.1** (WU-1/WU-2, R-AEE-021) | `go test -race -run 'TestCheckEmit_PayloadReportsOwnViolation' ./src/ai/...` → exit 1: `event_test.go:240: ai.CheckEmit(rejecting) = nil, want the planted violation surfaced — validate still returns nil unconditionally` / `--- FAIL: TestCheckEmit_PayloadReportsOwnViolation_SurfacedAsRule4` | Same command → exit 0: `--- PASS: TestCheckEmit_PayloadReportsOwnViolation_SurfacedAsRule4`, `ok .../src/ai` | `validate`'s doc comment refreshed to present tense ("a rule is added once a test needs one to fail" → describes the now-landed `rejectWith` mechanism); landed in the same edit as the 1-line `return w.rejectWith` GREEN change (comment + method body inseparable in one hunk) |
| **AI-41.2** (WU-3/WU-4, R-AIP-016) | `go test -race -run 'TestFailure_GoString' ./src/ai/...` → exit 1, 2 failing tests, verbatim leak evidence below | Same command → exit 0: all sub-tests `--- PASS` | `GoString`'s doc comment written directly citing R-AIP-016, sibling precedents (`Event.GoString`, `Part.GoString`), and the nil-safety-by-delegation rationale; landed in the same edit as the 1-line method body |

**AI-41.2 verbatim RED output** (all three canaries leak under `%#v`; nil case also fails; `%v/%s/%+v` already pass):

```
provider_failure_test.go:1298: fmt.Sprintf("%#v", f) = "&ai.Failure{category:0x9, retryable:false, retryAfter:ai.RetryDelay{delay:0, present:false}, rawLabel:\"raw-label-CANARY-77f3\", statusClass:0, requestID:\"request-id-CANARY-88e1\", cause:\"cause-CANARY-99a2\", delivery:0x1, partialOutput:false}", want "provider failure: unknown"
provider_failure_test.go:1287: ... contains the planted canary "raw-label-CANARY-77f3" / "request-id-CANARY-88e1" / "cause-CANARY-99a2", want it excluded
provider_failure_test.go:1323: fmt.Sprintf("%#v", (*ai.Failure)(nil)) = "(*ai.Failure)(nil)", want "no provider failure"
--- FAIL: TestFailure_GoString_RedactsLikeError (0.00s)
--- FAIL: TestFailure_GoString_NilReceiver_TotalByDelegation (0.00s)
FAIL	github.com/cachicamas/backend/agent/src/ai	0.584s
```

This is a non-vacuous RED, confirmed empirically: the value-kind cause (`canaryCause`, a defined `string` type) is reflected verbatim (`cause:"cause-CANARY-99a2"`) — exactly D-4's prediction. Had the cause canary instead been planted via `errors.New(...)` (pointer-shaped), it would have rendered as a bare hex address (`cause:(*errors.errorString)(0x...)`) and the assertion would already have passed today, which is the vacuous-RED trap D-4 exists to avoid.

## Work Unit Evidence

| WU | Focused test command and exact result | Runtime harness | Rollback boundary |
| --- | --- | --- | --- |
| WU-1/WU-2 | `go test -race -run 'TestCheckEmit_PayloadReportsOwnViolation' ./src/ai/...` → RED then GREEN, both observed verbatim above | N/A — in-package unit test, no runtime/integration boundary crossed | `git revert ca2ede7` drops the `rejectWith` field, `NewRejectingWitnessEvent`, and the rule-4 test; `validate` reverts to unconditional `nil` |
| WU-3/WU-4 | `go test -race -run 'TestFailure_GoString' ./src/ai/...` → RED then GREEN, both observed verbatim above | N/A — in-package unit test, no runtime/integration boundary crossed | `git revert b82cfdf` drops `GoString`, the canary tests, and the S-AIP-008 exemption; `%#v` reverts to reflective (latent-leak) rendering and the terminal-exclusivity guard reverts to its pre-change strictness (safe, since `Completion` no longer has a same-named collision to guard against) |
| WU-5 | `cd backend/agent && make test` → exit 0, 0 `--- FAIL`, all 7 packages `ok` | Same — full-module verification | No code change; verification only |

## Gate Results (verbatim)

**`make test`** (`go test -race -v ./...`, full module, run from `backend/agent/`):
```
EXIT:0
ok  	github.com/cachicamas/backend/agent/src/agenttest	2.188s
ok  	github.com/cachicamas/backend/agent/src/ai	4.261s
ok  	github.com/cachicamas/backend/agent/src/ai/internal/retry	1.968s
ok  	github.com/cachicamas/backend/agent/src/ai/openaicompat	165.895s
ok  	github.com/cachicamas/backend/agent/src/ai/openaicompat/openrouter	3.068s
ok  	github.com/cachicamas/backend/agent/src/ai/openaicompat/openrouter/conformance	1.737s
ok  	github.com/cachicamas/backend/agent/src/ai/openaicompat/openrouter/smoke	2.786s
```
0 occurrences of `--- FAIL` across the whole log.

**`make lint`**:
```
go vet ./...
bin/golangci-lint run --config=.golangci.yml ./...
0 issues.
```
Exit 0.

**`make build`**:
```
go build -trimpath ./...
```
Exit 0, no output.

**`git diff --stat origin/main`**:
```
 backend/agent/src/ai/event_test.go            |  57 +++++++++++++
 backend/agent/src/ai/export_test.go           |  33 ++++++--
 backend/agent/src/ai/provider_failure.go      |  17 ++++
 backend/agent/src/ai/provider_failure_test.go | 113 ++++++++++++++++++++++++--
 4 files changed, 208 insertions(+), 12 deletions(-)
```
220 changed lines total. Forecast was ~90–125 (Low risk, single PR, no chaining). Over-forecast, but the 1000-line session budget has 780 lines of headroom remaining — no decision or exception needed. `git diff --stat origin/main -- backend/agent/go.mod backend/agent/go.sum` is empty — `go.mod`/`go.sum` byte-identical, zero new imports on either leaf, as designed.

## Deviations from Design

**One unplanned, in-scope-file fix, not anticipated by design.md's blast-radius analysis**: adding `GoString()` to `*Failure` (locked D-1, implemented exactly as specified) collides by exported-method-name with the pre-existing `Completion.GoString()` (landed at AI-15). This tripped `TestErrorEvent_TerminalExclusivity_NeverBothCompletionAndError`'s `Failure_and_Completion_export_no_accessor_of_the_same_name_(S-AIP-008)` sub-test in `provider_failure_test.go` — a pre-existing, already-shipped guard for a *different* requirement (R-AIP-003, AI-19), not part of this change's spec/design/tasks, and not caught by design.md's "Blast Radius — verified" section because that section's check was a text grep for `%#v`/`GoString` occurrences, not a simulation of this test's reflection-based exhaustive method-set comparison.

**Why this is a real conflict, not a false alarm**: `S-AIP-008`'s Go implementation checked *zero* shared exported-method names between `Completion` and `*Failure`, full stop. `GoString` is now shared by both.

**Why the fix (narrowing the guard, not dropping D-1) is correct, not a silent deviation**:
1. D-1 (`GoString() string { return f.Error() }`) is implemented exactly as locked — unchanged.
2. `GoStringer` is a fixed-name interface (`GoString() string`); there is no alternate spelling `fmt` would recognize for `%#v`, so "pick a different method name" is not an option, and design.md already rejected `fmt.Formatter` for independent reasons (D-1 alternatives-considered).
3. Renaming `Completion.GoString` is out of scope (AI-15's own shipped surface, unrelated to this change, would ripple into every existing caller/test of `Completion`).
4. The scenario's own spec text (`openspec/specs/ai-provider-errors/spec.md:76`) is narrower than the test's implementation: *"no accessor name or return type is shared **in a way that lets a consumer read a category off a completion or a finish reason off a failure**"* — a qualified claim, not an unconditional one. `Error`/`String`/`GoString` each render a **fixed diagnostic string** (`"completion"` for `Completion`, `"provider failure: <category>"` for `Failure` — its own category, not confusable with the other type) — sharing these three names does not let a consumer read either type's data through the other.
5. The fix is narrow and additive: an explicit 3-name exemption map (`Error`, `String`, `GoString`) inside the existing loop. Every other exported name — including `Category`, `FinishReason`, and anything added in the future — is still checked with zero exemption; the two explicit data-accessor assertions (`FinishReason`/`Category`) are untouched.
6. Verified: after the fix, `TestErrorEvent_TerminalExclusivity_NeverBothCompletionAndError` and all its sub-tests (S-AIP-007, the narrowed S-AIP-008, S-AIP-009) pass; the guard's protective power for any *other* (data) accessor name is unchanged — reasoned directly from the code (the exemption map is checked first and `continue`s only for the three literal names; a hypothetical shared `Category` or any other name still fails the loop exactly as before).

This is flagged here explicitly for review, per "don't silently deviate." If reviewers disagree with narrowing an out-of-this-change's-declared-scope test, `git revert b82cfdf` isolates exactly this decision (see Rollback boundary above) without touching AI-41.1.

**Note (recorded at archive)**: `sdd-verify` recommended narrowing this exemption further to `{GoString}` alone (W-1, non-blocking). The orchestrator applied that narrowing in commit `b271637` after this apply-progress snapshot was written — see `archive-report.md` for the final, resolved state.

No other deviations. AI-41.1 (WU-1/WU-2) implements D-2/D-3 exactly as designed, no surprises.

## Issues Found

None beyond the S-AIP-008 collision above (fully resolved and gate-verified).

## Non-Negotiable Assertion Strength — self-check

- Rule-4 test (`TestCheckEmit_PayloadReportsOwnViolation_SurfacedAsRule4`): asserts pointer identity (`violation != planted`, not `errors.Is`/equality), sentinel survival (`errors.Is(err, sentinelRule)`), position equality (`reflect.DeepEqual(violation.Path(), planted.Path())`), AND both negative assertions (`!errors.Is(err, ai.ErrNotInVocabulary)`, `!errors.Is(err, ai.ErrOutOfRange)`), AND the zero-regression accepted case. No bare non-nil check anywhere in this test.
- Nil-receiver sub-test (`TestFailure_GoString_NilReceiver_TotalByDelegation`): asserts equality against `(*ai.Failure)(nil).Error()` computed once as `want`, never a hard-coded string literal; wraps each verb in a `recover()` to prove no panic, across all four verbs (`%v/%s/%+v/%#v`), not just `%#v`.

## Remaining Tasks

- [ ] Phase 4 (A.1–A.5): archive-phase spec/doc promotion — explicitly out of apply's scope, reserved for `sdd-archive` after this PR merges.

## Workload / PR Boundary

- Mode: single PR (exception-ok cached, not invoked — no overrun to except)
- Current work unit: WU-1 through WU-5, all complete
- Boundary: this batch starts from `origin/main @ f2e460d` and ends with `b82cfdf` — two commits, both work-unit-scoped and independently revertible
- Review budget impact: 220 changed lines against a 1000-line session budget (780 remaining); well under the generic 400-line low-risk threshold too
