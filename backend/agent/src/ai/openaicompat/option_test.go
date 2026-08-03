// AI-26.7 — every neutral generation option maps to its vendor field
// (R-ART-016); the usage opt-in's total, positive assertion (R-ART-017);
// the output-token limit's explicit-absence pin (R-ART-018).
//
// The escape hatch's own half of this node (R-ART-019, tasks.md 4.7-4.11)
// is NOT covered by this file — see tasks.md's Phase 4 evidence log and
// this package's doc.go "Escape hatch: reserved namespace not yet defined
// upstream" section for why: AI-25's landed artifact, searched
// exhaustively (client.go, credential.go, endpoint.go, request.go,
// doc.go, its own SDD proposal/spec/design/tasks, AI-24's decision, and
// doc 0002's AI-25 charter), never defines a reserved provider-extension
// namespace value, and this node's own instruction forbids inventing one
// here.
package openaicompat_test

import (
	"strings"
	"testing"

	"github.com/cachicamas/backend/agent/src/ai"
	"github.com/cachicamas/backend/agent/src/ai/openaicompat"
)

func init() {
	registerExpectations(
		expectationCase{
			name: "max output tokens option present with caller value",
			build: func() ai.Request {
				return mustRequest(ai.NewRequest("gpt-4o", []ai.Message{
					mustMessage(ai.NewMessage(ai.RoleUser, mustPart(ai.NewText("Hello, world!")))),
				}, ai.WithMaxOutputTokens(500)))
			},
			want: `{"model":"gpt-4o","messages":[{"role":"user","content":"Hello, world!"}],"max_tokens":500,"stream":true,"stream_options":{"include_usage":true}}`,
		},
		expectationCase{
			name: "temperature option present with caller value",
			build: func() ai.Request {
				return mustRequest(ai.NewRequest("gpt-4o", []ai.Message{
					mustMessage(ai.NewMessage(ai.RoleUser, mustPart(ai.NewText("Hello, world!")))),
				}, ai.WithTemperature(0.7)))
			},
			want: `{"model":"gpt-4o","messages":[{"role":"user","content":"Hello, world!"}],"temperature":0.7,"stream":true,"stream_options":{"include_usage":true}}`,
		},
		expectationCase{
			name: "top_p option present with caller value",
			build: func() ai.Request {
				return mustRequest(ai.NewRequest("gpt-4o", []ai.Message{
					mustMessage(ai.NewMessage(ai.RoleUser, mustPart(ai.NewText("Hello, world!")))),
				}, ai.WithTopP(0.9)))
			},
			want: `{"model":"gpt-4o","messages":[{"role":"user","content":"Hello, world!"}],"top_p":0.9,"stream":true,"stream_options":{"include_usage":true}}`,
		},
		expectationCase{
			name: "stop sequences option present with caller values in order",
			build: func() ai.Request {
				return mustRequest(ai.NewRequest("gpt-4o", []ai.Message{
					mustMessage(ai.NewMessage(ai.RoleUser, mustPart(ai.NewText("Hello, world!")))),
				}, ai.WithStopSequences("END", "STOP")))
			},
			want: `{"model":"gpt-4o","messages":[{"role":"user","content":"Hello, world!"}],"stop":["END","STOP"],"stream":true,"stream_options":{"include_usage":true}}`,
		},
		expectationCase{
			name: "tool_choice none renders the bare vendor string, no tools declared",
			build: func() ai.Request {
				choice := mustToolChoice(ai.NewToolChoice(ai.ToolChoiceNone))
				return mustRequest(ai.NewRequest("gpt-4o", []ai.Message{
					mustMessage(ai.NewMessage(ai.RoleUser, mustPart(ai.NewText("Hello, world!")))),
				}, ai.WithToolChoice(choice)))
			},
			want: `{"model":"gpt-4o","messages":[{"role":"user","content":"Hello, world!"}],"tool_choice":"none","stream":true,"stream_options":{"include_usage":true}}`,
		},
		expectationCase{
			name: "tool_choice naming a specific tool renders the vendor's function object, composed after tools",
			build: func() ai.Request {
				tool := mustTool(ai.NewTool("get_weather", "Get the current weather for a location.", []byte(`{"type":"object"}`)))
				toolSet := mustToolSet(ai.NewToolSet(tool))
				choice := mustToolChoice(ai.NewNamedToolChoice("get_weather"))
				return mustRequest(ai.NewRequest("gpt-4o", []ai.Message{
					mustMessage(ai.NewMessage(ai.RoleUser, mustPart(ai.NewText("What is the weather in Paris?")))),
				}, ai.WithTools(toolSet), ai.WithToolChoice(choice)))
			},
			want: `{"model":"gpt-4o","messages":[{"role":"user","content":"What is the weather in Paris?"}],` +
				`"tools":[{"type":"function","function":{"name":"get_weather","description":"Get the current weather for a location.","parameters":{"type":"object"}}}],` +
				`"tool_choice":{"type":"function","function":{"name":"get_weather"}},` +
				`"stream":true,"stream_options":{"include_usage":true}}`,
		},
	)
}

// mustToolChoice panics if err != nil. See mustRequest in translation_test.go.
func mustToolChoice(choice ai.ToolChoice, err error) ai.ToolChoice {
	if err != nil {
		panic(err)
	}
	return choice
}

// TestOption_ZeroValueVersusUnset_TranslatesDifferently proves R-ART-016 /
// S-ART-058 for the one option among this node's settable options that
// admits a legal zero value at all: temperature (>= 0, no upper bound —
// request.go's own boundsRule). The other four candidates have no legal
// zero-value counterpart to test against: max_tokens must be strictly
// positive (0 is ErrOutOfRange), top_p must be in (0, 1] (0 is
// ErrOutOfRange), stop sequences must carry at least one non-empty entry
// when applied (an empty slice is ErrEmpty), and a zero ToolChoice is not
// a vocabulary member at all (ToolChoice.ValidateAgainst's own rule 1).
// Temperature is also the realistic, common case this scenario exists to
// guard against: temperature: 0 requests fully deterministic sampling, a
// legitimate, frequently-used value an implementation keyed on Go's own
// float zero value (rather than the presence flag) would wrongly treat as
// "not set".
func TestOption_ZeroValueVersusUnset_TranslatesDifferently(t *testing.T) {
	message := mustMessage(ai.NewMessage(ai.RoleUser, mustPart(ai.NewText("Hello, world!"))))

	withZero := mustRequest(ai.NewRequest("gpt-4o", []ai.Message{message}, ai.WithTemperature(0)))
	unset := mustRequest(ai.NewRequest("gpt-4o", []ai.Message{message}))

	zeroBytes, err := openaicompat.Translate(withZero)
	if err != nil {
		t.Fatalf("Translate(temperature=0): unexpected error: %v", err)
	}
	unsetBytes, err := openaicompat.Translate(unset)
	if err != nil {
		t.Fatalf("Translate(temperature unset): unexpected error: %v", err)
	}

	if string(zeroBytes) == string(unsetBytes) {
		t.Fatalf("temperature=0 and temperature-unset produced identical bytes = %s, want different bodies — presence must be respected, not inferred from value", zeroBytes)
	}

	wantZero := `{"model":"gpt-4o","messages":[{"role":"user","content":"Hello, world!"}],"temperature":0,"stream":true,"stream_options":{"include_usage":true}}`
	if string(zeroBytes) != wantZero {
		t.Fatalf("Translate(temperature=0) =\n%s\nwant\n%s", zeroBytes, wantZero)
	}
	wantUnset := `{"model":"gpt-4o","messages":[{"role":"user","content":"Hello, world!"}],"stream":true,"stream_options":{"include_usage":true}}`
	if string(unsetBytes) != wantUnset {
		t.Fatalf("Translate(temperature unset) =\n%s\nwant\n%s", unsetBytes, wantUnset)
	}
}

// TestOption_MaxOutputTokensOmitted_FieldExplicitlyAbsent proves R-ART-018
// / S-ART-062: the mandatory-output-limit branch is a deliberate no-op for
// this vendor (doc.go's own section states why, citing doc 0002's merged
// amendment) — a request that never calls WithMaxOutputTokens translates
// to a body with no "max_tokens" field anywhere, asserted directly rather
// than inferred from the byte-exact expectation walk. S-ART-063 (the
// option present with the caller's value) is the registered "max output
// tokens option present with caller value" case above; the two together
// are what makes explicit absence non-vacuous — the field is asserted
// present when set (registered case) and asserted absent when it is not
// (this test), rather than merely missing from an expectation literal
// that could simply have forgotten to mention it.
func TestOption_MaxOutputTokensOmitted_FieldExplicitlyAbsent(t *testing.T) {
	req := mustRequest(ai.NewRequest("gpt-4o", []ai.Message{
		mustMessage(ai.NewMessage(ai.RoleUser, mustPart(ai.NewText("Hello, world!")))),
	}))
	got, err := openaicompat.Translate(req)
	if err != nil {
		t.Fatalf("Translate: unexpected error: %v", err)
	}
	if strings.Contains(string(got), "max_tokens") {
		t.Fatalf("Translate() = %s, want no \"max_tokens\" field at all when the option is unset", got)
	}
}
