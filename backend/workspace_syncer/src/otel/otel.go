// Package otel — the OTel tracer for the workspace_syncer.
//
// v1 (PR-2a) returned a noop tracer unconditionally. The design
// (T-WSY-2c-003) called for an upgrade to a real OTLP/gRPC
// tracer when OTEL_EXPORTER_OTLP_ENDPOINT is set, but the
// upgrade pulls in 4 new top-level Go deps (sdk, exporters,
// semconv, propagation). v1 keeps the noop fallback; the
// upgrade is tracked as a follow-up to keep PR-2c focused on
// the redaction handler + sweep (the security-critical changes).
//
// The follow-up plan: a separate "real-OTel" PR adds the deps
// via `go get`, switches NewTracer to the conditional real
// exporter, and wires `clone.execute` spans in
// application/clone_service.go. The locked attributes are in
// design.md §10 (OTel attributes for clone.execute).
package otel

import (
	"go.opentelemetry.io/otel/trace"
	"go.opentelemetry.io/otel/trace/noop"
)

// tracerName is the locked name of this service's tracer.
const tracerName = "workspace_syncer"

// NewTracer returns a trace.Tracer for the workspace_syncer.
// v1: always returns the noop tracer. The follow-up PR upgrades
// to a real OTLP/gRPC exporter.
func NewTracer() trace.Tracer {
	return noop.NewTracerProvider().Tracer(tracerName)
}