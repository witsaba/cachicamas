# Proposal — the scripted fake provider

> **Change**: `cachicamas-ai-fake-provider` · **Milestone**: AI-21 · **Nodes**: AI-21.1 … AI-21.8
> **Phase**: proposal · **Date**: 2026-08-02 · **Project**: cachicamas (witsaba)
> **Wave**: 3 "Prove" — first milestone; ships in the wave's single PR with AI-22 and AI-23
> **Depends on**: AI-20 (shipped) · **Blocks**: AI-22, AI-23, doc 0003 wave C
> **Input**: Engram `sdd/cachicamas-ai-fake-provider/explore`

---

## Intent

Layer 1 has a provider interface and no provider. Every test above it — the conformance suite,
the stream test kit, all of Layer 2's agent loop — needs a producer it can script, and today the
only one is `scriptProvider`, unexported inside `ai/provider_test.go` and reachable by nobody.
AI-21 promotes that shape into an importable package so no downstream test invents its own.

The risk this change manages is fidelity, not effort. Layer 2 will build on what the fake does,
not on what the document says: a fake that closes cleanly where the real contract drops events
teaches the wrong physics permanently.

## Scope

### In scope

1. **A `package agenttest` fake implementing `ai.ModelProvider`**, scriptable per call, with the
   exact AI-20 physics reproduced — not approximated: pre-stream validation before the
   already-cancelled case, one producer goroutine, one `defer close`, every send selecting on
   `ctx.Done()`, and the sanctioned saturated-buffer drop with a bare close.
2. **Seven script shapes** on top of that skeleton: text (21.1), tool call including the
   zero-delta start→end (21.2), terminal error across both partial-output states (21.3),
   held-open and saturated streams (21.4), pre- and mid-stream cancellation (21.5), request
   capture (21.6), reasoning including redacted and signature-only shapes that never leak into
   text events (21.7).
3. **Two new primitives with no precedent**: a test-controlled synchronization point for held-open
   streams (no wall-clock sleeps in scripted-schedule assertions), and a per-call script queue
   whose exhaustion fails loudly — never hangs, never repeats the last script (21.8).
4. Extending `agenttest/doc.go`, which today frames the package as proof-tests only.

### Out of scope

- Vendor wire-format mocking — AI-27's fixtures and AI-38's transcripts.
- The stream assertion helpers (AI-22) and the conformance suite (AI-23): siblings in this wave,
  separate changes.
- Deleting or replacing `ai/provider_test.go`'s `scriptProvider`; it stays local per its own
  header comment.
- **Deferred but related — two Wave 2 carryovers, deliberately not absorbed here.** Both are
  recorded in the promoted specs as "owned by Wave 3" *generically*; doc 0002 assigns neither to
  AI-21, AI-22 or AI-23, and neither appears in AI-21's charter or its eight leaves:
  - **W1** — `CheckEmit` rule 4 has no failure-path test (`specs/ai-event-envelope/spec.md`).
  - **W2** — `*Failure` has no redacting `GoString()`, unlike every sibling payload type
    (`specs/ai-provider-errors/spec.md`, R-AIP-009).

  They remain **unassigned**, not forgotten. Assigning them is a question for whoever scopes the
  next wave — do not mint a milestone number here.

## Capabilities

### New Capabilities

- `ai-fake-provider`: the scripted fake — its script vocabulary, its per-call queue and
  exhaustion behavior, its synchronization point, its request capture, and its obligation to
  reproduce the `ai-model-provider` / `ai-stream-lifecycle` physics exactly.

### Modified Capabilities

- None. `ai-model-provider`, `ai-stream-lifecycle`, `ai-event-envelope`, `ai-text-events`,
  `ai-tool-call-events`, `ai-reasoning-events`, `ai-provider-errors` and `ai-model-request` are
  **read and cited by identifier, never modified**.

## Approach

Extend `src/agenttest/` in place with the package's first non-`_test` files (exploration
approach 1; a sub-package is rejected because doc 0002 places the deliverable directly there and
the AST guard's `runtime.Caller` resolution assumes the direct-sibling layout). The core
`Stream()` copies `scriptProvider`'s concurrency shape verbatim in behavior — already stressed
under `-race` — and a fresh `Stamper` per call gives per-stream sequencing for free. Files split
one-shape-per-file, mirroring `src/ai`'s convention, each with a milestone-ID "why before what"
header.

Two items are real design work for `sdd-design`, not decided here: how exhaustion "fails loudly"
given the fixed `(<-chan Event, error)` signature (typed pre-stream error vs. a captured
`testing.TB` handle — the latter is a new pattern for this module), and the exact shape of the
release/synchronization primitive.

## Affected Areas

| Area | Impact | Description |
| --- | --- | --- |
| `backend/agent/src/agenttest/*.go` (`package agenttest`) | New | The fake, its script builder, queue and synchronization point |
| `backend/agent/src/agenttest/*_test.go` (`package agenttest_test`) | New | One proof file per AI-21 leaf |
| `backend/agent/src/agenttest/doc.go` | Modified | Reframe: proof package **and** importable test library |
| `backend/agent/src/ai/**` | Read-only | Contract source the fake must reproduce; unchanged |

## Risks

| Risk | Likelihood | Mitigation |
| --- | --- | --- |
| Convenience drift — the fake closes cleanly where the contract drops events | Medium | Charter's non-negotiable; spec states the drop path as a positive requirement, and cancellation tests assert the bare close under `-race` |
| The two unprecedented primitives (21.4 release, 21.8 exhaustion) get designed inside apply | Medium | Both explicitly deferred to `sdd-design` above; apply must not improvise them |
| Flaky suite from wall-clock coordination creeping in | Medium | Review-enforced authoring rule; `settleAfterCancel`'s test-side sleep must not be mistaken for the mechanism |
| Scope creep via W1/W2 | Low | Named and parked above; absorbing either is a signal to stop |
| Review budget over 400 lines | Certain | Wave accepted up front as `single-pr` at 5000 lines; leaf boundary is the commit boundary |

## Rollback Plan

Every change is additive: new files in `src/agenttest/` plus one doc comment. No shipped
signature moves and no existing test changes, so `src/ai` compiles and passes identically with
or without this change. `git revert` per leaf commit is a clean boundary. Because AI-22 and
AI-23 land in the same PR and both consume this fake, a full revert of AI-21 alone requires
reverting them too — otherwise revert the whole wave branch, which returns the tree to
main @ 37898c7 exactly.

## Dependencies

- AI-20 merged (shipped). Layer 1 stays stdlib-only — this package needs the stdlib and the
  module's own `ai` package, so `go.mod` gains nothing and no ADR is triggered.
- Strict TDD; `make test` from `backend/agent/` (`go test -race -v ./...`), `make lint` clean.

## Success Criteria

- [ ] A test in another package scripts start + two text deltas + complete and drains exactly
      those events, sequenced 1…N, terminated by close, with no network.
- [ ] Two fakes streaming concurrently sequence independently under `-race`.
- [ ] Tool calls reconstruct to exact argument bytes; a zero-delta call is indistinguishable
      from a delta-carrying one after reconstruction; interleaved calls keep their ordinals.
- [ ] A terminal error of any AI-19 category is scriptable in both partial-output states, and
      nothing follows it.
- [ ] A stream can be held open and released, and can saturate an unread consumer, both without
      a wall-clock sleep in any assertion.
- [ ] Mid-stream cancellation closes in bounded time with no send after close and no forced
      terminal; pre-call cancellation takes the pre-stream path with a typed error.
- [ ] Every field a request carried is assertable after the call, and later caller mutation
      cannot alter recorded history.
- [ ] Reasoning deltas, a byte-exact round-trip token, and redacted / signature-only shapes are
      scriptable, and none appear in text events.
- [ ] Consecutive calls consume consecutive scripts; an exhausted queue fails loudly, never
      hangs, never repeats.
- [ ] `make test` green and `make lint` clean; both AI-00 import guards and the AI-20 signature
      guard still pass.
