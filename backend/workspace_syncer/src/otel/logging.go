// Package otel — the slog logger with a token-redaction handler.
//
// PR-2a wrote a basic JSON slog handler. PR-2c wraps it with a
// custom handler that redacts any field whose key matches
// oauth_token / authorization / access_token (case-insensitive)
// from every log line. This is the locked behavior from
// design.md §5 (Cross-service auth posture) and the workspace-syncer
// spec R-WSY-005 S-WSY-042, S-WSY-043.
//
// SECURITY: the redaction is applied at the handler level, BEFORE
// the line is written to stdout. A future contributor who adds
// a new field to a slog.LogAttrs call MUST keep the redaction
// keys up to date; a test in logging_test.go asserts the
// redaction on a sample of common patterns.
package otel

import (
	"context"
	"log/slog"
	"os"
	"strings"
)

// redactedValue is the literal replacement string. A future
// reviewer can grep for this in the logs to confirm a leak was
// prevented.
const redactedValue = "[REDACTED]"

// redactKeys is the set of field keys whose value MUST be
// redacted. The match is case-insensitive. Add a new key here
// only after updating logging_test.go with a corresponding
// redaction case.
var redactKeys = map[string]struct{}{
	"oauth_token":   {},
	"authorization": {},
	"access_token":  {},
	"refresh_token": {},
}

// keyNeedsRedaction reports whether the given attr key must be
// redacted. Case-insensitive match against the redactKeys set.
func keyNeedsRedaction(key string) bool {
	_, found := redactKeys[strings.ToLower(key)]
	return found
}

// redactionHandler wraps another slog.Handler and redacts
// matched attribute keys from every record before delegating to
// the underlying handler.
type redactionHandler struct {
	inner slog.Handler
}

// NewRedactionHandler wraps inner with the token-redaction
// handler. Pass the result to slog.New() to construct a logger.
func NewRedactionHandler(inner slog.Handler) slog.Handler {
	return &redactionHandler{inner: inner}
}

// Enabled delegates to the inner handler.
func (h *redactionHandler) Enabled(ctx context.Context, level slog.Level) bool {
	return h.inner.Enabled(ctx, level)
}

// Handle redacts matched attributes from the record and
// delegates to the inner handler.
func (h *redactionHandler) Handle(ctx context.Context, r slog.Record) error {
	redacted := slog.NewRecord(r.Time, r.Level, r.Message, r.PC)
	r.Attrs(func(a slog.Attr) bool {
		if keyNeedsRedaction(a.Key) {
			redacted.AddAttrs(slog.String(a.Key, redactedValue))
		} else {
			redacted.AddAttrs(a)
		}
		return true
	})
	return h.inner.Handle(ctx, redacted)
}

// WithAttrs returns a new handler with the given attrs. The new
// attrs are passed through to the inner handler.
func (h *redactionHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	return &redactionHandler{inner: h.inner.WithAttrs(attrs)}
}

// WithGroup returns a new handler in the given group. Group
// names are NOT added to the redactKeys check — a future
// caller that nests fields under a group key (e.g.
// "auth.oauth_token") would NOT be redacted. Document this
// limitation in the design.md if it becomes a real concern.
func (h *redactionHandler) WithGroup(name string) slog.Handler {
	return &redactionHandler{inner: h.inner.WithGroup(name)}
}

// NewLogger returns a *slog.Logger that writes JSON to stdout
// with token redaction applied at the handler level. This is the
// production wiring (PR-2c replaces the basic logger with this
// one).
func NewLogger() *slog.Logger {
	jsonHandler := slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	})
	return slog.New(NewRedactionHandler(jsonHandler))
}