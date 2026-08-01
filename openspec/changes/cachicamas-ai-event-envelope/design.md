# Design: Event envelope with per-stream sequencing (AI-14)

> **Change**: `cachicamas-ai-event-envelope` · **Spec**: [`specs/ai-event-envelope/spec.md`](specs/ai-event-envelope/spec.md) (`R-AEE-001..020`) · **Module**: `backend/agent`, `package ai`
> Resolves the spec's five open items. Exported names below are **binding on AI-15…AI-20** — sibling designs cite them.

## Technical Approach

Mirror `content_part.go` exactly: struct envelope with one unexported payload field, kind derived by method, sealed unexported payload interface, slice-indexed registration table, redacting `String`/`GoString`. Sequencing is a per-stream value type with zero shared state. Invariants are a pure function over a recorded slice, driven only by descriptors. Everything reports through AI-04's landed `Invalid`/`FirstFailure`/sentinels — no new failure machinery.

## Exported Surface (binding)

```go
type EventKind uint8                    // zero = non-member; String(): name | "unset" | "eventkind(N)"
func EventKinds() []EventKind           // declaration order; EMPTY at AI-14 (eventKindFirst = eventKindEnd = 1)
type Sequence uint64                    // V-STR-13; zero value IS the unstamped sentinel
type BlockIndex uint64                  // V-STR-15; 1-based, zero = unset/rejected (matches AI-16 R-ATE-003)
type Event struct{ payload eventPayload; seq Sequence }
func (e Event) Kind() EventKind         // derived; nil payload → 0
func (e Event) Sequence() Sequence
func (e Event) String() string          // "event(<kind> seq=N)" — never payload bytes or lengths (NFR-AEE-D)
func (e Event) GoString() string
type Stamper struct{ last Sequence }    // per-stream; zero value ready; NO reset method
func (s *Stamper) Stamp(e Event) Event  // returns copy with seq = s.last+1; total, never errors
func CheckEmit(e Event) error           // THE producer emission boundary (R-AEE-010)
type BlockRole uint8                    // BlockRoleNone(=0, zero default) | BlockRoleStart | BlockRoleDelta | BlockRoleEnd
type Cardinality uint8                  // CardinalityAny(=0) | CardinalityAtMostOne
type EventDescriptor struct{ Role BlockRole; Cardinality Cardinality; Terminal bool }
func CheckStream(events []Event) StreamReport   // the AI-14.4 checker; nil/empty safe
type StreamReport struct{ err error; terminated bool }
func (r StreamReport) Violation() error // first violation in stream order, or nil (AI-04 value)
func (r StreamReport) Terminated() bool // true iff exactly-one-terminal path saw a terminal event
```

Internal: `eventPayload interface { kind() EventKind; validate(at Path) *Violation }` (sealed); `blockPayload interface { blockIndex() BlockIndex }` (optional second unexported interface — block kinds implement it; the checker asserts to this *shared unexported* interface, never to a concrete payload, satisfying R-AEE-015); registry `var eventRegistry []eventRegistration` where `eventRegistration struct{ name string; descriptor EventDescriptor }` — one table, so a kind structurally cannot register without a descriptor (R-AEE-014).

## Architecture Decisions

| # | Decision | Choice | Rejected | Rationale |
|---|----------|--------|----------|-----------|
| D3 | Terminal presence (open item 4) | `StreamReport` with `Violation()` + independent `Terminated() bool`; absence is informational, never an error | Sentinel error for absence; tri-state enum | AI-15/AI-19 promote presence by asserting `Terminated()` in their own tests — zero AI-14 edits; error-shape would force them to un-wrap a non-violation |
| D4 | Emission boundary (open item 3) | **Exported** `CheckEmit(e Event) error`, separate from `Stamp` | Combined `(*Stamper).Emit`; unexported check bridged via `export_test.go` | S-AEE-028/030 offer *already-stamped* events at the boundary, so stamping and checking are distinct acts; exported because AI-15…19 tests and AI-20's harness call it from outside-test packages |
| D5 | `CheckEmit` rule order (V-FAIL-04) | 1 payload present+registered → `ErrNotInVocabulary` at `At("event")`; 2 `seq == 0` → `ErrOutOfRange` at `At("event"), At("sequence")`; 3 block index vs `Role` (`Role≠None` requires index ≥ 1) → `ErrOutOfRange` at `At("event"), At("block")`; 4 `payload.validate` | Single combined error | Mirrors `Part.validate` ordering; positions render the offending field per NFR-AEE-C |
| D6 | Witness bridging (open item, R-AEE-004) | `export_test.go` (package `ai`, test-only): `const KindTestWitness EventKind = eventKindEnd`; `WitnessPayload` implementing both interfaces; `NewWitnessEvent(block BlockIndex) Event`; accessor `func (e Event) WitnessPayload() (WitnessPayload, bool)`; `RegisterTestKind(name string, d EventDescriptor) (EventKind, func())` (append + truncating cleanup); `NewTestEvent(k EventKind, block BlockIndex) Event`; `TestEventKinds()` = production ∪ test-registered | `//go:build test` tags; a real placeholder kind | `_test.go` files in package `ai` never reach a non-test build (S-AEE-013/017 by construction); dynamic `RegisterTestKind` gives S-AEE-047/054/057 arbitrary descriptors. Tests holding a registration must not use `t.Parallel` |
| D7 | Checker verdict → sentinel map | delta/end before start → `ErrMisplaced`; second start, second at-most-one event, second terminal → `ErrDuplicate`; event after terminal → `ErrMisplaced`; unterminated block → `ErrMalformed` (distinguishable from ordering per S-AEE-052). Positions: `AtIndex("event", int(seq))` (+ `AtIndex("block", int(idx))` when block-scoped) | New sentinels | AI-04's set is closed (R-AIE-003); `errors.Is` on distinct landed sentinels gives distinguishability. Checker does NOT verify 1…N contiguity — gap detection is AI-22.3 |
| D8 | Guard mechanism (open item 2) | In-process `go/parser` over non-`_test.go` files of the package dir + `go/types` check (source importer); qualification recurses **Named→Underlying (unaliased), struct fields at any depth, array elements**; base match = integer basics + `uintptr` + any named `sync/atomic` type. **No depth cap**: Go forbids by-value type cycles, so value-structure recursion terminates structurally. Aliases resolved via `types.Unalias`. Bare type parameters qualify (fail-closed; unreachable at package scope). **Slices/maps/pointers/chans/funcs are NOT recursed** | `os/exec go vet`-style subprocess; covering slices | Spec's floor omits slices, and covering them would flag `eventRegistry []eventRegistration` (descriptor holds `uint8` enums) — forcing a second allowlist entry that R-AEE-012's "exactly one" forbids. Reset detection: via `types.Info.Uses`, fail any package-level func that assigns/inc-decs a qualifying package-level var or calls `Store`/`Swap`/`CompareAndSwap` on one; `Add` stays legal (mint, not reset — `mintMessageID` passes) |
| D9 | Allowlist shape (AI-14.3) | `type allowlistEntry struct{ file, identifier, rationale string }`; `var sequenceStateAllowlist = []allowlistEntry{{ "message.go", "lastMessageID", "<names C3 + V-REQ-03 distinguishability vs V-STR-13 contiguity>" }}` declared in the guard test file | Comment-directive allowlist (`//ai:allow`) | Table-in-guard keeps addition a reviewable diff of the guard itself; staleness = no package-level var of that name in that file → fail; empty rationale → fail (S-AEE-036..039) |
| D10 | Checker shape | Recorded-slice only (proposal Q3 resolved) | Incremental feeder + wrapper | R-AEE-019 fixes slice-in/no-channel/no-mutation; incremental form is AI-22.3's charter |

## Data Flow

    NewXxxEvent (AI-15+/witness) → Event{seq:0} → (*Stamper).Stamp → Event{seq:n}
        → CheckEmit (emission boundary: vocabulary → sentinel → block index → payload rules)
        → recorded []Event → CheckStream → StreamReport{Violation(), Terminated()}

## File Changes

| File (`backend/agent/src/ai/`) | Action | Content |
|---|---|---|
| `event.go` | Create | `EventKind`, bounds, `eventRegistry`, `EventKinds`, `Event`, `eventPayload`, `CheckEmit`, redacting renders |
| `event_descriptor.go` | Create | `BlockRole`, `Cardinality`, `EventDescriptor`, `BlockIndex`, `blockPayload`, append-a-line kind-adding doc (6 steps: constant+move end, payload, constructor, accessor, registry line w/ descriptor, doc list) |
| `sequence.go` | Create | `Sequence`, `Stamper`; doc states sentinel 0, the `CheckEmit` boundary, and cross-stream meaninglessness (R-AEE-009) |
| `stream_check.go` | Create | `CheckStream`, `StreamReport`, C3 note |
| `export_test.go` | Create | D6 surface |
| `sequence_guard_test.go` | Create | D8 guard + D9 allowlist + C3 source comment (R-AEE-013) |
| `event_test.go`, `event_registry_test.go`, `sequence_test.go`, `stream_check_test.go` | Create | Scenario tests (`package ai_test`) |
| `doc.go` | Modify | One paragraph: events exist; pointer to sequence.go's cross-stream rule |

## Testing Strategy

| Layer | What | Approach |
|---|---|---|
| Unit (ai_test) | S-AEE-001..030, 043..065, 067, 069 | Table tests via export_test bridges; concurrency (S-AEE-023) with two goroutines under `-race` |
| Guard | S-AEE-031..042 | Guard runs green; scratch subjects added → red recorded in tasks.md → dropped (NFR-AEE-E) |
| Surface | S-AEE-017/019/064 | AST scan asserting no production `EventKind` constants/accumulator identifiers |
| Module | S-AEE-066, 070 | Existing import guards + `make test`/`make lint` from `backend/agent/` |

Strict TDD: every task red→green→refactor in spec order, outputs recorded in `tasks.md`.

## Threat Matrix

N/A — no routing, shell, subprocess, VCS/PR automation, executable-file classification, or process-integration boundary. The D8 guard is in-process stdlib parsing (unlike `import_boundary_test.go`'s existing `go list` subprocess, which is untouched).

## Migration / Rollout

No migration. Purely additive; rollback = revert PR (nothing imports the envelope until AI-15).

## Open Questions

None blocking. Two notes for siblings: (1) AI-15/AI-19 assert terminal presence via `StreamReport.Terminated()` — never by editing `CheckStream`; (2) proposal Q4 resolved: `(kind, Role, BlockIndex)` suffices until AI-16 — no per-block-kind identity in the descriptor.

> **Deviation note**: exceeds the 800-word design budget for the same reason the spec recorded — five sibling designs bind to these exact symbol names, and the house precedent (AI-10/AI-12 designs) carries decision tables at this density.
