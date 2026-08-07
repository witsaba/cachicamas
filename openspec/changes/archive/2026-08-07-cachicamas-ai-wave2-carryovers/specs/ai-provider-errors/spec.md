# Delta for `ai-provider-errors` — AI-41.2 discharges carryover `W2`

> **Change**: `cachicamas-ai-wave2-carryovers` · **Milestone**: AI-41 (doc 0002:2233–2257) · **Node**: AI-41.2 · **Wave 5 — Harden**
> **Carryover discharged**: `W2` — recorded at [`openspec/specs/ai-stream-testkit/spec.md:39`](../../../../specs/ai-stream-testkit/spec.md)
> **Status**: **delta** — amends [`openspec/specs/ai-provider-errors/spec.md`](../../../../specs/ai-provider-errors/spec.md): one `ADDED` requirement appended after `R-AIP-015` and before the `NFR-AIP-*` section
> **Format**: RFC 2119 + Given/When/Then, matching the canonical spec's `R-AIP-*` / `S-AIP-*` grammar
> **Sources**: [proposal.md](../../proposal.md) · [explore.md](../../explore.md) · [doc 0002 § 2233–2257](../../../../../docs/architecture/milestones/0002-cachicamas-ai-layer-1-task-graph.md)

---

## Identity

| Field | Value |
| --- | --- |
| **Change name** | `cachicamas-ai-wave2-carryovers` |
| **Milestone node** | AI-41.2 — "redaction is a property of the type, not of the formatting verb" |
| **Capability amended** | `ai-provider-errors` (per proposal § 8 Capabilities) |
| **Type** | Delta — **one `ADDED` requirement** (`R-AIP-016`, scenarios `S-AIP-056`, `S-AIP-057`); **no `MODIFIED`, no `REMOVED`, no `RENAMED`** |
| **Numbering** | Verified against the canonical spec at spec time: `R-AIP-015` and `S-AIP-055` are the current maxima; `NFR-AIP-A` … `NFR-AIP-E` are unchanged |
| **New NFR** | **None.** Totality for the new rendering is `NFR-AIP-B` restated in place, not a sixth NFR — see "Why no `NFR-AIP-F`" below |
| **Insertion point** | After `R-AIP-015` (end of § AI-19.5), before § "Non-functional requirements" |
| **Test runner** | `cd backend/agent && make test` (`go test -race -v ./...`) |
| **Module** | `backend/agent` — **layered** (not hexagonal), per ADR 0005 § D1 |

### What is being added

One behavior-only requirement closing the last verb through which the provider-failure payload can leak. The payload's textual rendering is already redacted and already proven against a planted secret. Because the payload is itself an error value, the plain, string and quoted verbs all reach that redacted rendering. The **Go-syntax verb** is the exception: with no Go-syntax rendering of its own, the formatting machinery falls back to reflecting over the payload's internal state, including the wrapped underlying cause, which may carry raw provider response text or credential-adjacent material.

The defect is therefore not "one verb prints too much". It is that **redaction is currently a property of which verb the caller reached for**, rather than a property of the payload — the exact posture this module's event envelope already rejects in its own contract text. `R-AIP-016` makes redaction total over the verb surface.

### Why no `NFR-AIP-F`

`NFR-AIP-B` (totality) already binds every exported entry point of this contract against panicking on extreme inputs, explicitly including a nil wrapped cause. The new rendering is such an entry point, so it inherits that obligation the moment it exists; a sixth non-functional requirement would restate an obligation that already applies and would imply the existing five did not cover new surface. `R-AIP-016` therefore cites `NFR-AIP-B` and `S-AIP-057` proves it for the new rendering.

---

## Verb-total redaction (AI-41, dated 2026-08-07)

> **Amended 2026-08-07 (AI-41)** by `cachicamas-ai-wave2-carryovers` (AI-41.2, Wave 5 — Harden). One behavior-only requirement added: **`R-AIP-016`** — the provider-failure payload's redaction is a property of the payload, not of the caller's formatting verb; the Go-syntax rendering agrees with the already-redacted textual rendering and never reaches the wrapped cause. This discharges carryover `W2`. No existing requirement is edited: `R-AIP-013` … `R-AIP-015` describe the two delivery paths and their accessor parity, which this requirement does not touch, and `NFR-AIP-A` … `NFR-AIP-E` stand unmodified.

### ADDED Requirements

#### R-AIP-016 — Redaction is a property of the failure payload, not of the caller's formatting verb

The provider-failure payload MUST render redacted output under **every** formatting verb a caller can reach, including the Go-syntax verb. Specifically:

1. A caller requesting a Go-syntax representation MUST NOT receive a representation derived by reflection over the payload's internal state, and MUST NOT through it reach the **wrapped underlying cause** — the one field that may carry raw provider response text, provider-body fragments, or credential-adjacent material.
2. The Go-syntax rendering MUST agree with the payload's already-redacted textual rendering, so redaction cannot drift between the two as either evolves.
3. No content excluded from the textual rendering MAY become reachable through any other verb. Redaction MUST NOT be reachable-or-not depending on the caller's formatting choice.
4. The obligation MUST hold **totally**: an absent failure payload MUST format under every one of those verbs without panicking, returning the contract's defined absent-failure rendering (`NFR-AIP-B`).

This requirement adds no accessor and no second failure type; it constrains rendering only, and therefore leaves `R-AIP-013`'s single-type rule and `R-AIP-015`'s accessor parity untouched.

#### Scenarios

- **S-AIP-056** — Given a failure payload wrapping a cause that carries a planted sentinel string standing in for raw provider text, when the payload is formatted with each of the plain, string, extended and Go-syntax verbs, then the planted sentinel appears in **none** of the four outputs, and no other fragment of the wrapped cause appears in the Go-syntax output.
- **S-AIP-057** — Given that same failure payload, when its Go-syntax rendering is compared byte-for-byte with its redacted textual rendering, then the two are identical; and given an **absent** failure payload, when it is formatted under each of those same four verbs, then each yields the contract's defined absent-failure rendering and none panics (`NFR-AIP-B`, `S-AIP-052`).

---

## Pins / regressions

| AI-41 node | Behavior leaf | Contract pin | Regression assertion |
| --- | --- | --- | --- |
| AI-41.2 | Planted cause sentinel absent from every verb | `NFR-AIP-B`, existing planted-secret proof of the textual rendering | Four-verb loop over a planted canary; no output contains it |
| AI-41.2 | Go-syntax agrees with the redacted textual rendering | `R-AIP-013` (one concrete type) | Byte-for-byte equality of the two renderings |
| AI-41.2 | Absent payload formats totally | `NFR-AIP-B`, `S-AIP-052` | Absent payload under all four verbs: defined rendering, no panic |
| AI-41.2 | No accessor or type added | `R-AIP-013`, `R-AIP-015` | Accessor parity across both delivery paths unchanged |
| AI-41.2 | Dependency purity | `NFR-AIP-A`, `S-AIP-051` | `backend/agent/go.mod` still declares zero requires; both import guards pass |

---

## Out of scope

| Item | Owner |
| --- | --- |
| Carryover `W1` (the emission boundary's payload-validation rule) | **AI-41.1** — `openspec/changes/cachicamas-ai-wave2-carryovers/specs/ai-event-envelope/spec.md` |
| Any other redaction behavior already proven — the textual rendering, the bounded raw provider label, the event and content-part renderings | **Already proven** — charter Out-of-scope, verbatim |
| The adversarial redaction sweep across the whole package surface | **AI-36** — consumes this result rather than repeating it |
| A textual-rendering method on the failure payload | **Rejected** (proposal Flag 1) — the payload is an error value, so its error rendering already serves those verbs; a second one would be unreachable |
| An equivalent obligation on the validation-failure value | **Not applicable** — that value is structurally safe under reflection (fixed sentinels and positions only); the asymmetry is why `W2` targets the provider-failure payload alone |
| The rendering method's Go identifier, receiver shape and doc-comment wording | **Design phase** — this spec pins behavior only; design owns identifiers |

---

## Acceptance criteria

1. `R-AIP-016` holds, verified by `S-AIP-056` and `S-AIP-057`.
2. A sentinel planted in the wrapped cause is absent from all four verb outputs; the assertion loops over the verbs rather than checking one.
3. The Go-syntax rendering and the redacted textual rendering are identical, so a later change to one cannot silently un-redact the other.
4. An absent failure payload formats totally under all four verbs (`NFR-AIP-B`); no new non-functional requirement is introduced.
5. No accessor, sentinel, category or second failure type is added; `R-AIP-013` … `R-AIP-015` remain unmodified.
6. No Go identifier — type, field, method, package path or signature — appears in this spec.
7. `NFR-AIP-A` holds: `backend/agent/go.mod` still declares zero requires and both import guards pass.
8. **At archive**: `R-AIP-016` is promoted into the canonical spec after `R-AIP-015` and before § "Non-functional requirements".
9. `cd backend/agent && make test` is green under `-race`; `make lint` is clean.
