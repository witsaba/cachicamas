# Proposal — `cachicamas-ai-cancellation` (AI-33)

> **Outcome**: Prove Layer 1's cancellation contract over a real HTTP transport. Five subnodes. Every scenario runs over text **and** tool-call streams. Built on the seams the explore already named.
>
> **One-line**: Wire contract is closed (AI-28 / AI-30 / AI-32); cancellation contract is the next gap; AI-33 closes it with one test file per cancellation moment plus a unifying resource-discipline pass.

## 1. Identity

| Field | Value |
| --- | --- |
| Change | `cachicamas-ai-cancellation` |
| Milestone | AI-33 — Prove cancellation and goroutine cleanup (doc 0002) |
| Wave | Wave 5 — Harden |
| Branch | `feat/ai-33-cancellation` |
| Worktree | `cachicamas-worktrees/ai-33` |
| Base | `main` @ `e9a8054` |
| Depends on | AI-28 (text events), AI-30 (tool-call events), AI-32 (incl. AI-32.3 bounded-wait S-AEM-051/052), AI-22.4 (leak-check mechanism) |
| Blocks | AI-34 (backpressure), AI-35 (retry / idempotency) |
| Module | `backend/agent` — **layered**, not hexagonal ([ADR 0005 § D1](../../../docs/adr/0005-promote-agent-stack-to-own-module.md#d1--dependency-rule-v2)) |
| Test runner | `cd backend/agent && make test` (runs `go test -race -v ./...`) |
| Discoverability | `openspec/changes/cachicamas-ai-cancellation/explore.md` (229 lines, 8 sections) |
| Charter | [`docs/architecture/milestones/0002-cachicamas-ai-layer-1-task-graph.md:1987–2037`](../../../docs/architecture/milestones/0002-cachicamas-ai-layer-1-task-graph.md) |

## 2. Intent

The Layer 1 wire contract is closed at AI-32; what remains is the **cancellation contract**. AI-33 proves four cancellation moments — before headers, between frames, during a blocked send, after completion — against a real HTTP transport, plus a cross-cutting resource-discipline pass asserting no goroutine or connection leaks across any exit path. The milestone produces **proof, not new behavior**: every test asserts a property the merged code already implements, plus a single surgical drain-before-close extension to `run()`'s defer chain (mirroring `capture.go:117–122`).

## 3. Scope

### 3.1 In scope

- Five subnodes: **AI-33.1** (pre-headers), **AI-33.2** (between frames), **AI-33.3** (blocked send), **AI-33.4** (after completion), **AI-33.5** (resource discipline).
- One test file per subnode against a real `*openaicompat.Client` over `httptest.NewServer`.
- Tool-call-stream variant for every scenario (charter line 1989).
- One surgical code change: drain-before-close in `run()`'s defer chain (`stream.go:344–345`), mirroring `capture.go:117–122`.
- A full-package leak-check wrapper for AI-33.5 (serial, mirrors `bridge_test.go` posture).

### 3.2 Out of scope

| Item | Owner |
| --- | --- |
| Layer 2+ stop semantics | Layer 2 / AI-34, AI-35 |
| Backpressure (buffer capacity, V-STR-08 falsification) | AI-34 |
| Retry / idempotency | AI-35 |
| Conformance suite content | unchanged — R-CNF-011 / R-CNF-012 already prove the abstract physics over the fake provider |
| AI-32.3 typed-failure semantics | unchanged — AI-33 proves it, doesn't rewrite it |
| New top-level Go dependencies | blocked — `RequireNoGoroutineLeak`'s stdlib-only posture (R-STK-009) is the contract |

## 4. Approach

**Strategy**: prove the contract end-to-end over a real `*http.Response` using the seams already shipped in `backend/agent/src/agenttest/`. Each AI-33.x subnode writes ONE failing test against `httptest.NewServer`, drives it GREEN against the merged producer, then asserts no goroutine leak. AI-33.5's resource-discipline pass extends `run()`'s defer chain with `io.Copy(io.Discard, resp.Body)` before the existing `defer resp.Body.Close()`, then runs a serial full-package leak check over every exit path on both stream kinds.

**Test seams already shipped** (cited from explore § 3 — reuse, do not re-author):

| Seam | Use for |
| --- | --- |
| `RequireNoGoroutineLeak(tb, scenario)` (`stream_kit_leak.go:107`) | Every AI-33 subnode |
| `DrainAndRecord(tb, ch, timeout)` (`stream_kit_record.go:63`) | AI-33.1 / 33.2 / 33.4 (drainable `out`) |
| `bridgeServeTranscripts(transcripts)` (`openrouter/conformance/bridge_test.go:141–158`) | AI-33.3 / 33.4 over text + tool-call bytes |
| `newDripHandler(chunks, interval)` (`timeout_test.go:20–39`) | AI-33.2 (idle-between-frames) |
| `contextBeforeFirstFrameServer(t)` (`stream_failure_test.go:391–405`) | AI-33.1 (cancel-before-Do variant) |
| `requireDrainedKinds` / `checkLifecyclePrefix` (`conformance_lifecycle.go:49, 70`) | Kind-level assertions across all five |

**Architectural foundation** (already in `main`; do not re-prove):

- Single producer goroutine at `stream.go:343`; `defer close(out)` at `:344` is the unique closing site (R-ATS-003, R-ATS-005).
- AI-32.3 bounded-wait with `emitFailureSendBound = 5s` (`stream.go:115`, `:437–470`, `:503–526`) — the seam AI-33.3 rides on.
- `capture.go:117–122`'s drain-before-close idiom — the pattern AI-33.5 extends into `run()`.

## 5. Recommended subtask order

Per explore § 6.2. Land leaves in this order so the most empirical uncertainty (AI-33.2) surfaces before the most politically charged (AI-33.3).

| # | Node | Depends on | Why this slot | Δ-line forecast |
| --- | --- | --- | --- | --- |
| 1 | **AI-33.1** cancel before headers | AI-22.4 | Smallest scope; already implemented (`stream.go:181–183`, `:217–230`, `:241–251`). Gap = real-HTTP proof only. Establishes `httptest.NewServer` + `contextBeforeFirstFrameServer` shape reused by the others. | ~150 |
| 2 | **AI-33.4** cancel after completion | AI-22.4 | Trivially-correct. Establishes race-detector posture and "exactly one close" assertion reuse for 33.2 / 33.3. | ~150 |
| 3 | **AI-33.2** cancel between frames | AI-22.4 | Most empirical (explore § 5.1). RED may force a `Body.Close()` watcher or ctx-aware wrap. Land before 33.3 so any surprise surfaces before the doc-text reconciliation. | ~200 |
| 4 | **AI-33.3** cancel during a blocked send | AI-22.4 | Carries the BLOCKING doc-text reconciliation (§ 7 below). Land after 33.2 so the read-loop behavior is settled. | ~200 |
| 5 | **AI-33.5** resource discipline | 33.1 … 33.4 | Drain in `run()` + full-package leak check. Most likely subnode to exceed 400 lines; chain to **33.5a** (drain impl) / **33.5b** (full-suite leak check) if so (explore § 6.3). | ~300 |

**Edge caveat** — if AI-33.2 RED forces a producer-shape change, the spec phase must amend doc 0002 under its living-graph clause. Detect at RED, amend in the same PR per doc 0002's `Living graph` rule.

## 6. Architecture decisions

### 6.1 Doc-text ↔ merged-code reconciliation (already mostly resolved)

Doc 0002 line 2023 says AI-33.3's outcome is *"stream closed without a terminal event — the sanctioned loss path of AI-20.3"*. AI-32.3's bounded-wait (now merged at `stream.go:437–470` and `:503–526`) emits a **typed terminal with bounded wait** when a live consumer is cancelled mid-send. The two are reconcilable because the existing live contract — `openspec/specs/ai-stream-lifecycle/spec.md` § 5 — already commits to **bounded close + abandoned-then-cancelled = no observable terminal**, and AI-32.3's `emitFailureSendBound = 5s` IS the bounded close. AI-33.3's spec wording will be pinned to the abandoned-then-cancelled framing, matching conformance R-CNF-012 verbatim (see Open decision § 7).

### 6.2 Producer's defer chain extension (AI-33.5)

`capture.go:117–122` proves the idiom; extending it into `run()` is a one-statement addition before the existing `defer resp.Body.Close()` at `stream.go:345`. **No new helper type, no new goroutine, no signature change.** The model stays "single producer" (R-ATS-003).

### 6.3 Read-loop ctx-awareness (AI-33.2 — empirical)

`stream.go:369–368`'s read loop is not ctx-aware; it depends on transport-level cancellation propagating through `Body.Read`. If the RED test shows the dependency does not hold for `httptest`-served HTTP/1.1 within `DefaultDrainTimeout + safety margin`, the implementation change is bounded: a `Body.Close()` watcher goroutine OR a ctx-aware wrapper. Either move MUST NOT introduce a persistent second goroutine (R-ATS-003) — if it must, the leak-check arithmetic is updated to count it, and doc 0002 gets a `[decision]` amendment.

## 7. Open decisions

| # | Decision | Default | Flip cost |
| --- | --- | --- | --- |
| 1 | **AI-33.3 scenario scope**: "truly-abandoned consumer (no reads)" vs "any saturated `out`"? | **Truly-abandoned** — pins to conformance R-CNF-012 wording verbatim; matches `ai-stream-lifecycle` § 5's `V-STR-07` abandonment definition and line 254's "abandoned-then-cancelled path is testable". | Flip to "saturated" = either re-assert "bounded-wait OR empty" or revert AI-32.3's typed-failure emission (S-AEM-051/052). Both are non-trivial doc amendments. |

This decision **binds the spec phase** once the proposal is approved.

## 8. Capabilities

| Type | Capability | Notes |
| --- | --- | --- |
| **New** | None | AI-33 adds no new public API; it proves existing behavior over a real HTTP transport. |
| **Modified** | `ai-stream-lifecycle` (likely) | AI-33.5's drain-before-close adds a body-lifecycle obligation the contract doesn't currently state. Spec phase writes a delta spec amending § 4 (`V-STR-05` family) per the contract's amendment rule (line 30–41). Decision finalised in the spec phase. |

## 9. Affected areas

| Area | Impact | Description |
| --- | --- | --- |
| `backend/agent/src/ai/openaicompat/stream.go` | Modified (likely, 1 line) | AI-33.5 adds `io.Copy(io.Discard, resp.Body)` before `defer resp.Body.Close()` at `:345`. |
| `backend/agent/src/ai/openaicompat/capture.go` | Unchanged (reference) | Source of the drain idiom (`:117–122`). |
| `backend/agent/src/ai/openaicompat/{stream,terminal,stream_failure}_test.go` family | Unchanged (reference) | Test seams reused. |
| `backend/agent/src/ai/openaicompat/a_i-33_{1..5}_test.go` | **New** | One test file per subnode; `package openaicompat` (internal) for 33.1–33.4, `package openaicompat_test` (external) for 33.5 (mirrors `bridge_test.go`). |
| `backend/agent/src/agenttest/` | Unchanged | `RequireNoGoroutineLeak` + `DrainAndRecord` reused as-is. |
| `openspec/specs/ai-stream-lifecycle/spec.md` | Modified (likely) | Delta spec amending § 4 with body-lifecycle clause, per the contract's amendment rule. |
| `openspec/specs/ai-provider-conformance-suite/spec.md` | Unchanged | Conformance suite is unchanged — R-CNF-011 / R-CNF-012 already prove the abstract physics over the fake. |

## 10. Risks

Carried from explore § 5 with one addition (R6).

| # | Risk | Likelihood | Mitigation |
| --- | --- | --- | --- |
| R1 | AI-33.2 RED forces an implementation change (watcher or ctx-aware wrap), inflating AI-33.5's leak-check arithmetic. | Med | RED-first per `openspec/AGENTS.md`. Amend doc 0002 in the same PR per living-graph clause. |
| R2 | AI-33.3 spec wording drifts from `ai-stream-lifecycle` § 5 / R-CNF-012. | Low (now) | Open decision § 7 pins it; spec phase cannot reopen. |
| R3 | New top-level Go dep sneaks in (e.g., a leak-check library). | Low | `openspec/AGENTS.md` rule 5 + R-STK-009. `RequireNoGoroutineLeak`'s stdlib-only posture is canonical. |
| R4 | Parallel test posture leaks into AI-33.5's full-package suite, breaking `RequireNoGoroutineLeak`'s contract. | Med | Explicitly serial suite wrapper. |
| R5 | Tool-call stream variant ignored on a subnode. | Med | Charter line 1989 binds; spec phase must include tool-call variant per leaf. |
| R6 (new) | AI-33.5's drain on a slow-but-alive consumer reads more bytes than `ai-fake-provider` expects, perturbing AI-21's scripted scenarios. | Low | Drain runs on `resp.Body` (HTTP layer), not on `ai.Event`s (application layer). Verify with the full AI-21 conformance run before merging AI-33.5. |

## 11. Rollback plan

This is a **proof milestone**, not a behavior milestone. Rollback is surgical.

- **Tests-only rollback** (AI-33.1–33.4): revert the new `a_i-33_{1..4}_test.go` files. No production code touched. The conformance suite is unchanged.
- **AI-33.5 rollback**: revert `a_i-33_5_test.go` **AND** revert the single added line in `run()`'s defer chain (`stream.go:345`). `capture.go`'s drain idiom is independent and stays.
- **If AI-33.2's RED forced a watcher-goroutine change**: rollback reverts both the test file and the watcher; the producer reverts to its single-goroutine shape (R-ATS-003 invariant restored).

All five subnodes ship as separate PRs; each PR's revert is its own rollback unit. No cross-PR state to unwind.

## 12. Success criteria

- [ ] AI-33.1 RED fails on `main`; GREEN on this branch; no leak; passes under `-race`.
- [ ] AI-33.2 GREEN with bounded close ≤ `DefaultDrainTimeout + safety`; no leak; passes under `-race`.
- [ ] AI-33.3 GREEN with conformance-R-CNF-012 wording verbatim.
- [ ] AI-33.4 race-detector coverage over 50+ repeats shows no panic, exactly one close.
- [ ] AI-33.5 drain helper present; full-package leak check passes on text + tool-call streams.
- [ ] `make test` from `backend/agent/` is green.
- [ ] `make lint` is green.
- [ ] No new top-level Go dep in `backend/agent/go.mod`.
- [ ] Each subnode ships as a separate PR, chained if any exceeds the 400-line review budget.

## 13. Dependencies (charter-level)

| Source | Why |
| --- | --- |
| AI-28 — text events | Text-stream scenarios require the text event chain (already merged). |
| AI-30 — tool-call events | Tool-call scenarios require the tool-call event chain (already merged). |
| AI-32 (incl. AI-32.3) — mid-stream failure + bounded-wait | The `emitFailureSendBound` seam at `stream.go:115` is what AI-33.3 rides on. |
| AI-22.4 — leak-check mechanism | `RequireNoGoroutineLeak` (`stream_kit_leak.go:107`) is the contract for leak assertions. |
| `agenttest/stream_kit_record.go` (`DrainAndRecord`) | Bounded drain for subnodes 33.1 / 33.2 / 33.4. |
| `openspec/specs/ai-stream-lifecycle/spec.md` § 5 | The live contract this milestone proves; § 7 (Open decision) pins wording to it. |

## 14. Review budget forecast

All five subnodes under 400 changed lines if seams are reused as planned (explore § 6.3). AI-33.5 is the only likely chain candidate:

- **AI-33.5a** — drain implementation (~80 lines, surgical change to `stream.go:345` + a focused test).
- **AI-33.5b** — full-suite leak check over every exit path on both stream kinds (~220 lines).

Chain only if AI-33.5's aggregate exceeds 400.

---

## 15. Next step

Hand off to `sdd-spec` for `openspec/changes/cachicamas-ai-cancellation/specs/`. The spec phase:

1. Confirms Open decision § 7 (default = truly-abandoned).
2. Writes delta specs for `ai-stream-lifecycle` (body lifecycle, AI-33.5) and per-subnode Given/When/Then scenarios over text **and** tool-call streams.
3. Marks every behavior leaf `(pin) *(regression)*` for the conformance assertions it must not break.
