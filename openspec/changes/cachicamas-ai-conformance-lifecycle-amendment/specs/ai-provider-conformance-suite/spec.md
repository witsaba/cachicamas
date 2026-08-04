# Delta for `ai-provider-conformance-suite` — the mandated response lifecycle

> **Change**: `cachicamas-ai-conformance-lifecycle-amendment` · **Amends**: [`openspec/specs/ai-provider-conformance-suite/spec.md`](../../../../specs/ai-provider-conformance-suite/spec.md)
> **Amended requirements (8, all MODIFIED)**: `R-CNF-005`, `R-CNF-006`, `R-CNF-007`, `R-CNF-008`, `R-CNF-009`, `R-CNF-011`, `R-CNF-014`, `R-CNF-016`
> **New requirements (6, ADDED)**: `R-CNF-019` (count derivation), `R-CNF-020` (durable negative), `R-CNF-021` (tolerance register), `R-CNF-022` (neutrality), `R-CNF-023` (capability-scoped entry point), `R-CNF-024` (script introspection) · **Removed / renamed**: none
> **Requirement total**: **14** (8 MODIFIED + 6 ADDED).
> **Scenario counts** — new scenarios `S-CLA-001 … S-CLA-035` = **35** (**22** `[test]`, **13** `[inspection]`; corrected 2026-08-04 per verify finding S4 — the original 23/12 header miscounted the split, the per-scenario markings were always authoritative); retained/edited predecessor scenarios carrying their original `S-CNF-0NN` ids = **24**. Every row of the proposal's 17-row scope table maps to at least one `S-CLA` scenario.
> **Sources**: proposal `openspec/changes/cachicamas-ai-conformance-lifecycle-amendment/proposal.md` · doc 0002 § AI-28.1.1 · `src/ai/response_start.go` (`ai.NewResponseStart`, `ResponseStart.ResponseID()`, `ResponseStart.ServedModel()`)
> This delta **strengthens and never contradicts** the shipped contract. No `src/ai` change, no `openaicompat` change: `NFR-CNF-A` … `NFR-CNF-F` hold unmodified and are not restated here.

> **Revision note (scope extension, 2026-08-04).** Coordinator ruling from AI-28's design validation (fresh-context validator, findings **C-2** and **C-3**), verified against landed code: `agenttest.Script`/`Step` expose **zero** methods and only unexported fields, so no external package can read a scripted `ai.Event` back out and AI-28's Script → SSE transcript bridge is unbuildable; and `RunConformance` iterates the whole 17-case registry with required capabilities non-waivable (`runRegisteredCase` errors on any skip of a non-optional capability) while `runConformanceCases` is unexported, so a text-only adapter cannot legally run just the text cases — which AI-28's acceptance ("the conformance text case passes against real transport") requires. This revision adds `R-CNF-023` and `R-CNF-024`, cited by ID from AI-28's corrected spec, and widens `R-CNF-022`'s neutrality statement to admit their surface. Requirement total 12 → **14**; new scenarios `S-CLA-029` … `S-CLA-035`. Nothing else in this delta changed.

---

## Definitions added by this amendment

- **A started response** — an interaction whose script models a provider that began producing output and therefore returns a stream carrier. Every stream-bearing conformance script is a started response. The pre-stream-failure path is **not** one: it returns no carrier at all, so no lifecycle event can precede it.
- **The lifecycle prefix** — the single `ai.EventKindResponseStart` event that a started response MUST carry as its first drained event, before any content event and before any terminal.
- **Scripted identity** — the `(responseID, servedModel)` pair the case's own script passed to `ai.NewResponseStart`.
- **A full conformant lifecycle stream** — a started response that opens with the lifecycle prefix, carries its content events, and ends on exactly one terminal a real producer would actually emit: a `Completion` on a clean transcript, or the case's own deliberate failure terminal where that is the case's charter. A stream cancelled before it reaches a terminal closes bare and carries none, per AI-20.3.

## MODIFIED Requirements

### Requirement: R-CNF-005 — Text arrives ordered, contiguous, and reconstructs exactly, behind the lifecycle prefix

The suite MUST assert that a text interaction produces the lifecycle prefix → block start → deltas → block end → terminal completion in that order, that sequence numbers run 1..N contiguously with no gap and no repeat, and that concatenating the deltas reconstructs the source text **byte-exactly**. Reconstruction MUST be proven with a multi-byte rune deliberately split across two deltas, so a subject that reassembles per-rune rather than per-byte fails. Ordering MUST be asserted by delegating to the shipped checker and contiguity to AI-22's `R-STK-006` helper; neither MUST be reimplemented.

The script MUST be a full conformant lifecycle stream, so the drained window MUST be asserted as an **exact** count and as a **positional** kind list derived from it (`R-CNF-019`) — exact counts are what caught this defect class and MUST NOT be relaxed into presence or subsequence assertions. The event at **position 0** MUST be the lifecycle prefix, and the suite MUST assert that it carries the scripted identity: `ResponseID()` and `ServedModel()` **equal** the scripted values, not merely non-empty.
(Previously: the drained window was four events — block start, two deltas, block end — with neither a lifecycle event nor a terminal asserted anywhere, and no identity obligation.)

#### Scenarios

- **S-CNF-012** — Given a scripted text interaction, when the suite drains and asserts it, then kinds appear in response-start/block-start/delta/block-end/completion order and sequences run 1..N with no gap.
- **S-CNF-013** — Given a text whose multi-byte rune is split across two adjacent deltas, when the deltas are concatenated, then the result equals the source byte-for-byte.
- **S-CNF-014** — Given a subject whose deltas arrive reordered, when the suite runs, then the case fails carrying the shipped checker's own verdict.
- **S-CLA-001** `[test]` — Given the `text/order_contiguity_byte_exact_reconstruction` script made a full conformant lifecycle stream — response start, text block start, two text deltas, text block end, completion — when it is drained, then **exactly six** events arrive and their kinds match that list positionally. (Six is read off the script, not incremented from the shipped four, which lacked **both** the prefix and the terminal.)
- **S-CLA-002** `[test]` — Given that same drained window, when the event at position 0 is read as a response start, then it carries one, and its `ResponseID()` and `ServedModel()` are **equal** to the values the script supplied.

### Requirement: R-CNF-006 — An empty completion is legal; the no-minimum rule binds text content, not the lifecycle

The suite MUST assert that a normally-finished interaction producing **no** text content is conformant: it terminates normally, carries its terminal event, and MUST NOT be reported as a failure, as an incomplete stream, or as a contiguity violation.

The suite MUST NOT require any minimum count of **text-content** events — no minimum number of text blocks, text deltas or content bytes — and MUST NOT infer a defect from their total absence. That prohibition is scoped to text content **only**. It MUST NOT be read as forbidding the suite from requiring the lifecycle prefix, because the prefix is not text content: an empty completion still **is** a started response, so it MUST carry the prefix followed by its terminal and nothing else. The empty-completion case MUST assert that exact two-event window, and MUST assert the prefix at position 0 with `ResponseID()` and `ServedModel()` **equal** to the scripted identity.
(Previously: the case asserted a one-event window and read `events[0]` as the completion, and the no-minimum sentence was unscoped, so it could be read as forbidding any mandated event.)

#### Scenarios

- **S-CNF-015** — Given a subject that finishes normally having emitted no text delta, when the suite runs the empty-completion case, then it passes and the terminal is present.
- **S-CNF-016** — Given the same interaction, when the contiguity assertion runs over its recording, then it reports no violation for the absent text block.
- **S-CLA-003** `[test]` — Given the `text/empty_completion_is_legal` script made a full conformant lifecycle stream — response start then completion — when it is drained, then **exactly two** events arrive: the response start at position 0 and the completion at position 1, with no text-content event of any kind between them. (The shipped script already carried its terminal, so two is the whole derived window.)
- **S-CLA-004** `[test]` — Given that drained window, when position 0 is read as a response start, then its `ResponseID()` and `ServedModel()` are **equal** to the scripted values.
- **S-CLA-005** `[inspection]` — Given a reader asking whether mandating the lifecycle prefix contradicts this requirement's no-minimum rule, when this requirement's text is read, then the rule is explicitly scoped to text-content events and explicitly does not bind the prefix, so no reconciliation is left to the reader.

### Requirement: R-CNF-007 — A tool call is reconstructible whole, fragmented or not, and carries an observable ordinal

The suite MUST assert that a tool call arrives with identity, name, exact argument bytes and an observable ordinal (`CAP-R-02`). It MUST cover a call whose argument bytes arrive **fragmented across several deltas interleaved with another call's**, reconstructing each call's bytes exactly and attributing every fragment to the right call. It MUST also cover a **whole call delivered with zero deltas**, which MUST be accepted rather than rejected for missing incremental delivery. Ordinals MUST be observable and MUST distinguish two calls to the same tool name.

The zero-delta case's script represents a started response and MUST be a full conformant lifecycle stream: the lifecycle prefix, the whole call, and a terminal completion whose finish reason is the closed-vocabulary tool-call value. Its exact drained count MUST be derived from that script (`R-CNF-019`). The fragmented and ordinal cases reconstruct by block identity and assert no count, so they MUST remain unchanged (`R-CNF-021`).
(Previously: the zero-delta case asserted a two-event window with neither a lifecycle event nor a terminal.)

#### Scenarios

- **S-CNF-017** — Given two concurrent tool calls whose argument fragments interleave, when the suite reconstructs them, then each call's argument bytes match its source exactly and no fragment is misattributed.
- **S-CNF-018** — Given a tool call delivered whole with zero argument deltas, when the suite runs, then the case passes — the zero-delta shape is mandatory to support.
- **S-CNF-019** — Given two calls to the same tool name, when their ordinals are read, then they differ and order the two calls unambiguously.
- **S-CLA-006** `[test]` — Given `tool_call/zero_delta_whole_call_accepted` made a full conformant lifecycle stream — response start, tool-call block start, tool-call block end, completion — when it is drained, then **exactly four** events arrive in that positional order, the call still reconstructs its full arguments from the end event alone, and the zero-delta shape still passes.

### Requirement: R-CNF-008 — Mixed text and tool content ends on the tool-call finish reason, behind the lifecycle prefix

The suite MUST assert an interaction that emits text content **and** a tool call, terminating normally with the finish reason that denotes a tool call. Text and tool events MUST both survive with their ordering invariants intact, and the finish reason MUST come from the closed vocabulary rather than from free text. The interaction is a started response and MUST be a full conformant lifecycle stream; its shipped script already ended on the terminal completion, so adding the lifecycle prefix is the only fixture change, and the exact drained count MUST be re-derived from the amended script (`R-CNF-019`).
(Previously: the case asserted a six-event window with no lifecycle event.)

#### Scenarios

- **S-CNF-020** — Given an interaction emitting a text block and then a tool call, when it finishes, then both are present, ordering holds, and the finish reason is the tool-call value from the closed vocabulary.
- **S-CLA-007** `[test]` — Given `tool_call/mixed_text_and_tool_ends_on_tool_call_finish_reason` made a full conformant lifecycle stream — response start, text block start, text delta, text block end, tool-call block start, tool-call block end, completion — when it is drained, then **exactly seven** events arrive and the last is still the completion carrying the tool-call finish reason.

### Requirement: R-CNF-009 — Exactly one terminal, and the partial-output discriminator is always answerable

The suite MUST assert that every stream carries **exactly one** terminal event and that nothing follows it — on the normal path and on both failure-delivery paths. It MUST assert that each failure states whether content preceded it (`CAP-R-05`), and MUST cover both sides: a failure after content emitted and a failure with none. A stream carrying two terminals, or a terminal followed by any further event, MUST fail.

Both **stream-bearing** subtests are started responses and MUST be full conformant lifecycle streams, with their exact drained counts derived from their amended scripts (`R-CNF-019`). The normal-finish subtest ends on its completion; the **mid-stream-failure** subtest models a response that **began** — the prefix is present and the failure terminal follows it, and that failure terminal **is** the terminal a real producer would emit on that path, so no completion is added. A failure with no output at all is the pre-stream path, which returns no carrier and MUST remain free of any lifecycle assertion. The discriminator case indexes the **last** event and MUST remain unchanged (`R-CNF-021`).
(Previously: both stream-bearing subtests asserted a one-event window, and "mid-stream" was not distinguished from "pre-output" at the fixture level.)

#### Scenarios

- **S-CNF-021** — Given a normally-finished stream, a pre-stream failure, and a mid-stream failure, when each is asserted, then each carries exactly one terminal and nothing follows it.
- **S-CNF-022** — Given a failure raised after two text deltas and a failure raised before any event, when the discriminator is read, then the first reports content preceded and the second reports none.
- **S-CNF-023** — Given a subject emitting a text delta after its terminal, when the suite runs, then the case fails naming the post-terminal event.
- **S-CLA-008** `[test]` — Given `terminal/exactly_one_terminal_every_path`'s `normal_finish` subtest as a full conformant lifecycle stream — response start then completion — when it is drained, then **exactly two** events arrive in that order and no second terminal is present.
- **S-CLA-009** `[test]` — Given the `mid_stream_failure` subtest as a response that began — response start then the mid-stream error terminal — when it is drained, then **exactly two** events arrive in that order and the error terminal is the stream's only terminal, while the `pre_stream_failure` subtest still returns a nil carrier and a non-nil failure with no lifecycle event asserted.

### Requirement: R-CNF-011 — Cancellation closes within bounded time and leaks no goroutine

The suite MUST assert that cancelling the caller-owned signal ends the stream within a bounded deadline (`CAP-R-04`) and that the stream is closed exactly once by its producer. Leak freedom MUST be asserted through AI-22's opt-in, amplitude-based `R-STK-007` helper; the suite MUST NOT install an implicit global leak check and MUST NOT call `t.Parallel()` in any test that uses it.

The scenario is a started response and MUST carry the lifecycle prefix. This case's charter is a **window** of the stream, not the whole of it — the caller cancels before any terminal can be reached, and AI-20.3 makes the stream close **bare** — so it MUST assert **positionally** (the first received event) and MUST NOT assert a full-stream exact count, and no terminal MUST be added to its script. That window-scoped choice is deliberate and is stated here rather than left implicit. The synchronous-handoff proof — receiving the **first** event to establish that the producer passed its first send before cancellation — MUST be preserved, so the expected first received event MUST become the prefix rather than the text block start.
(Previously: the case asserted the first received event was a text block start.)

#### Scenarios

- **S-CNF-026** — Given a stream cancelled mid-consumption, when the suite waits with a bounded deadline, then the stream closes before the deadline and is closed exactly once.
- **S-CNF-027** — Given the cancellation scenario repeated under the leak helper, when it completes, then no goroutine growth beyond the helper's stated tolerance is observed.
- **S-CNF-028** — Given every suite test using the leak helper, when a reviewer enumerates them, then none calls `t.Parallel()`.
- **S-CLA-010** `[test]` — Given `cancellation/bounded_close_leak_free` with its script prefixed by a response start, when the consumer performs its single synchronous first receive, then the event received is the response start, and cancelling afterwards still closes the stream within the bounded deadline with no leak.

### Requirement: R-CNF-014 — `CAP-O-01` reasoning content: emitted whole, and never leaking into text

WHEN a subject offers reasoning content, the suite MUST assert that reasoning arrives as its own blocks, that reasoning **never** appears in the text channel, and that a signature the subject carries round-trips byte-exactly (AI-03 § 6.1: advertised is advertised whole). Both the plain and the redacted reasoning doors MUST be covered, and the redacted bit MUST carry forward into every delta and end built from that start. WHEN the subject does not offer it, the entry MUST be `absent` and the skip MUST be reported.

The redacted subtest is a started response and MUST be a full conformant lifecycle stream — the lifecycle prefix, the redacted reasoning block start and end, and a terminal completion — with its exact drained count and its block-start index derived from that script (`R-CNF-019`). The plain-reasoning subtest scans the whole recording and asserts no count, so it MUST remain unchanged (`R-CNF-021`).
(Previously: the redacted subtest asserted a two-event window with no terminal, and read the reasoning block start at index 0.)

#### Scenarios

- **S-CNF-035** — Given a subject offering reasoning, when the suite runs the case, then reasoning arrives in its own blocks and no reasoning byte appears in any text event.
- **S-CNF-036** — Given a subject carrying a reasoning signature, when the block round-trips, then the signature is byte-identical to the source.
- **S-CNF-037** — Given a redacted reasoning block start, when its deltas and end are read, then each carries the redacted bit forward.
- **S-CNF-038** — Given a subject declaring reasoning not offered, when the suite runs, then the case is skipped with a report and the record entry is `absent`.
- **S-CLA-011** `[test]` — Given `reasoning/whole_blocks_never_leak_into_text`'s redacted subtest made a full conformant lifecycle stream — response start, redacted reasoning block start, reasoning block end, completion — when it is drained, then **exactly four** events arrive in that positional order and the reasoning block start is read at index **1**, still carrying its redacted bit.

### Requirement: R-CNF-016 — `CAP-O-03` cache-boundary honoring, and the required cases living in this node

The suite MUST cover `CAP-O-03` through the declared capability expectation of `R-CNF-002`: honoring MUST be asserted as consumer-visible behaviour when offered, and recorded `absent` when the factory declares it is not. This node MUST additionally carry two cases that are **required** because they exercise `CAP-R-03`: (a) a normally-finished stream ends carrying a finish reason from the closed vocabulary, iterated **exhaustively over all seven values**; and (b) usage honours absent-versus-zero per field, so an absent count is distinguishable from a reported zero. Because the finish-reason vocabulary exports no enumerator, the seven values MUST be hand-listed in the suite **behind a drift guard** that fails when the upstream vocabulary gains or loses a member; the suite MUST NOT modify `src/ai` to obtain an enumerator.

All three of this node's stream-bearing cases are started responses and MUST be full conformant lifecycle streams, with their exact drained counts and completion indices derived from their amended scripts (`R-CNF-019`). Each already ended on its terminal completion, so the lifecycle prefix is the only fixture change in all three. The drift guard's own arithmetic — that all seven values are covered — MUST be unaffected by the prefix.
(Previously: each of the three cases asserted a one-event window and read the completion at index 0.)

#### Scenarios

- **S-CNF-042** — Given a subject declaring cache-boundary honoring offered, when the suite exercises it, then honoring is observed as consumer-visible behaviour; given one declaring it not offered, the entry is `absent` with a reported skip.
- **S-CNF-043** — Given all seven finish reasons, when the suite iterates them, then each is reachable on a normally-finished stream and each is a closed-vocabulary value.
- **S-CNF-044** — Given a hypothetical eighth finish reason added upstream without a suite case, when the drift guard runs, then it fails naming the uncovered value; given a value removed, it likewise fails.
- **S-CNF-045** — Given a usage record with one count absent and another reported as zero, when the suite reads them, then the two are distinguishable and neither is coerced into the other.
- **S-CNF-046** — Given the standing of cases (a) and (b), when it is computed, then both are **required** despite living in the optional-capability node.
- **S-CLA-012** `[test]` — Given `cache_boundary/honoring_is_consumer_visible` prefixed by a response start, when the stream is drained, then **exactly two** events arrive and the completion is read at index **1**.
- **S-CLA-013** `[test]` — Given `finish_reason/all_seven_values_reachable_drift_guarded`, when each of its **seven** subtests drains its prefixed script, then each sees **exactly two** events with the completion at index **1**, and the drift guard still reports all seven values covered.
- **S-CLA-014** `[test]` — Given `usage/absent_vs_zero_distinguishable` prefixed by a response start, when the stream is drained, then **exactly two** events arrive, the completion is read at index **1**, and absent remains distinguishable from zero.

## ADDED Requirements

### Requirement: R-CNF-019 — Every exact count is derived from its own amended script, never incremented

Each amended case MUST script a **full conformant lifecycle stream** and MUST state its expected drained window as an explicit, written-out **event-kind list**; the exact count MUST be read off that list. A count MUST NOT be obtained by incrementing the shipped count, because the shipped fixtures are Script-driven fakes that could legally omit events a real producer always emits — the ordering case's shipped four-event window lacked **both** the lifecycle prefix **and** the terminal completion, so incrementing it by one would have produced a window no conformant producer can ever emit.

Deriving the window MUST include supplying any terminal a real producer would emit and the shipped script omitted, **except** where the case's charter is precisely a different or absent terminal: the mid-stream-failure path ends on its failure terminal, and a stream cancelled before termination closes bare and carries none. WHERE a case's charter genuinely concerns only a window of the stream rather than the whole of it, the case MAY assert positionally instead of by full-stream count, but that choice MUST be stated explicitly in this spec for that case rather than left to the reader.

#### Scenarios

- **S-CLA-027** `[test]` — Given each amended case, when its expected window in this spec is compared with the events its amended script emits, then the two agree kind-for-kind and the stated count equals the list's length — no case's count is a shipped count plus one where its script also gained a terminal.
- **S-CLA-028** `[inspection]` — Given every amended case, when its script is read, then it ends on the terminal a real producer would emit on that path — a completion on a clean transcript, the failure terminal on the mid-stream-failure path, and no terminal on the cancellation path that closes bare — and any case asserting positionally rather than by count says so explicitly.

### Requirement: R-CNF-020 — The lifecycle assertion's bite is a permanent, provable negative

Every case this amendment changes MUST be able to **fail** against a fixture that omits the lifecycle prefix. The suite MUST therefore factor the positional-and-identity check into a reusable, directly-callable check over an event slice, so that a **synthetic start-less slice** can be handed to it and provably rejected. That negative proof MUST be a **permanent artifact of the suite** — a durable guard test that ships and keeps running — and MUST NOT be a staged mutation that is demonstrated once and then deleted.

The failure message MUST name the absent lifecycle event, so a real adapter that never emits a response start is told what is missing rather than only that a count differed. A start-bearing fixture and a start-less fixture MUST produce **different outcomes** in every changed case; a case whose outcome is identical under both fixtures is a vacuous pass and MUST be treated as a defect in this amendment.

#### Scenarios

- **S-CLA-015** `[test]` — Given the extracted lifecycle check and a synthetic event slice whose first event is not a response start, when the check runs, then it reports a violation naming the absent lifecycle event, and this guard ships as a permanent test rather than a deleted staging step.
- **S-CLA-016** `[test]` — Given the extracted check and a slice whose position-0 response start carries an identity differing from the scripted values, when the check runs, then it reports a violation naming the mismatched field — proving equality, not mere presence, is what is asserted.
- **S-CLA-017** `[inspection]` — Given each of the ten changed cases, when its start-bearing script and an otherwise-identical start-less script are compared, then the two produce different outcomes in every one, so no changed case passes vacuously.

### Requirement: R-CNF-021 — The naturally-tolerant cases are recorded with the property that makes them tolerant

The suite's cases that are **not** amended MUST be recorded here together with the structural property that makes each one indifferent to a longer lifecycle prefix — end-indexing, block-identity reconstruction, whole-recording scanning, or never opening a stream. This register exists so a later lifecycle change can re-derive the changed/unchanged split **mechanically** from the assertion shape rather than by re-reading every case. A case whose assertion shape later becomes positional or count-exact MUST move out of this register in that change's own delta.

#### Scenarios

- **S-CLA-018** `[inspection]` — Given `tool_call/fragmented_interleaved_reconstructs_exactly`, when its assertions are read, then it reconstructs by block identity and asserts no count or leading index, so a prefixed script changes nothing.
- **S-CLA-019** `[inspection]` — Given `tool_call/ordinal_distinguishes_same_tool_name`, when its assertions are read, then it compares ordinals only, with no count or leading index.
- **S-CLA-020** `[inspection]` — Given `terminal/partial_output_discriminator_both_states`, when its assertions are read, then it indexes the **last** event, which a leading prefix does not move.
- **S-CLA-021** `[inspection]` — Given `terminal/all_nine_failure_categories_exhaustive`, when its assertions are read, then it likewise indexes the last event across all nine categories.
- **S-CLA-022** `[inspection]` — Given `cancellation/abandoned_then_cancelled_drops_bare`, when its assertions are read, then it scans the whole recording for an invented terminal, so one additional leading event is harmless and the bare-close claim is unweakened.
- **S-CLA-023** `[inspection]` — Given `redaction/sentinel_absent_from_every_rendering`, when its assertions are read, then it scans every event and every rendering, so one more event only widens the scan.
- **S-CLA-024** `[inspection]` — Given `token_counting/asked_of_the_provider_value`, when its assertions are read, then it never opens a stream, so no lifecycle event exists for it to observe.

### Requirement: R-CNF-022 — The amendment is behaviour-neutral outside the suite's own fixtures

This amendment MUST change only `agenttest` — its conformance fixtures, their arithmetic, the durable guard of `R-CNF-020`, and the two additive read-only surfaces of `R-CNF-023` and `R-CNF-024` — plus this spec. The end-to-end drivers MUST stay green: `RunConformance` against AI-21's fake MUST still return a **pass** verdict with all **eight** capability entries and standing unchanged, and every direct per-case caller in the suite's own test file MUST still pass. No `src/ai` file, no `openaicompat` file and no module dependency MUST change (`NFR-CNF-A`, `NFR-CNF-B` hold).

Both new surfaces MUST be **purely additive**: no existing exported behaviour changes, and no existing case's semantics change. Because `NFR-CNF-D` and its `S-CNF-063` forbid any edit to AI-21's `fake_*.go` and AI-22's `stream_kit_*.go`, `R-CNF-024`'s introspection MUST be delivered **without editing those files**, so `NFR-CNF-D` continues to hold unmodified.
(Previously — before the 2026-08-04 scope extension: the amendment claimed the only non-spec changes were fixtures plus the durable guard.)

#### Scenarios

- **S-CLA-025** `[test]` — Given the amended suite run end to end against AI-21's fake factory, when the verdict is computed, then it is a **pass**, the record carries exactly eight entries, and no entry is `not exercised`.
- **S-CLA-026** `[inspection]` — Given the merged diff, when a reviewer looks for edits under `src/ai/`, under `openaicompat`, or in `backend/agent/go.mod`, then none exists, and the only non-spec changes are `agenttest` conformance fixtures, the durable guard of `R-CNF-020`, and the additive read-only surfaces of `R-CNF-023` and `R-CNF-024`.

### Requirement: R-CNF-023 — A capability-scoped conformance entry point exists

The suite MUST offer an **exported** entry point that runs exactly those registered cases whose keyed required capability equals a caller-named capability. Cases outside that scope MUST be neither run nor reported — in particular, a case outside the scope MUST NOT surface as a skipped-required error, which is what makes a partial adapter able to run the cases it can satisfy at all. Today only the unexported `runConformanceCases` can express a subset, so an adapter that has implemented text and nothing else has no legal way to run the text cases.

The scoped entry point MUST preserve, unchanged: the factory-declaration defect checks of `R-CNF-002` (all three optional declarations remain mandatory, and an undeclared one still fails construction naming the capability), and each in-scope case's own semantics and standing. It MUST NOT be a waiver mechanism: the full `RunConformance` remains the **complete, unscoped gate**, its registry iteration and its non-waivable required cases unchanged, and a scoped run MUST NOT be presentable as evidence of full conformance. A scoped run MUST fail when any case within its scope fails.

#### Scenarios

- **S-CLA-029** `[test]` — Given the registry and a scoped run named for the streaming-text capability, when it runs, then exactly the two text cases execute and no other registered case does — observable through the executed subtest names or an equivalent observable — and no out-of-scope case is reported as a skipped required case.
- **S-CLA-030** `[test]` — Given a factory whose subject fails a case that lies **within** the named scope, when the scoped run executes, then the scoped run fails and names that case.
- **S-CLA-031** `[test]` — Given a factory that leaves one optional capability undeclared, when a scoped run is started, then it still fails at construction naming the undeclared capability, exactly as `R-CNF-002`'s `S-CNF-006` requires of the unscoped runner.
- **S-CLA-032** `[inspection]` — Given `RunConformance`, when its behaviour is compared before and after this change, then it still iterates the entire registry with required capabilities non-waivable, and a scoped run is nowhere accepted as a substitute for it.

### Requirement: R-CNF-024 — Scripts are introspectable, read-only, by external packages

A `Script` MUST be introspectable from **outside** the `agenttest` package through an exported, vendor-neutral, **read-only** surface sufficient to reconstruct its ordered steps: for each step, whether it emits an event — and, when it does, **which** `ai.Event` — or is instead a hold/gate step. Today `Script` and `Step` carry only unexported fields and no methods, so an external package cannot recover a scripted event at all, and a bridge that renders a suite-authored scenario into a vendor wire transcript cannot be written.

The surface MUST expose **no mutation**: no setter, no constructor that rewrites an existing script, and no aliasing of internal slices or backing arrays that would let a caller mutate a script through the value it was handed — copies or genuinely read-only views only. It MUST stay **vendor-neutral**: it MUST name no adapter, import no adapter package, and MUST NOT encode any wire format. It MUST be additive and MUST be delivered **without editing** AI-21's `fake_*.go` or AI-22's `stream_kit_*.go`, so `NFR-CNF-D` holds unmodified.

#### Scenarios

- **S-CLA-033** `[test]` — Given a script authored in one package and read from a **separate external test package**, when its steps are walked through the introspection surface, then the ordered list of emitted `ai.Event` values is reconstructible and equals, event for event, what a fake provider built from that same script emits when drained.
- **S-CLA-034** `[test]` — Given a script mixing emit steps with hold/gate steps, when it is introspected, then each step reports which of the two it is, and the emit steps' events are recoverable while the hold/gate steps are identified as carrying no event.
- **S-CLA-035** `[inspection]` — Given the introspection surface, when it is reviewed, then it exposes no setter and no mutating method, a caller that mutates anything it was handed cannot thereby alter the source script, it imports no adapter package and encodes no wire format, and no `fake_*.go` or `stream_kit_*.go` file was edited to provide it.
