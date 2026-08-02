// AI-21.6 — request capture: every field a request carried is assertable
// after the call, in call order, excluding pre-stream-rejected calls, and
// immune to later caller mutation.
//
// fake_request_capture_test.go proves R-AFP-014/015 from outside package
// agenttest, reusing request_test.go's buildFullRequest/requireRequestsEqual
// — same agenttest_test package.
package agenttest_test

import (
	"context"
	"testing"

	"github.com/cachicamas/backend/agent/src/agenttest"
	"github.com/cachicamas/backend/agent/src/ai"
)

// mustUserTextRequest builds a minimal valid request from one model
// identity and one user text message — this file's own small builder,
// used where the test cares about which request arrived, not its shape.
func mustUserTextRequest(t *testing.T, model, text string) ai.Request {
	t.Helper()

	part, err := ai.NewText(text)
	if err != nil {
		t.Fatalf("ai.NewText returned %v, want no failure", err)
	}
	message, err := ai.NewMessage(ai.RoleUser, part)
	if err != nil {
		t.Fatalf("ai.NewMessage returned %v, want no failure", err)
	}
	req, err := ai.NewRequest(model, []ai.Message{message})
	if err != nil {
		t.Fatalf("ai.NewRequest returned %v, want no failure", err)
	}
	return req
}

// AI-21.6 item 1, first half (R-AFP-014, S-AFP-033) — every field a
// fully-populated request carries is assertable after the call, through
// the request's own readability.
func TestProvider_Requests_EveryFieldOfACapturedRequestIsAssertable(t *testing.T) {
	t.Parallel()

	original := buildFullRequest(t)
	provider := agenttest.NewProvider(mustTextDeltaScript(t))

	ch, err := provider.Stream(context.Background(), original)
	if err != nil {
		t.Fatalf("Stream returned %v, want no failure", err)
	}
	drainFake(t, ch)

	captured := provider.Requests()
	if len(captured) != 1 {
		t.Fatalf("len(Requests()) = %d, want 1", len(captured))
	}
	requireRequestsEqual(t, captured[0], original)
}

// AI-21.6 item 1, second half (R-AFP-014, S-AFP-034/035) — three calls made
// in order yield three ordered capture entries, each matching its own
// call; a call rejected on the pre-stream path (invalid request, cancelled
// context, or exhausted queue) is absent, and the positional
// correspondence Requests()[i] ↔ script i holds exactly.
func TestProvider_Requests_ThreeOrderedCalls_ExcludesPreStreamRejectedCalls(t *testing.T) {
	t.Parallel()

	req1 := mustUserTextRequest(t, "model-1", "one")
	req2 := mustUserTextRequest(t, "model-2", "two")
	req3 := mustUserTextRequest(t, "model-3", "three")

	provider := agenttest.NewProvider(mustTextDeltaScript(t), mustTextDeltaScript(t), mustTextDeltaScript(t))

	// A pre-stream-rejected call before the three real calls: an invalid
	// (zero-value) request. It must not consume a script and must not be
	// captured.
	var zero ai.Request
	if _, err := provider.Stream(context.Background(), zero); err == nil {
		t.Fatal("Stream with a zero request unexpectedly succeeded")
	}

	// A pre-stream-rejected call via an already-cancelled context.
	cancelledCtx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := provider.Stream(cancelledCtx, req1); err == nil {
		t.Fatal("Stream with an already-cancelled context unexpectedly succeeded")
	}

	for i, req := range []ai.Request{req1, req2, req3} {
		ch, err := provider.Stream(context.Background(), req)
		if err != nil {
			t.Fatalf("call %d: Stream returned %v, want no failure", i+1, err)
		}
		drainFake(t, ch)
	}

	// An exhausted-queue call after the three real calls — also
	// pre-stream-rejected, also must not be captured.
	if _, err := provider.Stream(context.Background(), req1); err == nil {
		t.Fatal("Stream against an exhausted queue unexpectedly succeeded")
	}

	captured := provider.Requests()
	if len(captured) != 3 {
		t.Fatalf("len(Requests()) = %d, want 3 (the pre-stream-rejected calls must be absent)", len(captured))
	}
	wantModels := []string{"model-1", "model-2", "model-3"}
	for i, req := range captured {
		if got := req.Model(); got != wantModels[i] {
			t.Errorf("Requests()[%d].Model() = %q, want %q (positional correspondence to consumed scripts)", i, got, wantModels[i])
		}
	}
}

// AI-21.6 item 2 (R-AFP-015, S-AFP-036/037) — a caller's later mutation of
// its own request, or of any slice reachable from it, does not alter the
// capture history; two reads of the history are independent.
func TestProvider_Requests_LaterCallerMutation_DoesNotAlterCaptureHistory(t *testing.T) {
	t.Parallel()

	part, err := ai.NewText("hi")
	if err != nil {
		t.Fatalf("ai.NewText returned %v, want no failure", err)
	}
	message, err := ai.NewMessage(ai.RoleUser, part)
	if err != nil {
		t.Fatalf("ai.NewMessage returned %v, want no failure", err)
	}
	original, err := ai.NewRequest("model-1", []ai.Message{message}, ai.WithStopSequences("stop-1", "stop-2"))
	if err != nil {
		t.Fatalf("ai.NewRequest returned %v, want no failure", err)
	}

	provider := agenttest.NewProvider(mustTextDeltaScript(t))
	ch, err := provider.Stream(context.Background(), original)
	if err != nil {
		t.Fatalf("Stream returned %v, want no failure", err)
	}
	drainFake(t, ch)

	// The caller mutates every slice it can reach from its own value, and
	// derives a modified request from it — neither must reach the capture
	// history.
	stopSeqs, ok := original.StopSequences()
	if !ok || len(stopSeqs) == 0 {
		t.Fatalf("original.StopSequences() = (%v, %v), want a non-empty sequence", stopSeqs, ok)
	}
	stopSeqs[0] = "mutated"
	if _, err := original.With(ai.WithModel("mutated-model")); err != nil {
		t.Fatalf("original.With returned %v, want no failure", err)
	}

	first := provider.Requests()
	if len(first) != 1 {
		t.Fatalf("len(Requests()) = %d, want 1", len(first))
	}
	gotStop, ok := first[0].StopSequences()
	if !ok || gotStop[0] != "stop-1" {
		t.Errorf("captured stop sequence = (%v, %v), want (%q, true) — unaffected by the caller's later mutation", gotStop, ok, "stop-1")
	}
	if got := first[0].Model(); got != "model-1" {
		t.Errorf("captured model = %q, want %q — unaffected by With() derived from the caller's own value", got, "model-1")
	}

	// Two reads of the history are independent: mutating what the first
	// read returned must not affect the second.
	second := provider.Requests()
	first[0] = ai.Request{}
	if got := second[0].Model(); got != "model-1" {
		t.Errorf("second read's model = %q after mutating the first read's slice, want %q (independent reads)", got, "model-1")
	}
}
