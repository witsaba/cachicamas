// CH-02.1 — the conversation and its projection onto the wire vocabulary
// (R-CCP-001..005). Every scenario lives in package chat_test, proving the
// archetype's behaviour from outside the package, with no reach into
// unexported surface.
package chat_test

import (
	"context"
	"log/slog"
	"reflect"
	"sync"
	"testing"
	"time"

	"github.com/cachicamas/backend/agent/src/agenttest"
	"github.com/cachicamas/backend/agent/src/ai"
	"github.com/cachicamas/backend/agent/src/chat"
)

// TestConversation_SystemPrompt — S-CCP-010. Given the merged tree, when the
// archetype's system-prompt constant is read, then its value is
// byte-identical to the literal R-CCP-002 pins (D13).
func TestConversation_SystemPrompt(t *testing.T) {
	t.Parallel()

	const want = "You are the cachicamas chat assistant; answer the participant in plain, well-formatted text."
	if chat.SystemPrompt != want {
		t.Errorf("chat.SystemPrompt = %q, want %q", chat.SystemPrompt, want)
	}
}

// TestConversation_DrivesOneTurn — S-CCP-001, S-CCP-002 (turn one half),
// S-CCP-003, S-CCP-004, S-CCP-012, S-CCP-020..023, S-CCP-040, S-CCP-041.
// Given a conversation built from a scripted provider that answers in three
// fragments, when one prompt is driven to completion, then the projected
// stream opens the message once, carries the three fragments in the
// provider's own order, closes the message once with FinishReason "stop",
// and terminates with exactly one turn.end carrying FinishReason "stop" —
// and the provider recorded exactly one invocation, with the pinned system
// prompt.
func TestConversation_DrivesOneTurn(t *testing.T) {
	t.Parallel()

	fragments := []string{"alpha", "beta", "gamma"}
	provider := agenttest.NewProvider(scriptTextResponse(t, 1, fragments, ai.FinishReasonStop))

	conv, err := chat.NewConversation(chat.Config{Provider: provider})
	if err != nil {
		t.Fatalf("chat.NewConversation returned %v, want nil", err)
	}

	out, err := conv.Send(context.Background(), "hello")
	if err != nil {
		t.Fatalf("Send returned %v, want nil", err)
	}
	events := drainWire(t, out)

	var starts, ends int
	var deltaTexts []string
	var messageIndexesSeen []int
	for _, ev := range events {
		switch e := ev.(type) {
		case chat.MessageStart:
			starts++
			messageIndexesSeen = append(messageIndexesSeen, e.Index)
		case chat.MessageDelta:
			deltaTexts = append(deltaTexts, e.Delta)
			messageIndexesSeen = append(messageIndexesSeen, e.Index)
		case chat.MessageEnd:
			ends++
			messageIndexesSeen = append(messageIndexesSeen, e.Index)
			if e.FinishReason != "stop" {
				t.Errorf("MessageEnd.FinishReason = %q, want %q", e.FinishReason, "stop")
			}
		}
	}

	if starts != 1 {
		t.Errorf("message.start count = %d, want 1", starts)
	}
	if ends != 1 {
		t.Errorf("message.end count = %d, want 1", ends)
	}
	if !reflect.DeepEqual(deltaTexts, fragments) {
		t.Errorf("delta fragments = %v, want %v in provider order", deltaTexts, fragments)
	}
	for _, idx := range messageIndexesSeen {
		if idx != 0 {
			t.Errorf("message index = %d, want 0 for v1's single-message turn (S-CCP-040)", idx)
		}
	}

	if len(events) == 0 {
		t.Fatal("no wire events observed")
	}
	turnEnd, ok := events[len(events)-1].(chat.TurnEnd)
	if !ok {
		t.Fatalf("terminal event = %#v, want chat.TurnEnd", events[len(events)-1])
	}
	if turnEnd.FinishReason == nil {
		t.Fatal("TurnEnd.FinishReason is absent, want present (\"stop\") on a completed turn (S-CCP-052 negative control)")
	}
	if *turnEnd.FinishReason != "stop" {
		t.Errorf("TurnEnd.FinishReason = %q, want %q", *turnEnd.FinishReason, "stop")
	}

	// S-CCP-023: TurnEnd carries no usage field at all — a structural
	// guarantee of the wire.go type itself, asserted here by construction
	// rather than by a runtime field check.

	requests := provider.Requests()
	if len(requests) != 1 {
		t.Fatalf("provider recorded %d invocation(s), want exactly 1", len(requests))
	}

	// S-CCP-012: the system prompt the provider received is the pinned literal.
	wantSystem, serr := ai.NewSystemText(chat.SystemPrompt)
	if serr != nil {
		t.Fatalf("ai.NewSystemText returned %v, want nil", serr)
	}
	gotSystem, hasSystem := requests[0].SystemInstruction()
	if !hasSystem || !gotSystem.Equal(wantSystem) {
		t.Errorf("request system instruction = %v (present=%v), want %v", gotSystem, hasSystem, wantSystem)
	}
}

// TestConversation_TwoTurnContinuation — S-CCP-002 (turn two half), S-CCP-001
// applied per turn, S-CCP-054. Given a conversation that has completed one
// turn, when a second prompt is driven on the same conversation, then the
// second turn's request carries the first turn's user prompt and assistant
// reply, read off the scripted provider's own recorded requests, and both
// turns reach their own complete terminal bracket.
func TestConversation_TwoTurnContinuation(t *testing.T) {
	t.Parallel()

	script1 := scriptTextResponse(t, 1, []string{"first-reply"}, ai.FinishReasonStop)
	script2 := scriptTextResponse(t, 1, []string{"second-reply"}, ai.FinishReasonStop)
	provider := agenttest.NewProvider(script1, script2)

	conv, err := chat.NewConversation(chat.Config{Provider: provider})
	if err != nil {
		t.Fatalf("chat.NewConversation returned %v, want nil", err)
	}

	out1, err := conv.Send(context.Background(), "turn one prompt")
	if err != nil {
		t.Fatalf("Send (turn one) returned %v, want nil", err)
	}
	events1 := drainWire(t, out1)
	if !endsWithTurnEnd(events1) {
		t.Fatal("turn one: stream did not terminate with a turn.end")
	}

	out2, err := conv.Send(context.Background(), "turn two prompt")
	if err != nil {
		t.Fatalf("Send (turn two) returned %v, want nil", err)
	}
	events2 := drainWire(t, out2)
	if !endsWithTurnEnd(events2) {
		t.Fatal("turn two: stream did not terminate with a turn.end")
	}

	requests := provider.Requests()
	if len(requests) != 2 {
		t.Fatalf("provider recorded %d invocation(s), want exactly 2", len(requests))
	}

	var sawUserPromptOne, sawAssistantReplyOne bool
	for _, m := range requests[1].Messages() {
		for _, part := range m.Content() {
			text, ok := part.Text()
			if !ok {
				continue
			}
			if m.Role() == ai.RoleUser && text == "turn one prompt" {
				sawUserPromptOne = true
			}
			if m.Role() == ai.RoleAssistant && text == "first-reply" {
				sawAssistantReplyOne = true
			}
		}
	}
	if !sawUserPromptOne {
		t.Error("turn two's request does not carry turn one's user prompt (D0: History must be reused)")
	}
	if !sawAssistantReplyOne {
		t.Error("turn two's request does not carry turn one's assistant reply (D0: History must be reused)")
	}
}

// TestConversation_MessageIndex — S-CCP-025. Given a run whose stream
// carries a message_delta_text whose Layer 2 fragment index is non-zero,
// when the corresponding message.delta is read, then its index equals the
// archetype's per-message counter (0) and not the fragment index — so a
// projection that passed Layer 2's idx through FAILS.
func TestConversation_MessageIndex(t *testing.T) {
	t.Parallel()

	const nonZeroFragmentIdx = 7
	script := scriptTextResponse(t, nonZeroFragmentIdx, []string{"hello"}, ai.FinishReasonStop)
	provider := agenttest.NewProvider(script)

	conv, err := chat.NewConversation(chat.Config{Provider: provider})
	if err != nil {
		t.Fatalf("chat.NewConversation returned %v, want nil", err)
	}

	out, err := conv.Send(context.Background(), "hi")
	if err != nil {
		t.Fatalf("Send returned %v, want nil", err)
	}
	events := drainWire(t, out)

	var sawStart, sawDelta bool
	for _, ev := range events {
		switch e := ev.(type) {
		case chat.MessageStart:
			sawStart = true
			if e.Index != 0 {
				t.Errorf("MessageStart.Index = %d, want 0 (archetype counter, not L2 fragment index %d)", e.Index, nonZeroFragmentIdx)
			}
		case chat.MessageDelta:
			sawDelta = true
			if e.Index != 0 {
				t.Errorf("MessageDelta.Index = %d, want 0 (archetype counter, not L2 fragment index %d)", e.Index, nonZeroFragmentIdx)
			}
		}
	}
	if !sawStart || !sawDelta {
		t.Fatalf("sawStart=%v sawDelta=%v, want both true — the assertions above never ran", sawStart, sawDelta)
	}
}

// TestConversation_UnmappedEvents — S-CCP-030, S-CCP-031, S-CCP-034. Given a
// conversation constructed with a capturing slog handler, and a run whose
// stream inherently carries run_start, turn_start and turn_end, when the
// projection runs, then for each of those occurrences exactly one captured
// record exists whose attributes carry that kind's own String() token; the
// projected wire stream carries only message.start/delta/end and one
// terminal event; and the run completed normally with no wire error.
func TestConversation_UnmappedEvents(t *testing.T) {
	t.Parallel()

	provider := agenttest.NewProvider(scriptTextResponse(t, 1, []string{"hi"}, ai.FinishReasonStop))
	handler := newCapturingHandler()

	conv, err := chat.NewConversation(chat.Config{Provider: provider, Logger: slog.New(handler)})
	if err != nil {
		t.Fatalf("chat.NewConversation returned %v, want nil", err)
	}

	out, err := conv.Send(context.Background(), "hello")
	if err != nil {
		t.Fatalf("Send returned %v, want nil", err)
	}
	events := drainWire(t, out)

	for _, kind := range []string{"run_start", "turn_start", "turn_end"} {
		if got := handler.countFor(kind); got != 1 {
			t.Errorf("captured records for kind %q = %d, want exactly 1", kind, got)
		}
	}

	for _, ev := range events {
		switch ev.(type) {
		case chat.MessageStart, chat.MessageDelta, chat.MessageEnd, chat.TurnEnd:
			// mapped shapes only — an unmapped kind must never reach here.
		default:
			t.Errorf("unexpected wire event %#v projected — an unmapped kind must produce zero wire output", ev)
		}
	}

	if len(events) == 0 {
		t.Fatal("no wire events observed")
	}
	if _, ok := events[len(events)-1].(chat.TurnEnd); !ok {
		t.Errorf("terminal event = %#v, want chat.TurnEnd (run completed normally, no wire error)", events[len(events)-1])
	}
}

// scriptTextResponse builds a deterministic scripted text-only stream: one
// text block opened at blockIdx, one delta per fragment (in order), the
// block closed, then completion. Mirrors package agent_test's own
// loop_test.go:scriptTextResponse precedent, reproduced here rather than
// imported because that helper is unexported to a sibling test package.
func scriptTextResponse(t *testing.T, blockIdx ai.BlockIndex, fragments []string, finishReason ai.FinishReason) agenttest.Script {
	t.Helper()

	start, err := ai.NewTextBlockStart(blockIdx)
	if err != nil {
		t.Fatalf("ai.NewTextBlockStart returned %v, want no failure", err)
	}
	steps := []agenttest.Step{agenttest.Emit(start)}
	for _, fragment := range fragments {
		delta, derr := ai.NewTextDelta(blockIdx, fragment)
		if derr != nil {
			t.Fatalf("ai.NewTextDelta returned %v, want no failure", derr)
		}
		steps = append(steps, agenttest.Emit(delta))
	}
	end, err := ai.NewTextBlockEnd(blockIdx)
	if err != nil {
		t.Fatalf("ai.NewTextBlockEnd returned %v, want no failure", err)
	}
	completion, err := ai.NewCompletion(finishReason, ai.Usage{})
	if err != nil {
		t.Fatalf("ai.NewCompletion returned %v, want no failure", err)
	}
	steps = append(steps, agenttest.Emit(end), agenttest.Emit(completion))
	return agenttest.Script{Steps: steps}
}

// drainWire drains a WireEvent stream into a slice, stopping on channel
// close, bounded by a 5s deadline — mirrors package agent_test's own
// loop_test.go:drainSink precedent.
func drainWire(t *testing.T, out <-chan chat.WireEvent) []chat.WireEvent {
	t.Helper()

	var got []chat.WireEvent
	deadline := time.After(5 * time.Second)
	for {
		select {
		case ev, ok := <-out:
			if !ok {
				return got
			}
			got = append(got, ev)
		case <-deadline:
			t.Fatalf("drainWire: channel did not close within 5s (%d event(s) received so far)", len(got))
			return nil
		}
	}
}

// endsWithTurnEnd reports whether events' last element is a chat.TurnEnd —
// the "one complete terminal bracket" shape S-CCP-001/S-CCP-054 require.
func endsWithTurnEnd(events []chat.WireEvent) bool {
	if len(events) == 0 {
		return false
	}
	_, ok := events[len(events)-1].(chat.TurnEnd)
	return ok
}

// capturingHandler is a minimal slog.Handler capturing every record it
// receives, guarded by its own mutex (concurrent-safe: the projector
// goroutine is the only writer in practice, but a mutex costs nothing and
// keeps -race silent regardless of future callers).
type capturingHandler struct {
	mu      sync.Mutex
	records []capturedRecord
}

type capturedRecord struct {
	message string
	kind    string
}

func newCapturingHandler() *capturingHandler { return &capturingHandler{} }

func (h *capturingHandler) Enabled(context.Context, slog.Level) bool { return true }

func (h *capturingHandler) Handle(_ context.Context, r slog.Record) error {
	h.mu.Lock()
	defer h.mu.Unlock()
	rec := capturedRecord{message: r.Message}
	r.Attrs(func(a slog.Attr) bool {
		if a.Key == "kind" {
			rec.kind = a.Value.String()
		}
		return true
	})
	h.records = append(h.records, rec)
	return nil
}

func (h *capturingHandler) WithAttrs(_ []slog.Attr) slog.Handler { return h }
func (h *capturingHandler) WithGroup(_ string) slog.Handler      { return h }

// countFor returns how many captured records carry a "kind" attribute equal
// to kindToken — matching an identity (the kind's own String() token),
// never a shape (NFR-CCP-003).
func (h *capturingHandler) countFor(kindToken string) int {
	h.mu.Lock()
	defer h.mu.Unlock()
	n := 0
	for _, r := range h.records {
		if r.kind == kindToken {
			n++
		}
	}
	return n
}
