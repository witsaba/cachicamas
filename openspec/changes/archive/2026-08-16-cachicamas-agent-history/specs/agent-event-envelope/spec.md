# Delta for `agent-event-envelope` — AG-12 adds `L2C-07`, so `R-AEV-014`'s row count is amended

> **Change**: `cachicamas-agent-history` · **AG-12** (Layer 2, Wave 3, milestone 12 of 24), `0003:1227-1292`
> **Modifies**: `agent-event-envelope` ([`../../../../specs/agent-event-envelope/spec.md`](../../../../specs/agent-event-envelope/spec.md)) — `R-AEV-014` only.
> **Format**: Given/When/Then + RFC 2119 per `openspec/config.yaml` `rules.specs`. The archive step REPLACES the requirement block in the main spec with the MODIFIED block below; full-block preservation is mandatory.
> **Why this delta is mandatory**: `S-AEV-122` (`spec.md:217`) asserts that `expectedLayer2ContractRows` "contains 6 rows (`L2C-01`..`L2C-06`)". AG-12 lands `L2C-07` (`R-HIS-009`), which makes that assertion **false as written**. Leaving it is exactly the owning-spec-omission staleness this repository has been bitten by, so the amendment is mandatory, not optional. The new row's own content is specified by `agent-history` (`R-HIS-009`); this delta only keeps `R-AEV-014`'s claim about the table true.

## MODIFIED Requirements

### Requirement: L2C-06 doc-contract row references the four protocol families — `R-AEV-014`

The doc-contract guard's `expectedLayer2ContractRows` table MUST carry an `L2C-06` row added in the same commit as the AG-06 kinds and the `L2C-06` row in `doc.go`. The row's text states that the four protocol families (permission, cost, delegation, compaction) are constructible on the event stream and that the per-family semantics belong in `doc.go` prose, not in the guarded row, per `R-AGP-002`'s closed-amendment rule. The guardian of this amendment is the same `doc_contract_guard_test.go` that guards `L2C-01`..`L2C-05`.

This requirement owns `L2C-06`'s presence, position and text — **not** the table's total size. The table is append-only across milestones under `R-AGP-002`: a later milestone that declares a new package-wide guarantee appends its own row, and no scenario of this requirement may be written so that such an append falsifies it. As of AG-12 (`cachicamas-agent-history`) the committed table holds **7 rows (`L2C-01`..`L2C-07`)**, `L2C-07` being history's single-validated-commit-path guarantee, owned by `R-HIS-009` in [`../agent-history/spec.md`](../agent-history/spec.md).

(Previously: `S-AEV-122` asserted the table "contains 6 rows (`L2C-01`..`L2C-06`)" as a closed total, which AG-12's `L2C-07` makes false; the requirement did not distinguish owning `L2C-06` from owning the table's size.)

#### Scenarios

- **S-AEV-122** — Given the `expectedLayer2ContractRows` table in `doc_contract_guard_test.go`, when it is read, then the `L2C-06` row is present, in order, immediately after `L2C-05`, with row text referencing the four protocol families; the table holds 7 rows as of AG-12 (`L2C-01`..`L2C-07`), and any later appended row does not disturb `L2C-06`'s presence, position or text.
- **S-AEV-123** — Given a scratch edit that appends an `L2C-06` row to `doc.go` without adding its entry to `expectedLayer2ContractRows`, when the doc-contract guard runs, then it FAILS naming the unexpected row — the closed-amendment rule is observed, not bypassed. RED-recordable.
