// CH-02.1 — the projector: Layer 2's event sink projected onto WireEvent
// values (R-CCP-003, R-CCP-004, R-CCP-013). Its only inputs are the sink and
// Harness.Run's own return values — no second channel, no Layer 2 internal
// read.
package chat

import (
	"context"
	"log/slog"

	"github.com/cachicamas/backend/agent/src/agent"
	"github.com/cachicamas/backend/agent/src/ai"
)

// runResult carries Harness.Run's own return values from the runner
// goroutine to the projector goroutine across a buffered channel — the
// synchronization primitive that makes the cross-goroutine handoff
// race-free under -race (S-CCP-055).
type runResult struct {
	finish ai.FinishReason
	err    error
}

// project ranges sink, projecting exactly the four mapped kinds (R-CCP-003)
// and logging every other registered kind once per occurrence, on the
// switch's default branch so a kind Layer 2 registers later is logged
// automatically rather than silently dropped (R-CCP-004, D4). It holds the
// run_end payload — turn.end is projected from run_end, never from turn_end
// (D1) — until the range ends, then reads result to complete the ONE
// terminal wire event this turn emits (R-CCP-006), clears inFlight, and
// closes out exactly once.
func (c *Conversation) project(ctx context.Context, sink <-chan *agent.Event, result <-chan runResult, out chan<- WireEvent) {
	defer close(out)

	msgIndex := -1
	var runEnd agent.RunEnd
	var haveRunEnd bool

	for ev := range sink {
		switch ev.Kind() {
		case agent.EventKindMessageStartText:
			start, _ := ev.MessageStartText()
			msgIndex++
			out <- MessageStart{MessageID: start.MessageID().String(), Index: msgIndex}

		case agent.EventKindMessageDeltaText:
			delta, _ := ev.MessageDeltaText()
			out <- MessageDelta{Index: msgIndex, Delta: delta.Fragment()}

		case agent.EventKindMessageEndText:
			out <- MessageEnd{Index: msgIndex, FinishReason: FinishReasonStop}

		case agent.EventKindRunEnd:
			// Held, not projected here: the terminal wire event is built
			// at the single site below, once Harness.Run's own returned
			// finish reason is also known (D5).
			runEnd, haveRunEnd = ev.RunEnd()

		default:
			c.logger.LogAttrs(ctx, slog.LevelInfo, "unmapped agent event",
				slog.String("kind", ev.Kind().String()),
				slog.String("run", string(ev.Run())))
		}
	}

	res := <-result
	out <- terminalWireEvent(runEnd, haveRunEnd, res)

	c.mu.Lock()
	c.inFlight = false
	c.mu.Unlock()
}

// terminalWireEvent builds the turn's ONE terminal wire event from the held
// run_end payload and Harness.Run's own returned finish reason (R-CCP-006).
// haveRunEnd is defensive only: Harness.Run always emits exactly one
// run_end before its sink closes, on every exit path.
func terminalWireEvent(re agent.RunEnd, haveRunEnd bool, res runResult) WireEvent {
	if !haveRunEnd {
		return TurnEnd{}
	}
	switch re.Outcome() {
	case agent.RunOutcomeCompleted:
		reason := finishReasonToWire(res.finish)
		return TurnEnd{FinishReason: &reason}
	case agent.RunOutcomeInterrupted, agent.RunOutcomeShutdown:
		// windDownRun returns the zero ai.FinishReason on interrupt
		// (harness.go:396), and the wire's cancellation discriminator is
		// FinishReason's ABSENCE — never a minted placeholder (R-CCP-006,
		// D5): a completed run always carries a real reason, so absence is
		// the truthful cancellation attribution.
		return TurnEnd{FinishReason: nil}
	default:
		// agent.RunOutcomeFailed (CH-02.3, Phase 5) lands its own
		// Error{...} mapping (R-CCP-007, D6); until Phase 5 replaces this
		// branch, a failed run is provisionally indistinguishable from an
		// interrupted one here.
		return TurnEnd{FinishReason: nil}
	}
}
