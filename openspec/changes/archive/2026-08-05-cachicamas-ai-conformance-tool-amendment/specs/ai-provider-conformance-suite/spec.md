# Delta for `ai-provider-conformance-suite` — tool-call cases assert reconstruction, not delta shape

> **Change**: `cachicamas-ai-conformance-tool-amendment` · **Amends**: [`openspec/specs/ai-provider-conformance-suite/spec.md`](../../../../specs/ai-provider-conformance-suite/spec.md)
> **Stacks on**: `cachicamas-ai-conformance-lifecycle-amendment` (the *first* amendment). Every `## MODIFIED` block below is copied **from that change's post-state delta** ([`../../../cachicamas-ai-conformance-lifecycle-amendment/specs/ai-provider-conformance-suite/spec.md`](../../../cachicamas-ai-conformance-lifecycle-amendment/specs/ai-provider-conformance-suite/spec.md)), not from the shipped main spec, because `R-CNF-019`/`R-CNF-020`/`R-CNF-021` exist **only** there. Archive order is **first → second** (`R-CNF-026`).
> **Amended requirements (5, all MODIFIED)**: `R-CNF-007`, `R-CNF-008`, `R-CNF-019`, `R-CNF-020`, `R-CNF-021`
> **New requirements (2, ADDED)**: `R-CNF-025` (argument-channel precedence), `R-CNF-026` (stacking and archive order) · **Removed / renamed**: none
> **Requirement total**: **7** (5 MODIFIED + 2 ADDED).
> **Scenario counts** — new scenarios `S-CTA-001 … S-CTA-019` = **19** (**12** `[test]`, **7** `[inspection]`); predecessor scenarios carried forward = **20** (18 retained verbatim, 2 — `S-CLA-006`, `S-CLA-007` — **restated** in place because this amendment removes their exact whole-window kind lists). Every row of the proposal's 17-row scope table maps to at least one scenario.
> **Scenario series**: this delta opens the **`S-CTA-0NN`** series (Conformance Tool Amendment) rather than continuing `S-CLA`, so the two stacked amendments' scenarios stay attributable to their own change after both archive.
> **Sources**: proposal `openspec/changes/cachicamas-ai-conformance-tool-amendment/proposal.md` · neutral contract [`ai-tool-call-events`](../../../../specs/ai-tool-call-events/spec.md) `R-ATC-010` / `S-ATC-027` · `src/ai/tool_call_event.go` (`ai.NewToolCallEnd`) · AI-30 wire citations `C9.4` (the `arguments` field carries **zero** fragmentation semantics) and `C9.6` (chat scope has **no** per-call completeness signal — no end marker, no count, no done event) in `openspec/changes/cachicamas-ai-provider-tool-stream/citations.md`.
> This delta **relaxes exactly one thing** — an exact whole-window event-kind census in two cases — and **strengthens four** (identity equality, byte reconstruction across both channels, a boundary on count derivation, a re-anchored non-vacuity proof). No `src/ai`, `openaicompat`, `fake_*.go` or `stream_kit_*.go` change: `NFR-CNF-A` … `NFR-CNF-F` hold unmodified and are not restated here.

---

## Definitions added by this amendment

- **A fragmentable argument channel** — a tool call in a case's script whose argument bytes a conformant subject MAY deliver either whole on the call-end event or split across any number of `ToolCallDelta` events. The pinned OpenAI-compatible wire makes every non-empty-argument call one of these (`C9.4`, `C9.6`), and `R-ATC-010` forbids Layer 1 from exposing which delivery happened.
- **A relative-order (subsequence) kind assertion** — an assertion that the scripted kinds occur in the drained window in the scripted relative order, with the number of `ToolCallDelta` occurrences **unconstrained**. It is not a census: it neither counts the window nor pins each position.
- **The reconstruction observable** — the single argument-byte value the suite's reconstruction helper reports for one tool-call block: the call-end event's bytes **when non-empty**, else the concatenation of that block's delta fragments in arrival order. Never both, never summed (`R-CNF-025`).

## MODIFIED Requirements

### Requirement: R-CNF-007 — A tool call is reconstructible whole, fragmented or not, and carries an observable ordinal

The suite MUST assert that a tool call arrives with identity, name, exact argument bytes and an observable ordinal (`CAP-R-02`). It MUST cover a call whose argument bytes arrive **fragmented across several deltas interleaved with another call's**, reconstructing each call's bytes exactly and attributing every fragment to the right call. It MUST also cover a **whole call delivered with zero deltas**, which MUST be accepted rather than rejected for missing incremental delivery. Ordinals MUST be observable and MUST distinguish two calls to the same tool name.

The zero-delta case's script represents a started response and MUST remain a full conformant lifecycle stream: the lifecycle prefix, the whole call, and a terminal completion whose finish reason is the closed-vocabulary tool-call value. Its script carries a **fragmentable argument channel**, so the case MUST NOT assert an exact whole-window event-kind list; per `R-CNF-019` it MUST instead assert (a) the lifecycle prefix positionally **and by identity equality**, (b) the scripted kinds in **relative order** with delta occurrences unconstrained, and (c) the call's id, name and argument **bytes** by reconstruction keyed on block identity, through the single reconstruction observable of `R-CNF-025`.

This case's charter is hereby restated, honestly and not silently reinterpreted: it is **"a zero-delta *script* reconstructs identically however the subject delivers it"** — AI-21's fake delivers the bytes end-carried, while a conformant real adapter may deliver the same bytes delta-carried, and `R-ATC-010` makes the difference unobservable after reconstruction. `S-CNF-018`'s "delivered whole with zero argument deltas" therefore describes the **script**, never a constraint on the subject's drained event census.
(Previously — under the first amendment: the case asserted an **exact four-event** positional kind list `ResponseStart, ToolCallStart, ToolCallEnd, Completion`, and read the arguments "from the end event alone" — a window no conformant OpenAI-compatible adapter can emit for `{"q":"weather"}`, and a count `R-ATC-010` declares unobservable.)

#### Scenarios

- **S-CNF-017** — Given two concurrent tool calls whose argument fragments interleave, when the suite reconstructs them, then each call's argument bytes match its source exactly and no fragment is misattributed.
- **S-CNF-018** — Given a tool call delivered whole with zero argument deltas, when the suite runs, then the case passes — the zero-delta shape is mandatory to support.
- **S-CNF-019** — Given two calls to the same tool name, when their ordinals are read, then they differ and order the two calls unambiguously.
- **S-CLA-006** `[test]` *(restated by this amendment)* — Given `tool_call/zero_delta_whole_call_accepted` still scripted as a full conformant lifecycle stream, when it is drained, then the lifecycle prefix is at position 0 carrying the scripted identity, the kinds response-start → tool-call-start → tool-call-end → completion appear in that **relative** order, and the call reconstructs its full arguments — with **no exact whole-window count or positional kind list asserted anywhere in the case**.
- **S-CTA-001** `[test]` — Given the fake-shaped fixture (the shipped script: zero `ToolCallDelta` events, all argument bytes on the call-end), when the case runs, then it passes and the reconstruction observable equals `{"q":"weather"}` byte-for-byte.
- **S-CTA-002** `[test]` — Given an adapter-shaped fixture delivering the **same** call as one or more `ToolCallDelta` fragments with a zero-length call-end, when the same case assertions run over it, then it passes with the same reconstructed id, name and argument bytes as `S-CTA-001` — the wire-reality proof the shipped exact list fails.
- **S-CTA-003** `[test]` — Given a start-less fixture (the lifecycle prefix removed) for this case, when the case runs, then it **fails** through `checkLifecyclePrefix` naming the absent lifecycle event, so relaxing the census does not relax the lifecycle bite (`R-CNF-020`).
- **S-CTA-004** `[test]` — Given a fixture whose delivered argument bytes differ from the scripted bytes by one byte, when the case runs, then it fails naming the reconstruction mismatch.
- **S-CTA-005** `[inspection]` — Given a reader asking whether this amendment silently reinterprets `S-CNF-018`, when this requirement's text is read, then the case's charter is restated explicitly — zero-delta describes the script, not the subject's census — so no reinterpretation is left implicit.

### Requirement: R-CNF-008 — Mixed text and tool content ends on the tool-call finish reason, behind the lifecycle prefix

The suite MUST assert an interaction that emits text content **and** a tool call, terminating normally with the finish reason that denotes a tool call. Text and tool events MUST both survive with their ordering invariants intact, and the finish reason MUST come from the closed vocabulary rather than from free text. The interaction is a started response and MUST remain a full conformant lifecycle stream, opening on the lifecycle prefix and ending on the terminal completion.

Its script carries a **fragmentable argument channel**, so — exactly as in `R-CNF-007` — the case MUST NOT assert an exact whole-window event-kind list. It MUST assert the lifecycle prefix positionally **and by identity equality**, the scripted kinds in **relative order** with delta occurrences unconstrained, the survival of the scripted text delta, the tool call's id, name and argument bytes by reconstruction, and — retained unweakened — that the **last** event is the completion whose finish reason **equals** the tool-call value.
(Previously — under the first amendment: the case asserted an **exact seven-event** positional kind list, which the same forced `ToolCallDelta` makes unpassable for any conformant adapter.)

#### Scenarios

- **S-CNF-020** — Given an interaction emitting a text block and then a tool call, when it finishes, then both are present, ordering holds, and the finish reason is the tool-call value from the closed vocabulary.
- **S-CLA-007** `[test]` *(restated by this amendment)* — Given `tool_call/mixed_text_and_tool_ends_on_tool_call_finish_reason` still scripted as a full conformant lifecycle stream, when it is drained, then the lifecycle prefix is at position 0 carrying the scripted identity, the kinds response-start → text-block-start → text-delta → text-block-end → tool-call-start → tool-call-end → completion appear in that **relative** order, and the last event is still the completion carrying the tool-call finish reason — with **no exact whole-window count or positional kind list asserted**.
- **S-CTA-006** `[test]` — Given this case's drained window, when the relative-order assertion runs, then extra `ToolCallDelta` occurrences between the tool-call start and end do **not** fail it, while a text delta appearing **after** the tool-call start does.
- **S-CTA-007** `[test]` — Given an adapter-shaped fixture that fragments this case's tool arguments across two deltas, when the case runs unchanged, then it passes, the scripted text delta still survives, and the reconstructed arguments equal `{"q":"weather"}`.
- **S-CTA-008** `[test]` — Given a start-less fixture for this case, when it runs, then it **fails** through `checkLifecyclePrefix`; given a fixture whose prefix carries a different `ResponseID`, it likewise fails naming the mismatched field.

### Requirement: R-CNF-019 — Every exact count is derived from its own amended script, never incremented — and only where the contract makes it observable

Each amended case MUST script a **full conformant lifecycle stream** and MUST state its expected drained window as an explicit, written-out **event-kind list**; the exact count MUST be read off that list. A count MUST NOT be obtained by incrementing the shipped count, because the shipped fixtures are Script-driven fakes that could legally omit events a real producer always emits — the ordering case's shipped four-event window lacked **both** the lifecycle prefix **and** the terminal completion, so incrementing it by one would have produced a window no conformant producer can ever emit.

Deriving the window MUST include supplying any terminal a real producer would emit and the shipped script omitted, **except** where the case's charter is precisely a different or absent terminal: the mid-stream-failure path ends on its failure terminal, and a stream cancelled before termination closes bare and carries none. WHERE a case's charter genuinely concerns only a window of the stream rather than the whole of it, the case MAY assert positionally instead of by full-stream count, but that choice MUST be stated explicitly in this spec for that case rather than left to the reader.

**The boundary on exact counts.** An exact drained count is derivable **only where the neutral contract makes the counted quantity observable**. WHERE a case's script carries a **fragmentable argument channel**, the number of events in its window is a function of delta count, which `R-ATC-010` and `S-ATC-027` declare unobservable — so such a case MUST NOT assert an exact count or a full positional kind list, and MUST instead assert relative order plus reconstruction. Everywhere else the exact count remains **script-derivable and MUST NOT be relaxed**: exact counts are what caught this defect class. The relative-order assertion MUST be one **shared** helper, mirroring `requireDrainedKinds`' one-derivation-shape discipline; it MUST NOT be reimplemented per case.
(Previously — under the first amendment: every amended case derived an exact count with no stated boundary, so a case over a fragmentable channel was obliged to assert a quantity the contract declares unobservable.)

#### Scenarios

- **S-CLA-027** `[test]` — Given each amended case, when its expected window in this spec is compared with the events its amended script emits, then the two agree kind-for-kind and the stated count equals the list's length — no case's count is a shipped count plus one where its script also gained a terminal.
- **S-CLA-028** `[inspection]` — Given every amended case, when its script is read, then it ends on the terminal a real producer would emit on that path — a completion on a clean transcript, the failure terminal on the mid-stream-failure path, and no terminal on the cancellation path that closes bare — and any case asserting positionally rather than by count says so explicitly.
- **S-CTA-009** `[test]` — Given the two cases over a fragmentable argument channel, when their assertions are read, then both call the **one shared** relative-order helper and neither calls `requireDrainedKinds`; and given a subject that drops the scripted tool-call end entirely, the helper still fails naming the missing kind.
- **S-CTA-010** `[inspection]` — Given every one of the suite's 17 registered cases, when the boundary is applied, then exactly the two carrying a tool-call block in their script lose their exact count, and the other 15 keep script-derivable counts because no fragmentable channel exists in them.

### Requirement: R-CNF-020 — The lifecycle assertion's bite is a permanent, provable negative

Every case this amendment changes MUST be able to **fail** against a fixture that omits the lifecycle prefix. The suite MUST therefore factor the positional-and-identity check into a reusable, directly-callable check over an event slice, so that a **synthetic start-less slice** can be handed to it and provably rejected. That negative proof MUST be a **permanent artifact of the suite** — a durable guard test that ships and keeps running — and MUST NOT be a staged mutation that is demonstrated once and then deleted.

The failure message MUST name the absent lifecycle event, so a real adapter that never emits a response start is told what is missing rather than only that a count differed. A start-bearing fixture and a start-less fixture MUST produce **different outcomes** in every changed case; a case whose outcome is identical under both fixtures is a vacuous pass and MUST be treated as a defect in this amendment.

**Re-anchoring for the two tool-call cases.** Before this change, those two cases derived their bite partly from an exact count that implicitly proved position 0. Removing the census removes that implicit proof, so their bite MUST now be carried **explicitly** by `checkLifecyclePrefix` — position 0 plus `ResponseID`/`ServedModel` **equality**. This is a net **strengthening**: neither case carried any identity assertion before. The durable start-less guard MUST continue to fail **both** amended cases after the change; a start-less fixture that newly passes either one is a defect in this amendment, not a tolerated relaxation.
(Previously — under the first amendment: the non-vacuity obligation was stated over "the ten changed cases" without distinguishing which of them proved position 0 by count and which by an explicit check.)

#### Scenarios

- **S-CLA-015** `[test]` — Given the extracted lifecycle check and a synthetic event slice whose first event is not a response start, when the check runs, then it reports a violation naming the absent lifecycle event, and this guard ships as a permanent test rather than a deleted staging step.
- **S-CLA-016** `[test]` — Given the extracted check and a slice whose position-0 response start carries an identity differing from the scripted values, when the check runs, then it reports a violation naming the mismatched field — proving equality, not mere presence, is what is asserted.
- **S-CLA-017** `[inspection]` — Given each changed case of both amendments, when its start-bearing script and an otherwise-identical start-less script are compared, then the two produce different outcomes in every one, so no changed case passes vacuously.
- **S-CTA-011** `[test]` — Given the two cases amended here, each run twice — once start-bearing, once start-less — when the four outcomes are compared, then each case **passes** start-bearing and **fails** start-less, and the failure names the absent lifecycle event; a run in which either case passes both ways fails this scenario.

### Requirement: R-CNF-021 — The naturally-tolerant cases are recorded with the property that makes them tolerant

The suite's cases that are **not** amended MUST be recorded here together with the structural property that makes each one indifferent to a longer lifecycle prefix — end-indexing, block-identity reconstruction, whole-recording scanning, or never opening a stream. This register exists so a later lifecycle change can re-derive the changed/unchanged split **mechanically** from the assertion shape rather than by re-reading every case. A case whose assertion shape later becomes positional or count-exact MUST move out of this register in that change's own delta.

The register MUST additionally record the two cases this amendment converts, because after conversion their assertion shape is exactly the tolerant one — relative order plus block-identity reconstruction, with no count and no leading index beyond the explicit lifecycle check. A future lifecycle or dialect change MUST therefore be able to read this register and conclude, mechanically, that these two need no further amendment.
(Previously — under the first amendment: the register listed only the seven cases that had never needed amending.)

#### Scenarios

- **S-CLA-018** `[inspection]` — Given `tool_call/fragmented_interleaved_reconstructs_exactly`, when its assertions are read, then it reconstructs by block identity and asserts no count or leading index, so a prefixed script changes nothing.
- **S-CLA-019** `[inspection]` — Given `tool_call/ordinal_distinguishes_same_tool_name`, when its assertions are read, then it compares ordinals only, with no count or leading index.
- **S-CLA-020** `[inspection]` — Given `terminal/partial_output_discriminator_both_states`, when its assertions are read, then it indexes the **last** event, which a leading prefix does not move.
- **S-CLA-021** `[inspection]` — Given `terminal/all_nine_failure_categories_exhaustive`, when its assertions are read, then it likewise indexes the last event across all nine categories.
- **S-CLA-022** `[inspection]` — Given `cancellation/abandoned_then_cancelled_drops_bare`, when its assertions are read, then it scans the whole recording for an invented terminal, so one additional leading event is harmless and the bare-close claim is unweakened.
- **S-CLA-023** `[inspection]` — Given `redaction/sentinel_absent_from_every_rendering`, when its assertions are read, then it scans every event and every rendering, so one more event only widens the scan.
- **S-CLA-024** `[inspection]` — Given `token_counting/asked_of_the_provider_value`, when its assertions are read, then it never opens a stream, so no lifecycle event exists for it to observe.
- **S-CTA-012** `[inspection]` — Given `tool_call/zero_delta_whole_call_accepted` and `tool_call/mixed_text_and_tool_ends_on_tool_call_finish_reason` **after** this amendment, when their assertions are read, then each asserts relative order plus block-identity reconstruction with no count, so both now belong in this register with that recorded property.
- **S-CTA-013** `[inspection]` — Given the 15 registered cases this amendment does **not** touch — the 2 already-tolerant tool cases above plus `text/*` (2), `terminal/*` (3), `cancellation/*` (2), `reasoning`, `cache_boundary`, `finish_reason`, `usage`, `token_counting` (5) and `redaction` (1) — when each script is read, then none contains a tool-call block, so none carries a fragmentable argument channel and each keeps whatever count `R-CNF-019` already derived for it, unchanged by this delta.

## ADDED Requirements

### Requirement: R-CNF-025 — Reconstructed arguments have exactly one defined observable across both delivery channels

The suite's tool-call reconstruction helper MUST report, for each tool-call block, **one** argument-byte value derived by a single stated rule: **the call-end event's bytes when they are non-empty, otherwise the concatenation of that block's delta fragments in arrival order**. It MUST NOT sum, append or otherwise combine the two channels, because a subject that legally populates both would then be judged against double-counted bytes.

Both channels MUST be admissible. `ai.NewToolCallEnd` stores its argument bytes exactly as supplied with **no** substitution of `{}` for zero length, so a zero-length call-end is a legal event and MUST NOT be treated as a defect when delta fragments carry the payload. The helper MUST NOT expose delta count, fragment boundaries or any "was fragmented" indicator to a case, because `R-ATC-010`/`S-ATC-027` declare those unobservable — a case MUST be unable to distinguish the two deliveries through this helper.

#### Scenarios

- **S-CTA-014** `[test]` — Given a block whose call-end carries non-empty bytes and whose deltas carry nothing, when the helper reconstructs it, then the observable equals the call-end bytes exactly.
- **S-CTA-015** `[test]` — Given a block whose call-end carries zero-length bytes and whose deltas carry the payload in fragments, when the helper reconstructs it, then the observable equals the fragments concatenated in arrival order, and the zero-length end is not reported as a defect.
- **S-CTA-016** `[test]` — Given a block where **both** channels carry the same complete payload, when the helper reconstructs it, then the observable equals that payload exactly **once** — never doubled, and never the two channels concatenated.
- **S-CTA-017** `[inspection]` — Given the helper's surface, when a case author looks for a delta count, a fragment-boundary list or a fragmented flag, then none is reachable, so no case can assert a quantity `R-ATC-010` makes unobservable.

### Requirement: R-CNF-026 — This amendment stacks on the first, and archives after it

This delta's `## MODIFIED` blocks are copied from the **post-state** of `cachicamas-ai-conformance-lifecycle-amendment`, because `R-CNF-019`, `R-CNF-020` and `R-CNF-021` do not exist in the shipped main spec and `R-CNF-007`/`R-CNF-008` exist there only in their pre-first-amendment form. The two changes MUST therefore be archived in order: **first amendment, then this one**. Archiving this change against the shipped main spec — or before the first amendment — MUST be treated as a defect, because the archive step replaces requirements wholesale and would either fail to find `R-CNF-019`/`R-CNF-020`/`R-CNF-021` or silently discard the first amendment's text for `R-CNF-007`/`R-CNF-008`.

After both archives, the main spec MUST carry: this delta's text for `R-CNF-007`, `R-CNF-008`, `R-CNF-019`, `R-CNF-020`, `R-CNF-021`; this delta's `R-CNF-025` and `R-CNF-026`; and the first amendment's text, unaltered by this delta, for `R-CNF-005`, `R-CNF-006`, `R-CNF-009`, `R-CNF-011`, `R-CNF-014`, `R-CNF-016`, `R-CNF-022`, `R-CNF-023`, `R-CNF-024`.

#### Scenarios

- **S-CTA-018** `[inspection]` — Given each `## MODIFIED` block here, when it is diffed against the same requirement in the first amendment's delta, then it is that block edited — never the shipped main-spec block — and no scenario the first amendment added to those requirements is dropped except the two explicitly restated.
- **S-CTA-019** `[inspection]` — Given both changes archived in order, when the resulting main spec is read, then each of the fourteen requirement IDs above resolves to exactly one text, no `R-CNF-0NN` id is duplicated, and every `S-CLA` and `S-CTA` scenario id appears exactly once.
