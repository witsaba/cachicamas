# The task-graph milestone method (v2)

This is the full normative method for task-graph milestone documents, distilled from its first
complete execution (a 42/42-milestone delivery; the current project's exemplars are listed in
`SKILL.md` § Project Bindings). The output is one markdown document that a sequence of
implementers (human or agent) can walk leaf-by-leaf with SDD + Strict TDD, where every behavior
is a Gherkin scenario, every dependency is an edge, and every deferral is a recorded decision.
This skill owns the canonical node grammar and the renderable shapes (the DAG contract below);
each project ratifies that contract in an ADR of its own and names a founding method document
(both in its Project Bindings). A new document **cites** its project's ratifying ADR and founding
document; it never re-defines them.

The v2 deltas over the shipped exemplars: a mandatory intake → research → consistency phase before
any authoring; Gherkin scenarios instead of free-form test items; a mandatory rendered DAG tree per
wave; nesting to three levels; and normative text that is implementation-language-agnostic.

## The DAG contract (every document must conform; ratified per project by its DAG-convention ADR)

Two structures, named by these terms everywhere (files, ADRs, Engram, skills):

- **The containment tree** — document → wave → milestone → node. Strict tree, single parent at
  every level.
- **The dependency DAG** — the edges declared by `Depends on:` / `Blocks:` fields. Multi-parent,
  cross-wave, and cross-document edges are legal; a **cycle is a bug**.

Renderer-normative shapes (contract, not style): `## Wave <id> — <name>` · `### XX-NN — <title>`
with an `SDD change:` line and `**Charter**` bullets (`- **<Label>:** …`) ·
`#### XX-NN.p — <title>` (through `###### XX-NN.p.q.r` at maximum depth) with a backticked
`[type]` tag · every dependency edge is a **bare node id** inside a `Depends on:`/`Blocks:` field
(prose may surround ids; the ids alone are the edge list; `Parallel with:` is *not* an edge) · the
delivery-sequence table carries range, gate, and exit condition per wave. The `Depends on:` fields
are the **single source of truth for edges**; every mermaid block and table is a derived summary —
when they disagree, the field wins and the rendering is regenerated. New mermaid blocks copy the
canonical type palette verbatim (guard amber, decision violet, leaf slate, mechanical gray,
compound teal — the classDefs in `assets/milestone-doc-template.md`, restated in the ratifying
ADR's § D4); color never encodes alone. Never introduce a YAML/JSON sidecar or front-matter DAG —
rejected at contract ratification as a second source of truth.

## Phase 0 — Intake

Read the PRD / requirement / architecture document **completely** before forming any structure.
Produce a requirements inventory: every distinct claim, requirement, constraint, and non-goal the
source states, each with a stable handle (its section anchor or a short id you assign). This
inventory seeds the traceability spine — a requirement that never reaches a node is the failure
mode the spine exists to catch.

## Phase 1 — Deep research (state of the art)

Before authoring, run a web research pass on the problem domain: the current state of the art,
authoritative guidelines, known failure modes, and prior art worth imitating or avoiding. Record
the result as a **research digest** — findings with citations (link + one-line takeaway each) and
an explicit statement of what the research changes about the plan. The digest lands in the
document's *Sources and research* section, before the first wave. Research that changed nothing is
also a finding — say so. Skipping this phase is only legal when the user explicitly waives it.

## Phase 2 — Consistency check

Cross-examine the PRD against the research digest, the governing architecture references, every
ADR the plan executes, prior review findings (Engram), and the sibling plans it must interoperate
with. Produce an **inconsistency register**: each conflict, its two sides cited by section, and its
disposition — *reconciled* (with the reconciliation) or *flagged to the user* (blocking). Never
silently pick a side. An empty register is recorded as an empty register, not omitted. Blocking
flags go to the user before Phase 3 starts.

## Phase 3 — Fix the boundary

Write the **scope boundary** section before any milestone exists:

- **Owns:** every behavior the plan will implement, as a prose list.
- **Must not own:** everything adjacent that belongs elsewhere, each with its owner named.
- **Wording traps:** 1–3 phrases people will predictably misread, pre-empted.

A milestone that later proves to own something outside the boundary is a bug in one of the two —
fix the document, on the record.

**The authoring constraint (non-negotiable):** the document states *behaviors as Gherkin
scenarios* and *what evidence closes each node*. It never invents type names, field names,
signatures, or framework calls — each milestone's SDD cycle owns those. It cites only artifacts
that exist at HEAD. And it is **implementation-language-agnostic**: normative text names roles
("the evidence-gate command", "the race-condition detector", "the import guard"), while the
document's *method section* binds each role to this project's concrete tool once, in one place.

## Phase 4 — Identifiers and structure

- **Prefix:** two letters + `-NN`, unique per document (the Project Bindings list the prefixes
  already taken). Nodes are `XX-NN.p`, `XX-NN.p.q`, and at most `XX-NN.p.q.r` — **three levels
  below the milestone, never more**. Depth is a split budget, not a target: use depth 2–3 only
  when a node is genuinely a DAG of its own; the first execution needed depth 2 exactly twice in
  164 nodes.
- **Append-only rule:** once the first milestone merges, ids are never renumbered. New work
  appends the next free number; logical insertion points use a `Blocks:` field. Node splits append
  children, never renumber siblings. Amendments are dated blockquotes
  (`> **Amended YYYY-MM-DD** …`) with struck-through superseded text — never silent edits.
- **Required document sections, in order:** status header (counts, first milestone, entry gate,
  references, date, append-only rule) → authoring constraint → *Outcome first* → quick navigation
  → **sources and research** (digest + inconsistency register) → scope boundary → SDD milestone
  rules + method bindings (evidence-gate command, TDD cycle, language-specific tools) →
  method-inheritance citation of the project's founding method document → entry gate (mandatory when anything upstream must
  freeze first) → global dependency graph (mermaid) + delivery-sequence table → waves → completion
  checklist → explicitly-deferred register → traceability spine.

## Phase 5 — Waves

- **A wave is an MVI — a Minimum Valuable Implementation.** Each wave ends with something of
  demonstrable value, not a pile of parts. Number waves 1, 2, 3…; use **wave 0 only for a
  foundational wave** (scaffolding, guards, vocabulary) whose value is that everything after it is
  cheap and safe.
- **Waves are dependency order, not calendar order.** Each wave row in the delivery table carries
  its milestones, its **gate** (the upstream milestone or external event that must freeze first),
  and a one-line **exit condition** — the observable statement of the wave's value.
- **Multiple tracks** when subsets have different gates. A "priced early start" is legal only if
  recorded: what may begin early, against which unfrozen surface, accepting which rework.
- **Every cross-document dependency must be a named, frozen promise in the upstream document** —
  usually an item in its readiness/handoff milestone. If the plan needs a capability the upstream
  doc doesn't promise, the move is an upstream amendment under its living-graph clause, never an
  assumption.

## Phase 6 — Milestones and their inner DAGs

Every milestone section is: heading (`### XX-NN — <imperative title>`) · **`SDD change:` line
(mandatory — 42/42 in the first execution)** naming the kebab-case SDD change slug this milestone runs as,
plus what it closes (finding/gap/ADR/requirement ids) · **Charter** · the milestone's rendered
node DAG (mermaid) when it has 2+ nodes · child nodes.

- **One primary contract or behavior per milestone**, sized to the review budget (prefer < 250
  changed lines, reassess before 400 — a split trigger, not advice).
- **Charter fields:** Goal · Deliverable · Acceptance (Gherkin-shaped where it states behavior) ·
  Depends on (+ `Blocks:` when it gates others) · Out of scope (each item with its owner) · Notes.
  The charter is normative scope; the node graph below is its decomposition, and the two must
  agree at every depth.
- **Execution contract:** each milestone runs as **one SDD flow** under its declared change name;
  its leaves become the SDD tasks, and each scenario is driven **RED → implementation → GREEN →
  refactor (performance, clean code, the idioms of the implementation language) → review**, in
  that order (see `gherkin-authoring.md`).

**Node grammar** (canonical here; a project's founding document may restate and extend it):

| Type | Closes by |
| --- | --- |
| `[compound]` | all children closed + its own one-line exit check |
| `[leaf]` | every scenario taken red → green → refactored, in order |
| `[guard]` | mechanical check shown to **bite** (recorded red vs a scratch violation), then green |
| `[decision]` | a recorded artifact answering every closing-checklist question |
| `[mechanical]` | recorded objective check evidence — no red/green |

**Fractal invariants:** a node is compound or leaf, never both; a compound's scope is exactly the
union of its children (100 % rule); siblings never overlap — at every depth, including depth 3.
Any subtree cut out alone must be a well-formed plan.

**Ordering:** the first behavior leaf of any capability-bearing milestone is its **walking
skeleton** — the thinnest end-to-end path; later leaves widen it, never open a second unintegrated
front. Error paths follow happy paths; hardening follows function.

**Evidence gate:** one command closes a leaf, declared once in the method section (bound there to
the project's language and toolchain — the first execution bound it to a `make test` target running
the race detector). Exceptions must be scoped and named per node.

## Phase 7 — Gherkin

Every `[leaf]` carries 1–7 scenarios in Given/When/Then form; every `[guard]` carries its bite as
a scenario. `[decision]` and `[mechanical]` close by checklist/evidence, not scenarios. Full
anatomy, style rules, and anti-patterns: `gherkin-authoring.md`. This is what makes the document
language-agnostic *and* directly executable as TDD: the RED test is the scenario, transcribed.

## Phase 8 — Render every wave's DAG

**Every `## Wave` section opens with a mermaid flowchart showing the wave's entire tree**: every
milestone, every node at every depth (containment as subgraphs, dependency edges as arrows),
using the canonical palette with type-tag text on every node. This is mandatory — a reader
must see the whole DAG of a wave without reconstructing it from fields. The global graph
(wave-level) and per-milestone graphs stay as before. All three renderings are derived summaries:
regenerate them whenever a `Depends on:` field changes; on disagreement the field wins.

## Phase 9 — Closing sections

- **Completion checklist:** every box must have a closing node; the readiness/handoff milestone
  walks it item-by-item with citations.
- **Explicitly-deferred register:** absence must read as a decision. Every deferral names the
  *seam* where the capability will later attach and who decided the deferral.
- **Traceability spine (two-way):** every PRD requirement (Phase 0 inventory), research finding
  that changed the plan (Phase 1), inconsistency disposition (Phase 2), ADR clause, and checklist
  item → the node(s) closing it; every node traces back to scope. One row per half when a concern
  splits across documents — unowned halves are the most common coverage hole. A requirement with
  no node, or a node with no purpose, is a bug in the document.

## Phase 10 — Mechanical self-audit

1. Run `scripts/validate_taskgraph.py <doc.md>` — it enforces heading grammar, unique ids, edge
   resolution, acyclicity (naming the cycle's edges), depth ≤ 3, compound/leaf exclusivity,
   Gherkin presence per leaf, and the per-wave mermaid mandate. Fix every error before review.
2. **Edges vs summaries:** audit the global mermaid, per-wave trees, delivery table, and
   parallelism notes against the `Depends on:` fields — they drift.
3. **Anchor check:** verify every intra-repo `#anchor` against the real GitHub slug. Trap: a
   heading containing ` — ` (em dash) slugs to a **double hyphen**, not one.
4. **Grammar audit the script can't see:** no bite-proof-less `[guard]`, no mechanical scan hiding
   inside a `[leaf]` scenario list, every `[decision]` has a closing checklist, pins marked.
5. **Scope-of-citation check:** any allowlist, enum, or vocabulary cited from another document
   must actually cover this document's scope.

## Phase 11 — Adversarial review, then PR, then Engram

Spawn **one independent adversarial agent per document**. The prompt must: list the document and
every normative reference by path; instruct the agent to walk **every cross-reference and verify
it against the referenced file**; give it the seven review dimensions — coverage (every owned
concern has a node; nothing double-owned; nothing a non-goal forbids), boundary violations,
dependency-graph correctness, node-grammar conformance, internal consistency (charter vs nodes,
spine vs nodes, checklist vs nodes, anchors), feasibility break points (one-sitting leaves,
untestable scenarios, decisions sequenced after the code that needs them), and first-implementer
trip points; and demand **verified findings only**, each with severity (blocker/major/minor),
dimension, exact location, and 1–3 sentences of evidence, most severe first, plus an overall
verdict. Fix every blocker and major (and the cheap minors) before commit — adding
milestones/nodes is fine pre-merge; renumbering is fine only pre-merge.

Finish with a PR containing the document(s), and record the outcome in Engram (decision +
session-summary observations, per the session-close ritual).

## The living graph (execution-time rule, stated in every document)

Implementation *will* disprove parts of the plan. When a scenario's first red test cannot be
driven green in small steps: revert to green; record the discovery as graph structure (append the
prerequisite node, draw the edge, move remaining scenarios into children if the leaf became
compound); land the amendment **in the same PR** that resumes work. Newly discovered *scenarios*
append to the owning leaf's list instead. The first execution used this 35 times — the amendment
blockquote is the normal state of a living document, not an exception.

## Method sources

Canon TDD test lists (Beck) · BDD/Gherkin (North, Chelimsky) · Mikado method · HTN planning · WBS
100 % rule · INVEST/SPIDR/Elephant Carpaccio · Walking skeleton (Cockburn) / GOOS · the project's
SDD conventions. Full rationale: the founding method document's § "Method sources".
