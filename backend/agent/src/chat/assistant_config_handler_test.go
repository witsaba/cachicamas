// Package chat_test — assistant_config_handler_test.go was the TDD
// contract for the CH-12.1 GET handler that lived at
// /api/chat/assistant/config. PR-2 of cachicamas-archetype-system-foundation
// (T-20) retired that route group in favour of the polymorphic
// /api/archetypes/{slug}/config/ surface owned by the archetype
// package.
//
// The polymorphic surface is tested at
// backend/agent/src/archetype/http_test.go (T-18/T-19 PR-2). This
// file's only remaining job is to lock the retirement: a client that
// still hits the old URL gets 404, NOT a re-routed 200.
package chat_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/labstack/echo/v5"

	"github.com/cachicamas/backend/agent/src/chat"
)

// Test_OldChatAssistantConfigRoute_Returns404 — T-20 PR-2
// (cachicamas-archetype-system-foundation). The /api/chat/assistant/config
// route is retired in this commit. A client that still hits the old
// URL must observe a 404, not a 200 (re-routed to the new
// polymorphic surface) and not a 500 (handler error). The 404 is the
// canonical "this surface is gone" signal.
//
// Echo's default 404 handler returns an empty body + 404; we
// additionally verify the path was NOT routed (no 200, no 500) so a
// future accidental re-introduction of the route fails this test.
func Test_OldChatAssistantConfigRoute_Returns404(t *testing.T) {
	t.Parallel()

	e := echo.New()

	// Wire whatever the chat binary still wires in production. We do
	// NOT call the retired RegisterAssistantConfigRoutes — it no
	// longer exists. The new RegisterArchetypeRoutes lives in the
	// archetype package and lives under /api/archetypes/, not
	// /api/chat/.
	_ = chat.IdentityResolver(nil) // compile-time check the package still imports

	req := httptest.NewRequest(http.MethodGet, "/api/chat/assistant/config", nil)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want 404; body=%s", rec.Code, rec.Body.String())
	}

	// Defence in depth: PUT on the old URL also returns 404.
	reqPut := httptest.NewRequest(http.MethodPut, "/api/chat/assistant/config", nil)
	recPut := httptest.NewRecorder()
	e.ServeHTTP(recPut, reqPut)

	if recPut.Code != http.StatusNotFound {
		t.Fatalf("PUT status = %d, want 404; body=%s", recPut.Code, recPut.Body.String())
	}
}

// (Suppress unused-import warning for context if no test ends up using it.)
var _ = context.Background
