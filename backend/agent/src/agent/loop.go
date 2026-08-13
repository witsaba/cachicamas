// Package agent is Layer 2 of the cachicamas agent stack. This file
// adds AG-07's walking skeleton: the one-turn function `Turn`, the
// walking-skeleton options struct `TurnOptions`, and the per-call
// identity minters the loop uses to satisfy R-LSK-002 (two sequential
// turns share nothing).
//
// AG-07 is the first milestone where Layer 2 emits events from a live
// loop: AG-04/05/06's stream-validator tests fed the validator hand-built
// events, and AG-07 owns the producer side of AG-01's carrier decision
// (per `openspec/changes/cachicamas-agent-loop-skeleton/design.md`).
// The single public surface is `Turn`: a function form, not a value, so
// AG-13's later `Harness` can wrap it without changing the signature
// (D1a) and so two `Turn(...)` invocations share nothing by construction
// (D1a, R-LSK-002). The function mints a fresh `runID`, `turnID` and
// `LaneStamper` per call (D1a), translates each provider event into a
// bracket on the Layer 2 envelope, and emits the resulting events on
// `sink`, closing it before returning.
//
// # Substrate preservation
//
// AG-07 changes are limited to this file and `loop_test.go`. The
// substrate's envelope, descriptor vocabulary, validator, run / turn /
// message / tool / permission / cost / delegation / compaction families,
// and every-kind-constructible guard stay byte-unchanged (R-LSK-004,
// NFR-LSK-003): 4th consecutive "substrate untouched" milestone.
//
// # Walking-skeleton scope
//
// The function is the thinnest end-to-end path. Tools, hooks, errors
// beyond typed pass-through, permission, retry, context-check, cost
// events are AG-08…AG-18; value-form `Harness` is AG-13; multi-turn
// state is AG-21. AG-07 emits one assistant turn, end to end.
package agent

import (
	"context"
	"strconv"
	"sync/atomic"

	"github.com/cachicamas/backend/agent/src/ai"
)

// TurnOptions is the options struct for a single assistant turn.
// Walking-skeleton scope: trivial/zero fields; AG-08 introduces hooks,
// AG-09 introduces tool opts, AG-11 introduces retry opts.
type TurnOptions struct {
	// Model is the model identifier passed to the provider. Empty = provider default.
	Model string
	// MaxTokens is the optional max-tokens budget. Zero = provider default.
	MaxTokens int
}

// lastLoopRunIDCounter and lastLoopTurnIDCounter are the sources of
// run and turn identities minted at every Turn(...) call (R-LSK-002).
// They are package-level atomic counters for the walking-skeleton
// scope: AG-13's `Harness` will mint them in caller-supplied groups;
// for a single-turn function the simplest correct thing is one
// counter per identity, atomic-guarded for the race detector
// (NFR-LSK-002).
var (
	lastLoopRunIDCounter  atomic.Uint64
	lastLoopTurnIDCounter atomic.Uint64
)

// mintLoopRunID returns a fresh RunID — one no other Turn(...) call in
// this process has minted.
func mintLoopRunID() RunID {
	return RunID("run-lsk-" + strconv.FormatUint(lastLoopRunIDCounter.Add(1), 10))
}

// mintLoopTurnID returns a fresh TurnID — one no other Turn(...) call
// in this process has minted.
func mintLoopTurnID() TurnID {
	return TurnID("turn-lsk-" + strconv.FormatUint(lastLoopTurnIDCounter.Add(1), 10))
}

// mintLoopMessageID mints a fresh ai.MessageID for one bracket.
//
// Layer 1 seals MessageID against forgery (V-REQ-03: the field is
// unexported and minted only inside package ai through NewMessage).
// The loop therefore constructs a placeholder Message through the
// public surface and reads back its minted identity — the placeholder
// message itself is discarded, only the identity is kept (R-AMT-001,
// S-AGV-019: each bracket has its own identity).
//
// Walking-skeleton scope: this is the only mechanism by which the
// loop mints fresh message identities; AG-23 introduces a typed
// minting bridge that does not waste a placeholder.
func mintLoopMessageID() ai.MessageID {
	placeholderPart, _ := ai.NewText("loop-message-id-placeholder")
	placeholderMsg, _ := ai.NewMessage(ai.RoleUser, placeholderPart)
	return placeholderMsg.ID()
}

// Turn runs one assistant turn end-to-end and emits the resulting events
// to sink. The function is stateless across calls: two Turn(...)
// invocations share nothing by construction (no closure captures, no
// shared LaneStamper — each turn mints a fresh one, R-LSK-002). The
// function closes sink before returning.
//
// R-LSK-001..005 / S-LSK-001..007 — see
// openspec/specs/agent-loop-skeleton/spec.md.
func Turn(
	ctx context.Context,
	provider ai.ModelProvider,
	system string,
	transcript []ai.Message,
	opts TurnOptions,
	sink chan<- *Event,
) (ai.Message, ai.FinishReason, error) {
	runID := mintLoopRunID()
	turnID := mintLoopTurnID()
	stamper := &LaneStamper{}

	// Emit run_start and turn_start (R-AEV-002: stamped, contiguous,
	// 1-based; R-LSK-002: per-call stamper, no shared state).
	runStart, err := NewRunStart(runID)
	if err != nil {
		closeSink(sink)
		return ai.Message{}, 0, err
	}
	emitStamped(sink, stamper, runStart)

	turnStart, err := NewTurnStart(runID, turnID)
	if err != nil {
		closeSink(sink)
		return ai.Message{}, 0, err
	}
	emitStamped(sink, stamper, turnStart)

	// Build the request from system + transcript + opts (D5a, R-LSK-001).
	req, err := buildLoopRequest(opts, system, transcript)
	if err != nil {
		closeSink(sink)
		return ai.Message{}, 0, err
	}

	// Drain the provider's channel, translating each event into an
	// agent-level bracket (S-LSK-001, S-LSK-002).
	pCh, streamErr := provider.Stream(ctx, req)
	if streamErr != nil {
		// Pre-stream failure (V-PRV-04): no producer goroutine, no
		// channel. The walking skeleton closes sink and surfaces the
		// typed error to the caller.
		closeSink(sink)
		return ai.Message{}, 0, streamErr
	}

	turn := newTurnAccumulator(runID, turnID, stamper, sink)
	for ev := range pCh {
		if done := turn.translate(ev); done {
			// Completion: capture the finish reason and exit the loop.
			msg, finish := turn.finalize()
			closeSink(sink)
			return msg, finish, nil
		}
		if turn.fatal != nil {
			// Mid-stream terminal error: drain whatever is left so
			// the producer's goroutine is not stranded (S-LSK-002).
			_ = drainProvider(pCh)
			closeSink(sink)
			return ai.Message{}, 0, turn.fatal
		}
	}

	// Provider closed without a Completion: walking-skeleton scope
	// treats this as a "ran to empty" turn — close sink, return.
	closeSink(sink)
	return ai.Message{}, 0, nil
}

// emitStamped stamps ev through the LaneStamper and sends it on sink.
// The LaneStamper is the per-lane counter (R-AEV-002): every Turn
// call mints a fresh one (R-LSK-002).
func emitStamped(sink chan<- *Event, stamper *LaneStamper, ev Event) {
	stamped := stamper.Stamp(ev)
	sink <- &stamped
}

// closeSink closes the agent-level event sink. The producer owns and
// closes the channel (AG-01's chosen carrier, D2a).
func closeSink(sink chan<- *Event) {
	close(sink)
}

// drainProvider empties the provider's channel (S-LSK-002: no
// stranded producer). Walking-skeleton scope: read until close, drop
// every event on the floor. The empty-body for-range is intentional —
// drain means "consume so the producer's close has a reader"; the
// discarded events carry nothing the loop needs at this scope.
func drainProvider(ch <-chan ai.Event) error {
	for range ch {
		// Walking-skeleton scope: drop on the floor (see comment).
		_ = struct{}{}
	}
	return nil
}

// buildLoopRequest assembles the ai.Request from the system string,
// transcript and TurnOptions (D5a, R-LSK-001).
func buildLoopRequest(opts TurnOptions, system string, transcript []ai.Message) (ai.Request, error) {
	systemInst, err := ai.NewSystemText(system)
	if err != nil {
		return ai.Request{}, err
	}
	reqOpts := []ai.RequestOption{ai.WithSystemInstruction(systemInst)}
	if opts.Model != "" {
		reqOpts = append(reqOpts, ai.WithModel(opts.Model))
	}
	if opts.MaxTokens > 0 {
		reqOpts = append(reqOpts, ai.WithMaxOutputTokens(opts.MaxTokens))
	}
	return ai.NewRequest(modelForOpts(opts), transcript, reqOpts...)
}

// modelForOpts returns the request's model identity. Walking-skeleton
// scope: caller's TurnOptions.Model wins; otherwise a neutral default.
func modelForOpts(opts TurnOptions) string {
	if opts.Model != "" {
		return opts.Model
	}
	return "cachicamas-neutral-model-1"
}

// turnAccumulator is the per-turn walker that holds the in-flight
// bracket identities, the running text and reasoning fragment lists,
// the reasoning round-trip token (R-ARE-009), and the LaneStamper
// used to stamp every emitted event. Walking-skeleton scope: a single
// text or reasoning bracket at a time — AG-23 introduces multiple
// brackets.
//
// It is NOT shared across calls: every Turn(...) call mints a fresh
// turnAccumulator (R-LSK-002).
type turnAccumulator struct {
	runID   RunID
	turnID  TurnID
	stamper *LaneStamper
	sink    chan<- *Event
	textBracket struct {
		msgID     ai.MessageID
		started   bool
		ended     bool
		idx       uint32
		fragments []string
	}
	reasoningBracket struct {
		msgID     ai.MessageID
		started   bool
		ended     bool
		idx       uint32
		fragments []byte
		token     []byte
		hasToken  bool
	}
	finish   ai.FinishReason
	finishOk bool
	fatal    error
}

// newTurnAccumulator constructs a fresh per-turn walker.
func newTurnAccumulator(runID RunID, turnID TurnID, stamper *LaneStamper, sink chan<- *Event) *turnAccumulator {
	return &turnAccumulator{
		runID:   runID,
		turnID:  turnID,
		stamper: stamper,
		sink:    sink,
	}
}

// translate converts one provider event into an agent bracket event
// (or drops it on the floor per walking-skeleton scope). It also
// accumulates state: text fragments are appended, completion's
// finish reason is captured, terminal errors are recorded.
//
// Returns true when a Completion event arrives — the caller uses
// this signal to exit the drain loop and reconstruct msg.
func (t *turnAccumulator) translate(ev ai.Event) bool {
	switch ev.Kind() {
	case ai.EventKindResponseStart:
		// Layer 2 has no ResponseStart bracket (D4a): drop on the floor.
		return false

	case ai.EventKindTextBlockStart:
		msgID := mintLoopMessageID()
		t.textBracket.msgID = msgID
		t.textBracket.started = true
		t.textBracket.ended = false
		t.textBracket.idx = 0
		t.textBracket.fragments = nil
		out, err := NewMessageStartText(t.runID, t.turnID, msgID)
		if err != nil {
			t.fatal = err
			return false
		}
		emitStamped(t.sink, t.stamper, out)
		return false

	case ai.EventKindTextDelta:
		delta, _ := ev.TextDelta()
		idx := t.textBracket.idx
		t.textBracket.idx++
		t.textBracket.fragments = append(t.textBracket.fragments, delta.Delta())
		out, err := NewMessageDeltaText(t.runID, t.turnID, t.textBracket.msgID, idx, delta.Delta())
		if err != nil {
			t.fatal = err
			return false
		}
		emitStamped(t.sink, t.stamper, out)
		return false

	case ai.EventKindTextBlockEnd:
		t.textBracket.ended = true
		out, err := NewMessageEndText(t.runID, t.turnID, t.textBracket.msgID)
		if err != nil {
			t.fatal = err
			return false
		}
		emitStamped(t.sink, t.stamper, out)
		return false

	case ai.EventKindReasoningBlockStart:
		msgID := mintLoopMessageID()
		t.reasoningBracket.msgID = msgID
		t.reasoningBracket.started = true
		t.reasoningBracket.ended = false
		t.reasoningBracket.idx = 0
		t.reasoningBracket.fragments = nil
		t.reasoningBracket.token = nil
		t.reasoningBracket.hasToken = false
		out, err := NewMessageStartReasoning(t.runID, t.turnID, msgID)
		if err != nil {
			t.fatal = err
			return false
		}
		emitStamped(t.sink, t.stamper, out)
		return false

	case ai.EventKindReasoningDelta:
		delta, _ := ev.ReasoningDelta()
		idx := t.reasoningBracket.idx
		t.reasoningBracket.idx++
		t.reasoningBracket.fragments = append(t.reasoningBracket.fragments, delta.Fragment()...)
		out, err := NewMessageDeltaReasoning(t.runID, t.turnID, t.reasoningBracket.msgID, idx, string(delta.Fragment()))
		if err != nil {
			t.fatal = err
			return false
		}
		emitStamped(t.sink, t.stamper, out)
		return false

	case ai.EventKindReasoningBlockEnd:
		t.reasoningBracket.ended = true
		if blockEnd, ok := ev.ReasoningBlockEnd(); ok {
			token, hasToken := blockEnd.Token()
			if hasToken {
				t.reasoningBracket.token = append([]byte(nil), token...)
				t.reasoningBracket.hasToken = true
			}
		}
		out, err := NewMessageEndReasoning(t.runID, t.turnID, t.reasoningBracket.msgID)
		if err != nil {
			t.fatal = err
			return false
		}
		emitStamped(t.sink, t.stamper, out)
		return false

	case ai.EventKindCompletion:
		completion, _ := ev.Completion()
		t.finish = completion.FinishReason()
		t.finishOk = true
		return true

	case ai.EventKindError:
		if payload, ok := ev.ErrorPayload(); ok && payload != nil {
			t.fatal = payload
		} else {
			t.fatal = errUnknownProviderError
		}
		return false

	default:
		// Tool-call and reasoning events land here at Phase 2
		// (S-LSK-001 text-only scope); Phase 3 widens the switch
		// (S-LSK-005).
		return false
	}
}

// finalize closes the turn and run brackets, then reconstructs the
// assistant message from the accumulated fragments. Walking-skeleton
// scope: text-only reconstruction. Phase 3 widens to reasoning parts
// (S-LSK-005).
func (t *turnAccumulator) finalize() (ai.Message, ai.FinishReason) {
	// Emit turn_end (TurnOutcomeFinished) and run_end (RunOutcomeCompleted).
	// The walking skeleton's happy path is the only outcome handled
	// at this layer — failures beyond typed pass-through are
	// AG-08…AG-18.
	turnEnd, terr := NewTurnEnd(t.runID, t.turnID, TurnOutcomeFinished, nil)
	if terr == nil {
		emitStamped(t.sink, t.stamper, turnEnd)
	}
	runEnd, rerr := NewRunEnd(t.runID, RunOutcomeCompleted, nil)
	if rerr == nil {
		emitStamped(t.sink, t.stamper, runEnd)
	}

	// Reconstruct the assistant message from accumulated fragments.
	// Walking-skeleton scope: one text Part per complete text bracket,
	// one reasoning Part per complete reasoning bracket (carrying the
	// round-trip token byte-exact per R-ARE-010). Order: reasoning
	// before text (the order the walking skeleton handles them at).
	var parts []ai.Part
	if t.reasoningBracket.started && t.reasoningBracket.ended {
		text := string(t.reasoningBracket.fragments)
		var tokenArg []byte
		if t.reasoningBracket.hasToken {
			tokenArg = t.reasoningBracket.token
		}
		part, perr := ai.NewReasoning(text, tokenArg)
		if perr == nil {
			parts = append(parts, part)
		}
	}
	if t.textBracket.started && len(t.textBracket.fragments) > 0 {
		joined := ""
		for _, frag := range t.textBracket.fragments {
			joined += frag
		}
		part, perr := ai.NewText(joined)
		if perr == nil {
			parts = append(parts, part)
		}
	}
	if len(parts) == 0 {
		return ai.Message{}, t.finish
	}
	msg, mErr := ai.NewMessage(ai.RoleAssistant, parts...)
	if mErr != nil {
		return ai.Message{}, t.finish
	}
	return msg, t.finish
}

// errUnknownProviderError is the sentinel a mid-stream terminal error
// event without a payload carries — an unconstructed terminal is the
// provider's defect, not the loop's.
var errUnknownProviderError = errorString("agent: provider terminal error event with no payload")

// errorString is a tiny string-backed error for sentinels the walking
// skeleton does not want to allocate.
type errorString string

func (e errorString) Error() string { return string(e) }
