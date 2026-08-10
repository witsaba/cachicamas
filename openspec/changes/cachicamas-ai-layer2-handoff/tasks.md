# Tasks: AI-40 — Publish the Layer 2 readiness contract

> Discharges spec `ai-layer2-handoff` (R-L2H-001..009, NFR-L2H-A/B, S-L2H-001..042) and the
> `ai-provider-conformance-suite` delta (D6, S-CNF-088..090). Design AD-1..AD-7 binding.
> Strict TDD. Test runner `cd backend/agent && make test` (`-race`); lint `make lint`;
> build `make build`. `go.mod`/`go.sum` byte-identical to base (NFR-L2H-A, S-L2H-039).

## Binding corrections applied in this task list

1. Text-delta events use `ai.NewTextDelta(block, delta) (Event, error)` — NOT `NewTextBlockDelta`
   (verified `text_events.go:167`). `NewTextBlockStart`/`NewTextBlockEnd` are correct as designed.
2. All cited constructors (`NewText`, `NewMessage`, `NewRequest`, `NewCompletion`, `ErrorEvent`,
   `MidStreamFailure`, `NewToolCallStart/Delta/End`) return `(T, error)` — every call site MUST
   check the error; design pseudocode is schematic, not literal code.
3. WU-B's genuine RED is NOT the compile/`go vet` gate (that passes by design). Its RED is
   S-L2H-009's shape: deliberately break one `// Output:` line, run `make test`, record the
   failing run, then restore it.
4. Item 6's distinction clause differs between doc 0002 line 2402 ("…not reopened by this
   amendment") and line 2446 ("…not struck by this amendment") — WU-C/D adapts the trailing
   clause to context without weakening the AI-17-unaffected distinction.

## Review Workload Forecast

| Field | Value |
|-------|-------|
| Estimated changed lines | ~1,900–2,300 (code ~350–450; new tests ~450–550; docs/openspec ~1,100–1,300) |
| 400-line budget risk | High |
| Chained PRs recommended | No (locked: `single-pr`) |
| Suggested split | Single PR, 6 internal commits (WU-A..WU-F) |
| Delivery strategy | single-pr |
| Chain strategy | size-exception |

Decision needed before apply: No
Chained PRs recommended: No
Chain strategy: size-exception
400-line budget risk: High

Rationale: `size:exception` with automatic raising is pre-accepted for AI-40 only
(`state.yaml: delivery.size_exception`), so no further chain-strategy decision blocks apply.
Estimate includes openspec artifacts already in-branch (proposal.md ~227 ln, spec.md ~320 ln,
delta spec.md ~69 ln, design.md ~115 ln) plus this tasks.md and the new `decision.md`
(~250–350 ln) and doc 0002 edits (~30 ln) — all counted per guard rule since they ship in this PR.

### Suggested Work Units

| Unit | Goal | Focused test command | Runtime harness | Rollback boundary |
|------|------|-----------------------|------------------|--------------------|
| WU-A | `src/handoff/` consumer proof (AI-40.1) | `go test ./src/handoff/... -race -v` | `TestHandoff_ConsumerProof` (3 subtests) is the real scenario | Delete `src/handoff/` — nothing else references it |
| WU-B | 4 runnable examples (AI-40.2 examples) | `go test ./src/ai/... -race -run Example -v` | `make test` runs examples as real output-verified scenarios | Delete `example_test.go` — no production code touched |
| WU-C/D | Matrix + drift guard + 2 duties (AI-40.2 matrix, R-L2H-003..006) | `go test ./src/ai/openaicompat/openrouter/conformance/... -race -run DocMatrix -v` | Guard runs against real `src/ai/doc.go` bytes + `expectedOpenRouterRecord()` | Revert `doc_matrix_guard_test.go` + the two `doc.go` sections together (guard cites the matrix by name) |
| WU-E | `decision.md` (AI-40.3) | N/A — passive doc, structural readback | N/A — reason: no executable surface, cited artifact only | Delete `decision.md` — nothing cites it before archive |
| WU-F | Doc 0002 reconciliation + D6 confirm + bookkeeping | N/A — passive doc, structural readback | N/A — reason: markdown checklist edit only | Revert the amendment blockquote + checkbox flips |
| Gates | Final full-suite verification | `cd backend/agent && make test && make lint && make build` | Full `-race` suite is the real harness | N/A — verification only, no revert |

## Phase 0: Pre-flight

- [x] 0.1 Re-read `docs/architecture/milestones/0002-...md` lines 2414–2436 fresh; record the
      on-disk checkbox state actually seen (risk: checkbox staleness) before writing WU-E/WU-F.

## Phase 1 — WU-A: Consumer proof (`src/handoff/`, AI-40.1, R-L2H-001)

- [x] 1.1 RED: create `backend/agent/src/handoff/handoff_test.go` (`package handoff_test`,
      imports `context`, `errors`, `testing`, `.../src/ai`, `.../src/agenttest`) with
      `TestHandoff_ConsumerProof` and its three `t.Run` subtests (drain / scripted error /
      cancellation) per design AD-1. Run `go test ./src/handoff/...` — record the build failure
      (`package handoff` does not exist).
- [x] 1.2 GREEN: add `backend/agent/src/handoff/doc.go` (`package handoff`, ~5-line comment: the
      package is intentionally empty, the test is the deliverable). Run `go test ./src/handoff/...
      -race -v` — record all three subtests passing.
- [x] 1.3 Subtest "drains a scripted stream": `ai.NewText`→`ai.NewMessage`→`ai.NewRequest`;
      `agenttest.NewProvider(agenttest.Script{...})` emits `ai.NewTextBlockStart`,
      `ai.NewTextDelta` (not `NewTextBlockDelta`), `ai.NewTextBlockEnd`, then
      `ai.NewCompletion(ai.FinishReasonStop, usage)`; assert full drain, ordered event kinds via
      `Event.Kind()`, and `Requests()[0]` echoes the sent request. (S-L2H-002)
- [x] 1.4 Subtest "surfaces a scripted terminal error": last script step emits
      `ai.ErrorEvent(ai.MidStreamFailure(report, true))`; assert the error event is last and
      inspect only through `Event.ErrorPayload()` → `Failure.Category()` / `.Retryable()` — no
      vendor type read. (S-L2H-003)
- [x] 1.5 Subtest "exits boundedly on cancellation": `Hold(gate)` mid-script; cancel the request
      context after the gate is reached; assert the channel closes within a bounded
      `select`-with-timeout and no terminal event is forced (sanctioned loss path). (S-L2H-004)
- [x] 1.6 Confirm package clause of `handoff_test.go` is external (`handoff_test`, not `handoff`
      or `agenttest`/`agenttest_test`) — satisfies S-L2H-001.
- [x] 1.7 Run `go list -deps -test ./...` from `backend/agent` (or `make test`, which sweeps
      `import_boundary_test.go`) and confirm `src/handoff` reports zero vendor imports with NO
      edit to `import_boundary_test.go` or its allowlist. (S-L2H-005, S-L2H-006)

**Commit boundary**: one commit — `handoff_test.go` + `doc.go` travel together (RED recorded in
commit message per work-unit-commits convention, never left broken in history).

## Phase 2 — WU-B: Runnable examples (`src/ai/example_test.go`, AI-40.2, R-L2H-002)

- [x] 2.1 Compile-only gate (NOT the TDD RED — design AD-2): create `example_test.go` skeleton
      (`package ai_test`, imports incl. `.../src/agenttest`, one trivial `Example*`). Run
      `go vet ./src/ai/...` — record it proves `ai_test`→`agenttest` is acyclic. If it fails,
      invoke the D3 fallback: split streaming + tool-call examples into
      `src/agenttest/example_test.go` (`package agenttest_test`) and cross-link both godoc
      surfaces from `doc.go`.
- [x] 2.2 Write `ExampleNewRequest` (`ai.NewText`→`ai.NewMessage`→`ai.NewRequest`; prints model +
      message count) with a deterministic `// Output:` block. (S-L2H-008)
- [x] 2.3 Write `ExampleModelProvider_streaming` (`agenttest.NewProvider`, drain, `switch
      event.Kind()`; prints ordered kind names) with `// Output:` block. (S-L2H-008)
- [x] 2.4 Write `ExampleModelProvider_toolCallReconstruction` (scripted `ai.NewToolCallStart`,
      `ai.NewToolCallDelta`, `ai.NewToolCallEnd`; accumulate deltas, End's arguments authoritative
      per `ToolCallEnd`; prints name + reconstructed args) with `// Output:` block. (S-L2H-008)
- [x] 2.5 Write `ExampleFailure_inspection` (scripted terminal `ai.ErrorEvent`; prints
      `Category()` name + `Retryable()` bit via `Event.ErrorPayload()`) with `// Output:` block.
      (S-L2H-008)
- [x] 2.6 Confirm no credential, no network call, no `time.Sleep` anywhere in the file.
      (S-L2H-010)
- [x] 2.7 RED (genuine, per binding correction #3 / S-L2H-009): deliberately alter one character
      in `ExampleNewRequest`'s `// Output:` block. Run `cd backend/agent && make test` — record
      the failing example output (RED).
- [x] 2.8 GREEN: restore the correct `// Output:` line. Run `make test` — record all four
      examples passing under `-race`. (S-L2H-007)
- [x] 2.9 Confirm `go test ./src/ai/... -race -run Example` runs and verifies all four —
      `make test`'s normal `./...` sweep already includes them. (S-L2H-011)

**Commit boundary**: one commit — skeleton, four examples, and the RED/GREEN evidence travel
together; only the corrected file lands in the tree.

## Phase 3 — WU-C/D: Drift guard, capability matrix, inherited duties (AI-40.2, R-L2H-003..006)

- [x] 3.1 RED: create `backend/agent/src/ai/openaicompat/openrouter/conformance/doc_matrix_guard_test.go`
      (`package openrouter_conformance` — same package as `capability_record_test.go`, so
      `expectedOpenRouterRecord()` is a direct call). Resolve `src/ai/doc.go` via
      `runtime.Caller(0)` + `filepath.Join(dir, "..", "..", "..", "doc.go")`, mirroring
      `provider_signature_guard_test.go`. Parse rows with regex `^//\tCAP-[RO]-\d\d\b`; fail
      loudly (naming the path) on an unresolvable/unparseable file. Run the test — record RED:
      "found 0 of 9 rows" (matrix does not exist yet). (S-L2H-017 precondition)
- [x] 3.2 GREEN: append `# The first adapter's capability record` to `ai/doc.go` — nine rows in
      grammar `//\tCAP-R-01 streaming text  required  satisfied — <reason>` (and `CAP-O-0N` for
      optionals), entry-for-entry and in-order from `expectedOpenRouterRecord().Entries()`
      (`Capabilities()` order). Header names the generator
      (`TestOpenRouterAdapter_FullConformance`, `run_for_test.go`) and the committed expectation
      (`capability_record_test.go`). `CAP-O-01` row states the struck verdict (AI-29) and the
      reopen trigger inline (`R-OR-05`, `R-ACR-004`, ADR required) — never phrased as permanent.
      Run the guard — record GREEN. (R-L2H-003, S-L2H-012..015)
- [x] 3.3 Confirm the guard requires exactly nine rows (no vacuous pass on fewer) and compares
      capability ID, standing (`required`/`optional`), and outcome per row. (S-L2H-018,
      S-L2H-019)
- [x] 3.4 Append the two inherited-duty paragraphs beside the matrix section: (a) item-6 wire
      clause — not exercisable in v1, naming `AI-26.6`'s refusal and `AI-29.2`'s striking, with
      the AI-17/`R-ARE-009`/`R-ARE-010`/G12(b) stream-half-unaffected sentence lifted from doc
      0002 line 2446 (adapted trailing clause per binding correction #4 — do not weaken the
      distinction); (b) Layer-2-strips-`reasoning_details` duty, attributed to AI-29's recorded
      absence, citing AI-29's `decision.md` by identifier. Neither replaces the other.
      (R-L2H-005, R-L2H-006, S-L2H-020..025)
- [x] 3.5 Append `# The v1 surface freeze` pointer paragraph to `ai/doc.go`, citing the decision
      artifact **by change name** (`cachicamas-ai-layer2-handoff`) and doc 0002 § AI-40 — never
      a pre-archive path. Add one pointer sentence to `agenttest/doc.go`. (R-L2H-007, S-L2H-028,
      S-L2H-029)
- [x] 3.6 Bite/defeat-test (recorded, not committed): flip one published row's outcome text,
      rerun the guard, confirm it fails and names the differing row; revert. (S-L2H-018)
- [x] 3.7 Confirm the diff touches no conformance-suite behavior, no adapter file, and no
      exported symbol in `capability_record_test.go`/`run_for_test.go`. (S-L2H-016, NFR-L2H-A)
- [x] 3.8 Confirm item 6's checkbox is untouched by this WU (doc 0002 edit is WU-F's, not this
      one) and no artifact here claims item 6 closed. (S-L2H-023)

**Commit boundary**: one commit — guard test and the `doc.go` matrix/duties it protects travel
together (RED "found 0 of 9 rows" recorded in the commit message, never left broken in history).

## Phase 4 — WU-E: Compatibility statement (`decision.md`, AI-40.3, R-L2H-007..009)

- [x] 4.1 Create `openspec/changes/cachicamas-ai-layer2-handoff/decision.md` following the
      AI-29 `decision.md` shape (`src/ai/openaicompat/decision.md`), including the
      `[!IMPORTANT]` no-Go-identifier box (only landed test names + vendor wire fields
      permitted).
- [x] 4.2 §1 How to use (AG-03 author / doc-0003 reader / reviewer).
- [x] 4.3 §2 Frozen v1 surface, by capability (per Q3): request construction; content + tool
      vocabulary; breakpoints/options/rebuild; event contract + per-stream sequencing; provider
      interface + pre-stream order; typed taxonomy + partial-output discriminator; cancellation +
      bounded backpressure (N=0); retry policy; testing surface; observability boundary. Mark
      experimental/not-frozen explicitly (live-smoke, token counting, R-CNF-020..026 gap) — state
      "none" if any category is empty. (R-L2H-007, S-L2H-026, S-L2H-027)
- [x] 4.4 §3 Eighteen-item walk table (item → observed on-disk state from task 0.1 → closing
      node(s) per doc 0002 line 2514 spine → one-line evidence citation). Row 6 cites struck
      `AI-29.2` + wire-half-published-via-AI-40.2 + AI-17-stream-half-unaffected; row 18 cites
      `AI-40.1`/`AI-40.3`. Report status only — no re-verification of already-closed items'
      evidence. (R-L2H-008, S-L2H-030, S-L2H-031, S-L2H-034)
- [x] 4.5 §4 Never-cancelled posture: caller owns the context; producer blocks until it ends;
      abandoning without cancelling is a consumer contract violation; name `AI-23.5`/`AI-33.3` as
      the tested abandoned-then-cancelled neighbour; mark never-cancelled explicitly
      **untestable to termination** with a stated reason — no test-coverage claim. (R-L2H-009,
      S-L2H-035..037)
- [x] 4.6 §5 What Layer 2 inherits (both duties + doc 0003 `AG-03` gate). §6 Closing-checklist
      verification of AI-40.1/40.2/40.3's own three criteria, AI-29 §12 style.
- [x] 4.7 Structural readback: confirm the `doc.go` pointer (task 3.5) names this file by change
      name and carries a summary, not a duplicate enumeration. (S-L2H-029)

**Commit boundary**: one commit — passive doc, no code. Independently revertible except that
tasks 1–3's artifacts are cited by name inside it (cite-only, not modified by reverting this).

## Phase 5 — WU-F: Doc 0002 reconciliation + D6 confirm + bookkeeping (R-L2H-008, delta)

- [x] 5.1 Using task 0.1's fresh read, tick line 2428 (item 11) `[x]`, citing AI-33's close
      amendment (line 16).
- [x] 5.2 Tick line 2429 (item 12) `[x]`, citing AI-34's close amendment (line 18).
- [x] 5.3 Tick line 2431 (item 14) `[x]`, citing AI-23 (Wave 3) + AI-38.1 (line 48, PR #140,
      `5bc2da4e`).
- [x] 5.4 Tick line 2432 (item 15) `[x]`, citing AI-38's close amendment (line 48).
- [x] 5.5 Tick line 2433 (item 16) `[x]`, citing AI-36's close (line 24) + suite case AI-23.7.
- [x] 5.6 Tick line 2434 (item 17) `[x]`, citing AI-39.1 (PR #142, `b062be74`).
- [x] 5.7 Tick line 2435 (item 18) `[x]`, citing `AI-40.1`/`AI-40.3` (this change).
- [x] 5.8 Leave line 2423 (item 6) `[ ]` unchanged — by design (line 2446); confirm no artifact
      in this change ticks it. (S-L2H-023)
- [x] 5.9 Update line 3 Status: 41 → **42 of 42**; Remaining → none.
- [x] 5.10 Add `> Amended 2026-08-10 — AI-40 close` blockquote (AI-38/AI-39 pattern: counter,
      fresh `find` file counts measured at close, what-delivered, verify-gate evidence, archive
      link, Engram topic keys). Checkbox lines stay bare (Wave-2-close precedent). (S-L2H-030..033)
- [x] 5.11 Confirm D6 delta spec is already staged at
      `openspec/changes/cachicamas-ai-layer2-handoff/specs/ai-provider-conformance-suite/spec.md`
      (verified present, MODIFIED item 10 eight→nine, S-CNF-088..090). No further edit needed
      pre-archive — promotion into `openspec/specs/ai-provider-conformance-suite/spec.md`
      happens at `sdd-archive`, not here. Confirm `openspec/specs/ai-first-provider-decision/spec.md`
      stays untouched (S-CNF-090).
- [x] 5.12 State bookkeeping: `sdd-apply` marks each completed checkbox above `[x]` in this
      `tasks.md` and updates `openspec/changes/cachicamas-ai-layer2-handoff/state.yaml`
      (`phases.apply.status`, `status:` field) on completion — per the openspec convention, not
      as a separate code change.

**Commit boundary**: one commit — doc-0002 edits are independently revertible from Phases 1–4
(no code cites doc 0002's checkbox text).

## Phase 6: Final gates (NFR-L2H-A, NFR-L2H-B, S-L2H-038..042)

- [x] 6.1 `cd backend/agent && make test` — record green, `-race`, full `./...` sweep (includes
      WU-A/B/C/D's new tests, the boundary guard, the signature guard, and all pre-existing
      suites unmodified).
- [x] 6.2 `cd backend/agent && make lint` — record 0 issues.
- [x] 6.3 `cd backend/agent && make build` — record exit 0.
- [x] 6.4 `git diff` on `backend/agent/go.mod` and `go.sum` against base `b062be74` — confirm
      byte-identical. (S-L2H-039)
- [x] 6.5 `git diff --stat` — confirm no rename/move of `src/agenttest` or `src/ai`; signature
      guard (`provider_signature_guard_test.go`) still resolves and passes. (S-L2H-038)
- [x] 6.6 Guard sweep: re-confirm `import_boundary_test.go` reports zero vendor imports for
      `src/handoff` (task 1.7) and the doc-matrix guard is green (task 3.2) as part of the same
      `make test` run — no standalone re-run needed once 6.1 is green.
- [x] 6.7 Confirm every WU above recorded a genuine RED before its GREEN in this file's commit
      history (S-L2H-041): WU-A (compile fail), WU-B (broken Output line), WU-C/D (0-of-9-rows).

## Skipped rows

None — the design's threat matrix is N/A (no routing/shell/subprocess/VCS-automation boundary
added; `go list` is pre-existing AI-00.3 machinery, unmodified).
