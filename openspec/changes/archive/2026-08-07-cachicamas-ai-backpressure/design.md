# Design: Lock backpressure and buffer behavior (AI-34)

> **Change**: `cachicamas-ai-backpressure` · **Milestone**: AI-34 (doc 0002 lines 2041–2073, spec § 6 at `openspec/specs/ai-stream-lifecycle/spec.md:381–451`) · **Wave**: 5 — Harden · **Module**: `backend/agent/` (layered per ADR 0005 § D1 row 1 + § D3) · **Strategy**: single-pr · strict TDD on · stdlib-only · ~600 lines aggregate.

## Technical Approach

One-line production change at `backend/agent/src/ai/openaicompat/stream.go:223` plus a package-private constant declared alongside `streamReadBufferSize` at line 123. Two real-producer test files (`a_i-34_2_test.go`, `a_i-34_3_test.go`) prove the capacity is `cap(ch) == N` and that no unsanctioned loss path exists across slow / bursty / pause-resume consumer profiles. One real-producer test-only helper (`dripFramesServer`) is the only new code surface besides the const — it writes SSE frames one at a time with a configurable inter-frame gap, so AI-34.2 can drive a slow consumer against the **real** producer the way `slowSSEServer` (`stream_test.go:106`) already drives a stalled body against it. The 1-line change lands before AI-34.2's `cap(ch) == N` is RED-first against the new `N`. AI-34.1's `decision.md` records the workload, the numbers, the tie-break rule (doc 0002 line 432), and the chosen capacity — it is the source of truth the apply phase reads before the production edit. The conformance suite (`agenttest/conformance_cancellation.go:46–131`) is unchanged: R-CNF-011/012 prove drop physics, not capacity.

## Architecture Decisions

| # | Decision | Choice | Rejected alternative | Rationale |
|---|---|---|---|---|
| **D1** | Buffer constant — **location, name, visibility, type** | Package scope alongside `streamReadBufferSize` at `stream.go:123`; named `streamCarrierBuffer` (settled from proposal's `[proposed]`); package-private (lowercase); type `int` (event count). | Inline at line 223; `streamCarrierCapacity`; exported via `openaicompat.Config.BufferCapacity`; `uint` / named type. | Three package-private constants already cluster at lines 123–129 — grouping keeps the "constants first" idiom and deters future inlining at the single call site. The `stream<X><Buffer>` shape mirrors `streamReadBufferSize` at line 123 exactly; "capacity" is the spec's word but "buffer" is the local idiom, and `make(chan ai.Event, streamCarrierBuffer)` reads unambiguously. Q3 locked: package-private — mirrors `streamReadBufferSize` and `emitFailureSendBound`, and `Config`'s "no timeout, deadline or bound value" posture (`client.go:41`) forbids bound values in `Config` by design. `int` matches the spec's "64 events" framing (`spec.md:387`) and every slice/index local in this file (`nextWire int` at `bridge_test.go:133`, `wireIndex int` at `bridge_test.go:230`). |
| **D2** | `dripFramesServer` — **signature** | `func dripFramesServer(t *testing.T, frames []string, gap time.Duration) *httptest.Server` — handler writes each frame, flushes, sleeps `gap` between writes, selects on `r.Context().Done()` to abandon the drip on a cancelled request. | `(t, transcript string, gap)`; `(t, frames [][]byte, gap)`. | `frames []string` reads as "already-rendered SSE frames" and lets callers compose multi-frame transcripts by string concatenation, the way `ai33TextTranscript` already does (`a_i-33_3_test.go:114`). Matches `slowSSEServer(t, first string)` (`stream_test.go:106`) in shape; differs in pacing — `slowSSEServer` stalls once after a single flush (for cancellation tests); `dripFramesServer` paces frame-by-frame (for backpressure tests). |
| **D2** | `dripFramesServer` — **location + reach** | New file `backend/agent/src/ai/openaicompat/drip_frames_test.go` — internal `package openaicompat`. | Co-located inside `a_i-34_2_test.go`; under `backend/agent/src/agenttest/`. | `a_i-34_3_test.go` also needs the helper. A dedicated `_test.go` file makes the "shared test fixture" intent explicit; `agenttest/` is wrong because that package's doc pins it as dependency-free and stdlib-only (`stream_kit_leak.go:12–25`). Internal `package openaicompat` matches every AI-33 test file's posture (`a_i-33_3_test.go:86`, `bridge_test.go:43`) and lets the capacity assertion reference `streamCarrierBuffer` directly without an exported indirection. The credential-scan guard's scope is `package openaicompat_test` only (`stream.go:303–312`); internal placement does not break that guard. |
| **D3** | Measurement workload mechanics | Per explore § 4: M1 `len(out)` per emit, max across 20 runs / (kind, profile); M2 wait detection (>1 µs = waited), p99 wait duration; M3 `runtime.MemStats.HeapAlloc` delta / N concurrent streams, median over 5 runs. Workloads: 50-TextDelta text + 20-ToolCallDelta tool-call transcripts. Consumer profiles: 5 ms / 50 ms / pause-200 ms. | Benchmarks; tag-gated; -short gated. | **Workload host (build tag / `-short` / `-bench`) is Q2, deferred to `sdd-tasks`** — design shapes the workload mechanics, not the file/skip mechanism. The proposal binds the file to `measurement_test.go` in `package openaicompat`. |
| **D4** | Subnode order + PR shape | AI-34.1 first (measurement → `decision.md` → 1-line production change, in this order), AI-34.2 second (capacity + slow-consumer + `dripFramesServer` lands here), AI-34.3 third (exhaustive non-cancellation paths × leak-check × 50). One PR. | Split into 3 PRs; measurement behind a feature flag. | Single-PR pre-resolved at session preflight. Forecast ~600 lines aggregate; within 1000-line review budget. AI-34.2's `cap(ch) == N` is RED-first against the new `N`, so the 1-line change must land in the same PR before AI-34.2's test. |
| **D5** | `decision.md` shape | Verbatim per explore § 4.5: workload / measurements (M1 / M2 / M3 sub-tables) / decision (chosen `N`, constant vs configurable, tie-break applied yes/no). | Loose prose; spec-amendment-text inline. | Apply phase reads `decision.md` before the 1-line change — it must be unambiguous on `N`. Spec amendment (`spec.md:387` from "64 events" to "N events, per `decision.md`") is archive-phase, NOT this PR. |

## Data Flow

    M1/M2 instrumentation (AI-34.1 measurement_test.go)
    ─────────────────────────────────────────────────────
    dripFramesServer(frames, gap)   serveTranscripts (bridge_test.go:98)
            │                                │
            ▼                                ▼
    httptest.Server ──HTTP──→ *openaicompat.Client.Stream(ctx, req)
                                        │
                                        ▼
                       out := make(chan ai.Event, streamCarrierBuffer)
                                        │
                                        ▼
                            go run(ctx, resp, out)        (stream.go:224)
                                        │
                            per emit: len(out) capture (M1) +
                                       wait-time stamp  (M2)
                                        ▼
                       consumer (5 ms / 50 ms / pause-200 ms)
                                        ▼
                            DrainAndRecord → cap(ch)==N + ordered events

    AI-34.2 / AI-34.3 (real-producer test files)
    ───────────────────────────────────────────
    dripFramesServer ──→ *openaicompat.Client ──→ cap(ch)==streamCarrierBuffer
                                                ▼
                                      RequireNoGoroutineLeak × 50
                                                ▼
                                      DrainAndRecord → ordered events

## File Changes

| File | Action | Subnode | Description |
|---|---|---|---|
| `backend/agent/src/ai/openaicompat/stream.go` | Modify | 34.1 | Line 223: `make(chan ai.Event)` → `make(chan ai.Event, streamCarrierBuffer)`. **Constant declared at line 123+ scope** alongside `streamReadBufferSize` (not inlined at the call site). |
| `backend/agent/src/ai/openaicompat/drip_frames_test.go` | Create | 34.2 | ~20 lines. `package openaicompat`. `dripFramesServer(t, frames, gap)`. Internal test package so `a_i-34_2_test.go` and `a_i-34_3_test.go` reference it. |
| `backend/agent/src/ai/openaicompat/a_i-34_2_test.go` | Create | 34.2 | `cap(ch) == streamCarrierBuffer` RED-first against the real producer; slow-consumer ordering (5 ms / 50 ms / pause-200 ms drip); no-auxiliary-queue observation (capacity + producer-block sentinel). |
| `backend/agent/src/ai/openaicompat/a_i-34_3_test.go` | Create | 34.3 | 6-case matrix (slow / bursty / pause-resume × text / tool-call); each case wrapped in `agenttest.RequireNoGoroutineLeak`. Serial-only via `tb.Setenv("AGENTTEST_STREAM_KIT_LEAK_CHECK", "1")` (`stream_kit_leak.go:82, 110`). |
| `backend/agent/src/ai/openaicompat/measurement_test.go` | Create | 34.1 | M1 / M2 / M3 workload host. **Build tag / `-short` / `-bench` deferred to `sdd-tasks` (Q2)**. |
| `backend/agent/src/ai/openaicompat/a_i-33_3_test.go` | Modify | 34.1 | Line 28 doc-comment: `stream.go:209` → `stream.go:223`. 5-line edit, no RED/GREEN impact. |
| `openspec/changes/2026-08-07-cachicamas-ai-backpressure/decision.md` | Create | 34.1 | Workload + M1/M2/M3 sub-tables + chosen `N` + tie-break applied yes/no. **Apply phase reads this before the 1-line change.** |
| `openspec/specs/ai-stream-lifecycle/spec.md:387` | Modify (deferred) | archive | "starting capacity of 64 events" → "starting capacity of N events, per `decision.md`". **NOT this PR** — owned by AI-34's `sdd-archive` step. |

## Interfaces / Contracts

```go
// stream.go (package scope, near line 123)
const streamCarrierBuffer = <N>  // int; value frozen by decision.md

// drip_frames_test.go (package openaicompat, internal test pkg, ~20 lines)
func dripFramesServer(t *testing.T, frames []string, gap time.Duration) *httptest.Server
```

## Testing Strategy — strict TDD RED-GREEN-REFACTOR

| Subnode | RED test | GREEN |
|---|---|---|
| **34.1** | Author `decision.md` skeleton (workload, blank tables, blank chosen-N line). Author `measurement_test.go` with the three sub-tests; run → fail with producer still unbuffered. Author `a_i-33_3_test.go:28` doc-comment fix. | Run M1/M2/M3 (20/20/5 runs each). Fill `decision.md` with measured `N`. Edit `stream.go:223` to `make(chan ai.Event, streamCarrierBuffer)` where `streamCarrierBuffer = N`. Re-run `make test`. |
| **34.2** | Author `a_i-34_2_test.go`: `cap(ch) == streamCarrierBuffer` direct assertion → RED (unbuffered). Slow-consumer ordering test (drip `gap = 50 ms`, drain, assert every event in order) → RED. Pause-resume test (drip `gap = 200 ms` pause then resume) → RED. Author `drip_frames_test.go` as the helper. | Production change from 34.1 already landed; tests turn GREEN. |
| **34.3** | Author `a_i-34_3_test.go`: 6-case matrix (slow / bursty / pause-resume × text / tool-call) each wrapped in `RequireNoGoroutineLeak`. Run → RED if any path drops or leaks. | Tests turn GREEN against the buffered carrier (already shipped in 34.1). Confirm no `t.Parallel()` in this file (R-STK-008, `stream_kit_leak.go:94–106`). |

Shared helpers across all three: `serveTranscripts` (`bridge_test.go:98`), `mustClient` (`stream_test.go:50`), `validRequest` / `validToolCallRequest` (`stream_test.go:74`, `a_i-33_1_test.go:200`), `agenttest.DrainAndRecord` (`stream_kit_record.go:63`), `agenttest.RequireNoGoroutineLeak` (`stream_kit_leak.go:107`). **No new top-level Go dependency** — `backend/agent/go.mod` is unchanged (ADR 0005 § D1 row 1, § D3; AGENTS.md rule 5).

## Threat Matrix

N/A — no routing, shell, subprocess, VCS/PR automation, executable-file classification, or process-integration boundary. The only process boundary in the change is the existing `httptest.Server` shape `bridge_test.go` already uses.

## Migration / Rollout

No migration. The 1-line production change is revertible in isolation by re-setting `streamCarrierBuffer = 0` (rendezvous) at `stream.go:123` and reverting `stream.go:223` to `make(chan ai.Event)`. `decision.md` is preserved in the change folder as the historical record regardless. The spec amendment at `spec.md:387` is archive-phase and reverts independently.

## Open Questions

- **None blocking.** Q2 (workload host — build tag / `-short` / `-bench`) is deferred to `sdd-tasks`. The workload mechanics (M1/M2/M3, transcripts, consumer profiles) are settled here per explore § 4; only the file/skip mechanism is the sdd-tasks choice. If sdd-tasks picks `-short`, the deliverable shape is unchanged; if it picks `-bench`, the bench output is captured at PR-merge time and lifted verbatim into `decision.md`. Either way the production change reads `decision.md`'s chosen `N` before editing `stream.go:223`.
- **Future hardening (not this PR)**: if a real workload demands configurability later, promote `streamCarrierBuffer` to `openaicompat.Config.BufferCapacity` with a defaulting path. Named here so the path is not silent.
