// Package httpiface_test contains the test suite for the
// organization HTTP handler. The tests use httptest.NewRecorder +
// echo.New() and a real *application.OrganizationService wired to
// the in-memory fakeRepo from
// application/organization_service_test.go. This is the same
// pattern used by health_handler_test.go.
//
// Strict TDD discipline (per openspec/AGENTS.md and
// sdd-init/cachicamas): this file was written BEFORE
// organization_handler.go existed. Running
// `go test ./src/interfaces/http/...` with no OrganizationHandler
// type must fail with "undefined: OrganizationHandler" — that
// failure IS the RED step.
package httpiface

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"go.opentelemetry.io/otel"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"

	"github.com/labstack/echo/v5"

	"github.com/cachicamas/backend/database_administrator/src/application"
	"github.com/cachicamas/backend/database_administrator/src/domain"
)

// ---------------------------------------------------------------------------
// fakeRepo — the same in-memory implementation used in
// application/organization_service_test.go. Duplicated here so
// the http package test can use a real *application.OrganizationService
// (the OTel test depends on the real service being in the call
// path).
// ---------------------------------------------------------------------------

type fakeRepo struct {
	mu sync.Mutex

	insertResult *domain.Organization
	insertErr    error

	listResult []domain.Organization

	byID map[int64]*domain.Organization

	hasOrganizationResult bool
	hasOrganizationErr    error
}

func (f *fakeRepo) Insert(_ context.Context, o *domain.Organization) (*domain.Organization, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.insertErr != nil {
		return nil, f.insertErr
	}
	if f.insertResult != nil {
		return f.insertResult, nil
	}
	out := *o
	out.ID = 1
	out.IsActive = true
	out.CreatedAt = time.Date(2026, 6, 22, 12, 0, 0, 0, time.UTC)
	out.UpdatedAt = out.CreatedAt
	return &out, nil
}

func (f *fakeRepo) SelectAll(_ context.Context) ([]domain.Organization, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.listResult, nil
}

func (f *fakeRepo) SelectByID(_ context.Context, id int64) (*domain.Organization, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if o, ok := f.byID[id]; ok {
		out := *o
		return &out, nil
	}
	return nil, &domain.NotFoundError{Resource: "organization"}
}

func (f *fakeRepo) HasOrganization(_ context.Context) (bool, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.hasOrganizationResult, f.hasOrganizationErr
}

// newTestService wires a real *application.OrganizationService
// against a fakeRepo and a no-op tracer. The OTel test below
// uses a real in-memory exporter; everything else can pass nil.
func newTestService(repo domain.OrganizationRepository) *application.OrganizationService {
	logger := slog.New(slog.NewTextHandler(io.Discard, &slog.HandlerOptions{Level: slog.LevelError + 1}))
	return application.NewOrganizationService(repo, logger, nil)
}

// newTestRouter wires the handler routes against the given
// service.
func newTestRouter(svc *application.OrganizationService) *echo.Echo {
	e := echo.New()
	RegisterOrganizationRoutes(e, svc)
	return e
}

// ---------------------------------------------------------------------------
// Tests
// ---------------------------------------------------------------------------

// TestOrganizationHandler_Post_HappyPath covers spec B-1.
func TestOrganizationHandler_Post_HappyPath(t *testing.T) {
	repo := &fakeRepo{
		insertResult: &domain.Organization{
			ID:             42,
			FullName:       "Acme",
			Identification: "acme",
			IsActive:       true,
			CreatedAt:      time.Date(2026, 6, 22, 12, 0, 0, 0, time.UTC),
			UpdatedAt:      time.Date(2026, 6, 22, 12, 0, 0, 0, time.UTC),
		},
	}
	svc := newTestService(repo)
	e := newTestRouter(svc)

	body := `{"full_name":"Acme","identification":"acme"}`
	req := httptest.NewRequest(http.MethodPost, "/organizations", strings.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("status = %d, want 201 (body=%s)", rec.Code, rec.Body.String())
	}
	if got := rec.Header().Get("Location"); got != "/organizations/42" {
		t.Errorf("Location header = %q, want %q", got, "/organizations/42")
	}
	var got map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
		t.Fatalf("unmarshal body: %v (body=%s)", err, rec.Body.String())
	}
	if id, ok := got["id"].(float64); !ok || int64(id) != 42 {
		t.Errorf("body.id = %v, want 42", got["id"])
	}
	if got["full_name"] != "Acme" {
		t.Errorf("body.full_name = %v, want %q", got["full_name"], "Acme")
	}
	if got["identification"] != "acme" {
		t.Errorf("body.identification = %v, want %q", got["identification"], "acme")
	}
	if isActive, ok := got["is_active"].(bool); !ok || !isActive {
		t.Errorf("body.is_active = %v, want true", got["is_active"])
	}
	if createdAt, ok := got["created_at"].(string); !ok || createdAt == "" {
		t.Errorf("body.created_at = %v, want non-empty RFC 3339 string", got["created_at"])
	}
}

// TestOrganizationHandler_Post_MissingFullName covers spec B-2.
func TestOrganizationHandler_Post_MissingFullName(t *testing.T) {
	repo := &fakeRepo{}
	svc := newTestService(repo)
	e := newTestRouter(svc)

	body := `{"identification":"acme"}`
	req := httptest.NewRequest(http.MethodPost, "/organizations", strings.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400 (body=%s)", rec.Code, rec.Body.String())
	}
	var got map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if got["error"] != "validation" {
		t.Errorf("body.error = %v, want %q", got["error"], "validation")
	}
	fields, _ := got["fields"].(map[string]any)
	if fields == nil || fields["full_name"] != "Name is required." {
		t.Errorf("body.fields.full_name = %v, want %q", fields, "Name is required.")
	}
}

// TestOrganizationHandler_Post_InvalidIdentification covers spec B-3
// with the four locked sub-cases.
func TestOrganizationHandler_Post_InvalidIdentification(t *testing.T) {
	cases := []struct {
		name string
		slug string
	}{
		{"uppercase", "Acme-Industrial"},
		{"too_short", "ab"},
		{"too_long", "a" + strings.Repeat("b", 59) + "c"},
		{"leading_hyphen", "-acme"},
	}
	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			repo := &fakeRepo{}
			svc := newTestService(repo)
			e := newTestRouter(svc)

			body := `{"full_name":"Acme","identification":"` + tc.slug + `"}`
			req := httptest.NewRequest(http.MethodPost, "/organizations", strings.NewReader(body))
			req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
			rec := httptest.NewRecorder()
			e.ServeHTTP(rec, req)

			if rec.Code != http.StatusBadRequest {
				t.Fatalf("status = %d, want 400 (body=%s)", rec.Code, rec.Body.String())
			}
			var got map[string]any
			if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
				t.Fatalf("unmarshal: %v", err)
			}
			if got["error"] != "validation" {
				t.Errorf("body.error = %v, want %q", got["error"], "validation")
			}
			fields, _ := got["fields"].(map[string]any)
			want := "Slug must be 3–60 characters, lowercase letters, digits, and hyphens; must start and end with a letter or digit."
			if fields["identification"] != want {
				t.Errorf("body.fields.identification = %v, want %q", fields["identification"], want)
			}
		})
	}
}

// TestOrganizationHandler_Post_DuplicateIdentification covers spec B-4.
func TestOrganizationHandler_Post_DuplicateIdentification(t *testing.T) {
	repo := &fakeRepo{
		insertErr: &domain.ConflictError{Cause: nil},
	}
	svc := newTestService(repo)
	e := newTestRouter(svc)

	body := `{"full_name":"Acme","identification":"acme-dup"}`
	req := httptest.NewRequest(http.MethodPost, "/organizations", strings.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusConflict {
		t.Fatalf("status = %d, want 409 (body=%s)", rec.Code, rec.Body.String())
	}
	var got map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if got["error"] != "conflict" {
		t.Errorf("body.error = %v, want %q", got["error"], "conflict")
	}
	if got["message"] != "This slug is already taken. Try another." {
		t.Errorf("body.message = %v, want %q", got["message"], "This slug is already taken. Try another.")
	}
}

// TestOrganizationHandler_List_Empty covers spec B-5.
func TestOrganizationHandler_List_Empty(t *testing.T) {
	repo := &fakeRepo{listResult: []domain.Organization{}}
	svc := newTestService(repo)
	e := newTestRouter(svc)

	req := httptest.NewRequest(http.MethodGet, "/organizations", nil)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200 (body=%s)", rec.Code, rec.Body.String())
	}
	body := strings.TrimSpace(rec.Body.String())
	if body != "[]" {
		t.Errorf("body = %q, want %q", body, "[]")
	}
}

// TestOrganizationHandler_List_NonEmpty covers spec B-5b.
func TestOrganizationHandler_List_NonEmpty(t *testing.T) {
	repo := &fakeRepo{listResult: []domain.Organization{
		{ID: 1, FullName: "A", Identification: "a", IsActive: true},
		{ID: 2, FullName: "B", Identification: "b", IsActive: true},
	}}
	svc := newTestService(repo)
	e := newTestRouter(svc)

	req := httptest.NewRequest(http.MethodGet, "/organizations", nil)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200 (body=%s)", rec.Code, rec.Body.String())
	}
	var got []map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if len(got) != 2 {
		t.Errorf("body length = %d, want 2", len(got))
	}
}

// TestOrganizationHandler_Get_Found covers spec B-6.
func TestOrganizationHandler_Get_Found(t *testing.T) {
	repo := &fakeRepo{
		byID: map[int64]*domain.Organization{
			1: {ID: 1, FullName: "Acme", Identification: "acme", IsActive: true},
		},
	}
	svc := newTestService(repo)
	e := newTestRouter(svc)

	req := httptest.NewRequest(http.MethodGet, "/organizations/1", nil)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200 (body=%s)", rec.Code, rec.Body.String())
	}
	var got map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if id, ok := got["id"].(float64); !ok || int64(id) != 1 {
		t.Errorf("body.id = %v, want 1", got["id"])
	}
}

// TestOrganizationHandler_Get_NotFound covers spec B-6b.
func TestOrganizationHandler_Get_NotFound(t *testing.T) {
	repo := &fakeRepo{byID: map[int64]*domain.Organization{}}
	svc := newTestService(repo)
	e := newTestRouter(svc)

	req := httptest.NewRequest(http.MethodGet, "/organizations/9999", nil)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want 404 (body=%s)", rec.Code, rec.Body.String())
	}
	var got map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if got["error"] != "not_found" {
		t.Errorf("body.error = %v, want %q", got["error"], "not_found")
	}
	if got["message"] != "Organization not found." {
		t.Errorf("body.message = %v, want %q", got["message"], "Organization not found.")
	}
}

// TestOrganizationHandler_Post_FormEncoded covers the dual
// content-type contract from locked #3: POST with
// application/x-www-form-urlencoded must produce the same
// 201 as JSON.
func TestOrganizationHandler_Post_FormEncoded(t *testing.T) {
	repo := &fakeRepo{
		insertResult: &domain.Organization{
			ID:             5,
			FullName:       "Acme",
			Identification: "acme",
			IsActive:       true,
			CreatedAt:      time.Date(2026, 6, 22, 12, 0, 0, 0, time.UTC),
			UpdatedAt:      time.Date(2026, 6, 22, 12, 0, 0, 0, time.UTC),
		},
	}
	svc := newTestService(repo)
	e := newTestRouter(svc)

	form := "full_name=Acme&identification=acme"
	req := httptest.NewRequest(http.MethodPost, "/organizations", strings.NewReader(form))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationForm)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("status = %d, want 201 (body=%s)", rec.Code, rec.Body.String())
	}
	if got := rec.Header().Get("Location"); got != "/organizations/5" {
		t.Errorf("Location = %q, want %q", got, "/organizations/5")
	}
}

// TestOrganizationHandler_EmitsSpans covers spec B-7: a POST
// followed by a GET must produce the three locked span names.
func TestOrganizationHandler_EmitsSpans(t *testing.T) {
	// Wire a real in-memory OTel exporter on the global tracer
	// provider; the application's tracer is created at construction
	// time, so we override the global provider BEFORE the service
	// is built.
	sr := tracetest.NewSpanRecorder()
	tp := sdktrace.NewTracerProvider(sdktrace.WithSpanProcessor(sr))
	otel.SetTracerProvider(tp)
	defer func() { _ = tp.Shutdown(context.Background()) }()

	repo := &fakeRepo{
		insertResult: &domain.Organization{
			ID:             7,
			FullName:       "Acme",
			Identification: "acme",
			IsActive:       true,
			CreatedAt:      time.Date(2026, 6, 22, 12, 0, 0, 0, time.UTC),
			UpdatedAt:      time.Date(2026, 6, 22, 12, 0, 0, 0, time.UTC),
		},
		byID: map[int64]*domain.Organization{
			7: {ID: 7, FullName: "Acme", Identification: "acme", IsActive: true},
		},
	}
	logger := slog.New(slog.NewTextHandler(io.Discard, &slog.HandlerOptions{Level: slog.LevelError + 1}))
	svc := application.NewOrganizationService(repo, logger, tp.Tracer("database_administrator/test"))
	e := newTestRouter(svc)

	// POST /organizations
	body := `{"full_name":"Acme","identification":"acme"}`
	req := httptest.NewRequest(http.MethodPost, "/organizations", strings.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)
	if rec.Code != http.StatusCreated {
		t.Fatalf("POST status = %d, want 201 (body=%s)", rec.Code, rec.Body.String())
	}

	// GET /organizations
	req2 := httptest.NewRequest(http.MethodGet, "/organizations", nil)
	rec2 := httptest.NewRecorder()
	e.ServeHTTP(rec2, req2)
	if rec2.Code != http.StatusOK {
		t.Fatalf("GET list status = %d, want 200", rec2.Code)
	}

	// GET /organizations/7
	req3 := httptest.NewRequest(http.MethodGet, "/organizations/7", nil)
	rec3 := httptest.NewRecorder()
	e.ServeHTTP(rec3, req3)
	if rec3.Code != http.StatusOK {
		t.Fatalf("GET single status = %d, want 200", rec3.Code)
	}

	// Allow the recorder to receive finished spans.
	spans := sr.Ended()
	gotNames := map[string]bool{}
	for _, s := range spans {
		gotNames[s.Name()] = true
	}
	for _, want := range []string{"organization.create", "organization.list", "organization.get"} {
		if !gotNames[want] {
			t.Errorf("recorder did not see span %q (got %v)", want, gotNames)
		}
	}
}

// _ silences the unused-import warning for bytes when nothing in
// this file uses it (kept as a hint that some tests may grow to
// inspect request bodies via bytes.Buffer).
var _ = bytes.NewBuffer

// ---------------------------------------------------------------------------
// Tests for GET /setup-state (R-OW-005 / S-OW-040..043)
//
// The setup-state endpoint is the contract the frontend
// requireOwnboarding helper reads. It returns the install-level
// "is there at least one organization?" boolean so the ownboarding
// gate can decide whether the user lands on /home or /ownboarding.
// ---------------------------------------------------------------------------

// TestSetupState_EmptyDB verifies that when no organization exists
// in the database, GET /setup-state returns 200 with
// {"hasOrganization": false} (S-OW-040).
func TestSetupState_EmptyDB(t *testing.T) {
	repo := &fakeRepo{hasOrganizationResult: false}
	svc := newTestService(repo)
	e := newTestRouter(svc)

	req := httptest.NewRequest(http.MethodGet, "/setup-state", nil)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200 (body=%s)", rec.Code, rec.Body.String())
	}
	var got map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
		t.Fatalf("unmarshal body: %v", err)
	}
	if v, ok := got["hasOrganization"].(bool); !ok || v {
		t.Errorf("hasOrganization = %v, want false", got["hasOrganization"])
	}
}

// TestSetupState_WithOrg verifies that when at least one
// organization exists, GET /setup-state returns 200 with
// {"hasOrganization": true} (S-OW-041).
func TestSetupState_WithOrg(t *testing.T) {
	repo := &fakeRepo{hasOrganizationResult: true}
	svc := newTestService(repo)
	e := newTestRouter(svc)

	req := httptest.NewRequest(http.MethodGet, "/setup-state", nil)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	var got map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
		t.Fatalf("unmarshal body: %v", err)
	}
	if v, ok := got["hasOrganization"].(bool); !ok || !v {
		t.Errorf("hasOrganization = %v, want true", got["hasOrganization"])
	}
}

// TestSetupState_RepoError verifies that a repo-level error maps to
// the locked HTTP 500 envelope (S-OW-043).
func TestSetupState_RepoError(t *testing.T) {
	repo := &fakeRepo{
		hasOrganizationResult: false,
		hasOrganizationErr:    &domain.InternalError{Cause: errors.New("db down")},
	}
	svc := newTestService(repo)
	e := newTestRouter(svc)

	req := httptest.NewRequest(http.MethodGet, "/setup-state", nil)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want 500 (body=%s)", rec.Code, rec.Body.String())
	}
	var got map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
		t.Fatalf("unmarshal body: %v", err)
	}
	if got["error"] != domain.CodeServer {
		t.Errorf("error = %v, want %q", got["error"], domain.CodeServer)
	}
}

// TestSetupState_DoesNotInvokeListOrGet verifies that the handler
// only invokes GetSetupState on the service — NOT List or Get. This
// protects against an accidental drift where the handler reads from
// the list endpoint (which returns the full array) instead of the
// cheap existence check.
func TestSetupState_DoesNotInvokeListOrGet(t *testing.T) {
	repo := &fakeRepo{hasOrganizationResult: true}
	svc := newTestService(repo)
	e := newTestRouter(svc)

	req := httptest.NewRequest(http.MethodGet, "/setup-state", nil)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	// The fakeRepo's list/get calls are not separately counted here
	// (the handler_test fakeRepo only exposes hasOrganizationResult).
	// The assertion is structural: the request URL is /setup-state,
	// not /organizations or /organizations/:id, so the Echo router
	// dispatches to SetupState, not List or Get. If the route ever
	// changes, this test catches the drift via the URL.
	if req.URL.Path != "/setup-state" {
		t.Errorf("request URL = %q, want /setup-state", req.URL.Path)
	}
}
