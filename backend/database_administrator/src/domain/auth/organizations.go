// Package auth — organizations.go defines the Organization value
// object + the OrganizationRepository port used by the bootstrap
// service.
//
// CRITICAL invariants (locked at design §2.2 + spec §4):
//   - Organization.OwnerID is a 1:1 with auth.users.id (MVP
//     tenancy). The DB does NOT enforce 1:1 — a future multi-org
//     slice may lift this — so the bootstrap service MUST check
//     FindByOwnerID first and skip Create on hit (spec R-BE-002).
//   - Organization.Slug is partial-unique among non-deleted rows
//     (R-DB-003); the repository returns *ConflictError on the
//     SQLSTATE 23505 the partial unique index produces.
//   - Organization.CreatedAt is set ONLY at the DB layer
//     (DEFAULT now()); the domain constructor leaves it at zero.
package auth

import (
	"context"
	"errors"
	"strings"
	"time"
)

// Organization is the canonical domain value for an auth.organizations
// row. The fields mirror the SQL schema; the JSON tags let the me
// handler serialise the lookup response without a separate DTO.
type Organization struct {
	ID        int64     `json:"id"`
	OwnerID   int64     `json:"owner_id"`
	Name      string    `json:"name"`
	Slug      string    `json:"slug"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// NewOrganization is the canonical constructor for an in-memory
// Organization. Slug is left empty — the repository derives it via
// DeriveSlugFromEmail so the same caller cannot accidentally collide
// with another tenant's slug.
func NewOrganization(ownerID int64, name string) *Organization {
	return &Organization{
		OwnerID: ownerID,
		Name:    name,
	}
}

// Validate enforces the construction-time invariants the bootstrap
// service relies on: an Organization without a name or owner_id
// cannot be persisted (the DB NOT NULL constraints would reject it,
// but failing here yields a cleaner 400 response than a 500 from a
// pgx error).
func (o *Organization) Validate() error {
	if o == nil {
		return errors.New("auth.Organization.Validate: nil receiver")
	}
	if o.OwnerID <= 0 {
		return errors.New("auth.Organization.Validate: owner_id is required")
	}
	if o.Name == "" {
		return errors.New("auth.Organization.Validate: name is required")
	}
	return nil
}

// pymeNameFallback is the display name used when the email local-part
// is empty (design §1.2 step 4). Exported as a constant so the test
// can reference the same value the SQL COALESCE would yield.
const pymeNameFallback = "pyme"

// DerivePymeNameFromEmail computes the display name for a freshly-
// bootstrapped organization: the email local-part, lowercased, with
// the fallback "pyme" when the local-part is empty. Mirrors the SQL
// COALESCE(NULLIF(split_part($2, '@', 1), ''), 'pyme') in design
// §1.2 step 4, but computed in Go so the bootstrap service can set
// the value before INSERT (RETURNING reads back the row with name
// already populated).
//
// The function is a pure transformation: no DB, no side effects,
// trivially testable. Returns the lowercase local-part when present,
// otherwise "pyme".
func DerivePymeNameFromEmail(email string) string {
	local := strings.ToLower(email)
	if at := strings.IndexByte(local, '@'); at >= 0 {
		local = local[:at]
	}
	if local == "" {
		return pymeNameFallback
	}
	return local
}

// OrganizationRepository is the application-layer port the bootstrap
// service depends on. The Postgres adapter in
// infrastructure/postgres/organization_repo.go is the only known
// implementation in this slice; tests may use a fake.
type OrganizationRepository interface {
	// FindByOwnerID returns the live organization owned by the
	// given user_id, or (nil, nil) when no such row exists. The
	// bootstrap service relies on this to enforce 1:1 (MVP tenancy).
	FindByOwnerID(ctx context.Context, q Querier, ownerID int64) (*Organization, error)

	// FindByID returns the live organization with the given id, or
	// (nil, nil) when no such row exists.
	FindByID(ctx context.Context, q Querier, id int64) (*Organization, error)

	// Create persists a new organization. Returns the populated row
	// (including the DB-generated id + timestamps). Returns
	// *ConflictError when the slug is not unique among live rows
	// (the partial unique index on slug raises SQLSTATE 23505); the
	// bootstrap service treats this as a transient retry condition.
	Create(ctx context.Context, q Querier, o *Organization) (*Organization, error)
}