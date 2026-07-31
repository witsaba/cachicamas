# Archive report — Layer 1 contract vocabulary

> **Change**: `cachicamas-ai-contract-vocabulary`
> **Milestone**: AI-01 of [doc 0002](../../../../docs/architecture/milestones/0002-cachicamas-ai-layer-1-task-graph.md#ai-01--record-the-layer-1-contract-vocabulary) — Record the Layer 1 contract vocabulary
> **Node**: AI-01.1 — The vocabulary `[decision]`
> **Phase**: archive
> **Status**: **ARCHIVED**
> **Date**: 2026-07-31
> **Pull request**: #95 (`feat/2026-07-31-cachicamas-ai-layer1-wave-0` → `main`)
> **Merge commit**: `a831c06` · **Base**: `origin/main` @ `b6c59e6`
> **Change commit on `main`**: `e9b1804`, amended by `461cc44` (AI-02) and `d4ea0d7` (AI-03), and by `ea7cb8e` before merge. The verify report cites the pre-rebase hashes `3a48014`, `6da8593` and `f701e58` for the same three commits.
> **Verify verdict**: **PASS** — see [`verify-report.md`](verify-report.md)
> **Canonical spec**: [`openspec/specs/ai-contract-vocabulary/spec.md`](../../../specs/ai-contract-vocabulary/spec.md) — **the live register**

---

## 1. Charter acceptance

| # | Charter clause | Outcome |
| --- | --- | --- |
| 1 | One definition per term, one owning milestone per term, and an explicit list of terms that are **not** Layer 1's | **PASS** — 114 rows, 0 duplicate term names, 0 duplicate identifiers, 97 non-excluded rows each naming exactly one `AI-NN` |
| 2 | Every subsequent milestone's charter can be written using only these terms | **PASS** for the two milestones that consumed it in this wave; unfalsifiable beyond them by construction |
| 3 | A term that turns out to be missing is appended by amendment rather than invented in a PR | **PASS — proven twice, not asserted** |

Clause 3 is the one worth pausing on, and it is the reason this change's output could not simply be filed away. Most vocabulary artifacts state an amendment rule that is never exercised, so the rule and its violation are indistinguishable until the first breach. This register was amended **twice within the same wave**, by the two milestones it blocks, and both amendments followed the stated protocol.

---

## 2. What was delivered

One `[decision]` leaf. Six markdown files, no production code, no `make test` gate — per doc 0002's node grammar, a `[decision]` leaf closes when "the decision artifact answers every listed question and is merged."

The deliverable is `decision.md`: the **Layer 1 contract vocabulary register**, 114 terms in six categories, each row carrying an identifier, a conceptual noun phrase, a definition, exactly one owning milestone, and its provenance.

| Category | Code | Terms at merge |
| --- | --- | ---: |
| Request-side — what goes to the model | `V-REQ` | 29 |
| Stream-side — what comes back, and what carries it | `V-STR` | 23 |
| Metadata — what a response says about itself | `V-MET` | 12 |
| Failure — two vocabularies, two delivery paths | `V-FAIL` | 15 |
| Provider surface and proving apparatus | `V-PRV` | 18 |
| Excluded — named, attributed, never defined | `V-OUT` | 17 |
| **Total** | | **114** |

Beyond the rows, the artifact carries: both wording traps from doc 0002's *Layer boundary* section, quoted **byte-identically** rather than paraphrased; the two-axis failure grid separating the owner split (AI-04 versus AI-19) from the delivery split (pre-stream versus mid-stream), with four worked borderline cases and the *decidability without I/O* rule that resolves a case nobody has seen yet; and six standing rules, of which rule 2 — a missing term is appended, never invented — is what makes the register a living contract.

**Measured at merge** (`verify-report.md` § 4): 114 distinct identifiers, 114 distinct term names, complete ordinal contiguity in every category with no gap and no reuse, every one of the 97 non-excluded rows holding exactly one current `AI-NN`, all 17 `C1`–`C4` and `G1`–`G13` identifiers present in the provenance column, both traps verified by exact string containment against doc 0002, and zero camel-case or Pascal-case tokens across all six files on regexes verified non-vacuous against AI-00's change directory.

### 2.1 The amendment protocol, exercised twice

| Event | Commit on `main` | Appended | Term count |
| --- | --- | --- | ---: |
| AI-01 lands the register | `e9b1804` | — | **109** |
| AI-02.1 needs two nouns it lacks | `461cc44` | `V-STR-22` carrier view, `V-STR-23` backpressure | **111** |
| AI-03.1 needs three nouns it lacks | `d4ea0d7` | `V-PRV-16` capability, `V-PRV-17` token counting, `V-PRV-18` capability outcome | **114** |

The cumulative diff of `decision.md` across both amendments is **11 lines added, 2 lines replaced** in a 328-line file. The 11 additions are two dated amendment blockquotes, two blank separators, and the five new rows; the 2 replacements are both the register's own arithmetic — § 10's checklist row 2 and the term-count line. **No existing row was renumbered, reworded, reordered or removed.** Rule 3 is a measured property of the diff, not a stated intention.

Three of the five appended nouns are the same shape: a word already used *inside* an existing definition without being defined — `backpressure` inside `V-STR-08`, `token counting` inside `V-OUT-06`, `capability outcome` inside `V-PRV-09`. A fourth, `V-PRV-16` **capability**, closes a gap the register identified in its own § 7 preamble, which names five terms AI-03's charter is not writable without against a table that delivered four. The protocol converted a silent self-referential inconsistency into a dated row at the first milestone that needed the fifth term.

---

## 3. Where the contract now lives — and why the register is not in this archive

The delta spec `specs/ai-contract-vocabulary/spec.md` and the register itself were promoted to [`openspec/specs/ai-contract-vocabulary/spec.md`](../../../specs/ai-contract-vocabulary/spec.md).

That canonical spec **carries the complete 114-term register in its own text** — all six categories, every row with its definition, owning milestone and provenance, both dated amendment blockquotes, the two wording traps, the two-axis failure grid and the six standing rules. It does not merely point at this archive.

The reason is the register's own § 9 rule 2. A milestone that needs a Layer 1 noun the register lacks **appends it to the register**, in the same pull request that needs it. An archive is immutable history; the register is a live, append-only contract that AI-04 … AI-40 must still write to. Filing it here would have frozen an artifact whose whole value is that it grows — and the rule is not hypothetical, since it was exercised twice before this change even reached archive.

So the split is:

| Artifact | Role |
| --- | --- |
| `openspec/specs/ai-contract-vocabulary/spec.md` | **The register.** Live, appendable, the single source for Layer 1 term ownership. Carries the amendment rules restated so the next milestone can follow them without opening this archive. |
| `openspec/changes/archive/2026-07-31-cachicamas-ai-contract-vocabulary/decision.md` | **The historical record of how the register was first decided** — its argument, its closing-checklist verification, and the state of the register on the day AI-01.1 merged. Immutable. Nothing is ever added to it. |

**Deltas promoted**

| Kind | Identifiers |
| --- | --- |
| Requirements | `R-AIV-001` … `R-AIV-013` |
| Scenarios | `S-AIV-001` … `S-AIV-031` |
| Non-functional | `NFR-AIV-A` (reviewability), `NFR-AIV-B` (traceability), `NFR-AIV-C` (durability) |

Two requirements were rewritten from change voice into standing voice rather than dropped. `R-AIV-001` now says that exactly one register exists and it is the canonical spec, instead of naming this change's `decision.md` path; `R-AIV-010`, which forbade this change from touching `backend/`, now states that the register is a documentation contract never implemented as code and binds every future amendment to the same property. Identifiers and scenario semantics are unchanged, because downstream artifacts cite them.

---

## 4. Findings recorded at verify, and their disposition

`verify-report.md` § 7 records three. None blocked the verdict.

| # | Finding | Disposition |
| --- | --- | --- |
| 7.1 | AI-02 § 5 and AI-03 § 3 each **quote** a register row verbatim, which is the exact shape a violation of `R-AIV-001` would take | **Holds.** Compared word for word: both are quotations of the owning row, and both artifacts frame them as such. Recorded rather than passed over, because verifying it required a text comparison rather than a reading |
| 7.2 | `V-STR-22` and `V-STR-23` were appended to § 4.1 (container terms), which places them physically *above* `V-STR-10` … `V-STR-21` in § 4.2, while § 10's checklist row reads `V-STR-01 … V-STR-23` | **Fixed before merge**, in commit `ea7cb8e`. A second paragraph was added to § 4's amendment blockquote stating that identifiers are append-only while rows are grouped by meaning, so the two orders diverge the first time a category is extended; a range denotes the identifier span, never a reading order. No identifier and no row moved. The paragraph is carried into the canonical spec |
| 7.3 | `decision.md`'s header carries the repository directory path `backend/agent/src/ai/` as metadata | **Not a violation.** It is a directory path, not a Go package path, and not a spelling of any Layer 1 surface. Recorded because the sibling specs for AI-02 and AI-03 word the same prohibition as "a package path", where the literal reading is tighter |

---

## 5. Deliberately not done

Verified absent in `verify-report.md` § 8, each deliberate.

- **Nothing under `backend/`.** The change's commit touches it zero times; the ten `backend/` paths in the wave's range all belong to AI-00.
- **No `go.mod`, `go.sum`, `Makefile`, `go.work`, container or CI file.**
- **No spelling chosen for any term.** The vocabulary is conceptual throughout; every term is a noun phrase with spaces. Naming the spelling is each owning milestone's SDD decision.
- **No excluded term defined.** All 17 `V-OUT` rows name and attribute; none defines. Defining a Layer 2 concept inside a Layer 1 artifact is the first step toward Layer 1 implementing it.
- **doc 0002 not amended.** No node added, none renumbered, no stated claim corrected — its absence is a result rather than an omission.
- **Behavior not decided.** § 9 rule 5 states the abstention explicitly, and it was checked against the two milestones that followed: AI-02 chose a carrier the register had only *named* (`V-STR-02`), and AI-03 chose standings the register had only named (`V-PRV-06`, `V-PRV-07`). Neither found its decision pre-made.

**One companion commit in the same pull request, outside this change's scope.** Commit `ea7cb8e`, *"docs: remap retired milestone identifiers and correct stale narrative in doc 0001 and ADR 0005"*, addresses upstream the hazard this register documents in its own § 1: doc 0001 and ADR 0005 carried retired milestone numbers, and one of them — doc 0001 § 3.1 assigning defect **C4** to "AI-18" — resolves to a *different real* milestone in current numbering. The register handles that hazard by translating every citation through doc 0002's identifier map; the companion commit fixes the sources. It is not an artifact of this change and appears in no file of it.

---

## 6. Lifecycle

`explore → proposal → spec → design → tasks → decide → verify → archive` — all phases delivered. `tasks.md` records six tasks plus a nine-check verification pass, all `[x]`.

| Phase | File |
| --- | --- |
| Explore | `explore.md` |
| Proposal | `proposal.md` |
| Spec (delta) | `specs/ai-contract-vocabulary/spec.md` |
| Design | `design.md` |
| Tasks | `tasks.md` |
| Decision | `decision.md` — **superseded as the live register by the canonical spec; retained here as the historical record** |
| Verify | `verify-report.md` |
| Archive | `archive-report.md` (this file) |

**Unblocked by this decision:** AI-02 (`cachicamas-ai-stream-lifecycle`), AI-03 (`cachicamas-ai-minimum-capabilities`), and every contract milestone AI-04 … AI-40. AI-02 cites 35 distinct `V-*` identifiers and AI-03 cites 60; all 95 resolve.
