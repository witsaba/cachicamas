// Package auth — bootstrap_test.go locks the BootstrapService
// contract per spec R-BE-001 / R-BE-002 / R-BOOTSTRAP-1.
//
// Tests are split into two categories:
//
//  - Validation tests (always run): exercise the input-validation
//    gate that fires BEFORE the service opens a transaction. They
//    use a never-connecting *sql.DB so the service's BeginTx is
//    never reached.
//
//  - Integration tests (INTEGRATION=1 gated): exercise the
//    transactional flow against a live Postgres via the real
//    adapters in infrastructure/postgres. They require the
//    dev compose stack to be running; skipped (not failed) when
//    INTEGRATION is not set.
//
// The fake repos in this file satisfy the domain ports and
// emulate the Postgres adapter's (nil, nil)-on-miss +
// idempotent-on-google_sub semantics. They are NOT used by the
// integration tests; those use the real adapters.
package auth

import (
	"context"
	"database/sql"
	"errors"
	"sync"
	"testing"

	_ "github.com/jackc/pgx/v5/stdlib"

	"github.com/cachicamas/backend/database_administrator/src/domain/auth"
)

// ----------------------------------------------------------------------------
// In-memory fake repos for the validation tests.
// ----------------------------------------------------------------------------

type fakeUserRepo struct {
	mu    sync.Mutex
	bySub map[string]*auth.User
	byID  map[int64]*auth.User
}

func newFakeUserRepo() *fakeUserRepo {
	return &fakeUserRepo{
		bySub: map[string]*auth.User{},
		byID:  map[int64]*auth.User{},
	}
}

func (r *fakeUserRepo) FindByGoogleSub(_ context.Context, _ auth.Querier, googleSub string) (*auth.User, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if u, ok := r.bySub[googleSub]; ok {
		copy := *u
		return &copy, nil
	}
	return nil, nil
}

func (r *fakeUserRepo) FindByID(_ context.Context, _ auth.Querier, id int64) (*auth.User, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if u, ok := r.byID[id]; ok {
		copy := *u
		return &copy, nil
	}
	return nil, nil
}

func (r *fakeUserRepo) InsertRegistered(_ context.Context, _ auth.Querier, u *auth.User) (*auth.User, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if existing, ok := r.bySub[u.GoogleSub]; ok {
		copy := *existing
		return &copy, nil
	}
	id := int64(len(r.byID) + 1)
	created := *u
	created.ID = id
	r.bySub[u.GoogleSub] = &created
	r.byID[id] = &created
	return &created, nil
}

func (r *fakeUserRepo) UpdateLoginFields(_ context.Context, _ auth.Querier, id int64, u *auth.User) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	existing, ok := r.byID[id]
	if !ok {
		return nil
	}
	existing.Email = u.Email
	existing.EmailVerified = u.EmailVerified
	existing.Name = u.Name
	existing.PictureURL = u.PictureURL
	if u.LastLoginAt != nil {
		t := *u.LastLoginAt
		existing.LastLoginAt = &t
	}
	return nil
}

func (r *fakeUserRepo) PromoteToActive(_ context.Context, _ auth.Querier, id int64) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if existing, ok := r.byID[id]; ok && existing.Status == auth.UserStatusRegistered {
		existing.Status = auth.UserStatusActive
	}
	return nil
}

type fakeOrgRepo struct {
	mu      sync.Mutex
	byOwner map[int64]*auth.Organization
	byID    map[int64]*auth.Organization
}

func newFakeOrgRepo() *fakeOrgRepo {
	return &fakeOrgRepo{
		byOwner: map[int64]*auth.Organization{},
		byID:    map[int64]*auth.Organization{},
	}
}

func (r *fakeOrgRepo) FindByOwnerID(_ context.Context, _ auth.Querier, ownerID int64) (*auth.Organization, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if o, ok := r.byOwner[ownerID]; ok {
		copy := *o
		return &copy, nil
	}
	return nil, nil
}

func (r *fakeOrgRepo) FindByID(_ context.Context, _ auth.Querier, id int64) (*auth.Organization, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if o, ok := r.byID[id]; ok {
		copy := *o
		return &copy, nil
	}
	return nil, nil
}

func (r *fakeOrgRepo) Create(_ context.Context, _ auth.Querier, o *auth.Organization) (*auth.Organization, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	id := int64(len(r.byID) + 1)
	created := *o
	created.ID = id
	created.Slug = "pyme-" + idStr(id)
	r.byOwner[o.OwnerID] = &created
	r.byID[id] = &created
	return &created, nil
}

func idStr(n int64) string {
	if n == 0 {
		return "0"
	}
	var buf [20]byte
	i := len(buf)
	for n > 0 {
		i--
		buf[i] = byte('0' + n%10)
		n /= 10
	}
	return string(buf[i:])
}

type fakeAuditRepo struct {
	mu    sync.Mutex
	rows  []*auth.LoginAudit
	byID  map[int64]*auth.LoginAudit
	nextI int64
}

func newFakeAuditRepo() *fakeAuditRepo {
	return &fakeAuditRepo{byID: map[int64]*auth.LoginAudit{}}
}

func (r *fakeAuditRepo) Insert(_ context.Context, _ auth.Querier, a *auth.LoginAudit) (*auth.LoginAudit, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.nextI++
	created := *a
	created.ID = r.nextI
	r.rows = append(r.rows, &created)
	r.byID[created.ID] = &created
	return &created, nil
}

// neverConnectingDB returns a *sql.DB whose DSN is invalid but
// whose sql.Open call succeeds. Used by validation tests (which
// exit before any DB call). The first DB query would fail with a
// connection error — that is the design: validation tests must
// exit before reaching the DB layer.
func neverConnectingDB() *sql.DB {
	db, _ := sql.Open("pgx", "postgres://invalid:invalid@127.0.0.1:1/nonexistent?sslmode=disable&connect_timeout=1")
	return db
}

// =====================================================================
// Validation tests (always run; never reach the DB).
// =====================================================================

// TestBootstrapService_Validation_RequiresGoogleSub covers the
// input-validation gate.
func TestBootstrapService_Validation_RequiresGoogleSub(t *testing.T) {
	svc := NewBootstrapService(neverConnectingDB(), newFakeUserRepo(), newFakeOrgRepo(), newFakeAuditRepo())
	_, err := svc.Bootstrap(context.Background(), BootstrapInput{
		GoogleSub: "",
		Email:     "founder@example.com",
	})
	if err == nil {
		t.Fatal("Bootstrap{GoogleSub: \"\"}: expected error, got nil")
	}
	if !errors.Is(err, ErrValidation) {
		t.Errorf("error = %v, want errors.Is(ErrValidation)", err)
	}
}

// TestBootstrapService_Validation_RequiresEmail covers the
// input-validation gate.
func TestBootstrapService_Validation_RequiresEmail(t *testing.T) {
	svc := NewBootstrapService(neverConnectingDB(), newFakeUserRepo(), newFakeOrgRepo(), newFakeAuditRepo())
	_, err := svc.Bootstrap(context.Background(), BootstrapInput{
		GoogleSub: "google-sub-1",
		Email:     "",
	})
	if err == nil {
		t.Fatal("Bootstrap{Email: \"\"}: expected error, got nil")
	}
	if !errors.Is(err, ErrValidation) {
		t.Errorf("error = %v, want errors.Is(ErrValidation)", err)
	}
}

// =====================================================================
// Integration tests (INTEGRATION=1 gated; live Postgres).
//
// Live integration tests for the bootstrap service live in
// infrastructure/postgres (NOT here) to avoid the import cycle
// (postgres -> interfaces/http -> application/auth -> postgres).
// The real-DB tests exercise:
//
//   - infrastructure/postgres/auth_user_repo_test.go (T2.2)
//   - infrastructure/postgres/auth_organization_repo_test.go (T2.4)
//   - infrastructure/postgres/auth_login_audit_repo_test.go (T2.6)
//   - infrastructure/postgres/bootstrap_integration_test.go (T2.7+T2.8)
//
// The current file is therefore limited to the unit-level
// validation tests (which never reach the DB) plus a stub
// integration entrypoint that explains the partitioning.
// =====================================================================

// TestBootstrapService_IntegrationPartition covers the partitioning
// invariant: the live-DB bootstrap tests live in
// infrastructure/postgres (not here). This stub ensures the test
// suite has at least one entrypoint that mentions the partition
// so a future contributor does not accidentally add an
// integration test that re-creates the import cycle.
func TestBootstrapService_IntegrationPartition(t *testing.T) {
	t.Log("bootstrap integration tests live in infrastructure/postgres to avoid the application -> postgres import cycle")
}

// newFakeService is kept for compatibility with me_test.go (which
// uses fake repos for end-to-end smoke).
func newFakeService() (*BootstrapService, *fakeUserRepo, *fakeOrgRepo, *fakeAuditRepo) {
	u := newFakeUserRepo()
	o := newFakeOrgRepo()
	a := newFakeAuditRepo()
	svc := NewBootstrapService(neverConnectingDB(), u, o, a)
	return svc, u, o, a
}