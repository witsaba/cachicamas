# Tasks: Lock backpressure and buffer behavior (AI-34)

> **Change**: `cachicamas-ai-backpressure` · **Milestone**: AI-34 (doc 0002:2041–2073) · **Module**: `backend/agent/` · single-pr · strict TDD · stdlib-only · ~600 lines aggregate.

Subnode order in one PR: **AI-34.1** (measured buffer → 1-line production change + doc-fix) → **AI-34.2** (`dripFramesServer` + lossless ordering under pressure) → **AI-34.3** (exhaustive non-cancellation × leak-check × 50). Branch: `feat/ai-34-backpressure`.

## Review Workload Forecast

Decision needed before apply: No
Chained PRs recommended: No
Chain strategy: size:exception
400-line budget risk: Medium (outside 400, inside 1000 review budget)

## Q2 Decision (locked this phase)

**Workload host: (a) `//go:build measurement` build tag** at the top of `measurement_test.go`. Rationale: measurement runs are long (20× per (kind, profile), 5× per N), pollute the leak-check baseline (R-STK-008), and use `runtime.MemStats` deltas that conflict with `-race`. `Makefile` (`backend/agent/Makefile:91-92`) is `go test -race -v ./...` — no `-short` — so option (b) needs a Makefile edit; option (c) loses test-tracked assertions. Running: `go test -tags=measurement ./backend/agent/src/ai/openaicompat/...`.

## Subnode 34.1 — Measured buffer decision `[decision]`

- [x] T-34.1.1 — Author `openspec/changes/2026-08-07-cachicamas-ai-backpressure/decision.md` skeleton: workload (50-deltas text + 20-deltas tool-call × 5ms/50ms/pause-200ms), blank M1/M2/M3 sub-tables, blank Decision.
- [x] T-34.1.2 — Create `backend/agent/src/ai/openaicompat/measurement_test.go` (`//go:build measurement`, `package openaicompat`). M1 = `max len(out)`; M2 = wait count + p99; M3 = `HeapAlloc` delta / N (N ∈ {1,4,16,64}). Compiles under `-tags=measurement`.
- [x] T-34.1.3 — Run workload: 20× per (kind, profile) for M1/M2; 5× per N for M3. Fill `decision.md` sub-tables.
- [x] T-34.1.4 — Apply doc 0002:432 tie-break ("prefer smaller; backpressure observable > hidden latency"). Record chosen `N`. If tie-break lands at 0, record rationale.
- [x] T-34.1.5 (GREEN) — Edit `backend/agent/src/ai/openaicompat/stream.go`: add `const streamCarrierBuffer = N` near line 123; line 223 `make(chan ai.Event)` → `make(chan ai.Event, streamCarrierBuffer)`. `make test` green.
- [x] T-34.1.6 — 5-line doc-fix at `backend/agent/src/ai/openaicompat/a_i-33_3_test.go:28`: `stream.go:209` → `stream.go:223`. No RED/GREEN impact.

## Subnode 34.2 — Lossless ordering under pressure `[leaf]`

- [x] T-34.2.1 — Create `backend/agent/src/ai/openaicompat/drip_frames_test.go` (`package openaicompat`). `dripFramesServer(t, frames []string, gap time.Duration) *httptest.Server` writes each frame, flushes, sleeps `gap` (selects on `r.Context().Done()`).
- [x] T-34.2.2 (RED-first) — Create `backend/agent/src/ai/openaicompat/a_i-34_2_test.go` (`package openaicompat`). First test: `cap(ch) == streamCarrierBuffer` over real `*openaicompat.Client` + `httptest.Server`. RED-first: fails unbuffered; GREEN once T-34.1.5 lands.
- [x] T-34.2.3 — Confirm `cap(ch)` GREEN post T-34.1.5.
- [x] T-34.2.4 — Slow: `dripFramesServer(transcript, 50ms)`, drain to close. Every event in order; no terminal; no loss.
- [x] T-34.2.5 — Bursty: `dripFramesServer(transcript, 5ms)`.
- [x] T-34.2.6 — Pause-resume: `dripFramesServer(transcript, 200ms)`.
- [x] T-34.2.7 — No-auxiliary-queue: consumer paused at capacity; producer blocks on carrier; resume receives exactly carrier-held events.
- [x] T-34.2.8 (REFACTOR) — `make test` + `make lint` green; no `t.Parallel()` (R-STK-008).

## Subnode 34.3 — No unsanctioned loss path `[leaf]`

- [x] T-34.3.1 — Create `backend/agent/src/ai/openaicompat/a_i-34_3_test.go` (`package openaicompat`). Top: `tb.Setenv("AGENTTEST_STREAM_KIT_LEAK_CHECK", "1")` (R-STK-008). Reuse `dripFramesServer`.
- [x] T-34.3.2–T-34.3.7 — Six cases (slow/bursty/pause-resume × text/tool-call), each × 50 repeats wrapped in `agenttest.RequireNoGoroutineLeak(t, scenario)`.
- [x] T-34.3.8 (REFACTOR) — All 6 cases pass under `make test` (`-race`); 6 × 50 = 300 repeats within budget.

## Commit Shape (single PR, six commits per `work-unit-commits` skill)

1. `docs(ai): AI-34.1 decision.md skeleton` (T-34.1.1) · 2. `test(ai): measurement workload M1/M2/M3` (T-34.1.2–3) · 3. `feat(ai): apply streamCarrierBuffer = N` (T-34.1.4–5) · 4. `docs(ai): fix stream.go line-number drift in a_i-33_3_test.go:28` (T-34.1.6) · 5. `test(ai): dripFramesServer + lossless ordering under pressure` (T-34.2.1–7) · 6. `test(ai): exhaustive non-cancellation leak-check × 50` (T-34.3.1–8).

## Risk Mitigations (from design § Risks)

Doc-vs-code divergence → `decision.md` records Q1. Fake-vs-real asymmetry → `cap(ch)` on real producer only; conformance suite (`agenttest/conformance_cancellation.go:46–131`) untouched. Missing helper → `dripFramesServer` at T-34.2.1. Doc-comment drift → T-34.1.6. Tie-break at 0 → in scope at T-34.1.4.

## Spec Amendment & Test Budget

Archive phase owns `openspec/specs/ai-stream-lifecycle/spec.md:387` amendment: "64 events" → "N events, per `decision.md`". **NOT this PR.** Test gate: `cd backend/agent && make test` (`go test -race -v ./...`) GREEN at every commit; `make lint` GREEN at every commit. Workload: `go test -tags=measurement ./src/ai/openaicompat/...`.

## References (inline above)

Inputs: `openspec/changes/2026-08-07-cachicamas-ai-backpressure/{proposal.md, specs/ai-stream-lifecycle/spec.md, design.md}`; charter `docs/architecture/milestones/0002-cachicamas-ai-layer-1-task-graph.md:2041–2073, :432`; AI-33 pattern `a_i-33_{3,5a,5b}_test.go`; Engram `#2629–2632`.
