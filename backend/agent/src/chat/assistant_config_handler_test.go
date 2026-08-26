// Package chat_test — assistant_config_handler_test.go is the TDD
// contract for the AssistantConfig GET handler (REQ-CACAPI-001, design
// AD-4). Covers two scenarios from spec #4077:
//
//	- signed-in owner reads the config (200 + JSON body)
//	- anonymous caller is rejected (403)
//
// Plus a defence-in-depth case for loader errors.
//
// v1 scope key simplification: the chat Identity interface exposes only
// ParticipantID. The AssistantConfig row is keyed by org_id, and the
// v1 mapping is ParticipantID == orgID — a single-user workspace. The
// resolveParticipantIDToOrgID seam is intentionally deferred; a future
// multi-org workspace would add an OrgID method to the Identity
// interface or a lookup helper. This is documented in the handler
// itself and in tasks.md's "Open disagreements" section.
//
// Uses a fake `Loader` implementation (`fakeConfigLoader`) so the tests
// are hermetic — no Postgres, no INTEGRATION gate.
//
// RED at T-04: chat.HandleGetAssistantConfig and
// chat.RegisterAssistantConfigRoutes are undefined. GREEN (T-05) adds
// them.
package chat_test

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/labstack/echo/v5"

	"github.com/cachicamas/backend/agent/src/chat"
)

// fakeConfigLoader implements chat.Loader by returning whatever was
// installed via withResult. Used to drive the handler tests
// deterministically without spinning up Postgres.
type fakeConfigLoader struct {
	withResult chat.AssistantConfig
	withFound  bool
	withErr    error
}

func (f *fakeConfigLoader) LoadByOrg(_ context.Context, _ string) (chat.AssistantConfig, bool, error) {
	return f.withResult, f.withFound, f.withErr
}

// WithTx is part of the Loader interface; the handler never invokes it.
// Returning the same fake keeps the test hermetic.
func (f *fakeConfigLoader) WithTx(_ *sql.Tx) chat.Loader { return f }

// fakeIdentityResolver implements chat.IdentityResolver. When `signIn`
// is true it returns a synthetic Identity whose ParticipantID is also
// the orgID (v1 simplification); when false it returns (nil, false),
// which the handler interprets as anonymous and rejects with 403.
type fakeIdentityResolver struct {
	signIn      bool
	participant string
}

func (r *fakeIdentityResolver) IdentityFromRequest(_ context.Context, _ *http.Request) (chat.Identity, bool) {
	if !r.signIn {
		return nil, false
	}
	return &fakeIdentity{participantID: r.participant}, true
}

type fakeIdentity struct{ participantID string }

func (i *fakeIdentity) ParticipantID() string { return i.participantID }

// Test_HandleGetAssistantConfig_SignedInOwner — REQ-CACAPI-001 /
// Scenario "signed-in owner reads the config". Given a signed-in
// session, when GET /api/chat/assistant/config is called, then the
// response is 200 with the config JSON.
//
// RED at T-04: chat.HandleGetAssistantConfig is undefined.
func Test_HandleGetAssistantConfig_SignedInOwner(t *testing.T) {
	t.Parallel()

	expected := chat.DefaultAssistantConfig("user_alice", []string{"current_time", "summarize_conversation"})
	loader := &fakeConfigLoader{withResult: expected, withFound: true}
	resolver := &fakeIdentityResolver{signIn: true, participant: "user_alice"}

	e := echo.New()
	if err := chat.RegisterAssistantConfigRoutes(e, resolver, loader); err != nil {
		t.Fatalf("RegisterAssistantConfigRoutes: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/api/chat/assistant/config", nil)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body=%s", rec.Code, rec.Body.String())
	}

	var got chat.AssistantConfig
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
		t.Fatalf("decode body: %v", err)
	}
	if got.OrgID != expected.OrgID {
		t.Errorf("OrgID = %q, want %q", got.OrgID, expected.OrgID)
	}
	if got.SystemPrompt != expected.SystemPrompt {
		t.Errorf("SystemPrompt = %q, want %q", got.SystemPrompt, expected.SystemPrompt)
	}
	if got.Version != expected.Version {
		t.Errorf("Version = %d, want %d", got.Version, expected.Version)
	}
	if len(got.ToolAllowlist) != len(expected.ToolAllowlist) {
		t.Errorf("ToolAllowlist len = %d, want %d", len(got.ToolAllowlist), len(expected.ToolAllowlist))
	}
}

// Test_HandleGetAssistantConfig_Anonymous — REQ-CACAPI-001 / Scenario
// "anonymous caller is rejected". Given no signed-in session, when
// GET /api/chat/assistant/config is called, then the response is 403.
//
// RED at T-04: chat.HandleGetAssistantConfig is undefined.
func Test_HandleGetAssistantConfig_Anonymous(t *testing.T) {
	t.Parallel()

	loader := &fakeConfigLoader{}
	resolver := &fakeIdentityResolver{signIn: false}

	e := echo.New()
	if err := chat.RegisterAssistantConfigRoutes(e, resolver, loader); err != nil {
		t.Fatalf("RegisterAssistantConfigRoutes: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/api/chat/assistant/config", nil)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want 403; body=%s", rec.Code, rec.Body.String())
	}
}

// Test_HandleGetAssistantConfig_LoaderError — defence in depth: when
// the Loader returns a non-NoRows error, the handler surfaces a 500
// instead of leaking the error to the client body. The envelope format
// matches writeError's existing shape (kind=server).
//
// RED at T-04.
func Test_HandleGetAssistantConfig_LoaderError(t *testing.T) {
	t.Parallel()

	loader := &fakeConfigLoader{withErr: errors.New("synthetic db failure")}
	resolver := &fakeIdentityResolver{signIn: true, participant: "user_alice"}

	e := echo.New()
	if err := chat.RegisterAssistantConfigRoutes(e, resolver, loader); err != nil {
		t.Fatalf("RegisterAssistantConfigRoutes: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/api/chat/assistant/config", nil)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want 500; body=%s", rec.Code, rec.Body.String())
	}
	// The body must not echo the underlying error string.
	if strings.Contains(rec.Body.String(), "synthetic db failure") {
		t.Errorf("response body leaks the loader error string: %s", rec.Body.String())
	}
}
