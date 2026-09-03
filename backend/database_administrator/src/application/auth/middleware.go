// Package auth — middleware.go implements the X-Internal-Secret
// verifier (spec R-BE-001 / R-MIDDLEWARE-1 / S-BE-010 / S-BE-011).
//
// The verifier is a pure function: it takes the header value the
// caller observed + the expected secret and returns nil (allowed)
// or one of three sentinels (missing / wrong / misconfigured).
// The Echo wrapping lives in interfaces/http/auth_handler.go; this
// file is HTTP-framework-agnostic so the application layer test
// can exercise it directly.
//
// CRITICAL invariants (locked at spec §5 + design §2.2):
//   - Comparison MUST use crypto/subtle.ConstantTimeCompare to
//     avoid timing attacks. A naïve bytes.Equal comparison lets an
//     attacker probe the secret byte-by-byte.
//   - The verifier MUST distinguish "missing header" from "wrong
//     value" so the handler can surface different log lines (and
//     the failure_reason can be logged separately). The handler
//     maps both to HTTP 401 — the difference is operational, not
//     client-visible.
//   - The verifier MUST refuse to operate with an empty expected
//     secret. This is the defense-in-depth gate: a missing
//     AUTH_INTERNAL_SECRET env is a misconfiguration, not a bypass.
package auth

import (
	"crypto/subtle"
	"errors"
	"fmt"
)

// Sentinels returned by CheckInternalSecret. All three wrap an
// underlying message but are reachable via errors.Is for the HTTP
// handler's status-code mapping.
var (
	// ErrMissingInternalSecret is returned when the header is empty
	// (S-BE-010). Maps to HTTP 401.
	ErrMissingInternalSecret = errors.New("auth: missing X-Internal-Secret header")

	// ErrWrongInternalSecret is returned when the header is present
	// but does not match the expected secret (S-BE-011). Maps to
	// HTTP 401. The error message intentionally does NOT distinguish
	// "completely wrong" from "right secret at a different time" so
	// an attacker cannot probe.
	ErrWrongInternalSecret = errors.New("auth: invalid X-Internal-Secret header")

	// ErrEmptyExpected is returned when the expected secret is
	// empty. This is a misconfiguration gate: the HTTP layer
	// refuses to start the server when AUTH_INTERNAL_SECRET is not
	// set (mirrors the AUTH_COOKIE_SECRET crash rule in main.go).
	// Maps to HTTP 500 at the handler level (defensive — should
	// never be hit in production).
	ErrEmptyExpected = errors.New("auth: empty expected secret (misconfiguration)")
)

// CheckInternalSecret validates the X-Internal-Secret header value
// against the expected secret using a constant-time comparison. The
// function is pure: no I/O, no logging, no global state.
//
// Returns:
//   - nil when the header matches the expected secret exactly.
//   - ErrMissingInternalSecret when the header is empty.
//   - ErrWrongInternalSecret when the header is non-empty but does
//     not match.
//   - ErrEmptyExpected when the expected secret is empty (the
//     server is misconfigured; the handler MUST refuse to start).
func CheckInternalSecret(headerValue, expected string) error {
	if expected == "" {
		return ErrEmptyExpected
	}
	if headerValue == "" {
		return fmt.Errorf("%w (header empty)", wrapMissing())
	}
	// Constant-time compare; subtle.ConstantTimeCompare returns 1
	// only when the two slices have equal content AND equal length.
	// Length mismatch therefore surfaces as a wrong-secret error
	// (no timing leak on the length).
	if subtle.ConstantTimeCompare([]byte(headerValue), []byte(expected)) != 1 {
		return fmt.Errorf("%w (header does not match)", wrapWrong())
	}
	return nil
}

// wrapMissing / wrapWrong return the underlying sentinels so
// errors.Is works through the fmt.Errorf wrapping above. The
// constructors are kept private to force callers through
// CheckInternalSecret.
func wrapMissing() error { return ErrMissingInternalSecret }
func wrapWrong() error   { return ErrWrongInternalSecret }