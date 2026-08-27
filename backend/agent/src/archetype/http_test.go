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
//
// ListByType returns the pre-configured listByTypeViews slice (or
// listByTypeErr) so the list-handler tests can assert against a known
// shape. The list-handler tests install the slice directly; the
// per-slug tests do not touch it.
type fakeCatalogLoader struct {
	mu              sync.Mutex
	view            archetype.ArchetypeView
	found           bool
	err             error
	loads           int
	lastSlug        string
	lastOrgID       string
	listByTypeViews []archetype.ArchetypeView
	listByTypeErr   error
	listByTypeCalls int
}

func (f *fakeCatalogLoader) LoadBySlug(_ context.Context, slug, orgID string) (archetype.ArchetypeView, bool, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.loads++
	f.lastSlug = slug
	f.lastOrgID = orgID
	return f.view, f.found, f.err
}

func (f *fakeCatalogLoader) ListByType(_ context.Context, _ string) ([]archetype.ArchetypeView, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.listByTypeCalls++
	return f.listByTypeViews, f.listByTypeErr
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

// -----------------------------------------------------------------------------
// feat/archetype-list-endpoint (slice 3 — RED) — TDD contract for
// GET /api/archetypes (the directory list).
//
// Five scenarios per the spec's directory-list contract:
//   - 200 + sorted list of three types (system, general, owned) with
//     per-org override surfaced on the system one
//   - 200 + [] when the loader returns an empty list (NOT 404 — empty
//     directory is a valid state)
//   - 403 + error envelope on anonymous callers (same shape as the
//     per-slug arm's auth refusal)
//   - 500 + server envelope when the loader errors
//   - the list arm must mount even when the writer is nil
//     (RegisterArchetypeRoutes(e, resolver, loader, nil) still
//     exposes GET /api/archetypes)
//
// RED state: these tests reference archetype.HandleListArchetypes,
// which does not exist yet. The file fails to compile, which is the
// canonical RED for a new exported handler. The slice 4 feat commit
// adds the handler + the GET /api/archetypes route registration.
// -----------------------------------------------------------------------------

// Test_HandleListArchetypes_Authed_ReturnsAllTypes — three rows
// (system, general, owned), per-org override on the system one →
// 200 + the list shape, with the override surfaced and the list
// sorted by (type, slug) per the contract.
func Test_HandleListArchetypes_Authed_ReturnsAllTypes(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, 8, 27, 10, 0, 0, 0, time.UTC)
	views := []archetype.ArchetypeView{
		{
			Slug:        "general-one",
			Type:        "general",
			DisplayName: "General One",
			Tagline:     "First general archetype",
			Status:      "active",
			CreatedAt:   now,
			CreatedBy:   "seed",
		},
		{
			Slug:        "owned-one",
			Type:        "owned",
			DisplayName: "Owned One",
			Tagline:     "First owned archetype",
			Status:      "active",
			CreatedAt:   now,
			CreatedBy:   "seed",
		},
		{
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
		},
	}
	loader := &fakeCatalogLoader{listByTypeViews: views}
	resolver := &fakeResolver{signIn: true, orgID: "user_alice"}

	e := echo.New()
	if err := archetype.RegisterArchetypeRoutes(e, resolver, loader, nil); err != nil {
		t.Fatalf("RegisterArchetypeRoutes: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/api/archetypes", nil)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body=%s", rec.Code, rec.Body.String())
	}
	// Body is a JSON array (the spec's "re-check" correction — the
	// original "archetypes" envelope was a misleading example; the
	// Go side returns []ArchetypeView directly to match the TS
	// client's getJson<readonly ArchetypeView[]>() return type).
	var got []archetype.ArchetypeView
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
		t.Fatalf("decode body: %v; body=%s", err, rec.Body.String())
	}
	if len(got) != 3 {
		t.Fatalf("len(got) = %d, want 3; got=%+v", len(got), got)
	}
	// Sorted by (type, slug): general < owned < system.
	wantOrder := []string{"general-one", "owned-one", "assistant"}
	for i, want := range wantOrder {
		if got[i].Slug != want {
			t.Errorf("got[%d].Slug = %q, want %q", i, got[i].Slug, want)
		}
	}
	// Per-org override surfaced on the system row.
	assistant := got[2]
	if assistant.Override == nil {
		t.Fatal("assistant.Override = nil; want per-org override surfaced")
	}
	if assistant.Override.SystemPrompt != "you are the cachicamas assistant" {
		t.Errorf("assistant.Override.SystemPrompt = %q", assistant.Override.SystemPrompt)
	}
	if assistant.Override.Version != 4 {
		t.Errorf("assistant.Override.Version = %d, want 4", assistant.Override.Version)
	}
	if loader.listByTypeCalls != 1 {
		t.Errorf("ListByType called %d time(s), want 1", loader.listByTypeCalls)
	}
}

// Test_HandleListArchetypes_Empty_Returns200AndEmptyArray — loader
// returns nil/empty → 200 + [] (NOT 404). Empty directory is a
// valid state; the handler must not mistake "no rows" for "no
// route".
func Test_HandleListArchetypes_Empty_Returns200AndEmptyArray(t *testing.T) {
	t.Parallel()

	loader := &fakeCatalogLoader{listByTypeViews: nil}
	resolver := &fakeResolver{signIn: true, orgID: "user_alice"}

	e := echo.New()
	if err := archetype.RegisterArchetypeRoutes(e, resolver, loader, nil); err != nil {
		t.Fatalf("RegisterArchetypeRoutes: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/api/archetypes", nil)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body=%s", rec.Code, rec.Body.String())
	}
	body := strings.TrimSpace(rec.Body.String())
	if body != "[]" && body != "null" {
		t.Errorf("body = %q, want [] or null (empty JSON array)", body)
	}
}

// Test_HandleListArchetypes_Anonymous_Returns403 — resolver returns
// (nil, false) → 403 + error envelope. The Loader MUST NOT be
// called for anonymous callers (information leak prevention; same
// shape as the per-slug arm's anonymous refusal).
func Test_HandleListArchetypes_Anonymous_Returns403(t *testing.T) {
	t.Parallel()

	loader := &fakeCatalogLoader{listByTypeViews: []archetype.ArchetypeView{}}
	resolver := &fakeResolver{signIn: false}

	e := echo.New()
	if err := archetype.RegisterArchetypeRoutes(e, resolver, loader, nil); err != nil {
		t.Fatalf("RegisterArchetypeRoutes: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/api/archetypes", nil)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want 403; body=%s", rec.Code, rec.Body.String())
	}
	if loader.listByTypeCalls != 0 {
		t.Errorf("ListByType called %d time(s) on anonymous request, want 0", loader.listByTypeCalls)
	}
	// Body shape mirrors the per-slug 403 envelope.
	var env map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &env); err != nil {
		t.Fatalf("decode body: %v; body=%s", err, rec.Body.String())
	}
	if env["kind"] != "not_found" {
		t.Errorf("env.kind = %v, want not_found", env["kind"])
	}
	if env["message"] == nil || env["message"] == "" {
		t.Error("env.message missing; want non-empty string")
	}
}

// Test_HandleListArchetypes_LoaderError_Returns500 — loader returns
// a non-nil error → 500 + server envelope. The handler maps
// Loader errors to 500 (the error string is NOT echoed in the body
// to prevent information leaks; same discipline as the per-slug
// arm).
func Test_HandleListArchetypes_LoaderError_Returns500(t *testing.T) {
	t.Parallel()

	loader := &fakeCatalogLoader{listByTypeErr: errors.New("postgres exploded")}
	resolver := &fakeResolver{signIn: true, orgID: "user_alice"}

	e := echo.New()
	if err := archetype.RegisterArchetypeRoutes(e, resolver, loader, nil); err != nil {
		t.Fatalf("RegisterArchetypeRoutes: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/api/archetypes", nil)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want 500; body=%s", rec.Code, rec.Body.String())
	}
	var env map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &env); err != nil {
		t.Fatalf("decode body: %v; body=%s", err, rec.Body.String())
	}
	if env["kind"] != "server" {
		t.Errorf("env.kind = %v, want server", env["kind"])
	}
}

// Test_HandleListArchetypes_OnlyLoaderRequired — the list arm must
// mount even when the writer is nil. The list is read-only; it does
// not need a Writer. RegisterArchetypeRoutes(e, resolver, loader,
// nil) must still expose GET /api/archetypes (otherwise the GET
// surface would be hidden behind the PUT arm's writer
// availability, which is wrong).
func Test_HandleListArchetypes_OnlyLoaderRequired(t *testing.T) {
	t.Parallel()

	loader := &fakeCatalogLoader{listByTypeViews: []archetype.ArchetypeView{}}
	resolver := &fakeResolver{signIn: true, orgID: "user_alice"}

	e := echo.New()
	// Writer is nil — the per-slug PUT arm is skipped, but the
	// list arm must still register.
	if err := archetype.RegisterArchetypeRoutes(e, resolver, loader, nil); err != nil {
		t.Fatalf("RegisterArchetypeRoutes: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/api/archetypes", nil)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body=%s", rec.Code, rec.Body.String())
	}
	if loader.listByTypeCalls != 1 {
		t.Errorf("ListByType called %d time(s), want 1", loader.listByTypeCalls)
	}
}

    // -----------------------------------------------------------------------------
    // cachicamas-archetype-per-slug-overlay (RED — T-01..T-08) — TDD contract
    // for the new GET /api/archetypes/:slug handler (REQ-APSO-1..8).
    //
    // Eight scenarios per the spec's APSO scenarios:
    //   - Known slug + no override row → 200 + JSON with parent fields + NO override key
    //   - Known slug + override row → 200 + JSON WITHOUT override key (strip observable)
    //   - Unknown slug → 404 + ERR_UNKNOWN_SLUG
    //   - Anonymous → 403 + standard envelope, loader NOT called
    //   - Loader error → 500 + server envelope, no info leak
    //   - Empty slug path → 400 + validation envelope, loader NOT called
    //   - Registration discipline (loader=nil → new arm NOT mounted)
    //   - Route ordering: /:slug and /:slug/config/ resolve disjointly
    //
    // RED state: these tests reference archetype.HandleGetArchetype, which
    // does not exist yet. The file fails to compile, which is the canonical
    // RED for a new exported handler. The Commit-2 GREEN commit adds the
    // handler + the GET /api/archetypes/:slug route registration.
    // -----------------------------------------------------------------------------

    // Test_HandleGetArchetype_KnownSlug_200_NoOverride — REQ-APSO-1 +
    // REQ-APSO-2 / Scenario 2: loader returns a view WITHOUT Override →
    // 200 + JSON with the parent fields populated + NO override key.
    func Test_HandleGetArchetype_KnownSlug_200_NoOverride(t *testing.T) {
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
    	}
    	loader := &fakeCatalogLoader{view: view, found: true}
    	resolver := &fakeResolver{signIn: true, orgID: "user_alice"}

    	e := echo.New()
    	if err := archetype.RegisterArchetypeRoutes(e, resolver, loader, nil); err != nil {
    		t.Fatalf("RegisterArchetypeRoutes: %v", err)
    	}

    	req := httptest.NewRequest(http.MethodGet, "/api/archetypes/assistant", nil)
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
    	if got.DisplayName != "Assistant" {
    		t.Errorf("DisplayName = %q, want Assistant", got.DisplayName)
    	}
    	if got.Override != nil {
    		t.Errorf("Override = %+v, want nil", got.Override)
    	}
    	// The wire body MUST NOT contain the override key (even when
    	// view.Override is nil, the json:"override,omitempty" tag drops it).
    	if strings.Contains(rec.Body.String(), `"override"`) {
    		t.Errorf("body contains override key: %s", rec.Body.String())
    	}
    	if loader.lastSlug != "assistant" {
    		t.Errorf("loader called with slug %q, want assistant", loader.lastSlug)
    	}
    	if loader.lastOrgID != "user_alice" {
    		t.Errorf("loader called with orgID %q, want user_alice", loader.lastOrgID)
    	}
    }

    // Test_HandleGetArchetype_KnownSlug_WithOverrideRow_StillStripsOverride —
    // REQ-APSO-2 / Scenario 3: loader returns a view WITH Override
    // populated → 200 + JSON WITHOUT override key (strip step is
    // observable even when the underlying view has one).
    func Test_HandleGetArchetype_KnownSlug_WithOverrideRow_StillStripsOverride(t *testing.T) {
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

    	req := httptest.NewRequest(http.MethodGet, "/api/archetypes/assistant", nil)
    	rec := httptest.NewRecorder()
    	e.ServeHTTP(rec, req)

    	if rec.Code != http.StatusOK {
    		t.Fatalf("status = %d, want 200; body=%s", rec.Code, rec.Body.String())
    	}
    	// Strip is the contract: body MUST NOT carry an override key.
    	if strings.Contains(rec.Body.String(), `"override"`) {
    		t.Errorf("body contains override key (strip step failed): %s", rec.Body.String())
    	}
    	var got map[string]any
    	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
    		t.Fatalf("decode body: %v", err)
    	}
    	if _, ok := got["override"]; ok {
    		t.Errorf("body has override key: %v", got["override"])
    	}
    	if got["slug"] != "assistant" {
    		t.Errorf("slug = %v, want assistant", got["slug"])
    	}
    }

    // Test_HandleGetArchetype_UnknownSlug_404 — REQ-APSO-4 / Scenario 4:
    // loader returns found=false → 404 + ERR_UNKNOWN_SLUG envelope.
    func Test_HandleGetArchetype_UnknownSlug_404(t *testing.T) {
    	t.Parallel()

    	loader := &fakeCatalogLoader{found: false}
    	resolver := &fakeResolver{signIn: true, orgID: "user_alice"}

    	e := echo.New()
    	if err := archetype.RegisterArchetypeRoutes(e, resolver, loader, nil); err != nil {
    		t.Fatalf("RegisterArchetypeRoutes: %v", err)
    	}

    	req := httptest.NewRequest(http.MethodGet, "/api/archetypes/no-such-slug", nil)
    	rec := httptest.NewRecorder()
    	e.ServeHTTP(rec, req)

    	if rec.Code != http.StatusNotFound {
    		t.Fatalf("status = %d, want 404; body=%s", rec.Code, rec.Body.String())
    	}
    	if !strings.Contains(rec.Body.String(), "ERR_UNKNOWN_SLUG") {
    		t.Errorf("body missing ERR_UNKNOWN_SLUG code: %s", rec.Body.String())
    	}
    	if !strings.Contains(rec.Body.String(), "archetype slug is not registered") {
    		t.Errorf("body missing standard 404 message: %s", rec.Body.String())
    	}
    }

    // Test_HandleGetArchetype_Anonymous_403 — REQ-APSO-3 / Scenario 1:
    // resolver returns (nil, false) → 403 + standard envelope, loader NOT
    // called.
    func Test_HandleGetArchetype_Anonymous_403(t *testing.T) {
    	t.Parallel()

    	loader := &fakeCatalogLoader{}
    	resolver := &fakeResolver{signIn: false}

    	e := echo.New()
    	if err := archetype.RegisterArchetypeRoutes(e, resolver, loader, nil); err != nil {
    		t.Fatalf("RegisterArchetypeRoutes: %v", err)
    	}

    	req := httptest.NewRequest(http.MethodGet, "/api/archetypes/assistant", nil)
    	rec := httptest.NewRecorder()
    	e.ServeHTTP(rec, req)

    	if rec.Code != http.StatusForbidden {
    		t.Fatalf("status = %d, want 403; body=%s", rec.Code, rec.Body.String())
    	}
    	if loader.loads != 0 {
    		t.Errorf("loader called %d time(s) on anonymous request, want 0", loader.loads)
    	}
    	if !strings.Contains(rec.Body.String(), "identity not resolved") {
    		t.Errorf("body missing standard 403 message: %s", rec.Body.String())
    	}
    }

    // Test_HandleGetArchetype_LoaderError_500 — REQ-APSO-5 / Scenario 5:
    // loader returns err → 500 + server envelope, no info leak (the
    // underlying error string is NOT echoed in the body).
    func Test_HandleGetArchetype_LoaderError_500(t *testing.T) {
    	t.Parallel()

    	loader := &fakeCatalogLoader{err: errors.New("postgres: connection refused")}
    	resolver := &fakeResolver{signIn: true, orgID: "user_alice"}

    	e := echo.New()
    	if err := archetype.RegisterArchetypeRoutes(e, resolver, loader, nil); err != nil {
    		t.Fatalf("RegisterArchetypeRoutes: %v", err)
    	}

    	req := httptest.NewRequest(http.MethodGet, "/api/archetypes/assistant", nil)
    	rec := httptest.NewRecorder()
    	e.ServeHTTP(rec, req)

    	if rec.Code != http.StatusInternalServerError {
    		t.Fatalf("status = %d, want 500; body=%s", rec.Code, rec.Body.String())
    	}
    	if strings.Contains(rec.Body.String(), "postgres") {
    		t.Errorf("body leaks underlying error (postgres): %s", rec.Body.String())
    	}
    	if strings.Contains(rec.Body.String(), "connection refused") {
    		t.Errorf("body leaks underlying error (connection refused): %s", rec.Body.String())
    	}
    	if !strings.Contains(rec.Body.String(), "failed to load archetype config") {
    		t.Errorf("body missing standard 500 message: %s", rec.Body.String())
    	}
    }

    // Test_HandleGetArchetype_EmptySlug_400 — REQ-APSO-8 / Scenario 6: the
    // slug path parameter is whitespace-only (or otherwise empty after
    // TrimSpace) → 400 + validation envelope BEFORE the loader is called.
    func Test_HandleGetArchetype_EmptySlug_400(t *testing.T) {
    	t.Parallel()

    	loader := &fakeCatalogLoader{view: archetype.ArchetypeView{Slug: "assistant"}, found: true}
    	resolver := &fakeResolver{signIn: true, orgID: "user_alice"}

    	e := echo.New()
    	if err := archetype.RegisterArchetypeRoutes(e, resolver, loader, nil); err != nil {
    		t.Fatalf("RegisterArchetypeRoutes: %v", err)
    	}

    	// URL-encode a single space — Echo decodes that to a whitespace
    	// slug, which the handler trims and rejects as empty.
    	req := httptest.NewRequest(http.MethodGet, "/api/archetypes/%20", nil)
    	rec := httptest.NewRecorder()
    	e.ServeHTTP(rec, req)

    	if rec.Code != http.StatusBadRequest {
    		t.Fatalf("status = %d, want 400; body=%s", rec.Code, rec.Body.String())
    	}
    	if loader.loads != 0 {
    		t.Errorf("loader called %d time(s) on empty-slug request, want 0", loader.loads)
    	}
    	if !strings.Contains(rec.Body.String(), "missing slug path parameter") {
    		t.Errorf("body missing standard 400 message: %s", rec.Body.String())
    	}
    }

    // Test_HandleGetArchetype_OnlyLoaderRequired — REQ-APSO-6 + REQ-APSO-7 /
    // Scenario 7 part 1: the new arm MUST require loader != nil. With
    // loader=nil, the GET /:slug route is NOT mounted (404 returned by
    // Echo's default not-found handler).
    func Test_HandleGetArchetype_OnlyLoaderRequired(t *testing.T) {
    	t.Parallel()

    	// loader=nil, writer=non-nil → only the PUT arm is mounted.
    	// The new GET /:slug arm MUST NOT be mounted.
    	resolver := &fakeResolver{signIn: true, orgID: "user_alice"}
    	writer := &fakeWriter{}

    	e := echo.New()
    	if err := archetype.RegisterArchetypeRoutes(e, resolver, nil, writer); err != nil {
    		t.Fatalf("RegisterArchetypeRoutes: %v", err)
    	}

    	req := httptest.NewRequest(http.MethodGet, "/api/archetypes/assistant", nil)
    	rec := httptest.NewRecorder()
    	e.ServeHTTP(rec, req)

    	// The GET /:slug arm is intentionally NOT mounted when loader is
    	// nil — Echo returns 404 (no matching route). Any 5xx would
    	// indicate the handler ran with a nil loader (bug).
    	if rec.Code != http.StatusNotFound {
    		t.Fatalf("status = %d, want 404 (GET /:slug arm must NOT be mounted when loader=nil); body=%s",
    		rec.Code, rec.Body.String())
    	}
    }

    // Test_HandleGetArchetype_RouteOrderingWithConfigArm — REQ-APSO-6
    // (route ordering): GET /api/archetypes/assistant resolves to the
    // new per-slug handler (returns 200 + bare view, NO override key);
    // GET /api/archetypes/assistant/config/ resolves to the per-slug
    // /config/ handler (returns 200 + view WITH override key). The two
    // arms do NOT shadow each other (Echo's trie resolves by path
    // specificity).
    func Test_HandleGetArchetype_RouteOrderingWithConfigArm(t *testing.T) {
    	t.Parallel()

    	now := time.Date(2026, 8, 27, 10, 0, 0, 0, time.UTC)
    	viewWithOverride := archetype.ArchetypeView{
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
    	loader := &fakeCatalogLoader{view: viewWithOverride, found: true}
    	resolver := &fakeResolver{signIn: true, orgID: "user_alice"}

    	e := echo.New()
    	if err := archetype.RegisterArchetypeRoutes(e, resolver, loader, nil); err != nil {
    		t.Fatalf("RegisterArchetypeRoutes: %v", err)
    	}

    	// GET /api/archetypes/assistant → new arm → 200 + NO override key.
    	recBare := httptest.NewRecorder()
    	e.ServeHTTP(recBare, httptest.NewRequest(http.MethodGet, "/api/archetypes/assistant", nil))
    	if recBare.Code != http.StatusOK {
    		t.Fatalf("bare arm status = %d, want 200; body=%s", recBare.Code, recBare.Body.String())
    	}
    	if strings.Contains(recBare.Body.String(), `"override"`) {
    		t.Errorf("bare arm response leaked override key: %s", recBare.Body.String())
    	}

    	// GET /api/archetypes/assistant/config/ → /config/ arm → 200 + override key.
    	recConfig := httptest.NewRecorder()
    	e.ServeHTTP(recConfig, httptest.NewRequest(http.MethodGet, "/api/archetypes/assistant/config/", nil))
    	if recConfig.Code != http.StatusOK {
    		t.Fatalf("/config/ arm status = %d, want 200; body=%s", recConfig.Code, recConfig.Body.String())
    	}
    	if !strings.Contains(recConfig.Body.String(), `"override"`) {
    		t.Errorf("/config/ arm response missing override key: %s", recConfig.Body.String())
    	}
    }

    // (small helper to silence unused-imports when iterating)
    var _ = errors.New
