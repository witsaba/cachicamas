# Delta for `ai-stream-lifecycle` — AI-33 cancellation proof and resource discipline

> **Change**: `cachicamas-ai-cancellation` · **Milestone**: AI-33 — Prove cancellation and goroutine cleanup (doc 0002 § 1987–2037) · **Wave**: Wave 5 — Harden
> **Status**: **delta** — amends [`openspec/specs/ai-stream-lifecycle/spec.md`](../../../../specs/ai-stream-lifecycle/spec.md) § 4 (V-STR-05 ownership) and § 5 (V-STR-06/V-STR-07 cancellation) per the contract's amendment rule (line 30–41)
> **Format**: RFC 2119 + Given/When/Then; per-scenario pins marked `(pin) *(regression)*` against the conformance assertion they must not break
> **Conformance alignment**: `R-CNF-011` / `R-CNF-012` wording cited verbatim; `R-STK-007` / `R-STK-008` cited for the leak posture
> **Sources**: [proposal.md](../proposal.md) · [explore.md](../explore.md) · [doc 0002 § 1987–2037](../../../../../docs/architecture/milestones/0002-cachicamas-ai-layer-1-task-graph.md)

---

## Identity

| Field | Value |
| --- | --- |
| **Change name** | `cachicamas-ai-cancellation` |
| **Milestone** | AI-33 (doc 0002 lines 1987–2037) |
| **Capability amended** | `ai-stream-lifecycle` (per proposal § 8) |
| **Type** | Delta — `ADDED` requirements; **no `MODIFIED`, no `REMOVED`, no `RENAMED`** (the amended clauses are added to the existing § 4 and § 5 obligations; nothing previously stated is weakened) |
| **Conformance unchanged** | `R-CNF-011` and `R-CNF-012` are not modified; the AI-33 scenarios pin themselves to the verbatim wording already in `openspec/specs/ai-provider-conformance-suite/spec.md` lines 163–181 |
| **Test runner** | `cd backend/agent && make test` (`go test -race -v ./...`) |
| **Module** | `backend/agent` — **layered** (not hexagonal), per ADR 0005 § D1 |

### What is being added

A body-lifecycle clause (AI-33.5) — drain-before-close on every exit path of the producer — and four cancellation-moment clauses (AI-33.1 … AI-33.4) proven against a real HTTP transport, not the fake provider. The conformance suite already proves the abstract physics over the fake (`R-CNF-011`/`R-CNF-012`); AI-33 proves the **real producer matches that physics** over a `*http.Response` body.

### Producer seams this delta rides on (all already merged at base `main` @ `e9a8054`)

| Seam | Location | Subnodes that ride on it |
| --- | --- | --- |
| Pre-stream cancellation check / builders | `stream.go:181–183`, `:217–230`, `:241–251` | AI-33.1 |
| Single producer goroutine `run()` + `defer close(out)` (single closing site) + `defer resp.Body.Close()` (no drain) | `stream.go:343–345` | All four + AI-33.5 (drain is the surgical addition) |
| `emit` send-discipline (`out` vs `ctx.Done()` race) + AI-32.3 bounded-wait terminal (`emitFailureSendBound = 5s`, `stream.go:115`) | `stream.go:437–470`, `:503–526`, `:557–565` | AI-33.3 |
| Drain-before-close idiom (reference pattern) | `capture.go:117–122` | AI-33.5 |
| `RequireNoGoroutineLeak` (serial-only, stdlib-only) + `DrainAndRecord` (`DefaultDrainTimeout = 2s`) | `agenttest/stream_kit_leak.go:107`, `stream_kit_record.go:63` | Every subnode |
| Test fixtures: `bridgeServeTranscripts`, `newDripHandler`, `contextBeforeFirstFrameServer` | `openrouter/conformance/bridge_test.go:141–158`, `openaicompat/timeout_test.go:20–39`, `openaicompat/stream_failure_test.go:391–405` | AI-33.1, 33.2, 33.3, 33.4 |

---

## ADDED Requirements

### R-AIS-033 — Body lifecycle: drain-before-close on every exit path

> **Purpose.** The producer's defer chain currently closes `resp.Body` without draining it (`stream.go:345`). `capture.go:117–122` proves the drain-before-close idiom is already used on the bounded-capture path; this requirement extends that idiom to the producer's own defer chain so every exit path — completion, terminal error, and each cancellation moment — drains the response body before the close fires. The drain is the resource-discipline obligation that AI-33.5 proves.

The producer MUST drain the HTTP response body to a discarding sink before the deferred `resp.Body.Close()` fires, on every exit path. The drain MUST run as part of the producer's defer chain (the same defer site that owns `close(out)` at `stream.go:344`), MUST NOT introduce a second persistent goroutine (`R-ATS-003`'s single-producer model), and MUST complete before the deferred close returns. The drain MUST NOT depend on a new top-level Go dependency — `io.Copy` against `io.Discard` is the only acceptable mechanism. The drain MUST be silent: any error from the drain is the network's concern, not a Layer 1 contract concern.

#### Scenario: R-AIS-033 / S-1 — Drain fires on the normal completion path *(pin: `R-CNF-005`, `R-CNF-009`)*

- **GIVEN** a real `*http.Response` whose body carries more bytes than the consumer reads before the terminal event lands
- **WHEN** the producer reaches the `[DONE]` sentinel and exits normally (`stream.go:387–395`)
- **THEN** the response body is drained to `io.Discard` before `resp.Body.Close()` runs, AND the next request against the same transport succeeds without blocking on the prior connection's unread bytes

#### Scenario: R-AIS-033 / S-2 — Drain fires on the in-band terminal error path *(pin: `R-CNF-009`, `R-AEM-010`)*

- **GIVEN** a real `*http.Response` that emits an in-band error frame mid-stream after some body bytes have been sent
- **WHEN** the producer takes the in-band error branch (`stream.go:403–418`)
- **THEN** the response body is drained before the deferred close, AND the stream closes exactly once

#### Scenario: R-AIS-033 / S-3 — Drain fires on the malformed-frame terminal-error path *(pin: `R-CNF-009`, `R-AEM-022`)*

- **GIVEN** a real `*http.Response` whose body carries a frame the decoder rejects
- **WHEN** the producer takes the malformed-frame branch (`stream.go:420–424`)
- **THEN** the response body is drained before the deferred close, AND the transport's connection pool is not poisoned

#### Scenario: R-AIS-033 / S-4 — Drain fires on the pre-headers cancellation path *(pin: `R-CNF-011`)*

- **GIVEN** the caller's context is cancelled **before** `httpClient.Do` returns a response
- **WHEN** the producer never spawns (pre-stream cancellation path, `stream.go:181–183`)
- **THEN** the defer chain in `run()` is not reached, AND no body exists to drain, AND no leak is observed under `R-STK-007`'s amplitude check

#### Scenario: R-AIS-033 / S-5 — Drain fires on the between-frames cancellation path *(pin: `R-CNF-011`, `R-STK-028`)*

- **GIVEN** a real `*http.Response` whose body stalls between frames (use `newDripHandler` from `openaicompat/timeout_test.go:20–39`)
- **WHEN** the caller's context is cancelled while `run()` is blocked in `Body.Read` (`stream.go:370`)
- **THEN** the response body is drained before the deferred close, AND the stream closes within `DefaultDrainTimeout + safety margin`, AND `RequireNoGoroutineLeak` repeats show no growth beyond `leakTolerance`

#### Scenario: R-AIS-033 / S-6 — Drain fires on the blocked-send abandonment path *(pin: `R-CNF-012`)*

- **GIVEN** a consumer that has stopped reading and a real `*http.Response` whose body continues to emit bytes
- **WHEN** the producer is blocked mid-send (`emit` loses the `out <- ev` vs `ctx.Done()` race, `stream.go:557–565`) and the bounded-wait terminal fires (`stream.go:447–468`)
- **THEN** the response body is drained before the deferred close, AND the stream closes within `emitFailureSendBound`, AND no terminal event reaches the consumer (the abandoned-then-cancelled saturated path)

#### Scenario: R-AIS-033 / S-7 — Drain fires on the after-completion cancellation path *(pin: `R-CNF-009`, `R-CNF-011`)*

- **GIVEN** a real `*http.Response` whose terminal event has been emitted and the consumer has drained it
- **WHEN** the caller's context is cancelled after the producer has already exited (`run` has returned)
- **THEN** the response body was drained before the producer exited, AND the stream is already closed, AND cancelling the (now-defunct) context is a no-op that does not panic under `-race`

---

### R-AIS-034 — Cancellation before headers is reported without producing a stream

> **Purpose.** AI-33.1. The pre-stream cancellation path is already implemented (`stream.go:181–183`, `:217–230`, `:241–251`); what is missing is **proof against a real HTTP transport** using `httptest.NewServer` + `contextBeforeFirstFrameServer`. The scenario is the inverse of the conformance `R-CNF-011` amendment: conformance proves that after `ResponseStart` lands, cancellation drops the rest bare; AI-33.1 proves that **before any byte crosses**, cancellation returns the typed failure without ever returning a stream.

When the caller's context is cancelled **before** the producer's `httpClient.Do` returns a response (or equivalently, before any response byte crosses the wire), the call MUST return a pre-handover failure carrying `ai.FailureCategoryCancellation` and `ai.DeliveryPreStream`. No stream MUST be returned. No producer goroutine MUST be spawned. No HTTP response body MUST be opened. The failure's category and delivery axis MUST match `R-ATS-002` (pre-stream cancellation, `stream.go:217–230`).

#### Scenario: R-AIS-034 / S-1 — Text stream, ctx cancelled before `httpClient.Do` *(pin: `R-CNF-011`, `R-ATS-002`)*

- **GIVEN** a real `*openaicompat.Client` and an `httptest.NewServer` whose handler hangs before headers
- **WHEN** the caller invokes `Stream(ctx, req)` with a `ctx` already cancelled at call time
- **THEN** the call returns a non-nil `*ai.Failure` with `Category == FailureCategoryCancellation` and `Delivery == DeliveryPreStream`, AND the returned channel is nil, AND no goroutine was spawned, AND `RequireNoGoroutineLeak` (50 repeats) shows no growth

#### Scenario: R-AIS-034 / S-2 — Tool-call stream, ctx cancelled before `httpClient.Do` *(pin: `R-CNF-011`, `R-ATS-002`)*

- **GIVEN** the same setup, with a tool-call request rather than a text request
- **WHEN** the caller invokes `Stream(ctx, req)` with a cancelled `ctx`
- **THEN** the same observable outcome as R-AIS-034 / S-1 — typed failure, no stream, no goroutine, no leak

#### Scenario: R-AIS-034 / S-3 — Race: ctx cancelled while `httpClient.Do` is in flight *(pin: `R-CNF-011`, `R-ATS-002`)*

- **GIVEN** a real `*openaicompat.Client` and an `httptest.NewServer` whose handler accepts the connection but never writes a status line
- **WHEN** the caller invokes `Stream(ctx, req)` and cancels `ctx` after `httpClient.Do` returns but before `run()` reads any byte
- **THEN** the call returns a pre-handover `*ai.Failure` with `Category == FailureCategoryCancellation`, AND no stream was returned, AND no leak is observed

---

### R-AIS-035 — Cancellation between frames closes the stream within bounded time and frees the connection

> **Purpose.** AI-33.2. The producer's read loop (`stream.go:369–530`) is **not** context-aware; it depends on transport-level cancellation propagating through `Body.Read`. For an `httptest.NewServer` over HTTP/1.1, this is **empirical**: the test must stand up a stalling handler, cancel, and bound the close to `DefaultDrainTimeout + safety margin`. If the RED test shows the dependency does not hold, the implementation change is a `Body.Close()` watcher or a ctx-aware wrap — but **must not** introduce a persistent second goroutine (`R-ATS-003`).

When the caller's context is cancelled while the producer is blocked reading the next frame from the response body, the producer MUST exit within `DefaultDrainTimeout + safety margin`, the stream MUST close exactly once, the response body MUST be closed (so a stalled server cannot pin the connection), and no goroutine MUST outlive the call. Under `-race`, repeated runs MUST show no interleaving panic.

#### Scenario: R-AIS-035 / S-1 — Text stream, cancel while idle between frames *(pin: `R-CNF-011`, `R-STK-028`)*

- **GIVEN** a real `*openaicompat.Client` and an `httptest.NewServer` whose handler uses `newDripHandler(chunks, interval)` to emit one frame and then stall
- **WHEN** the caller invokes `Stream(ctx, req)`, receives the first event, then cancels `ctx` before the second frame
- **THEN** the producer exits within `DefaultDrainTimeout + safety margin`, the channel closes exactly once, the response body is closed (the next request against the same transport succeeds without blocking on the prior connection), AND `RequireNoGoroutineLeak` (50 repeats) shows no growth

#### Scenario: R-AIS-035 / S-2 — Tool-call stream, cancel while idle between frames *(pin: `R-CNF-011`, `R-STK-028`)*

- **GIVEN** the same setup, with a tool-call transcript (`bridgeServeTranscripts` over tool-call bytes)
- **WHEN** the caller invokes `Stream(ctx, req)`, receives the first tool-call event, then cancels `ctx` between frames
- **THEN** the same observable outcome as R-AIS-035 / S-1 — bounded close, single close, body closed, no leak

#### Scenario: R-AIS-035 / S-3 — Connection freed: next request against the same transport succeeds *(pin: `R-CNF-011`)*

- **GIVEN** the stalling-handler server and a `*http.Client` whose transport reuses connections
- **WHEN** scenario R-AIS-035 / S-1 completes and the test issues a second request against the same server
- **THEN** the second request completes promptly without waiting on the first connection's stale keep-alive slot

---

### R-AIS-036 — Truly-abandoned consumer + cancellation drops cleanly with no terminal invented

> **Purpose.** AI-33.3, **scoped per proposal § 7 Open decision to "truly-abandoned consumer"** (conformance `R-CNF-012` wording verbatim). The scenario pins the case where "consumer has stopped reading, `out` is saturated and never drained" — NOT "any saturated `out`". A slow-but-alive consumer mid-send is the AI-32.3 bounded-wait terminal case (`stream.go:437–470`), which is a different observation (see out-of-scope clause below).

**Pinned wording — cited verbatim from `openspec/specs/ai-provider-conformance-suite/spec.md` lines 173–181 (R-CNF-012):**

> "The suite MUST assert AI-20.3's saturated-drop physics: a consumer that stops reading until the buffer saturates, whose caller then cancels, MUST see the stream close **bare** — no terminal event invented, no undelivered event forced through, no leak. The suite MUST NOT assert the **abandoned-never-cancelled** path; that narrowing is inherited from `ai-stream-lifecycle` § 5's untestability ruling and is recorded here rather than left silently absent."

When the consumer has **stopped reading** and the producer is blocked mid-send and the caller's context is cancelled, the stream MUST close **bare** — no terminal event of any kind observed by the consumer, the events that did land intact, and no goroutine leak. The bounded-wait cap (`emitFailureSendBound = 5s`, `stream.go:115`) IS the bounded close; the abandonment is what makes the bounded-wait terminal fail to land.

#### Scenario: R-AIS-036 / S-1 — Text stream, truly abandoned consumer, then cancel *(pin: `R-CNF-012`, `R-STK-029`)*

- **GIVEN** a real `*openaicompat.Client` and an `httptest.NewServer` serving a text transcript via `bridgeServeTranscripts`
- **WHEN** the caller invokes `Stream(ctx, req)`, **never reads** from the channel, and immediately cancels `ctx` so the producer is blocked on the first send
- **THEN** the stream closes bare within `emitFailureSendBound + safety`, no `ai.Completion` and no `ai.ErrorEvent` is observed by any future reader, AND `RequireNoGoroutineLeak` (50 repeats) shows no growth

#### Scenario: R-AIS-036 / S-2 — Tool-call stream, truly abandoned consumer, then cancel *(pin: `R-CNF-012`, `R-STK-029`)*

- **GIVEN** the same setup, with a tool-call transcript
- **WHEN** the caller invokes `Stream(ctx, req)`, **never reads**, and cancels `ctx` while the producer is blocked sending the first tool-call event
- **THEN** the same observable outcome as R-AIS-036 / S-1 — bare close, no invented terminal, no leak

#### Scenario: R-AIS-036 / S-3 — Abandoned-never-cancelled path is **not** asserted *(pin: `R-CNF-012` narrowing, `R-STK-010`)*

- **GIVEN** the AI-33.3 test files
- **WHEN** a reviewer looks for an abandoned-never-cancelled test
- **THEN** none exists, AND the absence is recorded (the test file's doc comment cites `ai-stream-lifecycle` § 5's untestability ruling and `R-STK-010`'s deliberate narrowing)

---

### R-AIS-037 — Cancellation after completion is a no-op; close happens exactly once

> **Purpose.** AI-33.4. After `stream.go:393` emits the completion event and the producer returns, the goroutine is already gone. Cancelling a context whose stream has already closed MUST be a no-op: no interleaving panic in the consumer's drain, exactly one close on the channel, no leak under `-race`. This is the trivially-correct path; it exists to establish the race-detector posture that AI-33.2 and AI-33.3 will inherit.

When the caller's context is cancelled **after** the producer has emitted the terminal event and exited, the channel MUST already be closed (or close cleanly with no further events), the close MUST happen **exactly once**, no consumer-side interleaving panic MUST occur under `-race`, and no goroutine MUST outlive the call.

#### Scenario: R-AIS-037 / S-1 — Text stream, cancel after completion *(pin: `R-CNF-009`, `R-CNF-011`)*

- **GIVEN** a real `*openaicompat.Client` and an `httptest.NewServer` serving a short text transcript that ends in `[DONE]`
- **WHEN** the caller invokes `Stream(ctx, req)`, drains to close, then cancels `ctx`
- **THEN** the channel was closed exactly once, the recorded events carry exactly one terminal, AND no panic occurs under `-race` across 50 repeats

#### Scenario: R-AIS-037 / S-2 — Tool-call stream, cancel after completion *(pin: `R-CNF-009`, `R-CNF-011`)*

- **GIVEN** the same setup, with a tool-call transcript ending in `[DONE]`
- **WHEN** the caller invokes `Stream(ctx, req)`, drains to close, then cancels `ctx`
- **THEN** the same observable outcome as R-AIS-037 / S-1 — exactly one close, exactly one terminal, no panic, no leak

#### Scenario: R-AIS-037 / S-3 — Race: cancel and final receive interleave *(pin: `R-CNF-009`, `R-CNF-011`)*

- **GIVEN** the same text-transcript setup
- **WHEN** the caller issues `cancel()` concurrently with the final receive, across 50+ repeats
- **THEN** no panic occurs under `-race`, AND exactly one terminal is observed, AND the channel is closed exactly once

---

### R-AIS-038 — Full-package leak check covers every exit path on both stream kinds

> **Purpose.** AI-33.5's second clause (charter line 2036). A serial, full-package leak check wraps every exit path — completion, terminal error, each cancellation moment — across text **and** tool-call streams, using `RequireNoGoroutineLeak` as the canonical amplitude-based mechanism (`R-STK-007`/`R-STK-008`).

A single serial test suite MUST run every AI-33 exit path across both stream kinds through `RequireNoGoroutineLeak`. The suite MUST NOT call `t.Parallel()` (`R-STK-008`'s serial-only contract). Each scenario MUST be repeated the helper's stated number of times. The helper's stdlib-only posture (`R-STK-009`) MUST be preserved — `go.mod` MUST declare zero new requires.

#### Scenario: R-AIS-038 / S-1 — Full-package serial leak check passes *(pin: `R-STK-007`, `R-STK-008`, `R-STK-009`)*

- **GIVEN** the AI-33 test files for subnodes 33.1, 33.2, 33.3, 33.4
- **WHEN** a single serial test (`package openaicompat_test`, mirroring `bridge_test.go`'s posture) wraps each scenario in `RequireNoGoroutineLeak`
- **THEN** no goroutine growth beyond `leakTolerance` is observed on any path, AND no `t.Parallel()` call exists in any of the AI-33 test files

#### Scenario: R-AIS-038 / S-2 — Both stream kinds covered per scenario *(pin: `R-CNF-005`, `R-CNF-007`)*

- **GIVEN** the AI-33 test files
- **WHEN** a reviewer enumerates the scenarios per subnode
- **THEN** each of 33.1, 33.2, 33.3, 33.4 has at least one text-stream scenario AND at least one tool-call-stream scenario (charter line 1989)

#### Scenario: R-AIS-038 / S-3 — Module dependency unchanged *(pin: `R-STK-009`, `NFR-CNF-A`)*

- **GIVEN** `backend/agent/go.mod` at base `main` and after this delta is applied
- **WHEN** a reviewer diffs `go.mod`
- **THEN** no new require is added, AND `RequireNoGoroutineLeak`'s stdlib-only posture is preserved

---

## Pins / regressions

Every behavior leaf in this delta is pinned to a conformance or contract assertion it must not break. The pin is the conformance-or-contract identifier; the regression marking is the requirement that the merged AI-33 work must leave the conformance suite green and the contract assertions intact.

| AI-33 subnode | Behavior leaf | Conformance / contract pin | Regression assertion |
| --- | --- | --- | --- |
| AI-33.1 | Pre-headers cancellation, text | `R-CNF-011` | Conformance `cancellation/bounded_close_leak_free` stays green |
| AI-33.1 | Pre-headers cancellation, tool-call | `R-CNF-011`, `R-ATS-002` | Conformance suite unchanged |
| AI-33.2 | Between-frames cancellation, text | `R-CNF-011`, `R-STK-028` | Conformance cancellation case stays green |
| AI-33.2 | Between-frames cancellation, tool-call | `R-CNF-011`, `R-STK-028` | Conformance suite unchanged |
| AI-33.3 | Truly-abandoned + cancel, text | `R-CNF-012` (verbatim) | Conformance `cancellation/abandoned_then_cancelled_drops_bare` stays green |
| AI-33.3 | Truly-abandoned + cancel, tool-call | `R-CNF-012` (verbatim) | Conformance suite unchanged |
| AI-33.4 | After-completion cancellation, text | `R-CNF-009`, `R-CNF-011` | Conformance exactly-one-terminal case stays green |
| AI-33.4 | After-completion cancellation, tool-call | `R-CNF-009`, `R-CNF-011` | Conformance suite unchanged |
| AI-33.5 | Drain-before-close, every exit path | `R-CNF-009`, `R-ATS-003` | `R-ATS-003`'s single-producer model preserved |
| AI-33.5 | Full-package leak check | `R-STK-007`, `R-STK-008`, `R-STK-009` | Helper stdlib-only posture preserved; `go.mod` unchanged |
| All | Tool-call variant per subnode | `R-CNF-007`, `R-ATL-009` | Tool-call ordering invariants preserved across exit paths |
| All | Serial-only posture | `R-STK-008` | No `t.Parallel()` call introduced |

---

## Out of scope

Aligned with proposal § 3.2. Each item names the owner that does own it.

| Item | Owner |
| --- | --- |
| Layer 2+ stop semantics | Layer 2 / AI-34, AI-35 |
| Backpressure (buffer capacity, V-STR-08 falsification, the M1/M2/M3 measurements) | AI-34 |
| Retry / idempotency (`V-FAIL-15`'s never-retry-partial-output clause) | AI-35 |
| Conformance suite content | **Unchanged** — `R-CNF-011` / `R-CNF-012` already prove the abstract physics over the fake provider; AI-33 proves the real producer matches |
| AI-32.3 typed-failure semantics | **Unchanged** — AI-33 proves it, does not rewrite it. The bounded-wait terminal at `stream.go:437–470` and `:503–526` is the seam AI-33.3 rides on |
| New top-level Go dependencies | **Blocked** — `RequireNoGoroutineLeak`'s stdlib-only posture (`R-STK-009`, `stream_kit_leak.go:1–25`) is the contract; `go.mod` MUST stay unchanged |
| Slow-but-alive consumer mid-send (AI-32.3 bounded-wait observable terminal) | **Out of AI-33.3** — proposal § 7 pins AI-33.3 to truly-abandoned consumers only; the slow-but-alive case is the AI-32.3 typed-failure observation, which is asserted by the conformance suite already |
| Abandoned-never-cancelled path | **Untestable** — `ai-stream-lifecycle` § 5 rules it untestable to termination; `R-STK-010`'s narrowing records the absence on the record |
| Migration of ad-hoc drain helpers in `ai/provider_test.go` and `agenttest/fake_text_test.go` | A later tidy-up — out of scope per `ai-stream-testkit` spec lines 36–40 |
| Drain-on-slow-but-alive consumer perturbing AI-21 scripted scenarios | Proposal R6 — verified by running the full AI-21 conformance suite before merging AI-33.5 |

---

## Acceptance criteria

1. `R-AIS-033` through `R-AIS-038` hold, each verified by its scenarios.
2. Every scenario marked `(pin)` cites the conformance or contract identifier it must not break, and the merged AI-33 work leaves the conformance suite green for both `R-CNF-011` and `R-CNF-012`.
3. The AI-33.3 wording matches `R-CNF-012`'s wording verbatim (no drift).
4. Every subnode carries at least one text-stream scenario AND at least one tool-call-stream scenario (charter line 1989).
5. No new Go identifier, type name, method name, package path, or signature is invented in this spec — only existing identifiers from `stream.go`, `capture.go`, `agenttest/stream_kit_leak.go`, `agenttest/stream_kit_record.go`, and the conformance suite are cited.
6. No `t.Parallel()` call appears in any AI-33 test file (`R-STK-008`).
7. `backend/agent/go.mod` is unchanged (`R-STK-009`).
8. `make test` from `backend/agent/` is green under `-race`; `make lint` is clean.
