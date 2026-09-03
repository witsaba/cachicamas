// Package auth — bootstrap.go implements the BootstrapService use
// case (spec R-BE-001 / R-BE-002 / R-BOOTSTRAP-1).
//
// The service is the ONLY place that runs the atomic lookup-or-
// create sequence for users / organizations / login_audits. The
// HTTP handler is a thin wrapper that decodes the JSON request,
// invokes Bootstrap, and maps errors to HTTP envelopes.
//
// CRITICAL invariants (locked at design §2.2 + spec §5):
//   - All three writes (user upsert, organization create-if-absent,
//     audit insert) happen inside a single *sql.Tx. Any failure
//     rolls back the entire transaction so no partial state remains
//     (S-BE-020).
//   - Bootstrap is idempotent on google_sub (R-BOOTSTRAP-1): a
//     second call with the same google_sub returns the SAME user_id
//     and does NOT create a duplicate organization.
//   - The registered → active transition happens in the SAME TX as
//     the organization creation. If the user is already active
//     (subsequent bootstrap), the promote-to-active is a no-op and
//     the organization lookup short-circuits.
package auth

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/cachicamas/backend/database_administrator/src/domain/auth"
)

// BootstrapInput is the application-layer DTO for the bootstrap
// service. The HTTP handler decodes the JSON request body into this
// struct. Field names are lowercased to avoid JSON ambiguity; the
// handler is responsible for the wire contract.
type BootstrapInput struct {
	GoogleSub     string
	Email         string
	EmailVerified bool
	Name          string
	PictureURL    string
	IPAddress     string
	UserAgent     string
	// GeoIP enrichment (R-BE-005). nil when GEOIP_DB_PATH is empty
	// or the IP lookup missed.
	CountryCode *string
	City        *string
}

// BootstrapResult is the service's return type. The HTTP handler
// serialises it to {user_id, pyme_id, status}.
type BootstrapResult struct {
	UserID         int64
	OrganizationID int64
	Status         auth.UserStatus
}

// ErrValidation is the sentinel the bootstrap service returns for
// any malformed input (missing google_sub, missing email). The HTTP
// handler maps ErrValidation to HTTP 400.
var ErrValidation = errors.New("auth: validation failed")

// BootstrapService orchestrates the atomic lookup-or-create flow.
// It depends on the three repository ports defined in the domain
// package; the concrete Postgres adapters are wired in the
// composition root.
type BootstrapService struct {
	db    *sql.DB
	users auth.UserRepository
	orgs  auth.OrganizationRepository
	audit auth.LoginAuditRepository
}

// NewBootstrapService wires the three repositories against a shared
// *sql.DB. The db is captured here (rather than passed per-call) so
// the service can BEGIN its own transaction; the repositories accept
// a Querier (which *sql.Tx satisfies).
func NewBootstrapService(
	db *sql.DB,
	users auth.UserRepository,
	orgs auth.OrganizationRepository,
	audit auth.LoginAuditRepository,
) *BootstrapService {
	return &BootstrapService{
		db:    db,
		users: users,
		orgs:  orgs,
		audit: audit,
	}
}

// Bootstrap executes the lookup-or-create flow inside a single
// transaction. The algorithm mirrors design §1.2 step 1–5:
//
//  1. Validate input (google_sub + email non-empty).
//  2. Find user by google_sub.
//  3. If miss: InsertRegistered (idempotent on google_sub).
//  4. UpdateLoginFields (email lowercased, name, picture, last_login_at).
//  5. Lookup-or-create organization (1:1 with the user, MVP tenancy).
//  6. PromoteToActive (idempotent — only if status='registered').
//  7. Insert login_audit row (success).
//  8. Commit.
//
// Returns the resolved user_id + organization_id + status. Any
// failure rolls back the transaction; the caller sees the error
// and no partial state remains (S-BE-020).
func (s *BootstrapService) Bootstrap(ctx context.Context, in BootstrapInput) (BootstrapResult, error) {
	// 1. Input validation — fail fast before opening a transaction.
	if in.GoogleSub == "" {
		return BootstrapResult{}, fmt.Errorf("%w: google_sub is required", ErrValidation)
	}
	if in.Email == "" {
		return BootstrapResult{}, fmt.Errorf("%w: email is required", ErrValidation)
	}

	// 2. Open the transaction.
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return BootstrapResult{}, fmt.Errorf("auth.BootstrapService: begin: %w", err)
	}
	// Rollback is a no-op after a successful Commit (the *sql.Tx
	// tracks the state). defer is the safest place so a panic in
	// the middle of the function still rolls back.
	defer func() { _ = tx.Rollback() }()

	// 3. Lookup-or-insert user.
	user, err := s.users.FindByGoogleSub(ctx, tx, in.GoogleSub)
	if err != nil {
		return BootstrapResult{}, fmt.Errorf("auth.BootstrapService: find user: %w", err)
	}
	if user == nil {
		user, err = s.users.InsertRegistered(ctx, tx, auth.NewUser(in.Email, in.GoogleSub))
		if err != nil {
			return BootstrapResult{}, fmt.Errorf("auth.BootstrapService: insert user: %w", err)
		}
	}

	// 4. Update login fields (never created_at — S-DB-002 enforced
	// by the repo's explicit UPDATE column list).
	user.Email = in.Email
	user.EmailVerified = in.EmailVerified
	user.Name = in.Name
	user.PictureURL = in.PictureURL
	if err := s.users.UpdateLoginFields(ctx, tx, user.ID, user); err != nil {
		return BootstrapResult{}, fmt.Errorf("auth.BootstrapService: update user: %w", err)
	}

	// 5. Lookup-or-create organization (1:1, MVP tenancy).
	org, err := s.orgs.FindByOwnerID(ctx, tx, user.ID)
	if err != nil {
		return BootstrapResult{}, fmt.Errorf("auth.BootstrapService: find org: %w", err)
	}
	if org == nil {
		orgName := auth.DerivePymeNameFromEmail(user.Email)
		org, err = s.orgs.Create(ctx, tx, auth.NewOrganization(user.ID, orgName))
		if err != nil {
			return BootstrapResult{}, fmt.Errorf("auth.BootstrapService: create org: %w", err)
		}
	}

	// 6. Promote to active (idempotent — only when status='registered').
	// Runs AFTER organization creation so a user with no org remains
	// 'registered'; runs BEFORE audit insert so the audit row records
	// the post-transition status.
	if err := s.users.PromoteToActive(ctx, tx, user.ID); err != nil {
		return BootstrapResult{}, fmt.Errorf("auth.BootstrapService: promote: %w", err)
	}
	// Re-read the user so the returned status reflects the
	// post-promotion state (the promote call above is a no-op when
	// the user was already active).
	user, err = s.users.FindByID(ctx, tx, user.ID)
	if err != nil || user == nil {
		return BootstrapResult{}, fmt.Errorf("auth.BootstrapService: re-read user: %w", err)
	}

	// 7. Audit row.
	uid := user.ID
	audit := auth.NewLoginAudit(uid, "google", in.GoogleSub, true)
	audit.EmailAttempted = in.Email
	audit.IPAddress = in.IPAddress
	audit.UserAgent = in.UserAgent
	audit.CountryCode = in.CountryCode
	audit.City = in.City
	if _, err := s.audit.Insert(ctx, tx, audit); err != nil {
		return BootstrapResult{}, fmt.Errorf("auth.BootstrapService: insert audit: %w", err)
	}

	// 8. Commit.
	if err := tx.Commit(); err != nil {
		return BootstrapResult{}, fmt.Errorf("auth.BootstrapService: commit: %w", err)
	}

	return BootstrapResult{
		UserID:         user.ID,
		OrganizationID: org.ID,
		Status:         user.Status,
	}, nil
}