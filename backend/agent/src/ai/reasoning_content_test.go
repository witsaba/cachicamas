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
	"fmt"
	"strings"
	"sync"
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

// AI-07.3 item 1 — the token round-trips byte-identically through a message.
//
// *(pin)* — green from birth, and recorded as such. The property is a
// consequence of AI-07.2's storage, and doc 0002's leaf anatomy allows a
// green-from-birth item precisely for this case: a regression assertion over a
// property an earlier leaf established. What it protects is the failure doc
// 0001 § 3.2 describes — a signature not returned exactly fails multi-turn
// extended thinking with tool use, on the second turn, silently — which no
// later refactor may reintroduce.
//
// The trip is deliberately two hops. One hop would pass for an implementation
// that returned the caller's own slice; two hops, with the part read out of one
// message and placed into another, is the shape a rebuilt request has.
func TestReasoning_TokenThroughAMessage_RoundTripsByteIdentical(t *testing.T) {
	t.Parallel()

	const thought = "Two hops, so that a single pass-through cannot pass by accident."

	cases := opaqueTokens()
	cases = append(cases,
		struct {
			name  string
			token []byte
		}{"high Unicode encoded as bytes", []byte("héllo · 世界 · 🜁 · \U0001F9EA")},
		struct {
			name  string
			token []byte
		}{"longer than any plausible buffer boundary", bytes.Repeat([]byte("\x00\xff\xfe signature "), 8192)},
	)

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			want := bytes.Clone(tc.token)

			part, err := ai.NewReasoning(thought, tc.token)
			if err != nil {
				t.Fatalf("ai.NewReasoning returned %v, want no failure", err)
			}

			first, err := ai.NewMessage(ai.RoleAssistant, part)
			if err != nil {
				t.Fatalf("ai.NewMessage returned %v, want no failure", err)
			}

			// Read the part back out and re-attach it, which is what a rebuilt
			// request does with a transcript it was handed.
			readBack := first.Content()[0]
			second, err := ai.NewMessage(ai.RoleAssistant, readBack)
			if err != nil {
				t.Fatalf("ai.NewMessage returned %v on the second hop, want no failure", err)
			}

			for _, hop := range []struct {
				what string
				part ai.Part
			}{
				{"as constructed", part},
				{"out of the first message", readBack},
				{"out of the second message", second.Content()[0]},
			} {
				reasoning, ok := hop.part.Reasoning()
				if !ok {
					t.Fatalf("%s: part.Reasoning() reported no reasoning", hop.what)
				}

				got, present := reasoning.Token()
				if !present {
					t.Fatalf("%s: reasoning.Token() reported no token, want one", hop.what)
				}
				if len(got) != len(want) {
					t.Fatalf("%s: the token is %d bytes, want %d", hop.what, len(got), len(want))
				}
				if !bytes.Equal(got, want) {
					t.Errorf("%s: the token changed on the trip", hop.what)
				}
				if reasoning.Text() != thought {
					t.Errorf("%s: reasoning.Text() = %q, want %q — the round trip is a property of the payload, not of the token alone",
						hop.what, reasoning.Text(), thought)
				}
			}
		})
	}
}

// AI-07.3 item 2 — the token survives AI-05.3's copy semantics.
//
// A part that keeps the caller's slice shares its backing array with whatever
// the caller does next, and the symptom is content that changed with nobody
// writing to the message — doc 0002 calls its absence "the most confusing class
// of test failure in a streaming package", and AI-08 hit exactly this on schema
// bytes. There are two halves and a fix for one is not a fix for the other: the
// construction path takes the caller's slice, and the read path hands one out.
func TestReasoning_TokenAcrossCopies_IsUnaffectedByTheCallersSlice(t *testing.T) {
	t.Parallel()

	const thought = "A signature is only a signature if it is the same bytes."

	original := []byte("\x00\xffthe provider's signature\xfe\x01")
	want := bytes.Clone(original)

	t.Run("mutating the slice the caller supplied does not change the token", func(t *testing.T) {
		t.Parallel()

		supplied := bytes.Clone(want)

		part, err := ai.NewReasoning(thought, supplied)
		if err != nil {
			t.Fatalf("ai.NewReasoning returned %v, want no failure", err)
		}

		// The caller reuses its decode buffer, which is the ordinary thing for
		// a caller to do and the reason this is a hazard rather than a curio.
		for i := range supplied {
			supplied[i] = 0x2a
		}

		reasoning, _ := part.Reasoning()
		got, present := reasoning.Token()
		if !present {
			t.Fatalf("reasoning.Token() reported no token, want one")
		}
		if !bytes.Equal(got, want) {
			t.Errorf("the token became %x after the caller overwrote its own slice, want %x", got, want)
		}
	})

	t.Run("mutating the bytes an accessor returned does not change the token", func(t *testing.T) {
		t.Parallel()

		part, err := ai.NewReasoning(thought, bytes.Clone(want))
		if err != nil {
			t.Fatalf("ai.NewReasoning returned %v, want no failure", err)
		}
		reasoning, _ := part.Reasoning()

		handedOut, _ := reasoning.Token()
		for i := range handedOut {
			handedOut[i] = 0x2a
		}

		again, _ := reasoning.Token()
		if !bytes.Equal(again, want) {
			t.Errorf("the token became %x after a consumer overwrote the bytes it received, want %x", again, want)
		}
	})

	t.Run("copying a message copies the token exactly", func(t *testing.T) {
		t.Parallel()

		part, err := ai.NewReasoning(thought, bytes.Clone(want))
		if err != nil {
			t.Fatalf("ai.NewReasoning returned %v, want no failure", err)
		}
		msg, err := ai.NewMessage(ai.RoleAssistant, part)
		if err != nil {
			t.Fatalf("ai.NewMessage returned %v, want no failure", err)
		}

		copied := msg

		// One consumer rewrites what it received; the other must not see it.
		first, _ := copied.Content()[0].Reasoning()
		mine, _ := first.Token()
		for i := range mine {
			mine[i] = 0x2a
		}

		second, _ := msg.Content()[0].Reasoning()
		theirs, _ := second.Token()
		if !bytes.Equal(theirs, want) {
			t.Errorf("the copy's token is %x, want %x — two holders of a copy observed each other", theirs, want)
		}
	})

	t.Run("two consumers reading in parallel observe the same bytes", func(t *testing.T) {
		t.Parallel()

		part, err := ai.NewReasoning(thought, bytes.Clone(want))
		if err != nil {
			t.Fatalf("ai.NewReasoning returned %v, want no failure", err)
		}
		msg, err := ai.NewMessage(ai.RoleAssistant, part)
		if err != nil {
			t.Fatalf("ai.NewMessage returned %v, want no failure", err)
		}

		var wg sync.WaitGroup
		for range 8 {
			wg.Add(1)
			go func() {
				defer wg.Done()

				reasoning, _ := msg.Content()[0].Reasoning()
				token, _ := reasoning.Token()
				if !bytes.Equal(token, want) {
					t.Errorf("a parallel reader observed %x, want %x", token, want)
				}
				for i := range token {
					token[i] = 0x2a
				}
			}()
		}
		wg.Wait()
	})
}

// AI-07.3 — appended. A reasoning part compares with == like every other part.
//
// Discovered while fixing item 2's aliasing bug. content_part.go states the
// property as landed contract: "Part is a value... Equality with == is defined
// and compares payloads." A payload containing a slice makes == panic at
// runtime for that kind, so a second kind arriving would have silently removed
// a property of the first — the sort of regression that is invisible until a
// consumer writes the comparison the documentation invited.
func TestReasoningPart_Equality_ComparesPayloadsWithoutPanicking(t *testing.T) {
	t.Parallel()

	const thought = "Equality is part of what a value type promises."

	build := func(text string, token []byte) ai.Part {
		t.Helper()

		part, err := ai.NewReasoning(text, token)
		if err != nil {
			t.Fatalf("ai.NewReasoning returned %v, want no failure", err)
		}
		return part
	}

	signature := []byte("\x00\xffsignature\xfe")

	t.Run("two parts built from the same payload are equal", func(t *testing.T) {
		t.Parallel()

		if first, second := build(thought, signature), build(thought, signature); first != second {
			t.Errorf("two reasoning parts built from the same payload are not equal")
		}
	})

	t.Run("a copy is equal to its original", func(t *testing.T) {
		t.Parallel()

		part := build(thought, signature)
		if copied := part; copied != part {
			t.Errorf("a copied reasoning part is not equal to its original")
		}
	})

	t.Run("parts differing in payload, kind or presence are not equal", func(t *testing.T) {
		t.Parallel()

		textPart, err := ai.NewText(thought)
		if err != nil {
			t.Fatalf("ai.NewText returned %v, want no failure", err)
		}
		redacted, err := ai.NewRedactedReasoning(signature)
		if err != nil {
			t.Fatalf("ai.NewRedactedReasoning returned %v, want no failure", err)
		}

		reference := build(thought, signature)
		for _, tc := range []struct {
			what  string
			other ai.Part
		}{
			{"a different token", build(thought, []byte("another signature"))},
			{"a different reasoning text", build("a different thought entirely", signature)},
			{"no token rather than an empty one", build(thought, nil)},
			{"an empty token rather than none", build(thought, []byte{})},
			{"a redacted part", redacted},
			{"a text part", textPart},
			{"the part that was never constructed", ai.Part{}},
		} {
			if reference == tc.other {
				t.Errorf("a reasoning part equals %s, want them distinguished", tc.what)
			}
		}

		// The absent/empty distinction is preserved by equality too, which is
		// the same fact AI-07.2 item 1 asserts through the accessor.
		if build(thought, nil) == build(thought, []byte{}) {
			t.Errorf("a part with no token equals one with an empty token — equality collapsed the distinction")
		}
	})
}

// AI-07.4 item 1 — a redacted reasoning part replays its payload verbatim, and
// is distinguishable from a part that merely has no text.
//
// At least one provider ships encrypted redacted blocks that must be returned
// exactly as they arrived. "Redacted" and "this provider emitted no reasoning
// text" are two facts, not one: the first says a plaintext exists and was
// withheld, the second says none was produced. Collapsing them loses the only
// information an adapter has about what it is holding.
func TestReasoning_RedactedVariant_ReplaysItsPayloadVerbatim(t *testing.T) {
	t.Parallel()

	t.Run("the opaque payload survives a message trip byte for byte", func(t *testing.T) {
		t.Parallel()

		for _, tc := range opaqueTokens() {
			t.Run(tc.name, func(t *testing.T) {
				t.Parallel()

				want := bytes.Clone(tc.token)

				part, err := ai.NewRedactedReasoning(tc.token)
				if err != nil {
					t.Fatalf("ai.NewRedactedReasoning returned %v, want no failure", err)
				}
				msg, err := ai.NewMessage(ai.RoleAssistant, part)
				if err != nil {
					t.Fatalf("ai.NewMessage returned %v, want no failure", err)
				}

				reasoning, ok := msg.Content()[0].Reasoning()
				if !ok {
					t.Fatalf("part.Reasoning() reported no reasoning")
				}
				if got := reasoning.State(); got != ai.ReasoningStateRedacted {
					t.Errorf("reasoning.State() = %v, want %v", got, ai.ReasoningStateRedacted)
				}
				if reasoning.Text() != "" {
					t.Errorf("reasoning.Text() = %q on a redacted part, want the empty string", reasoning.Text())
				}

				got, present := reasoning.Token()
				if !present {
					t.Fatalf("reasoning.Token() reported no payload on a redacted part")
				}
				if !bytes.Equal(got, want) {
					t.Errorf("the redacted payload came back %x, want %x — an encrypted block must replay verbatim", got, want)
				}
			})
		}
	})

	t.Run("redacted is distinguishable from a part that merely has no text", func(t *testing.T) {
		t.Parallel()

		payload := []byte("\x00the same bytes, two different facts\xff")

		redacted, err := ai.NewRedactedReasoning(payload)
		if err != nil {
			t.Fatalf("ai.NewRedactedReasoning returned %v, want no failure", err)
		}
		tokenOnly, err := ai.NewReasoning("", payload)
		if err != nil {
			t.Fatalf("ai.NewReasoning returned %v, want no failure", err)
		}

		redactedReasoning, _ := redacted.Reasoning()
		tokenOnlyReasoning, _ := tokenOnly.Reasoning()

		if redactedReasoning.State() == tokenOnlyReasoning.State() {
			t.Fatalf("both parts report state %v — \"the provider withheld the plaintext\" and \"the provider emitted no reasoning text\" collapsed into one",
				redactedReasoning.State())
		}
		if redactedReasoning.State() != ai.ReasoningStateRedacted {
			t.Errorf("the redacted part reports state %v, want %v", redactedReasoning.State(), ai.ReasoningStateRedacted)
		}
		if tokenOnlyReasoning.State() != ai.ReasoningStateTokenOnly {
			t.Errorf("the token-only part reports state %v, want %v", tokenOnlyReasoning.State(), ai.ReasoningStateTokenOnly)
		}

		// Both carry the identical bytes, so nothing above passed because the
		// payloads happened to differ.
		redactedToken, _ := redactedReasoning.Token()
		tokenOnlyToken, _ := tokenOnlyReasoning.Token()
		if !bytes.Equal(redactedToken, tokenOnlyToken) {
			t.Errorf("the two parts carry different bytes, so the state comparison above proves less than it claims")
		}
	})
}

// AI-07.4 item 2 — the signature-only shape is constructible and valid.
//
// *(pin)* — green from birth over AI-07.1 item 3 and AI-07.2 item 1, and
// recorded as such. What it adds is the word **valid**: a part with a token and
// no text must be accepted by message construction rather than merely
// constructible, because doc 0001 § 3.3 row 2 needs "a state that can say 'this
// provider emitted no reasoning text'" to survive into a request rather than
// being filtered at the boundary as a degenerate part.
func TestReasoning_SignatureOnlyVariant_IsConstructibleAndValid(t *testing.T) {
	t.Parallel()

	signature := []byte("\x00\xfea cryptographic signature over content this part does not carry\xff")

	part, err := ai.NewReasoning("", signature)
	if err != nil {
		t.Fatalf("ai.NewReasoning returned %v, want no failure — the signature-only shape is valid", err)
	}

	msg, err := ai.NewMessage(ai.RoleAssistant, part)
	if err != nil {
		t.Fatalf("ai.NewMessage rejected a signature-only reasoning part with %v, want no failure", err)
	}

	reasoning, ok := msg.Content()[0].Reasoning()
	if !ok {
		t.Fatalf("part.Reasoning() reported no reasoning")
	}
	if got := reasoning.State(); got != ai.ReasoningStateTokenOnly {
		t.Errorf("reasoning.State() = %v, want %v — \"no reasoning text\" is a recorded state, not an empty string", got, ai.ReasoningStateTokenOnly)
	}
	if reasoning.Text() != "" {
		t.Errorf("reasoning.Text() = %q, want the empty string", reasoning.Text())
	}

	token, present := reasoning.Token()
	if !present {
		t.Fatalf("reasoning.Token() reported no token on a signature-only part")
	}
	if !bytes.Equal(token, signature) {
		t.Errorf("the signature came back %x, want %x", token, signature)
	}

	// A signature-only part alongside text content in one message is the shape
	// an assistant turn actually takes, and the two must not interfere.
	text, err := ai.NewText("The answer is 4.")
	if err != nil {
		t.Fatalf("ai.NewText returned %v, want no failure", err)
	}
	mixed, err := ai.NewMessage(ai.RoleAssistant, part, text)
	if err != nil {
		t.Fatalf("ai.NewMessage returned %v for reasoning beside text, want no failure", err)
	}
	if got := len(mixed.Content()); got != 2 {
		t.Fatalf("the message holds %d parts, want 2", got)
	}
	if kind := mixed.Content()[0].Kind(); kind != ai.PartKindReasoning {
		t.Errorf("the first part reports kind %v, want %v", kind, ai.PartKindReasoning)
	}
	if kind := mixed.Content()[1].Kind(); kind != ai.PartKindText {
		t.Errorf("the second part reports kind %v, want %v", kind, ai.PartKindText)
	}
}

// AI-07.4 item 3 — neither variant is confusable with a text part or with a
// part that was never constructed.
//
// *(pin)* over AI-07.1 item 2 and AI-06's seal, extended to the two shapes that
// carry no text: those are precisely the ones an accessor could plausibly
// answer for, since "no text" and "no payload" look alike from the outside and
// are not alike at all.
func TestReasoning_RedactedAndSignatureOnly_AreConfusableWithNothing(t *testing.T) {
	t.Parallel()

	payload := []byte("\x00opaque\xff")

	redacted, err := ai.NewRedactedReasoning(payload)
	if err != nil {
		t.Fatalf("ai.NewRedactedReasoning returned %v, want no failure", err)
	}
	signatureOnly, err := ai.NewReasoning("", payload)
	if err != nil {
		t.Fatalf("ai.NewReasoning returned %v, want no failure", err)
	}

	for _, tc := range []struct {
		name string
		part ai.Part
	}{{"redacted", redacted}, {"signature-only", signatureOnly}} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			if text, ok := tc.part.Text(); ok || text != "" {
				t.Errorf("part.Text() = (%q, %t), want (%q, false) — a part with no reasoning text is not a text part", text, ok, "")
			}
			if tc.part == (ai.Part{}) {
				t.Errorf("the part equals the zero Part, want a constructed part")
			}
			if !containsPartKind(ai.PartKinds(), tc.part.Kind()) {
				t.Errorf("part.Kind() = %v, which is not a member of the kind vocabulary", tc.part.Kind())
			}
			if _, err := ai.NewMessage(ai.RoleAssistant, tc.part); err != nil {
				t.Errorf("ai.NewMessage rejected the part with %v, want no failure — it is valid content, not an absent part", err)
			}

			// And the zero part is not confusable with it in the other
			// direction: the accessor that answers for this part must not
			// answer for a part that was never constructed.
			var unconstructed ai.Part
			if _, ok := unconstructed.Reasoning(); ok {
				t.Errorf("the zero Part reported reasoning")
			}
			if reasoning, ok := tc.part.Reasoning(); !ok {
				t.Errorf("part.Reasoning() reported no reasoning on a constructed reasoning part")
			} else if reasoning.State() == 0 {
				t.Errorf("reasoning.State() = %v, want a member of the vocabulary", reasoning.State())
			}
		})
	}
}

// AI-07.4 — appended. The reasoning payload renders without its payload.
//
// Discovered while writing item 3. content_part.go records why [ai.Part] has a
// String method: "fmt prints the unexported fields of a struct it has no String
// method for, so %v on a part would print the prompt". [ai.Reasoning] is the
// first exported payload *struct* in the package, so it is the first value the
// same sentence applies to, and it carries the two most sensitive things Layer
// 1 holds — a model's private deliberation, and a blob that may be
// credential-shaped. V-FAIL-13 puts the posture on the type rather than on
// which verb someone reached for, so every verb is asserted rather than the
// three a reader thinks of.
func TestReasoning_DiagnosticRendering_CarriesNoPayload(t *testing.T) {
	t.Parallel()

	const deliberation = "SENSITIVE-DELIBERATION"
	signature := []byte("SENSITIVE-SIGNATURE-BYTES")

	withText, err := ai.NewReasoning(deliberation, signature)
	if err != nil {
		t.Fatalf("ai.NewReasoning returned %v, want no failure", err)
	}
	redacted, err := ai.NewRedactedReasoning(signature)
	if err != nil {
		t.Fatalf("ai.NewRedactedReasoning returned %v, want no failure", err)
	}

	for _, tc := range []struct {
		name  string
		part  ai.Part
		state ai.ReasoningState
	}{
		{"reasoning with text and a token", withText, ai.ReasoningStateText},
		{"a redacted part", redacted, ai.ReasoningStateRedacted},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			reasoning, ok := tc.part.Reasoning()
			if !ok {
				t.Fatalf("part.Reasoning() reported no reasoning")
			}

			for _, verb := range []string{"%v", "%s", "%+v", "%#v", "%q"} {
				for what, rendered := range map[string]string{
					"the payload": fmt.Sprintf(verb, reasoning),
					"the part":    fmt.Sprintf(verb, tc.part),
				} {
					if strings.Contains(rendered, deliberation) {
						t.Errorf("%s rendered with %s reproduces the reasoning text: %s", what, verb, rendered)
					}
					if strings.Contains(rendered, string(signature)) {
						t.Errorf("%s rendered with %s reproduces the token: %s", what, verb, rendered)
					}
				}
			}

			// It still says something useful: the state, which is structural
			// and carries no caller data.
			if got, want := reasoning.String(), "reasoning("+tc.state.String()+")"; got != want {
				t.Errorf("reasoning.String() = %q, want %q", got, want)
			}
		})
	}
}

func containsPartKind(haystack []ai.PartKind, want ai.PartKind) bool {
	for _, got := range haystack {
		if got == want {
			return true
		}
	}
	return false
}

func containsReasoningState(haystack []ai.ReasoningState, want ai.ReasoningState) bool {
	for _, got := range haystack {
		if got == want {
			return true
		}
	}
	return false
}
