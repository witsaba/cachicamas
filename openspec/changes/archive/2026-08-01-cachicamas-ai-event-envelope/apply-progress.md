# Apply progress — event envelope with per-stream sequencing (AI-14)

> **Change**: `cachicamas-ai-event-envelope` · **Milestone**: AI-14 (Wave 2 "Stream")
> **Mode**: Strict TDD
> **Status**: complete — all 6 phases / 55 tasks done. `make test` green under `-race`. `make lint` `0 issues`.

> ## Provenance — read this first
>
> **This file was synthesized at archive time (2026-08-01) from [`tasks.md`](./tasks.md)'s Evidence Log**, not written by the original `sdd-apply` session. AI-14 is the only one of Wave 2's seven milestones whose apply session never wrote a separate `apply-progress.md`; it recorded the same substance — phase-by-phase RED/GREEN evidence, verbatim failure transcripts, a lint-gap discovery with its fix, and two honest deviation disclosures — inline in `tasks.md` instead.
>
> The gap was raised as **W5** in [`WAVE-2-VERIFY.md`](../WAVE-2-VERIFY.md) § 6, owned by `sdd-archive`, with the remedy stated as "either synthesize the artifact from `tasks.md` or record the substitution". This file is that synthesis. **Every claim below is traceable to `tasks.md`; nothing was inferred, reconstructed, or added.** Where `tasks.md` records an uncertainty or a gap, this file preserves it rather than smoothing it over. Where a fact the other six milestones' `apply-progress.md` files carry is simply not in `tasks.md`, this file says so instead of guessing.
>
> `tasks.md` remains the primary record. If the two ever disagree, `tasks.md` is authoritative.

## Session shape

Apply ran across **two sessions**, and the seam matters for reading the evidence below.

An initial run landed Phases 1–3 (commits `297f08d`, `78eb88a`, `ed7ffa8`) and then **disconnected before checkpointing**. A resumed session verified that work by reading it against `spec.md` and `design.md`, confirmed `make test` green, and continued from Phase 4.

The consequence, stated plainly: **Phases 1–3's RED evidence is retroactive verification, not observed RED.** `tasks.md` records this distinction itself and does not claim otherwise. Phases 4, 5 and 6 were implemented in the resumed session with genuine RED observed before each GREEN, and their transcripts are compile failures — the hardest kind to fabricate.

## Precondition (Phase 0)

AI-14 is Wave 2's foundation and had no Wave 2 predecessor to reconcile against. Its binding predecessors were AI-02 (`ai-stream-lifecycle`) and AI-04 (`ai-validation-errors`), both already merged. `openspec/specs/` carried no event capability at apply time, so `spec.md` was a full new capability spec with no delta to merge.

## Work Unit Evidence

| Evidence | Value |
|---|---|
| Focused test command and exact result | Per phase — `go test -run TestEvent -race ./src/ai/...` (AI-14.1), `go test -run 'TestStamper\|TestSequence\|TestCheckEmit' -race -v ./src/ai/...` (AI-14.2, all PASS), `go test -run 'TestSequenceGuard\|TestSequenceStateAllowlist' -race -v ./src/ai/...` (AI-14.3, 9 top-level / 12 including subtests PASS), `go test -race -v -run 'TestCheckStream\|TestEventRegistration\|TestExportedSurface\|TestEventDescriptorGoFile\|TestCheckEmit_BlockScoped' ./src/ai/...` (AI-14.4, 18 top-level PASS) |
| Runtime harness command/scenario and exact result | `make test` (`go test -race -v ./...`) in `backend/agent/` — `PASS`, `ok github.com/cachicamas/backend/agent/src/ai`. No runtime/integration boundary exists at this milestone: pure value construction and in-process AST/type scans, no I/O, no subprocess (recorded as `N/A` in every row of `tasks.md`'s Suggested Work Units table) |
| Rollback boundary | Per work unit, as `tasks.md` planned them: unit 1 (`event.go`, `event_descriptor.go`, `export_test.go`) — nothing imports them yet; unit 2 (`sequence.go`) — revert alone; unit 3 (`sequence_guard_test.go`) — revert alone; unit 4 (`stream_check.go`) — revert alone, leaving `event.go`/`sequence.go` unaffected |

## Commits

`tasks.md` names five commits directly: `297f08d` (foundation — ordering-descriptor skeleton), `78eb88a` (AI-14.1 envelope skeleton), `ed7ffa8` (AI-14.2 per-stream sequence), plus the AI-14.4 checker and the NFR close-out, which [`WAVE-2-VERIFY.md`](../WAVE-2-VERIFY.md) § 3.2 identifies as `4de995a` and `65d8be7`.

**Recorded uncertainty**: `tasks.md` does not name a commit for Phase 4 (the AI-14.3 guard), and the verify report's git log was filtered to `stream_check.go`, `event_descriptor.go` and `sequence.go` — so it would not show a commit touching only `sequence_guard_test.go`. The guard's own commit is therefore **not identified from the evidence available at archive time**. It is not claimed here.

## TDD Cycle Evidence

| Phase | Node | RED | GREEN | REFACTOR |
|---|---|---|---|---|
| 1 | foundation — descriptor skeleton | **Retroactive.** `event_descriptor_test.go`'s `TestEventDescriptor_ZeroValue_RoundTrips` (both subtests) predates `event_descriptor.go` in intent — RED implied by `297f08d`'s own "groundwork" framing, not observed | Green today: `BlockRole`, `Cardinality`, `EventDescriptor`, `BlockIndex`, unexported `blockPayload` | `gofmt -l` / `go vet` clean; no test changes needed |
| 2 | AI-14.1 envelope skeleton | **Retroactive**, verified against `78eb88a` | `TestEvent_Kind_IsDerivedFromPayloadAndNeverStored`, `TestCheckEmit_PayloadlessEvent_RejectedWithErrNotInVocabulary`, `TestEventPayload_Contract_IsSealed`, `TestEvent_ReadableExternally_NoTypeSwitchOverUnexportedTypes`, `TestEventKindRegistration_TheTestKindVocabulary_HasConstructorAndAccessor`, `TestEventKinds_ProductionVocabulary_IsEmpty` — all green | Deduped registry-table helpers against the `content_part.go` precedent |
| 3 | AI-14.2 per-stream sequence | **Retroactive**, verified against `ed7ffa8` | `TestStamper_Stamp_IsOneBasedAndContiguous`, `TestStamper_SequenceState_IsPerStreamNotProcess` (two-goroutine, `-race` clean), `TestSequence_CrossStreamComparison_IsPermittedAndMeaningless`, `TestSequenceGoFile_PackageDoc_StatesTheCrossStreamRule`, `TestCheckEmit_UnstampedSentinel_RejectedWithErrOutOfRange` — all PASS | No extraction needed — `CheckEmit` already composes its rule funcs through AI-04's existing `FirstFailure` combinator |
| 4 | AI-14.3 guard | **Observed.** `go test -run TestSequenceGuard -race -v ./src/ai/...` → build failure, **6 undefined-symbol errors**: `undefined: scanSequenceStateGuard`, `sequenceStateAllowlist`, `guardViolation`, all in `sequence_guard_test.go`. Second RED at 4.3: the two new allowlist-staleness tests FAILED against the 4.2 implementation (`violations = []`, `want one naming …`) | 4.2 → all five guard tests PASS (`S-AEE-031`…`035`). 4.4 → staleness and empty-rationale checks added, `S-AEE-036`…`039` PASS. 4.5 → doc-comment and failure-message tests PASS on first write | Guard failure-message formatting; `git diff --stat -- backend/agent/src/ai/message.go` → **empty** (`S-AEE-040` confirmed); `gofmt -l` empty; `go vet` clean |
| 5 | AI-14.4 ordering invariants | **Observed.** `go vet ./src/ai/...` → build failure, `vet: src/ai/stream_check_test.go:78:16: undefined: ai.CheckStream` — `stream_check.go`, `DescriptorOf` and the new registry/descriptor tests were all written before `stream_check.go` existed | `stream_check.go` (`CheckStream`, `StreamReport`, `checkBlockOrdering`, `blockIndexOf`), `CheckEmit` rules 3 and 4, `DescriptorOf` bridge → 18 top-level tests PASS **on the first implementation attempt**, no iteration for correctness, only one `gofmt -w` pass | Extracted `verdict(rule error, terminated bool, at ...Step) StreamReport`, consolidating five near-identical `StreamReport{err: Invalid(…)}` composite literals into one constructor (design.md D7's verdict map, now in one place). Re-ran `make test` → still PASS |
| 6 | Cross-cutting NFRs and closeout | **Observed.** `go vet ./src/ai/...` → build failure, `zero.String undefined (type ai.Event has no field or method String)` — the totality and rendering tests were written before `Event.String`/`GoString` existed | Added `(Event) String()`/`GoString()` → both tests PASS (11 subtests) on the first attempt. The totality table needed **no other production change**: nil/empty stream, zero `Event`, zero `Sequence`, all-unstamped stream and wrong-kind accessor were already panic-free against the Phase 1–5 implementation | See "Lint gap found and fixed" below |

**Note on the two green-from-birth pins.** `TestSequenceStateAllowlist_HasExactlyOneReasonedEntry` and `TestSequenceGuard_AllowlistEntryRemoved_FailsNamingLastMessageID` passed immediately at 4.3 and are documented as **pins** — green-from-birth regression assertions, exempt from red-first under doc 0002's leaf-anatomy rule, the same exemption `import_boundary_test.go`'s `TestLayer1_ModuleHasNoDependencies_ZeroRequires` already uses in this package. They assert a property already true of 4.2's data and logic (`S-AEE-036`, `S-AEE-037`). Recorded as pins rather than claimed as observed RED.

**Note on how GREEN was reached first-try twice.** Both Phase 4.2 and Phase 5 reached GREEN on the first implementation attempt. `tasks.md` explains 4.2: the landed package was scanned with a **throwaway `go/types` prototype before the guard was written**, confirming exactly one qualifying package-level var (`message.go`'s `lastMessageID`, a `sync/atomic.Uint64`) across all 21 non-test files. Phase 5 records the outcome without a comparable explanation, and volunteers that it was formatting-only iteration — an admission that lowers the apparent heroism of the record.

### Test Summary

- **Phases**: 6 · **Tasks**: 55 / 55 complete
- **Test files**: `event_descriptor_test.go`, `event_test.go`, `event_registry_test.go`, `sequence_test.go`, `sequence_guard_test.go`, `stream_check_test.go`, plus the `export_test.go` bridge
- **Layers used**: Unit only, in the external `ai_test` package, plus two in-process structural scans (`go/parser` + `go/types` for the AI-14.3 guard; AST doc-comment reads for the package-doc assertions). No E2E — Layer 1 has no runnable surface
- **Guard bite proofs**: five permanent scratch-package bites (`TestSequenceGuard_ScratchPackageLevelInteger_Fails` and four siblings), each recorded RED then dropped
- **Concurrency posture**: every test calling `RegisterTestKind` (directly or via `stream_check_test.go`'s `streamKind` helper) omits `t.Parallel()` throughout its scope, matching `export_test.go`'s own documented constraint — `eventRegistry` is one shared package-level slice, and a concurrent truncation from a parallel test would corrupt it. Verified clean under `go test -race`

## Files Changed

| File | Action | What Was Done |
|---|---|---|
| `backend/agent/src/ai/event_descriptor.go` | Created | `BlockRole` (`None`/`Start`/`Delta`/`End`), `Cardinality` (`Any`/`AtMostOne`), `EventDescriptor`, `BlockIndex uint64`, unexported `blockPayload`; the 6-step kind-adding doc comment plus the fragment-only / no-snapshot paragraph |
| `backend/agent/src/ai/event_descriptor_test.go` | Created | Zero-value round-trip; the doc-comment AST assertion for `S-AEE-065` |
| `backend/agent/src/ai/event.go` | Created | `EventKind`, `Event{payload, seq}`, sealed `eventPayload`, `eventRegistry`, `EventKinds()`, `CheckEmit`'s four rules, `Kind()`, `Sequence()`, `String()`/`GoString()` |
| `backend/agent/src/ai/event_test.go` | Created | Kind derivation, payload-less rejection, seal, external readability, unstamped sentinel, zero-block-index rejection, totality table, rendering canary |
| `backend/agent/src/ai/event_registry_test.go` | Created | Exhaustiveness assertion, descriptor-domain pinning, empty-production-vocabulary assertion |
| `backend/agent/src/ai/export_test.go` | Created | Test-only bridge: `KindTestWitness`, `WitnessPayload`, `NewWitnessEvent`, `RegisterTestKind`, `NewTestEvent`, `AllTestEventKinds`, `DescriptorOf` |
| `backend/agent/src/ai/sequence.go` | Created | `Sequence uint64`, `Stamper{last Sequence}`, `(*Stamper) Stamp`, and the cross-stream-meaninglessness package-doc paragraph |
| `backend/agent/src/ai/sequence_test.go` | Created | Contiguity, per-stream independence under `-race`, cross-stream comparison, the doc-reading assertion |
| `backend/agent/src/ai/sequence_guard_test.go` | Created | The `go/parser` + `go/types` package-wide scan, the one-entry allowlist, five scratch bites, staleness and empty-rationale checks |
| `backend/agent/src/ai/stream_check.go` | Created | `CheckStream`, `StreamReport`, `checkBlockOrdering`, `blockIndexOf`, `verdict` |
| `backend/agent/src/ai/stream_check_test.go` | Created | Block ordering, interleaving, unterminated blocks, cardinality, terminality, first-violation ordering, non-mutation, idempotence |
| `backend/agent/src/ai/doc.go` | Modified | Package-doc paragraph pointing at `sequence.go`'s cross-stream rule (landed in `ed7ffa8`; Phase 6.5 verified only, no new edit) |
| `backend/agent/src/ai/message.go` | **Untouched** | Confirmed empty diff — the allowlist accommodates the shipped `lastMessageID` counter rather than the milestone rewriting it (`S-AEE-040`) |
| `openspec/changes/cachicamas-ai-event-envelope/tasks.md` | Modified | All 55 tasks marked `[x]`, with the inline Evidence Log this file is synthesized from |

## Deviations from Design

Four, all disclosed by the original session in `tasks.md` and reproduced here without softening. The Wave 2 verify report catalogued them as **D1** … **D4** and cross-checked each.

1. **`CheckEmit` rule 4 has no failure-path test** (task 5.2). `design.md` D5 specifies four `CheckEmit` rules; `tasks.md`'s Phase 5 names only rule 3 as a task. Rule 4 (`e.payload.validate(…)`) **is implemented** and matches D5 — every existing `CheckEmit`-success test continues to pass, proving it correctly no-ops today — but it has no dedicated failure-path unit test at this milestone. The reason is structural: the only payload type AI-14 can construct is `WitnessPayload`, whose `validate()` is hardcoded to return `nil` per `export_test.go`'s own documented rationale ("a rule is added once a test needs one to fail"). Extending `WitnessPayload`'s exported construction surface to force a controllable failure would exceed `design.md` D6's binding, named shape for AI-15 … AI-20. The original disclosure deferred the coverage to "AI-15+, whose real payloads carry non-trivial `validate()` rules".
   > **Archive-time follow-up.** That deferral was **never discharged**, and the reason is not negligence: AI-18's design gate — working correctly — discovered that AI-15 … AI-19 all validate **eagerly in their constructors** and return `(Event, error)`, so an `Event` carrying a payload that fails its own `validate` is unconstructible through the public API. The receiver the deferral depended on was designed away, and no one connected the two decisions. Escalated to **W1** in the Wave 2 verify report and owned by Wave 3. Not a defect in this milestone's disclosure, which was accurate when written.
2. **`R-AEE-014` has no dedicated `S-AEE` scenario for its `CheckEmit`-boundary rejection** (task 5.2). The requirement's own text ("1-based with 0 rejected") is not enumerated as a scenario — `S-AEE-043`/`044`/`045` cover the descriptor table, not this rejection. It was tested directly against the requirement's prose instead, via `TestCheckEmit_BlockScopedEventWithZeroBlockIndex_RejectedWithErrOutOfRange` in `event_test.go`. Benign: the test exists and passes.
3. **`R-AEE-009`'s cross-stream doc paragraph landed in `sequence.go`'s header, not `doc.go`** (task 3.6). `tasks.md`'s literal wording said `doc.go`; `design.md`'s File Changes entry specified `sequence.go` ("docs: … cross-stream meaninglessness (R-AEE-009)"). **`design.md` is authoritative for file placement**, so the landed location is correct and `tasks.md` records the divergence rather than hiding it. The test reads it via `file.Comments[0]`.
4. **The witness-enumeration helper is named `AllTestEventKinds`, not `TestEventKinds`** (task 2.8). `design.md` D6 names it `TestEventKinds`; `go vet`'s `tests` analyzer requires any top-level `Test*` identifier in a `_test.go` file to have signature `func(t *testing.T)`. Renamed to satisfy the analyzer. `S-AEE-013`/`017` are unaffected — semantics unchanged, name only.

## Issues Found

**Lint gap found and fixed in Phase 6.** `make lint` (`golangci-lint`) had **not been run** by either the disconnected first session or the resumed one until Phase 6. It reported four `revive` findings:

1. `event_descriptor.go` — header comment attached directly to `package ai` with no blank line, competing with `doc.go` for "the" package comment.
2. `sequence.go` — same finding.
3. `stream_check.go` — same finding, **introduced by this session**.
4. `export_test.go` — `WitnessPayload.validate(at Path)` had an unused parameter (pre-existing since Phase 2; `at` was never read because the witness validates nothing at this milestone).

Fixed by separating `event.go`, `event_descriptor.go`, `sequence.go` and `stream_check.go`'s header comments from `package ai` with a blank line — leaving `doc.go` as the sole file carrying the attached package doc, the standard one-`doc.go`-per-package convention — and renaming `validate`'s parameter to `_`.

**The fix broke two existing tests, and that is recorded rather than quietly repaired.** `TestSequenceGoFile_PackageDoc_StatesTheCrossStreamRule` (pre-existing from Phase 3) and `TestEventDescriptorGoFile_Doc_StatesDeltaIsFragmentOnlyAndForbidsSnapshot` (this session's own, from Phase 5) both read `file.Doc`, which became `nil` once the comment detached from the package clause. Both were changed to read `file.Comments[0]` — still the file's leading comment group; only its attachment changed. Re-ran `make test` → PASS; re-ran `make lint` → **`0 issues`**.

> This finding class propagated: AI-17 hit the identical `revive` `package-comments` gap on `reasoning_event.go` and applied the same blank-line fix, closing the one lint issue AI-16 had left open at its own close (verify report **D10** → **D12**).

## NFR verification (Phase 6.6 – 6.9)

| NFR | Scenario | Evidence |
|---|---|---|
| **NFR-AEE-A** dependency purity | `S-AEE-066` | `go.mod` still `module github.com/cachicamas/backend/agent` + `go 1.26.3`, **zero `require` block**. `go test -race -v -run 'TestLayer1_ImportsOnlyStdlibAndItsOwnPackages_DenyByDefault\|TestLayer1_ModuleHasNoDependencies_ZeroRequires'` → both PASS |
| **NFR-AEE-B** totality | `S-AEE-067` | One consolidated table, `TestEvent_ExtremeInputs_NeverPanics` in `event_test.go`; nil/empty stream additionally covered by Phase 5's `TestCheckStream_NilAndEmptySlice_SafeAndReportNoViolation` |
| **NFR-AEE-C** failure reporting | `S-AEE-068` | `git log --oneline -1 -- backend/agent/src/ai/validation.go` → last touched by `0c6b8ca` (AI-10.3), confirming **no AI-14 commit ever edited `validation.go`**. `grep` for `errors.New(` across every AI-14 production file → **zero matches**. Every rejection uses one of AI-04's five pre-existing sentinels (`ErrNotInVocabulary`, `ErrOutOfRange`, `ErrMisplaced`, `ErrDuplicate`, `ErrMalformed`) |
| **NFR-AEE-D** redaction | `S-AEE-069` | Block-index canary (`canaryBlockIndex = 424242`): `String()`, `GoString()`, `%v`, `%s`, `%+v`, `%#v` all render exactly `"event(test_witness seq=N)"`, never the block index. Mirrors `content_part_test.go`'s `TestPart_String_CarriesNoPayload`. **Honest scope note from `tasks.md`**: AI-14 registers no production kind carrying arbitrary caller bytes (`R-AEE-006`), so there is no string or byte secret to plant at this milestone; the block index stands in as the value that must not leak. AI-15+'s payloads inherit this same `String()`/`GoString()` with no override needed, because `Event` never reads a payload field other than `Kind()` |
| **NFR-AEE-E** evidence | `S-AEE-070` | `tasks.md`'s Evidence Log, and this file |

**Final gate (6.9)**: `make test` (`go test -race -v ./...`) from `backend/agent/` → `PASS`, `ok github.com/cachicamas/backend/agent/src/ai`. `make lint` → `0 issues`. `gofmt -l src/ai/*.go` → empty.

## Remaining Tasks

None. 55/55 complete.

## Workload / PR Boundary

- Mode: `size:exception` per `tasks.md`'s Review Workload Forecast — `exception-ok`, `400-line budget risk: High`, `Decision needed before apply: No`, `Chained PRs recommended: No`, single PR under the accepted wave-wide ~5000-line budget
- Forecast: ~2600–3400 changed lines (7 production/guard files + 4 scenario test files + `export_test.go`). **Actual authored: 2 903 lines** — inside the forecast band, and the closest of Wave 2's seven forecasts to its own upper bound
- Boundary: AI-14 is Wave 2's first milestone and has no Wave 2 predecessor. It starts from the Wave 1 head and ends at the NFR close-out commit (`65d8be7`)
- The four internal work units were review and rollback granularity only, all landed in the one PR-bound branch

## Status

55/55 tasks complete. `make test` green under `-race`. `make lint` `0 issues`. Both AI-00 import guards passing, `go.mod` still zero requires. Verified **PASS** by [`WAVE-2-VERIFY.md`](../WAVE-2-VERIFY.md) § 4, which records AI-14 as "Met, and it carries the wave", and confirms from git that **no commit after this milestone's own NFR close-out touched `stream_check.go`, `event_descriptor.go` or `sequence.go`** across five sibling milestones registering twelve kinds — the strongest available evidence for `R-AEE-015`.
