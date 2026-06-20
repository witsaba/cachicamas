// Package otel wires the OpenTelemetry logging pipeline for the
// database_administrator service.
//
// Pipeline:
//
//	log/slog (stdlib) → otelslog bridge → OTel Logs SDK → OTLP/gRPC exporter
//	                  → otel-collector → debug exporter (stdout, dev only).
//
// Why slog + otelslog (and not zerolog or zap):
//
//   - slog is part of the Go standard library (Go 1.21+); no third-party
//     dependency for the logger API itself.
//   - otelslog is the official OTel-Go bridge (go.opentelemetry.io/contrib
//     /bridges/otelslog). It implements slog.Handler and injects trace_id /
//     span_id into every log record automatically.
//   - OTel Logs SDK + OTLP/gRPC exporter push the structured logs to the
//     collector. Jaeger v2 does not store logs (it is a tracing backend),
//     so the collector emits them via the debug exporter; you can grep
//     `docker compose logs otel-collector` for the TraceID and jump to the
//     matching span in Jaeger UI.
//
// zerolog and zap would each shave a few bytes off per log call, but
// neither has an official OTel bridge, and this service is not in a hot
// loop. We can swap to zap later (otelzap is official too) without
// touching call sites — slog.Handler is the seam.
package otel

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"time"

	"go.opentelemetry.io/contrib/bridges/otelslog"
	"go.opentelemetry.io/otel/exporters/otlp/otlplog/otlploggrpc"
	"go.opentelemetry.io/otel/log/global"
	sdklog "go.opentelemetry.io/otel/sdk/log"
)

// SetupLogging installs a global LoggerProvider backed by an OTLP/gRPC
// exporter and returns a *slog.Logger that emits records through it. The
// bridge (otelslog) attaches trace_id and span_id automatically when the
// slog call site carries a context.Context with an active span.
//
// Records also fall through to stderr via a slog JSON handler, so logs
// remain visible even when the OTel collector is unreachable (fail-open).
//
// The returned shutdown function flushes pending records and tears down
// the provider; call it before the process exits.
func SetupLogging(ctx context.Context, serviceName, serviceVersion string) (logger *slog.Logger, shutdown func(context.Context) error, err error) {
	// OTLP/gRPC exporter for logs. Honors OTEL_EXPORTER_OTLP_ENDPOINT
	// (default localhost:4317) and OTEL_EXPORTER_OTLP_PROTOCOL.
	exporter, err := otlploggrpc.New(ctx)
	if err != nil {
		return nil, nil, fmt.Errorf("otlploggrpc.New: %w", err)
	}

	// LoggerProvider with a BatchProcessor. Same rationale as the
	// tracer provider: amortize the cost of shipping records to the
	// collector.
	lp := sdklog.NewLoggerProvider(
		sdklog.WithProcessor(sdklog.NewBatchProcessor(exporter,
			sdklog.WithExportInterval(1*time.Second),
			sdklog.WithExportMaxBatchSize(512),
		)),
	)
	global.SetLoggerProvider(lp)

	// Build the slog logger. Two handlers in series:
	//
	//   1. otelslog.NewHandler — turns the slog record into an OTel log
	//      record, attaches trace_id/span_id from the context, and ships
	//      it through the LoggerProvider above (which exports via OTLP).
	//   2. slog.NewJSONHandler writing to stderr — a fail-open mirror so
	//      we still see logs locally if the collector is down.
	otelHandler := otelslog.NewHandler(serviceName,
		otelslog.WithLoggerProvider(lp),
		otelslog.WithVersion(serviceVersion),
	)
	stderrHandler := slog.NewJSONHandler(os.Stderr, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	})

	logger = slog.New(&teeHandler{primary: otelHandler, mirror: stderrHandler})

	shutdown = func(ctx context.Context) error {
		flushCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
		defer cancel()
		_ = lp.ForceFlush(flushCtx)
		return lp.Shutdown(ctx)
	}
	return logger, shutdown, nil
}

// teeHandler fans out a single slog.Record to two underlying handlers:
// the OTel bridge (primary) and a stderr JSON mirror. Both handlers must
// agree on Enabled() so we don't drop records based on level mismatch.
type teeHandler struct {
	primary slog.Handler
	mirror  slog.Handler
}

func (h *teeHandler) Enabled(ctx context.Context, level slog.Level) bool {
	return h.primary.Enabled(ctx, level) || h.mirror.Enabled(ctx, level)
}

func (h *teeHandler) Handle(ctx context.Context, r slog.Record) error {
	// Best-effort fan-out: ignore errors from each handler so a transient
	// failure in one (e.g. collector down) does not silence the other.
	if h.primary.Enabled(ctx, r.Level) {
		_ = h.primary.Handle(ctx, r.Clone())
	}
	if h.mirror.Enabled(ctx, r.Level) {
		_ = h.mirror.Handle(ctx, r.Clone())
	}
	return nil
}

func (h *teeHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	return &teeHandler{
		primary: h.primary.WithAttrs(attrs),
		mirror:  h.mirror.WithAttrs(attrs),
	}
}

func (h *teeHandler) WithGroup(name string) slog.Handler {
	return &teeHandler{
		primary: h.primary.WithGroup(name),
		mirror:  h.mirror.WithGroup(name),
	}
}
