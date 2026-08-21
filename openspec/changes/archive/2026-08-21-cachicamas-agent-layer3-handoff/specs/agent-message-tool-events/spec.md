# Delta — `agent-message-tool-events` (AG-23)

> **Change**: `cachicamas-agent-layer3-handoff` · **AG-23** · Target: `openspec/specs/agent-message-tool-events/spec.md`
> **Ops**: MODIFIED `Explicit non-requirements` — the **"Test-convenience wrappers"** row (`spec.md:109`), open since this spec shipped, is closed here on the merits: **DELIVERED, RELOCATED** to `backend/agent/src/apptest`, not `backend/agent/src/agenttest`. This is this spec's own first amendment of the row; no requirement ID or scenario ID is added or renumbered.
> **Decision**: `agent-layer3-handoff`'s `R-L3H-009` and design AD-3 (`D-2`'s rejection of widening `agenttest`), the identical resolution `agent-loop-skeleton`'s and `agent-protocol-events`' own deltas in this change record.

## Why this row closes here

This spec's own `Explicit non-requirements` list has carried the "Test-convenience wrappers" row, assigned to AG-23, since it shipped, and no intervening milestone amended it. AG-23 is Layer 2's last milestone: a row still open after it has nowhere left to be forwarded to, so `agent-layer3-handoff`'s `R-L3H-009` requires this row to resolve here, reachable from the shipped spec itself.

## Why RELOCATED, not delivered in place

Identical reasoning to the sibling deltas in this change, restated here so a reader who arrives at this spec alone still meets the full argument: Layer 1's own import guard (`src/ai/import_boundary_test.go`'s `layer1Patterns`) sweeps `agenttest` under Layer 1's rules and denies `src/agent` by name, so any implementation of a Layer 2 interface placed inside it would fail that guard immediately; independently, `backend/agent/src/agenttest/` is frozen byte-unchanged by a shipped scope fence (`hooks_test.go`'s `hksScopeFenceByteUnchangedFiles()`). The wrappers therefore ship as their own sibling package, `backend/agent/src/apptest`, named by the same discipline `agenttest` itself uses — for the layer that consumes it.

## Not modified, and why

| Element | Verdict |
|---|---|
| Every other row on the `Explicit non-requirements` list | **Reproduced verbatim.** Only the "Test-convenience wrappers" row gains an addition |
| `R-AMT-003`, `R-AMT-008`, `R-AMT-009` and every other formal requirement in this spec | **Byte-unchanged.** AG-23 emits no new event kind and edits no substrate file this spec names |
| `event_descriptor.go`, `stream_check.go`, `failure.go`, `sequence.go` | **Byte-unchanged**, confirmed this phase |
| `backend/agent/src/agenttest/` | **Byte-unchanged**, confirmed this phase — the relocation argument above depends on this remaining true |

## MODIFIED Explicit non-requirements

The list is reproduced in full; **one row is back-annotated as closed and none is removed.**

- **No edit** to `event_descriptor.go`, `stream_check.go`, `failure.go`, or `sequence.go` — AG-04's rule engine stays untouched; AD-1, AD-2, AD-5 hold. *(Still true at AG-23, whose one production edit is confined to `harness.go`.)*
- **No new top-level Go deps.** No `go.mod`/`go.sum` edit. No AG-03 guard edit. *(Still true at AG-23 — `agent-layer3-handoff`'s `R-L3H-011` — which adds zero module dependencies.)*
- **Permission, cost, delegation, compaction families** (`VL2-EVT-06`…`09`) — **AG-06**.
- **Live-loop emission** — **AG-07, AG-09**. AG-05 ships no producer.
- **Tool execution contract** (scheduling, rejoin) — **AG-09.1**. **Permission events around tools** — **AG-06, AG-10**.
- **Test-convenience wrappers** in `backend/agent/src/agenttest` — **AG-23**. **CLOSED by AG-23 (`cachicamas-agent-layer3-handoff`): DELIVERED, RELOCATED.** The wrappers ship as `backend/agent/src/apptest` — a new sibling package one layer up, never inside `backend/agent/src/agenttest`, which stays byte-unchanged as both the import guard and the shipped scope fence require.
- **CI** — every gate runs when a human runs `make test` in `backend/agent/`.
