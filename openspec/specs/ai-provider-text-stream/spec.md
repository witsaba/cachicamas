# Spec — translate the response lifecycle and text

> **Change**: `cachicamas-ai-provider-text-stream`
> **Milestone**: AI-28 · **Nodes**: AI-28.1.1, AI-28.1.2, AI-28.2, AI-28.3, AI-28.4, AI-28.5, AI-28.6 — all `[leaf]`
> **Phase**: spec (delta — ONE new capability `ai-provider-text-stream`, ADDED only; **zero modified capabilities**)
> **Canonical spec**: `openspec/specs/ai-provider-text-stream/spec.md` — created by `sdd-archive` from this delta
> **Format**: RFC 2119 keywords + Given/When/Then, per `openspec/config.yaml` `rules.specs`. Every scenario is independently verifiable.
> **Date**: 2026-08-04 · **Project**: cachicamas (witsaba)
> **Requirement IDs**: `R-ATS-0NN` · **Scenario IDs**: `S-ATS-0NN`
> **Wire-fact source of record**: [`citations.md`](../../citations.md) — claims **C1 … C8** against `github.com/openai/openai-openapi` at pinned commit `d4fb706e6e05d4cc9f1b33ca59b6e4f3e8edd439`
> **Binding predecessors, cited by identifier and never amended**: `ai-stream-lifecycle` (AI-14 … AI-16), `ai-text-events`, `ai-response-events`, `ai-provider-errors` (AI-19), `ai-completion-metadata` (AI-10/AI-13), `ai-stream-decoder` (AI-27), `ai-provider-conformance-suite` (AI-23), `ai-provider-client` (AI-25), `ai-request-translation` (AI-26) · [doc 0002](../../../../../docs/architecture/milestones/0002-cachicamas-ai-layer-1-task-graph.md) §§ AI-28.1.1 … AI-28.6 · [`proposal.md`](../../proposal.md)
> **Depends on**: AI-26, AI-27, AI-14 … AI-16, AI-19, AI-20.3, AI-23.2 · **Blocks**: AI-29 … AI-33
> **Named change dependency (AI-28.1.2 slice 2 only)**: `cachicamas-ai-conformance-lifecycle-amendment` must land `R-CNF-023` (capability-scoped conformance entry point) and `R-CNF-024` (read-only `Script`/`Step` introspection); `R-ATS-011` STOPS slice 2 if it does not.
> **Cross-milestone block**: `R-ATS-023` and `R-ATS-024` (AI-28.6) are **BLOCKED on AI-32.1**, which has not started.

---

## ADDED Requirements

## Purpose

`Translate` (AI-26) turns an `ai.Request` into wire bytes. `Decoder` (AI-27) turns bytes into `Frame`s. Nothing issues a request, reads a response, or produces a single `ai.Event`. This spec constrains the component that closes the arc: **the adapter's response lifecycle** — the producer that issues a request over real HTTP transport, drives the decoder, and emits a normalized, contract-conformant `ai.Event` stream.

Four distinctions shape every requirement below and are stated once here.

**The wire has no block framing; the adapter mints it (C3, load-bearing negative).** The Chat Completions stream defines no content-block start or stop signal of any kind. Text arrives solely as flat per-choice `delta.content` string fragments. Every text-block boundary this adapter emits is therefore **adapter-defined behavior satisfying AI-16's contract**, never a translation of a wire signal. `R-ATS-008` pins exactly when a block opens and closes; nothing downstream may treat those boundaries as vendor-reported facts.

**The wire has no mid-stream error frame (C6, negative).** The chat endpoint declares a single `200` response whose `text/event-stream` schema is `CreateChatCompletionStreamResponse` alone; no error-event union is reachable from it. This spec therefore **MUST NOT** specify parsing, recognising or mapping an in-band error payload. Frames that fail a pre-decode check, or that violate protocol order, are `R-ATS-020` … `R-ATS-024`'s typed malformed-response territory. Authoring the failure taxonomy is AI-32.1's, not this milestone's.

**Naming a category is not constructing a failure — and AI-28 is where construction starts.** AI-27 deliberately constructs no `ai.Failure` (`errors.go`, S-ASD-064): a pure byte decoder holds neither the `outputPreceded` fact `ai.MidStreamFailure` requires nor the carrier `ai.DeliveryPath` names. **AI-28 is the first milestone permitted to construct `ai.Failure` values** (`R-ATS-025`). `openaicompat.Category(err)` names the AI-19 category; this milestone supplies `outputPreceded` and the delivery path.

**Absent is not zero (AI-13.3).** `ai.TokenCount`'s zero value is *absent*; `ai.Tokens(0)` is a *reported zero*. A usage field the transcript never carried MUST read back absent. Separately, `usage`'s **presence** is asserted rather than silently accepted as an empty record (AI-24 § 13.1's opt-in trap) — the request side already emits `"stream_options":{"include_usage":true}` on every request (`appendStreamFields`, `body.go`).

Requirement count: **28** (`R-ATS-001` … `R-ATS-028`). Scenario count: **107** (`S-ATS-001` … `S-ATS-107`) — **89 [test]**, **18 [inspection]**.

*Counted mechanically over the scenario bullets below, not estimated. The eighteen `[inspection]` scenarios are `S-ATS-019`, `022`, `023`, `042`, `043`, `047`, `070`, `085`, `089`, `097`, `098`, `099`, `100`, `101`, `102`, `103`, `104`, `107`. Evidence rules key off these markings; a reconciliation finding a different split is reading a stale revision, not a downgraded test.*

> **Revision note (design-validation corrective, 2026-08-04) — revision 2, tally reconciliation.** This revision applies three corrections from the fresh-context design validation: the `S-ATS-032` / `S-ATS-074` self-defeating pair (`R-ATS-008`, `R-ATS-020`), the unrunnable conformance entry point (`R-ATS-011`), and the missing dialect-conventional label on the split-rune fixtures (`R-ATS-009`). Every correction restates existing text; **none adds, removes, renumbers or re-marks a requirement or scenario**. The tallies above were re-counted mechanically after editing and are unchanged: **28** requirements, **107** scenarios, **89 [test]** / **18 [inspection]**, with the same eighteen inspection IDs. The citation range widened to **C1 … C8** (see the citation gate).

## Definitions used by this spec

- **The producer** — the component this milestone ships: `(*Client).Stream(ctx, ai.Request) (<-chan ai.Event, error)`, satisfying `ai.ModelProvider`.
- **The carrier** — the `<-chan ai.Event` handed to the caller. Handover, not first event, is what makes a failure mid-stream (`ai.DeliveryMidStream`).
- **A chunk** — one decoded `openaicompat.Frame` whose data is a JSON object of schema `CreateChatCompletionStreamResponse`, `object` fixed to `"chat.completion.chunk"` (C1).
- **The terminal sentinel** — the SSE frame whose data payload is the six-byte literal `[DONE]` (C5).
- **The terminal chunk for a choice** — the chunk in which that choice's `finish_reason` is non-null (C2).
- **The usage chunk** — the extra chunk carrying `usage` with an **empty** `choices` array, streamed before `data: [DONE]` (C4).
- **Choice 0** — the choice whose `index` is `0`. This milestone maps choice 0 only; multi-choice fan-out is out of scope and no other choice index contributes text.
- **The content stream** — the ordered sequence of choice-0 deltas whose `content` is a JSON string (C7).
- **Normalized output event** — any `ai.Event` of a content-bearing kind already emitted on the carrier; the fact `outputPreceded` reports.
- **Scenario kind** — every scenario is marked **[test]** (runnable and failable under `make test` / `go test -race -count=1 ./...`) or **[inspection]** (a reviewer-checkable obligation over the artifact or shipped source, deterministic but not executed by the suite). Under Strict TDD, every **[test]** scenario MUST be demonstrated failing before the code satisfying it exists. A scenario that cannot be shown failing first is not a **[test]** scenario and MUST NOT be marked as one.

## The citation gate

Every statement of wire shape in this document resolves to a claim in `citations.md` (**C1 … C8**) or is explicitly labelled **dialect-conventional** with a fixture-pin obligation. `R-ATS-027` makes a violation detectable rather than merely discouraged. Citing a claim number that does not exist in `citations.md` is a spec defect, not a typo.

> **Revision note (design-validation corrective, 2026-08-04).** The claim range widened from **C1 … C7** to **C1 … C8**: `citations.md` gained **C8** (`CompletionUsage` — the streaming `usage` object's field set, required list and schema identity) after this spec's first revision. Range bounds updated here, in the header, in the claim table below, in `R-ATS-027` and in `S-ATS-102`. No existing citation changed meaning; no requirement or scenario was added, removed or re-marked.

| Claim | What it establishes |
| --- | --- |
| **C1** | Chunk schema `CreateChatCompletionStreamResponse`; `object` = `chat.completion.chunk`; required top-level `id`, `model`, `created`, `choices`, `object`; `usage`/`system_fingerprint`/`service_tier` optional |
| **C2** | Per-choice required `delta`, `finish_reason`, `index`; `finish_reason` enum {`stop`,`length`,`tool_calls`,`content_filter`,`function_call`}, `nullable: true`, null mid-stream, set on the terminal chunk whose `delta` is empty |
| **C3** | **NEGATIVE** — no block/content-part start or stop signal exists in Chat Completions streaming; the adapter mints boundaries |
| **C4** | `stream_options.include_usage` → one extra chunk before `data: [DONE]` carrying `usage` with an empty `choices` array; every preceding chunk carries `usage: null`; **not guaranteed if the stream is interrupted** |
| **C5** | Terminal sentinel is the SSE frame `data: [DONE]`; for Chat Completions it is **prose-documented** (the `include_usage` description presupposes it), **not a schema constant** — the only `enum: - "[DONE]"` at this commit belongs to the Assistants `DoneEvent` |
| **C6** | **NEGATIVE** — no mid-stream error payload is reachable from `POST /chat/completions`; only a `200` with a single stream schema |
| **C7** | Delta schema `ChatCompletionStreamResponseDelta` declares exactly five optional properties — `content`, `function_call` (deprecated), `tool_calls`, `role`, `refusal` — with no `required` list; `content` and `refusal` are additionally nullable; `include_obfuscation` documents an undeclared `obfuscation` field providers may add |
| **C8** | The streaming `usage` object is `CompletionUsage` — required `prompt_tokens`/`completion_tokens`/`total_tokens`, plus two wholly optional detail objects — and is the **same** schema the non-streaming chat response uses |

## Requirement ownership by node

| Node | Requirements | Status |
| --- | --- | --- |
| AI-28.1.1 — producer shell | `R-ATS-001` … `R-ATS-006` | ready |
| AI-28.1.2 — text mapping + conformance bridge | `R-ATS-007` … `R-ATS-011` | ready |
| AI-28.2 — terminal discipline and truncation | `R-ATS-012` … `R-ATS-014` | ready |
| AI-28.3 — absent-versus-zero fidelity | `R-ATS-015` … `R-ATS-016` | ready |
| AI-28.4 — unknown and delta-less tolerance | `R-ATS-017` … `R-ATS-019` | ready |
| AI-28.5 — protocol-order violations | `R-ATS-020` … `R-ATS-022` | ready |
| AI-28.6 — pre-decode response checks | `R-ATS-023` … `R-ATS-024` | **BLOCKED on AI-32.1** |
| Charter boundary and construction ownership | `R-ATS-025` … `R-ATS-028` | ready |

---

## AI-28.1.1 — Producer shell `[leaf]`

### R-ATS-001 — The adapter exposes one streaming entry point and satisfies `ai.ModelProvider`

The adapter MUST expose exactly one streaming entry point, `(*Client).Stream(ctx context.Context, req ai.Request) (<-chan ai.Event, error)`, and `*Client` MUST satisfy `ai.ModelProvider` at run time. It MUST NOT add a second method to the interface it satisfies, and MUST NOT introduce any alternative streaming entry point, overload or variant alongside it.

#### Scenarios

- **S-ATS-001** *[test]* — Given the landed adapter, when `any(&Client{}).(ai.ModelProvider)` is asserted at run time, then `ok` is `true`.
- **S-ATS-002** *[test]* — Given the landed adapter, when its method set is reflected over, then a method named `Stream` is present and its signature is exactly `func(context.Context, ai.Request) (<-chan ai.Event, error)`.
- **S-ATS-003** *[test]* — Given the shipped `src/ai/provider.go`, when `agenttest`'s existing `TestModelProviderInterface_SignatureGuard` runs unchanged, then it still reports exactly one method on `ai.ModelProvider` — this milestone satisfies the interface and never widens it.

### R-ATS-002 — The pre-stream contract holds: validate once, before I/O, before consulting the context

`Stream` MUST validate `req` exactly once, before any I/O and before it consults `ctx`. An invalid — including a zero-value, never-constructed — request MUST fail **directly** with a returned error and MUST NOT create the carrier or the producer goroutine. Only once `req` is valid MAY an already-cancelled `ctx` short-circuit the same way, reported under `ai.FailureCategoryCancellation`. Nothing MUST be observable before validation passes: no network connection attempt, no channel allocation, no goroutine.

#### Scenarios

- **S-ATS-004** *[test]* — Given a zero-value `ai.Request`, when `Stream` is called against a test server that records every inbound connection, then a non-nil error is returned, the returned channel is nil, and the server recorded zero inbound requests.
- **S-ATS-005** *[test]* — Given a zero-value `ai.Request`, when the goroutine count is sampled immediately before and after the failing `Stream` call, then it is unchanged within the established leak tolerance.
- **S-ATS-006** *[test]* — Given a **valid** request and a context already cancelled before the call, when `Stream` is called, then it returns a `*ai.Failure` whose `Category()` is `ai.FailureCategoryCancellation` and whose `Delivery()` is `ai.DeliveryPreStream`, and the server recorded zero inbound requests.
- **S-ATS-007** *[test]* — Given an **invalid** request and a context already cancelled before the call, when `Stream` is called, then the reported failure is the **validation** failure, not the cancellation one — validation is ordered strictly first.

### R-ATS-003 — A minimal transcript drains end to end, sequenced from 1, closed exactly once

Given a recorded transcript replayed by a local `httptest.Server`, the producer MUST drain as a fully normalized, contract-conformant stream: a response-start event, then the transcript's normalized content, then exactly one terminal event. Sequence numbers MUST run `1 … N` contiguously. The carrier MUST be closed exactly once, on **every** exit path, and MUST NOT be closed twice or left open.

#### Scenarios

- **S-ATS-008** *[test]* — Given a minimal transcript of one chunk carrying `id`, `model` and a terminal choice, plus the terminal sentinel, when the stream is drained, then the observed kinds are `EventKindResponseStart` followed by `EventKindCompletion`, in that order, and nothing else.
- **S-ATS-009** *[test]* — Given the same drained stream, when `ai.CheckStream` is run over the recorded events, then it reports no violation and `Terminated()` is `true`.
- **S-ATS-010** *[test]* — Given the same drained stream, when the sequence numbers are read, then they are `1, 2, … N` with no gap, no repeat and no zero.
- **S-ATS-011** *[test]* — Given a fully drained carrier, when a second receive is attempted, then it reports a closed channel, and the whole test runs under `-race` without a double-close panic.

### R-ATS-004 — The vendor's response identity and served model land in the start event

The producer MUST emit exactly one `ai.ResponseStart`, constructed from the first chunk's top-level `id` (response identity) and `model` (served model) — both **required** fields of the chunk schema (C1). Both values MUST be carried **byte-exact**: not trimmed, lower-cased, parsed, re-formatted or substituted. The producer MUST NOT compare the served model to the requested model, and MUST NOT mint, default or synthesize either value.

#### Scenarios

- **S-ATS-012** *[test]* — Given a transcript whose first chunk carries `"id":"chatcmpl-Xq7"` and `"model":"gizmo-4o-2026-05-13"`, when the stream is drained, then the response-start event's `ResponseID()` equals `chatcmpl-Xq7` and `ServedModel()` equals `gizmo-4o-2026-05-13`, byte for byte.
- **S-ATS-013** *[test]* — Given a request whose requested model differs from the transcript's `model`, when the stream is drained, then `ServedModel()` reports the **transcript's** value and the stream does not fail — no comparison is performed.
- **S-ATS-014** *[test]* — Given a transcript whose first chunk `id` carries surrounding whitespace and mixed case, when the stream is drained, then `ResponseID()` reproduces those bytes unchanged.
- **S-ATS-015** *[test]* — Given a transcript whose first chunk omits `id` (or carries it empty), when the stream is drained, then the stream terminates with a typed malformed-response failure rather than a response-start event carrying a minted or empty identity — `ai.NewResponseStart` requires both fields non-empty.

### R-ATS-005 — Cancellation honors AI-20.3: every send selects, one closing site, no leak

Every send on the carrier MUST select on the caller's cancellation signal, so a caller that stops consuming and cancels never wedges the producer. The producer MUST have exactly **one** closing site, reached on completion, on terminal failure and on cancellation alike. After cancellation the producer MUST stop reading the response body and MUST release its transport resources. The sanctioned loss path — cancellation racing a saturated buffer, dropping late events and closing with no terminal event — is the only sanctioned loss path; a stream closing with no terminal event and never cancelled is a defect.

#### Scenarios

- **S-ATS-016** *[test]* — Given a transcript longer than the carrier's buffer served by a deliberately slow test server, when the consumer cancels mid-stream and stops receiving, then `Stream`'s goroutine exits within a bounded deadline and the goroutine count returns to its pre-call level within tolerance.
- **S-ATS-017** *[test]* — Given a cancelled mid-stream drain, when the recorded events are inspected, then either a terminal event is present or the stream closed with none — and in the latter case the case is the cancellation path, not a silent truncation of a live stream.
- **S-ATS-018** *[test]* — Given three scenarios in one table — normal completion, terminal failure, and cancellation — when each is drained, then in every case the carrier is observed closed exactly once and the whole table runs clean under `-race -count=1`.
- **S-ATS-019** *[inspection]* — Given the shipped producer source, when a reviewer reads it, then exactly one `close(` call on the carrier exists in the package, it is reached by a `defer` on the producer goroutine, and every send on the carrier is inside a `select` carrying a `ctx.Done()` case.

### R-ATS-006 — The two AI-25 boundary guards are reconciled by planned, cited edits

`TestClient_HasNoStreamingEntryPoint` (S-APC-030) and `TestClient_DoesNotSatisfyModelProviderAtRuntime` (S-APC-031) assert the **absence** of what this node lands. They MUST be flipped to assert the post-AI-28 form — `Stream` exists; `*Client` satisfies `ai.ModelProvider` — and MUST NOT be weakened, skipped or deleted. `doc.go`'s stale prose ("Streaming behaviour arrives at AI-26") MUST be corrected in place. If this milestone exports a new sentinel identity, `TestPolicy_NoNewSentinelsExported` (S-ART-054) MUST be reconciled by **adding the named entry with its spec citation to the enumerated allowlist** — never by re-freezing the list empty and never by deleting the guard.

#### Scenarios

- **S-ATS-020** *[test]* — Given the reconciled boundary test file, when `TestClient_HasNoStreamingEntryPoint`'s successor runs (`TestClient_HasStreamingEntryPoint`, renamed to match its post-flip meaning — verify-report S2), then it **fails** if `Stream` is absent and passes only when the method exists — the assertion's polarity is inverted, not removed.
- **S-ATS-021** *[test]* — Given the reconciled boundary test file, when `TestClient_DoesNotSatisfyModelProviderAtRuntime`'s successor runs (`TestClient_SatisfiesModelProviderAtRuntime`, renamed to match its post-flip meaning — verify-report S2), then it asserts `ok == true` and fails when `*Client` stops satisfying `ai.ModelProvider`.
- **S-ATS-022** *[inspection]* — Given `provider_boundary_test.go` after this milestone, when a reviewer reads it, then both tests are present with inverted assertions and a doc comment naming AI-28 and this spec's requirement — neither is `t.Skip`ped, commented out or deleted.
- **S-ATS-023** *[inspection]* — Given `doc.go` after this milestone, when a reviewer reads its opening prose, then the "streaming arrives at AI-26" claim is corrected to name AI-28, and every sentinel this milestone exports (if any) appears in S-ART-054's allowlist with its `R-ATS-0NN` citation.

---

## AI-28.1.2 — Text mapping `[leaf]`

### R-ATS-007 — Each choice-0 `content` string becomes exactly one text delta, byte-exact

For each chunk, the producer MUST read choice 0's `delta` object and, when its `content` field is a JSON **string**, emit exactly one `ai.TextDelta` carrying those bytes unmodified (C7: `content` is `anyOf` string or null; C2: `delta` is a required per-choice field). It MUST NOT accumulate, re-encode, trim, validate as UTF-8, or re-fragment the bytes. A `content` field that is JSON `null`, or absent entirely, MUST contribute no text delta and MUST NOT be read as the empty string (C7 makes `content` optional **and** nullable).

#### Scenarios

- **S-ATS-024** *[test]* — Given a transcript of three chunks whose choice-0 `content` values are `"Hola"`, `", "` and `"mundo"`, when the stream is drained, then exactly three `ai.TextDelta` events are emitted whose `Delta()` values are those three strings, in that order.
- **S-ATS-025** *[test]* — Given a chunk whose choice-0 delta carries `"content":null`, when the stream is drained, then no text delta is emitted for it, and a following chunk carrying `"content":"tail"` still emits its own delta with `Delta() == "tail"` — the null neither emits nor disturbs the next fragment.
- **S-ATS-026** *[test]* — Given a chunk whose choice-0 delta carries only `"role":"assistant"` and no `content` key at all, when the stream is drained, then no text delta is emitted for it and the following content chunk's delta is emitted unchanged.
- **S-ATS-027** *[test]* — Given a chunk whose choice-0 `content` is the empty string `""`, when the stream is drained, then exactly one text delta is emitted whose `Delta()` is `""` — `ai.NewTextDelta` legalizes an empty fragment (R-ATE-008), and a present-but-empty fragment is distinguishable from an absent one.

### R-ATS-008 — Block boundaries are minted by the adapter, not read from the wire (C3)

The wire carries **no** block start or stop signal of any kind (C3, negative, load-bearing). The adapter MUST therefore mint text-block boundaries itself, as **adapter-defined behavior satisfying AI-16's contract**, under exactly this rule:

1. **At most one** text block is minted per stream, covering choice 0's content stream, at block index `1`.
2. The block **opens** — one `ai.TextBlockStart` — immediately before the first text delta the producer would emit, and never earlier.
3. The block **closes** — one `ai.TextBlockEnd` — when choice 0's terminal chunk arrives or the stream terminates cleanly, whichever comes first, and always **before** the completion event.
4. If the content stream is empty (no chunk ever carried a choice-0 `content` string), **no block is minted at all**: no start, no end.

The producer MUST NOT claim, in code comment, log or event, that any block boundary was reported by the vendor. Choice indices other than `0` MUST NOT open a block at this milestone.

**What may legally follow the terminal chunk, and what may not.** Between choice 0's terminal chunk and the terminal sentinel the wire carries only **delta-less** chunks and the **usage chunk** (C4: the extra chunk carrying `usage` with an empty `choices` array, streamed before `data: [DONE]`); rule 3's close is minted against that shape. A chunk carrying a choice-0 **`content` string** after that choice's terminal chunk is **not** this rule's territory at all: it is `R-ATS-020` row 1's *delta after close* violation and MUST end the stream in a typed malformed-response terminal (`S-ATS-074`), never in a normal completion. The two situations share no fixture and no outcome.

> **Revision note (design-validation corrective, 2026-08-04).** Added the paragraph above and rewrote `S-ATS-032`'s fixture. As first written, `S-ATS-032` and `S-ATS-074` bound the **same** transcript shape — choice-0 content arriving after the terminal chunk and before the sentinel — to **opposite** terminals: `S-ATS-032` required a clean `Completion`, `S-ATS-074` required a malformed-response failure. No producer could satisfy both. Resolved in favour of `R-ATS-020` row 1 (the violation reading) and re-pointed `S-ATS-032` at the shape the wire actually produces in that window, per C4. `S-ATS-031`'s premise was made explicit for the same reason — see its own note. No requirement, scenario or marking was added or removed.

#### Scenarios

- **S-ATS-028** *[test]* — Given a transcript of two content chunks and a terminal chunk, when the stream is drained, then the emitted kinds are exactly `ResponseStart, TextBlockStart, TextDelta, TextDelta, TextBlockEnd, Completion`, in that order.
- **S-ATS-029** *[test]* — Given the same drained stream, when every text event's `Block()` is read, then all three report block index `1`.
- **S-ATS-030** *[test]* — Given a transcript whose first chunk carries only `"role":"assistant"` and whose second carries the first `content` string, when the stream is drained, then the block start is emitted **between** them — after the role-only chunk produced nothing, immediately before the first delta.
- **S-ATS-031** *[test]* — Given a transcript with no `content` string anywhere — a role-only opening chunk carrying `id` and `model`, then a terminal chunk carrying `"finish_reason":"stop"` — when the stream is drained, then no `TextBlockStart` and no `TextBlockEnd` are emitted, and `ai.CheckStream` reports no unterminated-block violation. *(The opening chunk is named explicitly so this fixture is not `R-ATS-020` row 5's close-without-open shape.)*
- **S-ATS-032** *[test]* — Given a transcript whose terminal chunk is followed, before the sentinel, only by **delta-less chunks and the usage chunk** (C4) — no choice-0 `content` string anywhere after the terminal chunk — when the stream is drained, then the block end precedes the completion event, no text delta follows the block end, and the stream terminates in a normal completion. *(Contrast `S-ATS-074`: a **content** chunk in this same window is `R-ATS-020` row 1's violation and terminates in a malformed-response failure instead.)*

### R-ATS-009 — Concatenated deltas reconstruct the text byte-exactly, including across a split multi-byte rune

Concatenating the emitted deltas of a block in order MUST reproduce the transcript's content bytes exactly. The producer MUST NOT repair, replace or normalize a fragment that is invalid UTF-8 on its own because a multi-byte rune split across a delta boundary (AI-16.2, R-ATE-007). Fixtures proving this MUST place **meaningful content after the split point**, so a producer that silently dropped or reordered the tail cannot pass.

**Dialect-conventional premise carrying a fixture-pin obligation (`R-ATS-027`'s own form).** The split-rune shape is **not** cited wire behavior and no `C`-claim establishes it — none may be invented for it. A rune cannot be split across two `delta.content` values on a legal wire: JSON string escapes encode **code points**, not bytes, and RFC 8259 requires the text itself to be valid UTF-8, so no conforming producer can place half a rune inside a JSON string. The shape is expressible **only** as raw invalid-UTF-8 bytes written directly into the fixture's `content` value. These three scenarios are therefore **byte-level robustness pins against proxy or vendor misbehavior**, labelled dialect-conventional here, with the obligation discharged at fixture level: the committed transcript fixtures MUST carry those exact raw bytes, and the tests MUST read them through a path that preserves bytes rather than one that substitutes `U+FFFD`.

Labelling them dialect-conventional does **not** weaken them. All three remain **[test]** and MUST NOT be relaxed, softened to `[inspection]`, or dropped on the grounds that the wire cannot produce the shape — the requirement is precisely that a non-conforming intermediary cannot corrupt the adapter's output.

> **Revision note (design-validation corrective, 2026-08-04).** Added the two paragraphs above. `S-ATS-033`/`034`/`035` previously read as if a split rune were ordinary wire behavior, which `citations.md` never established and RFC 8259 forbids; under `R-ATS-027` an uncited wire-shape statement is a defect unless explicitly labelled dialect-conventional with a discharged fixture pin, so the label was owed. Behavior, fixtures, assertions and `[test]` markings are unchanged — this is an honest-labelling correction, not a downgrade. Counts unchanged: 28 requirements, 107 scenarios, 89 `[test]` / 18 `[inspection]`.

#### Scenarios

- **S-ATS-033** *[test]* — Given a **dialect-conventional robustness fixture** (see the pin obligation above) whose two adjacent choice-0 `content` values carry the raw bytes `"caf\xc3"` and `"\xa9 shop, abierto"` — the 2-byte encoding of `é` split across the boundary with fifteen further bytes after it — when the stream is drained, then concatenating the two deltas yields `"caf\xc3\xa9 shop, abierto"` byte for byte.
- **S-ATS-034** *[test]* — Given the same dialect-conventional fixture, when the first delta is inspected alone, then its bytes are `"caf\xc3"` unmodified — no replacement character, no truncation, no re-encoding.
- **S-ATS-035** *[test]* — Given a **dialect-conventional robustness fixture** splitting a 4-byte emoji across three deltas as raw bytes, with trailing text after the third, when the stream is drained, then the concatenation is byte-identical to the intended string and the emitted delta count is exactly three.

### R-ATS-010 — The completion event is built from the terminal chunk's `finish_reason` (C2)

`finish_reason` is JSON `null` on every chunk until the terminal chunk, which carries one of {`stop`, `length`, `tool_calls`, `content_filter`, `function_call`} with an **empty** `delta` (C2). The producer MUST emit exactly one `ai.Completion` whose finish reason is mapped from that value, and MUST NOT emit a completion while `finish_reason` is null. A `finish_reason` value outside C2's enum MUST NOT be silently mapped to `stop`.

#### Scenarios

- **S-ATS-036** *[test]* — Given a transcript whose terminal chunk carries `"finish_reason":"stop"` with `"delta":{}`, when the stream is drained, then exactly one completion event is emitted and its `FinishReason()` is the canonical stop outcome.
- **S-ATS-037** *[test]* — Given a transcript whose terminal chunk carries `"finish_reason":"length"`, when the stream is drained, then the completion's `FinishReason()` is the canonical length outcome, distinguishable from stop.
- **S-ATS-038** *[test]* — Given a transcript of five content chunks each carrying `"finish_reason":null`, when the stream is drained before the terminal chunk arrives, then no completion event was emitted at any point.
- **S-ATS-039** *[test]* — Given a terminal chunk carrying `"finish_reason":"quota_burned"` — outside C2's enum — when the stream is drained, then the stream terminates with a typed malformed-response failure and no completion event claiming `stop` is emitted.

### R-ATS-011 — AI-23.2's text conformance cases pass against real transport

The adapter MUST supply an `agenttest.Factory` whose `New` builds a **real** `*Client` speaking real HTTP to a local `httptest.Server` that replays the scripted interaction as an SSE transcript, and the conformance suite's **capability-scoped** entry point MUST report the `CapStreamingText` cases satisfied against it.

**The whole-registry entry point is not the target.** `agenttest.RunConformance` runs the entire seventeen-case registry with required-capability waivers forbidden, and its per-case runner is unexported, so a text-only provider cannot pass it and cannot select a subset of it. This milestone MUST therefore run the text-capability cases through a **capability-scoped conformance entry point**, and the bridge's transcript renderer MUST read the scripted events through a **read-only script introspection surface** rather than reaching into `agenttest`'s unexported state.

**Both surfaces are a named external dependency, not this change's to author.** They are `R-CNF-023` (capability-scoped conformance entry point) and `R-CNF-024` (read-only `Script`/`Step` introspection), ADDED by the separate change **`cachicamas-ai-conformance-lifecycle-amendment`**. This change MUST reference them by those requirement identifiers only, and MUST NOT invent, re-declare, shadow, vendor or locally re-implement either surface. The bridge remains **adapter-owned and test-only**: it MUST live in this package's test sources, MUST NOT be exported from a production file, and MUST NOT modify `agenttest` or the `ai-provider-conformance-suite` capability. Its mechanics are design's; the observable obligation is this requirement.

**STOP rule — the dependency is hard.** If `cachicamas-ai-conformance-lifecycle-amendment` lands **without** `R-CNF-023` and `R-CNF-024`, AI-28.1.2 slice 2 **STOPS** and the gap is raised against `ai-provider-conformance-suite` as its own change. It MUST NOT be worked around by modifying `agenttest` here, by forking a case, or by substituting an invented entry point.

**Recorded tension, not an absorbed edit.** `agenttest/conformance_text.go`'s two shipped cases assert **exact** event counts (`len(events) != 4`; `len(events) != 1`) over streams that contain no response-start event. A producer conforming to `R-ATS-004` necessarily emits one. If the bridge cannot satisfy both without changing the conformance suite, AI-28.1.2 **STOPS** and the reconciliation is raised as its own change against `ai-provider-conformance-suite` — it MUST NOT be absorbed here, and the conformance case MUST NOT be weakened, skipped or forked.

> **Revision note (design-validation corrective, 2026-08-04).** As first written this requirement targeted an **unrunnable** API: `S-ATS-040` called `agenttest.RunConformance`, which executes all seventeen registered cases with required-capability waivers forbidden and no exported per-case runner, so a text-only factory could never pass it; and `S-ATS-041`'s renderer premise assumed it could read scripted events that `agenttest.Script`/`Step` expose no accessor for. Rewritten to consume the two surfaces `cachicamas-ai-conformance-lifecycle-amendment` is adding — `R-CNF-023` and `R-CNF-024` — cited by identifier so no symbol is invented here, with the STOP rule made explicit. `S-ATS-040`, `S-ATS-041` and `S-ATS-043` restated accordingly; `S-ATS-042` unchanged. No requirement, scenario or marking was added, removed or re-marked.

#### Scenarios

- **S-ATS-040** *[test]* — Given the adapter-owned factory, when the `R-CNF-023` capability-scoped conformance entry point is run over the `CapStreamingText` cases **only**, then ~~the returned record reports every text-capability case satisfied~~ **every text-capability case passes in place — the scoped entry point returns nothing** *(restated 2026-08-10: `R-CNF-028`, promoted 2026-08-09, made the scoped runner return no record or verdict precisely so a scoped run can never masquerade as conformance evidence; the landed test already consumed no return value)*, no case outside that capability is executed, and the run performs no outbound network access beyond `127.0.0.1` — the whole-registry `agenttest.RunConformance` is not invoked.
- **S-ATS-041** *[test]* — Given the bridge's transcript renderer, which reads scripted events **solely** through `R-CNF-024`'s read-only `Script`/`Step` introspection, when a script carrying two deltas that split a multi-byte rune is rendered and replayed, then the drained deltas reconstruct the scripted text byte-exactly — the bridge itself does not repair encoding and reaches no unexported `agenttest` state.
- **S-ATS-042** *[inspection]* — Given the shipped sources, when a reviewer reads them, then the bridge exists only in `_test.go` files of `openaicompat`, and no file under `src/agenttest/` was modified by this change.
- **S-ATS-043** *[inspection]* — Given the named dependency and the recorded tension above, when a reviewer checks the landed state, then (a) `cachicamas-ai-conformance-lifecycle-amendment` is open or landed carrying **both** `R-CNF-023` and `R-CNF-024`; (b) the bridge reaches the scoped entry point and the introspection surface by those requirement identifiers and declares no substitute of its own; and (c) either both text conformance cases pass unmodified or the reconciliation is open as its own change against `ai-provider-conformance-suite` — in no case is a conformance case skipped, forked or edited inside this change, and if the amendment landed without those two requirements, slice 2 is recorded as **stopped** rather than worked around.

---

## AI-28.2 — Terminal discipline and truncation `[leaf]`

### R-ATS-012 — `data: [DONE]` is clean termination, never payload, never truncation (C5)

The producer MUST recognise the frame whose data payload is the six-byte literal `[DONE]` as **clean termination** of the stream (C5). It MUST NOT attempt to parse it as a chunk, MUST NOT emit any event from it, and MUST NOT let it trip the truncation detector. **Posture, stated exactly as C5 states it**: for Chat Completions this sentinel is **prose-documented** at the pinned commit — presupposed by the `include_usage` description — and is **not** a schema constant; the only `enum: - "[DONE]"` at that commit belongs to the Assistants `DoneEvent`. This is therefore a **dialect-conventional** recognition carrying a fixture-pin obligation, discharged by a committed transcript fixture, not a schema-derived fact.

#### Scenarios

- **S-ATS-044** *[test]* — Given a transcript ending with a terminal chunk, then `data: [DONE]`, then EOF, when the stream is drained, then the stream ends cleanly with the completion as its terminal event and no failure event.
- **S-ATS-045** *[test]* — Given the same transcript, when every emitted event is inspected, then none carries the literal text `[DONE]` in any field — the sentinel produced no payload.
> **Revision note (slice-6 coordinator ruling, 2026-08-04).** The original wording — "no trailing blank line before EOF" — described an unterminated sentinel *frame*. Under the frozen AI-27 framing contract (AI-27.6 EOF discipline; WHATWG dispatch: an unterminated event is never dispatched) an unterminated sentinel was never received, and the decoder deliberately exposes no pending-buffer state to special-case it — that shape is R-ATS-013's truncation case. The scenario's intent, that nothing is required *after* the sentinel, is preserved with a well-formed sentinel frame at exact EOF. Slice 3's landed tests already assert the corrected reading.

- **S-ATS-046** *[test]* — Given a transcript whose final bytes are `data: [DONE]` followed by its terminating blank line and then immediate EOF — nothing after the sentinel frame — when the stream is drained, then the outcome is identical to a transcript with further trailing bytes: a clean Completion, with the sentinel never parsed as payload and never reported as truncation.
- **S-ATS-047** *[inspection]* — Given the shipped fixture set, when a reviewer reads it, then at least one committed transcript fixture contains the exact bytes `data: [DONE]`, and the source recognising it cites **C5** together with its prose-documented, dialect-conventional posture.

### R-ATS-013 — A stream closing without the terminal sentinel is a typed terminal error with partial output preserved and flagged

When the connection closes before the terminal sentinel arrives, the producer MUST end the stream with a **typed terminal error event**, never silent success on a truncated message. The failure MUST report `ai.FailureCategoryMalformedResponse` (surfaced through `openaicompat.Category` over `ErrTruncated`), MUST be constructed as `ai.MidStreamFailure` with `ai.DeliveryMidStream`, and MUST carry `PartialOutput() == true` when any normalized output event preceded it and `false` when none did. Every event already emitted MUST remain delivered — partial output is preserved, not discarded.

#### Scenarios

- **S-ATS-048** *[test]* — Given a transcript of two content chunks whose connection then closes with no sentinel, when the stream is drained, then the last event is an error event, its `Category()` is `ai.FailureCategoryMalformedResponse`, and the two text deltas preceding it are present and byte-exact.
- **S-ATS-049** *[test]* — Given that same stream, when the terminal failure is inspected, then `PartialOutput()` is `true` and `Delivery()` is `ai.DeliveryMidStream`.
- **S-ATS-050** *[test]* — Given a transcript whose connection closes after the response start but before any content chunk, when the stream is drained, then the terminal failure reports `PartialOutput() == false` while `Delivery()` remains `ai.DeliveryMidStream` — handover, not first content, decides the delivery path.
- **S-ATS-051** *[test]* — Given a transcript cut in the middle of a `data:` line, when the stream is drained, then the pending partial frame is **not** dispatched as a complete chunk and no event derives from it.

### R-ATS-014 — Frames arriving after the terminal sentinel and before EOF are ignored

Frames that arrive after the terminal sentinel but before EOF MUST be ignored: not parsed into events, not surfaced as a failure, and not treated as a protocol-order violation. The stream's terminal event MUST remain the one determined before the sentinel.

#### Scenarios

- **S-ATS-052** *[test]* — Given a transcript of a terminal chunk, `data: [DONE]`, then a further well-formed content chunk, then EOF, when the stream is drained, then no text delta derives from the post-sentinel chunk and the stream's last event is the completion.
- **S-ATS-053** *[test]* — Given a transcript of `data: [DONE]` followed by a **malformed** frame then EOF, when the stream is drained, then the stream still ends cleanly with no failure event.
- **S-ATS-054** *[test]* — Given a transcript of `data: [DONE]` followed by a second `data: [DONE]` then EOF, when the stream is drained, then exactly one terminal event was emitted and no duplicate-terminal violation is reported.

---

## AI-28.3 — Absent-versus-zero fidelity `[leaf]`

### R-ATS-015 — Usage fields the transcript never carried read back absent, not zero

A usage field never present in the transcript MUST read back as an **absent** `ai.TokenCount` — `Count()` returning `ok == false` — and MUST NOT be flattened to `ai.Tokens(0)`. A field the provider reported as `0` MUST read back as a **reported zero** — `Count()` returning `(0, true)`. `usage: null` on a non-final chunk (C4) MUST contribute nothing and MUST NOT overwrite an already-populated field with absent or zero values.

#### Scenarios

- **S-ATS-055** *[test]* — Given a usage chunk carrying only `prompt_tokens` and `completion_tokens`, when the completion's `Usage()` is inspected, then the fields the transcript never carried report `ok == false` from `Count()`.
- **S-ATS-056** *[test]* — Given a usage chunk carrying an explicit `0` for a field, when the completion's `Usage()` is inspected, then that field reports `(0, true)` — distinguishable from the absent case in the same assertion table.
- **S-ATS-057** *[test]* — Given two transcripts identical except that one omits a usage field and the other reports it as `0`, when both are drained, then their completion events' usage records are **not** equal.
- **S-ATS-058** *[test]* — Given a transcript whose content chunks all carry `"usage":null` (C4) followed by a populated usage chunk, when the stream is drained, then the completion's usage carries the final chunk's reported counts and no field was reset by the preceding nulls.

### R-ATS-016 — The usage chunk's presence is asserted, never silently accepted as empty

The producer's own test suite MUST **assert** that the usage chunk arrived and populated the completion — the opt-in trap AI-24 § 13.1 records is a test that passes just as well when `usage` never appears. Per C4, the usage chunk carries an **empty `choices` array** and is streamed **before** `data: [DONE]`; the producer MUST accept an empty `choices` array without treating it as a protocol-order violation, MUST NOT emit a text event from it, and MUST NOT require it — C4 states the final usage chunk is not guaranteed if the stream is interrupted. A stream that ends cleanly without a usage chunk MUST yield a completion whose usage is wholly absent, not zero, and MUST NOT fail.

#### Scenarios

- **S-ATS-059** *[test]* — Given a transcript containing a usage chunk with an empty `choices` array before the sentinel, when the stream is drained, then the completion's usage reports at least one **present** count, asserted positively rather than by absence of failure.
- **S-ATS-060** *[test]* — Given a transcript from which the usage chunk has been deliberately removed, when the same assertion from `S-ATS-059` is run, then it **fails** — proving the assertion can distinguish a populated usage from a missing one.
- **S-ATS-061** *[test]* — Given a transcript whose usage chunk carries an empty `choices` array, when the stream is drained, then no text event and no additional response-start derive from it, and no protocol-order violation is reported.
- **S-ATS-062** *[test]* — Given a transcript that terminates cleanly with the sentinel but carries no usage chunk at all, when the stream is drained, then the completion is emitted, the stream reports no failure, and every usage field reports `ok == false`.

---

## AI-28.4 — Unknown and delta-less tolerance `[leaf]`

### R-ATS-017 — Unknown frame types and undeclared delta fields are skipped without corrupting adjacent accumulation

The producer MUST tolerate unknowns without corrupting the stream. Specifically: a frame whose SSE event type is not the default type MUST be skipped; a chunk whose top-level `object` is not `chat.completion.chunk` (C1) MUST be skipped; and a choice-0 delta carrying fields beyond C7's five declared properties — including the documented, **undeclared** `obfuscation` field that `include_obfuscation` records providers may add (C7) — MUST be accepted, with the unknown fields ignored and the declared ones mapped normally. Unknown-field tolerance is **mandatory**, not best-effort. Skipping MUST NOT drop, reorder or merge the deltas adjacent to the skipped material.

#### Scenarios

- **S-ATS-063** *[test]* — Given a transcript of `content:"alpha"`, then a frame with an unrecognised SSE event type, then `content:"omega"`, when the stream is drained, then exactly two deltas are emitted, in order, with values `alpha` and `omega`.
- **S-ATS-064** *[test]* — Given a chunk whose choice-0 delta carries `"content":"bravo"` alongside `"obfuscation":"xxxxxx"`, when the stream is drained, then one delta with value `bravo` is emitted and the stream reports no failure.
- **S-ATS-065** *[test]* — Given a chunk whose choice-0 delta carries `"content":"charlie"` alongside two invented sibling fields, when the stream is drained, then one delta with value `charlie` is emitted and no failure is reported.
- **S-ATS-066** *[test]* — Given a frame whose JSON object carries `"object":"chat.completion"` — not the chunk discriminator — between two content chunks, when the stream is drained, then it is skipped and the two surrounding deltas are emitted unchanged and in order.
- **S-ATS-067** *[test]* — Given a chunk whose choice-0 delta carries `"refusal":"no"` and no `content` (C7), when the stream is drained, then no text delta is emitted for it and the following content chunk's delta is unaffected — refusal mapping is not this milestone's.

### R-ATS-018 — Delta-less and content-less shapes normalize cleanly

A stream carrying no choice-0 `content` string anywhere MUST normalize cleanly: no block is minted (`R-ATS-008` rule 4), the completion is still emitted, and `ai.CheckStream` reports no violation. A stream whose only content is the **empty string** MUST mint a block containing exactly one empty delta, reconstructing to the empty string. Neither shape MUST be reported as truncation, as an incomplete stream, or as a failure. *(Doc 0002's "a content block that opens and closes with zero deltas" is unreachable verbatim on this wire, because C3 gives the wire no block-open signal; these are its two reachable normalizations.)*

#### Scenarios

- **S-ATS-068** *[test]* — Given a transcript of a role-only chunk, a terminal chunk and the sentinel, when the stream is drained, then the emitted kinds are exactly `ResponseStart, Completion` and `ai.CheckStream` reports no violation.
- **S-ATS-069** *[test]* — Given a transcript whose single content chunk carries `"content":""`, when the stream is drained, then the emitted kinds are `ResponseStart, TextBlockStart, TextDelta, TextBlockEnd, Completion` and the one delta's `Delta()` is `""`.
- **S-ATS-070** *[inspection]* — Given the shipped mapping source, when a reviewer reads it, then no code path enforces a minimum delta count, a minimum block count or a non-empty-text requirement anywhere.

### R-ATS-019 — Keep-alive frames interleaved anywhere are inert

SSE comment frames and other keep-alive material MUST perturb nothing: no event, no failure, no change to sequence numbering, and no effect on delta accumulation, wherever they appear — before the first chunk, between chunks, mid-block, and after the terminal chunk.

#### Scenarios

- **S-ATS-071** *[test]* — Given two transcripts identical except that one interleaves a keep-alive comment before every chunk, when both are drained, then the two emitted event sequences are equal in kind, order, sequence number and payload.
- **S-ATS-072** *[test]* — Given a transcript with a keep-alive between two deltas of one block, when the stream is drained, then the concatenated deltas are byte-identical to the keep-alive-free twin.
- **S-ATS-073** *[test]* — Given a transcript with keep-alives after the terminal chunk and before the sentinel, when the stream is drained, then the stream terminates cleanly and the terminal event is unchanged.

---

## AI-28.5 — Protocol-order violations `[leaf]`

### R-ATS-020 — Every structural violation in the table yields a typed malformed-response terminal

A buggy proxy can emit structurally impossible frame sequences at will. For each row of the table below, the producer MUST end the stream with a **typed malformed-response terminal error** — `ai.FailureCategoryMalformedResponse`, constructed as `ai.MidStreamFailure` over `ai.DeliveryMidStream`, with `PartialOutput()` reporting whether normalized output preceded it — and MUST preserve every event already emitted. It MUST NOT panic, MUST NOT silently discard the violating frame and continue, and MUST NOT emit a normal completion.

| Row | Wire shape at this dialect | Doc 0002's structural name |
| --- | --- | --- |
| 1 | a choice-0 `content` string arriving after that choice's terminal chunk **and before the terminal sentinel** | delta after close |
| 2 | a second chunk carrying a non-null `finish_reason` for choice 0 | duplicate close |
| 3 | a chunk whose top-level `id` differs from the established response identity | second response start |
| 4 | a choice item whose `index` is negative or absent while carrying `content` | delta with no open block |
| 5 | a terminal chunk (non-null `finish_reason`) with no preceding chunk establishing the response | close without open |

> **Revision note (design-validation corrective, 2026-08-04).** Row 1's window is now stated explicitly — **after the terminal chunk and before the terminal sentinel** — for two reasons. First, it is the half of the `S-ATS-032`/`S-ATS-074` disambiguation that lives here: content in that window is a violation, while delta-less and usage chunks in that window are `R-ATS-008`'s clean close. Second, it keeps row 1 from silently overlapping `R-ATS-014`, which already rules that frames arriving **after** the sentinel are ignored and explicitly **not** protocol-order violations (`S-ATS-052`). The row's outcome, its scenario and its structural name are unchanged.

#### Scenarios

- **S-ATS-074** *[test]* — Given row 1's transcript — `content:"one"`, terminal chunk, then `content:"two"`, all **before** the sentinel — when the stream is drained, then the deltas emitted are exactly `["one"]`, the terminal event is a malformed-response failure, and no delta carries `two`. *(Contrast `S-ATS-032`: the same window carrying only delta-less and usage chunks completes cleanly.)*
- **S-ATS-075** *[test]* — Given row 2's transcript with two non-null `finish_reason` chunks for choice 0, when the stream is drained, then the terminal event is a malformed-response failure and exactly one completion or zero completions were emitted — never two.
- **S-ATS-076** *[test]* — Given row 3's transcript whose third chunk carries a different `id`, when the stream is drained, then exactly one response-start event was emitted, the terminal event is a malformed-response failure, and the deltas preceding the offending chunk are preserved byte-exact.
- **S-ATS-077** *[test]* — Given row 4's transcript whose choice item omits `index` while carrying `content`, when the stream is drained, then the terminal event is a malformed-response failure and no delta derives from that choice item.
- **S-ATS-078** *[test]* — Given row 5's transcript beginning with a terminal chunk that carries no usable response identity, when the stream is drained, then the terminal event is a failure and `PartialOutput()` is `false`.
- **S-ATS-079** *[test]* — Given all five rows driven from one table-driven test, when each row is drained, then every row's terminal failure reports `ai.FailureCategoryMalformedResponse` and every row's assertion **fails** if the producer instead emitted a normal completion.

### R-ATS-021 — Malformed payload of a *known* type yields the same typed terminal, distinguished from an unknown type

A frame whose payload is malformed for its declared, **known** type — invalid JSON, or a chunk violating C1's required top-level field set — MUST yield the same typed malformed-response terminal as `R-ATS-020`. This MUST be distinguished from `R-ATS-017`'s unknown types, which skip by contract: an unrecognised shape skips, a recognised-but-broken shape fails.

#### Scenarios

- **S-ATS-080** *[test]* — Given a frame whose data is `{"object":"chat.completion.chunk","choices":` — truncated invalid JSON — when the stream is drained, then the terminal event is a malformed-response failure.
- **S-ATS-081** *[test]* — Given a chunk carrying `object` and `choices` but omitting the required `model` (C1), when the stream is drained, then the terminal event is a malformed-response failure.
- **S-ATS-082** *[test]* — Given one table containing both an unknown-type frame and a broken known-type frame, when each is drained, then the unknown-type case completes cleanly with its adjacent deltas intact while the broken known-type case terminates in a failure — the two outcomes are asserted in the same test and are different.

### R-ATS-022 — No input sequence causes a panic, and partial output always survives

No frame sequence — including every `R-ATS-020` row, `R-ATS-021`'s malformed payloads, empty frames, and frames whose `choices` array is empty or absent — MUST cause a panic in the producer or its consumer-facing carrier. Whatever normalized events were already emitted MUST remain delivered, in order and byte-exact, ahead of the terminal failure.

#### Scenarios

- **S-ATS-083** *[test]* — Given a table of every violating transcript in this node driven through one runner with a `recover()` probe, when all rows run, then zero panics are recorded.
- **S-ATS-084** *[test]* — Given a violating transcript preceded by three good deltas, when the stream is drained, then those three deltas are present, in order, byte-exact, ahead of the terminal failure.
- **S-ATS-085** *[inspection]* — Given the shipped state-machine source, when a reviewer reads it, then no map read is dereferenced without an existence check and no slice index is taken without a bound check on any path reachable from a decoded frame.

---

## AI-28.6 — Pre-decode response checks `[leaf]` — **BLOCKED on AI-32.1**

> **Blocked, in scope, sequenced last.** Doc 0002 states `AI-28.6 — Depends on: AI-28.1.1, AI-32.1`. AI-32.1 — the failure-status taxonomy — has not started. These two requirements are written now because they are **pre-decode HTTP behaviors** citable from doc 0002's own test list, and they consume the taxonomy rather than authoring it. If AI-32.1 has not landed when slices 1–6 are green, `R-ATS-023` and `R-ATS-024` become a **recorded carryover naming AI-32.1 as their blocker** — never silently dropped and never satisfied by an invented taxonomy.

### R-ATS-023 — A non-stream content type is refused before decoding, with a bounded body excerpt

When a success response carries a content type that is not the streaming media type — a proxy's HTML error page being the canonical case — the producer MUST refuse **before** handing any byte to the decoder, and MUST report a typed error carrying a **bounded** excerpt of the body. The content-type match MUST tolerate parameters (for example a charset parameter) and MUST be case-insensitive on the type and subtype. The excerpt MUST be bounded by a documented constant and MUST NOT reproduce credential material.

#### Scenarios

- **S-ATS-086** *[test]* — Given a `200` response whose content type is `text/html` and whose body is an HTML error page, when `Stream` is drained, then the stream terminates with a typed error, no text event was emitted, and the error's rendered text contains a bounded excerpt of the page.
- **S-ATS-087** *[test]* — Given a `200` response whose content type is `TEXT/EVENT-STREAM; charset=utf-8`, when the stream is drained, then it is **accepted** and decodes normally — the match tolerates case and parameters.
- **S-ATS-088** *[test]* — Given a `200` response with a non-stream content type and a body far larger than the documented excerpt bound, when the stream is drained, then the excerpt carried by the error is no longer than that bound.
> **Revision note (re-verify corrective, 2026-08-04, rev 4; W3 disposition: `obs-579c70ab5030532b` / Engram #2525).** S-ATS-089's second conjunct originally required "the credential-scan guard covers the excerpt path". That conjunct was unsatisfiable as written: the credential-scan guard's designed scope is **external-package** (`package openaicompat_test`) files only, and every test file in this change is internal-package **by load-bearing necessity** — the error-mapping milestone plants above-threshold credential sentinels in internal test files precisely because the guard cannot see them there, so widening the guard to internal files would break the build on those deliberate plants. By inspection, ZERO of the 13 AI-28 test files carry an external `package xxx_test` clause, so the guard's designed scope holds by posture, not by accident. The conjunct is narrowed to the guard's designed scope, with the disclosed residual (a package-internal sweep is possible future hardening, owned by a future guard change, not this milestone) recorded where the excerpt is built. The W3 disposition is **accepted, not fixed** — the spec acknowledges the rationale rather than treating it as a gap. Same corrective form as S-ATS-046 rev 3.

- **S-ATS-089** *[inspection]* — Given the shipped source and its tests, when a reviewer reads them, then the excerpt bound is a named constant, and the excerpt-path source records the credential posture: the credential-scan guard's designed scope is external-package (`package openaicompat_test`) test files, this change's tests are internal-package by load-bearing necessity (the error-mapping milestone's planted-sentinel interaction), zero internal-package occurrences of external-package clause form exist by inspection across the 13 AI-28 test files, and the package-internal sweep is a recorded possible future hardening rather than a silent gap.
- **S-ATS-090** *[test]* — Given a `200` response carrying **no** content-type header at all, when the stream is drained, then the response is refused before decoding with the same typed error rather than decoded optimistically.

### R-ATS-024 — Failure statuses route to the failure mapping before any decode

A response whose HTTP status indicates failure MUST be routed to AI-32's failure mapping **before** any byte reaches the decoder, observable as **zero normalized content events preceding the terminal**. The producer MUST NOT author its own status taxonomy; it consumes AI-32.1's.

#### Scenarios

- **S-ATS-091** *[test]* — Given a `429` response with a JSON error body, when `Stream` is called, then the drained stream carries zero content events before its terminal failure event.
- **S-ATS-092** *[test]* — Given a `500` response whose body is a valid SSE transcript, when the stream is drained, then no text delta is emitted from that body — the status decided the outcome before decoding.
- **S-ATS-093** *[test]* — Given a table of failure statuses across the 4xx and 5xx classes, when each is drained, then every row yields a terminal failure whose category comes from AI-32's mapping and none yields a completion.

---

## Charter boundary and construction ownership

### R-ATS-025 — AI-28 owns failure construction; the decoder only names a category

This milestone is the first permitted to **construct** `ai.Failure` values from adapter errors. It MUST derive the category through `openaicompat.Category(err)` for decoder errors, MUST supply the `outputPreceded` fact itself, and MUST choose the delivery path by **handover**: `ai.PreStreamFailure` (`ai.DeliveryPreStream`) before the carrier is returned, `ai.MidStreamFailure` (`ai.DeliveryMidStream`) after — including when no content preceded the failure. It MUST NOT modify `src/ai` or `openaicompat/errors.go`, and MUST NOT widen `ai.FailureCategory`: the nine-member vocabulary is closed and append-only (R-AIP-005), and AI-27's recorded cap-exceeded compromise stands.

#### Scenarios

- **S-ATS-094** *[test]* — Given a truncated transcript, when the terminal failure is inspected, then `errors.Is(err, ai.ErrMalformedResponse)` holds and the failure's `Category()` equals what `openaicompat.Category(openaicompat.ErrTruncated)` reports.
- **S-ATS-095** *[test]* — Given a pre-handover failure and a post-handover failure in one table, when each is inspected, then the first reports `ai.DeliveryPreStream` and the second `ai.DeliveryMidStream`.
- **S-ATS-096** *[test]* — Given a post-handover failure that emitted no content event, when it is inspected, then `Delivery()` is `ai.DeliveryMidStream` and `PartialOutput()` is `false` — the two facts are independent.
- **S-ATS-097** *[inspection]* — Given `ai.FailureCategories()` before and after this change, when the two are compared, then the vocabulary is identical — nine members, unwidened.

### R-ATS-026 — The charter boundary holds: no reasoning, no tool calls, no usage mapping, no new dependency

This milestone MUST NOT map reasoning content (AI-29), MUST NOT map `tool_calls` or `function_call` deltas in either direction (AI-30), MUST NOT implement the full usage field mapping or the cumulative merge (AI-31.2), and MUST NOT author the failure-status taxonomy (AI-32.1). `backend/agent/go.mod` MUST remain at **zero** requires; a third-party dependency is a hard blocker, not a tradeoff.

#### Scenarios

- **S-ATS-098** *[inspection]* — Given `backend/agent/go.mod` before and after this change, when the two are compared, then the file is byte-identical and its require set is empty.
- **S-ATS-099** *[inspection]* — Given the shipped production sources of this change, when a reviewer reads their imports, then every import is stdlib or an in-repo `src/ai` path.
- **S-ATS-100** *[inspection]* — Given the shipped mapping source, when a reviewer searches it, then no code path emits a reasoning event or a tool-call event, and `tool_calls`/`function_call` deltas are handled only by `R-ATS-017`'s skip rule.
- **S-ATS-101** *[inspection]* — Given the shipped usage handling, when a reviewer reads it, then it establishes presence and absent-versus-zero only, with no cumulative merge and no per-field taxonomy that AI-31.2 owns.

### R-ATS-027 — Every wire-shape claim resolves to a citation

Every statement of wire shape in the shipped source, tests and this document MUST cite a claim from `citations.md` (**C1 … C8**) or be explicitly labelled **dialect-conventional** with a discharged fixture-pin obligation. A citation reference that does not resolve to an existing claim in `citations.md` is a defect. Inference MUST NOT be recorded as fact.

The two dialect-conventional labels this spec carries are `R-ATS-012`'s `data: [DONE]` recognition (pinned by a committed transcript fixture) and `R-ATS-009`'s split-rune robustness fixtures (pinned by committed raw-byte fixtures). Any third such label MUST be introduced with its pinning fixture named in the same place.

#### Scenarios

- **S-ATS-102** *[inspection]* — Given every `C1` … `C8` reference in this spec and in the shipped source comments, when each is resolved against `citations.md`, then every reference names an existing claim and no reference names a claim number outside `C1` … `C8`.
- **S-ATS-103** *[inspection]* — Given the shipped source comments describing field names, framing or terminal behavior, when a reviewer reads them, then each carries either a `C`-claim citation or an explicit dialect-conventional label with its pinning fixture named.
- **S-ATS-104** *[inspection]* — Given `citations.md`'s pinned commit, when it is compared against `doc.go`'s existing request-side provenance pin, then both name `d4fb706e6e05d4cc9f1b33ca59b6e4f3e8edd439` — one pin, not two.

### R-ATS-028 — No mid-stream error frame is recognised or specified (C6)

Because the Chat Completions stream schema models **no** in-band error payload at the pinned commit (C6, negative), the producer MUST NOT parse, recognise, map or document an in-stream error frame. A frame that merely *looks* like an error payload MUST be handled by the existing rules — skipped as an unknown shape (`R-ATS-017`) or failed as a broken known shape (`R-ATS-021`) — and MUST NOT be given a bespoke error path. Recognising a vendor's in-band error frame is AI-32's charter.

#### Scenarios

- **S-ATS-105** *[test]* — Given a transcript containing a frame whose data is `{"error":{"message":"boom","type":"server_error"}}` between two content chunks, when the stream is drained, then no bespoke error event derives from it and the outcome is exactly what `R-ATS-017` or `R-ATS-021` prescribes for that shape.
- **S-ATS-106** *[test]* — Given a transcript containing an SSE frame whose event type is `error`, when the stream is drained, then it is skipped as an unknown event type and the surrounding deltas are emitted unchanged and in order.
- **S-ATS-107** *[inspection]* — Given the shipped source, when a reviewer searches it, then no symbol, branch or comment claims to map a Chat Completions in-stream error payload, and any such claim would contradict C6.

---

## Coverage of doc 0002's test lists

Every test-list item in doc 0002 §§ AI-28.1.1 … AI-28.6 maps to at least one scenario.

| Doc 0002 item | Scenarios |
| --- | --- |
| AI-28.1.1 #1 — minimal transcript drains, sequenced from 1, closed once | `S-ATS-008` … `S-ATS-011` |
| AI-28.1.1 #2 — response identity and model in the start event | `S-ATS-012` … `S-ATS-015` |
| AI-28.1.1 #3 — AI-20.3 cancellation and close discipline | `S-ATS-016` … `S-ATS-019` |
| AI-28.1.2 #1 — block start/deltas/end; byte-exact across a split rune | `S-ATS-024` … `S-ATS-035` |
| AI-28.1.2 #2 — AI-23.2 text case passes against real transport | `S-ATS-040` … `S-ATS-043` |
| AI-28.2 #1 — close without terminal → typed error, partial preserved and flagged | `S-ATS-048` … `S-ATS-051` |
| AI-28.2 #2 — data-only sentinel is clean termination, never payload, never truncation | `S-ATS-044` … `S-ATS-047` |
| AI-28.2 #3 — post-terminal, pre-EOF frames ignored | `S-ATS-052` … `S-ATS-054` |
| AI-28.3 #1 — absent usage fields are absent, not zero | `S-ATS-055` … `S-ATS-062` |
| AI-28.4 #1 — unknown frame/delta/block types skipped without corruption | `S-ATS-063` … `S-ATS-067` |
| AI-28.4 #2 — a zero-delta block normalizes cleanly | `S-ATS-068` … `S-ATS-070` |
| AI-28.4 #3 — keep-alives interleaved anywhere are inert | `S-ATS-071` … `S-ATS-073` |
| AI-28.5 #1 — the structural-violation table, never a panic | `S-ATS-074` … `S-ATS-079`, `S-ATS-083` … `S-ATS-085` |
| AI-28.5 #2 — malformed payload of a known type, distinguished from unknown | `S-ATS-080` … `S-ATS-082` |
| AI-28.6 #1 — non-stream content type refused pre-decode, bounded excerpt | `S-ATS-086` … `S-ATS-090` |
| AI-28.6 #2 — failure statuses route pre-decode, zero content events first | `S-ATS-091` … `S-ATS-093` |
| Guard reconciliation (proposal, wave-4 recorded lesson) | `S-ATS-020` … `S-ATS-023`, `S-ATS-097` |
| Citation gate (proposal § "The citation gate") | `S-ATS-047`, `S-ATS-102` … `S-ATS-104` |
| C6 negative (coordinator ruling) | `S-ATS-105` … `S-ATS-107` |

## Non-vacuity discipline

Scenarios above are written so their fixtures **can distinguish an implemented producer from an unimplemented one**. Concretely: `S-ATS-033` and `S-ATS-035` place meaningful content **after** the split point, so dropping the tail fails; `S-ATS-057` and `S-ATS-060` are negative controls asserting the twin case is *not* equal / that the assertion *fails* when the fact is removed; `S-ATS-082` asserts the skip and the failure outcomes are **different** in one test; `S-ATS-020` and `S-ATS-021` require the flipped guards to fail on regression, not merely to pass. A scenario whose fixture would pass identically against a no-op producer is a spec defect and MUST be rewritten, not marked `[inspection]` to avoid the problem.
---

## Charter delta (AI-31) — appended by `sdd-archive`

> **Archive provenance**: this block was appended on 2026-08-05 from
> `openspec/changes/cachicamas-ai-provider-completion/specs/ai-provider-text-stream/spec.md`
> during the archive of `cachicamas-ai-provider-completion` (AI-31). The pre-existing
> canonical spec (lines 1 … 493, AI-28 archive) is preserved byte-identically above.
> **No existing scenario was modified**; only an additive delta section was appended.

# Delta for ai-provider-text-stream

> **Change**: `cachicamas-ai-provider-completion` (AI-31) · **Phase**: spec · **Date**: 2026-08-04
> The `ai-provider-text-stream` capability is still an **in-flight, unpromoted** change
> (`openspec/changes/cachicamas-ai-provider-text-stream/specs/ai-provider-text-stream/spec.md`). This delta is
> written against that in-flight text and reconciled at archive, per the wave's recorded spec-promotion lesson.

## MODIFIED Requirements

### R-ATS-026 — The charter boundary holds as of AI-28: no reasoning, no tool calls, no usage taxonomy at that time, no new dependency

That milestone MUST NOT map reasoning content (AI-29), MUST NOT map `tool_calls` or `function_call` deltas in
either direction (AI-30), MUST NOT implement the full usage field mapping or the cumulative merge (AI-31.2),
and MUST NOT author the failure-status taxonomy (AI-32.1). `backend/agent/go.mod` MUST remain at **zero**
requires; a third-party dependency is a hard blocker, not a tradeoff.

The usage clause is **as of AI-28**. AI-31.2 has since discharged it: the per-field usage taxonomy and the
merge pin now live in the `ai-provider-completion` capability (`R-ACP-005`, `R-ACP-006`, `R-ACP-007`). The
boundary is recorded as moved, never quietly invalidated.
(Previously: `S-ATS-101` asserted the shipped usage handling carries "no per-field taxonomy that AI-31.2
owns" as a standing property; landing AI-31 makes that reading false.)

#### Scenarios

- **S-ATS-098** *[inspection]* — Given `backend/agent/go.mod` before and after that change, when the two are compared, then the file is byte-identical and its require set is empty.
- **S-ATS-099** *[inspection]* — Given the shipped production sources of that change, when a reviewer reads their imports, then every import is stdlib or an in-repo `src/ai` path.
- **S-ATS-100** *[inspection]* — Given the shipped mapping source, when a reviewer searches it, then no code path emits a reasoning event or a tool-call event, and `tool_calls`/`function_call` deltas are handled only by `R-ATS-017`'s skip rule.
- **S-ATS-101** *[inspection]* — Given the usage handling **as shipped by AI-28**, when a reviewer reads it at that revision, then it establishes presence and absent-versus-zero only, with no cumulative merge and no per-field taxonomy; and given the usage handling after AI-31, when a reviewer reads it, then the per-field taxonomy and the merge pin are present and are governed by `R-ACP-005`…`R-ACP-007` rather than by this requirement.