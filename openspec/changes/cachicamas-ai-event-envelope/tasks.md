# Tasks: Event envelope with per-stream sequencing (AI-14)

> Nodes: AI-14.1 envelope skeleton · AI-14.2 per-stream sequence · AI-14.3 no-process-global guard · AI-14.4 ordering invariants.
> Package `backend/agent/src/ai`. Strict TDD: every item RED → GREEN → REFACTOR, in spec order. Runner: `make test` (`go test -race -v ./...`) from `backend/agent/`.

## Review Workload Forecast

| Field | Value |
|-------|-------|
| Estimated changed lines | ~2600–3400 (7 production/guard files + 4 scenario test files + export_test.go, vs. `content_part.go` precedent: 316 prod + 1197 test lines for a smaller surface) |
| 400-line budget risk | High |
| Chained PRs recommended | No — resolved directly to size:exception per accepted wave-wide 5000-line budget |
| Suggested split | Single PR (internal work units below for review/rollback granularity only) |
| Delivery strategy | exception-ok |
| Chain strategy | size-exception |

Decision needed before apply: No
Chained PRs recommended: No
Chain strategy: size-exception
400-line budget risk: High

### Suggested Work Units (internal granularity, not separate PRs)

| Unit | Goal | Likely PR | Focused test command | Runtime harness | Rollback boundary |
|------|------|-----------|----------------------|-----------------|-------------------|
| 1 | AI-14.1 envelope: `event.go`, `event_descriptor.go`, `export_test.go` witness | PR 1 (exception) | `go test -run TestEvent -race ./backend/agent/src/ai/...` | N/A — pure library, no provider/runtime integration at this milestone | Revert the three files; nothing else imports them yet |
| 2 | AI-14.2 sequence: `sequence.go` | PR 1 (exception) | `go test -run TestSequence -race ./backend/agent/src/ai/...` | N/A | Revert `sequence.go` alone |
| 3 | AI-14.3 guard: `sequence_guard_test.go` + allowlist | PR 1 (exception) | `go test -run TestSequenceGuard ./backend/agent/src/ai/...` | N/A — in-process `go/types` scan, no subprocess | Revert `sequence_guard_test.go` alone |
| 4 | AI-14.4 invariants: `stream_check.go` | PR 1 (exception) | `go test -run TestCheckStream -race ./backend/agent/src/ai/...` | N/A | Revert `stream_check.go` alone; `event.go`/`sequence.go` unaffected |

## Phase 1: Foundation — descriptor skeleton (blocks all nodes)

- [x] 1.1 RED `event_descriptor_test.go`: assert `BlockRoleNone`/`CardinalityAny` are zero values, `EventDescriptor{}` zero-value round-trips (groundwork for R-AEE-014).
- [x] 1.2 GREEN `event_descriptor.go`: `BlockRole` (`BlockRoleNone=0`, `BlockRoleStart`, `BlockRoleDelta`, `BlockRoleEnd`), `Cardinality` (`CardinalityAny=0`, `CardinalityAtMostOne`), `EventDescriptor{Role BlockRole; Cardinality Cardinality; Terminal bool}`, `BlockIndex uint64`, unexported `blockPayload interface{ blockIndex() BlockIndex }`.
- [x] 1.3 REFACTOR: `gofmt`/`go vet` clean; no test changes needed.

## Phase 2: AI-14.1 — Envelope skeleton

- [x] 2.1 RED `event_test.go` R-AEE-001 (S-AEE-001..003): kind derived from payload, no settable kind field.
- [x] 2.2 GREEN `event.go`: `EventKind uint8`, `eventKindFirst`/`eventKindEnd = 1` (empty vocab), `Event{payload eventPayload; seq Sequence}`, unexported `eventPayload interface{ kind() EventKind; validate(at Path) *Violation }`, `(Event) Kind()`.
- [x] 2.3 RED R-AEE-002 (S-AEE-004..006): payload-less event → `ErrNotInVocabulary`.
- [x] 2.4 GREEN `event.go`: `CheckEmit(e Event) error` rule 1 (vocabulary) at `At("event")`.
- [x] 2.5 RED R-AEE-003 (S-AEE-007..009): `package ai_test` type fails to satisfy sealed `eventPayload`; all members unexported.
- [x] 2.6 GREEN: confirm `eventPayload` fully unexported (structural — no new production code).
- [x] 2.7 RED `event_registry_test.go` R-AEE-004 (S-AEE-010..013): exhaustiveness assertion, non-vacuous via test-only witness.
- [x] 2.8 GREEN `export_test.go` (D6): `KindTestWitness`, `WitnessPayload`, `NewWitnessEvent(block BlockIndex) Event`, `(Event) WitnessPayload() (WitnessPayload, bool)`, `RegisterTestKind`, `NewTestEvent`, `TestEventKinds()`; GREEN `event.go`: `eventRegistry []eventRegistration{name string; descriptor EventDescriptor}`, `EventKinds() []EventKind`.
- [x] 2.9 RED R-AEE-005 (S-AEE-014..016): external read via accessor, no type switch; mismatched accessor fails not panics.
- [x] 2.10 GREEN `event.go`: `(Event) Sequence()`; witness accessor pattern proven through `export_test.go`.
- [x] 2.11 RED `event_registry_test.go` R-AEE-006 (S-AEE-017..019): zero production kinds enumerated.
- [x] 2.12 GREEN: confirm `eventKindFirst == eventKindEnd == 1`, no production kind constants exist.
- [x] 2.13 REFACTOR: dedupe registry-table helpers against `content_part.go` precedent; `make test` for AI-14.1 slice; record red/green output.

## Phase 3: AI-14.2 — Per-stream sequence

- [x] 3.1 RED `sequence_test.go` R-AEE-007 (S-AEE-020..022): 1-based contiguous stamping; no external sequence setter; no reachable reset.
- [x] 3.2 GREEN `sequence.go`: `Sequence uint64`, `Stamper{last Sequence}`, `(*Stamper) Stamp(e Event) Event`.
- [x] 3.3 RED R-AEE-008 (S-AEE-023..025): two streams concurrently under `-race`; `Stamper` holds no package-level/atomic/mutex state.
- [x] 3.4 GREEN: confirm zero-value-ready `Stamper`, no shared state (structural + doc comment).
- [x] 3.5 RED doc-assertion test R-AEE-009 (S-AEE-026..027): package doc states cross-stream comparison is permitted and meaningless.
- [x] 3.6 GREEN `doc.go`: cross-stream rule paragraph. *(Landed as sequence.go's own package-doc comment, which `TestSequenceGoFile_PackageDoc_StatesTheCrossStreamRule` parses directly — matches design.md's File Changes entry for sequence.go, "docs: ... cross-stream meaninglessness (R-AEE-009)", rather than tasks.md's literal "doc.go" wording. design.md is authoritative for file placement.)*
- [x] 3.7 RED `event_test.go` R-AEE-010 (S-AEE-028..030): unstamped sentinel `0` rejected `ErrOutOfRange` at the `CheckEmit` boundary; stamped seq `1` accepted.
- [x] 3.8 GREEN `event.go`: `CheckEmit` rule 2 (`seq == 0` → `ErrOutOfRange` at `At("event"), At("sequence")`).
- [x] 3.9 REFACTOR: extract shared `CheckEmit` rule-ordering helper; `make test` for AI-14.2 slice; record red/green output. *(The shared rule-ordering helper is AI-04's existing `FirstFailure` combinator — `CheckEmit` already composes its rule funcs through it; no further extraction was needed.)*

## Phase 4: AI-14.3 — No process-global sequence-state guard

- [x] 4.1 RED `sequence_guard_test.go` R-AEE-011 (S-AEE-031..035): fails scratch `var scratchSeq uint64`, struct-wrapped counter, reset func; passes on landed package and on `_test.go` counters.
- [x] 4.2 GREEN `sequence_guard_test.go`: D8 `go/parser` + `go/types` guard — Named→Underlying (`types.Unalias`), struct fields any depth, array elements; base match = integer basics + `uintptr` + `sync/atomic` types; no depth cap; bare type params fail-closed; reset detection via `types.Info.Uses` (assign/inc-dec/`Store`/`Swap`/`CompareAndSwap`, `Add` legal).
- [x] 4.3 RED R-AEE-012 (S-AEE-036..039): allowlist has exactly one entry (`message.go`/`lastMessageID`); stale or empty-rationale entries fail.
- [x] 4.4 GREEN: `allowlistEntry{file, identifier, rationale}`, `sequenceStateAllowlist` table naming C3 and the `V-REQ-03` (distinguishability) vs `V-STR-13` (contiguity) distinction.
- [x] 4.5 RED+GREEN R-AEE-013 (S-AEE-041..042): guard source comment names C3, the retired process-global counter, and "not a smaller counter; the counter where the stream is"; failure message names file+identifier.
- [x] 4.6 REFACTOR: guard failure-message formatting; confirm `message.go`'s `lastMessageID` diff is empty (S-AEE-040); `make test` for AI-14.3 slice; record red/green output.

## Phase 5: AI-14.4 — Ordering invariants

- [x] 5.1 RED `event_registry_test.go` R-AEE-014 (S-AEE-043..045): every registered kind has a descriptor from stated domains; block-role kinds expose a readable block index.
- [x] 5.2 GREEN: wire `EventDescriptor` into `eventRegistration`; `event.go` `CheckEmit` rule 3 (block index vs `Role`, `Role≠None` requires index ≥1) → `ErrOutOfRange` at `At("event"), At("block")`. *(`eventRegistration.descriptor` was already wired in Phase 1/2. Additionally implemented design.md D5's rule 4 (`payload.validate`), which tasks.md does not separately enumerate but design.md requires — see Evidence Log for the honest gap note on its test coverage.)*
- [x] 5.3 RED `stream_check_test.go` R-AEE-015 (S-AEE-046..048): checker reads only `(kind, descriptor, block index, sequence)`, no concrete payload type referenced.
- [x] 5.4 GREEN `stream_check.go`: `CheckStream(events []Event) StreamReport`, `StreamReport{err error; terminated bool}`, `(StreamReport) Violation() error`, `(StreamReport) Terminated() bool`.
- [x] 5.5 RED R-AEE-016 (S-AEE-049..053): block ordering start≺delta≺end per index; interleaving legal; unterminated block distinguishable from ordering violation.
- [x] 5.6 GREEN: block-ordering pass in `CheckStream` per D7 verdict map (`ErrMisplaced` for out-of-order/after-terminal, `ErrMalformed` for unterminated), positions `AtIndex("event", seq)` + `AtIndex("block", idx)`.
- [x] 5.7 RED R-AEE-017 (S-AEE-054..056): `at-most-one` cardinality via witness kind → `ErrDuplicate` naming kind + second event's sequence.
- [x] 5.8 GREEN: cardinality pass in `CheckStream`.
- [x] 5.9 RED R-AEE-018 (S-AEE-057..060): `terminal` primitive; event-after-terminal and second-terminal → `ErrDuplicate`/`ErrMisplaced`; "no terminal event" reported as distinct informational outcome via `Terminated() == false`.
- [x] 5.10 GREEN: terminal pass in `CheckStream`; `StreamReport.Terminated()`.
- [x] 5.11 RED R-AEE-019 (S-AEE-061..063): first violation in stream order; slice input never mutated; no channel; idempotent re-run.
- [x] 5.12 GREEN: finalize `CheckStream` first-violation ordering and non-mutation contract.
- [x] 5.13 RED R-AEE-020 (S-AEE-064..065): no exported accumulator/transcript-rebuilder/reducer of deltas; add-a-delta-kind doc states fragment-only, forbids snapshot.
- [x] 5.14 GREEN `event_descriptor.go`: 6-step kind-adding doc comment (constant+move end, payload, constructor, accessor, registry line w/ descriptor, doc list). *(The 6-step list itself landed in Phase 1; this step added the "fragment-only, forbid snapshot" paragraph the same file comment needed for S-AEE-065.)*
- [x] 5.15 REFACTOR: consolidate `CheckStream` verdict-mapping switch; `make test` for AI-14.4 slice; record red/green output.

## Phase 6: Cross-cutting NFRs and closeout

- [ ] 6.1 RED `event_test.go`/`sequence_test.go`/`stream_check_test.go`: NFR-AEE-B totality table (S-AEE-067) — zero `Event`, zero `Sequence`, nil/empty stream, all-unstamped stream, wrong-kind accessor never panic.
- [ ] 6.2 GREEN: totality guards across `event.go`/`sequence.go`/`stream_check.go`.
- [ ] 6.3 RED `event_test.go`: NFR-AEE-D redaction (S-AEE-069) — `%v`/`%s`/`%#v` never render payload bytes or a derived length.
- [ ] 6.4 GREEN `event.go`: `(Event) String()`/`GoString()` render `"event(<kind> seq=N)"` only.
- [ ] 6.5 GREEN `doc.go`: finalize package doc paragraph (events exist; pointer to `sequence.go`'s cross-stream rule).
- [ ] 6.6 Verify NFR-AEE-A (S-AEE-066): `go.mod` still zero requires; both AI-00 import guards pass.
- [ ] 6.7 Verify NFR-AEE-C (S-AEE-068): every rejecting scenario resolves through AI-04's failure value/landed sentinels; no new sentinel introduced.
- [ ] 6.8 Record NFR-AEE-E (S-AEE-070) evidence: red/green output + refactor note per test-list item, per node, in this file.
- [ ] 6.9 Run `make test` (`go test -race -v ./...`) and `make lint` from `backend/agent/`; confirm green/clean before archive.

> **Deviation note**: exceeds the sdd-tasks 530-word budget. `NFR-AEE-E` requires every one of 20 requirements' test-list items tracked red→green→refactor with recorded output, and five sibling milestones (AI-15…AI-20) bind to these exact symbol names — the spec and design artifacts for this same change recorded the identical deviation for the same reason; house convention wins.

## Evidence Log (NFR-AEE-E, S-AEE-070)

Apply ran across two sessions: an initial run landed Phases 1–3 (commits `297f08d`, `78eb88a`, `ed7ffa8`) but disconnected before checkpointing; this resumed session verified that work by reading it against spec/design, confirmed `make test` green, then continued from Phase 4. Per-item red/green output below; full raw terminal logs are not pasted verbatim (would dominate this file) — each row names the exact test(s) and the observed result, which any reviewer can reproduce with the focused command shown per phase.

**Phase 1 (descriptor skeleton)** — verified retroactively against `297f08d`: `event_descriptor_test.go`'s `TestEventDescriptor_ZeroValue_RoundTrips` (both subtests) predates `event_descriptor.go` in intent (RED implied by the commit's own "groundwork" framing) and passes green today. `gofmt -l`/`go vet` clean.

**Phase 2 (AI-14.1 envelope)** — verified retroactively against `78eb88a`: `TestEvent_Kind_IsDerivedFromPayloadAndNeverStored`, `TestCheckEmit_PayloadlessEvent_RejectedWithErrNotInVocabulary`, `TestEventPayload_Contract_IsSealed`, `TestEvent_ReadableExternally_NoTypeSwitchOverUnexportedTypes` (event_test.go), `TestEventKindRegistration_TheTestKindVocabulary_HasConstructorAndAccessor`, `TestEventKinds_ProductionVocabulary_IsEmpty` (event_registry_test.go) all green today against `event.go`/`export_test.go`. The commit's own message records one deviation: design.md D6 names the witness-enumeration helper `TestEventKinds`; go vet's `tests` analyzer requires any `_test.go` top-level `Test*` identifier to have signature `func(t *testing.T)`, so it ships as `AllTestEventKinds` instead (S-AEE-013/017 unaffected — semantics unchanged, name only).

**Phase 3 (AI-14.2 sequence)** — verified retroactively against `ed7ffa8`: `TestStamper_Stamp_IsOneBasedAndContiguous`, `TestStamper_SequenceState_IsPerStreamNotProcess` (two-goroutine, `-race` clean), `TestSequence_CrossStreamComparison_IsPermittedAndMeaningless`, `TestSequenceGoFile_PackageDoc_StatesTheCrossStreamRule`, `TestCheckEmit_UnstampedSentinel_RejectedWithErrOutOfRange` (sequence_test.go) all green today. Focused command: `go test -run 'TestStamper|TestSequence|TestCheckEmit' -race -v ./src/ai/...` — all PASS.

**Phase 4 (AI-14.3 guard)** — implemented this session, genuine RED observed before each GREEN:
- 4.1 RED: `go test -run TestSequenceGuard -race -v ./src/ai/...` → build failure, `undefined: scanSequenceStateGuard` / `sequenceStateAllowlist` / `guardViolation` (6 undefined-symbol errors, all in `sequence_guard_test.go`).
- 4.2 GREEN: same command → `TestSequenceGuard_LandedPackage_Passes`, `_ScratchPackageLevelInteger_Fails`, `_ScratchStructWrappedInteger_Fails`, `_ScratchResetFunction_Fails`, `_TestFileCounter_Passes` all PASS (S-AEE-031..035). `TestSequenceGuard_LandedPackage_Passes` scanning the real package found exactly one qualifying package-level var (`message.go`'s `lastMessageID`, `sync/atomic.Uint64`) across all 21 non-test files — confirmed independently via a throwaway `go/types` prototype before writing the guard, so GREEN was reached on the first implementation attempt with no further iteration.
- 4.3 RED: added `TestSequenceGuard_AllowlistEntryNamesNonexistentIdentifier_FailsAsStale` and `TestSequenceGuard_AllowlistEntryHasEmptyRationale_Fails` → both FAILED against the 4.2 implementation (`violations = []`, `want one naming ...`), confirming staleness/empty-rationale detection did not yet exist. `TestSequenceStateAllowlist_HasExactlyOneReasonedEntry` and `TestSequenceGuard_AllowlistEntryRemoved_FailsNamingLastMessageID` passed immediately (documented as pins, green-from-birth, exempt from red-first — same exemption `import_boundary_test.go`'s `TestLayer1_ModuleHasNoDependencies_ZeroRequires` already uses in this package) since they assert a property already true of 4.2's data/logic (S-AEE-036, S-AEE-037).
- 4.4 GREEN: added staleness + empty-rationale checks to `scanSequenceStateGuard` → all of the above PASS (S-AEE-036..039).
- 4.5 RED+GREEN: added `TestSequenceGuardGoFile_PackageDoc_NamesC3AndTheFix` and `TestSequenceGuard_FailureMessage_NamesFileIdentifierAndPointsAtRationale` → both PASS against the doc comment and `guardViolation.String()`/reason text already written (S-AEE-041..042).
- 4.6 REFACTOR: `git diff --stat -- backend/agent/src/ai/message.go` → empty (S-AEE-040, confirmed). `gofmt -l src/ai/sequence_guard_test.go` → empty. `go vet ./src/ai/...` → clean. Focused command `go test -run 'TestSequenceGuard|TestSequenceStateAllowlist' -race -v ./src/ai/...` → 9 top-level tests (12 including subtests) PASS. Full `make test` → PASS, `ok github.com/cachicamas/backend/agent/src/ai`.

**Phase 5 (AI-14.4 ordering invariants)** — implemented this session:
- 5.1/5.3 RED: `go vet ./src/ai/...` → build failure, `vet: src/ai/stream_check_test.go:78:16: undefined: ai.CheckStream` (stream_check.go, `DescriptorOf`, and the new registry/descriptor tests were all written before any of `stream_check.go` existed).
- 5.2/5.4/5.6/5.8/5.10/5.12 GREEN: added `stream_check.go` (`CheckStream`, `StreamReport`, `checkBlockOrdering`, `blockIndexOf`), `event.go` `CheckEmit` rules 3 and 4, `export_test.go`'s `DescriptorOf` bridge, and new tests in `event_registry_test.go`, `event_test.go`, `event_descriptor_test.go`, `stream_check_test.go` (new file). `go test -race -v -run 'TestCheckStream|TestEventRegistration|TestExportedSurface|TestEventDescriptorGoFile|TestCheckEmit_BlockScoped' ./src/ai/...` → every test (18 top-level, including subtests) PASS on the first implementation attempt — no iteration needed for correctness, only one `gofmt -w` pass for formatting. Full `make test` → PASS.
- 5.15 REFACTOR: extracted `verdict(rule error, terminated bool, at ...Step) StreamReport` in `stream_check.go`, consolidating five near-identical `StreamReport{err: Invalid(...), ...}` composite literals into one constructor (design.md D7's verdict mapping, now in one place). Re-ran `make test` → still PASS.
- **Task-sequencing note (5.2)**: `eventRegistration.descriptor` was already wired to `EventDescriptor` in Phase 1/2 (predates this task by design) — 5.2's genuinely new work was `CheckEmit` rules 3 and 4.
- **Deviation/gap note (5.2, honest disclosure per "note design gaps, don't silently deviate")**: design.md D5 specifies four `CheckEmit` rules; tasks.md's Phase 5 only names rule 3 as a task. Rule 4 (`e.payload.validate(...)`) is implemented (matches D5, and every existing CheckEmit-success test continues to pass, proving it correctly no-ops today) but has no dedicated failure-path unit test at this milestone: the only payload type AI-14 can construct, `WitnessPayload`, has a `validate()` hardcoded to return `nil` per export_test.go's own documented rationale ("a rule is added once a test needs one to fail"). Extending `WitnessPayload`'s exported construction surface to force a controllable failure would exceed design.md D6's binding, named shape for AI-15…AI-20. AI-15+ (whose real payloads carry non-trivial `validate()` rules) will be the first to exercise this path.
- **Deviation note (5.2, honest disclosure)**: R-AEE-014's own requirement text ("1-based with 0 rejected") is not enumerated as its own S-AEE scenario for the `CheckEmit`-boundary rejection specifically (S-AEE-043/044/045 cover the descriptor table, not this rejection). Tested directly against the requirement's prose instead (`TestCheckEmit_BlockScopedEventWithZeroBlockIndex_RejectedWithErrOutOfRange`, event_test.go).
- **Concurrency note**: every test that calls `RegisterTestKind` (directly or via `stream_check_test.go`'s `streamKind` helper) does NOT call `t.Parallel()` anywhere in its scope, matching export_test.go's own documented constraint (`eventRegistry` is one shared package-level slice; a concurrent truncation from another parallel test would corrupt it, and `-race` would catch the underlying data race regardless of whether the resulting assertion happened to still be correct). Verified clean under `go test -race`.
