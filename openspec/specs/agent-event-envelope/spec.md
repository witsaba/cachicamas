# Spec — The agent event envelope and ordering invariants (`agent-event-envelope`)

> **Change**: `cachicamas-agent-event-envelope` · **Milestone**: AG-04 (Layer 2, Wave 1) of [doc 0003](../../../docs/architecture/milestones/0003-cachicamas-agent-layer-2-task-graph.md#ag-04--define-the-agent-event-envelope-and-ordering-invariants), `0003:410-517`
> **Nodes**: AG-04.1 `[leaf]` · AG-04.2 `[leaf]` · AG-04.3 `[leaf]` · AG-04.4 `[guard]`
> **Introduced by**: `openspec/changes/archive/2026-08-12-cachicamas-agent-event-envelope/`, opened as a PR against `main` on 2026-08-12 (PR number and merge commit to be back-filled once merged, per this repo's convention on already-merged specs).
> **Amended 2026-08-12 (AG-05)**: `R-AEV-007` and `R-AEV-010` MODIFIED (invariant 1 co-closure now joint with AG-05.1; scope-fence retightened from 4 to 15); `R-AEV-012` ADDED (documents the AG-04.4 extensibility experiment path AG-05 took). See delta spec at `openspec/changes/archive/2026-08-12-cachicamas-agent-message-tool-events/specs/agent-event-envelope/spec.md` and the AG-05 archive report at `openspec/changes/archive/2026-08-12-cachicamas-agent-message-tool-events/archive-report.md`.
> **Amended 2026-08-13 (AG-06)**: `R-AEV-010` MODIFIED (scope-fence retightened from 15 to 25; forbidden-names list retires); `R-AEV-012` MODIFIED (extensibility pattern restated to record AG-06 as the third kind-set following AG-04.4); `R-AEV-013`, `R-AEV-014`, `R-AEV-015` ADDED (registry stands at 25; L2C-06 doc-contract row; AG-06 envelope invariants compliance). See delta spec at `openspec/changes/archive/2026-08-13-cachicamas-agent-protocol-events/specs/agent-event-envelope/spec.md` and the AG-06 archive report at `openspec/changes/archive/2026-08-13-cachicamas-agent-protocol-events/archive-report.md`.
> **Status**: **live** — the invariants below hold for the lifetime of Layer 2, not only at the moment AG-04 merged
> **Format**: Given/When/Then + RFC 2119 per `openspec/config.yaml`. Every scenario is independently verifiable.
> **Identifier convention**: requirements `R-AEV-0NN`, scenarios `S-AEV-0NN`. Append-only. (`R-AGE-`/`S-AGE-` belong to `agent-event-delivery` and are not reused here.)
> **Evidence gate**: `cd backend/agent && make test` (`go test -race -v ./...`), plus `make lint`. No CI exists.
> **Authoring constraint**: this spec states obligations and observable behavior. It names **no** Layer 2 Go type, field or function; `sdd-design` owns the Go shape (doc 0003's authoring constraint). Layer 1 identifiers already shipped (`ai.Event`, `ai.FailureCategory`, `ai.CheckStream`, `ai.PreStreamFailure`, `ai.MidStreamFailure`) are cited as precedent and as alignment targets, never as Layer 2's own vocabulary.

## Purpose

Define the contract for the envelope every Layer 2 agent event travels in — identity, derived kind, per-lane ordering — together with the two event families AG-04 owns (run lifecycle `VL2-EVT-02`, turn lifecycle `VL2-EVT-03`), the typed-failure surface, and the **stream-contract validator** (`VL2-EVT-16`) that makes all of it assertable before any producer exists.

No producer exists until wave 2. Every requirement below is therefore stated so it is verifiable over **hand-built sequences** constructed through the package's public surface from an external test package (`0003:417-418`).

## Status — this file is the canonical home of the contract

`openspec/AGENTS.md` calls `openspec/specs/` the *source of truth (main specs)*. The envelope contract therefore lives here. The archived change folder at [`openspec/changes/archive/2026-08-12-cachicamas-agent-event-envelope/`](../../changes/archive/2026-08-12-cachicamas-agent-event-envelope/) is the historical record of how AG-04 was explored, proposed, designed, applied and verified — including all recorded REDs and any doc 0003 amendments the implementation forced.

A later milestone that needs to change these specifications — for example AG-05 adding message families to the validator — amends **this file**, in the same pull request, under its own ADR gate.

## Requirements

### R-AEV-001 — An event's kind derives from its payload and never disagrees with it

Every agent event MUST carry a kind that is **derived from its payload**, never stored alongside it, so that no construction route can produce an event whose kind and contents disagree (`VL2-EVT-01`, `openspec/changes/archive/2026-08-11-cachicamas-agent-contract-vocabulary/decision.md:162`). Layer 2's kind vocabulary MUST be its own closed set, independent of Layer 1's `ai.EventKind`, per `S-AGV-020`/`S-AGV-021` (`openspec/specs/agent-contract-vocabulary/spec.md:122-123`).

A validation gate MUST reject an event whose payload is absent (nil) and an event whose payload does not match the kind under which it is offered. The envelope's identity fields — run identity, turn identity, and parent identity — MUST be readable from a package **outside** `agent`, without a test-only build tag.

#### Scenarios

- **S-AEV-001** — Given an event constructed through the public surface from an external test package, when its kind is read, then the kind reported equals the kind implied by the payload it carries, and no public route sets the kind independently of the payload.
- **S-AEV-002** — Given an event offered with a nil payload, when it is validated, then validation FAILS and the failure names the missing payload rather than returning a zero-valued event.
- **S-AEV-003** — Given an event whose payload does not match the kind under which it is offered, when it is validated, then validation FAILS and the failure names both the offered kind and the payload it disagrees with.
- **S-AEV-004** — Given a valid event, when an external test package reads its run identity, turn identity and parent identity, then all three are readable through the exported surface with no build tag and no import of an internal or test-only package.
- **S-AEV-005** — Given the Layer 2 kind vocabulary, when it is enumerated, then it is a closed set declared by Layer 2 itself and no member is an alias, re-export or extension of a `ai.EventKind` member (`backend/agent/src/ai/event.go:74-142`).

### R-AEV-002 — Ordering is per-consumer lane, independent, contiguous and 1-based from birth

The envelope MUST expose a public ordering mechanism that stamps events with an ordering value that is **independent per consumer lane**, contiguous, and 1-based (`0003:445-448`). "Per-consumer stream" is AG-01's lane mechanism: one canonical internal stream, and per attached consumer a lane privately owning that consumer's receive-only carrier (`openspec/changes/archive/2026-08-11-cachicamas-agent-event-delivery/decision.md:230`). Layer 2's counter MUST be an independent agent-level counter and MUST NOT be `ai.Sequence` reused, aliased or extended (`S-AGV-021`).

Two lanes stamped concurrently MUST each remain independently contiguous and 1-based, and the mechanism MUST be exercised under the Go race detector (`make test` runs `go test -race`).

#### Scenarios

- **S-AEV-010** — Given two hand-built event sequences stamped through the envelope's public ordering mechanism, when each is checked by the stream-contract validator, then each carries an independent ordering starting at 1 with no gap and no repeat, and neither sequence's ordering is affected by the other's.
- **S-AEV-011** — Given the two sequences of `S-AEV-010` stamped from concurrently running goroutines, when the test runs under `-race`, then no data race is reported and both sequences remain contiguous and 1-based.
- **S-AEV-012** — Given a hand-built sequence whose ordering starts at a value other than 1, when the validator checks it, then it is REJECTED and the rejection names the ordering rule.
- **S-AEV-013** — Given a hand-built sequence with a gap or a repeat in its ordering, when the validator checks it, then it is REJECTED and the rejection names the offending position.

### R-AEV-003 — The parent identifier exists before any delegation mechanism does

The envelope MUST carry a parent identifier from birth: an event belonging to a delegated harness MUST carry its parent identifier, and a top-level event MUST carry none (`VL2-EVT-13`, `decision.md:174`). The field exists now because explicit nesting cannot be retrofitted; delegation semantics are AG-19's.

The absence of a parent identifier on a top-level event MUST be distinguishable from an unset or zero-valued identifier by inspection, not by convention.

#### Scenarios

- **S-AEV-020** — Given the envelope surface and no delegation mechanism in the package, when an event is constructed as belonging to a delegated harness, then it carries its parent identifier and that identifier is readable from an external package.
- **S-AEV-021** — Given an event constructed as top-level, when its parent identity is inspected, then it reports "no parent" as a distinguishable state rather than an ambiguous zero value.
- **S-AEV-022** — Given the package after this change, when its exported surface is enumerated, then it declares no delegation, subagent, or child-harness mechanism — only the parent identifier field and its accessor.

### R-AEV-004 — Exactly one run-start and one run-end bracket a run

The run lifecycle family (`VL2-EVT-02`) MUST require exactly one run-start preceding every other event and exactly one run-end following every other event on a run-scoped sequence. The run-end MUST carry a typed run outcome (`VL2-EVT-10`) whose values are **completed**, **interrupted** or **failed**; the outcome MUST NOT be absent — Layer 2 has no "sometimes no terminal at all" case at run scope (`decision.md:354`, `decision.md:541`).

**Nothing follows the terminal.** Any event appearing after run-end MUST cause the sequence to be rejected.

The run scope's owner and sole closer is the harness (`decision.md:354`). AG-04 ships no harness; this requirement constrains only what the validator accepts.

#### Scenarios

- **S-AEV-030** — Given a hand-built sequence in which one run-start precedes all other events and one run-end follows them, when the validator checks it, then it is ACCEPTED.
- **S-AEV-031** — Given a hand-built sequence containing two run-start events, when the validator checks it, then it is REJECTED naming the duplicate bracket opening.
- **S-AEV-032** — Given a hand-built sequence with no run-end, when the validator checks it, then it is REJECTED naming the unclosed run bracket.
- **S-AEV-033** — Given a hand-built sequence with any event after run-end, when the validator checks it, then it is REJECTED naming the event that follows the terminal.
- **S-AEV-034** — Given a hand-built sequence whose first event is not run-start, when the validator checks it, then it is REJECTED naming the event that preceded the bracket.
- **S-AEV-035** — Given run-end events constructed with each of completed, interrupted and failed, when each is inspected from an external package, then the outcome is reachable as a typed value distinguishing the three, and no public route constructs a run-end without one.

### R-AEV-005 — Turns nest strictly inside the run bracket and never overlap

The turn lifecycle family (`VL2-EVT-03`) MUST require turn-start/turn-end pairs that nest **strictly inside** the run bracket and never overlap for one harness. Turn-end MUST carry a typed turn outcome (`VL2-EVT-11`) distinguishing **model finished** from **turn aborted** (`decision.md:352`).

The turn scope's owner and sole closer is the loop, a different owner from the run scope's, because the loop is stateless and re-instantiated per turn (`decision.md:364`). AG-04 ships no loop; this requirement constrains only what the validator accepts.

#### Scenarios

- **S-AEV-040** — Given a hand-built sequence of one or more non-overlapping turn-start/turn-end pairs entirely inside the run bracket, when the validator checks it, then it is ACCEPTED.
- **S-AEV-041** — Given a hand-built sequence with a second turn-start before the preceding turn-end, when the validator checks it, then it is REJECTED naming the overlap.
- **S-AEV-042** — Given a hand-built sequence with a turn-start or turn-end outside the run bracket (before run-start or after run-end), when the validator checks it, then it is REJECTED naming the escaped turn bracket.
- **S-AEV-043** — Given a hand-built sequence with a turn-start never closed by a turn-end before run-end, when the validator checks it, then it is REJECTED naming the unclosed turn.
- **S-AEV-044** — Given turn-end events constructed with each outcome, when each is inspected from an external package, then "model finished" and "turn aborted" are distinguishable as typed values, and no public route constructs a turn-end without an outcome.

### R-AEV-006 — The stream-contract validator is production-exported and reusable

The stream-contract validator (`VL2-EVT-16`) MUST be exported from the **production** package `backend/agent/src/agent`, callable from another package without a test-only build tag, because it is reused wholesale by the Layer 3 readiness contract's kit at AG-23 (`VL2-SEAM-14`, `openspec/changes/archive/2026-08-11-cachicamas-agent-contract-vocabulary/decision.md:177`). It MUST NOT be an unexported test helper and MUST NOT live in a `_test.go` file.

It MUST accept a **finite, ordered sequence offered after the fact** (not a live channel), mirroring `ai.CheckStream`'s posture (`backend/agent/src/ai/stream_check.go:64`), and MUST accept a sequence only if **every** envelope invariant of `R-AEV-001`–`R-AEV-003` and **every** lifecycle bracket of `R-AEV-004`–`R-AEV-005` holds.

Its rejection report MUST name the offending position and the rule that rejected it, so a failure in a later milestone is actionable without reading the validator's source.

#### Scenarios

- **S-AEV-050** — Given a test file in a package other than `agent` and other than `agent_test` colocated helpers, when it calls the validator, then the call compiles and runs with no build tag and no import of a test-only package.
- **S-AEV-051** — Given the merged production sources, when the validator's declaration is located, then it is exported, declared in a non-`_test.go` file inside `backend/agent/src/agent`, and takes a finite ordered sequence rather than a channel.
- **S-AEV-052** — Given a sequence that violates exactly one rule, when the validator checks it, then the report names that rule and the position of the offending event, and does not report unrelated rules.
- **S-AEV-053** — Given a fully valid hand-built run containing run-start, two non-overlapping turn brackets, and run-end, when the validator checks it, then it is ACCEPTED with an empty violation set.
- **S-AEV-054** — Given the validator, when its checks are enumerated against this spec, then every rule of `R-AEV-001`–`R-AEV-005` that is expressible over a finite sequence has a corresponding check, and no check enforces a rule this spec does not state.

### R-AEV-007 — Invariant pin 1: a delta kind has no route to an accumulated-message payload (joint with AG-05.1)

The envelope's public construction surface MUST offer **no route** by which a delta kind can be attached to an accumulated-message payload (`VL2-EVT-12`, envelope invariant 1, `decision.md:173`). A delta carries an index and the new fragment only.

AG-04 registers no message family, so this requirement is a **structural pin on the construction surface**, not an assertion about a message-delta kind AG-05 will introduce. AG-05.1 introduces `message_delta_text` and `message_delta_reasoning`; AG-05.1's per-kind construction surface inherits this pin. Envelope invariant 1 is therefore closed **jointly by AG-04.3 + AG-05.1** (`0003:2203`); AG-05.1's co-closure is asserted by `S-AMT-021` (bite, in the new spec). The pin remains structural rather than instance-based; AG-05.1's two delta kinds do not weaken it — they extend the surface it guards.
(Previously: requirement text referenced AG-05.1 as forthcoming; co-closure is now joint and the bite lives in the AG-05 spec.)

#### Scenarios

- **S-AEV-060** — Given the envelope's public construction surface, when every exported route from a delta kind to a payload is enumerated, then none accepts or produces an accumulated-message payload, and the pin is asserted mechanically rather than by comment.
- **S-AEV-061** — Given the package documentation, when the delta rule is read, then it states that a delta carries an index and the new fragment only, and that the absence of an accumulated-snapshot route is the mechanism AG-05 inherits.

### R-AEV-008 — Invariant pin 4: a failure payload is a typed value, never a message string

A failure carried by the envelope MUST be reachable through a **typed-failure surface** on which the failure's **category** and its **cause** are inspectable as values (`VL2-EVT-15`, envelope invariant 4, `decision.md:176`). The category vocabulary MUST be aligned with Layer 1's failure taxonomy — `ai.FailureCategory` (`backend/agent/src/ai/provider_failure.go:49`) and `(*ai.Failure).Category()` (`provider_failure.go:432`) — as a **wrap**, not a reuse-as-is (`S-AGV-020`).

Setup failures (pre-stream) and stream failures (mid-stream) MUST be distinguishable, mirroring Layer 1's two-constructor, one-concrete-type shape (`provider_failure.go:610`, `:622`). **No code path MAY assign meaning to a message string**, and no scenario in this spec is satisfied by asserting on message text.

This requirement closes invariant 4 jointly with AG-11.2 (`0003:2203`); it does not close it alone. The loop-level typed-error emission path is AG-11's.

#### Scenarios

- **S-AEV-070** — Given a failure payload carried by the envelope, when a consumer in an external package inspects it through the typed-failure surface, then the category is reachable as a typed value and the cause is reachable as an inspectable error value.
- **S-AEV-071** — Given a failure payload constructed on the pre-stream path and one constructed on the mid-stream path, when each is inspected, then the two are distinguishable through the typed surface without parsing any text.
- **S-AEV-072** — Given the Layer 2 failure category vocabulary, when it is compared with `ai.FailureCategory`'s nine members (`provider_failure.go:49-103`), then every Layer 2 category maps to a stated Layer 1 category and the mapping is declared in source, not inferred.
- **S-AEV-073** — Given every test written for this capability, when their assertions are enumerated, then none asserts on the content of a failure message string as the carrier of meaning.

### R-AEV-009 — The every-kind-constructible guard, closing on a recorded bite

`backend/agent/src/agent/` MUST carry an executable guard that iterates **every event kind the package registers** and constructs a valid instance of each **through the public surface**, from an external test package. A kind that is registered but has no constructible payload MUST fail the guard, and the failure MUST **name the kind**.

The guard MUST be closed, not a containment check: a registered kind absent from the guard's witness table MUST fail, and a witness-table entry naming an unregistered kind MUST fail — mirroring Layer 1's bidirectional witness cross-check (`backend/agent/src/ai/event_registry_test.go:56-217`).

This guard closes on **bite proof**, not on green. A guard that has only ever been green is unproven.

The guard proves **construction-time exhaustiveness of the kind registry — nothing wider**. It MUST NOT be written or accepted as proof of envelope invariant 3 (observer asynchrony), nor of the full loop-level typed-error path.

#### Scenarios

- **S-AEV-080** — Given the repository with no violation present, when `cd backend/agent && go test ./src/agent/...` runs, then the every-kind-constructible guard passes and reports having constructed at least one instance per registered kind.
- **S-AEV-081** — Given the guard's source, when its construction path is read, then every instance is built through the package's exported surface from an external test package, and each constructed instance is then passed through the validation gate of `R-AEV-001`.
- **S-AEV-082** — **(bite)** Given a scratch event kind registered with no constructible payload, when the every-kind-constructible guard runs, then it FAILS and its message NAMES the offending kind. The failing output is recorded; the scratch kind is then removed and is absent from the merged diff.
- **S-AEV-083** — Given a witness-table entry naming a kind the registry does not contain, when the guard runs, then it FAILS naming the unknown entry — proving the cross-check is bidirectional and not a containment check.
- **S-AEV-084** — Given the guard's source, when its recorded scope note is read, then it states that the guard proves construction-time exhaustiveness only, and closes neither envelope invariant 3 nor the loop-level typed-error path.

### R-AEV-010 — AG-04/AG-05/AG-06 register exactly six families; the scope-fence now stands at 25

The package MUST register exactly the eight event families its milestones own across Wave 1: AG-04 owns run lifecycle (`VL2-EVT-02`) and turn lifecycle (`VL2-EVT-03`); AG-05 owns message lifecycle (`VL2-EVT-04`) and tool execution (`VL2-EVT-05`); AG-06 owns permission (`VL2-EVT-06`), cost (`VL2-EVT-07`), delegation (`VL2-EVT-08`), and compaction (`VL2-EVT-09`). The registry MUST hold exactly 25 kinds (4 AG-04 + 11 AG-05 + 10 AG-06) — no stub, placeholder, or reservation outside these eight families. The scope-fence `S-AEV-090` retightens from "exactly 15" to "exactly 25" in the same commit as the AG-06 kinds land; the forbidden-names list at `event_registry_test.go:326` retires (permission, cost, delegation, compaction are now AG-06's own).

The registry and the guard MUST remain **structurally extensible**: AG-07 onward MUST be able to add a kind by following the documented procedure without editing the validator's rule engine, mirroring the AG-04.4 extensibility experiment (`S-AEV-092`, restated in `R-AEV-012`; AG-06 followed the same path, recorded by `R-AEV-015`).
(Previously: scope-fence held at "exactly 15"; AG-06's four families were forbidden; AG-06 retightens to "exactly 25" and the forbidden-names list retires.)

#### Scenarios

- **S-AEV-090** — Given the package's registered kind set, when it is enumerated, then it contains exactly the run, turn, message, tool, permission, cost, delegation and compaction lifecycle kinds (25 kinds total: 4 AG-04 + 11 AG-05 + 10 AG-06) and no further kind of any name.
- **S-AEV-091** — Given the package documentation, when its "adding a kind" procedure is read, then it states the ordered steps a later milestone follows to register a new kind, and states that following them requires no edit to the validator's rule engine.
- **S-AEV-092** — Given the registry, when a new kind is added following the documented procedure in a scratch experiment, then the every-kind-constructible guard and the validator both continue to compile and run without edits to their own logic. Recorded, then reverted.

### R-AEV-011 — The ordering invariants and the membership criterion are stated in the package documentation

The package documentation MUST state the ordering invariants of `R-AEV-002` and MUST state the membership criterion the charter names verbatim in substance: **"if it is not on the stream, no frontend can render it and no log can reconstruct it"** (`0003:418`), as the criterion later families are judged by.

Both statements MUST be pinned by test, so a later edit that removes or contradicts them fails rather than drifts. Where the pin is the machine-checked doc-row table of `R-AGP-002` (`openspec/specs/agent-package-scaffold/spec.md:39-45`), any added row MUST be accompanied by its expectation-table amendment **in the same pull request**.

#### Scenarios

- **S-AEV-100** — Given `backend/agent/src/agent/`'s package documentation, when it is read, then it states that ordering is per consumer lane, independent, contiguous and 1-based, and that Layer 2's counter is not Layer 1's per-stream sequence.
- **S-AEV-101** — Given the package documentation, when it is read, then it states the membership criterion and identifies it as the criterion later event families are judged by.
- **S-AEV-102** — Given a scratch edit that removes or contradicts either statement, when `cd backend/agent && go test ./src/agent/...` runs, then a test FAILS naming the missing or divergent statement. Recorded, then reverted.

### R-AEV-012 — AG-04/AG-05/AG-06 followed the AG-04.4 extensibility experiment pattern

AG-04's 4 kinds, AG-05's 11 kinds, and AG-06's 10 kinds MUST all be registered by following the seven-step "adding a kind" procedure documented at `event_descriptor.go:13-46`, with no edit to `stream_check.go`, `event_registry_test.go`, or `failure.go`. The AG-04.4 extensibility experiment (`S-AEV-092`) is the documented path all three kind-sets took — proven by AG-04's `S-AEV-082` bite, AG-05's `S-AMT-081` bite (a 16th scratch kind fails the scope-fence by count), AG-06's `S-APE-081` bite (a 26th scratch kind fails by count), AG-05's `R-AMT-009` placement assertion (`S-AMT-080`, all 11 AG-05 kinds register under `PlacementTurn`), and AG-06's `R-AEV-015` invariant compliance assertion (`S-AEV-124`, AG-06's parent-identifier and `CardinalityAtMostOne` exercises).

#### Scenarios

- **S-AEV-110** — Given AG-05's 11 new kinds registered following the six-step procedure, when the validator checks a hand-built sequence containing a `message_start_text` outside an open turn, then it is REJECTED naming the `PlacementTurn` rule — proving the seam AG-04.3 reserved was actually exercised.
- **S-AEV-111** — Given the registry after AG-05, when `stream_check.go` and `event_registry_test.go` are diffed against the AG-04 merge (`967d043f`), then the diffs are empty for both files; AG-05's value is in the substrate's extensibility, demonstrated.
- **S-AEV-112** — Given the seven-step procedure doc, when it is read, then it states that any future kind declaring `Terminal: true` MUST be honored by the validator (the W3 latent-trap guard), that `CardinalityAtMostOne` is exercised by AG-06's `permission_resolution_remembered` (R-APE-003, `S-APE-082`), and that future kinds MAY opt into the same seam via the descriptor row.

### R-AEV-013 — AG-06's registry holds exactly 25 kinds; the scope-fence now stands at 25

The package MUST register exactly the eight event families its milestones own across Wave 1: AG-04 owns run lifecycle (`VL2-EVT-02`) and turn lifecycle (`VL2-EVT-03`); AG-05 owns message lifecycle (`VL2-EVT-04`) and tool execution (`VL2-EVT-05`); AG-06 owns permission (`VL2-EVT-06`), cost (`VL2-EVT-07`), delegation (`VL2-EVT-08`), and compaction (`VL2-EVT-09`). The registry MUST hold exactly 25 kinds (4 AG-04 + 11 AG-05 + 10 AG-06) — no stub, placeholder, or reservation. The scope-fence `S-AEV-090` retightens from "exactly 15" to "exactly 25" in the **same commit** as the AG-06 kinds land. The forbidden-names list at `event_registry_test.go:326` (the AG-04/AG-05 forward-fence against AG-06's families) retires — permission, cost, delegation, and compaction are AG-06's own.

#### Scenarios

- **S-AEV-120** — Given the package's registered kind set, when it is enumerated, then it contains exactly **25 kinds** (4 AG-04 + 11 AG-05 + 10 AG-06) — the four protocol families included — and no further kind of any name; the forbidden-names list is empty or absent.
- **S-AEV-121** — Given the every-kind-constructible guard, when it runs, then it constructs at least one instance of every registered kind (25 total) through the public surface from an external test package; the forbidden-names check does not flag any AG-06 family name.

### R-AEV-014 — L2C-06 doc-contract row references the four protocol families

The doc-contract guard's `expectedLayer2ContractRows` table MUST carry an `L2C-06` row added in the same commit as the AG-06 kinds and the `L2C-06` row in `doc.go`. The row's text states that the four protocol families (permission, cost, delegation, compaction) are constructible on the event stream and that the per-family semantics belong in `doc.go` prose, not in the guarded row, per `R-AGP-002`'s closed-amendment rule. The guardian of this amendment is the same `doc_contract_guard_test.go` that guards `L2C-01`..`L2C-05`.

#### Scenarios

- **S-AEV-122** — Given the `expectedLayer2ContractRows` table in `doc_contract_guard_test.go`, when it is read, then it contains 6 rows (`L2C-01`..`L2C-06`) — the new `L2C-06` row is present, in order, with row text referencing the four protocol families.
- **S-AEV-123** — Given a scratch edit that appends an `L2C-06` row to `doc.go` without adding its entry to `expectedLayer2ContractRows`, when the doc-contract guard runs, then it FAILS naming the unexpected row — the closed-amendment rule is observed, not bypassed. RED-recordable.

### R-AEV-015 — Protocol family kinds follow the AG-04.1 envelope invariants

Every AG-06 kind MUST follow the AG-04.1 envelope invariants: derived kind from payload (`R-AEV-001`); identity fields (run, turn, parent) readable from an external package (`R-AEV-001`/`R-AEV-003`); and per-lane ordering 1-based and contiguous (`R-AEV-002`). `subagent_started` and `subagent_ended` are the **first** non-`NewDelegatedRunStart` consumers of the parent identifier field (`event.go:362-366`, `R-AEV-003`) — the field has existed from AG-04.1 and AG-06.3 is its first extension. AG-06 closes invariant 2 (explicit nesting) **partially**; AG-19.1 closes it fully. The `CardinalityAtMostOne` seam reserved at AG-04.3 (`event_descriptor.go:103-120`) is exercised for the first time by `permission_resolution_remembered` (R-APE-003).

#### Scenarios

- **S-AEV-124** — Given the 10 new AG-06 kinds registered following the seven-step procedure, when the every-kind-constructible guard constructs each through the public surface and inspects them, then every kind's identity fields (run, turn, parent) are readable from an external package; `subagent_started`/`subagent_ended` distinguishably report `(parentID, true)`; `permission_resolution_remembered` declares `CardinalityAtMostOne`; `compaction_failed` declares `Terminal: false` — the three AG-06 substrate exercises are mechanically asserted.

## Non-functional requirements

### NFR-AEV-001 — External-package verifiability

Every scenario above MUST be verifiable by `cd backend/agent && make test`. Every behavioral test MUST live in an external test package (`package agent_test` or another package entirely), so that a behavior reachable only from inside the package is, for the purposes of this spec, not reachable at all — the charter's own acceptance criterion (`0003:418`).

### NFR-AEV-002 — Determinism and race cleanliness

Every test added by this change MUST be deterministic and hermetic — no network, no filesystem outside `t.TempDir()`, no environment dependence — and MUST pass under `-race`, which `make test` applies to the whole module.

### NFR-AEV-003 — Boundary guards stay green untouched

AG-03's `import_boundary_test.go` and `ambient_authority_test.go` MUST pass with **zero changes to their own logic**. AG-04 imports only `backend/agent/src/ai` and the standard library, both already admitted by `R-AGP-003`.

### NFR-AEV-004 — Review budget

`openspec/config.yaml` forecasts a 400-line review budget; this change ships as a single pull request under a pre-authorised `size:exception` against a 1000-line budget, forecast at 1400–2200 changed lines. The pull request description MUST state why the change does not fit the default budget.

## Explicit non-requirements — what this spec does NOT claim

Stated so that no test, guard or acceptance line is written as if AG-04 closes more than it does. The traceability spine (`0003:2203`) is authoritative:

| Envelope invariant | Closed by | AG-04's part |
| --- | --- | --- |
| 1 — indexed deltas | AG-04.3 **+ AG-05.1** | the construction-surface pin only (`R-AEV-007`) |
| 2 — explicit nesting | AG-04.1 **+ AG-19.1** | the parent identifier exists; delegation semantics are AG-19's (`R-AEV-003`) |
| 3 — non-blocking observers | **AG-01.1 + AG-20.2 — AG-04 absent** | **none.** No requirement here closes any part of invariant 3 |
| 4 — typed errors | AG-04.3 **+ AG-11.2** | the typed-failure surface exists; loop-level emission is AG-11's (`R-AEV-008`) |

**AG-04 closes no envelope invariant by itself.** No requirement, scenario, test name or acceptance line may claim otherwise.

Also out of scope, each with a named owner:

- Message lifecycle and tool execution families (`VL2-EVT-04`, `VL2-EVT-05`) — **AG-05** (shipped 2026-08-12, PR #164; kinds registered, scope-fence now at 15 per `R-AEV-010` retrospective, `R-AEV-012` documents the path taken).
- Permission, cost, delegation, compaction families (`VL2-EVT-06`…`VL2-EVT-09`) — **AG-06**.
- Delivery mechanics — carrier, buffering posture, observer attachment, the upward path — **AG-01**, already decided and merged.
- Any loop or harness behavior: no producer, no driver, no turn execution — **AG-07 onward**. The run and turn scopes have named owners (`decision.md:352-354`); AG-04 ships neither owner, only the validator that will judge them.
- Test-convenience wrappers over the validator in `backend/agent/src/agenttest` — open decision 4 of the proposal; the charter's dependency edges (`0003:419`) name only AG-01 and AG-03.
- `backend/agent/src/coding/`, `backend/agent/src/cmd/` — doc 0004. Their absence keeps AG-03's forbidden-prefix rows testable.
- Any new dependency: no `go.mod` or `go.sum` edit.
- CI. Every gate runs when a human runs `make test` in `backend/agent/`.

## Evidence discipline

`openspec/config.yaml` declares `apply.tdd: true`; strict TDD is active for this repo (`openspec/AGENTS.md`).

- **All four nodes are behavior, so all four are RED-first.** Each scenario closes on a recorded failing run followed by green.
- **`R-AEV-009` closes only on its recorded bite** (`S-AEV-082`): the scratch kind fails the guard **by name**, red recorded, scratch file absent from the merged diff. Green alone does not close it.
- **`R-AEV-011` closes on its recorded bite** (`S-AEV-102`).
- **`R-AEV-010` closes on its recorded extensibility experiment** (`S-AEV-092`), reverted before merge.

## Acceptance criteria

The contract holds when:

1. Every scenario `S-AEV-001` through `S-AEV-102` and `S-AEV-110` through `S-AEV-112` and `S-AEV-120` through `S-AEV-124` has recorded evidence.
2. `cd backend/agent && make test` and `make lint` are green, recorded pre- and post-change.
3. `backend/agent/go.mod` and `go.sum` are byte-unchanged.
4. An external-package test constructs, validates and inspects **every** kind this milestone registers.
5. The stream-contract validator is exported from the production package and callable from another package with no test-only build tag.
6. Two independently stamped hand-built sequences are checked under `-race` and each is contiguous and 1-based.
7. Every violating lifecycle permutation named by `R-AEV-004` and `R-AEV-005` is rejected with a named rule.
8. The every-kind-constructible guard's bite is recorded red and its scratch kind is absent from the merged diff.
9. Exactly the run, turn, message, tool, permission, cost, delegation and compaction lifecycle families are registered (25 kinds: 4 AG-04 + 11 AG-05 + 10 AG-06); no further kind of any name appears. (AG-05 retightened this scope-fence from 4 to 15 in the same commit as the AG-05 kinds landed; AG-06 retightens to 25 in the same commit as the AG-06 kinds land; AG-07 onward must follow the same pattern to extend to 26+.)
10. The ordering invariants and the membership criterion are in the package documentation and pinned by test.
11. AG-03's two boundary guards pass with zero changes to their own logic.
12. No spec line, test name or acceptance line claims AG-04 closes envelope invariant 3, or invariants 1, 2 or 4 on its own.

## Traceability

| Requirement | Charter node | Register rows | Charter scenario (`0003`) |
| --- | --- | --- | --- |
| `R-AEV-001` | AG-04.1 | `VL2-EVT-01` | `:439-443` |
| `R-AEV-002` | AG-04.1 | `VL2-EVT-01`; AG-01 § 5 | `:445-448` |
| `R-AEV-003` | AG-04.1 | `VL2-EVT-13` | `:450-453` |
| `R-AEV-004` | AG-04.2 | `VL2-EVT-02`, `VL2-EVT-10` | `:464-468`, `:476-479` |
| `R-AEV-005` | AG-04.2 | `VL2-EVT-03`, `VL2-EVT-11` | `:470-474` |
| `R-AEV-006` | AG-04.2 (asserted through by all nodes) | `VL2-EVT-16`, `VL2-SEAM-14` | `:417` (deliverable) |
| `R-AEV-007` | AG-04.3 + AG-05.1 (joint) | `VL2-EVT-12` | `:489-492` |
| `R-AEV-008` | AG-04.3 | `VL2-EVT-15` | `:494-498` |
| `R-AEV-009` | AG-04.4 | `VL2-EVT-01` (enforced by AG-04.4) | `:510-513` |
| `R-AEV-010` | AG-04.4 (scope fence) | `VL2-EVT-02`…`VL2-EVT-09` | `:420` |
| `R-AEV-011` | AG-04.1, AG-04.3 | — | `:418` (acceptance) |
| `R-AEV-012` | AG-05 (extensibility experiment) | `VL2-EVT-04`, `VL2-EVT-05` (the 11 new kinds) | `:543-583`, `:594-597` (AG-05 charter) |

All 11 charter Gherkin scenarios (`0003:438-514`) are represented; none is reduced.

## Recorded contradictions and assumptions

1. **Charter scenario "one run-start and one run-end" and "nothing follows the terminal" both land on `R-AEV-004`.** They are two charter scenarios but one bracket rule; they are kept as separate independently verifiable spec scenarios (`S-AEV-030`–`S-AEV-034`, with `S-AEV-033` carrying the terminal rule) rather than merged.
2. **The membership criterion is an acceptance clause, not a Gherkin scenario, in the charter** (`0003:418`). It is promoted to a requirement here (`R-AEV-011`) because the charter states it as a deliverable of AG-04 and it is otherwise unowned.
3. **Whether `R-AEV-011`'s pin is a new `doc.go` guarded row or a separate test is unresolved** — proposal open decision 3, owned by `sdd-design`. The requirement states the obligation; it does not choose the mechanism. If a row is added, `agent-package-scaffold` takes a delta amending its expectation table in the same pull request (`R-AGP-002`).
4. **`R-AEV-008`'s Go shape is unresolved** — proposal open decision 2. "Wrap, not reuse-as-is" is settled by `S-AGV-020`; the shape is design's.
5. **`R-AEV-007` is asserted over a construction surface that registers no delta kind at AG-04.** The pin is therefore structural (no route exists) rather than instance-based. If design concludes the pin is unassertable before AG-05 registers a delta kind, that is a finding to record against this requirement, not a reason to weaken it silently. **Closed at AG-05 archive (2026-08-12):** AG-05.1 introduced `message_delta_text` and `message_delta_reasoning`; the pin is now structural *and* instance-based. `S-AMT-021` (bite) mechanically asserts no route from a delta kind to an accumulated payload. Invariant 1's co-closure by AG-04.3 + AG-05.1 is now observed, not merely structural.
