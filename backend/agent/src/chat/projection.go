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
	"errors"
	"fmt"
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
	// design time).
	deniedSet := map[string]bool{}

	// CH-10 (R-CPM-006): the per-wireCallId tool-name map.
	// Layer 2's PermissionDecisionRequired carries the tool name;
	// the matching PermissionDecisionMade does NOT (the payload
	// is {callID, outcome, modifiedArguments, failure}). The
	// projector captures the tool name on the required event and
	// reads it back on the made event so the persisted
	// PermissionDecisionRecord carries the (R-CPM-006) Tool field.
	toolByCallID := map[string]string{}

	// CH-10 (R-CPM-007, D-6, F-CPM-001 closure): THREE parallel
	// accumulators thread state into buildTerminalExchange. CH-09's
	// F-CPM-001 defect is that `chat/projection.go:208-251`
	// (`buildTerminalExchange`) returned an Exchange with empty
	// ToolCalls / ToolResults (the fields are always zero at
	// production reload). The fix: catch ToolCallStart + ToolResult
	// + ToolEnd* events into slice state, then thread the slices
	// into buildTerminalExchange. CH-10 widens with a third
	// accumulator for permission decisions (the same defect
	// applies). S-CPM-017 / S-CPM-018 close the gap.
	var (
		toolCalls          []ToolCallRecord
		toolResults        []ToolResultRecord
		permissionDecisions []PermissionDecisionRecord
	)

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
			toolCalls = append(toolCalls, ToolCallRecord{
				WireCallID: start.CallID(),
				Tool:       start.Name(),
				Arguments:  string(start.Arguments()),
			})
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
			toolResults = append(toolResults, ToolResultRecord{
				WireCallID: end.CallID(),
				Outcome:    "success",
				Content:    string(end.Result()),
			})
			out <- ToolResult{
				WireCallID: end.CallID(),
				Outcome:    "success",
				Content:    string(end.Result()),
			}

		case agent.EventKindToolEndResultFailure:
			end, _ := ev.ToolEndResultFailure()
			toolResults = append(toolResults, ToolResultRecord{
				WireCallID: end.CallID(),
				Outcome:    "result_failure",
				Content:    string(end.Result()),
			})
			out <- ToolResult{
				WireCallID: end.CallID(),
				Outcome:    "result_failure",
				Content:    string(end.Result()),
			}

		case agent.EventKindToolEndExecutionFailure:
			// CH-10 (R-CPM-008, D-8) — the deny-state collapse
			// site. When the prior event for this wireCallId
			// was permission_decision_made{outcome: "deny"},
			// this arm SUPPRESSES the tool.result wire event
			// (the hold entry alone carries the user-facing
			// surface; no tool entry renders). S-CPM-019.
			end, _ := ev.ToolEndExecutionFailure()
			if deniedSet[end.CallID()] {
				// D-8 suppression — the hold entry's
				// "denied" state already surfaces the
				// outcome; the tool entry is
				// intentionally absent on the wire.
				break
			}
			failure, _ := end.Failure()
			toolResults = append(toolResults, ToolResultRecord{
				WireCallID:      end.CallID(),
				Outcome:         "execution_failure",
				Content:         "",
				FailureCategory: failure.Category().String(),
			})
			out <- ToolResult{
				WireCallID:      end.CallID(),
				Outcome:         "execution_failure",
				Content:         "",
				FailureCategory: failure.Category().String(),
			}

		// CH-10 (R-CPM-003, D-3, D-12) — permission event family
		// projection GREEN (closes T-01 RED scaffold). Three arms
		// project Layer 2's 3-event-per-decision permission model
		// onto the chat wire's 2-variant closed union:
		//
		//   - EventKindPermissionDecisionRequired → PermissionDecisionRequired
		//     (R-CPM-003 / S-CPM-007). The wire payload carries
		//     WireCallID + Tool + Arguments. The frontend
		//     renders a "waiting" hold entry.
		//
		//   - EventKindPermissionDecisionMade → PermissionDecisionMade
		//     (R-CPM-003 / S-CPM-008, D-12 collapse). The chat
		//     wire's CLOSED 2-value vocabulary
		//     "allow_once" | "deny" replaces Layer 2's 4-value
		//     PermissionOutcome: AllowOnce → "allow_once",
		//     AllowAlways → "allow_once" (D-12 collapse),
		//     Deny → "deny", ModifyInput → "deny" (D-12
		//     collapse). The ModifyInput modified arguments
		//     are dropped at the chat boundary (R-CPM-008 /
		//     deferred-register row 7). When Layer 2's outcome
		//     is Deny or ModifyInput, this arm ALSO populates
		//     deniedSet[wireCallId] (D-8 suppress-on-deny
		//     preparation).
		//
		//   - EventKindPermissionResolutionRemembered → DROPPED
		//     at the chat wire (D-12 collapse: no chat-side
		//     vocabulary for remembered rules; the closed chat
		//     surface has no "permission.decision.remembered"
		//     variant). No explicit case; falls through the
		//     switch's default arm's "unmapped agent event"
		//     log path. R-CPM-003 / S-CPM-009.
		case agent.EventKindPermissionDecisionRequired:
			req, _ := ev.PermissionDecisionRequired()
			// Capture the tool name so the matching made event
			// (which doesn't carry Name) can populate the
			// persistence record's Tool field.
			toolByCallID[req.CallID()] = req.Name()
			out <- PermissionDecisionRequired{
				WireCallID: req.CallID(),
				Tool:       req.Name(),
				Arguments:  string(req.Arguments()),
			}

		case agent.EventKindPermissionDecisionMade:
			made, _ := ev.PermissionDecisionMade()
			wireOutcome := collapseOutcome(made.Outcome())
			if wireOutcome == "deny" {
				deniedSet[made.CallID()] = true
			}
			// F-CPM-001 closure (CH-10 R-CPM-007): thread the
			// permission-decision record into the persistence
			// accumulator so production reload surfaces the
			// decision (the same defect that left ToolCalls /
			// ToolResults empty before CH-10's projector fix).
			permissionDecisions = append(permissionDecisions, PermissionDecisionRecord{
				WireCallID: made.CallID(),
				Tool:       toolByCallID[made.CallID()],
				Outcome:    wireOutcome,
			})
			out <- PermissionDecisionMade{
				WireCallID: made.CallID(),
				Outcome:    wireOutcome,
			}

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

	// Fix C (observability): when the run failed, record a
	// structured log carrying the typed category the wire's `event:
	// error` frame is about to surface, plus the bounded safe
	// metadata Failure exposes (raw label, retryable, status class).
	// Without this log line `docker logs cachicamas-agent-chat` is
	// silent on upstream provider failures — the chat app's failure
	// frame is the only signal, and it cannot distinguish upstream
	// 4xx from upstream 5xx from network timeout from a
	// classification bug. The typed category is the discriminator;
	// StatusClass reports the HTTP class when one survived (5xx →
	// upstream, 0/false → no transport response was in hand).
	if res.err != nil {
		var failure *ai.Failure
		if errors.As(res.err, &failure) {
			statusClass, haveStatusClass := failure.StatusClass()
			attrs := []slog.Attr{
				slog.String("participant_id", c.participantID),
				slog.String("category", failure.Category().String()),
				slog.String("vendor_label", failure.RawLabel()),
				slog.Bool("retryable", failure.Retryable()),
			}
			if haveStatusClass {
				attrs = append(attrs, slog.Int("status_class", statusClass))
			}
			c.logger.LogAttrs(ctx, slog.LevelError, "chat turn failed", attrs...)
		} else {
			// Unknown error shape — not an *ai.Failure. The wire
			// envelope still renders Kind:"server" / Message:"The
			// model provider is temporarily unavailable." from
			// terminalWireEvent's own branch, but we need the
			// underlying cause in the operator logs to distinguish
			// harness-level bugs from upstream-driven failures
			// that never went through openaicompat's typed mapper
			// (e.g. a cost-session failure, an interrupted run).
			// Log the dynamic type and the cause text. The cause
			// is the harness's own error, never the user prompt
			// body, so the D3 denylist on provider body excerpts
			// does not apply here.
			c.logger.LogAttrs(ctx, slog.LevelError, "chat turn failed",
				slog.String("participant_id", c.participantID),
				slog.String("category", "uncategorised"),
				slog.String("error_type", fmt.Sprintf("%T", res.err)),
				slog.String("error_message", res.err.Error()),
			)
		}
	}

	out <- terminalWireEvent(runEnd, haveRunEnd, res)

	// CH-06 (R-CCS-008, D-6): append BEFORE clearing inFlight. The
	// store's own mutex is taken inside Append; the Conversation.mu
	// is taken below. No lock cycles. A subscriber that receives
	// turn.end and fires a fast reload (CH-08.1) sees the
	// just-finished exchange.
	//
	// CH-10 (R-CPM-007, D-6, F-CPM-001 closure): the three parallel
	// accumulators thread state into buildTerminalExchange.
	// Without this threading, the persisted Exchange carries
	// zero ToolCalls / ToolResults / PermissionDecisions — the
	// reload surface would render an empty transcript for any
	// turn that included tool or permission activity. The fix
	// makes the chat projector observably equivalent to the
	// live wire for reload purposes. S-CPM-017 / S-CPM-018 close
	// the gap.
	exchange := buildTerminalExchange(prompt, runEnd, haveRunEnd, res, assistantText, messageIDs, toolCalls, toolResults, permissionDecisions)
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

// collapseOutcome translates Layer 2's PermissionOutcome enum to
// the chat wire's CLOSED 2-value vocabulary (D-12, R-CPM-003).
//
//	AllowOnce   → "allow_once"
//	AllowAlways → "allow_once"   (D-12 collapse — chat does not
//	                                surface rule persistence)
//	Deny        → "deny"
//	ModifyInput → "deny"         (D-12 collapse — chat does not
//	                                surface modified arguments;
//	                                modified args are dropped at the
//	                                chat boundary)
//
// Any out-of-vocab value falls back to "deny" (a typed
// refusal rather than a 4th wire token that would break the
// closed vocabulary).
func collapseOutcome(o agent.PermissionOutcome) string {
	switch o {
	case agent.PermissionOutcomeAllowOnce, agent.PermissionOutcomeAllowAlways:
		return "allow_once"
	case agent.PermissionOutcomeDeny, agent.PermissionOutcomeModifyInput:
		return "deny"
	default:
		// Defensive: PermissionDefer (zero) is never reached
		// here (decision_made events validate outcome non-zero
		// at the constructor; permission_protocol.go:202-204).
		// Treat unknown as deny — closed vocab refuses
		// surprises.
		return "deny"
	}
}

// buildTerminalExchange translates the held run_end + buffered res +
// accumulators into the eleven-field Exchange the store persists
// (D-7, R-CCS-004, R-CCS-005, R-CCS-006, R-CCS-009, R-CTS-006,
// R-CPM-006). It is package-private — called only from project()
// above. Position is left at its zero value here; the in-memory
// adapter (and CH-07's postgres adapter) assigns the insertion
// index on Append.
//
// CH-10 (R-CPM-007, D-6, F-CPM-001 closure): the function gains
// three accumulator parameters (toolCalls, toolResults,
// permissionDecisions) — slice state captured during the live
// projector drain. Without these parameters, Exchange.ToolCalls,
// Exchange.ToolResults, and Exchange.PermissionDecisions are
// always zero/nil at production reload (the F-CPM-001 defect;
// same shape applied to CH-10's new field). Threading the
// accumulators closes S-CPM-017 / S-CPM-018.
func buildTerminalExchange(
	prompt string,
	runEnd agent.RunEnd,
	haveRunEnd bool,
	res runResult,
	assistantText string,
	messageIDs []string,
	toolCalls []ToolCallRecord,
	toolResults []ToolResultRecord,
	permissionDecisions []PermissionDecisionRecord,
) Exchange {
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
	// message-start in the turn) — the copy is bounded. The
	// accumulator slices are passed by value (slices are header
	// values); the store's Load path applies its own defensive
	// copy (NFR-CCS-004, NFR-CCS-008, NFR-CCS-009 — same shape
	// for each slice field).
	ids := make([]string, len(messageIDs))
	copy(ids, messageIDs)

	return Exchange{
		PromptText:          prompt,
		AssistantText:       assistantText,
		Partial:             partial,
		TerminalKind:        kind,
		FailureCategory:     cat,
		FinishReason:        fin,
		MessageIDs:          ids,
		ToolCalls:           toolCalls,
		ToolResults:         toolResults,
		PermissionDecisions: permissionDecisions,
	}
}