// AG-21 — Phase 4: cross-run state (R-CNH-007; discharges
// agent-run-driver's R-RUN-001, agent-cancellation-tree's R-CAN-002/
// R-CAN-005 and non-requirement rows :202/:205, agent-turn-termination's
// multi-turn-state closure, and agent-loop-skeleton's :263 closure —
// S-CNH-014..016; proposal D2).
//
// The claim is NOT "no state carries over" — that would forbid a
// continuing conversation. It is: the only state that outlives a run
// is state the caller explicitly owns or a shipped requirement already
// enumerates, and the harness itself retains nothing.
package agent_test

import (
	"context"
	"errors"
	"strconv"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/cachicamas/backend/agent/src/agent"
	"github.com/cachicamas/backend/agent/src/agenttest"
	"github.com/cachicamas/backend/agent/src/ai"
)

// cnhNonceCounter mints a fresh, per-process-unique nonce per test
// execution — never a shape or substring another fixture could also
// produce (this repo's own ghost-assertion lesson).
var cnhNonceCounter atomic.Uint64

func cnhMintNonce() string {
	return "cnh-nonce-" + strconv.FormatUint(cnhNonceCounter.Add(1), 10)
}

// cnhCountCallIDInRequests counts how many times callID appears as a
// tool-call or tool-result identity anywhere across requests — the
// absence/presence measure both cross-run scenarios read.
func cnhCountCallIDInRequests(requests []ai.Request, callID string) int {
	count := 0
	for _, req := range requests {
		for _, msg := range req.Messages() {
			for _, part := range msg.Content() {
				if tc, ok := part.ToolCall(); ok && tc.ID() == callID {
					count++
				}
				if tr, ok := part.ToolResult(); ok && tr.CallID() == callID {
					count++
				}
			}
		}
	}
	return count
}

// ---------------------------------------------------------------------
// The absence half — nil History (S-CNH-014).
// ---------------------------------------------------------------------

// TestCrossRunState_NilHistory_AbsenceWithAntiGhostFloor — S-CNH-014.
// Given a harness value with no caller-supplied transcript, whose
// first run is interrupted mid-turn over a call genuinely in flight
// (dispatched to a cancellation-deaf tool, still unresolved at the
// moment the signal fires) and seeded with a uniquely minted nonce as
// that call's own identity, when a second run is driven on that same
// value, then the minted nonce appears nowhere in the second run's
// captured provider requests; and, as the anti-ghost floor, the same
// nonce IS found in the first run's own stream, proving the needle is
// findable at all.
//
// The defeat test (bite d) is recorded in apply-progress.md rather
// than embedded here as a second code path: re-running this identical
// absence assertion with h.History deliberately set to a real,
// caller-shared *agent.History (instead of nil) makes the assertion
// go RED, because the nonce then legitimately reaches run 2's request
// — proving this test is not vacuously green regardless of sharing.
func TestCrossRunState_NilHistory_AbsenceWithAntiGhostFloor(t *testing.T) {
	t.Parallel()
	cnhDriveCrossRunNilHistory(t)
}

// cnhDriveCrossRunNilHistory is TestCrossRunState_NilHistory_
// AbsenceWithAntiGhostFloor's own callable body — reused as one of the
// leak sweep's own serial drivers (Phase 5, R-CNH-005). The deaf
// tool's own release channel is closed explicitly, inline, once run 1
// has returned — never left to t.Cleanup, which would pile up unfired
// across the sweep's 50 repeats.
func cnhDriveCrossRunNilHistory(t *testing.T) {
	t.Helper()

	nonce := cnhMintNonce()
	callID := "call-" + nonce
	toolName := "cnh_crossrun_tool_nil"

	// release carries NO t.Cleanup fallback, unlike this file's own
	// gates elsewhere -- recorded here rather than left implicit
	// (sdd-verify round-2, MINOR-4). The tradeoff is deliberate and
	// correct for the sweep (Phase 5): a t.Cleanup registered per
	// iteration would not fire until the whole 50-repeat sweep test
	// ends, misreporting every iteration's own detached goroutine as
	// leaked at snapshot time -- the same reasoning already documented
	// for the pressure-2 driver (slow_consumer_pressure_test.go). The
	// accepted risk is narrow: only a t.Fatal inside readUntilToolStart
	// below, BEFORE close(release) at run 1's own return, would leave
	// this one goroutine parked -- a test-authoring bug, not a
	// production leak, and never reached on the shipped path.
	release := make(chan struct{})
	tool := BlockingScriptedTool(toolName, agent.EffectClassRead, release)
	reg := agent.NewMapRegistry(map[string]agent.Tool{toolName: tool})

	run1Provider := agenttest.NewProvider(scriptToolCallResponse(t, callID, toolName, []byte(`{}`)))
	h := &agent.Harness{
		Provider:  run1Provider,
		System:    "system prompt for cnh-crossrun-nil",
		Turn:      agent.TurnOptions{Tools: reg},
		Scheduler: &agent.Scheduler{WindDownBound: smallWindDownBound},
		// History is deliberately left nil: no caller-supplied transcript.
	}

	sink1 := make(chan *agent.Event, 256)
	resultCh1 := make(chan cnhRunOutcome, 1)
	go func() {
		msg, finish, err := h.Run(contextBackground(), firstMessage(t), sink1)
		resultCh1 <- cnhRunOutcome{msg, finish, err}
	}()

	events1 := readUntilToolStart(t, sink1, callID)
	h.Interrupt()
	events1 = append(events1, drainSink(t, sink1)...)
	got1 := <-resultCh1
	// AFTER run 1 returns: the detached goroutine is accounted for,
	// not merely excluded from the leak count (R-CNH-006).
	close(release)

	if !errors.Is(got1.err, agent.ErrInterrupted) {
		t.Fatalf("run 1 error = %v, want errors.Is(_, agent.ErrInterrupted)", got1.err)
	}

	// Anti-ghost floor: the minted call identity IS found in run 1's
	// own stream, before its absence from run 2 is believed.
	found := false
	for _, ev := range events1 {
		if start, ok := ev.ToolStart(); ok && start.CallID() == callID {
			found = true
		}
	}
	if !found {
		t.Fatal("anti-ghost floor: run 1's own tool_start for the minted call identity was not observed — the needle is not findable at all")
	}

	// Run 2, on the SAME harness value, nil History still.
	run2Provider := agenttest.NewProvider(scriptTextResponse(t, ai.FinishReasonStop))
	h.Provider = run2Provider

	sink2 := make(chan *agent.Event, 256)
	_, _, err2 := h.Run(contextBackground(), firstMessage(t), sink2)
	if err2 != nil {
		t.Fatalf("run 2 returned err = %v, want nil", err2)
	}
	drainSink(t, sink2)

	if got := cnhCountCallIDInRequests(run2Provider.Requests(), callID); got != 0 {
		t.Errorf("run 2's captured request(s) reference run 1's minted call identity %d time(s), want 0 — nil History: nothing carries over", got)
	}
}

// ---------------------------------------------------------------------
// The legitimate-carry half and the inventory (S-CNH-015).
// ---------------------------------------------------------------------

// TestCrossRunState_SharedHistory_LegitimateCarryAndInventory —
// S-CNH-015. Given a harness value whose transcript the caller DID
// supply, seeded with a genuinely PRE-EXISTING open call (never
// touched by run 1's own turn machinery at all, so it stays open until
// windDownRun's own SynthesizeOrphans repairs it — the same shape
// TestHarness_Interrupt_MidTurnEndsInterruptedWithOrphansSynthesized's
// "provider cancelled mid-stream" sub-test already proves;
// cancellation_interrupt_test.go's own comment on its OTHER sub-test
// records why an in-flight, self-resolving call is NOT this shape: "the
// turn's own call reached a genuine result ... distinct from orphan
// synthesis, which repairs only a call Schedule never returned a
// result for"), when the adversarial first run has wound down and a
// second run is driven, then the caller's transcript contains the
// synthesized orphan result of R-HIS-007 and the second run's captured
// provider request DOES contain it — the legitimate carry; and on the
// same value: a session-start observing hook fires exactly once across
// both runs; an interrupt issued BETWEEN the two runs is a no-op and
// the second run still reaches its own terminal (cancelRun cleared); a
// Steer issued after the first run returned is rejected typed and a
// Steer during the second run is accepted (the queue closed then
// reopened); and the shutdown flag stays unlatched on the interrupt
// path — proven implicitly by the second run's own acceptance, since a
// latched flag would refuse it typed and emit nothing.
func TestCrossRunState_SharedHistory_LegitimateCarryAndInventory(t *testing.T) {
	t.Parallel()
	cnhDriveCrossRunSharedHistory(t)
}

// cnhDriveCrossRunSharedHistory is TestCrossRunState_SharedHistory_
// LegitimateCarryAndInventory's own callable body — reused as one of
// the leak sweep's own serial drivers (Phase 5, R-CNH-005).
func cnhDriveCrossRunSharedHistory(t *testing.T) {
	t.Helper()

	nonce := cnhMintNonce()
	callID := "call-" + nonce
	toolName := "cnh_crossrun_tool_shared"

	seedPart, err := ai.NewToolCall(callID, toolName, []byte(`{}`))
	if err != nil {
		t.Fatalf("ai.NewToolCall: %v", err)
	}
	seedMsg, err := ai.NewMessage(ai.RoleAssistant, seedPart)
	if err != nil {
		t.Fatalf("ai.NewMessage: %v", err)
	}
	hist, err := agent.NewSeededHistory([]ai.Message{seedMsg})
	if err != nil {
		t.Fatalf("agent.NewSeededHistory: %v", err)
	}

	var mu sync.Mutex
	var sessionReports []agent.SessionStartReport
	var wg sync.WaitGroup
	wg.Add(1)
	hooks := agent.Hooks{
		SessionStart: []agent.SessionStartObserver{func(_ context.Context, r agent.SessionStartReport) {
			mu.Lock()
			sessionReports = append(sessionReports, r)
			mu.Unlock()
			wg.Done()
		}},
	}

	// Run 1: the seeded call is NEVER touched by this turn — it stays
	// genuinely open until windDownRun's own SynthesizeOrphans repairs
	// it. A held, otherwise-ordinary turn is interrupted mid-stream —
	// the "provider cancelled mid-stream" shape
	// (cancellation_interrupt_test.go:49-84).
	gate1 := agenttest.NewGate()
	t.Cleanup(gate1.Release)
	run1Provider := agenttest.NewProvider(heldTurnScript(t, gate1))
	h := &agent.Harness{Provider: run1Provider, System: "system prompt for cnh-crossrun-shared", History: hist, Hooks: hooks}

	sink1 := make(chan *agent.Event, 256)
	resultCh1 := make(chan cnhRunOutcome, 1)
	go func() {
		msg, finish, err := h.Run(contextBackground(), firstMessage(t), sink1)
		resultCh1 <- cnhRunOutcome{msg, finish, err}
	}()

	<-gate1.Reached()
	h.Interrupt()
	gate1.Release() // explicit, inline safety-net release (NFR-CNH-003); ctx.Done() already unblocks the Hold, this is idempotent.
	events1 := drainSink(t, sink1)
	got1 := <-resultCh1

	if !errors.Is(got1.err, agent.ErrInterrupted) {
		t.Fatalf("run 1 error = %v, want errors.Is(_, agent.ErrInterrupted)", got1.err)
	}
	if report := agent.CheckStream(events1); report.Violation() != nil {
		t.Errorf("CheckStream(run 1 events) = %v, want nil", report.Violation())
	}
	wg.Wait() // the session-start observer's own single fire, run 1.

	// The caller's transcript contains the synthesized orphan result
	// (R-HIS-007), matching the seeded call's own identity.
	synthesizedFound := false
	for _, e := range hist.Entries() {
		if e.Origin() != agent.EntryOriginSynthesized {
			continue
		}
		content := e.Message().Content()
		if len(content) == 0 {
			continue
		}
		if tr, ok := content[0].ToolResult(); ok && tr.CallID() == callID {
			synthesizedFound = true
		}
	}
	if !synthesizedFound {
		t.Fatal("hist.Entries() carries no EntryOriginSynthesized tool-result entry for the seeded open call — R-HIS-007's orphan synthesis did not run")
	}
	if err := hist.CloseTurn(); err != nil {
		t.Errorf("hist.CloseTurn() after wind-down = %v, want nil (no open call remains)", err)
	}

	// Between runs: an interrupt is a documented no-op (R-CAN-004) —
	// no run in flight, cancelRun is nil.
	h.Interrupt()

	// A Steer issued after run 1 returned is rejected typed.
	rejected, err := ai.NewMessage(ai.RoleUser, mustText(t, "cnh steer after run one returned"))
	if err != nil {
		t.Fatalf("ai.NewMessage: %v", err)
	}
	if err := h.Steer(rejected); err == nil {
		t.Fatal("Steer after run 1 returned = nil, want the typed post-terminal rejection")
	}

	// Run 2, on the SAME harness value, the SAME caller-supplied hist.
	// Held again, so the Steer below is proven to race no window: by
	// the time gate2.Reached() closes, Run's own entry-time queue
	// reopen has unconditionally already happened.
	gate2 := agenttest.NewGate()
	t.Cleanup(gate2.Release)
	run2Provider := agenttest.NewProvider(heldTurnScript(t, gate2), scriptTextResponse(t, ai.FinishReasonStop))
	h.Provider = run2Provider

	sink2 := make(chan *agent.Event, 256)
	resultCh2 := make(chan cnhRunOutcome, 1)
	go func() {
		msg, finish, err := h.Run(contextBackground(), firstMessage(t), sink2)
		resultCh2 <- cnhRunOutcome{msg, finish, err}
	}()

	<-gate2.Reached()

	// A Steer during run 2 is accepted — the queue reopened at Run
	// entry (R-RUN-001), not left closed from run 1's own terminal.
	steered, err := ai.NewMessage(ai.RoleUser, mustText(t, "cnh steer during run two"))
	if err != nil {
		t.Fatalf("ai.NewMessage: %v", err)
	}
	if err := h.Steer(steered); err != nil {
		t.Errorf("Steer during run 2 = %v, want nil — the queue must have reopened at Run entry", err)
	}
	gate2.Release()

	events2 := drainSink(t, sink2)
	got2 := <-resultCh2
	// Run 2's own acceptance (a nil error, a real emitted stream) is
	// itself the proof the shutdown flag stayed unlatched on the
	// interrupt path: a latched flag would have refused it typed
	// (ErrPromptAfterShutdown) and emitted no event at all.
	if got2.err != nil {
		t.Fatalf("run 2 returned err = %v, want nil (shutdown must stay unlatched on the interrupt path)", got2.err)
	}
	if report := agent.CheckStream(events2); report.Violation() != nil {
		t.Errorf("CheckStream(run 2 events) = %v, want nil", report.Violation())
	}

	// The legitimate carry: run 2's own captured request DOES contain
	// the synthesized orphan result (the caller's continuing
	// conversation).
	if got := cnhCountCallIDInRequests(run2Provider.Requests(), callID); got == 0 {
		t.Error("run 2's captured request(s) contain zero references to the seeded call's identity, want at least 1 — the caller-shared transcript must carry the synthesized orphan into run 2's request")
	}

	// The session-start latch: still exactly one report, across BOTH
	// runs, naming run 1's own identity.
	mu.Lock()
	reportCount := len(sessionReports)
	var firstRunID agent.RunID
	if reportCount > 0 {
		firstRunID = sessionReports[0].Run()
	}
	mu.Unlock()
	if reportCount != 1 {
		t.Errorf("session-start observer fired %d time(s) across two runs, want exactly 1", reportCount)
	}
	if len(events1) == 0 || events1[0].Run() != firstRunID {
		t.Errorf("session-start report names run %v, want run 1's own identity %v", firstRunID, events1[0].Run())
	}
}
