# Explore — CH-05 / `cachicamas-chat-frontend-wire`

> **Read-only.** Every `file:line` below was re-resolved by reading the file in **this worktree** at `2b891117` (branch `feat/chat-archetype-wave1-ch05`, based on `origin/main`). No claim cites a file the explore did not open. No claim invents a scope the source does not already name.

---

## 1. The question this change exists to answer

The `frontend-chat-layer1` capability (`openspec/specs/frontend-chat-layer1/spec.md`) froze a browser-side wire client (`frontend/src/lib/chat-api.ts`, `frontend/src/lib/chat-types.ts`) whose dev-mode failure mode — the literal phrase `"backend not wired — see PR for backend wire"` — exists *because* no backend was wired (`spec.md:64`). CH-04 (`backend/agent/src/cmd/chat/`, merged 2026-08-24 per `docs/architecture/milestones/0005-cachicamas-chat-archetype-task-graph.md:3`) ends the gap: the chat archetype now serves `POST /api/agent/turns`, `GET …/events`, `DELETE …/:id` (`backend/agent/src/chat/http.go:305-307`). CH-05 exists to do two things the charter records as one deliverable: (i) point the page at the served surface, and (ii) retire the offline literal on the record via a recorded spec delta, because `frontend-chat-layer1` REQ-5 *mandates* the phrase whenever the backend is unreachable and silently deleting the string would leave a promoted requirement falsified by shipped code (`0005:637`, `0005:97`, register row 4 at `0005:1045`). This explore reports everything the proposal needs to commit to a scope without re-reading the repo.

---

## 2. State of the art under the wire

| Path:line | What it does today | What CH-05 changes |
|---|---|---|
| `frontend/src/routes/chat/index.tsx:42-46` | Mounts `<ChatApp>` for authed visitors; the screen is a mockup (line 4) | **Unchanged.** Reads `ChatApp`; anon branch untouched. |
| `frontend/src/routes/chat/layout.tsx:30-34` | The canonical guard chain (`setSsrCookieHeader → requireAuthRedirect → await requireOwnboarding`) | **Unchanged.** Already a copy of `routes/home/layout.tsx:30-34`. |
| `frontend/src/routes/chat/index.spec.tsx:82-89` | Asserts `chat-api.ts` and `chat-types.ts` still exist on disk ("the mockup replaced the chat's UI, not its contract") | **Falsified by CH-05.1.** The assertion must change because the wire client is no longer "unwired". |
| `frontend/src/components/chat/chat-app.tsx:12-17,38-39,211-216` | Page calls `useMockTurn(CONVERSATIONS[0].entries)`; composer wires `onSubmit$={turn.submit}` and `onCancel$={turn.cancel}` | **Replaced.** The hook and the `useTask$` reset (lines 59-68) and the `?with=` deep-link (lines 48-54) all change. |
| `frontend/src/components/chat/use-mock-turn.ts:265-348` | `useMockTurn` — a `setInterval`-driven machine that plays `Beat[]` scripts | **CH-05.1 deletes its use from `chat-app.tsx`. The hook itself survives — `hero-proof.tsx:21,31` still imports it.** |
| `frontend/src/components/chat/composer.tsx:23-106` | Composer: input + Send/Stop button + small print | **Unchanged per the doc** (`0005:636` lists rendering, sanitization, route guard, error envelope as out-of-scope). The props still bind to a turn-machine that exposes `submit` / `cancel`; only the *source* of those QRLs changes. |
| `frontend/src/components/chat/transcript-line.tsx:1-249` | Renders one line of a `TranscriptEntry` (note/said/tool/hold/fault); uses `renderSanitizedMarkdown` for assistant text (line 105) | **Unchanged.** `0005:636`. The five `TranscriptEntry` kinds are an internal type; the line renders whatever the page hands it. |
| `frontend/src/lib/chat-api.ts:107` | `export const OFFLINE_LITERAL = "backend not wired — see PR for backend wire"` | **Deleted.** Retiring the literal is `0005:633`'s deliverable. |
| `frontend/src/lib/chat-api.ts:186-212` | `submitTurn(req)` — `POST /api/agent/turns` via `stateChangingFetch` (line 191); envelope parser mirrors `lib/api.ts:96-110` (lines 118-166) | **Consumed unchanged.** `0005:636`. The hook on the page calls it; CH-05.1 does not edit the client. |
| `frontend/src/lib/chat-api.ts:229-246` | `cancelTurn(req)` — `DELETE /api/agent/turns/:id` with `keepalive: true` (line 236) | **Consumed unchanged.** `0005:636`. |
| `frontend/src/lib/chat-api.ts:297-413` | `subscribeTurn(url, onEvent, onError?)` — `new EventSource(url, { withCredentials: true })` (line 309); `addEventListener` per known event name (lines 389-393); `turn.end` closes once (lines 370-372); pre-delta error surfaces `OFFLINE_LITERAL` (lines 401-408) | **Consumed unchanged.** `0005:636`. The onError/offline branch loses its only call site after CH-05.1 — see § 5. |
| `frontend/src/lib/chat-types.ts:27-32` | `ChatStreamEvent` union (`message.start` / `message.delta` / `message.end` / `turn.end` / `error`) | **Consumed unchanged.** `0005:636`. |
| `frontend/src/lib/chat-types.ts:104-117` | `ChatMessage`, `ChatSession` — the signal-store shapes the chat page hook owns (comment at `:100`) | **Reified.** A real `useChatStream` hook is what the page needs; today no such hook exists (the only file is `chat-api.ts:9`'s design note mentioning `use-chat-stream.ts`). |
| `frontend/src/components/chat/__fixtures__/single-turn.sse` | Recorded SSE transcript for `parseTranscript` byte-exact test (lines 1-18) | **Unchanged** for the wire-parser assertion; the test pair is `chat-types.ts:172-238` (`parseTranscript`) and `chat-api.spec.ts:78-83` (`singleTurnFixture`). |
| `backend/agent/src/chat/http.go:305-307` | `api.POST("/turns", HandleOpenTurn(registry))`, `api.GET("/turns/:id/events", HandleStreamEvents(registry))`, `api.DELETE("/turns/:id", HandleCancelTurn(registry))` mounted at `/api/agent` (line 303) | **Unchanged.** CH-05 does not write Go. |
| `backend/agent/src/chat/eventsource.go:31-48` | Five `event:` names sealed: `message.start`, `message.delta`, `message.end`, `turn.end`, `error` | **Unchanged.** The wire the page will read. |
| `backend/agent/src/cmd/chat/main.go:152-189` | Composition root `run(...)` wires `chat.RegisterRoutes(e, resolver, factory)` (line 169) and binds Echo at `:cfg.Port` (line 176) | **Unchanged.** CH-05 is a frontend change; the binary is reachable once `docker compose up` brings the chat service up. |

---

## 3. Frozen contract behaviour today — what CH-05.1 must keep, what CH-05.2 must amend

The frozen contract is `openspec/specs/frontend-chat-layer1/spec.md`. Every clause CH-05 touches is quoted below at its `spec.md` line. No clause is paraphrased.

| Clause | Verbatim citation | CH-05.1 stance | CH-05.2 stance |
|---|---|---|---|
| **REQ-1 (S-1.a)** submit-a-turn happy path | `spec.md:15` — *"Submitting a prompt from `/chat` SHALL open `POST /api/agent/turns`, open an `EventSource` on the returned `stream_url`, accumulate `event: message.delta` text into a single assistant bubble, and close the stream on `event: turn.end`."* S-1.a: `spec.md:21` | **Consumes unchanged.** The hook on the page is the only addition. | **No amendment.** |
| **REQ-1 (S-1.b)** | `spec.md:22` — *"Given a frame whose `event:` name is not in `{ message.start, message.delta, message.end, turn.end }`, when the frame arrives, then the client ignores it and the buffer is unchanged."* | **Consumes unchanged** (`chat-api.ts:380-383` already drops unknown events). | **No amendment.** |
| **REQ-2 (S-2.a)** unmount issues DELETE | `spec.md:32` — *"Given an open `EventSource` for `turn_id = T`, when the user navigates away from `/chat`, then `useVisibleTask$ cleanup()` runs and `DELETE /api/agent/turns/T` issues (via `stateChangingFetch` so `X-Requested-With: XMLHttpRequest` is set) and `EventSource.close()` runs exactly once."* | **Consumes unchanged.** Today the chat page has no such `cleanup()` — `chat-app.tsx:48,73` are `useVisibleTask$` blocks without `cleanup` parameters. CH-05.1 must add the DELETE-issuing cleanup; see § 9. | **No amendment.** |
| **REQ-2 (S-2.b)** Stop button | `spec.md:33` — *"Given an open turn, when the user clicks **Stop**, then `DELETE /api/agent/turns/T` fires with the same wrapper, and the input returns to 'ready' state, and the partial assistant text remains visible."* | **Consumes unchanged.** Composer (`composer.tsx:77-86`) already swaps Send for Stop; the wiring on the page must now point at a real cancel. | **No amendment.** |
| **REQ-2 (S-2.c)** | `spec.md:34` — *"Given `event: turn.end` arrives first, when the client closes per REQ-1, then no `DELETE` issues (the turn is already terminal)."* | **Consumes unchanged** (`chat-api.ts:311-312,370-372` already enforces single-close on `turn.end`). | **No amendment.** |
| **REQ-3 (S-3.a/b/c)** auth guard | `spec.md:38,44-46` — *"Unauthenticated access to `/chat` SHALL redirect via the existing `setSsrCookieHeader → requireAuthRedirect → requireOwnboarding` chain, identical to `routes/home/`."* | **Consumes unchanged** (`routes/chat/layout.tsx:30-34` already is the chain — `route-guard.spec.ts:32-70` is structural-only). | **No amendment.** |
| **REQ-4 (S-4.a)** mid-stream typed error | `spec.md:56` — *"Given an open turn, when the backend closes the stream with `event: error` carrying `kind: 'server'`, then the client renders one error bubble with the backend `message`, the input clears `disabled`, and NO retry timer schedules."* | **Consumes unchanged.** `chat-api.ts:373-379` already surfaces a typed `error` event. The renderer must produce an inline error line — see § 5 / fork B in § 10. | **No amendment.** |
| **REQ-4 (S-4.b/c)** HTTP error envelopes | `spec.md:57-58` — 400 `kind:"validation"` / 409 `kind:"conflict"` render inline; no retry. | **Consumes unchanged** (`chat-api.ts:131-166` already maps every envelope kind). | **No amendment.** |
| **REQ-5** dev-mode honest offline failure | `spec.md:62` — *"When the backend is unreachable from a dev browser, `chat-api.ts` SHALL surface `ApiErrorKind = 'offline'` whose `message` contains the literal `'backend not wired — see PR for backend wire'`; the client SHALL NOT silently retry, SHALL NOT fabricate a response, and SHALL NOT log only to console."* | **AMENDED.** The archetype now serves the wire, so the dev-mode offline *kind* is reachable only when the developer runs the frontend without the chat binary. The literal phrase is retired on the record; the discriminated union's `offline` kind survives (it is the right shape for any transient network failure). | **CH-05.2 amends REQ-5** to state that the offline *literal* is retired; the `kind:"offline"` arm of the error envelope stays. The replacement spec text must state the purpose (greppable architectural gap) is discharged by the archetype serving the wire. |
| **REQ-5 (S-5.a)** | `spec.md:68` — *"Given `pnpm dev` runs with no backend on the chat endpoint, when the user submits a prompt, then `chat-api.ts` resolves to `{ ok: false, kind: 'offline', message: 'backend not wired — see PR for backend wire' }`…"* | **Asserted-away by amendment.** | **Scenario's literal-string assertion is removed.** `kind:"offline"` resolution remains as a generic network-failure shape. |
| **REQ-5 (S-5.b)** | `spec.md:69` — *"Given `EventSource` opens in dev with no backend, when `onerror` fires before any `message.delta`, then the client converts the error into the same `kind: 'offline'` payload and the conversation accepts a fresh submit."* | **Consumes unchanged in shape; literal retired.** | **Scenario's literal-string assertion is removed; the `kind:"offline"` shape and the no-auto-retry / no-fake-bubble properties stay.** |
| **REQ-6 (S-6.a/b)** markdown sanitization | `spec.md:79-80` — assistant text through `renderSanitizedMarkdown` (`lib/markdown.ts:65-67`); no `dangerouslySetInnerHTML` of raw HTML. | **Consumes unchanged** (`transcript-line.tsx:105`; sanitizer allowlist at `markdown.ts:27-59`). | **No amendment.** |
| **REQ-7 (S-7.a/b/c)** per-file spec discipline | `spec.md:90-92` — every `*.ts`/`*.tsx` under `lib/chat-api.ts` and `components/chat/` has a colocated `*.spec.{ts,tsx}` that asserts at least one REQ-N scenario; no `it.skip`/`it.todo`/`xit(…)`. | **Consumes unchanged; carries a new file under the same rule.** CH-05.1's new `use-chat-stream.ts` must land with `use-chat-stream.spec.ts` (or equivalent). | **No amendment.** |

**What the doc says CH-05.2's amendment must do**, per `0005:673`: *"a spec delta modifying the offline-honesty requirement, stating that its purpose — making an unwired backend greppable — is discharged by the archetype now serving the wire, and that the literal is retired rather than forgotten. The merged tree contains no occurrence of the literal, and the amended requirement's scenarios no longer assert it."* The first half is the rationale; the second half is the mechanical proof CH-05.2 must produce — see § 4.

---

## 4. The retirement, in concrete terms

### 4.1 The literal

```text
"backend not wired — see PR for backend wire"
```

Defined once: `frontend/src/lib/chat-api.ts:107`:

```
107: export const OFFLINE_LITERAL = "backend not wired — see PR for backend wire";
```

It is a `const`, not a derived string. Two runtime callers: `chat-api.ts:111` (the `offlineMessage(err)` formatter — appends `(${detail})`) and `chat-api.ts:404,406` (the EventSource pre-delta error branch). Four doc-comment callers: `chat-api.ts:27,103,180,260` and `chat-types.ts:91`.

### 4.2 Every place it surfaces today

`rg -n "OFFLINE_LITERAL\|backend not wired" frontend/` returns 13 lines (verified):

- `frontend/src/lib/chat-types.ts:91` — comment in REQ-4 + REQ-5 narrative.
- `frontend/src/lib/chat-api.spec.ts:486` — **runtime assertion** on `submitTurn`'s offline error path.
- `frontend/src/lib/chat-api.spec.ts:662` — **runtime assertion** on `subscribeTurn`'s pre-delta error path.
- `frontend/src/lib/chat-api.ts:27` — design comment.
- `frontend/src/lib/chat-api.ts:103` — design comment (lines 99-106 are the rationale block).
- `frontend/src/lib/chat-api.ts:107` — the constant itself.
- `frontend/src/lib/chat-api.ts:111` — runtime formatter.
- `frontend/src/lib/chat-api.ts:180` — design comment on `submitTurn`.
- `frontend/src/lib/chat-api.ts:260` — design comment on the `ChatOfflineHandler` type.
- `frontend/src/lib/chat-api.ts:404` — runtime branch (`onError(OFFLINE_LITERAL)`).
- `frontend/src/lib/chat-api.ts:406` — runtime branch (same call, different guard).

### 4.3 Every test that asserts it

Two assertions, both inside `chat-api.spec.ts`, both inside the `describe("chat-api wire client (REQ-1, REQ-2, REQ-4, REQ-5)")` block (lines 386-671):

1. **`it("submitTurn maps network errors to kind=offline with the literal phrase (REQ-5)")`** — `chat-api.spec.ts:472-488`. Asserts `result.kind === "offline"` AND `result.message` contains `"backend not wired — see PR for backend wire"`.
2. **`it("subscribeTurn treats EventSource onerror before any message as offline (REQ-5 S-5.b)")`** — `chat-api.spec.ts:645-664`. Asserts the `onError` callback receives the literal phrase.

Both are `REQ-5` tests per the comment at `chat-api.spec.ts:368-381`. CH-05.1 must change both — see § 10 fork A.

### 4.4 What the spec says must be true after the merge

`0005:673` (the CH-05.2 check-evidence clause): *"The merged tree contains no occurrence of the literal, and the amended requirement's scenarios no longer assert it."* A spec-level grep like `rg "backend not wired" frontend/ openspec/specs/` returning zero hits is the mechanical proof.

---

## 5. Mock surface to replace — fork to be decided by the proposal

The chat page reads three names from `lib/mock/chat.ts`: `CONVERSATIONS`, `useMockTurn` (via `use-mock-turn.ts`), `CONVERSATIONS[0].entries` (the initial transcript seed). `lib/mock/staff.ts` supplies `AGENTS` and `agentBySlug` (the colleague the conversation is with). The mock surface that CH-05.1 touches is the page's *turn machine*, not the workplace shell's data — and the two are entangled in ways the proposal must resolve.

### 5.1 Files in scope of CH-05's review

| File | Lines | Used by | CH-05 disposition |
|---|---|---|---|
| `frontend/src/components/chat/use-mock-turn.ts` | 1-349 | `components/chat/chat-app.tsx:17,39` AND `components/marketing/hero-proof/hero-proof.tsx:21,31` | **The hook survives.** It must remain importable for `hero-proof.tsx` (`hero-proof.tsx:21` imports `useMockTurn`; `:31` calls it; `:44,50` call `turn.settle`/`turn.play`). The CH-05.1 change is *the page stops calling it*, not the file's deletion. |
| `frontend/src/lib/mock/chat.ts` | 1-504 | `use-mock-turn.ts:8` (uses `scriptFor`, `Beat`, `TranscriptEntry`); `chat-app.tsx:12` (uses `CONVERSATIONS`); `conversation-list.tsx:11` (uses `Conversation` type); `transcript-line.tsx:27` (uses `TranscriptEntry` type); `hero-proof.tsx:23` (uses `HERO_OPENING`, `HERO_SCRIPT`); `workspace/screens/front-desk.tsx:23` (uses `CONVERSATIONS`); spec files. | **Cannot be wholesale-deleted.** Multiple non-chat callers depend on its exports. See § 5.2 for the per-export tally. |
| `frontend/src/lib/mock/staff.ts` | 1-461 | **Everywhere**: `chat-app.tsx:13`, `conversation-list.tsx:12`, `transcript-line.tsx:28`, `sidebar.tsx:30`, `status.tsx:10`, `agent-directory.tsx:14`, `agent-profile.tsx:21`, `agent-card.tsx:18`, `avatar.tsx:18`, `teams-board.tsx:22`, `organization-panel.tsx:21`, `front-desk.tsx:30`, `hero-proof.tsx:24`, `routes/index.tsx:29`, `routes/agents/[slug]/index.tsx:11`. | **Out of scope.** It is the workplace shell's mock staff (D2 of the workplace proposal explicitly forbids surfacing it as a runtime concern). CH-05 does not write to it. |
| `frontend/src/components/chat/chat-app.tsx` | 1-220 | `routes/chat/index.tsx:42-46` | **The page itself.** The `useMockTurn` import (line 17) and the call (line 39) are replaced; the `CONVERSATIONS[0].entries` seed (line 39) is replaced; the `useTask$` reset on selection (lines 59-68) is replaced; the `?with=` deep-link (lines 48-54) loses its meaning until CH-08 lands. |
| `frontend/src/components/chat/conversation-list.tsx` | 1-59 | `chat-app.tsx:104-110,156-164` | **Conditional** — the list panel renders `CONVERSATIONS` (the mock). The fork in § 10 commits to one of: (a) keep rendering the mock list as historical-context for the now-real turn, or (b) drop the panel until CH-08.2. |
| `frontend/src/components/chat/transcript-line.tsx` | 1-249 | `chat-app.tsx:197-208`; `hero-proof.tsx:80-92` | **Unchanged per doc.** `0005:636`. The five `TranscriptEntry` kinds (mock) are kept; the page's real hook maps Qwik signals → `TranscriptEntry[]` for the line renderer. |
| `frontend/src/lib/mock/chat.spec.ts` | 1-153 | Tests `scriptFor`, `CONVERSATIONS`, the seed-content properties (`chat.spec.ts:19-153`) | **Survives** because `scriptFor` and `CONVERSATIONS` survive (hero-proof). |

### 5.2 The non-chat consumers of `lib/mock/chat.ts`

`grep -rn "from.*lib/mock/chat\|from.*lib/mock/staff" frontend/src/ | grep -v .spec.` (verified):

- `components/chat/use-mock-turn.ts:8` — `scriptFor`, `Beat`, `TranscriptEntry` (kept: hero-proof)
- `components/chat/chat-app.tsx:12` — `CONVERSATIONS` (**CH-05.1 review target**)
- `components/chat/conversation-list.tsx:11` — `Conversation` type (kept, conditionally per § 10)
- `components/chat/transcript-line.tsx:27` — `TranscriptEntry` type (kept: page hook maps to it)
- `components/marketing/hero-proof/hero-proof.tsx:23` — `HERO_OPENING`, `HERO_SCRIPT` (kept: marketing demo, frozen wire reference)
- `components/workspace/screens/front-desk.tsx:23` — `CONVERSATIONS` (kept: workplace shell "where you left off" panel)

Plus `lib/mock/chat.spec.ts:14` (kept).

### 5.3 The fork the proposal must name, not decide here

The explore cannot answer which of these the proposal commits to without reading user intent:

- **Fork A — "delete the literal, keep the `offline` kind."** The literal `OFFLINE_LITERAL` is removed. `chat-api.ts:107,111` become an `offlineMessage(err)` that returns a generic message (e.g. `"Couldn't reach the chat service. Is it running?"`). The `kind:"offline"` arm of `ApiResult<T>` (`api.ts:89-94`; `chat-types.ts:93-98`) survives. The two `chat-api.spec.ts` assertions at lines 472-488 and 645-664 lose the literal-substring check but keep the `kind === "offline"` check. This is the only fork the spec amendment can support — REQ-5 still mandates offline-honesty, just without the user-locked dev phrase.
- **Fork B — "delete the offline kind entirely."** REQ-5 is removed altogether; a network failure maps to `kind:"server"` (R-12's no-client-retry contract still applies — `api.ts:259-263`). The proposal must argue this contradicts the doc 0001 §4.1 *line 426-427* invocation of `chat-archetype-contract/R-CHT-009`'s enumerated REQ list, which this explore is not qualified to settle.

The proposal also has a smaller, scoped fork on the mock surface itself: see § 10 fork C.

---

## 6. Auth + render surface CH-05 consumes unchanged

These are the contracts CH-05.1 is forbidden from widening and the implementation it must reuse as-is, per `0005:636` and `chat-archetype-contract/R-CHT-009` (`openspec/specs/chat-archetype-contract/spec.md:177`).

### 6.1 The canonical guard chain (REQ-3)

`frontend/src/routes/home/layout.tsx:30-34` — read by the corrected citation at `chat-archetype-contract/spec.md:179`. The `frontend-chat-layer1/spec.md:134` citation (`routes/home/index.tsx:39-51`) is **stale** — `index.tsx` is now 45 lines, and the `onRequest` chain moved to the layout in the `cachicamas-frontend-workplace` change (D7 at `cachicamas-frontend-workplace/proposal.md`):

```
30: export const onRequest: RequestHandler = async (event) => {
31:   setSsrCookieHeader(event.request.headers.get("cookie") ?? "");
32:   requireAuthRedirect(event);
33:   await requireOwnboarding(event);
34: };
```

The chat route has the identical chain at `frontend/src/routes/chat/layout.tsx:30-34`. The `route-guard.spec.ts:32-70` assertion suite reads the body of `onRequest` (line 49) and enforces order (`cookieAt < authAt < onboardAt`) — this assertion will pass unchanged for the chat route.

### 6.2 The sanitization boundary (REQ-6)

`frontend/src/lib/markdown.ts:65-67`:

```
65: export function renderSanitizedMarkdown(md: string): string {
66:   const html = renderMarkdown(md);
67:   return DOMPurify.sanitize(html, SANITIZER_CONFIG) as string;
68: }
```

Allowlist at `markdown.ts:27-59` (tags: `p`, `br`, `strong`, `em`, `u`, `s`, `code`, `pre`, `blockquote`, `ul`, `ol`, `li`, `h1`-`h6`, `a`, `img`, `table`, `thead`, `tbody`, `tr`, `th`, `td`, `hr`; attributes: `href`, `src`, `alt`, `title`; URI regex `/^(?:https?:|mailto:|\/|#)/i` rejecting `javascript:` and `data:text/html`). Consumed at `transcript-line.tsx:105`:

```
105: dangerouslySetInnerHTML={renderSanitizedMarkdown(entry.text)}
```

`transcript-line.spec.tsx:73-91` asserts the allowlist holds (no `<script>`, no `onerror`); REQ-6 forbids hand-rolled sanitization.

### 6.3 The route guard (REQ-3, mirror of `routes/home/`)

`frontend/src/lib/require-auth-redirect.ts`, `frontend/src/lib/require-ownboarding.ts`, `frontend/src/lib/ssr-cookie-context.ts` — imported at `routes/chat/layout.tsx:15-17`. The same three are imported at `routes/home/layout.tsx:15-17` and `routes/organization/layout.tsx:15-17` and `routes/settings/layout.tsx:15-17`. CH-05.1 does not edit any of these.

### 6.4 The CSRF defense-in-depth wrapper

`frontend/src/lib/csrf.ts:28-39` — `stateChangingFetch(url, init?)` adds `X-Requested-With: XMLHttpRequest` on `POST`/`PUT`/`PATCH`/`DELETE` (line 17-22), passes safe methods through. `chat-api.ts:40` imports it; `chat-api.ts:191` uses it for `POST /api/agent/turns`; `chat-api.ts:234` uses it for `DELETE /api/agent/turns/:id` with `keepalive: true` (line 236). The `chat-api.spec.ts:467-468,543-545` assertions on `X-Requested-With` carry the requirement to the spec.

### 6.5 The `ApiResult<T>` envelope (REQ-4 + REQ-5)

`frontend/src/lib/api.ts:96-110` — discriminated union with five kinds: `validation` (carries `fields`), `conflict`, `not_found`, `server`, `offline`. `chat-api.ts:118-166` re-implements the same envelope parser inside `envelopeToResult`. `chat-types.ts:93-98` re-types the same five kinds as `ChatTurnError`. The offline kind is the fork in § 5.3 / § 10 fork A.

### 6.6 The vitest red-green target (REQ-7)

`frontend/package.json:28`:

```
"test:ci": "vitest --run --reporter=json"
```

`frontend/package.json:13`:

```
"verify": "pnpm lint && pnpm build.types && pnpm fmt.check && pnpm test:ci"
```

The CH-05.1 verification gate is `pnpm --filter @cachicamas/frontend test:ci` against `*.spec.{ts,tsx}` — strictly per-file spec discipline (`chat-archetype-contract/spec.md` and `frontend-chat-layer1/spec.md:82-92`).

---

## 7. Backend surface to point at

The wire CH-05.1 reads is the one CH-03 mounted (`backend/agent/src/chat/http.go`) and CH-04 brought up (`backend/agent/src/cmd/chat/main.go`). Both are merged on `main` as of `2b891117`. Cited below at the code the page reads against.

### 7.1 URL and method table

| HTTP method | Path | Backend handler | Returns | Reference |
|---|---|---|---|---|
| `POST` | `/api/agent/turns` | `chat.HandleOpenTurn(registry)` | `200` with `{turnId, streamUrl}` OR `400 validation` OR `409 conflict` | `http.go:305`, response shape at `:47-50`, error envelope at `:56-68` |
| `GET` (SSE) | `/api/agent/turns/:id/events` | `chat.HandleStreamEvents(registry)` | `text/event-stream` with `event: <name>\ndata: <json>\n\n` frames; closes on terminal frame or `ctx.Done()` | `http.go:306`, `:195-244`; SSE writer at `eventsource.go:62-79` |
| `DELETE` | `/api/agent/turns/:id` | `chat.HandleCancelTurn(registry)` | `204 No Content` (no-op for unknown or already-terminal) | `http.go:307`, `:255-276` |
| any refusal | | `chat.identityMiddleware(resolver)` (Echo middleware) | `401 server` (`error="server"`, `message="identity not resolved"`) | `http.go:80-92`, mounted at `:303` |
| cross-participant | | `getIdentity` check inside handlers | `403 not_found` (existence not leaked per `R-CHS-004.b`) | `http.go:206-217,264-270` |

### 7.2 POST request and response

Body the client sends (`openTurnRequest`, `http.go:39-43`):

```json
{ "id": "<client-minted UUID>", "prompt": "<text>", "systemHint": "<optional, ignored>" }
```

Response body (`openTurnResponse`, `http.go:47-50`):

```json
{ "turnId": "<string>", "streamUrl": "/api/agent/turns/<turnId>/events" }
```

The `streamUrl` is a **relative path** (`streamURLFor` at `http.go:280-282` returns the bare path). `chat-api.ts:306-308` prefixes `apiBaseUrl()` when the URL is relative. `chat-api.spec.ts:448-460` asserts both fields by name.

The handler returns `200` (not `202` — see the typo at `chat-api.spec.ts:449` where the test fixture uses `status: 202`; the real handler returns `http.StatusOK` at `http.go:175`). The mismatch is in the spec only and does not affect parsing — `envelopeToResult` checks `res.ok`, not the status code. **Worth a single-line correction in CH-05.1's spec cleanup.** Not load-bearing.

### 7.3 SSE event names the page reads

Sealed at `backend/agent/src/chat/eventsource.go:31-48` (`wireFrameName`), mirrored in the frontend at `chat-api.ts:266-272` (`KNOWN_EVENTS`):

| `event:` field | Wire shape (`data:` JSON) | Frontend variant (`chat-types.ts:27-32`) |
|---|---|---|
| `message.start` | `{"messageId": "<string>", "index": <int>}` | `{ kind: "message.start", messageId, index }` (`chat-types.ts:28`) |
| `message.delta` | `{"index": <int>, "delta": "<text>"}` | `{ kind: "message.delta", index, delta }` (`chat-types.ts:29`) |
| `message.end` | `{"index": <int>, "finishReason": "<stop\|length\|tool_calls\|refusal\|pause_turn\|content_filter\|unknown>"}` | `{ kind: "message.end", index, finishReason }` (`chat-types.ts:30`) |
| `turn.end` | `{"usage": <optional ChatUsage>, "finishReason": <optional>}` | `{ kind: "turn.end", usage?, finishReason? }` (`chat-types.ts:31`) |
| `error` | `{"kind": "<validation\|conflict\|not_found\|server>", "message": "<text>"}` | `{ kind: "error", error: ChatStreamError }` (`chat-types.ts:32`); `ChatStreamError` at `:59-63` |

Both ends use the seven-value `finishReason` vocabulary (`wire.go:73-80` ↔ `chat-types.ts:36-43`). `chat-types.ts:31`'s `turn.end` carries an *optional* `finishReason` — `wire.go:53-55`'s `TurnEnd{FinishReason *string}` is the cancellation discriminator (D5 at `wire.go:51-52`: nil means cancelled). `chat-api.ts:359-368` already serializes that correctly: `...(obj.finishReason ? { finishReason: ... } : {})`.

The canonical fixture `frontend/src/components/chat/__fixtures__/single-turn.sse` (1-18) is the byte-exact ground truth: `message.start → 3× message.delta → message.end → turn.end`. `chat-api.spec.ts:62-76` is the typed-event assertion that reads it.

### 7.4 Error envelope

`http.go:56-68` (`errorEnvelope`) — `{ "error": "<kind>", "message": "<text>", "fields": <map>? }`. JSON tags are lowercase; the parser at `chat-api.ts:131-166` reads `body.error` and `body.message` and `body.fields` exactly. Three mappings worth flagging:

- **400 or 422** + `error:"validation"` → `kind:"validation"` with synthesized `message = "<firstField>: <firstFieldMsg>"` (or generic fallback). Carries `fields: Record<string,string>`.
- **409** + `error:"conflict"` → `kind:"conflict"`. The chat backend emits this only for the in-flight conflict (`http.go:147-150` — `S-CHS-001.c`). The page can render the typed message inline (REQ-4 S-4.c).
- **5xx** (and any 4xx that doesn't match) → `kind:"server"` (REQ-4 S-4.a — generic 500/501 path).

The `offline` kind is **never** produced by the server. It is the client-side discriminator for transport failure (`chat-api.ts:198-204,238-244,401-408`). Removing it would violate the no-network-failure-shaped-as-typed-API-error property the architecture relies on.

---

## 8. Test patterns to mirror

### 8.1 The FakeES fixture (REQ-1 S-1.a, REQ-4 S-4.a, REQ-5 S-5.b)

**Location:** `frontend/src/lib/chat-api.spec.ts:394-425` — the class is **defined inline** in the test file's `describe("chat-api wire client (REQ-1, REQ-2, REQ-4, REQ-5)")` block (lines 386-671). `vitest`'s `beforeEach` at `:427-433` swaps `globalThis.EventSource = FakeES`, and `afterEach` at `:435-444` restores it.

**Stale citation, real consequence:** both `frontend-chat-layer1/spec.md:131` and `chat-api.spec.ts:380,390` claim the pattern lives at `frontend/src/lib/api.sse.spec.ts:75-307`. **That file does not exist** (`find frontend -name "*sse.spec*"` returns nothing; `api.ts` is 400 lines and contains no `subscribeWorkspaceSyncStream`). The pattern is now the inline class. The proposal's CH-05.2 amendment must either delete the spec's stale line or correct it (the spec lives under `openspec/specs/` and so is editable only via a delta, but `chat-api.spec.ts:380,390` are free to fix in CH-05.1).

The class itself (`chat-api.spec.ts:394-425`) is six methods:

```
394: class FakeES {
395:   url: string;
396:   withCredentials = false;
397:   listeners: Record<string, Array<(ev: MessageEvent<string>) => void>> = {};
398:   closed = false;
399:   constructor(url: string, opts?: { withCredentials?: boolean }) { ... }
404:   addEventListener(name: string, cb) { ... }
408:   removeEventListener(name: string, cb) { ... }
414:   close() { this.closed = true; }
417:   fire(name: string, data: unknown) { ... } // test-only — calls registered listeners
425: }
```

Every REQ-1 / REQ-2 / REQ-4 / REQ-5 assertion in `chat-api.spec.ts:561-671` uses it. The CH-05.1 spec author copies the class verbatim (the S-7.b REQ-7 assertion will catch any drift — see § 8.2).

### 8.2 The colocated `*.spec.{ts,tsx}` rule (REQ-7 S-7.a/S-7.b/S-7.c)

`frontend-chat-layer1/spec.md:84` — *"Each `*.ts` and `*.tsx` file under `frontend/src/lib/chat-api.ts` and `frontend/src/components/chat/` SHALL have a colocated `*.spec.ts` / `*.spec.tsx`…"*. The check is grep-driven (`spec.md:90-92`): `ls` shows every file has a sibling; `grep -E "REQ-[1-7]"` finds at least one reference; `pnpm test:ci` reports zero `it.skip`/`it.todo`/`xit`.

Files in scope today (verified by `ls`):

- `lib/chat-api.ts` ↔ `lib/chat-api.spec.ts` (672 lines)
- `lib/chat-types.ts` ↔ *no sibling* — but `chat-types.ts` is type-only (no runtime), and REQ-7 only requires the test for runtime files. S-7.a is satisfied because `chat-api.spec.ts:19-32` exhaustively probes the types. **No new spec file is required for `chat-types.ts`.**
- `components/chat/chat-app.tsx` ↔ `components/chat/chat-app.spec.tsx` (99 lines)
- `components/chat/composer.tsx` ↔ `components/chat/composer.spec.tsx` (103 lines)
- `components/chat/conversation-list.tsx` ↔ `components/chat/conversation-list.spec.tsx` (read length: 100 lines — assertion: history list behaviour)
- `components/chat/transcript-line.tsx` ↔ `components/chat/transcript-line.spec.tsx` (254 lines)
- `components/chat/use-mock-turn.ts` ↔ `components/chat/use-mock-turn.spec.ts` (248 lines)
- `routes/chat/index.tsx` ↔ `routes/chat/index.spec.tsx` (174 lines)
- `routes/chat/layout.tsx` ↔ `routes/chat/route-guard.spec.ts` (71 lines)

**New file CH-05.1 introduces:** a `useChatStream` hook (`components/chat/use-chat-stream.ts` is the name the design comment at `chat-api.ts:9` already reserves). It MUST ship with `use-chat-stream.spec.ts` or the S-7.a assertion goes red.

### 8.3 The red-green target

Per `frontend/package.json:28`: `pnpm --filter @cachicamas/frontend test:ci` (vitest `--run --reporter=json`). `openspec/AGENTS.md` "Strict TDD is on" applies — the test is the spec. The chat's behavioural assertions are property-style over the FakeES, not over real network, so the gate is local (no backend up required for vitest).

### 8.4 The non-frontend evidence gate

`0005:669` — CH-05.1's evidence gate is **both** the frontend suite and the backend suite. The backend's existing `cd backend/agent && go test -race -count=1 ./...` is unaffected by CH-05.1 (no Go changes). The frontend's `pnpm test:ci` is the only suite this change reddens or greens. CH-05.2 produces no tests; it produces the spec delta text and the grep-zero proof.

---

## 9. Risks — every non-obvious thing the proposal could trip over

### 9.1 The SSR-split `useVisibleTask$` cleanup

The chat page today has **no** `cleanup` callback in its `useVisibleTask$` blocks — see `chat-app.tsx:47-54` and `:73-79`. REQ-2 S-2.a (`frontend-chat-layer1/spec.md:32`) requires the unmount cleanup to issue `DELETE /api/agent/turns/T` via `stateChangingFetch` *and* call `EventSource.close()` exactly once. CH-05.1 must add the cleanup; the shape is `useVisibleTask$(({ cleanup, track }) => { ... cleanup(() => { cancelTurn(...); unsubscribe(); }) })`. The hook must guard against `unsubscribe` running twice (turn-end already closed). If the page mounts an EventSource on every conversation selection (the `useTask$` reset at `chat-app.tsx:59-68` will replay for every `selected` change), the cleanup chain becomes `track + cleanup` per selection. **A naive swap of `useMockTurn` for `useChatStream` without a cleanup will leave orphan turns server-side.** This is the single biggest non-obvious failure mode.

### 9.2 The typed `Event` parameter parsing

`chat-api.ts:314-393` parses each SSE frame as `JSON.parse(raw)` then a `switch (name as KnownEventName)`. The known-event set at `:266-272` is `{ "message.start", "message.delta", "message.end", "turn.end", "error" }`. The page-side hook reuses this function — a future `useChatStream` that drops or rewrites the parser will break the byte-exact `single-turn.sse` fixture test at `chat-api.spec.ts:60-83`. Mirror the parser, do not re-implement it.

### 9.3 The `keepalive: true` on DELETE

`chat-api.ts:236` — the cancel fetch is sent with `keepalive: true`. `chat-api.spec.ts:542` asserts this with `(init as RequestInit & { keepalive?: boolean }).keepalive).toBe(true)`. The browser only honours `keepalive` on `GET`/`POST` in some implementations but Echo accepts it on `DELETE` (verified by `chat-api.spec.ts:531-545`). CH-05.1 must keep this; the cancellation cleanup on unmount relies on it (`frontend-chat-layer1/spec.md:32`).

### 9.4 The doc's "out of scope" line

`0005:636` — *"Out of scope: rendering, sanitization, the route guard and the error envelope — all frozen by `frontend-chat-layer1` and consumed unchanged; conversation history on reload — owned by CH-08."* This is the seam fence. Any CH-05.1 PR that edits `transcript-line.tsx`, the `markdown.ts` allowlist, the `routes/chat/layout.tsx` guard chain, or the `chat-api.ts:118-166` envelope parser is widening scope and will fail review per the proposal house style (`openspec/changes/archive/2026-08-23-cachicamas-chat-package-scaffold/proposal.md:46-49` is the precedent — out-of-scope table is read with the change author's own citation). Edit the *hook*, not the renderer / sanitizer / guard / envelope.

### 9.5 The substrate preservation rule for `backend/agent`

`openspec/AGENTS.md` § "Substrate preservation in `backend/agent` (NFR-TLS-003)" — ten files form the invariant substrate (`event_descriptor.go`, `stream_check.go`, `failure.go`, `sequence.go`, `event.go`, `go.mod`, `go.sum`, `Makefile`, `.golangci.yml`, `import_boundary_test.go`). CH-05 **does not write Go** and does not touch the substrate. The risk is not in CH-05; the risk is that the proposal author makes a `go.mod` mention by accident in the diff description and a reviewer assumes a Layer-2 assertion was widened. The proposal should state explicitly *"no file under `backend/` is touched by this change"* in its out-of-scope table (the precedent at `cachicamas-chat-package-scaffold/proposal.md:216`).

### 9.6 The marketing `hero-proof` still uses `useMockTurn`

`frontend/src/components/marketing/hero-proof/hero-proof.tsx:21,31,44,50,89` — the public landing page's proof-of-stopping demo runs the same scripted machine the chat page currently runs. `cachicamas-frontend-workplace/proposal.md:D6` froze the chat's wire, not its UI — the hero-proof is *part of* the marketing surface and continues to play scripted beats on the public page regardless of what CH-05 does to the chat page. **`useMockTurn` and `HERO_OPENING`/`HERO_SCRIPT` are not deletable in CH-05.** The proposal that proposes deleting them is wrong.

### 9.7 The front-desk "where you left off" panel

`frontend/src/components/workspace/screens/front-desk.tsx:114-140` — the home screen lists the same `CONVERSATIONS` array the chat page lists. After CH-05.1, this panel still shows the seeded mock conversations on the home page (it is independent of CH-05's page change). The panel is **not** in CH-05's scope; touching it widens scope to the workplace shell.

### 9.8 The chat page's structural wiring assertion at `routes/chat/index.spec.tsx:82-89`

```js
test("routes/chat: the frozen browser wire is still on disk, unwired", () => {
  // The mockup replaced the chat's UI, not its contract. `chat-api.ts` and
  // `chat-types.ts` carry the frozen open-then-subscribe wire and stay put for
  // CH-05 to connect; deleting them would turn a swap into a rewrite.
  const dir = resolve(fileURLToPath(import.meta.url), "../../../lib");
  expect(existsSync(resolve(dir, "chat-api.ts"))).toBe(true);
  expect(existsSync(resolve(dir, "chat-types.ts"))).toBe(true);
});
```

This test is the assertion that CH-05.1 *falsifies*. After CH-05.1 the wire is connected, not unwired, and `chat-api.ts`/`chat-types.ts` are the wire client, not a frozen-but-unused artifact. The proposal must rewrite this test to assert something *true after* the change — e.g. that `chat-api.ts` still defines `submitTurn`, `cancelTurn`, and `subscribeTurn` (the wire surface CH-05.1 connects to).

### 9.9 The stale `subscribeWorkspaceSyncStream` citation

`frontend-chat-layer1/spec.md:17,130` references `subscribeWorkspaceSyncStream` at `frontend/src/lib/api.ts:896-977`. `api.ts` is 400 lines. The function does not exist; the line range is past EOF. This is pre-existing drift in the frozen contract, unrelated to CH-05 — but the CH-05.2 amendment may need to either delete the line or correct it (same condition as the `api.sse.spec.ts` stale reference at `frontend-chat-layer1/spec.md:131`). The explore flags both as opportunities for cleanup but does not require them.

### 9.10 The TDD red step is mandatory

`openspec/AGENTS.md` § "Strict TDD is on" — RED-GREEN-REFACTOR for any new behavior in the frontend. CH-05.1 introduces a new hook (`useChatStream`); the test (`use-chat-stream.spec.ts`) must exist, fail for the right reason, and only then be made green. The proposal's tasks.md must show the red step.

---

## 10. Open questions for the proposal phase

The explore cannot answer these from the repo; the user (or the proposal with a stated stance) must commit each one.

1. **Fork A — retire the literal, keep the `kind:"offline"` arm.** Does the proposal keep `ApiErrorKind = "offline"` (`api.ts:89-94`, `chat-types.ts:93-98`) and only delete the user-locked literal phrase? Or does it delete the `offline` kind entirely and surface network failures as `kind:"server"`?
   - **Implication if "keep the kind":** `chat-api.ts:107` (`OFFLINE_LITERAL`) and `:111` (`offlineMessage(err)`) are deleted; `:198-204,238-244,401-408` keep their `kind:"offline"` shape; the two test assertions at `chat-api.spec.ts:472-488` and `:645-664` lose the literal-substring check but keep the `kind:"offline"` check; REQ-5 S-5.a and S-5.b lose their literal-text assertion but keep the network-failure-shape assertion; the spec amendment's scenario bodies change from literal to "the typed offline message surfaces".
   - **Implication if "delete the kind":** `offline` is removed from `ApiErrorKind` and `ChatTurnError`; network errors fold into `kind:"server"`; REQ-5 is removed from `frontend-chat-layer1`; the two `chat-api.ts` catch branches return `kind:"server"` instead. This contradicts the spec's enumeration requirement at `chat-archetype-contract/spec.md:177` (`REQ-5` is in the inherited REQ list).

2. **Fork B — where does the deletion happen, mechanically?** Two possible shapes:
   - **Shape 1 — `OFFLINE_LITERAL` deleted; `offlineMessage(err)` returns a generic dev-honest phrase** (e.g. `"Couldn't reach the chat service. Is docker compose up?"` mirroring the pattern at `api.ts:161`). Spec amendment cites the generic phrase in S-5.a's body.
   - **Shape 2 — `OFFLINE_LITERAL` deleted; `offlineMessage(err)` returns the bare `err.message`** with no wrapping phrase. Spec amendment removes S-5.a's `message` assertion entirely; only `kind:"offline"` is asserted. The "honest dev failure" intent is preserved at the protocol level but no longer in the message.

3. **Fork C — chat-page mock surface: keep CONVERSATIONS as historical context or drop until CH-08.2?** The chat page today renders `CONVERSATIONS` as the left rail (chat-app.tsx:104-110,156-164). After CH-05.1 the page's active turn becomes real but the historical list is still the mock array. Two viable shapes:
   - **C-1 — keep the left rail as the seeded historical context.** Render the existing `CONVERSATIONS` mock as "your past conversations" (its current semantic). The active conversation's turn becomes real. The line renderer keeps using `TranscriptEntry`. The chat page is a hybrid of "real turn in front" + "mocked past behind". This is the minimal swap.
   - **C-2 — drop the left rail until CH-08.2.** The chat page shows only the active conversation. `ConversationList` is removed from `chat-app.tsx`. The mock array stays in `lib/mock/chat.ts` for the marketing `hero-proof` and the workplace `front-desk` (both unaffected). CH-08.2 lands `cachicamas-chat-conversation-list` and re-adds the rail.

4. **Fork D — new hook file location and shape.** The design comment at `chat-api.ts:9` and `chat-types.ts:100-103` reserves the name `use-chat-stream.ts`. The hook's signal-store contract (`ChatMessage` at `chat-types.ts:104-110`, `ChatSession` at `:112-116`) is named but unimplemented. Does the proposal:
   - **D-1 — add `useChatStream` as a Qwik hook returning `{ state, submit, cancel, decide? }`** mirroring the `useMockTurn` shape (`use-mock-turn.ts:55-75`) so the page-side swap is a one-line change.
   - **D-2 — add a different shape** (e.g. a `useSignal`/`useStore` pair that returns only the reactive surface, with the wire calls free functions). The shape decision affects what `chat-app.tsx:38-68` looks like after the swap.

5. **Fork E — `systemHint` handling.** `chat-types.ts:71-72` defines `ChatTurnRequest.systemHint?: string` as a reserved seam (`http.go:42` accepts and ignores it). Does CH-05.1 send `systemHint: undefined` explicitly, omit the field, or expose a UI control? The doc says nothing. The minimum-work shape (omit) is safe; the explicit-shape (send `undefined`) is forward-compatible.

6. **Fork F — the chat page's `?with=` deep-link.** `chat-app.tsx:47-54` opens the conversation whose `agentSlug` matches `?with=<slug>`. After CH-05.1 the page no longer holds a list of historical conversations — it holds a single active conversation. Two shapes:
   - **F-1 — keep `?with=` as the URL contract for "open the chat with this agent".** The page seeds an empty transcript and the agent comes from `staff.ts`. The parameter survives; the rail's `data-testid` selection goes away.
   - **F-2 — drop `?with=` until CH-08.** The front-desk panel's link (`front-desk.tsx:119`) becomes `/chat?with=finance` that no longer does anything. **This is a regression** in the workplace shell's wiring that the proposal must call out.

7. **Fork G — the chat page's `useTask$` reset on selection (`chat-app.tsx:59-68`).** Once C-1 or C-2 is chosen, this block becomes inert or is removed. Its three lines (`turn.state.entries = ...; turn.state.script = []; turn.state.beat = 0; turn.state.step = 0; turn.state.status = "idle"`) are the mock machine's API; the real hook has different field names (`state`, `messages`, etc.). The proposal must commit to whether the hook exposes a `reset(entries)` API for symmetry or whether the page drops the selection-reset entirely (a single-conversation page has nothing to reset).

8. **Fork H — verification gate for the live turn.** `0005:669` requires *"both suites — the frontend suite and the backend suite"*. The frontend suite proves the wire client and the hook. The backend suite is untouched by CH-05 (no Go changes). Is the "backend suite green" half of the evidence a *citation* (the merged main's existing green run at `2b891117`) or a *re-run* against this branch's tip? The proposal should commit to "citation only" — re-running the entire backend race suite for a frontend-only change is review-budget waste the precedent rejects (`cachicamas-chat-package-scaffold/proposal.md:46`).

9. **Fork I — the `chat-api.spec.ts:449` `status: 202` typo.** The handler returns `200` (`http.go:175`); the fixture returns `202`. Harmless today (the parser checks `res.ok`), but the proposal may fix it in passing. Or it may leave it for a future amendment; either is defensible. The CH-05 review checklist will catch this if it touches the file.

10. **Fork J — the two stale spec citations** (`subscribeWorkspaceSyncStream` at `frontend-chat-layer1/spec.md:17,130`; `api.sse.spec.ts:75-307` at `:131` and `chat-api.spec.ts:380,390`). CH-05.2 amends REQ-5; the amendment is the natural place to drop the dead reference to `api.sse.spec.ts` (the line is provably wrong). The `subscribeWorkspaceSyncStream` line at `:130` is unrelated to REQ-5 and is left for a future amendment. The `chat-api.spec.ts:380,390` comments are CH-05.1's free hand (not in a promoted spec).

---

## Hard-rule compliance recap

- No file under `backend/`, `frontend/`, `openspec/specs/`, `openspec/changes/archive/`, or `docs/` is modified by this explore. The only write is this file (`openspec/changes/cachicamas-chat-frontend-wire/explore.md`), which did not exist on the worktree before this phase.
- Every `path:line` resolves to this worktree at `2b891117`.
- No invented requirements, scope, or decisions. Quotes carry their citation line. The Gherkin scenarios at § 3 are quoted from the spec, not authored.
- No "we should" or "the change will" sentences. This document reports; the proposal decides. The forks at § 10 are explicitly named, not chosen.
- English, neutral/professional register. Code identifiers in backticks. No regional phrasing.