// AG-22 — Phase 4 (RED, design D-E / D-G scenario 2): the absence proof
// (R-AGO-004, R-AGO-005, R-AGO-008), mirroring AI-37's own
// ai37_denylist_test.go precedent (src/ai/openaicompat), adapted to
// Layer 2's own event vocabulary and driven against not-yet-instrumented
// code.
//
// Nine runtime-CONCATENATED sentinels are planted (design D-E): prompt,
// reasoning, tool-call arguments, tool-result, permission-decision
// arguments, compaction instruction, compaction summary, credential-
// shaped and header-shaped. Tool-call arguments and permission-decision
// arguments share ONE planted value — at Layer 2 they are literally the
// same call's argument bytes, read through two different accessors
// (ToolStart.Arguments(), PermissionDecisionRequired.Arguments()) — so
// eight distinct byte values cover the nine named vectors.
//
// Credential-shaped and header-shaped content have no delivery mechanism
// of their own at Layer 2 (no HTTP call, no header, no credential is
// ever constructed here — that is Layer 1's own surface, AI-37's
// denylist test already covers it there); they are included in
// denyEntries so sweep.SelfTest's positive control still proves the SCAN
// mechanism itself would catch them, and their absence from the corpus
// holds by construction (no code path in this package could ever
// produce them), the strongest form the other seven vectors are proven
// empirically instead.
package agent_test

import (
	"context"
	"testing"

	"github.com/cachicamas/backend/agent/src/agent"
	"github.com/cachicamas/backend/agent/src/agenttest"
	"github.com/cachicamas/backend/agent/src/agenttest/sweep"
	"github.com/cachicamas/backend/agent/src/agenttest/tracetest"
	"github.com/cachicamas/backend/agent/src/ai"
)

// agoDenylistCanaries names the eight runtime-assembled markers this
// test plants. Every value is built by concatenation, never as one
// contiguous literal, so this file's own source bytes never contain a
// needle the scan below searches other bytes for.
type agoDenylistCanaries struct {
	prompt                string
	reasoning             string
	toolArgs              string // covers ToolStart.Arguments() AND PermissionDecisionRequired.Arguments()
	toolResult            string
	compactionInstruction string
	summary               string
	credential            string
	header                string
}

func newAGODenylistCanaries() agoDenylistCanaries {
	return agoDenylistCanaries{
		prompt:                "AGO" + "-PROMPT-" + "CANARY-9f2a",
		reasoning:             "AGO" + "-REASONING-" + "CANARY-3c7e",
		toolArgs:              "AGO" + "-TOOLARGS-" + "CANARY-5b1d",
		toolResult:            "AGO" + "-TOOLRESULT-" + "CANARY-8e4f",
		compactionInstruction: "AGO" + "-COMPACTINSTR-" + "CANARY-4a1c",
		summary:               "AGO" + "-SUMMARY-" + "CANARY-6d2e",
		credential:            "AGO" + "-CREDENTIAL-" + "CANARY-2d6a",
		header:                "AGO" + "-HEADER-" + "CANARY-7a3b",
	}
}

// denyEntries builds the sweep deny-list, one entry per named vector
// (D-E), tool-args/permission-args sharing one needle as documented
// above.
func (c agoDenylistCanaries) denyEntries() []sweep.Entry {
	return []sweep.Entry{
		{Vector: "prompt", Needle: []byte(c.prompt)},
		{Vector: "reasoning", Needle: []byte(c.reasoning)},
		{Vector: "tool-call arguments (and permission-decision arguments)", Needle: []byte(c.toolArgs)},
		{Vector: "tool-result", Needle: []byte(c.toolResult)},
		{Vector: "compaction instruction", Needle: []byte(c.compactionInstruction)},
		{Vector: "compaction summary", Needle: []byte(c.summary)},
		{Vector: "credential", Needle: []byte(c.credential)},
		{Vector: "credential (bearer form)", Needle: []byte("Bearer " + c.credential)},
		{Vector: "header-shaped value", Needle: []byte(c.header)},
	}
}

// agoDenylistPromptMessage builds the user message carrying the prompt
// canary.
func agoDenylistPromptMessage(t *testing.T, promptCanary string) ai.Message {
	t.Helper()
	part, err := ai.NewText("hello " + promptCanary)
	if err != nil {
		t.Fatalf("ai.NewText(prompt) error = %v, want nil", err)
	}
	msg, err := ai.NewMessage(ai.RoleUser, part)
	if err != nil {
		t.Fatalf("ai.NewMessage(user) error = %v, want nil", err)
	}
	return msg
}

// agoDenylistReasoningToolCallScript builds turn one: a reasoning block
// carrying the reasoning canary, then a tool call whose arguments carry
// the tool-args canary, finishing with FinishReasonToolCalls.
func agoDenylistReasoningToolCallScript(t *testing.T, c agoDenylistCanaries, toolName, callID string) agenttest.Script {
	t.Helper()

	reasonStartEvent, err := ai.NewReasoningBlockStart(1)
	if err != nil {
		t.Fatalf("ai.NewReasoningBlockStart returned %v, want nil", err)
	}
	reasonStartPayload, ok := reasonStartEvent.ReasoningBlockStart()
	if !ok {
		t.Fatal("reasoning block start event carries no ReasoningBlockStart payload")
	}
	reasonDelta, err := ai.NewReasoningDelta(reasonStartPayload, []byte(c.reasoning))
	if err != nil {
		t.Fatalf("ai.NewReasoningDelta returned %v, want nil", err)
	}
	reasonEnd, err := ai.NewReasoningBlockEnd(reasonStartPayload, []byte{0x01, 0x02, 0x03})
	if err != nil {
		t.Fatalf("ai.NewReasoningBlockEnd returned %v, want nil", err)
	}

	toolArgs := []byte(`{"q":"` + c.toolArgs + `"}`)
	tcStart, err := ai.NewToolCallStart(2, callID, toolName)
	if err != nil {
		t.Fatalf("ai.NewToolCallStart returned %v, want nil", err)
	}
	tcDelta, err := ai.NewToolCallDelta(2, toolArgs)
	if err != nil {
		t.Fatalf("ai.NewToolCallDelta returned %v, want nil", err)
	}
	tcEnd, err := ai.NewToolCallEnd(2, toolArgs)
	if err != nil {
		t.Fatalf("ai.NewToolCallEnd returned %v, want nil", err)
	}
	completion, err := ai.NewCompletion(ai.FinishReasonToolCalls, ai.Usage{})
	if err != nil {
		t.Fatalf("ai.NewCompletion returned %v, want nil", err)
	}
	return agenttest.Script{Steps: []agenttest.Step{
		agenttest.Emit(reasonStartEvent),
		agenttest.Emit(reasonDelta),
		agenttest.Emit(reasonEnd),
		agenttest.Emit(tcStart),
		agenttest.Emit(tcDelta),
		agenttest.Emit(tcEnd),
		agenttest.Emit(completion),
	}}
}

// agoDenylistTextScript builds a plain text-only scripted stream whose
// content is text, finishing with FinishReasonStop — used for both the
// second turn of the tool-calling run and the compaction call's own
// completion (whose text is the "summary" the compaction commits).
func agoDenylistTextScript(t *testing.T, text string) agenttest.Script {
	t.Helper()
	start, err := ai.NewTextBlockStart(1)
	if err != nil {
		t.Fatalf("ai.NewTextBlockStart returned %v, want nil", err)
	}
	delta, err := ai.NewTextDelta(1, text)
	if err != nil {
		t.Fatalf("ai.NewTextDelta returned %v, want nil", err)
	}
	end, err := ai.NewTextBlockEnd(1)
	if err != nil {
		t.Fatalf("ai.NewTextBlockEnd returned %v, want nil", err)
	}
	completion, err := ai.NewCompletion(ai.FinishReasonStop, ai.Usage{})
	if err != nil {
		t.Fatalf("ai.NewCompletion returned %v, want nil", err)
	}
	return agenttest.Script{Steps: []agenttest.Step{
		agenttest.Emit(start),
		agenttest.Emit(delta),
		agenttest.Emit(end),
		agenttest.Emit(completion),
	}}
}

// TestObservability_DenylistAbsence is R-AGO-007's absence proof and
// R-AGO-008's five-guard non-vacuity harness, in the design D-E order:
// (1) SelfTest positive control, (2) AssertAllEndedOnce, (3) corpus
// non-empty + attribute-count floor, (4) event/outcome coverage, (5)
// every needle non-empty — THEN the scan itself.
func TestObservability_DenylistAbsence(t *testing.T) {
	c := newAGODenylistCanaries()
	provider := tracetest.NewProvider()

	// --- Guard 1: the positive control runs before any clean scan ---
	deny := c.denyEntries()
	if err := sweep.SelfTest(deny); err != nil {
		t.Fatalf("sweep.SelfTest(deny) = %v, want nil — every needle must bite its own synthetic corpus (R-AGO-008 item 1)", err)
	}

	var allEvents []agent.Event

	// --- Run A: reasoning + a permission-gated tool call, sharing
	// provider. The one call is DEFERRED on its first resolution and
	// woken (mirroring harness_suspension_test.go's driveSuspendedRun,
	// same agent_test package), so this run genuinely exercises BOTH
	// permission_decision_required and permission_decision_made —
	// R-AGO-008 item 4's own "permission" coverage word — rather than
	// only the latter. An always-AllowOnce policy (the prior shape)
	// never emits permission_decision_required at all: runPermissionGate
	// (scheduler.go) constructs that event on the Defer verdict only. ---
	toolName := "denylist_tool"
	tool := NewScriptedTool(toolName, agent.EffectClassRead, agent.Result{Outcome: agent.ToolOutcomeSuccess, Content: []byte(c.toolResult)})
	reg := agent.NewMapRegistry(map[string]agent.Tool{toolName: tool})
	const denylistCallID = "call-ago-denylist-001"
	resolveCount := 0
	policy := &wiringTestPolicy{resolve: func(context.Context, ai.ToolCall) agent.PermissionVerdict {
		resolveCount++
		if resolveCount == 1 {
			return agent.PermissionVerdict{Outcome: agent.PermissionDefer}
		}
		return agent.PermissionVerdict{Outcome: agent.PermissionOutcomeAllowOnce}
	}}

	turnOneScript := agoDenylistReasoningToolCallScript(t, c, toolName, denylistCallID)
	turnTwoScript := agoDenylistTextScript(t, "final answer, no canaries here")
	modelProviderA := agenttest.NewProvider(turnOneScript, turnTwoScript)

	// A caller-owned Scheduler is required to reach WakeParked while the
	// call is still parked mid-Schedule (design "Decision 2", the same
	// injection harness_suspension_test.go's driveSuspendedRun uses).
	schedA := &agent.Scheduler{}
	hA := agent.Harness{
		Provider:       modelProviderA,
		System:         "system prompt for denylist absence proof",
		Turn:           agent.TurnOptions{Tools: reg, PermissionPolicy: policy},
		Scheduler:      schedA,
		TracerProvider: provider,
	}
	sinkA := make(chan *agent.Event, 512)
	runAErrCh := make(chan error, 1)
	go func() {
		_, _, err := hA.Run(contextBackground(), agoDenylistPromptMessage(t, c.prompt), sinkA)
		runAErrCh <- err
	}()

	// Read the stream event by event (synchronization by channel reads
	// only, never a wall clock) so the parked call can be woken once its
	// own permission_decision_required is observed. Run closes sinkA
	// when it returns (harness.go's defer close(sink)), which ends this
	// range.
	sawDecisionRequired := false
	for ev := range sinkA {
		allEvents = append(allEvents, *ev)
		if sawDecisionRequired {
			continue
		}
		req, isReq := ev.PermissionDecisionRequired()
		if !isReq || req.CallID() != denylistCallID {
			continue
		}
		sawDecisionRequired = true
		if err := schedA.WakeParked(denylistCallID); err != nil {
			t.Fatalf("WakeParked(%q) = %v, want nil", denylistCallID, err)
		}
	}
	if !sawDecisionRequired {
		t.Fatal("Run A never observed permission_decision_required on the consumer stream")
	}
	if err := <-runAErrCh; err != nil {
		t.Fatalf("Run A returned err = %v, want nil", err)
	}

	// --- Run B: a compaction call, sharing the same provider ---
	hB, hist, _ := markedHarnessForCompaction(t, "denylist-001")
	hB.TracerProvider = provider
	compactionModelProvider := agenttest.NewProvider(agoDenylistTextScript(t, c.summary))
	reqB := agent.CompactionRequest{Provider: compactionModelProvider, Instruction: c.compactionInstruction, Cut: hist.Len()}
	sinkB := make(chan *agent.Event, 64)
	if err := hB.Compact(contextBackground(), reqB, sinkB); err != nil {
		t.Fatalf("Compact returned err = %v, want nil", err)
	}
	allEvents = append(allEvents, drainSink(t, sinkB)...)

	// --- Run C: a failure arm, sharing the same provider (R-AGO-008
	// item 3's "failure" coverage — Layer 2 registers no separate error
	// EventKind; failures ride RunOutcomeFailed/TurnOutcomeAborted). ---
	modelProviderC := agenttest.NewProvider(scriptTextThenTerminalFailure(t, ai.FailureCategoryUnavailable, true))
	hC := agent.Harness{Provider: modelProviderC, System: "system prompt for denylist failure arm", TracerProvider: provider}
	sinkC := make(chan *agent.Event, 64)
	if _, _, err := hC.Run(contextBackground(), firstMessage(t), sinkC); err == nil {
		t.Fatal("Run C returned err = nil, want a non-nil error (mid-stream terminal failure)")
	}
	allEvents = append(allEvents, drainSink(t, sinkC)...)

	// --- Guard 2: exactly-once precondition (reuses R-AGO-007's own
	// assertion on the same runs, per D-E's guard 2) ---
	if err := provider.AssertAllEndedOnce(); err != nil {
		t.Fatalf("AssertAllEndedOnce() = %v, want nil (R-AGO-008 item 2's exactly-once precondition)", err)
	}

	// --- Guard 3: corpus non-empty + attribute-count floor ---
	corpus := provider.Corpus()
	if len(corpus) == 0 {
		t.Fatal("Corpus() is empty — no span was recorded; the absence claim would pass vacuously (R-AGO-008 item 2)")
	}
	// AG-22 correction (MAJOR/MINOR-2, sdd-verify round 1): the floor was
	// a raw SUM of attribute occurrences across every span, compared
	// against the vocabulary's own 14-key cardinality — a family that
	// stopped recording entirely could still clear 14 on the strength of
	// the OTHER families' own repeated attributes, so the floor could
	// never actually detect that. The floor is now the recorded UNIQUE
	// key SET's own size — a family that stops recording drops this
	// count directly, one key at a time.
	//
	// wantMinUniqueKeys is 12, not the vocabulary's full 14: this run's
	// own scenario mix (Run A: reasoning + a deferred-then-allowed tool
	// call; Run B: a finishing compaction; Run C: a failed run) never
	// delegates and never detaches a tool call, so cachicamas.run.
	// parent_id and cachicamas.tool.detached — both iff-guarded absent
	// on every path this run exercises — are the two vocabulary members
	// this specific run cannot produce. Both are asserted present, with
	// real values, elsewhere (observability_attributes_test.go's own
	// scenarios D/E; observability_lifecycle_test.go's detached-arm
	// tests) — this floor's own job is non-vacuity for THIS run's
	// actually-reachable keys, not re-proving the full vocabulary.
	const wantMinUniqueKeys = 12
	uniqueKeys := make(map[string]bool)
	for _, span := range provider.Spans() {
		for _, kv := range span.Attributes() {
			uniqueKeys[string(kv.Key)] = true
		}
	}
	if len(uniqueKeys) < wantMinUniqueKeys {
		t.Errorf("recorded unique attribute key count = %d (keys: %v), want at least %d — a family that stopped recording would drop this floor directly (R-AGO-008 item 2)", len(uniqueKeys), uniqueKeys, wantMinUniqueKeys)
	}

	// --- Guard 4: event/outcome coverage — every drained kind this
	// scenario is scripted to produce must actually appear, so the
	// absence claim is proven over a run that genuinely used it. ---
	gotKinds := make(map[agent.EventKind]bool, len(allEvents))
	for _, ev := range allEvents {
		gotKinds[ev.Kind()] = true
	}
	wantKinds := []agent.EventKind{
		agent.EventKindRunStart, agent.EventKindRunEnd,
		agent.EventKindTurnStart, agent.EventKindTurnEnd,
		agent.EventKindToolStart, agent.EventKindToolEndSuccess,
		agent.EventKindPermissionDecisionRequired, agent.EventKindPermissionDecisionMade,
		agent.EventKindCompactionStarted, agent.EventKindCompactionFinished,
		// AG-22 correction (MAJOR-3, sdd-verify round 1): R-AGO-008 item
		// 4 names "reasoning" explicitly among the event kinds this
		// non-vacuity guard must cover, and Run A's own script
		// (agoDenylistReasoningToolCallScript) already drives a full
		// reasoning bracket — the coverage gap was in this list, not in
		// what Run A produces.
		agent.EventKindMessageStartReasoning, agent.EventKindMessageDeltaReasoning, agent.EventKindMessageEndReasoning,
	}
	for _, want := range wantKinds {
		if !gotKinds[want] {
			t.Errorf("drained recording never contained an event of kind %v — the absence claim must be proven over a run that used it (R-AGO-008 item 3)", want)
		}
	}
	sawFailedRun := false
	for _, ev := range allEvents {
		if end, ok := ev.RunEnd(); ok && end.Outcome() == agent.RunOutcomeFailed {
			sawFailedRun = true
		}
	}
	if !sawFailedRun {
		t.Error("drained recording never contained a run_end(Failed) — the failure arm's own coverage is unproven (R-AGO-008 item 3)")
	}

	// --- Guard 5: every needle non-empty ---
	for _, entry := range deny {
		if len(entry.Needle) == 0 {
			t.Fatalf("deny entry %q has an empty needle — the scan below would never have been able to find it", entry.Vector)
		}
	}

	// --- The absence scan itself (R-AGO-007) ---
	for _, entry := range deny {
		if _, found := sweep.Scan(corpus, []sweep.Entry{entry}); found {
			t.Errorf("denylist leak: vector %q found in the corpus (R-AGO-007) — failure names the vector only, never the matched bytes (S-AGO-029)", entry.Vector)
		}
	}
}
