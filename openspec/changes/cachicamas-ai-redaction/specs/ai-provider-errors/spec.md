# Delta for `ai-provider-errors` — AI-36 discharges the AI-41-inherited follow-up

> **Change**: `cachicamas-ai-redaction` · **Milestone**: AI-36 · **Inherited from**: AI-41 close, doc 0002 line 22 · **Wave 5 — Harden**
> **Status**: **delta** — amends [`openspec/specs/ai-provider-errors/spec.md`](../../../../specs/ai-provider-errors/spec.md): one `ADDED` requirement appended after `R-AIP-016`, before § "Non-functional requirements"
> **Format**: RFC 2119 + Given/When/Then, matching the canonical spec's `R-AIP-*` / `S-AIP-*` grammar
> **Sources**: [proposal.md](../../proposal.md) § 4 D-1 · [explore.md](../../explore.md) § 2

---

## Identity

| Field | Value |
| --- | --- |
| **Capability amended** | `ai-provider-errors` |
| **Type** | Delta — **one `ADDED` requirement** (`R-AIP-017`, scenarios `S-AIP-058` … `S-AIP-061`); `R-AIP-016` is **not** edited and `NFR-AIP-A` … `NFR-AIP-E` stand unmodified |
| **Numbering** | Re-verified at spec time across the whole `openspec/` tree, canonical and archived alike: maxima `R-AIP-016` (landed by AI-41) / `S-AIP-057`. `R-AIP-017` and `S-AIP-058` are free. |
| **Insertion point** | After `R-AIP-016`, before § "Non-functional requirements" |

### What is being added

`R-AIP-016` made redaction total over the **verb** surface for the payload as callers receive it. AI-41's close recorded, explicitly and in writing, that the guarantee is **scoped**: a copied value form of the payload is neither an error value nor a Go-syntax renderer, so a verbose rendering of it walks its internal state by reflection, wrapped cause included.

The chosen discharge is **not** to widen the guarantee by adding a value-shaped rendering. That rendering would also enter the pointer form's rendering set and would re-open `NFR-AIP-B`'s landed total nil-safety, to cover a shape no constructor produces. This delta instead states the scope, proves the leaking shape unreachable, and guards that proof.

### The falsifiability condition — load-bearing, not commentary

A naive proof here goes green **for the wrong reason**. Under Go-syntax rendering at nesting depth greater than zero, a pointer-shaped wrapped cause renders as a machine address rather than its text — so a canary planted in a pointer-shaped cause would not surface even if the mechanism were unsafe. The recording MUST therefore use a **value-shaped** cause, and MUST ship a control.

---

## ADDED Requirements

### R-AIP-017 — The failure payload's redaction scope is stated, and the shape that escapes it is unreachable

The redaction `R-AIP-016` guarantees MUST be stated as scoped, and the shape outside that scope MUST be provably unreachable through the module's published surface. Specifically:

1. The guarantee is scoped to the payload **as callers receive it**. A copied value form of the payload is outside that scope and MUST NOT be reachable through any published surface of the module.
2. No published function, method or field of the module MAY return or store the failure payload by value. A guard MUST fail if that ever stops being true, naming the offending shape.
3. The **actual** rendering of the copied value form MUST be recorded rather than assumed, and MUST be recorded using a **value-shaped** wrapped cause. A recording that uses a pointer-shaped cause does not satisfy this item, because such a cause discloses only a machine address at nesting depth and therefore proves nothing either way.
4. The recording MUST ship a positive control establishing that the placement used can bite.
5. The scope MUST be stated in prose carried alongside the payload itself, so a later reader learns the boundary from the payload rather than from this spec.
6. No rendering behavior MAY be added to the payload's published set. `R-AIP-016` and `NFR-AIP-B` MUST remain satisfied unchanged.

#### Scenarios

- **S-AIP-058** — Given a copied value form of the failure payload whose wrapped cause is **value-shaped** and carries a planted canary, when it is rendered under the extended and Go-syntax verbs, then the resulting renderings are recorded and compared against the payload's redacted textual rendering, so any disclosure of the canary is visible on the record rather than assumed absent.
- **S-AIP-059** — Given that same construction with a **pointer-shaped** wrapped cause instead, when it is rendered under the same verbs, then the canary does not surface — establishing that the pointer-shaped placement is vacuous and that `S-AIP-058`'s value-shaped placement is the one that can bite.
- **S-AIP-060** — Given the shipped module, when every published function, method and field is examined, then none returns or stores the failure payload by value; and given a published shape deliberately introduced that does, when the guard runs, then it fails and names that shape.
- **S-AIP-061** — Given this change applied, when the payload's published rendering set is compared with the set `R-AIP-016` landed, then the two are identical; and when an absent payload is rendered under the plain, string, extended and Go-syntax verbs, then each yields the contract's defined absent-payload rendering and none panics (`NFR-AIP-B`, `S-AIP-052`, `S-AIP-057`).

---

## Pins / regressions

| Behavior leaf | Contract pin | Regression assertion |
| --- | --- | --- |
| Copied form's rendering recorded, not assumed | `R-AIP-016` | Value-shaped cause carrying a canary, both verbs |
| The proof is non-vacuous | — | Pointer-shaped placement shown not to surface |
| Leaking shape unreachable | `R-AIP-013`, `R-AIP-015` | Structural guard over the published surface, plus a deliberate violation |
| Landed guarantees untouched | `R-AIP-016`, `NFR-AIP-B` | Published rendering set identical; absent-payload totality re-run |
| Dependency purity | `NFR-AIP-A`, `S-AIP-051` | `go.mod` still declares zero requires; both import guards pass |

## Out of scope

| Item | Owner |
| --- | --- |
| Adding a value-shaped rendering to the payload | **Rejected** (proposal D-1) — it would enter the pointer form's rendering set and re-open `NFR-AIP-B` for a shape no constructor produces |
| Re-litigating `R-AIP-016`'s verb totality | **AI-41** — landed and consumed here, not repeated |
| The same shape of scope question on other pointer-rendered internal carriers | **Not applicable** — those carriers are unpublished and hold no credential or content |
| Identifiers, receivers and doc-comment wording for the scope statement and the guard | **Design phase** — this spec pins behavior only |
