# Spec delta — `agent-package-scaffold` (amended by AG-04)

> **Change**: `cachicamas-agent-event-envelope` · **Milestone**: AG-04.3 (Layer 2, Wave 1)
> **Amends**: `openspec/specs/agent-package-scaffold/spec.md` (AG-03's live spec, last amended 2026-08-12 at the AG-03 archive phase)
> **Promotion discipline**: the `MODIFIED` block below restates the **entire** requirement, including every scenario whose wording did not change, so no landed proof is lost when the archive replaces the requirement in the main spec. No requirement and no scenario not named below changes.

## Why this spec is amended

Design AD-3 (`cachicamas-agent-event-envelope`) adds `L2C-04` — the stream-membership criterion — as a fourth guarded row in `backend/agent/src/agent/doc.go`, landed in the same commit as its byte-identical entry in `doc_contract_guard_test.go`'s `expectedLayer2ContractRows`.

**`R-AGP-002`'s own requirement text and every one of its scenarios (`S-AGP-010`..`S-AGP-015`) already state the mechanism in row-count-agnostic terms** — "rows covering, **at minimum**" (R-AGP-002 prose), "the parsed row set equals the committed table" (S-AGP-012), "a scratch edit that appends a new contract row… FAILS naming the unexpected row" (S-AGP-014). None hardcodes "three" anywhere. No requirement or scenario wording therefore needs to change to remain true of a four-row table.

This delta exists to make the change to the **committed table's own content** — which `R-AGP-002`'s own promotion-tracked history requires recording — visible in the capability's spec, per the header note's own instruction: *"A later milestone that needs to change one of these invariants… amends this file, in the same pull request, under its own ADR gate."* `L2C-04` is exactly such a change: a new row added to the table `S-AGP-012`..`S-AGP-015` describe, not a change to the mechanism itself.

**Re-verified during AG-04.3's apply**: the "same commit" ordering constraint the guard enforces was re-proven against the new four-row baseline — a scratch removal of `L2C-04`'s table entry (leaving the row itself in `doc.go`) reproduces exactly `S-AGP-013`/`S-AGP-014`'s own bite shape: `TestLayer2DocContract_MatchesTheCommittedTable` fails naming a 4-of-3 row-count mismatch. Recorded in `cachicamas-agent-event-envelope/apply-progress.md`.

---

## MODIFIED Requirements

### R-AGP-002 — The layer contract is a machine-checked doc-row table

*(MODIFIED 2026-08-12 by AG-04 — restates the entire requirement, including every unchanged scenario, so no landed proof is lost at archive. Wording is byte-identical to the AG-03-shipped requirement below; the change this amendment records is to the **committed expectation table's content** the guard compares against — `doc.go`/`doc_contract_guard_test.go` gained a fourth row, `L2C-04`, the stream-membership criterion (design AD-3) — not to any requirement or scenario text, none of which names a row count.)*

The package documentation MUST state the Layer 2 contract in machine-parseable rows covering, at minimum: the permitted-imports rule ([ADR 0005 § D1](../../../../docs/adr/0005-promote-agent-stack-to-own-module.md#d1--dependency-rule-v2) row 2), the no-I/O-of-its-own rule, and the event stream as the only upward contract.

A guard in the same package MUST read `backend/agent/src/agent/doc.go`'s **raw bytes**, match the pinned row grammar with a line pattern, and compare the parsed rows **entry-by-entry and in order** against a **closed, committed expectation table declared in the test source**. The comparison MUST be equality over the full row set, not a subset or containment check: a row present in the file but absent from the table MUST fail, and a table entry absent from the file MUST fail.

This mechanism is a recorded **substitution** for AG-03.1's cited "doc-guard byte-suffix convention", which exists nowhere in doc 0002 or doc 0003 as a worked mechanism. The substituted precedent is `backend/agent/src/ai/openaicompat/openrouter/conformance/doc_matrix_guard_test.go` (AI-40.2). The substitution MUST be stated in the guard's own source with its reason. Any later milestone that appends a guarded paragraph to this doc comment MUST amend the expectation table in the **same** pull request; the guard is what makes omitting that amendment a failing test rather than drift.

#### Scenarios

- **S-AGP-010** — Given `backend/agent/src/agent/doc.go`, when its documentation comment is read, then it carries rows stating the permitted-imports rule citing ADR 0005 § D1, the no-I/O rule, and the event stream as the only upward contract.
- **S-AGP-011** — Given the doc-row guard's source, when its file access is read, then it resolves `doc.go` from the test's own location rather than an assumed working directory, reads its raw bytes, and parses rows with a pinned line pattern — not by importing the package or reading `go/doc` output.
- **S-AGP-012** — Given the repository with no divergence, when `cd backend/agent && go test ./src/agent/...` runs, then the doc-row guard passes and the parsed row set equals the committed table in the same order. *(As of AG-04: the committed table has four rows, `L2C-01`..`L2C-04`; the scenario's own wording is unaffected.)*
- **S-AGP-013** — **(bite)** Given a scratch edit to `doc.go` that changes one contract row's text without amending the expectation table, when the doc-row guard runs, then it FAILS and its message names the divergent row and both the expected and the found text. The failing output is recorded; the scratch edit is then reverted.
- **S-AGP-014** — **(bite)** Given a scratch edit that appends a new contract row to `doc.go` without adding it to the expectation table, when the doc-row guard runs, then it FAILS naming the unexpected row — proving the comparison is closed and not a containment check. Recorded, then reverted. *(Re-proven at AG-04 against the four-row baseline by removing `L2C-04`'s own table entry — the same shape of bite, one row later.)*
- **S-AGP-015** — Given the doc-row guard's source, when its header comment is read, then it states that this mechanism substitutes for doc 0003's cited "byte-suffix convention", names `doc_matrix_guard_test.go` as the precedent it copies, and states that a later appended paragraph must amend the table in the same pull request.

---

## Unchanged

`R-AGP-001`, `R-AGP-003`, `R-AGP-004`, `R-AGP-005`, `R-AGP-006`, all non-functional requirements, and every scenario not restated above are unaffected by this change. AG-04 imports only `backend/agent/src/ai` and the standard library — already admitted by `R-AGP-003`'s allowlist — so no forward-guard or ambient-authority scenario changes.

At archive time the spec's own header note MUST gain an "Amended 2026-08-12 (AG-04)" entry recording that the committed doc-row table grew to four rows, in the same shape as the existing AG-03-archive-phase note, and the acceptance-criteria section MUST record that `R-AGP-002`'s scenarios were re-verified against the four-row table at AG-04's close.
