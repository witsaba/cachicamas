# ADR 0007: Adopt the DAG convention for task-graph documents

> Status: **Proposed** (2026-07-30)
> Author: cachicamas SDD pipeline (parent orchestrator)
> Companion: [ADR 0005](./0005-promote-agent-stack-to-own-module.md) — module boundaries the
> graphs plan against · docs [0002](../architecture/milestones/0002-cachicamas-ai-layer-1-task-graph.md) /
> [0003](../architecture/milestones/0003-cachicamas-agent-layer-2-task-graph.md) /
> [0004](../architecture/milestones/0004-cachicamas-coding-layer-3-task-graph.md) — the exemplar documents
> Skill: `.claude/skills/task-graph-milestone-doc/SKILL.md` — executes this convention for new documents

---

## Resolved TOC

- [D1 — A task graph is a DAG over a containment tree](#d1--a-task-graph-is-a-dag-over-a-containment-tree) — the two structures, named
- [D2 — The machine-readable contract](#d2--the-machine-readable-contract) — what a renderer may rely on
- [D3 — Two renderings, one source of truth](#d3--two-renderings-one-source-of-truth) — in-file mermaid and the web viewer
- [D4 — The canonical node-type palette](#d4--the-canonical-node-type-palette) — one color vocabulary everywhere

---

## Context

Docs 0002/0003/0004 call themselves *task graphs* but never state what kind of graph they are, and
the first attempt to render them exposed the ambiguity. A validation pass over the three parsed
documents (2026-07-30) measured:

| Doc | Milestones | Multi-parent milestones | Cross-wave dependency edges | Cross-layer edges |
| --- | --- | --- | --- | --- |
| 0002 (AI) | 41 | 29 | 38 | 0 |
| 0003 (AG) | 24 | 21 | 30 | 5 |
| 0004 (CO) | 25 | 18 | 35 | 11 |

68 of 90 milestones have more than one dependency parent. **The dependency structure is a directed
acyclic graph, not a tree**, and cannot be drawn as one without duplicating nodes or dropping
edges. At the same time, the *containment* structure — document → wave → milestone → node — **is**
a strict tree in all three documents: every milestone belongs to exactly one wave, every node to
exactly one milestone.

The documents were parseable into a renderer only because their formatting happened to be
consistent. Nothing guaranteed it: the shape was style, not contract. This ADR makes it contract,
so that any conforming document can be drawn — in the file itself via mermaid, or interactively on
the web — without per-document parser work.

## Decision

### D1 — A task graph is a DAG over a containment tree

Every task-graph document describes **two structures**, and every consumer (writer, reviewer,
renderer, agent) names them by these terms:

- **The containment tree** — document → wave → milestone → node, strict tree, single parent at
  every level. This is the *drawable skeleton*: the wave is the root of its subtree, milestones are
  its children, typed nodes are the leaves.
- **The dependency DAG** — the edges declared by `Depends on:` / `Blocks:` fields, at milestone and
  node level. Multi-parent, cross-wave, and cross-document edges are all legal and expected.
  Acyclicity is an invariant: a dependency cycle is a bug in the document.

A rendering draws the containment tree as its layout and overlays the dependency DAG on top
(edges, chips, or highlights — D3). A rendering that tries to lay out by dependency edges alone is
wrong by construction.

This vocabulary — *task DAG*, *containment tree*, *dependency edge* — is the convention across the
repo: in planning documents, in ADRs, in Engram records, and in skills.

### D2 — The machine-readable contract

A conforming document guarantees the following shapes, which a renderer may rely on without
document-specific code. These are exactly the shapes docs 0002/0003/0004 already use — this ADR
promotes them from house style to contract:

1. **Waves:** `## Wave <id> — <name>` headings. Everything until the next `##` heading belongs to
   the wave.
2. **Milestones:** `### XX-NN — <title>` headings inside a wave, followed by an optional
   `SDD change:` line and a `**Charter**` bullet list whose entries are `- **<Label>:** <text>`
   (Goal · Deliverable · Acceptance · Depends on · Out of scope · Notes).
3. **Nodes:** `#### XX-NN.p — <title> `[type]`` headings, `type` drawn from the doc 0002 node
   grammar (`leaf`, `guard`, `decision`, `mechanical`, `compound`), followed by a test/check/closing
   list of numbered items and `- **<Label>:**` metadata bullets.
4. **Dependency edges:** every edge is a bare node id (`XX-NN` or `XX-NN.p`) appearing in a
   `Depends on:` or `Blocks:` field. Prose may surround the ids; the ids alone are the edge list.
   Non-id dependencies (an ADR, a merged state) are prose, not edges. **`Parallel with:` is not an
   edge** and must be excluded before any cycle check — mutual parallelism read as dependency
   produces false 2-cycles (found and fixed in the first audit of this convention: AI-26 ∥ AI-27).
5. **Wave metadata:** the delivery-sequence table carries, per wave: milestone range, gate (where
   applicable), and exit condition.

The **`Depends on:` fields are the single source of truth for edges.** The global mermaid graph,
the delivery table, and parallelism notes are derived summaries (the skill's Step 6 audit already
enforces this direction).

> **Amended 2026-08-10 — item 3 widened to three nesting levels and Gherkin bodies; item 4
> follows.** Layer 1 completed with node ids capped at two levels below the milestone
> (`XX-NN.p.q`, used exactly twice in 164 nodes), and the v2 milestone format needs one more
> level for milestones that are genuinely a DAG of DAGs. Item 3 now reads: nodes are
> ~~`#### XX-NN.p — <title> `[type]`` headings~~ **`#### XX-NN.p`, `##### XX-NN.p.q`, and at most
> `###### XX-NN.p.q.r` headings — three levels below the milestone, never more — each carrying a
> backticked `[type]` tag**, followed by ~~a test/check/closing list of numbered items~~ **either
> Gherkin scenarios (fenced ` ```gherkin ` block or `- **Scenario:**` bullets — the v2 form for
> `[leaf]`/`[guard]`) or a numbered test/check/closing list (the pre-Gherkin form, still valid in
> merged documents)**, and `- **<Label>:**` metadata bullets. Item 4's edge-id shape widens
> accordingly: a bare id is ~~`XX-NN` or `XX-NN.p`~~ **`XX-NN` followed by up to three dotted
> numeric segments**. The mechanical validator
> (`.claude/skills/task-graph-milestone-doc/scripts/validate_taskgraph.py`) enforces both bounds.

### D3 — Two renderings, one source of truth

- **In-file:** mermaid blocks inside the document remain the offline/GitHub rendering. They are
  summaries, always re-derivable from the contract fields, and they use the D4 palette so a guard
  looks like a guard in every document.
- **Web:** interactive viewers are *generated by parsing the contract*, never hand-maintained.
  The reference implementation lives with this repo's tooling (parser + single-file HTML explorer);
  any future viewer consumes the same contract.
- Neither rendering is authoritative. When a rendering and a `Depends on:` field disagree, the
  field wins and the rendering is regenerated.

> **Amended 2026-08-10 — v2 documents add a mandatory per-wave rendering.** Every `## Wave`
> section of a document authored under the v2 skill opens with a mermaid flowchart of the wave's
> **entire containment subtree** (milestones as subgraphs, nodes at every depth) with dependency
> edges overlaid, in the D4 palette. It is a derived summary like every other rendering — the
> `Depends on:` fields still win — but its *presence* is contract for v2 documents, so a reader
> sees each wave's whole DAG without reconstructing it from fields. Merged pre-v2 documents
> (0002–0004) are not restyled retroactively.

> **Amended 2026-08-11 — a projection into a datastore is a rendering.** [ADR 0008 § D4](0008-adopt-the-cachicamas-delivery-loop.md#d4--a-projection-is-a-rendering--amending-adr-0007--d3)
> extends this section to cover a database projection of the contract, resolving the apparent
> conflict with the sidecar rejected in *Alternatives rejected*. A projection qualifies as a
> rendering — not as a second source of truth — only under this invariant:
>
> > Every column in the projection is exactly one of: **(P)** a pure projection of a byte range of a
> > source document identified by `(path, content_sha256)`, rebuildable by re-running the parser and
> > discardable without information loss; or **(S)** execution state that has no representation in any
> > document. No column is both.
>
> Mechanically: (P) rows are deleted and reinserted wholesale on reparse; ingest refuses to run when
> the document hash does not match the revision under review, so drift is a hard failure rather than a
> silent divergence; and there is no write path from the datastore back to a document and no update
> path on a (P) row. The `Depends on:` fields still win. A projection that stores anything authored —
> anything a human or agent edits *there* rather than in the document — is the rejected sidecar and
> this amendment does not license it.

### D4 — The canonical node-type palette

One color vocabulary for node types, used by every mermaid block and every viewer. Color never
encodes alone: the type tag text always accompanies it (the aphantasia/accessibility rule —
structure must be readable as text, not only visible as picture).

| Node | Fill | Stroke | Meaning |
| --- | --- | --- | --- |
| wave (root) | layer accent | layer accent | L1 `#dcfce7`/`#15803d` · L2 `#dbeafe`/`#1d4ed8` · L3 `#fef3c7`/`#b45309` |
| milestone / `[compound]` | layer soft accent | layer accent | container node |
| `[leaf]` | `#e2e8f0` | `#94a3b8` | behavior, red→green→refactor |
| `[guard]` | `#fef3c7` | `#d97706` | mechanical check, must bite |
| `[decision]` | `#ede9fe` | `#8b5cf6` | recorded artifact, closing checklist |
| `[mechanical]` | `#f1f5f9` | `#cbd5e1` | objective evidence, no red/green |
| `[compound]` | `#ccfbf1` | `#14b8a6` | children + exit check |

Canonical mermaid classDefs (copy verbatim into new documents):

```text
classDef leaf fill:#e2e8f0,stroke:#94a3b8,color:#1f2937
classDef guard fill:#fef3c7,stroke:#d97706,color:#1f2937
classDef decision fill:#ede9fe,stroke:#8b5cf6,color:#1f2937
classDef mechanical fill:#f1f5f9,stroke:#cbd5e1,color:#1f2937
classDef compound fill:#ccfbf1,stroke:#14b8a6,color:#1f2937
```

Existing documents keep their current mermaid styling — restyling merged documents is churn with
no information gain. New blocks, and any block touched for another reason, adopt the palette.

## Alternatives rejected

- **A structured sidecar (YAML/JSON front-matter or a `.dag` file per document).** A second source
  of truth that drifts from the prose within one amendment. The markdown *is* the data; the
  contract makes it parseable.
- **Laying out renderings by dependency edges.** Impossible without node duplication (68/90
  milestones are multi-parent) and unreadable when forced. The containment tree is the layout; the
  DAG is the overlay.
- **Renaming the documents to "task DAG".** Ids and filenames are append-only; the *term* is
  adopted going forward without renaming anything merged.

## Consequences

- The `task-graph-milestone-doc` skill states the contract (D2) and the palette (D4), so every new
  document is born render-ready.
- Renderers may hard-depend on the D2 shapes; a document that breaks them is non-conforming and
  fixed as a bug.
- The Step 6 self-audit gains one mechanical check: the edge list extracted from `Depends on:`
  fields must be acyclic.
- Engram decisions and observations about planning use the D1 vocabulary, so future recall matches
  the terms in the files.

## References

- Validation findings: session of 2026-07-30 (counts in [Context](#context)).
- Interactive reference viewer: parser + single-file explorer (artifact
  `claude.ai/code/artifact/b334fa64-a671-4b8c-bfef-64e99bd902b8`).
- Doc 0002 § "Node grammar" — owns the type definitions this ADR colors.
