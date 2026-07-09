// sync_stream_handler_test.go — TDD coverage for the SSE
// handler. The handler is the load-bearing piece of the
// polling-to-SSE migration; the tests pin the wire format and
// the delta-only behavior.
package httpiface

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/labstack/echo/v5"

	"github.com/cachicamas/backend/database_administrator/src/domain"
)

// fakeStreamer is a controllable SyncJobStreamer. The Get call
// returns the value of `next` (atomic via mu); the test can
// re-set `next` mid-stream to simulate a job transition.
type fakeStreamer struct {
	mu    sync.Mutex
	next  *domain.SyncJob
	calls int
	err   error
	delay time.Duration
}

func (f *fakeStreamer) GetLatestSyncJob(ctx context.Context, workspaceID int64) (*domain.SyncJob, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.calls++
	if f.delay > 0 {
		time.Sleep(f.delay)
	}
	if f.err != nil {
		return nil, f.err
	}
	return f.next, nil
}

func (f *fakeStreamer) set(j *domain.SyncJob) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.next = j
}

func discardLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}

// readSSE reads from the test response body line-by-line until
// the timeout expires (or n events are received, whichever
// comes first). Returns the data lines it found.
func readSSE(t *testing.T, body io.Reader, n int, timeout time.Duration) []string {
	t.Helper()
	out := make([]string, 0, n)
	deadline := time.NewTimer(timeout)
	defer deadline.Stop()
	br := make([]byte, 4096)
	var buf strings.Builder
	for len(out) < n {
		select {
		case <-deadline.C:
			return out
		default:
		}
		n2, _ := body.Read(br)
		if n2 > 0 {
			buf.Write(br[:n2])
			for {
				line, rest, ok := splitFirstLine(buf.String())
				if !ok {
					break
				}
				buf.Reset()
				buf.WriteString(rest)
				if strings.HasPrefix(line, "data: ") {
					out = append(out, strings.TrimPrefix(line, "data: "))
				}
			}
		} else {
			// Small sleep to avoid busy-looping when the body
			// has no data yet.
			time.Sleep(20 * time.Millisecond)
		}
	}
	return out
}

func splitFirstLine(s string) (string, string, bool) {
	idx := strings.IndexAny(s, "\n")
	if idx < 0 {
		return "", s, false
	}
	return s[:idx], s[idx+1:], true
}

// newSSEServer constructs a minimal test server that exposes
// the SSE handler on /workspaces/:id/sync/stream, behind a stub
// IdentityFromCookie middleware (so the auth check passes).
func newSSEServer(streamer SyncJobStreamer) *httptest.Server {
	logger := discardLogger()
	e, g := newSSERootAndGroup()
	h := NewSyncStreamHandler(streamer, logger)
	RegisterSyncStreamRoute(g, h)
	// *echo.Echo is the http.Handler; *echo.Group is not. The
	// root Echo is the http server. The group's routes are
	// registered on the root.
	return httptest.NewServer(e)
}

// newSSERootAndGroup returns (root, group) so the test can
// register routes on the auth-protected group while serving
// the root as the http.Handler. Implemented in
// sse_test_helper.go to share the same Echo + auth-stub
// middleware as the other test fixtures.

func TestSSE_InitialEventCarriesCurrentJob(t *testing.T) {
	streamer := &fakeStreamer{
		next: &domain.SyncJob{
			ID:          42,
			WorkspaceID: 7,
			Status:      "pending",
		},
	}
	srv := newSSEServer(streamer)
	defer srv.Close()

	req, _ := http.NewRequest(http.MethodGet, srv.URL+"/workspaces/7/sync/stream", nil)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("GET: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200", resp.StatusCode)
	}
	if got := resp.Header.Get("Content-Type"); got != "text/event-stream" {
		t.Errorf("Content-Type = %q, want text/event-stream", got)
	}

	events := readSSE(t, resp.Body, 1, 2*time.Second)
	if len(events) < 1 {
		t.Fatalf("got %d events, want >= 1 (the initial snapshot)", len(events))
	}
	var got syncResponse
	if err := json.Unmarshal([]byte(events[0]), &got); err != nil {
		t.Fatalf("unmarshal event[0] %q: %v", events[0], err)
	}
if got.JobID != 42 || got.Status != "pending" {
    		t.Errorf("event[0] = %+v, want job_id=42 status=pending", got)
    	}
    }

// TestSSE_EmitsNullThenClosesWhenNoJobExists is the
// regression test for the 2nd UAT bug (2026-07-08 clean
// rebuild): the SSE stream used to send ": empty" keepalive
// comments forever when no sync_job existed for the workspace.
// That wasted a connection AND confused the user (the stream
// looked broken because nothing happened).
//
// Correct behavior:
//   - The handler queries GetLatestSyncJob once.
//   - If the result is nil (no job ever enqueued), emit ONE
//     event with job_id=0 (or a null marker), then close the
//     stream. No keepalives.
//   - The client receives the null event, renders the
//     "Sync now" CTA, and does NOT reconnect.
//
// The fakeStreamer returns nil to simulate "no job yet".
func TestSSE_EmitsNullThenClosesWhenNoJobExists(t *testing.T) {
	streamer := &fakeStreamer{
		next: nil, // no sync_job exists for this workspace
	}
	srv := newSSEServer(streamer)
	defer srv.Close()

	req, _ := http.NewRequest(http.MethodGet, srv.URL+"/workspaces/99/sync/stream", nil)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("GET: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200", resp.StatusCode)
	}

	// Read events with a tight timeout. We expect exactly
	// ONE event (the null marker) and then the server should
	// close the connection (EOF on the body).
	events := readSSE(t, resp.Body, 1, 2*time.Second)
	if len(events) != 1 {
		t.Fatalf("got %d events, want exactly 1 (null marker); no keepalives allowed when no job exists", len(events))
	}

	var got syncResponse
	if err := json.Unmarshal([]byte(events[0]), &got); err != nil {
		t.Fatalf("unmarshal event[0] %q: %v", events[0], err)
	}
	if got.JobID != 0 {
		t.Errorf("event[0].job_id = %d, want 0 (null marker for no-job-yet)", got.JobID)
	}

	// The handler MUST close the stream after the null event
	// (so the browser can decide whether to reconnect).
	// Reading should hit EOF within a tight budget.
	body, _ := io.ReadAll(resp.Body)
	// Any further read should be empty.
	if len(body) > 0 {
		t.Errorf("expected EOF after null event; got extra bytes: %q", body)
	}
}

func TestSSE_OnlyEmitsOnChange(t *testing.T) {
	streamer := &fakeStreamer{
		next: &domain.SyncJob{ID: 1, Status: "pending"},
	}
	srv := newSSEServer(streamer)
	defer srv.Close()

	req, _ := http.NewRequest(http.MethodGet, srv.URL+"/workspaces/7/sync/stream", nil)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("GET: %v", err)
	}
	defer resp.Body.Close()

	// Read the initial event. The next poll (within ssePollInterval)
	// MUST NOT re-emit if the state is unchanged.
	events := readSSE(t, resp.Body, 1, 1*time.Second)
	if len(events) < 1 {
		t.Fatalf("got %d initial events, want 1", len(events))
	}

	// Wait for the second poll to fire (ssePollInterval = 750ms).
	// We do NOT change the streamer state, so no second event
	// should be emitted.
	time.Sleep(2 * ssePollInterval)
	more := readSSE(t, resp.Body, 1, 100*time.Millisecond)
	if len(more) > 0 {
		t.Errorf("got %d duplicate events on no-op polls, want 0 (delta-only)", len(more))
	}
}

func TestSSE_EmitsOnStateChange(t *testing.T) {
	streamer := &fakeStreamer{
		next: &domain.SyncJob{ID: 1, Status: "pending"},
	}
	srv := newSSEServer(streamer)
	defer srv.Close()

	req, _ := http.NewRequest(http.MethodGet, srv.URL+"/workspaces/7/sync/stream", nil)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("GET: %v", err)
	}
	defer resp.Body.Close()

	// Read the initial event.
	events := readSSE(t, resp.Body, 1, 1*time.Second)
	if len(events) < 1 {
		t.Fatalf("no initial event")
	}

	// Simulate the syncer's callback transitioning the job to
	// 'done' with commit_sha populated. The handler must emit a
	// second event on the next poll.
	streamer.set(&domain.SyncJob{
		ID:             1,
		Status:         "done",
		CommitSHAAfter: stringPtr("ec8fbc8a"),
	})
	events2 := readSSE(t, resp.Body, 1, 2*time.Second)
	if len(events2) < 1 {
		t.Fatalf("no event after state change; want 1 (the transition event)")
	}
	var got syncResponse
	if err := json.Unmarshal([]byte(events2[0]), &got); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if got.Status != "done" {
		t.Errorf("status = %q, want done", got.Status)
	}
	if got.CommitSHAAfter == nil || *got.CommitSHAAfter != "ec8fbc8a" {
		t.Errorf("commit_sha_after = %v, want ec8fbc8a", got.CommitSHAAfter)
	}
}

func TestSSE_PollErrorsDoNotKillStream(t *testing.T) {
	// The handler must NOT return on transient DB errors. The
	// loop should log + backoff + continue.
	streamer := &fakeStreamer{}
	srv := newSSEServer(streamer)
	defer srv.Close()

	req, _ := http.NewRequest(http.MethodGet, srv.URL+"/workspaces/7/sync/stream", nil)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("GET: %v", err)
	}
	defer resp.Body.Close()

	// Simulate transient errors.
	streamer.err = errors.New("connection reset by peer")
	time.Sleep(2 * ssePollInterval)
	// Then the DB recovers.
	streamer.err = nil
	streamer.set(&domain.SyncJob{ID: 1, Status: "done"})

	// We should eventually see the recovered event.
	events := readSSE(t, resp.Body, 1, 5*time.Second)
	if len(events) < 1 {
		t.Fatalf("stream died on transient errors (got %d events)", len(events))
	}
}

// TestSSE_RejectsUnauthenticatedRequests is the regression
// test for the UAT bug discovered 2026-07-08: the SSE endpoint
// was mounted on the root Echo (no authChain), so the
// IdentityFromCookie middleware never ran. A request with a
// valid session cookie (but going through /api/sync/stream
// without the auth chain) returned 400 "Authentication
// required" even though the cookie was valid.
//
// The fix: mount the SSE route on the same auth-protected
// group as the other sync routes (e.g. syncGroup in main.go).
// This test pins the handler-side contract: WITHOUT auth, the
// handler must reject with 400. With auth, the happy-path
// tests above pass.
func TestSSE_RejectsUnauthenticatedRequests(t *testing.T) {
	streamer := &fakeStreamer{
		next: &domain.SyncJob{ID: 1, Status: "pending"},
	}
	logger := discardLogger()
	e := echo.New()
	// Intentionally do NOT add the auth stub. The handler
	// must reject.
	h := NewSyncStreamHandler(streamer, logger)
	e.GET("/workspaces/:id/sync/stream", h.Stream)
	srv := httptest.NewServer(e)
	defer srv.Close()

	req, _ := http.NewRequest(http.MethodGet, srv.URL+"/workspaces/7/sync/stream", nil)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("GET: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("status = %d, want 400 (the auth check must fire when no identity is in context)", resp.StatusCode)
	}
	body, _ := io.ReadAll(resp.Body)
	if !strings.Contains(string(body), "Authentication required") {
		t.Errorf("body = %q, want it to contain 'Authentication required'", string(body))
	}
}

func stringPtr(s string) *string { return &s }

// newSSEEcho is a minimal Echo router used by the SSE tests.
// It registers a stub IdentityFromCookie middleware so the
// auth check passes without a real session cookie.
//
// Implementation lives in sse_test_helper.go (in the same
// package) so the test can call it without reaching into
// production-only helpers.
var _ = newSSEEcho // keep the helper referenced

var _ = echo.New
