# Explore — Layer 1 lock backpressure and buffer behavior

> **Change**: `cachicamas-ai-backpressure`
> **Milestone**: AI-34 — Lock backpressure and buffer behavior (Layer 1 / doc 0002)
> **Phase**: explore
> **Project**: cachicamas (witsaba)
> **Date**: 2026-08-07
> **Target module**: `backend/agent/` (Layer 1, layered not hexagonal — see [ADR 0005 § D1](../../../../docs/adr/0005-promote-agent-stack-to-own-module.md))
> **Authoring constraint**: doc 0002's authoring constraint binds every artifact of this change. **No Go identifier is invented here.** All names cited are already present in the merged source at this HEAD (`a78a941`, base `main`).

---

## 1. Identity

AI-34 is the milestone that locks Layer 1's backpressure behaviour: the carrier's buffer must be sized with measurements, ordered losslessly under pressure, and proved to have no unsanctioned loss path beyond the AI-20.3 / AI-33.3 saturated-drop exception. There are three subnodes: a decision (AI-34.1, `cap` and constant-vs-configurable) and two leaves that prove it (AI-34.2 capacity + ordering under pressure; AI-34.3 exhaustive non-cancellation paths). All work is over the **real** `*openaicompat.Client.Stream` carrier — the conformance suite already proves the abstract physics over the fake provider, and AI-33.3 already proves the abandoned-then-cancelled path against the real producer. AI-34 adds what neither has: **capacity as an observable property of the real producer, and no auxiliary queue.**

The milestone depends on AI-33 (cancellation contract, just landed 2026-08-07 PR #129) and AI-22.4 (leak-check mechanism, already shipped). It blocks AI-38 (first-adapter conformance roll-up). It does **not** reopen the AI-02.1 carrier decision: channels stay, the iterator option is closed, and the producer is the single goroutine that owns the carrier's lifetime (`stream.go:344`).

This exploration does not write production code or tests. It establishes the seams, names the drift between doc and code, surfaces the architecture decisions, and proposes the measurement workload.

---

## 2. Current state — what the merged source already does

Every named function, file and line below is already in `main` at this worktree (base `main` @ `a78a941`, the merge commit of PR #129).

### 2.1 The carrier is constructed as an **unbuffered** rendezvous — drift to flag at AI-34.1

- `backend/agent/src/ai/openaicompat/stream.go:223` — `out := make(chan ai.Event)` (UNBUFFERED). The variable `out` is returned at line 225 to the caller, and handed to `run` at line 224 as the producer's send target.
- `backend/agent/src/ai/openaicompat/a_i-33_3_test.go:28` — the AI-33.3 test file's own header comment cites "out is unbuffered (stream.go:209)" — that line number is the location at the time AI-33.3 was authored; the carrier construction has since drifted to line 223 (one comment block shift). Drift, not a second carrier. This must be noted at propose-time so a future contributor doesn't read the test comment as a primary source of truth.
- **The doc 0002 target is 64**, not 0. `docs/architecture/milestones/0002-cachicamas-ai-layer-1-task-graph.md:387` and `openspec/specs/ai-stream-lifecycle/spec.md:387` both specify "**bounded**, with a **starting capacity of 64 events**" as the AI-02.1 decision. The implementation has been unbuffered from AI-28.1.1 (slice 1) through AI-33 (PR #129, merged 2026-08-07). **No commit in `git log -- backend/agent/src/ai/openaicompat/stream.go` introduces a buffer argument or value other than 0.** The 64 number has been the contract's hypothesis and the spec's stated starting point; the implementation never carried one. This is a documented-but-actual divergence the AI-34.1 decision must reconcile on the record (see § 5.1).
- **The 64 number is not in `git log` as a code change either.** A search for `64` against `backend/agent/src/ai/openaicompat/stream.go` returns zero matches — the only buffer-shaped number on the carrier is the per-call I/O buffer `streamReadBufferSize = 32 * 1024` (`stream.go:123`), which bounds one `Body.Read` call, not the event carrier. The doc-0002 § 6's three measurements (M1 high-water, M2 producer-wait-frequency, M3 resident-memory) all run against a number that does not exist in code today; AI-34.1's deliverable is to run them anyway — "with measurements, not a preference" — and produce the first actual number the spec can cite.

### 2.2 The producer's send site selects on cancellation — the backpressure primitive

- `backend/agent/src/ai/openaicompat/stream.go:574–582` — `emit(ctx context.Context, out chan<- ai.Event, stamper *ai.Stamper, ev ai.Event) bool` is the single helper every event in this package goes through (`stream.go:570–573` doc comment). Its body is the canonical select:
  ```go
  select {
  case out <- stamped:
      return true
  case <-ctx.Done():
      return false
  }
  ```
- **Backpressure is structural.** Because `out` is unbuffered, this is a pure rendezvous: the producer blocks on `out <- stamped` until either a receiver receives or ctx is done. Cancellation wins the race immediately (R-ATS-005, AI-20.3 § 5). When `out` becomes buffered, this same select gives the producer a finite parking space and the consumer's read rate governs drain — the backpressure primitive is unchanged; the buffer is its parking surface.
- Every call site in `run` (`stream.go:407`, `:410`, `:423`, `:431`, `:439`, `:451`, `:455`, `:504`, `:511`, `:517`, `:544`) routes through `emit`. The mid-emit cancellation branch at `:454–487` (mid-emit, mid-events-loop) and the mid-read branch at `:520–543` (after `resp.Body.Read`) are the only two paths that drop a send and return early. Both are pinned to `ctx.Err() != nil` as the return condition, so the saturated-buffer shape only manifests at the next `emit` call when the producer has more to send than a paused consumer drains.

### 2.3 The AI-32.3 bounded-wait terminal — the seam AI-34 inherits, not rewrites

- `backend/agent/src/ai/openaicompat/stream.go:129` — `const emitFailureSendBound = 5 * time.Second`. This is the bound a terminal failure waits before giving up on a non-responsive consumer. It is **not** the carrier's buffer; it is the terminal send's fallthrough timer.
- `backend/agent/src/ai/openaicompat/stream.go:469–482` (mid-emit, after `emit` loses the ctx race) and `:530–540` (mid-read, after `Body.Read` fails and ctx is done) — both `select { case out <- endStamped: case <-time.After(emitFailureSendBound): }`. AI-32.3's documented S-AEM-051/052 mechanism (`openspec/changes/archive/2026-08-05-cachicamas-ai-provider-error-mapping/specs/ai-provider-error-mapping/spec.md:259–260`); unchanged by AI-34.
- `backend/agent/src/ai/openaicompat/stream.go:679–682` (`emitFailure`'s terminal-error send) — same shape. **Every terminal send selects on `time.After(emitFailureSendBound)`, never on `ctx.Done()`.** AI-33.3's "bare close" property depends on this: a truly-abandoned consumer causes the bounded-wait send to expire after 5 s, and the terminal event is dropped rather than forced through.
- **For AI-34 this means: the producer's terminal path already has a bounded-wait. There is no place where the producer would block indefinitely waiting for a consumer, even at the saturated terminal.** Buffer sizing for the producer's normal-path sends is the only thing AI-34 has to think about — the failure path is already bounded, and AI-33.3's bare-close property is therefore unconditional once `emitFailureSendBound` elapses, regardless of whether the carrier is buffered or not.

### 2.4 The AI-22.4 leak-check mechanism — AI-34.3's mechanics

- `backend/agent/src/agenttest/stream_kit_leak.go:107` — `RequireNoGoroutineLeak(tb, scenario)`: 50 repeats with a `runtime.NumGoroutine` before/after amplitude check; serial-only via `tb.Setenv("AGENTTEST_STREAM_KIT_LEAK_CHECK", "1")` (R-STK-008 mechanical guard). Already used by AI-33.5b (`a_i-33_5b_test.go`); the seam AI-34.3's exhaustive-path test rides on.
- `backend/agent/src/agenttest/stream_kit_record.go:63` — `DrainAndRecord(tb, ch, timeout)` with `DefaultDrainTimeout = 2 * time.Second` (line 24). Already used by every AI-33 test (`a_i-33_*.go` × 6 files) and the conformance cases.

### 2.5 The conformance suite's carrier coverage today — a gap, not a duplicate

- `backend/agent/src/agenttest/conformance_cancellation.go:46–85` — `cancellationBoundedCloseCase` (R-CNF-011): `Script{Buffer: 0, …}` over the **fake** provider; first event `ResponseStart`, then cancel, then `rec.Len() == 0`. Asserts no further events are received, but **not** capacity.
- `backend/agent/src/agenttest/conformance_cancellation.go:94–131` — `cancellationAbandonedThenCancelledCase` (R-CNF-012, pinned verbatim): 10-step unbuffered script over the **fake**; cancel immediately, drain. Asserts no terminal of any kind was invented.
- `backend/agent/src/agenttest/conformance_cancellation.go:91–93` — the comment is precise: *"An unbuffered channel with zero reads is saturated from the first scripted event onward by construction, so no wall-clock wait is needed to reach that state deterministically."* The suite proves the **saturated-drop physics**; it does **not** prove a particular capacity. **R-CNF-011/012 do not assert `cap(ch)`.** AI-34.2's "capacity equals decided size" is therefore a **new** assertion the conformance suite never made and the real producer never had to honour.
- `backend/agent/src/agenttest/fake_gate_test.go:80–143` — `TestProvider_UnreadConsumer_DeterministicallySaturatesBuffer_ThenResumedConsumerDropsNothing` (R-AFP-011): the only `cap(ch)` assertion in the entire codebase (line 108: `if got := cap(ch); got != 2`). It runs over the fake provider with `Buffer: 2` and asserts a slow-then-resumed consumer drains every scripted event losslessly. **This is the only existing pattern for "slow / pause / resume consumer loses nothing"**, and it is on the fake, not the real producer. AI-34.2 needs to author the same shape on the real `*openaicompat.Client` over a real HTTP transport.

### 2.6 The OpenRouter wrapper is a one-line forwarder — no buffer knob to land on

- `backend/agent/src/ai/openaicompat/openrouter/wrapper.go:84–86` — `func (p redactedProvider) Stream(ctx context.Context, req ai.Request) (<-chan ai.Event, error) { return p.Client.Stream(ctx, req) }`. The wrapper embeds `*openaicompat.Client` (line 79) and forwards verbatim. **No buffer field, no config surface for sizing.** If AI-34.1 decides "configurable", the configuration surface belongs on `openaicompat.Config` (`backend/agent/src/ai/openaicompat/client.go:43–58`), not on the wrapper; the wrapper is injection-only by design (R-OR-01, `wrapper.go:111`). The OpenRouter wrapper is therefore not in scope for AI-34's diff footprint.

### 2.7 The wire-frame flow into the carrier

- `backend/agent/src/ai/openaicompat/stream.go:185–226` — `Stream(ctx, req)`: validate → Translate → ctx.Err → `httpClient.Do` → `mapResponse` (status) → `isStreamContentType` → `out := make(chan ai.Event)` → `go run(ctx, resp, out)` → `return out, nil`. **The channel is allocated at line 223, between the last pre-handover check and the goroutine spawn.** A buffered variant replaces this one line; everything above and below is unaffected.
- `backend/agent/src/ai/openaicompat/stream.go:357–548` — `run` reads the body in 32 KiB chunks (`streamReadBufferSize = 32 * 1024`, line 123), feeds the decoder, applies the mapper, and emits each resulting event through `emit`. **One event per `emit` call**, so the carrier's buffer (when introduced) absorbs one emitted event per slot — not a chunk, not a body read. The wire frame and the carrier event are different layers.
- `backend/agent/src/ai/openaicompat/bridge_test.go:64–92` — `conformanceBridgeFactory`: the real `*openaicompat.Client` driven against a real `httptest.Server`. This is the proven shape for "real producer + real HTTP transport" tests (AI-33.1–33.5 all reuse `mustClient` + `serveTranscripts` from `stream_test.go:50–60` and `bridge_test.go:98–115`). AI-34.2 / 34.3 reuse the same seam.

### 2.8 The AI-2.1 carrier decision's other three grounds — closed, not reopened here

Per Engram obs **#2267** and `openspec/specs/ai-stream-lifecycle/spec.md:381–438`:

1. **Layer 2 wait semantics** — the agent loop suspends calls for permission (doc 0003 AG-10.3 line 527–532). When a permission gate is open, the consumer is blocked; the producer must wait, not buffer unboundedly. The carrier = channel mechanism lets the producer wait on a `select`-bound send while ctx remains cancellable.
2. **Hidden-buffer objection** — an iterator with an internal queue would be a hidden buffer the spec cannot bound (spec § "Why channels over iterators" lines 110–122). Channels force the buffer onto the contract; `cap(ch)` is the assertion.
3. **Exactly-one-terminal** — `defer close(out)` at `stream.go:358` is the single closing site. With a buffered channel, the same defer holds; the channel still closes exactly once on every exit path. R-CNF-009's invariant survives the buffer change.
4. **Iteration-shape connotation** — `range ch` reads as "iterating over what the producer produced"; `Iter` (`backend/agent/src/agenttest/stream_kit_iter.go:16`) is the carrier-view abstraction the AI-22.5 deliverables already ship. Buffering the channel does not change `Iter`'s semantics.

None of these grounds is affected by AI-34. The carrier = channel decision is closed.

---

## 3. Test seams available

All helpers below are already shipped; AI-34 consumes them.

| Seam | Symbol | Location | Notes |
| --- | --- | --- | --- |
| Capacity assertion (FAKE only today) | `cap(ch)` direct check | `backend/agent/src/agenttest/fake_gate_test.go:108` | `if got := cap(ch); got != 2` — pattern AI-34.2 lifts to the real producer. |
| Leak detection | `RequireNoGoroutineLeak(tb, scenario)` | `backend/agent/src/agenttest/stream_kit_leak.go:107` | 50 repeats, serial-only via `tb.Setenv` (R-STK-008); already used by AI-33.5b and the conformance cases. |
| Bounded drain | `DrainAndRecord(tb, ch, timeout)` | `backend/agent/src/agenttest/stream_kit_record.go:63` | `DefaultDrainTimeout = 2 * time.Second`. Fails `tb` on deadline; never partial-returns. |
| Lined-up kind check | `requireDrainedKinds(tb, events, want)` | `backend/agent/src/agenttest/conformance_lifecycle.go:70` | AI-34.3's "drained every non-cancelled scenario losslessly" rides on this. |
| Lifecycle prefix | `checkLifecyclePrefix(events, wantID, wantModel)` | `backend/agent/src/agenttest/conformance_lifecycle.go:49` | Reusable across text and tool-call. |
| Stalling handler (frame-spacing) | `newDripHandler(chunks, interval)` | `backend/agent/src/ai/openaicompat/timeout_test.go:20–39` | The closest existing seam for "idle between frames". AI-34.2 may want a "drip frames slowly so the consumer can pause between bursts" variant; see § 6 workload. |
| Verbatim transcript server | `serveTranscripts(transcripts)` | `backend/agent/src/ai/openaicompat/bridge_test.go:98–115` | Multi-call replay; works for AI-34 over both stream kinds. |
| External-test pattern (real producer over real HTTP) | `package openaicompat_test`, `httptest.NewServer` + `mustClient` | `backend/agent/src/ai/openaicompat/a_i-33_2_test.go`, `a_i-33_3_test.go`, etc. | Proven shape for AI-34.2 / 34.3. |
| Fake `Hold`/`Gate` saturation recipe | `Buffer: n + Emit×n + Hold(gate) + Emit(late)` | `backend/agent/src/agenttest/fake_gate_test.go:80–143` | The fake-side recipe; AI-34.2 needs the **real-producer** equivalent, which doesn't exist yet. See § 5.4. |
| Valid request fixtures | `validRequest(t)`, `validToolCallRequest(t)` | `backend/agent/src/ai/openaicompat/a_i-33_1_test.go:200`, `a_i-33_3_test.go` | Reused for both stream kinds per charter line 1989. |
| Serial-only enforcement | `tb.Setenv("AGENTTEST_STREAM_KIT_LEAK_CHECK", "1")` | `backend/agent/src/agenttest/stream_kit_leak.go:82, 110` | Mechanical guard; AI-34.3's exhaustive-path test must run serially. |
| Doc-comment probe (negative assertion) | regex over test file names | `backend/agent/src/ai/openaicompat/a_i-33_3_test.go:244–278` | The pattern AI-33.3 used for the abandoned-never-cancelled narrowing. AI-34 may use it to assert "no buffering variant hides a `drop` path" if that arises. |

**No new helper is required for AI-34.** The slow-consumer / pause-resume pattern for the **real** producer is the one gap; it requires a `dripServer` analogous to `newDripHandler` but writing wire frames instead of empty bodies — see § 5.4 for the proposed shape.

---

## 4. Proposed AI-34.1 measurement workload

The doc 0002 § 6 ("What would change it", lines 422–432) names three measurements AI-34.1 must run. The explore phase proposes the **exact** workload, frame rate, payload size, consumer delay profile, and runs-to-stable-number for each.

### 4.1 M1 — Buffer occupancy high-water mark

| Field | Value | Why |
| --- | --- | --- |
| **Workload** | A text-stream transcript with one `ResponseStart` + 1 content block (start + 50 `TextDelta` events + end) + `Completion` + `[DONE]`, served at "provider's natural cadence" (~ 1 chunk per 10 ms). Plus a tool-call variant (1 `ToolCallStart` + 20 `ToolCallDelta` + 1 `ToolCallEnd` + `Completion` + `[DONE]`). Both via `serveTranscripts` over a real `httptest.Server`. | Doc 0002 § "Why 64" line 409: "A run of streamed text arrives as tens of deltas, bracketed by a start and an end event, with metadata interleaved." 50 deltas matches "tens with headroom"; 20 tool deltas matches the byte-fragmented shape R-ATC-010 cites. |
| **Consumer profile** | "Bursty real-world" — read with a fixed inter-read interval of 5 ms (a plausible UI-token-arrival-to-render gap), single goroutine. | 5 ms is the order of one UI paint frame at 60 Hz; the consumer simulates the upstream agent-loop's per-event work emitting to the WebSocket / SSE frontend (`backend/agent/src/cmd/cachicamas/...` to be implemented by AI-39 / AI-40). |
| **Measurement** | At each `emit`, record `len(out)` immediately before the send. Report `max(len(out))` across the run. | `len(out)` is observable; the max is the high-water mark for that workload. |
| **Runs to stable number** | 20 runs per stream kind per consumer profile. The high-water mark stabilises within ~5 runs in the author's empirical experience; 20 buys a 4× margin against run-to-run noise on GHA. | Recorded as `n_runs` in the decision.md the AI-34.1 deliverable authors. |
| **Direction rule** (doc 0002 line 428) | Never approaches the capacity → shrink toward high-water mark + headroom. Regularly saturates → the burst assumption is wrong; investigate before growing. | Applied to whatever number AI-34.1 measures. |

### 4.2 M2 — Whether the producer ever waits, and for how long

| Field | Value | Why |
| --- | --- | --- |
| **Workload** | The same transcripts as M1. | Identical inputs isolate the wait-time measurement from the high-water one. |
| **Consumer profile** | Three profiles, in this order: (a) **bursty** — 5 ms inter-read, as M1; (b) **slow** — 50 ms inter-read, simulating a heavier per-event cost (history building, JSON marshal); (c) **pause-resume** — read nothing for 200 ms, then drain to close. | Covers the three shapes AI-34.3 names explicitly ("slow consumer, bursty consumer, pause-resume consumer", doc 0002 line 2071). |
| **Measurement** | At each `emit`, record whether the send returned within a 1 µs budget ("fast") or waited longer ("waited"). Report `count(waited) > budget`, and the p50/p95/p99 of wait durations. A wait occurs whenever `out` was full or unread at the moment of send. | "Waited > 1 µs" distinguishes a true rendezvous hit (a one-instruction block) from a slow consumer (microseconds-to-milliseconds block). The p99 is the load-bearing number — if it's < 1 ms across all three profiles, the producer is effectively never blocked and the buffer is too large. |
| **Runs to stable number** | 20 runs per (stream kind × profile). Same margin rationale as M1. | |
| **Direction rule** (doc 0002 line 429) | Never waits → buffer is too high to be measurable; shrink. Waits often, with cause = consumer latency, not network → evidence to grow (back to M1). | |

### 4.3 M3 — Resident memory per live stream at realistic concurrency

| Field | Value | Why |
| --- | --- | --- |
| **Workload** | A short text transcript (10 events, the minimum that exercises `ResponseStart` + 1 block + `Completion` + `[DONE]`). Run N=1, 4, 16, 64 concurrent streams via goroutines, each with its own `httptest.Server` (or one server with a request counter). | N=4 = the realistic single-turn concurrency (`doc 0003 § G5`: parallel tools, typically a handful). N=16 = a multi-turn run with subagent fan-out (`G7`). N=64 = a pathological upper bound, included so the curve is visible. |
| **Consumer profile** | "Normal" — 5 ms inter-read, as M1 / M2 bursty. | |
| **Measurement** | `runtime.MemStats.HeapAlloc` delta between before / after the N-stream run, divided by N. Expressed in bytes per live stream. | Approximation of resident memory; not as precise as `-memprofile`, but sufficient to detect "material at realistic concurrency" (doc 0002 line 430). For the proposed decision artefact, this number is the **only** one that justifies a shrink. |
| **Runs to stable number** | 5 runs per N (memory allocation has higher variance than occupancy); report median. | |
| **Direction rule** (doc 0002 line 430) | Material at realistic concurrency → shrink. | |

### 4.4 Tie-break rule (doc 0002 line 432, verbatim)

> When two capacities are indistinguishable on the measurements, prefer the smaller. Backpressure that can be observed is worth more than latency that was hidden, and the drop window is smaller.

This is the rule that lets AI-34.1 finish even if M1/M2 land at e.g. 17 and M3 is negligible. The decision artefact documents the workload, the numbers, the chosen capacity, and the rule that chose it.

### 4.5 Workload output — what the AI-34.1 decision artefact will lift verbatim

The decision artefact (`openspec/changes/2026-08-07-cachicamas-ai-backpressure/decision.md`, sibling of `proposal.md`) lifts these four sub-tables as its evidence section. The shape:

```markdown
## Measurements (M1 / M2 / M3)

### M1 — Buffer occupancy high-water mark
- workload: <transcript hash, stream kind, deltas-per-block>
- consumer: <5ms / 50ms / pause-200ms>
- runs: 20 per (kind, profile)
- high-water (median across runs): <N>
- high-water (p99 across runs): <M>

### M2 — Producer wait frequency
- workload: as M1
- consumer: as M1
- runs: 20 per (kind, profile)
- waited > 1µs: <count> / <total emits>
- p99 wait duration: <ms>

### M3 — Resident memory per live stream
- workload: 10-event minimal transcript
- concurrency: N ∈ {1, 4, 16, 64}
- HeapAlloc delta / N (median over 5 runs): <bytes>

## Decision
- chosen capacity: <N>
- constant vs configurable: <constant | configurable via openaicompat.Config>
- tie-break applied: yes/no
```

**This is the only thing the AI-34.1 decision artefact owns.** Everything else in the milestone (34.2 capacity assertion, 34.3 exhaustive paths) is a leaf against the chosen number.

---

## 5. Risks and constraints

### 5.1 The "buffer 64" vs "unbuffered" reconciliation must be on the record

Doc 0002 line 387 and `openspec/specs/ai-stream-lifecycle/spec.md:387` both name "starting capacity of 64 events" as the AI-02.1 decision's quantitative output. The implementation has been `make(chan ai.Event)` since AI-28.1.1 (slice 1), per `stream.go:223`. No commit on `git log -- backend/agent/src/ai/openaicompat/stream.go` carries `64`, `buffer(`, or any third argument to `make(chan ai.Event, …)`.

**Recommendation**: AI-34.1's decision artefact must explicitly note the divergence — "the spec's starting capacity was 64; the implementation has been unbuffered; this artefact records the first measured capacity against a real workload, named as either a confirmation of 64 or a measured change." Whether the chosen number is 64, 32, 0, or any other value is the decision; recording the divergence is non-optional. The same living-graph clause that doc 0002's preamble already invokes covers this — it is exactly the kind of "the doc said X, the code says Y, AI-34 picks one" reconciliation the precedent (AI-33.3 wording) handles.

**Risk**: without this note on the record, a future contributor reads spec.md:387 as "the carrier is currently 64-buffered" and is surprised to find `cap(ch) == 0` for any real-producer test. Recording the choice in the decision artefact closes that surprise.

### 5.2 The fake provider's `Script.Buffer` is independent of the real producer's capacity

- `backend/agent/src/agenttest/fake_script.go:51` — `Buffer int` field on `Script`, used at `fake_provider.go:114` as `make(chan ai.Event, script.Buffer)`.
- The fake's buffer is fully configurable; the real producer's buffer (when introduced) is a constant or a config field. **A test that asserts `cap(ch) == 4` against the real producer does not, today, assert the same against the fake unless the fake's script also uses `Buffer: 4`.** Conformance cases that run against both factories (the standard pattern) currently use `Buffer: 0` everywhere; AI-34.1's decision does not change that — but a future contributor adding a `Buffer: N` assertion to a conformance case must remember that the fake and the real producer do not necessarily agree on N.

**Recommendation**: AI-34.2's `cap(ch)` assertion belongs on the **real producer** (`package openaicompat`), not in the conformance suite, precisely to avoid this confusion. The conformance suite remains unchanged; AI-34.2 adds a real-producer test file alongside `a_i-33_*.go`.

### 5.3 The R-CNF-011 / R-CNF-012 conformance cases will not break under a buffered carrier

The conformance suite's two cancellation cases (`conformance_cancellation.go:46–131`) use `Buffer: 0` with the fake. They assert "no further event after cancel" and "no terminal invented" — both hold for any buffer size, because they describe behaviour of the producer after cancel, not capacity. A buffered real producer satisfies both unchanged. **No conformance change is required by AI-34.** AI-34.2's `cap(ch) == N` is a real-producer assertion, not a conformance amendment.

### 5.4 The slow-consumer pattern for the real producer does not exist yet — must be authored

`fake_gate_test.go:80–143` proves the fake-side pattern: `Buffer: n, Emit×n, Hold(gate), Emit(late)` then a slow consumer's drain. The real producer has no equivalent helper; today the closest is `slowSSEServer` (`backend/agent/src/ai/openaicompat/stream_test.go:106–122`), which writes one block then stalls — but that stalls the **server**, not the consumer, and is purpose-built for the AI-33.2 between-frames-cancellation proof.

**Recommendation**: AI-34.2 introduces a `dripFramesServer(transcript, frameInterval)` analogous to `slowSSEServer`, writing frames from the transcript one at a time with a configurable gap between writes. The pattern:

```go
// proposed — author during sdd-apply, not here
func dripFramesServer(t *testing.T, frames []string, gap time.Duration) *httptest.Server {
    return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        w.Header().Set("Content-Type", "text/event-stream")
        w.WriteHeader(http.StatusOK)
        for _, frame := range frames {
            _, _ = io.WriteString(w, frame)
            if f, ok := w.(http.Flusher); ok { f.Flush() }
            select {
            case <-time.After(gap):
            case <-r.Context().Done():
                return
            }
        }
    }))
}
```

This is the **only** new helper AI-34 introduces. ~15–20 lines, test-only, in the same package. It is the shape that lets AI-34.2 prove "producer blocks on a full buffer with the consumer reading slowly" against the real producer.

### 5.5 Existing tests asserting "unbuffered" or relying on the rendezvous will need a sweep

The AI-33 tests' file-level doc comments cite "out is unbuffered (stream.go:209)" (`a_i-33_3_test.go:28`) — that comment is informational, not an assertion, but it will become inaccurate if AI-34.1 keeps the buffer at 0 (then the comment is fine, just the line number has drifted) or sets it to N (then the comment is wrong). A search for "unbuffered" in AI-33 test files:

```
$ grep -l "unbuffered" backend/agent/src/ai/openaicompat/a_i-33_*.go
backend/agent/src/ai/openaicompat/a_i-33_3_test.go
```

Just one file, one comment. The fix is a 5-line doc-comment edit and belongs in AI-34.1's diff, not separately. AI-33's tests themselves do not assert `cap(ch) == 0`; they only describe the carrier's behaviour. **None of the AI-33 tests' RED/GREEN status is affected by a buffered carrier.**

### 5.6 The tie-break rule could land AI-34.1 at 0

If M1 high-water never approaches 32, M2 p99 wait < 1 ms, M3 per-stream < 1 KiB, the tie-break rule (line 432) prefers the smaller. **A measured answer of 0 (rendezvous) is in scope.** AI-34.1 must not pre-decide "64" or "32" — the deliverable is "with measurements, not a preference". If the measurements say 0, the decision artefact says 0, and the doc 0002 § 6 starting-capacity clause is amended (the spec already supports this under its own living-graph clause; see also the AI-02.1 decision's "constant-versus-configurable deferred" line 401).

### 5.7 Serial-only posture is non-negotiable through the whole change

`RequireNoGoroutineLeak`'s contract (`stream_kit_leak.go:96–106`) requires serial execution: `tb.Setenv` panics if a parallel ancestor has already called `t.Parallel()`. AI-34.3's exhaustive-path test will use `RequireNoGoroutineLeak`; therefore no `t.Parallel()` in that file. AI-34.2's slow-consumer tests can use `t.Parallel()` if they do not call the leak helper.

### 5.8 The work-unit split must respect the 400-line PR budget

Single-PR delivery was pre-resolved (`delivery_strategy: single-pr`, per the orchestrator's session preflight). The forecast per subnode:

| Subnode | Estimated Δ (production + tests) | Risk vs 400-line budget |
| --- | --- | --- |
| AI-34.1 | decision.md (~100 lines, prose) + `stream.go` 1-line change to `make(chan ai.Event, N)` + ~50 lines of measurement code (`dripFramesServer`, instrumentation). Total ~150 lines. | Low |
| AI-34.2 | 1 file `a_i-34_2_test.go`, ~200 lines (text + tool-call × capacity assertion + slow-consumer + pause-resume scenarios). | Low |
| AI-34.3 | 1 file `a_i-34_3_test.go`, ~150 lines (exhaustive-path slow / bursty / pause-resume × drain-then-assert-no-loss). | Low |
| **Aggregate** | **~500 lines** + decision.md (~100) = ~600 lines | **Medium** — within the 1000-line review budget; outside the 400-line PR budget. **Single PR is acceptable per `single-pr` delivery strategy; tasks.md may still recommend splitting for review focus if the test file exceeds 400 lines alone.** |

The forecast is rough; `sdd-tasks` refines it. The risk that pushes above 1000 lines is low — there is no production-shape change beyond the 1-line buffer argument.

---

## 6. Recommended subtask ordering

The dependency graph is fixed by doc 0002 (AI-34.1 → 34.2 → 34.3). The order within a single PR is:

1. **AI-34.1 — measured buffer decision.** Write the measurement code first (instrumented `dripFramesServer` + capacity-probe helpers). Run M1, M2, M3 against the chosen workload(s). Author `decision.md` with the numbers and the chosen capacity. **Production change here is the 1-line `make(chan ai.Event, N)` change**, with `N` set by the measurements and the tie-break rule. No new dependency. No conformance change.
2. **AI-34.2 — lossless ordering under pressure.** Capacity assertion: `cap(ch) == N` against the real producer. Slow-consumer test: drip frames at 5 ms / 50 ms / pause-200 ms, drain, assert every event arrived, in order, no `Completion` or `ErrorEvent` synthesized. No auxiliary queue: assert by capacity + observation that the producer blocks on send (a sentinel counter on the test goroutine observing a stalled producer goroutine is the load-bearing piece).
3. **AI-34.3 — exhaustive non-cancellation paths.** For each of slow / bursty / pause-resume × text / tool-call, drain to close (no cancel), assert every event arrived in order, no terminal invented, no leak across 50 repeats (R-STK-007 / `RequireNoGoroutineLeak`).

**All three subnodes can land in one PR** (single-PR strategy). The order above ensures AI-34.1's measurement runs and the 1-line production change land first, so AI-34.2's `cap(ch) == N` assertion is RED-first against the **new** `N`, not the old `0`.

---

## 7. Open questions for the user (BLOCKING for `sdd-propose`)

Each question is concrete and answerable from doc 0002 + the code. There are **three**.

### Q1 — Confirm the "buffer 64 vs unbuffered" reconciliation posture

The spec (`openspec/specs/ai-stream-lifecycle/spec.md:387`) names 64; the code (`stream.go:223`) is unbuffered. AI-34.1's decision artefact must record this. **Two acceptable postures:**

- **(a)** Treat the divergence as an implementation oversight: AI-34.1's deliverable is "the measured capacity, named against the spec's 64 starting point" — even if the measurement lands at, say, 17, the artefact cites the spec's 64 as the hypothesis and the measurements as the confirmation or change.
- **(b)** Treat the divergence as a known intentional state: AI-34.1's deliverable is "the measured capacity, recorded against the spec's clause that the number is a hypothesis, not a measurement" — the spec's wording is left as-is, and the decision artefact is the first actual number.

**Which posture do you want on the record?** (Posture (b) is what AI-02.1's own framing of § 6 ("a hypothesis, not a measurement") implies, but AI-33's living-graph precedent suggests posture (a) is what reviewers expect.)

### Q2 — Confirm where the measurement workload lives

The proposed workload (§ 4) needs a host: three options.

- **(a)** A new file `backend/agent/src/ai/openaicompat/measurement_test.go` that runs only with `-tags=measurement` (or similar build tag). CI runs it once per Wave; it's not part of the default `make test` green.
- **(b)** A new file `backend/agent/src/ai/openaicompat/measurement_test.go` that runs by default, gated by `-short` (skipped under `go test -short`).
- **(c)** A standalone `bench_test.go` file run via `go test -bench=. -benchtime=1x ./src/ai/openaicompat/...`. The decision artefact is the bench output captured at PR-merge time.

**Which host do you want?** (Option (b) is what the existing test suite's "fast default" convention implies — `go test -short` is the gating mechanism. Option (c) is the lightest review load but loses the test-tracked assertion.)

### Q3 — Confirm the constant-vs-configurable posture

Doc 0002 line 401 defers this to AI-34.1. Two acceptable answers:

- **(a)** **Constant.** A package-private `const streamCarrierBuffer = N` declared next to `streamReadBufferSize` (`stream.go:123`). No `Config` field. The harness cannot change the buffer.
- **(b)** **Configurable via `openaicompat.Config`.** A `BufferCapacity int` field on `Config` (`backend/agent/src/ai/openaicompat/client.go:43–58`), defaulting to the measured `N` when zero. The wrapper (`openrouter/wrapper.go:127`) does not change.

**Which do you want?** (Posture (a) is the simpler one — fewer moving parts, harder for a future adapter to ship a different value without re-arguing the measurements. Posture (b) is what `openaicompat.Config`'s "no timeout, deadline or bound value" posture — line 41, doc.go — argues against: no bound values appear in `Config`, by design.)

---

## 8. Ready-for-proposal hand-off

The orchestrator can proceed to `sdd-propose` for change `cachicamas-ai-backpressure` with these hand-offs:

- **Three open questions for the user** (Q1, Q2, Q3 above). Each is concrete and answerable from doc 0002 + the code. Q1 and Q3 are blocking for `sdd-propose` (the proposal's "Approach" section cannot finalise without them). Q2 is blocking for `sdd-tasks` (the task list needs to know which test-host mechanism to allocate).
- **No new top-level Go dependency.** Stdlib-only is the AI-34 contract, mirroring AI-33 / AI-22.4's posture (R-STK-009). The `dripFramesServer` helper is stdlib (`httptest`, `io`, `time`).
- **No production dependency changes.** The OTel API/SDK boundary (ADR 0005 § D3) is untouched.
- **No conformance amendment.** R-CNF-011/012 remain unchanged; AI-34.2's `cap(ch)` assertion lives in the real-producer test file, not the conformance suite.
- **Test file location convention:** Internal `package openaicompat` for AI-34.2 / AI-34.3 (matching `a_i-33_*.go`); one file per subnode. Decision artefact (`decision.md`) at `openspec/changes/2026-08-07-cachicamas-ai-backpressure/` (sibling of `proposal.md`).
- **The carrier decision is closed.** No amendment to AI-02.1. No re-opening the iterator option. AI-34 measures the buffer; AI-34 does not choose the carrier.

The orchestrator should pass exact `SKILL.md` paths (`go-testing`, `work-unit-commits`) when delegating the `sdd-apply` work, per `openspec/AGENTS.md` rule "Sub-agent launch contract".

---

## 9. References

- **Charter** — `docs/architecture/milestones/0002-cachicamas-ai-layer-1-task-graph.md:2041–2073`
- **Spec (AI-02.1 § 6 — buffering)** — `openspec/specs/ai-stream-lifecycle/spec.md:381–451`
- **Spec (M1/M2/M3 / tie-break)** — `openspec/specs/ai-stream-lifecycle/spec.md:422–432`
- **Conformance R-CNF-011 / R-CNF-012** — `openspec/specs/ai-provider-conformance-suite/spec.md:163–181`
- **Producer carrier construction** — `backend/agent/src/ai/openaicompat/stream.go:223` (current: `make(chan ai.Event)`)
- **Producer `emit` (backpressure primitive)** — `backend/agent/src/ai/openaicompat/stream.go:574–582`
- **AI-32.3 bounded-wait (unchanged seam)** — `backend/agent/src/ai/openaicompat/stream.go:115, 437–470, 503–526, 679–682`
- **AI-33.3 predecessor tests** — `backend/agent/src/ai/openaicompat/a_i-33_3_test.go` (true-abandonment, R-CNF-012 pinned verbatim)
- **Conformance cancellation cases** — `backend/agent/src/agenttest/conformance_cancellation.go:46–131`
- **Fake saturation recipe** — `backend/agent/src/agenttest/fake_gate_test.go:80–143`
- **Leak helper** — `backend/agent/src/agenttest/stream_kit_leak.go:107`
- **Drain helper** — `backend/agent/src/agenttest/stream_kit_record.go:63`
- **OpenRouter wrapper (forwarder)** — `backend/agent/src/ai/openaicompat/openrouter/wrapper.go:84–86`
- **Real-HTTP test pattern** — `backend/agent/src/ai/openaicompat/bridge_test.go:64–92`, `stream_test.go:50–60`
- **Layer 2 AG-10.3 (consumer pauses by design)** — `docs/architecture/milestones/0003-cachicamas-agent-layer-2-task-graph.md:527–532`
- **ADR 0005 § D1** (layered rule for `backend/agent/`) and § D3 (OTel boundary)
- **Engram obs #2267** — the AI-02.1 carrier decision context (buffer 64 starting number, four grounds)
- **openspec/AGENTS.md** (strict TDD, no new top-level deps, sub-agent skill paths)