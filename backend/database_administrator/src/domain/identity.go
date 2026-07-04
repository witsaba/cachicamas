// Package domain contains the core business types of the database
// administrator service. It has no dependencies on frameworks,
// transport, or infrastructure.
//
// This file defines the Identity aggregate used by the GitHub OAuth
// login slice (cachicamas-github-login). The struct mirrors the DDL
// of `identity.user` + a denormalized view of the matching
// `identity.account` row (provider + provider_account_id), so the
// verifier middleware can populate `c.Set("identity", *Identity)`
// without a second DB roundtrip.
//
// Per spec R-BAM-010, the field list is LOCKED. Any future rename
// or removal will fail `TestIdentity_StructFields` in identity_test.go.
package domain

import (
	"context"
)

// Identity is the domain entity for the `identity.user` table (DDL:
// migration/sql/20260703120000_github_login.sql) plus a denormalized
// snapshot of the matching `identity.account` row. The struct's
// `db` and `json` tags pin which column each field reads from on
// the wire.
//
// IMPORTANT: the field list is LOCKED per spec R-BAM-010. The order
// is also pinned (the test asserts against it). Do not reorder or
// remove a field without a spec amendment.
type Identity struct {
	ID                int64  `db:"id"                json:"id"`
	Email             string `db:"email"             json:"email"`
	Name              string `db:"name"              json:"name"`
	ImageURL          string `db:"image_url"         json:"image_url"`
	Provider          string `db:"provider"          json:"provider"`
	ProviderAccountID string `db:"provider_account_id" json:"provider_account_id"`
}

// IdentityRepository is the hexagonal port the application /
// interfaces layers use to resolve an Identity from a request's
// JWE-cookie claims. Implementations live under
// src/infrastructure/postgres/. The interface is deliberately small
// so a fake is trivial to write in tests (see
// application/identity_service_test.go).
//
// Contract:
//
//   - LookupByEmail(ctx, email) MUST return
//     *domain.IdentityNotFoundError when no row matches. The HTTP
//     handler maps that error to a 404 envelope (code=not_found)
//     without importing pgx itself.
//
//   - Email matching MUST be case-insensitive (Postgres CITEXT
//     column does this natively). Callers MAY pass mixed-case
//     emails; the result MUST be the same.
//
//   - All methods honour ctx.
type IdentityRepository interface {
	LookupByEmail(ctx context.Context, email string) (*Identity, error)
}

// IdentityNotFoundError signals that an identity lookup returned no
// row. The Email field carries the lookup input for diagnostics
// (the field name appears only in error logs, never in the HTTP
// response body — the locked not_found envelope replaces it).
//
// IdentityNotFoundError is part of the AppError contract so the
// HTTP handler can map it to the locked 404 envelope via Code().
// The handler does NOT need to import pgx to translate.
type IdentityNotFoundError struct {
	Email string
}

func (e *IdentityNotFoundError) Error() string {
	return `identity not found: "` + e.Email + `"`
}

// Code returns the locked vocabulary string `not_found`. The HTTP
// handler reads this via errors.As and maps to the
// `code: "not_found"` envelope.
func (e *IdentityNotFoundError) Code() string {
	return CodeNotFound
}
