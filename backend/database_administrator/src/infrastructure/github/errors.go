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
