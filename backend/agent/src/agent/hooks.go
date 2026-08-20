// Package agent is Layer 2 of the cachicamas agent stack. This file
// (hooks.go) hosts AG-20 — the complete hook taxonomy (R-HKS-001..012):
// the Hooks registration surface, the two function-type families, the
// three payload types, the typed observer-stall report, the per-run
// observer lane and its terminal-boundary snapshot.
//
// AG-20 discharges AG-08's own ":19" chain-composition promise
// (agent-pre-request-hook/spec.md) and lands the three remaining hook
// points doc 0003:1864-1918 named: pre-compact, post-turn,
// session-start. Harness.Hooks is the ONE registration surface
// (R-HKS-001); Harness gains NO exported method
// (scope_fence_test.go:102-105, harness_test.go:1031, both unedited by
// this change) — the surface is a field, checked by the unexported
// isZero predicate below, because Hooks holds function-typed members
// and is therefore not comparable.
//
// The two families are distinguished BY TYPE (R-HKS-001, design AD-2):
// mutating hooks (PreRequest, PreCompact) take a payload and return one
// with an error; observing hooks (PostTurn, SessionStart) have NO
// result parameters at all, so a function that could signal a mutation
// or a failure back to the runtime is unconstructible — the wrong
// thing is unreachable, not merely discouraged (the AG-18/AG-19 house
// rule).
package agent

import (
	"context"
	"sync"

	"github.com/cachicamas/backend/agent/src/ai"
)

// ---------------------------------------------------------------------
// The two function-type families (R-HKS-001, design AD-2)
// ---------------------------------------------------------------------

// PreRequestHook is a mutating pre-request chain element (R-HKS-002).
// It receives the loop's own ctx and the in-flight ai.Request, and
// returns a derived request or an error. TurnOptions.PreRequestHook
// (AG-08's shipped singular field, kept unamended and running FIRST)
// feeds Hooks.PreRequest[0]; element i's returned request is the input
// to element i+1.
type PreRequestHook func(ctx context.Context, req ai.Request) (ai.Request, error)

// PreCompactHook is a mutating pre-compact chain element (R-HKS-003).
// It receives the loop's own ctx and the resolved CompactionPlan, and
// returns a derived plan or an error. The returned plan's cut is fed
// back through the module's existing cut-resolution surgery
// UNCONDITIONALLY before use (R-CMP-004 as amended) — no new
// validation code is written for this hook's output.
type PreCompactHook func(ctx context.Context, plan CompactionPlan) (CompactionPlan, error)

// PostTurnObserver is an observing hook fired once per logical turn
// that ran (R-HKS-004). It has NO result parameters: an observer
// cannot signal a mutation or a failure back to the runtime — the
// wrong thing is unconstructible (R-HKS-001).
type PostTurnObserver func(ctx context.Context, report PostTurnReport)

// SessionStartObserver is an observing hook fired once per Harness
// VALUE (R-HKS-005), before that value's first turn. No result
// parameters, for the identical reason PostTurnObserver has none.
type SessionStartObserver func(ctx context.Context, report SessionStartReport)

// ObserverStallReporter receives one typed report per stalled or
// panicked observing-hook invocation, delivered off the streaming path
// (R-HKS-008). A nil reporter reports nothing — the module's standard
// nil-default posture (Failover, ContextStrategy, PermissionPolicy).
type ObserverStallReporter func(ObserverStall)

// ---------------------------------------------------------------------
// Hooks — the ONE registration surface (R-HKS-001)
// ---------------------------------------------------------------------

// Hooks is the whole exported hook-registration surface: the four hook
// families plus one nil-defaultable stall reporter. Harness.Hooks is
// the ONE registration surface; TurnOptions.Hooks is a TRANSPORT the
// harness fills on its per-turn copy, beside Continuation — never a
// second registration surface (R-HKS-001). Because Hooks holds
// function-typed fields it is NOT COMPARABLE — h.Turn.Hooks !=
// (Hooks{}) does not compile — so isZero below is the emptiness
// predicate every caller (including this file) MUST use.
//
// A direct Turn caller who sets an observing family on
// TurnOptions.Hooks gets INERTNESS, not refusal (R-HKS-001): Turn
// consumes only PreRequest — it owns no observer lane and no run
// identity, so PostTurn, SessionStart and StallReporter set on a
// directly-constructed TurnOptions fire nothing, report nothing, and
// change nothing observable.
type Hooks struct {
	// PreRequest is the pre-request chain, in registration order
	// (R-HKS-002, R-HKS-006). Composed AFTER TurnOptions.PreRequestHook.
	PreRequest []PreRequestHook

	// PreCompact is the pre-compact chain, in registration order
	// (R-HKS-003, R-HKS-006), spliced inside the compaction operation,
	// reaching both doors (R-CMP-001) through the one splice.
	PreCompact []PreCompactHook

	// PostTurn is the set of post-turn observers, dispatched in
	// registration order on the observer lane (R-HKS-004, R-HKS-006,
	// R-HKS-007).
	PostTurn []PostTurnObserver

	// SessionStart is the set of session-start observers, dispatched
	// in registration order on the observer lane, once per Harness
	// value (R-HKS-005, R-HKS-006, R-HKS-007).
	SessionStart []SessionStartObserver

	// StallReporter receives one typed report per stalled or panicked
	// observing-hook invocation (R-HKS-008). Nil reports nothing.
	StallReporter ObserverStallReporter
}

// isZero reports whether h carries no hook of any family and no
// reporter — the explicit predicate R-HKS-001/R-HKS-012 require,
// because Hooks holds function fields and is therefore not comparable
// (an equality against a zero value does not compile).
func (h Hooks) isZero() bool {
	return len(h.PreRequest) == 0 &&
		len(h.PreCompact) == 0 &&
		len(h.PostTurn) == 0 &&
		len(h.SessionStart) == 0 &&
		h.StallReporter == nil
}

// ---------------------------------------------------------------------
// Payload types — value types with unexported fields and read-only
// accessors (R-HKS-001): an observer cannot reach shared state through
// its argument.
// ---------------------------------------------------------------------

// CompactionPlan is what a PreCompactHook receives and returns
// (R-HKS-003): the resolved cut and the derived span. The span is
// derived and never settable — a hook can re-designate the cut but
// cannot forge a span (R-HKS-001).
type CompactionPlan struct {
	cut  int
	span CompactionSpan
}

// Cut returns the plan's cut — an index into the transcript the
// compaction operation resolved (R-HKS-003).
func (p CompactionPlan) Cut() int { return p.cut }

// Span returns the plan's derived compaction span.
func (p CompactionPlan) Span() CompactionSpan { return p.span }

// WithCut returns a CompactionPlan carrying cut in place of p's own —
// the ai.Request.With derivation posture (R-HKS-001). A pre-compact
// hook MAY re-designate the boundary, forward or backward: the
// module's existing retraction-only resolution surgery re-resolves the
// returned plan's cut UNCONDITIONALLY before use (R-CMP-004 as
// amended), so a forward-adjusting hook issues a new REQUEST rather
// than bypassing RESOLUTION.
func (p CompactionPlan) WithCut(cut int) CompactionPlan {
	p.cut = cut
	return p
}

// PostTurnReport is what a PostTurnObserver receives once per logical
// turn that ran (R-HKS-004). Cost is the sum over that logical turn's
// attempts; Attempts makes the retry semantics of Cost legible.
type PostTurnReport struct {
	run      RunID
	turn     TurnID
	outcome  TurnOutcome
	cost     CostFigures
	attempts int
}

// Run returns the report's run identity.
func (r PostTurnReport) Run() RunID { return r.run }

// Turn returns the report's turn identity.
func (r PostTurnReport) Turn() TurnID { return r.turn }

// Outcome returns the turn's outcome, read from the forwarded
// turn-close payload — the same vocabulary the stream carries, no
// member added (R-ATT-010).
func (r PostTurnReport) Outcome() TurnOutcome { return r.outcome }

// Cost returns the turn's cost — the sum over the logical turn's
// attempts (R-HKS-004); aborted attempts contribute none.
func (r PostTurnReport) Cost() CostFigures { return r.cost }

// Attempts returns the logical turn's attempt count.
func (r PostTurnReport) Attempts() int { return r.attempts }

// SessionStartReport is what a SessionStartObserver receives, once per
// Harness value (R-HKS-005).
type SessionStartReport struct {
	run RunID
}

// Run returns the firing run's identity.
func (r SessionStartReport) Run() RunID { return r.run }

// HookPoint discriminates ObserverStall.Point() (R-HKS-008).
type HookPoint uint8

const (
	// HookPointSessionStart names a stalled or panicked SessionStart
	// invocation.
	HookPointSessionStart HookPoint = iota + 1
	// HookPointPostTurn names a stalled or panicked PostTurn
	// invocation.
	HookPointPostTurn
)

// StallReason discriminates ObserverStall.Reason() (R-HKS-008, design
// AD-4): three values, refining the proposal's two — a queued victim
// is not a stalled culprit, and collapsing them misattributes.
type StallReason uint8

const (
	// StallOutstanding reports an invocation dispatched, not returned
	// at the terminal boundary. Delivered at the snapshot,
	// synchronously on Run's own goroutine.
	StallOutstanding StallReason = iota + 1
	// StallQueued reports an invocation never dispatched, still queued
	// behind a stall. Delivered at the snapshot, on Run's own
	// goroutine.
	StallQueued
	// StallPanicked reports an observer that panicked and was
	// recovered. Delivered at recovery, on the lane's own goroutine.
	StallPanicked
)

// ObserverStall is the typed report a stalled or panicked observing
// hook produces (R-HKS-008). Delivered to a nil-defaultable reporter,
// off the streaming path — never a stream event: an event announcing
// the stall would itself be a path from the stalled observer back onto
// the producer's stream, which is exactly what R-HKS-007/R-AGE-008
// forbid.
type ObserverStall struct {
	point  HookPoint
	index  int
	run    RunID
	reason StallReason
}

// Point returns which hook point stalled or panicked.
func (s ObserverStall) Point() HookPoint { return s.point }

// Index returns the stalled or panicked observer's own registration
// index at its point.
func (s ObserverStall) Index() int { return s.index }

// Run returns the run identity the stalled or panicked invocation
// belongs to.
func (s ObserverStall) Run() RunID { return s.run }

// Reason returns the three-valued discriminator (R-HKS-008).
func (s ObserverStall) Reason() StallReason { return s.reason }

// ---------------------------------------------------------------------
// The observer lane (R-HKS-007, R-HKS-008, design AD-3, AD-4, AD-5)
// ---------------------------------------------------------------------

// observerInvocation is one queued lane entry: a session-start or a
// post-turn dispatch, opaque to the lane's own drain loop, which only
// needs to invoke it and know how to attribute it.
type observerInvocation struct {
	point  HookPoint
	index  int
	run    RunID
	invoke func()
}

// observerLane is the per-run observer lane (R-HKS-007): an unbounded
// FIFO queue plus one drain goroutine, created only when at least one
// observing hook is registered (R-HKS-012 inertness — the nil path
// creates no lane, starts no goroutine, allocates no queue). Enqueue
// is a lock-append that never blocks — that, and not scheduling luck,
// is the non-blocking property (R-HKS-007). Dispatch is registration
// order = enqueue order (R-HKS-006), on the lane's own goroutine —
// never the run's, never any event-delivery goroutine.
//
// Every method has a nil-receiver no-op form, so call sites never
// branch on whether a lane exists — the inert path (nil lane) and the
// active path share one call shape.
type observerLane struct {
	reporter ObserverStallReporter

	mu       sync.Mutex
	cond     *sync.Cond
	queue    []observerInvocation
	inFlight *observerInvocation
	closed   bool

	reportMu sync.Mutex
}

// newObserverLane constructs and starts a lane iff hooks carries at
// least one observing hook (R-HKS-012). The returned value is nil on
// the inert path.
func newObserverLane(hooks Hooks) *observerLane {
	if len(hooks.PostTurn) == 0 && len(hooks.SessionStart) == 0 {
		return nil
	}
	l := &observerLane{reporter: hooks.StallReporter}
	l.cond = sync.NewCond(&l.mu)
	go l.drain()
	return l
}

// enqueue appends inv to the queue under lock and returns immediately
// — a lock-append, never blocking on the observer's own progress
// (R-HKS-007).
func (l *observerLane) enqueue(inv observerInvocation) {
	if l == nil {
		return
	}
	l.mu.Lock()
	l.queue = append(l.queue, inv)
	l.mu.Unlock()
	l.cond.Signal()
}

// enqueueSessionStart enqueues one invocation per registered
// SessionStart observer, in registration order (R-HKS-005, R-HKS-006).
// Session-start is the lane's FIRST invocation for the run that fires
// it, ordered before any post-turn invocation of that run — true by
// construction: Run calls this immediately after lane creation,
// strictly before any post-turn enqueue site is ever reached.
func (l *observerLane) enqueueSessionStart(observers []SessionStartObserver, run RunID) {
	if l == nil {
		return
	}
	report := SessionStartReport{run: run}
	for i, obs := range observers {
		obs := obs
		l.enqueue(observerInvocation{
			point: HookPointSessionStart,
			index: i,
			run:   run,
			invoke: func() {
				obs(context.Background(), report)
			},
		})
	}
}

// enqueuePostTurn enqueues one invocation per registered PostTurn
// observer, in registration order (R-HKS-004, R-HKS-006).
func (l *observerLane) enqueuePostTurn(observers []PostTurnObserver, report PostTurnReport) {
	if l == nil {
		return
	}
	for i, obs := range observers {
		obs := obs
		l.enqueue(observerInvocation{
			point: HookPointPostTurn,
			index: i,
			run:   report.run,
			invoke: func() {
				obs(context.Background(), report)
			},
		})
	}
}

// drain is the lane's own goroutine (R-HKS-007): pops the queue FIFO,
// records the in-flight invocation under the mutex (R-HKS-008's
// snapshot needs it), invokes it with a fresh, value-stripped
// context.Background()-rooted ctx — never runCtx, and deliberately not
// a cancellation-stripped derivation of it, because that would
// preserve context VALUES and hand an observing hook the delegation
// seam's one sanctioned door back onto a parent's event lane
// (R-HKS-007, design AD-3) — then clears in-flight and continues with
// the next queued invocation. Exits once the queue is empty AND the
// lane has been closed for enqueue (reportOutstanding's own snapshot,
// reached strictly after every enqueue site in Run has already run) —
// so a released observer leaves no steady-state goroutine (R-HKS-012).
// A permanently stalled observer (this goroutine blocked inside
// invokeRecovered) never reaches that check again: the goroutine leaks
// by design, the caller's leak (R-HKS-008 consequence 1).
func (l *observerLane) drain() {
	for {
		l.mu.Lock()
		for len(l.queue) == 0 && !l.closed {
			l.cond.Wait()
		}
		if len(l.queue) == 0 {
			l.mu.Unlock()
			return
		}
		inv := l.queue[0]
		l.queue = l.queue[1:]
		l.inFlight = &inv
		l.mu.Unlock()

		l.invokeRecovered(inv)

		l.mu.Lock()
		l.inFlight = nil
		l.mu.Unlock()
	}
}

// invokeRecovered runs inv.invoke, recovering a panic (R-HKS-008: an
// unrecovered panic on the lane would kill the process — the worst
// possible way for "observers never block the streaming path" to be
// true). A recovered panic is reported StallPanicked, on this SAME
// lane goroutine (design AD-4) — the drain goroutine is free at
// recovery, since the panicking hook returned by panicking — and the
// lane continues with the next queued invocation.
func (l *observerLane) invokeRecovered(inv observerInvocation) {
	defer func() {
		if r := recover(); r != nil {
			l.report(ObserverStall{point: inv.point, index: inv.index, run: inv.run, reason: StallPanicked})
		}
	}()
	inv.invoke()
}

// reportOutstanding runs the terminal-boundary snapshot (R-HKS-008,
// design AD-4): synchronously, on the CALLER's goroutine (Run's own,
// via a defer registered immediately after lane creation — LIFO places
// it FIRST among Run's defers, after every returning arm's own run_end
// has already been sent and before the queue close / cancel-clear /
// close(sink)). Under the lane mutex it reads the in-flight invocation
// (if any) and every still-queued invocation, and marks the lane
// closed so the drain goroutine can exit once it is next free to check
// — this call NEVER waits for the in-flight invocation to finish;
// "outstanding" is exactly the observation that it has not.
//
// This unconditional lock/unlock is also what makes the
// snapshot-or-count determinism lemma true for every test in this
// milestone's suite: an enqueued invocation's own side effect
// (recorded before the drain goroutine clears in-flight under this
// same mutex) happens-before this method's own lock acquisition,
// which happens-before Run's return — so a test reading its own
// recorder after Run returns needs no sleep and no poll to know
// whether a given hook already ran.
func (l *observerLane) reportOutstanding() {
	if l == nil {
		return
	}
	l.mu.Lock()
	var outstanding *observerInvocation
	if l.inFlight != nil {
		inv := *l.inFlight
		outstanding = &inv
	}
	queued := make([]observerInvocation, len(l.queue))
	copy(queued, l.queue)
	l.closed = true
	l.mu.Unlock()
	l.cond.Broadcast()

	if outstanding != nil {
		l.report(ObserverStall{point: outstanding.point, index: outstanding.index, run: outstanding.run, reason: StallOutstanding})
	}
	for _, inv := range queued {
		l.report(ObserverStall{point: inv.point, index: inv.index, run: inv.run, reason: StallQueued})
	}
}

// report invokes the reporter, serialized by one dedicated exclusion
// (R-HKS-008: all reporter invocations MUST be serialized by one
// dedicated exclusion — shared by both invocation sites, the lane
// goroutine's panic path and Run's own snapshot path) and
// recover-wrapped: the reporter is the last resort, it has no
// meta-reporter, and a reporter's own panic is discarded so process
// survival wins. A nil reporter reports nothing.
func (l *observerLane) report(stall ObserverStall) {
	if l.reporter == nil {
		return
	}
	l.reportMu.Lock()
	defer l.reportMu.Unlock()
	func() {
		defer func() { _ = recover() }()
		l.reporter(stall)
	}()
}
