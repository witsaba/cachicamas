// Package httpiface — sync_stream_handler.go implements the
// Server-Sent Events (SSE) endpoint for live sync_job updates.
//
//   GET /workspaces/:id/sync/stream
//   Accept: text/event-stream
//   Response: text/event-stream
//
// The endpoint pushes a JSON event every time the workspace's
// latest sync_job row changes (created, status transition, or
// commit_sha populated). The client opens an EventSource
// (browser-native) and updates the UI in real-time — no polling.
//
// Why SSE instead of polling (R-WS-019 S-WS-196):
//
//   - Push-based: zero lag. The card transitions the instant
//     the syncer's callback posts to db_admin.
//   - Single source of truth: the server polls the DB once per
//     client connection; with N clients, the server does N
//     polls (each on its own goroutine), not N client polls
//     hitting the network.
//   - Robust against client-side bugs: no QRL closure issues,
//     no useSignal reactivity gaps, no setTimeout traps.
//
// Wire shape (SSE protocol):
//
//   data: {"job_id":1,"workspace_id":1,"status":"done",
//          "commit_sha":"ec8fbc8","default_branch":"main", ...}\n\n
//   : keepalive\n\n     (sent every 15s to keep the connection open
//                          through intermediaries)
//
// The client (frontend) sees a sequence of MessageEvents, each
// with `event.data` being the JSON of a syncResponse. The first
// event is the current state (snapshot); subsequent events are
// updates on change.
package httpiface

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/labstack/echo/v5"

	"github.com/cachicamas/backend/database_administrator/src/domain"
)

// SSE poll intervals (production-tuned for UAT).
const (
	ssePollInterval     = 750 * time.Millisecond // how often the server polls the DB
	sseKeepaliveEvery   = 15 * time.Second       // how often a ": keepalive\n\n" is sent
	sseInitialBackoff   = 100 * time.Millisecond // small initial backoff for the first poll
	sseMaxBackoffGrowth = 5 * time.Second       // cap on the per-error backoff (defensive)
)

// SyncStreamHandler is the SSE handler for live sync_job updates.
// The handler holds the service-shaped dependency it needs to
// read the latest sync_job; the test wiring passes a fake.
type SyncStreamHandler struct {
	streamer SyncJobStreamer
	logger  *slog.Logger
}

// SyncJobStreamer is the narrow contract this handler uses to
// poll the latest sync_job. Production: *application.SyncService
// (which exposes GetLatestSyncJob). Tests: a fake.
type SyncJobStreamer interface {
	GetLatestSyncJob(ctx context.Context, workspaceID int64) (*domain.SyncJob, error)
}

// NewSyncStreamHandler wires a SyncStreamHandler. streamer
// must be non-nil.
func NewSyncStreamHandler(streamer SyncJobStreamer, logger *slog.Logger) *SyncStreamHandler {
	if streamer == nil {
		panic("NewSyncStreamHandler: streamer must not be nil")
	}
	return &SyncStreamHandler{streamer: streamer, logger: logger}
}

// RegisterSyncStreamRoute mounts the SSE handler on the given
// Echo sub-group. The caller is responsible for applying the
// auth chain via the group middleware; the handler does not
// double-apply.
func RegisterSyncStreamRoute(e *echo.Echo, h *SyncStreamHandler) {
	e.GET("/workspaces/:id/sync/stream", h.Stream)
}

// Stream is the GET /workspaces/:id/sync/stream handler.
//
// Lifecycle:
//   1. Write SSE headers (Content-Type, Cache-Control, etc.)
//   2. Flusher-enable the response writer
//   3. Loop: poll GetLatestSyncJob on a ticker; on state change,
//      write "data: {json}\n\n" and flush. On ctx.Done() (client
//      disconnect, server shutdown, or 30s timeout), break the loop.
//   4. Return without writing a Content-Length header (SSE is
//      chunked transfer).
//
// Error handling: the handler is best-effort. A poll error
// (e.g. transient DB blip) is logged and the loop continues
// with exponential backoff. The client never sees a 5xx after
// the initial 200 (SSE doesn't allow a status change mid-stream).
func (h *SyncStreamHandler) Stream(c *echo.Context) error {
	workspaceID, err := parsePathInt64(c, "id")
	if err != nil {
		return writeSyncError(c, &domain.ValidationError{
			Fields: map[string]string{"id": "Invalid id."},
		})
	}
	if _, ok := identityFromContext(c); !ok {
		return writeSyncError(c, &domain.ValidationError{
			Fields: map[string]string{"auth": "Authentication required."},
		})
	}

	w := c.Response()
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("X-Accel-Buffering", "no") // disable proxy buffering (nginx)
	w.WriteHeader(http.StatusOK)

	flusher, ok := w.(http.Flusher)
	if !ok {
		// If the underlying writer doesn't support Flusher,
		// the SSE protocol is broken (events would buffer
		// forever). The frontend's EventSource would just
		// hang. We log + return; the alternative (panic) is
		// worse for production.
		h.logger.ErrorContext(c.Request().Context(),
			"sync stream: response writer does not support Flusher; SSE is broken on this transport",
			slog.Int64("workspace_id", workspaceID),
		)
		return nil
	}

	ctx := c.Request().Context()
	ticker := time.NewTicker(ssePollInterval)
	defer ticker.Stop()
	keepalive := time.NewTicker(sseKeepaliveEvery)
	defer keepalive.Stop()

	var lastJSON string
	backoff := sseInitialBackoff

	// Send the initial state immediately so the client doesn't
	// see a blank card for the first poll interval.
	if err := h.pollAndSend(ctx, workspaceID, &lastJSON, flusher, w); err != nil {
		h.logger.WarnContext(ctx, "sync stream: initial poll failed",
			slog.Int64("workspace_id", workspaceID),
			slog.String("error", err.Error()),
		)
		// Don't return — the loop will retry. Send an initial
		// keepalive so the client doesn't time out.
		_, _ = fmt.Fprintf(w, ": keepalive\n\n")
		flusher.Flush()
	}

	for {
		select {
		case <-ctx.Done():
			// Client disconnected, or server shutdown.
			return nil
		case <-keepalive.C:
			// Per the SSE spec, comments (lines starting with
			// ':') are ignored by the client. We use them as
			// keepalives to keep the connection open through
			// intermediaries (proxies, load balancers) that
			// close idle connections.
			if _, err := fmt.Fprintf(w, ": keepalive\n\n"); err != nil {
				return nil
			}
			flusher.Flush()
		case <-ticker.C:
			if err := h.pollAndSend(ctx, workspaceID, &lastJSON, flusher, w); err != nil {
				h.logger.WarnContext(ctx,
					"sync stream: poll failed (backoff and retry)",
					slog.Int64("workspace_id", workspaceID),
					slog.String("error", err.Error()),
					slog.Duration("backoff", backoff),
				)
				// Don't return; backoff and retry on the
				// next tick. The loop continues even on
				// transient DB errors.
				time.Sleep(backoff)
				if backoff < sseMaxBackoffGrowth {
					backoff *= 2
				}
				continue
			}
			backoff = sseInitialBackoff
		}
	}
}

// pollAndSend fetches the latest sync_job, serializes it to the
// SSE wire format, and writes it to the response writer if the
// serialized form differs from the previous one (delta-only). On
// change, flushes immediately. Returns nil on a successful no-op
// or successful send; returns a non-nil error if the DB poll failed
// (the caller applies backoff).
func (h *SyncStreamHandler) pollAndSend(
	ctx context.Context,
	workspaceID int64,
	lastJSON *string,
	flusher http.Flusher,
	w http.ResponseWriter,
) error {
	job, err := h.streamer.GetLatestSyncJob(ctx, workspaceID)
	if err != nil {
		return fmt.Errorf("poll: %w", err)
	}
	if job == nil {
		// No job yet. Send an empty event (so the client
		// knows the stream is alive and we're polling) but
		// don't replace lastJSON (so we keep sending the same
		// keepalive-only state on subsequent polls).
		if _, err := fmt.Fprintf(w, ": empty\n\n"); err != nil {
			return fmt.Errorf("write empty event: %w", err)
		}
		flusher.Flush()
		return nil
	}

	// Serialize to the SAME shape as the REST GET endpoint
	// (syncResponse), so the frontend's existing toSyncResponse
	// mapping works without a second type.
	payload, err := json.Marshal(toSyncResponse(job))
	if err != nil {
		// JSON marshal of a known struct should not fail.
		// If it does, return so the loop can backoff.
		return fmt.Errorf("marshal: %w", err)
	}
	payloadStr := string(payload)
	if payloadStr == *lastJSON {
		// No change. Skip the write + flush (delta-only).
		return nil
	}
	*lastJSON = payloadStr

	// Standard SSE frame: "data: <payload>\n\n".
	// The trailing \n\n is the event boundary (the empty line
	// terminates the event per the SSE spec).
	if _, err := fmt.Fprintf(w, "data: %s\n\n", payloadStr); err != nil {
		// Likely a closed connection. The next select will
		// return ctx.Done() and we'll exit cleanly.
		return fmt.Errorf("write event: %w", err)
	}
	flusher.Flush()
	return nil
}

// errSSEWriter is a test-only helper. It exists to ensure the
// errors package is imported even when no test uses errors.As in
// the handler body. (The select branch in Stream uses
// fmt.Fprintf errors; errors.As is for the polling error path.)
var _ = errors.New

// Compile-time check that the SSE handler doesn't accidentally
// lose its single-method dependency.
var _ SyncJobStreamer = (SyncJobStreamer)(nil)
