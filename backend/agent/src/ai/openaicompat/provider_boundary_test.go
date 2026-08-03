package openaicompat

import (
	"reflect"
	"testing"

	"github.com/cachicamas/backend/agent/src/ai"
)

// TestClient_HasNoStreamingEntryPoint covers S-APC-030: no streaming entry
// point exists on the landed adapter — not even one that returns an
// "unimplemented" result, a nil stream, or any other stand-in. At AI-25 the
// adapter is a configured value only; streaming arrives at AI-26.
func TestClient_HasNoStreamingEntryPoint(t *testing.T) {
	t.Parallel()

	typ := reflect.TypeOf(&Client{})
	if _, ok := typ.MethodByName("Stream"); ok {
		t.Fatal("*Client exposes a Stream method at AI-25 — streaming arrives at AI-26 (S-APC-030)")
	}
}

// TestClient_DoesNotSatisfyModelProviderAtRuntime covers S-APC-031: a
// run-time type assertion reports the adapter does not implement
// ai.ModelProvider.
//
// This is deliberately NOT the compile-time idiom
// var _ ai.ModelProvider = (*Client)(nil), which would fail the *build*
// rather than fail as a test — the wrong failure mode for a milestone
// whose whole point is proving the adapter does not yet falsely advertise
// an interface it has not implemented. This test is expected to flip to
// asserting ok == true once AI-26 lands Stream.
func TestClient_DoesNotSatisfyModelProviderAtRuntime(t *testing.T) {
	t.Parallel()

	_, ok := any(&Client{}).(ai.ModelProvider)
	if ok {
		t.Fatal("*Client satisfies ai.ModelProvider at AI-25 — it must not until AI-26 ships Stream (S-APC-031)")
	}
}
