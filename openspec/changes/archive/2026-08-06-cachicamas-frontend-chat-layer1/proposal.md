# Proposal: cachicamas-frontend-chat-layer1

> **Status**: DRAFT
> **Change ID**: `cachicamas-frontend-chat-layer1`
> **Date**: 2026-08-06
> **Artifact store**: hybrid — file at `openspec/changes/cachicamas-frontend-chat-layer1/proposal.md` AND Engram topic `sdd/cachicamas-frontend-chat-layer1/proposal`
> **Depends on** (explore): `openspec/changes/cachicamas-frontend-chat-layer1/explore.md` (Engram #2575)
> **Project rules**: [`openspec/AGENTS.md`](../../AGENTS.md) — frontend Qwik 1.20 + Vite 7.3.1 + vitest; strict TDD red→green→refactor at apply time; vitest specs (`*.spec.{ts,tsx}`) are the red-green target for frontend.
> **Locked preflight**: `pace = auto` · `artifact_store.mode = both` · `delivery_strategy = single-pr` · `review_budget_lines = 800` · `chain_strategy = n/a` · `strict_tdd = true`. Forward `strict_tdd: true` to `sdd-apply` and `sdd-verify`. These are orchestrator-cached and NOT reopened by this proposal.
> **User-locked direction** (explore review): **Vitest-only verification, no dev-mode stub.** Recorded SSE transcripts as the red-green target. `pnpm dev` surfaces a clear `"backend not wired — see PR for backend wire"` error honestly. No backend stub in this PR.

---

## 1. Why

The cachicamas agent stack (doc 0001 §2.2, lines 153–218) draws three frontend slots — TUI, Print mode, and a reserved `Future: IDE / RPC` — all of which are `subgraph FE["Frontends — consume events only"]`. **No chat UI exists today.** Per doc 0001 §5.1 (lines 493–506) Layer 3's ports (`ToolSource`, `SkillSource`, `PromptSource`, `PermissionPolicy`, `Sandbox`, `PriceTable`) are wired with at least one implementation each, but Layer 3 has no `cmd/cachicamas` composition root on disk (doc 0001 §5.2, lines 518–535) and therefore exposes nothing on a port. Per doc 0001 §8 (lines 630–649) the TUI is explicitly deferred; "Print mode is the minimum viable frontend and the one that proves the event stream is sufficient" — but Print mode does not exist either. A web chat is the next obvious minimum-viable frontend and the architecture already treats it as a fourth consumer of the `CodingSessionEvent` stream. The user's framing ("you MUST to implement only the consume of this service"; "the back is who controls the keys and everything else") sets the ownership line: this change **only** builds the consumer; the Layer 3 HTTP surface is a separate, follow-up backend change. Verification on this PR is therefore mocked-SSE-transcript-driven (vitest red-green target), per the user's explore-review decision; a real end-to-end demo waits for the backend HTTP surface (cite explore.md R1).

**User-stated intent** (verbatim from the orchestrator prompt): *"you MUST to implement only the consume of this service … if something missing in the back just let me know but do not work in back … the back is who controls the keys and everything else … you just MUST to continue calling the backend services as usual in this frontend project."*

## 2. What changes (one-line per artifact)

| Artifact | One-line | Mirrors |
|---|---|---|
| `frontend/src/lib/chat-api.ts` | The chat's `ApiResult<T>` plus two helpers — `openChatTurn(input, onEvent, onError)` (POST + EventSource) and `cancelChatTurn(turnId)` (DELETE). Reuses `parseSSEResponse` and the `serverAwareFetch` plumbing. | `lib/prompts-api.ts`, `lib/skills-api.ts`, `lib/api.ts:896-977` |
| `frontend/src/components/chat/chat-window.tsx` | Visual surface: scrollable message list, streaming assistant bubble (re-renders accumulated buffer through `renderSanitizedMarkdown` per delta), input row, send + cancel buttons. Reads signals owned by `use-chat-stream`. | `components/workspace-sync-card/workspace-sync-card.tsx` (signal-driven render) |
| `frontend/src/components/chat/message-bubble.tsx` | Stateless bubble for one message — user vs assistant variant, sanitized markdown body, "thinking…" pill for unobserved reasoning. | `components/markdown-preview/` |
| `frontend/src/components/chat/chat-input.tsx` | Textarea + Send + Cancel; disabled state derived from `use-chat-stream` signals. | `components/ui/button/` |
| `frontend/src/components/chat/use-chat-stream.ts` | The Qwik hook — owns `useSignal`s for messages + streaming buffer + connection state; opens the EventSource in `useVisibleTask$` with `cleanup()`; POSTs via `stateChangingFetch`; cancellation through the `DELETE` route. | `components/workspace-sync-card/use-sync-status.ts:65-141` |
| `frontend/src/components/chat/__fixtures__/single-turn.sse` | Recorded SSE transcript — byte-identical to what a future backend must emit; pinned in vitest. | `lib/api.sse.spec.ts:75-307` (FakeES pattern) |
| `frontend/src/routes/chat/index.tsx` | Route shell — `onRequest` runs `setSsrCookieHeader` → `requireAuthRedirect` → `requireOwnboarding`, then renders `<ChatWindow />`. | `routes/home/index.tsx:39-51` |
| `frontend/src/routes/chat/layout.tsx` | Auth guard — `head.title` = "Chat — Cachicamas"; sidebar link from `routes/home/`. | `routes/home/layout.tsx` |

Plus one spec per source file (`*.spec.ts` / `*.spec.tsx`).

## 3. Impact / scope

| Area | Impact | Description | Mirrored pattern |
|---|---|---|---|
| `frontend/src/lib/chat-api.ts` | **New** | Chat client: SSE open + DELETE cancel, `ApiResult<T>` shape | `lib/prompts-api.ts:124-209`, `lib/api.ts:896-977` |
| `frontend/src/lib/chat-api.spec.ts` | **New** | Vitest — FakeES + recorded transcript assertion | `lib/api.sse.spec.ts:75-307` |
| `frontend/src/components/chat/chat-window.tsx` | **New** | The visual surface; signal-driven render | `components/workspace-sync-card/workspace-sync-card.tsx` |
| `frontend/src/components/chat/chat-window.spec.tsx` | **New** | Vitest — renders with stub hook, asserts delta accumulation + cancel | — |
| `frontend/src/components/chat/message-bubble.tsx` | **New** | Stateless bubble; markdown + sanitization | `components/markdown-preview/` |
| `frontend/src/components/chat/chat-input.tsx` | **New** | Textarea + Send/Cancel; derived disabled state | `components/ui/button/` |
| `frontend/src/components/chat/chat-input.spec.tsx` | **New** | Vitest — disabled states, submit propagation | — |
| `frontend/src/components/chat/use-chat-stream.ts` | **New** | The Qwik hook — `useVisibleTask$` + cleanup + POST + cancel | `components/workspace-sync-card/use-sync-status.ts:65-141` |
| `frontend/src/components/chat/use-chat-stream.spec.ts` | **New** | Vitest — mocked EventSource + mocked `stateChangingFetch` | `lib/api.sse.spec.ts:75-307` |
| `frontend/src/components/chat/__fixtures__/single-turn.sse` | **New** | Recorded transcript — pinned wire envelope | — |
| `frontend/src/routes/chat/index.tsx` | **New** | Route shell + `onRequest` guard chain | `routes/home/index.tsx:39-51` |
| `frontend/src/routes/chat/layout.tsx` | **New** | Auth gate + head + sidebar link | `routes/home/layout.tsx` |
| `frontend/src/routes/chat/index.spec.tsx` | **New** | Vitest — anon → `SignInRequiredCard`; SSR snapshot | — |
| `frontend/src/components/example/` (or `routes/home/`) | **Modified** (≤ 1 line) | One-line CTA from `/home` → `/chat` (defer to `sdd-propose` review) | explore.md §5.2 |
| `backend/`, `openspec/specs/`, `package.json`, `go.mod` | **None** | Hard rules | — |

## 4. Out of scope

| Excluded | Owner | Why not here |
|---|---|---|
| **Any backend code, including a stub** | A separate backend change | User explicit: *"if something missing in the back just let me know but do not work in back"*; explore.md R1 — vitest-only path means no stub server either |
| **Provider / model / credential selection in frontend** | Backend (`cmd/cachicamas`) | User: *"the back is who controls the keys and everything else"* |
| **Tool-call rendering, subagent nesting, permission protocol, streaming reasoning deltas, multimodal content** | Doc 0001 §8 + §7 G1/G7/G12 — *seam now*, no v1 impl | explore.md §6 |
| **Session persistence across reloads** | Layer 3 (doc 0001 §5.2) | Reload = empty chat in v1; document the boundary |
| **Real end-to-end demo against a live backend** | Backend HTTP-surface change (separate PR) | Explore.md R1 — backend gap is real and explicitly NOT closed in this PR |
| **Cost / usage display, provider/model picker UI, branching UI** | Doc 0001 §7 G10 + §8 + §7 G3 | Explore.md §6 |
| **New top-level frontend deps** (`@ai-sdk`, `openai`, `eventsource-parser`, …) | — | Explore.md §3: existing primitives are sufficient |
| **Any change to `backend/agent/src/ai/openaicompat/openrouter/` or Layer 1 packages** | Parallel OpenRouter change | Strict ownership: this change consumes, does not produce |
| **`openspec/specs/` modifications** | Existing `api-error-envelope`, `frontend-auth`, `frontend-frontend-csrf`, `frontend-markdown-rendering` are read-only contracts | §7 below |

## 5. Decisions

| ID | Decision | Basis | Trade-off rejected | Approved by | Date |
|---|---|---|---|---|---|
| **D1** | Wire format = **Server-Sent Events** (`text/event-stream`), reusing `frontend/src/lib/api.ts:831-881` `parseSSEResponse` and the `useVisibleTask$` subscriber pattern at `use-sync-status.ts:65-141`. | Existing primitives proven (`api.sse.spec.ts:13-307` covers parser + auto-close-on-terminal-status + null-marker). | **WebSocket** — no existing pattern; would re-implement reconnect / cancel / keepalive. | braejan (explore review) | 2026-08-06 |
| **D2** | HTTP shape = `POST /api/agent/turns` (opens a turn, returns `{ turn_id, stream_url }`); `DELETE /api/agent/turns/:id` (cancel). | Doc 0001 §4.2 (lines 449-455) — *"The cancellation tree. A user interrupt (abort this turn, keep the session) and a shutdown (flush and exit) are different signals and must remain distinguishable. Subagents inherit cancellation from their parent."* Cancel must be a discrete wire signal, not just a stream-close. | Long-poll, single-shot GET with replay, or close-on-disconnect only (loses the cancel direction). | braejan (explore review) | 2026-08-06 |
| **D3** | Auth = existing cookie session + CSRF defense-in-depth, identical to `prompts-api.ts` and `csrf.ts:28-39` `stateChangingFetch`. | Backend already authenticates the SSE route the same way it authenticates `subscribeWorkspaceSyncStream` (`lib/api.ts:907` sends `{ withCredentials: true }`). No new plumbing. | Bearer header / token storage / refresh flow — would widen the auth surface in `frontend/` and conflict with "back is who controls the keys". | braejan (explore review) | 2026-08-06 |
| **D4** | Verification posture = **vitest-only**. Recorded SSE transcripts as fixture files (`__fixtures__/single-turn.sse`). No dev-mode stub. `pnpm dev` surfaces `"backend not wired — see PR for backend wire"` honestly when a user submits a prompt without a backend. | User explicit at explore review; explore.md R1 — option (b). The frontend MUST NOT fabricate wire responses; honest failure is a feature, not a bug. | (a) ship a stub HTTP server in this PR — adds dev-only code that drifts from real wire; (c) depend on a backend PR first — blocks this PR indefinitely. | braejan (explore review) | 2026-08-06 |
| **D5** | Markdown rendering = reuse `frontend/src/lib/markdown.ts:65-67` `renderSanitizedMarkdown` (marked + isomorphic-dompurify). Re-render the accumulated buffer on each delta event. | Already pinned in deps; sanitize allowlist is already tight. | A heavier chat UI lib (`@assistant-ui/react`, `react-chat-elements`) — would widen deps and conflict with Qwik. | braejan (propose) | 2026-08-06 |
| **D6** | Route placement = `routes/chat/index.tsx` at `/chat`, gated by `require-auth-redirect.ts`. Sidebar link from `routes/home/`. | Per doc 0001 §2.2 row `CUSTOM`, a frontend slot has its own route. Top-level `/chat` keeps future subagent / multi-session UI unconstrained. | Nesting under `/home/chat` — cleanest inheritance but limits future UI shape. | braejan (propose) | 2026-08-06 |
| **D7** | Expected SSE event names (frontend expects; future backend MUST match — this is an EXPECTED contract, not invented vocabulary here): `event: message.start` · `event: message.delta` with `data: {"index": N, "delta": "..."}` · `event: message.end` · `event: turn.end` (terminal, closes the stream). | Doc 0001 §4.3 (lines 460-487) — *"Deltas carry an index, never a snapshot of the accumulated message. Consumers accumulate."* The shape parallels the canonical wire template at `backend/database_administrator/src/interfaces/http/sync_stream_handler.go:116-278` (snapshot-on-connect, then deltas, then terminal close) but with `message.delta` events instead of sync-status snapshots. **No new vocabulary invented by this change** — the names above mirror doc 0001 §4.3's *Message lifecycle* family. | WebSocket frames / binary frames / JSON-RPC — would not reuse `parseSSEResponse`. | braejan (propose) | 2026-08-06 |

## 6. PR shape (single-PR per preflight)

**Target**: `main`. **Branch**: `feat/frontend-chat-layer1`. **Single PR, no chain** (per `delivery_strategy = single-pr`).

### Per-file forecast (lines, naive → corrected ×2.5 midpoint)

| File | Purpose | Naive | ×2.5 mid |
|---|---|---|---|
| `frontend/src/lib/chat-api.ts` | client + types + typed `ChatStreamEvent` union | ~80–120 | ~200–300 |
| `frontend/src/lib/chat-api.spec.ts` | FakeES + transcript assertion + cancel + offline | ~150–250 | ~375–625 |
| `frontend/src/components/chat/chat-window.tsx` | visual surface | ~100–150 | ~250–375 |
| `frontend/src/components/chat/chat-window.spec.tsx` | delta accumulation + cancel | ~80–150 | ~200–375 |
| `frontend/src/components/chat/message-bubble.tsx` | bubble | ~40–80 | ~100–200 |
| `frontend/src/components/chat/chat-input.tsx` | input row | ~60–120 | ~150–300 |
| `frontend/src/components/chat/chat-input.spec.tsx` | disabled states + submit | ~60–120 | ~150–300 |
| `frontend/src/components/chat/use-chat-stream.ts` | Qwik hook (useVisibleTask$ + cleanup + POST + cancel) | ~60–100 | ~150–250 |
| `frontend/src/components/chat/use-chat-stream.spec.ts` | mocked EventSource + mocked POST | ~80–140 | ~200–350 |
| `frontend/src/routes/chat/index.tsx` | route shell + guard chain | ~40–80 | ~100–200 |
| `frontend/src/routes/chat/layout.tsx` | auth gate + head + sidebar link | ~20–40 | ~50–100 |
| `frontend/src/routes/chat/index.spec.tsx` | anon → SignInRequiredCard + SSR snapshot | ~60–120 | ~150–300 |
| `frontend/src/components/chat/__fixtures__/single-turn.sse` | recorded transcript | ~10–30 | (excluded — fixture bytes) |

**Total naive**: ~840–1500 lines. **Total corrected ×2.5 mid**: **~2100–3750 lines**. **Exceeds the 800-line `review_budget_lines` cap.** Per `delivery_strategy = single-pr` this is the user's choice and the proposal records it honestly.

**User must choose at `sdd-tasks` review (before that phase runs):**

- **(a) accept the over-budget as a maintainer-approved `size:exception`** — single PR with the forecast above; reviewers accept the cognitive load.
- **(b) split anyway** into PR #1 (client + tests + fixtures, no UI — the wire contract lands first) and PR #2 (UI consuming the client, with mocked client imported).

This proposal **does not pick**. Decision is surfaced to the user at `sdd-tasks` per the orchestrator's locked preflight.

## 7. Capabilities

> Contract between this proposal and the spec phase. `sdd-spec` reads this to know which spec files to create or update.

### New Capabilities

- **`frontend-chat-layer1`**: the chat UI as the fourth frontend consumer of the `CodingSessionEvent` stream (doc 0001 §2.2 row `CUSTOM`). Required scenarios:
  1. Submit a prompt → `POST /api/agent/turns` returns `{ turn_id, stream_url }` → `EventSource` opens on `stream_url` → text `message.delta` events accumulate in the assistant bubble → `turn.end` closes the stream → user sees the full message.
  2. Submit a prompt → user navigates away (route unmount) → `useVisibleTask$` cleanup fires → `DELETE /api/agent/turns/:id` issued (or `EventSource.close()` + DELETE for SSR-disconnected) → no orphaned request on the backend.
  3. No auth cookie → `requireAuthRedirect` short-circuits in `onRequest` → `SignInRequiredCard` rendered.
  4. Backend returns 503 mid-stream → bubble surfaces the typed `ApiError` inline (no client-side retry — retry is a harness concern per doc 0001 §4.1 lines 425-427 *"it reports a typed error; the harness decides"*).
  5. `pnpm dev` with no backend reachable → submitting a prompt surfaces `"backend not wired — see PR for backend wire"` honestly (the vitest-only decision D4).
  6. `message.start` arrives without a matching `message.end` (browser disconnects mid-stream) → no crash, no leak; the next mount opens a fresh `EventSource`.

→ New spec file: `openspec/specs/frontend-chat-layer1/spec.md`.

### Modified Capabilities

- **None.** `api-error-envelope`, `frontend-auth`, `frontend-frontend-csrf`, `frontend-markdown-rendering` are read-only contracts the chat consumes. No requirement of any existing spec changes.

## 8. Approach

1. **Compose, do not reimplement.** Reuse `lib/api.ts`'s `ApiResult<T>`, `safeFetch`, `serverAwareFetch`, `parseSSEResponse`, `withSsrCookieHeader` (lines 191–252, 831–881). Reuse `lib/markdown.ts:65-67` `renderSanitizedMarkdown` for assistant text. Reuse `require-auth-redirect.ts` for the route guard (per `routes/home/index.tsx:39-51`). Reuse `useVisibleTask$` from the workspace-sync pattern (`use-sync-status.ts:65-141`).
2. **Mock at the wire, not at the function.** The vitest spec asserts the SSE event names + data shapes against a recorded transcript fixture (`__fixtures__/single-turn.sse`). This is the strict-TDD discipline applied to a streaming endpoint — red is "spec fails because the client doesn't decode the event name"; green is "client decodes it"; refactor is "extract a typed `ChatStreamEvent` union".
3. **Honest dev failure.** The `chat-api.ts` client detects SSR-vs-browser and on the browser side, when the POST fails or the `EventSource` immediately errors, surfaces a single typed `ApiErrorKind = "offline"` (already in `lib/api.ts:96-110`) with a `message` that reads **"backend not wired — see PR for backend wire"**. No silent retry. No fabricated response. Per D4.
4. **Cancel is a wire signal, not just a stream-close.** `use-chat-stream` exposes `cancel(): void` which fires the DELETE route AND closes the `EventSource`. The `DELETE /api/agent/turns/:id` is the cancel direction the backend's cancellation tree needs (doc 0001 §4.2). The frontend's job is to send it on user intent and on `useVisibleTask$` cleanup.
5. **Markdown re-render on each delta is acceptable in v1.** `marked` is sync and small; buffer grows linearly with tokens; the optimization surface (throttle on a counter signal rather than the buffer text) is recorded as R5 in explore.md and deferred until measured.

## 9. Affected areas

All paths absolute from worktree root `/Users/braejan/workspace/witsaba/repositories/cachicamas-worktrees/feat-frontend-chat-layer1`:

```
openspec/changes/cachicamas-frontend-chat-layer1/proposal.md          (this file — new)
frontend/src/lib/chat-api.ts                                          (new)
frontend/src/lib/chat-api.spec.ts                                     (new)
frontend/src/components/chat/chat-window.tsx                          (new)
frontend/src/components/chat/chat-window.spec.tsx                     (new)
frontend/src/components/chat/message-bubble.tsx                       (new)
frontend/src/components/chat/chat-input.tsx                           (new)
frontend/src/components/chat/chat-input.spec.tsx                      (new)
frontend/src/components/chat/use-chat-stream.ts                       (new)
frontend/src/components/chat/use-chat-stream.spec.ts                  (new)
frontend/src/components/chat/__fixtures__/single-turn.sse             (new — recorded transcript)
frontend/src/routes/chat/index.tsx                                    (new)
frontend/src/routes/chat/layout.tsx                                   (new)
frontend/src/routes/chat/index.spec.tsx                               (new)
frontend/src/components/example/  OR  frontend/src/routes/home/       (≤1 line CTA — defer to sdd-propose review)
```

No files under `backend/`, `openspec/specs/`, `package.json`, `go.mod`, or `docker-compose.yaml` are touched.

---

## Risks (carried from explore.md, restated for propose review)

| # | Risk | Severity | Mitigation |
|---|---|---|---|
| **R1** | Backend HTTP surface does not exist (`cmd/cachicamas` + `/api/agent/turns`) — no real end-to-end possible until a separate backend PR lands. | **Blocker for live demo only; not for this PR** | D4: vitest-only with recorded SSE transcripts. `pnpm dev` surfaces an honest typed `offline` error. Cite doc 0001 §5.2. |
| **R2** | Forecast exceeds `review_budget_lines = 800` (corrected ~2100–3750 lines). `delivery_strategy = single-pr` is the user's choice. | **High** (review focus) | Surfaced at §6 as a user-confirmed fork: (a) `size:exception`, or (b) split into client-first / UI-second PRs. Decision deferred to `sdd-tasks` review. |
| **R3** | SSE in Qwik has known SSR-vs-browser quirks — `EventSource` is a browser global; `useVisibleTask$` runs on hydration. The first interaction must work on the FIRST browser visit, not the second. | Medium | Mirror `use-sync-status.ts:115-138` verbatim; spec covers the SSR-no-EventSource path. |
| **R4** | Strict-TDD mock-fixture authoring burden — recorded SSE transcripts must be byte-correct to the expected wire shape (D7), otherwise the spec proves a fictional contract. | Medium | Transcript is a checked-in fixture with one canonical turn (`message.start` → 5 `message.delta`s → `message.end` → `turn.end`); property-style assertion over the bytes (per OpenRouter §3 R2 lesson). |
| **R5** | Coupling with the parallel `add-openrouter-first-provider` change. Frontend is independent of vendor — this PR can land first OR the OpenRouter PR can land first. | Low | `delivery_strategy = single-pr` on this PR is independent of the OpenRouter PR's `auto-chain` topology. If OpenRouter lands first, this PR's `pnpm dev` still shows honest offline (D4) until a backend HTTP route exists. |

---

## Rollback plan

Single PR. Revert merge → no behavior change anywhere outside `frontend/src/`; no migration data, no DB schema, no env vars, no `go.mod` delta, no production route. The chat route becomes 404 and the home-page CTA (if added) reverts to its prior link. The recorded SSE transcript fixture is data, not behavior — its removal is harmless.

Partial rollback: reverting only the spec files (`.spec.ts`/`.spec.tsx`) leaves the implementation untested — same posture as any untested feature. If the spec surface is rejected, the correct move is to reject the whole PR and re-propose, not to strip the tests in isolation.

## Dependencies

- **`openspec/specs/frontend-runtime`** (read-only contract) — `useVisibleTask$` semantics, signal ownership, cleanup discipline.
- **`openspec/specs/frontend-auth`** (read-only contract) — `requireAuthRedirect`, cookie + CSRF chain.
- **`openspec/specs/api-error-envelope`** (read-only contract) — `ApiResult<T>` five-kind union (`validation`/`conflict`/`not_found`/`server`/`offline`) the chat client mirrors verbatim.
- **`openspec/specs/frontend-frontend-csrf`** (read-only contract) — `stateChangingFetch` wrapper the chat's POST uses.
- **`openspec/specs/frontend-markdown-rendering`** (read-only contract) — `renderSanitizedMarkdown` the assistant bubble uses.
- **None of the above are modified.** This PR adds `frontend-chat-layer1` as a consumer.

## Success criteria

- [ ] Single PR lands `feat/frontend-chat-layer1` → `main` with the file list in §9, no other files touched.
- [ ] `frontend/package.json` and `go.mod` are unchanged.
- [ ] `pnpm --filter @cachicamas/frontend test:ci` passes (vitest) with the new specs red→green per strict TDD.
- [ ] `pnpm --filter @cachicamas/frontend build.types` passes.
- [ ] `pnpm --filter @cachicamas/frontend dev` renders `/chat` for an authed user; submitting a prompt surfaces the typed `"backend not wired"` error (D4 honest dev failure).
- [ ] No backend file is modified; no new top-level frontend dependency is added.
- [ ] Recorded SSE transcript fixture pins the wire envelope (D7).
- [ ] Author confirms fork choice at `sdd-tasks` review: `(a) size:exception` OR `(b) split client-first / UI-second`.

## Notes for the following phases

- **`spec.md`** — the system under test is the chat UI as a frontend consumer. Requirement IDs `R-FCL-0NN` (frontend-chat-layer1). Six required scenarios in §7. Two scenarios must constrain the *expected contract*, not just the implementation — scenario 5 (D4 honest failure) and the event-name shape (D7) — because a list of test-passing scenarios that doesn't pin the wire envelope passes a spec that checks only behavior, and cannot be re-anchored when the backend lands.
- **`design.md`** — owns the typed `ChatStreamEvent` union, the `use-chat-stream` signal graph, the cancel race (DELETE vs EventSource close ordering), and the SSR-cookie handoff for the initial POST.
- **`tasks.md`** — five RED-GREEN-REFACTOR slices, one per leaf (client → hook → input → bubble → window → route), plus a final wiring slice. Vitest red on each slice before any green code lands. Forward `strict_tdd: true`.
- **`verify-report.md`** — must override the Go-centric `openspec/config.yaml` `test_command` (`go test ./...`) with `pnpm --filter @cachicamas/frontend test:ci` plus `pnpm --filter @cachicamas/frontend build.types` (per explore.md R7).