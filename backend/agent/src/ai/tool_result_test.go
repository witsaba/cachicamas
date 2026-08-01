// Tests for AI-09.3 — the tool result.
//
// External package, for AI-09.1's reason: the consumer this contract exists for
// is an adapter in a vendor package, and doc 0002 asks the read to be made from
// outside.
package ai_test

import (
	"fmt"
	"strings"
	"testing"

	"github.com/cachicamas/backend/agent/src/ai"
)

// AI-09.3 item 1 — the walking skeleton for the second kind.
//
// Construct a tool result, place it in a message, and read its correlation and
// its content back out from another package (V-REQ-18).
func TestToolResult_CorrelationAndContent_ReadBackFromAMessageInAnExternalPackage(t *testing.T) {
	t.Parallel()

	const (
		callID  = "toolu_01A09Kf7"
		content = "127.0.0.1\tlocalhost\n"
	)

	part, err := ai.NewToolResult(callID, content)
	if err != nil {
		t.Fatalf("ai.NewToolResult returned %v, want no failure", err)
	}

	message, err := ai.NewMessage(ai.RoleTool, part)
	if err != nil {
		t.Fatalf("ai.NewMessage returned %v, want no failure", err)
	}

	read := message.Content()
	if len(read) != 1 {
		t.Fatalf("message.Content() has %d elements, want 1", len(read))
	}

	result, ok := read[0].ToolResult()
	if !ok {
		t.Fatalf("read[0].ToolResult() reported no tool result on a part built by ai.NewToolResult")
	}

	if got := result.CallID(); got != callID {
		t.Errorf("result.CallID() = %q, want %q", got, callID)
	}
	if got := result.Content(); got != content {
		t.Errorf("result.Content() = %q, want %q", got, content)
	}
	if result.Failed() {
		t.Errorf("result.Failed() = true on a result built by ai.NewToolResult, want false")
	}
}

// AI-09.3 item 2 — the correlation round-trips exactly.
//
// Including identities an adapter minted. One provider assigns no tool-call
// identifiers at all (doc 0001 § 3.3 row 7), so its adapter mints synthetic ones
// and keeps the mapping — and that mapping must survive session serialisation
// and reload one layer up (doc 0001 § 5.2). A contract that parsed, prefixed or
// canonicalized the identity would make both unimplementable, which is why there
// is exactly one rule on it and the rule is that it exists.
func TestToolResult_CorrelationIdentity_RoundTripsExactlyIncludingSyntheticOnes(t *testing.T) {
	t.Parallel()

	identities := []struct {
		what string
		id   string
	}{
		{"a provider-assigned identity", "toolu_01A09Kf7Xr2mQ"},
		{"an identity an adapter minted because the provider assigns none", "cachicamas-synthetic-0000000000000003"},
		{"an identity carrying punctuation no parser of ours would accept", "call/17:3|{}"},
		// Built from escapes rather than literal bytes: a source file carrying
		// literal bidirectional control characters trips ST1018, and a fixture
		// is not worth a lint suppression.
		{"an identity carrying non-ASCII bytes", "call-café-中文-\U0001F600"},
		{"an identity that is a single byte", "x"},
	}

	for _, identity := range identities {
		t.Run(identity.what, func(t *testing.T) {
			t.Parallel()

			callPart, err := ai.NewToolCall(identity.id, "read_file", []byte(`{"path":"a"}`))
			if err != nil {
				t.Fatalf("ai.NewToolCall returned %v, want no failure", err)
			}
			resultPart, err := ai.NewToolResult(identity.id, "done")
			if err != nil {
				t.Fatalf("ai.NewToolResult returned %v, want no failure", err)
			}

			callMessage, err := ai.NewMessage(ai.RoleAssistant, callPart)
			if err != nil {
				t.Fatalf("ai.NewMessage returned %v, want no failure", err)
			}
			resultMessage, err := ai.NewMessage(ai.RoleTool, resultPart)
			if err != nil {
				t.Fatalf("ai.NewMessage returned %v, want no failure", err)
			}

			calls := ai.ToolCalls(callMessage.Content())
			if len(calls) != 1 {
				t.Fatalf("ai.ToolCalls returned %d calls, want 1", len(calls))
			}
			result, ok := resultMessage.Content()[0].ToolResult()
			if !ok {
				t.Fatal("the tool-result part reported no tool result")
			}

			if got := result.CallID(); got != identity.id {
				t.Errorf("result.CallID() = %q, want %q — the identity is carried, never rewritten", got, identity.id)
			}
			if result.CallID() != calls[0].ID() {
				t.Errorf("result.CallID() = %q and the call's ID() = %q; they must pair under ordinary equality, "+
					"so a consumer needs no comparison function of this package's", result.CallID(), calls[0].ID())
			}
		})
	}

	t.Run("an empty correlation is a caller-contract failure", func(t *testing.T) {
		t.Parallel()

		part, err := ai.NewToolResult("", "done")
		if err == nil {
			t.Fatal("ai.NewToolResult accepted an empty correlation; a result that names no call is an answer to nothing")
		}
		assertViolation(t, err, ai.ErrEmpty, "call")
		if part != (ai.Part{}) {
			t.Errorf("a rejected construction returned a non-zero part, want the zero ai.Part")
		}
	})

	t.Run("an empty content is legal, and means the tool produced no output", func(t *testing.T) {
		t.Parallel()

		part, err := ai.NewToolResult("call-1", "")
		if err != nil {
			t.Fatalf("ai.NewToolResult rejected an empty content with %v.\n"+
				"  A tool that produced no output is routine, and inventing a placeholder is an application decision (V-OUT-04).", err)
		}
		result, ok := part.ToolResult()
		if !ok {
			t.Fatal("part.ToolResult() reported no tool result")
		}
		if got := result.Content(); got != "" {
			t.Errorf("result.Content() = %q, want the empty content it was built with", got)
		}
	})
}

// AI-09.3 item 3 — a tool that failed is content, not an error.
//
// V-REQ-18 states it without hedging: a result that reports failure "is not a
// V-FAIL-01 and not a V-FAIL-05; it is ordinary content". A failing tool is a
// normal outcome the model must see and reason about, so nothing about it
// reaches AI-04's taxonomy — the failure is a state on a successfully
// constructed value.
func TestToolResult_ReportedFailure_IsDistinguishableAndIsNotAnError(t *testing.T) {
	t.Parallel()

	const (
		callID  = "toolu_01A09Kf7"
		output  = "open /etc/shadow: permission denied"
		success = "ok"
	)

	t.Run("constructing a failure produces no error and reads back as failed", func(t *testing.T) {
		t.Parallel()

		part, err := ai.NewToolFailure(callID, output)
		if err != nil {
			t.Fatalf("ai.NewToolFailure returned %v.\n"+
				"  A failing tool is not a caller-contract failure and not a provider failure; it is content the model must see.", err)
		}

		result := mustReadToolResultBack(t, part)
		if !result.Failed() {
			t.Error("result.Failed() = false on a result built by ai.NewToolFailure")
		}
		if got := result.Content(); got != output {
			t.Errorf("result.Content() = %q, want the tool's own failure output %q", got, output)
		}
		if got := result.CallID(); got != callID {
			t.Errorf("result.CallID() = %q, want %q — a failure correlates exactly as a success does", got, callID)
		}
	})

	t.Run("a success reads back as not failed", func(t *testing.T) {
		t.Parallel()

		if result := mustReadToolResultBack(t, mustToolResult(t, callID, success)); result.Failed() {
			t.Error("result.Failed() = true on a result built by ai.NewToolResult")
		}
	})

	t.Run("two results alike but for the outcome are distinguishable", func(t *testing.T) {
		t.Parallel()

		// Identical correlation, identical content. The outcome is the only
		// difference, and it must be the difference a consumer can see —
		// otherwise the model is told a tool succeeded when it did not.
		ok, err := ai.NewToolResult(callID, output)
		if err != nil {
			t.Fatalf("ai.NewToolResult returned %v, want no failure", err)
		}
		failed, err := ai.NewToolFailure(callID, output)
		if err != nil {
			t.Fatalf("ai.NewToolFailure returned %v, want no failure", err)
		}

		if mustReadToolResultBack(t, ok).Failed() == mustReadToolResultBack(t, failed).Failed() {
			t.Error("a success and a failure with identical correlation and content report the same outcome")
		}
		if ok == failed {
			t.Error("a success and a failure with identical correlation and content compare ==; the outcome is part of the value")
		}
	})

	t.Run("the failure is reachable through the same accessor and the same kind", func(t *testing.T) {
		t.Parallel()

		part, err := ai.NewToolFailure(callID, output)
		if err != nil {
			t.Fatalf("ai.NewToolFailure returned %v, want no failure", err)
		}
		if got := part.Kind(); got != ai.PartKindToolResult {
			t.Errorf("part.Kind() = %v, want %v — a failure is a tool result, not a kind of its own", got, ai.PartKindToolResult)
		}
	})
}

// AI-09.3 item 4 — appended during implementation, for AI-09.1 item 5's reason.
//
// A tool's output is the most sensitive payload this package carries: it is
// whatever a tool read off a disk or a network. A consumer holding an
// ai.ToolResult and writing "%v" would print it through reflection, so the
// redaction posture V-FAIL-13 puts on the type covers this type too.
//
// The outcome is rendered and the payload is not. The outcome is a two-valued
// state this package defines rather than caller data, and it is the one thing a
// reader of a log actually needs.
func TestToolResult_Rendering_NeverReproducesItsPayload(t *testing.T) {
	t.Parallel()

	const secret = "sk-live-4f2a9c" //nolint:gosec // a fixture, not a credential

	part, err := ai.NewToolFailure("call-"+secret, "the tool printed "+secret)
	if err != nil {
		t.Fatalf("ai.NewToolFailure returned %v, want no failure", err)
	}
	result, ok := part.ToolResult()
	if !ok {
		t.Fatal("part.ToolResult() reported no tool result")
	}

	for _, rendering := range []struct {
		verb string
		got  string
	}{
		{"%v", fmt.Sprintf("%v", result)},
		{"%s", result.String()},
		{"%#v", fmt.Sprintf("%#v", result)},
		{"%v on the part", fmt.Sprintf("%v", part)},
		{"%#v on the part", fmt.Sprintf("%#v", part)},
	} {
		if strings.Contains(rendering.got, secret) {
			t.Errorf("%s rendered %q, which reproduces the payload it carries", rendering.verb, rendering.got)
		}
	}

	if got := fmt.Sprintf("%v", part); got != "part(tool_result)" {
		t.Errorf("%%v on the part = %q, want %q", got, "part(tool_result)")
	}

	success := mustReadToolResultBack(t, mustToolResult(t, "call-1", "ok"))
	if result.String() == success.String() {
		t.Errorf("a failure and a success render identically as %q; the outcome is not caller data and is what a log reader needs", result.String())
	}
}

// mustToolResult constructs a successful tool result or fails the test.
func mustToolResult(t *testing.T, callID, content string) ai.Part {
	t.Helper()

	part, err := ai.NewToolResult(callID, content)
	if err != nil {
		t.Fatalf("ai.NewToolResult(%q, …) returned %v, want no failure", callID, err)
	}
	return part
}

// mustReadToolResultBack places a part in a message and reads the tool result
// back out, which is the path a consumer in another package walks.
func mustReadToolResultBack(t *testing.T, part ai.Part) ai.ToolResult {
	t.Helper()

	message, err := ai.NewMessage(ai.RoleTool, part)
	if err != nil {
		t.Fatalf("ai.NewMessage returned %v, want no failure", err)
	}
	result, ok := message.Content()[0].ToolResult()
	if !ok {
		t.Fatal("the part reported no tool result")
	}
	return result
}
