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
	"slices"
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
	// listByTypeFn lets a test model the real loader's per-org
	// projection (an override row only for the owning org) without
	// mutating the fake between sequential requests.
	listByTypeFn func(orgID string) []archetype.ArchetypeView
}

func (f *fakeCatalogLoader) LoadBySlug(_ context.Context, slug, orgID string) (archetype.ArchetypeView, bool, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.loads++
	f.lastSlug = slug
	f.lastOrgID = orgID
	return f.view, f.found, f.err
}

func (f *fakeCatalogLoader) ListByType(_ context.Context, orgID string) ([]archetype.ArchetypeView, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.listByTypeCalls++
	if f.listByTypeFn != nil {
		return f.listByTypeFn(orgID), f.listByTypeErr
	}
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

// Test_HandleGetArchetype_EmptyParticipantID_403 — REQ-APSO-3: a
// resolved identity with an empty ParticipantID receives the same standard
// 403 envelope as an unresolved identity, and the loader is not called.
func Test_HandleGetArchetype_EmptyParticipantID_403(t *testing.T) {
	t.Parallel()

	loader := &fakeCatalogLoader{}
	resolver := &fakeResolver{signIn: true, orgID: ""}

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
		t.Errorf("loader called %d time(s) on empty participant identity, want 0", loader.loads)
	}
	var envelope struct {
		Kind    string `json:"kind"`
		Message string `json:"message"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &envelope); err != nil {
		t.Fatalf("decode body: %v", err)
	}
	if envelope.Kind != "server" {
		t.Errorf("kind = %q, want server", envelope.Kind)
	}
	if envelope.Message != "identity not resolved" {
		t.Errorf("message = %q, want identity not resolved", envelope.Message)
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

// -----------------------------------------------------------------------------
// cachicamas-agent-catalog-config-reload (RED — T1.1, slice S1-R) — TDD
// contract for the optional `?type=` filter on GET /api/archetypes
// (CRL-R-001, scenarios CRL-S-001..009).
//
// Contract:
//   - Absent `type` → full directory, all three types (R-24 total set)
//   - Valid `type` ∈ {system, general, owned} → subset projection of the
//     same loader result, preserving (type ASC, slug ASC) order
//   - Invalid/empty `type` → 400 + {kind:validation, fields.code:ERR_UNKNOWN_TYPE},
//     loader NOT called (defence in depth)
//   - R-24 invariants inherited on every arm: stable order, archived rows
//     excluded, per-org override projected, empty result → 200 []
//
// RED state: the handler ignores ?type= today, so the invalid-filter
// and filtered-result tests fail while the absent-param test locks the
// unfiltered behavior it must keep.
// -----------------------------------------------------------------------------

// typeFilterSeed helpers — compact row constructors for the ?type=
// contract tests. The fake loader models the real loader's R-24
// output: rows ordered (type ASC, slug ASC), archived excluded at the
// WHERE boundary, per-org override attached only for user_alice.

var typeFilterNow = time.Date(2026, 8, 27, 10, 0, 0, 0, time.UTC)

func typeFilterRow(slug, typ string, status string) archetype.ArchetypeView {
	return archetype.ArchetypeView{
		Slug:        slug,
		Type:        typ,
		DisplayName: strings.ToUpper(slug[:1]) + slug[1:],
		Tagline:     "Row " + slug,
		Status:      status,
		CreatedAt:   typeFilterNow,
		CreatedBy:   "seed",
	}
}

func typeFilterViews(orgID string) []archetype.ArchetypeView {
	views := []archetype.ArchetypeView{
		typeFilterRow("general-one", "general", "active"),
		typeFilterRow("general-two", "general", "active"),
		typeFilterRow("owned-one", "owned", "active"),
		typeFilterRow("assistant", "system", "active"),
		typeFilterRow("system-two", "system", "active"),
	}
	if orgID == "user_alice" {
		views[3].Override = &archetype.ArchetypeOverride{
			SystemPrompt:   "you are the cachicamas assistant",
			ToolAllowlist:  []string{"current_time", "summarize_conversation"},
			DeferToolNames: []string{"summarize_conversation"},
			Version:        4,
			UpdatedAt:      typeFilterNow,
			UpdatedBy:      "user_alice",
		}
	}
	return views
}

// typeFilterSeedRaw includes the archived row the real loader excludes
// via its WHERE predicate; kept so the archived-exclusion contract
// has an explicit raw seed to contrast against (CRL-S-007).
func typeFilterSeedRaw() []archetype.ArchetypeView {
	archivedAt := typeFilterNow.Add(-24 * time.Hour)
	archived := typeFilterRow("archived-one", "general", "archived")
	archived.ArchivedAt = &archivedAt
	return append([]archetype.ArchetypeView{archived}, typeFilterViews("user_alice")...)
}

func newTypeFilterServer(t *testing.T, orgID string) (*echo.Echo, *fakeCatalogLoader) {
	t.Helper()
	loader := &fakeCatalogLoader{listByTypeFn: typeFilterViews}
	resolver := &fakeResolver{signIn: true, orgID: orgID}
	e := echo.New()
	if err := archetype.RegisterArchetypeRoutes(e, resolver, loader, nil); err != nil {
		t.Fatalf("RegisterArchetypeRoutes: %v", err)
	}
	return e, loader
}

func decodeListBody(t *testing.T, body []byte) []archetype.ArchetypeView {
	t.Helper()
	var got []archetype.ArchetypeView
	if err := json.Unmarshal(body, &got); err != nil {
		t.Fatalf("decode list body: %v; body=%s", err, string(body))
	}
	return got
}

func listSlugs(views []archetype.ArchetypeView) []string {
	slugs := make([]string, 0, len(views))
	for _, v := range views {
		slugs = append(slugs, v.Slug)
	}
	return slugs
}

func assertSlugOrder(t *testing.T, views []archetype.ArchetypeView, want []string) {
	t.Helper()
	got := listSlugs(views)
	if len(got) != len(want) {
		t.Fatalf("slugs = %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("got[%d].Slug = %q, want %q", i, got[i], want[i])
		}
	}
}

// Test_HandleListArchetypes_UnknownTypeFilter_Returns400WithCode —
// CRL-S-004 + CRL-S-005: any ?type= value outside {system, general,
// owned} (unknown, empty, junk) → 400 + validation envelope with
// fields.code=ERR_UNKNOWN_TYPE, non-empty message, loader NOT called
// (the filter is validated before any data access).
func Test_HandleListArchetypes_UnknownTypeFilter_Returns400WithCode(t *testing.T) {
	t.Parallel()

	cases := []string{"enterprise", "", "Sys%20Tem"}
	for _, value := range cases {
		t.Run("type="+value, func(t *testing.T) {
			e, loader := newTypeFilterServer(t, "user_alice")

			rec := httptest.NewRecorder()
			e.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/archetypes?type="+value, nil))

			if rec.Code != http.StatusBadRequest {
				t.Fatalf("status = %d, want 400; body=%s", rec.Code, rec.Body.String())
			}
			var env map[string]any
			if err := json.Unmarshal(rec.Body.Bytes(), &env); err != nil {
				t.Fatalf("decode envelope: %v; body=%s", err, rec.Body.String())
			}
			if env["kind"] != "validation" {
				t.Errorf("env.kind = %v, want validation", env["kind"])
			}
			if msg, _ := env["message"].(string); msg == "" {
				t.Error("env.message missing; want non-empty string")
			}
			fields, ok := env["fields"].(map[string]any)
			if !ok {
				t.Fatalf("env.fields missing or wrong shape: %v", env["fields"])
			}
			if fields["code"] != "ERR_UNKNOWN_TYPE" {
				t.Errorf("fields.code = %v, want ERR_UNKNOWN_TYPE", fields["code"])
			}
			if loader.listByTypeCalls != 0 {
				t.Errorf("ListByType called %d time(s), want 0 for invalid filter", loader.listByTypeCalls)
			}
		})
	}
}

// Test_HandleListArchetypes_TypeFilter_FiltersListPreservingOrder —
// CRL-S-002/003/006/007/008: each valid arm returns only that type's
// rows in (type ASC, slug ASC) order, the archived row never appears,
// the org-override survives for its owner (and only its owner), and
// repeated calls are stable.
func Test_HandleListArchetypes_TypeFilter_FiltersListPreservingOrder(t *testing.T) {
	t.Parallel()

	e, loader := newTypeFilterServer(t, "user_alice")

	arms := []struct {
		rawURL    string
		wantSlugs []string
	}{
		{"/api/archetypes?type=system", []string{"assistant", "system-two"}},
		{"/api/archetypes?type=general", []string{"general-one", "general-two"}},
		{"/api/archetypes?type=owned", []string{"owned-one"}},
	}
	for _, arm := range arms {
		var first []string
		for call := 1; call <= 2; call++ {
			rec := httptest.NewRecorder()
			e.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, arm.rawURL, nil))
			if rec.Code != http.StatusOK {
				t.Fatalf("%s status = %d, want 200; body=%s", arm.rawURL, rec.Code, rec.Body.String())
			}
			got := decodeListBody(t, rec.Body.Bytes())
			assertSlugOrder(t, got, arm.wantSlugs)
			order := listSlugs(got)
			if call == 1 {
				first = order
			} else if !slices.Equal(order, first) {
				t.Errorf("%s call %d order %v differs from call 1 %v", arm.rawURL, call, order, first)
			}
			// CRL-S-007: archived row never surfaces on any arm.
			if slices.Contains(order, "archived-one") {
				t.Errorf("%s returned archived row archived-one", arm.rawURL)
			}
		}
	}

	// CRL-S-008: owner org sees the override; non-owner does not.
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/archetypes?type=system", nil))
	for _, row := range decodeListBody(t, rec.Body.Bytes()) {
		if row.Slug != "assistant" {
			continue
		}
		if row.Override == nil || row.Override.SystemPrompt != "you are the cachicamas assistant" {
			t.Errorf("owner org assistant row override wrong: %+v", row.Override)
		}
	}
	eB, _ := newTypeFilterServer(t, "user_bob")
	recB := httptest.NewRecorder()
	eB.ServeHTTP(recB, httptest.NewRequest(http.MethodGet, "/api/archetypes?type=system", nil))
	for _, row := range decodeListBody(t, recB.Body.Bytes()) {
		if row.Slug == "assistant" && row.Override != nil {
			t.Errorf("non-owner org saw override on assistant row: %+v", row.Override)
		}
	}
	if loader.listByTypeCalls == 0 {
		t.Error("ListByType never called; filter arms must still read through the loader")
	}
}

// Test_HandleListArchetypes_AbsentType_ReturnsAllTypes — CRL-S-001 +
// CRL-S-006..008: no `type` param → the full directory across all
// three types (locks R-24's unfiltered total set), archived excluded,
// override projected for the owner, stable across calls.
func Test_HandleListArchetypes_AbsentType_ReturnsAllTypes(t *testing.T) {
	t.Parallel()

	e, _ := newTypeFilterServer(t, "user_alice")

	want := []string{"general-one", "general-two", "owned-one", "assistant", "system-two"}
	var first []string
	for call := 1; call <= 2; call++ {
		rec := httptest.NewRecorder()
		e.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/archetypes", nil))
		if rec.Code != http.StatusOK {
			t.Fatalf("status = %d, want 200; body=%s", rec.Code, rec.Body.String())
		}
		got := decodeListBody(t, rec.Body.Bytes())
		assertSlugOrder(t, got, want)
		typesSeen := map[string]bool{}
		for _, row := range got {
			typesSeen[row.Type] = true
			if row.Slug == "archived-one" {
				t.Error("absent arm returned archived row archived-one")
			}
		}
		for _, typ := range []string{"system", "general", "owned"} {
			if !typesSeen[typ] {
				t.Errorf("absent arm missing type %q", typ)
			}
		}
		order := listSlugs(got)
		if call == 1 {
			first = order
		} else if !slices.Equal(order, first) {
			t.Errorf("call %d order %v differs from call 1 %v", call, order, first)
		}
	}
	// Keep the raw-seed helper referenced so the archived-exclusion
	// contract stays documented in this test block.
	_ = typeFilterSeedRaw()
}

// Test_HandleListArchetypes_EmptyFilteredResult_Returns200EmptyArray —
// CRL-S-009: a valid filter with zero matching rows → 200 + literal []
// (never 404, never null).
func Test_HandleListArchetypes_EmptyFilteredResult_Returns200EmptyArray(t *testing.T) {
	t.Parallel()

	loader := &fakeCatalogLoader{listByTypeViews: []archetype.ArchetypeView{
		typeFilterRow("assistant", "system", "active"),
	}}
	resolver := &fakeResolver{signIn: true, orgID: "user_alice"}
	e := echo.New()
	if err := archetype.RegisterArchetypeRoutes(e, resolver, loader, nil); err != nil {
		t.Fatalf("RegisterArchetypeRoutes: %v", err)
	}

	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/archetypes?type=general", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body=%s", rec.Code, rec.Body.String())
	}
	if body := strings.TrimSpace(rec.Body.String()); body != "[]" {
		t.Errorf("body = %q, want literal empty array []", body)
	}
}
