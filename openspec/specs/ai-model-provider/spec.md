# Spec — the model provider interface

> **Milestone**: AI-20 — Define the provider interface (Wave 2 "Stream", the wave's **join point**) · **Nodes**: AI-20.1 `[leaf]` the interface, AI-20.2 `[leaf]` pre-stream contract, AI-20.3 `[leaf]` mid-stream contract, AI-20.4 `[guard]` signature guard, AI-20.5 `[leaf]` optional capabilities
> **Introduced by**: `openspec/changes/archive/2026-08-01-cachicamas-ai-model-provider/`, merged to `main` in PR #104 (commit `37898c7`)
> **Status**: **live** — this file carries the contract; later milestones cite it and amend it
> **Project**: cachicamas (witsaba) · **Target package**: `backend/agent/src/ai/` (Layer 1)
> **Closes**: **G13** — the stream carrier is pinned at the one boundary where it matters (AI-02.1 decided it; AI-20.4 makes it mechanical). It is also the mechanism doc 0002 records as meeting **G3**, optional token counting discovered by assertion (AI-03.1, AI-20.5), and it closes completion-checklist item 10 — no vendor type on the boundary, optional capabilities discovered rather than required (AI-20.1, AI-20.4, AI-20.5). Cancellation leak-freedom is AI-33; backpressure sizing is AI-34
> **Format**: RFC 2119 keywords + Given/When/Then, per `openspec/config.yaml`
> **Requirement IDs**: `R-AMP-0NN` · **Scenario IDs**: `S-AMP-0NN`
> **Binding predecessors, cited by identifier and never modified**:
> [`ai-stream-lifecycle`](../ai-stream-lifecycle/spec.md) — AI-02.1 § 3 carrier, § 4 ownership, § 5 cancellation, § 6 buffering, § 7 failure delivery, **§ 9 the eight statements**, § 10 "AI-20 — the provider interface";
> [`ai-minimum-capabilities`](../ai-minimum-capabilities/spec.md) — AI-03.1 § 9 discovery, § 12 "AI-20.5";
> [`ai-contract-vocabulary`](../ai-contract-vocabulary/spec.md) — `V-PRV-01` … `V-PRV-08`, `V-PRV-16`, `V-PRV-17`, `V-STR-04` … `V-STR-09`, `V-FAIL-10` … `V-FAIL-12`, `V-REQ-22`;
> [`ai-model-request`](../ai-model-request/spec.md) and [`ai-validation-errors`](../ai-validation-errors/spec.md) — the request and the failure value on the pre-stream path;
> [`ai-event-envelope`](../ai-event-envelope/spec.md) (AI-14) — the carrier's element type;
> [`ai-provider-errors`](../ai-provider-errors/spec.md) (AI-19) — the terminal error event and the cancellation category
> **Depends on**: AI-03, AI-10, AI-12, AI-14, AI-19 · **Blocks**: AI-21 and everything after it
> **Sources**: [doc 0002 — Layer 1 task graph](../../../docs/architecture/milestones/0002-cachicamas-ai-layer-1-task-graph.md#ai-20--define-the-provider-interface) §§ AI-20.1 … AI-20.5 (lines 1099–1161) · [doc 0001 — agent stack v2](../../../docs/architecture/0001-cachicamas-agent-stack-v2.md) § 2.2, § 4.3 invariant 4, § 9 · [ADR 0004](../../../docs/adr/0004-adopt-tau-3-layer-agentic-architecture.md) · [ADR 0005 § D2 / Guard C](../../../docs/adr/0005-promote-agent-stack-to-own-module.md)

---

## Purpose

Layer 1 has a complete vocabulary and no outward face. This spec constrains the **one call** every provider (`V-PRV-01`) offers — a normalized request in, a normalized event stream out — and the two contracts that hang off it: what holds **before** the carrier is handed over (`V-PRV-04`), and what holds **after** (`V-PRV-05`). It adds the mechanical pin that keeps a vendor type off the boundary, and it restates AI-03.1's discovery mechanism as an obligation on this surface.

Everything this spec declares was decided upstream. Its work is **spelling already-shipped decisions into one declaration and pinning them mechanically**, so that no downstream milestone re-derives them differently. Where an upstream decision is cited, it is cited — not re-argued and not adjusted.

## Status — this file is the canonical home of the contract

`openspec/AGENTS.md` calls `openspec/specs/` the *source of truth (main specs)*. The one-method interface, the pre-stream and mid-stream contracts, the signature guard's obligations and the optional-capability discovery rule therefore live here, in their own text, and not only as a pointer into the archive. The archived change folder at [`openspec/changes/archive/2026-08-01-cachicamas-ai-model-provider/`](../../changes/archive/2026-08-01-cachicamas-ai-model-provider/) is the historical record of how AI-20 was explored, proposed, designed, applied and verified.

**This file is Layer 1's outward face, and it is the one most exposed to erosion by a later adapter.** Three requirements carry that weight and are pinned mechanically rather than by convention: `R-AMP-002` (no vendor or wire type on the boundary, enforced by an AST walk with an import allowlist of exactly `{"context"}`), `R-AMP-014` … `R-AMP-016` (the guard itself, proven to bite twice), and `R-AMP-021` (the core method set never widens). Adding a second method to the provider contract, or admitting a second import to the declaring file, fails a test rather than a review.

**`NFR-AMP-C` binds every later milestone, not only AI-20.** A change that needs different stream-lifecycle or capability behaviour amends [`ai-stream-lifecycle`](../ai-stream-lifecycle/spec.md) or [`ai-minimum-capabilities`](../ai-minimum-capabilities/spec.md) in the pull request that needs it. It does not make a local judgement here.

## Definitions used by this spec

- **The boundary** — the declared provider interface (`V-PRV-03`): its method set, and every type appearing in that method set's parameters and results.
- **The carrier** — AI-02.1 § 3's decided stream carrier: a **receive-only channel** of AI-14's event envelope.
- **Handover** — the moment the streaming call returns a carrier to the caller. AI-02.1 § 7 makes this, and **not the first event**, the boundary between the two delivery paths.
- **Pre-stream** (`V-FAIL-11`) — before handover. **Mid-stream** (`V-FAIL-12`) — after handover, including a stream that fails having emitted no content.
- **A vendor or wire type** (`V-PRV-15`) — any type declared by, or structurally mirroring, one vendor's own request/response shape, and any type from a vendor SDK package.
- **The required surface** — AI-03.1 § 5's five required capabilities, `CAP-R-01` … `CAP-R-05`.

---

## AI-20.1 — The interface

### R-AMP-001 — Exactly one streaming method, with the decided in and out shapes

The provider interface MUST declare **exactly one** method. It MUST take a cancellable context and a normalized `Request` (`ai-model-request`), and it MUST return the AI-02.1 § 3 carrier together with an `error`. The interface MUST NOT declare a second streaming variant, a non-streaming convenience method, a batching method, or any capability-query method.

#### Scenarios

- **S-AMP-001** — Given the landed interface declaration, when a reader enumerates its method set, then it contains exactly one method, and that method takes a context and a normalized request and returns the carrier plus an error.
- **S-AMP-002** — Given the landed interface, when a reader looks for a second entry point — a non-streaming call, a batch call, or a "supports X" query — then none exists.
- **S-AMP-003** — Given the returned carrier, when a consumer inspects its direction, then it is receive-only and the consumer has no means of sending on it or closing it.

### R-AMP-002 — No vendor type and no wire type appears on the boundary

No vendor type and no wire type (`V-PRV-15`) MUST appear on the boundary, in either direction, whether as a parameter, a result, an element of the carrier, or a field reachable from a type named in the signature. The declaring file MUST import no vendor package. This is `V-PRV-03` stated as an obligation on this milestone, and doc 0001 § 9's rule — *"No vendor wire type crosses the Layer 1 method boundary"* — restated where it is enforced.

#### Scenarios

- **S-AMP-004** — Given the file declaring the interface, when its imports are enumerated, then every one is stdlib or an in-module Layer 1 package, and no vendor SDK or transport package appears.
- **S-AMP-005** — Given each type named in the method's parameters and results, when a reviewer traces it to its declaration, then each is declared inside Layer 1 or the standard library.
- **S-AMP-006** — Given the module's dependency graph, when `go.mod` is read, then it still declares zero requires, so no vendor type is even reachable.

### R-AMP-003 — The interface is implementable from outside package `ai`

A package that is not `ai` MUST be able to implement the interface, compile against it, and be exercised through it. The interface MUST NOT contain an unexported method, an embedded unexported interface, or any other construct that confines implementation to its declaring package. Every type the implementer must name MUST be exported and constructible from outside.

#### Scenarios

- **S-AMP-007** — Given a stub type declared in `src/agenttest/` (an external package), when it declares the streaming method and is assigned to the interface, then it compiles and satisfies the interface.
- **S-AMP-008** — Given that stub, when a test calls it through the interface and drains the returned carrier, then it is exercised end-to-end without the test importing anything unexported from `ai`.
- **S-AMP-009** — Given the landed interface, when a reader looks for an unexported method or an embedded unexported interface, then none exists.

### R-AMP-004 — The interface documentation carries the eight ownership statements and the enumerability clause

The interface's documentation MUST state, in substance, all **eight** statements of AI-02.1 § 9 — the producer creates and closes exactly once; the caller owns the cancellable context; a consumer ends a stream by draining to close or by cancelling; abandoning without cancelling is a contract violation rather than a supported mode; cancellation closes within bounded time; the buffer is bounded and backpressure means waiting rather than dropping; the single sanctioned loss path with its consumer-is-in-error clause; and the two delivery paths split at handover. It MUST additionally state AI-03.1 § 9's clause that optional capabilities are **separately asserted and enumerable**, so a consumer knows where to look. These are prose obligations, not spellings, and they MUST NOT be weakened, merged away, or replaced by a link alone.

#### Scenarios

- **S-AMP-010** — Given the landed documentation, when a reviewer walks AI-02.1 § 9's eight statements in order, then each is present in substance and none is contradicted.
- **S-AMP-011** — Given statement 4, when a reader looks for the abandonment sentence, then it says abandoning without cancelling is a contract violation and names cancellation as the alternative for a consumer that will not drain.
- **S-AMP-012** — Given the documentation, when a consumer looks for how to learn whether a provider offers an optional capability, then it states that optional capabilities are separately asserted contracts and that they are enumerable.

> **Carried forward.** All nine statements are present in the landed `ModelProvider` GoDoc, verified against the source specs at wave verification. But the only evidence they are present is that a reader looked: deleting one — for example rule 6, "the stream's buffer is bounded; backpressure means waiting, never dropping" — makes no test fail. The package already contains the exact tool for closing this (`sequence_guard_test.go` and `TestSequenceGoFile_PackageDoc_StatesTheCrossStreamRule` both parse Go doc comments and assert on their content). An eight-substring assertion over this GoDoc would make `R-AMP-004` mechanical. Recorded as `S5` in the Wave 2 verify report.

---

## AI-20.2 — Pre-stream contract (`V-PRV-04`)

### R-AMP-005 — An invalid request fails before any stream or producer exists

WHEN the request is invalid, the call MUST fail on the pre-stream path (`V-FAIL-11`): it MUST return AI-04's typed failure value, it MUST NOT return a usable carrier, and **no producer goroutine MUST be started**. The failure MUST NOT be delivered as an empty stream that closes, nor as a stream that immediately yields a failure — both are excluded by AI-02.1 § 7.

#### Scenarios

- **S-AMP-013** — Given an invalid request, when the streaming call is made, then it returns a non-nil AI-04 failure, `errors.Is` matches a landed sentinel, and the returned carrier is unusable as a stream.
- **S-AMP-014** — Given the same call under `-race` with a goroutine count taken immediately before and after, when the call returns its failure, then no additional goroutine is live.
- **S-AMP-015** — Given an invalid request, when a consumer attempts to drain whatever the call returned, then it receives no event and observes no stream to close — the failure is available only from the returned value.

### R-AMP-006 — Validation runs exactly once, before any I/O, and before the cancellation case

The single validation pass (`V-REQ-22`) MUST run **once**, before any I/O and before any stream allocation. Validation MUST be ordered **before** the already-cancelled-context case, and this order MUST be documented on the interface. A request that was **never constructed** — the zero value — MUST be rejected on this path rather than treated as an empty-but-valid request; the observable means by which that is decided is design's, but the rejection is required here.

#### Scenarios

- **S-AMP-016** — Given a request that is invalid AND a context that is already cancelled, when the streaming call is made, then the failure reported is the **validation** failure, not the cancellation category.
- **S-AMP-017** — Given a zero-value `Request` and a live context, when the streaming call is made, then it fails on the pre-stream path and the failure names the never-constructed request rather than reporting a successful empty call.
- **S-AMP-018** — Given the landed documentation, when a reader looks for the ordering of validation relative to an already-cancelled context, then it is stated explicitly.

### R-AMP-007 — An already-cancelled context is a pre-stream failure carrying the cancellation category

WHEN the request is **valid** and the context is already cancelled at call time, the call MUST fail on the same pre-stream path and MUST report the category AI-19 assigns to cancellation. It MUST NOT hand over a carrier that immediately closes, and it MUST NOT report a validation failure.

#### Scenarios

- **S-AMP-019** — Given a valid request and a context cancelled before the call, when the streaming call is made, then it returns a failure whose category is AI-19's cancellation category and returns no usable carrier.
- **S-AMP-020** — Given that same call, when goroutines are counted before and after under `-race`, then none was started.
- **S-AMP-021** — Given a caller that only ever inspects the returned failure, when it classifies both the invalid-request case and the already-cancelled case, then it classifies each from one vocabulary without consulting a stream.

### R-AMP-008 — Nothing observable happens before validation passes

Before validation passes there MUST be no observable effect: no I/O, no stream allocation, no goroutine, and no partial state left behind by a failed call. A failed pre-stream call MUST leave the provider value usable for a subsequent valid call.

#### Scenarios

- **S-AMP-022** — Given a provider whose test producer records every side effect it performs, when a call with an invalid request returns, then the recorder shows no I/O attempt and no stream allocation.
- **S-AMP-023** — Given a provider that has just rejected an invalid request, when a valid request is then submitted to the same value, then it succeeds and streams normally.

---

## AI-20.3 — Mid-stream contract (`V-PRV-05`)

### R-AMP-009 — One sender, one closing site, closed exactly once on every path

Once a carrier is handed over, exactly **one** goroutine MUST send on it, and the producer MUST contain exactly **one** closing site — one total, not one per path. That site MUST run on every exit path: completion, terminal error, cancellation, and an unwinding exit. It MUST run **after the last send attempt and never before**. Nothing but the producer MUST close the stream (`V-STR-04`, AI-02.1 § 4).

#### Scenarios

- **S-AMP-024** — Given a producer driven to normal completion, when the consumer drains to close, then the stream closes exactly once and a second receive reports a closed stream rather than panicking.
- **S-AMP-025** — Given the same producer driven to a terminal error, and separately to cancellation, when each run finishes under `-race`, then the stream closes exactly once on each path and no double-close occurs.
- **S-AMP-026** — Given the landed producer used by this milestone's tests, when a reviewer counts its closing sites and its sending goroutines, then there is exactly one of each.

### R-AMP-010 — Every send selects on cancellation; no send is unconditional

Every send the producer performs MUST wait on both the stream and cancellation. **No send MUST be unconditional**, including the terminal event. A producer whose consumer stops reading MUST therefore still exit when the context ends.

#### Scenarios

- **S-AMP-027** — Given a producer whose consumer stops reading after one event, when the caller cancels the context, then the producer exits and the stream closes, rather than blocking forever on a send.
- **S-AMP-028** — Given the producer's source, when a reviewer enumerates its sends, then each waits on cancellation as well as on the stream, and the terminal send is not an exception.

### R-AMP-011 — Cancellation closes within bounded time, with no send after close

Cancellation MUST close the stream within bounded time. "Bounded" is defined by exclusion (AI-02.1 § 5): once cancellation is observable the producer begins **no new blocking wait** on a transport and **no new blocking wait** on the consumer, and a backoff waits on the signal rather than sleeping. The whole property MUST hold under `-race`, and no send MUST occur after the close.

#### Scenarios

- **S-AMP-029** — Given a mid-stream cancellation and a test deadline, when the test waits for the stream to close under `-race`, then it closes before the deadline and the race detector reports nothing.
- **S-AMP-030** — Given the same run, when the test asserts that no send happened after the close, then the assertion holds — which follows structurally from `R-AMP-009`'s single sender whose final act is the close.
- **S-AMP-031** — Given a producer that would otherwise sleep between events, when cancellation arrives during that interval, then the producer observes the signal instead of finishing the sleep.

### R-AMP-012 — Exactly one loss path is sanctioned, and the contract says who is in error

On **cancellation** with a **saturated** buffer, late events MUST be dropped and the stream MUST close **without a terminal event**. The contract MUST state that a consumer treating a missing terminal **after its own cancellation** as corruption is the party in error. A stream that closes without a terminal event and was **never cancelled** MUST be treated as a producer defect — it is not a second loss path. Every other path MUST be lossless: backpressure means waiting, never dropping (`V-STR-23`).

#### Scenarios

- **S-AMP-032** — Given a producer emitting faster than an unread consumer drains until the buffer is saturated, when the caller cancels, then the stream closes with no terminal event delivered and the close is still bounded.
- **S-AMP-033** — Given the landed documentation, when a consumer author looks up a missing terminal after its own cancellation, then the contract states this is sanctioned and that the consumer, not the producer, carries the burden.
- **S-AMP-034** — Given a slow consumer that pauses and resumes without cancelling, when it eventually drains to close, then it receives every event in order and none was dropped.
- **S-AMP-035** — Given a stream that closes bare with no cancellation ever signalled, when the case is classified against this spec, then it is a producer defect, not a sanctioned loss.

### R-AMP-013 — This milestone proves the mid-stream contract with its own single-purpose producer

Because AI-21's scripted fake and AI-22's test kit are blocked **by** this change, AI-20.3 MUST prove these properties with a **single-purpose producer local to this milestone's tests**. That producer MUST NOT be exported, MUST NOT be presented as a reusable double, and MUST NOT pre-empt AI-21's design. Its exact shape is design's.

#### Scenarios

- **S-AMP-036** — Given the tests proving `R-AMP-009` … `R-AMP-012`, when a reader looks for the producer they drive, then it is defined in this milestone's test files and is not exported from the contract package.
- **S-AMP-037** — Given the landed package surface, when a reviewer looks for a reusable fake provider or a stream test kit, then neither exists — both remain AI-21's and AI-22's.

---

## AI-20.4 — Signature guard `[guard]`

### R-AMP-014 — A guard in `src/agenttest/` pins the boundary mechanically

A guard MUST live in `src/agenttest/` and MUST fail when the boundary changes: it MUST assert that the streaming method's parameter and result types are the neutral ones and the AI-02.1 § 3 carrier, and that **no imported vendor package appears in the interface's declaration**. This spec requires the guard and its verdict; **the inspection mechanism is design's** and is not constrained here beyond being automatic and part of `make test`.

#### Scenarios

- **S-AMP-038** — Given the unmodified boundary, when the guard runs as part of `go test -race ./...`, then it passes without manual steps.
- **S-AMP-039** — Given the guard, when it inspects the interface declaration, then its verdict names the method, its parameter types, its result types and the imports of the declaring file.

### R-AMP-015 — The guard resolves the Layer 1 source relative to its own source file and fails loudly

The guard MUST locate the Layer 1 source relative to **its own source file**, not from a working directory, an environment value, or a hard-coded absolute path — the sibling-layout dependency ADR 0005 Guard C names, here made explicit. If it cannot resolve or parse its target, it MUST **fail**; it MUST NOT skip, MUST NOT pass vacuously, and MUST NOT report success on an empty inspection.

#### Scenarios

- **S-AMP-040** — Given the guard run from a different working directory than the package directory, when it resolves its target, then it finds the same Layer 1 source and reaches the same verdict.
- **S-AMP-041** — Given a target path that cannot be resolved or parsed, when the guard runs, then it fails and its message names the path it tried, rather than reporting a pass.
- **S-AMP-042** — Given the sibling-layout dependency, when a reader looks for it, then it is recorded in `agenttest` documentation so that a future move of `src/ai` or `src/agenttest` is a loud failure rather than a silent one.

### R-AMP-016 — The guard is proven to bite, with two mutations

The guard MUST be proven to bite before it is trusted. **Two** scratch mutations MUST each be applied, observed to fail the guard, recorded, and reverted: (1) introducing a vendor type into the signature, and (2) changing the carrier. A guard whose bite is not recorded MUST be treated as unproven.

#### Scenarios

- **S-AMP-043** — Given a scratch mutation adding a vendor type to the streaming method, when the guard runs, then it fails and its message names the offending type; the mutation is reverted and the guard passes again.
- **S-AMP-044** — Given a scratch mutation changing the carrier — its direction or its element type — when the guard runs, then it fails and names the changed result; the mutation is reverted and the guard passes again.
- **S-AMP-045** — Given `tasks.md`, when a reviewer looks for the bite evidence, then both mutations carry recorded failing output and a recorded revert.

---

## AI-20.5 — Optional capabilities are discovered, not required

### R-AMP-017 — One separate contract per optional capability; only token counting is askable in v1

An optional capability (`V-PRV-07`) MUST be declared as an **additional contract beside** the provider contract, **one per capability**, asserted on the provider value at the point of use (AI-03.1 § 9). In v1 **only `CAP-O-02` token counting** (`V-PRV-17`) is askable and therefore gets a contract. `CAP-O-01` reasoning content and `CAP-O-03` honoring cache-boundary markers are **receive-side** — observed on the stream, never discovered — and MUST NOT get a discovery contract. A single aggregate optional contract MUST NOT be declared.

#### Scenarios

- **S-AMP-046** — Given the landed surface, when a reader enumerates the optional contracts, then there is exactly one, and it is token counting.
- **S-AMP-047** — Given the landed surface, when a reader looks for a discovery contract for reasoning content or for cache-marker honoring, then none exists, and the documentation states that both are observed rather than asked.
- **S-AMP-048** — Given the landed surface, when a reader looks for one contract carrying several optional behaviors, then none exists.

### R-AMP-018 — Discovery is asked of the provider value, and absence is clean

A consumer MUST discover an optional capability by asking **the provider value** whether it also satisfies the optional contract. It MUST NOT be asked of a model identity (`V-REQ-21`), a configuration entry, or a provider catalog. The result MUST be either the capability, usable immediately, or a **clean absence**: **not an error and not a zero**. Absence MUST NOT be reported as a validation or provider failure, MUST NOT be the *unsupported capability* category, and Layer 1 MUST NOT supply a substitute — no default implementation, no estimate, no synthesised figure.

#### Scenarios

- **S-AMP-049** — Given a provider value that advertises token counting, when a consumer asks it, then it obtains the capability and uses it immediately.
- **S-AMP-050** — Given a provider value that does not advertise it, when a consumer asks, then it observes an absence that is neither an error value nor a zero count, and no failure sentinel is produced.
- **S-AMP-051** — Given that absent case, when a consumer looks for a Layer 1 fallback figure, then none exists and the consumer owns its own fallback.
- **S-AMP-052** — Given the landed surface, when a reader looks for a provider-declared capability list, a configuration table, or a catalog lookup, then none exists.

### R-AMP-019 — Advertising binds, and it is advertised whole

A provider that satisfies the optional contract and then declines to answer MUST be treated as **non-conformant**, not absent. An advertised capability MUST be advertised whole — no partial satisfaction. Absence, by contrast, MUST cost an adapter zero lines: an adapter that does not offer the capability declares **nothing at all** — no field, no negative answer, no unsupported entry.

#### Scenarios

- **S-AMP-053** — Given a provider that satisfies the token-counting contract but refuses from inside it, when it is judged against this spec, then it is non-conformant rather than a clean absence.
- **S-AMP-054** — Given a provider that offers nothing optional, when its declaration is read, then it carries no negative marker of any kind, and it is still fully conformant.

### R-AMP-020 — A provider implementing only the required surface is fully conformant

A provider that implements only the required surface — `CAP-R-01` … `CAP-R-05` — and advertises nothing optional MUST be **fully conformant**. Conformance MUST be assertable by exercising the whole required path against such a value.

#### Scenarios

- **S-AMP-055** — Given the external-package stub advertising nothing optional, when the required path is exercised against it end-to-end, then every required obligation this spec states holds and no conformance gap is reported.
- **S-AMP-056** — Given that same value, when a consumer asks it for token counting, then it observes a clean absence and this does not make it non-conformant.

### R-AMP-021 — *(pin)* The core interface never widens

A guard-style assertion over the interface's **method set** MUST fail if an optional capability is folded into the core interface. The core provider contract MUST NOT acquire an optional behavior every adapter must implement or refuse from inside. This pin is the mechanical form of AI-03.1 § 9's prohibition, and it MUST be automatic rather than a review convention.

#### Scenarios

- **S-AMP-057** — Given the unmodified interface, when the method-set assertion runs, then it passes and reports the exact expected method set.
- **S-AMP-058** — Given a scratch mutation folding token counting into the core interface as a second method, when the assertion runs, then it fails and names the added method; the mutation is reverted.
- **S-AMP-059** — Given the pin, when a reviewer asks what it protects against, then the documentation names AI-03.1 § 9's four rejected alternatives — a widened core contract, a provider-returned capability list, a configuration-driven table, and a single aggregate optional contract.

---

## Non-functional requirements

### NFR-AMP-A — Dependency purity

The change MUST add no module dependency. `backend/agent/go.mod` MUST still carry zero requires, and both AI-00 import guards MUST still pass. Layer 1 MUST read no configuration.

- **S-AMP-060** — Given the change merged, when `go.mod` is read, then it declares no require; and when both import guards run, then both pass.

### NFR-AMP-B — Failure reporting

Every pre-stream rule violation in this spec MUST be reported through AI-04's existing failure value and its landed sentinels; every mid-stream failure MUST arrive as AI-19's single terminal error event and by no other route. **No new sentinel and no second failure type MUST be introduced by this change.**

- **S-AMP-061** — Given each pre-stream rejecting scenario above, when its failure is inspected, then it is AI-04's failure value and `errors.Is` matches a landed sentinel.
- **S-AMP-062** — Given a mid-stream failure, when a consumer looks for a second route — a re-inspected return value, a side channel, an accessor — then none exists.

### NFR-AMP-C — No upstream decision is reopened

This change MUST NOT reopen AI-02.1 or AI-03.1. Any needed change to either is an **amendment to that spec, in the pull request that needs it** — never a local judgement here. AI-14's event kinds and payloads, AI-19's categories, AI-34's buffer sizing and AI-35's retry policy MUST remain out of scope.

- **S-AMP-063** — Given the merged diff, when a reviewer looks for an edit to `ai-stream-lifecycle` or `ai-minimum-capabilities`, then either none exists, or it is an explicit amendment carrying its own justification.
- **S-AMP-064** — Given the landed surface, when a reviewer looks for a buffer-size decision, a retry policy, an event kind, or a failure category introduced here, then none exists.

### NFR-AMP-D — Totality

No exported function or method of this contract MUST panic for any input, including a zero-value request, a nil-or-cancelled context, and a provider value that advertises nothing optional.

- **S-AMP-065** — Given a table of those extreme inputs, when each is passed through every exported entry point of this contract, then none panics.

### NFR-AMP-E — Evidence

Every test-list item of AI-20.1 … AI-20.5 MUST be taken red → green → refactored **in order**, with both outputs recorded in `tasks.md`. The milestone MUST close on recorded green `make test` (`go test -race -v ./...`) in `backend/agent/` and clean `make lint`.

- **S-AMP-066** — Given `tasks.md`, when a reviewer walks the milestone's test-list items, then each carries recorded red output, recorded green output, and a refactor note, and the guard bite evidence of `R-AMP-016` is among them.

---

## Acceptance criteria

1. One streaming method takes a context and a normalized request and returns the AI-02.1 carrier plus an error, with no vendor or wire type anywhere in the signature and no second method on the interface.
2. A package that is not `ai` implements the interface, compiles, and is exercised through it.
3. The interface documentation carries all eight AI-02.1 § 9 ownership statements plus AI-03.1's enumerability clause.
4. An invalid request fails with a typed AI-04 error before any carrier or goroutine exists; a zero-value request is rejected on the same path.
5. An already-cancelled context reports AI-19's cancellation category on the pre-stream path, **after** validation, in the documented order.
6. The producer closes exactly once on completion, terminal error and cancellation; every send waits on cancellation; cancellation closes within bounded time under `-race` with no send after close.
7. The saturated-buffer cancellation path closes bare exactly as sanctioned, the contract names the consumer as the party in error, and no second loss path exists.
8. The signature guard passes automatically, resolves its target relative to its own source file, fails loudly when it cannot, and both bite mutations — a vendor type, a changed carrier — fail it; recorded and reverted.
9. Token counting is the only askable optional capability; a consumer discovers it where advertised and observes a clean absence otherwise, with no Layer 1 substitute.
10. A provider implementing only the required surface is fully conformant, and the method-set pin fails if an optional capability is folded into the core interface.
11. `make test` green under `-race`, `make lint` clean, both import guards passing, `go.mod` still zero requires.

## What this file deliberately leaves to Wave 3 and beyond

AI-20 declares the mid-stream contract and proves it with a single-purpose producer local to its own tests (`R-AMP-013`). It does **not** ship a reusable double or a stream test kit: those are AI-21 and AI-22, which this milestone unblocks. Cancellation leak-freedom under adversarial abandonment is AI-33; buffer sizing and backpressure limits are AI-34; retry and idempotency policy is AI-35. Each of those cites this file rather than restating it.
