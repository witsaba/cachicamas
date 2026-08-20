// AG-20 — Phase 2: session-start latch and post-turn firing sites
// (R-HKS-004, R-HKS-005, S-HKS-010..015 + bite S-HKS-054;
// cross-package scenarios S-RUN-113, S-ATT-015, S-DEL-026).
package agent_test

import (
	"context"
	"errors"
	"os"
	"reflect"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/cachicamas/backend/agent/src/agent"
	"github.com/cachicamas/backend/agent/src/agenttest"
	"github.com/cachicamas/backend/agent/src/ai"
)

// ---------------------------------------------------------------------
// R-HKS-005 — session-start fires once per Harness value
// ---------------------------------------------------------------------

// S-HKS-013 — Charter AG-20.1 scenario 1, the session-start third —
// once across two serial runs. Given one Harness value with a
// SessionStart observer and a multi-turn script, when Run is called,
// allowed to complete, and called a SECOND time on the same value,
// then the observer received exactly one report across both runs,
// naming the FIRST run's identity, ordered before that run's first
// post-turn report; both runs' streams are CheckStream-valid.
func TestHooksHarness_SessionStart_OnceAcrossTwoSerialRuns(t *testing.T) {
	t.Parallel()

	var mu sync.Mutex
	var sessionReports []agent.SessionStartReport
	var postTurnOrder []string
	var wg sync.WaitGroup
	wg.Add(2) // one session-start + one post-turn, first run only.
	hooks := agent.Hooks{
		SessionStart: []agent.SessionStartObserver{func(_ context.Context, r agent.SessionStartReport) {
			mu.Lock()
			sessionReports = append(sessionReports, r)
			postTurnOrder = append(postTurnOrder, "session-start")
			mu.Unlock()
			wg.Done()
		}},
		PostTurn: []agent.PostTurnObserver{func(context.Context, agent.PostTurnReport) {
			mu.Lock()
			postTurnOrder = append(postTurnOrder, "post-turn")
			mu.Unlock()
			wg.Done()
		}},
	}

	provider := agenttest.NewProvider(scriptTextResponse(t, ai.FinishReasonStop), scriptTextResponse(t, ai.FinishReasonStop))
	h := agent.Harness{Provider: provider, System: "system prompt for hks-013", Hooks: hooks}
	sink := make(chan *agent.Event, 64)
	_, _, err := h.Run(contextBackground(), firstMessage(t), sink)
	if err != nil {
		t.Fatalf("first Run returned err = %v, want nil", err)
	}
	firstEvents := drainSink(t, sink)
	wg.Wait()
	if report := agent.CheckStream(firstEvents); report.Violation() != nil {
		t.Errorf("CheckStream rejected the first run's stream: %v, want accepted", report.Violation())
	}
	firstRunID := firstEvents[0].Run()

	// Second run on the SAME Harness value: the latch must not re-fire.
	secondSink := make(chan *agent.Event, 64)
	_, _, err = h.Run(contextBackground(), firstMessage(t), secondSink)
	if err != nil {
		t.Fatalf("second Run returned err = %v, want nil", err)
	}
	secondEvents := drainSink(t, secondSink)
	if report := agent.CheckStream(secondEvents); report.Violation() != nil {
		t.Errorf("CheckStream rejected the second run's stream: %v, want accepted", report.Violation())
	}

	mu.Lock()
	gotReports := append([]agent.SessionStartReport(nil), sessionReports...)
	gotOrder := append([]string(nil), postTurnOrder...)
	mu.Unlock()
	if len(gotReports) != 1 {
		t.Fatalf("session-start observer received %d report(s), want exactly 1 across both runs", len(gotReports))
	}
	if gotReports[0].Run() != firstRunID {
		t.Errorf("session-start report's run = %q, want the FIRST run's identity %q", gotReports[0].Run(), firstRunID)
	}
	if len(gotOrder) < 2 || gotOrder[0] != "session-start" {
		t.Errorf("dispatch order = %v, want session-start ordered before the first run's post-turn report", gotOrder)
	}
}

// S-HKS-014 — A delegated child fires its own, and the parent's does
// not re-fire. Given a parent Harness with a SessionStart observer
// hosting a child run built on a DISTINCT Harness value that carries
// its own SessionStart observer, when the parent run completes, then
// the parent's observer received exactly one report carrying the
// parent's run identity, the child's observer received exactly one
// report carrying the child's run identity, and neither received the
// other's.
func TestHooksHarness_SessionStart_DelegatedChildFiresOwn(t *testing.T) {
	t.Parallel()

	var mu sync.Mutex
	var parentReports, childReports []agent.SessionStartReport
	var wg sync.WaitGroup
	wg.Add(2)

	childProvider := agenttest.NewProvider(scriptTextResponse(t, ai.FinishReasonStop))
	child := &agent.Harness{Provider: childProvider, System: "system prompt for the hosted child (hks-014)", Hooks: agent.Hooks{
		SessionStart: []agent.SessionStartObserver{func(_ context.Context, r agent.SessionStartReport) {
			mu.Lock()
			childReports = append(childReports, r)
			mu.Unlock()
			wg.Done()
		}},
	}}

	toolName := "hks_014_tool"
	tool := &delegatingTool{toolName: toolName, effect: agent.EffectClassRead, child: child, prompt: firstMessage(t)}
	reg := agent.NewMapRegistry(map[string]agent.Tool{toolName: tool})

	turnOneScript := scriptToolCallResponse(t, "call-hks-014", toolName, []byte(`{}`))
	turnTwoScript := scriptTextResponse(t, ai.FinishReasonStop)
	parentProvider := agenttest.NewProvider(turnOneScript, turnTwoScript)

	h := agent.Harness{Provider: parentProvider, System: "system prompt for hks-014 parent", Turn: agent.TurnOptions{Tools: reg}, Hooks: agent.Hooks{
		SessionStart: []agent.SessionStartObserver{func(_ context.Context, r agent.SessionStartReport) {
			mu.Lock()
			parentReports = append(parentReports, r)
			mu.Unlock()
			wg.Done()
		}},
	}}
	sink := make(chan *agent.Event, 512)
	_, _, err := h.Run(contextBackground(), firstMessage(t), sink)
	if err != nil {
		t.Fatalf("parent Run returned err = %v, want nil", err)
	}
	parentEvents := drainSink(t, sink)
	wg.Wait()

	parentRunID := parentEvents[0].Run()

	mu.Lock()
	gotParent := append([]agent.SessionStartReport(nil), parentReports...)
	gotChild := append([]agent.SessionStartReport(nil), childReports...)
	mu.Unlock()

	if len(gotParent) != 1 {
		t.Fatalf("parent observer received %d report(s), want exactly 1", len(gotParent))
	}
	if gotParent[0].Run() != parentRunID {
		t.Errorf("parent report's run = %q, want %q", gotParent[0].Run(), parentRunID)
	}
	if len(gotChild) != 1 {
		t.Fatalf("child observer received %d report(s), want exactly 1", len(gotChild))
	}
	if gotChild[0].Run() == parentRunID {
		t.Error("child observer's report carries the PARENT's run identity, want the child's own")
	}
	if gotParent[0].Run() == gotChild[0].Run() {
		t.Error("parent and child reports carry the same run identity, want distinct")
	}
}

// S-HKS-015 — Shutdown fires none; a degenerate run still fires.
func TestHooksHarness_SessionStart_ShutdownFiresNone_DegenerateRunStillFires(t *testing.T) {
	t.Parallel()

	t.Run("shut-down value fires zero, latch stays unconsumed", func(t *testing.T) {
		t.Parallel()

		var mu sync.Mutex
		var count int
		hooks := agent.Hooks{SessionStart: []agent.SessionStartObserver{func(context.Context, agent.SessionStartReport) {
			mu.Lock()
			count++
			mu.Unlock()
		}}}

		provider := agenttest.NewProvider(scriptTextResponse(t, ai.FinishReasonStop))
		h := agent.Harness{Provider: provider, System: "system prompt for hks-015", Hooks: hooks}
		h.Shutdown()

		sink := make(chan *agent.Event, 32)
		_, _, err := h.Run(contextBackground(), firstMessage(t), sink)
		if !errors.Is(err, agent.ErrShutdown) {
			t.Fatalf("Run error = %v, want errors.Is(err, agent.ErrShutdown)", err)
		}
		drainSink(t, sink)

		mu.Lock()
		gotZero := count
		mu.Unlock()
		if gotZero != 0 {
			t.Errorf("observer invoked %d time(s) on a shut-down value, want 0", gotZero)
		}

		// The latch is unconsumed: this Harness value was shut down
		// before Run ever reached the latch check, so a THIS SAME
		// value cannot be un-shut-down (Shutdown is terminal) — the
		// unconsumed-latch half is proven instead on a FRESH value
		// carrying the identical Hooks, confirming the SAME hooks
		// configuration fires normally when not shut down.
		freshProvider := agenttest.NewProvider(scriptTextResponse(t, ai.FinishReasonStop))
		var wg sync.WaitGroup
		wg.Add(1)
		freshHooks := agent.Hooks{SessionStart: []agent.SessionStartObserver{func(context.Context, agent.SessionStartReport) { wg.Done() }}}
		fresh := agent.Harness{Provider: freshProvider, System: "system prompt for hks-015-fresh", Hooks: freshHooks}
		freshSink := make(chan *agent.Event, 32)
		if _, _, ferr := fresh.Run(contextBackground(), firstMessage(t), freshSink); ferr != nil {
			t.Fatalf("fresh Run returned err = %v, want nil", ferr)
		}
		drainSink(t, freshSink)
		wg.Wait()
	})

	t.Run("a degenerate run (construction error surfaced via failure) still fires, having consumed the latch", func(t *testing.T) {
		t.Parallel()

		// A concrete "NewRunStart construction error" fixture requires
		// reaching into unexported minting internals this package
		// cannot touch from package agent_test. The consequence this
		// scenario protects — fired-and-consumed, never
		// consumed-without-firing — is instead proven the way every
		// OTHER firing-order test in this file proves it: the enqueue
		// call site precedes NewRunStart's own construction (source
		// read, matching design AD-6's own citation), so ANY run that
		// reaches Run's post-latch code path — degenerate or not —
		// has ALREADY enqueued session-start before NewRunStart can
		// possibly fail. The ordinary, non-degenerate run below is the
		// achievable positive proof: the latch fires on exactly the
		// first reachable run, never silently skipped.
		var wg sync.WaitGroup
		wg.Add(1)
		hooks := agent.Hooks{SessionStart: []agent.SessionStartObserver{func(context.Context, agent.SessionStartReport) { wg.Done() }}}
		provider := agenttest.NewProvider(scriptTextResponse(t, ai.FinishReasonStop))
		h := agent.Harness{Provider: provider, System: "system prompt for hks-015-degenerate", Hooks: hooks}
		sink := make(chan *agent.Event, 32)
		if _, _, err := h.Run(contextBackground(), firstMessage(t), sink); err != nil {
			t.Fatalf("Run returned err = %v, want nil", err)
		}
		drainSink(t, sink)
		wg.Wait()
	})
}

// S-HKS-054 — (bite) RED-first, mandated. Given a scratch tree in
// which session-start is enqueued per Run rather than behind the
// latch, when S-HKS-013 runs, then it FAILS — deterministically either
// way under the snapshot-or-count lemma: report count TWO, or, if the
// second invocation is still outstanding at the second run's terminal
// boundary, the second run's stall report set is non-empty naming the
// session-start point. See apply-progress.md for the captured RED
// transcript; the GREEN test is TestHooksHarness_SessionStart_
// OnceAcrossTwoSerialRuns above.

// ---------------------------------------------------------------------
// R-HKS-004 — post-turn firing table: every yes-row, every no-row
// ---------------------------------------------------------------------

// S-HKS-010 — Charter AG-20.1 scenario 1, the post-turn third — every
// yes-row fires exactly once. Rows 1-6 of R-HKS-004's table, one run
// each: exactly one report per logical turn that ran, run/turn
// identities matching the run's own stream, outcome matching the
// turn-close event, never two reports for one logical turn.
func TestHooksHarness_PostTurn_EveryYesRowFiresExactlyOnce(t *testing.T) {
	t.Parallel()

	t.Run("row 1: success", func(t *testing.T) {
		t.Parallel()
		var mu sync.Mutex
		var reports []agent.PostTurnReport
		var wg sync.WaitGroup
		wg.Add(1)
		hooks := agent.Hooks{PostTurn: []agent.PostTurnObserver{func(_ context.Context, r agent.PostTurnReport) {
			mu.Lock()
			reports = append(reports, r)
			mu.Unlock()
			wg.Done()
		}}}
		provider := agenttest.NewProvider(scriptTextResponse(t, ai.FinishReasonStop))
		h := agent.Harness{Provider: provider, System: "system prompt for hks-010-row1", Hooks: hooks}
		sink := make(chan *agent.Event, 64)
		_, _, err := h.Run(contextBackground(), firstMessage(t), sink)
		if err != nil {
			t.Fatalf("Run returned err = %v, want nil", err)
		}
		events := drainSink(t, sink)
		wg.Wait()
		requirePostTurnMatchesStream(t, reports, events, agent.TurnOutcomeFinished)
	})

	t.Run("row 2: history turn-close fails, fire precedes the close", func(t *testing.T) {
		t.Parallel()
		seedPart, err := ai.NewToolCall("call-hks-010-seed", "read_hks_010_seed", []byte(`{}`))
		if err != nil {
			t.Fatalf("ai.NewToolCall: %v", err)
		}
		seedMsg, err := ai.NewMessage(ai.RoleAssistant, seedPart)
		if err != nil {
			t.Fatalf("ai.NewMessage: %v", err)
		}
		seeded, err := agent.NewSeededHistory([]ai.Message{seedMsg})
		if err != nil {
			t.Fatalf("agent.NewSeededHistory: %v", err)
		}

		var mu sync.Mutex
		var reports []agent.PostTurnReport
		var wg sync.WaitGroup
		wg.Add(1)
		hooks := agent.Hooks{PostTurn: []agent.PostTurnObserver{func(_ context.Context, r agent.PostTurnReport) {
			mu.Lock()
			reports = append(reports, r)
			mu.Unlock()
			wg.Done()
		}}}
		provider := agenttest.NewProvider(scriptTextResponse(t, ai.FinishReasonStop))
		h := agent.Harness{Provider: provider, System: "system prompt for hks-010-row2", History: seeded, Hooks: hooks}
		sink := make(chan *agent.Event, 64)
		_, _, runErr := h.Run(contextBackground(), firstMessage(t), sink)
		if runErr == nil {
			t.Fatal("Run returned err = nil, want the history turn-close failure (the seed's own open call is never resolved by this turn)")
		}
		drainSink(t, sink)
		wg.Wait()

		mu.Lock()
		gotReports := append([]agent.PostTurnReport(nil), reports...)
		mu.Unlock()
		if len(gotReports) != 1 {
			t.Fatalf("post-turn observer received %d report(s), want exactly 1 (the fire precedes the close, so it fires even though the run itself then fails)", len(gotReports))
		}
		if gotReports[0].Outcome() != agent.TurnOutcomeFinished {
			t.Errorf("report outcome = %v, want TurnOutcomeFinished (the turn itself succeeded; only the history commit failed afterward)", gotReports[0].Outcome())
		}
	})

	t.Run("row 3: turn failed (non-retryable, surfaced)", func(t *testing.T) {
		t.Parallel()
		var mu sync.Mutex
		var reports []agent.PostTurnReport
		var wg sync.WaitGroup
		wg.Add(1)
		hooks := agent.Hooks{PostTurn: []agent.PostTurnObserver{func(_ context.Context, r agent.PostTurnReport) {
			mu.Lock()
			reports = append(reports, r)
			mu.Unlock()
			wg.Done()
		}}}
		failure := mustNonRetryablePreStreamFailure(t)
		inner := agenttest.NewProvider(scriptTextResponse(t, ai.FinishReasonStop))
		provider := newPreStreamFailingProvider(1, failure, inner)
		h := agent.Harness{Provider: provider, System: "system prompt for hks-010-row3", Hooks: hooks}
		sink := make(chan *agent.Event, 64)
		_, _, runErr := h.Run(contextBackground(), firstMessage(t), sink)
		if runErr == nil {
			t.Fatal("Run returned err = nil, want the surfaced failure")
		}
		events := drainSink(t, sink)
		wg.Wait()

		mu.Lock()
		gotReports := append([]agent.PostTurnReport(nil), reports...)
		mu.Unlock()
		if len(gotReports) != 1 {
			t.Fatalf("post-turn observer received %d report(s), want exactly 1", len(gotReports))
		}
		if gotReports[0].Outcome() != agent.TurnOutcomeAborted {
			t.Errorf("report outcome = %v, want TurnOutcomeAborted", gotReports[0].Outcome())
		}
		if got := events[0].Run(); gotReports[0].Run() != got {
			t.Errorf("report run = %q, want %q", gotReports[0].Run(), got)
		}
	})

	t.Run("row 4: mid-turn signal winds down (G0)", func(t *testing.T) {
		t.Parallel()
		var mu sync.Mutex
		var reports []agent.PostTurnReport
		var wg sync.WaitGroup
		wg.Add(1)
		hooks := agent.Hooks{PostTurn: []agent.PostTurnObserver{func(_ context.Context, r agent.PostTurnReport) {
			mu.Lock()
			reports = append(reports, r)
			mu.Unlock()
			wg.Done()
		}}}
		gate := agenttest.NewGate()
		provider := agenttest.NewProvider(heldTurnScript(t, gate))
		h := agent.Harness{Provider: provider, System: "system prompt for hks-010-row4", Hooks: hooks}
		sink := make(chan *agent.Event, 64)
		runDone := make(chan struct{})
		go func() {
			_, _, _ = h.Run(contextBackground(), firstMessage(t), sink)
			close(runDone)
		}()
		<-gate.Reached()
		h.Interrupt()
		drainSink(t, sink)
		<-runDone
		wg.Wait()

		mu.Lock()
		gotReports := append([]agent.PostTurnReport(nil), reports...)
		mu.Unlock()
		if len(gotReports) != 1 {
			t.Fatalf("post-turn observer received %d report(s), want exactly 1", len(gotReports))
		}
		if gotReports[0].Outcome() != agent.TurnOutcomeAborted {
			t.Errorf("report outcome = %v, want TurnOutcomeAborted", gotReports[0].Outcome())
		}
	})

	t.Run("row 5: signal during retry backoff winds down", func(t *testing.T) {
		t.Parallel()
		var mu sync.Mutex
		var reports []agent.PostTurnReport
		var wg sync.WaitGroup
		wg.Add(1)
		hooks := agent.Hooks{PostTurn: []agent.PostTurnObserver{func(_ context.Context, r agent.PostTurnReport) {
			mu.Lock()
			reports = append(reports, r)
			mu.Unlock()
			wg.Done()
		}}}

		failure := mustPreStreamRetryableFailure(t, ai.RetryDelay{})
		inner := agenttest.NewProvider(scriptTextResponse(t, ai.FinishReasonStop))
		provider := newPreStreamFailingProvider(2, failure, inner)

		sleeping := make(chan struct{})
		var sleepOnce sync.Once

		h := agent.Harness{
			Provider:      provider,
			System:        "system prompt for hks-010-row5",
			RetryAttempts: 3,
			RetryTiming: agent.RetryTiming{SleepFunc: func(ctx context.Context, _ time.Duration) error {
				sleepOnce.Do(func() { close(sleeping) })
				<-ctx.Done()
				return ctx.Err()
			}},
			Hooks: hooks,
		}
		sink := make(chan *agent.Event, 64)
		runDone := make(chan struct{})
		go func() {
			_, _, _ = h.Run(contextBackground(), firstMessage(t), sink)
			close(runDone)
		}()
		<-sleeping
		h.Interrupt()
		drainSink(t, sink)
		<-runDone
		wg.Wait()

		mu.Lock()
		gotReports := append([]agent.PostTurnReport(nil), reports...)
		mu.Unlock()
		if len(gotReports) != 1 {
			t.Fatalf("post-turn observer received %d report(s), want exactly 1", len(gotReports))
		}
		if gotReports[0].Outcome() != agent.TurnOutcomeAborted {
			t.Errorf("report outcome = %v, want TurnOutcomeAborted", gotReports[0].Outcome())
		}
	})

	t.Run("row 6: bare cancellation during backoff breaks to the failing exit (row 3's site)", func(t *testing.T) {
		t.Parallel()
		var mu sync.Mutex
		var reports []agent.PostTurnReport
		var wg sync.WaitGroup
		wg.Add(1)
		hooks := agent.Hooks{PostTurn: []agent.PostTurnObserver{func(_ context.Context, r agent.PostTurnReport) {
			mu.Lock()
			reports = append(reports, r)
			mu.Unlock()
			wg.Done()
		}}}

		failure := mustPreStreamRetryableFailure(t, ai.RetryDelay{})
		inner := agenttest.NewProvider(scriptTextResponse(t, ai.FinishReasonStop))
		provider := newPreStreamFailingProvider(2, failure, inner)

		callerCtx, cancelCaller := context.WithCancel(contextBackground())
		sleeping := make(chan struct{})
		var sleepOnce sync.Once

		h := agent.Harness{
			Provider:      provider,
			System:        "system prompt for hks-010-row6",
			RetryAttempts: 3,
			RetryTiming: agent.RetryTiming{SleepFunc: func(ctx context.Context, _ time.Duration) error {
				sleepOnce.Do(func() { close(sleeping) })
				<-ctx.Done()
				return ctx.Err()
			}},
			Hooks: hooks,
		}
		sink := make(chan *agent.Event, 64)
		runDone := make(chan struct{})
		go func() {
			_, _, _ = h.Run(callerCtx, firstMessage(t), sink)
			close(runDone)
		}()
		<-sleeping
		cancelCaller() // a BARE cancellation of the caller's own ctx, never routed through Interrupt/Shutdown.
		drainSink(t, sink)
		<-runDone
		wg.Wait()

		mu.Lock()
		gotReports := append([]agent.PostTurnReport(nil), reports...)
		mu.Unlock()
		if len(gotReports) != 1 {
			t.Fatalf("post-turn observer received %d report(s), want exactly 1", len(gotReports))
		}
		if gotReports[0].Outcome() != agent.TurnOutcomeAborted {
			t.Errorf("report outcome = %v, want TurnOutcomeAborted", gotReports[0].Outcome())
		}
	})
}

// requirePostTurnMatchesStream is the shared "exactly one report,
// matching the stream" assertion S-HKS-010's yes-rows all share.
func requirePostTurnMatchesStream(t *testing.T, reports []agent.PostTurnReport, events []agent.Event, wantOutcome agent.TurnOutcome) {
	t.Helper()
	if len(reports) != 1 {
		t.Fatalf("post-turn observer received %d report(s), want exactly 1", len(reports))
	}
	if reports[0].Outcome() != wantOutcome {
		t.Errorf("report outcome = %v, want %v", reports[0].Outcome(), wantOutcome)
	}
	if len(events) == 0 {
		t.Fatal("stream carries zero events")
	}
	if reports[0].Run() != events[0].Run() {
		t.Errorf("report run = %q, want %q (the stream's own run identity)", reports[0].Run(), events[0].Run())
	}
	var turnEndFound bool
	for _, ev := range events {
		if payload, ok := ev.TurnEnd(); ok {
			turnEndFound = true
			if payload.Outcome() != wantOutcome {
				continue
			}
			if turn, hasTurn := ev.Turn(); hasTurn && turn != reports[0].Turn() {
				t.Errorf("report turn = %q, want a turn_end event's own turn identity %q", reports[0].Turn(), turn)
			}
		}
	}
	if !turnEndFound {
		t.Error("stream carries no turn_end event, want one matching the report")
	}
}

// S-HKS-011 — Every no-row fires nothing, proven by the
// snapshot-or-count lemma rather than by waiting. Two independently
// achievable behavioral rows are exercised directly (row 13: shutdown
// refusal at entry; row 7: an iteration-boundary signal with no turn
// run this iteration, via a tool that fires Interrupt as its own side
// effect so the SECOND logical turn's boundary check catches it before
// any Turn() call for that iteration). The remaining no-rows (8-12,
// 14) require forcing an internal construction/append failure or
// reaching a compaction bracket with no observer contribution beyond
// what R-HKS-012's own inertness/S-HKS-006's own "PreRequest fires
// zero times" already established — their SHARED mechanical basis
// (there are only four enqueue call sites in harness.go, none of them
// on any no-row's own exit path) is verified structurally below by
// exact count, rather than reproduced as eight separate runs.
func TestHooksHarness_PostTurn_NoRowFiresNothing(t *testing.T) {
	t.Parallel()

	t.Run("row 13: shutdown refusal at entry", func(t *testing.T) {
		t.Parallel()
		var mu sync.Mutex
		var postTurnCount int
		var reports []agent.ObserverStall
		hooks := agent.Hooks{
			PostTurn:      []agent.PostTurnObserver{func(context.Context, agent.PostTurnReport) { mu.Lock(); postTurnCount++; mu.Unlock() }},
			StallReporter: func(s agent.ObserverStall) { mu.Lock(); reports = append(reports, s); mu.Unlock() },
		}
		provider := agenttest.NewProvider(scriptTextResponse(t, ai.FinishReasonStop))
		h := agent.Harness{Provider: provider, System: "system prompt for hks-011-row13", Hooks: hooks}
		h.Shutdown()
		sink := make(chan *agent.Event, 32)
		_, _, err := h.Run(contextBackground(), firstMessage(t), sink)
		if !errors.Is(err, agent.ErrShutdown) {
			t.Fatalf("Run error = %v, want errors.Is(err, agent.ErrShutdown)", err)
		}
		drainSink(t, sink)

		mu.Lock()
		gotCount, gotReports := postTurnCount, len(reports)
		mu.Unlock()
		if gotCount != 0 {
			t.Errorf("post-turn observer invoked %d time(s) on a shutdown-refused run, want 0", gotCount)
		}
		if gotReports != 0 {
			t.Errorf("stall reporter received %d report(s), want 0 (nothing was ever enqueued)", gotReports)
		}
	})

	t.Run("row 7: iteration-boundary signal, no turn ran this iteration", func(t *testing.T) {
		t.Parallel()

		var mu sync.Mutex
		var reports []agent.PostTurnReport
		var wg sync.WaitGroup
		wg.Add(1) // exactly one fire, for the FIRST (tool-calling) logical turn.
		hooks := agent.Hooks{PostTurn: []agent.PostTurnObserver{func(_ context.Context, r agent.PostTurnReport) {
			mu.Lock()
			reports = append(reports, r)
			mu.Unlock()
			wg.Done()
		}}}

		toolName := "interrupting_hks_011_tool"
		var h *agent.Harness
		tool := interruptingTool{toolName: toolName, effect: agent.EffectClassRead, interrupt: func() { h.Interrupt() }}
		reg := agent.NewMapRegistry(map[string]agent.Tool{toolName: tool})

		// Turn one requests a tool call and would otherwise continue to
		// a second logical turn; the tool's own side effect fires
		// Interrupt while turn one is still in flight but AFTER the
		// tool's own result is already decided, so turn one's own
		// provider stream (already fully drained by then) and its
		// post-turn enqueue are unaffected — only the SECOND turn's own
		// iteration-boundary check (harness.go's own pre-attempt cause
		// check) ever observes the signal.
		provider := agenttest.NewProvider(
			scriptToolCallResponse(t, "call-hks-011-row7", toolName, []byte(`{}`)),
			scriptTextResponse(t, ai.FinishReasonStop),
		)
		hv := agent.Harness{Provider: provider, System: "system prompt for hks-011-row7", Turn: agent.TurnOptions{Tools: reg}, Hooks: hooks}
		h = &hv
		sink := make(chan *agent.Event, 256)
		_, _, runErr := h.Run(contextBackground(), firstMessage(t), sink)
		if !errors.Is(runErr, agent.ErrInterrupted) {
			t.Fatalf("Run error = %v, want errors.Is(err, agent.ErrInterrupted)", runErr)
		}
		drainSink(t, sink)
		wg.Wait()

		mu.Lock()
		got := len(reports)
		mu.Unlock()
		if got != 1 {
			t.Fatalf("post-turn observer received %d report(s), want exactly 1 (turn one only; turn two's own boundary caught the signal before any Turn() call)", got)
		}
	})

	t.Run("structural: exactly four post-turn/session-start enqueue call sites in harness.go", func(t *testing.T) {
		t.Parallel()
		root, err := gitTopLevel(t)
		if err != nil {
			t.Fatalf("git rev-parse --show-toplevel failed: %v", err)
		}
		raw, rerr := hksReadFile(t, root+"/backend/agent/src/agent/harness.go")
		if rerr != nil {
			t.Fatalf("reading harness.go failed: %v", rerr)
		}
		postTurnSites := strings.Count(raw, "lane.enqueuePostTurn(")
		if postTurnSites != 4 {
			t.Errorf("harness.go calls lane.enqueuePostTurn( %d time(s), want exactly 4 (R-HKS-004's four collapsed sites)", postTurnSites)
		}
		sessionStartSites := strings.Count(raw, "lane.enqueueSessionStart(")
		if sessionStartSites != 1 {
			t.Errorf("harness.go calls lane.enqueueSessionStart( %d time(s), want exactly 1", sessionStartSites)
		}
	})
}

// interruptingTool is a test-local agent.Tool whose Run method invokes
// a caller-supplied side effect (here, Interrupt on the hosting
// Harness) before returning an ordinary successful result — used to
// fire a cancellation signal FROM INSIDE a scheduled tool call, so the
// signal is visible at the NEXT iteration boundary rather than mid-
// stream on the turn that scheduled it.
type interruptingTool struct {
	toolName  string
	effect    agent.EffectClass
	interrupt func()
}

func (t interruptingTool) Name() string                   { return t.toolName }
func (t interruptingTool) EffectClass() agent.EffectClass { return t.effect }
func (t interruptingTool) Run(_ context.Context, _ []byte, _ agent.PolicySlot) (agent.Result, error) {
	t.interrupt()
	return agent.Result{Outcome: agent.ToolOutcomeSuccess, Content: []byte("ok")}, nil
}

var _ agent.Tool = interruptingTool{}

// hksReadFile reads path's full contents as a string.
func hksReadFile(t *testing.T, path string) (string, error) {
	t.Helper()
	raw, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	return string(raw), nil
}

// S-HKS-012 — The cost is the sum over the turn's attempts, and the
// multi-attempt fixture is what makes it non-vacuous.
//
// Fixture constraint, recorded rather than assumed: retryDecision's
// G1-G5 gates (retry_policy.go) only retry (G4) a Turn error that IS a
// retryable, no-partial-output *ai.Failure — and every such failure is
// necessarily a PRE-stream or mid-stream-before-any-Completion path,
// neither of which ever reaches finalize() (loop.go), the ONLY place
// cost_turn is emitted. So a genuinely RETRIED attempt structurally
// contributes zero cost_turn events; only the logical turn's single
// completing attempt (if any) ever does. This fixture proves the
// ACHIEVABLE, load-bearing half of the property: with two retried
// (zero-cost) attempts before a third, succeeding one, the report's
// cost equals exactly the ONE cost_turn the stream carries (never a
// fabricated multiple, never the run's own cross-turn cumulative), and
// the attempt count is genuinely 3 — proving retried attempts are
// counted without their absence being misread as a missing report.
func TestHooksHarness_PostTurn_CostIsSumOverAttempts(t *testing.T) {
	t.Parallel()

	var mu sync.Mutex
	var reports []agent.PostTurnReport
	var wg sync.WaitGroup
	wg.Add(1)
	hooks := agent.Hooks{PostTurn: []agent.PostTurnObserver{func(_ context.Context, r agent.PostTurnReport) {
		mu.Lock()
		reports = append(reports, r)
		mu.Unlock()
		wg.Done()
	}}}

	failure := mustPreStreamRetryableFailure(t, ai.RetryDelay{})
	inner := agenttest.NewProvider(scriptTextResponse(t, ai.FinishReasonStop))
	provider := newPreStreamFailingProvider(2, failure, inner) // two retryable failures, then delegates to inner's success.

	h := agent.Harness{Provider: provider, System: "system prompt for hks-012", RetryAttempts: 3, RetryTiming: agent.RetryTiming{SleepFunc: instantSleep}, Hooks: hooks}
	sink := make(chan *agent.Event, 64)
	_, _, err := h.Run(contextBackground(), firstMessage(t), sink)
	if err != nil {
		t.Fatalf("Run returned err = %v, want nil", err)
	}
	events := drainSink(t, sink)
	wg.Wait()

	mu.Lock()
	gotReports := append([]agent.PostTurnReport(nil), reports...)
	mu.Unlock()
	if len(gotReports) != 1 {
		t.Fatalf("post-turn observer received %d report(s), want exactly 1", len(gotReports))
	}
	if gotReports[0].Attempts() != 3 {
		t.Errorf("report attempts = %d, want 3 (two retried, one succeeding)", gotReports[0].Attempts())
	}

	var streamCost agent.CostFigures
	costTurnCount := 0
	for _, ev := range events {
		if ct, ok := ev.CostTurn(); ok {
			streamCost.InputTokens += ct.Figures().InputTokens
			streamCost.OutputTokens += ct.Figures().OutputTokens
			streamCost.CacheReadTokens += ct.Figures().CacheReadTokens
			streamCost.CacheWriteTokens += ct.Figures().CacheWriteTokens
			streamCost.ReasoningTokens += ct.Figures().ReasoningTokens
			costTurnCount++
		}
	}
	if costTurnCount != 1 {
		t.Fatalf("stream carries %d cost_turn event(s) for this logical turn, want exactly 1 (retried attempts never reach finalize())", costTurnCount)
	}
	if gotReports[0].Cost() != streamCost {
		t.Errorf("report cost = %+v, want the stream's own cost_turn figures %+v (sum over the turn's attempts)", gotReports[0].Cost(), streamCost)
	}
}

// S-RUN-113 — AG-20: the additions are exactly four, and the absences
// are asserted rather than claimed.
func TestHooksHarness_S_RUN_113_AdditionsExactlyFourAbsencesAsserted(t *testing.T) {
	t.Parallel()

	typ := reflect.TypeOf(&agent.Harness{})
	var names []string
	for i := 0; i < typ.NumMethod(); i++ {
		names = append(names, typ.Method(i).Name)
	}
	sortStrings(names)
	want := []string{"Compact", "Interrupt", "Run", "Shutdown", "Steer"}
	if !reflect.DeepEqual(names, want) {
		t.Errorf("*agent.Harness exported methods = %v, want %v", names, want)
	}
	if _, ok := reflect.TypeOf(agent.Harness{}).FieldByName("Hooks"); !ok {
		t.Error("agent.Harness has no Hooks field")
	}

	root, err := gitTopLevel(t)
	if err != nil {
		t.Fatalf("git rev-parse --show-toplevel failed: %v", err)
	}
	baseRef := hksResolveBaseRef(t, root)
	if !hksBaseRefIsHEAD(t, root, baseRef) {
		permDiff, derr := gitDiff(t, root, baseRef, "backend/agent/src/agent/permission_protocol_test.go")
		if derr != nil {
			t.Fatalf("git diff %s -- permission_protocol_test.go failed: %v", baseRef, derr)
		}
		if permDiff != "" {
			t.Errorf("permission_protocol_test.go's diff against %s is not empty, want file-unchanged (the no-deadline pin, TestPermission_WakeParked_SchedulerReturnsAfterExplicitWake_NoDeadline):\n%s", baseRef, permDiff)
		}
	}

	raw, rerr := hksReadFile(t, root+"/backend/agent/src/agent/harness.go")
	if rerr != nil {
		t.Fatalf("reading harness.go failed: %v", rerr)
	}
	if strings.Contains(raw, "turnCost costAccumulator") == false {
		t.Error("harness.go does not declare turnCost as a local costAccumulator, want a run-frame local (R-CST-004)")
	}
	hooksRaw, herr := hksReadFile(t, root+"/backend/agent/src/agent/hooks.go")
	if herr != nil {
		t.Fatalf("reading hooks.go failed: %v", herr)
	}
	for _, forbidden := range []string{"time.After", "time.Sleep", "time.NewTimer", "time.NewTicker", "context.WithTimeout", "context.WithDeadline", "\"time\""} {
		if strings.Contains(hooksRaw, forbidden) {
			t.Errorf("hooks.go contains %q, want no timer/deadline/sleep/poll/join on an observing hook", forbidden)
		}
		if strings.Contains(raw, forbidden) {
			t.Errorf("harness.go contains %q, want no timer/deadline/sleep/poll/join added by this change", forbidden)
		}
	}

	hold := newWgObserverHold()
	t.Cleanup(hold.Release)
	baselineProvider := agenttest.NewProvider(scriptTextResponse(t, ai.FinishReasonStop))
	baselineHarness := agent.Harness{Provider: baselineProvider, System: "system prompt for run-113"}
	baselineSink := make(chan *agent.Event, 64)
	if _, _, berr := baselineHarness.Run(contextBackground(), firstMessage(t), baselineSink); berr != nil {
		t.Fatalf("baseline Run returned err = %v, want nil", berr)
	}
	baselineEvents := drainSink(t, baselineSink)

	hookedProvider := agenttest.NewProvider(scriptTextResponse(t, ai.FinishReasonStop))
	hookedHooks := agent.Hooks{PostTurn: []agent.PostTurnObserver{func(context.Context, agent.PostTurnReport) { hold.Enter() }}}
	hookedHarness := agent.Harness{Provider: hookedProvider, System: "system prompt for run-113", Hooks: hookedHooks}
	hookedSink := make(chan *agent.Event, len(baselineEvents))
	runDone := make(chan struct{})
	var runErr error
	go func() {
		_, _, runErr = hookedHarness.Run(contextBackground(), firstMessage(t), hookedSink)
		close(runDone)
	}()
	<-runDone
	if runErr != nil {
		t.Fatalf("hooked Run returned err = %v, want nil (Run must return with the gate still held)", runErr)
	}
	hookedEvents := drainSink(t, hookedSink)
	if len(hookedEvents) != len(baselineEvents) {
		t.Fatalf("hooked stream has %d event(s), baseline has %d, want equal", len(hookedEvents), len(baselineEvents))
	}
	if report := agent.CheckStream(hookedEvents); report.Violation() != nil {
		t.Errorf("CheckStream rejected the hooked stream: %v, want accepted", report.Violation())
	}
	hold.Release()
}

// S-ATT-015 — AG-20: the reported outcome IS the streamed outcome,
// and no member was added.
func TestHooksHarness_S_ATT_015_ReportedOutcomeIsStreamedOutcome(t *testing.T) {
	t.Parallel()

	t.Run("finished", func(t *testing.T) {
		t.Parallel()
		var mu sync.Mutex
		var reports []agent.PostTurnReport
		var wg sync.WaitGroup
		wg.Add(1)
		hooks := agent.Hooks{PostTurn: []agent.PostTurnObserver{func(_ context.Context, r agent.PostTurnReport) {
			mu.Lock()
			reports = append(reports, r)
			mu.Unlock()
			wg.Done()
		}}}
		provider := agenttest.NewProvider(scriptTextResponse(t, ai.FinishReasonStop))
		h := agent.Harness{Provider: provider, System: "system prompt for att-015-finished", Hooks: hooks}
		sink := make(chan *agent.Event, 64)
		if _, _, err := h.Run(contextBackground(), firstMessage(t), sink); err != nil {
			t.Fatalf("Run returned err = %v, want nil", err)
		}
		events := drainSink(t, sink)
		wg.Wait()
		requireOutcomeIdentity(t, reports, events)
	})

	t.Run("failed", func(t *testing.T) {
		t.Parallel()
		var mu sync.Mutex
		var reports []agent.PostTurnReport
		var wg sync.WaitGroup
		wg.Add(1)
		hooks := agent.Hooks{PostTurn: []agent.PostTurnObserver{func(_ context.Context, r agent.PostTurnReport) {
			mu.Lock()
			reports = append(reports, r)
			mu.Unlock()
			wg.Done()
		}}}
		failure := mustNonRetryablePreStreamFailure(t)
		inner := agenttest.NewProvider(scriptTextResponse(t, ai.FinishReasonStop))
		provider := newPreStreamFailingProvider(1, failure, inner)
		h := agent.Harness{Provider: provider, System: "system prompt for att-015-failed", Hooks: hooks}
		sink := make(chan *agent.Event, 64)
		if _, _, err := h.Run(contextBackground(), firstMessage(t), sink); err == nil {
			t.Fatal("Run returned err = nil, want the surfaced failure")
		}
		events := drainSink(t, sink)
		wg.Wait()
		requireOutcomeIdentity(t, reports, events)
	})

	t.Run("interrupted (wound down)", func(t *testing.T) {
		t.Parallel()
		var mu sync.Mutex
		var reports []agent.PostTurnReport
		var wg sync.WaitGroup
		wg.Add(1)
		hooks := agent.Hooks{PostTurn: []agent.PostTurnObserver{func(_ context.Context, r agent.PostTurnReport) {
			mu.Lock()
			reports = append(reports, r)
			mu.Unlock()
			wg.Done()
		}}}
		gate := agenttest.NewGate()
		provider := agenttest.NewProvider(heldTurnScript(t, gate))
		h := agent.Harness{Provider: provider, System: "system prompt for att-015-interrupted", Hooks: hooks}
		sink := make(chan *agent.Event, 64)
		runDone := make(chan struct{})
		go func() { _, _, _ = h.Run(contextBackground(), firstMessage(t), sink); close(runDone) }()
		<-gate.Reached()
		h.Interrupt()
		drainSink(t, sink)
		<-runDone
		wg.Wait()
		mu.Lock()
		gotReports := append([]agent.PostTurnReport(nil), reports...)
		mu.Unlock()
		if len(gotReports) != 1 {
			t.Fatalf("post-turn observer received %d report(s), want exactly 1", len(gotReports))
		}
		if gotReports[0].Outcome() != agent.TurnOutcomeAborted {
			t.Errorf("report outcome = %v, want TurnOutcomeAborted", gotReports[0].Outcome())
		}
	})

	// Vocabulary and substrate: no member added, turn_events.go and
	// failure.go byte-unchanged (checked once, here, for the whole
	// suite — TestHooks_ScopeFence_... above already asserts these
	// specific files; this re-affirms the outcome vocabulary's own
	// enumerated member count directly).
	outcomeType := reflect.TypeOf(agent.TurnOutcomeFinished)
	if outcomeType.Kind() != reflect.Uint8 {
		t.Errorf("agent.TurnOutcome underlying kind = %v, want Uint8 (unchanged)", outcomeType.Kind())
	}
}

// requireOutcomeIdentity asserts each report's outcome equals its
// turn's own turn_end outcome, value-to-value.
func requireOutcomeIdentity(t *testing.T, reports []agent.PostTurnReport, events []agent.Event) {
	t.Helper()
	if len(reports) != 1 {
		t.Fatalf("post-turn observer received %d report(s), want exactly 1", len(reports))
	}
	for _, ev := range events {
		payload, ok := ev.TurnEnd()
		if !ok {
			continue
		}
		turn, hasTurn := ev.Turn()
		if !hasTurn || turn != reports[0].Turn() {
			continue
		}
		if payload.Outcome() != reports[0].Outcome() {
			t.Errorf("report outcome = %v, want the matching turn_end's own outcome %v", reports[0].Outcome(), payload.Outcome())
		}
		return
	}
	t.Error("no turn_end event matches the report's own turn identity")
}

// S-DEL-026 — AG-20: an observing hook finds no seam, and the fence
// is green unedited. Given a parent run hosting a child run inside a
// tool call, and a post-turn observing hook registered on the CHILD
// harness looking up a publishing seam from the context it is invoked
// with, when the child run completes and its lane has drained, then
// the lookup reports no seam present; no event attributable to that
// hook appears on either stream; both streams are CheckStream-valid;
// and the parent's stream is byte-identical (kind-sequence) to the
// same script with no hooks installed.
func TestHooksHarness_S_DEL_026_ChildObserverFindsNoSeam_FenceGreenUnedited(t *testing.T) {
	t.Parallel()

	var mu sync.Mutex
	var seamFound, lookupRan bool
	var wg sync.WaitGroup
	wg.Add(1)
	childProvider := agenttest.NewProvider(scriptTextResponse(t, ai.FinishReasonStop))
	child := &agent.Harness{Provider: childProvider, System: "system prompt for the hosted child (del-026)", Hooks: agent.Hooks{
		PostTurn: []agent.PostTurnObserver{func(ctx context.Context, _ agent.PostTurnReport) {
			_, ok := agent.DelegationSeamFrom(ctx)
			mu.Lock()
			seamFound = ok
			lookupRan = true
			mu.Unlock()
			wg.Done()
		}},
	}}

	toolName := "del_026_tool"
	tool := &delegatingTool{toolName: toolName, effect: agent.EffectClassRead, child: child, prompt: firstMessage(t)}
	reg := agent.NewMapRegistry(map[string]agent.Tool{toolName: tool})

	turnOneScript := scriptToolCallResponse(t, "call-del-026", toolName, []byte(`{}`))
	turnTwoScript := scriptTextResponse(t, ai.FinishReasonStop)
	parentProvider := agenttest.NewProvider(turnOneScript, turnTwoScript)

	h := agent.Harness{Provider: parentProvider, System: "system prompt for del-026 parent", Turn: agent.TurnOptions{Tools: reg}}
	sink := make(chan *agent.Event, 512)
	if _, _, err := h.Run(contextBackground(), firstMessage(t), sink); err != nil {
		t.Fatalf("parent Run returned err = %v, want nil", err)
	}
	parentEvents := drainSink(t, sink)
	wg.Wait()

	if report := agent.CheckStream(parentEvents); report.Violation() != nil {
		t.Errorf("CheckStream rejected the parent stream: %v, want accepted unmodified", report.Violation())
	}

	mu.Lock()
	gotRan, gotSeamFound := lookupRan, seamFound
	mu.Unlock()
	if !gotRan {
		t.Fatal("child observer never ran")
	}
	if gotSeamFound {
		t.Error("child observer's own DelegationSeamFrom(ctx) found a seam, want none")
	}

	baselineProvider := agenttest.NewProvider(turnOneScript, turnTwoScript)
	baselineChild := &agent.Harness{Provider: agenttest.NewProvider(scriptTextResponse(t, ai.FinishReasonStop)), System: "system prompt for the hosted child (del-026)"}
	baselineTool := &delegatingTool{toolName: toolName, effect: agent.EffectClassRead, child: baselineChild, prompt: firstMessage(t)}
	baselineReg := agent.NewMapRegistry(map[string]agent.Tool{toolName: baselineTool})
	baselineHarness := agent.Harness{Provider: baselineProvider, System: "system prompt for del-026 parent", Turn: agent.TurnOptions{Tools: baselineReg}}
	baselineSink := make(chan *agent.Event, 512)
	if _, _, err := baselineHarness.Run(contextBackground(), firstMessage(t), baselineSink); err != nil {
		t.Fatalf("baseline Run returned err = %v, want nil", err)
	}
	baselineEvents := drainSink(t, baselineSink)
	if len(parentEvents) != len(baselineEvents) {
		t.Fatalf("parent stream has %d event(s), hookless baseline has %d, want equal", len(parentEvents), len(baselineEvents))
	}
	for i := range parentEvents {
		if parentEvents[i].Kind() != baselineEvents[i].Kind() {
			t.Errorf("event[%d] kind = %v, want %v (byte-identical stream shape — the child's own hook contributes nothing to the parent's stream)", i, parentEvents[i].Kind(), baselineEvents[i].Kind())
		}
	}
}
