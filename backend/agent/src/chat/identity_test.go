// CH-03.4 — IdentityResolver port (R-CHS-004, D1). The chat surface refuses
// any request whose IdentityResolver cannot resolve an Identity. Tests for
// the NoopIdentityResolver live in package chat_test (the archetype's own
// acceptance surface, matching every other chat test's package).

package chat_test

import (
	"context"
	"net/http/httptest"
	"testing"

	"github.com/cachicamas/backend/agent/src/chat"
)

// TestNoopIdentityResolver_ReturnsParticipant — S-CHS-004 context. Given a
// NoopIdentityResolver configured with a participant id, when its
// IdentityFromRequest method is called with any context and request, then it
// returns an Identity whose ParticipantID is the configured value AND the
// ok flag is true — so the HTTP handler proceeds to the registry check.
func TestNoopIdentityResolver_ReturnsParticipant(t *testing.T) {
	t.Parallel()

	resolver := chat.NoopIdentityResolver{Participant: "alice"}
	req := httptest.NewRequest("POST", "/api/agent/turns", nil)

	id, ok := resolver.IdentityFromRequest(context.Background(), req)
	if !ok {
		t.Fatal("NoopIdentityResolver.IdentityFromRequest returned ok=false, want true")
	}
	if id == nil {
		t.Fatal("NoopIdentityResolver.IdentityFromRequest returned nil Identity, want non-nil")
	}
	if got := id.ParticipantID(); got != "alice" {
		t.Errorf("NoopIdentityResolver.IdentityFromRequest returned ParticipantID=%q, want %q", got, "alice")
	}
}

// TestNoopIdentityResolver_EmptyParticipant_ReturnsFalse — defensive guard.
// Given a NoopIdentityResolver with an empty Participant, when its
// IdentityFromRequest is called, then it returns ok=false — so the HTTP
// handler emits the 401 "identity not resolved" envelope instead of
// synthesising a session for an empty participant id.
func TestNoopIdentityResolver_EmptyParticipant_ReturnsFalse(t *testing.T) {
	t.Parallel()

	resolver := chat.NoopIdentityResolver{Participant: ""}
	req := httptest.NewRequest("POST", "/api/agent/turns", nil)

	id, ok := resolver.IdentityFromRequest(context.Background(), req)
	if ok {
		t.Errorf("NoopIdentityResolver{Participant: \"\"} returned ok=true, want false (refusal for empty participant)")
	}
	if id != nil {
		t.Errorf("NoopIdentityResolver{Participant: \"\"} returned Identity=%#v, want nil (refusal for empty participant)", id)
	}
}
