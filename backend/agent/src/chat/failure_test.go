// CH-02.3 — a provider failure terminates the turn with the wire's typed
// error, and the conversation survives it (R-CCP-007, R-CCP-008, R-CCP-009).
// Every scenario lives in package chat_test.
package chat_test

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/cachicamas/backend/agent/src/agenttest"
	"github.com/cachicamas/backend/agent/src/ai"
	"github.com/cachicamas/backend/agent/src/chat"
)

// TestFailure_TerminatesWithTypedError — S-CCP-060, S-CCP-061, S-CCP-063.
// Given a scripted provider that fails after one fragment, when the turn is
// driven, then the projected stream's terminal event is the wire error
// shape, its kind is "server", its message is non-empty, and no turn.end
// follows it; and given that same conversation immediately afterwards, when
// a second prompt is driven against a scripted provider that succeeds, then
// that turn completes with a turn.end.
func TestFailure_TerminatesWithTypedError(t *testing.T) {
	t.Parallel()

	failScript := scriptTextThenFailure(t, false, true, ai.FailureCategoryUnavailable)
	successScript := scriptTextResponse(t, 1, []string{"recovered"}, ai.FinishReasonStop)
	provider := agenttest.NewProvider(failScript, successScript)

	conv, err := chat.NewConversation(chat.Config{Provider: provider})
	if err != nil {
		t.Fatalf("chat.NewConversation returned %v, want nil", err)
	}

	out, err := conv.Send(context.Background(), "hello")
	if err != nil {
		t.Fatalf("Send returned %v, want nil", err)
	}
	events := drainWire(t, out)
	if len(events) == 0 {
		t.Fatal("no wire events observed")
	}
	terminal := events[len(events)-1]
	errEvt, ok := terminal.(chat.Error)
	if !ok {
		t.Fatalf("terminal event = %#v, want chat.Error", terminal)
	}
	if errEvt.Kind != "server" {
		t.Errorf("error.Kind = %q, want %q", errEvt.Kind, "server")
	}
	if errEvt.Message == "" {
		t.Error("error.Message is empty, want a non-empty archetype phrase")
	}
	for _, ev := range events {
		if _, isTurnEnd := ev.(chat.TurnEnd); isTurnEnd {
			t.Errorf("a turn.end was projected on a failed run: %#v, want none", ev)
		}
	}

	out2, err := conv.Send(context.Background(), "try again")
	if err != nil {
		t.Fatalf("Send (recovery) returned %v, want nil", err)
	}
	events2 := drainWire(t, out2)
	if !endsWithTurnEnd(events2) {
		t.Fatalf("recovery turn's terminal event = %#v, want chat.TurnEnd", events2[len(events2)-1])
	}
}

// TestFailure_NoProviderStringOnWire — S-CCP-070, S-CCP-071. Given a
// scripted provider whose failure wraps a cause containing a uniquely
// minted token, when the turn is driven, then that token appears in no
// projected wire event of any shape, and the terminal error.Message is
// byte-identical to the archetype's fixed phrase for that category.
func TestFailure_NoProviderStringOnWire(t *testing.T) {
	t.Parallel()

	const mintedToken = "XZQ-CH02-PROBE-f3a9c2"
	cause := errors.New(mintedToken)
	failure, err := ai.MidStreamFailure(ai.FailureReport{
		Category: ai.FailureCategoryUnavailable,
		Cause:    cause,
	}, false)
	if err != nil {
		t.Fatalf("ai.MidStreamFailure returned %v, want no failure", err)
	}
	terminal, err := ai.ErrorEvent(failure)
	if err != nil {
		t.Fatalf("ai.ErrorEvent returned %v, want no failure", err)
	}
	provider := agenttest.NewProvider(agenttest.Script{Steps: []agenttest.Step{agenttest.Emit(terminal)}})

	conv, cerr := chat.NewConversation(chat.Config{Provider: provider})
	if cerr != nil {
		t.Fatalf("chat.NewConversation returned %v, want nil", cerr)
	}
	out, serr := conv.Send(context.Background(), "hello")
	if serr != nil {
		t.Fatalf("Send returned %v, want nil", serr)
	}
	events := drainWire(t, out)

	for _, ev := range events {
		if containsToken(t, ev, mintedToken) {
			t.Errorf("wire event %#v carries the minted provider token %q, want none", ev, mintedToken)
		}
	}

	if len(events) == 0 {
		t.Fatal("no wire events observed")
	}
	errEvt, ok := events[len(events)-1].(chat.Error)
	if !ok {
		t.Fatalf("terminal event = %#v, want chat.Error", events[len(events)-1])
	}
	if want := "The model provider is temporarily unavailable."; errEvt.Message != want {
		t.Errorf("error.Message = %q, want %q (byte-identical to the archetype's fixed phrase)", errEvt.Message, want)
	}
}

// TestFailure_NoArchetypeRetry — S-CCP-080, S-CCP-081, S-CCP-082 (D11:
// exactly 2 invocations, never 3; the third script must remain
// unconsumed on the happy path so the no-retry defeat is observable).
// Given a scripted provider that fails retryably on its first
// invocation, succeeds on its second, and has a third script queued
// that the harness is not expected to reach, when a turn is driven with
// Harness.RetryAttempts left unset, then the scripted provider records
// exactly 2 invocations, the third script remains unconsumed, and the
// turn completes. The third-script assertion is the falsifier that
// S-CCP-082 demanded: with only two scripts, an over-invocation cannot
// be observed because the third Stream call would never reach p.requests
// (R-AFP-020's ErrScriptsExhausted path is silent on capture).
func TestFailure_NoArchetypeRetry(t *testing.T) {
	t.Parallel()

	failOnce := scriptTextThenFailure(t, true, false, ai.FailureCategoryUnavailable)
	thenSucceed := scriptTextResponse(t, 1, []string{"recovered"}, ai.FinishReasonStop)
	unused := scriptTextResponse(t, 1, []string{"never-reached"}, ai.FinishReasonStop)
	provider := agenttest.NewProvider(failOnce, thenSucceed, unused)

	conv, err := chat.NewConversation(chat.Config{Provider: provider})
	if err != nil {
		t.Fatalf("chat.NewConversation returned %v, want nil", err)
	}
	out, err := conv.Send(context.Background(), "hello")
	if err != nil {
		t.Fatalf("Send returned %v, want nil", err)
	}
	events := drainWire(t, out)
	if !endsWithTurnEnd(events) {
		t.Fatalf("terminal event = %#v, want chat.TurnEnd (the retry succeeds within the harness's own bound)", events[len(events)-1])
	}

	if got := len(provider.Requests()); got != 2 {
		t.Errorf("provider recorded %d invocation(s), want exactly 2 (D11: H=3 bounds TOTAL attempts; the archetype schedules none of its own)", got)
	}
	if got := provider.ScriptsRemaining(); got < 1 {
		t.Errorf("post-harness ScriptsRemaining = %d, want >= 1 (the third script was available, the harness chose not to take it — a 3rd invocation by the archetype would have consumed it)", got)
	}
}

// scriptTextThenFailure builds a script that emits a short text bracket
// (only when outputPreceded) then a terminal mid-stream *ai.Failure of the
// given category, carrying retryable and outputPreceded exactly as given —
// mirrors package agent_test's own retry_policy_test.go:
// scriptTextThenMidStreamFailure precedent, reproduced here because that
// helper is unexported to a sibling test package.
func scriptTextThenFailure(t *testing.T, retryable, outputPreceded bool, category ai.FailureCategory) agenttest.Script {
	t.Helper()

	var steps []agenttest.Step
	if outputPreceded {
		start, err := ai.NewTextBlockStart(1)
		if err != nil {
			t.Fatalf("ai.NewTextBlockStart returned %v, want no failure", err)
		}
		delta, err := ai.NewTextDelta(1, "partial-before-failure")
		if err != nil {
			t.Fatalf("ai.NewTextDelta returned %v, want no failure", err)
		}
		end, err := ai.NewTextBlockEnd(1)
		if err != nil {
			t.Fatalf("ai.NewTextBlockEnd returned %v, want no failure", err)
		}
		steps = append(steps, agenttest.Emit(start), agenttest.Emit(delta), agenttest.Emit(end))
	}

	failure, err := ai.MidStreamFailure(ai.FailureReport{
		Category:  category,
		Retryable: retryable,
	}, outputPreceded)
	if err != nil {
		t.Fatalf("ai.MidStreamFailure returned %v, want no failure", err)
	}
	terminal, err := ai.ErrorEvent(failure)
	if err != nil {
		t.Fatalf("ai.ErrorEvent returned %v, want no failure", err)
	}
	steps = append(steps, agenttest.Emit(terminal))
	return agenttest.Script{Steps: steps}
}

// containsToken reports whether ev's own string fields carry token — the
// widest possible surface for "no field of any shape leaks it".
func containsToken(t *testing.T, ev chat.WireEvent, token string) bool {
	t.Helper()
	switch e := ev.(type) {
	case chat.MessageStart:
		return strings.Contains(e.MessageID, token)
	case chat.MessageDelta:
		return strings.Contains(e.Delta, token)
	case chat.MessageEnd:
		return strings.Contains(e.FinishReason, token)
	case chat.TurnEnd:
		return e.FinishReason != nil && strings.Contains(*e.FinishReason, token)
	case chat.Error:
		return strings.Contains(e.Kind, token) || strings.Contains(e.Message, token)
	default:
		t.Fatalf("containsToken: unhandled WireEvent type %T", ev)
		return false
	}
}
