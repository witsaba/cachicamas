// AI-21.8 — sequential-call scripting, exhaustion and totality.
//
// fake_queue_test.go proves R-AFP-019/020 and NFR-AFP-D from outside
// package agenttest, reusing this package's shared helpers — same
// agenttest_test package as its siblings.
package agenttest_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/cachicamas/backend/agent/src/agenttest"
	"github.com/cachicamas/backend/agent/src/ai"
)

// AI-21.8 item 1 (R-AFP-019, S-AFP-045…047) — consecutive calls on one
// fake consume consecutive scripts from its queue, in enqueue order, each
// exactly once; the queue's own observable behaviour after two of three
// calls shows exactly one unconsumed script remains.
func TestProvider_ConsecutiveCalls_ConsumeConsecutiveScriptsInOrderExactlyOnce(t *testing.T) {
	t.Parallel()

	t.Run("tool-call script then text script: call one yields the tool call, call two yields the text, in order", func(t *testing.T) {
		t.Parallel()

		provider := agenttest.NewProvider(
			mustToolCallScript(t, 1, "call-1", "search", `{"q":"x"}`),
			mustTextDeltaScript(t),
		)

		ch1, err := provider.Stream(context.Background(), mustSimpleRequest(t))
		if err != nil {
			t.Fatalf("call 1: Stream returned %v, want no failure", err)
		}
		got1 := drainFake(t, ch1)
		if len(got1) == 0 || got1[0].Kind() != ai.EventKindToolCallStart {
			t.Fatalf("call 1's first event kind = %v, want %v (the tool-call script)", got1[0].Kind(), ai.EventKindToolCallStart)
		}

		ch2, err := provider.Stream(context.Background(), mustSimpleRequest(t))
		if err != nil {
			t.Fatalf("call 2: Stream returned %v, want no failure", err)
		}
		got2 := drainFake(t, ch2)
		if len(got2) == 0 || got2[0].Kind() != ai.EventKindTextBlockStart {
			t.Fatalf("call 2's first event kind = %v, want %v (the text script)", got2[0].Kind(), ai.EventKindTextBlockStart)
		}
	})

	t.Run("two identical-length scripts: each call consumes its own script, neither observes the other's events", func(t *testing.T) {
		t.Parallel()

		provider := agenttest.NewProvider(
			mustToolCallScript(t, 1, "call-A", "search", `{"q":"a"}`),
			mustToolCallScript(t, 1, "call-B", "search", `{"q":"b"}`),
		)

		chA, err := provider.Stream(context.Background(), mustSimpleRequest(t))
		if err != nil {
			t.Fatalf("call 1: Stream returned %v, want no failure", err)
		}
		callA := reconstructToolCall(drainFake(t, chA), 1)
		if callA.id != "call-A" {
			t.Errorf("call 1's reconstructed id = %q, want %q", callA.id, "call-A")
		}

		chB, err := provider.Stream(context.Background(), mustSimpleRequest(t))
		if err != nil {
			t.Fatalf("call 2: Stream returned %v, want no failure", err)
		}
		callB := reconstructToolCall(drainFake(t, chB), 1)
		if callB.id != "call-B" {
			t.Errorf("call 2's reconstructed id = %q, want %q", callB.id, "call-B")
		}
	})

	t.Run("three scripts: after two calls, exactly one unconsumed script remains, observed through the queue's own behaviour", func(t *testing.T) {
		t.Parallel()

		provider := agenttest.NewProvider(mustTextDeltaScript(t), mustTextDeltaScript(t), mustTextDeltaScript(t))

		for i := range 2 {
			ch, err := provider.Stream(context.Background(), mustSimpleRequest(t))
			if err != nil {
				t.Fatalf("call %d: Stream returned %v, want no failure", i+1, err)
			}
			drainFake(t, ch)
		}

		// The third call succeeds — proving one script was still queued.
		ch3, err := provider.Stream(context.Background(), mustSimpleRequest(t))
		if err != nil {
			t.Fatalf("call 3: Stream returned %v, want no failure (one script must have remained)", err)
		}
		drainFake(t, ch3)

		// The fourth call fails — proving zero remain now.
		if _, err := provider.Stream(context.Background(), mustSimpleRequest(t)); !errors.Is(err, agenttest.ErrScriptsExhausted) {
			t.Errorf("call 4's error = %v, want errors.Is(err, agenttest.ErrScriptsExhausted) (zero scripts remain after exactly three calls)", err)
		}
	})
}

// AI-21.8 item 2 (R-AFP-020, S-AFP-048…050) — a call against an exhausted
// queue fails loudly and immediately: within a bounded test deadline that
// the call itself never reaches, naming the exhaustion; it neither
// replays the previous script nor returns a stream that closes as a clean
// empty success.
func TestProvider_CallAgainstExhaustedQueue_FailsLoudlyWithinBoundedDeadline(t *testing.T) {
	t.Parallel()

	provider := agenttest.NewProvider(mustTextDeltaScript(t))

	ch1, err := provider.Stream(context.Background(), mustSimpleRequest(t))
	if err != nil {
		t.Fatalf("first call: Stream returned %v, want no failure", err)
	}
	first := drainFake(t, ch1)
	if len(first) != 4 {
		t.Fatalf("first call drained %d event(s), want 4", len(first))
	}

	done := make(chan struct{})
	var ch2 <-chan ai.Event
	var err2 error
	go func() {
		defer close(done)
		ch2, err2 = provider.Stream(context.Background(), mustSimpleRequest(t))
	}()

	select {
	case <-done:
	case <-time.After(boundedFakeTimeout):
		t.Fatal("the exhausted call did not return within the bounded deadline — it must fail immediately, never hang (S-AFP-050)")
	}

	if ch2 != nil {
		t.Error("the exhausted call returned a non-nil channel, want no carrier — not a clean-closing empty success (S-AFP-049)")
	}
	if err2 == nil {
		t.Fatal("the exhausted call returned a nil error, want a loud failure naming the exhaustion")
	}
	if !errors.Is(err2, agenttest.ErrScriptsExhausted) {
		t.Errorf("errors.Is(err2, agenttest.ErrScriptsExhausted) = false on %v, want the exhaustion sentinel named", err2)
	}
}

// requireNoPanic converts a panic on the calling goroutine into a test
// failure naming it, rather than letting it crash the test binary — used
// only in the totality table below, where a panic is itself the bug under
// test.
func requireNoPanic(t *testing.T) {
	t.Helper()
	if r := recover(); r != nil {
		t.Fatalf("panicked: %v, want no panic for this input (NFR-AFP-D)", r)
	}
}

// AI-21.8's totality proof (NFR-AFP-D, S-AFP-056) — every exported entry
// point handles each of a table of extreme inputs without panicking,
// except R-AFP-020's own documented loud-failure path.
func TestProvider_ExtremeInputs_NoExportedEntryPointPanics_ExceptTheDocumentedExhaustionPath(t *testing.T) {
	t.Parallel()

	t.Run("Stream with a zero-value request does not panic", func(t *testing.T) {
		t.Parallel()
		defer requireNoPanic(t)

		var zero ai.Request
		if _, err := agenttest.NewProvider(mustTextDeltaScript(t)).Stream(context.Background(), zero); err == nil {
			t.Fatal("Stream with a zero-value request unexpectedly succeeded")
		}
	})

	t.Run("Stream with a nil context does not panic", func(t *testing.T) {
		t.Parallel()
		defer requireNoPanic(t)

		//nolint:staticcheck // deliberately proving NFR-AFP-D's nil-context totality
		ch, err := agenttest.NewProvider(mustTextDeltaScript(t)).Stream(nil, mustSimpleRequest(t))
		if err != nil {
			t.Fatalf("Stream(nil, ...) returned %v, want no failure", err)
		}
		drainFake(t, ch)
	})

	t.Run("Stream with an already-cancelled context does not panic", func(t *testing.T) {
		t.Parallel()
		defer requireNoPanic(t)

		ctx, cancel := context.WithCancel(context.Background())
		cancel()
		_, _ = agenttest.NewProvider(mustTextDeltaScript(t)).Stream(ctx, mustSimpleRequest(t))
	})

	t.Run("Stream with an empty script does not panic and closes cleanly", func(t *testing.T) {
		t.Parallel()
		defer requireNoPanic(t)

		ch, err := agenttest.NewProvider(agenttest.Script{}).Stream(context.Background(), mustSimpleRequest(t))
		if err != nil {
			t.Fatalf("Stream with an empty script returned %v, want no failure", err)
		}
		got := drainFake(t, ch)
		if len(got) != 0 {
			t.Errorf("drained %d event(s) from an empty script, want 0", len(got))
		}
	})

	t.Run("Stream against a never-populated queue fails via the documented exhaustion path, not a panic", func(t *testing.T) {
		t.Parallel()
		defer requireNoPanic(t)

		ch, err := agenttest.NewProvider().Stream(context.Background(), mustSimpleRequest(t))
		if ch != nil {
			t.Error("Stream against a never-populated queue returned a non-nil channel, want no carrier")
		}
		if !errors.Is(err, agenttest.ErrScriptsExhausted) {
			t.Errorf("errors.Is(err, agenttest.ErrScriptsExhausted) = false on %v", err)
		}
	})

	t.Run("Requests on a freshly constructed provider does not panic", func(t *testing.T) {
		t.Parallel()
		defer requireNoPanic(t)

		got := agenttest.NewProvider().Requests()
		if len(got) != 0 {
			t.Errorf("Requests() on a fresh provider = %v, want empty", got)
		}
	})
}
