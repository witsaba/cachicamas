# AI-34 sdd-apply — apply-progress

> **Change**: `cachicamas-ai-backpressure` · **Milestone**: AI-34 (doc 0002:2041–2073) · **Wave**: 5 — Harden · **Module**: `backend/agent/` · **Strategy**: single-pr · **Strict TDD**: on · **Branch**: `feat/ai-34-backpressure` (pushed to origin).

This file mirrors the Engram observation `sdd/cachicamas-ai-backpressure/apply-progress` (#2635). The Engram copy is authoritative for SDD pipeline hand-off; this file is the on-disk record in the change folder.

## Status: success — 6/6 commits landed, branch pushed, all gates green

### Commits (oldest first)

| SHA | Type | Tasks | Subject |
|---|---|---|---|
| `a110e84` | docs | T-34.1.1 | `docs(ai): AI-34.1 decision.md skeleton` |
| `719489e` | test | T-34.1.2, T-34.1.3 | `test(ai): measurement workload M1/M2/M3 + decision.md numbers` |
| `3b7deb4` | feat | T-34.1.4, T-34.1.5 | `feat(ai): apply streamCarrierBuffer = 0` |
| `e621a84` | docs | T-34.1.6 | `docs(ai): fix stream.go line-number drift in a_i-33_3_test.go:28` |
| `9151234` | test | T-34.2.1–7 | `test(ai): dripFramesServer + lossless ordering under pressure` |
| `db768c7` | test | T-34.3.1–8 | `test(ai): exhaustive non-cancellation leak-check × 50` |

### Test gates (all green)

- `cd backend/agent && make test` → `ok agenttest 2.064s`, `ok src/ai 3.219s`, `ok src/ai/openaicompat 54.305s` (the 54s includes the 6 leak-check subtests × 50 repeats), and 3 other packages cached. No FAIL.
- `cd backend/agent && make lint` → `go vet ./...` + `golangci-lint run --config=.golangci.yml ./...` → `0 issues`.
- `cd backend/agent && go test -tags=measurement -v -run TestMeasurement_Workload_M1M2M3 ./src/ai/openaicompat/...` → `PASS: TestMeasurement_Workload_M1M2M3 (27.57s)`.

### Decision (`decision.md`)

- **Chosen capacity**: `N = 0` (rendezvous, unbuffered).
- **Constant vs configurable**: **constant** (`const streamCarrierBuffer = 0` at stream.go:123+, package-private).
- **Tie-break applied**: **yes** — M1 high-water = 0; M2 bursty p99 ≈ 750 µs would COLLAPSE to ~0 µs if buffered (hidden-latency the tie-break warns against); M3 per-stream heap is dominated by setup overhead, not the carrier. "Prefer the smaller" picks `0` over `N ≥ 1`.

### Files changed

| File | Action | Lines | Description |
|---|---|---|---|
| `openspec/changes/2026-08-07-cachicamas-ai-backpressure/decision.md` | Created | +96 | AI-34.1 measurement evidence + chosen capacity + tie-break reasoning |
| `backend/agent/src/ai/openaicompat/measurement_test.go` | Created | +438 | M1/M2/M3 workload host, `//go:build measurement` |
| `backend/agent/src/ai/openaicompat/stream.go` | Modified | +15/−1 | `streamCarrierBuffer = 0` const at :123+; line 223 `make(chan ai.Event)` → `make(chan ai.Event, streamCarrierBuffer)` |
| `backend/agent/src/ai/openaicompat/a_i-33_3_test.go` | Modified | +1/−1 | Doc-comment line-number drift fix |
| `backend/agent/src/ai/openaicompat/drip_frames_test.go` | Created | +80 | `dripFramesServer(t, frames, gap)` — only new helper AI-34 introduces |
| `backend/agent/src/ai/openaicompat/a_i-34_2_test.go` | Created | +338 | cap(ch) == N + slow/bursty/pause-resume ordering + no-auxiliary-queue |
| `backend/agent/src/ai/openaicompat/a_i-34_3_test.go` | Created | +208 | Exhaustive (profile × kind) × 50 repeats leak-check |

**Aggregate diff**: +1176 / −2 lines. `size:exception` was accepted preflight (Engram obs #2634); the review budget is 1000 changed lines per preflight, and the actual is over by ~176 lines. The over-budget is dominated by `measurement_test.go`'s full M1/M2/M3 instrumentation (438 vs forecast ~50) and `a_i-34_2_test.go`'s helper functions (338 vs forecast ~200).

### Hard-rule compliance

- ✅ `streamCarrierBuffer` is a `const` (not a `var`) at package scope, alongside `streamReadBufferSize`. Name and location locked per design D1; value locked per decision.md tie-break.
- ✅ No new dependencies. `backend/agent/go.mod` unchanged.
- ✅ Conformance suite (`backend/agent/src/agenttest/conformance_cancellation.go:46–131`) untouched.
- ✅ Spec amendment `openspec/specs/ai-stream-lifecycle/spec.md:387` NOT modified — archive-phase per proposal § "Spec amendment follow-up".
- ✅ Build tag uses modern syntax `//go:build measurement` (Go 1.18+).
- ✅ No `t.Parallel()` in `a_i-34_3_test.go` (R-STK-008). All 6 leak-check scenarios run serially.
- ✅ No `Co-Authored-By` trailer in any commit.
- ✅ Conventional Commits format on all 6 commits.

### Status

**22/22 tasks complete. Ready for `sdd-verify`.**

## Key Learnings

1. The chosen `N=0` (rendezvous) emerges from measurement, not preference — M2's bursty p99 collapse to 0 µs at buffered `N` is exactly the hidden-latency the doc 0002 tie-break rule warns against.
2. Building `httptest.Server` INSIDE the `RequireNoGoroutineLeak` closure (not outside) is the load-bearing detail — outside the closure, keep-alive listener goroutines inflate the leak amplitude past `leakTolerance=25`.
3. The M3 per-stream heap DECREASES with N (78 → 5 KiB across N=1 → N=64), meaning the headline is dominated by one-time setup cost — the carrier's own contribution is `0` bytes at `N=0`.
