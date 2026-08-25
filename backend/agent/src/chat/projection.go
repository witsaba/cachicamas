// CH-02.1 — the projector: Layer 2's event sink projected onto WireEvent
// values (R-CCP-003, R-CCP-004, R-CCP-013). Its only inputs are the sink and
// Harness.Run's own return values — no second channel, no Layer 2 internal
// read.
//
// CH-06 — widens the projector to track assistantText (accumulated from
// MessageDelta text, R-CCS-004) and messageIDs (pushed from MessageStart
// MessageID, R-CCS-009), and to call c.store.Append at the terminal
// wire event site (R-CCS-008, D-6) BEFORE clearing inFlight.

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
//
// CH-06: prompt is the user-submitted prompt that drives this turn
// (D-7). The projector tracks assistantText (accumulated from
// MessageDelta.Fragment, R-CCS-004) and messageIDs (pushed from
// MessageStart.MessageID, R-CCS-009). Between the terminal wire event
// send and the inFlight clear, the projector builds an Exchange and
// appends it to c.store — the append runs BEFORE inFlight clears
// (R-CCS-008, D-6) so a subscriber that receives the terminal wire
// event and fires a fast reload sees the just-finished exchange.
func (c *Conversation) project(ctx context.Context, prompt string, sink <-chan *agent.Event, result <-chan runResult, out chan<- WireEvent) {
	defer close(out)

	msgIndex := -1
	var runEnd agent.RunEnd
	var haveRunEnd bool

	// CH-06 accumulators (D-7, R-CCS-004 / R-CCS-009).
	var assistantText string
	var messageIDs []string

	// CH-10 (R-CPM-008, D-8): the per-wireCallId "denied" set the
	// projector populates on permission_decision_made{deny} and
	// consults in the tool.result arm to suppress the wire event
	// for the same wireCallId (the F-CPM-002/003 closure at
	// design time). In T-02 the set is declared but stays empty
	// because no projection arm yet populates it; T-04 finalizes
	// the population rule + the consultation suppression. Declared
	// above the switch so the consultation site compiles.
	deniedSet := map[string]bool{}

	for ev := range sink {
		switch ev.Kind() {
		case agent.EventKindMessageStartText:
			start, _ := ev.MessageStartText()
			msgIndex++
			messageIDs = append(messageIDs, start.MessageID().String())
			out <- MessageStart{MessageID: start.MessageID().String(), Index: msgIndex}

		case agent.EventKindMessageDeltaText:
			delta, _ := ev.MessageDeltaText()
			assistantText += delta.Fragment()
			out <- MessageDelta{Index: msgIndex, Delta: delta.Fragment()}

		case agent.EventKindMessageEndText:
			out <- MessageEnd{Index: msgIndex, FinishReason: FinishReasonStop}

		// CH-09 — Layer 2 tool-event family projection (T-04 GREEN;
		// R-CTS-004, R-CTS-005, D-3, D-6, NFR-CTS-002). Five arms
		// project Layer 2's 5-event-per-call tool bracket model:
		//
		//   - EventKindToolStart → ToolCallStart (wire id, tool name,
		//     arguments bytes → string)
		//   - EventKindToolProgress → DROPPED at the chat wire (D-6
		//     collapse). No explicit case; falls through the
		//     switch's default arm's "unmapped agent event" log.
		//     Keeping NO explicit case is part of the deliberate
		//     posture (NFR-CTS-002).
		//   - EventKindToolEndSuccess → ToolResult{Outcome:"success"}
		//   - EventKindToolEndResultFailure → ToolResult{Outcome:"result_failure"}
		//   - EventKindToolEndExecutionFailure → ToolResult{Outcome:"execution_failure",
		//     FailureCategory:failure.Category().String()}; Content is
		//     empty — no provider text leaks (R-CCP-008 / D6 mirror).
		//
		// WireCallID carries the same value byte-for-byte from
		// Layer 2's start call id to the chat-side ToolResult so
		// the frontend can correlate ToolCallStart ↔ ToolResult
		// (S-CTS-010..012).
		case agent.EventKindToolStart:
			start, _ := ev.ToolStart()
			out <- ToolCallStart{
				WireCallID: start.CallID(),
				Tool:       start.Name(),
				Arguments:  string(start.Arguments()),
			}

		case agent.EventKindToolProgress:
			// D-6 / NFR-CTS-002 — ToolProgress is DROPPED at the chat
			// wire. Deliberately no-op: emitting nothing on this
			// branch is the production posture. The default arm
			// below still records one "unmapped agent event" log so
			// an operator can see ToolProgress landed if tracing.
			_ = ev

		case agent.EventKindToolEndSuccess:
			end, _ := ev.ToolEndSuccess()
			out <- ToolResult{
				WireCallID: end.CallID(),
				Outcome:    "success",
				Content:    string(end.Result()),
			}

		case agent.EventKindToolEndResultFailure:
			end, _ := ev.ToolEndResultFailure()
			out <- ToolResult{
				WireCallID: end.CallID(),
				Outcome:    "result_failure",
				Content:    string(end.Result()),
			}

		case agent.EventKindToolEndExecutionFailure:
			// CH-10 RED scaffold #2 (R-CPM-008, D-8) — the
			// deny-state collapse site. In T-02 the consultation
			// rule is a placeholder; the deniedSet stays empty
			// because no projection arm yet populates it. T-04
			// finalizes the suppression rule: when the prior
			// event for this wireCallId was
			// permission_decision_made{deny}, this arm SUPPRESSES
			// emission. Until T-04 the production projection
			// still emits the tool.result wire event normally.
			end, _ := ev.ToolEndExecutionFailure()
			failure, _ := end.Failure()
			_ = deniedSet // T-04 reads this map; T-02 declares it
			out <- ToolResult{
				WireCallID:      end.CallID(),
				Outcome:         "execution_failure",
				Content:         "",
				FailureCategory: failure.Category().String(),
			}

		// CH-10 (R-CPM-003, D-3) — permission event family projection
		// RED scaffold #2. Three arms pre-empt T-04's GREEN:
		//
		//   - EventKindPermissionDecisionRequired → PermissionDecisionRequired (placeholder)
		//   - EventKindPermissionDecisionMade     → PermissionDecisionMade (placeholder;
		//                                            T-04 also populates deniedSet on Deny)
		//   - EventKindPermissionResolutionRemembered → DROPPED at the chat wire
		//                                            (D-12 collapse: no chat-side
		//                                            vocabulary; falls through the
		//                                            switch's default arm's
		//                                            "unmapped agent event" log)
		//
		// In T-02 each arm is a no-op placeholder so the projector
		// compiles and the wireFrameName switch stays exhaustive
		// once T-04 lands. The strict-TDD RED scaffold pre-empts
		// the GREEN; S-CPM-007/008/009 close at T-04.
		case agent.EventKindPermissionDecisionRequired:
			// Placeholder body; T-04 projects this to a real
			// PermissionDecisionRequired wire event.
			_ = ev

		case agent.EventKindPermissionDecisionMade:
			// Placeholder body; T-04 projects this to a real
			// PermissionDecisionMade wire event AND populates
			// deniedSet when Layer 2's outcome is Deny or
			// ModifyInput (the D-12 chat-side collapse).
			_ = ev

		case agent.EventKindPermissionResolutionRemembered:
			// D-12 wire collapse — DROPPED at the chat wire.
			// The closed chat vocabulary has no
			// "permission.decision.remembered" variant; chat
			// does not surface rule persistence. The default
			// arm below still records one "unmapped agent
			// event" log so an operator can see the event
			// landed if tracing.
			_ = ev

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

	// CH-06 (R-CCS-008, D-6): append BEFORE clearing inFlight. The
	// store's own mutex is taken inside Append; the Conversation.mu
	// is taken below. No lock cycles. A subscriber that receives
	// turn.end and fires a fast reload (CH-08.1) sees the
	// just-finished exchange.
	exchange := buildTerminalExchange(prompt, runEnd, haveRunEnd, res, assistantText, messageIDs)
	if err := c.store.Append(c.participantID, exchange); err != nil {
		c.logger.LogAttrs(ctx, slog.LevelError, "conversation store append failed",
			slog.String("participant_id", c.participantID),
			slog.String("error", err.Error()))
	}

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
	case agent.RunOutcomeFailed:
		// The error IS the terminal event; no turn.end follows it
		// (R-CCP-007). Message is composed ONLY from phraseFor's own fixed
		// table — never failure.Unwrap().Error() or any wrapped ai.Failure's
		// composed string (R-CCP-008, D6): no provider-authored text may
		// reach the wire.
		failure, _ := re.Failure()
		return Error{Kind: "server", Message: phraseFor(failure.Category())}
	default:
		return TurnEnd{FinishReason: nil}
	}
}

// buildTerminalExchange translates the held run_end + buffered res +
// accumulators into the eight-field Exchange the store persists (D-7,
// R-CCS-004, R-CCS-005, R-CCS-006, R-CCS-009). It is package-private
// — called only from project() above. Position is left at its zero
// value here; the in-memory adapter (and CH-07's postgres adapter)
// assigns the insertion index on Append.
func buildTerminalExchange(prompt string, runEnd agent.RunEnd, haveRunEnd bool, res runResult, assistantText string, messageIDs []string) Exchange {
	kind := TerminalKindCompleted
	var fin *ai.FinishReason
	var cat ai.FailureCategory
	partial := false

	if !haveRunEnd {
		// Defensive only: Harness.Run always emits exactly one
		// run_end before its sink closes (terminalWireEvent:77-78).
		// Default to completed with empty FinishReason and zero
		// failure category — same shape a fresh Exchange would carry.
		kind = TerminalKindCompleted
	} else {
		switch runEnd.Outcome() {
		case agent.RunOutcomeCompleted:
			reason := res.finish
			fin = &reason
		case agent.RunOutcomeInterrupted, agent.RunOutcomeShutdown:
			kind = TerminalKindCancelled
			partial = true
		case agent.RunOutcomeFailed:
			kind = TerminalKindFailed
			if f, ok := runEnd.Failure(); ok {
				cat = f.Category()
			}
		}
	}

	// Copy the messageIDs slice so caller-side mutation cannot corrupt
	// the persisted record. The slice is short (one entry per
	// message-start in the turn) — the copy is bounded.
	ids := make([]string, len(messageIDs))
	copy(ids, messageIDs)

	return Exchange{
		PromptText:      prompt,
		AssistantText:   assistantText,
		Partial:         partial,
		TerminalKind:    kind,
		FailureCategory: cat,
		FinishReason:    fin,
		MessageIDs:      ids,
	}
}