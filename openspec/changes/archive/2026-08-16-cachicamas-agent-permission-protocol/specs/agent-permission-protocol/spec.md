# Promotion record — `agent-permission-protocol` (new capability)

> **Change**: `cachicamas-agent-permission-protocol` · **AG-10** (Layer 2, Wave 2, milestone 10 of 24; doc 0003 `1005-1111`)
> **Status**: `agent-permission-protocol` is a **new capability**, not a delta against an existing one. Following the AG-09 precedent (`b2ab3867`), the normative spec is written directly to the capability directory rather than staged as a delta here.
> **Normative text**: [`openspec/specs/agent-permission-protocol/spec.md`](../../../../specs/agent-permission-protocol/spec.md) — 12 requirements (`R-APP-001`…`R-APP-012`), 13 scenarios (`S-APP-001`…`S-APP-013`), 4 bites (`S-PPB-001`…`S-PPB-004`), in Given/When/Then + RFC 2119 form per `openspec/config.yaml` `rules.specs`.
> **Cross-cut delta**: [`../agent-loop-skeleton/spec.md`](../agent-loop-skeleton/spec.md) — the `TurnOptions.PermissionPolicy` injection point modifies the pre-existing `agent-loop-skeleton` capability (`R-LSK-001`), and that MODIFIED block is what the archive step merges.

This file carries no requirement text of its own. It exists so the change folder records which capability AG-10 creates and where its normative spec lives.

## ADDED Capabilities

| Capability | Requirements | Scenarios | Location |
|---|---|---|---|
| `agent-permission-protocol` | `R-APP-001`…`R-APP-012` (12) | `S-APP-001`…`S-APP-013` (13) + 4 bites | `openspec/specs/agent-permission-protocol/spec.md` |

## MODIFIED Capabilities

| Capability | Requirement modified | Delta |
|---|---|---|
| `agent-loop-skeleton` | `R-LSK-001` — loop surface: `TurnOptions` gains `PermissionPolicy` (nil = identity bypass) | `specs/agent-loop-skeleton/spec.md` |
