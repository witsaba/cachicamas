# Delta for `ai-event-envelope` — AI-41.1 discharges carryover `W1`

> **Change**: `cachicamas-ai-wave2-carryovers` · **Milestone**: AI-41 (doc 0002:2233–2257) · **Node**: AI-41.1 · **Wave 5 — Harden**
> **Carryover discharged**: `W1` — recorded at [`openspec/specs/ai-stream-testkit/spec.md:39`](../../../../specs/ai-stream-testkit/spec.md) and at [`openspec/specs/ai-event-envelope/spec.md:322–324`](../../../../specs/ai-event-envelope/spec.md) ("Carried forward")
> **Status**: **delta** — amends [`openspec/specs/ai-event-envelope/spec.md`](../../../../specs/ai-event-envelope/spec.md): one `ADDED` requirement appended after `R-AEE-020`, plus an append-only discharge note on the "Carried forward" section at archive time
> **Format**: RFC 2119 + Given/When/Then, matching the canonical spec's `R-AEE-*` / `S-AEE-*` grammar
> **Sources**: [proposal.md](../../proposal.md) · [explore.md](../../explore.md) · [doc 0002 § 2233–2257](../../../../../docs/architecture/milestones/0002-cachicamas-ai-layer-1-task-graph.md)

---

## Identity

| Field | Value |
| --- | --- |
| **Change name** | `cachicamas-ai-wave2-carryovers` |
| **Milestone node** | AI-41.1 — "the emission boundary's payload-validation rule is proven directly" |
| **Capability amended** | `ai-event-envelope` (per proposal § 8 Capabilities) |
| **Type** | Delta — **one `ADDED` requirement** (`R-AEE-021`, scenarios `S-AEE-071`, `S-AEE-072`); **no `MODIFIED`, no `REMOVED`, no `RENAMED`** |
| **Numbering** | Verified against the canonical spec at spec time: `R-AEE-020` and `S-AEE-070` are the current maxima |
| **Section amended at archive** | "Carried forward" (`:322–324`) gains an append-only discharge blockquote; `R-AEE-021` is promoted after `R-AEE-020` and before the `NFR-AEE-*` section |
| **Test runner** | `cd backend/agent && make test` (`go test -race -v ./...`) |
| **Module** | `backend/agent` — **layered** (not hexagonal), per ADR 0005 § D1 |

### What is being added

One behavior-only requirement pinning the last untested rule of the producer's emission-boundary check: the rule that consults the offered event's own payload validation. AI-14 recorded the gap and offered two resolutions — make a rejecting payload constructible, or record the rule as deliberately unreachable defence. AI-41.1's charter wording ("proven directly") selects the first. `R-AEE-021` therefore requires both the rejection behavior **and** its attributability, because a failure-path test that passes because an *earlier* rule fired proves nothing about this rule.

### Why this could not be proven before

Every production payload validates eagerly at construction and never yields an invalid value, so an event carrying a payload that fails its own validation is unconstructible through the public surface. The only payload this capability can construct directly is the test-only witness payload, whose validation reported success unconditionally. The gap is therefore a *constructibility* gap, not a behavior gap — which is exactly why the canonical spec parked it rather than testing around it.

---

## Emission-boundary payload validation (AI-41, dated 2026-08-07)

> **Amended 2026-08-07 (AI-41)** by `cachicamas-ai-wave2-carryovers` (AI-41.1, Wave 5 — Harden). One behavior-only requirement added: **`R-AEE-021`** — the emission-boundary check's payload-validation rule surfaces the offered payload's own rejection, proven directly and attributably. This discharges carryover `W1` by the first of the two resolutions the "Carried forward" section offers. No existing requirement is edited: `R-AEE-010`'s stamping rule and the earlier boundary rules are unpolluted, because this requirement adds an obligation on a rule they do not describe.

### ADDED Requirements

#### R-AEE-021 — The emission boundary surfaces the offered payload's own rejection, proven directly

As its final rule, the producer's emission-boundary check MUST consult the offered event's payload's own validation and MUST surface that payload's rejection **unchanged** — the same failure value, the same landed sentinel, and the same position the payload itself produced, at the event's own position. The boundary MUST NOT substitute a rejection of its own, MUST NOT introduce a new sentinel (`NFR-AEE-C`), and MUST NOT swallow the payload's rejection into a generic verdict.

This rule MUST be **proven directly** by a failure-path test rather than recorded as deliberately unreachable defence. The capability MUST therefore make a rejecting payload constructible from test-support code only. That construction MUST be additive: it MUST NOT widen any exported surface, MUST NOT reach a non-test build (`S-AEE-013`, `S-AEE-017`), and MUST leave every pre-existing construction path producing a payload that reports success, so that no already-landed behavior changes.

The proof MUST be **attributable**. The offered event MUST satisfy every earlier rule of the boundary check by construction, so that the payload-validation rule is the only rule that can be responsible for the rejection, and the assertion MUST match the surfaced rejection by identity against the planted one — never merely by observing that some rejection occurred.

#### Scenarios

- **S-AEE-071** — Given a stamped event whose test-only payload is configured to report a distinct, planted rejection, when the event is offered at the producer's emission boundary, then the boundary rejects it, and the returned failure **is that exact planted rejection** — matched by identity against the planted value, carrying its sentinel and its position unchanged — not merely a non-empty verdict.
- **S-AEE-072** — Given that same event, when a reviewer walks the boundary check's earlier rules against it, then each is satisfied by construction — the event carries a payload, it is stamped with a legal non-sentinel sequence (`R-AEE-010`), and its kind's descriptor declares no block role, so the block-scoped rule exits without a verdict — and therefore only the payload-validation rule can have produced `S-AEE-071`'s rejection; and given the same event with its payload left in its default state, when it is offered at the same boundary, then it is accepted.

---

## Carried-forward discharge (append-only, applied at archive)

The canonical spec's "Carried forward" section (`:322–324`) MUST NOT be rewritten or deleted — the record of *why* the gap existed is load-bearing history. It MUST gain an appended blockquote, immediately after the existing paragraph, in the canonical spec's established amendment form:

> **Discharged 2026-08-07 (AI-41)** by `cachicamas-ai-wave2-carryovers` (AI-41.1). `W1` is closed by the **first** of the two resolutions this section offers: the test-only witness payload gains a controllable rejection, so the emission boundary's payload-validation rule is proven **directly** rather than recorded as deliberately unreachable defence. The behavior is pinned by `R-AEE-021` (`S-AEE-071`, `S-AEE-072`). This gap is no longer carried forward.

Leaving this note absent — or leaving it present after a revert of the code that earns it — reproduces the exact failure mode AI-41 exists to stop.

---

## Pins / regressions

| AI-41 node | Behavior leaf | Contract pin | Regression assertion |
| --- | --- | --- | --- |
| AI-41.1 | Payload's own rejection surfaced unchanged | `NFR-AEE-C` (no new sentinel), `R-AEE-010` | Surfaced failure is the planted value by identity, with its sentinel and position intact |
| AI-41.1 | Attributability — only the payload rule can fire | `R-AEE-010`, `S-AEE-030` | Earlier rules structurally satisfied; default-state payload on the same event is accepted |
| AI-41.1 | Test-support construction never reaches production | `S-AEE-013`, `S-AEE-017` | No exported surface widened; the rejecting construction is unreachable from a non-test build |
| AI-41.1 | Zero regression on landed behavior | `NFR-AEE-B` (totality) | Every pre-existing construction path yields a payload reporting success; every landed test passes unchanged |
| AI-41.1 | Dependency purity | `NFR-AEE-A` | `backend/agent/go.mod` still declares zero requires; both import guards pass |

---

## Out of scope

| Item | Owner |
| --- | --- |
| Carryover `W2` (the provider-failure payload's Go-syntax rendering) | **AI-41.2** — `openspec/changes/cachicamas-ai-wave2-carryovers/specs/ai-provider-errors/spec.md` |
| The other three boundary rules' failure paths | **Already proven** — charter Out-of-scope, verbatim: behavior already covered is not re-litigated |
| The adversarial redaction sweep across the package surface | **AI-36** — consumes this result rather than repeating it |
| A second test-only payload type with an always-failing validation | **Rejected** (proposal Flag 2) — duplicates the existing witness payload's other proofs |
| The rejecting construction's Go identifiers, field shape and parallel-safety call | **Design phase** — this spec pins behavior only; design owns identifiers |

---

## Acceptance criteria

1. `R-AEE-021` holds, verified by `S-AEE-071` and `S-AEE-072`.
2. The surfaced rejection is asserted by identity against the planted one; a test that asserts only "some rejection occurred" does not satisfy `S-AEE-071`.
3. The earlier boundary rules are shown satisfied by construction for the offered event, and the same event with a default-state payload is accepted.
4. Every pre-existing test of the capability passes unchanged; no exported surface is widened.
5. No Go identifier — type, field, method, package path or signature — appears in this spec; behavior is stated at the contract level, as the canonical spec does.
6. `NFR-AEE-A` holds: `backend/agent/go.mod` still declares zero requires and both import guards pass.
7. **At archive**: `R-AEE-021` is promoted into the canonical spec after `R-AEE-020`, and the "Carried forward" section carries the append-only discharge blockquote quoted above.
8. `cd backend/agent && make test` is green under `-race`; `make lint` is clean.
