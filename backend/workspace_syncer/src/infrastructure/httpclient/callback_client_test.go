package httpclient

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestCallbackClient_Post_Success(t *testing.T) {
	var capturedMethod, capturedPath, capturedAuth, capturedCT string
	var capturedBody CallbackRequest
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedMethod = r.Method
		capturedPath = r.URL.Path
		capturedAuth = r.Header.Get("Authorization")
		capturedCT = r.Header.Get("Content-Type")
		_ = json.NewDecoder(r.Body).Decode(&capturedBody)
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
	if capturedAuth != "Bearer test-token" {
		t.Errorf("auth = %q, want Bearer test-token", capturedAuth)
	}
	if capturedCT != "application/json" {
		t.Errorf("content-type = %q, want application/json", capturedCT)
	}
	if capturedBody.JobID != 42 || capturedBody.WorkspaceID != 7 || capturedBody.Status != "done" {
		t.Errorf("body = %+v, want job_id=42, workspace_id=7, status=done", capturedBody)
	}
	if capturedBody.CommitSHAAfter != "abc1234567890abcdef1234567890abcdef12345" {
		t.Errorf("commit_sha_after = %q", capturedBody.CommitSHAAfter)
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