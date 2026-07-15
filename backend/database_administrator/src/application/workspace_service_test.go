// Package application_test — workspace_service_test.go covers the 5
// use cases of WorkspaceService (Create / List / Get / Update /
// Delete) plus its OTel + slog + GitHubAccessor integration. Same
// patterns as organization_service_test.go: hand-rolled fakeRepo +
// fakeGitHubAccessor + in-memory OTel span recorder + recording slog
// handler.
//
// Strict TDD discipline (per openspec/AGENTS.md): this file was
// originally written per the 20-task list in
// sdd/2026-07-06-workspaces/tasks §PR1b-ii.b. Each task pair
// (RED + GREEN) was captured in the apply-progress artifact at
// sdd/2026-07-06-workspaces/apply-progress-pr1b-ii-b.
//
// 2026-07-08-workspaces-simplify: dropped the 9 test cases for
// AddRepository / RemoveRepository / ListRepositories (the
// workspace_repository table no longer exists) plus the 3 fake
// methods that backed them. Renamed PrimaryRepo -> Repository
// in validCreateInput.
package application_test

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"sync"
	"testing"
	"time"

	"go.opentelemetry.io/otel"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
	"go.opentelemetry.io/otel/trace"

	"github.com/cachicamas/backend/database_administrator/src/application"
	"github.com/cachicamas/backend/database_administrator/src/domain"
	"github.com/cachicamas/backend/database_administrator/src/interfaces/http/tokenctx"
)

// ---------------------------------------------------------------------------
// fakeRepo — in-memory WorkspaceRepository. Records every call so tests
// can assert "did the service call X?" and "what did it pass?".
// ---------------------------------------------------------------------------

type wsFakeRepo struct {
	mu sync.Mutex

	// Insert
	insertErr   error
	insertCalls int
	insertArg   *domain.Workspace // captured

	// SelectAllByOrg
	selectAllResult []domain.Workspace
	selectAllErr    error
	selectAllCalls  int
	selectAllLimit  int
	selectAllOrgID  int64

	// SelectByID
	byID     map[int64]*domain.Workspace
	byErr    error
	getCalls int

	// UpdateName
	updateNameErr    error
	updateNameCalls  int
	updateNameID     int64
	updateNameArg    string
	updateNameResult *domain.Workspace

	// SoftDelete
	deleteErr   error
	deleteCalls int
	deleteID    int64

	// MarkSynced (PR-3b)
	markSyncedErr    error
	markSyncedCalls  int
	markSyncedID     int64
	markSyncedCommit string
	markSyncedBranch string
}

func (f *wsFakeRepo) Insert(_ context.Context, w *domain.Workspace) (*domain.Workspace, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.insertCalls++
	f.insertArg = w
	if f.insertErr != nil {
		return nil, f.insertErr
	}
	out := *w
	out.ID = 100
	out.CreatedAt = time.Date(2026, 7, 6, 12, 0, 0, 0, time.UTC)
	out.UpdatedAt = out.CreatedAt
	return &out, nil
}

func (f *wsFakeRepo) SelectAllByOrg(_ context.Context, orgID int64, limit int) ([]domain.Workspace, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.selectAllCalls++
	f.selectAllOrgID = orgID
	f.selectAllLimit = limit
	return f.selectAllResult, f.selectAllErr
}

func (f *wsFakeRepo) SelectByID(_ context.Context, id int64) (*domain.Workspace, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.getCalls++
	if f.byErr != nil {
		return nil, f.byErr
	}
	if w, ok := f.byID[id]; ok {
		out := *w
		return &out, nil
	}
	return nil, &domain.NotFoundError{Resource: "workspace"}
}

func (f *wsFakeRepo) UpdateName(_ context.Context, id int64, name string) (*domain.Workspace, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.updateNameCalls++
	f.updateNameID = id
	f.updateNameArg = name
	if f.updateNameErr != nil {
		return nil, f.updateNameErr
	}
	if f.updateNameResult != nil {
		out := *f.updateNameResult
		return &out, nil
	}
	// Default: update the row that SelectByID would return.
	out := domain.Workspace{ID: id, Name: name, CreatedAt: time.Date(2026, 7, 6, 12, 0, 0, 0, time.UTC), UpdatedAt: time.Now().UTC()}
	return &out, nil
}

func (f *wsFakeRepo) SoftDelete(_ context.Context, id int64) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.deleteCalls++
	f.deleteID = id
	return f.deleteErr
}

func (f *wsFakeRepo) MarkSynced(_ context.Context, id int64, commitSHA, defaultBranch string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.markSyncedCalls++
	f.markSyncedID = id
	f.markSyncedCommit = commitSHA
	f.markSyncedBranch = defaultBranch
	return f.markSyncedErr
}

// ---------------------------------------------------------------------------
// fakeGitHubAccessor — in-memory GitHubAccessor. The default is
// "everything accessible"; tests flip specific IDs to false to exercise
// the not-accessible path.
// ---------------------------------------------------------------------------

type fakeGitHubAccessor struct {
	mu         sync.Mutex
	accessible map[int64]bool
	errForID   map[int64]error // if set, returns this error (not bool) for the id
	calls      int
}

func (g *fakeGitHubAccessor) IsRepoAccessible(_ context.Context, id int64) (bool, error) {
	g.mu.Lock()
	defer g.mu.Unlock()
	g.calls++
	if err, ok := g.errForID[id]; ok {
		return false, err
	}
	if v, ok := g.accessible[id]; ok {
		return v, nil
	}
	return true, nil // default
}

// ---------------------------------------------------------------------------
// Test helpers (mirrors organization_service_test.go)
// ---------------------------------------------------------------------------

func wsNewRecordingLogger() (*slog.Logger, *wsSyncBuf) {
	buf := &wsSyncBuf{}
	return slog.New(slog.NewJSONHandler(buf, &slog.HandlerOptions{Level: slog.LevelDebug})), buf
}

type wsSyncBuf struct {
	mu  sync.Mutex
	out []byte
}

func (b *wsSyncBuf) Write(p []byte) (int, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.out = append(b.out, p...)
	return len(p), nil
}

func (b *wsSyncBuf) String() string {
	b.mu.Lock()
	defer b.mu.Unlock()
	return string(b.out)
}

func wsNewTestTracer() (trace.Tracer, *tracetest.SpanRecorder) {
	// Real signature: trace.Tracer + *tracetest.SpanRecorder. The
	// concrete types are imported at the top of the file.
	sr2 := tracetest.NewSpanRecorder()
	tp := sdktrace.NewTracerProvider(sdktrace.WithSpanProcessor(sr2))
	otel.SetTracerProvider(tp)
	return tp.Tracer("database_administrator/test"), sr2
}

func wsAttrMap(span sdktrace.ReadOnlySpan) map[string]string {
	out := make(map[string]string)
	for _, kv := range span.Attributes() {
		out[string(kv.Key)] = kv.Value.String()
	}
	return out
}

// withToken is a one-liner that injects a fake access_token into the
// test context. Mirrors the production middleware's WithGitHubToken call.
func withToken(ctx context.Context, token string) context.Context {
	return tokenctx.WithGitHubToken(ctx, token)
}

// validCreateInput returns a happy-path CreateWorkspaceInput the tests
// reuse.
func validCreateInput() domain.CreateWorkspaceInput {
	return domain.CreateWorkspaceInput{
		OrganizationID: 1,
		OwnerUserID:    int64Ptr(99),
		Name:           "frontend-app",
		Repository: domain.Repository{
			GitHubID: 12345,
			FullName: "octocat/frontend-app",
			Owner:    "octocat",
			Name:     "frontend-app",
		},
	}
}

func int64Ptr(v int64) *int64 { return &v }

// ---------------------------------------------------------------------------
// T-WS-1BiiB-001..003: Create
// ---------------------------------------------------------------------------

// T-WS-1BiiB-001/002 GREEN: happy path — validates input, verifies
// GitHub accessibility, inserts, opens span with locked attributes.
func TestWorkspaceService_Create_HappyPath(t *testing.T) {
	repo := &wsFakeRepo{}
	gh := &fakeGitHubAccessor{} // default: everything accessible
	tr, sr := wsNewTestTracer()
	logger, _ := wsNewRecordingLogger()

	svc := application.NewWorkspaceService(repo, nil, gh, logger, tr)
	ctx, cancel := context.WithTimeout(withToken(context.Background(), "test-token"), 5*time.Second)
	defer cancel()

	out, err := svc.Create(ctx, validCreateInput())
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if out.ID != 100 {
		t.Errorf("Create returned ID = %d, want 100", out.ID)
	}
	if repo.insertCalls != 1 {
		t.Errorf("repo.Insert calls = %d, want 1", repo.insertCalls)
	}
	if gh.calls != 1 {
		t.Errorf("githubClient.IsRepoAccessible calls = %d, want 1", gh.calls)
	}

	spans := sr.Ended()
	if len(spans) != 1 {
		t.Fatalf("recorded spans = %d, want 1", len(spans))
	}
	if name := spans[0].Name(); name != "workspace.create" {
		t.Errorf("span name = %q, want workspace.create", name)
	}
	attrs := wsAttrMap(spans[0])
	for _, want := range []struct{ k, v string }{
		{"http.method", "POST"},
		{"http.route", "/workspaces"},
		{"http.status_code", "201"},
		{"workspace.id", "100"},
	} {
		if got := attrs[want.k]; got != want.v {
			t.Errorf("span attr %s = %q, want %q", want.k, got, want.v)
		}
	}
}

// T-WS-1BiiB-003a: IsRepoAccessible returns false → ValidationError
// with field primary_repository = MsgRepoNotAccessible.
func TestWorkspaceService_Create_RepoNotAccessible_ReturnsValidationError(t *testing.T) {
	repo := &wsFakeRepo{}
	gh := &fakeGitHubAccessor{accessible: map[int64]bool{12345: false}}
	tr, _ := wsNewTestTracer()
	logger, _ := wsNewRecordingLogger()

	svc := application.NewWorkspaceService(repo, nil, gh, logger, tr)
	ctx, cancel := context.WithTimeout(withToken(context.Background(), "test-token"), 5*time.Second)
	defer cancel()

	_, err := svc.Create(ctx, validCreateInput())
	if err == nil {
		t.Fatalf("Create: expected error, got nil")
	}
	var verr *domain.ValidationError
	if !errors.As(err, &verr) {
		t.Fatalf("Create returned %T, want *ValidationError", err)
	}
	if verr.Fields["repository"] != domain.MsgRepoNotAccessible {
		t.Errorf("Fields[repository] = %q, want %q",
			verr.Fields["repository"], domain.MsgRepoNotAccessible)
	}
	if repo.insertCalls != 0 {
		t.Errorf("repo.Insert calls = %d, want 0 (validation must short-circuit)", repo.insertCalls)
	}
}

// T-WS-1BiiB-003b: IsRepoAccessible errors → wrapped error.
func TestWorkspaceService_Create_GitHubAccessorError_ReturnsWrapped(t *testing.T) {
	repo := &wsFakeRepo{}
	boom := errors.New("github api 503")
	gh := &fakeGitHubAccessor{errForID: map[int64]error{12345: boom}}
	tr, _ := wsNewTestTracer()
	logger, _ := wsNewRecordingLogger()

	svc := application.NewWorkspaceService(repo, nil, gh, logger, tr)
	ctx, cancel := context.WithTimeout(withToken(context.Background(), "test-token"), 5*time.Second)
	defer cancel()

	_, err := svc.Create(ctx, validCreateInput())
	if err == nil {
		t.Fatalf("Create: expected error from GitHub accessor, got nil")
	}
	if !errors.Is(err, boom) {
		t.Errorf("err = %v, want it to wrap the github accessor error", err)
	}
	if !strings.Contains(err.Error(), "verify repo") {
		t.Errorf("err = %q, want it to include 'verify repo' prefix", err.Error())
	}
	if repo.insertCalls != 0 {
		t.Errorf("repo.Insert calls = %d, want 0", repo.insertCalls)
	}
}

// T-WS-1BiiB-003c: no token in context → GitHubNotConnectedError,
// no span, no GitHub call, no repo call.
func TestWorkspaceService_Create_NoToken_ReturnsGitHubNotConnected(t *testing.T) {
	repo := &wsFakeRepo{}
	gh := &fakeGitHubAccessor{}
	tr, sr := wsNewTestTracer()
	logger, _ := wsNewRecordingLogger()

	svc := application.NewWorkspaceService(repo, nil, gh, logger, tr)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_, err := svc.Create(ctx, validCreateInput())
	if err == nil {
		t.Fatalf("Create: expected GitHubNotConnectedError, got nil")
	}
	var gerr *domain.GitHubNotConnectedError
	if !errors.As(err, &gerr) {
		t.Fatalf("Create returned %T, want *GitHubNotConnectedError", err)
	}
	if gerr.Code() != domain.CodeGitHubNotConnected {
		t.Errorf("Code() = %q, want %q", gerr.Code(), domain.CodeGitHubNotConnected)
	}
	if gh.calls != 0 {
		t.Errorf("githubClient.calls = %d, want 0 (must short-circuit on missing token)", gh.calls)
	}
	if repo.insertCalls != 0 {
		t.Errorf("repo.Insert calls = %d, want 0", repo.insertCalls)
	}
	spanRec := sr
	if len(spanRec.Ended()) != 0 {
		t.Errorf("no-token path must not open a span, got %d", len(spanRec.Ended()))
	}
}

// Create: validation short-circuit — no name, repo, etc.
func TestWorkspaceService_Create_InvalidInput_NoSpanNoRepoCall(t *testing.T) {
	repo := &wsFakeRepo{}
	gh := &fakeGitHubAccessor{}
	tr, sr := wsNewTestTracer()
	logger, _ := wsNewRecordingLogger()

	svc := application.NewWorkspaceService(repo, nil, gh, logger, tr)
	ctx, cancel := context.WithTimeout(withToken(context.Background(), "test-token"), 5*time.Second)
	defer cancel()

	in := validCreateInput()
	in.Name = "" // invalid
	_, err := svc.Create(ctx, in)
	if err == nil {
		t.Fatalf("Create: expected validation error, got nil")
	}
	if repo.insertCalls != 0 {
		t.Errorf("repo.Insert calls = %d, want 0 (validation must short-circuit)", repo.insertCalls)
	}
	if gh.calls != 0 {
		t.Errorf("githubClient.calls = %d, want 0 (validation must short-circuit)", gh.calls)
	}
	if len(sr.Ended()) != 0 {
		t.Errorf("validation path must not open a span")
	}
}

// ---------------------------------------------------------------------------
// T-WS-1BiiB-004/005: List
// ---------------------------------------------------------------------------

func TestWorkspaceService_List_HappyPath(t *testing.T) {
	repo := &wsFakeRepo{
		selectAllResult: []domain.Workspace{
			{ID: 1, Name: "alpha"},
			{ID: 2, Name: "beta"},
		},
	}
	tr, sr := wsNewTestTracer()
	logger, _ := wsNewRecordingLogger()

	svc := application.NewWorkspaceService(repo, nil, nil, logger, tr)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	out, err := svc.List(ctx, 1, 100)
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(out) != 2 {
		t.Errorf("len(out) = %d, want 2", len(out))
	}
	if repo.selectAllOrgID != 1 {
		t.Errorf("repo.SelectAllByOrg(orgID) = %d, want 1", repo.selectAllOrgID)
	}
	if repo.selectAllLimit != 100 {
		t.Errorf("repo.SelectAllByOrg(limit) = %d, want 100", repo.selectAllLimit)
	}

	spans := sr.Ended()
	if len(spans) != 1 || spans[0].Name() != "workspace.list" {
		t.Errorf("expected one workspace.list span, got %+v", spans)
	}
	attrs := wsAttrMap(spans[0])
	if attrs["http.route"] != "/workspaces" {
		t.Errorf("http.route = %q, want /workspaces", attrs["http.route"])
	}
	if attrs["workspace.count"] != "2" {
		t.Errorf("workspace.count = %q, want 2", attrs["workspace.count"])
	}
}

// ---------------------------------------------------------------------------
// T-WS-1BiiB-006/007: Get
// ---------------------------------------------------------------------------

func TestWorkspaceService_Get_Found(t *testing.T) {
	repo := &wsFakeRepo{
		byID: map[int64]*domain.Workspace{
			7: {ID: 7, Name: "alpha"},
		},
	}
	tr, _ := wsNewTestTracer()
	logger, _ := wsNewRecordingLogger()

	svc := application.NewWorkspaceService(repo, nil, nil, logger, tr)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	out, err := svc.Get(ctx, 7)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if out.Name != "alpha" {
		t.Errorf("Name = %q, want alpha", out.Name)
	}
}

func TestWorkspaceService_Get_NotFound(t *testing.T) {
	repo := &wsFakeRepo{} // empty byID → NotFoundError
	tr, _ := wsNewTestTracer()
	logger, _ := wsNewRecordingLogger()

	svc := application.NewWorkspaceService(repo, nil, nil, logger, tr)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_, err := svc.Get(ctx, 999)
	if err == nil {
		t.Fatalf("Get: expected NotFoundError, got nil")
	}
	var nerr *domain.NotFoundError
	if !errors.As(err, &nerr) {
		t.Errorf("err = %T, want *NotFoundError", err)
	}
}

// ---------------------------------------------------------------------------
// T-WS-1BiiB-008/009/010: Update (rename only; primary repo dropped)
// ---------------------------------------------------------------------------

func TestWorkspaceService_Update_RenameOnly(t *testing.T) {
	repo := &wsFakeRepo{
		byID: map[int64]*domain.Workspace{
			7: {ID: 7, Name: "old-name"},
		},
		updateNameResult: &domain.Workspace{
			ID:        7,
			Name:      "new-name",
			UpdatedAt: time.Date(2026, 7, 6, 13, 0, 0, 0, time.UTC),
		},
	}
	tr, _ := wsNewTestTracer()
	logger, _ := wsNewRecordingLogger()

	svc := application.NewWorkspaceService(repo, nil, nil, logger, tr)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	newName := "new-name"
	out, err := svc.Update(ctx, 7, domain.UpdateWorkspaceInput{Name: &newName})
	if err != nil {
		t.Fatalf("Update: %v", err)
	}
	if out.Name != "new-name" {
		t.Errorf("Name = %q, want new-name", out.Name)
	}
	if repo.updateNameArg != "new-name" {
		t.Errorf("repo.UpdateName arg = %q, want new-name", repo.updateNameArg)
	}
}

// T-WS-1BiiB-010a: Update with Name=nil is a no-op that returns the
// current workspace via Get.
func TestWorkspaceService_Update_NilName_NoOp(t *testing.T) {
	repo := &wsFakeRepo{
		byID: map[int64]*domain.Workspace{
			7: {ID: 7, Name: "alpha"},
		},
	}
	tr, _ := wsNewTestTracer()
	logger, _ := wsNewRecordingLogger()

	svc := application.NewWorkspaceService(repo, nil, nil, logger, tr)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	out, err := svc.Update(ctx, 7, domain.UpdateWorkspaceInput{Name: nil})
	if err != nil {
		t.Fatalf("Update: %v", err)
	}
	if out.Name != "alpha" {
		t.Errorf("Name = %q, want alpha (no-op should return current)", out.Name)
	}
	if repo.updateNameCalls != 0 {
		t.Errorf("repo.UpdateName calls = %d, want 0 (nil-name is a no-op)", repo.updateNameCalls)
	}
}

// T-WS-1BiiB-010b: Update with duplicate name → ConflictError.
func TestWorkspaceService_Update_DuplicateName_ReturnsConflict(t *testing.T) {
	repo := &wsFakeRepo{
		updateNameErr: &domain.ConflictError{Cause: fmt.Errorf("pgx 23505")},
	}
	tr, _ := wsNewTestTracer()
	logger, _ := wsNewRecordingLogger()

	svc := application.NewWorkspaceService(repo, nil, nil, logger, tr)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	newName := "dup"
	_, err := svc.Update(ctx, 7, domain.UpdateWorkspaceInput{Name: &newName})
	if err == nil {
		t.Fatalf("Update: expected ConflictError, got nil")
	}
	var cerr *domain.ConflictError
	if !errors.As(err, &cerr) {
		t.Errorf("err = %T, want *ConflictError", err)
	}
}

// ---------------------------------------------------------------------------
// T-WS-1BiiB-011/012: Delete
// ---------------------------------------------------------------------------

func TestWorkspaceService_Delete_Success(t *testing.T) {
	repo := &wsFakeRepo{}
	tr, _ := wsNewTestTracer()
	logger, _ := wsNewRecordingLogger()

	svc := application.NewWorkspaceService(repo, nil, nil, logger, tr)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := svc.Delete(ctx, 42); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	if repo.deleteID != 42 {
		t.Errorf("repo.SoftDelete(id) = %d, want 42", repo.deleteID)
	}
	if repo.deleteCalls != 1 {
		t.Errorf("repo.SoftDelete calls = %d, want 1", repo.deleteCalls)
	}
}

func TestWorkspaceService_Delete_NotFound(t *testing.T) {
	repo := &wsFakeRepo{deleteErr: &domain.NotFoundError{Resource: "workspace"}}
	tr, _ := wsNewTestTracer()
	logger, _ := wsNewRecordingLogger()

	svc := application.NewWorkspaceService(repo, nil, nil, logger, tr)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err := svc.Delete(ctx, 999)
	if err == nil {
		t.Fatalf("Delete: expected NotFoundError, got nil")
	}
	var nerr *domain.NotFoundError
	if !errors.As(err, &nerr) {
		t.Errorf("err = %T, want *NotFoundError", err)
	}
}
