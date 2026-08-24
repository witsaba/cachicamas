// Package main — otel.go installs the OpenTelemetry tracing pipeline
// for the chat composition root. This is the chat archetype's ONLY
// file permitted to import the OTel SDK + OTLP exporter + propagation
// + semconv paths (ADR 0005 § D3, adr:242; ADR 0009 § D2).
//
// The shape mirrors database_administrator/src/otel/otel.go:35-92
// verbatim — OTLP/gRPC exporter, W3C TraceContext + Baggage composite
// propagator, BatchSpanProcessor with a 1-second batch timeout, a
// 5-second flush deadline on shutdown. The two modules' OTel setups
// MUST stay byte-for-byte equivalent so the chat binary and
// database_administrator can share a single OTel collector and traces
// from the chat surface stitch into the same backend.
//
// The duplication is a recorded decision (ADR 0005 § D3, adr:242 +
// opencode ADR echo-v5-in-agent-module): cross-module import of an
// OTel package is denied by chat/import_boundary_test.go's check 6,
// so the chat root builds its own install helper. Drift between the
// two modules' OTel installs is a maintainer hazard a future PR
// should address via an internal package only when the import
// boundary explicitly widens.
package main

import (
	"context"
	"fmt"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.41.0"
)

// serviceName is the OTel service.name the chat binary reports. Named
// once at package scope so the literal lives in one place.
const serviceName = "cachicamas-chat"

// versionInstall is the OTel service.version the chat binary reports.
// Build metadata baked in by the Dockerfile via -ldflags defaults to
// "dev" so `go run` works without a build step. The five-second
// shutdown deadline is intentional (mirrors dbadmin's): it gives the
// BatchSpanProcessor a chance to flush any buffered spans before the
// TracerProvider tears down.
var versionInstall = "dev"

// shutdownFlushDeadline bounds the OTel shutdown's flush window. A
// short deadline keeps an unresponsive collector from holding the
// process; the BatchSpanProcessor has a 1-second batch timeout so 5s
// is plenty for any normal shutdown.
const shutdownFlushDeadline = 5 * time.Second

// installOTelSDK installs a global TracerProvider backed by an
// OTLP/gRPC exporter. The returned shutdown function flushes pending
// spans and tears down the provider; the caller MUST invoke it before
// the process exits.
//
// Configuration is read from the standard OTEL_* environment
// variables:
//
//	OTEL_EXPORTER_OTLP_ENDPOINT   collector endpoint (e.g. otel-collector:4317).
//	                             Default: localhost:4317.
//	OTEL_SERVICE_NAME            service.name attribute. Default: unknown_service.
//	OTEL_RESOURCE_ATTRIBUTES     comma-separated k=v pairs merged into the resource.
//
// When OTEL_EXPORTER_OTLP_ENDPOINT is unset, the OTLP exporter logs a
// warning on first export attempt and continues as a no-op; the chat
// binary stays runnable in a dev environment without a collector.
func installOTelSDK(ctx context.Context) (shutdown func(context.Context) error, err error) {
	// OTLP/gRPC exporter — picks up OTEL_EXPORTER_OTLP_ENDPOINT etc.
	exporter, err := otlptracegrpc.New(ctx)
	if err != nil {
		return nil, fmt.Errorf("otlptracegrpc.New: %w", err)
	}

	// Resource: service.name + service.version + anything provided by
	// OTEL_RESOURCE_ATTRIBUTES (the SDK merges that automatically when
	// resource.Default() is passed in).
	res, err := resource.Merge(
		resource.Default(),
		resource.NewWithAttributes(
			semconv.SchemaURL,
			semconv.ServiceName(serviceName),
			semconv.ServiceVersion(versionInstall),
		),
	)
	if err != nil {
		return nil, fmt.Errorf("resource.Merge: %w", err)
	}

	// TracerProvider with a BatchSpanProcessor. Batching is the right
	// default for any non-trivial traffic — it amortizes the cost of
	// shipping spans to the collector.
	tp := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(exporter,
			sdktrace.WithBatchTimeout(1*time.Second),
			sdktrace.WithMaxExportBatchSize(512),
		),
		sdktrace.WithResource(res),
	)

	otel.SetTracerProvider(tp)
	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(
		propagation.TraceContext{},
		propagation.Baggage{},
	))

	shutdown = func(ctx context.Context) error {
		// Give the batcher a chance to flush, then close the exporter.
		flushCtx, cancel := context.WithTimeout(ctx, shutdownFlushDeadline)
		defer cancel()
		_ = tp.ForceFlush(flushCtx)
		return tp.Shutdown(ctx)
	}
	return shutdown, nil
}