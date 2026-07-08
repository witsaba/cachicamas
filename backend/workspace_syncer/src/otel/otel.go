// Package otel wires the OpenTelemetry tracer and the slog logger
// for the workspace_syncer service. Mirrors the same pattern in
// database_administrator/src/otel/.
//
// PR-2a uses a no-op tracer (otel.Tracer wraps to a noop tracer when
// the OTLP env var is empty). PR-2c upgrades the no-op to a real
// OTLP/gRPC tracer once the `clone.execute` span is introduced.
package otel

import (
	"go.opentelemetry.io/otel/trace"
	"go.opentelemetry.io/otel/trace/noop"
)

// NewTracer returns a trace.Tracer for the workspace_syncer. In v1
// (PR-2a) the tracer is a no-op; PR-2c upgrades this to a real OTLP
// exporter when OTEL_EXPORTER_OTLP_ENDPOINT is set.
//
// The tracer name `workspace_syncer` is locked — it is the value
// Jaeger/OTel UI uses to filter spans emitted by this service.
func NewTracer() trace.Tracer {
	// In PR-2a we always return the no-op tracer. PR-2c adds the
	// conditional real tracer; the rest of the service code
	// already uses the trace.Tracer interface so the upgrade is
	// non-breaking.
	return noop.NewTracerProvider().Tracer("workspace_syncer")
}
