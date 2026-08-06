# Tasks: cachicamas-frontend-chat-layer1

## 1. Task overview

This plan delivers the frontend-only Qwik chat consumer, its typed SSE client, recorded-wire tests, route guard, and navigation entry as a single-PR target; strict TDD is mandatory (`strict_tdd: true`) and must be forwarded to `sdd-apply` and `sdd-verify`. Every RED→GREEN→REFACTOR step follows `openspec/AGENTS.md` §Strict TDD (lines 46–61). The proposal §6 forecast is far above the locked 800-line cap, so the orchestrator must surface the `size:exception` versus client-first/UI-second split before apply.

## 2. Forecast vs budget

Ranges below use proposal §6's corrected ×2.5 heuristic; fixture bytes are shown but excluded from authored-line risk.

| New file | Planned corrected lines |
|---|---:|
| `frontend/src/lib/chat-types.ts` | ~100–200 |
| `frontend/src/lib/chat-api.ts` | ~150–250 |
| `frontend/src/lib/chat-api.spec.ts` | ~375–625 |
| `frontend/src/components/chat/use-chat-stream.ts` | ~150–250 |
| `frontend/src/components/chat/use-chat-stream.spec.ts` | ~200–350 |
| `frontend/src/components/chat/chat-window.tsx` | ~250–375 |
| `frontend/src/components/chat/chat-window.spec.tsx` | ~200–375 |
| `frontend/src/components/chat/message-bubble.tsx` | ~100–200 |
| `frontend/src/components/chat/message-bubble.spec.tsx` | ~100–200 |
| `frontend/src/components/chat/chat-input.tsx` | ~150–300 |
| `frontend/src/components/chat/chat-input.spec.tsx` | ~150–300 |
| `frontend/src/components/chat/__fixtures__/single-turn.sse` | ~10–30 (fixture bytes; excluded) |
| `frontend/src/routes/chat/index.tsx` | ~100–200 |
| `frontend/src/routes/chat/layout.tsx` | ~50–100 |
| `frontend/src/routes/chat/index.spec.tsx` | ~150–300 |
| Modified `frontend/src/components/avatar-dropdown/avatar-dropdown.tsx` | ~1–5 |
| **Rounded authored forecast** | **~2,100–3,750** |

Forecast: ~2,100–3,750 corrected lines across 14 files (12 new + 1 modified + 1 fixture). This **exceeds** the 800-line review budget. The user's preflight locked `delivery_strategy = single-pr`, so the path is to confirm `size:exception` at the orchestrator's review gate (per the Review Workload Guard, single-pr requires maintainer-approved `size:exception` before sdd-apply). The alternative — split into PR #1 (client + tests + fixtures, no UI) and PR #2 (UI consuming the client) — was surfaced in proposal §6 and remains open. **Surface this fork to the user at the orchestrator's next interaction; do not pick.**

### 2.1 Resolution (post-review-gate)

**Status**: maintainer-approved `size:exception`. The orchestrator surfaced the fork and the user picked **"Approve size:exception (single PR)"** at the review gate. This PR ships as one branch (`feat/frontend-chat-layer1` → `main`) with all 10 tasks in one merged PR. Reviewer burden is acknowledged higher than the 800-line cap; one reviewer sees the chat end-to-end. Per the orchestrator's `single-pr` preflight, this is the maintainer override that authorises `sdd-apply` to proceed without re-surfacing the fork. Pattern recorded at Engram topic `workflow/cachicamas-size-exception-pattern` (observation #2584) so future frontend changes in this repo can reference the precedent.

## 3. Tasks

> **Checkbox reconciliation (sdd-archive, 2026-08-06)**: The persisted tasks artifact was delivered by `sdd-apply` with checkboxes still unchecked on disk, but every T-01..T-10 has an executed commit on the branch (`d23b651`/`fb7cd93`/`abf511c`/`3d3c2ff`/`79fa8ad`/`b1fe40e`/`3f4ab44`/`46b168d`/`6afb044`/`6379fe2`) plus the in-batch QRL fix (`104a73b`). Both `apply-progress` (Engram #2585) and `verify-report` (#2600, PASS verdict, 78/78 chat tests) prove completion; the orchestrator's launch handoff explicitly confirms `Apply outcome: PASS`. Per the `sdd-archive` Task Completion Gate, this is an exceptional mechanical reconciliation — full reason recorded in `archive-report.md` §3 + §5. The apply agent owns normal checkbox completion; this repair only ensures the archived audit trail does not ship with stale unchecked tasks for completed work.

- [x] **T-01 — Add typed chat contracts** (REQ-1, REQ-4, REQ-6)
  - **Scope**: Establish the discriminated event, request/response, error, message, session, and fixture types before UI work.
  - **Files**: `frontend/src/lib/chat-types.ts`; typed assertions in `frontend/src/lib/chat-api.spec.ts`.
  - **RED** — Add an `assertNever` exhaustiveness spec and fixture-import type assertion; it fails because `chat-types.ts` does not export `ChatStreamEvent` (fixture import remains red until T-02). **Verification**: `pnpm --filter @cachicamas/frontend test:ci -- chat-api.spec.ts`
  - **GREEN** — Write the §2 contracts verbatim from `design.md` into `chat-types.ts`; make no `*.tsx` changes.
  - **REFACTOR** — Centralize `assertNever`/REQ comments without widening the union.
  - **Commit boundary**: `feat(chat): define typed stream contracts` — `chat-types.ts` and its assertions, per `work-unit-commits`.
  - **Verification**: `pnpm --filter @cachicamas/frontend verify`
  - **Rollback**: Revert only the new type/assertion files; no runtime behavior changes.

- [x] **T-02 — Record the single-turn SSE fixture** (REQ-1, REQ-7)
  - **Scope**: Pin the D7 wire envelope before client implementation.
  - **Files**: `frontend/src/components/chat/__fixtures__/single-turn.sse`; `frontend/src/lib/chat-api.spec.ts`.
  - **RED** — Import the fixture and assert `parseTranscript(fixture).length > 0`; it fails with “fixture not found”. **Verification**: `pnpm --filter @cachicamas/frontend test:ci -- chat-api.spec.ts`
  - **GREEN** — Add byte-correct `message.start`, three `message.delta`s, `message.end`, and terminal `turn.end`; in `frontend/src/lib/chat-api.spec.ts` export the typed `const expectedEvents: ChatStreamEvent[]` beside the raw fixture import.
  - **REFACTOR** — Keep one canonical transcript and explicit event-order assertion.
  - **Commit boundary**: `test(chat): record single-turn SSE transcript` — fixture and spec, per `work-unit-commits`.
  - **Verification**: `pnpm --filter @cachicamas/frontend verify`
  - **Rollback**: Remove only the fixture/test additions; client behavior is unchanged.

- [x] **T-03 — Implement the typed chat API** (REQ-1, REQ-2, REQ-4, REQ-5)
  - **Scope**: Add POST, typed EventSource subscription, DELETE cancellation, and honest offline mapping.
  - **Files**: `frontend/src/lib/chat-api.ts`; `frontend/src/lib/chat-api.spec.ts`.
  - **RED** — Against `FakeES` (mirror `lib/api.sse.spec.ts:75-307`), assert `submitTurn` success, `cancelTurn` DELETE, mid-stream `event: error` → `ChatStreamError`, and offline POST → `ApiResult.kind = "offline"` with a message containing `"backend not wired — see PR for backend wire"`; all fail because the client is absent. **Verification**: `pnpm --filter @cachicamas/frontend test:ci -- chat-api.spec.ts`
  - **GREEN** — Implement `submitTurn`, `cancelTurn`, `subscribeTurn`; reuse chunk-split discipline and dispatch with `EventSource.addEventListener("message.<x>", ...)` per D1 refinement; never retry.
  - **REFACTOR** — Extract shared response/error decoding while preserving `ApiResult` semantics.
  - **Commit boundary**: `feat(chat): typed client with vitest specs against recorded SSE transcript` — API and spec, per `work-unit-commits`.
  - **Verification**: `pnpm --filter @cachicamas/frontend verify`
  - **Rollback**: Revert `chat-api.ts` and its spec; no routes or existing API code change.

- [x] **T-04 — Implement the chat stream hook** (REQ-1, REQ-2, REQ-7)
  - **Scope**: Own `ChatSession` signals/store, event mutation, cancellation, and hydration cleanup.
  - **Files**: `frontend/src/components/chat/use-chat-stream.ts`; `use-chat-stream.spec.ts`.
  - **RED** — Mock `EventSource`/client; assert `useChatStream$` returns a `ChatSession`, `applyEvent(message.delta)` appends to the last assistant bubble, and `useVisibleTask$` unmount cleanup leaves no goroutine/stream/listener leak; it fails because the hook is absent. **Verification**: `pnpm --filter @cachicamas/frontend test:ci -- use-chat-stream.spec.ts`
  - **GREEN** — Use `useSignal` + `useStore`, register typed listeners inside `useVisibleTask$`, close naturally on `turn.end`, and issue DELETE plus close on Stop/unmount.
  - **REFACTOR** — Isolate the pure event mutator and make cleanup idempotent.
  - **Commit boundary**: `feat(chat): manage streaming session lifecycle` — hook and spec, per `work-unit-commits`.
  - **Verification**: `pnpm --filter @cachicamas/frontend verify`
  - **Rollback**: Revert hook/spec only; API contracts remain available.

- [x] **T-05 — Render the chat window** (REQ-1, REQ-4)
  - **Scope**: Compose message bubbles, stream state, inline errors, and input slot.
  - **Files**: `frontend/src/components/chat/chat-window.tsx`; `chat-window.spec.tsx`.
  - **RED** — Render a pre-seeded store and assert user/assistant bubbles plus offline error; it fails because `ChatWindow` is absent. **Verification**: `pnpm --filter @cachicamas/frontend test:ci -- chat-window.spec.tsx`
  - **GREEN** — Render `ChatMessage[]` and `<ChatInput />`, preserving partial text and inline typed errors.
  - **REFACTOR** — Keep window orchestration separate from bubble/input presentation.
  - **Commit boundary**: `feat(chat): render chat window` — window and spec, per `work-unit-commits`.
  - **Verification**: `pnpm --filter @cachicamas/frontend verify`
  - **Rollback**: Revert window/spec; stream client and hook remain independent.

- [x] **T-06 — Sanitize assistant message bubbles** (REQ-6)
  - **Scope**: Render model markdown through the existing sanitizer only.
  - **Files**: `frontend/src/components/chat/message-bubble.tsx`; `message-bubble.spec.tsx`.
  - **RED** — Assert a delta containing `<script>`/`onerror` produces no executable DOM after DOMPurify sanitization; it fails because the bubble is absent. **Verification**: `pnpm --filter @cachicamas/frontend test:ci -- message-bubble.spec.tsx`
  - **GREEN** — Render assistant text with `lib/markdown.ts:65-67` `renderSanitizedMarkdown`; do not use `dangerouslySetInnerHTML`/`innerHTML` directly.
  - **REFACTOR** — Share role/status classes without changing the allowlist.
  - **Commit boundary**: `feat(chat): sanitize assistant message rendering` — bubble and spec, per `work-unit-commits`.
  - **Verification**: `pnpm --filter @cachicamas/frontend verify`
  - **Rollback**: Revert bubble/spec; no API or session behavior changes.

- [x] **T-07 — Add chat input controls** (REQ-1, REQ-2)
  - **Scope**: Provide prompt entry, Enter submit, Send, and Stop states.
  - **Files**: `frontend/src/components/chat/chat-input.tsx`; `chat-input.spec.tsx`.
  - **RED** — Assert input is disabled during streaming and Enter invokes submit; it fails because the component is absent. **Verification**: `pnpm --filter @cachicamas/frontend test:ci -- chat-input.spec.tsx`
  - **GREEN** — Add textarea/button controls with disabled state derived from `session.status` and Stop wired to cancellation.
  - **REFACTOR** — Normalize keyboard handling and prevent empty submits.
  - **Commit boundary**: `feat(chat): add prompt input controls` — input and spec, per `work-unit-commits`.
  - **Verification**: `pnpm --filter @cachicamas/frontend verify`
  - **Rollback**: Revert input/spec; window can temporarily retain its slot.

- [x] **T-08 — Add the protected `/chat` route** (REQ-3, REQ-5, REQ-6, REQ-7)
  - **Scope**: Wire route shell, auth/onboarding guard, head metadata, and offline rendering.
  - **Files**: `frontend/src/routes/chat/index.tsx`, `layout.tsx`, `index.spec.tsx`.
  - **RED** — Assert unauthenticated access redirects via the existing chain and backend-unreachable state renders the offline error; it fails because the route is absent. **Verification**: `pnpm --filter @cachicamas/frontend test:ci -- routes/chat/index.spec.tsx`
  - **GREEN** — Render `<ChatWindow />`; in the route `layout.tsx` call `setSsrCookieHeader → requireAuthRedirect → requireOwnboarding` and preserve existing cookie policy.
  - **REFACTOR** — Keep guard logic copied only at the route boundary and make head title explicit.
  - **Commit boundary**: `feat(chat): expose protected chat route` — route files/spec, per `work-unit-commits`.
  - **Verification**: `pnpm --filter @cachicamas/frontend verify`
  - **Rollback**: Remove `routes/chat/`; existing routes remain untouched.

- [x] **T-09 — Link Chat from live authenticated navigation** (D6, REQ-3)
  - **Scope**: Add one `/chat` discoverability link, overriding the stale design target if necessary.
  - **Files**: `frontend/src/components/avatar-dropdown/avatar-dropdown.tsx`; existing `avatar-dropdown.spec.tsx` (verify `example/example.tsx` is not a sidebar).
  - **RED** — Assert the opened authenticated navigation menu contains `<a href="/chat">`; it fails because no link exists. **Verification**: `pnpm --filter @cachicamas/frontend test:ci -- avatar-dropdown.spec.tsx`
  - **GREEN** — Add one `<Link href="/chat">Chat</Link>`/equivalent `MenuItem` to `AvatarDropdown`; `example/example.tsx` is only a counter and is not in the app shell, so this override is documented.
  - **REFACTOR** — Preserve existing menu ordering/accessibility and add no new navigation abstraction.
  - **Commit boundary**: `feat(chat): link from sidebar` — chosen navigation component/spec, per `work-unit-commits`.
  - **Verification**: `pnpm --filter @cachicamas/frontend verify`
  - **Rollback**: Revert only the `/chat` menu item and its assertion.

- [x] **T-10 — End-to-end verification** (REQ-1–REQ-7)
  - **Scope**: Verify the completed frontend without changing code.
  - **Files**: None.
  - **RED** — Run the proving commands before declaring green; Playwright may fail because the backend HTTP surface does not exist, which is expected per proposal §4/R2. **Verification**: `pnpm --filter @cachicamas/frontend test:ci`
  - **GREEN** — Run `pnpm --filter @cachicamas/frontend verify` and `pnpm --filter @cachicamas/frontend test:e2e -- --grep chat`; record expected backend-unwired e2e failure separately.
  - **REFACTOR** — N/A; verification-only.
  - **Commit boundary**: N/A—no code change or commit; retain the verification receipt.
  - **Verification**: `pnpm --filter @cachicamas/frontend verify` plus the chat Playwright command.
  - **Rollback**: N/A.

## 4. Dependency order

`T-01 → T-02 → T-03 → T-04 → T-05 → T-06 → T-07 → T-08 → T-09 → T-10`; T-01 establishes contracts, T-02 pins bytes, T-03 supplies the client, then state/UI/route/navigation compose around it. Per `work-unit-commits`, T-01–T-09 each form one behavior-sized commit with its tests; T-10 is the final receipt.

## 5. Review Workload Forecast

| Metric | Value |
|---|---|
| Files (planned) | 14 (13 new + 1 modified) |
| Naive line count | ~840–1,500 |
| Corrected (×2.5 midpoint) | ~2,100–3,750 |
| Chained PRs recommended | No (single-pr preflight) |
| 400-line budget risk | **High** (forecast exceeds 800 cap) |
| Decision needed before apply | **Yes — confirm `size:exception` or split at orchestrator review** |

Decision needed before apply: Yes
Chained PRs recommended: No
Chain strategy: pending
400-line budget risk: High

Conditional work-unit evidence (not a fork selection): PR #1 client + tests + fixture; focused `pnpm --filter @cachicamas/frontend test:ci -- chat-api.spec.ts`; runtime `N/A` because no backend HTTP surface exists; rollback client/spec/fixture. PR #2 UI + route + navigation; focused `pnpm --filter @cachicamas/frontend test:ci -- chat-window.spec.tsx routes/chat/index.spec.tsx`; runtime `N/A` for the same reason; rollback UI/route/navigation files.

## 6. Risks (carried + new)

- **T-08 e2e**: the backend does not exist, so Playwright live-chat e2e will fail by design; record it as expected, not as a frontend workaround.
- **T-02 fixture**: byte shape must match the backend wire exactly; if future event names differ, T-02 forces the fixture and contract to change first.
- **T-09 location**: design names `components/example/example.tsx`, but it is not a sidebar in this worktree; verify the live navigation at apply time and retain the documented `AvatarDropdown` override if it remains the better target.

The design threat matrix is explicitly `N/A`; no additional threat RED tests are applicable.

## 7. Next recommended

`sdd-apply` (the next dependency-ready phase per the Dependency Graph). Apply will be launched only after the user confirms `size:exception` at the orchestrator's gate.
