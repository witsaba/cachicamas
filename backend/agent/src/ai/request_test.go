// Tests for AI-10.1 — the walking skeleton of the normalized request.
//
// External package, for AI-06's reason: the consumer this contract exists for
// is an adapter in a vendor package, and doc 0002 makes readability from
// outside constitutive of the request rather than incidental to it. Defect C2
// was a Layer 1 whose request could not be read from another package at all.

package ai_test

import (
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
