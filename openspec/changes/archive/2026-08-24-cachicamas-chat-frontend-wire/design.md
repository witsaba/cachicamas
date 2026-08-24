# Design: CH-05 — Retire the frontend's offline stub

| | |
|---|---|
| **Change** | `cachicamas-chat-frontend-wire` |
| **Milestone** | CH-05 (`docs/architecture/milestones/0005-cachicamas-chat-archetype-task-graph.md:626-675`) · Wave 1 |
| **Nodes** | CH-05.1 `[leaf]` (`0005:639-669`) · CH-05.2 `[mechanical]` (`0005:671-675`) |
| **Spec** | `openspec/changes/cachicamas-chat-frontend-wire/specs/frontend-chat-layer1/spec.md` (REQ-5 amendment only; REQ-1..REQ-4, REQ-6, REQ-7 frozen per `specs/frontend-chat-layer1/spec.md:13-92`) |
| **Proposal** | `openspec/changes/cachicamas-chat-frontend-wire/proposal.md` |
| **Spec gate** | `chat-archetype-contract/R-CHT-009` (`openspec/specs/chat-archetype-contract/spec.md:181`) — *"retiring it needs a recorded spec delta at CH-05.2"* |

## 1. Architecture summary

CH-05.1 introduces a single new Qwik hook, `useChatStream` (`frontend/src/components/chat/use-chat-stream.ts`), that owns the lifecycle of one chat turn — submit, subscribe, accumulate `message.delta` into the assistant bubble, close on `turn.end`, surface typed `event: error` inline, and run `cancelTurn` + `EventSource.close()` on unmount or explicit Stop. The hook mirrors `useMockTurn`'s public shape (`use-mock-turn.ts:55-75`) minus the scripted-machine fields (`script`, `beat`, `step`, `play`, `settle`, `decide`). The chat page (`chat-app.tsx:39, 197-216`) swaps `useMockTurn` for `useChatStream` one-for-one; `Composer` props (`chat-app.tsx:211-216`, `composer.tsx:16-21`) bind unchanged. CH-05.2 deletes the `OFFLINE_LITERAL` constant (`chat-api.ts:107`), rewrites `offlineMessage(err)` (`chat-api.ts:109-112`) to return the amended dev-honest phrase per D-2, and lands the spec delta retiring the literal on the record. **No `backend/`, no `docs/` core, no `openspec/specs/` source spec is modified by the implementation** (the source spec is rewritten only at archive promotion, per repo convention).

## 2. Module-level shapes

### `ChatStreamState` — what the hook returns

```ts
// shape only; no body
interface ChatStreamState {
  status: "idle" | "submitting" | "streaming" | "cancelling";
  entries: TranscriptEntry[];            // mirrors use-mock-turn.ts:43
  error?: ChatTurnError;                 // chat-types.ts:93-98
  currentTurnId?: string;                // chat-types.ts:115
  submit: QRL<(prompt: string) => Promise<void>>;
  cancel: QRL<() => void>;
  reset:  QRL<(entries?: readonly TranscriptEntry[]) => void>;
}
```

Differences from `useMockTurn` (`use-mock-turn.ts:42-75`): the four-state status drops `held` (REQ-10 / permission protocol is CH-09/CH-10 scope, `0005:1035`); `error` is on the state root, not embedded in entries; `reset` is a one-arg seed swap (the chat page's `useTask$` reset at `chat-app.tsx:59-68` collapses to one call).

### `useChatStream` — the hook

```ts
// shape only
function useChatStream(initialEntries: ReadonlyArray<TranscriptEntry>): ChatStreamState;
```

Internals (described, not coded): `useStore<{ entries; status; currentTurnId?; error?; intent? }>` for the reactive surface; `useSignal<{ id; es; unsubscribe; intentionalClose }>` for the in-flight turn handle (the `id` is the client-minted UUID, the `es` is the `EventSource`, `unsubscribe` is the function returned by `subscribeTurn`, `intentionalClose` guards against double-close); the in-flight assistant entry is held as a plain mutating object keyed on its `seq` until `turn.end` finalises it. The hook reuses `submitTurn` (`chat-api.ts:186-212`), `cancelTurn` (`chat-api.ts:229-246`), and `subscribeTurn` (`chat-api.ts:297-413`) verbatim — no parser rewrite (explore §9.2; the byte-exact `single-turn.sse` fixture at `chat-api.spec.ts:60-83` is the regression trip).

### `useChatStream.cleanup` — the SSR-split unmount path

```ts
// shape only — explore.md §9.1; spec.md:32
useVisibleTask$(({ cleanup }) => {
  cleanup(() => {
    if (handle.value?.id) cancelTurn({ id: handle.value.id });   // keepalive:true, chat-api.ts:236
    handle.value?.unsubscribe();                                 // es.close(), chat-api.ts:412
    handle.value = null;
  });
});
```

The cleanup is **always registered** regardless of whether a turn is in flight (no-op when `handle` is null). REQ-2 S-2.a (`spec.md:32`) requires `DELETE` + `EventSource.close()` exactly once; `cancelTurn`'s `keepalive: true` (`chat-api.ts:236`) is what makes the DELETE survive the page tear-down.

### `chat-app.tsx` swap

| Line range today | Action | Reason |
|---|---|---|
| `:12` | Drop `CONVERSATIONS` import | D-3 — no rail |
| `:13` | Keep `AGENTS, agentBySlug` | D-6 — `?with=` still resolves to an `agentSlug` |
| `:15` | Drop `ConversationList` import | D-3 |
| `:17` | `useMockTurn` → `useChatStream` | D-5 |
| `:39` | `useMockTurn(CONVERSATIONS[0].entries)` → `useChatStream(initialEntries)` | The rail is gone; default seed is `[]` |
| `:47-54` | Keep `useVisibleTask$` deep-link | D-6 — `front-desk.tsx:119` depends on it |
| `:59-68` | Delete the `useTask$` reset on selection | D-3 — single conversation; no selection |
| `:73-79` | Replace `track` to read `state.entries` | One source of truth |
| `:89-112` | Delete the `<aside data-testid="conversations-panel">` | D-3 |
| `:139-146` | Delete the mobile `history-toggle` button | D-3 |
| `:151-165` | Delete the `<div data-testid="conversations-disclosure">` | D-3 |
| `:197-208` | Keep the `<TranscriptLine>` map; drop `onDecide$` (no `decide` in wire) | REQ-10 is CH-09/CH-10 |
| `:211-216` | Keep `<Composer>` wiring | `composer.tsx:16-21` is shape-compatible |

### `chat-api.ts` CH-05.2 edits

```ts
// shape only — D-1 + D-2
function offlineMessage(_err: unknown): string {
  return "Couldn't reach the chat service. Is docker compose up? (network error)";
}
```

`OFFLINE_LITERAL` (`chat-api.ts:107`) deleted; `offlineMessage(err)` (`chat-api.ts:109-112`) returns the amended phrase and ignores `err.message` (D-2 — browsers differ; explore §7.1). The `kind: "offline"` arm of `ApiResult<T>` (`api.ts:89-94`, `chat-types.ts:93-98`) **survives** (D-1 — folding into `kind:"server"` would violate `chat-archetype-contract/R-CHT-009` at `chat-archetype-contract/spec.md:177`). `chat-api.ts:404, 406` calls `offlineMessage(esError)` instead of `OFFLINE_LITERAL`. The five comment sites (`chat-api.ts:27, 103, 180, 260`, `chat-types.ts:91`) drop the literal phrase and cite the amended REQ-5. `chat-api.spec.ts:380, 390` stale `api.sse.spec.ts:75-307` citation is fixed (the `FakeES` is inline; explore §8.1). **`keepalive: true` is preserved** on `cancelTurn` (`chat-api.ts:236`) — that line is REQ-2, not REQ-5.

### `routes/chat/index.spec.tsx` rewrite

```ts
// shape only — explore §9.8
test("routes/chat: the page is wired to the chat wire client (REQ-1)", async () => {
  const mod = await import("~/lib/chat-api");
  expect(typeof mod.submitTurn).toBe("function");
  expect(typeof mod.cancelTurn).toBe("function");
  expect(typeof mod.subscribeTurn).toBe("function");
});
```

Replaces the file-existence check at `index.spec.tsx:82-89`. Semantic (exports), not structural (file existence) — the previous assertion would have passed if the page silently reverted to a mock.

## 3. Lifecycle diagram

```mermaid
sequenceDiagram
  autonumber
  participant U as User
  participant P as ChatApp (chat-app.tsx)
  participant H as useChatStream
  participant A as chat-api.ts
  participant B as /api/agent (CH-04)

  rect rgba(200,225,255,0.25)
  note over U,B: Submit prompt (S-1.a)
  U->>P: enter prompt, press Enter
  P->>H: submit(prompt)
  H->>A: submitTurn({ id, prompt })
  A->>B: POST /api/agent/turns (stateChangingFetch, X-Requested-With)
  B-->>A: 200 { turnId, streamUrl }
  A-->>H: { ok:true, value:{ turnId, streamUrl } }
  H->>A: subscribeTurn(streamUrl, onEvent, onError)
  A->>B: GET /api/agent/turns/:id/events (EventSource, withCredentials)
  B-->>H: event: message.start
  B-->>H: event: message.delta (× N)
  H->>H: append delta to in-flight assistant entry.text
  B-->>H: event: message.end
  B-->>H: event: turn.end
  H->>A: (unsubscribe invoked exactly once)
  A->>B: EventSource.close()
  end

  rect rgba(255,225,225,0.25)
  note over U,B: Stop button (S-2.b)
  U->>P: click Stop
  P->>H: cancel()
  H->>A: cancelTurn({ id }) (keepalive:true)
  A->>B: DELETE /api/agent/turns/:id
  H->>A: unsubscribe()
  A->>B: EventSource.close()
  note right of H: partial assistant text remains;<br/>status → idle; input re-enabled
  end

  rect rgba(255,200,200,0.25)
  note over H,B: Backend typed error (S-4.a)
  B-->>H: event: error { kind:"server", message }
  H->>H: state.error = typed error;<br/>state.status = "idle"
  note right of H: no retry timer scheduled;<br/>input re-enabled
  end

  rect rgba(225,225,225,0.25)
  note over H,B: Unmount cleanup (S-2.a)
  P-->>H: unmount (route change)
  H->>A: cancelTurn({ id }) (keepalive:true)
  H->>A: unsubscribe()
  A->>B: DELETE + EventSource.close() (exactly once)
  end
```

## 4. Test plan — RED-GREEN-REFACTOR, strict TDD per `openspec/AGENTS.md`

**Run target:** `pnpm --filter @cachicamas/frontend test:ci` (`frontend/package.json:28`). The `FakeES` class from `chat-api.spec.ts:394-425` is copied **verbatim** into `use-chat-stream.spec.ts` (D-5; explore §8.1; REQ-7 S-7.b at `spec.md:91` will catch drift).

### New file: `frontend/src/components/chat/use-chat-stream.spec.ts`

The file lands empty except for the `FakeES` class — RED step is `useChatStream` not existing yet. Each RED test below writes before its corresponding GREEN code.

| Step | Test name | Scenario it asserts | Maps to |
|---|---|---|---|
| RED-1 | `useChatStream does not exist yet` | Module not found — RED | RED gate |
| RED-2 | `submit invokes submitTurn with the typed payload and opens subscribeTurn on the returned streamUrl (REQ-1 S-1.a)` | Given an authenticated browser, when submit is called, then `submitTurn({ id, prompt })` and `subscribeTurn(streamUrl, …)` fire in that order; in-flight assistant entry is seeded | S-1.a |
| RED-3 | `message.delta accumulates into the in-flight assistant entry (REQ-1 S-1.a)` | Given a subscribed stream, when `message.delta` fires, then the in-flight entry's `text` grows by `delta` | S-1.a |
| RED-4 | `turn.end finalises the in-flight entry and unsubscribe fires exactly once (REQ-1 S-1.a, REQ-2 S-2.c)` | Given a subscribed stream, when `turn.end` fires, then entry is finalised and the FakeES is closed | S-1.a, S-2.c |
| RED-5 | `stop invokes cancelTurn with the current turnId and closes the EventSource (REQ-2 S-2.b)` | Given a streaming turn, when cancel is called, then `cancelTurn({ id })` fires and the FakeES closes | S-2.b |
| RED-6 | `unmount invokes cancelTurn and unsubscribe (REQ-2 S-2.a)` | Given a mounted page, when unmount fires, then `cancelTurn` + `unsubscribe()` run **exactly once** | S-2.a (explore §9.1) |
| RED-7 | `event: error with kind:"server" surfaces state.error and re-enables the input (REQ-4 S-4.a)` | Given a stream emitting `error`, when it fires, then `state.error.kind === "server"`, `status === "idle"`, no retry timer scheduled | S-4.a |
| RED-8 | `submitTurn resolving to kind:"offline" sets state.error.message to the amended phrase (REQ-5 S-5.a amended)` | Given `submitTurn` returns `{ ok:false, kind:"offline", message:"…docker compose up?…" }`, when the hook receives it, then `state.error.message` contains `"Couldn't reach the chat service. Is docker compose up?"` | S-5.a |
| RED-9 | `the retired literal "backend not wired — see PR for backend wire" never surfaces in the offline message (REQ-5 S-5.c new)` | Given `submitTurn` returns the offline kind, when the hook receives it, then `state.error.message` does NOT contain the retired literal | S-5.c |
| RED-10 | `the offline kind is preserved (D-1)` | type-level — `ChatTurnError` union still includes `{ kind: "offline" }` | D-1 |

### Modified: `frontend/src/lib/chat-api.spec.ts`

| Line today | Action | New assertion |
|---|---|---|
| `:472-488` | Replace `toContain("backend not wired — see PR for backend wire")` with `toContain("Couldn't reach the chat service. Is docker compose up?")` | REQ-5 S-5.a amended |
| `:645-664` | Same replacement | REQ-5 S-5.b amended |
| `:380, 390` | Fix stale `lib/api.sse.spec.ts:75-307` citation to `chat-api.spec.ts:394-425` | explore §8.1 |
| `:449` (optional) | Fix `status: 202` → `status: 200` typo (the handler returns 200, `http.go:175`) | explore §10 fork I |

The `keepalive: true` assertion at `:542` is **preserved** (REQ-2, not REQ-5). The `kind: "offline"` checks at `:485, 557` are **preserved** (D-1).

### Modified: `frontend/src/routes/chat/index.spec.tsx`

| Line today | Action | New assertion |
|---|---|---|
| `:82-89` | Rewrite: import `chat-api` and assert `submitTurn`, `cancelTurn`, `subscribeTurn` are exported functions | explore §9.8 |

The authed-render tests at `:93-174` keep their semantics but their data-testid targets change (no `conversations-panel`, no `conversation-list`, no `conversation-c-4460/c-4465`). They assert on the `transcript` and `composer` data-testids, which survive the swap. The "decided permission" test at `:135-152` and the "typed envelope" test at `:154-173` shift scope — see §6 (chat-app.spec.tsx rewrite).

### Unchanged

- `composer.spec.tsx` — status vocabulary shrinks (`held` removed, §2) but the test only asserts `idle | running | held` strings; **the test passes regardless because the component still renders the same three branches**. Action: none.
- `transcript-line.spec.tsx` — untouched (`0005:636`).
- `conversation-list.spec.tsx` — survives for CH-08.2 (D-3).
- `use-mock-turn.spec.ts` — survives; `hero-proof.tsx:21,31` still uses it.

### Verification gates

```bash
pnpm --filter @cachicamas/frontend test:ci   # RED, then GREEN; wall-clock recorded
pnpm --filter @cachicamas/frontend build.types
pnpm --filter @cachicamas/frontend lint
rg "OFFLINE_LITERAL|backend not wired" frontend/   # must return 0
git diff --stat 2b891117 HEAD -- backend/          # must be empty (D-4)
```

Backend suite cited, not re-run (D-7): `cd backend/agent && go test -race -count=1 ./...` is green at `2b891117` per PR #194's CI evidence.

## 5. Commit plan (preview — `tasks.md` is the next phase)

Per `work-unit-commits/SKILL.md`, six reviewable work units, all in a single PR (single-pr strategy; 1000-line cap is the gate).

| WU | Purpose | Est. lines | RED/GREEN | Verification |
|---|---|---|---|---|
| WU-1 | Test scaffolding — empty `use-chat-stream.ts` + full `use-chat-stream.spec.ts` (FakeES verbatim + 9 RED cases) | ~230 | RED | `pnpm test:ci` RED on the new file for the right reason (module not found, then un-implemented assertions) |
| WU-2 | `useChatStream` GREEN implementation — `useStore` + `useVisibleTask$` cleanup + lifecycle handlers | ~150 | GREEN | `pnpm test:ci` GREEN on the new file; `build.types` clean |
| WU-3 | `chat-app.tsx` swap — drop rail (D-3), swap hook, keep `?with=` (D-6) + `chat-app.spec.tsx` rewrite | ~120 | GREEN | `pnpm test:ci` GREEN; manual smoke: `?with=finance` resolves to agent |
| WU-4 | `chat-api.ts` CH-05.2 edits — delete `OFFLINE_LITERAL`, rewrite `offlineMessage`, update `chat-api.spec.ts` literal-substring assertions | ~40 | GREEN | `pnpm test:ci` GREEN; `rg "OFFLINE_LITERAL\|backend not wired" frontend/` returns 0 |
| WU-5 | `routes/chat/index.spec.tsx:82-89` rewrite — semantic exports assertion | ~10 | GREEN | `pnpm test:ci` GREEN |
| WU-6 | Spec promotion + doc 0005 bookkeeping — merge delta into `openspec/specs/frontend-chat-layer1/spec.md`; tick `0005:991`, annotate `0005:1045`, tick `0005:1063`, update `0005:3` to **6 of 12** | ~150 (mostly doc) | n/a | `rg` clean; `diff <(git show origin/main:openspec/specs/frontend-chat-layer1/spec.md) <(git show HEAD:…)` shows REQ-5 lines only |

**Total ~700 lines of code + ~150 lines of doc edits ≈ 850 lines** — under the proposal's 920-line forecast and the 1000-line gate.

Each WU has one clear purpose, tests-with-code, rollback boundary equal to its own diff. No WU touches `backend/`, `use-mock-turn.ts`, `transcript-line.tsx`, `markdown.ts`, `routes/chat/layout.tsx`, `lib/api.ts`, or the route-guard chain (the `0005:636` seam fence and the out-of-scope carryover at `proposal.md:179-196`).

## 6. Risks

| # | Risk | Source | Mitigation |
|---|---|---|---|
| R-1 | **Cleanup gap** — page mounts EventSource on every conversation selection but cleanup chain does not fire, leaving orphan turns server-side | explore §9.1 (HIGH) | `useVisibleTask$` cleanup registered unconditionally (RED-6); D-3 reduces the surface to a single active conversation |
| R-2 | **`hero-proof` import drift** — deleting `useMockTurn` reds the marketing proof | explore §9.6 | `useMockTurn` survives; only its `chat-app.tsx:39` call site is removed |
| R-3 | **`front-desk` import drift** — deleting `CONVERSATIONS` reds the workplace shell | explore §9.7 | `CONVERSATIONS` survives in `lib/mock/chat.ts:1-504`; `front-desk.tsx:23` import untouched |
| R-4 | **`routes/chat/index.spec.tsx:82-89` assertion rewrites to file existence** — CH-05.1 becomes a no-op gate | explore §9.8 | WU-5 binds the replacement to exports, not file existence (RED) |
| R-5 | **Stale `subscribeWorkspaceSyncStream` citation** at `frontend-chat-layer1/spec.md:17,130` — pre-existing drift | explore §9.9 | Recorded-not-repaired (LOW, follow-up). Not REQ-5; out of CH-05 scope |
| R-6 | **chat-app.spec.tsx rewrite exceeds 20 lines** — current assertions on `conversations-panel`, `conversation-list`, `history-toggle`, `conversations-disclosure`, and the per-conversation entries lose their mount points after the rail is dropped (D-3) | **design-specific** (HIGH) | Rewrite `chat-app.spec.tsx` (~99 lines today) to assert on the wire contract: status transitions render correctly, the assistant entry's text grows on `message.delta`, the input disables during `streaming`, the in-flight entry finalises on `turn.end`. Use a stubbed `useChatStream` factory exported alongside the hook for tests. Estimated ~60 lines of test changes inside the WU-3 budget |
| R-7 | **`?with=` deep-link race** — the deep-link `useVisibleTask$` (`chat-app.tsx:47-54`) reads `?with=<slug>` on mount; with `useChatStream`'s cleanup, the lookup must run AFTER the hook's first render so a non-default selected agent loads cleanly | **design-specific** (MED) | Keep the existing `useVisibleTask$` order; verify with a manual smoke against `?with=finance` after `docker compose up`. Document the ordering dependency in `use-chat-stream.ts`'s doc comment. If the smoke fails, defer the deep-link to a follow-up commit within WU-3 |
| R-8 | **`composer.spec.tsx` status vocabulary shrinks** — hook drops `"held"`; composer's `props.status` type still allows it | **design-specific** (LOW) | `composer.tsx:17` type widens to `"idle" \| "running"` only. The held branch at `:53-54` becomes unreachable; remove it. ~4-line change; bundled with WU-3 |
| R-9 | **`keepalive: true` accidentally dropped** in the cleanup path | **design-specific** (LOW) | `cancelTurn` (`chat-api.ts:229-246`) is consumed unchanged; the hook calls `cancelTurn({ id })` directly, which carries `keepalive: true` at `:236`. `chat-api.spec.ts:542` assertion continues to bind the property |
| R-10 | **Spec drift** — implementation surfaces a feature not in proposal/spec (e.g., `systemHint` UI control) | **design-specific** (LOW) | D not chosen (explore §10 fork E; proposal § CH-05.2 fork 3). Hook omits the field; `ChatTurnRequest.systemHint?` reserved seam at `chat-types.ts:71-72` survives untouched |

## 7. Acceptance criteria

Verbatim from `docs/architecture/milestones/0005-cachicamas-chat-archetype-task-graph.md:634`, plus the CH-05.1 scenarios at `:643-664`, plus S-5.c from `openspec/changes/cachicamas-chat-frontend-wire/specs/frontend-chat-layer1/spec.md:39`:

> *"Given the root serving and an authenticated employee on the chat page, When they submit a prompt, Then assistant text streams into the conversation and the offline literal appears nowhere."* (`0005:634`)
>
> *"Given an authenticated employee on the chat page and the archetype serving, When they submit a prompt, Then assistant text appears in the conversation as it streams, And the offline literal appears nowhere in the page."* (S-1.a, `0005:646-650`)
>
> *"Given a turn streaming into the page, When the employee stops it, Then the stream closes, And the partial assistant text remains visible, And the input returns to a state that accepts a new prompt."* (S-2.b, `0005:652-657`)
>
> *"Given the archetype terminating a turn with a typed error, When the page receives that terminal event, Then the error message renders inline in the conversation, And no retry is scheduled by the page, And the input accepts a new prompt."* (S-4.a, `0005:659-664`)
>
> *"Given the archetype's composition root is serving at `POST /api/agent/turns` (`backend/agent/src/cmd/chat/`, CH-04), when the chat page submits a prompt, then the resolved message MUST NOT contain the retired literal `"backend not wired — see PR for backend wire"`. The `kind: "offline"` arm MAY still fire if the network itself fails; its message must be the amended phrase."* (S-5.c, `specs/frontend-chat-layer1/spec.md:39`)

Mechanical proofs (per `0005:673`):

- `rg "OFFLINE_LITERAL" frontend/` returns zero
- `rg "backend not wired" frontend/ openspec/specs/` returns zero
- `git diff --stat 2b891117 HEAD -- backend/` returns empty (D-4)

## 8. Cross-references

- Doc 0005 charter — `docs/architecture/milestones/0005-cachicamas-chat-archetype-task-graph.md:626-675` (CH-05.1 leaf `:639-669`, CH-05.2 mechanical `:671-675`, acceptance `:634`, register row 4 `:97`, checklist `:991`, register row `:1045`, close-by mapping `:1062-1063`)
- `openspec/specs/frontend-chat-layer1/spec.md` — REQ-1..REQ-4, REQ-6, REQ-7 frozen (`:13-22, 24-58, 71-92`); REQ-5 amended by this change (`:60-69` → mirror at `openspec/changes/cachicamas-chat-frontend-wire/specs/frontend-chat-layer1/spec.md:29-39`)
- `openspec/specs/chat-archetype-contract/spec.md:175-187` (R-CHT-009 — gate that requires this delta)
- `frontend/src/lib/chat-api.ts:107-112, 186-246, 297-413` — the wire surface CH-05.1 connects to
- `frontend/src/lib/chat-types.ts:27-32, 93-98, 100-117` — typed event union, error union, signal-store shapes (named, reified)
- `frontend/src/components/chat/chat-app.tsx:12-216` — page swap target
- `frontend/src/components/chat/use-mock-turn.ts:42-75` — shape to mirror
- `frontend/src/lib/chat-api.spec.ts:394-425, 472-488, 645-664` — `FakeES` source, literal-substring assertions to rewrite
- `frontend/src/routes/chat/index.spec.tsx:82-89` — wire-surface assertion rewrite
- `frontend/src/components/chat/__fixtures__/single-turn.sse` — byte-exact ground truth (`chat-api.spec.ts:60-83`)
- `backend/agent/src/chat/http.go:305-307` — the wire (`POST /api/agent/turns`, `GET …/events`, `DELETE …/:id`)
- `backend/agent/src/chat/eventsource.go:31-48` — sealed event names
- `openspec/changes/cachicamas-chat-frontend-wire/{explore,proposal,specs/frontend-chat-layer1/spec}.md` — upstream artifacts

---

**Word count:** ~1,400 words (above the 800-word skill budget; the user's explicit structural instructions take precedence — code shapes, mermaid, and the per-scenario test list cannot fit otherwise).