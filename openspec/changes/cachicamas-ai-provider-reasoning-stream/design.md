# Design — the reasoning stream: a recorded capability absence

> **Change**: `cachicamas-ai-provider-reasoning-stream`
> **Milestone**: AI-29 · **Nodes**: AI-29.0 `[decision]`; AI-29.1 … AI-29.3 struck
> **Phase**: design
> **Date**: 2026-08-04
> **Inputs**: `proposal.md` · `specs/ai-provider-reasoning-stream/spec.md` (`R-ARS-001…018`, `S-ARS-001…047`) · AI-24's `design.md`/`decision.md` (the mold) · doc 0002 · landed `backend/agent/src/ai/openaicompat/` (chunk.go, stream.go, tolerance_test.go)
> **Boundary**: requirements are the spec's. This design owns `decision.md`'s structure, the doc 0002 amendment mechanics, and the pin test's shape. No Layer 1 contract identifier appears in anything this design specifies for `decision.md` (R-ARS-018).

---

## 1. Chunk-decode verification (the one growth point) — result: not live

Read, not assumed: `decodeChunk` (chunk.go:198–204) is the **only** chunk-JSON decode path — plain `json.Unmarshal` into `wireChunk`/`wireChoice`/`wireDelta`; `wireDelta` declares only `content`. `DisallowUnknownFields` appears nowhere in the package (tolerance_test.go:22 states this as a documented property; grep confirms). The only other `json.Unmarshal` calls (failure_map.go:171/178) parse failure-status HTTP bodies, never stream chunks. Undeclared delta members are therefore ignored **on every path** — alongside content (S-ATS-064/065 already pin sibling-field tolerance) and as a delta's only member (S-ATS-067 pins the refusal-only shape, identical in mechanism). **S-ARS-038/039 are expected GREEN against landed code**; the pin test's RED is a compile/first-run capture, not a behavioral RED. If the pin test goes RED for real: **stop — no production Go lands in this change** (R-ARS-017). The defect is AI-28's (chunk decode ownership); record it as a finding, re-scope the fix to a corrective AI-28 change, and note that the absence verdict itself is unaffected — a leak would be a tolerance defect, not a capability.

## 2. Structure of `decision.md`

> **Corrective (design-validation, 2026-08-04).** (F2) R-ARS-018's restatement below now carries all three permission classes — the earlier draft dropped named landed production functions, the class § 5's mechanism column depends on. (F3) § 5 gains a fifth row homing S-ARS-035 (`declaredAbsentSkipReason` never skips a required capability), previously unhomed; re-check of the remaining scenarios found no other body-property homed by headline only (S-ARS-033/034 are rows 2/3's own subjects; S-ARS-007 is row 1's). (F4) Every § 5 row now names a landed test **function**; row 2's declaration has no landed test, so this change's one test file gains a sibling function asserting it — landed by this change's apply, carried by tasks.

AI-24's mold (checklist-ordered spine, method before application, closing verification table), sized for one two-item checklist:

```
§1  How to use this document
§2  The verdict                    ← absence, stated once; no co-equal branch (R-ARS-001/002)
§3  Evidence rules                 ← the three labels, stated once (R-ARS-008)
§4  The grounds                    ← C7's closed five-property set; C8's count-not-block (R-ARS-002)
§5  The landed absence machinery   ← four mechanisms + the required-capability guard, five rows,
                                     each with location + proving test function (R-ARS-003, R-ARS-014)
§6  The deadlock and the basis     ← doc 0002 line 2221 edge; pinned dialect; AI-38.2 routing (R-ARS-004)
§7  The price                      ← acceptance restated in absence terms; neutral contract frozen (R-ARS-006)
§8  CAP-O-01 confirmed             ← AI-24 § 8 owner; not re-derived (R-ARS-007)
§9  Reopen triggers                ← two observations, owners, un-strike procedure (R-ARS-005)
§10 What AI-38 / AI-40 inherit     ← incl. item 6's restate-and-publish via AI-40.2 (R-ARS-012 artifact side)
§11 Closing-checklist verification ← two rows: AI-29.0 items 1 and 2 → answering sections
```

**Evidence rules (§ 3)**: every wire claim carries exactly one label — **pinned-dialect citation** (`C7`/`C8`, `cachicamas-ai-provider-text-stream/citations.md`, quoted at the pinned commit), **landed-test citation** (a named test function in a file on the branch), or **inference** (with its confirming condition, normally AI-38.2).

**§ 5's five rows** (mechanism → proving test function):

| # | Mechanism | Proving test function |
| --- | --- | --- |
| 1 | Translation-time refusal — `refuseReasoning`, first inside `Translate` | `TestRefusalCases_FailWithUnsupportedCapability` (reasoning_refusal_test.go — registry-driven, all three reasoning states at every message and intra-message position, S-ART-051; satisfies S-ARS-007's every-position clause) |
| 2 | Conformance factory declares reasoning explicitly `false` (bridge_test.go:46-73) | **No landed test exists.** Ruling: this change's one test file gains a sibling function, `TestConformanceFactory_DeclaresReasoningExplicitlyFalse`, asserting the bridge factory's three optional-capability declarations (reasoning `false` among them, explicit non-nil — S-ARS-033). Landed by this change's apply; the tasks phase carries it; still one `_test.go` file, zero production Go (R-ARS-017 holds) |
| 3 | Up-front `absent` recording — `applyDeclaredAbsences` (conformance_suite.go:330) | `TestConformanceCapabilities_ReasoningDeclaredAbsent_SkippedRecordedAbsent` (S-CNF-038, reasoning's own entry) and `TestConformanceSkeleton_DeclaredCacheBoundaryAbsent_RecordsAbsentNeverNotExercised` (S-CNF-004, the mechanism generally) — homes S-ARS-034 |
| 4 | Unmapped reasoning token count — `usageFromWire` never touches the reasoning field | `TestUsage_OnlyPromptAndCompletionTokens_UnmappedFieldsReadAbsent` (S-ATS-055) and `TestUsage_NoUsageChunkAtAll_CompletesWithWhollyAbsentUsage` (S-ATS-062) |
| 5 | Required-capability guard — `declaredAbsentSkipReason` (`backend/agent/src/agenttest/conformance_suite.go:441`) returns no-skip for every required capability before consulting any declaration | `TestConformanceSkeleton_DeclaredAbsentSkipReason_NeverSkipsARequiredCapability` (conformance_suite_test.go:470-477, "required capabilities are never skippable") — homes S-ARS-035 |

**§ 9's triggers**: (1) the AI-38/AI-39 backend documents a reasoning-bearing delta field — **owner: AI-38.2** (the expected-vs-generated comparison, either-direction finding rule restated); (2) a future pinned dialect commit declares one — **owner: the next dialect re-pin's change driver**. Un-strike procedure: reopen AI-29, remove the strikethrough under a new dated amendment, AI-29.1…29.3 resume with their test lists intact. R-ARS-018's authoring constraint governs the whole artifact: Layer 1 nouns as noun phrases or cited `CAP-*`/`R-*` IDs; three permitted identifier classes, verbatim from the spec — a named landed test function, a named landed production function whose behavior is cited as evidence, and the vendor's wire-level field names.

## 3. The doc 0002 amendment plan

> **Corrective (design-validation, 2026-08-04).** (F1) B5's `Strike: none` was a contradiction: the Wave-2-close sentence at line 2347 — "Item 6 also stays open, unchanged: its wire half is AI-26.6 / AI-29.2" — IS genuinely superseded by the restate-and-publish ruling, and now receives the same strike-and-restate treatment B6 gives the analogous G12(b) sentence. Checkbox handling unchanged. (Low 1 → B7) The AI-29 mermaid graph still rendered AI-29.1…29.3 as live nodes; relabelled. (Low 2 → B8) AI-40.2 gains its own dated blockquote recording the inherited item-6 publication duty, under the node's exact title `AI-40.2 — Capability matrix and examples` — the earlier "readiness contract" naming was loose and is corrected throughout.

One convention throughout: `> **Amended 2026-08-04 (AI-29)** …` blockquote under every touched heading (R-ARS-009); strikethrough **only** for genuinely superseded claims. Exact edit list:

| # | Target (line ≈) | Strike | Blockquote content |
| --- | --- | --- | --- |
| B1 | `### AI-29` charter (1743) | acceptance clause's second half (`~~the round-trip token is captured byte-exact…~~`) | Deliverable resolves to the documented-capability-absence branch; acceptance restated in absence terms: no reasoning event emitted, no reasoning-bearing request replayed (R-ARS-010, R-ARS-006); enumerates B7's mermaid relabel as an edit this blockquote governs |
| B2 | `#### AI-29.0` (1764) | none — AI-24's 2026-08-03 note stays intact | Checklist **closed with absence**; the prior `strongly indicates` note upgraded to a verdict by this node; the against-the-exact-backend clause recorded unsatisfiable (line 2221 edge, deadlock); decided against the pinned dialect; confirmation routed to AI-38.2; both reopen triggers named; links `decision.md` |
| B3 | `#### AI-29.1 / 29.2 / 29.3` (1773–1791) | test-list bodies struck `~~…~~`, text legible, never deleted | Per node: no subject for **this adapter**; un-strike condition = either § 9 trigger fires (R-ARS-010, S-ARS-024) |
| B4 | AI-07.3 out-of-scope line (554) · AI-21.7 item 3 (1247) · AI-24.1 item 3 (1429) | inline strike of the dangling `AI-29.1`/`AI-29.2` pointer only | One-line dated blockquote under each heading re-pointing the reader at this change's `decision.md` — stating what to consult, not only that the target is gone (R-ARS-011, S-ARS-026) |
| B5 | Layer-1 completion checklist (item 6, near 2320–2347) | the Wave-2-close blockquote's superseded sentence at 2347: `~~Item 6 also stays open, unchanged: its wire half is AI-26.6 / AI-29.2.~~` — struck and restated; the checkbox itself stays open-but-resolved, and AI-17's stream half is explicitly not struck | Item 6's wire half **not exercisable in v1**; causes: AI-26.6 landed as refusal, AI-29.2 struck here; published through **AI-40.2 — Capability matrix and examples** as an obligation on that node; no node appended, and why (R-ARS-012, S-ARS-027…030) |
| B6 | `G12(b)` spine row (2402) + checklist→nodes mapping item 6 (2415) | `~~Wire-proven by AI-29.2 and AI-26.6 (Wave 4)~~`; mapping's `AI-29.2` entry struck | Blockquote under each subsection heading: the row/mapping now state the not-exercisable-in-v1 resolution with AI-40.2 as publishing owner (R-ARS-013, S-ARS-031/032) |
| B7 | AI-29 mermaid graph (1755–1762) | none — labels amended, not deleted | Relabel `X1`/`X2`/`X3` node text to carry `(struck)` — e.g. `AI-29.1 (struck)<br/>reasoning is never text` — so the rendered DAG matches the strikes; the edit is enumerated in B1's dated blockquote, the graph's own governing heading (R-ARS-009) |
| B8 | `#### AI-40.2 — Capability matrix and examples` (2298) | none (additive) | Dated blockquote recording the duty this decision routes to it: publish item 6's wire clause as not-exercisable-in-v1 alongside the Layer-2-strips-reasoning duty it already carries, so a reader arriving at AI-40.2 sees the obligation without walking back to this change (R-ARS-012, S-ARS-028) |

Milestone-level mentions of "AI-29" (lines 203, 1672, 2221, 2236 …) are **not** re-pointed: the milestone exists and is closed; only the struck-leaf references dangle.

**The S-ARS-025 grep gate, run against this table** — every `AI-29.1`/`AI-29.2` hit outside the AI-29 section, with its disposition:

| Line | Context | Disposition |
| --- | --- | --- |
| 554 | AI-07.3 out-of-scope: "survival through the wire — AI-26.6 and AI-29.2" | **Re-pointed** at this decision — B4 |
| 1247 | AI-21.7 item 3: "the same wall AI-29.1 requires of the real adapter" | **Re-pointed** at this decision — B4 |
| 1429 | AI-24.1 item 3: "does it sign reasoning blocks (drives AI-29.2)" | **Re-pointed** at this decision — B4 |
| 2347 | Wave-2-close: "Item 6 also stays open, unchanged: its wire half is AI-26.6 / AI-29.2" | **Struck and restated** — B5 (corrected: genuinely superseded) |
| 2402 | G12(b) row: "Wire-proven by AI-29.2 and AI-26.6 (Wave 4)" | **Struck and restated** — B6 |
| 2415 | Checklist→nodes mapping item 6: "… AI-26.6, AI-29.2" | **Struck** from the node list, mapping restated — B6 |

Every hit is re-pointed or struck; none dangles (S-ARS-025 passes by construction). The same grep re-runs at verify against the actual diff.

## 4. The pin test — `backend/agent/src/ai/openaicompat/reasoning_absence_test.go`

> **Corrective (design-validation, 2026-08-04).** (F5) The durable-guard row's precedent claim was wrong: tolerance_test.go's header names the **staged-and-reverted-mutation** and **comparative-twin** non-vacuity idioms — it does not name a durable guard. The permanently-executable self-inversion guard is a **new idiom this change introduces**, extending those precedents, and the row now says so. (F4) The file also gains the sibling declaration-assertion function ruled in § 2 row 2.

tolerance_test.go's home idiom exactly: same package conventions, `sseServer`/`mustClient`/`drainAll` helpers, `t.Parallel()`, scenario-ID-bearing failure messages. Two test functions ship in this one file: the pin test below, and § 2 row 2's `TestConformanceFactory_DeclaresReasoningExplicitlyFalse`.

| Decision | Choice | Rejected | Rationale |
| --- | --- | --- | --- |
| Fixture A shape | delta 1: declared content + `reasoning_content` carrying sentinel `RSN-EXT-SENTINEL-7f3a` (once in fixture, S-ARS-041); delta 2 **after** it: distinct content asserted intact and in order (S-ARS-040); terminal chunk; `[DONE]` | sentinel-free or trailing-position fixture | Content after the extension field makes an empty-remainder pass impossible (R-ARS-016) |
| S-ARS-038's terminal | comparative twin — drain the identical stream without the extension field; assert same kinds, sequences and terminal | asserting only "no error event" | The twin makes "same terminal the extension-free stream reaches" mechanical, per the keep-alive twin precedent (S-ATS-071) |
| Fixture B | one delta carrying **only** the extension field (S-ARS-039) | folding into fixture A | The boundary shape needs its own drain: no event minted for that delta, clean completion — S-ATS-067's proven shape |
| Leak assertions | a named assertion helper: no text event contains any byte of the sentinel (S-ARS-036); no reasoning-typed event by kind (S-ARS-037) | ad-hoc inline asserts | The helper is what the durable guard below inverts |
| S-ARS-042 durable guard | **in-test self-inversion, permanently executable**: a further test feeds the assertion helper a synthetic event list in which the sentinel was routed into a text delta, and requires the helper to report failure; plus one staged-and-reverted production mutation at apply time, recorded in the evidence log | staged mutation alone | A reverted mutation proves non-vacuity once; the synthetic-inversion guard proves it on every future run. This guard is a **new idiom this change introduces** — tolerance_test.go's header names only the staged-reverted-mutation and comparative-twin precedents, which this extends |

## 5. Zero-production boundary and slicing

**R-ARS-017 verification (S-ARS-043/044)**: diff scope by command — the change diff restricted to `backend/` lists exactly one added `_test.go` file; `go.mod` and `go.work` byte-identical; no build file touched (S-ARS-047). Growth path if violated: § 1's stop-and-re-scope rule.

**Slicing**: one PR (session strategy: one small PR; forecast well under budget — one test file plus markdown). **RED order**: the pin test's RED is a compile/first-run capture only — § 1's verification predicts GREEN on first run, and non-vacuity is carried by the durable inversion guard plus the recorded staged mutation. The documentation artifacts (`decision.md`, doc 0002 amendments) have **no RED phase**: `[decision]` nodes close on a merged artifact, verified by the 40 `[inspection]` scenarios, per the AI-24 precedent.

## 6. File changes

| File | Action |
| --- | --- |
| `openspec/changes/cachicamas-ai-provider-reasoning-stream/design.md` | Create (this file) |
| `…/decision.md` | Create per § 2 (apply) |
| `…/tasks.md` | Create (tasks) |
| `docs/architecture/milestones/0002-…-task-graph.md` | Amend per § 3, B1–B6 (apply) |
| `backend/agent/src/ai/openaicompat/reasoning_absence_test.go` | Create per § 4 (apply) |
| Any production Go, `go.mod`, `go.work`, existing `openspec/specs/**` | **None** (R-ARS-017/018) |

## 7. Threat matrix

N/A — no routing, shell, subprocess, VCS/PR automation, executable-file classification, or process-integration boundary. Documentation plus one additive test.

## 8. Open questions

None blocking. The proposal's three assumed answers stand and the spec binds them: item 6 resolves by restate-and-publish through AI-40.2 (R-ARS-012), the pin test ships (R-ARS-015), and no Wave-6 pre-shaping occurs (reopen handled by R-ARS-005's triggers).

## 9. Next phase

`tasks.md` — one task per AI-29.0 checklist item, one amendment task covering B1–B8 with the S-ARS-025 grep check, the pin-test task carrying the inversion-guard obligation and § 2 row 2's declaration-assertion sibling function, and the inspection verification pass. RED phases: pin test only.
