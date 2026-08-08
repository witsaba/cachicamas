// AI-36 judgment-day round 1 — regression pins for the confirmed findings
// against redactCredential and the capture/redaction interaction
// (stream.go). Internal package by load-bearing necessity: the
// truncation-edge case positions its hostile echo against the REAL
// captureLimit (capture.go), and the growth-bound case drives the
// unexported redactCredential directly — neither is expressible from
// package openaicompat_test without duplicating the constant it must
// stay in lockstep with.
package openaicompat

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/cachicamas/backend/agent/src/ai"
)

// jdRenderEveryVerbAndUnwrapChain mirrors a_i-36_1_test.go's
// renderEveryVerbAndUnwrapChain (which lives in the external test package
// and cannot be shared): it renders err under the plain, string, extended
// and Go-syntax verbs, then walks its errors.Unwrap chain, rendering each
// hop's own Error() plus the same four verbs, keyed by a label naming the
// exact rendering.
func jdRenderEveryVerbAndUnwrapChain(err error) map[string]string {
	out := map[string]string{
		"top-level %v":  fmt.Sprintf("%v", err),
		"top-level %s":  fmt.Sprintf("%s", err), //nolint:staticcheck // intentional verb coverage
		"top-level %+v": fmt.Sprintf("%+v", err),
		"top-level %#v": fmt.Sprintf("%#v", err),
	}
	depth := 0
	for cause := errors.Unwrap(err); cause != nil; cause = errors.Unwrap(cause) {
		depth++
		prefix := fmt.Sprintf("unwrap-depth-%d", depth)
		out[prefix+" .Error()"] = cause.Error()
		out[prefix+" %v"] = fmt.Sprintf("%v", cause)
		out[prefix+" %s"] = fmt.Sprintf("%s", cause) //nolint:staticcheck // intentional verb coverage
		out[prefix+" %+v"] = fmt.Sprintf("%+v", cause)
		out[prefix+" %#v"] = fmt.Sprintf("%#v", cause)
	}
	return out
}

// jdStreamRefusal drives one Stream call against server with the given
// credential and returns the refusal error, asserting the pre-handover
// refusal shape the hostile servers in this file always force.
func jdStreamRefusal(t *testing.T, server *httptest.Server, credential string) error {
	t.Helper()

	client, err := New(Config{
		Endpoint:   server.URL,
		Credential: NewCredential(credential),
		HTTPClient: server.Client(),
	})
	if err != nil {
		t.Fatalf("New() error = %v, want nil", err)
	}

	part, err := ai.NewText("JD1-PROMPT-fixed")
	if err != nil {
		t.Fatalf("ai.NewText() error = %v, want nil", err)
	}
	msg, err := ai.NewMessage(ai.RoleUser, part)
	if err != nil {
		t.Fatalf("ai.NewMessage() error = %v, want nil", err)
	}
	req, err := ai.NewRequest("gpt-4o", []ai.Message{msg})
	if err != nil {
		t.Fatalf("ai.NewRequest() error = %v, want nil", err)
	}

	ch, streamErr := client.Stream(context.Background(), req)
	if streamErr == nil {
		t.Fatal("Stream() error = nil, want a non-streaming content-type refusal")
	}
	if ch != nil {
		t.Error("Stream() returned a non-nil channel alongside a non-nil error, want no carrier on a pre-handover refusal")
	}
	return streamErr
}

// TestHostileServer_CredentialStraddlesCaptureLimit_NoTokenPrefixSurvivesTruncationEdge
// is the judgment-day round-1 finding-2 pin: captureBody cuts the body at
// captureLimit with no credential awareness, and refuseNonStreamContentType
// captures FIRST and redacts SECOND — so a hostile server that positions
// its Authorization echo across that boundary used to leave an
// attacker-chosen PREFIX of the raw bearer token at the truncation edge,
// where whole-needle replacement cannot see it. The module's own doctrine
// treats a prefix of a secret as a secret (ai/completion.go, the smoke
// sweep's 4-character secret-prefix vector), so no fragment of 4 bytes or
// more may survive.
func TestHostileServer_CredentialStraddlesCaptureLimit_NoTokenPrefixSurvivesTruncationEdge(t *testing.T) {
	t.Parallel()

	// Assembled at runtime (credential-scan posture, design.md AD-4).
	sentinelCredential := "AI36-JD1-TRUNC-EDGE-CANARY-" + "9f2eQ7"

	// Position the echo so the captureLimit cut lands mid-token, leaving
	// exactly survivingTokenBytes raw token bytes at the truncation edge.
	const survivingTokenBytes = 12
	padLen := captureLimit - len("auth=") - len("Bearer ") - survivingTokenBytes
	if padLen <= 0 {
		t.Fatalf("padLen = %d, want > 0 — captureLimit too small for this fixture's arithmetic", padLen)
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(bytes.Repeat([]byte("p"), padLen))
		// The echo starts exactly survivingTokenBytes + len("Bearer ")
		// + len("auth=") bytes before the cut; everything past the cut
		// is drained, never captured.
		_, _ = fmt.Fprintf(w, "auth=%s past-the-limit-tail", r.Header.Get("Authorization"))
	}))
	t.Cleanup(server.Close)

	streamErr := jdStreamRefusal(t, server, sentinelCredential)

	renderings := jdRenderEveryVerbAndUnwrapChain(streamErr)

	// Fixture self-checks: the excerpt must really have overflowed (the
	// truncation marker is present iff the body exceeded captureLimit),
	// and the echo's own prefix must have reached the captured region —
	// otherwise the cut landed in the padding and this test would pass
	// vacuously without ever putting a token fragment at the edge.
	sawMarker, sawEchoStart := false, false
	for _, rendered := range renderings {
		if strings.Contains(rendered, truncationMarker) {
			sawMarker = true
		}
		if strings.Contains(rendered, "auth=") {
			sawEchoStart = true
		}
	}
	if !sawMarker {
		t.Fatal("no rendering carries the truncation marker — the fixture body did not overflow captureLimit, so the truncation edge was never exercised")
	}
	if !sawEchoStart {
		t.Fatal("no rendering carries the echo prefix \"auth=\" — the cut landed before the echo, so no token fragment ever reached the truncation edge")
	}

	// The surviving fragment is sentinelCredential[:survivingTokenBytes];
	// any surviving prefix of 4 bytes or more necessarily contains the
	// token's first 4 bytes, so that one needle covers every fragment
	// length this fixture can produce.
	tokenPrefix := sentinelCredential[:4]
	for label, rendered := range renderings {
		if strings.Contains(rendered, tokenPrefix) {
			t.Errorf("rendering %q retains a prefix of the raw bearer token at the truncation edge (needle %q) — a prefix of a secret is still a secret (JD round 1, finding 2)", label, tokenPrefix)
		}
	}
}

// TestHostileServer_CaseFoldsItsEcho_FoldedCredentialNeverSurfaces is the
// judgment-day round-1 finding-3 pin: redactCredential used to match both
// needles byte-exactly, so a hostile server that case-folds its echo
// (RFC 7235 already permits any casing of the scheme keyword, and the
// echoing server chooses its own casing for the rest) evaded both — and
// the existing regression pin could not see the gap because its deny list
// was built from the same exact-case sentinel as the plant. Here the
// plant is the server's own strings.ToLower of the Authorization header
// while the needles are the DERIVED folded forms, never the exact-case
// sentinel itself.
func TestHostileServer_CaseFoldsItsEcho_FoldedCredentialNeverSurfaces(t *testing.T) {
	t.Parallel()

	// Deliberately mixed-case, assembled at runtime (credential-scan
	// posture, design.md AD-4).
	sentinelCredential := "AI36-JD1-CaSe-FoLd-CANARY-" + "Qz7Xk"
	foldedToken := strings.ToLower(sentinelCredential)
	if foldedToken == sentinelCredential {
		t.Fatal("the sentinel credential carries no uppercase byte — plant and needle would coincide and the case-folding gap this test pins would be invisible (JD round 1, finding 3)")
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		// The whole header value is folded: scheme AND token. The scheme
		// keyword has defined case-insensitivity; the token bytes are the
		// caller's own credential differing only by ASCII case.
		_, _ = fmt.Fprintf(w, "auth=%s", strings.ToLower(r.Header.Get("Authorization")))
	}))
	t.Cleanup(server.Close)

	streamErr := jdStreamRefusal(t, server, sentinelCredential)

	renderings := jdRenderEveryVerbAndUnwrapChain(streamErr)
	for label, rendered := range renderings {
		if strings.Contains(rendered, foldedToken) {
			t.Errorf("rendering %q retains the case-folded token %q — a case-variant of a secret is still that secret's material (JD round 1, finding 3)", label, foldedToken)
		}
	}
}

// TestRedactCredential_ShortCredentialNeverGrowsExcerptPastCaptureBound is
// the judgment-day round-1 finding-5 pin: NewCredential/New enforce only
// non-empty (client.go), so a 1-byte credential is a valid live value —
// and substituting every occurrence of it in a captureLimit-sized excerpt
// with the longer placeholder used to grow the result to roughly ten
// times AI-32.5's size bound, while the surrounding comment merely
// asserted the bound was preserved. The invariant must be enforced, not
// asserted: no redacted excerpt may exceed the largest capture the
// capture path itself can emit.
func TestRedactCredential_ShortCredentialNeverGrowsExcerptPastCaptureBound(t *testing.T) {
	t.Parallel()

	// The placeholder's real length — the growth-argument arithmetic in
	// stream.go's comments reasons about this exact number, so pin it
	// (the pre-fix comment claimed 11; it is 10).
	if len(credentialRedactedPlaceholder) != 10 {
		t.Errorf("len(credentialRedactedPlaceholder) = %d, want 10 — stream.go's growth-bound reasoning cites this byte count (JD round 1, finding 5)", len(credentialRedactedPlaceholder))
	}

	excerpt := bytes.Repeat([]byte("a"), captureLimit)
	got := redactCredential(excerpt, NewCredential("a"))
	bound := captureLimit + len(truncationMarker)
	if len(got) > bound {
		t.Errorf("len(redactCredential(worst-case excerpt, 1-byte credential)) = %d, want <= %d (captureLimit + len(truncationMarker), AI-32.5's own maximum) — redaction must preserve the capture bound, not merely assert it (JD round 1, finding 5)", len(got), bound)
	}
}
