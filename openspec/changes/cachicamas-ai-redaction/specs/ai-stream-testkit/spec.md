# Delta for `ai-stream-testkit` — AI-36.3.1 failure-output hygiene

> **Change**: `cachicamas-ai-redaction` · **Milestone**: AI-36 (doc 0002:2179–2184) · **Node**: AI-36.3 item 1 · **Wave 5 — Harden**
> **Status**: **delta** — amends [`openspec/specs/ai-stream-testkit/spec.md`](../../../../specs/ai-stream-testkit/spec.md): one `ADDED` requirement appended after `R-STK-013`
> **Format**: RFC 2119 + Given/When/Then, matching the canonical spec's `R-STK-*` / `S-STK-*` grammar
> **Sources**: [proposal.md](../../proposal.md) § 2 WU-4, § 5 · [explore.md](../../explore.md) § 4

---

## Identity

| Field | Value |
| --- | --- |
| **Capability amended** | `ai-stream-testkit` |
| **Type** | Delta — **one `ADDED` requirement** (`R-STK-014`, scenarios `S-STK-047` … `S-STK-049`); no `MODIFIED`, `REMOVED` or `RENAMED` |
| **Numbering** | Re-verified at spec time across the whole `openspec/` tree, canonical and archived alike: maxima `R-STK-013` / `S-STK-046`. `R-STK-014` and `S-STK-047` are free. |
| **Insertion point** | After `R-STK-013` |

### What is being added

`R-STK-004` already requires bounded payload rendering. This delta proves that bound **adversarially**: a sentinel deliberately routed into a diverging payload must not survive into the helper's own failure report. The one place a redaction discipline is easiest to forget is the output produced when an assertion fails, because that output is written for a human and read under time pressure.

---

## ADDED Requirements

### R-STK-014 — The assertion helpers' own failure output is proven sentinel-free, with a control

When an assertion helper reports a divergence, its report is itself a diagnostic surface and MUST obey the module's redaction discipline. Specifically:

1. A divergence report MUST summarize the diverging payload within the established bounded-excerpt rule and MUST NOT reproduce a sentinel credential or a sentinel content body planted in that payload.
2. The bound MUST hold for **every** event kind. No kind MAY render its payload in full, and no sentinel longer than the bound MAY be reconstructible from the report.
3. The report MUST remain useful: it MUST still locate the first divergence by position and name the diverging kind.
4. The proof MUST ship a **positive control** demonstrating the check can fail. An absence claim over failure output that cannot fail does not satisfy this requirement.

#### Scenarios

- **S-STK-047** — Given two recordings that diverge in a payload carrying a planted sentinel credential and a planted sentinel content body, when the assertion helper reports the divergence, then the report names the first diverging position and kind and contains neither sentinel.
- **S-STK-048** — Given a report deliberately constructed to reproduce one of those sentinels, when the same check runs, then it fails and names the vector — proving `S-STK-047`'s absence claim is falsifiable.
- **S-STK-049** — Given a divergence in each event kind in turn, with a sentinel longer than the established bound planted in each payload, when each report is produced, then every kind renders a bounded summary, none renders its payload in full, and the planted sentinel is not reconstructible from any report.

---

## Pins / regressions

| Behavior leaf | Contract pin | Regression assertion |
| --- | --- | --- |
| Report is sentinel-free | `R-STK-004` bounded rendering | Both sentinel vectors planted into a diverging payload |
| Absence claim is falsifiable | — | Report deliberately built to leak is detected |
| Bound holds for every kind | `R-STK-004` | Loop over every event kind with an over-bound sentinel |
| Report stays useful | `R-STK-003` first-divergence rule | Position and kind still named |

## Out of scope

| Item | Owner |
| --- | --- |
| Changing the established excerpt bound itself | **AI-22.2** — reused, not re-litigated |
| The wire-error-body size bound | **AI-32.5** — charter Out-of-scope, verbatim |
| Observability rendering | **AI-37** |
| Any new module dependency | Non-goal — `go.mod`/`go.sum` byte-identical |
