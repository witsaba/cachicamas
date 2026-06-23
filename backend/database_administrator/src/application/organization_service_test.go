// Package application_test contains the test suite for the
// organization service. The tests use a hand-rolled fakeRepo so
// they are pure unit tests — no DB, no HTTP, no live OTel
// collector. The OTel assertions are made against an in-memory
// span recorder (sdktrace/tracetest), the same pattern used by
// application/migration_service_test.go.
//
// Strict TDD discipline (per openspec/AGENTS.md and
// sdd-init/cachicamas): this file was written BEFORE
// organization_service.go existed. Running
// `go test ./src/application/...` with no OrganizationService
// type must fail with "undefined: OrganizationService" — that
// failure IS the RED step.
package application_test

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"sync"
	"testing"
	"time"

	"go.opentelemetry.io/otel"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
	"go.opentelemetry.io/otel/trace"

	"github.com/cachicamas/backend/database_administrator/src/application"
	"github.com/cachicamas/backend/database_administrator/src/domain"
)

// ---------------------------------------------------------------------------
// fakeRepo — an in-memory implementation of
// domain.OrganizationRepository used only by this test file. It
// records the calls it received so the test can assert on them.
// It does NOT touch a database.
// ---------------------------------------------------------------------------

type fakeRepo struct {
	mu sync.Mutex

	// insertResult / insertErr are returned by Insert.
	insertResult *domain.Organization
	insertErr    error
	insertCalls  int

	// listResult / listErr are returned by SelectAll.
	listResult []domain.Organization
	listErr    error
	listCalls  int

	// byID maps id -> *Organization for SelectByID. If an id is not
	// present, the adapter returns *domain.NotFoundError.
	byID   map[int64]*domain.Organization
	byErr  error // optional: returned for a particular id lookup
	getCalls int
}

func (f *fakeRepo) Insert(_ context.Context, o *domain.Organization) (*domain.Organization, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.insertCalls++
	if f.insertErr != nil {
		return nil, f.insertErr
	}
	// If the test supplied a canned result, return it verbatim (the
	// test is asserting on what the service does with the result, not
	// on the result itself).
	if f.insertResult != nil {
		return f.insertResult, nil
	}
	// Otherwise, return a copy of o with a synthetic ID. The service
	// is allowed to assume the returned *Organization has a populated
	// ID — that's the contract every Postgres adapter honors.
	out := *o
	out.ID = 42
	out.IsActive = true
	out.CreatedAt = time.Date(2026, 6, 22, 12, 0, 0, 0, time.UTC)
	out.UpdatedAt = out.CreatedAt
	return &out, nil
}

func (f *fakeRepo) SelectAll(_ context.Context) ([]domain.Organization, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.listCalls++
	return f.listResult, f.listErr
}

func (f *fakeRepo) SelectByID(_ context.Context, id int64) (*domain.Organization, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.getCalls++
	if f.byErr != nil {
		return nil, f.byErr
	}
	if o, ok := f.byID[id]; ok {
		out := *o
		return &out, nil
	}
	return nil, &domain.NotFoundError{Resource: "organization"}
}

// newRecordingLogger returns a slog.Logger writing JSON records
// into a buffer so the test can assert log content.
func newRecordingLogger() (*slog.Logger, *syncBuf) {
	buf := &syncBuf{}
	return slog.New(slog.NewJSONHandler(buf, &slog.HandlerOptions{Level: slog.LevelDebug})), buf
}

type syncBuf struct {
	mu  sync.Mutex
	out []byte
}

func (b *syncBuf) Write(p []byte) (int, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.out = append(b.out, p...)
	return len(p), nil
}

func (b *syncBuf) String() string {
	b.mu.Lock()
	defer b.mu.Unlock()
	return string(b.out)
}

// newTestTracer wires an in-memory OTel tracer that records every
// span it creates. The returned recorder lets the test assert that
// OrganizationService.Create / List / Get opened a span with the
// expected attributes.
func newTestTracer() (trace.Tracer, *tracetest.SpanRecorder) {
	sr := tracetest.NewSpanRecorder()
	tp := sdktrace.NewTracerProvider(sdktrace.WithSpanProcessor(sr))
	otel.SetTracerProvider(tp)
	return tp.Tracer("database_administrator/test"), sr
}

// attrKeyValue flattens a span's attributes into a map keyed by
// attribute key with stringified values.
func attrKeyValue(span sdktrace.ReadOnlySpan) map[string]string {
	out := make(map[string]string)
	for _, kv := range span.Attributes() {
		out[string(kv.Key)] = kv.Value.String()
	}
	return out
}

// ---------------------------------------------------------------------------
// Tests
// ---------------------------------------------------------------------------

// TestOrganizationService_Create_OpensSpanWithAttributes covers the
// B-7 happy path: Create opens a span named "organization.create"
// with the locked attributes (http.method, http.route,
// http.status_code, organization.id after a successful insert).
func TestOrganizationService_Create_OpensSpanWithAttributes(t *testing.T) {
	repo := &fakeRepo{
		insertResult: &domain.Organization{
			ID:             7,
			FullName:       "Acme",
			Identification: "acme",
			IsActive:       true,
			CreatedAt:      time.Date(2026, 6, 22, 12, 0, 0, 0, time.UTC),
			UpdatedAt:      time.Date(2026, 6, 22, 12, 0, 0, 0, time.UTC),
		},
	}
	tracer, sr := newTestTracer()
	logger, _ := newRecordingLogger()

	svc := application.NewOrganizationService(repo, logger, tracer)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	out, err := svc.Create(ctx, domain.CreateOrganizationInput{
		FullName:       "Acme",
		Identification: "acme",
	})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if out.ID != 7 {
		t.Errorf("Create returned ID = %d, want 7", out.ID)
	}

	spans := sr.Ended()
	if len(spans) != 1 {
		t.Fatalf("recorded spans = %d, want 1 (got %+v)", len(spans), spans)
	}
	gotSpan := spans[0]
	if gotSpan.Name() != "organization.create" {
		t.Errorf("span name = %q, want %q", gotSpan.Name(), "organization.create")
	}
	attrMap := attrKeyValue(gotSpan)
	for _, want := range []struct{ k, v string }{
		{"http.method", "POST"},
		{"http.route", "/organizations"},
		{"http.status_code", "201"},
		{"organization.id", "7"},
	} {
		if got := attrMap[want.k]; got != want.v {
			t.Errorf("span attr %s = %q, want %q", want.k, got, want.v)
		}
	}
}

// TestOrganizationService_Create_ValidationError_DoesNotCallRepo
// covers B-2: invalid input short-circuits the repo call (no DB
// round-trip, no span) and the returned error is a
// *ValidationError with Code() == "validation".
func TestOrganizationService_Create_ValidationError_DoesNotCallRepo(t *testing.T) {
	repo := &fakeRepo{}
	tracer, sr := newTestTracer()
	logger, _ := newRecordingLogger()

	svc := application.NewOrganizationService(repo, logger, tracer)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_, err := svc.Create(ctx, domain.CreateOrganizationInput{
		// missing full_name
		Identification: "acme",
	})
	if err == nil {
		t.Fatalf("Create: expected error, got nil")
	}
	var verr *domain.ValidationError
	if !errors.As(err, &verr) {
		t.Fatalf("Create returned %T, want *ValidationError", err)
	}
	if verr.Code() != "validation" {
		t.Errorf("Code() = %q, want %q", verr.Code(), "validation")
	}
	if repo.insertCalls != 0 {
		t.Errorf("fakeRepo.insertCalls = %d, want 0 (validation must short-circuit)", repo.insertCalls)
	}
	if len(sr.Ended()) != 0 {
		t.Errorf("validation path must not open a span, got %d", len(sr.Ended()))
	}
}

// TestOrganizationService_Create_DuplicateReturnsConflictError
// covers B-4: when the repo returns *ConflictError (the adapter
// has already translated the unique violation), the service
// propagates the typed error unchanged so the handler can map to
// 409. The service MUST NOT wrap further.
func TestOrganizationService_Create_DuplicateReturnsConflictError(t *testing.T) {
	causeBoom := errors.New("pgx unique violation 23505")
	repo := &fakeRepo{
		insertErr: &domain.ConflictError{Cause: causeBoom},
	}
	tracer, _ := newTestTracer()
	logger, _ := newRecordingLogger()

	svc := application.NewOrganizationService(repo, logger, tracer)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_, err := svc.Create(ctx, domain.CreateOrganizationInput{
		FullName:       "Acme",
		Identification: "acme-dup",
	})
	if err == nil {
		t.Fatalf("Create: expected error, got nil")
	}
	var cerr *domain.ConflictError
	if !errors.As(err, &cerr) {
		t.Fatalf("Create returned %T, want *ConflictError", err)
	}
	if cerr.Code() != "conflict" {
		t.Errorf("Code() = %q, want %q", cerr.Code(), "conflict")
	}
}

// TestOrganizationService_List_EmptyAndNonEmpty covers B-5 + B-5b:
// List opens span "organization.list" with http.method /
// http.route / http.status_code / organization.count attributes.
func TestOrganizationService_List_EmptyAndNonEmpty(t *testing.T) {
	t.Run("empty", func(t *testing.T) {
		repo := &fakeRepo{listResult: []domain.Organization{}}
		tracer, sr := newTestTracer()
		logger, _ := newRecordingLogger()

		svc := application.NewOrganizationService(repo, logger, tracer)
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		got, err := svc.List(ctx)
		if err != nil {
			t.Fatalf("List: %v", err)
		}
		if len(got) != 0 {
			t.Errorf("List returned %d rows, want 0", len(got))
		}
		if repo.listCalls != 1 {
			t.Errorf("fakeRepo.listCalls = %d, want 1", repo.listCalls)
		}

		spans := sr.Ended()
		if len(spans) != 1 {
			t.Fatalf("recorded spans = %d, want 1 (got %+v)", len(spans), spans)
		}
		if spans[0].Name() != "organization.list" {
			t.Errorf("span name = %q, want %q", spans[0].Name(), "organization.list")
		}
		attrMap := attrKeyValue(spans[0])
		for _, want := range []struct{ k, v string }{
			{"http.method", "GET"},
			{"http.route", "/organizations"},
			{"http.status_code", "200"},
			{"organization.count", "0"},
		} {
			if got := attrMap[want.k]; got != want.v {
				t.Errorf("span attr %s = %q, want %q", want.k, got, want.v)
			}
		}
	})

	t.Run("non-empty", func(t *testing.T) {
		repo := &fakeRepo{listResult: []domain.Organization{
			{ID: 1, FullName: "A", Identification: "a"},
			{ID: 2, FullName: "B", Identification: "b"},
		}}
		tracer, sr := newTestTracer()
		logger, _ := newRecordingLogger()

		svc := application.NewOrganizationService(repo, logger, tracer)
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		got, err := svc.List(ctx)
		if err != nil {
			t.Fatalf("List: %v", err)
		}
		if len(got) != 2 {
			t.Errorf("List returned %d rows, want 2", len(got))
		}

		spans := sr.Ended()
		if len(spans) != 1 {
			t.Fatalf("recorded spans = %d, want 1", len(spans))
		}
		attrMap := attrKeyValue(spans[0])
		if got := attrMap["organization.count"]; got != "2" {
			t.Errorf("span attr organization.count = %q, want %q", got, "2")
		}
	})
}

// TestOrganizationService_Get_NotFound covers B-6b: Get opens
// span "organization.get" with http.method / http.route /
// http.status_code / organization.id attributes and propagates
// *NotFoundError unchanged.
func TestOrganizationService_Get_NotFound(t *testing.T) {
	repo := &fakeRepo{
		byID:  map[int64]*domain.Organization{},
		byErr: nil,
	}
	tracer, sr := newTestTracer()
	logger, _ := newRecordingLogger()

	svc := application.NewOrganizationService(repo, logger, tracer)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_, err := svc.Get(ctx, 99)
	if err == nil {
		t.Fatalf("Get: expected error, got nil")
	}
	var nerr *domain.NotFoundError
	if !errors.As(err, &nerr) {
		t.Fatalf("Get returned %T, want *NotFoundError", err)
	}
	if nerr.Code() != "not_found" {
		t.Errorf("Code() = %q, want %q", nerr.Code(), "not_found")
	}

	spans := sr.Ended()
	if len(spans) != 1 {
		t.Fatalf("recorded spans = %d, want 1", len(spans))
	}
	if spans[0].Name() != "organization.get" {
		t.Errorf("span name = %q, want %q", spans[0].Name(), "organization.get")
	}
	attrMap := attrKeyValue(spans[0])
	for _, want := range []struct{ k, v string }{
		{"http.method", "GET"},
		{"http.route", "/organizations/:id"},
		{"http.status_code", "404"},
		{"organization.id", "99"},
	} {
		if got := attrMap[want.k]; got != want.v {
			t.Errorf("span attr %s = %q, want %q", want.k, got, want.v)
		}
	}
}

// TestOrganizationService_Get_Found is a companion to the NotFound
// case: when the row exists the service returns it and the span
// carries organization.id.
func TestOrganizationService_Get_Found(t *testing.T) {
	repo := &fakeRepo{
		byID: map[int64]*domain.Organization{
			1: {ID: 1, FullName: "Acme", Identification: "acme", IsActive: true},
		},
	}
	tracer, _ := newTestTracer()
	logger, _ := newRecordingLogger()

	svc := application.NewOrganizationService(repo, logger, tracer)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	got, err := svc.Get(ctx, 1)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if got.ID != 1 {
		t.Errorf("Get returned ID = %d, want 1", got.ID)
	}
}

// _ silences the unused-import warning for io when nothing in this
// file imports it (kept here as a hint that some helpers may
// grow to read from an io.Reader in the future).
var _ = io.EOF
