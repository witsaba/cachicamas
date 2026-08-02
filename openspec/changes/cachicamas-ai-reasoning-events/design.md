# Design: cachicamas-ai-reasoning-events (AI-17)

> **Predecessors**: [`proposal.md`](./proposal.md) · [`spec.md`](./specs/ai-reasoning-events/spec.md) · Engram `sdd/cachicamas-ai-reasoning-events/{proposal,spec}`
> **Read at design time**: AI-14 spec (`../cachicamas-ai-event-envelope/specs/ai-event-envelope/spec.md` — its `design.md` has NOT landed), AI-16 spec (`R-ATE-003/004`), `reasoning_content.go`, `validation.go`.

## Technical Approach

One new file, `backend/agent/src/ai/reasoning_event.go`, adds three payload types and their constructors, in AI-07's one-file-per-kind layout. Three entries are appended to AI-14's kind registry, kind table and descriptor table (exact file spelling gated on AI-14's design — see Open Questions). Kind is derived from payload (`R-AEE-001`); payloads are sealed (`R-AEE-003`); no accumulator ships (`R-ARE-008`). Reconstruction and state derivation are test-local, implemented by feeding the reconstructed `(redacted, text, token)` into AI-07's own doors (`NewReasoning` / `NewRedactedReasoning`) so S-ARE-033's "same derivation as `Reasoning.State()`" holds by construction, not by imitation.

## Architecture Decisions

### Decision: R-ARE-013 provisional sentence is KEPT — a redacted block rejects non-empty deltas

**Choice**: Keep the rejection reading. S-ARE-036 stands. Sentinel: `ErrMisplaced` at a position naming the fragment — "valid in itself, not permitted where it appears" is exactly a text fragment on a redacted block. A **zero-length** fragment on a redacted block is accepted (spec rejects only non-empty; empty fragments carry nothing).
**Alternatives considered**: carry-and-report (allow text, surface both signals to Layer 2).
**Rationale**: (1) AI-07 parity — `Reasoning` has no redacted+text shape; `NewRedactedReasoning(token []byte)` accepts no text, and `State()` makes redacted win. A wire shape the part model cannot represent has no "equivalent constructed part", making S-ARE-033 unsatisfiable and re-creating doc 0001 § 3.1's two-strategies defect one level up. (2) Redaction means plaintext *withheld* (V-REQ-10); text beside it is two contradictory signals, and passing both through violates the package's "no second copy two writers can disagree about" rule. (3) It is decidable at construction (below), matching AI-04's caller-contract posture.

### Decision: delta and end constructors take the typed `ReasoningBlockStart` payload, not a bare index

**Choice**: `NewReasoningDelta(start ReasoningBlockStart, fragment []byte)` and `NewReasoningBlockEnd(start ReasoningBlockStart, token []byte)`. The block index is inherited from the start; the redaction bit is visible at construction.
**Alternatives considered**: (a) AI-16-style independent `(blockIndex, fragment)` constructors plus a cross-event block validator; (b) a redacted bool parameter on the delta constructor.
**Rationale**: S-ARE-036 requires *construction-time* rejection "for that block", which only works if the constructor sees the block's redaction bit. (a) needs an exported cross-event validator — colliding with `R-ARE-008`'s no-reconstruction posture and duplicating AI-14's payload-independent checker territory; (b) is a second writable copy of the redaction bit. Inheriting the index also makes S-ARE-007 (same index on all four events) structurally guaranteed, and a zero-value start (index 0) makes delta/end construction fail `ErrOutOfRange` at the block-index field, satisfying S-ARE-008/009 for all three kinds. S-ARE-035 (redacted end without token → `ErrEmpty` at token, mirroring `Reasoning.validate` rule 3) also becomes a construction rule of the end. Recorded as a deliberate family divergence from AI-16 — redaction exists only in reasoning.

### Decision: redaction is a second constructor door, not a parameter

**Choice**: `NewReasoningBlockStart(blockIndex)` and `NewRedactedReasoningBlockStart(blockIndex)`.
**Alternatives considered**: `redacted bool` parameter.
**Rationale**: verbatim AI-07 precedent (`NewRedactedReasoning`) — the one non-derivable bit gets a door of its own, set at construction and by nothing else (D1).

### Decision: fragment and token stored as `string`; `MaxTextLen` fragment cap kept

**Choice**: payload fields are strings (bytes-in-string, per `text_content.go`/`reasoning_content.go`); accessors convert at the boundary, copying both ways. Keep `R-ARE-007`'s `MaxTextLen` cap (`ErrOutOfRange`).
**Rationale**: aliasing (S-ARE-026) and `==` comparability (S-ARE-029, event equality) — `reasoning_content.go`'s token-as-string rationale applies verbatim. The cap reuses the existing constant; no new bound (aligned with AI-16 `R-ATE-009`, reconcile if its design drops it).

## Interfaces / Contracts

```go
// Kinds appended to AI-14's eventRegistry (event.go); descriptors per R-AEE-014,
// using AI-14's landed event_descriptor.go enums verbatim:
// start {Role: BlockRoleStart, Cardinality: CardinalityAny, Terminal: false}
// delta {Role: BlockRoleDelta, Cardinality: CardinalityAny, Terminal: false}
// end   {Role: BlockRoleEnd,   Cardinality: CardinalityAny, Terminal: false}
EventKindReasoningBlockStart, EventKindReasoningDelta, EventKindReasoningBlockEnd

type ReasoningBlockStart struct{ /* blockIndex BlockIndex; redacted bool */ }
func (ReasoningBlockStart) BlockIndex() uint64      // exported accessor unwraps AI-14's BlockIndex
func (ReasoningBlockStart) Redacted() bool          // R-ARE-012: only here
func (ReasoningBlockStart) blockIndex() BlockIndex  // unexported — satisfies AI-14's blockPayload

type ReasoningDelta struct{ /* blockIndex BlockIndex; fragment string */ }
func (ReasoningDelta) BlockIndex() uint64
func (ReasoningDelta) Fragment() []byte             // raw bytes; may be invalid UTF-8
func (ReasoningDelta) blockIndex() BlockIndex       // unexported — satisfies AI-14's blockPayload

type ReasoningBlockEnd struct{ /* blockIndex BlockIndex; token string; hasToken bool */ }
func (ReasoningBlockEnd) BlockIndex() uint64
func (ReasoningBlockEnd) Token() ([]byte, bool)     // AI-07 presence contract, verbatim
func (ReasoningBlockEnd) blockIndex() BlockIndex    // unexported — satisfies AI-14's blockPayload

func NewReasoningBlockStart(blockIndex uint64) (Event, error)
func NewRedactedReasoningBlockStart(blockIndex uint64) (Event, error)
func NewReasoningDelta(start ReasoningBlockStart, fragment []byte) (Event, error)
func NewReasoningBlockEnd(start ReasoningBlockStart, token []byte) (Event, error)

func (Event) ReasoningBlockStart() (ReasoningBlockStart, bool)  // Part.Reasoning() shape
func (Event) ReasoningDelta() (ReasoningDelta, bool)
func (Event) ReasoningBlockEnd() (ReasoningBlockEnd, bool)
```

Validation order (V-FAIL-04, where before what, emptiness before bound):

| Constructor | Rules in order |
|---|---|
| both start doors | index 0 → `ErrOutOfRange` at block index |
| delta | index 0 → `ErrOutOfRange`; redacted start ∧ non-empty fragment → `ErrMisplaced` at fragment; fragment > `MaxTextLen` → `ErrOutOfRange` at fragment. No `ErrEmpty` (R-ARE-007) |
| end | index 0 → `ErrOutOfRange`; redacted start ∧ nil token → `ErrEmpty` at token (present zero-length token satisfies); token > `MaxReasoningTokenLen` → `ErrOutOfRange` at token |

`nil` token = absent, any non-nil (incl. length 0) = present — AI-07's rule verbatim. All payloads get `String`/`GoString` naming kind and block index only, never contents or lengths (NFR-AEE-D posture).

## Data Flow

    producer (AI-22+): Start door ─► delta(start, frag)* ─► end(start, token?)
                          │  AI-14 stamper assigns per-stream sequence at emission
                          ▼
    consumer: partition by BlockIndex ─► concat fragments (consumer-local)
                          └─► test-local: NewReasoning / NewRedactedReasoning ─► State()

## File Changes

| File | Action | Description |
|------|--------|-------------|
| `backend/agent/src/ai/reasoning_event.go` | Create | Three payloads (each implementing `blockPayload.blockIndex() BlockIndex`), four constructors (two start doors), accessors, validation |
| `backend/agent/src/ai/event.go` (AI-14, landed design) | Modify | Append 3 `EventKind` constants + 3 `eventRegistry` entries with descriptors |
| `backend/agent/src/ai/event_descriptor.go` (AI-14, landed design) | Read-only | Reuse landed `BlockRole`, `Cardinality`, `EventDescriptor`, `BlockIndex`, `blockPayload` — no edit needed |
| `backend/agent/src/ai/reasoning_event_test.go` | Create | S-ARE-001…043 tests; test-local concatenator + AI-07-door reconstructor |
| `reasoning_content.go`, `reasoning_content_test.go` | Untouched | NFR-ARE-A; `opaqueTokens()` reused via test helper access |

## Testing Strategy

| Layer | What to Test | Approach |
|-------|-------------|----------|
| Unit (RED first, in spec order) | Registry membership + exhaustiveness legs; family distinctness; index rules; fragment fidelity incl. rune split; token byte-exactness over `opaqueTokens()`; aliasing; redacted/token-only reconstruction; totality table; fmt redaction | External `ai_test` package; `make test` (`go test -race -v ./...`) in `backend/agent/` |
| Integration / E2E | N/A — pure contract package, no I/O | — |

## Threat Matrix

N/A — no routing, shell, subprocess, VCS/PR automation, executable-file classification, or process-integration boundary.

## Migration / Rollout

No migration. Purely additive; revert removes three kinds and one file (proposal rollback plan).

## Reconciliation against AI-14 and AI-16 (both design phases now landed; code not yet applied)

Performed at tasks time, against `../cachicamas-ai-event-envelope/design.md` (AI-14, authoritative) and `../cachicamas-ai-text-events/design.md` (AI-16). No code exists yet for `event.go`, `event_descriptor.go`, `sequence.go`, `stream_check.go`, or `text_events.go` in this worktree — `sdd-apply` has not run for AI-14/15/16. This reconciliation is therefore against the two designs' pinned "Exported Surface (binding)" / "Interfaces / Contracts" sections, which are binding on siblings per AI-14's design header.

1. **Block-index integer type — CONFIRMED, one addition required.** AI-14 exports `type BlockIndex uint64` (1-based, zero = unset/rejected) in `event_descriptor.go`, plus an unexported `blockPayload interface{ blockIndex() BlockIndex }` that AI-14's checker type-asserts against for shared-space enforcement (R-AEE-015). AI-16's landed design keeps its **public** constructor parameter and accessor as plain `uint64` (not the `BlockIndex` type), converting internally. This design already picked `uint64` for its own public accessors/constructors — matches AI-16, no public-surface change. What was missing: the three reasoning payloads did not declare the unexported `blockIndex() BlockIndex` method. **Added above** (Interfaces / Contracts) so AI-14's `CheckStream`/registry can see reasoning block indices in the shared per-stream space (R-ARE-004). Internal storage changes from a bare `blockIndex uint64` field to `blockIndex BlockIndex`, converted at the `BlockIndex() uint64` accessor boundary — no public signature change.
2. **Registry/descriptor file spelling — CONFIRMED.** AI-14's landed File Changes table names `event.go` (EventKind, eventRegistry, Event, CheckEmit) and `event_descriptor.go` (BlockRole, Cardinality, EventDescriptor, BlockIndex, blockPayload) as two separate files. This design's File Changes table updated to modify `event.go` only (append constants + registry rows) and treat `event_descriptor.go` as read-only reuse.
3. **Constructor return shape — CONFIRMED.** Both AI-14 (witness constructors) and AI-16 (`NewTextBlockStart` etc.) return `(Event, error)`. This design's four constructors already match.
4. **Descriptor enum spelling — CONFIRMED.** AI-14 exports `BlockRole` (`BlockRoleNone`/`BlockRoleStart`/`BlockRoleDelta`/`BlockRoleEnd`) and `Cardinality` (`CardinalityAny`/`CardinalityAtMostOne`). Interfaces / Contracts block above updated from lower-case placeholder spelling (`blockRole: start`) to the exact landed identifiers.
5. **`R-ATE-009` cap decision — CONFIRMED KEPT.** AI-16's design resolved its own `[provisional]` marker to kept (`MaxTextLen` fragment cap ships). This design's `R-ARE-007` cap decision (reuse `MaxTextLen`, no new bound) already matches; no change needed.
6. **Constructor parameter shape divergence — CONFIRMED, deliberate.** AI-16 uses independent `(block uint64, ...)` constructors per kind; this design's delta/end constructors take the typed `ReasoningBlockStart` payload instead (documented above as a deliberate family divergence, needed for S-ARE-036's construction-time redaction check). No conflict: AI-14's registry/checker is payload-shape-agnostic (`blockPayload` interface only), so both families can coexist.

No further reconciliation blockers. `sdd-apply` may proceed once AI-14/AI-15/AI-16 code lands (per this milestone's stated dependency on AI-07 + AI-15, parallel with AI-16).
