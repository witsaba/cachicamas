// Tests for AI-12.3 — the provider escape hatch: typed, namespaced, opaque.
//
// External package, for AI-06's reason: the consumer this contract exists
// for is an adapter in a vendor package, and doc 0002 makes readability from
// outside constitutive rather than incidental. Two fake translators —
// claiming "alpha" and "beta" — stand in for that adapter, per design.md
// § 11: no production adapter, no network, no vendor package.

package ai_test

import (
	"bytes"
	"fmt"
	"strings"
	"testing"

	"github.com/cachicamas/backend/agent/src/ai"
)

// alphaTranslator is a fake translator claiming provider namespace "alpha".
func alphaTranslator(r ai.Request) ([]byte, bool) {
	ext, ok := r.ProviderExtension("alpha")
	if !ok {
		return nil, false
	}
	return ext.Value(), true
}

// betaTranslator is a fake translator claiming provider namespace "beta".
func betaTranslator(r ai.Request) ([]byte, bool) {
	ext, ok := r.ProviderExtension("beta")
	if !ok {
		return nil, false
	}
	return ext.Value(), true
}

// betaRender is a fake translator that renders a deterministic byte sequence
// from the parts of a request the "beta" provider cares about: the model
// identity, plus its own claimed namespace when present. It stands in for a
// real adapter's request-to-wire translation, so item 2's test can assert
// two whole RENDERINGS are byte-identical rather than merely that two
// lookups individually agree.
func betaRender(r ai.Request) []byte {
	out := []byte(r.Model())
	if ext, ok := r.ProviderExtension("beta"); ok {
		out = append(out, ext.Value()...)
	}
	return out
}

// AI-12.3 item 1 — a provider-namespaced opaque value survives to the
// adapter that claims the namespace, byte-exact, including bytes that are
// not valid UTF-8 (R-REX-006, S-REX-026, S-REX-027, S-REX-034).
//
// Two sub-cases triangulate: printable ASCII (the case a naive
// string-shaped implementation would pass by accident) and bytes that are
// not valid UTF-8 at all (the case V-REQ-06's byte-exactness is actually
// about).
func TestProviderExtension_ClaimedByItsNamespace_SurvivesByteExact(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name  string
		value []byte
	}{
		{"printable_ascii", []byte("a plain provider-specific payload")},
		{"not_valid_utf8", []byte{0xff, 0xfe, 'p', 'a', 'y', 'l', 'o', 'a', 'd', 0x00, 0x80}},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()

			request, err := ai.NewRequest("m", []ai.Message{userTextMessage(t, "go")}, ai.WithProviderExtension("alpha", c.value))
			if err != nil {
				t.Fatalf("ai.NewRequest returned %v, want no failure", err)
			}

			got, ok := alphaTranslator(request)
			if !ok {
				t.Fatalf("alphaTranslator found no alpha extension, want the one supplied")
			}
			if !bytes.Equal(got, c.value) {
				t.Errorf("alphaTranslator's value = %q, want %q — byte-identical", got, c.value)
			}
		})
	}
}

// AI-12.3 item 2 — an adapter for a DIFFERENT provider finds no value under
// a namespace it does not claim, and its own translation is byte-identical
// whether or not another provider's extension is present. This pair is the
// milestone's acceptance clause (R-REX-007, S-REX-032, S-REX-033).
func TestProviderExtension_ForeignNamespace_IsInvisibleAndDoesNotAffectTranslation(t *testing.T) {
	t.Parallel()

	messages := []ai.Message{userTextMessage(t, "go")}

	without, err := ai.NewRequest("m", messages)
	if err != nil {
		t.Fatalf("ai.NewRequest returned %v, want no failure", err)
	}
	withAlpha, err := ai.NewRequest("m", messages, ai.WithProviderExtension("alpha", []byte("alpha-value")))
	if err != nil {
		t.Fatalf("ai.NewRequest returned %v, want no failure", err)
	}

	// S-REX-032 — beta finds no alpha extension.
	if _, ok := betaTranslator(withAlpha); ok {
		t.Errorf("betaTranslator found a value under alpha's namespace, want none")
	}

	// S-REX-033 — beta's whole rendering is byte-identical with and without
	// alpha present.
	got, want := betaRender(withAlpha), betaRender(without)
	if !bytes.Equal(got, want) {
		t.Errorf("betaRender(withAlpha) = %q, want %q (== betaRender(without)) — "+
			"a third provider's namespace must not affect this one's translation", got, want)
	}
}

// AI-12.3 item 3, validation half — the pass-through is inert in
// validation: two requests differing only in a third provider's namespace
// validate identically, whether both succeed or both fail the same rule
// elsewhere (R-REX-007, S-REX-035, S-REX-036).
func TestProviderExtension_InThirdNamespace_IsInertInValidation(t *testing.T) {
	t.Parallel()

	t.Run("both_construct_successfully", func(t *testing.T) {
		// S-REX-035.
		t.Parallel()

		messages := []ai.Message{userTextMessage(t, "go")}
		if _, err := ai.NewRequest("m", messages); err != nil {
			t.Fatalf("ai.NewRequest without an extension returned %v, want no failure", err)
		}
		if _, err := ai.NewRequest("m", messages, ai.WithProviderExtension("gamma", []byte("v"))); err != nil {
			t.Fatalf("ai.NewRequest with a gamma extension returned %v, want no failure", err)
		}
	})

	t.Run("both_fail_the_same_unrelated_rule_identically", func(t *testing.T) {
		// S-REX-036.
		t.Parallel()

		messages := []ai.Message{userTextMessage(t, "go")}

		_, withoutErr := ai.NewRequest("m", messages, ai.WithTemperature(-1))
		_, withErr := ai.NewRequest("m", messages, ai.WithTemperature(-1), ai.WithProviderExtension("gamma", []byte("v")))

		requireViolation(t, withoutErr, ai.ErrOutOfRange, "temperature")
		requireViolation(t, withErr, ai.ErrOutOfRange, "temperature")
	})
}

// AI-12.3 item 3, equality half — extensions participate in equality,
// structurally: two requests identical except for one extension VALUE are
// NOT equal, and a rebuild preserves the region (R-REX-007, S-REX-037,
// S-REX-038).
func TestRequest_Equal_ComparesProviderExtensions(t *testing.T) {
	t.Parallel()

	t.Run("differing_only_in_an_extension_value_are_not_equal", func(t *testing.T) {
		// S-REX-037.
		t.Parallel()

		messages := []ai.Message{userTextMessage(t, "go")}
		first, err := ai.NewRequest("m", messages, ai.WithProviderExtension("alpha", []byte("v1")))
		if err != nil {
			t.Fatalf("ai.NewRequest returned %v, want no failure", err)
		}
		second, err := ai.NewRequest("m", messages, ai.WithProviderExtension("alpha", []byte("v2")))
		if err != nil {
			t.Fatalf("ai.NewRequest returned %v, want no failure", err)
		}

		if first.Equal(second) {
			t.Errorf("first.Equal(second) = true, want false — the extension values differ")
		}
		if second.Equal(first) {
			t.Errorf("second.Equal(first) = true, want false — Equal must be symmetric about inequality too")
		}
	})

	t.Run("a_rebuild_preserves_the_region", func(t *testing.T) {
		// S-REX-038.
		t.Parallel()

		source, err := ai.NewRequest("m", []ai.Message{userTextMessage(t, "go")}, ai.WithProviderExtension("alpha", []byte("v1")))
		if err != nil {
			t.Fatalf("ai.NewRequest returned %v, want no failure", err)
		}

		derived, err := source.With(ai.WithTemperature(0.5)) // unrelated change
		if err != nil {
			t.Fatalf("source.With(...) returned %v, want no failure", err)
		}

		if derived.Equal(source) {
			t.Errorf("derived.Equal(source) = true, want false — they differ where the change was made")
		}
		ext, ok := derived.ProviderExtension("alpha")
		if !ok || !bytes.Equal(ext.Value(), []byte("v1")) {
			t.Errorf("derived.ProviderExtension(\"alpha\") = (%v, %t), want the source's value preserved", ext, ok)
		}
	})
}

// AI-12.3 item 4 (appended) — the region's own rules: an empty or
// whitespace-only namespace, and an empty value, each fail with ErrEmpty at
// the ordinal position (R-REX-008, S-REX-039 … S-REX-041).
func TestNewRequest_ProviderExtensionRuleViolations_FailWithTheDocumentedSentinels(t *testing.T) {
	t.Parallel()

	messages := []ai.Message{userTextMessage(t, "go")}

	cases := []struct {
		what     string
		option   ai.RequestOption
		wantRule error
		wantPath string
	}{
		{"an_empty_namespace", ai.WithProviderExtension("", []byte("v")), ai.ErrEmpty, "extensions[0].namespace"},
		{"a_whitespace-only_namespace", ai.WithProviderExtension(" \t\n ", []byte("v")), ai.ErrEmpty, "extensions[0].namespace"},
		{"a_nil_value", ai.WithProviderExtension("alpha", nil), ai.ErrEmpty, "extensions[0].value"},
		{"an_empty-but-non-nil_value", ai.WithProviderExtension("alpha", []byte{}), ai.ErrEmpty, "extensions[0].value"},
	}

	for _, c := range cases {
		t.Run(c.what, func(t *testing.T) {
			t.Parallel()

			_, err := ai.NewRequest("m", messages, c.option)
			requireViolation(t, err, c.wantRule, c.wantPath)
		})
	}
}

// AI-12.3 item 4 (appended), the other direction — a whitespace-only value
// is a real, legal value because a value is opaque bytes this package does
// not know are whitespace; a namespace has no format rule beyond emptiness;
// the second extension's failure is positioned by ordinal, never by
// namespace; and the rule applies identically through the rebuild path
// (S-REX-042 … S-REX-045).
func TestNewRequest_ProviderExtension_LegalEdgesAndOrdinalPositioning(t *testing.T) {
	t.Parallel()

	t.Run("a_whitespace-only_value_constructs_and_reads_back", func(t *testing.T) {
		// S-REX-042.
		t.Parallel()

		request, err := ai.NewRequest("m", []ai.Message{userTextMessage(t, "go")}, ai.WithProviderExtension("alpha", []byte(" ")))
		if err != nil {
			t.Fatalf("ai.NewRequest returned %v, want no failure", err)
		}
		ext, ok := request.ProviderExtension("alpha")
		if !ok || !bytes.Equal(ext.Value(), []byte(" ")) {
			t.Errorf("request.ProviderExtension(\"alpha\") = (%v, %t), want a legal whitespace value", ext, ok)
		}
	})

	t.Run("a_second_extensions_empty_namespace_is_positioned_by_ordinal", func(t *testing.T) {
		// S-REX-043.
		t.Parallel()

		_, err := ai.NewRequest("m", []ai.Message{userTextMessage(t, "go")},
			ai.WithProviderExtension("alpha", []byte("v1")),
			ai.WithProviderExtension("", []byte("v2")),
		)
		requireViolation(t, err, ai.ErrEmpty, "extensions[1].namespace")
	})

	t.Run("punctuation_dots_and_non-ASCII_bytes_in_a_namespace_construct_because_no_format_rule_is_imposed", func(t *testing.T) {
		// S-REX-044.
		t.Parallel()

		if _, err := ai.NewRequest("m", []ai.Message{userTextMessage(t, "go")}, ai.WithProviderExtension("acme.vendor-v2_ünïcödé", []byte("v"))); err != nil {
			t.Errorf("ai.NewRequest returned %v, want no failure — no format rule is imposed on a namespace", err)
		}
	})

	t.Run("the_same_rule_applies_through_the_rebuild_path", func(t *testing.T) {
		// S-REX-045.
		t.Parallel()

		source := buildDerivableRequest(t)
		_, err := source.With(ai.WithProviderExtension("", []byte("v")))
		requireViolation(t, err, ai.ErrEmpty, "extensions[0].namespace")
	})
}

// AI-12.3 item 5 (appended) — re-applying a namespace is last-wins and
// keeps its FIRST read-back ordinal; the region is a slice, never a map
// (R-REX-006, S-REX-028 … S-REX-031).
func TestProviderExtension_ReapplyingANamespace_IsLastWinsAndKeepsItsFirstOrdinal(t *testing.T) {
	t.Parallel()

	t.Run("a_namespace_never_applied_is_absent_not_a_zero_value", func(t *testing.T) {
		// S-REX-028.
		t.Parallel()

		request, err := ai.NewRequest("m", []ai.Message{userTextMessage(t, "go")})
		if err != nil {
			t.Fatalf("ai.NewRequest returned %v, want no failure", err)
		}
		if _, ok := request.ProviderExtension("alpha"); ok {
			t.Errorf("ProviderExtension(\"alpha\") on a request with no extensions reported present, want absent")
		}
	})

	t.Run("Value_copies_out", func(t *testing.T) {
		// S-REX-029.
		t.Parallel()

		value := []byte("mutable-check")
		request, err := ai.NewRequest("m", []ai.Message{userTextMessage(t, "go")}, ai.WithProviderExtension("alpha", value))
		if err != nil {
			t.Fatalf("ai.NewRequest returned %v, want no failure", err)
		}

		ext, _ := request.ProviderExtension("alpha")
		read := ext.Value()
		read[0] = 'X' // the reader mutates what it received

		reread, _ := request.ProviderExtension("alpha")
		if !bytes.Equal(reread.Value(), value) {
			t.Errorf("after a reader mutated the slice it received from Value(), the request's extension value changed — " +
				"the request handed out its own storage")
		}
	})

	t.Run("applying_a_then_b_then_a_again_keeps_a_first_with_the_second_value", func(t *testing.T) {
		// S-REX-030.
		t.Parallel()

		request, err := ai.NewRequest("m", []ai.Message{userTextMessage(t, "go")},
			ai.WithProviderExtension("a", []byte("first")),
			ai.WithProviderExtension("b", []byte("only")),
			ai.WithProviderExtension("a", []byte("second")),
		)
		if err != nil {
			t.Fatalf("ai.NewRequest returned %v, want no failure", err)
		}

		got := request.ProviderExtensions()
		if len(got) != 2 {
			t.Fatalf("len(ProviderExtensions()) = %d, want 2", len(got))
		}
		if got[0].Namespace() != "a" || !bytes.Equal(got[0].Value(), []byte("second")) {
			t.Errorf(`ProviderExtensions()[0] = (%q, %q), want ("a", "second") — first ordinal, second value`, got[0].Namespace(), got[0].Value())
		}
		if got[1].Namespace() != "b" || !bytes.Equal(got[1].Value(), []byte("only")) {
			t.Errorf(`ProviderExtensions()[1] = (%q, %q), want ("b", "only")`, got[1].Namespace(), got[1].Value())
		}
	})

	t.Run("a_namespace_for_which_no_adapter_exists_constructs_because_layer_1_recognises_none", func(t *testing.T) {
		// S-REX-031.
		t.Parallel()

		if _, err := ai.NewRequest("m", []ai.Message{userTextMessage(t, "go")}, ai.WithProviderExtension("no-such-provider-anyone-registered", []byte("v"))); err != nil {
			t.Errorf("ai.NewRequest returned %v, want no failure — Layer 1 recognises no namespace and rejects none", err)
		}
	})
}

// AI-12.3 item 6 (appended) — the extension region renders no payload
// through any formatting verb, and the rendering still names the region
// and its count (R-REX-010, S-REX-050 … S-REX-052).
func TestProviderExtension_Formatting_RendersNoPayloadThroughAnyVerbButNamesTheCount(t *testing.T) {
	t.Parallel()

	const (
		namespaceSecret = "SECRET-NAMESPACE-vendor-x"
		valueSecret     = "SECRET-EXTENSION-VALUE"
	)

	request, err := ai.NewRequest("m", []ai.Message{userTextMessage(t, "go")}, ai.WithProviderExtension(namespaceSecret, []byte(valueSecret)))
	if err != nil {
		t.Fatalf("ai.NewRequest returned %v, want no failure", err)
	}
	ext, ok := request.ProviderExtension(namespaceSecret)
	if !ok {
		t.Fatalf("request.ProviderExtension(...) reported absent, want present")
	}

	verbs := []string{"%v", "%s", "%+v", "%#v"}
	secrets := map[string]string{"the namespace": namespaceSecret, "the value": valueSecret}

	for _, verb := range verbs {
		t.Run(verb, func(t *testing.T) {
			t.Parallel()

			// S-REX-050 — the request's own rendering.
			renderedRequest := fmt.Sprintf(verb, request)
			for region, secret := range secrets {
				if strings.Contains(renderedRequest, secret) {
					t.Errorf("fmt.Sprintf(%q, request) leaked %s: %q", verb, region, renderedRequest)
				}
			}

			// S-REX-052 — the extension value formatted directly.
			renderedExt := fmt.Sprintf(verb, ext)
			for region, secret := range secrets {
				if strings.Contains(renderedExt, secret) {
					t.Errorf("fmt.Sprintf(%q, extension) leaked %s: %q", verb, region, renderedExt)
				}
			}
		})
	}

	// S-REX-051 — the rendering still names the region and its count, the
	// established "N <region>" shape TestRequest_Formatting_Names... already
	// pins for tools and cache boundaries.
	rendered := request.String()
	if !strings.Contains(rendered, "1 extensions") {
		t.Errorf("request.String() = %q, want it to contain %q", rendered, "1 extensions")
	}
}
