# Spec index — `cachicamas-ai-observability` (AI-37, Add the observability boundary)

> **Milestone**: AI-37 · **Wave 5 — Harden**, doc 0002's last open milestone · **Module**: `backend/agent/` (layered, ADR 0005 § D1)
> **Governing ADR**: [ADR 0005 § D3](../../../docs/adr/0005-promote-agent-stack-to-own-module.md#d3--observability-boundary) — the import table, the twelve-attribute allowlist, the absolute content denylist
> **Delivery**: single PR, `size:exception` pre-granted and auto-raising. **Strict TDD**: active.
> **Dependency**: this is the second and **last** milestone permitted to add one.

This file is an index. The binding delta specs are the per-capability files under `specs/`, per this repo's OpenSpec layout (`openspec/AGENTS.md` § "SDD artifact layout").

## Deltas

| # | Capability | Type | Requirements | Scenarios | Charter leaf | Delta file |
| --- | --- | --- | --- | --- | --- | --- |
| 1 | `ai-observability-boundary` | **New full spec** | `R-AOB-001` … `R-AOB-009` | `S-AOB-001` … `S-AOB-041` | AI-37.1 – AI-37.4 | [`specs/ai-observability-boundary/spec.md`](specs/ai-observability-boundary/spec.md) |
| 2 | `agent-module-scaffold` | Delta — 3 `MODIFIED`, 1 `ADDED` | `R-AGM-001`, `R-AGM-003`, `R-AGM-005` modified; `R-AGM-008` added | `S-AGM-068` … `S-AGM-073` new; `S-AGM-001` … `S-AGM-004`, `S-AGM-020` … `S-AGM-023`, `S-AGM-040` … `S-AGM-048` restated | AI-37.1 t1 | [`specs/agent-module-scaffold/spec.md`](specs/agent-module-scaffold/spec.md) |
| 3 | `ai-provider-client` | Delta — 2 `MODIFIED`, 1 `ADDED` | `R-APC-001`, `R-APC-003` modified; `R-APC-016` added | `S-APC-081` … `S-APC-085` new; `S-APC-001` … `S-APC-016` restated | AI-37.1 (D-1), AI-37.4 | [`specs/ai-provider-client/spec.md`](specs/ai-provider-client/spec.md) |

**Totals**: 18 requirements touched — 11 new (9 `R-AOB-*` + 2 `NFR-AOB-*`), 2 added (`R-AGM-008`, `R-APC-016`), 5 modified (`R-AGM-001`, `R-AGM-003`, `R-AGM-005`, `R-APC-001`, `R-APC-003`). **52 new scenarios** (41 + 6 + 5); 28 landed scenarios restated verbatim-or-amended so the archive step loses none. Recounted directly from the three delta files at this round, not carried forward from any earlier count.

## Numbering — re-verified at spec time

Every prefix was re-grepped across the **whole** `openspec/` tree at spec time: canonical specs, the active change folder, and archived change deltas alike. AI-36's spec index records why a canonical-only grep under-reads (`archive/2026-08-08-cachicamas-ai-redaction/spec.md:30, 43`).

| Prefix | Canonical max | Archived/active-delta max | Assigned | Note |
| --- | --- | --- | --- | --- |
| `R-AOB` | — | — | **001+** | New prefix. The token `AOB` occurs nowhere in the tree except this change's own `proposal.md`. |
| `S-AOB` | — | — | **001+** | Same. |
| `R-AGM` | 007 | 007 | **008** | matches proposal § 8 |
| `S-AGM` | 067 | 067 | **068+** | |
| `R-APC` | 015 | 015 | **016** | matches proposal § 8 |
| `S-APC` | 080 | 080 | **081+** | |

## Cross-cutting disciplines these deltas encode

1. **The dependency boundary is pinned by set equality, not by prefix subset.** The allowlist matcher is prefix-based, so an ecosystem-root entry would silently admit that ecosystem's metric, baggage, propagation and auto-instrumentation packages. `R-AGM-008` is the load-bearing half of AI-37.1 for exactly that reason.
2. **Every absence claim ships a falsifier.** `R-AOB-008` requires **four** independent non-vacuity guards, because a positive control proves the needles bite and proves nothing about whether the corpus was populated. This is the residual AI-36 recorded for AI-37 by name (doc 0002:24).
3. **Where an exact source of truth is missing, the claim is narrowed and the narrowing is recorded.** `R-AOB-006` makes the terminal-failure omission of `http.response.status_code` a requirement clause with a follow-up owner, not an implicit gap — the AI-41 precedent (doc 0002:22).
4. **No Go identifier appears in any delta.** Behaviour-level prose throughout. Semantic-convention attribute **keys** are the single deliberate exception, because AI-37.2 test 2 requires them spelled exactly.
5. **Every `MODIFIED` block restates its entire requirement**, including every unchanged scenario, so AI-00's and AI-25's landed proofs survive the archive step.

## Non-goals, named explicitly

| Item | Owner |
| --- | --- |
| Dashboards, exporters, application tracing setup | The composition root — charter Out-of-scope verbatim (doc 0002:2198) |
| Metrics, baggage, propagation, contrib, the SDK | Each needs **its own ADR** (§ D3 closing blockquote) |
| Trace-context propagation and header injection | Not this milestone — needs an excluded module, named by no charter test |
| An exact status code on the terminal-failure path | `ai-provider-errors`, per `R-AOB-006`'s recorded follow-up |
| Any conformance-suite change or a tenth capability member | Not this milestone — the charter is silent; if AI-38 makes tracing cross-adapter, AI-38 owns it |
| Tracing in Layer 2/3 or the composition root | Docs 0003/0004 |
| AI-36's other three recorded residuals | AI-37 discharges exactly **one** of the four: the corpus-non-empty guard |

## Assumptions this phase had to make

| # | Ambiguity in the proposal | Spec-level assumption |
| --- | --- | --- |
| A-1 | § 8 offered `R-AGM-008` as optional ("may carry the closure pin if `sdd-spec` prefers it separable"). | **Taken as separable.** The closure pin is the half that fails silently without it; a requirement of its own makes it independently citable and independently verifiable. |
| A-2 | § 8 named `R-AGM-001` and `R-AGM-005` as modified but not `R-AGM-003`. | **`R-AGM-003` is also modified.** Risk 7.2's workspace-divergence obligation and the first-ever generated workspace-sum file belong to that requirement, and leaving them unspec'd would make `S-AGM-068` homeless. |
| A-3 | The proposal did not state where the require-set/closure pin lives relative to AI-37's own capability. | **`agent-module-scaffold` owns the pin; `ai-observability-boundary` owns the import *choice*.** Duplicating the pin in both would create two sources of truth for one assertion. |
| A-4 | § 8 said `R-APC-001` "MODIFIED"; the mechanism of the amendment was left to this phase. | **Narrowly scoped second proof shape**, admissible only for an injectable whose effect does not reach the wire, with the field-read prohibition intact on both shapes. |

## Next

`sdd-design` (parallel with this phase) owns five open deliverables: the measured require set, versions and package closure under the workspace; the recording tracer's interface-satisfaction strategy; the precise uniform span-closing site proven against every terminal exit; the corpus builder's field walk and the recorder's captured-field-count assertion; and the shape of `R-AOB-008` item 4's build-time substitution. Then `sdd-tasks`.

---

> **Archive-time reconciliation note (2026-08-08).** `verify-report-final.md` found this index's own arithmetic stale (the "Totals" line above and the two prefix-range rows in "Deltas") after two Judgment Day correction rounds added `S-AOB-040`, `S-AOB-041` and `S-APC-085`. The counts stated above (18 requirements — 11 new + 2 added + 5 modified; 52 new scenarios [41+6+5]; 28 restated) are the **corrected, final** counts, recounted directly from the three delta files during the doc-only reconciliation round that preceded this archive. They supersede any earlier count this file itself carried during the correction rounds. See `verify-report-final.md` Part 3 for the full arithmetic audit.
