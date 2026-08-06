# ai-provider-completion — Specification

> **Change**: `cachicamas-ai-provider-completion` · **Milestone**: AI-31 (nodes AI-31.1, AI-31.2, AI-31.3)
> **Phase**: spec · **Date**: 2026-08-04 · **Project**: cachicamas (witsaba)
> **Capability type**: NEW (full spec, not a delta). The `ai-provider-text-stream` charter delta lives in
> the sibling file `../ai-provider-text-stream/spec.md` of this same change.
> **Binding rulings**: proposal decisions **D1**, **D2**, **D3** as ruled by the coordinator — not reopened here.

> **Revision note (design-validation corrective, 2026-08-04).** Four findings from a fresh-context design
> validation are applied in place: the doc 0002 invalidation table now enumerates **all five** voided sites
> (C2); `S-ACP-017` is reworked into an arithmetically-impossible-under-exclusivity proof (G1); `R-ACP-012`'s
> third-label clause is made explicitly conditional (G4); and this package's transcript-fixture convention —
> the inline string literal committed in test source — is defined so "named pinning fixture" has a fixed
> referent (G2). Requirement and scenario counts are **unchanged**: the C2 fix extends `S-ACP-029`'s site list
> rather than adding IDs.

**Mechanical count — 12 requirements, 31 scenarios (18 `[test]`, 13 `[inspection]`).**

| Requirement | Scenarios | Node |
| --- | --- | --- |
| R-ACP-001 wire finish-reason mapping | S-ACP-001…003 | AI-31.1 |
| R-ACP-002 unreachable neutral values recorded | S-ACP-004…005 | AI-31.1 |
| R-ACP-003 novel stop value stays a typed malformed response (D1) | S-ACP-006…008 | AI-31.1 |
| R-ACP-004 matched stop-sequence value recorded absent (U4) | S-ACP-009…010 | AI-31.1 |
| R-ACP-005 usage detail fields reach the completion event | S-ACP-011…014 | AI-31.2 |
| R-ACP-006 cache semantics: map raw, record the un-attested relation | S-ACP-015…017 | AI-31.2 |
| R-ACP-007 usage-frame multiplicity and non-double-counting merge | S-ACP-018…020 | AI-31.2 |
| R-ACP-008 never invent, never assume frame position | S-ACP-021…023 | AI-31.3 |
| R-ACP-009 landed absent-versus-zero behaviour is frozen | S-ACP-024…025 | all |
| R-ACP-010 `CapCompletionMetadata` is not bridge-run (D3) | S-ACP-026…027 | AI-31.1 |
| R-ACP-011 doc 0002 amendment obligations | S-ACP-028…029 | all |
| R-ACP-012 every wire statement resolves to a citation | S-ACP-030…031 | all |

## Purpose

This capability owns the OpenAI-compatible adapter's **terminal metadata mapping**: which wire finish
values reach which neutral finish reason, which neutral values this dialect cannot express, which usage
counts reach `ai.Usage`, what is honestly claimable about their arithmetic, and what the adapter does with
metadata it did not expect. Scope is mapping and its recorded limits. It adds no field to `ai.Completion`,
widens no neutral vocabulary, and prices nothing.

**Citation namespace**: `U1`…`U6` from this change's `citations.md`; `C1`…`C8` from the AI-28 change's
`citations.md` remain citable for chunk-level facts. No other label is admissible.

---

## Requirements

### R-ACP-001 — The five wire finish values map onto the neutral vocabulary

The adapter MUST map each member of this dialect's five-member `finish_reason` enum (**U5**, both chat enums
are exactly `{stop, length, tool_calls, content_filter, function_call}`) onto the neutral value below, and
MUST NOT map any of them onto a value other than the one named.

| Wire value | Neutral value | Basis |
| --- | --- | --- |
| `stop` | `FinishReasonStop` | U5 enum membership + `ai.NormalizeFinishReason` table |
| `length` | `FinishReasonLength` | idem |
| `tool_calls` | `FinishReasonToolCalls` | idem |
| `function_call` (deprecated, still enum-legal) | `FinishReasonToolCalls` | idem |
| `content_filter` | `FinishReasonContentFilter` | idem |

`content_filter` MUST NOT map to `FinishReasonRefusal`. `finish_reason.go` draws the line at
`FinishReasonContentFilter`: a content filter is "a decision taken **beside** the model … none of which is
the right response to a model that declined." Collapsing them erases a vocabulary slot AI-13.1 paid for.

#### Scenarios

- **S-ACP-001** *[test]* — Given a table over exactly the five wire values, when each terminates a transcript,
  then the completion event's finish reason is the row's neutral value and the table's row count is asserted
  to be 5, so a sixth wire member added later fails the count before it fails a mapping.
- **S-ACP-002** *[test]* — Given a transcript whose terminal chunk carries `"function_call"`, when it is
  drained, then the completion reports `FinishReasonToolCalls` and no error event is emitted — the deprecated
  member is enum-legal, not novel.
- **S-ACP-003** *[test]* — Given a transcript terminating on `"content_filter"`, when it is drained, then the
  completion reports `FinishReasonContentFilter` **and** the same assertion run against `FinishReasonRefusal`
  fails, proving the two are distinguished rather than coincidentally equal.

### R-ACP-002 — The neutral values unreachable on this dialect are enumerated with their basis

The adapter MUST NOT produce `FinishReasonRefusal`, `FinishReasonPauseTurn` or `FinishReasonUnknown` from any
wire `finish_reason`, and this spec MUST record each unreachability with its basis and its reopen trigger.

| Neutral value | Why unreachable here | Reopen trigger |
| --- | --- | --- |
| `FinishReasonRefusal` | **U5 NEGATIVE**: no chat `finish_reason` member spells refusal; none of the 92 `refusal` hits adjoins a `finish_reason` block. `delta.refusal` (**C7**) is the only refusal-shaped channel and its companion `finish_reason` is undocumented, so mapping its presence would be uncited inference (D2, ruling (a)). | The pinned dialect gains a refusal finish member; or a real-backend transcript at AI-38 pins the companion value. |
| `FinishReasonPauseTurn` | **U5 NEGATIVE**: no pause channel exists in chat scope (`pause`/`paused` hits are all the fine-tuning pause endpoint). AI-31.1's pause-resume lossiness note is therefore **vacuous for this adapter**: no paused response can arrive, so no skipped block can be replayed wrong. | The pinned dialect gains a pause finish member. |
| `FinishReasonUnknown` | Unreachable **by design, not by omission**: the landed strict gate rejects an out-of-enum value as a typed malformed response (`S-ATS-039`), so an unrecognised stop value never becomes a completion at all (D1). | Only a deliberate reversal of D1. |

`delta.refusal` tolerance stays exactly as landed (`S-ATS-067`); this requirement changes no refusal-content
behaviour.

#### Scenarios

- **S-ACP-004** *[inspection]* — Given this table and the shipped source comments, when a reviewer resolves
  each basis, then every one names `U5` (or `C7`/`S-ATS-039` for the companion facts) and each row carries a
  stated reopen trigger rather than an open question.
- **S-ACP-005** *[test]* — Given the five-value table of S-ACP-001 driven exhaustively, when each completion's
  finish reason is compared against `{Refusal, PauseTurn, Unknown}`, then none matches; and as the negative
  control, a transcript carrying an out-of-enum finish value produces a typed malformed response rather than
  `FinishReasonUnknown`, so the "never Unknown" claim is not satisfied vacuously by absence of input.

### R-ACP-003 — A novel wire stop value stays a typed malformed response (D1 discharge split)

The adapter MUST continue to reject any `finish_reason` outside the five-member enum — including a case or
whitespace variant of a legal member — as a typed malformed response. `R-ATS-010`/`S-ATS-039` stand
unamended.

Doc 0002's AI-31.1 item 2 ("a novel vendor stop value maps to unknown without error, with the raw label
preserved") is discharged at the **neutral normalizer**, not at this adapter: `ai.NormalizeFinishReason` is
total and crash-free — an unrecognised spelling returns `FinishReasonUnknown` and never panics — which is
exactly the normalizer-crash bug class the item pins, and it was proven at AI-13. A dialect adapter that
receives an out-of-dialect value is talking to something that is not the pinned dialect, and an honest typed
failure beats silent absorption.

"The raw label preserved" has **no neutral home**: `ai.Completion` carries `reason` and `usage` and nothing
else, and widening AI-13 is out of charter. The raw label MUST instead survive in the malformed failure's own
diagnostic chain, and the adapter MUST NOT invent a completion-side carrier for it.

#### Scenarios

- **S-ACP-006** *[test]* — Given transcripts whose terminal `finish_reason` is `"STOP"`, `" stop"` and
  `"halted"`, when each is drained, then each yields a typed malformed response and no completion event, and
  `S-ATS-039` runs unmodified alongside.
- **S-ACP-007** *[inspection]* — Given `ai.NormalizeFinishReason`, when a reviewer reads it, then it is total
  over every input string, returns `FinishReasonUnknown` for an unrecognised spelling, and has no panic path —
  the discharge this split relies on, resident at the neutral layer and already tested there.
- **S-ACP-008** *[inspection]* — Given the malformed-response failure produced by S-ACP-006, when a reviewer
  reads its diagnostic chain, then the offending raw label is present in it; and when the reviewer reads
  `ai.Completion`, then it exposes no raw-label field and none was added.

### R-ACP-004 — The matched stop-sequence value is recorded as a deliberate absence

This dialect's wire carries **no** field reporting which stop sequence fired (**U4 NEGATIVE**:
`stop_sequence` → 0 hits, `stop_reason` → 0 hits; the chunk, choice and delta schemas declare no stop-related
field beyond `finish_reason` itself; `StopConfiguration` states only that "the returned text will not contain
the stop sequence"). The adapter MUST NOT derive, reconstruct or infer a matched-sequence value, and the
absence MUST be recorded in the shipped source and in this spec as deliberate and cited, never as a silent
omission.

#### Scenarios

- **S-ACP-009** *[inspection]* — Given the shipped terminal-mapping source, when a reviewer searches it for
  stop-sequence handling, then the only statement present is a `U4`-cited recorded absence, and no code path
  reads, stores or synthesises a matched-sequence value.
- **S-ACP-010** *[test]* — Given a transcript that terminates on `finish_reason: "stop"` after a request
  carrying four stop sequences, when it is drained, then the completion carries the finish reason and nothing
  identifying which sequence matched; and as the negative control, a transcript whose terminal chunk carries an
  **extra** unknown `stop_sequence` key is drained to the same completion, proving the absence assertion holds
  because nothing is read, not because nothing was offered.

### R-ACP-005 — Every usage detail field the wire reports reaches the completion event

The adapter MUST map the following (**C8**'s nested detail objects) onto `ai.Usage`, preserving the landed
absent-versus-zero discipline: a field is present in the neutral record only when its wire key arrived,
including when it arrived as an explicit `0`.

| Wire field | Neutral field | Documented relation |
| --- | --- | --- |
| `prompt_tokens` | `Usage.Input` | see R-ACP-006 |
| `completion_tokens` | `Usage.Output` | contains `Reasoning` (**U3**, cited) |
| `prompt_tokens_details.cached_tokens` | `Usage.CacheRead` | **U1 SILENT** — see R-ACP-006 |
| `prompt_tokens_details.cache_write_tokens` | `Usage.CacheWrite` | **U2 SILENT** — see R-ACP-006 |
| `completion_tokens_details.reasoning_tokens` | `Usage.Reasoning` | `Output ⊇ Reasoning` (**U3**) |

**U3 is cited and load-bearing**: the chat schema itself states, in `rejected_prediction_tokens`'s
description, that "like reasoning tokens, these tokens are still counted in the total completion tokens for
purposes of billing, output, and context window limits." This is the only usage-arithmetic sentence in chat
scope, and it makes `completion_tokens ⊇ reasoning_tokens` a schema fact — which is exactly what AI-13.4's
`Output` requires ("Output is the tokens the model produced, **including** everything counted in Reasoning").
`Reasoning` is therefore mapped with that relation stated, not merely copied.

`total_tokens` MUST stay undecoded: the neutral record has no term for it.

#### Scenarios

- **S-ACP-011** *[test]* — Given a usage chunk carrying `prompt_tokens`, `completion_tokens`,
  `prompt_tokens_details.cached_tokens`, `.cache_write_tokens` and `completion_tokens_details.reasoning_tokens`,
  when the stream is drained, then the completion's `Input`, `Output`, `CacheRead`, `CacheWrite` and
  `Reasoning` each report present with the wire's exact value.
- **S-ACP-012** *[test]* — Given the same transcript with the two detail objects removed, when it is drained,
  then `CacheRead`, `CacheWrite` and `Reasoning` all report **absent** while `Input` and `Output` are
  unchanged — the negative control for S-ACP-011, in the `S-ATS-057`/`S-ATS-060` form: the assertion that
  passes in S-ACP-011 must fail here.
- **S-ACP-013** *[test]* — Given two transcripts, one omitting `cached_tokens` and one carrying
  `"cached_tokens": 0`, when both are drained, then the two `ai.Usage` records are **not equal**, and the
  second reports `(0, true)` while the first reports absent.
- **S-ACP-014** *[test]* — Given a usage chunk with `completion_tokens: 100` and `reasoning_tokens: 40`, when
  it is drained, then `Reasoning` is 40, `Output` is 100, `Output ≥ Reasoning` holds, and `Output` is **not**
  adjusted by `Reasoning` — the U3 containment relation, asserted rather than assumed.

### R-ACP-006 — Cache semantics: map raw, and record the relation this adapter cannot attest

AI-13.4's operative sentence, quoted verbatim from `backend/agent/src/ai/usage.go`:

> "Input is the tokens of the request the provider processed fresh, excluding everything counted in CacheRead
> and CacheWrite."

and, immediately after: "This record is exclusive, so an adapter for an inclusive vendor subtracts before it
fills this field in."

**U1 and U2 are SILENT.** `prompt_tokens` is described only as "Number of tokens in the prompt";
`cached_tokens` only as "Cached tokens present in the prompt"; `cache_write_tokens` only as "The unadjusted
number of prompt tokens written to cache". The single subset statement in the whole document —
"Cached tokens **here** are counted as a subset of input tokens" — is Realtime-local and its own "here" marks
it surface-local. It is an analogy, not a citation for chat scope.

Therefore the adapter:

1. MUST map `prompt_tokens → Usage.Input`, `cached_tokens → Usage.CacheRead` and
   `cache_write_tokens → Usage.CacheWrite` as the vendor's **raw** values.
2. MUST NOT perform any arithmetic adjustment — in particular MUST NOT subtract `cached_tokens` or
   `cache_write_tokens` from `Input`. Subtracting on documented silence is inference recorded as fact, which
   `R-ATS-027` forbids.
3. MUST record, in the shipped source and in this spec, that **this adapter cannot yet attest AI-13.4's
   exclusivity relation for `Input`**: the neutral contract defines `Input` as exclusive of the cache terms,
   the wire is silent on whether `prompt_tokens` is inclusive of them, and a consumer summing
   `Input + CacheRead + CacheWrite` may therefore double-count on this dialect. The limitation is recorded, not
   papered over by an invented subtraction and not hidden by omitting the mapping.
4. MUST route the discharge obligation — pinning the relation against a real cache-hit transcript — to
   **AI-38.2**'s real-backend evidence, named in the same place as the limitation.

This is the honest branch: the alternative readings either fabricate arithmetic on silence (2) or withhold
counts the vendor actually reported (leaving `CacheRead`/`CacheWrite` unmapped), and the neutral record's own
design — "absent means the provider did not tell me" — has no way to express "reported but suppressed".

#### Scenarios

- **S-ACP-015** *[test]* — Given a usage chunk with `prompt_tokens: 1000` and `cached_tokens: 800`, when it is
  drained, then `Input` reports exactly `1000` and `CacheRead` exactly `800` — no subtraction was performed,
  and an implementation that subtracted would report `200` and fail here.
- **S-ACP-016** *[inspection]* — Given the shipped usage-mapping source and this spec, when a reviewer reads
  them, then both state the un-attested exclusivity limitation, quote or cite `U1` and `U2` as the silence that
  forces it, quote AI-13.4's `Input` sentence as the contract not yet attested, and name AI-38.2 as the
  discharge route.
- **S-ACP-017** *[test]* — Given a usage chunk whose cache-bearing values are **arithmetically impossible under
  exclusivity** — `prompt_tokens: 500` with `cached_tokens: 800`, so no exclusive reading of `Input` can be
  derived and every subtraction yields a negative — when it is drained, then the completion reports
  `Input = 500` and `CacheRead = 800` raw, no error or rejection occurs, and no count is clamped, reconciled or
  made negative. This proves the stronger claim `S-ACP-015` cannot: not merely that subtraction did not happen
  on one plausible input, but that **no consistency arithmetic is enforced anywhere on the path**, because a
  path enforcing any would have to fail, clamp or adjust here.

  > **Revision note (design-validation corrective, 2026-08-04).** This scenario previously asked for a
  > cache-hit fixture whose "recorded observation" was asserted as such. That was self-fulfilling — a
  > hand-authored fixture cannot witness a vendor's real arithmetic, and the resulting test was observably
  > identical to `S-ACP-015`. The impossible-under-exclusivity construction is the strongest raw-mapping proof
  > an authored fixture can carry, and it is distinguishable from `S-ACP-015` by construction rather than by
  > intent. Real-transcript evidence for the relation itself remains routed to AI-38.2 (clause 4 above); this
  > scenario deliberately does **not** pretend to supply it.

### R-ACP-007 — Usage-frame multiplicity: zero or one populated frame, and a merge that cannot double-count

**U6** fixes the multiplicity contract: under `stream_options: {"include_usage": true}` the spec describes
"**an additional chunk**" streamed before `data: [DONE]`, whose `usage` holds "the token usage statistics for
the entire request" and whose `choices` "will always be an empty array", while "All other chunks will also
include a `usage` field, but with a null value"; and that chunk "may not" arrive at all if the stream is
interrupted. **Zero or one populated usage frame is the complete contract**, and the cumulative-per-frame trap
doc 0002 AI-31.2 item 2 names is therefore **conventionally absent from this dialect** — it is not a schema
possibility here.

Because some servers sharing this dialect nonetheless emit several populated usage frames, the adapter MUST
keep the landed D10 rule — **last populated frame wins, overwritten wholesale, never folded** — and MUST pin
it by fixture as **dialect-conventional**, proving no double-count. The rule is the existing mechanism; this
requirement adds the proof, not a new mechanism.

#### Scenarios

- **S-ACP-018** *[test]* — Given a transcript with content chunks carrying `usage: null` followed by exactly
  one populated usage chunk with empty `choices`, when it is drained, then the completion's usage is that
  chunk's values exactly, and the preceding nulls neither reset nor contribute — U6's single-frame shape,
  behaving exactly as today.
- **S-ACP-019** *[test]* — Given a **dialect-conventional** fixture carrying three populated usage frames with
  `prompt_tokens` `10`, `20`, `30`, when it is drained, then `Input` reports `30` and never `60` — last
  populated wins, no fold, no sum.
- **S-ACP-020** *[inspection]* — Given the shipped merge source and this spec, when a reviewer reads them, then
  the multi-frame case carries an explicit dialect-conventional label with its pinning fixture named in the
  same place, and the single-frame contract cites `U6`.

### R-ACP-008 — Never invent a value, never assume a frame's position

The adapter MUST gather terminal metadata wherever in the stream it appears, MUST NOT panic on an unexpected
frame shape or position, and MUST NOT substitute a value for a field the transcript never carried. A frame
carrying only metadata and no content MUST be tolerated and MUST derive no content event.

#### Scenarios

- **S-ACP-021** *[test]* — Given a usage chunk carrying `prompt_tokens_details` with `cached_tokens` present
  and `cache_write_tokens` omitted, when it is drained, then `CacheRead` is present, `CacheWrite` is absent, no
  zero is substituted, and no panic occurs.
- **S-ACP-022** *[test]* — Given a transcript whose populated usage frame arrives **before** choice 0's
  terminal chunk rather than after it, when it is drained, then the completion still reports that frame's
  counts — position is not assumed.
- **S-ACP-023** *[test]* — Given a metadata-only frame with an empty `choices` array and no delta, when it is
  drained, then zero content events derive from it, the stream check reports no protocol violation, and the
  completion is unaffected beyond its usage.

### R-ACP-009 — The landed absent-versus-zero behaviour is frozen, not merely expected to survive

`S-ATS-055 … S-ATS-062` MUST continue to pass **unmodified**. Their transcripts carry no detail objects, so
every field this change adds is absent for them and `Input` is unchanged by an absent subtrahend. Any need to
edit one of those tests or their fixtures is a regression signal, not a maintenance step.

#### Scenarios

- **S-ACP-024** *[inspection]* — Given `backend/agent/src/ai/openaicompat/usage_test.go` before and after this
  change, when the two are diffed, then the `S-ATS-055 … S-ATS-062` test bodies, their assertions and their
  transcript fixtures are byte-identical.
- **S-ACP-025** *[test]* — Given the full package test run on every slice of this change, when it completes,
  then all eight of `S-ATS-055 … S-ATS-062` pass.

### R-ACP-010 — `CapCompletionMetadata` is not run through the conformance bridge (D3)

This change MUST NOT extend `RunConformanceFor` to `CapCompletionMetadata`, and MUST record why. The
seven-value drift guard (`finishReasonExhaustivenessCase`) runs against the `Script` fake by design; the
real-adapter bridge's `writeTerminalChunk` renders `reason.String()` straight onto the wire, so the case would
emit `"refusal"`, `"pause_turn"` and `"unknown"` — all correctly rejected by the strict gate of R-ACP-003 — and
`finish_reason/all_seven_values_reachable_drift_guarded` would fail against the real adapter. The obligation
MUST be recorded against **AI-38.2**'s expected-versus-generated capability report, where declared-capability
standing is decided. Consistent with `R-CNF-023`: a scoped run is never full-conformance evidence.

#### Scenarios

- **S-ACP-026** *[inspection]* — Given this spec and the shipped notes, when a reviewer reads the
  `CapCompletionMetadata` entry, then it states the wire-expressibility gap, names the three values the bridge
  would have to render, and routes the obligation to AI-38.2 by name.
- **S-ACP-027** *[inspection]* — Given `backend/agent/src/agenttest/` before and after this change, when the
  two are diffed, then the package is unmodified and no capability was added to the bridge's run set.

### R-ACP-011 — The doc 0002 amendments this change owes are written, dated and specific

Landing this change makes **five** statements in
`docs/architecture/milestones/0002-cachicamas-ai-layer-1-task-graph.md` misleading unless amended. The
amendments MUST be made in scope, dated, and MUST NOT rewrite history silently.

> **Revision note (design-validation corrective, 2026-08-04).** The prior text said "two statements" and
> tabled three sites. That undercounted: `R-ACP-002`'s own findings void more of doc 0002 than the table
> listed. The D2 ruling voids AI-31.1 item 3's refusal discriminator (neither branch can arise if no wire
> value spells refusal) and the AI-31.1 Note's pause-resume lossiness (which `R-ACP-002` already states is
> vacuous here). All five sites are now enumerated with their amendment actions, and `S-ACP-029`'s site list
> is extended to span the new rows.

| # | Location (doc 0002) | Amendment owed |
| --- | --- | --- |
| 1 | AI-31.1 test list, item 1 ("Every vendor stop value maps to its normalized reason — including refusal and pause to their own AI-13.1 values, never to unknown") | A dated note that the refusal/pause clause is **vacuous for this dialect** per `U5`/D2: no wire finish value spells refusal or pause, so nothing maps to those values here and nothing maps them to unknown either. The item's surviving obligation is the five-value table of `R-ACP-001`. |
| 2 | AI-31.1 test list, item 2 ("A novel vendor stop value maps to unknown without error, with the raw label preserved") | A dated amendment recording the **D1 discharge split**: the item is discharged at `ai.NormalizeFinishReason` (totality, crash-free, proven at AI-13); this dialect's schema-closed enum keeps strict rejection at the adapter (`S-ATS-039`); "the raw label preserved" has no neutral home and lives in the malformed failure's diagnostic chain. |
| 3 | AI-31.1 test list, item 3 ("Refusal after partial output and refusal before any output both normalize with the right reason and the right partial-output posture") | A dated note that the item is **unexercisable on this dialect** per `U5`/D2: no wire `finish_reason` spells refusal, so neither the after-partial-output branch nor the before-any-output branch can arise. AI-19.4's discriminator is untouched and still applies wherever a refusal outcome *can* arise; it simply cannot arise here. Reopens with `R-ACP-002`'s refusal trigger. |
| 4 | AI-31.1 Note ("pause-style finishes resume by replaying received content verbatim … If a paused response can contain block types normalization skips, resume is lossy") | A dated note that the Note is **vacuous for this adapter**: no pause channel exists in chat scope (`U5`), so no paused response can arrive and no skipped block can be replayed wrong. This restates `R-ACP-002`'s own finding at the site that raises the concern, so the two do not disagree. |
| 5 | AI-31 charter, Acceptance line ("refusal and pause map to their own values rather than to unknown") | A dated note that the clause is **unsatisfiable-and-unviolated** on this dialect per `U5`/D2, and that the charter's remaining acceptance obligation — "terminal events contain every available normalized field and never invent an unavailable one" — is carried by `R-ACP-005`…`R-ACP-008`. |

#### Scenarios

- **S-ACP-028** *[inspection]* — Given doc 0002 after this change, when a reviewer reads AI-31.1 item 2, then a
  dated amendment records the discharge split with all three of its parts and the original wording remains
  visible rather than overwritten.
- **S-ACP-029** *[inspection]* — Given doc 0002 after this change, when a reviewer reads **each of the four
  remaining sites in the table above** — AI-31.1 item 1, AI-31.1 item 3, the AI-31.1 Note, and the AI-31
  charter Acceptance line — then each carries a dated note pointing at `U5` and D2 for its vacuity or
  unexercisability on this dialect, and no site named in the table is left unamended; and when the reviewer
  compares those notes against `R-ACP-002`'s unreachability table, then the two agree on every value and
  neither asserts a reachability the other denies.

### R-ACP-012 — Every wire-shape statement resolves to a citation or a labelled pin

Every statement of wire shape in the shipped source, tests and this document MUST resolve to a claim in this
change's `citations.md` (**U1 … U6**) or the AI-28 change's (**C1 … C8**), or MUST be explicitly labelled
**dialect-conventional** with its pinning fixture named in the same place. Inference MUST NOT be recorded as
fact (`R-ATS-027`).

**Fixture convention (definitional).** This package does not keep transcripts in `testdata/` — that directory
holds only the error-map corpus, and every landed streaming transcript is an **inline string literal committed
in the test source**. That inline literal IS this package's fixture convention, and a "named pinning fixture"
therefore means **the named test function whose literal carries the transcript**. A pin naming a path under
`testdata/` would be naming a convention this package does not use.

**The dialect-conventional labels this spec introduces:**

| Label | Claim it pins | Pinning fixture (named test function) |
| --- | --- | --- |
| multi-frame usage | Several populated usage frames can arrive, beyond `U6`'s zero-or-one schema contract, and the merge must not fold them | `R-ACP-007`'s `S-ACP-019` test function and its inline three-frame transcript |

Exactly **one** such label exists in this spec. `R-ACP-006`'s cache posture is deliberately **not** one: it
rests on documented silence (`U1`, `U2`) recorded as an un-attested limitation, and `S-ACP-017`'s
impossible-under-exclusivity transcript is a synthetic probe of the adapter's own arithmetic rather than a
claim about what this dialect emits, so it asserts no wire shape needing a pin.

**IF** a third — or any further — dialect-conventional claim is introduced by a later change to this
capability, **THEN** it MUST be introduced with its pinning fixture named alongside it in the table above.
This clause creates no obligation for another label to exist.

> **Revision note (design-validation corrective, 2026-08-04).** Three corrections here. The third-label clause
> previously read "a third MUST be introduced with its pin named alongside it", whose literal English asserts
> that a third must exist; it is now explicitly conditional. The fixture convention was undefined, leaving
> `S-ACP-031`'s inspection without a fixed target; it is now definitional. And the label count fell from two to
> one as a knock-on of `S-ACP-017`'s rework: the cache-hit fixture it named was removed, and the replacement
> probe makes no wire-shape claim.

#### Scenarios

- **S-ACP-030** *[inspection]* — Given every `U`- and `C`-reference in this spec and in the shipped source
  comments, when each is resolved, then every reference names an existing claim and none names a label outside
  `U1 … U6` or `C1 … C8`.
- **S-ACP-031** *[inspection]* — Given every dialect-conventional label in this change, when a reviewer resolves
  each against the label table above, then each names a **test function** whose inline transcript literal pins
  it, that function exists in the shipped test source, and no pin names a `testdata/` path this package does not
  use; and `citations.md`'s pinned commit matches `doc.go`'s existing provenance pin — one pin, not two.
