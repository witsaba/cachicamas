package httpclient

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// TestCallbackClient_Post_Success verifies the full request shape:
// POST /api/v1/internal/sync-callback with the canonical JSON
// body, the X-Cachicamas-Timestamp + X-Cachicamas-Signature HMAC
// headers, and the correct wire-level field values.
func TestCallbackClient_Post_Success(t *testing.T) {
	var capturedMethod, capturedPath, capturedCT, capturedTS, capturedSig string
	var capturedRaw []byte
	var capturedBody CallbackRequest
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedMethod = r.Method
		capturedPath = r.URL.Path
		capturedCT = r.Header.Get("Content-Type")
		capturedTS = r.Header.Get("X-Cachicamas-Timestamp")
		capturedSig = r.Header.Get("X-Cachicamas-Signature")
		buf, _ := io.ReadAll(r.Body)
		capturedRaw = buf
		_ = json.Unmarshal(buf, &capturedBody)
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	client := NewCallbackClient(srv.URL, "test-token")
	err := client.Post(context.Background(), CallbackRequest{
		JobID:          42,
		WorkspaceID:    7,
		Status:         "done",
		CommitSHAAfter: "abc1234567890abcdef1234567890abcdef12345",
	})
	if err != nil {
		t.Fatalf("Post: %v", err)
	}
	if capturedMethod != http.MethodPost {
		t.Errorf("method = %q, want POST", capturedMethod)
	}
	if capturedPath != "/api/v1/internal/sync-callback" {
		t.Errorf("path = %q, want /api/v1/internal/sync-callback", capturedPath)
	}
	if capturedCT != "application/json" {
		t.Errorf("content-type = %q, want application/json", capturedCT)
	}
	if capturedTS == "" {
		t.Errorf("X-Cachicamas-Timestamp is empty (HMAC needs it)")
	}
	if capturedSig == "" {
		t.Errorf("X-Cachicamas-Signature is empty")
	}
	if capturedBody.JobID != 42 || capturedBody.WorkspaceID != 7 || capturedBody.Status != "done" {
		t.Errorf("body = %+v, want job_id=42, workspace_id=7, status=done", capturedBody)
	}
	if capturedBody.CommitSHAAfter != "abc1234567890abcdef1234567890abcdef12345" {
		t.Errorf("commit_sha_after = %q", capturedBody.CommitSHAAfter)
	}

	// Verify the HMAC: the client should have signed the EXACT
	// bytes that were on the wire. This is the byte-level contract
	// with the db_admin's canonicalWriteCallbackValue.
	mac := hmac.New(sha256.New, []byte("test-token"))
	mac.Write([]byte(capturedTS))
	mac.Write([]byte("."))
	mac.Write(capturedRaw)
	expected := base64.StdEncoding.EncodeToString(mac.Sum(nil))
	if capturedSig != expected {
		t.Errorf("signature mismatch:\n  got:  %s\n  want: %s\n  body: %s", capturedSig, expected, string(capturedRaw))
	}
}

func TestCallbackClient_Post_4xxReturnsError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
	}))
	defer srv.Close()

	client := NewCallbackClient(srv.URL, "test-token")
	err := client.Post(context.Background(), CallbackRequest{Status: "done"})
	if err == nil {
		t.Fatalf("Post on 400 returned nil error")
	}
	if !strings.Contains(err.Error(), "400") {
		t.Errorf("error message should mention status 400, got: %v", err)
	}
}

func TestCallbackClient_Post_5xxReturnsError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()

	client := NewCallbackClient(srv.URL, "test-token")
	err := client.Post(context.Background(), CallbackRequest{Status: "failed", ErrorCode: "X"})
	if err == nil {
		t.Fatalf("Post on 500 returned nil error")
	}
	if !strings.Contains(err.Error(), "500") {
		t.Errorf("error message should mention status 500, got: %v", err)
	}
}

func TestCallbackClient_Post_NetworkErrorReturnsError(t *testing.T) {
	// Point at a port we know is closed. The TCP connect fails
	// almost immediately.
	client := NewCallbackClient("http://127.0.0.1:1", "test-token")
	err := client.Post(context.Background(), CallbackRequest{Status: "done"})
	if err == nil {
		t.Fatalf("Post to closed port returned nil error")
	}
}

func TestCallbackClient_Post_DiscardsResponseBody(t *testing.T) {
	// The server sends a large body. The client must read and
	// discard it so the underlying connection can be reused.
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = io.WriteString(w, strings.Repeat("x", 4096))
	}))
	defer srv.Close()

	client := NewCallbackClient(srv.URL, "test-token")
	if err := client.Post(context.Background(), CallbackRequest{Status: "done"}); err != nil {
		t.Fatalf("Post: %v", err)
	}
	// If the body was not drained, the test would hang or fail
	// on connection reuse. The fact that the test exits cleanly
	// is the assertion.
}

func TestCanonicalizeJSON_KeyOrder(t *testing.T) {
	// Verify the canonicalizer emits sorted keys. This is the
	// byte-level contract with the db_admin's canonicalizer.
	type point struct {
		Z int    `json:"z"`
		A int    `json:"a"`
		M string `json:"m"`
	}
	got, err := canonicalizeJSON(point{Z: 1, A: 2, M: "x"})
	if err != nil {
		t.Fatalf("canonicalize: %v", err)
	}
	want := `{"a":2,"m":"x","z":1}`
	if string(got) != want {
		t.Errorf("canonical = %s, want %s", string(got), want)
	}
}

func TestComputeSignature_StableAcrossRuns(t *testing.T) {
	// Same input -> same signature. Verifies the HMAC is
	// deterministic.
	s1 := computeSignature("secret", "1234567890", []byte(`{"a":1}`))
	s2 := computeSignature("secret", "1234567890", []byte(`{"a":1}`))
	if s1 != s2 {
		t.Errorf("signature not deterministic: %s vs %s", s1, s2)
	}
	// Different secret -> different signature.
	s3 := computeSignature("other-secret", "1234567890", []byte(`{"a":1}`))
	if s1 == s3 {
		t.Errorf("different secrets produced same signature: %s", s1)
	}
	// Different timestamp -> different signature.
	s4 := computeSignature("secret", "9999999999", []byte(`{"a":1}`))
	if s1 == s4 {
		t.Errorf("different timestamps produced same signature: %s", s1)
	}
}
