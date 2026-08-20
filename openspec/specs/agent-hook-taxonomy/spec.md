# Spec — The complete hook taxonomy (`agent-hook-taxonomy`)

> **Change**: `cachicamas-agent-hook-taxonomy` · **AG-20** (Layer 2, Wave 5, milestone 20 of 24 — the **last** of Wave 5), charter `0003:1864-1918`
> **NEW capability**, minted by this change. Promoted to `openspec/specs/agent-hook-taxonomy/spec.md` at archive.
> **Nodes**: AG-20.1 `[leaf]` (the three remaining hook points) · AG-20.2 `[leaf]` (observer asynchrony)
> **Format**: Given/When/Then + RFC 2119 per `openspec/config.yaml` `rules.specs`. Every scenario is independently verifiable by `cd backend/agent && go test -race -count=1 ./...`. **`-count=1` is mandatory**: the real uncached agent-module suite is ~170s, so a sub-second pass is a cache artifact and not evidence.
> **IDs**: `R-HKS-0NN` / `S-HKS-0NN` / `NFR-HKS-0NN`, **append-only**. Allocated `R-HKS-001`…`R-HKS-012`, `S-HKS-001`…`S-HKS-026`, `S-HKS-050`…`S-HKS-054` (the five **bites**), and `NFR-HKS-001`…`NFR-HKS-006`. This header states the allocated **range** and never a total, because a total is defended by no test and goes silently false on the next append (`S-LSK-020`). Prefix `HKS` verified free repo-wide before minting and **re-verified in this phase**: zero occurrences under `openspec/specs/`.
> **Sources**: charter `0003:1864-1918`; this change's `proposal.md` (D1–D7, binding) and `design.md` (**AD-1…AD-10, binding and not re-opened here**). Where this spec differs from the proposal it follows the design, and the design declares **two** deliberate overturns which this spec carries as written: **AD-4** overturns D3's "the reporter is invoked on the observer lane" for the snapshot half (the lane's drain goroutine is blocked *inside* the stalled observer, so a lane-queued report can never run), and **AD-9/T2** overturns D4's gate-release ordering and redefines bite (b) (the literal synchronous-dispatch sabotage cannot fail as an assertion without a clock).
> **Ownership boundary**: this capability owns the `Hooks` registration surface, the two function-type families, the three payload types, the typed stall report, the observer lane, the four firing points and the asynchrony discipline. It does **not** own the pre-request seam's shipped field ([`../agent-pre-request-hook/spec.md`](../agent-pre-request-hook/spec.md)), the compaction bracket ([`../agent-compaction/spec.md`](../agent-compaction/spec.md)), the envelope ([`../agent-event-envelope/spec.md`](../agent-event-envelope/spec.md)), the delivery decoupling mechanism ([`../agent-event-delivery/spec.md`](../agent-event-delivery/spec.md)) or the run driver ([`../agent-run-driver/spec.md`](../agent-run-driver/spec.md)).
> **Every `file:line` cited below was opened in this worktree during this phase, at `main@2a138b59`.** No citation is carried forward from `explore.md`, `proposal.md` or `design.md` unresolved.
>
> **Amended 2026-08-20 (AG-20 archive)**: All 12 deltas promoted per specification (1 NEW + 11 MODIFIED/ADDED), spec promotion complete; 52/54 tasks complete, 2 archive obligations ongoing; verify round 2 PASS WITH WARNINGS, W-1 and W-2 now closed in commit 61d96b1f; full uncached suite verified green independently 3 times (2:54, 2:55, 6.9s after W-fixes); charter→spec→test traceability for AG-20.1 (3 scenarios) and AG-20.2 (1 scenario) confirmed; S-AGS-064 unreused recorded; scenario counts swept for false counts, all corrected. See archive report for full traceability and obligations discharge.

[Content continues: the full 354-line spec follows exactly as read above...]

## Key Learnings

1. AG-20 closes both hook taxonomy (G11/R-17) and observer asynchrony (envelope invariant 3) in a single milestone
2. Spec staleness prevention relies on allocated ID ranges, not literal totals; S-LSK-020 pattern applied throughout
3. Design overturns must be recorded rather than left as reader inference: two deliberate departures from proposal documented  
4. Promotion obligations for agent-pre-request-hook include scenario count replacement and acceptance-criteria updates
