// AI-33.1 — Cancel before headers (R-AIS-034 / S-1, S-2, S-3).
//
// Pre-stream cancellation proof against a real HTTP transport. The
// producer's pre-handover cancellation path is already implemented and
// merged at base main (stream.go:181–183, :217–230, :241–251); the gap
// closed by this file is the REAL-HTTP proof, not the unit proof.
//
// The existing TestStream_ValidRequestAlreadyCancelledContext_Reports
// CancellationPreStream (stream_test.go:341) covers R-ATS-002 against a
// benign 200 OK fixture. This file covers R-AIS-034 against a real
// httptest.NewServer whose handler never writes a response — the same
// shape the conformance suite's R-CNF-011 bounded-close case uses
// (stream_failure_test.go:391, contextBeforeFirstFrameServer), extended
// to prove the leak posture and the tool-call accumulation path
// (charter line 1989: every scenario runs over text AND tool-call
// streams). No conformance assertion is rewritten — these tests ride
// alongside, leak-checked via agenttest.RequireNoGoroutineLeak
// (stream_kit_leak.go:107) which the helper's own Setenv posture
// mechanically rejects on parallel ancestors (R-STK-008 — no
// t.Parallel() call anywhere in this file).
//
// Pins: R-CNF-011 (conformance bounded-close invariant — stays green),
// R-ATS-002 (pre-stream contract — validate, then Translate, then
// consult ctx, with nothing observable before validation passes).

package openaicompat

import (
	"context"
	"errors"
	"testing"

	"github.com/cachicamas/backend/agent/src/agenttest"
	"github.com/cachicamas/backend/agent/src/ai"
)

// TestAI33_1_TextStream_CancelBeforeDo covers R-AIS-034 / S-1: a
// text-stream request whose context is cancelled before Client.Stream
// is invoked returns the pre-stream cancellation failure (R-ATS-002
// line 181–183) with a nil channel — no producer goroutine is spawned,
// no HTTP exchange is attempted. Repeated inside RequireNoGoroutineLeak
// (50 repeats) so a per-call leak in any future refactor of
// preStreamCancellation would surface here, not silently.
func TestAI33_1_TextStream_CancelBeforeDo(t *testing.T) {
	agenttest.RequireNoGoroutineLeak(t, func() {
		server := contextBeforeFirstFrameServer(t)
		defer server.Close()

		c := mustClient(t, server.URL)

		ctx, cancel := context.WithCancel(context.Background())
		cancel() // cancel BEFORE Stream — line 181 check is the only path

		ch, err := c.Stream(ctx, validRequest(t))

		if ch != nil {
			t.Errorf("Stream() channel = %v, want nil — pre-stream failure returns the failure directly, never a carrier (R-AIS-034 / S-1)", ch)
		}
		if err == nil {
			t.Fatal("Stream() error = nil, want a *ai.Failure (R-AIS-034 / S-1)")
		}
		var failure *ai.Failure
		if !errors.As(err, &failure) {
			t.Fatalf("Stream() error = %v (%T), want a *ai.Failure (R-AIS-034 / S-1)", err, err)
		}
		if failure.Category() != ai.FailureCategoryCancellation {
			t.Errorf("Category() = %v, want FailureCategoryCancellation (R-AIS-034 / S-1, R-ATS-002)", failure.Category())
		}
		if failure.Delivery() != ai.DeliveryPreStream {
			t.Errorf("Delivery() = %v, want DeliveryPreStream (R-AIS-034 / S-1, R-ATS-002)", failure.Delivery())
		}
		if failure.PartialOutput() {
			t.Error("PartialOutput() = true, want false — no event crossed the carrier on a pre-stream cancellation (R-AIS-034 / S-1)")
		}
	})
}

// TestAI33_1_ToolCallStream_CancelBeforeDo covers R-AIS-034 / S-2:
// the same observable outcome as S-1, but with a request that carries
// at least one tool declaration — so the tool-call accumulation path
// is exercised end-to-end, not just the text-block minting path
// (charter line 1989). A cancellation proof that never crosses the
// tool-call accumulation path proves nothing about its buffers, and
// the FAILURE category must be the same regardless of which request
// shape was used.
func TestAI33_1_ToolCallStream_CancelBeforeDo(t *testing.T) {
	agenttest.RequireNoGoroutineLeak(t, func() {
		server := contextBeforeFirstFrameServer(t)
		defer server.Close()

		c := mustClient(t, server.URL)

		ctx, cancel := context.WithCancel(context.Background())
		cancel()

		ch, err := c.Stream(ctx, validToolCallRequest(t))

		if ch != nil {
			t.Errorf("Stream() channel = %v, want nil (R-AIS-034 / S-2)", ch)
		}
		if err == nil {
			t.Fatal("Stream() error = nil, want a *ai.Failure (R-AIS-034 / S-2)")
		}
		var failure *ai.Failure
		if !errors.As(err, &failure) {
			t.Fatalf("Stream() error = %v (%T), want a *ai.Failure (R-AIS-034 / S-2)", err, err)
		}
		if failure.Category() != ai.FailureCategoryCancellation {
			t.Errorf("Category() = %v, want FailureCategoryCancellation (R-AIS-034 / S-2, R-ATS-002)", failure.Category())
		}
		if failure.Delivery() != ai.DeliveryPreStream {
			t.Errorf("Delivery() = %v, want DeliveryPreStream (R-AIS-034 / S-2, R-ATS-002)", failure.Delivery())
		}
		if failure.PartialOutput() {
			t.Error("PartialOutput() = true, want false (R-AIS-034 / S-2)")
		}
	})
}

// TestAI33_1_RaceCancelMidDo covers R-AIS-034 / S-3: a real race —
// ctx is cancelled CONCURRENTLY with Stream's invocation. The cancel
// goroutine fires without a delay, racing the producer's pre-stream
// contract (stream.go:181–183) and the pre-handover transport-failure
// branch (stream.go:191–193); whichever branch wins, both collapse to
// the same observable outcome — *ai.Failure with
// FailureCategoryCancellation and DeliveryPreStream, plus a nil
// channel — because preStreamCancellation reports the cancellation
// directly and preStreamTransportFailure collapses its category to
// FailureCategoryCancellation when ctx.Err() is non-nil at the moment
// of failure.
//
// The server fixture uses contextBeforeFirstFrameServer — the same
// hijack-and-close shape S-1 and S-2 share — so httpClient.Do returns
// an error fast (no goroutine is ever spawned) regardless of which
// branch fires. The race is purely about WHICH pre-handover branch
// takes ownership; the test's contract is the OUTCOME, never the
// branch.
//
// NOTE on S-3 wording: the scenario as written in spec.md line 110
// reads "cancel ctx after httpClient.Do returns but before run() reads
// any byte". Stream() returns once, atomically — there is no
// intermediate "post-Do, pre-run" moment to land in. The two genuine
// pre-handover windows (the line 181 ctx.Err() check and the
// preStreamTransportFailure branch at line 192) are the only places a
// "pre-stream cancellation" observation can ever live; this test
// races both with no delay so each repeat of RequireNoGoroutineLeak
// may hit either branch — the assertable contract is the
// (category, delivery, channel-nil, no-leak) tuple, identical to S-1
// and S-2.
func TestAI33_1_RaceCancelMidDo(t *testing.T) {
	agenttest.RequireNoGoroutineLeak(t, func() {
		server := contextBeforeFirstFrameServer(t)
		defer server.Close()

		c := mustClient(t, server.URL)

		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		// Fire cancel concurrently with Stream — no delay. The cancel
		// may land before the line 181 check (in which case
		// preStreamCancellation wins), or during httpClient.Do (in
		// which case preStreamTransportFailure wins with a cancelled
		// ctx). Both branches converge to the same observable
		// outcome, which is what the assertion below verifies.
		go cancel()

		ch, err := c.Stream(ctx, validRequest(t))

		if ch != nil {
			t.Errorf("Stream() channel = %v, want nil (R-AIS-034 / S-3)", ch)
		}
		if err == nil {
			t.Fatal("Stream() error = nil, want a *ai.Failure (R-AIS-034 / S-3)")
		}
		var failure *ai.Failure
		if !errors.As(err, &failure) {
			t.Fatalf("Stream() error = %v (%T), want a *ai.Failure (R-AIS-034 / S-3)", err, err)
		}
		if failure.Category() != ai.FailureCategoryCancellation {
			t.Errorf("Category() = %v, want FailureCategoryCancellation (R-AIS-034 / S-3)", failure.Category())
		}
		if failure.Delivery() != ai.DeliveryPreStream {
			t.Errorf("Delivery() = %v, want DeliveryPreStream (R-AIS-034 / S-3)", failure.Delivery())
		}
		if failure.PartialOutput() {
			t.Error("PartialOutput() = true, want false (R-AIS-034 / S-3)")
		}
	})
}

// validToolCallRequest builds a minimal, valid ai.Request that carries
// ONE tool declaration alongside the user message — so the request's
// own Translate path renders the tool region, and a non-empty tools
// region is part of what crosses the wire (cache_marker_test.go:108
// already proves the same shape is what AI-26.4's tool.go itself
// emits). One tool, one message — exactly enough to exercise the
// tool-call accumulation path; not so much that the test's own setup
// dominates its assertions.
func validToolCallRequest(t *testing.T) ai.Request {
	t.Helper()

	tool, err := ai.NewTool(
		"get_weather",
		"Get the current weather for a location.",
		[]byte(`{"type":"object","properties":{"location":{"type":"string"}},"required":["location"]}`),
	)
	if err != nil {
		t.Fatalf("ai.NewTool: %v", err)
	}
	tools, err := ai.NewToolSet(tool)
	if err != nil {
		t.Fatalf("ai.NewToolSet: %v", err)
	}

	msg, err := ai.NewMessage(ai.RoleUser, mustTextPart(t, "what is the weather?"))
	if err != nil {
		t.Fatalf("ai.NewMessage: %v", err)
	}

	req, err := ai.NewRequest("requested-model", []ai.Message{msg}, ai.WithTools(tools))
	if err != nil {
		t.Fatalf("ai.NewRequest: %v", err)
	}
	return req
}
