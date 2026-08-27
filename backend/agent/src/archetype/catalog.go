// Package archetype — catalog.go: polymorphic Loader by slug.
//
// Implements design AD-1 (Class Table Inheritance) + AD-3 (system
// defaults live on system_archetypes only) + AD-4 (polymorphic-by-slug
// Loader) + AD-9 (status vocabulary terminal predicate).
//
// The Loader interface here is the new polymorphic surface:
// LoadBySlug(ctx, slug, orgID) returns ArchetypeView with parent +
// (when type='system') child columns populated. The previous
// LoadByKindAndOrg surface lives in config.go and is migrated to this
// shape in T-13.
//
// Per-org overrides live in archetype_configurations(archetype_slug,
// org_id); the JOIN in loadBySlugImpl is the single source of truth for
// "system row wins unless the org has customised" semantics.
//
// Future-addition seam (R-J):
//   To add a 'general' or 'owned' child table, add a CREATE TABLE
//   keyed by (slug) REFERENCES archetypes(slug), add a case to
//   ArchetypeView.ChildColumns(), and widen the JOIN in
//   loadBySlugImpl. No interface change required.
package archetype

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"
)

// ArchetypeView is the polymorphic read result. Parent columns are
// always populated when found=true. Child columns are populated only
// for type='system' today (and only via ChildColumns() — the public
// field surface stays in ArchetypeView so the JSON wire shape is
// stable across future child-table additions).
type ArchetypeView struct {
	// Parent — always present when found=true.
	Slug        string     `json:"slug"`
	Type        string     `json:"type"` // 'system' | 'general' | 'owned'
	DisplayName string     `json:"display_name"`
	Tagline     string     `json:"tagline"`
	Status      string     `json:"status"` // 'active' | 'draft' | 'archived'
	ArchivedAt  *time.Time `json:"archived_at,omitempty"`
	CreatedAt   time.Time  `json:"created_at"`
	CreatedBy   string     `json:"created_by"`

	// Per-org override (optional — present iff archetype_configurations
	// has a row for (archetype_slug, orgID)). nil when no override.
	Override *ArchetypeOverride `json:"override,omitempty"`

	// Private — holds the system child row when type='system' and the
	// child was joined. Accessed via ChildColumns() so future kinds
	// extend the surface without changing the public fields.
	systemChild *SystemArchetype
}

// SystemArchetype holds the type-specific columns for type='system'
// (SD-CASF-002). PK = FK to archetypes(slug); every row corresponds
// to exactly one parent.
type SystemArchetype struct {
	BundleVersion string `json:"bundle_version"`
	IsCritical    bool   `json:"is_critical"`
}

// ChildColumns returns the child-row struct for the archetype's type,
// or (nil, false) when the child is absent or the type has no child
// table yet. Callers must type-assert to access type-specific fields:
//
//	if sys, ok := view.ChildColumns().(*SystemArchetype); ok { … }
//
// The (any, bool) signature is the forward-compat seam: a future
// general_archetypes row returns *GeneralArchetype, ok=true; an
// unknown type returns (nil, false) — the Loader never errors on a
// missing child table (REQ-CASF-LD-05).
func (v ArchetypeView) ChildColumns() (any, bool) {
	switch v.Type {
	case "system":
		return v.systemChild, v.systemChild != nil
	default:
		return nil, false
	}
}

// ArchetypeOverride is the per-org customisation row stored in
// archetype_configurations(archetype_slug, org_id).
type ArchetypeOverride struct {
	SystemPrompt   string    `json:"system_prompt"`
	ToolAllowlist  []string  `json:"tool_allowlist"`
	DeferToolNames []string  `json:"defer_tool_names"`
	Model          *string   `json:"model,omitempty"`
	Version        int       `json:"version"`
	UpdatedAt      time.Time `json:"updated_at"`
	UpdatedBy      string    `json:"updated_by"`
}

// ErrUnknownArchetypeSlug replaces ErrUnknownArchetypeKind for the
// polymorphic-by-slug surface. Used by the handler to map to
// 404 + ERR_UNKNOWN_SLUG.
var ErrUnknownArchetypeSlug = errors.New("archetype: unknown slug")

// CatalogLoader is the polymorphic read port keyed by slug.
//
// LoadBySlug returns ArchetypeView, found, error. For type='system'
// it JOINs system_archetypes for child columns; for type='general' /
// 'owned' it tolerates a missing child table by returning
// (parent columns, child=nil, found=true). The Loader MUST NOT
// auto-write on absent row (REQ-CACS-003 inheritance); an absent
// system row returns DefaultConfigView(slug, orgID, knownTools) and
// found=false.
//
// WithTx returns a tx-bound Loader that issues SELECT … FOR SHARE
// on the join so concurrent Writers block until the tx commits.
//
// T-09 (PR-1 of cachicamas-archetype-system-foundation) introduces
// this as `CatalogLoader`. T-13 renames it to `Loader` and replaces
// the previous kind-keyed Loader in config.go.
type CatalogLoader interface {
	LoadBySlug(ctx context.Context, slug, orgID string) (ArchetypeView, bool, error)
	WithTx(tx *sql.Tx) CatalogLoader
}

// catalogPostgresLoader is the Postgres-backed implementation of
// CatalogLoader. Renamed to PostgresLoader in T-13.
type catalogPostgresLoader struct {
	db *sql.DB
	tx *sql.Tx
}

// NewCatalogLoader returns a CatalogLoader that reads from the
// supplied *sql.DB pool. Renamed to NewPostgresLoader in T-13.
func NewCatalogLoader(db *sql.DB) CatalogLoader {
	return &catalogPostgresLoader{db: db}
}

// WithTx returns a CatalogLoader bound to the supplied transaction.
// The tx-bound Loader's LoadBySlug issues SELECT … FOR SHARE on the
// archetypes + system_archetypes + archetype_configurations join.
func (l *catalogPostgresLoader) WithTx(tx *sql.Tx) CatalogLoader {
	return &catalogPostgresLoader{tx: tx}
}

// loadBySlugImpl is the shared body of LoadBySlug. Both the
// db-bound and tx-bound variants dispatch here.
func (l *catalogPostgresLoader) loadBySlugImpl(ctx context.Context, slug, orgID string) (ArchetypeView, bool, error) {
	if strings.TrimSpace(slug) == "" {
		return ArchetypeView{}, false, ErrUnknownArchetypeSlug
	}
	if strings.TrimSpace(orgID) == "" {
		return ArchetypeView{}, false, errors.New("archetype loader: orgID must be non-empty")
	}

	// Step 1: read the parent row + the type-specific child row (when
	// type='system') + the per-org override (when one exists).
	// The single JOIN avoids the two-step lookup that the previous
	// Loader required.
	//
	// SELECT … FOR SHARE when tx-bound; otherwise plain SELECT.
	const baseSelect = `
SELECT
    a.slug,
    a.type,
    a.display_name,
    a.tagline,
    a.status,
    a.archived_at,
    a.created_at,
    a.created_by,
    sa.bundle_version,
    sa.is_critical,
    c.system_prompt,
    c.tool_allowlist,
    c.defer_tool_names,
    c.model,
    c.version,
    c.updated_at,
    c.updated_by
FROM archetypes a
LEFT JOIN system_archetypes sa ON sa.slug = a.slug
LEFT JOIN archetype_configurations c
    ON c.archetype_slug = a.slug AND c.org_id = $2
WHERE a.slug = $1
  AND NOT (a.status = 'archived' OR a.archived_at IS NOT NULL)`

	query := baseSelect
	if l.tx != nil {
		query = baseSelect + `
FOR SHARE OF a, sa, c`
	}

	var (
		row *sql.Row
	)
	args := []any{slug, orgID}
	if l.tx != nil {
		row = l.tx.QueryRowContext(ctx, query, args...)
	} else {
		row = l.db.QueryRowContext(ctx, query, args...)
	}

	var (
		gotSlug, gotType, gotDisplay, gotTagline, gotStatus, gotCreatedBy string
		gotArchivedAt                                                      sql.NullTime
		gotCreatedAt                                                       time.Time
		childBundle                                                        sql.NullString
		childCritical                                                      sql.NullBool
		overridePrompt                                                     sql.NullString
		overrideAllowlist, overrideDefers                                  []byte
		overrideModel                                                      sql.NullString
		overrideVersion                                                    sql.NullInt64
		overrideUpdatedAt                                                  sql.NullTime
		overrideUpdatedBy                                                  sql.NullString
	)
	if err := row.Scan(
		&gotSlug, &gotType, &gotDisplay, &gotTagline, &gotStatus,
		&gotArchivedAt, &gotCreatedAt, &gotCreatedBy,
		&childBundle, &childCritical,
		&overridePrompt, &overrideAllowlist, &overrideDefers,
		&overrideModel, &overrideVersion, &overrideUpdatedAt, &overrideUpdatedBy,
	); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			// No parent row — return the safe-default fallback. This is
			// the "system catalogue empty" outage path. The Loader MUST
			// NOT auto-write (REQ-CACS-003).
			known := RegisteredToolNames()
			return DefaultConfigView(slug, orgID, known), false, nil
		}
		return ArchetypeView{}, false, fmt.Errorf("archetype loader: scan row for slug=%q org=%q: %w", slug, orgID, err)
	}

	view := ArchetypeView{
		Slug:        gotSlug,
		Type:        gotType,
		DisplayName: gotDisplay,
		Tagline:     gotTagline,
		Status:      gotStatus,
		CreatedAt:   gotCreatedAt,
		CreatedBy:   gotCreatedBy,
	}
	if gotArchivedAt.Valid {
		t := gotArchivedAt.Time
		view.ArchivedAt = &t
	}

	// Child columns: type='system' only today.
	if gotType == "system" && childBundle.Valid {
		view.systemChild = &SystemArchetype{
			BundleVersion: childBundle.String,
			IsCritical:    childCritical.Bool,
		}
	}

	// Per-org override: present iff a row exists in
	// archetype_configurations for (slug, orgID).
	if overridePrompt.Valid {
		ov := &ArchetypeOverride{
			SystemPrompt: overridePrompt.String,
		}
		if len(overrideAllowlist) > 0 {
			if err := json.Unmarshal(overrideAllowlist, &ov.ToolAllowlist); err != nil {
				return ArchetypeView{}, false, fmt.Errorf("archetype loader: decode override tool_allowlist: %w", err)
			}
		}
		if len(overrideDefers) > 0 {
			if err := json.Unmarshal(overrideDefers, &ov.DeferToolNames); err != nil {
				return ArchetypeView{}, false, fmt.Errorf("archetype loader: decode override defer_tool_names: %w", err)
			}
		}
		if overrideModel.Valid {
			m := overrideModel.String
			ov.Model = &m
		}
		if overrideVersion.Valid {
			ov.Version = int(overrideVersion.Int64)
		}
		if overrideUpdatedAt.Valid {
			ov.UpdatedAt = overrideUpdatedAt.Time
		}
		ov.UpdatedBy = overrideUpdatedBy.String
		view.Override = ov
	}

	return view, true, nil
}

// LoadBySlug reads the polymorphic ArchetypeView for the supplied
// (slug, orgID) pair. See CatalogLoader contract for behaviour matrix.
func (l *catalogPostgresLoader) LoadBySlug(ctx context.Context, slug, orgID string) (ArchetypeView, bool, error) {
	return l.loadBySlugImpl(ctx, slug, orgID)
}

// DefaultConfigView is the polymorphic-by-slug twin of DefaultConfig
// (config.go). Pure factory; same input → same output; no I/O.
// Used by the Loader's absent-row fallback (REQ-CACS-003).
func DefaultConfigView(slug, orgID string, knownTools []string) ArchetypeView {
	allowlist := append([]string(nil), knownTools...)
	defers := append([]string(nil), DefaultDeferToolNames...)
	prompt := DefaultChatSystemPrompt
	return ArchetypeView{
		Slug:        slug,
		Type:        "system",
		DisplayName: "Assistant",
		Tagline:     "Your default assistant",
		Status:      "active",
		CreatedBy:   "default",
		Override: &ArchetypeOverride{
			SystemPrompt:   prompt,
			ToolAllowlist:  allowlist,
			DeferToolNames: defers,
			Model:          nil,
			Version:        1,
			UpdatedBy:      "default",
			UpdatedAt:      time.Time{},
		},
	}
}
