# CH-10 apply-progress — cachicamas-chat-permission

> CH-10 of doc 0005 (`0005:936-947`). Branch `feat/chat-permission-ch10` based at `b4562280`.
> Strict TDD is ACTIVE; cached runs are NOT evidence.
> Runners: `cd backend/agent && make test` (uncached, race-clean), `cd frontend && pnpm --filter @cachicamas/frontend test:ci`.

## Plan (per #3992)

9 WUs producing ~10 conventional commits. T-03 splits into T-03a (port + tool) and T-03b (HTTP); T-05 splits into T-05a (Exchange widening + memory + 0003) and T-05b (postgres + 0004). All commits stay under 400 LOC; total PR ~2500 with `size:exception` pre-authorised at preflight.

## Status

| WU | Title | Commits | Status |
|----|-------|---------|--------|
| T-01 | RED scaffold #1 — 2 empty WireEvent variants | 1 | [ ] pending |
| T-02 | RED scaffold #2 — 4-arm stub projector | 2 | [ ] pending |
| T-03a | CH-10.1 — port + tool + composition-root wire | 3 | [ ] pending |
| T-03b | CH-10.1 — HTTP reverse-channel | 4 | [ ] pending |
| T-04 | CH-10.2 — wire projection + SSE + D-8 deny collapse | 5 | [ ] pending |
| T-05a | CH-10.3a — Exchange widens + memory + 0003 | 6 | [ ] pending |
| T-05b | CH-10.3b — postgres sibling + 0004 + INTEGRATION | 7 | [ ] pending |
| T-06 | CH-10.4 — frontend delta | 8 | [ ] pending |
| T-07 | CH-10.5 — F-CPM-001 projector-accumulator fix | 9 | [ ] pending |
| T-08 | CH-10.6 — substrate + wire-fragmentation guards | 10 | [ ] pending |
| T-09 | doc 0005 closure + spec promotion + archive | 11 | [ ] pending |

## Per-WU evidence gate (every WU must clear)

- `cd backend/agent && make test` — race-clean, uncached
- `cd backend/agent && make lint` — 0 issues
- `cd backend/agent && make build/chat` — produces `./bin/chat`
- `cd frontend && pnpm --filter @cachicamas/frontend test:ci` — green
- `cd frontend && pnpm --filter @cachicamas/frontend lint` — 0 errors
- `cd frontend && pnpm --filter @cachicamas/frontend build.types` — clean

After T-05b specifically: `cd backend/agent && INTEGRATION=1 make test` — S-CCS-021 GREEN.
After T-08 specifically: `cd backend/agent && git diff --stat main..HEAD -- backend/agent/src/agent/` — empty.