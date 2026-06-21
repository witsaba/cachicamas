// Package domain contains the core business types of the database
// administrator service. It has no dependencies on frameworks,
// transport, or infrastructure.
//
// This file defines the hexagonal port for the SQL migration
// runner. Per design §3 + §14, the application layer calls this
// port; the migration package provides the goose-backed adapter.
// Neither this file nor anything under src/domain/ imports from
// src/migration/.
package domain

import (
	"context"
	"time"
)

// Version is the result of one applied migration, returned by
// Runner.Up and Runner.Status. The fields mirror goose's internal
// representation:
//
//   - ID          — the timestamp prefix of the migration file, parsed
//     as int64 (e.g. 20260621120000 from
//     20260621120000_hello_world.sql).
//   - Description — the part of the filename after the timestamp
//     prefix (e.g. "hello_world"). Goose fills this from the filename
//     automatically.
//   - AppliedAt   — the wall-clock time at which the migration was
//     applied (Postgres `tstamp` column in `public.schema_migrations`).
type Version struct {
	ID          int64
	Description string
	AppliedAt   time.Time
}

// Runner is the hexagonal port the application layer uses to apply
// and inspect SQL migrations. Implementations live in
// src/migration/ (the goose-backed adapter). The interface is
// deliberately small so a fake is trivial to write in
// application/migration_service_test.go (see spec R-DBMIG-030,
// scenario S-DBMIG-031).
//
// Contract:
//
//   - Up(ctx) applies every pending migration (in lexicographic order)
//     and returns the slice of versions it applied. If the bookkeeping
//     table is up to date, it returns an empty slice and a nil error.
//   - Status(ctx) returns the slice of versions already in the
//     bookkeeping table (regardless of whether they need re-running).
//   - Both methods MUST honour the context — a cancelled context
//     must abort promptly.
type Runner interface {
	Up(ctx context.Context) ([]Version, error)
	Status(ctx context.Context) ([]Version, error)
}
