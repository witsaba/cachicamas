// Package httpclient contains the HTTP client that the
// workspace_syncer uses to post outcomes back to
// database_administrator's /internal/sync-callback endpoint. See
// design.md §5 (Cross-service contract — callback shape).
//
// SECURITY: the bearer token is captured by value at construction
// time. The client is the ONLY caller of this package's Post
// method; the rest of the service never sees the token.
package httpclient

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// DefaultTimeoutSeconds bounds the HTTP round-trip to
// database_administrator. The database_administrator is a
// control-plane service: its responses should be sub-second. 30s
// is a generous ceiling that catches stuck connections without
// holding up the workspace_syncer's goroutine.
const DefaultTimeoutSeconds = 30

// CallbackRequest is the body posted to
// database_administrator's POST /internal/sync-callback endpoint.
// See design.md §5 — Cross-service contract.
type CallbackRequest struct {
	JobID          int64  `json:"job_id"`
	WorkspaceID    int64  `json:"workspace_id"`
	Status         string `json:"status"` // "done" | "failed"
	CommitSHAAfter string `json:"commit_sha_after,omitempty"`
	ErrorCode      string `json:"error_code,omitempty"`
	ErrorMessage   string `json:"error_message,omitempty"`
	StartedAt      string `json:"started_at,omitempty"`
	FinishedAt     string `json:"finished_at,omitempty"`
}

// CallbackClient posts CallbackRequest bodies to database_administrator.
// It is the only outbound HTTP client in the workspace_syncer.
type CallbackClient struct {
	baseURL string
	token   string
	client  *http.Client
}

// NewCallbackClient constructs a CallbackClient. The baseURL is
// the database_administrator's root (e.g.
// "http://database_administrator:8080"); the client appends
// "/internal/sync-callback" itself.
func NewCallbackClient(baseURL, token string) *CallbackClient {
	return &CallbackClient{
		baseURL: baseURL,
		token:   token,
		client: &http.Client{
			Timeout: DefaultTimeoutSeconds * time.Second,
		},
	}
}

// Post posts the callback body to database_administrator. Returns
// nil on a 2xx response; returns a non-nil error on a non-2xx
// response or a network error. The error wraps the HTTP status code
// so the application layer can log it.
//
// The body is encoded as JSON. The response body is discarded
// (read + closed) so the underlying connection can be reused.
func (c *CallbackClient) Post(ctx context.Context, req CallbackRequest) error {
	body, err := json.Marshal(req)
	if err != nil {
		return fmt.Errorf("callback: marshal body: %w", err)
	}

	url := c.baseURL + "/internal/sync-callback"
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("callback: build request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+c.token)

	resp, err := c.client.Do(httpReq)
	if err != nil {
		return fmt.Errorf("callback: post: %w", err)
	}
	defer func() {
		_, _ = io.Copy(io.Discard, resp.Body)
		_ = resp.Body.Close()
	}()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("callback: non-2xx response: status=%d", resp.StatusCode)
	}
	return nil
}