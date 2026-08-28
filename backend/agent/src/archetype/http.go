// Package archetype — http.go: polymorphic-by-slug HTTP handlers
// (T-19 PR-2 of cachicamas-archetype-system-foundation).
//
// The handlers expose the polymorphic-by-slug surface that the spec
// locks as SD-CASF-006 + SD-CASF-007:
//
//	GET  /api/archetypes/{slug}/config/  -> 200 / 403 / 404
//	PUT  /api/archetypes/{slug}/config/  -> 200 / 400 / 403 / 404
//
// The old `/api/chat/assistant/config` route group is retired in T-20
// (cmd/chat/main.go swaps the registration; chat/http.go drops the
// old handlers).
//
// Why these live in the archetype package (not chat/):
// the wire group is keyed by `slug`, not by an archetype kind. The
// chat package owns one consumer of the polymorphic Loader/Writer;
// the archetype package owns the generic-by-slug surface. Composition
// roots (today: cmd/chat/main.go) wire the route registration against
// the chat binary's specific loader+writer pair.
//
// Auth shape (AD-7 from design obs #4106):
//   - resolver is the production IdentityResolver (chat-package
//     identity) bridged into archetype.IdentityResolver via a small
//     closure in main.go. The handler refuses anonymous callers with
//     403 BEFORE the Loader is called (edge case 3 of spec §4).
//   - orgID is derived server-side from the JWE / session cookie;
//     the client never supplies it.
//   - Unknown slug surfaces 404 + ERR_UNKNOWN_SLUG. The polymorphic
//     Loader returns found=false for an absent system row; the PUT
//     path surfaces the Writer's ErrUnknownArchetypeSlug as 404
//     (the FK violation that the spec's edge case 4 demands).
//
// Validation rules are byte-equivalent (semantically) to the prior
// change's config.go:480-508 block — prompt non-empty, length <= 4000
// chars, no <script/<iframe substring, allowlist non-empty, every
// name in RegisteredToolNames, defer ⊆ allowlist.
package archetype

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"github.com/labstack/echo/v5"
)

// -----------------------------------------------------------------------------
// Auth surface
// -----------------------------------------------------------------------------

// Identity is the minimal contract the handler needs from a resolved
// session. ParticipantID doubles as the orgID for v1 (the chat
// composition root's v1 mapping is participantID == orgID; the
// multi-org mapping seam is intentionally deferred — see
// chat/doc.go and the design's AD-7).
type Identity interface {
	ParticipantID() string
}

// IdentityResolver extracts an Identity from an HTTP request. Returns
// (nil, false) for anonymous callers (the handler maps to 403).
//
// The chat package's chat.IdentityResolver satisfies this interface
// structurally (chat.Identity has a ParticipantID() method with the
// same shape). cmd/chat/main.go bridges via a closure so the
// archetype package does not import chat.
type IdentityResolver interface {
	IdentityFromRequest(ctx context.Context, r *http.Request) (Identity, bool)
}

// -----------------------------------------------------------------------------
// GET /api/archetypes/{slug}/config/
// -----------------------------------------------------------------------------

// HandleGetArchetypeConfig reads `:slug` from the path, derives orgID
// from the session, calls the polymorphic CatalogLoader, and returns
// the ArchetypeView JSON. The handler refuses anonymous callers with
// 403 (edge case 3 of spec §4); a Loader-found=false (no row) maps
// to 404 + ERR_UNKNOWN_SLUG.
func HandleGetArchetypeConfig(resolver IdentityResolver, loader CatalogLoader) echo.HandlerFunc {
	return func(c *echo.Context) error {
		if resolver == nil {
			return writeError(c, http.StatusInternalServerError, "server",
				"archetype resolver not wired", nil)
		}
		if loader == nil {
			return writeError(c, http.StatusInternalServerError, "server",
				"archetype loader not wired", nil)
		}
		ident, resolved := resolver.IdentityFromRequest(c.Request().Context(), c.Request())
		if !resolved || ident == nil {
			return writeError(c, http.StatusForbidden, "server",
				"identity not resolved", nil)
		}
		orgID := strings.TrimSpace(ident.ParticipantID())
		if orgID == "" {
			return writeError(c, http.StatusForbidden, "server",
				"identity missing participant id", nil)
		}
		slug := strings.TrimSpace(c.Param("slug"))
		if slug == "" {
			return writeError(c, http.StatusBadRequest, "validation",
				"missing slug path parameter", nil)
		}
		view, found, err := loader.LoadBySlug(c.Request().Context(), slug, orgID)
		if err != nil {
			// Defence in depth: do not echo the underlying error string
			// in the body (information leak). The error is logged at
			// higher layers via the default slog handler.
			return writeError(c, http.StatusInternalServerError, "server",
				"failed to load archetype config", nil)
		}
		if !found {
			return writeError(c, http.StatusNotFound, "not_found",
				"archetype slug is not registered", map[string]string{
					"code": "ERR_UNKNOWN_SLUG",
				})
		}
		return c.JSON(http.StatusOK, view)
	}
}

// -----------------------------------------------------------------------------
// GET /api/archetypes/{slug}
// -----------------------------------------------------------------------------

// HandleGetArchetype reads `:slug` from the path, derives orgID
// from the session, calls the polymorphic CatalogLoader, and
// returns the ArchetypeView JSON WITH THE OVERRIDE BLOCK STRIPPED.
// This is the "profile overlay" endpoint — it returns the parent +
// system-only child columns but does NOT carry the per-org
// customisation row.
//
// Stripping happens inline before serialising so the wire body
// always matches the parent-view contract documented in
// `frontend/src/lib/hooks/use-system-archetype.ts:syntheticFallbackView`.
// Callers that need the per-org override MUST hit /config/
// (HandleGetArchetypeConfig) instead.
//
// Error envelope parity with HandleGetArchetypeConfig:
//   - anonymous → 403
//   - found=false → 404 + ERR_UNKNOWN_SLUG
//   - loader err → 500, no info leak
//   - empty slug after TrimSpace → 400 (defence in depth)
//
// @see REQ-APSO-1..8
func HandleGetArchetype(resolver IdentityResolver, loader CatalogLoader) echo.HandlerFunc {
	return func(c *echo.Context) error {
		if resolver == nil {
			return writeError(c, http.StatusInternalServerError, "server",
				"archetype resolver not wired", nil)
		}
		if loader == nil {
			return writeError(c, http.StatusInternalServerError, "server",
				"archetype loader not wired", nil)
		}
		ident, resolved := resolver.IdentityFromRequest(c.Request().Context(), c.Request())
		if !resolved || ident == nil {
			return writeError(c, http.StatusForbidden, "server",
				"identity not resolved", nil)
		}
		orgID := strings.TrimSpace(ident.ParticipantID())
		if orgID == "" {
			return writeError(c, http.StatusForbidden, "server",
				"identity not resolved", nil)
		}
		slug := strings.TrimSpace(c.Param("slug"))
		if slug == "" {
			return writeError(c, http.StatusBadRequest, "validation",
				"missing slug path parameter", nil)
		}
		view, found, err := loader.LoadBySlug(c.Request().Context(), slug, orgID)
		if err != nil {
			// Defence in depth: do not echo the underlying error string
			// in the body (information leak). The error is logged at
			// higher layers via the default slog handler.
			return writeError(c, http.StatusInternalServerError, "server",
				"failed to load archetype config", nil)
		}
		if !found {
			return writeError(c, http.StatusNotFound, "not_found",
				"archetype slug is not registered", map[string]string{
					"code": "ERR_UNKNOWN_SLUG",
				})
		}
		// AD-DSN-2: strip the per-org override block inline. The
		// `json:"override,omitempty"` tag drops the nil pointer so the
		// wire body NEVER carries an `override` key.
		view.Override = nil
		return c.JSON(http.StatusOK, view)
	}
}

// -----------------------------------------------------------------------------
// PUT /api/archetypes/{slug}/config/
// -----------------------------------------------------------------------------

// HandlePutArchetypeConfig reads `:slug` from the path, validates the
// body against the same rules as the prior change's config.go:480-508
// block, and on success calls Writer.WriteConfig to persist the row +
// append one audit log row in the same transaction (REQ-CACL-002).
//
// Validation rejections return 400 + a per-rule error code WITHOUT
// touching the Writer (no row write, no log append). Anonymous → 403.
// Unknown slug → 404 + ERR_UNKNOWN_SLUG (the Writer surfaces
// ErrUnknownArchetypeSlug when the FK to archetypes(slug) is missing;
// the handler maps that to 404 per the spec's edge case 4).
func HandlePutArchetypeConfig(resolver IdentityResolver, writer Writer) echo.HandlerFunc {
	return func(c *echo.Context) error {
		if resolver == nil {
			return writeError(c, http.StatusInternalServerError, "server",
				"archetype resolver not wired", nil)
		}
		if writer == nil {
			return writeError(c, http.StatusInternalServerError, "server",
				"archetype writer not wired", nil)
		}
		ident, resolved := resolver.IdentityFromRequest(c.Request().Context(), c.Request())
		if !resolved || ident == nil {
			return writeError(c, http.StatusForbidden, "server",
				"identity not resolved", nil)
		}
		orgID := strings.TrimSpace(ident.ParticipantID())
		if orgID == "" {
			return writeError(c, http.StatusForbidden, "server",
				"identity missing participant id", nil)
		}
		slug := strings.TrimSpace(c.Param("slug"))
		if slug == "" {
			return writeError(c, http.StatusBadRequest, "validation",
				"missing slug path parameter", nil)
		}

		var body putArchetypeConfigRequest
		if err := jsonDecode(c, &body); err != nil {
			return writeError(c, http.StatusBadRequest, "validation",
				"invalid JSON body: "+err.Error(), nil)
		}
		if code, verr := validatePutArchetypeConfig(body); verr != nil {
			return writeError(c, http.StatusBadRequest, "validation",
				verr.Error(), map[string]string{"code": code})
		}

		update := ConfigUpdate{
			SystemPrompt:   strings.TrimSpace(body.SystemPrompt),
			ToolAllowlist:  append([]string(nil), body.ToolAllowlist...),
			DeferToolNames: append([]string(nil), body.DeferToolNames...),
		}
		cfg, werr := writer.WriteConfig(c.Request().Context(), slug, orgID, update, orgID)
		if werr != nil {
			// Map the package-level validation sentinels to 400 so a
			// defence-in-depth rejection inside WriteConfig surfaces
			// with the right status (not 500). ErrUnknownArchetypeSlug
			// maps to 404 (the slug is the source of truth in
			// `archetypes`; the FK violation rolls back the tx).
			if errors.Is(werr, ErrUnknownArchetypeSlug) {
				return writeError(c, http.StatusNotFound, "not_found",
					"archetype slug is not registered",
					map[string]string{"code": "ERR_UNKNOWN_SLUG"})
			}
			switch {
			case errors.Is(werr, ErrSystemPromptEmpty),
				errors.Is(werr, ErrSystemPromptTooLong),
				errors.Is(werr, ErrSystemPromptContainsHTML),
				errors.Is(werr, ErrToolAllowlistEmpty),
				errors.Is(werr, ErrUnknownToolName),
				errors.Is(werr, ErrDeferToolNotInAllowlist):
				return writeError(c, http.StatusBadRequest, "validation",
					werr.Error(), nil)
			default:
				return writeError(c, http.StatusInternalServerError, "server",
					"failed to write archetype config", nil)
			}
		}
		return c.JSON(http.StatusOK, cfg)
	}
}

// -----------------------------------------------------------------------------
// Validation (mirrors chat/http.go:validatePutAssistantConfig; spec
// REQ-CACAPI-002/003 + design AD-7)
// -----------------------------------------------------------------------------

type putArchetypeConfigRequest struct {
	SystemPrompt   string   `json:"system_prompt"`
	ToolAllowlist  []string `json:"tool_allowlist"`
	DeferToolNames []string `json:"defer_tool_names"`
	Model          *string  `json:"model,omitempty"`
}

// validatePutArchetypeConfig enforces the PUT body validation rules.
// Returns (error-code, sentinel-error) on failure; nil on success.
//
// Code values map 1:1 with the spec's distinct-codes contract
// (API-04). The body envelope returned to the client carries the
// `code` so the frontend can map the rejection to per-field UI
// affordances.
func validatePutArchetypeConfig(body putArchetypeConfigRequest) (string, error) {
	prompt := strings.TrimSpace(body.SystemPrompt)
	if prompt == "" {
		return "ERR_PROMPT_EMPTY", ErrSystemPromptEmpty
	}
	if len(prompt) > MaxSystemPromptLength {
		return "ERR_PROMPT_TOO_LONG", ErrSystemPromptTooLong
	}
	lower := strings.ToLower(prompt)
	if strings.Contains(lower, "<script") || strings.Contains(lower, "<iframe") {
		return "ERR_HTML_PATTERN", ErrSystemPromptContainsHTML
	}
	if len(body.ToolAllowlist) == 0 {
		return "ERR_ALLOWLIST_EMPTY", ErrToolAllowlistEmpty
	}
	registered := make(map[string]struct{}, len(RegisteredToolNames()))
	for _, n := range RegisteredToolNames() {
		registered[n] = struct{}{}
	}
	allow := make(map[string]struct{}, len(body.ToolAllowlist))
	for _, n := range body.ToolAllowlist {
		allow[n] = struct{}{}
		if _, ok := registered[n]; !ok {
			return "ERR_UNKNOWN_TOOL", ErrUnknownToolName
		}
	}
	for _, n := range body.DeferToolNames {
		if _, ok := allow[n]; !ok {
			return "ERR_DEFER_NOT_IN_ALLOWLIST", ErrDeferToolNotInAllowlist
		}
	}
	return "", nil
}

// -----------------------------------------------------------------------------
// GET /api/archetypes (the directory list)
// -----------------------------------------------------------------------------

// HandleListArchetypes reads orgID from the session, calls
// CatalogLoader.ListByType, and returns the directory list as a
// JSON array of ArchetypeView. The handler refuses anonymous
// callers with 403 (edge case 3 of spec §4); a Loader error maps
// to 500 (the error string is NOT echoed in the body to prevent
// information leaks). An empty result is a valid state and returns
// 200 + [] (NOT 404).
//
// The handler is registered at the bare /api/archetypes path
// (no slug). The response body is the JSON array itself (NOT an
// `{archetypes: [...]}` envelope) so the TS client's
// getJson<readonly ArchetypeView[]>() return type matches the
// Go side byte-for-byte.
//
// The optional `?type=` query parameter is validated against the
// three registered types. Absent → the full directory (all three
// types, R-24's unfiltered total set). Valid value (system |
// general | owned) → a subset projection of the same loader
// result, preserving (type ASC, slug ASC) order, so the R-24
// invariants (stable ordering, archived exclusion, per-org
// override projection, 200 []) are inherited unchanged. Any other
// present value (unknown, empty, junk) → 400 + validation
// envelope with fields.code=ERR_UNKNOWN_TYPE, and the loader is
// NOT called (the filter is validated before any data access).
func HandleListArchetypes(resolver IdentityResolver, loader CatalogLoader) echo.HandlerFunc {
	return func(c *echo.Context) error {
		if resolver == nil {
			return writeError(c, http.StatusInternalServerError, "server",
				"archetype resolver not wired", nil)
		}
		if loader == nil {
			return writeError(c, http.StatusInternalServerError, "server",
				"archetype loader not wired", nil)
		}
		ident, resolved := resolver.IdentityFromRequest(c.Request().Context(), c.Request())
		if !resolved || ident == nil {
			return writeError(c, http.StatusForbidden, "not_found",
				"anonymous", nil)
		}
		orgID := strings.TrimSpace(ident.ParticipantID())
		if orgID == "" {
			return writeError(c, http.StatusForbidden, "server",
				"identity missing participant id", nil)
		}
		// Optional ?type= filter: validate BEFORE any data access.
		wantType, hasType := "", false
		if raw, present := c.QueryParams()["type"]; present {
			hasType = true
			wantType = strings.TrimSpace(strings.Join(raw, ""))
			switch wantType {
			case "system", "general", "owned":
			default:
				return writeError(c, http.StatusBadRequest, "validation",
					"unknown type filter: want system, general, or owned",
					map[string]string{"code": "ERR_UNKNOWN_TYPE"})
			}
		}
		views, err := loader.ListByType(c.Request().Context(), orgID)
		if err != nil {
			// Defence in depth: do not echo the underlying error string
			// in the body (information leak). The error is logged at
			// higher layers via the default slog handler.
			return writeError(c, http.StatusInternalServerError, "server",
				"failed to list archetypes", nil)
		}
		// Always emit a JSON array (never null) so the client can
		// iterate without a null-check. An empty loader result
		// becomes `[]` on the wire. A valid ?type= filter is a subset
		// projection of the same loader result (never widens it), so
		// an empty projection is a non-nil `[]`, not an error.
		if views == nil {
			views = []ArchetypeView{}
		}
		if hasType {
			subset := make([]ArchetypeView, 0, len(views))
			for _, v := range views {
				if v.Type == wantType {
					subset = append(subset, v)
				}
			}
			views = subset
		}
		return c.JSON(http.StatusOK, views)
	}
}

// -----------------------------------------------------------------------------
// Route registration
// -----------------------------------------------------------------------------

// RegisterArchetypeRoutes mounts the polymorphic /api/archetypes
// surface: the directory list at /api/archetypes (GET, read-only)
// and the per-slug config at /api/archetypes/{slug}/config/ (GET
// + PUT). The resolver is required. The loader and writer are
// each optional independently: passing nil for one registers only
// the other handler (the list arm needs the loader; the per-slug
// GET arm needs the loader; the per-slug PUT arm needs the
// writer). Passing nil for both returns an error (mirrors the
// chat-package contract; the test in http_test.go locks this).
//
// Route order matters: the bare /api/archetypes arm is registered
// BEFORE the /:slug/config/ arm. Echo's trie-based router matches
// the more specific path first, so the list arm takes precedence
// over any slug-style path under the same group. Reversing the
// order would not actually break the routing (the path shapes are
// disjoint: `""` vs `/:slug/config/`), but listing the list arm
// first matches the reader's expectation that "the directory
// comes before the per-slug drill-down" and matches the order the
// tests in http_test.go exercise the registration in.
func RegisterArchetypeRoutes(e *echo.Echo, resolver IdentityResolver, loader CatalogLoader, writer Writer) error {
	if e == nil {
		return errors.New("archetype: RegisterArchetypeRoutes requires a non-nil *echo.Echo")
	}
	if resolver == nil {
		return errors.New("archetype: RegisterArchetypeRoutes requires a non-nil IdentityResolver")
	}
	if loader == nil && writer == nil {
		return errors.New("archetype: RegisterArchetypeRoutes requires at least one of CatalogLoader / Writer")
	}
	g := e.Group("/api/archetypes")
	if loader != nil {
		// List arm — registered first per the contract above.
		g.GET("", HandleListArchetypes(resolver, loader))
		// Profile-overlay arm: GET /:slug returns the parent view with the
		// override block stripped (REQ-APSO-1..8). Disjoint path shape
		// from /:slug/config/ — Echo's trie resolves by specificity.
		g.GET("/:slug", HandleGetArchetype(resolver, loader))
		g.GET("/:slug/config/", HandleGetArchetypeConfig(resolver, loader))
	}
	if writer != nil {
		g.PUT("/:slug/config/", HandlePutArchetypeConfig(resolver, writer))
	}
	return nil
}

// -----------------------------------------------------------------------------
// Internal helpers (mirror chat/http.go conventions)
// -----------------------------------------------------------------------------

// errorEnvelope is the JSON shape every non-2xx response carries.
// Mirrors chat/http.go:errorEnvelope so the chat and archetype
// packages share a single client-side parser contract.
type errorEnvelope struct {
	Kind    string            `json:"kind"`
	Message string            `json:"message"`
	Fields  map[string]string `json:"fields,omitempty"`
}

// writeError sends a JSON error envelope with the supplied status +
// kind + message + optional fields map (typically a single
// {"code": "ERR_HTML_PATTERN"} entry for validation rejections).
func writeError(c *echo.Context, status int, kind, message string, fields map[string]string) error {
	return c.JSON(status, errorEnvelope{
		Kind:    kind,
		Message: message,
		Fields:  fields,
	})
}

// jsonDecode decodes the request body with DisallowUnknownFields so
// typos in the body fail loudly rather than silently. Mirrors
// chat/http.go:jsonDecode.
func jsonDecode(c *echo.Context, dst any) error {
	dec := json.NewDecoder(c.Request().Body)
	dec.DisallowUnknownFields()
	return dec.Decode(dst)
}
