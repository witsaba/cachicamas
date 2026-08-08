# Delta for `ai-provider-client` — AI-36.2 header and configuration redaction

> **Change**: `cachicamas-ai-redaction` · **Milestone**: AI-36 (doc 0002:2172–2177) · **Node**: AI-36.2 · **Wave 5 — Harden**
> **Status**: **delta** — amends [`openspec/specs/ai-provider-client/spec.md`](../../../../../specs/ai-provider-client/spec.md): one `ADDED` requirement appended after `R-APC-014`
> **Format**: RFC 2119 + Given/When/Then, matching the canonical spec's `R-APC-*` / `S-APC-*` grammar
> **Sources**: [proposal.md](../../proposal.md) § 4 D-5/D-6 · [explore.md](../../explore.md) § 1, § 2, § 3

> **Archive-time note (2026-08-08).** Promoted into the canonical spec at close. Relative links in this file were re-resolved from four levels to **five** to match the archived location; nothing else was changed. Item 2's required empirical outcome — **the behavior did not already hold; it was newly established** — is recorded in the canonical spec beneath `S-APC-078`.

---

## Identity

| Field | Value |
| --- | --- |
| **Capability amended** | `ai-provider-client` |
| **Type** | Delta — **one `ADDED` requirement** (`R-APC-015`, scenarios `S-APC-077` … `S-APC-080`); no `MODIFIED`, `REMOVED` or `RENAMED` |
| **Numbering** | Re-verified at spec time across the whole `openspec/` tree, canonical and archived alike: maxima `R-APC-014` / `S-APC-076`. `R-APC-015` and `S-APC-077` are free. |
| **Insertion point** | After `R-APC-014` |

### What is being added

Today the credential is redacted **one layer up** — at the wrapping provider value — and proven there. It has never been proven at the adapter's own configuration value, and never at all at the adapter's configured client value. Item 2 of this requirement is stated as an **invariant of the value**, deliberately independent of whether production code turns out to be needed to satisfy it: the strict-TDD run that establishes the current behavior is the evidence, and its outcome is recorded either way.

Item 3 is stronger than the charter asks. The charter requires redaction to be *opt-out-explicit*; an admission list is *opt-in by construction*, so a header is excluded until someone deliberately admits it.

---

## ADDED Requirements

### R-APC-015 — The credential is redacted by the values that hold it, and header-capturing diagnostics reproduce no header

Redaction MUST be a property of the adapter's own values, not of which value the caller happened to render, and not of a wrapper standing in front of them. Specifically:

1. Rendering the adapter's **configuration value** MUST NOT disclose the credential under any rendering a caller can reach — the plain, string, extended and Go-syntax verbs, and the structured-serialization form.
2. Rendering the adapter's **configured client value** MUST NOT disclose the credential under any of those same renderings, **whether or not** it is wrapped. This is an invariant of the value; whether it already holds or requires added rendering behavior MUST be established empirically and the outcome recorded.
3. Any diagnostic that captures request or response headers MUST reproduce only header names admitted by an explicit admission list. Credential-bearing headers MUST be excluded **by default**; admission MUST be the explicit act, never exclusion.
4. No diagnostic in the module MAY capture the whole header set.
5. Each absence claim above MUST ship a positive control proving the assertion can fail.

#### Scenarios

- **S-APC-077** — Given an adapter configuration value built with a sentinel credential, when it is rendered under the plain, string, extended and Go-syntax verbs and under structured serialization, then the sentinel appears in none of the five outputs; and given an unredacted stand-in carrying the same sentinel, when the same check runs, then it fails, proving the assertion bites.
- **S-APC-078** — Given a **bare, unwrapped** configured client value built with a sentinel credential, when it is rendered under those same five forms, then the sentinel appears in none of them; and given the run that establishes this, when the milestone closes, then its outcome is recorded — either as behavior already holding and pinned against regression, or as behavior newly established.
- **S-APC-079** — Given a response carrying a credential-bearing header alongside a sentinel header value, when the header-capturing diagnostic is rendered, then it reproduces no header name and no header value from that response; and given a rendering deliberately built to reproduce one, when the same check runs, then it fails.
- **S-APC-080** — Given the shipped module, when every diagnostic that reads headers is enumerated, then none captures the whole header set, each reads only through an explicit admission list, and a header newly present on a response is absent from every diagnostic until it is explicitly admitted.

---

## Pins / regressions

| Behavior leaf | Contract pin | Regression assertion |
| --- | --- | --- |
| Configuration value redacted | Credential's own redacting renderings | Five-form loop over a planted sentinel |
| Configured client value redacted | Wrapping provider's landed precedent | Same loop against the bare, unwrapped value |
| Header capture is admission-listed | Landed rate-limit telemetry behavior | No name and no value reproduced |
| No whole-header-set capture | — | Structural enumeration over every diagnostic |
| Absence claims are falsifiable | — | Control per claim |

## Out of scope

| Item | Owner |
| --- | --- |
| The credential-attachment proof — that the credential *reaches* the transport | **AI-25.3** |
| A general-purpose, reusable header-redaction utility | Non-goal — **AI-37** owns one if it needs one |
| Observability attributes and allowlists | **AI-37** |
| Any new module dependency | Non-goal — `go.mod`/`go.sum` byte-identical |
| The names, receivers and doc-comment wording of any rendering behavior added under item 2 | **Design phase** — this spec pins behavior only |
