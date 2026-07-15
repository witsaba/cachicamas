// Package httpiface — internal_callback_handler.go is the
// receiver for the workspace_syncer's POST /internal/sync-callback
// (2026-07-08-workspace-sync-clone PR-3b). The endpoint is
// reached only by the workspace_syncer over the docker network;
// it is NOT mounted under the user-JWE auth chain. The auth
// posture mirrors the identity callback (HMAC + anti-replay
// timestamp window) but uses a different shared secret
// (SYNC_CALLBACK_SECRET) so a leak of one does not compromise
// the other.
//
// Wire protocol (locked):
//
//	POST /api/v1/internal/sync-callback
//	Headers:
//	  Content-Type: application/json
//	  X-Cachicamas-Timestamp: <unix_ms string>
//	  X-Cachicamas-Signature: base64(HMAC_SHA256(secret, ts + "." + canonical_json))
//	Body: { job_id, status, commit_sha?, default_branch?, error_code?, error_message? }
//
// Status codes (locked):
//   - 204 No Content on success
//   - 401 Unauthorized on bad/expired/missing signature or timestamp
//   - 422 Unprocessable Entity on schema validation failure
//   - 500 Internal Server Error on service error
package httpiface

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"sort"
	"strconv"
	"time"

	"github.com/labstack/echo/v5"

	"github.com/cachicamas/backend/database_administrator/src/application"
	"github.com/cachicamas/backend/database_administrator/src/domain"
)

// SyncCallbackProcessor is the slice of *application.SyncService
// the handler consumes. Tests can wire a fake without standing up
// the full hexagonal graph.
type SyncCallbackProcessor interface {
	ProcessSyncCallback(ctx context.Context, jobID int64, status, commitSHA, defaultBranch, errorCode, errorMessage string) (*domain.SyncJob, error)
}

// SyncCallbackHandler is the receiver for the workspace_syncer's
// POST /api/v1/internal/sync-callback.
type SyncCallbackHandler struct {
	processor SyncCallbackProcessor
	secret    []byte
	logger    *slog.Logger
}

// NewSyncCallbackHandler wires a SyncCallbackHandler. Panics on
// bad config (mirrors the NewIdentityHandler posture): the
// caller (main.go) treats that as a startup crash.
func NewSyncCallbackHandler(processor SyncCallbackProcessor, secret string, logger *slog.Logger) *SyncCallbackHandler {
	if processor == nil {
		panic("NewSyncCallbackHandler: processor must not be nil")
	}
	if secret == "" {
		panic("NewSyncCallbackHandler: secret must not be empty (set SYNC_CALLBACK_SECRET in env)")
	}
	if len(secret) < 32 {
		panic(fmt.Sprintf("NewSyncCallbackHandler: secret must be at least 32 raw bytes (got %d); generate with `openssl rand -base64 32`", len(secret)))
	}
	if logger == nil {
		panic("NewSyncCallbackHandler: logger must not be nil")
	}
	return &SyncCallbackHandler{
		processor: processor,
		secret:    []byte(secret),
		logger:    logger,
	}
}

// RegisterInternalSyncCallbackRoute mounts the handler on the
// given Echo instance. The route is mounted on the public/internal
// route group (NOT under the JWE-verifier middleware).
func RegisterInternalSyncCallbackRoute(e *echo.Echo, processor SyncCallbackProcessor, secret string, logger *slog.Logger) {
	h := NewSyncCallbackHandler(processor, secret, logger)
	e.POST("/api/v1/internal/sync-callback", h.HandleSyncCallback)
}

// syncCallbackBody is the JSON body the workspace_syncer POSTs.
// Optional fields are pointers so we can distinguish "not set"
// from "set to empty string".
type syncCallbackBody struct {
	JobID        int64   `json:"job_id"`
	Status       string  `json:"status"`
	CommitSHA    *string `json:"commit_sha,omitempty"`
	DefaultBranch *string `json:"default_branch,omitempty"`
	ErrorCode    *string `json:"error_code,omitempty"`
	ErrorMessage *string `json:"error_message,omitempty"`
}

// HandleSyncCallback is the HTTP transport for the workspace_syncer's
// callback. See the package doc for the wire contract.
func (h *SyncCallbackHandler) HandleSyncCallback(c *echo.Context) error {
	req := c.Request()

	// 1. Read body.
	rawBody, err := io.ReadAll(req.Body)
	if err != nil {
		return h.writeUnauthorized(c, "body_read_failed")
	}

	// 2. Parse + canonicalize.
	var parsed any
	if err := json.Unmarshal(rawBody, &parsed); err != nil {
		return h.writeUnprocessable(c, "body is not valid JSON: "+err.Error())
	}
	canonical, err := canonicalizeCallbackJSON(parsed)
	if err != nil {
		return h.writeUnprocessable(c, "body could not be canonicalized: "+err.Error())
	}

	// 3. Read + validate headers.
	tsHeader := req.Header.Get("X-Cachicamas-Timestamp")
	if tsHeader == "" {
		return h.writeUnauthorized(c, "missing_timestamp")
	}
	tsUnix, err := strconv.ParseInt(tsHeader, 10, 64)
	if err != nil {
		return h.writeUnauthorized(c, "malformed_timestamp")
	}
	sigHeader := req.Header.Get("X-Cachicamas-Signature")
	if sigHeader == "" {
		return h.writeUnauthorized(c, "missing_signature")
	}
	sigRaw, err := base64.StdEncoding.DecodeString(sigHeader)
	if err != nil {
		return h.writeUnauthorized(c, "malformed_signature_encoding")
	}

	// 4. Anti-replay window (5 minutes, same as identity callback).
	nowMs := time.Now().UnixMilli()
	delta := nowMs - tsUnix
	if delta < 0 {
		delta = -delta
	}
	if time.Duration(delta)*time.Millisecond > antiReplayWindow {
		return h.writeUnauthorized(c, "timestamp_outside_window")
	}

	// 5. Constant-time HMAC compare.
	mac := hmac.New(sha256.New, h.secret)
	mac.Write([]byte(tsHeader))
	mac.Write([]byte("."))
	mac.Write(canonical)
	expected := mac.Sum(nil)
	if subtle.ConstantTimeCompare(expected, sigRaw) != 1 {
		return h.writeUnauthorized(c, "bad_signature")
	}

	// 6. Body schema validation.
	var body syncCallbackBody
	if err := json.Unmarshal(rawBody, &body); err != nil {
		return h.writeUnprocessable(c, "body schema mismatch: "+err.Error())
	}
	if body.JobID <= 0 {
		return h.writeUnprocessable(c, "job_id is required and must be > 0")
	}
	if body.Status != domain.SyncJobStatusDone && body.Status != domain.SyncJobStatusFailed {
		return h.writeUnprocessable(c, "status must be done or failed")
	}
	if body.Status == domain.SyncJobStatusDone {
		if body.CommitSHA == nil || *body.CommitSHA == "" {
			return h.writeUnprocessable(c, "commit_sha is required on status=done")
		}
	}

	// 7. Dispatch.
	commitSHA := ""
	if body.CommitSHA != nil {
		commitSHA = *body.CommitSHA
	}
	defaultBranch := ""
	if body.DefaultBranch != nil {
		defaultBranch = *body.DefaultBranch
	}
	errorCode := ""
	if body.ErrorCode != nil {
		errorCode = *body.ErrorCode
	}
	errorMessage := ""
	if body.ErrorMessage != nil {
		errorMessage = *body.ErrorMessage
	}

	if _, err := h.processor.ProcessSyncCallback(req.Context(), body.JobID, body.Status, commitSHA, defaultBranch, errorCode, errorMessage); err != nil {
		var verr *domain.ValidationError
		if errors.As(err, &verr) {
			return h.writeUnprocessable(c, "validation: "+err.Error())
		}
		h.logger.ErrorContext(req.Context(), "sync-callback: service error",
			slog.String("error", err.Error()),
			slog.Int64("job_id", body.JobID),
			slog.String("status", body.Status),
		)
		return c.JSON(http.StatusInternalServerError, map[string]any{
			"code":    "internal_error",
			"message": "An internal error occurred while processing the sync callback.",
		})
	}

	return c.NoContent(http.StatusNoContent)
}

// writeUnauthorized emits the locked 401 envelope and logs the
// reason. Mirrors identity_handler.writeUnauthorized; reuses the
// shared antiReplayWindow constant.
func (h *SyncCallbackHandler) writeUnauthorized(c *echo.Context, reason string) error {
	h.logger.WarnContext(c.Request().Context(), "sync-callback rejected",
		slog.String("reason", reason),
	)
	return c.JSON(http.StatusUnauthorized, map[string]any{
		"code":    "unauthorized",
		"message": "signature verification failed",
	})
}

// writeUnprocessable emits the locked 422 envelope.
func (h *SyncCallbackHandler) writeUnprocessable(c *echo.Context, reason string) error {
	h.logger.WarnContext(c.Request().Context(), "sync-callback invalid body",
		slog.String("reason", reason),
	)
	return c.JSON(http.StatusUnprocessableEntity, map[string]any{
		"code":    "unprocessable_entity",
		"message": reason,
	})
}

// canonicalizeCallbackJSON is the RFC 8785-lite canonicalizer for
// the sync-callback body. Mirrors identity_handler.canonicalizeJSON
// but lives here to keep the two secrets' canonicalizers isolated
// (a future change to one schema MUST NOT silently affect the other).
func canonicalizeCallbackJSON(v any) ([]byte, error) {
	var buf bytes.Buffer
	if err := canonicalWriteCallbackValue(&buf, v); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func canonicalWriteCallbackValue(buf *bytes.Buffer, v any) error {
	switch x := v.(type) {
	case nil:
		buf.WriteString("null")
	case bool:
		if x {
			buf.WriteString("true")
		} else {
			buf.WriteString("false")
		}
	case string:
		b, err := json.Marshal(x)
		if err != nil {
			return err
		}
		buf.Write(b)
	case float64:
		// json.Unmarshal decodes JSON numbers as float64. We
		// accept whole numbers as integers (write without a
		// decimal point) so the canonical bytes match the wire
		// shape; non-whole numbers are rejected (the closed
		// schema does not include floats).
		if x == float64(int64(x)) {
			buf.WriteString(strconv.FormatInt(int64(x), 10))
			return nil
		}
		return fmt.Errorf("canonicalWriteCallbackValue: non-integer numeric value not allowed: %v", x)
	case map[string]any:
		keys := make([]string, 0, len(x))
		for k := range x {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		buf.WriteByte('{')
		for i, k := range keys {
			if i > 0 {
				buf.WriteByte(',')
			}
			kb, err := json.Marshal(k)
			if err != nil {
				return err
			}
			buf.Write(kb)
			buf.WriteByte(':')
			if err := canonicalWriteCallbackValue(buf, x[k]); err != nil {
				return err
			}
		}
		buf.WriteByte('}')
	case []any:
		buf.WriteByte('[')
		for i, e := range x {
			if i > 0 {
				buf.WriteByte(',')
			}
			if err := canonicalWriteCallbackValue(buf, e); err != nil {
				return err
			}
		}
		buf.WriteByte(']')
	default:
		return fmt.Errorf("canonicalWriteCallbackValue: unsupported type %T", v)
	}
	return nil
}

// Compile-time check that SyncCallbackHandler can be wired by
// the application layer (it satisfies the SyncCallbackProcessor
// interface via the production adapter in main.go).
var _ SyncCallbackProcessor = (*application.SyncService)(nil)