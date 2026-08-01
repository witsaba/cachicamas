// Tests for AI-11.1 and AI-11.2 — cache-boundary markers, their cap and their
// cascade order.
//
// External package, for the reason every contract test in this package
// gives: the only consumer a marker exists for is an adapter, and every
// adapter lives in another package (R-ACB-003).
package ai_test

import (
	"fmt"
	"slices"
	"strings"
	"testing"

	"github.com/cachicamas/backend/agent/src/ai"
)

// AI-11.1 item 1, S-ACB-001 — a constructed system-instruction segment can
// carry a cache-boundary marker, and marking it changes nothing but the
// marker: the result reports it is a cache boundary and its text reads back
// byte-equal to the original's.
func TestSegment_MarkCacheBoundary_RoundTripsThroughConstruction(t *testing.T) {
	t.Parallel()

	const text = "You are a travel planning assistant."

	seg := segment(t, text)
	marked := seg.MarkCacheBoundary()

	if !marked.IsCacheBoundary() {
		t.Errorf("marked.IsCacheBoundary() = false, want true")
	}
	if got := marked.Text(); got != text {
		t.Errorf("marked.Text() = %q, want %q", got, text)
	}
}

// AI-11.1 item 1, S-ACB-002 — a constructed tool declaration can carry a
// cache-boundary marker, and marking it changes nothing but the marker: the
// result reports it is a cache boundary and its name, description and
// schema bytes read back unchanged.
func TestTool_MarkCacheBoundary_RoundTripsThroughConstruction(t *testing.T) {
	t.Parallel()

	const (
		name        = "search_flights"
		description = "search for flights between two airports"
	)
	schema := []byte(`{"type":"object"}`)

	tool, err := ai.NewTool(name, description, schema)
	if err != nil {
		t.Fatalf("ai.NewTool returned %v, want no failure", err)
	}
	marked := tool.MarkCacheBoundary()

	if !marked.IsCacheBoundary() {
		t.Errorf("marked.IsCacheBoundary() = false, want true")
	}
	if got := marked.Name(); got != name {
		t.Errorf("marked.Name() = %q, want %q", got, name)
	}
	if got := marked.Description(); got != description {
		t.Errorf("marked.Description() = %q, want %q", got, description)
	}
	if got := string(marked.Schema()); got != string(schema) {
		t.Errorf("marked.Schema() = %q, want %q", got, schema)
	}
}

// AI-11.1 item 1, S-ACB-003 — a constructed message can carry a
// cache-boundary marker, and marking it changes nothing but the marker: the
// result reports it is a cache boundary and its role, its ordered content
// and its identity read back unchanged.
func TestMessage_MarkCacheBoundary_RoundTripsThroughConstruction(t *testing.T) {
	t.Parallel()

	msg := requireMessage(t, ai.RoleAssistant, textPart(t, "plan a trip"))
	marked := msg.MarkCacheBoundary()

	if !marked.IsCacheBoundary() {
		t.Errorf("marked.IsCacheBoundary() = false, want true")
	}
	if got := marked.Role(); got != ai.RoleAssistant {
		t.Errorf("marked.Role() = %v, want %v", got, ai.RoleAssistant)
	}
	if got, ok := marked.Content()[0].Text(); !ok || got != "plan a trip" {
		t.Errorf("marked.Content()[0].Text() = (%q, %t), want (%q, true)", got, ok, "plan a trip")
	}
	if marked.ID() != msg.ID() {
		t.Errorf("marked.ID() = %v, want %v — a marked copy carries the same identity", marked.ID(), msg.ID())
	}
}

// AI-11.1 item 1, S-ACB-004 — placement returns a copy: on all three
// carriers, the value MarkCacheBoundary was called on still reports itself
// unmarked after the call.
func TestCarrier_MarkCacheBoundary_LeavesTheOriginalValueUnmarked(t *testing.T) {
	t.Parallel()

	t.Run("segment", func(t *testing.T) {
		t.Parallel()

		original := segment(t, "be terse")
		_ = original.MarkCacheBoundary()

		if original.IsCacheBoundary() {
			t.Errorf("original.IsCacheBoundary() = true after MarkCacheBoundary() was called on it, want false")
		}
	})

	t.Run("tool", func(t *testing.T) {
		t.Parallel()

		original := requireTool(t, "search_flights")
		_ = original.MarkCacheBoundary()

		if original.IsCacheBoundary() {
			t.Errorf("original.IsCacheBoundary() = true after MarkCacheBoundary() was called on it, want false")
		}
	})

	t.Run("message", func(t *testing.T) {
		t.Parallel()

		original := requireMessage(t, ai.RoleUser, textPart(t, "hello"))
		_ = original.MarkCacheBoundary()

		if original.IsCacheBoundary() {
			t.Errorf("original.IsCacheBoundary() = true after MarkCacheBoundary() was called on it, want false")
		}
	})
}

// AI-11.1 item 1, S-ACB-005 and S-ACB-006 — a never-marked carrier of each
// kind reports it is not a cache boundary, and marking an already-marked
// carrier again is idempotent. Tool and Message are non-comparable, so
// idempotence is asserted field-wise rather than with ==.
func TestCarrier_NeverMarkedOrMarkedTwice_ReportsCorrectly(t *testing.T) {
	t.Parallel()

	t.Run("segment", func(t *testing.T) {
		t.Parallel()

		fresh := segment(t, "be terse")
		if fresh.IsCacheBoundary() {
			t.Errorf("a never-marked segment reports IsCacheBoundary() = true, want false")
		}

		once := fresh.MarkCacheBoundary()
		twice := once.MarkCacheBoundary()
		if twice != once {
			t.Errorf("marking an already-marked segment again changed it: %v, want %v", twice, once)
		}
		if !twice.IsCacheBoundary() {
			t.Errorf("twice.IsCacheBoundary() = false, want true")
		}
	})

	t.Run("tool", func(t *testing.T) {
		t.Parallel()

		fresh := requireTool(t, "search_flights")
		if fresh.IsCacheBoundary() {
			t.Errorf("a never-marked tool reports IsCacheBoundary() = true, want false")
		}

		once := fresh.MarkCacheBoundary()
		twice := once.MarkCacheBoundary()
		if once.Name() != twice.Name() || once.Description() != twice.Description() ||
			string(once.Schema()) != string(twice.Schema()) || once.IsCacheBoundary() != twice.IsCacheBoundary() {
			t.Errorf("marking an already-marked tool again changed it: %+v, want %+v", twice, once)
		}
		if !twice.IsCacheBoundary() {
			t.Errorf("twice.IsCacheBoundary() = false, want true")
		}
	})

	t.Run("message", func(t *testing.T) {
		t.Parallel()

		fresh := requireMessage(t, ai.RoleUser, textPart(t, "hello"))
		if fresh.IsCacheBoundary() {
			t.Errorf("a never-marked message reports IsCacheBoundary() = true, want false")
		}

		once := fresh.MarkCacheBoundary()
		twice := once.MarkCacheBoundary()
		if once.ID() != twice.ID() || once.Role() != twice.Role() || once.IsCacheBoundary() != twice.IsCacheBoundary() {
			t.Errorf("marking an already-marked message again changed it: %+v, want %+v", twice, once)
		}
		if !twice.IsCacheBoundary() {
			t.Errorf("twice.IsCacheBoundary() = false, want true")
		}
	})
}

// markedEquivalentRequest builds two requests from the same shape — a
// system segment, one declared tool with a tool choice, and one message —
// where marked additionally carries a marker on every carrier. It is the
// fixture AI-11.1 item 2 shares across S-ACB-016 … S-ACB-019.
func markedEquivalentRequest(t *testing.T) (plain, marked ai.Request) {
	t.Helper()

	buildOne := func(mark bool) ai.Request {
		seg := segment(t, "be terse")
		if mark {
			seg = seg.MarkCacheBoundary()
		}
		system, err := ai.NewSystemInstruction(seg)
		if err != nil {
			t.Fatalf("ai.NewSystemInstruction returned %v, want no failure", err)
		}

		tool := requireTool(t, "search_flights")
		if mark {
			tool = tool.MarkCacheBoundary()
		}
		toolSet, err := ai.NewToolSet(tool)
		if err != nil {
			t.Fatalf("ai.NewToolSet returned %v, want no failure", err)
		}
		choice := requireNamedToolChoice(t, "search_flights")

		msg := requireMessage(t, ai.RoleUser, textPart(t, "plan a trip"))
		if mark {
			msg = msg.MarkCacheBoundary()
		}

		request, err := ai.NewRequest(
			"m", []ai.Message{msg},
			ai.WithSystemInstruction(system),
			ai.WithTools(toolSet),
			ai.WithToolChoice(choice),
		)
		if err != nil {
			t.Fatalf("ai.NewRequest(mark=%t) returned %v, want no failure", mark, err)
		}
		return request
	}

	return buildOne(false), buildOne(true)
}

// AI-11.1 item 2, S-ACB-016 — a valid request and its twin carrying markers
// on every carrier both construct successfully: a marker never decides
// whether a request is valid.
func TestRequest_MarkedTwinOfAValidRequest_BothConstructSuccessfully(t *testing.T) {
	t.Parallel()

	plain, marked := markedEquivalentRequest(t)

	if got := plain.Model(); got != "m" {
		t.Errorf("plain.Model() = %q, want %q", got, "m")
	}
	if got := marked.Model(); got != "m" {
		t.Errorf("marked.Model() = %q, want %q", got, "m")
	}
}

// AI-11.1 item 2, S-ACB-017 — an invalid request and its twin carrying the
// same markers fail with the same rule class at the same rendered position:
// a marker never causes a request to fail either.
func TestRequest_MarkedTwinOfAnInvalidRequest_FailsWithTheSameRuleAtTheSamePosition(t *testing.T) {
	t.Parallel()

	buildWithEmptyModel := func(mark bool) (ai.Request, error) {
		msg := requireMessage(t, ai.RoleUser, textPart(t, "hello"))
		if mark {
			msg = msg.MarkCacheBoundary()
		}
		return ai.NewRequest("", []ai.Message{msg})
	}

	_, plainErr := buildWithEmptyModel(false)
	_, markedErr := buildWithEmptyModel(true)

	requireViolation(t, plainErr, ai.ErrEmpty, "model")
	requireViolation(t, markedErr, ai.ErrEmpty, "model")
}

// AI-11.1 item 2, S-ACB-018 — every combination of marking a segment, a
// declaration and a message constructs successfully: there is no placement
// rule by role, by content kind, or by which carriers are marked together.
func TestRequest_EveryMarkerPlacementCombination_Constructs(t *testing.T) {
	t.Parallel()

	for markSegment := range 2 {
		for markTool := range 2 {
			for markMessage := range 2 {
				t.Run("", func(t *testing.T) {
					t.Parallel()

					seg := segment(t, "be terse")
					if markSegment == 1 {
						seg = seg.MarkCacheBoundary()
					}
					system, err := ai.NewSystemInstruction(seg)
					if err != nil {
						t.Fatalf("ai.NewSystemInstruction returned %v, want no failure", err)
					}

					tool := requireTool(t, "search_flights")
					if markTool == 1 {
						tool = tool.MarkCacheBoundary()
					}
					toolSet, err := ai.NewToolSet(tool)
					if err != nil {
						t.Fatalf("ai.NewToolSet returned %v, want no failure", err)
					}

					msg := requireMessage(t, ai.RoleUser, textPart(t, "plan a trip"))
					if markMessage == 1 {
						msg = msg.MarkCacheBoundary()
					}

					_, err = ai.NewRequest(
						"m", []ai.Message{msg},
						ai.WithSystemInstruction(system),
						ai.WithTools(toolSet),
					)
					if err != nil {
						t.Errorf("ai.NewRequest(segment=%d, tool=%d, message=%d) returned %v, want no failure",
							markSegment, markTool, markMessage, err)
					}
				})
			}
		}
	}
}

// AI-11.1 item 2, S-ACB-019 — a request whose marked segment carries a
// single character constructs: minimum cacheable prefix length is not
// decidable from the request alone (AI-19's, not this layer's).
func TestRequest_MarkedSingleCharacterSegment_Constructs(t *testing.T) {
	t.Parallel()

	seg := segment(t, "x").MarkCacheBoundary()
	system, err := ai.NewSystemInstruction(seg)
	if err != nil {
		t.Fatalf("ai.NewSystemInstruction returned %v, want no failure", err)
	}

	request, err := ai.NewRequest(
		"m", []ai.Message{requireMessage(t, ai.RoleUser, textPart(t, "hello"))},
		ai.WithSystemInstruction(system),
	)
	if err != nil {
		t.Fatalf("ai.NewRequest returned %v, want no failure", err)
	}
	read, ok := request.SystemInstruction()
	if !ok || read.Segments()[0].Text() != "x" {
		t.Errorf("request.SystemInstruction() = (%v, %t), want a one-character marked segment", read, ok)
	}
}

// buildRequestMarkingOneCarrier constructs a request from one segment, one
// tool and one message, optionally marking exactly one named carrier.
// AI-11.1 item 2's equality tests use it to isolate which carrier's marker
// state Request.Equal is sensitive to.
func buildRequestMarkingOneCarrier(t *testing.T, markCarrier string) ai.Request {
	t.Helper()

	seg := segment(t, "be terse")
	tool := requireTool(t, "search_flights")
	msg := requireMessage(t, ai.RoleUser, textPart(t, "plan a trip"))

	switch markCarrier {
	case "segment":
		seg = seg.MarkCacheBoundary()
	case "tool":
		tool = tool.MarkCacheBoundary()
	case "message":
		msg = msg.MarkCacheBoundary()
	case "":
		// no marker placed anywhere
	default:
		t.Fatalf("buildRequestMarkingOneCarrier: unknown carrier %q", markCarrier)
	}

	system, err := ai.NewSystemInstruction(seg)
	if err != nil {
		t.Fatalf("ai.NewSystemInstruction returned %v, want no failure", err)
	}
	toolSet, err := ai.NewToolSet(tool)
	if err != nil {
		t.Fatalf("ai.NewToolSet returned %v, want no failure", err)
	}

	request, err := ai.NewRequest(
		"m", []ai.Message{msg},
		ai.WithSystemInstruction(system),
		ai.WithTools(toolSet),
	)
	if err != nil {
		t.Fatalf("ai.NewRequest returned %v, want no failure", err)
	}
	return request
}

// AI-11.1 item 2, S-ACB-013 and S-ACB-014 — two requests identical but for
// one marker are not equal under the documented equality, on every carrier a
// marker can be placed on; two requests carrying identical markers on
// identical ordinals are equal (R-ACB-004).
//
// All three carriers are exercised, not only the segment design.md's own
// example names: Request.Equal composes SystemInstruction.Equal (which
// picks up a new Segment field for free, because Segment stays comparable),
// but Message.Equal and the request's own toolsEqual name their compared
// fields explicitly and do not see a new field unless it is named there —
// explore.md § 8 rows 4-5, corrected.
func TestRequest_MarkersDifferByOneMarker_AreNotEqualButIdenticalMarkersAreEqual(t *testing.T) {
	t.Parallel()

	unmarked := buildRequestMarkingOneCarrier(t, "")

	for _, carrier := range []string{"segment", "tool", "message"} {
		t.Run(carrier, func(t *testing.T) {
			t.Parallel()

			markedOnce := buildRequestMarkingOneCarrier(t, carrier)
			if unmarked.Equal(markedOnce) {
				t.Errorf("unmarked.Equal(markedOnce) = true for carrier %q, want false — "+
					"they differ in exactly one marker", carrier)
			}
			if markedOnce.Equal(unmarked) {
				t.Errorf("markedOnce.Equal(unmarked) = true for carrier %q, want false — Equal must be symmetric about inequality too", carrier)
			}

			markedAgain := buildRequestMarkingOneCarrier(t, carrier)
			if !markedOnce.Equal(markedAgain) {
				t.Errorf("two independently built requests carrying identical markers on carrier %q "+
					"are not equal, want equal", carrier)
			}
		})
	}
}

// AI-11.1 item 2, S-ACB-015 — a marked request read region by region through
// the exported surface and rebuilt from what was read is equal to the
// original under the documented equality: no marker is lost in the round
// trip (R-ACB-004).
func TestRequest_MarkedRequestReadAndRebuilt_IsEqualToTheOriginal(t *testing.T) {
	t.Parallel()

	original := buildRequestMarkingOneCarrier(t, "segment")

	// Read every region back through the exported surface alone.
	system, hasSystem := original.SystemInstruction()
	if !hasSystem {
		t.Fatalf("original.SystemInstruction() reported absent, want present")
	}
	var rebuiltSegments []ai.Segment
	for _, seg := range system.Segments() {
		rebuiltSeg, err := ai.NewSegment(seg.Text())
		if err != nil {
			t.Fatalf("ai.NewSegment returned %v, want no failure", err)
		}
		if seg.IsCacheBoundary() {
			rebuiltSeg = rebuiltSeg.MarkCacheBoundary()
		}
		rebuiltSegments = append(rebuiltSegments, rebuiltSeg)
	}
	rebuiltSystem, err := ai.NewSystemInstruction(rebuiltSegments...)
	if err != nil {
		t.Fatalf("ai.NewSystemInstruction returned %v, want no failure", err)
	}

	tools, hasTools := original.Tools()
	if !hasTools {
		t.Fatalf("original.Tools() reported absent, want present")
	}
	var rebuiltTools []ai.Tool
	for _, tool := range tools.Tools() {
		rebuiltTool, err := ai.NewTool(tool.Name(), tool.Description(), tool.Schema())
		if err != nil {
			t.Fatalf("ai.NewTool returned %v, want no failure", err)
		}
		if tool.IsCacheBoundary() {
			rebuiltTool = rebuiltTool.MarkCacheBoundary()
		}
		rebuiltTools = append(rebuiltTools, rebuiltTool)
	}
	rebuiltToolSet, err := ai.NewToolSet(rebuiltTools...)
	if err != nil {
		t.Fatalf("ai.NewToolSet returned %v, want no failure", err)
	}

	var rebuiltMessages []ai.Message
	for _, msg := range original.Messages() {
		rebuiltMsg, err := ai.NewMessage(msg.Role(), msg.Content()...)
		if err != nil {
			t.Fatalf("ai.NewMessage returned %v, want no failure", err)
		}
		if msg.IsCacheBoundary() {
			rebuiltMsg = rebuiltMsg.MarkCacheBoundary()
		}
		rebuiltMessages = append(rebuiltMessages, rebuiltMsg)
	}

	rebuilt, err := ai.NewRequest(
		original.Model(), rebuiltMessages,
		ai.WithSystemInstruction(rebuiltSystem),
		ai.WithTools(rebuiltToolSet),
	)
	if err != nil {
		t.Fatalf("ai.NewRequest returned %v rebuilding the request, want no failure", err)
	}

	if !original.Equal(rebuilt) {
		t.Errorf("original.Equal(rebuilt) = false, want true — the marker was lost in the round trip")
	}
}

// AI-11.1 item 3, S-ACB-011 — a request built in this external-package test
// from marked segments, marked declarations and marked messages observes
// every marker at the ordinal it was placed on, through the existing region
// accessors alone: SystemInstruction.Segments, ToolSet.Tools and
// Request.Messages. No new accessor is added on SystemInstruction or
// ToolSet (R-ACB-003).
func TestRequest_MarkersOnEveryOrdinal_AreObservableThroughExistingAccessors(t *testing.T) {
	t.Parallel()

	segments := []ai.Segment{
		segment(t, "first"),
		segment(t, "second").MarkCacheBoundary(),
		segment(t, "third"),
	}
	system, err := ai.NewSystemInstruction(segments...)
	if err != nil {
		t.Fatalf("ai.NewSystemInstruction returned %v, want no failure", err)
	}

	tools := []ai.Tool{
		requireTool(t, "search_flights"),
		requireTool(t, "book_flight").MarkCacheBoundary(),
	}
	toolSet, err := ai.NewToolSet(tools...)
	if err != nil {
		t.Fatalf("ai.NewToolSet returned %v, want no failure", err)
	}

	messages := []ai.Message{
		requireMessage(t, ai.RoleUser, textPart(t, "one")),
		requireMessage(t, ai.RoleUser, textPart(t, "two")).MarkCacheBoundary(),
		requireMessage(t, ai.RoleUser, textPart(t, "three")),
	}

	request, err := ai.NewRequest(
		"m", messages,
		ai.WithSystemInstruction(system),
		ai.WithTools(toolSet),
	)
	if err != nil {
		t.Fatalf("ai.NewRequest returned %v, want no failure", err)
	}

	readSystem, _ := request.SystemInstruction()
	wantSegmentMarks := []bool{false, true, false}
	for i, seg := range readSystem.Segments() {
		if got := seg.IsCacheBoundary(); got != wantSegmentMarks[i] {
			t.Errorf("readSystem.Segments()[%d].IsCacheBoundary() = %t, want %t", i, got, wantSegmentMarks[i])
		}
	}

	readTools, _ := request.Tools()
	wantToolMarks := []bool{false, true}
	for i, tool := range readTools.Tools() {
		if got := tool.IsCacheBoundary(); got != wantToolMarks[i] {
			t.Errorf("readTools.Tools()[%d].IsCacheBoundary() = %t, want %t", i, got, wantToolMarks[i])
		}
	}

	wantMessageMarks := []bool{false, true, false}
	for i, msg := range request.Messages() {
		if got := msg.IsCacheBoundary(); got != wantMessageMarks[i] {
			t.Errorf("request.Messages()[%d].IsCacheBoundary() = %t, want %t", i, got, wantMessageMarks[i])
		}
	}
}

// AI-11.1 item 3, S-ACB-012 — reading each region twice observes identical
// markers, because each read returns a fresh copy and no consumer can
// rewrite another's view.
func TestRequest_MarkersReadTwice_AreIdentical(t *testing.T) {
	t.Parallel()

	request := buildRequestMarkingOneCarrier(t, "segment")

	firstSystem, _ := request.SystemInstruction()
	secondSystem, _ := request.SystemInstruction()
	if !slices.Equal(firstSystem.Segments(), secondSystem.Segments()) {
		t.Errorf("two reads of SystemInstruction().Segments() disagree: %v vs %v", firstSystem.Segments(), secondSystem.Segments())
	}

	// A consumer mutating what it read must not affect a later read.
	read := firstSystem.Segments()
	read[0] = ai.Segment{}
	stillMarked := false
	rereadSystem, _ := request.SystemInstruction()
	for _, seg := range rereadSystem.Segments() {
		if seg.IsCacheBoundary() {
			stillMarked = true
		}
	}
	if !stillMarked {
		t.Errorf("after a consumer rewrote the slice it read, the request's own marker disappeared")
	}
}

// AI-11.1 item 4, S-ACB-007 … S-ACB-009 — marking the zero Segment, the zero
// Tool and the zero Message returns a value still detectable as
// unconstructed by the same test that detected the original, and each is
// still rejected by its enclosing constructor with ErrEmpty at its index
// (R-ACB-002): a marker cannot make an unconstructed value usable.
func TestCarrier_MarkCacheBoundaryOnTheZeroValue_StillRejectedByItsEnclosingConstructor(t *testing.T) {
	t.Parallel()

	t.Run("segment", func(t *testing.T) {
		t.Parallel()

		var zero ai.Segment
		marked := zero.MarkCacheBoundary()
		if !marked.IsZero() {
			t.Errorf("the zero Segment's marked form reports IsZero() = false, want true")
		}

		_, err := ai.NewSystemInstruction(marked)
		requireViolation(t, err, ai.ErrEmpty, "system[0]")
	})

	t.Run("tool", func(t *testing.T) {
		t.Parallel()

		var zero ai.Tool
		marked := zero.MarkCacheBoundary()
		if marked.Name() != "" {
			t.Errorf("the zero Tool's marked form reports Name() = %q, want empty", marked.Name())
		}

		_, err := ai.NewToolSet(marked)
		requireViolation(t, err, ai.ErrEmpty, "tools[0]")
	})

	t.Run("message", func(t *testing.T) {
		t.Parallel()

		var zero ai.Message
		marked := zero.MarkCacheBoundary()
		if !marked.ID().IsZero() {
			t.Errorf("the zero Message's marked form reports ID().IsZero() = false, want true")
		}

		_, err := ai.NewRequest("m", []ai.Message{marked})
		requireViolation(t, err, ai.ErrEmpty, "messages[0]")
	})
}

// AI-11.1 item 4, S-ACB-010 — none of the three marked zero values reports
// itself a cache boundary.
func TestCarrier_MarkCacheBoundaryOnTheZeroValue_ReportsNotACacheBoundary(t *testing.T) {
	t.Parallel()

	var zeroSegment ai.Segment
	if got := zeroSegment.MarkCacheBoundary().IsCacheBoundary(); got {
		t.Errorf("the zero Segment's marked form reports IsCacheBoundary() = true, want false")
	}

	var zeroTool ai.Tool
	if got := zeroTool.MarkCacheBoundary().IsCacheBoundary(); got {
		t.Errorf("the zero Tool's marked form reports IsCacheBoundary() = true, want false")
	}

	var zeroMessage ai.Message
	if got := zeroMessage.MarkCacheBoundary().IsCacheBoundary(); got {
		t.Errorf("the zero Message's marked form reports IsCacheBoundary() = true, want false")
	}
}

// AI-11.1 item 5, S-ACB-038 — a marked system-instruction segment holding a
// secret renders through the default, string, extended and Go-syntax verbs
// naming that it is a cache boundary and reproducing no secret (R-ACB-010).
func TestSegment_MarkedRendering_NamesTheBoundaryAndReproducesNoSecret(t *testing.T) {
	t.Parallel()

	const secret = "SECRET-SYSTEM-PROMPT"

	marked := segment(t, secret).MarkCacheBoundary()

	for _, verb := range []string{"%v", "%s", "%+v", "%#v"} {
		rendered := fmt.Sprintf(verb, marked)
		if strings.Contains(rendered, secret) {
			t.Errorf("fmt.Sprintf(%q, marked) leaked the segment text: %q", verb, rendered)
		}
		if !strings.Contains(rendered, "cache boundary") {
			t.Errorf("fmt.Sprintf(%q, marked) = %q, want it to name the cache boundary", verb, rendered)
		}
	}
}

// AI-11.1 item 5, S-ACB-039 — an unmarked segment's rendering is
// byte-identical to what it was before this milestone.
//
// system_instruction_test.go:288 already asserts segment(t, "a").String()
// == "segment" and must stay green; this test names the property directly
// under this leaf's own banner rather than relying on that assertion alone.
func TestSegment_UnmarkedRendering_IsUnchangedFromBeforeThisMilestone(t *testing.T) {
	t.Parallel()

	if got, want := segment(t, "a").String(), "segment"; got != want {
		t.Errorf("segment.String() = %q, want %q — unchanged by AI-11.1", got, want)
	}
}
