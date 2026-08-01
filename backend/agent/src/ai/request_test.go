// Tests for AI-10.1 — the walking skeleton of the normalized request.
//
// External package, for AI-06's reason: the consumer this contract exists for
// is an adapter in a vendor package, and doc 0002 makes readability from
// outside constitutive of the request rather than incidental to it. Defect C2
// was a Layer 1 whose request could not be read from another package at all.

package ai_test

import (
	"bytes"
	"errors"
	"fmt"
	"slices"
	"strings"
	"testing"

	"github.com/cachicamas/backend/agent/src/ai"
)

// requireViolation asserts that a failure is AI-04's one concrete type, reports
// the expected rule class, and renders the expected position.
//
// Both queries are made, because AI-04's whole shape is two axes with one
// mechanism each: errors.Is names the class, errors.As names the position. A
// test that made only the first would pass against a rule reported at the wrong
// region, which is the failure this package's positions exist to prevent.
func requireViolation(t *testing.T, err error, wantRule error, wantPath string) {
	t.Helper()

	if err == nil {
		t.Fatalf("got no failure, want %v at %q", wantRule, wantPath)
	}
	if !errors.Is(err, wantRule) {
		t.Errorf("errors.Is(err, %v) = false on %v", wantRule, err)
	}
	var violation *ai.Violation
	if !errors.As(err, &violation) {
		t.Fatalf("errors.As(err, *ai.Violation) = false on %v", err)
	}
	if got := violation.Path().String(); got != wantPath {
		t.Errorf("violation.Path() = %q, want %q", got, wantPath)
	}
}

// userTextMessage builds the one message the skeleton needs, failing the test
// rather than the request when a dependency of this milestone misbehaves.
func userTextMessage(t *testing.T, text string) ai.Message {
	t.Helper()

	part, err := ai.NewText(text)
	if err != nil {
		t.Fatalf("ai.NewText returned %v, want no failure", err)
	}
	message, err := ai.NewMessage(ai.RoleUser, part)
	if err != nil {
		t.Fatalf("ai.NewMessage returned %v, want no failure", err)
	}
	return message
}

// AI-10.1 item 1 — the walking skeleton.
//
// A model identity and one user text message construct, validate, and both read
// back exactly from another package (V-REQ-20, V-REQ-21).
func TestRequest_ModelAndOneUserTextMessage_ReadBackFromAnExternalPackage(t *testing.T) {
	t.Parallel()

	const (
		model = "cachicamas-neutral-model-1"
		text  = "Summarise the diff.\n"
	)

	request, err := ai.NewRequest(model, []ai.Message{userTextMessage(t, text)})
	if err != nil {
		t.Fatalf("ai.NewRequest returned %v, want no failure", err)
	}

	if got := request.Model(); got != model {
		t.Errorf("request.Model() = %q, want %q", got, model)
	}

	messages := request.Messages()
	if len(messages) != 1 {
		t.Fatalf("request.Messages() has %d elements, want 1", len(messages))
	}
	if got := messages[0].Role(); got != ai.RoleUser {
		t.Errorf("messages[0].Role() = %v, want %v", got, ai.RoleUser)
	}

	content := messages[0].Content()
	if len(content) != 1 {
		t.Fatalf("messages[0].Content() has %d elements, want 1", len(content))
	}
	read, ok := content[0].Text()
	if !ok {
		t.Fatalf("content[0].Text() reported no text on a part built by ai.NewText")
	}
	if read != text {
		t.Errorf("content[0].Text() = %q, want %q", read, text)
	}
}

// AI-10.1 item 2 — the required regions.
//
// A request has exactly two required regions, and each reports through AI-04's
// vocabulary at its own position (R-AMR-002). Whitespace-only is folded into
// emptiness deliberately: a model name made of spaces names nothing, and
// treating it as present would push the failure to a provider round trip.
func TestNewRequest_MissingRequiredRegion_FailsWithAnAI04Sentinel(t *testing.T) {
	t.Parallel()

	valid := []ai.Message{userTextMessage(t, "hello")}

	cases := []struct {
		what     string
		model    string
		messages []ai.Message
		wantRule error
		wantPath string
	}{
		{"an empty model identity", "", valid, ai.ErrEmpty, "model"},
		{"a whitespace-only model identity", " \t\n ", valid, ai.ErrEmpty, "model"},
		{"a nil message sequence", "m", nil, ai.ErrEmpty, "messages"},
		{"an empty message sequence", "m", []ai.Message{}, ai.ErrEmpty, "messages"},
		{"a message that skipped its constructor", "m", []ai.Message{{}}, ai.ErrEmpty, "messages[0]"},
		{"a skipped message after a valid one", "m", []ai.Message{valid[0], {}}, ai.ErrEmpty, "messages[1]"},
	}

	for _, c := range cases {
		t.Run(c.what, func(t *testing.T) {
			t.Parallel()

			request, err := ai.NewRequest(c.model, c.messages)
			requireViolation(t, err, c.wantRule, c.wantPath)

			if got := request.Model(); got != "" {
				t.Errorf("request.Model() = %q on a failed construction, want the zero request", got)
			}
		})
	}
}

// AI-10.1 item 2, the other direction — a model identity this layer cannot
// recognise is not a caller-contract failure.
//
// The register's own worked borderline case (§ 6.3): an unrecognised model is a
// provider failure, because recognition is not decidable from the request
// alone. Only emptiness is. Without this test the emptiness rule would be one
// careless commit away from becoming a catalog check.
func TestNewRequest_UnrecognisedModelIdentity_ConstructsBecauseRecognitionIsNotDecidableHere(t *testing.T) {
	t.Parallel()

	const model = "no-provider-offers-this-model-2099"

	request, err := ai.NewRequest(model, []ai.Message{userTextMessage(t, "hello")})
	if err != nil {
		t.Fatalf("ai.NewRequest returned %v, want no failure", err)
	}
	if got := request.Model(); got != model {
		t.Errorf("request.Model() = %q, want %q", got, model)
	}
}

// AI-10.1 item 3 — the generation options carry through and read back.
//
// The neutral vocabulary decided by V-REQ-26's admission test, and no more:
// maximum output tokens, temperature, top-p, stop sequences. design.md § 8.2
// names every rejected candidate and the provider that fails it; each is AI-12's
// escape hatch, not a fifth option here.
func TestRequest_GenerationOptions_CarryThroughConstructionAndReadBack(t *testing.T) {
	t.Parallel()

	stops := []string{"</done>", "\n\nHuman:"}

	request, err := ai.NewRequest(
		"m", []ai.Message{userTextMessage(t, "hello")},
		ai.WithMaxOutputTokens(4096),
		ai.WithTemperature(0.2),
		ai.WithTopP(0.9),
		ai.WithStopSequences(stops...),
	)
	if err != nil {
		t.Fatalf("ai.NewRequest returned %v, want no failure", err)
	}

	if got, set := request.MaxOutputTokens(); got != 4096 || !set {
		t.Errorf("request.MaxOutputTokens() = (%d, %t), want (4096, true)", got, set)
	}
	if got, set := request.Temperature(); got != 0.2 || !set {
		t.Errorf("request.Temperature() = (%v, %t), want (0.2, true)", got, set)
	}
	if got, set := request.TopP(); got != 0.9 || !set {
		t.Errorf("request.TopP() = (%v, %t), want (0.9, true)", got, set)
	}
	got, set := request.StopSequences()
	if !set {
		t.Fatalf("request.StopSequences() reported unset, want set")
	}
	if !slices.Equal(got, stops) {
		t.Errorf("request.StopSequences() = %q, want %q", got, stops)
	}
}

// AI-10.1 item 3 — absence is structural, so unset and zero are different
// requests.
//
// V-MET-11's distinction applied on the request side. A sentinel value meaning
// "unset" would make a caller who deliberately asked for a temperature of zero
// indistinguishable from one who asked for nothing, and the two mean opposite
// things to a provider: sample deterministically, versus use your default.
func TestRequest_UnappliedGenerationOption_IsDistinguishableFromOneSetToItsZeroValue(t *testing.T) {
	t.Parallel()

	messages := []ai.Message{userTextMessage(t, "hello")}

	bare, err := ai.NewRequest("m", messages)
	if err != nil {
		t.Fatalf("ai.NewRequest returned %v, want no failure", err)
	}
	if got, set := bare.Temperature(); got != 0 || set {
		t.Errorf("bare.Temperature() = (%v, %t), want (0, false)", got, set)
	}
	if got, set := bare.MaxOutputTokens(); got != 0 || set {
		t.Errorf("bare.MaxOutputTokens() = (%d, %t), want (0, false)", got, set)
	}
	if got, set := bare.TopP(); got != 0 || set {
		t.Errorf("bare.TopP() = (%v, %t), want (0, false)", got, set)
	}
	if got, set := bare.StopSequences(); got != nil || set {
		t.Errorf("bare.StopSequences() = (%q, %t), want (nil, false)", got, set)
	}

	zeroed, err := ai.NewRequest("m", messages, ai.WithTemperature(0))
	if err != nil {
		t.Fatalf("ai.NewRequest returned %v, want no failure", err)
	}
	if got, set := zeroed.Temperature(); got != 0 || !set {
		t.Errorf("zeroed.Temperature() = (%v, %t), want (0, true)", got, set)
	}
}

// AI-10.1 item 3 — re-applying an option is last-wins, and that is the seam
// AI-12 rebuilds through.
//
// design.md § 2.2 records this as landed deliberately rather than incidentally:
// a per-request override in AI-12 is the same RequestOption applied again, so
// the behaviour has to be defined before AI-12 depends on it.
func TestRequest_ReappliedGenerationOption_IsLastWins(t *testing.T) {
	t.Parallel()

	request, err := ai.NewRequest(
		"m", []ai.Message{userTextMessage(t, "hello")},
		ai.WithTemperature(0.9),
		ai.WithTemperature(0.1),
	)
	if err != nil {
		t.Fatalf("ai.NewRequest returned %v, want no failure", err)
	}
	if got, set := request.Temperature(); got != 0.1 || !set {
		t.Errorf("request.Temperature() = (%v, %t), want (0.1, true)", got, set)
	}
}

// AI-10.1 item 4 (appended) — option bounds decidable from the request alone.
//
// Discovered while writing item 3: a request carrying a maximum of -1 tokens
// constructed happily, and that is a value no provider accepts, decidable
// without I/O — the register's own definition of a caller-contract failure.
// Appended rather than deferred, because adding a bound later is the tightening
// direction, which proposal.md records as the expensive one.
func TestNewRequest_OutOfBoundGenerationOption_FailsAtItsOwnPosition(t *testing.T) {
	t.Parallel()

	messages := []ai.Message{userTextMessage(t, "hello")}

	cases := []struct {
		what     string
		option   ai.RequestOption
		wantRule error
		wantPath string
	}{
		{"zero maximum output tokens", ai.WithMaxOutputTokens(0), ai.ErrOutOfRange, "maxOutputTokens"},
		{"negative maximum output tokens", ai.WithMaxOutputTokens(-1), ai.ErrOutOfRange, "maxOutputTokens"},
		{"a negative temperature", ai.WithTemperature(-0.1), ai.ErrOutOfRange, "temperature"},
		{"a top-p of zero", ai.WithTopP(0), ai.ErrOutOfRange, "topP"},
		{"a top-p above one", ai.WithTopP(1.0001), ai.ErrOutOfRange, "topP"},
		{"a negative top-p", ai.WithTopP(-0.5), ai.ErrOutOfRange, "topP"},
		{"the stop-sequence option applied with no sequences", ai.WithStopSequences(), ai.ErrEmpty, "stopSequences"},
		{"an empty stop sequence", ai.WithStopSequences("a", ""), ai.ErrEmpty, "stopSequences[1]"},
	}

	for _, c := range cases {
		t.Run(c.what, func(t *testing.T) {
			t.Parallel()

			_, err := ai.NewRequest("m", messages, c.option)
			requireViolation(t, err, c.wantRule, c.wantPath)
		})
	}
}

// AI-10.1 item 4 (appended), the other direction — the bounds that are
// deliberately absent, and the boundary values that are legal.
//
// design.md § 8.3: providers disagree on temperature's cap (1.0 versus 2.0), so
// a bound right for one is a caller-contract failure this package invented for
// the other. A too-high temperature is a provider failure and reports through
// AI-19. Without this test the missing cap reads as an oversight and the next
// reader closes it.
func TestNewRequest_BoundaryGenerationOptions_ConstructBecauseNoRuleForbidsThem(t *testing.T) {
	t.Parallel()

	messages := []ai.Message{userTextMessage(t, "hello")}

	cases := []struct {
		what   string
		option ai.RequestOption
	}{
		{"a temperature above every provider's cap, because this package invents none", ai.WithTemperature(4)},
		{"a temperature of exactly zero", ai.WithTemperature(0)},
		{"a top-p of exactly one", ai.WithTopP(1)},
		{"a maximum of exactly one token", ai.WithMaxOutputTokens(1)},
		{"a stop sequence that is only whitespace, which is a real stop sequence", ai.WithStopSequences(" ")},
	}

	for _, c := range cases {
		t.Run(c.what, func(t *testing.T) {
			t.Parallel()

			if _, err := ai.NewRequest("m", messages, c.option); err != nil {
				t.Errorf("ai.NewRequest returned %v, want no failure", err)
			}
		})
	}
}

// AI-10.1 item 5 (appended) — the message region is copied on the way out.
//
// Discovered while implementing item 3's stop-sequence accessor: that one
// cloned and Messages() did not, so a reader could rewrite the request it was
// handed. AI-10.6 owns immutability as a whole — both directions, plus the
// constructor side and equality — and this appended case covers only the leak
// that was landing here, because shipping it across a commit boundary to be
// found later is how the aliasing bugs message.go warns about survive.
func TestRequest_Messages_AreNotMutableThroughWhatAReaderReceived(t *testing.T) {
	t.Parallel()

	first := userTextMessage(t, "first")
	second := userTextMessage(t, "second")

	request, err := ai.NewRequest("m", []ai.Message{first, second})
	if err != nil {
		t.Fatalf("ai.NewRequest returned %v, want no failure", err)
	}

	read := request.Messages()
	read[0] = second

	again := request.Messages()
	if again[0].ID() != first.ID() {
		t.Errorf("request.Messages()[0] changed after a reader rewrote the slice it received")
	}
}

// AI-10.1 item 7 (appended) — a request renders no region's payload through any
// verb (R-AMR-017).
//
// The request carries every payload in the package at once — the prompt, the
// model's deliberation, tool arguments, tool results — which makes it the
// highest-value leak in Layer 1. AI-07 found exactly this on Reasoning and
// AI-09 landed the same posture on ToolCall and ToolResult; V-FAIL-13 puts it
// on the type rather than on anyone's discipline.
//
// %#v is enumerated deliberately and not by implication. Without GoString it
// falls back to reflection and prints every unexported field, which would make
// the posture a property of which verb a caller reached for.
func TestRequest_Formatting_RendersNoRegionPayloadThroughAnyVerb(t *testing.T) {
	t.Parallel()

	const (
		modelSecret = "SECRET-MODEL-IDENTITY"
		textSecret  = "SECRET-PROMPT-BODY"
		stopSecret  = "SECRET-STOP-SEQUENCE"
	)

	request, err := ai.NewRequest(
		modelSecret, []ai.Message{userTextMessage(t, textSecret)},
		ai.WithStopSequences(stopSecret),
		ai.WithTemperature(0.5),
	)
	if err != nil {
		t.Fatalf("ai.NewRequest returned %v, want no failure", err)
	}

	verbs := []string{"%v", "%s", "%+v", "%#v"}
	secrets := map[string]string{
		"the model identity": modelSecret,
		"a message's text":   textSecret,
		"a stop sequence":    stopSecret,
	}

	for _, verb := range verbs {
		t.Run(verb, func(t *testing.T) {
			t.Parallel()

			rendered := fmt.Sprintf(verb, request)
			for region, secret := range secrets {
				if strings.Contains(rendered, secret) {
					t.Errorf("fmt.Sprintf(%q, request) leaked %s: %q", verb, region, rendered)
				}
			}
		})
	}
}

// AI-10.1 item 7 (appended), the other direction — the rendering still says
// something.
//
// Redaction that renders nothing is not a posture, it is a deleted method. A
// diagnostic reader must still learn the request's shape: which regions are
// present, and how many elements each ordered region holds.
func TestRequest_Formatting_NamesThePresentRegionsAndTheirElementCounts(t *testing.T) {
	t.Parallel()

	request, err := ai.NewRequest(
		"m", []ai.Message{userTextMessage(t, "a"), userTextMessage(t, "b")},
		ai.WithTemperature(0.5),
	)
	if err != nil {
		t.Fatalf("ai.NewRequest returned %v, want no failure", err)
	}

	rendered := request.String()
	for _, want := range []string{"request(", "model", "2 messages", "temperature"} {
		if !strings.Contains(rendered, want) {
			t.Errorf("request.String() = %q, want it to contain %q", rendered, want)
		}
	}
	for _, unwanted := range []string{"topP", "maxOutputTokens", "stopSequences"} {
		if strings.Contains(rendered, unwanted) {
			t.Errorf("request.String() = %q, want it to omit the unapplied option %q", rendered, unwanted)
		}
	}

	if got := fmt.Sprintf("%#v", request); got != rendered {
		t.Errorf("fmt.Sprintf(\"%%#v\", request) = %q, want it to match String() = %q", got, rendered)
	}
}

// --- AI-10.3 — messages, tools and tool choice on the request ----------------

// requireMessage builds a message of a given role, failing the test rather than
// the request when a dependency of this milestone misbehaves.
//
// Every AI-10.3 test needs messages of all three roles carrying parts of all
// four kinds, and a builder that reported a dependency's failure as this
// milestone's would hide the one thing these tests are about.
func requireMessage(t *testing.T, role ai.Role, parts ...ai.Part) ai.Message {
	t.Helper()

	message, err := ai.NewMessage(role, parts...)
	if err != nil {
		t.Fatalf("ai.NewMessage(%v) returned %v, want no failure", role, err)
	}
	return message
}

// The four part builders these tests compose messages from.
//
// One per kind rather than one generic unwrapper, because Go will not forward a
// two-result call into a helper that also takes *testing.T, and because a
// builder named for its kind is what makes a role/kind table row readable as
// one line.
func textPart(t *testing.T, text string) ai.Part {
	t.Helper()

	part, err := ai.NewText(text)
	if err != nil {
		t.Fatalf("ai.NewText returned %v, want no failure", err)
	}
	return part
}

func reasoningPart(t *testing.T, text string) ai.Part {
	t.Helper()

	part, err := ai.NewReasoning(text, nil)
	if err != nil {
		t.Fatalf("ai.NewReasoning returned %v, want no failure", err)
	}
	return part
}

func toolCallPart(t *testing.T, id, name string) ai.Part {
	t.Helper()

	part, err := ai.NewToolCall(id, name, []byte(`{"q":"go"}`))
	if err != nil {
		t.Fatalf("ai.NewToolCall returned %v, want no failure", err)
	}
	return part
}

// partKinds projects a message's content onto the kinds it carries, in order,
// which is the form an order assertion is readable in.
func partKinds(content []ai.Part) []ai.PartKind {
	kinds := make([]ai.PartKind, 0, len(content))
	for _, part := range content {
		kinds = append(kinds, part.Kind())
	}
	return kinds
}

// AI-10.3 item 1 *(pin)* — message order and intra-message content order are
// preserved exactly through construction and readback (R-AMR-008).
//
// S-AMR-033, S-AMR-034 and S-AMR-035 in one test, because they are three
// statements about one property and splitting them would let two pass while the
// property was broken.
//
// Green from birth: AI-10.1 copied the message sequence and AI-05 copies a
// message's content, so the property is already there. It is pinned rather than
// assumed because *every* cross-region rule AI-10.3 lands reads content by
// index — the duplicate rule reports "the second occurrence", the correlation
// rule scans "the whole request in order" — and a position by index is
// meaningless the moment order is not preserved. Both scratch violations are
// recorded biting in tasks.md.
func TestRequest_MessageAndContentOrder_ArePreservedThroughConstructionAndReadback(t *testing.T) {
	t.Parallel()

	// S-AMR-034 — one assistant message carrying three kinds in a known order.
	// Assistant, because design.md § 5.1 is the only role that may carry all
	// three, and a test that used a forbidden cell here would start failing on
	// item 3 for a reason that had nothing to do with order.
	mixed := requireMessage(t, ai.RoleAssistant,
		textPart(t, "first"),
		reasoningPart(t, "second"),
		toolCallPart(t, "call-1", "search"),
	)

	// S-AMR-033 — five messages in a known order, distinguishable by their text.
	messages := []ai.Message{
		userTextMessage(t, "one"),
		mixed,
		userTextMessage(t, "three"),
		userTextMessage(t, "four"),
		userTextMessage(t, "five"),
	}

	request, err := ai.NewRequest("m", messages)
	if err != nil {
		t.Fatalf("ai.NewRequest returned %v, want no failure", err)
	}

	read := request.Messages()
	if len(read) != len(messages) {
		t.Fatalf("request.Messages() returned %d messages, want %d", len(read), len(messages))
	}
	for i := range messages {
		if read[i].ID() != messages[i].ID() {
			t.Errorf("request.Messages()[%d] is not the message built at position %d — "+
				"the sequence was reordered between construction and readback", i, i)
		}
	}

	wantKinds := []ai.PartKind{ai.PartKindText, ai.PartKindReasoning, ai.PartKindToolCall}
	if got := partKinds(read[1].Content()); !slices.Equal(got, wantKinds) {
		t.Errorf("the second message's content kinds = %v, want %v", got, wantKinds)
	}
	if text, ok := read[1].Content()[0].Text(); !ok || text != "first" {
		t.Errorf("the second message's first part = (%q, %t), want (%q, true)", text, ok, "first")
	}

	// S-AMR-035 — the reader holds a copy. Reordering what it received must not
	// reorder the request, and the second reader must not observe the first.
	slices.Reverse(read)
	reread := request.Messages()
	for i := range messages {
		if reread[i].ID() != messages[i].ID() {
			t.Errorf("after the reader reversed the slice it was handed, request.Messages()[%d] "+
				"is not the message built at position %d — the request handed out its own storage", i, i)
		}
	}
}

// requireToolSet builds a tool set from names, failing the test on AI-08's
// errors rather than reporting them as this milestone's.
func requireToolSet(t *testing.T, names ...string) ai.ToolSet {
	t.Helper()

	tools := make([]ai.Tool, 0, len(names))
	for _, name := range names {
		tool, err := ai.NewTool(name, "a declared tool", []byte(`{"type":"object"}`))
		if err != nil {
			t.Fatalf("ai.NewTool(%q) returned %v, want no failure", name, err)
		}
		tools = append(tools, tool)
	}
	set, err := ai.NewToolSet(tools...)
	if err != nil {
		t.Fatalf("ai.NewToolSet returned %v, want no failure", err)
	}
	return set
}

// requireNamedToolChoice builds a specific tool choice, failing the test on
// AI-08's errors.
func requireNamedToolChoice(t *testing.T, name string) ai.ToolChoice {
	t.Helper()

	choice, err := ai.NewNamedToolChoice(name)
	if err != nil {
		t.Fatalf("ai.NewNamedToolChoice(%q) returned %v, want no failure", name, err)
	}
	return choice
}

// AI-10.3 item 2 — the tool set and the tool choice attach to the request, and
// both read back from another package (R-AMR-010, S-AMR-038).
//
// They are two independently optional regions, so each carries its own applied
// flag. S-AMR-042 is the reason: a request with tools and no choice is legal,
// and it is a different request from one that chose [ai.ToolChoiceNone].
// Collapsing the two would delete a distinction a provider acts on.
func TestRequest_ToolSetAndToolChoice_AttachAndReadBack(t *testing.T) {
	t.Parallel()

	tools := requireToolSet(t, "search", "fetch", "write")
	choice := requireNamedToolChoice(t, "fetch")

	request, err := ai.NewRequest(
		"m", []ai.Message{userTextMessage(t, "go")},
		ai.WithTools(tools), ai.WithToolChoice(choice),
	)
	if err != nil {
		t.Fatalf("ai.NewRequest returned %v, want no failure", err)
	}

	readTools, hasTools := request.Tools()
	if !hasTools {
		t.Fatalf("request.Tools() reported no tool set, want the one applied")
	}
	gotNames := make([]string, 0, readTools.Len())
	for _, tool := range readTools.Tools() {
		gotNames = append(gotNames, tool.Name())
	}
	if want := []string{"search", "fetch", "write"}; !slices.Equal(gotNames, want) {
		t.Errorf("the request's tool declarations = %v, want %v in the order supplied", gotNames, want)
	}

	readChoice, hasChoice := request.ToolChoice()
	if !hasChoice {
		t.Fatalf("request.ToolChoice() reported no choice, want the one applied")
	}
	if got := readChoice.Mode(); got != ai.ToolChoiceSpecific {
		t.Errorf("the request's tool-choice mode = %v, want %v", got, ai.ToolChoiceSpecific)
	}
	if name, ok := readChoice.Name(); !ok || name != "fetch" {
		t.Errorf("the request's tool-choice name = (%q, %t), want (%q, true)", name, ok, "fetch")
	}

	// S-AMR-042 — omitting the choice is not choosing none.
	withoutChoice, err := ai.NewRequest(
		"m", []ai.Message{userTextMessage(t, "go")}, ai.WithTools(tools),
	)
	if err != nil {
		t.Fatalf("ai.NewRequest with tools and no choice returned %v, want no failure", err)
	}
	if _, hasChoice := withoutChoice.ToolChoice(); hasChoice {
		t.Errorf("a request built without ai.WithToolChoice reports a choice, want none — " +
			"omitting the choice and choosing ToolChoiceNone are different requests")
	}
}

// AI-10.3 item 2 — AI-08.3's cross-validation runs at the request boundary,
// with its three rules and its positions unchanged (R-AMR-010).
//
// The positions are asserted and not only the classes, because the position is
// the observable proof that [ai.ToolChoice.ValidateAgainst] was *called* rather
// than reimplemented here. A reimplementation would report at a request-shaped
// position — "toolChoice" beneath something — and pass a class-only assertion.
// tool_choice.go names this caller by name: "AI-10.3 re-runs them at the request
// boundary, which is why they live on a method a request can call".
func TestNewRequest_ToolChoiceAgainstTheToolSet_ReportsAI08sRulesAtTheirOwnPositions(t *testing.T) {
	t.Parallel()

	messages := []ai.Message{userTextMessage(t, "go")}

	cases := []struct {
		name     string
		options  []ai.RequestOption
		wantRule error
		wantPath string
	}{{
		// S-AMR-041 — AI-08.3 rule 1. A choice that skipped its constructor.
		name:     "a tool choice that was never constructed",
		options:  []ai.RequestOption{ai.WithToolChoice(ai.ToolChoice{})},
		wantRule: ai.ErrNotInVocabulary,
		wantPath: "toolChoice",
	}, {
		// S-AMR-039 — AI-08.3 rule 2, reported instead of rule 3 although a
		// specific choice against an empty set violates both. Against no tools
		// at all, "you declared no tools" is the more fundamental fact.
		name:     "a specific tool choice and no tool set at all",
		options:  []ai.RequestOption{ai.WithToolChoice(requireNamedToolChoice(t, "fetch"))},
		wantRule: ai.ErrEmpty,
		wantPath: "tools",
	}, {
		// S-AMR-040 — AI-08.3 rule 3.
		name: "a specific tool choice naming an undeclared tool",
		options: []ai.RequestOption{
			ai.WithTools(requireToolSet(t, "search")),
			ai.WithToolChoice(requireNamedToolChoice(t, "fetch")),
		},
		wantRule: ai.ErrUnresolvedReference,
		wantPath: "toolChoice.name",
	}}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			_, err := ai.NewRequest("m", messages, tc.options...)
			requireViolation(t, err, tc.wantRule, tc.wantPath)
		})
	}
}

// AI-10.3 item 2 *(refactor)* — the two new regions render as shape and never
// as payload (R-AMR-017).
//
// AI-10.1's leak table could not cover these regions because they did not exist
// yet, and both carry caller data: a tool's name is the caller's vocabulary, and
// a specific choice names one of them. What renders is the count and the mode —
// a count is shape, and a mode is this package's own closed vocabulary rather
// than anything the caller wrote.
func TestRequest_Formatting_NamesTheToolRegionsWithoutNamingAnyTool(t *testing.T) {
	t.Parallel()

	const chosen = "secret_tool_name"

	request, err := ai.NewRequest(
		"m", []ai.Message{userTextMessage(t, "go")},
		ai.WithTools(requireToolSet(t, "secret_other_tool", chosen)),
		ai.WithToolChoice(requireNamedToolChoice(t, chosen)),
	)
	if err != nil {
		t.Fatalf("ai.NewRequest returned %v, want no failure", err)
	}

	rendered := request.String()
	for _, want := range []string{"2 tools", "toolChoice(specific)"} {
		if !strings.Contains(rendered, want) {
			t.Errorf("request.String() = %q, want it to contain %q", rendered, want)
		}
	}
	for _, verb := range []string{"%v", "%s", "%+v", "%#v"} {
		if got := fmt.Sprintf(verb, request); strings.Contains(got, "secret") {
			t.Errorf("fmt.Sprintf(%q, request) leaked a tool name: %q", verb, got)
		}
	}
}

// toolResultPart completes the four part builders textPart, reasoningPart and
// toolCallPart already define, one per registered kind.
func toolResultPart(t *testing.T, callID, content string) ai.Part {
	t.Helper()

	part, err := ai.NewToolResult(callID, content)
	if err != nil {
		t.Fatalf("ai.NewToolResult returned %v, want no failure", err)
	}
	return part
}

// AI-10.3 item 3 — role versus content kind is enforced from design.md §
// 5.1's table, all twelve cells, in both directions, reporting ErrMisplaced
// (R-AMR-011).
//
// One row per cell, mirroring the table's own shape, so a later loosening —
// design.md § 5.2's stated rollback direction — is one row here too.
//
// The permitted tool_result cell carries a companion message: an assistant
// tool call the result answers. Without it, this cell's request would be an
// orphan tool result the moment item 4 lands that rule, and an already-decided
// permission would start failing for a reason that has nothing to do with
// role versus kind — item 1's test applied the identical discipline to this
// item in advance.
func TestNewRequest_RoleVersusContentKind_EnforcesTheDocumentedTable(t *testing.T) {
	t.Parallel()

	toolResultCompanion := []ai.Message{
		requireMessage(t, ai.RoleAssistant, toolCallPart(t, "table-call", "search")),
	}

	cases := []struct {
		name      string
		role      ai.Role
		part      ai.Part
		companion []ai.Message
		permitted bool
	}{
		// The five permitted cells (S-AMR-046).
		{"text_under_user", ai.RoleUser, textPart(t, "hi"), nil, true},
		{"text_under_assistant", ai.RoleAssistant, textPart(t, "hi"), nil, true},
		{"reasoning_under_assistant", ai.RoleAssistant, reasoningPart(t, "because"), nil, true},
		{"tool_call_under_assistant", ai.RoleAssistant, toolCallPart(t, "c", "search"), nil, true},
		{"tool_result_under_tool", ai.RoleTool, toolResultPart(t, "table-call", "42"), toolResultCompanion, true},

		// The seven forbidden cells: S-AMR-043 (reasoning under user), S-AMR-044
		// (tool_result under assistant), S-AMR-045 (text under tool), and the
		// rest of the table.
		{"reasoning_under_user", ai.RoleUser, reasoningPart(t, "because"), nil, false},
		{"tool_call_under_user", ai.RoleUser, toolCallPart(t, "c", "search"), nil, false},
		{"tool_result_under_user", ai.RoleUser, toolResultPart(t, "c", "42"), nil, false},
		{"tool_result_under_assistant", ai.RoleAssistant, toolResultPart(t, "c", "42"), nil, false},
		{"text_under_tool", ai.RoleTool, textPart(t, "hi"), nil, false},
		{"reasoning_under_tool", ai.RoleTool, reasoningPart(t, "because"), nil, false},
		{"tool_call_under_tool", ai.RoleTool, toolCallPart(t, "c", "search"), nil, false},
	}
	if len(cases) != 12 {
		t.Fatalf("the table has %d cases, want 12 — one per (kind, role) cell", len(cases))
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			messages := append([]ai.Message{requireMessage(t, tc.role, tc.part)}, tc.companion...)
			_, err := ai.NewRequest("m", messages)
			if tc.permitted {
				if err != nil {
					t.Fatalf("ai.NewRequest returned %v, want no failure — %v is permitted under %v", err, tc.part.Kind(), tc.role)
				}
				return
			}
			requireViolation(t, err, ai.ErrMisplaced, "messages[0].content[0]")
		})
	}
}

// AI-10.3 item 4 — tool-call and tool-result correlation is checked across
// the request (R-AMR-012, design.md §§ 6, 7): an orphan result is rejected,
// an orphan call is legal, a duplicate call identity is rejected at its
// second occurrence, and a result preceding its call is legal — the
// deliberate non-decision about ordering, pinned rather than assumed.
func TestNewRequest_ToolCallAndResultCorrelation_ChecksAcrossTheRequest(t *testing.T) {
	t.Parallel()

	t.Run("an_orphan_tool_result_is_rejected", func(t *testing.T) {
		t.Parallel()

		// S-AMR-047 — a result naming a call identity the request never
		// declares.
		messages := []ai.Message{
			requireMessage(t, ai.RoleTool, toolResultPart(t, "no-such-call", "42")),
		}
		_, err := ai.NewRequest("m", messages)
		requireViolation(t, err, ai.ErrUnresolvedReference, "messages[0].content[0]")
	})

	t.Run("an_orphan_tool_call_is_legal", func(t *testing.T) {
		t.Parallel()

		// S-AMR-048 — a call awaiting its result is the ordinary mid-turn
		// state; repairing it is V-OUT-02's, one layer up.
		messages := []ai.Message{
			requireMessage(t, ai.RoleAssistant, toolCallPart(t, "call-1", "search")),
		}
		if _, err := ai.NewRequest("m", messages); err != nil {
			t.Fatalf("ai.NewRequest returned %v, want no failure — an unanswered call is legal", err)
		}
	})

	t.Run("a_duplicate_call_identity_fails_at_the_second_occurrence", func(t *testing.T) {
		t.Parallel()

		// S-AMR-049 — two tool calls sharing one identity, in two messages so
		// the reported position also proves the scan crosses message
		// boundaries and not only content within one message.
		messages := []ai.Message{
			requireMessage(t, ai.RoleAssistant, toolCallPart(t, "dup", "search")),
			requireMessage(t, ai.RoleAssistant, toolCallPart(t, "dup", "fetch")),
		}
		_, err := ai.NewRequest("m", messages)
		requireViolation(t, err, ai.ErrDuplicate, "messages[1].content[0]")
	})

	t.Run("a_result_before_its_call_is_legal", func(t *testing.T) {
		t.Parallel()

		// S-AMR-050 — the deliberate non-decision about ordering, pinned: a
		// call must exist "anywhere in the request", not necessarily earlier.
		messages := []ai.Message{
			requireMessage(t, ai.RoleTool, toolResultPart(t, "call-1", "42")),
			requireMessage(t, ai.RoleAssistant, toolCallPart(t, "call-1", "search")),
		}
		if _, err := ai.NewRequest("m", messages); err != nil {
			t.Fatalf("ai.NewRequest returned %v, want no failure — ordering repair is V-OUT-02's, not this layer's", err)
		}
	})
}

// --- AI-10.4 — validation happens once, before I/O ---------------------------

// AI-10.4 item 1 — the first failure in the documented order is reported,
// identically across many runs (R-AMR-014, design.md § 9.1).
//
// design.md § 4 already makes the order data — a []Rule handed to
// FirstFailure, evaluated in slice order, with no map iteration anywhere in
// the path — so this test asserts the property rather than building it. It is
// green from birth, and closed the way every green-from-birth check in this
// package is: by showing it bite a scratch violation (recorded in tasks.md).
func TestNewRequest_MultipleViolations_ReportsTheDocumentedOrderFirstAcrossManyRuns(t *testing.T) {
	t.Parallel()

	const runs = 100

	t.Run("model_and_messages_both_violated_reports_the_model_failure", func(t *testing.T) {
		t.Parallel()

		// S-AMR-053 — rule 1 (model) precedes rule 2 (messages); both empty at
		// once must always report the model failure, never the messages one.
		for range runs {
			_, err := ai.NewRequest("", nil)
			requireViolation(t, err, ai.ErrEmpty, "model")
		}
	})

	t.Run("four_regions_violated_at_once_reports_the_earliest_rule", func(t *testing.T) {
		t.Parallel()

		// S-AMR-054 — one rule violated from each of four regions: system
		// (rule 5, an applied-but-unconstructed instruction), an orphan tool
		// result (rule 8), a tool choice against an undeclared tool set
		// (rule 9), and an out-of-range temperature (rule 10). Rule 5 is the
		// earliest of the four, so it must win on every run.
		for range runs {
			_, err := ai.NewRequest(
				"m",
				[]ai.Message{requireMessage(t, ai.RoleTool, toolResultPart(t, "no-such-call", "42"))},
				ai.WithSystemInstruction(ai.SystemInstruction{}),
				ai.WithToolChoice(requireNamedToolChoice(t, "fetch")),
				ai.WithTemperature(-1),
			)
			requireViolation(t, err, ai.ErrEmpty, "system")
		}
	})
}

// requireTool builds a tool declaration, failing the test on AI-08's errors
// rather than reporting them as this milestone's.
func requireTool(t *testing.T, name string) ai.Tool {
	t.Helper()

	tool, err := ai.NewTool(name, "a declared tool", []byte(`{"type":"object"}`))
	if err != nil {
		t.Fatalf("ai.NewTool(%q) returned %v, want no failure", name, err)
	}
	return tool
}

// AI-10.4 item 2, first clause — a request that constructed successfully
// re-examines clean, region by region, against each region's own contract
// (R-AMR-013, design.md § 9.2, S-AMR-052).
//
// Content is re-examined by rebuilding a message from what was read back:
// [ai.Part]'s own validate method is unexported, so reconstruction through
// [ai.NewMessage] — which calls it — is the honest external-package proxy.
// Tool choice is re-examined by calling [ai.ToolChoice.ValidateAgainst] again
// on the read-back tool set, the same method NewRequest's rule 9 calls.
//
// This sub-test is green from birth for the same reason design.md § 9.2
// gives: it asserts that the call happened, not the rule itself, and the rule
// is proven elsewhere (AI-10.1's content tests, AI-10.3 item 2's
// cross-validation table). It is closed by showing it bite in tasks.md,
// against a scratch request.go with rule 9 temporarily removed.
func TestNewRequest_ThatConstructed_ReExaminesCleanUnderEveryRegionsOwnContract(t *testing.T) {
	t.Parallel()

	tools := requireToolSet(t, "search", "fetch")
	choice := requireNamedToolChoice(t, "fetch")
	system, err := ai.NewSystemText("be terse")
	if err != nil {
		t.Fatalf("ai.NewSystemText returned %v, want no failure", err)
	}

	messages := []ai.Message{
		userTextMessage(t, "hello"),
		requireMessage(t, ai.RoleAssistant, toolCallPart(t, "call-1", "search")),
		requireMessage(t, ai.RoleTool, toolResultPart(t, "call-1", "42")),
	}

	request, err := ai.NewRequest("m", messages,
		ai.WithSystemInstruction(system),
		ai.WithTools(tools),
		ai.WithToolChoice(choice),
		ai.WithTemperature(0.5),
	)
	if err != nil {
		t.Fatalf("ai.NewRequest returned %v, want no failure", err)
	}

	for i, message := range request.Messages() {
		if _, err := ai.NewMessage(message.Role(), message.Content()...); err != nil {
			t.Errorf("re-validating messages[%d]'s content returned %v, want no failure — "+
				"a request cannot hand out content its own message contract would reject", i, err)
		}
	}

	readTools, hasTools := request.Tools()
	readChoice, hasChoice := request.ToolChoice()
	if !hasTools || !hasChoice {
		t.Fatalf("request.Tools() = (_, %t), request.ToolChoice() = (_, %t), want both true", hasTools, hasChoice)
	}
	if err := readChoice.ValidateAgainst(readTools); err != nil {
		t.Errorf("readChoice.ValidateAgainst(readTools) returned %v, want no failure — "+
			"a tool choice this package attached to a request must still validate against that request's own tool set", err)
	}
}

// AI-10.4 item 2, second clause — a duplicate tool name cannot reach a
// request, because [ai.NewToolSet] already refuses to construct a set that
// holds one (R-AMR-013, design.md § 9.2).
//
// This is the asymmetric case design.md names explicitly: there is no
// separate duplicate-name rule at the request boundary to test, because
// [ai.WithTools] can never be handed a [ai.ToolSet] that holds one. The bite
// proof for the underlying rule already lives in tool_set_test.go (AI-08's
// own suite); this test's role is only the integration fact — that the
// unreachability is what makes R-AMR-013's second clause true at the request
// level, not a duplicate of AI-08's coverage.
func TestNewRequest_DuplicateToolName_CannotReachARequestBecauseNewToolSetRefusesIt(t *testing.T) {
	t.Parallel()

	_, err := ai.NewToolSet(requireTool(t, "search"), requireTool(t, "search"))
	if err == nil {
		t.Fatalf(`ai.NewToolSet(search, search) constructed, want ErrDuplicate — ` +
			`a request cannot hold what NewToolSet already refuses to build`)
	}
	if !errors.Is(err, ai.ErrDuplicate) {
		t.Errorf("ai.NewToolSet's failure = %v, want errors.Is(err, ai.ErrDuplicate)", err)
	}
}

// --- AI-10.6 — immutability ----------------------------------------------

// AI-10.6 item 1 — mutating anything a reader returned leaves the request
// observably unchanged (R-AMR-016, design.md § 11.1).
//
// Three regions, three sub-tests, because each returns a different element
// type and a generic table would hide more than it would save. "messages"
// restates AI-10.1 item 5's property under this milestone's own requirement
// (S-AMR-057) rather than skipping it: that scenario belongs here formally,
// even though the property was already proven when it landed.
// "stop_sequences" and "system_instruction_segments" close a real gap —
// S-AMR-021 and S-AMR-023 were part of R-AMR-004 and R-AMR-005's scenario
// lists when AI-10.1 and AI-10.2 landed, but neither had a dedicated test
// until this leaf.
func TestRequest_ReadbackMutation_LeavesTheRequestUnchanged(t *testing.T) {
	t.Parallel()

	t.Run("messages", func(t *testing.T) {
		t.Parallel()

		messages := []ai.Message{userTextMessage(t, "one"), userTextMessage(t, "two")}
		request, err := ai.NewRequest("m", messages)
		if err != nil {
			t.Fatalf("ai.NewRequest returned %v, want no failure", err)
		}

		read := request.Messages()
		originalFirstID := read[0].ID()
		slices.Reverse(read) // the reader rewrites what it received

		reread := request.Messages()
		if reread[0].ID() != originalFirstID {
			t.Errorf("after the reader reversed the slice it received, request.Messages()[0] changed — the request handed out its own storage")
		}
	})

	t.Run("stop_sequences", func(t *testing.T) {
		t.Parallel()

		stops := []string{"</a>", "</b>"}
		request, err := ai.NewRequest("m", []ai.Message{userTextMessage(t, "go")}, ai.WithStopSequences(stops...))
		if err != nil {
			t.Fatalf("ai.NewRequest returned %v, want no failure", err)
		}

		read, ok := request.StopSequences()
		if !ok {
			t.Fatalf("request.StopSequences() reported unset, want set")
		}
		read[0] = "SCRATCH-MUTATED" // the reader rewrites what it received

		reread, _ := request.StopSequences()
		if reread[0] != stops[0] {
			t.Errorf("after the reader mutated the slice it received, request.StopSequences()[0] = %q, want %q — "+
				"the request handed out its own storage", reread[0], stops[0])
		}
	})

	t.Run("system_instruction_segments", func(t *testing.T) {
		t.Parallel()

		segmentOne, err := ai.NewSegment("first")
		if err != nil {
			t.Fatalf("ai.NewSegment returned %v, want no failure", err)
		}
		segmentTwo, err := ai.NewSegment("second")
		if err != nil {
			t.Fatalf("ai.NewSegment returned %v, want no failure", err)
		}
		system, err := ai.NewSystemInstruction(segmentOne, segmentTwo)
		if err != nil {
			t.Fatalf("ai.NewSystemInstruction returned %v, want no failure", err)
		}

		request, err := ai.NewRequest("m", []ai.Message{userTextMessage(t, "go")}, ai.WithSystemInstruction(system))
		if err != nil {
			t.Fatalf("ai.NewRequest returned %v, want no failure", err)
		}

		readSystem, hasSystem := request.SystemInstruction()
		if !hasSystem {
			t.Fatalf("request.SystemInstruction() reported absent, want present")
		}
		read := readSystem.Segments()
		slices.Reverse(read) // the reader rewrites what it received

		rereadSystem, _ := request.SystemInstruction()
		reread := rereadSystem.Segments()
		if reread[0] != segmentOne {
			t.Errorf("after the reader reversed the segment slice it received, the request's first segment changed — " +
				"the request handed out its own storage")
		}
	})
}

// AI-10.6 item 2 — mutating the values passed to the constructor leaves the
// constructed request observably unchanged (R-AMR-016, S-AMR-058,
// design.md § 11.1).
//
// This is AI-05.3's property at request scope, and it is the one Go hides:
// message.go documents the variadic-spread trap — a call spread with "..."
// does not copy a slice, so a constructor that did not clone would alias the
// caller's own backing array. Two sub-cases: the message sequence NewRequest
// takes as a parameter, and the stop-sequence slice WithStopSequences takes
// variadic.
func TestNewRequest_ConstructorInputMutatedAfterConstruction_LeavesTheRequestUnchanged(t *testing.T) {
	t.Parallel()

	t.Run("the_message_sequence", func(t *testing.T) {
		t.Parallel()

		messages := []ai.Message{userTextMessage(t, "one"), userTextMessage(t, "two")}
		originalFirstID := messages[0].ID()

		request, err := ai.NewRequest("m", messages)
		if err != nil {
			t.Fatalf("ai.NewRequest returned %v, want no failure", err)
		}

		slices.Reverse(messages) // the caller mutates the slice it passed in

		read := request.Messages()
		if read[0].ID() != originalFirstID {
			t.Errorf("after the caller reversed the slice it passed to ai.NewRequest, request.Messages()[0] changed — " +
				"NewRequest aliased the caller's storage instead of cloning it")
		}
	})

	t.Run("the_stop_sequence_slice", func(t *testing.T) {
		t.Parallel()

		stops := []string{"</a>", "</b>"}

		request, err := ai.NewRequest("m", []ai.Message{userTextMessage(t, "go")}, ai.WithStopSequences(stops...))
		if err != nil {
			t.Fatalf("ai.NewRequest returned %v, want no failure", err)
		}

		stops[0] = "SCRATCH-MUTATED" // the caller mutates the slice it passed in

		read, _ := request.StopSequences()
		if read[0] != "</a>" {
			t.Errorf("after the caller mutated the slice it passed to ai.WithStopSequences, request.StopSequences()[0] = %q, want %q — "+
				"WithStopSequences aliased the caller's storage instead of cloning it", read[0], "</a>")
		}
	})
}

// buildEquatableRequest constructs a request exercising every region, so
// TestRequest_Equal can build two independently and compare them.
func buildEquatableRequest(t *testing.T) ai.Request {
	t.Helper()

	segment, err := ai.NewSegment("be terse")
	if err != nil {
		t.Fatalf("ai.NewSegment returned %v, want no failure", err)
	}
	system, err := ai.NewSystemInstruction(segment)
	if err != nil {
		t.Fatalf("ai.NewSystemInstruction returned %v, want no failure", err)
	}
	tools := requireToolSet(t, "search", "fetch")
	choice := requireNamedToolChoice(t, "fetch")

	request, err := ai.NewRequest(
		"m", []ai.Message{userTextMessage(t, "plan a trip")},
		ai.WithSystemInstruction(system),
		ai.WithTools(tools),
		ai.WithToolChoice(choice),
		ai.WithTemperature(0.5),
		ai.WithStopSequences("</done>"),
	)
	if err != nil {
		t.Fatalf("ai.NewRequest returned %v, want no failure", err)
	}
	return request
}

// AI-10.6 item 3 — two requests constructed from identical inputs compare
// equal by the documented equality, and neither is affected by operations on
// the other (R-AMR-016, S-AMR-059, S-AMR-060, design.md § 11.2).
//
// Equal is exported: design.md § 11.2 records the default as yes, and this is
// the reason cashed — AI-10.5's round trip needs it, AI-26 needs it, and a
// comparison every consumer re-derives is a comparison every consumer gets
// subtly wrong.
func TestRequest_Equal_IdenticalInputsCompareEqualAndOperationsDoNotLeak(t *testing.T) {
	t.Parallel()

	first := buildEquatableRequest(t)
	second := buildEquatableRequest(t)

	// S-AMR-059 — equal despite independently minted message identities. The
	// identity check below is what proves the fixture actually exercises the
	// exclusion design.md § 11.2 states, rather than passing by coincidence.
	if first.Messages()[0].ID() == second.Messages()[0].ID() {
		t.Fatalf("the two requests' first message shares one MessageID — the fixture failed to produce " +
			"independently constructed messages, so this test cannot prove identity is excluded")
	}
	if !first.Equal(second) {
		t.Fatalf("first.Equal(second) = false, want true — two requests built from identical inputs " +
			"must compare equal under the documented equality, message identity excluded")
	}
	if !second.Equal(first) {
		t.Errorf("second.Equal(first) = false, want true — Equal must be symmetric")
	}

	// S-AMR-060 — reading and mutating what a reader received from one must
	// not affect the other.
	read := first.Messages()
	slices.Reverse(read)
	if !first.Equal(second) {
		t.Errorf("first.Equal(second) = false after a reader mutated what it received from first, want true — " +
			"the two requests must remain independent")
	}

	// Triangulation: a request differing in exactly one region is unequal —
	// otherwise Equal could be a function that always returns true.
	third, err := ai.NewRequest("m", []ai.Message{userTextMessage(t, "plan a trip")})
	if err != nil {
		t.Fatalf("ai.NewRequest returned %v, want no failure", err)
	}
	if first.Equal(third) {
		t.Errorf("first.Equal(third) = true, want false — third carries none of first's optional regions")
	}
	if third.Equal(first) {
		t.Errorf("third.Equal(first) = true, want false — Equal must be symmetric about inequality too")
	}
}

// --- AI-12.1 — copy-on-write rebuild ----------------------------------------

// requestSnapshot captures every region of a request through its own
// accessors, so a test can assert "every region unchanged" or "one region
// changed" without repeating nine accessor calls inline (S-REX-002).
//
// It is this file's own snapshot rather than a reuse of agenttest_test's
// requireRequestsEqual: the two packages cannot share a _test.go helper, and
// AI-06's reason for external-package tests — the consumer this contract
// exists for reads from outside — applies to ai_test independently of
// agenttest_test.
type requestSnapshot struct {
	model          string
	messageIDs     []ai.MessageID
	hasSystem      bool
	system         ai.SystemInstruction
	hasTools       bool
	toolNames      []string
	hasChoice      bool
	choiceMode     ai.ToolChoiceMode
	choiceName     string
	choiceNamesOne bool
	maxTokens      int
	hasMaxTokens   bool
	temperature    float64
	hasTemperature bool
	topP           float64
	hasTopP        bool
	stopSequences  []string
	hasStopSeq     bool
}

// snapshotRequest reads every region of r through its exported accessors
// alone, matching the discipline agenttest_test's rebuildFromReadback
// documents: nothing here touches a field directly.
func snapshotRequest(r ai.Request) requestSnapshot {
	s := requestSnapshot{model: r.Model()}
	for _, message := range r.Messages() {
		s.messageIDs = append(s.messageIDs, message.ID())
	}
	s.system, s.hasSystem = r.SystemInstruction()
	if tools, hasTools := r.Tools(); hasTools {
		s.hasTools = true
		for _, tool := range tools.Tools() {
			s.toolNames = append(s.toolNames, tool.Name())
		}
	}
	if choice, hasChoice := r.ToolChoice(); hasChoice {
		s.hasChoice = true
		s.choiceMode = choice.Mode()
		s.choiceName, s.choiceNamesOne = choice.Name()
	}
	s.maxTokens, s.hasMaxTokens = r.MaxOutputTokens()
	s.temperature, s.hasTemperature = r.Temperature()
	s.topP, s.hasTopP = r.TopP()
	s.stopSequences, s.hasStopSeq = r.StopSequences()
	return s
}

// requireSnapshotEqual asserts got and want capture the same regions,
// reporting every mismatch rather than stopping at the first — a derivation
// bug that touches two regions should not hide the second behind the first.
func requireSnapshotEqual(t *testing.T, what string, got, want requestSnapshot) {
	t.Helper()

	if got.model != want.model {
		t.Errorf("%s: Model() = %q, want %q", what, got.model, want.model)
	}
	if !slices.Equal(got.messageIDs, want.messageIDs) {
		t.Errorf("%s: message identities = %v, want %v", what, got.messageIDs, want.messageIDs)
	}
	if got.hasSystem != want.hasSystem || (got.hasSystem && !got.system.Equal(want.system)) {
		t.Errorf("%s: SystemInstruction() = (%v, %t), want (%v, %t)", what, got.system, got.hasSystem, want.system, want.hasSystem)
	}
	if got.hasTools != want.hasTools || !slices.Equal(got.toolNames, want.toolNames) {
		t.Errorf("%s: Tools() names = %v (present %t), want %v (present %t)", what, got.toolNames, got.hasTools, want.toolNames, want.hasTools)
	}
	if got.hasChoice != want.hasChoice || got.choiceMode != want.choiceMode || got.choiceName != want.choiceName || got.choiceNamesOne != want.choiceNamesOne {
		t.Errorf("%s: ToolChoice() = (%v, %q, %t, present %t), want (%v, %q, %t, present %t)",
			what, got.choiceMode, got.choiceName, got.choiceNamesOne, got.hasChoice,
			want.choiceMode, want.choiceName, want.choiceNamesOne, want.hasChoice)
	}
	if got.hasMaxTokens != want.hasMaxTokens || got.maxTokens != want.maxTokens {
		t.Errorf("%s: MaxOutputTokens() = (%d, %t), want (%d, %t)", what, got.maxTokens, got.hasMaxTokens, want.maxTokens, want.hasMaxTokens)
	}
	if got.hasTemperature != want.hasTemperature || got.temperature != want.temperature {
		t.Errorf("%s: Temperature() = (%v, %t), want (%v, %t)", what, got.temperature, got.hasTemperature, want.temperature, want.hasTemperature)
	}
	if got.hasTopP != want.hasTopP || got.topP != want.topP {
		t.Errorf("%s: TopP() = (%v, %t), want (%v, %t)", what, got.topP, got.hasTopP, want.topP, want.hasTopP)
	}
	if got.hasStopSeq != want.hasStopSeq || !slices.Equal(got.stopSequences, want.stopSequences) {
		t.Errorf("%s: StopSequences() = (%q, %t), want (%q, %t)", what, got.stopSequences, got.hasStopSeq, want.stopSequences, want.hasStopSeq)
	}
}

// buildDerivableRequest constructs a request exercising every region AI-12.1
// must prove reachable by the rebuild path (design.md § 8.1: model, system,
// tools, tool choice, and all four generation options), so every AI-12.1 and
// AI-12.2 test derives from one well-known baseline rather than restating a
// nine-option construction call each time.
func buildDerivableRequest(t *testing.T) ai.Request {
	t.Helper()

	segment, err := ai.NewSegment("be terse")
	if err != nil {
		t.Fatalf("ai.NewSegment returned %v, want no failure", err)
	}
	system, err := ai.NewSystemInstruction(segment)
	if err != nil {
		t.Fatalf("ai.NewSystemInstruction returned %v, want no failure", err)
	}
	tools := requireToolSet(t, "search", "fetch")
	choice := requireNamedToolChoice(t, "fetch")

	request, err := ai.NewRequest(
		"m-source", []ai.Message{userTextMessage(t, "plan a trip")},
		ai.WithSystemInstruction(system),
		ai.WithTools(tools),
		ai.WithToolChoice(choice),
		ai.WithMaxOutputTokens(256),
		ai.WithTemperature(0.5),
		ai.WithTopP(0.8),
		ai.WithStopSequences("</done>"),
	)
	if err != nil {
		t.Fatalf("ai.NewRequest returned %v, want no failure", err)
	}
	return request
}

// AI-12.1 item 1 — deriving a request that changes one option leaves the
// source observably unmodified, and the derived request carries the change
// (R-REX-001, S-REX-001, S-REX-002).
//
// Two sub-cases triangulate across the two shapes an option's value takes: a
// scalar (temperature) and a slice (stop sequences). A rebuild that merely
// copied r's top-level scalar fields but aliased a slice-typed region would
// pass the first and fail the second.
func TestRequest_DeriveWithChangedOption_LeavesTheOriginalUnmodified(t *testing.T) {
	t.Parallel()

	t.Run("a_scalar_option", func(t *testing.T) {
		t.Parallel()

		source := buildDerivableRequest(t)
		before := snapshotRequest(source)

		derived, err := source.With(ai.WithTemperature(0.99))
		if err != nil {
			t.Fatalf("source.With(ai.WithTemperature(0.99)) returned %v, want no failure", err)
		}

		// S-REX-001 — the derived request reports the new value.
		if got, set := derived.Temperature(); got != 0.99 || !set {
			t.Errorf("derived.Temperature() = (%v, %t), want (0.99, true)", got, set)
		}

		// S-REX-001 — the source reports its original value, unaffected.
		if got, set := source.Temperature(); got != 0.5 || !set {
			t.Errorf("source.Temperature() = (%v, %t), want (0.5, true) — deriving must not affect the source", got, set)
		}

		// S-REX-002 — every region of the source, captured before the
		// derivation, still compares equal to its value after.
		requireSnapshotEqual(t, "source after deriving", snapshotRequest(source), before)
	})

	t.Run("a_slice_typed_option", func(t *testing.T) {
		t.Parallel()

		source := buildDerivableRequest(t)
		before := snapshotRequest(source)

		derived, err := source.With(ai.WithStopSequences("</changed>"))
		if err != nil {
			t.Fatalf("source.With(ai.WithStopSequences(...)) returned %v, want no failure", err)
		}

		if got, set := derived.StopSequences(); !set || !slices.Equal(got, []string{"</changed>"}) {
			t.Errorf("derived.StopSequences() = (%q, %t), want ([\"</changed>\"], true)", got, set)
		}
		if got, set := source.StopSequences(); !set || !slices.Equal(got, []string{"</done>"}) {
			t.Errorf("source.StopSequences() = (%q, %t), want ([\"</done>\"], true) — deriving must not affect the source", got, set)
		}

		requireSnapshotEqual(t, "source after deriving", snapshotRequest(source), before)
	})
}

// AI-12.1 item 4 (appended) — WithModel is last-wins over NewRequest's own
// model parameter, exactly as re-applying any other option is: a deliberate
// disposition, pinned rather than left to inference so the next reader does
// not close it by reflex (design.md § 4, S-REX-007).
func TestNewRequest_WithModelBesideTheParameter_LastApplicationWins(t *testing.T) {
	t.Parallel()

	request, err := ai.NewRequest("a", []ai.Message{userTextMessage(t, "hello")}, ai.WithModel("b"))
	if err != nil {
		t.Fatalf("ai.NewRequest returned %v, want no failure", err)
	}
	if got := request.Model(); got != "b" {
		t.Errorf(`request.Model() = %q, want "b" — ai.WithModel must win over the constructor's own "a" parameter`, got)
	}
}

// AI-12.1 item 4 (appended) — the model identity and the messages, the two
// regions NewRequest takes only as parameters, are reachable by the rebuild
// path like every other region (design.md § 4, S-REX-007, S-REX-008).
func TestRequest_DeriveReplacingModelOrMessages_TheDerivedRequestCarriesTheReplacement(t *testing.T) {
	t.Parallel()

	t.Run("model", func(t *testing.T) {
		t.Parallel()

		source := buildDerivableRequest(t)
		derived, err := source.With(ai.WithModel("m-derived"))
		if err != nil {
			t.Fatalf("source.With(ai.WithModel(...)) returned %v, want no failure", err)
		}
		if got := derived.Model(); got != "m-derived" {
			t.Errorf("derived.Model() = %q, want %q", got, "m-derived")
		}
		if got := source.Model(); got != "m-source" {
			t.Errorf("source.Model() = %q, want %q — deriving must not affect the source", got, "m-source")
		}
	})

	t.Run("messages_in_supplied_order", func(t *testing.T) {
		t.Parallel()

		source := buildDerivableRequest(t)
		first := userTextMessage(t, "first replacement")
		second := userTextMessage(t, "second replacement")

		derived, err := source.With(ai.WithMessages(first, second))
		if err != nil {
			t.Fatalf("source.With(ai.WithMessages(...)) returned %v, want no failure", err)
		}

		got := derived.Messages()
		if len(got) != 2 || got[0].ID() != first.ID() || got[1].ID() != second.ID() {
			t.Fatalf("derived.Messages() carries %d messages in the wrong order, want [first, second] as supplied", len(got))
		}

		sourceMessages := source.Messages()
		if len(sourceMessages) != 1 || sourceMessages[0].ID() == first.ID() {
			t.Errorf("source.Messages() changed after deriving — want the source's own single message, unaffected")
		}
	})
}

// AI-12.1 item 2 (+ item 2's markers row, task 1.7) — the rebuild path is
// total over every region landed today: each has a derive step (through
// [ai.Request.With]) and a changed observer, so a region added without a
// rebuild path fails this table rather than passing unnoticed (R-REX-002,
// S-REX-007 … S-REX-011, design.md § 8).
//
// The markers row asserts design.md § 13.2 row 7's resolved branch A:
// cache-boundary markers ride on Segment/Tool/Message and are reached
// TRANSITIVELY through WithMessages here — no marker-specific RequestOption
// exists or is added by AI-12.
//
// The table length is asserted against design.md § 8.1's documented count so
// a region silently dropped from the table — not merely a region whose
// production path breaks — also fails this test, the same idiom
// TestNewRequest_RoleVersusContentKind_EnforcesTheDocumentedTable uses for
// its twelve role/kind cells.
func TestRequest_TotalityOfTheRebuildPath_EveryRegionIsReachable(t *testing.T) {
	t.Parallel()

	type regionCase struct {
		name    string
		derive  func(t *testing.T, r ai.Request) ai.Request
		changed func(r ai.Request) bool
	}

	regions := []regionCase{
		{
			name: "model",
			derive: func(t *testing.T, r ai.Request) ai.Request {
				derived, err := r.With(ai.WithModel("m-changed"))
				if err != nil {
					t.Fatalf("With(WithModel) returned %v, want no failure", err)
				}
				return derived
			},
			changed: func(r ai.Request) bool { return r.Model() == "m-changed" },
		},
		{
			name: "messages",
			derive: func(t *testing.T, r ai.Request) ai.Request {
				derived, err := r.With(ai.WithMessages(userTextMessage(t, "changed message")))
				if err != nil {
					t.Fatalf("With(WithMessages) returned %v, want no failure", err)
				}
				return derived
			},
			changed: func(r ai.Request) bool {
				msgs := r.Messages()
				if len(msgs) != 1 {
					return false
				}
				text, ok := msgs[0].Content()[0].Text()
				return ok && text == "changed message"
			},
		},
		{
			name: "system_instruction",
			derive: func(t *testing.T, r ai.Request) ai.Request {
				segment, err := ai.NewSegment("changed instruction")
				if err != nil {
					t.Fatalf("ai.NewSegment returned %v, want no failure", err)
				}
				system, err := ai.NewSystemInstruction(segment)
				if err != nil {
					t.Fatalf("ai.NewSystemInstruction returned %v, want no failure", err)
				}
				derived, err := r.With(ai.WithSystemInstruction(system))
				if err != nil {
					t.Fatalf("With(WithSystemInstruction) returned %v, want no failure", err)
				}
				return derived
			},
			// S-REX-053 — observed through the accessor on the DERIVED
			// request: the region is stored twice on Request, and a rebuild
			// that refreshed only the copy inside options would revert the
			// top-level pair SystemInstruction() reads (design.md § 2.2.1).
			changed: func(r ai.Request) bool {
				system, ok := r.SystemInstruction()
				return ok && system.Len() == 1 && system.Segments()[0].Text() == "changed instruction"
			},
		},
		{
			name: "tools",
			derive: func(t *testing.T, r ai.Request) ai.Request {
				// "fetch" stays declared so the source's own tool choice —
				// naming "fetch" — remains valid on the derived request too;
				// only the tool set's membership is what this row proves.
				set := requireToolSet(t, "changed_tool", "fetch")
				derived, err := r.With(ai.WithTools(set))
				if err != nil {
					t.Fatalf("With(WithTools) returned %v, want no failure", err)
				}
				return derived
			},
			changed: func(r ai.Request) bool {
				tools, ok := r.Tools()
				return ok && tools.Declares("changed_tool")
			},
		},
		{
			name: "tool_choice",
			derive: func(t *testing.T, r ai.Request) ai.Request {
				// "search" is already declared by the source's tool set, so
				// changing only the choice keeps the derived request valid.
				derived, err := r.With(ai.WithToolChoice(requireNamedToolChoice(t, "search")))
				if err != nil {
					t.Fatalf("With(WithToolChoice) returned %v, want no failure", err)
				}
				return derived
			},
			changed: func(r ai.Request) bool {
				choice, ok := r.ToolChoice()
				if !ok {
					return false
				}
				name, namesOne := choice.Name()
				return namesOne && name == "search"
			},
		},
		{
			name: "max_output_tokens",
			derive: func(t *testing.T, r ai.Request) ai.Request {
				derived, err := r.With(ai.WithMaxOutputTokens(999))
				if err != nil {
					t.Fatalf("With(WithMaxOutputTokens) returned %v, want no failure", err)
				}
				return derived
			},
			changed: func(r ai.Request) bool {
				got, ok := r.MaxOutputTokens()
				return ok && got == 999
			},
		},
		{
			name: "temperature",
			derive: func(t *testing.T, r ai.Request) ai.Request {
				derived, err := r.With(ai.WithTemperature(0.11))
				if err != nil {
					t.Fatalf("With(WithTemperature) returned %v, want no failure", err)
				}
				return derived
			},
			changed: func(r ai.Request) bool {
				got, ok := r.Temperature()
				return ok && got == 0.11
			},
		},
		{
			name: "top_p",
			derive: func(t *testing.T, r ai.Request) ai.Request {
				derived, err := r.With(ai.WithTopP(0.33))
				if err != nil {
					t.Fatalf("With(WithTopP) returned %v, want no failure", err)
				}
				return derived
			},
			changed: func(r ai.Request) bool {
				got, ok := r.TopP()
				return ok && got == 0.33
			},
		},
		{
			name: "stop_sequences",
			derive: func(t *testing.T, r ai.Request) ai.Request {
				derived, err := r.With(ai.WithStopSequences("</changed>"))
				if err != nil {
					t.Fatalf("With(WithStopSequences) returned %v, want no failure", err)
				}
				return derived
			},
			changed: func(r ai.Request) bool {
				got, ok := r.StopSequences()
				return ok && slices.Equal(got, []string{"</changed>"})
			},
		},
		{
			// design.md § 13.2 row 7, branch A: markers ride on the values
			// and are reached TRANSITIVELY. No WithCacheBoundary-shaped
			// option exists; the derive step below is WithMessages, exactly
			// as the messages row above, and what differs is the observer.
			name: "cache_boundary_markers",
			derive: func(t *testing.T, r ai.Request) ai.Request {
				marked := userTextMessage(t, "marked").MarkCacheBoundary()
				derived, err := r.With(ai.WithMessages(marked))
				if err != nil {
					t.Fatalf("With(WithMessages) returned %v, want no failure", err)
				}
				return derived
			},
			changed: func(r ai.Request) bool {
				msgs := r.Messages()
				return len(msgs) == 1 && msgs[0].IsCacheBoundary()
			},
		},
	}

	if len(regions) != 10 {
		t.Fatalf("the table has %d regions, want 10 — one per region design.md § 8.1 lists as landed today "+
			"(cache-boundary markers included; provider extensions are AI-12.3's 11th and not yet part of this leaf)", len(regions))
	}

	for _, region := range regions {
		t.Run(region.name, func(t *testing.T) {
			t.Parallel()

			source := buildDerivableRequest(t)
			derived := region.derive(t, source)

			if !region.changed(derived) {
				t.Errorf("region %q: the rebuild path did not reach it — the derived request does not observe the supplied change", region.name)
			}
			if region.changed(source) {
				t.Errorf("region %q: deriving changed the SOURCE too — the rebuild path is not copy-on-write for this region", region.name)
			}
		})
	}
}

// AI-12.1 item 3 (pin) — opaque payloads survive a rebuild byte-identically:
// reasoning round-trip tokens and tool-call argument bytes (R-REX-003,
// S-REX-012, S-REX-013). Extends AI-07.3's property across the rebuild path.
// The provider-extension-value sibling (S-REX-014) is deferred to AI-12.3
// (task 3.10), because the region does not exist until then.
func TestRequest_DeriveWithUnrelatedChange_PreservesOpaquePayloadsByteIdentically(t *testing.T) {
	t.Parallel()

	t.Run("reasoning_round_trip_token", func(t *testing.T) {
		t.Parallel()

		token := []byte{0xff, 0xfe, 0x00, 0x80, 'o', 'k'} // deliberately not valid UTF-8
		reasoning, err := ai.NewReasoning("thinking about it", token)
		if err != nil {
			t.Fatalf("ai.NewReasoning returned %v, want no failure", err)
		}
		message, err := ai.NewMessage(ai.RoleAssistant, reasoning)
		if err != nil {
			t.Fatalf("ai.NewMessage returned %v, want no failure", err)
		}
		source, err := ai.NewRequest("m", []ai.Message{message})
		if err != nil {
			t.Fatalf("ai.NewRequest returned %v, want no failure", err)
		}

		derived, err := source.With(ai.WithTemperature(0.4)) // unrelated change
		if err != nil {
			t.Fatalf("source.With(...) returned %v, want no failure", err)
		}

		got, ok := derived.Messages()[0].Content()[0].Reasoning()
		if !ok {
			t.Fatalf("derived message's content[0].Reasoning() reported false, want the reasoning part")
		}
		gotToken, hasToken := got.Token()
		if !hasToken || !bytes.Equal(gotToken, token) {
			t.Errorf("derived reasoning token = (%q, %t), want (%q, true) — byte-identical to the source's", gotToken, hasToken, token)
		}
	})

	t.Run("tool_call_argument_bytes", func(t *testing.T) {
		t.Parallel()

		args := []byte(`{"b":  2,   "a":1}`) // non-canonical whitespace and key order, deliberately
		call, err := ai.NewToolCall("call-1", "search", args)
		if err != nil {
			t.Fatalf("ai.NewToolCall returned %v, want no failure", err)
		}
		message, err := ai.NewMessage(ai.RoleAssistant, call)
		if err != nil {
			t.Fatalf("ai.NewMessage returned %v, want no failure", err)
		}
		source, err := ai.NewRequest("m", []ai.Message{message})
		if err != nil {
			t.Fatalf("ai.NewRequest returned %v, want no failure", err)
		}

		derived, err := source.With(ai.WithTemperature(0.4)) // unrelated change
		if err != nil {
			t.Fatalf("source.With(...) returned %v, want no failure", err)
		}

		got, ok := derived.Messages()[0].Content()[0].ToolCall()
		if !ok {
			t.Fatalf("derived message's content[0].ToolCall() reported false, want the tool call")
		}
		if !bytes.Equal(got.Arguments(), args) {
			t.Errorf("derived tool-call arguments = %q, want %q — byte-identical to the source's", got.Arguments(), args)
		}
	})
}

// AI-12.1 item 5 (appended) — deriving with no options succeeds and equals
// the source; a failed derivation returns the zero request and leaves the
// source unmodified (S-REX-003, S-REX-004).
func TestRequest_DeriveEdgeCases_NoOptionsSucceedsAndAFailedDeriveLeavesTheSourceUnmodified(t *testing.T) {
	t.Parallel()

	t.Run("no_options_succeeds_and_equals_the_source", func(t *testing.T) {
		t.Parallel()

		source := buildDerivableRequest(t)

		derived, err := source.With()
		if err != nil {
			t.Fatalf("source.With() returned %v, want no failure", err)
		}
		if !derived.Equal(source) {
			t.Errorf("source.With() = %v, want a request equal to the source", derived)
		}
	})

	t.Run("a_failed_derivation_returns_the_zero_request_and_leaves_the_source_unmodified", func(t *testing.T) {
		t.Parallel()

		source := buildDerivableRequest(t)
		before := snapshotRequest(source)

		derived, err := source.With(ai.WithTemperature(-1)) // violates boundsRule
		if err == nil {
			t.Fatalf("source.With(ai.WithTemperature(-1)) returned no failure, want ErrOutOfRange")
		}
		requireViolation(t, err, ai.ErrOutOfRange, "temperature")

		if !derived.Equal(ai.Request{}) || derived.Model() != "" {
			t.Errorf("a failed derivation returned a non-zero request, want the zero Request so a caller that ignored the error cannot mistake it for a derived one")
		}

		requireSnapshotEqual(t, "source after a failed derivation", snapshotRequest(source), before)
	})
}

// AI-12.1 item 6 (appended) — the rebuild copies on the way in: mutating a
// slice passed to a slice-typed option after deriving leaves the derived
// request unchanged, matching NewRequest's own discipline (S-REX-005).
func TestRequest_DeriveOptionInputMutatedAfterDeriving_LeavesTheDerivedRequestUnchanged(t *testing.T) {
	t.Parallel()

	t.Run("the_message_sequence_passed_to_WithMessages", func(t *testing.T) {
		t.Parallel()

		first := userTextMessage(t, "first")
		second := userTextMessage(t, "second")
		messages := []ai.Message{first, second}

		source := buildDerivableRequest(t)
		derived, err := source.With(ai.WithMessages(messages...))
		if err != nil {
			t.Fatalf("source.With(ai.WithMessages(...)) returned %v, want no failure", err)
		}

		slices.Reverse(messages) // the caller mutates the slice it passed in

		read := derived.Messages()
		if read[0].ID() != first.ID() {
			t.Errorf("after the caller reversed the slice passed to ai.WithMessages, derived.Messages()[0] changed — " +
				"WithMessages aliased the caller's storage instead of cloning it")
		}
	})

	t.Run("the_stop_sequence_slice_passed_to_WithStopSequences", func(t *testing.T) {
		t.Parallel()

		stops := []string{"</a>", "</b>"}

		source := buildDerivableRequest(t)
		derived, err := source.With(ai.WithStopSequences(stops...))
		if err != nil {
			t.Fatalf("source.With(ai.WithStopSequences(...)) returned %v, want no failure", err)
		}

		stops[0] = "SCRATCH-MUTATED" // the caller mutates the slice it passed in

		read, _ := derived.StopSequences()
		if read[0] != "</a>" {
			t.Errorf("after the caller mutated the slice passed to ai.WithStopSequences, derived.StopSequences()[0] = %q, want %q — "+
				"WithStopSequences aliased the caller's storage instead of cloning it", read[0], "</a>")
		}
	})

	// S-REX-006, the reader-side sibling: mutating a slice a reader received
	// FROM the derived request must not change it on re-read. The same
	// mechanism NewRequest's readers already prove (TestRequest_Messages_...,
	// TestRequest_ReadbackMutation_...), restated here so the property is
	// pinned for a DERIVED request specifically and not only a constructed
	// one.
	t.Run("a_slice_a_reader_received_from_the_derived_request", func(t *testing.T) {
		t.Parallel()

		source := buildDerivableRequest(t)
		derived, err := source.With(ai.WithStopSequences("</a>", "</b>"))
		if err != nil {
			t.Fatalf("source.With(...) returned %v, want no failure", err)
		}

		read, _ := derived.StopSequences()
		read[0] = "SCRATCH-MUTATED" // the reader rewrites what it received

		reread, _ := derived.StopSequences()
		if reread[0] != "</a>" {
			t.Errorf("after a reader mutated the slice it received from derived.StopSequences(), reread[0] = %q, want %q — "+
				"the derived request handed out its own storage", reread[0], "</a>")
		}
	})
}

// --- AI-12.2 — per-request options -------------------------------------

// AI-12.2 item 1 — a per-request option override replaces the
// construction-time value, observable via readback; an option absent from
// the derivation falls through with the source's value AND presence flag; a
// newly supplied option flips absent to present on the derived request only
// (R-REX-004, S-REX-015 … S-REX-019).
func TestRequest_OverrideGenerationOption_ReplacesTheConstructionTimeValueOrFallsThrough(t *testing.T) {
	t.Parallel()

	t.Run("an_override_replaces_the_value_and_the_source_keeps_its_own", func(t *testing.T) {
		// S-REX-015.
		t.Parallel()

		source, err := ai.NewRequest("m", []ai.Message{userTextMessage(t, "go")}, ai.WithTemperature(0.2))
		if err != nil {
			t.Fatalf("ai.NewRequest returned %v, want no failure", err)
		}
		derived, err := source.With(ai.WithTemperature(0.9))
		if err != nil {
			t.Fatalf("source.With(ai.WithTemperature(0.9)) returned %v, want no failure", err)
		}
		if got, set := derived.Temperature(); got != 0.9 || !set {
			t.Errorf("derived.Temperature() = (%v, %t), want (0.9, true)", got, set)
		}
		if got, set := source.Temperature(); got != 0.2 || !set {
			t.Errorf("source.Temperature() = (%v, %t), want (0.2, true)", got, set)
		}
	})

	t.Run("an_unsupplied_option_falls_through_with_the_sources_value_and_presence_flag", func(t *testing.T) {
		// S-REX-016.
		t.Parallel()

		source, err := ai.NewRequest("m", []ai.Message{userTextMessage(t, "go")}, ai.WithMaxOutputTokens(2048))
		if err != nil {
			t.Fatalf("ai.NewRequest returned %v, want no failure", err)
		}
		derived, err := source.With(ai.WithTemperature(0.4)) // maxOutputTokens not supplied
		if err != nil {
			t.Fatalf("source.With(...) returned %v, want no failure", err)
		}
		if got, set := derived.MaxOutputTokens(); got != 2048 || !set {
			t.Errorf("derived.MaxOutputTokens() = (%d, %t), want (2048, true) — unsupplied options fall through unchanged", got, set)
		}
	})

	t.Run("absent_then_supplied_flips_absent_to_present_on_the_derived_request_only", func(t *testing.T) {
		// S-REX-017.
		t.Parallel()

		source, err := ai.NewRequest("m", []ai.Message{userTextMessage(t, "go")})
		if err != nil {
			t.Fatalf("ai.NewRequest returned %v, want no failure", err)
		}
		if _, set := source.StopSequences(); set {
			t.Fatalf("source.StopSequences() reported set before deriving, want unset")
		}

		derived, err := source.With(ai.WithStopSequences("</a>"))
		if err != nil {
			t.Fatalf("source.With(ai.WithStopSequences(...)) returned %v, want no failure", err)
		}
		if got, set := derived.StopSequences(); !set || !slices.Equal(got, []string{"</a>"}) {
			t.Errorf("derived.StopSequences() = (%q, %t), want ([\"</a>\"], true)", got, set)
		}
		if _, set := source.StopSequences(); set {
			t.Errorf("source.StopSequences() reports set after deriving, want it to remain unset")
		}
	})

	t.Run("overriding_to_the_zero_value_is_distinguishable_from_unset", func(t *testing.T) {
		// S-REX-018.
		t.Parallel()

		source, err := ai.NewRequest("m", []ai.Message{userTextMessage(t, "go")}, ai.WithTemperature(0.7))
		if err != nil {
			t.Fatalf("ai.NewRequest returned %v, want no failure", err)
		}
		derived, err := source.With(ai.WithTemperature(0))
		if err != nil {
			t.Fatalf("source.With(ai.WithTemperature(0)) returned %v, want no failure", err)
		}
		if got, set := derived.Temperature(); got != 0 || !set {
			t.Errorf("derived.Temperature() = (%v, %t), want (0, true) — set to zero, not unset", got, set)
		}
	})

	t.Run("applying_the_same_option_twice_in_one_derivation_is_last_wins", func(t *testing.T) {
		// S-REX-019.
		t.Parallel()

		source := buildDerivableRequest(t)
		derived, err := source.With(ai.WithTemperature(0.11), ai.WithTemperature(0.22))
		if err != nil {
			t.Fatalf("source.With(...) returned %v, want no failure", err)
		}
		if got, set := derived.Temperature(); got != 0.22 || !set {
			t.Errorf("derived.Temperature() = (%v, %t), want (0.22, true) — the last application wins", got, set)
		}
	})
}

// AI-12.2 item 2 — derive-time validation runs the SAME rules, in the SAME
// documented order, reporting the SAME classes at the SAME positions as
// construction: for each bounded option, the derive-path failure is
// indistinguishable from the construction failure by errors.Is class AND
// rendered position (R-REX-005, S-REX-020 … S-REX-022). Table-driven, one
// row per rule, each row's construction-side failure serving as the anchor
// its derive-side failure must match.
func TestRequest_DeriveTimeValidation_MatchesConstructionsClassAndPosition(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name     string
		option   ai.RequestOption
		wantRule error
		wantPath string
	}{
		{"an_empty_model_identity", ai.WithModel(""), ai.ErrEmpty, "model"},
		{"an_empty_message_sequence", ai.WithMessages(), ai.ErrEmpty, "messages"},
		{"zero_maximum_output_tokens", ai.WithMaxOutputTokens(0), ai.ErrOutOfRange, "maxOutputTokens"},
		{"a_negative_temperature", ai.WithTemperature(-1), ai.ErrOutOfRange, "temperature"},
		{"a_top-p_above_one", ai.WithTopP(1.5), ai.ErrOutOfRange, "topP"},
		{"an_empty_stop_sequence", ai.WithStopSequences("a", ""), ai.ErrEmpty, "stopSequences[1]"},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()

			// Anchor: what construction itself reports for this exact
			// violation.
			_, constructionErr := ai.NewRequest("m", []ai.Message{userTextMessage(t, "go")}, c.option)
			requireViolation(t, constructionErr, c.wantRule, c.wantPath)

			// The derive path, from an otherwise fully valid source, must be
			// indistinguishable.
			source := buildDerivableRequest(t)
			_, deriveErr := source.With(c.option)
			requireViolation(t, deriveErr, c.wantRule, c.wantPath)
		})
	}
}

// AI-12.2 item 3 (appended), first clause — a cross-region rule the source
// satisfied and the derivation breaks fails at derive time: replacing the
// tool set so the source's own tool choice no longer resolves
// (R-REX-005, S-REX-023).
func TestRequestWith_ToolSetReplacedSoTheChoiceNoLongerResolves_FailsAtDeriveTime(t *testing.T) {
	t.Parallel()

	source := buildDerivableRequest(t) // tools {search, fetch}, choice names "fetch"

	replacement := requireToolSet(t, "search") // no longer declares "fetch"
	_, err := source.With(ai.WithTools(replacement))
	requireViolation(t, err, ai.ErrUnresolvedReference, "toolChoice.name")
}

// AI-12.2 item 4 (appended) — derive-time first-failure is deterministic and
// follows the SAME documented order as construction: a derivation violating
// two rules at once, run many times, reports the identical class at the
// identical position every time, and it is the one construction reports
// first for the same regions. Equivalence is asserted BETWEEN THE TWO DOORS,
// never against an absolute ordinal (R-REX-005, S-REX-025). Uses two rules
// whose landed relative order is known: duplicate tool-call identities
// (rule 7) precedes orphan tool results (rule 8) — design.md § 3.
func TestRequestWith_TwoRulesViolatedAtOnce_ReportsTheDocumentedOrderFirstAcrossManyRuns(t *testing.T) {
	t.Parallel()

	const runs = 100

	// Violates rule 7 (a duplicate call identity "dup") and rule 8 (an
	// orphan tool result naming a call that does not exist) at once.
	violating := []ai.Message{
		requireMessage(t, ai.RoleAssistant, toolCallPart(t, "dup", "search")),
		requireMessage(t, ai.RoleAssistant, toolCallPart(t, "dup", "fetch")),
		requireMessage(t, ai.RoleTool, toolResultPart(t, "no-such-call", "42")),
	}

	// Anchor: construction reports rule 7 first.
	_, constructionErr := ai.NewRequest("m", violating)
	requireViolation(t, constructionErr, ai.ErrDuplicate, "messages[1].content[0]")

	source := buildDerivableRequest(t)
	for range runs {
		_, err := source.With(ai.WithMessages(violating...))
		requireViolation(t, err, ai.ErrDuplicate, "messages[1].content[0]")
	}
}

// AI-12.2 item 5 (appended by Phase 0, task 0.3's row 8 — branch A) — the
// breakpoint cap re-runs on the derive path for free: a derivation that
// pushes the total marker count past ai.MaxCacheBoundaries fails with
// ErrOutOfRange at the position construction reports. No AI-12 production
// code: the cap rule (AI-11's) is already row 10 of requestDraft.rules().
func TestRequestWith_DerivationPastTheCacheBoundaryCap_FailsAtDeriveTime(t *testing.T) {
	t.Parallel()

	// MaxCacheBoundaries + 1 marked tools, "fetch" among them so the
	// source's own tool choice stays resolvable and only the cap fails.
	tools := []ai.Tool{requireTool(t, "fetch").MarkCacheBoundary()}
	for i := range ai.MaxCacheBoundaries {
		tools = append(tools, requireTool(t, fmt.Sprintf("extra_%d", i)).MarkCacheBoundary())
	}
	overCap, err := ai.NewToolSet(tools...)
	if err != nil {
		t.Fatalf("ai.NewToolSet returned %v, want no failure", err)
	}

	// Anchor: construction reports the cap failure.
	_, constructionErr := ai.NewRequest("m", []ai.Message{userTextMessage(t, "go")}, ai.WithTools(overCap))
	requireViolation(t, constructionErr, ai.ErrOutOfRange, "cacheBoundaries")

	source := buildDerivableRequest(t) // valid, well under the cap
	_, err = source.With(ai.WithTools(overCap))
	requireViolation(t, err, ai.ErrOutOfRange, "cacheBoundaries")
}
