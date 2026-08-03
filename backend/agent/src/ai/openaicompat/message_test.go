// AI-26.3 — every readable content-part variant translates, message and
// intra-message part order are preserved exactly, and consecutive
// same-role messages are never merged (R-ART-007, R-ART-008, R-ART-009).
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
			name: "message with two ordered text parts renders as a content array",
			build: func() ai.Request {
				return mustRequest(ai.NewRequest("gpt-4o", []ai.Message{
					mustMessage(ai.NewMessage(ai.RoleUser,
						mustPart(ai.NewText("First, check the weather.")),
						mustPart(ai.NewText("Second, book the flight.")),
					)),
				}))
			},
			want: `{"model":"gpt-4o","messages":[` +
				`{"role":"user","content":[{"type":"text","text":"First, check the weather."},{"type":"text","text":"Second, book the flight."}]}` +
				`],"stream":true,"stream_options":{"include_usage":true}}`,
		},
		expectationCase{
			name: "message with three ordered text parts preserves that exact order",
			build: func() ai.Request {
				return mustRequest(ai.NewRequest("gpt-4o", []ai.Message{
					mustMessage(ai.NewMessage(ai.RoleAssistant,
						mustPart(ai.NewText("alpha")),
						mustPart(ai.NewText("beta")),
						mustPart(ai.NewText("gamma")),
					)),
				}))
			},
			want: `{"model":"gpt-4o","messages":[` +
				`{"role":"assistant","content":[{"type":"text","text":"alpha"},{"type":"text","text":"beta"},{"type":"text","text":"gamma"}]}` +
				`],"stream":true,"stream_options":{"include_usage":true}}`,
		},
		expectationCase{
			name: "multi-turn conversation preserves a distinctive message order",
			build: func() ai.Request {
				return mustRequest(ai.NewRequest("gpt-4o", []ai.Message{
					mustMessage(ai.NewMessage(ai.RoleUser, mustPart(ai.NewText("Plan a weekend trip to Lisbon.")))),
					mustMessage(ai.NewMessage(ai.RoleAssistant, mustPart(ai.NewText("Here is a two-day itinerary.")))),
					mustMessage(ai.NewMessage(ai.RoleUser, mustPart(ai.NewText("Add a fado show on the first night.")))),
				}))
			},
			want: `{"model":"gpt-4o","messages":[` +
				`{"role":"user","content":"Plan a weekend trip to Lisbon."},` +
				`{"role":"assistant","content":"Here is a two-day itinerary."},` +
				`{"role":"user","content":"Add a fado show on the first night."}` +
				`],"stream":true,"stream_options":{"include_usage":true}}`,
		},
		expectationCase{
			name: "two consecutive same-role messages render as two distinct wire objects",
			build: func() ai.Request {
				return mustRequest(ai.NewRequest("gpt-4o", []ai.Message{
					mustMessage(ai.NewMessage(ai.RoleUser, mustPart(ai.NewText("What's the weather in Paris?")))),
					mustMessage(ai.NewMessage(ai.RoleUser, mustPart(ai.NewText("Also check London.")))),
				}))
			},
			want: `{"model":"gpt-4o","messages":[` +
				`{"role":"user","content":"What's the weather in Paris?"},` +
				`{"role":"user","content":"Also check London."}` +
				`],"stream":true,"stream_options":{"include_usage":true}}`,
		},
		expectationCase{
			name: "three consecutive same-role messages render as three distinct wire objects, no concatenation",
			build: func() ai.Request {
				return mustRequest(ai.NewRequest("gpt-4o", []ai.Message{
					mustMessage(ai.NewMessage(ai.RoleUser, mustPart(ai.NewText("Search three flights for me.")))),
					mustMessage(ai.NewMessage(ai.RoleAssistant, mustPart(ai.NewText("Here is the first option.")))),
					mustMessage(ai.NewMessage(ai.RoleAssistant, mustPart(ai.NewText("Here is the second option.")))),
					mustMessage(ai.NewMessage(ai.RoleAssistant, mustPart(ai.NewText("Here is the third option.")))),
				}))
			},
			want: `{"model":"gpt-4o","messages":[` +
				`{"role":"user","content":"Search three flights for me."},` +
				`{"role":"assistant","content":"Here is the first option."},` +
				`{"role":"assistant","content":"Here is the second option."},` +
				`{"role":"assistant","content":"Here is the third option."}` +
				`],"stream":true,"stream_options":{"include_usage":true}}`,
		},
	)
}

// TestMessage_AdjacentSwap_TranslatesDifferently proves S-ART-030: two
// requests differing only by a swap of two adjacent messages translate to
// different bodies. Both sides are anchored against a hard-coded literal
// — not merely compared to each other — so the test cannot pass either
// because message-order preservation is not yet implemented (it already
// was, incidentally, before this slice: body.go's slice-1 loop never
// reordered anything) or because both translations could be wrong the
// same way relative to truth while still differing from each other for an
// unrelated reason. See doc.go and this session's own evidence log for the
// disclosure that this scenario is green on first write, with falsifiability
// proven separately by a staged, reverted mutation.
func TestMessage_AdjacentSwap_TranslatesDifferently(t *testing.T) {
	first := mustMessage(ai.NewMessage(ai.RoleUser, mustPart(ai.NewText("Plan a weekend trip to Lisbon."))))
	second := mustMessage(ai.NewMessage(ai.RoleAssistant, mustPart(ai.NewText("Here is a two-day itinerary."))))
	third := mustMessage(ai.NewMessage(ai.RoleUser, mustPart(ai.NewText("Add a fado show on the first night."))))

	forward := mustRequest(ai.NewRequest("gpt-4o", []ai.Message{first, second, third}))
	// Adjacent swap: second and third exchange positions.
	swapped := mustRequest(ai.NewRequest("gpt-4o", []ai.Message{first, third, second}))

	forwardBytes, err := openaicompat.Translate(forward)
	if err != nil {
		t.Fatalf("Translate(forward): unexpected error: %v", err)
	}
	swappedBytes, err := openaicompat.Translate(swapped)
	if err != nil {
		t.Fatalf("Translate(swapped): unexpected error: %v", err)
	}

	wantForward := `{"model":"gpt-4o","messages":[` +
		`{"role":"user","content":"Plan a weekend trip to Lisbon."},` +
		`{"role":"assistant","content":"Here is a two-day itinerary."},` +
		`{"role":"user","content":"Add a fado show on the first night."}` +
		`],"stream":true,"stream_options":{"include_usage":true}}`
	wantSwapped := `{"model":"gpt-4o","messages":[` +
		`{"role":"user","content":"Plan a weekend trip to Lisbon."},` +
		`{"role":"user","content":"Add a fado show on the first night."},` +
		`{"role":"assistant","content":"Here is a two-day itinerary."}` +
		`],"stream":true,"stream_options":{"include_usage":true}}`

	if string(forwardBytes) != wantForward {
		t.Fatalf("Translate(forward) =\n%s\nwant\n%s", forwardBytes, wantForward)
	}
	if string(swappedBytes) != wantSwapped {
		t.Fatalf("Translate(swapped) =\n%s\nwant\n%s", swappedBytes, wantSwapped)
	}
	if string(forwardBytes) == string(swappedBytes) {
		t.Fatalf("Translate(forward) == Translate(swapped) = %s, want different bytes — order must be carried, not incidentally equal", forwardBytes)
	}
}

// TestMessage_PartOrderSwap_TranslatesDifferently proves S-ART-029
// non-vacuously, closing a gap the registered "three ordered text parts"
// case (init, above) leaves open: "alpha", "beta", "gamma" is already in
// ascending lexical order, so a hypothetical implementation bug that
// sorted a message's parts instead of preserving caller order would
// render that specific case's array in exactly the order its hard-coded
// expectation names, passing vacuously. This test's two part values are
// chosen so forward order ("zulu report", "alpha report") is descending —
// its own reverse happens to BE ascending — so a sort-by-value bug would
// make Translate(forward) render as [alpha, zulu], failing wantForward
// outright, rather than coincidentally matching it. Both sides are
// anchored against a hard-coded literal, and asserted to differ, mirroring
// TestMessage_AdjacentSwap_TranslatesDifferently's own shape one level
// down: message order there, intra-message part order here.
func TestMessage_PartOrderSwap_TranslatesDifferently(t *testing.T) {
	first := mustPart(ai.NewText("zulu report"))
	second := mustPart(ai.NewText("alpha report"))

	forward := mustRequest(ai.NewRequest("gpt-4o", []ai.Message{
		mustMessage(ai.NewMessage(ai.RoleUser, first, second)),
	}))
	reversed := mustRequest(ai.NewRequest("gpt-4o", []ai.Message{
		mustMessage(ai.NewMessage(ai.RoleUser, second, first)),
	}))

	forwardBytes, err := openaicompat.Translate(forward)
	if err != nil {
		t.Fatalf("Translate(forward): unexpected error: %v", err)
	}
	reversedBytes, err := openaicompat.Translate(reversed)
	if err != nil {
		t.Fatalf("Translate(reversed): unexpected error: %v", err)
	}

	wantForward := `{"model":"gpt-4o","messages":[` +
		`{"role":"user","content":[{"type":"text","text":"zulu report"},{"type":"text","text":"alpha report"}]}` +
		`],"stream":true,"stream_options":{"include_usage":true}}`
	wantReversed := `{"model":"gpt-4o","messages":[` +
		`{"role":"user","content":[{"type":"text","text":"alpha report"},{"type":"text","text":"zulu report"}]}` +
		`],"stream":true,"stream_options":{"include_usage":true}}`

	if string(forwardBytes) != wantForward {
		t.Fatalf("Translate(forward) =\n%s\nwant\n%s", forwardBytes, wantForward)
	}
	if string(reversedBytes) != wantReversed {
		t.Fatalf("Translate(reversed) =\n%s\nwant\n%s", reversedBytes, wantReversed)
	}
	if string(forwardBytes) == string(reversedBytes) {
		t.Fatalf("Translate(forward) == Translate(reversed) = %s, want different bytes — intra-message part order must be carried, not incidentally equal", forwardBytes)
	}
}

// partKindDisposition records, for one ai.PartKinds() member, whether this
// phase renders it or defers it — and, when deferred, precisely which
// later phase/node owns it and what that phase must verify. This is a
// phase-5-local bookkeeping table, distinct from AI-26.8's own production
// policy.go table (Phase 8): it exists only to keep S-ART-027's "none is
// unaddressed" claim mechanically total for this node, walking
// ai.PartKinds() itself rather than a hand-maintained prose list that
// could silently rot as the vocabulary grows.
type partKindDisposition struct {
	kind    ai.PartKind
	handled bool
	handoff string // required when handled is false; empty when handled
}

var partKindDispositions = []partKindDisposition{
	{kind: ai.PartKindText, handled: true},
	{
		kind:    ai.PartKindReasoning,
		handled: false,
		handoff: "AI-26.6 (Phase 7): policy.go's refuse() door must intercept a reasoning part — every state, every position — before appendBody starts appending anything, returning ai.PreStreamFailure+ai.ErrUnsupportedCapability naming \"reasoning\", with no wire body produced even when the reasoning part sits after otherwise-renderable messages. message.go's current panic is a transitional safety net, not the replacement site.",
	},
	{
		kind:    ai.PartKindToolCall,
		handled: false,
		handoff: "AI-26.5 (Phase 6): a tool call renders as an element of a SEPARATE top-level tool_calls array on the assistant message object (claim 2, doc.go) — never as a content-part array element. Phase 6 must intercept PartKindToolCall in appendMessageObject/appendMessageContent, not grow appendContentPartObject's case for it.",
	},
	{
		kind:    ai.PartKindToolResult,
		handled: false,
		handoff: "AI-26.5 (Phase 6): a tool result renders as its own distinct wire message carrying tool_call_id and content directly on the message object (R-ART-012) — never as a content-part array element on some other message. Phase 6 must special-case a RoleTool message in appendMessageObject itself, not grow appendContentPartObject's case for it.",
	},
}

// lookupPartKindDisposition returns kind's recorded disposition and
// whether one was found — a linear scan over a slice, per this package's
// own "no map decides anything" convention (design.md "Map discipline";
// the same reasoning role.go/content_part.go already apply to their own
// tables).
func lookupPartKindDisposition(kind ai.PartKind) (partKindDisposition, bool) {
	for _, d := range partKindDispositions {
		if d.kind == kind {
			return d, true
		}
	}
	return partKindDisposition{}, false
}

// TestMessage_PartKindCoverage proves S-ART-027: every member of
// ai.PartKinds() is accounted for, mechanically — by walking the
// vocabulary's own enumerator, not a hand-maintained list this test could
// silently outgrow. A kind with no entry in partKindDispositions fails,
// naming it.
func TestMessage_PartKindCoverage(t *testing.T) {
	for _, kind := range ai.PartKinds() {
		t.Run(kind.String(), func(t *testing.T) {
			disposition, ok := lookupPartKindDisposition(kind)
			if !ok {
				t.Fatalf("ai.PartKind %s has no recorded disposition — ai.PartKinds() grew and partKindDispositions was not updated (S-ART-027)", kind)
			}

			switch disposition.kind {
			case ai.PartKindText:
				assertKindTranslates(t, disposition)
			case ai.PartKindReasoning:
				message := mustMessage(ai.NewMessage(ai.RoleAssistant, mustPart(ai.NewReasoning("thinking it through", nil))))
				assertKindIsDeferred(t, disposition, message)
			case ai.PartKindToolCall:
				message := mustMessage(ai.NewMessage(ai.RoleAssistant, mustPart(ai.NewToolCall("call_1", "get_weather", []byte(`{}`)))))
				assertKindIsDeferred(t, disposition, message)
			case ai.PartKindToolResult:
				// A tool result must correlate to a tool call somewhere in
				// the same request (ai.NewRequest's own
				// unresolvedToolResultRule, R-AMR-012) — "somewhere", not
				// "earlier" — so a bare tool-result-only request fails at
				// mustRequest, before Translate is ever reached. The
				// satisfying tool call is therefore supplied as a SECOND
				// message, positioned after the tool-result message under
				// test: message order is preserved (R-ART-008, this
				// file's own TestMessage_AdjacentSwap_TranslatesDifferently),
				// so appendMessagesField reaches the tool-result message
				// first and panics naming "tool_result" before it ever
				// reaches the tool-call message — which is itself also a
				// deferred kind and would otherwise panic naming
				// "tool_call" instead, misattributing the observed panic.
				resultMessage := mustMessage(ai.NewMessage(ai.RoleTool, mustPart(ai.NewToolResult("call_1", "sunny"))))
				callMessage := mustMessage(ai.NewMessage(ai.RoleAssistant, mustPart(ai.NewToolCall("call_1", "get_weather", []byte(`{}`)))))
				assertKindIsDeferred(t, disposition, resultMessage, callMessage)
			default:
				t.Fatalf("ai.PartKind %s has a disposition entry but this test's own switch was not updated to exercise it", kind)
			}
		})
	}
}

// assertKindTranslates proves a "handled: true" kind actually translates
// without error — the coverage test's own positive half.
func assertKindTranslates(t *testing.T, disposition partKindDisposition) {
	t.Helper()
	if !disposition.handled {
		t.Fatalf("assertKindTranslates called for a kind recorded handled: false (%s)", disposition.kind)
	}
	req := mustRequest(ai.NewRequest("gpt-4o", []ai.Message{
		mustMessage(ai.NewMessage(ai.RoleUser, mustPart(ai.NewText("coverage: text renders")))),
	}))
	if _, err := openaicompat.Translate(req); err != nil {
		t.Fatalf("Translate: unexpected error for a handled kind (%s): %v", disposition.kind, err)
	}
}

// assertKindIsDeferred proves a deferred kind is neither silently rendered
// nor silently dropped: translating a request whose FIRST message carries
// only that kind currently panics with a message naming the kind by its
// own registered name (disposition.kind.String()), rather than succeeding
// with a wrong or incomplete body, and rather than merely panicking for
// some unrelated reason (R-ART-007). Checking the panic message's content,
// not only that a panic occurred, is deliberate: a generic "content shape
// not supported" panic that names no kind would satisfy a bare panic check
// just as well without this file's own per-kind handling existing at all —
// exactly the vacuous-pass shape ("fixture cannot distinguish implemented
// from not-implemented") this milestone's evidence discipline exists to
// catch.
//
// messages accepts more than one message so a kind whose construction
// rules need a cross-request correlate (a tool result needs a tool call
// "somewhere" in the request, ai.NewRequest's own unresolvedToolResultRule)
// can still build a request that survives mustRequest — the correlate is
// supplied as a later message, never the first, so the panic this function
// observes is always attributable to messages[0]'s own kind.
//
// It also requires disposition.handoff to be recorded, so a "handled:
// false" entry can never ship without stating which later phase owns it
// and what that phase must verify.
func assertKindIsDeferred(t *testing.T, disposition partKindDisposition, messages ...ai.Message) {
	t.Helper()
	if disposition.handled {
		t.Fatalf("assertKindIsDeferred called for a kind recorded handled: true (%s)", disposition.kind)
	}
	if disposition.handoff == "" {
		t.Fatalf("ai.PartKind %s is recorded deferred but carries no hand-off note — S-ART-027 requires every unaddressed kind to name what later phase owns it", disposition.kind)
	}
	if len(messages) == 0 {
		t.Fatalf("assertKindIsDeferred called with no messages")
	}

	req := mustRequest(ai.NewRequest("gpt-4o", messages))

	recovered := func() (recovered any) {
		defer func() {
			recovered = recover()
		}()
		_, _ = openaicompat.Translate(req)
		return nil
	}()

	if recovered == nil {
		t.Fatalf("Translate did not panic for a deferred kind (%s) — it must fail loudly, not silently render or silently drop the part (R-ART-007)", disposition.kind)
	}
	panicMessage, ok := recovered.(string)
	if !ok {
		t.Fatalf("Translate panicked with a non-string value (%v, %T) for a deferred kind (%s)", recovered, recovered, disposition.kind)
	}
	if !strings.Contains(panicMessage, disposition.kind.String()) {
		t.Fatalf("Translate panicked with %q, want a message naming the deferred kind %q", panicMessage, disposition.kind.String())
	}
}
