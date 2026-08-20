// AG-21 — Phase 1: shared state fixtures and signal firers for the
// combined-state x signal matrix (R-CNH-002, design AD-1/AD-2/AD-3).
//
// Every arranger below composes ONLY existing agenttest primitives
// (Gate, Provider, Script/Step's Emit/Hold) and this package's own
// existing local fixtures (delegatingTool, compactAfterNStrategy,
// scriptToolCallResponse, heldTurnScript, EchoScriptedTool,
// deferThenAllowPolicy, scriptedPermissionPolicy) — no new agenttest
// surface, no new Step shape (R-CNH-002, D5). The ONE new test-local
// assertion helper this change adds is requireHistoryPaired, below;
// it lives in package agent_test, never in agenttest.
//
// Synchronization throughout is exclusively Gate.Reached()/Release(),
// a synchronous nil Steer return, or an event read off a sink — never
// a clock (NFR-CNH-002). A defensive time.After deadline appears at a
// few waits below in the SAME shape drainSink/readUntilToolStart/
// buildCancellationFixture already use elsewhere in this package: it
// only guards against a genuinely stuck test and never decides
// "pending" or gates a signal itself.
package agent_test

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"
	"time"

	"github.com/cachicamas/backend/agent/src/agent"
	"github.com/cachicamas/backend/agent/src/agenttest"
	"github.com/cachicamas/backend/agent/src/ai"
)

// cnhSignal names which of the three signals a cell fires (R-CNH-001).
type cnhSignal int

const (
	cnhInterrupt cnhSignal = iota
	cnhShutdown
	cnhProviderFailure
)

// cnhRunOutcome carries Harness.Run's three return values across the
// goroutine boundary through a channel — the same posture
// harness_steering_test.go's resultCh and harness_suspension_test.go's
// suspendedRunOutcome both already use.
type cnhRunOutcome struct {
	msg    ai.Message
	finish ai.FinishReason
	err    error
}

// cnhArrangement is what each of the four state arrangers hands back
// to runCombinedCell: a run already driven to its own pending point
// (proven by a happens-before edge the production code itself
// provides), the consumer sink, the run's own completion channel,
// every event observed while proving the pending state, and a
// ready-to-call fire closure that fires this arrangement's own chosen
// signal. childTool and requestsAtPark are populated only by the two
// arrangers whose own cells need them.
type cnhArrangement struct {
	h        *agent.Harness
	sink     chan *agent.Event
	resultCh chan cnhRunOutcome
	pre      []agent.Event
	fire     func()

	// childTool is set only by arrangeChildHarnessActive, so the
	// child-cell containment checks (Phase 2) can reach the child's
	// own captured stream after the parent's run returns (R-DEL-005).
	childTool *delegatingTool

	// requestsAtPark is set only by arrangeSuspensionPending's
	// provider-failure branch: the injected provider's own recorded
	// request count at the moment the park was proven — S-CNH-005's
	// "no provider request recorded between the park and its
	// resolution" evidence, captured rather than assumed.
	requestsAtPark int
}

// cnhFailureCategory is the one failure category every provider-failure
// cell's terminal event carries — Unavailable, the same category
// wrapHarnessFailure already uses for a plain cancellation-unrelated
// cause, and deliberately non-retryable (FailureReport.Retryable's zero
// value) so retryDecision's G2 gate surfaces it immediately rather than
// retrying (retry_policy.go).
const cnhFailureCategory = ai.FailureCategoryUnavailable

// cnhFailureAfterHold builds a script that blocks at gate, then emits
// (optionally) a short text bracket followed by exactly one mid-stream
// terminal failure — design AD-2's shared "provider failure" signal
// shape: Hold, then an Emit whose firing is bounded by which provider
// stream is live (an Emit cannot fire itself; gate.Release() does).
// withText mirrors the steering-queued cell's own script content
// (design AD-2: "Hold(gate), Emit(start/delta/end), Emit(ErrorEvent)");
// the compaction and child-harness cells need no preceding text.
func cnhFailureAfterHold(t *testing.T, gate *agenttest.Gate, withText bool) agenttest.Script {
	t.Helper()

	steps := []agenttest.Step{agenttest.Hold(gate)}
	if withText {
		start, err := ai.NewTextBlockStart(1)
		if err != nil {
			t.Fatalf("ai.NewTextBlockStart: %v", err)
		}
		delta, err := ai.NewTextDelta(1, "cnh held-then-failure text")
		if err != nil {
			t.Fatalf("ai.NewTextDelta: %v", err)
		}
		end, err := ai.NewTextBlockEnd(1)
		if err != nil {
			t.Fatalf("ai.NewTextBlockEnd: %v", err)
		}
		steps = append(steps, agenttest.Emit(start), agenttest.Emit(delta), agenttest.Emit(end))
	}
	failure, err := ai.MidStreamFailure(ai.FailureReport{Category: cnhFailureCategory}, withText)
	if err != nil {
		t.Fatalf("ai.MidStreamFailure: %v", err)
	}
	terminal, err := ai.ErrorEvent(failure)
	if err != nil {
		t.Fatalf("ai.ErrorEvent: %v", err)
	}
	steps = append(steps, agenttest.Emit(terminal))
	return agenttest.Script{Steps: steps}
}

// cnhHeldSummaryScript builds a script that blocks at gate, then plays
// out an ordinary text completion — scriptTextResponse's own steps,
// prefixed with a Hold — design AD-2's compaction-provider shape for
// the interrupt/shutdown cells ("Hold(compGate) + summary").
func cnhHeldSummaryScript(t *testing.T, gate *agenttest.Gate) agenttest.Script {
	t.Helper()
	inner := scriptTextResponse(t, ai.FinishReasonStop)
	steps := make([]agenttest.Step, 0, len(inner.Steps)+1)
	steps = append(steps, agenttest.Hold(gate))
	steps = append(steps, inner.Steps...)
	return agenttest.Script{Steps: steps}
}

// fireInterrupt and fireShutdown are the two signal firers shared,
// unweakened, across every state: Harness.Interrupt/Shutdown never
// differ by what the run happens to be doing when they fire
// (R-CAN-001, R-CAN-004) — the "3 signal firers" this phase's tasks
// name alongside the 4 state arrangers. The third, provider-failure,
// has no single shared call: an Emit's own firing is bounded by which
// provider stream is live (design AD-2), so each arranger below builds
// its own provider-failure fire closure around its own gate or wake.
func fireInterrupt(h *agent.Harness) func() { return func() { h.Interrupt() } }
func fireShutdown(h *agent.Harness) func()  { return func() { h.Shutdown() } }

// requireHistoryPaired is this change's ONE new assertion helper
// (D5/R-CNH-002): every assistant tool-call part in hist's committed
// transcript has a matching result entry — real (ai.NewToolResult) or
// R-HIS-007-synthesized (ai.NewToolFailure, indistinguishable from a
// real result through Part.ToolResult() by design) — and no turn is
// left open, read through History's own CloseTurn semantics (R-HIS-003:
// commitCloseTurnOp validates rule 4 without writing anything, so
// calling it here is a read-only check, not a mutation).
func requireHistoryPaired(tb testing.TB, hist *agent.History) {
	tb.Helper()

	pending := map[string]bool{}
	for _, e := range hist.Entries() {
		msg := e.Message()
		switch msg.Role() {
		case ai.RoleAssistant:
			for _, part := range msg.Content() {
				if tc, ok := part.ToolCall(); ok {
					pending[tc.ID()] = true
				}
			}
		case ai.RoleTool:
			for _, part := range msg.Content() {
				if tr, ok := part.ToolResult(); ok {
					delete(pending, tr.CallID())
				}
			}
		}
	}
	if len(pending) != 0 {
		tb.Errorf("requireHistoryPaired: %d tool call(s) have no matching result entry (real or synthesized): %v", len(pending), pending)
	}

	if err := hist.CloseTurn(); err != nil {
		tb.Errorf("requireHistoryPaired: hist.CloseTurn() = %v, want nil (no open call, no turn left open)", err)
	}
}

// ---------------------------------------------------------------------
// State 1 — a permission-deferred call parked on the gate
// (design AD-2 row 1).
// ---------------------------------------------------------------------

// arrangeSuspensionPending drives a run to a call parked on the
// permission-decision gate (a policy returning PermissionDefer),
// proven pending by readUntilDecisionRequired's own channel read —
// never by elapsed time (R-CAN-003). For cnhProviderFailure the policy
// allows on its second Resolve (deferThenAllowPolicy, already shipped
// for S-RUN-090) and turn two's own script is the one that terminally
// fails, fired by waking the park: "the failure lands at the first
// provider boundary reachable from the pending state" (design AD-2),
// because no main-provider stream is live while parked (Schedule runs
// only after the turn's own provider stream reaches Completion). For
// cnhInterrupt/cnhShutdown the call stays parked (a policy that always
// defers) and the signal fires directly against it.
func arrangeSuspensionPending(t *testing.T, signal cnhSignal, label string) *cnhArrangement {
	t.Helper()

	callID := "call-cnh-susp-" + label
	toolName := "cnh_susp_tool_" + label
	tool := EchoScriptedTool(toolName, agent.EffectClassRead)
	reg := agent.NewMapRegistry(map[string]agent.Tool{toolName: tool})

	turnOneScript := scriptToolCallResponse(t, callID, toolName, []byte(`{}`))

	var provider *agenttest.Provider
	var policy agent.PermissionPolicy
	sched := &agent.Scheduler{}

	switch signal {
	case cnhProviderFailure:
		turnTwoScript := scriptTextThenTerminalFailure(t, cnhFailureCategory, true)
		provider = agenttest.NewProvider(turnOneScript, turnTwoScript)
		policy = &deferThenAllowPolicy{wakeIssued: &atomic.Bool{}}
	default:
		provider = agenttest.NewProvider(turnOneScript)
		policy = &scriptedPermissionPolicy{resolve: func(_ context.Context, _ ai.ToolCall) agent.PermissionVerdict {
			return agent.PermissionVerdict{Outcome: agent.PermissionDefer}
		}}
	}

	hist := agent.NewHistory()
	h := &agent.Harness{
		Provider:  provider,
		System:    "system prompt for cnh-suspension-" + label,
		Turn:      agent.TurnOptions{Tools: reg, PermissionPolicy: policy},
		Scheduler: sched,
		History:   hist,
	}

	sink := make(chan *agent.Event, 512)
	resultCh := make(chan cnhRunOutcome, 1)
	go func() {
		msg, finish, err := h.Run(contextBackground(), firstMessage(t), sink)
		resultCh <- cnhRunOutcome{msg, finish, err}
	}()

	pre, decision := readUntilDecisionRequired(t, sink)
	if decision.CallID() != callID {
		t.Fatalf("permission_decision_required.CallID() = %q, want %q", decision.CallID(), callID)
	}

	arr := &cnhArrangement{h: h, sink: sink, resultCh: resultCh, pre: pre}

	switch signal {
	case cnhInterrupt:
		arr.fire = fireInterrupt(h)
	case cnhShutdown:
		arr.fire = fireShutdown(h)
	case cnhProviderFailure:
		arr.requestsAtPark = len(provider.Requests())
		arr.fire = func() {
			if err := sched.WakeParked(callID); err != nil {
				t.Errorf("WakeParked(%q) = %v, want nil", callID, err)
			}
		}
	}
	return arr
}

// ---------------------------------------------------------------------
// State 2 — a steering message accepted with a nil return while the
// turn's provider stream is held (design AD-2 row 2).
// ---------------------------------------------------------------------

// arrangeSteeringQueued drives a run whose turn-one script holds at a
// Gate, proves the queued-and-undelivered state by a nil Steer return
// while the gate is still held, then wires the requested signal. A
// second, ordinary completing script is always queued behind turn
// one's: on the correct path the run winds down (interrupt/shutdown,
// ctx cancellation before the script's own completion step ever sends)
// or fails (provider failure, turn one's own bracket) strictly before
// the outer loop ever reaches it, so it is never consumed there — it
// exists so that bite (a)'s sabotage (deleting the interrupt/shutdown
// call) produces the clean outcome mismatch S-CNH-003 requires (the
// held script completes, the queued steer forces a real turn two,
// the run reaches RunOutcomeCompleted) rather than a hang.
func arrangeSteeringQueued(t *testing.T, signal cnhSignal, label string) *cnhArrangement {
	t.Helper()

	gate := agenttest.NewGate()
	t.Cleanup(gate.Release)

	var turnOneScript agenttest.Script
	if signal == cnhProviderFailure {
		turnOneScript = cnhFailureAfterHold(t, gate, true)
	} else {
		turnOneScript = heldTurnScript(t, gate)
	}
	turnTwoScript := scriptTextResponse(t, ai.FinishReasonStop)
	provider := agenttest.NewProvider(turnOneScript, turnTwoScript)

	hist := agent.NewHistory()
	h := &agent.Harness{Provider: provider, System: "system prompt for cnh-steering-" + label, History: hist}

	sink := make(chan *agent.Event, 512)
	resultCh := make(chan cnhRunOutcome, 1)
	go func() {
		msg, finish, err := h.Run(contextBackground(), firstMessage(t), sink)
		resultCh <- cnhRunOutcome{msg, finish, err}
	}()

	select {
	case <-gate.Reached():
	case <-time.After(5 * time.Second):
		t.Fatal("cnh steering: turn one never reached the Hold gate")
	}

	steered, err := ai.NewMessage(ai.RoleUser, mustText(t, "cnh steered message "+label))
	if err != nil {
		t.Fatalf("ai.NewMessage: %v", err)
	}
	if err := h.Steer(steered); err != nil {
		t.Fatalf("Steer returned %v, want nil — the message must be accepted while the gate is still held", err)
	}

	arr := &cnhArrangement{h: h, sink: sink, resultCh: resultCh}
	switch signal {
	case cnhInterrupt:
		// h.Interrupt() happens-before gate.Release() in this same
		// goroutine, so by the time the gate releases, runCtx is
		// already cancelled: even on the race where the Hold's own
		// select resolves via gate.released rather than ctx.Done(),
		// the run's OUTER iteration-boundary cause check still winds
		// it down correctly (harness.go's top-of-loop check). The
		// release is a deliberate safety net, load-bearing only when
		// bite (a) deletes the Interrupt() call above it — then the
		// gate still releases, the held script completes Stop, the
		// queued steer forces a real turn two, and the run reaches
		// RunOutcomeCompleted: the clean mismatch S-CNH-003 requires,
		// never a hang.
		arr.fire = func() { h.Interrupt(); gate.Release() }
	case cnhShutdown:
		arr.fire = func() { h.Shutdown(); gate.Release() }
	case cnhProviderFailure:
		arr.fire = func() { gate.Release() }
	}
	return arr
}

// ---------------------------------------------------------------------
// State 3 — a compaction call in flight (design AD-2 row 3).
// ---------------------------------------------------------------------

// arrangeCompactionMidRun first drives one ordinary completed turn to
// obtain a real turn mark (markedHarnessForCompaction), then drives a
// SECOND harness reusing that same History, whose ContextStrategy
// requests a compaction at its very first turn boundary — a compaction
// call in flight, proven pending by <-compGate.Reached(). For
// cnhProviderFailure the injected compaction provider's own script
// fails, and the failure is fired by compGate.Release() (the run
// itself is never interrupted, so the harness's own iteration-boundary
// cause check never matches and the run falls through to its own next
// turn, per R-CMP-010/AD-11 — a scriptTextResponse-shaped turn one
// script is provided for exactly that fallthrough). For
// cnhInterrupt/cnhShutdown the compaction provider's script is an
// ordinary held summary; firing the signal cancels the run's own ctx,
// which the Hold's own select already races against ctx.Done(), so no
// separate gate release is needed.
func arrangeCompactionMidRun(t *testing.T, signal cnhSignal, label string) *cnhArrangement {
	t.Helper()

	_, hist, _ := markedHarnessForCompaction(t, "cnh-"+label)

	compGate := agenttest.NewGate()
	t.Cleanup(compGate.Release)

	var compScript agenttest.Script
	if signal == cnhProviderFailure {
		compScript = cnhFailureAfterHold(t, compGate, false)
	} else {
		compScript = cnhHeldSummaryScript(t, compGate)
	}
	strategy := &compactAfterNStrategy{request: func(prompt agent.ContextPrompt) agent.CompactionRequest {
		return agent.CompactionRequest{
			Provider:    agenttest.NewProvider(compScript),
			Instruction: "summarize for cnh-compaction-" + label,
			Cut:         len(prompt.Transcript),
		}
	}}

	// Turn one's own provider: reached only once the compaction attempt
	// (successful or failed) has resolved and the run's own cause check
	// found no cancellation — the provider-failure cell's fallthrough,
	// and otherwise never reached for interrupt/shutdown (the run winds
	// down at that same cause check first).
	provider := agenttest.NewProvider(scriptTextResponse(t, ai.FinishReasonStop))

	h := &agent.Harness{
		Provider:        provider,
		System:          "system prompt for cnh-compaction-" + label,
		History:         hist,
		ContextStrategy: strategy,
	}

	sink := make(chan *agent.Event, 512)
	resultCh := make(chan cnhRunOutcome, 1)
	go func() {
		msg, finish, err := h.Run(contextBackground(), firstMessage(t), sink)
		resultCh <- cnhRunOutcome{msg, finish, err}
	}()

	select {
	case <-compGate.Reached():
	case <-time.After(5 * time.Second):
		t.Fatal("cnh compaction: compaction call never reached the Hold gate")
	}

	arr := &cnhArrangement{h: h, sink: sink, resultCh: resultCh}
	switch signal {
	case cnhInterrupt:
		// h.Interrupt() cancels runCtx, which the Hold's own select
		// already races against ctx.Done(); compGate.Release() is an
		// explicit, inline safety-net release (NFR-CNH-003) rather
		// than left to the t.Cleanup above — harmless once ctx is
		// already cancelled, idempotent either way.
		arr.fire = func() { h.Interrupt(); compGate.Release() }
	case cnhShutdown:
		arr.fire = func() { h.Shutdown(); compGate.Release() }
	case cnhProviderFailure:
		arr.fire = func() { compGate.Release() }
	}
	return arr
}

// ---------------------------------------------------------------------
// State 4 — a child harness hosted inside a tool call, its own provider
// stream held (design AD-2 row 4).
// ---------------------------------------------------------------------

// arrangeChildHarnessActive hosts a child agent.Harness inside a
// delegatingTool, proven pending by <-childGate.Reached() — the
// child's own stream is live and held. For cnhProviderFailure the
// CHILD's provider script fails and firing releases childGate; the
// parent is never itself interrupted, so a second parent-turn script
// is supplied for the parent's own fallthrough (R-DEL-002 containment,
// S-CNH-002). For cnhInterrupt/cnhShutdown the PARENT is signalled
// while the child's stream is held — mirroring buildCancellationFixture
// (cancellation_test.go) generalized to both signals — and the cause
// propagates transitively through the tool's own context derivation
// into the child's run, winding the child down first (R-DEL-006).
func arrangeChildHarnessActive(t *testing.T, signal cnhSignal, label string) *cnhArrangement {
	t.Helper()

	childGate := agenttest.NewGate()
	t.Cleanup(childGate.Release)

	var childScript agenttest.Script
	if signal == cnhProviderFailure {
		childScript = cnhFailureAfterHold(t, childGate, false)
	} else {
		childScript = heldTurnScript(t, childGate)
	}
	childProvider := agenttest.NewProvider(childScript)
	child := &agent.Harness{Provider: childProvider, System: "system prompt for cnh-child-" + label}

	toolName := "cnh_child_tool_" + label
	tool := &delegatingTool{toolName: toolName, effect: agent.EffectClassRead, child: child, prompt: firstMessage(t)}
	reg := agent.NewMapRegistry(map[string]agent.Tool{toolName: tool})

	callID := "call-cnh-child-" + label
	turnOneScript := scriptToolCallResponse(t, callID, toolName, []byte(`{}`))

	var provider *agenttest.Provider
	if signal == cnhProviderFailure {
		provider = agenttest.NewProvider(turnOneScript, scriptTextResponse(t, ai.FinishReasonStop))
	} else {
		provider = agenttest.NewProvider(turnOneScript)
	}

	hist := agent.NewHistory()
	h := &agent.Harness{
		Provider: provider,
		System:   "system prompt for cnh-child-parent-" + label,
		Turn:     agent.TurnOptions{Tools: reg},
		// A generous ceiling (NFR-CAN-002): never a synchronization
		// point — buildCancellationFixture's own precedent
		// (cancellation_test.go:88).
		Scheduler: &agent.Scheduler{WindDownBound: 5 * time.Second},
		History:   hist,
	}

	sink := make(chan *agent.Event, 512)
	resultCh := make(chan cnhRunOutcome, 1)
	go func() {
		msg, finish, err := h.Run(contextBackground(), firstMessage(t), sink)
		resultCh <- cnhRunOutcome{msg, finish, err}
	}()

	select {
	case <-childGate.Reached():
	case <-time.After(5 * time.Second):
		t.Fatal("cnh child-harness: child stream never reached the Hold gate")
	}

	arr := &cnhArrangement{h: h, sink: sink, resultCh: resultCh, childTool: tool}
	switch signal {
	case cnhInterrupt:
		// h.Interrupt() propagates the cause transitively into the
		// child's own runCtx; childGate.Release() is an explicit,
		// inline safety-net release (NFR-CNH-003) rather than left to
		// the t.Cleanup above.
		arr.fire = func() { h.Interrupt(); childGate.Release() }
	case cnhShutdown:
		arr.fire = func() { h.Shutdown(); childGate.Release() }
	case cnhProviderFailure:
		arr.fire = func() { childGate.Release() }
	}
	return arr
}

// ---------------------------------------------------------------------
// The shared assertion core (design AD-1/AD-3).
// ---------------------------------------------------------------------

// cellCase is one row of the combined-state x signal matrix (design
// AD-1): a state label, a signal label, the arranger to run, and two
// optional per-row overrides for the two cells whose Then this
// requirement's own text records as adjusted rather than uniform.
type cellCase struct {
	state  string
	signal string
	sig    cnhSignal
	arrange func(t *testing.T, signal cnhSignal, label string) *cnhArrangement

	// adjustedThen, when non-nil, REPLACES assertion 4's uniform check
	// for this cell (R-CNH-001's recorded charter deviation:
	// compaction x provider-failure, suspension x provider-failure).
	adjustedThen func(t *testing.T, arr *cnhArrangement, got cnhRunOutcome, events []agent.Event)

	// containment, when non-nil, is S-CNH-002's extra child-cell check,
	// run after the four (or adjusted) assertions above.
	containment func(t *testing.T, arr *cnhArrangement, events []agent.Event)
}

// runCombinedCell is the ONE shared core every cell in the matrix (and
// the leak sweep's own serial re-drive of these same cells) runs
// through — R-CNH-001's four uniform assertions, never twelve copies
// of them, so no cell can have its assertion core weakened alone
// (design AD-1's own rationale against a 12-named-functions shape).
func runCombinedCell(t *testing.T, label string, tc cellCase) {
	t.Helper()

	arr := tc.arrange(t, tc.sig, label)
	arr.fire()

	events := append(append([]agent.Event(nil), arr.pre...), drainSink(t, arr.sink)...)
	got := <-arr.resultCh // assertion 1: the run returned — a receive on the run's own completion channel, never a wall clock.

	// Assertion 2 — contract-ordered events: CheckStream over the full
	// drained stream reports no violation, stream unmodified.
	if report := agent.CheckStream(events); report.Violation() != nil {
		t.Errorf("%s: CheckStream(events) = %v, want nil (stream unmodified)", label, report.Violation())
	}

	// Assertion 3 — valid history, no open call.
	requireHistoryPaired(t, arr.h.History)

	// Assertion 4 — outcome and error match the firing signal, unless
	// this cell carries R-CNH-001's recorded adjusted Then.
	if tc.adjustedThen != nil {
		tc.adjustedThen(t, arr, got, events)
	} else {
		assertUniformOutcome(t, label, tc.sig, got, events)
	}

	if tc.containment != nil {
		tc.containment(t, arr, events)
	}
}

// assertUniformOutcome is R-CNH-001 assertion 4's uniform shape: an
// interrupt/shutdown ends the run with the matching outcome, no
// *Failure, and the matching (never the other) sentinel; a provider
// failure ends the run failed with a typed *Failure on the R-RUN-011
// path.
func assertUniformOutcome(t *testing.T, label string, sig cnhSignal, got cnhRunOutcome, events []agent.Event) {
	t.Helper()

	if len(events) == 0 {
		t.Fatalf("%s: zero events observed", label)
	}
	last := events[len(events)-1]
	runEnd, ok := last.RunEnd()
	if !ok {
		t.Fatalf("%s: last event kind = %v, want run_end", label, last.Kind())
	}

	switch sig {
	case cnhInterrupt:
		if !errors.Is(got.err, agent.ErrInterrupted) {
			t.Errorf("%s: Run error = %v, want errors.Is(_, agent.ErrInterrupted)", label, got.err)
		}
		if errors.Is(got.err, agent.ErrShutdown) {
			t.Errorf("%s: Run error unexpectedly also satisfies errors.Is(_, agent.ErrShutdown)", label)
		}
		if runEnd.Outcome() != agent.RunOutcomeInterrupted {
			t.Errorf("%s: run_end outcome = %v, want RunOutcomeInterrupted", label, runEnd.Outcome())
		}
		if _, hasFailure := runEnd.Failure(); hasFailure {
			t.Errorf("%s: run_end carries a *Failure, want none for an interrupted run", label)
		}
	case cnhShutdown:
		if !errors.Is(got.err, agent.ErrShutdown) {
			t.Errorf("%s: Run error = %v, want errors.Is(_, agent.ErrShutdown)", label, got.err)
		}
		if errors.Is(got.err, agent.ErrInterrupted) {
			t.Errorf("%s: Run error unexpectedly also satisfies errors.Is(_, agent.ErrInterrupted)", label)
		}
		if runEnd.Outcome() != agent.RunOutcomeShutdown {
			t.Errorf("%s: run_end outcome = %v, want RunOutcomeShutdown", label, runEnd.Outcome())
		}
		if _, hasFailure := runEnd.Failure(); hasFailure {
			t.Errorf("%s: run_end carries a *Failure, want none for a shutdown run", label)
		}
	case cnhProviderFailure:
		if got.err == nil {
			t.Fatalf("%s: Run error = nil, want a non-nil failure", label)
		}
		if runEnd.Outcome() != agent.RunOutcomeFailed {
			t.Errorf("%s: run_end outcome = %v, want RunOutcomeFailed", label, runEnd.Outcome())
		}
		if _, hasFailure := runEnd.Failure(); !hasFailure {
			t.Errorf("%s: run_end carries no *Failure, want one for a failed run (R-RUN-011)", label)
		}
	}
}
