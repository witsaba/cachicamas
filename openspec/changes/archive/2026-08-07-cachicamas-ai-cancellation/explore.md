# Explore — Layer 1 cancellation proof and goroutine cleanup

> **Change**: `cachicamas-ai-cancellation`
> **Milestone**: AI-33 — Prove cancellation and goroutine cleanup (Layer 1 / doc 0002)
> **Phase**: explore
> **Project**: cachicamas (witsaba)
> **Date**: 2026-08-06
> **Target module**: `backend/agent/` (Layer 1, layered not hexagonal — see [ADR 0005 § D1](../../../../docs/adr/0005-promote-agent-stack-to-own-module.md))
> **Authoring constraint**: doc 0002's authoring constraint binds every artifact of this change. **No Go identifier is invented here.** All names cited are already present in the merged source at this HEAD.

---

## 1. Identity

AI-33 is the milestone that closes out the Layer 1 cancellation contract: every cancellation moment — before headers, between frames, during a blocked send, after completion — must close the stream cleanly, free the goroutine, and never leak a connection. There are five subnodes, four independent cancellation moments (AI-33.1 … AI-33.4) and one unifying resource-discipline pass over them all (AI-33.5, depends on 33.1 … 33.4). Every scenario runs **over text AND tool-call streams** (charter, line 1989) — a cancellation proof that never crosses the tool-call accumulation path proves nothing about its buffers.

The milestone depends on AI-28 (text events), AI-30 (tool-call events), AI-32 (mid-stream failure / typed terminal failures — already includes the AI-32.3 bounded-wait pattern S-AEM-051/052) and AI-22.4 (leak-check mechanism — already shipped). It blocks AI-34 (backpressure) and AI-35 (retry/idempotency).

This exploration is the opening pass. It does not write production code or tests; it establishes the test seams, names the overlaps, surfaces the architecture decisions, and ranks the ordering.

---

## 2. What the merged source already does (the seams AI-33 attaches to)

Every named function below is **already in `main`** at this worktree (`feat/ai-33-cancellation`, base `main` @ `e9a8054`).

### 2.1 Pre-stream cancellation (AI-33.1) — *largely shipped*

The producer checks context cancellation **before any I/O, channel allocation, or goroutine spawn** — the full AI-33.1 contract is in three lines:

- `backend/agent/src/ai/openaicompat/stream.go:181–183` — the explicit `ctx.Err()` check between `Translate(req)` and `httpReq, err := c.newRequest(...)`.
- `backend/agent/src/ai/openaicompat/stream.go:217–230` — `preStreamCancellation(ctxErr)` builds the typed failure (`ai.FailureCategoryCancellation`, `ai.DeliveryPreStream`).
- `backend/agent/src/ai/openaicompat/stream.go:241–251` — `preStreamTransportFailure` collapses the category to `FailureCategoryCancellation` when `ctx.Err() != nil` at the point of request build / `httpClient.Do` failure.

The remaining gap for AI-33.1 is **proof against a real HTTP connection** (the existing path uses the fake provider; the preamble is testable via `httptest.NewServer` that hangs before headers, already shaped by `contextBeforeFirstFrameServer` in `stream_failure_test.go:391–405`, though used for pre-stream disconnects not cancel-before-headers today).

### 2.2 The single-producer model (AI-33.2, AI-33.3 base)

- `backend/agent/src/ai/openaicompat/stream.go:343` — `func run(ctx context.Context, resp *http.Response, out chan<- ai.Event)` is the **only** producer goroutine (R-ATS-003, R-ATS-005). It is started by `Stream` at line 210.
- `stream.go:344` — `defer close(out)` is the **single closing site**; it fires on every exit path. This is the architectural seam the AI-33.2/-33.3 assertions ride on.
- `stream.go:345` — `defer func() { _ = resp.Body.Close() }()` — closes the body on every exit path. **It does NOT drain** (see § 2.6 below).

### 2.3 The AI-20.3 sanctioned-loss pipe (AI-33.3 base pattern)

- `stream.go:557–565` — `emit(ctx, out, stamper, ev)` selects on `out <- stamped` vs `<-ctx.Done()`. When cancellation wins, it returns false without inventing a terminal event. This is exactly the pattern `fake_provider.go:192–210` `produce()` implements (R-AMP-010 reference shape) and exactly what AI-20.3 promised. Every event this package produces goes through `emit` (design.md D6).

### 2.4 The AI-32.3 bounded-wait refinement (the AI-33.3 nuance)

The real producer does **more** than `emit` on the cancellation branch. After `emit` returns false, the call site checks `ctx.Err()` and, if cancelled, sends a typed terminal failure with a bounded wait:

- `stream.go:437–470` — mid-emit race: when `emit` loses AND `ctx.Err() != nil`, emit a `TextBlockEnd` + `ErrorEvent` with `select { case out <- : case <-time.After(emitFailureSendBound) }` (the 5-second cap declared at `stream.go:115`).
- `stream.go:503–526` — read-failure + cancel path: same shape, same `emitFailureSendBound`.

This is the AI-32.3 amendment (S-AEM-051/052) — a **typed failure-with-bounded-wait**, not a silent loss. The bounded wait is the seam that reconciles this with the conformance suite's R-CNF-011/012 assertions about "no terminal invented" (see § 4.2).

### 2.5 Conformance cases (already registered; reference behaviour, not the AI-33 proof target)

- `backend/agent/src/agenttest/conformance_cancellation.go:34–37` — `init()` registers two cases:
  - `cancellation/bounded_close_leak_free` (line 35) → `cancellationBoundedCloseCase` at line 46.
  - `cancellation/abandoned_then_cancelled_drops_bare` (line 36) → `cancellationAbandonedThenCancelledCase` at line 94.
- `conformance_cancellation.go:46–85` — `cancellationBoundedCloseCase` runs the fake provider, receives the first `EventKindResponseStart`, cancels, expects no further events (`rec.Len() == 0`), and asserts leak-freedom via `RequireNoGoroutineLeak`. The current `Script` shape: `Buffer: 0` (unbuffered), `Steps: [ResponseStart, TextBlockStart, TextDelta, TextBlockEnd]`, no terminal.
- `conformance_cancellation.go:94–132` — `cancellationAbandonedThenCancelledCase` is the saturated-from-the-start path; runs the fake with zero consumer reads, cancels immediately, asserts no error and no completion terminal was invented.

**These cases prove the fake provider's shape, not the real producer's.** AI-33 is the milestone that proves the real producer matches that shape over a real HTTP connection.

### 2.6 The AI-33.5 seam — drain-before-close is missing from `run()`

The drain-before-close idiom **already exists** in the package — but only on the failure path:

- `backend/agent/src/ai/openaicompat/capture.go:94–124` — `captureBody(rc io.ReadCloser)` runs `io.ReadAll(io.LimitReader(rc, captureLimit+1))` (the bounded probe), then **`io.Copy(io.Discard, rc)`** at line 122 to drain the unread remainder before the deferred `Close` at line 95. This is the S-AEM-059 slow-loris guard.
- `capture_proof_test.go:125–189` — proof artifact for the drain + single-close invariant.

The producer's `run()` does **NOT** do this:

- `stream.go:345` — only `defer func() { _ = resp.Body.Close() }()`. No drain.

Per AI-33.5's charter (line 2035: "the failure and cancellation paths are exactly the ones naive implementations leak on"), the `run()` defer chain must be extended: on every exit path — completion, terminal error, each cancellation moment — `resp.Body` must be drained to `io.Discard` *before* the close. This is the surgical move.

---

## 3. Test seams available

All helpers below are already shipped in `backend/agent/src/agenttest`; AI-33 consumes them, does not author new ones unless a gap is surfaced (see § 5.5).

| Seam | Symbol | Location | Notes |
| --- | --- | --- | --- |
| Leak detection | `RequireNoGoroutineLeak(tb, scenario)` | `stream_kit_leak.go:107` | Serial-only via `tb.Setenv` (R-STK-008); 50 repeats with 25 jitter tolerance; **coverage narrowed to cancellation + abandoned-then-cancelled paths only** (R-STK-010). Third-party `go.uber.org/goleak` REJECTED (R-STK-009) to keep `agenttest` dependency-free. |
| Bounded drain | `DrainAndRecord(tb, ch, timeout)` | `stream_kit_record.go:63` | `DefaultDrainTimeout = 2 * time.Second` (line 24). Fails `tb` on deadline, never blocks past it, never partial-returns (R-STK-001). |
| Lined-up kind check | `requireDrainedKinds(tb, events, want)` | `conformance_lifecycle.go:70` | Exact count + positional kind, the AI-33.x-33.4 assertions ride on. |
| Lifecycle prefix | `checkLifecyclePrefix(events, wantID, wantModel)` | `conformance_lifecycle.go:49` | Pure error (R-CNF-020). Useful for AI-33 across text and tool-call streams. |
| Stalling handler (frame-spacing) | `newDripHandler(chunks, interval)` | `openaicompat/timeout_test.go:20–39` | Writes N chunks with a delay between each — the closest existing seam for "idle between frames" (AI-33.2). |
| Stalling handler (pre-stream disconnect) | `contextBeforeFirstFrameServer(t)` | `openaicompat/stream_failure_test.go:391–405` | Hijacks and closes the conn before any response — covers a different shape; useful for the AI-33.1 variant where ctx is cancelled before the first byte. |
| Verbatim transcript server | `bridgeServeTranscripts(transcripts)` | `openaicompat/openrouter/conformance/bridge_test.go:141–158` | Used with text and tool-call bytes; works for AI-33.3 and AI-33.4 over both stream kinds. |
| External-test pattern | `package agenttest_test` at `openaicompat/openrouter/conformance/run_for_test.go:12–19, 83–140` | — | Drives a real `*openaicompat.Client` against a local `httptest.Server`. This is the proven shape for AI-33's RED tests; the conformance suite's internal-call shape cannot cover a real HTTP round-trip. |
| Stream-closed helper for tools | `Helper` | `conformance_redaction.go:92` | Used at 16 sites across the conformance kit. |
| Conformance runner | `FakeFactory()`, `RunConformanceFor(t, f, capability)` | `conformance_suite.go:186–202`, `conformance_scoped.go` | Useful for sanity-running AI-22's existing seams against the proposed `run()` change during AI-33.5. |
| Identity check (R-CNF-016 etc.) | `Declaration cross-checks` | `conformance_suite.go:358` | Not used by AI-33 directly but signals what the conformance suite already asserts about subject behaviour. |

---

## 4. Architecture decisions surfaced

### 4.1 AI-32.3 has already moved the goalposts the doc-0002 text leans on

Doc 0002's AI-33.3 charter (line 2023) says: *"cancellation unblocks it: late events dropped, stream closed without a terminal event — the sanctioned loss path of AI-20.3, now proven against the real producer and leak-checked."*

That sentence was authored before AI-32.3's amendment. The merged run() at `stream.go:437–470` (mid-emit) and `stream.go:503–526` (mid-read) **already emits a typed terminal failure** when cancellation lands mid-stream — except that the send is bounded to `emitFailureSendBound = 5s` (`stream.go:115`). The "no terminal invented" property R-CNF-012 asserts is therefore **conditional**, not absolute: it holds when no consumer is reading (the bounded-wait expires, the event is dropped); it does NOT hold when the consumer is alive but slow. Conformance scope already locks this in (the abandoned-then-cancelled case is the only path the conformance suite asserts).

**Implication for AI-33.3's spec/test.** The test must pin which scenario it is asserting:

- (a) Truly abandoned consumer (no reads at all) → no terminal of *any* kind observable, channel closes bare → maps onto R-CNF-012 directly.
- (b) Live consumer, slow, mid-send, ctx cancelled → AI-32.3 typed failure IS observed → matches stream.go:447–468.

**Recommended spec wording for AI-33.3**: "WHEN the consumer has stopped reading and the producer is blocked mid-send (`out` is saturated and never drained) THEN cancellation unblocks it: late events dropped, stream closes within `emitFailureSendBound` without the bounded failure actually being delivered, and leak-checked." Pin the scenario to "consumer stopped reading", not "any saturated sender", so the assertion matches the conformance amendment's terminology exactly.

**This must be called out to the user at propose time** — it is exactly the doc text ↔ merged-code reconciliation doc 0002's "the living-graph clause" exists for.

### 4.2 AI-33.5's drain-before-close extension to `run()` is the same idiom already proven

`capture.go:117–122` already shows the recipe:

```go
// Drain the unread remainder (slow-loris guard, S-AEM-059) before
// the deferred Close fires.
_, _ = io.Copy(io.Discard, rc)
return capturedBody{data: data}
```

The same `io.Copy(io.Discard, rc)` (or a `io.CopyBuffer` variant if the existing `buf` is reused) belongs in the `run()` defer chain. The change is small and the proof pattern (`capture_proof_test.go:125–189`) translates directly: an `httptest` server that writes more bytes than the producer consumes, asserting that the connection is freed (the harness's `httptest.Server.Close()` does not block; the next request against the same server succeeds).

### 4.3 The producer's read-loop is NOT ctx-aware — AI-33.2's hardest test

`stream.go:369–368`'s read loop is `n, readErr := resp.Body.Read(buf)` — a synchronous block. If the transport honours ctx via `http.Client.Do` (it does in Go 1.26+ for HTTP/2), cancellation propagates through the request and Body.Read returns an error after the connection is torn down. **But this depends on the transport, the protocol (HTTP/1.1 vs HTTP/2), and whether any consumer has yet sent a request that opened the keep-alive slot.**

For an `httptest.NewServer` over HTTP/1.1 (the test-only harness), the question is empirical. The test must stand up a stalling handler (`newDripHandler(2, 1s)` and never sent the second chunk), open the stream, cancel ctx, and bound the close to `DefaultDrainTimeout + safety margin`. **If the test fails on a busy GHA runner**, an implementation change may be required (a watcher goroutine that calls `resp.Body.Close()` on `ctx.Done()`, or wrapping the body in a `ctx-awareReader`). The doc 0002 line-2017 test for AI-33.2 accepts a bounded wait, so a 2-second cap is consistent with the existing posture.

The "stuck body" case is *not* the same as the conformance suite's leaked-goroutine scenario. Whatever mechanic emerges (read-side close-watcher or transport-level cancellation) MUST be serial-only (`Stream` is called serially from the test) and MUST NOT introduce a second goroutine into the "single producer" model (R-ATS-003). If it does, the model becomes "producer + ctx watcher", which means leak-free AI-33.5 has to count both.

### 4.4 AI-33.4 (after completion) is the trivially-correct test in this set

`stream.go:393` (`emit(ctx, out, stamper, completion)`) and the `return` on the next line mean the producer exits *before* the consumer can observe completion on a separate goroutine. After `close(out)`, cancellation is a no-op for `run` — the goroutine is already gone. The AI-33.4 scenario is therefore a **race-detector proof** that `cancel()` arriving anywhere on the producer's normal exit path produces:

1. no interleaving panic in the consumer's drain,
2. exactly one close of `out`,
3. no leaked goroutine under `-race`.

The `RequireNoGoroutineLeak` repeat-loop already gives the race-detector coverage. The "exactly one close" assertion is satisfied by the single `defer close(out)` at `stream.go:344` and is testable by counting `runtime.SetFinalizer` / a custom counter on a one-goroutine-aware channel — or by simple drain observation (the channel either returns events and then closes, or returns zero and closes; never closes twice).

### 4.5 AI-33.1's two faces — before-headers vs the conformance amendment's "stream cancelled vs ResponseStart present"

The conformance amendment (`openspec/changes/archive/2026-08-05-cachicamas-ai-conformance-lifecycle-amendment/`) added ResponseStart to the `cancellation/bounded_close_leak_free` script (lines 12–19 of its `design.md`), but kept "no terminal — cancelled stream closes bare (AI-20.3)". The amended assertion: first received kind is `EventKindResponseStart`; `rec.Len() == 0` after cancel.

For the real-producer AI-33.1 test the wording must say "WHEN cancellation lands before the response begins, BEFORE any ResponseStart is received" — the post-amendment conformance case asserts ResponseStart IS received, so the AI-33.1 real test is the **inverse**: ctx cancelled before any byte crosses. The existing `contextBeforeFirstFrameServer` (`stream_failure_test.go:391–405`) is the right fixture; the AI-33.1 case adds cancellation to it (cancel-before-Do, not the existing hijack-after-accept). Expect a `*ai.Failure` with `DeliveryPreStream` and `FailureCategoryCancellation`; the channel is nil; no goroutine was spawned.

---

## 5. Risks and constraints

### 5.1 AI-33.2 read-loop may require a code change, not just a test

If the empirical test reveals that `httptest`-served HTTP/1.1 does not unblock `Body.Read` on ctx cancellation within `DefaultDrainTimeout + safety`, AI-33.2 becomes a `[decision]`-or-`[guard]` of its own: either wrap the body in a ctx-aware reader or introduce a watcher goroutine that closes the body on ctx.Done. Either move risks changing the producer's goroutine count (R-ATS-003), which in turn changes AI-33.5's leak tolerance arithmetic. **Recommendation: write the test FIRST (RED) and let the failure dictate the implementation shape.**

### 5.2 Pin the "no terminal invented" wording to a concrete test condition

The AI-33.3 charter's "stream closed without a terminal event" is true under bounded-wait + truly-abandoned-consumer. It is **false** under bounded-wait + slow-but-alive consumer. The spec must pick one; the explore phase recommends "truly-abandoned" so the conformance suite's R-CNF-012 wording stays verbatim. Otherwise the spec diverges from the merged code's behaviour and AI-33.5's "every exit path closes the body" still holds — but the test would assert something the bounded-wait actively prevents.

### 5.3 No new Go dependencies

`openspec/AGENTS.md` rule 5: "New top-level dependency ⇒ ADR first." The leak-check mechanism decision (R-STK-009 in `stream_kit_leak.go`) explicitly rejected `go.uber.org/goleak` for the same reason. AI-33 must not introduce new module deps — `RequireNoGoroutineLeak`'s stdlib-only implementation is the canonical seam, full stop.

### 5.4 Serial-only posture is non-negotiable through the whole change

`RequireNoGoroutineLeak`'s contract (`stream_kit_leak.go:96–106`) requires serial execution: `tb.Setenv` panics if a parallel ancestor has already called `t.Parallel()`. AI-33.5's "full-package leak check passes with the AI-22.4 mechanism applied wholesale" (line 2037) **must run as a single serial suite wrapper** — not a parallel-by-default pattern. The test files for AI-33 cannot use `t.Parallel()` for any test that calls `RequireNoGoroutineLeak`.

### 5.5 No test-fixture additions needed for the core proof

Every named seam in § 3 already exists. The orchestrator should expect AI-33's PRD/spec/tasks to **reuse, not re-author** the helpers above. A `newStallingHandler` helper might be useful as a single-line wrapper around `newDripHandler` if readability suffers — but it is a refactor, not a new harness.

### 5.6 Both stream kinds must be covered per the charter

AI-33's charter (line 1989) explicitly states every node runs over **text AND tool-call streams**. Tool-call streams are byte-fragmented (R-ATC-010) which means: more accumulator state held in the producer, more blocks to close on a typed-failure emission, more bounded-wait fails. The text-only R-CNF-011 conformance case is **not** sufficient evidence — each AI-33 subnode needs a tool-call variant too.

---

## 6. Recommended subtask ordering

### 6.1 The dependency graph (already stated by doc 0002)

AI-33.1, AI-33.2, AI-33.3, AI-33.4 are independent **.leaves**; AI-33.5 depends on all four. Doc 0002's mermaid (line 1999–2005) makes this explicit. The ordering question is which 33.x to land first.

### 6.2 Recommended order (one PR per leaf, chained when any exceeds the 400-line review budget)

1. **AI-33.1 — cancel before headers.** Smallest scope, already largely implemented (the codebase handles pre-stream cancellation; the gap is the real-HTTP proof). Highest confidence the GREEN is a one-test-file PR.
2. **AI-33.4 — cancel after completion.** The trivially-correct path. Establishes the race-detector posture that AI-33.2 and AI-33.3 will inherit.
3. **AI-33.2 — cancel between frames.** The most empirically uncertain. RED may require an implementation change (see § 5.1); that change might be substantial enough to require its own `[decision]` amendment to doc 0002 under the living-graph clause. Land it earlier than AI-33.3 so any surprise is caught before the more politically charged loss-path work.
4. **AI-33.3 — cancel during a blocked send.** Carries the doc-text reconciliation in § 4.1. Land this after AI-33.2 so the producer's read-loop behaviour is settled.
5. **AI-33.5 — resource discipline.** Depends on all four. The full-package leak check + drain-before-close in `run()` belongs in its own PR. The drain is the surgical code change (see § 2.6 and § 4.2); the rest is asserting that `RequireNoGoroutineLeak` passes for every exit path across text AND tool-call streams.

### 6.3 PR-budget plan

Forecast well under 400 changed lines per subnode if the existing helpers are reused. AI-33.5 is the only one likely to add a non-trivial chunk (drain helper in `run()` + a leak-check suite wrapper that walks every exit path). If the aggregate exceeds 400 lines, `sdd-tasks` should chain into two PRs: AI-33.5a (drain implementation) and AI-33.5b (full-package leak check).

---

## 7. Ready-for-proposal hand-off

The orchestrator can proceed to `sdd-propose` for change `cachicamas-ai-cancellation` with these hand-offs:

- **One open question for the user (BLOCKING for spec):** Confirm the AI-33.3 scenario scoping per § 4.1 — "truly-abandoned consumer (no reads)" vs "any saturated consumer". The doc 0002 text leans one way (the conformance amendment locks it in); the wording of the AI-33.3 spec must pick. AI-33.1 and AI-33.4 have no such ambiguity; AI-33.2's question resolves itself at the RED step.
- **Test file location convention:** Internal `package openaicompat` for the real-producer `/a_i-33_*.go` tests (matching `stream_failure_test.go`, `terminal_test.go`, etc.); external `package openaicompat_test` for the AI-33.5 full-package leak-check, mirroring `bridge_test.go`. Conformance suite is **unchanged** by this change — it already proves the abstract physics.
- **No new top-level Go deps.** `RequireNoGoroutineLeak`'s stdlib-only posture is the contract.
- **No production dependency changes.** The OTel API/SDK boundary (ADR 0005 § D3) is untouched.

The orchestrator should pass exact `SKILL.md` paths (`go-testing`, `work-unit-commits`) when delegating the `sdd-apply` work, per `openspec/AGENTS.md` rule "Sub-agent launch contract".

---

## 8. References

- **Charter** — `docs/architecture/milestones/0002-cachicamas-ai-layer-1-task-graph.md:1987–2037`
- **Predecessor amendments already merged** — `openspec/changes/archive/2026-08-05-cachicamas-ai-conformance-lifecycle-amendment/design.md` (added `ResponseStart` prefix to `cancellation/bounded_close_leak_free`)
- **Conformance suite cancellation cases** — `backend/agent/src/agenttest/conformance_cancellation.go`
- **Leak helper** — `backend/agent/src/agenttest/stream_kit_leak.go`
- **Drain helper** — `backend/agent/src/agenttest/stream_kit_record.go` (`DrainAndRecord`)
- **Producer** — `backend/agent/src/ai/openaicompat/stream.go` (lines 162–212 entry, 343–531 `run`, 557–565 `emit`, 597–666 `emitFailure`)
- **Drain-before-close idiom** — `backend/agent/src/ai/openaicompat/capture.go:117–122`
- **External test pattern** — `backend/agent/src/ai/openaicompat/openrouter/conformance/run_for_test.go`
- **ADR 0005 § D1** (layered rule for `backend/agent/`) and § D3 (OTel boundary)
- **openspec/AGENTS.md** (strict TDD, no new top-level deps, sub-agent skill paths)
