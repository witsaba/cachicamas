// AI-34.2 — dripFramesServer helper: the only new helper AI-34 introduces.
// ~20 lines, test-only, `package openaicompat` so a_i-34_2_test.go and
// a_i-34_3_test.go can share it without exporting anything through
// src/agenttest.
//
// # Why this helper, and why internal
//
// The slow-consumer pattern for the *real* producer does not exist
// before AI-34 (explore § 5.4, design D2). `slowSSEServer`
// (stream_test.go:106) stalls the server AFTER one flush — purpose-built
// for AI-33.2's between-frames cancellation proof, not for the
// backpressure workload. `dripFramesServer` paces frames frame-by-frame
// with a configurable inter-frame gap, so a slow / bursty / pause-resume
// consumer can be driven against the real producer at a known pace
// (design D2).
//
// Internal `package openaicompat` is the only way to keep
// `a_i-34_2_test.go`'s RED-first `cap(ch) == streamCarrierBuffer`
// assertion referring to the package-private constant directly
// (proposal § 19, design D2's "reach"). `agenttest/` is wrong: that
// package's own doc pins it as dependency-free and stdlib-only
// (stream_kit_leak.go:12–25); the helper is test-fixture-only, never
// used by any agenttest surface.
//
// # Cancellation posture
//
// The select on `r.Context().Done()` is the same idiom `slowSSEServer`
// uses (stream_test.go:117) and the same posture `dripFramesServer`'s
// own design D2 settles: a cancelled request means the test is tearing
// down, not that the drip should keep going and waste cycles.
//
// # No new helpers, no new deps
//
// stdlib only: `net/http`, `net/http/httptest`, `io`, `time`. The
// helper has no test of its own — its consumer is the AI-34.2 / AI-34.3
// tests that import it. Reuse posture matches
// `serveTranscripts` (bridge_test.go:98): "calls no *testing.T method
// inside the handler; the test goroutine does any failure work".

package openaicompat

import (
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

// dripFramesServer returns a local test server whose handler writes each
// pre-rendered SSE frame to the wire, flushes, then waits for `gap` (or
// the request's own context to cancel, whichever comes first) before
// the next frame. The pace is what a slow / bursty / pause-resume
// consumer reads against — frames arrive one at a time at a known
// interval, the way a real provider's natural cadence looks.
//
// The helper is test-only; nothing under src/agenttest is permitted to
// depend on it. Frames are already-rendered SSE bytes — callers compose
// multi-frame transcripts the way `ai33TextTranscript`
// (a_i-33_3_test.go:114) already does.
func dripFramesServer(t *testing.T, frames []string, gap time.Duration) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(http.StatusOK)
		for _, frame := range frames {
			if _, err := io.WriteString(w, frame); err != nil {
				return
			}
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
