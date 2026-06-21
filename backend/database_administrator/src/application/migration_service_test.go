// Package application contains the use cases of the database
// administrator service. This test file exercises the migration
// service against a fake domain.Runner (no live DB required), per
// spec R-DBMIG-030 scenario S-DBMIG-031.
//
// Strict TDD discipline (per openspec/AGENTS.md + sdd-init/cachicamas):
// this file was written BEFORE migration_service.go existed; running
// `go test ./src/application/...` against this file with no
// migration_service.go must fail with "undefined: NewMigrationService"
// — that failure IS the RED step.
package application

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

	"github.com/cachicamas/backend/database_administrator/src/domain"
)

// ---------------------------------------------------------------------------
// fakeRunner — an in-memory implementation of domain.Runner used
// only by this test file. It records the calls it received so the
// test can assert on them. It does NOT touch a database.
// ---------------------------------------------------------------------------

type fakeRunner struct {
	mu sync.Mutex

	// upResult / upErr are returned by Up.
	upResult []domain.Version
	upErr    error
	upCalls  int

	// statusResult / statusErr are returned by Status.
	statusResult []domain.Version
	statusErr    error
	statusCalls  int
}

func (f *fakeRunner) Up(_ context.Context) ([]domain.Version, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.upCalls++
	return f.upResult, f.upErr
}

func (f *fakeRunner) Status(_ context.Context) ([]domain.Version, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.statusCalls++
	return f.statusResult, f.statusErr
}

// recordingLogger returns a slog.Logger writing JSON records into a
// buffer so the test can assert log content. It also feeds a
// discard stderr writer so test output is not polluted.
type recordingLogger struct {
	mu  sync.Mutex
	buf *syncBuf
}

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

// ---------------------------------------------------------------------------
// Test helpers
// ---------------------------------------------------------------------------

// newTestTracer wires an in-memory OTel tracer that records every
// span it creates. The returned recorder lets the test assert that
// MigrationService.Up opened a span named "migration.up" with the
// expected attributes.
func newTestTracer() (trace.Tracer, *tracetest.SpanRecorder) {
	sr := tracetest.NewSpanRecorder()
	tp := sdktrace.NewTracerProvider(sdktrace.WithSpanProcessor(sr))
	otel.SetTracerProvider(tp)
	return tp.Tracer("database_administrator/test"), sr
}

// ---------------------------------------------------------------------------
// Tests — RED FIRST, GREEN once migration_service.go lands.
// ---------------------------------------------------------------------------

// TestMigrationService_Up_HappyPath covers the success path:
// the service calls the runner, wraps the call in an OTel span,
// and emits an INFO slog line. RED until migration_service.go
// exists.
func TestMigrationService_Up_HappyPath(t *testing.T) {
	wantVersions := []domain.Version{
		{ID: 20260621120000, Description: "hello_world", AppliedAt: time.Unix(1700000000, 0).UTC()},
	}
	fake := &fakeRunner{upResult: wantVersions}

	tracer, sr := newTestTracer()
	logger, buf := newRecordingLogger()

	svc := NewMigrationService(fake, logger, tracer)
	if svc == nil {
		t.Fatalf("NewMigrationService returned nil")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	got, err := svc.Up(ctx)
	if err != nil {
		t.Fatalf("Up: %v", err)
	}
	if len(got) != 1 || got[0].ID != 20260621120000 {
		t.Errorf("Up returned %+v, want one version with ID 20260621120000", got)
	}
	if fake.upCalls != 1 {
		t.Errorf("fake.upCalls = %d, want 1", fake.upCalls)
	}

	// OTel span assertion: one span named "migration.up" with the
	// expected attributes.
	spans := sr.Ended()
	if len(spans) != 1 {
		t.Fatalf("recorded spans = %d, want 1 (got %+v)", len(spans), spans)
	}
	gotSpan := spans[0]
	if gotSpan.Name() != "migration.up" {
		t.Errorf("span name = %q, want %q", gotSpan.Name(), "migration.up")
	}
	// Attributes we expect on success: db.system, migration.dir,
	// migration.applied_count, migration.duration_ms.
	attrMap := attrKeyValue(gotSpan)
	for _, want := range []struct{ k, v string }{
		{"db.system", "postgresql"},
		{"migration.dir", "sql"},
	} {
		if got := attrMap[want.k]; got != want.v {
			t.Errorf("span attr %s = %q, want %q", want.k, got, want.v)
		}
	}
	if got := attrMap["migration.applied_count"]; got != "1" {
		t.Errorf("span attr migration.applied_count = %q, want \"1\"", got)
	}
	if got := attrMap["migration.duration_ms"]; got == "" {
		t.Errorf("span attr migration.duration_ms must be set")
	}

	// Log line assertion: an INFO line containing "applied_count" and
	// "duration_ms".
	logs := buf.String()
	if logs == "" {
		t.Errorf("expected an INFO log line, got nothing")
	}
	if !contains(logs, "applied_count") || !contains(logs, "duration_ms") {
		t.Errorf("log line missing required fields, got: %s", logs)
	}
	if contains(logs, "\"level\":\"ERROR\"") {
		t.Errorf("happy path must not log ERROR, got: %s", logs)
	}
}

// TestMigrationService_Up_ZeroApplied covers the no-op boot path:
// the runner returns an empty slice and a nil error. The service
// must still open a span, log INFO with applied_count=0, and
// return the empty slice unchanged.
func TestMigrationService_Up_ZeroApplied(t *testing.T) {
	fake := &fakeRunner{upResult: nil}
	tracer, sr := newTestTracer()
	logger, _ := newRecordingLogger()

	svc := NewMigrationService(fake, logger, tracer)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	got, err := svc.Up(ctx)
	if err != nil {
		t.Fatalf("Up: %v", err)
	}
	if len(got) != 0 {
		t.Errorf("Up returned %d versions, want 0", len(got))
	}

	spans := sr.Ended()
	if len(spans) != 1 {
		t.Fatalf("recorded spans = %d, want 1", len(spans))
	}
	attrMap := attrKeyValue(spans[0])
	if got := attrMap["migration.applied_count"]; got != "0" {
		t.Errorf("span attr migration.applied_count = %q, want \"0\"", got)
	}
}

// TestMigrationService_Up_Error covers the failure path: the
// runner returns an error; the service must surface it, set the
// span status to Error, attach migration.error / migration.error.kind,
// and log ERROR.
func TestMigrationService_Up_Error(t *testing.T) {
	boom := errors.New("boom from runner")
	fake := &fakeRunner{upErr: boom}

	tracer, sr := newTestTracer()
	logger, buf := newRecordingLogger()

	svc := NewMigrationService(fake, logger, tracer)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	got, err := svc.Up(ctx)
	if !errors.Is(err, boom) {
		t.Errorf("Up err = %v, want %v", err, boom)
	}
	if got != nil {
		t.Errorf("Up returned %+v on error, want nil", got)
	}

	spans := sr.Ended()
	if len(spans) != 1 {
		t.Fatalf("recorded spans = %d, want 1", len(spans))
	}
	attrMap := attrKeyValue(spans[0])
	if got := attrMap["migration.error"]; got == "" {
		t.Errorf("span attr migration.error must be set on failure")
	}
	if got := attrMap["migration.error.kind"]; got == "" {
		t.Errorf("span attr migration.error.kind must be set on failure")
	}

	logs := buf.String()
	if !contains(logs, "\"level\":\"ERROR\"") {
		t.Errorf("expected an ERROR log line on failure, got: %s", logs)
	}
}

// TestMigrationService_Status_Delegates covers spec R-DBMIG-030:
// Status is a thin pass-through. The service must NOT add a span
// (status isn't part of the migration.up observability surface),
// just return whatever the runner gives back.
func TestMigrationService_Status_Delegates(t *testing.T) {
	want := []domain.Version{
		{ID: 1, Description: "one"},
		{ID: 2, Description: "two"},
	}
	fake := &fakeRunner{statusResult: want}

	tracer, sr := newTestTracer()
	logger, _ := newRecordingLogger()

	svc := NewMigrationService(fake, logger, tracer)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	got, err := svc.Status(ctx)
	if err != nil {
		t.Fatalf("Status: %v", err)
	}
	if len(got) != 2 || got[0].ID != 1 || got[1].ID != 2 {
		t.Errorf("Status returned %+v, want two versions", got)
	}
	if fake.statusCalls != 1 {
		t.Errorf("fake.statusCalls = %d, want 1", fake.statusCalls)
	}
	if len(sr.Ended()) != 0 {
		t.Errorf("Status must not open a span, got %d", len(sr.Ended()))
	}
}

// ---------------------------------------------------------------------------
// helpers
// ---------------------------------------------------------------------------

// attrKeyValue flattens a span's attributes into a map keyed by
// attribute key with stringified values, so the test can assert
// presence and value with simple `if got != want` checks.
func attrKeyValue(span sdktrace.ReadOnlySpan) map[string]string {
	out := make(map[string]string)
	for _, kv := range span.Attributes() {
		out[string(kv.Key)] = kv.Value.Emit()
	}
	return out
}

// contains is a tiny case-sensitive substring search — stdlib
// strings.Contains is fine too, but this keeps the test self-
// contained in style.
func contains(haystack, needle string) bool {
	for i := 0; i+len(needle) <= len(haystack); i++ {
		if haystack[i:i+len(needle)] == needle {
			return true
		}
	}
	return false
}

// _ silences the unused-import warning for io when nothing in this
// file imports it (kept here as a hint that some helpers may
// grow to read from an io.Reader in the future).
var _ = io.EOF