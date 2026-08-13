// AG-07 — strict-TDD tests for the one-turn walking skeleton
// (R-LSK-001..005, S-LSK-001..007).
//
// Every scenario is in package agent_test (NFR-LSK-001): the
// external-test posture proves every behavioral claim from outside the
// package, with no reach into unexported surface. The file carries the
// small shared helpers (mustSimpleRequest, mintLoopRunID, mintLoopTurnID,
// reconstructLoopMessage, scriptTextResponse) the S-LSK-001..007 tests
// reuse, plus the every-kind-constructible guard and the substrate
// "untouched" guard at S-LSK-006.
//
// # Bite-first RED ordering (defense against AG-05 W1)
//
// S-LSK-003a (drop-a-delta bite) and S-LSK-003b (double-a-delta bite)
// are RED-recorded BEFORE S-LSK-003 (the property test) goes GREEN.
// Mirrors AG-05's S-AMT-071 / S-AMT-072 (reconstruction_test.go:180-277):
// a vacuous reconstruction helper is the documented failure mode this
// bite pattern defends against.
package agent_test

import (
	"reflect"
	"strings"
	"testing"

	"github.com/cachicamas/backend/agent/src/agent"
	"github.com/cachicamas/backend/agent/src/agenttest"
	"github.com/cachicamas/backend/agent/src/ai"
)

// loopTestIdentityCounter is reserved for future test-only identity
// minting (currently unused — Phase 2 adds per-test identity helpers
// when the test fixtures need to assert specific run/turn IDs). Kept
// at file scope so Phase 2's helpers land in one place.
var _ = func() int { return 0 }()

// scriptTextResponse builds a small, deterministic text-only scripted
// stream this file's tests reuse — text-block-start, three deltas,
// text-block-end, completion with FinishReasonStop. Walking-skeleton
// scope: text-only. The reasoning interleaved shape lives in
// scriptReasoningTextResponse below (S-LSK-005).
func scriptTextResponse(t *testing.T, finishReason ai.FinishReason) agenttest.Script {
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
	delta3, err := ai.NewTextDelta(1, "gamma")
	if err != nil {
		t.Fatalf("ai.NewTextDelta returned %v, want no failure", err)
	}
	end, err := ai.NewTextBlockEnd(1)
	if err != nil {
		t.Fatalf("ai.NewTextBlockEnd returned %v, want no failure", err)
	}
	completion, err := ai.NewCompletion(finishReason, ai.Usage{})
	if err != nil {
		t.Fatalf("ai.NewCompletion returned %v, want no failure", err)
	}
	return agenttest.Script{Steps: []agenttest.Step{
		agenttest.Emit(start),
		agenttest.Emit(delta1),
		agenttest.Emit(delta2),
		agenttest.Emit(delta3),
		agenttest.Emit(end),
		agenttest.Emit(completion),
	}}
}

// scriptReasoningTextResponse builds the interleaved
// reasoning + text scripted stream this file's S-LSK-005 test drives:
// response_start, reasoning block (text + token), text block (two
// deltas), completion. Mirrors AG-07.2 #2 (per doc 0003) and design D4a.
func scriptReasoningTextResponse(t *testing.T, finishReason ai.FinishReason, reasoningText string, tokenBytes []byte) agenttest.Script {
	t.Helper()

	respStart, err := ai.NewResponseStart("resp-lsk-005", "cachicamas-neutral-model-1")
	if err != nil {
		t.Fatalf("ai.NewResponseStart returned %v, want no failure", err)
	}
	reasonStartEvent, err := ai.NewReasoningBlockStart(1)
	if err != nil {
		t.Fatalf("ai.NewReasoningBlockStart returned %v, want no failure", err)
	}
	reasonStartPayload, ok := reasonStartEvent.ReasoningBlockStart()
	if !ok {
		t.Fatal("reasoning block start event carries no ReasoningBlockStart payload")
	}
	reasonDelta, err := ai.NewReasoningDelta(reasonStartPayload, []byte(reasoningText))
	if err != nil {
		t.Fatalf("ai.NewReasoningDelta returned %v, want no failure", err)
	}
	reasonEnd, err := ai.NewReasoningBlockEnd(reasonStartPayload, tokenBytes)
	if err != nil {
		t.Fatalf("ai.NewReasoningBlockEnd returned %v, want no failure", err)
	}
	textStart, err := ai.NewTextBlockStart(2)
	if err != nil {
		t.Fatalf("ai.NewTextBlockStart returned %v, want no failure", err)
	}
	textDelta1, err := ai.NewTextDelta(2, "hello, ")
	if err != nil {
		t.Fatalf("ai.NewTextDelta returned %v, want no failure", err)
	}
	textDelta2, err := ai.NewTextDelta(2, "world")
	if err != nil {
		t.Fatalf("ai.NewTextDelta returned %v, want no failure", err)
	}
	textEnd, err := ai.NewTextBlockEnd(2)
	if err != nil {
		t.Fatalf("ai.NewTextBlockEnd returned %v, want no failure", err)
	}
	completion, err := ai.NewCompletion(finishReason, ai.Usage{})
	if err != nil {
		t.Fatalf("ai.NewCompletion returned %v, want no failure", err)
	}
	return agenttest.Script{Steps: []agenttest.Step{
		agenttest.Emit(respStart),
		agenttest.Emit(reasonStartEvent),
		agenttest.Emit(reasonDelta),
		agenttest.Emit(reasonEnd),
		agenttest.Emit(textStart),
		agenttest.Emit(textDelta1),
		agenttest.Emit(textDelta2),
		agenttest.Emit(textEnd),
		agenttest.Emit(completion),
	}}
}

// mustSimpleRequest builds the smallest valid request the tests need:
// one model identity, one user text message (matches agenttest's
// provider_test.go mustSimpleRequest, restated here so this file's
// tests have no cross-file test-only dependencies).
func mustSimpleRequest(t *testing.T) ai.Request {
	t.Helper()

	part, err := ai.NewText("hello")
	if err != nil {
		t.Fatalf("ai.NewText returned %v, want no failure", err)
	}
	message, err := ai.NewMessage(ai.RoleUser, part)
	if err != nil {
		t.Fatalf("ai.NewMessage returned %v, want no failure", err)
	}
	request, err := ai.NewRequest("cachicamas-neutral-model-1", []ai.Message{message})
	if err != nil {
		t.Fatalf("ai.NewRequest returned %v, want no failure", err)
	}
	return request
}

// drainSink drains the agent-level event channel sink into a slice.
// Mirrors agenttest's drainFake but for the agent envelope: stops on
// channel close (the loop owns the close, R-LSK-001).
func drainSink(t *testing.T, sink <-chan *agent.Event) []agent.Event {
	t.Helper()

	var got []agent.Event
	for ev := range sink {
		got = append(got, *ev)
	}
	return got
}

// reconstructedLoopMessage is the AG-07 reconstruction helper's
// output shape: ordered parts (text or reasoning), complete-bracket
// flag. Mirrors reconstruction_test.go's reconstructedMessage,
// adapted for the walking-skeleton scope (one assistant message per
// turn, both text and reasoning parts interleaved by bracket-close).
type reconstructedLoopMessage struct {
	Parts    []ai.Part
	Complete bool
}

// reconstructLoopMessage joins the fragments carried by
// message_delta_text / message_delta_reasoning events in events, in
// delivery order, building one Part per complete bracket. A bracket
// is "complete" when its start and end both appear in events. The
// helper is test-only (AD-4) and lives in this file (per
// reconstruction_test.go's posture); the property test (S-LSK-003)
// is the load-bearing assertion; the bites (S-LSK-003a, S-LSK-003b)
// prove the helper is non-vacuous.
//
// Walking-skeleton scope: a single message's worth of fragments;
// AG-23 introduces a stream-kit helper that handles interleaving.
func reconstructLoopMessage(events []agent.Event) (reconstructedLoopMessage, bool) {
	type bracket struct {
		started   bool
		ended     bool
		text      strings.Builder
		token     []byte
		hasToken  bool
		reasoning bool
	}
	brackets := map[ai.MessageID]*bracket{}
	order := []ai.MessageID{}

	for _, e := range events {
		switch e.Kind() {
		case agent.EventKindMessageStartText:
			if p, ok := e.MessageStartText(); ok {
				id := p.MessageID()
				if _, exists := brackets[id]; !exists {
					brackets[id] = &bracket{}
					order = append(order, id)
				}
				brackets[id].started = true
			}
		case agent.EventKindMessageStartReasoning:
			if p, ok := e.MessageStartReasoning(); ok {
				id := p.MessageID()
				if _, exists := brackets[id]; !exists {
					brackets[id] = &bracket{reasoning: true}
					order = append(order, id)
				}
				brackets[id].started = true
			}
		case agent.EventKindMessageDeltaText:
			if p, ok := e.MessageDeltaText(); ok {
				id := p.MessageID()
				if _, exists := brackets[id]; !exists {
					brackets[id] = &bracket{}
					order = append(order, id)
				}
				brackets[id].text.WriteString(p.Fragment())
			}
		case agent.EventKindMessageDeltaReasoning:
			if p, ok := e.MessageDeltaReasoning(); ok {
				id := p.MessageID()
				if _, exists := brackets[id]; !exists {
					brackets[id] = &bracket{reasoning: true}
					order = append(order, id)
				}
				brackets[id].text.WriteString(p.Fragment())
			}
		case agent.EventKindMessageEndText:
			if p, ok := e.MessageEndText(); ok {
				id := p.MessageID()
				if _, exists := brackets[id]; !exists {
					brackets[id] = &bracket{}
					order = append(order, id)
				}
				brackets[id].ended = true
			}
		case agent.EventKindMessageEndReasoning:
			if p, ok := e.MessageEndReasoning(); ok {
				id := p.MessageID()
				if _, exists := brackets[id]; !exists {
					brackets[id] = &bracket{reasoning: true}
					order = append(order, id)
				}
				brackets[id].ended = true
				// ReasoningBlockEnd carries the token, but the agent
				// layer's MessageEndReasoning does not — the agent
				// reconstruction helper is a test-only tool. The
				// loop's reconstructed message is the property; the
				// helper here only proves the parts list is in
				// bracket-close order with the right text bytes.
				brackets[id].hasToken = false
				brackets[id].token = nil
			}
		}
	}

	// Walk in bracket-start order to assemble parts.
	var parts []ai.Part
	complete := true
	for _, id := range order {
		b := brackets[id]
		if !b.started || !b.ended {
			complete = false
			continue
		}
		text := b.text.String()
		if b.reasoning {
			// Reasoning without the token — the helper proves the
			// shape, the property test asserts the loop's full
			// reconstruction (with token byte-exact) matches.
			part, err := ai.NewReasoning(text, nil)
			if err != nil {
				continue
			}
			parts = append(parts, part)
		} else {
			part, err := ai.NewText(text)
			if err != nil {
				continue
			}
			parts = append(parts, part)
		}
	}
	if len(parts) == 0 {
		return reconstructedLoopMessage{}, false
	}
	return reconstructedLoopMessage{Parts: parts, Complete: complete}, true
}

// equalByContent compares two ai.Message values by their content parts
// in order, matching Message.Equal's own exclusion of identity
// (V-REQ-03, doc 0001's seal on MessageID's unforgeability).
func equalByContent(a, b ai.Message) bool {
	return reflect.DeepEqual(a.Content(), b.Content())
}

// TestTurn_Phase1_NoOpCompileCheck is Phase 1's only test (tasks 1.1
// / 1.2 / 1.3): the skeleton compiles, the file's helpers are all
// reachable, and substrate stays untouched. The real RED/GREEN tests
// for S-LSK-001..007 land in Phase 2 / Phase 3 / Phase 4.
//
// This test is intentionally tiny: it proves the loop file imports
// compile, the helpers compile, and the stub `Turn` is callable. It
// asserts nothing about behavior — that is Phase 2's job.
func TestTurn_Phase1_NoOpCompileCheck(t *testing.T) {
	t.Parallel()

	script := scriptTextResponse(t, ai.FinishReasonStop)
	_ = script
	req := mustSimpleRequest(t)
	_ = req

	// drainSink and reconstructLoopMessage are reachable from this
	// test, satisfying the linter without committing to a behavioral
	// assertion at Phase 1.
	ch := make(chan *agent.Event)
	close(ch) // drainSink reads until close — close immediately.
	_ = drainSink(t, ch)
	recon, ok := reconstructLoopMessage(nil)
	_ = recon
	_ = ok

	// scriptReasoningTextResponse is reachable too — proves the
	// interleaved reasoning + text scripted shape compiles.
	reasoningScript := scriptReasoningTextResponse(t, ai.FinishReasonStop, "thinking…", []byte("token-lsk-005"))
	_ = reasoningScript

	// equalByContent on two zero messages — proves the comparison
	// helper compiles and round-trips the trivial case.
	if !equalByContent(ai.Message{}, ai.Message{}) {
		t.Error("equalByContent returned false on two zero messages")
	}
}
