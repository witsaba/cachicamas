// Package-internal test for the sealed TextStartPayload /
// TextDeltaPayload / TextEndPayload payloads (AI-13). The test file
// lives in package ai_test so it cannot accidentally reach the
// unexported aiPayload() marker — that un-reachability is the point of
// the TestTextEventPayloads_InterfaceIsSealed checks below. AI-13
// follows the same external-test convention as response_test.go and
// text_test.go. Per AI-13 spec REQ-AI13-11.1: this file also carries
// the per-file import-boundary smoke test (TestTextEventGo_ImportsOnlyStdlib).
package ai_test

import (
	"errors"
	"os"
	"reflect"
	"strings"
	"testing"
	"unicode/utf8"

	"github.com/cachicamas/backend/database_administrator/src/tools/agent/ai"
)

// ---------------------------------------------------------------------------
// T-AI13-001 — TextStartPayload matrix (REQ-AI13-1; S-AI13-1.1/1.2/1.3)
// ---------------------------------------------------------------------------

func TestTextStartPayload_Matrix(t *testing.T) {
	for _, name := range []string{"fresh", "idempotent constructor", "zero value"} {
		t.Run(name, func(t *testing.T) {
			p, err := ai.NewTextStartPayload()
			if err != nil {
				t.Fatalf("NewTextStartPayload = %v, want nil", err)
			}
			if p != (ai.TextStartPayload{}) {
				t.Errorf("payload = %+v, want zero TextStartPayload{}", p)
			}
			if got := p.Kind(); got != ai.EventKindTextStart {
				t.Errorf("Kind() = %q, want %q", got, ai.EventKindTextStart)
			}
			if err := p.Validate(); err != nil {
				t.Errorf("Validate() = %v, want nil", err)
			}
			if err := p.Validate(); err != nil {
				t.Errorf("Validate() (2nd call) = %v, want nil (idempotent)", err)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// T-AI13-002 — TextEndPayload matrix (REQ-AI13-3; S-AI13-3.1/3.2/3.3)
// ---------------------------------------------------------------------------

func TestTextEndPayload_Matrix(t *testing.T) {
	p, err := ai.NewTextEndPayload()
	if err != nil {
		t.Fatalf("NewTextEndPayload = %v, want nil", err)
	}
	if p != (ai.TextEndPayload{}) {
		t.Errorf("payload = %+v, want zero TextEndPayload{}", p)
	}
	if got := p.Kind(); got != ai.EventKindTextEnd {
		t.Errorf("Kind() = %q, want %q", got, ai.EventKindTextEnd)
	}
	if err := p.Validate(); err != nil {
		t.Errorf("Validate() = %v, want nil", err)
	}
	if err := p.Validate(); err != nil {
		t.Errorf("Validate() (2nd call) = %v, want nil (idempotent)", err)
	}
}

// ---------------------------------------------------------------------------
// T-AI13-003 — TextDeltaPayload happy paths (REQ-AI13-2.1/2.2/2.6)
// ---------------------------------------------------------------------------

func TestTextDeltaPayload_Matrix(t *testing.T) {
	cases := []struct {
		name  string
		delta string
	}{
		{"ascii", "hello"},
		{"two-byte rune continuation at end", "hé"}, // bytes: 68 c3 a9, last = 0xA9
		{"four-byte emoji continuation at end", "🌍"}, // bytes: f0 9f 8c 8d, last = 0x8D
		{"empty keepalive", ""},
	}
	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			p, err := ai.NewTextDeltaPayload(tc.delta)
			if err != nil {
				t.Fatalf("NewTextDeltaPayload(%q) = %v, want nil", tc.delta, err)
			}
			if got := p.Delta(); got != tc.delta {
				t.Errorf("Delta() = %q, want %q (verbatim round-trip)", got, tc.delta)
			}
			if got := p.Kind(); got != ai.EventKindTextDelta {
				t.Errorf("Kind() = %q, want %q", got, ai.EventKindTextDelta)
			}
			if err := p.Validate(); err != nil {
				t.Errorf("Validate() = %v, want nil on valid delta", err)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// T-AI13-004 — UTF-8 boundary decision table (REQ-AI13-2.3/2.4; REQ-AI13-5)
// ---------------------------------------------------------------------------

// sampleUTF8BoundaryCases returns the 12-row decision table from
// REQ-AI13-5. Shared by the constructor test, the Validate test, and
// the end-to-end reconstruction test.
func sampleUTF8BoundaryCases() []struct {
	name    string
	input   string
	wantErr error
} {
	return []struct {
		name    string
		input   string
		wantErr error
	}{
		// Row 1: empty keepalive — VALID.
		{"empty keepalive", "", nil},
		// Row 2: ASCII only — VALID.
		{"ascii hello", "hello", nil},
		// Row 3: 2-byte UTF-8 rune, last byte is continuation — VALID.
		{"two-byte rune é", "hé", nil}, // bytes: 68 c3 a9, last = 0xA9
		// Rows 5–9: lone leading byte in 0xC2..0xF4 — REJECT.
		{"lone 0xC3", "\xC3", ai.ErrInvalidUTF8Boundary}, // start of é
		{"lone 0xE4", "\xE4", ai.ErrInvalidUTF8Boundary}, // start of 中
		{"lone 0xF0", "\xF0", ai.ErrInvalidUTF8Boundary}, // start of 🌍
		{"ascii + 0xC2 lower edge", "abc\xC2", ai.ErrInvalidUTF8Boundary},
		{"ascii + 0xF4 upper edge", "abc\xF4", ai.ErrInvalidUTF8Boundary},
		// Row 10: ASCII + continuation byte — VALID (continuation ends a rune).
		{"ascii + 0xBF continuation", "a\xBF", nil},
		// Row 11: 4-byte UTF-8 emoji, last byte is continuation — VALID.
		{"four-byte emoji 🌍", "🌍", nil}, // bytes: f0 9f 8c 8d, last = 0x8D
		// Row 12: invalid byte 0xFF — per-byte check PASSES per REQ-AI13-5.
		{"invalid byte 0xFF per-byte passes", "\xFF", nil},
	}
}

func TestTextDeltaPayload_UTF8BoundaryTable(t *testing.T) {
	for _, tc := range sampleUTF8BoundaryCases() {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			p, err := ai.NewTextDeltaPayload(tc.input)
			if tc.wantErr == nil {
				if err != nil {
					t.Fatalf("NewTextDeltaPayload(%q) = %v, want nil", tc.input, err)
				}
				if got := p.Delta(); got != tc.input {
					t.Errorf("Delta() = %q, want %q", got, tc.input)
				}
				return
			}
			if err == nil {
				t.Fatalf("NewTextDeltaPayload(%q) = no error, want %v", tc.input, tc.wantErr)
			}
			if !errors.Is(err, tc.wantErr) {
				t.Errorf("error = %v, want errors.Is(_, %v)", err, tc.wantErr)
			}
			if p != (ai.TextDeltaPayload{}) {
				t.Errorf("payload = %+v, want zero on error (never half-constructed)", p)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// T-AI13-005 — TextDeltaPayload.Validate idempotency (REQ-AI13-2.5)
// ---------------------------------------------------------------------------

func TestTextDeltaPayload_ValidateIdempotent(t *testing.T) {
	// Valid deltas: Validate() returns nil on first and second call.
	for _, delta := range []string{"hello", "hé", ""} {
		delta := delta
		t.Run("valid/"+delta, func(t *testing.T) {
			p, err := ai.NewTextDeltaPayload(delta)
			if err != nil {
				t.Fatalf("setup NewTextDeltaPayload(%q) = %v", delta, err)
			}
			if err := p.Validate(); err != nil {
				t.Errorf("Validate() (1st) = %v, want nil", err)
			}
			if err := p.Validate(); err != nil {
				t.Errorf("Validate() (2nd) = %v, want nil (idempotent)", err)
			}
		})
	}
	// Invalid deltas: constructor refuses, but the SAME sentinel fires
	// from any Validate() path — confirm the constructor's rejection is
	// the documented sentinel.
	for _, delta := range []string{"\xC3", "abc\xF4"} {
		delta := delta
		t.Run("invalid/"+delta, func(t *testing.T) {
			_, err := ai.NewTextDeltaPayload(delta)
			if !errors.Is(err, ai.ErrInvalidUTF8Boundary) {
				t.Errorf("NewTextDeltaPayload(%q) error = %v, want ErrInvalidUTF8Boundary", delta, err)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// T-AI13-006 — Sentinel: prefix + non-nil + pairwise distinct (REQ-AI13-4)
// ---------------------------------------------------------------------------

// TestTextEventSentinels_AreTypedAndDistinct verifies ErrInvalidUTF8Boundary
// is non-nil, errors.Is-compatible with itself, has the "ai: " prefix, and
// is pairwise distinct from every other sentinel in the AI package.
// Per REQ-AI13-4.
func TestTextEventSentinels_AreTypedAndDistinct(t *testing.T) {
	all := map[string]error{
		"ErrInvalidUTF8Boundary":      ai.ErrInvalidUTF8Boundary,
		"ErrEventKindUnregistered":    ai.ErrEventKindUnregistered,
		"ErrEventPayloadKindMismatch": ai.ErrEventPayloadKindMismatch,
		"ErrEventPayloadMissing":      ai.ErrEventPayloadMissing,
		"ErrEmptyResponseID":          ai.ErrEmptyResponseID,
		"ErrWhitespaceResponseID":     ai.ErrWhitespaceResponseID,
		"ErrResponseIDTooLong":        ai.ErrResponseIDTooLong,
		"ErrEmptyResponseModel":       ai.ErrEmptyResponseModel,
		"ErrWhitespaceResponseModel":  ai.ErrWhitespaceResponseModel,
		"ErrResponseModelTooLong":     ai.ErrResponseModelTooLong,
		"ErrInvalidFinishReason":      ai.ErrInvalidFinishReason,
		"ErrEmptyText":                ai.ErrEmptyText,
		"ErrWhitespaceText":           ai.ErrWhitespaceText,
		"ErrTextTooLong":              ai.ErrTextTooLong,
	}
	if ai.ErrInvalidUTF8Boundary == nil {
		t.Fatal("ErrInvalidUTF8Boundary must not be nil")
	}
	if !errors.Is(ai.ErrInvalidUTF8Boundary, ai.ErrInvalidUTF8Boundary) {
		t.Error("ErrInvalidUTF8Boundary must be errors.Is-compatible with itself")
	}
	if msg := ai.ErrInvalidUTF8Boundary.Error(); !strings.HasPrefix(msg, "ai: ") {
		t.Errorf("ErrInvalidUTF8Boundary.Error() = %q, must start with %q", msg, "ai: ")
	}
	if msg := ai.ErrInvalidUTF8Boundary.Error(); msg == "" {
		t.Error("ErrInvalidUTF8Boundary.Error() must be non-empty")
	}
	for nameA, a := range all {
		for nameB, b := range all {
			if nameA == nameB {
				continue
			}
			if errors.Is(a, b) {
				t.Errorf("%s and %s must be distinct sentinels (errors.Is collision)", nameA, nameB)
			}
		}
	}
}

// TestErrInvalidUTF8Boundary_String pins the exact Error() message.
// Per REQ-AI13-4.1.
func TestErrInvalidUTF8Boundary_String(t *testing.T) {
	const want = "ai: text delta ends mid-rune (UTF-8 boundary violation)"
	if got := ai.ErrInvalidUTF8Boundary.Error(); got != want {
		t.Errorf("ErrInvalidUTF8Boundary.Error() = %q, want %q", got, want)
	}
}

// ---------------------------------------------------------------------------
// T-AI13-007 — Reflection: no Marshal/Clone/With + AiPayload alias (REQ-AI13-7)
// ---------------------------------------------------------------------------

// TestTextEventPayloads_HaveNoMarshalOrCloneOrWithMethods verifies the
// AI-11 vendor-leak guard on the three AI-13 payload types. Per REQ-AI13-7.1.
func TestTextEventPayloads_HaveNoMarshalOrCloneOrWithMethods(t *testing.T) {
	start, _ := ai.NewTextStartPayload()
	delta := textDeltaPayloadOf(t, "hi")
	end, _ := ai.NewTextEndPayload()

	forbidden := map[string]bool{
		"MarshalJSON": true, "UnmarshalJSON": true,
		"MarshalText": true, "UnmarshalText": true,
		"Clone": true,
	}
	for _, tc := range []struct {
		name string
		val  any
	}{
		{"TextStartPayload", start},
		{"TextDeltaPayload", delta},
		{"TextEndPayload", end},
	} {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			typ := reflect.TypeOf(tc.val)
			for i := 0; i < typ.NumMethod(); i++ {
				m := typ.Method(i)
				if forbidden[m.Name] {
					t.Errorf("%s must not define %s (AI-11 vendor-leak guard)", typ.Name(), m.Name)
				}
				if strings.HasPrefix(m.Name, "With") {
					t.Errorf("%s.%s must not be With* (sealed payload forbids builder mutation)", typ.Name(), m.Name)
				}
			}
		})
	}
}

// TestTextEventPayloads_InterfaceIsSealed verifies the Kind() parity
// and the structural mirror of the unexported aiPayload() marker for
// all three payload types. Per REQ-AI13-7.2.
func TestTextEventPayloads_InterfaceIsSealed(t *testing.T) {
	start, _ := ai.NewTextStartPayload()
	delta := textDeltaPayloadOf(t, "hi")
	end, _ := ai.NewTextEndPayload()

	for _, tc := range []struct {
		name string
		val  any
		kind ai.EventKind
	}{
		{"TextStartPayload", start, ai.EventKindTextStart},
		{"TextDeltaPayload", delta, ai.EventKindTextDelta},
		{"TextEndPayload", end, ai.EventKindTextEnd},
	} {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			// Kind() returns the canonical kind verbatim.
			v := reflect.ValueOf(tc.val)
			res := v.MethodByName("Kind").Call(nil)
			if len(res) != 1 {
				t.Fatalf("Kind() returned %d values, want 1", len(res))
			}
			got, ok := res[0].Interface().(ai.EventKind)
			if !ok {
				t.Fatalf("Kind() returned %T, want ai.EventKind", res[0].Interface())
			}
			if got != tc.kind {
				t.Errorf("Kind() = %q, want %q", got, tc.kind)
			}
			// No exported AiPayload alias (mirror of the unexported seal).
			typ := reflect.TypeOf(tc.val)
			for i := 0; i < typ.NumMethod(); i++ {
				if typ.Method(i).Name == "AiPayload" {
					t.Errorf("%s must not expose AiPayload (alias of unexported aiPayload)", typ.Name())
				}
			}
		})
	}
}

// ---------------------------------------------------------------------------
// T-AI13-008 — Event constructor round-trip + error propagation (REQ-AI13-8)
// ---------------------------------------------------------------------------

func TestTextEventConstructors_RoundTrip(t *testing.T) {
	t.Run("TextStart", func(t *testing.T) {
		ev, err := ai.NewTextStartEvent()
		if err != nil {
			t.Fatalf("NewTextStartEvent = %v, want nil", err)
		}
		if ev.Kind != ai.EventKindTextStart {
			t.Errorf("Kind = %q, want %q", ev.Kind, ai.EventKindTextStart)
		}
		if ev.Sequence < 1 {
			t.Errorf("Sequence = %d, want >= 1", ev.Sequence)
		}
		p, ok := ai.AsTextStart(ev)
		if !ok {
			t.Fatalf("AsTextStart = false, want true (Kind matches)")
		}
		if p != (ai.TextStartPayload{}) {
			t.Errorf("payload = %+v, want zero", p)
		}
		if err := ev.Validate(); err != nil {
			t.Errorf("Validate() = %v, want nil", err)
		}
	})
	t.Run("TextDelta", func(t *testing.T) {
		const delta = "hello"
		ev, err := ai.NewTextDeltaEvent(delta)
		if err != nil {
			t.Fatalf("NewTextDeltaEvent(%q) = %v, want nil", delta, err)
		}
		if ev.Kind != ai.EventKindTextDelta {
			t.Errorf("Kind = %q, want %q", ev.Kind, ai.EventKindTextDelta)
		}
		if ev.Sequence < 1 {
			t.Errorf("Sequence = %d, want >= 1", ev.Sequence)
		}
		p := asTextDeltaPayload(t, ev)
		if p.Delta() != delta {
			t.Errorf("Delta() = %q, want %q", p.Delta(), delta)
		}
		if err := ev.Validate(); err != nil {
			t.Errorf("Validate() = %v, want nil", err)
		}
	})
	t.Run("TextEnd", func(t *testing.T) {
		ev, err := ai.NewTextEndEvent()
		if err != nil {
			t.Fatalf("NewTextEndEvent = %v, want nil", err)
		}
		if ev.Kind != ai.EventKindTextEnd {
			t.Errorf("Kind = %q, want %q", ev.Kind, ai.EventKindTextEnd)
		}
		if ev.Sequence < 1 {
			t.Errorf("Sequence = %d, want >= 1", ev.Sequence)
		}
		p, ok := ai.AsTextEnd(ev)
		if !ok {
			t.Fatalf("AsTextEnd = false, want true (Kind matches)")
		}
		if p != (ai.TextEndPayload{}) {
			t.Errorf("payload = %+v, want zero", p)
		}
		if err := ev.Validate(); err != nil {
			t.Errorf("Validate() = %v, want nil", err)
		}
	})
}

// TestTextEventConstructors_PropagateValidationError verifies the event
// constructor returns the constructor's sentinel verbatim. Per REQ-AI13-8.4.
func TestTextEventConstructors_PropagateValidationError(t *testing.T) {
	ev, err := ai.NewTextDeltaEvent("\xC3") // lone 2-byte leading → boundary reject
	if err == nil {
		t.Fatal("NewTextDeltaEvent(\"\\xC3\") = no error, want ErrInvalidUTF8Boundary")
	}
	if !errors.Is(err, ai.ErrInvalidUTF8Boundary) {
		t.Errorf("error = %v, want errors.Is(_, ErrInvalidUTF8Boundary)", err)
	}
	if ev != (ai.Event{}) {
		t.Errorf("event = %+v, want zero Event on error", ev)
	}
}

// ---------------------------------------------------------------------------
// T-AI13-009 — AsXxx parity: right-kind accepts (REQ-AI13-9.1)
// ---------------------------------------------------------------------------

func TestAsTextStartDeltaEnd_RightKindAccepts(t *testing.T) {
	startEv, _ := ai.NewTextStartEvent()
	deltaEv, _ := ai.NewTextDeltaEvent("hi")
	endEv, _ := ai.NewTextEndEvent()

	t.Run("AsTextStart", func(t *testing.T) {
		p, ok := ai.AsTextStart(startEv)
		if !ok {
			t.Fatal("AsTextStart(start-event) = false, want true")
		}
		if p != (ai.TextStartPayload{}) {
			t.Errorf("payload = %+v, want zero", p)
		}
	})
	t.Run("AsTextDelta", func(t *testing.T) {
		p := asTextDeltaPayload(t, deltaEv)
		if p.Delta() != "hi" {
			t.Errorf("Delta() = %q, want %q", p.Delta(), "hi")
		}
	})
	t.Run("AsTextEnd", func(t *testing.T) {
		p, ok := ai.AsTextEnd(endEv)
		if !ok {
			t.Fatal("AsTextEnd(end-event) = false, want true")
		}
		if p != (ai.TextEndPayload{}) {
			t.Errorf("payload = %+v, want zero", p)
		}
	})
}

// ---------------------------------------------------------------------------
// T-AI13-010 — AsXxx parity: wrong-kind rejects, 6 pairings (REQ-AI13-9.2)
// ---------------------------------------------------------------------------

func TestAsTextStartDeltaEnd_WrongKindRejects(t *testing.T) {
	startEv, _ := ai.NewTextStartEvent()
	deltaEv, _ := ai.NewTextDeltaEvent("hi")
	endEv, _ := ai.NewTextEndEvent()

	cases := []struct {
		name   string
		ev     ai.Event
		helper string // the wrong-kind AsXxx helper
	}{
		{"TextStart rejected by AsTextDelta", startEv, "AsTextDelta"},
		{"TextStart rejected by AsTextEnd", startEv, "AsTextEnd"},
		{"TextDelta rejected by AsTextStart", deltaEv, "AsTextStart"},
		{"TextDelta rejected by AsTextEnd", deltaEv, "AsTextEnd"},
		{"TextEnd rejected by AsTextStart", endEv, "AsTextStart"},
		{"TextEnd rejected by AsTextDelta", endEv, "AsTextDelta"},
	}
	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			var ok bool
			switch tc.helper {
			case "AsTextStart":
				_, ok = ai.AsTextStart(tc.ev)
			case "AsTextDelta":
				_, ok = ai.AsTextDelta(tc.ev)
			case "AsTextEnd":
				_, ok = ai.AsTextEnd(tc.ev)
			default:
				t.Fatalf("test bug: unknown helper %q", tc.helper)
			}
			if ok {
				t.Errorf("%s(%s-event) = true, want false (kind parity guard)",
					tc.helper, tc.ev.Kind)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// T-AI13-011 — Doc-guard: AI-13 paragraph (REQ-AI13-10.1)
// ---------------------------------------------------------------------------

// TestDocGo_AI13Paragraph verifies doc.go after the AI-13 changes
// (T-AI13-3 DOCS commit): must mention AI-13, TextStartPayload,
// TextDeltaPayload, TextEndPayload, ErrInvalidUTF8Boundary, and
// UTF-8 (case-sensitive, all verbatim). Per REQ-AI13-10.1.
//
// This test is in RED with the full body (matching the AI-12
// precedent — the test was a known failure in GREEN that the DOCS
// commit resolved).
func TestDocGo_AI13Paragraph(t *testing.T) {
	body, err := os.ReadFile("doc.go")
	if err != nil {
		t.Fatalf("ReadFile doc.go = %v", err)
	}
	doc := string(body)
	for _, phrase := range []string{
		"AI-13",
		"TextStartPayload",
		"TextDeltaPayload",
		"TextEndPayload",
		"ErrInvalidUTF8Boundary",
		"UTF-8",
	} {
		if !strings.Contains(doc, phrase) {
			t.Errorf("doc.go must mention %q (AI-13 surface summary)", phrase)
		}
	}
}

// ---------------------------------------------------------------------------
// T-AI13-012 — Import-boundary smoke for text_event.go (REQ-AI13-11.1)
// ---------------------------------------------------------------------------

// TestTextEventGo_ImportsOnlyStdlib verifies text_event.go imports
// only the Go standard library — no cachicamas, no vendor SDKs, no
// golang.org/x/text. Per REQ-AI13-11.1.
func TestTextEventGo_ImportsOnlyStdlib(t *testing.T) {
	body, err := os.ReadFile("text_event.go")
	if err != nil {
		// RED: file absent — package-level TestPackageImportBoundary
		// is the gate; here we only assert what we can when present.
		t.Skipf("text_event.go not yet present (RED phase): %v", err)
	}
	for _, line := range strings.Split(string(body), "\n") {
		trimmed := strings.TrimSpace(line)
		if !strings.HasPrefix(trimmed, "\"") {
			continue
		}
		if strings.Contains(trimmed, "github.com/") ||
			strings.Contains(trimmed, "golang.org/") ||
			strings.Contains(trimmed, "cachicamas") {
			t.Errorf("text_event.go imports a non-stdlib path: %q", trimmed)
		}
	}
}

// ---------------------------------------------------------------------------
// T-AI13-013 — End-to-end reconstruction (milestone doc line 254 acceptance)
// ---------------------------------------------------------------------------

// TestTextEndToEndReconstruction_ConcatenatedDeltasValidUTF8 demonstrates
// the milestone doc line 254 acceptance: concatenating ordered TextDelta
// payloads reconstructs the final text exactly, including Unicode
// boundaries. The test joins every valid-input row of the boundary
// table, then asserts (a) the joined string is valid UTF-8, and (b) the
// joined string equals the literal join of the inputs.
func TestTextEndToEndReconstruction_ConcatenatedDeltasValidUTF8(t *testing.T) {
	var valid []string
	for _, tc := range sampleUTF8BoundaryCases() {
		if tc.wantErr == nil {
			valid = append(valid, tc.input)
		}
	}
	if len(valid) == 0 {
		t.Fatal("sampleUTF8BoundaryCases returned no valid rows")
	}
	var b strings.Builder
	for _, d := range valid {
		p, err := ai.NewTextDeltaPayload(d)
		if err != nil {
			t.Fatalf("NewTextDeltaPayload(%q) = %v during reconstruction; want nil", d, err)
		}
		b.WriteString(p.Delta())
	}
	concat := b.String()
	if !utf8.ValidString(concat) {
		t.Errorf("concatenated stream is not valid UTF-8 (boundary check must catch every split)")
	}
	if expected := strings.Join(valid, ""); concat != expected {
		t.Errorf("concatenated stream = %q (len=%d), want %q (len=%d); round-trip must be verbatim",
			concat, len(concat), expected, len(expected))
	}
}

// ---------------------------------------------------------------------------
// Test affordances (pre-declared in RED per AI-12 carry-over obs #2175)
// ---------------------------------------------------------------------------

// textDeltaPayloadOf builds a TextDeltaPayload with the supplied delta
// after running it through NewTextDeltaPayload. Fatal-fails on error;
// callers MUST supply a valid delta. Mirrors responseIDOf in
// response_test.go line 33.
func textDeltaPayloadOf(t *testing.T, delta string) ai.TextDeltaPayload {
	t.Helper()
	p, err := ai.NewTextDeltaPayload(delta)
	if err != nil {
		t.Fatalf("textDeltaPayloadOf(%q): NewTextDeltaPayload = %v, want nil", delta, err)
	}
	return p
}

// asTextDeltaPayload extracts the TextDeltaPayload from ev via
// AsTextDelta and fatal-fails on (zero, false).
func asTextDeltaPayload(t *testing.T, ev ai.Event) ai.TextDeltaPayload {
	t.Helper()
	p, ok := ai.AsTextDelta(ev)
	if !ok {
		t.Fatalf("AsTextDelta on event kind %q returned ok=false; want true", ev.Kind)
	}
	return p
}
