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

- [ ] 1.1 RED `event_descriptor_test.go`: assert `BlockRoleNone`/`CardinalityAny` are zero values, `EventDescriptor{}` zero-value round-trips (groundwork for R-AEE-014).
- [ ] 1.2 GREEN `event_descriptor.go`: `BlockRole` (`BlockRoleNone=0`, `BlockRoleStart`, `BlockRoleDelta`, `BlockRoleEnd`), `Cardinality` (`CardinalityAny=0`, `CardinalityAtMostOne`), `EventDescriptor{Role BlockRole; Cardinality Cardinality; Terminal bool}`, `BlockIndex uint64`, unexported `blockPayload interface{ blockIndex() BlockIndex }`.
- [ ] 1.3 REFACTOR: `gofmt`/`go vet` clean; no test changes needed.

## Phase 2: AI-14.1 — Envelope skeleton

- [ ] 2.1 RED `event_test.go` R-AEE-001 (S-AEE-001..003): kind derived from payload, no settable kind field.
- [ ] 2.2 GREEN `event.go`: `EventKind uint8`, `eventKindFirst`/`eventKindEnd = 1` (empty vocab), `Event{payload eventPayload; seq Sequence}`, unexported `eventPayload interface{ kind() EventKind; validate(at Path) *Violation }`, `(Event) Kind()`.
- [ ] 2.3 RED R-AEE-002 (S-AEE-004..006): payload-less event → `ErrNotInVocabulary`.
- [ ] 2.4 GREEN `event.go`: `CheckEmit(e Event) error` rule 1 (vocabulary) at `At("event")`.
- [ ] 2.5 RED R-AEE-003 (S-AEE-007..009): `package ai_test` type fails to satisfy sealed `eventPayload`; all members unexported.
- [ ] 2.6 GREEN: confirm `eventPayload` fully unexported (structural — no new production code).
- [ ] 2.7 RED `event_registry_test.go` R-AEE-004 (S-AEE-010..013): exhaustiveness assertion, non-vacuous via test-only witness.
- [ ] 2.8 GREEN `export_test.go` (D6): `KindTestWitness`, `WitnessPayload`, `NewWitnessEvent(block BlockIndex) Event`, `(Event) WitnessPayload() (WitnessPayload, bool)`, `RegisterTestKind`, `NewTestEvent`, `TestEventKinds()`; GREEN `event.go`: `eventRegistry []eventRegistration{name string; descriptor EventDescriptor}`, `EventKinds() []EventKind`.
- [ ] 2.9 RED R-AEE-005 (S-AEE-014..016): external read via accessor, no type switch; mismatched accessor fails not panics.
- [ ] 2.10 GREEN `event.go`: `(Event) Sequence()`; witness accessor pattern proven through `export_test.go`.
- [ ] 2.11 RED `event_registry_test.go` R-AEE-006 (S-AEE-017..019): zero production kinds enumerated.
- [ ] 2.12 GREEN: confirm `eventKindFirst == eventKindEnd == 1`, no production kind constants exist.
- [ ] 2.13 REFACTOR: dedupe registry-table helpers against `content_part.go` precedent; `make test` for AI-14.1 slice; record red/green output.

## Phase 3: AI-14.2 — Per-stream sequence

- [ ] 3.1 RED `sequence_test.go` R-AEE-007 (S-AEE-020..022): 1-based contiguous stamping; no external sequence setter; no reachable reset.
- [ ] 3.2 GREEN `sequence.go`: `Sequence uint64`, `Stamper{last Sequence}`, `(*Stamper) Stamp(e Event) Event`.
- [ ] 3.3 RED R-AEE-008 (S-AEE-023..025): two streams concurrently under `-race`; `Stamper` holds no package-level/atomic/mutex state.
- [ ] 3.4 GREEN: confirm zero-value-ready `Stamper`, no shared state (structural + doc comment).
- [ ] 3.5 RED doc-assertion test R-AEE-009 (S-AEE-026..027): package doc states cross-stream comparison is permitted and meaningless.
- [ ] 3.6 GREEN `doc.go`: cross-stream rule paragraph.
- [ ] 3.7 RED `event_test.go` R-AEE-010 (S-AEE-028..030): unstamped sentinel `0` rejected `ErrOutOfRange` at the `CheckEmit` boundary; stamped seq `1` accepted.
- [ ] 3.8 GREEN `event.go`: `CheckEmit` rule 2 (`seq == 0` → `ErrOutOfRange` at `At("event"), At("sequence")`).
- [ ] 3.9 REFACTOR: extract shared `CheckEmit` rule-ordering helper; `make test` for AI-14.2 slice; record red/green output.

## Phase 4: AI-14.3 — No process-global sequence-state guard

- [ ] 4.1 RED `sequence_guard_test.go` R-AEE-011 (S-AEE-031..035): fails scratch `var scratchSeq uint64`, struct-wrapped counter, reset func; passes on landed package and on `_test.go` counters.
- [ ] 4.2 GREEN `sequence_guard_test.go`: D8 `go/parser` + `go/types` guard — Named→Underlying (`types.Unalias`), struct fields any depth, array elements; base match = integer basics + `uintptr` + `sync/atomic` types; no depth cap; bare type params fail-closed; reset detection via `types.Info.Uses` (assign/inc-dec/`Store`/`Swap`/`CompareAndSwap`, `Add` legal).
- [ ] 4.3 RED R-AEE-012 (S-AEE-036..039): allowlist has exactly one entry (`message.go`/`lastMessageID`); stale or empty-rationale entries fail.
- [ ] 4.4 GREEN: `allowlistEntry{file, identifier, rationale}`, `sequenceStateAllowlist` table naming C3 and the `V-REQ-03` (distinguishability) vs `V-STR-13` (contiguity) distinction.
- [ ] 4.5 RED+GREEN R-AEE-013 (S-AEE-041..042): guard source comment names C3, the retired process-global counter, and "not a smaller counter; the counter where the stream is"; failure message names file+identifier.
- [ ] 4.6 REFACTOR: guard failure-message formatting; confirm `message.go`'s `lastMessageID` diff is empty (S-AEE-040); `make test` for AI-14.3 slice; record red/green output.

## Phase 5: AI-14.4 — Ordering invariants

- [ ] 5.1 RED `event_registry_test.go` R-AEE-014 (S-AEE-043..045): every registered kind has a descriptor from stated domains; block-role kinds expose a readable block index.
- [ ] 5.2 GREEN: wire `EventDescriptor` into `eventRegistration`; `event.go` `CheckEmit` rule 3 (block index vs `Role`, `Role≠None` requires index ≥1) → `ErrOutOfRange` at `At("event"), At("block")`.
- [ ] 5.3 RED `stream_check_test.go` R-AEE-015 (S-AEE-046..048): checker reads only `(kind, descriptor, block index, sequence)`, no concrete payload type referenced.
- [ ] 5.4 GREEN `stream_check.go`: `CheckStream(events []Event) StreamReport`, `StreamReport{err error; terminated bool}`, `(StreamReport) Violation() error`, `(StreamReport) Terminated() bool`.
- [ ] 5.5 RED R-AEE-016 (S-AEE-049..053): block ordering start≺delta≺end per index; interleaving legal; unterminated block distinguishable from ordering violation.
- [ ] 5.6 GREEN: block-ordering pass in `CheckStream` per D7 verdict map (`ErrMisplaced` for out-of-order/after-terminal, `ErrMalformed` for unterminated), positions `AtIndex("event", seq)` + `AtIndex("block", idx)`.
- [ ] 5.7 RED R-AEE-017 (S-AEE-054..056): `at-most-one` cardinality via witness kind → `ErrDuplicate` naming kind + second event's sequence.
- [ ] 5.8 GREEN: cardinality pass in `CheckStream`.
- [ ] 5.9 RED R-AEE-018 (S-AEE-057..060): `terminal` primitive; event-after-terminal and second-terminal → `ErrDuplicate`/`ErrMisplaced`; "no terminal event" reported as distinct informational outcome via `Terminated() == false`.
- [ ] 5.10 GREEN: terminal pass in `CheckStream`; `StreamReport.Terminated()`.
- [ ] 5.11 RED R-AEE-019 (S-AEE-061..063): first violation in stream order; slice input never mutated; no channel; idempotent re-run.
- [ ] 5.12 GREEN: finalize `CheckStream` first-violation ordering and non-mutation contract.
- [ ] 5.13 RED R-AEE-020 (S-AEE-064..065): no exported accumulator/transcript-rebuilder/reducer of deltas; add-a-delta-kind doc states fragment-only, forbids snapshot.
- [ ] 5.14 GREEN `event_descriptor.go`: 6-step kind-adding doc comment (constant+move end, payload, constructor, accessor, registry line w/ descriptor, doc list).
- [ ] 5.15 REFACTOR: consolidate `CheckStream` verdict-mapping switch; `make test` for AI-14.4 slice; record red/green output.

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
