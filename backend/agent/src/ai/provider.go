// AI-20 — the provider interface: the one call every provider offers.
//
// This file declares ModelProvider (V-PRV-01, V-PRV-03), the sole point of
// contact between Layer 1 and a vendor adapter: a cancellable context and a
// normalized Request in, a normalized Event stream out. Every rule this
// method's own documentation states was decided upstream —
// ai-stream-lifecycle (AI-02.1) settled the carrier, ownership,
// cancellation, buffering and failure-delivery rules restated in substance
// below; ai-minimum-capabilities (AI-03.1) settled the discovery mechanism
// TokenCounter exercises. Nothing here re-argues either decision, and this
// file adds no capability, event kind or failure category of its own.
//
// # No vendor type on the boundary, mechanically pinned
//
// The declaring file imports only "context" (R-AMP-002): Request and Event
// are Layer 1's own normalized types, and a vendor SDK type never appears in
// this signature, directly or nested inside a named type reachable from it.
// src/agenttest's signature guard resolves and parses this file to hold that
// pin as a build-time failure rather than a review convention (R-AMP-014).

package ai

import "context"

// ModelProvider is the one call every provider offers (V-PRV-01): a
// cancellable context and a normalized [Request] in, a receive-only stream
// of normalized [Event] values and an error out (V-PRV-03). There is no
// second streaming variant, no non-streaming convenience method and no
// capability-query method on this interface — an optional capability is a
// separate, additional contract asserted on the provider value at the point
// of use (see [TokenCounter]), never a widening of this one (R-AMP-001,
// R-AMP-021).
//
// # The eight ownership rules (ai-stream-lifecycle §§ 3–9)
//
// Restated here in substance, because a consumer reads this method's
// documentation rather than the design record that argued them:
//
//  1. The provider creates the stream and closes it exactly once. One
//     goroutine sends; one closing site exists; it runs on every exit path,
//     after the last send attempt. Nothing else closes the stream — not the
//     caller, not a test helper, not a consumer above Layer 1.
//  2. The caller owns ctx, the cancellable context, and the stream's
//     lifetime is bounded by it: every send waits on both the stream and
//     cancellation, so the stream is always cancellable, including while
//     its buffer is full.
//  3. A caller ends a stream in exactly one of two ways: it drains the
//     stream to its close, or it cancels ctx.
//  4. Abandoning a stream without cancelling ctx is a contract violation,
//     not a supported mode. It is stated rather than enforced — no test
//     proves a goroutine never exits — so a caller that will not drain MUST
//     cancel.
//  5. Cancellation closes the stream within bounded time: once cancellation
//     is observable, the provider begins no new blocking wait on the
//     network or on the caller, and any backoff waits on ctx rather than
//     sleeping.
//  6. The stream's buffer is bounded. Backpressure means waiting, never
//     dropping: a full buffer makes the provider wait, on the same terms as
//     rule 2.
//  7. Exactly one loss path is sanctioned: cancellation racing a saturated
//     buffer drops the late events and closes the stream with no terminal
//     event. A caller that treats a missing terminal after its OWN
//     cancellation as corruption is the party in error; a stream that
//     closes with no terminal and was never cancelled is a provider defect,
//     not a second loss path.
//  8. A [Request] that never becomes a stream reports its failure directly:
//     no stream and no producer goroutine are created (see the pre-stream
//     contract below). A stream that fails after being handed over reports
//     through its terminal error event instead, including when no content
//     preceded the failure.
//
// # Pre-stream contract (V-PRV-04)
//
// Stream validates req exactly once, before any I/O, and before it
// consults ctx: an invalid — including a never-constructed, zero-value —
// Request fails here with a typed validation failure, and neither the
// carrier nor the producer goroutine is created. Only once req is valid
// does an already-cancelled ctx short-circuit the same way, reported as
// AI-19's cancellation category. Nothing is observable before validation
// passes: no I/O, no stream allocation, no goroutine.
//
// # Mid-stream contract (V-PRV-05)
//
// Once handed over, the stream ends with exactly one terminal event — a
// normal completion, or the single terminal error event carrying AI-19's
// failure — the rule 7 sanctioned loss path aside.
//
// # Optional capabilities are separate and enumerable
//
// A capability beyond this method — token counting, today (see
// [TokenCounter]) — is asked of the provider value through its own,
// separately declared contract, never discovered from this interface or
// from a catalog. Every optional contract this package declares is
// enumerable in source, so a consumer always knows where to look for what a
// provider may additionally offer.
type ModelProvider interface {
	// Stream sends req to the provider and returns the normalized events it
	// answers with, honoring every rule stated above.
	Stream(ctx context.Context, req Request) (<-chan Event, error)
}

// TokenCounter is the one askable optional capability this package declares
// in v1 (CAP-O-02, V-PRV-17): whether a provider can report how many tokens
// a [Request] would consume, without sending it.
//
// # Advertised only by satisfying it
//
// A provider advertises TokenCounter by implementing it, and by no other
// means: no field, no flag, no registration call, no catalog entry
// (ai-minimum-capabilities §9). A consumer asks the provider VALUE, never a
// model identity or a configuration entry:
//
//	counter, ok := provider.(TokenCounter)
//
// ok == false is a clean absence (R-AMP-018): not an error, not a zero
// value standing in for one, and this package supplies no substitute,
// estimate or default. A provider that satisfies this contract and then
// declines to answer is non-conformant, not absent (R-AMP-019) —
// advertising binds.
//
// # One contract, never widened into the core interface
//
// This is the only optional contract this package declares in v1
// (R-AMP-017): one contract per capability, never an aggregate covering
// several. A provider implementing only [ModelProvider] and advertising
// nothing optional is fully conformant (R-AMP-020). Folding this method
// into [ModelProvider] itself is the exact widening R-AMP-021 pins the
// signature guard against — src/agenttest's guard fails the moment
// ModelProvider declares a second method.
type TokenCounter interface {
	// CountTokens reports how many tokens req would consume, without
	// sending req to the provider.
	CountTokens(ctx context.Context, req Request) (TokenCount, error)
}
