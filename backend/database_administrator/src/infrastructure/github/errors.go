package github

import "errors"

// UnauthorizedError signals that GitHub rejected the OAuth access_token
// (401). The caller (GitHubHandler) maps this to HTTP 502 with
// `error: "github_unauthorized"` per S-WS-082.
type UnauthorizedError struct {
	Cause error
}

func (e *UnauthorizedError) Error() string {
	if e.Cause != nil {
		return "github unauthorized: " + e.Cause.Error()
	}
	return "github unauthorized"
}

func (e *UnauthorizedError) Unwrap() error { return e.Cause }

// RateLimitedError signals that GitHub returned 403 (rate-limited). The
// caller maps this to HTTP 502 with `error: "github_rate_limited"` per
// S-WS-085.
type RateLimitedError struct {
	Cause error
}

func (e *RateLimitedError) Error() string {
	if e.Cause != nil {
		return "github rate-limited: " + e.Cause.Error()
	}
	return "github rate-limited"
}

func (e *RateLimitedError) Unwrap() error { return e.Cause }

// ParseError signals that GitHub returned a 2xx but the body was not
// valid JSON. The caller maps this to HTTP 502 with a generic message.
type ParseError struct {
	Cause error
}

func (e *ParseError) Error() string {
	if e.Cause != nil {
		return "github parse: " + e.Cause.Error()
	}
	return "github parse error"
}

func (e *ParseError) Unwrap() error { return e.Cause }

// AsUnauthorized reports whether err is a *UnauthorizedError. The handler
// uses this in errors.As.
func AsUnauthorized(err error) (*UnauthorizedError, bool) {
	var u *UnauthorizedError
	if errors.As(err, &u) {
		return u, true
	}
	return nil, false
}

// AsRateLimited reports whether err is a *RateLimitedError.
func AsRateLimited(err error) (*RateLimitedError, bool) {
	var r *RateLimitedError
	if errors.As(err, &r) {
		return r, true
	}
	return nil, false
}

// NotFoundError signals that GitHub returned 404 for the requested
// resource (repo deleted, renamed, or private to a different org).
// Added in 2026-07-08-workspace-sync-clone PR-3b so the
// permission-validation use case can branch on a distinct typed
// error (rather than re-parsing the message).
type NotFoundError struct {
	Cause error
}

func (e *NotFoundError) Error() string {
	if e.Cause != nil {
		return "github not found: " + e.Cause.Error()
	}
	return "github not found"
}

func (e *NotFoundError) Unwrap() error { return e.Cause }

// AsNotFound reports whether err is a *NotFoundError.
func AsNotFound(err error) (*NotFoundError, bool) {
	var n *NotFoundError
	if errors.As(err, &n) {
		return n, true
	}
	return nil, false
}
