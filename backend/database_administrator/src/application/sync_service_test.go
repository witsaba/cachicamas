package application

import (
	"context"
	"errors"
	"testing"

	"github.com/cachicamas/backend/database_administrator/src/domain"
)

// fakeSyncJobRepo is a minimal in-memory SyncJobRepository for
// the SyncService unit tests. The behavior on Insert is
// controlled by the `insertErr` field; GetLatestForWorkspace
// returns the most recently inserted job (matching the real
// "ORDER BY id DESC LIMIT 1" semantics).
type fakeSyncJobRepo struct {
	jobs      []*domain.SyncJob
	insertErr error
	nextID    int64
	getErr    error
}

func (f *fakeSyncJobRepo) Insert(ctx context.Context, job *domain.SyncJob) (*domain.SyncJob, error) {
	if f.insertErr != nil {
		return nil, f.insertErr
	}
	f.nextID++
	job.ID = f.nextID
	f.jobs = append(f.jobs, job)
	return job, nil
}

func (f *fakeSyncJobRepo) GetLatestForWorkspace(ctx context.Context, workspaceID int64) (*domain.SyncJob, error) {
	if f.getErr != nil {
		return nil, f.getErr
	}
	// Most recent first.
	for i := len(f.jobs) - 1; i >= 0; i-- {
		if f.jobs[i].WorkspaceID == workspaceID {
			return f.jobs[i], nil
		}
	}
	return nil, nil
}

func (f *fakeSyncJobRepo) Update(ctx context.Context, job *domain.SyncJob) error {
	return nil
}

func TestSyncService_EnqueueSync_FreshInsert(t *testing.T) {
	repo := &fakeSyncJobRepo{}
	svc := NewSyncService(repo, nil, nil, nil)

	id, isFresh, existing, err := svc.EnqueueSync(context.Background(), 7, domain.SyncJobTriggerManual)
	if err != nil {
		t.Fatalf("EnqueueSync: %v", err)
	}
	if !isFresh {
		t.Errorf("isFresh = false, want true")
	}
	if id == 0 {
		t.Errorf("id = 0, want > 0")
	}
	if existing == nil {
		t.Errorf("existing = nil, want the persisted job")
	}
	if existing.Status != domain.SyncJobStatusPending {
		t.Errorf("status = %q, want pending", existing.Status)
	}
}

func TestSyncService_EnqueueSync_SingleFlightHit(t *testing.T) {
	// The first call succeeds; the second call must report
	// the existing job (idempotent enqueue).
	repo := &fakeSyncJobRepo{
		insertErr: nil,
	}
	// On the SECOND Insert, simulate a unique-violation.
	repo.insertErr = nil
	// Pre-seed: first call inserted a job.
	svc := NewSyncService(repo, nil, nil, nil)
	_, _, _, _ = svc.EnqueueSync(context.Background(), 7, domain.SyncJobTriggerManual)
	// Switch the fake to return ConflictError on the second Insert.
	repo.insertErr = &domain.ConflictError{Cause: errors.New("duplicate key")}

	id, isFresh, existing, err := svc.EnqueueSync(context.Background(), 7, domain.SyncJobTriggerManual)
	if err != nil {
		t.Fatalf("EnqueueSync on single-flight hit: %v", err)
	}
	if isFresh {
		t.Errorf("isFresh = true, want false (single-flight hit)")
	}
	if id == 0 {
		t.Errorf("id = 0, want the existing job's id")
	}
	if existing == nil {
		t.Errorf("existing = nil, want the existing job")
	}
}

func TestSyncService_EnqueueSync_OtherError(t *testing.T) {
	// A non-conflict error on Insert must propagate.
	repo := &fakeSyncJobRepo{
		insertErr: errors.New("network down"),
	}
	svc := NewSyncService(repo, nil, nil, nil)
	_, _, _, err := svc.EnqueueSync(context.Background(), 7, domain.SyncJobTriggerManual)
	if err == nil {
		t.Errorf("EnqueueSync on non-conflict error: got nil error, want propagated")
	}
}

func TestSyncService_GetLatestSyncJob_NoJobs(t *testing.T) {
	repo := &fakeSyncJobRepo{}
	svc := NewSyncService(repo, nil, nil, nil)
	job, err := svc.GetLatestSyncJob(context.Background(), 7)
	if err != nil {
		t.Fatalf("GetLatestSyncJob: %v", err)
	}
	if job != nil {
		t.Errorf("job = %+v, want nil (no jobs in repo)", job)
	}
}

func TestSyncService_GetLatestSyncJob_HasJob(t *testing.T) {
	repo := &fakeSyncJobRepo{}
	svc := NewSyncService(repo, nil, nil, nil)
	_, _, _, _ = svc.EnqueueSync(context.Background(), 7, domain.SyncJobTriggerManual)
	job, err := svc.GetLatestSyncJob(context.Background(), 7)
	if err != nil {
		t.Fatalf("GetLatestSyncJob: %v", err)
	}
	if job == nil {
		t.Fatalf("job = nil, want the enqueued job")
	}
	if job.WorkspaceID != 7 {
		t.Errorf("WorkspaceID = %d, want 7", job.WorkspaceID)
	}
}
