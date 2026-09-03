//go:build integration

// Package postgres — auth_organization_repo_test.go locks the
// AuthOrganizationRepo adapter contract at the DB level per spec
// R-DB-003 / R-BOOTSTRAP-1 / S-DB-020 / S-DB-021.
//
// Tests are INTEGRATION=1 gated; skipped (not failed) without a
// live Postgres connection.
package postgres

import (
	"context"
	"testing"

	"github.com/cachicamas/backend/database_administrator/src/domain/auth"
)

// TestAuthOrganizationRepo_Create covers the create path: a fresh
// organization MUST have a deterministic slug derived from owner_id.
func TestAuthOrganizationRepo_Create(t *testing.T) {
	db := authIntegrationDB(t)
	resetAuthTablesForRepoTest(t, db)

	userRepo := NewUserRepo(db)
	orgRepo := NewAuthOrganizationRepo(db)

	ctx := context.Background()
	u := auth.NewUser("imm@example.com", "imm-sub-1")
	created, err := userRepo.InsertRegistered(ctx, db, u)
	if err != nil {
		t.Fatalf("InsertRegistered: %v", err)
	}

	org := auth.NewOrganization(created.ID, "Acme Studio")
	got, err := orgRepo.Create(ctx, db, org)
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if got.OwnerID != created.ID {
		t.Errorf("OwnerID = %d, want %d", got.OwnerID, created.ID)
	}
	if got.Name != "Acme Studio" {
		t.Errorf("Name = %q, want %q", got.Name, "Acme Studio")
	}
	if got.Slug == "" {
		t.Errorf("Slug = empty, want non-empty (deterministic from owner_id)")
	}
}

// TestAuthOrganizationRepo_FindByOwnerID covers the 1:1 lookup.
func TestAuthOrganizationRepo_FindByOwnerID(t *testing.T) {
	db := authIntegrationDB(t)
	resetAuthTablesForRepoTest(t, db)

	userRepo := NewUserRepo(db)
	orgRepo := NewAuthOrganizationRepo(db)

	ctx := context.Background()
	u := auth.NewUser("imm@example.com", "imm-sub-1")
	created, err := userRepo.InsertRegistered(ctx, db, u)
	if err != nil {
		t.Fatalf("InsertRegistered: %v", err)
	}
	org := auth.NewOrganization(created.ID, "Acme Studio")
	createdOrg, err := orgRepo.Create(ctx, db, org)
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	got, err := orgRepo.FindByOwnerID(ctx, db, created.ID)
	if err != nil {
		t.Fatalf("FindByOwnerID: %v", err)
	}
	if got == nil {
		t.Fatal("FindByOwnerID: nil")
	}
	if got.ID != createdOrg.ID {
		t.Errorf("ID = %d, want %d", got.ID, createdOrg.ID)
	}
}

// TestAuthOrganizationRepo_FindByOwnerID_MissReturnsNil covers the
// not-found contract.
func TestAuthOrganizationRepo_FindByOwnerID_MissReturnsNil(t *testing.T) {
	db := authIntegrationDB(t)
	resetAuthTablesForRepoTest(t, db)
	orgRepo := NewAuthOrganizationRepo(db)

	got, err := orgRepo.FindByOwnerID(context.Background(), db, 9999)
	if err != nil {
		t.Fatalf("FindByOwnerID: %v", err)
	}
	if got != nil {
		t.Errorf("FindByOwnerID on miss: got=%v, want nil", got)
	}
}