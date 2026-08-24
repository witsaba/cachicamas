// CH-03 test helpers — small, production-safe adapters used by the
// chat_test package and any future white-box test that needs to drive
// the surface without a real auth implementation. CH-04 will replace
// the production IdentityResolver with a shim over the
// database_administrator's IdentityFromCookie; these helpers stay
// for tests that want a deterministic identity per request.
//
// None of these helpers read the environment, install observability,
// or cross a network boundary — they are the minimum surface a
// deterministic test needs to assert the cross-participant guard
// (S-CHS-004.b), the 401 refusal (S-CHS-004.a) and the open-turn
// happy path (S-CHS-001.a) without standing up a full identity chain.

package chat

import (
	"context"
	"net/http"
	"strings"
)

// HeaderParticipantResolver returns an IdentityResolver that
// resolves the participant id from the named request header
// (canonical: "X-Test-Participant"). An empty header value,
// a missing header, or a header value that is whitespace-only
// returns (nil, false), which the identity middleware maps to
// 401 server (S-CHS-004.a). The header name is parameterised
// so a future test can name a different header without colliding
// with the http library's reserved words.
func HeaderParticipantResolver(headerName string) IdentityResolver {
	return headerParticipantResolver(headerName)
}

type headerParticipantResolver string

// IdentityFromRequest satisfies the IdentityResolver interface:
// reads the configured header from the request and returns a
// headerParticipantIdentity carrying the trimmed value.
func (h headerParticipantResolver) IdentityFromRequest(_ context.Context, r *http.Request) (Identity, bool) {
	v := strings.TrimSpace(r.Header.Get(string(h)))
	if v == "" {
		return nil, false
	}
	return headerParticipantIdentity{id: v}, true
}

type headerParticipantIdentity struct{ id string }

func (i headerParticipantIdentity) ParticipantID() string { return i.id }
