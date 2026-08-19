// Package agent is Layer 2 of the cachicamas agent stack. This file
// (harness.go) hosts AG-13.1/AG-13.2/AG-13.3 — the multi-turn run driver
// (R-RUN-001..012).
//
// `Harness` drives `Turn` (loop.go) N times to a run's terminal decision,
// through the same public one-turn surface the skeleton's external tests
// use (`R-RUN-006`, charter AG-13.1 scenario 3) — it owns the run bracket,
// the run identity, the shared `LaneStamper`, the steering queue and the
// scheduler value, and it never calls `Schedule` directly (the R-LSK-006
// reconciliation, design.md §"The Reconciliation").
//
// Value-form, no constructor, mirroring `Scheduler`'s own posture
// (tool.go:229): zero surface beyond two methods, `Run` and `Steer`. Nil
// optional fields are resolved to defaults at `Run` entry into LOCALS —
// the caller's own struct fields are never mutated, with exactly one
// recorded exception: `Run` sets `LeaveSinkOpen` once on the scheduler it
// drives, before the first turn (R-RUN-001, R-RUN-012).
package agent

import (
	"context"
	"errors"
	"slices"
	"strconv"
	"sync"
	"sync/atomic"

	"github.com/cachicamas/backend/agent/src/ai"
)

// lastHarnessRunIDCounter mints run-hrn-<n> identities — a package-local
// monotonic counter, provenance-distinct from the loop's own run-lsk-
// minter (R-RUN-004). A reader of any single event can therefore tell a
// harness-driven run from a bare one-turn Turn run by identity provenance
// alone.
var lastHarnessRunIDCounter atomic.Uint64

// mintHarnessRunID returns a fresh RunID no other Harness.Run call in
// this process has minted.
func mintHarnessRunID() RunID {
	return RunID("run-hrn-" + strconv.FormatUint(lastHarnessRunIDCounter.Add(1), 10))
}

// Harness drives one run to its terminal decision (R-RUN-001). Exported
// configuration fields, no constructor, no interface — the AG-04..AG-12
// rule that a concrete boundary type stays a struct until a second
// implementation actually arrives.
type Harness struct {
	// Provider is the model provider every turn streams from. Required.
	Provider ai.ModelProvider

	// System is the system prompt every turn is built with.
	System string

	// Turn is the per-turn base options (Model, MaxTokens, PreRequestHook,
	// Tools, PermissionPolicy). Run derives each turn's own Continuation
	// from a copy of this value — Turn is passed by value, so the
	// caller's own struct is never mutated by that derivation.
	Turn TurnOptions

	// Scheduler is the caller-owned wake handle: injecting it (rather
	// than handing one back) is what makes WakeParked reachable while a
	// turn is blocked (design "Decision 2"). Nil constructs one. Run
	// sets LeaveSinkOpen on the resolved value before the first turn —
	// the sole caller-field mutation this type performs.
	Scheduler *Scheduler

	// History is the caller-owned transcript. Nil constructs one via
	// NewHistory.
	History *History

	// RetryAttempts is H, the attempt bound for one logical turn
	// (R-RTY-005): the harness re-invokes Turn for the same logical
	// turn while R-RTY-001's predicate says "retry", up to H total
	// Turn invocations (one initial attempt plus at most H-1
	// retries). Zero-default: a non-positive value selects
	// defaultRetryAttempts (retry_policy.go), the
	// Scheduler.WindDownBound idiom (scheduler.go:611-613).
	RetryAttempts int

	// RetryTiming is the injected clock and wait-function seam a G4
	// retry's backoff wait is built from (R-RTY-006, design AD-5).
	// Zero-default: an unconfigured value resolves to production
	// defaults via applyRetryTimingDefaults (retry_policy.go).
	RetryTiming RetryTiming

	// Failover is the failover seam consulted exactly once at G5, the
	// attempt bound's exhaustion (R-RTY-010, design AD-7). Nil-default:
	// a nil value is never called and behaves as a decline
	// (R-RTY-011, the inertness pin) — NoOpFailoverPolicy{} is the one
	// shipped, installable implementation with the identical behavior.
	Failover FailoverPolicy

	// ContextStrategy is the context seam consulted exactly once per
	// LOGICAL turn, at the turn boundary below (R-CTX-001). Nil-default:
	// a nil value is never consulted and no accounting is resolved — the
	// nil path is byte-for-byte the pre-AG-17 path (R-CTX-004, the
	// inertness pin) — NoOpContextStrategy{} is the one shipped,
	// installable implementation with the identical behavior.
	ContextStrategy ContextStrategy

	// ContextBudget is Layer 3's stated token budget, possibly absent
	// (R-CTX-005). Zero-default: the zero value reads as absent, never
	// as a budget of zero tokens.
	ContextBudget ContextBudget

	// queue is the steering FIFO (R-RUN-008). Zero-value ready — a
	// Harness{} constructs one implicitly.
	queue steeringQueue

	// signalMu guards cancelRun and shutdown (AG-14, R-CAN-001,
	// R-CAN-004). One mutex, not sync.Once: a shut-down-then-reused
	// harness (R-CAN-002) needs a FRESH cancel registration on each
	// Run call, which sync.Once cannot give back once fired; atomics
	// cannot couple "read the flag" with "invoke the cancel func"
	// atomically the way this mutex does.
	signalMu sync.Mutex

	// cancelRun is the current run's cancel-cause function, live only
	// while a Run call is in flight (set at Run entry, cleared at Run
	// exit, both under signalMu). Interrupt/Shutdown invoke it under
	// the same mutex; between runs it is nil, so a signal fired with
	// no run in flight is a documented no-op.
	cancelRun context.CancelCauseFunc

	// shutdown is the terminal, one-way refusal flag Shutdown latches
	// (R-CAN-005). It is per-value bookkeeping only — no transcript,
	// never resumed — and does not pre-empt AG-21's cross-run state
	// (agent-run-driver/spec.md:72). Not yet consulted at Run entry;
	// that wiring is Phase 7.
	shutdown bool
}

// Interrupt fires the interrupt signal (R-CAN-001, R-CAN-002): under
// signalMu, if a run is in flight it invokes that run's cancel-cause
// function with ErrInterrupted. A second Interrupt during the first's
// wind-down, or an Interrupt with no run in flight, observes either a
// live cancel func — invoking it again is Go's documented no-op on an
// already-cancelled context — or nil, and either way does nothing
// further (R-CAN-004): no channel is ever closed by a signal, so
// there is no double-close to race.
func (h *Harness) Interrupt() {
	h.signalMu.Lock()
	defer h.signalMu.Unlock()
	if h.cancelRun != nil {
		h.cancelRun(ErrInterrupted)
	}
}

// Shutdown fires the shutdown signal (R-CAN-001, R-CAN-005): under
// signalMu, it invokes the in-flight run's cancel-cause function with
// ErrShutdown (the same no-op-safe posture as Interrupt) and latches
// the terminal, one-way shutdown flag. The flag's consequence at Run
// entry (the typed post-shutdown refusal) is wired in Phase 7; this
// method only sets it.
func (h *Harness) Shutdown() {
	h.signalMu.Lock()
	defer h.signalMu.Unlock()
	h.shutdown = true
	if h.cancelRun != nil {
		h.cancelRun(ErrShutdown)
	}
}

// steeringQueue is Harness's own FIFO mailbox for Steer (R-RUN-008).
// Zero-value ready: mutex + FIFO slice + closed flag.
type steeringQueue struct {
	mu     sync.Mutex
	items  []ai.Message
	closed bool
}

// enqueue appends msg to the queue in arrival order. It returns the typed
// rejection once the queue has been closed by the run's terminal decision
// (R-RUN-001's post-terminal Steer contract) and never drops a message
// accepted before that point (zero drops).
func (q *steeringQueue) enqueue(msg ai.Message) error {
	q.mu.Lock()
	defer q.mu.Unlock()
	if q.closed {
		return ai.Invalid(ai.ErrMisplaced, ai.At("steering"))
	}
	q.items = append(q.items, msg)
	return nil
}

// drain removes and returns every currently queued message, in arrival
// order, leaving the queue open but empty. Called at every turn boundary
// before the next request transcript is built (design.md run algorithm
// step 4a) — not the atomic terminal-decision step, see takeOrClose.
func (q *steeringQueue) drain() []ai.Message {
	q.mu.Lock()
	defer q.mu.Unlock()
	if len(q.items) == 0 {
		return nil
	}
	out := q.items
	q.items = nil
	return out
}

// takeOrClose is the atomic terminal-decision step (R-RUN-002): under one
// critical section, either take every queued message (the queue stays
// open, a new turn is warranted — a message queued during the final turn
// yields a new turn, never a drop) or, when none are queued, close the
// queue so a later Steer is rejected typed. A check-then-close without
// the lock would drop a Steer already accepted with a nil return; there
// is no such window here.
func (q *steeringQueue) takeOrClose() (took bool, messages []ai.Message) {
	q.mu.Lock()
	defer q.mu.Unlock()
	if len(q.items) > 0 {
		out := q.items
		q.items = nil
		return true, out
	}
	q.closed = true
	return false, nil
}

// close marks the queue closed under the same critical section every
// other queue operation uses. It is idempotent — closing an
// already-closed queue (the ordinary takeOrClose success path) is a
// no-op — so every exit from Run can call it unconditionally, including
// the R-RUN-011 failure path, without a check-then-close window that
// spec.md:89 forbids. See Run's own deferred call for why this is the
// single place every termination path flows through.
func (q *steeringQueue) close() {
	q.mu.Lock()
	defer q.mu.Unlock()
	q.closed = true
}

// reopen clears the closed flag under the same critical section every
// other queue operation uses (AG-14, R-CAN-002 delta, R-RUN-001).
// Called at Run entry, unless the terminal shutdown flag is set
// (R-CAN-005, wired in Phase 7): a value whose prior run ended —
// completed, failed, or interrupted — left the queue closed, and a
// new run on that same value must accept Steer again rather than
// meeting the state the prior run left behind.
func (q *steeringQueue) reopen() {
	q.mu.Lock()
	defer q.mu.Unlock()
	q.closed = false
}

// Steer offers msg to the in-flight run (R-RUN-001, R-RUN-008). A nil
// return guarantees msg enters the transcript before the run's next
// provider call — zero drops. After the run's terminal decision it
// returns the typed rejection ai.Invalid(ai.ErrMisplaced,
// ai.At("steering")); Steer never blocks and never touches the loop.
func (h *Harness) Steer(msg ai.Message) error {
	return h.queue.enqueue(msg)
}

// sendStamped stamps ev through stamper and sends it on sink — the
// harness's own run-bracket emission (design.md run algorithm steps 2
// and 6). Distinct from loop.go's own emitStamped: the harness reaches
// the loop through no channel but Turn's own public one-turn surface
// (R-RUN-006).
func sendStamped(sink chan<- *Event, stamper *LaneStamper, ev Event) {
	stamped := stamper.Stamp(ev)
	sink <- &stamped
}

// transcriptFromHistory reads back hist's committed entries as the
// []ai.Message slice the next Turn call's request transcript is built
// from (design.md run algorithm step 4b) — through History's own public
// Entries accessor, never a second store.
func transcriptFromHistory(hist *History) []ai.Message {
	entries := hist.Entries()
	out := make([]ai.Message, len(entries))
	for i, e := range entries {
		out[i] = e.Message()
	}
	return out
}

// wrapHarnessFailure wraps an arbitrary error as the typed *Failure a
// run_end(RunOutcomeFailed) event carries (R-RUN-011), through the
// package's own public constructors only — ai.MidStreamFailure (Layer 1)
// then NewFailure (this package's own typed-failure surface). Kept local
// to harness.go rather than reusing scheduler.go's own equivalent
// helper, so this file's failure path never reaches past what R-RUN-006's
// source-scan guard already permits.
func wrapHarnessFailure(cause error) (*Failure, error) {
	aiFailure, ferr := ai.MidStreamFailure(ai.FailureReport{
		Category: ai.FailureCategoryUnavailable,
		Cause:    cause,
	}, false)
	if ferr != nil {
		return nil, ferr
	}
	return NewFailure(aiFailure)
}

// typedHarnessFailureFromError is wrapHarnessFailure's
// conditionally-routed sibling (AG-15.4, R-RTY-012, design AD-4; the
// AG-14 typedCancellationFailureFromError shape, scheduler.go:1088-
// 1099): when cause already IS an *ai.Failure, it is preserved WHOLE
// by wrapping the identical pointer through NewFailure — the same
// precedent loop.go's own turn_end emission already uses
// (loop.go:387-388) — rather than reconstructed from a
// Category-only report, which is what loses every other field
// (Retryable, RetryAfter, PartialOutput, Delivery, the cause chain).
// When cause is NOT an *ai.Failure, this routes to the untouched
// wrapHarnessFailure, so a plain-error cause keeps today's
// byte-identical Unavailable behavior.
func typedHarnessFailureFromError(cause error) (*Failure, error) {
	var aiFailure *ai.Failure
	if errors.As(cause, &aiFailure) {
		return NewFailure(aiFailure)
	}
	return wrapHarnessFailure(cause)
}

// windDownRun is the cancellation counterpart to failRun (R-CAN-002,
// R-CAN-005, the R-RUN-011 carve-out): synthesize orphans over the
// transcript (R-HIS-007's first production caller — idempotent per
// R-HIS-008, so a turn whose results were already committed
// synthesizes nothing), close the turn, emit run_end carrying the
// firing signal's own outcome with no *Failure, then return cause
// unwrapped as Run's own error — the caller's errors.Is/errors.As
// chain reaches the sentinel directly, mirroring failRun's own
// posture for its typed rejection. Best-effort on the two history
// calls' own errors, exactly as failRun is best-effort on
// wrapHarnessFailure/NewRunEnd: cause is the authoritative return
// value regardless.
func (h *Harness) windDownRun(sink chan<- *Event, stamper *LaneStamper, runID RunID, hist *History, total costAccumulator, cause error) (ai.Message, ai.FinishReason, error) {
	_, _ = hist.SynthesizeOrphans()
	_ = hist.CloseTurn()

	outcome := RunOutcomeInterrupted
	if errors.Is(cause, ErrShutdown) {
		outcome = RunOutcomeShutdown
	}
	// AG-16 (R-CST-006, R-CAN-002's amended enumerated order): emit
	// the run-scoped final cost figure immediately before the
	// run-close — tokens spent before a cancellation are real spend.
	// Best-effort and independent of the run-close's own construction:
	// if the payload cannot be built, the emission is skipped and the
	// wind-down proceeds unchanged.
	if sessionEvent, serr := total.sessionEvent(runID, CostLabelFinal); serr == nil {
		sendStamped(sink, stamper, sessionEvent)
	}
	if runEnd, rerr := NewRunEnd(runID, outcome, nil); rerr == nil {
		sendStamped(sink, stamper, runEnd)
	}
	return ai.Message{}, 0, cause
}

// failRun is R-RUN-011's failure path: on a non-nil Turn error, emit
// run_end(RunOutcomeFailed, failure) built through the public
// constructors, then return — no append, no CloseTurn, no retry or
// fallback. cause is returned unwrapped as Run's own error, so the
// caller's errors.Is/errors.As chain reaches the turn's own typed
// rejection; the wrapped *Failure is used only for the event payload.
func (h *Harness) failRun(sink chan<- *Event, stamper *LaneStamper, runID RunID, total costAccumulator, cause error) (ai.Message, ai.FinishReason, error) {
	// AG-16 (R-CST-006): tokens spent before a failure are real spend
	// — emit the run-scoped final cost figure immediately before the
	// run-close, best-effort and independent of the run-close's own
	// construction. The failing turn itself contributed no cost_turn
	// (it closed aborted), so total already carries only what really
	// ran.
	if sessionEvent, serr := total.sessionEvent(runID, CostLabelFinal); serr == nil {
		sendStamped(sink, stamper, sessionEvent)
	}
	if failure, ferr := typedHarnessFailureFromError(cause); ferr == nil {
		if runEnd, rerr := NewRunEnd(runID, RunOutcomeFailed, failure); rerr == nil {
			sendStamped(sink, stamper, runEnd)
		}
	}
	return ai.Message{}, 0, cause
}

// Run drives one run to its terminal decision: repeated Turn calls
// sharing one run identity and one contiguous lane, until a terminal
// finish reason with an empty steering queue (R-RUN-002, R-RUN-003). See
// design.md's run algorithm for the full description.
//
// Phase 2 increment: the loop iterates on ToolCalls/PauseTurn and
// resolves every other finish reason atomically against the steering
// queue. The failure path (a non-nil Turn error) lands in a later
// increment of this same file.
func (h *Harness) Run(ctx context.Context, prompt ai.Message, sink chan<- *Event) (ai.Message, ai.FinishReason, error) {
	sched := h.Scheduler
	if sched == nil {
		sched = &Scheduler{}
	}
	// The one caller-field mutation Run performs: the harness needs the
	// per-turn sink left open so it can emit the turn-close event after
	// Schedule returns (R-LSK-001 point 3).
	sched.LeaveSinkOpen = true

	hist := h.History
	if hist == nil {
		hist = NewHistory()
	}

	// AG-14 (R-CAN-001): derive this run's own cancel-cause context
	// once, under signalMu, and store the cancel func so
	// Interrupt/Shutdown can reach it. runCtx — not the caller's bare
	// ctx — is threaded into every downstream call (Turn, and through
	// it Schedule and tool.Run), so a signal reaches the whole tree
	// through the existing ctx parameters with zero signature changes.
	// Cleared at Run exit under the same mutex, then cancelled with a
	// nil cause (Go's documented no-op semantics for an
	// already-terminated run) — the deferred call runs last, after the
	// clear, so a signal racing the very end of Run never dereferences
	// a stale func.
	h.signalMu.Lock()
	// AG-14 (R-CAN-005): a value already shut down refuses every
	// subsequent Run typed, before any identity is minted and before
	// any event is emitted — the terminal, one-way flag Shutdown
	// latches. Checked first in this critical section, ahead of the
	// queue reopen and the cancel-cause derivation below, neither of
	// which this path reaches: the queue stays exactly as the prior
	// run's wind-down left it (closed), and no cancelRun is ever
	// registered for a run that never starts.
	if h.shutdown {
		h.signalMu.Unlock()
		close(sink)
		return ai.Message{}, 0, ErrPromptAfterShutdown
	}
	// AG-14 (R-CAN-002 delta, R-RUN-001): reopen the steering queue in
	// the SAME critical section as the cancel-cause derivation below,
	// unless the terminal shutdown flag is already set — that half
	// (the typed post-shutdown refusal) is Phase 7's wiring above;
	// this is only the reopen a serially-reused, non-shutdown value
	// needs.
	if !h.shutdown {
		h.queue.reopen()
	}
	runCtx, cancel := context.WithCancelCause(ctx)
	h.cancelRun = cancel
	h.signalMu.Unlock()

	// AG-14 remediation (verify-report.md MINOR-3): close(sink) MUST be
	// registered BEFORE the cancelRun-clearing defer below, so LIFO
	// execution clears cancelRun FIRST and closes the sink SECOND. A
	// stream-only consumer that starts run #2 the instant it observes
	// this sink close (R-CAN-005's own shape) can then never observe the
	// close before cancelRun has already been cleared — closing the
	// window where run #2's own registration could be overwritten by
	// this run's own still-pending clear.
	defer close(sink)

	defer func() {
		h.signalMu.Lock()
		h.cancelRun = nil
		h.signalMu.Unlock()
		cancel(nil)
	}()

	// Every exit from this function — the terminal-decision success
	// path below, every failRun call site on the R-RUN-011 failure
	// path, and the NewRunStart early return that follows — MUST leave
	// the queue closed, so a Steer reaching the harness after Run has
	// returned always gets R-RUN-001's typed rejection, never a nil
	// return promising a delivery that can no longer happen. One
	// deferred, idempotent close covers every current return statement
	// AND every future one a later change might add; the success path
	// already closes the queue itself via takeOrClose, so this is then
	// a verified no-op there (verify-report.md MAJOR-1).
	defer h.queue.close()

	runID := mintHarnessRunID()
	stamper := &LaneStamper{}

	// AG-16 (R-CST-004): the run-scoped cumulative, a local in Run's
	// own stack frame — never a Harness field (R-CAN-002). Declared
	// before the run-open so it is in scope for every failRun/
	// windDownRun call site below, including the ones that precede
	// the first turn. Written only by the live forwarder goroutine
	// further down and read only after <-forwarderDone has
	// synchronized with it (close-of-channel happens-before), so
	// every read below is race-free by construction.
	var total costAccumulator

	runStart, err := NewRunStart(runID)
	if err != nil {
		return ai.Message{}, 0, err
	}
	sendStamped(sink, stamper, runStart)

	if err := hist.Append(prompt); err != nil {
		return h.failRun(sink, stamper, runID, total, err)
	}

	var lastMsg ai.Message
	var lastFinish ai.FinishReason

	for {
		// AG-14 (R-CAN-001, R-RUN-011 carve-out): the iteration
		// boundary consults the cause before starting another Turn,
		// so a run whose signal fired between turns winds down
		// instead of making a futile provider call on an
		// already-cancelled context. Checked first in the loop body,
		// every iteration including the first (always uncancelled
		// there, since Interrupt/Shutdown can only reach a live
		// cancelRun, set above).
		if cause := context.Cause(runCtx); errors.Is(cause, ErrInterrupted) || errors.Is(cause, ErrShutdown) {
			return h.windDownRun(sink, stamper, runID, hist, total, cause)
		}

		for _, m := range h.queue.drain() {
			if err := hist.Append(m); err != nil {
				return h.failRun(sink, stamper, runID, total, err)
			}
		}

		transcript := transcriptFromHistory(hist)

		// AG-17 (R-CTX-001, R-CTX-002): the context seam, consulted
		// exactly once per LOGICAL turn — at AG-13's turn boundary,
		// outside the attempt loop below — never per attempt. A
		// per-attempt consultation would let a future compacting
		// verdict (AG-18) mutate the transcript between two attempts
		// of one logical turn, making R-RTY-002's "identical
		// transcript, reused BY REFERENCE" (harness.go:550-561)
		// unprovable by the exact argument its comment relies on.
		// A nil strategy is never consulted and no accounting is
		// resolved: the nil path is byte-for-byte the pre-AG-17 path.
		if h.ContextStrategy != nil {
			h.ContextStrategy.Resolve(runCtx, ContextPrompt{
				Transcript: slices.Clone(transcript),
				Budget:     h.ContextBudget,
				Accounting: resolveTokenAccounting(runCtx, h.Provider, h.Turn, h.System, transcript),
			})
		}

		bound := h.RetryAttempts
		if bound <= 0 {
			bound = defaultRetryAttempts
		}

		// AG-15.2 (R-RTY-006): resolved once per logical turn, mirroring
		// Layer 1's own Loop, which reseeds its jitter source once per
		// pre-stream retry budget (ai/internal/retry/retry.go:78-79) —
		// the harness's analogous unit is one logical turn's own inner
		// attempt loop below, not the whole multi-turn run.
		timing := applyRetryTimingDefaults(h.RetryTiming)
		jitter := newRetryJitter(timing.JitterSeed, timing.NowFunc())

		var msg ai.Message
		var finish ai.FinishReason
		var terr error
		var verdict retryVerdict

		// AG-15.1 (R-RTY-001, R-RTY-002, design AD-2): the inner
		// attempt loop re-invokes Turn for this SAME logical turn on
		// a retry verdict, over the transcript slice built once
		// above and reused BY REFERENCE across attempts — this, not
		// an assumption, is what makes "identical transcript"
		// provable: no failure path writes History (Turn's only
		// history writer is finishContinuationTurn, reached solely on
		// the completion arm), and the steering drain above runs only
		// once per LOGICAL turn, outside this loop, so no steered
		// message can interleave two attempts. This is a fresh inner
		// loop around the Turn call, NOT a continue of the outer run
		// loop above.
		for attempt := 1; ; attempt++ {
			turnSink := make(chan *Event)
			forwarderDone := make(chan struct{})
			go func() {
				defer close(forwarderDone)
				for ev := range turnSink {
					// AG-16 (R-CST-004, R-RUN-003): a pure read on the
					// existing single forwarding path — the event is
					// still forwarded exactly once, unmodified. Every
					// attempt that reaches a Completion emits its own
					// cost_turn inside its own bracket, so this counts
					// retries by construction, with no retry-awareness
					// here at all.
					if ct, ok := ev.CostTurn(); ok {
						total.add(ct)
					}
					sink <- ev
				}
			}()

			turnOpts := h.Turn
			turnOpts.Continuation = &TurnContinuation{
				Run:       runID,
				Stamper:   stamper,
				Scheduler: sched,
				History:   hist,
			}

			msg, finish, terr = Turn(runCtx, h.Provider, h.System, transcript, turnOpts, turnSink)
			<-forwarderDone

			if terr == nil {
				break
			}

			// AG-14 (R-RUN-011 carve-out; R-RTY-001 gate G0, stays
			// ahead of the retry predicate in its existing shape and
			// site, unchanged): a Turn error caused by a harness
			// signal takes the wind-down path, not the failure path
			// and never the retry path — re-checked against
			// context.Cause(runCtx) directly (not by unwrapping
			// terr) so the routing decision is independent of
			// exactly how Turn surfaced the cause. A bare
			// cancellation of the caller's own context, not routed
			// through Interrupt/Shutdown, matches neither sentinel
			// and falls through to the retry predicate below
			// unchanged (the scope line, R-CAN-001).
			if cause := context.Cause(runCtx); errors.Is(cause, ErrInterrupted) || errors.Is(cause, ErrShutdown) {
				return h.windDownRun(sink, stamper, runID, hist, total, cause)
			}

			// R-RTY-001 gates G1-G5, first match wins. Any verdict
			// other than "retry" — surface (G1-G3), or the attempt
			// bound reached (G5, consulting the failover seam below)
			// — falls through to the ordinary failure path below.
			verdict = retryDecision(terr, attempt, bound)
			if verdict != verdictRetry {
				break
			}

			// AG-15.2 (R-RTY-007, R-RTY-008, design AD-6): G4 fired —
			// wait before re-invoking Turn. The wait selects on runCtx,
			// never a background context, so an interrupt or shutdown
			// fired while backing off aborts the wait promptly. On a
			// non-nil sleep error, re-consult the run context's own
			// cause: a match against a cancellation sentinel routes to
			// wind-down exactly like an iteration-boundary signal; a
			// bare cancellation of the caller's own context, matching
			// neither sentinel, falls through to the ordinary failure
			// surface below with the original terr still set — parity
			// with the existing scope rule (R-CAN-001).
			delay := retryWaitDelay(terr, attempt, timing, jitter)
			if serr := timing.SleepFunc(runCtx, delay); serr != nil {
				if cause := context.Cause(runCtx); errors.Is(cause, ErrInterrupted) || errors.Is(cause, ErrShutdown) {
					return h.windDownRun(sink, stamper, runID, hist, total, cause)
				}
				break
			}
		}

		if terr != nil {
			// AG-15.3 (R-RTY-010): G5 fired — the attempt bound was
			// reached, not merely surfaced (G1-G3). Consult the
			// failover seam exactly once, before the terminal report,
			// with the true attempt count and the final attempt's own
			// typed failure. A nil policy is never called (the
			// inertness pin, R-RTY-011); v1's FailoverVerdict is
			// always the declining zero value, so the exhausted-retry
			// report always surfaces regardless of the verdict read.
			if verdict == verdictExhausted && h.Failover != nil {
				var failure *ai.Failure
				if errors.As(terr, &failure) {
					h.Failover.Resolve(runCtx, FailoverPrompt{Attempts: bound, Failure: failure})
				}
			}

			// R-RUN-011: a failed turn ends the run typed, with no
			// append, no CloseTurn, and no fallback. The turn's own
			// typed closing brackets (turn_end(Aborted) on every
			// pre-stream and mid-stream path, per AG-15 D1) were
			// already forwarded by the loop above, inside each
			// attempt's own still-open turn bracket; failRun closes
			// the run's own bracket.
			return h.failRun(sink, stamper, runID, total, terr)
		}

		if err := hist.CloseTurn(); err != nil {
			return h.failRun(sink, stamper, runID, total, err)
		}

		lastMsg, lastFinish = msg, finish

		switch finish {
		case ai.FinishReasonToolCalls, ai.FinishReasonPauseTurn:
			// AG-16 (R-CST-005): the driver decided to run another
			// logical turn — the run has not concluded, so the running
			// total is a running Estimate, not yet Final.
			if sessionEvent, serr := total.sessionEvent(runID, CostLabelEstimate); serr == nil {
				sendStamped(sink, stamper, sessionEvent)
			}
			continue
		default:
			took, queued := h.queue.takeOrClose()
			if took {
				for _, m := range queued {
					if err := hist.Append(m); err != nil {
						return h.failRun(sink, stamper, runID, total, err)
					}
				}
				// AG-16 (R-CST-005): a steered message arrived at the
				// terminal decision, so the driver runs another logical
				// turn — same Estimate rule as the ToolCalls/PauseTurn
				// arm above.
				if sessionEvent, serr := total.sessionEvent(runID, CostLabelEstimate); serr == nil {
					sendStamped(sink, stamper, sessionEvent)
				}
				continue
			}
		}
		break
	}

	// AG-16 (R-CST-005, R-CST-006): the run-terminal cost_session,
	// immediately before the success run-close, correcting every
	// earlier estimate. Best-effort and independent of the run-close's
	// own construction.
	if sessionEvent, serr := total.sessionEvent(runID, CostLabelFinal); serr == nil {
		sendStamped(sink, stamper, sessionEvent)
	}
	runEnd, rerr := NewRunEnd(runID, RunOutcomeCompleted, nil)
	if rerr == nil {
		sendStamped(sink, stamper, runEnd)
	}
	return lastMsg, lastFinish, nil
}
