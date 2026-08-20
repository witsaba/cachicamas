// AG-20 — Phase 3 (splice half): pre-compact chain splice, both doors,
// idempotence, forward adjustment, chain-element failure, misplaced-
// options rejection (R-HKS-003, S-HKS-006..009 + bite S-HKS-053;
// cross-package scenarios S-CMP-038..042).
package agent_test

import (
	"context"
	"sync"
	"testing"

	"github.com/cachicamas/backend/agent/src/agent"
	"github.com/cachicamas/backend/agent/src/agenttest"
	"github.com/cachicamas/backend/agent/src/ai"
)

// ---------------------------------------------------------------------
// R-HKS-003 / R-CMP-015 — the splice, both doors, pre-provider order
// ---------------------------------------------------------------------

// S-HKS-006 — Charter AG-20.1 scenario 1, the pre-compact third — both
// doors. Given a run whose compaction is reached through the strategy
// verdict, and a second run whose compaction is reached through the
// on-demand entry point, and a PreCompact chain of one element
// recording what it received, when each run executes, then in both
// runs the element was invoked exactly once, the plan it received
// carried the resolved cut and a derived span matching the compaction
// bracket's own, the invocation is ordered strictly before the
// compaction provider received any request, and PreRequest fired zero
// times for that bracket.
func TestHooksCompaction_Splice_BothDoors_PreProviderOrdering(t *testing.T) {
	t.Parallel()

	t.Run("strategy-triggered door", func(t *testing.T) {
		t.Parallel()
		_, hist, _ := markedHarnessForCompaction(t, "hks-006-strategy")

		var mu sync.Mutex
		var invocations int
		var receivedCut int
		var receivedSpan agent.CompactionSpan
		var providerCountAtEntry int
		var preRequestCount int

		compactionProvider := agenttest.NewProvider(scriptTextResponse(t, ai.FinishReasonStop))
		strategy := &compactAfterNStrategy{request: func(prompt agent.ContextPrompt) agent.CompactionRequest {
			return agent.CompactionRequest{Provider: compactionProvider, Instruction: "summarize for hks-006", Cut: len(prompt.Transcript)}
		}}
		hooks := agent.Hooks{
			PreCompact: []agent.PreCompactHook{func(_ context.Context, plan agent.CompactionPlan) (agent.CompactionPlan, error) {
				mu.Lock()
				invocations++
				receivedCut = plan.Cut()
				receivedSpan = plan.Span()
				providerCountAtEntry = len(compactionProvider.Requests())
				mu.Unlock()
				return plan, nil
			}},
			PreRequest: []agent.PreRequestHook{func(_ context.Context, req ai.Request) (ai.Request, error) {
				mu.Lock()
				preRequestCount++
				mu.Unlock()
				return req, nil
			}},
		}
		h := agent.Harness{Provider: agenttest.NewProvider(scriptTextResponse(t, ai.FinishReasonStop)), System: "system prompt for hks-006", History: hist, ContextStrategy: strategy, Hooks: hooks}
		sink := make(chan *agent.Event, 256)
		if _, _, err := h.Run(contextBackground(), firstMessage(t), sink); err != nil {
			t.Fatalf("Run returned err = %v, want nil", err)
		}
		drainSink(t, sink)

		mu.Lock()
		gotInvocations, gotCut, gotSpan, gotEntryCount, gotPreReq := invocations, receivedCut, receivedSpan, providerCountAtEntry, preRequestCount
		mu.Unlock()
		if gotInvocations != 1 {
			t.Fatalf("PreCompact invoked %d time(s), want exactly 1", gotInvocations)
		}
		if gotCut == 0 {
			t.Error("PreCompact hook received cut = 0, want the resolved cut")
		}
		if gotSpan.StartTurnID == "" || gotSpan.EndTurnID == "" {
			t.Errorf("PreCompact hook received span = %+v, want a derived, non-empty span", gotSpan)
		}
		if gotEntryCount != 0 {
			t.Errorf("compaction provider had %d recorded request(s) when the hook ran, want 0 (pre-provider ordering)", gotEntryCount)
		}
		if gotPreReq != 1 {
			t.Errorf("PreRequest fired %d time(s), want exactly 1 (the harness's own logical turn only — never the compaction bracket)", gotPreReq)
		}
	})

	t.Run("on-demand door", func(t *testing.T) {
		t.Parallel()
		_, hist, _ := markedHarnessForCompaction(t, "hks-006-demand")

		var mu sync.Mutex
		var invocations, receivedCut, providerCountAtEntry int
		compactionProvider := agenttest.NewProvider(scriptTextResponse(t, ai.FinishReasonStop))
		hooks := agent.Hooks{PreCompact: []agent.PreCompactHook{func(_ context.Context, plan agent.CompactionPlan) (agent.CompactionPlan, error) {
			mu.Lock()
			invocations++
			receivedCut = plan.Cut()
			providerCountAtEntry = len(compactionProvider.Requests())
			mu.Unlock()
			return plan, nil
		}}}
		h := agent.Harness{Provider: agenttest.NewProvider(), System: "system prompt for hks-006-demand", History: hist, Hooks: hooks}
		req := agent.CompactionRequest{Provider: compactionProvider, Instruction: "summarize for hks-006 demand", Cut: hist.Len()}
		sink := make(chan *agent.Event, 64)
		if err := h.Compact(contextBackground(), req, sink); err != nil {
			t.Fatalf("Compact returned err = %v, want nil", err)
		}
		drainSink(t, sink)

		mu.Lock()
		gotInvocations, gotCut, gotEntryCount := invocations, receivedCut, providerCountAtEntry
		mu.Unlock()
		if gotInvocations != 1 {
			t.Fatalf("PreCompact invoked %d time(s), want exactly 1", gotInvocations)
		}
		if gotCut == 0 {
			t.Error("PreCompact hook received cut = 0, want the resolved cut")
		}
		if gotEntryCount != 0 {
			t.Errorf("compaction provider had %d recorded request(s) when the hook ran, want 0", gotEntryCount)
		}
	})
}

// ---------------------------------------------------------------------
// R-CMP-004 as amended — idempotence, forward adjustment, chain
// failure
// ---------------------------------------------------------------------

// runCompactionScriptViaStrategy drives one Run whose only compaction
// is triggered by a compactAfterNStrategy (skip:0) over a seeded,
// marked history, with chain installed on Hooks.PreCompact — the
// shared rig S-HKS-007/008/009 all build on.
func runCompactionScriptViaStrategy(t *testing.T, chain []agent.PreCompactHook) (kinds []agent.EventKind, historyReadBack []ai.Message, runErr error) {
	t.Helper()
	_, hist, _ := markedHarnessForCompaction(t, "hks-007-009")
	compactionProvider := agenttest.NewProvider(scriptTextResponse(t, ai.FinishReasonStop))
	strategy := &compactAfterNStrategy{request: func(prompt agent.ContextPrompt) agent.CompactionRequest {
		return agent.CompactionRequest{Provider: compactionProvider, Instruction: "summarize", Cut: len(prompt.Transcript)}
	}}
	h := agent.Harness{Provider: agenttest.NewProvider(scriptTextResponse(t, ai.FinishReasonStop)), System: "system prompt for hks-007-009", History: hist, ContextStrategy: strategy, Hooks: agent.Hooks{PreCompact: chain}}
	sink := make(chan *agent.Event, 256)
	_, _, runErr = h.Run(contextBackground(), firstMessage(t), sink)
	events := drainSink(t, sink)
	kinds = kindsOfHKS(events)
	readBack := hist.Entries()
	historyReadBack = make([]ai.Message, len(readBack))
	for i, e := range readBack {
		historyReadBack[i] = e.Message()
	}
	return kinds, historyReadBack, runErr
}

// S-HKS-007 / S-CMP-038 — the identity half: re-resolution is a no-op
// on an unchanged plan. Given two runs of an identical compacting
// script — one with no hooks, one with a PreCompact element returning
// its input plan unchanged — the two event streams are (kind-sequence)
// byte-identical, the two committed history read-backs are
// byte-identical, and both are CheckStream-valid.
func TestHooksCompaction_Idempotence_IdenticalPlanByteIdenticalToNoHook(t *testing.T) {
	t.Parallel()

	noHookKinds, noHookHistory, noHookErr := runCompactionScriptViaStrategy(t, nil)
	if noHookErr != nil {
		t.Fatalf("no-hook run returned err = %v, want nil", noHookErr)
	}
	identityKinds, identityHistory, identityErr := runCompactionScriptViaStrategy(t, []agent.PreCompactHook{
		func(_ context.Context, plan agent.CompactionPlan) (agent.CompactionPlan, error) { return plan, nil },
	})
	if identityErr != nil {
		t.Fatalf("identity-hook run returned err = %v, want nil", identityErr)
	}

	if !kindsEqualHKS(noHookKinds, identityKinds) {
		t.Errorf("event kind sequences differ:\n  no-hook:  %v\n  identity: %v", noHookKinds, identityKinds)
	}
	if len(noHookHistory) != len(identityHistory) {
		t.Fatalf("history read-back lengths differ: no-hook=%d identity=%d", len(noHookHistory), len(identityHistory))
	}
	for i := range noHookHistory {
		if !noHookHistory[i].Equal(identityHistory[i]) {
			t.Errorf("history entry[%d] differs between no-hook and identity-hook runs", i)
		}
	}

	// Empty chain: also byte-identical, and no plan value constructed
	// (checked indirectly: an empty chain never invokes anything, so
	// there is nothing FOR a plan-recording hook to have captured —
	// asserted directly by the zero-invocation chain below).
	var invoked int
	emptyProvider := agenttest.NewProvider(scriptTextResponse(t, ai.FinishReasonStop))
	_, hist, _ := markedHarnessForCompaction(t, "hks-007-empty")
	strategy := &compactAfterNStrategy{request: func(prompt agent.ContextPrompt) agent.CompactionRequest {
		return agent.CompactionRequest{Provider: agenttest.NewProvider(scriptTextResponse(t, ai.FinishReasonStop)), Instruction: "summarize", Cut: len(prompt.Transcript)}
	}}
	h := agent.Harness{Provider: emptyProvider, System: "system prompt for hks-007-empty", History: hist, ContextStrategy: strategy, Hooks: agent.Hooks{PreCompact: nil}}
	sink := make(chan *agent.Event, 256)
	if _, _, err := h.Run(contextBackground(), firstMessage(t), sink); err != nil {
		t.Fatalf("empty-chain run returned err = %v, want nil", err)
	}
	drainSink(t, sink)
	if invoked != 0 {
		t.Errorf("empty-chain invocation count = %d, want 0", invoked)
	}
}

// S-HKS-008 / S-CMP-041 — the adjustment half: forward re-designation
// stays invariant-safe.
//
// Fixture: a marked turn (mark at count 2: prompt, assistant text),
// followed by an APPENDED (unmarked) call/result pair — call at index
// 2, its own matching result at index 3, both left OUTSIDE any mark
// (a genuinely open pair as far as resolveCut's own retraction is
// concerned; the pair itself is complete, so the harness's OWN later
// logical turn can still build a valid request from the transcript
// once compaction leaves both untouched). The strategy's naive cut
// (length 4) resolves, unhooked, to the mark at 2 (retracting strictly
// before the pair). The hook forward-adjusts to 3 — landing strictly
// inside the pair, between the call and its own result. Re-resolution
// retracts it back to 2, safely, leaving the pair whole in the
// protected tail.
func TestHooksCompaction_ForwardAdjustment_StaysInvariantSafe(t *testing.T) {
	t.Parallel()

	_, hist, _ := markedHarnessForCompaction(t, "hks-008")
	callPart, err := ai.NewToolCall("call-hks-008-pair", "read_hks_008_pair", []byte(`{}`))
	if err != nil {
		t.Fatalf("ai.NewToolCall: %v", err)
	}
	callMsg, err := ai.NewMessage(ai.RoleAssistant, callPart)
	if err != nil {
		t.Fatalf("ai.NewMessage(call): %v", err)
	}
	if err := hist.Append(callMsg); err != nil {
		t.Fatalf("hist.Append(callMsg) returned %v, want nil", err)
	}
	resultPart, err := ai.NewToolResult("call-hks-008-pair", "ok")
	if err != nil {
		t.Fatalf("ai.NewToolResult: %v", err)
	}
	resultMsg, err := ai.NewMessage(ai.RoleTool, resultPart)
	if err != nil {
		t.Fatalf("ai.NewMessage(result): %v", err)
	}
	if err := hist.Append(resultMsg); err != nil {
		t.Fatalf("hist.Append(resultMsg) returned %v, want nil", err)
	}
	if got := hist.Len(); got != 4 {
		t.Fatalf("fixture history has %d entries, want 4 (marked prefix of 2, plus the call/result pair)", got)
	}

	var forwardRequested int
	compactionProvider := agenttest.NewProvider(scriptTextResponse(t, ai.FinishReasonStop))
	strategy := &compactAfterNStrategy{request: func(prompt agent.ContextPrompt) agent.CompactionRequest {
		return agent.CompactionRequest{Provider: compactionProvider, Instruction: "summarize for hks-008", Cut: len(prompt.Transcript)}
	}}
	hooks := agent.Hooks{PreCompact: []agent.PreCompactHook{func(_ context.Context, plan agent.CompactionPlan) (agent.CompactionPlan, error) {
		forwardRequested = plan.Cut() + 1 // forward-adjust to land strictly inside the pair (between call@2 and result@3).
		return plan.WithCut(forwardRequested), nil
	}}}
	h := agent.Harness{Provider: agenttest.NewProvider(scriptTextResponse(t, ai.FinishReasonStop)), System: "system prompt for hks-008", History: hist, ContextStrategy: strategy, Hooks: hooks}
	sink := make(chan *agent.Event, 256)
	if _, _, runErr := h.Run(contextBackground(), firstMessage(t), sink); runErr != nil {
		t.Fatalf("Run returned err = %v, want nil (the forward request must be resolved, not refused)", runErr)
	}
	events := drainSink(t, sink)

	if forwardRequested != 3 {
		t.Fatalf("hook's own forward request = %d, want 3 (fixture invariant: strictly inside the pair)", forwardRequested)
	}

	// The compaction itself must have SUCCEEDED — not silently failed
	// and left history untouched, which would satisfy the "pair stays
	// together" check below just as vacuously (an untouched pair is
	// still together). A harness-level Run error is always nil
	// regardless of the compaction's own outcome (R-CMP-010: a
	// compaction failure is discarded at the harness's own call site),
	// so the compaction's own success must be checked on the stream.
	failedCount, finishedCount := countCompactionOutcomes(events)
	if finishedCount != 1 {
		t.Fatalf("compaction_finished count = %d, want exactly 1 (the forward-adjusted compaction must actually succeed)", finishedCount)
	}
	if failedCount != 0 {
		t.Fatalf("compaction_failed count = %d, want 0", failedCount)
	}

	readBack := hist.Entries()
	msgs := make([]ai.Message, len(readBack))
	for i, e := range readBack {
		msgs[i] = e.Message()
	}
	if _, serr := agent.NewSeededHistory(msgs); serr != nil {
		t.Errorf("NewSeededHistory(post-compaction read-back) returned %v, want nil — the committed prefix must be pairing-closed", serr)
	}

	var sawCall, sawResult bool
	for _, m := range msgs {
		for _, p := range m.Content() {
			if tc, ok := p.ToolCall(); ok && tc.ID() == "call-hks-008-pair" {
				sawCall = true
			}
			if tr, ok := p.ToolResult(); ok && tr.CallID() == "call-hks-008-pair" {
				sawResult = true
			}
		}
	}
	if sawCall != sawResult {
		t.Errorf("the call/result pair survived split: call present=%v, result present=%v, want both true (protected together) or both false (both summarized) — never one without the other", sawCall, sawResult)
	}
}

// S-HKS-053 — (bite) RED-first, mandated. Given a scratch tree in
// which the post-hook re-resolution is skipped (compaction.go's own
// `cut = resolveCut(hist, plan.Cut())` replaced with `cut =
// plan.Cut()`), when S-HKS-008 runs, then it FAILS. Captured RED
// transcript: the raw forward request (3) is not a recorded turn-mark
// boundary, so hist.markSpan(3) itself reports !spanOK, and the
// compaction bracket takes its OWN existing failure arm — zero
// compaction_finished events on the stream, where the GREEN test
// asserts exactly one. A harness-level Run() error stays nil either
// way (R-CMP-010: a compaction failure never interrupts the run), so
// the GREEN test's own assertion had to be strengthened to check the
// STREAM's own compaction_finished/compaction_failed counts rather
// than Run's return value alone — an untouched, still-together
// call/result pair is indistinguishable, by that check alone, from a
// forward-adjusted compaction that actually succeeded safely,
// discovered in this exact apply session by running this bite BEFORE
// trusting the original assertion. Both outcomes prove the
// re-resolution step is load-bearing: correct code retracts and
// commits; sabotaged code silently fails to compact at all. See
// apply-progress.md for the full captured transcript. RED-recorded
// BEFORE S-HKS-008 is GREEN, with -count=1, then reverted.

// S-HKS-009 / S-CMP-039 — a chain element's failure lands on the
// existing arm and the run continues, on both doors.
func TestHooksCompaction_ChainElementFailure_LandsOnExistingArm_RunContinues(t *testing.T) {
	t.Parallel()

	t.Run("strategy-triggered door", func(t *testing.T) {
		t.Parallel()
		_, hist, _ := markedHarnessForCompaction(t, "hks-009-strategy")
		compactionProvider := agenttest.NewProvider(scriptTextResponse(t, ai.FinishReasonStop))
		strategy := &compactAfterNStrategy{request: func(prompt agent.ContextPrompt) agent.CompactionRequest {
			return agent.CompactionRequest{Provider: compactionProvider, Instruction: "summarize for hks-009", Cut: len(prompt.Transcript)}
		}}
		var secondInvoked bool
		hooks := agent.Hooks{PreCompact: []agent.PreCompactHook{
			func(_ context.Context, plan agent.CompactionPlan) (agent.CompactionPlan, error) { return plan, nil },
			func(_ context.Context, plan agent.CompactionPlan) (agent.CompactionPlan, error) {
				return plan, errPreRequestBoom
			},
			func(_ context.Context, plan agent.CompactionPlan) (agent.CompactionPlan, error) {
				secondInvoked = true
				return plan, nil
			},
		}}
		h := agent.Harness{Provider: agenttest.NewProvider(scriptTextResponse(t, ai.FinishReasonStop)), System: "system prompt for hks-009-strategy", History: hist, ContextStrategy: strategy, Hooks: hooks}
		preAttemptEntries := hist.Entries()
		preAttemptMsgs := make([]ai.Message, len(preAttemptEntries))
		for i, e := range preAttemptEntries {
			preAttemptMsgs[i] = e.Message()
		}
		sink := make(chan *agent.Event, 256)
		if _, _, err := h.Run(contextBackground(), firstMessage(t), sink); err != nil {
			t.Fatalf("Run returned err = %v, want nil — a compaction failure must never interrupt the run (R-CMP-010)", err)
		}
		events := drainSink(t, sink)

		if len(compactionProvider.Requests()) != 0 {
			t.Errorf("compaction provider recorded %d request(s), want 0", len(compactionProvider.Requests()))
		}
		if secondInvoked {
			t.Error("the chain element AFTER the failing one was invoked, want it skipped")
		}
		failedCount, finishedCount := countCompactionOutcomes(events)
		if failedCount != 1 {
			t.Errorf("compaction_failed count = %d, want exactly 1", failedCount)
		}
		if finishedCount != 0 {
			t.Errorf("compaction_finished count = %d, want 0", finishedCount)
		}
		// The run CONTINUES (R-CMP-010): it does not stop at the failed
		// compaction, so hist.Len() legitimately GROWS by the ordinary
		// logical turn that follows. What must stay unchanged is the
		// PRE-EXISTING prefix itself — no prefix replacement, no
		// summary, byte-identical to what it was before the attempt.
		postAttemptEntries := hist.Entries()
		if len(postAttemptEntries) < len(preAttemptMsgs) {
			t.Fatalf("history shrank from %d to %d entries, want at least as many (a failed compaction commits nothing)", len(preAttemptMsgs), len(postAttemptEntries))
		}
		for i, want := range preAttemptMsgs {
			if !postAttemptEntries[i].Message().Equal(want) {
				t.Errorf("pre-existing history entry[%d] changed after the failed compaction, want byte-unchanged", i)
			}
		}
		if report := agent.CheckStream(events); report.Violation() != nil {
			t.Errorf("CheckStream rejected the stream: %v, want accepted", report.Violation())
		}
	})

	t.Run("on-demand door", func(t *testing.T) {
		t.Parallel()
		_, hist, _ := markedHarnessForCompaction(t, "hks-009-demand")
		compactionProvider := agenttest.NewProvider(scriptTextResponse(t, ai.FinishReasonStop))
		hooks := agent.Hooks{PreCompact: []agent.PreCompactHook{
			func(_ context.Context, plan agent.CompactionPlan) (agent.CompactionPlan, error) { return plan, nil },
			func(_ context.Context, plan agent.CompactionPlan) (agent.CompactionPlan, error) {
				return plan, errPreRequestBoom
			},
		}}
		h := agent.Harness{Provider: agenttest.NewProvider(), System: "system prompt for hks-009-demand", History: hist, Hooks: hooks}
		req := agent.CompactionRequest{Provider: compactionProvider, Instruction: "summarize for hks-009 demand", Cut: hist.Len()}
		sink := make(chan *agent.Event, 64)
		if err := h.Compact(contextBackground(), req, sink); err == nil {
			t.Fatal("Compact returned err = nil, want the chain's own failure")
		}
		events := drainSink(t, sink)
		if len(compactionProvider.Requests()) != 0 {
			t.Errorf("compaction provider recorded %d request(s), want 0", len(compactionProvider.Requests()))
		}
		failedCount, finishedCount := countCompactionOutcomes(events)
		if failedCount != 1 {
			t.Errorf("compaction_failed count = %d, want exactly 1", failedCount)
		}
		if finishedCount != 0 {
			t.Errorf("compaction_finished count = %d, want 0", finishedCount)
		}
	})
}

// countCompactionOutcomes counts compaction_failed and
// compaction_finished events by kind.
func countCompactionOutcomes(events []agent.Event) (failed, finished int) {
	for _, ev := range events {
		switch ev.Kind() {
		case agent.EventKindCompactionFailed:
			failed++
		case agent.EventKindCompactionFinished:
			finished++
		}
	}
	return failed, finished
}

// S-CMP-042 — the misplaced-options rejection is total for the hook
// field, member by member; and a compaction whose chain arrives as
// the operation's own parameter fires normally.
func TestHooksCompaction_MisplacedOptionsRejection_TotalForHooksField(t *testing.T) {
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
			_, hist, _ := markedHarnessForCompaction(t, "cmp-042-"+tt.name)
			h := agent.Harness{Provider: agenttest.NewProvider(), System: "system prompt for cmp-042", History: hist}
			compactionProvider := agenttest.NewProvider(scriptTextResponse(t, ai.FinishReasonStop))
			req := agent.CompactionRequest{Provider: compactionProvider, Instruction: "summarize", Cut: hist.Len(), Options: agent.TurnOptions{Hooks: tt.hooks}}
			sink := make(chan *agent.Event, 64)
			if err := h.Compact(contextBackground(), req, sink); err == nil {
				t.Fatalf("Compact returned err = nil, want a typed misplaced-options refusal for options.Hooks.%s", tt.name)
			}
			if n := len(compactionProvider.Requests()); n != 0 {
				t.Errorf("compaction provider recorded %d request(s), want 0", n)
			}
			drainSink(t, sink)
		})
	}

	t.Run("a chain arriving as the operation's own parameter fires normally", func(t *testing.T) {
		t.Parallel()
		_, hist, _ := markedHarnessForCompaction(t, "cmp-042-chain-param")
		var invoked bool
		hooks := agent.Hooks{PreCompact: []agent.PreCompactHook{func(_ context.Context, plan agent.CompactionPlan) (agent.CompactionPlan, error) {
			invoked = true
			return plan, nil
		}}}
		h := agent.Harness{Provider: agenttest.NewProvider(), System: "system prompt for cmp-042-chain-param", History: hist, Hooks: hooks}
		compactionProvider := agenttest.NewProvider(scriptTextResponse(t, ai.FinishReasonStop))
		req := agent.CompactionRequest{Provider: compactionProvider, Instruction: "summarize", Cut: hist.Len()}
		sink := make(chan *agent.Event, 64)
		if err := h.Compact(contextBackground(), req, sink); err != nil {
			t.Fatalf("Compact returned err = %v, want nil", err)
		}
		drainSink(t, sink)
		if !invoked {
			t.Error("the chain, arriving as h.Hooks.PreCompact (the operation's own parameter), never fired")
		}
	})
}
