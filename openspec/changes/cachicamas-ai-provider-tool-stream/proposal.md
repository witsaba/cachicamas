# Proposal — translate the tool-call stream

> **Change**: `cachicamas-ai-provider-tool-stream` · **Milestone**: AI-30 · **Nodes**: AI-30.1 … AI-30.5
> **Phase**: proposal · **Date**: 2026-08-04 · **Project**: cachicamas (witsaba)
> **Wave**: 4 "Connect the vendor" · base = the AI-28 chain tip at apply time (today `feat/ai-28-7-pre-decode-checks`)
> **Depends on**: AI-18 (`ai.ToolCallStart/Delta/End`), AI-28 (producer + `mapperState`), AI-19, AI-23.3
> **Parallel with**: AI-29, AI-31 — logically independent, but the feature-branch chain **serializes the applies**; all three edit `stream_state.go`, so the second and third to land rebase.

---

## Intent

The request half of tool calling shipped at AI-26.5: `Translate` renders `tool_calls` on the assistant
message and `role:"tool"` results, byte-proven (`tool_result_test.go`). The response half is **discarded**.
`wireDelta` (chunk.go) decodes exactly one field — `content` — so the model's answer to a tool declaration
falls through `R-ATS-017`'s skip rule and vanishes silently: no event, no error, no trace.

Until this lands, Layer 2 has no agent loop. `agenttest`'s four `CapToolCalls` conformance cases (AI-23.3)
have never been run against real transport, because the adapter advertises text only.

## Scope

### In scope

1. **Per-call accumulation (AI-30.1)** — decode the delta's `tool_calls` array; open one neutral block per
   vendor call keyed by its wire position; accumulate argument fragments in **per-call buffers**; identity
   and tool name readable from the start event before any argument byte; assembly exactly once, at the
   call's end; a **documented per-call byte cap** with a typed failure (AI-27.5 bounds one frame — this
   bounds the sum).
2. **Empty and zero-fragment calls (AI-30.2)** — an empty first fragment is a no-op; a call closing with
   zero accumulated bytes normalizes to `ai`'s canonical empty-arguments form (`"{}"`, `tool_call.go`'s
   `emptyToolArguments`) rather than failing on empty input; a whole (unfragmented) call normalizes
   identically to its fragmented twin.
3. **Argument-byte fidelity (AI-30.3)** — reassembled arguments byte-identical to the concatenated
   fragments, including escape sequences and extreme numeric forms; **nothing re-marshals them**. The
   byte-preserving `json.RawMessage` + `unquoteJSONString` idiom (chunk.go D2) is reused, never re-invented.
4. **Truncation and malformation (AI-30.4)** — a mid-arguments cutoff and a malformed assembled payload
   both terminate with the same typed failure, **carrying the raw partial fragment, bounded**; never a
   panic, never a silent discard.
5. **Ordinal preservation (AI-30.5)** — each call's ordinal position observable regardless of interleaving,
   derived stream-side per `R-ATC-012` (this package ships no accumulator).
6. **Conformance surface** — extend the test-only bridge to render tool-call events into SSE, and run
   `agenttest.RunConformanceFor(…, CapToolCalls)`. This is the milestone's acceptance proof.

### Out of scope

- **Validating arguments against a tool's schema, and executing anything** (charter). Bytes, not meaning.
- **Widening `ai.FailureCategory`** — see "The tenth category" below.
- **Reasoning (AI-29)** and **usage/finish-reason mapping (AI-31)**.
- **Any new module dependency.** `backend/agent/go.mod` stays at zero requires — a hard blocker.
- **A new exported sentinel.** The cap and malformation causes stay unexported, wrapping
  `ai.ErrMalformedResponse` (the `errMalformedIdentity` precedent), so the `S-ART-054` allowlist —
  today exactly `ErrFrameTooLarge` / `ErrTruncated` — is untouched and AI-32's future `ErrInBandErrorFrame`
  entry has no position to collide with.

## Capabilities

### New Capabilities

- `ai-provider-tool-stream`: the OpenAI-compatible adapter's tool-call response mapping — per-call
  accumulation and bounding, empty/zero-fragment normalization, argument-byte fidelity, truncation and
  malformation failure, and ordinal preservation.

### Modified Capabilities

- **None.** `ai-tool-call-events`, `ai-provider-conformance-suite`, `ai-provider-errors` and
  `ai-stream-lifecycle` are cited by identifier and satisfied, never amended.

## How `R-ATS-026` is superseded, cleanly

`R-ATS-026` (sibling change `cachicamas-ai-provider-text-stream`, **unarchived** — no promoted spec exists
to delta) reads: *"This milestone MUST NOT map `tool_calls` or `function_call` deltas in either direction
(AI-30)."* It is a **charter boundary scoped to AI-28's own shipped slice**, and it names AI-30 as the
owner of the deferred behaviour. AI-30 therefore **discharges** it; it does not amend or weaken it.

The only live artifact is `S-ATS-100`, an **inspection** scenario ("no code path emits … a tool-call event,
and `tool_calls`/`function_call` deltas are handled only by `R-ATS-017`'s skip rule"). Verified:
**no executable guard encodes it** — `grep` over `openaicompat/*_test.go` finds no test asserting the
absence of tool-call emission, so nothing flips like `S-APC-030/031` did. The sanctioned reconciliation is
documentary: this change's spec carries **one requirement recording the discharge** ("`R-ATS-026`'s
tool-call clause and `S-ATS-100`'s tool-call half bind AI-28's shipped state as of AI-28.6; AI-30 supersedes
them for `tool_calls`, and AI-29 independently for reasoning"), plus a `doc.go` correction in the same
disclose-a-correction form the package already uses. AI-29 makes the *same-shaped, disjoint* statement for
reasoning — the two must not both rewrite one shared clause, or the chain conflicts.

## The tenth category — recorded, not absorbed

`errors.go` states the resource-exhaustion category may be appended **only once a second consumer exists**,
and names **AI-30.1 item 4 as that second consumer**. This change **records that the second consumer now
exists** and stops there. The per-call cap keeps `ErrFrameTooLarge`'s exact compromise form — cap exceeded →
typed failure naming `FailureCategoryMalformedResponse`, with the compromise disclosed in the same comment
shape. Widening `ai.FailureCategory` belongs to the taxonomy owner as its own change, never to this
mapping-only milestone.

## Approach

**`mapperState` grows a per-call map, not a second state machine.** Today it mints one text block at the
constant `textBlockIndex = 1`. Tool calls need stream-wide-unique block indices that never collide with the
text block, keyed by the vendor's per-element position — so block-index minting becomes a small, explicit
allocator (design decision D1, below). Fragment accumulation lives in that map; nothing else in the
producer shell changes, and `stream.go`'s `isOutputEvent` widens to count tool-call kinds toward
`PartialOutput`.

**Failure causes follow the landed idiom**: unexported `fmt.Errorf(… %w, ai.ErrMalformedResponse)` values
beside the existing five, surfaced through `emitFailure`'s fallback category. The bounded raw fragment
reuses `capture.go`'s existing `captureLimit` rather than inventing a second bound.

## The citation gate — the delta-side element shape is UNCITED

`C7` establishes that the delta object carries `tool_calls`, *"array of
`ChatCompletionMessageToolCallChunk`"*, and a deprecated `function_call` object. **The element's own schema
is not quoted anywhere in `citations.md`.** The request-side shape *is* cited (`doc.go` claims 2/3,
`ChatCompletionMessageToolCall`) and **must not be assumed identical** to the streaming chunk element.

`sdd-spec` MUST NOT state a delta-side field name as fact before a **C9** citation lands against the pinned
`openai-openapi` commit `d4fb706e6e05d4cc9f1b33ca59b6e4f3e8edd439`. The open claims are listed in
"Citation gaps" below. Inference is not admissible.

## Affected Areas

| Area | Impact | Description |
| --- | --- | --- |
| `.../openaicompat/chunk.go` | Modified | `wireDelta` gains `tool_calls` (+ `function_call` disposition); new element type, byte-preserving |
| `.../openaicompat/stream_state.go` | Modified | Per-call accumulation map, block-index minting, cap, call-close rule |
| `.../openaicompat/stream.go` | Modified | `isOutputEvent` widens to tool-call kinds |
| `.../openaicompat/errors.go` | Modified | Second-consumer note recorded; no new exported sentinel |
| `.../openaicompat/doc.go` | Modified | `R-ATS-026` discharge disclosed |
| `.../openaicompat/*_test.go` | New | One proof file per node; SSE fixtures; bridge tool-call rendering |
| `backend/agent/src/ai/`, `src/agenttest/` | Read-only | AI-18 constructors and the conformance suite consumed as-is |
| `backend/agent/go.mod` | Unchanged | Zero requires |

## Delivery

`delivery_strategy: auto-chain` · `chain_strategy: feature-branch-chain` · budget 5000 · node = PR boundary.

| Slice | Node | Targets | Note |
| --- | --- | --- | --- |
| 1 | AI-30.1 per-call accumulation + cap + bridge rendering | AI-28 chain tip | Largest. **Split if** the cap/failure machinery or the bridge extension alone exceeds budget |
| 2 | AI-30.2 empty and zero-fragment calls | slice 1 | Turn on `CapToolCalls` here if all four cases pass |
| 3 | AI-30.3 argument-byte fidelity | slice 1 (rebased on 2) | Fixture-heavy, small production delta |
| 4 | AI-30.4 truncation and malformation | slice 3 | Depends on AI-30.1 + AI-19 |
| 5 | AI-30.5 ordinal preservation | slice 4 | Small; re-verifies the `CapToolCalls` run |

`sdd-tasks` owns the line forecast; repo precedent runs ×2–4 over naive estimates.

## Risks

| Risk | Likelihood | Mitigation |
| --- | --- | --- |
| The delta element's field names are implemented from the request-side schema by analogy | **High** | The C9 citation gate above is a spec precondition; the node stops rather than guesses |
| Block-index minting collides with the text block, or with AI-29's reasoning blocks landing in parallel | **High** | D1 is an explicit design decision, decided once for all three milestones; the chain serializes, so the second lander rebases against a *stated* scheme, not a discovered one |
| "When does a call end?" has no wire signal and is invented | **High** | Listed as an open citation (C9-6); assembly-once (AI-30.1 item 3) cannot be specified without it |
| The bridge's tool-call SSE rendering is larger than AI-30.1 itself | Medium | Named as slice-1 scope up front, pre-declared as a candidate split (the AI-28.1.2 bridge precedent) |
| The bounded raw fragment leaks model-generated argument bytes into logs | Medium | Deliberate disposition required: `capturedBody` deliberately does *not* reproduce bytes (`R-AEM-016`) while `nonStreamContentType` does — the spec must pick one and say why |
| `R-ATS-026`/`S-ATS-100` are silently contradicted instead of discharged | Medium | One explicit supersession requirement + `doc.go` correction, disjoint from AI-29's |

## Rollback Plan

Every change is additive inside `openaicompat/`, plus one `doc.go` prose correction. `src/ai`, `agenttest`
and Waves 1–3 compile and pass identically with or without it; `go.mod` is untouched. `git revert` per slice
commit is a clean boundary; reverting the milestone returns the adapter to text-only mapping, with
`tool_calls` deltas falling back through `R-ATS-017`'s skip rule exactly as today. No consumer exists yet
(Layer 2 has not started), so nothing downstream breaks.

## Dependencies

- **AI-18** (`ai.NewToolCallStart/Delta/End`) — shipped.
- **AI-28** producer shell, `mapperState`, decoder, conformance bridge — merged into the chain base.
- **AI-23.3** `CapToolCalls` conformance cases — shipped, never yet run against transport.
- **A C9 citation round** against the pinned `openai-openapi` commit — **not started; blocks the spec phase.**
- Stdlib only; strict TDD; `make test` (`go test -race -count=1 ./...`) and `make lint` clean.

## Success Criteria

- [ ] A transcript interleaving fragments for two concurrent calls reconstructs both exactly, with zero
      cross-contamination.
- [ ] Call identity and tool name are readable from the start event before any argument byte arrives.
- [ ] Arguments are byte-identical to the concatenated fragments, asserted byte-equal against the fixture;
      nothing re-marshals them.
- [ ] A zero-fragment call closes as canonical empty arguments; a whole call and its fragmented twin
      normalize identically.
- [ ] A runaway fragment sequence into one call hits the documented cap with a typed failure.
- [ ] A mid-arguments cutoff and a malformed assembled payload both yield a typed failure carrying the
      bounded raw partial fragment — never a panic.
- [ ] Each call's ordinal is observable regardless of interleaving.
- [ ] `RunConformanceFor(…, CapToolCalls)` passes all four AI-23.3 cases against real transport.
- [ ] Every delta-side wire-shape claim in `spec.md` resolves to a C9 citation, or its node is stopped.
- [ ] `ai.FailureCategories()` is unwidened; no new exported sentinel; `go.mod` byte-identical.

## Citation gaps for `sdd-spec` (the C9 round)

1. `ChatCompletionMessageToolCallChunk`'s full property set, types, and `required` list.
2. Whether the element carries a per-element `index`, and whether it indexes the call array (the key
   AI-30.1 item 1's "vendor's block position" depends on).
3. Whether `id` and `function.name` appear only on a call's first element, and whether either may itself
   arrive fragmented across chunks (decides whether the start event can be emitted immediately).
4. Whether `function.arguments` is a plain string carrying fragments (byte-preservation applies) and
   whether `type` is a single-member enum.
5. The deprecated `function_call` delta's fragmentation semantics — and therefore whether AI-30 maps it or
   records a deliberate skip.
6. **How a call is known to be complete** — there is no per-call end signal on the wire. Assembly-once
   (AI-30.1 item 3) and the malformation check (AI-30.4) both hinge on this.

## Proposal question round (unanswered — executor could not prompt interactively)

1. Should the deprecated `function_call` delta be mapped, or refused/skipped with a recorded disposition?
2. May the bounded raw partial fragment appear in the failure's rendered text (support-debuggable), or must
   it be reachable only through an accessor (leak-averse, `R-AEM-016`'s posture)?
3. What per-call argument cap is the product answer — a fixed byte ceiling, or a multiple of the frame cap?
4. If a vendor delivers a tool call the neutral events cannot represent (fragmented tool *name*), is
   failing the stream acceptable, or must the adapter buffer until representable?

Assumptions used in the absence of answers: `function_call` is skipped with a recorded disposition; the
fragment is reachable but bounded; the cap is a package constant; representability gaps fail typed rather
than silently.
