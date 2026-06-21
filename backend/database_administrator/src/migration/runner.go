// Package migration wires the embedded SQL migrations to the goose
// runner. This file is the ONLY file (besides postgres/driver.go,
// which imports pgx) that imports github.com/pressly/goose and its
// lock subpackage. Per design §3, the hexagonal boundary is:
//
//   - runner.go (this file) is the ONLY importer of pressly/goose
//     and pressly/goose/v3/lock.
//   - domain.Runner is the port; runner.go satisfies it.
//   - application/migration_service.go calls the port; it does NOT
//     import this package.
//
// The runner wraps every Up call in an OTel span (`migration.up`)
// and a slog.Info line (per design §10 + spec R-DBMIG-040). Status
// is a thin pass-through to goose's bookkeeping query.
package migration

import (
	"context"
	"database/sql"
	"fmt"
	"io/fs"
	"log/slog"
	"time"

	"github.com/pressly/goose/v3"
	"github.com/pressly/goose/v3/lock"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"

	"github.com/cachicamas/backend/database_administrator/src/domain"
)

// migrationsDir is the path passed to goose as the migration
// directory. Because we feed goose a SUB-FS whose root is already
// the migrations folder (see newProvider), "." is the correct value.
// goose v3.27.1's collectMigrationsFS interprets the second arg
// relative to the FS root, so passing "sql" against an FS whose
// root IS the migration folder would look for `sql/sql/` — wrong.
// Keeping this as a const preserves the option to switch back to a
// root-level FS by changing the value here AND in newProvider.
const migrationsDir = "."

// GooseRunner is the goose-backed adapter that satisfies
// domain.Runner. It holds the *sql.DB it dials through and the name
// of the bookkeeping table (overriding goose's default
// `goose_db_version`).
type GooseRunner struct {
	db        *sql.DB
	tableName string
	logger    *slog.Logger
	tracer    trace.Tracer
}

// NewGooseRunner constructs a GooseRunner. It does NOT dial — the
// caller passes in an already-opened *sql.DB (typically produced by
// postgres.Open in the composition root). The constructor is cheap
// and side-effect-free; the first goose.Up call is what actually
// touches the database.
//
// tableName is the bookkeeping-table name (default
// "schema_migrations" — the table provisioned by
// infra/postgres/init/01-init.sql).
func NewGooseRunner(db *sql.DB, tableName string, logger *slog.Logger) *GooseRunner {
	if tableName == "" {
		// Match the locked default from design §4; never silently use
		// goose's `goose_db_version` because that would create a
		// second table the operator has to look after.
		tableName = "schema_migrations"
	}
	if logger == nil {
		// Defensive: never pass a nil logger downstream.
		logger = slog.Default()
	}
	return &GooseRunner{
		db:        db,
		tableName: tableName,
		logger:    logger,
		tracer:    otel.Tracer("database_administrator/migration"),
	}
}

// Up applies every pending migration in lexicographic order and
// returns the slice of versions it applied (empty on a no-op boot).
//
// Order of operations:
//
//  1. Build a fresh goose.Provider configured with the session
//     locker (non-blocking pg_try_advisory_lock per design §3 +
//     spec R-DBMIG-020), the bookkeeping table name, and the
//     embed.FS as the migration source. A fresh Provider per call
//     is cheap (it does not connect) and lets multiple goroutines
//     share a single GooseRunner.
//  2. Open the OTel span `migration.up` (per design §10 + spec
//     R-DBMIG-040) and record start time.
//  3. Call provider.Up (which acquires the advisory lock, applies
//     migrations, releases the lock).
//  4. Emit slog.Info (success) or slog.Error (failure), set span
//     attributes, return.
func (r *GooseRunner) Up(ctx context.Context) ([]domain.Version, error) {
	provider, err := r.newProvider(ctx)
	if err != nil {
		return nil, fmt.Errorf("migration.Up: build provider: %w", err)
	}

	ctx, span := r.tracer.Start(ctx, "migration.up",
		trace.WithSpanKind(trace.SpanKindInternal),
		trace.WithAttributes(
			attribute.String("db.system", "postgresql"),
			attribute.String("migration.dir", "sql"),
			attribute.String("migration.table", r.tableName),
		),
	)
	defer span.End()

	start := time.Now()

	results, applyErr := provider.Up(ctx)
	durationMs := time.Since(start).Milliseconds()
	span.SetAttributes(attribute.Int64("migration.duration_ms", durationMs))

	if applyErr != nil {
		span.RecordError(applyErr)
		span.SetStatus(codes.Error, applyErr.Error())
		span.SetAttributes(
			attribute.String("migration.error", applyErr.Error()),
			attribute.String("migration.error.kind", classifyError(applyErr)),
		)
		r.logger.ErrorContext(ctx, "migration.up failed",
			slog.String("error", applyErr.Error()),
			slog.String("error.kind", classifyError(applyErr)),
			slog.Int64("duration_ms", durationMs),
		)
		return nil, applyErr
	}

	applied := resultsToVersions(results)
	span.SetAttributes(attribute.Int("migration.applied_count", len(applied)))
	r.logger.InfoContext(ctx, "migration.up applied",
		slog.Int("applied_count", len(applied)),
		slog.Int64("duration_ms", durationMs),
	)
	return applied, nil
}

// Status returns the slice of versions currently recorded in the
// bookkeeping table. Used by the application layer for
// `migrationService.Status(ctx)` (operator-facing "what's been
// applied so far?").
func (r *GooseRunner) Status(ctx context.Context) ([]domain.Version, error) {
	rows, err := r.db.QueryContext(ctx,
		fmt.Sprintf(`SELECT version_id, tstamp FROM %s WHERE is_applied = TRUE ORDER BY version_id ASC`,
			quoteIdent(r.tableName)),
	)
	if err != nil {
		return nil, fmt.Errorf("migration.Status: query %s: %w", r.tableName, err)
	}
	defer rows.Close()

	var out []domain.Version
	for rows.Next() {
		var v domain.Version
		if err := rows.Scan(&v.ID, &v.AppliedAt); err != nil {
			return nil, fmt.Errorf("migration.Status: scan: %w", err)
		}
		v.Description = descriptionFromVersion(v.ID)
		out = append(out, v)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("migration.Status: rows: %w", err)
	}
	return out, nil
}

// newProvider constructs a fresh goose.Provider per call. The
// Provider is the lock-aware unit in goose v3.27.1: it owns the
// session locker, the embed.FS, and the bookkeeping table name.
// Constructing it once per call (rather than once per GooseRunner)
// is intentional: it is a value object, it does not connect, and
// it lets the runner be called from concurrent goroutines (e.g.
// parallel integration tests).
//
// goose expects the FS root to be the migrations directory itself
// (see goose/goose_embed_test.go: fs.Sub then goose.SetBaseFS, then
// goose.Up(db, ".")). We use fs.Sub to re-root the embed.FS at
// "sql" so the "." path passed to provider.Up resolves correctly.
func (r *GooseRunner) newProvider(_ context.Context) (*goose.Provider, error) {
	locker, err := lock.NewPostgresSessionLocker()
	if err != nil {
		return nil, fmt.Errorf("lock.NewPostgresSessionLocker: %w", err)
	}

	subFS, err := fs.Sub(MigrationsFS, "sql")
	if err != nil {
		return nil, fmt.Errorf("fs.Sub(MigrationsFS, sql): %w", err)
	}

	return goose.NewProvider(goose.DialectPostgres, r.db, subFS,
		goose.WithSessionLocker(locker),
		goose.WithTableName(r.tableName),
	)
}

// resultsToVersions converts the goose result slice to the
// hexagonal domain type. The domain type is intentionally
// decoupled from goose so application code does not import goose.
func resultsToVersions(results []*goose.MigrationResult) []domain.Version {
	if len(results) == 0 {
		return nil
	}
	out := make([]domain.Version, 0, len(results))
	for _, r := range results {
		if r == nil || r.Source == nil {
			continue
		}
		out = append(out, domain.Version{
			ID:          r.Source.Version,
			Description: deriveDescription(r.Source.Path),
			// goose v3.27.1 does not expose AppliedAt through
			// MigrationResult; the caller can read it from
			// public.schema_migrations via Status() if needed.
			// We set AppliedAt to zero so callers that ignore it
			// do not accidentally treat it as a real timestamp.
			AppliedAt: time.Time{},
		})
	}
	return out
}

// deriveDescription strips the timestamp prefix and `.sql` suffix
// from a migration filename to produce the `Description` field.
// Example: "20260621120000_hello_world.sql" -> "hello_world".
func deriveDescription(path string) string {
	base := path
	if i := lastIndexByte(base, '/'); i >= 0 {
		base = base[i+1:]
	}
	if len(base) > 4 && base[len(base)-4:] == ".sql" {
		base = base[:len(base)-4]
	}
	// The timestamp prefix is always exactly 14 digits followed by '_'.
	if len(base) > 15 && base[14] == '_' {
		return base[15:]
	}
	return base
}

// descriptionFromVersion is the inverse — given a version ID, look
// up the description in the embed.FS. This is what Status uses to
// enrich a row read from the database with a human-readable label.
// Best-effort: returns "" if the embed.FS doesn't contain the file
// (e.g. someone ran the runner against an older binary that shipped
// a different set of migrations).
func descriptionFromVersion(versionID int64) string {
	name := fmt.Sprintf("%014d", versionID)
	entries, err := MigrationsFS.ReadDir("sql")
	if err != nil {
		return ""
	}
	prefix := name + "_"
	for _, e := range entries {
		base := e.Name()
		if len(base) > 4 && base[len(base)-4:] == ".sql" {
			base = base[:len(base)-4]
		}
		if len(base) > len(prefix) && base[:len(prefix)] == prefix {
			return base[len(prefix):]
		}
	}
	return ""
}

// classifyError maps an error returned from goose.Up to one of the
// four canonical `migration.error.kind` values from design §10.
// Best-effort: when the cause is unknown we return "embed_error"
// as a catch-all rather than "unknown" so the field is never blank.
func classifyError(err error) string {
	if err == nil {
		return ""
	}
	msg := err.Error()
	switch {
	case msgContains(msg, "advisory lock") || msgContains(msg, "lock_timeout") || msgContains(msg, "try_advisory_lock"):
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

// lastIndexByte is a tiny helper to avoid importing strings just
// for one call site.
func lastIndexByte(s string, b byte) int {
	for i := len(s) - 1; i >= 0; i-- {
		if s[i] == b {
			return i
		}
	}
	return -1
}

// quoteIdent quotes a SQL identifier so a malicious or accidental
// table name cannot inject SQL. We control the table name (it's a
// struct field, not user input), but defense in depth is cheap.
func quoteIdent(name string) string {
	return `"` + name + `"`
}

// Compile-time check that GooseRunner satisfies domain.Runner.
// Mirrors the check in runner_test.go; if the public surface drifts
// the build breaks here, not in a downstream consumer.
var _ domain.Runner = (*GooseRunner)(nil)
