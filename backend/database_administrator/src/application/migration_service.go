// Package application contains the use cases of the database
// administrator service. This file implements the migration use
// case: it orchestrates the domain.Runner port with OTel + slog
// instrumentation per design §10 + spec R-DBMIG-040.
//
// Hexagonal boundary (design §3):
//
//   - This file imports domain (the port) and the stdlib
//     observability stack (slog, OTel trace). It does NOT import
//     pressly/goose or jackc/pgx.
//   - The migration runner lives in src/migration/ (the adapter).
//   - main.go wires a goose-backed domain.Runner into this service.
package application

import (
	"context"
	"errors"
	"log/slog"
	"time"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
	"go.opentelemetry.io/otel/trace/noop"

	"github.com/cachicamas/backend/database_administrator/src/domain"
)

// migrationUpSpanName is the OTel span name for the migration Up
// operation. Centralised so the application service and any future
// instrumented callers use the same name (so Jaeger queries like
// `service.name = "database_administrator" AND name = "migration.up"`
// work uniformly).
const migrationUpSpanName = "migration.up"

// MigrationService is the use case that applies pending migrations
// via the domain.Runner port. It is the ONLY caller of the port in
// the application layer; main.go is the composition root that
// wires the port to a concrete adapter.
type MigrationService struct {
	runner domain.Runner
	logger *slog.Logger
	tracer trace.Tracer
}

// NewMigrationService constructs a MigrationService. The runner is
// the hexagonal port; production code wires it to a goose-backed
// adapter (see src/migration/runner.go), tests wire it to a fake.
// tracer is the OTel Tracer used to open the `migration.up` span;
// production code wires it via otel.Tracer("database_administrator"),
// tests wire it via an in-memory recorder.
func NewMigrationService(r domain.Runner, logger *slog.Logger, tracer trace.Tracer) *MigrationService {
	if logger == nil {
		logger = slog.Default()
	}
	if tracer == nil {
		tracer = noop.NewTracerProvider().Tracer("application/migration_service")
	}
	return &MigrationService{
		runner: r,
		logger: logger,
		tracer: tracer,
	}
}

// Up applies any pending migrations by delegating to the runner
// port. It opens an OTel span `migration.up`, emits an slog line
// on success or failure, and returns whatever the runner returned
// (applied versions + nil on success, nil + error on failure).
//
// OTel attributes set on the span (per design §10 + spec R-DBMIG-040):
//
//   - db.system               = "postgresql"           (always)
//   - migration.dir           = "sql"                  (always)
//   - migration.duration_ms   = int64 wall-clock ms    (always)
//   - migration.applied_count = int N                  (success only)
//   - migration.error         = err.Error()            (failure only)
//   - migration.error.kind    = pg_advisory_lock_timeout |
//     pg_query_error | pg_connect_error | embed_error (failure only)
func (s *MigrationService) Up(ctx context.Context) ([]domain.Version, error) {
	ctx, span := s.tracer.Start(ctx, migrationUpSpanName,
		trace.WithSpanKind(trace.SpanKindInternal),
		trace.WithAttributes(
			attribute.String("db.system", "postgresql"),
			attribute.String("migration.dir", "sql"),
		),
	)
	defer span.End()

	start := time.Now()
	applied, err := s.runner.Up(ctx)
	durationMs := time.Since(start).Milliseconds()
	span.SetAttributes(attribute.Int64("migration.duration_ms", durationMs))

	if err != nil {
		s.recordFailure(ctx, span, err, durationMs)
		return nil, err
	}

	span.SetAttributes(attribute.Int("migration.applied_count", len(applied)))
	s.logger.InfoContext(ctx, "migration.up applied",
		slog.Int("applied_count", len(applied)),
		slog.Int64("duration_ms", durationMs),
	)
	return applied, nil
}

// Status returns the slice of versions already in the bookkeeping
// table by delegating to the runner port. Status is NOT wrapped in
// an OTel span: the runner's own bookkeeping query is fast, the
// operator-facing UI does not need per-call tracing, and emitting
// a span per Status call would inflate Jaeger with noise.
//
// If you ever need to trace Status (e.g. for debugging a slow
// operator dashboard), add `migration.status` here.
func (s *MigrationService) Status(ctx context.Context) ([]domain.Version, error) {
	return s.runner.Status(ctx)
}

// recordFailure centralizes the failure-path telemetry: span
// status, span attributes, and slog.Error. Centralizing avoids
// drift between the two paths (a real risk as the file evolves).
func (s *MigrationService) recordFailure(ctx context.Context, span trace.Span, err error, durationMs int64) {
	span.RecordError(err)
	span.SetStatus(codes.Error, err.Error())
	span.SetAttributes(
		attribute.String("migration.error", err.Error()),
		attribute.String("migration.error.kind", classifyError(err)),
	)
	s.logger.ErrorContext(ctx, "migration.up failed",
		slog.String("error", err.Error()),
		slog.String("error.kind", classifyError(err)),
		slog.Int64("duration_ms", durationMs),
	)
}

// classifyError maps an error from domain.Runner.Up to one of the
// four canonical migration.error.kind values from design §10. The
// classification is deliberately string-based (we do not import
// pgx or goose here to keep the application package decoupled from
// the adapter layer). Strings that appear in pgx error messages
// (connection refused, syntax error, etc.) are mapped to the
// closest canonical kind.
func classifyError(err error) string {
	if err == nil {
		return ""
	}
	msg := err.Error()
	switch {
	case msgContains(msg, "advisory lock") || msgContains(msg, "lock_timeout"):
		return "pg_advisory_lock_timeout"
	case msgContains(msg, "connect") || msgContains(msg, "dial") || msgContains(msg, "refused") || msgContains(msg, "reset by peer"):
		return "pg_connect_error"
	case msgContains(msg, "syntax") || msgContains(msg, "relation") || msgContains(msg, "column") || msgContains(msg, "permission denied"):
		return "pg_query_error"
	}
	return "embed_error"
}

// msgContains returns true if needle appears anywhere in haystack.
// Case-sensitive; we don't normalize because we want pg error
// fragments (which are case-sensitive) to match.
//
// Mirrors the helper in src/migration/runner.go — duplicated
// rather than exported so the application package stays free of
// migration-package imports.
func msgContains(haystack, needle string) bool {
	if len(needle) == 0 {
		return true
	}
	if len(needle) > len(haystack) {
		return false
	}
	for i := 0; i+len(needle) <= len(haystack); i++ {
		if haystack[i:i+len(needle)] == needle {
			return true
		}
	}
	return false
}

// Compile-time check that MigrationService satisfies any future
// interface the composition root may declare. Currently main.go
// just calls Up / Status directly, but having this anchor makes
// refactoring safer.
var _ interface {
	Up(ctx context.Context) ([]domain.Version, error)
	Status(ctx context.Context) ([]domain.Version, error)
} = (*MigrationService)(nil)

// _ silences the unused-import warning for errors when this file
// stops calling it. (errors.Is is used in the test file, not here,
// but keeping the import ready for future use is intentional.)
var _ = errors.Is