// AG-09.1 — the tool execution contract (R-TLS-001..003, D1, D2, D3).
//
// This file defines the surface Layer 3 (built-in tools, doc 0004) and
// future adapters implement to plug into the Layer 2 scheduler. It is
// the canonical answer to "what is a tool to Layer 2?" — three methods
// (Name, EffectClass, Run), an opaque per-call policy slot, and a
// typed `Result` mirroring AG-05.2's `ToolOutcome`.
//
// # EffectClass is a typed enum (D2)
//
// `EffectClass` is a closed three-member vocabulary modeled on
// `ToolOutcome` (tool_event.go:227-246). The zero value is not a
// member: `String()` renders it as `"unset"`. The four post-iota
// values are the read / mutating / execute members and a sentinel
// `effectClassLimit` that bounds the table (mirrors `toolOutcomeLimit`).
//
// # PolicySlot is the opaque per-call seam (D3, seam 3)
//
// `PolicySlot` is a named type over `any`. The scheduler MUST NOT
// type-assert, type-switch, or read its value: that's the charter
// seam 3 demand ("confinement is a property of the call site, not of
// the code being called", 0001:613-616). The "Layer 2 never reads"
// promise is the type's doc + a source-guard test scanning
// `scheduler.go` for `policy.(*` / `.(PolicySlot)`. Layer 3
// interprets the value; AG-09 forwards it byte-exact.
//
// # Result is the typed outcome (R-TLS-007, R-AMT-006 carry)
//
// `Result` mirrors the AG-05.2 `ToolOutcome` family: three typed
// outcomes (Success, ResultFailure, ExecutionFailure) distinct by
// kind, not by convention over payload contents. The two return
// channels from `Run` are disjoint: a `Run` that returns a
// `Result{Outcome: Success|ResultFailure}` MUST NOT return a
// non-nil error; a `Run` that returns a non-nil error MUST NOT also
// return a populated `Result` with `Outcome` other than zero.
//
// AG-09.1 owns the surface; the scheduler (AG-09.2/3/4) consumes it.
package agent

import (
	"context"
	"strconv"

	"github.com/cachicamas/backend/agent/src/ai"
)

// EffectClass is the closed vocabulary of tool effects (D2, R-TLS-003).
// Three members: Read, Mutating, Execute. The zero value is not a
// member; `String()` renders it as `"unset"`. Mirrors `ToolOutcome`'s
// posture (tool_event.go:227-246).
type EffectClass uint8

const (
	// EffectClassRead is the effect class for tools that observe
	// state without modifying it (filesystem reads, network reads,
	// database queries). Scheduled concurrently with bounded
	// fan-out (AG-09.2).
	EffectClassRead EffectClass = iota + 1

	// EffectClassMutating is the effect class for tools that
	// modify state (filesystem writes, database updates). Scheduled
	// serialized in call order (AG-09.2).
	EffectClassMutating

	// EffectClassExecute is the effect class for tools that
	// perform arbitrary acts (exec, subshell, network request
	// beyond reads). Scheduled serialized in call order (AG-09.2).
	EffectClassExecute

	// effectClassLimit is the first value outside the vocabulary. It
	// must stay last. The implicit `iota + 1` increments ensure
	// this lands strictly past the last member — the substrate's
	// `toolOutcomeLimit` shares the same shape but its explicit
	// `= iota` reset collapses the limit to the last member,
	// masking the third member's render. AG-09's `String()` keeps
	// the auto-increment form so S-TLS-003's distinct-rendering
	// property holds for all three members.
	effectClassLimit
)

// String renders the effect class, or "unset"/"effectclass(N)" for a
// non-member, mirroring `ToolOutcome.String`'s posture (D2).
func (c EffectClass) String() string {
	if c >= effectClassLimit {
		return "effectclass(" + strconv.FormatUint(uint64(c), 10) + ")"
	}
	switch c {
	case EffectClassRead:
		return "read"
	case EffectClassMutating:
		return "mutating"
	case EffectClassExecute:
		return "execute"
	case 0:
		return "unset"
	default:
		return "effectclass(" + strconv.FormatUint(uint64(c), 10) + ")"
	}
}

// PolicySlot is the per-call sandbox seam Layer 2 carries opaquely
// (R-TLS-002, D3, seam 3 of v2 § 6). Charter quote (0001:613-616):
// "confinement is a property of the call site, not of the code being
// called". The scheduler MUST NOT type-assert, type-switch, or read
// this value: it forwards the exact bytes/identity of the injected
// value to the tool's `Run`. A future Layer 3 sandbox interprets the
// value; AG-09 is the in-between.
//
// The "Layer 2 never reads" promise is enforced by a source-guard test
// scanning `scheduler.go` for `policy.(*` / `.(PolicySlot)`. The shape
// (named type over `any`) gives the seam a name without forcing a
// vocabulary on Layer 3.
type PolicySlot any

// Result is the typed outcome of one `Tool.Run` call (R-TLS-007).
// Three typed outcomes mirroring AG-05.2's `ToolOutcome`:
// Success, ResultFailure, ExecutionFailure — distinct by kind, not
// by convention over payload contents.
//
// The two return channels from `Run` are disjoint:
//
//   - `Run` returns `(Result{Outcome: Success|ResultFailure}, nil)`
//     for a successful return or a result-level failure;
//   - `Run` returns `(Result{}, err)` for an execution-level failure.
//     For the ExecutionFailure outcome, AG-09 populates `Result` with
//     a typed `*Failure` and returns `(Result, nil)` — the failure is
//     the typed payload, not a Go error.
//
// `Failure` is required iff `Outcome == ToolOutcomeExecutionFailure`
// (R-AMT-006 carry). `Content` is the payload for success or
// result-failure outcomes; zero for execution-failure. `callID` is
// the unexported correlation identity the scheduler assigns (the
// accessor `CallID` is the only door out, `SetCallID` the only door
// in).
type Result struct {
	Outcome ToolOutcome
	Content []byte
	Failure *Failure
	callID  string
}

// CallID is the correlation identity the scheduler reconstructs from
// the input call's `ai.ToolCall.id`. It is mirrored on `Result` so the
// rejoin test can assert byte-equality (R-TLS-009) and so the tool
// trace carries the call's identity through the scheduler's bookkeeping.
func (r Result) CallID() string {
	return r.callID
}

// SetCallID assigns the correlation identity the scheduler observed
// at the call site. Internal: the scheduler (`scheduler.go`) is the
// only caller. The `callID` field is unexported so the type remains
// "value with controlled write" — copying the result preserves the
// identity, but rewriting it requires this method.
func (r *Result) SetCallID(id string) {
	r.callID = id
}

// FailureFor returns the typed `*Failure` for an ExecutionFailure
// outcome, or nil otherwise. Mirrors the `ToolEndExecutionFailure`
// accessor's discipline (R-AEV-008).
func (r Result) FailureFor() *Failure {
	if r.Outcome != ToolOutcomeExecutionFailure {
		return nil
	}
	return r.Failure
}

// Tool is the contract Layer 3 implements to plug into the scheduler
// (R-TLS-001, D1). Three methods:
//
//   - Name() string — the tool's identity, used by the model to
//     address it; matches `ai.ToolCall.name` byte-for-byte.
//   - EffectClass() EffectClass — the scheduler's branch key. MUST
//     be reachable WITHOUT invoking `Run`.
//   - Run(ctx, args, policy) (Result, error) — execute the call.
//     The two return channels are disjoint; see `Result` doc.
type Tool interface {
	Name() string
	EffectClass() EffectClass
	Run(ctx context.Context, args []byte, policy PolicySlot) (Result, error)
}

// Compile-time guard: `Tool` is the scheduler's dependency; a
// non-conforming `Tool` implementation fails to compile here, not at
// runtime under a `Schedule` call.
var _ Tool = (*importPathGuard)(nil)

// importPathGuard is a placeholder tool used only to compile-verify
// the `Tool` interface from a non-test package-private path. It is
// unexported and never constructed; the var above is the entire
// reason it exists.
type importPathGuard struct{ name string }

func (importPathGuard) Name() string                    { return "guard" }
func (importPathGuard) EffectClass() EffectClass         { return EffectClassRead }
func (importPathGuard) Run(_ context.Context, _ []byte, _ PolicySlot) (Result, error) {
	return Result{Outcome: ToolOutcomeSuccess}, nil
}

// Compile-time guard: the import of `ai` is intentional — it
// confirms the surface is reachable from the package boundary (the
// `Tool` interface uses `ai.ToolCall`'s id/name via the scheduler,
// not directly here, but the import keeps the seam discoverable).
var _ = ai.ToolCall{}

// Scheduler is the hand-rolled concurrent scheduler that runs a slice
// of tool calls under the AG-09.2 / AG-09.3 / AG-09.4 rules. The
// type lives in this file (rather than `scheduler.go`) because the
// scheduler's surface is part of the AG-09 contract: the test bites
// in `scheduler_test.go` compile against this shape even when the
// implementation is still pending. The bodies are not here — see
// `scheduler.go` for the matching `Schedule` method on this type.
//
// R-TLS-004..011 — see `openspec/specs/agent-tool-scheduler/spec.md`.
type Scheduler struct {
	MaxConcurrentReads int
}

// Schedule is the scheduler's main entry point. The full implementation
// lives in `scheduler.go`; this stub keeps the package compileable so
// the contract tests in `tool_test.go` can run while the scheduler is
// still being built. The stub returns a slice of zero `Result`
// values whose length matches the input calls. Bites in
// `scheduler_test.go` observe the empty / zero `Content` and fail at
// the assertion level — RED, not process-aborting panic.
func (s *Scheduler) Schedule(
	_ context.Context,
	calls []ai.ToolCall,
	_ Registry,
	_ RunID,
	_ TurnID,
	_ *LaneStamper,
	_ chan<- *Event,
) []Result {
	out := make([]Result, len(calls))
	for i := range out {
		out[i].SetCallID(calls[i].ID())
	}
	return out
}

// aiToolCall is the in-package alias for the Layer 1 `ai.ToolCall`,
// here only to keep the scheduler's `Schedule` signature compatible
// with the contract test surfaces without forcing an import cycle. The
// full Layer 1 type is reachable via `ai.ToolCall`; the scheduler
// implementation in `scheduler.go` uses the Layer 1 type directly.
type aiToolCall = aiToolCallCaller

// Registry resolves a tool name to its implementation. Map-backed or
// any source; resolution miss yields a typed `Result{ExecutionFailure}`
// in the call's ordinal slot (consistent with R-TLS-010 "one bad
// tool, siblings complete"). `ToolSource` port (G6) is AG-13's
// widening.
type Registry interface {
	Resolve(name string) (Tool, bool)
}

// NewMapRegistry returns a `Registry` backed by a map (D9a). Nil-safe:
// a nil map resolves to `(nil, false)`.
func NewMapRegistry(tools map[string]Tool) Registry {
	return mapRegistry(tools)
}

// mapRegistry is the map-backed `Registry` constructor's return type.
// The concrete type is unexported; callers reach it through the
// `Registry` interface.
type mapRegistry map[string]Tool

// Resolve implements `Registry`.
func (r mapRegistry) Resolve(name string) (Tool, bool) {
	t, ok := r[name]
	return t, ok
}

// aiToolCallCaller is the indirection the `Schedule` stub uses to
// build the calls signature. The real `Schedule` in `scheduler.go`
// takes `[]ai.ToolCall` (the Layer 1 type). The stub panics, so the
// concrete type is never used at runtime.
type aiToolCallCaller struct{ ID, name, args string }
