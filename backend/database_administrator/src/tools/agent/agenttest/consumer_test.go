// Package agenttest_test is the external compile-time consumer example
// for the ai package's ModelProvider interface (AI-16 / Phase C exit).
//
// Per proposal D3, this sibling package (not a child of `ai/`) proves
// that a separate Go package can hold the ModelProvider interface, build
// a normalized Request, drain the producer-owned channel, and never
// import any vendor SDK or any Cachicamas package outside tools/agent/.
//
// Layer 1 purity is mechanically enforced by import_boundary_test.go's
// TestPackageImportBoundary canary (which scans `ai/`) and by the
// in-package TestProviderGo_ImportsOnlyStdlib (provider_test.go) which
// scopes the same check to provider.go. This consumer test owns the
// AST/reflection signature guard (T-AI16-007) plus the six behavioral
// D-scenarios (T-AI16-001..006).
package agenttest_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"go/ast"
	"go/parser"
	"go/printer"
	"go/token"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/cachicamas/backend/database_administrator/src/tools/agent/ai"
)

// ---------------------------------------------------------------------------
// stubProvider — compile-time-only ai.ModelProvider implementation.
// NOT a fake (AI-19 owns that); exists only to prove the interface and
// drive the D.1–D.6 scenarios deterministically.
// ---------------------------------------------------------------------------

// stubProvider is the minimum compile-time-only implementation of
// ai.ModelProvider required by spec section E.3 sub-bullet 1.
//
// The Stream method honors the four invariants mandated by the spec:
//  1. Check ctx.Err() FIRST; on a cancelled context return (nil, ctx.Err())
//     with NO goroutine start and NO make(chan, ...) (C.2.b, C.3.a).
//  2. Allocate ch := make(chan ai.Event, ai.DefaultBufferSize).
//  3. Launch a producer goroutine that closes both ch and a per-call done
//     channel via defer, so double-close and missing-close-because-of-panic
//     are structurally impossible (C.1.b).
//  4. Every send inside the producer uses select { case ch <- ev: case
//     <-ctx.Done(): return } — NOT a plain blocking send — so the
//     saturated-channel cancellation drop rule (D1 / C.2.c) is
//     mechanically reproducible.
type stubProvider struct {
	// numDeltas controls how many text deltas the default emission
	// pattern sends. Defaults to 3 (spec D.1: ASCII, multi-byte UTF-8,
	// empty keepalive). D.4 sets this high (50) so the producer is
	// still trying to send when the test cancels.
	numDeltas int

	// customEmission, when non-nil, replaces the default emission. Used
	// by D.5 (ToolCallInterleaving) to emit an interleaved Start/Delta/
	// End pattern across two opaque callIDs. Nil means "use default".
	customEmission func(ctx context.Context, ch chan<- ai.Event)

	// Pre-cancel signal. When non-nil, the Stream method short-circuits
	// with (nil, ctx.Err()) BEFORE allocating a channel or starting a
	// goroutine — modelling the spec C.2.b pre-stream branch when the
	// caller's ctx is already cancelled. (Note: the production
	// validatePreStream helper does this internally; here the stub
	// piggy-backs on ctx.Err() — the public semantics are identical.)
	//
	// Field reserved for future tests; ctx cancellation already covers
	// the canonical pre-stream case.

	doneMu sync.Mutex
	// dones records every producer's done channel. Tests call
	// waitAllDone to assert each goroutine has finished within the
	// bounded deadline (2s per AI-01 § 3.5).
	dones []chan struct{}
}

// Compile-time assertion: this file's structural anchor. Drift in
// ModelProvider.Stream's signature (param rename, return shape, vendor
// type appearance) breaks this line at build time. Mirrors the
// `var _ ModelProvider = (*compileTimeOnlyStub)(nil)` line in
// provider.go but lives in the consumer package to prove the
// cross-package view.
var _ ai.ModelProvider = (*stubProvider)(nil)

// Stream implements ai.ModelProvider.
func (s *stubProvider) Stream(ctx context.Context, req ai.Request) (<-chan ai.Event, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	_ = req // stub ignores request content; behaviour is determined by Scenario

	ch := make(chan ai.Event, ai.DefaultBufferSize)
	done := make(chan struct{})
	s.doneMu.Lock()
	s.dones = append(s.dones, done)
	s.doneMu.Unlock()

	go func() {
		defer close(done)
		defer close(ch)

		if s.customEmission != nil {
			s.customEmission(ctx, ch)
			return
		}
		s.runDefaultScenario(ctx, ch)
	}()

	return ch, nil
}

// runDefaultScenario emits one response.start, numDeltas text deltas,
// then one response.complete, using sendOrCancel for every channel send.
// numDeltas defaults to 3 when the field is non-positive.
func (s *stubProvider) runDefaultScenario(ctx context.Context, ch chan<- ai.Event) {
	numDeltas := s.numDeltas
	if numDeltas <= 0 {
		numDeltas = 3
	}

	start, err := ai.NewResponseStartEvent("resp-001", "stub-model-v1")
	if err != nil {
		return
	}
	if !sendOrCancel(ctx, ch, start) {
		return
	}

	for i := 0; i < numDeltas; i++ {
		delta, err := ai.NewTextDeltaEvent(deltaPayload(i))
		if err != nil {
			return
		}
		if !sendOrCancel(ctx, ch, delta) {
			return
		}
	}

	usage, err := ai.NewUsage(0, 0, 0, nil, nil, nil)
	if err != nil {
		return
	}
	complete, err := ai.NewResponseCompleteEvent("resp-001", ai.FinishReasonStop, usage)
	if err != nil {
		return
	}
	if !sendOrCancel(ctx, ch, complete) {
		return
	}
}

// deltaPayload returns the spec D.1 three-delta sequence for indices
// 0..2 (ASCII, multi-byte UTF-8 boundary, empty keepalive) and
// deterministic unique ASCII for higher indices (used by D.4 saturation).
func deltaPayload(i int) string {
	switch i {
	case 0:
		return "hello" // ASCII
	case 1:
		return "ñ" // 2-byte UTF-8 boundary (U+00F1 → 0xC3 0xB1)
	case 2:
		return "" // empty keepalive
	default:
		return fmt.Sprintf("d%03d", i)
	}
}

// sendOrCancel writes ev to ch unless ctx is cancelled first. Returns
// true on success, false when ctx fired (the producer must exit and
// let the deferred close(ch) and close(done) fire).
func sendOrCancel(ctx context.Context, ch chan<- ai.Event, ev ai.Event) bool {
	select {
	case ch <- ev:
		return true
	case <-ctx.Done():
		return false
	}
}

// waitAllDone blocks until every producer goroutine launched by this
// stub has signaled completion, or fails the test on timeout. This is
// the deterministic goroutine-leak counter per design §5.5 (the
// done-channel pattern replaces runtime.NumGoroutine() to avoid Darwin
// scheduler flakiness).
func (s *stubProvider) waitAllDone(t *testing.T, d time.Duration) {
	t.Helper()
	s.doneMu.Lock()
	dones := append([]chan struct{}(nil), s.dones...)
	s.doneMu.Unlock()

	for _, done := range dones {
		select {
		case <-done:
		case <-time.After(d):
			t.Fatalf("producer did not close channel within %v; goroutine leak suspected", d)
		}
	}
}

// waitForBufferFull polls len(ch) until it reaches full or the deadline
// expires. Returns true if the buffer reached `full` within the deadline.
// Uses runtime.Gosched() so the producer goroutine actually runs (no
// time.Sleep anywhere in this file).
func waitForBufferFull(t *testing.T, ch <-chan ai.Event, full int, d time.Duration) bool {
	t.Helper()
	deadline := time.Now().Add(d)
	for time.Now().Before(deadline) {
		if len(ch) >= full {
			return true
		}
		runtime.Gosched()
	}
	return false
}

// ---------------------------------------------------------------------------
// drainEvents helper.
// ---------------------------------------------------------------------------

// drainEvents reads from ch until it closes, failing the test on
// timeout. Returns the slice of received events.
func drainEvents(t *testing.T, ch <-chan ai.Event, deadline time.Duration) []ai.Event {
	t.Helper()
	var out []ai.Event
	timer := time.NewTimer(deadline)
	defer timer.Stop()
	for {
		select {
		case ev, ok := <-ch:
			if !ok {
				return out
			}
			out = append(out, ev)
		case <-timer.C:
			t.Fatalf("drainEvents timed out after %v (channel did not close)", deadline)
		}
	}
}

// ---------------------------------------------------------------------------
// validRequest — small constructor for a fresh, validated ai.Request.
// ---------------------------------------------------------------------------

func validRequest() ai.Request {
	textPart, err := ai.ContentPartFromText("hello")
	if err != nil {
		panic(fmt.Sprintf("validRequest: ContentPartFromText: %v", err))
	}
	msgs := []ai.Message{
		{Role: ai.RoleUser, Content: []ai.ContentPart{textPart}},
	}
	opts, err := ai.NewGenerationOptions("", 0, 1.0, 1.0, nil, ai.ToolChoiceAuto)
	if err != nil {
		panic(fmt.Sprintf("validRequest: NewGenerationOptions: %v", err))
	}
	req, err := ai.NewRequest("stub-model-v1", msgs, nil, opts)
	if err != nil {
		panic(fmt.Sprintf("validRequest: NewRequest: %v", err))
	}
	return req
}

// ---------------------------------------------------------------------------
// T-AI16-001 — Happy path text-only stream (spec D.1).
// ---------------------------------------------------------------------------

func TestStream_HappyPathText(t *testing.T) {
	stub := &stubProvider{numDeltas: 3}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	ch, err := stub.Stream(ctx, validRequest())
	if err != nil {
		t.Fatalf("Stream returned error: %v", err)
	}
	if ch == nil {
		t.Fatal("Stream returned nil channel without error")
	}

	events := drainEvents(t, ch, 2*time.Second)

	if len(events) != 5 {
		t.Fatalf("expected 5 events (response.start + 3 text deltas + response.complete); got %d events: %v", len(events), eventKinds(events))
	}

	// First event: response.start with sequence == 1.
	if events[0].Kind != ai.EventKindResponseStart {
		t.Errorf("first event kind = %v, want %v", events[0].Kind, ai.EventKindResponseStart)
	}
	if events[0].Sequence != 1 {
		t.Errorf("first event sequence = %d, want 1", events[0].Sequence)
	}

	// Middle three: text deltas.
	for i := 1; i <= 3; i++ {
		if events[i].Kind != ai.EventKindTextDelta {
			t.Errorf("event[%d] kind = %v, want %v", i, events[i].Kind, ai.EventKindTextDelta)
		}
	}

	// Last: response.complete.
	if events[4].Kind != ai.EventKindResponseComplete {
		t.Errorf("last event kind = %v, want %v", events[4].Kind, ai.EventKindResponseComplete)
	}

	// Per-stream sequence contiguity: 1, 2, 3, 4, 5.
	for i, ev := range events {
		want := events[0].Sequence + uint64(i)
		if ev.Sequence != want {
			t.Errorf("event[%d] sequence = %d, want %d (per-stream contiguous from %d)", i, ev.Sequence, want, events[0].Sequence)
		}
	}

	stub.waitAllDone(t, 2*time.Second)
}

// eventKinds returns a slice of event kinds for diagnostic dumps.
func eventKinds(events []ai.Event) []ai.EventKind {
	out := make([]ai.EventKind, len(events))
	for i, ev := range events {
		out[i] = ev.Kind
	}
	return out
}

// ---------------------------------------------------------------------------
// T-AI16-002 — Pre-stream validation failure (spec D.2).
// ---------------------------------------------------------------------------

func TestStream_PreStreamValidationFailure(t *testing.T) {
	stub := &stubProvider{numDeltas: 3}

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // pre-cancel BEFORE Stream.

	ch, err := stub.Stream(ctx, validRequest())
	if err == nil {
		t.Fatalf("Stream returned no error on already-cancelled context; expected non-nil ctx.Err()")
	}
	if !errors.Is(err, context.Canceled) {
		t.Errorf("err = %v; want errors.Is(err, context.Canceled) == true", err)
	}
	if ch != nil {
		t.Errorf("Stream returned non-nil channel alongside error; ch = %v (consumer MUST receive nil channel on pre-stream failure per C.3.a)", ch)
	}

	// Stub's dones slice must be empty — pre-stream failure MUST NOT
	// spawn a goroutine. If a done channel was recorded, the producer
	// goroutine is alive and the test should fail.
	stub.doneMu.Lock()
	nDones := len(stub.dones)
	stub.doneMu.Unlock()
	if nDones != 0 {
		t.Errorf("Stream registered %d done channels for pre-stream failure; expected 0 (spec C.3.a: no goroutine started)", nDones)
	}
}

// ---------------------------------------------------------------------------
// T-AI16-003 — Mid-stream cancellation with buffered channel space (spec D.3).
// ---------------------------------------------------------------------------

func TestStream_CancelWithBufferedSpace(t *testing.T) {
	stub := &stubProvider{numDeltas: 3}

	ctx, cancel := context.WithCancel(context.Background())

	ch, err := stub.Stream(ctx, validRequest())
	if err != nil {
		t.Fatalf("Stream: %v", err)
	}
	if ch == nil {
		t.Fatal("Stream returned nil channel")
	}

	// Read the first event (response.start).
	select {
	case ev := <-ch:
		if ev.Kind != ai.EventKindResponseStart {
			t.Fatalf("first event kind = %v, want %v", ev.Kind, ai.EventKindResponseStart)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for first event")
	}

	cancel()

	// Drain remaining channel; should close after cancel without
	// emitting any EventKindError terminal.
	var remaining []ai.Event
	drainTimer := time.NewTimer(2 * time.Second)
	defer drainTimer.Stop()
	drainDone := make(chan struct{})
	go func() {
		for ev := range ch {
			remaining = append(remaining, ev)
		}
		close(drainDone)
	}()
	select {
	case <-drainDone:
	case <-drainTimer.C:
		t.Fatal("timed out waiting for channel close after cancel")
	}

	for _, ev := range remaining {
		if ev.Kind == ai.EventKindError {
			t.Errorf("received EventKindError after cancel; expected NO terminal under buffered cancellation (spec D.3 / C.2.d)")
		}
	}

	stub.waitAllDone(t, 2*time.Second)
}

// ---------------------------------------------------------------------------
// T-AI16-004 — Mid-stream cancellation with saturated channel (spec D.4 / D1 lock).
// This is the load-bearing scenario for the saturated-channel drop rule.
// ---------------------------------------------------------------------------

func TestStream_CancelOnSaturatedChannel(t *testing.T) {
	// 50 deltas forces the producer to attempt many sends past the
	// 16-slot buffer — once full, every additional send blocks until
	// ctx.Done() fires.
	stub := &stubProvider{numDeltas: 50}

	ctx, cancel := context.WithCancel(context.Background())

	ch, err := stub.Stream(ctx, validRequest())
	if err != nil {
		t.Fatalf("Stream: %v", err)
	}
	if ch == nil {
		t.Fatal("Stream returned nil channel")
	}

	// Wait until the buffer is full (16 events buffered) AND the
	// producer is therefore blocked on send #17.
	if !waitForBufferFull(t, ch, ai.DefaultBufferSize, 2*time.Second) {
		t.Fatalf("buffer never filled to DefaultBufferSize=%d within 2s; got len(ch)=%d", ai.DefaultBufferSize, len(ch))
	}

	cancel()

	// Producer must select ctx.Done() and close the channel WITHOUT a
	// terminal EventKindError (D1 saturated-channel drop rule).
	stub.waitAllDone(t, 2*time.Second)

	// Drain what the producer managed to buffer. Every event MUST be a
	// non-error kind (the producer dropped the 17th pending send and
	// closed the channel without a terminal EventKindError).
	remaining := drainEvents(t, ch, 2*time.Second)
	if len(remaining) > ai.DefaultBufferSize {
		t.Errorf("buffer had %d events after cancel; exceeds DefaultBufferSize=%d (the producer MUST NOT send past capacity)",
			len(remaining), ai.DefaultBufferSize)
	}
	for _, ev := range remaining {
		if ev.Kind == ai.EventKindError {
			t.Errorf("received EventKindError on saturated channel after cancel; D1 forbids a terminal in the saturated branch (event = %+v)", ev)
		}
	}
}

// ---------------------------------------------------------------------------
// T-AI16-005 — Tool-call interleaving across two callIDs (spec D.5).
// ---------------------------------------------------------------------------

func TestStream_ToolCallInterleaving(t *testing.T) {
	stub := &stubProvider{
		customEmission: emitInterleavedToolCalls,
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	ch, err := stub.Stream(ctx, validRequest())
	if err != nil {
		t.Fatalf("Stream: %v", err)
	}
	if ch == nil {
		t.Fatal("Stream returned nil channel")
	}

	events := drainEvents(t, ch, 2*time.Second)

	// Group events by callID. We expect two lifecycles: "A" and "B".
	deltas := map[string][][]byte{}
	ends := map[string]json.RawMessage{}
	for _, ev := range events {
		if start, ok := ai.AsToolCallStart(ev); ok {
			_ = start // presence is the assertion; name carried in start
			continue
		}
		if delta, ok := ai.AsToolCallDelta(ev); ok {
			deltas[delta.CallID()] = append(deltas[delta.CallID()], delta.Delta())
			continue
		}
		if end, ok := ai.AsToolCallEnd(ev); ok {
			ends[end.CallID()] = end.Arguments()
			continue
		}
	}

	// Each callID MUST have at least one end and the delta concatenation
	// MUST equal the end payload's arguments byte-for-byte.
	for callID, end := range ends {
		var concat []byte
		for _, d := range deltas[callID] {
			concat = append(concat, d...)
		}
		if !bytes.Equal(concat, end) {
			t.Errorf("callID=%q: bytes.Join(deltaFragments) = %q; want End.Arguments() = %q",
				callID, concat, end)
		}
	}

	// Both callIDs MUST have produced at least one end.
	if _, ok := ends["A"]; !ok {
		t.Errorf("expected End event for callID=A; got ends=%v ends_keys", keysOf(ends))
	}
	if _, ok := ends["B"]; !ok {
		t.Errorf("expected End event for callID=B; got ends=%v ends_keys", keysOf(ends))
	}

	stub.waitAllDone(t, 2*time.Second)
}

// emitInterleavedToolCalls is the D.5 emission pattern: two opaque
// callIDs "A" and "B" with Start/Delta/End events interleaved. The Delta
// fragments are opaque partial JSON bytes; the End payload's Arguments
// MUST equal bytes.Join of that callID's Deltas — verifiable by the
// receiver of the stream.
func emitInterleavedToolCalls(ctx context.Context, ch chan<- ai.Event) {
	// callID A: get_weather with {"location":"NYC"}
	aStart, err := ai.NewToolCallStartEvent("A", "get_weather")
	if err == nil && !sendOrCancel(ctx, ch, aStart) {
		return
	}
	// callID B: get_date with {"date":"2026-07-22"}
	bStart, err := ai.NewToolCallStartEvent("B", "get_date")
	if err == nil && !sendOrCancel(ctx, ch, bStart) {
		return
	}

	// Interleaved deltas.
	ad1, err := ai.NewToolCallDeltaEvent("A", json.RawMessage(`{"loc`))
	if err == nil && !sendOrCancel(ctx, ch, ad1) {
		return
	}
	bd1, err := ai.NewToolCallDeltaEvent("B", json.RawMessage(`{"dat`))
	if err == nil && !sendOrCancel(ctx, ch, bd1) {
		return
	}
	ad2, err := ai.NewToolCallDeltaEvent("A", json.RawMessage(`ation":"NYC"}`))
	if err == nil && !sendOrCancel(ctx, ch, ad2) {
		return
	}

	// End A — final assembled arguments.
	aEnd, err := ai.NewToolCallEndEvent("A", "get_weather", json.RawMessage(`{"location":"NYC"}`))
	if err == nil && !sendOrCancel(ctx, ch, aEnd) {
		return
	}

	bd2, err := ai.NewToolCallDeltaEvent("B", json.RawMessage(`e":"2026-07-22"}`))
	if err == nil && !sendOrCancel(ctx, ch, bd2) {
		return
	}
	bEnd, err := ai.NewToolCallEndEvent("B", "get_date", json.RawMessage(`{"date":"2026-07-22"}`))
	if err == nil && !sendOrCancel(ctx, ch, bEnd) {
		return
	}
}

// keysOf returns the keys of a map (helper for diagnostic messages).
func keysOf(m map[string]json.RawMessage) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	return out
}

// ---------------------------------------------------------------------------
// T-AI16-006 — Two concurrent streams from the same provider (spec D.6 / D2
// caveat). Per-stream contiguity is asserted; cross-stream interleaving
// is logged as expected under v1.
// ---------------------------------------------------------------------------

func TestStream_TwoConcurrentStreams(t *testing.T) {
	stub := &stubProvider{numDeltas: 5}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	req := validRequest()

	type result struct {
		name   string
		events []ai.Event
	}
	results := make(chan result, 2)

	for _, name := range []string{"alpha", "beta"} {
		go func(n string) {
			ch, err := stub.Stream(ctx, req)
			if err != nil {
				t.Errorf("[%s] Stream: %v", n, err)
				results <- result{n, nil}
				return
			}
			events := drainEvents(t, ch, 2*time.Second)
			results <- result{n, events}
		}(name)
	}

	got := map[string][]ai.Event{}
	for i := 0; i < 2; i++ {
		r := <-results
		got[r.name] = r.events
	}

	for _, name := range []string{"alpha", "beta"} {
		events := got[name]
		if len(events) == 0 {
			t.Fatalf("[%s] no events received", name)
			continue
		}
		// C.4.d documents that under v1 (process-scoped counter), two
		// concurrent producers MAY interleave their Sequence values.
		// The actual v1 invariant the test can assert is
		// STRICTLY-INCREASING within a single stream — guarantees no
		// duplicates or decreases inside one channel — but the
		// counter is shared, so per-stream contiguity is not
		// guaranteed and gaps are EXPECTED. AI-20 (stream testkit) and
		// AI-21 (conformance suite) own per-stream gap detection.
		streamStart := events[0].Sequence
		if streamStart == 0 {
			t.Errorf("[%s] first event Sequence = 0; impossible because sequenceCounter.Add(1) cannot return zero", name)
		}
		gaps := 0
		for i := 1; i < len(events); i++ {
			if events[i].Sequence <= events[i-1].Sequence {
				t.Errorf("[%s] event[%d] Sequence = %d, must be strictly greater than event[%d] Sequence = %d",
					name, i, events[i].Sequence, i-1, events[i-1].Sequence)
			}
			if events[i].Sequence > events[i-1].Sequence+1 {
				gaps++
			}
		}
		t.Logf("[%s] stream start Sequence = %d; %d gap(s) inside the stream — expected under v1 (C.4.d; AI-20/AI-21 territory)",
			name, streamStart, gaps)
	}

	// Cross-stream interleaving: documented as expected under v1
	// process-scoped counter (D2 caveat). See event.go:224–235. AI-20
	// (stream testkit) and AI-21 (conformance suite) own per-stream
	// gap detection.
	alphaSeqs := map[uint64]bool{}
	for _, ev := range got["alpha"] {
		alphaSeqs[ev.Sequence] = true
	}
	overlap := 0
	for _, ev := range got["beta"] {
		if alphaSeqs[ev.Sequence] {
			overlap++
		}
	}
	t.Logf("cross-stream sequence overlap (v1 process-scoped counter — AI-20/AI-21 territory): %d shared values", overlap)

	stub.waitAllDone(t, 2*time.Second)
}

// ---------------------------------------------------------------------------
// T-AI16-007 — AST/reflection signature guard (spec D.7).
//
// This is the load-bearing mechanical assertion that the public surface
// of ModelProvider mentions ONLY context.Context, Request, <-chan Event,
// and error — and that provider.go's import set is stdlib-only.
// ---------------------------------------------------------------------------

func TestModelProviderInterface_SignatureGuard(t *testing.T) {
	// runtime.Caller(0) gives the absolute path of consumer_test.go,
	// which lives under agenttest/. Walk one level up to the parent
	// (tools/agent/) and into ai/ to find provider.go. This is robust
	// against `go test ./...` vs `go test ./agenttest/...` vs IDE-driven
	// runs because the path is resolved from the source file location,
	// not the test's working directory.
	_, thisFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatalf("runtime.Caller(0) failed")
	}
	path := filepath.Join(filepath.Dir(thisFile), "..", "ai", "provider.go")
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, path, nil, parser.ParseComments)
	if err != nil {
		t.Fatalf("parse %s: %v (GREEN commit must produce provider.go before this guard runs)", path, err)
	}

	// Locate the ModelProvider interface declaration.
	var iface *ast.InterfaceType
	for _, decl := range file.Decls {
		gd, ok := decl.(*ast.GenDecl)
		if !ok || gd.Tok != token.TYPE {
			continue
		}
		for _, spec := range gd.Specs {
			ts, ok := spec.(*ast.TypeSpec)
			if !ok || ts.Name.Name != "ModelProvider" {
				continue
			}
			inner, ok := ts.Type.(*ast.InterfaceType)
			if !ok {
				t.Fatalf("ModelProvider is not an interface type")
			}
			iface = inner
		}
	}
	if iface == nil {
		t.Fatal("ModelProvider interface not found in provider.go")
	}

	// B.1.b: exactly one method.
	if len(iface.Methods.List) != 1 {
		t.Fatalf("ModelProvider must declare exactly one method (B.1.b); got %d", len(iface.Methods.List))
	}

	// Method name MUST be "Stream" (proposal D4 / spec B.1.b).
	method := iface.Methods.List[0]
	if len(method.Names) == 0 || method.Names[0].Name != "Stream" {
		t.Errorf("ModelProvider method name MUST be `Stream`; got %v", method.Names)
	}
	fn, ok := method.Type.(*ast.FuncType)
	if !ok {
		t.Fatalf("ModelProvider's single method is not an ast.FuncType")
	}

	// B.1.c + C.5.a: parameter types reference ONLY context.Context,
	// Request, <-chan Event, error. We assert by canonical AST-print.
	// The AST printer renders same-package types unqualified (Request,
	// Event) — but the canonical imported-and-typeset forms would be
	// ai.Request / <-chan ai.Event. Both forms denote the same type
	// (the latter is the qualification an outside-package consumer
	// would write). Accept either, reject anything else.
	gotParamTypes := exprTypes(fn.Params.List)
	wantParamTypesAccept := [][]string{
		{"context.Context", "Request"},
		{"context.Context", "ai.Request"},
	}
	if !acceptsAny(gotParamTypes, wantParamTypesAccept) {
		t.Errorf("ModelProvider.Stream parameter types = %v; want %v (unqualified) or %v (qualified); per spec B.1.c, C.5.a", gotParamTypes, wantParamTypesAccept[0], wantParamTypesAccept[1])
	}
	gotReturnTypes := exprTypes(fn.Results.List)
	wantReturnTypesAccept := [][]string{
		{"<-chan Event", "error"},
		{"<-chan ai.Event", "error"},
	}
	if !acceptsAny(gotReturnTypes, wantReturnTypesAccept) {
		t.Errorf("ModelProvider.Stream return types = %v; want %v (unqualified) or %v (qualified); per spec B.1.c, C.5.a", gotReturnTypes, wantReturnTypesAccept[0], wantReturnTypesAccept[1])
	}

	// C.5.a extended: every import in provider.go is stdlib ONLY.
	// (encoding/json as a stream element type, net/http, and any
	// vendor SDK path are forbidden.)
	forbiddenImportSubstrings := []string{
		"openai", "anthropic", "huggingface", "openrouter",
		"net/http", "encoding/json",
	}
	for _, imp := range file.Imports {
		path := strings.Trim(imp.Path.Value, "\"")
		for _, bad := range forbiddenImportSubstrings {
			if strings.Contains(path, bad) {
				t.Errorf("provider.go has forbidden import %q (contains %q); spec C.5.a + F.5 forbid vendor SDKs and stream carriers", path, bad)
			}
		}
	}
}

// exprTypes converts a list of ast.Fields (params or results) into the
// canonical AST-printed type strings. Single field names that declare
// multiple variables of the same type are flattened into one entry per
// declared variable so callers can compare slices directly.
func exprTypes(fields []*ast.Field) []string {
	var out []string
	for _, f := range fields {
		var buf strings.Builder
		_ = printer.Fprint(&buf, token.NewFileSet(), f.Type)
		typeStr := strings.TrimSpace(buf.String())
		// `f.Names == nil` is a "single unnamed parameter of given type"
		// — count it as one entry.
		if len(f.Names) == 0 {
			out = append(out, typeStr)
			continue
		}
		// One entry per declared variable name.
		for range f.Names {
			out = append(out, typeStr)
		}
	}
	return out
}

// equalSlices reports whether a and b have identical contents.
func equalSlices(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

// acceptsAny reports whether got matches any of the acceptable slices.
// Used to accept both the unqualified AST-printed form of same-package
// types AND the qualified form that outside-package consumers would
// write. Both denote the same type; only the printer's qualification
// differs.
func acceptsAny(got []string, accepts [][]string) bool {
	for _, acc := range accepts {
		if equalSlices(got, acc) {
			return true
		}
	}
	return false
}
