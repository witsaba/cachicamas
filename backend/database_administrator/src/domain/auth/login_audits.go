// Package auth — login_audits.go defines the LoginAudit value
// object + the LoginAuditRepository port used by the bootstrap
// service.
//
// CRITICAL invariants (locked at spec §4 / §5):
//   - LoginAudit.UserID is nullable so failed logins before the user
//     row exists still produce an audit row (S-DB-030). The DB column
//     allows NULL with ON DELETE SET NULL on the FK.
//   - LoginAudit.FailureReason is locked to one of four values
//     (state_mismatch | code_exchange_failed | userinfo_failed |
//     internal_error) per R-BE-007. The empty string is the success
//     marker; anything else not in the vocabulary is rejected at
//     construction.
//   - GeoIP fields (CountryCode, City) are optional: a GeoIP-disabled
//     run (empty GEOIP_DB_PATH) MUST be able to build an audit row
//     without them (R-BE-005 / S-BE-050).
package auth

import (
	"context"
	"errors"
	"fmt"
	"time"
)

// LoginAudit is the canonical domain value for an auth.login_audits
// row. The fields mirror the SQL schema; the JSON tags let the
// bootstrap handler serialise debug responses (rare; the audit row
// is normally written, not returned).
type LoginAudit struct {
	ID              int64     `json:"id"`
	UserID          *int64    `json:"user_id,omitempty"`
	EmailAttempted  string    `json:"email_attempted,omitempty"`
	Provider        string    `json:"provider"`
	ProviderSubject string    `json:"provider_subject,omitempty"`
	IPAddress       string    `json:"ip_address,omitempty"`
	UserAgent       string    `json:"user_agent,omitempty"`
	CountryCode     *string   `json:"country_code,omitempty"`
	City            *string   `json:"city,omitempty"`
	Success         bool      `json:"success"`
	FailureReason   string    `json:"failure_reason,omitempty"`
	LoginAt         time.Time `json:"login_at"`
}

// Locked failure_reason vocabulary (spec R-BE-007). Adding a new
// value here is a spec change; removing one breaks persisted data.
// The empty string is the success marker and is NOT in the
// vocabulary — callers MUST NOT pass "" to NewFailedLoginAudit.
const (
	FailureReasonStateMismatch      = "state_mismatch"
	FailureReasonCodeExchangeFailed = "code_exchange_failed"
	FailureReasonUserInfoFailed     = "userinfo_failed"
	FailureReasonInternalError      = "internal_error"
)

// allowedFailureReasons is the closed set of accepted failure_reason
// values. Kept private so callers MUST go through NewFailedLoginAudit
// — the closure is the spec-mandated guarantee that an unknown
// value cannot reach the audit table.
var allowedFailureReasons = map[string]struct{}{
	FailureReasonStateMismatch:      {},
	FailureReasonCodeExchangeFailed: {},
	FailureReasonUserInfoFailed:     {},
	FailureReasonInternalError:      {},
}

// isAllowedFailureReason reports whether the given string is in the
// closed failure_reason vocabulary. Exported as a helper only for
// the test suite; production code uses NewFailedLoginAudit which
// rejects unknown values internally.
func isAllowedFailureReason(reason string) bool {
	_, ok := allowedFailureReasons[reason]
	return ok
}

// NewLoginAudit builds a LoginAudit for the success path. The
// timestamp is left at the zero value so the repository INSERT path
// uses the column's DEFAULT now(). The bootstrap service reads the
// DB-generated value back via RETURNING (rare; audit rows are
// normally write-only).
func NewLoginAudit(userID int64, provider, providerSubject string, success bool) *LoginAudit {
	uid := userID
	return &LoginAudit{
		UserID:          &uid,
		Provider:        provider,
		ProviderSubject: providerSubject,
		Success:         success,
	}
}

// NewFailedLoginAudit builds a LoginAudit for a failed login. Returns
// nil when the reason is not in the locked vocabulary so the bootstrap
// handler can branch on the nil-receiver result without importing a
// secondary validation helper.
func NewFailedLoginAudit(userID int64, provider, providerSubject, reason string) *LoginAudit {
	if !isAllowedFailureReason(reason) {
		return nil
	}
	uid := userID
	return &LoginAudit{
		UserID:          &uid,
		Provider:        provider,
		ProviderSubject: providerSubject,
		Success:         false,
		FailureReason:   reason,
	}
}

// Validate enforces the construction-time invariants the bootstrap
// service relies on. A successful audit MUST have an empty
// failure_reason; a failed audit MUST carry one from the vocabulary.
// An empty Provider is rejected (the DB column is NOT NULL).
func (a *LoginAudit) Validate() error {
	if a == nil {
		return errors.New("auth.LoginAudit.Validate: nil receiver")
	}
	if a.Provider == "" {
		return errors.New("auth.LoginAudit.Validate: provider is required")
	}
	if a.Success {
		if a.FailureReason != "" {
			return fmt.Errorf("auth.LoginAudit.Validate: success audits cannot carry a failure_reason (got %q)", a.FailureReason)
		}
		return nil
	}
	if !isAllowedFailureReason(a.FailureReason) {
		return fmt.Errorf("auth.LoginAudit.Validate: failure_reason %q is not in the locked vocabulary", a.FailureReason)
	}
	return nil
}

// LoginAuditRepository is the application-layer port the bootstrap
// service depends on. The Postgres adapter in
// infrastructure/postgres/auth_login_audit_repo.go is the only known
// implementation in this slice; tests may use a fake.
type LoginAuditRepository interface {
	// Insert persists a new audit row. userID may be nil for failed
	// logins that occurred before the user row was created (S-DB-030).
	// country_code / city may be nil when GeoIP is disabled or the
	// IP lookup missed (R-BE-005).
	Insert(ctx context.Context, q Querier, a *LoginAudit) (*LoginAudit, error)
}