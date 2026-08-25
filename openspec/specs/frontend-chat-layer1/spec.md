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

### REQ-5 — Dev-mode honest offline failure (amended 2026-08-24)

**Statement.** When the chat endpoint is unreachable from a dev browser, `chat-api.ts` SHALL surface `ApiErrorKind = "offline"` whose `message` contains the substring `"Couldn't reach the chat service. Is docker compose up?"`; the client SHALL NOT silently retry, SHALL NOT fabricate a response, and SHALL NOT log only to console.

**Rationale.** The original REQ-5 stated the purpose was to make an architectural gap greppable. The gap is now closed: the archetype's composition root (`backend/agent/src/cmd/chat/`, CH-04, PR #194, commit `2b891117`) serves `POST /api/agent/turns`, and `frontend/src/components/chat/use-chat-stream.ts` (CH-05.1) submits against it. The literal that REQ-5 mandated — `"backend not wired — see PR for backend wire"` — is therefore no longer true. Deleting it without a recorded spec delta would leave a promoted requirement falsified by shipped code and nothing would fail (`0005:97`, `0005:637`). This amendment retires the literal **on the record**, replacing it with a generic dev-honest phrase that preserves the original REQ-5's intent: a dev-mode network failure is greppable, never silently retried or fabricated, without claiming a backend is unwired when one now is. The `kind: "offline"` arm of `ApiResult<T>` (`frontend/src/lib/api.ts:89-94`, `chat-types.ts:93-98`) survives — it is the right shape for any transient network failure and is never produced by the server (D-1).

**Scenarios.**

- **S-5.a (amended)** — Given `pnpm dev` runs with no backend reachable on the chat endpoint, when the user submits a prompt, then `chat-api.ts` resolves to `{ ok: false, kind: "offline", message: "Couldn't reach the chat service. Is docker compose up? (network error)" }`, the input shows that message inline, no retry timer starts, no fake assistant bubble is inserted.
- **S-5.b (amended)** — Given `EventSource` opens in dev with no backend reachable, when `onerror` fires before any `message.delta`, then the client converts the error into the same `kind: "offline"` payload and the conversation accepts a fresh submit.
- **S-5.c (new, 2026-08-24)** — Given the archetype's composition root is serving at `POST /api/agent/turns` (`backend/agent/src/cmd/chat/`, CH-04), when the chat page submits a prompt, then the resolved message MUST NOT contain the retired literal `"backend not wired — see PR for backend wire"`. The `kind: "offline"` arm MAY still fire if the network itself fails; its message must be the amended phrase.

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
- [ ] REQ-5 names an error string (amended post-CH-05.2: substring `"Couldn't reach the chat service. Is docker compose up?"`) — the literal `OFFLINE_LITERAL` constant is retired in the merged tree.
- [ ] REQ-6 forbids `dangerouslySetInnerHTML` and asserts the sanitizer allowlist, not a hand-rolled one.
- [ ] REQ-7 forces per-file spec coverage — a future contributor cannot land impl without a green-then-violet proof.
- [ ] zero file in `backend/`, `infra/`, or `docker-compose*.yaml` is touched by this spec's contract.
- [ ] no requirement contradicts `frontend-auth`, `frontend-runtime`, or `home-page` (all read-only).

---

# Amendments added by CH-08 (2026-08-24) — `cachicamas-chat-resume-in-browser`

> **Change**: `cachicamas-chat-resume-in-browser` · **CH-08** (Wave 2) of [doc 0005](../../../docs/architecture/milestones/0005-cachicamas-chat-archetype-task-graph.md#ch-08--resume-a-conversation-in-the-browser) (`0005:842-899`)
> **Amends**: this spec (`frontend-chat-layer1`) — **additive only**. Every REQ-1..REQ-7 requirement is **byte-unchanged**. The closed `ChatStreamEvent` union (REQ-1, REQ-7) is **not widened**. New identifiers: **REQ-8**, **REQ-9**, **S-FCL-008**, **S-FCL-009**, **S-FCL-010**, **S-FCL-011**. The header-trimmed README pointer records the CH-08 charter and the Gherkin scenarios at `0005:859-873` and `0005:880-895` as the source-of-truth wording the new S-FCL-NNN scenarios transcribe.

## ADDED Requirements

### REQ-8 — On mount, the page fetches the participant's recorded transcript and conversation summary list and renders them

On the Qwik `useVisibleTask$` mount of `chat-app.tsx`, the page MUST issue `GET /api/agent/conversations/:id` (CH-08.1, R-CRI-001) and `GET /api/agent/conversations` (CH-08.2, R-CRI-002) in parallel against the participant id surfaced from `requireSession` (`routes/chat/index.tsx`). The page MUST call `useChatStream.reset(entries)` (CH-05 hook API) with the recorded exchanges from the reload endpoint, seeding the buffer without opening an `EventSource`. The page MUST re-mount the `ConversationList` component (`conversation-list.tsx`, surviving from CH-05.1, dropped per CH-05 D-3) against the wire summary list — `id` is the participant id, `age` is derived from `lastActivityAt` via a relative-time helper, `turnCount` is rendered alongside. The deep-link `?with=<slug>` MUST continue to resolve to an `agentSlug` from `staff.ts` (CH-05 D-6, decisions D-4) but MUST NOT drive which conversation loads — the participant's recorded transcript is the source of truth.

#### Scenario: S-FCL-008 — reloading the page restores the conversation (Gherkin verbatim, `0005:862-866`)

- Given an employee who has completed two turns and reloads the page
- When the page loads
- Then both exchanges are shown in their original order
- And the input accepts a new prompt that continues the same conversation

#### Scenario: S-FCL-009 — a reload during a streaming turn shows what was recorded (Gherkin verbatim, `0005:868-872`)

- Given an employee who reloads while a turn is streaming
- When the page loads
- Then the exchanges recorded before the reload are shown
- And the page does not claim the turn is still streaming

#### Scenario: S-FCL-010 — a participant sees their own conversations and no others (Gherkin verbatim, `0005:885-889`)

- Given two participants who have each held conversations
- When one of them requests their list
- Then the list contains only their own conversations
- And each entry identifies its conversation well enough to open it

#### Scenario: S-FCL-011 — a participant with no conversations gets an empty list (Gherkin verbatim, `0005:891-895`)

- Given an authenticated participant who has never held a conversation
- When they request their list
- Then the list is empty
- And the response is a success rather than a not-found

### REQ-9 — A reload that catches an in-flight turn MUST NOT claim streaming

On a reload that catches an in-flight turn (the participant reloaded the tab mid-stream), the page MUST render the exchanges recorded before the reload (REQ-8, S-FCL-009). The page MUST NOT claim the turn is still streaming — `Composer`'s `status` MUST resolve to `"idle"` once both GETs return; `turn.status` (the `useChatStream` signal) MUST NOT be set to `"streaming"` on mount. The page MUST NOT auto-open an `EventSource` for the prior conversation id; the next `EventSource` is opened only on the next `submit()` call. The closed `ChatStreamEvent` union (REQ-1, REQ-7) is not widened by this delta — the read-side wire types `ConversationSummaryDTO` and `ExchangeDTO` are new types adjacent to it, never additions to it.

---

# CH-09 amendment — `cachicamas-chat-tool-source`

> **Change**: `cachicamas-chat-tool-source` · **CH-09** (Wave 3, 10 of 12) of [doc 0005](../../../docs/architecture/milestones/0005-cachicamas-chat-archetype-task-graph.md#ch-09--offer-tools-through-a-tool-source-port) (`0005:923-934`)
> **Amends**: this spec (`frontend-chat-layer1`) — **additive only**. **REQ-1..7 byte-unchanged**. New identifiers: **REQ-8**, **REQ-9**, **REQ-10**, **REQ-11**, **S-FCL-012**, **S-FCL-013**, **S-FCL-014**, **S-FCL-015**, **S-FCL-016**, **S-FCL-017**. Identifier-append-only per CH-08 precedent at the CH-08 amendment header (`frontend-chat-layer1/spec.md:154-157`).
>
> **D-7 surfacing (deliberate)** — the closed `ChatStreamEvent` union (REQ-1, REQ-7) is widened by four new variants below. REQ-7's spec text forbids **new fields on existing variants**; CH-09 widens with **new variants on the union, not new fields on existing variants**. Each of REQ-8..11 below carries explicit wording that distinguishes variant-addition from field-addition, so a future reviewer cannot misread the original REQ-7 as forbidding it. The wording is the source-of-truth distinction this amendment records.
>
> **RE-7 preserved verbatim** — REQ-7's "closed union" wording at `frontend-chat-layer1/spec.md:83-94` is **not modified**. The widening allowance lives in REQ-8..11's rationale lines, NOT in a modification of REQ-7's text.

## ADDED Requirements

### REQ-8 — `tool.call.start` SSE event

A new SSE event with `event: tool.call.start` MUST be added as a **new variant** on the closed `ChatStreamEvent` union (`frontend/src/lib/chat-types.ts:27-32`). The payload shape MUST be `{ wireCallId: string, tool: string, arguments: string }` — all field names lowercase JSON keys, parallel to the closed `ExchangeDTO` precedent (`chat-types.ts:152-161`). The variant discriminator MUST be `"tool.call.start"`.

**D-7: this requirement adds a new variant on the closed `ChatStreamEvent` union; it does NOT add new fields on existing variants REQ-1..7 enumerate. The "closed union" wording in REQ-7 forbids field additions, not variant additions — see `cachicamas-chat-tool-source/proposal.md` D-7 and `cachicamas-chat-tool-source/explore.md` for the deliberate additive widening. The wire's `wireFrameName` switch at `backend/agent/src/chat/eventsource.go:31-48` and the `parseTranscript` switch at `chat-types.ts:236-272` are extended by this requirement; a new variant without a case is a compile error (S-CTS-007, S-CTS-024).**

#### Scenario: S-FCL-012 — `parseTranscript` parses `tool.call.start` into a typed variant (Gherkin verbatim, explore #3952; mirrors `cachicamas-chat-tool-source/spec.md` S-CTS-013)

- Given `parseTranscript` reading a `tool.call.start` frame with payload `{"wireCallId":"c1","tool":"current_time","arguments":"{}"}`
- When the frame is parsed
- Then the resulting `ChatStreamEvent` is `{kind: "tool.call.start", wireCallId: "c1", tool: "current_time", arguments: "{}"}` (typed and exhaustive)

#### Scenario: S-FCL-013 — a malformed `tool.call.start` JSON is silently dropped (Gherkin verbatim, explore #3952; mirrors `cachicamas-chat-tool-source/spec.md` S-CTS-014)

- Given a `tool.call.start` frame with malformed JSON
- When parsed
- Then the frame is silently dropped (S-1.b mirror at `chat-types.ts:206-210`)

### REQ-9 — `tool.call.delta` SSE event

A new SSE event with `event: tool.call.delta` MUST be added as a **new variant** on the closed `ChatStreamEvent` union. The payload shape MUST be `{ wireCallId: string, delta: string }`. **The variant is reserved for future progress-bearing tools; v1 does not emit this event.** The wire shape (`wireFrameName` case + `parseTranscript` case + `KNOWN_EVENTS` entry at `chat-api.ts:285-291`) MUST exist so a future long-running MCP tool can land here without a wire shape change.

**D-7: new variant, not new field. Reserved-but-unused at v1 per `cachicamas-chat-tool-source/spec.md` NFR-CTS-002 / D-6. A future tool engineer asking for progress has a place to put it; v1 deliberately leaves it empty.**

### REQ-10 — `tool.call.end` SSE event

A new SSE event with `event: tool.call.end` MUST be added as a **new variant** on the closed `ChatStreamEvent` union. The payload shape MUST be `{ wireCallId: string, outcome: string }`. **The variant is reserved; v1 collapses Layer 2's three `ToolEnd*` kinds into `tool.result` (D-6) — `tool.call.end` is therefore not emitted at v1.** The wire shape MUST exist so a future v2 dynamic-source surface has a place to split the bracket from the result.

**D-7: new variant, not new field. Reserved-but-unused at v1 per `cachicamas-chat-tool-source/spec.md` NFR-CTS-002 / D-6.**

### REQ-11 — `tool.result` SSE event

A new SSE event with `event: tool.result` MUST be added as a **new variant** on the closed `ChatStreamEvent` union. The payload shape MUST be `{ wireCallId: string, tool: string, outcome: "success" | "result_failure" | "execution_failure", content: string, failureCategory: string }`. `failureCategory` is non-empty ONLY when `outcome == "execution_failure"` (mirroring R-CCP-008 / D6's "no provider text leaks" rule). At v1 the chat projector collapses Layer 2's three `ToolEnd*` kinds into this single chat-side event (D-6); one transcript entry per tool call (D-4).

**D-7: new variant, not new field. The discriminator value `"tool.result"` is new; no existing REQ-1..7 variant is modified. Mirrors `cachicamas-chat-tool-source/spec.md` R-CTS-004 (`ToolResult` variant) byte-for-byte.**

#### Scenario: S-FCL-014 — `exchangesToEntries` renders tool entries from `ExchangeDTO` (Gherkin verbatim, explore #3952; mirrors `cachicamas-chat-tool-source/spec.md` S-CTS-019)

- Given an `ExchangeDTO{ToolCalls: [c1], ToolResults: [r1]}`
- When `exchangesToEntries([exchange])` runs
- Then the returned array contains exactly one `kind: "tool"` entry with `tool: c1.tool`, `args: parseArgs(c1.arguments)`, `result: r1.content`, `state: "done"` — appended AFTER the assistant said entry

#### Scenario: S-FCL-015 — a live `tool.call.start` SSE frame opens a "running" tool entry (Gherkin verbatim, explore #3952; mirrors `cachicamas-chat-tool-source/spec.md` S-CTS-020)

- Given a `tool.call.start` SSE frame arriving mid-turn
- When the page receives it
- Then a new transcript entry with `kind: "tool"`, `state: "running"`, the tool name and parsed args is appended

#### Scenario: S-FCL-016 — a `tool.result` with `execution_failure` closes the entry as "failed" (Gherkin verbatim, explore #3952; mirrors `cachicamas-chat-tool-source/spec.md` S-CTS-021)

- Given the matching `tool.result` frame with `outcome: "execution_failure"`
- When received
- Then the tool entry's `state` becomes `"failed"` and `result` carries the failure phrase (no provider text — R-CCP-008 / D6 mirror on the wire)

#### Scenario: S-FCL-017 — `turn.end` after a tool execution allows the next assistant bubble to stream tool-result-aware text (Gherkin verbatim, explore #3952; mirrors `cachicamas-chat-tool-source/spec.md` S-CTS-022 with **F-CHT-9.3 wording amendment**)

- Given a tool call whose model emits `turn.end` after the `tool.result` frame
- When the page receives the `turn.end` frame (any `finishReason`, including `"tool_calls"`)
- Then the assistant text bubble that follows the tool entry continues to accumulate any subsequent `message.delta` frames — keyed on the original `assistantId`. The continuation is finishReason-agnostic; the `finishReason: "tool_calls"` value carries the model's signal but is not explicitly gated in the hook (`use-chat-stream.ts:269-281` marks the entry final on `turn.end`; `use-chat-stream.ts:202-209` continues to append subsequent `message.delta` frames to the same entry). See `cachicamas-chat-tool-source/spec.md` S-CTS-022 for the implementation rationale and the `use-chat-stream.spec.tsx` `S-CTS-022` covering test.

## Untemporal-invariant register (CH-09 addition)

The D-7 record above is preserved here as a **structural marker**: REQ-7's closed-union wording at `frontend-chat-layer1/spec.md:83-94` is **not modified**; the widening allowance lives in REQ-8..11's rationale lines. The CTS-* scenarios (S-CTS-013, S-CTS-014, S-CTS-019..022) are the source of truth for the S-FCL-012..017 scenarios above; any future amendment MUST update CTS-* first and reference it from S-FCL-*.
