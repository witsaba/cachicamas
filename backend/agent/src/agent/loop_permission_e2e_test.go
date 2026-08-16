// AG-10 remediation (verify-report W8) — the design.md:76 E2E:
// "Scripted fake provider drives a turn with one each: deferred,
// denied and modified call, asserting the full stream
// decision_required -> decision_made -> tool_start (modified args) ->
// tool_end_*". Before this, only TestTurn_PermissionPolicy_WiredToSchedule
// (loop_tool_dispatch_test.go) existed — a single-Deny wiring proof.
//
// Deviation from design.md's literal wording, recorded here rather
// than silently applied: the deferred call in this E2E does NOT
// reach decision_made via an external wake. loop.go's own comment at
// the Schedule call site says so directly — "the loop's own
// upward-path wake wiring (Scheduler.WakeParked) is AG-13's scope" —
// Turn() constructs its *Scheduler as an unexported local variable
// (loop.go, inside the FinishReasonToolCalls branch) and returns only
// (ai.Message, ai.FinishReason, error); it exposes no handle any
// caller could use to reach WakeParked for a call parked inside that
// Turn() invocation. Building the literal wake-resume E2E design.md
// describes would require widening Turn's public surface — an
// AG-13-scoped design decision this remediation round has no mandate
// to make.
//
// What this test proves instead, fully achievable within Turn()'s
// CURRENT public contract: all three outcomes are reachable from
// Turn(), not just Deny. RTLS-008 (permission_protocol_test.go)
// already proves the mixed-outcome matrix at the Schedule level; this
// proves the same three outcomes are reachable one layer up, through
// the actual public surface an AG-13 consumer calls. Deny and
// ModifyInput complete fully with the exact stream shape design.md
// specifies (decision_made, tool_start with modified args, tool_end);
// Defer produces decision_required and then aborts via context
// cancellation — the one release path actually reachable from outside
// Turn() at this milestone — proving sibling isolation (R-APP-008/009)
// holds at the Turn() level too, not merely at Schedule's.
package agent_test

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/cachicamas/backend/agent/src/agent"
	"github.com/cachicamas/backend/agent/src/agenttest"
	"github.com/cachicamas/backend/agent/src/ai"
)

func TestTurn_PermissionPolicy_E2E_DeferDenyModify(t *testing.T) {
	t.Parallel()

	denialInner, err := ai.MidStreamFailure(ai.FailureReport{Category: ai.FailureCategoryUnavailable}, false)
	if err != nil {
		t.Fatalf("ai.MidStreamFailure: %v", err)
	}
	denial, err := agent.NewFailure(denialInner)
	if err != nil {
		t.Fatalf("agent.NewFailure: %v", err)
	}
	const modifiedArgs = `{"slot":"e2e-modified"}`

	var mu sync.Mutex
	resolvedNames := map[string]int{}
	policy := &wiringTestPolicy{
		resolve: func(_ context.Context, call ai.ToolCall) agent.PermissionVerdict {
			mu.Lock()
			resolvedNames[call.Name()]++
			mu.Unlock()
			switch call.Name() {
			case "deferred_tool":
				return agent.PermissionVerdict{Outcome: agent.PermissionDefer}
			case "denied_tool":
				return agent.PermissionVerdict{Outcome: agent.PermissionOutcomeDeny, Failure: denial}
			case "modified_tool":
				return agent.PermissionVerdict{Outcome: agent.PermissionOutcomeModifyInput, ModifiedArgs: []byte(modifiedArgs)}
			default:
				return agent.PermissionVerdict{Outcome: agent.PermissionOutcomeAllowOnce}
			}
		},
	}

	deferredTool := EchoScriptedTool("deferred_tool", agent.EffectClassRead)
	deniedTool := EchoScriptedTool("denied_tool", agent.EffectClassRead)
	modifiedTool := EchoScriptedTool("modified_tool", agent.EffectClassRead)
	tools := map[string]agent.Tool{
		"deferred_tool": deferredTool,
		"denied_tool":   deniedTool,
		"modified_tool": modifiedTool,
	}

	tc1Start, err := ai.NewToolCallStart(1, "call-deferred", "deferred_tool")
	if err != nil {
		t.Fatalf("ai.NewToolCallStart(deferred): %v", err)
	}
	tc1Delta, err := ai.NewToolCallDelta(1, []byte(`{}`))
	if err != nil {
		t.Fatalf("ai.NewToolCallDelta(deferred): %v", err)
	}
	tc1End, err := ai.NewToolCallEnd(1, []byte(`{}`))
	if err != nil {
		t.Fatalf("ai.NewToolCallEnd(deferred): %v", err)
	}

	tc2Start, err := ai.NewToolCallStart(2, "call-denied", "denied_tool")
	if err != nil {
		t.Fatalf("ai.NewToolCallStart(denied): %v", err)
	}
	tc2Delta, err := ai.NewToolCallDelta(2, []byte(`{}`))
	if err != nil {
		t.Fatalf("ai.NewToolCallDelta(denied): %v", err)
	}
	tc2End, err := ai.NewToolCallEnd(2, []byte(`{}`))
	if err != nil {
		t.Fatalf("ai.NewToolCallEnd(denied): %v", err)
	}

	tc3Start, err := ai.NewToolCallStart(3, "call-modified", "modified_tool")
	if err != nil {
		t.Fatalf("ai.NewToolCallStart(modified): %v", err)
	}
	tc3Delta, err := ai.NewToolCallDelta(3, []byte(`{"slot":"e2e-original"}`))
	if err != nil {
		t.Fatalf("ai.NewToolCallDelta(modified): %v", err)
	}
	tc3End, err := ai.NewToolCallEnd(3, []byte(`{"slot":"e2e-original"}`))
	if err != nil {
		t.Fatalf("ai.NewToolCallEnd(modified): %v", err)
	}

	completion, err := ai.NewCompletion(ai.FinishReasonToolCalls, ai.Usage{})
	if err != nil {
		t.Fatalf("ai.NewCompletion: %v", err)
	}

	script := agenttest.Script{Steps: []agenttest.Step{
		agenttest.Emit(tc1Start), agenttest.Emit(tc1Delta), agenttest.Emit(tc1End),
		agenttest.Emit(tc2Start), agenttest.Emit(tc2Delta), agenttest.Emit(tc2End),
		agenttest.Emit(tc3Start), agenttest.Emit(tc3Delta), agenttest.Emit(tc3End),
		agenttest.Emit(completion),
	}}
	provider := agenttest.NewProvider(script)
	sink := make(chan *agent.Event, 256)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	ready, drainDone, drainedPtr := drainUntilDecisionRequired(sink, "call-deferred")

	turnDone := make(chan struct{})
	var msg ai.Message
	var finish ai.FinishReason
	var turnErr error
	go func() {
		defer close(turnDone)
		msg, finish, turnErr = agent.Turn(
			ctx,
			provider,
			"system for E2E policy wiring",
			firstMessageForTest(t),
			agent.TurnOptions{
				Tools:            agent.NewMapRegistry(tools),
				PermissionPolicy: policy,
			},
			sink,
		)
	}()

	select {
	case <-ready:
	case <-time.After(2 * time.Second):
		t.Fatal("decision_required for call-deferred not observed within 2s")
	}
	// The one release path reachable from outside Turn() at this
	// milestone (see package doc comment above): cancel, do not wake.
	cancel()

	select {
	case <-turnDone:
	case <-time.After(2 * time.Second):
		t.Fatal("Turn did not return within 2s of cancelling ctx")
	}
	<-drainDone
	events := *drainedPtr

	if turnErr != nil {
		t.Fatalf("Turn returned err = %v, want nil", turnErr)
	}
	if finish != ai.FinishReasonToolCalls {
		t.Errorf("Turn finish reason = %v, want FinishReasonToolCalls", finish)
	}
	_ = msg

	// --- deferred_tool: decision_required, then abort, never executes.
	deferredRequired := 0
	deferredMade := 0
	deferredEndFailure := 0
	for _, ev := range events {
		if req, ok := ev.PermissionDecisionRequired(); ok && req.CallID() == "call-deferred" {
			deferredRequired++
		}
		if made, ok := ev.PermissionDecisionMade(); ok && made.CallID() == "call-deferred" {
			deferredMade++
		}
		if end, ok := ev.ToolEndExecutionFailure(); ok && end.CallID() == "call-deferred" {
			deferredEndFailure++
		}
	}
	if deferredRequired != 1 {
		t.Errorf("permission_decision_required for call-deferred count = %d, want 1", deferredRequired)
	}
	if deferredMade != 0 {
		t.Errorf("permission_decision_made for call-deferred count = %d, want 0 (never woken — Turn() exposes no wake path yet)", deferredMade)
	}
	if deferredEndFailure != 1 {
		t.Errorf("tool_end_execution_failure for call-deferred count = %d, want 1 (ctx-cancel abort)", deferredEndFailure)
	}
	if inv := deferredTool.Invocations(); inv != 0 {
		t.Errorf("deferred_tool invocations = %d, want 0 (aborted before execution)", inv)
	}

	// --- denied_tool: decision_made{Deny}, never executes, typed end failure.
	deniedMade := 0
	deniedEndFailure := 0
	for _, ev := range events {
		if made, ok := ev.PermissionDecisionMade(); ok && made.CallID() == "call-denied" {
			deniedMade++
			if made.Outcome() != agent.PermissionOutcomeDeny {
				t.Errorf("decision_made for call-denied outcome = %v, want Deny", made.Outcome())
			}
		}
		if end, ok := ev.ToolEndExecutionFailure(); ok && end.CallID() == "call-denied" {
			deniedEndFailure++
		}
	}
	if deniedMade != 1 {
		t.Errorf("permission_decision_made for call-denied count = %d, want 1", deniedMade)
	}
	if deniedEndFailure != 1 {
		t.Errorf("tool_end_execution_failure for call-denied count = %d, want 1 (R-APP-005 typed denial)", deniedEndFailure)
	}
	if inv := deniedTool.Invocations(); inv != 0 {
		t.Errorf("denied_tool invocations = %d, want 0 (Deny skips execution)", inv)
	}

	// --- modified_tool: decision_made{ModifyInput}, tool_start with the
	// modified bytes (transparency, R-APP-006), tool_end_success.
	modifiedMade := 0
	modifiedStart := 0
	modifiedEndSuccess := 0
	for _, ev := range events {
		if made, ok := ev.PermissionDecisionMade(); ok && made.CallID() == "call-modified" {
			modifiedMade++
			if made.Outcome() != agent.PermissionOutcomeModifyInput {
				t.Errorf("decision_made for call-modified outcome = %v, want ModifyInput", made.Outcome())
			}
		}
		if start, ok := ev.ToolStart(); ok && start.CallID() == "call-modified" {
			modifiedStart++
			if got := string(start.Arguments()); got != modifiedArgs {
				t.Errorf("tool_start for call-modified arguments = %q, want %q (R-APP-006 transparency)", got, modifiedArgs)
			}
		}
		if end, ok := ev.ToolEndSuccess(); ok && end.CallID() == "call-modified" {
			modifiedEndSuccess++
		}
	}
	if modifiedMade != 1 {
		t.Errorf("permission_decision_made for call-modified count = %d, want 1", modifiedMade)
	}
	if modifiedStart != 1 {
		t.Errorf("tool_start for call-modified count = %d, want 1", modifiedStart)
	}
	if modifiedEndSuccess != 1 {
		t.Errorf("tool_end_success for call-modified count = %d, want 1", modifiedEndSuccess)
	}
	if inv := modifiedTool.Invocations(); inv != 1 {
		t.Errorf("modified_tool invocations = %d, want 1", inv)
	}
	if got := string(modifiedTool.RecordedArgs()[0]); got != modifiedArgs {
		t.Errorf("modified_tool received args = %q, want %q (bytes reach the tool and the stream in lockstep)", got, modifiedArgs)
	}

	mu.Lock()
	defer mu.Unlock()
	if resolvedNames["deferred_tool"] != 1 {
		t.Errorf("policy.Resolve invocations for deferred_tool = %d, want 1 (aborted before any wake could re-enter Resolve)", resolvedNames["deferred_tool"])
	}
	if resolvedNames["denied_tool"] != 1 {
		t.Errorf("policy.Resolve invocations for denied_tool = %d, want 1", resolvedNames["denied_tool"])
	}
	if resolvedNames["modified_tool"] != 1 {
		t.Errorf("policy.Resolve invocations for modified_tool = %d, want 1", resolvedNames["modified_tool"])
	}
}
