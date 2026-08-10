---
name: task-graph-milestone-doc
description: "Trigger: PRD, requirement, architecture doc, milestones document, task graph, waves, DAG, plan implementation. Turn a PRD into a Gherkin task-DAG milestone doc: waves as MVIs, SDD + Strict TDD per node, ADR 0007 shapes."
license: Apache-2.0
metadata:
  author: "braejan"
  version: "2.0"
---

# PRD → task-graph milestone document

## Activation Contract

Load when planning work that will be executed milestone-by-milestone through the SDD pipeline:
receiving a PRD/requirement/architecture document, creating or amending a milestones document
(docs 0002/0003/0004 format), or restructuring an existing task graph.

## Hard Rules

- Follow the full method in `references/method.md`; it is normative. Read it before authoring.
- Output conforms to the ADR 0007 DAG contract (as amended for depth 3): the `Depends on:` /
  `Blocks:` fields are the single source of truth for edges; mermaid and tables are derived
  summaries; a cycle is a bug.
- Waves are MVIs — each ends with demonstrable value. Wave 0 only for foundational work.
- Every milestone declares its **`SDD change:`** slug and runs as one SDD flow; every scenario is
  driven RED → implementation → GREEN → refactor → review (Strict TDD).
- Every `[leaf]`/`[guard]` behavior is a Gherkin scenario per `references/gherkin-authoring.md`;
  the document is implementation-language-agnostic (tool bindings live in its method section only).
- Every wave opens with a rendered mermaid DAG tree of its entire contents (all depths).
- Nodes nest at most 3 levels below a milestone. Ids are append-only after first merge; edits to
  merged content are dated amendment blockquotes.
- Never skip the intake and consistency phases; only the research phase (Phase 1) may be skipped,
  and only on the user's explicit waiver.

## Decision Gates

| Situation | Action |
| --- | --- |
| Source contradicts research/ADRs | Inconsistency register: reconcile or flag to user — never pick silently |
| Node too big for one sitting / > 7 scenarios / > 400 changed lines | Split: append children (inner DAG, ≤ 3 levels) |
| Capability out of scope | Deferred register + named seam, not silence |
| Plan disproved during execution | Living-graph amendment in the same PR that resumes work |
| Doc predates Gherkin (0002–0004) | Validate with `--profile v1`; restructure only on user request |

## Execution Steps

1. Intake the PRD → requirements inventory (Phase 0).
2. Web-research the SOTA → research digest with citations (Phase 1).
3. Consistency check → inconsistency register; surface blockers to the user (Phase 2).
4. Boundary → identifiers → waves → milestones → Gherkin leaves → per-wave renders →
   closing sections (Phases 3–9 of `references/method.md`).
5. Self-audit: run `scripts/validate_taskgraph.py <doc.md>`; fix every error (Phase 10).
6. Adversarial review per document; fix blockers/majors; PR; record in Engram (Phase 11).

## Output Contract

Return: the milestone document path(s); validator output (0 errors); the review verdict with
findings fixed; the PR; and the Engram observation ids saved.

## References

- `references/method.md` — the full normative method (phases 0–11, living graph).
- `references/gherkin-authoring.md` — scenario anatomy, style rules, anti-patterns.
- `scripts/validate_taskgraph.py` — mechanical validator (self-tested; `--self-test`).
- `assets/milestone-doc-template.md` — skeleton document with all required sections.
- Exemplars: `docs/architecture/milestones/0002…0004`; contract: `docs/adr/0007…`.
