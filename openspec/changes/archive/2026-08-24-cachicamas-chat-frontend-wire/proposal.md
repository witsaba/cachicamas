# Proposal: CH-05 — Retire the frontend's offline stub

| | |
|---|---|
| **Change** | `cachicamas-chat-frontend-wire` |
| **Milestone** | CH-05 (`docs/architecture/milestones/0005-cachicamas-chat-archetype-task-graph.md:626-675`) · Wave 1 · **6 of 12** once merged |
| **Closes** | R-13, R-16, register row 4 |
| **Nodes** | CH-05.1 `[leaf]` (`0005:639-669`) · CH-05.2 `[mechanical]` (`0005:671-675`) |
| **Depends on** | CH-04 — merged (PR #194, `2b891117`). **Blocks** nothing in Wave 1 (CH-06 onwards depend on the archetype + the page, not on this change specifically) |
| **Worktree / base** | `cachicamas-worktrees/feat-chat-archetype-wave1-ch05`, branch `feat/chat-archetype-wave1-ch05`, based on `origin/main` @ `2b891117` |
| **Artifact store** | engram (`openspec/changes/…` + Engram `sdd/cachicamas-chat-frontend-wire/*`) — user-locked in preflight |
| **Delivery** | single-pr · review budget **1000** counted lines, **pre-authorised extension** for the evidence gate (the user's explicit instruction is "extend if needed"; the gate is the 1000-line cap, not a hard 400) |
| **Evidence runner** | `pnpm --filter @cachicamas/frontend test:ci` (vitest red-green), plus `pnpm --filter @cachicamas/frontend build.types` and `pnpm --filter @cachicamas/frontend lint`. **Backend unchanged** — CH-04's `cd backend/agent && go test -race -count=1 ./...` at `2b891117` is cited as the evidence the wire's server side is green, **not** re-run. A `(cached)` result is not evidence |
| **Prefix** | **`F5W`** (frontend, CH-05 wire) — verified free this phase: `rg -n "F5W-" frontend/ docs/architecture/milestones/0005-cachicamas-chat-archetype-task-graph.md openspec/specs/frontend-chat-layer1/ openspec/changes/ openspec/specs/` returns 0 hits. `FCL` was rejected because the original `frontend-chat-layer1` capability owns it (`R-FCL-0NN`, `openspec/changes/archive/2026-08-06-cachicamas-frontend-chat-layer1/proposal.md:206`) and the proposal cannot rename the existing prefix |
| **Exploration** | `openspec/changes/cachicamas-chat-frontend-wire/explore.md` · Engram `#3869` |
| **Binding decisions** | Engram `#3870` — three user forks, settled as D-1, D-2, D-3 below |

> Every `file:line` below was re-resolved by reading the file in **this worktree** during this phase, or is a verbatim quote from the doc 0005 charter (`0005:626-675`). Where the exploration and this phase disagree, this phase's reading governs.

---

## Intent

The chat page is a mockup that apologises for a backend that now exists. `frontend/src/components/chat/chat-app.tsx:39` calls `useMockTurn(CONVERSATIONS[0].entries)`; the page renders pre-baked scripted beats driven by `setInterval`, not real wire events. The wire client itself — `frontend/src/lib/chat-api.ts` and its sibling `frontend/src/lib/chat-types.ts` — was frozen in `cachicamas-frontend-chat-layer1` to *exactly the right shape* (`openspec/specs/frontend-chat-layer1/spec.md:15` — *"Submitting a prompt from `/chat` SHALL open `POST /api/agent/turns`, open an `EventSource` on the returned `stream_url`, accumulate `event: message.delta` text into a single assistant bubble, and close the stream on `event: turn.end`."*) but the page never called it. CH-04 (`backend/agent/src/chat/http.go:305-307`) mounted `POST /api/agent/turns`, `GET …/events` and `DELETE …/:id` on the same `2b891117` main; CH-05 is the natural surface that wires the page to what the backend now serves.

The charter makes the second half of this milestone load-bearing. The literal `"backend not wired — see PR for backend wire"` (`chat-api.ts:107`) was mandated by REQ-5 (`spec.md:62`) so a developer running the frontend without the backend would see a phrase that grepped straight to the gap. The gap is now closed, and **silently deleting the string would leave a promoted requirement falsified by shipped code** (`0005:637`, `0005:97`). `chat-archetype-contract` R-CHT-009 (`openspec/specs/chat-archetype-contract/spec.md:181`) repeats the rule: *"retiring it needs a recorded spec delta at CH-05.2."* The delta is the second of two deliverables the charter treats as one (`0005:633`), and the merged tree's mechanical proof — `rg "OFFLINE_LITERAL\|backend not wired" frontend/` returns zero — is part of `0005:673`'s check evidence, not an afterthought.

CH-05 is therefore two real changes that cannot be reordered: the leaf first (`0005:639-669`, scoped to one new hook and the page's swap) and the spec amendment second (`0005:671-675`, which cannot precede the leaf because REQ-5 S-5.a's *"the offline literal appears nowhere in the page"* scenario is read against a page that no longer renders the literal).

## Scope

### In scope

**CH-05.1 — `frontend/src/components/chat/use-chat-stream.ts` (new) + `use-chat-stream.spec.ts` (new, colocated).** A Qwik hook that calls `submitTurn`, opens `subscribeTurn` on the returned `stream_url`, accumulates `message.delta` frames into a signal of `TranscriptEntry[]` (`chat-types.ts:27-32`, mirrored at `chat-types.ts:104-117`), and exposes `{ submit, cancel, state, messages }` mirroring the `useMockTurn` shape (`use-mock-turn.ts:55-75`) so the page swap is one-for-one. The hook carries `useVisibleTask$(({cleanup, track}) => { … cleanup(() => { cancelTurn(req); unsubscribe(); }) })` — the cleanup REQ-2 S-2.a (`spec.md:32`) requires but `chat-app.tsx:47-54, 73-79` lacks today. The S-7.a spec assertion (`spec.md:84-92`) requires the spec sibling.

**CH-05.1 — `frontend/src/components/chat/chat-app.tsx` (modified).** Swap `useMockTurn` (`chat-app.tsx:17, 39`) for `useChatStream`; replace the `useTask$` reset on selection (`chat-app.tsx:59-68` — its three reset lines are mock-machine API and become inert) with a hook-driven state. Keep the `?with=` deep-link contract (`chat-app.tsx:47-54`) — D-6 retains it; the deep-link still selects the active conversation, the rail's `data-testid` selection goes away (D-3). The `<Composer>` props (`chat-app.tsx:211-216`) bind unchanged.

**CH-05.1 — `frontend/src/components/chat/conversation-list.tsx` import removed from `chat-app.tsx`** (D-3). The mock rail is dropped until CH-08.2 (`cachicamas-chat-conversation-list`) lands. `ConversationList` itself survives — `chat.ts:11` exports it and `chat.spec.ts` asserts on it — because the marketing `hero-proof` and the workplace `front-desk` panel are unaffected.

**CH-05.1 — `frontend/src/routes/chat/index.spec.tsx` (modified).** Rewrite the assertion at `index.spec.tsx:82-89` (`routes/chat: the frozen browser wire is still on disk, unwired`). After CH-05.1 the wire is connected, not unwired; the test must assert the wire surface — that `chat-api.ts` exports `submitTurn`, `cancelTurn` and `subscribeTurn` — rather than file existence.

**CH-05.2 — `openspec/changes/cachicamas-chat-frontend-wire/specs/frontend-chat-layer1/spec.md` (new, mirrored per repo convention).** Spec delta on `frontend-chat-layer1` REQ-5: the literal is retired; the `kind:"offline"` arm of the error envelope (`api.ts:89-94`, `chat-types.ts:93-98`) survives as the generic network-failure shape. The amendment cites the greppable architectural gap as discharged by CH-04 and reproduced under `0005:673`'s *"the merged tree contains no occurrence of the literal, and the amended requirement's scenarios no longer assert it"*.

**CH-05.2 — `frontend/src/lib/chat-api.ts` (modified).** Delete `OFFLINE_LITERAL` (`chat-api.ts:107`); rewrite `offlineMessage(err)` (`chat-api.ts:109-112`) to return the generic dev-honest phrase per D-2; rewrite `chat-api.ts:404, 406` to call the new `offlineMessage(err)`; rewrite the design comments at `chat-api.ts:27, 103, 180, 260` and `chat-types.ts:91` to drop the literal phrase.

**CH-05.2 — `frontend/src/lib/chat-api.spec.ts` (modified).** Rewrite the assertion at `chat-api.spec.ts:472-488` so `result.message` is asserted to contain the generic phrase (not the retired literal); same at `chat-api.spec.ts:645-664`. Both remain REQ-5 tests, but the literal-substring check is replaced.

**Doc 0005 bookkeeping** in the same PR: tick the checklist row at `0005:991`; annotate register row 4 (`0005:1045`) as closed; tick the close-by mapping at `0005:1063`.

### Out of scope

| Excluded | Owner / reason |
|---|---|
| Any file under `backend/` — including `backend/agent/src/chat/*`, `backend/agent/go.{mod,sum}`, `backend/agent/Makefile`, `.golangci.yml` | D-4. Substrate preservation (NFR-TLS-003, `openspec/AGENTS.md` § "Substrate preservation in `backend/agent`"); the wire CH-05.1 reads is CH-04's merge-state |
| Rendering, sanitization, the route guard and the error envelope | `0005:636`. Frozen by `frontend-chat-layer1` and consumed unchanged |
| Conversation history on reload | `0005:636`. Owned by CH-08 (`cachicamas-chat-conversation-list`); the rail's mock array stays in `lib/mock/chat.ts` for `hero-proof` and `front-desk` until CH-08.2 |
| The marketing `hero-proof` mock | explore.md §9.6. `components/marketing/hero-proof/hero-proof.tsx:21, 31, 44, 50, 89` still imports `useMockTurn`; `useMockTurn` survives CH-05 |
| The workplace `front-desk` "where you left off" panel | explore.md §9.7. `components/workspace/screens/front-desk.tsx:114-140` still renders the mock `CONVERSATIONS` |
| The CH-05.2 dead-citation cleanup beyond `api.sse.spec.ts:75-307` | explore.md §9.9. The `subscribeWorkspaceSyncStream` reference at `frontend-chat-layer1/spec.md:17, 130` is unrelated to REQ-5 and is left for a future amendment |
| A new test for the optional `chat-api.spec.ts:380, 390` comment drift about `api.sse.spec.ts:75-307` | explore.md §10 fork J. The comments are CH-05.1's free hand (not in a promoted spec) but the cleanup is recorded-not-repaired, not in-scope here |
| A systemHint UI control | explore.md §10 fork E. `chat-types.ts:71-72` reserves `systemHint?`; CH-05.1 omits the field (the minimum-work shape) |
| Provider / model / credential selection in frontend | doc 0001 §5.2 — "the back is who controls the keys" |
| Tool-call rendering, subagent nesting, permission protocol, streaming reasoning, multimodal content | doc 0001 §8 + §7 G1/G7/G12 |
| Real end-to-end demo against a live backend | D-vitest-only — vitest with FakeES is the v1 evidence posture |

## Resolved decisions

### D-1 — Retire only the literal phrase; keep `ApiErrorKind = "offline"` as the network-error arm

**Decided** (user, Engram `#3870` item 1, was explore.md §10 fork A). The amendment deletes the user-locked string but leaves `offline` in the `ApiResult<T>` discriminated union (`api.ts:89-94`, `chat-types.ts:93-98`) and the matching `ChatTurnError` shape (`chat-types.ts:93-98`). The `kind:"offline"` arm is the right shape for any transient network failure and is never produced by the server (explore.md §7.4) — removing it would fold transport failure into `kind:"server"`, which `frontend-chat-layer1` REQ-4 already covers for HTTP error envelopes, collapsing a meaningful client-side distinction (`explore.md §7.4`).

**Why not also delete the offline kind.** `chat-archetype-contract` R-CHT-009 (`spec.md:177`) enumerates REQ-1 … REQ-7 as the inherited wire, and `chat-archetype-contract` R-CHT-009 (`spec.md:181`) is the gate that forbids silent retirement. The shape amendment is the literal retirement — exactly what `0005:637` and `0005:673` describe — no more.

**Consequence for the spec delta.** REQ-5's *"the literal appears nowhere in the page"* scenario is read against a page that no longer renders the literal; the `kind:"offline"` shape assertion survives; REQ-5 S-5.a and S-5.b lose the literal-substring check but keep the network-failure-shape check.

### D-2 — `offlineMessage(err)` returns a generic dev-honest phrase — not `err.message` alone, not the retired literal

**Decided** (user, Engram `#3870` item 2, was explore.md §10 fork B shape 1). The replacement phrase is `"Couldn't reach the chat service. Is docker compose up? (network error)"`. It mirrors the dev-honest formatter at `api.ts:161` (a hard-coded message for a network failure) and preserves the *shape* of REQ-5's *"the client SHALL NOT silently retry, SHALL NOT fabricate a response, and SHALL NOT log only to console"* invariant (the message is honest about what went wrong without naming an absent backend).

**Why this and not bare `err.message`.** A bare `err.message` on a failed `fetch` is browser-specific (`TypeError("Failed to fetch")` in Chrome, `TypeError("NetworkError when attempting to fetch resource.")` in Firefox — explore.md §7.1). The phrase stays dev-honest across browsers while keeping the developer pointed at the right command (`docker compose up`).

**Consequence for the spec.** REQ-5's amended scenarios reference the generic phrase as the typed offline `message`; the amendment drops the literal-text greppability clause from REQ-5 because the gap that made the original literal greppable is now closed by CH-04.

### D-3 — Drop `ConversationList` from `chat-app.tsx` until CH-08.2; keep the mock array elsewhere

**Decided** (user, Engram `#3870` item 3, was explore.md §10 fork C-2). `chat-app.tsx:15` loses the `ConversationList` import; `<aside data-testid="conversations-panel">` (`:89-`) and the mobile disclosure (`:151-`) are removed. `CONVERSATIONS` stays in `lib/mock/chat.ts:1-504` because `components/marketing/hero-proof/hero-proof.tsx:23` imports `HERO_OPENING` and `HERO_SCRIPT` from the same file, and `components/workspace/screens/front-desk.tsx:23` imports `CONVERSATIONS` for the workplace shell's home panel — both unaffected by CH-05.

**Why drop and not keep the rail as historical context.** The active conversation's turn is now real but the historical list is still the seeded mock array (explore.md §5). Two real events would render side-by-side: a real streaming answer in the right column and a list of scripted conversations on the left. That asymmetry reads as a bug to anyone watching the page; CH-08.2 lands `cachicamas-chat-conversation-list` and re-adds the rail against a real history source.

**Consequence for the page.** `chat-app.tsx:38` (`useSignal(CONVERSATIONS[0].id)`) loses its `selected` signal once the rail is gone — the page holds a single active conversation. `chat-app.tsx:81-82` collapses to a single agent lookup. The `?with=` deep-link at `chat-app.tsx:47-54` is preserved (D-6) but reads `agentSlug` from `staff.ts` rather than from a mock conversation.

### D-4 — No `backend/` file is touched by this change

**Decided.** Substrate preservation (NFR-TLS-003, `openspec/AGENTS.md`) is a hard rule for `backend/agent/`. CH-05.1 reads the wire CH-04 mounted (`backend/agent/src/chat/http.go:305-307`, `eventsource.go:31-48`, `cmd/chat/main.go:152-189`) but writes nothing there. The proposal explicitly carries this as an out-of-scope row in the table above so a reviewer scanning the diff cannot mistake a `go.mod` mention in a comment for a `go.mod` edit.

**Why stated explicitly.** explore.md §9.5 names the failure mode: a comment mentioning `go.mod` is read as a `go.mod` edit, a Layer-2 substrate assertion is widened, the substrate's nine-file invariant reds. The chat-package-scaffold precedent (`openspec/changes/archive/2026-08-23-cachicamas-chat-package-scaffold/proposal.md:216`) carries an identical explicit row in its affected-areas table.

### D-5 — One new hook file (`use-chat-stream.ts`) + one new spec sibling, mirroring the `FakeES` pattern verbatim

**Decided.** The hook lives at `frontend/src/components/chat/use-chat-stream.ts` (the path `chat-api.ts:9`'s design comment reserves) and ships with `use-chat-stream.spec.ts`. REQ-7 S-7.a (`frontend-chat-layer1/spec.md:84-92`) requires the colocated spec; the S-7.b grep assertion (`spec.md:91`) is satisfied by `it("REQ-1 S-1.a — submit and stream deltas", ...)`-style names.

**Test pattern.** REQ-1 / REQ-2 / REQ-4 / REQ-5 assertions on the wire client copy `FakeES` from `chat-api.spec.ts:394-425` verbatim — explore.md §8.1 names this as the only tested-once pattern this repo has. The S-7.b assertion will catch drift (any `useChatStream`-side rewrite of the parser will break `chat-api.spec.ts:60-83`'s byte-exact `single-turn.sse` assertion unless it mirrors the existing parser — explore.md §9.2).

**Why a hook and not a free-function refactor.** The `useTask$` reset on conversation selection (`chat-app.tsx:59-68`) needs a signal-store surface that a hook naturally exposes; the `Composer` props (`chat-app.tsx:211-216`) already bind to a turn machine (`useMockTurn`) and the swap is one-for-one. The hook's internal shape is owned by `design.md`, not this proposal — the proposal commits to *what* CH-05 produces, not to the internal structure of `useChatStream`.

### D-6 — Keep the `?with=` deep-link contract; preserve the active-conversation selection shape

**Decided** (was explore.md §10 fork F-1). `chat-app.tsx:47-54`'s `useVisibleTask$` that reads `?with=<slug>` survives; the parameter now resolves to an `agentSlug` lookup in `staff.ts` rather than a conversation in the mock array. The front-desk panel's `/chat?with=finance` link (`front-desk.tsx:119`) keeps working — it opens the chat with that agent. **Not** a regression in the workplace shell's wiring.

**Why not drop the parameter.** The workplace shell's navigation contract is part of the shipped product surface. Dropping the parameter would silently break `front-desk.tsx:119` and any other link the shell emits.

### D-7 — Evidence gate for the backend suite is **citation**, not re-run

**Decided** (was explore.md §10 fork H). `0005:669` requires *"both suites — the frontend suite and the backend suite"*. The frontend suite is `pnpm --filter @cachicamas/frontend test:ci` (vitest) — run on this branch's tip. The backend suite is `cd backend/agent && go test -race -count=1 ./...` — *not* re-run on this branch's tip, because CH-05 writes no Go file. The closing evidence for the backend suite is the citation of CH-04's `2b891117` green run, with wall-clock recorded in PR #194's CI log or its merge commit.

**Why citation and not re-run.** Re-running a 3-5 minute race suite for a frontend-only PR is review-budget waste the chat-package-scaffold precedent (`openspec/changes/archive/2026-08-23-cachicamas-chat-package-scaffold/proposal.md:46-49`) rejects: out-of-scope tables are read with the change author's own citation, and a re-run invites a reviewer to ask whether the race run was really uncached.

**Verification:** `git diff --stat 2b891117 HEAD -- backend/` returns empty. If it does not, this decision is void and the gate is re-run.

## Plan (CH-05.1 → CH-05.2)

### CH-05.1 — Stream a real turn into the chat page `[leaf]`

| Path | What changes |
|---|---|
| `frontend/src/components/chat/use-chat-stream.ts` | **New.** A Qwik hook. Mirrors `useMockTurn`'s shape (`use-mock-turn.ts:55-75`) but backed by `submitTurn` (`chat-api.ts:186-212`) and `subscribeTurn` (`chat-api.ts:297-413`). Carries `useVisibleTask$(({cleanup, track}) => { … cleanup(() => cancelTurn(req); unsubscribe(); }) })` for REQ-2 S-2.a (`frontend-chat-layer1/spec.md:32`). |
| `frontend/src/components/chat/use-chat-stream.spec.ts` | **New.** Colocated per REQ-7 S-7.a (`spec.md:84-92`). Copies `FakeES` from `chat-api.spec.ts:394-425` verbatim. Asserts REQ-1 S-1.a (submit + accumulate deltas + close on turn.end), REQ-2 S-2.a (cleanup issues DELETE with `keepalive: true`), REQ-4 S-4.a (typed error renders inline, no retry), REQ-5 S-5.b (pre-delta EventSource `onerror` surfaces the offline kind). Names carry the `REQ-N` identifier per S-7.b (`spec.md:91`). |
| `frontend/src/components/chat/chat-app.tsx` | **Modified.** Replace `useMockTurn` (`chat-app.tsx:17, 39`) with `useChatStream`. Drop the `useTask$` reset on selection (`chat-app.tsx:59-68`) — the active conversation is now single, not selectable. Drop `<ConversationList>` imports (`chat-app.tsx:15, 104-110, 156-164`) per D-3. Keep `<Composer>` wiring (`chat-app.tsx:211-216`) — its props still bind to a turn machine with `submit`/`cancel`/`status`. Keep `?with=` deep-link (`chat-app.tsx:47-54`) per D-6. |
| `frontend/src/routes/chat/index.spec.tsx:82-89` | **Modified.** Rewrite the "frozen browser wire is still on disk, unwired" assertion (explore.md §9.8) to assert that `chat-api.ts` exports `submitTurn`, `cancelTurn`, and `subscribeTurn` — the wire surface CH-05.1 connects to. The file-existence assertion is replaced. |

**Optional cleanup (recommended as follow-up, not in-scope here):** `chat-api.spec.ts:380, 390` carries the stale `lib/api.sse.spec.ts:75-307` comment (explore.md §8.1). The `FakeES` is now inline at `chat-api.spec.ts:394-425`. Cleaning the comment is CH-05.1's free hand but the substantive change is the inline class, which already happened.

### CH-05.2 — Amend the frozen frontend contract on the record `[mechanical]`

| Path | What changes |
|---|---|
| `openspec/changes/cachicamas-chat-frontend-wire/specs/frontend-chat-layer1/spec.md` | **New.** Mirror of `openspec/specs/frontend-chat-layer1/spec.md` (per repo convention; archive-time executor promotes the delta). Amends REQ-5's *Statement* (`spec.md:62`) and rationale (`spec.md:64`) per D-1/D-2. Amends REQ-5 S-5.a (`spec.md:68`) and S-5.b (`spec.md:69`) to assert the generic dev-honest phrase in place of the literal; the `kind:"offline"` shape survives. Adds the rationale clause mandated by `0005:673` — *"its purpose — making an unwired backend greppable — is discharged by the archetype now serving the wire"* — with the citation to CH-04 (`backend/agent/src/chat/http.go:305-307`). |
| `frontend/src/lib/chat-api.ts:107` | **Modified.** `OFFLINE_LITERAL` constant deleted. The replacement phrase lives in `offlineMessage(err)` (`chat-api.ts:109-112`), per D-2. |
| `frontend/src/lib/chat-api.ts:109-112` | **Modified.** `offlineMessage(err)` returns `"Couldn't reach the chat service. Is docker compose up? (network error)"` instead of `${OFFLINE_LITERAL} (${detail})`. The `(err.message)` detail is dropped — the phrase is browser-portable. |
| `frontend/src/lib/chat-api.ts:404, 406` | **Modified.** Both `onError(OFFLINE_LITERAL)` calls become `onError(offlineMessage(esError))` — or, since `es` does not surface a structured error, the new constant `OFFLINE_GENERIC` is exported and reused. The choice is owned by `design.md`. |
| `frontend/src/lib/chat-api.ts:27, 103, 180, 260` | **Modified.** Design comments drop the literal phrase. The `(user-locked D4)` reference becomes `(replaced by CH-05.2 — see openspec/specs/frontend-chat-layer1 REQ-5)`. |
| `frontend/src/lib/chat-types.ts:91` | **Modified.** REQ-4 + REQ-5 narrative comment drops the literal. |
| `frontend/src/lib/chat-api.spec.ts:472-488` | **Modified.** The assertion `expect(result.message).toContain("backend not wired — see PR for backend wire")` becomes `expect(result.message).toContain("docker compose up")`. The `result.kind === "offline"` check is preserved (D-1). |
| `frontend/src/lib/chat-api.spec.ts:645-664` | **Modified.** Same replacement; the `kind:"offline"` assertion is preserved. |

**Mechanical proof, `0005:673`'s second clause.** `rg "OFFLINE_LITERAL|backend not wired" frontend/` returns zero lines on the merged tree. This is part of the closing evidence, not a follow-up — the spec amendment is not "merged" until the grep is zero.

## Evidence gate

Per `0005:669` — *"this leaf's evidence gate is both suites — the frontend suite and the backend suite — because the behavior is only true across the pair."*

**Frontend suite** (run, this PR's tip):

| Command | Asserts | Recorded |
|---|---|---|
| `pnpm --filter @cachicamas/frontend test:ci` | All vitest suites green, including the new `use-chat-stream.spec.ts`. Red-green TDD (`openspec/AGENTS.md` *Strict TDD*) demands the spec file lands first and fails for the right reason before the hook makes it green | Wall-clock duration |
| `pnpm --filter @cachicamas/frontend build.types` | TypeScript clean | Exit code |
| `pnpm --filter @cachicamas/frontend lint` | ESLint clean | Exit code |

**Backend suite** (cited, not re-run — D-7): `cd backend/agent && go test -race -count=1 ./...` is green at `2b891117` per PR #194's CI evidence. CH-05 writes no Go file (`git diff --stat 2b891117 HEAD -- backend/` empty is the gate's mechanical form). If the diff is not empty, D-7 is void and the suite is re-run uncached with wall-clock recorded.

**A `(cached)` result is not evidence**, mirroring the chat-package-scaffold precedent (`openspec/changes/archive/2026-08-23-cachicamas-chat-package-scaffold/proposal.md:13`).

## Risks

| Risk | Likelihood | Mitigation |
|---|---|---|
| The page mounts an EventSource on every conversation selection but the cleanup chain does not fire, leaving orphan turns server-side | Med | explore.md §9.1 names this as the single biggest non-obvious failure mode. The `useChatStream` hook's `useVisibleTask$` carries a `cleanup` callback that issues `cancelTurn(req)` and `unsubscribe()` exactly once (REQ-2 S-2.a, `frontend-chat-layer1/spec.md:32`). D-3 (drop the rail) further reduces the surface — single active conversation, single cleanup |
| The hook rewrites the SSE parser and breaks the byte-exact `single-turn.sse` fixture at `chat-api.spec.ts:60-83` | Low | explore.md §9.2. The hook reuses `subscribeTurn` (`chat-api.ts:297-413`); only the page-side wiring changes. The S-7.b grep + the `FakeES` mirror make any drift go red |
| `keepalive: true` on DELETE is dropped in the cleanup path, breaking REQ-2 S-2.a | Low | `cancelTurn` (`chat-api.ts:229-246`) carries `keepalive: true` at line 236; the cleanup callback calls `cancelTurn(req)` directly. The existing `chat-api.spec.ts:542` assertion continues to bind the property |
| The literal "backend not wired — see PR for backend wire" survives in a doc-comment or test name the proposal did not enumerate | Low | The mechanical proof at `0005:673` is `rg "OFFLINE_LITERAL\|backend not wired" frontend/` returning zero — *any* occurrence. The grep is run on the merged tree, not on the files the proposal enumerates |
| The spec delta is merged but the wire client's `offlineMessage(err)` rewrite is forgotten, leaving a half-amended REQ-5 | Med | CH-05.2 is a single PR. The proposal's evidence-gate table binds the spec delta and the chat-api.ts rewrite to the same merge |
| A CH-05.1 PR edits `transcript-line.tsx`, `markdown.ts`, `routes/chat/layout.tsx` or `chat-api.ts:118-166`, widening scope per `0005:636` | Med | explore.md §9.4. The out-of-scope table above is the seam fence. The chat-page hook is what changes; renderer / sanitizer / guard / envelope are read-only |
| `useMockTurn` is deleted and `hero-proof.tsx` reds | Med | explore.md §9.6. `use-mock-turn.ts` survives; only its call site in `chat-app.tsx:39` is removed. `hero-proof.tsx:21, 31` continues to import it. The out-of-scope table forbids deletion |
| `ConversationList` deletion cascades — `conversation-list.tsx`'s spec file `conversation-list.spec.tsx` is removed | Low | D-3 deletes only the `chat-app.tsx` import + JSX, not the file. `conversation-list.tsx:1-59` and its spec sibling survive for CH-08.2 to consume |
| The `routes/chat/index.spec.tsx:82-89` assertion rewrite uses a weak replacement (e.g. asserts file existence of `chat-api.ts`) and CH-05.1 becomes a no-op gate | Med | D-5 binds the replacement to *exports* of `submitTurn` / `cancelTurn` / `subscribeTurn` — semantic assertions, not file-existence checks |
| The frontend-chat-layer1 spec mirror under `openspec/changes/cachicamas-chat-frontend-wire/specs/frontend-chat-layer1/spec.md` is not promoted to `openspec/specs/frontend-chat-layer1/spec.md` at archive and the delta lives only in the change folder | Low | The repo convention (`openspec/AGENTS.md` "SDD artifact layout" and the chat-package-scaffold precedent) is that archive-time promotion merges the delta into the promoted spec. The `verify-report.md` records the post-archive state |

## Out-of-scope carryover

The next reviewer should check this PR does **not** touch:

- `backend/` — any file, including `backend/agent/src/chat/*`, `backend/agent/go.{mod,sum}`, `backend/agent/Makefile`, `.golangci.yml` (D-4, NFR-TLS-003)
- `frontend/src/components/chat/transcript-line.tsx` — rendering is `0005:636` (explore.md §9.4)
- `frontend/src/lib/markdown.ts` — sanitization is `0005:636`
- `frontend/src/lib/api.ts` — the error envelope is `0005:636`
- `frontend/src/lib/chat-api.ts:118-166` (`envelopeToResult`) — frozen by REQ-4 and consumed unchanged
- `frontend/src/lib/csrf.ts`, `frontend/src/lib/require-auth-redirect.ts`, `frontend/src/lib/require-ownboarding.ts`, `frontend/src/lib/ssr-cookie-context.ts` — the route guard chain is `0005:636`
- `frontend/src/routes/chat/layout.tsx:30-34` — the canonical guard chain is `0005:636`
- `frontend/src/lib/mock/chat.ts` — survives because `hero-proof.tsx:23` and `front-desk.tsx:23` import from it (explore.md §9.6, §9.7)
- `frontend/src/lib/mock/staff.ts` — the workplace shell's mock staff is out of CH-05's scope
- `frontend/src/components/chat/use-mock-turn.ts` — survives because `hero-proof.tsx:21, 31` imports it (explore.md §9.6)
- `frontend/src/components/chat/conversation-list.tsx` — survives for CH-08.2 (D-3)
- `frontend/src/components/marketing/hero-proof/hero-proof.tsx` — explore.md §9.6
- `frontend/src/components/workspace/screens/front-desk.tsx` — explore.md §9.7
- `openspec/specs/frontend-chat-layer1/spec.md:17, 130` — the `subscribeWorkspaceSyncStream` dead citation is recorded-not-repaired (explore.md §9.9)

## Doc 0005 bookkeeping

Same-PR doc-tick plan:

| Site | Action |
|---|---|
| `0005:991` | Tick `[ ]` → `[x]` — *"The offline literal is retired by a recorded spec amendment, not a silent deletion — closed by CH-05.2"* |
| `0005:1045` | Annotate *"Register 4 — the mandated offline literal"* row → *"closed by CH-05.2"* with the PR link |
| `0005:1063` | Tick the close-by mapping `CH-05.2 → R-13, register 4` (the row already exists; only its unchecked mark changes) |
| `0005:1062` | Tick `CH-05.1 → R-12, R-16` if it is unchecked (verify in the same PR — `git diff` against `2b891117` shows whether the row is already ticked) |
| `0005:3` | Move status line to *"6 of 12"* |

No other doc 0005 edit is in scope. The close-by mappings for CH-04 (`0005:1060-1061`) are already ticked; CH-05.2's row carries no `R-13` amendment beyond the line annotation.

## Capabilities

### New capabilities

**None.** This change amends `frontend-chat-layer1` REQ-5 — a spec delta, not a new capability.

### Modified capabilities

- **`frontend-chat-layer1`** — REQ-5's *Statement* and *Rationale* amended to retire the literal phrase; S-5.a and S-5.b amended to assert the generic dev-honest phrase (D-2) in place of the literal, with the `kind:"offline"` shape preserved (D-1). No other REQ changes. The amendment cites the chat archetype's wire (`backend/agent/src/chat/http.go:305-307`) as the discharge of the greppable-gap rationale. Prefix **`F5W`** (verified free above) is used for the delta's identifiers if the delta carries any; the archive-time promotion merges them into the existing R-FCL-NNN scheme per repo convention.

## Affected areas

| Area | Impact | Description |
|---|---|---|
| `frontend/src/components/chat/use-chat-stream.ts` | **New** | The Qwik hook (D-5) |
| `frontend/src/components/chat/use-chat-stream.spec.ts` | **New** | Colocated per REQ-7 S-7.a |
| `frontend/src/components/chat/chat-app.tsx` | **Modified** | Hook swap (`:17, 39`); drop rail (`:15, 104-110, 156-164`); keep `?with=` (`:47-54`) per D-6 |
| `frontend/src/lib/chat-api.ts:107` | **Modified** | `OFFLINE_LITERAL` deleted |
| `frontend/src/lib/chat-api.ts:109-112` | **Modified** | `offlineMessage(err)` returns the generic phrase (D-2) |
| `frontend/src/lib/chat-api.ts:404, 406` | **Modified** | New offline constant or inline phrase replaces `OFFLINE_LITERAL` |
| `frontend/src/lib/chat-api.ts:27, 103, 180, 260` | **Modified** | Design comments drop the literal |
| `frontend/src/lib/chat-types.ts:91` | **Modified** | REQ-4 + REQ-5 narrative comment |
| `frontend/src/lib/chat-api.spec.ts:472-488` | **Modified** | Literal-substring check replaced (D-2) |
| `frontend/src/lib/chat-api.spec.ts:645-664` | **Modified** | Same |
| `frontend/src/routes/chat/index.spec.tsx:82-89` | **Modified** | Wire-surface assertion replaces file-existence check (explore.md §9.8) |
| `openspec/changes/cachicamas-chat-frontend-wire/specs/frontend-chat-layer1/spec.md` | **New** | Spec mirror with REQ-5 delta (per repo convention) |
| `openspec/specs/frontend-chat-layer1/spec.md` | **Modified at archive** | The delta merges into the promoted spec |
| `docs/architecture/milestones/0005-…md` | **Modified** | `:991` ticked, `:1045` annotated, `:1063` ticked, `:3` → **6 of 12** |
| `frontend/src/components/chat/transcript-line.tsx`, `lib/markdown.ts`, `lib/api.ts:118-166`, `lib/csrf.ts`, `lib/require-auth-redirect.ts`, `lib/require-ownboarding.ts`, `lib/ssr-cookie-context.ts`, `routes/chat/layout.tsx` | **Unchanged** | `0005:636` seam fence |
| `frontend/src/lib/mock/chat.ts`, `frontend/src/lib/mock/staff.ts`, `frontend/src/components/chat/use-mock-turn.ts`, `frontend/src/components/chat/conversation-list.tsx` | **Unchanged** | Survive for `hero-proof`, `front-desk`, CH-08.2 (explore.md §9.6, §9.7, D-3) |
| `frontend/src/components/marketing/hero-proof/hero-proof.tsx`, `frontend/src/components/workspace/screens/front-desk.tsx` | **Unchanged** | explore.md §9.6, §9.7 |
| `backend/**` | **Unchanged** | D-4 — substrate preservation |

## Rollback plan

**Before merge:** discard the branch. The worktree touches no shared state.

**After merge, if a hook assertion or spec delta is wrong:** revert the single PR. The change adds two new files (`use-chat-stream.ts` + `use-chat-stream.spec.ts`) and a delta spec mirror; modifies six files in `frontend/src/` (`chat-app.tsx`, `chat-api.ts`, `chat-types.ts`, `chat-api.spec.ts`, `index.spec.tsx`); amends one promoted spec at archive; and edits four lines of doc 0005. It touches no `backend/` file, no `go.mod`/`go.sum`/`Makefile`/`.golangci.yml`, no schema, no wire, no configuration, and no run-time behaviour outside the chat page itself — the revert is complete by construction and needs no migration or data step.

**Partial rollback is available and preferable if only one node is wrong.** CH-05.1 and CH-05.2 are separate nodes with separate closing conditions. Reverting CH-05.2 alone leaves the wire client hooked up but the spec amendment un-merged; the chat page works against the served surface but REQ-5 still mandates the literal phrase, which falsifies on the next PR that re-introduces the constant.

**After CH-06 merges, revert is unavailable.** The correction path is then an amending SDD change against `frontend-chat-layer1` (spec) and `chat-api.ts` (code), recording what changed and why.

## Dependencies

- **CH-04 merged** — PR #194, `2b891117`. `backend/agent/src/chat/http.go:305-307` mounts the wire; `eventsource.go:31-48` seals the SSE event names; `cmd/chat/main.go:152-189` brings the binary up under `docker compose`. The citation is the source of D-7.
- **AG-23 merged**, Layer 2 complete; `frontend-chat-layer1` frozen; `chat-archetype-contract` R-CHT-009 promoted.
- Read-only: `frontend-chat-layer1` spec, `chat-archetype-contract` spec, ADR 0009, `frontend/src/lib/api.ts` envelope pattern.
- **There is no `openspec` CLI, no `.github/`, and no CI in this repo.** Every gate is human-run.

## Success criteria

| # | Criterion | Evidence at verify |
|---|---|---|
| 1 | The chat page's `useChatStream` hook streams a submitted prompt into the conversation and the offline literal appears nowhere on the page | `pnpm test:ci` red-green record; manual smoke against `docker compose up` |
| 2 | Stopping a turn from the page issues `DELETE` via `cancelTurn` and `EventSource.close()` runs exactly once; the partial assistant text remains | REQ-2 S-2.a/S-2.b assertions in `use-chat-stream.spec.ts` |
| 3 | A backend `event: error` of `kind:"server"` renders inline; no retry is scheduled by the page; the input accepts a new prompt | REQ-4 S-4.a assertion in `use-chat-stream.spec.ts` |
| 4 | `chat-api.ts:107` no longer exists; `OFFLINE_LITERAL` is deleted; the literal phrase is unreachable in the merged tree | `rg "OFFLINE_LITERAL" frontend/` returns zero; `rg "backend not wired" frontend/ openspec/specs/` returns zero |
| 5 | The REQ-5 amendment cites CH-04 as discharging the greppable-gap rationale, per `0005:673` | `specs/frontend-chat-layer1/spec.md` *Rationale* carries the citation; reviewed by the proposal's own author and one other |
| 6 | `kind:"offline"` survives in the `ApiResult<T>` discriminated union and in `chat-api.spec.ts` assertions | `rg "kind.*offline" frontend/src/lib/` returns the expected set; no test fires on `kind:"offline"` having changed |
| 7 | `useMockTurn` survives and `hero-proof.tsx` still imports it | `grep "useMockTurn" frontend/src/components/marketing/hero-proof/` returns hits; `pnpm test:ci` green on `use-mock-turn.spec.ts` |
| 8 | No `backend/` file is touched | `git diff --stat 2b891117 HEAD -- backend/` empty |
| 9 | No spec-level requirement of `frontend-chat-layer1` other than REQ-5 changes | `diff <(git show origin/main:openspec/specs/frontend-chat-layer1/spec.md) <(git show HEAD:openspec/specs/frontend-chat-layer1/spec.md)` shows REQ-5 lines touched, no other lines |
| 10 | The `routes/chat/index.spec.tsx:82-89` assertion is rewritten to assert the wire surface, not file existence | `index.spec.tsx` carries the new assertion; the file-existence check is gone |
| 11 | Frontend suite green, uncached | `pnpm --filter @cachicamas/frontend test:ci` wall-clock recorded |
| 12 | Backend suite cited, not re-run, per D-7 | PR description cites the `2b891117` green run; `git diff --stat 2b891117 HEAD -- backend/` empty |
| 13 | Bookkeeping | `0005:991` ticked; `0005:1045` annotated; `0005:1063` ticked; `0005:3` reads **6 of 12** |

## Review Workload Forecast

| Component | Est. counted lines |
|---|---|
| `components/chat/use-chat-stream.ts` | 110 |
| `components/chat/use-chat-stream.spec.ts` | 175 |
| `components/chat/chat-app.tsx` (modifications) | 35 |
| `lib/chat-api.ts` (offline literal retirement + comments) | 18 |
| `lib/chat-api.spec.ts` (assertion rewrites) | 12 |
| `lib/chat-types.ts` (comment only) | 2 |
| `routes/chat/index.spec.tsx` (assertion rewrite) | 8 |
| `openspec/changes/.../specs/frontend-chat-layer1/spec.md` (REQ-5 delta) | 35 |
| SDD artifacts (design, tasks, apply-progress, verify-report) | 520 |
| doc 0005 bookkeeping | 4 |
| **Total** | **~920** |

The overage over the 1000-line cap is comfortable; the 400-line Go-style default is meaningless here (no Go is touched). `size:exception` is **not** required.

```
Decision needed before apply: No
Chained PRs recommended: No
400-line budget risk: N/A
```

## Proposal question round (auto mode — recorded, not asked)

Execution mode is `auto`; the three blocking product forks were already answered by the user (Engram `#3870`). These three remaining forks were decided from the charter rather than asked, and are stated so a reviewer can overturn each deliberately:

1. **Where does the spec delta live — in the change folder's mirror, or directly in `openspec/specs/frontend-chat-layer1/spec.md`?** Decided **change-folder mirror** (per repo convention; archive-time promotion merges the delta into the promoted spec). If overturned, the delta is committed directly to `openspec/specs/` and loses the change-folder audit trail. The chat-package-scaffold precedent (`openspec/changes/archive/2026-08-23-cachicamas-chat-package-scaffold/proposal.md:189-191`) follows the mirror shape.
2. **Is the optional `chat-api.spec.ts:380, 390` comment cleanup (the dead `api.sse.spec.ts:75-307` reference) in-scope?** Decided **out of scope, follow-up** (recommended in § CH-05.1 above). If included, two comment lines change and the spec consistency improves, but the substantive change is the inline `FakeES` class which already happened (explore.md §8.1).
3. **Does the new hook file carry a `systemHint` UI control?** Decided **no** (D, explore.md §10 fork E). The minimum-work shape is to omit the field. If a UI control is desired, it lands in CH-05.1 as a follow-up commit on top of the same PR — the wire's `systemHint` reservation at `chat-types.ts:71-72` already accepts it.