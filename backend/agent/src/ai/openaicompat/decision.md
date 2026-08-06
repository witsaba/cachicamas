# Decision — the reasoning stream: a recorded capability absence

> **Change**: `cachicamas-ai-provider-reasoning-stream`
> **Milestone**: AI-29 — Translate the reasoning stream
> **Node**: AI-29.0 — Reasoning emission policy `[decision]` · AI-29.1 · AI-29.2 · AI-29.3 struck by this change
> **Status**: decided
> **Date**: 2026-08-04
> **Project**: cachicamas (witsaba) · **Target package**: none — `[decision]` nodes ship no code
> **Closes**: doc 0002's AI-29.0 closing checklist (two items)
> **Sources**: [doc 0002 — Layer 1 task graph](../../../docs/architecture/milestones/0002-cachicamas-ai-layer-1-task-graph.md) § AI-29 · [AI-24's decision record](../../../cachicamas-ai-first-provider-decision/decision.md) §§ 5, 7, 8, 12, 13.2 · [AI-24's spec](../../../cachicamas-ai-first-provider-decision/specs/ai-first-provider-decision/spec.md) (the `R-APD` artifact-as-subject model this decision follows) · [the pinned-dialect citations](../../../cachicamas-ai-provider-text-stream/citations.md) **C7**, **C8** · [the v1 capability set](../../specs/ai-minimum-capabilities/spec.md) — `CAP-O-01`, § 6, § 10
> **Binding predecessor**: AI-24's `decision.md` § 7 (`strongly indicates` indication), § 8 (`CAP-O-01` expected `absent`, **pending AI-29.0's confirmation**), and § 13.2 (the AI-29.0 reservation this node discharges)

> [!IMPORTANT]
> **This artifact decides standings and evidence, not code.** No Go type name, field name, method name, interface name or package identifier belonging to the Layer 1 contract appears here. Three identifier classes are permitted: a named landed test function, a named landed production function whose behavior is being cited as evidence, and the vendor's wire-level field names. The closing checklist and the evidence rule require all three (`R-ARS-018`).

---

## 1. How to use this document

**If you are writing AI-38:** § 9 names you as the confirming node. § 10 restates the either-direction finding rule that compares your generated capability record against AI-24 § 8's expected one.

**If you are writing AI-40:** § 11 records the publication duty AI-40.2 inherits — completion-checklist item 6's wire clause as not-exercisable-in-v1, alongside the Layer-2-strips-reasoning duty that node already carries.

**If you are writing a future adapter against a signing dialect:** § 9 names both reopen triggers. § 4's struck text remains legible for you. The un-strike procedure is stated once in § 9 and is the inverse of this amendment.

**If you are reviewing this artifact:** § 12 walks AI-29.0's two-item closing checklist against it, item by item. § 5 is where a missing row or a mis-marked mechanism is most expensive.

**Every reference to an existing spec below is by identifier only** — `CAP-O-01`, `R-ARS-*`, `R-APD-*` resolve to [the v1 capability set](../../specs/ai-minimum-capabilities/spec.md) and [the AI-29 spec](../specs/ai-provider-reasoning-stream/spec.md). Nothing in either is redefined or modified here.

---

## 2. The verdict

Both answers, before any argument, for the reader who came for one.

| Question | Verdict |
| --- | --- |
| Does v1 emit reasoning events for the first provider? | **No — documented capability absence.** The first adapter emits no reasoning event of any kind, for any reasoning-bearing request, at any position in the stream. |
| What follows for AI-29.1 … AI-29.3 and AI-23.8's capability outcome? | **AI-29.1, AI-29.2 and AI-29.3 are struck** under the living-graph clause (§ 4); **AI-23.8 records `absent` as the capability outcome** — a result, not a gap (§ 5, § 8). |

The verdict's grounds are the pinned dialect's own schema (§ 4). The verdict's price is restated in absence terms (§ 7). The verdict's reopen path is two observable triggers, each with a named owner (§ 9).

**The verdict was made against the pinned dialect** (§ 6), because the exact backend named in AI-29.0's own text does not exist at decision time — doc 0002 line 2221 makes AI-38 depend on AI-29, so waiting deadlocks the graph. Confirmation is routed forward to AI-38.2's expected-versus-generated comparison (§ 9, § 10).

---

## 3. Evidence rules

Three labels bind every wire claim in this artifact. Stated once, so no section restates its own standard of proof.

1. **Pinned-dialect citation.** A wire claim that resolves to **C7** or **C8** at the commit AI-24 pinned, as quoted in `cachicamas-ai-provider-text-stream/citations.md`. The claim carries the citation identifier inline.
2. **Landed-test citation.** A claim resolved to a named test function in a file already on the branch — the test is the proof, the file path the location. The claim carries the function name inline.
3. **Inference.** A claim that is neither of the above. It must state what would confirm it, and that condition is normally AI-38.2's expected-versus-generated comparison (§ 9, § 10).

An unlabelled wire claim is a defect. A wire claim labelled a pinned-dialect citation when neither citation states it is a defect — the verdict's entire grounds are the pinned dialect's own schema.

---

## 4. The grounds

The verdict's grounds are the pinned dialect's own schema — what the wire actually carries, observed at the pinned commit AI-24 recorded.

**C7, the streamed delta schema.** The `ChatCompletionStreamResponseDelta` schema at the pinned commit declares **exactly five properties**: `content`, `function_call`, `tool_calls`, `role`, `refusal`. It declares no `required` list. **No reasoning field exists on this wire** (pinned-dialect citation, C7). The five named properties are exhaustive of what the schema permits a delta to carry: there is no `reasoning`, `reasoning_content`, `thinking` or cognate field at the schema level, and a JSON decoder that consults only the schema will not surface one.

**C8, the reasoning-shaped datum that does exist.** `reasoning_tokens` exists only as an optional `integer` inside `completion_tokens_details` (pinned-dialect citation, C8). It is a **count**, not a block. It carries no text, no signature, and nothing a `%#v`-style round trip could replay. A count and a block are distinguished explicitly, so a reader cannot mistake the reported count for a partial capability: one is a number, the other is content with a cryptographic seal, and the two are not degrees of the same thing.

**The combination: a count is not a block, and a closed set with no reasoning field is not a partial field.** C7's closed five-property set means there is no wire-level reasoning content to decode even in principle; C8's count means even what the dialect reports is not the thing the request-side contract asks for. The verdict of absence rests on both together.

**The overturning case, stated plainly.** Some servers sharing this dialect emit a non-standard `reasoning_content`-style extension field that is **not** part of the shared dialect itself. AI-24 § 7 named this case; this change restates it: a `reasoning_content` extension inside a delta object is the only concrete shape that could overturn this verdict, and it cannot be observed before a backend exists. § 9 names both reopen triggers, both of which are observable rather than planned.

---

## 5. The landed absence machinery

**Closing-checklist item 1's posture half.** Four mechanisms already implement the absence in landed code. Each is named below with the production location that implements it and the named landed test that proves it. The absence is therefore a posture the adapter actively takes, not a unimplemented node — the difference matters, because an unimplemented node is indistinguishable from "nobody wrote it" while a posture is mechanically enforced.

| # | Mechanism | Location | Proving test function |
| --- | --- | --- | --- |
| 1 | Translation-time refusal — `refuseReasoning` runs first inside `Translate`, before any body assembly, walking the whole request and failing on every reasoning part at every message position and every intra-message position | `backend/agent/src/ai/openaicompat/policy.go` (the `refuseReasoning` function) and `backend/agent/src/ai/openaicompat/translation.go` (the call site, the first statement of `Translate`) | `TestRefusalCases_FailWithUnsupportedCapability` (landed-test citation, `backend/agent/src/ai/openaicompat/reasoning_refusal_test.go` — registry-driven, all three reasoning states at every message and intra-message position; `S-ART-051`/`S-ARS-007`'s every-position clause) |
| 2 | Conformance factory declares reasoning explicitly `false` — a non-nil pointer to `false`, distinguishable from omission | `backend/agent/src/ai/openaicompat/bridge_test.go:46-73` (the `conformanceBridgeFactory` function's `Reasoning: &reasoningOffered` declaration, with `reasoningOffered = false`) | **No landed test exists for the declaration itself.** Landed by this change: `TestConformanceFactory_DeclaresReasoningExplicitlyFalse` (in this change's `reasoning_absence_test.go`), asserting the bridge factory's three optional-capability declarations — reasoning `false`, token counting `false`, cache boundary `false` — each an explicit non-nil pointer, not an unset field (landed-test citation, `S-ARS-033`) |
| 3 | Up-front `absent` recording — `applyDeclaredAbsences` runs before any case for the capability, marking it `OutcomeAbsent` from the declaration alone, independently of whether any case for that capability is registered | `backend/agent/src/agenttest/conformance_suite.go:330` (the `applyDeclaredAbsences` function) | `TestConformanceCapabilities_ReasoningDeclaredAbsent_SkippedRecordedAbsent` (landed-test citation, `backend/agent/src/agenttest/conformance_suite_test.go` — reasoning's own absent entry, `S-CNF-038`) and `TestConformanceSkeleton_DeclaredCacheBoundaryAbsent_RecordsAbsentNeverNotExercised` (landed-test citation, `backend/agent/src/agenttest/conformance_suite_test.go` — the up-front mechanism generally, `S-CNF-004`; homes `S-ARS-034`) |
| 4 | Unmapped reasoning token count — `usageFromWire` never reads the `reasoning_tokens` field; the `Reasoning` field on the neutral usage type stays absent whenever the wire reports a count | `backend/agent/src/ai/openaicompat/usage.go` (`usageFromWire`) and `backend/agent/src/ai/openaicompat/usage_test.go` (the proving tests) | `TestUsage_OnlyPromptAndCompletionTokens_UnmappedFieldsReadAbsent` (landed-test citation, `backend/agent/src/ai/openaicompat/usage_test.go:54` — `S-ATS-055`) and `TestUsage_NoUsageChunkAtAll_CompletesWithWhollyAbsentUsage` (landed-test citation, `backend/agent/src/ai/openaicompat/usage_test.go:267` — `S-ATS-062`) |
| 5 | Required-capability guard — `declaredAbsentSkipReason` returns no-skip for every required capability before consulting any declaration, so the skip-and-record path is reachable **only** for an optional capability, never for a required one | `backend/agent/src/agenttest/conformance_suite.go:441` (the `declaredAbsentSkipReason` function) | `TestConformanceSkeleton_DeclaredAbsentSkipReason_NeverSkipsARequiredCapability` (landed-test citation, `backend/agent/src/agenttest/conformance_suite_test.go:470-477` — `R-CNF-004`, `S-CNF-011`; homes `S-ARS-035`) |

The first four rows are the absence machinery itself; the fifth is the guard that keeps the machinery from silently swallowing a required capability. Together they prove that absence is a posture: every layer that could reach reasoning, at request, at factory declaration, at capability outcome, and at usage field, takes the absent branch mechanically.

**Reading the four rows honestly.** Three are tests already shipped on the branch (`TestRefusalCases_FailWithUnsupportedCapability`, the absent recording pair, the unmapped-fields pair); one is the `declaredAbsentSkipReason` guard. None of the four was written for this change. The fifth row's guard was written for AI-23.8 — this change is the first artifact that names it as part of the reasoning absence machinery.

---

## 6. The deadlock and the basis

**Closing-checklist item 2 (decision basis).** AI-29.0's text defers the decision to "the exact backend chosen for AI-38/AI-39", and doc 0002 line 2221 records that **AI-38 depends on AI-29 … AI-37** (`Depends on: AI-23, AI-29 … AI-37. · Blocks: AI-39, AI-40.`). The dependency edge makes the deferral unsatisfiable as written: waiting for the backend named by a downstream node deadlocks the graph.

The decision is therefore made against the **pinned dialect** — the only artifact that exists at decision time — and not presented as confirmed against any backend. The pinned dialect is named as the basis; the absence of a named backend is recorded, not glossed over.

Confirmation is routed forward to **AI-38.2** — the expected-versus-generated capability comparison AI-24 § 8 already armed, where "a difference in either direction is a finding": an unexpected `absent` on an entry expected `satisfied` is a regression, and an unexpected `satisfied` on an entry expected `absent` means the adapter grew a capability nobody reviewed. AI-38.2 owns the comparison, not this artifact.

---

## 7. The price

**Closing-checklist item 2 (price half).** The AI-29 charter's acceptance clause loses its second half in v1. Restated in absence terms so it remains checkable:

- **Original acceptance (AI-29 charter, `decision.md`'s parent artifact § 4):** *"Provider reasoning never leaks into text events; the round-trip token is captured byte-exact and replays through AI-26.6 unchanged."*
- **Absence-terms restatement (this decision):** No reasoning event is emitted by this adapter, for any reasoning-bearing request, at any position in the stream; no reasoning-bearing request is replayed — AI-26.6's refusal is the only path that touches reasoning, and it fails before any byte reaches the wire.

The restatement is checkable by the four mechanisms in § 5: a translation-time refusal for any reasoning part (row 1), an explicit `false` declaration that no test or behavior will trip (row 2), an up-front `absent` outcome recorded before any case for the capability runs (row 3), and an unmapped `reasoning_tokens` count (row 4). A reviewer can verify each independently without a mapping existing.

**The neutral reasoning contract stays contract-mandatory and frozen.** AI-07's neutral reasoning shape — byte-exact capture, byte-exact replay — and AI-17's stream-half byte-exact preservation are **not** deprecated by this decision. The strike is scoped to **this adapter**: the dialect signs no reasoning blocks (§ 4), so no v1 adapter exercises the signature-preservation path. A future adapter against a signing dialect — Anthropic's `thinking` blocks, or any dialect that surfaces a reasoned block on the wire — needs the neutral shape unchanged, and AI-24 § 5 priced this loss in the same artifact as the win.

The strike is therefore a loss of the first exercising consumer, not a loss of the contract. Per AI-24 § 3's fairness rule, an orphaned consumer is not a deprecated contract; the contract remains mandatory independently of who emits it.

---

## 8. `CAP-O-01` confirmed

**Closing-checklist item 1 (capability outcome half).** AI-24 § 8 named `CAP-O-01`'s expected outcome as **`absent`** — the dialect signs no reasoning blocks, only an opaque `reasoning_tokens` count — and marked the entry **pending AI-29.0's confirmation**. This decision **confirms** that expected outcome.

Confirmation, not re-derivation: the `absent` expected outcome was stated by AI-24, with its basis recorded there. This artifact does not restate the basis — it states only that the basis AI-24 recorded (the dialect's no-signed-blocks observation, the count-not-block observation) holds under this decision's grounds (§ 4), and that the verdict of absence is therefore consistent with AI-24's expected outcome.

**Absence is a legal outcome for an optional capability.** Per the v1 capability set § 6: *"an adapter that lacks every one of these [optional capabilities] is fully conformant."* Three recorded absences in AI-24 § 8 (`CAP-O-01`, `CAP-O-02`, `CAP-O-03`) are not an under-powered vendor; they are the expected, conformant shape of this specific dialect. This change adds no new absence beyond those three.

**AI-38.2's standing obligation is restated.** The expected-versus-generated capability comparison must assert the **generated** record against this expected outcome. A `CAP-O-01 = satisfied` entry in the generated record, against this verdict, would mean the adapter grew a capability nobody reviewed — a finding, not a silent pass.

---

## 9. Reopen triggers

**Closing-checklist item 1 (reopen half).** Two triggers reopen AI-29 and un-strike AI-29.1 … AI-29.3. Each is stated as an **observation**, not an intention — a condition someone can check, with a named owner, so a trigger nobody watches cannot fire.

| # | Trigger | Owner |
| --- | --- | --- |
| 1 | The backend selected for **AI-38 / AI-39** is documented to carry a reasoning-bearing field on its streamed delta — i.e., its API documentation or its emitted wire bytes, observed against the pinned dialect, include a reasoning-shaped field that C7's closed five-property set does not name | **AI-38.2** — the expected-versus-generated capability comparison; the same mechanism that confirms the verdict (§ 8, § 10) is the mechanism that detects its overturning case |
| 2 | The dialect schema at a **future pinned commit** declares a reasoning-bearing field on its streamed delta — i.e., a re-pin of the dialect, recorded in the same citation shape AI-24 used, names a reasoning property where C7 named five | The **next dialect re-pin's change driver** — whoever next writes `cachicamas-ai-provider-text-stream/citations.md` with a re-pinned C7 carrying a reasoning field |

Either trigger alone reopens AI-29. The un-strike procedure is the inverse of this amendment:

1. A new dated amendment lands in doc 0002 under the AI-29 heading, naming the trigger that fired and the observation that proved it.
2. The struck text under AI-29.1, AI-29.2 and AI-29.3 loses its `~~…~~` strikethrough under the same amendment; the text stays legible throughout.
3. AI-29.0's checklist reopens; this artifact is superseded by a new one carrying the affirmative verdict and § 5's five-row mechanism table expanded to cover the now-extant reasoning event.

The pinned-dialect citation that would force trigger #2 to fire is **C7 itself at a future commit**; the citation that would force trigger #1 to fire is **AI-38.2's generated capability record, with `CAP-O-01 = satisfied`** against AI-24 § 8's expected `absent`. Both are checkable now: trigger #1 against AI-38.2's eventual run; trigger #2 against any re-pin's published citations.

---

## 10. What AI-38 inherits

doc 0002: AI-38 *"Depends on: AI-23, AI-29 … AI-37."* AI-38 inherits one standing obligation from this decision; it is written in AI-38's own terms, sufficient to plan it without reopening this decision.

| Milestone | What it inherits |
| --- | --- |
| **AI-38.2** | § 8's confirmed `CAP-O-01 = absent` expected outcome as the comparison's expected half, with the either-direction finding rule restated: a generated `CAP-O-01 = satisfied` is a finding, not a silent pass — the adapter grew a capability nobody reviewed. § 9 trigger #1 lives here, observed against the same comparison. |

AI-38 inherits nothing else from this decision. The four landed mechanisms in § 5 are not AI-38's concern; they were the AI-23.8 and AI-26 suites' delivery. AI-38's first concrete adapter is the first moment the absence verdict is observable from outside, which is exactly why § 9 routes trigger #1 through AI-38.2.

---

## 11. What AI-40 inherits

doc 0002: AI-40 *"Goal: Freeze the v1 surface `cachicamas_agent` may consume. Deliverable: Runnable package examples, a compatibility statement, the supported-capability matrix, and a fake-provider example for future Layer 2 tests."* AI-40 inherits one additive publication duty; it is written for **AI-40.2 — Capability matrix and examples**, the leaf that already carries the supported-capability matrix from AI-38.2.

| Milestone | What it inherits |
| --- | --- |
| **AI-40.2** | Completion-checklist item 6's wire clause, restated as **not exercisable in v1**, alongside the Layer-2-strips-reasoning duty this node already carries. The cause is named: AI-26.6 landed as a refusal and AI-29.2 is struck by this change. The clause is restated as an obligation on AI-40.2 — a reader arriving at that node sees the obligation without walking back to this change. The clause is not closed by appending a new milestone or leaf; the path has no v1 consumer, and a node with no consumer does not earn an identifier (§ 12 restates this resolution as a finding the amendment records). |

The stream half of item 6, already closed by AI-17 (`R-ARE-009`/`R-ARE-010` — the `G12(b)` spine row's first three-quarter close), is **not** struck and **not** reopened: restating the wire half does not read as reopening a closed property. AI-40.2's existing `G12(b)` row already carries the stream-half closure; the inherited wire-half restatement sits beside it, not in place of it.

---

## 12. Closing-checklist verification

AI-29.0's two closing-checklist items, walked against this artifact.

| # | Item (doc 0002's own words) | Where answered | Status |
| --- | --- | --- | --- |
| 1 | Record whether v1 emits reasoning events for the first provider or documents a capability absence — AI-03.1 makes both legal, and every sibling node below assumes emission | § 2 (verdict), § 4 (grounds), § 5 (machinery), § 8 (`CAP-O-01` confirmed) | **answered** — documented capability absence, recorded with C7 and C8 and the landed-test citations, and AI-23.8 records "absent" as this adapter's capability outcome (§ 8) |
| 2 | If absence wins: AI-29.1 … AI-29.3 are struck under the living-graph clause and AI-23.8 records "absent" as this adapter's capability outcome, which is a result rather than a gap | § 5 row 3 (AI-23.8's `applyDeclaredAbsences` records absent as a conclusion, not a gap), § 8 (capability outcome confirmed), doc 0002's amendment B3 (the three leaves struck, with test lists legible) | **answered** — three leaves struck legibly, AI-23.8's outcome is the absent conclusion recorded by `applyDeclaredAbsences`, both with named landed-test citations |

**Milestone acceptance, restated and checked.** doc 0002's AI-29 acceptance clause, original text: *"Provider reasoning never leaks into text events; the round-trip token is captured byte-exact and replays through AI-26.6 unchanged."* Restated in absence terms (§ 7): no reasoning event is emitted by this adapter, for any reasoning-bearing request, at any position in the stream; no reasoning-bearing request is replayed. The restatement is checkable by § 5's four mechanisms without a mapping existing.

**Node status.** AI-29.0 closes on merge of this artifact. Per doc 0002's node grammar, a `[decision]` leaf produces no production code and closes when "the decision artifact answers every listed question and is merged." No `make test` gate applies to this decision — this change touches nothing under `backend/` except the additive `reasoning_absence_test.go` file (§ 5 row 2's landed test function).

**Unblocked by this decision:** AI-38 (per § 10), AI-40 (per § 11), and through them in turn AI-39 and the AI-41 carryovers. AI-29.1, AI-29.2 and AI-29.3 are struck legibly under doc 0002's living-graph clause (amendment B3) and reopen only on the conditions named in § 9.

**Not closed by this decision:** completion-checklist item 6's wire half. The clause is restated as not-exercisable-in-v1 (§ 11) and published through AI-40.2; no new milestone or leaf identifier is appended to close it, because the path has no v1 consumer and a node with no consumer does not earn an identifier. This resolution is recorded here, by name, so a reader arriving at item 6 does not read the unchecked box as a missing piece of work.