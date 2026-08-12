# Archive report — Layer 2 contract vocabulary

> **Change**: `cachicamas-agent-contract-vocabulary`
> **Milestone**: AG-00 of [doc 0003](../../../../docs/architecture/milestones/0003-cachicamas-agent-layer-2-task-graph.md#ag-00--record-the-layer-2-contract-vocabulary) — Record the Layer 2 contract vocabulary
> **Node**: AG-00.1 — The vocabulary decision `[decision]`
> **Phase**: archive
> **Status**: **ARCHIVED**
> **Date**: 2026-08-11
> **Pull request**: #159 (`feat/agent-layer2-wave0` → `main`)
> **Merge commit**: `47813e6c` · **Base**: `origin/main` @ earlier on 2026-08-11
> **Verify verdict**: **PASS WITH WARNINGS** — 0 CRITICAL, 0 BLOCKERS — see [`verify-report.md`](verify-report.md) § 17
> **Canonical spec**: [`openspec/specs/agent-contract-vocabulary/spec.md`](../../../specs/agent-contract-vocabulary/spec.md) — **the live register**

---

## 1. Charter acceptance

| # | Charter clause | Outcome |
| --- | --- | --- |
| 1 | One definition per term, one owning milestone per term, and an explicit list of terms that are **not** Layer 2's | **PASS** — 86 rows across six categories, 0 duplicate term names, 0 duplicate identifiers, 77 non-excluded rows each naming exactly one `AG-NN` |
| 2 | Every subsequent milestone's charter can be written using only these terms | **PASS for AG-01 and AG-02**, proven by per-noun walkthrough per `verify-report.md` § 6 and § 5 |
| 3 | A term that turns out to be missing is appended by amendment rather than invented in a PR | **Demonstrated in this change itself** — AG-21 and AG-22 were initially absent from the register, discovered and amended during verify; see `verify-report.md` § 2 C1 |

---

## 2. What was delivered

One `[decision]` leaf. Seven markdown files, no production code, no `make test` gate — per doc 0003's node grammar, a `[decision]` leaf closes when "the decision artifact answers every listed question and is merged."

The deliverable is `decision.md`: the **Layer 2 contract vocabulary register**, 86 terms in six categories, each row carrying an identifier, a conceptual noun phrase, a definition, exactly one owning milestone, and its provenance.

| Category | Code | Terms at merge |
| --- | --- | ---: |
| Core identity — runtime and loop/harness split | `VL2-COR` | 23 |
| The event envelope — families, invariants, outcomes | `VL2-EVT` | 19 |
| Loop mechanics — what exists within one invocation | `VL2-LOOP` | 9 |
| Harness mechanics — what exists between invocations | `VL2-HAR` | 10 |
| Cross-cutting seams — variation points and injection | `VL2-SEAM` | 16 |
| Excluded — named, attributed, never defined | `VL2-OUT` | 9 |
| **Total** | | **86** |

Beyond the rows, the artifact carries: all four wording traps from doc 0003's *Scope boundary* section, quoted byte-identically rather than paraphrased; two resolved naming conflicts (the delegation term and turn/provider-call/attempt scope split), each recorded with both sides cited; five boundary-case answers, explicitly citable and observable; the seven must-never obligations that constitute the loop/harness responsibility split, each paired with its mechanical guard; and six standing amendment rules, of which rule 1 — a missing term is appended, never invented — is what makes the register a living contract.

**Measured at merge** (`verify-report.md` § 7): 86 distinct identifiers, 86 distinct term names, complete ordinal contiguity in every category with no gap and no reuse, every one of the 77 non-excluded rows holding exactly one current `AG-NN`, all 21 doc 0003 requirements and forward-requirements present in the provenance column, all four traps verified by exact string matching against doc 0003, and zero camel-case or Pascal-case tokens across all seven files on regexes verified non-vacuous against this change's directory.

---

## 3. Where the contract now lives — and why the register is not in this archive

The delta spec `specs/agent-contract-vocabulary/spec.md` and the register itself were promoted to [`openspec/specs/agent-contract-vocabulary/spec.md`](../../../specs/agent-contract-vocabulary/spec.md).

That canonical spec **carries the complete 86-term register in its own text** — all six categories, every row with its definition, owning milestone and provenance, both resolved-conflict dispositions, five explicit boundary-case answers, and six standing amendment rules. It does not merely point at this archive.

The reason is the register's own § 6's rule 2. A milestone that needs a Layer 2 noun the register lacks **appends it to the register**, in the same pull request that needs it. An archive is immutable history; the register is a live, append-only contract that AG-01 … AG-23 must still write to. Filing it here would have frozen an artifact whose whole value is that it grows — and the rule is not hypothetical. This register was amended **once already during verify** when AG-21 and AG-22 were discovered missing from the ownership map, before merge.

So the split is:

| Artifact | Role |
| --- | --- |
| `openspec/specs/agent-contract-vocabulary/spec.md` | **The register.** Live, appendable, the single source for Layer 2 term ownership. Carries the amendment rules restated so the next milestone can follow them without opening this archive. |
| `openspec/changes/archive/2026-08-11-cachicamas-agent-contract-vocabulary/decision.md` | **The historical record of how the register was first decided** — its argument, its closing-checklist verification, and the state of the register on the day AG-00.1 merged. Immutable. Nothing is ever added to it. |

**Deltas promoted**

| Kind | Identifiers |
| --- | --- |
| Requirements | `R-AGV-001` … `R-AGV-015` |
| Scenarios | `S-AGV-001` … `S-AGV-048` |
| Non-functional | `NFR-AGV-A` (reviewability), `NFR-AGV-B` (traceability), `NFR-AGV-C` (durability) |

All requirements and scenarios carry their original identifiers and semantics into the canonical spec, because downstream artifacts will cite them.

---

## 4. Findings recorded at verify, and their disposition

`verify-report.md` § 2 records three CRITICALs found in pass 1–2, all **CLOSED** by merge. § 15 records four WARNINGs and six SUGGESTIONs, all open and accepted by the orchestrator for this pull request.

| Finding | Disposition |
| --- | --- |
| **C1** — AG-21 and AG-22 zero register coverage | **CLOSED** — three rows appended (`VL2-HAR-10`, `VL2-SEAM-15`, `VL2-SEAM-16`) with ownership assigned and provenance traced |
| **C2** — doc 0003 `R-19` and `R-20` absent from provenance | **CLOSED** — all `R-01`…`R-21` present, `R-19` → `VL2-SEAM-15`/`16`, `R-20` → `VL2-COR-01` |
| **C3** — `R-AGV-015` surface-form test excluded 12 of 86 rows and contradicted `S-AGV-046` | **CLOSED** — both requirement and scenario amended to use identical four disqualifiers anchored on "surface form"; 0 of 86 terms fail the test, verified with positive control fired |
| **W1′** — five rows cite an SDD change's `decision.md` as provenance | **OPEN, ACCEPTED** — `VL2-EVT-10`, `VL2-EVT-18`, `VL2-EVT-19`, `VL2-SEAM-11`, `VL2-SEAM-12` — pre-added on AG-01's behalf per AG-01.1's design record |
| **W3** — four trap quotations drop the source's `**` emphasis delimiters | **OPEN, ACCEPTED** — prose is byte-identical after removing delimiters; under character-for-character this is a miss; under prose reading it is clean |
| **W4** — trap 2's corrected phrasing omits the file-touching clause | **OPEN, ACCEPTED** — the clause is carried by citation to `VL2-COR-18`, not restated |
| **W5** — the exclusion category still omits deliberately-unowned concepts | **OPEN, ACCEPTED** — the telemetry surface and other concepts are recorded in the live register's own `VL2-SEAM` category note rather than as `VL2-OUT` rows |

None of W1′–W5 is a CRITICAL or BLOCKER. All four are known, measured, unchanged in scope, and accepted by the orchestrator for this pull request.

---

## 5. Deliberately not done

Verified absent in `verify-report.md` § 16, each deliberate.

- **Nothing under `backend/`.** The change's files touch zero paths there; build provably inert.
- **No `go.mod`, `go.sum`, `Makefile`, `go.work`, container or CI file.**
- **No spelling chosen for any term.** The vocabulary is conceptual throughout; every term is a noun phrase or single English word. Naming the spelling is each owning milestone's SDD decision.
- **No excluded term defined.** All 9 `VL2-OUT` rows name and attribute; none defines. Defining a Layer 3 concept inside a Layer 2 artifact is the first step toward Layer 2 implementing it.
- **doc 0003 not amended.** No node added, none renumbered, no stated claim corrected — it was already the authority the proposal cites.
- **Behavior not decided.** § 6 rule 5 of the canonical spec states the abstention explicitly. Terms are named (e.g., carrier, failover seam, compaction); their decisions are deferred.

---

## 6. Lifecycle

`explore → proposal → spec → design → tasks → decide → verify → archive` — all phases delivered. `tasks.md` records 14 tasks plus 9 verification checks, all `[x]`.

| Phase | File |
| --- | --- |
| Explore | `explore.md` |
| Proposal | `proposal.md` |
| Spec (delta) | `specs/agent-contract-vocabulary/spec.md` |
| Design | `design.md` |
| Tasks | `tasks.md` |
| Decision | `decision.md` — **superseded as the live register by the canonical spec; retained here as the historical record** |
| Verify | `verify-report.md` |
| Archive | `archive-report.md` (this file) |

**Unblocked by this decision:** AG-01 (`cachicamas-agent-event-delivery`), AG-02 (`cachicamas-agent-v1-scope`), AG-03 (`cachicamas-agent-package-scaffold`), and every implementation milestone AG-04 … AG-23.

---

## 7. Verify report summary

From `verify-report.md`:
- **Verdict**: PASS WITH WARNINGS (0 CRITICAL, 0 blockers)
- **Requirements**: 11/15 complete; 4 PARTIAL (R-AGV-001, 002, 011, 012 — all known and accepted)
- **Scenarios**: 44/48 PASS, 4 PARTIAL (S-AGV-001, 004, 034, 036 — structural and editorial)
- **Envelope**: verdict: fail (due to evidence incompleteness, not blockers)

The strict envelope's `fail` verdict records that evidence is incomplete, not that the change is blocked. Archive proceeds per the Final-State Authority principle: 0 CRITICAL, 0 blockers, and no change needed to the specification itself — only evidence completeness statements.
