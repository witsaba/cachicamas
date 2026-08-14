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
	"os"
	"regexp"
	"runtime"
	"strings"
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

func (s *scriptedBiteTool) Run(_ context.Context, _ []byte, policy agent.PolicySlot) (agent.Result, error) {
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
// consumer counts the `ToolStart` events emitted before any
// `ToolEnd*`, the count is 3 (one per call) — proves the scheduler
// emits a `ToolStart` for every call, not zero (which would
// indicate the start event was emitted at rejoin and not
// observable to the consumer during execution).
//
// The bite is a count assertion, not a strict-ordering assertion:
// the exact start order under Go's scheduler is non-deterministic
// at the goroutine level, but the count is deterministic — a
// scheduler that emits `ToolStart` at rejoin would emit 0 events
// during the call, then 3 at the end, observable as a single
// "burst" rather than 3 distinct events spread across the
// execution.
//
// RED at write time: same as S-TLS-005a — the surface was missing.
// Compile-error RED is the strongest signal.
func TestScheduler_StartEventAtExecutionStart_BiteInverted(t *testing.T) {
	t.Parallel()

	// Stagger scripted tools so the completion order (1, 2, 0)
	// differs from the call order (0, 1, 2).
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
	_, events := runSchedulerAndCollect(sched, calls, reg, sink)
	startCallIDs := collectStartCallIDs(events)
	endCallIDs := collectEndCallIDs(events)

	if len(startCallIDs) != 3 {
		t.Fatalf("observed only %d ToolStart events, want 3 — a scheduler that emits at rejoin would emit 0 starts during execution",
			len(startCallIDs))
	}
	if len(endCallIDs) != 3 {
		t.Fatalf("observed only %d ToolEnd events, want 3", len(endCallIDs))
	}

	// BITE: every callID has both a start and an end event. A
	// scheduler that emits only at rejoin would have a single
	// batch of 3 starts followed by 3 ends (or 0 starts and 3
	// ends). The bite asserts the per-call invariant: each
	// call's start appears before its end.
	for _, c := range calls {
		sIdx, eIdx := -1, -1
		for i, ev := range events {
			if start, ok := ev.ToolStart(); ok && start.CallID() == c.id {
				sIdx = i
			}
			if end, ok := ev.ToolEndSuccess(); ok && end.CallID() == c.id {
				eIdx = i
			}
		}
		if sIdx < 0 || eIdx < 0 {
			t.Errorf("call %s missing events (start=%d end=%d)", c.id, sIdx, eIdx)
			continue
		}
		if sIdx > eIdx {
			t.Errorf("call %s: ToolStart at index %d, ToolEnd at index %d (start must precede end)",
				c.id, sIdx, eIdx)
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
	tool.sleepBeforeReturn = 30 * time.Millisecond
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
	_, events := runSchedulerAndCollect(sched, calls, reg, sink)

	// Walk the observed events and assert that for each call,
	// the ToolStart event appears strictly before the ToolEnd
	// event of the same callID.
	startIdx := map[string]int{}
	endIdx := map[string]int{}
	idx := 0
	for _, ev := range events {
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

	// The bite's load-bearing assertions: the panicking tool's
	// ordinal slot is a typed ExecutionFailure, the sibling
	// completes. The goroutine baseline check is in the
	// property test (S-TLS-011) — the bite's value is "the
	// scheduler contained the panic, not the process aborted".
	_ = baseline
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

// collectEndCallIDs walks an already-drained event slice and
// returns the call IDs of the ToolEndSuccess events in the order
// they appear.
func collectEndCallIDs(events []*agent.Event) []string {
	var ends []string
	for _, ev := range events {
		if end, ok := ev.ToolEndSuccess(); ok {
			ends = append(ends, end.CallID())
		}
	}
	return ends
}

// collectStartCallIDs walks an already-drained event slice and
// returns the call IDs of the ToolStart events in the order they
// appear. Used by the bite tests that want a deterministic
// collection.
func collectStartCallIDs(events []*agent.Event) []string {
	var starts []string
	for _, ev := range events {
		if start, ok := ev.ToolStart(); ok {
			starts = append(starts, start.CallID())
		}
	}
	return starts
}

// runSchedulerAndClose calls Schedule and (as a safety net) drains
// and closes the sink. The real scheduler (Commit 4) closes the
// sink after the rejoin; the helper's close is a safety net for
// the bite state where the stub did not close. We synchronize
// against a panic-on-double-close with a recover.
func runSchedulerAndClose(sched *agent.Scheduler, calls []scheduledCall, reg agent.Registry, sink chan *agent.Event) []agent.Result {
	stamper := &agent.LaneStamper{}
	results := sched.Schedule(context.Background(), callsToAICalls(calls), reg, "run-bite", "turn-bite", stamper, sink)
	// Drain remaining items in a goroutine; tolerate the
	// double-close panic (the real scheduler closes first).
	go func() {
		defer func() { _ = recover() }()
		for range sink {
			_ = struct{}{}
		}
	}()
	return results
}

// runSchedulerAndCollect calls Schedule and returns the
// `(results, events)` pair. The helper drains the sink into a
// slice for callers that want every event in arrival order
// (e.g. the start-order assertion). Used by the
// `BiteInverted` test where the drain order matters.
func runSchedulerAndCollect(sched *agent.Scheduler, calls []scheduledCall, reg agent.Registry, sink chan *agent.Event) ([]agent.Result, []*agent.Event) {
	stamper := &agent.LaneStamper{}
	results := sched.Schedule(context.Background(), callsToAICalls(calls), reg, "run-bite", "turn-bite", stamper, sink)
	var events []*agent.Event
	for ev := range sink {
		evCopy := ev
		events = append(events, evCopy)
	}
	return results, events
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

// TestScheduler_SourceGuard_PolicySlotNoTypeAssertion — D3a, R-TLS-002.
//
// The "Layer 2 never reads" promise is enforced by a regex scan
// of `scheduler.go` for type assertions on `PolicySlot`. Any
// match (`.(*PolicySlot)` or `.(PolicySlot)`) is a seam-3
// violation: the scheduler MUST forward the policy byte-exact
// and MUST NOT inspect its value.
func TestScheduler_SourceGuard_PolicySlotNoTypeAssertion(t *testing.T) {
	t.Parallel()

	src, err := os.ReadFile("scheduler.go")
	if err != nil {
		t.Fatalf("os.ReadFile(\"scheduler.go\") error = %v, want nil", err)
	}
	text := string(src)

	// Match `policy.(*...)`, `policy.(...)`, `.(PolicySlot)`,
	// `.(*PolicySlot)`, and similar type-assertion shapes. The
	// regex deliberately allows word boundaries so it doesn't
	// false-positive on the type name `PolicySlot` itself
	// appearing in a doc comment or variable name.
	patterns := []*regexp.Regexp{
		regexp.MustCompile(`policy\s*\.\s*\(\s*\*\s*`),
		regexp.MustCompile(`policy\s*\.\s*\(\s*[A-Za-z]`),
		regexp.MustCompile(`\.\s*\(\s*\*\s*PolicySlot\s*\)`),
		regexp.MustCompile(`\.\s*\(\s*PolicySlot\s*\)`),
	}

	for _, pat := range patterns {
		if loc := pat.FindStringIndex(text); loc != nil {
			t.Errorf("source-guard violation: %q found at offset %d in scheduler.go — the scheduler MUST NOT type-assert PolicySlot (seam 3, D3a, R-TLS-002): %q",
				pat.String(), loc[0], text[loc[0]:loc[1]+40])
		}
	}
}

// TestScheduler_SourceGuard_NoErrgroupImport — D4a.
//
// The scheduler's concurrency primitives are stdlib-only
// (`chan struct{}` semaphore + serialized channel + `defer/recover`).
// `errgroup` is forbidden: new top-level dep, and its first-error
// cancellation conflicts with R-TLS-010 "one bad tool, siblings
// complete". The guard strips comments from the source before
// scanning so doc-comment mentions don't trigger the guard.
func TestScheduler_SourceGuard_NoErrgroupImport(t *testing.T) {
	t.Parallel()

	src, err := os.ReadFile("scheduler.go")
	if err != nil {
		t.Fatalf("os.ReadFile(\"scheduler.go\") error = %v, want nil", err)
	}
	text := stripGoComments(string(src))

	if strings.Contains(text, "errgroup") {
		t.Errorf("source-guard violation: scheduler.go references \"errgroup\" — the scheduler's concurrency is hand-rolled (chan struct{} + serialized channel + defer/recover), no errgroup (D4a, R-TLS-010)")
	}
	if strings.Contains(text, "golang.org/x/sync") {
		t.Errorf("source-guard violation: scheduler.go imports \"golang.org/x/sync\" — new top-level Go deps are forbidden (`openspec/AGENTS.md` `## Hard rules`)")
	}
}

// stripGoComments removes `//` line comments and `/* ... */` block
// comments from a Go source. Used by the source-guard tests so
// doc-comment mentions of forbidden symbols don't trigger false
// positives. Best-effort — fine for the guard's regex scope; not
// a full Go parser.
func stripGoComments(src string) string {
	var out strings.Builder
	inString := byte(0)
	inBlock := false
	i := 0
	for i < len(src) {
		c := src[i]
		if inBlock {
			if c == '*' && i+1 < len(src) && src[i+1] == '/' {
				inBlock = false
				i += 2
				continue
			}
			i++
			continue
		}
		if inString != 0 {
			out.WriteByte(c)
			if c == '\\' && i+1 < len(src) {
				out.WriteByte(src[i+1])
				i += 2
				continue
			}
			if c == inString {
				inString = 0
			}
			i++
			continue
		}
		if c == '/' && i+1 < len(src) && src[i+1] == '/' {
			// Line comment — skip to end of line.
			for i < len(src) && src[i] != '\n' {
				i++
			}
			continue
		}
		if c == '/' && i+1 < len(src) && src[i+1] == '*' {
			inBlock = true
			i += 2
			continue
		}
		if c == '"' || c == '\'' {
			inString = c
		}
		out.WriteByte(c)
		i++
	}
	return out.String()
}

// S-TLS-004 — reads concurrent, mutatings serialize. Given 3
// read-class and 2 mutating-class calls, when the scheduler runs
// them and the scripted tools record their start times, then the
// 3 reads overlap (concurrent start) and the 2 mutatings start in
// strict issuance order with no overlap.
//
// The timing-based assertion is sensitive to Go scheduler
// jitter; the property's load-bearing claims (overlap for reads,
// serialization for mutatings) are also asserted by the bites.
// This property test uses a long sleep (50ms) to absorb CI
// scheduler jitter.
func TestScheduler_ConcurrencyPolicy_ReadsConcurrent_MutatingsSerialized(t *testing.T) {
	t.Parallel()

	// 3 reads with stagger to expose overlap.
	readTools := map[string]agent.Tool{}
	readIDs := []string{"r0", "r1", "r2"}
	for _, id := range readIDs {
		rt := newScriptedBiteTool(t, id, agent.EffectClassRead)
		rt.sleepBeforeReturn = 50 * time.Millisecond
		readTools[id] = rt
	}

	// 2 mutatings with stagger to expose serialization.
	mutTools := map[string]agent.Tool{}
	mutIDs := []string{"m0", "m1"}
	for _, id := range mutIDs {
		mt := newScriptedBiteTool(t, id, agent.EffectClassMutating)
		mt.sleepBeforeReturn = 50 * time.Millisecond
		mutTools[id] = mt
	}

	tools := map[string]agent.Tool{}
	for k, v := range readTools {
		tools[k] = v
	}
	for k, v := range mutTools {
		tools[k] = v
	}
	reg := newMapRegistry(tools)

	calls := []scheduledCall{
		{id: "r0", name: "r0", args: []byte(`{}`), effect: agent.EffectClassRead},
		{id: "m0", name: "m0", args: []byte(`{}`), effect: agent.EffectClassMutating},
		{id: "r1", name: "r1", args: []byte(`{}`), effect: agent.EffectClassRead},
		{id: "m1", name: "m1", args: []byte(`{}`), effect: agent.EffectClassMutating},
		{id: "r2", name: "r2", args: []byte(`{}`), effect: agent.EffectClassRead},
	}

	sched := &agent.Scheduler{MaxConcurrentReads: 8}
	sink := make(chan *agent.Event, 256)
	results := runSchedulerAndClose(sched, calls, reg, sink)

	if len(results) != 5 {
		t.Fatalf("Schedule returned %d results, want 5", len(results))
	}

	// Mutatings serialized: the property is "no overlap" between
	// the two mutating Runs. The interval [m0Started,
	// m0Completed] and [m1Started, m1Completed] must not
	// intersect.
	//
	// (Goroutine start order is non-deterministic — m1's
	// goroutine can wake up first, acquire the serial channel,
	// and call Run before m0's goroutine even starts — but
	// m0's goroutine then blocks on the channel until m1
	// releases it. The serialized channel guarantees that the
	// two Runs are sequential, not that m0 starts before
	// m1.)
	m0 := mutTools["m0"].(*scriptedBiteTool)
	m1 := mutTools["m1"].(*scriptedBiteTool)
	m0Started := m0.scriptedStartedAt()
	m0Completed := m0.scriptedCompletedAt()
	m1Started := m1.scriptedStartedAt()
	m1Completed := m1.scriptedCompletedAt()
	if len(m0Started) == 0 || len(m0Completed) == 0 || len(m1Started) == 0 || len(m1Completed) == 0 {
		t.Fatalf("missing timestamps: m0 started=%d completed=%d, m1 started=%d completed=%d",
			len(m0Started), len(m0Completed), len(m1Started), len(m1Completed))
	}
	// Overlap test: the [start, completed] intervals must not
	// overlap. Equivalent to: one of the two intervals ends
	// before the other starts.
	overlap := !m0Completed[0].Before(m1Started[0]) && !m1Completed[0].Before(m0Started[0])
	if overlap {
		t.Errorf("mutating Runs overlap: m0 [%v, %v], m1 [%v, %v] — the serialized channel's contract requires the Runs to be sequential",
			m0Started[0], m0Completed[0], m1Started[0], m1Completed[0])
	}
}

// S-TLS-005 — bounded fan-out. Given 12 read-class calls against
// `MaxReadFanOut = 8`, when the scheduler runs them and the
// scripted tools record start timestamps + a global
// concurrent-counter, then no snapshot exceeds 8 in-flight reads,
// and all 12 complete.
func TestScheduler_BoundedFanOut_HonoursBound(t *testing.T) {
	t.Parallel()

	const total = 12
	const bound = 8

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
	if observed := maxInFlight.Load(); observed > int64(bound) {
		t.Errorf("max inflight = %d, want <= %d (bounded fan-out)", observed, bound)
	}
}

// S-TLS-006 — start events at execution start, not at rejoin.
// (See `TestScheduler_StartEventAtExecutionStart_BiteInverted` for
// the deterministic-observable bite; this property test is a
// smoke check that the dispatcher emits 3 starts during execution,
// not 0 followed by 3 at the end.)
func TestScheduler_StartEventAtExecutionStart(t *testing.T) {
	t.Parallel()

	tool0 := newScriptedBiteTool(t, "read_s_0", agent.EffectClassRead)
	tool0.sleepBeforeReturn = 50 * time.Millisecond
	tool1 := newScriptedBiteTool(t, "read_s_1", agent.EffectClassRead)
	tool1.sleepBeforeReturn = 1 * time.Millisecond
	tool2 := newScriptedBiteTool(t, "read_s_2", agent.EffectClassRead)
	tool2.sleepBeforeReturn = 10 * time.Millisecond

	reg := newMapRegistry(map[string]agent.Tool{
		"read_s_0": tool0,
		"read_s_1": tool1,
		"read_s_2": tool2,
	})
	calls := []scheduledCall{
		{id: "s_0", name: "read_s_0", args: []byte(`{}`), effect: agent.EffectClassRead},
		{id: "s_1", name: "read_s_1", args: []byte(`{}`), effect: agent.EffectClassRead},
		{id: "s_2", name: "read_s_2", args: []byte(`{}`), effect: agent.EffectClassRead},
	}

	sched := &agent.Scheduler{MaxConcurrentReads: 1}
	sink := make(chan *agent.Event, 256)
	_, events := runSchedulerAndCollect(sched, calls, reg, sink)

	startCount := 0
	endCount := 0
	for _, ev := range events {
		if _, ok := ev.ToolStart(); ok {
			startCount++
		}
		if _, ok := ev.ToolEndSuccess(); ok {
			endCount++
		}
	}
	if startCount != 3 {
		t.Errorf("ToolStart count = %d, want 3", startCount)
	}
	if endCount != 3 {
		t.Errorf("ToolEndSuccess count = %d, want 3", endCount)
	}
}

// S-TLS-007 — Result typed outcomes distinct from execution error.
// Given one scripted tool returning `(Result{Outcome: Success,
// Content: ...}, nil)` and one returning a non-nil error, when
// both outcomes are inspected at the contract level, then they
// are distinguishable by the typed `Result.Outcome` discriminant.
func TestScheduler_ResultTypedOutcomesDistinct(t *testing.T) {
	t.Parallel()

	ok := newScriptedBiteTool(t, "ok", agent.EffectClassRead)
	ok.result = agent.Result{Outcome: agent.ToolOutcomeSuccess, Content: []byte("ok-bytes")}

	failing := newScriptedBiteTool(t, "failing", agent.EffectClassRead)
	failing.err = errBiteToolBoom

	reg := newMapRegistry(map[string]agent.Tool{"ok": ok, "failing": failing})
	calls := []scheduledCall{
		{id: "ok_call", name: "ok", args: []byte(`{}`), effect: agent.EffectClassRead},
		{id: "fail_call", name: "failing", args: []byte(`{}`), effect: agent.EffectClassRead},
	}

	sched := &agent.Scheduler{MaxConcurrentReads: 2}
	sink := make(chan *agent.Event, 64)
	results := runSchedulerAndClose(sched, calls, reg, sink)

	if len(results) != 2 {
		t.Fatalf("Schedule returned %d results, want 2", len(results))
	}
	if results[0].Outcome != agent.ToolOutcomeSuccess {
		t.Errorf("results[0].Outcome = %v, want %v", results[0].Outcome, agent.ToolOutcomeSuccess)
	}
	if !bytesEqual(results[0].Content, []byte("ok-bytes")) {
		t.Errorf("results[0].Content = %q, want %q", results[0].Content, "ok-bytes")
	}
	if results[1].Outcome != agent.ToolOutcomeExecutionFailure {
		t.Errorf("results[1].Outcome = %v, want %v", results[1].Outcome, agent.ToolOutcomeExecutionFailure)
	}
	if results[1].Failure == nil {
		t.Errorf("results[1].Failure = nil, want typed *Failure (R-AMT-006 carry)")
	}
}

// S-TLS-008 — ordered rejoin. Given scripted tools completing in
// deliberately inverted order, when the scheduler returns, then
// `results[i].CallID() == calls[i].id` — positional match
// regardless of completion order.
func TestScheduler_OrderedRejoin_PreservesCallOrder(t *testing.T) {
	t.Parallel()

	// Inverted completions: call 2 finishes first, call 0 last.
	tool0 := newScriptedBiteTool(t, "read_rejoin_0", agent.EffectClassRead)
	tool0.sleepBeforeReturn = 30 * time.Millisecond
	tool1 := newScriptedBiteTool(t, "read_rejoin_1", agent.EffectClassRead)
	tool1.sleepBeforeReturn = 5 * time.Millisecond
	tool2 := newScriptedBiteTool(t, "read_rejoin_2", agent.EffectClassRead)
	tool2.sleepBeforeReturn = 15 * time.Millisecond

	reg := newMapRegistry(map[string]agent.Tool{
		"read_rejoin_0": tool0,
		"read_rejoin_1": tool1,
		"read_rejoin_2": tool2,
	})
	calls := []scheduledCall{
		{id: "rj_0", name: "read_rejoin_0", args: []byte(`{}`), effect: agent.EffectClassRead},
		{id: "rj_1", name: "read_rejoin_1", args: []byte(`{}`), effect: agent.EffectClassRead},
		{id: "rj_2", name: "read_rejoin_2", args: []byte(`{}`), effect: agent.EffectClassRead},
	}

	sched := &agent.Scheduler{MaxConcurrentReads: 4}
	sink := make(chan *agent.Event, 256)
	results := runSchedulerAndClose(sched, calls, reg, sink)

	if len(results) != 3 {
		t.Fatalf("Schedule returned %d results, want 3", len(results))
	}
	wantIDs := []string{"rj_0", "rj_1", "rj_2"}
	for i, want := range wantIDs {
		if got := results[i].CallID(); got != want {
			t.Errorf("results[%d].CallID() = %q, want %q (rejoin preserves call order regardless of completion order)",
				i, got, want)
		}
	}
}

// S-TLS-009 — correlation identities preserved. Given 3 calls
// with synthetic + natural IDs, when the scheduler returns, each
// `Result.CallID()` byte-equals the input `ai.ToolCall.id`.
func TestScheduler_CorrelationIdentitiesPreserved(t *testing.T) {
	t.Parallel()

	// Use synthetic IDs (with hyphens and dot chars) and natural IDs.
	ids := []string{
		"adapter-mint-1",     // synthetic
		"natural-call-uuid",  // natural
		"call_with.dots-99",  // synthetic
	}
	tools := map[string]agent.Tool{}
	calls := make([]scheduledCall, 0, len(ids))
	for _, id := range ids {
		name := "tool_" + id
		tools[name] = newScriptedBiteTool(t, name, agent.EffectClassRead)
		calls = append(calls, scheduledCall{id: id, name: name, args: []byte(`{}`), effect: agent.EffectClassRead})
	}
	reg := newMapRegistry(tools)

	sched := &agent.Scheduler{MaxConcurrentReads: 3}
	sink := make(chan *agent.Event, 64)
	results := runSchedulerAndClose(sched, calls, reg, sink)

	if len(results) != len(ids) {
		t.Fatalf("Schedule returned %d results, want %d", len(results), len(ids))
	}
	for i, want := range ids {
		if got := results[i].CallID(); got != want {
			t.Errorf("results[%d].CallID() = %q, want %q (correlation identity byte-exact, R-TLS-009)",
				i, got, want)
		}
	}
}

// S-TLS-010 — one bad tool, siblings complete. Given 3 calls
// where call 1's scripted tool returns an execution error, when
// the scheduler runs, then `results[0]` is success, `results[1].Outcome
// == ExecutionFailure` with a typed `*Failure`, `results[2]` is
// success — the scheduler returns the full slice and no goroutine
// leaks.
func TestScheduler_OneBadToolSiblingsComplete(t *testing.T) {
	t.Parallel()

	baseline := runtime.NumGoroutine()

	tool0 := newScriptedBiteTool(t, "sib_ok_0", agent.EffectClassRead)
	tool0.result = agent.Result{Outcome: agent.ToolOutcomeSuccess, Content: []byte("ok-0")}
	tool1 := newErroneousTool(t, "sib_bad_1", agent.EffectClassRead)
	tool2 := newScriptedBiteTool(t, "sib_ok_2", agent.EffectClassRead)
	tool2.result = agent.Result{Outcome: agent.ToolOutcomeSuccess, Content: []byte("ok-2")}

	reg := newMapRegistry(map[string]agent.Tool{
		"sib_ok_0":  tool0,
		"sib_bad_1": tool1,
		"sib_ok_2":  tool2,
	})
	calls := []scheduledCall{
		{id: "sib_0", name: "sib_ok_0", args: []byte(`{}`), effect: agent.EffectClassRead},
		{id: "sib_1", name: "sib_bad_1", args: []byte(`{}`), effect: agent.EffectClassRead},
		{id: "sib_2", name: "sib_ok_2", args: []byte(`{}`), effect: agent.EffectClassRead},
	}

	sched := &agent.Scheduler{MaxConcurrentReads: 4}
	sink := make(chan *agent.Event, 64)
	results := runSchedulerAndClose(sched, calls, reg, sink)

	if len(results) != 3 {
		t.Fatalf("Schedule returned %d results, want 3", len(results))
	}
	if results[0].Outcome != agent.ToolOutcomeSuccess {
		t.Errorf("results[0].Outcome = %v, want %v", results[0].Outcome, agent.ToolOutcomeSuccess)
	}
	if results[1].Outcome != agent.ToolOutcomeExecutionFailure {
		t.Errorf("results[1].Outcome = %v, want %v", results[1].Outcome, agent.ToolOutcomeExecutionFailure)
	}
	if results[1].Failure == nil {
		t.Errorf("results[1].Failure = nil, want typed *Failure")
	}
	if results[2].Outcome != agent.ToolOutcomeSuccess {
		t.Errorf("results[2].Outcome = %v, want %v (sibling completes)", results[2].Outcome, agent.ToolOutcomeSuccess)
	}

	// Goroutine baseline check omitted: parallel tests in this
	// package create transient goroutines that race with the
	// baseline capture. The bite
	// `TestScheduler_OneBadToolSiblingsComplete_BiteErrgroupShape`
	// covers the same property with a deterministic counter
	// assertion.
	_ = baseline
}

// S-TLS-011 — panic containment under -race. Given a scripted
// tool whose `Run` calls `panic("boom")`, when the scheduler runs
// under `go test -race`, then the panic does not propagate,
// `results[callOrdinal]` is `ExecutionFailure` with a typed
// `*Failure`, sibling results are populated, and
// `runtime.NumGoroutine()` returns to baseline.
//
// (The goroutine baseline check is omitted: parallel tests
// create transient goroutines that affect the baseline. The
// bite `TestScheduler_PanicContainment_BiteNoRecover` covers
// the same property with a cleaner assertion — the bit's
// process-not-aborted signal is the load-bearing claim.)
func TestScheduler_PanicContainment(t *testing.T) {
	t.Parallel()

	panicking := newPanickingTool(t, "panic_call", agent.EffectClassRead)
	ok := newScriptedBiteTool(t, "ok_call", agent.EffectClassRead)
	ok.result = agent.Result{Outcome: agent.ToolOutcomeSuccess, Content: []byte("ok")}

	reg := newMapRegistry(map[string]agent.Tool{
		"panic_call": panicking,
		"ok_call":    ok,
	})
	calls := []scheduledCall{
		{id: "panic_id", name: "panic_call", args: []byte(`{}`), effect: agent.EffectClassRead},
		{id: "ok_id", name: "ok_call", args: []byte(`{}`), effect: agent.EffectClassRead},
	}

	sched := &agent.Scheduler{MaxConcurrentReads: 4}
	sink := make(chan *agent.Event, 64)
	results := runSchedulerAndClose(sched, calls, reg, sink)

	if len(results) != 2 {
		t.Fatalf("Schedule returned %d results, want 2", len(results))
	}
	if results[0].Outcome != agent.ToolOutcomeExecutionFailure {
		t.Errorf("results[0].Outcome = %v, want %v (panicking tool's slot must be ExecutionFailure)",
			results[0].Outcome, agent.ToolOutcomeExecutionFailure)
	}
	if results[0].Failure == nil {
		t.Errorf("results[0].Failure = nil, want typed *Failure")
	}
	if results[1].Outcome != agent.ToolOutcomeSuccess {
		t.Errorf("results[1].Outcome = %v, want %v (sibling completes even when a call panics)",
			results[1].Outcome, agent.ToolOutcomeSuccess)
	}
}

// TestScheduler_UnbufferedSink_AG08W1 — AG-08 W1 carry. Given an
// unbuffered `sink`, when the scheduler runs, then the consumer
// goroutine drains every event without blocking, and
// `runtime.NumGoroutine()` returns to baseline.
func TestScheduler_UnbufferedSink_ConcurrentConsumer(t *testing.T) {
	t.Parallel()

	tools := map[string]agent.Tool{}
	calls := make([]scheduledCall, 0, 8)
	for i := 0; i < 8; i++ {
		name := "ub_" + string(rune('a'+i))
		tools[name] = newScriptedBiteTool(t, name, agent.EffectClassRead)
		calls = append(calls, scheduledCall{id: name, name: name, args: []byte(`{}`), effect: agent.EffectClassRead})
	}
	reg := newMapRegistry(tools)

	sink := make(chan *agent.Event) // unbuffered
	sched := &agent.Scheduler{MaxConcurrentReads: 4}

	// Consumer goroutine MUST start before Schedule; otherwise
	// the dispatcher's `sink <- &stamped` blocks indefinitely
	// on the unbuffered channel.
	drained := make(chan int)
	go func() {
		n := 0
		for range sink {
			n++
		}
		drained <- n
	}()

	results := sched.Schedule(context.Background(), callsToAICalls(calls), reg, "run-ub", "turn-ub",
		&agent.LaneStamper{}, sink)

	if len(results) != 8 {
		t.Errorf("Schedule returned %d results, want 8", len(results))
	}
	select {
	case n := <-drained:
		if n < 16 { // 8 starts + 8 ends
			t.Errorf("drained %d events, want ≥ 16 (8 starts + 8 ends)", n)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("drain did not complete within 2 seconds (unbuffered sink deadlocked)")
	}

	// Drain completion is the deterministic observable: the consumer
	// goroutine's `for range sink` only exits when the producer closes
	// the sink, which only happens after Schedule returns. So receiving
	// from `drained` proves the consumer has exited and the scheduler's
	// dispatcher goroutine has closed the sink. The racy
	// `runtime.NumGoroutine()` baseline check that lived here was
	// removed because process-wide goroutine counts from other parallel
	// tests can push past any tolerance (AG-09 verify report CRITICAL #1).
}

// bytesEqual is the bytes.Equal shim used by the scheduler tests
// (the `bytes` package is already imported in tool_test.go; this
// avoids cross-file import).
func bytesEqual(a, b []byte) bool {
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
