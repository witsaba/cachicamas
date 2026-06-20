// Package otel wires the OpenTelemetry tracing pipeline for the
// database_administrator service. It is intentionally minimal:
//
//   - SetupTracing installs a global TracerProvider that exports OTLP/gRPC
//     traces to the collector configured via the standard OTEL_* env vars
//     (OTEL_EXPORTER_OTLP_ENDPOINT, OTEL_SERVICE_NAME,
//     OTEL_RESOURCE_ATTRIBUTES).
//   - Middleware returns an Echo middleware that opens a server span per
//     HTTP request, propagates W3C trace context, and records the standard
//     http.* attributes plus the response status.
//
// We do not depend on the otelecho contrib package because it only
// supports labstack/echo/v4 — we are on v5. The middleware below is
// ~50 lines of glue and uses the same semantic conventions as
// otelecho / otelhttp.
package otel

import (
	"context"
	"fmt"
	"time"

	"github.com/labstack/echo/v5"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.41.0"
	"go.opentelemetry.io/otel/trace"
)

// SetupTracing installs a global TracerProvider backed by an OTLP/gRPC
// exporter. The returned shutdown function flushes pending spans and tears
// down the provider; call it before the process exits.
//
// Configuration is read from the standard OTEL_* environment variables:
//
//	OTEL_EXPORTER_OTLP_ENDPOINT   collector endpoint (e.g. otel-collector:4317).
//	                             Default: localhost:4317.
//	OTEL_SERVICE_NAME            service.name attribute. Default: unknown_service.
//	OTEL_RESOURCE_ATTRIBUTES     comma-separated k=v pairs merged into the resource.
func SetupTracing(ctx context.Context, serviceName, serviceVersion string) (shutdown func(context.Context) error, err error) {
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
			semconv.ServiceVersion(serviceVersion),
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
		flushCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
		defer cancel()
		_ = tp.ForceFlush(flushCtx)
		return tp.Shutdown(ctx)
	}
	return shutdown, nil
}

// Middleware returns an Echo middleware that opens a server span per HTTP
// request. The span name follows the "{METHOD} {ROUTE}" convention used by
// most tracers (otelecho, otelhttp, otelmux). When the route is unknown
// (404), it falls back to "{METHOD} {PATH}".
//
// Trace context is extracted from incoming request headers via the
// configured propagator (W3C traceparent by default), so traces from
// upstream callers stitch together automatically.
func Middleware(serviceName string) echo.MiddlewareFunc {
	tracer := otel.Tracer(serviceName)
	propagator := otel.GetTextMapPropagator()

	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c *echo.Context) error {
			req := c.Request()

			// Extract any incoming trace context from headers.
			ctx := propagator.Extract(req.Context(), propagation.HeaderCarrier(req.Header))

			// Pick a span name based on whether the route is matched.
			route := c.Path()    // registered template, e.g. "/users/:id"
			path := req.URL.Path // concrete path, e.g. "/users/42"
			spanName := req.Method + " " + route
			if route == "" {
				spanName = req.Method + " " + path
			}

			ctx, span := tracer.Start(ctx, spanName,
				trace.WithSpanKind(trace.SpanKindServer),
				trace.WithAttributes(
					semconv.HTTPRequestMethodKey.String(req.Method),
					semconv.URLPath(path),
					semconv.URLScheme(req.URL.Scheme),
					semconv.UserAgentOriginal(req.UserAgent()),
					attribute.String("http.route", route),
				),
			)
			defer span.End()

			// Make the span available to downstream handlers via context.
			c.SetRequest(req.WithContext(ctx))

			err := next(c)

			// Record status code + mark span error if the handler returned
			// a non-nil error or wrote a 5xx. Echo v5 returns
			// http.ResponseWriter from c.Response(); the underlying *Response
			// carries the Status int.
			status := responseStatus(c)
			span.SetAttributes(semconv.HTTPResponseStatusCode(status))
			if err != nil {
				span.RecordError(err)
			}
			if status >= 500 {
				span.SetStatus(codes.Error, errString(err))
			}
			return err
		}
	}
}

// responseStatus pulls the status code out of the Echo v5 Response
// wrapper. Echo v5's Context.Response() returns http.ResponseWriter; the
// concrete *echo.Response has a Status int field. If we can't unwrap it
// (e.g. a custom writer), we fall back to 0 (unset).
func responseStatus(c *echo.Context) int {
	type statuser interface{ Status() int }
	if s, ok := c.Response().(statuser); ok {
		return s.Status()
	}
	// Fall back to the concrete *echo.Response (covers the default writer).
	if r, ok := c.Response().(*echo.Response); ok {
		return r.Status
	}
	return 0
}

func errString(err error) string {
	if err == nil {
		return ""
	}
	return err.Error()
}
