# Design — indexed text block events

> **Change**: `cachicamas-ai-text-events` · **Milestone**: AI-16 (Wave 2 "Stream")
> **Predecessor**: [`proposal.md`](./proposal.md) · [`specs/ai-text-events/spec.md`](./specs/ai-text-events/spec.md) · Engram `sdd/cachicamas-ai-text-events/proposal` (#2348)
> **Status**: designed against AI-14/AI-15 surfaces that had **not landed** at design time. **Reconciled 2026-08-01 at sdd-tasks** against AI-14's now-landed `design.md` (`cachicamas-ai-event-envelope`) — see "Reconciliation register" below for the resolved outcome of every A-row. AI-14/AI-15 code itself has **not been applied yet** in this worktree (no `event.go`, `response_start.go`, or `completion.go` exist) — AI-16's tasks are hard-gated behind both landing first.

## Technical Approach

One new family file in `backend/agent/src/ai` defines three payload types mirroring the AI-06/AI-09 house style exactly: exported payload struct with unexported fields, unexported `kind()`/`validate(at Path) *Violation`, one `New…` constructor per kind returning `(Event, error)`, one typed accessor per kind on `Event`, getter methods per field. Each payload additionally implements AI-14's unexported `blockIndex() BlockIndex` method, satisfying its `blockPayload` interface so `CheckStream`/`CheckEmit` can read the block index without a type switch. The three kinds are registered in AI-14's `eventRegistry` (in `event.go`) with descriptors `EventDescriptor{Role: BlockRoleStart|BlockRoleDelta|BlockRoleEnd, Cardinality: CardinalityAny, Terminal: false}` — `BlockRoleDelta` already exists in AI-14's landed `BlockRole` enum, anticipating this milestone. AI-14's registry-exhaustiveness coverage is extended with the three new kinds. All rejection flows through AI-04's `Violation` with landed sentinels only.

## Architecture Decisions

### Decision: R-ATE-009 is KEPT — the `MaxTextLen` fragment cap ships

**Choice**: `NewTextDelta` rejects a fragment longer than `MaxTextLen` bytes with `ErrOutOfRange` at `delta`. `S-ATE-022`/`S-ATE-023` stand as written; the `[provisional]` marker resolves to **kept**.
**Alternatives considered**: (a) no bound — rejected: the delta would be the only unbounded caller-supplied byte carrier in Layer 1, and `MaxTextLen`'s own GoDoc rationale ("make an unbounded value decidable from the request alone", V-REQ-24 reasoning) applies verbatim to a fragment; (b) a new smaller per-fragment constant — rejected: proposal decision 6 forbids inventing a bound, and any smaller cap could reject a fragment a legal complete text can contain.
**Rationale**: the cap is unobservable to a correct producer (a fragment of a legal text is never longer than the text, which `NewText` bounds at `MaxTextLen`), costs one `len()` comparison, and keeps a stream→`Part` materialization path from emitting fragments the `NewText` boundary would later reject. **Recorded asymmetry**: only the single fragment is bounded; the block's reconstructed total is not, because bounding it requires accumulation, which R-ATE-011 forbids at Layer 1. Enforcement of a total, if ever wanted, belongs to AI-23.

### Decision: exported payload structs, sealed by unexported fields

**Choice**: `TextBlockStart`, `TextDelta`, `TextBlockEnd` are exported structs with unexported fields, implementing AI-14's sealed payload interface — the `ToolCall`/`Reasoning` precedent (multi-field kinds export the payload type; only single-datum `Text()` returns the bare value).
**Alternatives**: unexported payloads with multi-return accessors (`(uint64, string, bool)`) — rejected: no house precedent, and getters cannot be documented per field.
**Rationale**: `tool_call.go:45` and `reasoning_content.go:191` are the binding precedent for kinds carrying more than one datum.

### Decision: block index is AI-14's `BlockIndex` type, stamped as a constructor argument (A4 — RESOLVED)

**Choice**: every constructor takes `block BlockIndex` first, using AI-14's landed `type BlockIndex uint64` (1-based, zero = unset/rejected — AI-14's own GoDoc cites AI-16's `R-ATE-003` verbatim). `validate` rejects `block == 0` with `ErrOutOfRange` at `block` before any other rule (V-FAIL-04 order). The producer stamps it; nothing derives it. Each payload additionally implements unexported `blockIndex() BlockIndex`, satisfying AI-14's `blockPayload` interface.
**Alternatives**: signed `int` (rejected: admits negatives needing a second rule); raw `uint64` (superseded — AI-14's landed design defines a dedicated `BlockIndex` type, resolving the original A4 "deferred" branch in favor of adopting it directly, matching every other block-scoped kind).
**Rationale**: proposal decisions 2–3; matches AI-14.2's per-stream sequence posture; adopting the landed type avoids an unexported-conversion seam between this payload and AI-14's checker. 0 as the rejected never-stamped sentinel satisfies S-ATE-006/-008. AI-14's `CheckEmit` (D5 item 3) independently re-checks `block >= 1` for any `Role != BlockRoleNone` at the emission boundary — this is a second, redundant check at a different boundary (emission vs. construction), not a substitute for construction-time validation, which alone satisfies R-ATE-003's "rejected at construction" wording.

### Decision: one family file, not three kind files

**Choice**: all three payloads live in `text_events.go`; tests in `text_events_test.go` (`package ai_test`).
**Alternatives**: one file per kind (the AI-06 header's "one file for one kind") — deviated from deliberately.
**Rationale**: the three kinds are one lifecycle sharing one block-index rule and one family GoDoc; the spec's scenarios (interleave, reconstruct, zero-delta) are cross-kind. If AI-15 lands a per-kind layout first, apply mirrors AI-15 (A5) — the first Wave 2 registrant sets the layout precedent.

### Decision: the fragment is a `string` of raw bytes

**Choice**: field `delta string`; getter `Delta() string`; GoDoc uses the word **byte** and states a single fragment may be invalid UTF-8 on its own (S-ATE-018). No emptiness rule, no `textPayload.validate` reuse, no trimming, no re-encoding.
**Alternatives**: `[]byte` — rejected: needs copy-in/copy-out to preserve payload immutability (proposal approach 3); Go strings carry arbitrary bytes.
**Rationale**: `text_content.go`'s documented posture ("bytes that are not well-formed UTF-8 … reads back exactly"), extended to fragments; R-ATE-007/-008.

## Interfaces / Contracts

```go
// Kinds appended to AI-14's EventKind block (event.go); registered names
// follow the tool_call snake style: "text_block_start", "text_delta",
// "text_block_end". block uses AI-14's landed BlockIndex type (event_descriptor.go).
EventKindTextBlockStart, EventKindTextDelta, EventKindTextBlockEnd

type TextBlockStart struct{ block BlockIndex }
func NewTextBlockStart(block BlockIndex) (Event, error)
func (e Event) TextBlockStart() (TextBlockStart, bool)
func (s TextBlockStart) Block() BlockIndex
func (s TextBlockStart) blockIndex() BlockIndex // unexported; satisfies AI-14's blockPayload

type TextDelta struct {
    block BlockIndex
    delta string // raw bytes; may be invalid UTF-8 alone; never validated as text
}
func NewTextDelta(block BlockIndex, delta string) (Event, error)
func (e Event) TextDelta() (TextDelta, bool)
func (d TextDelta) Block() BlockIndex
func (d TextDelta) Delta() string
func (d TextDelta) blockIndex() BlockIndex

type TextBlockEnd struct{ block BlockIndex }
func NewTextBlockEnd(block BlockIndex) (Event, error)
func (e Event) TextBlockEnd() (TextBlockEnd, bool)
func (b TextBlockEnd) Block() BlockIndex
func (b TextBlockEnd) blockIndex() BlockIndex
```

Validation order (V-FAIL-04): start/end — `block >= 1` else `ErrOutOfRange` at `block`. Delta — `block >= 1` at `block`, then `len(delta) <= MaxTextLen` else `ErrOutOfRange` at `delta`. Nothing else; zero-length and whitespace-only fragments pass. Accessors on a zero `Event` return the zero payload and `false`, never panic (NFR-ATE-B). `blockIndex()` is a pure accessor over the stored field — never independently validated, so it can never diverge from `Block()`.

## Data Flow

    producer (AI-28 adapter, future)
      │ stamps block index (arg)          AI-14 envelope stamps sequence
      ▼                                    ▼
    NewTextBlockStart/Delta/End ──► Event ──► stream ──► consumer partitions
                                                          by Block() alone;
                                                          test-local concatenator
                                                          proves byte fidelity

Two stamps, two owners: the block index at payload construction (this change), the per-stream sequence at emission (AI-14). Zero-delta legality is emergent — no rule requires a delta between start and end.

## File Changes

| File | Action | Description |
|------|--------|-------------|
| `backend/agent/src/ai/text_events.go` | Create | Three payloads, constructors, accessors, getters, `blockIndex()` |
| `backend/agent/src/ai/text_events_test.go` | Create | `ai_test` pkg: lifecycle, interleave, rune-split, zero-delta, totality table, test-local concatenator |
| `backend/agent/src/ai/event.go` (A2 — RESOLVED, AI-14's landed file) | Modify | Three `EventKind` constants (extend `eventKindEnd` bound), three `eventRegistry` rows with descriptors |
| `backend/agent/src/ai/event_registry_test.go` (A2 — RESOLVED, AI-14's landed exhaustiveness/registry test file) | Modify | Three coverage rows; scratch-kind bite proof re-run |
| `backend/agent/src/ai/doc.go` | Possibly modify | Package comment mention |

## Reconciliation register (resolved against AI-14's landed `design.md`, 2026-08-01)

| ID | Assumption | Source | Resolution |
|----|------------|--------|------------|
| A1 | `Event`, `EventKind`, sealed payload iface with `kind()`/`validate(at Path) *Violation` | AI-14 proposal "mirrors content_part.go" | **CONFIRMED** — AI-14's landed `Event struct{ payload eventPayload; seq Sequence }` and `eventPayload interface { kind() EventKind; validate(at Path) *Violation }` match verbatim. No design change. |
| A2 | Registry table + guard file names and registration mechanics | AI-14 proposal (files unnamed) | **CONFIRMED, names updated** — registry lives in `event.go` (`eventRegistry []eventRegistration`), coverage/exhaustiveness assertions live in `event_registry_test.go`. File Changes table updated above. |
| A3 | Descriptor spelling `{blockRole: none\|start\|delta\|end, cardinality, terminal}` | AI-14 proposal D1/D2 | **CONFIRMED** — landed as `EventDescriptor{ Role BlockRole; Cardinality Cardinality; Terminal bool }`, with `BlockRoleStart`/`BlockRoleDelta`/`BlockRoleEnd` already defined in AI-14's `BlockRole` enum (delta anticipated). |
| A4 | Index type `uint64`; whether AI-14.4's checker reads block index via an unexported payload method | AI-14 D1 "(kind, block index, descriptor)" | **RESOLVED — superseded**: AI-14 landed a dedicated `type BlockIndex uint64` plus an unexported `blockPayload interface { blockIndex() BlockIndex }` that `CheckStream`/`CheckEmit` assert against generically (never a concrete payload type, per AI-14's R-AEE-015). Design updated above to use `BlockIndex` throughout and to add `blockIndex()` on all three payloads. |
| A5 | Family-file layout | this design | **Explicit consistency call, kept as-is**: AI-15's design also chose one-file-per-kind (`response_start.go`, `completion.go`), but neither AI-14 nor AI-15 code has landed in this worktree yet — no actual precedent file exists to mirror. AI-16 deliberately **keeps its single `text_events.go` layout** (already-landed design decision, re-litigating file layout is out of scope for tasks) rather than pre-emptively splitting into three files. Recorded as a known one-file-per-kind house-style divergence, not an error. |

## Testing Strategy

| Layer | What to Test | Approach |
|-------|-------------|----------|
| Unit (RED first, strict TDD) | S-ATE-001…032 per leaf order AI-16.1 → 16.2 → 16.3 | `ai_test` external package; table-driven; `errors.Is` + `Path` rendering asserts |
| Guard | Registry exhaustiveness bites | Scratch kind in AI-14's guard test |
| Race | Whole package | `make test` (`go test -race -v ./...`) |
| Surface | R-ATE-011, S-ATE-011/-013/-027 | Export-enumeration assertions in test |

## Threat Matrix

N/A — no routing, shell, subprocess, VCS/PR automation, executable-file classification, or process-integration boundary.

## Migration / Rollout

No migration. Rollback per proposal: revert commit range; only surviving edits are AI-14's three registry rows.

## Open Questions

- [x] A1–A5 — resolved 2026-08-01 at `sdd-tasks` against AI-14's landed `design.md`; see reconciliation register. None blocks tasks planning.
- [ ] Proposal question 3 (test-only accumulator in `agenttest`) — design holds decision 7: strictly test-local; AI-17/AI-18 copy the ~10-line concatenator rather than coupling to a shared helper.
- [ ] **Blocking precondition, not a design open item**: AI-14 and AI-15 code have not been applied in this worktree yet (no `event.go`, `response_start.go`, `completion.go`). AI-16's `sdd-apply` MUST NOT start until both land — see `tasks.md` Phase 0.

> **Deviation note**: exceeds the 800-word design budget; the landed design precedent in this repository (archived Wave 1 designs) carries pinned signatures and assumption registers at this density; house convention wins.
