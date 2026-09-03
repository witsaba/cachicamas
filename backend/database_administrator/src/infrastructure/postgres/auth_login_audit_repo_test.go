//go:build integration

// Package postgres — auth_login_audit_repo_test.go locks the
// AuthLoginAuditRepo adapter contract at the DB level per spec
// R-DB-004 / R-BE-007 / S-DB-030 / S-DB-031.
//
// Tests are INTEGRATION=1 gated; skipped (not failed) without a
// live Postgres connection.
package postgres

import (
	"context"
	"testing"

	"github.com/cachicamas/backend/database_administrator/src/domain/auth"
)

// TestAuthLoginAuditRepo_Insert covers the success-path insert.
func TestAuthLoginAuditRepo_Insert(t *testing.T) {
	db := authIntegrationDB(t)
	resetAuthTablesForRepoTest(t, db)

	userRepo := NewUserRepo(db)
	auditRepo := NewAuthLoginAuditRepo(db)

	ctx := context.Background()
	u := auth.NewUser("imm@example.com", "imm-sub-1")
	created, err := userRepo.InsertRegistered(ctx, db, u)
	if err != nil {
		t.Fatalf("InsertRegistered: %v", err)
	}

	audit := auth.NewLoginAudit(created.ID, "google", "imm-sub-1", true)
	audit.EmailAttempted = "imm@example.com"
	audit.IPAddress = "127.0.0.1"
	audit.UserAgent = "Mozilla/5.0"

	inserted, err := auditRepo.Insert(ctx, db, audit)
	if err != nil {
		t.Fatalf("Insert: %v", err)
	}
	if inserted.ID <= 0 {
		t.Errorf("ID = %d, want > 0", inserted.ID)
	}
	if !inserted.Success {
		t.Errorf("Success = false, want true")
	}
	if inserted.FailureReason != "" {
		t.Errorf("FailureReason = %q, want empty (success)", inserted.FailureReason)
	}
	if inserted.UserID == nil || *inserted.UserID != created.ID {
		t.Errorf("UserID = %v, want ptr to %d", inserted.UserID, created.ID)
	}
}

// TestAuthLoginAuditRepo_Insert_NullUserID covers S-DB-030: a
// failed login before the user row exists MUST be auditable with
// user_id = NULL. We construct the LoginAudit directly with
// UserID: nil rather than going through NewFailedLoginAudit(0, ...)
// because the constructor coerces a zero int64 into a *int64 pointer
// (not nil), which would write user_id = 0 to the DB and trip the
// auth.users(id) FK. The canonical "no user" sentinel in this
// codebase is a nil *int64, not a pointer-to-zero.
func TestAuthLoginAuditRepo_Insert_NullUserID(t *testing.T) {
	db := authIntegrationDB(t)
	resetAuthTablesForRepoTest(t, db)

	auditRepo := NewAuthLoginAuditRepo(db)
	audit := &auth.LoginAudit{
		UserID:          nil, // no user exists yet (S-DB-030)
		Provider:        "google",
		ProviderSubject: "no-such-sub",
		Success:         false,
		FailureReason:   auth.FailureReasonStateMismatch,
		EmailAttempted:  "nope@example.com",
	}

	inserted, err := auditRepo.Insert(context.Background(), db, audit)
	if err != nil {
		t.Fatalf("Insert: %v", err)
	}
	if inserted.UserID != nil {
		t.Errorf("UserID = %v, want nil (failed login before user)", *inserted.UserID)
	}
	if inserted.Success {
		t.Errorf("Success = true, want false")
	}
}

// TestAuthLoginAuditRepo_Insert_GeoIPOptional covers R-BE-005: a
// successful login without GeoIP fields writes a row with
// country_code / city NULL.
func TestAuthLoginAuditRepo_Insert_GeoIPOptional(t *testing.T) {
	db := authIntegrationDB(t)
	resetAuthTablesForRepoTest(t, db)

	userRepo := NewUserRepo(db)
	auditRepo := NewAuthLoginAuditRepo(db)

	ctx := context.Background()
	u := auth.NewUser("imm@example.com", "imm-sub-1")
	created, err := userRepo.InsertRegistered(ctx, db, u)
	if err != nil {
		t.Fatalf("InsertRegistered: %v", err)
	}

	audit := auth.NewLoginAudit(created.ID, "google", "imm-sub-1", true)
	// no CountryCode, no City
	inserted, err := auditRepo.Insert(ctx, db, audit)
	if err != nil {
		t.Fatalf("Insert: %v", err)
	}
	if inserted.CountryCode != nil {
		t.Errorf("CountryCode = %v, want nil (GeoIP disabled)", *inserted.CountryCode)
	}
	if inserted.City != nil {
		t.Errorf("City = %v, want nil (GeoIP disabled)", *inserted.City)
	}
}