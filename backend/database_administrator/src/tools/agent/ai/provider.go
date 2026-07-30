// Package-level note for the AI-16 ModelProvider interface (provider.go).
//
// AI-16 file note: provider.go declares the ModelProvider interface —
// the single Layer 1 entry point cachicamas_agent holds to start a model
// turn (per ADR 0004). The interface is frozen by the AI-16 design;
// concrete adapter implementations live in their own Go packages
// (AI-22+) and embed `var _ ai.ModelProvider = (*ConcreteProvider)(nil)`
// so that any drift fails at build time.
//
// The unexported helpers below (newStreamBuffer, validatePreStream,
// compileTimeOnlyStub) are documentation anchors for adapter authors;
// they are not part of the public contract. production adapters use
// them as templates or replace them with equivalent logic. The
// canonical compile-time assertion `var _ ModelProvider =
// (*compileTimeOnlyStub)(nil)` is the structural proof that the
// interface and the stub are method-set compatible — drift in either
// side breaks this line at build time.
//
// Imports: stdlib `context` ONLY. The TestPackageImportBoundary canary
// (import_boundary_test.go:25) mechanically enforces this; any cachicamas
// path or vendor SDK appearing in provider.go fails the Layer 1 boundary
// check.
package ai

import (
	"context"
)

// ModelProvider is the Layer 1 entry point cachicamas_agent holds to
// start a model turn.
//
// # Contract preamble (every provider implementation's GoDoc MUST include)
//
//	Stream(ctx, req) returns a receive-only chan ai.Event. The first
//	event on a fresh producer is EventKindResponseStart (sequence 1);
//	the stream terminates with at most one EventKindResponseComplete
//	OR one EventKindError (the two are mutually exclusive). The caller
//	MUST drain the channel until it is closed. The producer goroutine
//	is the sole owner of the channel's lifetime; the caller MUST NOT
//	attempt to close it (the type system prevents this). On ctx
//	cancellation, the producer observes ctx.Done(), terminates its
//	work within a bounded time, and closes the channel. Under
//	cancellation while the producer is blocked or unblockable on a
//	full channel, late events are dropped and the channel is closed
//	without a terminal event (Proposal #2235 § 2 D1, AI-01 § 4
//	Scenario C).
//
// # Lifecycle anchors
//
//   - Channel direction, closure authority, cancellation propagation,
//     and backpressure baseline: AI-01 § 3.1–§ 3.10.
//   - Pre-stream Go-error vs. mid-stream terminal-event split: AI-01
//     § 9 and § 4 Scenarios A & B.
//   - Per-stream Sequence contract, v1 process-scoped counter caveat:
//     Proposal #2235 § 2 D2 and event.go:224–235.
//
// # Import boundary
//
//	The public interface declaration mentions only context.Context,
//	Request, Event, <-chan, and error. Layer 1 (this package) may
//	import only the Go standard library and vendor SDKs (ADR 0004).
//	A concrete provider adapter MAY use vendor SDKs privately; the
//	conversion to ai.Event MUST occur before any value crosses this
//	method boundary.
//
// # Sequence gap detection
//
//	Event.Sequence is per-stream, 1-based, contiguous, and
//	producer-assigned. The v1 implementation uses a process-scoped
//	atomic counter (event.go:235) so two concurrent streams MAY
//	interleave their sequence values; AI-20 (stream testkit) and
//	AI-21 (conformance suite) own per-stream gap detection.
//	TODO(ai-20): Sequence gap detection belongs here.
type ModelProvider interface {
	// Stream starts a model turn and returns a receive-only channel of
	// normalized events. The first event MUST be EventKindResponseStart
	// with Sequence == 1. The stream terminates with at most one
	// EventKindResponseComplete OR one EventKindError (mutually
	// exclusive), followed by channel close. The caller MUST drain
	// until closure.
	//
	// Pre-stream validation failures (including an already-cancelled
	// ctx) MUST surface as a non-nil error with a nil channel; no
	// goroutine is started. Mid-stream failures MUST surface as a
	// terminal EventKindError on the channel, followed by close.
	//
	// ctx is the sole liveness signal: the producer observes
	// ctx.Done() and closes the channel within a bounded time. Under
	// cancellation while the producer is blocked on a full channel,
	// late events are dropped and the channel is closed WITHOUT a
	// terminal EventKindError (Proposal #2235 § 2 D1).
	//
	// req is the package-owned concrete Request value (AI-09). It is
	// passed by value per the package's immutable value-object
	// convention.
	Stream(ctx context.Context, req Request) (<-chan Event, error)
}

// DefaultBufferSize is the v1 channel buffer capacity used by every
// ModelProvider adapter's producer goroutine. AI-01 § 3.6 locks this
// value at 16; AI-32 owns refinement (build-tag override for tests,
// dynamic sizing under backpressure telemetry). Provider adapters
// MUST honor this constant in v1.
//
// TODO(ai-32): refine or expose configurable.
const DefaultBufferSize = 16

// newStreamBuffer returns the standard v1 producer channel buffer.
// Adapters MUST use this helper (or the equivalent literal
// `make(chan Event, DefaultBufferSize)`) so the v1 buffer discipline
// is mechanically uniform across implementations.
//
// This helper is unexported; it is a documentation anchor, not part
// of the public contract. Adapters may inline the `make` call
// without depending on this symbol.
func newStreamBuffer() chan Event {
	return make(chan Event, DefaultBufferSize)
}

// validatePreStream implements the pre-stream branch of Stream. It
// returns ctx.Err() when ctx is already cancelled and
// req.Validate()'s error when req is structurally invalid. On any
// failure the producer goroutine is NOT started and no channel is
// allocated. Adapters MUST call this helper before allocating the
// producer channel so the pre-stream branch is mechanically identical
// across implementations.
//
// This helper is unexported; it is a documentation anchor, not part
// of the public contract. Adapters may inline the same two-line
// check without depending on this symbol.
func validatePreStream(ctx context.Context, req Request) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	return req.Validate()
}

// compileTimeOnlyStub is a zero-value compile-time assertion that
// any concrete type satisfying ModelProvider must end up method-set
// equivalent to the interface. The stub's body is intentionally
// trivial; the assertion itself is the contract.
//
// Drift in the ModelProvider interface (adding a method, renaming
// Stream, changing a parameter type) breaks the assertion line below
// at build time.
type compileTimeOnlyStub struct{}

// Stream is the compile-time-only stub satisfying ModelProvider. The
// body intentionally returns nil for both the channel and the error
// because no real producer runs in this stub — the assertion
// `var _ ModelProvider = (*compileTimeOnlyStub)(nil)` below is the
// contract, not the Stream method's body.
func (compileTimeOnlyStub) Stream(_ context.Context, _ Request) (<-chan Event, error) {
	return nil, nil
}

// This is the canonical compile-time assertion. Drift in the
// ModelProvider interface (adding a method, renaming Stream,
// changing a parameter type) breaks this line at build time. The
// apply phase places this in provider.go as a documented anchor; the
// consumer test (agenttest/consumer_test.go) places an equivalent
// assertion against its own stubProvider to prove the cross-package
// view.
var _ ModelProvider = (*compileTimeOnlyStub)(nil)
