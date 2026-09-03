// Package auth — me_test.go locks the MeService contract per spec
// R-BE-003 / R-ME-1 / S-BE-030.
//
// The tests use INTEGRATION=1 (gated) and the migration runner
// fixture pattern from src/migration/runner_test.go.
package auth

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/cachicamas/backend/database_administrator/src/domain/auth"
	"github.com/cachicamas/backend/database_administrator/src/infrastructure/postgres"
)

// TestMeService_ReturnsUserAndOrganization covers R-BE-003 /
// S-BE-030: GET /internal/me/:user_id for an existing user MUST
// return the user's data + their organization in one roundtrip.
// Both objects come back populated; the response shape matches
// the documented {user, organization} envelope.
func TestMeService_ReturnsUserAndOrganization(t *testing.T) {
	db := integrationDB(t)
	resetAuthTables(t, db)
	bootstrapSvc, _, _, _ := newTestBootstrapService(t, db)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	out, err := bootstrapSvc.Bootstrap(ctx, BootstrapInput{
		GoogleSub: "google-sub-1",
		Email:     "founder@example.com",
		Name:      "Founder",
		PictureURL: "https://example.com/pic.png",
	})
	if err != nil {
		t.Fatalf("Bootstrap: %v", err)
	}

	me := NewMeService(
		db,
		postgres.NewUserRepo(db),
		postgres.NewAuthOrganizationRepo(db),
	)
	got, err := me.Get(ctx, out.UserID)
	if err != nil {
		t.Fatalf("MeService.Get: unexpected error %v", err)
	}
	if got == nil {
		t.Fatal("MeService.Get: returned nil result")
	}
	if got.User == nil {
		t.Fatal("MeService.Get: User is nil")
	}
	if got.User.ID != out.UserID {
		t.Errorf("MeService.Get.User.ID = %d, want %d", got.User.ID, out.UserID)
	}
	if got.User.Email != "founder@example.com" {
		t.Errorf("MeService.Get.User.Email = %q, want %q", got.User.Email, "founder@example.com")
	}
	if got.User.Status != auth.UserStatusActive {
		t.Errorf("MeService.Get.User.Status = %q, want %q", got.User.Status, auth.UserStatusActive)
	}
	if got.Organization == nil {
		t.Fatal("MeService.Get: Organization is nil")
	}
	if got.Organization.ID != out.OrganizationID {
		t.Errorf("MeService.Get.Organization.ID = %d, want %d", got.Organization.ID, out.OrganizationID)
	}
	if got.Organization.OwnerID != out.UserID {
		t.Errorf("MeService.Get.Organization.OwnerID = %d, want %d", got.Organization.OwnerID, out.UserID)
	}
}

// TestMeService_UnknownUser_ReturnsNotFound covers the 404 envelope
// rule: a request for a user_id that does not exist MUST return
// *auth.NotFoundError so the HTTP handler maps it to 404.
func TestMeService_UnknownUser_ReturnsNotFound(t *testing.T) {
	db := integrationDB(t)
	resetAuthTables(t, db)
	me := NewMeService(
		db,
		postgres.NewUserRepo(db),
		postgres.NewAuthOrganizationRepo(db),
	)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_, err := me.Get(ctx, 9999)
	if err == nil {
		t.Fatal("MeService.Get(9999): expected NotFoundError, got nil")
	}
	var notFound *auth.NotFoundError
	if !errors.As(err, &notFound) {
		t.Errorf("MeService.Get(9999): error = %v, want errors.As(*auth.NotFoundError)", err)
	}
}

// TestMeService_InvalidID_ReturnsValidation covers the input
// validation gate: a non-positive user_id is rejected before any
// DB roundtrip.
func TestMeService_InvalidID_ReturnsValidation(t *testing.T) {
	db := integrationDB(t)
	resetAuthTables(t, db)
	me := NewMeService(
		db,
		postgres.NewUserRepo(db),
		postgres.NewAuthOrganizationRepo(db),
	)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	for _, id := range []int64{0, -1, -42} {
		_, err := me.Get(ctx, id)
		if err == nil {
			t.Errorf("MeService.Get(%d): expected error, got nil", id)
		}
		if !errors.Is(err, ErrValidation) {
			t.Errorf("MeService.Get(%d): error = %v, want errors.Is(ErrValidation)", id, err)
		}
	}
}