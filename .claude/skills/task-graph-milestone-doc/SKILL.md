---
name: task-graph-milestone-doc
description: Turn a PRD, project, or requirement into a fractal TDD task-graph milestone document — the docs 0002/0003/0004 format — with typed nodes, milestone charters, waves and entry gates, a traceability spine, and a mandatory adversarial review pass. Use whenever planning work that will be executed milestone-by-milestone through the SDD pipeline.
---

# Specify a PRD / project / requirement as a graph task milestone document

This skill encodes the method behind `docs/architecture/milestones/0002…0004`. The output is one
markdown document that a sequence of implementers (human or agent) can walk leaf-by-leaf with
red–green–refactor, where every claim is testable, every dependency is an edge, and every deferral
is a recorded decision. The exemplars are the three shipped documents:

- `docs/architecture/milestones/0002-cachicamas-ai-layer-1-task-graph.md` (from-zero plan)
- `docs/architecture/milestones/0003-cachicamas-agent-layer-2-task-graph.md` (plan gated on an upstream freeze)
- `docs/architecture/milestones/0004-cachicamas-coding-layer-3-task-graph.md` (multi-track gates, executes an ADR)

Read at least one exemplar before writing. Doc 0002 owns the canonical definitions of the node
grammar, leaf anatomy, split triggers, and the living-graph clause — a new document **cites** those
sections; it never re-defines them.

## Step 0 — Gather the sources

Collect, and cite by section throughout: the PRD/requirement itself; the governing architecture
reference; every ADR whose decisions the plan executes; prior review findings (Engram) that name
defects or gaps; and the sibling plans it must interoperate with. If a normative source contradicts
another, the document must reconcile or flag the conflict explicitly (see doc 0003's AG-06 note for
the pattern) — never silently pick a side.

## Step 1 — Fix the boundary before any milestone exists

Write the **Layer/Scope boundary** section first:

- **Owns:** every behavior the plan will implement, as a prose list.
- **Must not own:** everything adjacent that belongs elsewhere, each with its owner named.
- **Wording traps:** 1–3 phrases people will predictably misread, pre-empted (e.g. "the loop
  executes tools" vs "the loop *schedules* execution").

A milestone that later proves to own something outside the boundary is a bug in one of the two —
fix the document, on the record.

## Step 2 — Choose identifiers and structure

- **Prefix:** two letters + `-NN`, unique per document (`AI-`, `AG-`, `CO-` are taken). Nodes are
  `XX-NN.p` and `XX-NN.p.q`.
- **Append-only rule (verbatim from doc 0002):** once the first milestone merges, ids are never
  renumbered. New work appends the next free number; logical insertion points use a `Blocks:`
  field. Node splits append children, never renumber siblings. Amendments are blockquotes
  (`> **Amended YYYY-MM-DD** …`) with struck-through superseded text — never silent edits.
- **Required document sections, in order:** status header (counts, first milestone, entry gate,
  references, date, append-only rule) → authoring constraint → *Outcome first* → quick navigation →
  boundary → SDD milestone rules → method-inheritance section citing doc 0002 → entry gate → global
  dependency graph (mermaid) + delivery-sequence table → waves of milestones → completion checklist
  → explicitly-deferred register → traceability spine.

**The authoring constraint (non-negotiable):** the document states *behaviors* and *what a test
must prove*. It never invents type names, field names, or signatures — each milestone's SDD cycle
owns those. It cites only artifacts that exist at HEAD; "shipped" may only describe things that are
actually merged.

## Step 3 — Decompose: milestones into waves, waves behind gates

- **One primary contract or behavior per milestone.** Prefer < 250 changed lines, reassess before
  400 — the review budget is a split trigger, not advice.
- **Waves are dependency order, not calendar order.** Each wave row in the delivery table carries
  its milestones, its **gate** (the upstream milestone that must freeze first), and a one-line
  **exit condition**.
- **Multiple tracks** when subsets have different gates (doc 0004's resource/contract/session
  tracks). A "priced early start" is legal only if recorded: what may begin early, against which
  unfrozen surface, accepting which rework.
- **Every cross-document dependency must be a named, frozen promise in the upstream document** —
  usually an item in its readiness/handoff milestone. If the plan needs a capability the upstream
  doc doesn't promise (compact-on-demand, seeded construction, a budget source), the move is an
  upstream amendment under its living-graph clause, never an assumption and never a workaround.

## Step 4 — Write each milestone

Every milestone section is: heading (`### XX-NN — <imperative title>`) · SDD change slug · what it
closes (finding/gap/ADR ids) · **Charter** · optional per-milestone mermaid · child nodes.

**Charter fields:** Goal · Deliverable · Acceptance (observable, testable) · Depends on (+
`Blocks:` when it gates others) · Out of scope (each item with the node that owns it) · Notes for
anything an implementer would otherwise trip on. The charter is normative scope; the node graph
below is its decomposition, and the two must agree at every depth.

**Node grammar** (doc 0002 owns the definitions — summary):

| Type | Closes by |
| --- | --- |
| `[compound]` | all children closed + its own one-line exit check |
| `[leaf]` | every test-list item taken red → green → refactored, in order |
| `[guard]` | mechanical check shown to **bite** (recorded red vs a scratch violation), then green |
| `[decision]` | a recorded artifact answering every closing-checklist question |
| `[mechanical]` | recorded objective check evidence (build output, diff scan) — no red/green |

**Fractal invariants:** a node is compound or leaf, never both; a compound's scope is exactly the
union of its children (100 % rule); siblings never overlap — at every depth. Any subtree cut out
alone must be a well-formed plan.

**Leaf anatomy:** 1–7 test-list items phrased as observable behavior (`WHEN … THEN …` or a
property) — "add a mutex" is illegal, "two concurrent streams each observe contiguous sequences
under `-race`" is legal. Regression assertions are marked `*(pin)*`. Plus: Depends on · Out of
scope (what keeps siblings exclusive) · Split if (pre-declared fission trigger where foreseeable).

**Ordering:** the first behavior leaf of any capability-bearing milestone is its **walking
skeleton** — the thinnest end-to-end path; later leaves widen it, never open a second unintegrated
front. Error paths follow happy paths; hardening follows function.

**Evidence gate:** one command closes a leaf (e.g. green `make test` in the module). Exceptions
(two-module tests, opt-in live checks, binary builds) must be scoped and named in the method
section, per leaf.

## Step 5 — The closing sections

- **Completion checklist:** every box must have a closing node; the readiness/handoff milestone
  walks it item-by-item with citations.
- **Explicitly-deferred register:** absence must read as a decision. Every deferral names the
  *seam* where the capability will later attach and who decided the deferral.
- **Traceability spine (two-way):** every finding, gap, ADR clause, and checklist item → the
  node(s) closing it; every node traces back to scope. **One row per half** when a concern splits
  across documents (protocol half vs policy half) — unowned halves are the most common coverage
  hole. A finding with no node, or a node with no purpose, is a bug in the document.

## Step 6 — Mechanical self-audit (before any review)

1. **Edges vs summaries:** the global mermaid graph, delivery table, and any parallelism notes are
   *summaries* of the milestone-level `Depends on:` fields — audit each against the fields; they
   drift (this was 4 of the review findings on docs 0003/0004).
2. **Id audit:** `grep -o 'XX-[0-9]*\(\.[0-9]*\)*' | sort | uniq -c` — every cited id must exist in
   the target document's headings. Cite **milestone-level ids only** into documents that are not
   yet frozen; sub-node citations survive only into merged docs.
3. **Anchor check:** verify every intra-repo `#anchor` against the real GitHub slug. Trap: a
   heading containing ` — ` (em dash) slugs to a **double hyphen** (`alive--the`), not one.
4. **Grammar audit:** no bite-proof-less `[guard]`, no mechanical scan hiding inside a `[leaf]`
   test list, every `[decision]` has a closing checklist, pins marked.
5. **Scope-of-citation check:** any allowlist, enum, or vocabulary cited from another document must
   actually cover this document's scope (ADR 0005 § D3's allowlist covers Layer 1 spans only — a
   Layer 2 doc citing it needed its own vocabulary decision).

## Step 7 — Adversarial review, then improve, then PR

Spawn **one independent adversarial agent per document** (parallel across documents). The prompt
must: list the document and every normative reference by path; instruct the agent to walk **every
cross-reference and verify it against the referenced file**; give it the seven review dimensions —
coverage (every owned concern has a node; nothing double-owned; nothing a non-goal forbids),
boundary violations, dependency-graph correctness, node-grammar conformance, internal consistency
(charter vs nodes, spine vs nodes, checklist vs nodes, anchors), feasibility break points
(one-sitting leaves, untestable items, decisions sequenced after the code that needs them), and
first-implementer trip points; and demand **verified findings only**, each with severity
(blocker/major/minor), dimension, exact location, and 1–3 sentences of evidence, most severe first,
plus an overall verdict. Fix every blocker and major (and the cheap minors) before commit — adding
milestones/nodes is fine pre-merge; renumbering is fine only pre-merge.

Finish with a PR containing the document(s), and record the outcome in Engram (decision +
session-summary observations, per the session-close ritual).

## The living graph (execution-time rule, stated in every document)

Implementation *will* disprove parts of the plan. When a leaf's first red test cannot be driven
green in small steps: revert to green; record the discovery as graph structure (append the
prerequisite node, draw the edge, move remaining items into children if the leaf became compound);
land the amendment **in the same PR** that resumes work. Newly discovered *test cases* append to
the owning leaf's list instead.

## Method sources

Canon TDD test lists (Beck) · Mikado method · HTN planning · WBS 100 % rule · INVEST/SPIDR/Elephant
Carpaccio · Walking skeleton (Cockburn) / GOOS · this repo's openspec SDD conventions. Full
rationale: doc 0002 § "Method sources".
