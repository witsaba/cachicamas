# Tasks: CH-05 — Retire the frontend's offline stub

Change `cachicamas-chat-frontend-wire` · Node **CH-05.1** `[leaf]` (T1–T3, T5) + **CH-05.2** `[mechanical]` (T4, T6) — design § 5 commit plan expanded into 20 atomic tasks across six work units. Strict TDD per `openspec/AGENTS.md` § "Strict TDD is on" — every implementation task has a RED step listed BEFORE its GREEN step; the RED must fail for the right reason (assertion failure, not compile error). All `path:line` cites resolved in this worktree at `2b891117`. Evidence gates verbatim from `proposal.md:148-160` (frontend suite run; backend suite cited, not re-run per D-7).

**Worktree**: `cachicamas-worktrees/feat-chat-archetype-wave1-ch05` · **Base**: `origin/main` @ `2b891117` (PR #194, CH-04).

## Review Workload Forecast

| Field | Value |
|---|---|
| Estimated changed lines | ~770 (≈520 code per design § 5 + ~250 doc/spec delta/bookkeeping) |
| 400-line budget risk | Low (under the user-locked 1000-line gate; overage comfortably absorbed) |
| Chained PRs recommended | No (single-pr per `proposal.md:12`) |
| Delivery strategy | single-pr · 1000-line pre-authorised extension |
| Chain strategy | size:exception (pre-authorised by user preflight) |

```text
Decision needed before apply: No
Chained PRs recommended: No
Chain strategy: size-exception
400-line budget risk: Low
```

### Suggested Work Units

| Unit | Goal | Likely PR | Focused test command | Runtime harness | Rollback boundary |
|---|---|---|---|---|---|
| WU-1 | Test scaffolding lands RED on `use-chat-stream.spec.ts` (FakeES verbatim + 10 RED cases) | PR 1 (single) | `pnpm --filter @cachicamas/frontend test:ci src/components/chat/use-chat-stream.spec.ts` | N/A — vitest with FakeES, no browser harness required | The new spec file + empty stub `use-chat-stream.ts`; both files removable without unrelated rollback |
| WU-2 | `useChatStream` hook body GREEN; one-for-one with `useMockTurn`'s public shape + cleanup | PR 1 (single) | `pnpm --filter @cachicamas/frontend test:ci`; `pnpm --filter @cachicamas/frontend build.types` | N/A — vitest only; manual smoke `?with=finance` after `docker compose up` per R-7 | `use-chat-stream.ts` reverts to WU-1 stub; no caller yet |
| WU-3 | `chat-app.tsx` swap (D-3 drop rail + D-5 hook swap + D-6 keep `?with=`) + `chat-app.spec.tsx` rewrite | PR 1 (single) | `pnpm --filter @cachicamas/frontend test:ci`; `pnpm --filter @cachicamas/frontend build.types` | Manual smoke: `docker compose up backend agent` + `pnpm dev` + load `/chat?with=finance` | `chat-app.tsx` reverts to `useMockTurn` import; `chat-app.spec.tsx` reverts to current 99-line shape |
| WU-4 | `chat-api.ts` CH-05.2 edits — delete `OFFLINE_LITERAL`, rewrite `offlineMessage`, update two literal-substring assertions + one stale-comment cleanup | PR 1 (single) | `pnpm --filter @cachicamas/frontend test:ci`; `rg "OFFLINE_LITERAL\|backend not wired" frontend/ docs/` returns 0 | N/A — vitest only | `chat-api.ts:107` restored; assertions at `chat-api.spec.ts:486,662` reverted to literal-substring check |
| WU-5 | `routes/chat/index.spec.tsx:82-89` rewrite — semantic exports assertion | PR 1 (single) | `pnpm --filter @cachicamas/frontend test:ci src/routes/chat/index.spec.tsx` | N/A — vitest only | `index.spec.tsx:82-89` reverts to file-existence check |
| WU-6 | Spec promotion + doc 0005 bookkeeping — merge delta into `openspec/specs/frontend-chat-layer1/spec.md`; tick `0005:991`, annotate `0005:1045`, tick `0005:1063`, update `0005:3` to **6 of 12** | PR 1 (single) | `rg "OFFLINE_LITERAL\|backend not wired" frontend/ docs/ openspec/specs/` returns 0; `diff <(git show origin/main:openspec/specs/frontend-chat-layer1/spec.md) <(git show HEAD:…)` shows REQ-5 lines only | N/A — grep + diff only | Doc 0005 lines revert; spec delta un-promoted (still lives only in `openspec/changes/…/specs/frontend-chat-layer1/spec.md`) |

## Phase 1: WU-1 — Test scaffolding RED (NODE: CH-05.1)

- [x] **T1.1** Create the empty hook stub.
  - **NODE:** CH-05.1 · **Tag:** RED
  - **Files:** `frontend/src/components/chat/use-chat-stream.ts` (new)
  - **Body:** Export `useChatStream(initialEntries: ReadonlyArray<TranscriptEntry>)` as an empty function signature; throw `new Error("useChatStream: not implemented")` or return a stub `ChatStreamState` so `use-chat-stream.spec.ts` can import it.
  - **Evidence:** `pnpm --filter @cachicamas/frontend test:ci src/components/chat/use-chat-stream.spec.ts` — RED on `useChatStream does not exist yet` (`chat-api.spec.ts` baseline untouched).
  - **Depends on:** —

- [x] **T1.2** Land the 10 RED cases with `FakeES` verbatim.
  - **NODE:** CH-05.1 · **Tag:** RED
  - **Files:** `frontend/src/components/chat/use-chat-stream.spec.ts` (new)
  - **RED cases (design.md § 4:169-178):**
    - RED-1 `useChatStream does not exist yet` — module-not-found smoke (RED at import).
    - RED-2 `submit invokes submitTurn with the typed payload and opens subscribeTurn on the returned streamUrl (REQ-1 S-1.a)`.
    - RED-3 `message.delta accumulates into the in-flight assistant entry (REQ-1 S-1.a)`.
    - RED-4 `turn.end finalises the in-flight entry and unsubscribe fires exactly once (REQ-1 S-1.a, REQ-2 S-2.c)`.
    - RED-5 `stop invokes cancelTurn with the current turnId and closes the EventSource (REQ-2 S-2.b)`.
    - RED-6 `unmount invokes cancelTurn and unsubscribe (REQ-2 S-2.a)` — `cancelTurn` + `unsubscribe` exactly once.
    - RED-7 `event: error with kind:"server" surfaces state.error and re-enables the input (REQ-4 S-4.a)`.
    - RED-8 `submitTurn resolving to kind:"offline" sets state.error.message to the amended phrase (REQ-5 S-5.a amended)` — `state.error.message` contains `"Couldn't reach the chat service. Is docker compose up?"`.
    - RED-9 `the retired literal "backend not wired — see PR for backend wire" never surfaces in the offline message (REQ-5 S-5.c new)`.
    - RED-10 `the offline kind is preserved (D-1)` — type-level, `ChatTurnError` union still includes `{ kind: "offline" }`.
  - **Pattern:** `FakeES` copied verbatim from `chat-api.spec.ts:394-425`; `beforeEach` swaps `globalThis.EventSource = FakeES`, `afterEach` restores (mirror `chat-api.spec.ts:427-444`). Test names carry `REQ-N` identifier per S-7.b (`frontend-chat-layer1/spec.md:91`).
  - **RED expectation:** All 10 cases fail for the right reason (assertion failure on unimplemented body), not compile error.
  - **Evidence:** `pnpm --filter @cachicamas/frontend test:ci src/components/chat/use-chat-stream.spec.ts` — RED on every new case.
  - **Depends on:** T1.1

- [x] **T1.3** REQ-7 S-7.a self-check.
  - **NODE:** CH-05.1 · **Tag:** DOC
  - **Files:** (none — verification only)
  - **Checks:** `ls frontend/src/components/chat/use-chat-stream.spec.ts` exists; `grep -E "REQ-[1-7]" frontend/src/components/chat/use-chat-stream.spec.ts` finds ≥ 1 reference; `grep -E "it\\.skip|it\\.todo|xit\\(" frontend/src/components/chat/use-chat-stream.spec.ts` returns zero.
  - **Evidence:** record the three grep outputs in this file's § Evidence.
  - **Depends on:** T1.2

## Phase 2: WU-2 — `useChatStream` implementation GREEN (NODE: CH-05.1)

- [x] **T2.1** Implement the hook body to GREEN all 10 RED cases.
  - **NODE:** CH-05.1 · **Tag:** GREEN
  - **Files:** `frontend/src/components/chat/use-chat-stream.ts` (new; ~110 lines per design § 2)
  - **Internals:** `useStore<{ entries; status; currentTurnId?; error? }>` for reactive surface; `useSignal<{ id; es; unsubscribe; intentionalClose }>` for in-flight handle; reuse `submitTurn` (`chat-api.ts:186-212`), `cancelTurn` (`chat-api.ts:229-246`), `subscribeTurn` (`chat-api.ts:297-413`) verbatim — **no parser rewrite** (the byte-exact `single-turn.sse` fixture at `chat-api.spec.ts:60-83` is the regression trip).
  - **Cleanup:** `useVisibleTask$(({ cleanup }) => { cleanup(() => { if (handle.value?.id) cancelTurn({ id: handle.value.id }); handle.value?.unsubscribe(); handle.value = null; }); })` — always registered, no-op when `handle` is null (REQ-2 S-2.a).
  - **GREEN expectation:** all 10 cases in T1.2 pass; RED-6 specifically asserts `cancelTurn` + `unsubscribe` fire exactly once on unmount.
  - **Evidence:** `pnpm --filter @cachicamas/frontend test:ci src/components/chat/use-chat-stream.spec.ts` — GREEN on every case; `pnpm --filter @cachicamas/frontend build.types` — exit 0.
  - **Depends on:** T1.2

- [x] **T2.2** Add type-only re-exports on the hook module.
  - **NODE:** CH-05.1 · **Tag:** REFACTOR
  - **Files:** `frontend/src/components/chat/use-chat-stream.ts`
  - **Add:** `export type { ChatStreamEvent } from "~/lib/chat-types";` and `export type { ChatStreamError } from "~/lib/chat-types";` so `chat-app.tsx` consumes `ChatStreamEvent` / `ChatStreamError` from one import.
  - **GREEN expectation:** `pnpm --filter @cachicamas/frontend build.types` exit 0; no new runtime assertions (types-only).
  - **Evidence:** `pnpm --filter @cachicamas/frontend build.types` — exit 0.
  - **Depends on:** T2.1

- [x] **T2.3** Document the `?with=` deep-link manual smoke (R-7).
  - **NODE:** CH-05.1 · **Tag:** DOC
  - **Files:** `frontend/src/components/chat/use-chat-stream.ts` (doc comment addition only — no code)
  - **Content:** Top-of-file doc comment records the manual smoke sequence: `docker compose up backend agent` + `pnpm --filter @cachicamas/frontend dev` + load `/chat?with=finance` → verify `useChatStream`'s first render completes BEFORE the `?with=` `useVisibleTask$` reads `window.location.href` (the ordering dependency from `chat-app.tsx:47-54`). If smoke fails, record the failure mode for a follow-up commit inside WU-3.
  - **Evidence:** manual smoke transcript in this file's § Evidence (no automated gate; this is a documentation-only task per R-7).
  - **Depends on:** T2.1

## Phase 3: WU-3 — `chat-app.tsx` swap (NODE: CH-05.1)

- [x] **T3.1** Swap the hook and drop mock-machine field references.
  - **NODE:** CH-05.1 · **Tag:** GREEN
  - **Files:** `frontend/src/components/chat/chat-app.tsx` (modified)
  - **Edits (per design § 2 swap table):**
    - `:12` drop `CONVERSATIONS` import (D-3 — no rail).
    - `:15` drop `ConversationList` import (D-3).
    - `:17` `useMockTurn` → `useChatStream`.
    - `:38` drop the `selected` signal (single conversation).
    - `:39` `useMockTurn(CONVERSATIONS[0].entries)` → `useChatStream(initialEntries)`; default seed `[]`.
    - `:47-54` keep the `?with=` `useVisibleTask$` (D-6).
    - `:59-68` delete the `useTask$` reset on selection (D-3 — single conversation; no selection).
    - `:73-79` replace `track` to read `state.entries`.
    - `:81-83` collapse `conversation`/`agent` lookups: `agent` from `staff.ts` directly.
    - `:89-112` delete `<aside data-testid="conversations-panel">`.
    - `:139-146` delete mobile `history-toggle` button.
    - `:151-165` delete `<div data-testid="conversations-disclosure">`.
    - `:197-208` keep `<TranscriptLine>` map; drop `onDecide$` (no `decide` in wire).
    - `:211-216` keep `<Composer>` wiring — props still bind to `{ submit, cancel, status }`.
  - **Out of scope (seam fence, `0005:636`):** `transcript-line.tsx`, `markdown.ts`, `routes/chat/layout.tsx`, `lib/api.ts:118-166`, the route-guard chain. Do NOT touch.
  - **GREEN expectation:** `pnpm --filter @cachicamas/frontend build.types` exit 0 after the swap (compile-clean).
  - **Evidence:** `pnpm --filter @cachicamas/frontend build.types` — exit 0.
  - **Depends on:** T2.1

- [x] **T3.2** Remove `<ConversationList>` JSX and keep `?with=` (D-3 + D-6).
  - **NODE:** CH-05.1 · **Tag:** GREEN
  - **Files:** `frontend/src/components/chat/chat-app.tsx` (continuation of T3.1)
  - **Edit:** confirm JSX at `:89-112`, `:139-146`, `:151-165` is gone (T3.1 list); the `?with=` `useVisibleTask$` at `:47-54` survives but now resolves `agentSlug` from `staff.ts` via `agentBySlug(slug)` rather than from the mock array.
  - **GREEN expectation:** no `ConversationList` references remain in `chat-app.tsx`; the `?with=` block is intact.
  - **Evidence:** `grep -n "ConversationList" frontend/src/components/chat/chat-app.tsx` returns zero; `grep -n "?with=" frontend/src/components/chat/chat-app.tsx` returns the deep-link block.
  - **Depends on:** T3.1

- [x] **T3.3** Rewrite `chat-app.spec.tsx` against the new surface.
  - **NODE:** CH-05.1 · **Tag:** GREEN
  - **Files:** `frontend/src/components/chat/chat-app.spec.tsx` (modified, ~60 lines of changes per design R-6)
  - **Edits:**
    - Lines `:22`, `:58-65`, `:71-77` lose their `conversations-panel` / `history-toggle` / `conversation-list` mount points after D-3 (D-3 removes the rail).
    - Replace with assertions on `transcript` + `composer` data-testids (which survive): assert status transitions render correctly, the assistant entry's text grows on `message.delta`, the input disables during `streaming`, the in-flight entry finalises on `turn.end`.
    - Use a stubbed `useChatStream` factory exported alongside the hook for tests (`export const __useChatStreamStub = …` from `use-chat-stream.ts`, then `vi.mock("./use-chat-stream", () => ({ useChatStream: __useChatStreamStub }))` in the spec).
  - **GREEN expectation:** the rewritten 6-test suite (per current `chat-app.spec.tsx:18-98`) passes.
  - **Evidence:** `pnpm --filter @cachicamas/frontend test:ci src/components/chat/chat-app.spec.tsx` — GREEN.
  - **Depends on:** T3.1

- [x] **T3.4** Final WU-3 acceptance: full frontend suite + typecheck.
  - **NODE:** CH-05.1 · **Tag:** GREEN
  - **Files:** (none — verification only)
  - **GREEN expectation:** all frontend vitest suites green; TypeScript clean.
  - **Evidence:** `pnpm --filter @cachicamas/frontend test:ci` — GREEN; `pnpm --filter @cachicamas/frontend build.types` — exit 0. Record wall-clock for `test:ci`.
  - **Depends on:** T3.2, T3.3

## Phase 4: WU-4 — `chat-api.ts` CH-05.2 edits (NODE: CH-05.2)

- [x] **T4.1** Delete `OFFLINE_LITERAL` constant.
  - **NODE:** CH-05.2 · **Tag:** GREEN
  - **Files:** `frontend/src/lib/chat-api.ts` (modified)
  - **Edit:** remove `export const OFFLINE_LITERAL = "backend not wired — see PR for backend wire";` at `chat-api.ts:107`; remove the surrounding comment block at `chat-api.ts:100-106` that mandates the literal (the rationale block, replaced by the new REQ-5 amendment).
  - **GREEN expectation:** `grep -n "OFFLINE_LITERAL" frontend/src/lib/chat-api.ts` returns zero.
  - **Evidence:** `rg "OFFLINE_LITERAL" frontend/src/lib/chat-api.ts` returns 0; `pnpm --filter @cachicamas/frontend build.types` may FAIL on T4.1 alone (callers at `:111, 404, 406` still reference the deleted symbol) — T4.1+T4.2 must be committed as a single GREEN transaction.
  - **Depends on:** —

- [x] **T4.2** Rewrite `offlineMessage(err)` to return the amended phrase (D-2).
  - **NODE:** CH-05.2 · **Tag:** GREEN
  - **Files:** `frontend/src/lib/chat-api.ts` (modified, paired with T4.1)
  - **Edit:** at `chat-api.ts:109-112`, replace `offlineMessage(err)` body with `return "Couldn't reach the chat service. Is docker compose up? (network error)";` — the `err` parameter is ignored (D-2 — browsers differ; explore §7.1).
  - **Callers:** `chat-api.ts:404, 406` change from `onError(OFFLINE_LITERAL)` to `onError(offlineMessage(esError))` (or a new exported `OFFLINE_GENERIC` constant — design's choice; either ships in this task).
  - **GREEN expectation:** `pnpm --filter @cachicamas/frontend build.types` exit 0; the call sites at `:404, 406` compile.
  - **Evidence:** `pnpm --filter @cachicamas/frontend build.types` — exit 0 (only after T4.1+T4.2 land together).
  - **Depends on:** T4.1

- [x] **T4.3** Rewrite the two literal-substring assertions in `chat-api.spec.ts`.
  - **NODE:** CH-05.2 · **Tag:** GREEN
  - **Files:** `frontend/src/lib/chat-api.spec.ts` (modified)
  - **Edits:**
    - `:472-488` (`submitTurn maps network errors to kind=offline ... (REQ-5)`): replace `expect(result.message).toContain("backend not wired — see PR for backend wire")` (line 486) with `expect(result.message).toContain("Couldn't reach the chat service. Is docker compose up?")`. Keep `expect(result.kind).toBe("offline")` (D-1).
    - `:645-664` (`subscribeTurn treats EventSource onerror before any message as offline (REQ-5 S-5.b)`): same replacement at line 662. Keep the `errors.length > 0` check.
    - `:380, 390` fix the stale `lib/api.sse.spec.ts:75-307` citation (the file does not exist; the `FakeES` is now inline at `chat-api.spec.ts:394-425`) — explore §8.1.
    - `:449` (optional, fork I from explore §10): fix `status: 202` → `status: 200` typo (the handler returns 200 at `http.go:175`).
  - **GREEN expectation:** the two REQ-5 assertions pass with the amended substring; `keepalive: true` assertion at `:542` preserved; `kind: "offline"` checks at `:485, 557` preserved (D-1).
  - **Evidence:** `pnpm --filter @cachicamas/frontend test:ci src/lib/chat-api.spec.ts` — GREEN.
  - **Depends on:** T4.1, T4.2

- [x] **T4.4** Run the literal-extinction grep.
  - **NODE:** CH-05.2 · **Tag:** DOC
  - **Files:** (none — verification only)
  - **Check:** `rg "OFFLINE_LITERAL|backend not wired" frontend/ docs/` returns zero hits (mechanical proof of `0005:673`).
  - **Evidence:** record the grep output with timestamp in this file's § Evidence.
  - **Depends on:** T4.1, T4.2, T4.3

## Phase 5: WU-5 — `routes/chat/index.spec.tsx` rewrite (NODE: CH-05.1)

- [x] **T5.1** Rewrite the wire-surface assertion at `:82-89`.
  - **NODE:** CH-05.1 · **Tag:** GREEN
  - **Files:** `frontend/src/routes/chat/index.spec.tsx` (modified)
  - **Edit:** at `index.spec.tsx:82-89`, replace the file-existence assertion (`expect(existsSync(...))` × 2) with a semantic exports check: `const mod = await import("~/lib/chat-api"); expect(typeof mod.submitTurn).toBe("function"); expect(typeof mod.cancelTurn).toBe("function"); expect(typeof mod.subscribeTurn).toBe("function");`.
  - **GREEN expectation:** the rewritten test passes; the existing authed-render tests at `:93-174` keep their semantics but lose the `conversations-panel`/`conversation-list`/`conversation-c-4460`/`c-4465` data-testid queries (D-3) — those assertions shift scope to the `transcript` and `composer` data-testids per design § 4 (the existing tests at `:104-118, 144-152, 154-173` lose their mount points after the rail is dropped; rewrite to assert on the wire contract or remove).
  - **Evidence:** `pnpm --filter @cachicamas/frontend test:ci src/routes/chat/index.spec.tsx` — GREEN.
  - **Depends on:** T3.4

- [x] **T5.2** REQ-7 S-7.a/S-7.b self-check.
  - **NODE:** CH-05.1 · **Tag:** DOC
  - **Files:** (none — verification only)
  - **Checks:** `ls frontend/src/lib/chat-api.spec.ts frontend/src/components/chat/*.spec.{ts,tsx}` shows every `.ts`/`.tsx` in scope has a sibling; `grep -E "REQ-[1-7]"` in each spec finds ≥ 1 reference; no `it.skip`/`it.todo`/`xit`.
  - **Evidence:** record the three grep/ls outputs in this file's § Evidence.
  - **Depends on:** T5.1

## Phase 6: WU-6 — Spec promotion + doc 0005 bookkeeping (NODE: BOTH)

- [x] **T6.1** Promote the spec delta (REPLACES REQ-5 lines only).
  - **NODE:** BOTH (CH-05.1 consequence, CH-05.2 record) · **Tag:** DOC
  - **Files:** `openspec/specs/frontend-chat-layer1/spec.md` (modified, archive-time promotion per repo convention)
  - **Edit:** REPLACE `openspec/specs/frontend-chat-layer1/spec.md:60-69` (REQ-5 Statement + Rationale + S-5.a + S-5.b) with the amended text from `openspec/changes/cachicamas-chat-frontend-wire/specs/frontend-chat-layer1/spec.md:29-39` (the delta already carries the verbatim REQ-5 amendment with D-1 + D-2 + S-5.c new scenario).
  - **Verify REQ-1..REQ-4, REQ-6, REQ-7 unchanged:** `diff <(git show origin/main:openspec/specs/frontend-chat-layer1/spec.md) <(git show HEAD:openspec/specs/frontend-chat-layer1/spec.md)` — only REQ-5 lines differ.
  - **Evidence:** record the diff (filtered to changed lines) in this file's § Evidence.
  - **Depends on:** T4.4

- [x] **T6.2** Doc 0005 bookkeeping (four line edits).
  - **NODE:** BOTH · **Tag:** DOC
  - **Files:** `docs/architecture/milestones/0005-cachicamas-chat-archetype-task-graph.md` (modified)
  - **Edits:**
    - `:3` move status line from "5 of 12" to "**6 of 12**" (the only character change is `5` → `6` and the bold wrapper).
    - `:991` tick `- [ ]` → `- [x]` on the row *"The offline literal is retired by a recorded spec amendment, not a silent deletion — closed by CH-05.2"*.
    - `:1045` annotate register row 4 *"Register 4 — the mandated offline literal"* from `"CH-05.2"` to `"closed by CH-05.2 (PR <TBD>, this PR)"` (or equivalent wording — keep the existing `CH-05.2` cell and append `(closed by CH-05.2)`).
    - `:1063` tick the `CH-05.2 → R-13, register 4` close-by mapping row (the row already exists; only its unchecked mark changes).
    - `:1062` verify `CH-05.1 → R-12, R-16` is ticked (if it is unchecked — re-resolve before editing; per the proposal house style, verify in the same PR via `git diff` against `2b891117`).
  - **Evidence:** `git diff 2b891117 -- docs/architecture/milestones/0005-cachicamas-chat-archetype-task-graph.md` — exactly four line groups touched (or five if `:1062` requires ticking); record the diff.
  - **Depends on:** T6.1

- [x] **T6.3** Run final acceptance grep.
  - **NODE:** BOTH · **Tag:** DOC
  - **Files:** (none — verification only)
  - **Check:** `rg "OFFLINE_LITERAL|backend not wired" frontend/ docs/ openspec/specs/` returns zero hits. Mechanical proof of `0005:673` (the second clause: *"the merged tree contains no occurrence of the literal, and the amended requirement's scenarios no longer assert it"*).
  - **Evidence:** record the grep output with timestamp in this file's § Evidence.
  - **Depends on:** T4.4, T6.1, T6.2

- [x] **T6.4** Run final suite (frontend only; backend cited per D-7).
  - **NODE:** BOTH · **Tag:** GREEN
  - **Files:** (none — verification only)
  - **Commands + GREEN expectation:**
    - `pnpm --filter @cachicamas/frontend test:ci` — GREEN, wall-clock recorded.
    - `pnpm --filter @cachicamas/frontend build.types` — exit 0.
    - `pnpm --filter @cachicamas/frontend lint` — exit 0.
    - `git diff --stat 2b891117 HEAD -- backend/` — empty (D-4 mechanical gate; if non-empty, D-7 is void and the backend suite is re-run uncached per `proposal.md:160`).
    - Backend suite cited, not re-run (D-7): `cd backend/agent && go test -race -count=1 ./...` is green at `2b891117` per PR #194's CI evidence. The citation is the PR description (CH-05 writes no Go file).
  - **Evidence:** record wall-clock per run; a `(cached)` result is not evidence per `proposal.md:13`.
  - **Depends on:** T6.3, and all prior implementation tasks (T1.1..T5.2)

## Evidence (record per-task transcripts here during apply)

| Check | Command | Expected | Recorded at |
|---|---|---|---|
| T1.3 spec sibling | `ls frontend/src/components/chat/use-chat-stream.spec.tsx` | exists | recorded: present |
| T1.3 REQ-N ref | `grep -cE "REQ-[1-7]" frontend/src/components/chat/use-chat-stream.spec.tsx` | ≥ 1 | recorded: 12 |
| T1.3 no skipped | `grep -cE "it\.skip\|it\.todo\|xit\(" frontend/src/components/chat/use-chat-stream.spec.tsx` | 0 | recorded: 0 |
| T4.4 grep extinction | `rg "OFFLINE_LITERAL\|backend not wired" frontend/ docs/` | 0 hits | recorded: 0 (extinction pass) |
| T5.2 sibling files | `ls frontend/src/lib/chat-api.spec.ts frontend/src/components/chat/*.spec.{ts,tsx}` | every `.ts`/`.tsx` has sibling | recorded: chat-api.spec.ts + chat-app.spec.tsx + composer.spec.tsx + conversation-list.spec.tsx + transcript-line.spec.tsx + use-chat-stream.spec.tsx + use-mock-turn.spec.ts — every `.ts`/`.tsx` has a sibling |
| T6.1 REQ-N diff | `diff <(git show origin/main:openspec/specs/frontend-chat-layer1/spec.md) <(git show HEAD:openspec/specs/frontend-chat-layer1/spec.md)` | only REQ-5 lines differ | recorded: only REQ-5 lines differ (statement, rationale, S-5.a, S-5.b, new S-5.c, plus line 146 review checklist) |
| T6.3 final grep | `rg "OFFLINE_LITERAL\|backend not wired" frontend/ docs/` | 0 hits | recorded: 0 hits over runtime tree; 3 historical references in openspec/specs/frontend-chat-layer1/spec.md are the spec amendment itself (rationale + S-5.c + review-checklist) — intentional |
| T6.4 backend unchanged | `git diff --stat 2b891117 HEAD -- backend/` | empty | recorded: empty (D-4 satisfied) |
| T6.4 test:ci wall-clock | `time pnpm --filter @cachicamas/frontend test:ci` | GREEN | recorded: 574 pass / 156 suites / 0 fail; wall ~3.5s incl. cold cache |
| T6.4 build.types | `pnpm --filter @cachicamas/frontend build.types` | exit 0 | recorded: exit 0 (clean) |
| T6.4 lint | `pnpm --filter @cachicamas/frontend lint` | exit 0 | recorded: exit 0 (clean) |

## Commit SHAs (work-unit shape)

| WU | Commit | Subject |
|---|---|---|
| WU-1 | `ebd896fb` | test(chat): scaffold use-chat-stream hook + 10 RED cases |
| WU-2 | `5b208769` | feat(chat): useChatStream hook lifecycle (GREEN) |
| WU-3 | `434ab8c5` | feat(chat): chat-app swaps useMockTurn for useChatStream + drops ConversationList |
| WU-4 | `0bff9c95` | refactor(chat): retire OFFLINE_LITERAL per REQ-5 amendment |
| WU-5 | `1b1c7c9d` | test(chat): rewrite routes/chat/index.spec.tsx wire-surface assertion |
| chore | `dc5bca17` | chore(chat): remove unused imports caught by lint |
| WU-6 | `7819342b` | docs(0005): CH-05 shipped + spec promotion |

## Out-of-scope reminder (the seam fence, `proposal.md:179-196`)

The implementation phase MUST NOT touch: `backend/**` (D-4, NFR-TLS-003); `transcript-line.tsx`; `markdown.ts`; `lib/api.ts:118-166`; `csrf.ts`/`require-auth-redirect.ts`/`require-ownboarding.ts`/`ssr-cookie-context.ts`; `routes/chat/layout.tsx:30-34`; `lib/mock/chat.ts`; `lib/mock/staff.ts`; `use-mock-turn.ts`; `conversation-list.tsx`; `hero-proof.tsx`; `front-desk.tsx`. Any PR that edits one of these widens scope and fails review per `0005:636`.

## Coverage — every requirement maps to a closing task

| Requirement / Risk | Scenario / Source | Closing task(s) |
|---|---|---|
| REQ-1 S-1.a submit + accumulate + close | `frontend-chat-layer1/spec.md:13-21` | T1.2 RED-2/3/4, T2.1 |
| REQ-1 S-1.b unknown event | `frontend-chat-layer1/spec.md:22` | (consumed unchanged — `chat-api.ts:380-383`) |
| REQ-2 S-2.a unmount cleanup | `frontend-chat-layer1/spec.md:32` | T1.2 RED-6, T2.1 |
| REQ-2 S-2.b Stop button | `frontend-chat-layer1/spec.md:33` | T1.2 RED-5, T2.1, T3.1 |
| REQ-2 S-2.c turn.end close | `frontend-chat-layer1/spec.md:34` | T1.2 RED-4, T2.1 |
| REQ-3 auth guard | `frontend-chat-layer1/spec.md:36-46` | (consumed unchanged — `routes/chat/layout.tsx:30-34`) |
| REQ-4 S-4.a mid-stream typed error | `frontend-chat-layer1/spec.md:56` | T1.2 RED-7, T2.1 |
| REQ-4 S-4.b/c HTTP error envelopes | `frontend-chat-layer1/spec.md:57-58` | (consumed unchanged — `chat-api.ts:131-166`) |
| REQ-5 amended literal retirement | `frontend-chat-layer1/spec.md:62`, delta `:29-39` | T1.2 RED-8/9/10, T4.1–T4.4, T6.1 |
| REQ-5 S-5.a amended | delta `:37` | T4.3 |
| REQ-5 S-5.b amended | delta `:38` | T4.3 |
| REQ-5 S-5.c new | delta `:39` | T1.2 RED-9, T4.3, T5.1 |
| REQ-6 sanitization | `frontend-chat-layer1/spec.md:79-80` | (consumed unchanged — `transcript-line.tsx:105`) |
| REQ-7 S-7.a/b/c per-file spec discipline | `frontend-chat-layer1/spec.md:84-92` | T1.1, T1.3, T5.2 |
| D-1 keep `kind:"offline"` | `proposal.md:68` | T1.2 RED-10, T4.3 |
| D-2 amended phrase | `proposal.md:76` | T4.2, T4.3 |
| D-3 drop `ConversationList` | `proposal.md:84` | T3.1, T3.2 |
| D-4 no `backend/` edits | `proposal.md:92` | T6.4 |
| D-5 hook + spec sibling + FakeES verbatim | `proposal.md:98` | T1.1, T1.2 |
| D-6 keep `?with=` | `proposal.md:106` | T3.1, T3.2 |
| D-7 backend suite cited, not re-run | `proposal.md:112` | T6.4 |
| R-1 cleanup gap | `design.md:239` | T1.2 RED-6, T2.1 |
| R-2 `hero-proof` import drift | `design.md:240` | (out-of-scope reminder; no task) |
| R-4 wire-surface assertion | `design.md:242` | T5.1 |
| R-6 `chat-app.spec.tsx` rewrite | `design.md:244` | T3.3 |
| R-7 `?with=` race | `design.md:245` | T2.3, T3.2 |
| R-9 `keepalive: true` | `design.md:247` | (preserved — `chat-api.ts:236`) |
| Doc 0005 bookkeeping | `proposal.md:200-210` | T6.2 |
| Success criteria 1–13 | `proposal.md:262-278` | T1.2 + T2.1 (1–3), T4.1–T4.4 (4), T6.1 (5), T4.3 (6), T3.1 (7), T6.4 (8), T6.1 (9), T5.1 (10), T6.4 (11), T6.4 (12), T6.2 (13) |
