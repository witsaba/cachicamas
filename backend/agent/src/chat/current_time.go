// Package chat — CH-09.1 chat.CurrentTimeTool (D-2, R-CTS-003, NFR-CTS-001).
//
// The first tool behind chat.ToolSource. It returns the system clock
// in RFC3339. EffectClass is EffectClassRead (no state mutation, no
// exec). The constructor accepts an injectable NowFunc so tests can
// drive a fixed clock (the testability seam).
//
// Args validation: the tool's schema is the empty object `{}`. Any
// non-empty JSON object yields ToolOutcomeResultFailure with a typed
// error message — never silently ignored. JSON syntax errors yield
// the same typed refusal.
//
// This is a deliberately minimal tool — no business meaning, no
// product behavior. The seam is the milestone's product; this
// implementation proves the seam end-to-end (args JSON-validated,
// result rendered, error path closed, time injection preserved for
// test determinism).
package chat

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/cachicamas/backend/agent/src/agent"
)

// CurrentTimeTool is the chat archetype's first tool (D-2). It
// implements agent.Tool with an injectable clock. The struct is
// exported so a test can read the wrapped NowFunc if needed; the
// field is unexported so callers cannot mutate it post-construction.
type CurrentTimeTool struct {
	now func() time.Time
}

// NewCurrentTimeTool constructs a CurrentTimeTool with the given clock
// function. now is required (a nil function would panic at Run-time;
// the constructor refuses nil so the misconfiguration surfaces
// immediately at composition-root wiring, not at first call).
func NewCurrentTimeTool(now func() time.Time) *CurrentTimeTool {
	if now == nil {
		panic("chat: NewCurrentTimeTool requires a non-nil now func")
	}
	return &CurrentTimeTool{now: now}
}

// Name returns the tool's identity — the literal "current_time"
// (R-CTS-003). It must match what the model addresses in its
// ai.ToolCall.name byte-for-byte.
func (t *CurrentTimeTool) Name() string { return "current_time" }

// EffectClass returns EffectClassRead (NFR-CTS-001). Reading the
// system clock is an observation, not a mutation or execution.
func (t *CurrentTimeTool) EffectClass() agent.EffectClass {
	return agent.EffectClassRead
}

// Run returns the system clock in RFC3339 form. The args schema is
// `{}`; any non-empty JSON object or malformed JSON yields
// ToolOutcomeResultFailure with a typed error message. On the happy
// path (args == "{}" or empty), Result.Content is the RFC3339
// rendering of t.now() and Outcome is ToolOutcomeSuccess. The
// scheduler fills CallID; the tool itself leaves it empty
// (R-CTS-003 / S-CTT-001).
func (t *CurrentTimeTool) Run(_ context.Context, args []byte, _ agent.PolicySlot) (agent.Result, error) {
	// Empty bytes is a documented synonym for "{}" — Layer 1's
	// ai.NewToolCall admits both, and a tool called with no
	// arguments is the expected call shape for current_time.
	if len(args) == 0 {
		return agent.Result{
			Outcome: agent.ToolOutcomeSuccess,
			Content: []byte(t.now().Format(time.RFC3339)),
		}, nil
	}

	// Validate that args is the JSON empty object `{}`. Anything
	// else (including a non-empty object) is a typed refusal — the
	// tool refuses unknown args rather than silently ignoring them
	// (S-CTT-002 / threat matrix "Args JSON validation").
	var probe map[string]json.RawMessage
	if err := json.Unmarshal(args, &probe); err != nil {
		return agent.Result{
			Outcome: agent.ToolOutcomeResultFailure,
			Content: []byte(fmt.Sprintf("current_time: args must be a JSON object, got: %v", err)),
		}, nil
	}
	if len(probe) != 0 {
		keys := make([]string, 0, len(probe))
		for k := range probe {
			keys = append(keys, k)
		}
		return agent.Result{
			Outcome: agent.ToolOutcomeResultFailure,
			Content: []byte(fmt.Sprintf("current_time: unexpected args %v (schema is {})", keys)),
		}, nil
	}

	return agent.Result{
		Outcome: agent.ToolOutcomeSuccess,
		Content: []byte(t.now().Format(time.RFC3339)),
	}, nil
}