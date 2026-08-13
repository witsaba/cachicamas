// AG-09 — end-to-end tool dispatch test (loop + scheduler wire-up).
//
// This file is the load-bearing test for S-LSK-008 (one cycle per
// turn) + S-LSK-009 (the AG-09 wire-up: Turn consumes AI-18
// tool-call events and calls Schedule exactly once). It drives
// the loop with a scripted provider that emits one tool-call
// triplet (start, deltas, end) + a `Completion{FinishReason:
// FinishReasonToolCalls}`, asserts that the loop calls
// `Schedule` exactly once, and that the resulting tool events
// are emitted on `sink`.
//
// (The test is in the existing `agent_test` external package
// per NFR-LSK-001; the substrate-untouched test in
// `loop_test.go` widens its filter to include this file.)
package agent_test

import (
	"context"
	"testing"
	"time"

	"github.com/cachicamas/backend/agent/src/agent"
	"github.com/cachicamas/backend/agent/src/agenttest"
	"github.com/cachicamas/backend/agent/src/ai"
)

// S-LSK-008 — AG-09 wire-up: one cycle per turn. Given a provider
// that streams one tool-call triplet + a
// Completion{FinishReasonToolCalls}, when `Turn` runs, the loop
// consumes the AI-18 tool-call events, calls `Schedule` exactly
// once between `provider.Stream` close and `finalize`, and emits
// the AG-05.2 tool events on `sink` in rejoin order.
//
// (The "exactly once" property is asserted by the bite
// S-LSK-008a below; the property test here asserts the
// observable: 3 tool events on the sink for 3 tool calls.)
func TestTurn_ToolDispatch_OneCyclePerTurn(t *testing.T) {
	t.Parallel()

	// 3 scripted tools: 1 read, 1 mutating, 1 execute.
	read := EchoScriptedTool("read_file", agent.EffectClassRead)
	mut := EchoScriptedTool("write_file", agent.EffectClassMutating)
	exec := EchoScriptedTool("run_shell", agent.EffectClassExecute)

	tools := map[string]agent.Tool{
		"read_file":  read,
		"write_file": mut,
		"run_shell":  exec,
	}

	// Build the tool-call triplet. Each call has a different
	// block index (Layer 1's tool-call blocks are stream-wide
	// indexed). The argument bytes are valid JSON so the
	// loop's NewToolCall construction succeeds.
	tc1Start, err := ai.NewToolCallStart(1, "call-1", "read_file")
	if err != nil {
		t.Fatalf("ai.NewToolCallStart: %v", err)
	}
	tc1Delta, err := ai.NewToolCallDelta(1, []byte(`{"path":"x"}`))
	if err != nil {
		t.Fatalf("ai.NewToolCallDelta: %v", err)
	}
	tc1End, err := ai.NewToolCallEnd(1, []byte(`{"path":"x"}`))
	if err != nil {
		t.Fatalf("ai.NewToolCallEnd: %v", err)
	}
	tc2Start, err := ai.NewToolCallStart(2, "call-2", "write_file")
	if err != nil {
		t.Fatalf("ai.NewToolCallStart: %v", err)
	}
	tc2Delta, err := ai.NewToolCallDelta(2, []byte(`{"path":"y","data":"hello"}`))
	if err != nil {
		t.Fatalf("ai.NewToolCallDelta: %v", err)
	}
	tc2End, err := ai.NewToolCallEnd(2, []byte(`{"path":"y","data":"hello"}`))
	if err != nil {
		t.Fatalf("ai.NewToolCallEnd: %v", err)
	}
	tc3Start, err := ai.NewToolCallStart(3, "call-3", "run_shell")
	if err != nil {
		t.Fatalf("ai.NewToolCallStart: %v", err)
	}
	tc3Delta, err := ai.NewToolCallDelta(3, []byte(`{"cmd":"ls"}`))
	if err != nil {
		t.Fatalf("ai.NewToolCallDelta: %v", err)
	}
	tc3End, err := ai.NewToolCallEnd(3, []byte(`{"cmd":"ls"}`))
	if err != nil {
		t.Fatalf("ai.NewToolCallEnd: %v", err)
	}
	completion, err := ai.NewCompletion(ai.FinishReasonToolCalls, ai.Usage{})
	if err != nil {
		t.Fatalf("ai.NewCompletion: %v", err)
	}

	script := agenttest.Script{Steps: []agenttest.Step{
		agenttest.Emit(tc1Start),
		agenttest.Emit(tc1Delta),
		agenttest.Emit(tc1End),
		agenttest.Emit(tc2Start),
		agenttest.Emit(tc2Delta),
		agenttest.Emit(tc2End),
		agenttest.Emit(tc3Start),
		agenttest.Emit(tc3Delta),
		agenttest.Emit(tc3End),
		agenttest.Emit(completion),
	}}

	provider := agenttest.NewProvider(script)
	sink := make(chan *agent.Event, 256)

	_, _, err = agent.Turn(
		context.Background(),
		provider,
		"system for dispatch",
		firstMessageForTest(t),
		agent.TurnOptions{Tools: agent.NewMapRegistry(tools)},
		sink,
	)
	if err != nil {
		t.Fatalf("Turn returned err = %v, want nil", err)
	}

	// All 3 tools should have been invoked exactly once
	// (one cycle per turn, S-LSK-008).
	if got := read.Invocations(); got != 1 {
		t.Errorf("read_file invocations = %d, want 1", got)
	}
	if got := mut.Invocations(); got != 1 {
		t.Errorf("write_file invocations = %d, want 1", got)
	}
	if got := exec.Invocations(); got != 1 {
		t.Errorf("run_shell invocations = %d, want 1", got)
	}

	// Sink should carry: run_start, turn_start, 3 × (tool_start,
	// tool_end_success), turn_end, run_end. The tool events are
	// bracketed by the loop's own brackets.
	events := drainSinkForTest(t, sink)
	var startCount, endCount int
	for _, ev := range events {
		if _, ok := ev.ToolStart(); ok {
			startCount++
		}
		if _, ok := ev.ToolEndSuccess(); ok {
			endCount++
		}
	}
	if startCount != 3 {
		t.Errorf("ToolStart count = %d, want 3 (one cycle per turn, S-LSK-008)", startCount)
	}
	if endCount != 3 {
		t.Errorf("ToolEndSuccess count = %d, want 3 (one cycle per turn, S-LSK-008)", endCount)
	}
}

// S-LSK-008a — RED bite. The one-cycle-per-turn property is
// non-vacuous: a hypothetical `Turn` that re-invokes the
// scheduler after `Schedule` returns would re-run the tools and
// increment the per-tool invocation count past 1. RED-recorded
// BEFORE S-LSK-008 is GREEN.
//
// (The observable: with 1 tool call scripted, the per-tool
// `Invocations()` count is 1. A "re-enter" loop would report 2
// or more — proving the one-cycle invariant is non-vacuous.)
func TestTurn_ToolDispatch_OneCyclePerTurn_BiteReEnter(t *testing.T) {
	t.Parallel()

	read := EchoScriptedTool("read_file", agent.EffectClassRead)
	tc1Start, _ := ai.NewToolCallStart(1, "call-1", "read_file")
	tc1Delta, _ := ai.NewToolCallDelta(1, []byte(`{}`))
	tc1End, _ := ai.NewToolCallEnd(1, []byte(`{}`))
	completion, _ := ai.NewCompletion(ai.FinishReasonToolCalls, ai.Usage{})
	script := agenttest.Script{Steps: []agenttest.Step{
		agenttest.Emit(tc1Start),
		agenttest.Emit(tc1Delta),
		agenttest.Emit(tc1End),
		agenttest.Emit(completion),
	}}

	provider := agenttest.NewProvider(script)
	sink := make(chan *agent.Event, 256)

	_, _, err := agent.Turn(
		context.Background(),
		provider,
		"system for bite",
		firstMessageForTest(t),
		agent.TurnOptions{Tools: agent.NewMapRegistry(map[string]agent.Tool{"read_file": read})},
		sink,
	)
	if err != nil {
		t.Fatalf("Turn returned err = %v, want nil", err)
	}

	if got := read.Invocations(); got > 1 {
		t.Errorf("one-cycle bite DID NOT bite: read_file invocations = %d, want == 1 (S-LSK-008, the wording-trap from 0003:107-112)",
			got)
	}
}

// firstMessageForTest returns a minimal user message for the
// dispatch tests. Lifted from the AG-07 helper to keep this
// file self-contained.
func firstMessageForTest(t *testing.T) []ai.Message {
	t.Helper()
	part, err := ai.NewText("hello dispatch")
	if err != nil {
		t.Fatalf("ai.NewText: %v", err)
	}
	msg, err := ai.NewMessage(ai.RoleUser, part)
	if err != nil {
		t.Fatalf("ai.NewMessage: %v", err)
	}
	return []ai.Message{msg}
}

// drainSinkForTest drains the agent-level event channel into a
// slice. Mirrors `drainSink` in loop_test.go's helpers; the
// external-package import path differs.
func drainSinkForTest(t *testing.T, sink <-chan *agent.Event) []agent.Event {
	t.Helper()
	var got []agent.Event
	deadline := time.After(2 * time.Second)
	for {
		select {
		case ev, ok := <-sink:
			if !ok {
				return got
			}
			got = append(got, *ev)
		case <-deadline:
			t.Fatalf("drainSinkForTest: sink did not close within 2s (%d events received)", len(got))
			return nil
		}
	}
}
