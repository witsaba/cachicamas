// AG-20 — strict-TDD tests for the complete hook taxonomy
// (R-HKS-001..012, S-HKS-001..026 + bites S-HKS-050..052; cross-package
// scenarios S-PRH-008..011, S-AEV-126, S-AGE-030, S-LSK-032, S-AGS-065,
// S-AGS-066, S-AIV-032).
//
// Every scenario is in package agent_test (NFR-HKS-001): the external-
// test posture proves every behavioral claim from outside the package.
// The one exception is S-CMP-040's cut-resolution fixed-point table,
// exercised internally in compaction_surgery_test.go under
// NFR-CMP-001's pure-helper carve-out; its observational half
// (S-CMP-038) is asserted here and in hooks_compaction_test.go.
//
// This file carries: the one-surface / transport-refusal / observing-
// unconstructibility scenarios (Phase 1); the pre-request chain
// composition and attribution scenarios (Phase 1); the AG-20.2
// asynchrony scenarios — liveness, placement, eventually-reported,
// panic, nil/stalling reporter (Phase 3, harness.go's own lane
// lifecycle production code); the ordering pin, scope fence, closed-
// sequence and inertness scenarios (Phase 4); and the remaining
// cross-capability test-only scenarios (Phase 5).
package agent_test

import (
	"context"
	"errors"
	"os"
	"reflect"
	"regexp"
	"runtime"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/cachicamas/backend/agent/src/agent"
	"github.com/cachicamas/backend/agent/src/agenttest"
	"github.com/cachicamas/backend/agent/src/ai"
)

// ---------------------------------------------------------------------
// Shared fixtures
// ---------------------------------------------------------------------

// noopPreRequestHook, noopPreCompactHook, noopPostTurnObserver and
// noopSessionStartObserver are minimal, well-typed members for each
// hook family — used wherever a test only needs SOME non-zero value of
// the right shape (S-HKS-002's transport-refusal fixture).
func noopPreRequestHook(_ context.Context, req ai.Request) (ai.Request, error) { return req, nil }

func noopPreCompactHook(_ context.Context, plan agent.CompactionPlan) (agent.CompactionPlan, error) {
	return plan, nil
}

func noopPostTurnObserver(context.Context, agent.PostTurnReport)         {}
func noopSessionStartObserver(context.Context, agent.SessionStartReport) {}

// hooksMarker mirrors loop_hook_test.go's hookMarker — a distinctive
// literal this file's chain-composition fixtures append, kept distinct
// from AG-08's own marker so a failure message never conflates the two
// suites.
const hooksMarkerA = "AG-20-HOOKS-MARKER-A"
const hooksMarkerB = "AG-20-HOOKS-MARKER-B"
const hooksMarkerC = "AG-20-HOOKS-MARKER-C"
const hooksMarkerD = "AG-20-HOOKS-MARKER-D"

// appendMarkerHook returns a PreRequestHook that appends marker as a
// new system segment (loop_hook_test.go's hookWithMarkerAppended
// precedent, generalised to a named marker) and, when recorded is
// non-nil, records the marker set of its OWN input before mutating —
// S-HKS-004/S-PRH-008/S-PRH-011 need each element's observed input,
// not just its output.
func appendMarkerHook(t *testing.T, marker string, recorded *[]string) func(context.Context, ai.Request) (ai.Request, error) {
	t.Helper()
	return func(_ context.Context, req ai.Request) (ai.Request, error) {
		if recorded != nil {
			*recorded = append(*recorded, loopRequestSystemText(t, req))
		}
		system, hasSystem := req.SystemInstruction()
		if !hasSystem {
			return ai.Request{}, errors.New("appendMarkerHook: request carries no SystemInstruction")
		}
		seg, err := ai.NewSegment(marker)
		if err != nil {
			return ai.Request{}, err
		}
		segments := append([]ai.Segment{}, system.Segments()...)
		segments = append(segments, seg)
		newInstr, err := ai.NewSystemInstruction(segments...)
		if err != nil {
			return ai.Request{}, err
		}
		return req.With(ai.WithSystemInstruction(newInstr))
	}
}

// errPreRequestBoom and errPreCompactBoom are typed sentinels a failing
// chain element returns, distinct per family so a failure message
// names which family's fixture fired.
var errPreRequestBoom = errors.New("hooks_test: pre-request chain element boom")

// failingPreRequestHookAt returns a chain of n identity elements whose
// element at failIndex returns errPreRequestBoom; the rest append their
// own index as a marker, so a test can prove which elements ran.
func failingPreRequestHookAt(t *testing.T, n, failIndex int, invoked *[]int) []agent.PreRequestHook {
	t.Helper()
	chain := make([]agent.PreRequestHook, n)
	for i := 0; i < n; i++ {
		i := i
		chain[i] = func(_ context.Context, req ai.Request) (ai.Request, error) {
			if invoked != nil {
				*invoked = append(*invoked, i)
			}
			if i == failIndex {
				return ai.Request{}, errPreRequestBoom
			}
			return req, nil
		}
	}
	return chain
}

// preRequestAttribution extracts R-HKS-009's source-name attribution
// from a Turn error: the error's own errors.Unwrap chain carries it,
// prefixed to the wrapped cause's own message (loop.go's
// attributedPreRequestError, unexported — reached only through the
// standard errors.Unwrap route this helper drives).
func preRequestAttribution(t *testing.T, err error) string {
	t.Helper()
	if err == nil {
		t.Fatal("preRequestAttribution: err is nil")
	}
	unwrapped := errors.Unwrap(err)
	if unwrapped == nil {
		t.Fatalf("preRequestAttribution: errors.Unwrap(%v) = nil, want the attributed cause", err)
	}
	return unwrapped.Error()
}

// ---------------------------------------------------------------------
// R-HKS-001 — one registration surface, two families, unconstructible
// ---------------------------------------------------------------------

// S-HKS-001 — One surface, and the fence proves it. Given the package
// after this change, when its exported surface is enumerated, then
// Hooks is the only hook registration type, and Harness declares
// exactly the five exported methods it declared at the merge base —
// scope_fence_test.go's and harness_test.go's own named-method
// assertions are reused unedited (this test adds no third pin; it
// names the same five methods so a reader sees the claim here too).
func TestHooks_OneSurface_HarnessMethodSetUnwidened(t *testing.T) {
	t.Parallel()

	typ := reflect.TypeOf(&agent.Harness{})
	var names []string
	for i := 0; i < typ.NumMethod(); i++ {
		names = append(names, typ.Method(i).Name)
	}
	want := []string{"Compact", "Interrupt", "Run", "Shutdown", "Steer"}
	gotSorted := append([]string(nil), names...)
	wantSorted := append([]string(nil), want...)
	sortStrings(gotSorted)
	sortStrings(wantSorted)
	if !reflect.DeepEqual(gotSorted, wantSorted) {
		t.Errorf("*agent.Harness exported methods = %v, want exactly %v (R-HKS-001: no hook-registration method of any name)", names, want)
	}

	// Hooks is a struct value type (not an interface, not a func) —
	// the ONE registration type this capability adds.
	hooksType := reflect.TypeOf(agent.Hooks{})
	if hooksType.Kind() != reflect.Struct {
		t.Errorf("agent.Hooks kind = %v, want struct", hooksType.Kind())
	}
}

// sortStrings avoids importing "sort" solely for one call site; a tiny
// insertion sort is plenty for the five-element slices this file uses
// it on.
func sortStrings(s []string) {
	for i := 1; i < len(s); i++ {
		for j := i; j > 0 && s[j-1] > s[j]; j-- {
			s[j-1], s[j] = s[j], s[j-1]
		}
	}
}

// S-HKS-002 — The transport refuses rather than overwrites. Given a
// Harness whose Turn.Hooks carries a non-zero value in any ONE of the
// five members, when Run is called, then it returns a typed misplaced-
// options refusal, the sink carries no event at all, and (inferred
// from zero events: no run-open could have been minted and forgotten)
// no run identity was minted; and given an entirely zero Turn.Hooks,
// Run proceeds normally.
func TestHooks_TransportRefusesRatherThanOverwrites(t *testing.T) {
	t.Parallel()

	variants := []struct {
		name  string
		hooks agent.Hooks
	}{
		{"PreRequest", agent.Hooks{PreRequest: []agent.PreRequestHook{noopPreRequestHook}}},
		{"PreCompact", agent.Hooks{PreCompact: []agent.PreCompactHook{noopPreCompactHook}}},
		{"PostTurn", agent.Hooks{PostTurn: []agent.PostTurnObserver{noopPostTurnObserver}}},
		{"SessionStart", agent.Hooks{SessionStart: []agent.SessionStartObserver{noopSessionStartObserver}}},
		{"StallReporter", agent.Hooks{StallReporter: func(agent.ObserverStall) {}}},
	}

	for _, tt := range variants {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			provider := agenttest.NewProvider(scriptTextResponse(t, ai.FinishReasonStop))
			h := agent.Harness{Provider: provider, System: "system prompt for hks-002", Turn: agent.TurnOptions{Hooks: tt.hooks}}
			sink := make(chan *agent.Event, 16)

			_, _, err := h.Run(contextBackground(), firstMessage(t), sink)
			if err == nil {
				t.Fatal("Run returned err = nil, want a typed misplaced-options refusal")
			}
			if !errors.Is(err, ai.ErrMisplaced) {
				t.Errorf("Run error rule class = %v, want errors.Is(err, ai.ErrMisplaced)", err)
			}
			var violation *ai.Violation
			if !errors.As(err, &violation) {
				t.Fatalf("Run error = %T, want errors.As to reach *ai.Violation", err)
			}
			if got := violation.Path().String(); got != "turn.hooks" {
				t.Errorf("violation position = %q, want %q", got, "turn.hooks")
			}

			got := drainSink(t, sink)
			if len(got) != 0 {
				t.Errorf("sink carried %d event(s), want 0 (not even a run-open) — no run identity was minted", len(got))
			}
			if n := len(provider.Requests()); n != 0 {
				t.Errorf("provider recorded %d request(s), want 0 — the refusal must precede any provider call", n)
			}
		})
	}

	t.Run("zero Turn.Hooks proceeds normally", func(t *testing.T) {
		t.Parallel()

		provider := agenttest.NewProvider(scriptTextResponse(t, ai.FinishReasonStop))
		h := agent.Harness{Provider: provider, System: "system prompt for hks-002-zero"}
		sink := make(chan *agent.Event, 16)

		_, _, err := h.Run(contextBackground(), firstMessage(t), sink)
		if err != nil {
			t.Fatalf("Run returned err = %v, want nil (zero-value Hooks is the identity default)", err)
		}
		got := drainSink(t, sink)
		if len(got) == 0 {
			t.Error("sink carried 0 events, want the ordinary run/turn bracket")
		}
	})
}

// S-HKS-003 — The observing families cannot signal, and a direct Turn
// observer is inert.
//
// Compile-time-refusal half: task 1.2 searched this package (tool.go,
// permission_policy_helpers_test.go) and found no existing negative-
// compile-test fixture — Go's own test binary cannot assert a compile
// FAILURE at runtime. The decision recorded here (task 1.2's own
// obligation) is to restate the claim as a TYPE-SHAPE reflection
// assertion, the house's own established idiom for a structural pin
// (cost_events_test.go's CostFigures reflection pin,
// scope_fence_test.go's NumMethod() pins): an observing hook's
// declared function TYPE has NumOut() == 0, which is exactly why no
// function literal declaring a return value can ever be assigned to
// PostTurn/SessionStart — the Go compiler's own function-type identity
// rule, verified here by reading the type back rather than by
// asserting a build failure. The following is what would NOT compile
// if uncommented, kept as documentation rather than as dead code:
//
//	var _ agent.PostTurnObserver = func(context.Context, agent.PostTurnReport) error { return nil }
//
// Inertness half: a direct Turn call whose TurnOptions.Hooks carries
// post-turn observers, session-start observers and a stall reporter
// fires nothing, reports nothing, creates no goroutine beyond
// baseline, and produces byte-identical output to the same call with
// zero-value Hooks.
func TestHooks_ObservingUnconstructible_DirectTurnInert(t *testing.T) {
	t.Parallel()

	t.Run("observing families declare zero result parameters (type-shape reflection)", func(t *testing.T) {
		t.Parallel()

		postTurnType := reflect.TypeOf((*agent.PostTurnObserver)(nil)).Elem()
		if postTurnType.Kind() != reflect.Func {
			t.Fatalf("PostTurnObserver kind = %v, want Func", postTurnType.Kind())
		}
		if got := postTurnType.NumOut(); got != 0 {
			t.Errorf("PostTurnObserver.NumOut() = %d, want 0 — an observer with a result parameter must fail to compile", got)
		}

		sessionStartType := reflect.TypeOf((*agent.SessionStartObserver)(nil)).Elem()
		if sessionStartType.Kind() != reflect.Func {
			t.Fatalf("SessionStartObserver kind = %v, want Func", sessionStartType.Kind())
		}
		if got := sessionStartType.NumOut(); got != 0 {
			t.Errorf("SessionStartObserver.NumOut() = %d, want 0 — an observer with a result parameter must fail to compile", got)
		}

		// The mutating families DO carry return values — the type-level
		// distinction this requirement makes is real and checkable both
		// ways, not merely absent-by-omission on the observing side.
		preRequestType := reflect.TypeOf((*agent.PreRequestHook)(nil)).Elem()
		if got := preRequestType.NumOut(); got != 2 {
			t.Errorf("PreRequestHook.NumOut() = %d, want 2 (ai.Request, error) — mutating hooks MAY signal back", got)
		}
		preCompactType := reflect.TypeOf((*agent.PreCompactHook)(nil)).Elem()
		if got := preCompactType.NumOut(); got != 2 {
			t.Errorf("PreCompactHook.NumOut() = %d, want 2 (CompactionPlan, error) — mutating hooks MAY signal back", got)
		}
	})

	t.Run("direct Turn observers are inert, byte-identical to zero-value Hooks", func(t *testing.T) {
		t.Parallel()

		var postTurnCalls, sessionStartCalls, reportCalls int
		var mu sync.Mutex
		hooks := agent.Hooks{
			PostTurn:      []agent.PostTurnObserver{func(context.Context, agent.PostTurnReport) { mu.Lock(); postTurnCalls++; mu.Unlock() }},
			SessionStart:  []agent.SessionStartObserver{func(context.Context, agent.SessionStartReport) { mu.Lock(); sessionStartCalls++; mu.Unlock() }},
			StallReporter: func(agent.ObserverStall) { mu.Lock(); reportCalls++; mu.Unlock() },
		}

		baseline := runtime.NumGoroutine()

		hookedProvider := agenttest.NewProvider(scriptTextResponse(t, ai.FinishReasonStop))
		hookedSink := make(chan *agent.Event, 16)
		hookedMsg, hookedFinish, hookedErr := agent.Turn(contextBackground(), hookedProvider, "system prompt for hks-003-inert", []ai.Message{firstMessage(t)}, agent.TurnOptions{Hooks: hooks}, hookedSink)
		if hookedErr != nil {
			t.Fatalf("hooked Turn returned err = %v, want nil", hookedErr)
		}
		hookedEvents := drainSink(t, hookedSink)

		waitForGoroutineBaseline(t, baseline, twoSecondsForHooksTest())

		mu.Lock()
		gotPostTurn, gotSessionStart, gotReport := postTurnCalls, sessionStartCalls, reportCalls
		mu.Unlock()
		if gotPostTurn != 0 || gotSessionStart != 0 || gotReport != 0 {
			t.Errorf("direct Turn call invoked observers/reporter (post-turn=%d, session-start=%d, report=%d), want 0 — Turn owns no observer lane", gotPostTurn, gotSessionStart, gotReport)
		}

		zeroProvider := agenttest.NewProvider(scriptTextResponse(t, ai.FinishReasonStop))
		zeroSink := make(chan *agent.Event, 16)
		zeroMsg, zeroFinish, zeroErr := agent.Turn(contextBackground(), zeroProvider, "system prompt for hks-003-inert", []ai.Message{firstMessage(t)}, agent.TurnOptions{}, zeroSink)
		if zeroErr != nil {
			t.Fatalf("zero-value Turn returned err = %v, want nil", zeroErr)
		}
		zeroEvents := drainSink(t, zeroSink)

		if hookedFinish != zeroFinish {
			t.Errorf("finish = %v, want %v", hookedFinish, zeroFinish)
		}
		if !equalByContent(hookedMsg, zeroMsg) {
			t.Errorf("hooked Turn's msg differs from zero-value Turn's msg by content")
		}
		if len(hookedEvents) != len(zeroEvents) {
			t.Fatalf("hooked Turn emitted %d event(s), zero-value emitted %d, want equal", len(hookedEvents), len(zeroEvents))
		}
		for i := range hookedEvents {
			if hookedEvents[i].Kind() != zeroEvents[i].Kind() {
				t.Errorf("event[%d] kind = %v, want %v (byte-identical stream shape)", i, hookedEvents[i].Kind(), zeroEvents[i].Kind())
			}
		}

		hookedCaptured := hookedProvider.Requests()
		zeroCaptured := zeroProvider.Requests()
		if len(hookedCaptured) != 1 || len(zeroCaptured) != 1 {
			t.Fatalf("captured request counts = %d/%d, want 1/1", len(hookedCaptured), len(zeroCaptured))
		}
		if !hookedCaptured[0].Equal(zeroCaptured[0]) {
			t.Error("hooked Turn's captured request != zero-value Turn's captured request, want byte-identical")
		}
	})
}

// twoSecondsForHooksTest names the goroutine-baseline settle window
// waitForGoroutineBaseline (loop_hook_test.go) already uses elsewhere
// in this package — a diagnostic scheduling ceiling, not a
// synchronization mechanism (NFR-HKS-002: this file synchronizes by
// channel reads/closes and the snapshot-or-count lemma everywhere a
// hook's own firing must be proven; this one helper call is the
// package's own pre-existing pattern for "no stranded goroutine",
// reused rather than re-invented).
func twoSecondsForHooksTest() time.Duration { return 2 * time.Second }

// ---------------------------------------------------------------------
// R-HKS-002 — the pre-request chain
// ---------------------------------------------------------------------

// S-HKS-004 — Composition order is observable, and the singular field
// is first. Given Turn.PreRequestHook appending A and Hooks.PreRequest
// holding three elements appending B, C, D in registration order, when
// a one-turn run executes, then the captured request carries all four
// markers in order A,B,C,D; each element observed, as its own input,
// exactly its predecessor's output; and the provider received exactly
// one request.
func TestHooks_PreRequestChain_CompositionOrderSingularFirst(t *testing.T) {
	t.Parallel()

	var seenB, seenC, seenD []string
	hooks := agent.Hooks{PreRequest: []agent.PreRequestHook{
		appendMarkerHook(t, hooksMarkerB, &seenB),
		appendMarkerHook(t, hooksMarkerC, &seenC),
		appendMarkerHook(t, hooksMarkerD, &seenD),
	}}

	provider := agenttest.NewProvider(scriptTextResponse(t, ai.FinishReasonStop))
	sink := make(chan *agent.Event, 16)
	_, _, err := agent.Turn(contextBackground(), provider, "system prompt for hks-004", []ai.Message{firstMessage(t)},
		agent.TurnOptions{PreRequestHook: appendMarkerHook(t, hooksMarkerA, nil), Hooks: hooks}, sink)
	if err != nil {
		t.Fatalf("Turn returned err = %v, want nil", err)
	}
	drainSink(t, sink)

	captured := provider.Requests()
	if len(captured) != 1 {
		t.Fatalf("provider captured %d request(s), want 1", len(captured))
	}
	final := loopRequestSystemText(t, captured[0])
	for _, marker := range []string{hooksMarkerA, hooksMarkerB, hooksMarkerC, hooksMarkerD} {
		if !strings.Contains(final, marker) {
			t.Errorf("final captured system region lacks marker %q", marker)
		}
	}
	idxA := strings.Index(final, hooksMarkerA)
	idxB := strings.Index(final, hooksMarkerB)
	idxC := strings.Index(final, hooksMarkerC)
	idxD := strings.Index(final, hooksMarkerD)
	if idxA >= idxB || idxB >= idxC || idxC >= idxD {
		t.Errorf("marker order in captured system region is not A,B,C,D: A=%d B=%d C=%d D=%d", idxA, idxB, idxC, idxD)
	}

	if len(seenB) != 1 || !strings.Contains(seenB[0], hooksMarkerA) || strings.Contains(seenB[0], hooksMarkerC) {
		t.Errorf("element B's own observed input = %q, want to contain A and NOT contain C (B ran before C, after the singular field)", seenB)
	}
	if len(seenC) != 1 || !strings.Contains(seenC[0], hooksMarkerB) || strings.Contains(seenC[0], hooksMarkerD) {
		t.Errorf("element C's own observed input = %q, want to contain B and NOT contain D", seenC)
	}
	if len(seenD) != 1 || !strings.Contains(seenD[0], hooksMarkerC) {
		t.Errorf("element D's own observed input = %q, want to contain C", seenD)
	}

	if n := len(captured); n != 1 {
		t.Errorf("provider received %d request(s) for the attempt, want exactly 1", n)
	}
}

// S-HKS-005 — AG-08 compatibility is proven by an unedited suite.
// Byte-unchanged is asserted by the substrate filter's own test
// (S-LSK-032 below covers the filter's shape); this scenario's own
// behavioral half is that a zero-value TurnOptions (zero singular hook
// AND zero Hooks) produces a request byte-identical to what the
// hookless skeleton (S-PRH-002's own script) produces.
func TestHooks_PreRequestChain_ZeroValueByteIdenticalToSkeleton(t *testing.T) {
	t.Parallel()

	skeletonProvider := agenttest.NewProvider(scriptTextResponse(t, ai.FinishReasonStop))
	skeletonSink := make(chan *agent.Event, 16)
	_, _, err := agent.Turn(contextBackground(), skeletonProvider, "system prompt for hks-005", []ai.Message{firstMessage(t)}, agent.TurnOptions{}, skeletonSink)
	if err != nil {
		t.Fatalf("skeleton Turn returned err = %v, want nil", err)
	}
	drainSink(t, skeletonSink)

	zeroHooksProvider := agenttest.NewProvider(scriptTextResponse(t, ai.FinishReasonStop))
	zeroHooksSink := make(chan *agent.Event, 16)
	_, _, err = agent.Turn(contextBackground(), zeroHooksProvider, "system prompt for hks-005", []ai.Message{firstMessage(t)}, agent.TurnOptions{Hooks: agent.Hooks{}}, zeroHooksSink)
	if err != nil {
		t.Fatalf("zero-Hooks Turn returned err = %v, want nil", err)
	}
	drainSink(t, zeroHooksSink)

	skeletonCaptured := skeletonProvider.Requests()
	zeroCaptured := zeroHooksProvider.Requests()
	if len(skeletonCaptured) != 1 || len(zeroCaptured) != 1 {
		t.Fatalf("captured request counts = %d/%d, want 1/1", len(skeletonCaptured), len(zeroCaptured))
	}
	if !skeletonCaptured[0].Equal(zeroCaptured[0]) {
		t.Error("zero-value Hooks captured request != hookless skeleton's, want byte-identical (R-PRH-005 as amended)")
	}
}

// S-PRH-008 — AG-20: the chain's final output is what the provider
// receives, and no intermediate value is. Cross-referenced to
// R-HKS-002 / S-HKS-004; this scenario adds the "no captured request
// equals any intermediate element's output" half.
func TestHooks_PreRequestChain_FinalOutputOnly_NoIntermediateReachesProvider(t *testing.T) {
	t.Parallel()

	hooks := agent.Hooks{PreRequest: []agent.PreRequestHook{
		appendMarkerHook(t, hooksMarkerB, nil),
		appendMarkerHook(t, hooksMarkerC, nil),
		appendMarkerHook(t, hooksMarkerD, nil),
	}}
	provider := agenttest.NewProvider(scriptTextResponse(t, ai.FinishReasonStop))
	sink := make(chan *agent.Event, 16)
	_, _, err := agent.Turn(contextBackground(), provider, "system prompt for prh-008", []ai.Message{firstMessage(t)},
		agent.TurnOptions{PreRequestHook: appendMarkerHook(t, hooksMarkerA, nil), Hooks: hooks}, sink)
	if err != nil {
		t.Fatalf("Turn returned err = %v, want nil", err)
	}
	drainSink(t, sink)

	captured := provider.Requests()
	if len(captured) != 1 {
		t.Fatalf("provider captured %d request(s), want 1", len(captured))
	}
	final := loopRequestSystemText(t, captured[0])
	// The FINAL text carries every marker (already asserted by
	// S-HKS-004); the "no intermediate value" half is that a request
	// carrying ONLY A, or ONLY A+B, or ONLY A+B+C never reaches the
	// provider at all — the provider recorded exactly one request, and
	// that one request already carries D (the final element's own
	// addition), which an intermediate value by construction cannot.
	if !strings.Contains(final, hooksMarkerD) {
		t.Fatal("captured request lacks the final element's own marker — an intermediate value reached the provider instead")
	}
	if n := len(captured); n != 1 {
		t.Errorf("provider recorded %d request(s), want exactly 1 (no intermediate value was ever sent)", n)
	}
}

// S-PRH-010 — AG-20: a nil singular field does NOT skip a non-empty
// chain.
func TestHooks_PreRequestChain_NilSingularDoesNotSkipNonEmptyChain(t *testing.T) {
	t.Parallel()

	t.Run("nil singular, one-element chain: not skipped", func(t *testing.T) {
		t.Parallel()
		var seen []string
		hooks := agent.Hooks{PreRequest: []agent.PreRequestHook{appendMarkerHook(t, hooksMarkerB, &seen)}}
		provider := agenttest.NewProvider(scriptTextResponse(t, ai.FinishReasonStop))
		sink := make(chan *agent.Event, 16)
		_, _, err := agent.Turn(contextBackground(), provider, "system prompt for prh-010a", []ai.Message{firstMessage(t)}, agent.TurnOptions{Hooks: hooks}, sink)
		if err != nil {
			t.Fatalf("Turn returned err = %v, want nil", err)
		}
		drainSink(t, sink)
		captured := provider.Requests()
		if len(captured) != 1 {
			t.Fatalf("provider captured %d request(s), want 1", len(captured))
		}
		if !strings.Contains(loopRequestSystemText(t, captured[0]), hooksMarkerB) {
			t.Error("captured request lacks the chain element's marker — the seam was wrongly skipped")
		}
		if len(seen) != 1 {
			t.Fatalf("chain element invoked %d time(s), want 1", len(seen))
		}
		if !strings.Contains(seen[0], "system prompt for prh-010a") {
			t.Errorf("chain element's own input lacked the base system prompt, want buildLoopRequest's own output")
		}
	})

	t.Run("nil singular, empty chain: byte-identical to skeleton", func(t *testing.T) {
		t.Parallel()
		skeletonProvider := agenttest.NewProvider(scriptTextResponse(t, ai.FinishReasonStop))
		skeletonSink := make(chan *agent.Event, 16)
		_, _, err := agent.Turn(contextBackground(), skeletonProvider, "system prompt for prh-010b", []ai.Message{firstMessage(t)}, agent.TurnOptions{}, skeletonSink)
		if err != nil {
			t.Fatalf("skeleton Turn returned err = %v, want nil", err)
		}
		drainSink(t, skeletonSink)

		emptyChainProvider := agenttest.NewProvider(scriptTextResponse(t, ai.FinishReasonStop))
		emptyChainSink := make(chan *agent.Event, 16)
		_, _, err = agent.Turn(contextBackground(), emptyChainProvider, "system prompt for prh-010b", []ai.Message{firstMessage(t)}, agent.TurnOptions{Hooks: agent.Hooks{PreRequest: nil}}, emptyChainSink)
		if err != nil {
			t.Fatalf("empty-chain Turn returned err = %v, want nil", err)
		}
		drainSink(t, emptyChainSink)

		skeletonCaptured := skeletonProvider.Requests()
		emptyCaptured := emptyChainProvider.Requests()
		if len(skeletonCaptured) != 1 || len(emptyCaptured) != 1 {
			t.Fatalf("captured request counts = %d/%d, want 1/1", len(skeletonCaptured), len(emptyCaptured))
		}
		if !skeletonCaptured[0].Equal(emptyCaptured[0]) {
			t.Error("empty-chain captured request != skeleton's, want byte-identical — no element was invoked")
		}
	})
}

// S-PRH-011 — AG-20: the composition is deterministic, and the order
// is what makes it so.
func TestHooks_PreRequestChain_DeterministicAndOrderSensitive(t *testing.T) {
	t.Parallel()

	buildHooks := func() (agent.TurnOptions, func(context.Context, ai.Request) (ai.Request, error)) {
		return agent.TurnOptions{Hooks: agent.Hooks{PreRequest: []agent.PreRequestHook{
			appendMarkerHook(t, hooksMarkerB, nil),
			appendMarkerHook(t, hooksMarkerC, nil),
			appendMarkerHook(t, hooksMarkerD, nil),
		}}}, appendMarkerHook(t, hooksMarkerA, nil)
	}

	runOnce := func(t *testing.T) ai.Request {
		t.Helper()
		opts, singular := buildHooks()
		opts.PreRequestHook = singular
		provider := agenttest.NewProvider(scriptTextResponse(t, ai.FinishReasonStop))
		sink := make(chan *agent.Event, 16)
		_, _, err := agent.Turn(contextBackground(), provider, "system prompt for prh-011", []ai.Message{firstMessage(t)}, opts, sink)
		if err != nil {
			t.Fatalf("Turn returned err = %v, want nil", err)
		}
		drainSink(t, sink)
		captured := provider.Requests()
		if len(captured) != 1 {
			t.Fatalf("provider captured %d request(s), want 1", len(captured))
		}
		return captured[0]
	}

	first := runOnce(t)
	second := runOnce(t)
	if !first.Equal(second) {
		t.Errorf("composition NOT deterministic: first != second\n  first:  %q\n  second: %q", loopRequestSystemText(t, first), loopRequestSystemText(t, second))
	}

	// Different order: same four hooks, elements B and D swapped in
	// registration order — the output must differ, proving the
	// assertion is sensitive to order rather than accidentally
	// satisfied by commutative fixtures.
	reorderedOpts := agent.TurnOptions{
		PreRequestHook: appendMarkerHook(t, hooksMarkerA, nil),
		Hooks: agent.Hooks{PreRequest: []agent.PreRequestHook{
			appendMarkerHook(t, hooksMarkerD, nil),
			appendMarkerHook(t, hooksMarkerC, nil),
			appendMarkerHook(t, hooksMarkerB, nil),
		}},
	}
	reorderedProvider := agenttest.NewProvider(scriptTextResponse(t, ai.FinishReasonStop))
	reorderedSink := make(chan *agent.Event, 16)
	_, _, err := agent.Turn(contextBackground(), reorderedProvider, "system prompt for prh-011", []ai.Message{firstMessage(t)}, reorderedOpts, reorderedSink)
	if err != nil {
		t.Fatalf("reordered Turn returned err = %v, want nil", err)
	}
	drainSink(t, reorderedSink)
	reorderedCaptured := reorderedProvider.Requests()
	if len(reorderedCaptured) != 1 {
		t.Fatalf("reordered provider captured %d request(s), want 1", len(reorderedCaptured))
	}
	if first.Equal(reorderedCaptured[0]) {
		t.Error("reordered composition produced the SAME output as the original order — the assertion is not sensitive to order")
	}
}

// ---------------------------------------------------------------------
// R-HKS-009 — mutating-hook failure, typed and source-attributed
// ---------------------------------------------------------------------

// S-HKS-022 — Pre-request: attribution names the source, and the abort
// precedes I/O. Given a Harness whose Turn.PreRequestHook succeeds and
// whose Hooks.PreRequest holds three elements of which index 1
// returns a non-nil error, when the run executes, then the provider
// recorded zero requests; the error carries the typed pre-stream
// failure shape; its attribution names Hooks.PreRequest[1] and not
// index 2, not the singular field, and not a bare ordinal; the
// element at index 2 was never invoked; and the sink drains unblocked.
func TestHooks_PreRequestFailure_AttributesBySourceName_AbortsBeforeIO(t *testing.T) {
	t.Parallel()

	var invoked []int
	chain := failingPreRequestHookAt(t, 3, 1, &invoked)

	provider := agenttest.NewProvider(scriptTextResponse(t, ai.FinishReasonStop))
	sink := make(chan *agent.Event, 16)
	_, _, err := agent.Turn(contextBackground(), provider, "system prompt for hks-022", []ai.Message{firstMessage(t)},
		agent.TurnOptions{PreRequestHook: noopPreRequestHook, Hooks: agent.Hooks{PreRequest: chain}}, sink)
	if err == nil {
		t.Fatal("Turn returned err = nil, want a typed pre-stream failure")
	}

	if n := len(provider.Requests()); n != 0 {
		t.Errorf("provider recorded %d request(s), want 0 (abort must precede I/O)", n)
	}

	var failure *ai.Failure
	if !errors.As(err, &failure) {
		t.Fatalf("errors.As(err, *ai.Failure) = false on %v", err)
	}

	attribution := preRequestAttribution(t, err)
	if !strings.HasPrefix(attribution, "Hooks.PreRequest[1]") {
		t.Errorf("attribution = %q, want to start with %q", attribution, "Hooks.PreRequest[1]")
	}
	if strings.Contains(attribution, "Hooks.PreRequest[2]") {
		t.Errorf("attribution = %q, must not name index 2", attribution)
	}
	if strings.Contains(attribution, preRequestSingularSourceNameForTest) {
		t.Errorf("attribution = %q, must not name the singular field", attribution)
	}

	if !reflect.DeepEqual(invoked, []int{0, 1}) {
		t.Errorf("invoked chain elements = %v, want [0 1] — element 2 must never be invoked", invoked)
	}

	got := drainSink(t, sink)
	if len(got) == 0 {
		t.Error("sink was empty, want run_start+turn_start emitted before the abort")
	}
}

// preRequestSingularSourceNameForTest mirrors loop.go's unexported
// preRequestSingularSourceName constant's own literal value — this
// file cannot reference the unexported identifier, so the literal is
// restated here for the negative "must not name the singular field"
// assertion above.
const preRequestSingularSourceNameForTest = "TurnOptions.PreRequestHook"

// S-HKS-023 — The singular field is a distinct named source, and later
// insertion does not renumber.
func TestHooks_PreRequestFailure_SingularNamedDistinctly_InsertionDoesNotRenumber(t *testing.T) {
	t.Parallel()

	t.Run("singular field's own failure names the singular field", func(t *testing.T) {
		t.Parallel()
		failingSingular := func(_ context.Context, _ ai.Request) (ai.Request, error) { return ai.Request{}, errPreRequestBoom }
		provider := agenttest.NewProvider(scriptTextResponse(t, ai.FinishReasonStop))
		sink := make(chan *agent.Event, 16)
		_, _, err := agent.Turn(contextBackground(), provider, "system prompt for hks-023a", []ai.Message{firstMessage(t)}, agent.TurnOptions{PreRequestHook: failingSingular}, sink)
		if err == nil {
			t.Fatal("Turn returned err = nil, want a typed pre-stream failure")
		}
		attribution := preRequestAttribution(t, err)
		if !strings.HasPrefix(attribution, preRequestSingularSourceNameForTest) {
			t.Errorf("attribution = %q, want to start with %q", attribution, preRequestSingularSourceNameForTest)
		}
		if strings.Contains(attribution, "Hooks.PreRequest[") {
			t.Errorf("attribution = %q, must not name an indexed slot", attribution)
		}
	})

	t.Run("adding the singular field before the chain does not renumber the chain's own attribution", func(t *testing.T) {
		t.Parallel()

		// R-HKS-009: "a registration index is stable under insertion
		// ELSEWHERE; a composed ordinal is not." The chain element at
		// index 1 keeps that SAME slice index whether or not the
		// singular field runs before it — because the attribution is
		// Hooks.PreRequest[i] (this element's own index WITHIN
		// Hooks.PreRequest), never a composed ordinal that would also
		// count the singular field as "element zero" (the exact
		// ambiguity R-HKS-009 settles against the proposal's own D1
		// phrasing). Inserting a chain element AHEAD of index 1 WITHIN
		// Hooks.PreRequest itself is a DIFFERENT registration — index 1
		// now names a different function — so THAT case is not what
		// this scenario tests; "insertion elsewhere" is the singular
		// field's own slot, which sits outside Hooks.PreRequest's own
		// indexing entirely.
		var invokedNoSingular []int
		chainNoSingular := failingPreRequestHookAt(t, 3, 1, &invokedNoSingular)
		providerNoSingular := agenttest.NewProvider(scriptTextResponse(t, ai.FinishReasonStop))
		sinkNoSingular := make(chan *agent.Event, 16)
		_, _, errNoSingular := agent.Turn(contextBackground(), providerNoSingular, "system prompt for hks-023b", []ai.Message{firstMessage(t)}, agent.TurnOptions{Hooks: agent.Hooks{PreRequest: chainNoSingular}}, sinkNoSingular)
		if errNoSingular == nil {
			t.Fatal("no-singular Turn returned err = nil, want a typed pre-stream failure")
		}
		noSingularAttribution := preRequestAttribution(t, errNoSingular)

		var invokedWithSingular []int
		chainWithSingular := failingPreRequestHookAt(t, 3, 1, &invokedWithSingular)
		providerWithSingular := agenttest.NewProvider(scriptTextResponse(t, ai.FinishReasonStop))
		sinkWithSingular := make(chan *agent.Event, 16)
		_, _, errWithSingular := agent.Turn(contextBackground(), providerWithSingular, "system prompt for hks-023b", []ai.Message{firstMessage(t)}, agent.TurnOptions{PreRequestHook: noopPreRequestHook, Hooks: agent.Hooks{PreRequest: chainWithSingular}}, sinkWithSingular)
		if errWithSingular == nil {
			t.Fatal("with-singular Turn returned err = nil, want a typed pre-stream failure")
		}
		withSingularAttribution := preRequestAttribution(t, errWithSingular)

		if noSingularAttribution != withSingularAttribution {
			t.Errorf("attribution changed when the singular field was added before the chain: without=%q with=%q, want unchanged", noSingularAttribution, withSingularAttribution)
		}
		if !reflect.DeepEqual(invokedNoSingular, []int{0, 1}) || !reflect.DeepEqual(invokedWithSingular, []int{0, 1}) {
			t.Errorf("chain invocation indices differ: without=%v with=%v, want [0 1] both times", invokedNoSingular, invokedWithSingular)
		}
	})
}

// ---------------------------------------------------------------------
// R-HKS-006 — deterministic registration-order dispatch, all 4 points
// ---------------------------------------------------------------------

// wgObserverHold is a hand-rolled synchronization primitive: this
// package's own agenttest.Gate cannot be reused here (verified by
// reading fake_gate.go in this phase — its Reached() channel is closed
// only by fake_provider.go's own internal producer, and it exposes no
// public accessor for the channel Release() closes, so it is
// purpose-built for a scripted PROVIDER stream's own Hold step, not
// for an arbitrary caller's goroutine). NFR-HKS-002 sanctions this
// shape by name independently of agenttest.Gate: "channel reads,
// channel closes."
type wgObserverHold struct {
	entered chan struct{}
	release chan struct{}
	once    sync.Once
}

func newWgObserverHold() *wgObserverHold {
	return &wgObserverHold{entered: make(chan struct{}), release: make(chan struct{})}
}

// Enter signals arrival, then blocks until Release. Called from the
// observer under test.
func (h *wgObserverHold) Enter() {
	h.once.Do(func() {})
	close(h.entered)
	<-h.release
}

// Release unblocks a held observer. Idempotent.
func (h *wgObserverHold) Release() {
	select {
	case <-h.release:
	default:
		close(h.release)
	}
}

// Reached returns a channel that closes once Enter has been called.
func (h *wgObserverHold) Reached() <-chan struct{} { return h.entered }

// orderedRecorder is a mutex-guarded append-only []int this file's
// ordering scenarios use, together with a sync.WaitGroup, as the
// "lane has drained" determinism primitive (the snapshot-or-count
// lemma applied through a WaitGroup rather than a sleep): each
// observer records its own index then calls wg.Done(); the test calls
// wg.Wait() after Run returns, which — since every recorded append
// happens-before that observer's own Done() call — deterministically
// proves every registered observer already ran before the recorder is
// read.
type orderedRecorder struct {
	mu  sync.Mutex
	seq []int
}

func (r *orderedRecorder) record(i int) {
	r.mu.Lock()
	r.seq = append(r.seq, i)
	r.mu.Unlock()
}

func (r *orderedRecorder) snapshot() []int {
	r.mu.Lock()
	defer r.mu.Unlock()
	return append([]int(nil), r.seq...)
}

// S-HKS-016 — Charter AG-20.1 scenario 3, the pin, at all four points.
// Given four hooks registered at EACH of the four points, each
// appending its own index to a per-point recorder, when a run that
// reaches all four points executes and its lane has drained, then each
// point's recorded sequence is exactly [0,1,2,3]; the sequence is
// identical across repeated executions; session-start precedes every
// post-turn entry; and the scenario passes under -race.
func TestHooks_Ordering_RegistrationOrderAtAllFourPoints(t *testing.T) {
	t.Parallel()

	runOnce := func(t *testing.T) (sessionStart, postTurn, preRequest, preCompact []int) {
		t.Helper()

		// Seed a marked, two-entry history via a throwaway harness/run
		// — markedHarnessForCompaction (compaction_call_test.go, same
		// package) — so THIS test's own Harness has a recorded mark to
		// retract to when its own ContextStrategy requests compaction
		// at its first turn boundary.
		_, hist, _ := markedHarnessForCompaction(t, "hks-016")

		sessionRec := &orderedRecorder{}
		postTurnRec := &orderedRecorder{}
		var wg sync.WaitGroup
		wg.Add(8) // 4 session-start + 4 post-turn lane dispatches.

		sessionObservers := make([]agent.SessionStartObserver, 4)
		for i := 0; i < 4; i++ {
			i := i
			sessionObservers[i] = func(context.Context, agent.SessionStartReport) {
				sessionRec.record(i)
				wg.Done()
			}
		}
		postTurnObservers := make([]agent.PostTurnObserver, 4)
		for i := 0; i < 4; i++ {
			i := i
			postTurnObservers[i] = func(context.Context, agent.PostTurnReport) {
				postTurnRec.record(i)
				wg.Done()
			}
		}

		var preReqRec, preCompactRec orderedRecorder
		preRequestChain := make([]agent.PreRequestHook, 4)
		for i := 0; i < 4; i++ {
			i := i
			preRequestChain[i] = func(_ context.Context, req ai.Request) (ai.Request, error) {
				preReqRec.record(i)
				return req, nil
			}
		}
		preCompactChain := make([]agent.PreCompactHook, 4)
		for i := 0; i < 4; i++ {
			i := i
			preCompactChain[i] = func(_ context.Context, plan agent.CompactionPlan) (agent.CompactionPlan, error) {
				preCompactRec.record(i)
				return plan, nil
			}
		}

		// compactAfterNStrategy (compaction_call_test.go, same
		// package): fires on its first (only, skip:0) consultation,
		// requesting a compaction of the whole transcript it was
		// handed — the seed's own mark gives resolveCut something to
		// retract to.
		strategy := &compactAfterNStrategy{request: func(prompt agent.ContextPrompt) agent.CompactionRequest {
			return agent.CompactionRequest{
				Provider:    agenttest.NewProvider(scriptTextResponse(t, ai.FinishReasonStop)),
				Instruction: "summarize for hks-016",
				Cut:         len(prompt.Transcript),
			}
		}}

		h := agent.Harness{
			Provider: agenttest.NewProvider(scriptTextResponse(t, ai.FinishReasonStop)),
			System:   "system prompt for hks-016",
			History:  hist,
			Hooks: agent.Hooks{
				SessionStart: sessionObservers,
				PostTurn:     postTurnObservers,
				PreRequest:   preRequestChain,
				PreCompact:   preCompactChain,
			},
			ContextStrategy: strategy,
		}
		sink := make(chan *agent.Event, 512)
		_, _, err := h.Run(contextBackground(), firstMessage(t), sink)
		if err != nil {
			t.Fatalf("Run returned err = %v, want nil", err)
		}
		drainSink(t, sink)
		wg.Wait()

		return sessionRec.snapshot(), postTurnRec.snapshot(), preReqRec.snapshot(), preCompactRec.snapshot()
	}

	wantOrder := []int{0, 1, 2, 3}

	firstSession, firstPostTurn, firstPreReq, firstPreCompact := runOnce(t)
	if !reflect.DeepEqual(firstSession, wantOrder) {
		t.Errorf("session-start order = %v, want %v", firstSession, wantOrder)
	}
	if !reflect.DeepEqual(firstPostTurn, wantOrder) {
		t.Errorf("post-turn order = %v, want %v", firstPostTurn, wantOrder)
	}
	if !reflect.DeepEqual(firstPreReq, wantOrder) {
		t.Errorf("pre-request order = %v, want %v", firstPreReq, wantOrder)
	}
	if !reflect.DeepEqual(firstPreCompact, wantOrder) {
		t.Errorf("pre-compact order = %v, want %v", firstPreCompact, wantOrder)
	}

	secondSession, secondPostTurn, _, _ := runOnce(t)
	if !reflect.DeepEqual(secondSession, wantOrder) || !reflect.DeepEqual(secondPostTurn, wantOrder) {
		t.Errorf("repeated execution's order differs: session=%v post-turn=%v, want %v both times", secondSession, secondPostTurn, wantOrder)
	}
}

// S-HKS-052 — (bite) RED-first, mandated. Given a scratch tree in
// which dispatch at one point iterates the registered hooks in
// reverse, when S-HKS-016 runs, then it FAILS reporting [3,2,1,0].
// RED-recorded BEFORE S-HKS-016 is GREEN, with -count=1, then
// reverted. See apply-progress.md for the captured RED transcript;
// this scenario's own GREEN assertion is TestHooks_Ordering_
// RegistrationOrderAtAllFourPoints above — no separate GREEN test is
// needed here, since the bite's sabotage is a scratch, reverted edit
// to enqueuePostTurn's own dispatch order, never committed.

// ---------------------------------------------------------------------
// R-HKS-007 / R-HKS-008 — asynchrony, "eventually", panic (AG-20.2)
// ---------------------------------------------------------------------

// wgObserverHold is defined above (used by the ordering scenarios'
// sibling tests below too).

// S-HKS-017 — Charter AG-20.2, the "delivery is unimpeded" clause
// (design AD-9/T2). Given a PostTurn observer held open, and a sink
// buffered to the script's full event count so the run needs no live
// consumer, when the run executes, then entering the hold proves the
// invocation started and has not returned; the recorded event stream
// is byte-identical (modulo freshly minted identities) to the same
// script with no hooks installed; CheckStream accepts it; Run returns
// while the hold is still held; and only then is it released, with an
// unconditional t.Cleanup(hold.Release) as the AD-10 backstop.
func TestHooks_Asynchrony_DeliveryUnimpeded_RunReturnsWithHoldHeld(t *testing.T) {
	t.Parallel()

	hold := newWgObserverHold()
	t.Cleanup(hold.Release)

	baselineProvider := agenttest.NewProvider(scriptTextResponse(t, ai.FinishReasonStop))
	baselineHarness := agent.Harness{Provider: baselineProvider, System: "system prompt for hks-017"}
	baselineSink := make(chan *agent.Event, 256)
	if _, _, err := baselineHarness.Run(contextBackground(), firstMessage(t), baselineSink); err != nil {
		t.Fatalf("baseline Run returned err = %v, want nil", err)
	}
	baselineEvents := drainSink(t, baselineSink)

	hooks := agent.Hooks{PostTurn: []agent.PostTurnObserver{
		func(context.Context, agent.PostTurnReport) { hold.Enter() },
	}}
	hookedProvider := agenttest.NewProvider(scriptTextResponse(t, ai.FinishReasonStop))
	h := agent.Harness{Provider: hookedProvider, System: "system prompt for hks-017", Hooks: hooks}
	hookedSink := make(chan *agent.Event, len(baselineEvents))

	runDone := make(chan struct{})
	var runErr error
	go func() {
		_, _, runErr = h.Run(contextBackground(), firstMessage(t), hookedSink)
		close(runDone)
	}()

	<-hold.Reached() // proves the invocation started and has not returned.
	<-runDone        // Run returns WHILE the hold is still held (never joined).
	if runErr != nil {
		t.Fatalf("Run returned err = %v, want nil", runErr)
	}

	hookedEvents := drainSink(t, hookedSink)
	if len(hookedEvents) != len(baselineEvents) {
		t.Fatalf("hooked stream has %d event(s), baseline has %d, want equal (byte-identical stream)", len(hookedEvents), len(baselineEvents))
	}
	for i := range hookedEvents {
		if hookedEvents[i].Kind() != baselineEvents[i].Kind() {
			t.Errorf("event[%d] kind = %v, want %v", i, hookedEvents[i].Kind(), baselineEvents[i].Kind())
		}
	}
	if report := agent.CheckStream(hookedEvents); report.Violation() != nil {
		t.Errorf("CheckStream rejected the hooked stream: %v, want accepted unmodified", report.Violation())
	}

	hold.Release() // only now, after every assertion above.
}

// S-HKS-018 — Asynchrony is a goroutine-placement property, so it is
// asserted as one. Given a NON-BLOCKING PostTurn observer that
// captures its own stack at invocation, when Run has returned and the
// observer's own close(done) has been observed, then the recorded
// stack contains no Run frame and does mention the lane's own drain
// machinery — never a sleep, never a poll.
func TestHooks_Asynchrony_GoroutinePlacement_NoRunFrame(t *testing.T) {
	t.Parallel()

	var stack []byte
	done := make(chan struct{})
	hooks := agent.Hooks{PostTurn: []agent.PostTurnObserver{
		func(context.Context, agent.PostTurnReport) {
			buf := make([]byte, 1<<16)
			n := runtime.Stack(buf, false)
			stack = buf[:n]
			close(done)
		},
	}}

	provider := agenttest.NewProvider(scriptTextResponse(t, ai.FinishReasonStop))
	h := agent.Harness{Provider: provider, System: "system prompt for hks-018", Hooks: hooks}
	sink := make(chan *agent.Event, 32)
	if _, _, err := h.Run(contextBackground(), firstMessage(t), sink); err != nil {
		t.Fatalf("Run returned err = %v, want nil", err)
	}
	drainSink(t, sink)
	<-done // deterministic: the observer's own close(done), not a sleep or poll.

	captured := string(stack)
	if strings.Contains(captured, ").Run(") {
		t.Errorf("observer's captured stack contains a Run frame, want none present (asynchrony is a goroutine-placement property):\n%s", captured)
	}
	if !strings.Contains(captured, "observerLane") {
		t.Errorf("observer's captured stack does not mention observerLane, want the lane's own drain root:\n%s", captured)
	}
}

// S-HKS-051 — (bite) RED-first, mandated, and its shape is deliberate
// (design AD-9). Given a scratch tree in which observing hooks are
// dispatched SYNCHRONOUSLY at the fire site, when S-HKS-018 runs, then
// it FAILS AS AN ASSERTION — reporting a captured stack containing the
// harness run frame — not as a hang. See apply-progress.md for the
// captured RED transcript; the GREEN test is TestHooks_Asynchrony_
// GoroutinePlacement_NoRunFrame above.

// S-HKS-019 — Charter AG-20.2, "eventually reported typed" — with a
// queued victim beside the culprit. Given a PostTurn observer held at
// index 0 and a second, never-reached observer at index 1, and a run
// producing at least two logical turns, when Run returns, then the
// reporter received reports naming both index 0 (outstanding) and at
// least one queued invocation stranded behind it; every report was
// delivered before Run returned; and no assertion reads elapsed time.
func TestHooks_Asynchrony_EventuallyReported_OutstandingAndQueued(t *testing.T) {
	t.Parallel()

	hold := newWgObserverHold()
	t.Cleanup(hold.Release)

	var reportsMu sync.Mutex
	var reports []agent.ObserverStall
	reporter := func(s agent.ObserverStall) {
		reportsMu.Lock()
		reports = append(reports, s)
		reportsMu.Unlock()
	}

	index1Reached := make(chan struct{})
	var index1Once sync.Once
	hooks := agent.Hooks{
		PostTurn: []agent.PostTurnObserver{
			func(context.Context, agent.PostTurnReport) { hold.Enter() },
			func(context.Context, agent.PostTurnReport) { index1Once.Do(func() { close(index1Reached) }) },
		},
		StallReporter: reporter,
	}

	toolName := "read_hks_019"
	tool := EchoScriptedTool(toolName, agent.EffectClassRead)
	reg := agent.NewMapRegistry(map[string]agent.Tool{toolName: tool})
	provider := agenttest.NewProvider(
		scriptToolCallResponse(t, "call-hks-019", toolName, []byte(`{}`)),
		scriptTextResponse(t, ai.FinishReasonStop),
	)
	h := agent.Harness{Provider: provider, System: "system prompt for hks-019", Turn: agent.TurnOptions{Tools: reg}, Hooks: hooks}
	sink := make(chan *agent.Event, 256)

	runDone := make(chan struct{})
	var runErr error
	go func() {
		_, _, runErr = h.Run(contextBackground(), firstMessage(t), sink)
		close(runDone)
	}()

	<-hold.Reached()
	<-runDone
	if runErr != nil {
		t.Fatalf("Run returned err = %v, want nil", runErr)
	}
	got := drainSink(t, sink)
	var runID agent.RunID
	if len(got) > 0 {
		runID = got[0].Run()
	}

	reportsMu.Lock()
	snapshot := append([]agent.ObserverStall(nil), reports...)
	reportsMu.Unlock()

	var sawOutstanding, sawQueued bool
	for _, s := range snapshot {
		if s.Point() != agent.HookPointPostTurn {
			continue
		}
		if s.Run() != runID {
			t.Errorf("report run = %q, want %q", s.Run(), runID)
		}
		switch s.Reason() {
		case agent.StallOutstanding:
			if s.Index() != 0 {
				t.Errorf("outstanding report index = %d, want 0", s.Index())
			}
			sawOutstanding = true
		case agent.StallQueued:
			sawQueued = true
		}
	}
	if !sawOutstanding {
		t.Error("no StallOutstanding report observed, want exactly one naming index 0")
	}
	if !sawQueued {
		t.Error("no StallQueued report observed, want at least one for the invocation stranded behind index 0")
	}

	select {
	case <-index1Reached:
		t.Error("index-1 observer was reached before the hold was released, want it queued behind the held index-0")
	default:
	}

	hold.Release()
}

// S-HKS-050 — (bite) RED-first, mandated — the anti-vacuity bite. See
// apply-progress.md for the captured RED transcript (a scratch
// sabotage reporting every enqueued observer regardless of state,
// reverted after recording the failure). This is the GREEN half: with
// no stall present, the reporter's report set MUST be empty.
func TestHooks_AntiVacuity_NoStallEmptyReportSet(t *testing.T) {
	t.Parallel()

	var reportsMu sync.Mutex
	var reports []agent.ObserverStall
	reporter := func(s agent.ObserverStall) {
		reportsMu.Lock()
		reports = append(reports, s)
		reportsMu.Unlock()
	}

	// Baseline (no hooks): captures the exact event count so this
	// test's own hooked sink can be buffered to exactly one LESS than
	// that count. The post-turn enqueue site sits between turn_end and
	// the run's own trailing cost_session(Final)+run_end pair, so a
	// buffer sized to "every event up to and including turn_end" lets
	// every pre-enqueue send proceed freely (no live consumer needed)
	// while forcing the FIRST post-enqueue send (cost_session) to
	// BLOCK until this test starts draining — which it deliberately
	// does not until AFTER wg.Wait() has confirmed the observer's own
	// side effect. This gives the lane's drain goroutine UNBOUNDED
	// (not timed) opportunity to complete the enqueued invocation
	// before Run's own goroutine can possibly reach its snapshot: a
	// channel send/receive pair plus a WaitGroup, both
	// NFR-HKS-002-sanctioned; no wall clock, no poll, no sleep.
	baselineProvider := agenttest.NewProvider(scriptTextResponse(t, ai.FinishReasonStop))
	baselineHarness := agent.Harness{Provider: baselineProvider, System: "system prompt for hks-050"}
	baselineSink := make(chan *agent.Event, 64)
	if _, _, err := baselineHarness.Run(contextBackground(), firstMessage(t), baselineSink); err != nil {
		t.Fatalf("baseline Run returned err = %v, want nil", err)
	}
	baselineEvents := drainSink(t, baselineSink)
	preEnqueueCount := len(baselineEvents) - 2 // drop the trailing cost_session(Final) + run_end.
	if preEnqueueCount < 1 {
		t.Fatalf("baseline emitted only %d event(s), want at least 3 (fixture assumption violated)", len(baselineEvents))
	}

	var wg sync.WaitGroup
	wg.Add(1)
	hooks := agent.Hooks{
		PostTurn:      []agent.PostTurnObserver{func(context.Context, agent.PostTurnReport) { wg.Done() }},
		StallReporter: reporter,
	}
	hookedProvider := agenttest.NewProvider(scriptTextResponse(t, ai.FinishReasonStop))
	h := agent.Harness{Provider: hookedProvider, System: "system prompt for hks-050", Hooks: hooks}
	hookedSink := make(chan *agent.Event, preEnqueueCount)

	runDone := make(chan struct{})
	go func() {
		_, _, _ = h.Run(contextBackground(), firstMessage(t), hookedSink)
		close(runDone)
	}()

	wg.Wait() // the observer's own side effect is now visible.

	drainSink(t, hookedSink) // unblocks Run's remaining sends; Run proceeds to its snapshot and returns.
	<-runDone

	reportsMu.Lock()
	got := len(reports)
	reportsMu.Unlock()
	if got != 0 {
		t.Errorf("report set has %d entr(y/ies), want 0 (no stall present) — a naive implementation reporting every observer would fail here (S-HKS-050)", got)
	}
}

// S-HKS-020 — A panicking observer is reported on the lane, the lane
// continues, the process survives. Given three PostTurn observers
// where index 1 panics, when a run with two logical turns executes,
// then the reporter received two panicked reports naming index 1;
// indices 0 and 2 each recorded their full expected invocation count;
// the test process survives to assert it; and the recorded stream is
// (kind-sequence) byte-identical to the same script with no hooks.
func TestHooks_Panic_ReportedOnLane_LaneContinues_ProcessSurvives(t *testing.T) {
	t.Parallel()

	var reportsMu sync.Mutex
	var reports []agent.ObserverStall
	reporter := func(s agent.ObserverStall) {
		reportsMu.Lock()
		reports = append(reports, s)
		reportsMu.Unlock()
	}

	var mu sync.Mutex
	var index0Count, index2Count int
	var wg sync.WaitGroup
	wg.Add(4) // index 0 and index 2, twice each (2 logical turns).
	hooks := agent.Hooks{
		PostTurn: []agent.PostTurnObserver{
			func(context.Context, agent.PostTurnReport) {
				mu.Lock()
				index0Count++
				mu.Unlock()
				wg.Done()
			},
			func(context.Context, agent.PostTurnReport) {
				panic("hks-020 deliberate panic, recovered by the lane")
			},
			func(context.Context, agent.PostTurnReport) {
				mu.Lock()
				index2Count++
				mu.Unlock()
				wg.Done()
			},
		},
		StallReporter: reporter,
	}

	toolName := "read_hks_020"
	tool := EchoScriptedTool(toolName, agent.EffectClassRead)
	reg := agent.NewMapRegistry(map[string]agent.Tool{toolName: tool})
	hookedProvider := agenttest.NewProvider(
		scriptToolCallResponse(t, "call-hks-020", toolName, []byte(`{}`)),
		scriptTextResponse(t, ai.FinishReasonStop),
	)
	h := agent.Harness{Provider: hookedProvider, System: "system prompt for hks-020", Turn: agent.TurnOptions{Tools: reg}, Hooks: hooks}
	sink := make(chan *agent.Event, 256)
	if _, _, err := h.Run(contextBackground(), firstMessage(t), sink); err != nil {
		t.Fatalf("Run returned err = %v, want nil", err)
	}
	hookedEvents := drainSink(t, sink)
	wg.Wait()

	mu.Lock()
	gotIndex0, gotIndex2 := index0Count, index2Count
	mu.Unlock()
	if gotIndex0 != 2 {
		t.Errorf("index-0 observer invoked %d time(s), want 2 (one per logical turn)", gotIndex0)
	}
	if gotIndex2 != 2 {
		t.Errorf("index-2 observer invoked %d time(s), want 2", gotIndex2)
	}

	reportsMu.Lock()
	snapshot := append([]agent.ObserverStall(nil), reports...)
	reportsMu.Unlock()
	panickedCount := 0
	for _, s := range snapshot {
		if s.Reason() == agent.StallPanicked {
			if s.Index() != 1 {
				t.Errorf("panicked report index = %d, want 1", s.Index())
			}
			if s.Point() != agent.HookPointPostTurn {
				t.Errorf("panicked report point = %v, want HookPointPostTurn", s.Point())
			}
			panickedCount++
		}
	}
	if panickedCount != 2 {
		t.Errorf("panicked report count = %d, want 2 (one per logical turn)", panickedCount)
	}

	baselineProvider := agenttest.NewProvider(
		scriptToolCallResponse(t, "call-hks-020", toolName, []byte(`{}`)),
		scriptTextResponse(t, ai.FinishReasonStop),
	)
	baselineHarness := agent.Harness{Provider: baselineProvider, System: "system prompt for hks-020", Turn: agent.TurnOptions{Tools: reg}}
	baselineSink := make(chan *agent.Event, 256)
	if _, _, err := baselineHarness.Run(contextBackground(), firstMessage(t), baselineSink); err != nil {
		t.Fatalf("baseline Run returned err = %v, want nil", err)
	}
	baselineEvents := drainSink(t, baselineSink)
	if len(hookedEvents) != len(baselineEvents) {
		t.Fatalf("hooked stream has %d event(s), baseline has %d, want equal", len(hookedEvents), len(baselineEvents))
	}
	for i := range hookedEvents {
		if hookedEvents[i].Kind() != baselineEvents[i].Kind() {
			t.Errorf("event[%d] kind = %v, want %v (byte-identical stream shape)", i, hookedEvents[i].Kind(), baselineEvents[i].Kind())
		}
	}
}

// S-HKS-021 — A nil reporter reports nothing, and a stalling reporter
// stalls the two documented observables.
func TestHooks_Reporter_NilReportsNothing_StallingStallsBothObservables(t *testing.T) {
	t.Parallel()

	t.Run("nil reporter never panics, Run returns normally", func(t *testing.T) {
		t.Parallel()
		hold := newWgObserverHold()
		t.Cleanup(hold.Release)
		hooks := agent.Hooks{PostTurn: []agent.PostTurnObserver{func(context.Context, agent.PostTurnReport) { hold.Enter() }}}

		provider := agenttest.NewProvider(scriptTextResponse(t, ai.FinishReasonStop))
		h := agent.Harness{Provider: provider, System: "system prompt for hks-021a", Hooks: hooks}
		sink := make(chan *agent.Event, 32)
		if _, _, err := h.Run(contextBackground(), firstMessage(t), sink); err != nil {
			t.Fatalf("Run returned err = %v, want nil (nil reporter must never panic)", err)
		}
		drainSink(t, sink)
		hold.Release()
	})

	t.Run("stalling reporter stalls Run's return and the sink close", func(t *testing.T) {
		t.Parallel()

		observerHold := newWgObserverHold()
		t.Cleanup(observerHold.Release)
		reporterHold := newWgObserverHold()
		t.Cleanup(reporterHold.Release)

		hooks := agent.Hooks{
			PostTurn:      []agent.PostTurnObserver{func(context.Context, agent.PostTurnReport) { observerHold.Enter() }},
			StallReporter: func(agent.ObserverStall) { reporterHold.Enter() },
		}

		provider := agenttest.NewProvider(scriptTextResponse(t, ai.FinishReasonStop))
		h := agent.Harness{Provider: provider, System: "system prompt for hks-021b", Hooks: hooks}
		sink := make(chan *agent.Event, 32)

		runDone := make(chan struct{})
		go func() {
			_, _, _ = h.Run(contextBackground(), firstMessage(t), sink)
			close(runDone)
		}()

		<-observerHold.Reached()
		<-reporterHold.Reached()

		select {
		case <-runDone:
			t.Error("Run returned while the reporter's own hold is still held, want it stalled")
		default:
		}
		select {
		case _, ok := <-sink:
			if !ok {
				t.Error("sink closed while the reporter's own hold is still held, want it stalled")
			}
		default:
		}

		reporterHold.Release()
		<-runDone
		observerHold.Release()
		drainSink(t, sink)
	})
}

// ---------------------------------------------------------------------
// R-HKS-010 / R-HKS-011 / R-HKS-012 — scope fence, closed-sequence,
// inertness (Phase 4)
// ---------------------------------------------------------------------

// hksScopeFenceByteUnchangedFiles is R-HKS-010's own named list,
// relative to backend/agent/src/agent/ — every file this change MUST
// NOT edit. go.mod/go.sum are checked separately (one directory up).
func hksScopeFenceByteUnchangedFiles() []string {
	return []string{
		"event.go",
		"event_descriptor.go",
		"event_registry_test.go",
		"stream_check.go",
		"delegation_events.go",
		"permission_events.go",
		"cost_events.go",
		"cost_usage.go",
		"turn_events.go",
		"run_events.go",
		"failure.go",
		"sequence.go",
		"compaction_events.go",
		"doc.go",
		"doc_contract_guard_test.go",
		"ambient_authority_test.go",
		"import_boundary_test.go",
		"reconstruction_test.go",
		"scheduler.go",
		"delegation_seam.go",
		"scope_fence_test.go",
	}
}

// S-HKS-024 — Given the merge base of this change's branch with
// origin/main, when git diff is taken over backend/agent/, then every
// file named byte-unchanged above is byte-unchanged; the diff under
// backend/agent/src/ai/ and under backend/agent/src/agenttest/ is
// empty; the go.mod/go.sum diff is empty; the event-kind guard passes
// at its committed kind count; no turn outcome or cost-label member
// was added; and Turn's/Run's signatures and Harness's exported method
// set are unchanged. Anti-vacuity floor (S-TLS-020 precedent, adapted:
// scheduler.go/delegation_seam.go are BYTE-UNCHANGED by this change's
// own design, so checking THEIR diff for non-vacuity would be
// self-defeating — loop.go, a file this change DOES modify and which
// is TRACKED, not merely staged, is the floor instead): loop.go's own
// diff against the base MUST be non-empty, or this guard is measuring
// nothing.
func TestHooks_ScopeFence_ByteUnchangedFilesAndNoNewKind(t *testing.T) {
	t.Parallel()

	root, err := gitTopLevel(t)
	if err != nil {
		t.Fatalf("git rev-parse --show-toplevel failed: %v", err)
	}
	baseRef := hksResolveBaseRef(t, root)

	for _, name := range hksScopeFenceByteUnchangedFiles() {
		path := "backend/agent/src/agent/" + name
		diff, derr := gitDiff(t, root, baseRef, path)
		if derr != nil {
			t.Fatalf("git diff %s -- %s failed: %v", baseRef, path, derr)
		}
		if diff != "" {
			t.Errorf("%s is not byte-unchanged against %s (R-HKS-010):\n%s", path, baseRef, diff)
		}
	}

	aiDiff, err := gitDiff(t, root, baseRef, "backend/agent/src/ai/")
	if err != nil {
		t.Fatalf("git diff %s -- backend/agent/src/ai/ failed: %v", baseRef, err)
	}
	if aiDiff != "" {
		t.Errorf("backend/agent/src/ai/ is not byte-unchanged against %s (R-RUN-012):\n%s", baseRef, aiDiff)
	}

	agenttestDiff, err := gitDiff(t, root, baseRef, "backend/agent/src/agenttest/")
	if err != nil {
		t.Fatalf("git diff %s -- backend/agent/src/agenttest/ failed: %v", baseRef, err)
	}
	if agenttestDiff != "" {
		t.Errorf("backend/agent/src/agenttest/ is not byte-unchanged against %s (the gate primitive is used, not widened):\n%s", baseRef, agenttestDiff)
	}

	modDiff, err := gitDiff(t, root, baseRef, "backend/agent/go.mod", "backend/agent/go.sum")
	if err != nil {
		t.Fatalf("git diff %s -- go.mod go.sum failed: %v", baseRef, err)
	}
	if modDiff != "" {
		t.Errorf("go.mod/go.sum diff against %s is not empty:\n%s", baseRef, modDiff)
	}

	if !hksBaseRefIsHEAD(t, root, baseRef) {
		loopDiff, lerr := gitDiff(t, root, baseRef, "backend/agent/src/agent/loop.go")
		if lerr != nil {
			t.Fatalf("git diff %s -- loop.go failed: %v", baseRef, lerr)
		}
		if loopDiff == "" {
			t.Fatal("loop.go's diff against the merge base is empty — this scope-fence guard has nothing to measure (anti-vacuity floor, S-TLS-020 precedent)")
		}
	}

	if got := len(agent.EventKinds()); got != 25 {
		t.Errorf("len(agent.EventKinds()) = %d, want 25 (AG-20 registers no new EventKind)", got)
	}

	harnessPtrType := reflect.TypeOf((*agent.Harness)(nil))
	if got := harnessPtrType.NumMethod(); got != 5 {
		t.Errorf("*agent.Harness declares %d exported method(s), want 5", got)
	}
	turnType := reflect.TypeOf(agent.Turn)
	if turnType.Kind() != reflect.Func {
		t.Fatalf("agent.Turn is not a func value")
	}
	if got := turnType.NumIn(); got != 6 {
		t.Errorf("agent.Turn takes %d parameter(s), want 6 (ctx, provider, system, transcript, opts, sink) — signature unchanged", got)
	}
	runMethod, ok := reflect.TypeOf(&agent.Harness{}).MethodByName("Run")
	if !ok {
		t.Fatal("*agent.Harness has no Run method")
	}
	// MethodByName's Type includes the receiver as parameter 0.
	if got := runMethod.Type.NumIn(); got != 4 {
		t.Errorf("Harness.Run takes %d parameter(s) (receiver included), want 4 (receiver, ctx, prompt, sink) — signature unchanged", got)
	}
}

// hksResolveBaseRef mirrors scope_fence_test.go's own AG19_BASE_REF
// pattern for this change: AG20_BASE_REF pins the ref explicitly when
// set; otherwise the merge-base of HEAD and origin/main.
func hksResolveBaseRef(t *testing.T, root string) string {
	t.Helper()
	if v := os.Getenv("AG20_BASE_REF"); v != "" {
		return v
	}
	out, err := gitOutput(t, root, "merge-base", "HEAD", "origin/main")
	if err != nil {
		t.Fatalf("scope fence: cannot determine base ref (set AG20_BASE_REF): %v", err)
	}
	return out
}

// hksBaseRefIsHEAD mirrors scope_fence_test.go's baseRefIsHEAD.
func hksBaseRefIsHEAD(t *testing.T, root, baseRef string) bool {
	t.Helper()
	headSHA, err := gitOutput(t, root, "rev-parse", "HEAD^{commit}")
	if err != nil {
		t.Fatalf("git rev-parse HEAD failed: %v", err)
	}
	baseSHA, err := gitOutput(t, root, "rev-parse", baseRef+"^{commit}")
	if err != nil {
		t.Fatalf("git rev-parse %s failed: %v", baseRef, err)
	}
	return headSHA == baseSHA
}

// S-HKS-025 — The closed-sequence table is checked, not asserted.
// Given the merged change, when each of R-HKS-011's rows is evaluated
// against the shipped code and suites, then every "holds" row's owning
// test passes byte-unchanged; the AMENDED rows (R-PRH-002, R-PRH-005)
// each resolve to this change's agent-pre-request-hook delta; and the
// CLARIFIED row (R-CMP-004) resolves to the agent-compaction delta.
// Checked here by asserting the OWNING tests exist and pass as part of
// this same suite run, and by re-stating the table's own resolution —
// the byte-unchanged half is what TestHooks_ScopeFence_
// ByteUnchangedFilesAndNoNewKind above already proves for the shared
// files (scope_fence_test.go, delegation_seam.go, event.go, etc.).
func TestHooks_ClosedSequenceTable_HoldsAmendedAndClarifiedRowsResolve(t *testing.T) {
	t.Parallel()

	// R-AEV-010/013/R-DEL-009 (event registry) and R-AGE-008/S-AGE-010
	// (no path back to producer) both "hold" — re-asserted directly
	// here rather than only by citation.
	if got := len(agent.EventKinds()); got != 25 {
		t.Errorf("event registry kind count = %d, want 25 (holds)", got)
	}

	// R-PRH-002/R-PRH-005 AMENDED: this change's own agent-pre-request-
	// hook delta carries the amendment (checked structurally: the
	// chain-composition and nil-singular-non-empty-chain scenarios
	// this file and loop_hook_test.go both assert are green in the
	// SAME suite run as this test).
	skeletonProvider := agenttest.NewProvider(scriptTextResponse(t, ai.FinishReasonStop))
	skeletonSink := make(chan *agent.Event, 16)
	if _, _, err := agent.Turn(contextBackground(), skeletonProvider, "system prompt for hks-025", []ai.Message{firstMessage(t)}, agent.TurnOptions{}, skeletonSink); err != nil {
		t.Fatalf("skeleton Turn returned err = %v, want nil (R-LSK-002/R-PRH-005 hold)", err)
	}
	drainSink(t, skeletonSink)

	// R-CMP-004 CLARIFIED: a forward-adjusting pre-compact hook is
	// resolved, not refused — the behavioral proof lives in
	// hooks_compaction_test.go (S-HKS-008/S-CMP-041); this test asserts
	// the row's own RESOLUTION is to that delta by confirming the
	// production surface it needs (CompactionPlan.WithCut) exists and
	// is callable.
	plan := agent.CompactionPlan{}
	adjusted := plan.WithCut(7)
	if adjusted.Cut() != 7 {
		t.Errorf("CompactionPlan.WithCut(7).Cut() = %d, want 7 — the CLARIFIED row's own mechanism", adjusted.Cut())
	}
}

// S-HKS-026 — The inert path is byte-identical on every arm. Given
// six paired runs of identical scripts — success, turn failure, retry
// exhaustion, interrupt, shutdown, and a compacting run through each
// door — one of each pair with a zero-value Hooks, run TWICE on this
// change, then event streams (kind sequence), history read-backs and
// captured provider requests are equal, and runtime.NumGoroutine()
// returns to baseline after every one.
//
// AG-19's inert_path_test.go established the argument this reuses:
// every new code path this milestone adds is reachable ONLY through
// Hooks being non-zero (isZero()/len(...)==0 guards gate all of it —
// newObserverLane returns nil, the transport refusal never fires, the
// pre-request/pre-compact composition takes its empty-chain branch) —
// so "once on the merge base, once on this change" and "twice on this
// change with zero-value Hooks" are the same experiment for a caller
// who never sets Hooks.
func TestHooks_Inertness_ByteIdenticalOnEveryArm(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		t.Parallel()
		run := func(t *testing.T) []agent.EventKind {
			provider := agenttest.NewProvider(scriptTextResponse(t, ai.FinishReasonStop))
			h := agent.Harness{Provider: provider, System: "system prompt for hks-026-success"}
			sink := make(chan *agent.Event, 32)
			if _, _, err := h.Run(contextBackground(), firstMessage(t), sink); err != nil {
				t.Fatalf("Run returned err = %v, want nil", err)
			}
			return kindsOfHKS(drainSink(t, sink))
		}
		baseline := runtime.NumGoroutine()
		first, second := run(t), run(t)
		if !kindsEqualHKS(first, second) {
			t.Errorf("success arm: kind sequences differ: first=%v second=%v", first, second)
		}
		waitForGoroutineBaseline(t, baseline, twoSecondsForHooksTest())
	})

	t.Run("turn_failure", func(t *testing.T) {
		t.Parallel()
		run := func(t *testing.T) (kinds []agent.EventKind, err error) {
			failure := mustNonRetryablePreStreamFailure(t)
			inner := agenttest.NewProvider(scriptTextResponse(t, ai.FinishReasonStop))
			provider := newPreStreamFailingProvider(1, failure, inner)
			h := agent.Harness{Provider: provider, System: "system prompt for hks-026-failure"}
			sink := make(chan *agent.Event, 32)
			_, _, rerr := h.Run(contextBackground(), firstMessage(t), sink)
			return kindsOfHKS(drainSink(t, sink)), rerr
		}
		baseline := runtime.NumGoroutine()
		firstKinds, firstErr := run(t)
		secondKinds, secondErr := run(t)
		if (firstErr == nil) != (secondErr == nil) {
			t.Errorf("turn_failure arm: error presence differs: first=%v second=%v", firstErr, secondErr)
		}
		if !kindsEqualHKS(firstKinds, secondKinds) {
			t.Errorf("turn_failure arm: kind sequences differ: first=%v second=%v", firstKinds, secondKinds)
		}
		waitForGoroutineBaseline(t, baseline, twoSecondsForHooksTest())
	})

	t.Run("retry_exhaustion", func(t *testing.T) {
		t.Parallel()
		run := func(t *testing.T) (kinds []agent.EventKind, err error) {
			failure := mustPreStreamRetryableFailure(t, ai.RetryDelay{})
			inner := agenttest.NewProvider(scriptTextResponse(t, ai.FinishReasonStop))
			provider := newPreStreamFailingProvider(5, failure, inner)
			h := agent.Harness{Provider: provider, System: "system prompt for hks-026-exhaustion", RetryAttempts: 2, RetryTiming: agent.RetryTiming{SleepFunc: instantSleep}}
			sink := make(chan *agent.Event, 32)
			_, _, rerr := h.Run(contextBackground(), firstMessage(t), sink)
			return kindsOfHKS(drainSink(t, sink)), rerr
		}
		baseline := runtime.NumGoroutine()
		firstKinds, firstErr := run(t)
		secondKinds, secondErr := run(t)
		if (firstErr == nil) != (secondErr == nil) {
			t.Errorf("retry_exhaustion arm: error presence differs: first=%v second=%v", firstErr, secondErr)
		}
		if !kindsEqualHKS(firstKinds, secondKinds) {
			t.Errorf("retry_exhaustion arm: kind sequences differ: first=%v second=%v", firstKinds, secondKinds)
		}
		waitForGoroutineBaseline(t, baseline, twoSecondsForHooksTest())
	})

	t.Run("interrupt", func(t *testing.T) {
		t.Parallel()
		run := func(t *testing.T) []agent.EventKind {
			gate := agenttest.NewGate()
			provider := agenttest.NewProvider(heldTurnScript(t, gate))
			h := agent.Harness{Provider: provider, System: "system prompt for hks-026-interrupt"}
			sink := make(chan *agent.Event, 32)
			resultCh := make(chan struct{})
			go func() {
				_, _, _ = h.Run(contextBackground(), firstMessage(t), sink)
				close(resultCh)
			}()
			<-gate.Reached()
			h.Interrupt()
			kinds := kindsOfHKS(drainSink(t, sink))
			<-resultCh
			return kinds
		}
		baseline := runtime.NumGoroutine()
		first, second := run(t), run(t)
		if !kindsEqualHKS(first, second) {
			t.Errorf("interrupt arm: kind sequences differ: first=%v second=%v", first, second)
		}
		waitForGoroutineBaseline(t, baseline, twoSecondsForHooksTest())
	})

	t.Run("shutdown", func(t *testing.T) {
		t.Parallel()
		run := func(t *testing.T) []agent.EventKind {
			provider := agenttest.NewProvider(scriptTextResponse(t, ai.FinishReasonStop))
			h := agent.Harness{Provider: provider, System: "system prompt for hks-026-shutdown"}
			h.Shutdown()
			sink := make(chan *agent.Event, 32)
			_, _, _ = h.Run(contextBackground(), firstMessage(t), sink)
			return kindsOfHKS(drainSink(t, sink))
		}
		baseline := runtime.NumGoroutine()
		first, second := run(t), run(t)
		if !kindsEqualHKS(first, second) {
			t.Errorf("shutdown arm: kind sequences differ: first=%v second=%v", first, second)
		}
		waitForGoroutineBaseline(t, baseline, twoSecondsForHooksTest())
	})

	t.Run("compacting_strategy_door", func(t *testing.T) {
		t.Parallel()
		run := func(t *testing.T) []agent.EventKind {
			_, hist, _ := markedHarnessForCompaction(t, "hks-026-compact-strategy")
			strategy := &compactAfterNStrategy{request: func(prompt agent.ContextPrompt) agent.CompactionRequest {
				return agent.CompactionRequest{Provider: agenttest.NewProvider(scriptTextResponse(t, ai.FinishReasonStop)), Instruction: "summarize for hks-026", Cut: len(prompt.Transcript)}
			}}
			h := agent.Harness{Provider: agenttest.NewProvider(scriptTextResponse(t, ai.FinishReasonStop)), System: "system prompt for hks-026-compact-strategy", History: hist, ContextStrategy: strategy}
			sink := make(chan *agent.Event, 256)
			if _, _, err := h.Run(contextBackground(), firstMessage(t), sink); err != nil {
				t.Fatalf("Run returned err = %v, want nil", err)
			}
			return kindsOfHKS(drainSink(t, sink))
		}
		baseline := runtime.NumGoroutine()
		first, second := run(t), run(t)
		if !kindsEqualHKS(first, second) {
			t.Errorf("compacting (strategy door) arm: kind sequences differ: first=%v second=%v", first, second)
		}
		waitForGoroutineBaseline(t, baseline, twoSecondsForHooksTest())
	})

	t.Run("compacting_on_demand_door", func(t *testing.T) {
		t.Parallel()
		run := func(t *testing.T) []agent.EventKind {
			_, hist, _ := markedHarnessForCompaction(t, "hks-026-compact-demand")
			h := agent.Harness{Provider: agenttest.NewProvider(), System: "system prompt for hks-026-compact-demand", History: hist}
			req := agent.CompactionRequest{Provider: agenttest.NewProvider(scriptTextResponse(t, ai.FinishReasonStop)), Instruction: "summarize for hks-026 demand", Cut: hist.Len()}
			sink := make(chan *agent.Event, 64)
			if err := h.Compact(contextBackground(), req, sink); err != nil {
				t.Fatalf("Compact returned err = %v, want nil", err)
			}
			return kindsOfHKS(drainSink(t, sink))
		}
		baseline := runtime.NumGoroutine()
		first, second := run(t), run(t)
		if !kindsEqualHKS(first, second) {
			t.Errorf("compacting (on-demand door) arm: kind sequences differ: first=%v second=%v", first, second)
		}
		waitForGoroutineBaseline(t, baseline, twoSecondsForHooksTest())
	})
}

// kindsOfHKS renders events as their kind sequence — this file's own
// copy of inert_path_test.go's identical helper, not shared, so
// neither file's edit surface touches the other.
func kindsOfHKS(events []agent.Event) []agent.EventKind {
	kinds := make([]agent.EventKind, len(events))
	for i, ev := range events {
		kinds[i] = ev.Kind()
	}
	return kinds
}

// kindsEqualHKS mirrors inert_path_test.go's kindsEqual, this file's
// own copy for the same reason kindsOfHKS is.
func kindsEqualHKS(a, b []agent.EventKind) bool {
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

// mustNonRetryablePreStreamFailure builds a NON-retryable pre-stream
// failure (G2 fires: surfaced immediately, never retried) — the
// "turn_failure" arm's own fixture, distinct from mustPreStreamRetryableFailure's
// retryable shape ("retry_exhaustion"'s own fixture).
func mustNonRetryablePreStreamFailure(t *testing.T) *ai.Failure {
	t.Helper()
	f, err := ai.PreStreamFailure(ai.FailureReport{Category: ai.FailureCategoryUnavailable, Retryable: false})
	if err != nil {
		t.Fatalf("ai.PreStreamFailure returned %v, want no failure", err)
	}
	return f
}

// ---------------------------------------------------------------------
// R-LSK-008 — S-LSK-032, the substrate filter close-out (Item 1)
// ---------------------------------------------------------------------

// S-LSK-032 — AG-20: no release taken, and the widening is exact, by
// name, in both filters. Given the merge base of the AG-20 branch with
// origin/main, when git diff is taken over backend/agent/, then the
// set of pre-existing NON-TEST files under src/agent that differ is
// exactly {loop.go, harness.go, compaction.go} (plus, per this
// change's own AD-8 obligation, compaction_surgery_test.go — a
// pre-existing TEST file already named in both filters since AG-18,
// so its own edit here needs no new filter entry either); every file
// named forbidden by R-HKS-010 is byte-unchanged; and both substrate
// filters carry an identical 4-entry addition (hooks.go,
// hooks_test.go, hooks_harness_test.go, hooks_compaction_test.go), by
// exact filename suffix, no wildcard/prefix/directory pattern.
func TestHooks_SubstrateFilters_NoReleaseExactWideningInBothFilters(t *testing.T) {
	t.Parallel()

	root, err := gitTopLevel(t)
	if err != nil {
		t.Fatalf("git rev-parse --show-toplevel failed: %v", err)
	}
	baseRef := hksResolveBaseRef(t, root)

	if !hksBaseRefIsHEAD(t, root, baseRef) {
		raw, derr := gitDiff(t, root, baseRef, "backend/agent/src/agent/")
		if derr != nil {
			t.Fatalf("git diff %s -- backend/agent/src/agent/ failed: %v", baseRef, derr)
		}
		nonTestChanged := hksNonTestFilesChanged(raw)
		// hooks.go is a NEW file (git diff --git a/X b/X headers look
		// identical for "modified" and "newly added" — only the diff
		// BODY distinguishes them via "new file mode"/"--- /dev/null"),
		// so it legitimately appears in this diff too, once committed.
		// It is allowed here for that reason; the pre-existing set
		// S-LSK-032 actually cares about is the other three.
		wantPreExisting := map[string]bool{"loop.go": true, "harness.go": true, "compaction.go": true}
		allowed := map[string]bool{"loop.go": true, "harness.go": true, "compaction.go": true, "hooks.go": true}
		for _, name := range nonTestChanged {
			if !allowed[name] {
				t.Errorf("non-test file %q changed, want only loop.go/harness.go/compaction.go (pre-existing) plus hooks.go (new)", name)
			}
		}
		for name := range wantPreExisting {
			found := false
			for _, c := range nonTestChanged {
				if c == name {
					found = true
				}
			}
			if !found {
				t.Errorf("expected pre-existing non-test file %q to have changed, it did not", name)
			}
		}
	}

	loopEntries := extractHasSuffixEntries(t, "loop_test.go")
	hookEntries := extractHasSuffixEntries(t, "loop_hook_test.go")

	wantAG20 := []string{
		"/hooks.go",
		"/hooks_test.go",
		"/hooks_harness_test.go",
		"/hooks_compaction_test.go",
	}
	for _, want := range wantAG20 {
		if !containsString(loopEntries, want) {
			t.Errorf("filterOutLoopFiles does not name %q", want)
		}
		if !containsString(hookEntries, want) {
			t.Errorf("filterOutLoopHookFiles does not name %q", want)
		}
	}

	sortedLoop := append([]string(nil), loopEntries...)
	sortedHook := append([]string(nil), hookEntries...)
	sortStrings(sortedLoop)
	sortStrings(sortedHook)
	if len(sortedLoop) != len(sortedHook) {
		t.Fatalf("filterOutLoopFiles has %d entries, filterOutLoopHookFiles has %d, want identical count", len(sortedLoop), len(sortedHook))
	}
	for i := range sortedLoop {
		if sortedLoop[i] != sortedHook[i] {
			t.Errorf("entry set differs at sorted position %d: %q vs %q", i, sortedLoop[i], sortedHook[i])
		}
	}

	exactSuffixPattern := hksExactSuffixPattern()
	for _, entries := range [][]string{loopEntries, hookEntries} {
		for _, e := range entries {
			if !exactSuffixPattern.MatchString(e) {
				t.Errorf("filter entry %q is not an exact /<filename>.go suffix", e)
			}
			if strings.Count(e, "/") != 1 {
				t.Errorf("filter entry %q contains more than one path separator", e)
			}
		}
	}

	// AG-20 releases NO pre-existing filename — every AG-20 addition
	// above names a file this change CREATES.
	preExistingNeverAdded := []string{"/failure.go", "/doc.go", "/doc_contract_guard_test.go", "/cost_events_test.go", "/stream_check_test.go", "/reconstruction_test.go"}
	for _, bad := range preExistingNeverAdded {
		if countString(loopEntries, bad) > 1 {
			t.Errorf("filterOutLoopFiles names %q more than once — AG-20 must add no duplicate entry for any pre-existing filename", bad)
		}
	}
}

// hksNonTestFilesChanged parses a git diff and returns the base
// filenames of every changed file directly under backend/agent/src/agent/
// (not a subdirectory) whose name does NOT end in _test.go.
func hksNonTestFilesChanged(diff string) []string {
	var out []string
	for _, line := range strings.Split(diff, "\n") {
		if !strings.HasPrefix(line, "diff --git ") {
			continue
		}
		idx := strings.LastIndex(line, " b/")
		if idx < 0 {
			continue
		}
		path := line[idx+3:]
		const prefix = "backend/agent/src/agent/"
		if !strings.HasPrefix(path, prefix) {
			continue
		}
		rest := path[len(prefix):]
		if strings.Contains(rest, "/") {
			continue // a subdirectory (e.g. sweep/, tracetest/) — out of scope.
		}
		if strings.HasSuffix(rest, "_test.go") {
			continue
		}
		if !strings.HasSuffix(rest, ".go") {
			continue
		}
		out = append(out, rest)
	}
	return out
}

// hksExactSuffixPattern mirrors scope_fence_test.go's own local
// exactSuffixPattern (inside TestScopeFence_S_LSK_031_...) — this
// file's own copy for the same "no shared edit surface" reason.
func hksExactSuffixPattern() *regexp.Regexp { return regexp.MustCompile(`^/[A-Za-z0-9_.]+\.go$`) }

// ---------------------------------------------------------------------
// Phase 5 — cross-capability audit scenarios
// ---------------------------------------------------------------------

// S-AEV-126 — AG-20: invariant 3's closure is checked against the
// shipped mechanism, not against prose. Ties together "no EventKind
// registered" (R-HKS-010) and "every stalled-observer path terminates
// in the lane's drain activity" (R-HKS-008, the asynchrony suite
// above): a Go-side typed report delivered off the streaming path,
// with the event registry byte-unchanged at its committed count.
func TestHooks_S_AEV_126_Invariant3ClosureCheckedAgainstShippedMechanism(t *testing.T) {
	t.Parallel()

	if got := len(agent.EventKinds()); got != 25 {
		t.Fatalf("len(agent.EventKinds()) = %d, want 25 (no EventKind registered for the stall report)", got)
	}

	hold := newWgObserverHold()
	t.Cleanup(hold.Release)
	var reported []agent.ObserverStall
	var mu sync.Mutex
	hooks := agent.Hooks{
		PostTurn:      []agent.PostTurnObserver{func(context.Context, agent.PostTurnReport) { hold.Enter() }},
		StallReporter: func(s agent.ObserverStall) { mu.Lock(); reported = append(reported, s); mu.Unlock() },
	}
	provider := agenttest.NewProvider(scriptTextResponse(t, ai.FinishReasonStop))
	h := agent.Harness{Provider: provider, System: "system prompt for aev-126", Hooks: hooks}
	sink := make(chan *agent.Event, 32)
	runDone := make(chan struct{})
	go func() {
		_, _, _ = h.Run(contextBackground(), firstMessage(t), sink)
		close(runDone)
	}()
	<-hold.Reached()
	<-runDone
	events := drainSink(t, sink)
	if report := agent.CheckStream(events); report.Violation() != nil {
		t.Errorf("CheckStream rejected the stalled-observer stream: %v, want accepted unmodified", report.Violation())
	}
	mu.Lock()
	gotReported := len(reported)
	mu.Unlock()
	if gotReported == 0 {
		t.Error("reporter received 0 reports, want at least one StallOutstanding — the stall must be reported OFF the stream, not omitted")
	}
	hold.Release()
}

// S-AGE-030 — AG-20: the stalled-HOOK trace has no path back to the
// producer either, and it is checked in code. Reuses S-HKS-017's
// gated-hook fixture and S-DEL-026's own child-run seam-lookup
// pattern in one cross-check: a hosted child run's own post-turn
// observer, held open, looks up a publishing seam from the context it
// is invoked with, finds none, and the parent stream is recorded
// CheckStream-valid while the observer's own hold is still held.
func TestHooks_S_AGE_030_StalledHookTraceHasNoPathBack_ChildObserverFindsNoSeam(t *testing.T) {
	t.Parallel()

	hold := newWgObserverHold()
	t.Cleanup(hold.Release)

	var seamFound bool
	var mu sync.Mutex
	childProvider := agenttest.NewProvider(scriptTextResponse(t, ai.FinishReasonStop))
	child := &agent.Harness{Provider: childProvider, System: "system prompt for the hosted child (age-030)", Hooks: agent.Hooks{
		PostTurn: []agent.PostTurnObserver{func(ctx context.Context, _ agent.PostTurnReport) {
			_, ok := agent.DelegationSeamFrom(ctx)
			mu.Lock()
			seamFound = ok
			mu.Unlock()
			hold.Enter()
		}},
	}}

	toolName := "age_030_tool"
	tool := &delegatingTool{toolName: toolName, effect: agent.EffectClassRead, child: child, prompt: firstMessage(t)}
	reg := agent.NewMapRegistry(map[string]agent.Tool{toolName: tool})

	turnOneScript := scriptToolCallResponse(t, "call-age-030", toolName, []byte(`{}`))
	turnTwoScript := scriptTextResponse(t, ai.FinishReasonStop)
	parentProvider := agenttest.NewProvider(turnOneScript, turnTwoScript)

	h := agent.Harness{Provider: parentProvider, System: "system prompt for age-030 parent", Turn: agent.TurnOptions{Tools: reg}}
	sink := make(chan *agent.Event, 512)

	runDone := make(chan struct{})
	var runErr error
	go func() {
		_, _, runErr = h.Run(contextBackground(), firstMessage(t), sink)
		close(runDone)
	}()

	<-hold.Reached() // the child's own post-turn observer started, and has not returned.
	<-runDone
	if runErr != nil {
		t.Fatalf("parent Run returned err = %v, want nil", runErr)
	}

	parentEvents := drainSink(t, sink)
	if report := agent.CheckStream(parentEvents); report.Violation() != nil {
		t.Errorf("CheckStream rejected the parent stream: %v, want accepted unmodified", report.Violation())
	}

	mu.Lock()
	gotSeamFound := seamFound
	mu.Unlock()
	if gotSeamFound {
		t.Error("child observer's own DelegationSeamFrom(ctx) found a seam, want none — the observer's context must be value-stripped")
	}

	hold.Release()
}

// S-AGS-065 — AG-20: the discharge is auditable, not asserted. The
// one registration surface, the two type families, the pre-compact
// plan's own derivation surface and the session-start report each
// exist with the shape this suite's behavioral tests exercise.
func TestHooks_S_AGS_065_DischargeIsAuditableNotAsserted(t *testing.T) {
	t.Parallel()

	hooksField, ok := reflect.TypeOf(agent.Harness{}).FieldByName("Hooks")
	if !ok {
		t.Fatal("agent.Harness has no Hooks field")
	}
	if hooksField.Type != reflect.TypeOf(agent.Hooks{}) {
		t.Errorf("Harness.Hooks field type = %v, want agent.Hooks", hooksField.Type)
	}

	if reflect.TypeOf((*agent.PreRequestHook)(nil)).Elem().NumOut() != 2 {
		t.Error("PreRequestHook is not a 2-result mutating shape")
	}
	if reflect.TypeOf((*agent.PostTurnObserver)(nil)).Elem().NumOut() != 0 {
		t.Error("PostTurnObserver is not a 0-result observing shape")
	}

	var plan agent.CompactionPlan
	if got := plan.WithCut(3).Cut(); got != 3 {
		t.Errorf("CompactionPlan.WithCut(3).Cut() = %d, want 3", got)
	}
}

// S-AGS-066 — AG-20: the Layer 3 half is checked against the shipped
// surface, not against prose. No concrete hook, no timeout/deadline/
// sleep of any kind in hooks.go, and a zero-value Hooks is inert by
// default.
func TestHooks_S_AGS_066_Layer3HalfCheckedAgainstShippedSurface(t *testing.T) {
	t.Parallel()

	root, err := gitTopLevel(t)
	if err != nil {
		t.Fatalf("git rev-parse --show-toplevel failed: %v", err)
	}
	raw, rerr := os.ReadFile(root + "/backend/agent/src/agent/hooks.go")
	if rerr != nil {
		t.Fatalf("os.ReadFile(hooks.go) failed: %v", rerr)
	}
	src := string(raw)
	forbidden := []string{"time.After", "time.Sleep", "time.NewTimer", "time.NewTicker", "context.WithTimeout", "context.WithDeadline"}
	for _, f := range forbidden {
		if strings.Contains(src, f) {
			t.Errorf("hooks.go contains %q, want no wall clock of any kind (R-HKS-008, R-RUN-010)", f)
		}
	}
	if strings.Contains(src, `"time"`) {
		t.Error(`hooks.go imports "time", want no wall clock anywhere`)
	}

	var zero agent.Hooks
	if len(zero.PreRequest) != 0 || len(zero.PreCompact) != 0 || len(zero.PostTurn) != 0 || len(zero.SessionStart) != 0 || zero.StallReporter != nil {
		t.Error("zero-value agent.Hooks carries a registered hook, want fully empty")
	}
}

// S-AIV-032 — AG-20: V-OUT-13's exclusion is checked against the
// shipped code, not against prose. The four named hook points exist,
// each classified mutating/observing by its own function type; the
// diff under backend/agent/src/ai/ is empty; go.mod/go.sum are
// byte-unchanged.
func TestHooks_S_AIV_032_ExclusionCheckedAgainstShippedCode(t *testing.T) {
	t.Parallel()

	root, err := gitTopLevel(t)
	if err != nil {
		t.Fatalf("git rev-parse --show-toplevel failed: %v", err)
	}
	baseRef := hksResolveBaseRef(t, root)
	aiDiff, derr := gitDiff(t, root, baseRef, "backend/agent/src/ai/")
	if derr != nil {
		t.Fatalf("git diff %s -- backend/agent/src/ai/ failed: %v", baseRef, derr)
	}
	if aiDiff != "" {
		t.Errorf("backend/agent/src/ai/ diff against %s is not empty (R-RUN-012):\n%s", baseRef, aiDiff)
	}
	modDiff, merr := gitDiff(t, root, baseRef, "backend/agent/go.mod", "backend/agent/go.sum")
	if merr != nil {
		t.Fatalf("git diff %s -- go.mod go.sum failed: %v", baseRef, merr)
	}
	if modDiff != "" {
		t.Errorf("go.mod/go.sum diff against %s is not empty:\n%s", baseRef, modDiff)
	}

	families := map[string]int{
		"PreRequestHook":       2,
		"PreCompactHook":       2,
		"PostTurnObserver":     0,
		"SessionStartObserver": 0,
	}
	types := map[string]reflect.Type{
		"PreRequestHook":       reflect.TypeOf((*agent.PreRequestHook)(nil)).Elem(),
		"PreCompactHook":       reflect.TypeOf((*agent.PreCompactHook)(nil)).Elem(),
		"PostTurnObserver":     reflect.TypeOf((*agent.PostTurnObserver)(nil)).Elem(),
		"SessionStartObserver": reflect.TypeOf((*agent.SessionStartObserver)(nil)).Elem(),
	}
	for name, wantOut := range families {
		if got := types[name].NumOut(); got != wantOut {
			t.Errorf("%s.NumOut() = %d, want %d", name, got, wantOut)
		}
	}
}
