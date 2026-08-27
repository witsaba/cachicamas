// Package chat_test — assistant_config_put_handler_test.go was the
// TDD contract for the CH-12.3 PUT handler that lived at
// /api/chat/assistant/config. PR-2 of cachicamas-archetype-system-foundation
// (T-20) retired that route group in favour of the polymorphic
// /api/archetypes/{slug}/config/ surface owned by the archetype
// package. PUT validation + Writer atomic semantics are now tested
// at backend/agent/src/archetype/http_test.go (T-18/T-19 PR-2).
//
// This file is kept as a single retirement assertion: a PUT against
// the old URL must return 404, never a re-routed 200. The companion
// GET test lives in assistant_config_handler_test.go.
package chat_test

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/labstack/echo/v5"
)

// Test_OldChatAssistantConfigPutRoute_Returns404 — T-20 PR-2
// (cachicamas-archetype-system-foundation). Defence in depth: a PUT
// against the retired /api/chat/assistant/config URL must return
// 404, not 200 (would indicate the route was accidentally
// re-introduced) and not 500 (handler error). Mirrors the GET
// retirement test in assistant_config_handler_test.go.
func Test_OldChatAssistantConfigPutRoute_Returns404(t *testing.T) {
	t.Parallel()

	e := echo.New()

	// Body shape matches the prior change's PUT contract so a
	// future accidental re-introduction that forwards the body
	// would not return 404 from a JSON-parse error.
	body := bytes.NewReader([]byte(`{
		"system_prompt": "you are a helpful assistant",
		"tool_allowlist": ["current_time"],
		"defer_tool_names": []
	}`))
	req := httptest.NewRequest(http.MethodPut, "/api/chat/assistant/config", body)
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want 404; body=%s", rec.Code, rec.Body.String())
	}
}
