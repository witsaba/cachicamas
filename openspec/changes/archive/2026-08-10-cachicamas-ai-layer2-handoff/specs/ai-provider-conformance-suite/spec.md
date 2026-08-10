# Delta for `ai-provider-conformance-suite`

> **Change**: `cachicamas-ai-layer2-handoff` (AI-40) · **Fork**: proposal **D6**
> **Target**: `openspec/specs/ai-provider-conformance-suite/spec.md`
> **New scenario IDs**: `S-CNF-088` … `S-CNF-090` — `S-CNF-086` and `S-CNF-087` are already taken in the promoted spec (lines 157, 163, 445), so numbering resumes above the observed maximum.
> **Style**: append-only amendment — one stale line corrected, no requirement text, scenario, or
> acceptance item added or removed.
> **Scope note**: this delta corrects documentation drift only. It changes **no** suite behavior, **no**
> adapter, and **no** committed expectation. `ai-first-provider-decision` (`spec.md:125`, "eight
> entries") is deliberately **not** touched: it describes the AI-24-era decision artifact as of its own
> date, and correcting it would rewrite history rather than fix drift.

## Rationale

`R-CNF-017` was amended on 2026-08-07 by AI-35 (`cachicamas-ai-retry-policy`) from "eight entries,
always" to **nine**, and its scenarios `S-CNF-047`, `S-CNF-048` and `S-CNF-076` were restated to match.
Acceptance criterion **item 10** was missed by that amendment and still reads "exactly eight entries"
(`spec.md:343`). It contradicts the requirement it cites. AI-40 publishes the nine-row matrix as Layer
1's exit contract, so shipping the freeze on top of a canonical spec that says "eight" would publish a
contradiction at the handoff gate. Recorded but unfixed by AI-38; fixed here.

## MODIFIED Requirements

### Requirement: Acceptance criterion 10 — the record's totality and verdict rule

Acceptance criterion **item 10** of `ai-provider-conformance-suite` is corrected to state **nine**
entries, matching the requirement it already cites (`R-CNF-017`, amended 2026-08-07 by AI-35) and its
scenarios (`S-CNF-047`, `S-CNF-048`, `S-CNF-076`). No other clause of item 10 changes: standing still
comes from AI-03, the outcome set is still the closed four-value set, any `not exercised` entry is still
inconclusive, and a failed required entry still cannot pass.

(Previously: item 10 read "exactly eight entries", stale since AI-35's nine-entry amendment.)

**Exact replacement — `openspec/specs/ai-provider-conformance-suite/spec.md:343`:**

- **Before**: `10. The record carries exactly eight entries with standing from AI-03 and the four-value outcome set; any `not exercised` entry is **inconclusive** and a failed required entry cannot pass (`R-CNF-017`, `R-CNF-018`).`
- **After**: `10. The record carries exactly **nine** entries with standing from AI-03 and the four-value outcome set; any `not exercised` entry is **inconclusive** and a failed required entry cannot pass (`R-CNF-017`, `R-CNF-018`).`

At archive, promotion MUST also append the following dated amendment note beneath the acceptance-criteria
list, following the AI-35 amendment pattern already used in this spec:

> **Amended 2026-08-10 (AI-40)** by `cachicamas-ai-layer2-handoff`: acceptance item 10 corrected from
> "eight entries" to **nine**. AI-35 amended `R-CNF-017` and its scenarios on 2026-08-07 but missed this
> acceptance line; AI-38 recorded the drift without fixing it. Documentation-only correction — no suite
> behavior, adapter, or committed expectation changed.

#### Scenario: S-CNF-088 — The acceptance criteria agree with the requirement they cite

- **GIVEN** the promoted `ai-provider-conformance-suite` spec after this change is archived
- **WHEN** acceptance item 10 is compared to `R-CNF-017`, `S-CNF-047` and `S-CNF-076`
- **THEN** all four state **nine** entries over `CAP-R-01…05` and `CAP-O-01…04`
- **AND** no occurrence of "eight entries" remains anywhere in this spec except inside a dated historical
  amendment note that explicitly describes the superseded state

#### Scenario: S-CNF-089 — The correction is documentation-only

- **GIVEN** the merged change
- **WHEN** the diff is inspected for edits to `src/agenttest/conformance_*.go`, to any adapter under
  `src/ai/openaicompat/**`, or to the committed capability-record expectation
- **THEN** none exists
- **AND** the conformance suite's own tests pass unmodified under `-race`

#### Scenario: S-CNF-090 — The AI-24-era decision spec is left alone

- **GIVEN** the merged change
- **WHEN** `openspec/specs/ai-first-provider-decision/spec.md` is compared to base
- **THEN** it is byte-identical, including its "eight entries" line, which records that artifact as of
  its own date
