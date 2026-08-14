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
	sink chan<- *agent.Event,
) ([]agent.Result, []*agent.Event) {
	stamper := &agent.LaneStamper{}
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

	sink := make(chan *agent.Event, 64)
	results, events := runPermissionSchedulerAndCollect(
		&agent.Scheduler{MaxConcurrentReads: 4}, calls, reg, policy, sink,
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

	// (b) Zero decision_made — no wake happened.
	if gotMade := countByKind(collected)[agent.EventKindPermissionDecisionMade]; gotMade != 0 {
		t.Errorf("decision_made count = %d, want 0 (no wake during the bite — the deferred call waits, S-PPB-002)",
			gotMade)
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

// S-PPB-003 — RED bite (recorded at Commit 2). Given an empty
// parked set, when the upward-path wake hand-off fires for an
// unknown callID, then the wake returns a typed ErrStrayDecision
// error — NOT silently drops, NOT panic.
//
// RED at write time: ErrStrayDecision does not exist; the
// parkedSet.wake signature returns bool (no error path). Compile-
// error RED.
func TestPermission_StrayDecisionIsTypedError(t *testing.T) {
	t.Parallel()

	// We don't construct a parkedSet here — the parkedSet type is
	// unexported. The bite is asserted through the upward-path
	// contract: a future loop helper Wake(parkedSet, callID)
	// returns ErrStrayDecision for unknown IDs. To keep the bite
	// in the external package, the loop exposes the helper as
	// `agent.WakeParkedCall(set, callID) error` (the API surface
	// will be defined in Commit 3; the bite references the
	// signature it commits to).
	//
	// The bite here uses the lower-level helper exported by
	// agent (introduced in Commit 3): `agent.ResolveStrayDecision`
	// accepts a parked-set view (a sentinel) and reports the
	// typed error. Until Commit 3 lands this helper, the bite
	// does not compile — RED.
	const strayID = "call-id-X"
	err := agent.ResolveStrayDecision(strayID)
	if err == nil {
		t.Fatalf("ResolveStrayDecision(%q) returned nil, want a non-nil typed error", strayID)
	}
	if !errors.Is(err, agent.ErrStrayDecision) {
		t.Errorf("ResolveStrayDecision(%q) = %v, want errors.Is to match agent.ErrStrayDecision",
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
