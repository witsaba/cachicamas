# Delta for `ai-request-translation` — AI-36.3.2 widened credential scan

> **Change**: `cachicamas-ai-redaction` · **Milestone**: AI-36 (doc 0002:2179–2184) · **Node**: AI-36.3 item 2 · **Wave 5 — Harden**
> **Status**: **delta** — amends [`openspec/specs/ai-request-translation/spec.md`](../../../../../specs/ai-request-translation/spec.md): one `ADDED` requirement appended after `R-ART-021`, amending `R-ART-004` **append-only**
> **Format**: RFC 2119 + Given/When/Then, matching the canonical spec's `R-ART-*` / `S-ART-*` grammar
> **Sources**: [proposal.md](../../proposal.md) § 4 D-3 · [explore.md](../../explore.md) § 5

> **Archive-time note (2026-08-08).** Promoted into the canonical spec at close. Relative links in this file were re-resolved from four levels to **five** to match the archived location; nothing else was changed. `R-ART-004` was confirmed **not edited** in the canonical spec — its text and `S-ART-013` … `S-ART-017` stand verbatim, and all five were re-run green under the widened rule.

---

## Identity

| Field | Value |
| --- | --- |
| **Capability amended** | `ai-request-translation` |
| **Type** | Delta — **one `ADDED` requirement** (`R-ART-022`, scenarios `S-ART-090` … `S-ART-094`). `R-ART-004` is **not** edited: it is amended append-only by the addition, and its scenarios `S-ART-013` … `S-ART-017` stand unmodified. |
| **Numbering** | Re-verified at spec time across the whole `openspec/` tree, canonical and archived alike: maxima `R-ART-021` / `S-ART-089`. `R-ART-022` and `S-ART-090` are free. |
| **Insertion point** | After `R-ART-021` |

### What is being added

`R-ART-004` scans one directory and excludes a whole **category** of files — every package-internal test file — because some of them plant credential-shaped values on purpose. That exclusion is a disclosed gap: an *accidental* real credential in any of those files is invisible. This delta replaces a silent category with a declared, enumerated, asserted list. Widening the scan therefore **increases** coverage rather than relocating the hole, and adding a new exemption becomes a diff a reviewer must approve.

---

## ADDED Requirements

### R-ART-022 — The scan covers the whole adapter tree, and every unswept file is declared and enumerated

The credential scan's surface MUST extend over the whole adapter tree, and no file MAY be unswept without a declaration a reviewer can see. Specifically:

1. The scan MUST cover, recursively, every test file and every fixture file under the adapter tree — including its nested provider, live-smoke, conformance and fixture directories — not a single directory.
2. A file MAY be exempt from the scan **only** if it carries an explicit in-file declaration that it plants a credential-shaped value deliberately. No exemption by package kind, directory, naming convention or file extension is permitted.
3. The set of declaration-bearing files MUST match a **committed, enumerated list**, and that match MUST itself be asserted. Adding a declaration to a file absent from the list, or removing one from a listed file, MUST fail until the list is updated.
4. The scan's own patterns, the declaration text and the committed list MUST be assembled at run time rather than appearing as contiguous literals, so the scan does not report its own sources.
5. No scan failure output MAY reproduce the value it matched; it MUST name the file and the vector only.
6. `R-ART-004`'s existing obligations and its scenarios `S-ART-013` … `S-ART-017` MUST continue to hold under the widened rule.

#### Scenarios

- **S-ART-090** — Given a credential-shaped value planted in a nested directory of the adapter tree that the previous scan never read, when the scan runs, then it reports that file — proving the widening, and proving the scan can fail.
- **S-ART-091** — Given a file carrying the explicit deliberate-plant declaration, when the scan runs, then that file is not reported; and given the identical content with the declaration removed, when the scan runs again, then it is reported.
- **S-ART-092** — Given the shipped tree, when the set of declaration-bearing files is compared with the committed enumerated list, then the two match exactly; and given a declaration added to a file absent from that list, when the comparison runs, then it fails and names the offending file.
- **S-ART-093** — Given a scan run that reports a match, when its output is inspected, then it names the file and the vector and reproduces none of the matched value; and given the scan's own sources, the declaration text and the committed list, when the scan runs over them, then none is reported.
- **S-ART-094** — Given the widened scan in place, when `R-ART-004`'s existing scenarios `S-ART-013` … `S-ART-017` are re-run, then all five still pass, and the obligation previously satisfied by excluding package-internal test files is now satisfied by the declaration rule instead.

---

## Pins / regressions

| Behavior leaf | Contract pin | Regression assertion |
| --- | --- | --- |
| Recursive coverage | `R-ART-004` | Planted value in a previously-unread directory is reported |
| Declaration-only exemption | `R-ART-004`'s disclosed gap | Same content with and without the declaration |
| Allowlist is itself asserted | — | Declaration-bearing set compared to the committed list |
| Runtime-assembled patterns | Landed scan discipline | Scan does not report its own sources |
| Prior scenarios preserved | `S-ART-013` … `S-ART-017` | All five re-run green |

## Out of scope

| Item | Owner |
| --- | --- |
| Migrating existing package-internal test files to the external-package convention | Non-goal — this delta makes the exemption reviewable, it does not restructure test packages |
| The wire-error-body size bound the deliberate plants exercise | **AI-32.5** — charter Out-of-scope, verbatim |
| The inventory of which files currently carry a deliberate plant | **Design phase** — a named design deliverable; `sdd-apply` MUST NOT discover it at apply time |
| Any new module dependency | Non-goal — `go.mod`/`go.sum` byte-identical |
