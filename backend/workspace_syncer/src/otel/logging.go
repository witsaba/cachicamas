// Package otel — the slog logger wiring for the workspace_syncer.
//
// PR-2a uses a basic JSON slog handler that writes to stdout. The
// redaction handler (which redacts oauth_token / authorization /
// access_token from every log line) lands in PR-2c. The rest of the
// service code already passes *slog.Logger around so the upgrade is
// non-breaking.
package otel

import (
	"log/slog"
	"os"
)

// NewLogger returns a *slog.Logger that writes JSON to stdout. The
// default level is Info. PR-2c wraps this with a redaction handler
// (see design.md §5 — Cross-service auth posture).
func NewLogger() *slog.Logger {
	return slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))
}
