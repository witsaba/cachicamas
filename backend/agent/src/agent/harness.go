// AG-13.1/AG-13.2/AG-13.3 — the multi-turn run driver (R-RUN-001..012).
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

	// queue is the steering FIFO (R-RUN-008). Zero-value ready — a
	// Harness{} constructs one implicitly.
	queue steeringQueue
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

// Run drives one run to its terminal decision: repeated Turn calls
// sharing one run identity and one contiguous lane, until a terminal
// finish reason with an empty steering queue (R-RUN-002, R-RUN-003). See
// design.md's run algorithm for the full description.
//
// Phase 2 increment: single-turn happy path only — the loop iterates
// exactly once regardless of finish reason. Iteration and the failure
// path land in later increments of this same file.
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

	defer close(sink)

	runID := mintHarnessRunID()
	stamper := &LaneStamper{}

	runStart, err := NewRunStart(runID)
	if err != nil {
		return ai.Message{}, 0, err
	}
	sendStamped(sink, stamper, runStart)

	if err := hist.Append(prompt); err != nil {
		return ai.Message{}, 0, err
	}

	transcript := transcriptFromHistory(hist)

	turnSink := make(chan *Event)
	forwarderDone := make(chan struct{})
	go func() {
		defer close(forwarderDone)
		for ev := range turnSink {
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

	msg, finish, _ := Turn(ctx, h.Provider, h.System, transcript, turnOpts, turnSink)
	<-forwarderDone

	_ = hist.CloseTurn()

	// Phase 2 increment: the terminal decision is always taken after
	// exactly one turn — iteration lands in a later increment. The
	// atomic take-or-close step still runs so a post-terminal Steer is
	// rejected typed rather than silently accepted (R-RUN-001).
	h.queue.takeOrClose()

	runEnd, rerr := NewRunEnd(runID, RunOutcomeCompleted, nil)
	if rerr == nil {
		sendStamped(sink, stamper, runEnd)
	}
	return msg, finish, nil
}
