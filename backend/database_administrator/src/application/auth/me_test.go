// Package auth — me_test.go locks the MeService contract per spec
// R-BE-003 / R-ME-1 / S-BE-030.
//
// The tests use in-memory fake repos (no Postgres dependency) so
// the application/auth test binary does not pull in
// infrastructure/postgres (which would create an import cycle).
// The MeService's *sql.DB is set to nil because the fake repos
// ignore the Querier parameter (they maintain their own in-memory
// state).
package auth

import (
	"context"
	"errors"
	"testing"

	"github.com/cachicamas/backend/database_administrator/src/domain/auth"
)

// TestMeService_ReturnsUserAndOrganization covers R-BE-003 /
// S-BE-030: GET /internal/me/:user_id for an existing user MUST
// return the user's data + their organization in one roundtrip.
//
// Uses shared fakes between BootstrapService and MeService so the
// "same DB" semantics work. Bootstrap's BeginTx is mocked by a
// neverConnectingDB — the test runs only the parts of Bootstrap
// that touch the fake repos (FindByGoogleSub, InsertRegistered,
// UpdateLoginFields, PromoteToActive, audit.Insert, org.FindByOwnerID,
// org.Create). For the fake repos, BeginTx returns a *sql.Tx that
// is never used.
//
// Wait — we can't fake BeginTx returning a non-nil *sql.Tx. So
// this test calls a SHORTCUT: it pre-populates the fake repos to
// simulate "a previous bootstrap has already happened", then
// invokes only MeService.Get. The Bootstrap service is exercised
// in a separate test (bootstrap_service_test.go with the real DB).
func TestMeService_ReturnsUserAndOrganization(t *testing.T) {
	u := newFakeUserRepo()
	o := newFakeOrgRepo()
	a := newFakeAuditRepo()

	// Pre-populate the fake state machine: one user + one org.
	user := auth.NewUser("founder@example.com", "google-sub-1")
	user.Name = "Founder"
	user.Status = auth.UserStatusActive
	created, _ := u.InsertRegistered(context.Background(), nil, user)

	org := auth.NewOrganization(created.ID, "founder")
	createdOrg, _ := o.Create(context.Background(), nil, org)

	me := NewMeService(nil, u, o)
	ctx := context.Background()
	got, err := me.Get(ctx, created.ID)
	if err != nil {
		t.Fatalf("MeService.Get: %v", err)
	}
	if got == nil {
		t.Fatal("MeService.Get: returned nil")
	}
	if got.User == nil {
		t.Fatal("MeService.Get: User is nil")
	}
	if got.User.ID != created.ID {
		t.Errorf("User.ID = %d, want %d", got.User.ID, created.ID)
	}
	if got.User.Email != "founder@example.com" {
		t.Errorf("User.Email = %q, want %q", got.User.Email, "founder@example.com")
	}
	if got.User.Status != auth.UserStatusActive {
		t.Errorf("User.Status = %q, want %q", got.User.Status, auth.UserStatusActive)
	}
	if got.Organization == nil {
		t.Fatal("Organization is nil")
	}
	if got.Organization.OwnerID != created.ID {
		t.Errorf("Organization.OwnerID = %d, want %d", got.Organization.OwnerID, created.ID)
	}
	if got.Organization.ID != createdOrg.ID {
		t.Errorf("Organization.ID = %d, want %d", got.Organization.ID, createdOrg.ID)
	}
	_ = a // audit not exercised in this test path
}

// TestMeService_UnknownUser_ReturnsNotFound covers the 404
// envelope rule.
func TestMeService_UnknownUser_ReturnsNotFound(t *testing.T) {
	u := newFakeUserRepo()
	o := newFakeOrgRepo()
	me := NewMeService(nil, u, o)
	_, err := me.Get(context.Background(), 9999)
	if err == nil {
		t.Fatal("MeService.Get(9999): expected NotFoundError, got nil")
	}
	var notFound *auth.NotFoundError
	if !errors.As(err, &notFound) {
		t.Errorf("error = %v, want errors.As(*auth.NotFoundError)", err)
	}
}

// TestMeService_InvalidID_ReturnsValidation covers the input
// validation gate.
func TestMeService_InvalidID_ReturnsValidation(t *testing.T) {
	u := newFakeUserRepo()
	o := newFakeOrgRepo()
	me := NewMeService(nil, u, o)
	for _, id := range []int64{0, -1, -42} {
		_, err := me.Get(context.Background(), id)
		if err == nil {
			t.Errorf("MeService.Get(%d): expected error, got nil", id)
		}
		if !errors.Is(err, ErrValidation) {
			t.Errorf("MeService.Get(%d): error = %v, want errors.Is(ErrValidation)", id, err)
		}
	}
}