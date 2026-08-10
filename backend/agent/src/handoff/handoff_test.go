// AI-40.1 — the consumer proof: a tiny external-package test standing in
// for the future Layer 2 in miniature (R-L2H-001, design AD-1).
//
// This file lives in package handoff_test — neither the neutral contract
// package (src/ai) nor the test-support package (src/agenttest) — so its
// own import closure proves what a genuine third-party consumer's would
// be: a request built, a fake provider driven, a stream drained to its
// terminal event, a scripted error inspected through the neutral failure
// surface only, and a cancellation honored within a bounded deadline.
// Its zero-vendor-import property is established by the existing
// module-wide boundary guard (import_boundary_test.go), never by a new
// guard or by inspection (R-L2H-001).
package handoff_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/cachicamas/backend/agent/src/agenttest"
	"github.com/cachicamas/backend/agent/src/ai"
)

// boundedWait bounds every select-with-timeout in this file, so a
// producer defect that never signals fails the test instead of hanging
// it (matching this module's own boundedFakeTimeout convention,
// src/agenttest/fake_text_test.go).
const boundedWait = 2 * time.Second

// mustConsumerRequest builds one minimal, valid request — the shape any
// Layer 2 caller assembles before invoking a provider — failing the test
// loudly on any construction error rather than silently proceeding with
// a zero Request.
func mustConsumerRequest(t *testing.T) ai.Request {
	t.Helper()

	part, err := ai.NewText("What is the capital of France?")
	if err != nil {
		t.Fatalf("ai.NewText returned %v, want no failure", err)
	}
	message, err := ai.NewMessage(ai.RoleUser, part)
	if err != nil {
		t.Fatalf("ai.NewMessage returned %v, want no failure", err)
	}
	req, err := ai.NewRequest("consumer-model", []ai.Message{message})
	if err != nil {
		t.Fatalf("ai.NewRequest returned %v, want no failure", err)
	}
	return req
}

// TestHandoff_ConsumerProof is AI-40.1's whole deliverable: the future
// Layer 2 in miniature, driving the AI-21 fake exactly as a third-party
// consumer would (R-L2H-001). Its three subtests are independent
// scenarios, not steps of one flow: a full drain, a scripted terminal
// error, and a bounded cancellation.
func TestHandoff_ConsumerProof(t *testing.T) {
	t.Run("drains a scripted stream", func(t *testing.T) {
		t.Parallel()

		req := mustConsumerRequest(t)

		start, err := ai.NewTextBlockStart(1)
		if err != nil {
			t.Fatalf("ai.NewTextBlockStart returned %v, want no failure", err)
		}
		delta, err := ai.NewTextDelta(1, "Paris")
		if err != nil {
			t.Fatalf("ai.NewTextDelta returned %v, want no failure", err)
		}
		end, err := ai.NewTextBlockEnd(1)
		if err != nil {
			t.Fatalf("ai.NewTextBlockEnd returned %v, want no failure", err)
		}
		completion, err := ai.NewCompletion(ai.FinishReasonStop, ai.Usage{})
		if err != nil {
			t.Fatalf("ai.NewCompletion returned %v, want no failure", err)
		}

		provider := agenttest.NewProvider(agenttest.Script{
			Steps: []agenttest.Step{
				agenttest.Emit(start),
				agenttest.Emit(delta),
				agenttest.Emit(end),
				agenttest.Emit(completion),
			},
		})

		ch, err := provider.Stream(context.Background(), req)
		if err != nil {
			t.Fatalf("Stream returned %v, want no failure", err)
		}

		wantKinds := []ai.EventKind{
			ai.EventKindTextBlockStart,
			ai.EventKindTextDelta,
			ai.EventKindTextBlockEnd,
			ai.EventKindCompletion,
		}
		var gotKinds []ai.EventKind
		for event := range ch {
			gotKinds = append(gotKinds, event.Kind())
		}

		if len(gotKinds) != len(wantKinds) {
			t.Fatalf("drained %d events %v, want %d events %v", len(gotKinds), gotKinds, len(wantKinds), wantKinds)
		}
		for i, want := range wantKinds {
			if gotKinds[i] != want {
				t.Errorf("event %d kind = %v, want %v", i, gotKinds[i], want)
			}
		}

		requests := provider.Requests()
		if len(requests) != 1 {
			t.Fatalf("len(Requests()) = %d, want 1", len(requests))
		}
		if !requests[0].Equal(req) {
			t.Errorf("Requests()[0] = %v, does not echo the sent request %v", requests[0], req)
		}
	})

	t.Run("surfaces a scripted terminal error", func(t *testing.T) {
		t.Parallel()

		req := mustConsumerRequest(t)

		start, err := ai.NewTextBlockStart(1)
		if err != nil {
			t.Fatalf("ai.NewTextBlockStart returned %v, want no failure", err)
		}
		end, err := ai.NewTextBlockEnd(1)
		if err != nil {
			t.Fatalf("ai.NewTextBlockEnd returned %v, want no failure", err)
		}
		failure, err := ai.MidStreamFailure(ai.FailureReport{
			Category:  ai.FailureCategoryUnavailable,
			Retryable: true,
		}, true)
		if err != nil {
			t.Fatalf("ai.MidStreamFailure returned %v, want no failure", err)
		}
		errEvent, err := ai.ErrorEvent(failure)
		if err != nil {
			t.Fatalf("ai.ErrorEvent returned %v, want no failure", err)
		}

		provider := agenttest.NewProvider(agenttest.Script{
			Steps: []agenttest.Step{
				agenttest.Emit(start),
				agenttest.Emit(end),
				agenttest.Emit(errEvent),
			},
		})

		ch, err := provider.Stream(context.Background(), req)
		if err != nil {
			t.Fatalf("Stream returned %v, want no failure", err)
		}

		var events []ai.Event
		for event := range ch {
			events = append(events, event)
		}
		if len(events) == 0 {
			t.Fatal("drained zero events, want at least the scripted terminal error")
		}

		last := events[len(events)-1]
		if last.Kind() != ai.EventKindError {
			t.Fatalf("last drained event kind = %v, want %v", last.Kind(), ai.EventKindError)
		}

		// Inspected only through the neutral failure surface — never a
		// vendor type (S-L2H-003).
		payload, ok := last.ErrorPayload()
		if !ok {
			t.Fatal("last event carries no error payload")
		}
		if got := payload.Category(); got != ai.FailureCategoryUnavailable {
			t.Errorf("Category() = %v, want %v", got, ai.FailureCategoryUnavailable)
		}
		if !payload.Retryable() {
			t.Error("Retryable() = false, want true")
		}
	})

	t.Run("exits boundedly on cancellation", func(t *testing.T) {
		t.Parallel()

		req := mustConsumerRequest(t)

		// The Hold is the script's only step: ai.CheckStream validates a
		// script's full event list up front, before any goroutine starts
		// (fake_provider.go), and would reject a block opened by this
		// script but never closed — even one abandoned past a Hold — as
		// malformed. A held stream with no preceding block-scoped event
		// is the well-formed way to script "cancelled before anything
		// closes".
		gate := agenttest.NewGate()

		provider := agenttest.NewProvider(agenttest.Script{
			Steps: []agenttest.Step{
				agenttest.Hold(gate),
			},
		})

		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		ch, err := provider.Stream(ctx, req)
		if err != nil {
			t.Fatalf("Stream returned %v, want no failure", err)
		}

		select {
		case <-gate.Reached():
		case <-time.After(boundedWait):
			t.Fatal("producer never reached the Hold step within the bound")
		}

		cancel()
		if !errors.Is(ctx.Err(), context.Canceled) {
			t.Fatalf("ctx.Err() = %v, want context.Canceled", ctx.Err())
		}

		// The channel must close within a bounded deadline, attributed to
		// cancellation, with no terminal event forced through — the
		// sanctioned loss path (S-L2H-004).
		select {
		case event, ok := <-ch:
			if ok {
				t.Fatalf("received %v after cancellation, want the channel to close with no terminal event forced", event.Kind())
			}
		case <-time.After(boundedWait):
			t.Fatal("channel did not close within the bound after cancellation")
		}
	})
}
