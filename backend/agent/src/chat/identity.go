// CH-03.4 — the identity port (R-CHS-004, D1). The chat archetype's HTTP
// surface refuses any request whose IdentityResolver cannot resolve an
// Identity. The interface is intentionally narrow: Identity carries one
// method (ParticipantID), and IdentityResolver carries one method
// (IdentityFromRequest). Production wiring (CH-04) plugs in a shim that
// reaches database_administrator's IdentityFromCookie; CH-03 ships the
// port and a NoopIdentityResolver for tests, no production auth anywhere
// below the composition root.

package chat

import (
	"context"
	"net/http"
)

// Identity is the opaque value the chat surface derives from a request. It
// is the chat archetype's own type — production callers (CH-04) construct
// it from database_administrator/src/interfaces/http/auth_middleware.go's
// domain.Identity via a thin shim, so this package never imports
// database_administrator (R-07).
type Identity interface {
	// ParticipantID is the stable, archetype-local id that keys the
	// Conversation registry (CH-03.1). It MUST NOT be empty for the
	// HTTP surface to proceed — NoopIdentityResolver and the production
	// shim both enforce that.
	ParticipantID() string
}

// IdentityResolver is the port the HTTP surface calls once per request to
// translate it into an Identity. Returns (Identity, true) to proceed,
// (nil, false) — or (nil, true), never observed but defensively treated as
// a refusal — to be refused with the locked 401 "identity not resolved"
// envelope (R-CHS-004.a).
//
// IdentityFromRequest MUST be safe to call from any goroutine: the surface
// invokes it once per request, on the request handler's goroutine, with the
// request's own context.
type IdentityResolver interface {
	IdentityFromRequest(ctx context.Context, r *http.Request) (Identity, bool)
}

// NoopIdentityResolver is the test-only resolver: every request resolves
// to a single configured participant id. An empty Participant is treated
// as "no identity" (returns false) so a test that builds a
// NoopIdentityResolver{} still drives the 401 path, not a phantom session.
type NoopIdentityResolver struct {
	Participant string
}

// IdentityFromRequest returns (noopIdentity{Participant}, true) when
// Participant is non-empty, and (nil, false) otherwise. The false branch
// is the test-time stand-in for the production path's "cookie missing or
// expired" rejection — no synthesised identity, no registry entry.
func (n NoopIdentityResolver) IdentityFromRequest(_ context.Context, _ *http.Request) (Identity, bool) {
	if n.Participant == "" {
		return nil, false
	}
	return noopIdentity{participant: n.Participant}, true
}

// noopIdentity is the Identity implementation NoopIdentityResolver hands
// out. Unexported because no production code outside this package ever
// constructs it.
type noopIdentity struct {
	participant string
}

// ParticipantID returns the configured participant id. Never empty when
// produced by NoopIdentityResolver.IdentityFromRequest's ok=true branch
// (the empty case short-circuits to false there).
func (n noopIdentity) ParticipantID() string { return n.participant }
