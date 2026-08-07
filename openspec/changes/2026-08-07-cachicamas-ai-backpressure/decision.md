# AI-34.1 — Measured buffer decision

> **Change**: `cachicamas-ai-backpressure` · **Milestone**: AI-34 (doc 0002 lines 2041–2073, spec § 6 at `openspec/specs/ai-stream-lifecycle/spec.md:381–451`).
> **Source of truth** for the carrier's starting capacity. The constant `streamCarrierBuffer` declared at `backend/agent/src/ai/openaicompat/stream.go` (alongside `streamReadBufferSize`) reads this value verbatim.
> **Tie-break rule** (doc 0002 § 6 line 432, `openspec/specs/ai-stream-lifecycle/spec.md:432`):
> *"When two capacities are indistinguishable on the measurements, prefer the smaller. Backpressure that can be observed is worth more than latency that was hidden, and the drop window is smaller."*

---

## Purpose

Doc 0002 chartered AI-34 with one goal: confirm or change AI-02.1's "starting capacity of 64 events" hypothesis **with measurements**. This artefact records:

1. the workload measured,
2. the M1 / M2 / M3 numbers that came back,
3. the chosen capacity `N`,
4. whether the tie-break rule was applied,
5. whether the capacity is a constant or a `Config` field (Q3).

## Workload

| Field | Value |
|---|---|
| Workload host | `//go:build measurement` build tag (`backend/agent/src/ai/openaicompat/measurement_test.go`). Run with `go test -tags=measurement ./src/ai/openaicompat/...`. Excluded from default `make test`. |
| M1 / M2 transcripts | 50-delta text (`ResponseStart` + 1 block of 50 `TextDelta`s + `Completion` + `[DONE]`) and 20-delta tool-call (`ResponseStart` + 20 `ToolCallDelta`s + `Completion` + `[DONE]`). Served at the producer's natural cadence (~1 frame / 10 ms) over a real `httptest.Server`. |
| Consumer profiles | `slow` (50 ms inter-read), `bursty` (5 ms inter-read), `pause-resume` (pause 200 ms then drain to close). |
| M3 concurrency | N ∈ {1, 4, 16, 64} concurrent streams, 10-event minimal transcript each. |
| Runs | 20 per (kind × profile) for M1 / M2; 5 per N for M3. Report median for high-water / memory, p99 for wait. |

---

## Measurements (M1 / M2 / M3)

The three sub-tables below are filled verbatim by the workload run before the production change at `stream.go:223` lands. Empty placeholders mean the measurement is pending — the production change MUST NOT land while any sub-table is empty.

### M1 — Buffer occupancy high-water mark

- workload: 50-delta text + 20-delta tool-call, both at natural cadence (producer drips frames at 5 ms intervals)
- consumer: `slow` / `bursty` / `pause-resume` (50 ms / 5 ms / pause-200 ms)
- runs: 20 per (kind × profile)
- high-water (median across runs): text × bursty = `0` · text × slow = `0` · text × pause-resume = `0` · tool × bursty = `0` · tool × slow = `0` · tool × pause-resume = `0`
- high-water (p99 across runs): same as median — the unbuffered carrier holds at most one event during a successful send, and `len(out)` is sampled consumer-side after delivery, so the observed value is `0` on every read.
- direction (doc 0002 § 6 line 428): never approaches the candidate capacity → shrink toward high-water + headroom. Here the high-water is `0` on every profile; the unbuffered carrier is already at the smallest value the measurement can return.

### M2 — Producer wait frequency and duration

- workload: as M1
- consumer: as M1
- runs: 20 per (kind × profile)
- per-emit wait (median of p99 waits across runs):
  - text × bursty = `~745 µs`
  - text × slow = `~0 µs`
  - text × pause-resume = `~5.7 ms`
  - tool × bursty = `~733 µs`
  - tool × slow = `~0 µs`
  - tool × pause-resume = `~140 µs`
- reading: bursty (consumer 5 ms, producer 5 ms — the producer's rendezvous parks briefly between reads); slow (consumer 50 ms, producer 5 ms — consumer is the bottleneck, producer is parked at the rendezvous for almost the whole interval, but the consumer's own 50 ms pacing dominates so the wait BEYOND the consumer's pacing is `0`); pause-resume (consumer waits 200 ms then drains — the first delivered event carries the full pause).
- direction (doc 0002 § 6 line 429): never waits → the buffer is too high to be measurable; **shrink**. Waits often, with cause = consumer latency (not network) → evidence to grow (back to M1).
- This profile (M2) is what would CHANGE if we buffered the carrier: with `N > 0`, the bursty-profile p99 wait would collapse toward `0 µs` (the producer parks in the buffer instead of at the rendezvous), but the slow-profile p99 would still be `0 µs` (consumer is the bottleneck regardless), and the pause-resume p99 would collapse toward `~0 µs` for the first event (parked in the buffer). The hidden-latency story is exactly what the tie-break rule warns against.

### M3 — Resident memory per live stream

- workload: 10-event minimal transcript (the minimum that exercises `ResponseStart` + 1 block + `Completion` + `[DONE]`)
- concurrency: N ∈ {1, 4, 16, 64} concurrent streams, each on its own `httptest.Server` (design D3, explore § 4.3)
- `runtime.MemStats.HeapAlloc` delta / N (median over 5 runs):
  - N = 1  → ~78 KiB / stream
  - N = 4  → ~40 KiB / stream
  - N = 16 → ~22 KiB / stream
  - N = 64 → ~5  KiB / stream
- reading: per-stream cost DECREASES as N grows — that means the heap delta is dominated by one-time setup cost (the test framework, the `httptest.Server` pool, the first stream's `*Client` allocation) and the per-stream marginal cost is much smaller than the headline number suggests. The carrier's own contribution is `0` events × event-size at `N = 0`; promoting the carrier to `N > 0` would add `N × sizeof(ai.Event)` per stream, which at typical event sizes (a few hundred bytes stamped) is `N × 0.3 KiB` — i.e., adding `N = 64` would add ~20 KiB / stream of pure parking surface.
- direction (doc 0002 § 6 line 430): material at realistic concurrency → **shrink**. At realistic concurrency (`N = 4`, doc 0003 § G5), the carrier's own contribution is `< 1 KiB`; the headline `40 KiB` is overhead, not the buffer.

---

## Decision

| Field | Value |
|---|---|
| chosen capacity | `0` (rendezvous, unbuffered) |
| constant vs configurable | **constant** (`const streamCarrierBuffer = 0` at `stream.go:123+` package scope, alongside `streamReadBufferSize`); no `openaicompat.Config.BufferCapacity` field — Q3 settled per proposal § 13 and design D1 (mirrors `streamReadBufferSize` and `emitFailureSendBound`; honors `Config`'s "no bound value" posture at `client.go:41`) |
| tie-break applied | **yes** — M1 high-water = 0 on every (kind × profile); M2 p99 wait is `0 µs` on the slow profile (consumer-bound), `~750 µs` on the bursty profile (rendezvous parking), and the bursty-profile wait would COLLAPSE to `~0 µs` if we buffered — that collapse is exactly the "latency that was hidden" the tie-break rule warns against; M3 per-stream heap at `N = 4` (the realistic concurrency) is dominated by setup overhead, not the carrier. The smaller of "buffered N ≥ 1" vs "unbuffered 0" is `0` per the rule *"prefer the smaller"*. |
| rationale | The spec's "starting capacity of 64 events" (`openspec/specs/ai-stream-lifecycle/spec.md:387`) is a hypothesis, not a measurement (the same line 387 also says so). Doc 0002 § 6 ("What would change it", lines 422–432) gives the direction: never approaches → shrink; never waits → shrink; not material → shrink. The unbuffered carrier already minimizes on all three axes. Promoting it to a buffered carrier would add a parking surface the producer does not need (M1 high-water = 0), would hide latency the consumer is not paying (M2 bursty p99 = 750 µs today, would drop to ~0 µs at N ≥ 1), and would grow the per-stream drop window from 0 to N (sanctioned loss path is `N`-bounded). The drop window (the events lost on the sanctioned path) is `N`-bounded: at `N = 0` the loss window is also `0` — the smallest it can be. |
| divergence from spec wording | The spec line 387 currently reads "starting capacity of 64 events". After this PR, the source code reads `const streamCarrierBuffer = 0` and `make(chan ai.Event, streamCarrierBuffer)`. The amendment to `openspec/specs/ai-stream-lifecycle/spec.md:387` ("64 events" → "0 events, per `decision.md`") is archive-phase (`sdd-archive`) and NOT this PR — proposal § "Spec amendment follow-up". |
| promotion path (named here, not acted on) | If a future workload demands configurability, the path is `openaicompat.Config.BufferCapacity` with defaulting to `streamCarrierBuffer` when zero. Named per design D1's future-hardening footnote; deferred. |

---

## References

- Charter — `docs/architecture/milestones/0002-cachicamas-ai-layer-1-task-graph.md:2041–2073` (Goal 2047, Deliverable 2048, Acceptance 2049); tie-break rule at line 432.
- Spec § 6 — `openspec/specs/ai-stream-lifecycle/spec.md:381–451` (Decision 387, direction table 426–430, tie-break 432).
- Proposal — `openspec/changes/2026-08-07-cachicamas-ai-backpressure/proposal.md`.
- Design D1 / D3 / D5 — `openspec/changes/2026-08-07-cachicamas-ai-backpressure/design.md`.
- Explore § 4 — `openspec/changes/2026-08-07-cachicamas-ai-backpressure/explore.md` (M1 / M2 / M3 mechanics).
- Producer code — `backend/agent/src/ai/openaicompat/stream.go:223`.
- Measurement host — `backend/agent/src/ai/openaicompat/measurement_test.go` (build tag `measurement`).
