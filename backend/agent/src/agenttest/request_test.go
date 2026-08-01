// AI-10.5 — the whole-request round trip, proven from another package.
//
// Lives here rather than in ai_test, for design.md § 13's reason: the
// milestone's acceptance is readability from another package, and agenttest
// is the package that already proves that for AI-06 (content_part_test.go).
// This is the property AI-26's translator will consume, proven before AI-26
// exists.
package agenttest_test

import (
	"bytes"
	"slices"
	"testing"

	"github.com/cachicamas/backend/agent/src/ai"
)

// partKindReaders maps each registered content-part kind to a function that
// reads a part of that kind through the exported surface alone and rebuilds
// an equivalent part from what it read.
//
// This is the single source both AI-10.5 items share: item 1's round trip
// calls it to rebuild content, and item 2 asserts it holds one entry per
// member of ai.PartKinds() — so a kind added without an entry here fails
// immediately, rather than silently falling out of the round trip it is
// supposed to prove.
var partKindReaders = map[ai.PartKind]func(t *testing.T, part ai.Part) ai.Part{
	ai.PartKindText: func(t *testing.T, part ai.Part) ai.Part {
		t.Helper()

		text, ok := part.Text()
		if !ok {
			t.Fatalf("part.Text() reported false on a part of kind %v", part.Kind())
		}
		rebuilt, err := ai.NewText(text)
		if err != nil {
			t.Fatalf("ai.NewText returned %v rebuilding a text part, want no failure", err)
		}
		return rebuilt
	},
	ai.PartKindReasoning: func(t *testing.T, part ai.Part) ai.Part {
		t.Helper()

		reasoning, ok := part.Reasoning()
		if !ok {
			t.Fatalf("part.Reasoning() reported false on a part of kind %v", part.Kind())
		}
		token, hasToken := reasoning.Token()
		if !hasToken {
			token = nil // NewReasoning treats nil, not an empty slice, as "no token".
		}

		var (
			rebuilt ai.Part
			err     error
		)
		if reasoning.State() == ai.ReasoningStateRedacted {
			rebuilt, err = ai.NewRedactedReasoning(token)
		} else {
			rebuilt, err = ai.NewReasoning(reasoning.Text(), token)
		}
		if err != nil {
			t.Fatalf("rebuilding reasoning returned %v, want no failure", err)
		}
		return rebuilt
	},
	ai.PartKindToolCall: func(t *testing.T, part ai.Part) ai.Part {
		t.Helper()

		call, ok := part.ToolCall()
		if !ok {
			t.Fatalf("part.ToolCall() reported false on a part of kind %v", part.Kind())
		}
		rebuilt, err := ai.NewToolCall(call.ID(), call.Name(), call.Arguments())
		if err != nil {
			t.Fatalf("ai.NewToolCall returned %v rebuilding a tool call, want no failure", err)
		}
		return rebuilt
	},
	ai.PartKindToolResult: func(t *testing.T, part ai.Part) ai.Part {
		t.Helper()

		result, ok := part.ToolResult()
		if !ok {
			t.Fatalf("part.ToolResult() reported false on a part of kind %v", part.Kind())
		}
		var (
			rebuilt ai.Part
			err     error
		)
		if result.Failed() {
			rebuilt, err = ai.NewToolFailure(result.CallID(), result.Content())
		} else {
			rebuilt, err = ai.NewToolResult(result.CallID(), result.Content())
		}
		if err != nil {
			t.Fatalf("rebuilding a tool result returned %v, want no failure", err)
		}
		return rebuilt
	},
}

// AI-10.5 item 2 (pin) — the walk's kind handling is exhaustive over the
// registered kinds (R-AMR-015, S-AMR-056).
//
// Driven from ai.PartKinds() rather than a literal count of four: a kind
// added to the vocabulary without a matching entry in partKindReaders fails
// this test immediately. This consumes AI-06.4's registration rather than
// duplicating it, and it is the mechanism that keeps V-REQ-06 true for a kind
// nobody has written the round-trip walk for yet.
func TestPartKindReaders_EveryRegisteredKind_HasAReader(t *testing.T) {
	t.Parallel()

	for _, kind := range ai.PartKinds() {
		if _, ok := partKindReaders[kind]; !ok {
			t.Errorf("ai.PartKind %v is registered in ai.PartKinds() but this round-trip walk's "+
				"partKindReaders has no reader for it — a kind added without a readable accessor "+
				"here fails this pin (R-AMR-015, S-AMR-056)", kind)
		}
	}
}

// AI-10.5 item 1 — a whole request round-trips, exhaustively over the
// registered kinds (R-AMR-015, S-AMR-055, design.md § 10).
//
// The walk builds a request holding every registered content-part kind —
// text, reasoning with its round-trip token, tool call, tool result — plus
// system segments, tools, a tool choice and every generation option; reads
// every region back through the exported surface alone; rebuilds a request
// from what it read; and asserts the rebuild is equal to the original under
// region-wise readback equality (design.md § 11.2, landed by AI-10.6).
//
// Message identity is excluded from the comparison on purpose: V-REQ-03 makes
// two messages built from identical inputs deliberately distinguishable, so
// comparing MessageID would make this property false by construction for any
// two requests at all — design.md § 11.2 states this explicitly.
func TestRequest_WholeRequestRoundTrip_ReconstructsAnEqualRequest(t *testing.T) {
	t.Parallel()

	original := buildFullRequest(t)
	rebuilt := rebuildFromReadback(t, original)

	requireRequestsEqual(t, original, rebuilt)
}

// buildFullRequest constructs a request holding every registered content-part
// kind plus every optional region, entirely through the exported surface.
func buildFullRequest(t *testing.T) ai.Request {
	t.Helper()

	text, err := ai.NewText("Plan a three-day trip to Paris.")
	if err != nil {
		t.Fatalf("ai.NewText returned %v, want no failure", err)
	}
	reasoning, err := ai.NewReasoning("The user wants a short itinerary; a flight search comes first.", []byte("opaque-round-trip-token"))
	if err != nil {
		t.Fatalf("ai.NewReasoning returned %v, want no failure", err)
	}
	toolCall, err := ai.NewToolCall("call-1", "search_flights", []byte(`{"from":"NYC","to":"CDG"}`))
	if err != nil {
		t.Fatalf("ai.NewToolCall returned %v, want no failure", err)
	}
	toolResult, err := ai.NewToolResult("call-1", `[{"flight":"AF23","price":612}]`)
	if err != nil {
		t.Fatalf("ai.NewToolResult returned %v, want no failure", err)
	}

	userMessage, err := ai.NewMessage(ai.RoleUser, text)
	if err != nil {
		t.Fatalf("ai.NewMessage(RoleUser) returned %v, want no failure", err)
	}
	assistantMessage, err := ai.NewMessage(ai.RoleAssistant, reasoning, toolCall)
	if err != nil {
		t.Fatalf("ai.NewMessage(RoleAssistant) returned %v, want no failure", err)
	}
	toolMessage, err := ai.NewMessage(ai.RoleTool, toolResult)
	if err != nil {
		t.Fatalf("ai.NewMessage(RoleTool) returned %v, want no failure", err)
	}
	toolMessage = toolMessage.MarkCacheBoundary() // AI-11.1 — see the comment on segmentTwo above.

	segmentOne, err := ai.NewSegment("You are a travel planning assistant.")
	if err != nil {
		t.Fatalf("ai.NewSegment returned %v, want no failure", err)
	}
	segmentTwo, err := ai.NewSegment("Always cite prices in USD.")
	if err != nil {
		t.Fatalf("ai.NewSegment returned %v, want no failure", err)
	}
	// AI-11.1 — one marked carrier per region exercises the round trip's
	// marker leg (R-ACB-004): a marker that survived every other assertion
	// here but vanished on rebuild would be the ten-times-the-bill defect
	// design.md warns about, with no error and no wrong answer.
	segmentTwo = segmentTwo.MarkCacheBoundary()
	system, err := ai.NewSystemInstruction(segmentOne, segmentTwo)
	if err != nil {
		t.Fatalf("ai.NewSystemInstruction returned %v, want no failure", err)
	}

	searchTool, err := ai.NewTool("search_flights", "search for flights between two airports", []byte(`{"type":"object"}`))
	if err != nil {
		t.Fatalf("ai.NewTool returned %v, want no failure", err)
	}
	bookTool, err := ai.NewTool("book_flight", "book a specific flight", []byte(`{"type":"object"}`))
	if err != nil {
		t.Fatalf("ai.NewTool returned %v, want no failure", err)
	}
	bookTool = bookTool.MarkCacheBoundary() // AI-11.1 — see the comment on segmentTwo above.
	tools, err := ai.NewToolSet(searchTool, bookTool)
	if err != nil {
		t.Fatalf("ai.NewToolSet returned %v, want no failure", err)
	}

	choice, err := ai.NewNamedToolChoice("search_flights")
	if err != nil {
		t.Fatalf("ai.NewNamedToolChoice returned %v, want no failure", err)
	}

	request, err := ai.NewRequest(
		"cachicamas-neutral-model-1",
		[]ai.Message{userMessage, assistantMessage, toolMessage},
		ai.WithSystemInstruction(system),
		ai.WithTools(tools),
		ai.WithToolChoice(choice),
		ai.WithMaxOutputTokens(2048),
		ai.WithTemperature(0.7),
		ai.WithTopP(0.9),
		ai.WithStopSequences("</plan>", "\n\nUser:"),
	)
	if err != nil {
		t.Fatalf("ai.NewRequest returned %v, want no failure", err)
	}
	return request
}

// rebuildFromReadback reads every region of request through the exported
// surface alone and constructs a new request from what it read.
//
// No field of request is ever touched directly: every value that reaches
// ai.NewRequest below was obtained by calling an accessor. This is the exact
// loop AI-26's translator will write.
func rebuildFromReadback(t *testing.T, request ai.Request) ai.Request {
	t.Helper()

	messages := make([]ai.Message, 0, len(request.Messages()))
	for _, message := range request.Messages() {
		content := make([]ai.Part, 0, len(message.Content()))
		for _, part := range message.Content() {
			reader, ok := partKindReaders[part.Kind()]
			if !ok {
				t.Fatalf("no reader registered for part kind %v — see TestPartKindReaders_EveryRegisteredKind_HasAReader", part.Kind())
			}
			content = append(content, reader(t, part))
		}
		rebuiltMessage, err := ai.NewMessage(message.Role(), content...)
		if err != nil {
			t.Fatalf("ai.NewMessage returned %v rebuilding a message, want no failure", err)
		}
		// AI-11.1 — carry the message's cache-boundary marker across the
		// rebuild (R-ACB-004). Message.Role and Message.Content say nothing
		// about it, so a walk that read only those two would silently drop
		// every marker on a message.
		if message.IsCacheBoundary() {
			rebuiltMessage = rebuiltMessage.MarkCacheBoundary()
		}
		messages = append(messages, rebuiltMessage)
	}

	var opts []ai.RequestOption

	if system, hasSystem := request.SystemInstruction(); hasSystem {
		segments := make([]ai.Segment, 0, system.Len())
		for _, segment := range system.Segments() {
			rebuiltSegment, err := ai.NewSegment(segment.Text())
			if err != nil {
				t.Fatalf("ai.NewSegment returned %v rebuilding a segment, want no failure", err)
			}
			// AI-11.1 — carry the segment's cache-boundary marker across the
			// rebuild (R-ACB-004). Segment stays comparable, so
			// requireRequestsEqual's slices.Equal catches a dropped marker
			// "for free"; carrying it across here is what makes the round
			// trip actually preserve it rather than merely be caught losing it.
			if segment.IsCacheBoundary() {
				rebuiltSegment = rebuiltSegment.MarkCacheBoundary()
			}
			segments = append(segments, rebuiltSegment)
		}
		rebuiltSystem, err := ai.NewSystemInstruction(segments...)
		if err != nil {
			t.Fatalf("ai.NewSystemInstruction returned %v rebuilding the system instruction, want no failure", err)
		}
		opts = append(opts, ai.WithSystemInstruction(rebuiltSystem))
	}

	if tools, hasTools := request.Tools(); hasTools {
		declared := tools.Tools()
		rebuiltTools := make([]ai.Tool, 0, len(declared))
		for _, tool := range declared {
			rebuiltTool, err := ai.NewTool(tool.Name(), tool.Description(), tool.Schema())
			if err != nil {
				t.Fatalf("ai.NewTool returned %v rebuilding a tool, want no failure", err)
			}
			// AI-11.1 — carry the tool's cache-boundary marker across the
			// rebuild (R-ACB-004). Name, Description and Schema say nothing
			// about it, so a walk that read only those three would silently
			// drop every marker on a declaration.
			if tool.IsCacheBoundary() {
				rebuiltTool = rebuiltTool.MarkCacheBoundary()
			}
			rebuiltTools = append(rebuiltTools, rebuiltTool)
		}
		rebuiltSet, err := ai.NewToolSet(rebuiltTools...)
		if err != nil {
			t.Fatalf("ai.NewToolSet returned %v rebuilding the tool set, want no failure", err)
		}
		opts = append(opts, ai.WithTools(rebuiltSet))
	}

	if choice, hasChoice := request.ToolChoice(); hasChoice {
		var (
			rebuiltChoice ai.ToolChoice
			err           error
		)
		if name, namesOne := choice.Name(); namesOne {
			rebuiltChoice, err = ai.NewNamedToolChoice(name)
		} else {
			rebuiltChoice, err = ai.NewToolChoice(choice.Mode())
		}
		if err != nil {
			t.Fatalf("rebuilding the tool choice returned %v, want no failure", err)
		}
		opts = append(opts, ai.WithToolChoice(rebuiltChoice))
	}

	if tokens, ok := request.MaxOutputTokens(); ok {
		opts = append(opts, ai.WithMaxOutputTokens(tokens))
	}
	if temperature, ok := request.Temperature(); ok {
		opts = append(opts, ai.WithTemperature(temperature))
	}
	if topP, ok := request.TopP(); ok {
		opts = append(opts, ai.WithTopP(topP))
	}
	if stopSequences, ok := request.StopSequences(); ok {
		opts = append(opts, ai.WithStopSequences(stopSequences...))
	}

	rebuilt, err := ai.NewRequest(request.Model(), messages, opts...)
	if err != nil {
		t.Fatalf("ai.NewRequest returned %v rebuilding the request, want no failure", err)
	}
	return rebuilt
}

// requireRequestsEqual asserts a and b are equal under region-wise readback
// equality (design.md § 11.2): every accessor's result compared, message
// identity excluded.
func requireRequestsEqual(t *testing.T, a, b ai.Request) {
	t.Helper()

	if a.Model() != b.Model() {
		t.Errorf("Model() = %q, want %q", b.Model(), a.Model())
	}

	requireMessagesEqual(t, a.Messages(), b.Messages())

	aSystem, aHasSystem := a.SystemInstruction()
	bSystem, bHasSystem := b.SystemInstruction()
	if aHasSystem != bHasSystem {
		t.Fatalf("SystemInstruction() presence = %t, want %t", bHasSystem, aHasSystem)
	}
	if aHasSystem && !slices.Equal(aSystem.Segments(), bSystem.Segments()) {
		t.Errorf("system instruction segments are not equal")
	}

	aTools, aHasTools := a.Tools()
	bTools, bHasTools := b.Tools()
	if aHasTools != bHasTools {
		t.Fatalf("Tools() presence = %t, want %t", bHasTools, aHasTools)
	}
	if aHasTools {
		requireToolsEqual(t, aTools.Tools(), bTools.Tools())
	}

	aChoice, aHasChoice := a.ToolChoice()
	bChoice, bHasChoice := b.ToolChoice()
	if aHasChoice != bHasChoice {
		t.Fatalf("ToolChoice() presence = %t, want %t", bHasChoice, aHasChoice)
	}
	if aHasChoice {
		if aChoice.Mode() != bChoice.Mode() {
			t.Errorf("ToolChoice().Mode() = %v, want %v", bChoice.Mode(), aChoice.Mode())
		}
		aName, aNamesOne := aChoice.Name()
		bName, bNamesOne := bChoice.Name()
		if aNamesOne != bNamesOne || aName != bName {
			t.Errorf("ToolChoice().Name() = (%q, %t), want (%q, %t)", bName, bNamesOne, aName, aNamesOne)
		}
	}

	aMax, aHasMax := a.MaxOutputTokens()
	bMax, bHasMax := b.MaxOutputTokens()
	if aHasMax != bHasMax || aMax != bMax {
		t.Errorf("MaxOutputTokens() = (%d, %t), want (%d, %t)", bMax, bHasMax, aMax, aHasMax)
	}

	aTemp, aHasTemp := a.Temperature()
	bTemp, bHasTemp := b.Temperature()
	if aHasTemp != bHasTemp || aTemp != bTemp {
		t.Errorf("Temperature() = (%v, %t), want (%v, %t)", bTemp, bHasTemp, aTemp, aHasTemp)
	}

	aTopP, aHasTopP := a.TopP()
	bTopP, bHasTopP := b.TopP()
	if aHasTopP != bHasTopP || aTopP != bTopP {
		t.Errorf("TopP() = (%v, %t), want (%v, %t)", bTopP, bHasTopP, aTopP, aHasTopP)
	}

	aStop, aHasStop := a.StopSequences()
	bStop, bHasStop := b.StopSequences()
	if aHasStop != bHasStop || !slices.Equal(aStop, bStop) {
		t.Errorf("StopSequences() = (%q, %t), want (%q, %t)", bStop, bHasStop, aStop, aHasStop)
	}
}

// requireMessagesEqual compares two message sequences role by role and
// content part by content part. Message identity is excluded — see this
// file's package comment on TestRequest_WholeRequestRoundTrip... for why.
func requireMessagesEqual(t *testing.T, a, b []ai.Message) {
	t.Helper()

	if len(a) != len(b) {
		t.Fatalf("len(Messages()) = %d, want %d", len(b), len(a))
	}
	for i := range a {
		if a[i].Role() != b[i].Role() {
			t.Errorf("messages[%d].Role() = %v, want %v", i, b[i].Role(), a[i].Role())
		}
		// AI-11.1 — a message's cache-boundary marker (R-ACB-004) is not part
		// of Role() or Content(); without this check a rebuild that dropped
		// the marker would satisfy every other assertion in this file.
		if a[i].IsCacheBoundary() != b[i].IsCacheBoundary() {
			t.Errorf("messages[%d].IsCacheBoundary() = %t, want %t", i, b[i].IsCacheBoundary(), a[i].IsCacheBoundary())
		}
		requirePartsEqual(t, i, a[i].Content(), b[i].Content())
	}
}

// requirePartsEqual compares two content sequences kind by kind and payload
// by payload, dispatching on each part's own kind.
func requirePartsEqual(t *testing.T, messageIndex int, a, b []ai.Part) {
	t.Helper()

	if len(a) != len(b) {
		t.Fatalf("len(messages[%d].Content()) = %d, want %d", messageIndex, len(b), len(a))
	}
	for j := range a {
		if a[j].Kind() != b[j].Kind() {
			t.Errorf("messages[%d].content[%d].Kind() = %v, want %v", messageIndex, j, b[j].Kind(), a[j].Kind())
			continue
		}
		switch a[j].Kind() {
		case ai.PartKindText:
			aText, _ := a[j].Text()
			bText, _ := b[j].Text()
			if aText != bText {
				t.Errorf("messages[%d].content[%d].Text() = %q, want %q", messageIndex, j, bText, aText)
			}
		case ai.PartKindReasoning:
			aReasoning, _ := a[j].Reasoning()
			bReasoning, _ := b[j].Reasoning()
			if aReasoning.State() != bReasoning.State() {
				t.Errorf("messages[%d].content[%d].Reasoning().State() = %v, want %v", messageIndex, j, bReasoning.State(), aReasoning.State())
			}
			if aReasoning.Text() != bReasoning.Text() {
				t.Errorf("messages[%d].content[%d].Reasoning().Text() = %q, want %q", messageIndex, j, bReasoning.Text(), aReasoning.Text())
			}
			aToken, aHasToken := aReasoning.Token()
			bToken, bHasToken := bReasoning.Token()
			if aHasToken != bHasToken || !bytes.Equal(aToken, bToken) {
				t.Errorf("messages[%d].content[%d].Reasoning().Token() = (%q, %t), want (%q, %t)", messageIndex, j, bToken, bHasToken, aToken, aHasToken)
			}
		case ai.PartKindToolCall:
			aCall, _ := a[j].ToolCall()
			bCall, _ := b[j].ToolCall()
			if aCall.ID() != bCall.ID() || aCall.Name() != bCall.Name() {
				t.Errorf("messages[%d].content[%d].ToolCall() = (%q, %q), want (%q, %q)", messageIndex, j, bCall.ID(), bCall.Name(), aCall.ID(), aCall.Name())
			}
			if !bytes.Equal(aCall.Arguments(), bCall.Arguments()) {
				t.Errorf("messages[%d].content[%d].ToolCall().Arguments() = %q, want %q", messageIndex, j, bCall.Arguments(), aCall.Arguments())
			}
		case ai.PartKindToolResult:
			aResult, _ := a[j].ToolResult()
			bResult, _ := b[j].ToolResult()
			if aResult.CallID() != bResult.CallID() || aResult.Content() != bResult.Content() || aResult.Failed() != bResult.Failed() {
				t.Errorf("messages[%d].content[%d].ToolResult() = (%q, %q, %t), want (%q, %q, %t)",
					messageIndex, j, bResult.CallID(), bResult.Content(), bResult.Failed(),
					aResult.CallID(), aResult.Content(), aResult.Failed())
			}
		default:
			t.Fatalf("requirePartsEqual: unhandled kind %v — see TestPartKindReaders_EveryRegisteredKind_HasAReader", a[j].Kind())
		}
	}
}

// requireToolsEqual compares two tool-declaration sequences in order.
func requireToolsEqual(t *testing.T, a, b []ai.Tool) {
	t.Helper()

	if len(a) != len(b) {
		t.Fatalf("len(Tools()) = %d, want %d", len(b), len(a))
	}
	for i := range a {
		if a[i].Name() != b[i].Name() || a[i].Description() != b[i].Description() {
			t.Errorf("tools[%d] = (%q, %q), want (%q, %q)", i, b[i].Name(), b[i].Description(), a[i].Name(), a[i].Description())
		}
		if !bytes.Equal(a[i].Schema(), b[i].Schema()) {
			t.Errorf("tools[%d].Schema() = %q, want %q", i, b[i].Schema(), a[i].Schema())
		}
		// AI-11.1 — a tool declaration's cache-boundary marker (R-ACB-004) is
		// not part of Name(), Description() or Schema(); without this check a
		// rebuild that dropped the marker would satisfy every other
		// assertion in this file.
		if a[i].IsCacheBoundary() != b[i].IsCacheBoundary() {
			t.Errorf("tools[%d].IsCacheBoundary() = %t, want %t", i, b[i].IsCacheBoundary(), a[i].IsCacheBoundary())
		}
	}
}
