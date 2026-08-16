// AG-10 — strict-TDD tests for the permission protocol
// (R-APP-001..012, S-APP-001..013 + bites S-PPB-001..004).
//
// Every scenario is in package agent_test (NFR-PRH-001 carry;
// NFR-TLS-001; AG-07 W6 / AG-08 / AG-09 carry): the external-test
// posture proves every behavioral claim from outside the package,
// with no reach into unexported surface.
//
// # Bite-first RED ordering (defense against AG-05 W1)
//
// S-PPB-001..004 (immediate allow / defer emit order / stray
// rejection / remembered cardinality) are RED-recorded BEFORE the
// corresponding property scenarios GREEN. The bites are the
// non-vacuous-helper defense AG-09 carried into AG-10.
//
// # Substrate preservation (NFR-TLS-003 7th carry)
//
// `TestTurn_SubstrateUntouched` (loop_test.go) widens its filter to
// exclude this file and `permission_protocol.go` at apply time.
// The substrate's 10-file list stays byte-untouched.
//
// # Implementation shape
//
// AG-10's file set:
//   - permission_protocol.go     — port + verdict + parkedSet (new)
//   - permission_protocol_test.go (this file)
//
// Substrate untouched:
//   - event_descriptor.go, stream_check.go, failure.go, sequence.go,
//     event.go, go.mod, go.sum, Makefile, .golangci.yml,
//     import_boundary_test.go (the canonical NFR-TLS-003 list).
package agent_test

import (
	"context"
	"errors"
	"runtime"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/cachicamas/backend/agent/src/agent"
	"github.com/cachicamas/backend/agent/src/ai"
)

// scriptedPermissionPolicy is an in-test agent.PermissionPolicy
// whose Resolve and Remember are configurable per-instance. The
// test sets `Resolve` and `Remember` to the desired behaviors;
// the script records invocation counts and tool names for
// assertion.
//
// Lives in the external package agent_test so the protocol
// contract is exercised through the public surface only
// (NFR-PRH-001, NFR-TLS-001 carry). The `var _ agent.PermissionPolicy`
// alias at the bottom of the file is the compile-time guard.
type scriptedPermissionPolicy struct {
	resolve  func(ctx context.Context, call ai.ToolCall) agent.PermissionVerdict
	remember func(ctx context.Context, toolName string, outcome agent.PermissionOutcome) bool

	mu          sync.Mutex
	resolveCalls   []ai.ToolCall
	rememberCalls  []string
	resolveCount   atomic.Int64
	rememberCount  atomic.Int64
}

func (p *scriptedPermissionPolicy) Resolve(ctx context.Context, call ai.ToolCall) agent.PermissionVerdict {
	p.resolveCount.Add(1)
	p.mu.Lock()
	p.resolveCalls = append(p.resolveCalls, call)
	p.mu.Unlock()
	if p.resolve == nil {
		// Default: AllowOnce for everything.
		return agent.PermissionVerdict{Outcome: agent.PermissionOutcomeAllowOnce}
	}
	return p.resolve(ctx, call)
}

func (p *scriptedPermissionPolicy) Remember(ctx context.Context, toolName string, outcome agent.PermissionOutcome) bool {
	p.rememberCount.Add(1)
	p.mu.Lock()
	p.rememberCalls = append(p.rememberCalls, toolName)
	p.mu.Unlock()
	if p.remember == nil {
		// Default: never remember.
		return false
	}
	return p.remember(ctx, toolName, outcome)
}

func (p *scriptedPermissionPolicy) seenResolveToolNames() []string {
	p.mu.Lock()
	defer p.mu.Unlock()
	out := make([]string, len(p.resolveCalls))
	for i, c := range p.resolveCalls {
		out[i] = c.Name()
	}
	return out
}

func (p *scriptedPermissionPolicy) resolveInvocations() int64 {
	return p.resolveCount.Load()
}

func (p *scriptedPermissionPolicy) rememberInvocations() int64 {
	return p.rememberCount.Load()
}

// scheduledPermissionCall mirrors scheduler_test.go's `scheduledCall`
// helper but tagged for the permission test files so the two
// stay independent (one test file per milestone keeps helper
// diffs trivial).
type scheduledPermissionCall struct {
	id   string
	name string
	args []byte
}

// permissionCallsToAICalls converts the in-test scheduledPermissionCall
// shape into the Layer 1 ai.ToolCall slice the agent.Scheduler
// expects.
func permissionCallsToAICalls(calls []scheduledPermissionCall) []ai.ToolCall {
	parts := make([]ai.Part, len(calls))
	for i, c := range calls {
		part, err := ai.NewToolCall(c.id, c.name, c.args)
		if err != nil {
			panic("permissionCallsToAICalls: invalid ToolCall at index " + itoa(i) + ": " + err.Error())
		}
		parts[i] = part
	}
	out := make([]ai.ToolCall, len(parts))
	for i, p := range parts {
		tc, ok := p.ToolCall()
		if !ok {
			panic("permissionCallsToAICalls: NewToolCall returned non-tool-call part at index " + itoa(i))
		}
		out[i] = tc
	}
	return out
}

// runPermissionSchedulerAndCollect invokes Schedule with the new
// signature (which Commit 3 will land) and drains the sink for
// inspection. Mirrors runSchedulerAndCollect in scheduler_test.go.
//
// RED in Commit 2: the new signature does not exist yet — the
// bites are compile-error RED, which AG-09's bites also were
// (scheduler_test.go:155-160). Commit 3 lands the signature and
// drives the bites GREEN.
func runPermissionSchedulerAndCollect(
	sched *agent.Scheduler,
	calls []scheduledPermissionCall,
	reg agent.Registry,
	policy agent.PermissionPolicy,
) ([]agent.Result, []*agent.Event) {
	stamper := &agent.LaneStamper{}
	sink := make(chan *agent.Event, 64)
	results := sched.Schedule(
		context.Background(),
		permissionCallsToAICalls(calls),
		reg,
		"run-permission",
		"turn-permission",
		policy,
		stamper,
		sink,
	)
	var events []*agent.Event
	for ev := range sink {
		evCopy := ev
		events = append(events, evCopy)
	}
	return results, events
}

// countByKind walks events and returns how many carry each
// discriminator kind, used by the bites to assert absence
// (immediate-allow produces no decision_required) and presence
// (defer produces exactly one per parked call).
func countByKind(events []*agent.Event) map[agent.EventKind]int {
	out := map[agent.EventKind]int{}
	for _, ev := range events {
		out[ev.Kind()]++
	}
	return out
}

// S-PPB-001 — RED bite (recorded at Commit 2). Given a policy that
// returns AllowOnce for one call, when Schedule runs, then:
//
//   (a) NO permission_decision_required event is emitted on the
//       sink — AllowOnce is synchronous and bypasses the gate;
//   (b) the tool IS invoked exactly once;
//   (c) results[0].Outcome is Success.
//
// RED at write time: the production code under test (Schedule
// with the `policy PermissionPolicy` parameter, plus the per-call
// gate) does not exist yet — the bite is compile-error RED, the
// strongest signal a missing surface can deliver
// (scheduler_test.go:155-160 + AG-05 W1 defense).
func TestPermission_ImmediateAllow_NoEvent(t *testing.T) {
	t.Parallel()

	policy := &scriptedPermissionPolicy{
		resolve: func(_ context.Context, _ ai.ToolCall) agent.PermissionVerdict {
			return agent.PermissionVerdict{Outcome: agent.PermissionOutcomeAllowOnce}
		},
	}

	tool := EchoScriptedTool("read_file", agent.EffectClassRead)
	reg := newMapRegistry(map[string]agent.Tool{"read_file": tool})

	calls := []scheduledPermissionCall{
		{id: "imm_0", name: "read_file", args: []byte(`{}`)},
	}

	results, events := runPermissionSchedulerAndCollect(
		&agent.Scheduler{MaxConcurrentReads: 4}, calls, reg, policy,
	)

	// (a) Zero permission_decision_required events: AllowOnce bypasses
	// the gate. A scheduler that always emitted decision_required
	// regardless of verdict would fail this assertion.
	if got := countByKind(events)[agent.EventKindPermissionDecisionRequired]; got != 0 {
		t.Errorf("immediate allow emitted %d permission_decision_required event(s), want 0 — AllowOnce bypasses the gate (R-APP-001, S-PPB-001 bite)",
			got)
	}

	// (b) The tool was invoked exactly once — synchronous AllowOnce
	// does not park the call.
	if inv := tool.Invocations(); inv != 1 {
		t.Errorf("read_file invocations = %d, want 1 (AllowOnce synchronous, no park)", inv)
	}

	// (c) The result slot is populated with success.
	if len(results) != 1 {
		t.Fatalf("Schedule returned %d results, want 1", len(results))
	}
	if results[0].Outcome != agent.ToolOutcomeSuccess {
		t.Errorf("results[0].Outcome = %v, want %v", results[0].Outcome, agent.ToolOutcomeSuccess)
	}
}

// R-APP-002 / D4 — task 2.2. Deferring a call MUST emit
// permission_decision_required and that emission MUST reach `sink`
// BEFORE the parked wait blocks — the spec's exact words ("emission
// reaches sink before the parked wait blocks"), not merely "before
// the call goroutine enqueues onto an internal buffer".
//
// Renamed from TestPermission_DeferEmitsAndParks: the old name
// described what the test drove, not the property it proved (and
// the old body never established the ordering at all — see below).
//
// The old implementation could not honor this: the gate sent the
// decision_required onto a BUFFERED internal `emissions` channel and
// immediately proceeded to `parked.park()` + the parked `select`,
// entirely decoupled from whether a separate dispatcher goroutine
// had actually forwarded the event to `sink` yet. That gap is fixed
// (this milestone) by giving the decision_required emission an ack
// channel the dispatcher closes only after `sink <- &stamped`
// completes; the gate now blocks on that ack before calling
// `parked.park()`.
//
// This test defeats the gap directly rather than trusting the fix:
// with `sink` UNBUFFERED and nothing reading from it, WakeParked on
// the deferred call's ID MUST keep failing (ErrStrayDecision —
// nothing registered yet) for as long as nothing reads from sink,
// because an unbuffered channel cannot have delivered the event
// without a reader. Only once the event is actually read does the
// call become parked (and only then does WakeParked succeed).
func TestPermission_DeferEmitsBeforePark(t *testing.T) {
	t.Parallel()

	var attempt atomic.Int64
	policy := &scriptedPermissionPolicy{
		resolve: func(_ context.Context, _ ai.ToolCall) agent.PermissionVerdict {
			if attempt.Add(1) == 1 {
				return agent.PermissionVerdict{Outcome: agent.PermissionDefer}
			}
			// Wake's re-evaluation (D-A): finish as AllowOnce.
			return agent.PermissionVerdict{Outcome: agent.PermissionOutcomeAllowOnce}
		},
	}

	tool := EchoScriptedTool("order_tool", agent.EffectClassRead)
	reg := newMapRegistry(map[string]agent.Tool{"order_tool": tool})
	calls := []scheduledPermissionCall{{id: "order_0", name: "order_tool", args: []byte(`{}`)}}

	sched := &agent.Scheduler{MaxConcurrentReads: 4}
	stamper := &agent.LaneStamper{}
	sink := make(chan *agent.Event) // UNBUFFERED — no room to hide an undelivered event.

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	doneCh := make(chan []agent.Result, 1)
	go func() {
		doneCh <- sched.Schedule(ctx, permissionCallsToAICalls(calls), reg, "run-order", "turn-order", policy, stamper, sink)
	}()

	// Poll WakeParked WITHOUT reading from sink at all. A premature
	// success would prove the call parked before decision_required
	// could possibly have reached sink.
	pollDeadline := time.Now().Add(300 * time.Millisecond)
	for time.Now().Before(pollDeadline) {
		if err := sched.WakeParked("order_0"); err == nil {
			t.Fatal("WakeParked succeeded before anything read from sink — the call parked before decision_required could have reached sink (R-APP-002/D4 ordering violated)")
		}
		time.Sleep(2 * time.Millisecond)
	}

	// The only possible next readable event, since nothing has been
	// read yet and sink is unbuffered: decision_required for order_0.
	var ev *agent.Event
	select {
	case ev = <-sink:
	case <-time.After(1 * time.Second):
		t.Fatal("no event reached sink within 1s (decision_required for order_0 was never delivered)")
	}
	req, ok := ev.PermissionDecisionRequired()
	if !ok || req.CallID() != "order_0" {
		t.Fatalf("first event on sink = %#v, want permission_decision_required for order_0", ev)
	}

	// NOW the call must be parked — its event has been delivered.
	// The ack only proves the dispatcher's send has completed; the
	// call goroutine still needs a scheduling turn to run
	// parked.park() after waking from <-reqAck, so poll briefly
	// rather than asserting on a single attempt (this is ordinary
	// goroutine-scheduling latency, not the R-APP-002/D4 ordering
	// this test defends — that property is the lower bound already
	// proven above: WakeParked never succeeds before the event is
	// read).
	registerDeadline := time.Now().Add(500 * time.Millisecond)
	var wakeErr error
	for time.Now().Before(registerDeadline) {
		wakeErr = sched.WakeParked("order_0")
		if wakeErr == nil {
			break
		}
		time.Sleep(time.Millisecond)
	}
	if wakeErr != nil {
		t.Fatalf("WakeParked(%q) = %v, want nil now that decision_required has been delivered", "order_0", wakeErr)
	}

	drainDone := make(chan struct{})
	go func() {
		defer close(drainDone)
		for range sink {
			// Drain to completion so Schedule's dispatcher never
			// blocks on a full sink.
			_ = struct{}{}
		}
	}()

	select {
	case results := <-doneCh:
		if len(results) != 1 {
			t.Fatalf("Schedule returned %d results, want 1", len(results))
		}
		if results[0].Outcome != agent.ToolOutcomeSuccess {
			t.Errorf("results[0].Outcome = %v, want Success", results[0].Outcome)
		}
	case <-time.After(1 * time.Second):
		t.Fatal("Schedule did not return within 1s after the wake")
	}
	<-drainDone
	if inv := tool.Invocations(); inv != 1 {
		t.Errorf("order_tool invocations = %d, want 1", inv)
	}
}

// S-PPB-003 — re-pointed at the real production wake surface (D-A).
// The original bite exercised a vacuous helper (`ResolveStrayDecision`,
// deleted) that discarded its parameter and always returned the
// sentinel — it would have passed identically for a KNOWN callID
// too. This version drives `Scheduler.WakeParked`, the actual
// upward-path hand-off: given an idle Scheduler with no Schedule
// call in flight (there is no parked set to search, let alone the
// unknown callID), when WakeParked fires for an unknown callID, then
// it returns the typed ErrStrayDecision error — NOT a silent drop,
// NOT a panic.
//
// The complementary scenario — a stray wake against a Schedule call
// that DOES have a genuinely parked call, proving the stray wake
// touches nothing (S-APP-004's second clause) — is
// TestPermission_WakeParked_UnknownCallID_TypedRejection_NoTouch
// below.
func TestPermission_StrayDecisionIsTypedError(t *testing.T) {
	t.Parallel()

	sched := &agent.Scheduler{}
	const strayID = "call-id-X"
	err := sched.WakeParked(strayID)
	if err == nil {
		t.Fatalf("WakeParked(%q) returned nil, want a non-nil typed error", strayID)
	}
	if !errors.Is(err, agent.ErrStrayDecision) {
		t.Errorf("WakeParked(%q) = %v, want errors.Is to match agent.ErrStrayDecision",
			strayID, err)
	}
}

// S-PPB-004 / task 2.4 — re-pointed at a real Schedule run. The
// original bite hand-built two permission_resolution_remembered
// events and called CheckStream directly, exercising zero AG-10
// code — it duplicated permission_events_test.go:263-317's S-APE-082
// bite verbatim. This version drives the same CardinalityAtMostOne
// rejection from events a real Schedule call actually emits.
//
// R-APP-010's suppression check runs BEFORE policy.Resolve and the
// remembered-write happens AFTER policy.Remember returns — for two
// SEQUENTIAL calls to the same tool name (the ordinary case,
// TestPermission_RememberedSuppressesSubsequentAsk) that ordering
// prevents a second ask entirely, so a single well-behaved Schedule
// call cannot naturally reach two resolution_remembered emissions
// for the same tool name. Two calls racing TRULY CONCURRENTLY for an
// identical tool name is a narrower, documented gap this bite proves
// the validator backstops rather than a gap AG-10.4 closes: both
// goroutines can pass the "not yet remembered" check before either's
// write lands, independently reach AllowAlways, and each
// independently emit resolution_remembered. A synchronization
// barrier forces that race deterministically (not by goroutine-
// scheduling luck), so the bite is neither flaky nor vacuous.
func TestPermission_RememberedCardinality_SecondEmissionRejected(t *testing.T) {
	t.Parallel()

	var enteredResolve sync.WaitGroup
	enteredResolve.Add(2)
	release := make(chan struct{})
	policy := &scriptedPermissionPolicy{
		resolve: func(_ context.Context, _ ai.ToolCall) agent.PermissionVerdict {
			enteredResolve.Done()
			<-release // rendezvous: force both calls to pass the suppression check before either can record it
			return agent.PermissionVerdict{Outcome: agent.PermissionOutcomeAllowAlways}
		},
		remember: func(_ context.Context, _ string, _ agent.PermissionOutcome) bool {
			return true
		},
	}

	tool := EchoScriptedTool("fs.write", agent.EffectClassRead)
	reg := newMapRegistry(map[string]agent.Tool{"fs.write": tool})
	calls := []scheduledPermissionCall{
		{id: "race_0", name: "fs.write", args: []byte(`{}`)},
		{id: "race_1", name: "fs.write", args: []byte(`{}`)},
	}

	go func() {
		enteredResolve.Wait()
		close(release)
	}()

	runID := agent.RunID("run-app-082-real")
	turnID := agent.TurnID("turn-app-082-real")

	// One shared LaneStamper across the caller-built brackets and
	// Schedule's own emissions — exactly how Turn() assembles a
	// stream in production (loop.go: the stamper is minted once,
	// used for run_start/turn_start, handed to Schedule, then reused
	// for turn_end/run_end). A per-section stamper would produce a
	// non-contiguous sequence and CheckStream would reject the
	// stream for the wrong reason.
	stamper := &agent.LaneStamper{}
	runStart, err := agent.NewRunStart(runID)
	if err != nil {
		t.Fatalf("agent.NewRunStart: %v", err)
	}
	turnStart, err := agent.NewTurnStart(runID, turnID)
	if err != nil {
		t.Fatalf("agent.NewTurnStart: %v", err)
	}
	stampedRunStart := stamper.Stamp(runStart)
	stampedTurnStart := stamper.Stamp(turnStart)

	sink := make(chan *agent.Event, 64)
	results := (&agent.Scheduler{MaxConcurrentReads: 4}).Schedule(
		context.Background(),
		permissionCallsToAICalls(calls),
		reg,
		runID,
		turnID,
		policy,
		stamper,
		sink,
	)
	var scheduleEvents []agent.Event
	for ev := range sink {
		scheduleEvents = append(scheduleEvents, *ev)
	}

	if len(results) != 2 {
		t.Fatalf("Schedule returned %d results, want 2", len(results))
	}
	rememberedCount := 0
	for _, ev := range scheduleEvents {
		if _, ok := ev.PermissionResolutionRemembered(); ok {
			rememberedCount++
		}
	}
	if rememberedCount != 2 {
		t.Fatalf("resolution_remembered count = %d, want 2 (the forced race must make both calls independently remember for this bite to exercise anything — the property under test is that the VALIDATOR then rejects the resulting stream, not that the scheduler prevents the race)", rememberedCount)
	}

	turnEnd, err := agent.NewTurnEnd(runID, turnID, agent.TurnOutcomeFinished, nil)
	if err != nil {
		t.Fatalf("agent.NewTurnEnd: %v", err)
	}
	runEnd, err := agent.NewRunEnd(runID, agent.RunOutcomeCompleted, nil)
	if err != nil {
		t.Fatalf("agent.NewRunEnd: %v", err)
	}
	stampedTurnEnd := stamper.Stamp(turnEnd)
	stampedRunEnd := stamper.Stamp(runEnd)

	stream := make([]agent.Event, 0, len(scheduleEvents)+4)
	stream = append(stream, stampedRunStart, stampedTurnStart)
	stream = append(stream, scheduleEvents...)
	stream = append(stream, stampedTurnEnd, stampedRunEnd)

	report := agent.CheckStream(stream)
	if report.Violation() == nil {
		t.Fatalf("CheckStream accepted two permission_resolution_remembered events for the same tool name; CardinalityAtMostOne seam (R-APE-003, S-APE-082) MUST reject the second — the validator is the backstop for AG-10's documented concurrent-remember race")
	}
	if !errors.Is(report.Violation(), ai.ErrDuplicate) {
		t.Errorf("rejection error = %v, want errors.Is to match ai.ErrDuplicate (CardinalityAtMostOne seam)", report.Violation())
	}
}

// Compile-time guard: scriptedPermissionPolicy must satisfy
// agent.PermissionPolicy. A non-conforming implementation fails
// to compile here, not at runtime under a Schedule call.
var _ agent.PermissionPolicy = (*scriptedPermissionPolicy)(nil)

// --- AG-10.2 four-outcome tests -------------------------------------

// S-APP-005 (AG-10.2 AllowOnce) — sync AllowOnce emits a
// permission_decision_made{outcome=AllowOnce} BEFORE the tool runs
// (R-APP-004). The decision event rides through the existing
// emissions channel; the tool executes with the original arguments;
// the result is success.
//
// RED-recorded at Commit 4 (the AG-10.1 implementation did not emit
// decision_made on sync paths — that was the immediate-allow
// property). GREEN at this commit.
func TestPermission_FourOutcomes_AllowOnce_EmitsDecisionMade(t *testing.T) {
	t.Parallel()

	policy := &scriptedPermissionPolicy{
		resolve: func(_ context.Context, _ ai.ToolCall) agent.PermissionVerdict {
			return agent.PermissionVerdict{Outcome: agent.PermissionOutcomeAllowOnce}
		},
	}

	tool := EchoScriptedTool("read_file", agent.EffectClassRead)
	reg := newMapRegistry(map[string]agent.Tool{"read_file": tool})

	calls := []scheduledPermissionCall{
		{id: "allow_once_0", name: "read_file", args: []byte(`{"path":"/x"}`)},
	}

	results, events := runPermissionSchedulerAndCollect(
		&agent.Scheduler{MaxConcurrentReads: 4}, calls, reg, policy,
	)

	// No decision_required (sync verdict).
	if got := countByKind(events)[agent.EventKindPermissionDecisionRequired]; got != 0 {
		t.Errorf("decision_required count = %d, want 0 (AllowOnce is sync, R-APP-001)", got)
	}
	// One decision_made{AllowOnce}.
	madeEvs := []agent.PermissionDecisionMade{}
	for _, ev := range events {
		if made, ok := ev.PermissionDecisionMade(); ok {
			madeEvs = append(madeEvs, made)
		}
	}
	if len(madeEvs) != 1 {
		t.Fatalf("decision_made count = %d, want 1 (R-APP-004)", len(madeEvs))
	}
	if madeEvs[0].Outcome() != agent.PermissionOutcomeAllowOnce {
		t.Errorf("decision_made outcome = %v, want AllowOnce", madeEvs[0].Outcome())
	}
	if madeEvs[0].CallID() != "allow_once_0" {
		t.Errorf("decision_made callID = %q, want %q", madeEvs[0].CallID(), "allow_once_0")
	}
	// Tool ran and result is success.
	if inv := tool.Invocations(); inv != 1 {
		t.Errorf("read_file invocations = %d, want 1", inv)
	}
	if len(results) != 1 {
		t.Fatalf("Schedule returned %d results, want 1", len(results))
	}
	if results[0].Outcome != agent.ToolOutcomeSuccess {
		t.Errorf("results[0].Outcome = %v, want Success", results[0].Outcome)
	}
}

// S-APP-006 (AG-10.2 Deny) — Deny populates the ordinal Result slot
// with `Result{Outcome: ExecutionFailure, Failure: <typed denial>}`
// (R-APP-005). The denial is visible to the model via the typed
// Result. The decision_made{Deny} event carries the typed failure.
//
// RED-recorded at Commit 4 (AG-10.1 stubbed Deny to "proceed").
// GREEN at this commit.
func TestPermission_FourOutcomes_Deny_TypedResultAndDecisionMade(t *testing.T) {
	t.Parallel()

	// Construct a typed *agent.Failure the test can identify.
	inner, err := ai.MidStreamFailure(ai.FailureReport{
		Category: ai.FailureCategoryUnavailable,
	}, false)
	if err != nil {
		t.Fatalf("ai.MidStreamFailure: %v", err)
	}
	denial, err := agent.NewFailure(inner)
	if err != nil {
		t.Fatalf("agent.NewFailure: %v", err)
	}

	policy := &scriptedPermissionPolicy{
		resolve: func(_ context.Context, _ ai.ToolCall) agent.PermissionVerdict {
			return agent.PermissionVerdict{
				Outcome: agent.PermissionOutcomeDeny,
				Failure: denial,
			}
		},
	}

	tool := EchoScriptedTool("dangerous_tool", agent.EffectClassRead)
	reg := newMapRegistry(map[string]agent.Tool{"dangerous_tool": tool})

	calls := []scheduledPermissionCall{
		{id: "deny_0", name: "dangerous_tool", args: []byte(`{"cmd":"rm -rf /"}`)},
	}

	results, events := runPermissionSchedulerAndCollect(
		&agent.Scheduler{MaxConcurrentReads: 4}, calls, reg, policy,
	)

	// Result slot is a typed ExecutionFailure (R-APP-005).
	if len(results) != 1 {
		t.Fatalf("Schedule returned %d results, want 1", len(results))
	}
	if results[0].Outcome != agent.ToolOutcomeExecutionFailure {
		t.Errorf("results[0].Outcome = %v, want ExecutionFailure (Deny surfaces as typed outcome, NOT Go error)",
			results[0].Outcome)
	}
	if results[0].Failure == nil {
		t.Fatal("results[0].Failure = nil, want typed *Failure carrying the denial")
	}
	if results[0].Failure.Category() != ai.FailureCategoryUnavailable {
		t.Errorf("results[0].Failure.Category = %v, want Unavailable", results[0].Failure.Category())
	}

	// Tool was NOT invoked — Deny skips execution.
	if inv := tool.Invocations(); inv != 0 {
		t.Errorf("dangerous_tool invocations = %d, want 0 (Deny skips execution)", inv)
	}

	// decision_made{Deny} event with the typed failure (R-APP-005).
	madeEvs := []agent.PermissionDecisionMade{}
	for _, ev := range events {
		if made, ok := ev.PermissionDecisionMade(); ok {
			madeEvs = append(madeEvs, made)
		}
	}
	if len(madeEvs) != 1 {
		t.Fatalf("decision_made count = %d, want 1", len(madeEvs))
	}
	if madeEvs[0].Outcome() != agent.PermissionOutcomeDeny {
		t.Errorf("decision_made outcome = %v, want Deny", madeEvs[0].Outcome())
	}
	if madeEvs[0].CallID() != "deny_0" {
		t.Errorf("decision_made callID = %q, want %q", madeEvs[0].CallID(), "deny_0")
	}
	f, ok := madeEvs[0].Failure()
	if !ok {
		t.Errorf("decision_made{Failure} returned no failure")
	} else if f != denial {
		t.Errorf("decision_made failure = %v, want the typed denial passed to Resolve", f)
	}
}

// S-APP-007 (AG-10.2 ModifyInput transparency) — ModifyInput
// substitutes the arguments before ToolStart emission and Tool.Run;
// `ToolStart.Arguments()` byte-equals `decision_made.ModifiedArguments()`.
// The session log reconstructs "what ran" from the modified bytes.
//
// RED-recorded at Commit 4 (AG-10.1 stubbed ModifyInput to proceed
// without substitution). GREEN at this commit.
func TestPermission_FourOutcomes_ModifyInput_SubstitutesArguments(t *testing.T) {
	t.Parallel()

	const modifiedArgs = `{"cmd":"ls"}`

	policy := &scriptedPermissionPolicy{
		resolve: func(_ context.Context, _ ai.ToolCall) agent.PermissionVerdict {
			return agent.PermissionVerdict{
				Outcome:      agent.PermissionOutcomeModifyInput,
				ModifiedArgs: []byte(modifiedArgs),
			}
		},
	}

	// Capture what arguments the tool actually sees.
	receivedArgsCh := make(chan []byte, 1)
	tool := NewScriptedTool("modify_tool", agent.EffectClassRead,
		agent.Result{Outcome: agent.ToolOutcomeSuccess, Content: []byte("ok")},
	)
	tool.Script = func(_ context.Context, args []byte, _ agent.PolicySlot) (agent.Result, error) {
		// Copy because the scheduler reuses buffers.
		cp := append([]byte(nil), args...)
		select {
		case receivedArgsCh <- cp:
		default:
		}
		return agent.Result{Outcome: agent.ToolOutcomeSuccess, Content: []byte("ok")}, nil
	}
	reg := newMapRegistry(map[string]agent.Tool{"modify_tool": tool})

	calls := []scheduledPermissionCall{
		{id: "modify_0", name: "modify_tool", args: []byte(`{"cmd":"rm -rf /"}`)},
	}

	results, events := runPermissionSchedulerAndCollect(
		&agent.Scheduler{MaxConcurrentReads: 4}, calls, reg, policy,
	)

	// decision_made{ModifyInput, modifiedArgs}.
	madeEvs := []agent.PermissionDecisionMade{}
	for _, ev := range events {
		if made, ok := ev.PermissionDecisionMade(); ok {
			madeEvs = append(madeEvs, made)
		}
	}
	if len(madeEvs) != 1 {
		t.Fatalf("decision_made count = %d, want 1", len(madeEvs))
	}
	if madeEvs[0].Outcome() != agent.PermissionOutcomeModifyInput {
		t.Errorf("decision_made outcome = %v, want ModifyInput", madeEvs[0].Outcome())
	}
	if string(madeEvs[0].ModifiedArguments()) != modifiedArgs {
		t.Errorf("decision_made modified args = %q, want %q (R-APP-006 transparency)",
			madeEvs[0].ModifiedArguments(), modifiedArgs)
	}

	// ToolStart.Arguments() byte-equals decision_made.ModifiedArguments().
	var startEvs []agent.ToolStart
	for _, ev := range events {
		if start, ok := ev.ToolStart(); ok {
			startEvs = append(startEvs, start)
		}
	}
	if len(startEvs) != 1 {
		t.Fatalf("ToolStart count = %d, want 1", len(startEvs))
	}
	if string(startEvs[0].Arguments()) != modifiedArgs {
		t.Errorf("ToolStart.Arguments() = %q, want %q (modify-input transparency, R-APP-006)",
			startEvs[0].Arguments(), modifiedArgs)
	}

	// Tool ran with modified args.
	select {
	case got := <-receivedArgsCh:
		if string(got) != modifiedArgs {
			t.Errorf("tool received args = %q, want %q (the substituted bytes)", got, modifiedArgs)
		}
	case <-time.After(time.Second):
		t.Fatal("tool did not receive any args within 1s")
	}

	// Result is success.
	if len(results) != 1 {
		t.Fatalf("Schedule returned %d results, want 1", len(results))
	}
	if results[0].Outcome != agent.ToolOutcomeSuccess {
		t.Errorf("results[0].Outcome = %v, want Success", results[0].Outcome)
	}
}

// S-APP-008 (AG-10.2 AllowAlways, AG-10.4 emission) — AllowAlways
// emits decision_made{AllowAlways}. The `permission_resolution_remembered`
// emission depends on policy.Remember's boolean return — wired in
// Commit 6 (R-APP-007). For Commit 4, the assertion is the
// decision_made event alone.
//
// RED-recorded at Commit 4 (AG-10.1 stubbed AllowAlways to no-event).
// GREEN at this commit (the AllowAlways emit).
func TestPermission_FourOutcomes_AllowAlways_EmitsDecisionMade(t *testing.T) {
	t.Parallel()

	policy := &scriptedPermissionPolicy{
		resolve: func(_ context.Context, _ ai.ToolCall) agent.PermissionVerdict {
			return agent.PermissionVerdict{Outcome: agent.PermissionOutcomeAllowAlways}
		},
		remember: func(_ context.Context, _ string, _ agent.PermissionOutcome) bool {
			return false // Commit 6 wires the resolution_remembered emission.
		},
	}

	tool := EchoScriptedTool("remember_tool", agent.EffectClassRead)
	reg := newMapRegistry(map[string]agent.Tool{"remember_tool": tool})

	calls := []scheduledPermissionCall{
		{id: "always_0", name: "remember_tool", args: []byte(`{}`)},
	}

	results, events := runPermissionSchedulerAndCollect(
		&agent.Scheduler{MaxConcurrentReads: 4}, calls, reg, policy,
	)

	// One decision_made{AllowAlways}.
	madeEvs := []agent.PermissionDecisionMade{}
	for _, ev := range events {
		if made, ok := ev.PermissionDecisionMade(); ok {
			madeEvs = append(madeEvs, made)
		}
	}
	if len(madeEvs) != 1 {
		t.Fatalf("decision_made count = %d, want 1", len(madeEvs))
	}
	if madeEvs[0].Outcome() != agent.PermissionOutcomeAllowAlways {
		t.Errorf("decision_made outcome = %v, want AllowAlways", madeEvs[0].Outcome())
	}

	// Tool ran and result is success.
	if inv := tool.Invocations(); inv != 1 {
		t.Errorf("remember_tool invocations = %d, want 1", inv)
	}
	if len(results) != 1 {
		t.Fatalf("Schedule returned %d results, want 1", len(results))
	}
	if results[0].Outcome != agent.ToolOutcomeSuccess {
		t.Errorf("results[0].Outcome = %v, want Success", results[0].Outcome)
	}
}

// --- AG-10.3 sibling isolation + cancellation tests ------------------

// S-APP-009 (AG-10.3 sibling isolation) — a parked call MUST NOT
// block sibling calls (R-APP-008). The parked call's goroutine
// holds its semaphore/serialize slot, but other goroutines can
// still acquire other slots and execute. The sibling reaches its
// result slot in call order regardless of the parked call's state.
//
// RED-recorded at Commit 5 (the parked-call goroutine holds the
// read-semaphore slot until wake; sibling reads can still proceed
// because the semaphore is bounded, not exclusive). GREEN at this
// commit.
//
// Defect 8/9 fix: cancel() now runs CONCURRENTLY with the park (as
// soon as decision_required is observed on sink), not after Schedule
// has already returned — the previous version called Schedule
// synchronously with nothing else able to release the park, so it
// always burned the full 2-second ctx deadline before "cancelling"
// anything.
func TestPermission_SuspensionDoesNotBlock_SiblingReachesOrdinalSlot(t *testing.T) {
	t.Parallel()

	// The parked call uses a tool that blocks on a channel — it
	// only proceeds when woken. The sibling has a different tool
	// that returns success. We expect: parked call never invokes
	// its tool; sibling invokes its tool exactly once and reaches
	// success; Schedule returns both results in call order.
	parkRelease := make(chan struct{})
	defer close(parkRelease) // safety net for the parked tool
	parkedTool := BlockingScriptedTool("parked_tool", agent.EffectClassRead, parkRelease)
	siblingTool := EchoScriptedTool("sibling_tool", agent.EffectClassRead)

	policy := &scriptedPermissionPolicy{
		resolve: func(_ context.Context, call ai.ToolCall) agent.PermissionVerdict {
			if call.Name() == "parked_tool" {
				return agent.PermissionVerdict{Outcome: agent.PermissionDefer}
			}
			return agent.PermissionVerdict{Outcome: agent.PermissionOutcomeAllowOnce}
		},
	}

	reg := newMapRegistry(map[string]agent.Tool{
		"parked_tool": parkedTool,
		"sibling_tool": siblingTool,
	})

	calls := []scheduledPermissionCall{
		{id: "parked_0", name: "parked_tool", args: []byte(`{}`)},
		{id: "sibling_1", name: "sibling_tool", args: []byte(`{}`)},
	}

	// ctx's own 2s deadline is now a pure safety net — the real
	// release path below is cancel() fired the instant
	// decision_required is observed, not the deadline itself.
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	stamper := &agent.LaneStamper{}
	sink := make(chan *agent.Event, 64)
	ready, drainDone, drainedPtr := drainUntilNDecisionsRequired(sink, 1)

	doneCh := make(chan []agent.Result, 1)
	go func() {
		doneCh <- (&agent.Scheduler{MaxConcurrentReads: 4}).Schedule(
			ctx,
			permissionCallsToAICalls(calls),
			reg,
			"run-sibling",
			"turn-sibling",
			policy,
			stamper,
			sink,
		)
	}()

	select {
	case <-ready:
	case <-time.After(2 * time.Second):
		t.Fatal("decision_required for parked_0 not observed within 2s")
	}
	// Cancel concurrently with the park — the parked call observes
	// ctx.Done() in the AG-10.3 path (R-APP-009).
	cancel()

	var results []agent.Result
	select {
	case results = <-doneCh:
	case <-time.After(2 * time.Second):
		t.Fatal("Schedule did not return within 2s after cancel")
	}
	<-drainDone
	drained := *drainedPtr

	// Two results, in call order.
	if len(results) != 2 {
		t.Fatalf("Schedule returned %d results, want 2 (parked + sibling)", len(results))
	}
	// The parked call's ordinal slot is a typed abort failure
	// (AG-10.3 R-APP-009: cancellation discipline).
	if results[0].Outcome != agent.ToolOutcomeExecutionFailure {
		t.Errorf("parked result.Outcome = %v, want ExecutionFailure (AG-10.3 cancellation path)",
			results[0].Outcome)
	}
	if results[0].Failure == nil {
		t.Errorf("parked result.Failure = nil, want typed *Failure")
	}
	// The sibling call succeeded (R-APP-008 sibling isolation).
	if results[1].Outcome != agent.ToolOutcomeSuccess {
		t.Errorf("sibling result.Outcome = %v, want Success (R-APP-008 sibling isolation)",
			results[1].Outcome)
	}
	// The parked call's tool was never invoked (parking blocked it).
	if inv := parkedTool.Invocations(); inv != 0 {
		t.Errorf("parked_tool invocations = %d, want 0 (parked before wake)", inv)
	}
	// The sibling's tool was invoked exactly once.
	if inv := siblingTool.Invocations(); inv != 1 {
		t.Errorf("sibling_tool invocations = %d, want 1", inv)
	}
	// decision_required for the parked call, decision_made{AllowOnce}
	// for the sibling — but no decision_made for the parked call
	// (the parked call never wakes).
	if got := countByKind(drained)[agent.EventKindPermissionDecisionRequired]; got != 1 {
		t.Errorf("decision_required count = %d, want 1 (one parked call)", got)
	}
}

// S-APP-010 (AG-10.3 cancellation wind-down) — on ctx cancel mid-
// park, BOTH parked calls observe ctx.Done() and write typed
// ExecutionFailure{aborted} into their ordinal slots. The rejoin
// slice is fully populated; Schedule returns without leaking
// goroutines (R-APP-009).
//
// RED-recorded at Commit 5. GREEN at this commit.
//
// Defect 8/9 fix: cancel() now runs CONCURRENTLY with the park (as
// soon as both decision_required events are observed on sink), not
// after Schedule has already returned — the previous version called
// Schedule synchronously with nothing else able to release the
// parked calls, so it always burned the full 2-second ctx deadline.
// This version also does NOT call t.Parallel(): the goroutine-leak
// assertion below needs to run in isolation from sibling tests'
// transient goroutines (Go only releases t.Parallel() tests to run
// concurrently once every top-level test in the binary has started,
// so a serial test's body runs before any of them do — the same
// reasoning documented at awaitGoroutineBaseline / the R-TLS-008
// source-guard test). AG-09 dropped an equivalent check for being
// racy against exactly that t.Parallel() interference
// (scheduler_test.go:439, :1046); this test adds it back correctly.
func TestPermission_CancellationMidPark_PopulatesRejoinAndNoLeak(t *testing.T) {
	runtime.GC()
	baseline, _ := awaitGoroutineBaseline(runtime.NumGoroutine(), 200*time.Millisecond)

	// Two parked calls; both block on per-call channels that
	// nothing will close (the test cancels ctx to unblock them).
	parkedA := BlockingScriptedTool("parked_a", agent.EffectClassRead, make(chan struct{}))
	parkedB := BlockingScriptedTool("parked_b", agent.EffectClassRead, make(chan struct{}))

	policy := &scriptedPermissionPolicy{
		resolve: func(_ context.Context, _ ai.ToolCall) agent.PermissionVerdict {
			return agent.PermissionVerdict{Outcome: agent.PermissionDefer}
		},
	}

	reg := newMapRegistry(map[string]agent.Tool{
		"parked_a": parkedA,
		"parked_b": parkedB,
	})

	calls := []scheduledPermissionCall{
		{id: "parked_a_0", name: "parked_a", args: []byte(`{}`)},
		{id: "parked_b_1", name: "parked_b", args: []byte(`{}`)},
	}

	// ctx's own 2s deadline is now a pure safety net.
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	stamper := &agent.LaneStamper{}
	sink := make(chan *agent.Event, 64)
	ready, drainDone, drainedPtr := drainUntilNDecisionsRequired(sink, 2)

	doneCh := make(chan []agent.Result, 1)
	go func() {
		doneCh <- (&agent.Scheduler{MaxConcurrentReads: 4}).Schedule(
			ctx,
			permissionCallsToAICalls(calls),
			reg,
			"run-cancel",
			"turn-cancel",
			policy,
			stamper,
			sink,
		)
	}()

	select {
	case <-ready:
	case <-time.After(2 * time.Second):
		t.Fatal("did not observe 2 decision_required events within 2s")
	}
	// Cancel concurrently with both parks.
	cancel()

	var results []agent.Result
	select {
	case results = <-doneCh:
	case <-time.After(2 * time.Second):
		t.Fatal("Schedule did not return within 2s after cancel")
	}
	<-drainDone
	drained := *drainedPtr

	// Two results, both typed ExecutionFailure (AG-10.3 R-APP-009).
	if len(results) != 2 {
		t.Fatalf("Schedule returned %d results, want 2 (full rejoin even on cancel)", len(results))
	}
	for i := range results {
		if results[i].Outcome != agent.ToolOutcomeExecutionFailure {
			t.Errorf("results[%d].Outcome = %v, want ExecutionFailure (AG-10.3 cancellation path)",
				i, results[i].Outcome)
		}
		if results[i].Failure == nil {
			t.Errorf("results[%d].Failure = nil, want typed *Failure", i)
		}
	}

	// Two decision_required events (one per parked call); zero
	// decision_made events (no wake before cancel).
	if got := countByKind(drained)[agent.EventKindPermissionDecisionRequired]; got != 2 {
		t.Errorf("decision_required count = %d, want 2 (one per parked call)", got)
	}
	if got := countByKind(drained)[agent.EventKindPermissionDecisionMade]; got != 0 {
		t.Errorf("decision_made count = %d, want 0 (no wake before cancel)", got)
	}

	// No tool invocations (parked calls never reached Tool.Run).
	if inv := parkedA.Invocations(); inv != 0 {
		t.Errorf("parked_a invocations = %d, want 0", inv)
	}
	if inv := parkedB.Invocations(); inv != 0 {
		t.Errorf("parked_b invocations = %d, want 0", inv)
	}

	// No goroutine leak (R-APP-009's "no task waits forever" —
	// defect 9's missing assertion): both parked call goroutines and
	// the dispatcher have exited by the time doneCh/drainDone fired
	// above; settle-then-sample, not a single racy snapshot.
	if last, settled := awaitGoroutineBaseline(baseline, 2*time.Second); !settled {
		t.Errorf("goroutine count did not settle back to baseline %d within 2s (last observed %d) — possible leak", baseline, last)
	}
}

// --- D-A wake-path tests ---------------------------------------------

// wakeParkedWithRetry calls sched.WakeParked(callID) until it
// succeeds or timeout elapses, failing the test if it never does.
//
// A single immediate attempt right after observing decision_required
// on sink can race: the ack that unblocks the decision_required
// sender's own goroutine (R-APP-002/D4) only proves the EVENT
// reached sink — it says nothing about whether that same goroutine
// has since had a scheduling turn to actually execute
// parked.park(callID) afterward. Retrying absorbs that ordinary
// goroutine-scheduling latency without weakening the ordering
// guarantee itself, which TestPermission_DeferEmitsBeforePark proves
// directly (with no retry: WakeParked must never succeed before the
// event is read at all).
func wakeParkedWithRetry(t *testing.T, sched *agent.Scheduler, callID string, timeout time.Duration) {
	t.Helper()
	deadline := time.Now().Add(timeout)
	var err error
	for time.Now().Before(deadline) {
		err = sched.WakeParked(callID)
		if err == nil {
			return
		}
		time.Sleep(time.Millisecond)
	}
	t.Fatalf("WakeParked(%q) = %v after retrying for %s, want nil", callID, err, timeout)
}

// drainUntilDecisionRequired starts a goroutine draining sink into
// the returned drainedEvents, closing readyCh the first time a
// permission_decision_required event for wantCallID is observed
// (wantCallID == "" matches the first decision_required of any
// call). doneCh closes once sink itself closes (drain complete).
//
// Callers use readyCh to synchronize a concurrent WakeParked/cancel
// with the park — NEVER after Schedule has already returned (that
// anti-pattern turns a "wake" or "cancel" test into a "deadline
// expired" test; defects 8/9 fixed this exact bug in three existing
// tests). Callers must only read *drainedEvents after receiving from
// doneCh (the goroutine still appends to it until sink closes).
func drainUntilDecisionRequired(sink <-chan *agent.Event, wantCallID string) (readyCh <-chan struct{}, doneCh <-chan struct{}, drainedEvents *[]*agent.Event) {
	events := make([]*agent.Event, 0, 16)
	ready := make(chan struct{})
	done := make(chan struct{})
	var readyOnce sync.Once
	go func() {
		defer close(done)
		for ev := range sink {
			events = append(events, ev)
			if req, ok := ev.PermissionDecisionRequired(); ok {
				if wantCallID == "" || req.CallID() == wantCallID {
					readyOnce.Do(func() { close(ready) })
				}
			}
		}
	}()
	return ready, done, &events
}

// D-A property 1 — waking the call parked under callID resumes it:
// the parked goroutine's select unblocks, re-enters policy.Resolve
// (design data flow "wake: re-evaluate verdict"), and the call
// completes using the fresh verdict.
//
// RED at write time: Scheduler.WakeParked does not exist yet —
// compile-error RED (the established convention in this file,
// AG-05 W1 defense).
func TestPermission_WakeParked_ResumesAndCompletes(t *testing.T) {
	t.Parallel()

	var resolveN atomic.Int64
	policy := &scriptedPermissionPolicy{
		resolve: func(_ context.Context, _ ai.ToolCall) agent.PermissionVerdict {
			if resolveN.Add(1) == 1 {
				return agent.PermissionVerdict{Outcome: agent.PermissionDefer}
			}
			// Second Resolve is the wake's re-evaluation (D-A).
			return agent.PermissionVerdict{Outcome: agent.PermissionOutcomeAllowOnce}
		},
	}

	tool := EchoScriptedTool("waking_tool", agent.EffectClassRead)
	reg := newMapRegistry(map[string]agent.Tool{"waking_tool": tool})
	calls := []scheduledPermissionCall{{id: "wake_0", name: "waking_tool", args: []byte(`{}`)}}

	sched := &agent.Scheduler{MaxConcurrentReads: 4}
	stamper := &agent.LaneStamper{}
	sink := make(chan *agent.Event, 64)
	ready, drainDone, drained := drainUntilDecisionRequired(sink, "wake_0")

	doneCh := make(chan []agent.Result, 1)
	go func() {
		doneCh <- sched.Schedule(context.Background(), permissionCallsToAICalls(calls), reg, "run-wake", "turn-wake", policy, stamper, sink)
	}()

	select {
	case <-ready:
	case <-time.After(2 * time.Second):
		t.Fatal("decision_required for wake_0 not observed within 2s")
	}

	wakeParkedWithRetry(t, sched, "wake_0", time.Second)

	var results []agent.Result
	select {
	case results = <-doneCh:
	case <-time.After(2 * time.Second):
		t.Fatal("Schedule did not return within 2s after WakeParked")
	}
	<-drainDone

	if len(results) != 1 {
		t.Fatalf("Schedule returned %d results, want 1", len(results))
	}
	if results[0].Outcome != agent.ToolOutcomeSuccess {
		t.Errorf("results[0].Outcome = %v, want Success (the woken call resumed and completed)", results[0].Outcome)
	}
	if inv := tool.Invocations(); inv != 1 {
		t.Errorf("waking_tool invocations = %d, want 1 (wake resumed the call)", inv)
	}
	if n := resolveN.Load(); n != 2 {
		t.Errorf("policy.Resolve invocation count = %d, want 2 (wake re-enters Resolve and processes the fresh verdict, design data flow)", n)
	}
	madeAllowOnce := 0
	for _, ev := range *drained {
		if made, ok := ev.PermissionDecisionMade(); ok && made.CallID() == "wake_0" && made.Outcome() == agent.PermissionOutcomeAllowOnce {
			madeAllowOnce++
		}
	}
	if madeAllowOnce != 1 {
		t.Errorf("decision_made{AllowOnce} for wake_0 count = %d, want 1 (the fresh verdict from the re-evaluation)", madeAllowOnce)
	}
}

// D-A property 2 — waking an unknown callID is a typed rejection
// AND touches no parked call (S-APP-004's second clause: the
// original bite never asserted this half). A genuinely parked
// sibling call is proven untouched — still parked, tool not
// invoked — before it is correctly woken so the test does not leak
// a goroutine.
func TestPermission_WakeParked_UnknownCallID_TypedRejection_NoTouch(t *testing.T) {
	t.Parallel()

	releaseTool := make(chan struct{})
	parkedTool := BlockingScriptedTool("parked_tool", agent.EffectClassRead, releaseTool)

	var resolveN atomic.Int64
	policy := &scriptedPermissionPolicy{
		resolve: func(_ context.Context, _ ai.ToolCall) agent.PermissionVerdict {
			if resolveN.Add(1) == 1 {
				return agent.PermissionVerdict{Outcome: agent.PermissionDefer}
			}
			return agent.PermissionVerdict{Outcome: agent.PermissionOutcomeAllowOnce}
		},
	}
	reg := newMapRegistry(map[string]agent.Tool{"parked_tool": parkedTool})
	calls := []scheduledPermissionCall{{id: "parked_real_0", name: "parked_tool", args: []byte(`{}`)}}

	sched := &agent.Scheduler{MaxConcurrentReads: 4}
	stamper := &agent.LaneStamper{}
	sink := make(chan *agent.Event, 64)
	ready, drainDone, _ := drainUntilDecisionRequired(sink, "parked_real_0")

	doneCh := make(chan []agent.Result, 1)
	go func() {
		doneCh <- sched.Schedule(context.Background(), permissionCallsToAICalls(calls), reg, "run-stray", "turn-stray", policy, stamper, sink)
	}()

	select {
	case <-ready:
	case <-time.After(2 * time.Second):
		t.Fatal("decision_required for parked_real_0 not observed within 2s")
	}

	// Wake an unrelated, unknown callID. Must be rejected AND must
	// not touch the genuinely parked call.
	err := sched.WakeParked("call-id-X")
	if err == nil {
		t.Fatalf("WakeParked(%q) = nil, want a non-nil typed error", "call-id-X")
	}
	if !errors.Is(err, agent.ErrStrayDecision) {
		t.Errorf("WakeParked(%q) = %v, want errors.Is to match agent.ErrStrayDecision", "call-id-X", err)
	}
	if inv := parkedTool.Invocations(); inv != 0 {
		t.Errorf("parked_tool invocations = %d, want 0 (a stray wake to an unrelated callID must not touch the genuinely parked call, S-APP-004)", inv)
	}

	// Now wake the real one so the goroutine finishes cleanly.
	close(releaseTool)
	wakeParkedWithRetry(t, sched, "parked_real_0", time.Second)

	select {
	case results := <-doneCh:
		if len(results) != 1 {
			t.Fatalf("Schedule returned %d results, want 1", len(results))
		}
		if results[0].Outcome != agent.ToolOutcomeSuccess {
			t.Errorf("results[0].Outcome = %v, want Success (the real call still resumes after the stray wake)", results[0].Outcome)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("Schedule did not return within 2s after waking the real parked call")
	}
	<-drainDone
}

// --- D-B shutdown-ordering approval test -----------------------------

// D-B — park release has exactly TWO production paths: WakeParked
// (close by callID) and ctx.Done(). This test drives a context with
// NO deadline at all (plain context.Background(), no WithTimeout /
// WithCancel) — the ONLY way Schedule can possibly return is the
// explicit WakeParked call below. If Schedule ever came to depend on
// a defensive sweep at Schedule exit (a wg.Wait()-then-closeAll
// ordering can never fire — a parked goroutine never calls wg.Done()
// until it is released, so by the time such a sweep would run, every
// goroutine has already exited through one of the two real paths),
// this test would hang and fail on the deadline below.
//
// Approval test (strict-tdd.md "Approval Testing for refactoring"):
// written to capture and freeze this already-correct contract BEFORE
// the D-B cleanup deletes the unreachable closeAll sweep, the dead
// wakeErr-non-nil branch, and errParkedSetShutdown — it passes
// identically before and after, proving the deletion changed no
// observable behavior.
func TestPermission_WakeParked_SchedulerReturnsAfterExplicitWake_NoDeadline(t *testing.T) {
	t.Parallel()

	var resolveN atomic.Int64
	policy := &scriptedPermissionPolicy{
		resolve: func(_ context.Context, _ ai.ToolCall) agent.PermissionVerdict {
			if resolveN.Add(1) == 1 {
				return agent.PermissionVerdict{Outcome: agent.PermissionDefer}
			}
			return agent.PermissionVerdict{Outcome: agent.PermissionOutcomeAllowOnce}
		},
	}

	tool := EchoScriptedTool("no_deadline_tool", agent.EffectClassRead)
	reg := newMapRegistry(map[string]agent.Tool{"no_deadline_tool": tool})
	calls := []scheduledPermissionCall{{id: "no_deadline_0", name: "no_deadline_tool", args: []byte(`{}`)}}

	sched := &agent.Scheduler{MaxConcurrentReads: 4}
	stamper := &agent.LaneStamper{}
	sink := make(chan *agent.Event, 64)
	ready, drainDone, _ := drainUntilDecisionRequired(sink, "no_deadline_0")

	doneCh := make(chan []agent.Result, 1)
	go func() {
		// context.Background(): no deadline, no cancel. Nothing but
		// an explicit WakeParked can ever unblock this call.
		doneCh <- sched.Schedule(context.Background(), permissionCallsToAICalls(calls), reg, "run-no-deadline", "turn-no-deadline", policy, stamper, sink)
	}()

	select {
	case <-ready:
	case <-time.After(2 * time.Second):
		t.Fatal("decision_required for no_deadline_0 not observed within 2s")
	}

	wakeParkedWithRetry(t, sched, "no_deadline_0", time.Second)

	select {
	case results := <-doneCh:
		if len(results) != 1 {
			t.Fatalf("Schedule returned %d results, want 1", len(results))
		}
		if results[0].Outcome != agent.ToolOutcomeSuccess {
			t.Errorf("results[0].Outcome = %v, want Success", results[0].Outcome)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("Schedule did not return within 2s on a deadline-less context — the only release path (WakeParked) was called explicitly above")
	}
	<-drainDone
	if inv := tool.Invocations(); inv != 1 {
		t.Errorf("no_deadline_tool invocations = %d, want 1", inv)
	}
}

// --- AG-10.4 Remember tests -------------------------------------------

// S-APP-008 — AllowAlways MUST invoke Policy.Remember (R-APP-007);
// a true return emits exactly one permission_resolution_remembered,
// a false return suppresses it. Both branches still execute the
// call — Remember gates the EVENT, not execution.
func TestPermission_AllowAlways_Remember_Branches(t *testing.T) {
	t.Parallel()

	type testCase struct {
		name           string
		remember       bool
		wantRemembered int
	}
	cases := []testCase{
		{name: "Remember=true emits resolution_remembered", remember: true, wantRemembered: 1},
		{name: "Remember=false suppresses resolution_remembered", remember: false, wantRemembered: 0},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			var rememberCalledWith agent.PermissionOutcome
			var rememberCalledName string
			policy := &scriptedPermissionPolicy{
				resolve: func(_ context.Context, _ ai.ToolCall) agent.PermissionVerdict {
					return agent.PermissionVerdict{Outcome: agent.PermissionOutcomeAllowAlways}
				},
				remember: func(_ context.Context, toolName string, outcome agent.PermissionOutcome) bool {
					rememberCalledName = toolName
					rememberCalledWith = outcome
					return tc.remember
				},
			}

			tool := EchoScriptedTool("remember_branch_tool", agent.EffectClassRead)
			reg := newMapRegistry(map[string]agent.Tool{"remember_branch_tool": tool})
			calls := []scheduledPermissionCall{{id: "remember_0", name: "remember_branch_tool", args: []byte(`{}`)}}

			results, events := runPermissionSchedulerAndCollect(
				&agent.Scheduler{MaxConcurrentReads: 4}, calls, reg, policy,
			)

			if n := policy.rememberInvocations(); n != 1 {
				t.Fatalf("policy.Remember invocation count = %d, want 1 (R-APP-007 MUST invoke Remember on AllowAlways)", n)
			}
			if rememberCalledName != "remember_branch_tool" {
				t.Errorf("Remember called with toolName = %q, want %q", rememberCalledName, "remember_branch_tool")
			}
			if rememberCalledWith != agent.PermissionOutcomeAllowAlways {
				t.Errorf("Remember called with outcome = %v, want AllowAlways", rememberCalledWith)
			}

			rememberedEvs := 0
			for _, ev := range events {
				if rem, ok := ev.PermissionResolutionRemembered(); ok {
					rememberedEvs++
					if rem.ToolName() != "remember_branch_tool" {
						t.Errorf("resolution_remembered.ToolName() = %q, want %q", rem.ToolName(), "remember_branch_tool")
					}
					if rem.Outcome() != agent.PermissionOutcomeAllowAlways {
						t.Errorf("resolution_remembered.Outcome() = %v, want AllowAlways", rem.Outcome())
					}
				}
			}
			if rememberedEvs != tc.wantRemembered {
				t.Errorf("resolution_remembered count = %d, want %d", rememberedEvs, tc.wantRemembered)
			}

			// Remember gates the EVENT, not execution — the call
			// runs in both branches.
			if len(results) != 1 {
				t.Fatalf("Schedule returned %d results, want 1", len(results))
			}
			if results[0].Outcome != agent.ToolOutcomeSuccess {
				t.Errorf("results[0].Outcome = %v, want Success (Remember gates the event, not execution)", results[0].Outcome)
			}
			if inv := tool.Invocations(); inv != 1 {
				t.Errorf("remember_branch_tool invocations = %d, want 1", inv)
			}
		})
	}
}

// S-APP-011 — a remembered AllowAlways (Remember == true) silences
// asking for identical subsequent calls in the SAME Schedule call
// (R-APP-010): the second call to the same tool name is NOT
// consulted at all (policy.Resolve is not invoked a second time for
// it), while a call to a DIFFERENT tool name is still consulted
// normally (the suppression is scoped per tool name, not global).
//
// The three calls are all EffectClassMutating (serialized, call-
// order sequential) so the suppression state written after call_0
// is guaranteed visible before call_1 starts — a concurrent
// read-class scheduling of two identical-tool-name calls has a
// genuine TOCTOU race between the pre-Resolve suppression check and
// the post-Remember state write that this test deliberately avoids
// by using the serialized lane (documented as a discovered risk in
// the apply-progress artifact, not fixed here — R-APP-010 only
// requires suppressing calls that occur temporally after the
// remembering takes effect).
func TestPermission_RememberedSuppressesSubsequentAsk(t *testing.T) {
	t.Parallel()

	policy := &scriptedPermissionPolicy{
		resolve: func(_ context.Context, _ ai.ToolCall) agent.PermissionVerdict {
			return agent.PermissionVerdict{Outcome: agent.PermissionOutcomeAllowAlways}
		},
		remember: func(_ context.Context, _ string, _ agent.PermissionOutcome) bool {
			return true
		},
	}

	toolA := EchoScriptedTool("remembered_tool", agent.EffectClassMutating)
	toolB := EchoScriptedTool("other_tool", agent.EffectClassMutating)
	reg := newMapRegistry(map[string]agent.Tool{
		"remembered_tool": toolA,
		"other_tool":      toolB,
	})

	calls := []scheduledPermissionCall{
		{id: "first_0", name: "remembered_tool", args: []byte(`{}`)},
		{id: "second_1", name: "remembered_tool", args: []byte(`{}`)},
		{id: "third_2", name: "other_tool", args: []byte(`{}`)},
	}

	results, events := runPermissionSchedulerAndCollect(
		&agent.Scheduler{MaxConcurrentReads: 4}, calls, reg, policy,
	)

	// Resolve was consulted exactly twice: once for the first
	// remembered_tool call (which remembers it), once for
	// other_tool (a different tool name, never suppressed). The
	// second remembered_tool call is NOT consulted at all.
	if n := policy.resolveInvocations(); n != 2 {
		t.Errorf("policy.Resolve invocation count = %d, want 2 (the second identical call must not be consulted, R-APP-010)", n)
	}
	seenNames := policy.seenResolveToolNames()
	for _, name := range seenNames {
		if name != "remembered_tool" && name != "other_tool" {
			t.Errorf("policy.Resolve seen an unexpected tool name %q", name)
		}
	}
	remCount := 0
	for _, name := range seenNames {
		if name == "remembered_tool" {
			remCount++
		}
	}
	if remCount != 1 {
		t.Errorf("policy.Resolve saw remembered_tool %d time(s), want 1 (the suppressed second call skips Resolve entirely)", remCount)
	}

	// Remember invoked exactly twice — once per DISTINCT tool name
	// that reaches an AllowAlways verdict (remembered_tool's first
	// call, and other_tool, which is never suppressed since it is a
	// different tool name). The suppressed second remembered_tool
	// call contributes zero additional Remember invocations.
	if n := policy.rememberInvocations(); n != 2 {
		t.Errorf("policy.Remember invocation count = %d, want 2 (one per distinct tool name reaching AllowAlways)", n)
	}

	// Exactly one resolution_remembered PER distinct tool name —
	// never two for the same tool name (CardinalityAtMostOne
	// preserved by construction, not merely by the validator,
	// S-APE-082 carry).
	rememberedByTool := map[string]int{}
	for _, ev := range events {
		if rem, ok := ev.PermissionResolutionRemembered(); ok {
			rememberedByTool[rem.ToolName()]++
		}
	}
	if len(rememberedByTool) != 2 {
		t.Errorf("resolution_remembered distinct tool names = %d, want 2, got %v", len(rememberedByTool), rememberedByTool)
	}
	if rememberedByTool["remembered_tool"] != 1 {
		t.Errorf("resolution_remembered count for remembered_tool = %d, want exactly 1 (the suppressed second call must not re-trigger it)", rememberedByTool["remembered_tool"])
	}
	if rememberedByTool["other_tool"] != 1 {
		t.Errorf("resolution_remembered count for other_tool = %d, want exactly 1", rememberedByTool["other_tool"])
	}

	// Zero permission_decision_required — no call ever defers in
	// this scenario.
	if got := countByKind(events)[agent.EventKindPermissionDecisionRequired]; got != 0 {
		t.Errorf("decision_required count = %d, want 0", got)
	}

	// All three calls executed successfully — suppression bypasses
	// asking, not execution.
	if len(results) != 3 {
		t.Fatalf("Schedule returned %d results, want 3", len(results))
	}
	for i, r := range results {
		if r.Outcome != agent.ToolOutcomeSuccess {
			t.Errorf("results[%d].Outcome = %v, want Success", i, r.Outcome)
		}
	}
	if inv := toolA.Invocations(); inv != 2 {
		t.Errorf("remembered_tool invocations = %d, want 2 (both calls still execute)", inv)
	}
	if inv := toolB.Invocations(); inv != 1 {
		t.Errorf("other_tool invocations = %d, want 1", inv)
	}
}

// --- Task 4.3 — R-TLS-008 source guard --------------------------------

// drainUntilNDecisionsRequired starts a goroutine draining sink into
// the returned drainedEvents, closing readyCh once at least n
// distinct permission_decision_required events have been observed.
// Mirrors drainUntilDecisionRequired's synchronization discipline
// (readers must only inspect *drainedEvents after doneCh closes).
func drainUntilNDecisionsRequired(sink <-chan *agent.Event, n int) (readyCh <-chan struct{}, doneCh <-chan struct{}, drainedEvents *[]*agent.Event) {
	events := make([]*agent.Event, 0, 32)
	ready := make(chan struct{})
	done := make(chan struct{})
	var readyOnce sync.Once
	go func() {
		defer close(done)
		seen := 0
		for ev := range sink {
			events = append(events, ev)
			if _, ok := ev.PermissionDecisionRequired(); ok {
				seen++
				if seen >= n {
					readyOnce.Do(func() { close(ready) })
				}
			}
		}
	}()
	return ready, done, &events
}

// awaitGoroutineBaseline polls runtime.NumGoroutine() until it
// settles at or below baseline, or timeout elapses. Deterministic
// under -race's extra scheduling overhead: goroutine teardown after
// a channel close / wg.Done() is not synchronous, so the caller must
// poll rather than sample once immediately (a prior milestone,
// AG-09, dropped a single-sample goroutine-baseline check for being
// racy against t.Parallel() siblings — see scheduler_test.go:439,
// :1046 — this helper's polling-until-settled approach, combined
// with the caller NOT calling t.Parallel(), is what makes the
// baseline in this file's R-TLS-008 test deterministic instead).
func awaitGoroutineBaseline(baseline int, timeout time.Duration) (last int, settled bool) {
	deadline := time.Now().Add(timeout)
	for {
		runtime.Gosched()
		last = runtime.NumGoroutine()
		if last <= baseline {
			return last, true
		}
		if time.Now().After(deadline) {
			return last, false
		}
		time.Sleep(5 * time.Millisecond)
	}
}

// R-TLS-008 source guard (task 4.3) — one Schedule call driving 5
// calls (3 Defer + 2 immediate) through all four typed outcomes
// (AllowOnce, AllowAlways, Deny, ModifyInput), asserting:
//
//  1. full ordinal-order rejoin — results[i] carries call i's
//     identity and outcome regardless of completion order or which
//     calls were parked;
//  2. correct per-slot outcomes for every one of the 5 calls;
//  3. no goroutine leak under -race (a settled runtime.NumGoroutine()
//     baseline, not a single racy sample).
//
// This test does NOT call t.Parallel() — deliberately, unlike every
// other test in this file. Go's test runner only releases
// t.Parallel() siblings to run concurrently once every top-level
// test in the binary has been started; a serial test therefore runs
// in true isolation from them (no sibling goroutine churn pollutes
// its NumGoroutine() samples). This is what makes the goroutine
// check below deterministic rather than the flaky pattern AG-09
// discovered and reverted.
func TestPermission_RTLS008_SourceGuard_MixedOutcomesFullRejoin(t *testing.T) {
	// runtime.GC() + a Gosched settle lets any goroutines left over
	// from earlier sequential tests in this run finish tearing down
	// before the baseline is captured.
	runtime.GC()
	baseline, _ := awaitGoroutineBaseline(runtime.NumGoroutine(), 200*time.Millisecond)

	denialInner, err := ai.MidStreamFailure(ai.FailureReport{Category: ai.FailureCategoryUnavailable}, false)
	if err != nil {
		t.Fatalf("ai.MidStreamFailure: %v", err)
	}
	denial, err := agent.NewFailure(denialInner)
	if err != nil {
		t.Fatalf("agent.NewFailure: %v", err)
	}
	const modifiedArgs3 = `{"slot":"3-modified"}`
	const modifiedArgs4 = `{"slot":"4-modified"}`

	type plan struct {
		first agent.PermissionVerdict
		wake  agent.PermissionVerdict // consulted only if first.Outcome == PermissionDefer
	}
	plans := map[string]plan{
		"rtls_0": {first: agent.PermissionVerdict{Outcome: agent.PermissionDefer},
			wake: agent.PermissionVerdict{Outcome: agent.PermissionOutcomeAllowOnce}},
		"rtls_1": {first: agent.PermissionVerdict{Outcome: agent.PermissionOutcomeAllowAlways}},
		"rtls_2": {first: agent.PermissionVerdict{Outcome: agent.PermissionDefer},
			wake: agent.PermissionVerdict{Outcome: agent.PermissionOutcomeDeny, Failure: denial}},
		"rtls_3": {first: agent.PermissionVerdict{Outcome: agent.PermissionOutcomeModifyInput, ModifiedArgs: []byte(modifiedArgs3)}},
		"rtls_4": {first: agent.PermissionVerdict{Outcome: agent.PermissionDefer},
			wake: agent.PermissionVerdict{Outcome: agent.PermissionOutcomeModifyInput, ModifiedArgs: []byte(modifiedArgs4)}},
	}
	attempts := map[string]*atomic.Int64{
		"rtls_0": {}, "rtls_1": {}, "rtls_2": {}, "rtls_3": {}, "rtls_4": {},
	}

	policy := &scriptedPermissionPolicy{
		resolve: func(_ context.Context, call ai.ToolCall) agent.PermissionVerdict {
			p := plans[call.ID()]
			if attempts[call.ID()].Add(1) == 1 {
				return p.first
			}
			return p.wake
		},
		remember: func(_ context.Context, _ string, _ agent.PermissionOutcome) bool { return false },
	}

	tool0 := EchoScriptedTool("rtls_tool_0", agent.EffectClassRead)
	tool1 := EchoScriptedTool("rtls_tool_1", agent.EffectClassRead)
	tool2 := EchoScriptedTool("rtls_tool_2", agent.EffectClassMutating)
	tool3 := EchoScriptedTool("rtls_tool_3", agent.EffectClassMutating)
	tool4 := EchoScriptedTool("rtls_tool_4", agent.EffectClassRead)
	reg := newMapRegistry(map[string]agent.Tool{
		"rtls_tool_0": tool0, "rtls_tool_1": tool1, "rtls_tool_2": tool2,
		"rtls_tool_3": tool3, "rtls_tool_4": tool4,
	})

	calls := []scheduledPermissionCall{
		{id: "rtls_0", name: "rtls_tool_0", args: []byte(`{}`)},
		{id: "rtls_1", name: "rtls_tool_1", args: []byte(`{}`)},
		{id: "rtls_2", name: "rtls_tool_2", args: []byte(`{}`)},
		{id: "rtls_3", name: "rtls_tool_3", args: []byte(`{"slot":"3-original"}`)},
		{id: "rtls_4", name: "rtls_tool_4", args: []byte(`{}`)},
	}

	sched := &agent.Scheduler{MaxConcurrentReads: 4}
	stamper := &agent.LaneStamper{}
	sink := make(chan *agent.Event, 128)
	ready, drainDone, drained := drainUntilNDecisionsRequired(sink, 3)

	doneCh := make(chan []agent.Result, 1)
	go func() {
		doneCh <- sched.Schedule(context.Background(), permissionCallsToAICalls(calls), reg, "run-rtls008", "turn-rtls008", policy, stamper, sink)
	}()

	select {
	case <-ready:
	case <-time.After(2 * time.Second):
		t.Fatal("did not observe 3 decision_required events within 2s")
	}

	for _, id := range []string{"rtls_0", "rtls_2", "rtls_4"} {
		wakeParkedWithRetry(t, sched, id, time.Second)
	}

	var results []agent.Result
	select {
	case results = <-doneCh:
	case <-time.After(2 * time.Second):
		t.Fatal("Schedule did not return within 2s after waking all 3 parked calls")
	}
	<-drainDone
	_ = drained

	// (1) + (2): full ordinal-order rejoin, correct per-slot outcomes.
	if len(results) != 5 {
		t.Fatalf("Schedule returned %d results, want 5", len(results))
	}
	wantCallIDs := []string{"rtls_0", "rtls_1", "rtls_2", "rtls_3", "rtls_4"}
	for i, want := range wantCallIDs {
		if got := results[i].CallID(); got != want {
			t.Errorf("results[%d].CallID() = %q, want %q (ordinal-order rejoin)", i, got, want)
		}
	}
	wantOutcomes := []agent.ToolOutcome{
		agent.ToolOutcomeSuccess,          // rtls_0: Defer -> wake -> AllowOnce
		agent.ToolOutcomeSuccess,          // rtls_1: immediate AllowAlways
		agent.ToolOutcomeExecutionFailure, // rtls_2: Defer -> wake -> Deny
		agent.ToolOutcomeSuccess,          // rtls_3: immediate ModifyInput
		agent.ToolOutcomeSuccess,          // rtls_4: Defer -> wake -> ModifyInput
	}
	for i, want := range wantOutcomes {
		if results[i].Outcome != want {
			t.Errorf("results[%d].Outcome = %v, want %v", i, results[i].Outcome, want)
		}
	}
	if results[2].Failure == nil {
		t.Error("results[2].Failure = nil, want the typed denial (rtls_2, Deny via wake)")
	}

	// Each tool invoked exactly once (no double-execution, no
	// missed execution across the mixed defer/immediate paths).
	for name, tool := range map[string]*ScriptedTool{
		"rtls_tool_0": tool0, "rtls_tool_1": tool1, "rtls_tool_3": tool3, "rtls_tool_4": tool4,
	} {
		if inv := tool.Invocations(); inv != 1 {
			t.Errorf("%s invocations = %d, want 1", name, inv)
		}
	}
	if inv := tool2.Invocations(); inv != 0 {
		t.Errorf("rtls_tool_2 invocations = %d, want 0 (Deny skips execution)", inv)
	}

	// ModifyInput transparency for both the immediate (rtls_3) and
	// wake-resolved (rtls_4) ModifyInput calls: the tool received
	// exactly the modified bytes.
	if got := string(tool3.RecordedArgs()[0]); got != modifiedArgs3 {
		t.Errorf("rtls_tool_3 received args = %q, want %q", got, modifiedArgs3)
	}
	if got := string(tool4.RecordedArgs()[0]); got != modifiedArgs4 {
		t.Errorf("rtls_tool_4 received args = %q, want %q", got, modifiedArgs4)
	}

	// (3) no goroutine leak: settle-then-sample, not a single racy
	// snapshot. This test's own goroutines (the Schedule-launching
	// goroutine, the drain goroutine) have both signaled completion
	// above (doneCh, drainDone) before this check runs.
	if last, settled := awaitGoroutineBaseline(baseline, 2*time.Second); !settled {
		t.Errorf("goroutine count did not settle back to baseline %d within 2s (last observed %d) — possible leak", baseline, last)
	}
}

// --- Defect 6 — swallowed decision_made constructor errors -----------

// A ModifyInput verdict with empty ModifiedArgs makes
// NewPermissionDecisionMade fail construction (modified_arguments
// MUST be non-empty for that outcome). Before the fix, the gate
// silently swallowed the constructor error (`if err == nil { emit }`)
// and STILL let the call proceed — with the call's ORIGINAL,
// unmodified arguments (a nil modifiedArgs falls back to
// call.Arguments() in executeCall) and with NO decision_made event
// on the stream at all. That silently broke R-APP-006: nobody can
// tell from the stream what the policy actually decided, or that
// modify-input transparency failed, while the tool still ran. The
// fix treats a decision_made constructor failure as a typed
// execution failure — the call does not proceed.
func TestPermission_ModifyInput_ConstructorFailure_DoesNotSilentlyProceed(t *testing.T) {
	t.Parallel()

	policy := &scriptedPermissionPolicy{
		resolve: func(_ context.Context, _ ai.ToolCall) agent.PermissionVerdict {
			// Malformed verdict: ModifyInput without modified args.
			// NewPermissionDecisionMade rejects empty
			// modified_arguments for this outcome.
			return agent.PermissionVerdict{
				Outcome:      agent.PermissionOutcomeModifyInput,
				ModifiedArgs: nil,
			}
		},
	}

	tool := EchoScriptedTool("modify_bad", agent.EffectClassRead)
	reg := newMapRegistry(map[string]agent.Tool{"modify_bad": tool})
	calls := []scheduledPermissionCall{{id: "modify_bad_0", name: "modify_bad", args: []byte(`{}`)}}

	results, events := runPermissionSchedulerAndCollect(
		&agent.Scheduler{MaxConcurrentReads: 4}, calls, reg, policy,
	)

	if len(results) != 1 {
		t.Fatalf("Schedule returned %d results, want 1", len(results))
	}
	if results[0].Outcome != agent.ToolOutcomeExecutionFailure {
		t.Errorf("results[0].Outcome = %v, want ExecutionFailure (a decision_made constructor failure MUST NOT silently proceed, defect 6 fix)",
			results[0].Outcome)
	}
	if results[0].Failure == nil {
		t.Errorf("results[0].Failure = nil, want typed *Failure describing the constructor error")
	}
	if inv := tool.Invocations(); inv != 0 {
		t.Errorf("modify_bad invocations = %d, want 0 (the call must not silently execute when decision_made cannot be constructed)", inv)
	}
	for _, ev := range events {
		if _, ok := ev.PermissionDecisionMade(); ok {
			t.Errorf("decision_made event present despite a construction failure — the event must not be fabricated from a failed constructor")
		}
	}
}
