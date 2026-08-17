// AG-14 — the cancellation vocabulary (R-CAN-001, R-CAN-006).
//
// This file declares pure vocabulary only: two errors.Is sentinels
// (ErrInterrupted, ErrShutdown), a typed post-shutdown prompt refusal
// that wraps the shutdown sentinel (ErrPromptAfterShutdown), an
// errors.As-extractable carrier naming a tool call still running past
// the wind-down bound (DetachedCallError), and the package's
// documented wind-down bound default (defaultWindDownBound). No
// behavior is wired to any of these here — harness.go's signal
// methods, loop.go's cause check and scheduler.go's detach select
// (this and later AG-14 phases) are the consumers.
//
// # No "fmt" import, deliberately
//
// "fmt" transitively imports "os" and "io/fs" (confirmed the same way
// src/ai/json_syntax.go's own package comment records for
// encoding/json: `go list -deps fmt` reports both in the closure),
// which the production-closure network/filesystem guard
// (import_boundary_test.go, TestLayer2_ProductionClosure_
// ContainsNoNetworkOrFilesystemPackage) forbids by exact path. This
// file instead follows scheduler.go's own established idiom for typed
// sentinel errors: small unexported struct types implementing Error()
// (and, here, Unwrap()) by hand, plus strconv.Quote for the same
// quoted rendering "%q" would produce.
package agent

import (
	"errors"
	"strconv"
	"time"
)

// ErrInterrupted is the errors.Is sentinel an interrupted run's
// returned error satisfies (R-CAN-001, R-CAN-002). It shares no
// identity with [ErrShutdown]: the two name distinct signals, never
// one signal wearing two names.
var ErrInterrupted = errors.New("agent: run interrupted")

// ErrShutdown is the errors.Is sentinel a shut-down run's returned
// error satisfies (R-CAN-001, R-CAN-005).
var ErrShutdown = errors.New("agent: harness shut down")

// promptAfterShutdownError is ErrPromptAfterShutdown's concrete type:
// a fixed message plus an Unwrap route to ErrShutdown, so
// errors.Is(ErrPromptAfterShutdown, ErrShutdown) holds without an
// import of "fmt" (see the package comment).
type promptAfterShutdownError struct{}

func (promptAfterShutdownError) Error() string {
	return "agent: run refused, harness already shut down"
}

// Unwrap reaches ErrShutdown — never ErrInterrupted — through the
// standard single-error errors.Unwrap route.
func (promptAfterShutdownError) Unwrap() error { return ErrShutdown }

// ErrPromptAfterShutdown is the typed refusal [Harness.Run] returns
// for a prompt offered after the harness has been shut down
// (R-CAN-001, R-CAN-005). It wraps [ErrShutdown], so
// errors.Is(err, ErrShutdown) holds for it; it does not wrap
// [ErrInterrupted].
var ErrPromptAfterShutdown error = promptAfterShutdownError{}

// DetachedCallError is the errors.As-extractable cause naming a tool
// call still running once the wind-down bound has expired (R-CAN-001,
// R-CAN-006). Third-party tool code has no goroutine-kill primitive;
// this value is how the overrun is named — by tool and call identity —
// rather than silently abandoned.
type DetachedCallError struct {
	// Tool is the overrunning call's tool name.
	Tool string
	// CallID is the overrunning call's identity.
	CallID string
}

// Error renders the detached-call report for a diagnostic reader,
// naming the tool and the call identity — never any argument or
// result content. strconv.Quote reproduces "%q"'s quoting without an
// import of "fmt" (see the package comment).
func (e *DetachedCallError) Error() string {
	return "agent: tool " + strconv.Quote(e.Tool) + " call " + strconv.Quote(e.CallID) + " is still running past the wind-down bound"
}

// defaultWindDownBound is the documented default wind-down bound
// (R-CAN-006): an order of magnitude above goroutine-scheduling jitter
// under -race (agenttest's own settle window is 50ms,
// stream_kit_leak.go:77). Armed only by cancellation (a healthy run
// never pays it) and overridable per Scheduler value through the
// zero-default Scheduler.WindDownBound field (AG-14 Phase 9).
const defaultWindDownBound = 100 * time.Millisecond
