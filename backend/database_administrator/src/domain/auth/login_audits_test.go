// Package auth — login_audits_test.go locks the LoginAudit value-
// object contract per spec R-DB-004 / R-LOGIN-1 / R-BE-007.
//
// The tests are pure-domain: no DB, no HTTP. The integration tests
// in src/infrastructure/postgres/auth_login_audit_repo_test.go cover
// the DB-backed behaviour under INTEGRATION=1.
package auth

import (
	"strings"
	"testing"
)

// failureReason values locked by spec R-BE-007. The constants below
// are the canonical source of truth — a typo here breaks the
// bootstrap handler's failure_reason column write.
const (
	failureReasonStateMismatch      = "state_mismatch"
	failureReasonCodeExchangeFailed = "code_exchange_failed"
	failureReasonUserInfoFailed     = "userinfo_failed"
	failureReasonInternalError      = "internal_error"
)

// TestLoginAudit_NewLoginAudit covers the constructor invariants: a
// LoginAudit built without an explicit failure_reason MUST default to
// "" (success path). A builder that wants to record a failure sets
// the FailureReason field directly.
func TestLoginAudit_NewLoginAudit(t *testing.T) {
	a := NewLoginAudit(42, "google", "google-sub-123", true)
	if a == nil {
		t.Fatal("NewLoginAudit: returned nil")
	}
	if a.UserID == nil || *a.UserID != 42 {
		t.Errorf("NewLoginAudit().UserID = %v, want ptr to 42", a.UserID)
	}
	if a.Provider != "google" {
		t.Errorf("NewLoginAudit().Provider = %q, want %q", a.Provider, "google")
	}
	if a.ProviderSubject != "google-sub-123" {
		t.Errorf("NewLoginAudit().ProviderSubject = %q, want %q", a.ProviderSubject, "google-sub-123")
	}
	if !a.Success {
		t.Errorf("NewLoginAudit().Success = false, want true (default)")
	}
	if a.FailureReason != "" {
		t.Errorf("NewLoginAudit().FailureReason = %q, want empty (success default)", a.FailureReason)
	}
}

// TestLoginAudit_NewFailedLoginAudit covers the failure-builder
// invariant: a LoginAudit built via NewFailedLoginAudit MUST carry
// Success=false and a non-empty FailureReason from the locked
// vocabulary. An empty or unknown reason must be rejected so the
// audit table never holds an undocumented value.
func TestLoginAudit_NewFailedLoginAudit(t *testing.T) {
	cases := []struct {
		reason string
		want   string
	}{
		{failureReasonStateMismatch, failureReasonStateMismatch},
		{failureReasonCodeExchangeFailed, failureReasonCodeExchangeFailed},
		{failureReasonUserInfoFailed, failureReasonUserInfoFailed},
		{failureReasonInternalError, failureReasonInternalError},
	}
	for _, c := range cases {
		t.Run(c.reason, func(t *testing.T) {
			a := NewFailedLoginAudit(42, "google", "google-sub-123", c.reason)
			if a == nil {
				t.Fatal("NewFailedLoginAudit: returned nil")
			}
			if a.Success {
				t.Errorf("NewFailedLoginAudit().Success = true, want false")
			}
			if a.FailureReason != c.want {
				t.Errorf("NewFailedLoginAudit().FailureReason = %q, want %q", a.FailureReason, c.want)
			}
		})
	}
}

// TestLoginAudit_NewFailedLoginAudit_UnknownReason covers the locked-
// vocabulary gate: any reason outside the four allowed values MUST
// be rejected at construction time so a future typo in the bootstrap
// handler cannot push an undocumented string into the audit table.
func TestLoginAudit_NewFailedLoginAudit_UnknownReason(t *testing.T) {
	for _, reason := range []string{"", "garbage", "invalid_state", "STATE_MISMATCH"} {
		t.Run("reason="+reason, func(t *testing.T) {
			a := NewFailedLoginAudit(42, "google", "google-sub-123", reason)
			if a == nil {
				// The constructor may return nil for unknown reasons.
				return
			}
			if a.FailureReason != "" {
				t.Errorf("NewFailedLoginAudit(%q).FailureReason = %q, want empty (rejected)", reason, a.FailureReason)
			}
		})
	}
}

// TestLoginAudit_Validate covers the construction-time invariants
// the bootstrap service relies on: a LoginAudit must always carry
// a Provider (the DB column is NOT NULL with default 'google'); a
// successful audit must have an empty FailureReason; a failed audit
// must carry a non-empty FailureReason from the locked vocabulary.
func TestLoginAudit_Validate(t *testing.T) {
	t.Run("missing provider", func(t *testing.T) {
		a := &LoginAudit{Success: true, ProviderSubject: "google-sub-123"}
		if err := a.Validate(); err == nil {
			t.Fatal("LoginAudit{Provider: \"\"}.Validate(): expected error, got nil")
		}
	})
	t.Run("success with failure reason", func(t *testing.T) {
		a := &LoginAudit{Provider: "google", ProviderSubject: "google-sub-123", Success: true, FailureReason: failureReasonInternalError}
		if err := a.Validate(); err == nil {
			t.Fatal("LoginAudit{Success: true, FailureReason set}.Validate(): expected error, got nil")
		}
	})
	t.Run("failure with empty reason", func(t *testing.T) {
		a := &LoginAudit{Provider: "google", ProviderSubject: "google-sub-123", Success: false}
		if err := a.Validate(); err == nil {
			t.Fatal("LoginAudit{Success: false, FailureReason empty}.Validate(): expected error, got nil")
		}
	})
	t.Run("valid success", func(t *testing.T) {
		a := NewLoginAudit(42, "google", "google-sub-123", true)
		if err := a.Validate(); err != nil {
			t.Fatalf("LoginAudit{Success: true}.Validate(): unexpected error %v", err)
		}
	})
	t.Run("valid failure", func(t *testing.T) {
		a := NewFailedLoginAudit(42, "google", "google-sub-123", failureReasonStateMismatch)
		if err := a.Validate(); err != nil {
			t.Fatalf("LoginAudit{valid failure}.Validate(): unexpected error %v", err)
		}
	})
}

// TestLoginAudit_FailureReasonVocabulary covers the closed-vocabulary
// invariant: the four locked values are accepted; everything else is
// rejected. This is the gate that prevents an audit table from
// accumulating undocumented values.
func TestLoginAudit_FailureReasonVocabulary(t *testing.T) {
	allowed := []string{
		failureReasonStateMismatch,
		failureReasonCodeExchangeFailed,
		failureReasonUserInfoFailed,
		failureReasonInternalError,
	}
	for _, reason := range allowed {
		t.Run("allowed/"+reason, func(t *testing.T) {
			if !isAllowedFailureReason(reason) {
				t.Errorf("isAllowedFailureReason(%q) = false, want true", reason)
			}
		})
	}
	for _, reason := range []string{"", "garbage", "unknown", "STATE_MISMATCH", strings.ToUpper(failureReasonStateMismatch)} {
		t.Run("denied/"+reason, func(t *testing.T) {
			if isAllowedFailureReason(reason) {
				t.Errorf("isAllowedFailureReason(%q) = true, want false", reason)
			}
		})
	}
}

// TestLoginAudit_GeoFieldsOptional covers the R-BE-005 / R-GEOIP-1
// invariant: GeoIP fields (country_code / city) are nullable on the
// schema and must be omittable from the constructor. A GeoIP-disabled
// run (empty GEOIP_DB_PATH) MUST be able to build a LoginAudit with
// nil CountryCode / City without error.
func TestLoginAudit_GeoFieldsOptional(t *testing.T) {
	a := NewLoginAudit(42, "google", "google-sub-123", true)
	if a.CountryCode != nil {
		t.Errorf("NewLoginAudit().CountryCode = %v, want nil (GeoIP disabled default)", a.CountryCode)
	}
	if a.City != nil {
		t.Errorf("NewLoginAudit().City = %v, want nil (GeoIP disabled default)", a.City)
	}
}