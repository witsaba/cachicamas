# Spec index — `cachicamas-ai-redaction` (AI-36, Enforce secret redaction)

> **Milestone**: AI-36 · **Wave 5 — Harden** · **Module**: `backend/agent/` (layered, ADR 0005 § D1)
> **Delivery**: single PR with maintainer-accepted `size:exception`, budget raised to **1600 changed lines**. The proposal's two-PR chain recommendation is **overruled by the maintainer** and is not planned.
> **Strict TDD**: active. **Dependencies**: none added — `go.mod`/`go.sum` stay byte-identical.

> **Archive-time annotation (2026-08-08, `sdd-archive`).** The contingent row below **resolved to LANDED**: D-4's hostile-server case fired the credential branch, so `R-AEM-019` is binding, not conditional. The final closed totals are therefore **6 binding requirements / 23 binding scenarios**, not the 5/21 stated below. The body of this file is preserved as authored at spec time; only this annotation is added. See `archive-report.md` § "Specification Amendment Summary".

This file is an index. The binding delta specs are the per-capability files under `specs/`, per this repo's OpenSpec layout (`openspec/AGENTS.md` § "SDD artifact layout").

## Deltas

| # | Capability | Requirement | Scenarios | Charter leaf | Delta file |
| --- | --- | --- | --- | --- | --- |
| 1 | `ai-provider-conformance-suite` | `R-CNF-027` | `S-CNF-077` … `S-CNF-081` | AI-36.1 t1/t2/t3 | [`specs/ai-provider-conformance-suite/spec.md`](specs/ai-provider-conformance-suite/spec.md) |
| 2 | `ai-provider-client` | `R-APC-015` | `S-APC-077` … `S-APC-080` | AI-36.2 t1/t2 | [`specs/ai-provider-client/spec.md`](specs/ai-provider-client/spec.md) |
| 3 | `ai-request-translation` | `R-ART-022` | `S-ART-090` … `S-ART-094` | AI-36.3 t2 | [`specs/ai-request-translation/spec.md`](specs/ai-request-translation/spec.md) |
| 4 | `ai-stream-testkit` | `R-STK-014` | `S-STK-047` … `S-STK-049` | AI-36.3 t1 | [`specs/ai-stream-testkit/spec.md`](specs/ai-stream-testkit/spec.md) |
| 5 | `ai-provider-errors` | `R-AIP-017` | `S-AIP-058` … `S-AIP-061` | AI-41 close, doc 0002:22 | [`specs/ai-provider-errors/spec.md`](specs/ai-provider-errors/spec.md) |
| 6 | `ai-provider-error-mapping` | `R-AEM-019` **(contingent)** | `S-AEM-071`, `S-AEM-072` | AI-36.1 t2 (D-4) | [`specs/ai-provider-error-mapping/spec.md`](specs/ai-provider-error-mapping/spec.md) |

**Totals**: 5 binding requirements / 21 binding scenarios (5 + 4 + 5 + 3 + 4, in table order), plus 1 contingent requirement / 2 contingent scenarios.

## Numbering — corrected against the proposal

Every prefix was re-grepped at spec time across the **whole** `openspec/` tree, canonical specs **and** archived change deltas.

| Prefix | Canonical max | Archived-delta max | Assigned | Note |
| --- | --- | --- | --- | --- |
| `R-CNF` | 019 | **026** | **027** | ⚠️ **Proposal § 8 said `R-CNF-020`; that is a collision.** `R-CNF-020` … `R-CNF-024` are defined by the archived conformance **lifecycle** amendment and `R-CNF-025`/`R-CNF-026` by the archived conformance **tool** amendment. Neither set was promoted into the canonical file, so a canonical-only grep under-reads the true maximum by seven. |
| `R-APC` | 014 | 014 | 015 | matches proposal |
| `R-ART` | 021 | 021 | 022 | matches proposal |
| `R-STK` | 013 | 013 | 014 | matches proposal |
| `R-AIP` | 016 | 016 | 017 | matches proposal; `R-AIP-016` is AI-41's |
| `R-AEM` | 018 | 018 | 019 | matches proposal; contingent |
| `S-CNF` | 076 | 076 | 077+ | |
| `S-APC` | 076 | 076 | 077+ | |
| `S-ART` | 089 | 089 | 090+ | |
| `S-STK` | 046 | 046 | 047+ | |
| `S-AIP` | 057 | 057 | 058+ | |
| `S-AEM` | 070 | 070 | 071+ | contingent |

> **Standing hazard for later milestones.** The canonical `ai-provider-conformance-suite` spec has never absorbed `R-CNF-020` … `R-CNF-026`, yet other canonical specs already cite `R-CNF-023` and `R-CNF-024` as binding dependencies. Any future author grepping only `openspec/specs/` will re-derive the same wrong maximum. This is a promotion gap in the archive step, not in this change; it is recorded here rather than fixed here.

> **Archive-time confirmation (2026-08-08).** The hazard above was re-verified at close and **remains open by design**. `sdd-archive` promoted `R-CNF-027` and deliberately did **not** renumber or backfill `R-CNF-020` … `R-CNF-026`. An identifier note explaining the gap was added to the canonical conformance spec so a future author reads it in place; closing the gap remains a separate, recorded repository defect.

## Cross-cutting disciplines these deltas encode

1. **Every absence claim has a positive control.** The proposal names vacuous absence as this milestone's characteristic failure mode. Each sweep requirement carries a scenario proving the sweep detects a planted leak — `S-CNF-079`, `S-APC-077`/`S-APC-079`, `S-ART-090`, `S-STK-048`, `S-AIP-059`, `S-AEM-071`.
2. **No sweep reprints its match.** Reports name the vector, file, position or verb only.
3. **No Go identifier appears in any delta.** Behavior-level prose throughout, matching `R-AIP-016`'s landed register.
4. **The contingent requirement is held, not dropped.** `sdd-apply` resolves `R-AEM-019` to *landed* or *not-triggered with recorded evidence*.

## Non-goals, named explicitly

| Item | Owner |
| --- | --- |
| The credential-attachment proof | **AI-25.3** — charter Out-of-scope, verbatim |
| The wire-error-body size bound | **AI-32.5** — charter Out-of-scope, verbatim |
| The observability boundary and all OTel work | **AI-37** — AI-36 is its precondition, not its first slice |
| A general-purpose header-redaction utility | Non-goal — zero consumers today |
| Any new dependency | Non-goal — AI-37, not AI-36, is the second and last dependency-permitting milestone |

## Next

`sdd-design` (parallel with this phase) owns four open deliverables: the deliberate-plant file inventory, the shared sweep's placement, the canary-placement shape that makes `S-AIP-058` non-vacuous, and the hostile-server stub. Then `sdd-tasks`.
