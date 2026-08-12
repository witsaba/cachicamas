# Design — `cachicamas-agent-event-envelope` (AG-04)

> Inputs: `proposal.md` (this change, including its four open decisions plus risk 5's concurrency question), `explore.md` § 8, charter `0003:410-517`, AG-01 `decision.md` § 5/§ 6/§ 10, AG-00 register rows `VL2-EVT-01..16`/`VL2-SEAM-14`, and the Layer 1 sources cited per decision below. All new tests are `package agent_test`; evidence gate is `make test` in `backend/agent/` (`go test -race -v ./...`). Doc 0003's authoring constraint assigns Go naming to this phase — every type named here is named deliberately, with no biological or cognitive metaphor: this is the portable agent runtime.

## Technical Approach

Four workpieces, one PR, four commits (one per node): (1) the envelope — identity, derived kind, emission gate, per-lane ordering (AG-04.1); (2) the run/turn lifecycle families and the production-exported stream-contract validator with a two-level scope engine (AG-04.2); (3) the two invariant pins — no delta route, typed failure as a thin wrap over `*ai.Failure` (AG-04.3); (4) the witness-table every-kind-constructible guard over exactly the four registered kinds, closed on a recorded bite (AG-04.4). One guarded row (`L2C-04`) is added to `doc.go` with its expectation-table entry in the same commit.

## Architecture Decisions

### AD-1 — The validator mirrors Layer 1's descriptor-driven engine, with the scope dimension added to the descriptor, not to the engine's knowledge of kinds (resolves open decision 1)

**Choice**: `agent.CheckStream(events []Event) StreamReport` — a production-exported function over a finite ordered slice, mirroring `ai.CheckStream`'s architecture: kind → registry → `EventDescriptor` → generic rule engine that never special-cases a kind by name and never reads a concrete payload type (`stream_check.go:45-49`, the R-AEE-015 posture). The Layer 2 descriptor carries the new dimension as data: `Bracket BracketRole` (none / opens-run / closes-run / opens-turn / closes-turn), `Placement` (run-scoped vs must-be-inside-an-open-turn), `Cardinality`, `Terminal bool`. The engine holds a fixed two-level scope state machine (run open → at most one turn open, because turns never overlap per `decision.md:352-356`) driven purely by descriptor fields plus envelope identity. Its per-event reads are: kind, descriptor, sequence, and the envelope's own identity fields (run, turn, parent) — identity is envelope data, not payload data, so the "no concrete payload type" rule survives the new dimension intact.

**Two deliberate divergences from `ai.CheckStream`, both evidence-forced**:
1. **Contiguity is checked here, not in a test kit.** Layer 1 explicitly excluded 1…N contiguity from `CheckStream` and gave it to `agenttest.CheckContiguity` (`stream_check.go:52-54`, `agenttest/doc.go:20-22`). AG-04.1 scenario 2 names the validator itself as what proves "independent, contiguous, 1-based" (`0003:445-448`), and AD-4 rules `agenttest` out — so `agent.CheckStream` checks per-stream contiguity (each event's sequence = previous + 1, starting at 1).
2. **A complete run bracket is required, not optional.** Layer 1 keeps `Terminated()` independent of violations because Layer 1 has a sanctioned bare-close loss path (`stream_check.go:14-19`). Layer 2 has no "sometimes no terminal at all" case at run scope (`decision.md:358`, `decision.md:541`): a sequence with no run-start, no run-end, an unclosed turn at run-end, or anything after run-end is rejected. An empty or nil slice is therefore rejected too — it carries no run bracket.

**Alternative rejected — a materially new, kind-aware nesting checker** (a hand-written state machine that switches on `EventKindTurnStart` etc. by name): it would satisfy AG-04's 11 scenarios in less code today, but AG-05 and AG-06 would then have to edit the checker's own source per family, which is exactly the corner risk 6 names; Layer 1's registry was extended by AI-15…AI-19 with zero checker changes precisely because the descriptor carries the rules (`event_descriptor.go:126-130`). `VL2-SEAM-14`/`VL2-EVT-16` make the checker AG-23's wholesale-reused kit surface (`AG-00 decision.md:177, :236`) — a kind-by-name checker would put every later family's ordering semantics inside a file AG-23 freezes.

**Alternative rejected — reusing `ai.CheckStream` itself**: impossible by type — it takes `[]ai.Event`, and `S-AGV-020` (spec.md:122) forbids presenting the wrap as a reuse. The mirror is of shape, not of code.

**Not pre-built, deliberately**: block/delta machinery (Layer 1's `BlockRole`/`BlockIndex`). AG-04 registers no delta kind (AD-3's pin depends on that absence), and `VL2-EVT-12` assigns indexed-delta semantics to AG-04.3 **+ AG-05.1**. AG-05 extends the descriptor with its message-block dimension under its own change — an amendment with its own justification, per the rollback plan's forward-fix preference — while its kinds join the run/turn/placement/cardinality/terminal rules already here with zero engine edits.

**Consequences**: AG-05/AG-06 add kinds by registry row + witness entry (AD-5's procedure); AG-06's at-most-one-style protocol kinds use `Cardinality` unchanged; AG-23 imports `agent.CheckStream` from the production package with no test-only build tag (proposal success criterion 5).

### AD-2 — The typed-failure payload is a thin wrap holding `*ai.Failure`, never a parallel vocabulary (resolves open decision 2)

**Choice**: an exported value type `agent.Failure` with one unexported field holding the wrapped `*ai.Failure`, one constructor `NewFailure(f *ai.Failure) (*Failure, error)` (nil rejected), and delegating accessors: `Category() ai.FailureCategory`, `Delivery() ai.DeliveryPath`, `Retryable() bool`, `Unwrap() error` (returns the wrapped `*ai.Failure`, keeping the whole `errors.Is` sentinel chain of `provider_failure.go:179-189` reachable through the envelope). It is carried by the run-end payload when `RunOutcome` is failed and by the turn-end payload when `TurnOutcome` is aborted-by-failure — AG-04 registers no separate error kind, because the register gives AG-04 exactly the two lifecycle families (`AG-00 decision.md:163-164`) and lists no Layer 2 error family at all; failures ride the typed outcomes.

**Why this satisfies what is settled**: `S-AGV-020` requires the wrap (the envelope-level payload is a new Layer 2 type); AG-04.3's "category and cause reachable as values" is met by `Category()`/`Unwrap()`; `VL2-EVT-15`'s pre-stream/mid-stream distinguishability is met by `Delivery()`, which is exactly the axis Layer 1 already carries (`provider_failure.go:226-247, :528-533`) — the explore's own reading ("structurally mirroring `Failure.Category()`/`Failure.Unwrap()`, not inventing a parallel vocabulary", explore.md § 2).

**Alternative rejected — an independent category/cause-compatible value type**: it would re-declare the closed nine-member category vocabulary (`provider_failure.go:53-103`) and its per-category sentinels, creating a second source of truth that silently misses any category Layer 1 appends (the vocabulary is append-only, R-AIP-005), would cost ~200 duplicated lines against a High sizing risk, and buys nothing `Delivery()` does not already give. **What would falsify AD-2**: an agent-origin failure at AG-11 that cannot be expressed as an `ai.Failure` (Layer 1's constructors are public — `PreStreamFailure`/`MidStreamFailure`, `provider_failure.go:610-624` — so Layer 2 can mint one; if AG-11's loop-level taxonomy outgrows that, AG-11 amends this type in its own change, never by parsing message strings).

**Consequence for AG-05**: the tool-execution family's "execution itself failed" outcome (`VL2-EVT-05`) reuses this same wrap type. For AG-23: the kit inspects failures through `Category()`/`Delivery()` values, satisfying "no code path assigns meaning to a message string".

### AD-3 — One new guarded row, `L2C-04`, carrying the membership criterion (resolves open decision 3)

**Choice**: yes — `doc.go` gains exactly one row, and `doc_contract_guard_test.go`'s `expectedLayer2ContractRows` gains its byte-identical entry in the same commit (the guard fails on count first, then per-row byte-exact diff — verified at `doc_contract_guard_test.go:105-123`):

```
//	L2C-04	Stream membership: a fact belongs on the event stream or it does not exist upward — if it is not on the stream, no frontend can render it and no log can reconstruct it. This is the criterion every later event family is judged by (doc 0003 AG-04 acceptance; agent-event-envelope).
```

**Rationale**: the charter's acceptance makes the membership criterion a documentary obligation ("stated in the package docs as the criterion later families are judged by", `0003:418`), and AG-05/AG-06 must be able to cite it when arguing a family in or out — that is precisely what the machine-checked row grammar exists for (`doc.go:12-14`). The three existing rows (L2C-01/02/03, `doc.go:16-18`) do not carry it: L2C-03 states the stream is the only upward contract, not the criterion for what belongs on it.

**The four ordering invariants are deliberately NOT guarded rows.** They are stated in `doc.go` prose (below the contract table) and pinned behaviorally by the validator's rejection scenarios — because AG-04 closes none of them alone (`0003:2203`: invariant 1 with AG-05.1, 2 with AG-19.1, 3 without AG-04 at all, 4 with AG-11.2), and freezing partial-invariant wording as a byte-exact contract row would either overclaim (risk 4) or need rewording in a later milestone's PR, churning the guard. The prose states the partial-closure map explicitly as a non-claim.

**Consequence**: `agent-package-scaffold` takes a spec delta (`S-AGP-012..015`'s committed-table pin now describes four rows) — the proposal's conditional Modified capability activates. `sdd-tasks` must sequence the row and the table entry in one commit.

### AD-4 — AG-04's tests do not import `src/agenttest` (resolves open decision 4)

**Choice**: no `agenttest` import anywhere in AG-04. Confirmed on three grounds, not inferred from absence alone:
1. **Type impossibility**: the kit is typed over Layer 1 — `RequireValidStream`/`CheckContiguity` delegate to `ai.CheckStream` over `[]ai.Event`, and `Provider` is an `ai.ModelProvider` (`agenttest/doc.go:13-23`). No helper in it can receive an `agent.Event`; there is nothing to use.
2. **Guard check, answered rather than assumed**: importing `agenttest` from `package agent_test` would actually PASS AG-03's guard — `allowedTestPrefixes` admits `…/src/agenttest` in the test closure (`import_boundary_test.go:137`, check 2), and the production closure never sees test files. So the guard does not decide this; the decision is on merit, and the merit is (1).
3. **Charter alignment**: dependency edges are AG-01 and AG-03 only (`0003:419`), and the invariants must be "assertable over hand-built sequences" (`0003:417`) — AG-04's tests construct events exclusively through the Layer 2 public surface.

**Recorded so it is not misread later**: `agenttest/doc.go:33-34` says "every Layer 2 agent-loop test is built on all three: the fake, the kit and the suite" — that binds Wave 2's loop tests, which drive a real producer, not this milestone's envelope tests. The Layer 2 analog of the kit's convenience wrappers is AG-23's, per `VL2-EVT-16`'s own reuse clause.

### AD-5 — The lane's forwarding activity stamps; the counter stays unsynchronized, and the safety argument is AG-01's lane ownership (resolves proposal risk 5 / decision 5)

**Choice**: `agent.Sequence` (uint64, 1-based, zero = never stamped) and `agent.LaneStamper` (`struct{ last Sequence }`, zero value ready, `Stamp(Event) Event`) — a distinct type, explicitly not `ai.Sequence` (`S-AGV-021`, spec.md:123), mirroring `ai.Stamper`'s shape (`sequence.go:50-60`). **The stamping actor is the lane's forwarding activity**: AG-01 § 5 decides that each attached consumer's lane is "fed by its own forwarding activity that privately owns that consumer's receive-only carrier" (`decision.md:230`) — exactly one goroutine writes a given lane, by the mechanism's own ownership rule, restated recursively at `decision.md:267`. One `LaneStamper` belongs to one lane and is touched only by that lane's forwarding activity, so two stampers never share memory — the same stronger-than-a-lock argument Layer 1 records at `sequence.go:9-18`, with the guarantee's source moved from AI-02 § 4 (one producer per provider stream) to AG-01 § 5 (one forwarding activity per lane). The name `LaneStamper` carries the ownership scope on purpose, so the safety argument is legible at every use site.

**What the design does if the topology is not single-producer**: nothing silent. No mutex and no atomic — synchronizing the counter would make a two-writer lane pass `-race` while still interleaving stamps, masking a topology defect (a violation of AG-01's decided mechanism, which "a downstream milestone that finds itself deciding a delivery property is proposing an amendment" forbids) as safe concurrency. The type's doc states the one-writer-per-lane precondition and cites AG-01 § 5; AG-04.1 scenario 2 runs two goroutines, each stamping its own hand-built stream through its own `LaneStamper`, under `-race` — the detector is the tripwire, exactly as `TestStamper_SequenceState_IsPerStreamNotProcess` is for Layer 1. **What would falsify AD-5**: AG-20.2/AG-21's pressure tests finding a lane with two writers — that is an amendment to AG-01's mechanism first, and only then to this counter.

**At AG-04 there is no producer**: tests play the forwarding-activity role, one stamper per hand-built stream. Wave 2's real distribution step inherits the precondition as written.

### AD-6 — Vocabulary reuse for verdicts: AI-04's violation values, no second failure framework

**Choice**: validator and constructor failures are reported through Layer 1's existing caller-contract vocabulary — `ai.Invalid`, `ai.ErrNotInVocabulary`, `ai.ErrOutOfRange`, `ai.ErrDuplicate`, `ai.ErrMisplaced`, `ai.ErrMalformed` — never a new sentinel set. This is `stream_check.go:26-28`'s own posture ("no second failure type, no new sentinel") applied across the layer boundary L2C-01 already authorizes imports over. The reuse-vs-wrap register governs domain identities (events, ordering, provider failure); the AI-04 violation vocabulary is the imported library's error-reporting surface, and duplicating it would add ~150 lines and a second `errors.Is` dialect for the same six rule classes. Rejected alternative — Layer 2 sentinels mirroring AI-04's — reconsidered only if a Layer 2 rule class arises that AI-04's six cannot name; none of the 11 scenarios needs one.

## Data Flow — the validator's scope engine

```
CheckStream(events []Event)
  state: seq_expect=1 · run=UNOPENED→OPEN→CLOSED · turn=NONE→OPEN(turnID)
  per event (kind → registry → descriptor; reads envelope identity only):
    1. seq != seq_expect            → violation (contiguity, 1-based)
    2. run == CLOSED                → violation (nothing follows the terminal)
    3. Bracket transition:
         opens-run   : run != UNOPENED → duplicate    | else run=OPEN
         closes-run  : turn == OPEN    → malformed    | else run=CLOSED
         opens-turn  : turn == OPEN    → overlap      | else turn=OPEN(id)
         closes-turn : turn != OPEN(id)→ misplaced    | else turn=NONE
         none        : (AG-05+ kinds)
    4. run != OPEN && !opens-run     → misplaced (everything inside the bracket)
    5. Placement == turn && turn != OPEN → misplaced   (AG-05's seam, unused now)
    6. Cardinality at-most-one repeated  → duplicate   (AG-06's seam, unused now)
  end: run != CLOSED → violation (incomplete run; empty slice rejected here)
```

Single-package change: no cross-service flow exists, so no sequence diagram beyond this state sketch (config.yaml `rules.design` diagram rule satisfied by explicit statement; no config file changes anywhere in this change, so the before/after-diff rule is vacuously met and said so here).

## File Changes

| File | Action | Node | Owns |
| --- | --- | --- | --- |
| `backend/agent/src/agent/event.go` | Create | AG-04.1 | `Event`, `EventKind` + 4 constants, registry table, `EventKinds()`, `Kind()` derivation, identity accessors, `CheckEmit` |
| `backend/agent/src/agent/event_descriptor.go` | Create | AG-04.1 | `EventDescriptor`, `BracketRole`, `Placement`, `Cardinality`; Layer 2's documented six-step adding-a-kind procedure |
| `backend/agent/src/agent/sequence.go` | Create | AG-04.1 | `Sequence`, `LaneStamper` (AD-5's precondition documented) |
| `backend/agent/src/agent/run_events.go` | Create | AG-04.2 | run-start/run-end payloads, `RunOutcome`, constructors + accessors |
| `backend/agent/src/agent/turn_events.go` | Create | AG-04.2 | turn-start/turn-end payloads, `TurnOutcome` |
| `backend/agent/src/agent/failure.go` | Create | AG-04.3 | `Failure` wrap (AD-2) |
| `backend/agent/src/agent/stream_check.go` | Create | AG-04.2 | `CheckStream`, `StreamReport`, scope engine |
| `backend/agent/src/agent/doc.go` | Modify | AG-04.3 | `L2C-04` row; ordering invariants + partial-closure non-claim in prose |
| `backend/agent/src/agent/doc_contract_guard_test.go` | Modify | AG-04.3 | fourth `expectedLayer2ContractRows` entry, same commit as the row |
| `backend/agent/src/agent/envelope_test.go` | Create | AG-04.1 | scenarios 1-3 (kind derivation/validation gate; per-lane `-race` ordering; parent-before-delegation) |
| `backend/agent/src/agent/stream_check_test.go` | Create | AG-04.2 | scenarios 4-6 (run bracket + outcome; turn nesting + outcome; nothing-after-terminal) |
| `backend/agent/src/agent/invariant_pin_test.go` | Create | AG-04.3 | scenarios 7-8 (no delta route; typed failure values) |
| `backend/agent/src/agent/event_registry_test.go` | Create | AG-04.4 | witness-table guard + bite (scenario 9-11 group) |
| `import_boundary_test.go`, `ambient_authority_test.go` | Unchanged | — | must stay green with zero logic edits (proposal criterion 13) |
| `go.mod`, `go.sum`, `Makefile`, `.golangci.yml` | Byte-unchanged | — | recorded diff |

## Interfaces / Contracts

Names chosen here under doc 0003's authoring grant; all package-qualified (`agent.`), mirroring Layer 1's short-name discipline — qualification is the disambiguator, `AgentEvent`-style stutter rejected.

```go
type EventKind uint8            // closed; zero not a member; derived, never stored (event.go:33-41 posture)
const (
    EventKindRunStart EventKind = iota + 1   // VL2-EVT-02
    EventKindRunEnd                          // VL2-EVT-02 + VL2-EVT-10; Terminal
    EventKindTurnStart                       // VL2-EVT-03
    EventKindTurnEnd                         // VL2-EVT-03 + VL2-EVT-11
)
func EventKinds() []EventKind   // enumerates the declared constant space, NOT the registry (event.go:235-241's stated reason)

type RunID string               // opaque run identity; delegation's parent identifier is a RunID (VL2-EVT-13)
type TurnID string
type RunOutcome uint8           // RunOutcomeCompleted / RunOutcomeInterrupted / RunOutcomeFailed; zero not a member
type TurnOutcome uint8          // TurnOutcomeFinished / TurnOutcomeAborted; zero not a member

type Event struct{ /* unexported: payload, seq, run, turn, parent */ }
func (e Event) Kind() EventKind          // derived from payload; nil payload → 0, not a member
func (e Event) Sequence() Sequence
func (e Event) Run() RunID
func (e Event) Turn() (TurnID, bool)     // false for run-scoped events
func (e Event) Parent() (RunID, bool)    // false for a top-level event — the field exists before delegation does (0003:450-453)

func NewRunStart(run RunID) (Event, error)
func NewDelegatedRunStart(run, parent RunID) (Event, error)   // the only door that sets parent at AG-04
func NewRunEnd(run RunID, outcome RunOutcome, f *Failure) (Event, error)   // f required iff RunOutcomeFailed, forbidden otherwise
func NewTurnStart(run RunID, turn TurnID) (Event, error)
func NewTurnEnd(run RunID, turn TurnID, outcome TurnOutcome, f *Failure) (Event, error)
func (e Event) RunStart() (RunStart, bool)    // + RunEnd/TurnStart/TurnEnd accessors, (T, bool), never panic

type Failure struct{ /* unexported: wraps *ai.Failure */ }    // AD-2
func NewFailure(f *ai.Failure) (*Failure, error)
func (f *Failure) Category() ai.FailureCategory
func (f *Failure) Delivery() ai.DeliveryPath                  // pre-stream vs mid-stream, VL2-EVT-15
func (f *Failure) Retryable() bool
func (f *Failure) Unwrap() error                              // the wrapped *ai.Failure; sentinel chain intact

type Sequence uint64                                          // NOT ai.Sequence (S-AGV-021); zero = never stamped
type LaneStamper struct{ /* last Sequence */ }                // one per consumer lane; written only by that lane's forwarding activity (AD-5)
func (s *LaneStamper) Stamp(e Event) Event

type EventDescriptor struct{ Bracket BracketRole; Placement Placement; Cardinality Cardinality; Terminal bool }
func CheckEmit(e Event) error                                 // registered kind → stamped → payload's own rules (CheckEmit's order, event.go:339)
func CheckStream(events []Event) StreamReport                 // AD-1; production-exported, VL2-EVT-16
func (r StreamReport) Violation() error                       // first violation in stream order; no independent Terminated() — see AD-1 divergence 2
```

**Adding a kind — Layer 2's six-step procedure** (documented in `event_descriptor.go`'s comment; AG-05/AG-06 follow it, mirroring `event_descriptor.go:12-32`):
1. declare the constant at the end of the `EventKind` block and move the end bound past it;
2. add the unexported payload type with `kind()` and `validate()`;
3. add the exported constructor `New<Kind>`;
4. add the exported accessor `func (e Event) <Kind>() (T, bool)`;
5. add the registry row pairing the kind's name with its `EventDescriptor` (one table — a kind structurally cannot register without a descriptor);
6. add the name to `EventKind`'s documented list AND the witness entry in `event_registry_test.go` — the guard's bidirectional cross-check fails the same PR that skips either half.

**The every-kind-constructible guard** (AG-04.4): `map[agent.EventKind]eventKindWitness` with two legs per kind — a no-argument constructor closure returning `(agent.Event, error)` and a payload-accessor closure `func(agent.Event) (any, bool)` — cross-checked bidirectionally against `agent.EventKinds()` (`event_registry_test.go:56-176`'s exact shape). It iterates exactly the four kinds AG-04 registers; a fifth entry or a fifth constant alone fails by name. Unlike AI-14, no `export_test.go` witness bridge is needed: AG-04 registers real production kinds from birth, so the guard lives wholly in `package agent_test` over the public surface — which is itself the charter's external-package acceptance criterion (`0003:418`). Bite proof: plant a scratch constant + registry row with no constructible payload → guard fails naming it (red recorded) → delete; scratch absent from the merged diff.

## Testing Strategy (Strict TDD — 11 scenarios, each RED first)

| Phase | Scenarios (charter `0003:438-514`) | RED first, then | File |
| --- | --- | --- | --- |
| AG-04.1 | kind derives from payload + validation gate + external identity reads; two lanes stamped under `-race`, each contiguous 1-based, checked by `CheckStream`; parent present on delegated, absent on top-level | envelope + stamper + minimal `CheckStream` contiguity core | `envelope_test.go` |
| AG-04.2 | run bracket accepted / every violating permutation rejected (two run-starts, no run-end, event after run-end) + typed run outcome; turn nesting (outside-run, overlap rejected) + typed turn outcome; nothing follows the terminal | scope engine rules 2-4, end-of-slice rule | `stream_check_test.go` |
| AG-04.3 | no construction route from any delta kind to an accumulated payload (pinned as: registry set == exactly the 4 bracket kinds, no accumulated-message constructor exported); failure category+cause reachable as values, pre/mid distinguishable, no message-string meaning | `failure.go`, `doc.go` row + guard table entry | `invariant_pin_test.go` |
| AG-04.4 | guard bites on a scratch kind with no constructible payload, by name (red recorded); iterates exactly AG-04's kinds | witness table | `event_registry_test.go` |
| Regression | AG-03's three guards green, zero logic edits; `make lint` clean; `go.mod`/`go.sum` byte-diff empty | — | existing files |

Non-claims restated in test names/comments per risk 4: nothing asserts invariant 3, and nothing claims sole closure of invariants 1, 2 or 4 (`0003:2203`).

## Threat Matrix

N/A — no routing, shell, subprocess, VCS/PR automation, executable-file classification, or process-integration boundary. AG-04 adds zero I/O and zero subprocess use; the only existing subprocesses are AG-03's test-only constant-argv `go list` invocations, unchanged.

## Migration / Rollout

No migration, no config change, no dependency change. Rollback per proposal levels 1-3 (revert `doc.go` row + table entry; delete AG-04's files returning the package to post-AG-03 state; revert the merge). Forward-fix preference stands: a rule too strict for AG-05's real sequences is amended in AG-05's change with its own justification, never deleted here to pass.

## Open Questions

None blocking. All four proposal open decisions plus the concurrency question are resolved above (AD-1..AD-5), each with its falsifier recorded where evidence could not fully close it (AD-2: AG-11's taxonomy fit; AD-5: a two-writer lane observed at AG-20.2/AG-21 amends AG-01 first).
