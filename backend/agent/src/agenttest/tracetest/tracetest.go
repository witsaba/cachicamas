// Package tracetest is a hand-rolled recording implementation of the
// OpenTelemetry tracing API (AI-37, proposal D-2): a Provider, Tracer and
// Span that retain every field the API lets a caller set on a span, so
// AI-37's every later proof — span lifecycle (R-AOB-003), attribute mapping
// (R-AOB-004), and denylist absence with its non-vacuity guards
// (R-AOB-007/R-AOB-008) — has something to inspect afterward.
//
// # Why a hand-rolled recorder, not the OTel SDK's own test fake
//
// A test-only OTel SDK dependency is genuinely blocked at Layer 1, not
// merely inconvenient: import_boundary_test.go's forbiddenPrefixes carries
// go.opentelemetry.io/otel/sdk, checked BEFORE the allowlist, over the
// package set `go list -deps -test` produces — a _test.go import lands in
// that set exactly as a production import does. Admitting it would mean
// deleting a forbidden-prefix row, "the exact failure ADR 0005 § Guard A
// was written against" (agent-module-scaffold/spec.md:20). ADR 0005 § D3's
// table marks go.opentelemetry.io/otel/sdk/… ❌ for L1 with no test-only
// column, and its closing blockquote requires "its own ADR" for any
// additional OTel module.
//
// # Why this package, not src/ai/internal/...
//
// It follows the agenttest/sweep precedent exactly, and for the same
// recorded reason: agenttest imports "testing" package-wide
// (conformance_redaction.go), so a subpackage that imports neither
// "testing" nor anything else is what makes this helper importable from
// any package in the module (agenttest/sweep/sweep.go's own package
// comment). src/ai/internal/... would be excluded by Go's own internal
// rule from agenttest itself — the external-consumer-proof location ADR
// 0005 § D2 designates.
//
// # Embedding the noop types (AD-3), and the hazard that motivates AD-4
//
// Provider, Tracer and Span each embed their go.opentelemetry.io/otel/trace/noop
// counterpart. The noop package documents this exact use: "This
// implementation can be embedded in other implementations of the
// OpenTelemetry trace API. Doing so will mean the implementation defaults
// to no operation for methods it does not implement" (noop.go). Embedding
// noop rather than the bare embedded.* marker types means a version bump
// that adds a method to the Span/Tracer/TracerProvider interfaces does not
// break compilation here — but an inherited, un-overridden method would
// then silently NOT record, a corpus blind spot. tracetest_test.go's own
// TestSpanInterface_MethodCounts is the mitigation: it pins the current
// method counts (Span 11, Tracer 2, TracerProvider 2 — interface NumMethod
// counts the unexported embedded.* marker method too) and fails loudly the
// moment the API grows, forcing a reviewed update instead of a silent gap.
package tracetest

import (
	"bytes"
	"context"
	"fmt"
	"slices"
	"strings"
	"sync"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
	"go.opentelemetry.io/otel/trace/noop"
)

// Provider is a recording trace.TracerProvider: every span started through
// a Tracer it produces is retained in memory, readable back through
// Corpus(), Started(), Ended() and AssertAllEndedOnce.
type Provider struct {
	noop.TracerProvider

	mu    sync.Mutex
	spans []*Span
}

// NewProvider returns a fresh, empty recording Provider.
func NewProvider() *Provider { return &Provider{} }

// Tracer returns a recording Tracer bound to p — every span it starts is
// retained by p and contributes to p.Corpus().
func (p *Provider) Tracer(name string, _ ...trace.TracerOption) trace.Tracer {
	return &tracer{provider: p, name: name}
}

// record appends span to p's retained spans. Called only from tracer.Start.
func (p *Provider) record(span *Span) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.spans = append(p.spans, span)
}

// Started returns the number of spans p has produced across every Tracer
// derived from it.
func (p *Provider) Started() int {
	p.mu.Lock()
	defer p.mu.Unlock()
	return len(p.spans)
}

// Spans returns a copy of every span p has produced, in start order — the
// per-span accessor AI-37's attribute-mapping proof reads (Provider.Corpus
// gives a single flattened denylist-shaped corpus, not per-span structure).
func (p *Provider) Spans() []*Span {
	p.mu.Lock()
	defer p.mu.Unlock()
	return slices.Clone(p.spans)
}

// Ended returns the number of retained spans whose End has been called at
// least once.
func (p *Provider) Ended() int {
	p.mu.Lock()
	defer p.mu.Unlock()
	n := 0
	for _, s := range p.spans {
		if s.EndCount() > 0 {
			n++
		}
	}
	return n
}

// AssertAllEndedOnce reports every started span whose End has not been
// called exactly once (R-AOB-003's exactly-once proof, the span-leak
// analogue of this module's existing goroutine-leak discipline) as one
// formatted error naming each, or nil when every span has ended exactly
// once.
//
// Returning an error rather than taking a *testing.T mirrors this
// package's own no-"testing"-import constraint (see the package comment)
// and the agenttest/sweep precedent (Scan/SelfTest both return values, not
// a T) — the pass/fail call stays the caller's own test's.
func (p *Provider) AssertAllEndedOnce() error {
	p.mu.Lock()
	spans := slices.Clone(p.spans)
	p.mu.Unlock()

	var bad []string
	for i, s := range spans {
		if n := s.EndCount(); n != 1 {
			bad = append(bad, fmt.Sprintf("span %d (name %q) ended %d time(s), want exactly 1", i, s.Name(), n))
		}
	}
	if len(bad) == 0 {
		return nil
	}
	return fmt.Errorf("tracetest: %d span(s) not ended exactly once: %s", len(bad), strings.Join(bad, "; "))
}

// Corpus returns a deterministic serialization of every field the tracing
// API allows a caller to set on any span p has produced: span name, status
// code and description, every attribute key and Value.String(), every
// event name and its own attribute keys/values, every RecordError string,
// and every link's attribute keys/values (R-AOB-007, R-AOB-008 item 4).
//
// Rendering always goes through the tracing API's own String accessor —
// never %#v or %v over a recorder struct (AI-41 lesson, doc 0002:22): a
// canary planted in a pointer-shaped field renders as a bare hex address
// under a structure dump and proves nothing. attribute.Value.String is
// used in place of the deprecated attribute.Value.Emit — both render
// through the API's own accessor rather than a struct dump; String is the
// one the API itself now recommends.
func (p *Provider) Corpus() []byte {
	p.mu.Lock()
	defer p.mu.Unlock()

	var b bytes.Buffer
	for _, s := range p.spans {
		s.writeCorpus(&b)
	}
	return b.Bytes()
}

// tracer is a recording trace.Tracer bound to one Provider.
type tracer struct {
	noop.Tracer

	provider *Provider
	name     string
}

// Start creates a recording Span, retains it on the owning Provider, and
// returns a context carrying it (R-AOB-003: the span starts before any
// caller-visible work).
func (t *tracer) Start(ctx context.Context, spanName string, opts ...trace.SpanStartOption) (context.Context, trace.Span) {
	cfg := trace.NewSpanStartConfig(opts...)
	span := &Span{
		provider: t.provider,
		name:     spanName,
		kind:     cfg.SpanKind(),
		attrs:    slices.Clone(cfg.Attributes()),
	}
	t.provider.record(span)
	return trace.ContextWithSpan(ctx, span), span
}

// recordedEvent is one AddEvent call's retained name and attributes.
type recordedEvent struct {
	name  string
	attrs []attribute.KeyValue
}

// Span is a recording trace.Span: every method a caller can use to attach
// content to a span is overridden to record into mutex-guarded state,
// never falling through to the embedded noop.Span (AD-3's hazard,
// mitigated by tracetest_test.go's AD-4 method-count pin).
type Span struct {
	noop.Span

	provider *Provider
	kind     trace.SpanKind

	mu         sync.Mutex
	name       string
	attrs      []attribute.KeyValue
	events     []recordedEvent
	links      []trace.Link
	errs       []string
	statusCode codes.Code
	statusDesc string
	endCount   int
}

// End records one End call. IsRecording reports false once EndCount() > 0.
func (s *Span) End(_ ...trace.SpanEndOption) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.endCount++
}

// AddEvent records name and every attribute options carries.
func (s *Span) AddEvent(name string, options ...trace.EventOption) {
	cfg := trace.NewEventConfig(options...)
	s.mu.Lock()
	defer s.mu.Unlock()
	s.events = append(s.events, recordedEvent{name: name, attrs: cfg.Attributes()})
}

// AddLink records link, attributes included.
func (s *Span) AddLink(link trace.Link) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.links = append(s.links, link)
}

// IsRecording reports true until End has been called at least once — the
// recorder contract tracetest_test.go pins.
func (s *Span) IsRecording() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.endCount == 0
}

// RecordError renders err.Error() into the corpus at record time (the
// same "an event, not a status change" contract span.go documents: err.Error()
// is captured, but SetStatus is a separate call this method never makes).
// A nil err is a no-op, matching the real API's documented behavior.
func (s *Span) RecordError(err error, _ ...trace.EventOption) {
	if err == nil {
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.errs = append(s.errs, err.Error())
}

// SpanContext returns an empty SpanContext. Nothing in this milestone's
// corpus walk or attribute mapping needs a non-zero one; the method exists
// only so Span satisfies trace.Span.
func (s *Span) SpanContext() trace.SpanContext { return trace.SpanContext{} }

// SetStatus mirrors the real API's own precedence rule (span.go: "provided
// the status hasn't already been set to a higher value before (OK > Error >
// Unset)... description is only included... when the code is for an
// error"): codes.Ok always wins and clears any description; codes.Error is
// retained, with its description, only while the status has not already
// been upgraded to Ok.
func (s *Span) SetStatus(code codes.Code, description string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if code == codes.Ok {
		s.statusCode = codes.Ok
		s.statusDesc = ""
		return
	}
	if code == codes.Error && s.statusCode != codes.Ok {
		s.statusCode = codes.Error
		s.statusDesc = description
	}
}

// SetName records a new span name.
func (s *Span) SetName(name string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.name = name
}

// SetAttributes records kv, overwriting any existing attribute of the same
// key — the real API's own documented overwrite rule (span.go).
func (s *Span) SetAttributes(kv ...attribute.KeyValue) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, attr := range kv {
		replaced := false
		for i, existing := range s.attrs {
			if existing.Key == attr.Key {
				s.attrs[i] = attr
				replaced = true
				break
			}
		}
		if !replaced {
			s.attrs = append(s.attrs, attr)
		}
	}
}

// TracerProvider returns the owning Provider — never nil, matching the
// real API's own contract (span.go:75-77: "a working provider"). This is
// the one method that MUST be overridden rather than inherited from
// noop.Span, which returns a fresh noop.TracerProvider{} instead of the
// span's own owner.
func (s *Span) TracerProvider() trace.TracerProvider { return s.provider }

// EndCount returns how many times End has been called on s.
func (s *Span) EndCount() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.endCount
}

// Name returns s's current name (after any SetName calls) — a test-support
// accessor, not part of trace.Span.
func (s *Span) Name() string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.name
}

// Attributes returns a copy of every attribute SetAttributes has recorded
// on s (including those supplied via trace.WithAttributes at Start,
// tracer.Start above), in recording order — a test-support accessor, not
// part of trace.Span. AI-37's per-key attribute-mapping proof needs typed,
// per-key reads; Corpus() alone gives only a flattened, denylist-shaped
// string.
func (s *Span) Attributes() []attribute.KeyValue {
	s.mu.Lock()
	defer s.mu.Unlock()
	return slices.Clone(s.attrs)
}

// Status returns s's current status code and description — a test-support
// accessor, not part of trace.Span.
func (s *Span) Status() (codes.Code, string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.statusCode, s.statusDesc
}

// writeCorpus appends every field the tracing API lets a caller set on s to
// b, one field per line, each rendered through the API's own String
// accessor.
func (s *Span) writeCorpus(b *bytes.Buffer) {
	s.mu.Lock()
	defer s.mu.Unlock()

	b.WriteString(s.name)
	b.WriteByte('\n')
	b.WriteString(s.statusCode.String())
	b.WriteByte('\n')
	b.WriteString(s.statusDesc)
	b.WriteByte('\n')
	for _, kv := range s.attrs {
		b.WriteString(string(kv.Key))
		b.WriteByte('=')
		b.WriteString(kv.Value.String())
		b.WriteByte('\n')
	}
	for _, ev := range s.events {
		b.WriteString(ev.name)
		b.WriteByte('\n')
		for _, kv := range ev.attrs {
			b.WriteString(string(kv.Key))
			b.WriteByte('=')
			b.WriteString(kv.Value.String())
			b.WriteByte('\n')
		}
	}
	for _, errStr := range s.errs {
		b.WriteString(errStr)
		b.WriteByte('\n')
	}
	for _, link := range s.links {
		for _, kv := range link.Attributes {
			b.WriteString(string(kv.Key))
			b.WriteByte('=')
			b.WriteString(kv.Value.String())
			b.WriteByte('\n')
		}
	}
}
