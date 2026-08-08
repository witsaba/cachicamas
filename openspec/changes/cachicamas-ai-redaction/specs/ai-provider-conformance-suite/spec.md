# Delta for `ai-provider-conformance-suite` — AI-36.1 sentinel sweep

> **Change**: `cachicamas-ai-redaction` · **Milestone**: AI-36 (doc 0002:2150–2184) · **Node**: AI-36.1 · **Wave 5 — Harden**
> **Status**: **delta** — amends [`openspec/specs/ai-provider-conformance-suite/spec.md`](../../../../specs/ai-provider-conformance-suite/spec.md): one `ADDED` requirement appended after `R-CNF-019`
> **Format**: RFC 2119 + Given/When/Then, matching the canonical spec's `R-CNF-*` / `S-CNF-*` grammar
> **Sources**: [proposal.md](../../proposal.md) § 4 D-2/D-4, § 8 · [explore.md](../../explore.md) § 4, § 5

---

## Identity

| Field | Value |
| --- | --- |
| **Capability amended** | `ai-provider-conformance-suite` |
| **Type** | Delta — **one `ADDED` requirement** (`R-CNF-027`, scenarios `S-CNF-077` … `S-CNF-081`); no `MODIFIED`, `REMOVED` or `RENAMED` |
| **Numbering** | Re-verified at spec time across the **whole** `openspec/` tree. Canonical maxima are `R-CNF-019` / `S-CNF-076`, **but** `R-CNF-020` … `R-CNF-026` are already defined by two archived amendments (`2026-08-05-cachicamas-ai-conformance-lifecycle-amendment` → 020–024; `2026-08-05-cachicamas-ai-conformance-tool-amendment` → 025–026) that were never promoted into the canonical file. `R-CNF-027` is therefore the first genuinely free identifier — **not** `R-CNF-020` as the proposal § 8 estimated. |
| **Insertion point** | After `R-CNF-019`, before § "Non-functional requirements" |

### What is being added

`R-CNF-013` already proves a planted sentinel absent from one registered case. This delta makes that proof a **property any adapter and any future failure path inherits**, and makes the absence claim itself falsifiable. The milestone's characteristic failure mode is a sweep that passes because the sentinel never reached the surface under test; every absence claim here is therefore paired with a control.

---

## ADDED Requirements

### R-CNF-027 — Redaction is proven by one inheritable sweep whose absence claim is falsifiable

The suite MUST prove sentinel absence through a **single** sweep capability that every consumer reaches, rather than through per-consumer re-implementations. Specifically:

1. A distinctive sentinel **credential**, configured into the adapter under test, MUST appear in no error rendering, no wrapped cause reachable through a rendering, no verbose or Go-syntax rendering, and no event metadatum, across **every** failure path the suite can trigger.
2. A distinctive sentinel **content body**, sent through the adapter, MUST be equally absent from every error rendering, every log field and every event metadatum.
3. Exactly **one** sweep implementation MUST exist in the module. The suite's redaction case and the live-smoke sweep MUST both reach it, and both MUST keep their already-published names, so consumers bound to them are not broken.
4. A newly registered failure path MUST inherit the sweep without a second sweep being written.
5. Every absence claim MUST ship a **positive control** proving the sweep detects a planted leak. An absence assertion that cannot fail does not satisfy this requirement.
6. No sweep report MAY reproduce the bytes it matched. A report MUST name the vector, the file, the event position or the verb only.

#### Scenarios

- **S-CNF-077** — Given an adapter configured with a sentinel credential, when every failure path the suite can trigger is exercised and each resulting error rendering, wrapped cause, verbose rendering, Go-syntax rendering and event metadatum is swept, then the sentinel credential appears in none of them.
- **S-CNF-078** — Given a request whose content carries a sentinel body, when the same failure paths are exercised and every error rendering, log field and event metadatum is swept, then the sentinel body appears in none of them.
- **S-CNF-079** — Given a deliberately planted leak of each sentinel in turn, when the sweep runs, then it reports a failure for each, and given that report, when it is inspected, then it names the vector and position and contains neither sentinel.
- **S-CNF-080** — Given the shipped module, when its sweep implementations are enumerated, then exactly one exists; and when the suite's redaction case and the live-smoke sweep are each run, then both reach that one implementation, both keep their previously published names, and the tests bound to those names stay green.
- **S-CNF-081** — Given a provider that echoes the caller's authorization value and the caller's request body back inside a non-streaming response, when the resulting refusal is rendered and swept, then the sentinel credential is absent; and given any disclosure attributable solely to the provider replaying the caller's own content, when the milestone closes, then that residual is recorded in writing as a named residual rather than silently absent.

---

## Pins / regressions

| Behavior leaf | Contract pin | Regression assertion |
| --- | --- | --- |
| Credential sentinel absent everywhere | `R-CNF-013` | Sweep over every triggerable failure path |
| Content sentinel absent everywhere | `R-CNF-013` | Sweep over every rendering and event metadatum |
| One implementation, two consumers | Live-smoke sweep's published names | Consumers' landed tests stay green, names unchanged |
| Absence claim is falsifiable | `S-CNF-079` | Planted leak detected for each vector |
| No sweep reprints a match | `R-CNF-013` | Report contains vector and position only |

## Out of scope

| Item | Owner |
| --- | --- |
| The credential-attachment proof | **AI-25.3** — charter Out-of-scope, verbatim |
| The wire-error-body size bound | **AI-32.5** — charter Out-of-scope, verbatim |
| Observability boundary, spans, attribute allowlists | **AI-37** |
| A general-purpose header-redaction utility | Non-goal — zero consumers today |
| Any new module dependency | Non-goal — `go.mod`/`go.sum` stay byte-identical |
