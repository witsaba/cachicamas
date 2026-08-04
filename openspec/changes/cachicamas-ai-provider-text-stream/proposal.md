# Proposal — translate the response lifecycle and text

> **Change**: `cachicamas-ai-provider-text-stream` · **Milestone**: AI-28 · **Nodes**: AI-28.1.1, AI-28.1.2, AI-28.2 … AI-28.6
> **Phase**: proposal · **Date**: 2026-08-04 · **Project**: cachicamas (witsaba)
> **Wave**: 4 "Connect the vendor" · base branch `feat/ai-28-0-integration-base` (AI-26 ∪ AI-27 merged)
> **Depends on**: AI-26, AI-27, AI-14 … AI-16, AI-19, AI-20.3, AI-23.2 · **Blocks**: AI-29 … AI-33
> **Cross-milestone block**: AI-28.6 depends on **AI-32.1**, which has not started (§ "The one blocked node")

---

## Intent

The adapter today is two half-arcs that never meet. `Translate` (AI-26) turns an `ai.Request` into wire
bytes; `Decoder` (AI-27) turns bytes into `Frame`s. **Nothing issues a request, nothing reads a response,
and nothing produces a single `ai.Event`.** `provider_boundary_test.go` still asserts, mechanically, that
`*Client` has no `Stream` method and does not satisfy `ai.ModelProvider` (S-APC-030, S-APC-031).

AI-28 is the walking skeleton: the first time a real HTTP transcript becomes a normalized, contract-conformant
event stream end to end, served by a local `httptest.Server`. It is also the first milestone that may
**construct** an `ai.Failure` from a decoder error — AI-27 deliberately does not (recorded ruling #4;
`errors.go` S-ASD-064), because only a producer holds the `outputPreceded` fact `ai.MidStreamFailure`
requires and the carrier `Delivery` names.

## Scope

### In scope

1. **Producer shell (AI-28.1.1)** — `(*Client).Stream(ctx, ai.Request) (<-chan ai.Event, error)`: issue,
   read, `Feed`/`Finish`, emit, close. One producer goroutine, one closing site on every exit path, every
   send selecting on cancellation (AI-20.3, the `agenttest` fake's own physics — reproduced, not
   approximated). Response identity and served model land in `ai.NewResponseStart`, whose two fields are
   both non-empty-required.
2. **Text mapping (AI-28.1.2)** — chunk → `TextBlockStart` / `TextDelta` / `TextBlockEnd` / `Completion`,
   with concatenated deltas reconstructing the text byte-exactly across a delta boundary **inside a
   multi-byte rune** (AI-16.2). Includes the **conformance bridge**: `agenttest.Factory.New` takes
   `...Script`, so AI-23.2's text case can only run "against real transport" through a Script → SSE
   transcript → `httptest.Server` → `openaicompat` adapter seam that does not exist yet.
3. **Terminal discipline (AI-28.2)** — `data: [DONE]` recognized as clean termination, never payload,
   never tripping truncation; close without the terminal → typed terminal **error** with partial output
   preserved and flagged; post-terminal, pre-EOF frames ignored.
4. **Absent-versus-zero (AI-28.3)** — usage fields never present in the transcript are **absent**, not
   zero (AI-13.3), and `usage`'s presence is **asserted**, not silently accepted as empty (AI-24 § 13.1's
   opt-in trap). The request-side half is already discharged: `appendStreamFields` (body.go) emits
   `"stream":true,"stream_options":{"include_usage":true}` on every request today.
5. **Unknown and delta-less tolerance (AI-28.4)** — unknown frame/delta/block types skipped without
   corrupting adjacent accumulation; a zero-delta block normalizes cleanly; keep-alives are inert.
6. **Protocol-order violations (AI-28.5)** — a table over structural violations (delta with no open block,
   delta after close, duplicate open, close without open, second response start) plus malformed payload of
   a *known* type → typed malformed-response terminal, partial output preserved, **never a panic**.
7. **Pre-decode response checks (AI-28.6)** — non-stream content type refused before decoding with a
   bounded body excerpt (match tolerant of parameters and case); failure statuses routed to the failure
   mapping before any decode. **Sequenced last, blocked on AI-32.1** — see below.

### Out of scope

- **Reasoning (AI-29)** and **tool calls (AI-30)** — no reasoning or `tool_calls` mapping, in either
  direction, at this milestone.
- **The full usage field mapping and cumulative merge (AI-31.2)** — AI-28.3 proves *absent ≠ zero* and
  presence only; it does not map the object.
- **The failure-status taxonomy itself (AI-32.1)** — AI-28.6 *consumes* it; it does not author it.
- **Any new module dependency.** `backend/agent/go.mod` carries **zero requires**. Stdlib only. A proposal
  needing a third-party dependency is a hard blocker, not a tradeoff.
- **Widening `ai.FailureCategory`** — the 9-member vocabulary is closed and append-only (R-AIP-005); AI-27's
  recorded compromise stands and is not reopened here.

## Capabilities

### New Capabilities

- `ai-provider-text-stream`: the OpenAI-compatible adapter's response lifecycle — producer shell and
  cancellation discipline, text-block normalization, terminal-sentinel and truncation discipline,
  absent-versus-zero usage fidelity, unknown/delta-less tolerance, protocol-order violation handling, and
  pre-decode response checks.

### Modified Capabilities

- **None.** `ai-stream-lifecycle`, `ai-text-events`, `ai-response-events`, `ai-provider-errors`,
  `ai-completion-metadata`, `ai-stream-decoder` and `ai-provider-conformance-suite` are **cited by
  identifier and satisfied, never amended**. AI-28 is the first *consumer* of all of them; if any turns out
  to need a requirement change, that is its own change, not absorbed here.

## Approach

**A producer goroutine over a read loop, not a generic pipeline.** `Stream` validates, builds the request
from `Translate`'s bytes through the existing unexported `newRequest`, does the pre-decode checks
(AI-28.6), then hands the body to one goroutine that reads into `Decoder.Feed`, maps each `Frame` to zero
or more events through a per-stream `ai.Stamper`, and calls `Decoder.Finish` at EOF. A **state machine
owns block lifecycle** — index-keyed open/closed accumulators — because AI-28.5's whole test list is that
machine refusing to crash on frames a buggy proxy can emit at will.

**Decoder errors become `ai.Failure` here, and only here.** `openaicompat.Category(err)` names the AI-19
category; this milestone adds `outputPreceded` (did any normalized output event precede?) and constructs
`ai.MidStreamFailure` / `ai.PreStreamFailure`, wrapped as the stream's terminal `ai.ErrorEvent`.

**Slicing is doc 0002's own.** AI-28.1 is pre-split into AI-28.1.1 and AI-28.1.2 **as separate PRs by the
document's design** — it is the single largest implementation step in Layer 1. That split is not a budget
accommodation this proposal invented and must not be collapsed.

## The citation gate — response-side wire shapes are uncited

`doc.go`'s four provenance claims, pinned to `openai-openapi` commit `d4fb706e6e05d4cc9f1b33ca59b6e4f3e8edd439`,
are **all request-side**: system-role placement, tool-call argument encoding, `tools[].function.parameters`,
role alternation. AI-24 § 3's source-label rule binds § 10's *framing* claims; it does not label the
response **chunk body** schema at all.

Every one of the following is therefore a claim AI-28 **must cite against the same pinned commit before
specifying behaviour as fact** — inference is not admissible for wire shapes:

| Claim needing citation | Where it is currently asserted |
| --- | --- |
| The streaming chunk's response-identity and served-model field names | AI-24 § 8 `CAP-R-01`/`CAP-R-03`, unlabelled |
| Text arrives as a per-choice `delta` content field | AI-24 § 8 `CAP-R-01` ("delta.content"), unlabelled |
| The completion signal's field name and its position relative to `usage` | AI-24 § 8 `CAP-R-03` ("finish_reason"), unlabelled |
| The in-stream `usage` object's shape and which fields it populates | AI-24 § 13.1, unlabelled |
| Whether a block-start signal exists at all, or is implicit | AI-24 § 8 calls it "an implicit block start" |

Two shapes **are** already cited and must not be re-derived: the SSE framing (WHATWG § 9.2, AI-24 § 10.1)
and the `data: [DONE]` terminal sentinel plus data-only default event type (AI-24 § 10.2,
dialect-conventional, fixture-pin obligation already discharged by AI-27).

## The one blocked node — AI-28.6

Doc 0002 states `AI-28.6 — Depends on: AI-28.1.1, AI-32.1`. **AI-32.1 has not started.** AI-28.6 stays
**in scope, sequenced last, explicitly blocked**, so the coordinator can fast-track AI-32.1 in parallel on
its own branch. It is neither silently dropped nor pretended unblocked. If AI-32.1 has not landed when
slices 1–6 are green, AI-28.6 becomes a stated, recorded carryover with its blocker named — not an
absorbed guess at a taxonomy another milestone owns.

## Guard reconciliation is expected, and is planned work

Wave 4's recorded lesson (Engram `sdd/cachicamas-ai-layer-1/wave-4-state`): change-scoped guards fire when
chains meet, and the answer is an **enumerated allowlist with citations**, never weakening or deleting the
guard. Two guards bite at AI-28.1.1:

| Guard | Why it bites | Sanctioned reconciliation |
| --- | --- | --- |
| `TestClient_HasNoStreamingEntryPoint` (S-APC-030) | `Stream` lands | Flip to asserting the method exists — its own doc comment already anticipates the flip |
| `TestClient_DoesNotSatisfyModelProviderAtRuntime` (S-APC-031) | `*Client` becomes an `ai.ModelProvider` | Flip to `ok == true`; the test was deliberately written as a runtime assertion, not a compile-time one, for exactly this |
| `TestPolicy_NoNewSentinelsExported` (S-ART-054) | Only if AI-28 exports a new sentinel (e.g. a non-stream-content-type identity) | Add the named entry with its spec citation to the allowlist — never re-freeze it empty |

`doc.go`'s own opening prose ("Streaming behaviour arrives at AI-26") is **stale**: AI-26 shipped `Translate`
only. AI-28.1.1 corrects it in place, per this package's disclose-a-correction practice.

## Affected Areas

| Area | Impact | Description |
| --- | --- | --- |
| `backend/agent/src/ai/openaicompat/*.go` | New | Producer, block state machine, chunk mapping, failure construction, pre-decode checks |
| `backend/agent/src/ai/openaicompat/*_test.go` | New | One proof file per node; `httptest.Server` transcript fixtures |
| `backend/agent/src/ai/openaicompat/provider_boundary_test.go` | Modified | S-APC-030/031 flip (planned, cited) |
| `backend/agent/src/ai/openaicompat/doc.go` | Modified | Stale AI-26 streaming prose corrected; AI-28 rulings recorded |
| `backend/agent/src/agenttest/` | Read-only | `Factory`, `RunConformance`, stream kit consumed as-is |
| `backend/agent/src/ai/` | Read-only | Event constructors, `Stamper`, `CheckStream`, AI-19 constructors cited by identifier |
| `backend/agent/go.mod` | Unchanged | Zero requires. Any new require is a hard blocker |
| `docs/architecture/milestones/0002-…md` | Living-graph | Amended by *apply* only if a node splits |

## Delivery

`delivery_strategy: auto-chain` · `chain_strategy: feature-branch-chain` · budget 5000 ·
base `feat/ai-28-0-integration-base`. Doc 0002: "the node boundary is the PR-chain boundary."

| Slice | Node | Targets | Note |
| --- | --- | --- | --- |
| 1 | AI-28.1.1 producer shell | base | Largest single step; `Split if` the goroutine/close lifecycle alone exceeds budget |
| 2 | AI-28.1.2 text mapping + conformance bridge | slice 1 | The bridge may justify its own sub-split |
| 3 | AI-28.2 terminal discipline | slice 2 | |
| 4 | AI-28.3 absent-vs-zero | slice 3 | Small |
| 5 | AI-28.4 unknown / delta-less | slice 4 | |
| 6 | AI-28.5 protocol-order violations | slice 5 | Table-driven; depends only on 28.1.1 |
| 7 | AI-28.6 pre-decode checks | slice 6 | **Blocked on AI-32.1** — omitted from the chain if unlanded |

`sdd-tasks` owns the line forecast. Repo precedent runs ×2–4 over naive estimates (AI-21 forecast 900–1400,
landed 2241), and AI-28 is doc 0002's own largest milestone: chaining is **mandatory**, and slices 1 and 2
should each be assumed at risk of an internal split.

## Risks

| Risk | Likelihood | Mitigation |
| --- | --- | --- |
| A response-side wire shape is implemented from inference rather than citation | **High** | The citation gate above is a spec precondition: `sdd-spec` must not state an uncited field name as fact. Each row cites the pinned commit or the node stops |
| The conformance bridge (`Script` → SSE → `httptest.Server`) is larger than AI-28.1.2 itself | **High** | Named as scope in slice 2, not discovered mid-apply; pre-declared as a candidate internal split |
| AI-32.1 never lands in this wave, silently dropping AI-28.6 | **High** | AI-28.6 stays in scope, last, with its blocker written into the delivery table; a carryover is recorded, never absorbed |
| Slice 1 alone exceeds the review budget | Medium | Doc 0002 already pre-declares the `Split if`: close discipline splits from the emit path |
| A `-race` flake in the producer/cancellation path | Medium | Physics reproduced from `agenttest/fake_provider.go`'s proven shape, not re-invented; `-race -count=1` on every slice |
| A new exported sentinel silently breaks S-ART-054 at chain-merge time | Medium | Anticipated above; allowlist entry with citation is the sanctioned path |
| AI-28.5's state machine invents behaviour AI-30 will contradict | Medium | Text-only block indices at this milestone; tool-call accumulation is explicitly AI-30's |

## Rollback Plan

Every change is additive inside `openaicompat/`, except the two anticipated guard flips and the `doc.go`
prose correction — all three confined to that package. `src/ai`, `agenttest` and Waves 1–3 compile and pass
identically with or without this change; `go.mod` is untouched. `git revert` per slice commit is a clean
boundary; reverting the milestone means reverting slice 1's branch point, at which the two guards return to
their AI-25 assertions and `*Client` stops advertising `ai.ModelProvider`. AI-29 … AI-33 have not started,
so no consumer breaks.

## Dependencies

- AI-26 (`Translate`, request body) and AI-27 (`Decoder`, `Frame`, `Category`) merged into the base branch — **satisfied**.
- AI-19 constructors, AI-14 … AI-16 event types, AI-20.3 cancellation contract, AI-23.2 conformance case — all shipped.
- **AI-32.1 — not started.** Blocks AI-28.6 only.
- Stdlib only; no ADR triggered, and keeping it that way is a constraint, not an outcome.
- Strict TDD; `make test` (`go test -race -count=1 ./...`) and `make lint` clean from `backend/agent/`.

## Success Criteria

- [ ] A recorded text transcript replayed through a local test server drains as a fully normalized,
      contract-conformant stream: response start → text block → completion, sequenced from 1, closed once.
- [ ] The vendor's response identity and served model land in the start event's normalized fields.
- [ ] Concatenated deltas reconstruct the text byte-exactly, including across a split multi-byte rune.
- [ ] AI-23.2's text conformance case passes against real transport, through the new bridge.
- [ ] `data: [DONE]` terminates cleanly; a connection closing without it yields a typed terminal error with
      partial output preserved and flagged; post-terminal frames are ignored.
- [ ] Absent usage fields read as absent, not zero; `usage`'s presence is asserted, not assumed.
- [ ] Unknown frame/delta/block types and keep-alives perturb nothing; a zero-delta block is clean.
- [ ] Every protocol-order violation in the table yields a typed malformed-response terminal, never a panic.
- [ ] Cancellation honors AI-20.3: every send selects, and close discipline is the contract's.
- [ ] Every response-side wire-shape claim in `spec.md` carries a citation to the pinned openai-openapi
      commit, or its node is stopped rather than guessed.
- [ ] AI-28.6 is either landed (AI-32.1 available) or recorded as a carryover naming AI-32.1 as its blocker.
- [ ] `make test` green, `make lint` clean, AI-00 import guards pass, `go.mod` unchanged.
