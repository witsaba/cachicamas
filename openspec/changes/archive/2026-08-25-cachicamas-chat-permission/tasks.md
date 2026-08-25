# CH-10 — Tasks (cachicamas-chat-permission)

> Head reference for the task graph; full content in `engram://sdd/cachicamas-chat-permission/tasks` (observation #3992).

## Review Workload Forecast

| Field | Value |
|-------|-------|
| Estimated changed lines | 1800-3000 |
| 400-line budget risk | High |
| Chained PRs recommended | No (single-pr per preflight; size:exception pre-authorised at 1500) |
| Delivery strategy | single-pr (exception-ok per preflight) |

## Suggested Work Units

| Unit | Goal | Commit | Status |
|------|------|--------|--------|
| T-01 | RED scaffold #1 — 2 empty WireEvent variants | `75724077` | [x] DONE |
| T-02 | RED scaffold #2 — 4-arm stub projector | `8d6239a1` | [x] DONE |
| T-03a | CH-10.1 — port + tool + composition-root wire | `601e2230` | [x] DONE |
| T-03b | CH-10.1 — HTTP reverse-channel | `f59790bf` | [x] DONE |
| T-04 | CH-10.2 — wire projection + SSE + D-8 deny collapse | `6518db17` | [x] DONE |
| T-05a | CH-10.3a — Exchange widens + memory + 0003 | `41f1cfcb` | [x] DONE |
| T-05b | CH-10.3b — postgres sibling + 0004 + INTEGRATION | `1b9fd35d` | [x] DONE |
| T-06 | CH-10.4 — frontend delta + 11-arm parseTranscript | `09e76584` | [x] DONE |
| T-07 | CH-10.5 — F-CPM-001 projector-accumulator fix | `b918abba` | [x] DONE |
| T-08 | CH-10.6 — substrate + wire-fragmentation guards | `fdcb4d96` | [x] DONE |
| T-09 | Doc 0005 closure + spec promotion + archive | (archive) | [x] DONE |

## Constraints (binding, from preflight + AGENTS.md)

- Pre-authorised `size:exception` at preflight (review budget 1500 lines).
- No new Go top-level deps — `pgx/v5/stdlib` + `pressly/goose/v3` cover the postgres surface.
- No file under `backend/agent/src/agent/` modified (NFR-TLS-003 substrate preservation).
- Spec identifiers append-only — `R-CPM-001..099`, `S-CPM-001..199`, `NFR-CPM-001..099`.
- Strict TDD — every WU has a RED scaffold that pre-empts it.
- Cached `make test` is NOT evidence.
- F-CPM-001 closure lands in T-07 (CH-10.5 own WU).
- F-CPM-002/003 closure baked into T-04 (CH-10.2 wire projection arm).