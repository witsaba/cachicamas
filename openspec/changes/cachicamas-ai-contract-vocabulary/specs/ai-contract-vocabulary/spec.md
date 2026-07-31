# Spec — Layer 1 contract vocabulary

> **Change**: `cachicamas-ai-contract-vocabulary`
> **Milestone**: AI-01 · **Node**: AI-01.1 `[decision]`
> **Phase**: spec (delta)
> **Canonical spec**: `openspec/specs/ai-contract-vocabulary/spec.md` — created by `sdd-archive` from this delta
> **Format**: RFC 2119 keywords + Given/When/Then, per `openspec/config.yaml`
> **Date**: 2026-07-31
> **Requirement IDs**: `R-AIV-0NN` · **Scenario IDs**: `S-AIV-0NN`

---

## Purpose

AI-01.1 is a `[decision]` leaf. It ships no Go, so there is no runtime behavior to specify. The subject of this spec is the **artifact**: `decision.md`, the recorded Layer 1 contract vocabulary. Every requirement below constrains that document, and every scenario is a property a reviewer can check against it by inspection, deterministically, without running anything.

This distinction is load-bearing and is stated once here so no later phase mistakes it: a scenario in this spec reads "given the register, when …, then …" — the register is the system under test.

---

## Definitions used by this spec

- **The register** — the categorized table of terms in `decision.md`, comprising all six categories.
- **A row** — one term in the register: its identifier, its term name, its definition, its owning milestone, and its provenance.
- **Owning milestone** — the milestone identifier `AI-NN` whose charter *defines* the term, as opposed to any charter that merely uses it.
- **Excluded term** — a term named in the register's exclusion category, which Layer 1 deliberately does not own; its owner is a layer or a composition root, not a Layer 1 milestone.
- **Wording trap** — one of the two sentences quoted from doc 0002's *Layer boundary* section.

---

## R-AIV-001 — The artifact exists and is singular

The change MUST produce exactly one vocabulary artifact, at `openspec/changes/cachicamas-ai-contract-vocabulary/decision.md`. No other file in the repository MAY carry a competing definition of a Layer 1 term. The artifact MUST be the single source for Layer 1 term ownership until it is superseded by an appended decision node under AI-01.

### Scenarios

- **S-AIV-001** — Given the change directory, when a reviewer lists its files, then exactly one file named `decision.md` is present AND it contains the register AND no other artifact of this change restates a term definition as normative.

---

## R-AIV-002 — One definition per term

Every term in the register MUST carry exactly one definition. A term MUST NOT be defined twice, in two categories, or with two wordings. Where a concept recurs in a later milestone, the register MUST express the recurrence as a cross-reference from the single owning row, never as a second definition.

### Scenarios

- **S-AIV-002** — Given the register, when a reviewer collects every term name across all six categories, then no term name appears in more than one row AND every row's definition text is unique.
- **S-AIV-003** — Given the term `call ordinal`, which the AI-01.1 closing checklist groups stream-side while the concept originates in the tool-call content contract, when a reviewer inspects the register, then exactly one row defines it AND that row names AI-09 as owner AND the row cross-references AI-18.3 and AI-30.5 as restatements rather than defining the term again there.

---

## R-AIV-003 — One owning milestone per term

Every non-excluded term in the register MUST name exactly one owning milestone, expressed as a current `AI-NN` identifier from doc 0002. Two rows MUST NOT claim the same term. A row MUST NOT name two owners, name a range, or leave the owner unstated.

### Scenarios

- **S-AIV-004** — Given the register, when a reviewer reads the owner column of every non-excluded row, then each cell holds exactly one `AI-NN` identifier AND that identifier exists as a milestone heading in doc 0002.
- **S-AIV-005** — Given any milestone `AI-NN` cited as an owner, when a reviewer opens that milestone's charter in doc 0002, then the charter's goal or deliverable covers the owned term — an owner whose charter does not cover the term is a defect in this artifact, not in doc 0002.

---

## R-AIV-004 — Stable, append-only term identifiers

Every term MUST carry a stable identifier of the form `V-<CAT>-nn`, where `<CAT>` is the category code and `nn` is an ordinal within that category. Identifiers MUST be append-only: a term added later takes the next free ordinal in its category, and existing identifiers MUST NOT be renumbered, reused, or reordered. A superseded term MUST retain its identifier with its definition struck through.

### Scenarios

- **S-AIV-006** — Given the register, when a reviewer reads any category's identifiers in document order, then ordinals are unique within the category AND no ordinal is reused across a superseded and a live row.
- **S-AIV-007** — Given a term discovered missing after this change merges, when it is added, then it receives the next free ordinal in its category AND is introduced by a dated amendment blockquote under its category heading AND no existing identifier changes.

---

## R-AIV-005 — Category completeness against the closing checklist

The register MUST define, at minimum, every term named by AI-01.1's closing checklist items 1 through 4:

- **Request-side** (item 1): role, message, content part and its kinds, tool declaration, tool choice, tool call, tool result, system-instruction segment, cache-boundary marker, generation option, provider escape hatch.
- **Stream-side** (item 2): event, event kind, payload, sequence, stream, terminal event, delta, block index, call ordinal.
- **Metadata** (item 3): finish reason, usage, token-count field, absence versus zero.
- **Failure** (item 4): caller-contract failure, provider/transport failure, pre-stream failure delivery, mid-stream failure delivery.

The register MAY define additional terms beyond this minimum where AI-01's acceptance criterion requires them.

### Scenarios

- **S-AIV-008** — Given AI-01.1's closing-checklist items 1 through 4, when a reviewer checks each named term against the register, then every named term resolves to exactly one row.
- **S-AIV-009** — Given the register contains terms beyond the checklist minimum, when a reviewer asks why each exists, then the artifact names the downstream milestone charter that could not be written without it.

---

## R-AIV-006 — Failure terms are separated by owner and by delivery path

The register MUST separate **caller-contract failure** (AI-04's territory) from **provider/transport failure** (AI-19's territory), stating the line between them, and MUST separately state the **pre-stream versus mid-stream** delivery split. The two separations are orthogonal and MUST be presented as such: the owner split says *whose failure it is*, the delivery split says *how the caller observes it*.

### Scenarios

- **S-AIV-010** — Given the failure category of the register, when a reviewer reads it, then caller-contract failure names AI-04 as owner AND provider/transport failure names AI-19 as owner AND the boundary between them is stated as a rule a reader can apply to a new case.
- **S-AIV-011** — Given the failure category, when a reviewer looks for the delivery split, then pre-stream failure delivery and mid-stream failure delivery are both defined AND both are attributed to AI-19 as the milestone that implements them as one vocabulary over two paths AND the artifact records that AI-02.1 decides the observable shape.
- **S-AIV-012** — Given a borderline failure case — a request the caller could not have known was invalid without contacting the provider — when a reader applies the stated boundary, then the register resolves it to exactly one side, and the artifact shows that resolution as a worked example.

---

## R-AIV-007 — Excluded terms carry a named owner

The register MUST list, at minimum, these nine terms as excluded from Layer 1, each with the layer or component that owns it: agent turn, transcript, session, tool execution, permission, compaction, cost, price, frontend. An excluded term MUST NOT be defined by this artifact; it MUST be named, attributed, and — where the exclusion is easy to misread — accompanied by the Layer 1 concept it is commonly confused with.

### Scenarios

- **S-AIV-013** — Given the exclusion register, when a reviewer checks the nine terms of AI-01.1 checklist item 5, then all nine are present AND each names an owner that is a layer, a port, or the composition root — never a Layer 1 milestone.
- **S-AIV-014** — Given the excluded term `transcript`, when a reviewer reads its row, then the row names Layer 2's harness as owner AND distinguishes it from the Layer 1 term `message`, which AI-05's charter calls "the smallest addressable unit of a transcript".
- **S-AIV-015** — Given any excluded term, when a reviewer looks for a definition of it in the register, then none is present — the row states ownership and the confusable Layer 1 neighbour only.

---

## R-AIV-008 — Provenance and identifier translation

Where a term exists to close a specific recorded defect or gap, its row MUST cite the identifier (`C1`–`C4`, `G1`–`G13`) and the doc 0001 section it derives from. Every milestone identifier cited in the artifact MUST be a **current** doc 0002 identifier. Where a source document uses a retired identifier, the artifact MUST translate it through doc 0002's identifier map and MUST NOT reproduce the retired number.

### Scenarios

- **S-AIV-016** — Given the term `round-trip token`, when a reviewer reads its row, then it cites doc 0001 § 3.2 and § 3.3 row 2 and gap `G12(b)` AND names AI-07 as owner — not the retired identifier AI-45 that doc 0001 § 3.2 still carries.
- **S-AIV-017** — Given the term `terminal error event`, when a reviewer reads its row, then it cites defect `C4` AND names AI-19 as owner — not the retired identifier AI-18 that doc 0001 § 3.1 still carries, which in current numbering is a different milestone.
- **S-AIV-018** — Given every `AI-NN` identifier in `decision.md` that **attributes ownership or provenance**, when a reviewer resolves each against doc 0002's milestone headings, then every one resolves AND none is a retired identifier. One exception, and only one: the artifact MAY name retired identifiers inside its explicit retired-identifier warning, where naming them is the point; such a mention MUST be adjacent to the current identifier that replaces it.

---

## R-AIV-009 — No Go identifiers

No artifact of this change MAY contain a Go type name, field name, method name, interface name, or package identifier belonging to the future Layer 1 surface. Terms MUST be expressed as conceptual noun phrases. Naming the *spelling* of a term is each owning milestone's SDD decision.

### Scenarios

- **S-AIV-019** — Given any artifact of this change, when a reviewer scans for camel-case or Pascal-case single-token names, struct or interface declarations, or field lists, then none is present.
- **S-AIV-020** — Given the term `provider escape hatch`, when a reviewer reads its row, then the definition describes what the concept must be able to carry and what nothing in Layer 1 may do with it, and states no name, shape, or signature.

---

## R-AIV-010 — No production code and no module change

This change MUST NOT create, modify, or delete any file under `backend/`, any `go.mod`, any `go.sum`, any `Makefile`, or any build or container configuration. Its complete output is markdown under its own change directory.

### Scenarios

- **S-AIV-021** — Given the change's diff, when a reviewer lists changed paths, then every path is under `openspec/changes/cachicamas-ai-contract-vocabulary/` AND every changed file has a `.md` suffix.
- **S-AIV-022** — Given the repository before and after the change, when `make build` and `make test` are run in every module, then their results are byte-identical in outcome — the change is provably inert with respect to the build.

---

## R-AIV-011 — Growth is by amendment, never by invention

After this change merges, a Layer 1 term that turns out to be missing, wrong, or double-owned MUST be corrected by an amendment to the register, landed in the same pull request that needs the correction. A downstream milestone MUST NOT introduce a new Layer 1 term in its own SDD without appending it here. An amendment MUST follow doc 0002's convention: a dated blockquote under the touched heading, struck-through text for superseded claims, and no silent edit.

### Scenarios

- **S-AIV-023** — Given a downstream milestone whose SDD needs a term the register does not carry, when that SDD is written, then the same pull request appends the term to the register with the next free ordinal in its category AND a dated amendment blockquote records the addition.
- **S-AIV-024** — Given a superseded definition, when a reviewer reads its row, then the old text is struck through and remains visible AND the replacing text is present AND the row's identifier is unchanged.
- **S-AIV-025** — Given any pull request for milestones AI-02 through AI-40, when a reviewer applies the review checklist, then a Layer 1 noun used in that PR's artifacts either resolves to a register row or is accompanied by the amendment that adds it.

---

## R-AIV-012 — Downstream charters are expressible in these terms

The register MUST be complete enough that every milestone charter from AI-02 through AI-40 can be written using only its terms plus ordinary English. This is AI-01's acceptance criterion, restated normatively so it is checkable rather than aspirational.

### Scenarios

- **S-AIV-026** — Given AI-02's charter as written in doc 0002, when a reviewer maps each domain noun in it to the register, then every one resolves — specifically carrier, ownership, cancellation, buffering, and the failure-delivery split.
- **S-AIV-027** — Given AI-03's charter as written in doc 0002, when a reviewer maps each domain noun in it to the register, then every one resolves — specifically capability, required capability, optional capability, capability discovery, and capability record.
- **S-AIV-028** — Given any charter from AI-04 through AI-40, when a reviewer finds a domain noun that does not resolve, then that finding is recorded as a defect in this artifact and closed by amendment under `R-AIV-011` — not by inventing the term in that milestone's SDD.

---

## R-AIV-013 — The two wording traps are restated verbatim

The artifact MUST restate both wording traps from doc 0002's *Layer boundary* section, quoted verbatim rather than paraphrased, each accompanied by the record that it has already caused one wrong decision.

### Scenarios

- **S-AIV-029** — Given the artifact, when a reviewer compares its trap quotations against doc 0002's *Layer boundary* paragraph, then both sentences match character for character, including their qualifying clauses.
- **S-AIV-030** — Given the first trap, when a reviewer reads the accompanying text, then it states that Layer 1 owns the provider-neutral transport representation of tool declarations and tool calls AND that Layer 1 must not execute tools, resolve tool names, or own application behavior AND the register's tool-related rows are consistent with both halves.
- **S-AIV-031** — Given the second trap, when a reviewer reads the accompanying text, then it states that the claim applies only after adapters exist AND that adding a new vendor still requires a new adapter unless it is genuinely compatible with an existing transport.

---

## Non-functional requirements

### NFR-AIV-A — Reviewability

- The register MUST be readable in one sitting by someone who has read doc 0002's *Layer boundary* section and nothing else.
- Every row MUST fit the pattern `identifier · term · definition · owner · provenance` so a reviewer can scan one column at a time.
- The artifact MUST NOT require the reader to open doc 0001 to understand a definition; citations are for provenance, not for comprehension.

### NFR-AIV-B — Traceability

- Every `C1`–`C4` and every `G1`–`G13` identifier with a Layer 1 obligation MUST appear at least once in the register's provenance column, so the vocabulary is auditable against doc 0002's traceability spine.
- Every citation MUST name a document and a section, not a line number, because line numbers drift.

### NFR-AIV-C — Durability

- The artifact MUST NOT cite any shipped code as evidence. doc 0002's authoring constraint applies: no Layer 1 code exists, so every citation points at a contract document, an ADR, or the architecture reference.

---

## Acceptance criteria

The change is accepted when:

1. `R-AIV-001` through `R-AIV-013` hold, each verified by its scenarios.
2. All six items of AI-01.1's closing checklist are answered in `decision.md`, and `tasks.md` records the verification item by item.
3. The change's diff contains six markdown files and nothing else.
4. No Go identifier appears anywhere in the change.
5. AI-02's and AI-03's charters are demonstrably expressible in the register's terms (`S-AIV-026`, `S-AIV-027`) — the handoff those two milestones' SDDs consume.
