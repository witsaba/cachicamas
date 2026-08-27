// Package archetype_test — http_test.go: TDD contract for the polymorphic
// HTTP handlers that back /api/archetypes/{slug}/config/.
//
// This file is T-18 (PR-2 of cachicamas-archetype-system-foundation).
// Eight scenarios per the spec's API-01..API-04 capability:
//   - GET  /api/archetypes/{slug}/config/ — known slug → 200
//   - GET  /api/archetypes/{slug}/config/ — unknown slug → 404
//   - GET  /api/archetypes/{slug}/config/ — anonymous → 403
//   - PUT  /api/archetypes/{slug}/config/ — valid body persists
//   - PUT  /api/archetypes/{slug}/config/ — <script in prompt → 400
//   - PUT  /api/archetypes/{slug}/config/ — 4001 chars → 400
//   - PUT  /api/archetypes/{slug}/config/ — every validation rule distinct
//   - PUT  /api/archetypes/{slug}/config/ — unknown slug → 404 (FK)
//
// The handlers do not exist yet when this file ships — that's the RED.
// Production code lands in T-19.
package archetype_test

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
	"time"

	"github.com/labstack/echo/v5"

	"github.com/cachicamas/backend/agent/src/archetype"
)

// -----------------------------------------------------------------------------
// Test doubles
// -----------------------------------------------------------------------------

// fakeCatalogLoader implements archetype.CatalogLoader by returning whatever
// was installed via withView. The handler never invokes WithTx, but the
// interface requires it; returning the same fake keeps the suite hermetic.
type fakeCatalogLoader struct {
	mu        sync.Mutex
	view      archetype.ArchetypeView
	found     bool
	err       error
	loads     int
	lastSlug  string
	lastOrgID string
}

func (f *fakeCatalogLoader) LoadBySlug(_ context.Context, slug, orgID string) (archetype.ArchetypeView, bool, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.loads++
	f.lastSlug = slug
	f.lastOrgID = orgID
	return f.view, f.found, f.err
}

func (f *fakeCatalogLoader) WithTx(_ *sql.Tx) archetype.CatalogLoader { return f }

// fakeWriter implements archetype.Writer. Records every WriteConfig call so
// the tests can assert the handler called it once and the right body fields.
type fakeWriter struct {
	mu         sync.Mutex
	result     archetype.ArchetypeConfig
	err        error
	appends    int
	lastSlug   string
	lastOrgID  string
	lastUpdate archetype.ConfigUpdate
}

func (f *fakeWriter) WriteConfig(_ context.Context, slug, orgID string, update archetype.ConfigUpdate, _ string) (archetype.ArchetypeConfig, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.appends++
	f.lastSlug = slug
	f.lastOrgID = orgID
	f.lastUpdate = update
	return f.result, f.err
}

// fakeResolver returns (orgID, true) when signIn is true; ("", false)
// otherwise. The handler interprets (false) as anonymous → 403.
type fakeResolver struct {
	signIn bool
	orgID  string
}

func (r *fakeResolver) IdentityFromRequest(_ context.Context, _ *http.Request) (archetype.Identity, bool) {
	if !r.signIn {
		return nil, false
	}
	return &fakeIdentity{orgID: r.orgID}, true
}

type fakeIdentity struct{ orgID string }

func (i *fakeIdentity) ParticipantID() string { return i.orgID }

// -----------------------------------------------------------------------------
// Test fixtures
// -----------------------------------------------------------------------------

// validPutBody returns a body that passes every server-side check
// (mirror of the chat-package test's validPutBody).
func validPutBody() map[string]any {
	return map[string]any{
		"system_prompt":    "you are a helpful assistant",
		"tool_allowlist":   []string{"current_time", "summarize_conversation"},
		"defer_tool_names": []string{"summarize_conversation"},
	}
}

func newPutRequest(t *testing.T, body map[string]any) *http.Request {
	t.Helper()
	raw, err := json.Marshal(body)
	if err != nil {
		t.Fatalf("json.Marshal: %v", err)
	}
	req := httptest.NewRequest(http.MethodPut, "/api/archetypes/assistant/config/", bytes.NewReader(raw))
	req.Header.Set("Content-Type", "application/json")
	return req
}

// -----------------------------------------------------------------------------
// GET scenarios
// -----------------------------------------------------------------------------

// Test_HandleGetArchetypeConfig_KnownSlug_200 — REQ-CASF-API-01 /
// "GET on a known system slug returns 200 with parent + child columns".
// Spec: API-01.
func Test_HandleGetArchetypeConfig_KnownSlug_200(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, 8, 27, 10, 0, 0, 0, time.UTC)
	view := archetype.ArchetypeView{
		Slug:        "assistant",
		Type:        "system",
		DisplayName: "Assistant",
		Tagline:     "Your default assistant",
		Status:      "active",
		CreatedAt:   now,
		CreatedBy:   "seed",
		Override: &archetype.ArchetypeOverride{
			SystemPrompt:   "you are the cachicamas assistant",
			ToolAllowlist:  []string{"current_time", "summarize_conversation"},
			DeferToolNames: []string{"summarize_conversation"},
			Version:        4,
			UpdatedAt:      now,
			UpdatedBy:      "user_alice",
		},
	}
	loader := &fakeCatalogLoader{view: view, found: true}
	resolver := &fakeResolver{signIn: true, orgID: "user_alice"}

	e := echo.New()
	if err := archetype.RegisterArchetypeRoutes(e, resolver, loader, nil); err != nil {
		t.Fatalf("RegisterArchetypeRoutes: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/api/archetypes/assistant/config/", nil)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body=%s", rec.Code, rec.Body.String())
	}

	var got archetype.ArchetypeView
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
		t.Fatalf("decode body: %v", err)
	}
	if got.Slug != "assistant" {
		t.Errorf("Slug = %q, want assistant", got.Slug)
	}
	if got.Type != "system" {
		t.Errorf("Type = %q, want system", got.Type)
	}
	if got.Override == nil {
		t.Fatal("Override = nil; want per-org override surfaced")
	}
	if got.Override.SystemPrompt != "you are the cachicamas assistant" {
		t.Errorf("Override.SystemPrompt = %q, want %q", got.Override.SystemPrompt, "you are the cachicamas assistant")
	}
	if got.Override.Version != 4 {
		t.Errorf("Override.Version = %d, want 4", got.Override.Version)
	}
	if loader.lastSlug != "assistant" {
		t.Errorf("Loader called with slug %q, want assistant", loader.lastSlug)
	}
	if loader.lastOrgID != "user_alice" {
		t.Errorf("Loader called with orgID %q, want user_alice", loader.lastOrgID)
	}
}

// Test_HandleGetArchetypeConfig_UnknownSlug_404 — REQ-CASF-API-01 /
// "GET on an unknown slug returns 404". Spec: API-01.
func Test_HandleGetArchetypeConfig_UnknownSlug_404(t *testing.T) {
	t.Parallel()

	// Loader returns found=false (the canonical "no row" signal maps
	// to 404 + ERR_UNKNOWN_SLUG per the spec's API-01 contract).
	loader := &fakeCatalogLoader{found: false}
	resolver := &fakeResolver{signIn: true, orgID: "user_alice"}

	e := echo.New()
	if err := archetype.RegisterArchetypeRoutes(e, resolver, loader, nil); err != nil {
		t.Fatalf("RegisterArchetypeRoutes: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/api/archetypes/no-such-slug/config/", nil)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want 404; body=%s", rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), "ERR_UNKNOWN_SLUG") {
		t.Errorf("body missing ERR_UNKNOWN_SLUG code: %s", rec.Body.String())
	}
}

// Test_HandleGetArchetypeConfig_Anonymous_403 — REQ-CASF-API-01 /
// "GET anonymous returns 403". Spec: API-01. Edge case 3 of spec §4.
func Test_HandleGetArchetypeConfig_Anonymous_403(t *testing.T) {
	t.Parallel()

	loader := &fakeCatalogLoader{}
	resolver := &fakeResolver{signIn: false}

	e := echo.New()
	if err := archetype.RegisterArchetypeRoutes(e, resolver, loader, nil); err != nil {
		t.Fatalf("RegisterArchetypeRoutes: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/api/archetypes/assistant/config/", nil)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want 403; body=%s", rec.Code, rec.Body.String())
	}
	if loader.loads != 0 {
		t.Errorf("Loader called %d time(s) on anonymous request, want 0", loader.loads)
	}
}

// -----------------------------------------------------------------------------
// PUT scenarios
// -----------------------------------------------------------------------------

// Test_HandlePutArchetypeConfig_ValidPersists_AppendsOneLogRow —
// REQ-CASF-API-02 / "PUT with a valid body increments version and
// appends one log row". Spec: API-02.
func Test_HandlePutArchetypeConfig_ValidPersists_AppendsOneLogRow(t *testing.T) {
	t.Parallel()

	writer := &fakeWriter{
		result: archetype.ArchetypeConfig{
			Slug:           archetype.AssistantSlug,
			OrgID:          "user_alice",
			SystemPrompt:   "you are a helpful assistant",
			ToolAllowlist:  []string{"current_time", "summarize_conversation"},
			DeferToolNames: []string{"summarize_conversation"},
			Version:        5,
		},
	}
	resolver := &fakeResolver{signIn: true, orgID: "user_alice"}

	e := echo.New()
	if err := archetype.RegisterArchetypeRoutes(e, resolver, nil, writer); err != nil {
		t.Fatalf("RegisterArchetypeRoutes: %v", err)
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
	if writer.lastSlug != "assistant" {
		t.Errorf("WriteConfig slug = %q, want assistant", writer.lastSlug)
	}
	if writer.lastOrgID != "user_alice" {
		t.Errorf("WriteConfig orgID = %q, want user_alice", writer.lastOrgID)
	}

	var got archetype.ArchetypeConfig
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
		t.Fatalf("decode body: %v", err)
	}
	if got.Version != 5 {
		t.Errorf("Version = %d, want 5 (server response, not request body)", got.Version)
	}
}

// Test_HandlePutArchetypeConfig_HTMLPattern_400 — REQ-CASF-API-02 /
// "PUT with <script returns 400 with ERR_HTML_PATTERN". Spec: API-02.
func Test_HandlePutArchetypeConfig_HTMLPattern_400(t *testing.T) {
	t.Parallel()

	writer := &fakeWriter{}
	resolver := &fakeResolver{signIn: true, orgID: "user_alice"}

	e := echo.New()
	if err := archetype.RegisterArchetypeRoutes(e, resolver, nil, writer); err != nil {
		t.Fatalf("RegisterArchetypeRoutes: %v", err)
	}

	body := validPutBody()
	body["system_prompt"] = "you are helpful <script>alert(1)</script>"
	req := newPutRequest(t, body)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400; body=%s", rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), "ERR_HTML_PATTERN") {
		t.Errorf("body missing ERR_HTML_PATTERN code: %s", rec.Body.String())
	}
	if writer.appends != 0 {
		t.Errorf("WriteConfig called %d time(s) on rejected body, want 0", writer.appends)
	}
}

// Test_HandlePutArchetypeConfig_OversizedPrompt_400 — REQ-CASF-API-02 /
// "PUT with 4001 chars returns 400 with ERR_PROMPT_TOO_LONG". Spec: API-02.
func Test_HandlePutArchetypeConfig_OversizedPrompt_400(t *testing.T) {
	t.Parallel()

	writer := &fakeWriter{}
	resolver := &fakeResolver{signIn: true, orgID: "user_alice"}

	e := echo.New()
	if err := archetype.RegisterArchetypeRoutes(e, resolver, nil, writer); err != nil {
		t.Fatalf("RegisterArchetypeRoutes: %v", err)
	}

	body := validPutBody()
	body["system_prompt"] = strings.Repeat("a", 4001)
	req := newPutRequest(t, body)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400; body=%s", rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), "ERR_PROMPT_TOO_LONG") {
		t.Errorf("body missing ERR_PROMPT_TOO_LONG code: %s", rec.Body.String())
	}
	if writer.appends != 0 {
		t.Errorf("WriteConfig called %d time(s) on rejected body, want 0", writer.appends)
	}
}

// Test_HandlePutArchetypeConfig_DistinctErrorCodes_PerRule —
// REQ-CASF-API-04 / "every validation rule rejects with a distinct code".
// Spec: API-04. Covers all six validation sentinels.
func Test_HandlePutArchetypeConfig_DistinctErrorCodes_PerRule(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name     string
		mutate   func(b map[string]any)
		wantCode string
	}{
		{
			name:     "empty prompt",
			mutate:   func(b map[string]any) { b["system_prompt"] = "" },
			wantCode: "ERR_PROMPT_EMPTY",
		},
		{
			name:     "oversized prompt",
			mutate:   func(b map[string]any) { b["system_prompt"] = strings.Repeat("a", 4001) },
			wantCode: "ERR_PROMPT_TOO_LONG",
		},
		{
			name:     "HTML pattern",
			mutate:   func(b map[string]any) { b["system_prompt"] = "you are <script>alert(1)</script>" },
			wantCode: "ERR_HTML_PATTERN",
		},
		{
			name:     "empty allowlist",
			mutate:   func(b map[string]any) { b["tool_allowlist"] = []string{} },
			wantCode: "ERR_ALLOWLIST_EMPTY",
		},
		{
			name: "defer not in allowlist",
			mutate: func(b map[string]any) {
				b["tool_allowlist"] = []string{"current_time"}
				b["defer_tool_names"] = []string{"summarize_conversation"}
			},
			wantCode: "ERR_DEFER_NOT_IN_ALLOWLIST",
		},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			writer := &fakeWriter{}
			resolver := &fakeResolver{signIn: true, orgID: "user_alice"}

			e := echo.New()
			if err := archetype.RegisterArchetypeRoutes(e, resolver, nil, writer); err != nil {
				t.Fatalf("RegisterArchetypeRoutes: %v", err)
			}

			body := validPutBody()
			tc.mutate(body)
			req := newPutRequest(t, body)
			rec := httptest.NewRecorder()
			e.ServeHTTP(rec, req)

			if rec.Code != http.StatusBadRequest {
				t.Fatalf("status = %d, want 400; body=%s", rec.Code, rec.Body.String())
			}
			if !strings.Contains(rec.Body.String(), tc.wantCode) {
				t.Errorf("body missing %s code: %s", tc.wantCode, rec.Body.String())
			}
			if writer.appends != 0 {
				t.Errorf("WriteConfig called %d time(s) on rejected body, want 0", writer.appends)
			}
		})
	}
}

// Test_HandlePutArchetypeConfig_UnknownSlug_FKViolation_404 —
// REQ-CASF-API-02 + spec edge case 4: PUT with an unknown slug surfaces
// 404 + ERR_UNKNOWN_SLUG. The Writer is the canonical place to surface
// the FK violation; the handler maps that to 404.
func Test_HandlePutArchetypeConfig_UnknownSlug_FKViolation_404(t *testing.T) {
	t.Parallel()

	writer := &fakeWriter{err: archetype.ErrUnknownArchetypeSlug}
	resolver := &fakeResolver{signIn: true, orgID: "user_alice"}

	e := echo.New()
	if err := archetype.RegisterArchetypeRoutes(e, resolver, nil, writer); err != nil {
		t.Fatalf("RegisterArchetypeRoutes: %v", err)
	}

	req := newPutRequest(t, validPutBody())
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want 404; body=%s", rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), "ERR_UNKNOWN_SLUG") {
		t.Errorf("body missing ERR_UNKNOWN_SLUG code: %s", rec.Body.String())
	}
}

// Test_RegisterArchetypeRoutes_RejectsNilLoaderAndWriter — defensive:
// the route registration must reject a (nil loader, nil writer) pair
// (mirrors the chat-package RegisterAssistantConfigRoutes contract).
func Test_RegisterArchetypeRoutes_RejectsNilLoaderAndWriter(t *testing.T) {
	t.Parallel()

	e := echo.New()
	if err := archetype.RegisterArchetypeRoutes(e, nil, nil, nil); err == nil {
		t.Fatal("expected error when resolver is nil, got nil")
	}
	if err := archetype.RegisterArchetypeRoutes(e, &fakeResolver{signIn: true, orgID: "u"}, nil, nil); err == nil {
		t.Fatal("expected error when both loader and writer are nil, got nil")
	}
}

// (small helper to silence unused-imports when iterating)
var _ = errors.New
