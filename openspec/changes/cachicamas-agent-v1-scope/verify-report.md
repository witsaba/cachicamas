```yaml
schema: gentle-ai.verify-result/v1
evidence_revision: sha256:faa9dfcc355201ab16ae27a22ec9711b95ee1e54c41bc294ec14156a987c821b
verdict: fail
blockers: 0
critical_findings: 0
requirements: 11/15
scenarios: 56/60
test_command: grep -n 'Closes:' docs/architecture/milestones/0003-cachicamas-agent-layer-2-task-graph.md
test_exit_code: 0
test_output_hash: sha256:07c5b4fd8bcac4f5560bb623785297d16f793bb8b625b11c8ca6648d96c39b26
build_command: git status --short
build_exit_code: 0
build_output_hash: sha256:f0c3127a0054726fada8af57b1d7de2feb475dad52379dc755f1c1e93f2f6035
```

# Verify report — the Layer 2 v1 capability scope

> **Change**: `cachicamas-agent-v1-scope`
> **Milestone**: AG-02 · **Node**: AG-02.1 — The scope decision `[decision]`
> **Phase**: verify
> **Verdict**: **PASS WITH WARNINGS** — 0 CRITICAL · 5 WARNING · 4 SUGGESTION · 0 closure-blocking
> **Machine envelope verdict**: `fail`, `blockers: 0`, `critical_findings: 0`. The `gentle-ai.verify-result/v1` envelope admits only `pass`/`fail`, and refuses `pass` whenever the evidence is incomplete. Four of sixty scenarios do not pass *as written* (§ 12), so `pass` would be a verdict this report did not derive. `blockers: 0` records the substantive finding: **nothing here blocks closure.**
> **Date**: 2026-08-11
> **Worktree**: `cachicamas-worktrees/agent-layer2-wave0` · **Branch**: `feat/agent-layer2-wave0`
> **Artifact under verification**: `decision.md`, 554 lines, `sha256:faa9dfcc…c821b`
> **Mode**: `[decision]` leaf — no production code, no `make test` gate (doc 0003 node grammar). Every check below is inspection, and **every claim the apply phase made was re-derived by command against the source rather than taken on the artifact's word**.

---

## 0. What this verification did not inherit

The apply phase reported nine claims. Each was treated as a hypothesis to defeat and re-run from the source documents. Seven survived unchanged, one was corrected, and one was refined:

| Apply's claim | This verification |
| --- | --- |
| 20 verdict identifiers (`AGS-I` 8, `AGS-S` 3, `AGS-D` 3, `AGS-X` 5) | **Corrected — 19, not 20.** 8 + 3 + 3 + 5 = 19. `decision.md` § 2.4 itself states "14 verdicts + 5 cross-checks = **19** entries" and is correct; the relayed figure was off by one. Confirmed by `grep -o 'AGS-[ISDX]-[0-9][0-9]' \| sort -u` → 19 distinct identifiers. |
| ADR 0005 § D4 covers G1, G3, G5 only; G7 and G11 bare | **Confirmed**, row by row, against the § D4 table. See § 4. |
| F3 touches exactly five entries (four inclusions + one explicit exclusion) | **Confirmed.** |
| Zero Go identifiers | **Confirmed for camel/Pascal identifiers** (positive control run first); **qualified** for package paths — see SUGGESTION S4. |
| 27 distinct `VL2-*` citations all resolve in AG-00's register | **Confirmed** — 27/27, each with ≥1 defining row. |
| Forward pass clean on all 14 rows | **Confirmed**, R-10 … R-18 quoted verbatim against doc 0003 lines 2208–2216. |
| Reverse pass built from exactly the 20 `Closes:` lines | **Confirmed** — my own grep returns the same 20 milestones. |
| Orphan check confirms all 6 doc 0004 node citations | **Confirmed** — all 11 `CO-*` identifiers exist; doc 0004's own spine (lines 906–911) independently states every mapping. |
| S-AGS-035's AG-14/AG-21 gap is a spec precision note, not a defect | **Conclusion upheld, stated reason rejected as over-broad.** See § 6. |

---

## 1. Deliverable and task completeness

```
$ find openspec/changes/cachicamas-agent-v1-scope -type f | sort
./decision.md
./design.md
./explore.md
./proposal.md
./specs/agent-v1-scope/spec.md
./tasks.md

$ find . -name 'decision.md' | wc -l
1
```

| Item | Status |
| --- | --- |
| `decision.md` present, singular | ✅ exactly one |
| Tasks checked / unchecked | **42 / 0** — `grep -c '^- \[x\]'` = 42, `grep -c '^- \[ \]'` = 0 |
| Spec requirements | 15 (`grep -c '^## R-AGS-'`) |
| Spec scenarios | 60 (`grep -c '^- \*\*S-AGS-'`; 60 distinct ids) |
| Verdict identifiers in `decision.md` | 19 (`AGS-I` 8, `AGS-S` 3, `AGS-D` 3, `AGS-X` 5) |

No task blocks verification.

---

## 2. Diff scope and hygiene (build gate)

```
$ git status --short
?? openspec/changes/cachicamas-agent-contract-vocabulary/
?? openspec/changes/cachicamas-agent-event-delivery/
?? openspec/changes/cachicamas-agent-v1-scope/

$ git diff --stat HEAD
(empty — zero tracked-file modifications)
```

Zero edits to any merged document: doc 0001, doc 0003, doc 0004 and every ADR are untouched. Six new files under the change directory. **S-AGS-046 and S-AGS-059 hold.**

### The no-Go-identifier constraint, with a positive control first

The regex was validated against a real Go file before being trusted against the artifacts:

```
$ grep -ohE '\b[a-z][a-z0-9]*[A-Z][A-Za-z0-9]*\b|\b[A-Z][a-z0-9]+[A-Z][A-Za-z0-9]*\b' \
    backend/agent/src/agenttest/cache_boundary_test.go | sort -u | wc -l
52          # control fires: CacheBoundaries, ErrOutOfRange, MarkCacheBoundary, …

$ same regex over decision.md design.md explore.md proposal.md tasks.md specs/agent-v1-scope/spec.md
(zero hits in every file)
```

**Zero camel-case or Pascal-case identifiers in any of the six artifacts.** Method-shaped names (`Foo()`) and Go package paths were scanned separately — see SUGGESTION S4 for the one qualification.

---

## 3. `R-AGS-002` — the total walk, re-derived from doc 0001 § 7

Doc 0001 § 7's register is lines 695–707: thirteen rows, G1 … G13. The ownership test ("does `L2` appear anywhere in the Owner column?") was re-applied by hand to each Owner cell:

| G | Owner column (doc 0001 § 7, verbatim) | L2-owned? | `decision.md` § 2.1 outcome |
| --- | --- | --- | --- |
| G1 | `L2 protocol, L3 policy` | yes | `AGS-I-01` |
| G2 | `L3` | no | `AGS-X-01` |
| G3 | `L2` | yes | `AGS-I-02`, `AGS-S-01`, `AGS-D-03` |
| G4 | `L1 places, L2 stabilises` | yes (L2 half) | `AGS-I-03` |
| G5 | `L2` | yes | `AGS-I-04` |
| G6 | `L3` | no | `AGS-X-02` |
| G7 | `L2` | yes | `AGS-I-05`, `AGS-S-02`, `AGS-D-01` |
| G8 | `L1 taxonomy, L2 policy` | yes (L2 half) | `AGS-I-06`, `AGS-S-03`, `AGS-D-02` |
| G9 | `L1` | no | `AGS-X-03` |
| G10 | `L2 emits, L3 prices` | yes (L2 half) | `AGS-I-07` |
| G11 | `L2 + L3` | yes | `AGS-I-08` |
| G12 | `split` | no | `AGS-X-04` |
| G13 | `L1` | no | `AGS-X-05` |

**Independently derived counts: 8 owned + 5 not owned = 13.** These are the artifact's own stated counts (§ 2.2), and they are correct. No register row is absent from the walk; none is duplicated.

**Charter cross-check (S-AGS-008).** All eight documented defaults in doc 0003's AG-02 charter (line 251) were located verbatim, one grep each — `documented default: implement — it is unretrofittable`, `default: implement, AG-18`, `default: implement, AG-08.2`, `G5 parallel tools (default: implement)`, `prove re-entrancy, ship no subagent tool`, `default: retry in v1, failover as a named seam`, `default: implement cost events`, `default: taxonomy complete` — each returning exactly 1. **Eight defaults, eight owned rows, one to one.**

---

## 4. The ADR 0005 § D4 per-citation check — the highest-value check in this verification

`docs/adr/0005-promote-agent-stack-to-own-module.md` § D4 (lines 280–294) was opened and its `v1 verdict` column read row by row:

| § D4 row | Line | Exact `v1 verdict` cell |
| --- | --- | --- |
| G1 | 282 | `Seam now, implement in L2` |
| G3 | 284 | `Seam now, implement in L2` |
| G5 | 286 | `Seam now, implement in L2` |
| **G7** | 288 | `Seam now` — **bare** |
| **G11** | 292 | `Seam now` — **bare** |

Now the reverse direction: where does `decision.md` cite the fuller phrasing?

| Entry | Register row | Cites `"Seam now, implement in L2"`? | Source supports it? |
| --- | --- | --- | --- |
| `AGS-I-01` (line 185) | G1 | yes | ✅ |
| `AGS-I-02` (line 192) | G3 | yes | ✅ |
| `AGS-I-04` (line 206) | G5 | yes | ✅ |
| `AGS-I-05` (line 213) | G7 | **no** — states the cell "reads bare `Seam now`, with no `implement in L2` suffix" and rests on the charter default + AG-19 | ✅ correct refusal |
| `AGS-I-08` (line 234) | G11 | **no** — defers to § 8's precision note, which rests on the charter default + AG-08/AG-20 | ✅ correct refusal |

**No citation the source does not contain.** § 8's F3 table (lines 392–397) reproduces the five cells exactly as they read. F3's own closing paragraph (line 405) separates the two cautions correctly: G11 is corrected *inside* F3 because F3's rebuttal could be misread as covering it uniformly; G7 needs no correction because F3 never claimed it. **`S-AGS-044` passes on evidence, not on assertion.**

Corroborating milestones named by S-AGS-044 — AG-10, AG-18, AG-09 (for G1/G3/G5) and AG-08, AG-20 (for G11) — are all present in F3, and all five exist in doc 0003 as `###` milestone headings.

---

## 5. Both audit passes, re-run independently

### 5.1 Forward pass — `AGS-*` declared nodes vs doc 0003's Traceability spine

Every R-row quoted in `decision.md` § 9.1 was compared byte-for-byte against `docs/architecture/milestones/0003-…md` lines 2208–2216:

| Verdict | Declared nodes | Spine row (source, verbatim) | Agreement |
| --- | --- | --- | --- |
| `AGS-I-01` | AG-06.1, AG-10.1–10.4 | R-10 `AG-06.1, AG-10.1, AG-10.2, AG-10.3, AG-10.4; policy half → doc 0004` | ✅ |
| `AGS-I-02` + `AGS-S-01` | AG-06.4, AG-18.1–18.5 ∪ AG-17.1, AG-17.2 | R-11 `AG-06.4, AG-17.1, AG-17.2, AG-18.1, AG-18.2, AG-18.3, AG-18.4, AG-18.5` | ✅ union = 8 of 8 |
| `AGS-I-03` | AG-08.2 (+AG-12.1) | R-12 `AG-08.2; history append-only AG-12.1; …` | ✅ |
| `AGS-I-04` | AG-09.2, AG-09.3 | R-13 `AG-09.2, AG-09.3; ordinal from doc 0002 AI-30` | ✅ |
| `AGS-I-05` + `AGS-S-02` | AG-06.3, AG-19.1–19.3 | R-14 `AG-06.3, AG-19.1, AG-19.2, AG-19.3; production tool deferred` | ✅ |
| `AGS-I-06` + `AGS-S-03` | AG-15.1, AG-15.2 ∪ AG-15.3 | R-15 `AG-15.1, AG-15.2, AG-15.3; loop-never-retries AG-11.2` | ✅ (see SUGGESTION S1) |
| `AGS-I-07` | AG-06.2, AG-16.1 | R-16 `AG-06.2, AG-16.1; compaction spend AG-18.1; pricing → doc 0004` | ✅ |
| `AGS-I-08` | AG-08.1, AG-20.1, AG-20.2 | R-17 `AG-08.1, AG-20.1, AG-20.2` | ✅ |
| `AGS-D-01/02/03` | AG-19 / AG-15.3 / AG-18.1 | Deferred table rows 2186 / 2187 / 2188 | ✅ verbatim |

**All 14 rows clean — independently confirmed.** No disagreement between the verdict-declared column and the spine column.

**Node existence (S-AGS-028, S-AGS-029).** All **32** distinct `AG-NN.N` identifiers cited anywhere in `decision.md` were checked against doc 0003; every one resolves to a `#### AG-NN.N` node heading. Zero dangling references.

### 5.2 Reverse pass — my own grep

```
$ grep -n 'Closes:' docs/architecture/milestones/0003-cachicamas-agent-layer-2-task-graph.md
203:  … Closes: R-20's start condition …                            → AG-00
331:  … Closes: R-01, R-02, R-03 mechanically …                     → AG-03
412:  … Closes: R-04 (lifecycle families), R-05 …                   → AG-04
520:  … Closes: R-04 (the two high-volume families).                → AG-05
604:  … Closes: R-04 — the four families … G1, G10, G7, G3's …      → AG-06
773:  … Closes: R-06's stateless core …                             → AG-07
835:  … Closes: R-12; seam 1 of v2 § 6 …                            → AG-08
904:  … Closes: **G5** (R-13); seams 2 and 3's Layer 2 anchor …     → AG-09
1007: … Closes: **G1**'s protocol half (R-10); seam 2 …             → AG-10
1115: … Closes: R-08's typed mid-stream path …                      → AG-11
1229: … Closes: R-07's boundary enforcement …                       → AG-12
1296: … Closes: R-08's driving loop …                               → AG-13
1446: … Closes: **G8**'s Layer 2 half (R-15) …                      → AG-15
1529: … Closes: **G10**'s Layer 2 half (R-16) …                     → AG-16
1601: … Closes: seams 5 and 6 (R-18) …                              → AG-17
1656: … Closes: **G3** (R-11) …                                     → AG-18
1794: … Closes: **G7**'s structural half (R-14); seam 12 …          → AG-19
1865: … Closes: **G11** (R-17) …                                    → AG-20
2044: … Closes: R-19 …                                              → AG-22
2096: … Closes: R-21 …                                              → AG-23

$ grep -c 'Closes:' …
20
```

**My row count: 20.** Identical set to `decision.md` § 9.2's twenty rows. Every returned milestone appears in the table; every table row corresponds to a returned line; every `Closes:` string in the table is a faithful (elided) quotation of the source line. **No disagreement between the two corroborating columns in either pass.** Closing-checklist item 2 is satisfied.

The four milestones with **no** `Closes:` field are AG-01 (line 224), AG-02 (line 246), AG-14 (line 1373) and AG-21 (line 1964) — read directly, not inferred.

---

## 6. Adjudication: `S-AGS-035`, AG-14 and AG-21

The apply phase self-reported this and declined to treat it as closure-blocking. It was re-derived from scratch, and the reasoning was tested rather than inherited.

**The facts, by command.** `S-AGS-035` lists nine "foundational milestones" — AG-03, AG-04, AG-05, AG-12, AG-13, **AG-14**, **AG-21**, AG-22, AG-23 — whose reverse-pass rows should name a base-architecture identifier. The grep above returns neither AG-14 nor AG-21. Both charter lines were read directly: AG-14 (line 1373) reads `SDD change: cachicamas-agent-cancellation-tree · Interrupt ≠ shutdown, and both ≠ deadline.` and AG-21 (line 1964) reads `SDD change: cachicamas-agent-concurrency-hardening · The AI-33/AI-34 of this layer, over the whole assembly.` — no `Closes:` field on either.

**Verdict: a spec-scenario imprecision, not a decision defect.** `R-AGS-009` defines the reverse pass's domain as "one row per doc 0003 milestone carrying a `Closes:` field". AG-14 and AG-21 carry none, so they correctly have no row; inventing one would require fabricating a citation the source does not contain — precisely the error F1 and F3 exist to prevent. `decision.md` § 9.2's precision note (line 479) records this openly instead of smoothing it. That is the right disposition.

**Where the apply phase's stated reason does not survive.** It wrote that this "reflects no inconsistency inside doc 0003's graph", and grouped AG-14 and AG-21 with AG-01 and AG-02 as one phenomenon. Neither holds:

| Milestone | `Closes:`? | doc 0003 "Nodes trace back to scope" (line 2243+) | Does the **forward** "Requirements → closing nodes" table reach it? |
| --- | --- | --- | --- |
| AG-01 | no | R-05 (invariant 3), R-09 | **yes** — `AG-01.1` appears in R-05 (line 2203) *and* R-09 (line 2207) |
| AG-02 | no | "v2 § 7's register" — not an R-row | n/a, and none expected |
| **AG-14** | no | **R-08** (wind-down), v2 § 4.2 | **no** — R-08 (line 2206) names no AG-14 node. `AG-14.1` appears only under **R-09** — a *different* requirement from the one the reverse table names |
| **AG-21** | no | **R-05** (invariant 3 under pressure) | **no** — R-05 (line 2203) names no AG-21 node, and no `AG-21.x` appears anywhere in the forward table (verified by grep over lines 2195–2220) |

Doc 0003's own TOC (line 47) advertises the spine as "Requirement → node, **two-way**". It is one-way for AG-21 and directionally mismatched for AG-14. So there *is* a small asymmetry inside doc 0003's graph — the apply phase's stated reason is over-broad, and AG-01/AG-02 are not the same phenomenon (AG-01 is fully two-way; AG-02 targets a register rather than an R-row).

**Is it closure-blocking? No.** `R-AGS-010`'s rule is scoped to (i) a disagreement between the forward pass's two columns and (ii) a reverse-pass mismatch. Neither obtains. AG-14 and AG-21 close no G-concern and bear no verdict, so no verdict's evidence depends on them. The apply phase's **conclusion** stands; its **reason** is replaced by the table above. Recorded as WARNING W3, with a doc 0003 follow-up suggestion.

---

## 7. `R-AGS-007` — the seam account

R-18's own text, verified verbatim at doc 0003 line 72: *"Seams 1, 2, 3, 5, 6, 7, 8 and 12 of v2 § 6 exist in v1, each with its default named."*

The eight required seams in `decision.md` § 7.1 match doc 0003's R-18 spine row (line 2216) **node for node**: 1→AG-08.1, 2→AG-10.1, 3→AG-09.1, 5→AG-17.1, 6→AG-17.2, 7→AG-11.2+AG-15.1, 8→AG-15.3, 12→AG-19.1–19.3. Doc 0001 § 6 (lines 651–664) defines exactly twelve seams; every seam name in § 7.1 and § 7.2 matches its catalog row.

**Two distinct omission reasons, as required:**

- **Reason 1 — Layer 1 contract items already shipped:** seams 9 (`AI-12`), 10 (`AI-10`/`AI-11`), 11 (`AI-07`). The § 6 grouping quotation (doc 0001 lines 670–672) is faithful, including the correct handling of seam 1 (which is in the grouping but *is* required, so the artifact says so explicitly).
- **Reason 2 — Layer 3's, not Layer 1's:** seam 4, per G6's owner, doc 0004 `CO-02.1`, named as **the exception** to § 6's own Layer-1-urgency grouping.

All seven `AI-NN` milestones cited (AI-02, AI-07, AI-10, AI-11, AI-12, AI-13, AI-18) exist as headings in doc 0002. **8 + 4 = 12, no seam unaccounted. R-AGS-007 holds.**

---

## 8. `R-AGS-012` — the no-orphan check, re-verified against doc 0004 itself

All eleven `CO-*` identifiers cited by `decision.md` exist in `docs/architecture/milestones/0004-…md`: `CO-02.1`, `CO-03.1`, `CO-03.2`, `CO-04.1`, `CO-05.1`, `CO-08.2`, `CO-16.1`, `CO-18.1`, `CO-20.2`, `CO-24.1`, `CO-24.2` — each with ≥2 occurrences.

Doc 0004's own Traceability spine (lines 906–911) independently states every one of the six mappings, and `decision.md` § 10 quotes it accurately:

| Concern | doc 0004 spine, verbatim (line) |
| --- | --- |
| G1 policy half | `CO-03.1, CO-03.2; persistence CO-16.1; UI CO-20.2` (906) |
| G2 | `CO-04.1; consulted by CO-06 … CO-08; process-tree kill CO-08.2` (907) |
| G6 | `CO-02.1` (908) |
| G10 Layer 3 half | `CO-05.1, CO-18.1` (909) |
| G11 Layer 3 half | `CO-24.1, CO-24.2` (910) |
| G4 payoff | `CO-24.1` (911) |

**No orphan. The corroboration is genuinely independent** — it is doc 0004's own table, not this decision's cross-reference.

---

## 9. G3's four structural defenses against the conflation reading

Each was checked at its own arrival point, not merely counted:

| # | Arrival point | Present? | Does it block the reading *there*? |
| --- | --- | --- | --- |
| 1 | The lists (§ 2.4) | ✅ | Yes — G3 yields `AGS-I-02`, `AGS-S-01` and `AGS-D-03` across three separate lists. There is no single "G3 verdict" token to cite, so the citation that would carry the conflation is unwritable. |
| 2 | The entry (§ 5.2, line 245) | ✅ | Yes — `AGS-S-01`'s negative clause names the misreading in its own text, in bold, and cross-references `AGS-I-02` by identifier. A reader who arrives at the seam entry alone cannot leave with the conflation. |
| 3 | The audit (§ 9.1, line 428) | ✅ | Yes — `AGS-I-02`'s forward-pass row lists `AG-18.1`–`18.5` as the evidence the verdict's validity *rests on*. A reader checking the audit meets AG-18 as a dependency, not as discretionary work. |
| 4 | The inheritance table (§ 11, line 513) | ✅ | Yes — AG-18's row states its five leaves are "**obligations, not options**", in that milestone's own terms, and repeats that it holds regardless of whether the strategy ever triggers compaction. |

**All four exist and each blocks the reading at its own arrival point.** `S-AGS-016` and `S-AGS-051` hold.

---

## 10. `R-AGS-013` — the seven inheritance rows

| Milestone | Row kind | Verdict identifiers named | Verdict |
| --- | --- | --- | --- |
| AG-17 | full | `AGS-S-01`, seam 5, seam 6 | ✅ in AG-17's own terms |
| AG-18 | full | `AGS-I-02`, `AGS-D-03`, AG-18.1–18.5 | ✅ "obligations, not options" |
| AG-19 | full | `AGS-I-05`, `AGS-S-02`, `AGS-D-01` | ✅ three identifiers, three postures |
| AG-09 | pointer | `AGS-I-04` | ✅ |
| AG-15 | pointer | `AGS-I-06`, `AGS-S-03` | ✅ |
| AG-16 | pointer | `AGS-I-07` | ✅ |
| AG-20 | pointer | `AGS-I-08` | ✅ |

**Three full + four pointers = seven.** Completeness re-derived independently: every doc 0003 milestone whose charter depends on AG-02 (`Depends on:` fields at lines 1608, 1634, 1663, 1694, 1801, 1829 → AG-17, AG-18, AG-19, AG-20) carries a row, as do the three "Explicitly deferred" rows whose "Decided by" column cites AG-02. **`S-AGS-052` holds — no milestone citing AG-02 is left without a governing identifier.**

---

## 11. Requirement-by-requirement verdict

| Requirement | Verdict | Deciding evidence |
| --- | --- | --- |
| `R-AGS-001` | **PASS** | one `decision.md`; § 13 maps both checklist items with status; § 1 and § 13 both state the `[decision]`-leaf gate |
| `R-AGS-002` | **PASS** | 13-row walk re-derived from doc 0001 lines 695–707; 8 + 5 = 13; eight charter defaults located verbatim |
| `R-AGS-003` | **PASS** | every `AGS-I`/`AGS-S`/`AGS-D` entry carries identifier, row citation, obliges, does-not-oblige, discharging node; `AGS-S` adds trivial implementation + seam number |
| `R-AGS-004` | **PARTIAL** | splitting rule stated once (§ 4.2); G3's split named with its misreading; **but** per-row entry counts are 3/3/3, not `S-AGS-014`'s 2/2/3, and § 4.2's table lists 2 halves for G3 and G8 against 3 for G7 — **W2** |
| `R-AGS-005` | **PASS** | five `AGS-X` entries with the exact owners and nodes required; class stated as cross-check; G13 footnote names AI-02 |
| `R-AGS-006` | **PASS** | doc 0003 lines 2186–2188 are exactly the three rows citing AG-02; three `AGS-D` entries, one to one; the other three rows' deciders quoted correctly |
| `R-AGS-007` | **PASS** | eight seams node-for-node against R-18's spine row; two distinct omission reasons; 8 + 4 = 12 |
| `R-AGS-008` | **PASS** | 14 rows, one per verdict; R-10 … R-17 verified verbatim; all 32 cited AG nodes exist |
| `R-AGS-009` | **PARTIAL** | reverse pass reproducible and correct at 20/20 both ways; **but** `S-AGS-035`'s premise is false for AG-14 and AG-21 — **W3** — and § 9.2's derived 9/11 split is wrong — **W1** |
| `R-AGS-010` | **PASS** | § 9.3 states the rule, names checklist item 2 as authority, refuses the recorded-risk route, derives the result after the tables; F1–F4 distinguished from graph mismatches |
| `R-AGS-011` | **PARTIAL** | standing rule stated; F2/F3/F4 each open with the opposing reading; § D4 cited only where the source supports it; **but** F1 does not state its opposing reading affirmatively first and omits the strongest counter-evidence — **W4** |
| `R-AGS-012` | **PASS** | six-row orphan table; 11/11 `CO-*` nodes exist; doc 0004's own spine independently corroborates all six |
| `R-AGS-013` | **PASS** | 3 full + 4 pointer rows; completeness re-derived from doc 0003's `Depends on:` fields |
| `R-AGS-014` | **PASS** | § 12: append-only, no renumbering, struck-through supersession, count updates, two revision routes, F1's third narrower case (see S3) |
| `R-AGS-015` | **PARTIAL** | no Layer 3 semantics decided; 27/27 `VL2-*` resolve; zero camel/Pascal identifiers; **but** `S-AGS-058`'s "package path" clause has two literal hits — **S4** |

**11 of 15 fully verified; 4 partial; 0 failed.**

---

## 12. Scenario-by-scenario

`✅` pass · `◐` partial · `⊘` judged by inspecting the detection mechanism, as the spec itself directs

| Scenario | Verdict | Deciding evidence |
| --- | --- | --- |
| S-AGS-001 | ✅ | `find` → one `decision.md`; the only `AGS-*-NN` tokens outside it are namespace *examples* (tasks.md:65, spec.md:24), not normative restatements |
| S-AGS-002 | ✅ | § 13 table: item 1 → § 5 (+§ 2, § 4, § 6), item 2 → § 9; both "Satisfied" |
| S-AGS-003 | ✅ | decision.md:22 and :552 — no production code, closes on merge, no `make test` gate |
| S-AGS-004 | ✅ | all thirteen rows walked in register order; none absent (§ 3 above) |
| S-AGS-005 | ✅ | § 3 states the Owner-column rule in re-appliable form |
| S-AGS-006 | ✅ | eight and five named individually; 8 + 5 = 13, re-derived independently |
| S-AGS-007 | ⊘ ✅ | **Stated against a hypothetical defective artifact — judged by inspecting the detection mechanism, not by producing the defect.** The mechanism is the register-ordered walk (§ 2.1) plus two named counts (§ 2.2): deleting a row leaves a visible gap in the ordered table *and* falsifies "8 + 5 = 13". Both are checkable without the reviewer noticing an absent paragraph. Mechanism adequate. |
| S-AGS-008 | ✅ | eight charter defaults located verbatim, one grep each; one-to-one with the eight owned rows |
| S-AGS-009 | ✅ | all 14 entries carry the six parts. Note: class is carried by the section header plus the identifier's class letter rather than a labeled per-entry field, and `AGS-D` names its node under "Held by" — both satisfy the requirement's substance |
| S-AGS-010 | ✅ | `AGS-S-01` "never compacts" seam 5; `AGS-S-02` "no subagents exist" seam 12; `AGS-S-03` "no failover occurs" seam 8 |
| S-AGS-011 | ✅ | decision.md:213 quotes doc 0001:746 verbatim: *"Sandboxing, MCP, and subagents. Seams 3, 4 and 12 exist; the implementations do not."* |
| S-AGS-012 | ⊘ ✅ | every entry carries an explicit "Does not oblige" bullet; an entry lacking it fails the S-AGS-009 read. Mechanism adequate |
| S-AGS-013 | ✅ | § 4.2 states the rule once, before § 5's entries; § 2.4's summary tables carry a forward pointer plus the count statement |
| S-AGS-014 | ◐ | **Counts do not match.** Actual entries per row: G3 = 3 (`AGS-I-02`, `AGS-S-01`, `AGS-D-03`), G7 = 3, G8 = 3 (`AGS-I-06`, `AGS-S-03`, `AGS-D-02`). The scenario expects 2/2/3. The substantive clauses ("each with its own identifier and class", "no half left implicit") **do** hold. See W2 |
| S-AGS-015 | ✅ | `AGS-I-02` → AG-18.1–18.5; `AGS-S-01` → "never compact", seam 5, AG-17.1 |
| S-AGS-016 | ✅ | § 4.3 states the chain step by step and names it wrong |
| S-AGS-017 | ✅ | § 2.4 and § 4.3 both state that 14 ≠ 8 because three rows split |
| S-AGS-018 | ✅ | G2→CO-04.1, G6→CO-02.1, G9→AI-12, G12→AI-07/AI-13/AI-18, G13→AI-02 — all present |
| S-AGS-019 | ✅ | § 6 opening: "not verdicts … so 'eight verdicted rows' is a demonstrated count rather than a claim" |
| S-AGS-020 | ✅ | `AGS-X-05` footnote names AI-02 and calls AG-01's self-description an analogy |
| S-AGS-021 | ✅ | doc 0003:2186–2188 are exactly the three AG-02-decided rows; 2189–2191 cite other deciders, quoted correctly in § 5.3 |
| S-AGS-022 | ✅ | AG-19 / AG-15.3 / AG-18.1 named; reasons stated as what deferral protects |
| S-AGS-023 | ✅ | node-for-node against doc 0003:2216, all eight |
| S-AGS-024 | ✅ | seams 9/10/11 → AI-12 / AI-10+AI-11 / AI-07; seam 4 → Layer 3 per G6's owner |
| S-AGS-025 | ✅ | § 7.2 names seam 4 "the **exception**" to § 6's grouping |
| S-AGS-026 | ✅ | § 7.3: 8 + 4 = 12 against doc 0001 § 6's twelve rows |
| S-AGS-027 | ✅ | 14 rows, one per verdict identifier |
| S-AGS-028 | ✅ | every `AGS-I` row names ≥1 milestone; all 32 cited node ids resolve to doc 0003 headings |
| S-AGS-029 | ✅ | all three `AGS-S` rows name seam-bearing nodes that exist |
| S-AGS-030 | ✅ | R-10 (G1), R-11 (G3), R-13 (G5), R-14 (G7), R-15 (G8), R-16 (G10), R-17 (G11) all verified verbatim |
| S-AGS-031 | ✅ | § 9.1 closing paragraph states the column's purpose |
| S-AGS-032 | ✅ | the exact grep is printed as a fenced command |
| S-AGS-033 | ✅ | my own grep returns the same 20 milestones; both directions match |
| S-AGS-034 | ✅ | every G-naming row maps to a covered verdict; none names an unverdicted G-concern |
| S-AGS-035 | ◐ | 7 of the 9 named milestones have rows naming a base identifier; AG-14 and AG-21 have no `Closes:` field and correctly have no row. Spec imprecision — see § 6 and W3 |
| S-AGS-036 | ✅ | § 9.3 names closing-checklist item 2 as the authority |
| S-AGS-037 | ✅ | § 9.3: "not a note, not a risk carried forward" |
| S-AGS-038 | ✅ | result stated after the tables, explicitly as their conclusion |
| S-AGS-039 | ✅ | § 8 preamble and § 9.3 both distinguish F1–F4 from graph mismatches |
| S-AGS-040 | ✅ | § 8 preamble states the standing rule once |
| S-AGS-041 | ◐ | F2, F3 and F4 each open "**The literal reading, stated first**". **F1 does not** — it opens "The defect is real", and never airs the strongest opposing reading. See W4 |
| S-AGS-042 | ✅ | doc 0001:699 (G5, Seam `2`) and :695 (G1, Seam `2`) both verified directly; G13's empty seam cell verified at :707; verdict taken from R-13; doc 0001 amendment named as the follow-up route |
| S-AGS-043 | ✅ | doc 0001:711–715 quoted verbatim; `AGS-I-07` implement-now, token-only |
| S-AGS-044 | ✅ | **§ 4 above.** § D4 read row by row; fuller phrasing cited only for G1/G3/G5; explicitly refused for G7 and G11; all five corroborating milestones named and existent |
| S-AGS-045 | ✅ | F4 recorded as analogy, consistent with S-AGS-020 |
| S-AGS-046 | ✅ | `git diff --stat HEAD` empty; F1 recorded not repaired, with the authority reason |
| S-AGS-047 | ✅ | six-row orphan table covers every Layer 3 assignment the decision makes |
| S-AGS-048 | ✅ | all 11 `CO-*` nodes exist; doc 0004 spine lines 906–911 corroborate all six mappings |
| S-AGS-049 | ⊘ ✅ | **Stated against a hypothetical — judged by inspecting the detection mechanism.** The table is keyed by *assignment*, so an assignment with no node produces a row with an empty node cell rather than silence in prose. Mechanism adequate |
| S-AGS-050 | ✅ | AG-17, AG-18, AG-19 full rows naming their verdict identifiers |
| S-AGS-051 | ✅ | AG-18's row: "obligations, not options" |
| S-AGS-052 | ✅ | four pointer rows; no AG-02-citing milestone left without one (re-derived from doc 0003's `Depends on:` fields) |
| S-AGS-053 | ✅ | § 12 states the amendment route and prohibits the local downstream verdict |
| S-AGS-054 | ✅ | append-only, no renumbering, struck-through, counts updated, AI-03 § 13 + the 2026-08-10 amendment cited. Precedent verified real — see S3 for where it lives |
| S-AGS-055 | ✅ | no sandbox semantics, permission policy content or token→money conversion decided; owning doc 0004 node named in each case |
| S-AGS-056 | ✅ | deletion test applied to every Layer 3 mention; each is an owner name plus a node identifier |
| S-AGS-057 | ✅ | 27 distinct `VL2-*` citations, 27/27 resolving to defining rows in AG-00's register |
| S-AGS-058 | ◐ | zero camel/Pascal identifiers in all six files (positive control fired at 52 on a real Go file); but the "package path" clause has two literal hits — see S4 |
| S-AGS-059 | ✅ | only markdown under the change dir; zero tracked-file modifications; nothing under `backend/` |
| S-AGS-060 | ✅ | § 1's namespace paragraph, using `AGS-I-01` (a real entry) as its example |

**56 pass · 4 partial · 0 fail.** Three scenarios (`S-AGS-007`, `S-AGS-012`, `S-AGS-049`) were judged by inspecting the detection mechanism, because they are stated against a hypothetical defective artifact and are not verifiable by producing the defect. This is said explicitly rather than scored as an ordinary pass.

---

## 13. Issues

### CRITICAL — none

No verdict, node citation, seam mapping, audit result, orphan assignment or inheritance row failed verification. Nothing found here blocks the closing checklist: item 1 and item 2 are both satisfied on evidence.

### WARNING

**W1 — § 9.2's derived split contradicts its own table.** Line 477 states: *"**Nine** of the twenty close a base-architecture identifier only … **eleven** name a register row."* Counting the twenty rows of the table immediately above it: base-identifier-only = **10** (AG-00, AG-03, AG-04, AG-05, AG-07, AG-11, AG-12, AG-13, AG-22, AG-23); register-naming = **10** (AG-06, AG-08, AG-09, AG-10, AG-15, AG-16, AG-17, AG-18, AG-19, AG-20). Both stated numbers are off by one in opposite directions, which is why they still sum to twenty. This is a derived count that disagrees with the table it summarizes, in the paragraph whose whole purpose is to derive the result *from* the tables (`S-AGS-038`). The pass/fail outcome of the reverse pass is unaffected — every row still matches both ways.

**W2 — `S-AGS-014`'s per-row counts are unachievable, and § 4.2's table is internally inconsistent.** The scenario expects G8 = 2, G3 = 2, G7 = 3 verdict entries. The artifact has 3/3/3, because `R-AGS-006` separately mandates three `AGS-D` entries and they land on G7, G8 and G3 respectively. The spec is therefore in tension with itself, and the artifact resolved it in favour of the more specific, enumerated requirement — the right call. But § 4.2's splitting table then lists **three** halves for G7 while listing only **two** for G3 and for G8, contradicting § 2.1's outcome column, § 2.4's lists, § 4.3's "8 + 3 + 3 = 14" and § 5's entries. Either § 4.2's table gains the two missing halves, or `S-AGS-014` is amended at archive time. This needs a decision, not a silent pass.

**W3 — `S-AGS-035`'s premise is false for AG-14 and AG-21, and the apply phase's stated reason is over-broad.** Full adjudication in § 6. The conclusion (not closure-blocking) is upheld; the reason ("no inconsistency inside doc 0003's graph"; AG-01/AG-02 the same phenomenon) is not. Doc 0003's spine, advertised as two-way, is one-way for AG-21 and directionally mismatched for AG-14. AG-01, by contrast, is fully two-way.

**W4 — F1 does not state its opposing reading affirmatively first, and omits the strongest counter-evidence.** `S-AGS-041` requires the opposed reading be stated affirmatively, with its own citation, before the answer. F2, F3 and F4 each open with a bolded "The literal reading, stated first". F1 opens with "**The defect is real**" — the disposition — and its heading ("cites Seam '2', **which is wrong**") pre-announces the answer. More materially, F1 asserts that "the catalog contains no seam for parallel-tool scheduling" without engaging the strongest case for doc 0001's cell: doc 0001 § 6 line 654 puts seam 2's **Lives on** at *"the tool-scheduling path in the loop"*, doc 0003's AG-09 charter (line 904) calls AG-09 — the G5 milestone — *"seams 2 and 3's Layer 2 anchor"*, and doc 0003's own reverse spine (line 2252) maps `AG-09 → R-13, R-18 (seams 2–3)`. Two independent doc 0003 statements corroborate a seam-2 association for exactly the concern F1 says has none. F1's *disposition* is safe either way — `AGS-I-04`'s verdict is taken from R-13 and states it does not reproduce the seam cell — but F1 overstates the certainty of the defect it records.

**W5 — the vocabulary dependency is unmerged.** All 27 `VL2-*` citations resolve only into `openspec/changes/cachicamas-agent-contract-vocabulary/`, an untracked sibling change in the same wave. Every normative Layer 2 noun in `decision.md` therefore depends on AG-00 landing in the same pull request. This is the intended wave-0 design, not a defect, but it is a hard delivery coupling: `decision.md` has no resolvable vocabulary if AG-00 is split out.

### SUGGESTION

**S1 — § 9.1's `AGS-I-06` row says its union with `AGS-S-03` equals "R-15's full set exactly (3 of 3)".** R-15's cell (doc 0003:2213) reads `AG-15.1, AG-15.2, AG-15.3; loop-never-retries AG-11.2` — four node identifiers. `AGS-I-06`'s entry does name AG-11.2 as corroborating, so nothing is unaccounted, but "full set" is imprecise.

**S2 — the doc 0001 § 7 disposition-rule quotation is compressed.** F2 and F3 render it as *"A row marked seam now reserves the place and no further work happens in v1."* The source (doc 0001:687–689) reads *"A row marked* seam now *means [§ 6](#…) reserves the place and no further work happens in v1."* The meaning is preserved; the italic quotation marks imply verbatim.

**S3 — § 12's AI-03 precedent points at a document that does not carry the amendment.** The claim (§ 12: *"AI-03 § 13's discipline — itself exercised once, 2026-08-10, appending `CAP-O-04`"*) is **true**, but the amendment lives in the promoted canonical spec `openspec/specs/ai-minimum-capabilities/spec.md` (line 23's dated blockquote; line 84's `CAP-O-04` row; line 604's `~~all three~~ **all four**` struck-through count) — not in `openspec/changes/archive/2026-07-31-cachicamas-ai-minimum-capabilities/decision.md`, which the artifact's own header links as "AI-03's decision" and which contains neither `CAP-O-04` nor `2026-08-10`. A reader following the header link will not find the precedent. Naming the promoted spec would close this.

**S4 — `S-AGS-058`'s "package path" clause is stricter than the constraint it derives from.** Two literal hits exist: `backend/agent/` (decision.md:8) and `backend/agent/src/{ai,agenttest}/**` (explore.md:163). Doc 0003's actual authoring constraint (line 15) forbids *"type names, field names, or signatures"* and says nothing about paths — and doc 0003 itself writes `backend/agent/src/agent/` (line 21) and `src/{ai,agenttest,handoff}` (line 92). This reads as spec over-reach rather than an artifact defect; worth reconciling at archive time so the promoted spec does not carry an unmeetable clause.

---

## 14. Closing-checklist verification, independently re-walked

| # | AG-02.1 closing-checklist item | This verification's finding |
| --- | --- | --- |
| 1 | One verdict per L2-owned register row, citing the register, classed, trivial implementation named where applicable | **SATISFIED.** All eight owned rows re-derived from doc 0001 § 7's Owner column carry ≥1 verdict; all fourteen entries cite `doc 0001 § 7 G-NN`; all three `AGS-S` entries name a concrete trivial behavior and a seam number from doc 0001 § 6 |
| 2 | Verdicts consistent with doc 0003's graph; a mismatch fixed before closing | **SATISFIED.** Forward pass: 14/14 clean against the Traceability spine, re-verified verbatim. Reverse pass: 20/20 against my own `grep -n 'Closes:'`, both directions. **No disagreement between the two corroborating columns in either pass.** The AG-14/AG-21 question is a spec-wording matter, not a graph mismatch (§ 6) |

**Milestone acceptance** (doc 0003:252): *"Every G-concern owned by L2 in v2 § 7 has a verdict here; any verdict that diverges from a documented default rebuts it explicitly."* First clause: satisfied (§ 3). Second clause: satisfied, with W4's qualification that F1 states its case more strongly than the sources support.

---

## 15. What could not be verified

- **Nothing was left unverified by inability.** Every claim in `decision.md` that cites a location was resolved against that location.
- **Three scenarios were verified by mechanism rather than by execution** (`S-AGS-007`, `S-AGS-012`, `S-AGS-049`), because the spec states them against a hypothetical defective artifact. This is recorded as a mechanism judgment, not scored as an ordinary pass.
- **No runtime evidence exists or is required.** AG-02.1 is a `[decision]` leaf: doc 0003's node grammar gives it no test gate, and no production code is shipped. The commands recorded in this report's envelope are the artifact's own stated reproduction procedures, executed for real.

---

## Final verdict

**PASS WITH WARNINGS.** The decision artifact closes both checklist items on evidence. Its highest-risk property — that ADR 0005 § D4 is cited only where § D4 actually says what is cited — holds under a per-row read of the source. Both audit passes are reproducible and clean. The five warnings are precision defects in derived counts, in one finding's rhetorical structure, and in two spec scenarios whose premises the source does not support; none invalidates a verdict, a node mapping, or an audit result.

**Recommended before archive**: resolve W1 (the 9/11 count), W2 (§ 4.2's table or `S-AGS-014`), and decide whether `S-AGS-035` and `S-AGS-058` are amended in the promoted spec. W3's doc 0003 spine asymmetry belongs in its own change, on the same route F1 names for doc 0001.
