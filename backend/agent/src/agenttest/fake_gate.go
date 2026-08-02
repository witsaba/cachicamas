// AI-21.4 — Gate: a two-channel synchronization point a scripted Hold step
// blocks its producer on, so a held stream is released by a test-controlled
// signal, never by an elapsed sleep (R-AFP-010).
//
// Reached is what removes the wall clock from a test's assertions: once it
// closes, the test knows the stream is open and blocked at the Hold, with
// no sleep required to find out. Release is idempotent and unblocks it —
// both closes are sync.Once-guarded, and one Gate belongs to one Hold step
// (design.md's Gate decision).
package agenttest

import "sync"

// Gate is a single-use synchronization point one Script's Hold step blocks
// its producer on, until the test releases it or the stream's context is
// cancelled.
//
// The zero Gate is not ready to use; construct one with NewGate.
type Gate struct {
	reached     chan struct{}
	released    chan struct{}
	reachedOnce sync.Once
	releaseOnce sync.Once
}

// NewGate constructs a ready-to-use Gate.
func NewGate() *Gate {
	return &Gate{
		reached:  make(chan struct{}),
		released: make(chan struct{}),
	}
}

// Reached returns a channel that closes the moment the producer arrives at
// this Gate's Hold step (R-AFP-010).
func (g *Gate) Reached() <-chan struct{} { return g.reached }

// Release unblocks the producer waiting at this Gate's Hold step. It is
// idempotent: calling it more than once, or before the producer ever
// arrives, is safe.
func (g *Gate) Release() {
	g.releaseOnce.Do(func() { close(g.released) })
}

// markReached signals that the producer has arrived at this Gate's Hold
// step. Unexported: only fake_provider.go's producer may signal it, and
// only once.
func (g *Gate) markReached() {
	g.reachedOnce.Do(func() { close(g.reached) })
}
