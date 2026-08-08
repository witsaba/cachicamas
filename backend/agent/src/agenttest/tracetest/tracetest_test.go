// AI-37 (WU-7, D-2/AD-3/AD-4) — the hand-rolled recording tracer this
// milestone's every later proof depends on: R-AOB-003 (span lifecycle),
// R-AOB-004 (attribute mapping), R-AOB-007/R-AOB-008 (denylist absence +
// non-vacuity). A test-only OTel SDK is genuinely blocked at Layer 1
// (proposal D-2; import_boundary_test.go's forbiddenPrefixes carries
// go.opentelemetry.io/otel/sdk, checked before the allowlist), so this
// package hand-rolls a minimal recording TracerProvider/Tracer/Span
// against the API interfaces instead of importing the SDK's own fake.
//
// External test package (tracetest_test): every assertion below drives
// tracetest through its exported surface only — the same external-
// consumer-proof posture ADR 0005 § D2 assigns to agenttest as a whole.
package tracetest_test

import (
	"context"
	"errors"
	"reflect"
	"strings"
	"testing"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"

	"github.com/cachicamas/backend/agent/src/agenttest/tracetest"
)

// TestProvider_StartRecordsEveryObservableMethod covers the basic
// recording contract: every Span method a caller can use to attach
// content is observable back through Corpus(), and TracerProvider()/
// IsRecording() behave per the documented recorder contract (D-2, AD-3).
func TestProvider_StartRecordsEveryObservableMethod(t *testing.T) {
	t.Parallel()

	p := tracetest.NewProvider()
	tracer := p.Tracer("test-tracer")
	ctx, span := tracer.Start(context.Background(), "test-span",
		trace.WithAttributes(attribute.String("start.key", "start-value-SENTINEL")))

	if !span.IsRecording() {
		t.Fatal("IsRecording() = false immediately after Start, want true (recorder contract: true until End)")
	}
	if span.TracerProvider() == nil {
		t.Fatal("TracerProvider() = nil, want the owning provider (span.go:75-77 — a working provider, never nil)")
	}
	if trace.SpanFromContext(ctx) != span {
		t.Fatal("Start's returned context does not carry the same span value")
	}

	span.SetName("renamed-span-SENTINEL")
	span.SetAttributes(attribute.String("attr.key", "attr-value-SENTINEL"))
	span.AddEvent("event-name-SENTINEL", trace.WithAttributes(attribute.String("event.attr.key", "event-attr-value-SENTINEL")))
	span.AddLink(trace.Link{Attributes: []attribute.KeyValue{attribute.String("link.attr.key", "link-attr-value-SENTINEL")}})
	span.RecordError(errors.New("recorded-error-SENTINEL"))
	span.SetStatus(codes.Error, "status-desc-SENTINEL")

	corpus := string(p.Corpus())
	for _, want := range []string{
		"start-value-SENTINEL",
		"renamed-span-SENTINEL",
		"attr-value-SENTINEL",
		"event-name-SENTINEL",
		"event-attr-value-SENTINEL",
		"link-attr-value-SENTINEL",
		"recorded-error-SENTINEL",
		"status-desc-SENTINEL",
	} {
		if !strings.Contains(corpus, want) {
			t.Errorf("Corpus() does not contain %q — a recorder method's effect is not walked by Corpus()", want)
		}
	}

	span.End()
	if span.IsRecording() {
		t.Error("IsRecording() = true after End, want false")
	}
}

// TestProvider_EndIncrementsEndCount_AssertAllEndedOnce covers the
// exactly-once proof (R-AOB-003): AssertAllEndedOnce reports nil when
// every started span has ended exactly once.
func TestProvider_EndIncrementsEndCount_AssertAllEndedOnce(t *testing.T) {
	t.Parallel()

	p := tracetest.NewProvider()
	tracer := p.Tracer("test-tracer")

	_, spanA := tracer.Start(context.Background(), "span-a")
	_, spanB := tracer.Start(context.Background(), "span-b")
	spanA.End()
	spanB.End()

	if err := p.AssertAllEndedOnce(); err != nil {
		t.Errorf("AssertAllEndedOnce() = %v, want nil on two properly-ended spans", err)
	}
	if got := p.Started(); got != 2 {
		t.Errorf("Started() = %d, want 2", got)
	}
	if got := p.Ended(); got != 2 {
		t.Errorf("Ended() = %d, want 2", got)
	}
}

// TestProvider_AssertAllEndedOnce_DetectsUnendedSpan is the mutation-style
// control for the assertion above: a started, never-ended span must make
// AssertAllEndedOnce report a non-nil error naming it — the assertion
// detects a leak rather than assuming closure.
func TestProvider_AssertAllEndedOnce_DetectsUnendedSpan(t *testing.T) {
	t.Parallel()

	p := tracetest.NewProvider()
	tracer := p.Tracer("test-tracer")
	_, span := tracer.Start(context.Background(), "leaked-span-SENTINEL")
	_ = span // deliberately never ended

	err := p.AssertAllEndedOnce()
	if err == nil {
		t.Fatal("AssertAllEndedOnce() = nil, want a non-nil error naming the unended span")
	}
	if !strings.Contains(err.Error(), "leaked-span-SENTINEL") {
		t.Errorf("AssertAllEndedOnce() error = %v, want it to name the leaked span", err)
	}
}

// TestProvider_AssertAllEndedOnce_DetectsDoubleEnd is the double-end half
// of the same control: End called twice on one span must also make
// AssertAllEndedOnce report a non-nil error.
func TestProvider_AssertAllEndedOnce_DetectsDoubleEnd(t *testing.T) {
	t.Parallel()

	p := tracetest.NewProvider()
	tracer := p.Tracer("test-tracer")
	_, span := tracer.Start(context.Background(), "double-ended-span")
	span.End()
	span.End()

	if err := p.AssertAllEndedOnce(); err == nil {
		t.Fatal("AssertAllEndedOnce() = nil, want a non-nil error naming the double-ended span")
	}
}

// TestCorpus_EmptyProviderReportsEmptyCorpus is the negative control the
// corpus-non-empty guard (R-AOB-008 item 2) depends on: a provider with no
// started spans produces an empty corpus, so a real run's non-empty corpus
// is evidence of something, not a permanently-true tautology.
func TestCorpus_EmptyProviderReportsEmptyCorpus(t *testing.T) {
	t.Parallel()

	p := tracetest.NewProvider()
	if got := p.Corpus(); len(got) != 0 {
		t.Errorf("Corpus() on an empty provider = %d bytes, want 0", len(got))
	}
	if got := p.Started(); got != 0 {
		t.Errorf("Started() on an empty provider = %d, want 0", got)
	}
}

// TestSpanInterface_MethodCounts is AD-4's pin: a version bump that adds a
// method to trace.Span, trace.Tracer or trace.TracerProvider must fail this
// test loudly rather than silently no-oping through the embedded noop
// fallback (AD-3's hazard). NumMethod on an interface type counts
// unexported methods too — trace.Span's embedded.Span contributes one
// (span()), which is why the count is 11, not 10.
func TestSpanInterface_MethodCounts(t *testing.T) {
	t.Parallel()

	if got := reflect.TypeOf((*trace.Span)(nil)).Elem().NumMethod(); got != 11 {
		t.Errorf("trace.Span interface has %d methods, want 11 (10 exported + embedded.Span's unexported span()) — the recording Span must be updated to match (AD-4)", got)
	}
	if got := reflect.TypeOf((*trace.Tracer)(nil)).Elem().NumMethod(); got != 2 {
		t.Errorf("trace.Tracer interface has %d methods, want 2 (AD-4)", got)
	}
	if got := reflect.TypeOf((*trace.TracerProvider)(nil)).Elem().NumMethod(); got != 2 {
		t.Errorf("trace.TracerProvider interface has %d methods, want 2 (AD-4)", got)
	}
}

// spanContentBearingMethods enumerates the trace.Span methods whose
// arguments carry caller-supplied string-shaped content a leak could
// travel through, and the ones that carry none. A version bump that grows
// the interface (AD-4's method-count pin, above) forces a reviewer to
// revisit this table and decide which side the new method belongs to —
// this table is what turns that decision into a review diff instead of a
// silent gap in Corpus()'s walk (risk 7.6, S-AOB-034).
var spanContentBearingMethods = map[string]bool{
	"End":            false, // no SpanEndOption in this API version carries attributes (config.go: attributeOption implements SpanStartEventOption, not SpanEndEventOption)
	"AddEvent":       true,  // event name + event attributes
	"AddLink":        true,  // link attributes
	"IsRecording":    false, // no caller-supplied value
	"RecordError":    true,  // err.Error() text
	"SpanContext":    false, // no caller-supplied value
	"SetStatus":      true,  // status description
	"SetName":        true,  // span name
	"SetAttributes":  true,  // attribute key/value pairs
	"TracerProvider": false, // no caller-supplied value
}

// TestCorpus_CapturedFieldCountMatchesAPISurface covers S-AOB-034: the
// number of trace.Span methods Corpus() walks the effect of equals the
// number the API allows a caller to use to attach content to a span — a
// newly settable field cannot be recorded and then skipped by the corpus
// builder.
func TestCorpus_CapturedFieldCountMatchesAPISurface(t *testing.T) {
	t.Parallel()

	spanType := reflect.TypeOf((*trace.Span)(nil)).Elem()
	if got := spanType.NumMethod() - 1; got != len(spanContentBearingMethods) { // -1: embedded.Span's unexported span()
		t.Fatalf("trace.Span has %d exported methods, but spanContentBearingMethods enumerates %d — update the table (AD-4/S-AOB-034)", got, len(spanContentBearingMethods))
	}
	wantContentBearing := 0
	for _, carries := range spanContentBearingMethods {
		if carries {
			wantContentBearing++
		}
	}

	// Behavioral proof, not just the static table above: drive one span
	// through every content-bearing method with a distinct sentinel per
	// method, and confirm Corpus() contains all of them.
	p := tracetest.NewProvider()
	_, span := p.Tracer("t").Start(context.Background(), "span")
	sentinels := map[string]string{
		"AddEvent":      "CFC-ADDEVENT-SENTINEL",
		"AddLink":       "CFC-ADDLINK-SENTINEL",
		"RecordError":   "CFC-RECORDERROR-SENTINEL",
		"SetStatus":     "CFC-SETSTATUS-SENTINEL",
		"SetName":       "CFC-SETNAME-SENTINEL",
		"SetAttributes": "CFC-SETATTRIBUTES-SENTINEL",
	}
	span.AddEvent(sentinels["AddEvent"])
	span.AddLink(trace.Link{Attributes: []attribute.KeyValue{attribute.String("k", sentinels["AddLink"])}})
	span.RecordError(errors.New(sentinels["RecordError"]))
	span.SetStatus(codes.Error, sentinels["SetStatus"])
	span.SetName(sentinels["SetName"])
	span.SetAttributes(attribute.String("k", sentinels["SetAttributes"]))

	corpus := string(p.Corpus())
	captured := 0
	for method, sentinel := range sentinels {
		if strings.Contains(corpus, sentinel) {
			captured++
		} else {
			t.Errorf("Corpus() does not contain the %s sentinel %q", method, sentinel)
		}
	}
	if captured != wantContentBearing {
		t.Errorf("Corpus() captured %d of the %d content-bearing methods' effects, want all %d (S-AOB-034)", captured, wantContentBearing, wantContentBearing)
	}
}

// TestProvider_TwoTracersFromOneProvider_ShareTheSameCorpus covers a
// property AI-37.2's per-attribute tests will lean on: two Tracer values
// obtained from the same Provider (as openaicompat.New would derive once
// and later request tracer(s) from) both feed the one Provider's Corpus.
func TestProvider_TwoTracersFromOneProvider_ShareTheSameCorpus(t *testing.T) {
	t.Parallel()

	p := tracetest.NewProvider()
	tracerA := p.Tracer("scope-a")
	tracerB := p.Tracer("scope-b")

	_, spanA := tracerA.Start(context.Background(), "span-from-a")
	_, spanB := tracerB.Start(context.Background(), "span-from-b")
	spanA.End()
	spanB.End()

	if got := p.Started(); got != 2 {
		t.Errorf("Started() = %d, want 2 (both tracers feed the same provider)", got)
	}
	corpus := string(p.Corpus())
	if !strings.Contains(corpus, "span-from-a") || !strings.Contains(corpus, "span-from-b") {
		t.Errorf("Corpus() = %q, want both span names present", corpus)
	}
}
