// AG-22 correction (MAJOR-1, MAJOR-2, sdd-verify round 1): R-AGO-002's
// attribute vocabulary was essentially unasserted — only 3 of 14 keys
// appeared anywhere in the test suite, and S-AGO-014's own value-equality
// claim (the charter's headline acceptance, "attributes from the decided
// vocabulary with values equal to the corresponding events'", `0003:2080`)
// covered 3 of the table's 21 rows. This file drives every iff-branch of
// every row at least once — both the PRESENT arm and the ABSENT arm — and
// asserts value equality generically, by correlating each recorded span's
// own bracket-identity attribute back to the drained event stream, rather
// than hand-picking one span per subtest as the four proof-type files do.
//
// Closes: S-AGO-013 (subset, no other key), S-AGO-014 (every row, value
// equality), S-AGO-017 (presence-typed absence, both directions),
// S-AGO-018 (status/description), S-AGO-021 (error.type only on typed
// failure).
//
// NOT closed here, disclosed rather than silently skipped: S-AGO-023
// ("Given a traced run in which Layer 1's own request span is also
// recorded..."). agenttest.Provider (used by every fixture in this
// package) is a scripted ai.ModelProvider fake that never reaches
// src/ai/openaicompat's real, instrumented adapter — AI-37's own request
// span is therefore never recorded by any fixture this package can drive
// without standing up that adapter behind a mocked HTTP transport, a
// materially larger undertaking than this correction's proportional
// scope. S-AGO-020 (grep-based, the same limitation) was already
// disclosed as statically evidenced only, by sdd-verify's own report.
package agent_test

import (
	"context"
	"testing"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"

	"github.com/cachicamas/backend/agent/src/agent"
	"github.com/cachicamas/backend/agent/src/agenttest"
	"github.com/cachicamas/backend/agent/src/agenttest/tracetest"
	"github.com/cachicamas/backend/agent/src/ai"
)

// agoAttributeVocabulary is R-AGO-002's own closed 14-key vocabulary
// (ADR 0005 § D3 Layer 2 extension) — S-AGO-013's own target set: the
// key union of every span this capability records MUST be a subset of
// exactly this map, and no other key may appear under any name.
var agoAttributeVocabulary = map[string]bool{
	"gen_ai.operation.name":            true,
	"cachicamas.run.id":                true,
	"cachicamas.run.parent_id":         true,
	"cachicamas.run.outcome":           true,
	"cachicamas.turn.id":               true,
	"cachicamas.turn.outcome":          true,
	"gen_ai.tool.name":                 true,
	"gen_ai.tool.call.id":              true,
	"cachicamas.tool.ordinal":          true,
	"cachicamas.tool.outcome":          true,
	"cachicamas.tool.detached":         true,
	"cachicamas.compaction.id":         true,
	"cachicamas.compaction.summary_id": true,
	"error.type":                       true,
}

// attrBool returns the bool value of the first attribute in attrs keyed
// key, and whether one was found — attrString's (observability_nesting_
// test.go) sibling for cachicamas.tool.detached.
func attrBool(attrs []attribute.KeyValue, key string) (bool, bool) {
	for _, kv := range attrs {
		if string(kv.Key) == key {
			return kv.Value.AsBool(), true
		}
	}
	return false, false
}

// attrInt64 returns the int64 value of the first attribute in attrs
// keyed key, and whether one was found — attrString's sibling for
// cachicamas.tool.ordinal.
func attrInt64(attrs []attribute.KeyValue, key string) (int64, bool) {
	for _, kv := range attrs {
		if string(kv.Key) == key {
			return kv.Value.AsInt64(), true
		}
	}
	return 0, false
}

// agoAssertStatus is S-AGO-018's own shared assertion: the error status
// exactly when the bracket's own outcome is its failure member, with the
// description carrying the failure category name ONLY — otherwise the ok
// status.
func agoAssertStatus(t *testing.T, family, id string, span *tracetest.Span, wantFailed bool, wantCategory string) {
	t.Helper()
	gotCode, gotDesc := span.Status()
	if wantFailed {
		if gotCode != codes.Error {
			t.Errorf("%s span %s status code = %v, want Error (S-AGO-018)", family, id, gotCode)
		}
		if gotDesc != wantCategory {
			t.Errorf("%s span %s status description = %q, want %q — the failure category name only, never a wrapped error's text (S-AGO-018)", family, id, gotDesc, wantCategory)
		}
		return
	}
	if gotCode != codes.Ok {
		t.Errorf("%s span %s status code = %v, want Ok (S-AGO-018)", family, id, gotCode)
	}
}

// agoAssertRunSpanRow is R-AGO-002's run row, S-AGO-014 (value equality)
// and S-AGO-017 (presence-typed absence) both directions.
func agoAssertRunSpanRow(t *testing.T, span *tracetest.Span, events []agent.Event) {
	t.Helper()
	runIDStr, ok := attrString(span.Attributes(), "cachicamas.run.id")
	if !ok {
		t.Errorf("run span %q missing cachicamas.run.id", span.Name())
		return
	}
	runID := agent.RunID(runIDStr)

	if got, ok := attrString(span.Attributes(), "gen_ai.operation.name"); !ok || got != "invoke_agent" {
		t.Errorf("run span %s gen_ai.operation.name = (%q, present=%v), want (\"invoke_agent\", true)", runIDStr, got, ok)
	}

	var wantParent agent.RunID
	var wantParentOK, haveStart, haveEnd bool
	var runEnd agent.RunEnd
	for _, ev := range events {
		if ev.Run() != runID {
			continue
		}
		if _, isStart := ev.RunStart(); isStart && !haveStart {
			wantParent, wantParentOK = ev.Parent()
			haveStart = true
		}
		if re, isEnd := ev.RunEnd(); isEnd && !haveEnd {
			runEnd = re
			haveEnd = true
		}
	}
	if haveStart {
		gotParent, gotOK := attrString(span.Attributes(), "cachicamas.run.parent_id")
		if wantParentOK != gotOK {
			t.Errorf("run span %s cachicamas.run.parent_id presence = %v, want %v (Event.Parent()'s own presence, S-AGO-017)", runIDStr, gotOK, wantParentOK)
		} else if wantParentOK && gotParent != string(wantParent) {
			t.Errorf("run span %s cachicamas.run.parent_id = %q, want %q", runIDStr, gotParent, wantParent)
		}
	}
	if !haveEnd {
		t.Errorf("run span %s: no run_end found for this run in the drained stream", runIDStr)
		return
	}

	if got, ok := attrString(span.Attributes(), "cachicamas.run.outcome"); !ok || got != runEnd.Outcome().String() {
		t.Errorf("run span %s cachicamas.run.outcome = (%q, present=%v), want (%q, true)", runIDStr, got, ok, runEnd.Outcome().String())
	}

	failure, hasFailure := runEnd.Failure()
	category := ""
	if hasFailure {
		category = failure.Category().String()
	}
	gotErrType, gotErrOK := attrString(span.Attributes(), "error.type")
	switch {
	case hasFailure && !gotErrOK:
		t.Errorf("run span %s error.type absent, want present = %q (S-AGO-021)", runIDStr, category)
	case hasFailure && gotErrType != category:
		t.Errorf("run span %s error.type = %q, want %q", runIDStr, gotErrType, category)
	case !hasFailure && gotErrOK:
		t.Errorf("run span %s error.type = %q, want ABSENT — RunEnd carries no failure (S-AGO-017, S-AGO-021)", runIDStr, gotErrType)
	}

	agoAssertStatus(t, "run", runIDStr, span, hasFailure, category)
}

// agoAssertTurnSpanRow is R-AGO-002's turn row.
func agoAssertTurnSpanRow(t *testing.T, span *tracetest.Span, events []agent.Event) {
	t.Helper()
	turnIDStr, ok := attrString(span.Attributes(), "cachicamas.turn.id")
	if !ok {
		t.Errorf("turn span %q missing cachicamas.turn.id", span.Name())
		return
	}
	turnID := agent.TurnID(turnIDStr)

	var turnEnd agent.TurnEnd
	var runIDForTurn agent.RunID
	haveEnd := false
	for _, ev := range events {
		tid, isTurnEvent := ev.Turn()
		if !isTurnEvent || tid != turnID {
			continue
		}
		if te, isEnd := ev.TurnEnd(); isEnd && !haveEnd {
			turnEnd = te
			runIDForTurn = ev.Run()
			haveEnd = true
		}
	}
	if !haveEnd {
		t.Errorf("turn span %s: no turn_end found for this turn in the drained stream", turnIDStr)
		return
	}

	if got, ok := attrString(span.Attributes(), "cachicamas.run.id"); !ok || got != string(runIDForTurn) {
		t.Errorf("turn span %s cachicamas.run.id = (%q, present=%v), want (%q, true)", turnIDStr, got, ok, runIDForTurn)
	}
	if got, ok := attrString(span.Attributes(), "cachicamas.turn.outcome"); !ok || got != turnEnd.Outcome().String() {
		t.Errorf("turn span %s cachicamas.turn.outcome = (%q, present=%v), want (%q, true)", turnIDStr, got, ok, turnEnd.Outcome().String())
	}

	failure, hasFailure := turnEnd.Failure()
	category := ""
	if hasFailure {
		category = failure.Category().String()
	}
	gotErrType, gotErrOK := attrString(span.Attributes(), "error.type")
	switch {
	case hasFailure && !gotErrOK:
		t.Errorf("turn span %s error.type absent, want present = %q (S-AGO-021)", turnIDStr, category)
	case hasFailure && gotErrType != category:
		t.Errorf("turn span %s error.type = %q, want %q", turnIDStr, gotErrType, category)
	case !hasFailure && gotErrOK:
		t.Errorf("turn span %s error.type = %q, want ABSENT — TurnEnd carries no failure (S-AGO-017, S-AGO-021)", turnIDStr, gotErrType)
	}

	agoAssertStatus(t, "turn", turnIDStr, span, hasFailure, category)
}

// agoAssertToolSpanRow is R-AGO-002's tool row. No scenario this
// correlation covers detaches — cachicamas.tool.detached is therefore
// asserted unconditionally ABSENT here; MAJOR-5's own dedicated proof
// (observability_lifecycle_test.go's detached_wind_down row) covers the
// PRESENT arm.
func agoAssertToolSpanRow(t *testing.T, span *tracetest.Span, events []agent.Event) {
	t.Helper()
	callID, ok := attrString(span.Attributes(), "gen_ai.tool.call.id")
	if !ok {
		t.Errorf("tool span %q missing gen_ai.tool.call.id", span.Name())
		return
	}

	var start agent.ToolStart
	haveStart := false
	var outcomeStr, failureCategory string
	hasFailure, haveEnd := false, false
	for _, ev := range events {
		if ts, isStart := ev.ToolStart(); isStart && ts.CallID() == callID {
			start = ts
			haveStart = true
			continue
		}
		if te, isEnd := ev.ToolEndSuccess(); isEnd && te.CallID() == callID {
			outcomeStr = te.Outcome().String()
			haveEnd = true
			continue
		}
		if te, isEnd := ev.ToolEndResultFailure(); isEnd && te.CallID() == callID {
			outcomeStr = te.Outcome().String()
			haveEnd = true
			continue
		}
		if te, isEnd := ev.ToolEndExecutionFailure(); isEnd && te.CallID() == callID {
			outcomeStr = te.Outcome().String()
			haveEnd = true
			if f, fok := te.Failure(); fok {
				hasFailure = true
				failureCategory = f.Category().String()
			}
			continue
		}
	}
	if !haveStart {
		t.Errorf("tool span %s: no tool_start found for this call in the drained stream", callID)
		return
	}
	if got, ok := attrString(span.Attributes(), "gen_ai.tool.name"); !ok || got != start.Name() {
		t.Errorf("tool span %s gen_ai.tool.name = (%q, present=%v), want (%q, true)", callID, got, ok, start.Name())
	}
	if got, ok := attrInt64(span.Attributes(), "cachicamas.tool.ordinal"); !ok || got != int64(start.Ordinal()) {
		t.Errorf("tool span %s cachicamas.tool.ordinal = (%d, present=%v), want (%d, true)", callID, got, ok, start.Ordinal())
	}
	if !haveEnd {
		t.Errorf("tool span %s: no tool-end event found for this call in the drained stream", callID)
		return
	}
	if got, ok := attrString(span.Attributes(), "cachicamas.tool.outcome"); !ok || got != outcomeStr {
		t.Errorf("tool span %s cachicamas.tool.outcome = (%q, present=%v), want (%q, true)", callID, got, ok, outcomeStr)
	}

	gotErrType, gotErrOK := attrString(span.Attributes(), "error.type")
	switch {
	case hasFailure && !gotErrOK:
		t.Errorf("tool span %s error.type absent, want present = %q (S-AGO-021)", callID, failureCategory)
	case hasFailure && gotErrType != failureCategory:
		t.Errorf("tool span %s error.type = %q, want %q", callID, gotErrType, failureCategory)
	case !hasFailure && gotErrOK:
		t.Errorf("tool span %s error.type = %q, want ABSENT (S-AGO-017, S-AGO-021)", callID, gotErrType)
	}

	if _, gotDetachedOK := attrBool(span.Attributes(), "cachicamas.tool.detached"); gotDetachedOK {
		t.Errorf("tool span %s cachicamas.tool.detached present, want ABSENT — no scenario this correlation drives detaches (S-AGO-017)", callID)
	}

	agoAssertStatus(t, "tool", callID, span, hasFailure, failureCategory)
}

// agoAssertCompactionSpanRow is R-AGO-002's compaction row.
func agoAssertCompactionSpanRow(t *testing.T, span *tracetest.Span, events []agent.Event) {
	t.Helper()
	compactionID, ok := attrString(span.Attributes(), "cachicamas.compaction.id")
	if !ok {
		t.Errorf("compaction span %q missing cachicamas.compaction.id", span.Name())
		return
	}

	var runIDForCompaction agent.RunID
	var turnIDForCompaction agent.TurnID
	haveStarted := false
	var summaryID, failureCategory string
	hasSummary, hasFailure, haveEnd := false, false, false

	for _, ev := range events {
		if started, isStarted := ev.CompactionStarted(); isStarted && started.CompactionID() == compactionID {
			runIDForCompaction = ev.Run()
			if tid, hasTurn := ev.Turn(); hasTurn {
				turnIDForCompaction = tid
			}
			haveStarted = true
			continue
		}
		if finished, isFinished := ev.CompactionFinished(); isFinished && finished.CompactionID() == compactionID {
			summaryID = finished.SummaryID()
			hasSummary = true
			haveEnd = true
			continue
		}
		if failed, isFailed := ev.CompactionFailed(); isFailed && failed.CompactionID() == compactionID {
			if f, fok := failed.Failure(); fok {
				hasFailure = true
				failureCategory = f.Category().String()
			}
			haveEnd = true
			continue
		}
	}
	if !haveStarted {
		t.Errorf("compaction span %s: no compaction_started found in the drained stream", compactionID)
		return
	}
	if !haveEnd {
		t.Errorf("compaction span %s: no compaction_finished or compaction_failed found in the drained stream", compactionID)
		return
	}

	var turnOutcome string
	haveTurnOutcome := false
	for _, ev := range events {
		tid, hasTurn := ev.Turn()
		if !hasTurn || tid != turnIDForCompaction {
			continue
		}
		if te, isEnd := ev.TurnEnd(); isEnd {
			turnOutcome = te.Outcome().String()
			haveTurnOutcome = true
		}
	}

	if got, ok := attrString(span.Attributes(), "cachicamas.run.id"); !ok || got != string(runIDForCompaction) {
		t.Errorf("compaction span %s cachicamas.run.id = (%q, present=%v), want (%q, true)", compactionID, got, ok, runIDForCompaction)
	}
	if got, ok := attrString(span.Attributes(), "cachicamas.turn.id"); !ok || got != string(turnIDForCompaction) {
		t.Errorf("compaction span %s cachicamas.turn.id = (%q, present=%v), want (%q, true)", compactionID, got, ok, turnIDForCompaction)
	}
	if haveTurnOutcome {
		if got, ok := attrString(span.Attributes(), "cachicamas.turn.outcome"); !ok || got != turnOutcome {
			t.Errorf("compaction span %s cachicamas.turn.outcome = (%q, present=%v), want (%q, true)", compactionID, got, ok, turnOutcome)
		}
	}

	gotSummary, gotSummaryOK := attrString(span.Attributes(), "cachicamas.compaction.summary_id")
	switch {
	case hasSummary && !gotSummaryOK:
		t.Errorf("compaction span %s cachicamas.compaction.summary_id absent, want present = %q (S-AGO-017)", compactionID, summaryID)
	case hasSummary && gotSummary != summaryID:
		t.Errorf("compaction span %s cachicamas.compaction.summary_id = %q, want %q", compactionID, gotSummary, summaryID)
	case !hasSummary && gotSummaryOK:
		t.Errorf("compaction span %s cachicamas.compaction.summary_id = %q, want ABSENT — compaction did not finish (S-AGO-017)", compactionID, gotSummary)
	}

	gotErrType, gotErrOK := attrString(span.Attributes(), "error.type")
	switch {
	case hasFailure && !gotErrOK:
		t.Errorf("compaction span %s error.type absent, want present = %q (S-AGO-021)", compactionID, failureCategory)
	case hasFailure && gotErrType != failureCategory:
		t.Errorf("compaction span %s error.type = %q, want %q", compactionID, gotErrType, failureCategory)
	case !hasFailure && gotErrOK:
		t.Errorf("compaction span %s error.type = %q, want ABSENT (S-AGO-017, S-AGO-021)", compactionID, gotErrType)
	}

	agoAssertStatus(t, "compaction", compactionID, span, hasFailure, failureCategory)
}

// errAGOAttributesToolFailure is the plain error scenario D's tool
// returns — mirrors observability_lifecycle_test.go's own
// errAGOLifecycleToolFailure, kept distinct so a failure in one file's
// fixture is never misattributed to the other's.
var errAGOAttributesToolFailure = errAGOAttributesToolFailureType{}

type errAGOAttributesToolFailureType struct{}

func (errAGOAttributesToolFailureType) Error() string {
	return "agent_test: attribute-vocabulary fixture's deliberate tool execution failure"
}

// TestObservability_AttributeVocabulary_SubsetAndValueEquality drives six
// scenarios sharing one tracetest.Provider — a non-delegated tool-calling
// run (A), a finishing compaction (B), a failed run (C), a tool execution
// failure (D), a delegated run (E), and a compaction failure arm (F) —
// so every iff-branch of R-AGO-002's table fires at least once in both
// directions, then correlates every recorded span back to the drained
// event stream generically rather than by hand-picked example.
func TestObservability_AttributeVocabulary_SubsetAndValueEquality(t *testing.T) {
	t.Parallel()

	provider := tracetest.NewProvider()
	var allEvents []agent.Event

	// --- Scenario A: non-delegated, successful, tool-calling run ---
	{
		toolName := "attrs_tool_a"
		tool := EchoScriptedTool(toolName, agent.EffectClassRead)
		reg := agent.NewMapRegistry(map[string]agent.Tool{toolName: tool})
		turnOneScript := scriptToolCallResponse(t, "call-attrs-a-001", toolName, []byte(`{}`))
		turnTwoScript := scriptTextResponse(t, ai.FinishReasonStop)
		modelProvider := agenttest.NewProvider(turnOneScript, turnTwoScript)
		h := agent.Harness{
			Provider:       modelProvider,
			System:         "system prompt for attribute vocabulary scenario A",
			Turn:           agent.TurnOptions{Tools: reg},
			TracerProvider: provider,
		}
		sink := make(chan *agent.Event, 256)
		if _, _, err := h.Run(contextBackground(), firstMessage(t), sink); err != nil {
			t.Fatalf("scenario A Run returned err = %v, want nil", err)
		}
		allEvents = append(allEvents, drainSink(t, sink)...)
	}

	// --- Scenario B: a compaction that FINISHES ---
	{
		h, hist, _ := markedHarnessForCompaction(t, "attrs-b")
		h.TracerProvider = provider
		compactionModelProvider := agenttest.NewProvider(scriptTextResponse(t, ai.FinishReasonStop))
		req := agent.CompactionRequest{Provider: compactionModelProvider, Instruction: "summarize turn one for the attribute vocabulary proof", Cut: hist.Len()}
		sink := make(chan *agent.Event, 64)
		if err := h.Compact(contextBackground(), req, sink); err != nil {
			t.Fatalf("scenario B Compact returned err = %v, want nil", err)
		}
		allEvents = append(allEvents, drainSink(t, sink)...)
	}

	// --- Scenario C: a failed run (mid-stream failure, no tool) ---
	{
		modelProvider := agenttest.NewProvider(scriptTextThenTerminalFailure(t, ai.FailureCategoryUnavailable, true))
		h := agent.Harness{Provider: modelProvider, System: "system prompt for attribute vocabulary scenario C", TracerProvider: provider}
		sink := make(chan *agent.Event, 64)
		if _, _, err := h.Run(contextBackground(), firstMessage(t), sink); err == nil {
			t.Fatal("scenario C Run returned err = nil, want a non-nil error (mid-stream terminal failure)")
		}
		allEvents = append(allEvents, drainSink(t, sink)...)
	}

	// --- Scenario D: a tool execution failure ---
	{
		toolName := "attrs_tool_d"
		tool := &ScriptedTool{
			toolName: toolName,
			Effect:   agent.EffectClassRead,
			Script: func(context.Context, []byte, agent.PolicySlot) (agent.Result, error) {
				return agent.Result{}, errAGOAttributesToolFailure
			},
		}
		reg := agent.NewMapRegistry(map[string]agent.Tool{toolName: tool})
		turnOneScript := scriptToolCallResponse(t, "call-attrs-d-001", toolName, []byte(`{}`))
		turnTwoScript := scriptTextResponse(t, ai.FinishReasonStop)
		modelProvider := agenttest.NewProvider(turnOneScript, turnTwoScript)
		h := agent.Harness{
			Provider:       modelProvider,
			System:         "system prompt for attribute vocabulary scenario D",
			Turn:           agent.TurnOptions{Tools: reg},
			TracerProvider: provider,
		}
		sink := make(chan *agent.Event, 256)
		// The run's own overall error/continuation shape is not this
		// scenario's concern (agoLifecycleToolExecutionFailure's own
		// precedent) — it exists to drive a genuine
		// ToolOutcomeExecutionFailure exit.
		_, _, _ = h.Run(contextBackground(), firstMessage(t), sink)
		allEvents = append(allEvents, drainSink(t, sink)...)
	}

	// --- Scenario E: a delegated run — the child's run.parent_id row ---
	{
		childModelProvider := agenttest.NewProvider(scriptTextResponse(t, ai.FinishReasonStop))
		child := &agent.Harness{
			Provider:       childModelProvider,
			System:         "system prompt for the attribute vocabulary hosted child",
			TracerProvider: provider,
		}
		toolName := "attrs_delegating_tool_e"
		tool := &delegatingTool{toolName: toolName, effect: agent.EffectClassRead, child: child, prompt: firstMessage(t)}
		reg := agent.NewMapRegistry(map[string]agent.Tool{toolName: tool})

		turnOneScript := scriptToolCallResponse(t, "call-attrs-e-001", toolName, []byte(`{}`))
		turnTwoScript := scriptTextResponse(t, ai.FinishReasonStop)
		parentModelProvider := agenttest.NewProvider(turnOneScript, turnTwoScript)
		h := agent.Harness{
			Provider:       parentModelProvider,
			System:         "system prompt for attribute vocabulary scenario E",
			Turn:           agent.TurnOptions{Tools: reg},
			TracerProvider: provider,
		}
		sink := make(chan *agent.Event, 512)
		if _, _, err := h.Run(contextBackground(), firstMessage(t), sink); err != nil {
			t.Fatalf("scenario E parent Run returned err = %v, want nil", err)
		}
		allEvents = append(allEvents, drainSink(t, sink)...)
		allEvents = append(allEvents, tool.ChildEvents()...)
	}

	// --- Scenario F: a compaction failure arm (empty instruction) ---
	{
		h, _, _ := markedHarnessForCompaction(t, "attrs-f")
		h.TracerProvider = provider
		req := agent.CompactionRequest{Provider: agenttest.NewProvider()}
		sink := make(chan *agent.Event, 64)
		if err := h.Compact(contextBackground(), req, sink); err == nil {
			t.Fatal("scenario F Compact returned err = nil, want a non-nil error (empty instruction)")
		}
		allEvents = append(allEvents, drainSink(t, sink)...)
	}

	spans := provider.Spans()
	if len(spans) == 0 {
		t.Fatal("provider recorded zero spans; every assertion below would be vacuous")
	}

	// --- S-AGO-013: the recorded key union is a subset of the closed
	// vocabulary, and no other key appears under any name. ---
	for _, span := range spans {
		for _, kv := range span.Attributes() {
			if !agoAttributeVocabulary[string(kv.Key)] {
				t.Errorf("span %q recorded key %q, which is outside the closed § D3 Layer 2 vocabulary (R-AGO-002, S-AGO-013)", span.Name(), kv.Key)
			}
		}
	}

	// --- S-AGO-014/017/018/021: row-by-row value equality, presence-
	// typed absence and status, correlated per span family. ---
	for _, span := range spans {
		switch {
		case span.Name() == invokeAgentSpanNameForTest:
			agoAssertRunSpanRow(t, span, allEvents)
		case span.Name() == turnSpanNameForTest:
			agoAssertTurnSpanRow(t, span, allEvents)
		case len(span.Name()) >= len(toolSpanNamePrefixForTest) && span.Name()[:len(toolSpanNamePrefixForTest)] == toolSpanNamePrefixForTest:
			agoAssertToolSpanRow(t, span, allEvents)
		case span.Name() == compactionSpanNameForTest:
			agoAssertCompactionSpanRow(t, span, allEvents)
		default:
			t.Errorf("recorded span %q belongs to none of the four families R-AGO-002 names", span.Name())
		}
	}
}
