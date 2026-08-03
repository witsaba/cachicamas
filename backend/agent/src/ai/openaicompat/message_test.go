// AI-26.3 — every readable content-part variant translates, message and
// intra-message part order are preserved exactly, and consecutive
// same-role messages are never merged (R-ART-007, R-ART-008, R-ART-009).
package openaicompat_test

import (
	"errors"
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

// partKindDisposition records, for one ai.PartKinds() member, whether
// this phase renders it or refuses it with a typed error. This is a
// phase-5-local bookkeeping table, distinct from AI-26.8's own
// production policy.go table (Phase 8): it exists only to keep
// S-ART-027's "none is unaddressed" claim mechanically total for this
// node, walking ai.PartKinds() itself rather than a hand-maintained prose
// list that could silently rot as the vocabulary grows.
//
// A third disposition — deferred, accounted for by neither renders nor
// refuses, only a recorded hand-off to a later phase — existed through
// AI-26.3/AI-26.5, for PartKindReasoning specifically. AI-26.6 (Phase 7)
// resolved that hand-off (refused, not rendered), which leaves zero
// currently-declared ai.PartKinds() members deferred; the "deferred"
// disposition and its own proof (a checked, panic-based assertion) were
// removed with it rather than kept as unreachable dead code no test
// exercised — lookupPartKindDisposition's own ok==false branch, below,
// remains the mechanism that catches a genuinely new, unaccounted-for
// kind, deferred or otherwise, the moment ai.PartKinds() grows again.
type partKindDisposition struct {
	kind    ai.PartKind
	handled bool
	refused bool // only meaningful when handled is true: renders (false) vs. refuses with a typed error (true)
}

var partKindDispositions = []partKindDisposition{
	{kind: ai.PartKindText, handled: true},
	// PartKindReasoning was AI-26.3's (Phase 5) own hand-off, recorded here
	// at that phase in this file's prior revision — see that revision for
	// its exact text. AI-26.6 (Phase 7, policy.go's refuse() door,
	// refuseReasoning) now intercepts every reasoning part, every state,
	// every position, before Translate ever calls appendBody: it is
	// "handled" in the sense that S-ART-027 means (accounted for,
	// mechanically, not merely by inspection), but handled BY REFUSAL, not
	// by rendering — assertKindRefuses (below) is this table's own
	// positive proof of that, distinct from assertKindTranslates' proof
	// for the three kinds that render. See doc.go's "Reasoning replay
	// refuses; Layer 2 must strip first" section (R-ART-015).
	{kind: ai.PartKindReasoning, handled: true, refused: true},
	// PartKindToolCall and PartKindToolResult carried the AI-26.5 (Phase 6)
	// hand-off recorded here at AI-26.3 (Phase 5) — see this file's own
	// prior revision for its exact text. Both are now handled: a tool call
	// renders as an element of message.go's own tool_calls array
	// (splitToolCalls, appendToolCallObject); a tool result renders as its
	// own distinct role:"tool" wire message (appendToolResultMessages,
	// appendToolResultObject). See doc.go's "Tool calls render in their
	// own tool_calls array; tool results render as their own distinct
	// role:\"tool\" messages" section (R-ART-012, R-ART-013, R-ART-014).
	{kind: ai.PartKindToolCall, handled: true},
	{kind: ai.PartKindToolResult, handled: true},
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
				message := mustMessage(ai.NewMessage(ai.RoleUser, mustPart(ai.NewText("coverage: text renders"))))
				assertKindTranslates(t, disposition, message)
			case ai.PartKindReasoning:
				message := mustMessage(ai.NewMessage(ai.RoleAssistant, mustPart(ai.NewReasoning("thinking it through", nil))))
				assertKindRefuses(t, disposition, message)
			case ai.PartKindToolCall:
				message := mustMessage(ai.NewMessage(ai.RoleAssistant, mustPart(ai.NewToolCall("call_1", "get_weather", []byte(`{}`)))))
				assertKindTranslates(t, disposition, message)
			case ai.PartKindToolResult:
				// A tool result must correlate to a tool call somewhere in
				// the same request (ai.NewRequest's own
				// unresolvedToolResultRule, R-AMR-012) — "somewhere", not
				// "earlier" — so a bare tool-result-only request fails at
				// mustRequest, before Translate is ever reached. The
				// satisfying tool call is therefore supplied as a SECOND
				// message. Unlike this file's prior revision (AI-26.3),
				// misattribution is no longer a risk to guard against:
				// this phase (AI-26.5) renders both kinds, so
				// assertKindTranslates only has to observe "no error",
				// never attribute a panic to the right one of two
				// deferred kinds.
				resultMessage := mustMessage(ai.NewMessage(ai.RoleTool, mustPart(ai.NewToolResult("call_1", "sunny"))))
				callMessage := mustMessage(ai.NewMessage(ai.RoleAssistant, mustPart(ai.NewToolCall("call_1", "get_weather", []byte(`{}`)))))
				assertKindTranslates(t, disposition, resultMessage, callMessage)
			default:
				t.Fatalf("ai.PartKind %s has a disposition entry but this test's own switch was not updated to exercise it", kind)
			}
		})
	}
}

// assertKindTranslates proves a "handled: true, refused: false" kind
// actually translates without error — the coverage test's own
// render-side proof — using messages that genuinely carry that kind. It
// does not fall back to a generic Text-only smoke request: doing so would
// pass regardless of whether the kind under test renders at all, exactly
// the "fixture cannot distinguish implemented from not-implemented" trap
// this milestone's own evidence discipline exists to catch (assertKindRefuses,
// below, names the same trap for the refusal-side proof).
func assertKindTranslates(t *testing.T, disposition partKindDisposition, messages ...ai.Message) {
	t.Helper()
	if !disposition.handled || disposition.refused {
		t.Fatalf("assertKindTranslates called for a kind not recorded handled+not-refused (%s)", disposition.kind)
	}
	if len(messages) == 0 {
		t.Fatalf("assertKindTranslates called with no messages")
	}
	req := mustRequest(ai.NewRequest("gpt-4o", messages))
	if _, err := openaicompat.Translate(req); err != nil {
		t.Fatalf("Translate: unexpected error for a handled kind (%s): %v", disposition.kind, err)
	}
}

// assertKindRefuses proves a "handled: true, refused: true" kind fails
// translation with a typed AI-19 refusal, reachable via
// errors.Is(err, ai.ErrUnsupportedCapability) through ai.Failure's own Is
// method — never with an unrecovered panic, which is what this same kind
// (PartKindReasoning) produced before AI-26.6 (Phase 7) landed the
// refusal door (a plain, checked panic-content assertion, the same
// "check content, not occurrence" discipline this function now applies
// to the returned error instead). Checking error identity rather than
// merely "Translate returned a non-nil error" is deliberate: a
// translation that failed for some unrelated reason would satisfy a bare
// non-nil check just as well, which is exactly the vacuous-pass shape
// this milestone's evidence discipline exists to catch (this coverage
// test's own analogue of reasoning_refusal_test.go's
// TestRefusalCases_FailWithUnsupportedCapability).
//
// It also asserts no wire body is produced alongside the error (R-ART-015
// — "no wire body is produced"), and that the returned error's own
// unwrapped cause names disposition.kind.String() ("reasoning") — the
// same "name the kind, not just fail somehow" discipline the pre-AI-26.6
// panic-content check applied to the panic message it used to check.
func assertKindRefuses(t *testing.T, disposition partKindDisposition, messages ...ai.Message) {
	t.Helper()
	if !disposition.handled || !disposition.refused {
		t.Fatalf("assertKindRefuses called for a kind not recorded handled+refused (%s)", disposition.kind)
	}
	if len(messages) == 0 {
		t.Fatalf("assertKindRefuses called with no messages")
	}

	req := mustRequest(ai.NewRequest("gpt-4o", messages))

	got, err := openaicompat.Translate(req)
	if err == nil {
		t.Fatalf("Translate: no error for a refused kind (%s) — R-ART-015 requires a typed refusal, not a silent success", disposition.kind)
	}
	if got != nil {
		t.Fatalf("Translate: got %d byte(s) of wire body alongside a refusal error for kind %s, want none", len(got), disposition.kind)
	}
	if !errors.Is(err, ai.ErrUnsupportedCapability) {
		t.Fatalf("Translate error = %v, want errors.Is(err, ai.ErrUnsupportedCapability) for a refused kind (%s)", err, disposition.kind)
	}
	cause := errors.Unwrap(err)
	if cause == nil || !strings.Contains(cause.Error(), disposition.kind.String()) {
		t.Fatalf("Translate error's unwrapped cause = %v, want it to name the refused kind %q", cause, disposition.kind.String())
	}
}
