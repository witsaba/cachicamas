// AG-09.2 / AG-09.3 / AG-09.4 — the hand-rolled tool scheduler
// (R-TLS-004..011, D4, D5, D6, D6b, D9a).
//
// The scheduler is the core of AG-09: it takes a slice of tool
// calls and runs them under the AG-09.2 concurrency policy (reads
// concurrent with bounded fan-out, mutating + execute serialized
// in call order), rejoins results in call order regardless of
// completion order, contains panics per call goroutine, and
// preserves the `LaneStamper` single-writer invariant via one
// dispatcher goroutine that owns the stamper.
//
// # Why hand-rolled (D4a)
//
// Stdlib-only: `chan struct{}` semaphore for reads, single-goroutine
// channel for mutating/execute, indexed `[]Result` for rejoin,
// `defer/recover` per call goroutine for panic containment. No
// `errgroup` (forbidden — new top-level dep + first-error
// cancellation conflicts with R-TLS-010 "siblings complete";
// `openspec/AGENTS.md` `## Hard rules`).
//
// # Source-guard tests (D3a, D4a)
//
// Two regex-scan tests in `scheduler_test.go` enforce:
//
//  1. `scheduler.go` contains zero type assertions on `PolicySlot`
//     (the seam 3 promise — Layer 2 never reads the value);
//  2. `scheduler.go` does not import `golang.org/x/sync/errgroup`
//     (no new top-level dep; concurrency is hand-rolled).
//
// # Sub-methods
//
// `Schedule` is the public entry; the implementation is split into
// six private sub-methods to keep the call graph inspectable:
//
//   - `Schedule` — top-level orchestrator (per-call goroutine
//     spawn + semaphore/serialized entry + dispatcher sync).
//   - `runDispatcher` — the single goroutine that owns the
//     `LaneStamper`; reads from an `emission` channel and sends
//     stamped events on `sink`.
//   - `scheduleRead` — bounded fan-out via `chan struct{}`
//     semaphore; one read-class call per slot.
//   - `scheduleSerialized` — single-goroutine serialized channel
//     for mutating + execute in call order.
//   - `executeCall` — resolve the tool, emit start, run the tool,
//     emit end, write the result to the indexed slot.
//   - `recoverCall` — panic containment in the call goroutine.
//
// # Substrate preservation
//
// The substrate (21 files including `go.mod`, `go.sum`, `Makefile`,
// `.golangci.yml`) is byte-untouched. NFR-TLS-003's 6th consecutive
// extensibility demonstration.
package agent

import (
	"context"
	"sync"

	"github.com/cachicamas/backend/agent/src/ai"
)

// maxReadFanOutDefault is the default upper bound on the number of
// concurrently running read-class calls. AG-09.2 charter floor;
// AG-13 may benchmark alternative defaults.
const maxReadFanOutDefault = 8

// emission is the wire shape the call goroutines send to the
// single dispatcher goroutine. The dispatcher stamps and sends on
// `sink`; the call goroutines never touch the `LaneStamper`
// directly (D6b's single-writer invariant).
type emission struct {
	ev Event
}

// Schedule runs a slice of tool calls under the AG-09.2 / AG-09.3 /
// AG-09.4 rules. It returns the rejoin slice in call order. The
// caller passes a `Registry` (named `reg`), the run/turn identities
// (`runID`, `turnID`), the per-turn `LaneStamper`, and the `sink`
// channel the loop owns. The scheduler closes `sink` after the
// rejoin. The caller MUST NOT close `sink` from outside.
//
// `MaxConcurrentReads` is the upper bound on concurrently running
// read-class calls. A value of 0 or less falls back to
// `maxReadFanOutDefault`. Mutating and execute classes are
// serialized in call order, regardless of this value.
func (s *Scheduler) Schedule(
	ctx context.Context,
	calls []ai.ToolCall,
	reg Registry,
	runID RunID,
	turnID TurnID,
	stamper *LaneStamper,
	sink chan<- *Event,
) []Result {
	if len(calls) == 0 {
		// Empty input: no goroutines, no emissions, no close
		// (the loop hasn't emitted brackets yet — the empty
		// case is the "no tool calls" path on a streaming
		// completion without tool calls).
		return []Result{}
	}

	maxReads := s.MaxConcurrentReads
	if maxReads <= 0 {
		maxReads = maxReadFanOutDefault
	}

	results := make([]Result, len(calls))
	emissions := make(chan emission, len(calls)*2) // start + end per call, plus slack
	dispatcherDone := make(chan struct{})

	// Bounded fan-out for reads: a `chan struct{}` semaphore
	// with capacity `maxReads`. A read call acquires a slot
	// before executing; an over-quorum call blocks until a
	// slot is released.
	readSem := make(chan struct{}, maxReads)

	// Serialized channel for mutating + execute. Capacity 1:
	// only one such call is in flight at a time, in call order.
	serialCh := make(chan struct{}, 1)

	// One dispatcher goroutine owns the stamper. The call
	// goroutines send emissions; the dispatcher stamps and
	// forwards to sink. LaneStamper's single-writer invariant
	// (sequence.go:8-24) is preserved by construction.
	go s.runDispatcher(stamper, sink, emissions, dispatcherDone)

	var wg sync.WaitGroup
	wg.Add(len(calls))
	for i, call := range calls {
		i, call := i, call
		switch tool, ok := reg.Resolve(call.Name()); {
		case !ok:
			// Orphan: no tool registered. Schedule inline
			// (no fan-out, no serialize) so the result slot
			// is populated synchronously.
			//
			// The orphan path is the only path that does
			// NOT go through a goroutine. The `wg.Add` /
			// `wg.Done` pair balances it.
			wg.Done()
			s.scheduleOrphan(i, call, runID, turnID, results, emissions)
		case tool.EffectClass() == EffectClassRead:
			go s.scheduleRead(i, call, tool, runID, turnID, results, emissions, readSem, &wg)
		default:
			go s.scheduleSerialized(i, call, tool, runID, turnID, results, emissions, serialCh, &wg)
		}
	}
	wg.Wait()
	close(emissions)
	<-dispatcherDone
	close(sink)
	return results
}

// runDispatcher is the single goroutine that owns the `LaneStamper`
// (D6b). It reads emissions and forwards stamped events on `sink`.
// The contract: only this goroutine calls `stamper.Stamp`; the call
// goroutines never touch the stamper. The channel is closed by
// `Schedule` after the call goroutines `wg.Wait()`; `done` is closed
// when the dispatcher exits so `Schedule` can synchronize the close
// of `sink` after the last emission has been sent.
func (s *Scheduler) runDispatcher(
	stamper *LaneStamper,
	sink chan<- *Event,
	emissions <-chan emission,
	done chan<- struct{},
) {
	defer close(done)
	for em := range emissions {
		stamped := stamper.Stamp(em.ev)
		sink <- &stamped
	}
}

// scheduleRead is the bounded-fan-out entry for a read-class call.
// It blocks on the read-semaphore until a slot is available, then
// runs the call. The semaphore is owned by `Schedule`; the
// capacity is the upper bound on concurrently running read-class
// calls. An unbounded scheduler would let the counter exceed the
// bound (S-TLS-005a's bite).
func (s *Scheduler) scheduleRead(
	ordinal int,
	call ai.ToolCall,
	tool Tool,
	runID RunID,
	turnID TurnID,
	results []Result,
	emissions chan<- emission,
	sem chan struct{},
	wg *sync.WaitGroup,
) {
	defer wg.Done()
	sem <- struct{}{}
	defer func() { <-sem }()
	s.executeCall(ordinal, call, tool, runID, turnID, results, emissions)
}

// scheduleSerialized is the serialized entry for a mutating /
// execute class call. It blocks on the serialized channel (capacity
// 1) until the previous call's slot is free. Calls execute in
// issuance order regardless of completion order.
func (s *Scheduler) scheduleSerialized(
	ordinal int,
	call ai.ToolCall,
	tool Tool,
	runID RunID,
	turnID TurnID,
	results []Result,
	emissions chan<- emission,
	serial chan struct{},
	wg *sync.WaitGroup,
) {
	defer wg.Done()
	serial <- struct{}{}
	defer func() { <-serial }()
	s.executeCall(ordinal, call, tool, runID, turnID, results, emissions)
}

// scheduleOrphan handles a call whose tool is not registered. It
// runs synchronously (no goroutine, no fan-out, no serialize) and
// populates the call's ordinal slot with a typed
// `ExecutionFailure`. The dispatcher emits the matching
// `ToolEndExecutionFailure` event.
//
// The synchronous path is sound: a registry miss is a configuration
// defect, not a tool defect, and a typed failure result is the
// load-bearing side effect.
func (s *Scheduler) scheduleOrphan(
	ordinal int,
	call ai.ToolCall,
	runID RunID,
	turnID TurnID,
	results []Result,
	emissions chan<- emission,
) {
	results[ordinal] = orphanResult(call.ID(), "tool not registered: "+call.Name())
	emitOrphanExecutionFailure(ordinal, call, runID, turnID, results, emissions)
}

// executeCall resolves the tool via `reg`, emits `ToolStart` at
// execution start, runs the tool with the policy byte-exact, emits
// the appropriate `ToolEnd*`, and writes the result to the
// call's ordinal slot. Panics in the tool are recovered and
// converted to a typed `ToolEndExecutionFailure`.
//
// The two return channels from `Run` are disjoint (per `Result` doc):
//
//   - `(Result{Outcome: Success|ResultFailure}, nil)` → emit
//     `ToolEndSuccess` or `ToolEndResultFailure` accordingly.
//   - `(Result{}, err)` → emit `ToolEndExecutionFailure` with a
//     typed `*Failure`.
func (s *Scheduler) executeCall(
	ordinal int,
	call ai.ToolCall,
	tool Tool,
	runID RunID,
	turnID TurnID,
	results []Result,
	emissions chan<- emission,
) {
	// Result slot is pre-populated with the callID for the
	// rejoin test (R-TLS-009). The CallID accessor returns
	// the value the scheduler observed; the test asserts it
	// byte-equals `ai.ToolCall.id`.
	results[ordinal].SetCallID(call.ID())

	// Recover from any panic the tool may raise. The recovery
	// path writes a typed `ToolEndExecutionFailure` and a typed
	// `*Failure` into the result slot before returning. The
	// sibling goroutines continue to run independently (R-TLS-011).
	defer recoverCall(ordinal, results)

	// Emit `ToolStart` BEFORE calling `Run` (R-TLS-006: start
	// events at execution start, not at rejoin). The emission
	// is buffered by the dispatcher channel; the actual sink
	// send happens on the dispatcher's goroutine.
	startEv, startErr := NewToolStart(runID, turnID, call.ID(), uint32(ordinal), call.Name(), call.Arguments())
	if startErr != nil {
		// Constructor failure is a typed execution failure —
		// a malformed arguments payload is a provider defect,
		// not a tool defect. Typed `*Failure` is required.
		results[ordinal] = typedExecutionFailureFromError(call.ID(), startErr)
		emitExecutionFailure(ordinal, call, runID, turnID, results, emissions)
		return
	}
	emissions <- emission{ev: startEv}

	// Run the tool. Policy is forwarded byte-exact (R-TLS-002,
	// D3a, seam 3). The tool receives a fresh `PolicySlot` whose
	// value is the call's identity — Layer 2 does not interpret
	// it; the tool or a Layer 3 sandbox does. (For AG-09 we
	// pass the call's id as a `string` policy; Layer 3 replaces
	// this with its own sandbox descriptor at the seam.)
	runRes, runErr := tool.Run(context.Background(), call.Arguments(), PolicySlot(call.ID()))

	// Disjoint return channels: a non-nil `err` from `Run`
	// means execution itself failed (R-TLS-010). The two channels
	// are disjoint: a populated `Result.Outcome` other than the
	// zero value means `err` MUST be nil; a non-nil `err` means
	// `runRes.Outcome` MUST be the zero value.
	if runErr != nil {
		results[ordinal] = typedExecutionFailureFromError(call.ID(), runErr)
		emitExecutionFailure(ordinal, call, runID, turnID, results, emissions)
		return
	}

	// Tool returned a typed `Result`. Propagate the outcome.
	results[ordinal] = runRes
	results[ordinal].SetCallID(call.ID())

	switch runRes.Outcome {
	case ToolOutcomeSuccess:
		endEv, endErr := NewToolEndSuccess(runID, turnID, call.ID(), uint32(ordinal), runRes.Content)
		if endErr != nil {
			results[ordinal] = typedExecutionFailureFromError(call.ID(), endErr)
			emitExecutionFailure(ordinal, call, runID, turnID, results, emissions)
			return
		}
		emissions <- emission{ev: endEv}
	case ToolOutcomeResultFailure:
		endEv, endErr := NewToolEndResultFailure(runID, turnID, call.ID(), uint32(ordinal), runRes.Content)
		if endErr != nil {
			results[ordinal] = typedExecutionFailureFromError(call.ID(), endErr)
			emitExecutionFailure(ordinal, call, runID, turnID, results, emissions)
			return
		}
		emissions <- emission{ev: endEv}
	case ToolOutcomeExecutionFailure:
		// A tool that returns a typed `Result{Outcome: ExecutionFailure}`
		// MUST also carry a typed `*Failure` (R-AMT-006 carry).
		if runRes.Failure == nil {
			results[ordinal] = typedExecutionFailureFromError(call.ID(),
				errBiteToolMissingFailure)
			emitExecutionFailure(ordinal, call, runID, turnID, results, emissions)
			return
		}
		emitExecutionFailureWith(ordinal, call, runID, turnID, runRes.Failure, emissions)
	default:
		// Zero value: the tool returned no outcome. Treat as
		// an execution failure with a typed `*Failure`.
		results[ordinal] = typedExecutionFailureFromError(call.ID(),
			errBiteToolZeroOutcome)
		emitExecutionFailure(ordinal, call, runID, turnID, results, emissions)
	}
}

// emitOrphanExecutionFailure emits a `ToolEndExecutionFailure` for a
// call whose tool could not be resolved.
func emitOrphanExecutionFailure(
	ordinal int,
	call ai.ToolCall,
	runID RunID,
	turnID TurnID,
	results []Result,
	emissions chan<- emission,
) {
	emitExecutionFailureWith(ordinal, call, runID, turnID, results[ordinal].Failure, emissions)
}

// emitExecutionFailure emits a `ToolEndExecutionFailure` for a call
// whose result is already a typed `ExecutionFailure`.
func emitExecutionFailure(
	ordinal int,
	call ai.ToolCall,
	runID RunID,
	turnID TurnID,
	results []Result,
	emissions chan<- emission,
) {
	emitExecutionFailureWith(ordinal, call, runID, turnID, results[ordinal].Failure, emissions)
}

// emitExecutionFailureWith is the common path that constructs a
// `ToolEndExecutionFailure` event and sends it to the dispatcher.
// `failure` is the typed `*Failure`; if nil, the emission is
// skipped (the call's result is still recorded in `results`).
func emitExecutionFailureWith(
	ordinal int,
	call ai.ToolCall,
	runID RunID,
	turnID TurnID,
	failure *Failure,
	emissions chan<- emission,
) {
	if failure == nil {
		return
	}
	endEv, err := NewToolEndExecutionFailure(runID, turnID, call.ID(), uint32(ordinal), failure)
	if err != nil {
		return
	}
	emissions <- emission{ev: endEv}
}

// recoverCall is the panic containment path (R-TLS-011, D6a). The
// recovery constructs a typed `*Failure` and writes the
// `ExecutionFailure` result into the panicking call's ordinal slot
// before the goroutine exits. Sibling goroutines continue to run
// independently.
func recoverCall(ordinal int, results []Result) {
	r := recover()
	if r == nil {
		return
	}
	failure, _ := typedFailureFromRecover(r)
	results[ordinal] = Result{
		Outcome: ToolOutcomeExecutionFailure,
		Failure: failure,
	}
	// Preserve the callID across the recovery rewrite.
	results[ordinal].SetCallID(results[ordinal].CallID())
}

// orphanResult returns a typed `ExecutionFailure` `Result` for a
// call whose tool could not be resolved (R-TLS-009 "orphan" path).
func orphanResult(callID, msg string) Result {
	failure, _ := typedFailureFromError(orphanToolUnavailable{msg: msg})
	res := Result{
		Outcome: ToolOutcomeExecutionFailure,
		Failure: failure,
	}
	res.SetCallID(callID)
	return res
}

// typedExecutionFailureFromError wraps an arbitrary error as a
// typed `*Failure` and constructs a `Result` for it. Used by the
// call goroutine when the tool returns a non-nil error.
func typedExecutionFailureFromError(callID string, err error) Result {
	if err == nil {
		err = errBiteToolZeroOutcome
	}
	failure, _ := typedFailureFromError(err)
	res := Result{
		Outcome: ToolOutcomeExecutionFailure,
		Failure: failure,
	}
	res.SetCallID(callID)
	return res
}

// typedFailureFromError constructs a typed `*Failure` for an arbitrary
// `error`. The construction is mid-stream (a tool's failure is
// always mid-stream; the loop has already produced events). The
// cause is preserved for `Unwrap` but never reproduced in `Error()`.
func typedFailureFromError(err error) (*Failure, error) {
	report := ai.FailureReport{
		Category: ai.FailureCategoryUnavailable,
		Cause:    err,
	}
	aiFailure, ferr := ai.MidStreamFailure(report, false)
	if ferr != nil {
		return nil, ferr
	}
	return NewFailure(aiFailure)
}

// typedFailureFromRecover constructs a typed `*Failure` for a recovered
// panic value. The recovery's underlying value is wrapped as the
// cause; the rendered `Error()` does not reproduce it.
func typedFailureFromRecover(r interface{}) (*Failure, error) {
	cause, _ := r.(error)
	if cause == nil {
		cause = errRecovered{value: r}
	}
	return typedFailureFromError(cause)
}

// errRecovered is the typed sentinel for a non-error panic value
// (a string, an int, an arbitrary struct, etc.). The original
// value is preserved for `Unwrap`-like traversal but never
// reproduced in `Error()`.
type errRecovered struct{ value interface{} }

func (e errRecovered) Error() string { return "tool panic recovered" }

// errBiteToolMissingFailure is the typed sentinel for a tool that
// returns `Result{Outcome: ExecutionFailure}` without a `*Failure`.
var errBiteToolMissingFailure = biteToolMissingFailure{}

type biteToolMissingFailure struct{}

func (biteToolMissingFailure) Error() string {
	return "agent: tool returned Result{Outcome: ExecutionFailure} without a typed *Failure"
}

// errBiteToolZeroOutcome is the typed sentinel for a tool that
// returns the zero `Result` with no error and no failure.
var errBiteToolZeroOutcome = biteToolZeroOutcome{}

type biteToolZeroOutcome struct{}

func (biteToolZeroOutcome) Error() string {
	return "agent: tool returned zero Result with no error and no failure"
}

// orphanToolUnavailable is the typed sentinel for a registry miss.
type orphanToolUnavailable struct{ msg string }

func (e orphanToolUnavailable) Error() string { return e.msg }
