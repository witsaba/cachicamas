# ADR 0008: Adopt the cachicamas delivery loop

> Status: **Accepted** (2026-08-11, issue #151, PR #152)
> Author: cachicamas parent orchestrator, from an adversarial review of the proposed pipeline
> Companion: [PRD 0001 — The cachicamas delivery loop](../prd/0001-cachicamas-delivery-loop.md)
> Amends: [ADR 0007 § D3](0007-adopt-dag-convention-for-task-graphs.md) — see § D4
> Depends on: [ADR 0005 § D1](0005-promote-agent-stack-to-own-module.md) (module boundary), [ADR 0007](0007-adopt-dag-convention-for-task-graphs.md) (DAG contract)

---

## Resolved TOC

| § | Settles |
|---|---|
| [Context](#context) | Why a decision is needed now |
| [Decision drivers](#decision-drivers) | What any option had to satisfy |
| [Options considered](#options-considered) | The three mixtures, priced |
| [D1](#d1--adopt-option-c-markdown-authors-the-database-projects-agents-execute) | The route |
| [D2](#d2--the-node-type-selects-the-lane-and-feature-is-not-a-level) | Lanes, and the level that is not added |
| [D3](#d3--judges-are-differentiated-by-input-not-by-effort) | The review protocol |
| [D4](#d4--a-projection-is-a-rendering--amending-adr-0007--d3) | The ADR 0007 amendment |
| [D5](#d5--delete-the-duplicated-planning-artifacts) | What the lightness actually comes from |
| [Consequences](#consequences) | What we pay |

---

## Context

cachicamas plans well and executes twice. The plan lives in three milestone documents governed by ADR 0007 with a working validator; Layer 1 closed 42 of 42 milestones through it. Execution runs through an eight-phase SDD that produces five durable files per change on top of a milestone-document section that already carries a Charter and 1–7 Gherkin scenarios — the Charter *is* a proposal and the Gherkin *is* a spec, so both are written twice.

Three facts force a decision now rather than later.

**Layer 2 has not started.** Doc 0003 holds 24 unbuilt milestones and doc 0004 holds 25. Whatever loop runs them will run 49 times. Choosing after AG-00 means changing the loop mid-layer.

**The database cannot hold the loop.** `backend/database_administrator` carries a `project → requirement → milestone → task → spec → spec_phase` chain that models this pipeline almost exactly — `spec_phase` already enumerates eight of the phases by name. It has zero Go code, and `milestone` is declared `requirement_id BIGINT PRIMARY KEY REFERENCES requirement(id)`: the primary key *is* the foreign key, so one requirement admits exactly one milestone. Doc 0002 has 42. The schema is not merely unimplemented; it is unusable as written.

**The repository contradicts itself in four verified places** (PRD 0001 § 3): five migrations cite a forward-only rule `openspec/AGENTS.md` does not contain while the governing spec mandates the opposite; that governing spec was never promoted out of its archive folder; the two "blind judges" of `judgment-day` differ only in name and reasoning effort; and there is no CI, so every gate is a promise an agent makes about itself.

An adversarial review of the initially proposed pipeline (`PRD → Milestones → Features → Specs → TDD Red → Implementation → TDD Green → Refactor → PR → two judges on the changed lines → human merges`) returned six blockers and ten majors. The three most consequential: the "Features" level duplicates a level the grammar already has; reviewing only the changed lines is blind to omission by construction; and nothing in the proposal advances state — the documented failure of every lightweight alternative on the market.

## Decision drivers

1. **The plan is not the problem.** ADR 0007 and the milestone documents stay untouched. Any option that disturbs them pays for a rebuild of 91 milestones.
2. **Lighter must be measurable**, not asserted. If the artifact count does not fall, the claim is false.
3. **Something must close the loop.** State advances on evidence, by a named actor, or the loop is a to-do list.
4. **Omission must be catchable.** A requirement never implemented produces no changed line.
5. **The module boundary is load-bearing.** `src/domain/imports_test.go` and ADR 0005 § D1 mean the agent module cannot import the database service. Agents reach state over HTTP or not at all.
6. **The human keeps merge authority** and gains the decisions that are actually theirs.
7. **No second source of truth for the plan.** ADR 0007 D3 rejected a structured sidecar; any option storing plan structure must answer that rejection rather than ignore it.

## Options considered

### Option A — Skills only, tracker-native

Adopt roughly six small skills in Matt Pocock's shape. The plan stays in markdown. Execution state lives in GitHub issues and labels. The database is not touched.

**For:** cheapest by a wide margin — no migration, no Go, no parser, no drift surface. Ships in days. Fully reversible: deleting the skills restores the status quo.

**Against:** it is a personal workflow, not a product capability. cachicamas-the-product learns nothing, and the dead planning chain in `database_administrator` stays dead permanently. The frontier across 91 milestones cannot be computed — GitHub's blocking edges would have to be mirrored by hand from `Depends on:` fields, which is exactly the drift ADR 0007 warns about, relocated to a tracker. It inherits every documented defect of the source material unmitigated: nothing closes the loop, ~50% autonomous trigger rate, prose delegation failing silently. Concurrency has no lease, so two sessions can claim the same node.

### Option B — The database is the tracker

Make the database authoritative for waves, milestones, nodes, edges and scenarios. Markdown documents become an export rendered *from* the database.

**For:** one queryable system of record. Frontier, claims, evidence and verdicts are all first-class. cachicamas gains a genuine product capability — it can drive and display the loop for any project, which is the direction `requirement`/`milestone`/`spec_phase` was originally reaching for.

**Against:** it directly reverses ADR 0007 D2, which fixes `Depends on:` fields as the single source of truth and states *"the markdown **is** the data."* It requires an authoring interface before anyone can plan anything, so Layer 2 stalls behind a UI. It loses `git blame` on the plan, loses plan changes appearing in a PR diff next to the prose they change, and strands three shipped documents whose amendment blockquotes are their own history. Largest build by a wide margin, and the least reversible.

### Option C — Markdown authors, the database projects, agents execute

The milestone document remains the only authored source of plan structure; ADR 0007 is untouched. `validate_taskgraph.py` emits a canonical projection that a service ingests into Postgres as derived rows stamped with `(path, content_sha256, parser_version)`. Execution state — phase events, claims, leases, verdicts, PR candidates, evidence — lives only in the database and has no markdown representation. Every column is either a projection (P) or execution state (S), never both. There is no write path from database to document.

**For:** no second source of truth for the plan, and the property is *checkable* rather than aspirational — (P) rows are deleted and reinserted wholesale, and ingest refuses to run on a hash mismatch. The three shipped documents work unchanged. Frontier, lease, fencing epoch, typed evidence and immutable review candidates all become real. cachicamas gains the product capability of Option B without its authoring problem.

**Against:** it needs a parser, and a second implementation of the ADR-0007 grammar in Go would drift from the Python one while CI stays green — mitigated by making the validator emit the projection rather than re-parsing. It inherits state-orphaning: editing a document reshuffles (P) rows, and a scenario whose text changes after it went green has no clean reconciliation. That cost is real, unavoidable, and the honest answer is detection, not prevention. More build than A.

### Priced comparison

| | A — skills only | B — database authors | **C — projection** |
|---|---|---|---|
| ADR 0007 | untouched | **reversed** | untouched + one amendment |
| Plan authoring | markdown | UI, must be built first | markdown |
| Frontier across 91 milestones | hand-mirrored | query | query |
| Claim / lease / fencing | none | yes | yes |
| Evidence bound to an immutable commit | no | yes | yes |
| Product capability for cachicamas | none | full | full |
| New drift surface | tracker labels | — | (P) rows, hash-gated |
| Build size | days | months | weeks |
| Reversibility | total | poor | good — drop (P) rows, keep the documents |

## Decision

### D1 — Adopt Option C: markdown authors, the database projects, agents execute

The milestone document is the only authored source of plan structure. The database holds a derived projection of it plus execution state that exists nowhere else. Requirements are stated in PRD 0001 § 6; the schema shape is § 7.

Option A is rejected because it permanently strands the planning chain and cannot compute a frontier without recreating the drift ADR 0007 exists to prevent. Option B is rejected because it reverses a decision that is working, and because stalling Layer 2 behind a plan-authoring UI trades a solved problem for an unsolved one.

### D2 — The node type selects the lane, and "feature" is not a level

The containment tree is document → wave → milestone → node, with nodes nesting up to three levels (ADR 0007 D2.3 as amended 2026-08-10). A `[compound]` node is what "feature" was reaching for, and doc 0002 already writes the mapping: *"the leaves of that milestone's graph become its `tasks.md` phases."* Node ≡ task. Adding "feature" would make an eighth level on a seven-level tree and would require defining its cardinality against `[compound]`, which cannot be done because a `[compound]` node *is* a feature with children.

The `[type]` tag therefore selects the execution lane rather than decorating the diagram: `[leaf]`/`[guard]` run the TDD lane; `[decision]` runs a research → recorded artifact → human sign-off lane producing no code and given no Gherkin; `[mechanical]` runs an evidence lane exempt from red/green; `[compound]` is never claimed.

This is not a refinement. Layer 2 opens with AG-01 and AG-02, both `[decision]` nodes whose deliverable is *"a recorded decision, no production code"* — a single `Specs → Red → Implementation` path cannot execute the first two things Layer 2 needs.

### D3 — Judges are differentiated by input, not by effort

Two judges review every candidate. **Judge-SPEC** sees the node's Gherkin, its Charter Acceptance line and its dependency neighbours, and is denied the diff on its first pass; its deliverable is an enumeration of *scenario → the test that proves it*, listing every scenario with no proving test. **Judge-DIFF** sees `base_sha...head_sha` and the repository's documented standards, and is denied the spec.

Reviewing only the changed lines is blind to omission by construction: a requirement never implemented produces zero changed lines, and the `[guard]` bite — which doc 0002 closes on *"the recorded red run against the scratch violation"* — happens in a scratch tree that never enters any diff. Diff-only is one axis of two and must be the second.

Reasoning effort is a budget, not a differentiator. `jd-judge-a` and `jd-judge-b` currently differ in four lines — a name, two letters, and `effort: medium` → `effort: high` — which makes them one reviewer sampled twice; under `judgment-day`'s rule *"fix only severe findings confirmed by both judges"*, correlated judges make dual confirmation nearly free while feeling like corroboration.

Both judges are read-only with no delegation capability. Verdicts bind to an immutable candidate `(base_sha, head_sha, diff_sha256)`. At most two correction rounds; re-judgment sees only the frozen ledger and the fix delta; terminal states are `approved` and `escalated`; disagreement escalates to the human. **A third judge is never added** — a third correlated clone contributes majority-of-correlated-errors, not information.

### D4 — A projection is a rendering — amending ADR 0007 § D3

ADR 0007 D3 rejected a structured sidecar as *"a second source of truth that drifts from the prose within one amendment."* A database table is a worse sidecar than a YAML file, because a YAML file at least appears in the PR diff beside the prose it drifted from. The tension is real and is resolved here rather than ignored.

ADR 0007 D3 already says of renderings: *"Neither rendering is authoritative… the field wins and the rendering is regenerated."* **A projection is a rendering.** What makes it one rather than a sidecar is a column partition plus three mechanical consequences:

> **Invariant.** Every column in the planning schema is exactly one of: **(P)** a pure projection of a byte range of a source document identified by `(path, content_sha256)`, rebuildable by re-running the parser and discardable without information loss; or **(S)** execution state that has no representation in any document. No column is both.

1. (P) rows are deleted and reinserted wholesale on reparse. They hold nothing that could be lost.
2. Ingest refuses to run when the document hash does not match the revision under review — drift is a hard failure, never a silent divergence.
3. There is no write path from the database back to a document, and no update path on a (P) row.

ADR 0007 § D3 gains a dated amendment blockquote recording this, so the resolution is on the record rather than implicit.

### D5 — Delete the duplicated planning artifacts

The loop is lighter only if something is deleted, and the deletion is named: per change, `proposal.md` (the Charter is the proposal), the delta `specs/<capability>/spec.md` (the Gherkin is the spec), `tasks.md` (the scenarios are the task list, in order), `verify-report.md` (replaced by typed evidence on the phase event), and `archive` as a phase (replaced by ticking the completion-checklist box and recording the merge).

Kept: `design.md`, and only where it *is* the deliverable — a `[decision]` node's recorded artifact. Kept: `openspec/specs/` as the promoted, living capability register, which is the durable memory of what the system guarantees; the per-change delta was the duplication, not the register.

Net: one durable artifact per milestone — the milestone-document section, already written — plus the PR.

**Stated honestly:** the loop has eleven stages against SDD's eight phases and adds a review stage SDD never had. It is lighter in artifacts and heavier in gates. Without this deletion the claim of lightness is simply false, and the loop becomes a third pipeline strictly heavier than both.

## Alternatives rejected

| Rejected | Why |
|---|---|
| **A "feature" level** between milestone and spec | D2. It is an eighth level on a seven-level tree, and it cannot be given a cardinality against `[compound]` |
| **Sizing work by "a single fresh context window"** | A 400-line Go diff is nowhere near one, and a `[decision]` node can exhaust one producing zero lines. The 400-line review budget already in `openspec/config.yaml` is the constraint that actually binds |
| **A polymorphic `dependency_edge(from_kind, from_id, …)` table** | No referential integrity, so an edge survives its vertex and the acyclicity check terminates early and reports *clean* on corrupt data. No cascade, and a low-cardinality leading index column on every recursive-CTE hop |
| **A single `depends_on` column (self-FK)** | ADR 0007 measured 68 of 90 milestones with more than one dependency parent. Not a trade-off — an impossibility |
| **Expand–migrate–contract for the `milestone` primary key** | It protects live readers; there are none. Three migrations and three PRs for a table with no rows. Replaced by one migration guarded by a row-count assertion that aborts loudly if the "no rows" inference is wrong |
| **Storing scenarios inside `spec.content`** | The scenario is the unit of claim, of phase state and of agent work. Inside a text column you cannot answer which of seven is green, compute a frontier, or resume after a crash |
| **A stored consensus column on reviews** | Derived state that drifts. A tie is two verdict rows with different outcomes — a query |
| **Adding `refactor` and `changes_requested` to the `spec_phase` CHECK** | Every future phase would be a `DROP CONSTRAINT`/`ADD CONSTRAINT` migration taking `ACCESS EXCLUSIVE`. The phase vocabulary becomes rows |
| **Keeping interval rows for phase state** | Closing an interval requires an UPDATE, which breaks append-only, and is exactly why v1 needed an "at most one open phase" index it then deferred. Events have nothing to keep open, so the invariant disappears instead of being enforced |
| **A skill named `code-review`** | Collides with Claude Code's built-in `/code-review`, which is diff-only and posts inline comments — not this protocol |
| **Naming the loop after a mind, a brain, or any organism** | The repository corrected exactly this in Layer 2's naming on 2026-08-10. The loop is named for what it does |

## Enforcement

Three things must be settled before the first migration, because each is a live contradiction that would block review (PRD 0001 § 3):

1. Promote `openspec/changes/archive/2026-06-22-witsaba-core-tables/specs/witsaba-core-tables/spec.md` verbatim to `openspec/specs/witsaba-core-tables/`, so the requirements this work rescinds appear as rescinded rather than vanishing.
2. Settle the migration reversibility rule in `openspec/AGENTS.md` and amend or rescind the promoted spec's D1/D2 to match. No further migration may cite a rule that does not exist.
3. Add CI running the per-module test and lint commands and `validate_taskgraph.py` over every milestone document. An eleven-gate loop with zero automated enforcement points is a loop of promises, and this costs less than any single stage it protects.

## Consequences

### Positive

- ADR 0007 and the three milestone documents survive untouched; 91 milestones of contract keep their value.
- The frontier, the claim, the evidence and the verdict become queryable facts instead of assertions in a transcript.
- Omission becomes catchable — the failure class that diff-only review cannot see by construction.
- The dead planning chain in `database_administrator` becomes the product's first real capability rather than schema nobody writes.
- The human gains four decisions that were being made by acquiescence, and loses none.
- A force-push during review strands prior verdicts structurally, with no invalidation logic to write or forget.

### Negative / risk

- A parser and an ingest path must exist and stay correct. Two implementations of the ADR-0007 grammar would drift while CI stayed green; mitigated by having the validator emit the projection, not by discipline.
- **State orphaning has no clean answer.** Editing a document reshuffles (P) rows; a scenario whose text changes after going green can be rebound silently (corruption) or discarded (waste). We choose detection and forced re-verification, and record that this is a trade-off, not a solution.
- Eleven stages is more gates than eight phases. If D5's deletion does not land, this is strictly worse than what it replaces.
- Retiring `milestone`/`task`/`spec`/`spec_phase` discards a design that was specified in 49 requirement sections. Rescinding them explicitly is the cost of not silently overwriting them.

### Neutral

- `judgment-day` is not replaced; its bounded-correction machinery is imported wholesale. What changes is judge differentiation, which is currently effort-based and therefore broken.
- The loop is headless. A UI is out of scope and may never be needed.
- `docs/prd/` opens its own append-only numbering sequence; the milestone document that consumes PRD 0001 takes the next number in the `docs/architecture/` stack sequence, with prefix `DF-`.

## References

- [PRD 0001 — The cachicamas delivery loop](../prd/0001-cachicamas-delivery-loop.md)
- [ADR 0007 — Adopt a DAG convention for task graphs](0007-adopt-dag-convention-for-task-graphs.md)
- [ADR 0005 — Promote the agent stack to its own module](0005-promote-agent-stack-to-own-module.md) § D1
- [ADR 0006 — Resolve skill and prompt source of truth](0006-resolve-skill-and-prompt-source-of-truth.md)
- `.claude/skills/task-graph-milestone-doc/references/method.md`
- `openspec/AGENTS.md`, `openspec/config.yaml`
- Matt Pocock, *Skills for Real Engineers* — https://github.com/mattpocock/skills
- Colin Eberhardt, *Putting Spec Kit Through Its Paces* — https://blog.scottlogic.com/2025/11/26/putting-spec-kit-through-its-paces-radical-idea-or-reinvented-waterfall.html
- Jesse Vincent, *Claude Code skills not triggering?* — https://blog.fsck.com/2025/12/17/claude-code-skills-not-triggering/
