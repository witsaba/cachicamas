// AI-37 — the observability boundary: the span seam (WU-4, R-AOB-003) and,
// starting at the WU-5/WU-6 commit, the § D3 attribute mapping
// (R-AOB-004…006). Layer 1 imports exactly the OpenTelemetry tracing API
// (proposal D-1, ADR 0005 § D3): otel/trace, otel/trace/noop, otel/attribute,
// otel/codes — never the root go.opentelemetry.io/otel global-getter
// package. A tracer arrives only by injection (openaicompat.Config,
// client.go); New defaults it with the API's own no-op provider
// (R-APC-016), so every call site below is unconditional and no
// adapter-side nil check on a tracing value exists anywhere in this
// package (R-AOB-009).
//
// # span.RecordError is never called in production code
//
// ai.Failure causes can embed response-body excerpts
// (nonStreamContentType, stream.go) that ADR 0005 § D3's absolute content
// denylist forbids on a span. Every failure this file records onto a span
// carries only the closed nine-member category vocabulary's own name
// (ai.FailureCategory.String()) — never err.Error(), never the cause
// itself.

package openaicompat

import (
	"context"
	"errors"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"

	"github.com/cachicamas/backend/agent/src/ai"
)

// Attribute keys (NFR-AOB-002): each ADR 0005 § D3 key is a literal
// constant, spelled exactly once in this file, and referenced from every
// recording site — never assembled at a call site. WU-4 (the span seam)
// needs only the two below; the remaining ten land at the WU-5/WU-6
// commit.
const (
	retryCountKey = "retry.count"
	errorTypeKey  = "error.type"
)

// spanName is the constant GenAI operation-name span AI-37 starts for
// every logical request (AD-8): cardinality 1, trivially denylist-safe —
// never a per-model name, which would multiply cardinality for no
// attribute value gen_ai.request.model doesn't already carry.
const spanName = "chat"

// tracerInstrumentationName is the instrumentation-scope name every
// Tracer this package derives is created with (provider.go:35-38's own
// naming convention: "the Go package name of the library providing
// instrumentation").
const tracerInstrumentationName = "github.com/cachicamas/backend/agent/src/ai/openaicompat"

// startSpan starts the one span AI-37 wraps a logical request in
// (R-AOB-003): before the first transport attempt (stream.go's own Stream,
// immediately after the R-ATS-002 validation gate), with
// trace.SpanKindClient (AD-8). Every retry attempt therefore falls inside
// this span, never beside it. Request-derived start-time attributes
// (gen_ai.system, gen_ai.request.model, gen_ai.request.max_tokens) land at
// the WU-5/WU-6 commit.
func startSpan(ctx context.Context, tracer trace.Tracer) (context.Context, trace.Span) {
	return tracer.Start(ctx, spanName, trace.WithSpanKind(trace.SpanKindClient))
}

// recordRetryCount sets retry.count (AD-6's carrier, valid on every exit
// of retry.Loop) as its own call, so every path out of Stream — success
// into run's own finalizer, and both pre-handover failure exits via
// endSpanPreHandover — records it exactly once.
func recordRetryCount(span trace.Span, retries int) {
	span.SetAttributes(attribute.Int(retryCountKey, retries))
}

// endSpanPreHandover ends a span whose logical request never reached
// run's own finalizer (AD-5): Stream's own two pre-handover failure exits
// — the retry.Loop error return, and the non-streaming content-type
// refusal. Sets retry.count and, when err carries a classified failure,
// error.type plus a codes.Error status naming the category — never
// err.Error() (see this file's own package comment on why
// span.RecordError is never called). http.response.status_code on the
// content-type-refusal path (AD-5: the code is in hand on that path) lands
// at the WU-5/WU-6 commit alongside the rest of the § D3 attribute table.
func endSpanPreHandover(span trace.Span, retries int, err error) {
	recordRetryCount(span, retries)
	var failure *ai.Failure
	if errors.As(err, &failure) && failure != nil {
		category := failure.Category().String()
		span.SetAttributes(attribute.String(errorTypeKey, category))
		span.SetStatus(codes.Error, category)
	}
	span.End()
}

// spanOutcome accumulates what run's own eleven post-handover terminal
// paths (stream.go) observe, so the span finalizer renders it exactly
// once (AD-5). completion/haveCompletion are consumed starting at the
// WU-5/WU-6 commit (the § D3 attribute mapping); this commit populates
// them at every terminal path so that later commit is additive, not a
// second pass over every one of them. The WU-5/WU-6 commit also adds an
// eventCount field once run gains the counter that increments it
// (stream.event_count) — introducing an unused field ahead of its only
// writer would fail this module's own lint gate.
type spanOutcome struct {
	completion     ai.Completion
	haveCompletion bool
	failure        *ai.Failure
}

// finalizeSpan renders outcome onto span exactly once and ends it — the
// one place run's eleven post-handover terminal paths converge
// (R-AOB-003, AD-5). The full § D3 attribute mapping (gen_ai.*,
// http.response.status_code, stream.event_count) lands at the WU-5/WU-6
// commit; this commit sets error.type/status on failure and codes.Ok on
// success — the span seam's own minimal terminal state.
func finalizeSpan(span trace.Span, outcome *spanOutcome) {
	if outcome.failure != nil {
		category := outcome.failure.Category().String()
		span.SetAttributes(attribute.String(errorTypeKey, category))
		span.SetStatus(codes.Error, category)
	} else {
		span.SetStatus(codes.Ok, "")
	}
	span.End()
}
