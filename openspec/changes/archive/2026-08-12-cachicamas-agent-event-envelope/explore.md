# Exploration — AG-04 the agent event envelope and ordering invariants

> Milestone AG-04 (Layer 2 Wave 1), `docs/architecture/milestones/0003-cachicamas-agent-layer-2-task-graph.md:410-517`. SDD change slug: `cachicamas-agent-event-envelope`. Artifact store: hybrid (Engram + OpenSpec). Engram topic key: `sdd/cachicamas-agent-event-envelope/explore`.

## 1. What already exists — Layer 1 `ai.Event` vs the Layer 2 agent event (the exact boundary)

Package `agent` (`backend/agent/src/agent/`) currently contains **only `doc.go`** (19 lines, `backend/agent/src/agent/doc.go:1-19`) — zero production types, zero behavior. AG-04 is the **first milestone to add real Go production code** to Layer 2. All existing files in the package are `_test.go` guard files (`import_boundary_test.go`, `ambient_authority_test.go`, `doc_contract_guard_test.go`), all in `package agent_test`.

**Layer 1's `ai.Event`** (`backend/agent/src/ai/event.go:281-284`) is a **stream-scoped** value: `{payload eventPayload; seq Sequence}`. Its `Sequence` (`backend/agent/src/ai/sequence.go:41`) is stamped 1-based, contiguous, **per provider-response stream**, by a `Stamper` (`sequence.go:50-60`) that is a bare struct with one unsynchronized counter — safe only because AI-02 guarantees one producer goroutine per stream. Layer 1 registers 12 production `EventKind`s (`event.go:74-142`): responsestart, completion, reasoning x3, text x3, tool-call x3, error.

**The Layer 2 agent event is explicitly NOT the same thing**, per AG-00's canonical register (`openspec/changes/archive/2026-08-11-cachicamas-agent-contract-vocabulary/decision.md:117` requirement, and the actual register rows at `.../decision.md:162-180`, promoted live at `openspec/specs/agent-contract-vocabulary/spec.md`):

- **Reused as-is** by Layer 2: message identity, tool-call identity, finish reasons, usage (`openspec/specs/agent-contract-vocabulary/spec.md:117`, `S-AGV-019`).
- **Wrapped, not reused**: events, ordering, and failure (`spec.md:117`, `S-AGV-020/021`). The wrap requirement is explicit and normative: *"the Layer 1 event is stream-scoped while the Layer 2 envelope is run/turn-scoped... Layer 2 ordering is an independent agent-level counter and explicitly not the Layer 1 per-stream sequence"* (`S-AGV-020`, `S-AGV-021`).
- Concretely: `VL2-EVT-01` **event kind** (`decision.md:162`) is Layer 2's own closed, derived-from-payload discriminator, mechanically exhaustive like `ai.EventKind` but a *different* vocabulary — not an extension of `ai.EventKind`'s twelve members. AG-04 registers only the run/turn lifecycle members of it (`VL2-EVT-02`, `VL2-EVT-03` families); `VL2-EVT-04`...`VL2-EVT-09` (message, tool, permission, cost, delegation, compaction) are AG-05's and AG-06's, not AG-04's, even though the kind vocabulary itself is one closed set AG-04.4's guard iterates going forward.
- Ordering is **per-consumer-stream** (agent-level, section 5 below), not per-provider-response-stream — a genuinely new counter, not `ai.Sequence` reused or subclassed.

**Precedent mechanisms Layer 2 should structurally mirror** (same shape, not the same type — the milestone doc forbids naming Layer 2 types):

- `EventKind` derivation-not-storage + `CheckEmit` validation gate: `event.go:33-72` (kind vocabulary), `event.go:321-368` (`CheckEmit`).
- Ordering: `Sequence`/`Stamper`, 1-based, per-owning-scope, unsynchronized-by-single-producer-guarantee: `sequence.go:32-60`.
- Ordering-invariant checker over a finite slice: `CheckStream`/`StreamReport`, `backend/agent/src/ai/stream_check.go:64-118` — reads only (kind, descriptor, block index, sequence), never a concrete payload type (`stream_check.go:1-10, 45-63`).
- Kind registry + exhaustiveness guard: `eventRegistry` (`event.go:162-211`) + the witness-table exhaustiveness test `TestEventKindRegistration_TheTestKindVocabulary_HasConstructorAndAccessor` (`backend/agent/src/ai/event_registry_test.go:56-217`), driven by the six-step "adding a kind" procedure documented at `backend/agent/src/ai/event_descriptor.go:12-32`.

## 2. The Layer 1 failure taxonomy, and what "aligned with" concretely means for AG-04.3

**Real types, real file**: `backend/agent/src/ai/provider_failure.go`.

- `FailureCategory` (uint8, closed, append-only, 9 members): `Authentication, Authorization, RateLimit, Unavailable, Timeout, Cancellation, MalformedResponse, UnsupportedCapability, Unknown` (`provider_failure.go:49-103`), each with its own `errors.Is`-compatible sentinel (`provider_failure.go:179-205`) — no umbrella sentinel, by design (`provider_failure.go:174-178`).
- `Failure` (`provider_failure.go:320-330`): unexported fields, pointer-receiver accessors, two axes kept perpendicular (`PartialOutput() bool` at `:515-520`, independent of `Delivery() DeliveryPath` at `:528-533`), `Category()`, `RetryAfter()`, `StatusClass()`, `RequestID()`, `Unwrap()`/`Is()` for `errors.Is`/`errors.As`.
- Two constructors, same concrete type, no converter: `PreStreamFailure` (`:610-612`) and `MidStreamFailure` (`:622-624`) — the "same concrete type, both paths" property (`provider_failure.go:13-21`, `R-AIP-013`).
- `ErrorEvent(f *Failure) (Event, error)` (`:644-652`) wraps a mid-stream `*Failure` as the stream's terminal event payload.

**"Aligned with" is already decided, not open**: AG-00's register states failure is a **wrapped** identity, not reused as-is (section 1 above). `VL2-LOOP-09` **typed turn failure** (`decision.md:196`) — owned by AG-11, not AG-04 — is explicit: *"Layer 1's failure taxonomy carried into turn scope: category, retryability, and the partial-output discriminator... inspectable as typed values... never a message string."* AG-04.3's own scenario only needs the envelope-level piece of this: a typed-failure surface where "category and cause are reachable as values" (the milestone doc's own wording, `0003:494-498`) — i.e. Layer 2's envelope-level failure payload should expose `ai.FailureCategory`-shaped classification and an unwrappable cause, structurally mirroring `Failure.Category()`/`Failure.Unwrap()`, not inventing a parallel vocabulary. `VL2-EVT-15` (`decision.md:176`) confirms the scope AG-04 itself owns: *"a failure is a typed value reachable through the typed-failure surface, never a message string... setup failures (pre-stream) and stream failures (mid-stream) are distinguishable"* — the pre-stream/mid-stream distinction is explicitly inherited from Layer 1's `DeliveryPath` shape (`provider_failure.go:226-247`).

**Open for design, not resolved by this exploration**: whether Layer 2's envelope-level failure payload directly embeds/re-exports `*ai.Failure` (thin wrap) or defines its own value type that is merely category/cause-compatible. The register commits to "wrapped, not reused as-is" but does not fix the Go shape — correctly, per the milestone doc's authoring constraint. This is a design-phase decision, flagged, not resolved here.

## 3. The stream-contract validator — required shape (from the register, `VL2-EVT-16`)

`decision.md:177`: *"The reusable checker that runs over any hand-built or produced agent event sequence and accepts it only if every envelope invariant and lifecycle bracket holds. Exists so the invariants are assertable before any producer exists, and is reused wholesale by the Layer 3 readiness contract's kit (`VL2-SEAM-14`)."*

Consequences, all evidenced:

- It **must be reusable by AG-23's kit** (`decision.md:236`, `VL2-SEAM-14`), meaning it cannot be buried as an unexported test helper — it needs to be reachable the way `ai.CheckStream` is production-exported (`stream_check.go:64`) and then convenience-wrapped by `agenttest`'s `RequireValidStream`/`CheckContiguity` (`backend/agent/src/agenttest/doc.go:20-22`, delegating to `ai.CheckStream`). The precedent strongly suggests an equivalent split for Layer 2: a production-package validator function, optionally wrapped by a test-support sibling for ergonomics — mirroring AI-14's `CheckStream` (production) + AI-22's `agenttest.RequireValidStream` (test convenience wrapper) relationship exactly.
- It takes a **finite, ordered slice offered after the fact**, not a channel (`stream_check.go:45-54`'s own documented posture, and the milestone charter's own words: "no producer exists until wave 2... assertable over hand-built sequences," `0003:417`).
- It must check envelope invariants (indexed deltas, explicit nesting, typed errors) **and** lifecycle brackets (one run-start/run-end, non-overlapping turn brackets, nothing after terminal) — i.e. it is broader than `ai.CheckStream`'s scope (which never enforced a run/turn nesting discipline, only block start/delta/end and single-terminal).
- **Open, genuinely unresolved**: whether Layer 2's validator is a wholesale reuse of `ai.CheckStream`'s descriptor-driven design (kind -> `EventDescriptor` -> generic rule engine) or a new implementation because Layer 2 needs an *additional* nesting dimension (turn-inside-run) that Layer 1's single-level block model does not have. This is a real design fork, not a naming question — flagged for `sdd-design`, not resolved by assumption.

## 4. The registered event kind registry — the "every kind constructible" guard's real precedent

**Direct, working precedent exists in this repo** (contrast with AG-03's cycle, which correctly found *no* precedent for its own doc-guard convention — that gap is different from this one, and this one is NOT a gap):

- `eventRegistry` (`event.go:162-211`) pairs every kind with a name and an `EventDescriptor` in one table — "a kind cannot register without a descriptor" is structural (concrete struct field, not pointer/interface), machine-proven by `TestEventRegistration_DescriptorField_IsAConcreteRequiredType` (`event_registry_test.go:308-326`, an AST scan of the struct declaration).
- The exhaustiveness guard itself: a `map[EventKind]eventKindWitness` table (`event_registry_test.go:56-149`) where each entry supplies **leg 1** (a no-argument constructor closure returning `(Event, error)`) and **leg 2** (a payload accessor closure `func(Event) (any, bool)`), cross-checked bidirectionally against `AllTestEventKinds()` so a kind reachable by one path and not the other fails (`event_registry_test.go:163-176`).
- The six-step "adding a kind" procedure is documented once (`event_descriptor.go:12-32`) and followed by every Layer 1 kind-adding milestone (AI-15...AI-19) with zero changes to the checker itself.

**AG-04.4 should structurally mirror this witness-table pattern** for its own (smaller, run/turn-only) kind set — this is the load-bearing "C4 lesson" the milestone doc cites (`0003:412`, `503-517`): a kind two files declare mandatory but that cannot be constructed. The mechanism to prevent it already exists, verified, in this repo, at the file:line citations above.

## 5. Per-consumer-stream ordering — AG-01's actual decision, not the milestone doc's paraphrase

AG-04.1 scenario 2 (`0003:445-448`) says ordering is "per-consumer-stream and 1-based." The milestone doc's own wording is a paraphrase; the binding decision is AG-01.1's, archived at `openspec/changes/archive/2026-08-11-cachicamas-agent-event-delivery/decision.md`:

- Section 5 (`decision.md:224-313`) decides the **observer model**: one canonical internal stream (single receiver: "the distribution step"), and **per attached consumer, a "lane"** fed by its own forwarding activity, privately owning that consumer's receive-only carrier (`decision.md:230`, `VL2-SEAM-11`/`VL2-SEAM-12`). "Per-consumer-stream" in AG-04.1's scenario **is this lane** — each attached consumer (primary frontend, session logger, cost meter, ...) gets its own independently-ordered, contiguous, 1-based stream, exactly as the scenario states: *"two hand-built event streams stamped through the envelope's public ordering mechanism... each stream carries an independent, contiguous, 1-based ordering"* (`0003:445-448`).
- Section 6 (`decision.md:316-388`) decides the **three nested ownership scopes** (turn: the loop; run: the harness; delegated run: the child harness), each with exactly one owner and exactly one closer — this is what gives AG-04.2's run-start/run-end and turn-start/turn-end brackets their *owners*, cited explicitly in `decision.md:550-555` ("What AG-04 takes from this decision").
- **Important nuance the milestone doc does not spell out**: per the traceability spine (`0003:2201-2248`), AG-04 does **not** close envelope invariant 3 (non-blocking observers) by itself — that is closed by `AG-01.1` (the decoupling mechanism) + `AG-20.2` (the stalled-observer test). AG-04.3's own two scenarios pin only invariants 1 (indexed deltas) and 4 (typed errors); invariant 2 (explicit nesting) is only *partly* closed at AG-04 (the field exists; full semantics close at AG-19.1); invariant 4 is only partly closed at AG-04 (the typed-failure surface exists; the loop's actual typed-error emission closes at AG-11.2). **AG-04.4's guard must not be written as if it proves invariant 3 or the full loop-level typed-failure path — it proves only construction-time exhaustiveness of the kind registry.**

## 6. Scope boundary — exact family ownership (the highest-risk area for AG-04 overreach)

From `decision.md`'s live register (section 5.2, `VL2-EVT` category, `decision.md:156-180`) and doc 0003's own dependency graph (`0003:279-327`):

| Family | Owner | Register row |
| --- | --- | --- |
| Run lifecycle | **AG-04** | `VL2-EVT-02` |
| Turn lifecycle | **AG-04** | `VL2-EVT-03` |
| Message lifecycle | AG-05 | `VL2-EVT-04` |
| Tool execution | AG-05 | `VL2-EVT-05` |
| Permission | AG-06 | `VL2-EVT-06` |
| Cost | AG-06 | `VL2-EVT-07` |
| Delegation | AG-06 | `VL2-EVT-08` |
| Compaction | AG-06 | `VL2-EVT-09` |

AG-04 registers **exactly two** of the eight v2 section 4.3 families (run + turn lifecycle). The kind *vocabulary mechanism* (envelope, event kind, the registry, the validator, AG-04.4's guard) is shared infrastructure all eight families sit inside, but AG-04 itself must construct **no** message/tool/permission/cost/delegation/compaction payload — those are explicitly out of scope on AG-04's own charter (`0003:420`: "Out of scope: Message/tool families (AG-05); the four new families (AG-06)"). The most likely scope error is AG-04.4's guard accidentally requiring a payload shape or registering a placeholder kind for a family it does not own — the guard must iterate exactly the kinds AG-04 itself registers (run-start, run-end, turn-start, turn-end at minimum, per `VL2-EVT-02`/`VL2-EVT-03`/`VL2-EVT-10`/`VL2-EVT-11`), nothing else, while remaining structurally ready for AG-05/AG-06 to extend the same table later (exactly as Layer 1's registry table is extended, uneventfully, by every later kind-adding milestone).

## 7. Sizing forecast

AG-03 (scaffold + two guards, **zero production behavior**) shipped **1005 lines**, 5 over the pre-authorized 1000-line ceiling (`openspec/changes/archive/2026-08-12-cachicamas-agent-package-scaffold/archive-report.md:87-89`, flagged as W4, accepted via `size:exception`).

AG-04 is **larger in kind, not just degree**: it is the first milestone with actual production types (envelope, kind derivation, per-lane ordering, at least four constructible run/turn payloads, a typed-failure surface, a stream-contract validator) plus 4 nodes carrying 11 Gherkin scenarios total (3+3+2+1 bite-proof) each needing its own RED-then-GREEN test, plus the doc-contract guard's expectation table if a guarded paragraph is added to `doc.go`. For comparison, Layer 1's closest analogues: `event.go` (369 lines) + `event_descriptor.go` (144) + `sequence.go` (61) + `stream_check.go` (175) = **749 lines of production code alone** for AI-14's foundation (which registered *zero* production kinds — AG-04 must register at least four), plus `event_registry_test.go` (407 lines) for exhaustiveness alone, not counting AI-14's own scenario tests. A conservative estimate: **500-750 production lines + 900-1500 test lines, approximately 1400-2200 total changed lines** — plausibly 1.5-2x AG-03's already-over-ceiling size. Flagged explicitly to `sdd-tasks`/`sdd-propose` as high risk against the 1000-line budget.

## 8. Genuine open decisions / gaps — flagged, not resolved

1. **Stream-contract validator's Go shape**: production-exported function reused wholesale by AG-23's kit (settled requirement, `VL2-EVT-16`), vs. whether it needs a materially different internal design from `ai.CheckStream` because Layer 2 needs an extra nesting dimension (turn-inside-run) Layer 1's single-level block checker never had. **Real design fork, not a naming question.**
2. **Typed-failure payload shape**: thin wrap/embed of `*ai.Failure` vs. an independent category/cause-compatible value type. The register commits to "wrapped, not reused as-is" (settled) but not to a Go shape (open, correctly deferred to design/spec per the milestone doc's authoring constraint).
3. **Whether AG-04 adds a guarded paragraph to `doc.go`'s machine-checked layer-contract table.** The three existing rows (`doc.go:16-18`, L2C-01/02/03) cover imports, no-I/O, and "the event stream is the only upward contract" — none of them is envelope-specific. If AG-04's own package doc comment needs a fourth clause describing the envelope, `doc_contract_guard_test.go`'s substitution mechanism (`doc_contract_guard_test.go:4-22`) already resolves *how* to add it. Whether AG-04 *needs* a new row at all is undecided — a design-phase call, not invented here.
4. **Whether AG-04's tests touch `agenttest` at all.** Per the charter's own words ("no producer exists until wave 2... assertable over hand-built sequences," `0003:417`) and AG-04's dependency edges (`AG-01`, `AG-03` only — not AI-21/AI-22, `0003:419`), the answer is very likely "no, AG-04 constructs events directly through its own public surface" — but this is inferred from dependency-graph absence, not from an explicit statement anywhere, so it should be confirmed rather than assumed once spec/design begins.

## 9. Affected areas (not yet created)

- `backend/agent/src/agent/` — new production `.go` files (envelope, kind derivation, ordering, run/turn lifecycle payloads, typed-failure surface, stream-contract validator) — the package's first production code.
- `backend/agent/src/agent/doc_contract_guard_test.go` and `doc.go` — possibly touched if a guarded paragraph is added (open decision 3 above); both AG-03 guards (`import_boundary_test.go`, `ambient_authority_test.go`) must stay green with zero changes needed to their own logic (AG-04 imports only `src/ai` + stdlib, already allowed).
- New `_test.go` files carrying AG-04's 11 Gherkin scenarios plus AG-04.4's witness-table guard.

## Risks

1. **Sizing risk is high** — section 7 above; plausibly 1.4-2.2x AG-03's already-over-ceiling 1005 lines.
2. **Scope creep into AG-05/AG-06's families** is the most likely correctness defect (section 6) — the guard must iterate exactly AG-04's own registered kinds, structurally ready for later extension, never a placeholder for message/tool/permission/cost/delegation/compaction kinds.
3. **The stream-contract validator's design fork** is unresolved and load-bearing for AG-05, AG-06, and AG-23 — a wrong early choice (e.g. building it non-reusably, inline in a test file) would need rework at AG-23.
4. **Invariant coverage is partial at AG-04** (section 5) — AG-04.3 pins only invariants 1 and 4, not 2 (full nesting, AG-19.1) or 3 (observer asynchrony, AG-01.1+AG-20.2). A guard or test overreaching into those is a scope/acceptance-criterion mismatch.
