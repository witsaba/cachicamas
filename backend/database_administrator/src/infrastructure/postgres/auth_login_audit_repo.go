// Package postgres — auth_login_audit_repo.go is the pgx-backed
// adapter that satisfies auth.LoginAuditRepository. The pgx import
// stays restricted to this package per the hexagonal rule.
//
// The file is named with the `auth_` prefix to coexist with any
// future legacy login audit adapter (none currently exists; the
// prefix keeps the naming convention consistent with the
// organization slice).
//
// The adapter inserts a single audit row per call. No UPDATE /
// DELETE paths are exposed: the audit table is append-mostly by
// design (spec §4 R-DB-004).
package postgres

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/cachicamas/backend/database_administrator/src/domain/auth"
)

// AuthLoginAuditRepo is the pgx-backed adapter that satisfies
// auth.LoginAuditRepository for the auth.login_audits table.
type AuthLoginAuditRepo struct {
	db *sql.DB
}

// NewAuthLoginAuditRepo constructs an AuthLoginAuditRepo. The
// caller passes in an already-opened *sql.DB (typically produced by
// migration/postgres.Open in the composition root).
func NewAuthLoginAuditRepo(db *sql.DB) *AuthLoginAuditRepo {
	return &AuthLoginAuditRepo{db: db}
}

// Compile-time check that AuthLoginAuditRepo satisfies
// auth.LoginAuditRepository.
var _ auth.LoginAuditRepository = (*AuthLoginAuditRepo)(nil)

// authLoginAuditColumnList is the canonical column order for
// RETURNING queries. The adapter does not need it today (no SELECT
// path) but is kept for symmetry with the user + organization
// adapters and to make a future FindByUserID test trivial.
const authLoginAuditColumnList = "id, user_id, email_attempted, provider, provider_subject, ip_address, user_agent, country_code, city, success, failure_reason, login_at"

// scanAuthLoginAuditRow reads one row into a fresh *auth.LoginAudit.
// The column order MUST match authLoginAuditColumnList. Defined so
// a future ListByUserID (spec §5 R-DB-004 audit history) can reuse
// it; the bootstrap service does not call this directly.
func scanAuthLoginAuditRow(row interface {
	Scan(dest ...any) error
}) (*auth.LoginAudit, error) {
	var (
		a               auth.LoginAudit
		userID          sql.NullInt64
		emailAttempted  sql.NullString
		providerSub     sql.NullString
		ipAddress       sql.NullString
		userAgent       sql.NullString
		countryCode     sql.NullString
		city            sql.NullString
		failureReason   sql.NullString
	)
	if err := row.Scan(
		&a.ID,
		&userID,
		&emailAttempted,
		&a.Provider,
		&providerSub,
		&ipAddress,
		&userAgent,
		&countryCode,
		&city,
		&a.Success,
		&failureReason,
		&a.LoginAt,
	); err != nil {
		return nil, err
	}
	if userID.Valid {
		v := userID.Int64
		a.UserID = &v
	}
	if emailAttempted.Valid {
		a.EmailAttempted = emailAttempted.String
	}
	if providerSub.Valid {
		a.ProviderSubject = providerSub.String
	}
	if ipAddress.Valid {
		a.IPAddress = ipAddress.String
	}
	if userAgent.Valid {
		a.UserAgent = userAgent.String
	}
	if countryCode.Valid {
		v := countryCode.String
		a.CountryCode = &v
	}
	if city.Valid {
		v := city.String
		a.City = &v
	}
	if failureReason.Valid {
		a.FailureReason = failureReason.String
	}
	return &a, nil
}

// Insert persists a new audit row. userID may be nil for failed
// logins that occurred before the user row was created (S-DB-030).
// country_code / city may be nil when GeoIP is disabled or the
// IP lookup missed (R-BE-005).
//
// The adapter writes each column with the explicit nullable-coalesced
// value rather than relying on database/sql's automatic NULL handling
// for zero-length strings, so a misconfigured driver cannot
// accidentally write '' instead of NULL into the GeoIP columns.
func (r *AuthLoginAuditRepo) Insert(ctx context.Context, q auth.Querier, a *auth.LoginAudit) (*auth.LoginAudit, error) {
	if a == nil {
		return nil, errors.New("postgres.AuthLoginAuditRepo.Insert: nil audit")
	}
	if err := a.Validate(); err != nil {
		return nil, fmt.Errorf("postgres.AuthLoginAuditRepo.Insert: validate: %w", err)
	}
	row := q.QueryRowContext(ctx,
		`INSERT INTO auth.login_audits
            (user_id, email_attempted, provider, provider_subject,
             ip_address, user_agent, country_code, city,
             success, failure_reason)
         VALUES
            ($1, NULLIF($2, ''), $3, NULLIF($4, ''),
             NULLIF($5, ''), NULLIF($6, ''), NULLIF($7, ''), NULLIF($8, ''),
             $9, NULLIF($10, ''))
         RETURNING `+authLoginAuditColumnList,
		nullableInt64Ptr(a.UserID),
		a.EmailAttempted,
		a.Provider,
		a.ProviderSubject,
		a.IPAddress,
		a.UserAgent,
		nullableStringPtrFromRef(a.CountryCode),
		nullableStringPtrFromRef(a.City),
		a.Success,
		a.FailureReason,
	)
	created, err := scanAuthLoginAuditRow(row)
	if err != nil {
		return nil, fmt.Errorf("postgres.AuthLoginAuditRepo.Insert: %w", err)
	}
	return created, nil
}

// nullableInt64Ptr returns untyped nil for a nil *int64 so the
// database/sql driver writes SQL NULL instead of the zero value.
// Centralised to keep the Insert call site readable.
func nullableInt64Ptr(p *int64) any {
	if p == nil {
		return nil
	}
	return *p
}

// nullableStringPtrFromRef returns untyped nil for a nil *string so
// the database/sql driver writes SQL NULL. Mirrors nullableStringPtr
// (which takes a string value, not a pointer) but for the *string
// case used by the GeoIP columns.
func nullableStringPtrFromRef(p *string) any {
	if p == nil {
		return nil
	}
	return *p
}