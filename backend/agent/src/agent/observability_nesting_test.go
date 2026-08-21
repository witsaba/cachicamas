// AG-22 — Phase 4 (RED, design D-G scenario 1): the nesting proof
// (R-AGO-006) and its per-key attribute-mapping half (R-AGO-002), driven
// against not-yet-instrumented code. Every subtest below wires a fresh
// tracetest.Provider into Harness.TracerProvider (or a delegation
// fixture's child Harness), drives a scripted run, and asserts a parent
// chain and per-key attribute values that can only hold once Phases 5-8
// instrument harness.go/loop.go/scheduler.go/compaction.go.
//
// The genuine RED right now: provider.Started() == 0 on every subtest —
// nothing opens a span yet. That is design's bite #2. As each seam
// lands (Phase 5: run; Phase 6: turn; Phase 7: tool/delegation; Phase 8:
// compaction), the corresponding subtest's early floor passes and its
// real parent-chain/attribute assertions start exercising real
// production code — this file is written once, not revisited per phase.
package agent_test

import (
	"testing"

	"go.opentelemetry.io/otel/attribute"

	"github.com/cachicamas/backend/agent/src/agent"
	"github.com/cachicamas/backend/agent/src/agenttest"
	"github.com/cachicamas/backend/agent/src/agenttest/tracetest"
	"github.com/cachicamas/backend/agent/src/ai"
)

// spanNamed returns the first span in spans whose Name() equals name, or
// nil. A test-local lookup helper — tracetest.Provider.Spans() returns
// spans in start order, not indexed by name.
func spanNamed(spans []*tracetest.Span, name string) *tracetest.Span {
	for _, s := range spans {
		if s.Name() == name {
			return s
		}
	}
	return nil
}

// spanWithNamePrefix returns the first span in spans whose Name() starts
// with prefix, or nil — the tool span's name is per-call
// ("execute_tool <toolname>", ADR 0005 § D3 Layer 2 extension), so exact
// match does not apply to it the way it does to run/turn/compaction.
func spanWithNamePrefix(spans []*tracetest.Span, prefix string) *tracetest.Span {
	for _, s := range spans {
		if len(s.Name()) >= len(prefix) && s.Name()[:len(prefix)] == prefix {
			return s
		}
	}
	return nil
}

// attrString returns the string value of the first attribute in attrs
// keyed key, and whether one was found.
func attrString(attrs []attribute.KeyValue, key string) (string, bool) {
	for _, kv := range attrs {
		if string(kv.Key) == key {
			return kv.Value.AsString(), true
		}
	}
	return "", false
}

// TestObservability_Nesting_RunTurnToolCompactionDelegation covers
// R-AGO-006 (nesting, including delegation) and R-AGO-002's per-key
// attribute mapping (run.id/turn.id/tool identity/compaction.id byte-
// equal to the event field each mirrors), one subtest per span family
// so each layer's RED clears independently as Phases 5-8 land.
func TestObservability_Nesting_RunTurnToolCompactionDelegation(t *testing.T) {
	t.Run("run_span_root_no_parent", func(t *testing.T) {
		t.Parallel()

		provider := tracetest.NewProvider()
		modelProvider := agenttest.NewProvider(scriptTextResponse(t, ai.FinishReasonStop))
		h := agent.Harness{Provider: modelProvider, System: "system prompt for nesting-run", TracerProvider: provider}

		sink := make(chan *agent.Event, 64)
		_, _, err := h.Run(contextBackground(), firstMessage(t), sink)
		if err != nil {
			t.Fatalf("Run returned err = %v, want nil", err)
		}
		events := drainSink(t, sink)

		if got := provider.Started(); got == 0 {
			t.Fatalf("provider.Started() = 0, want > 0 — no span was recorded for the run bracket")
		}
		runSpan := spanNamed(provider.Spans(), invokeAgentSpanNameForTest)
		if runSpan == nil {
			t.Fatalf("no span named %q recorded", invokeAgentSpanNameForTest)
		}
		if got := runSpan.Parent(); got != nil {
			t.Errorf("run span Parent() = %v, want nil — Run's own span is top-level", got)
		}

		wantRunID := findRunStartRunID(t, events)
		if got, ok := attrString(runSpan.Attributes(), "cachicamas.run.id"); !ok || got != string(wantRunID) {
			t.Errorf("run span cachicamas.run.id = (%q, present=%v), want (%q, true) — must equal the drained run_start's Event.Run()", got, ok, wantRunID)
		}
	})

	t.Run("turn_span_parent_is_run_span", func(t *testing.T) {
		t.Parallel()

		provider := tracetest.NewProvider()
		modelProvider := agenttest.NewProvider(scriptTextResponse(t, ai.FinishReasonStop))
		h := agent.Harness{Provider: modelProvider, System: "system prompt for nesting-turn", TracerProvider: provider}

		sink := make(chan *agent.Event, 64)
		if _, _, err := h.Run(contextBackground(), firstMessage(t), sink); err != nil {
			t.Fatalf("Run returned err = %v, want nil", err)
		}
		drainSink(t, sink)

		if got := provider.Started(); got == 0 {
			t.Fatalf("provider.Started() = 0, want > 0 — no span was recorded for the turn bracket")
		}
		spans := provider.Spans()
		runSpan := spanNamed(spans, invokeAgentSpanNameForTest)
		turnSpan := spanNamed(spans, turnSpanNameForTest)
		if runSpan == nil {
			t.Fatal("no run span recorded")
		}
		if turnSpan == nil {
			t.Fatal("no turn span recorded")
		}
		if got := turnSpan.Parent(); got != runSpan {
			t.Errorf("turn span Parent() = %v, want the exact run span %v", got, runSpan)
		}
	})

	t.Run("tool_span_parent_is_turn_span", func(t *testing.T) {
		t.Parallel()

		provider := tracetest.NewProvider()
		toolName := "nesting_tool"
		tool := EchoScriptedTool(toolName, agent.EffectClassRead)
		reg := agent.NewMapRegistry(map[string]agent.Tool{toolName: tool})

		turnOneScript := scriptToolCallResponse(t, "call-nest-tool-001", toolName, []byte(`{}`))
		turnTwoScript := scriptTextResponse(t, ai.FinishReasonStop)
		modelProvider := agenttest.NewProvider(turnOneScript, turnTwoScript)

		h := agent.Harness{
			Provider:       modelProvider,
			System:         "system prompt for nesting-tool",
			Turn:           agent.TurnOptions{Tools: reg},
			TracerProvider: provider,
		}
		sink := make(chan *agent.Event, 256)
		if _, _, err := h.Run(contextBackground(), firstMessage(t), sink); err != nil {
			t.Fatalf("Run returned err = %v, want nil", err)
		}
		events := drainSink(t, sink)

		if got := provider.Started(); got == 0 {
			t.Fatalf("provider.Started() = 0, want > 0 — no span was recorded for the tool bracket")
		}
		spans := provider.Spans()
		turnSpan := spanNamed(spans, turnSpanNameForTest)
		toolSpan := spanWithNamePrefix(spans, toolSpanNamePrefixForTest)
		if turnSpan == nil {
			t.Fatal("no turn span recorded")
		}
		if toolSpan == nil {
			t.Fatalf("no span with name prefix %q recorded", toolSpanNamePrefixForTest)
		}
		if got := toolSpan.Parent(); got != turnSpan {
			t.Errorf("tool span Parent() = %v, want the exact turn span %v (whichever turn issued the call)", got, turnSpan)
		}

		wantCallID := findToolStartCallID(t, events, toolName)
		if got, ok := attrString(toolSpan.Attributes(), "gen_ai.tool.call.id"); !ok || got != wantCallID {
			t.Errorf("tool span gen_ai.tool.call.id = (%q, present=%v), want (%q, true) — must equal the drained tool_start's CallID()", got, ok, wantCallID)
		}
	})

	t.Run("delegation_child_run_span_parent_is_delegating_tool_span", func(t *testing.T) {
		t.Parallel()

		provider := tracetest.NewProvider()

		childModelProvider := agenttest.NewProvider(scriptTextResponse(t, ai.FinishReasonStop))
		child := &agent.Harness{
			Provider:       childModelProvider,
			System:         "system prompt for the hosted nesting child",
			TracerProvider: provider, // design D-F: the same Provider wired into the child.
		}

		toolName := "nesting_delegating_tool"
		tool := &delegatingTool{toolName: toolName, effect: agent.EffectClassRead, child: child, prompt: firstMessage(t)}
		reg := agent.NewMapRegistry(map[string]agent.Tool{toolName: tool})

		turnOneScript := scriptToolCallResponse(t, "call-nest-del-001", toolName, []byte(`{}`))
		turnTwoScript := scriptTextResponse(t, ai.FinishReasonStop)
		parentModelProvider := agenttest.NewProvider(turnOneScript, turnTwoScript)

		h := agent.Harness{
			Provider:       parentModelProvider,
			System:         "system prompt for nesting-delegation parent",
			Turn:           agent.TurnOptions{Tools: reg},
			TracerProvider: provider,
		}
		sink := make(chan *agent.Event, 512)
		if _, _, err := h.Run(contextBackground(), firstMessage(t), sink); err != nil {
			t.Fatalf("parent Run returned err = %v, want nil", err)
		}
		drainSink(t, sink)

		if got := provider.Started(); got == 0 {
			t.Fatalf("provider.Started() = 0, want > 0 — no span was recorded for the delegating tool call")
		}
		spans := provider.Spans()
		toolSpan := spanWithNamePrefix(spans, toolSpanNamePrefixForTest)
		if toolSpan == nil {
			t.Fatalf("no span with name prefix %q recorded on the parent's provider", toolSpanNamePrefixForTest)
		}

		var childRunSpan *tracetest.Span
		for _, s := range spans {
			if s.Name() == invokeAgentSpanNameForTest && s.Parent() == toolSpan {
				childRunSpan = s
				break
			}
		}
		if childRunSpan == nil {
			t.Fatalf("no run span whose Parent() is the delegating tool's own span was found among %d recorded span(s)", len(spans))
		}
	})

	t.Run("compaction_span_recorded_standalone_root", func(t *testing.T) {
		t.Parallel()

		provider := tracetest.NewProvider()
		h, hist, _ := markedHarnessForCompaction(t, "nest-004")
		h.TracerProvider = provider

		compactionModelProvider := agenttest.NewProvider(scriptTextResponse(t, ai.FinishReasonStop))
		req := agent.CompactionRequest{Provider: compactionModelProvider, Instruction: "summarize turn one for nesting proof", Cut: hist.Len()}

		sink := make(chan *agent.Event, 64)
		if err := h.Compact(contextBackground(), req, sink); err != nil {
			t.Fatalf("Compact returned err = %v, want nil", err)
		}
		events := drainSink(t, sink)

		if got := provider.Started(); got == 0 {
			t.Fatalf("provider.Started() = 0, want > 0 — no span was recorded for the compaction bracket")
		}

		// AG-22 correction (MAJOR-6, sdd-verify round 1, S-AGO-019): the
		// compaction bracket carries EXACTLY ONE span, of the
		// compaction family, and no separate turn span was opened
		// alongside it (R-AGO-002's "one bracket, one span" clause).
		// markedHarnessForCompaction's own setup run uses a Harness with
		// no TracerProvider set (an unrelated no-op provider), so every
		// span this provider recorded belongs to THIS Compact call only
		// — h.Compact's own run span, plus whatever runCompaction opens.
		var compactionSpans, turnSpans []*tracetest.Span
		for _, s := range provider.Spans() {
			switch s.Name() {
			case compactionSpanNameForTest:
				compactionSpans = append(compactionSpans, s)
			case turnSpanNameForTest:
				turnSpans = append(turnSpans, s)
			}
		}
		if len(compactionSpans) != 1 {
			t.Fatalf("recorded %d span(s) named %q, want exactly 1 (S-AGO-019)", len(compactionSpans), compactionSpanNameForTest)
		}
		if len(turnSpans) != 0 {
			t.Errorf("recorded %d span(s) named %q for a compaction bracket, want 0 — no turn span may be opened alongside the compaction span (S-AGO-019, R-AGO-002)", len(turnSpans), turnSpanNameForTest)
		}
		compactionSpan := compactionSpans[0]

		wantCompactionID := findCompactionStartedID(t, events)
		if got, ok := attrString(compactionSpan.Attributes(), "cachicamas.compaction.id"); !ok || got != wantCompactionID {
			t.Errorf("compaction span cachicamas.compaction.id = (%q, present=%v), want (%q, true) — must equal the drained compaction_started's CompactionID()", got, ok, wantCompactionID)
		}

		// AG-22 correction (MAJOR-6, S-AGO-041): the compaction span's
		// parent is the span of the bracket that REQUESTED it. On
		// Harness.Compact's own door, the requester is Compact's own
		// run span (runCompaction ambiently acquires from the ctx
		// Compact already opened its run span onto) — never a turn
		// span, since Compact opens none.
		requesterSpan := spanNamed(provider.Spans(), invokeAgentSpanNameForTest)
		if requesterSpan == nil {
			t.Fatal("no run span recorded — cannot verify the compaction span's own parent")
		}
		if got := compactionSpan.Parent(); got != requesterSpan {
			t.Errorf("compaction span Parent() = %v, want the exact requesting run span %v (S-AGO-041)", got, requesterSpan)
		}
	})
}

// The four constants below restate ADR 0005 § D3's Layer 2 span names as
// this test file's own literals — deliberately NOT a reference to the
// unexported observability.go constants (invokeAgentSpanName etc.):
// this file proves the OBSERVABLE contract (what a span consumer would
// see), independent of whichever internal constant name production code
// happens to use.
const (
	invokeAgentSpanNameForTest = "invoke_agent"
	turnSpanNameForTest        = "turn"
	toolSpanNamePrefixForTest  = "execute_tool"
	compactionSpanNameForTest  = "compact"
)

// findRunStartRunID returns the RunID the drained events' run_start
// carries, failing the test if none is found.
func findRunStartRunID(t *testing.T, events []agent.Event) agent.RunID {
	t.Helper()
	for _, ev := range events {
		if _, ok := ev.RunStart(); ok {
			return ev.Run()
		}
	}
	t.Fatal("no run_start event found in the drained stream")
	return ""
}

// findToolStartCallID returns the CallID of the drained tool_start event
// for toolName, failing the test if none is found.
func findToolStartCallID(t *testing.T, events []agent.Event, toolName string) string {
	t.Helper()
	for _, ev := range events {
		if start, ok := ev.ToolStart(); ok && start.Name() == toolName {
			return start.CallID()
		}
	}
	t.Fatalf("no tool_start event for tool %q found in the drained stream", toolName)
	return ""
}

// findCompactionStartedID returns the CompactionID of the drained
// compaction_started event, failing the test if none is found.
func findCompactionStartedID(t *testing.T, events []agent.Event) string {
	t.Helper()
	for _, ev := range events {
		if started, ok := ev.CompactionStarted(); ok {
			return started.CompactionID()
		}
	}
	t.Fatal("no compaction_started event found in the drained stream")
	return ""
}
