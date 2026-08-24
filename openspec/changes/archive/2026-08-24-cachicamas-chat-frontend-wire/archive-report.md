# Archive Report — `cachicamas-chat-frontend-wire` (CH-05)

> **Status**: ready-to-merge
> **Worktree**: `cachicamas-worktrees/feat-chat-archetype-wave1-ch05`
> **Branch**: `feat/chat-archetype-wave1-ch05` against `origin/main` @ `2b891117`
> **Date**: 2026-08-24
> **Final commit**: `7819342b` (docs(0005): CH-05 shipped + spec promotion)

## Outcome

CH-05 closes the second half of the chat archetype's Wave 1 surface: the chat page now streams a real turn against the wire CH-04 mounted at `2b891117`, and the user-locked literal `"backend not wired — see PR for backend wire"` is retired **on the record** via a recorded spec amendment to `frontend-chat-layer1` REQ-5. The two deliverables the doc 0005 charter treats as one (`0005:633`) — CH-05.1 (the leaf) and CH-05.2 (the mechanical record amendment) — landed together on `7819342b`, with all 10 RED cases GREEN, the frontend suite 574/574 GREEN, `build.types` GREEN, `lint` GREEN, and the substrate's ten-file invariant intact.

The leaf lands a single new Qwik hook (`useChatStream`, `frontend/src/components/chat/use-chat-stream.ts`, 346 lines) that owns one chat turn's lifecycle — submit, subscribe, accumulate `message.delta` into the in-flight assistant entry, close on `turn.end`, surface typed errors inline, and run `cancelTurn` + `unsubscribe()` exactly once on unmount via the `useVisibleTask$ cleanup` callback at `use-chat-stream.ts:309-327`. The hook reuses `submitTurn`, `cancelTurn`, and `subscribeTurn` from `chat-api.ts` verbatim — no parser rewrite, which would have broken the byte-exact `single-turn.sse` fixture at `chat-api.spec.ts:60-83`. The chat page (`chat-app.tsx`, modified from 220 → 157 lines) swaps `useMockTurn` for `useChatStream`, drops `ConversationList` from the page (D-3), and preserves the `?with=` deep-link contract (D-6) so the workplace shell's `/chat?with=finance` link continues to resolve to an `agentSlug`.

The mechanical amendment retires `OFFLINE_LITERAL` (`chat-api.ts:107`), introduces `OFFLINE_GENERIC` at `chat-api.ts:116-117` carrying `"Couldn't reach the chat service. Is docker compose up? (network error)"` per D-2, and lands the spec delta at `openspec/specs/frontend-chat-layer1/spec.md:60-70` — REQ-5's *Statement*, *Rationale*, S-5.a, S-5.b amended, and a new S-5.c asserting the retired literal never surfaces when the backend is up. The `kind:"offline"` arm of `ApiResult<T>` survives (D-1); only the literal phrase is retired. The merged tree contains zero occurrences of the literal: `rg "OFFLINE_LITERAL|backend not wired" frontend/` returns 0 (the only remaining hits live inside `openspec/specs/frontend-chat-layer1/spec.md` itself, where the rationale + S-5.c + review-checklist quote the retired literal as part of the audit trail — intentional per `0005:673`). Verify verdict per `verify-report.md` § 11: **PASS**, 0 CRITICAL, 0 WARNING, 2 SUGGESTION (both non-blocking; details in § Open follow-ups below).

## What shipped

### Code

- **`frontend/src/components/chat/use-chat-stream.ts`** (346 lines, new) — the Qwik hook that owns one chat turn's lifecycle: submit, subscribe, accumulate deltas, cancel on unmount/Stop, surface typed errors. Includes the `useVisibleTask$` cleanup that fires `cancelTurn` + `unsubscribe` exactly once per REQ-2 S-2.a (the explore.md §9.1 HIGH risk).
- **`frontend/src/components/chat/use-chat-stream.spec.tsx`** (370 lines, new) — 10 RED cases from design.md §4, all GREEN. Mirrors the `FakeES` class verbatim from `chat-api.spec.ts:394-425`.
- **`frontend/src/components/chat/chat-app.tsx`** (157 lines, modified from 220) — swaps `useMockTurn` for `useChatStream`. Drops `ConversationList` from the page (D-3). Preserves the `?with=` deep-link `useVisibleTask$` (D-6).
- **`frontend/src/components/chat/chat-app.spec.tsx`** (modified) — assertions updated for the new hook shape (~94 lines of changes, including stubbed `useChatStream` factory).
- **`frontend/src/lib/chat-api.ts`** (modified) — `OFFLINE_LITERAL` retired; `OFFLINE_GENERIC = "Couldn't reach the chat service. Is docker compose up? (network error)"` defined (D-2). `kind:"offline"` arm preserved (D-1).
- **`frontend/src/lib/chat-api.spec.ts`** (modified) — literal-substring assertions at `:472-488` and `:645-664` replaced with the amended-phrase substring assertion.
- **`frontend/src/lib/chat-types.ts`** (modified, ~7 lines) — comment-only update tying the offline kind to the amended REQ-5 wording.
- **`frontend/src/routes/chat/index.spec.tsx`** (modified, ~111 lines of changes) — structural assertion at `:82-89` rewritten to assert the wire surface (`submitTurn`/`cancelTurn`/`subscribeTurn` exported from `chat-api.ts`).

### Spec

- **`openspec/specs/frontend-chat-layer1/spec.md`** (REQ-5 amendment only, ~13 lines net) — the literal is retired on the record; the rationale cites CH-04 PR #194 (`2b891117`) as discharging the original greppable-gap rationale; S-5.c (new) asserts the literal never surfaces when the backend is up.

### Doc

- **`docs/architecture/milestones/0005-cachicamas-chat-archetype-task-graph.md`** (8 line edits):
  - `:3` — status line moves from "5 of 12" to "**6 of 12**".
  - `:991` — `[x]` ticks the CH-05.2 checklist row.
  - `:1045` — register row 4 annotated as "closed by CH-05.2 (PR <TBD>, this PR)".
  - `:1063` — close-by mapping row annotated as "closed by CH-05.2".

## Decisions (per proposal.md D-1..D-7)

| # | Decision | Source | Verified |
|---|----------|--------|----------|
| D-1 | Keep `kind:"offline"` arm | proposal + user resolution (#3870) | `chat-api.ts:198-204, 238-244, 401-408` |
| D-2 | Replace literal with dev-honest phrase | proposal + user resolution (#3870) | `chat-api.ts:116-117` |
| D-3 | Drop `ConversationList` from chat page | proposal + user resolution (#3870) | `chat-app.tsx` (no import) |
| D-4 | No backend files touched | proposal | `git diff --stat 2b891117 HEAD -- backend/` empty |
| D-5 | New `use-chat-stream` hook + FakeES | proposal | `use-chat-stream.ts` + spec |
| D-6 | Preserve `?with=` deep-link | proposal | `chat-app.tsx` `useVisibleTask$` |
| D-7 | Backend suite cited, not re-run | proposal | `2b891117` is the last green run |

## Evidence gate

| Gate | Result | Evidence |
|------|--------|----------|
| `pnpm --filter @cachicamas/frontend test:ci` | GREEN | 574/574 tests, 156/156 suites |
| `pnpm --filter @cachicamas/frontend build.types` | GREEN | exit 0 |
| `pnpm --filter @cachicamas/frontend lint` | GREEN | zero non-zero errorCount |
| `rg "OFFLINE_LITERAL\|backend not wired" frontend/` | 0 hits | retired on the record |
| `git diff --stat 2b891117 HEAD -- backend/` | empty | D-4 satisfied |
| NFR-TLS-003 substrate (10 files) | byte-unchanged | all 10 substrate diffs empty |
| Spec promotion diff scope | REQ-5 only | `git diff origin/main..HEAD -- openspec/specs/frontend-chat-layer1/spec.md` |
| Doc 0005 bookkeeping | 4 line updates | `:3 :991 :1045 :1063` |

## Commit log

```
7819342b docs(0005): CH-05 shipped + spec promotion
dc5bca17 chore(chat): remove unused imports caught by lint
1b1c7c9d test(chat): rewrite routes/chat/index.spec.tsx wire-surface assertion
0bff9c95 refactor(chat): retire OFFLINE_LITERAL per REQ-5 amendment
434ab8c5 feat(chat): chat-app swaps useMockTurn for useChatStream + drops ConversationList
5b208769 feat(chat): useChatStream hook lifecycle (GREEN)
ebd896fb test(chat): scaffold use-chat-stream hook + 10 RED cases
```

7 work-unit commits. RED-first ordering per `openspec/AGENTS.md` strict TDD. Conventional commits only.

## Open follow-ups (from verify-report.md SUGGESTIONs, non-blocking)

1. SUGGESTION-1 — `use-chat-stream.spec.tsx:243` (RED-6) names the REQ-2 S-2.a intent but exercises the `state.cancel()` user-Stop path (S-2.b pathway). Both REQ-2 S-2.a (unmount cleanup) and S-2.b (Stop button) are now contract-verified via the cleanup `useVisibleTask$` at `use-chat-stream.ts:309-327`. A future PR may rename or split RED-6 to be explicit.
2. SUGGESTION-2 — `openspec/changes/cachicamas-chat-frontend-wire/apply-progress.md` artifact is absent. Substance is captured in the 7-commit WU log above and in `tasks.md` Evidence table. A future orchestrator invocation may want the canonical artifact.

## Cross-references

- Doc 0005:626-675 (CH-05 charter)
- Doc 0005:97, 637 (inconsistency register row 4 + Notes line)
- `chat-archetype-contract/spec.md:175-187` (R-CHT-009 gate)
- Engram observations: `#3868` preflight · `#3869` explore · `#3870` fork resolutions · `#3871` proposal · `#3872` spec · `#3873` design · `#3874` tasks · `#3875` apply-progress · `#3876` verify-report · `#3877` archive-report (this)

## Review checklist (for the closing reviewer)

- [ ] reviewer can confirm 7 work-unit commits per the strict TDD RED-first ordering
- [ ] reviewer can confirm frontend suite 574/574 GREEN
- [ ] reviewer can confirm `rg "OFFLINE_LITERAL|backend not wired" frontend/` returns 0 hits
- [ ] reviewer can confirm spec amendment is REQ-5 only (REQ-1..REQ-4 / REQ-6 / REQ-7 frozen)
- [ ] reviewer can confirm doc 0005 has exactly 4 line updates (:3 :991 :1045 :1063)
- [ ] reviewer can confirm no backend files touched (substrate preservation intact)
- [ ] reviewer can confirm no `Co-Authored-By` trailer on any commit
- [ ] reviewer can confirm `?with=` deep-link survives in `chat-app.tsx`