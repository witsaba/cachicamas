// Tests for AI-07 — reasoning content and its round-trip token.
//
// The package under test is imported by its module path from the external test
// package ai_test, so every assertion below is written against exactly the
// surface a consumer in another package sees. That matters more here than it
// did for text: doc 0001 § 3.1 records that the retired Layer 1's reasoning
// type was the one that broke the content-part pattern, and the cheapest way to
// notice a second strategy is to be unable to write the first one's loop.

package ai_test

import (
	"bytes"
	"errors"
	"strings"
	"testing"
	"unicode/utf8"

	"github.com/cachicamas/backend/agent/src/ai"
)

// AI-07.1 item 1 — a reasoning part is constructed and read back through the
// AI-06.1 strategy, exactly like text: no second strategy, no special case.
//
// The body below is the loop AI-26 will write, with one identifier changed from
// the text case. If reasoning ever needs a different loop, this test is where
// that shows up.
func TestReasoning_ConstructedAndReadBack_UsesTheContentPartStrategy(t *testing.T) {
	t.Parallel()

	const thought = "The user asked for a sum, so I will add the two numbers."

	part, err := ai.NewReasoning(thought, nil)
	if err != nil {
		t.Fatalf("ai.NewReasoning returned %v, want no failure", err)
	}
	if got := part.Kind(); got != ai.PartKindReasoning {
		t.Fatalf("part.Kind() = %v, want %v", got, ai.PartKindReasoning)
	}

	msg, err := ai.NewMessage(ai.RoleAssistant, part)
	if err != nil {
		t.Fatalf("ai.NewMessage returned %v, want no failure", err)
	}

	content := msg.Content()
	if len(content) != 1 {
		t.Fatalf("msg.Content() returned %d elements, want 1", len(content))
	}

	switch got := content[0]; got.Kind() {
	case ai.PartKindReasoning:
		reasoning, ok := got.Reasoning()
		if !ok {
			t.Fatalf("part.Reasoning() reported no reasoning on a part of kind %v", got.Kind())
		}
		if reasoning.Text() != thought {
			t.Errorf("reasoning.Text() = %q, want %q", reasoning.Text(), thought)
		}
	default:
		t.Fatalf("the part read back reports kind %v, want %v", got.Kind(), ai.PartKindReasoning)
	}
}

// AI-07.1 item 2 — reasoning and text are structurally distinct.
//
// V-REQ-09's last clause is the requirement: "distinct from text content at
// every layer — reasoning is never rendered as, merged into, or substituted for
// text." The two are built from the identical string on purpose, so nothing
// below can pass because the payloads happen to differ.
func TestReasoning_AndText_AreStructurallyDistinct(t *testing.T) {
	t.Parallel()

	const shared = "the same characters, in two different kinds of part"

	reasoningPart, err := ai.NewReasoning(shared, nil)
	if err != nil {
		t.Fatalf("ai.NewReasoning returned %v, want no failure", err)
	}
	textPart, err := ai.NewText(shared)
	if err != nil {
		t.Fatalf("ai.NewText returned %v, want no failure", err)
	}

	t.Run("no accessor yields reasoning content as text", func(t *testing.T) {
		t.Parallel()

		text, ok := reasoningPart.Text()
		if ok {
			t.Errorf("reasoningPart.Text() reported text content on a reasoning part")
		}
		if text != "" {
			t.Errorf("reasoningPart.Text() = %q, want the zero value rather than the reasoning", text)
		}
	})

	t.Run("no accessor yields text content as reasoning", func(t *testing.T) {
		t.Parallel()

		reasoning, ok := textPart.Reasoning()
		if ok {
			t.Errorf("textPart.Reasoning() reported reasoning on a text part")
		}
		if reasoning.Text() != "" {
			t.Errorf("textPart.Reasoning() yielded %q, want the zero payload", reasoning.Text())
		}
	})

	t.Run("a part that was never constructed yields neither", func(t *testing.T) {
		t.Parallel()

		var unconstructed ai.Part

		if _, ok := unconstructed.Reasoning(); ok {
			t.Errorf("the zero Part reported reasoning")
		}
		if _, ok := unconstructed.Text(); ok {
			t.Errorf("the zero Part reported text content")
		}
	})

	t.Run("a consumer switching on kind cannot conflate them", func(t *testing.T) {
		t.Parallel()

		if reasoningPart.Kind() == textPart.Kind() {
			t.Fatalf("both parts report kind %v, so a switch reaches one branch for two kinds", textPart.Kind())
		}

		visited := make(map[ai.PartKind]int, 2)
		for _, part := range []ai.Part{reasoningPart, textPart} {
			switch part.Kind() {
			case ai.PartKindReasoning:
				if _, ok := part.Reasoning(); !ok {
					t.Errorf("the reasoning branch was reached by a part carrying no reasoning")
				}
				visited[ai.PartKindReasoning]++
			case ai.PartKindText:
				if _, ok := part.Text(); !ok {
					t.Errorf("the text branch was reached by a part carrying no text")
				}
				visited[ai.PartKindText]++
			default:
				t.Errorf("a part reached no branch, reporting kind %v", part.Kind())
			}
		}

		if visited[ai.PartKindReasoning] != 1 || visited[ai.PartKindText] != 1 {
			t.Errorf("the switch visited %v, want each branch exactly once", visited)
		}
	})
}

// AI-07.1 item 3 — the reasoning state vocabulary is closed, and each state is
// constructible.
//
// V-REQ-10 exists "so that 'no reasoning text' is a recorded state rather than
// an empty string", which is only true if the state that records it can be
// built. Three members: with text, redacted, and token-only — the last being
// both V-REQ-10's signature-only shape and its "a provider that emitted no
// reasoning text at all", which doc 0002's AI-07.4 item 2 names as one shape.
func TestReasoningStates_Vocabulary_IsClosedStableAndConstructible(t *testing.T) {
	t.Parallel()

	token := []byte("an opaque provider signature")

	t.Run("each state is constructible and reports itself", func(t *testing.T) {
		t.Parallel()

		cases := []struct {
			name  string
			build func() (ai.Part, error)
			want  ai.ReasoningState
		}{
			{
				name:  "with text",
				build: func() (ai.Part, error) { return ai.NewReasoning("I will add the two numbers.", token) },
				want:  ai.ReasoningStateText,
			},
			{
				name:  "redacted",
				build: func() (ai.Part, error) { return ai.NewRedactedReasoning(token) },
				want:  ai.ReasoningStateRedacted,
			},
			{
				name:  "token-only",
				build: func() (ai.Part, error) { return ai.NewReasoning("", token) },
				want:  ai.ReasoningStateTokenOnly,
			},
		}

		for _, tc := range cases {
			t.Run(tc.name, func(t *testing.T) {
				t.Parallel()

				part, err := tc.build()
				if err != nil {
					t.Fatalf("construction returned %v, want no failure", err)
				}
				reasoning, ok := part.Reasoning()
				if !ok {
					t.Fatalf("part.Reasoning() reported no reasoning on a part of kind %v", part.Kind())
				}
				if got := reasoning.State(); got != tc.want {
					t.Errorf("reasoning.State() = %v, want %v", got, tc.want)
				}
				if !containsReasoningState(ai.ReasoningStates(), reasoning.State()) {
					t.Errorf("reasoning.State() = %v, which is not a member of the vocabulary", reasoning.State())
				}
			})
		}
	})

	t.Run("the vocabulary is closed, stable and not rewritable by a consumer", func(t *testing.T) {
		t.Parallel()

		states := ai.ReasoningStates()
		if len(states) != 3 {
			t.Fatalf("ai.ReasoningStates() enumerates %d members (%v), want 3", len(states), states)
		}

		want := []ai.ReasoningState{ai.ReasoningStateText, ai.ReasoningStateRedacted, ai.ReasoningStateTokenOnly}
		for i, state := range want {
			if states[i] != state {
				t.Errorf("ai.ReasoningStates()[%d] = %v, want %v — the order is the declaration order", i, states[i], state)
			}
		}

		for i := range states {
			states[i] = 0
		}
		if again := ai.ReasoningStates(); again[0] != ai.ReasoningStateText || len(again) != 3 {
			t.Errorf("ai.ReasoningStates() = %v after a consumer rewrote the slice it received, want the vocabulary unchanged", again)
		}
	})

	t.Run("the zero value is not a member and renders as unset", func(t *testing.T) {
		t.Parallel()

		var unset ai.ReasoningState

		if containsReasoningState(ai.ReasoningStates(), unset) {
			t.Errorf("the zero ReasoningState is a member of the vocabulary, want a non-member")
		}
		if got := unset.String(); got != "unset" {
			t.Errorf("ReasoningState(0).String() = %q, want %q", got, "unset")
		}
	})

	t.Run("a member renders as its registered name and a non-member identifiably", func(t *testing.T) {
		t.Parallel()

		names := map[ai.ReasoningState]string{
			ai.ReasoningStateText:      "text",
			ai.ReasoningStateRedacted:  "redacted",
			ai.ReasoningStateTokenOnly: "token-only",
		}
		for state, want := range names {
			if got := state.String(); got != want {
				t.Errorf("%d.String() = %q, want %q", uint8(state), got, want)
			}
		}

		wild := ai.ReasoningState(200)
		if got := wild.String(); got != "reasoningstate(200)" {
			t.Errorf("ReasoningState(200).String() = %q, want %q", got, "reasoningstate(200)")
		}
	})
}

// AI-07.2 item 1 — a reasoning part carries an opaque token alongside its state
// and its text, and an absent token is distinguishable from an empty one.
//
// This is the distinction doc 0002 says "a naive design collapses and an
// adapter cannot recover", and it is load-bearing on the wire rather than
// pedantic: a provider that returned an empty token and a provider that
// returned no token are two facts, and the return trip must reproduce the one
// that happened. The naive design is one byte slice where nil and zero-length
// are both "nothing to see"; this package stores the presence separately, so
// nothing that touches the bytes can lose it.
func TestReasoning_AbsentToken_IsDistinguishableFromAnEmptyToken(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name        string
		build       func() (ai.Part, error)
		wantPresent bool
		wantLen     int
	}{
		{
			name:        "no token at all",
			build:       func() (ai.Part, error) { return ai.NewReasoning("I will add the two numbers.", nil) },
			wantPresent: false,
		},
		{
			name:        "a token that is present and zero bytes long",
			build:       func() (ai.Part, error) { return ai.NewReasoning("I will add the two numbers.", []byte{}) },
			wantPresent: true,
			wantLen:     0,
		},
		{
			name:        "a token with bytes in it",
			build:       func() (ai.Part, error) { return ai.NewReasoning("I will add the two numbers.", []byte("sig")) },
			wantPresent: true,
			wantLen:     3,
		},
		{
			name:        "a redacted part whose payload is present and zero bytes long",
			build:       func() (ai.Part, error) { return ai.NewRedactedReasoning([]byte{}) },
			wantPresent: true,
			wantLen:     0,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			part, err := tc.build()
			if err != nil {
				t.Fatalf("construction returned %v, want no failure", err)
			}

			// The distinction must survive the trip a message puts it through,
			// so it is asserted at construction and again after a round trip.
			msg, err := ai.NewMessage(ai.RoleAssistant, part)
			if err != nil {
				t.Fatalf("ai.NewMessage returned %v, want no failure", err)
			}

			for _, where := range []struct {
				what string
				part ai.Part
			}{{"as constructed", part}, {"read back out of a message", msg.Content()[0]}} {
				reasoning, ok := where.part.Reasoning()
				if !ok {
					t.Fatalf("%s: part.Reasoning() reported no reasoning", where.what)
				}

				token, present := reasoning.Token()
				if present != tc.wantPresent {
					t.Errorf("%s: reasoning.Token() reported present = %t, want %t", where.what, present, tc.wantPresent)
				}
				if !tc.wantPresent {
					continue
				}
				if len(token) != tc.wantLen {
					t.Errorf("%s: reasoning.Token() returned %d bytes, want %d", where.what, len(token), tc.wantLen)
				}
			}
		})
	}

	t.Run("a part with neither text nor token carries nothing and is rejected", func(t *testing.T) {
		t.Parallel()

		for _, tc := range []struct {
			name  string
			build func() (ai.Part, error)
		}{
			{"no text and no token", func() (ai.Part, error) { return ai.NewReasoning("", nil) }},
			{"redacted with no payload to replay", func() (ai.Part, error) { return ai.NewRedactedReasoning(nil) }},
		} {
			t.Run(tc.name, func(t *testing.T) {
				t.Parallel()

				part, err := tc.build()
				if err == nil {
					t.Fatalf("construction returned no failure, want one")
				}
				if !errors.Is(err, ai.ErrEmpty) {
					t.Errorf("errors.Is(err, ai.ErrEmpty) = false, want true (err = %v)", err)
				}
				var violation *ai.Violation
				if !errors.As(err, &violation) {
					t.Fatalf("errors.As(err, *ai.Violation) = false, want true")
				}
				if got := violation.Path().String(); got != "token" {
					t.Errorf("violation.Path() = %q, want %q", got, "token")
				}
				if part != (ai.Part{}) {
					t.Errorf("a rejected construction returned a non-zero part, want the zero Part")
				}
			})
		}
	})
}

// opaqueTokens are the byte classes doc 0002's AI-07.2 item 2 names, plus the
// ones an implementation is most likely to be tempted by.
//
// Every entry is built from escapes rather than from literal bytes in the
// source: a literal bidi control character in a Go file is the Trojan Source
// hazard staticcheck's ST1018 exists to catch, and a token fixture is exactly
// where one would hide.
func opaqueTokens() []struct {
	name  string
	token []byte
} {
	everyByte := make([]byte, 256)
	for i := range everyByte {
		everyByte[i] = byte(i)
	}

	return []struct {
		name  string
		token []byte
	}{
		{"every byte value from 0x00 to 0xff", everyByte},
		{"not well-formed UTF-8", []byte{0xff, 0xfe, 0x80, 0x81, 0xc3, 0x28}},
		{"a lone UTF-16 surrogate half, encoded as bytes", []byte{0xed, 0xa0, 0x80}},
		{"not valid JSON", []byte("{\"signature\": \x00 unterminated")},
		{"not printable", []byte{0x00, 0x01, 0x02, 0x1b, 0x7f}},
		{"an embedded zero byte between printable bytes", []byte("head\x00tail")},
		{"bidirectional control characters, built from escapes", []byte("start\u202Ereversed\u202Cend")},
		{"something that looks like base64 and must not be decoded", []byte("eyJhbGciOiJIUzI1NiJ9====")},
		{"something that looks like a rendered violation", []byte("content[0].token: required value is empty")},
		{"a single zero byte", []byte{0x00}},
	}
}

// AI-07.2 item 2 — nothing in this package interprets, validates, normalizes or
// length-caps the token beyond a documented sanity bound.
//
// V-REQ-11 is unambiguous: the token is stored and returned "byte-identically"
// and never parsed, reformatted, re-encoded or interpreted. Every temptation an
// implementation has here — validate the encoding, decode what looks like
// base64, trim what looks like padding, cut it to a sensible length — produces
// a byte a provider will not accept back, and the resulting failure appears on
// the second turn of a tool-using conversation.
func TestReasoning_OpaqueToken_IsNeitherInterpretedNorNormalized(t *testing.T) {
	t.Parallel()

	t.Run("every byte class survives construction and readback unaltered", func(t *testing.T) {
		t.Parallel()

		for _, tc := range opaqueTokens() {
			t.Run(tc.name, func(t *testing.T) {
				t.Parallel()

				want := bytes.Clone(tc.token)

				part, err := ai.NewReasoning("I will add the two numbers.", tc.token)
				if err != nil {
					t.Fatalf("ai.NewReasoning returned %v, want no failure — nothing validates a token", err)
				}
				reasoning, ok := part.Reasoning()
				if !ok {
					t.Fatalf("part.Reasoning() reported no reasoning")
				}

				got, present := reasoning.Token()
				if !present {
					t.Fatalf("reasoning.Token() reported no token, want one")
				}
				if len(got) != len(want) {
					t.Fatalf("reasoning.Token() returned %d bytes, want %d — a length change is a normalization", len(got), len(want))
				}
				if !bytes.Equal(got, want) {
					t.Errorf("reasoning.Token() = %x, want %x", got, want)
				}
			})
		}
	})

	t.Run("a token that is not well-formed UTF-8 is stored, not repaired", func(t *testing.T) {
		t.Parallel()

		malformed := []byte{0xff, 0xfe, 0x80}
		if utf8.Valid(malformed) {
			t.Fatalf("the fixture is well-formed UTF-8, so this test would prove nothing")
		}

		part, err := ai.NewReasoning("", malformed)
		if err != nil {
			t.Fatalf("ai.NewReasoning returned %v, want no failure", err)
		}
		reasoning, _ := part.Reasoning()
		got, _ := reasoning.Token()

		if utf8.Valid(got) {
			t.Errorf("reasoning.Token() came back well-formed UTF-8, so something repaired it")
		}
		if !bytes.Equal(got, malformed) {
			t.Errorf("reasoning.Token() = %x, want %x — no replacement character, no repair", got, malformed)
		}
	})

	t.Run("the documented sanity bound is exact and exported", func(t *testing.T) {
		t.Parallel()

		if ai.MaxReasoningTokenLen <= 0 {
			t.Fatalf("ai.MaxReasoningTokenLen = %d, want a positive bound a caller can check against", ai.MaxReasoningTokenLen)
		}

		atBound := bytes.Repeat([]byte{0xab}, ai.MaxReasoningTokenLen)
		if _, err := ai.NewReasoning("", atBound); err != nil {
			t.Errorf("a token exactly at ai.MaxReasoningTokenLen returned %v, want no failure", err)
		}

		const sentinel = "SENTINEL-TOKEN-BYTES"
		overBound := append(bytes.Repeat([]byte{0xab}, ai.MaxReasoningTokenLen+1-len(sentinel)), sentinel...)

		part, err := ai.NewReasoning("", overBound)
		if err == nil {
			t.Fatalf("a token one byte over the bound returned no failure, want one")
		}
		if !errors.Is(err, ai.ErrOutOfRange) {
			t.Errorf("errors.Is(err, ai.ErrOutOfRange) = false, want true (err = %v)", err)
		}
		var violation *ai.Violation
		if !errors.As(err, &violation) {
			t.Fatalf("errors.As(err, *ai.Violation) = false, want true")
		}
		if got := violation.Path().String(); got != "token" {
			t.Errorf("violation.Path() = %q, want %q", got, "token")
		}
		if part != (ai.Part{}) {
			t.Errorf("a rejected construction returned a non-zero part, want the zero Part")
		}
		if strings.Contains(err.Error(), sentinel) {
			t.Errorf("the rendered failure reproduces bytes of the token it rejected: %q", err.Error())
		}
	})
}

// AI-07.2 — appended. The construction rules fail through AI-04's sentinels, in
// the documented order.
//
// Appended while writing item 2: the bound on the token had a home, and the
// bound on the reasoning *text* had none. Reasoning text is model-visible text
// and takes the package's existing bound, so the rule is the same one
// text_content.go states — but a rule nothing asserts is a rule nothing keeps,
// and the ordering between the four is contract under V-FAIL-04.
func TestNewReasoning_RuleViolations_FailWithTheDocumentedSentinels(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name  string
		build func() (ai.Part, error)
		want  error
		at    string
	}{
		{
			name:  "reasoning text of only whitespace",
			build: func() (ai.Part, error) { return ai.NewReasoning("  \t\n ", []byte("sig")) },
			want:  ai.ErrEmpty,
			at:    "text",
		},
		{
			name:  "reasoning text of a single space, with no token either",
			build: func() (ai.Part, error) { return ai.NewReasoning(" ", nil) },
			want:  ai.ErrEmpty,
			at:    "text",
		},
		{
			name:  "reasoning text one byte over the documented bound",
			build: func() (ai.Part, error) { return ai.NewReasoning(strings.Repeat("x", ai.MaxTextLen+1), nil) },
			want:  ai.ErrOutOfRange,
			at:    "text",
		},
		{
			name:  "no text and no token",
			build: func() (ai.Part, error) { return ai.NewReasoning("", nil) },
			want:  ai.ErrEmpty,
			at:    "token",
		},
		{
			name:  "a token one byte over the documented bound",
			build: func() (ai.Part, error) { return ai.NewReasoning("", make([]byte, ai.MaxReasoningTokenLen+1)) },
			want:  ai.ErrOutOfRange,
			at:    "token",
		},
		{
			// The order is contract: a whitespace-only text and an over-bound
			// token are both broken, and the text rule wins because text is the
			// model-visible half.
			name: "text and token both broken reports the text",
			build: func() (ai.Part, error) {
				return ai.NewReasoning("   ", make([]byte, ai.MaxReasoningTokenLen+1))
			},
			want: ai.ErrEmpty,
			at:   "text",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			part, err := tc.build()
			if err == nil {
				t.Fatalf("construction returned no failure, want one")
			}
			if !errors.Is(err, tc.want) {
				t.Errorf("errors.Is(err, %v) = false, want true (err = %v)", tc.want, err)
			}
			for _, other := range []error{ai.ErrEmpty, ai.ErrNotInVocabulary, ai.ErrOutOfRange, ai.ErrMalformed, ai.ErrUnresolvedReference} {
				if other == tc.want {
					continue
				}
				if errors.Is(err, other) {
					t.Errorf("errors.Is(err, %v) = true, want false — a failure names one rule class (err = %v)", other, err)
				}
			}

			var violation *ai.Violation
			if !errors.As(err, &violation) {
				t.Fatalf("errors.As(err, *ai.Violation) = false, want true")
			}
			if got := violation.Path().String(); got != tc.at {
				t.Errorf("violation.Path() = %q, want %q", got, tc.at)
			}
			if part != (ai.Part{}) {
				t.Errorf("a rejected construction returned a non-zero part, want the zero Part")
			}
		})
	}

	t.Run("the failure never reproduces the reasoning it was given", func(t *testing.T) {
		t.Parallel()

		const secret = "the model's private deliberation"

		_, err := ai.NewReasoning(secret+strings.Repeat("x", ai.MaxTextLen), nil)
		if err == nil {
			t.Fatalf("construction returned no failure, want one")
		}
		if strings.Contains(err.Error(), secret) {
			t.Errorf("the rendered failure reproduces the reasoning text: %q", err.Error())
		}
	})
}

func containsReasoningState(haystack []ai.ReasoningState, want ai.ReasoningState) bool {
	for _, got := range haystack {
		if got == want {
			return true
		}
	}
	return false
}
