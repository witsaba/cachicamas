// AG-18 — compaction: the call, the surgery, the record
// (R-CMP-001..014).
//
// Compaction is one internal operation, runCompaction, reachable through
// exactly two doors (R-CMP-001): the context strategy's verdict,
// captured and acted on at the run driver's turn boundary
// (harness.go), and the on-demand [Harness.Compact] entry point this
// file also defines. Both run the operation inside a dedicated
// compaction turn bracket, built from the existing exported
// NewTurnStart/NewTurnEnd constructors, so the compaction family's
// PlacementTurn rule (stream_check.go:161) is satisfied with the
// validator byte-unchanged.
//
// # Cut resolution (R-CMP-004, design AD-1)
//
// resolveCut is a pure, read-only helper: it retracts a naive cut
// backward only, to the greatest recorded turn-mark boundary at or
// before it, verified pairing-closed by construction (a mark can only
// be committed when History's open-call set was empty at that instant
// — see [turnMark]'s own doc comment in history.go). It never widens
// History's exported surface to reach private pairing state, and it
// never expands the cut forward: forward expansion would compact
// entries the strategy's own boundary designated protected, which the
// charter's protection clause forbids at MUST level.

package agent

import (
	"context"
	"errors"
	"strconv"
	"strings"
	"sync/atomic"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"

	"github.com/cachicamas/backend/agent/src/ai"
)

// resolveCut resolves naiveCut against hist's recorded turn marks and
// pairing state (R-CMP-004, design AD-1): retract to the greatest
// recorded turn-mark boundary <= naiveCut, then verify the resulting
// prefix is pairing-closed by read-only correlation over the public
// read-back, retracting further on a straddling pair. Termination is
// structural: cut strictly decreases every iteration and is bounded
// below by 0.
//
// A resolved cut of 0 means there is nothing to compact — no recorded
// mark exists at or before naiveCut, or the transcript is empty — and
// the caller MUST report that as a typed failure rather than commit an
// empty replacement (R-CMP-004's own rule; this helper only resolves
// the boundary, it never decides success or failure).
func resolveCut(hist *History, naiveCut int) int {
	messages := transcriptFromHistory(hist)
	cut := naiveCut
	if cut > len(messages) {
		cut = len(messages)
	}
	if cut < 0 {
		cut = 0
	}

	for cut > 0 {
		cut = hist.markBoundaryAtOrBefore(cut)
		if cut == 0 {
			return 0
		}
		straddle, open := earliestOpenMessageIndex(messages[:cut])
		if !open {
			return cut
		}
		// AD-1 step 3: a pair straddles the boundary — retract to the
		// earliest open call's own entry index and re-verify from the
		// top (re-applying step 1's mark retraction to the new
		// candidate). straddle < cut always (resolveOpenSet only ever
		// reports an index inside the scanned prefix), so this
		// strictly decreases every iteration.
		cut = straddle
	}
	return 0
}

// earliestOpenMessageIndex reports the message index of the earliest
// tool call left open once every message has been scanned in order, by
// the same correlation [resolveOpenSet] runs for an ordinary append —
// a read-only replay over the public read-back, never a reach into
// History's private pairing state (R-CMP-004: "MUST NOT widen
// History's exported surface"). Mirrors commitCloseTurnOp's own
// "h.open[0] is the first-issued offender" reading: resolveOpenSet
// appends new opens in issuance order and only ever removes a resolved
// one, so the earliest remaining open call is always at index 0.
func earliestOpenMessageIndex(messages []ai.Message) (int, bool) {
	var open []openCall
	for i, m := range messages {
		open, _ = resolveOpenSet(open, m, i)
	}
	if len(open) == 0 {
		return 0, false
	}
	return open[0].entryIndex, true
}

// lastCompactionTurnIDCounter mints compaction's own turn identities, in
// a namespace disjoint from mintLoopTurnID's "turn-lsk-" (R-CMP-008): a
// reader can never confuse a compaction bracket's TurnID with a model
// turn's.
var lastCompactionTurnIDCounter atomic.Uint64

// mintCompactionTurnID returns a fresh TurnID no other compaction
// operation in this process has minted.
func mintCompactionTurnID() TurnID {
	return TurnID("turn-cmp-" + strconv.FormatUint(lastCompactionTurnIDCounter.Add(1), 10))
}

// emitCompactionFailedArm closes the compaction bracket on a failure arm
// (R-CMP-003, R-CMP-009, R-CMP-010): compaction_finished and
// compaction_failed never both appear for one operation, so every
// non-success exit of runCompaction routes through here exactly once.
// It emits compaction_failed carrying a typed failure derived from
// cause via the same typedHarnessFailureFromError this file's siblings
// use, then turn_end — outcome TurnOutcomeAborted carries that same
// failure (TurnEnd.validate's failure-iff-aborted rule); any other
// outcome carries none. Construction is best-effort, matching
// wrapHarnessFailure's own callers (failRun, windDownRun): on a
// construction failure the affected emission is skipped, and cause is
// always returned as the operation's own error regardless, so a caller
// never loses the underlying reason to a secondary construction defect.
//
// AG-22 (design D-B, D-D, R-AGO-002, R-AGO-007; CRITICAL-2 correction):
// compactionClose is runCompaction's own compaction span's terminal-
// attribute state — always a valid pointer to a real local in
// runCompaction's own frame, regardless of whether the compaction span
// itself ever opened (D-D's practically-unreachable iff-rule edge:
// compaction_started never constructed). Set here, uniformly, at this
// ONE shared closing function every non-success exit of runCompaction
// already routes through; runCompaction's own already-registered defer
// (when the span did open) is what actually ends it — writing these
// fields when no defer was registered is simply inert. The span's
// status is ALWAYS the error status here (this function's whole purpose
// is the fail arm), even though outcome is not always TurnOutcomeAborted
// — the ReplacePrefix-commit-failure caller passes TurnOutcomeFinished
// because the underlying model turn itself finished normally even
// though the compaction's own commit failed. error.type reflects "the
// fail arm ran", not the turn outcome (R-AGO-002's compaction row).
func emitCompactionFailedArm(sink chan<- *Event, stamper *LaneStamper, runID RunID, turnID TurnID, compactionID string, compactionClose *spanCloseState, outcome TurnOutcome, cause error) error {
	failure, ferr := typedHarnessFailureFromError(cause)
	category := ""
	if ferr == nil {
		if fev, cerr := NewCompactionFailed(runID, turnID, compactionID, failure); cerr == nil {
			sendStamped(sink, stamper, fev)
		}
		endFailure := failure
		if outcome != TurnOutcomeAborted {
			endFailure = nil
		}
		if tend, terr := NewTurnEnd(runID, turnID, outcome, endFailure); terr == nil {
			sendStamped(sink, stamper, tend)
		}
		category = failure.Category().String()
	}
	compactionClose.outcome = outcome.String()
	compactionClose.failed = true
	compactionClose.category = category
	return cause
}

// compactionCallOutcome is what draining the compaction's own provider
// stream produced.
type compactionCallOutcome struct {
	completion ai.Completion
	// summary is the candidate summary message reconstructed from the
	// stream's text and reasoning fragments, unchanged — R-CMP-005:
	// Layer 2 authors no wrapper, no prefix, no re-serialisation. It is
	// meaningful only when err == nil and completion.FinishReason() ==
	// ai.FinishReasonStop.
	summary ai.Message
	err     error
}

// drainCompactionCall drains pCh, accumulating text and reasoning
// fragments into a candidate summary message. It reconstructs no
// agent-level bracket event: R-CMP-008's bracket carries only
// turn_start, compaction_started, cost_turn, compaction_finished |
// compaction_failed and turn_end — never a message bracket — so
// accumulation here is silent by design, the same text/reasoning
// handling turnAccumulator (loop.go) performs, without its per-fragment
// emission half. It never dispatches a tool call: compaction's request
// carries no Tools (asserted by runCompaction before this is ever
// reached), so a model that attempts one leaves FinishReasonToolCalls
// as the completion's own finish reason, which the caller's non-Stop
// check already routes to the failed arm — no separate tool-call
// rejection is needed here.
func drainCompactionCall(pCh <-chan ai.Event) compactionCallOutcome {
	var textFragments []string
	var reasoningFragments []byte
	var reasoningToken []byte
	hasReasoningToken := false
	var completion ai.Completion
	gotCompletion := false
	var fatal error

	for ev := range pCh {
		switch ev.Kind() {
		case ai.EventKindTextDelta:
			if d, ok := ev.TextDelta(); ok {
				textFragments = append(textFragments, d.Delta())
			}
		case ai.EventKindReasoningDelta:
			if d, ok := ev.ReasoningDelta(); ok {
				reasoningFragments = append(reasoningFragments, d.Fragment()...)
			}
		case ai.EventKindReasoningBlockEnd:
			if be, ok := ev.ReasoningBlockEnd(); ok {
				if tok, has := be.Token(); has {
					reasoningToken = append([]byte(nil), tok...)
					hasReasoningToken = true
				}
			}
		case ai.EventKindCompletion:
			completion, _ = ev.Completion()
			gotCompletion = true
		case ai.EventKindError:
			if payload, ok := ev.ErrorPayload(); ok {
				fatal = payload
			}
		}
	}

	if fatal != nil {
		return compactionCallOutcome{err: fatal}
	}
	if !gotCompletion {
		return compactionCallOutcome{err: errors.New("compaction: provider stream closed without a completion")}
	}

	var parts []ai.Part
	if len(reasoningFragments) > 0 || hasReasoningToken {
		var tokenArg []byte
		if hasReasoningToken {
			tokenArg = reasoningToken
		}
		if part, perr := ai.NewReasoning(string(reasoningFragments), tokenArg); perr == nil {
			parts = append(parts, part)
		}
	}
	if len(textFragments) > 0 {
		joined := strings.Join(textFragments, "")
		if part, perr := ai.NewText(joined); perr == nil {
			parts = append(parts, part)
		}
	}

	var summary ai.Message
	if len(parts) > 0 {
		summary, _ = ai.NewMessage(ai.RoleAssistant, parts...)
	}
	return compactionCallOutcome{completion: completion, summary: summary}
}

// runCompaction is the one internal operation both compaction doors
// invoke (R-CMP-001): a dedicated turn bracket wrapping exactly one
// non-tool completion request on the injected provider, cut resolution
// (AD-1), the prefix-replacement commit, and the compaction-family
// events, in R-CMP-008's order. total is mutated directly through its
// pointer — the explicit fold (R-CMP-003, S-CST-016): the harness's own
// turnSink-draining forwarder never sees a compaction event, since
// these are emitted on sink directly via sendStamped, never through
// turnSink.
//
// It returns the compaction's own outcome — nil on a committed
// replacement, the operation's cause otherwise — for a caller that
// needs it (Harness.Compact's own run-close outcome). The
// strategy-triggered door (harness.go) discards this return value: a
// compaction failure is visible on the stream and nowhere else
// (R-CMP-010, R-CMP-013) and never interrupts the run.
//
// chain is AG-20's pre-compact chain (R-HKS-003, design AD-8): an
// unexported parameter the harness supplies from Harness.Hooks —
// never smuggled through req.Options, which the misplaced-options
// rejection below refuses. A nil/empty chain is inert: no invocation,
// no CompactionPlan value, byte-for-byte today's path.
func runCompaction(ctx context.Context, hist *History, req CompactionRequest, runID RunID, stamper *LaneStamper, sink chan<- *Event, total *costAccumulator, chain []PreCompactHook) error {
	compactionTurnID := mintCompactionTurnID()
	compactionID := string(compactionTurnID)

	if turnStart, terr := NewTurnStart(runID, compactionTurnID); terr == nil {
		sendStamped(sink, stamper, turnStart)
	}
	// AG-22 (design D-A, D-D, R-AGO-002, R-AGO-007; CRITICAL-2
	// correction): the compaction span opens co-located with
	// compaction_started's own emission, ambiently acquired from
	// whatever's already on ctx (the run span, from either door —
	// Harness.Run's turn boundary or Harness.Compact). compactionSpan
	// stays nil iff NewCompactionStarted itself never constructed
	// (compactionID is always non-empty in practice, so this is D-D's
	// iff-rule edge, not a reachable path) — its finalizer is only
	// registered when the span actually opens, matching that no
	// compaction_started event was ever emitted on that arm either.
	//
	// The finalizer is registered as a `defer` IMMEDIATELY at open —
	// never at an individual return site — so it runs on every exit
	// from runCompaction, including a panic, per R-AGO-007's own MUST.
	// Every closing site below (emitCompactionFailedArm's one shared
	// body, and the success tail) now WRITES compactionClose's fields
	// instead of calling finalizeCompactionSpan itself.
	var compactionSpan trace.Span
	var compactionClose spanCloseState
	if started, serr := NewCompactionStarted(runID, compactionTurnID, compactionID); serr == nil {
		sendStamped(sink, stamper, started)
		tracer := tracerFromContext(ctx)
		ctx, compactionSpan = tracer.Start(ctx, compactionSpanName, trace.WithAttributes(
			attribute.String(runIDKey, string(runID)),
			attribute.String(turnIDKey, string(compactionTurnID)),
			attribute.String(compactionIDKey, compactionID),
		))
		defer func() {
			finalizeCompactionSpan(compactionSpan, compactionClose.outcome, compactionClose.summaryID, compactionClose.failed, compactionClose.category)
		}()
	}

	// Every failure from here through the provider call inclusive is
	// the iff's "no completion" arm: aborted, no cost_turn (R-CMP-003).
	if req.Instruction == "" {
		return emitCompactionFailedArm(sink, stamper, runID, compactionTurnID, compactionID, &compactionClose, TurnOutcomeAborted,
			ai.Invalid(ai.ErrEmpty, ai.At("instruction")))
	}
	// AG-20 (R-HKS-001, R-CMP-015, S-CMP-042): Hooks joins the
	// misplaced-options rejection — the pre-compact chain reaches this
	// operation ONLY as the explicit chain parameter above, never
	// smuggled through the request's own options.
	if req.Options.Tools != nil || req.Options.Continuation != nil || !req.Options.Hooks.isZero() {
		return emitCompactionFailedArm(sink, stamper, runID, compactionTurnID, compactionID, &compactionClose, TurnOutcomeAborted,
			ai.Invalid(ai.ErrMisplaced, ai.At("options")))
	}

	cut := resolveCut(hist, req.Cut)
	if cut == 0 {
		return emitCompactionFailedArm(sink, stamper, runID, compactionTurnID, compactionID, &compactionClose, TurnOutcomeAborted,
			ai.Invalid(ai.ErrEmpty, ai.At("cut")))
	}
	// Span derivation (R-CMP-012, design AD-9) is captured BEFORE the
	// commit below, which shrinks hist's own marks. Structurally
	// unreachable in practice — resolveCut only ever returns 0 or an
	// actual mark's own count, so markSpan always finds it — but
	// checked rather than assumed, per R-CMP-012's own "MUST NOT
	// fabricate a turn identifier" rule.
	startTurnID, endTurnID, spanOK := hist.markSpan(cut)
	if !spanOK {
		return emitCompactionFailedArm(sink, stamper, runID, compactionTurnID, compactionID, &compactionClose, TurnOutcomeAborted,
			ai.Invalid(ai.ErrEmpty, ai.At("span")))
	}

	// AG-20 (R-HKS-003, design AD-8): the pre-compact chain, spliced
	// here — after the naive cut is resolved and the span derived,
	// strictly before the provider call below — reaches BOTH doors
	// (R-CMP-001) through this one operation; no second splice is
	// added. len(chain) > 0 guards inertness (R-HKS-012): a nil/empty
	// chain constructs no CompactionPlan value and this path stays
	// byte-for-byte today's.
	if len(chain) > 0 {
		plan := CompactionPlan{cut: cut, span: CompactionSpan{StartTurnID: startTurnID, EndTurnID: endTurnID}}
		var herr error
		for _, hook := range chain {
			plan, herr = hook(ctx, plan)
			if herr != nil {
				return emitCompactionFailedArm(sink, stamper, runID, compactionTurnID, compactionID, &compactionClose, TurnOutcomeAborted, herr)
			}
		}
		// The chain's returned cut is re-resolved UNCONDITIONALLY
		// (R-CMP-004 as amended) — an implementation MUST NOT skip
		// this on the grounds the plan "looks unchanged": skipping is
		// exactly what splits a call/result pair (bite S-HKS-053 /
		// S-CMP-041). The span is then re-derived from the
		// (possibly moved) resolved cut, pre-commit, so
		// NewCompactionFinished below reports the span that was
		// actually used.
		cut = resolveCut(hist, plan.Cut())
		if cut == 0 {
			return emitCompactionFailedArm(sink, stamper, runID, compactionTurnID, compactionID, &compactionClose, TurnOutcomeAborted,
				ai.Invalid(ai.ErrEmpty, ai.At("cut")))
		}
		startTurnID, endTurnID, spanOK = hist.markSpan(cut)
		if !spanOK {
			return emitCompactionFailedArm(sink, stamper, runID, compactionTurnID, compactionID, &compactionClose, TurnOutcomeAborted,
				ai.Invalid(ai.ErrEmpty, ai.At("span")))
		}
	}

	prefix := transcriptFromHistory(hist)[:cut]
	pReq, berr := buildLoopRequest(req.Options, req.Instruction, prefix)
	if berr != nil {
		return emitCompactionFailedArm(sink, stamper, runID, compactionTurnID, compactionID, &compactionClose, TurnOutcomeAborted, berr)
	}

	pCh, serr := req.Provider.Stream(ctx, pReq)
	if serr != nil {
		return emitCompactionFailedArm(sink, stamper, runID, compactionTurnID, compactionID, &compactionClose, TurnOutcomeAborted, serr)
	}

	outcome := drainCompactionCall(pCh)
	if outcome.err != nil {
		cause := outcome.err
		if cancelCause := context.Cause(ctx); errors.Is(cancelCause, ErrInterrupted) || errors.Is(cancelCause, ErrShutdown) {
			cause = cancelCause
		}
		return emitCompactionFailedArm(sink, stamper, runID, compactionTurnID, compactionID, &compactionClose, TurnOutcomeAborted, cause)
	}

	// A completion was obtained: cost_turn is emitted from here on, on
	// every remaining arm (R-CMP-003's iff — the "commit succeeds" row
	// and the "rejected" row both answer yes). This emission site
	// performs the accumulation itself against total (S-CST-016).
	if costEvent, cerr := newCostTurnFromUsage(runID, compactionTurnID, CostLabelFinal, outcome.completion.Usage()); cerr == nil {
		sendStamped(sink, stamper, costEvent)
		if ct, ok := costEvent.CostTurn(); ok {
			total.add(ct)
		}
	}

	if outcome.completion.FinishReason() != ai.FinishReasonStop {
		return emitCompactionFailedArm(sink, stamper, runID, compactionTurnID, compactionID, &compactionClose, outcomeForFinish(outcome.completion.FinishReason()),
			ai.Invalid(ai.ErrOutOfRange, ai.At("finish_reason")))
	}

	// The commit point (R-CMP-010, design AD-11): unreachable until the
	// cut was resolved and mark-aligned, and the provider returned a
	// Stop completion. Nothing above this line has touched hist's own
	// pointee.
	if rerr := hist.ReplacePrefix(cut, outcome.summary); rerr != nil {
		return emitCompactionFailedArm(sink, stamper, runID, compactionTurnID, compactionID, &compactionClose, TurnOutcomeFinished, rerr)
	}

	if fev, ferr := NewCompactionFinished(runID, compactionTurnID, compactionID, CompactionSpan{StartTurnID: startTurnID, EndTurnID: endTurnID}, outcome.summary.ID().String()); ferr == nil {
		sendStamped(sink, stamper, fev)
	}
	if tend, terr := NewTurnEnd(runID, compactionTurnID, TurnOutcomeFinished, nil); terr == nil {
		sendStamped(sink, stamper, tend)
	}
	// AG-22 (design D-B, D-D, R-AGO-002, R-AGO-007; CRITICAL-2
	// correction): the success tail — exactly ONE span covers the whole
	// compaction bracket (R-AGO-002's "one bracket, one span" clause; no
	// separate turn span was ever opened alongside it).
	// cachicamas.compaction.summary_id is set from the SAME expression
	// NewCompactionFinished above was given, never re-read from its own
	// (best-effort, possibly-failed) construction. Recording these
	// fields is harmless even when compactionSpan is nil (no defer was
	// ever registered in that case, so nothing reads them) — the defer
	// registered at open is what actually ends the span.
	compactionClose.outcome = TurnOutcomeFinished.String()
	compactionClose.summaryID = outcome.summary.ID().String()
	return nil
}

// ErrRunInFlight is Harness.Compact's typed refusal sentinel when a run
// is already in flight (R-CMP-011) — a plain Go sentinel error, the
// ErrInterrupted/ErrShutdown house pattern, not a Layer 1 rule class:
// backend/agent/src/ai/ stays untouched.
var ErrRunInFlight = errors.New("agent: compaction refused, a run is in flight")

// Compact is the on-demand door onto the same operation the context
// strategy's verdict reaches at the run driver's turn boundary
// (R-CMP-001, R-CMP-011, design AD-10). It emits its own minimal,
// independently CheckStream-valid stream: run_start, the compaction
// bracket runCompaction builds, the run's cumulative cost event, and a
// run close — never a bare compaction bracket with no run bracket,
// since CheckStream rejects any stream lacking a complete run bracket.
//
// It refuses, synchronously and typed, whenever a run is in flight —
// detected through the SAME signalMu/cancelRun state Interrupt and
// Shutdown already read, never a new field: History carries no mutex,
// so a concurrent on-demand mutation would be a data race whether or
// not a turn is open, and run-in-flight subsumes turn-in-flight
// (R-CMP-011's own evidence). After a terminal shutdown it refuses on
// the same route as a post-shutdown prompt (ErrPromptAfterShutdown).
// Neither refusal emits any event on sink and neither queues the
// demand in any form.
func (h *Harness) Compact(ctx context.Context, req CompactionRequest, sink chan<- *Event) error {
	h.signalMu.Lock()
	if h.shutdown {
		h.signalMu.Unlock()
		return ErrPromptAfterShutdown
	}
	if h.cancelRun != nil {
		h.signalMu.Unlock()
		return ErrRunInFlight
	}
	h.signalMu.Unlock()

	hist := h.History
	if hist == nil {
		hist = NewHistory()
	}

	runID := mintHarnessRunID()
	stamper := &LaneStamper{}
	defer close(sink)

	runStart, err := NewRunStart(runID)
	if err != nil {
		return err
	}

	// AG-22 (design D-A, D-D, R-AGO-001, R-AGO-002, R-AGO-007;
	// CRITICAL-2 correction): the run span opens here, co-located with
	// run_start — after both refusals above, neither of which ever
	// emits an event. h.TracerProvider defaults to the tracing API's
	// own no-op provider when nil (tracerFromHarness).
	//
	// The finalizer is registered as a `defer` IMMEDIATELY at open —
	// never at an individual return site — so it runs on every exit
	// from Compact, including a panic, per R-AGO-007's own MUST.
	// Compact's own two closing sites below (the failure arm, the
	// success tail) now WRITE runClose's fields instead of calling
	// finalizeRunSpan themselves.
	tracer := tracerFromHarness(h.TracerProvider)
	ctx, runSpan := tracer.Start(ctx, invokeAgentSpanName, trace.WithAttributes(
		attribute.String(genAIOperationNameKey, genAIOperationNameInvokeAgent),
		attribute.String(runIDKey, string(runID)),
	))
	var runClose spanCloseState
	defer func() {
		finalizeRunSpan(runSpan, runClose.outcome, runClose.failed, runClose.category)
	}()
	if parentID, ok := runStart.Parent(); ok {
		runSpan.SetAttributes(attribute.String(runParentIDKey, string(parentID)))
	}

	sendStamped(sink, stamper, runStart)

	var total costAccumulator
	compactErr := runCompaction(ctx, hist, req, runID, stamper, sink, &total, h.Hooks.PreCompact)

	if sessionEvent, serr := total.sessionEvent(runID, CostLabelFinal); serr == nil {
		sendStamped(sink, stamper, sessionEvent)
	}

	if compactErr != nil {
		category := ""
		if failure, ferr := typedHarnessFailureFromError(compactErr); ferr == nil {
			if runEnd, rerr := NewRunEnd(runID, RunOutcomeFailed, failure); rerr == nil {
				sendStamped(sink, stamper, runEnd)
			}
			category = failure.Category().String()
		}
		runClose.outcome = RunOutcomeFailed.String()
		runClose.failed = true
		runClose.category = category
		return compactErr
	}

	runEnd, rerr := NewRunEnd(runID, RunOutcomeCompleted, nil)
	if rerr == nil {
		sendStamped(sink, stamper, runEnd)
	}
	runClose.outcome = RunOutcomeCompleted.String()
	return nil
}
