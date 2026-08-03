// AI-22.5 — the carrier view (V-STR-22): an iteration shape over a stream
// the consumer already holds. It never owns the stream and never closes it
// (design.md D5); it lives entirely in this package, a direct sibling of
// src/ai, so AI-20.4's signature guard — which parses only
// src/ai/provider.go — passes unmodified.

package agenttest

import (
	"context"
	"iter"

	"github.com/cachicamas/backend/agent/src/ai"
)

// Iter is a carrier view over a <-chan ai.Event the caller already holds
// (R-STK-011). It reads from that channel; it never closes it — a
// receive-only channel cannot even compile a close call — and it never
// takes ownership of the stream: the provider boundary keeps speaking the
// decided carrier untouched.
type Iter struct {
	ch  <-chan ai.Event
	err error
}

// NewIter builds a view over ch (R-STK-011). A nil ch is safe: [Iter.Events]
// yields nothing rather than panicking or blocking forever (NFR-STK-E).
func NewIter(ch <-chan ai.Event) *Iter {
	return &Iter{ch: ch}
}

// Events returns an [iter.Seq] yielding every event from the underlying
// stream, in order, until the channel closes, ctx is done, or the caller's
// range loop stops calling the yield function (R-STK-011, R-STK-012).
// Abandoning the view mid-iteration — breaking out of the range loop —
// leaves the stream in exactly the state the underlying [ai.ModelProvider]
// contract puts it in; the view adds no closing, cancelling or draining
// behavior of its own. A nil ctx is treated as [context.Background]
// (NFR-STK-E), matching this module's own nil-context convention.
func (it *Iter) Events(ctx context.Context) iter.Seq[ai.Event] {
	return func(yield func(ai.Event) bool) {
		if it.ch == nil {
			return
		}
		if ctx == nil {
			ctx = context.Background()
		}
		for {
			select {
			case ev, ok := <-it.ch:
				if !ok {
					return
				}
				if payload, isErrorEvent := ev.ErrorPayload(); isErrorEvent {
					it.err = payload
				}
				if !yield(ev) {
					return
				}
			case <-ctx.Done():
				if it.err == nil {
					it.err = ctx.Err()
				}
				return
			}
		}
	}
}

// Err reports the stream's terminal outcome after [Iter.Events]'s loop
// ends (R-STK-012): the terminal error event's *[ai.Failure], surfaced via
// [ai.Event.ErrorPayload], when the stream ended in one; otherwise ctx's
// own error when cancellation ended the loop instead; otherwise nil.
// Because range-over-func cannot itself return a value, this is how the
// error surfaces — a bufio.Scanner-style post-loop check. Calling Err
// before the loop has run at all reports nil, the zero value, rather than
// panicking (NFR-STK-E).
func (it *Iter) Err() error {
	return it.err
}
