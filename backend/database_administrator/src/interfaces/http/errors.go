// Package httpiface — errors.go introduces the closed-vocabulary
// ErrorKind enum + ClassifyError helper for the handler layer.
//
// 2026-08-02-security-vulnerability-remediation (M-5, M-6): the
// previous design echoed `err.Error()` into both the response
// body and the slog line. An attacker that controls part of the
// JSON body — and thereby controls the contents of the error
// returned by json.Decode — could exfiltrate internals or shape
// the response. The fix:
//
//   - replace `+err.Error()` in the response body with a fixed
//     "body could not be parsed" / "body schema mismatch"
//     message (the operating code is in the body, never the
//     raw error text).
//   - replace `slog.String("err", err.Error())` with a fixed
//     closed-vocabulary `error_kind` key (decode_failed |
//     validation_failed | not_found | conflict | internal).
//     The detailed error is logged with `slog.Debug` only
//     (not bound for production; see design §3).
//
// The classification is purely a label for the operator; the
// client sees the same HTTP envelope shape as before, only
// with the sanitized message instead of the raw error.
package httpiface

import (
	"context"
	"errors"
	"fmt"
	"log/slog"

	"github.com/cachicamas/backend/database_administrator/src/domain"
)

// ErrorKind is the closed vocabulary used in the slog `error_kind`
// attribute. Adding new values MUST be coordinated with the wider
// observability stack (the existing dashboards bucket on these).
type ErrorKind string

// Documented ErrorKind values. Each maps to a fixed, sanitized
// log message and (where relevant) HTTP status.
const (
	// ErrorKindDecodeFailed: the request body could not be parsed
	// (json.Unmarshal error, EOF, etc.). Sanitized response:
	// "body could not be parsed"; HTTP 422.
	ErrorKindDecodeFailed ErrorKind = "decode_failed"

	// ErrorKindValidationFailed: the parsed body failed the
	// domain validator. Sanitized response: the field-level
	// `fields` map (NOT the raw error); HTTP 400.
	ErrorKindValidationFailed ErrorKind = "validation_failed"

	// ErrorKindNotFound: the requested resource does not exist
	// (or is in a different tenant). Sanitized response: "not
	// found"; HTTP 404.
	ErrorKindNotFound ErrorKind = "not_found"

	// ErrorKindConflict: a unique-constraint violation (name
	// already exists, etc.). Sanitized response: "conflict";
	// HTTP 409.
	ErrorKindConflict ErrorKind = "conflict"

	// ErrorKindInternal: anything else. Sanitized response:
	// "An internal error occurred."; HTTP 500.
	ErrorKindInternal ErrorKind = "internal"
)

// ClassifyError maps a domain error to the closed ErrorKind and
// returns a sanitized message safe for the response body. The
// returned message is FIXED per ErrorKind — no err.Error() text
// ever reaches the client.
//
// The caller is responsible for the HTTP status mapping (use
// httpStatusForErrorKind); the helper only handles the error
// classification + sanitization.
func ClassifyError(err error) (kind ErrorKind, clientMessage string) {
	if err == nil {
		return ErrorKindInternal, "an internal error occurred"
	}

	var verr *domain.ValidationError
	if errors.As(err, &verr) {
		return ErrorKindValidationFailed, "validation failed"
	}

	var cerr *domain.ConflictError
	if errors.As(err, &cerr) {
		return ErrorKindConflict, "conflict"
	}

	var nerr *domain.NotFoundError
	if errors.As(err, &nerr) {
		return ErrorKindNotFound, "not found"
	}

	// Default to internal. The decoder error paths land here
	// (json.Unmarshal returns raw errors that we never want to
	// echo into the response).
	return ErrorKindInternal, "an internal error occurred"
}

// httpStatusForErrorKind maps the closed vocabulary to the locked
// HTTP status codes. The mapping is load-bearing: the spec hashes
// (status, code) pairs to gate the wire contract.
func httpStatusForErrorKind(kind ErrorKind) int {
	switch kind {
	case ErrorKindDecodeFailed:
		return 422
	case ErrorKindValidationFailed:
		return 400
	case ErrorKindNotFound:
		return 404
	case ErrorKindConflict:
		return 409
	case ErrorKindInternal:
		return 500
	default:
		return 500
	}
}

// LogSanitized emits a slog line that carries the closed-vocabulary
// error_kind + the fixed sanitized message. The raw err.Error() is
// logged at DEBUG level only (slog.Warn does not surface it). This
// is the only sanctioned pattern for handler-level error logs that
// touch user-controlled input.
func LogSanitized(ctx context.Context, logger *slog.Logger, op, slug string, kind ErrorKind, err error) {
	if logger == nil {
		return
	}
	attrs := []any{
		slog.String("op", op),
		slog.String("slug", slug),
		slog.String("error_kind", string(kind)),
	}
	logger.LogAttrs(ctx, slog.LevelInfo, "prompt request failed", attrsFromAny(attrs)...)
	// Keep the raw error around for debug-level diagnostics
	// — never bound for production.
	logger.DebugContext(ctx, "prompt request failed (raw error)",
		slog.String("op", op),
		slog.String("error", errorString(err)),
	)
}

// attrsFromAny is a tiny helper to convert the variadic []any into
// the typed form LogAttrs needs. We keep it local so the package
// surface stays small.
func attrsFromAny(in []any) []slog.Attr {
	out := make([]slog.Attr, 0, len(in))
	for _, a := range in {
		switch v := a.(type) {
		case slog.Attr:
			out = append(out, v)
		default:
			out = append(out, slog.Any("attr", fmt.Sprintf("%v", v)))
		}
	}
	return out
}

// errorString returns err.Error() or "" for nil. Centralized so
// callers can't accidentally log a nil dereference.
func errorString(err error) string {
	if err == nil {
		return ""
	}
	return err.Error()
}
