// Package application_test contains the test suite for the
// identity service. The tests use a hand-rolled fakeIdentityRepo so
// they are pure unit tests — no DB, no HTTP, no live OTel
// collector. The OTel assertions are made against an in-memory span
// recorder (sdktrace/tracetest), the same pattern used by
// application/organization_service_test.go.
//
// Strict TDD discipline (per openspec/AGENTS.md and
// sdd-init/cachicamas): this file was written BEFORE
// identity_service.go existed. Running
// `go test ./src/application/...` with no IdentityService type must
// fail with "undefined: IdentityService" — that failure IS the RED
// step.
package application_test

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"sync"
	"testing"

	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"

	"github.com/cachicamas/backend/database_administrator/src/application"
	"github.com/cachicamas/backend/database_administrator/src/domain"
)

// ---------------------------------------------------------------------------
// fakeIdentityRepo — in-memory IdentityRepository used only by
// this test. Records calls so the test can assert.
// ---------------------------------------------------------------------------

type fakeIdentityRepo struct {
	mu sync.Mutex

	// byEmail maps lower-case email -> *Identity. If absent,
	// LookupByEmail returns *domain.IdentityNotFoundError.
	byEmail       map[string]*domain.Identity
	byErr         error // optional: returned for any lookup
	lookupCalls   int
	lastLookupArg string
}

func (f *fakeIdentityRepo) LookupByEmail(_ context.Context, email string) (*domain.Identity, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.lookupCalls++
	f.lastLookupArg = email
	if f.byErr != nil {
		return nil, f.byErr
	}
	got, ok := f.byEmail[email]
	if !ok {
		return nil, &domain.IdentityNotFoundError{Email: email}
	}
	// Defensive copy so the test cannot observe downstream mutation.
	dup := *got
	return &dup, nil
}

// Compile-time guard.
var _ domain.IdentityRepository = (*fakeIdentityRepo)(nil)

// ---------------------------------------------------------------------------
// Helper: build a service with an OTel span recorder so tests can
// assert on span attributes.
// ---------------------------------------------------------------------------

func newServiceWithRecorder(t *testing.T, repo domain.IdentityRepository) (*application.IdentityService, *tracetest.SpanRecorder) {
	t.Helper()
	rec := tracetest.NewSpanRecorder()
	tp := sdktrace.NewTracerProvider(sdktrace.WithSpanProcessor(rec))
	silent := slog.New(slog.NewTextHandler(io.Discard, nil))
	svc := application.NewIdentityService(repo, silent, tp.Tracer("identity-service-test"))
	t.Cleanup(func() { _ = tp.Shutdown(context.Background()) })
	return svc, rec
}

// ---------------------------------------------------------------------------
// Tests
// ---------------------------------------------------------------------------

// TestIdentityService_LookupByEmail_Hit covers the happy path:
// the repo returns an identity; the service returns it unchanged.
// The OTel span `identity.lookup` is opened and closed; the
// result+error path run inside it.
func TestIdentityService_LookupByEmail_Hit(t *testing.T) {
	want := &domain.Identity{
		ID:                42,
		Email:             "braejan@example.com",
		Name:              "braejan",
		ImageURL:          "https://example.com/avatar.png",
		Provider:          "github",
		ProviderAccountID: "12345",
	}
	repo := &fakeIdentityRepo{
		byEmail: map[string]*domain.Identity{
			"braejan@example.com": want,
		},
	}
	svc, rec := newServiceWithRecorder(t, repo)

	got, err := svc.LookupByEmail(context.Background(), "braejan@example.com")
	if err != nil {
		t.Fatalf("LookupByEmail: unexpected error: %v", err)
	}
	if got.ID != want.ID || got.Email != want.Email || got.Provider != "github" {
		t.Errorf("LookupByEmail returned unexpected identity:\n got  = %+v\n want = %+v", got, want)
	}

	// OTel: one `identity.lookup` span was started and ended.
	spans := rec.Ended()
	if len(spans) != 1 {
		t.Fatalf("expected exactly one ended span; got %d", len(spans))
	}
	if got := spans[0].Name(); got != "identity.lookup" {
		t.Errorf("span name:\n got  = %q\n want = %q", got, "identity.lookup")
	}
}

// TestIdentityService_LookupByEmail_Miss covers the not-found
// path: the service returns *domain.IdentityNotFoundError so the
// HTTP handler can map it to a 404 envelope via errors.As.
func TestIdentityService_LookupByEmail_Miss(t *testing.T) {
	repo := &fakeIdentityRepo{byEmail: map[string]*domain.Identity{}}
	svc, _ := newServiceWithRecorder(t, repo)

	_, err := svc.LookupByEmail(context.Background(), "ghost@example.com")
	if err == nil {
		t.Fatalf("expected not-found error; got nil")
	}
	var target *domain.IdentityNotFoundError
	if !errors.As(err, &target) {
		t.Fatalf("error chain does not carry *domain.IdentityNotFoundError: %v", err)
	}
	if target.Email != "ghost@example.com" {
		t.Errorf("expected error.Email = ghost@example.com; got %q", target.Email)
	}
}

// TestIdentityService_LookupByEmail_RepoError covers an arbitrary
// repo error (e.g., DB unreachable): the service must propagate it
// unchanged so the HTTP handler can wrap it in a 5xx envelope.
func TestIdentityService_LookupByEmail_RepoError(t *testing.T) {
	boom := errors.New("connection refused")
	repo := &fakeIdentityRepo{byErr: boom}
	svc, _ := newServiceWithRecorder(t, repo)

	_, err := svc.LookupByEmail(context.Background(), "any@example.com")
	if !errors.Is(err, boom) {
		t.Fatalf("expected error to wrap boom; got %v", err)
	}
	if repo.lookupCalls != 1 {
		t.Errorf("expected exactly one LookupByEmail call; got %d", repo.lookupCalls)
	}
}
