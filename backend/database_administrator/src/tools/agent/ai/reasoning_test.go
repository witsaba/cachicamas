package ai_test

import (
	"errors"
	"strings"
	"testing"

	"github.com/cachicamas/backend/database_administrator/src/tools/agent/ai"
)

// Compile-time assertion: Reasoning IS the ContentPart. If the wrapper
// pattern is reintroduced, this line fails to compile because the wrapper
// type (not Reasoning) would be the runtime value of ContentPartFromReasoning.
// Per design-resolution obs #2059 and AI-06 spec rev 2 #2057 § A req 3.
//
// The literal Reasoning{} compiles here because Reasoning is an exported
// type with unexported fields — the fields are inaccessible from
// ai_test, but the zero literal is accessible. At runtime the literal
// has ReasoningState(""), which ReasoningState.IsValid() MUST reject
// (covered by TestReasoning_ZeroValueInvalid below).
var _ ai.ContentPart = ai.Reasoning{}

// mustReasoningPart wraps ContentPartFromReasoning for slice-literal
// contexts. Test inputs are all hardcoded valid combinations, so the
// error is guaranteed nil. Prefer binding the error directly in
// production-style tests.
func mustReasoningPart(state ai.ReasoningState, payload string) ai.ContentPart {
	p, _ := ai.ContentPartFromReasoning(state, payload)
	return p
}

// TestMaxReasoningSummaryLength_Value pins MaxReasoningSummaryLength to
// 1 KiB (1024 bytes). Per AI-06 spec § A req 2 — the constant MUST be
// 1024. If this drifts, every Redacted boundary assertion below breaks.
func TestMaxReasoningSummaryLength_Value(t *testing.T) {
	const want = 1 << 10 // 1 KiB = 1024 bytes
	if ai.MaxReasoningSummaryLength != want {
		t.Errorf("MaxReasoningSummaryLength = %d, want %d", ai.MaxReasoningSummaryLength, want)
	}
}

// TestMaxReasoningStreamedLength_Value pins MaxReasoningStreamedLength
// to 1 MiB (1,048,576 bytes). Per AI-06 spec § A req 2 — the constant
// MUST be 1 MiB.
func TestMaxReasoningStreamedLength_Value(t *testing.T) {
	const want = 1 << 20 // 1 MiB = 1,048,576 bytes
	if ai.MaxReasoningStreamedLength != want {
		t.Errorf("MaxReasoningStreamedLength = %d, want %d", ai.MaxReasoningStreamedLength, want)
	}
}

// TestReasoningState_AllConstantsPresent verifies the three canonical
// ReasoningState constants exist and have stable, distinct wire-format
// strings. Per AI-06 spec § A req 1.
func TestReasoningState_AllConstantsPresent(t *testing.T) {
	want := map[ai.ReasoningState]string{
		ai.ReasoningAbsent:   "absent",
		ai.ReasoningRedacted: "redacted",
		ai.ReasoningStreamed: "streamed",
	}
	if got := len(want); got != 3 {
		t.Fatalf("AI-06 spec § A req 1 requires exactly 3 ReasoningState variants, found %d", got)
	}
	seen := make(map[ai.ReasoningState]string, len(want))
	for s, wire := range want {
		if string(s) != wire {
			t.Errorf("ReasoningState constant %q has wire-format %q, want %q", s, string(s), wire)
		}
		if prev, dup := seen[s]; dup {
			t.Errorf("duplicate ReasoningState constant %q (wire %q) shadows %q", s, wire, prev)
		}
		seen[s] = wire
	}
}

// TestReasoningState_String_StableIdentifiers verifies ReasoningState.String()
// returns the same wire-format string as the underlying typed-string value.
// Per AI-06 spec § A req 1 — wire-format stability for AI-11.
func TestReasoningState_String_StableIdentifiers(t *testing.T) {
	cases := []struct {
		s    ai.ReasoningState
		want string
	}{
		{ai.ReasoningAbsent, "absent"},
		{ai.ReasoningRedacted, "redacted"},
		{ai.ReasoningStreamed, "streamed"},
	}
	for _, tc := range cases {
		t.Run(string(tc.s), func(t *testing.T) {
			if got := tc.s.String(); got != tc.want {
				t.Errorf("ReasoningState(%q).String() = %q, want %q", tc.s, got, tc.want)
			}
		})
	}
}

// TestReasoningState_IsValid_AcceptsCanonical verifies ReasoningState.IsValid()
// returns true for the 3 canonical constants. Per AI-06 spec § A req 1
// scenario "IsValid accepts canonical constants".
func TestReasoningState_IsValid_AcceptsCanonical(t *testing.T) {
	canonical := []ai.ReasoningState{
		ai.ReasoningAbsent,
		ai.ReasoningRedacted,
		ai.ReasoningStreamed,
	}
	for _, s := range canonical {
		t.Run(string(s), func(t *testing.T) {
			if !s.IsValid() {
				t.Errorf("canonical ReasoningState %q must be valid (IsValid returned false)", s)
			}
		})
	}
}

// TestReasoningState_IsValid_RejectsZeroValue verifies the zero value
// ReasoningState("") is rejected by IsValid(). Per AI-06 spec § A req 1 —
// zero values cannot silently become valid wire values (AI-04 invariant
// extended to ReasoningState).
func TestReasoningState_IsValid_RejectsZeroValue(t *testing.T) {
	var zero ai.ReasoningState
	if zero.IsValid() {
		t.Error("zero-value ReasoningState(\"\") must be invalid")
	}
}

// TestReasoningState_IsValid_RejectsUnknown verifies IsValid() returns
// false for ReasoningState values outside the 3-variant canonical set.
// Per AI-06 spec § A req 1 scenario "rejects unknowns".
func TestReasoningState_IsValid_RejectsUnknown(t *testing.T) {
	unknown := []ai.ReasoningState{
		"thinking",
		"summary",
		"raw",
		" ABSENT", // leading space
		"absent ", // trailing space
		"Absent",  // mixed case
		"ABSENT",  // all caps
	}
	for _, s := range unknown {
		t.Run(string(s), func(t *testing.T) {
			if s.IsValid() {
				t.Errorf("unknown ReasoningState %q must be invalid", s)
			}
		})
	}
}

// TestContentPartFromReasoning_Absent verifies ReasoningAbsent ignores
// the payload entirely. Per AI-06 spec § A req 2 — Absent is always valid
// regardless of payload length or content.
func TestContentPartFromReasoning_Absent(t *testing.T) {
	cases := []string{
		"",                        // empty
		"hello",                   // short
		strings.Repeat("a", 1024), // 1 KiB (would fail Redacted limit if treated as summary)
		strings.Repeat("a", ai.MaxReasoningStreamedLength),   // 1 MiB
		strings.Repeat("a", ai.MaxReasoningStreamedLength+1), // 1 MiB + 1 byte
		"\n\t\u00A0\u3000", // whitespace-only
	}
	for _, in := range cases {
		t.Run(fmtSummary(in), func(t *testing.T) {
			part, err := ai.ContentPartFromReasoning(ai.ReasoningAbsent, in)
			if err != nil {
				t.Fatalf("ContentPartFromReasoning(Absent, %q) returned error %v, want nil (Absent ignores payload)", in, err)
			}
			if part == nil {
				t.Fatalf("ContentPartFromReasoning(Absent, %q) returned nil ContentPart", in)
			}
			if got := part.Kind(); got != ai.KindReasoning {
				t.Errorf("part.Kind() = %q, want %q", got, ai.KindReasoning)
			}
		})
	}
}

// TestContentPartFromReasoning_Redacted_Boundary verifies ReasoningRedacted
// accepts payloads up to and including MaxReasoningSummaryLength, and
// rejects payloads of MaxReasoningSummaryLength + 1 byte. Per AI-06 spec
// § A req 2 scenario "Redacted enforces boundaries".
func TestContentPartFromReasoning_Redacted_Boundary(t *testing.T) {
	t.Run("exactly max", func(t *testing.T) {
		in := strings.Repeat("a", ai.MaxReasoningSummaryLength)
		part, err := ai.ContentPartFromReasoning(ai.ReasoningRedacted, in)
		if err != nil {
			t.Fatalf("ContentPartFromReasoning(Redacted, %d bytes) returned error %v, want nil (boundary must be accepted)", len(in), err)
		}
		if part == nil {
			t.Fatalf("ContentPartFromReasoning(Redacted, %d bytes) returned nil ContentPart", len(in))
		}
	})
	t.Run("max+1 byte", func(t *testing.T) {
		in := strings.Repeat("a", ai.MaxReasoningSummaryLength+1)
		part, err := ai.ContentPartFromReasoning(ai.ReasoningRedacted, in)
		if !errors.Is(err, ai.ErrReasoningSummaryTooLong) {
			t.Errorf("ContentPartFromReasoning(Redacted, %d bytes) error = %v, want ErrReasoningSummaryTooLong", len(in), err)
		}
		if part != nil {
			t.Errorf("ContentPartFromReasoning(Redacted, %d bytes) returned non-nil ContentPart on error path", len(in))
		}
	})
}

// TestContentPartFromReasoning_Redacted_HappyPath verifies short
// summaries (well under the 1 KiB limit) are accepted. Triangulates
// TestContentPartFromReasoning_Redacted_Boundary — many inputs, same
// expected outcome.
func TestContentPartFromReasoning_Redacted_HappyPath(t *testing.T) {
	cases := []string{
		"a", // single byte
		"the model declined to share its reasoning",       // typical summary
		strings.Repeat("a", 100),                          // 100 B
		strings.Repeat("a", ai.MaxReasoningSummaryLength), // exactly 1 KiB
	}
	for _, in := range cases {
		t.Run(fmtSummary(in), func(t *testing.T) {
			part, err := ai.ContentPartFromReasoning(ai.ReasoningRedacted, in)
			if err != nil {
				t.Fatalf("ContentPartFromReasoning(Redacted, %q) returned error %v, want nil", in, err)
			}
			if part == nil {
				t.Fatalf("ContentPartFromReasoning(Redacted, %q) returned nil ContentPart", in)
			}
		})
	}
}

// TestContentPartFromReasoning_Streamed_Boundary verifies ReasoningStreamed
// accepts payloads up to and including MaxReasoningStreamedLength, and
// rejects payloads of MaxReasoningStreamedLength + 1 byte. Per AI-06 spec
// § A req 2 scenario "Streamed enforces boundaries".
func TestContentPartFromReasoning_Streamed_Boundary(t *testing.T) {
	t.Run("exactly max", func(t *testing.T) {
		in := strings.Repeat("a", ai.MaxReasoningStreamedLength)
		part, err := ai.ContentPartFromReasoning(ai.ReasoningStreamed, in)
		if err != nil {
			t.Fatalf("ContentPartFromReasoning(Streamed, %d bytes) returned error %v, want nil (boundary must be accepted)", len(in), err)
		}
		if part == nil {
			t.Fatalf("ContentPartFromReasoning(Streamed, %d bytes) returned nil ContentPart", len(in))
		}
	})
	t.Run("max+1 byte", func(t *testing.T) {
		in := strings.Repeat("a", ai.MaxReasoningStreamedLength+1)
		part, err := ai.ContentPartFromReasoning(ai.ReasoningStreamed, in)
		if !errors.Is(err, ai.ErrReasoningStreamTooLong) {
			t.Errorf("ContentPartFromReasoning(Streamed, %d bytes) error = %v, want ErrReasoningStreamTooLong", len(in), err)
		}
		if part != nil {
			t.Errorf("ContentPartFromReasoning(Streamed, %d bytes) returned non-nil ContentPart on error path", len(in))
		}
	})
}

// TestContentPartFromReasoning_Streamed_Empty verifies ErrEmptyReasoningStream
// for empty input. Per AI-06 spec § A req 2 scenario "Streamed+empty".
// Streamed requires non-empty content; Absent does not.
func TestContentPartFromReasoning_Streamed_Empty(t *testing.T) {
	part, err := ai.ContentPartFromReasoning(ai.ReasoningStreamed, "")
	if !errors.Is(err, ai.ErrEmptyReasoningStream) {
		t.Errorf("ContentPartFromReasoning(Streamed, \"\") error = %v, want ErrEmptyReasoningStream", err)
	}
	if part != nil {
		t.Errorf("ContentPartFromReasoning(Streamed, \"\") returned non-nil ContentPart on error path")
	}
}

// TestContentPartFromReasoning_Streamed_Whitespace verifies
// ErrWhitespaceReasoningStream for whitespace-only input. Per AI-06 spec
// § A req 2 scenario "Streamed+whitespace". Triangulates with the
// empty-input case — both are rejected, but with distinct sentinels.
func TestContentPartFromReasoning_Streamed_Whitespace(t *testing.T) {
	cases := []string{
		" ",                 // ASCII space
		"\t",                // tab
		"\n",                // newline
		"  \t\n\r",          // mix of ASCII whitespace
		"\u00A0",            // non-breaking space
		"\u3000",            // ideographic space
		" \n\u00A0\t\u3000", // mix of whitespace runes
	}
	for _, in := range cases {
		t.Run(fmtWhitespace(in), func(t *testing.T) {
			part, err := ai.ContentPartFromReasoning(ai.ReasoningStreamed, in)
			if !errors.Is(err, ai.ErrWhitespaceReasoningStream) {
				t.Errorf("ContentPartFromReasoning(Streamed, %q) error = %v, want ErrWhitespaceReasoningStream", in, err)
			}
			if part != nil {
				t.Errorf("ContentPartFromReasoning(Streamed, %q) returned non-nil ContentPart on error path", in)
			}
		})
	}
}

// TestContentPartFromReasoning_UnknownState verifies ErrInvalidReasoningState
// for state values outside the 3-variant canonical set. The state check
// runs FIRST, so payload validation is bypassed. Per AI-06 spec § A req 1
// scenario "rejects unknowns".
func TestContentPartFromReasoning_UnknownState(t *testing.T) {
	cases := []string{
		"thinking",
		"summary",
		"raw",
		"",
	}
	for _, st := range cases {
		t.Run(st, func(t *testing.T) {
			// Use a payload that would otherwise be valid for any state.
			part, err := ai.ContentPartFromReasoning(ai.ReasoningState(st), "valid payload")
			if !errors.Is(err, ai.ErrInvalidReasoningState) {
				t.Errorf("ContentPartFromReasoning(unknown=%q, valid) error = %v, want ErrInvalidReasoningState", st, err)
			}
			if part != nil {
				t.Errorf("ContentPartFromReasoning(unknown=%q, valid) returned non-nil ContentPart on error path", st)
			}
		})
	}
}

// TestReasoning_KindStateText_RoundTrip verifies the accessors
// Kind/State/Text return the constructor inputs for every variant.
// Per AI-06 spec § A req 3 scenario "Layer 2 reads variant + payload via
// public accessors".
func TestReasoning_KindStateText_RoundTrip(t *testing.T) {
	type tc struct {
		state   ai.ReasoningState
		payload string
	}
	cases := []tc{
		{ai.ReasoningAbsent, ""},
		{ai.ReasoningAbsent, "ignored"},
		{ai.ReasoningRedacted, "the model declined to share its reasoning"},
		{ai.ReasoningRedacted, strings.Repeat("r", ai.MaxReasoningSummaryLength)},
		{ai.ReasoningStreamed, "trace-X"},
		{ai.ReasoningStreamed, strings.Repeat("s", ai.MaxReasoningStreamedLength)},
	}
	for _, c := range cases {
		t.Run(string(c.state)+"/"+fmtSummary(c.payload), func(t *testing.T) {
			part, err := ai.ContentPartFromReasoning(c.state, c.payload)
			if err != nil {
				t.Fatalf("ContentPartFromReasoning(%q, %q) returned error %v, want nil", c.state, c.payload, err)
			}
			r, ok := part.(ai.Reasoning)
			if !ok {
				t.Fatalf("ContentPartFromReasoning(%q, %q) returned runtime type %T, want ai.Reasoning (no wrapper allowed per #2059)", c.state, c.payload, part)
			}
			if got := r.Kind(); got != ai.KindReasoning {
				t.Errorf("r.Kind() = %q, want %q", got, ai.KindReasoning)
			}
			if got := r.State(); got != c.state {
				t.Errorf("r.State() = %q, want %q", got, c.state)
			}
			if got := r.Text(); got != c.payload {
				t.Errorf("r.Text() = %q, want %q", got, c.payload)
			}
		})
	}
}

// TestReasoning_Layer2TypeAssertion pins the Layer 2 inspectability
// invariant: part.(ai.Reasoning) MUST succeed at runtime. If a wrapper
// (e.g., reasoningPart) is reintroduced, this assertion returns false
// and the test fails. Per AI-06 spec § A req 3 — this is the core
// acceptance criterion the design integrity proof in #2060 commits to.
//
// The test also asserts that Kind/State/Text do NOT leak any vendor
// metadata, model tags, or provider identifiers: the only strings
// present in the accessors are the canonical constants and the payload
// (which the test fully controls).
func TestReasoning_Layer2TypeAssertion(t *testing.T) {
	part, err := ai.ContentPartFromReasoning(ai.ReasoningStreamed, "trace-X")
	if err != nil {
		t.Fatalf("ContentPartFromReasoning returned error %v, want nil", err)
	}
	if part == nil {
		t.Fatal("ContentPartFromReasoning returned nil ContentPart")
	}
	r, ok := part.(ai.Reasoning)
	if !ok {
		t.Fatalf("part.(ai.Reasoning) = (zero, false); runtime type is %T (wrapper regression — see design-resolution #2059)", part)
	}
	// Pin the accessors on a streamed Reasoning with a controlled payload.
	if got := r.Kind(); got != ai.KindReasoning {
		t.Errorf("r.Kind() = %q, want %q (the only Kind Reasoning reports is KindReasoning)", got, ai.KindReasoning)
	}
	if got := r.State(); got != ai.ReasoningStreamed {
		t.Errorf("r.State() = %q, want %q", got, ai.ReasoningStreamed)
	}
	if got := r.Text(); got != "trace-X" {
		t.Errorf("r.Text() = %q, want %q (no vendor metadata must leak — payload is exactly what the caller passed)", got, "trace-X")
	}
	// Triangulation: pin that the State accessor never returns a string
	// outside the 3-variant canonical set. This catches a hypothetical
	// wrapper that smuggles vendor metadata through State(). Use a
	// payload that is valid for every variant under test (Absent
	// accepts any payload including ""; Redacted/Streamed need a
	// non-whitespace payload <= their limits).
	for _, want := range []ai.ReasoningState{ai.ReasoningAbsent, ai.ReasoningRedacted, ai.ReasoningStreamed} {
		p, err := ai.ContentPartFromReasoning(want, "x")
		if err != nil {
			t.Fatalf("triangulation setup: ContentPartFromReasoning(%q, %q) = %v, want nil", want, "x", err)
		}
		rr, ok := p.(ai.Reasoning)
		if !ok {
			t.Fatalf("triangulation setup: part.(ai.Reasoning) failed for state %q — wrapper regression", want)
		}
		if got := rr.State(); got != want {
			t.Errorf("Layer 2 type assertion for state %q returned %q from State() — possible vendor-metadata leak", want, got)
		}
	}
}

// TestReasoning_ZeroValueInvalid verifies that a literal Reasoning{}
// (constructed outside the sanctioned constructor) yields a
// ReasoningState("") that ReasoningState.IsValid() rejects. The
// constructor is the only path that produces valid Reasoning values.
// Per AI-06 spec § A req 3 zero-value scenario.
func TestReasoning_ZeroValueInvalid(t *testing.T) {
	var zero ai.Reasoning
	if zero.State().IsValid() {
		t.Error("literal Reasoning{} has State() with ReasoningState(\"\") which must be invalid (IsValid returned true)")
	}
	if zero.Text() != "" {
		t.Errorf("literal Reasoning{}.Text() = %q, want \"\" (zero-value Text accessor must return empty)", zero.Text())
	}
	// Pin the discriminator: even the zero value must satisfy ContentPart.
	if got := zero.Kind(); got != ai.KindReasoning {
		t.Errorf("literal Reasoning{}.Kind() = %q, want %q (zero-value Reasoning still satisfies the ContentPart interface)", got, ai.KindReasoning)
	}
}

// TestMessage_Validate_AcceptsReasoningAbsentOnly verifies a Message
// containing a single ReasoningAbsent part passes Validate (no new
// Validate logic). Per AI-06 spec § A req 4 — Message.Validate stays
// structural-only.
func TestMessage_Validate_AcceptsReasoningAbsentOnly(t *testing.T) {
	m := ai.Message{
		Role: ai.RoleAssistant,
		Content: []ai.ContentPart{
			mustReasoningPart(ai.ReasoningAbsent, ""),
		},
	}
	if err := m.Validate(); err != nil {
		t.Errorf("Message{Content: [ReasoningAbsent]}.Validate() = %v, want nil", err)
	}
}

// TestMessage_Validate_AcceptsTextPlusReasoningAbsent verifies a
// Message with both a Text part and a ReasoningAbsent part passes
// Validate. Per AI-06 spec § A req 4 scenario "Reasoning-only and
// Text+Absent messages pass Validate".
func TestMessage_Validate_AcceptsTextPlusReasoningAbsent(t *testing.T) {
	m := ai.Message{
		Role: ai.RoleUser,
		Content: []ai.ContentPart{
			mustPart("hi"),
			mustReasoningPart(ai.ReasoningAbsent, ""),
		},
	}
	if err := m.Validate(); err != nil {
		t.Errorf("Message{Content: [text(hi), ReasoningAbsent]}.Validate() = %v, want nil", err)
	}
}

// TestMessage_Validate_RejectsInterfaceNilWithReasoning verifies
// ErrNilContentPart when one of the Content slots is an interface-nil
// value while the OTHER slot holds a valid Reasoning part. Per AI-06
// spec § A req 3 zero-value scenario — the typed-nil trap is avoided
// because Reasoning is struct-valued.
//
// The test pins the invariant on a Message that ALSO contains a valid
// Reasoning part, so the interface-nil check is exercised across the
// reasoning-aware message path. Renamed from a would-be generic name
// to avoid collision with the existing AI-05 nil-part test.
func TestMessage_Validate_RejectsInterfaceNilWithReasoning(t *testing.T) {
	parts := []ai.ContentPart{
		mustReasoningPart(ai.ReasoningAbsent, ""),
		nil,
	}
	m := ai.Message{Role: ai.RoleUser, Content: parts}
	if err := m.Validate(); !errors.Is(err, ai.ErrNilContentPart) {
		t.Errorf("Message{Content: [ReasoningAbsent, nil]}.Validate() = %v, want ErrNilContentPart", err)
	}
}

// TestErrSentinelsDistinct verifies the five Reasoning sentinels are
// distinct errors (callers can branch on which constraint failed) and
// the prefix follows the package convention "ai: ...".
func TestErrSentinelsDistinct(t *testing.T) {
	seen := map[error]string{
		ai.ErrInvalidReasoningState:     "ErrInvalidReasoningState",
		ai.ErrEmptyReasoningStream:      "ErrEmptyReasoningStream",
		ai.ErrWhitespaceReasoningStream: "ErrWhitespaceReasoningStream",
		ai.ErrReasoningSummaryTooLong:   "ErrReasoningSummaryTooLong",
		ai.ErrReasoningStreamTooLong:    "ErrReasoningStreamTooLong",
	}
	if len(seen) != 5 {
		t.Errorf("expected 5 distinct sentinel errors, found %d", len(seen))
	}
	// Cross-check: distinct sentinels must NOT alias each other via errors.Is.
	pairs := []struct {
		a, b error
	}{
		{ai.ErrInvalidReasoningState, ai.ErrEmptyReasoningStream},
		{ai.ErrInvalidReasoningState, ai.ErrWhitespaceReasoningStream},
		{ai.ErrInvalidReasoningState, ai.ErrReasoningSummaryTooLong},
		{ai.ErrInvalidReasoningState, ai.ErrReasoningStreamTooLong},
		{ai.ErrEmptyReasoningStream, ai.ErrWhitespaceReasoningStream},
		{ai.ErrEmptyReasoningStream, ai.ErrReasoningSummaryTooLong},
		{ai.ErrEmptyReasoningStream, ai.ErrReasoningStreamTooLong},
		{ai.ErrWhitespaceReasoningStream, ai.ErrReasoningSummaryTooLong},
		{ai.ErrWhitespaceReasoningStream, ai.ErrReasoningStreamTooLong},
		{ai.ErrReasoningSummaryTooLong, ai.ErrReasoningStreamTooLong},
	}
	for _, p := range pairs {
		if errors.Is(p.a, p.b) {
			t.Errorf("%v must NOT alias %v via errors.Is", p.a, p.b)
		}
	}
	// Pin the package prefix convention.
	for err, name := range seen {
		if msg := err.Error(); !strings.HasPrefix(msg, "ai: ") {
			t.Errorf("%s.Error() = %q, want prefix %q (package convention)", name, msg, "ai: ")
		}
	}
}

// TestErrSentinels_ExportedAndTyped verifies each Reasoning sentinel
// is non-nil, has a non-empty message, and is compatible with errors.Is.
func TestErrSentinels_ExportedAndTyped(t *testing.T) {
	cases := map[error]string{
		ai.ErrInvalidReasoningState:     "ErrInvalidReasoningState",
		ai.ErrEmptyReasoningStream:      "ErrEmptyReasoningStream",
		ai.ErrWhitespaceReasoningStream: "ErrWhitespaceReasoningStream",
		ai.ErrReasoningSummaryTooLong:   "ErrReasoningSummaryTooLong",
		ai.ErrReasoningStreamTooLong:    "ErrReasoningStreamTooLong",
	}
	for err, name := range cases {
		if err == nil {
			t.Errorf("%s must not be nil", name)
			continue
		}
		if err.Error() == "" {
			t.Errorf("%s.Error() must be non-empty", name)
		}
		if !errors.Is(err, err) {
			t.Errorf("%s must be compatible with errors.Is", name)
		}
	}
}

// fmtSummary returns a short, sanitized label for t.Run. We cannot use
// the raw payload in sub-test names because it may contain newlines,
// tabs, or exceed Go's identifier rules.
func fmtSummary(s string) string {
	if len(s) > 20 {
		return "len=" + itoa(len(s))
	}
	return sanitize(s)
}

// fmtWhitespace mirrors fmtSummary but is used for whitespace-only inputs
// where sanitize would collapse them all to the same key.
func fmtWhitespace(s string) string {
	return "runes=" + itoa(runeCount(s))
}

// sanitize replaces whitespace runes with a printable token so t.Run names
// stay unique and readable.
func sanitize(s string) string {
	r := strings.NewReplacer(
		" ", "_SP_",
		"\t", "_TAB_",
		"\n", "_NL_",
		"\r", "_CR_",
		"\u00A0", "_NBSP_",
		"\u3000", "_IDEO_",
	)
	return r.Replace(s)
}

// itoa is a small strconv.Itoa replacement to keep imports tight.
func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	neg := n < 0
	if neg {
		n = -n
	}
	var buf [20]byte
	i := len(buf)
	for n > 0 {
		i--
		buf[i] = byte('0' + n%10)
		n /= 10
	}
	if neg {
		i--
		buf[i] = '-'
	}
	return string(buf[i:])
}

// runeCount returns the number of Unicode runes in s.
func runeCount(s string) int {
	n := 0
	for range s {
		n++
	}
	return n
}
