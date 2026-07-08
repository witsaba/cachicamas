package otel

import (
	"bytes"
	"context"
	"encoding/json"
	"log/slog"
	"strings"
	"testing"
	"time"
)

// timeNow returns the current time. We wrap time.Now in a tiny
// helper so the test file is easier to refactor (and so the
// slog.NewRecord signature change doesn't propagate everywhere).
func timeNow() time.Time { return time.Now() }
var _ = timeNow

// captureLogger returns a *slog.Logger whose output is captured
// in the returned bytes.Buffer. The handler is wrapped with
// NewRedactionHandler so the tests exercise the production
// redaction.
func captureLogger(level slog.Level) (*slog.Logger, *bytes.Buffer) {
	var buf bytes.Buffer
	inner := slog.NewJSONHandler(&buf, &slog.HandlerOptions{Level: level})
	return slog.New(NewRedactionHandler(inner)), &buf
}

func decodeLog(t *testing.T, buf *bytes.Buffer) map[string]any {
	t.Helper()
	var entry map[string]any
	if err := json.Unmarshal(bytes.TrimSpace(buf.Bytes()), &entry); err != nil {
		t.Fatalf("decode log entry: %v\nraw: %s", err, buf.String())
	}
	return entry
}

func TestRedactionHandler_RedactsOauthToken(t *testing.T) {
	logger, buf := captureLogger(slog.LevelInfo)
	logger.Info("test",
		slog.String("oauth_token", "gho_real_token_value"),
		slog.String("user", "octocat"),
	)
	entry := decodeLog(t, buf)
	if entry["oauth_token"] != redactedValue {
		t.Errorf("oauth_token = %v, want %q", entry["oauth_token"], redactedValue)
	}
	if entry["user"] != "octocat" {
		t.Errorf("user = %v, want octocat (non-redacted field should pass through)", entry["user"])
	}
}

func TestRedactionHandler_RedactsAuthorization(t *testing.T) {
	logger, buf := captureLogger(slog.LevelInfo)
	logger.Info("test",
		slog.String("authorization", "Bearer gho_real"),
	)
	entry := decodeLog(t, buf)
	if entry["authorization"] != redactedValue {
		t.Errorf("authorization = %v, want %q (the whole value is redacted; the 'Bearer ' prefix does not survive)", entry["authorization"], redactedValue)
	}
}

func TestRedactionHandler_RedactsCaseInsensitive(t *testing.T) {
	logger, buf := captureLogger(slog.LevelInfo)
	logger.Info("test",
		slog.String("OAuth_Token", "gho_xxx"),
		slog.String("AUTHORIZATION", "Bearer gho_yyy"),
	)
	entry := decodeLog(t, buf)
	if entry["OAuth_Token"] != redactedValue {
		t.Errorf("OAuth_Token = %v, want %q", entry["OAuth_Token"], redactedValue)
	}
	if entry["AUTHORIZATION"] != redactedValue {
		t.Errorf("AUTHORIZATION = %v, want %q (whole value redacted)", entry["AUTHORIZATION"], redactedValue)
	}
}

func TestRedactionHandler_PreservesNonRedactedFields(t *testing.T) {
	logger, buf := captureLogger(slog.LevelInfo)
	logger.Info("test",
		slog.String("job_id", "42"),
		slog.String("status", "running"),
		slog.String("commit_sha", "abc1234"),
	)
	entry := decodeLog(t, buf)
	if entry["job_id"] != "42" {
		t.Errorf("job_id = %v, want 42", entry["job_id"])
	}
	if entry["status"] != "running" {
		t.Errorf("status = %v, want running", entry["status"])
	}
	if entry["commit_sha"] != "abc1234" {
		t.Errorf("commit_sha = %v, want abc1234", entry["commit_sha"])
	}
}

func TestRedactionHandler_PreservesTokenShape(t *testing.T) {
	// The literal "[REDACTED]" is a marker; a future reviewer
	// can grep for it in production logs to confirm a leak
	// was prevented.
	logger, buf := captureLogger(slog.LevelInfo)
	logger.Info("test", slog.String("oauth_token", "gho_abc"))
	out := buf.String()
	if !strings.Contains(out, redactedValue) {
		t.Errorf("output must contain the redacted marker; got: %s", out)
	}
	if strings.Contains(out, "gho_abc") {
		t.Errorf("output must NOT contain the original token; got: %s", out)
	}
}

func TestRedactionHandler_WithAttrs(t *testing.T) {
	// WithAttrs returns a new handler; the redaction must
	// still apply to attrs added via WithAttrs.
	logger, buf := captureLogger(slog.LevelInfo)
	logger = logger.With(slog.String("request_id", "req-1"))
	logger.Info("test", slog.String("oauth_token", "gho_xxx"))
	out := buf.String()
	if !strings.Contains(out, redactedValue) {
		t.Errorf("WithAttrs output must redact; got: %s", out)
	}
}

func TestRedactionHandler_Disabled(t *testing.T) {
	// The handler respects the inner handler's Enabled. A
	// logger configured at Error level must NOT emit Info
	// records (the redaction handler does not need to be
	// tested in this case — the inner handler handles it).
	logger, buf := captureLogger(slog.LevelError)
	logger.Info("test", slog.String("oauth_token", "gho_xxx"))
	if buf.Len() != 0 {
		t.Errorf("Info record must be dropped at Error level; got: %s", buf.String())
	}
}

func TestRedactionHandler_PreservesContext(t *testing.T) {
	// The Handle method MUST propagate ctx to the inner handler
	// so trace correlation works. This is a smoke test.
	logger, buf := captureLogger(slog.LevelInfo)
	rec := slog.NewRecord(timeNow(), slog.LevelInfo, "msg", 0)
	if err := logger.Handler().Handle(context.Background(), rec); err != nil {
		t.Fatalf("Handle: %v", err)
	}
	if !strings.Contains(buf.String(), `"msg":"msg"`) {
		t.Errorf("Handle failed to propagate; got: %s", buf.String())
	}
}