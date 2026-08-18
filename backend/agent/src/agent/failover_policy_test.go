// AG-15.3 — strict-TDD tests for the failover seam (R-RTY-010..012,
// S-RTY-013, S-RTY-014). package agent_test throughout (NFR-RTY-001).
package agent_test

import (
	"context"
	"errors"
	"os"
	"strings"
	"sync"
	"testing"

	"github.com/cachicamas/backend/agent/src/agent"
	"github.com/cachicamas/backend/agent/src/agenttest"
	"github.com/cachicamas/backend/agent/src/ai"
)

// recordingFailoverPolicy is a test-local agent.FailoverPolicy that
// records every Resolve call, in order, and always declines (the
// zero FailoverVerdict — v1's only constructible value).
type recordingFailoverPolicy struct {
	mu     sync.Mutex
	prompt []agent.FailoverPrompt
}

func (p *recordingFailoverPolicy) Resolve(_ context.Context, prompt agent.FailoverPrompt) agent.FailoverVerdict {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.prompt = append(p.prompt, prompt)
	return agent.FailoverVerdict{}
}

func (p *recordingFailoverPolicy) Calls() []agent.FailoverPrompt {
	p.mu.Lock()
	defer p.mu.Unlock()
	out := make([]agent.FailoverPrompt, len(p.prompt))
	copy(out, p.prompt)
	return out
}

// TestFailover_ConsultedOnceAtExhaustion — S-RTY-013, charter AG-15.3
// scenario 1. Given a recording failover policy installed on the
// harness and a provider wrapper failing retryably pre-stream on more
// calls than H, when the run is driven to its terminal, then the
// policy's consult method is called exactly once, with an attempt
// count equal to H and the final attempt's typed failure; the policy
// is not consulted at all in a second run whose failure is
// non-retryable and therefore surfaces at G2; and the shipped
// declining implementation's documentation names the real
// implementation's obligations.
func TestFailover_ConsultedOnceAtExhaustion(t *testing.T) {
	t.Parallel()

	t.Run("consulted exactly once with H and the final failure, at exhaustion", func(t *testing.T) {
		t.Parallel()

		const bound = 3
		failure := mustPreStreamRetryableFailure(t, ai.RetryDelay{})
		inner := agenttest.NewProvider(scriptTextResponse(t, ai.FinishReasonStop))
		provider := newPreStreamFailingProvider(bound+5, failure, inner)

		policy := &recordingFailoverPolicy{}
		h := agent.Harness{
			Provider:      provider,
			System:        "system prompt for failover consulted once",
			History:       agent.NewHistory(),
			RetryAttempts: bound,
			RetryTiming:   agent.RetryTiming{SleepFunc: instantSleep},
			Failover:      policy,
		}

		sink := make(chan *agent.Event, 128)
		_, _, err := h.Run(contextBackground(), firstMessage(t), sink)
		if err == nil {
			t.Fatal("Run returned err = nil, want the exhausted-retry failure")
		}
		_ = drainSink(t, sink)

		calls := policy.Calls()
		if len(calls) != 1 {
			t.Fatalf("Resolve called %d time(s), want exactly 1 (consulted once at exhaustion)", len(calls))
		}
		if calls[0].Attempts != bound {
			t.Errorf("FailoverPrompt.Attempts = %d, want %d (H)", calls[0].Attempts, bound)
		}
		if calls[0].Failure == nil {
			t.Fatal("FailoverPrompt.Failure is nil, want the final attempt's typed failure")
		}
		var wantFailure *ai.Failure
		if !errors.As(err, &wantFailure) {
			t.Fatal("Run's returned error does not reach an *ai.Failure via errors.As")
		}
		if calls[0].Failure != wantFailure {
			t.Error("FailoverPrompt.Failure is not the identical *ai.Failure pointer Run's error unwraps to")
		}
	})

	t.Run("never consulted when the failure is non-retryable (surfaces at G2)", func(t *testing.T) {
		t.Parallel()

		nonRetryable, ferr := ai.PreStreamFailure(ai.FailureReport{
			Category:  ai.FailureCategoryUnavailable,
			Retryable: false,
		})
		if ferr != nil {
			t.Fatalf("ai.PreStreamFailure returned %v, want no failure", ferr)
		}
		inner := agenttest.NewProvider(scriptTextResponse(t, ai.FinishReasonStop))
		provider := newPreStreamFailingProvider(5, nonRetryable, inner)

		policy := &recordingFailoverPolicy{}
		h := agent.Harness{
			Provider:      provider,
			System:        "system prompt for failover never consulted at G2",
			History:       agent.NewHistory(),
			RetryAttempts: 3,
			RetryTiming:   agent.RetryTiming{SleepFunc: instantSleep},
			Failover:      policy,
		}

		sink := make(chan *agent.Event, 64)
		_, _, err := h.Run(contextBackground(), firstMessage(t), sink)
		if err == nil {
			t.Fatal("Run returned err = nil, want the non-retryable failure")
		}
		_ = drainSink(t, sink)

		if calls := policy.Calls(); len(calls) != 0 {
			t.Errorf("Resolve called %d time(s), want 0 (G2 surfaces immediately, never consulting the seam)", len(calls))
		}
	})
}

// TestFailoverPolicy_DocumentsRealImplementationObligations —
// S-RTY-013's second Given/Then: the interface's documentation names
// re-counting the context budget and restarting the cache prefix as
// the real, post-v1 implementation's obligations (R-RTY-010,
// AGS-D). Read as source text -- the obligation is documentation, not
// code (design AD-7).
func TestFailoverPolicy_DocumentsRealImplementationObligations(t *testing.T) {
	t.Parallel()

	data, err := os.ReadFile("failover_policy.go")
	if err != nil {
		t.Fatalf("os.ReadFile(%q) returned %v, want no error", "failover_policy.go", err)
	}
	content := string(data)

	for _, want := range []string{"context budget", "cache prefix"} {
		if !strings.Contains(content, want) {
			t.Errorf("failover_policy.go's FailoverPolicy documentation is missing the obligation phrase %q (R-RTY-010)", want)
		}
	}
}

// TestFailover_InertnessNilVsNoOp — S-RTY-014, charter AG-15.3
// scenario 2. One failing fixture defined once and driven twice --
// first on a harness whose Failover field is nil, then on a harness
// carrying the shipped declining implementation -- must be
// indistinguishable to a consumer: the same event kinds, outcomes and
// failure evidence in the same order, and the same returned error
// (R-RTY-011).
func TestFailover_InertnessNilVsNoOp(t *testing.T) {
	t.Parallel()

	const bound = 3
	failure := mustPreStreamRetryableFailure(t, ai.RetryDelay{})

	run := func(t *testing.T, failover agent.FailoverPolicy) ([]agent.Event, error) {
		t.Helper()
		inner := agenttest.NewProvider(scriptTextResponse(t, ai.FinishReasonStop))
		provider := newPreStreamFailingProvider(bound+5, failure, inner)
		h := agent.Harness{
			Provider:      provider,
			System:        "system prompt for failover inertness pin",
			History:       agent.NewHistory(),
			RetryAttempts: bound,
			RetryTiming:   agent.RetryTiming{SleepFunc: instantSleep},
			Failover:      failover,
		}
		sink := make(chan *agent.Event, 128)
		_, _, err := h.Run(contextBackground(), firstMessage(t), sink)
		got := drainSink(t, sink)
		return got, err
	}

	nilEvents, nilErr := run(t, nil)
	noOpEvents, noOpErr := run(t, agent.NoOpFailoverPolicy{})

	if len(nilEvents) != len(noOpEvents) {
		t.Fatalf("event count: nil Failover = %d, NoOpFailoverPolicy = %d, want equal", len(nilEvents), len(noOpEvents))
	}
	for i := range nilEvents {
		a, b := nilEvents[i], noOpEvents[i]
		if a.Kind() != b.Kind() {
			t.Errorf("event[%d].Kind(): nil = %v, NoOp = %v, want equal", i, a.Kind(), b.Kind())
		}
		_, aHasTurn := a.Turn()
		_, bHasTurn := b.Turn()
		if aHasTurn != bHasTurn {
			t.Errorf("event[%d] turn-scoped: nil = %v, NoOp = %v, want equal", i, aHasTurn, bHasTurn)
		}
		if re, ok := a.RunEnd(); ok {
			reB, okB := b.RunEnd()
			if !okB || re.Outcome() != reB.Outcome() {
				t.Errorf("event[%d] run_end outcome mismatch", i)
			}
			fa, hasA := re.Failure()
			fb, hasB := reB.Failure()
			if hasA != hasB {
				t.Errorf("event[%d] run_end failure presence: nil = %v, NoOp = %v", i, hasA, hasB)
			}
			if hasA && hasB && (fa.Category() != fb.Category() || fa.Retryable() != fb.Retryable()) {
				t.Errorf("event[%d] run_end failure evidence mismatch", i)
			}
		}
		if te, ok := a.TurnEnd(); ok {
			teB, okB := b.TurnEnd()
			if !okB || te.Outcome() != teB.Outcome() {
				t.Errorf("event[%d] turn_end outcome mismatch", i)
			}
			fa, hasA := te.Failure()
			fb, hasB := teB.Failure()
			if hasA != hasB {
				t.Errorf("event[%d] turn_end failure presence: nil = %v, NoOp = %v", i, hasA, hasB)
			}
			if hasA && hasB && (fa.Category() != fb.Category() || fa.Retryable() != fb.Retryable() || fa.PartialOutput() != fb.PartialOutput()) {
				t.Errorf("event[%d] turn_end failure evidence mismatch", i)
			}
		}
	}

	if (nilErr == nil) != (noOpErr == nil) {
		t.Fatalf("returned error nil-ness: nil Failover = %v, NoOp = %v, want equal", nilErr, noOpErr)
	}
	for _, sentinel := range []error{agent.ErrInterrupted, agent.ErrShutdown} {
		if errors.Is(nilErr, sentinel) != errors.Is(noOpErr, sentinel) {
			t.Errorf("errors.Is(_, %v) differs between nil Failover and NoOp", sentinel)
		}
	}
	var nilFailure, noOpFailure *ai.Failure
	nilHas := errors.As(nilErr, &nilFailure)
	noOpHas := errors.As(noOpErr, &noOpFailure)
	if nilHas != noOpHas {
		t.Fatalf("errors.As to *ai.Failure: nil Failover = %v, NoOp = %v, want equal", nilHas, noOpHas)
	}
	if nilHas && noOpHas && nilFailure.Category() != noOpFailure.Category() {
		t.Errorf("failure category: nil Failover = %v, NoOp = %v, want equal", nilFailure.Category(), noOpFailure.Category())
	}

	nilReport := agent.CheckStream(nilEvents)
	if v := nilReport.Violation(); v != nil {
		t.Errorf("CheckStream rejected the nil-Failover stream: %v, want it accepted unmodified", v)
	}
	noOpReport := agent.CheckStream(noOpEvents)
	if v := noOpReport.Violation(); v != nil {
		t.Errorf("CheckStream rejected the NoOpFailoverPolicy stream: %v, want it accepted unmodified", v)
	}
}
