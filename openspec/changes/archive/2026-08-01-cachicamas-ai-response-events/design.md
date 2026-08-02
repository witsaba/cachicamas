# Design — response lifecycle events

> **Change**: `cachicamas-ai-response-events` · **Milestone**: AI-15 (Wave 2 "Stream")
> **Predecessors**: [`proposal.md`](./proposal.md) · [`specs/ai-response-events/spec.md`](./specs/ai-response-events/spec.md) (`R-ARP-001` … `R-ARP-011`, `S-ARP-001` … `S-ARP-041`) · Engram `sdd/cachicamas-ai-response-events/proposal` (obs #2162)
> **Depends on**: AI-13 (merged: `finish_reason.go`, `usage.go`, `validation.go`) · AI-14 (spec landed; design/code pending — see gate below)

## Technical Approach

Two new one-file-per-kind payloads in `backend/agent/src/ai`, in the exact layout `tool_call.go` established: exported opaque value type with unexported fields, package-owned constructor as the only door in, typed accessor on the event, redacting `String`/`GoString`. Both kinds land by **appending** to AI-14's kind constant block, registration table, and descriptor table — never by editing AI-14's checker.

**Blocking open item 1 is resolved by AI-14's landed spec, not by this design.** `R-AEE-017` (`at-most-one`) and `R-AEE-018` (`terminal`) ship both invariants as descriptor-driven primitives proven with witness kinds, and `R-AEE-015` forbids the checker from naming any kind. AI-15 therefore obtains `R-ARP-005` and `R-ARP-009` **by registration alone**; no checker generalization is needed and no delta on `ai-event-envelope` is raised.

## AI-14 Reconciliation Gate — RESOLVED at tasks time

AI-14's `design.md` has now landed (`openspec/changes/cachicamas-ai-event-envelope/design.md`). Every symbol below previously tagged **(AI-14)** as provisional has been diffed against that landed design:

- `Event`, `EventKind`, `Event.Kind()`, `Event.Sequence()` — match verbatim, no rename.
- The "registration table, and descriptor table" wording below is corrected: AI-14 ships **one** combined table, `var eventRegistry []eventRegistration` (`eventRegistration struct{ name string; descriptor EventDescriptor }`), living in `event.go`. Both new kinds append one line each to that single table, embedding an inline `EventDescriptor{Role, Cardinality, Terminal}` literal. `event_descriptor.go`'s types (`BlockRole`, `Cardinality`, `EventDescriptor`, `BlockIndex`) are consumed unchanged, not modified.
- The checker entry point is `CheckStream(events []Event) StreamReport`, not a generic "AI-14 checker" — the Data Flow diagram below is corrected to name it and to include the previously-omitted `CheckEmit` step.
- The emission path additionally passes through the exported `CheckEmit(e Event) error` boundary (AI-14 D4/D5) between `(*Stamper).Stamp` and recording. This step was missing from the original Data Flow diagram and is now added.
- The "no terminal event" informational outcome (AI-14 spec open item 4, consumed by `S-ARP-032`) resolves to `StreamReport.Terminated() bool` returning `false` with `Violation() == nil` — informational, never an error, per AI-14's decision D3.

All **(AI-14)** name pins below are now FINAL. `sdd-apply` remains gated behind **AI-14's code** landing, and that gate is satisfied: `event.go`, `event_descriptor.go`, `sequence.go` and `stream_check.go` are merged, and every symbol pinned above was diffed against those landed files rather than against AI-14's design prose.

> **Corrected 2026-08-01 at Wave 2 archive.** ~~"which is satisfied (AI-14 archived, `openspec/specs/ai-event-envelope/` promoted per doc 0002 milestone tracking)"~~ — **that claim was false when written.** AI-14's OpenSpec change had *not* been archived and `openspec/specs/ai-event-envelope/` did not exist at AI-15's design or apply time. AI-15's own `apply-progress.md` deviation #2 records the discrepancy correctly and flagged it for a later archive pass. The accurate sequence is: AI-15's `sdd-apply` correctly proceeded once AI-14's **code** merged — which is the gate that actually blocks implementation — while AI-14's OpenSpec **promotion** happened later, in the Wave 2 archive pass that also archived this change. `openspec/specs/ai-event-envelope/spec.md` exists as of that pass. Nothing about the pinned symbols or the design changes; only the claim about archive state was wrong, and it is corrected here rather than frozen into the archive.

## Architecture Decisions

| # | Decision | Alternatives rejected | Rationale |
|---|---|---|---|
| D1 | Two independent payload types with derived kinds | One payload with a start/end discriminant | `R-ARP-001`/`R-ARP-006`; keeps AI-14's C4 exhaustiveness table honest — one constructible payload per registered kind |
| D2 | Constructors return `(Event, error)`, unstamped (sequence 0 sentinel) | Return bare payload, wrap later | Mirrors `NewToolCall → (Part, error)`; stamping belongs to the producer boundary (`R-AEE-010`); zero `Event` on failure is unmistakable |
| D3 | Accessors `ResponseID()` / `ServedModel()` | `ID()` / `Model()` | `Model()` invites conflation with request-side `V-REQ-21` (`Request.Model()`); `ServedModel` names `V-STR-25` exactly (`S-ARP-009`); byte-exact plain strings per `ToolCall.ID()` precedent, never minted `MessageID`-style |
| D4 | Both kinds register `blockRole: none` | — | Neither event carries a block index; AI-16's 1-based block-index space (`R-ATE-003`) is untouched — spec open item 4 confirmed |
| D5 | Response-start: `cardinality: at-most-one`, `terminal: false`. Completion: `cardinality: at-most-one`, `terminal: true` | Completion as `cardinality: any` (terminal alone already implies ≤ 1 via `R-AEE-018`) | Explicit `at-most-one` states the rule where it is read; checker reports first violation only, so redundancy cannot double-report (`S-ARP-016`, `S-ARP-026`, `S-ARP-029`) |
| D6 | `FinishReason` and `Usage` held by value, embedded as-is | Pointers, re-encoding, parallel types | `R-ARP-007`/`R-ARP-008`; `TokenCount` value semantics carry the presence bit across the boundary untouched, so absence survives for free; keeps payloads comparable |

## Interfaces / Contracts

```go
// response_start.go — V-STR-19, V-STR-24, V-STR-25
type ResponseStart struct{ responseID, servedModel string }
func NewResponseStart(responseID, servedModel string) (Event, error)   // seq 0, unstamped
func (s ResponseStart) ResponseID() string   // byte-exact, never transformed
func (s ResponseStart) ServedModel() string  // byte-exact, never transformed
func (e Event) ResponseStart() (ResponseStart, bool)                   // accessor shape per Part.ToolCall()
// kind: EventKindResponseStart; eventRegistry line: {name: "responsestart", descriptor: EventDescriptor{Role: BlockRoleNone, Cardinality: CardinalityAtMostOne, Terminal: false}}

// completion.go — V-STR-20
type Completion struct { reason FinishReason; usage Usage }
func NewCompletion(reason FinishReason, usage Usage) (Event, error)
func (c Completion) FinishReason() FinishReason
func (c Completion) Usage() Usage
func (e Event) Completion() (Completion, bool)
// kind: EventKindCompletion; eventRegistry line: {name: "completion", descriptor: EventDescriptor{Role: BlockRoleNone, Cardinality: CardinalityAtMostOne, Terminal: true}}
```

Validation, first-failure order per `V-FAIL-04`: `ResponseStart` — `ErrEmpty` at `response_id`, then `ErrEmpty` at `served_model` (`S-ARP-010`…`013`). `Completion` — delegate to `FinishReason.Validate` at `finish_reason` (`ErrNotInVocabulary`), then `Usage.Validate` at `usage` (`ErrOutOfRange`); a fully absent `Usage` passes (`S-ARP-025`). Both payloads implement AI-14's sealed `eventPayload` interface: `kind() EventKind` and `validate(at Path) *Violation` (confirmed signature, landed). `String()`/`GoString()` return `"responsestart"` / `"completion"` — `V-FAIL-13` posture, no field, no length.

## Data Flow

    adapter ──NewResponseStart/NewCompletion──▶ Event{seq:0}
        │                                         │
        │                              (*Stamper).Stamp (per-stream)
        ▼                                         ▼
                                            Event{seq:n} ──▶ CheckEmit
                                                              (vocabulary → sentinel → block index → payload.validate)
                                                                  ▼
                                            recorded []Event ──▶ CheckStream ──▶ StreamReport{Violation(), Terminated()}
                                                              (verdict via descriptors only; no AI-15 kind named in its source)

## File Changes

| File | Action | Description |
|---|---|---|
| `backend/agent/src/ai/response_start.go` | Create | Payload, constructor, accessors, validation, redaction |
| `backend/agent/src/ai/response_start_test.go` | Create | External `ai_test` scenarios for AI-15.1 |
| `backend/agent/src/ai/completion.go` | Create | Payload, constructor, accessors, validation, redaction |
| `backend/agent/src/ai/completion_test.go` | Create | External `ai_test` scenarios for AI-15.2, incl. empty-response stream |
| `backend/agent/src/ai/event.go` | Modify | Append `EventKindResponseStart`, `EventKindCompletion` constants (move `eventKindEnd`); append two `eventRegistration` lines (name + inline `EventDescriptor` literal) to `eventRegistry`; append two entries to the kind-doc list (6-step recipe in `event_descriptor.go`) — no existing constant, table row or entry edited |
| `openspec/specs/ai-contract-vocabulary/spec.md` | Modify | Append `V-STR-24`, `V-STR-25`, dated § 4 blockquote, § 10 counts 116 → 118 (`R-ARP-011`) |

## Testing Strategy

| Layer | What | Approach |
|---|---|---|
| Unit (TDD, in order) | AI-15.1 item 1: fields, byte-exactness, `ErrEmpty`; item 2: `at-most-one` via checker. AI-15.2 item 1: embed unchanged + absence; item 2: terminal; item 3: empty response legal vs. no-terminal | Red → green → refactor per item, outputs recorded in `tasks.md`; external `ai_test` package; `make test` (`-race`) |
| Registry | Exhaustiveness table now covers two production kinds (`S-ARP-003`, `S-ARP-018`) | Extend AI-14's assertion by appending witnesses |
| Review-only | Register amendment diffs (`S-ARP-034`…`037`), no-branch-in-checker (`S-ARP-016`, `S-ARP-026`) | Reviewer/diff tasks, not Go tests |

## Threat Matrix

N/A — no routing, shell, subprocess, VCS/PR automation, executable-file classification, or process-integration boundary.

## Migration / Rollout

No migration. Purely additive; rollback is `git revert` of the commit range (proposal's plan holds — no checker generalization means an even smaller range).

## Open Questions

None blocking. Both prior open items are resolved by the AI-14 Reconciliation Gate above: symbol names confirmed final; `S-ARP-032`'s "no terminal event" case reads `StreamReport.Terminated() == false` with `Violation() == nil`.
