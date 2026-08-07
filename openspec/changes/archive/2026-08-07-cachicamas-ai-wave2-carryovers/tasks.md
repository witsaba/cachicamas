# Tasks: Discharge the Wave-2 carryovers (AI-41)

> **Change**: `cachicamas-ai-wave2-carryovers` · **Milestone**: AI-41 (doc 0002:2233–2257)
> **Wave**: 5 — Harden · **Module**: `backend/agent/` (layered per ADR 0005 § D1, NOT hexagonal — `cd backend/agent && make test`)
> **Strategy**: single-pr · strict TDD on · stdlib-only · review budget **1000 changed lines** (session-level carryover from the AI-35 amendment; both leaves are far under it)
> **Inputs**: [`design.md`](./design.md) (12 sections, 5 decisions D-1..D-5); specs [`ai-event-envelope`](./specs/ai-event-envelope/spec.md), [`ai-provider-errors`](./specs/ai-provider-errors/spec.md), [`ai-stream-testkit`](./specs/ai-stream-testkit/spec.md); house style [`archive/2026-08-07-cachicamas-ai-retry-policy/tasks.md`](../2026-08-07-cachicamas-ai-retry-policy/tasks.md).
> **Reference work**: Engram obs `#2662` (explore), `#2663` (proposal), `#2665` (spec), `#2666` (design).

---

## Review Workload Forecast

| Field | Value |
| --- | --- |
| Estimated changed lines | **~90–125** (apply phase only; archive-phase spec/doc promotion is separate, not part of the PR diff) |
| 400-line budget risk | **Low** (also well under the session's actual 1000-line budget) |
| Chained PRs recommended | No |
| Suggested split | Single PR |
| Delivery strategy | exception-ok (cached, not invoked — no overrun to except) |
| Chain strategy | pending (not applicable — single small PR) |

```text
Decision needed before apply: No
Chained PRs recommended: No
Chain strategy: pending
400-line budget risk: Low
```

**Per-file estimate** (design.md § 6, unchanged at task time):

| File | Action | Est. lines |
| --- | --- | --- |
| `backend/agent/src/ai/export_test.go` | Modify | ~15–20 |
| `backend/agent/src/ai/event_test.go` | Modify | ~30–45 |
| `backend/agent/src/ai/provider_failure.go` | Modify | ~6–8 |
| `backend/agent/src/ai/provider_failure_test.go` | Modify | ~35–50 |

Two independent leaves (AI-41.1, AI-41.2), both additive, zero new files, zero new imports, `go.mod` untouched. No natural PR seam is worth creating at this size — chained/stacked PRs would add review overhead for no reduction in reviewer load.

### Suggested Work Units

| Unit | Goal | Likely PR | Focused test command | Runtime harness | Rollback boundary |
| --- | --- | --- | --- | --- | --- |
| WU-1/2 | AI-41.1: witness payload controllable failure + rule-4 proof | PR 1 (part) | `go test -race -run 'TestCheckEmit_PayloadReportsOwnViolation' ./src/ai/...` | N/A (in-package unit test) | `git revert` drops `export_test.go`'s field/constructor + the new test; `validate` reverts to unconditional nil |
| WU-3/4 | AI-41.2: redacting `GoString()` on `*ai.Failure` | PR 1 (part) | `go test -race -run 'TestFailure_GoString' ./src/ai/...` | N/A (in-package unit test) | `git revert` drops the `GoString` method + canary test; `%#v` reverts to reflective (latent-leak) rendering |
| WU-5 | Full-suite verification + lint | PR 1 (part) | `cd backend/agent && make test` | N/A | No code change; verification only |

---

## Phase 1: AI-41.1 — Emission-boundary failure path (`R-AEE-021`)

### WU-1 — RED

- [x] 1.1 In `backend/agent/src/ai/export_test.go`, add `rejectWith *Violation` field to `WitnessPayload` (nil default; both existing constructors — `NewWitnessEvent`, `NewTestEvent` — build the struct by composite literal and leave it nil, so every landed test stays unchanged). Add test-only constructor `NewRejectingWitnessEvent(block BlockIndex, reject *Violation) Event`. Leave `validate` returning `nil` unconditionally — RED must stay red.
- [x] 1.2 In `backend/agent/src/ai/event_test.go`, add a new test (name indicative: `TestCheckEmit_PayloadReportsOwnViolation_SurfacedAsRule4`) with `t.Parallel()` (D-3 — registers no test kind; uses the statically registered `KindTestWitness`, whose descriptor is `Role: BlockRoleNone`, so `CheckEmit`'s rule 3 early-exits and only rule 4 can fire). Body:
  - Build `planted := ai.Invalid(sentinelRule, ai.At(...))` (a distinct sentinel/position from `ErrNotInVocabulary`/`ErrOutOfRange`).
  - Build and stamp an event via `ai.NewRejectingWitnessEvent(block, planted)`; call `ai.CheckEmit`.
  - Assert `errors.As(err, &violation)` finds a `*ai.Violation` whose pointer **is** `planted` (identity, not just equal fields).
  - Assert `errors.Is(err, sentinelRule)` is true (through `Violation.Unwrap`).
  - Assert `violation.Path()` equals `planted.Path()` (position intact — spec Q2, R-AEE-021's "same position" clause).
  - Assert `!errors.Is(err, ai.ErrNotInVocabulary)` and `!errors.Is(err, ai.ErrOutOfRange)` — rules 1–3 must **not** have fired (risk 7.3 / S-AEE-072's attributability clause). A bare `err != nil` check does not satisfy this task.
  - Assert the **same event's construction with a default-state payload** (`ai.NewWitnessEvent(block)`, stamped) is accepted: `ai.CheckEmit(...) == nil` — the zero-regression half of S-AEE-072.
- [x] 1.3 Confirm RED: `cd backend/agent && go test -race -run 'TestCheckEmit_PayloadReportsOwnViolation' ./src/ai/...` fails — `CheckEmit` returns `nil` because `validate` is still hard-coded to return `nil`. **Actual RED observed**: `event_test.go:240: ai.CheckEmit(rejecting) = nil, want the planted violation surfaced — validate still returns nil unconditionally` — `--- FAIL: TestCheckEmit_PayloadReportsOwnViolation_SurfacedAsRule4`, exit 1.

### WU-2 — GREEN / REFACTOR

- [x] 2.1 In `export_test.go`, change `validate` to `return w.rejectWith`.
- [x] 2.2 Confirm GREEN: the WU-1 focused command passes. **Actual**: `--- PASS: TestCheckEmit_PayloadReportsOwnViolation_SurfacedAsRule4`, `ok github.com/cachicamas/backend/agent/src/ai`.
- [x] 2.3 Refactor: refresh `validate`'s doc comment (`export_test.go:63–67`) — it currently reads "a rule is added once a test needs one to fail"; that day has now arrived, restate in the present tense. Landed in the same edit as 2.1 (single-line method body + its own doc comment).
- [x] 2.4 Run the full package to confirm zero regressions: `cd backend/agent && go test -race -v ./src/ai/...`. **Actual**: exit 0, all 6 `./src/ai/...` packages `ok`, 0 `--- FAIL` lines.

## Phase 2: AI-41.2 — Redacting failure-payload formatting (`R-AIP-016`)

### WU-3 — RED

- [x] 3.1 In `backend/agent/src/ai/provider_failure_test.go`, add a test-local `type canaryCause string` implementing `error` (`func (c canaryCause) Error() string { return string(c) }`) — a **value-kind** cause, per D-4.
- [x] 3.2 Write a new test that plants **three** canaries: `FailureReport{Category: ..., RawLabel: <canary1>, RequestID: <canary2>, Cause: canaryCause(<canary3>)}`, built via `ai.PreStreamFailure`/`mustPreStreamFailureReport`. Loop `%v`/`%s`/`%+v`/`%#v` (the `content_part_test.go:414–419` shape) asserting none of the three canaries appear in any rendering.
  - **Mandatory placement per D-4**: the cause canary MUST be a value-kind cause (`canaryCause`, not `errors.New(...)`). A pointer-shaped cause (e.g. `fmt.Errorf`-wrapped) renders as a bare hex address under `%#v` today — planting the cause canary only there would make RED pass **before** the fix, a vacuous RED. `RawLabel`/`RequestID` (string struct fields) and the value-kind cause are the three positions that leak today and must all be covered.
- [x] 3.3 Add a nil-receiver sub-test: assert `fmt.Sprintf("%#v", (*ai.Failure)(nil))` **equals** `(*ai.Failure)(nil).Error()` — never a hard-coded string literal — across `%v/%s/%+v/%#v`, and never panics.
- [x] 3.4 Confirm RED: `cd backend/agent && go test -race -run 'TestFailure_GoString' ./src/ai/...` fails on the `%#v` case (string fields + value-kind cause reflected verbatim today); `%v/%s/%+v` already pass via the existing `Error()` dispatch. **Actual RED observed** (verbatim, all three canaries leak under `%#v`, nil case also fails):
  ```
  provider_failure_test.go:1298: fmt.Sprintf("%#v", f) = "&ai.Failure{category:0x9, retryable:false, retryAfter:ai.RetryDelay{delay:0, present:false}, rawLabel:\"raw-label-CANARY-77f3\", statusClass:0, requestID:\"request-id-CANARY-88e1\", cause:\"cause-CANARY-99a2\", delivery:0x1, partialOutput:false}", want "provider failure: unknown"
  provider_failure_test.go:1287: ... contains the planted canary "raw-label-CANARY-77f3" / "request-id-CANARY-88e1" / "cause-CANARY-99a2", want it excluded
  provider_failure_test.go:1323: fmt.Sprintf("%#v", (*ai.Failure)(nil)) = "(*ai.Failure)(nil)", want "no provider failure"
  --- FAIL: TestFailure_GoString_RedactsLikeError / --- FAIL: TestFailure_GoString_NilReceiver_TotalByDelegation, exit 1
  ```
  Confirms D-4 empirically: the value-kind cause (`cause:"cause-CANARY-99a2"`) is reflected verbatim — a pointer-shaped cause would have rendered as a hex address instead, which would have made this RED vacuous.

### WU-4 — GREEN / REFACTOR

- [x] 4.1 In `backend/agent/src/ai/provider_failure.go`, add:
  ```go
  func (f *Failure) GoString() string { return f.Error() }
  ```
  with a doc comment citing `R-AIP-016` and matching the sibling doc-comment style (`completion.go:101–106`, `content_part.go:303`, `event.go:319` — delegate-don't-reflect posture, nil-safety by delegation per D-5, no second nil check).
- [x] 4.2 Confirm GREEN: the WU-3 focused command passes. **Actual**: `--- PASS: TestFailure_GoString_RedactsLikeError`, `--- PASS: TestFailure_GoString_NilReceiver_TotalByDelegation` (all sub-tests), `ok github.com/cachicamas/backend/agent/src/ai`.
- [x] 4.3 Refactor: doc-comment wording polish only, no behavior change. Landed in the same edit as 4.1.
- [x] 4.4 Run the full package to confirm zero regressions: `cd backend/agent && go test -race -v ./src/ai/...`.
  - **Unplanned discovery, resolved in this step**: adding `GoString` to `*Failure` collided by exported-method-name with `Completion.GoString` (both already existed pre-change), tripping the pre-existing, out-of-this-change's-scope terminal-exclusivity guard `TestErrorEvent_TerminalExclusivity_NeverBothCompletionAndError/Failure_and_Completion_export_no_accessor_of_the_same_name_(S-AIP-008)` (R-AIP-003, `provider_failure_test.go`, landed at AI-19, not surfaced by design.md's blast-radius grep since it is a reflection-based exhaustive check, not a text occurrence). Resolved by narrowing that test's disjointness check to exempt exactly `{Error, String, GoString}` — the fixed-text diagnostic renderers, which the scenario's own text says are not disqualifying ("... in a way that lets a consumer read a category off a completion or a finish reason off a failure") — while every data accessor (`Category`, `FinishReason`, and anything added later) is still checked with zero exemption. See apply-progress for full reasoning; flagged as a risk/deviation for review.
  - **Actual**: exit 0, all 6 `./src/ai/...` packages `ok`, 0 `--- FAIL` lines, including the corrected terminal-exclusivity guard.

## Phase 3: Full-suite verification

### WU-5

- [x] 5.1 `cd backend/agent && make test` (`go test -race -v ./...`) — all green, zero regressions across the module. **Actual**: exit 0, 0 `--- FAIL` lines, all 7 module packages `ok` (`agenttest`, `ai`, `ai/internal/retry`, `ai/openaicompat`, `ai/openaicompat/openrouter`, `.../conformance`, `.../smoke`).
- [x] 5.2 `cd backend/agent && make lint` — 0 issues. **Actual**: `go vet ./...` clean, `golangci-lint run --config=.golangci.yml ./...` → `0 issues.`, exit 0.
- [x] 5.3 `cd backend/agent && make build` — compiles cleanly. **Actual**: `go build -trimpath ./...`, exit 0, no output.
- [x] 5.4 Confirm `go.mod` / `go.sum` are byte-identical to the pre-change state (`git diff --stat backend/agent/go.mod backend/agent/go.sum` empty) — both leaves add zero imports. **Actual**: `git diff --stat origin/main -- backend/agent/go.mod backend/agent/go.sum` → empty output, confirmed byte-identical. Overall `git diff --stat origin/main`: 4 files changed, 208 insertions(+), 12 deletions(-) = 220 changed lines (forecast was ~90–125; over-forecast mainly due to the unplanned S-AIP-008 guard fix above) — well within the 1000-line session budget.

---

## Phase 4: Archive-phase work units (NOT part of the apply-phase PR diff)

These run at `sdd-archive`, after the PR above merges and both requirements are proven in code. Landing any of these before the code merges reproduces the exact failure mode AI-41 exists to stop (spec acceptance criteria, `ai-stream-testkit` delta rule 3).

- [x] A.1 Promote `R-AEE-021` (with `S-AEE-071`, `S-AEE-072`) into `openspec/specs/ai-event-envelope/spec.md`, inserted after `R-AEE-020` (currently at line 264), verbatim per `specs/ai-event-envelope/spec.md` in this change. **Done at archive**: inserted before the `Non-functional requirements` heading, verbatim per the delta.
- [x] A.2 Append the dated discharge blockquote to `openspec/specs/ai-event-envelope/spec.md`'s "Carried forward" section (currently `:322–324`) — append-only, exact wording from the delta's "Carried-forward discharge" section. **Done at archive**: existing paragraph preserved byte-for-byte; blockquote appended after it.
- [x] A.3 Promote `R-AIP-016` (with `S-AIP-056`, `S-AIP-057`) into `openspec/specs/ai-provider-errors/spec.md`, inserted after `R-AIP-015` (currently line 225) and before "Non-functional requirements" (currently line 237). **Done at archive**: inserted verbatim per the delta at that exact point.
- [x] A.4 Amend `openspec/specs/ai-stream-testkit/spec.md:39` — append the dated discharge clause to the existing ledger sentence (byte-for-byte preserved), naming date, milestone, change name, and both requirement identifiers (`R-AEE-021`, `R-AIP-016`), exact target text quoted in `specs/ai-stream-testkit/spec.md` of this change. **Done at archive**: line now reads "discharged by AI-41", existing sentences preserved verbatim.
- [x] A.5 Amend `docs/architecture/milestones/0002-cachicamas-ai-layer-1-task-graph.md`: append an "Amended 2026-08-07 — AI-41 close (Wave 5 — Harden, Wave-2 carryovers)" blockquote after the AI-35 close blockquote (currently line 20); bump the shipped counter `36 of 42` → `37 of 42`; update the "**Remaining in Wave 5:**" list to drop AI-41; update the status line at the top of the doc (line 3). **Caution**: AI-41 ships before AI-36/AI-37 — do NOT claim a contiguous "landed AI range through AI-41"; state the shipped counter and the explicit remaining-milestone list only, matching the append-only, non-renumbering discipline the doc's own header rules require (lines 9–10). **Done at archive**: status line now reads "AI-00 through AI-35 plus AI-41 are landed and verified"; shipped counter 37 of 42; Remaining trimmed to AI-36, AI-37; new dated blockquote added recording both subnodes, three commits (`ca2ede7`, `b82cfdf`, `b271637`), the +214/−12 diff, W-1 resolved, W-2 recorded, and AI-36 unblocked. **One tooling limitation noted**: this archive execution had no Bash/shell tool available, so the change-folder move, `git commit`, and the final `make test`/`make lint`/`make build` gate re-run described in the archive plan could not be executed by this phase; see `archive-report.md` for the exact scope completed and what remains for a follow-up with shell access.

---

## Strict TDD ordering summary

| Leaf | RED | GREEN | REFACTOR |
| --- | --- | --- | --- |
| AI-41.1 | WU-1 (field + constructor + rule-4 test; `validate` still nil) | WU-2.1 (`return w.rejectWith`) | WU-2.3 (doc comment) |
| AI-41.2 | WU-3 (canary loop per D-4 + nil-identity sub-test) | WU-4.1 (`GoString` delegate) | WU-4.3 (doc comment) |

Leaves are independent; AI-41.1 then AI-41.2 (W1/W2 order), full gate (`make test -race`, `make lint`) after each per WU-2.4/WU-4.4, and once more at WU-5 for the whole module.

## Risk

| Item | Class | Mitigation |
| --- | --- | --- |
| Rule-4 test could be weakened to a bare non-nil check | CRITICAL if it happens | Task 1.2 pins the exact assertion set (identity, sentinel, position, negative rules 1–3) — `S-AEE-071` is unmet by anything weaker |
| Canary placed only in a pointer-shaped cause | CRITICAL — produces a vacuous RED | Task 3.2 pins the value-kind-cause requirement explicitly (D-4) |
| Nil-receiver assertion hard-coded to a string literal | WARNING — drifts silently if `noProviderFailure`'s text changes | Task 3.3 pins equality against `(*ai.Failure)(nil).Error()` |
| Archive clause written before code lands | WARNING — reproduces AI-41's own target failure mode | Phase 4 explicitly gated as archive-phase, after PR merge |
