// Package chat_test — assistant_config_put_handler_test.go is the TDD
// contract for the AssistantConfig PUT handler (REQ-CACAPI-002/003,
// design AD-5). Seven scenarios from spec #4077:
//
//	- signed-in owner PUTs a valid body → 200 + new config
//	- oversized system_prompt → 400
//	- HTML pattern in system_prompt → 400
//	- unknown tool name in tool_allowlist → 400
//	- defer name not in allowlist → 400
//	- anonymous caller → 403
//	- successful PUT appends one audit-log row
//
// Uses a fake `archetype.Writer` so the suite is hermetic — no
// Postgres, no INTEGRATION gate.
package chat_test

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

	"github.com/labstack/echo/v5"

	"github.com/cachicamas/backend/agent/src/archetype"
	"github.com/cachicamas/backend/agent/src/chat"
)

// fakeWriter implements archetype.Writer by returning whatever was
// installed via withResult and recording every Append call.
type fakeWriter struct {
	mu      sync.Mutex
	result  archetype.ArchetypeConfig
	err     error
	appends int
}

func (f *fakeWriter) WriteConfig(_ context.Context, _ archetype.ArchetypeKind, _ string, _ archetype.ConfigUpdate, _ string) (archetype.ArchetypeConfig, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.appends++
	return f.result, f.err
}

// WithTx is part of the Writer interface; the handler never invokes it.
// Returning the same fake keeps the test hermetic.
func (f *fakeWriter) WithTx(_ *sql.Tx) archetype.Loader { return nil }

// putBody is the JSON shape the PUT handler accepts.
type putBody struct {
	SystemPrompt   string   `json:"system_prompt"`
	ToolAllowlist  []string `json:"tool_allowlist"`
	DeferToolNames []string `json:"defer_tool_names"`
}

func newPutRequest(t *testing.T, body putBody) *http.Request {
	t.Helper()
	raw, err := json.Marshal(body)
	if err != nil {
		t.Fatalf("json.Marshal: %v", err)
	}
	req := httptest.NewRequest(http.MethodPut, "/api/chat/assistant/config", bytes.NewReader(raw))
	req.Header.Set("Content-Type", "application/json")
	return req
}

// validPutBody returns a body that passes every server-side check.
func validPutBody() putBody {
	return putBody{
		SystemPrompt:   "you are a helpful assistant",
		ToolAllowlist:  []string{"current_time", "summarize_conversation"},
		DeferToolNames: []string{"summarize_conversation"},
	}
}

// Test_HandlePutAssistantConfig_ValidPersistsAndIncrementsVersion —
// REQ-CACAPI-002 / Scenario "valid PUT persists and increments
// version".
func Test_HandlePutAssistantConfig_ValidPersistsAndIncrementsVersion(t *testing.T) {
	t.Parallel()

	writer := &fakeWriter{
		result: archetype.ArchetypeConfig{
			Kind:           archetype.KindChat,
			OrgID:          "user_alice",
			SystemPrompt:   "you are a helpful assistant",
			ToolAllowlist:  []string{"current_time", "summarize_conversation"},
			DeferToolNames: []string{"summarize_conversation"},
			Version:        5,
		},
	}
	resolver := &fakeIdentityResolver{signIn: true, participant: "user_alice"}

	e := echo.New()
	if err := chat.RegisterAssistantConfigRoutes(e, resolver, nil, writer); err != nil {
		t.Fatalf("RegisterAssistantConfigRoutes: %v", err)
	}

	req := newPutRequest(t, validPutBody())
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body=%s", rec.Code, rec.Body.String())
	}
	if writer.appends != 1 {
		t.Errorf("WriteConfig called %d time(s), want 1", writer.appends)
	}

	var got archetype.ArchetypeConfig
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
		t.Fatalf("decode body: %v", err)
	}
	if got.Version != 5 {
		t.Errorf("Version = %d, want 5", got.Version)
	}
	if got.SystemPrompt != "you are a helpful assistant" {
		t.Errorf("SystemPrompt = %q, want %q", got.SystemPrompt, "you are a helpful assistant")
	}
}

// Test_HandlePutAssistantConfig_OversizedSystemPrompt_400 —
// REQ-CACAPI-002 / Scenario "oversized system_prompt is rejected".
func Test_HandlePutAssistantConfig_OversizedSystemPrompt_400(t *testing.T) {
	t.Parallel()

	writer := &fakeWriter{}
	resolver := &fakeIdentityResolver{signIn: true, participant: "user_alice"}

	e := echo.New()
	if err := chat.RegisterAssistantConfigRoutes(e, resolver, nil, writer); err != nil {
		t.Fatalf("RegisterAssistantConfigRoutes: %v", err)
	}

	oversized := strings.Repeat("a", 4001)
	body := validPutBody()
	body.SystemPrompt = oversized
	req := newPutRequest(t, body)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400; body=%s", rec.Code, rec.Body.String())
	}
	if writer.appends != 0 {
		t.Errorf("WriteConfig called %d time(s) on rejected body, want 0", writer.appends)
	}
}

// Test_HandlePutAssistantConfig_HTMLPattern_400 — REQ-CACAPI-002 /
// Scenario "HTML pattern in system_prompt is rejected".
func Test_HandlePutAssistantConfig_HTMLPattern_400(t *testing.T) {
	t.Parallel()

	writer := &fakeWriter{}
	resolver := &fakeIdentityResolver{signIn: true, participant: "user_alice"}

	e := echo.New()
	if err := chat.RegisterAssistantConfigRoutes(e, resolver, nil, writer); err != nil {
		t.Fatalf("RegisterAssistantConfigRoutes: %v", err)
	}

	body := validPutBody()
	body.SystemPrompt = "you are helpful <script>alert(1)</script>"
	req := newPutRequest(t, body)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400; body=%s", rec.Code, rec.Body.String())
	}
	if writer.appends != 0 {
		t.Errorf("WriteConfig called %d time(s) on rejected body, want 0", writer.appends)
	}
}

// Test_HandlePutAssistantConfig_UnknownToolName_400 — REQ-CACAPI-003 /
// Scenario "unknown tool name fails the whole request".
func Test_HandlePutAssistantConfig_UnknownToolName_400(t *testing.T) {
	t.Parallel()

	writer := &fakeWriter{}
	resolver := &fakeIdentityResolver{signIn: true, participant: "user_alice"}

	e := echo.New()
	if err := chat.RegisterAssistantConfigRoutes(e, resolver, nil, writer); err != nil {
		t.Fatalf("RegisterAssistantConfigRoutes: %v", err)
	}

	body := putBody{
		SystemPrompt:   "you are helpful",
		ToolAllowlist:  []string{"current_time", "no_such_tool"},
		DeferToolNames: []string{},
	}
	req := newPutRequest(t, body)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400; body=%s", rec.Code, rec.Body.String())
	}
	if writer.appends != 0 {
		t.Errorf("WriteConfig called %d time(s) on rejected body, want 0", writer.appends)
	}
}

// Test_HandlePutAssistantConfig_DeferNotInAllowlist_400 — defence in
// depth: a defer name not present in tool_allowlist is rejected.
func Test_HandlePutAssistantConfig_DeferNotInAllowlist_400(t *testing.T) {
	t.Parallel()

	writer := &fakeWriter{}
	resolver := &fakeIdentityResolver{signIn: true, participant: "user_alice"}

	e := echo.New()
	if err := chat.RegisterAssistantConfigRoutes(e, resolver, nil, writer); err != nil {
		t.Fatalf("RegisterAssistantConfigRoutes: %v", err)
	}

	body := putBody{
		SystemPrompt:   "you are helpful",
		ToolAllowlist:  []string{"current_time"},
		DeferToolNames: []string{"summarize_conversation"}, // not in allowlist
	}
	req := newPutRequest(t, body)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400; body=%s", rec.Code, rec.Body.String())
	}
	if writer.appends != 0 {
		t.Errorf("WriteConfig called %d time(s) on rejected body, want 0", writer.appends)
	}
}

// Test_HandlePutAssistantConfig_Anonymous_403 — defence in depth: an
// anonymous caller must NOT be able to PUT (no audit row, no row
// mutation).
func Test_HandlePutAssistantConfig_Anonymous_403(t *testing.T) {
	t.Parallel()

	writer := &fakeWriter{}
	resolver := &fakeIdentityResolver{signIn: false}

	e := echo.New()
	if err := chat.RegisterAssistantConfigRoutes(e, resolver, nil, writer); err != nil {
		t.Fatalf("RegisterAssistantConfigRoutes: %v", err)
	}

	req := newPutRequest(t, validPutBody())
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want 403; body=%s", rec.Code, rec.Body.String())
	}
	if writer.appends != 0 {
		t.Errorf("WriteConfig called %d time(s) on anonymous PUT, want 0", writer.appends)
	}
}

// Test_HandlePutAssistantConfig_WriterError_500 — defence in depth:
// when the Writer returns an error, the handler surfaces a 500 (no
// internal-error leak in the body).
func Test_HandlePutAssistantConfig_WriterError_500(t *testing.T) {
	t.Parallel()

	writer := &fakeWriter{err: errors.New("synthetic db failure")}
	resolver := &fakeIdentityResolver{signIn: true, participant: "user_alice"}

	e := echo.New()
	if err := chat.RegisterAssistantConfigRoutes(e, resolver, nil, writer); err != nil {
		t.Fatalf("RegisterAssistantConfigRoutes: %v", err)
	}

	req := newPutRequest(t, validPutBody())
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want 500; body=%s", rec.Code, rec.Body.String())
	}
	if strings.Contains(rec.Body.String(), "synthetic db failure") {
		t.Errorf("response body leaks the writer error string: %s", rec.Body.String())
	}
}
