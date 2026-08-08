// AI-37 Judgment Day remediation (item 1, CRITICAL) — R-AOB-005.
//
// Both blind judges independently found the same defect: on the `[DONE]`
// terminal-sentinel path, run set outcome.completion/haveCompletion
// BEFORE calling sendEvent(completion), and discarded the send's own
// bool. emit returns false when ctx.Done() wins the race against the
// send; the carrier is unbuffered (streamCarrierBuffer == 0), so a
// caller that cancels and stops draining deterministically loses the
// completion. outcome.failure stayed nil, so the deferred finalizeSpan
// took the SUCCESS branch — codes.Ok, http.response.status_code,
// gen_ai.response.finish_reasons and every gen_ai.usage.* count recorded
// for a request whose stream consumer received no completion at all.
// This contradicts R-AOB-005: "[usage attributes] MUST equal the usage
// carried by the completion event that the consumer receives ... so that
// what a telemetry consumer sees and what the stream consumer sees can
// never disagree."
//
// This file is the deterministic repro both judges converged on: a
// transcript ending in `data: [DONE]` where the caller reads exactly the
// ResponseStart event, cancels, and never reads again.
package openaicompat_test

import (
	"context"
	"testing"
	"time"

	"go.opentelemetry.io/otel/codes"

	"github.com/cachicamas/backend/agent/src/agenttest/tracetest"
	"github.com/cachicamas/backend/agent/src/ai"
)

// ai37TerminalRaceTranscript is the same minimal well-formed completion
// shape "success_done" / "unretried_success" already use elsewhere in
// this milestone's own tests: one terminal chunk (finish reason "stop",
// no content — so no text block is ever opened), then the sentinel. That
// shape emits exactly two events -- ResponseStart, then Completion --
// which is what makes "read exactly the first event, then stop" land
// precisely on the completion send.
const ai37TerminalRaceTranscript = "data: {\"id\":\"c\",\"model\":\"m\",\"object\":\"chat.completion.chunk\",\"created\":1700000000,\"choices\":[{\"index\":0,\"delta\":{},\"finish_reason\":\"stop\"}]}\n\ndata: [DONE]\n\n"

// TestAI37_TerminalCompletionSend_LosingTheRaceRecordsFailure is the
// CRITICAL fix's own falsifier. The carrier is unbuffered
// (streamCarrierBuffer == 0), so run's completion send blocks on its own
// select until either a reader arrives (this test never provides a
// second one) or ctx.Done() wins. Cancelling after the one read and then
// never reading again reproduces the deterministic loss both judges
// named.
func TestAI37_TerminalCompletionSend_LosingTheRaceRecordsFailure(t *testing.T) {
	t.Parallel()

	provider := tracetest.NewProvider()
	server := ai37SSEServer(t, ai37TerminalRaceTranscript)
	c := ai37MustClient(t, server.URL, provider)
	ctx, cancel := context.WithCancel(context.Background())

	ch, err := c.Stream(ctx, ai37ValidRequest(t))
	if err != nil {
		t.Fatalf("Stream() error = %v, want nil", err)
	}

	first, ok := <-ch
	if !ok {
		t.Fatal("channel closed before the first event, want ResponseStart")
	}
	if first.Kind() != ai.EventKindResponseStart {
		t.Fatalf("first event kind = %v, want ResponseStart", first.Kind())
	}

	// Deliberately stop draining here -- never read ch again -- then
	// cancel. Reading again would race the very send this test exists to
	// observe losing, so the span's own end state is detected by polling
	// the provider instead of touching the channel a second time.
	cancel()

	// Nobody drains ch from here on, so the fix's own best-effort
	// ErrorEvent send blocks on its bounded-wait select for the full
	// emitFailureSendBound (5s, stream.go) before finalizeSpan ever runs
	// -- this poll's own deadline MUST clear that with margin, not race
	// it at the same 5s ai37DrainTimeout other rows in this package use
	// for an actively-draining consumer.
	const ai37TerminalRacePollTimeout = 10 * time.Second
	deadline := time.Now().Add(ai37TerminalRacePollTimeout)
	for {
		if err := provider.AssertAllEndedOnce(); err == nil {
			break
		}
		if time.Now().After(deadline) {
			t.Fatal("timed out waiting for the span to end")
		}
		time.Sleep(2 * time.Millisecond)
	}

	spans := provider.Spans()
	if len(spans) != 1 {
		t.Fatalf("provider recorded %d span(s), want 1", len(spans))
	}
	code, _ := spans[0].Status()
	if code == codes.Ok {
		t.Error("span status = codes.Ok, want codes.Error -- the stream consumer never received the completion, so finalizeSpan must not take the success branch (R-AOB-005, CRITICAL)")
	}

	attrs := spans[0].Attributes()
	if _, ok := ai37AttrByKey(attrs, "gen_ai.response.finish_reasons"); ok {
		t.Error("gen_ai.response.finish_reasons present, want absent -- usage/finish-reason attributes belong to the success branch only, and the consumer never received this completion")
	}
	if _, ok := ai37AttrByKey(attrs, "error.type"); !ok {
		t.Error("error.type absent, want present -- the span must report the terminal-send failure through the same failure branch every other cancellation path uses")
	}
}
