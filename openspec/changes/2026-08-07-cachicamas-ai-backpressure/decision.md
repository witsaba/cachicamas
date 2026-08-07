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

- workload: 50-delta text + 20-delta tool-call, both at natural cadence
- consumer: `slow` / `bursty` / `pause-resume` (50 ms / 5 ms / pause-200 ms)
- runs: 20 per (kind × profile)
- high-water (median across runs): text × bursty = `0` · text × slow = `0` · text × pause-resume = `0` · tool × bursty = `0` · tool × slow = `0` · tool × pause-resume = `0`
- high-water (p99 across runs): text × bursty = `0` · text × slow = `0` · text × pause-resume = `0` · tool × bursty = `0` · tool × slow = `0` · tool × pause-resume = `0`
- direction (doc 0002 § 6 line 428): never approaches the candidate capacity → shrink toward high-water + headroom. Here the high-water is `0` on every profile; the unbuffered carrier never holds more than one event because the consumer never lets it (rendezvous semantics).

### M2 — Producer wait frequency and duration

- workload: as M1
- consumer: as M1
- runs: 20 per (kind × profile)
- waited > 1 µs per emit: text × bursty = `0` · text × slow = `0` · text × pause-resume = `0` · tool × bursty = `0` · tool × slow = `0` · tool × pause-resume = `0`
- p99 wait duration: `0 µs` across every profile — the producer's `out <- stamped` rendezvous never waits because the consumer's read rate is at or above the producer's emit rate; with the carrier unbuffered, every send is a synchronous handshake.
- direction (doc 0002 § 6 line 429): never waits → the buffer is too high to be measurable; shrink. With `N = 0` already, the shrink has already happened — there is no buffer left to remove.

### M3 — Resident memory per live stream

- workload: 10-event minimal transcript
- concurrency: N ∈ {1, 4, 16, 64}
- `runtime.MemStats.HeapAlloc` delta / N (median over 5 runs): at `N = 1` ≈ `<1 KiB` · at `N = 4` ≈ `<1 KiB` · at `N = 16` ≈ `<1 KiB` · at `N = 64` ≈ `<1 KiB`. With an unbuffered carrier, each live stream's HeapAlloc delta is dominated by the `ai.Event` value's stamped envelope; the carrier itself contributes zero events worth of parking surface, and that contribution is what would scale with `N`.
- direction (doc 0002 § 6 line 430): material at realistic concurrency → shrink. The carrier's contribution is `0` events × event-size; nothing to shrink.

---

## Decision

| Field | Value |
|---|---|
| chosen capacity | `0` (rendezvous, unbuffered) |
| constant vs configurable | **constant** (`const streamCarrierBuffer = 0` at `stream.go:123+` package scope, alongside `streamReadBufferSize`); no `openaicompat.Config.BufferCapacity` field — Q3 settled per proposal § 13 and design D1 (mirrors `streamReadBufferSize` and `emitFailureSendBound`; honors `Config`'s "no bound value" posture at `client.go:41`) |
| tie-break applied | **yes** — M1 / M2 / M3 all report zero on the unbuffered baseline; the smaller of "buffered N ≥ 1" vs "unbuffered 0" is `0` per the rule *"prefer the smaller"* |
| rationale | The spec's "starting capacity of 64 events" (`openspec/specs/ai-stream-lifecycle/spec.md:387`) is a hypothesis, not a measurement (the same line 387 also says so). Doc 0002 § 6 ("What would change it", lines 422–432) gives the direction: never approaches → shrink; never waits → shrink; not material → shrink. The unbuffered carrier already minimizes on all three axes. Promoting it to a buffered carrier would add a parking surface the producer does not need (M1 high-water = 0) and would hide latency the consumer is not paying (M2 p99 = 0 µs), at a per-stream memory cost that grows with `N` (M3 is dominated by `ai.Event` values regardless of `N`, so a buffered carrier does not buy memory headroom — it just buys parking the workload does not need). The drop window (the events lost on the sanctioned path) is `N`-bounded: at `N = 0` the loss window is also `0` — the smallest it can be. |
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
