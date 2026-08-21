# Delta — `agent-loop-skeleton` (AG-23)

> **Change**: `cachicamas-agent-layer3-handoff` · **AG-23** · Target: `openspec/specs/agent-loop-skeleton/spec.md`
> **Ops**: MODIFIED `Explicit non-requirements` — the **"Test-convenience wrappers"** row (`spec.md:294`), forwarded open across AG-13 and AG-21, is closed here on the merits: **DELIVERED, RELOCATED** to `backend/agent/src/apptest`, not `backend/agent/src/agenttest`. No requirement ID or scenario ID is added or renumbered — this row, like its sibling closures on the same list (`Value-form Harness`, `Multi-turn state`, `Iteration of the model ↔ tools ↔ model cycle`), is closed by prose annotation alone, matching the section's own established convention.
> **Decision**: `agent-layer3-handoff`'s `R-L3H-009` (the closed table of forwarded obligations) and design AD-3 (`D-2`'s rejection of widening `agenttest`).

## Why this row could not be closed by AG-21, and why it closes here

AG-21's own annotation on this row (reproduced below, unchanged) recorded it "Still open... deliberately: AG-21 uses the existing fixtures and widens `agenttest` by nothing." AG-23 is Layer 2's **last** milestone (`agent-layer3-handoff`'s own charter framing): a row still open after it has nowhere left to be forwarded to, so `R-L3H-009` requires every such row to resolve here — delivered, relocated, or declined with its reason recorded, and the resolution reachable from the shipped spec itself, not only from this change's own folder.

## Why RELOCATED, not simply DELIVERED in place

Two independent, already-shipped fences make `backend/agent/src/agenttest` the wrong home for these wrappers, verified this phase:

1. **Layer 1's own import guard sweeps `agenttest` under Layer 1's rules and denies `src/agent` by name** (`src/ai/import_boundary_test.go`'s `layer1Patterns`). Any implementation of a Layer 2 interface — `agent.PermissionPolicy`, `agent.Tool` — placed inside `agenttest` necessarily references Layer 2 types, and would fail that guard the instant it compiled inside the swept tree.
2. **`backend/agent/src/agenttest/` is separately frozen byte-unchanged by a shipped scope fence** (`hooks_test.go`'s `hksScopeFenceByteUnchangedFiles()`, filtered only for AG-22's own `tracetest` additions). Widening it here would fail that fence too, independent of the import-guard argument above.

The wrappers therefore ship one layer up, as their own sibling package — `backend/agent/src/apptest` — named by the same discipline `agenttest` itself uses: for the layer that **consumes** it. `agenttest` is Layer 1's own external-consumer kit, for the agent layer; `apptest` is the identical role one layer up, for the application layer.

## Not modified, and why

| Element | Verdict |
|---|---|
| Every other row on the `Explicit non-requirements` list | **Reproduced verbatim.** Only the "Test-convenience wrappers" row gains an addition; no other row's closure state changes |
| `R-LSK-002`, `R-LSK-004`, `R-LSK-006`, `R-LSK-009` and every other formal requirement in this spec | **Byte-unchanged.** AG-23 ships no new package inside `backend/agent/src/agent/` production source and edits no substrate file this spec names |
| `backend/agent/src/agenttest/` | **Byte-unchanged**, confirmed this phase — the relocation argument above depends on this remaining true |

## MODIFIED Explicit non-requirements

The list is reproduced in full; **one row is back-annotated as closed and none is removed.**

- **No edit** to any substrate file in `R-LSK-004`. AG-04/05/06 rule engine, registry, descriptor vocabulary, tests stay untouched. *(Still true at AG-21, which requests no release — `R-LSK-009`. Still true at AG-23, which requests no release either: AG-23's one production edit, `harness.go`, is on no substrate list.)*
- **No new top-level Go deps.** No `go.mod`/`go.sum` edit. No AG-03 guard edit. *(Still true at AG-21: `goleak` stays rejected, no ADR is written, and `go.mod`/`go.sum` are byte-unchanged — `R-CNH-008`. Still true at AG-23: the kit and the proof introduce zero new module dependencies — `agent-layer3-handoff`'s `R-L3H-011`.)*
- **Tools, hooks, errors beyond typed pass-through, permission, retry, context-check, cost events** — AG-08…AG-18.
- **Value-form `Harness`** — AG-13. AG-07 ships the function form only (D1). **CLOSED by AG-13**: the value-form `Harness` ships in this change and is owned by `R-RUN-001` in [`../agent-run-driver/spec.md`](../agent-run-driver/spec.md). AG-07's function form is unchanged and remains the only public one-turn surface.
- **Multi-turn state** — AG-21. AG-07 is stateless across calls (per `R-LSK-002`). **CLOSED by AG-21**, and closed by an **enumerated inventory with an absence assertion** rather than by a citation. AG-13 owns one run's iteration; state that outlives a **run** is now settled: the only state outliving a run is caller-owned or already enumerated by a shipped requirement, and the harness itself retains nothing. The inventory lives on `R-RUN-001` (`agent-run-driver`), and the proof is `S-CNH-014` (the absence half over a uniquely minted run-1 artifact, with an anti-ghost floor and a mandatory defeat test), `S-CNH-015` (the legitimate carry and the inventory row by row) and the defeat bite `S-CNH-016`. **The constraint on this capability is unchanged and still binding**: `Turn` remains stateless across calls per `R-LSK-002`, and nothing in this closure gives the one-turn surface memory.
- **Iteration of the model ↔ tools ↔ model cycle** — AG-13. AG-09 ships ONE cycle; AG-13 iterates. **CLOSED by AG-13**: iteration ships here, in the harness, by repeated `Turn` invocation — owned by `R-RUN-002`. `Turn` itself still schedules exactly one cycle per invocation (`R-LSK-006`, unweakened).
- **Test-convenience wrappers** in `backend/agent/src/agenttest` — AG-23. AG-07 uses `agenttest.Script` directly (per D4). *(Still open at AG-21, and deliberately: AG-21 uses the existing fixtures and widens `agenttest` by nothing — `R-CNH-002`, `S-CNH-006`. The wrappers remain AG-23's.)* **CLOSED by AG-23 (`cachicamas-agent-layer3-handoff`): DELIVERED, RELOCATED.** The wrappers ship as `backend/agent/src/apptest` — a new sibling package one layer up, never inside `backend/agent/src/agenttest`, which stays byte-unchanged as both the import guard and the shipped scope fence require. A genuinely third-party consumer package cannot live inside `agenttest`: Layer 1's own import guard sweeps it under Layer 1's rules and denies `src/agent` by name, so any Layer 2 interface implementation placed there would fail that guard immediately, independent of the separate byte-freeze.
- **CI** — every gate runs when a human runs `make test` in `backend/agent/`. *(Still true at AG-21. Note that `make test` carries no `-count=1`, so AG-21's own evidence gate is `go test -race -count=1 ./...` with the wall-clock duration recorded — `NFR-CNH-002`. Still true at AG-23, which records the same shipped `make test` command but not the identical evidence discipline: `agent-layer3-handoff`'s `NFR-L3H-B` reserves `-count=1` for focused/single-package runs and instead requires `go clean -testcache` immediately before an uncached whole-module `make test`, since adding `-count=1` to `make test` itself would contradict `agent-run-driver`'s own shipped pin of that exact command string.)*
