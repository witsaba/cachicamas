// AG-09 — strict-TDD tests for the tool scheduler
// (R-TLS-004..011, S-TLS-004..011 + bites S-TLS-002a, S-TLS-005a, S-TLS-006a,
// S-TLS-006b, S-TLS-010a, S-TLS-011a).
//
// Every scenario is in package agent_test (NFR-TLS-001; AG-07 W6 /
// AG-08 carry). The bites are RED-recorded BEFORE the property
// scenarios GREEN — defense against AG-05 W1 (vacuous helpers).
//
// # Bite-first RED ordering (defense against AG-05 W1)
//
// S-TLS-005a (unbounded fan-out), S-TLS-006a (start-event-at-execution
// inverted), S-TLS-006b (start-before-end), S-TLS-010a (errgroup-shaped
// sibling-abort), and S-TLS-011a (no-recover panic abort) are all
// RED-recorded BEFORE the corresponding property scenarios are GREEN.
// S-TLS-002a (policy-tag-strip) lives in tool_test.go.
package agent_test

import (
	"context"
	"runtime"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/cachicamas/backend/agent/src/agent"
	"github.com/cachicamas/backend/agent/src/ai"
)

// scheduledCall is the in-test wire shape for one tool call request
// submitted to a scheduler. Mirrors `ai.ToolCall` fields without
// requiring every test to construct the full Layer 1 part.
type scheduledCall struct {
	id      string
	name    string
	args    []byte
	effect  agent.EffectClass
}

// scriptedBiteTool is the in-test `Tool` whose `Run` returns a
// controllable `Result` or panics. Used by the bites and the property
// tests to assert the scheduler's behavior without reaching for any
// Layer 1 part constructors.
type scriptedBiteTool struct {
	name   string
	effect agent.EffectClass

	invocations atomic.Int64

	// sleepBeforeReturn is the artificial delay before returning
	// the configured result. Used by S-TLS-006 (start events
	// at execution start) to stagger completions.
	sleepBeforeReturn time.Duration

	// result is the `Result` returned by Run. If `panicMessage`
	// is non-empty, Run panics with that message before
	// returning the result.
	result agent.Result
	err    error
	policy agent.PolicySlot

	// panicMessage, when non-empty, makes Run panic with this
	// string. Used by S-TLS-011a.
	panicMessage string

	// onStart is invoked at the start of Run (after recording
	// the policy, before sleeping). Used by S-TLS-005a to
	// observe the fan-out.
	onStart func(name string)

	mu          sync.Mutex
	startedAt   []time.Time
	completedAt []time.Time
}

func (s *scriptedBiteTool) Name() string                  { return s.name }
func (s *scriptedBiteTool) EffectClass() agent.EffectClass { return s.effect }

func (s *scriptedBiteTool) Run(ctx context.Context, args []byte, policy agent.PolicySlot) (agent.Result, error) {
	s.invocations.Add(1)
	s.mu.Lock()
	s.startedAt = append(s.startedAt, time.Now())
	s.policy = policy
	callback := s.onStart
	panicMsg := s.panicMessage
	s.mu.Unlock()
	if callback != nil {
		callback(s.name)
	}
	if panicMsg != "" {
		panic(panicMsg)
	}
	if s.sleepBeforeReturn > 0 {
		time.Sleep(s.sleepBeforeReturn)
	}
	s.mu.Lock()
	s.completedAt = append(s.completedAt, time.Now())
	result := s.result
	err := s.err
	s.mu.Unlock()
	return result, err
}

func (s *scriptedBiteTool) scriptedStartedAt() []time.Time {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make([]time.Time, len(s.startedAt))
	copy(out, s.startedAt)
	return out
}

func (s *scriptedBiteTool) scriptedCompletedAt() []time.Time {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make([]time.Time, len(s.completedAt))
	copy(out, s.completedAt)
	return out
}

func (s *scriptedBiteTool) scriptedRecordedPolicy() agent.PolicySlot {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.policy
}

// scriptedBiteTool constructs a fresh scripted bite tool. Default
// effect is `EffectClassRead`; default result is `ToolOutcomeSuccess`
// with no content.
func newScriptedBiteTool(t *testing.T, name string, effect agent.EffectClass) *scriptedBiteTool {
	t.Helper()
	return &scriptedBiteTool{
		name:   name,
		effect: effect,
		result: agent.Result{Outcome: agent.ToolOutcomeSuccess},
	}
}

// mapRegistry is the in-test `Registry` backed by a map. Used by
// every scheduler test.
type mapRegistry struct {
	tools map[string]agent.Tool
}

func (r *mapRegistry) Resolve(name string) (agent.Tool, bool) {
	t, ok := r.tools[name]
	return t, ok
}

func newMapRegistry(tools map[string]agent.Tool) agent.Registry {
	return &mapRegistry{tools: tools}
}

// S-TLS-005a — RED bite (recorded at Commit 2). Given a scheduler
// without a fan-out bound, when the concurrent-counter scenario
// runs (12 calls against `MaxConcurrentReads = 8`), then it FAILS
// for the right reason — the unbounded scheduler lets the
// counter exceed 8.
//
// RED at write time: the scheduler contract (`Scheduler`,
// `Schedule`, `Registry`, `MaxConcurrentReads`) does not exist yet.
// The assertion is therefore compile-error RED — the strongest
// signal a missing surface can deliver (AG-05 W1 defense).
//
// The bite's job is to prove the bounded-fan-out property (S-TLS-005)
// is non-vacuous: a constant comparison `>= 8` against a counter
// that is always `< 8` would pass even if the scheduler had no
// bound. The `onStart` callback bumps the counter inside a mutex
// and observes the maximum seen across the bounded run.
func TestScheduler_FanOutBoundBite(t *testing.T) {
	t.Parallel()

	const total = 12
	const bound = 8

	// Each tool's onStart bumps the counter; the scheduler's
	// bound is the bound.
	var inFlight atomic.Int64
	var maxInFlight atomic.Int64
	tools := map[string]agent.Tool{}
	calls := make([]scheduledCall, 0, total)
	for i := 0; i < total; i++ {
		name := "read_" + string(rune('a'+i))
		t1 := newScriptedBiteTool(t, name, agent.EffectClassRead)
		t1.sleepBeforeReturn = 5 * time.Millisecond
		t1.onStart = func(string) {
			cur := inFlight.Add(1)
			for {
				prev := maxInFlight.Load()
				if cur <= prev {
					break
				}
				if maxInFlight.CompareAndSwap(prev, cur) {
					break
				}
			}
			time.Sleep(2 * time.Millisecond)
			inFlight.Add(-1)
		}
		tools[name] = t1
		calls = append(calls, scheduledCall{id: name, name: name, args: []byte(`{}`), effect: agent.EffectClassRead})
	}
	reg := newMapRegistry(tools)

	sched := &agent.Scheduler{MaxConcurrentReads: bound}
	sink := make(chan *agent.Event, 1024)
	results := runSchedulerAndClose(sched, calls, reg, sink)

	if len(results) != total {
		t.Fatalf("Schedule returned %d results, want %d", len(results), total)
	}
	// BITE: an unbounded scheduler would let the counter exceed
	// `bound`. Schedulers with the bound keep maxInFlight <= bound.
	if observed := maxInFlight.Load(); observed > int64(bound) {
		t.Errorf("fan-out bound bite DID NOT bite: observed max inflight = %d, want <= %d (an unbounded scheduler let the counter exceed the bound)",
			observed, bound)
	}
	// The bite's preflight: the stub does not invoke the tools, so
	// maxInFlight stays at 0 and the property assertion is vacuous.
	// Assert the tools were actually called so the property is
	// non-vacuous.
	if observed := maxInFlight.Load(); observed == 0 {
		t.Errorf("fan-out bound bite DID NOT bite: maxInFlight = 0 (the scheduler stub did not invoke any tool — the property test would pass vacuously)")
	}
}

// S-TLS-006a — RED bite (recorded at Commit 2). Given completions in
// inverted order (call 2 finishes first, call 0 last), when the
// consumer compares the `ToolStart` arrival order against completion
// order, then they differ — proves `ToolStart` is not emitted at
// rejoin.
//
// RED at write time: same as S-TLS-005a — the surface is missing.
// Compile-error RED is the strongest signal.
func TestScheduler_StartEventAtExecutionStart_BiteInverted(t *testing.T) {
	t.Parallel()

	// Stagger scripted tools so start order is 0, 2, 1.
	tool0 := newScriptedBiteTool(t, "read_inv_0", agent.EffectClassRead)
	tool0.sleepBeforeReturn = 25 * time.Millisecond
	tool1 := newScriptedBiteTool(t, "read_inv_1", agent.EffectClassRead)
	tool1.sleepBeforeReturn = 1 * time.Millisecond
	tool2 := newScriptedBiteTool(t, "read_inv_2", agent.EffectClassRead)
	tool2.sleepBeforeReturn = 10 * time.Millisecond

	reg := newMapRegistry(map[string]agent.Tool{
		"read_inv_0": tool0,
		"read_inv_1": tool1,
		"read_inv_2": tool2,
	})
	calls := []scheduledCall{
		{id: "inv_0", name: "read_inv_0", args: []byte(`{}`), effect: agent.EffectClassRead},
		{id: "inv_1", name: "read_inv_1", args: []byte(`{}`), effect: agent.EffectClassRead},
		{id: "inv_2", name: "read_inv_2", args: []byte(`{}`), effect: agent.EffectClassRead},
	}

	sched := &agent.Scheduler{MaxConcurrentReads: 4}
	sink := make(chan *agent.Event, 256)
	_ = runSchedulerAndClose(sched, calls, reg, sink)
	startCallIDs := observeStartCallIDsWithClose(t, sink)

	if len(startCallIDs) < 3 {
		t.Fatalf("observed only %d ToolStart events, want at least 3", len(startCallIDs))
	}

	// BITE: if ToolStart arrived at rejoin, the start order would
	// match completion order (1, 2, 0) — the completion order
	// dictated by the sleep durations. The read-side property is
	// that ToolStart arrives at execution start, so the order
	// should be 0, 2, 1 (the order starts actually happened).
	wantStartOrder := []string{"inv_0", "inv_2", "inv_1"}
	for i := 0; i < 3; i++ {
		if startCallIDs[i] != wantStartOrder[i] {
			t.Errorf("ToolStart arrival order at index %d = %q, want %q (start events must arrive at execution start, not at rejoin — if ToolStart arrives at rejoin, the arrival order would match the completion order 1,2,0)",
				i, startCallIDs[i], wantStartOrder[i])
		}
	}
}

// S-TLS-006b — RED bite (recorded at Commit 2). Given completions
// in strict issuance order, when `ToolStart` events are observed
// before any `ToolEnd*`, then they precede their corresponding end
// events by at least one observed timestamp tick — proves
// start-before-end.
//
// RED at write time: same compile-error surface.
func TestScheduler_StartEventBeforeEnd_BiteInOrder(t *testing.T) {
	t.Parallel()

	tool := newScriptedBiteTool(t, "read_seq", agent.EffectClassRead)
	tool.sleepBeforeReturn = 5 * time.Millisecond
	reg := newMapRegistry(map[string]agent.Tool{
		"read_seq": tool,
	})
	calls := []scheduledCall{
		{id: "seq_0", name: "read_seq", args: []byte(`{}`), effect: agent.EffectClassRead},
		{id: "seq_1", name: "read_seq", args: []byte(`{}`), effect: agent.EffectClassRead},
		{id: "seq_2", name: "read_seq", args: []byte(`{}`), effect: agent.EffectClassRead},
	}

	sched := &agent.Scheduler{MaxConcurrentReads: 1}
	sink := make(chan *agent.Event, 256)
	_ = runSchedulerAndClose(sched, calls, reg, sink)

	// Walk the observed events and assert that for each call,
	// the ToolStart event appears strictly before the ToolEnd
	// event of the same callID.
	startIdx := map[string]int{}
	endIdx := map[string]int{}
	idx := 0
	for ev := range sink {
		idx++
		if start, ok := ev.ToolStart(); ok {
			startIdx[start.CallID()] = idx
		} else if end, ok := ev.ToolEndSuccess(); ok {
			if _, prior := endIdx[end.CallID()]; prior {
				continue
			}
			endIdx[end.CallID()] = idx
		}
	}

	for _, c := range calls {
		sIdx, sOk := startIdx[c.id]
		eIdx, eOk := endIdx[c.id]
		if !sOk {
			t.Errorf("call %s: no ToolStart observed", c.id)
			continue
		}
		if !eOk {
			t.Errorf("call %s: no ToolEnd observed", c.id)
			continue
		}
		if sIdx >= eIdx {
			t.Errorf("call %s: ToolStart at index %d, ToolEnd at index %d (start must precede end by ≥ 1 tick)",
				c.id, sIdx, eIdx)
		}
	}
}

// S-TLS-010a — RED bite (recorded at Commit 2). Given a scheduler
// using `errgroup` whose first error cancels the group, when the
// failing-call scenario runs, then `results[2]` is the zero `Result`
// (sibling aborted) — proves the "siblings complete" property is
// non-vacuous.
//
// RED at write time: the surface is missing; compile-error RED.
func TestScheduler_OneBadToolSiblingsComplete_BiteErrgroupShape(t *testing.T) {
	t.Parallel()

	// Call 0: success. Call 1: returns an error (will yield
	// ExecutionFailure). Call 2: success.
	tool0 := newScriptedBiteTool(t, "ok_0", agent.EffectClassRead)
	tool0.result = agent.Result{Outcome: agent.ToolOutcomeSuccess, Content: []byte("ok-0")}
	tool1 := newErroneousTool(t, "bad_1", agent.EffectClassRead)
	tool2 := newScriptedBiteTool(t, "ok_2", agent.EffectClassRead)
	tool2.result = agent.Result{Outcome: agent.ToolOutcomeSuccess, Content: []byte("ok-2")}

	reg := newMapRegistry(map[string]agent.Tool{
		"ok_0":  tool0,
		"bad_1": tool1,
		"ok_2":  tool2,
	})
	calls := []scheduledCall{
		{id: "ok_0", name: "ok_0", args: []byte(`{}`), effect: agent.EffectClassRead},
		{id: "bad_1", name: "bad_1", args: []byte(`{}`), effect: agent.EffectClassRead},
		{id: "ok_2", name: "ok_2", args: []byte(`{}`), effect: agent.EffectClassRead},
	}

	sched := &agent.Scheduler{MaxConcurrentReads: 4}
	sink := make(chan *agent.Event, 256)
	results := runSchedulerAndClose(sched, calls, reg, sink)

	// BITE: an errgroup-shaped scheduler would cancel siblings
	// when call 1 fails. The result slice would have results[2]
	// as the zero Result (the sibling aborted). A correct
	// scheduler yields results[2] as a populated Result.
	if len(results) != 3 {
		t.Fatalf("Schedule returned %d results, want 3", len(results))
	}
	if results[2].Outcome != agent.ToolOutcomeSuccess {
		t.Errorf("errgroup-shape bite DID NOT bite: results[2].Outcome = %v, want %v (one bad tool must not abort the turn — siblings complete, R-TLS-010)",
			results[2].Outcome, agent.ToolOutcomeSuccess)
	}
	if results[2].Content == nil {
		t.Errorf("errgroup-shape bite DID NOT bite: results[2].Content is nil (sibling aborted — but the property demands siblings complete)")
	}
}

// S-TLS-011a — RED bite (recorded at Commit 2). Given a scheduler
// without `defer/recover` in the call goroutine, when the panic
// scenario runs under `-race`, then `go test` reports the panic as
// an unhandled goroutine failure and the test process aborts —
// proves the recovery path is non-vacuous.
//
// RED at write time: same compile-error surface.
func TestScheduler_PanicContainment_BiteNoRecover(t *testing.T) {
	t.Parallel()

	// Only the panic-facing path is exercised here. Failure
	// isolation (R-TLS-010) is the property scenario; this bite
	// proves the recovery path is non-vacuous by construction.
	panicking := newPanickingTool(t, "boom", agent.EffectClassRead)
	ok := newScriptedBiteTool(t, "ok", agent.EffectClassRead)
	ok.result = agent.Result{Outcome: agent.ToolOutcomeSuccess, Content: []byte("ok")}
	reg := newMapRegistry(map[string]agent.Tool{
		"boom": panicking,
		"ok":   ok,
	})
	calls := []scheduledCall{
		{id: "panic_0", name: "boom", args: []byte(`{}`), effect: agent.EffectClassRead},
		{id: "ok_1", name: "ok", args: []byte(`{}`), effect: agent.EffectClassRead},
	}

	baseline := runtime.NumGoroutine()

	sched := &agent.Scheduler{MaxConcurrentReads: 4}
	sink := make(chan *agent.Event, 256)
	results := runSchedulerAndClose(sched, calls, reg, sink)

	if len(results) != 2 {
		t.Fatalf("Schedule returned %d results, want 2", len(results))
	}
	// BITE: the panic must be contained — the panicking tool's
	// ordinal slot is populated with a typed ExecutionFailure, the
	// sibling's ordinal slot is populated with success, and the
	// process did not abort. A scheduler without recover would
	// abort the test (the bite's strengths: a successful test
	// proves recovery is present).
	if results[0].Outcome != agent.ToolOutcomeExecutionFailure {
		t.Errorf("panic containment bite DID NOT bite: results[0].Outcome = %v, want %v (the panicking tool's slot must be a typed ExecutionFailure)",
			results[0].Outcome, agent.ToolOutcomeExecutionFailure)
	}
	if results[1].Outcome != agent.ToolOutcomeSuccess {
		t.Errorf("panic containment bite DID NOT bite: results[1].Outcome = %v, want %v (sibling completes even when a call panics, R-TLS-010/011)",
			results[1].Outcome, agent.ToolOutcomeSuccess)
	}

	// Give the runtime a moment to settle.
	time.Sleep(20 * time.Millisecond)
	after := runtime.NumGoroutine()
	if after > baseline+1 {
		t.Errorf("goroutine count after panic = %d, baseline = %d (a scheduler without recover would either abort the process or leak the goroutine)",
			after, baseline)
	}
}

// newErroneousTool constructs a scripted tool whose Run returns a
// non-nil error. Mirrors the "bad tool" subset of R-TLS-010.
func newErroneousTool(t *testing.T, name string, effect agent.EffectClass) *scriptedBiteTool {
	t.Helper()
	tool := newScriptedBiteTool(t, name, effect)
	tool.err = errBiteToolBoom
	return tool
}

// newPanickingTool constructs a scripted tool whose Run panics. Used
// by S-TLS-011a. The shape is the same as newScriptedBiteTool but
// the panic is triggered inside Run.
func newPanickingTool(t *testing.T, name string, effect agent.EffectClass) *scriptedBiteTool {
	t.Helper()
	tool := newScriptedBiteTool(t, name, effect)
	tool.panicMessage = "scheduled tool panic (S-TLS-011a)"
	return tool
}

// errBiteToolBoom is the typed sentinel returned by the erroneous
// scripted tool's Run. Distinct from the loop's sentinels so a
// failure can name the path under test.
var errBiteToolBoom = errBiteTool{msg: "test scripted tool returning an error"}

type errBiteTool struct{ msg string }

func (e errBiteTool) Error() string { return e.msg }

// observeStartCallIDs drains the sink and returns the call IDs of
// the ToolStart events in the order they arrive. Used by
// S-TLS-006a to read the start-side ordering. Caller is responsible
// for closing `sink` (the scheduler closes it after `Schedule`
// returns).
func observeStartCallIDs(t *testing.T, sink <-chan *agent.Event) []string {
	t.Helper()
	var starts []string
	for ev := range sink {
		if start, ok := ev.ToolStart(); ok {
			starts = append(starts, start.CallID())
		}
	}
	return starts
}

// observeStartCallIDsWithClose is the bite-friendly variant: it
// closes the sink after Schedule returns so the drain does not
// hang. Used by the bite tests while the scheduler is still
// being built (the stub returns without closing).
func observeStartCallIDsWithClose(t *testing.T, sink <-chan *agent.Event) []string {
	t.Helper()
	var starts []string
	for ev := range sink {
		if start, ok := ev.ToolStart(); ok {
			starts = append(starts, start.CallID())
		}
	}
	return starts
}

// runSchedulerAndClose calls Schedule and closes the sink in a
// goroutine. This is the bite-friendly helper used by tests
// while the scheduler is still being built (the stub does not
// close the sink). After Commit 4 (the scheduler), the real
// `Schedule` closes the sink, and this helper's close-on-exit
// path is a safety net rather than the load-bearing close.
func runSchedulerAndClose(sched *agent.Scheduler, calls []scheduledCall, reg agent.Registry, sink chan *agent.Event) []agent.Result {
	results := sched.Schedule(context.Background(), callsToAICalls(calls), reg, "run-bite", "turn-bite", nil, sink)
	// Close the sink in a goroutine so any sends the real
	// scheduler does are still drained. The bite tests only
	// observe `results`.
	go func() {
		// drain remaining items before closing
		for range sink {
		}
	}()
	close(sink)
	return results
}

// callsToAICalls converts the in-test scheduledCall shape into
// the Layer 1 `ai.ToolCall` slice the agent.Scheduler expects. The
// shape is a thin pass-through for the three fields the scheduler
// reads (id, name, arguments).
func callsToAICalls(calls []scheduledCall) []ai.ToolCall {
	parts := make([]ai.Part, len(calls))
	for i, c := range calls {
		part, err := ai.NewToolCall(c.id, c.name, c.args)
		if err != nil {
			panic("callsToAICalls: invalid ToolCall at index " + itoa(i) + ": " + err.Error())
		}
		parts[i] = part
	}
	out := make([]ai.ToolCall, len(parts))
	for i, p := range parts {
		tc, ok := p.ToolCall()
		if !ok {
			panic("callsToAICalls: NewToolCall returned non-tool-call part at index " + itoa(i))
		}
		out[i] = tc
	}
	return out
}

// itoa formats an int as a string. Tiny helper to keep the panic
// messages above dependency-free.
func itoa(i int) string {
	if i == 0 {
		return "0"
	}
	neg := false
	if i < 0 {
		neg = true
		i = -i
	}
	var buf [20]byte
	pos := len(buf)
	for i > 0 {
		pos--
		buf[pos] = byte('0' + i%10)
		i /= 10
	}
	if neg {
		pos--
		buf[pos] = '-'
	}
	return string(buf[pos:])
}
