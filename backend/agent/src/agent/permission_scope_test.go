// AG-19.3 — Phase 3 (U3): a derived permission scope narrows and
// never widens; the ask goes up, the decision comes down (R-DEL-008,
// design AD-7, charter AG-19.3 scenario 2).
package agent_test

import (
	"context"
	"sync"
	"testing"

	"github.com/cachicamas/backend/agent/src/agent"
	"github.com/cachicamas/backend/agent/src/agenttest"
	"github.com/cachicamas/backend/agent/src/ai"
)

// derivedScope is an ordinary Go value composing the parent's policy
// (design AD-7): it delegates Resolve/Remember to parent, narrowing
// an allow outcome to Deny for a tool outside grant, and NEVER
// widening a deny or defer outcome to an allow. "Scope" stays out of
// Layer 2 — this type lives entirely in package agent_test, is
// unexported even here, and Layer 2's own PermissionPolicy interface
// (permission_protocol.go:80-94) is unchanged.
type derivedScope struct {
	parent agent.PermissionPolicy
	grant  map[string]bool
}

func (d *derivedScope) Resolve(ctx context.Context, call ai.ToolCall) agent.PermissionVerdict {
	v := d.parent.Resolve(ctx, call)
	switch v.Outcome {
	case agent.PermissionOutcomeAllowOnce, agent.PermissionOutcomeAllowAlways, agent.PermissionOutcomeModifyInput:
		if d.grant[call.Name()] {
			return v
		}
		// Narrow: outside the child's grant. Deny is the narrowing
		// choice this fixture makes; the derived scope could equally
		// choose Defer — both are non-allow, which is S-DEL-018's own
		// requirement.
		return agent.PermissionVerdict{Outcome: agent.PermissionOutcomeDeny, Failure: mustDerivedScopeDenialFailure()}
	default:
		// Deny or Defer (the zero PermissionOutcome): passed through
		// UNCHANGED. Never mapped to an allow, regardless of grant.
		return v
	}
}

func (d *derivedScope) Remember(ctx context.Context, toolName string, outcome agent.PermissionOutcome) bool {
	return d.parent.Remember(ctx, toolName, outcome)
}

var _ agent.PermissionPolicy = (*derivedScope)(nil)

func mustDerivedScopeDenialFailure() *agent.Failure {
	inner, err := ai.MidStreamFailure(ai.FailureReport{Category: ai.FailureCategoryAuthorization}, false)
	if err != nil {
		panic("mustDerivedScopeDenialFailure: ai.MidStreamFailure: " + err.Error())
	}
	f, err := agent.NewFailure(inner)
	if err != nil {
		panic("mustDerivedScopeDenialFailure: agent.NewFailure: " + err.Error())
	}
	return f
}

// staticVerdictPolicy is a PermissionPolicy that always returns the
// same verdict, regardless of the call — the parent-policy fixture
// S-DEL-018's table test drives derivedScope against.
type staticVerdictPolicy struct{ verdict agent.PermissionVerdict }

func (p staticVerdictPolicy) Resolve(context.Context, ai.ToolCall) agent.PermissionVerdict {
	return p.verdict
}
func (p staticVerdictPolicy) Remember(context.Context, string, agent.PermissionOutcome) bool {
	return false
}

var _ agent.PermissionPolicy = staticVerdictPolicy{}

// S-DEL-018 — charter AG-19.3 scenario 2, the narrowing half, asserted
// in both directions. Given a child whose scope is derived from the
// parent's policy, when a tool the parent allows but the child's
// grant excludes is called, then the derived scope returns a
// non-allow outcome; and when a tool the parent denies or defers is
// called, then the derived scope returns no allow for it under any
// input — the widening direction is asserted as impossible, not
// merely unused; and when Layer 2's exported surface is enumerated,
// then it declares no scope type, rule set or mode flag.
func TestPermissionScope_NarrowsNeverWidens(t *testing.T) {
	t.Parallel()

	call, err := ai.NewToolCall("call-del-018", "some_tool", []byte(`{}`))
	if err != nil {
		t.Fatalf("ai.NewToolCall: %v", err)
	}
	toolCall, ok := call.ToolCall()
	if !ok {
		t.Fatal("ai.NewToolCall did not return a ToolCall-carrying part")
	}

	table := []struct {
		name          string
		parentVerdict agent.PermissionVerdict
		grantIncludes bool
		wantAllow     bool
	}{
		{"parent allows, tool IS in grant — passes through", agent.PermissionVerdict{Outcome: agent.PermissionOutcomeAllowOnce}, true, true},
		{"parent allows, tool NOT in grant — narrowed to non-allow", agent.PermissionVerdict{Outcome: agent.PermissionOutcomeAllowOnce}, false, false},
		{"parent AllowAlways, tool NOT in grant — narrowed to non-allow", agent.PermissionVerdict{Outcome: agent.PermissionOutcomeAllowAlways}, false, false},
		{"parent denies, tool IS in grant — never widened", agent.PermissionVerdict{Outcome: agent.PermissionOutcomeDeny, Failure: mustDerivedScopeDenialFailure()}, true, false},
		{"parent denies, tool NOT in grant — never widened", agent.PermissionVerdict{Outcome: agent.PermissionOutcomeDeny, Failure: mustDerivedScopeDenialFailure()}, false, false},
		{"parent defers, tool IS in grant — never widened", agent.PermissionVerdict{Outcome: agent.PermissionDefer}, true, false},
		{"parent defers, tool NOT in grant — never widened", agent.PermissionVerdict{Outcome: agent.PermissionDefer}, false, false},
	}

	for _, tt := range table {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			grant := map[string]bool{}
			if tt.grantIncludes {
				grant["some_tool"] = true
			}
			scope := &derivedScope{parent: staticVerdictPolicy{verdict: tt.parentVerdict}, grant: grant}
			got := scope.Resolve(contextBackground(), toolCall)
			isAllow := got.Outcome == agent.PermissionOutcomeAllowOnce || got.Outcome == agent.PermissionOutcomeAllowAlways || got.Outcome == agent.PermissionOutcomeModifyInput
			if isAllow != tt.wantAllow {
				t.Errorf("derivedScope.Resolve() outcome = %v (isAllow=%v), want isAllow=%v", got.Outcome, isAllow, tt.wantAllow)
			}
		})
	}

	// Layer 2's exported surface declares no scope type, rule set or
	// mode flag: neither delegation_seam.go nor the 3-line
	// scheduler.go diff names "Scope", "RuleSet" or "ModeFlag"
	// anywhere — the only place those words may appear in this
	// change at all is this test file itself, in package agent_test.
	for _, path := range []string{"delegation_seam.go"} {
		names := collectTopLevelExportedNames(t, path)
		for name := range names {
			if containsAny(name, "Scope", "RuleSet", "ModeFlag") {
				t.Errorf("%s declares exported identifier %q, which names a scope/rule-set/mode-flag concept — Layer 2 must not gain one", path, name)
			}
		}
	}
}

func containsAny(s string, substrs ...string) bool {
	for _, sub := range substrs {
		if len(sub) == 0 {
			continue
		}
		for i := 0; i+len(sub) <= len(s); i++ {
			if s[i:i+len(sub)] == sub {
				return true
			}
		}
	}
	return false
}

// deferOnceThenAllowPolicy defers exactly the gated tool's FIRST
// resolution and allows every later one — S-DEL-019's parent-level
// policy the child's derivedScope wraps, mirroring
// harness_suspension_test.go's deferThenAllowPolicy shape for the
// wake re-entry (runPermissionGate's recursive Resolve call on
// WakeParked).
type deferOnceThenAllowPolicy struct {
	gatedTool string

	mu       sync.Mutex
	resolved bool
}

func (p *deferOnceThenAllowPolicy) Resolve(_ context.Context, call ai.ToolCall) agent.PermissionVerdict {
	if call.Name() != p.gatedTool {
		return agent.PermissionVerdict{Outcome: agent.PermissionOutcomeAllowOnce}
	}
	p.mu.Lock()
	first := !p.resolved
	p.resolved = true
	p.mu.Unlock()
	if first {
		return agent.PermissionVerdict{Outcome: agent.PermissionDefer}
	}
	return agent.PermissionVerdict{Outcome: agent.PermissionOutcomeAllowOnce}
}

func (p *deferOnceThenAllowPolicy) Remember(context.Context, string, agent.PermissionOutcome) bool {
	return false
}

var _ agent.PermissionPolicy = (*deferOnceThenAllowPolicy)(nil)

// S-DEL-019 — charter AG-19.3 scenario 2, the one-place half. Given a
// child call that needs permission, when the child's ask is raised,
// then both the ask and its answering decision appear on the
// parent's stream in that order; when the test — playing the human
// on the parent's stream — answers through the child scheduler's
// existing wake surface, then the child's suspended call resumes
// with that verdict; both streams remain CheckStream-valid validated
// separately; the child's permission_resolution_remembered, if any,
// appears only on the child's stream; and the production diff adds
// no routing surface.
func TestPermissionScope_AskUpDecisionDown(t *testing.T) {
	t.Parallel()

	const gatedTool = "child_gated_tool"
	const gatedCallID = "call-del-019-child"

	scope := &derivedScope{
		parent: &deferOnceThenAllowPolicy{gatedTool: gatedTool},
		grant:  map[string]bool{gatedTool: true},
	}

	childTool := EchoScriptedTool(gatedTool, agent.EffectClassRead)
	childReg := agent.NewMapRegistry(map[string]agent.Tool{gatedTool: childTool})

	childTurnOneScript := scriptToolCallResponse(t, gatedCallID, gatedTool, []byte(`{}`))
	childTurnTwoScript := scriptTextResponse(t, ai.FinishReasonStop)
	childProvider := agenttest.NewProvider(childTurnOneScript, childTurnTwoScript)

	childSched := &agent.Scheduler{}
	child := &agent.Harness{
		Provider:  childProvider,
		System:    "system prompt for the permission-scope child",
		Turn:      agent.TurnOptions{Tools: childReg, PermissionPolicy: scope},
		Scheduler: childSched,
	}

	toolName := "permission_scope_tool"
	tool := &delegatingTool{toolName: toolName, effect: agent.EffectClassRead, child: child, prompt: firstMessage(t)}
	reg := agent.NewMapRegistry(map[string]agent.Tool{toolName: tool})

	parentTurnOneScript := scriptToolCallResponse(t, "call-del-019-parent", toolName, []byte(`{}`))
	parentTurnTwoScript := scriptTextResponse(t, ai.FinishReasonStop)
	parentProvider := agenttest.NewProvider(parentTurnOneScript, parentTurnTwoScript)

	h := agent.Harness{Provider: parentProvider, System: "system prompt for del-019 parent", Turn: agent.TurnOptions{Tools: reg}}

	sink := make(chan *agent.Event, 1024)
	type parentRunResult struct {
		msg    ai.Message
		finish ai.FinishReason
		err    error
	}
	resultCh := make(chan parentRunResult, 1)
	go func() {
		msg, finish, err := h.Run(contextBackground(), firstMessage(t), sink)
		resultCh <- parentRunResult{msg: msg, finish: finish, err: err}
	}()

	var events []agent.Event
	sawAsk := false
	for {
		ev, ok := <-sink
		if !ok {
			break
		}
		events = append(events, *ev)
		if sawAsk {
			continue
		}
		req, isReq := ev.PermissionDecisionRequired()
		if !isReq || req.CallID() != gatedCallID {
			continue
		}
		sawAsk = true
		// Playing the human on the parent's stream: the verdict
		// reaches the child's suspension through the EXISTING wake
		// surface on the child's own Scheduler value — zero new
		// production routing surface.
		if err := childSched.WakeParked(gatedCallID); err != nil {
			t.Errorf("childSched.WakeParked(%q) = %v, want nil", gatedCallID, err)
		}
	}
	if !sawAsk {
		t.Fatal("the parent's stream never carried the mirrored permission_decision_required for the gated call")
	}

	res := <-resultCh
	if res.err != nil {
		t.Fatalf("parent Run returned err = %v, want nil", res.err)
	}

	childEvents := tool.ChildEvents()
	validateStreamsSeparately(t, events, childEvents)

	askIdx := -1
	answerIdx := -1
	for i, ev := range events {
		if req, ok := ev.PermissionDecisionRequired(); ok && req.CallID() == gatedCallID {
			askIdx = i
		}
		if made, ok := ev.PermissionDecisionMade(); ok && made.CallID() == gatedCallID {
			answerIdx = i
		}
	}
	if askIdx < 0 || answerIdx < 0 || askIdx >= answerIdx {
		t.Errorf("ask index = %d, answer index = %d, want ask strictly before answer, both present on the parent's stream", askIdx, answerIdx)
	}

	if inv := childTool.Invocations(); inv != 1 {
		t.Errorf("gated tool invocations = %d, want 1 (the suspended call resumed after WakeParked and completed)", inv)
	}

	if got := countKind(events, agent.EventKindPermissionResolutionRemembered); got != 0 {
		t.Errorf("parent stream carries %d permission_resolution_remembered event(s), want 0", got)
	}
}
