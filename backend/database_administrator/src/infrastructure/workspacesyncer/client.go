// Package workspacesyncer — the database_administrator-side HTTP
// client that talks to the workspace_syncer service. This is the
// only file in database_administrator that calls the syncer.
//
// Why a dedicated package: the syncer's wire contract is its own
// shape (POST /internal/clone-and-validate, bearer-token auth,
// async 202 response). Co-locating that contract in a single file
// keeps the database_administrator's composition root tidy and
// gives the test suite a small, focused surface to mock with
// httptest.
//
// Auth: INTERNAL_SERVICE_TOKEN bearer header. The syncer uses
// crypto/subtle.ConstantTimeCompare to verify; we send the
// header value unchanged. The token is never logged.
package workspacesyncer

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// StartSyncRequest is the wire body for POST
// /internal/clone-and-validate. The field tags are locked (work
// with the syncer's cloneRequestBody struct).
type StartSyncRequest struct {
	JobID         int64  `json:"job_id"`
	WorkspaceID   int64  `json:"workspace_id"`
	Owner         string `json:"owner"`
	Repo          string `json:"repo"`
	DefaultBranch string `json:"default_branch"`
	OAuthToken    string `json:"oauth_token"`
}

// StartSyncResponse is the wire body the syncer returns on 202.
// The database_administrator uses job_id + status to confirm the
// job is in flight; the actual outcome is delivered later via the
// internal callback.
type StartSyncResponse struct {
	JobID  int64  `json:"job_id"`
	Status string `json:"status"`
}

// Client is the HTTP transport to the workspace_syncer. It carries
// no per-request state; all methods take the per-request data
// (oauth token, etc.) from the call site.
type Client struct {
	BaseURL    string
	BearerToken string
	HTTPClient  *http.Client
}

// NewClient returns a Client wired to baseURL with a 30-second
// timeout (the syncer's clone-and-validate returns 202 immediately
// and does the actual work in a goroutine, so 30s is generous
// headroom for the HTTP round-trip). bearerToken is the shared
// INTERNAL_SERVICE_TOKEN (docker network trust + bearer; see
// adr/workspace-syncer-internal-auth).
func NewClient(baseURL, bearerToken string) *Client {
	return &Client{
		BaseURL:    baseURL,
		BearerToken: bearerToken,
		HTTPClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// StartSync POSTs to /internal/clone-and-validate. The 202
// response signals the syncer has accepted the job; the use case
// (or the workspace auto-enqueue path) does NOT block on the
// outcome — the callback delivers the result.
//
// Error model:
//   - *UnauthorizedError on 401 (token mismatch; the syncer is
//     refusing to talk to us).
//   - *UnavailableError on transport failures (TCP refused,
//     timeout). Caller should log + mark the job failed so the UI
//     can show the failure and the user can retry.
//   - *ValidationError on 4xx (4xx other than 401: schema or
//     permission rejection from the syncer).
//   - wrapped error on unexpected 5xx (the syncer crashed mid-way;
//     the job is best-effort reaped by the workspace_syncer's
//     startup sweep).
func (c *Client) StartSync(ctx context.Context, req StartSyncRequest) (*StartSyncResponse, error) {
	body, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("workspacesyncer.StartSync: marshal: %w", err)
	}
	u := c.BaseURL + "/internal/clone-and-validate"
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, u, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("workspacesyncer.StartSync: build request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+c.BearerToken)
	httpReq.Header.Set("Accept", "application/json")

	resp, err := c.HTTPClient.Do(httpReq)
	if err != nil {
		return nil, &UnavailableError{Cause: err}
	}
	defer func() { _ = resp.Body.Close() }()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("workspacesyncer.StartSync: read body: %w", err)
	}

	switch resp.StatusCode {
	case http.StatusAccepted:
		var out StartSyncResponse
		if err := json.Unmarshal(respBody, &out); err != nil {
			return nil, fmt.Errorf("workspacesyncer.StartSync: parse 202 body: %w", err)
		}
		return &out, nil
	case http.StatusUnauthorized:
		return nil, &UnauthorizedError{Cause: fmt.Errorf("http %d", resp.StatusCode)}
	case http.StatusBadRequest, http.StatusUnprocessableEntity:
		return nil, &ValidationError{Cause: fmt.Errorf("http %d: %s", resp.StatusCode, string(respBody))}
	default:
		return nil, &UnavailableError{Cause: fmt.Errorf("http %d: %s", resp.StatusCode, string(respBody))}
	}
}

// UnauthorizedError is returned when the syncer rejected the
// INTERNAL_SERVICE_TOKEN (401).
type UnauthorizedError struct {
	Cause error
}

func (e *UnauthorizedError) Error() string {
	if e.Cause != nil {
		return "workspacesyncer unauthorized: " + e.Cause.Error()
	}
	return "workspacesyncer unauthorized"
}
func (e *UnauthorizedError) Unwrap() error { return e.Cause }

// ValidationError is returned for 4xx other than 401 (schema
// mismatch, missing field). The use case layer logs the cause and
// marks the sync_job as failed.
type ValidationError struct {
	Cause error
}

func (e *ValidationError) Error() string {
	if e.Cause != nil {
		return "workspacesyncer validation: " + e.Cause.Error()
	}
	return "workspacesyncer validation error"
}
func (e *ValidationError) Unwrap() error { return e.Cause }

// UnavailableError is returned on transport failures (TCP refused,
// timeout) or unexpected 5xx. The use case logs the cause and
// marks the sync_job as failed.
type UnavailableError struct {
	Cause error
}

func (e *UnavailableError) Error() string {
	if e.Cause != nil {
		return "workspacesyncer unavailable: " + e.Cause.Error()
	}
	return "workspacesyncer unavailable"
}
func (e *UnavailableError) Unwrap() error { return e.Cause }