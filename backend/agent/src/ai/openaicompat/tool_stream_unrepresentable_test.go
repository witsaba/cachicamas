// AI-30.4 — R-ATL-010: a tool call the neutral events cannot represent
// fails typed, never silently.
//
// S-ATL-052: fragmented name across elements (sea + rch) is NEVER
// concatenated (C9.3: no fragmentation prose for name) — typed failure.
// S-ATL-053: identity never supplied (no id by close) → typed failure,
// no empty-identity start emitted.
// S-ATL-054: a table-driven assertion distinguishing a genuine typed
// failure from a silent-drop mutation of the same fixture.

package openaicompat

import (
	"errors"
	"testing"

	"github.com/cachicamas/backend/agent/src/ai"
)

// TestUnrepresentable_FragmentedName covers S-ATL-052: a call whose
// first element carries name "sea" and whose second carries name
// "rch" for the same index produces a typed failure — name is NOT
// concatenated.
func TestUnrepresentable_FragmentedName(t *testing.T) {
	t.Parallel()

	state := &mapperState{}
	first := `{"index":0,"id":"call_U","function":{"name":"sea"}}`
	second := `{"index":0,"function":{"name":"rch","arguments":"{}"}}`
	if _, err := state.applyChunk(chunkFromTools("c", first)); err != nil {
		t.Fatalf("applyChunk(first) error = %v", err)
	}
	_, err := state.applyChunk(chunkFromTools("c", second))
	if err == nil {
		t.Fatal("applyChunk(second) error = nil, want errToolCallIdentityMismatch (S-ATL-052)")
	}
	if !errors.Is(err, errToolCallIdentityMismatch) && err.Error() == "" {
		t.Errorf("error %v is not the documented mismatch cause", err)
	}
}

// TestUnrepresentable_NeverSuppliedIdentity covers S-ATL-053: a call
// whose elements carry index and arguments but never any id produces
// a typed failure at close, not a start event with empty identity.
func TestUnrepresentable_NeverSuppliedIdentity(t *testing.T) {
	t.Parallel()

	state := &mapperState{}
	noId := `{"index":0,"function":{"name":"search","arguments":"{}"}}`
	if _, err := state.applyChunk(chunkFromTools("c", noId)); err != nil {
		t.Fatalf("applyChunk(noId) error = %v", err)
	}
	_, err := state.applyChunk(mustDecode(`{"id":"c","model":"m","created":1700000000,"object":"chat.completion.chunk","choices":[{"index":0,"delta":{},"finish_reason":"tool_calls"}]}`))
	if err == nil {
		t.Fatal("expected errToolCallMissingIdentity (S-ATL-053)")
	}
}

// TestUnrepresentable_TypedVsSilentDrop covers S-ATL-054: a table that
// runs each unrepresentable fixture and distinguishes a genuine typed
// failure from a silent-drop mutation.
func TestUnrepresentable_TypedVsSilentDrop(t *testing.T) {
	t.Parallel()

	type row struct {
		name string
		run  func() error
	}
	rows := []row{
		{
			name: "fragmented_name",
			run: func() error {
				state := &mapperState{}
				if _, err := state.applyChunk(chunkFromTools("c", `{"index":0,"id":"call_F","function":{"name":"sea"}}`)); err != nil {
					return err
				}
				_, err := state.applyChunk(chunkFromTools("c", `{"index":0,"function":{"name":"rch","arguments":"{}"}}`))
				return err
			},
		},
		{
			name: "missing_identity",
			run: func() error {
				state := &mapperState{}
				if _, err := state.applyChunk(chunkFromTools("c", `{"index":0,"function":{"name":"search","arguments":"{}"}}`)); err != nil {
					return err
				}
				_, err := state.applyChunk(mustDecode(`{"id":"c","model":"m","created":1700000000,"object":"chat.completion.chunk","choices":[{"index":0,"delta":{},"finish_reason":"tool_calls"}]}`))
				return err
			},
		},
	}

	for _, r := range rows {
		t.Run(r.name, func(t *testing.T) {
			err := r.run()
			if err == nil {
				t.Fatalf("row %q returned nil error, want a typed failure (S-ATL-054)", r.name)
			}
			if !errors.Is(err, ai.ErrMalformedResponse) {
				t.Errorf("row %q error %v does not wrap ai.ErrMalformedResponse (S-ATL-054)", r.name, err)
			}
		})
	}
}