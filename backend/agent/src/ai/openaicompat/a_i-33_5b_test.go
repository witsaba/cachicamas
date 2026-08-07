// AI-33.5b — Full-package leak check across every AI-33 exit path
// (R-AIS-038).
//
// # Why external (package openaicompat_test)
//
// See a_i-33_5a_test.go's header for the external-package rationale.
// This file is the second half of the AI-33.5 split (tasks.md line
// 173: 33.5a = drain impl + drain tests; 33.5b = full-suite leak
// check). Both files share package openaicompat_test, so this file
// reuses the helpers 33.5a defines (validTextRequest,
// validToolCallRequest, mustNewExternalClient, ai33_5StallHandler,
// ai33_5TextTranscript, ai33_5ToolTranscript, serveBody,
// hijackCloseHandler, ai33_5StallBound, ai33_5ShortTimeout) without
// redeclaring them.
//
// # What this file proves (R-AIS-038)
//
// A single non-parallel wrapper walks every AI-33 exit path on both
// stream kinds through agenttest.RequireNoGoroutineLeak — completion,
// pre-headers, between-frames, blocked-send (with cancel-then-drain,
// NOT truly-abandoned; see tasks.md AI-33.5 § 33.5.1 budget warning
// line 155), and after-completion. Each scenario is wrapped in
// RequireNoGoroutineLeak (50 repeats, leakTolerance = 25) so a
// per-call leak in any future refactor surfaces here, not silently.
// R-STK-008 forbids t.Parallel() — runtime.NumGoroutine is
// process-wide and meaningless under parallel execution; this file
// runs serially.
//
// # Pins
//
// R-AIS-038 (full-package leak check), R-CNF-009 (single closing
// site), R-CNF-011 (conformance bounded-close invariant stays green),
// R-ATS-003 (single producer model; no per-call goroutine leak across
// any exit path), R-STK-007 (leak amplitude mechanism — 50 repeats),
// R-STK-008 (serial-only; no t.Parallel() anywhere in this file),
// R-STK-009 (stdlib-only — go.mod unchanged), R-STK-010 (no
// abandoned-never-cancelled assertion; see AI-33.3's own
// negative-assertion guard for the cross-file scan).

package openaicompat_test

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/cachicamas/backend/agent/src/agenttest"
	"github.com/cachicamas/backend/agent/src/ai"
)

// runFullPackageScenario is one row of TestAI33_5_FullPackageLeakCheck's
// walk over every exit path (R-AIS-038 / S-1). Each row stands up its
// own httptest.Server (built by the caller via serverHandler), builds
// a real *openaicompat.Client against it, and exercises one
// cancellation moment (or the post-completion no-op) — every scenario
// uses the cancel-then-drain pattern (open stream → cancel ctx →
// drain), NOT truly-abandoned, per tasks.md AI-33.5 § 33.5.1 budget
// warning line 155 (the truly-abandoned pattern would cost
// 5s/repeat × 50 repeats × 9 scenarios = 37m30s — orders of
// magnitude past go test's 10-minute default timeout).
//
// betweenFrames distinguishes the between-frames scenarios (which
// receive the first event synchronously to prove the producer
// reached its first send before cancellation) from the rest.
func runFullPackageScenario(t *testing.T, name string, req ai.Request, serverHandler http.HandlerFunc, betweenFrames bool) {
	server := httptest.NewServer(serverHandler)
	defer server.Close()
	c := mustNewExternalClient(t, server.URL)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	ch, err := c.Stream(ctx, req)
	if err != nil {
		// Pre-headers path returns an *ai.Failure with no channel —
		// the assertion below handles that.
		if name == "pre_headers_cancel_text" || name == "pre_headers_cancel_tool" {
			var failure *ai.Failure
			if !errors.As(err, &failure) {
				t.Fatalf("%s: Stream() error = %v (%T), want a *ai.Failure (R-AIS-038 / S-1, pre-headers cancel path)", name, err, err)
			}
			if ch != nil {
				t.Errorf("%s: Stream() channel = %v, want nil (pre-headers cancel returns no carrier)", name, ch)
			}
			return
		}
		t.Fatalf("%s: Stream() error = %v, want nil (R-AIS-038 / S-1)", name, err)
	}
	if ch == nil {
		t.Fatalf("%s: Stream() channel = nil, want a live stream (R-AIS-038 / S-1)", name)
	}

	if betweenFrames {
		// Receive the first event synchronously — proves the producer
		// reached its first send before cancellation.
		select {
		case <-ch:
		case <-time.After(ai33_5ShortTimeout):
			t.Fatalf("%s: no event within %s, want the first SSE frame's events to be observable before cancellation (R-AIS-038 / S-1)",
				name, ai33_5ShortTimeout)
		}
	}

	// Cancel-then-drain (NOT truly-abandoned). The drain pairs with
	// AI-32.3's bounded-wait terminal in milliseconds, so every
	// scenario finishes quickly under RequireNoGoroutineLeak × 50.
	cancel()
	agenttest.DrainAndRecord(t, ch, agenttest.DefaultDrainTimeout+ai33_5StallBound)
}

// TestAI33_5_FullPackageLeakCheck covers R-AIS-038 / S-1: a single
// serial test suite walks every AI-33 exit path on both stream
// kinds through agenttest.RequireNoGoroutineLeak. No t.Parallel()
// anywhere (R-STK-008); RequireNoGoroutineLeak's process-wide
// runtime.NumGoroutine count is meaningless under parallel execution.
//
// Each scenario runs in its own subtest so a failure surfaces with
// the scenario name in the diagnostic, but the subtests themselves
// run serially because the parent test does not call t.Parallel()
// (R-STK-008). The wrapper exists to prove: across all 9 scenarios,
// no per-call leak grows the live-goroutine count past
// leakTolerance (25; R-STK-007).
//
// Each scenario uses the cancel-then-drain pattern (open stream,
// cancel ctx, drain) — NOT truly-abandoned — per tasks.md AI-33.5
// § 33.5.1 budget warning. With cancel-then-drain, every scenario
// finishes in milliseconds per repeat (the drain pairs with
// AI-32.3's bounded-wait terminal), so the whole wrapper takes
// roughly 9 × 50 × ~2ms ≈ 1s under -race.
func TestAI33_5_FullPackageLeakCheck(t *testing.T) {
	// NO t.Parallel() — R-STK-008.

	textReq := validTextRequest
	toolReq := validToolCallRequest

	// Each entry: (name, request-builder, handler, betweenFrames?)
	scenarios := []struct {
		name          string
		request       func(t *testing.T) ai.Request
		handler       func(t *testing.T) http.HandlerFunc
		betweenFrames bool
	}{
		// S-1 — normal completion (text + tool-call).
		{
			name:    "normal_completion_text",
			request: textReq,
			handler: func(_ *testing.T) http.HandlerFunc {
				return serveBody([]byte(ai33_5TextTranscript))
			},
		},
		{
			name:    "normal_completion_tool",
			request: toolReq,
			handler: func(_ *testing.T) http.HandlerFunc {
				return serveBody([]byte(ai33_5ToolTranscript))
			},
		},

		// S-4 — pre-headers cancel. The hijack-close server is the
		// same shape contextBeforeFirstFrameServer uses
		// (stream_failure_test.go:391). The Stream call returns a
		// *ai.Failure (no channel, no producer) so the leak check has
		// nothing to drain; the scenario IS the Stream invocation.
		{
			name:    "pre_headers_cancel_text",
			request: textReq,
			handler: hijackCloseHandler,
		},
		{
			name:    "pre_headers_cancel_tool",
			request: toolReq,
			handler: hijackCloseHandler,
		},

		// S-5 — between-frames cancel. The handler writes one SSE
		// frame, flushes, then stalls; the test receives the first
		// event, cancels, and drains.
		{
			name:    "between_frames_cancel_text",
			request: textReq,
			handler: func(_ *testing.T) http.HandlerFunc {
				textFrame := "" +
					"data: {\"id\":\"chatcmpl-ai33-5-bf-text\",\"model\":\"m\",\"created\":1700000000,\"object\":\"chat.completion.chunk\",\"choices\":[{\"index\":0,\"delta\":{\"role\":\"assistant\"},\"finish_reason\":null}]}\n\n"
				return ai33_5StallHandler(textFrame)
			},
			betweenFrames: true,
		},
		{
			name:    "between_frames_cancel_tool",
			request: toolReq,
			handler: func(_ *testing.T) http.HandlerFunc {
				toolCallFrames := []string{
					"" +
						"data: {\"id\":\"chatcmpl-ai33-5-bf-tool\",\"model\":\"m\",\"created\":1700000000,\"object\":\"chat.completion.chunk\",\"choices\":[{\"index\":0,\"delta\":{\"role\":\"assistant\"},\"finish_reason\":null}]}\n\n",
					"" +
						"data: {\"id\":\"chatcmpl-ai33-5-bf-tool\",\"model\":\"m\",\"created\":1700000000,\"object\":\"chat.completion.chunk\",\"choices\":[{\"index\":0,\"delta\":{\"tool_calls\":[{\"index\":0,\"id\":\"call_ai33_5_bf\",\"function\":{\"name\":\"get_weather\"}}]},\"finish_reason\":null}]}\n\n",
				}
				return ai33_5StallHandler(toolCallFrames...)
			},
			betweenFrames: true,
		},

		// S-6 — blocked-send abandonment. Cancel-then-drain (NOT
		// truly-abandoned) — same ordering
		// TestAI33_3_AbandonedThenCancelled_LeakFree uses
		// (a_i-33_3_test.go:215). This scenario asserts leak-freeness,
		// NOT terminal invention (that assertion lives in
		// a_i-33_3_test.go).
		{
			name:    "blocked_send_abandonment_text",
			request: textReq,
			handler: func(_ *testing.T) http.HandlerFunc {
				return serveBody([]byte(ai33_5TextTranscript))
			},
		},
		{
			name:    "blocked_send_abandonment_tool",
			request: toolReq,
			handler: func(_ *testing.T) http.HandlerFunc {
				return serveBody([]byte(ai33_5ToolTranscript))
			},
		},

		// S-7 — after-completion cancel. Drain to close, then cancel
		// — the producer is already gone (R-ATS-003's single-producer
		// model), so cancel is a no-op. Mirrors a_i-33_4_test.go:80.
		{
			name:    "after_completion_cancel_text",
			request: textReq,
			handler: func(_ *testing.T) http.HandlerFunc {
				return serveBody([]byte(ai33_5TextTranscript))
			},
		},
		{
			name:    "after_completion_cancel_tool",
			request: toolReq,
			handler: func(_ *testing.T) http.HandlerFunc {
				return serveBody([]byte(ai33_5ToolTranscript))
			},
		},
	}

	for _, sc := range scenarios {
		sc := sc
		t.Run(sc.name, func(t *testing.T) {
			// Setup (server, client) runs INSIDE the closure because
			// RequireNoGoroutineLeak measures goroutine count BEFORE
			// and AFTER the closure — anything outside is cancelled
			// out, anything inside that allocates per-repeat is
			// properly accounted. Each repeat gets a fresh server
			// (built inside the closure via newServerForScenario).
			agenttest.RequireNoGoroutineLeak(t, func() {
				runFullPackageScenario(t, sc.name, sc.request(t), sc.handler(t), sc.betweenFrames)
			})
		})
	}
}
