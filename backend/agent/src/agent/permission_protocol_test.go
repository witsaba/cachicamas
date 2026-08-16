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

// S-PPB-002 — RED bite (recorded at Commit 2). Given a policy that
// returns Defer for one call and AllowOnce for a sibling, when
// Schedule runs without an external wake (the parked call waits
// until test ends), then:
//
//   (a) ONE permission_decision_required event reaches the sink;
//   (b) ZERO permission_decision_made events reach the sink (no
//       wake happened — the policy hasn't decided yet);
//   (c) ZERO tool_start events for the deferred call (the call is
//       parked; sibling AllowOnce emits no decision_required, runs
//       to completion).
//
// RED at write time: same compile-error surface as S-PPB-001.
func TestPermission_DeferEmitsAndParks(t *testing.T) {
	t.Parallel()

	policy := &scriptedPermissionPolicy{
		resolve: func(_ context.Context, call ai.ToolCall) agent.PermissionVerdict {
			if call.ID() == "deferred_0" {
				return agent.PermissionVerdict{Outcome: agent.PermissionDefer}
			}
			return agent.PermissionVerdict{Outcome: agent.PermissionOutcomeAllowOnce}
		},
	}

	deferredTool := BlockingScriptedTool("deferred_tool", agent.EffectClassRead, make(chan struct{}))
	allowedTool := EchoScriptedTool("allowed_tool", agent.EffectClassRead)

	reg := newMapRegistry(map[string]agent.Tool{
		"deferred_tool": deferredTool,
		"allowed_tool":  allowedTool,
	})

	calls := []scheduledPermissionCall{
		{id: "deferred_0", name: "deferred_tool", args: []byte(`{}`)},
		{id: "allowed_1", name: "allowed_tool", args: []byte(`{}`)},
	}

	sink := make(chan *agent.Event, 64)
	// Bounded test: the parked call waits forever; bound by ctx
	// deadline. We assert the gate-emission happens before the
	// parked wait blocks siblings (sibling emits no decision).
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	stamper := &agent.LaneStamper{}
	results := (&agent.Scheduler{MaxConcurrentReads: 4}).Schedule(
		ctx,
		permissionCallsToAICalls(calls),
		reg,
		"run-defer",
		"turn-defer",
		policy,
		stamper,
		sink,
	)

	// Drain the events we have so far (sibling may emit ToolEnd
	// while the deferred call is parked).
	collected := []*agent.Event{}
	drainDone := make(chan struct{})
	go func() {
		defer close(drainDone)
		for ev := range sink {
			collected = append(collected, ev)
		}
	}()
	// Cancel the context to unblock the parked call (which observes
	// ctx.Done() in the AG-10.3 path) so the test does not hang.
	cancel()
	<-drainDone

	// (a) Exactly one decision_required — for the deferred call.
	gotReq := countByKind(collected)[agent.EventKindPermissionDecisionRequired]
	if gotReq != 1 {
		t.Errorf("decision_required count = %d, want 1 (deferring one call emits exactly one decision_required, R-APP-002)",
			gotReq)
	}

	// (b) Zero decision_made for the deferred call — the deferred
	// call has not been answered yet, so its decision_made never
	// reaches the stream. (The AllowOnce sibling DOES emit a
	// decision_made per Commit 4's R-APP-004 wiring — the bite
	// filters out the sibling's emission.)
	deferredMade := 0
	for _, ev := range collected {
		if made, ok := ev.PermissionDecisionMade(); ok && made.CallID() == "deferred_0" {
			deferredMade++
		}
	}
	if deferredMade != 0 {
		t.Errorf("decision_made for deferred_0 count = %d, want 0 (no wake during the bite — the deferred call waits, S-PPB-002)",
			deferredMade)
	}

	// (c) The deferred call never reached ToolStart (parked).
	deferredStarted := false
	for _, ev := range collected {
		if start, ok := ev.ToolStart(); ok && start.CallID() == "deferred_0" {
			deferredStarted = true
		}
	}
	if deferredStarted {
		t.Errorf("deferred call reached ToolStart before wake — the parked-set discipline (R-APP-002) is broken")
	}

	// Sanity: the AllowOnce sibling executed and the deferred
	// call's result slot is a typed abort failure (AG-10.3
	// cancellation path — the test cancels ctx to unblock the
	// parked wait).
	if len(results) != 2 {
		t.Errorf("Schedule returned %d results, want 2 (deferred + sibling)", len(results))
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

// S-PPB-004 — RED bite (recorded at Commit 2). The
// CardinalityAtMostOne rule on permission_resolution_remembered
// rejects a second event for the same toolName. The bite
// constructs a hand-built stream with two resolution_remembered
// events for "fs.write" and asserts that CheckStream rejects the
// second one at its 0-based slice index with ai.ErrDuplicate.
//
// RED at write time: nothing. This bite is runtime RED — it
// drives the validator against the documented descriptor
// (event.go:320-323) and asserts the rejection rule bites. The
// production code being defended is AG-10.4's single-emission
// discipline: any future scheduler implementation that emits two
// resolution_remembered events for the same toolName triggers
// the validator's rule, surfacing the bug at validation time.
//
// Mirrors S-APE-082 (permission_events_test.go:263-317) — AG-10
// re-asserts the seam from the apply-phase perspective.
func TestPermission_RememberedCardinality_SecondEmissionRejected(t *testing.T) {
	t.Parallel()

	runID := agent.RunID("run-app-082")
	turnID := agent.TurnID("turn-app-082")

	first, err := agent.NewPermissionResolutionRemembered(runID, "fs.write", agent.PermissionOutcomeAllowAlways)
	if err != nil {
		t.Fatalf("first NewPermissionResolutionRemembered: %v", err)
	}
	second, err := agent.NewPermissionResolutionRemembered(runID, "fs.write", agent.PermissionOutcomeAllowAlways)
	if err != nil {
		t.Fatalf("second NewPermissionResolutionRemembered: %v", err)
	}

	runStart, _ := agent.NewRunStart(runID)
	turnStart, _ := agent.NewTurnStart(runID, turnID)
	turnEnd, _ := agent.NewTurnEnd(runID, turnID, agent.TurnOutcomeFinished, nil)
	runEnd, _ := agent.NewRunEnd(runID, agent.RunOutcomeCompleted, nil)

	var lane agent.LaneStamper
	stream := []agent.Event{
		lane.Stamp(runStart),
		lane.Stamp(turnStart),
		lane.Stamp(first),
		lane.Stamp(second),
		lane.Stamp(turnEnd),
		lane.Stamp(runEnd),
	}

	report := agent.CheckStream(stream)
	if report.Violation() == nil {
		t.Fatalf("CheckStream accepted two permission_resolution_remembered events for the same tool name; CardinalityAtMostOne seam (R-APE-003, S-APE-082) MUST reject the second — AG-10.4's single-emission discipline defends this rule")
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

	// The parked call waits forever (parkRelease is closed only at
	// test end). The test cancels ctx to unblock Schedule exit so
	// the parked call aborts via the AG-10.3 cancellation path.
	// A timeout ensures the test fails loudly if the parked
	// call fails to observe ctx.Done() within the test's budget.
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	stamper := &agent.LaneStamper{}
	sink := make(chan *agent.Event, 64)
	// Drain goroutine MUST be set up BEFORE Schedule blocks; otherwise
	// the sink sits unread while the parked call waits, and the
	// dispatcher is stuck on its emissions send.
	drained := []*agent.Event{}
	drainDone := make(chan struct{})
	go func() {
		defer close(drainDone)
		for ev := range sink {
			drained = append(drained, ev)
		}
	}()
	results := (&agent.Scheduler{MaxConcurrentReads: 4}).Schedule(
		ctx,
		permissionCallsToAICalls(calls),
		reg,
		"run-sibling",
		"turn-sibling",
		policy,
		stamper,
		sink,
	)
	// Cancel ctx to unblock the parked call (AG-10.3 path) — the
	// timeout above also serves as a safety net.
	cancel()
	<-drainDone

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
func TestPermission_CancellationMidPark_PopulatesRejoinAndNoLeak(t *testing.T) {
	t.Parallel()

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

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	stamper := &agent.LaneStamper{}
	sink := make(chan *agent.Event, 64)
	// Drain goroutine MUST be set up BEFORE Schedule blocks.
	drained := []*agent.Event{}
	drainDone := make(chan struct{})
	go func() {
		defer close(drainDone)
		for ev := range sink {
			drained = append(drained, ev)
		}
	}()
	results := (&agent.Scheduler{MaxConcurrentReads: 4}).Schedule(
		ctx,
		permissionCallsToAICalls(calls),
		reg,
		"run-cancel",
		"turn-cancel",
		policy,
		stamper,
		sink,
	)
	cancel()
	<-drainDone

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
}

// --- D-A wake-path tests ---------------------------------------------

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

	if err := sched.WakeParked("wake_0"); err != nil {
		t.Fatalf("WakeParked(%q) = %v, want nil", "wake_0", err)
	}

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
	if err := sched.WakeParked("parked_real_0"); err != nil {
		t.Fatalf("WakeParked(%q) = %v, want nil", "parked_real_0", err)
	}

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

	if err := sched.WakeParked("no_deadline_0"); err != nil {
		t.Fatalf("WakeParked(%q) = %v, want nil", "no_deadline_0", err)
	}

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
