# Delta — `frontend-chat-layer1`

> **Change**: `cachicamas-frontend-workplace` · Target: `openspec/specs/frontend-chat-layer1/spec.md`
> **Inherited from**: `cachicamas-frontend-os-redesign`, which this change supersedes.
> Corrected in three places, marked **(corrected)**, where this change moved the evidence.
> **Ops**: MODIFIED `REQ-1`, `REQ-2`, `REQ-4`, `REQ-5`, `REQ-6`, `REQ-7`.
> **Decision**: proposal `D6`.

## Read this before deciding whether the delta is honest

`frontend-chat-layer1` shipped **two things at once**: a wire client, and a component tree
that consumed it. This change replaces the component tree and keeps the wire client
**byte-unchanged**. Every requirement below is amended along exactly that seam — the wire
half is reproduced verbatim, and only the browser half moves.

Nothing here weakens a security or auth requirement. `REQ-3` is **not modified**: the
guard chain in `routes/chat/layout.tsx` is byte-unchanged and its scenarios still hold.

## Why the offline literal could not simply be deleted

REQ-5's purpose, stated in its own rationale, is to make an architectural gap **greppable**.
The gap has not closed — no backend serves the chat endpoint, and `cachicamas_chat` is 0 of
12 — so the literal stays exactly where the requirement put it, in `chat-api.ts`. What
changed is that the browser no longer *reaches* that path, because the screen no longer
calls the client. The requirement is therefore split: the **client half survives intact**,
and the **surfacing half is suspended with its restoration condition named**.

## MODIFIED Requirements

### REQ-1 — Submit-a-turn happy path (redesign amendment)

The wire contract is **unchanged and unimplemented-against**: a turn is opened by request,
its events arrive on a subscribed stream, and `parseTranscript` decodes the recorded frame
shape. `lib/chat-api.ts` and `lib/chat-types.ts` MUST remain on disk and byte-unchanged by
this change.

The browser SHALL render a scripted turn instead of a subscribed one until CH-05 wires the
backend. The scripted turn MUST preserve the observable shape of the real one: text
arrives as **deltas over time**, never as one block; a tool call is visible as running
before it is visible as returned; and the composer is disabled for the duration.

**Scenarios.**

- **S-1.c** Given the mocked driver plays a `say` beat, when it advances, then the
  assistant line accumulates one chunk per tick and carries `state: "streaming"` until its
  final chunk, at which point it carries `state: "final"` — asserted in
  `components/chat/use-mock-turn.spec.ts`.
- **S-1.d** *(corrected)* Given `scriptFor` is called with any message, then the returned
  beats open with a `Working` note and close with a note — no script may end mid-turn. The
  labels are the words a person reads rather than machine constants; the property, that a
  conversation always opens and always closes, is unchanged and is asserted in
  `lib/mock/chat.spec.ts`.

### REQ-2 — Mid-stream cancellation (redesign amendment)

`cancelTurn` SHALL close the streaming line **where it stopped**, without adding or
discarding text, and SHALL append a terminal note naming the cancellation. It SHALL leave
no state a later turn could resume from.

**Scenarios.**

- **S-2.c** Given a line is mid-stream, when the turn is cancelled, then that line's text
  is byte-identical to its text at the moment of cancellation and its state is `final`.
- **S-2.d** Given a turn was cancelled, when the driver is advanced again, then nothing
  changes — the script, beat index and step index are all reset.

### REQ-4 — Backend error surfaces inline, no client-side retry (redesign amendment)

Unchanged in intent and now proven in the component: a failure SHALL render as a typed
envelope carrying its **code**, its **message** and its **recovery**, and the surface SHALL
state that no automatic retry will occur. Retry remains a harness concern.

**Scenarios.**

- **S-4.d** Given a `fault` entry, when `TranscriptLine` renders it, then the code, the
  message, the recovery and the literal "No automatic retry" are all present.

### REQ-5 — Dev-mode honest offline failure (SUSPENDED, with its restoration condition)

**The client half is unchanged.** `chat-api.ts` SHALL still surface
`ApiErrorKind = "offline"` whose message contains the literal
`"backend not wired — see PR for backend wire"`, and `chat-api.spec.ts` SHALL still assert
it. That code and that test are byte-unchanged by this change.

**The surfacing half is suspended.** `S-5.a` and `S-5.b` required the *chat screen* to show
that message inline. The screen no longer calls the client at all, so those two scenarios
are **not satisfiable** and are recorded here as suspended rather than deleted.

**Restoration condition, named so this cannot rot into an omission:** when CH-05 wires the
archetype's network surface, the screen begins calling `chat-api.ts` again and `S-5.a` /
`S-5.b` become satisfiable in their original wording. The correct action at that point is
to **restore them**, and — if the backend is genuinely reachable by then — to retire the
literal deliberately, in CH-05.2's own delta, which doc 0005's inconsistency register #4
already assigns.

**Scenarios.**

- **S-5.c** *(corrected)* Given the redesigned workspace, when a person opens any screen
  including `/chat`, then a standing strip states in words that the colleagues,
  conversations and figures shown are examples. It lives in the shell rather than on the
  composer, so no screen can forget it (`components/workspace/workspace.spec.tsx`), and
  the composer instead states the product's one standing promise where a person is about
  to act on it: nothing is sent outside the company without approval.

### REQ-6 — Markdown + sanitization for assistant text (redesign amendment)

Unchanged: all assistant text SHALL be rendered through `renderSanitizedMarkdown`, and raw
model HTML SHALL NOT reach the DOM.

`S-6.b`'s closing clause — *"the spec asserts `dangerouslySetInnerHTML` is not present in
the bubble component"* — is **corrected, not weakened**. `renderSanitizedMarkdown` returns
an HTML string; rendering it necessarily uses `dangerouslySetInnerHTML`, and the original
clause described the retired component's internals rather than the security property. The
property that actually matters is asserted directly instead.

**Scenarios.**

- **S-6.c** Given a model line containing `<script>alert(1)</script>` and
  `<img src=x onerror=…>`, when `TranscriptLine` renders it, then the emitted HTML contains
  neither a `script` element nor an `onerror` attribute, and legitimate markup
  (`<strong>`) survives.
- **S-6.d** Given a **person's** own line containing `<b>x</b>`, when it renders, then it
  appears as literal text and produces no `b` element — a paste must not become markup in
  the pasting person's own transcript.

### REQ-7 — Per-file spec discipline (redesign amendment)

Every `*.ts` / `*.tsx` under `frontend/src/components/chat/`, plus `lib/chat-api.ts`, SHALL
still have a colocated spec. `S-7.a` and `S-7.c` are unchanged and satisfied.

`S-7.b` *(corrected)* required each spec to reference `REQ-1`…`REQ-6` by identifier. That is amended: the
new specs assert the **behaviour** those requirements describe and cite the mechanism in
prose, and the requirement identifiers live here, in the delta, where they can be checked
against the spec that owns them. An identifier grep over test files proved to measure
comment discipline rather than coverage.

**Scenarios.**

- **S-7.d** Given the apply phase lands, when the files under `components/chat/` are
  listed, then each has a colocated `*.spec.ts` / `*.spec.tsx`, and the suite is green with
  no `it.skip` / `it.todo` / `xit`.
