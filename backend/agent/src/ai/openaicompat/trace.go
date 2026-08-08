// AI-37 — the observability boundary: the span seam (WU-4, R-AOB-003) and
// the § D3 attribute mapping (WU-5/WU-6, R-AOB-004…006). Layer 1 imports
// exactly the OpenTelemetry tracing API (proposal D-1, ADR 0005 § D3):
// otel/trace, otel/trace/noop, otel/attribute, otel/codes — never the root
// go.opentelemetry.io/otel global-getter package. A tracer arrives only by
// injection (openaicompat.Config, client.go); New defaults it with the
// API's own no-op provider (R-APC-016), so every call site below is
// unconditional and no adapter-side nil check on a tracing value exists
// anywhere in this package (R-AOB-009).
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

// Attribute keys (NFR-AOB-002): each of the twelve ADR 0005 § D3 keys is a
// literal constant, spelled exactly once in this file, and referenced from
// every recording site — never assembled at a call site.
const (
	genAISystemKey           = "gen_ai.system"
	requestModelKey          = "gen_ai.request.model"
	requestMaxTokensKey      = "gen_ai.request.max_tokens"
	finishReasonsKey         = "gen_ai.response.finish_reasons"
	usageInputTokensKey      = "gen_ai.usage.input_tokens"
	usageOutputTokensKey     = "gen_ai.usage.output_tokens"
	usageCacheReadTokensKey  = "gen_ai.usage.cache_read_tokens"
	usageCacheWriteTokensKey = "gen_ai.usage.cache_write_tokens"
	statusCodeKey            = "http.response.status_code"
	retryCountKey            = "retry.count"
	streamEventCountKey      = "stream.event_count"
	errorTypeKey             = "error.type"
)

// spanName is the constant GenAI operation-name span AI-37 starts for
// every logical request (AD-8): cardinality 1, trivially denylist-safe —
// never a per-model name, which would multiply cardinality for no
// attribute value gen_ai.request.model doesn't already carry.
const spanName = "chat"

// genAISystem names the dialect this package speaks — not the vendor
// behind the endpoint (AD-7, proposal D-3a): the same adapter serves
// OpenAI, OpenRouter and any compatible gateway, so a vendor value would
// be a lie. Two adapters on two different vendor endpoints record the
// identical value (S-AOB-018).
const genAISystem = "openai_compat"

// tracerInstrumentationName is the instrumentation-scope name every
// Tracer this package derives is created with (provider.go:35-38's own
// naming convention: "the Go package name of the library providing
// instrumentation").
const tracerInstrumentationName = "github.com/cachicamas/backend/agent/src/ai/openaicompat"

// startSpan starts the one span AI-37 wraps a logical request in
// (R-AOB-003): before the first transport attempt (stream.go's own Stream,
// immediately after the R-ATS-002 validation gate), with
// trace.SpanKindClient (AD-8) and the three request-derived start-time
// attributes (tracer.go:30-32 recommends attributes at start for
// samplers). Every retry attempt therefore falls inside this span, never
// beside it.
func startSpan(ctx context.Context, tracer trace.Tracer, req ai.Request) (context.Context, trace.Span) {
	return tracer.Start(ctx, spanName,
		trace.WithSpanKind(trace.SpanKindClient),
		trace.WithAttributes(startAttributes(req)...),
	)
}

// startAttributes builds the three request-derived attributes recorded at
// span start (S-AOB-012…014): gen_ai.system (the dialect constant, AD-7),
// gen_ai.request.model (verbatim), and gen_ai.request.max_tokens
// (presence-typed — S-AOB-016: omitted, never a fabricated zero, when the
// request carries no bound).
func startAttributes(req ai.Request) []attribute.KeyValue {
	attrs := []attribute.KeyValue{
		attribute.String(genAISystemKey, genAISystem),
		attribute.String(requestModelKey, req.Model()),
	}
	if maxTokens, ok := req.MaxOutputTokens(); ok {
		attrs = append(attrs, attribute.Int(requestMaxTokensKey, maxTokens))
	}
	return attrs
}

// recordRetryCount sets retry.count (AD-6's carrier, valid on every exit
// of retry.Loop) as its own call, so every path out of Stream — success
// into run's own finalizer, and both pre-handover failure exits via
// endSpanPreHandover — records it exactly once.
func recordRetryCount(span trace.Span, retries int) {
	span.SetAttributes(attribute.Int(retryCountKey, retries))
}

// endSpanPreHandover ends a span whose logical request never reached
// run's own finalizer (AD-5): Stream's own two pre-handover failure exits.
// Sets retry.count and, when err carries a classified failure, error.type
// plus a codes.Error status naming the category — never err.Error() (see
// this file's own package comment on why span.RecordError is never
// called).
//
// haveStatusCode/statusCode carry R-AOB-006's split (D-3b): the
// retry.Loop-error exit has no *http.Response at all (haveStatusCode ==
// false — only a status CLASS ever survived that path, and the class must
// never be recorded under this or any other key, S-AOB-023). The
// content-type-refusal exit DOES have the exact code in hand
// (resp.StatusCode, stream.go) — recorded when supplied, exactly as the
// success terminal records it in finalizeSpan.
func endSpanPreHandover(span trace.Span, retries int, haveStatusCode bool, statusCode int, err error) {
	recordRetryCount(span, retries)
	if haveStatusCode {
		span.SetAttributes(attribute.Int(statusCodeKey, statusCode))
	}
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
// once (AD-5).
type spanOutcome struct {
	completion     ai.Completion
	haveCompletion bool
	failure        *ai.Failure
	// eventCount is AD-6's stream.event_count carrier: incremented at
	// every confirmed send onto the carrier — run's own direct sends and
	// every send emitFailure performs on run's behalf (R-AOB-005,
	// S-AOB-021: it must equal the number of events actually drained).
	eventCount int
}

// finalizeSpan renders outcome onto span exactly once and ends it — the
// one place run's eleven post-handover terminal paths converge
// (R-AOB-003, AD-5). statusCode is resp.StatusCode (stream.go's run is
// always called with a successful, content-type-accepted response, so the
// exact code is always in hand here — unlike endSpanPreHandover's own
// split).
func finalizeSpan(span trace.Span, statusCode int, outcome *spanOutcome) {
	span.SetAttributes(attribute.Int(streamEventCountKey, outcome.eventCount))
	// AI-37 Judgment Day round 2 (R-AOB-006 bullet 2, S-AOB-041): statusCode
	// is always resp.StatusCode -- the one winning response run itself is
	// only ever launched with (see this function's own doc comment above)
	// -- so it is genuinely "in hand" on every post-handover exit this
	// finalizer renders, failure or success alike, exactly like the
	// pre-handover content-type-refusal exit already records it
	// (endSpanPreHandover, S-AOB-040). Recorded unconditionally, before
	// branching on outcome.failure, so the failure branch below no longer
	// omits a value that was never actually in doubt: the round-1 spec
	// amendment's own general clause already said as much, the code just
	// had not carried it through to this branch.
	span.SetAttributes(attribute.Int(statusCodeKey, statusCode))

	if outcome.failure != nil {
		category := outcome.failure.Category().String()
		span.SetAttributes(attribute.String(errorTypeKey, category))
		span.SetStatus(codes.Error, category)
		span.End()
		return
	}

	if outcome.haveCompletion {
		span.SetAttributes(terminalCompletionAttributes(outcome.completion)...)
	}
	span.SetStatus(codes.Ok, "")
	span.End()
}

// terminalCompletionAttributes builds the finish-reason and usage
// attributes a successfully completed stream carries (S-AOB-012…013,
// S-AOB-016…017): gen_ai.response.finish_reasons is a one-element array
// (Layer 1 tracks exactly one reason); each usage count is presence-typed
// individually, never a fabricated zero standing in for absence; the
// reasoning-token breakdown (Usage.Reasoning) is never recorded under any
// key (it is a breakdown of Output, not a term beside it — usage.go).
func terminalCompletionAttributes(completion ai.Completion) []attribute.KeyValue {
	attrs := []attribute.KeyValue{
		attribute.StringSlice(finishReasonsKey, []string{completion.FinishReason().String()}),
	}
	usage := completion.Usage()
	if v, ok := usage.Input.Count(); ok {
		attrs = append(attrs, attribute.Int64(usageInputTokensKey, v))
	}
	if v, ok := usage.Output.Count(); ok {
		attrs = append(attrs, attribute.Int64(usageOutputTokensKey, v))
	}
	if v, ok := usage.CacheRead.Count(); ok {
		attrs = append(attrs, attribute.Int64(usageCacheReadTokensKey, v))
	}
	if v, ok := usage.CacheWrite.Count(); ok {
		attrs = append(attrs, attribute.Int64(usageCacheWriteTokensKey, v))
	}
	return attrs
}
