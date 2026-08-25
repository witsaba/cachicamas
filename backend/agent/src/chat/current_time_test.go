// CH-09.1 — CurrentTimeTool tests (S-CTT-001, S-CTT-002, S-CTT-003).
// The three scenarios are transcribed verbatim from explore #3952.
// Tests use a fixed clock (injected via NewCurrentTimeTool) so
// assertions are byte-exact against an RFC3339-formatted timestamp.

package chat_test

import (
	"context"
	"testing"
	"time"

	"github.com/cachicamas/backend/agent/src/agent"
	"github.com/cachicamas/backend/agent/src/chat"
)

// fixedTestTime is the clock the tests inject. Pinned at a value
// that does not collide with the system clock at the test's run time,
// so any leakage between Run and the system's time.Now would surface
// as a diff in S-CTT-001.
var fixedTestTime = time.Date(2026, 8, 25, 7, 17, 0, 0, time.UTC)

// S-CTT-001 — Given CurrentTimeTool{NowFunc: func() time.Time {
// return fixedTime }}, When Run(ctx, []byte("{}"), nil) runs, Then
// Result.Content is fixedTime.Format(time.RFC3339) and Result.Outcome
// == ToolOutcomeSuccess and Result.CallID() is empty (the scheduler
// fills it).
func TestCurrentTimeTool_Run_EmptyArgs_ReturnsRFC3339(t *testing.T) {
	tool := chat.NewCurrentTimeTool(func() time.Time { return fixedTestTime })

	res, err := tool.Run(context.Background(), []byte("{}"), nil)
	if err != nil {
		t.Fatalf("Run returned error %v, want nil", err)
	}
	if res.Outcome != agent.ToolOutcomeSuccess {
		t.Errorf("Result.Outcome = %v, want ToolOutcomeSuccess", res.Outcome)
	}
	if got := string(res.Content); got != fixedTestTime.Format(time.RFC3339) {
		t.Errorf("Result.Content = %q, want %q", got, fixedTestTime.Format(time.RFC3339))
	}
	if got := res.CallID(); got != "" {
		t.Errorf("Result.CallID() = %q, want empty (scheduler fills)", got)
	}
}

// S-CTT-001 (extra) — Empty bytes is the documented synonym for "{}".
// The Run method admits both; the empty-bytes form is what Layer 1's
// ai.NewToolCall mints when the model passes no arguments.
func TestCurrentTimeTool_Run_EmptyBytes_ReturnsRFC3339(t *testing.T) {
	tool := chat.NewCurrentTimeTool(func() time.Time { return fixedTestTime })

	res, err := tool.Run(context.Background(), nil, nil)
	if err != nil {
		t.Fatalf("Run(nil) returned error %v, want nil", err)
	}
	if res.Outcome != agent.ToolOutcomeSuccess {
		t.Errorf("Result.Outcome = %v, want ToolOutcomeSuccess", res.Outcome)
	}
	if got := string(res.Content); got != fixedTestTime.Format(time.RFC3339) {
		t.Errorf("Result.Content = %q, want %q", got, fixedTestTime.Format(time.RFC3339))
	}
}

// S-CTT-002 — Given CurrentTimeTool, When Run(ctx,
// []byte(`{"timezone":"UTC"}`), nil) runs, Then it returns
// Result{Outcome: ToolOutcomeResultFailure, Content: <error message>}
// (args JSON-validated against {} schema), never silently ignored.
func TestCurrentTimeTool_Run_NonEmptyArgs_ReturnsTypedFailure(t *testing.T) {
	tool := chat.NewCurrentTimeTool(func() time.Time { return fixedTestTime })

	cases := []struct {
		name string
		args []byte
	}{
		{"known_extra_field", []byte(`{"timezone":"UTC"}`)},
		{"empty_value_field", []byte(`{"x":""}`)},
		{"nested_object", []byte(`{"nested":{"a":1}}`)},
		{"array_value", []byte(`{"k":[1,2]}`)},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			res, err := tool.Run(context.Background(), tc.args, nil)
			if err != nil {
				t.Fatalf("Run returned error %v, want nil", err)
			}
			if res.Outcome != agent.ToolOutcomeResultFailure {
				t.Errorf("Result.Outcome = %v, want ToolOutcomeResultFailure", res.Outcome)
			}
			if len(res.Content) == 0 {
				t.Errorf("Result.Content is empty, want a typed error message")
			}
		})
	}
}

// S-CTT-002 (extra) — Malformed JSON is a typed refusal at the
// same boundary, not a panic. The error message names the syntax
// problem so an operator can debug.
func TestCurrentTimeTool_Run_MalformedJSON_ReturnsTypedFailure(t *testing.T) {
	tool := chat.NewCurrentTimeTool(func() time.Time { return fixedTestTime })

	res, err := tool.Run(context.Background(), []byte(`{not-json`), nil)
	if err != nil {
		t.Fatalf("Run returned error %v, want nil", err)
	}
	if res.Outcome != agent.ToolOutcomeResultFailure {
		t.Errorf("Result.Outcome = %v, want ToolOutcomeResultFailure", res.Outcome)
	}
	if len(res.Content) == 0 {
		t.Errorf("Result.Content is empty, want a typed error message")
	}
}

// S-CTT-003 — Given CurrentTimeTool, When EffectClass() runs, Then
// it returns agent.EffectClassRead. And when Name() runs, Then it
// returns "current_time". Both are pure-value assertions, no side
// effects.
func TestCurrentTimeTool_NameAndEffectClass(t *testing.T) {
	tool := chat.NewCurrentTimeTool(func() time.Time { return fixedTestTime })

	if got := tool.Name(); got != "current_time" {
		t.Errorf("Name() = %q, want %q", got, "current_time")
	}
	if got := tool.EffectClass(); got != agent.EffectClassRead {
		t.Errorf("EffectClass() = %v, want EffectClassRead", got)
	}
}

// S-CTT-003 (extra) — NewCurrentTimeTool(nil) panics immediately.
// A misconfigured composition root surfaces at startup, not at first
// call — the constructor's invariant is "non-nil clock".
func TestNewCurrentTimeTool_NilPanics(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Errorf("NewCurrentTimeTool(nil) did not panic, want panic")
		}
	}()
	_ = chat.NewCurrentTimeTool(nil)
}