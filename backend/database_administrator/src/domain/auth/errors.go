// Package auth — errors.go defines the domain error vocabulary the
// Postgres adapters translate pgx-specific errors into. Keeping the
// errors in the domain layer (rather than the postgres package)
// preserves the hexagonal rule that the handler maps domain errors
// to HTTP status codes without ever importing pgx or database/sql.
package auth

import "fmt"

// ConflictError signals a uniqueness violation raised by a DB unique
// or partial-unique index (Postgres SQLSTATE 23505). The adapter
// wraps the underlying pgx error so callers can choose to log the
// cause while still branching on the error type.
//
// The Resource field carries the SQL table name so the handler can
// surface a more specific 409 envelope when needed (the existing
// domain.ConflictError follows the same pattern).
type ConflictError struct {
	Resource string
	Cause    error
}

// Error implements the error interface.
func (e *ConflictError) Error() string {
	if e.Cause != nil {
		return fmt.Sprintf("auth.ConflictError[%s]: %v", e.Resource, e.Cause)
	}
	return fmt.Sprintf("auth.ConflictError[%s]", e.Resource)
}

// Unwrap exposes the underlying pgx error so callers can use
// errors.Is / errors.As to detect the SQLSTATE.
func (e *ConflictError) Unwrap() error {
	return e.Cause
}

// NotFoundError signals a missing row from a SELECT (Postgres
// "no rows in result set"). The adapter returns (nil, nil) instead
// for ordinary lookups; this error is reserved for the cases where
// the caller asked for a single-row result by primary key and the
// row is gone (the me handler's GET /internal/me/:user_id path).
type NotFoundError struct {
	Resource string
}

// Error implements the error interface.
func (e *NotFoundError) Error() string {
	return fmt.Sprintf("auth.NotFoundError[%s]", e.Resource)
}