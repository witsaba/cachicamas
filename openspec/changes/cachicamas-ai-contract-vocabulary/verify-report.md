# Verify report — Layer 1 contract vocabulary

> **Change**: `cachicamas-ai-contract-vocabulary`
> **Milestone**: AI-01 of [doc 0002](../../../docs/architecture/milestones/0002-cachicamas-ai-layer-1-task-graph.md#ai-01--record-the-layer-1-contract-vocabulary)
> **Node**: AI-01.1 — The vocabulary `[decision]`
> **Phase**: verify
> **Status**: **PASS**
> **Date**: 2026-07-31
> **Branch**: `feat/2026-07-31-cachicamas-ai-layer1-wave-0`
> **Base**: `origin/main` @ `b6c59e6`
> **Change commit**: `3a48014` · **Amended by**: `6da8593` (AI-02), `f701e58` (AI-03)
> **Mode**: `[decision]` leaf — no production code, no `make test` gate. Every check below is inspection, run mechanically where a count or a comparison could settle it.

---

## 1. Charter acceptance

AI-01's charter states three clauses beyond its deliverable. Each is verified against the artifact, not against intent.

| # | Charter clause | Verdict | Evidence |
| --- | --- | --- | --- |
| 1 | One definition per term, one owning milestone per term, and an explicit list of terms that are **not** Layer 1's | **PASS** — 114 rows, 0 duplicate term names, 0 duplicate identifiers, 97 non-excluded rows each naming exactly one `AI-NN` | § 4.1, § 4.2 |
| 2 | Every subsequent milestone's charter can be written using only these terms | **PASS** for the two milestones that consumed it in this wave; unfalsifiable beyond them by construction | § 6 |
| 3 | A term that turns out to be missing is appended by amendment rather than invented in a PR | **PASS — proven twice, not asserted** | § 5 |

Clause 3 is the one worth pausing on. Most vocabulary artifacts state an amendment rule that is never exercised, so the rule and its violation are indistinguishable until the first breach. This register was amended twice within the same wave, by the two milestones it blocks, and both amendments followed the stated protocol. § 5 records the measurement.

---

## 2. Deliverable inventory

| Path | Present | Note |
| --- | :---: | --- |
| `decision.md` | ✅ | the register; 328 lines, 114 rows, six categories |
| `proposal.md` | ✅ | |
| `specs/ai-contract-vocabulary/spec.md` | ✅ | `R-AIV-001` … `R-AIV-013`, `S-AIV-001` … `S-AIV-031`, three NFRs |
| `design.md` | ✅ | |
| `explore.md` | ✅ | |
| `tasks.md` | ✅ | six tasks + a nine-check verification pass, all `[x]` |
| anything under `backend/` | ✅ absent | § 8 |
| any `go.mod`, `go.sum`, `Makefile`, build or container file | ✅ absent | § 8 |

`git show --name-only 3a48014` lists **exactly six files**, every one markdown under `openspec/changes/cachicamas-ai-contract-vocabulary/`. `tasks.md` acceptance criterion 4 — "adds six markdown files and modifies nothing else" — holds byte for byte.

---

## 3. Closing-checklist closure

The verification for a `[decision]` leaf is not that the artifact *discusses* each item. It is that each item is **closed** — that a downstream author reading only this register would not have to invent the thing the item names. Each of AI-01.1's six items is checked below against the register's actual rows.

| # | Closing-checklist item | Verdict | What was checked |
| --- | --- | --- | --- |
| 1 | Request-side terms: role, message, content part and its kinds, tool declaration, tool choice, tool call, tool result, system-instruction segment, cache-boundary marker, generation option, provider escape hatch | **CLOSED** | All **eleven** named terms resolve: `V-REQ-01`, `-02`, `-04`/`-05`, `-12`, `-15`, `-16`, `-18`, `-19`, `-23`, `-26`, `-28`. "and its kinds" is delivered as a closed registered set (`V-REQ-05`) plus the four kind rows `V-REQ-08`, `-09`, `-16`, `-18` |
| 2 | Stream-side terms: event, event kind, payload, sequence, stream, terminal event, delta, block index, call ordinal | **CLOSED** | All **nine** resolve: `V-STR-10`, `-11`, `-12`, `-13`, `-01`, `-18`, `-16`, `-15`, `-21` |
| 3 | Metadata terms: finish reason, usage, token-count field, absence versus zero | **CLOSED** | All **four** resolve: `V-MET-01`, `-09`, `-10`, `-11` — plus the complete closed finish-reason vocabulary `V-MET-02` … `-08`, which item 3 does not require and `G12(c)` does |
| 4 | Failure terms defined and separated: caller-contract (AI-04) versus provider/transport (AI-19), **and** the pre-stream versus mid-stream delivery split | **CLOSED** | § 6 states the two separations as **orthogonal**, with a 2×2 diagram, and gives the boundary as a rule — *decidability without I/O* — rather than as two examples. `V-FAIL-11`/`V-FAIL-12` carry the delivery split |
| 5 | Excluded terms with their owner named: agent turn, transcript, session, tool execution, permission, compaction, cost, price, frontend | **CLOSED** | All **nine** present as `V-OUT-01` … `V-OUT-09`, each with a non-Layer-1 owner (Layer 2, Layer 3, the composition root, or above Layer 3) and none carrying a definition |
| 6 | The two wording traps restated, because both have already caused one wrong decision each | **CLOSED** | § 2, both **byte-identical** to doc 0002 line 55 — verified by string containment, not by eye |

### 3.1 The three properties that make item 4 closed rather than discussed

Item 4 is the one an artifact can appear to answer while leaving the work undone. Three things had to be present, and all three are:

1. **The lower-left cell is empty, and the reason is structural.** `V-REQ-22` requires validation to run once, before any I/O, which is what keeps caller-contract failures out of the mid-stream path. The diagram states the emptiness; the register states its cause. An artifact that only drew the grid would leave a downstream author unable to tell whether the empty cell is a gap.
2. **The boundary is a rule, not a taxonomy.** "Decidable from the request alone, without contacting a provider" is applicable to a case nobody has seen yet. § 6.3 names the three things the test is explicitly *not*: severity, blame, and where in the code the violation was noticed.
3. **Four borderline cases are worked**, including the one that inverts on a single word — an *empty* model identity is a caller-contract failure, an *unrecognised* one is a provider/transport failure delivered pre-stream. `S-AIV-012` asks for one; the artifact supplies four.

### 3.2 Item 5's negative property

An excluded term must be **named and not defined** — defining a Layer 2 concept inside a Layer 1 artifact is the first step toward Layer 1 implementing it. Checked row by row: every `V-OUT` row's third column is an owner and its fourth is either the confusable Layer 1 neighbour or a note about why the concept is not Layer 1's. No `V-OUT` row contains a definition of the excluded term itself. `S-AIV-015` holds.

`V-OUT-02` **transcript** carries the specific distinction `S-AIV-014` names: it points at `V-REQ-02` **message** and quotes AI-05's charter — "the smallest addressable unit of a transcript" — so the pair that is most often collapsed is separated at the definitional level.

---

## 4. Register integrity, measured

Everything in this section is a count or a comparison produced mechanically, not read off the artifact's own claims.

### 4.1 Row counts

| Category | § 10 claims | Actually counted | Match |
| --- | ---: | ---: | :---: |
| `V-REQ` request-side | 29 | **29** | ✅ |
| `V-STR` stream-side | 23 | **23** | ✅ |
| `V-MET` metadata | 12 | **12** | ✅ |
| `V-FAIL` failure | 15 | **15** | ✅ |
| `V-PRV` provider surface | 18 | **18** | ✅ |
| `V-OUT` excluded | 17 | **17** | ✅ |
| **Total** | **114** | **114** | ✅ |

Counted by matching rows against `^\| \`V-[A-Z]+-\d+\`` across the whole file. The artifact's own arithmetic in § 10 is correct in every cell and in the sum.

### 4.2 Uniqueness and ownership

| Property | Result |
| --- | --- |
| Duplicate `V-*` identifiers | **none** — 114 ids, 114 distinct |
| Duplicate term names across all six categories | **none** — 114 term names, 114 distinct |
| Ordinal contiguity within each category | complete: `V-REQ` 01–29, `V-STR` 01–23, `V-MET` 01–12, `V-FAIL` 01–15, `V-PRV` 01–18, `V-OUT` 01–17, with no gap and no reuse |
| Non-excluded rows whose owner cell is not exactly one `AI-NN` | **none** — all 97 hold a single current milestone identifier, no range, no pair, no blank |
| Excluded rows whose owner is a Layer 1 milestone | **none** — all 17 name a layer, a port, or the composition root |

`R-AIV-002` and `R-AIV-003` therefore hold as measured properties, not as claims.

### 4.3 Traceability against the defect and gap spine

`NFR-AIV-B` requires every `C1`–`C4` and every `G1`–`G13` identifier with a Layer 1 obligation to appear at least once in the provenance column. Extracted the provenance column of all 114 rows and searched:

```
C1 ✅  C2 ✅  C3 ✅  C4 ✅
G1 ✅  G2 ✅  G3 ✅  G4 ✅  G5 ✅  G6 ✅  G7 ✅
G8 ✅  G9 ✅  G10 ✅  G11 ✅  G12 ✅  G13 ✅
```

**17 of 17.** The register is auditable against doc 0002's traceability spine with no unreferenced identifier.

### 4.4 Trap fidelity

`S-AIV-029` requires both traps to match doc 0002 "character for character". Verified by exact string containment of each trap sentence in both files:

| Trap | In doc 0002 line 55 | In `decision.md` § 2 |
| --- | :---: | :---: |
| "Layer 1 does not know what a tool is" is too broad | ✅ | ✅ identical |
| "Provider swap is a config change" applies only after adapters exist | ✅ | ✅ identical |

Both include their qualifying clauses. This is the check most likely to be waved through, and it is the one the artifact's own § 2 explains cannot be — paraphrase is how trap 1 became too broad in the first place.

### 4.5 No Go identifiers

`R-AIV-009`, and doc 0002's authoring constraint behind it.

```
camelCase tokens    (\b[a-z]+[A-Z][a-zA-Z]*\b)          →  0
PascalCase tokens   (\b[A-Z][a-z]+[A-Z][a-zA-Z]*\b)     →  0
```

Zero across all six files. The regexes are not vacuous: run against AI-00's change directory as a control they return `importPath`, `runGoListDeps`, `testImports`, `xTestImports`, `ImportPath`, `OpenTelemetry`, `TestDomainLayer`, `TestImports` — so the scan works and this change is genuinely clean.

The all-caps tokens present are `AI`, `AIV`, `REQ`, `STR`, `MET`, `FAIL`, `PRV`, `OUT`, `CAT`, `NN` (identifier and identifier-template fragments), `MUST`/`MAY`/`NOT`/`AND`/`THEN`/`WHEN` (RFC 2119 and Gherkin), and `ADR`, `SDD`, `PR`, `API`, `RFC`, `NFR`. None is a Go identifier.

The literal tokens `func` and `struct` appear only inside the prose words *range-over-func*, *structural*, *structurally*, *construction*, *constructible*, and inside `spec.md`'s own prohibition ("struct or interface declarations"). No declaration, no signature, no field list.

---

## 5. The amendment protocol, exercised twice

`R-AIV-011` is the requirement that separates a living register from a snapshot. It was exercised by both downstream milestones in this wave, and the exercise is the evidence.

| Event | Commit | Appended | Term count |
| --- | --- | --- | ---: |
| AI-01 lands the register | `3a48014` | — | **109** |
| AI-02.1 needs two nouns it lacks | `6da8593` | `V-STR-22` carrier view, `V-STR-23` backpressure | **111** |
| AI-03.1 needs three nouns it lacks | `f701e58` | `V-PRV-16` capability, `V-PRV-17` token counting, `V-PRV-18` capability outcome | **114** |

### 5.1 What the diffs actually contain

Cumulative diff of `decision.md` from `3a48014` to `HEAD`: **11 lines added, 2 lines removed.**

The 11 additions are: two dated amendment blockquotes, two blank separator lines, and the five new rows. The 2 removals are both **replacements of the register's own arithmetic** — § 10's checklist row 2 (`V-STR-01 … V-STR-21` → `… V-STR-23`) and the term-count line (109 → 111 → 114). Nothing else in a 328-line file moved.

So `S-AIV-024`'s stronger sibling holds mechanically: **no existing row was renumbered, reworded, reordered or removed** across two amendments by two different milestones. Rule 3 is not a stated intention here; it is a measured property of the diff.

### 5.2 Both amendments are dated blockquotes, and both justify themselves

Neither amendment is a bare append. Each states *why the register lacked the term*, which is the part that makes the rule self-enforcing:

- `V-STR-23` **backpressure** — the word already appeared **inside `V-STR-08`'s definition** without being defined. An undefined word inside a definition is precisely the drift this register exists to prevent.
- `V-PRV-17` **token counting** — same shape, one category over: the phrase already appeared inside `V-OUT-06`'s definition undefined, where it silently collapses into `V-MET-09` **usage**.
- `V-PRV-18` **capability outcome** — same shape again, inside `V-PRV-09`.
- `V-PRV-16` **capability** — closes a gap **AI-01 identified in its own § 7 preamble**: that preamble names five terms AI-03's charter is not writable without, and the table delivered four.

That last one is worth recording as a finding in the register's favour. The artifact contained a self-referential inconsistency at merge — a preamble promising five terms against a table of four — and the protocol caught it at the first milestone that needed the fifth, which is exactly the failure mode `R-AIV-011` exists to convert from a silent gap into a dated row.

---

## 6. Spec conformance

All thirteen requirements in `specs/ai-contract-vocabulary/spec.md` verified. The clauses where the verdict needed judgement rather than a command are listed; the remainder are settled by the counts in § 4.

| Requirement | Verdict | Note |
| --- | --- | --- |
| `R-AIV-001` — the artifact is singular and no other file carries a competing definition | **PASS**, with the judgement in § 7.1 | AI-02 and AI-03 each **quote** a register row; verified those quotations are verbatim rather than independent definitions |
| `R-AIV-003` / `S-AIV-005` — the owning milestone's charter must actually cover the term | **PASS** | Spot-checked the assignments most likely to be wrong: `V-STR-21` call ordinal → AI-09 (tool-call content contract, where the concept originates) rather than AI-14; `V-FAIL-14` redaction → AI-36; `V-FAIL-15` retry policy → AI-35. Each matches the named charter's stated scope |
| `R-AIV-005` / `S-AIV-009` — terms beyond the checklist minimum must name the downstream charter that could not be written without them | **PASS**, qualified | The explicit acceptance-criterion argument is made **once**, in § 7's preamble for the whole `V-PRV` category. For the other 73 extra terms the justification is carried by the Owner column, which names the milestone whose charter defines the term. Satisfied in substance; the argument is per-category rather than per-row |
| `R-AIV-008` / `S-AIV-018` — every ownership or provenance citation is a **current** identifier | **PASS** | The register's own § 1 names the three retired numbers (AI-43→AI-11, AI-45→AI-07, AI-47→AI-02) each beside its replacement, which is the single exception `S-AIV-018` permits. The live trap is handled explicitly: doc 0001 § 3.1 assigns **C4** to "AI-18", and AI-18 exists today as a different real milestone — the register assigns C4 to **AI-19** in all four rows that cite it |
| `R-AIV-009` — no Go identifiers | **PASS** | § 4.5, measured |
| `R-AIV-010` — no file under `backend/`, no module or build file | **PASS** | § 8 |
| `R-AIV-011` — growth by amendment | **PASS — exercised twice** | § 5 |
| `R-AIV-012` / `S-AIV-026`, `S-AIV-027` — AI-02's and AI-03's charters are expressible in these terms | **PASS**, and stronger than the scenario asks | Both charters were not merely *mappable* — both milestones were **written** from this register in this wave. AI-02 cites 35 distinct `V-*` identifiers, AI-03 cites 60. **Every one of the 95 resolves to a live row**; zero dangling citations |
| `R-AIV-013` — traps verbatim | **PASS** | § 4.4, byte-compared |
| `NFR-AIV-B` — traceability | **PASS** | § 4.3, 17 of 17 |
| `NFR-AIV-C` — no citation of shipped code | **PASS** | Every citation points at doc 0001, doc 0002, doc 0003 or an ADR. No Layer 1 code exists to cite |

---

## 7. Findings

Three observations. None blocks the verdict; all three are recorded because a verify-report that finds nothing has usually not looked.

### 7.1 The closest call — a downstream artifact quoting a register row

`R-AIV-001` says no other file may carry a competing definition of a Layer 1 term. AI-03's decision § 3 opens with a block quotation:

> A capability is a behavior a consumer can ask a provider for, and whose presence or absence the consumer can observe.

Compared against `V-PRV-16`'s definition text: **identical wording**. It is a quotation of the owning row, not a second definition, and AI-03 states as much ("`V-PRV-16` names the unit. Its content, for this decision's purposes"). AI-02 § 5 does the same for `V-STR-06`. Both hold, and both are the shape a genuine violation would take, so they are recorded rather than passed over.

### 7.2 Two appended rows sit before lower ordinals in document order

`V-STR-22` and `V-STR-23` were appended to § 4.1 (container terms), which places them physically **before** `V-STR-10` … `V-STR-21` in § 4.2 (content terms). § 10's checklist row now reads "`V-STR-01` … `V-STR-23`", which suggests a contiguous span the reader will not find contiguously.

Rule 3 forbids renumbering, reusing and reordering *identifiers*; no identifier moved, and the placement is semantically correct — a carrier view and backpressure are container properties, and § 4's own preamble states that definition order in this category is load-bearing. **Not a violation.** Cosmetic, and worth a sentence in § 4's preamble at archive if anyone cares.

> **Resolved 2026-07-31, before merge.** A paragraph was added to § 4's amendment blockquote stating that identifiers are append-only while rows are grouped by meaning, so the two orders diverge the first time a category is extended; a range such as `V-STR-01 … V-STR-23` denotes the identifier span, never a reading order. No identifier or row moved.

### 7.3 A path token that the constraint does not reach

The header of `decision.md` carries `**Target package**: `backend/agent/src/ai/` (Layer 1)`, and § 10's node-status paragraph says "there is nothing in `backend/agent/` that this change touches". These are repository directory paths in metadata, not Go package paths (`github.com/cachicamas/backend/agent/src/ai`) and not spellings of any Layer 1 surface. `S-AIV-019`'s scan targets camel-case names, declarations and field lists, none of which is present. **Not a violation**; recorded because the sibling specs for AI-02 and AI-03 word the same prohibition as "a package path", where the literal reading is tighter.

---

## 8. Out-of-scope confirmations

Verified **not** done, each deliberate:

- **Nothing under `backend/`.** `git diff --name-only origin/main..HEAD -- backend/` returns ten paths, all of them `backend/agent/**` plus `backend/database_administrator/src/domain/imports_test.go` — every one belonging to **AI-00**, none to this change. `git show --name-only 3a48014` touches `backend/` zero times.
- **No `go.mod`, `go.sum`, `Makefile`, `go.work`, container or CI file.** None appears in this change's commit.
- **No spelling chosen for any term.** The vocabulary is conceptual throughout; every term is a noun phrase with spaces.
- **No excluded term defined.** All 17 `V-OUT` rows name and attribute; none defines.
- **doc 0002 not amended.** This change adds no node, renumbers none, and corrects no stated claim in the graph — so there is no living-graph amendment to record, and its absence is a result rather than an omission.
- **Behavior not decided.** § 9 rule 5 states the abstention explicitly: this artifact does not choose the stream carrier (AI-02), the capability matrix or discovery mechanism (AI-03), validation granularity (AI-04), or the content-part strategy (AI-06). Checked against the two milestones that followed: AI-02 chose a carrier the register had only *named* (`V-STR-02`), and AI-03 chose standings the register had only named (`V-PRV-06`, `V-PRV-07`). Neither found its decision pre-made.

---

## 9. Verdict

**PASS.** All six closing-checklist items are closed against the register's actual rows rather than against its own summary. The measured register is 114 rows with zero duplicate identifiers, zero duplicate term names, complete ordinal contiguity, one owning milestone on every one of the 97 non-excluded rows, all 17 defect and gap identifiers traceable, both wording traps byte-identical to their source, and zero Go identifiers.

The strongest evidence is not any single check: it is that the register was **consumed twice in the same wave** by the two milestones it blocks, that all 95 of their `V-*` citations resolve, and that the two nouns each milestone found missing were appended under the stated protocol with 11 added lines, 2 replaced lines, and not one existing row disturbed.

**Ready for archive** once the wave's PR merges.
