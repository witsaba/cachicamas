// Package httpiface sync_handler_test.go — strict TDD coverage for
// the 2026-07-08-workspace-sync-clone PR-3b sync HTTP handlers.
// Uses a tiny fakeSyncEnqueuer (implements SyncEnqueuer) and a
// fake dispatcher. Echo v5 middleware seeds the identity.
package httpiface_test

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/labstack/echo/v5"

	"github.com/cachicamas/backend/database_administrator/src/domain"
	httpiface "github.com/cachicamas/backend/database_administrator/src/interfaces/http"
)

// fakeSyncEnqueuer is a minimal in-memory SyncEnqueuer.
type fakeSyncEnqueuer struct {
	mu                      sync.Mutex
	enqueueJobID            int64
	enqueueFresh            bool
	enqueueExisting         *domain.SyncJob
	enqueueErr              error
	getLatestExisting       *domain.SyncJob
	getLatestErr            error
	enqueueCallsWorkspaceID []int64
}

func (f *fakeSyncEnqueuer) EnqueueSync(_ context.Context, workspaceID int64, _ string) (int64, bool, *domain.SyncJob, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.enqueueCallsWorkspaceID = append(f.enqueueCallsWorkspaceID, workspaceID)
	return f.enqueueJobID, f.enqueueFresh, f.enqueueExisting, f.enqueueErr
}

func (f *fakeSyncEnqueuer) GetLatestSyncJob(_ context.Context, _ int64) (*domain.SyncJob, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.getLatestExisting, f.getLatestErr
}

type fakeDispatcher struct {
	mu    sync.Mutex
	calls int
}

func (f *fakeDispatcher) StartSync(_ context.Context, _, _ int64, _, _, _, _ string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.calls++
	return nil
}

// seedIdentityMW seeds c.Set with a valid *Identity so the
// handler's auth check passes.
func seedIdentityMW() echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c *echo.Context) error {
			c.Set(httpiface.IdentityContextKey, &domain.Identity{ID: 1, Email: "x@y.z"})
			return next(c)
		}
	}
}

// newSyncTestHandler wires SyncHandler with a fake enqueuer. The
// returned *fakeSyncEnqueuer lets the test inspect calls.
func newSyncTestHandler(t *testing.T, enq *fakeSyncEnqueuer, disp *fakeDispatcher) (*httpiface.SyncHandler, *fakeSyncEnqueuer, *fakeDispatcher) {
	t.Helper()
	h := httpiface.NewSyncHandler(enq, disp, nil)
	return h, enq, disp
}

func TestSyncHandler_Post_FreshEnqueue_202(t *testing.T) {
	createdAt := time.Date(2026, 7, 8, 12, 0, 0, 0, time.UTC)
	job := &domain.SyncJob{
		ID:          42,
		WorkspaceID: 7,
		Status:      domain.SyncJobStatusPending,
		TriggeredBy: domain.SyncJobTriggerManual,
		Attempts:    0,
		CreatedAt:   createdAt,
	}
	enq := &fakeSyncEnqueuer{
		enqueueJobID:    42,
		enqueueFresh:    true,
		enqueueExisting: job,
	}
	disp := &fakeDispatcher{}
	h, _, dispObs := newSyncTestHandler(t, enq, disp)

	e := echo.New()
	g := e.Group("", seedIdentityMW())
	httpiface.RegisterSyncRoutes(g, h)

	req := httptest.NewRequest(http.MethodPost, "/workspaces/7/sync", nil)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusAccepted {
		t.Errorf("status = %d, want 202; body=%s", rec.Code, rec.Body.String())
	}
	var body map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("body is not valid JSON: %v", err)
	}
	if body["job_id"].(float64) != 42 {
		t.Errorf("job_id = %v, want 42", body["job_id"])
	}
	if body["status"] != domain.SyncJobStatusPending {
		t.Errorf("status = %v, want pending", body["status"])
	}
	if dispObs.calls != 1 {
		t.Errorf("dispatcher calls = %d, want 1", dispObs.calls)
	}
}

func TestSyncHandler_Post_SingleFlightHit_200(t *testing.T) {
	createdAt := time.Date(2026, 7, 8, 12, 0, 0, 0, time.UTC)
	job := &domain.SyncJob{
		ID:          42,
		WorkspaceID: 7,
		Status:      domain.SyncJobStatusRunning,
		TriggeredBy: domain.SyncJobTriggerManual,
		Attempts:    0,
		CreatedAt:   createdAt,
	}
	enq := &fakeSyncEnqueuer{
		enqueueJobID:    42,
		enqueueFresh:    false, // single-flight hit
		enqueueExisting: job,
	}
	disp := &fakeDispatcher{}
	h, _, dispObs := newSyncTestHandler(t, enq, disp)

	e := echo.New()
	g := e.Group("", seedIdentityMW())
	httpiface.RegisterSyncRoutes(g, h)

	req := httptest.NewRequest(http.MethodPost, "/workspaces/7/sync", nil)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("status = %d, want 200 (single-flight hit)", rec.Code)
	}
	var body map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("body: %v", err)
	}
	if body["status"] != domain.SyncJobStatusRunning {
		t.Errorf("status = %v, want running", body["status"])
	}
	if dispObs.calls != 0 {
		t.Errorf("dispatcher calls = %d, want 0 (no dispatch on single-flight hit)", dispObs.calls)
	}
}

func TestSyncHandler_Post_InvalidID_400(t *testing.T) {
	enq := &fakeSyncEnqueuer{}
	disp := &fakeDispatcher{}
	h, _, _ := newSyncTestHandler(t, enq, disp)

	e := echo.New()
	g := e.Group("", seedIdentityMW())
	httpiface.RegisterSyncRoutes(g, h)

	req := httptest.NewRequest(http.MethodPost, "/workspaces/notanint/sync", nil)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want 400", rec.Code)
	}
	var body map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("body: %v", err)
	}
	if body["error"] != "validation" {
		t.Errorf("error = %v, want validation", body["error"])
	}
}

func TestSyncHandler_Post_ServiceError_500(t *testing.T) {
	enq := &fakeSyncEnqueuer{
		enqueueErr: errors.New("boom"),
	}
	disp := &fakeDispatcher{}
	h, _, _ := newSyncTestHandler(t, enq, disp)

	e := echo.New()
	g := e.Group("", seedIdentityMW())
	httpiface.RegisterSyncRoutes(g, h)

	req := httptest.NewRequest(http.MethodPost, "/workspaces/7/sync", nil)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)
	if rec.Code != http.StatusInternalServerError {
		t.Errorf("status = %d, want 500", rec.Code)
	}
}

func TestSyncHandler_Get_NoJobs_404(t *testing.T) {
	enq := &fakeSyncEnqueuer{
		getLatestExisting: nil, // no jobs
	}
	disp := &fakeDispatcher{}
	h, _, _ := newSyncTestHandler(t, enq, disp)

	e := echo.New()
	g := e.Group("", seedIdentityMW())
	httpiface.RegisterSyncRoutes(g, h)

	req := httptest.NewRequest(http.MethodGet, "/workspaces/7/sync", nil)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)
	if rec.Code != http.StatusNotFound {
		t.Errorf("status = %d, want 404", rec.Code)
	}
	var body map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("body: %v", err)
	}
	if body["error"] != "workspace_not_synced_yet" {
		t.Errorf("error = %v, want workspace_not_synced_yet", body["error"])
	}
}

func TestSyncHandler_Get_HasJob_200(t *testing.T) {
	enq := &fakeSyncEnqueuer{
		getLatestExisting: &domain.SyncJob{
			ID:          42,
			WorkspaceID: 7,
			Status:      domain.SyncJobStatusDone,
			TriggeredBy: domain.SyncJobTriggerAutoOnCreate,
			Attempts:    1,
			CreatedAt:   time.Date(2026, 7, 8, 12, 0, 0, 0, time.UTC),
		},
	}
	disp := &fakeDispatcher{}
	h, _, _ := newSyncTestHandler(t, enq, disp)

	e := echo.New()
	g := e.Group("", seedIdentityMW())
	httpiface.RegisterSyncRoutes(g, h)

	req := httptest.NewRequest(http.MethodGet, "/workspaces/7/sync", nil)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Errorf("status = %d, want 200; body=%s", rec.Code, rec.Body.String())
	}
	var body map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("body: %v", err)
	}
	if body["workspace_id"].(float64) != 7 {
		t.Errorf("workspace_id = %v, want 7", body["workspace_id"])
	}
	if body["status"] != domain.SyncJobStatusDone {
		t.Errorf("status = %v, want done", body["status"])
	}
	if body["triggered_by"] != domain.SyncJobTriggerAutoOnCreate {
		t.Errorf("triggered_by = %v, want auto_on_create", body["triggered_by"])
	}
}

func TestSyncHandler_Get_InvalidID_400(t *testing.T) {
	enq := &fakeSyncEnqueuer{}
	disp := &fakeDispatcher{}
	h, _, _ := newSyncTestHandler(t, enq, disp)

	e := echo.New()
	g := e.Group("", seedIdentityMW())
	httpiface.RegisterSyncRoutes(g, h)

	req := httptest.NewRequest(http.MethodGet, "/workspaces/abc/sync", nil)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want 400", rec.Code)
	}
}