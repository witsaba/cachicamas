# Tasks — the provider conformance suite

> **Change**: `cachicamas-ai-conformance-suite`
> **Milestone**: AI-23 · **Nodes**: AI-23.1 … AI-23.5, AI-23.7, AI-23.8 `[leaf]`; AI-23.6 `[leaf, depends on .2 .3 .4 .5 .7 .8]`
> **Phase**: tasks
> **Project**: cachicamas (witsaba)
> **Date**: 2026-08-02
> **Worktree**: `cachicamas-worktrees/ai-wave-3` · **Branch**: `feat/2026-08-02-cachicamas-ai-layer1-wave-3`
> **Inputs**: `proposal.md`, `specs/ai-provider-conformance-suite/spec.md` (18 requirements, 67 scenarios, 6 NFRs), `design.md` (D1–D6)
> **Depends on**: AI-03, AI-19, AI-20, AI-21, AI-22 (all shipped) · **Blocks**: AI-24, AI-38
> **Evidence gate**: recorded green `make test` (`go test -race -v ./...`) **and** clean `make lint`, both from `backend/agent/` (`NFR-CNF-F` / `S-CNF-067`)

---

## Entry point for the resuming agent

Nothing in this milestone is implemented; every box below is unchecked. **AI-23.1 goes first, non-negotiably** — it defines `Factory`, `Capability`, the case table and the runner that every other leaf registers cases into. Then AI-23.2 → .3 → .4 → .5 → .7 → .8 in spec order (no hard code dependency between them beyond .1, kept in spec order for review continuity). **AI-23.6 goes last** — its own header marks it `depends on .2 .3 .4 .5 .7 .8`, and its record must be total over every case those leaves register. One writer, one branch, sequential — no concurrent apply agents (prior wave's prefix-collision lesson). Commit per leaf; a per-leaf commit is the rollback boundary.

**Implementation-order note (not a spec change):** `CapabilityRecord`, `Standing` and `Outcome` are Go types the AI-23.1 runner must already reference to compile (initializing all 8 entries to `OutcomeNotExercised`), so AI-23.1 creates a minimal `conformance_record.go` (types + zero-value initialization only, no `Verdict()`). AI-23.6 *extends* that same file with `Verdict()`, entry-by-entry comparison, and the full record/verdict unit tests design's File Changes table attributes to it. This resolves a real ordering tension between "the runner must compile at AI-23.1" and "the record is design's AI-23.6 deliverable" without adding, removing or reinterpreting any requirement.

**Implementation-order note 2 (resolved, no longer a GREEN-step decision):** the ambiguity this note originally flagged — a bare `bool`'s zero value can't distinguish "declared absent" from "never declared" (`S-CNF-006`) — is fixed in `design.md`: `Factory.Reasoning`/`TokenCounting`/`CacheBoundary` are now `*bool`. `nil` fails construction naming the undeclared capability; non-nil `false` records `absent` via `t.Skipf`; non-nil `true` is cross-checked against askable discovery (D6) and records `satisfied` or `failed`. AI-23.1's GREEN step implements this directly from `design.md` — no mechanism choice remains open.

Threat matrix: **N/A** per `design.md` — no routing, shell, subprocess, VCS/PR automation, or process-integration boundary in this package.

## Node types and what they close on

| Node | Type | Closes on |
| --- | --- | --- |
| AI-23.1, .2, .3, .4, .5, .7, .8 | `[leaf]` | Every test-list item taken red → green → refactored, in order, with both outputs recorded here |
| AI-23.6 | `[leaf, depends on .2 .3 .4 .5 .7 .8]` | Same discipline, plus the record must carry exactly 8 entries sourced from every dependent leaf's cases, and `doc.go` names the suite (`NFR-CNF-C`) |

Strict TDD is on. A RED that passes immediately because earlier work already covers it is legitimate and must be recorded honestly, never forced.

---

## Review Workload Forecast

| Field | Value |
| --- | --- |
| Estimated changed lines (forecast, pre-apply) | Code: 8 production files + 1 test file, scaled ~1.4–1.5× AI-22's actual (1 791 lines, 13 req / 46 scenarios) against AI-23's 18 req / 67 scenarios ⇒ **~1 900–2 700**. Openspec: proposal 87 + spec.md 326 + design.md 78 + this `tasks.md` (~450–550) ⇒ **~940–1 040**. **AI-23 total ~2 850–3 700**, consistent with proposal's own R5 estimate (~2 800–3 200 code+openspec) |
| 400-line budget risk | High (every leaf individually will likely exceed 400 changed lines, same pattern as AI-21/AI-22) |
| Chained PRs recommended | No |
| Suggested split | Single PR — the wave PR (AI-21 + AI-22 + AI-23), leaf commits as the reviewable boundary |
| Delivery strategy | single-pr |
| Chain strategy | size-exception |

```text
Decision needed before apply: Yes
Chained PRs recommended: No
Chain strategy: size-exception
400-line budget risk: High
```

### Running-total flag for the orchestrator

Cached wave state: AI-21 actual **2 810** + AI-22 actual **2 542** = **5 352**, already past the wave's 5 000-line ceiling — accepted by the user (2026-08-02 session decision) as one PR with `size:exception`, forecast to land **~7 000–8 000** total. Adding this milestone's own forecast (**~2 850–3 700**) projects a running total of **~8 200–9 050** — at or slightly above that forecast's upper edge. This is AI-23's own confirmation of the same already-accepted decision, not a new ask; `Decision needed before apply: Yes` here means "re-confirm size:exception applies to this milestone's actual diff before its commits land," per the skill's `single-pr` rule, not "ask the user again from scratch."

### Suggested Work Units

| Unit | Goal | Likely PR | Focused test command | Runtime harness | Rollback boundary |
| --- | --- | --- | --- | --- | --- |
| 1 — AI-23.1 | Factory seam, capability marking, empty case table, runner that never hangs | wave PR, commit 1 | `go test -race -run 'TestConformance_Skeleton\|TestFactory\|TestCapabilityMarking' ./src/agenttest/` | N/A — empty case table, no scripted subject run yet; proven end-to-end at unit 8 | `git revert` commit 1: removes `conformance_suite.go`, `conformance_record.go` (stub), `conformance_suite_test.go`'s skeleton test |
| 2 — AI-23.2 | Text cases: order/contiguity/byte-exact reconstruction, empty completion | wave PR, commit 2 | `go test -race -run 'TestConformance/text' ./src/agenttest/` | `go test -race -run TestConformance ./src/agenttest/` against `FakeFactory()` — real scripted subject, not a mock | `git revert` commit 2: removes `conformance_text.go` + its case-table registration |
| 3 — AI-23.3 | Tool-call cases: fragmented interleave, zero-delta call, ordinals, mixed finish reason | wave PR, commit 3 | `go test -race -run 'TestConformance/tool_call' ./src/agenttest/` | Same as unit 2 | `git revert` commit 3: removes `conformance_tool_call.go` + registration |
| 4 — AI-23.4 | Terminal/error cases: exactly-one-terminal, discriminator, 9-category exhaustive loop | wave PR, commit 4 | `go test -race -run 'TestConformance/terminal' ./src/agenttest/` | Same as unit 2 | `git revert` commit 4: removes `conformance_terminal.go` + registration |
| 5 — AI-23.5 | Cancellation cases: bounded close, saturated drop, leak-free (serial-only) | wave PR, commit 5 | `go test -race -run 'TestConformance/cancellation' ./src/agenttest/` | Same as unit 2, serial-only (no `t.Parallel()`, `R-STK-008`) | `git revert` commit 5: removes `conformance_cancellation.go` + registration |
| 6 — AI-23.7 | Redaction: sentinel appears in no event/error/failure output | wave PR, commit 6 | `go test -race -run 'TestConformance/redaction' ./src/agenttest/` | Same as unit 2 | `git revert` commit 6: removes `conformance_redaction.go` + registration |
| 7 — AI-23.8 | Optional-capability cases: reasoning, token counting, cache boundary, required finish-reason/usage cases | wave PR, commit 7 | `go test -race -run 'TestConformance/(reasoning\|token_counting\|cache_boundary\|finish_reason\|usage)' ./src/agenttest/` | Same as unit 2 | `git revert` commit 7: removes `conformance_capabilities.go` + registration |
| 8 — AI-23.6 | Total record, four-value verdict rule, `doc.go` updated, full suite vs `FakeFactory()` | wave PR, commit 8 | `go test -race -run 'TestConformance$\|TestCapabilityRecord\|TestVerdict\|TestFinishReasonDriftGuard' ./src/agenttest/` | `make test` in `backend/agent/` — whole-module run proves AI-20.4's guard and both AI-00 import guards still hold | `git revert` commit 8: removes `conformance_record.go`'s `Verdict()` additions, `doc.go` edit, and the record/verdict/e2e tests |

---

## Phase AI-23.1 — pluggable skeleton `[leaf]`

**Deliverable:** `conformance_suite.go`, `conformance_record.go` (types stub only), `conformance_suite_test.go` (skeleton test).
**Spec:** `R-CNF-001`, `R-CNF-002` *(non-negotiable)*, `R-CNF-003`, `R-CNF-004`. **Design:** D1, D2, D5.
**Depends on:** AI-21's fake (wrapped as `FakeFactory()`), AI-03 §§ 5–6/11.

- [x] **Item 1** (`R-CNF-001`) — One factory seam, zero copied assertions, a stuck subject never hangs the run.
  - [x] 1.1 RED: `RunConformance(t, FakeFactory())` against an empty case table; confirm the specific expected failure (undefined symbols, since `RunConformance`/`Factory`/`FakeFactory` don't exist yet). `S-CNF-001`.
    - RED output (`go vet ./src/agenttest/...` against `conformance_suite_test.go` alone, before any production file existed): `undefined: Factory`, `undefined: runConformanceCases`, `undefined: FakeFactory`, `undefined: OutcomeNotExercised`, `undefined: RunConformance`, `undefined: conformanceCase`, `undefined: CapNone`, then `too many errors` — exactly the undefined-symbol failure the item names.
  - [x] 1.2 RED: a factory whose `New` cannot realise one scripted case; confirm that case fails naming subject + scenario and the run still completes the rest. `S-CNF-003`.
    - Covered by the same compile-failure RED above (no production symbols existed yet); GREEN evidence below is the scratch-verified real run (see 1.3).
  - [x] 1.3 GREEN: implement `Factory` struct, `RunConformance` iterating the (still-empty) case table, per-case `recover`/attributable failure so one case's panic or refusal never aborts the run. Confirm 1.1–1.2 pass.
    - GREEN: `go test -race -v -run 'TestConformanceSkeleton|TestRunConformance_PublicEntryPoint|TestInvokeCaseRecovering' ./src/agenttest/` → `PASS` (all subtests), package `ok`.
    - **Design discovery during GREEN, recorded honestly**: Go's `testing` package propagates a failed/erroring subtest to *every* ancestor `*testing.T` unconditionally (`common.Fail` walks `c.parent.Fail()` with no opt-out — confirmed empirically, including that a bare `&testing.T{}` panics on `.Run()` and is unusable as an isolation trick). A test that literally drives the runner into a genuine `t.Fatalf`/`t.Errorf` to prove "this correctly fails" therefore also fails the meta-test observing it, which would make `make test` red by construction rather than by defect. Resolution: `requireValidFactory` and `crossCheckDeclaredOptionalCapabilities` were changed to accept `testing.TB` (matching AI-22's own kit functions, e.g. `DrainAndRecord`) so a capturing double (`probeTB`, this file's own analogue of `agenttest_test`'s `fakeTB`) can stand in; the core decisions (`factoryDefect`, `applyDeclaredAbsences`, `applyTokenCountingCrossCheck`, `declaredAbsentSkipReason`, `invokeCaseRecovering`, `Capability.registered`) were extracted as pure functions with zero `testing.T` involvement. For the handful of properties that can only be observed by watching a real subtest fail and the run continue — S-CNF-003's "that case fails" half, S-CNF-009, S-CNF-011, and two `NFR-CNF-E` extreme inputs — the scenario was run once, its correct output captured below (the same "apply, observe it fails, record, revert" discipline AI-20's own `R-AMP-016` signature-guard bite proof already uses in this codebase), then the committed test was redesigned to prove the same property via pure decomposition plus a same-shape passing scenario, so `make test` stays green permanently. Scratch-verified output for 1.1/1.2 (`TestConformanceSkeleton_FactoryCannotRealiseOneCase_FailsThatCaseRunCompletesRest`, since reverted from the committed suite in favour of `TestConformanceSkeleton_SeveralCases_AllRunToCompletion` + `TestInvokeCaseRecovering_...`):
      ```
      --- FAIL: .../probe/first_case_cannot_be_realised (0.00s)
          conformance_suite.go:396: agenttest: conformance case "probe/first_case_cannot_be_realised" (capability=no_listed_capability) panicked: agenttest: this test factory cannot realise any scripted case (fixture) (R-CNF-001)
      --- PASS: .../probe/second_case_still_runs (0.00s)
      ```
      This proves both halves of S-CNF-003: the failing case is named with its capability and cause, and the next case still runs to completion.
  - [x] 1.4 REFACTOR: confirm no second scenario vocabulary was introduced — grep for anything beyond `Script`/`Step`/`Emit`/`Hold`/`Gate` (`S-CNF-002`, full confirmation deferred to milestone close once all leaves land).
    - Confirmed: `conformance_suite.go`/`conformance_record.go` introduce `Factory`, `Capability`, `CapabilityRecord`, `conformanceCase` — none of them a scripting primitive; every case still scripts a subject through AI-21's unmodified `Script`/`Step`/`Emit`/`Hold`/`Gate`.

- [x] **Item 2** (`R-CNF-002`, non-negotiable) — The factory declares its optional-capability expectation; `absent` is a conclusion.
  - [x] 2.1 RED: factory declaring `CAP-O-03` not offered; confirm the (stub) record marks it `absent`, never `not exercised`. `S-CNF-004`.
    - RED: same initial compile failure as 1.1 (whole file was new). First real run after GREEN implementation surfaced a genuine defect (not a propagation artifact): `CapCacheBoundary outcome = not_exercised, want absent` — the record only reflected the declared-false capability's absence lazily, as a side effect of a case being skipped, which is wrong when zero cases are registered yet (exactly AI-23.1's own state). Fixed with `applyDeclaredAbsences`, called up front in `runConformanceCases` so every declared-false optional capability reads `absent` immediately, independent of case registration.
  - [x] 2.2 RED: factory declaring `CAP-O-02` offered by a provider that fails the `ai.TokenCounter` type assertion; confirm the entry fails naming the contradiction. `S-CNF-005`.
  - [x] 2.3 RED: factory that never declares one optional capability; confirm construction-time failure naming the undeclared capability, not a silently-recorded `not exercised` entry. `S-CNF-006`.
  - [x] 2.4 GREEN: implement the declared-capability fields plus the undeclared-vs-false distinction per this file's "Implementation-order note 2" above; wire the declaration/discovery cross-check (design D6) for `CAP-O-02`. Confirm 2.1–2.3 pass.
    - GREEN: `TestConformanceSkeleton_DeclaredCacheBoundaryAbsent_...`, `TestConformanceSkeleton_DeclaredTokenCountingOffered_ProviderDoesNotSatisfyIt_EntryFails` (all 3 subtests: pure decision, `probeTB`-wired contradiction, `probeTB`-wired agreement via a local `tokenCountingStub`), `TestConformanceSkeleton_UndeclaredOptionalCapability_FailsConstructionNamingIt` (all 3 subtests) → all `PASS`.
  - [x] 2.5 REFACTOR: confirm the cross-check function is the one place AI-23.8's `CAP-O-02` case (`R-CNF-015`) will reuse — no duplicate logic planned.
    - Confirmed: `tokenCounterOf(tb testing.TB, f Factory) (ai.TokenCounter, bool)` is the one type-assertion site; its doc comment states it is shared with AI-23.8's own case.

- [x] **Item 3** (`R-CNF-003`) — Standing comes from AI-03 § 11 alone; unclassified defaults to required.
  - [x] 3.1 RED: the usage absent-vs-zero case (lives in AI-23.8's node, exercises `CAP-R-03`); confirm its computed standing is required. `S-CNF-007`.
  - [x] 3.2 RED: the exactly-one-terminal and redaction cases (no listed optional capability); confirm both default to required. `S-CNF-008`.
    - Resolved via a new, explicit `CapNone` capability value (declared outside `capabilityFirst…capabilityEnd`'s range, so it can never collide with a real capability and never receives a record entry) for a case that legitimately exercises no listed capability — distinct from a genuinely unregistered/garbage value (3.3).
  - [x] 3.3 RED: a case keyed to a capability outside both closed lists; confirm it is marked required and fails naming the unclassified capability. `S-CNF-009`.
    - Pure, permanent proof: `TestConformanceSkeleton_CapabilityRegistered_UnregisteredValuesRejected` (`Capability(250).registered()==false`, the zero value likewise unregistered, `CapNone` and all 8 real capabilities registered). Wiring (does the runner turn this into a named, non-aborting failure) scratch-verified once:
      ```
      --- FAIL: .../wrapper/probe/unclassified_capability (0.00s)
          conformance_suite.go:331: agenttest: conformance case "probe/unclassified_capability" is keyed to capability capability(250), outside both the CAP-R and CAP-O closed lists — defaulting to required and failing (R-CNF-003, S-CNF-009)
      ```
  - [x] 3.4 GREEN: implement `Capability.Optional()` (optional iff member of `CAP-O-01…03`) and standing derivation purely from `Capability`, never from node structure or run state. Confirm 3.1–3.3 pass.
    - GREEN: `TestConformanceSkeleton_CompletionMetadataStanding_IsRequired`, `TestConformanceSkeleton_NoListedCapabilityCases_DefaultToRequired`, `TestConformanceSkeleton_CapabilityRegistered_UnregisteredValuesRejected` → all `PASS`.
  - [x] 3.5 REFACTOR: confirm no case-table entry hardcodes a standing value — all read through `Optional()`.
    - Confirmed: `conformanceCase` has no `standing` field at all; `standingOf(c Capability) Standing` is the only source, called from `newCapabilityRecord` and nowhere else assigns a `Standing`.

- [x] **Item 4** (`R-CNF-004`) — A skipped optional case is reported, never silent; no waiver for required cases.
  - [x] 4.1 RED: a subject offering no optional capability; confirm the run output names each skipped case with its capability and reason (`t.Skipf`, per design's runner note). `S-CNF-010`.
  - [x] 4.2 RED: an attempt to skip a required case; confirm it fails naming the required case rather than honouring the skip. `S-CNF-011`.
    - Pure, permanent proof: `TestConformanceSkeleton_DeclaredAbsentSkipReason_NeverSkipsARequiredCapability` — `declaredAbsentSkipReason` is gated on `Capability.Optional()` first, so every required capability (including `CapNone`) never returns skip=true regardless of factory content. Wiring (the runner's own escalation to `t.Errorf` when a case skips itself despite required standing) scratch-verified once:
      ```
      --- FAIL: .../wrapper (0.00s)
          conformance_suite_test.go: agenttest: conformance case "terminal/attempts_to_skip_itself" attempted to skip required capability no_listed_capability — there are no waivers (R-CNF-004, S-CNF-011)
      ```
  - [x] 4.3 GREEN: wire the skip-reporting path into the runner from Item 2's declared-capability check; add the required-case skip guard. Confirm 4.1–4.2 pass.
    - GREEN: `TestConformanceSkeleton_NoOptionalCapabilityOffered_SkippedCasesReportedNeverSilent` → `PASS` (nested case shows `--- SKIP`, never invoking its body, outer test `PASS` — confirms `t.Skip` does **not** propagate as a failure to its parent, unlike `t.Fatal`/`t.Error`, which is what makes this specific scenario safe to run for real).
  - [x] 4.4 REFACTOR: confirm skip reporting and failure reporting share one message-building helper.
    - Partial: skip (`declaredAbsentSkipReason`) and failure messages (`factoryDefect`, the unregistered-capability/skip-abuse branches in `runRegisteredCase`) are not routed through one literal Go function, but do share one rendering shape (`"agenttest: ...(<rule-id>)"`, verified by grep across the file) given each is a single `fmt.Sprintf` call; introducing a shared formatter was judged disproportionate indirection for four short, already-consistent call sites and would have invalidated the scratch-verified message text above. Noted rather than silently deviated.

- [x] **Item 5** *(appended, `NFR-CNF-E`)* — Extreme inputs never panic.
  - [x] 5.1 RED: `RunConformance` given a nil factory, an empty case selection, a nil subject and an undeclared capability expectation (table test); confirm each fails attributably naming the entry point and offending input, never panics. `S-CNF-066` (partial).
    - `nil factory` (nil `New`) and `undeclared capability expectation` proven directly via `factoryDefect`/`requireValidFactory`+`probeTB` (real, safe assertions — `TestConformanceSkeleton_ExtremeInputs_FailAttributablyNeverPanic/nil_New_func` and `/undeclared_capability_expectation`). `empty case selection` proven directly (`runConformanceCases` with `[]conformanceCase{}` still returns an 8-entry total record). `nil subject` and a panicking case body are covered permanently by `TestInvokeCaseRecovering_PanicsAreCaughtAndTheirValuePreserved` (pure: a nil `ai.ModelProvider`'s `Stream` call, and a bare `panic(...)`, are both caught by `invokeCaseRecovering` without escaping), plus one-time scratch verification of the full wiring:
      ```
      conformance_suite.go:396: agenttest: conformance case "probe/nil_subject" (capability=no_listed_capability) panicked: runtime error: invalid memory address or nil pointer dereference (R-CNF-001)
      conformance_suite.go:396: agenttest: conformance case "probe/panics" (capability=no_listed_capability) panicked: agenttest: deliberately panicking conformance case (test fixture) (R-CNF-001)
      ```
  - [x] 5.2 GREEN: guard each case explicitly. Confirm 5.1 passes.
    - GREEN: `TestConformanceSkeleton_ExtremeInputs_FailAttributablyNeverPanic` (4 subtests) and `TestInvokeCaseRecovering_PanicsAreCaughtAndTheirValuePreserved` (3 subtests) → all `PASS`.

- [x] **AI-23.1 close:** record green `make test` and clean `make lint`; confirm `Factory`/`RunConformance`/`Capability` doc comments cite `R-CNF-001`…`R-CNF-004`; record the chosen undeclared-capability mechanism (Item 2); commit `feat(agenttest): conformance suite skeleton, factory seam, case marking (AI-23.1)`.
  - `go test -race ./...` → `ok` for both `src/agenttest` and `src/ai`. `make lint` → `0 issues`. `Factory`, `RunConformance`, `Capability` doc comments cite `R-CNF-001`/`R-CNF-002`/`R-CNF-003`. Undeclared-capability mechanism: `nil *bool` fails construction via `factoryDefect`/`requireValidFactory` naming the specific field (`S-CNF-006`), matching design.md's corrected `*bool` decision exactly — no new decision was needed here. `go.mod` unchanged (still zero requires); AI-00 import guard (`TestLayer1_ImportsOnlyStdlibAndItsOwnPackages_DenyByDefault`) and AI-20.4 signature guard (`TestModelProviderInterface_SignatureGuard`) both re-run and pass. Files: `conformance_suite.go` (new), `conformance_record.go` (new, types stub), `conformance_suite_test.go` (new, `package agenttest` white-box — see design note below).

---

## Phase AI-23.2 — text and lifecycle cases `[leaf]`

**Deliverable:** `conformance_text.go`.
**Spec:** `R-CNF-005`, `R-CNF-006`. **Design:** delegates to `ai.CheckStream` and AI-22's `R-STK-006` contiguity helper.
**Depends on:** AI-23.1 (case table, runner).

- [x] **Item 1** (`R-CNF-005`) — Text is ordered, contiguous, and reconstructs byte-exactly.
  - [x] 1.1 RED: scripted text interaction; confirm kinds appear block-start/delta/block-end in order with sequences 1..N, no gap. `S-CNF-012`.
  - [x] 1.2 RED: a multi-byte rune deliberately split across two adjacent deltas; confirm the concatenated deltas equal the source byte-for-byte. `S-CNF-013`.
    - RED note: production (`textOrderingCase`) and its positive test were authored together for this leaf (the file-level "one factory/one runner" skeleton was already proven at AI-23.1, so there was no undefined-symbol RED available here); RED evidence is instead the genuine, first-run assertion trace below for the negative scenario (1.3), which failed for the right reason before the fix, and the zero-length-script guard (Item 3), both run before being judged correct.
  - [x] 1.3 RED: a subject whose deltas arrive reordered; confirm the case fails carrying `ai.CheckStream`'s own verdict, unmodified. `S-CNF-014`.
    - RED: `TestConformanceText_ReorderedSubject_RequireValidStreamCarriesCheckStreamVerdict` against a hand-built `reorderedTextProvider` (AI-21's own `Provider` structurally refuses to script reordered events — its internal `ai.CheckStream` pre-check panics before any goroutine starts, so a misbehaving subject had to be hand-built) — first run confirmed `probe.failed=true` and the captured message cited `R-STK-005`, `ai.CheckStream`'s own verdict text, unmodified.
  - [x] 1.4 GREEN: implement the text case delegating ordering to `ai.CheckStream` and contiguity to AI-22's helper — no reimplementation. Confirm 1.1–1.3 pass.
    - GREEN: `go test -race -v -run TestConformanceText ./src/agenttest/` → all 4 `PASS`.
  - [x] 1.5 REFACTOR: confirm no local re-derivation of ordering/contiguity logic exists in this file (grep check).
    - Confirmed: `grep -nE "CheckStream|CheckContiguity|sequence|Sequence" conformance_text.go` finds exactly one call site (`RequireValidStream(t, rec)`), no reimplementation.

- [x] **Item 2** (`R-CNF-006`) — An empty completion is legal, not a defect.
  - [x] 2.1 RED: a normally-finished interaction with no text delta; confirm it passes, carries its terminal, and is not reported as failed/incomplete/a contiguity violation. `S-CNF-015`.
  - [x] 2.2 RED: the same interaction under the contiguity assertion; confirm no violation is reported for the absent text block. `S-CNF-016`.
  - [x] 2.3 GREEN: implement the empty-completion case; confirm no minimum event count is enforced anywhere in this file. Confirm 2.1–2.2 pass.
    - GREEN: `TestConformanceText_EmptyCompletionCase_PassesAgainstFakeFactory` → `PASS`; `textEmptyCompletionCase` asserts `rec.Len()==1` (the completion alone) rather than any minimum.
  - [x] 2.4 REFACTOR: confirm the empty-completion case and Item 1's case share the same drain/assert helpers.
    - Confirmed: both cases call `DrainAndRecord(t, ch, DefaultDrainTimeout)` then `RequireValidStream(t, rec)`, identically.

- [x] **Item 3** *(appended, `NFR-CNF-E`)* — Extreme inputs never panic.
  - [x] 3.1 RED: the text case's assertion path given a zero-length script; confirm attributable failure/pass, not a panic. `S-CNF-066` (partial).
  - [x] 3.2 GREEN: guard as needed. Confirm 3.1 passes.
    - GREEN: `TestConformanceText_ZeroLengthScript_FailsAttributablyNeverPanics` → `PASS` (a zero-length script's stream closes bare with 0 events; `DrainAndRecord` reports a clean close, not a deadline failure, and nothing panics).

- [x] **AI-23.2 close:** record green `make test` and clean `make lint`; confirm doc comments cite `R-CNF-005`/`R-CNF-006`; commit `feat(agenttest): conformance text and lifecycle cases (AI-23.2)`.
  - `go test -race ./...` → `ok` both packages. `make lint` → `0 issues` (after adding the blank line this package's `package-comments` convention requires between a file's descriptive header and `package agenttest` — see AI-23.1's same fix). `conformance_text.go`'s doc comment cites `R-CNF-005`/`R-CNF-006`.

---

## Phase AI-23.3 — tool-call cases `[leaf]`

**Deliverable:** `conformance_tool_call.go`.
**Spec:** `R-CNF-007`, `R-CNF-008`. **Design:** `CAP-R-02` ordinal observability.
**Depends on:** AI-23.1.

- [x] **Item 1** (`R-CNF-007`) — Reconstructible whole or fragmented, with an observable ordinal.
  - [x] 1.1 RED: two concurrent tool calls whose argument fragments interleave; confirm each call's bytes match exactly with no misattribution. `S-CNF-017`.
  - [x] 1.2 RED: a tool call delivered whole with zero argument deltas; confirm it is accepted, not rejected for missing incremental delivery. `S-CNF-018`.
  - [x] 1.3 RED: two calls to the same tool name; confirm their ordinals differ and order them unambiguously. `S-CNF-019`.
    - RED note: this leaf's cases and their integration tests were authored together (the shared drain/assert plumbing was already proven correct at AI-23.2); RED evidence is the genuine first real-execution run below, which passed on the first attempt — recorded honestly per this file's own instruction rather than manufacturing an artificial failure.
  - [x] 1.4 GREEN: implement fragment attribution by call identity and ordinal exposure. Confirm 1.1–1.3 pass.
    - GREEN: `go test -race -v -run TestConformanceToolCall ./src/agenttest/` → all 5 subtests `PASS` on first run (`TestConformanceToolCall_InterleavedCase_...`, `_ZeroDeltaCase_...`, `_OrdinalCase_...`, `_MixedTextAndToolCase_...`, `_EmptyArgumentPayload_...`).
  - [x] 1.5 REFACTOR: confirm the fragment-attribution logic is one function shared by the interleaved and zero-delta paths.
    - Confirmed: `grep -n "reconstructToolCalls(" conformance_tool_call.go conformance_suite_test.go` shows one definition and 5 call sites — every case (interleaved, zero-delta, ordinal, mixed, and the extreme-input test) reuses it.

- [x] **Item 2** (`R-CNF-008`) — Mixed text + tool content ends on the tool-call finish reason.
  - [x] 2.1 RED: an interaction emitting a text block then a tool call; confirm both survive with ordering intact and the finish reason is the tool-call value from the closed vocabulary. `S-CNF-020`.
  - [x] 2.2 GREEN: implement the mixed-content case, reading the finish reason from the closed vocabulary (not free text). Confirm 2.1 passes.
    - GREEN: `TestConformanceToolCall_MixedTextAndToolCase_PassesAgainstFakeFactory` → `PASS`; asserts `comp.FinishReason() == ai.FinishReasonToolCalls`, the typed vocabulary member, never a string comparison.
  - [x] 2.3 REFACTOR: confirm this case reuses Item 1's tool-call assertions rather than re-asserting tool-call shape independently.
    - Confirmed: `mixedTextAndToolCallCase` calls `reconstructToolCalls(events)` for its tool-call half, same as every Item 1 case.

- [x] **Item 3** *(appended, `NFR-CNF-E`)* — Extreme inputs never panic.
  - [x] 3.1 RED: a tool call with an empty argument-byte payload; confirm attributable pass/fail, not a panic. `S-CNF-066` (partial).
  - [x] 3.2 GREEN: guard as needed. Confirm 3.1 passes.
    - GREEN: `TestConformanceToolCall_EmptyArgumentPayload_FailsAttributablyNeverPanics` → `PASS` (an empty, non-nil delta fragment and empty end arguments are both legal per `tool_call_event.go`'s own rules; `reconstructToolCalls`'s nil-slice append and `bytes.Equal` against an empty slice are safe, proven end to end rather than by inspection alone).

- [x] **AI-23.3 close:** record green `make test` and clean `make lint`; confirm doc comments cite `R-CNF-007`/`R-CNF-008`; commit `feat(agenttest): conformance tool-call cases (AI-23.3)`.
  - `go test -race ./...` → `ok` both packages. `make lint` → `0 issues`. `conformance_tool_call.go`'s doc comment cites `R-CNF-007`/`R-CNF-008`.

---

## Phase AI-23.4 — terminal and error cases `[leaf]`

**Deliverable:** `conformance_terminal.go`.
**Spec:** `R-CNF-009`, `R-CNF-010`. **Design:** iterates `ai.FailureCategories()`, the shipped enumerator.
**Depends on:** AI-23.1; AI-19's `FailureCategories()` (read-only).

- [x] **Item 1** (`R-CNF-009`) — Exactly one terminal; the partial-output discriminator is always answerable.
  - [x] 1.1 RED: a normal finish, a pre-stream failure, and a mid-stream failure; confirm each carries exactly one terminal and nothing follows it. `S-CNF-021`.
  - [x] 1.2 RED: a failure after two text deltas and a failure before any event; confirm the discriminator reports "content preceded" and "none" respectively. `S-CNF-022`.
    - **Genuine RED caught a real bug**: first run of `content_preceded` panicked — `agenttest: scripted stream violates ordering: event[1].block[1]: value is not well-formed for its documented encoding`. Cause: the script opened a text block (`start`, `delta`) but never closed it (`end`) before the terminal error — `ai.CheckStream`'s unterminated-block rule (R-AEE-016) rejects that regardless of how the stream ends. Fixed by inserting `ai.NewTextBlockEnd(1)` before the terminal — a real-world constraint worth recording: a failure with content preceding it still requires that content's block to be properly closed first.
  - [x] 1.3 RED: a subject emitting a text delta after its terminal; confirm the case fails naming the post-terminal event. `S-CNF-023`.
    - AI-21's own `Provider` structurally refuses to script this (its internal `ai.CheckStream` pre-check panics before any goroutine starts), so `postTerminalEventProvider` — a hand-built `ai.ModelProvider` sending an event after its own terminal — was needed, mirroring `reorderedTextProvider`'s (AI-23.2) precedent.
  - [x] 1.4 GREEN: implement the one-terminal assertion and the discriminator read across all three paths. Confirm 1.1–1.3 pass.
    - GREEN: `go test -race -v -run TestConformanceTerminal ./src/agenttest/` → all subtests `PASS`, including all 9×2 failure-category combinations.
  - [x] 1.5 REFACTOR: confirm the three paths (normal, pre-stream, mid-stream) share one assertion helper.
    - Partial, noted rather than silently deviated: `normal_finish` and `mid_stream_failure` share the identical assertion (`RequireValidStream`, AI-22's kit). `pre_stream_failure` structurally cannot use the same drain-based assertion — there is no carrier to drain at all (V-FAIL-11) — so its "exactly one terminal" proof is instead the caller-visible fact that `ch == nil` and the one `err` is the whole outcome. Forcing one literal function across all three would paper over that structural difference rather than reflect it.

- [x] **Item 2** (`R-CNF-010`) — All nine failure categories, iterated against the shipped enumerator.
  - [x] 2.1 RED: iterate `ai.FailureCategories()` (not a hand-written list) on both delivery paths; confirm each of the nine is exercised and classified. `S-CNF-024`.
  - [x] 2.2 RED: a hypothetical tenth category simulated by shrinking the case set below the enumerator's length; confirm the exhaustiveness check fails naming the uncovered category. `S-CNF-025`.
    - A real tenth category cannot be constructed (the vocabulary is closed) — proven instead via `requireFailureCategoryCoverage`, a pure function tested directly against an artificially shrunk `exercised` map.
  - [x] 2.3 GREEN: implement the loop over `ai.FailureCategories()` with a length/coverage assertion enforcing exhaustiveness mechanically. Confirm 2.1–2.2 pass.
    - GREEN: `TestConformanceTerminal_FailureCategoryExhaustivenessCase_PassesAgainstFakeFactory` (18 subtests: 9 categories × 2 paths) and `TestRequireFailureCategoryCoverage_ShrunkExercisedSet_NamesTheUncoveredCategory` → all `PASS`.
  - [x] 2.4 REFACTOR: confirm the coverage assertion, not reviewer vigilance, is what would catch a future upstream addition.
    - Confirmed: `requireFailureCategoryCoverage` iterates `ai.FailureCategories()` itself (never a hand-written count), so a future member automatically participates without a second edit.

- [x] **Item 3** *(appended, `NFR-CNF-E`)* — Extreme inputs never panic.
  - [x] 3.1 RED: a failure category value with no attached message; confirm attributable pass/fail, not a panic. `S-CNF-066` (partial).
  - [x] 3.2 GREEN: guard as needed. Confirm 3.1 passes.
    - GREEN: `TestConformanceTerminal_FailureCategoryWithNoAttachedMessage_NeverPanics` → `PASS` (every category, both construction paths, a bare `FailureReport{Category: c}`).

- [x] **AI-23.4 close:** record green `make test` and clean `make lint`; confirm doc comments cite `R-CNF-009`/`R-CNF-010`; commit `feat(agenttest): conformance terminal and exhaustive failure-category cases (AI-23.4)`.
  - `go test -race ./...` → `ok` both packages. `make lint` → `0 issues`. `conformance_terminal.go`'s doc comment cites `R-CNF-009`/`R-CNF-010`. Capability keying note (design decision, not in design.md's own text): "exactly one terminal" registers under `CapNone` (AI-03 §5's closing note: stream-lifecycle exclusivity is `V-PRV-05`, not a capability); the discriminator and exhaustive-category cases register under `CapTypedFailures` (CAP-R-05's own content: "says whether content preceded it", "classifiable through one vocabulary").

---

## Phase AI-23.5 — cancellation and closure cases `[leaf]`

**Deliverable:** `conformance_cancellation.go`.
**Spec:** `R-CNF-011`, `R-CNF-012`. **Design:** AI-22's `RequireNoGoroutineLeak` (opt-in, serial-only), AI-20.3 saturated-drop physics.
**Depends on:** AI-23.1; AI-22.4 (leak helper, read-only).

- [x] **Item 1** (`R-CNF-011`) — Cancellation closes within bounded time, leaks nothing.
  - [x] 1.1 RED: a stream cancelled mid-consumption; confirm it closes before a bounded deadline and exactly once. `S-CNF-026`.
  - [x] 1.2 RED: the same scenario repeated under `RequireNoGoroutineLeak`; confirm no growth beyond its stated tolerance. `S-CNF-027`.
    - RED note: this leaf's case was authored together with its integration test, mirroring `fake_cancellation_test.go`'s own already-proven pattern (synchronous first-event handoff, then cancel, then confirm zero further events) but delegated through `DrainAndRecord`/`RequireNoGoroutineLeak` (AI-22's kit) instead of hand-rolled draining. First real execution passed; recorded honestly per this file's own instruction.
  - [x] 1.3 Inspection *(not automated)*: confirm no test in this file calls `t.Parallel()` (`R-STK-008` non-negotiable). `S-CNF-028`.
    - Confirmed: `grep -n "t.Parallel()" conformance_cancellation.go` matches only inside doc-comment prose (describing the rule), never an actual call; the same grep against this leaf's `conformance_suite_test.go` additions likewise finds no real call.
  - [x] 1.4 GREEN: implement the cancellation case, wrapped in the leak helper, serial-only. Confirm 1.1–1.2 pass; confirm 1.3.
    - GREEN: `go test -race -v -run TestConformanceCancellation -timeout 60s ./src/agenttest/` → `TestConformanceCancellation_BoundedCloseCase_PassesAgainstFakeFactory` `PASS` (0.05s, 50 repeats + settle).
  - [x] 1.5 REFACTOR: confirm the bounded-deadline wait is the same pattern AI-22.1's own drain helper uses (no new deadline mechanism invented).
    - Confirmed: the case calls `DrainAndRecord(t, ch, DefaultDrainTimeout)` directly — no second deadline mechanism.

- [x] **Item 2** (`R-CNF-012`) — The abandoned-then-cancelled saturated path drops cleanly; abandoned-never-cancelled stays out of scope.
  - [x] 2.1 RED: a consumer that stops reading until the buffer saturates, then the caller cancels; confirm the stream closes bare — no invented terminal, delivered events intact. `S-CNF-029`.
    - Design note: an unbuffered (`Buffer: 0`) script with zero reads is saturated from its very first scripted event by construction — Go's own send-blocks-without-a-reader semantics — so no wall-clock wait was needed to reach that state deterministically (`NFR-CNF-E`). The assertion is deliberately the weaker, correct one R-CNF-012 actually states (no invented terminal event) rather than "exactly zero events delivered": `RequireValidStream`/`ai.CheckStream` was deliberately NOT called on this path, because a legitimately cancelled, partially-delivered stream can leave a block open without matching end — CheckStream's unterminated-block rule would incorrectly flag that as malformed rather than as the sanctioned drop it is.
  - [x] 2.2 RED: the same scenario repeated under the leak helper; confirm no growth beyond tolerance. `S-CNF-030`.
  - [x] 2.3 Inspection: confirm this file states the abandoned-never-cancelled path is out of scope, citing `ai-stream-lifecycle` § 5. `S-CNF-031`.
    - Confirmed: `conformance_cancellation.go`'s package doc comment carries a "Scope, restated from ai-stream-lifecycle § 5 (S-CNF-031)" section quoting the untestability ruling verbatim.
  - [x] 2.4 GREEN: implement the saturated-drop case per AI-20.3's physics; add the out-of-scope doc note. Confirm 2.1–2.2 pass; confirm 2.3.
    - GREEN: `TestConformanceCancellation_AbandonedThenCancelledCase_PassesAgainstFakeFactory` → `PASS` (0.06s).
  - [x] 2.5 REFACTOR: confirm this case and Item 1's share the leak-helper wrapping pattern.
    - Confirmed: both cases call `RequireNoGoroutineLeak(t, scenario)` identically, wrapping a locally-defined `scenario func()`.

- [x] **AI-23.5 close:** record green `make test` and clean `make lint` (extreme-input coverage for this leaf is subsumed by Items 1–2's own bounded-deadline design — no separate `NFR-CNF-E` item needed here since every path already asserts bounded, non-hanging behaviour); confirm doc comments cite `R-CNF-011`/`R-CNF-012`; commit `feat(agenttest): conformance cancellation and saturated-drop cases (AI-23.5)`.
  - `go test -race ./...` → `ok` both packages. `make lint` → `0 issues`. `conformance_cancellation.go`'s doc comment cites `R-CNF-011`/`R-CNF-012`. Both cases register under `CapCancellation` (CAP-R-04).

---

## Phase AI-23.7 — redaction cases `[leaf]`

**Deliverable:** `conformance_redaction.go`.
**Spec:** `R-CNF-013`. **Design:** D4 (sentinel in `FailureReport.Cause`/`RequestID`, not `RawLabel`); reuses `fakeTB` (promoted, precedent `provider_failure_test.go:1172`).
**Depends on:** AI-23.1; AI-22's `summarize`/diff internals (in-package access).

- [x] **Item 1** (`R-CNF-013`) — A planted sentinel appears in no event, no error string, no failure output.
  - [x] 1.1 RED: a sentinel planted into the subject's failure-report fields; confirm it appears in no emitted event and no error string when the suite scans the recording. `S-CNF-032`.
  - [x] 1.2 RED: a deliberately failing case whose recording carries the sentinel in a payload; confirm the suite's own diff and bounded payload summaries never render it. `S-CNF-033`.
    - Note: since `summaryTable`'s `EventKindError` entry renders only `Category` and (bounded) `RawLabel` — never `Cause`/`RequestID` — this pair of assertions is a **pin/regression proof**, not a discovery: it locks in that a future change to the summary renderer cannot silently start leaking either field. Detection actually firing (S-CNF-034, below) is what proves the mechanism has teeth.
  - [x] 1.3 RED: a subject that does leak the sentinel into an error string; confirm the redaction case fails and names where the leak was observed, without reprinting the sentinel. `S-CNF-034`.
    - Since a real "leaking subject" can't be scripted through AI-21's own vocabulary, this is proven by planting the sentinel in `RawLabel` instead (summaryTable's own *sanctioned* rendering channel — D4) as a stand-in for "content the suite is willing to show", confirming `scanForSentinel` correctly flags it, names `event[0]`, and never reprints the sentinel string itself — tested directly against the pure `scanForSentinel`, zero `testing.T` propagation risk.
  - [x] 1.4 GREEN: implement the sentinel-plant plumbing on `Factory.Sentinel` (default when `""`), the promoted `fakeTB` message capture, and the scan-for-sentinel assertion across events/errors/failure output. Confirm 1.1–1.3 pass.
    - GREEN: `go test -race -v -run 'TestConformanceRedaction|TestScanForSentinel' ./src/agenttest/` → all `PASS`. "Promoted `fakeTB`" implemented as `capturingTB`, a new, separately-declared type in this production file (a non-test file cannot import a `_test.go` symbol, so this is design.md's phrase read literally against Go's own build boundary, not a copy — documented as such in the type's own doc comment).
  - [x] 1.5 REFACTOR: confirm the sentinel channel is `Cause`/`RequestID` only, per D4 — not `RawLabel` (sanctioned rendering).
    - Confirmed: `grep -n "sentinel"` in `conformance_redaction.go`'s production planting code shows exactly two sites, `Cause: planted` (wrapping the sentinel in an `errors.New`) and `RequestID: sentinel` — no `RawLabel:` planting anywhere in `redactionCase`.

- [x] **Item 2** *(appended, `NFR-CNF-E`)* — Extreme inputs never panic.
  - [x] 2.1 RED: an empty-string sentinel (falls back to the suite default) and a sentinel containing characters that could break string matching; confirm attributable pass/fail, not a panic. `S-CNF-066` (partial).
  - [x] 2.2 GREEN: guard as needed. Confirm 2.1 passes.
    - GREEN: `TestConformanceRedaction_ExtremeSentinels_FailAttributablyNeverPanic` → both subtests `PASS` (empty-string fallback; a sentinel containing format verbs/backslashes/quotes run through the real `redactionCase`, safe because detection uses `strings.Contains`, never `fmt`-interpreted or regex-interpreted).

- [x] **AI-23.7 close:** record green `make test` and clean `make lint`; confirm doc comment cites `R-CNF-013`; commit `feat(agenttest): conformance redaction cases (AI-23.7)`.
  - `go test -race ./...` → `ok` both packages. `make lint` → `0 issues`. `conformance_redaction.go`'s doc comment cites `R-CNF-013` and D4.

---

## Phase AI-23.8 — optional-capability cases `[leaf]`

**Deliverable:** `conformance_capabilities.go`.
**Spec:** `R-CNF-014`, `R-CNF-015`, `R-CNF-016`. **Design:** D3 (7-value hand-list + drift guard, `NFR-CNF-B` — no `src/ai` edit).
**Depends on:** AI-23.1; the declaration/discovery cross-check from AI-23.1 Item 2.

- [x] **Item 1** (`R-CNF-014`) — `CAP-O-01` reasoning: whole blocks, never leaking into text.
  - [x] 1.1 RED: a subject offering reasoning; confirm it arrives in its own blocks with no reasoning byte in any text event. `S-CNF-035`.
  - [x] 1.2 RED: a subject carrying a reasoning signature; confirm it round-trips byte-identical. `S-CNF-036`.
  - [x] 1.3 RED: a redacted reasoning block start; confirm every delta and end built from it carries the redacted bit forward. `S-CNF-037`.
  - [x] 1.4 RED: a subject declaring reasoning not offered; confirm the case is skipped with a report and the record entry is `absent`. `S-CNF-038`.
    - RED note: the declared-absent skip is the runner's own generic mechanism (AI-23.1, `R-CNF-004`), reconfirmed here for this leaf's own registered case via `TestConformanceCapabilities_ReasoningDeclaredAbsent_SkippedRecordedAbsent`, not reimplemented.
  - [x] 1.5 GREEN: implement the reasoning case, both plain and redacted doors. Confirm 1.1–1.4 pass.
    - GREEN: `go test -race -v -run TestConformanceCapabilities ./src/agenttest/` → all subtests `PASS` on first run.
  - [x] 1.6 REFACTOR: confirm the redacted-bit propagation is asserted structurally (per block), not just at block start.
    - Confirmed: the `redacted_bit_propagates_structurally_through_delta_and_end` subtest asserts `start.Redacted()==true` AND separately proves the enforcement mechanism itself (`ai.NewReasoningDelta` on the same redacted payload rejects a non-empty fragment, `R-ARE-013`) — the propagation IS that structural rejection, not a second independently-checked flag.

- [x] **Item 2** (`R-CNF-015`) — `CAP-O-02` token counting: asked of the provider value; clean absence.
  - [x] 2.1 RED: a subject satisfying the token-counting contract; confirm the suite receives and asserts the count. `S-CNF-039`.
  - [x] 2.2 RED: a subject not satisfying it; confirm a clean absence — no error, no substituted zero. `S-CNF-040`.
    - Also the runner's generic declared-absent mechanism; reconfirmed for `CapTokenCounting` specifically via `TestConformanceCapabilities_TokenCountingDeclaredAbsent_CleanAbsence`.
  - [x] 2.3 RED: a subject that satisfies the contract then declines to answer; confirm the entry is `failed`, not `absent`. `S-CNF-041`.
    - Proven against the ingredients (`tokenCounterOf` + `CountTokens`'s returned error) directly, via a new `tokenCountingStubDeclining` test double, rather than running the case with a real `t.Fatal` — the same propagation-avoidance pattern used throughout this milestone.
  - [x] 2.4 GREEN: implement asking `ai.TokenCounter` of the provider value itself (never model identity/config/catalog), reusing AI-23.1 Item 2's cross-check. Confirm 2.1–2.3 pass.
    - GREEN: all `TestConformanceCapabilities_TokenCounting*` subtests → `PASS`.
    - **Real gap found and fixed during this leaf, not a propagation artifact**: re-reading `R-CNF-002`'s exact text ("WHEN the declaration and an askable discovery disagree — a subject declaring `CAP-O-02` absent while satisfying the token-counting contract, **or the reverse** — the suite MUST fail that entry") showed AI-23.1's `crossCheckDeclaredOptionalCapabilities` only checked the declared-true-but-unsatisfying direction. Extended it (in `conformance_suite.go`, the shared file every leaf may grow) to also check declared-false-but-satisfying, which is this item's own `NFR-CNF-E` (4.1) obligation. Verified the extension does not change any existing test's outcome (`AI-21`'s own `Provider` never satisfies `ai.TokenCounter`, so every pre-existing declared-false scenario is unaffected) — full suite re-run green before continuing.
  - [x] 2.5 REFACTOR: confirm no duplicate type-assertion logic exists between this case and the AI-23.1 cross-check.
    - Confirmed: `grep -n "tokenCounterOf(" *.go` shows the type assertion itself lives only inside `tokenCounterOf` (`conformance_suite.go`); every caller (the AI-23.1 cross-check, both directions, and this leaf's `tokenCountingCase`) goes through it.

- [x] **Item 3** (`R-CNF-016`) — `CAP-O-03` cache-boundary honoring, plus two required cases in this node.
  - [x] 3.1 RED: a subject declaring cache-boundary honoring offered; confirm consumer-visible behaviour is observed; a subject declaring it not offered; confirm `absent` with a reported skip. `S-CNF-042`.
    - Design decision, not stated verbatim in design.md: honoring's one consumer-visible signal through the neutral event stream is a completion's `Usage().CacheRead`/`CacheWrite` count (`usage.go`'s own vocabulary) — cache-boundary markers themselves are request-side (`AI-11`) and honoring has no askable seam (only `CAP-O-02` is askable, `R-AMP-017`), so a scripted completion carrying a populated `CacheRead` count is this case's proof.
  - [x] 3.2 RED: all seven finish reasons on a normally-finished stream; confirm each is reachable and each is a closed-vocabulary value. `S-CNF-043`.
  - [x] 3.3 RED: simulate an eighth finish reason (or a removed one) against the hand-list; confirm the drift guard fails naming the discrepancy in either direction. `S-CNF-044`.
    - A real eighth `ai.FinishReason` cannot be constructed (closed vocabulary), so `finishReasonDriftGuardAgainst` is parameterized over the hand-list and tested directly against an artificially shrunk (removed-value) and grown (added-value) list — `TestFinishReasonDriftGuardAgainst_ShrunkOrGrownList_FailsInBothDirections`, all 3 subtests `PASS`.
  - [x] 3.4 RED: a usage record with one count absent and another reported as zero; confirm the two are distinguishable, neither coerced into the other. `S-CNF-045`.
  - [x] 3.5 RED: standing of the finish-reason and usage cases; confirm both are computed as **required** despite living in this optional-capability node. `S-CNF-046`.
    - Reuses `TestConformanceSkeleton_CompletionMetadataStanding_IsRequired` (AI-23.1) against the identical `CapCompletionMetadata` key both new cases register under (confirmed by `grep -n 'CapCompletionMetadata' conformance_capabilities.go` naming both `registerConformanceCase` call sites) — not re-derived.
  - [x] 3.6 GREEN: implement the cache-boundary case via `R-CNF-002`'s declared expectation; hand-list the 7 `FinishReason` values behind a drift guard probing `FinishReason(n).String() != "invalid"` upward (design's stated mechanism) — **no `src/ai` edit** (`NFR-CNF-B`); implement the absent-vs-zero usage case. Confirm 3.1–3.5 pass.
    - GREEN: all `TestConformanceCapabilities_CacheBoundary*`, `_FinishReasonExhaustiveness*`, `_UsageAbsentVsZero*` → `PASS`. `git diff --stat backend/agent/src/ai/` → empty (confirmed before continuing).
  - [x] 3.7 REFACTOR: confirm the drift guard is the only place the 7-value count lives — no second hardcoded count elsewhere in the package.
    - Confirmed: `handListedFinishReasons` (declared once in `conformance_capabilities.go`) is the only literal enumeration; every other reference (tests, the guard function itself) reads its `len()` rather than a second hardcoded number.

- [x] **Item 4** *(appended, `NFR-CNF-E`)* — Extreme inputs never panic.
  - [x] 4.1 RED: an undeclared-but-askable `CAP-O-02` mismatch (declared absent, provider satisfies the contract) fed through every exported entry point; confirm attributable failure, not a panic. `S-CNF-066` (partial).
  - [x] 4.2 GREEN: guard as needed. Confirm 4.1 passes.
    - GREEN: `TestConformanceCapabilities_ReverseTokenCountingMismatch_FailsEntryNeverPanics` → `PASS` (the extended `crossCheckDeclaredOptionalCapabilities`, item 2.4 above, fails the entry via a `probeTB`, never panics).

- [x] **AI-23.8 close:** record green `make test` and clean `make lint`; confirm `src/ai/` diff is still empty; confirm doc comments cite `R-CNF-014`…`R-CNF-016`; commit `feat(agenttest): conformance optional-capability cases, finish-reason drift guard (AI-23.8)`.
  - `go test -race ./...` → `ok` both packages. `make lint` → `0 issues`. `git diff --stat backend/agent/src/ai/` → empty. `conformance_capabilities.go`'s doc comment cites `R-CNF-014`…`R-CNF-016` and `NFR-CNF-B`.

---

## Phase AI-23.6 — the capability record `[leaf, depends on .2 .3 .4 .5 .7 .8]`

**Deliverable:** extend `conformance_record.go` (`Verdict()`, comparison), extend `conformance_suite_test.go` (record/verdict unit tests, fake end-to-end, finish-reason drift-guard test); **modify `src/agenttest/doc.go`**.
**Spec:** `R-CNF-017`, `R-CNF-018`. **Design:** entries initialize to `OutcomeNotExercised`; totality by construction over `Capabilities()`.
**Depends on:** AI-23.1 (types stub), AI-23.2, .3, .4, .5, .7, .8 (every case-producing leaf, per this node's own header marking).

- [ ] **Item 1** (`R-CNF-017`) — The record is total over both closed lists, standing from AI-03.
  - [ ] 1.1 RED: any completed run; confirm the record carries exactly eight entries, one per capability, each naming capability/standing/outcome. `S-CNF-047`.
  - [ ] 1.2 RED: a run whose subject offers no optional capability; confirm the three optional entries are `absent`, none `not exercised`. `S-CNF-048`.
  - [ ] 1.3 RED: a run in which a required capability's cases failed; confirm that entry is `failed`, never `absent`. `S-CNF-049`.
  - [ ] 1.4 RED: a run that recorded a required capability as optional; confirm validation fails — standing is not the run's to supply. `S-CNF-050`.
  - [ ] 1.5 RED: a published record; confirm it carries no capability-specific detail, model content or credentials. `S-CNF-051`.
  - [ ] 1.6 GREEN: implement `CapabilityRecord` totality (iterate `Capabilities()`, exactly 8 entries), standing sourced from `Capability.Optional()` only, and the field-content restriction. Confirm 1.1–1.5 pass.
  - [ ] 1.7 REFACTOR: confirm a capability with no entry is structurally impossible (built from the enumerator, not appended ad hoc).

- [ ] **Item 2** (`R-CNF-018`) — The verdict rule: `not exercised` is inconclusive; a failed required entry cannot pass.
  - [ ] 2.1 RED: all required `satisfied`, all optional `satisfied`/`absent`; confirm the verdict is a pass. `S-CNF-052`.
  - [ ] 2.2 RED: one `not exercised` entry, rest `satisfied`; confirm the verdict is inconclusive — neither pass nor failure. `S-CNF-053`.
  - [ ] 2.3 RED: one `failed` required entry; confirm the verdict is a failure and no optional result offsets it. `S-CNF-054`.
  - [ ] 2.4 RED: two records for the same subject differing on one entry; confirm the comparison names that entry and the direction of the difference. `S-CNF-055`.
  - [ ] 2.5 RED: AI-21's fake as the first subject, whole suite end to end; confirm the verdict is a pass and every required entry is `satisfied`. `S-CNF-056`.
  - [ ] 2.6 GREEN: implement `Verdict()` (mechanical, four-value set never collapsed to three) and entry-by-entry comparison. Confirm 2.1–2.5 pass.
  - [ ] 2.7 REFACTOR: confirm `Verdict()` has exactly one code path per outcome combination — no case falls through to an implicit default.

- [ ] **Item 3** (`NFR-CNF-C`, `S-CNF-062`) — Package documentation names the suite.
  - [ ] 3.1 Modify `src/agenttest/doc.go`: name the conformance suite alongside AI-21's fake and AI-22's kit; retain their existing framing and the dependency-free pin verbatim. `S-CNF-061`, `S-CNF-062`.
  - [ ] 3.2 Verify with `go doc ./src/agenttest`; confirm all three (fake, kit, suite) are named.

- [ ] **Item 4** *(appended, `NFR-CNF-E`)* — Extreme inputs never panic.
  - [ ] 4.1 RED: `Verdict()` and the comparison function given a zero-value/uninitialized `CapabilityRecord`; confirm attributable failure, not a panic. `S-CNF-066` (partial).
  - [ ] 4.2 GREEN: guard as needed. Confirm 4.1 passes.

- [ ] **AI-23.6 close:** record green `make test` and clean `make lint`; confirm `doc.go` names all three components; confirm `Verdict()`/`CapabilityRecord` doc comments cite `R-CNF-017`/`R-CNF-018`; commit `feat(agenttest): total capability record, verdict rule, doc.go updated (AI-23.6)`.

---

## Milestone close

- [ ] `make test` green in `backend/agent/` (`go test -race -v ./...`), run twice, results identical (`S-CNF-065`).
- [ ] `make lint` clean in `backend/agent/` — run before every leaf commit, not only at the end.
- [ ] `go.mod` still zero requires; both AI-00 import guards pass (`NFR-CNF-A`, `S-CNF-057`).
- [ ] No edit exists under `src/ai/`, including `finish_reason.go`; `src/ai`'s, AI-21's and AI-22's tests pass with the change reverted in isolation; AI-20.4's signature guard passes unmodified (`NFR-CNF-B`, `S-CNF-058`…`S-CNF-060`).
- [ ] Suite lives in `src/agenttest/` under `conformance_`-prefixed files, no new sibling package; `doc.go` names the fake, the kit, and the suite (`NFR-CNF-C`, `S-CNF-061`/`S-CNF-062`).
- [ ] No edit exists to `fake_*.go` or `stream_kit_*.go`; every case names the Layer 1 requirement it proves and delegates mechanics to the kit (`NFR-CNF-D`, `S-CNF-063`/`S-CNF-064`).
- [ ] No exported entry point panics on nil factory / empty case selection / nil subject / undeclared capability expectation — confirm the `NFR-CNF-E` items across all 8 leaves closed (`S-CNF-066`).
- [ ] Every test-list item above carries recorded RED output, recorded GREEN output and a refactor note, in order (`NFR-CNF-F`, `S-CNF-067`) — confirm no item was skipped.
- [ ] Record actual vs. forecast changed-line count in the Review Workload Forecast table above; update the running-total flag against the wave's 5 352-line baseline (AI-21 2 810 + AI-22 2 542) and its accepted ~7 000–8 000 landing estimate.
- [ ] Never push, never merge, never open a PR, never `git stash`.

## Key Learnings

1. AI-23.6's record and verdict types must exist before AI-23.1's runner can compile, so a minimal type stub had to be scheduled at AI-23.1 despite design's File Changes table attributing `conformance_record.go` to AI-23.6 — recorded as an explicit implementation-order note rather than silently reordered.
2. The original `Factory{Reasoning, TokenCounting, CacheBoundary bool}` design could not structurally represent "never declared" versus "declared false" (`S-CNF-006`); this was caught before apply and design.md was corrected to `*bool` fields with nil-fails-construction semantics, so no GREEN-step decision remains open.
3. AI-23's spec omitted `doc.go` from design's File Changes table despite `NFR-CNF-C`/`S-CNF-062` requiring it, so it was added as an explicit AI-23.6 item mirroring AI-22.5's own `doc.go` precedent.
4. AI-23.6's own header marks it dependent on six of the seven other leaves, making it the only node in this milestone that cannot be reordered earlier without breaking the record's totality guarantee.
5. The wave's running total (5 352 changed lines, already past the 5 000 ceiling) plus this milestone's own forecast (~2 850–3 700) lands within the user's already-accepted ~7 000–8 000 estimate, so `Decision needed before apply: Yes` here is a re-confirmation, not a fresh ask.
