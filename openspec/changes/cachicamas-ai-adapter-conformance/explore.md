# Explore — AI-38: Run full deterministic adapter conformance (Wave 6 — Hand off)

> **Change**: `cachicamas-ai-adapter-conformance`
> **Milestone**: AI-38 (doc 0002, lines 2289–2327) · **Wave**: 6 — Hand off
> **Artifact store**: hybrid (file + Engram) · **Engram**: `#2761` (`sdd/cachicamas-ai-adapter-conformance/explore`)
> **Worktree**: `/Users/braejan/workspace/witsaba/repositories/cachicamas-worktrees/ai-38-adapter-conformance`

This file materializes Engram observation `#2761` verbatim in structure and content. The
exploration session had no Write tool and no Bash tool, so this file is a faithful
transcription, not a second exploration pass. Corrections applied by the proposal phase after
re-reading the code are recorded in the closing "Corrections after re-verification" section —
the original body is left as written so the two can be compared.

No `.codegraph/` index exists in the worktree; investigation was done with Read/Grep/Glob only.
`go build` / `go test` could NOT be run to confirm compile or pass status; this MUST be verified
in `sdd-apply` / `sdd-verify`.

---

## Central finding — a substantial pre-existing scaffold already sits in this worktree

The worktree already contains a large "PR #2" implementation attempt, self-labeled `AI-38` in
comments, that pre-dates this SDD run. No `openspec/changes/cachicamas-ai-adapter-conformance/`
directory existed at exploration time, so no SDD artifact trail explained the code.

This existing code **does not meet the milestone's own acceptance bar**:

- `openspec/specs/ai-openrouter-first-provider/spec.md` R-OR-06 (already-promoted spec) requires
  `agenttest.RunConformance` end-to-end with **all five required capabilities passing**:
  `CapStreamingText`, `CapToolCalls`, `CapCompletionMetadata`, `CapCancellation`, `CapTerminal`.
- The existing scaffold
  (`backend/agent/src/ai/openaicompat/openrouter/conformance/run_for_test.go`) only runs
  `CapStreamingText` and `CapToolCalls` via the **scoped** `agenttest.RunConformanceFor` (never
  the full `RunConformance`). `RunConformanceFor`'s own doc says explicitly: "a scoped run's
  verdict ... is never presentable as evidence of full conformance (S-CLA-032)."
- `CapCompletionMetadata`, `CapCancellation`, and Terminal cases are `t.Skip`'d with comments
  that admit the case "would FAIL by construction" against the real adapter, citing reopening
  `AI-32` (in-band error chunks) or `AI-20.3` (cancellation-as-`ErrorEvent`) as out of scope for
  "PR #2".
- This contradicts the milestone charter (doc 0002): "Any suite case the adapter cannot pass is
  a defect in one of the two, resolved before this node closes. There are no waivers."
- AI-38.1 test 3 requires transcripts be **regenerable** via "a recording helper captures a real
  stream into the exact fixture format". Every fixture in the scaffold
  (`openrouter/conformance/fixtures/*.go`) is hand-authored Go byte-literal SSE text. No
  recording/capture helper exists anywhere in the repo (`Record|Recorder|CaptureTranscript` —
  zero hits outside unrelated trace/redaction code). This is the exact "hand-typed fixtures
  drift from real wire behavior" failure mode AI-38.1 exists to close.

## 1. How the AI-23 suite plugs in a subject

- Entry point: `agenttest.RunConformance(t, agenttest.Factory)` — `agenttest/conformance_suite.go:252`.
  Runs the whole `conformanceRegistry` case table; returns a total `CapabilityRecord` (9 entries).
- `Factory{New func(tb, ...Script) ai.ModelProvider; Reasoning, TokenCounting, CacheBoundary, Retry *bool; Sentinel string}`
  (`conformance_suite.go:165`) — every optional capability must be explicitly declared true/false
  or construction fails (S-CNF-006, no undeclared capabilities).
- A **scoped** door also exists: `agenttest.RunConformanceFor(t, f, capability)`
  (`conformance_scoped.go:33`) — documented as never usable as full-conformance evidence.
- AI-21's fake (`agenttest.FakeFactory()`) is the only subject the **unscoped** `RunConformance`
  is actually exercised against today (`conformance_suite_test.go:124,1372`) — no real adapter
  has ever been run through the unscoped entry point.
- AI-35's `CapRetry` case body lives in package `openaicompat` (`conformance_retry_test.go`) and
  self-registers via the exported twin `agenttest.RegisterConformanceCase` from its own `init()`,
  because it needs raw HTTP-level scripting the `ai.ModelProvider` factory interface cannot
  express. The case body bypasses `f.New` and builds its own `httptest.Server` + real
  `*openaicompat.Client`. This is the precedent AI-38 should follow for HTTP-only assertions.
- AI-23.1 exports test-only twins (`FactoryDefectForTest`, `NewCapabilityRecordForTest`,
  `SetOutcomeForTest`) "reserved for adapter-specific conformance cases whose test code lives in
  another package" — built for AI-38 by AI-23.1's own design.

## 2. Replay/transcript infrastructure that exists — and the regeneration gap

- `backend/agent/src/ai/openaicompat/bridge_test.go` (459 lines): `conformanceBridgeFactory()`
  renders an `agenttest.Script` to raw SSE bytes on the fly (`renderScript`), serves it from a
  real `httptest.Server`, and drives it through a real `*openaicompat.Client`. All four optional
  capabilities declared `false`. Only `TestConformanceBridge_StreamingText` and
  `TestConformanceBridge_ToolCalls` run (both scoped). `CapCompletionMetadata` is explicitly NOT
  extended here, and the gap is routed "to AI-38.2" in a comment — the shipped adapter code
  itself already expects AI-38 to resolve this.
- `backend/agent/src/ai/openaicompat/openrouter/conformance/` (package `openrouter_conformance`,
  non-idiomatic underscore name per an unlocated "task plan § PR #2 2.1" — that task-plan
  document is NOT present anywhere in the repo, only referenced from code comments):
  - `run_for_test.go` — 5 per-capability drivers; 2 active (Streaming/Tools), 3 `t.Skip`'d
    (Metadata/Cancellation/Terminal) with the out-of-scope admissions above.
  - `capability_record_test.go` — asserts the factory's optional-capability bool pointers are
    `&false` (AI-24 §8's `absent × 3`), plus one `RunConformanceFor(CapStreamingText)` sanity
    call. NOT a run of the full generated `CapabilityRecord`.
  - `fixtures_test.go` / `fixtures/*.go` — hand-authored SSE byte blobs (structural shape tests
    only, "not golden-tested bit for bit").
  - `reasoning_extension_test.go` — direct single-fixture replay proving OpenRouter's renamed
    `reasoning_details` / `reasoning` fields are dropped, not leaked (R-ARS-015 / R-OR-06 sub-scenario 2).
  - `bridge_test.go` — same package, the OpenRouter wrapper's `conformanceBridgeFactory()`.
- **Missing entirely**: any recording/capture helper that hits a real endpoint and serializes the
  response into this fixture format. AI-38.1 test 3 is unimplemented.

## 3. Capability report (AI-23.6) and AI-24.1's expected record

- `CapabilityRecord` (`conformance_record.go`) is a fixed 9-entry array, each entry
  `{Capability, Standing, Outcome}`. `Outcome ∈ {Satisfied, Absent, Failed, NotExercised}`
  (`NotExercised` is a real 4th member, never the zero value). `Verdict()` computes pass/fail/
  inconclusive mechanically. `CompareCapabilityRecords(got, want)` diffs entry-by-entry — built
  exactly for AI-38.2's "assert against AI-24.1's recorded expectation" need.
- AI-24.1's expected record is NOT a data file — it is spec prose:
  `openspec/specs/ai-openrouter-first-provider/spec.md` R-OR-05 ("CAP-O-01 = absent under default
  model `openai/gpt-4o`") and R-OR-06 ("the capability record equals AI-24 §8's expected
  `absent × 3`"). AI-38.2 must emit a **generated** report and assert it with
  `CompareCapabilityRecords`, not hand-check declared bools — R-CNF-004's absence must come from
  an actual suite run reaching `applyDeclaredAbsences`.
- AI-29 already struck reasoning for this vendor/model: a generated `CAP-O-01 = satisfied` here
  would be FINDING #1 (reopen trigger).

## 4. AI-27.3's boundary-split mechanism and integration-level reuse

- `backend/agent/src/ai/openaicompat/offset_sweep_test.go` (decoder level, package
  `openaicompat_test`). Mechanism: registry `sweepTranscripts []sweepTranscript{name, transcript, want}`;
  `checkSweepFixtureBound` enforces every fixture ≤ `maxSweepFixtureBytes` (S-ASD-049, inclusive
  bound); the sweep decodes the SAME transcript split at every byte offset and asserts identical
  decoded frames vs. the canonical unsplit decode.
  `TestOffsetSweep_CanonicalDecodeMatchesHardCodedFrames` anchors the canonical decode against a
  hard-coded frame list so the sweep cannot vacuously pass against a uniformly-broken decoder.
- Reuse "at the integration level" (AI-38.3) means: take each conformance transcript, split it at
  every adversarial byte offset, feed each split version through the same `httptest.Server`
  bridge + real `*openaicompat.Client` + the suite, and assert the SAME conformance outcome (not
  just "same decoded frames" — now "same suite verdict") at every split point. This is new code;
  no existing test drives a conformance **case** across a boundary sweep. Needs its own
  fixture-size guard (`checkSweepFixtureBound` is an unexported `_test.go`-local helper today).

## 5. Capability gap matrix (required vs. optional) for openaicompat/openrouter

| Capability | Standing | Current real-adapter status |
|---|---|---|
| `CapStreamingText` (CAP-R-01) | Required | PASS via scoped `RunConformanceFor`, both bridges |
| `CapToolCalls` (CAP-R-02) | Required | PASS via scoped `RunConformanceFor`, both bridges |
| `CapCompletionMetadata` (CAP-R-03) | Required | **NOT RUN** — `t.Skip`'d; finish-reason exhaustiveness is inherently unreachable for 3/7 values on this dialect (strict gate correctly rejects Refusal/PauseTurn/Unknown as malformed) |
| `CapCancellation` (CAP-R-04) | Required | **NOT RUN** — `t.Skip`'d; openaicompat's shipped contract turns ctx-cancel into an `ai.ErrorEvent`, the suite's cancellation cases assert a bare close with no invented terminal — direct semantic conflict between two shipped, spec-approved behaviors |
| `CapTypedFailures` (CAP-R-05) | Required | Partially covered — `CapRetry`'s case body proves typed-failure categories at HTTP-transport level; `conformance_terminal.go`'s own cases are **NOT RUN**, needing in-band `ErrorEvent`-as-wire-chunk rendering openaicompat's declared contract does not have |
| `CapReasoningContent` (CAP-O-01) | Optional | Declared absent (matches AI-29's struck verdict for `openai/gpt-4o`); drop behavior separately proven |
| `CapTokenCounting` (CAP-O-02) | Optional | Declared absent; not cross-checked via a real `RunConformance` run |
| `CapCacheBoundary` (CAP-O-03) | Optional | Declared absent |
| `CapRetry` (CAP-O-04) | Optional | Declared `true` in `openaicompat`'s own retry-case factory; declared `false` in both bridge factories — inconsistent between the two factories that supposedly "inherit suite cases without rewriting assertions" |

Net: 2 of 5 required capabilities pass against the real adapter today; 3 required capabilities and
the `CapRetry` wiring remain open. This is the actual scope of AI-38.1, not a near-complete task.

## 6. Shape-of-change estimate

Already present (not new work, but correctness/compliance unverified — no test run possible this
session): ~1,600+ lines across `openrouter/conformance/*.go`, `openaicompat/bridge_test.go`,
`openaicompat/conformance_retry_test.go`.

New work implied by a genuinely-compliant AI-38:

- Recording/capture helper, gated like AI-39 behind `OPENROUTER_API_KEY`, serializing a real SSE
  stream into the fixture format the bridges already consume — 150–300 lines + regenerated
  fixtures (generated goldens, excluded from authored-risk count per `sdd-phase-common.md` §E).
- Resolution of the CapCompletionMetadata / CapCancellation / CapTerminal gap — genuinely open
  design question; ~100 lines (bridge-side accommodation) to several hundred plus spec amendments
  (reopening AI-20.3 / AI-32 / ACP-002 surfaces).
- AI-38.2 capability-report generation + `CompareCapabilityRecords` assertion — 80–150 lines.
- AI-38.3 boundary-replay matrix — 150–300 lines, reusing `checkSweepFixtureBound`'s shape.
- `CapRetry` wiring parity between the two factories — 20–40 lines.

Total realistic new-authored estimate: roughly 500–1,000+ lines, on top of ~1,600 already-present
lines of uncertain quality.

## Approaches

1. **Widen-and-reconcile (spec-literal)** — build the recording helper first (regenerate today's
   hand-typed fixtures from real captures), then treat every suite-case-vs-adapter mismatch as an
   AI-38.1-test-2 "defect in one of the two", resolved via adapter widening or a suite-side
   accommodation. Finish with AI-38.2's generated capability report and AI-38.3's
   integration-level boundary sweep.
   - **Pros**: the only approach that satisfies the already-promoted R-OR-06 and the milestone's
     explicit "no waivers" charter; recorded transcripts deliver AI-38.1's stated purpose instead
     of reproducing the failure mode.
   - **Cons**: high effort; touches 2–3 already-archived spec surfaces (AI-20.3, AI-32,
     openaicompat's strict finish-reason gate) — real architecture risk and review cost; the
     cancellation/finish-reason/terminal resolution is a genuine open design question.
   - **Effort**: High.
2. **Scope-reduce via ADR (formalize the existing skips)** — accept 2-of-5 required-capability
   coverage as final, draft an ADR + spec amendment narrowing R-OR-06, route the rest to a future
   milestone.
   - **Pros**: much lower effort; fits comfortably inside review-budget guards; reopens nothing.
   - **Cons**: directly contradicts the "no waivers" charter and the promoted R-OR-06 (5 named
     required capabilities) — exactly the silent-scope-narrowing the milestone flags as a reopen
     trigger; requires human sign-off on weakening an accepted spec, a governance decision outside
     a normal explore→propose flow.
   - **Effort**: Medium (the ADR/amendment process, not the code).

## Recommendation

Approach 1, but `sdd-propose` / `sdd-design` MUST NOT assume the CapCompletionMetadata /
CapCancellation / CapTerminal resolution direction — that is a genuine open technical question
(bridge accommodation vs. suite-case amendment vs. adapter widening) to be decided explicitly in
`sdd-design`, likely after a recorded real transcript reveals what the wire actually does on
cancellation and error paths.

## Risks

- Pre-existing scaffold's compile/pass status is UNVERIFIED — `sdd-apply` must run `make test`
  early and treat any failure as a defect in the found code, not assume it is green.
- The CapCancellation and CapCompletionMetadata gaps may force reopening already-archived spec
  surfaces (`ai-stream-lifecycle`, `ai-provider-error-mapping`, `ai-provider-completion`).
- No task-plan document referenced in code comments ("task plan § PR #2") exists anywhere in the
  repo — the prior scaffold's own planning artifact is missing; do not assume it is recoverable.
- The 400-line reviewer budget (`sdd-phase-common.md` §E) and the 1000-line PR budget are both at
  real risk of being exceeded.
- `CapRetry` is declared inconsistently between factories — a latent defect independent of AI-38
  that AI-38.2's capability-report work will surface.

## Ready for proposal

Yes, with the caveat above: scope the proposal around Approach 1 but flag the CAP-R-03/04/05
resolution direction as a design-phase decision, and recommend chained PR slices from the outset.

---

## Corrections after re-verification (proposal phase, 2026-08-09)

Re-reading the code and the OpenSpec tree corrected four points in the record above. The original
text is kept unedited; these supersede it where they conflict.

| # | Original claim | Corrected fact |
|---|---|---|
| 1 | The scaffold is "leftover/abandoned work-in-progress" with no artifact trail | It is **committed, archived work**: `openspec/changes/archive/2026-08-06-add-openrouter-first-provider/` holds the full proposal/design/tasks/verify/archive set. PR #2 of that chain deliberately deferred full conformance to AI-38. It is expected state, not stray code. |
| 2 | The suite-vs-adapter cancellation conflict traces to AI-20.3 | The current adapter behaviour is governed by **AI-32.3 / `ai-provider-error-mapping` R-AEM-014, S-AEM-051…055** ("MUST still surface a typed terminal failure on cancel/deadline"), which itself amended AI-20.3's silent-loss path. The conflict is therefore suite (`R-CNF-011` / `R-CNF-012`) vs. error-mapping spec, both currently promoted. |
| 3 | Three capability families fail an unscoped run | `capability_record_test.go` records its own RED evidence: an earlier draft calling `agenttest.RunConformance` failed on **five** case families — cancellation, terminal, `finish_reason/refusal`, `usage/absent_vs_zero`, and redaction. The gap is wider than the three skipped drivers. |
| 4 | The `add-openrouter-first-provider` change directory is unarchived leftover | It was archived on 2026-08-06. A **later, orphan revision** re-created the directory carrying only a revised `R-OR-07` (env-var gate, no CI workflow, citing ADR 0005's no-`.github/` posture). The promoted `openspec/specs/` copy still carries the older `workflow_dispatch` text, and the repo has no `.github/` directory at all. That divergence is R-OR-07/R-OR-08 territory — AI-39's, not AI-38's. |
