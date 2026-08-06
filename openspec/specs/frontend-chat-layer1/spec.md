# Spec — `frontend-chat-layer1`

> **Domain**: `frontend-chat-layer1` · **Change**: `cachicamas-frontend-chat-layer1` · **Type**: New capability (the `CUSTOM` slot in [`docs/architecture/0001-cachicamas-agent-stack-v2.md` §2.2](../../../../docs/architecture/0001-cachicamas-agent-stack-v2.md) is reserved-but-unimplemented) · **Created**: 2026-08-06 · **Persistence**: hybrid (`openspec/specs/frontend-chat-layer1/spec.md` + Engram topic `sdd/cachicamas-frontend-chat-layer1/spec`).

## Purpose

Acceptance contract for the web chat UI as the **fourth frontend** consumer of the `CodingSessionEvent` stream (doc 0001 §2.2, lines 153–218 — the `FE["Frontends — consume events only"]` subgraph lists TUI, Print mode, and the reserved `CUSTOM` slot). The chat is a consumer only: it MUST NOT define Layer 3 ports, MUST NOT take ownership of credentials/provider/model, and MUST NOT touch `backend/` or `infra/`. Verification is **vitest-only** per the user-locked direction D4 — `pnpm --filter @cachicamas/frontend test:ci` against `*.spec.{ts,tsx}` is the red-green target (`openspec/AGENTS.md` *Strict TDD*).

Conventions: RFC 2119 keywords; Given/When/Then scenarios (project `openspec/config.yaml` `rules.specs`); wire format `text/event-stream` via `frontend/src/lib/api.ts:831-881` and `:896-977`; error envelope `ApiResult<T>` at `frontend/src/lib/api.ts:96-110`; markdown via `renderSanitizedMarkdown` at `frontend/src/lib/markdown.ts:65-67`; CSRF via `stateChangingFetch` at `frontend/src/lib/csrf.ts:28-39`; route guard from `frontend/src/routes/home/index.tsx:39-51`.

## Requirements

### REQ-1 — Submit-a-turn happy path

**Statement.** Submitting a prompt from `/chat` SHALL open `POST /api/agent/turns`, open an `EventSource` on the returned `stream_url`, accumulate `event: message.delta` text into a single assistant bubble, and close the stream on `event: turn.end`.

**Rationale.** Doc 0001 §2.2 lines 153–218 — frontends consume `CodingSessionEvent`. §4.3 invariant 1 lines 477–487 — *deltas carry an index, consumers accumulate*; this REQ fixes the consumer side. Implementation mirrors `subscribeWorkspaceSyncStream` (`frontend/src/lib/api.ts:896-977`).

**Scenarios.**

- **S-1.a** Given an authenticated user on `/chat` with empty state, when the user submits a prompt, then `POST /api/agent/turns` issues with `{ withCredentials: true }` (`serverAwareFetch`) and resolves to `{ turn_id, stream_url }`, then `EventSource(stream_url, { withCredentials: true })` opens, then every `message.delta` appends `data.delta` to the assistant bubble's buffer, then on `event: turn.end` `EventSource.close()` runs exactly once.
- **S-1.b** Given a frame whose `event:` name is not in `{ message.start, message.delta, message.end, turn.end }`, when the frame arrives, then the client ignores it and the buffer is unchanged.

### REQ-2 — Mid-stream cancellation

**Statement.** When `/chat` unmounts (or any code path closes the SSE connection for a reason other than a natural `turn.end`), the client SHALL issue `DELETE /api/agent/turns/:id` in addition to `EventSource.close()`.

**Rationale.** Doc 0001 §4.2 lines 450–455 — cancellation tree is harness-owned ("a user interrupt … and a shutdown … are different signals"). Closing only the EventSource leaves an orphaned turn. Cancel is a discrete wire signal (proposal D2).

**Scenarios.**

- **S-2.a** Given an open `EventSource` for `turn_id = T`, when the user navigates away from `/chat`, then `useVisibleTask$ cleanup()` runs and `DELETE /api/agent/turns/T` issues (via `stateChangingFetch` so `X-Requested-With: XMLHttpRequest` is set) and `EventSource.close()` runs exactly once.
- **S-2.b** Given an open turn, when the user clicks **Stop**, then `DELETE /api/agent/turns/T` fires with the same wrapper, and the input returns to "ready" state, and the partial assistant text remains visible.
- **S-2.c** Given `event: turn.end` arrives first, when the client closes per REQ-1, then no `DELETE` issues (the turn is already terminal).

### REQ-3 — Auth guard

**Statement.** Unauthenticated access to `/chat` SHALL redirect via the existing `setSsrCookieHeader → requireAuthRedirect → requireOwnboarding` chain, identical to `routes/home/`.

**Rationale.** Doc 0001 §2.2 places the chat as an authed consumer. The canonical chain at `frontend/src/routes/home/index.tsx:39-51` is the proven path — no new redirect target, no new cookie policy.

**Scenarios.**

- **S-3.a** Given no `authjs.session-token` cookie, when the orchestrator GETs `/chat`, then `requireAuthRedirect` throws and the response is `302` to `/auth/signin?callbackUrl=%2Fchat`.
- **S-3.b** Given a valid session but missing onboarding, when `requireOwnboarding(event)` resolves, then its `event.redirect(...)` propagates and the response is `302` to the existing onboarding route.
- **S-3.c** Given a valid session and onboarding-complete, when `onRequest` resolves, then `<ChatWindow />` renders with `messages: []`.

### REQ-4 — Backend error surfaces inline, no client-side retry

**Statement.** When the backend returns a typed `ApiError` (5xx, or 4xx mapped to `validation` / `conflict` / `not_found` / `server`), the chat SHALL render the error message inline in the conversation and MUST NOT auto-retry; the conversation SHALL remain in a state where the user can submit a new turn.

**Rationale.** Doc 0001 §4.1 lines 425–427 — *retries are a harness concern* ("decide whether to retry … the harness decides"). The error envelope's `message` is already a render-safe `string` (`frontend/src/lib/api.ts:96-110`).

**Scenarios.**

- **S-4.a** Given an open turn, when the backend closes the stream with `event: error` carrying `kind: "server"`, then the client renders one error bubble with the backend `message`, the input clears `disabled`, and NO retry timer schedules.
- **S-4.b** Given the user submits a prompt the backend rejects with 400 `kind: "validation"`, then the input shows the per-field validation inline and remains submittable, and no retry fires.
- **S-4.c** Given the backend returns 409 `kind: "conflict"` mid-stream, then the bubble displays the conflict message and no auto-retry fires.

### REQ-5 — Dev-mode honest offline failure

**Statement.** When the backend is unreachable from a dev browser, `chat-api.ts` SHALL surface `ApiErrorKind = "offline"` whose `message` contains the literal `"backend not wired — see PR for backend wire"`; the client SHALL NOT silently retry, SHALL NOT fabricate a response, and SHALL NOT log only to console.

**Rationale.** User-locked D4 (proposal §5) + explore.md R1. The backend HTTP surface does not yet exist (doc 0001 §5.2 lines 518–535 — `cmd/cachicamas` is not on disk). Surfacing the literal phrase makes the gap greppable in DevTools/CI; a fake success would mask an architectural gap.

**Scenarios.**

- **S-5.a** Given `pnpm dev` runs with no backend on the chat endpoint, when the user submits a prompt, then `chat-api.ts` resolves to `{ ok: false, kind: "offline", message: "backend not wired — see PR for backend wire" }`, the input shows that message inline, no retry timer starts, no fake assistant bubble is inserted.
- **S-5.b** Given `EventSource` opens in dev with no backend, when `onerror` fires before any `message.delta`, then the client converts the error into the same `kind: "offline"` payload and the conversation accepts a fresh submit.

### REQ-6 — Markdown + sanitization for assistant text

**Statement.** All assistant message text SHALL be rendered through `renderSanitizedMarkdown`; raw HTML from the model SHALL NOT be injected via `dangerouslySetInnerHTML` or `innerHTML`.

**Rationale.** The sanitizer is already pinned (`marked` + `isomorphic-dompurify`, allowlist at `frontend/src/lib/markdown.ts:46-59`). Re-render-per-delta is acceptable for v1 (proposal §8 step 5). The XSS-posture MUST inherit, not widen.

**Scenarios.**

- **S-6.a** Given three deltas that together form `**bold** and `code`\n\n- item`, when the bubble renders, then the bubble's HTML comes from `renderSanitizedMarkdown` and contains `<strong>`, `<code>`, and `<ul><li>`, and no other element types appear.
- **S-6.b** Given a delta containing `<script>alert(1)</script>` or `<img src=x onerror=alert(1)>`, when the bubble re-renders, then the rendered DOM contains the literal text (escaped by the allowlist) — NO `<script>` element, NO `onerror` attribute; the spec asserts `dangerouslySetInnerHTML` is not present in the bubble component.

### REQ-7 — Per-file spec discipline (strict TDD)

**Statement.** Each `*.ts` and `*.tsx` file under `frontend/src/lib/chat-api.ts` and `frontend/src/components/chat/` SHALL have a colocated `*.spec.ts` / `*.spec.tsx` that asserts at least one Given/When/Then scenario from REQ-1 .. REQ-6.

**Rationale.** `openspec/AGENTS.md` *Strict TDD is on* — *the test is the spec*. The chat's wire (D7) and state machines (REQ-1 buffer, REQ-2 cancel ordering) require colocated assertions to be reviewer-scannable.

**Scenarios.**

- **S-7.a** Given the apply phase lands, when `ls frontend/src/lib/chat-api.spec.ts frontend/src/components/chat/*.spec.{ts,tsx}` runs, then each `.ts` and `.tsx` in `chat-api.ts` and `components/chat/` has a matching spec sibling.
- **S-7.b** Given a spec file exists, when its body is grepped for `REQ-1`, `REQ-2`, … `REQ-6`, then at least one reference is present in a comment or test name.
- **S-7.c** Given a spec file lands an `it.skip`, `it.todo`, or `xit(…)`, when `pnpm test:ci` runs, then the test counts as red (strict TDD treats partial-red as red).

## Scenarios index

| Scenario | REQ | Likely spec file |
| --- | --- | --- |
| S-1.a | REQ-1 | `chat-api.spec.ts`, `use-chat-stream.spec.ts`, `chat-window.spec.tsx` |
| S-1.b | REQ-1 | `chat-api.spec.ts` |
| S-2.a / S-2.b / S-2.c | REQ-2 | `use-chat-stream.spec.ts`, `chat-input.spec.tsx`, `chat-api.spec.ts` |
| S-3.a / S-3.b / S-3.c | REQ-3 | `routes/chat/index.spec.tsx` |
| S-4.a / S-4.b / S-4.c | REQ-4 | `chat-api.spec.ts`, `chat-window.spec.tsx` |
| S-5.a / S-5.b | REQ-5 | `chat-api.spec.ts`, `use-chat-stream.spec.ts` |
| S-6.a / S-6.b | REQ-6 | `message-bubble.spec.tsx` |
| S-7.a / S-7.b / S-7.c | REQ-7 | orchestrator grep + `pnpm test:ci` |

## Out of scope (mirrored from `proposal.md` §4)

- **Any backend code, including a stub.** Owner: a separate backend change. Rationale: explore.md R1 + user "do not work in back".
- **Provider / model / credential selection in frontend.** Owner: `cmd/cachicamas`. Rationale: "the back is who controls the keys."
- **Tool-call rendering, subagent nesting, permission protocol, streaming reasoning, multimodal content.** Owner: doc 0001 §8 + §7 G1/G7/G12. Rationale: explore.md §6.
- **Session persistence across reloads.** Owner: Layer 3 (doc 0001 §5.2). Reload = empty chat in v1.
- **Real end-to-end demo against a live backend.** Owner: backend HTTP-surface change. D4 vitest-only is the v1 posture.
- **Cost / usage display, provider/model picker UI, branching UI.** Owner: doc 0001 §7 G10 + §8. Deferred.
- **New top-level frontend deps** (`@ai-sdk`, `openai`, `eventsource-parser`). Owner: none — existing primitives suffice (explore.md §3).
- **Any change to `backend/agent/src/ai/openaicompat/openrouter/` or Layer 1 packages.** Owner: parallel OpenRouter change.
- **Modifications to existing specs** (`api-error-envelope`, `frontend-auth`, `frontend-runtime`, `home-page`, …). Owner: none — read-only contracts the chat consumes.

## Cross-references

- Doc 0001 §2.2 (lines 153–218) — chat as fourth `CUSTOM` frontend.
- Doc 0001 §4.1 (lines 425–427) — retry is a harness concern, never the frontend's.
- Doc 0001 §4.2 (lines 450–455) — cancellation tree is harness-owned; frontend signals intent.
- Doc 0001 §4.3 (lines 460–487) — event envelope invariants (deltas carry an index, errors are typed).
- Doc 0001 §5.1 (lines 493–506) — Layer 3 ports; the chat consumes but defines none.
- Doc 0001 §5.2 (lines 518–535) — `cmd/cachicamas` is the only composition root; justifies the no-backend-touch rule.
- Doc 0001 §8 (lines 630–649) — non-goals (TUI deferred, no multimodal, no branching UI).
- `frontend/src/lib/api.ts:96-110` — `ApiResult<T>` discriminated union.
- `frontend/src/lib/api.ts:831-881` — `parseSSEResponse` (chat's parser reuses).
- `frontend/src/lib/api.ts:896-977` — `subscribeWorkspaceSyncStream` (SSR guard at 901, `{withCredentials: true}` at 907).
- `frontend/src/lib/api.sse.spec.ts:75-307` — `FakeES` pattern the chat specs mirror.
- `frontend/src/lib/csrf.ts:28-39` — `stateChangingFetch` for POST/DELETE.
- `frontend/src/lib/markdown.ts:65-67` — `renderSanitizedMarkdown`.
- `frontend/src/routes/home/index.tsx:39-51` — `onRequest` guard chain the chat route mirrors.
- `openspec/changes/cachicamas-frontend-chat-layer1/explore.md` — frontend primitives surveyed + backend gap named.
- `openspec/changes/cachicamas-frontend-chat-layer1/proposal.md` — decisions D1–D7, success criteria, D4 vitest-only posture.
- `openspec/AGENTS.md` — strict TDD; vitest is the frontend red-green target.

## Review checklist

- [ ] every REQ reads as a SHALL/MUST statement (no how-to).
- [ ] every REQ has ≥1 happy-path AND ≥1 failure-path in Given/When/Then.
- [ ] REQ-2 cites the harness-cancellation rationale from §4.2 (not just code reuse).
- [ ] REQ-4 explicitly forbids client-side retry — no `setTimeout(retry)` in the implementation.
- [ ] REQ-5 names the **literal** error string (`"backend not wired — see PR for backend wire"`) — greppable.
- [ ] REQ-6 forbids `dangerouslySetInnerHTML` and asserts the sanitizer allowlist, not a hand-rolled one.
- [ ] REQ-7 forces per-file spec coverage — a future contributor cannot land impl without a green-then-violet proof.
- [ ] zero file in `backend/`, `infra/`, or `docker-compose*.yaml` is touched by this spec's contract.
- [ ] no requirement contradicts `frontend-auth`, `frontend-runtime`, or `home-page` (all read-only).
