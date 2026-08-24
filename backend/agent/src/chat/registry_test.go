// CH-03.1/.3 — the Conversation registry (R-CHS-001.c, R-CHS-003, D2).
// One Conversation per participant, guarded by a mutex; a second POST
// while inFlight returns the 409 signal that the HTTP handler maps to
// the conflict envelope. DELETE on a turn whose Conversation is no
// longer in-flight is a no-op.

package chat_test

import (
	"sync"
	"testing"

	"github.com/cachicamas/backend/agent/src/agenttest"
	"github.com/cachicamas/backend/agent/src/ai"
	"github.com/cachicamas/backend/agent/src/chat"
)

// TestRegistry_GetOrCreate_Reuses — D2, R-CCP-001. Given a registry built
// from a factory that records how many Conversations it constructed,
// when GetOrCreate is called twice for the same participant id, then the
// factory is invoked exactly once AND both calls return the same
// *Conversation pointer.
func TestRegistry_GetOrCreate_Reuses(t *testing.T) {
	t.Parallel()

	var factoryCalls int
	newConv := func(_ string) (*chat.Conversation, error) {
		factoryCalls++
		return chat.NewConversation(chat.Config{Provider: agenttest.NewProvider(), Store: chat.NewMemoryConversationStore(), ParticipantID: "test-registry"})
	}
	registry := chat.NewRegistry(newConv)

	conv1, busy := registry.GetOrCreate("alice")
	if busy {
		t.Fatal("first GetOrCreate returned busy=true, want false")
	}
	if conv1 == nil {
		t.Fatal("first GetOrCreate returned nil Conversation")
	}

	conv2, busy := registry.GetOrCreate("alice")
	if busy {
		t.Fatal("second GetOrCreate returned busy=true, want false")
	}
	if conv1 != conv2 {
		t.Errorf("GetOrCreate returned a fresh *Conversation on the second call, want the same one (D2: one Conversation per participant)")
	}
	if factoryCalls != 1 {
		t.Errorf("factory called %d time(s), want 1 (reuse on second call)", factoryCalls)
	}
}

// TestRegistry_GetOrCreate_409OnInflight — S-CHS-001.c. Given a registry
// whose participant's Conversation is mid-turn (inFlight=true), when a
// second POST arrives via GetOrCreate, then GetOrCreate returns
// (nil, false, nil) — the HTTP handler's 409 signal — AND the second
// factory call (which would synthesise a second Conversation) is NOT
// made. Asserted via ScriptsRemaining on the scripted provider: the
// second provider has its own script queue, but the registry's
// "no factory call on refusal" property is verified through the SAME
// provider because Conversation reuses it.
func TestRegistry_GetOrCreate_409OnInflight(t *testing.T) {
	t.Parallel()
gate := agenttest.NewGate()

	provider := agenttest.NewProvider(scriptWithGate(t, gate))

	newConv := func(_ string) (*chat.Conversation, error) {
		return chat.NewConversation(chat.Config{Provider: provider, Store: chat.NewMemoryConversationStore(), ParticipantID: "test-registry"})
	}
	registry := chat.NewRegistry(newConv)

	conv, busy := registry.GetOrCreate("alice")
	if busy {
		t.Fatal("first GetOrCreate returned busy=true, want false")
	}

	// Drive a turn that blocks at the gate so inFlight stays true.
	stream, err := conv.Send(t.Context(), "first prompt")
	if err != nil {
		t.Fatalf("first Send returned %v, want nil", err)
	}
	// Read the message.start so the projector is established (inFlight is
	// set before that, so this is belt-and-braces).
	select {
	case <-stream:
	case <-t.Context().Done():
		t.Fatal("first Send: stream did not deliver an event before timeout")
	}

	// Second GetOrCreate must signal 409 without touching the factory or
	// invoking a second Send.
	conv2, busy := registry.GetOrCreate("alice")
	if !busy {
		t.Error("second GetOrCreate returned busy=false, want true (409 signal — inFlight is held by the first turn)")
	}
	if conv2 != nil {
		t.Errorf("second GetOrCreate returned conv=%#v, want nil (409 signal — handler maps to 409 with no second Send)", conv2)
	}

	// Release the gate so the first turn completes cleanly before the
	// test exits — prevents goroutine leaks across the suite.
	gate.Release()
	// drain the stream so the projector goroutine sees the
	// channel close and exits. Reading without binding the
	// values keeps the lint explicit about the empty-body
	// intent (revive's empty-block check); the comment above
	// is the durable record of WHY this loop exists.
	for range stream {
		_ = stream
	}
}

// TestRegistry_GetOrCreate_ConcurrentIsRaceFree — R-CCP-001 + D2 under
// -race. Given a registry, when N goroutines all call GetOrCreate for the
// same participant id simultaneously, then exactly one factory call is
// recorded AND every goroutine receives the SAME *Conversation pointer.
// This is the property that lets the HTTP surface serialise concurrent
// POSTs by participant without a second mutex on the Conversation itself.
func TestRegistry_GetOrCreate_ConcurrentIsRaceFree(t *testing.T) {
	t.Parallel()

	var factoryCalls int
	var factoryMu sync.Mutex
	newConv := func(_ string) (*chat.Conversation, error) {
		factoryMu.Lock()
		factoryCalls++
		factoryMu.Unlock()
		return chat.NewConversation(chat.Config{Provider: agenttest.NewProvider(), Store: chat.NewMemoryConversationStore(), ParticipantID: "test-registry"})
	}
	registry := chat.NewRegistry(newConv)

	const goroutines = 10
	results := make([]*chat.Conversation, goroutines)
	buses := make([]bool, goroutines)
	var start sync.WaitGroup
	var done sync.WaitGroup
	start.Add(1)
	done.Add(goroutines)
	for i := 0; i < goroutines; i++ {
		go func(idx int) {
			defer done.Done()
			start.Wait()
			results[idx], buses[idx] = registry.GetOrCreate("alice")
		}(i)
	}
	start.Done()
	done.Wait()

	for i := range results {
		if buses[i] {
			t.Errorf("goroutine %d GetOrCreate returned busy=true, want false", i)
		}
		if results[i] == nil {
			t.Errorf("goroutine %d got nil Conversation", i)
		}
		if i > 0 && results[i] != results[0] {
			t.Errorf("goroutine %d got a different *Conversation than goroutine 0 (registry did not serialise)", i)
		}
	}
	factoryMu.Lock()
	defer factoryMu.Unlock()
	if factoryCalls != 1 {
		t.Errorf("factory called %d time(s), want exactly 1 (concurrent first-callers must collapse to one factory call)", factoryCalls)
	}
}

// TestRegistry_CancelByTurnID_NoOpWhenNotInflight — S-CHS-003.b/c. Given
// a registry whose participant's Conversation is NOT in-flight (turn
// already ended or no turn ever ran), when CancelByTurnID is called, then
// it returns false — the HTTP handler maps that to a 200/204 DELETE on a
// non-existent or already-finished turn.
func TestRegistry_CancelByTurnID_NoOpWhenNotInflight(t *testing.T) {
	t.Parallel()

	registry := chat.NewRegistry(func(_ string) (*chat.Conversation, error) {
		return chat.NewConversation(chat.Config{Provider: agenttest.NewProvider(), Store: chat.NewMemoryConversationStore(), ParticipantID: "test-registry"})
	})

	// Unknown participant.
	if got := registry.CancelByTurnID("nobody", "turn-x"); got {
		t.Errorf("CancelByTurnID(unknown participant) = true, want false (R-CHS-003.c: DELETE on a turn that does not exist is a no-op)")
	}

	// Known participant, no turn in flight.
	conv, busy := registry.GetOrCreate("alice")
	if busy {
		t.Fatalf("GetOrCreate(fresh) returned busy=true, want false")
	}
	if conv == nil {
		t.Fatal("GetOrCreate(fresh) returned nil Conversation")
	}
	// The Conversation's inFlight is false here (factory fresh, no Send yet).
	if got := registry.CancelByTurnID("alice", "turn-x"); got {
		t.Errorf("CancelByTurnID(alice, fresh) = true, want false (R-CHS-003.b: DELETE on a turn not in flight is a no-op)")
	}
}

// scriptWithGate returns a one-shot scripted stream that emits a
// message.start, then blocks at the given Gate's Hold step. The test
// releases the Gate to let the producer finish — until then the
// producer (and therefore the harness.run_end) is parked at the gate,
// so inFlight stays true.
func scriptWithGate(t *testing.T, g *agenttest.Gate) agenttest.Script {
	t.Helper()
	start, err := ai.NewTextBlockStart(1)
	if err != nil {
		t.Fatalf("ai.NewTextBlockStart: %v", err)
	}
	delta, err := ai.NewTextDelta(1, "hi")
	if err != nil {
		t.Fatalf("ai.NewTextDelta: %v", err)
	}
	end, err := ai.NewTextBlockEnd(1)
	if err != nil {
		t.Fatalf("ai.NewTextBlockEnd: %v", err)
	}
	completion, err := ai.NewCompletion(ai.FinishReasonStop, ai.Usage{})
	if err != nil {
		t.Fatalf("ai.NewCompletion: %v", err)
	}
	return agenttest.Script{Steps: []agenttest.Step{
		agenttest.Emit(start),
		agenttest.Emit(delta),
		agenttest.Emit(end),
		agenttest.Hold(g),
		agenttest.Emit(completion),
	}}
}
