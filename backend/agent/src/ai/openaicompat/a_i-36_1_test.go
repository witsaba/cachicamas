// AI-36 (WU-2, D-4) — the hostile-server case: a provider that echoes the
// caller's own authorization value and request body into a non-streaming
// response forces refuseNonStreamContentType's captured excerpt to carry
// them (stream.go:326-348). This file is the empirical decider for D-4
// (proposal.md § 4): if the credential vector bites, R-AEM-019 is
// promoted and stream.go gains a small, targeted redaction (task 2.4a);
// if it does not, the contingent requirement is recorded not-triggered
// with this file's own evidence (task 2.4b). The RED failure this test
// produces before any production fix lands IS the positive control
// S-CNF-081 requires: it proves the sweep can detect the leak before the
// fix exists (tasks.md task 2.3).
package openaicompat_test

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/cachicamas/backend/agent/src/agenttest/sweep"
	"github.com/cachicamas/backend/agent/src/ai"
	"github.com/cachicamas/backend/agent/src/ai/openaicompat"
)

// hostileEchoServer answers every request with 200, Content-Type:
// application/json (never the streaming media type, which forces
// refuseNonStreamContentType), and a body that echoes the caller's own
// Authorization header and request body verbatim — the adversarial shape
// S-CNF-081/D-4 requires: "a provider that echoes the caller's
// authorization value and the caller's request body back inside a
// non-streaming response".
func hostileEchoServer(t *testing.T) *httptest.Server {
	t.Helper()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = fmt.Fprintf(w, "auth=%s body=%s", r.Header.Get("Authorization"), body)
	}))
	t.Cleanup(server.Close)
	return server
}

// hostileContentTypeEchoServer answers every request with 200 and a
// Content-Type header whose PARAMETER echoes the caller's own
// Authorization header verbatim — the judgment-day variant of the
// hostile shape above: the media type is still application/json, so
// isStreamContentType refuses on the media type alone, while the
// parameter survives inside the raw header value the refusal stores.
// Before the fix, nonStreamContentType.Error() interpolated that raw
// value with %q and reproduced the credential (AI-36 JD round 1,
// finding 1).
func hostileContentTypeEchoServer(t *testing.T) *httptest.Server {
	t.Helper()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", `application/json; echo="`+r.Header.Get("Authorization")+`"`)
		w.WriteHeader(http.StatusOK)
		_, _ = io.WriteString(w, `{"ok":true}`)
	}))
	t.Cleanup(server.Close)
	return server
}

// mustSentinelPromptRequest builds a minimal valid ai.Request whose sole
// message carries the sentinel prompt text.
func mustSentinelPromptRequest(t *testing.T, sentinelPrompt string) ai.Request {
	t.Helper()
	part, err := ai.NewText(sentinelPrompt)
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
	return req
}

// renderEveryVerbAndUnwrapChain renders err under the plain, string,
// extended and Go-syntax verbs, then walks its errors.Unwrap chain
// (R-ATS-023: a caller unwraps the terminal failure to reach the
// excerpt), rendering each hop's own Error() plus the same four verbs.
// The result is keyed by a label naming the exact rendering, so a failing
// assertion names precisely which one leaked.
func renderEveryVerbAndUnwrapChain(err error) map[string]string {
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

// TestHostileServer_EchoesAuthorizationAndBody_CredentialNeverSurfacesInAnyRendering
// is the D-4 empirical decider AND S-CNF-081's positive control (tasks.md
// task 2.3): a RED here — before any production fix — proves the sweep
// detects the leak; the same assertions, kept in place after a fix lands
// (or confirmed already clean), become the regression pin.
func TestHostileServer_EchoesAuthorizationAndBody_CredentialNeverSurfacesInAnyRendering(t *testing.T) {
	t.Parallel()

	const sentinelCredential = "AI36-D4-HOSTILE-CREDENTIAL-CANARY-7c1e9a"
	const sentinelPrompt = "AI36-D4-HOSTILE-PROMPT-CANARY-2b8f4d"

	server := hostileEchoServer(t)

	client, err := openaicompat.New(openaicompat.Config{
		Endpoint:   server.URL,
		Credential: openaicompat.NewCredential(sentinelCredential),
		HTTPClient: server.Client(),
	})
	if err != nil {
		t.Fatalf("openaicompat.New() error = %v, want nil", err)
	}

	req := mustSentinelPromptRequest(t, sentinelPrompt)

	ch, streamErr := client.Stream(context.Background(), req)
	if streamErr == nil {
		t.Fatal("Stream() error = nil, want a non-streaming content-type refusal (the hostile server never answers text/event-stream)")
	}
	if ch != nil {
		t.Error("Stream() returned a non-nil channel alongside a non-nil error, want no carrier on a pre-handover refusal")
	}

	var failure *ai.Failure
	if !errors.As(streamErr, &failure) {
		t.Fatalf("errors.As(err, *ai.Failure) = false; err = %v, want a typed provider failure", streamErr)
	}
	if failure.Category() != ai.FailureCategoryMalformedResponse {
		t.Errorf("failure.Category() = %v, want %v (R-ATS-023's own content-type refusal category)", failure.Category(), ai.FailureCategoryMalformedResponse)
	}

	credentialDeny := []sweep.Entry{
		{Vector: "sentinel credential", Needle: []byte(sentinelCredential)},
		{Vector: "bearer-prefixed sentinel credential", Needle: []byte("Bearer " + sentinelCredential)},
	}
	if selfTestErr := sweep.SelfTest(credentialDeny); selfTestErr != nil {
		t.Fatalf("sweep.SelfTest(credential deny list) = %v, want nil (AD-2 positive control)", selfTestErr)
	}

	renderings := renderEveryVerbAndUnwrapChain(streamErr)
	// The chain must actually have been walked: D-4's leak lived at unwrap
	// depth 1 (nonStreamContentType's captured excerpt), so a rendering set
	// holding only the four top-level verbs would scan past the one place
	// that matters and still report clean. Requiring a depth-1 key is that
	// property; a length check is not, because the map is built with
	// exactly four literal entries before the walk begins.
	walkedToDepthOne := false
	for label := range renderings {
		if strings.HasPrefix(label, "unwrap-depth-1 ") {
			walkedToDepthOne = true
			break
		}
	}
	if !walkedToDepthOne {
		t.Fatalf("renderEveryVerbAndUnwrapChain produced no unwrap-depth-1 rendering out of %d — the Unwrap chain was not walked, so the scan below never reaches the depth D-4's credential leak lived at (S-CNF-081)", len(renderings))
	}

	for label, rendered := range renderings {
		if vector, found := sweep.Scan([]byte(rendered), credentialDeny); found {
			t.Errorf("rendering %q leaked the caller's own credential (vector=%q) via the hostile server's echo (D-4/S-CNF-081)", label, vector)
		}
	}

	// The prompt-body echo is a named residual (D-4, S-CNF-081's residual
	// clause), NOT asserted absent here: the excerpt is the provider's own
	// response, and suppressing its echo of the caller's own content would
	// defeat R-ATS-023's diagnostic purpose. This block only records, in
	// the test log, whether it was observed on this run.
	promptLeaked := false
	for _, rendered := range renderings {
		if strings.Contains(rendered, sentinelPrompt) {
			promptLeaked = true
			break
		}
	}
	t.Logf("named residual (S-CNF-081 item 2, D-4 item 4): prompt-body echo observed in a rendering = %v — the excerpt is the provider's own response; suppressing its echo of the caller's own content would defeat R-ATS-023's diagnostic purpose, so this is recorded, not fixed", promptLeaked)
}

// TestHostileServer_EchoesAuthorizationIntoContentTypeParameter_CredentialNeverSurfaces
// is the judgment-day round-1 finding-1 pin: a provider that echoes the
// caller's own Authorization value into a Content-Type PARAMETER (not the
// body) must not be able to make the refusal's rendered text reproduce
// it. isStreamContentType compares only the parsed media type, so the
// parameter rides along in the raw header value; the stored content type
// must therefore be redacted exactly as the body excerpt is, while the
// offending media type itself stays visible (R-ATS-023).
func TestHostileServer_EchoesAuthorizationIntoContentTypeParameter_CredentialNeverSurfaces(t *testing.T) {
	t.Parallel()

	const sentinelCredential = "AI36-JD1-CT-PARAM-CANARY-5e9d2b"

	server := hostileContentTypeEchoServer(t)

	client, err := openaicompat.New(openaicompat.Config{
		Endpoint:   server.URL,
		Credential: openaicompat.NewCredential(sentinelCredential),
		HTTPClient: server.Client(),
	})
	if err != nil {
		t.Fatalf("openaicompat.New() error = %v, want nil", err)
	}

	ch, streamErr := client.Stream(context.Background(), mustSentinelPromptRequest(t, "AI36-JD1-CT-PARAM-PROMPT-8a3c1f"))
	if streamErr == nil {
		t.Fatal("Stream() error = nil, want a non-streaming content-type refusal (the hostile server never answers text/event-stream)")
	}
	if ch != nil {
		t.Error("Stream() returned a non-nil channel alongside a non-nil error, want no carrier on a pre-handover refusal")
	}

	var failure *ai.Failure
	if !errors.As(streamErr, &failure) {
		t.Fatalf("errors.As(err, *ai.Failure) = false; err = %v, want a typed provider failure", streamErr)
	}
	if failure.Category() != ai.FailureCategoryMalformedResponse {
		t.Errorf("failure.Category() = %v, want %v (R-ATS-023's own content-type refusal category)", failure.Category(), ai.FailureCategoryMalformedResponse)
	}

	credentialDeny := []sweep.Entry{
		{Vector: "sentinel credential", Needle: []byte(sentinelCredential)},
		{Vector: "bearer-prefixed sentinel credential", Needle: []byte("Bearer " + sentinelCredential)},
	}
	if selfTestErr := sweep.SelfTest(credentialDeny); selfTestErr != nil {
		t.Fatalf("sweep.SelfTest(credential deny list) = %v, want nil (AD-2 positive control)", selfTestErr)
	}

	renderings := renderEveryVerbAndUnwrapChain(streamErr)
	walkedToDepthOne := false
	for label := range renderings {
		if strings.HasPrefix(label, "unwrap-depth-1 ") {
			walkedToDepthOne = true
			break
		}
	}
	if !walkedToDepthOne {
		t.Fatalf("renderEveryVerbAndUnwrapChain produced no unwrap-depth-1 rendering out of %d — the Unwrap chain was not walked, so the scan below never reaches the depth the content-type leak lives at", len(renderings))
	}

	sawOffendingMediaType := false
	for label, rendered := range renderings {
		if vector, found := sweep.Scan([]byte(rendered), credentialDeny); found {
			t.Errorf("rendering %q leaked the caller's own credential (vector=%q) via the hostile Content-Type parameter echo (JD round 1, finding 1)", label, vector)
		}
		if strings.Contains(rendered, "application/json") {
			sawOffendingMediaType = true
		}
	}
	// R-ATS-023 requires the refusal to name the offending type: redaction
	// must remove only the credential occurrences, never the media type.
	if !sawOffendingMediaType {
		t.Error("no rendering names the offending media type application/json — redaction must strip only the credential, not the content type's diagnostic value (R-ATS-023)")
	}
}
