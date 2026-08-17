// AG-13 — the multi-turn run driver. Phase 0 opens the continuation
// seam (TurnOptions.Continuation, R-LSK-001) and the scheduler's
// sink-ownership seam (Scheduler.LeaveSinkOpen, R-TLS-012); Phase 1
// wires the continuation path through Turn (identity/brackets,
// schedule-before-finalize, and the R-HIS-010 transcript commits).
// Later phases (AG-13.1..AG-13.3, the R-APP-002 parked-wait bite) add
// the Harness type itself and its own tests to this same file, so a
// helper added for one phase is available to a later phase's without
// duplication — loop_test.go's own precedent (NFR-LSK-001: every
// scenario in package agent_test).
package agent_test

import (
	"errors"
	"testing"

	"github.com/cachicamas/backend/agent/src/agent"
	"github.com/cachicamas/backend/agent/src/ai"
)

// AG-13.0 — S-LSK-014. Given a TurnOptions carrying a continuation
// with any one member absent, when Turn runs, then it returns a typed
// rejection naming the absent member's position, no event whatsoever
// is emitted on sink, and the sink is left in the state the caller
// gave it — a half-configured continuation never produces a partial
// stream. Table-driven over all four members (triangulation: the
// "naming the absent member" property must hold for each, not just
// whichever one a hardcoded check happens to catch).
func TestTurn_ContinuationHalfConfigured_RejectedTypedNoEmission(t *testing.T) {
	t.Parallel()

	full := func() *agent.TurnContinuation {
		return &agent.TurnContinuation{
			Run:       "run-lsk-014",
			Stamper:   &agent.LaneStamper{},
			Scheduler: &agent.Scheduler{},
			History:   agent.NewHistory(),
		}
	}

	tests := []struct {
		name    string
		mutate  func(*agent.TurnContinuation)
		wantPos string
	}{
		{"run absent", func(c *agent.TurnContinuation) { c.Run = "" }, "continuation.run"},
		{"stamper absent", func(c *agent.TurnContinuation) { c.Stamper = nil }, "continuation.stamper"},
		{"scheduler absent", func(c *agent.TurnContinuation) { c.Scheduler = nil }, "continuation.scheduler"},
		{"history absent", func(c *agent.TurnContinuation) { c.History = nil }, "continuation.history"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			cont := full()
			tt.mutate(cont)
			sink := make(chan *agent.Event, 16)

			_, _, err := agent.Turn(
				contextBackground(),
				nil, // MUST never be reached — validation happens before any provider interaction.
				"system prompt for lsk-014",
				[]ai.Message{firstMessage(t)},
				agent.TurnOptions{Continuation: cont},
				sink,
			)
			if err == nil {
				t.Fatal("Turn returned err = nil, want a typed rejection for a half-configured continuation")
			}
			if !errors.Is(err, ai.ErrEmpty) {
				t.Errorf("Turn error rule class = %v, want errors.Is(err, ai.ErrEmpty)", err)
			}
			var violation *ai.Violation
			if !errors.As(err, &violation) {
				t.Fatalf("Turn error = %T, want errors.As to reach *ai.Violation", err)
			}
			if got := violation.Path().String(); got != tt.wantPos {
				t.Errorf("violation position = %q, want %q (naming the absent member)", got, tt.wantPos)
			}

			// No event whatsoever on sink: a non-blocking receive
			// must hit default — no data ready, and (proven below)
			// not closed either.
			select {
			case ev, ok := <-sink:
				t.Fatalf("sink carries an event/close signal (ev=%v, ok=%v), want zero events and an untouched channel", ev, ok)
			default:
			}

			// The sink is left in the state the caller gave it —
			// Turn must not have closed it. Closing it here
			// ourselves must not panic; a panic would prove Turn
			// already closed it.
			func() {
				defer func() {
					if r := recover(); r != nil {
						t.Fatalf("closing sink after Turn panicked (%v) — Turn already closed it, want the sink left untouched", r)
					}
				}()
				close(sink)
			}()
		})
	}
}
