# Spec delta — `frontend-chat-layer1`

> **Delta from:** `openspec/specs/frontend-chat-layer1/spec.md` (REQ-5 only)
> **Change:** `cachicamas-chat-frontend-wire` · CH-05.2 [mechanical] (`docs/architecture/milestones/0005-cachicamas-chat-archetype-task-graph.md:671-675`)
> **Closes:** R-13 (`0005:69`), R-16 (`0005:72`), register row 4 (`0005:1045`)
> **Date:** 2026-08-24

## Why this delta exists

Doc 0005's CH-05.2 charter (`0005:671-675`) is a single-paragraph record-amendment load that cannot precede CH-05.1. It binds three decisions (D-1, D-2, D-3) to one operational rule, quoted verbatim from `0005:673`: *"a spec delta modifying the offline-honesty requirement, stating that its purpose — making an unwired backend greppable — is discharged by the archetype now serving the wire, and that the literal is retired rather than forgotten. The merged tree contains no occurrence of the literal, and the amended requirement's scenarios no longer assert it."* The rule exists because `chat-archetype-contract` R-CHT-009 (`openspec/specs/chat-archetype-contract/spec.md:181`) forbids the silent retirement of any element REQ-5 mandates: *"retiring it needs a recorded spec delta at CH-05.2."*

The gap that REQ-5 was designed to expose is now closed. CH-04 (`backend/agent/src/cmd/chat/`, PR #194, commit `2b891117`) mounts `POST /api/agent/turns`, `GET …/events`, and `DELETE …/:id` (`backend/agent/src/chat/http.go:305-307`); CH-05.1 (`use-chat-stream.ts`, `chat-app.tsx`) submits against it. The literal that REQ-5 mandated — `"backend not wired — see PR for backend wire"` — is therefore no longer true. Deleting it without a recorded spec delta would leave a promoted requirement falsified by shipped code and nothing would fail (`0005:637`, `0005:97`). This delta retires the literal **on the record**, replacing it with a generic dev-honest phrase that preserves REQ-5's original intent — a dev-mode network failure is greppable, never silently retried, never fabricated — without claiming a backend is unwired when one now is.

## Inherited, unchanged

The following requirements of `frontend-chat-layer1` are inherited from the promoted spec and unchanged by this delta:

- **REQ-1 (`spec.md:13-22`)** — Submitting a prompt from `/chat` SHALL open `POST /api/agent/turns`, stream `message.delta` into one assistant bubble, and close on `turn.end`.
- **REQ-2 (`spec.md:24-34`)** — A non-terminal close (unmount, Stop) SHALL issue `DELETE /api/agent/turns/:id` in addition to `EventSource.close()`; a `turn.end` close SHALL NOT.
- **REQ-3 (`spec.md:36-46`)** — Unauthenticated access SHALL redirect via the `setSsrCookieHeader → requireAuthRedirect → requireOwnboarding` chain, identical to `routes/home/`.
- **REQ-4 (`spec.md:48-58`)** — A typed backend error SHALL render inline; the client MUST NOT auto-retry; the conversation SHALL accept a fresh submit.
- **REQ-6 (`spec.md:71-80`)** — Assistant text SHALL be rendered through `renderSanitizedMarkdown`; raw HTML SHALL NOT be injected via `dangerouslySetInnerHTML`.
- **REQ-7 (`spec.md:82-92`)** — Each `*.ts` / `*.tsx` under `chat-api.ts` and `components/chat/` SHALL have a colocated `*.spec.{ts,tsx}` asserting at least one REQ-N scenario; no `it.skip` / `it.todo` / `xit(…)`.

## Amendment — REQ-5 retired on 2026-08-24

> The text below **replaces** REQ-5 in the promoted `frontend-chat-layer1/spec.md:60-69`. The amendment is recorded; on archive, the source spec is rewritten to match.

### REQ-5 — Dev-mode honest network failure (amended)

**Statement.** When the chat endpoint is unreachable from a dev browser, `chat-api.ts` SHALL surface `ApiErrorKind = "offline"` whose `message` contains the substring `"Couldn't reach the chat service. Is docker compose up?"`; the client SHALL NOT silently retry, SHALL NOT fabricate a response, and SHALL NOT log only to console.

**Rationale.** The original REQ-5 stated the purpose was to make an architectural gap greppable. The gap is now closed: the archetype's composition root (`backend/agent/src/cmd/chat/`, CH-04, PR #194, commit `2b891117`) serves `POST /api/agent/turns`, and `frontend/src/components/chat/use-chat-stream.ts` (CH-05.1) submits against it. The literal that REQ-5 mandated — `"backend not wired — see PR for backend wire"` — is therefore no longer true. Deleting it without a recorded spec delta would leave a promoted requirement falsified by shipped code and nothing would fail (`0005:97`, `0005:637`). This amendment retires the literal **on the record**, replacing it with a generic dev-honest phrase that preserves the original REQ-5's intent: a dev-mode network failure is greppable, never silently retried or fabricated, without claiming a backend is unwired when one now is. The `kind: "offline"` arm of `ApiResult<T>` (`frontend/src/lib/api.ts:89-94`, `chat-types.ts:93-98`) survives — it is the right shape for any transient network failure and is never produced by the server (D-1).

**Scenarios.**

- **S-5.a (amended)** — Given `pnpm dev` runs with no backend reachable on the chat endpoint, when the user submits a prompt, then `chat-api.ts` resolves to `{ ok: false, kind: "offline", message: "Couldn't reach the chat service. Is docker compose up? (network error)" }`, the input shows that message inline, no retry timer starts, no fake assistant bubble is inserted.
- **S-5.b (amended)** — Given `EventSource` opens in dev with no backend reachable, when `onerror` fires before any `message.delta`, then the client converts the error into the same `kind: "offline"` payload and the conversation accepts a fresh submit.
- **S-5.c (new, 2026-08-24)** — Given the archetype's composition root is serving at `POST /api/agent/turns` (`backend/agent/src/cmd/chat/`, CH-04), when the chat page submits a prompt, then the resolved message MUST NOT contain the retired literal `"backend not wired — see PR for backend wire"`. The `kind: "offline"` arm MAY still fire if the network itself fails; its message must be the amended phrase.

## Scenarios index (delta)

| Scenario | REQ | Status | Likely spec file |
| --- | --- | --- | --- |
| S-5.a | REQ-5 (amended) | New wording — assert amended phrase, not the retired literal | `chat-api.spec.ts` |
| S-5.b | REQ-5 (amended) | New wording — assert amended phrase, not the retired literal | `chat-api.spec.ts`, `use-chat-stream.spec.ts` |
| S-5.c | REQ-5 (amended, new) | New — assert literal never surfaces when backend is up | `chat-api.spec.ts`, `routes/chat/index.spec.tsx` |

## Out of scope (mirrored from `proposal.md`)

- Any change to REQ-1..REQ-4, REQ-6, REQ-7. Owner: this delta; rationale: `0005:636` ("rendering, sanitization, the route guard and the error envelope — all frozen by `frontend-chat-layer1` and consumed unchanged").
- Backend code that would re-introduce the literal. Owner: none; rationale: the literal is retired, not relocated.
- The marketing `hero-proof`'s mock or the workplace `front-desk`'s mock. Owner: `cachicamas-frontend-workplace`; rationale: `explore.md §9.6, §9.7`.

## Cross-references

- Doc 0005 § "CH-05 — Retire the frontend's offline stub" (`docs/architecture/milestones/0005-cachicamas-chat-archetype-task-graph.md:626-675`)
- Doc 0005 § "Inconsistency register" row 4 (`0005:97`)
- Doc 0005 register row 4 (`0005:1045`)
- `chat-archetype-contract/spec.md:175-187` (R-CHT-009 — the gate that requires this delta)
- `openspec/specs/frontend-chat-layer1/spec.md:60-69` (REQ-5 as promoted — replaced by this delta on archive)
- `openspec/changes/cachicamas-chat-frontend-wire/explore.md` (read-only investigation that surfaced the retirement scope)
- `openspec/changes/cachicamas-chat-frontend-wire/proposal.md` (D-1, D-2, D-3)

## Review checklist

- [ ] reviewer can confirm the amended REQ-5 retains the `kind: "offline"` arm (D-1)
- [ ] reviewer can confirm the amended message substring is the new dev-honest phrase, not the retired literal (D-2)
- [ ] reviewer can confirm S-5.c asserts the literal never surfaces when the backend is up
- [ ] reviewer can confirm no REQ-1..REQ-4 / REQ-6 / REQ-7 wording changed
- [ ] reviewer can confirm the amendment text matches the rationale quoted from doc 0005
- [ ] reviewer can confirm the spec is dated and closes R-13, R-16, register row 4