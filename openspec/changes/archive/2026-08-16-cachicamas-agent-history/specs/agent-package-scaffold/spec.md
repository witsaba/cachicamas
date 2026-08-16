# Delta for `agent-package-scaffold` — AG-12 back-annotates the doc-row baseline to seven rows

> **Change**: `cachicamas-agent-history` · **AG-12** (Layer 2, Wave 3, milestone 12 of 24), `0003:1227-1292`
> **Modifies**: `agent-package-scaffold` ([`../../../../specs/agent-package-scaffold/spec.md`](../../../../specs/agent-package-scaffold/spec.md)) — `R-AGP-002` only.
> **Format**: Given/When/Then + RFC 2119 per `openspec/config.yaml` `rules.specs`. The archive step REPLACES the requirement block in the main spec with the MODIFIED block below; full-block preservation is mandatory.
> **Why this delta**: `S-AGP-012` and `S-AGP-014` (`spec.md:53,55`) carry "as of AG-04 … four rows" parentheticals that were already two rows stale before AG-12 and become three rows stale with `L2C-07`. The normative wording of `R-AGP-002` is unaffected — the guard is an ordered equality over whatever the committed table holds — so this is a **back-annotation**, recording AG-12's row so the parentheticals stop drifting. `L2C-07`'s own content is owned by `R-HIS-009` in [`../agent-history/spec.md`](../agent-history/spec.md).

## MODIFIED Requirements

### Requirement: The layer contract is a machine-checked doc-row table — `R-AGP-002`

The package documentation MUST state the Layer 2 contract in machine-parseable rows covering, at minimum: the permitted-imports rule ([ADR 0005 § D1](../../../../../docs/adr/0005-promote-agent-stack-to-own-module.md#d1--dependency-rule-v2) row 2), the no-I/O-of-its-own rule, and the event stream as the only upward contract.

A guard in the same package MUST read `backend/agent/src/agent/doc.go`'s **raw bytes**, match the pinned row grammar with a line pattern, and compare the parsed rows **entry-by-entry and in order** against a **closed, committed expectation table declared in the test source**. The comparison MUST be equality over the full row set, not a subset or containment check: a row present in the file but absent from the table MUST fail, and a table entry absent from the file MUST fail.

This mechanism is a recorded **substitution** for AG-03.1's cited "doc-guard byte-suffix convention", which exists nowhere in doc 0002 or doc 0003 as a worked mechanism. The substituted precedent is `backend/agent/src/ai/openaicompat/openrouter/conformance/doc_matrix_guard_test.go` (AI-40.2). The substitution MUST be stated in the guard's own source with its reason. Any later milestone that appends a guarded paragraph to this doc comment MUST amend the expectation table in the **same** pull request; the guard is what makes omitting that amendment a failing test rather than drift.

The requirement is stated over **whatever the committed table holds**; it never fixes a row count. The row baseline has grown `L2C-01`..`L2C-03` (AG-03) → `L2C-04` (AG-04) → `L2C-05` (AG-05) → `L2C-06` (AG-06) → `L2C-07` (AG-12, history's single validated commit path), each row landing in the same pull request as its expectation-table entry.

(Previously: identical normative text; `S-AGP-012` and `S-AGP-014` carried "as of AG-04 … four rows" parentheticals that no longer described the committed baseline, and the requirement did not state that the count is deliberately unfixed.)

#### Scenarios

- **S-AGP-010** — Given `backend/agent/src/agent/doc.go`, when its documentation comment is read, then it carries rows stating the permitted-imports rule citing ADR 0005 § D1, the no-I/O rule, and the event stream as the only upward contract.
- **S-AGP-011** — Given the doc-row guard's source, when its file access is read, then it resolves `doc.go` from the test's own location rather than an assumed working directory, reads its raw bytes, and parses rows with a pinned line pattern — not by importing the package or reading `go/doc` output.
- **S-AGP-012** — Given the repository with no divergence, when `cd backend/agent && go test ./src/agent/...` runs, then the doc-row guard passes and the parsed row set equals the committed table in the same order. *(As of AG-12: the committed table has seven rows, `L2C-01`..`L2C-07`; the scenario's own wording is unaffected.)*
- **S-AGP-013** — **(bite)** Given a scratch edit to `doc.go` that changes one contract row's text without amending the expectation table, when the doc-row guard runs, then it FAILS and its message names the divergent row and both the expected and the found text. The failing output is recorded; the scratch edit is then reverted.
- **S-AGP-014** — **(bite)** Given a scratch edit that appends a new contract row to `doc.go` without adding it to the expectation table, when the doc-row guard runs, then it FAILS naming the unexpected row — proving the comparison is closed and not a containment check. Recorded, then reverted. *(Re-proven at AG-12 against the seven-row baseline by `S-HIS-081`, which withholds `L2C-07`'s own table entry — the same shape of bite, three rows later.)*
- **S-AGP-015** — Given the doc-row guard's source, when its header comment is read, then it states that this mechanism substitutes for doc 0003's cited "byte-suffix convention", names `doc_matrix_guard_test.go` as the precedent it copies, and states that a later appended paragraph must amend the table in the same pull request.
