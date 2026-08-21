// AG-22 — the observability boundary: Layer 2's OpenTelemetry tracing
// API seam (design D-A) and the ADR 0005 § D3 attribute vocabulary
// (design D-B), mirroring AI-37's own shape (src/ai/openaicompat/trace.go)
// at four span families — invoke_agent (run), turn, execute_tool {name}
// (tool call) and compact (compaction) — instead of Layer 1's single
// request span.
//
// A tracer arrives only by injection (Harness.TracerProvider) or by
// ambient acquisition from an already-recording span already occupying
// ctx (tracerFromContext, below) — never the process-global getter
// go.opentelemetry.io/otel.Tracer(...), which design D-A rejects: its
// own measured closure contains go.opentelemetry.io/auto/sdk, an SDK
// path S-AGP-024 forbids any allowlist entry from naming.
//
// This file holds the constants, the two tracer-acquisition helpers
// (injected, for Harness.Run/Harness.Compact; ambient, for every nested
// seam) and one finalizer per span family (D-D's placement rule: a span
// opens iff its bracket's Start event is emitted, via a deferred
// finalizer registered at open). harness.go, loop.go, scheduler.go and
// compaction.go wire these in at Phases 5-8.
//
// # span.RecordError is never called in production code
//
// Every failure a finalizer below records onto a span carries only the
// closed nine-member ai.FailureCategory vocabulary's own name
// (agent.Failure.Category().String()) — never a Failure's Unwrap() text,
// and never any Result/Arguments content (design D-E's per-field
// denylist, ADR 0005 § D3's absolute denylist).
package agent

import (
	"context"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
	"go.opentelemetry.io/otel/trace/noop"
)

// tracerScope is the instrumentation-scope name every Tracer this
// package derives is created with — AI-37's own naming convention
// (openaicompat/trace.go: "the Go package name of the library providing
// instrumentation").
const tracerScope = "github.com/cachicamas/backend/agent/src/agent"

// Span names (ADR 0005 § D3, Layer 2 extension). invokeAgentSpanName,
// turnSpanName and compactionSpanName are complete span names on their
// own. toolSpanNamePrefix is NOT a complete name by itself — it is
// combined with a tool's own name at the call site via toolSpanName,
// below, because the tool span's name is per-call, not a fixed constant.
const (
	invokeAgentSpanName = "invoke_agent"
	turnSpanName        = "turn"
	toolSpanNamePrefix  = "execute_tool"
	compactionSpanName  = "compact"
)

// toolSpanName builds one tool-call span's name: the GenAI convention
// prefix plus the tool's own name (ADR 0005 § D3 Layer 2 extension,
// "execute_tool {name}"). Pure — no attribute assembly, no side effect.
func toolSpanName(toolName string) string {
	return toolSpanNamePrefix + " " + toolName
}

// Attribute keys (ADR 0005 § D3, Layer 2 extension): each key is a
// literal constant, spelled exactly once in this file and referenced
// from every recording site — never assembled at a call site (mirrors
// openaicompat/trace.go's own attribute-key discipline).
const (
	genAIOperationNameKey  = "gen_ai.operation.name"
	runIDKey               = "cachicamas.run.id"
	runParentIDKey         = "cachicamas.run.parent_id"
	runOutcomeKey          = "cachicamas.run.outcome"
	turnIDKey              = "cachicamas.turn.id"
	turnOutcomeKey         = "cachicamas.turn.outcome"
	toolNameKey            = "gen_ai.tool.name"
	toolCallIDKey          = "gen_ai.tool.call.id"
	toolOrdinalKey         = "cachicamas.tool.ordinal"
	toolOutcomeKey         = "cachicamas.tool.outcome"
	toolDetachedKey        = "cachicamas.tool.detached"
	compactionIDKey        = "cachicamas.compaction.id"
	compactionSummaryIDKey = "cachicamas.compaction.summary_id"

	// errorTypeKey is the one key both this file's spans and AI-37's
	// own request span use (ADR 0005 § D3 Layer 2 extension: "the
	// shared standard key reused on a different span, not a
	// re-record").
	errorTypeKey = "error.type"
)

// genAIOperationNameInvokeAgent is the constant value gen_ai.operation.name
// carries on every run span — a GenAI convention-required value, not an
// attribute key of its own.
const genAIOperationNameInvokeAgent = "invoke_agent"

// tracerFromContext resolves the ambient tracer a nested seam (Turn, the
// scheduler, runCompaction) records onto: the TracerProvider owning
// whatever span already occupies ctx (design D-A). When ctx carries no
// recording span, trace.SpanFromContext returns the tracing API's own
// no-op span, whose TracerProvider() is itself a no-op provider — so
// this never panics and needs no nil check (R-AOB-009's Layer 1
// precedent, restated for Layer 2; NFR-AGO-001).
func tracerFromContext(ctx context.Context) trace.Tracer {
	return trace.SpanFromContext(ctx).TracerProvider().Tracer(tracerScope)
}

// tracerFromHarness resolves the tracer Harness.Run and Harness.Compact
// record their own run span onto: h.TracerProvider (design D-A),
// defaulting to the tracing API's own no-op provider when tp is nil, so
// a zero-value Harness stays inert and nothing panics (R-AGO-001,
// NFR-AGO-001). This is the ONE place a nil TracerProvider is resolved;
// every nested seam (Turn, the scheduler, runCompaction) acquires its
// own tracer ambiently instead, via tracerFromContext above.
func tracerFromHarness(tp trace.TracerProvider) trace.Tracer {
	if tp == nil {
		tp = noop.NewTracerProvider()
	}
	return tp.Tracer(tracerScope)
}

// spanOutcome renders the closed-vocabulary status ADR 0005 § D3's
// Layer 2 extension pins onto span: codes.Error with the category name
// as its description iff failed is true, codes.Ok otherwise. category is
// ignored when failed is false. Every finalizer below is called with a
// category already extracted from a *Failure that outcome-typing
// guarantees is present on the failed arm
// (agent.Failure.Category().String()) — never a Failure's Unwrap() text
// (design D-E).
func spanOutcome(span trace.Span, failed bool, category string) {
	if failed {
		span.SetAttributes(attribute.String(errorTypeKey, category))
		span.SetStatus(codes.Error, category)
		return
	}
	span.SetStatus(codes.Ok, "")
}

// finalizeRunSpan renders the run bracket's terminal attributes (outcome,
// and iff failed the category) onto span and ends it exactly once
// (design D-B, D-D). Called from Harness.Run's own bracket-closing
// funnel (failRun, windDownRun, the success tail — Phase 5), and from
// Turn's own nil-continuation run bracket (Phase 6).
func finalizeRunSpan(span trace.Span, outcome string, failed bool, category string) {
	span.SetAttributes(attribute.String(runOutcomeKey, outcome))
	spanOutcome(span, failed, category)
	span.End()
}

// finalizeTurnSpan renders the turn bracket's terminal attributes onto
// span and ends it exactly once (design D-B, D-D). Called at every one
// of Turn's own closing sites (emitPreStreamAbort, finishContinuation-
// Turn, the mid-stream fatal block, the cancellation branch, both
// normal-completion tails — Phase 6).
func finalizeTurnSpan(span trace.Span, outcome string, failed bool, category string) {
	span.SetAttributes(attribute.String(turnOutcomeKey, outcome))
	spanOutcome(span, failed, category)
	span.End()
}

// finalizeToolSpan renders the tool-call bracket's terminal attributes
// onto span and ends it exactly once (design D-B, D-D) — including the
// detached-arm marker (D-D), set only when detached is true (D-B: "iff
// true"). Called from executeCall's single deferred finalizer,
// registered at open, covering every post-gate exit (Phase 7).
func finalizeToolSpan(span trace.Span, outcome string, detached bool, failed bool, category string) {
	span.SetAttributes(attribute.String(toolOutcomeKey, outcome))
	if detached {
		span.SetAttributes(attribute.Bool(toolDetachedKey, true))
	}
	spanOutcome(span, failed, category)
	span.End()
}

// finalizeCompactionSpan renders the compaction bracket's terminal
// attributes onto span and ends it exactly once (design D-B, D-D) —
// including the summary identity, set only when the compaction finished
// (D-B: "iff finished"). Called from emitCompactionFailedArm (every
// fail arm) and runCompaction's own success tail — the two closing
// sites the compaction bracket's exactly-one-span rule funnels through
// (Phase 8).
func finalizeCompactionSpan(span trace.Span, outcome string, summaryID string, failed bool, category string) {
	span.SetAttributes(attribute.String(turnOutcomeKey, outcome))
	if summaryID != "" {
		span.SetAttributes(attribute.String(compactionSummaryIDKey, summaryID))
	}
	spanOutcome(span, failed, category)
	span.End()
}
