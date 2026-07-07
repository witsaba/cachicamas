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
	"time"
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
//   - LookupByProviderAccountID(ctx, provider, accountID) MUST
//     return *domain.IdentityNotFoundError when no row matches.
//     This is the canonical lookup path for the JWE verifier
//     middleware (PR-3 IdentityFromCookie) — it's stable across
//     email changes and forward-compatible for multi-provider
//     support. Callers SHOULD prefer it over LookupByEmail when
//     the JWE claims carry the provider + account_id.
//
//   - Email matching MUST be case-insensitive (Postgres CITEXT
//     column does this natively). Callers MAY pass mixed-case
//     emails; the result MUST be the same.
//
//   - All methods honour ctx.
//
// Upsert (added in cachicamas-identity-signin-callback): the
// service layer passes a closed IdentityEvent value (the slice's
// HTTP handler validates and canonicalizes the wire body, then
// builds an IdentityEvent and dispatches). The repo selects the
// identity.user row by case-insensitive email, INSERTs a new
// identity.user row when missing, then INSERTs an identity.account
// row with ON CONFLICT (provider, provider_account_id) DO NOTHING.
// Returns the resolved Identity (new or reused).
type IdentityRepository interface {
	LookupByEmail(ctx context.Context, email string) (*Identity, error)
	LookupByProviderAccountID(ctx context.Context, provider, providerAccountID string) (*Identity, error)
	Upsert(ctx context.Context, ev IdentityEvent) (*Identity, error)
}

// IdentityEvent is the closed value type the HTTP handler hands
// to the application service for an OAuth-driven identity
// persistence roundtrip. The fields mirror the wire contract
// (docs/adr/0003-add-identity-callback-hmac.md): the handler
// validates + canonicalizes the body, then extracts these fields
// and never holds the raw OAuth tokens (they are discarded after
// HMAC verification; the identity.account schema does not store
// tokens in this slice — see ADR 0003 §"Forward notes").
//
// The field list is LOCKED for this slice (handler depends on it
// by reflection-free direct construction; future additions need a
// spec amendment + new IdentityEvent fields + new repo signature).
//
// PR1a (2026-07-06-workspaces) extended the struct with 5 OAuth
// token fields (AccessToken, RefreshToken, ExpiresAt, TokenType,
// Scope). These are persisted in identity.account columns added
// by migration 20260706120000; pre-PR1a rows have NULL columns, so
// handlers can rely on no-op niltolerance. The handler accepts the
// fields in the wire body (the previous slice already forwarded
// them through identity-callback-client.ts to match the Go-side
// request struct); this slice wires persistence all the way through
// domain -> service -> repo -> SQL.
type IdentityEvent struct {
	Email             string
	Name              string
	ImageURL          string
	Provider          string
	ProviderAccountID string
	AccessToken       *string
	RefreshToken      *string
	ExpiresAt         *time.Time
	TokenType         *string
	Scope             *string
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
