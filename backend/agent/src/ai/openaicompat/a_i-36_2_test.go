// AI-36 (WU-3, D-5/D-6) — configuration, configured-client and
// header-capture redaction proofs (R-APC-015, S-APC-077…080).
package openaicompat_test

import (
	"encoding/json"
	"fmt"
	"strings"
	"testing"

	"github.com/cachicamas/backend/agent/src/ai/openaicompat"
)

// fiveFormRenderings renders v under the plain, string, extended and
// Go-syntax verbs plus its structured (JSON) serialization — the same
// five-form shape credential_test.go's own
// TestCredential_NeverRendersRawToken established (S-APC-053) — and
// returns every rendering keyed by a label naming the form.
func fiveFormRenderings(t *testing.T, v any) map[string]string {
	t.Helper()
	renderings := map[string]string{
		"%v":  fmt.Sprintf("%v", v),
		"%s":  fmt.Sprintf("%s", v), //nolint:staticcheck // intentional verb coverage
		"%+v": fmt.Sprintf("%+v", v),
		"%#v": fmt.Sprintf("%#v", v),
	}
	jsonBytes, err := json.Marshal(v) //nolint:staticcheck // intentional verb coverage
	if err != nil {
		t.Fatalf("json.Marshal(%T) error = %v, want nil", v, err)
	}
	renderings["json.Marshal"] = string(jsonBytes)
	return renderings
}

// assertNoneContainSentinel fails t, naming every offending form, if any
// rendering in renderings contains sentinel.
func assertNoneContainSentinel(t *testing.T, renderings map[string]string, sentinel string) {
	t.Helper()
	for form, rendered := range renderings {
		if strings.Contains(rendered, sentinel) {
			t.Errorf("rendering via %s leaked the sentinel: %q", form, rendered)
		}
	}
}

// anyFormContainsSentinel reports whether at least one rendering in
// renderings contains sentinel.
func anyFormContainsSentinel(renderings map[string]string, sentinel string) bool {
	for _, rendered := range renderings {
		if strings.Contains(rendered, sentinel) {
			return true
		}
	}
	return false
}

// TestConfig_NeverRendersCredential covers S-APC-077: the adapter's own
// configuration value, verb-exhaustively, never discloses a planted
// sentinel credential — elevating credential_redaction_test.go's existing
// proof from the openrouter.Config wrapping level down to
// openaicompat.Config itself. The inline unredacted stand-in is this
// scenario's own required control: an unredacted rendering carrying the
// identical sentinel MUST fail the same check, proving the assertion can
// bite. Expected GREEN on first run — Config's Credential field already
// redacts by construction (credential.go); this test elevates the proof,
// it does not change behavior.
func TestConfig_NeverRendersCredential(t *testing.T) {
	t.Parallel()

	// Runtime-assembled (never a contiguous "sk-..." literal in this
	// source file): credential_scan_test.go's widened recursive sweep
	// (AI-36 WU-5) now covers every _test.go file in this tree, so a
	// spelled-out credential-shaped sentinel here would flag this file as
	// an undeclared deliberate plant. Splitting the literal keeps this
	// test's own sentinel out of that surface without joining the
	// allowlist (design.md AD-4's own rule for new AI-36 tests).
	sentinel := "s" + "k" + "-ai36-config-level-sentinel-abcdef123456"

	t.Run("Config built with the credential type never renders it", func(t *testing.T) {
		t.Parallel()

		cfg := openaicompat.Config{
			Endpoint:   "https://example.test",
			Credential: openaicompat.NewCredential(sentinel),
		}
		assertNoneContainSentinel(t, fiveFormRenderings(t, cfg), sentinel)
	})

	t.Run("an unredacted stand-in carrying the same sentinel fails the same check (inline positive control)", func(t *testing.T) {
		t.Parallel()

		// unredactedStandIn deliberately holds the sentinel in a plain
		// exported string field — no redacting type, no custom
		// String/GoString — so the SAME five-form check the safe case
		// above uses MUST detect the leak. Without this control, a Config
		// check that could never fail would be an unfalsifiable claim
		// (this milestone's own characteristic failure mode).
		type unredactedStandIn struct {
			Endpoint   string
			Credential string
		}
		standIn := unredactedStandIn{Endpoint: "https://example.test", Credential: sentinel}

		if renderings := fiveFormRenderings(t, standIn); !anyFormContainsSentinel(renderings, sentinel) {
			t.Fatal("the unredacted stand-in did not leak the sentinel through any of the five forms, want at least one to — the positive control must be able to fail (S-APC-077)")
		}
	})
}

// TestClient_BareUnwrapped_NeverRendersCredential is the genuine D-5
// empirical decider (S-APC-078): does a bare, unwrapped *Client (or
// Client value) built with a sentinel credential disclose it under any of
// the five forms? production code never formats a bare *Client today
// (redactedProvider short-circuits at depth 0 first) — this test is what
// establishes the answer empirically rather than assuming it. A RED here
// is the empirical trigger for task 3.3a's production fix; a GREEN here
// means the property already holds and this test stays as the regression
// pin (task 3.3b), per D-5.
func TestClient_BareUnwrapped_NeverRendersCredential(t *testing.T) {
	t.Parallel()

	// Runtime-assembled — see TestConfig_NeverRendersCredential's own
	// comment for why (AI-36 WU-5's widened scan).
	sentinel := "s" + "k" + "-ai36-client-level-sentinel-fedcba654321"

	c, err := openaicompat.New(openaicompat.Config{
		Endpoint:   "https://example.test",
		Credential: openaicompat.NewCredential(sentinel),
	})
	if err != nil {
		t.Fatalf("openaicompat.New() error = %v, want nil", err)
	}

	t.Run("*Client (pointer)", func(t *testing.T) {
		t.Parallel()
		assertNoneContainSentinel(t, fiveFormRenderings(t, c), sentinel)
	})

	t.Run("Client (value, dereferenced)", func(t *testing.T) {
		t.Parallel()
		assertNoneContainSentinel(t, fiveFormRenderings(t, *c), sentinel)
	})
}
