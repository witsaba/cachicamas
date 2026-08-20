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
	"errors"
	"strconv"
	"strings"
	"sync/atomic"

	"github.com/cachicamas/backend/agent/src/ai"
)

// TurnOptions is the options struct for a single assistant turn.
// Walking-skeleton scope: trivial/zero fields; AG-08 introduces hooks,
// AG-09 introduces tool opts, AG-11 introduces retry opts.
//
// AG-08 extends the surface with a pre-request hook seam
// (R-PRH-001..007): the hook is invoked once per Turn between
// buildLoopRequest (loop.go:132) and provider.Stream (loop.go:140).
// Nil is the identity default; a non-nil hook may derive a new
// ai.Request via req.With(...) and must NOT mutate the loop's input
// in place (R-REX-001).
//
// AG-09 extends the surface with a `Tools` registry (R-TLS-001..011).
// On `Completion{FinishReason: FinishReasonToolCalls}` the loop
// consumes the AI-18 tool-call events accumulated during the
// stream and dispatches them to the scheduler. Nil is the
// identity default: a zero-value `Tools` produces the same
// tool-call failure results as a registry whose Resolve always
// returns false (R-TLS-009 "one bad tool does not abort the turn").
type TurnOptions struct {
	// Model is the model identifier passed to the provider. Empty = provider default.
	// (AG-07 — unchanged)
	Model string
	// MaxTokens is the optional max-tokens budget. Zero = provider default.
	// (AG-07 — unchanged)
	MaxTokens int

	// PreRequestHook is invoked once per Turn, after buildLoopRequest
	// (loop.go:132) and before provider.Stream (loop.go:140). It
	// receives the fully-assembled ai.Request and the loop's own ctx.
	// It returns a derived ai.Request (typically via req.With(...)) or
	// an error.
	//
	// Nil is the identity default: Turn skips the seam and proceeds
	// unchanged, preserving AG-07's R-LSK-002 byte-stability on
	// identical inputs (R-PRH-005, S-PRH-002).
	//
	// A non-nil hook that returns an error aborts the turn before I/O:
	// sink is closed, and Turn returns (ai.Message{}, 0, *ai.PreStreamFailure)
	// reusing the existing pre-stream-failure path (R-PRH-003, S-PRH-003).
	//
	// Hooks must NOT mutate the loop's input in place; ai.Request is
	// a value type whose With(...) rebuilds from r.options
	// (R-REX-001, request.go:325-336). For identical inputs across
	// calls, the hook must produce byte-equal outputs (R-PRH-007,
	// S-PRH-006).
	PreRequestHook func(ctx context.Context, req ai.Request) (ai.Request, error)

	// Tools is the registry keyed by tool name (D9a, R-TLS-001).
	// The loop resolves each model request via registry.Resolve(name);
	// an unresolved name yields a typed
	// Result{Outcome: ExecutionFailure, Failure: ...} in that call's
	// ordinal slot — consistent with R-TLS-009 "one bad tool does
	// not abort the turn".
	//
	// Nil is the identity default: a zero-value TurnOptions
	// produces the same tool-call failure results as a TurnOptions
	// with a registry whose Resolve always returns false. Non-breaking
	// zero-value extension; AG-07/AG-08 tests that don't set Tools
	// continue to pass byte-stable.
	//
	// The loop calls `Schedule` exactly once per Turn between
	// `provider.Stream` close and `finalize` when the provider's
	// completion carries `FinishReasonToolCalls` (R-TLS-008 wording
	// trap; AG-13 owns iteration).
	Tools Registry

	// PermissionPolicy is the Layer-2 permission gate (AG-10,
	// R-APP-001..011, D-C). The loop forwards it byte-exact to
	// `Schedule`.
	//
	// Nil is the identity default: `Schedule` bypasses the gate —
	// every call is treated as AllowOnce, no permission events are
	// emitted, and the call runs exactly as AG-09's pre-AG-10
	// scheduler did. This is a legitimate, permanent bypass (not a
	// placeholder): AG-07/AG-08/AG-09's existing nil-policy tests
	// stay byte-stable (R-LSK-002 carry).
	//
	// A non-nil policy is consulted for every scheduled call.
	// Layer 3 supplies the implementation (doc 0004 CO-03,
	// R-APP-011); Layer 2 owns only the protocol that drives it.
	// The loop's own upward-path wake (`Scheduler.WakeParked`,
	// D-A) is AG-13's wiring — out of scope here.
	PermissionPolicy PermissionPolicy

	// Continuation is AG-13's run-continuation seam (R-LSK-001,
	// design "Decision 1"). Nil is the identity default: every path
	// through Turn is byte-stable pre-AG-13 behavior — identity
	// minted fresh, run brackets emitted unconditionally, a fresh
	// per-call lane, a locally constructed scheduler, finalize-first
	// ordering, and no transcript wiring. This is the whole of
	// AG-13's compatibility claim (R-LSK-002 carry, S-LSK-015).
	//
	// Non-nil requires every member of TurnContinuation set: a
	// half-configured continuation is rejected typed, before any
	// event is emitted (validateContinuation, S-LSK-014).
	Continuation *TurnContinuation

	// Hooks is AG-20's transport: Harness.Run fills this on its
	// per-turn copy of Turn, beside Continuation — it is NEVER a
	// second registration surface (R-HKS-001, hooks.go). Turn itself
	// consumes only Hooks.PreRequest (through applyPreRequestHook,
	// composed after the singular field above); the observing
	// families and the reporter are INERT on a directly-constructed
	// TurnOptions — Turn owns no observer lane and no run identity
	// (R-HKS-001, S-HKS-003).
	//
	// Zero-value Hooks is the identity default: byte-identical to
	// today's path on every arm (R-HKS-012).
	Hooks Hooks
}

// TurnContinuation is AG-13's run-continuation seam: the four things a
// multi-turn run must share across turns, as one all-or-nothing group
// (design "Decision 1"). Nil TurnOptions.Continuation is pre-AG-13
// behavior, byte-stable on every path.
type TurnContinuation struct {
	// Run is the caller-minted run identity; Turn joins it instead of
	// minting one (loop.go:132's forecast — AG-13's Harness mints
	// identities in caller-supplied groups).
	Run RunID

	// Stamper is the caller-owned lane stamper; Turn uses it instead
	// of creating a fresh one, so N invocations share one contiguous
	// 1-based lane.
	Stamper *LaneStamper

	// Scheduler is the caller-owned scheduler; injecting it (rather
	// than handing one back) is what makes WakeParked reachable while
	// Turn is blocked (design "Decision 2"). The caller is
	// responsible for the scheduler's LeaveSinkOpen field: Turn emits
	// the turn-close event after Schedule returns on this path
	// (R-LSK-001 point 3), which would be a send on a closed channel
	// unless the caller set it.
	Scheduler *Scheduler

	// History is the caller-owned transcript store; Turn appends this
	// turn's own messages to it — the assistant message and its tool
	// results (R-HIS-010).
	History *History
}

// validateContinuation reports whether cont is well-formed: nil is
// always valid (the identity default); non-nil requires every member
// set, checked in field-declaration order so the first absent member
// is the one reported (V-FAIL-04 carry). A half-configured
// continuation must be caught before Turn emits anything at all
// (S-LSK-014) — there is no "run brackets replayed" fallback path.
func validateContinuation(cont *TurnContinuation) error {
	if cont == nil {
		return nil
	}
	if cont.Run == "" {
		return ai.Invalid(ai.ErrEmpty, ai.At("continuation"), ai.At("run"))
	}
	if cont.Stamper == nil {
		return ai.Invalid(ai.ErrEmpty, ai.At("continuation"), ai.At("stamper"))
	}
	if cont.Scheduler == nil {
		return ai.Invalid(ai.ErrEmpty, ai.At("continuation"), ai.At("scheduler"))
	}
	if cont.History == nil {
		return ai.Invalid(ai.ErrEmpty, ai.At("continuation"), ai.At("history"))
	}
	return nil
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
	// AG-13 (R-LSK-001, S-LSK-014): a half-configured continuation is
	// rejected before ANY emission and before the sink is touched at
	// all — no closeSink, no partial stream. This check MUST run
	// before identity minting or the run bracket, so it is the first
	// statement in the function body.
	if err := validateContinuation(opts.Continuation); err != nil {
		return ai.Message{}, 0, err
	}

	var runID RunID
	var stamper *LaneStamper
	if opts.Continuation != nil {
		// AG-13 continuation path (R-LSK-001 point 1): join the
		// caller-supplied run identity and lane stamper instead of
		// minting/constructing fresh ones, so N invocations share one
		// contiguous 1-based lane.
		runID = opts.Continuation.Run
		stamper = opts.Continuation.Stamper
	} else {
		runID = mintLoopRunID()
		stamper = &LaneStamper{}
	}
	// TurnID is still minted fresh per call on every path — N distinct
	// turn brackets need N identities (R-LSK-001 point 1 carry).
	turnID := mintLoopTurnID()

	// Emit turn_start unconditionally; run_start ONLY on the nil path
	// (R-AEV-002: stamped, contiguous, 1-based; R-LSK-002: per-call
	// stamper on the nil path, no shared state). AG-13 continuation
	// path: the caller's run owns the run bracket (R-LSK-001 point 2)
	// — Turn emits no run-open and no run-close on any path.
	if opts.Continuation == nil {
		runStart, err := NewRunStart(runID)
		if err != nil {
			closeSink(sink)
			return ai.Message{}, 0, err
		}
		emitStamped(sink, stamper, runStart)
	}

	turnStart, err := NewTurnStart(runID, turnID)
	if err != nil {
		closeSink(sink)
		return ai.Message{}, 0, err
	}
	emitStamped(sink, stamper, turnStart)

	// Build the request from system + transcript + opts (D5a, R-LSK-001).
	req, err := buildLoopRequest(opts, system, transcript)
	if err != nil {
		// AG-15 D1 (R-LSK-001 amendment, S-LSK-021): once turn_start
		// has been emitted, every failure exit MUST close the turn
		// bracket it opened before closing the sink — mirroring the
		// mid-stream block below (loop.go:398-406). Best-effort: if
		// the wrap or a constructor fails, the emission is skipped
		// and this path's original error is still returned unchanged
		// (the ferr == nil / terr == nil guard posture already at
		// loop.go:388-404).
		emitPreStreamAbort(sink, stamper, runID, turnID, opts.Continuation == nil, err)
		closeSink(sink)
		return ai.Message{}, 0, err
	}

	// AG-08.1 — pre-request hook seam (R-PRH-001..007). Nil is the
	// identity default (R-PRH-005, D4a); a non-nil hook returns a
	// derived ai.Request via req.With(...) (R-REX-001). On hook error
	// the turn aborts BEFORE provider.Stream, mirroring the
	// pre-stream-failure path (R-PRH-003, D2): close sink, return
	// (ai.Message{}, 0, *ai.PreStreamFailure) carrying a
	// hook-attributing FailureReport.Category (UnsupportedCapability).
	req, err = applyPreRequestHook(ctx, req, opts.PreRequestHook, opts.Hooks.PreRequest)
	if err != nil {
		typedErr, perr := ai.PreStreamFailure(ai.FailureReport{
			Category:    ai.FailureCategoryUnsupportedCapability,
			StatusClass: 4,
			// AG-20 (R-HKS-009): Cause carries the source-attributed
			// error applyPreRequestHook wrapped — err's own
			// errors.Unwrap chain, not a fresh construction. The
			// caller reaches the source name through
			// errors.Unwrap(returnedErr).Error() (S-HKS-022/023,
			// S-PRH-009).
			Cause: err,
		})
		// AG-15 D1: the same bracket-closing obligation as the build
		// path above. typedErr is already the *ai.Failure this path
		// returns, so preStreamAbortFailure's errors.As arm wraps it
		// AS ITSELF — one construction feeds both the returned error
		// and the emitted failure. Skipped when perr != nil: no typed
		// failure exists to emit in that (practically unreachable)
		// case.
		if perr == nil {
			emitPreStreamAbort(sink, stamper, runID, turnID, opts.Continuation == nil, typedErr)
		}
		closeSink(sink)
		if perr != nil {
			return ai.Message{}, 0, perr
		}
		return ai.Message{}, 0, typedErr
	}

	// Drain the provider's channel, translating each event into an
	// agent-level bracket (S-LSK-001, S-LSK-002).
	pCh, streamErr := provider.Stream(ctx, req)
	if streamErr != nil {
		// Pre-stream failure (V-PRV-04): no producer goroutine, no
		// channel. The walking skeleton closes sink and surfaces the
		// typed error to the caller. AG-15 D1: the same
		// bracket-closing obligation as the other two pre-stream
		// paths, before the close.
		emitPreStreamAbort(sink, stamper, runID, turnID, opts.Continuation == nil, streamErr)
		closeSink(sink)
		return ai.Message{}, 0, streamErr
	}

	turn := newTurnAccumulator(runID, turnID, stamper, sink, opts.Continuation != nil)
	for ev := range pCh {
		if done := turn.translate(ev); done {
			// AG-13 (R-LSK-001 point 3): the continuation path
			// reorders to schedule-before-finalize, so tool and
			// permission events land inside the still-open turn
			// bracket. The nil path below keeps the original
			// finalize-first order byte-stable (S-LSK-015).
			if opts.Continuation != nil {
				return finishContinuationTurn(ctx, turn, opts, runID, turnID, stamper, sink)
			}
			// Completion: capture the finish reason and exit the loop.
			// If the completion carries tool calls, dispatch them
			// through the scheduler (AG-09 wire-up, R-TLS-001..011).
			// The scheduler emits tool events on `sink` and the
			// loop's `stamper` owns the single-writer invariant.
			// The scheduler closes the sink after the rejoin, so
			// the loop's finalize emits its closing brackets
			// BEFORE the scheduler runs. The wire-up is therefore:
			// finalize-first, schedule-second, close-third.
			msg, finish := turn.finalize()
			if turn.finish == ai.FinishReasonToolCalls && len(turn.toolCalls) > 0 {
				sched := &Scheduler{MaxConcurrentReads: maxReadFanOutDefault}
				// AG-10 (D-C): forward the caller's policy
				// byte-exact. Nil remains a legitimate bypass
				// (TurnOptions.PermissionPolicy doc comment);
				// the loop's own upward-path wake wiring
				// (Scheduler.WakeParked) is AG-13's scope.
				_ = sched.Schedule(ctx, turn.toolCalls, opts.Tools, runID, turnID, opts.PermissionPolicy, stamper, sink)
			}
			closeSink(sink)
			return msg, finish, nil
		}
		if turn.fatal != nil {
			// Mid-stream terminal error: drain whatever is left so
			// the producer's goroutine is not stranded (S-LSK-002).
			_ = drainProvider(pCh)

			// D1's type fork: a typed *ai.Failure — the
			// ev.ErrorPayload() arm of translate()'s
			// EventKindError case — gets the typed-Aborted
			// treatment (R-ATT-005..008). An internal
			// construction error (a plain Go error from one of
			// the loop's own event constructors) stays
			// byte-unchanged from pre-AG-11: no emission, drain,
			// close, return.
			if pf, ok := turn.fatal.(*ai.Failure); ok {
				if failure, ferr := NewFailure(pf); ferr == nil {
					// On the nil path, one NewFailure call feeds
					// both emissions (task 5.3's identity pin):
					// turnEnd.Failure() == runEnd.Failure() by
					// pointer, proving single construction, not
					// two independent wraps. AG-13 continuation
					// path: turn_end(Aborted) is still emitted,
					// but no run_end — the caller's run owns
					// that bracket on every path, including this
					// one (R-LSK-001 point 2).
					if turnEnd, terr := NewTurnEnd(runID, turnID, TurnOutcomeAborted, failure); terr == nil {
						emitStamped(sink, stamper, turnEnd)
					}
					if opts.Continuation == nil {
						if runEnd, rerr := NewRunEnd(runID, RunOutcomeFailed, failure); rerr == nil {
							emitStamped(sink, stamper, runEnd)
						}
					}
				}
				// D7: no second provider.Stream call — the
				// turn winds down here, after exactly one.
				msg := turn.reconstructMessage()
				closeSink(sink)
				return msg, 0, turn.fatal
			}
			closeSink(sink)
			return ai.Message{}, 0, turn.fatal
		}
	}

	// AG-14 (R-CAN-001): Turn's first-ever cancellation observation,
	// checked BEFORE the "ran to empty" normalization below so a
	// cancelled run is never misread as a quietly completed one. A
	// provider stream that closes bare because ctx was cancelled
	// through a harness signal (R-CNF-011/R-CNF-012: the fake's
	// producer selects on ctx.Done() and returns with no terminal
	// event) is architecturally identical, at this point, to one that
	// simply ran out of events — context.Cause is what tells the two
	// apart. A bare cancellation of the caller's own context, not
	// routed through a harness signal, does not match either sentinel
	// here and falls through to the existing normalization unchanged
	// (the scope line, R-CAN-001).
	//
	// Turn still owns its own turn bracket on this path, exactly as
	// the existing mid-stream-fatal branch below does for a genuine
	// provider error: CheckStream rejects a run-close while a turn is
	// still open (BracketRoleClosesRun's own rule), so turn_end MUST
	// close it before Run's wind-down closes the run bracket. The
	// outcome is TurnOutcomeAborted — the closest existing member, no
	// new one introduced — carrying a typed *Failure built with
	// ai.FailureCategoryCancellation (a pre-existing Layer 1
	// vocabulary member, not new here). This is the TURN's own
	// bracket only; R-CAN-002's "no *Failure" rule governs the RUN's
	// run_end, which windDownRun builds separately with none.
	if cause := context.Cause(ctx); errors.Is(cause, ErrInterrupted) || errors.Is(cause, ErrShutdown) {
		if failure, ferr := cancellationTurnFailure(cause); ferr == nil {
			if turnEnd, terr := NewTurnEnd(runID, turnID, TurnOutcomeAborted, failure); terr == nil {
				emitStamped(sink, stamper, turnEnd)
			}
		}
		closeSink(sink)
		return ai.Message{}, 0, cause
	}

	// Provider closed without a Completion: walking-skeleton scope
	// treats this as a "ran to empty" turn — normalize the finish
	// reason to FinishReasonStop, the documented fallback for a
	// stream that finished without a Completion event (per AI-13.1's
	// unknown-stop semantics: the absence is recorded, not
	// corrected), BEFORE finalize() dispatches on it (D2). Without
	// this the dispatch would see the zero FinishReason, which is not
	// a vocabulary member.
	if turn.finish == 0 {
		turn.finish = ai.FinishReasonStop
	}
	msg, finish := turn.finalize()
	closeSink(sink)
	return msg, finish, nil
}

// finishContinuationTurn runs the continuation path's completion,
// once the provider stream ends in a Completion (R-LSK-001 points
// 2-5, R-HIS-010). Unlike the nil path, this schedules BEFORE
// finalize: the turn's tool and permission events must land inside
// the still-open turn bracket, because CheckStream rejects a
// PlacementTurn event outside an open turn or after the terminal
// run_end (which finalize-first would produce, since the nil path's
// finalize also fires the loop's own turn-close before Schedule
// runs). It captures the rejoin (the `_ =` discard ends here, on this
// path only), lets reconstructMessage additionally carry the turn's
// own ai.ToolCall parts, and then commits the turn's own messages —
// the assistant message and one tool-result message per rejoin
// result, in call order — to the continuation's History. Turn does
// NOT call CloseTurn: that stays the run driver's, at the turn
// boundary (R-RUN-005).
func finishContinuationTurn(
	ctx context.Context,
	t *turnAccumulator,
	opts TurnOptions,
	runID RunID,
	turnID TurnID,
	stamper *LaneStamper,
	sink chan<- *Event,
) (ai.Message, ai.FinishReason, error) {
	var results []Result
	if t.finish == ai.FinishReasonToolCalls && len(t.toolCalls) > 0 {
		// AG-10 (D-C carry): forward the caller's policy byte-exact;
		// nil remains a legitimate bypass. The caller's injected
		// scheduler is used (not a locally constructed one), so its
		// LeaveSinkOpen setting governs whether this call closes
		// sink — it MUST NOT, since finalize below still needs to
		// emit turn_end on it.
		results = opts.Continuation.Scheduler.Schedule(ctx, t.toolCalls, opts.Tools, runID, turnID, opts.PermissionPolicy, stamper, sink)
	}

	msg, finish := t.finalize()

	if !msg.ID().IsZero() {
		if err := opts.Continuation.History.Append(msg); err != nil {
			closeSink(sink)
			return msg, finish, err
		}
	}
	for _, r := range results {
		resultMsg, err := toolResultMessage(r)
		if err == nil {
			err = opts.Continuation.History.Append(resultMsg)
		}
		if err != nil {
			closeSink(sink)
			return msg, finish, err
		}
	}

	closeSink(sink)
	return msg, finish, nil
}

// toolResultMessage builds the ai.RoleTool message the continuation
// path appends for one rejoin result (R-HIS-010 point 2): the success
// form for ToolOutcomeSuccess, the failure form for both failure
// outcomes — verified ai/tool_result.go:77,105. The outcome rides the
// Layer 1 failure form itself (Part.ToolResult().Failed()), never a
// content sentinel. ToolOutcomeResultFailure carries the tool's own
// failure output in Content (tool.go's Result doc);
// ToolOutcomeExecutionFailure carries none, so its typed *Failure's
// redacted diagnostic renders the content instead — still content the
// model can reason about.
func toolResultMessage(r Result) (ai.Message, error) {
	var part ai.Part
	var err error
	switch r.Outcome {
	case ToolOutcomeSuccess:
		part, err = ai.NewToolResult(r.CallID(), string(r.Content))
	case ToolOutcomeResultFailure:
		part, err = ai.NewToolFailure(r.CallID(), string(r.Content))
	case ToolOutcomeExecutionFailure:
		content := ""
		if f := r.FailureFor(); f != nil {
			content = f.Unwrap().Error()
		}
		part, err = ai.NewToolFailure(r.CallID(), content)
	default:
		part, err = ai.NewToolFailure(r.CallID(), "")
	}
	if err != nil {
		return ai.Message{}, err
	}
	return ai.NewMessage(ai.RoleTool, part)
}

// cancellationTurnFailure wraps cause as the typed *Failure the
// cause-check branch's turn_end(Aborted) carries (AG-14, R-CAN-001).
// ai.FailureCategoryCancellation is a pre-existing Layer 1 vocabulary
// member (provider_failure.go), not introduced by this change; the
// construction mirrors harness.go's own wrapHarnessFailure, through
// the same public constructors, with the cancellation category in
// place of Unavailable.
func cancellationTurnFailure(cause error) (*Failure, error) {
	aiFailure, ferr := ai.MidStreamFailure(ai.FailureReport{
		Category: ai.FailureCategoryCancellation,
		Cause:    cause,
	}, false)
	if ferr != nil {
		return nil, ferr
	}
	return NewFailure(aiFailure)
}

// preStreamAbortFailure builds the typed *Failure a pre-stream
// abort's turn_end carries (R-LSK-001, AG-15 D1 — agent-loop-skeleton
// delta AD-1 sub-question 1). An err that already IS an *ai.Failure —
// the hook path's own typed pre-stream failure (loop.go:319-322) and
// whatever provider.Stream returned (loop.go:332-338) — is wrapped as
// itself via errors.As, preserving its category, retryability,
// retry-after, partial-output and delivery readings unchanged (the
// retry.go:119-125 precedent: errors.As over a bare type assertion).
// A plain Go error — the request-build path's own error
// (loop.go:304-308) — is wrapped as a fresh pre-stream failure, since
// a pre-stream abort truthfully crossed no carrier
// (ai.DeliveryPreStream, provider_failure.go:610-612). One helper
// serves all three call sites.
func preStreamAbortFailure(err error) (*Failure, error) {
	var aiFailure *ai.Failure
	if errors.As(err, &aiFailure) {
		return NewFailure(aiFailure)
	}
	built, ferr := ai.PreStreamFailure(ai.FailureReport{
		Category: ai.FailureCategoryUnavailable,
		Cause:    err,
	})
	if ferr != nil {
		return nil, ferr
	}
	return NewFailure(built)
}

// emitPreStreamAbort emits the turn_end(Aborted) — and, on the
// nil-continuation path (ownsRunBracket), the run_end(Failed) — that
// AG-15 D1 adds ahead of each of Turn's three pre-stream failure
// paths, before the caller closes sink (R-LSK-001 amendment,
// S-LSK-021). Best-effort, matching the mid-stream block's own guard
// posture (loop.go:387-406): if preStreamAbortFailure or a
// constructor fails, the emission is skipped silently and the
// caller's own original error is still returned unchanged.
func emitPreStreamAbort(sink chan<- *Event, stamper *LaneStamper, runID RunID, turnID TurnID, ownsRunBracket bool, err error) {
	failure, ferr := preStreamAbortFailure(err)
	if ferr != nil {
		return
	}
	if turnEnd, terr := NewTurnEnd(runID, turnID, TurnOutcomeAborted, failure); terr == nil {
		emitStamped(sink, stamper, turnEnd)
	}
	if ownsRunBracket {
		if runEnd, rerr := NewRunEnd(runID, RunOutcomeFailed, failure); rerr == nil {
			emitStamped(sink, stamper, runEnd)
		}
	}
}

// outcomeForFinish maps a normalized ai.FinishReason to its TurnOutcome
// (R-ATT-001/002, D4). Total over the finish-reason vocabulary — the
// default case is defensive and unreachable in production: both call
// sites (the completion path, validated non-zero by ai.NewCompletion,
// and the no-completion path, normalized by Turn ahead of the call per
// D2) hand this function a member value.
func outcomeForFinish(finish ai.FinishReason) TurnOutcome {
	switch finish {
	case ai.FinishReasonStop:
		return TurnOutcomeFinished
	case ai.FinishReasonLength:
		return TurnOutcomeLengthLimited
	case ai.FinishReasonToolCalls:
		return TurnOutcomeToolCalls
	case ai.FinishReasonContentFilter:
		return TurnOutcomeContentFiltered
	case ai.FinishReasonRefusal:
		return TurnOutcomeRefused
	case ai.FinishReasonPauseTurn:
		return TurnOutcomePaused
	case ai.FinishReasonUnknown:
		return TurnOutcomeUnknown
	default:
		return 0 // no outcome; NewTurnEnd rejects it.
	}
}

// emitStamped stamps ev through the LaneStamper and sends it on sink.
// The LaneStamper is the per-lane counter (R-AEV-002): every Turn
// call mints a fresh one (R-LSK-002).
func emitStamped(sink chan<- *Event, stamper *LaneStamper, ev Event) {
	stamped := stamper.Stamp(ev)
	sink <- &stamped
}

// closeSink closes the agent-level event sink. The producer owns
// and closes the channel (AG-01's chosen carrier, D2a). AG-09:
// the scheduler may have already closed the channel if the
// wire-up invoked `Schedule` between `provider.Stream` close
// and `finalize`. The double-close is tolerated via a
// `recover` — the contract that matters is "the channel is
// closed by the time `Turn` returns"; the close path is a
// "best effort" once the scheduler has taken over.
func closeSink(sink chan<- *Event) {
	defer func() { _ = recover() }()
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

// preRequestSingularSourceName and preRequestChainSourceName are
// R-HKS-009's two source-name vocabularies: the shipped singular
// field's own fixed name, and a chain element's own registration
// index — never a bare ordinal over the composed sequence, which a
// later insertion would renumber (a renumbered attribution is a lie a
// green suite never catches).
const preRequestSingularSourceName = "TurnOptions.PreRequestHook"

// preRequestChainSourceName renders a chain element's own registration-
// indexed name. strconv.Itoa, not "fmt" — cancellation.go's own
// package comment records why: "fmt" transitively imports "os" and
// "io/fs", which the production-closure import-boundary guard
// (import_boundary_test.go) forbids by exact path.
func preRequestChainSourceName(i int) string {
	return "Hooks.PreRequest[" + strconv.Itoa(i) + "]"
}

// attributedPreRequestError wraps a hook's own error with its SOURCE's
// own name (R-HKS-009), reachable through the standard single-error
// errors.Unwrap route — the "fmt.Errorf(%w, ...)" shape built by hand,
// mirroring cancellation.go's promptAfterShutdownError precedent, so
// this file adds no "fmt" import.
type attributedPreRequestError struct {
	source string
	err    error
}

func (e *attributedPreRequestError) Error() string { return e.source + ": " + e.err.Error() }

func (e *attributedPreRequestError) Unwrap() error { return e.err }

// applyPreRequestHook composes the pre-request chain (R-HKS-002, AG-20
// widening of AG-08's shipped shape): the singular field, when
// non-nil, is invoked FIRST, on the request buildLoopRequest produced;
// its returned request is the input to chain[0]; element i's returned
// request is the input to element i+1; and the chain's FINAL element's
// returned request — and only that value — is what this returns. Nil
// singular AND an empty chain is the identity default (R-PRH-005 as
// amended): req returns unchanged, preserving AG-07's R-LSK-002
// byte-stability.
//
// A non-nil error from any source aborts the sequence — no later
// element is invoked — and is wrapped with its SOURCE's own name
// (R-HKS-009) via %w, reachable by the caller's errors.Unwrap chain:
// TurnOptions.PreRequestHook for the singular field, or
// Hooks.PreRequest[i] for a chain element at index i.
func applyPreRequestHook(
	ctx context.Context,
	req ai.Request,
	hook func(ctx context.Context, req ai.Request) (ai.Request, error),
	chain []PreRequestHook,
) (ai.Request, error) {
	next := req
	if hook != nil {
		out, err := hook(ctx, next)
		if err != nil {
			return ai.Request{}, &attributedPreRequestError{source: preRequestSingularSourceName, err: err}
		}
		next = out
	}
	for i, elem := range chain {
		out, err := elem(ctx, next)
		if err != nil {
			return ai.Request{}, &attributedPreRequestError{source: preRequestChainSourceName(i), err: err}
		}
		next = out
	}
	return next, nil
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
	// continuation reports whether this turn runs on AG-13's
	// continuation path (opts.Continuation != nil, R-LSK-001 point 2):
	// finalize skips the run-close bracket (the caller's run owns
	// it), and reconstructMessage additionally carries the turn's own
	// ai.ToolCall parts (R-LSK-001 point 5, R-HIS-010 point 1). Never
	// set on the nil path, so nil-path behavior stays byte-stable.
	continuation bool
	textBracket  struct {
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
	// toolCalls accumulates the AI-18 tool-call events the
	// provider streams. On Completion{FinishReasonToolCalls}
	// the loop forwards the slice to the scheduler (AG-09).
	// Per-block fields hold the in-flight tool call's identity
	// and argument bytes; the completed ToolCall is appended
	// to `calls` on ToolCallEnd.
	toolCalls       []ai.ToolCall
	toolResults     []Result
	currentToolID   string
	currentToolName string
	currentToolArgs strings.Builder
	finish          ai.FinishReason
	finishOk        bool
	fatal           error
	// usage is the turn's ai.Usage, captured from the Completion event
	// (AG-16, R-CST-001). Its zero value is ai.Usage{} — every figure
	// absent — which is exactly the correct reading for a turn whose
	// provider closed without ever emitting a Completion (S-CST-002):
	// finalize() still runs, and the emitted cost_turn reports all
	// five figures absent rather than five invented zeros.
	usage ai.Usage
}

// newTurnAccumulator constructs a fresh per-turn walker. continuation
// reports whether this turn runs on AG-13's continuation path
// (opts.Continuation != nil) — see turnAccumulator.continuation.
func newTurnAccumulator(runID RunID, turnID TurnID, stamper *LaneStamper, sink chan<- *Event, continuation bool) *turnAccumulator {
	return &turnAccumulator{
		runID:        runID,
		turnID:       turnID,
		stamper:      stamper,
		sink:         sink,
		continuation: continuation,
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
		// AG-16 (R-CST-001, R-CST-003): capture usage beside the
		// existing finish capture. No emission here — finalize()
		// converts and emits it (loop.go:1001).
		t.usage = completion.Usage()
		return true

	case ai.EventKindToolCallStart:
		// AG-09 wire-up: accumulate AI-18 tool-call events. The
		// loop translates the three call events (start / delta /
		// end) into a `[]ai.ToolCall` it forwards to the scheduler
		// on Completion. Drops on the floor are no longer
		// correct once the model emits a FinishReasonToolCalls
		// completion (R-TLS-001, R-TLS-009).
		if start, ok := ev.ToolCallStart(); ok {
			t.currentToolID = start.ID()
			t.currentToolName = start.Name()
			t.currentToolArgs.Reset()
		}
		return false

	case ai.EventKindToolCallDelta:
		// Fragment is raw bytes (R-ATC-005). The loop stores
		// them as a string builder; NewToolCall will validate
		// JSON well-formedness at the seam 2 boundary.
		if delta, ok := ev.ToolCallDelta(); ok {
			t.currentToolArgs.Write(delta.Fragment())
		}
		return false

	case ai.EventKindToolCallEnd:
		// ToolCallEnd carries the complete argument bytes
		// (R-ATC-006, R-ATC-007). Construct a Layer 1 ToolCall
		// and append to the in-flight list. The scheduler takes
		// the slice; loop.go never decodes, re-marshals, or
		// re-orders the bytes (R-ATC-006 carry, V-REQ-17).
		if _, ok := ev.ToolCallEnd(); ok && t.currentToolID != "" {
			args := t.currentToolArgs.String()
			// Layer 1 NewToolCall validates JSON well-formedness
			// (R-ATC-007); a malformed payload is a typed failure
			// the loop records as fatal so the next emit is
			// rejected by the validator.
			part, terr := ai.NewToolCall(t.currentToolID, t.currentToolName, []byte(args))
			if terr != nil {
				t.fatal = terr
				t.currentToolID = ""
				return false
			}
			tc, _ := part.ToolCall()
			t.toolCalls = append(t.toolCalls, tc)
			t.currentToolID = ""
		}
		return false

	case ai.EventKindError:
		// Mid-stream terminal error: record the failure and let the
		// drain loop break. drainProvider empties whatever is left
		// so the producer's goroutine is not stranded (S-LSK-002).
		// The else branch is unreachable through agenttest's Emit
		// (which rejects an event that was never constructed), but
		// it stays defensive: an unconstructed terminal is the
		// provider's defect, not the loop's.
		if payload, ok := ev.ErrorPayload(); ok {
			t.fatal = payload
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
	// AG-16 (R-CST-001): finalize is reached only by a turn that
	// closes non-aborted (every aborted exit returns before this
	// point), so its first emission is the turn's own cost_turn,
	// inside the bracket it opened and before turn_end — the iff's
	// emitting half. Figures and presence come from t.usage, captured
	// beside t.finish at the Completion case; a turn that never saw
	// one (the no-Completion fallback) still reaches finalize with
	// t.usage at its zero value, so every figure reports absent
	// rather than an invented zero. Best-effort: a construction
	// failure skips the emission and changes nothing else finalize
	// does.
	if costTurn, cerr := newCostTurnFromUsage(t.runID, t.turnID, CostLabelFinal, t.usage); cerr == nil {
		emitStamped(t.sink, t.stamper, costTurn)
	}

	// Emit turn_end (dispatched from t.finish via outcomeForFinish,
	// R-ATT-002) and run_end (RunOutcomeCompleted — refusal, pause and
	// unknown are turn-level distinctions only; D5). Failures beyond
	// typed pass-through on this, the non-fatal path, are out of
	// scope: they never reach finalize (the fatal path returns
	// earlier, before the drain loop's normal exit).
	outcome := outcomeForFinish(t.finish)
	turnEnd, terr := NewTurnEnd(t.runID, t.turnID, outcome, nil)
	if terr == nil {
		emitStamped(t.sink, t.stamper, turnEnd)
	}
	// AG-13 (R-LSK-001 point 2): the continuation path emits no
	// run-close — the caller's run owns that bracket; a terminal
	// finish reason with a queued steering message must not close the
	// run. The nil path stays byte-stable (S-LSK-015).
	if !t.continuation {
		runEnd, rerr := NewRunEnd(t.runID, RunOutcomeCompleted, nil)
		if rerr == nil {
			emitStamped(t.sink, t.stamper, runEnd)
		}
	}

	return t.reconstructMessage(), t.finish
}

// reconstructMessage rebuilds the assistant message from the fragments
// accumulated so far (D6): one text Part per complete text bracket, one
// reasoning Part per complete reasoning bracket (carrying the
// round-trip token byte-exact per R-ARE-010), and one text Part per
// tool result. Order: reasoning before text before tool results — the
// order the walking skeleton handles them at.
//
// Both finalize() (the normal-completion path) and Turn's mid-stream
// fatal branch (R-ATT-007) call this — one reconstruction, two callers,
// no semantic fork. On the fatal path a bracket left open (started but
// not ended) still drops, matching finalize()'s existing rule
// unchanged.
func (t *turnAccumulator) reconstructMessage() ai.Message {
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
	// AG-13 (R-LSK-001 point 5, R-HIS-010 point 1) — continuation
	// path only: append the turn's own tool-call parts, provider-
	// exact bytes (re-derived from each ai.ToolCall's own accessors,
	// which return byte-equal bytes to what NewToolCall was given),
	// so a pure-tool turn yields a constructible assistant message
	// whose calls the transcript's pairing invariant can match. The
	// nil path never sets t.continuation, so this stays unreachable
	// there (S-LSK-015 byte-stability).
	if t.continuation {
		for _, tc := range t.toolCalls {
			part, perr := ai.NewToolCall(tc.ID(), tc.Name(), tc.Arguments())
			if perr == nil {
				parts = append(parts, part)
			}
		}
	}
	// AG-09 — append tool-result parts when the turn carries
	// tool calls (the loop accumulated them during translate;
	// the scheduler ran them between Completion and finalize).
	// The tool results are stored as text parts in the order
	// the scheduler returned them.
	if len(t.toolResults) > 0 {
		for _, r := range t.toolResults {
			// Tool outcomes are typed; we serialize the result
			// payload as a text part for the message transcript.
			// R-TLS-007's typed `Result` reaches the caller as a
			// transcript part; future AGs widen this surface.
			if r.Content == nil {
				continue
			}
			part, perr := ai.NewText(string(r.Content))
			if perr == nil {
				parts = append(parts, part)
			}
		}
	}
	if len(parts) == 0 {
		return ai.Message{}
	}
	msg, mErr := ai.NewMessage(ai.RoleAssistant, parts...)
	if mErr != nil {
		return ai.Message{}
	}
	return msg
}
