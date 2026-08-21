// AG-22 — Phase 4 (RED, design D-G scenario 4): the exactly-once proof
// (R-AGO-007) over every exit family, table-driven, mirroring the
// AG-21/AG-14 wind-down discipline in telemetry form. Every row drives
// its own fresh tracetest.Provider through Harness.TracerProvider and
// asserts AssertAllEndedOnce() plus the started-count-equals-ended-count
// invariant. The genuine RED right now (design's bite, task 4.4): every
// row fails on zero spans started — the run/turn/tool seams are not
// instrumented until Phases 5-7. -count=1 always: a cached result is not
// evidence for a table this timing-sensitive.
package agent_test

import (
	"context"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"testing"

	"github.com/cachicamas/backend/agent/src/agent"
	"github.com/cachicamas/backend/agent/src/agenttest"
	"github.com/cachicamas/backend/agent/src/agenttest/tracetest"
	"github.com/cachicamas/backend/agent/src/ai"
)

// TestObservability_Lifecycle_ExactlyOnceAcrossEveryExitFamily is
// R-AGO-007's own table: normal completion, two typed mid-stream failure
// categories, a tool execution failure, a contained tool panic,
// scheduler-observed cancellation, and the detached wind-down arm — one
// subtest per exit family, so each clears independently as Phases 5-7
// land.
func TestObservability_Lifecycle_ExactlyOnceAcrossEveryExitFamily(t *testing.T) {
	rows := []struct {
		name   string
		driver func(t *testing.T) *tracetest.Provider
	}{
		{"normal_completion", agoLifecycleNormalCompletion},
		{"run_failed_category_unavailable", func(t *testing.T) *tracetest.Provider {
			return agoLifecycleMidStreamFailure(t, ai.FailureCategoryUnavailable)
		}},
		{"run_failed_category_rate_limit", func(t *testing.T) *tracetest.Provider {
			return agoLifecycleMidStreamFailure(t, ai.FailureCategoryRateLimit)
		}},
		{"tool_execution_failure", agoLifecycleToolExecutionFailure},
		{"tool_panic_contained", agoLifecycleToolPanicContained},
		{"cancellation_interrupted", agoLifecycleCancellationInterrupted},
		{"detached_wind_down", agoLifecycleDetachedWindDown},
	}

	for _, row := range rows {
		t.Run(row.name, func(t *testing.T) {
			provider := row.driver(t)

			if err := provider.AssertAllEndedOnce(); err != nil {
				t.Errorf("AssertAllEndedOnce() = %v, want nil (R-AGO-007's own exactly-once proof)", err)
			}
			if got := provider.Started(); got == 0 {
				t.Fatalf("provider.Started() = 0, want > 0 — no span was recorded on this exit family yet (the relevant seam is not instrumented until Phases 5-7)")
			}
			if started, ended := provider.Started(), provider.Ended(); started != ended {
				t.Errorf("provider.Started() = %d, provider.Ended() = %d, want equal (R-AGO-007's started-count-equals-ended-count invariant)", started, ended)
			}
		})
	}
}

// TestObservability_Lifecycle_CompactionExactlyOnceAcrossEveryExitFamily
// is R-AGO-007's own compaction-family exit table (AG-22 correction,
// MAJOR-6, sdd-verify round 1, S-AGO-053: "a table of compaction exits —
// the success tail and each failure arm"). The run/turn/tool table above
// never drove a single compaction call — this table closes that gap for
// every REACHABLE exit runCompaction's own source names: the success
// tail, and the empty-instruction, misplaced-options, cut-resolves-to-
// zero, non-Stop-finish and mid-call-terminal-failure arms of
// emitCompactionFailedArm. (The span-derivation-failure and
// ReplacePrefix-commit-failure arms are structurally unreachable given
// resolveCut's own contract, per compaction.go's own comments, and are
// not driven here for the same reason the two "unreachable defensive
// branches" elsewhere in this package are disclosed rather than forced.)
func TestObservability_Lifecycle_CompactionExactlyOnceAcrossEveryExitFamily(t *testing.T) {
	rows := []struct {
		name   string
		driver func(t *testing.T) *tracetest.Provider
	}{
		{"compaction_finished", agoLifecycleCompactionFinished},
		{"compaction_failed_empty_instruction", agoLifecycleCompactionFailedEmptyInstruction},
		{"compaction_failed_misplaced_options", agoLifecycleCompactionFailedMisplacedOptions},
		{"compaction_failed_cut_zero", agoLifecycleCompactionFailedCutZero},
		{"compaction_failed_non_stop_finish", agoLifecycleCompactionFailedNonStopFinish},
		{"compaction_failed_mid_call", agoLifecycleCompactionFailedMidCall},
	}

	for _, row := range rows {
		t.Run(row.name, func(t *testing.T) {
			provider := row.driver(t)

			if err := provider.AssertAllEndedOnce(); err != nil {
				t.Errorf("AssertAllEndedOnce() = %v, want nil (R-AGO-007's own exactly-once proof, S-AGO-053)", err)
			}
			if got := provider.Started(); got == 0 {
				t.Fatalf("provider.Started() = 0, want > 0 — no span was recorded on this exit family")
			}
			if started, ended := provider.Started(), provider.Ended(); started != ended {
				t.Errorf("provider.Started() = %d, provider.Ended() = %d, want equal (R-AGO-007's started-count-equals-ended-count invariant)", started, ended)
			}
		})
	}
}

// agoLifecycleCompactionFinished drives a compaction that FINISHES —
// the success tail row.
func agoLifecycleCompactionFinished(t *testing.T) *tracetest.Provider {
	t.Helper()
	provider := tracetest.NewProvider()
	h, hist, _ := markedHarnessForCompaction(t, "cmp-lifecycle-finished")
	h.TracerProvider = provider
	compactionModelProvider := agenttest.NewProvider(scriptTextResponse(t, ai.FinishReasonStop))
	req := agent.CompactionRequest{Provider: compactionModelProvider, Instruction: "summarize for the compaction lifecycle table", Cut: hist.Len()}
	sink := make(chan *agent.Event, 64)
	if err := h.Compact(contextBackground(), req, sink); err != nil {
		t.Fatalf("Compact returned err = %v, want nil", err)
	}
	drainSink(t, sink)
	return provider
}

// agoLifecycleCompactionFailedEmptyInstruction drives Compact with an
// empty Instruction — runCompaction's own first refusal.
func agoLifecycleCompactionFailedEmptyInstruction(t *testing.T) *tracetest.Provider {
	t.Helper()
	provider := tracetest.NewProvider()
	h, hist, _ := markedHarnessForCompaction(t, "cmp-lifecycle-empty-instr")
	h.TracerProvider = provider
	req := agent.CompactionRequest{Provider: agenttest.NewProvider(), Instruction: "", Cut: hist.Len()}
	sink := make(chan *agent.Event, 64)
	if err := h.Compact(contextBackground(), req, sink); err == nil {
		t.Fatal("Compact returned err = nil, want a non-nil error (empty instruction)")
	}
	drainSink(t, sink)
	return provider
}

// agoLifecycleCompactionFailedMisplacedOptions drives Compact with
// Options.Tools set — the misplaced-options refusal (R-HKS-001,
// R-CMP-015).
func agoLifecycleCompactionFailedMisplacedOptions(t *testing.T) *tracetest.Provider {
	t.Helper()
	provider := tracetest.NewProvider()
	h, hist, _ := markedHarnessForCompaction(t, "cmp-lifecycle-misplaced")
	h.TracerProvider = provider
	toolName := "cmp_lifecycle_misplaced_tool"
	reg := agent.NewMapRegistry(map[string]agent.Tool{toolName: EchoScriptedTool(toolName, agent.EffectClassRead)})
	req := agent.CompactionRequest{
		Provider:    agenttest.NewProvider(),
		Instruction: "summarize",
		Cut:         hist.Len(),
		Options:     agent.TurnOptions{Tools: reg},
	}
	sink := make(chan *agent.Event, 64)
	if err := h.Compact(contextBackground(), req, sink); err == nil {
		t.Fatal("Compact returned err = nil, want a non-nil error (misplaced Tools option)")
	}
	drainSink(t, sink)
	return provider
}

// agoLifecycleCompactionFailedCutZero drives Compact against a FRESH
// history with no committed turn marks at all, so resolveCut resolves
// to 0 — the cut-resolves-to-zero refusal.
func agoLifecycleCompactionFailedCutZero(t *testing.T) *tracetest.Provider {
	t.Helper()
	provider := tracetest.NewProvider()
	h := &agent.Harness{
		Provider:       agenttest.NewProvider(),
		System:         "system prompt for compaction lifecycle cut-zero",
		History:        agent.NewHistory(),
		TracerProvider: provider,
	}
	req := agent.CompactionRequest{Provider: agenttest.NewProvider(), Instruction: "summarize", Cut: 0}
	sink := make(chan *agent.Event, 64)
	if err := h.Compact(contextBackground(), req, sink); err == nil {
		t.Fatal("Compact returned err = nil, want a non-nil error (cut resolves to 0)")
	}
	drainSink(t, sink)
	return provider
}

// agoLifecycleCompactionFailedNonStopFinish drives a compaction call
// whose own completion carries a non-Stop finish reason (reusing
// scriptToolCallResponse's own ToolCalls-finishing script —
// drainCompactionCall never dispatches the call, only reads
// FinishReason()).
func agoLifecycleCompactionFailedNonStopFinish(t *testing.T) *tracetest.Provider {
	t.Helper()
	provider := tracetest.NewProvider()
	h, hist, _ := markedHarnessForCompaction(t, "cmp-lifecycle-non-stop")
	h.TracerProvider = provider
	compactionModelProvider := agenttest.NewProvider(scriptToolCallResponse(t, "call-cmp-lifecycle-001", "unused_tool_name", []byte(`{}`)))
	req := agent.CompactionRequest{Provider: compactionModelProvider, Instruction: "summarize", Cut: hist.Len()}
	sink := make(chan *agent.Event, 64)
	if err := h.Compact(contextBackground(), req, sink); err == nil {
		t.Fatal("Compact returned err = nil, want a non-nil error (non-Stop finish reason)")
	}
	drainSink(t, sink)
	return provider
}

// agoLifecycleCompactionFailedMidCall drives a compaction call whose
// provider terminates mid-stream with a typed failure.
func agoLifecycleCompactionFailedMidCall(t *testing.T) *tracetest.Provider {
	t.Helper()
	provider := tracetest.NewProvider()
	h, hist, _ := markedHarnessForCompaction(t, "cmp-lifecycle-mid-call")
	h.TracerProvider = provider
	compactionModelProvider := agenttest.NewProvider(scriptTextThenTerminalFailure(t, ai.FailureCategoryUnavailable, true))
	req := agent.CompactionRequest{Provider: compactionModelProvider, Instruction: "summarize", Cut: hist.Len()}
	sink := make(chan *agent.Event, 64)
	if err := h.Compact(contextBackground(), req, sink); err == nil {
		t.Fatal("Compact returned err = nil, want a non-nil error (mid-call terminal failure)")
	}
	drainSink(t, sink)
	return provider
}

// agoLifecycleNormalCompletion drives a plain successful text-only run.
func agoLifecycleNormalCompletion(t *testing.T) *tracetest.Provider {
	t.Helper()
	provider := tracetest.NewProvider()
	modelProvider := agenttest.NewProvider(scriptTextResponse(t, ai.FinishReasonStop))
	h := agent.Harness{Provider: modelProvider, System: "system prompt for lifecycle normal completion", TracerProvider: provider}

	sink := make(chan *agent.Event, 64)
	if _, _, err := h.Run(contextBackground(), firstMessage(t), sink); err != nil {
		t.Fatalf("Run returned err = %v, want nil", err)
	}
	drainSink(t, sink)
	return provider
}

// agoLifecycleMidStreamFailure drives a run whose provider fails
// mid-stream with the given category — RunOutcomeFailed/
// TurnOutcomeAborted (TestTurn_FatalPath_EmitsTypedBrackets's own
// fixture, scriptTextThenTerminalFailure, reused).
func agoLifecycleMidStreamFailure(t *testing.T, category ai.FailureCategory) *tracetest.Provider {
	t.Helper()
	provider := tracetest.NewProvider()
	modelProvider := agenttest.NewProvider(scriptTextThenTerminalFailure(t, category, true))
	h := agent.Harness{Provider: modelProvider, System: "system prompt for lifecycle mid-stream failure", TracerProvider: provider}

	sink := make(chan *agent.Event, 64)
	if _, _, err := h.Run(contextBackground(), firstMessage(t), sink); err == nil {
		t.Fatal("Run returned err = nil, want a non-nil error (mid-stream terminal failure)")
	}
	drainSink(t, sink)
	return provider
}

// agoLifecycleToolExecutionFailure drives a run whose one tool call
// returns a plain error from Run — ToolOutcomeExecutionFailure,
// non-panic arm.
func agoLifecycleToolExecutionFailure(t *testing.T) *tracetest.Provider {
	t.Helper()
	provider := tracetest.NewProvider()

	toolName := "lifecycle_failing_tool"
	tool := &ScriptedTool{
		toolName: toolName,
		Effect:   agent.EffectClassRead,
		Script: func(context.Context, []byte, agent.PolicySlot) (agent.Result, error) {
			return agent.Result{}, errAGOLifecycleToolFailure
		},
	}
	reg := agent.NewMapRegistry(map[string]agent.Tool{toolName: tool})

	turnOneScript := scriptToolCallResponse(t, "call-lifecycle-failure-001", toolName, []byte(`{}`))
	turnTwoScript := scriptTextResponse(t, ai.FinishReasonStop)
	modelProvider := agenttest.NewProvider(turnOneScript, turnTwoScript)

	h := agent.Harness{
		Provider:       modelProvider,
		System:         "system prompt for lifecycle tool execution failure",
		Turn:           agent.TurnOptions{Tools: reg},
		TracerProvider: provider,
	}
	sink := make(chan *agent.Event, 256)
	// The run's own overall error/continuation shape after one tool's
	// execution failure is not this row's concern (that is R-TLS/R-RUN
	// territory, proven elsewhere) — this row exists to drive a genuine
	// ToolOutcomeExecutionFailure exit through the tool span's own
	// finalizer (Phase 7).
	_, _, _ = h.Run(contextBackground(), firstMessage(t), sink)
	drainSink(t, sink)
	return provider
}

// errAGOLifecycleToolFailure is the plain error agoLifecycleToolExecutionFailure's
// tool returns.
var errAGOLifecycleToolFailure = errAGOLifecycleToolFailureType{}

type errAGOLifecycleToolFailureType struct{}

func (errAGOLifecycleToolFailureType) Error() string {
	return "agent_test: lifecycle fixture's deliberate tool execution failure"
}

// agoLifecycleToolPanicContained drives a run whose one tool call
// panics. The scheduler contains the panic and converts it to a typed
// ExecutionFailure (TestScheduler_PanicContainment's own proof) — Run
// does not itself panic, so no recover() is needed here; this row's own
// purpose is proving the tool span still ends exactly once despite the
// underlying panic unwinding through executeCall's own frame (Phase 7),
// not the run's error shape.
func agoLifecycleToolPanicContained(t *testing.T) *tracetest.Provider {
	t.Helper()
	provider := tracetest.NewProvider()

	toolName := "lifecycle_panicking_tool"
	tool := PanickingScriptedTool(toolName, agent.EffectClassRead, "agent_test: lifecycle fixture's deliberate panic")
	reg := agent.NewMapRegistry(map[string]agent.Tool{toolName: tool})

	turnOneScript := scriptToolCallResponse(t, "call-lifecycle-panic-001", toolName, []byte(`{}`))
	turnTwoScript := scriptTextResponse(t, ai.FinishReasonStop)
	modelProvider := agenttest.NewProvider(turnOneScript, turnTwoScript)

	h := agent.Harness{
		Provider:       modelProvider,
		System:         "system prompt for lifecycle tool panic",
		Turn:           agent.TurnOptions{Tools: reg},
		TracerProvider: provider,
	}
	sink := make(chan *agent.Event, 256)
	_, _, _ = h.Run(contextBackground(), firstMessage(t), sink)
	drainSink(t, sink)
	return provider
}

// agoLifecycleCancellationInterrupted drives a run whose one tool call
// observes ctx.Done() (CancellationObservingScriptedTool) and Interrupt
// fires while it is in flight — RunOutcomeInterrupted, the
// scheduler-visible cancellation arm distinct from the detached/deaf
// wind-down arm below.
func agoLifecycleCancellationInterrupted(t *testing.T) *tracetest.Provider {
	t.Helper()
	provider := tracetest.NewProvider()

	release := make(chan struct{})
	t.Cleanup(func() { close(release) })

	toolName := "lifecycle_cancel_observing_tool"
	tool, started, _ := CancellationObservingScriptedTool(toolName, agent.EffectClassRead, release)
	reg := agent.NewMapRegistry(map[string]agent.Tool{toolName: tool})

	modelProvider := agenttest.NewProvider(scriptToolCallResponse(t, "call-lifecycle-cancel-001", toolName, []byte(`{}`)))
	h := agent.Harness{
		Provider:       modelProvider,
		System:         "system prompt for lifecycle cancellation",
		Turn:           agent.TurnOptions{Tools: reg},
		TracerProvider: provider,
	}

	sink := make(chan *agent.Event, 256)
	resultCh := make(chan error, 1)
	go func() {
		_, _, err := h.Run(contextBackground(), firstMessage(t), sink)
		resultCh <- err
	}()

	<-started
	h.Interrupt()
	drainSink(t, sink)
	<-resultCh
	return provider
}

// agoLifecyclePreRequestHookPanic drives a run whose PreRequestHook
// panics — the deliberately-UNRECOVERED mutating-hook posture
// agent-v1-scope's R-AGS-016 records by name ("Unrecovered mutating
// hooks... Untouched and unexercised" — inherited from AG-20, honoured
// through AG-21). Neither Turn nor Harness.Run installs a recover()
// anywhere in this call chain, so the panic propagates all the way up;
// this driver installs the ONLY recover, at its own top, mirroring
// agoParityRun's own recover-and-report shape, so the test binary
// survives and the caller can inspect what was recorded before the
// unwind reached here.
//
// Driven through the BARE Turn entry point (nil continuation), not
// Harness.Run: a panic reaching Turn from INSIDE Harness.Run's own
// attempt loop (harness.go) never reaches that loop's own
// `<-forwarderDone` join, abandoning the per-attempt forwarder
// goroutine — an independent, pre-existing race between that goroutine's
// still-in-flight `sink <- ev` and Run's own deferred close(sink), never
// caused by this milestone's span-lifetime fix and out of its scope.
// Turn's own nil-continuation path opens the identical run+turn span
// pair Harness.Run does (loop.go's own D-D comments), so this drives the
// exact same R-AGO-007 claim through a race-free entry point: a manually
// started root span seeds ctx with this test's own tracetest.Provider,
// since Turn acquires its tracer AMBIENTLY from ctx, never by injection
// (R-AGO-001).
func agoLifecyclePreRequestHookPanic(t *testing.T) (provider *tracetest.Provider, recovered any) {
	t.Helper()
	provider = tracetest.NewProvider()
	modelProvider := agenttest.NewProvider(scriptTextResponse(t, ai.FinishReasonStop))

	rootTracer := provider.Tracer("agent_test/lifecycle_pre_request_hook_panic")
	ctx, rootSpan := rootTracer.Start(contextBackground(), "test_root")
	defer rootSpan.End()

	sink := make(chan *agent.Event, 64)
	opts := agent.TurnOptions{
		PreRequestHook: func(context.Context, ai.Request) (ai.Request, error) {
			panic("agent_test: lifecycle fixture's deliberate hook panic")
		},
	}
	func() {
		defer func() {
			recovered = recover()
		}()
		_, _, _ = agent.Turn(ctx, modelProvider, "system prompt for lifecycle pre-request hook panic", []ai.Message{firstMessage(t)}, opts, sink)
	}()
	return provider, recovered
}

// TestObservability_Lifecycle_NoSpanForPreStartRefusals is S-AGO-054
// (AG-22 correction, MAJOR-6, sdd-verify round 1): "a run refused before
// it emits any event, a turn refused before its start event, a tool
// call gated out by the permission gate before its start event exists,
// and a compaction refused before its started event... no span was
// started for that bracket." Each subtest asserts provider.Started()
// stays exactly what it was before the refused bracket, or 0 where
// nothing else in the scenario opened a span first.
func TestObservability_Lifecycle_NoSpanForPreStartRefusals(t *testing.T) {
	t.Run("run_refused_before_any_event", func(t *testing.T) {
		t.Parallel()
		// S-HKS-002: a non-zero Turn.Hooks is refused before any
		// identity is minted and before any event is emitted —
		// hooks_test.go's own TestHooks_TransportRefusesRatherThanOverwrites
		// establishes "no run identity was minted" for this exact path.
		provider := tracetest.NewProvider()
		modelProvider := agenttest.NewProvider(scriptTextResponse(t, ai.FinishReasonStop))
		h := agent.Harness{
			Provider:       modelProvider,
			System:         "system prompt for S-AGO-054 run refusal",
			Turn:           agent.TurnOptions{Hooks: agent.Hooks{SessionStart: []agent.SessionStartObserver{noopSessionStartObserver}}},
			TracerProvider: provider,
		}
		sink := make(chan *agent.Event, 16)
		if _, _, err := h.Run(contextBackground(), firstMessage(t), sink); err == nil {
			t.Fatal("Run returned err = nil, want the typed misplaced-hooks refusal")
		}
		drainSink(t, sink)
		if got := provider.Started(); got != 0 {
			t.Errorf("provider.Started() = %d, want 0 — the run was refused before run_start, so no run span may open (S-AGO-054)", got)
		}
	})

	t.Run("turn_refused_before_its_start_event", func(t *testing.T) {
		t.Parallel()
		// validateContinuation is Turn's own FIRST statement (S-LSK-014):
		// a half-configured Continuation is rejected before identity
		// minting and before turn_start.
		provider := tracetest.NewProvider()
		modelProvider := agenttest.NewProvider(scriptTextResponse(t, ai.FinishReasonStop))
		opts := agent.TurnOptions{
			Continuation: &agent.TurnContinuation{Run: "run-s-ago-054-turn-refusal"}, // Stamper/Scheduler/History all nil — half-configured.
		}
		// S-LSK-014's own contract: this refusal touches sink not at
		// all — "no closeSink, no partial stream" — so drainSink (which
		// blocks for a close) does not apply here; the channel simply
		// carries nothing to read.
		sink := make(chan *agent.Event, 16)
		if _, _, err := agent.Turn(contextBackground(), modelProvider, "system prompt for S-AGO-054 turn refusal", []ai.Message{firstMessage(t)}, opts, sink); err == nil {
			t.Fatal("Turn returned err = nil, want the typed half-configured-continuation refusal")
		}
		if got := provider.Started(); got != 0 {
			t.Errorf("provider.Started() = %d, want 0 — the turn was refused before turn_start, so no span may open (S-AGO-054)", got)
		}
	})

	t.Run("tool_call_gated_out_before_its_start_event", func(t *testing.T) {
		t.Parallel()
		// R-AGO-005 item 3 / R-AGO-007: "no span opens for a gated-out
		// call either — the permission gate runs entirely before this
		// point." A Deny verdict never reaches ToolStart or Run.
		provider := tracetest.NewProvider()
		toolName := "s_ago_054_denied_tool"
		tool := EchoScriptedTool(toolName, agent.EffectClassRead)
		reg := agent.NewMapRegistry(map[string]agent.Tool{toolName: tool})

		innerFailure, ferr := ai.MidStreamFailure(ai.FailureReport{Category: ai.FailureCategoryUnavailable}, false)
		if ferr != nil {
			t.Fatalf("ai.MidStreamFailure: %v", ferr)
		}
		denial, derr := agent.NewFailure(innerFailure)
		if derr != nil {
			t.Fatalf("agent.NewFailure: %v", derr)
		}
		policy := &wiringTestPolicy{resolve: func(context.Context, ai.ToolCall) agent.PermissionVerdict {
			return agent.PermissionVerdict{Outcome: agent.PermissionOutcomeDeny, Failure: denial}
		}}

		modelProvider := agenttest.NewProvider(
			scriptToolCallResponse(t, "call-s-ago-054-001", toolName, []byte(`{}`)),
			scriptTextResponse(t, ai.FinishReasonStop),
		)
		h := agent.Harness{
			Provider:       modelProvider,
			System:         "system prompt for S-AGO-054 gated-out tool",
			Turn:           agent.TurnOptions{Tools: reg, PermissionPolicy: policy},
			TracerProvider: provider,
		}
		sink := make(chan *agent.Event, 256)
		if _, _, err := h.Run(contextBackground(), firstMessage(t), sink); err != nil {
			t.Fatalf("Run returned err = %v, want nil", err)
		}
		drainSink(t, sink)

		if got := provider.Started(); got == 0 {
			t.Fatal("provider.Started() = 0, want > 0 — the surrounding run/turn spans must still have opened")
		}
		if toolSpan := spanWithNamePrefix(provider.Spans(), toolSpanNamePrefixForTest); toolSpan != nil {
			t.Errorf("a span named %q was recorded for a gated-out call, want none (S-AGO-054)", toolSpan.Name())
		}
	})

	t.Run("compaction_refused_before_its_started_event", func(t *testing.T) {
		t.Parallel()
		// Harness.Compact's own two refusals (shutdown, run-in-flight)
		// fire before runCompaction is ever called — before Compact's
		// own run span opens, let alone compaction_started.
		provider := tracetest.NewProvider()
		h := agent.Harness{
			Provider:       agenttest.NewProvider(),
			System:         "system prompt for S-AGO-054 compaction refusal",
			History:        agent.NewHistory(),
			TracerProvider: provider,
		}
		h.Shutdown()
		req := agent.CompactionRequest{Provider: agenttest.NewProvider(), Instruction: "summarize"}
		// Compact's own shutdown/run-in-flight refusals return BEFORE
		// its `defer close(sink)` is ever registered — sink is touched
		// not at all, so drainSink (which blocks for a close) does not
		// apply here.
		sink := make(chan *agent.Event, 16)
		if err := h.Compact(contextBackground(), req, sink); err == nil {
			t.Fatal("Compact returned err = nil, want ErrPromptAfterShutdown")
		}
		if got := provider.Started(); got != 0 {
			t.Errorf("provider.Started() = %d, want 0 — Compact refused before its own run span could open, let alone compaction_started (S-AGO-054)", got)
		}
	})
}

// TestObservability_Lifecycle_PreRequestHookPanicEndsEverySpanExactlyOnce
// is R-AGO-007's own panic-unwind clause ("a panic unwinding through the
// frame" is named explicitly among the terminal paths a finalizer
// registered at open must cover), driven through a REAL, reachable
// panic rather than asserted by inspection: the deliberately-unrecovered
// PreRequestHook seam (R-AGS-016). Before the fix this fails naming
// Started=2 Ended=0 (the run and turn spans both leak) — recorded below,
// then closed by registering each span's finalizer as a `defer` at open
// (matching the tool span's own already-correct pattern) instead of at
// each individual return site.
func TestObservability_Lifecycle_PreRequestHookPanicEndsEverySpanExactlyOnce(t *testing.T) {
	provider, recovered := agoLifecyclePreRequestHookPanic(t)
	if recovered == nil {
		t.Fatal("PreRequestHook panic did not propagate through Harness.Run — want a panic reaching this test's own recover, per R-AGS-016's unrecovered-mutating-hook posture")
	}
	t.Logf("panic recovered by caller: %v", recovered)
	t.Logf("Started=%d Ended=%d", provider.Started(), provider.Ended())
	if err := provider.AssertAllEndedOnce(); err != nil {
		t.Errorf("AssertAllEndedOnce() = %v, want nil — a panic unwinding through Turn/Harness.Run MUST still end every span exactly once (R-AGO-007)", err)
	}
}

// agoLifecycleDetachedCallID is the detached fixture's own fixed call
// identity — named so TestObservability_Lifecycle_DetachedArm_
// DecidedAttributes (S-AGO-055) can correlate the recorded tool span
// back to this exact call without duplicating the literal.
const agoLifecycleDetachedCallID = "call-lifecycle-detached-001"

// agoLifecycleDetachedWindDown mirrors
// TestHarness_WindDown_DeafToolCannotHoldRunHostage exactly (a
// cancellation-deaf BlockingScriptedTool, a small injected
// WindDownBound), with TracerProvider wired in. It releases and JOINS
// the detached goroutine itself before returning — R-AGO-007's own rule
// (task 7.4): the goroutine outlives Run's own return by design (D-D),
// and the caller's AssertAllEndedOnce() must run only after it is
// genuinely accounted for, never merely excluded via t.Cleanup's late
// timing.
func agoLifecycleDetachedWindDown(t *testing.T) *tracetest.Provider {
	t.Helper()
	provider := tracetest.NewProvider()

	release := make(chan struct{})
	finished := make(chan struct{})
	toolName := "lifecycle_detached_tool"
	tool := &ScriptedTool{
		toolName: toolName,
		Effect:   agent.EffectClassRead,
		Script: func(context.Context, []byte, agent.PolicySlot) (agent.Result, error) {
			<-release
			close(finished)
			return agent.Result{Outcome: agent.ToolOutcomeSuccess, Content: []byte("released")}, nil
		},
	}
	reg := agent.NewMapRegistry(map[string]agent.Tool{toolName: tool})

	callID := agoLifecycleDetachedCallID
	modelProvider := agenttest.NewProvider(scriptToolCallResponse(t, callID, toolName, []byte(`{}`)))
	sched := &agent.Scheduler{WindDownBound: smallWindDownBound}
	h := agent.Harness{
		Provider:       modelProvider,
		System:         "system prompt for lifecycle detached wind-down",
		Turn:           agent.TurnOptions{Tools: reg},
		Scheduler:      sched,
		TracerProvider: provider,
	}

	sink := make(chan *agent.Event, 256)
	resultCh := make(chan error, 1)
	go func() {
		_, _, err := h.Run(contextBackground(), firstMessage(t), sink)
		resultCh <- err
	}()

	readUntilToolStart(t, sink, callID)
	h.Interrupt()
	drainSink(t, sink)
	<-resultCh

	close(release)
	<-finished // join the detached goroutine BEFORE this driver returns.
	return provider
}

// TestObservability_Lifecycle_DetachedArm_DecidedAttributes is S-AGO-055
// (AG-22 correction, MAJOR-5, sdd-verify round 1): the detached tool
// span's own DECIDED attributes — the one thing this milestone chose to
// decide explicitly (D-D) — were previously left unguarded; the
// `detached_wind_down` table row above asserted only
// AssertAllEndedOnce()/Started()/Started()==Ended(), never
// cachicamas.tool.detached, error.type or the status those decided
// values sit beside.
func TestObservability_Lifecycle_DetachedArm_DecidedAttributes(t *testing.T) {
	provider := agoLifecycleDetachedWindDown(t)

	var toolSpan *tracetest.Span
	for _, span := range provider.Spans() {
		if callID, ok := attrString(span.Attributes(), "gen_ai.tool.call.id"); ok && callID == agoLifecycleDetachedCallID {
			toolSpan = span
			break
		}
	}
	if toolSpan == nil {
		t.Fatalf("no recorded span carries gen_ai.tool.call.id = %q", agoLifecycleDetachedCallID)
	}

	if got, ok := attrBool(toolSpan.Attributes(), "cachicamas.tool.detached"); !ok || !got {
		t.Errorf("detached tool span cachicamas.tool.detached = (%v, present=%v), want (true, true) (S-AGO-055)", got, ok)
	}
	wantCategory := ai.FailureCategoryCancellation.String()
	if got, ok := attrString(toolSpan.Attributes(), "error.type"); !ok || got != wantCategory {
		t.Errorf("detached tool span error.type = (%q, present=%v), want (%q, true) — the detached arm's own typed cancellation category (S-AGO-055)", got, ok, wantCategory)
	}
	agoAssertStatus(t, "tool", agoLifecycleDetachedCallID, toolSpan, true, wantCategory)
}

// TestObservability_Lifecycle_DetachedArm_GoroutineHoldsNoSpanHandle is
// S-AGO-056 (AG-22 correction, MAJOR-5): "the detached goroutine's own
// code is read, then it holds no span handle and calls no span method."
// scheduler.go's own comment already argues this (runToolWithWindDown's
// inner goroutine receives only ctx, tool, args and policy — no span
// parameter — so it is LEXICALLY unable to reference the tool span at
// all, being a separate function call, not a closure over executeCall's
// locals) but the claim was never checked against the shipped source,
// only asserted in prose. This scans runToolWithWindDown's own function
// SIGNATURE (go/parser, mirroring observability_scope_test.go's
// agoDeclMentionsOtel/agoOtelImportAliases convention) for any
// OpenTelemetry-imported identifier — if a future change ever added a
// span parameter to this function, the inner goroutine WOULD gain the
// ability to reference one, and this guard would catch that at the
// exact point it becomes possible, not by inspecting the goroutine body
// for evidence it happens not to use it.
func TestObservability_Lifecycle_DetachedArm_GoroutineHoldsNoSpanHandle(t *testing.T) {
	root, err := gitTopLevel(t)
	if err != nil {
		t.Fatalf("git rev-parse --show-toplevel failed: %v", err)
	}
	path := root + "/backend/agent/src/agent/scheduler.go"
	if _, statErr := os.Stat(path); statErr != nil {
		t.Fatalf("os.Stat(%q) failed: %v", path, statErr)
	}

	fset := token.NewFileSet()
	file, ferr := parser.ParseFile(fset, path, nil, 0)
	if ferr != nil {
		t.Fatalf("parser.ParseFile(%q) error = %v, want nil", path, ferr)
	}
	aliases := agoOtelImportAliases(file)

	found := false
	for _, decl := range file.Decls {
		fn, ok := decl.(*ast.FuncDecl)
		if !ok || fn.Recv == nil || fn.Name.Name != "runToolWithWindDown" {
			continue
		}
		found = true
		if agoDeclMentionsOtel(fn.Type, aliases) {
			t.Errorf("runToolWithWindDown's own signature mentions an OpenTelemetry-imported identifier — S-AGO-056 requires the detached goroutine hold no span handle; this function's signature is the only channel through which its own inner goroutine could ever receive one (it is a separate function call, not a closure over executeCall's locals)")
		}
	}
	if !found {
		t.Fatal("runToolWithWindDown method not found in scheduler.go — the S-AGO-056 structural claim has nothing to check")
	}
}
