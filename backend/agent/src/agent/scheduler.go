// Package agent is Layer 2 of the cachicamas agent stack. This
// file (scheduler.go) hosts AG-09.2 / AG-09.3 / AG-09.4 — the
// hand-rolled tool scheduler (R-TLS-004..011, D4, D5, D6, D6b,
// D9a).
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

	// ack is optional (nil for every emission except
	// permission_decision_required, AG-10 R-APP-002/D4). When
	// non-nil, the dispatcher closes it immediately after
	// `sink <- &stamped` completes, so a sender that blocks on
	// `<-ack` is guaranteed the event has actually reached `sink` —
	// not merely that it was enqueued onto this buffered channel.
	ack chan struct{}
}

// Schedule runs a slice of tool calls under the AG-09.2 / AG-09.3 /
// AG-09.4 rules. It returns the rejoin slice in call order. The
// caller passes a `Registry` (named `reg`), the run/turn identities
// (`runID`, `turnID`), the per-turn `LaneStamper`, and the `sink`
// channel the loop owns.
//
// Sink ownership (AG-13, R-TLS-012, `s.LeaveSinkOpen`): with the flag
// at its zero value (false), the scheduler closes `sink` after the
// rejoin, exactly as AG-09 always did — the caller MUST NOT close
// `sink` from outside. With the flag set, the scheduler leaves `sink`
// open after the rejoin and the caller becomes responsible for
// closing it exactly once. Every other step below — the parked-set
// clear, the emissions close, the dispatcher join, the ordered
// rejoin — is unchanged in behavior and in order; only the close is
// conditional.
//
// `MaxConcurrentReads` is the upper bound on concurrently running
// read-class calls. A value of 0 or less falls back to
// `maxReadFanOutDefault`. Mutating and execute classes are
// serialized in call order, regardless of this value.
//
// `policy` is the Layer-2 permission gate (AG-10). A nil policy
// bypasses the gate — every call is treated as AllowOnce, no events
// are emitted, and the call runs as AG-09's pre-AG-10 scheduler
// did. A non-nil policy consults `policy.Resolve` for every call;
// a `Defer` verdict emits `permission_decision_required` and parks
// the call on a per-call `chan struct{}` keyed by `callID` while
// siblings continue. AG-10.1 covers the immediate-allow path
// (sync allow/deny) and the defer path (park + wait for wake or
// context cancel); AG-10.2 widens to four typed outcomes, AG-10.3
// wires the cancellation wind-down, AG-10.4 wires remembered
// resolutions.
func (s *Scheduler) Schedule(
	ctx context.Context,
	calls []ai.ToolCall,
	reg Registry,
	runID RunID,
	turnID TurnID,
	policy PermissionPolicy,
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

	// Per-Schedule parked set for the AG-10 permission gate.
	// Keyed by `callID`; each entry is a `chan struct{}` the call
	// goroutine blocks on while parked, and the upward-path wake
	// closes. The set is shared across all call goroutines in
	// this Schedule call (R-LSK-002 carry: one Schedule call, one
	// parked set).
	//
	// D-A: also published to `s.parked` (guarded by `s.parkedMu`)
	// so `WakeParked`, called from a goroutine outside this
	// Schedule call's own tree, can reach it. Cleared back to nil
	// at Schedule exit (below) so the field's lifetime matches the
	// parked set's — exactly one Schedule call (R-LSK-002 carry).
	parked := newParkedSet()
	s.parkedMu.Lock()
	s.parked = parked
	s.parkedMu.Unlock()

	// Per-Schedule remembered-tool state for the AG-10.4 Remember
	// gate (R-APP-010). Lives one Schedule call, exactly like
	// parked (R-LSK-002 carry) — unlike parked it needs no external
	// handle, so it stays a plain local threaded through the call
	// goroutines.
	remembered := newRememberedSet()

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
			go s.scheduleRead(ctx, i, call, tool, runID, turnID, policy, parked, remembered, results, emissions, readSem, &wg)
		default:
			go s.scheduleSerialized(ctx, i, call, tool, runID, turnID, policy, parked, remembered, results, emissions, serialCh, &wg)
		}
	}
	wg.Wait()
	// D-B: by the time wg.Wait() returns, every call goroutine —
	// including any that were parked — has already exited through
	// one of park release's exactly two production paths:
	// WakeParked (upward-path wake) or ctx.Done() (cancellation,
	// AG-10.3 R-APP-009). A parked goroutine never calls wg.Done()
	// until it is released, so a defensive sweep placed here would
	// run only after every parked call has already gone through one
	// of those two paths — it would be provably unreachable-with-
	// effect. AG-10.3 originally carried such a sweep
	// (parked.closeAll()); it was deleted (D-B) once this ordering
	// was shown to make it dead code, not a safety net.
	//
	// D-A: the parked set's external handle is only valid while
	// calls from this Schedule call may still be in flight — clear
	// it now so a WakeParked racing the tail of this call observes
	// "no active parked set" (ErrStrayDecision) rather than a set
	// whose entries can no longer matter (R-LSK-002 carry).
	s.parkedMu.Lock()
	s.parked = nil
	s.parkedMu.Unlock()
	close(emissions)
	<-dispatcherDone
	// AG-13 (R-TLS-012): the close is the one conditional step: the
	// zero-value default (false) preserves AG-09 behavior exactly;
	// LeaveSinkOpen: true hands sink-close ownership to the caller,
	// who MUST close it exactly once.
	if !s.LeaveSinkOpen {
		close(sink)
	}
	return results
}

// WakeParked is the upward-path hand-off (D-A: "the loop owns the
// upward-path wake (close by callID); the scheduler owns the parked
// set"). It delivers a decision to the call parked under callID: the
// parked goroutine's `select` in `runPermissionGate` unblocks and
// re-enters `policy.Resolve`, processing the fresh verdict exactly as
// it would a first-time synchronous verdict (design data flow "wake:
// re-evaluate verdict").
//
// Returns ErrStrayDecision — and touches no parked call — when callID
// is not currently parked: either no Schedule call has calls in
// flight right now, or callID was never parked (or has already been
// woken / resolved) (R-APP-003, S-APP-003/S-APP-004).
func (s *Scheduler) WakeParked(callID string) error {
	s.parkedMu.Lock()
	parked := s.parked
	s.parkedMu.Unlock()
	if parked == nil || !parked.wake(callID) {
		return ErrStrayDecision
	}
	return nil
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
		if em.ack != nil {
			// R-APP-002/D4: the sender waiting on ack now has proof
			// the event reached sink, not merely that it reached
			// this dispatcher's internal buffer.
			close(em.ack)
		}
	}
}

// scheduleRead is the bounded-fan-out entry for a read-class call.
// It blocks on the read-semaphore until a slot is available, then
// runs the call. The semaphore is owned by `Schedule`; the
// capacity is the upper bound on concurrently running read-class
// calls. An unbounded scheduler would let the counter exceed the
// bound (S-TLS-005a's bite).
//
// `policy`, `parked` and `remembered` thread the AG-10 permission
// gate through the call goroutine; `ctx` is the per-Schedule context
// used to unblock parked calls on cancellation (AG-10.3). A nil
// `policy` bypasses the gate (AG-09 behavior preserved).
func (s *Scheduler) scheduleRead(
	ctx context.Context,
	ordinal int,
	call ai.ToolCall,
	tool Tool,
	runID RunID,
	turnID TurnID,
	policy PermissionPolicy,
	parked *parkedSet,
	remembered *rememberedSet,
	results []Result,
	emissions chan<- emission,
	sem chan struct{},
	wg *sync.WaitGroup,
) {
	defer wg.Done()
	sem <- struct{}{}
	defer func() { <-sem }()
	s.executeCall(ctx, ordinal, call, tool, runID, turnID, policy, parked, remembered, results, emissions)
}

// scheduleSerialized is the serialized entry for a mutating /
// execute class call. It blocks on the serialized channel (capacity
// 1) until the previous call's slot is free. Calls execute in
// issuance order regardless of completion order.
//
// `policy`, `parked` and `remembered` thread the AG-10 permission
// gate; `ctx` is the per-Schedule context (AG-10.3 cancellation).
func (s *Scheduler) scheduleSerialized(
	ctx context.Context,
	ordinal int,
	call ai.ToolCall,
	tool Tool,
	runID RunID,
	turnID TurnID,
	policy PermissionPolicy,
	parked *parkedSet,
	remembered *rememberedSet,
	results []Result,
	emissions chan<- emission,
	serial chan struct{},
	wg *sync.WaitGroup,
) {
	defer wg.Done()
	serial <- struct{}{}
	defer func() { <-serial }()
	s.executeCall(ctx, ordinal, call, tool, runID, turnID, policy, parked, remembered, results, emissions)
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
//
// AG-10 extension: `policy`, `parked`, `remembered`, and `ctx`
// thread the permission gate. A nil `policy` bypasses the gate
// (AG-09 behavior). A non-nil policy consults `Resolve` for every
// call not already suppressed by `remembered` (R-APP-010); sync
// verdicts proceed, `Defer` parks the call on `parked.park(call.ID())`
// and waits for wake OR ctx cancel. `ctx` is the Schedule-level
// cancellation context (AG-10.3).
func (s *Scheduler) executeCall(
	ctx context.Context,
	ordinal int,
	call ai.ToolCall,
	tool Tool,
	runID RunID,
	turnID TurnID,
	policy PermissionPolicy,
	parked *parkedSet,
	remembered *rememberedSet,
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

	// AG-10 permission gate (R-APP-001..010). A nil policy
	// bypasses the gate; a non-nil policy consults Resolve — unless
	// `remembered` already silences this call's tool name — and may
	// park the call until wake or ctx cancel.
	gateProceed, modifiedArgs, _ := s.runPermissionGate(
		ctx, ordinal, call, runID, turnID, policy, parked, remembered, results, emissions,
	)
	if !gateProceed {
		// Gate already wrote the typed failure to the result slot
		// and emitted the matching tool-end event (defensive default
		// / Deny / ModifyInput-constructor-failure / cancellation-
		// abort — every proceed=false path). The call does not reach
		// ToolStart or Run; nothing further to do here.
		return
	}

	// Emit `ToolStart` BEFORE calling `Run` (R-TLS-006: start
	// events at execution start, not at rejoin). The emission
	// is buffered by the dispatcher channel; the actual sink
	// send happens on the dispatcher's goroutine.
	//
	// AG-10.2 modify-input transparency (R-APP-006): when the
	// gate's verdict is `ModifyInput`, the tool runs with the
	// substituted arguments and ToolStart carries the modified
	// bytes. Commit 3 leaves `modifiedArgs` empty for the
	// immediate-allow path; Commit 4 fills it for ModifyInput.
	startEv, startErr := NewToolStart(runID, turnID, call.ID(), uint32(ordinal), call.Name(), call.Arguments())
	if modifiedArgs != nil {
		startEv, startErr = NewToolStart(runID, turnID, call.ID(), uint32(ordinal), call.Name(), modifiedArgs)
	}
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
	runArgs := call.Arguments()
	if modifiedArgs != nil {
		runArgs = modifiedArgs
	}
	runRes, runErr := tool.Run(context.Background(), runArgs, PolicySlot(call.ID()))

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

// runPermissionGate is the per-call AG-10 gate (R-APP-001..010). It
// consults `policy.Resolve` once per call — unless `remembered`
// already silences this call's tool name (R-APP-010) — and
// dispatches the verdict:
//
//   - nil policy        → bypass (AG-09 behavior preserved; the
//                         call proceeds as if the gate were absent).
//   - remembered tool   → bypass without consulting Resolve at all
//                         (R-APP-010: "not asked", not merely "not
//                         told" — the suppression happens before
//                         any Resolve call).
//   - PermissionDefer   → emit `permission_decision_required`,
//                         park the call on `parked.park(callID)`,
//                         wait for the wake hand-off OR ctx
//                         cancellation. On wake, re-enter
//                         policy.Resolve and process the fresh
//                         verdict (D-A). On ctx cancel, populate
//                         the result slot with a typed abort
//                         failure and return proceed=false.
//   - AllowOnce         → emit `permission_decision_made{outcome=
//                         AllowOnce}` (R-APP-004), return
//                         proceed=true with no modifications.
//   - AllowAlways       → emit `permission_decision_made{outcome=
//                         AllowAlways}` (R-APP-007), invoke
//                         `policy.Remember`; true records the tool
//                         name in `remembered` and emits
//                         `permission_resolution_remembered`, false
//                         suppresses the emission (AG-10.4).
//   - Deny              → emit `permission_decision_made{outcome=
//                         Deny, failure=verdict.Failure}` (R-APP-005),
//                         populate the result slot with
//                         `Result{ExecutionFailure, typedDenial}`
//                         so the model sees the denial as a typed
//                         outcome (NOT a Go error), return
//                         proceed=false.
//   - ModifyInput       → emit `permission_decision_made{outcome=
//                         ModifyInput, modifiedArgs}` (R-APP-006),
//                         return proceed=true with `modifiedArgs`
//                         populated so executeCall rewrites the
//                         ToolStart payload and the tool's Run
//                         input. The bytes reach the tool and the
//                         stream in lockstep (transparency bite).
func (s *Scheduler) runPermissionGate(
	ctx context.Context,
	ordinal int,
	call ai.ToolCall,
	runID RunID,
	turnID TurnID,
	policy PermissionPolicy,
	parked *parkedSet,
	remembered *rememberedSet,
	results []Result,
	emissions chan<- emission,
) (proceed bool, modifiedArgs []byte, abortFailure *Failure) {
	// Bypass: nil policy preserves AG-09 behavior byte-clean.
	if policy == nil {
		return true, nil, nil
	}

	// R-APP-010: a remembered AllowAlways silences subsequent
	// identical calls for the rest of this Schedule call — the call
	// proceeds WITHOUT ever consulting Resolve (not merely without
	// emitting an event; "unasked", per the design data flow).
	if remembered.remembered(call.Name()) {
		return true, nil, nil
	}

	verdict := policy.Resolve(ctx, call)

	// Defer: emit decision_required, park, wait for wake OR
	// ctx cancel. AG-10.3 cancellation discipline writes a typed
	// abort failure into the result slot before returning, so
	// the rejoin slice is fully populated even when the run is
	// aborted mid-park (R-APP-009).
	if verdict.Outcome == PermissionDefer {
		reqEv, err := NewPermissionDecisionRequired(runID, turnID, call.ID(), call.Name(), call.Arguments())
		if err != nil {
			// Constructor failure is a typed execution failure
			// (R-APP-001's identity requirements reject empty
			// arguments, etc.). The result slot carries the typed
			// failure; the dispatcher emits the matching
			// tool_end_execution_failure.
			results[ordinal] = typedExecutionFailureFromError(call.ID(), err)
			emitExecutionFailure(ordinal, call, runID, turnID, results, emissions)
			return false, nil, nil
		}
		// W3 remediation: register the parked entry BEFORE emitting
		// decision_required, not after. The prior ordering emitted
		// first and parked second, so a consumer that read
		// decision_required off sink and immediately called
		// WakeParked(callID) could race ahead of registration — the
		// entry did not exist yet, and the wake was rejected with
		// ErrStrayDecision even though the call was genuinely about
		// to park. Registration is NOT the parked wait: parked.park
		// only inserts a channel into a map and returns — it does
		// not block — so R-APP-002/D4 ("emission reaches sink
		// before the parked wait blocks") still holds: the parked
		// WAIT is the select below, which still only runs after the
		// ack confirms the emission reached sink.
		parkCh := parked.park(call.ID())

		// R-APP-002/D4: "emission reaches sink before the parked
		// wait blocks" — a stronger guarantee than "enqueued onto
		// this buffered channel". Wait for the ack the dispatcher
		// closes only after sink <- &stamped completes, so the
		// select below cannot run until the event has genuinely
		// reached sink.
		//
		// W4 remediation: this wait is cancellation-aware. A bare
		// <-reqAck receive meant an abandoned sink (dispatcher
		// blocked forever on sink <- &stamped) hung this goroutine
		// — and, transitively, wg.Wait() and Schedule — with
		// cancelling the run context unable to release it. On
		// ctx.Done() here the parked entry is deregistered (it can
		// never be woken; this goroutine will never reach the
		// select on parkCh below) and the call aborts with the same
		// typed failure the mid-park cancellation path below
		// produces.
		reqAck := make(chan struct{})
		emissions <- emission{ev: reqEv, ack: reqAck}
		select {
		case <-reqAck:
		case <-ctx.Done():
			parked.remove(call.ID())
			abort := typedExecutionFailureFromError(call.ID(), ctx.Err())
			results[ordinal] = abort
			emitExecutionFailure(ordinal, call, runID, turnID, results, emissions)
			return false, nil, abort.Failure
		}

		select {
		case <-parkCh:
			// Upward-path wake (D-A) — the ONLY producer that
			// closes this channel is Scheduler.WakeParked (D-B:
			// park release has exactly two production paths, this
			// and ctx.Done() below). Re-enter policy.Resolve and
			// process the fresh verdict — the recursive call
			// reuses this same function's full outcome dispatch,
			// including the Defer branch (a policy that defers
			// again on wake is a policy quirk, not a protocol
			// bug: the call simply re-parks).
			return s.runPermissionGate(ctx, ordinal, call, runID, turnID, policy, parked, remembered, results, emissions)
		case <-ctx.Done():
			// Mid-park cancel: typed abort failure (AG-10.3
			// R-APP-009). The rejoin slice is fully populated
			// even when the run is aborted. Deregister (W3/W4:
			// every exit path removes the parked entry) so a late
			// WakeParked observes ErrStrayDecision rather than
			// spuriously closing a channel nothing will ever read.
			parked.remove(call.ID())
			abort := typedExecutionFailureFromError(call.ID(), ctx.Err())
			results[ordinal] = abort
			emitExecutionFailure(ordinal, call, runID, turnID, results, emissions)
			return false, nil, abort.Failure
		}
	}

	// AllowOnce: emit decision_made{AllowOnce}, proceed. A
	// constructor failure is a typed execution failure — the same
	// not-a-silent-drop posture as the decision_required constructor
	// above (defect 6 fix): a dropped decision_made must not let the
	// call proceed as though nothing happened.
	if verdict.Outcome == PermissionOutcomeAllowOnce {
		madeEv, err := NewPermissionDecisionMade(runID, turnID, call.ID(),
			PermissionOutcomeAllowOnce, nil, nil)
		if err != nil {
			abort := typedExecutionFailureFromError(call.ID(), err)
			results[ordinal] = abort
			emitExecutionFailure(ordinal, call, runID, turnID, results, emissions)
			return false, nil, abort.Failure
		}
		emissions <- emission{ev: madeEv}
		return true, nil, nil
	}

	// AllowAlways: emit decision_made{AllowAlways}, proceed, then
	// MUST invoke policy.Remember (R-APP-007). A true return records
	// the tool name in `remembered` (R-APP-010: silences subsequent
	// identical calls for the rest of this Schedule call) and emits
	// `permission_resolution_remembered`; a false return suppresses
	// the emission (preserves CardinalityAtMostOne, R-APE-003).
	if verdict.Outcome == PermissionOutcomeAllowAlways {
		madeEv, err := NewPermissionDecisionMade(runID, turnID, call.ID(),
			PermissionOutcomeAllowAlways, nil, nil)
		if err != nil {
			// Defect 6 fix: do not silently proceed (and, in
			// particular, do not silently invoke policy.Remember)
			// when the decision cannot even be recorded on the
			// stream.
			abort := typedExecutionFailureFromError(call.ID(), err)
			results[ordinal] = abort
			emitExecutionFailure(ordinal, call, runID, turnID, results, emissions)
			return false, nil, abort.Failure
		}
		emissions <- emission{ev: madeEv}
		if policy.Remember(ctx, call.Name(), PermissionOutcomeAllowAlways) {
			// W1 fix: rememberIfAbsent is a compare-and-set under
			// rememberedSet's own mutex. Two parallel calls to the
			// identical tool name can both pass the pre-Resolve
			// suppression check above (both observed "not yet
			// remembered") and both independently reach this branch
			// — but only the FIRST to call rememberIfAbsent gets
			// true; the loser gets false and suppresses its own
			// emission below, so at most one resolution_remembered
			// reaches the stream per tool name (R-APE-003
			// CardinalityAtMostOne) by construction. This does not
			// serialize Resolve or otherwise touch AG-09.2's
			// read-class concurrency policy — both calls still run
			// concurrently and both still execute; only the SECOND
			// emission is suppressed.
			if !remembered.rememberIfAbsent(call.Name()) {
				return true, nil, nil
			}
			// S2 (verify-report SUGGESTION): unlike defect 6's other
			// four constructor-failure sites in this function, a
			// resolution_remembered constructor failure here is
			// intentionally left best-effort, not upgraded to a
			// typed abort. By this point the primary decision is
			// ALREADY durably recorded (decision_made{AllowAlways}
			// above succeeded) and the state change already took
			// effect (rememberIfAbsent above already ran and won).
			// Treating this constructor failure as an abort would
			// retract execution of a call the stream has already
			// told the model is allowed — a strictly worse outcome
			// than dropping one best-effort telemetry event. The
			// failure is also unreachable in practice:
			// NewPermissionResolutionRemembered rejects only an
			// empty run, an empty toolName, or an out-of-vocabulary
			// outcome; run is the SAME value NewPermissionDecisionMade
			// already validated non-empty three lines above, toolName
			// is call.Name() (ai.ToolCall's own non-empty invariant),
			// and outcome is the hardcoded, always-valid
			// PermissionOutcomeAllowAlways constant.
			rememberedEv, rerr := NewPermissionResolutionRemembered(runID, call.Name(), PermissionOutcomeAllowAlways)
			if rerr == nil {
				emissions <- emission{ev: rememberedEv}
			}
		}
		return true, nil, nil
	}

	// Deny: typed rejection (R-APP-005, R-AEV-008). Populate the
	// result slot with `Result{ExecutionFailure, verdict.Failure}`
	// so the model sees the denial as a typed outcome — NOT a Go
	// error (which would hide the typed-failure surface). The
	// dispatcher emits `tool_end_execution_failure` from the
	// populated slot.
	if verdict.Outcome == PermissionOutcomeDeny {
		madeEv, err := NewPermissionDecisionMade(runID, turnID, call.ID(),
			PermissionOutcomeDeny, nil, verdict.Failure)
		if err != nil {
			// Defect 6 fix: a dropped decision_made must not be
			// silently absorbed into "proceed=false with no
			// explanation" — surface the constructor failure itself
			// as the typed execution failure.
			abort := typedExecutionFailureFromError(call.ID(), err)
			results[ordinal] = abort
			emitExecutionFailure(ordinal, call, runID, turnID, results, emissions)
			return false, nil, abort.Failure
		}
		emissions <- emission{ev: madeEv}
		var failRes Result
		if verdict.Failure != nil {
			failRes = Result{Outcome: ToolOutcomeExecutionFailure, Failure: verdict.Failure}
			failRes.SetCallID(call.ID())
		} else {
			// Defensive default: a Deny without a typed failure
			// is a policy defect; the typed failure surface still
			// requires a Failure. The result carries a typed
			// execution failure derived from a sentinel cause so
			// the rejoin slot is fully populated.
			failRes = typedExecutionFailureFromError(call.ID(),
				errPermissionDeniedWithoutFailure)
		}
		results[ordinal] = failRes
		emitExecutionFailure(ordinal, call, runID, turnID, results, emissions)
		return false, nil, failRes.Failure
	}

	// ModifyInput: substitute args, emit decision_made with the
	// modified bytes (R-APP-006 transparency). The caller
	// (executeCall) reads `modifiedArgs` and rewrites both the
	// ToolStart emission AND the tool's Run input.
	if verdict.Outcome == PermissionOutcomeModifyInput {
		madeEv, err := NewPermissionDecisionMade(runID, turnID, call.ID(),
			PermissionOutcomeModifyInput, verdict.ModifiedArgs, nil)
		if err != nil {
			// Defect 6 fix: a policy that returns ModifyInput without
			// populating ModifiedArgs (or otherwise producing an
			// invalid decision_made) must not silently execute with
			// the ORIGINAL arguments while the stream stays silent —
			// that breaks R-APP-006's transparency guarantee exactly
			// when it matters most (nobody can tell what actually
			// ran). Surface the constructor failure as a typed
			// execution failure instead.
			abort := typedExecutionFailureFromError(call.ID(), err)
			results[ordinal] = abort
			emitExecutionFailure(ordinal, call, runID, turnID, results, emissions)
			return false, nil, abort.Failure
		}
		emissions <- emission{ev: madeEv}
		return true, append([]byte(nil), verdict.ModifiedArgs...), nil
	}

	// Defensive default for any future PermissionOutcome value the
	// gate does not yet handle (mirrors AG-09's zero-outcome guard
	// at `executeCall`'s default branch).
	return true, nil, nil
}

// errPermissionDeniedWithoutFailure is the typed sentinel for a
// `Deny` verdict that lacks the required `*Failure` (R-AEV-008).
// A policy that returns Deny without populating the typed failure
// is a policy defect — the result slot still carries a typed
// execution failure so the rejoin is fully populated and the model
// sees the denial.
var errPermissionDeniedWithoutFailure = permissionDeniedWithoutFailure{}

type permissionDeniedWithoutFailure struct{}

func (permissionDeniedWithoutFailure) Error() string {
	return "agent: policy returned Deny without a typed *Failure (R-AEV-008 violation)"
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
