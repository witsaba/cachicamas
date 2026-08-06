# Explore: cachicamas-frontend-chat-layer1

> **Status**: exploration complete — ready for `sdd-propose`.
> **Change name**: `cachicamas-frontend-chat-layer1`
> **Working tree**: `feat/frontend-chat-layer1` (fresh from `main`)
> **Strict TDD**: enabled (`openspec/config.yaml` `apply.tdd: true`; frontend red-green target = vitest, `*.spec.{ts,tsx}`).

---

## 1. Context

The user wants a simple Qwik chat UI that **consumes** the cachicamas Layer 1
agent stack already documented in
[`docs/architecture/0001-cachicamas-agent-stack-v2.md`](../0001-cachicamas-agent-stack-v2.md)
(§2.2 *Layer view*: Frontends — consume events only; §5 *Layer 3 — the coding
application*: the AgentHarness is stateful, fronts the model adapter, and is the
seam a user interacts with). The chat is the **first consumer** of the
`CodingSessionEvent` stream (doc 0001 §2.2). Per §8 *Non-goals*, a TUI was
explicitly deferred ("Print mode is the minimum viable frontend"); a web chat
is the next obvious minimum-viable frontend, and the architecture already
treats `Future: IDE / RPC` as a frontend slot (§2.2 row `CUSTOM`).

The user's instruction is unambiguous about scope: **consume the backend; do
not touch the backend**. The backend keeps ownership of credentials, provider
selection, model routing, and every Layer 1/2/3 policy decision. The frontend
is a pure consumer.

---

## 2. Quick path

1. **Read the gap (§4)** — there is no `/chat` HTTP endpoint anywhere in the
   repo today. The frontend cannot wire to a real backend until one exists.
2. **Reuse the SSE plumbing (§3, §5)** — `lib/api.ts` already ships a
   production-grade SSE parser, auto-close-on-terminal-status, and a Qwik
   hook pattern (`use-sync-status.ts`) that the chat will mirror.
3. **Decide the development backend posture (§6 risks)** — propose must
   surface two options: stub a fake SSE server in dev OR land a tiny backend
   PR first. **Do not silently assume the backend will land in time.**

---

## 3. Frontend findings — what already exists

The Qwik app (`frontend/`, Qwik 1.20 + Vite 7.3.1, vitest for unit tests,
Playwright for e2e) is a mature consumer of the `database_administrator`
service on `http://localhost:8080`. Every primitive the chat needs is already
present; **the chat is a composition of existing pieces, not a new framework**.

| Need | Exists at | Notes |
|---|---|---|
| Discriminated API error union (`ApiResult<T>`, 5 kinds) | `frontend/src/lib/api.ts:96-110` | `validation`/`conflict`/`not_found`/`server`/`offline`; the chat client must mirror this exact shape |
| SSR-cookie forwarding + dual-runtime fetch | `frontend/src/lib/api.ts:191-252` (`serverAwareFetch`, `withSsrCookieHeader`, `ssrBaseUrl`) | The chat will reuse this verbatim — `useTask$` on SSR must carry the inbound `authjs.session-token` to the backend |
| Form-encoded + JSON body helpers, envelope mapping | `frontend/src/lib/api.ts:344-389, 698-727`; `frontend/src/lib/prompts-api.ts:124-209`; `frontend/src/lib/skills-api.ts:123-237` | Three sibling services. The chat will be a fourth sibling with the same conventions |
| CSRF defense-in-depth (`X-Requested-With: XMLHttpRequest`) | `frontend/src/lib/csrf.ts:28-39` (`stateChangingFetch`) | The POST that opens a chat turn is state-changing; use this wrapper |
| **SSE parser** (one-call reusable) | `frontend/src/lib/api.ts:831-881` (`parseSSEResponse`) | Already handles data frames, comment keepalives, malformed JSON. Spec coverage in `api.sse.spec.ts:13-67` |
| **Browser EventSource subscriber** with auto-close | `frontend/src/lib/api.ts:896-977` (`subscribeWorkspaceSyncStream`) | Closes on terminal status, closes on null marker, suppresses spurious post-close `onerror`. Spec coverage in `api.sse.spec.ts:75-307` |
| Markdown rendering with sanitization | `frontend/src/lib/markdown.ts:65-67` (`renderSanitizedMarkdown`) | Uses `marked` (sync) + `isomorphic-dompurify` with a tight allowlist. Streaming markdown chunks are pre-existing problem-space; the chat will buffer → re-render on each event |
| Cookie policy (HttpOnly, SameSite=Lax, Path=/) | `frontend/src/lib/auth-cookie-config.ts:33-37` | No frontend-side changes; just inherits |
| Route guard chain (`requireAuthRedirect` → `requireOwnboarding`) | `frontend/src/lib/require-auth-redirect.ts`, `require-ownboarding.ts`; consumed in `frontend/src/routes/home/index.tsx:39-51` | The chat route must do the same `onRequest` dance (capture cookie via `setSsrCookieHeader` first, then `requireAuthRedirect`, then `requireOwnboarding`) |
| SSR cookie capture | `frontend/src/lib/ssr-cookie-context.ts`; `frontend/src/lib/with-ssr-cookie.ts` | Wire pattern lives at `frontend/src/routes/home/index.tsx:42` — the chat route copies this |
| `useSession`, `useSignIn`, `useSignOut` | `frontend/src/routes/plugin@auth.ts` | Already used by `layout.tsx` and every protected route — the chat will follow |
| Reusable hook for "subscribe to a server-pushed event stream" | `frontend/src/components/workspace-sync-card/use-sync-status.ts:65-141` | This is the canonical pattern: `useSignal` for the live value, `useVisibleTask$` to open the EventSource, `cleanup()` to close it, error handler triggers a one-shot recovery refresh. **The chat's `useChatStream` will be a near-copy with different parse + abort logic** |
| Markdown preview component (server-safe, sanitized) | `frontend/src/components/markdown-preview/` | Reuse for streaming-message rendering |
| `Button` primitive | `frontend/src/components/ui/button/` | Use for send button |
| Authed home route shape | `frontend/src/routes/home/index.tsx` | The chat can either nest as `/home/chat` (cleanest — inherits the home guard) or stand alone at `/chat` with its own `onRequest` |
| Vitest setup | `frontend/vite.config.ts:34-45` (`test.exclude: ["e2e/**"]`) | Confirmed — `**/*.spec.{ts,tsx}` are picked up; `e2e/**` runs separately under Playwright |
| Test runner | `frontend/package.json:27-28` (`test.unit`, `test:ci`) | `pnpm test:ci` = `vitest --run` |

**No new top-level dependency is needed.** `frontend/package.json` already has
`marked`, `isomorphic-dompurify`, `zod`, and the `EventSource` global (browser)
— no streaming library, no AI SDK, no fetch library. The chat composes
existing primitives only.

---

## 4. Backend findings — and the gap

### 4.1 What exists today (verified by direct read of `cmd/server/main.go` and `interfaces/http/`)

- **`database_administrator` (Go 1.26.3, Echo v5) on port 8080.** Authed CRUD
  HTTP surface: `/health`, `/organizations`, `/workspaces`, `/workspaces/:id`,
  `/workspaces/:id/sync`, `/workspaces/:id/sync/stream` (SSE — see
  `backend/database_administrator/src/interfaces/http/sync_stream_handler.go`
  for the canonical SSE wire pattern), `/prompts` (7 endpoints), `/skills`
  (7 endpoints), `/github/repos` (proxy), `/whoami`, `/identity/*`. Cookie
  auth via `IdentityFromCookie` middleware. SSE pattern is already proven:
  see `sync_stream_handler.go:130-200` for `Content-Type: text/event-stream`,
  `X-Accel-Buffering: no`, keepalive ticker at 15s, snapshot-on-connect,
  close-on-terminal-status semantics. The chat's wire format will mirror this.
- **`backend/workspace_syncer`** — internal HTTP callback client only
  (`infrastructure/httpclient/callback_client.go`); no external surface; not
  relevant to the chat.
- **`backend/agent`** — Layer 1 (`src/ai/`) + Layer 1 conformance kit
  (`src/agenttest/`) only. **NO `cmd/`, no `interfaces/http/`, no echo, no
  `http.Server`.** The module imports nothing from the other two
  (`backend/agent/go.mod` is a 3-line file: `module … / agent`, `go 1.26.3`).
  This is by design — see doc 0001 §2.2 and
  [`docs/adr/0005-promote-agent-stack-to-own-module.md`](../../adr/0005-promote-agent-stack-to-own-module.md).

### 4.2 What `add-openrouter-first-provider` concretizes (verified by reading `proposal.md`)

The OpenRouter change is **backend-only** and explicitly out-of-scope for an
HTTP surface:

- §3 *Subsystems touched*: all writes are inside `backend/agent/src/ai/openaicompat/openrouter/`
  (a new sibling sub-package wrapping the existing generic adapter). The
  `openaicompat/` package itself is unchanged; `agenttest/` is unchanged;
  `go.mod` is unchanged (zero new `require` lines).
- §4 *Out of scope* explicitly names: "Layer 3 (`src/coding`) composition
  root", "AI-40 publication of the capability matrix", and the entire user-
  facing transport layer.
- §6 row Q2 documents the **expected** wiring pattern (Layer 3 reads env /
  config and passes an opaque bearer string into `openrouter.Config.Credential`)
  but does NOT implement Layer 3.

**Net effect on this change**: OpenRouter concretization makes the *adapter
factory* real, but it does not add any wire surface the frontend can call.
There is still nothing for the chat to talk to.

### 4.3 The backend gap — named

| Missing | Why it matters for this change | Owner (per doc 0001 §5.2) |
|---|---|---|
| **An HTTP server in `backend/agent`** | doc 0001 §2.2 draws `cmd/cachicamas` as the **only** composition root. Until `cmd/cachicamas/src/` exists, there is no process that wires `Provider → Harness → CodingSession` and exposes anything on a port. The chat literally has no backend to call. | Layer 3 composition root — `backend/agent/src/cmd/cachicamas/` (does not exist on disk today; doc 0002/0004 reserve the work) |
| **An authed HTTP route that opens a turn** | The chat needs `POST /agent/turns` (or similar) that accepts `{prompt, history?, options?}` and returns an SSE stream of normalized events. Mirror of `SyncStreamHandler` (`sync_stream_handler.go:116-278`) but for the `CodingSessionEvent` envelope (doc 0001 §4.3). | Same — `backend/agent/src/cmd/cachicamas` + an `interfaces/http/` handler if that composition root runs Echo (or a thin stdlib listener) |
| **An authed route that cancels a turn** | The chat's "Stop" button must close the underlying turn. Cancellation tree is a harness concern (doc 0001 §4.2); the wire needs `DELETE /agent/turns/:id` or a close-on-disconnect contract. | Same |
| **Optional: provider/model catalog** | The user said *"the back is who controls the keys"*; the chat does NOT need this in v1. A single default model behind the endpoint is sufficient. | Out of scope for this change (deferred — see §6) |

### 4.4 Wire sketch the chat's client will assume (NOT to be designed here)

> This is a sketch only; the actual wire contract lives in a future backend
> change. The chat's tests will mock this contract end-to-end.

```
POST /api/agent/turns        (authed; JSON body)
  body: { prompt: string, conversation_id?: string, options?: {...} }
  resp 202: { turn_id: "trn_…", stream_url: "/api/agent/turns/trn_…/events" }

GET /api/agent/turns/:id/events   (authed; text/event-stream)
  frame 1: snapshot of current turn state
  data: { kind: "lifecycle|message|tool|permission|cost|subagent", … }   (doc 0001 §4.3)
  : keepalive\n\n                       (every 15s)
  ─ close on terminal event or terminal status ─

DELETE /api/agent/turns/:id     (authed; user-initiated stop)
  204 on success; the SSE stream closes with a "cancelled" terminal event
```

The `SyncStreamHandler` at `sync_stream_handler.go:130-200` is the existing
template: same `Content-Type`, same `X-Accel-Buffering: no`, same keepalive
ticker, same snapshot-then-deltas cadence, same close-on-terminal discipline.

---

## 5. Reuse plan — files to create vs modify

**Target: ≤ 4 new files, ≤ 1 modified file.** Strict TDD writes the spec
FIRST in each case (frontend red = vitest spec failing for the right reason,
green = minimum implementation, refactor = clean up while green).

### 5.1 New files (proposed — sdd-propose may revise)

| File | Purpose | Pattern mirrored | Spec |
|---|---|---|---|
| `frontend/src/lib/chat-api.ts` | The chat's `ApiResult<T>` + the two SSE helpers: `openChatTurn(input, onEvent, onError)` (POST + EventSource), `cancelChatTurn(turnId)` (DELETE). Reuses `parseSSEResponse` and the `apiBaseUrl()` / `serverAwareFetch` plumbing from `lib/api.ts`. Mirrors the `lib/prompts-api.ts` + `lib/skills-api.ts` shape | `frontend/src/lib/prompts-api.ts`, `frontend/src/lib/skills-api.ts`, `frontend/src/lib/api.ts:896-977` | `frontend/src/lib/chat-api.spec.ts` (vitest) — records a transcript fixture, asserts every event is surfaced in order; asserts terminal closes the EventSource; asserts DELETE issues a fetch with `X-Requested-With: XMLHttpRequest`; asserts offline maps to `kind: "offline"`; asserts the `apiBaseUrl()` SSR vs browser split |
| `frontend/src/components/chat/chat-panel.tsx` | The visual surface: scrollable message list (user + assistant bubbles), streaming assistant bubble (renders each delta via `renderSanitizedMarkdown` on a buffer), input row with `Button` + textarea, send + cancel buttons. Reads signals populated by `useChatStream` | `frontend/src/components/workspace-sync-card/workspace-sync-card.tsx` for the signal-driven render pattern | `frontend/src/components/chat/chat-panel.spec.tsx` (vitest) — renders with a stub `useChatStream`, asserts user message appears on send, asserts streaming deltas accumulate, asserts cancel closes the stream |
| `frontend/src/components/chat/use-chat-stream.ts` | The Qwik hook: owns the `useSignal`s for messages + streaming buffer + connection state; opens the EventSource in `useVisibleTask$` with `cleanup()`; wires `stateChangingFetch` for the initial POST; handles abort via a controller-style unsubscribe | `frontend/src/components/workspace-sync-card/use-sync-status.ts:65-141` | `frontend/src/components/chat/use-chat-stream.spec.ts` (vitest) — mocked EventSource + mocked `stateChangingFetch`; asserts open + cleanup + cancel wiring |
| `frontend/src/routes/chat/index.tsx` | The route shell: `onRequest` does `setSsrCookieHeader` → `requireAuthRedirect` → `requireOwnboarding` (verbatim from `home/index.tsx:39-51`), then renders `<ChatPanel />`. `head.title` = "Chat — Cachicamas" | `frontend/src/routes/home/index.tsx` | `frontend/src/routes/chat/index.spec.tsx` (vitest) — asserts anon visitor renders `SignInRequiredCard`; asserts SSR snapshot of an empty chat panel |

### 5.2 Modified files (≤ 1)

| File | Change |
|---|---|
| `frontend/src/components/example/` or a one-line home-page CTA | Optional: a "Try the chat" link from `/home` to `/chat/`. **Defer to sdd-propose** — may be unnecessary if the user wants `/chat/` to be the discoverable entry from the avatar dropdown |

### 5.3 What is NOT being created

- No new `lib/markdown-streaming.ts` — reuse `renderSanitizedMarkdown` on the
  accumulated buffer (re-render per delta is cheap; `marked` is sync).
- No new SSE parser — `parseSSEResponse` already exists.
- No new `safeFetch` / `serverAwareFetch` — reuse from `lib/api.ts`.
- No new top-level deps.

---

## 6. Out of scope (so sdd-propose does not drift)

Per doc 0001 §8 *Non-goals for v1* and the user's *"I need you analyze this
code and understand how to use it the LLM calls"* framing (consume, not build):

| Excluded | Why |
|---|---|
| Tool-call rendering (display + click-to-expand argument/result JSON) | Doc 0001 §8 defers tool sources / MCP / subagents; chat v1 is text-in/text-out only. The events arrive but the UI ignores `kind: "tool"` |
| Permission protocol (`kind: "permission"` events) | Doc 0001 §7 G1 row is "seam now"; no v1 implementation. Chat v1 cannot ask the user for permission; if a turn would need one, it surfaces the event as a status pill (or hides it) |
| Subagent nesting | Doc 0001 §7 G7 is "seam now". Chat v1 flattens any subagent events into a single assistant message |
| Streaming reasoning deltas (`kind: "reasoning"`) | Doc 0001 §4.3 lists reasoning as a distinct content family; chat v1 does NOT surface reasoning text to the user (UI shows it as a collapsed "thinking…" pill or omits it entirely) |
| Multimodal content (image/audio) | Doc 0001 §8 explicitly defers — "Multimodal content beyond text. Image and audio discriminators exist … no constructor produces them" |
| Session persistence across reloads | Doc 0001 §5.2 ("append-only records with parent chains … under the user's home directory") is a Layer 3 concern; chat v1 keeps no client-side session. Reload = empty chat |
| Cost / usage display | Doc 0001 §7 G10 defers per-turn cost events to L2/L3; chat v1 ignores `kind: "cost"` |
| Provider / model picker UI | User said *"the back is who controls the keys"* — chat v1 picks nothing. Backend defaults to the configured model |
| Any HTTP work in `backend/agent` or `backend/database_administrator` | User explicitly forbade it. If a backend gap blocks development, the answer is a stub server, NOT a backend change in this PR |
| New top-level frontend deps (no `@ai-sdk`, no `openai`, no `eventsource-parser`) | The existing primitives are sufficient. A future change may revisit if streaming markdown re-render becomes a perf issue |
| Touching `backend/agent/src/ai/openaicompat/openrouter/` or any of the Layer 1 packages | Strict ownership: this change is a consumer, not a producer |

---

## 7. Risks

| # | Risk | Severity | Mitigation surface |
|---|---|---|---|
| **R1** | **Backend HTTP gap (§4.3) blocks real end-to-end testing.** No `/api/agent/turns` route exists; the chat cannot wire to a live backend during development. | **Blocker for `verify-report`** | sdd-propose must commit to **one** of: (a) ship a stub HTTP server in this PR (a single `routes/chat/_stub.ts` + a `chat-stub-server.ts` test fixture); (b) accept that verification is purely on mocked SSE transcripts and `pnpm test:ci` is the only test command; (c) make this change depend on a follow-up backend change that lands FIRST. **User must choose at proposal review.** |
| **R2** | **Strict TDD mocks the streaming wire — risk of green tests that don't reflect real wire.** The vitest spec records a JSON transcript fixture; if the wire envelope drifts (e.g. a future backend change moves to a different event-shape), the frontend breaks silently. | Medium | The spec records the wire envelope as a named `WireEvent` type in `chat-api.ts`; a property-style test asserts the parser survives a recorded byte transcript. Any wire change forces the spec to change first. The OpenRouter change has no wire surface to drift; the real drift hazard is the future Layer 3 composition root |
| **R3** | **SSE in Qwik has known quirks.** `EventSource` is a browser global; `useVisibleTask$` runs on hydration. SSR cannot subscribe. The "send a message, see streaming reply" flow must work on the **first** browser interaction, not on the second. | Medium | Mirror the proven pattern in `use-sync-status.ts:115-138` (useVisibleTask$ with cleanup). The initial POST that opens a turn goes through `stateChangingFetch` BEFORE the EventSource opens, so the chat's first interaction is "POST returns 202 + turn_id" → "open EventSource on the returned stream URL". Add a spec for the SSR-no-EventSource path (returns a no-op cleanup) |
| **R4** | **OpenRouter change merge ordering.** This change and `add-openrouter-first-provider` are independent worktrees. The frontend chat does not depend on OpenRouter specifically (it just POSTs to `/api/agent/turns`), but the user's framing ("the layer 1 and even in main exists the layer 1 implementation") implies they intend to test end-to-end against OpenRouter eventually. | Low | Both changes target `main` via separate PRs. sdd-propose will declare this PR's **coupling**: **independent** (frontend consumes the contract, not the vendor). If the OpenRouter PR lands FIRST, this PR can wire to its smoke-test credentials locally; if THIS PR lands FIRST, it ships with mocked wire and waits for the backend. Neither ordering blocks the other |
| **R5** | **Re-rendering performance on long streams.** `marked` re-parses the entire accumulated buffer on every delta event. For a 2000-token reply, that is 2000 re-parses of increasingly long text. | Low–Medium | Acceptable for v1 (text-first UI, simple buffer). Mitigation surface: throttle re-render to every N tokens OR use a `useTask$` with `track()` on a counter signal, NOT the buffer. **Defer optimization until measured.** Add a `[pin]` test that asserts the streaming path does NOT exceed the render budget on a recorded 2000-token transcript |
| **R6** | **Auth cookie on the SSE EventSource.** The browser must send `authjs.session-token` with the EventSource request; backend must accept it. `subscribeWorkspaceSyncStream` already does `{ withCredentials: true }` (`lib/api.ts:907`) and the sync endpoint accepts it on the auth-protected group. The chat's new SSE endpoint must follow the same wiring. | Low | Trivial — copy the wiring; the auth chain at the backend is the backend's job |
| **R7** | **`openspec/config.yaml` test_command is `go test ./...` — wrong tool for frontend.** The config is Go-centric; the chat's verify phase will not run that command. | Low | sdd-verify must override the test command per change (the orchestrator's preflight allows per-phase overrides). The canonical frontend test command is `pnpm --filter @cachicamas/frontend test:ci` plus `pnpm --filter @cachicamas/frontend build.types` |

---

## 8. Next recommended phase

**`sdd-propose`**, with the following prerequisites for that phase:

- This `explore.md` is committed (OpenSpec) + saved (Engram, topic
  `sdd/cachicamas-frontend-chat-layer1/explore`).
- The orchestrator's preflight already declares
  `delivery_strategy = single-pr` and `review_budget_lines = 800` — sdd-propose
  honors those (no chained PRs unless the budget forecast flips High at
  sdd-tasks).
- **sdd-propose MUST ask the user to resolve R1** before writing the proposal:
  *stub server / mocked-only verification / depend on a backend PR first?*
  This is the only fork in the road; everything else is mechanical.
- sdd-propose should produce the four new file spec skeletons, the route
  shell, and the test_command override for the frontend. It must NOT propose
  any change to `backend/agent`, `backend/database_administrator`, or the
  `openspec/specs/` tree (frontend-only).

---

## 9. References

- **Architecture (cited line numbers refer to the doc as read in this worktree)**:
  - `docs/architecture/0001-cachicamas-agent-stack-v2.md:153-218` (§2.2 Layer view — Frontends consume events only; Custom frontend slot)
  - `docs/architecture/0001-cachicamas-agent-stack-v2.md:233-281` (§2.3 Turn sequence — six load-bearing turn mechanics)
  - `docs/architecture/0001-cachicamas-agent-stack-v2.md:403-489` (§4.2 Harness + §4.3 Event envelope — six event families)
  - `docs/architecture/0001-cachicamas-agent-stack-v2.md:492-536` (§5.1 ports + §5.2 composition root — `cmd/cachicamas` is the only composition root)
  - `docs/architecture/0001-cachicamas-agent-stack-v2.md:630-649` (§8 Non-goals — TUI deferred, no multimodal, no session branching UI)
- **ADRs**:
  - `docs/adr/0004-adopt-tau-3-layer-agentic-architecture.md`
  - `docs/adr/0005-promote-agent-stack-to-own-module.md` (D1 dependency rule: agent module imports nothing from other modules)
- **OpenSpec context**:
  - `openspec/project.md` — stack + conventions (currently Go-centric; the chat's proposal overrides the test_command)
  - `openspec/config.yaml` — `apply.tdd: true` (forwarded to sdd-apply, sdd-verify)
  - `openspec/AGENTS.md` — strict TDD; vitest is the frontend red-green target
- **Adjacent in-progress change (independent, backend-only)**:
  - `openspec/changes/add-openrouter-first-provider/proposal.md` — concretizes the adapter; does NOT add an HTTP surface
- **Existing frontend files the chat mirrors**:
  - `frontend/src/lib/api.ts:831-977` (SSE parser + EventSource subscriber with auto-close)
  - `frontend/src/lib/api.sse.spec.ts:13-307` (full SSE spec coverage)
  - `frontend/src/lib/api.ts:191-252` (serverAwareFetch — SSR-vs-browser split)
  - `frontend/src/lib/csrf.ts:28-39` (stateChangingFetch — CSRF defense-in-depth)
  - `frontend/src/lib/markdown.ts:65-67` (renderSanitizedMarkdown)
  - `frontend/src/lib/ssr-cookie-context.ts` + `setSsrCookieHeader` (SSR cookie plumbing)
  - `frontend/src/components/workspace-sync-card/use-sync-status.ts:65-141` (the canonical "subscribe to a server-pushed event stream" hook pattern)
  - `frontend/src/routes/home/index.tsx:39-51` (the `onRequest` guard chain)
- **Existing backend SSE template** (NOT a chat route, but the wire pattern is identical):
  - `backend/database_administrator/src/interfaces/http/sync_stream_handler.go:116-278`
