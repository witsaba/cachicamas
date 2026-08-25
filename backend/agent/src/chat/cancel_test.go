// CH-02.2 — cancelling a turn in flight (R-CCP-006). Every scenario lives in
// package chat_test.
package chat_test

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/cachicamas/backend/agent/src/agenttest"
	"github.com/cachicamas/backend/agent/src/agent"
	"github.com/cachicamas/backend/agent/src/ai"
	"github.com/cachicamas/backend/agent/src/chat"
)

// TestCancel_MidStream — S-CCP-050, S-CCP-051, S-CCP-054. Given a turn held
// mid-stream at a scripted test gate after two fragments have been
// projected, when the conversation is asked to cancel that turn, then the
// projected stream carries exactly one terminal event, whose FinishReason
// is absent, and the two message.delta events already projected are
// present and unchanged.
func TestCancel_MidStream(t *testing.T) {
	t.Parallel()

	gate := agenttest.NewGate()
	provider := agenttest.NewProvider(heldAfterTwoFragmentsScript(t, gate))

	conv, err := chat.NewConversation(chat.Config{Provider: provider, Store: chat.NewMemoryConversationStore(), ParticipantID: "test-cancel",
		ToolSource: chat.FromAgentRegistry(agent.NewMapRegistry(nil)), PermissionPolicy: chat.NewDefaultPermissionPolicy(nil)})
	if err != nil {
		t.Fatalf("chat.NewConversation returned %v, want nil", err)
	}

	out, err := conv.Send(context.Background(), "hello")
	if err != nil {
		t.Fatalf("Send returned %v, want nil", err)
	}

	first := readN(t, out, 3) // message.start + 2 deltas
	<-gate.Reached()

	if outcome := conv.Cancel(); outcome != chat.CancelRequested {
		t.Errorf("Cancel() = %v, want CancelRequested", outcome)
	}

	rest := drainWire(t, out)
	all := append(first, rest...)

	var terminalCount int
	var deltas []chat.MessageDelta
	for _, ev := range all {
		switch e := ev.(type) {
		case chat.TurnEnd:
			terminalCount++
		case chat.Error:
			terminalCount++
		case chat.MessageDelta:
			deltas = append(deltas, e)
		}
	}
	if terminalCount != 1 {
		t.Fatalf("terminal event count = %d, want exactly 1", terminalCount)
	}
	turnEnd, ok := all[len(all)-1].(chat.TurnEnd)
	if !ok {
		t.Fatalf("terminal event = %#v, want chat.TurnEnd", all[len(all)-1])
	}
	if turnEnd.FinishReason != nil {
		t.Errorf("TurnEnd.FinishReason = %q, want absent (nil) on a cancelled turn", *turnEnd.FinishReason)
	}

	wantDeltas := []string{"alpha", "beta"}
	if len(deltas) != len(wantDeltas) {
		t.Fatalf("delta count = %d, want %d (the two already projected before cancellation, unchanged)", len(deltas), len(wantDeltas))
	}
	for i, d := range deltas {
		if d.Delta != wantDeltas[i] {
			t.Errorf("delta[%d] = %q, want %q", i, d.Delta, wantDeltas[i])
		}
	}
}

// TestCancel_AlreadyFinished — S-CCP-053. Given a turn that has already
// reached its terminal event, when the conversation is asked to cancel it,
// then the outcome reported is a no-op distinguishable from both success
// and error.
func TestCancel_AlreadyFinished(t *testing.T) {
	t.Parallel()

	provider := agenttest.NewProvider(scriptTextResponse(t, 1, []string{"done"}, ai.FinishReasonStop))
	conv, err := chat.NewConversation(chat.Config{Provider: provider, Store: chat.NewMemoryConversationStore(), ParticipantID: "test-cancel",
		ToolSource: chat.FromAgentRegistry(agent.NewMapRegistry(nil)), PermissionPolicy: chat.NewDefaultPermissionPolicy(nil)})
	if err != nil {
		t.Fatalf("chat.NewConversation returned %v, want nil", err)
	}

	out, err := conv.Send(context.Background(), "hi")
	if err != nil {
		t.Fatalf("Send returned %v, want nil", err)
	}
	if !endsWithTurnEnd(drainWire(t, out)) {
		t.Fatal("turn did not terminate with a turn.end before Cancel was called")
	}

	if outcome := conv.Cancel(); outcome != chat.CancelNoOp {
		t.Errorf("Cancel() (after completion) = %v, want CancelNoOp", outcome)
	}
}

// TestCancel_ConcurrentWithConsumption — S-CCP-055. Given the merged tree
// with the race detector enabled, when a cancellation is requested
// concurrently with stream consumption, then the suite reports no data
// race and the stream still terminates exactly once.
func TestCancel_ConcurrentWithConsumption(t *testing.T) {
	t.Parallel()

	gate := agenttest.NewGate()
	provider := agenttest.NewProvider(heldAfterTwoFragmentsScript(t, gate))
	conv, err := chat.NewConversation(chat.Config{Provider: provider, Store: chat.NewMemoryConversationStore(), ParticipantID: "test-cancel",
		ToolSource: chat.FromAgentRegistry(agent.NewMapRegistry(nil)), PermissionPolicy: chat.NewDefaultPermissionPolicy(nil)})
	if err != nil {
		t.Fatalf("chat.NewConversation returned %v, want nil", err)
	}

	out, err := conv.Send(context.Background(), "hi")
	if err != nil {
		t.Fatalf("Send returned %v, want nil", err)
	}

	var wg sync.WaitGroup
	wg.Add(2)
	var events []chat.WireEvent
	go func() {
		defer wg.Done()
		events = drainWire(t, out)
	}()
	go func() {
		defer wg.Done()
		<-gate.Reached()
		conv.Cancel()
	}()
	wg.Wait()

	var terminalCount int
	for _, ev := range events {
		switch ev.(type) {
		case chat.TurnEnd, chat.Error:
			terminalCount++
		}
	}
	if terminalCount != 1 {
		t.Errorf("terminal event count = %d, want exactly 1", terminalCount)
	}
}

// heldAfterTwoFragmentsScript builds a scripted stream that emits a
// message-start and two deltas, then blocks the producer at gate before
// ever closing the text block — the fixture CH-02.2's mid-stream-cancel
// scenarios drive: two fragments already projected, one blocked.
func heldAfterTwoFragmentsScript(t *testing.T, gate *agenttest.Gate) agenttest.Script {
	t.Helper()

	start, err := ai.NewTextBlockStart(1)
	if err != nil {
		t.Fatalf("ai.NewTextBlockStart returned %v, want no failure", err)
	}
	delta1, err := ai.NewTextDelta(1, "alpha")
	if err != nil {
		t.Fatalf("ai.NewTextDelta returned %v, want no failure", err)
	}
	delta2, err := ai.NewTextDelta(1, "beta")
	if err != nil {
		t.Fatalf("ai.NewTextDelta returned %v, want no failure", err)
	}
	end, err := ai.NewTextBlockEnd(1)
	if err != nil {
		t.Fatalf("ai.NewTextBlockEnd returned %v, want no failure", err)
	}
	completion, err := ai.NewCompletion(ai.FinishReasonStop, ai.Usage{})
	if err != nil {
		t.Fatalf("ai.NewCompletion returned %v, want no failure", err)
	}
	return agenttest.Script{Steps: []agenttest.Step{
		agenttest.Emit(start),
		agenttest.Emit(delta1),
		agenttest.Emit(delta2),
		agenttest.Hold(gate),
		agenttest.Emit(end),
		agenttest.Emit(completion),
	}}
}

// readN reads exactly n events off out, bounded by a 5s deadline.
func readN(t *testing.T, out <-chan chat.WireEvent, n int) []chat.WireEvent {
	t.Helper()

	got := make([]chat.WireEvent, 0, n)
	deadline := time.After(5 * time.Second)
	for len(got) < n {
		select {
		case ev, ok := <-out:
			if !ok {
				t.Fatalf("readN: channel closed early after %d/%d events", len(got), n)
				return nil
			}
			got = append(got, ev)
		case <-deadline:
			t.Fatalf("readN: did not receive %d events within 5s (got %d)", n, len(got))
			return nil
		}
	}
	return got
}
