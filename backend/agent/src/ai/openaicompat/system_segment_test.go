// AI-26.2 — ordered system segments render preserving order and content
// (R-ART-005).
package openaicompat_test

import (
	"testing"

	"github.com/cachicamas/backend/agent/src/ai"
	"github.com/cachicamas/backend/agent/src/ai/openaicompat"
)

func init() {
	registerExpectations(
		expectationCase{
			name: "three ordered system segments",
			build: func() ai.Request {
				system := mustSystemInstruction(ai.NewSystemInstruction(
					mustSegment(ai.NewSegment("You are a travel planning assistant.")),
					mustSegment(ai.NewSegment("Be terse.")),
					mustSegment(ai.NewSegment("Never invent flight numbers.")),
				))
				return mustRequest(ai.NewRequest("gpt-4o", []ai.Message{
					mustMessage(ai.NewMessage(ai.RoleUser, mustPart(ai.NewText("Hello, world!")))),
				}, ai.WithSystemInstruction(system)))
			},
			want: `{"model":"gpt-4o","messages":[` +
				`{"role":"system","content":"You are a travel planning assistant."},` +
				`{"role":"system","content":"Be terse."},` +
				`{"role":"system","content":"Never invent flight numbers."},` +
				`{"role":"user","content":"Hello, world!"}` +
				`],"stream":true,"stream_options":{"include_usage":true}}`,
		},
	)
}

// mustSegment panics if err != nil. See mustRequest in translation_test.go.
func mustSegment(segment ai.Segment, err error) ai.Segment {
	if err != nil {
		panic(err)
	}
	return segment
}

// mustSystemInstruction panics if err != nil. See mustRequest in
// translation_test.go.
func mustSystemInstruction(system ai.SystemInstruction, err error) ai.SystemInstruction {
	if err != nil {
		panic(err)
	}
	return system
}

// TestSystemSegment_ReversedOrderTwin_TranslatesDifferently proves
// S-ART-019: two requests whose segments are identical in content but
// reversed in order translate to different bodies — order is genuinely
// carried, not incidentally equal.
func TestSystemSegment_ReversedOrderTwin_TranslatesDifferently(t *testing.T) {
	build := func(segments ...ai.Segment) ai.Request {
		system, err := ai.NewSystemInstruction(segments...)
		if err != nil {
			t.Fatalf("ai.NewSystemInstruction returned %v, want no failure", err)
		}
		request, err := ai.NewRequest("gpt-4o", []ai.Message{
			mustMessage(ai.NewMessage(ai.RoleUser, mustPart(ai.NewText("Hello, world!")))),
		}, ai.WithSystemInstruction(system))
		if err != nil {
			t.Fatalf("ai.NewRequest returned %v, want no failure", err)
		}
		return request
	}

	first, err := ai.NewSegment("first")
	if err != nil {
		t.Fatalf("ai.NewSegment returned %v, want no failure", err)
	}
	second, err := ai.NewSegment("second")
	if err != nil {
		t.Fatalf("ai.NewSegment returned %v, want no failure", err)
	}

	forward := build(first, second)
	reversed := build(second, first)

	forwardBytes, err := openaicompat.Translate(forward)
	if err != nil {
		t.Fatalf("Translate(forward): unexpected error: %v", err)
	}
	reversedBytes, err := openaicompat.Translate(reversed)
	if err != nil {
		t.Fatalf("Translate(reversed): unexpected error: %v", err)
	}

	if string(forwardBytes) == string(reversedBytes) {
		t.Fatalf("Translate(forward) == Translate(reversed) = %s, want different bytes — order must be carried, not incidentally equal", forwardBytes)
	}
}

// TestSystemSegment_NoSystemInstruction_HasNoPlaceholderEntry proves
// S-ART-020: a request carrying no system instruction at all translates
// with no empty or placeholder system entry — absence renders as
// absence, distinctly named here rather than left to be inferred from
// translation_test.go's own unrelated cases, none of which attach a
// system instruction either.
func TestSystemSegment_NoSystemInstruction_HasNoPlaceholderEntry(t *testing.T) {
	request, err := ai.NewRequest("gpt-4o", []ai.Message{
		mustMessage(ai.NewMessage(ai.RoleUser, mustPart(ai.NewText("No system instruction is attached to this request.")))),
	})
	if err != nil {
		t.Fatalf("ai.NewRequest returned %v, want no failure", err)
	}

	got, err := openaicompat.Translate(request)
	if err != nil {
		t.Fatalf("Translate: unexpected error: %v", err)
	}
	want := `{"model":"gpt-4o","messages":[{"role":"user","content":"No system instruction is attached to this request."}],"stream":true,"stream_options":{"include_usage":true}}`
	if string(got) != want {
		t.Fatalf("Translate() =\n%s\nwant\n%s", got, want)
	}
}
