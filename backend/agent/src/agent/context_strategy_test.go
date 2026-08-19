// AG-17.1 — strict-TDD tests for the context strategy seam
// (R-CTX-001..005, S-CTX-001..010, bites S-CTX-003/008).
//
// Every scenario lives in package agent_test (NFR-CTX-001): the
// external-test posture proves every behavioral claim from outside the
// package, with no reach into unexported surface.
package agent_test

import (
	"context"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"reflect"
	"strings"
	"sync"
	"testing"

	"github.com/cachicamas/backend/agent/src/agent"
	"github.com/cachicamas/backend/agent/src/agenttest"
	"github.com/cachicamas/backend/agent/src/ai"
)

// TestContextVerdict_HasNoFields — S-CTX-005, charter AG-17.1 scenario
// 2, the type half. The shipped verdict type reports zero fields when
// inspected reflectively, and carries no exported method and no
// constructor function that could produce a value distinguishable
// from its zero value: the only value NoOpContextStrategy.Resolve can
// ever return IS the zero value, because no other value exists to
// return (R-CTX-003).
func TestContextVerdict_HasNoFields(t *testing.T) {
	t.Parallel()

	typ := reflect.TypeOf(agent.ContextVerdict{})
	if typ.NumField() != 0 {
		t.Fatalf("agent.ContextVerdict has %d field(s), want 0 (R-CTX-003: no field can request compaction)", typ.NumField())
	}
	if typ.NumMethod() != 0 {
		t.Errorf("agent.ContextVerdict (value type) declares %d exported method(s), want 0 -- a method could still manufacture a distinguishable value", typ.NumMethod())
	}
	ptrTyp := reflect.TypeOf(&agent.ContextVerdict{})
	if ptrTyp.NumMethod() != 0 {
		t.Errorf("*agent.ContextVerdict declares %d exported method(s), want 0", ptrTyp.NumMethod())
	}

	var zero agent.ContextVerdict
	verdict := agent.NoOpContextStrategy{}.Resolve(context.Background(), agent.ContextPrompt{})
	if verdict != zero {
		t.Errorf("NoOpContextStrategy.Resolve() = %#v, want the zero ContextVerdict", verdict)
	}
}

// TestContextBudget_AbsentVsStatedZero — S-CTX-009. The zero
// ContextBudget reads as absence through the two-result accessor;
// ContextBudgetOf(0) reads a limit of zero together with presence;
// the two values are NOT equal -- absence and a stated nought are
// distinguishable, exactly as ai.Tokens(0) is distinguishable from
// ai.TokenCount{} (ai/usage.go:34-47).
func TestContextBudget_AbsentVsStatedZero(t *testing.T) {
	t.Parallel()

	var zero agent.ContextBudget
	if limit, present := zero.Limit(); present || limit != 0 {
		t.Errorf("zero ContextBudget.Limit() = (%d, %v), want (0, false) -- the zero value is absence, not a budget of zero tokens", limit, present)
	}

	statedZero := agent.ContextBudgetOf(0)
	limit, present := statedZero.Limit()
	if !present || limit != 0 {
		t.Errorf("ContextBudgetOf(0).Limit() = (%d, %v), want (0, true) -- a stated zero is present", limit, present)
	}

	if zero == statedZero {
		t.Error("the zero ContextBudget equals ContextBudgetOf(0); absence and a stated nought MUST be distinguishable values")
	}
}

// TestContextBudget_NegativeIsAbsentTotal — S-CTX-010, first half.
// ContextBudgetOf is total: a negative input yields the absent zero
// value, with no error and no panic. Triangulated across several
// negative inputs, including the int64 minimum, so a Fake-It
// implementation that special-cases -1 cannot pass.
func TestContextBudget_NegativeIsAbsentTotal(t *testing.T) {
	t.Parallel()

	var zero agent.ContextBudget

	for _, n := range []int64{-1, -2, -1000, -9223372036854775808} {
		got := agent.ContextBudgetOf(n)
		if limit, present := got.Limit(); present || limit != 0 {
			t.Errorf("ContextBudgetOf(%d).Limit() = (%d, %v), want (0, false) -- a negative limit is not a budget", n, limit, present)
		}
		if got != zero {
			t.Errorf("ContextBudgetOf(%d) = %#v, want the zero ContextBudget (%#v)", n, got, zero)
		}
	}
}

// TestTokenAccounting_UnreadableWithoutSource — S-CTX-014, the
// distinguishability half. The accounting type exposes no exported
// field and exactly ONE exported method -- Tokens() (int64,
// TokenSource) -- so the figure is physically unreadable without its
// provenance in the same call. The NumOut()==2 assertion is the
// load-bearing check bite S-CTX-015 defeats: a bare single-result
// Tokens() int64 declared BESIDE this one would push NumMethod to 2,
// and removing the two-result accessor entirely is a compile failure
// at every call site below -- both are valid RED evidence for the
// bite.
func TestTokenAccounting_UnreadableWithoutSource(t *testing.T) {
	t.Parallel()

	typ := reflect.TypeOf(agent.TokenAccounting{})
	for i := range typ.NumField() {
		if f := typ.Field(i); f.IsExported() {
			t.Fatalf("agent.TokenAccounting exports field %q, want zero exported fields (R-CTX-008)", f.Name)
		}
	}
	if typ.NumMethod() != 1 {
		t.Fatalf("agent.TokenAccounting declares %d exported method(s), want exactly 1 (Tokens() (int64, TokenSource)) -- a second accessor would let a consumer bypass the provenance", typ.NumMethod())
	}
	method := typ.Method(0)
	if method.Name != "Tokens" {
		t.Fatalf("agent.TokenAccounting's one method is named %q, want %q", method.Name, "Tokens")
	}
	if got := method.Type.NumOut(); got != 2 {
		t.Fatalf("Tokens() declares %d result(s), want exactly 2 (figure, source) -- a single-result accessor would let a consumer read the figure without meeting its provenance", got)
	}

	var zero agent.TokenAccounting
	tokens, source := zero.Tokens()
	if tokens != 0 || source != agent.TokenSourceUnavailable {
		t.Errorf("zero TokenAccounting.Tokens() = (%d, %v), want (0, TokenSourceUnavailable) -- the zero value reads as no figure, never as 0 tokens", tokens, source)
	}

	rendered := map[string]bool{}
	for _, s := range []agent.TokenSource{agent.TokenSourceUnavailable, agent.TokenSourceReported, agent.TokenSourceEstimated} {
		rendered[s.String()] = true
	}
	if len(rendered) != 3 {
		t.Errorf("the three TokenSource values render to %d distinct string(s), want 3", len(rendered))
	}
}

// --- Phase 5 -- the seam consultation (R-CTX-001..004) --------------------

// recordingContextStrategy is a test-local agent.ContextStrategy that
// records every Resolve call's prompt, in order, and always returns
// the zero ContextVerdict (v1's only constructible value). requestsFn,
// when set, snapshots how many requests the provider has recorded at
// the MOMENT each consultation runs -- the mechanical proof that a
// consultation happens BEFORE that turn's own first provider call,
// since Resolve executes synchronously ahead of Turn's own
// provider.Stream call (S-CTX-001).
type recordingContextStrategy struct {
	mu         sync.Mutex
	prompts    []agent.ContextPrompt
	requestsFn func() int
	countsAt   []int
}

func (s *recordingContextStrategy) Resolve(_ context.Context, prompt agent.ContextPrompt) agent.ContextVerdict {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.prompts = append(s.prompts, prompt)
	if s.requestsFn != nil {
		s.countsAt = append(s.countsAt, s.requestsFn())
	}
	return agent.ContextVerdict{}
}

func (s *recordingContextStrategy) Prompts() []agent.ContextPrompt {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make([]agent.ContextPrompt, len(s.prompts))
	copy(out, s.prompts)
	return out
}

func (s *recordingContextStrategy) CountsAtConsultation() []int {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make([]int, len(s.countsAt))
	copy(out, s.countsAt)
	return out
}

var _ agent.ContextStrategy = (*recordingContextStrategy)(nil)

// mutatingContextStrategy appends to and overwrites the Transcript
// slice it receives -- proving the seam hands out a fresh CLONE, not
// the harness's own slice (R-CTX-002): a mutation here must never
// reach what the turn actually sends.
type mutatingContextStrategy struct{}

func (mutatingContextStrategy) Resolve(_ context.Context, prompt agent.ContextPrompt) agent.ContextVerdict {
	for i := range prompt.Transcript {
		prompt.Transcript[i] = ai.Message{}
	}
	_ = append(prompt.Transcript, ai.Message{})
	return agent.ContextVerdict{}
}

var _ agent.ContextStrategy = mutatingContextStrategy{}

// isCompactionKind reports whether k is one of the compaction-family
// kinds AG-06 minted (compaction_events.go) — AG-17 must emit none of
// them on any path (R-CTX-003, R-CTX-012).
func isCompactionKind(k agent.EventKind) bool {
	switch k {
	case agent.EventKindCompactionStarted, agent.EventKindCompactionFinished, agent.EventKindCompactionFailed:
		return true
	default:
		return false
	}
}

// assertEventStreamsStructurallyIdentical asserts a and b carry the
// same event count, the same Kind() at every position, the same
// turn-scoping, and — for every event kind this file's scripts
// produce — the same payload content (R-CTX-004's byte-identity pin).
//
// RunID/TurnID VALUES are the one deliberate exclusion: they are
// freshly minted per Harness.Run call by design (R-RUN-004's
// provenance-distinctness), so two INDEPENDENT runs can never share
// them even when identical in every other observable respect. Every
// other field this file's scripts populate is compared.
func assertEventStreamsStructurallyIdentical(t *testing.T, a, b []agent.Event) {
	t.Helper()

	if len(a) != len(b) {
		t.Fatalf("event count differs: %d vs %d, want equal", len(a), len(b))
	}
	for i := range a {
		ea, eb := a[i], b[i]
		if ea.Kind() != eb.Kind() {
			t.Errorf("event[%d].Kind() = %v, want %v", i, ea.Kind(), eb.Kind())
			continue
		}
		_, aHasTurn := ea.Turn()
		_, bHasTurn := eb.Turn()
		if aHasTurn != bHasTurn {
			t.Errorf("event[%d] turn-scoped: %v vs %v, want equal", i, aHasTurn, bHasTurn)
		}

		switch ea.Kind() {
		case agent.EventKindRunEnd:
			ra, _ := ea.RunEnd()
			rb, _ := eb.RunEnd()
			if ra.Outcome() != rb.Outcome() {
				t.Errorf("event[%d] run_end outcome: %v vs %v", i, ra.Outcome(), rb.Outcome())
			}
			fa, hasA := ra.Failure()
			fb, hasB := rb.Failure()
			if hasA != hasB {
				t.Errorf("event[%d] run_end failure presence: %v vs %v", i, hasA, hasB)
			}
			if hasA && hasB && fa.Category() != fb.Category() {
				t.Errorf("event[%d] run_end failure category: %v vs %v", i, fa.Category(), fb.Category())
			}
		case agent.EventKindTurnEnd:
			ta, _ := ea.TurnEnd()
			tb, _ := eb.TurnEnd()
			if ta.Outcome() != tb.Outcome() {
				t.Errorf("event[%d] turn_end outcome: %v vs %v", i, ta.Outcome(), tb.Outcome())
			}
			fa, hasA := ta.Failure()
			fb, hasB := tb.Failure()
			if hasA != hasB {
				t.Errorf("event[%d] turn_end failure presence: %v vs %v", i, hasA, hasB)
			}
			if hasA && hasB && (fa.Category() != fb.Category() || fa.PartialOutput() != fb.PartialOutput()) {
				t.Errorf("event[%d] turn_end failure evidence mismatch", i)
			}
		case agent.EventKindToolStart:
			ta, _ := ea.ToolStart()
			tb, _ := eb.ToolStart()
			if ta.Name() != tb.Name() || string(ta.Arguments()) != string(tb.Arguments()) || ta.Ordinal() != tb.Ordinal() {
				t.Errorf("event[%d] tool_start content mismatch: %+v vs %+v", i, ta, tb)
			}
		case agent.EventKindToolEndSuccess:
			ta, _ := ea.ToolEndSuccess()
			tb, _ := eb.ToolEndSuccess()
			if string(ta.Result()) != string(tb.Result()) || ta.Ordinal() != tb.Ordinal() {
				t.Errorf("event[%d] tool_end_success content mismatch: %+v vs %+v", i, ta, tb)
			}
		case agent.EventKindMessageDeltaText:
			da, _ := ea.MessageDeltaText()
			db, _ := eb.MessageDeltaText()
			if da.Fragment() != db.Fragment() || da.Index() != db.Index() {
				t.Errorf("event[%d] message_delta_text content mismatch: %+v vs %+v", i, da, db)
			}
		}
	}
}

// scriptedTwoTurnHarness builds the canonical tool-call-then-text
// two-logical-turn harness this file's AG-17.1 tests share (mirrors
// TestHarness_TwoTurnRun_CompletesToTerminal's shape): turn one ends
// in a tool call, turn two completes with plain text.
func scriptedTwoTurnHarness(t *testing.T, toolAndCallSuffix string, strategy agent.ContextStrategy, budget agent.ContextBudget) (*agent.Harness, *agenttest.Provider) {
	t.Helper()

	toolName := "read_ctx_" + toolAndCallSuffix
	callID := "call-ctx-" + toolAndCallSuffix
	tool := EchoScriptedTool(toolName, agent.EffectClassRead)
	reg := agent.NewMapRegistry(map[string]agent.Tool{toolName: tool})

	provider := agenttest.NewProvider(
		scriptToolCallResponse(t, callID, toolName, []byte(`{"x":1}`)),
		scriptTextResponse(t, ai.FinishReasonStop),
	)
	h := &agent.Harness{
		Provider:        provider,
		System:          "system prompt for context strategy seam test " + toolAndCallSuffix,
		Turn:            agent.TurnOptions{Tools: reg},
		ContextStrategy: strategy,
		ContextBudget:   budget,
	}
	return h, provider
}

// TestHarness_ContextStrategy_ConsultedOncePerLogicalTurn — S-CTX-001,
// charter AG-17.1 scenario 1, clean half. A run of two logical turns,
// each completing on its first attempt, consults the recording
// strategy EXACTLY twice, each consultation strictly before that
// turn's own first provider call; and the run's outcome, returned
// error and event stream match the identical script driven with no
// strategy installed.
func TestHarness_ContextStrategy_ConsultedOncePerLogicalTurn(t *testing.T) {
	t.Parallel()

	h, provider := scriptedTwoTurnHarness(t, "001", nil, agent.ContextBudget{})
	strategy := &recordingContextStrategy{requestsFn: func() int { return len(provider.Requests()) }}
	h.ContextStrategy = strategy

	sink := make(chan *agent.Event, 256)
	_, finish, err := h.Run(contextBackground(), firstMessage(t), sink)
	if err != nil {
		t.Fatalf("Run returned err = %v, want nil", err)
	}
	events := drainSink(t, sink)

	prompts := strategy.Prompts()
	if len(prompts) != 2 {
		t.Fatalf("recorder holds %d consultation(s), want exactly 2", len(prompts))
	}
	if counts := strategy.CountsAtConsultation(); len(counts) != 2 || counts[0] != 0 || counts[1] != 1 {
		t.Errorf("provider request counts AT the moment of each consultation = %v, want [0 1] -- each consultation must precede that turn's own first provider call", counts)
	}

	baseH, _ := scriptedTwoTurnHarness(t, "001", nil, agent.ContextBudget{})
	baseSink := make(chan *agent.Event, 256)
	_, baseFinish, baseErr := baseH.Run(contextBackground(), firstMessage(t), baseSink)
	baseEvents := drainSink(t, baseSink)

	if (err == nil) != (baseErr == nil) {
		t.Errorf("returned error nil-ness differs: with strategy = %v, without = %v", err, baseErr)
	}
	if finish != baseFinish {
		t.Errorf("finish reason differs: with strategy = %v, without = %v", finish, baseFinish)
	}
	assertEventStreamsStructurallyIdentical(t, events, baseEvents)
}

// TestHarness_ContextStrategy_RetriedTurnConsultsOnce — S-CTX-002, the
// reading this requirement exists to exclude. The first logical turn
// fails retryably once (no partial output) then completes; the second
// completes cleanly — three provider attempts across two logical
// turns — and the recorder holds EXACTLY two consultations, not
// three; the two transcripts the strategy received are the two
// distinct per-logical-turn transcripts, never the same one twice.
func TestHarness_ContextStrategy_RetriedTurnConsultsOnce(t *testing.T) {
	t.Parallel()

	toolName, callID := "read_ctx_002", "call-ctx-002"
	tool := EchoScriptedTool(toolName, agent.EffectClassRead)
	reg := agent.NewMapRegistry(map[string]agent.Tool{toolName: tool})
	inner := agenttest.NewProvider(
		scriptToolCallResponse(t, callID, toolName, []byte(`{"x":1}`)),
		scriptTextResponse(t, ai.FinishReasonStop),
	)
	failure := mustPreStreamRetryableFailure(t, ai.RetryDelay{})
	provider := newPreStreamFailingProvider(1, failure, inner) // fails the FIRST attempt only

	strategy := &recordingContextStrategy{}
	h := agent.Harness{
		Provider:        provider,
		System:          "system prompt for S-CTX-002",
		Turn:            agent.TurnOptions{Tools: reg},
		RetryAttempts:   3,
		RetryTiming:     agent.RetryTiming{SleepFunc: instantSleep},
		ContextStrategy: strategy,
	}

	sink := make(chan *agent.Event, 256)
	_, _, err := h.Run(contextBackground(), firstMessage(t), sink)
	if err != nil {
		t.Fatalf("Run returned err = %v, want nil (the retry succeeds within the bound)", err)
	}
	drainSink(t, sink)

	if got := len(provider.Requests()); got != 3 {
		t.Fatalf("wrapper recorded %d request(s), want exactly 3 (2 logical turns, the first retried once)", got)
	}

	prompts := strategy.Prompts()
	if len(prompts) != 2 {
		t.Fatalf("recorder holds %d consultation(s), want exactly 2 -- NOT 3 (R-CTX-001: the unit is the LOGICAL TURN, never the attempt)", len(prompts))
	}
	if len(prompts[0].Transcript) >= len(prompts[1].Transcript) {
		t.Errorf("prompt[0].Transcript has %d message(s), prompt[1].Transcript has %d -- want the second STRICTLY longer (turn one's committed entries), proving the two consultations received distinct transcripts", len(prompts[0].Transcript), len(prompts[1].Transcript))
	}
}

// TestHarness_ContextStrategy_PromptCarriesTranscriptAndBudget —
// S-CTX-004, the inputs half. Each recorded prompt's transcript is
// element-equal to the messages of the request that turn's provider
// call actually carried, and its budget echoes the stated limit
// together with a present reading. A strategy that appends to and
// overwrites the slice it received does not change what the provider
// records — the clone, not a convention, is what makes that true.
func TestHarness_ContextStrategy_PromptCarriesTranscriptAndBudget(t *testing.T) {
	t.Parallel()

	strategy := &recordingContextStrategy{}
	h, provider := scriptedTwoTurnHarness(t, "004", strategy, agent.ContextBudgetOf(4096))
	sink := make(chan *agent.Event, 256)
	if _, _, err := h.Run(contextBackground(), firstMessage(t), sink); err != nil {
		t.Fatalf("Run returned err = %v, want nil", err)
	}
	drainSink(t, sink)

	prompts := strategy.Prompts()
	requests := provider.Requests()
	if len(prompts) != 2 || len(requests) != 2 {
		t.Fatalf("got %d prompt(s) and %d request(s), want 2 and 2", len(prompts), len(requests))
	}
	for i, p := range prompts {
		want := requests[i].Messages()
		if len(p.Transcript) != len(want) {
			t.Fatalf("prompt[%d].Transcript has %d message(s), want %d (element-equal to the sent request's messages)", i, len(p.Transcript), len(want))
		}
		for j := range want {
			if !p.Transcript[j].Equal(want[j]) {
				t.Errorf("prompt[%d].Transcript[%d] != request[%d].Messages()[%d]", i, j, i, j)
			}
		}
		limit, present := p.Budget.Limit()
		if !present || limit != 4096 {
			t.Errorf("prompt[%d].Budget.Limit() = (%d, %v), want (4096, true)", i, limit, present)
		}
	}

	mutating := mutatingContextStrategy{}
	mh, mProvider := scriptedTwoTurnHarness(t, "004", mutating, agent.ContextBudgetOf(4096))
	mSink := make(chan *agent.Event, 256)
	if _, _, err := mh.Run(contextBackground(), firstMessage(t), mSink); err != nil {
		t.Fatalf("mutating-strategy Run returned err = %v, want nil", err)
	}
	drainSink(t, mSink)
	mutatedRequests := mProvider.Requests()
	if len(mutatedRequests) != len(requests) {
		t.Fatalf("mutating-strategy run recorded %d request(s), want %d", len(mutatedRequests), len(requests))
	}
	for i := range requests {
		if !mutatedRequests[i].Equal(requests[i]) {
			t.Errorf("request[%d] differs between the non-mutating and mutating-strategy runs -- the seam must hand the strategy a fresh CLONE", i)
		}
	}
}

// TestHarness_ContextStrategy_NoOpInstalled_ZeroCompactionEvents —
// S-CTX-006. Installing the shipped never-compact default: the
// compile-time guard binds it to the seam, every consultation returns
// the zero verdict (mechanically true, since none other exists), and
// no compaction-family event appears anywhere on the recorded stream.
func TestHarness_ContextStrategy_NoOpInstalled_ZeroCompactionEvents(t *testing.T) {
	t.Parallel()

	var strategy agent.ContextStrategy = agent.NoOpContextStrategy{} // compile-time guard binds
	h, _ := scriptedTwoTurnHarness(t, "006", strategy, agent.ContextBudget{})

	sink := make(chan *agent.Event, 256)
	if _, _, err := h.Run(contextBackground(), firstMessage(t), sink); err != nil {
		t.Fatalf("Run returned err = %v, want nil", err)
	}
	events := drainSink(t, sink)
	if len(events) == 0 {
		t.Fatal("zero events observed")
	}
	for _, ev := range events {
		if isCompactionKind(ev.Kind()) {
			t.Errorf("event kind %v is a compaction-family kind, want none anywhere on the stream", ev.Kind())
		}
	}
}

// TestHarness_ContextStrategy_NilVsNoOpByteIdentical — S-CTX-007,
// charter AG-17.1 scenario 2 (the pin). Two runs of the identical
// multi-turn script against the identical non-counting fake provider
// -- run A with the seam field nil, run B with the shipped
// never-compact default installed -- produce structurally identical
// recorded event streams (R-CTX-004's byte-identity comparison, up to
// the one deliberate identity exclusion documented on
// assertEventStreamsStructurallyIdentical), neither carries any
// compaction-family kind, and both runs' outcomes are equal.
func TestHarness_ContextStrategy_NilVsNoOpByteIdentical(t *testing.T) {
	t.Parallel()

	run := func(t *testing.T, strategy agent.ContextStrategy, budget agent.ContextBudget) (ai.FinishReason, []agent.Event, error) {
		t.Helper()
		h, _ := scriptedTwoTurnHarness(t, "007", strategy, budget)
		sink := make(chan *agent.Event, 256)
		_, finish, err := h.Run(contextBackground(), firstMessage(t), sink)
		return finish, drainSink(t, sink), err
	}

	nilFinish, nilEvents, nilErr := run(t, nil, agent.ContextBudget{})
	noOpFinish, noOpEvents, noOpErr := run(t, agent.NoOpContextStrategy{}, agent.ContextBudgetOf(1000))

	if (nilErr == nil) != (noOpErr == nil) {
		t.Fatalf("returned error nil-ness: nil = %v, NoOp = %v, want equal", nilErr, noOpErr)
	}
	if nilFinish != noOpFinish {
		t.Errorf("finish reason: nil = %v, NoOp = %v, want equal", nilFinish, noOpFinish)
	}
	assertEventStreamsStructurallyIdentical(t, nilEvents, noOpEvents)

	for _, ev := range nilEvents {
		if isCompactionKind(ev.Kind()) {
			t.Errorf("nil-field run: event kind %v is compaction-family, want none", ev.Kind())
		}
	}
	for _, ev := range noOpEvents {
		if isCompactionKind(ev.Kind()) {
			t.Errorf("NoOp run: event kind %v is compaction-family, want none", ev.Kind())
		}
	}
}

// --- Phase 6 -- externally-observed AG-17.2 scenarios ---------------------

// assertNoTokenCountingInterfaceDeclared parses every non-test source
// file in this package's own directory and fails if any declares an
// interface carrying a CountTokens method — Layer 2 MUST declare NO
// counting contract of its own; ai.TokenCounter is the only one
// (R-CTX-006, R-AMP-017).
func assertNoTokenCountingInterfaceDeclared(t *testing.T) {
	t.Helper()

	entries, err := os.ReadDir(".")
	if err != nil {
		t.Fatalf("os.ReadDir(%q) returned %v, want nil", ".", err)
	}

	fset := token.NewFileSet()
	for _, entry := range entries {
		name := entry.Name()
		if entry.IsDir() || !strings.HasSuffix(name, ".go") || strings.HasSuffix(name, "_test.go") {
			continue
		}
		file, perr := parser.ParseFile(fset, name, nil, 0)
		if perr != nil {
			t.Fatalf("parser.ParseFile(%q) returned %v, want nil", name, perr)
		}
		for _, decl := range file.Decls {
			gen, ok := decl.(*ast.GenDecl)
			if !ok || gen.Tok != token.TYPE {
				continue
			}
			for _, spec := range gen.Specs {
				ts, ok := spec.(*ast.TypeSpec)
				if !ok {
					continue
				}
				iface, ok := ts.Type.(*ast.InterfaceType)
				if !ok || iface.Methods == nil {
					continue
				}
				for _, method := range iface.Methods.List {
					for _, methodName := range method.Names {
						if methodName.Name == "CountTokens" {
							t.Errorf("%s declares interface %s with a CountTokens method -- Layer 2 must declare NO token-counting interface of its own (R-CTX-006, R-AMP-017)", name, ts.Name.Name)
						}
					}
				}
			}
		}
	}
}

// TestHarness_ContextStrategy_Accounting_DiscoveryAndFallback —
// S-CTX-011, charter AG-17.2 scenario 1, the discovery half. A
// counting-capable exported fake reports its own figure with the
// Reported provenance, and the fixture recorded a counting call built
// from the turn's own transcript; the existing non-counting exported
// fake, driven with the identical script, makes no counting call at
// all and reports Estimated. Layer 2 declares no token-counting
// interface of its own.
func TestHarness_ContextStrategy_Accounting_DiscoveryAndFallback(t *testing.T) {
	t.Parallel()

	toolName, callID := "read_ctx_011", "call-ctx-011"
	tool := EchoScriptedTool(toolName, agent.EffectClassRead)
	reg := agent.NewMapRegistry(map[string]agent.Tool{toolName: tool})

	countingProvider := agenttest.NewCountingProvider(ai.Tokens(4321),
		scriptToolCallResponse(t, callID, toolName, []byte(`{"x":1}`)),
		scriptTextResponse(t, ai.FinishReasonStop),
	)
	countingStrategy := &recordingContextStrategy{}
	countingH := agent.Harness{Provider: countingProvider, System: "system prompt for S-CTX-011", Turn: agent.TurnOptions{Tools: reg}, ContextStrategy: countingStrategy}
	countingSink := make(chan *agent.Event, 256)
	if _, _, err := countingH.Run(contextBackground(), firstMessage(t), countingSink); err != nil {
		t.Fatalf("Run (counting) returned err = %v, want nil", err)
	}
	drainSink(t, countingSink)

	countingPrompts := countingStrategy.Prompts()
	countingCalls := countingProvider.CountRequests()
	if len(countingPrompts) != 2 || len(countingCalls) != 2 {
		t.Fatalf("counting run: got %d prompt(s) and %d counting call(s), want 2 and 2", len(countingPrompts), len(countingCalls))
	}
	for i, p := range countingPrompts {
		tokens, source := p.Accounting.Tokens()
		if source != agent.TokenSourceReported || tokens != 4321 {
			t.Errorf("counting run: prompt[%d].Accounting = (%d, %v), want (4321, Reported)", i, tokens, source)
		}
		if len(countingCalls[i].Messages()) != len(p.Transcript) {
			t.Errorf("counting run: counting call[%d] carries %d message(s), prompt[%d].Transcript carries %d -- want the counting call built from the turn's own transcript", i, len(countingCalls[i].Messages()), i, len(p.Transcript))
		}
	}

	nonCountingProvider := agenttest.NewProvider(
		scriptToolCallResponse(t, callID, toolName, []byte(`{"x":1}`)),
		scriptTextResponse(t, ai.FinishReasonStop),
	)
	nonCountingStrategy := &recordingContextStrategy{}
	nonCountingH := agent.Harness{Provider: nonCountingProvider, System: "system prompt for S-CTX-011", Turn: agent.TurnOptions{Tools: reg}, ContextStrategy: nonCountingStrategy}
	nonCountingSink := make(chan *agent.Event, 256)
	if _, _, err := nonCountingH.Run(contextBackground(), firstMessage(t), nonCountingSink); err != nil {
		t.Fatalf("Run (non-counting) returned err = %v, want nil", err)
	}
	drainSink(t, nonCountingSink)

	nonCountingPrompts := nonCountingStrategy.Prompts()
	if len(nonCountingPrompts) != 2 {
		t.Fatalf("non-counting run: recorder holds %d consultation(s), want 2", len(nonCountingPrompts))
	}
	for i, p := range nonCountingPrompts {
		_, source := p.Accounting.Tokens()
		if source != agent.TokenSourceEstimated {
			t.Errorf("non-counting run: prompt[%d].Accounting source = %v, want Estimated (no counting call is possible against this fake)", i, source)
		}
	}

	assertNoTokenCountingInterfaceDeclared(t)
}

// TestHarness_Accounting_PreHookDivergenceRecorded — S-CTX-019, DD6.
// A counting-capable fixture plus a request-mutating PreRequestHook:
// the accounting the strategy received corresponds to the pre-hook
// request the turn's own builder produced, and the fixture's captured
// counting request DIFFERS from the provider's recorded sent request
// -- the divergence is recorded, never asserted away. The shipped
// documentation of the reported provenance and the prompt's
// accounting field both state the pre-hook semantic.
func TestHarness_Accounting_PreHookDivergenceRecorded(t *testing.T) {
	t.Parallel()

	extraPart, err := ai.NewText("appended by the pre-request hook, downstream of the turn boundary")
	if err != nil {
		t.Fatalf("ai.NewText returned %v, want nil", err)
	}
	extra, err := ai.NewMessage(ai.RoleUser, extraPart)
	if err != nil {
		t.Fatalf("ai.NewMessage returned %v, want nil", err)
	}
	hook := func(_ context.Context, req ai.Request) (ai.Request, error) {
		return req.With(ai.WithMessages(append(req.Messages(), extra)...))
	}

	provider := agenttest.NewCountingProvider(ai.Tokens(999), scriptTextResponse(t, ai.FinishReasonStop))
	strategy := &recordingContextStrategy{}
	h := agent.Harness{
		Provider:        provider,
		System:          "system prompt for S-CTX-019",
		Turn:            agent.TurnOptions{PreRequestHook: hook},
		ContextStrategy: strategy,
	}
	sink := make(chan *agent.Event, 64)
	if _, _, err := h.Run(contextBackground(), firstMessage(t), sink); err != nil {
		t.Fatalf("Run returned err = %v, want nil", err)
	}
	drainSink(t, sink)

	prompts := strategy.Prompts()
	if len(prompts) != 1 {
		t.Fatalf("recorder holds %d consultation(s), want exactly 1 (one logical turn)", len(prompts))
	}
	tokens, source := prompts[0].Accounting.Tokens()
	if source != agent.TokenSourceReported || tokens != 999 {
		t.Fatalf("Accounting = (%d, %v), want (999, Reported)", tokens, source)
	}

	countedRequests := provider.CountRequests()
	sentRequests := provider.Requests()
	if len(countedRequests) != 1 || len(sentRequests) != 1 {
		t.Fatalf("got %d counted request(s) and %d sent request(s), want 1 and 1", len(countedRequests), len(sentRequests))
	}
	if countedRequests[0].Equal(sentRequests[0]) {
		t.Error("the counted (pre-hook) request equals the sent (post-hook) request -- want them to DIVERGE, since the hook appends a message downstream of the accounting point")
	}
	if len(sentRequests[0].Messages()) != len(countedRequests[0].Messages())+1 {
		t.Errorf("sent request has %d message(s), counted request has %d -- want the sent one to carry exactly one MORE (the hook's own appended message)", len(sentRequests[0].Messages()), len(countedRequests[0].Messages()))
	}

	for _, file := range []string{"token_accounting.go", "context_strategy.go"} {
		data, rerr := os.ReadFile(file)
		if rerr != nil {
			t.Fatalf("os.ReadFile(%q) returned %v, want nil", file, rerr)
		}
		if !strings.Contains(strings.ToLower(string(data)), "pre-hook request") {
			t.Errorf("%s's documentation does not state \"pre-hook request\" (R-CTX-011: the reported provenance and the prompt's accounting field must both state this semantic, never claim exactness)", file)
		}
	}
}

// --- Phase 7 -- closed-sequence proof & substrate verification -----------
// (R-CTX-012, NFR-CTX-002..004)

// TestHarness_ContextStrategy_ClosedSequencesUnaffected — S-CTX-020.
// A multi-logical-turn run with the never-compact default installed
// carries zero compaction-family events, and its kind sequence is
// equal, position for position, to the identical run with the seam
// nil. TestTurn_WalkingSkeleton_EmitsContractEventOrder (S-LSK-001)
// and TestHarness_CostSession_FinalOnShutdownRun (S-CAN-013) are run
// unmodified and byte-unchanged alongside this file -- confirmed by
// the full-suite evidence in apply-progress.md, not re-asserted here.
func TestHarness_ContextStrategy_ClosedSequencesUnaffected(t *testing.T) {
	t.Parallel()

	run := func(t *testing.T, strategy agent.ContextStrategy) []agent.EventKind {
		t.Helper()
		h, _ := scriptedTwoTurnHarness(t, "020", strategy, agent.ContextBudget{})
		sink := make(chan *agent.Event, 256)
		if _, _, err := h.Run(contextBackground(), firstMessage(t), sink); err != nil {
			t.Fatalf("Run returned err = %v, want nil", err)
		}
		events := drainSink(t, sink)
		kinds := make([]agent.EventKind, len(events))
		for i, ev := range events {
			kinds[i] = ev.Kind()
		}
		return kinds
	}

	nilKinds := run(t, nil)
	noOpKinds := run(t, agent.NoOpContextStrategy{})

	if len(nilKinds) != len(noOpKinds) {
		t.Fatalf("kind sequence length: nil = %d, NoOp = %d, want equal", len(nilKinds), len(noOpKinds))
	}
	for i := range nilKinds {
		if nilKinds[i] != noOpKinds[i] {
			t.Errorf("kind[%d]: nil = %v, NoOp = %v, want equal", i, nilKinds[i], noOpKinds[i])
		}
		if isCompactionKind(nilKinds[i]) || isCompactionKind(noOpKinds[i]) {
			t.Errorf("kind[%d] is compaction-family, want none anywhere on either stream", i)
		}
	}
}

// TestNoRelease_SubstrateByteUnchanged — S-CTX-021, R-CTX-012, part of
// NFR-CTX-004. Diffing the merge base against the working tree: the
// named substrate files are byte-unchanged, the diff under
// backend/agent/src/ai/ is empty, go.mod/go.sum are byte-unchanged,
// and expectedLayer2ContractRows still carries exactly eight rows.
func TestNoRelease_SubstrateByteUnchanged(t *testing.T) {
	root, err := gitTopLevel(t)
	if err != nil {
		t.Fatalf("git rev-parse --show-toplevel failed: %v", err)
	}

	mainRef := os.Getenv("AG07_BASE_REF")
	if mainRef == "" {
		out, err := gitOutput(t, root, "merge-base", "HEAD", "origin/main")
		if err != nil {
			t.Fatalf("cannot determine base ref (set AG07_BASE_REF): %v", err)
		}
		mainRef = out
	}

	untouched := []string{
		"backend/agent/src/agent/doc.go",
		"backend/agent/src/agent/doc_contract_guard_test.go",
		"backend/agent/src/agent/stream_check.go",
		"backend/agent/src/agent/event.go",
		"backend/agent/src/agent/event_descriptor.go",
		"backend/agent/src/agent/event_registry_test.go",
		"backend/agent/src/agent/compaction_events.go",
		"backend/agent/src/agent/cost_events.go",
		"backend/agent/src/agent/cost_usage.go",
		"backend/agent/src/agent/history.go",
	}
	for _, path := range untouched {
		diff, err := gitDiff(t, root, mainRef, path)
		if err != nil {
			t.Fatalf("git diff %s -- %s failed: %v", mainRef, path, err)
		}
		if len(diff) != 0 {
			t.Errorf("%s is NOT byte-unchanged against %s (R-CTX-012, NFR-CTX-004):\n%s", path, mainRef, diff)
		}
	}

	aiDiff, err := gitDiff(t, root, mainRef, "backend/agent/src/ai/")
	if err != nil {
		t.Fatalf("git diff %s -- backend/agent/src/ai/ failed: %v", mainRef, err)
	}
	if len(aiDiff) != 0 {
		t.Errorf("backend/agent/src/ai/ is NOT byte-unchanged against %s -- AG-17 must never edit Layer 1:\n%s", mainRef, aiDiff)
	}

	goModSum, err := gitDiff(t, root, mainRef, "backend/agent/go.mod", "backend/agent/go.sum")
	if err != nil {
		t.Fatalf("git diff %s -- go.mod go.sum failed: %v", mainRef, err)
	}
	if len(goModSum) != 0 {
		t.Errorf("go.mod / go.sum drifted from %s:\n%s", mainRef, goModSum)
	}

	if len(expectedLayer2ContractRows) != 8 {
		t.Errorf("expectedLayer2ContractRows carries %d row(s), want exactly 8 (L2C-01..L2C-08) -- AG-17 declares no new package-wide contract row", len(expectedLayer2ContractRows))
	}
}
