# Proposal — the reasoning stream: a recorded capability absence

> **Change**: `cachicamas-ai-provider-reasoning-stream`
> **Milestone**: AI-29 — Translate the reasoning stream
> **Nodes**: AI-29.0 — Reasoning emission policy `[decision]` · AI-29.1 … AI-29.3 — proposed **struck** under the living-graph clause
> **Status**: proposed
> **Phase**: proposal
> **Project**: cachicamas (witsaba)
> **Date**: 2026-08-04
> **Scope**: documentation, plus **one `_test.go` file**. Zero production Go. Zero `go.mod` change.
> **Depends on**: AI-07, AI-17, AI-24, AI-26, AI-28. **Blocks**: AI-38.
> **Node grammar**: `[decision]` — a recorded choice with a closing checklist. No production code.

---

## Intent

Close AI-29.0's checklist. doc 0002 offers two legal outcomes — emit reasoning events, or record a **documented capability absence** (AI-03.1 makes both legal). The evidence gathered for AI-24 and AI-28 points at absence, and this change records it with its grounds, its price, and its reopen triggers, so AI-38 is unblocked against a decided node rather than an open one.

## The evidence, and where it points

| # | Source | Finding |
| --- | --- | --- |
| 1 | `cachicamas-ai-provider-text-stream/citations.md` **C7** | `ChatCompletionStreamResponseDelta` at pinned commit `d4fb706…` declares **exactly five** properties — `content`, `function_call`, `tool_calls`, `role`, `refusal` — and no `required` list. **No reasoning field exists on this wire.** |
| 2 | citations **C8** | `reasoning_tokens` exists only as an optional `integer` inside `completion_tokens_details` — a **count**, not a block: no text, no signature, nothing replayable. |
| 3 | `cachicamas-ai-first-provider-decision/decision.md` § 7, § 8 (`CAP-O-01`), § 12, § 13.2 | AI-24 answered "does it sign reasoning blocks?" with **no**, predicted `CAP-O-01 = absent`, and reserved the verdict for this node without pre-empting it. |
| 4 | Landed adapter | `refuseReasoning` (policy.go) runs first inside `Translate` (translation.go:25); `reasoning_refusal_test.go` proves refusal for all three `ReasoningState`s at every message and intra-message position; `usage_test.go` (S-ATS-055, S-ATS-062) proves `Usage.Reasoning` stays **absent**; `bridge_test.go:46-73` declares `Reasoning: false` on the conformance factory. |
| 5 | Landed suite | `applyDeclaredAbsences` records `OutcomeAbsent` as a **conclusion** (S-CNF-004/038); `declaredAbsentSkipReason` skips only optional capabilities and never a required one. AI-23.8's "absent, not a silent skip" is already mechanical. |

**Assessment: absence.** No evidence found against it. The one stated overturning case — a non-standard `reasoning_content`-style extension some servers sharing this dialect emit — is not part of the pinned dialect and cannot be observed before a backend exists.

## The scheduling fact that forces the decision now

AI-29.0 says the call is "made against the exact backend chosen for AI-38/AI-39". **doc 0002 line 2221: AI-38 depends on AI-29 … AI-37; AI-39 depends on AI-38.** The backend is never named earlier than the node that is supposed to inform this one, so waiting deadlocks the graph. The resolution is to decide against the **pinned dialect** — the only artifact that exists — and route confirmation forward: AI-38.2 already compares the generated capability record to AI-24 § 8's expected one, where "a difference in either direction is a finding".

## Scope

### In scope — one PR

| Artifact | Content |
| --- | --- |
| `proposal.md` | This file |
| `specs/ai-provider-reasoning-stream/spec.md` | `R-ARS-0NN` — properties of the decision artifact, checkable by inspection, plus the one behavioral scenario below. *(Corrected 2026-08-04: this row originally named `ai-provider-reasoning-absence`/`R-ARB-0NN`; the spec landed under the change's own capability name `ai-provider-reasoning-stream` with `R-ARS-*` IDs per orchestrator directive — one capability name per change, matching the folder.)* |
| `design.md` | Structure and evidence rules for `decision.md` |
| `tasks.md` | One task per closing-checklist item, the amendment block, the verification pass |
| `decision.md` | **The deliverable**: absence, its evidence, its price, its reopen triggers, and what AI-38 / AI-40 inherit |
| `…/openaicompat/reasoning_absence_test.go` | **The only Go.** A delta carrying a `reasoning_content` extension field is ignored and never reaches a text event — the reopen trigger's case, pinned as a regression rather than asserted |
| doc 0002 | **Amended** — one dated block (below) |

### Out of scope

| Excluded | Owner | Why |
| --- | --- | --- |
| Any production Go | — | `[decision]` grammar; the absence is already implemented by refusal + declaration |
| Reasoning event *mapping* | nobody in v1 | That is the outcome being decided against |
| Changing AI-07 / AI-17's neutral reasoning contract | frozen | Contract-mandatory regardless of who emits it (AI-24 § 5) |
| Stripping reasoning before replay | **AI-40** | Recorded ruling #3: Layer 2 strips; AI-26.6 refuses |
| Re-deciding `CAP-O-01`'s expected outcome | AI-24 § 8 | This change **confirms** it; AI-38.2 asserts the generated one |

## The living-graph amendments this decision forces

Dated blockquotes, strikethrough only where a claim is genuinely superseded, landing in this PR (clause rule 4; AI-24's 2026-08-03 block is the precedent).

| # | Target | Amendment |
| --- | --- | --- |
| 1 | **AI-29 charter** | Deliverable resolves to the "documented capability absence" branch; acceptance restated in absence terms — no reasoning event is ever emitted, and no reasoning-bearing request is ever replayed |
| 2 | **AI-29.0** | Checklist closed: **absence**, with the AI-24 note's `strongly indicates` upgraded to a verdict and the reopen triggers named |
| 3 | **AI-29.1 · AI-29.2 · AI-29.3** | **Struck** — no subject for this adapter. Text stays legible for a future adapter against a signing dialect |
| 4 | **AI-21 item 3 · AI-24.1 item 3 · AI-07 out-of-scope line** | Cross-references to AI-29.1 / AI-29.2 re-pointed at this decision |
| 5 | **Traceability spine `G12(b)` · completion-checklist item 6** | See the finding below |

## The finding this change must not bury

**Completion-checklist item 6 — "round-trip tokens survive byte-exact through normalization, rebuild, and *the wire*" — loses its wire half entirely.** The Wave-2-close amendment (doc 0002 line 2347) names AI-26.6 and AI-29.2 as that half. AI-26.6 landed as a **refusal**, and this change strikes AI-29.2. Under absence, **no v1 node can close that box.** AI-24 § 5 priced this ("no v1 adapter exercises the signature-preservation path"), but no amendment has yet moved the checklist to match. This change must either restate item 6's wire clause as not-exercisable-in-v1 and publish that through AI-40.2's capability matrix, or append a node. **Recommendation: restate and publish; do not append a node for a path with no v1 consumer.** Flagged for the driver's decision.

## Capabilities

### New Capabilities
- `ai-provider-reasoning-stream` (capability name; the verdict it records is reasoning *absence*): the recorded reasoning-emission verdict for the first adapter — evidence, price, reopen triggers, and the inspection set that proves absence is coherent rather than merely unimplemented. The system under test is **the artifact**, as it was for AI-24.

### Modified Capabilities
- **None.** `ai-reasoning-content`, `ai-reasoning-events`, `ai-minimum-capabilities` and `ai-provider-conformance-suite` are cited by identifier and unchanged.

## Approach

1. **Cite the wire, do not characterize it.** Every absence claim resolves to C7/C8 at the pinned commit, or to a landed test name.
2. **Prove absence is a posture, not a hole.** Enumerate the four mechanisms already landed — translation-time refusal, `Reasoning: false` declaration, `OutcomeAbsent` recording, unmapped `reasoning_tokens` — and name the test that proves each.
3. **State the reopen triggers as observations, not intentions.** (a) The AI-38/AI-39 backend is documented to emit a reasoning delta field; (b) the dialect spec at a future pinned commit declares one. Either one reopens AI-29 and un-strikes 29.1 … 29.3.
4. **Pin the trigger's case.** One test proving a `reasoning_content` extension field is ignored and never leaks into text — the acceptance clause discharged by behavior, not by absence of code.
5. **Amend in the same PR, dated, and carry the item-6 finding into the block** rather than leaving a box that silently cannot close.

## Affected areas

| Area | Change | Risk |
| --- | --- | --- |
| `openspec/changes/cachicamas-ai-provider-reasoning-stream/` | New markdown files | None — new directory |
| `docs/architecture/milestones/0002-…-task-graph.md` | One dated amendment block, five entries | Low — dated, no silent edit |
| `backend/agent/src/ai/openaicompat/reasoning_absence_test.go` | One new test file | Low — test-only, additive |
| `backend/agent/**` production Go, `go.mod`, `go.work` | **None** | — |

## Risks

| Risk | Likelihood | Impact | Mitigation |
| --- | --- | --- | --- |
| The decision is made before the AI-38/AI-39 backend is named | **Certain** | Medium — an extension-emitting server would be discovered late | Deadlock argument stated explicitly; reopen triggers named; AI-38.2's expected-vs-generated comparison is the standing detector |
| Striking 29.1 … 29.3 reads as dropping a contract | Medium | Medium — AI-07/AI-17 look dead | Strike scoped to *this adapter*; the neutral contract is restated as frozen and consumer-less, per AI-24 § 5 |
| Checklist item 6 stays quietly unclosable | **High if unamended** | High — Layer 1 "complete" with a box no node owns | Amendment 5, surfaced above as a driver decision rather than an executor choice |
| A `reasoning_content` server is used anyway and content is silently dropped | Low | Medium — reasoning lost, not leaked | The pin test documents drop-not-leak as the deliberate behavior; AI-40's readiness contract states it |
| Absence is read as an adapter defect | Low | Low | `ai-minimum-capabilities` § 6: an adapter lacking every optional capability is fully conformant |

## Rollback plan

`git revert` of the single commit. Nothing generates from these files, nothing imports them, and the one test is additive. Partial rollback is rejected for the same reason AI-24 recorded: reverting only the doc 0002 block leaves `decision.md` citing node states that no longer match the graph. If the amendment block is rejected in review, reject the whole change and re-propose.

## Dependencies

- **AI-24** — hard. Supplies § 7's answer, `CAP-O-01`'s expected outcome, and the reservation this node discharges.
- **AI-28** — hard. Supplies C7/C8 and the undeclared-field tolerance (S-ATS-064/065/067) the pin test builds on.
- **AI-26** — hard. `refuseReasoning` is the landed absence posture on the request side.
- **AI-07 / AI-17 / AI-23.8** — read and cited, never modified.

## Success criteria

- [ ] AI-29.0's closing checklist is answered in `decision.md`: **absence**, recorded with C7/C8 and the landed-test citations.
- [ ] Both reopen triggers are stated as observable conditions with named owners.
- [ ] Every one of the four already-landed absence mechanisms is named with the test that proves it.
- [ ] doc 0002 carries one dated amendment block: charter, AI-29.0, the three struck leaves, the re-pointed cross-references, and the item-6 / `G12(b)` resolution.
- [ ] `CAP-O-01 = absent` is confirmed, not re-decided, and AI-38.2's comparison obligation is restated.
- [ ] The change adds no production Go and no `go.mod` line; `make test` stays green with one added test file.

## Proposal question round

Not asked interactively (no user channel from this phase). Three product questions whose answers would change the artifact — assumed as stated, correctable before spec/design:

1. **Item 6's box** — restate the wire clause as not-exercisable-in-v1 and publish it through AI-40.2 (assumed), or append a node so the box can close?
2. **The pin test** — keep the one `_test.go` (assumed: yes, it discharges AI-29's acceptance clause behaviorally), or ship a documentation-only change in the strict AI-24 mold?
3. **Reopen ergonomics** — is a `reasoning_content`-emitting backend a plausible near-term deployment target (assumed: no), or should the decision pre-shape a Wave-6 follow-up node for it?

## Notes for the following phases

- **`spec.md`** — system under test is the artifact; IDs `R-ARS-0NN` / `S-ARS-0NN` (corrected 2026-08-04, see Affected Areas), inspection-checkable, with exactly one behavioral scenario (the extension-field pin). Requirements must constrain the *grounds and the reopen triggers*, not only the verdict — a spec that checks only "absence is recorded" passes an artifact nobody can reverse.
- **`design.md`** — owns `decision.md`'s structure, the evidence-label rule (pinned-commit citation vs landed-test citation vs inference), and the amendment block's exact targets.
- **`tasks.md`** — no red-green phases for the `[decision]` node; one red-green pair for the pin test only. Forecast is well under the 400-line review budget.
