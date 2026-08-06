# Tasks — the first provider and its transport

> **Change**: `cachicamas-ai-first-provider-decision` · Nodes AI-24.1 + AI-24.2, both `[decision]`
> Zero red-green phases — decision nodes ship no code; closes on a merged artifact, verified by inspection (design § 7).

## Review Workload Forecast

| Field | Value |
|---|---|
| Estimated changed lines | Naive ~1,900 (proposal 197 + spec 332 + design 155 + tasks ~120 + decision.md ~900 + doc0002 diff ~220 + testkit line ~3). Corrected 2–4x (this repo's confirmed undershoot) on the not-yet-written pieces: decision.md 1,800–3,600, doc0002 diff 440–880. **Realistic total ≈ 3,000–5,300 lines.** |
| 400-line budget risk | High (generic reviewer default) |
| Chained PRs recommended | No — **and structurally unavailable**: `R-APD-016` requires all doc 0002 amendments to land in the **same PR** as `decision.md`, so this change cannot be split across a decision/amendment boundary |
| Suggested split | Single PR, one child slice under the Wave-4 tracker (`feat/2026-08-03-cachicamas-ai-layer1-wave-4`) |
| Delivery strategy | auto-chain (session-cached) |
| Chain strategy | feature-branch-chain (Wave-4 level); this change is one non-split child PR within it |

Decision needed before apply: No
Chained PRs recommended: No
Chain strategy: feature-branch-chain
400-line budget risk: High

**Risk flag**: the corrected upper bound (~5,300) sits at/over the cached 5,000-line Wave-4 slice budget. Because `R-APD-016` forbids splitting decision.md from the doc 0002 amendment, the only overage remedy is maintainer `size:exception` — not chaining. Recommend the coordinator re-measure actual diff at apply time before treating this as pre-cleared.

### Suggested Work Units (commits inside the one PR)

| Unit | Goal | Focused check | Runtime harness | Rollback boundary |
|---|---|---|---|---|
| 1 | `decision.md` skeleton + AI-24.1 spine (§1–§8, tasks 1.1–2.5) | Inspection per task checks below | N/A — no code, no `make test` subject | Revert commit; no downstream code depends on it yet |
| 2 | AI-24.2 spine + closing sections (§9–§14, tasks 3.1–4.3) | Inspection per task checks below | N/A | Revert commit; independent of Unit 3 |
| 3 | doc 0002 amendments A0a–A8 + testkit status line (task 5.1) | Grep consistency pass (task 6.5/6.6) | N/A | Must revert together with Unit 1+2 in the same PR per `R-APD-016`; not independently revertible post-merge |
| 4 | Verification pass (tasks 6.1–6.7) | Inspection only, no new content | N/A | N/A — read-only pass, fixes folded back into Units 1–3 |

## Phase 1: Decision artifact foundation

- [x] 1.1 Create `decision.md`: header, § 1 (per-audience entry points), § 2 (both verdicts, one table, before any argument), § 3 (evidence rules stated once). — `R-APD-001`. Check: exactly one decision artifact exists; § 2 precedes § 4 (S-APD-001, S-APD-002).

## Phase 2: AI-24.1 spine (checklist order)

- [x] 2.1 § 4 seven-axis comparison (chosen dialect vs Anthropic-native vs one vendor-SDK class); total product, every cell cited to a vendor doc/spec clause/in-repo fact. — `R-APD-002`, `R-APD-003` (grounding). Check: S-APD-003–009.
- [x] 2.2 § 5 priced losses (AI-07 signature path, AI-11 cap enforcement), both stated contract-mandatory regardless. — `R-APD-003`. Check: S-APD-008, S-APD-009.
- [x] 2.3 § 6 four divergences answered (caching, tool-result shape, output-limit, tool-call IDs), each naming the downstream node + consequence. — `R-APD-004`. Check: S-APD-010–013.
- [x] 2.4 § 7 two further questions (tool-call-arg fragmentation; reasoning signature) — reasoning answer stated as *indicated*, AI-29.0 named owner, not pre-empted. — `R-APD-005`, `R-APD-010`. Check: S-APD-014, S-APD-015, S-APD-028, S-APD-029.
- [x] 2.5 § 8 expected capability report, 8 rows total over both closed lists, floor clause first, outcomes only `satisfied`/`absent`, `CAP-R-03` names the usage opt-in, `CAP-O-01` marked pending AI-29.0. — `R-APD-006`, `R-APD-007`, `R-APD-008`, `R-APD-009`. Check: S-APD-016–027.

## Phase 3: AI-24.2 spine (checklist order — parallel-eligible with Phase 2)

- [x] 3.1 § 9 transport decision + ADR gate recorded as evaluated/no-op; zero-requires fact stated command-verifiably; gate transferred to AI-37; AI-00.3 allowlist outcome + guard-comment routing to AI-25 stated. — `R-APD-011`, `R-APD-012`. Check: S-APD-031–036.
- [x] 3.2 § 10.1 spec-mandated framing (cited WHATWG § 9.2) / § 10.2 dialect-conventional (`[DONE]`, data-only), each with its AI-27 fixture-pin obligation; no dialect claim under § 10.1. — `R-APD-013`, `R-APD-014`. Check: S-APD-037–043.
- [x] 3.3 § 11 credential boundary: receives / never-reads (enumerated observable behaviors) / origin (Layer 3 composition root). — `R-APD-015`. Check: S-APD-044–047.

## Phase 4: Closing sections (depends on Phase 2 + 3)

- [x] 4.1 § 12 inheritance table, one row per AI-25…AI-32, in that milestone's own terms; obligations labelled with failure mode. — `R-APD-018`. Check: S-APD-058–060.
- [x] 4.2 § 13.1 usage opt-in trap · § 13.2 AI-29.0 reservation (note, not verdict) · § 13.3 Wave-2 carryover assignment (AI-41 owner, `Blocks: AI-36`, both Wave-5 scheduling reasons). — `R-APD-017` (artifact half), `R-APD-010` (§ 13.2). Check: S-APD-054–057.
- [x] 4.3 § 14 closing-checklist verification table: 7 rows both nodes, each a single contiguous span; node-status line; unblocked list AI-25…AI-32. — `R-APD-001`. Check: S-APD-001, S-APD-002.

## Phase 5: Living-graph amendment (doc 0002) — LOAD-BEARING, do not deprioritize

- [x] 5.1 Land dated amendments **A0a–A8** (2026-08-03) in `docs/architecture/milestones/0002-cachicamas-ai-layer-1-task-graph.md` per design § 4 table verbatim, plus the `ai-stream-testkit` carryover status-line edit. **A1 (the AI-25.2 guard-mechanism correction) goes first** — it blocks AI-25's SDD cycle, already in flight; do not sequence this after Phase 6. A2–A4 dated, not-applicable/no-op, **no strikethrough**. A6 appends `AI-41` (Wave 5, `Blocks: AI-36`). A8 moves the denominator 41→42. — `R-APD-016`, `R-APD-017` (doc-0002 half), `R-APD-010` (A5 is a note). Check: S-APD-048–057.

## Phase 6: Verification pass (design § 7, ranked; parallel-eligible across ranks, sequential before merge)

- [x] 6.1 Rank 1: A1 wording exact ("a call-site scan over the adapter package's own source files"), import-path/transitive-import reason stated.
- [x] 6.2 Rank 2: § 10 separation holds; every § 10.1 citation independently checked against WHATWG § 9.2's actual text (no fabricated citation) — S-APD-043.
- [x] 6.3 Rank 3: § 8 totality, floor clause opens section, standing copied from AI-03.
- [x] 6.4 Rank 4: A5 is a note not a verdict; § 13.2 names AI-29.0; no `[decision]` node struck anywhere.
- [x] 6.5 Rank 5 (atomic — do not split across commits): all five surfaces agree on **42** — header `> **Status:**` line, A8 blockquote below the header, Quick-navigation AI-41 anchor, mermaid W5 label, Delivery-sequence W5 row — plus § 13.3 present with both Wave-5 reasons, and negative check: AI-36's `Depends on:` line unedited.
- [x] 6.6 Rank 6: amendment convention — dated blockquote under every touched heading, strikethrough only on genuinely superseded claims (never A2–A4), same PR, append-only ordinals.
- [x] 6.7 Rank 7: no Go identifier of the Layer 1 contract anywhere (only stdlib import path, module manifest, vendor wire-field names permitted); zero files under `backend/`; diff is markdown-only. — `R-APD-019`. Check: S-APD-061–064.
