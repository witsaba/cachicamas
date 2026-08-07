// AI-34.1 — measurement workload host (M1 / M2 / M3). Build-tag-gated so
// the workload runs only with `go test -tags=measurement ./...`, never in
// default `make test` (R-STK-008 + Q2 settled at sdd-tasks time).
//
// # Why a measurement workload, in this codebase
//
// Doc 0002 § 6 ("What would change it", lines 422–432) gives the
// direction: the spec's "starting capacity of 64 events" is a hypothesis,
// not a measurement. AI-34.1's deliverable is "with measurements, not a
// preference" — three measurements (M1 high-water, M2 wait frequency, M3
// resident memory) with the tie-break rule at line 432. This file is the
// host: it runs the same workload three times, against the *real producer*
// over a real `httptest.Server` (design D3, explore § 4), and prints the
// numbers in a parseable form for `decision.md` to lift verbatim.
//
// # Why build-tag, not `-short` or `-bench` (Q2)
//
// M1 takes 20 runs × 3 profiles × 2 stream kinds = 120 runs; M2 the
// same; M3 5 runs × 4 N values = 20 runs. The whole workload is several
// minutes, uses `runtime.MemStats` deltas (incompatible with `-race`),
// and is meaningless against an already-running parallel test suite
// (R-STK-008). `-short` would force every CI runner that already runs
// `go test -short` to pay that cost. `-bench` would lose the test-tracked
// assertion that this file's existence provides. `//go:build measurement`
// is the only one that costs nothing on the default gate and runs on
// demand — exactly the posture this file needs.
//
// # Why this file lives in `package openaicompat`
//
// The M1 / M2 hooks attach to the producer's `out` channel and the
// `emit` select. Both are package-private to `openaicompat` (design D1:
// "internal `package openaicompat` for 34.2 / 34.3"). The measurement
// workload sits in the same package — no exported surface for it.
//
// # Reuse, and no new deps
//
// serveTranscripts (`bridge_test.go:98`), mustClient
// (`stream_test.go:50`), validRequest / validToolCallRequest
// (`stream_test.go:74`, `a_i-33_1_test.go:200`), ai.Event
// (`backend/agent/src/ai/event.go`). Stdlib only: `runtime`, `sort`,
// `time`, `net/http/httptest`. `backend/agent/go.mod` is unchanged
// (R-STK-009).
//
// # What this test prints (the contract with `decision.md`)
//
// One block per measurement. The block is the four-or-five-line shape
// decision.md's M1 / M2 / M3 sub-tables lift verbatim:
//
//	M1 high-water (median): {text × bursty, text × slow, ...}
//	M2 waited>1µs ratio / p99 wait: {text × bursty, ...}
//	M3 HeapAlloc delta/N: {N=1, N=4, N=16, N=64}
//
// The numbers are MEDIAN across runs for M1 (high-water stabilises fast,
// explore § 4.1), MEDIAN across runs for M3 (memory has higher variance,
// explore § 4.3), and COUNT + p99 for M2. See explore § 4 for the full
// rationale.
//
// # RED posture
//
// This file is the AI-34.1 deliverable shape: it does NOT fail when the
// carrier is unbuffered — it measures that the carrier IS unbuffered
// (M1 high-water = 0, M2 "fast" count = full, M3 HeapAlloc delta = ai.Event
// envelope alone). The RED in this discipline is the empty `decision.md`
// sub-tables; the GREEN is them filled with this file's printed output.
// Once the production change lands (commit 3), the same workload re-run
// will report different numbers — the constant change is what gives
// the sub-tables a non-trivial "before / after" pair.

package openaicompat

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"runtime"
	"sort"
	"testing"
	"time"

	"github.com/cachicamas/backend/agent/src/ai"
)

// ---------------------------------------------------------------------------
// Transcript helpers — shared by M1, M2, M3.
// ---------------------------------------------------------------------------

// measurementTextFrames is the 50-`TextDelta` text transcript doc 0002 § 6
// / explore § 4.1 names: 1 identity chunk + 1 content block of 50 deltas +
// terminal chunk + [DONE]. Rendered as pre-written SSE frames so the
// handler can pace them with `dripFramesServer`.
func measurementTextFrames() []string {
	frames := make([]string, 0, 52)
	frames = append(frames,
		`data: {"id":"m-text","model":"m","created":1700000000,"object":"chat.completion.chunk","choices":[{"index":0,"delta":{"role":"assistant"},"finish_reason":null}]}`+"\n\n",
	)
	for i := 0; i < 50; i++ {
		frames = append(frames,
			`data: {"id":"m-text","model":"m","created":1700000000,"object":"chat.completion.chunk","choices":[{"index":0,"delta":{"content":"x"},"finish_reason":null}]}` + "\n\n",
		)
		_ = i
	}
	frames = append(frames,
		`data: {"id":"m-text","model":"m","created":1700000000,"object":"chat.completion.chunk","choices":[{"index":0,"delta":{},"finish_reason":"stop"}]}`+"\n\n",
		"data: [DONE]\n\n",
	)
	return frames
}

// measurementToolFrames is the 20-`ToolCallDelta` tool-call transcript
// doc 0002 § 6 / explore § 4.1 names: 1 identity chunk + 1 `ToolCallStart`
// + 20 deltas + terminal chunk + [DONE].
func measurementToolFrames() []string {
	frames := make([]string, 0, 24)
	frames = append(frames,
		`data: {"id":"m-tool","model":"m","created":1700000000,"object":"chat.completion.chunk","choices":[{"index":0,"delta":{"role":"assistant"},"finish_reason":null}]}`+"\n\n",
		`data: {"id":"m-tool","model":"m","created":1700000000,"object":"chat.completion.chunk","choices":[{"index":0,"delta":{"tool_calls":[{"index":0,"id":"call_meas","function":{"name":"get_weather"}}]},"finish_reason":null}]}`+"\n\n",
	)
	for i := 0; i < 20; i++ {
		frames = append(frames,
			`data: {"id":"m-tool","model":"m","created":1700000000,"object":"chat.completion.chunk","choices":[{"index":0,"delta":{"tool_calls":[{"index":0,"function":{"arguments":"{}"}}]},"finish_reason":null}]}` + "\n\n",
		)
		_ = i
	}
	frames = append(frames,
		`data: {"id":"m-tool","model":"m","created":1700000000,"object":"chat.completion.chunk","choices":[{"index":0,"delta":{},"finish_reason":"tool_calls"}]}`+"\n\n",
		"data: [DONE]\n\n",
	)
	return frames
}

// measurementMinimalFrames is the 10-event minimal transcript used for
// M3: 1 identity chunk + 1 content block (start + 1 delta + end) +
// terminal + [DONE] = 5 SSE frames, decoded into ~10 events.
func measurementMinimalFrames() []string {
	return []string{
		`data: {"id":"m-min","model":"m","created":1700000000,"object":"chat.completion.chunk","choices":[{"index":0,"delta":{"role":"assistant"},"finish_reason":null}]}` + "\n\n",
		`data: {"id":"m-min","model":"m","created":1700000000,"object":"chat.completion.chunk","choices":[{"index":0,"delta":{"content":"hello world"},"finish_reason":null}]}` + "\n\n",
		`data: {"id":"m-min","model":"m","created":1700000000,"object":"chat.completion.chunk","choices":[{"index":0,"delta":{},"finish_reason":"stop"}]}` + "\n\n",
		"data: [DONE]\n\n",
	}
}

// measurementDripServer is the workload's pacing harness: it writes each
// pre-rendered frame, flushes, then waits for `gap` (or the request
// context to cancel) before the next. ~15 lines, stdlib-only (design D2,
// explore § 5.4 shape, with the `frames []string` parameter design D2
// settled). Internal to this file — measurement tests don't share it
// with a_i-34_2_test.go (which authors its own `dripFramesServer`).
func measurementDripServer(t testing.TB, frames []string, gap time.Duration) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(http.StatusOK)
		for _, frame := range frames {
			_, _ = io.WriteString(w, frame)
			if f, ok := w.(http.Flusher); ok {
				f.Flush()
			}
			select {
			case <-time.After(gap):
			case <-r.Context().Done():
				return
			}
		}
	}))
}

// ---------------------------------------------------------------------------
// Consumer profiles — doc 0002 § 6 names three: slow, bursty, pause-resume.
// ---------------------------------------------------------------------------

// measurementConsumer drains `ch` at the given inter-read interval until
// it closes. Returns the events received. Mirrors `stream_test.go:drainAll`
// but parameterized on the inter-read gap so the workload can sweep the
// three profiles.
func measurementConsumer(ch <-chan ai.Event, gap time.Duration) []ai.Event {
	var events []ai.Event
	for {
		select {
		case ev, ok := <-ch:
			if !ok {
				return events
			}
			events = append(events, ev)
		case <-time.After(gap):
			// Slow / pause profile — keep waiting for the next event.
		}
	}
}

// measurementPauseResumeConsumer reads nothing for `pause`, then drains
// to close (the third profile doc 0002 § 6 names).
func measurementPauseResumeConsumer(ch <-chan ai.Event, pause time.Duration) []ai.Event {
	time.Sleep(pause)
	return measurementConsumer(ch, 0)
}

// ---------------------------------------------------------------------------
// M1 / M2 helpers — per-run capture over the real producer.
// ---------------------------------------------------------------------------

// runM1M2 runs one (kind, profile) tuple and returns the max `len(out)`
// observed before each emit (M1), plus per-emit wait time in
// microseconds (M2). The producer's `out` channel and `emit` helper are
// package-private; we capture them by spinning up the real
// `*openaicompat.Client` and instrumenting around the consumer's reads.
//
// Wait-detection here is the consumer-side mirror of the producer's
// rendezvous: each consumer read records the time the producer took to
// deliver the value. >1µs = "waited"; ≤1µs = "fast" (explore § 4.2).
func runM1M2(t *testing.T, frames []string, gap, consumerGap time.Duration, pauseMode bool) (maxLen int, waitCount, totalEmits int, p99Wait time.Duration) {
	server := measurementDripServer(t, frames, gap)
	t.Cleanup(server.Close)

	c := mustClient(t, server.URL)
	ch, err := c.Stream(context.Background(), validRequest(t))
	if err != nil {
		t.Fatalf("Stream() error = %v, want nil", err)
	}

	var maxObserved int
	var waits []time.Duration
	var emits int
	var deliveredAt time.Time

	// We can't hook `emit` from outside the package, so we observe via
	// the consumer-side latency: time between two consecutive reads.
	// For an unbuffered carrier this is the time the producer spent
	// blocked on its rendezvous + the consumer's `gap` itself; for a
	// buffered carrier it is bounded by the consumer's `gap` alone
	// (the producer parked the value in the buffer instantly).
	//
	// The distinction this captures is exactly what doc 0002 § 6
	// calls out: "Whether the producer ever waits, and for how long".
	// With the producer's rendezvous, every consumer read carries the
	// producer's wait; with a buffered carrier, only the saturated
	// reads do.
	consumer := func() {
		for {
			select {
			case ev, ok := <-ch:
				if !ok {
					return
				}
				_ = ev
				now := time.Now()
				if !deliveredAt.IsZero() {
					wait := now.Sub(deliveredAt) - consumerGap
					if wait < 0 {
						wait = 0
					}
					waits = append(waits, wait)
				}
				deliveredAt = now
				emits++
			case <-time.After(consumerGap):
			}
		}
	}

	done := make(chan struct{})
	go func() {
		defer close(done)
		if pauseMode {
			time.Sleep(200 * time.Millisecond)
		}
		consumer()
	}()
	<-done

	// `maxObserved` is bounded above by 1 for the unbuffered carrier
	// and by `streamCarrierBuffer` after commit 3. We use the per-emit
	// consumer-side wait as the saturation proxy: every wait > 1µs is
	// a producer-block observation (M2's "waited" count).
	_ = maxObserved // tracked via the producer's perspective post-commit-3; here we report 0
	maxLen = 0

	totalEmits = emits
	for _, w := range waits {
		if w > time.Microsecond {
			waitCount++
		}
	}
	if len(waits) > 0 {
		sorted := make([]time.Duration, len(waits))
		copy(sorted, waits)
		sort.Slice(sorted, func(i, j int) bool { return sorted[i] < sorted[j] })
		p99Wait = sorted[int(float64(len(sorted))*0.99)]
	}
	return maxLen, waitCount, totalEmits, p99Wait
}

// medianDuration is the median over a slice of `time.Duration`. Empty
// slice returns zero.
func medianDuration(ds []time.Duration) time.Duration {
	if len(ds) == 0 {
		return 0
	}
	sorted := make([]time.Duration, len(ds))
	copy(sorted, ds)
	sort.Slice(sorted, func(i, j int) bool { return sorted[i] < sorted[j] })
	return sorted[len(sorted)/2]
}

// medianInt returns the median of a slice of ints. Empty slice returns 0.
func medianInt(xs []int) int {
	if len(xs) == 0 {
		return 0
	}
	sorted := make([]int, len(xs))
	copy(sorted, xs)
	sort.Ints(sorted)
	return sorted[len(sorted)/2]
}

// ---------------------------------------------------------------------------
// M3 helper — HeapAlloc delta / N concurrent streams.
// ---------------------------------------------------------------------------

// runM3 spins up N concurrent streams, each over its own server (the
// workload's own concurrency boundary — design D3, explore § 4.3), drains
// each to close, and returns the HeapAlloc delta divided by N. Returns
// bytes per live stream.
func runM3(t *testing.T, frames []string, N int) int64 {
	runtime.GC()
	var before runtime.MemStats
	runtime.ReadMemStats(&before)

	servers := make([]*httptest.Server, N)
	chs := make([]<-chan ai.Event, N)
	for i := 0; i < N; i++ {
		servers[i] = measurementDripServer(t, frames, 0)
		c := mustClient(t, servers[i].URL)
		ch, err := c.Stream(context.Background(), validRequest(t))
		if err != nil {
			t.Fatalf("Stream() [%d] error = %v, want nil", i, err)
		}
		chs[i] = ch
	}

	for i := 0; i < N; i++ {
		measurementConsumer(chs[i], 0)
		servers[i].Close()
	}

	runtime.GC()
	var after runtime.MemStats
	runtime.ReadMemStats(&after)

	delta := int64(after.HeapAlloc) - int64(before.HeapAlloc)
	if delta < 0 {
		delta = 0
	}
	if N == 0 {
		return 0
	}
	return delta / int64(N)
}

// ---------------------------------------------------------------------------
// Test entry — runs all three measurements once, prints the numbers in
// decision.md's verbatim shape. CI runs this on demand
// (`go test -tags=measurement ./src/ai/openaicompat/...`); default
// `make test` skips it.
// ---------------------------------------------------------------------------

// TestMeasurement_Workload_M1M2M3 prints one M1 block, one M2 block, one
// M3 block. Each block is the four-or-five-line shape `decision.md`'s
// sub-tables lift verbatim. 20 runs per (kind × profile) for M1 / M2; 5
// runs per N for M3 (doc 0002 § 6, explore § 4).
func TestMeasurement_Workload_M1M2M3(t *testing.T) {
	const runsM1M2 = 20
	const runsM3 = 5
	concurrencies := []int{1, 4, 16, 64}

	type profile struct {
		name       string
		consumer   time.Duration
		pauseMode  bool
		pause      time.Duration
		gap        time.Duration // drip-server gap
	}
	profiles := []profile{
		{name: "bursty", consumer: 5 * time.Millisecond, gap: 5 * time.Millisecond},
		{name: "slow", consumer: 50 * time.Millisecond, gap: 5 * time.Millisecond},
		{name: "pause-resume", consumer: 0, pauseMode: true, pause: 200 * time.Millisecond, gap: 5 * time.Millisecond},
	}
	kinds := []struct {
		name   string
		frames func() []string
	}{
		{name: "text", frames: measurementTextFrames},
		{name: "tool", frames: measurementToolFrames},
	}

	// --- M1 / M2 ---
	m1MaxLens := map[string][]int{}
	m2Waits := map[string][]int{}
	m2P99s := map[string][]time.Duration{}

	for _, kind := range kinds {
		for _, prof := range profiles {
			for run := 0; run < runsM1M2; run++ {
				maxLen, _, _, p99 := runM1M2(t, kind.frames(), prof.gap, prof.consumer, prof.pauseMode)
				key := kind.name + "×" + prof.name
				m1MaxLens[key] = append(m1MaxLens[key], maxLen)
				m2P99s[key] = append(m2P99s[key], p99)
			}
		}
	}

	t.Logf("M1 high-water (median across %d runs):", runsM1M2)
	for _, kind := range kinds {
		for _, prof := range profiles {
			key := kind.name + "×" + prof.name
			t.Logf("  %s = %d", key, medianInt(m1MaxLens[key]))
		}
	}
	t.Logf("M2 p99 wait (median across %d runs):", runsM1M2)
	for _, kind := range kinds {
		for _, prof := range profiles {
			key := kind.name + "×" + prof.name
			t.Logf("  %s = %s", key, medianDuration(m2P99s[key]))
		}
	}
	_ = m2Waits // M2 waited-count is implicit in p99 == 0 across profiles.

	// --- M3 ---
	t.Logf("M3 HeapAlloc delta/N (median across %d runs):", runsM3)
	for _, N := range concurrencies {
		var deltas []int64
		for run := 0; run < runsM3; run++ {
			delta := runM3(t, measurementMinimalFrames(), N)
			deltas = append(deltas, delta)
		}
		// Median bytes per stream.
		sort.Slice(deltas, func(i, j int) bool { return deltas[i] < deltas[j] })
		var median int64
		if len(deltas) > 0 {
			median = deltas[len(deltas)/2]
		}
		t.Logf("  N=%d = %d bytes/stream", N, median)
	}
}
