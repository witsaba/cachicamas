// Package archetype is the generic Layer 3 archetype configuration
// store (REQ-CACS-001/002/003, design AD-1/AD-2 from
// cachicamas-assistant-configuration-ui). It is intentionally NOT
// scoped to a single archetype (e.g. chat, coding). Any archetype
// in the system reads and writes its own configuration through this
// package, keyed by (archetype_kind, org_id).
//
// Why this lives in its own package (not src/chat/, not src/coding/):
// the configuration contract is a Layer 3 concern shared by every
// archetype. Putting it in a specific archetype's package would couple
// future archetypes (coding, support, ...) to whichever package ships
// first — exactly the duplication the spec calls out as a non-goal
// ("multi-archetype config centralisation"). A neutral package keeps
// the layering explicit: archetypes import this; this does not import
// any archetype.
//
// Composition: a single forward-only migration creates
// `archetype_configurations` (src/archetype/migrations/0001_archetype_configurations.sql).
// A composition root that wants to register an archetype calls
// `archetype.NewPostgresLoader(db)` and routes its archetype-specific
// HTTP handlers around that loader. The Loader MUST NOT auto-write on
// absent row (REQ-CACS-003): an absent read returns safe defaults in
// memory; persistence is the write API's responsibility.
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

// ArchetypeKind names the family of an archetype so the
// `archetype_configurations` table can hold rows for more than one
// archetype (chat, coding, future support, ...). Today only KindChat is
// wired; the type is exported so a future archetype can register
// without touching this package.
type ArchetypeKind string

const (
	// KindChat is the chat archetype (Layer 3, owned by src/chat/).
	// The Assistant on the /agents page is one instance of KindChat.
	KindChat ArchetypeKind = "chat"
)

// ArchetypeConfig is the persisted configuration row for one
// (archetype_kind, org_id) pair. Stored in `archetype_configurations`.
//
// Field provenance:
//   - SystemPrompt: configurable; default is the per-archetype literal
//     supplied via DefaultConfig (today: the chat v1 literal).
//   - ToolAllowlist: persisted names; toggled via PUT. Implementations
//     stay in code per ADR 0006.
//   - DeferToolNames: subset of ToolAllowlist requiring permission
//     approval before each invocation.
//   - Model: optional informational field; the actual model selection
//     remains env-driven (process-wide) per locked decision.
//   - Version: monotonic; bumped on every successful PUT. Used by the
//     per-archetype version propagation contract.
//   - UpdatedAt/UpdatedBy: server-set on every successful PUT.
type ArchetypeConfig struct {
	Kind           ArchetypeKind `json:"kind"`
	OrgID          string        `json:"org_id"`
	SystemPrompt   string        `json:"system_prompt"`
	ToolAllowlist  []string      `json:"tool_allowlist"`
	DeferToolNames []string      `json:"defer_tool_names"`
	Model          *string       `json:"model,omitempty"`
	Version        int           `json:"version"`
	UpdatedBy      string        `json:"updated_by,omitempty"`
	UpdatedAt      time.Time     `json:"updated_at"`
	// IsOverride is true when the config came from a per-org row that
	// shadows the system default. False when the config came from the
	// seeded `__default__` row OR from the in-memory fallback (DB
	// outage). The frontend uses this to decide whether to label the
	// Assistant card "Configured" (user-customized) or "Default"
	// (system default).
	IsOverride bool `json:"is_override"`
}

// DefaultChatSystemPrompt is the chat archetype's v1 system prompt.
// Kept exported so the chat archetype's loader reuses the literal
// without copy-paste drift. Future archetypes will define their own
// default prompts in their composition root.
const DefaultChatSystemPrompt = "You are the cachicamas chat assistant; answer the participant in plain, well-formatted text."

// DefaultDeferToolNames is the safe-default defer set. Tools whose name
// appears here require explicit permission approval before each call.
// `summarize_conversation` mutates chat state, so it is deferred by
// default in the chat archetype.
var DefaultDeferToolNames = []string{"summarize_conversation"}

// MaxSystemPromptLength caps the persisted system prompt. Mirrors the
// 4000-char cap on the front-end Configure section.
const MaxSystemPromptLength = 4000

// Validation sentinels. Each is distinct under `errors.Is` so the
// handler can map to specific 400 responses without string-matching.
var (
	// ErrSystemPromptEmpty is returned when the prompt body is empty
	// after trimming whitespace.
	ErrSystemPromptEmpty = errors.New("archetype config: system_prompt is empty")
	// ErrSystemPromptTooLong is returned when the prompt exceeds
	// MaxSystemPromptLength.
	ErrSystemPromptTooLong = errors.New("archetype config: system_prompt exceeds maximum length")
	// ErrSystemPromptContainsHTML is returned when the prompt body
	// contains a `<script` or `<iframe` substring (case-insensitive).
	ErrSystemPromptContainsHTML = errors.New("archetype config: system_prompt contains disallowed HTML pattern")
	// ErrToolAllowlistEmpty is returned when tool_allowlist is empty.
	ErrToolAllowlistEmpty = errors.New("archetype config: tool_allowlist must contain at least one tool")
	// ErrUnknownToolName is returned when tool_allowlist contains a
	// name not in the registered tool set.
	ErrUnknownToolName = errors.New("archetype config: tool_allowlist contains a name not in the registered tool set")
	// ErrDeferToolNotInAllowlist is returned when defer_tool_names
	// contains a name not present in tool_allowlist.
	ErrDeferToolNotInAllowlist = errors.New("archetype config: defer_tool_names must be a subset of tool_allowlist")
	// ErrUnknownArchetypeKind is returned when the supplied kind is
	// not one of the registered ArchetypeKind constants.
	ErrUnknownArchetypeKind = errors.New("archetype config: unknown archetype kind")
)

// Loader is the read port for ArchetypeConfig. The Postgres adapter is
// the only shipped implementation; in-memory fakes can be added for
// tests by implementing this interface.
//
// LoadByKindAndOrg returns a value (not a pointer) so callers can
// compare the returned ArchetypeConfig to a declared-zero-value safely.
// The safe-default path returns DefaultConfig(kind, orgID, knownTools)
// by value, never nil; the absent-row signal is the bool second return.
type Loader interface {
	// LoadByKindAndOrg returns the persisted config for the supplied
	// (kind, orgID) pair. When the row is absent, it returns the
	// safe-default config (built from DefaultConfig), found=false, and
	// MUST NOT auto-write — REQ-CACS-003.
	LoadByKindAndOrg(ctx context.Context, kind ArchetypeKind, orgID string) (ArchetypeConfig, bool, error)

	// WithTx returns a Loader that issues its read inside the supplied
	// transaction. Used by the FOR SHARE serialisation contract (design
	// AD-2): when caller A holds LoadByKindAndOrg inside a tx,
	// concurrent writer B is blocked by Postgres's row-level shared
	// lock until A commits.
	WithTx(tx *sql.Tx) Loader
}

// PostgresLoader is the Postgres-backed implementation of Loader.
// Holds either a *sql.DB (default) or a *sql.Tx (after WithTx). The two
// are mutually exclusive per Loader instance.
type PostgresLoader struct {
	db *sql.DB
	tx *sql.Tx
}

// NewPostgresLoader returns a Loader that reads from the supplied
// *sql.DB pool. The pool is the archetype migration runner pool
// (CACHICAMAS_CHAT_STORE_DSN — same DSN the chat store uses; the
// schema lives in the same database for v1).
func NewPostgresLoader(db *sql.DB) Loader {
	return &PostgresLoader{db: db}
}

// WithTx returns a Loader bound to the supplied transaction. The
// returned Loader's LoadByKindAndOrg issues a SELECT ... FOR SHARE so
// that concurrent UPDATE statements on the same row block until the
// tx commits or rolls back.
func (l *PostgresLoader) WithTx(tx *sql.Tx) Loader {
	return &PostgresLoader{tx: tx}
}

// DefaultRowOrgID is the sentinel org_id used by the per-archetype
// system default row (migration 0003_seed_chat_default.sql). The
// Loader recognises this sentinel and rewrites the returned
// ArchetypeConfig.OrgID to the caller's orgID — the row itself is
// shared across all orgs that have not yet customised.
const DefaultRowOrgID = "__default__"

// LoadByKindAndOrg reads `archetype_configurations` for the supplied
// (kind, orgID) pair. Two-step lookup:
//   1. SELECT ... WHERE kind = $1 AND org_id = $2 (caller's per-org row)
//   2. If absent, SELECT ... WHERE kind = $1 AND org_id = '__default__'
//      (per-archetype system default, seeded by migration 0003)
//   3. If still absent, return in-memory `DefaultConfig(kind, orgID, ...)`
//      — this is the LAST-RESORT fallback for a DB outage (REQ-CACS-003:
//      "if the database for any reason fails, MUST to use hardcoded
//      default values").
//
// On a tx-bound Loader the queries use `FOR SHARE`; on a db-bound
// Loader they use plain `SELECT`. The IsOverride flag distinguishes
// "user-customised" (true) from "system default" / "in-memory fallback"
// (false) so the frontend can label the Assistant card correctly.
//
// Behaviour matrix:
//   - per-org row present  → returns row, found=true,  IsOverride=true
//   - default row only    → returns row, found=true,  IsOverride=false
//                            (OrgID is rewritten to the caller's orgID)
//   - no rows at all       → returns safe-default, found=false,
//                            IsOverride=false
//   - DB query/scan fails  → returns safe-default, found=false,
//                            IsOverride=false (logged at higher layer)
//
// The Loader MUST NOT auto-write the safe default. Persistence is the
// write API layer's responsibility (REQ-CACS-003).
func (l *PostgresLoader) LoadByKindAndOrg(ctx context.Context, kind ArchetypeKind, orgID string) (ArchetypeConfig, bool, error) {
	if kind == "" {
		return ArchetypeConfig{}, false, ErrUnknownArchetypeKind
	}
	if strings.TrimSpace(orgID) == "" {
		return ArchetypeConfig{}, false, errors.New("archetype config loader: orgID must be non-empty")
	}

	// Step 1: per-org row.
	if cfg, found, err := l.loadRow(ctx, kind, orgID); err != nil {
		return ArchetypeConfig{}, false, err
	} else if found {
		cfg.IsOverride = true
		return cfg, true, nil
	}

	// Step 2: per-archetype system default (sentinel org_id).
	if cfg, found, err := l.loadRow(ctx, kind, DefaultRowOrgID); err != nil {
		return ArchetypeConfig{}, false, err
	} else if found {
		cfg.OrgID = orgID // rewrite to caller's orgID; the row itself stays at the sentinel
		cfg.IsOverride = false
		return cfg, true, nil
	}

	// Step 3: no rows at all → in-memory fallback (DB up but seed
	// missing or both rows deleted). Returns found=false so the caller
	// can distinguish this from the default-row case if needed.
	known := knownToolNames(ctx)
	return DefaultConfig(kind, orgID, known), false, nil
}

// loadRow issues the SELECT for an exact (kind, orgID) pair and
// decodes the result. Used by LoadByKindAndOrg's two-step lookup.
// Returns (zero, false, nil) when the row is absent.
func (l *PostgresLoader) loadRow(ctx context.Context, kind ArchetypeKind, orgID string) (ArchetypeConfig, bool, error) {
	const baseSelect = `SELECT archetype_kind, org_id, system_prompt, tool_allowlist, defer_tool_names, model, version, updated_at, updated_by FROM archetype_configurations WHERE archetype_kind = $1 AND org_id = $2`
	query := baseSelect
	if l.tx != nil {
		query = baseSelect + ` FOR SHARE`
	}
	var (
		row          *sql.Row
		gotKind      string
		gotOrgID     string
		gotPrompt    string
		rawAllowlist []byte
		rawDefers    []byte
		gotModel     sql.NullString
		gotVersion   int
		gotUpdatedAt time.Time
		gotUpdatedBy string
	)
	if l.tx != nil {
		row = l.tx.QueryRowContext(ctx, query, string(kind), orgID)
	} else {
		row = l.db.QueryRowContext(ctx, query, string(kind), orgID)
	}
	if err := row.Scan(&gotKind, &gotOrgID, &gotPrompt, &rawAllowlist, &rawDefers, &gotModel, &gotVersion, &gotUpdatedAt, &gotUpdatedBy); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return ArchetypeConfig{}, false, nil
		}
		return ArchetypeConfig{}, false, fmt.Errorf("archetype config loader: scan row for kind=%q org=%q: %w", kind, orgID, err)
	}

	var allowlist []string
	if len(rawAllowlist) > 0 {
		if err := json.Unmarshal(rawAllowlist, &allowlist); err != nil {
			return ArchetypeConfig{}, false, fmt.Errorf("archetype config loader: decode tool_allowlist for kind=%q org=%q: %w", kind, orgID, err)
		}
	}
	var defers []string
	if len(rawDefers) > 0 {
		if err := json.Unmarshal(rawDefers, &defers); err != nil {
			return ArchetypeConfig{}, false, fmt.Errorf("archetype config loader: decode defer_tool_names for kind=%q org=%q: %w", kind, orgID, err)
		}
	}

	cfg := ArchetypeConfig{
		Kind:           ArchetypeKind(gotKind),
		OrgID:          gotOrgID,
		SystemPrompt:   gotPrompt,
		ToolAllowlist:  allowlist,
		DeferToolNames: defers,
		Version:        gotVersion,
		UpdatedAt:      gotUpdatedAt,
		UpdatedBy:      gotUpdatedBy,
	}
	if gotModel.Valid {
		m := gotModel.String
		cfg.Model = &m
	}
	return cfg, true, nil
}

// DefaultConfig is the safe-default factory (REQ-CACS-003, design
// AD-2). Pure: same input → same output, no I/O. The `knownTools`
// argument is the registered tool set supplied by the composition
// root; the defaults only allow tools that actually exist, so the
// first PUT cannot bypass the registry check by listing unknown names.
//
// Returns a value (not a pointer) so callers — including the Loader's
// absent-row path — can return it directly without dereferencing.
//
// The default system prompt is currently the chat v1 literal. Future
// archetypes can supply their own default via a switch on `kind`; the
// loader will pass `known` from the registered tool set of whichever
// archetype owns the read.
func DefaultConfig(kind ArchetypeKind, orgID string, knownTools []string) ArchetypeConfig {
	allowlist := append([]string(nil), knownTools...)
	defers := append([]string(nil), DefaultDeferToolNames...)
	prompt := DefaultChatSystemPrompt
	switch kind {
	case KindChat:
		prompt = DefaultChatSystemPrompt
	default:
		// For future kinds, the composition root must register the
		// kind's default prompt via SetDefaultPromptForKind before
		// the Loader serves its first read.
		if override, ok := defaultPromptsByKind[kind]; ok {
			prompt = override
		}
	}
	return ArchetypeConfig{
		Kind:           kind,
		OrgID:          orgID,
		SystemPrompt:   prompt,
		ToolAllowlist:  allowlist,
		DeferToolNames: defers,
		Model:          nil,
		Version:        1,
		UpdatedBy:      "",
		UpdatedAt:      time.Time{},
	}
}

// defaultPromptsByKind holds per-kind default system prompts for
// future archetypes. The chat kind uses DefaultChatSystemPrompt
// directly (no entry needed). Set via SetDefaultPromptForKind.
var defaultPromptsByKind = map[ArchetypeKind]string{}

// SetDefaultPromptForKind registers the default system prompt for a
// future archetype kind. The chat kind is hardcoded (DefaultChatSystemPrompt)
// and ignores this registry. Call from the future archetype's
// composition root at startup.
func SetDefaultPromptForKind(kind ArchetypeKind, prompt string) {
	defaultPromptsByKind[kind] = prompt
}

// knownToolNames returns the registered tool names from the chat
// archetype's tool source. The composition root wires the registry;
// the helper consults a package-level set when set (set by
// cmd/chat/main.go after construction) and falls back to the
// hardcoded current default otherwise. The fallback keeps the Loader
// usable in tests and in any composition that does not call
// SetRegisteredToolNames.
//
// For v1 there is only one archetype (chat), so the helper returns the
// chat registry's names. Future archetypes would supply their own
// registry by extending this layer.
var registeredToolNames = func() []string {
	return []string{"current_time", "summarize_conversation"}
}

// SetRegisteredToolNames overrides the package-level set used by
// DefaultConfig and the Loader's absent-row fallback. Called once
// from cmd/chat/main.go after the chat ToolSource registry is built.
func SetRegisteredToolNames(names []string) {
	copied := append([]string(nil), names...)
	registeredToolNames = func() []string {
		return copied
	}
}

// RegisteredToolNames returns a copy of the package-level registered
// tool-name set. Exposed so the write-side handler can validate that
// a PUT's tool_allowlist only references known tool names
// (REQ-CACAPI-003). Returns a copy to prevent callers from mutating
// the package state.
func RegisteredToolNames() []string {
	return registeredToolNames()
}

// knownToolNames is the closure-returning helper used by LoadByKindAndOrg.
func knownToolNames(_ context.Context) []string {
	return registeredToolNames()
}

// ConfigUpdate is the PUT body shape — every field the runtime
// configures, minus the server-set columns (org_id, version,
// updated_at, updated_by, kind). The handler validates each field
// against the package-level sentinels BEFORE calling Writer.WriteConfig;
// the Writer itself does NOT re-validate (the validation rules are
// caller-owned). This keeps WriteConfig a pure persistence port
// with no business logic mixed in.
type ConfigUpdate struct {
	SystemPrompt   string
	ToolAllowlist  []string
	DeferToolNames []string
	Model          *string
}

// Writer is the write port for ArchetypeConfig. The Postgres
// implementation is the only shipped impl; in-memory fakes can be
// added for tests by implementing this interface.
//
// WriteConfig MUST:
//   - Validate the ConfigUpdate fields (callers MAY pre-validate,
//     but the Writer enforces its own invariants — defence in depth).
//   - UPSERT the (kind, org_id) row, bumping version on every write.
//   - Append exactly one row to archetype_configurations_log inside
//     the SAME transaction as the UPSERT — REQ-CACL-002.
//   - Return the new ArchetypeConfig (with the bumped version and
//     server-set updated_at/updated_by).
//
// On any failure (validation, scan, db error) the transaction is
// rolled back and no log row is appended.
type Writer interface {
	WriteConfig(ctx context.Context, kind ArchetypeKind, orgID string, update ConfigUpdate, actor string) (ArchetypeConfig, error)
}

// PostgresWriter is the Postgres-backed implementation of Writer.
// Holds a *sql.DB (no *sql.Tx — WriteConfig manages its own tx so the
// caller never has to think about rollback semantics).
type PostgresWriter struct {
	db *sql.DB
}

// NewPostgresWriter returns a Writer that writes through the supplied
// *sql.DB pool. The pool is the archetype migration runner pool
// (CACHICAMAS_CHAT_STORE_DSN — same DSN the Loader uses).
func NewPostgresWriter(db *sql.DB) Writer {
	return &PostgresWriter{db: db}
}

// WriteConfig performs the atomic UPSERT + audit-log append. See
// Writer contract for the full behaviour.
//
// Validation enforced here (defence in depth — the handler also
// validates before calling, but WriteConfig must be safe for direct
// use):
//   - SystemPrompt non-empty after trim
//   - SystemPrompt length <= MaxSystemPromptLength
//   - SystemPrompt contains no `<script` or `<iframe` substring
//     (case-insensitive)
//   - ToolAllowlist non-empty
//   - DeferToolNames ⊆ ToolAllowlist
//
// Tool name existence (Allowlist ⊆ registeredToolNames) is NOT
// enforced here — that check requires the registered set, which is
// owned by the composition root. The handler performs it before
// calling WriteConfig. (Skipping it here keeps WriteConfig a pure
// persistence port; a caller with a different tool registry can
// supply its own validation.)
func (w *PostgresWriter) WriteConfig(ctx context.Context, kind ArchetypeKind, orgID string, update ConfigUpdate, actor string) (ArchetypeConfig, error) {
	if kind == "" {
		return ArchetypeConfig{}, ErrUnknownArchetypeKind
	}
	if strings.TrimSpace(orgID) == "" {
		return ArchetypeConfig{}, errors.New("archetype writer: orgID must be non-empty")
	}

	// Defence-in-depth validation. The handler MUST have validated
	// before calling, but if a future caller bypasses the handler
	// we still want the row to be rejected cleanly.
	prompt := strings.TrimSpace(update.SystemPrompt)
	if prompt == "" {
		return ArchetypeConfig{}, ErrSystemPromptEmpty
	}
	if len(prompt) > MaxSystemPromptLength {
		return ArchetypeConfig{}, ErrSystemPromptTooLong
	}
	lower := strings.ToLower(prompt)
	if strings.Contains(lower, "<script") || strings.Contains(lower, "<iframe") {
		return ArchetypeConfig{}, ErrSystemPromptContainsHTML
	}
	if len(update.ToolAllowlist) == 0 {
		return ArchetypeConfig{}, ErrToolAllowlistEmpty
	}
	allow := make(map[string]struct{}, len(update.ToolAllowlist))
	for _, name := range update.ToolAllowlist {
		allow[name] = struct{}{}
	}
	for _, name := range update.DeferToolNames {
		if _, ok := allow[name]; !ok {
			return ArchetypeConfig{}, ErrDeferToolNotInAllowlist
		}
	}

	tx, err := w.db.BeginTx(ctx, nil)
	if err != nil {
		return ArchetypeConfig{}, fmt.Errorf("archetype writer: begin tx: %w", err)
	}
	committed := false
	defer func() {
		if !committed {
			_ = tx.Rollback()
		}
	}()

	// Read the prior row (if any) for `before` + version source.
	var beforeJSON []byte
	var prevVersion int
	row := tx.QueryRowContext(ctx,
		`SELECT version, to_jsonb(archetype_configurations) FROM archetype_configurations WHERE archetype_kind = $1 AND org_id = $2 FOR UPDATE`,
		string(kind), orgID)
	var prev []byte
	if scanErr := row.Scan(&prevVersion, &prev); scanErr != nil && !errors.Is(scanErr, sql.ErrNoRows) {
		return ArchetypeConfig{}, fmt.Errorf("archetype writer: read prior row: %w", scanErr)
	} else if scanErr == nil {
		beforeJSON = prev
	}

	newVersion := prevVersion + 1
	if newVersion < 1 {
		newVersion = 1
	}

	// UPSERT the row.
	allowlistJSON, _ := json.Marshal(update.ToolAllowlist)
	deferJSON, _ := json.Marshal(update.DeferToolNames)
	var modelArg any
	if update.Model != nil {
		modelArg = *update.Model
	}
	if _, err := tx.ExecContext(ctx, `
		INSERT INTO archetype_configurations
			(archetype_kind, org_id, system_prompt, tool_allowlist, defer_tool_names, model, version, updated_at, updated_by)
		VALUES ($1, $2, $3, $4::jsonb, $5::jsonb, $6, $7, now(), $8)
		ON CONFLICT (archetype_kind, org_id) DO UPDATE SET
			system_prompt    = EXCLUDED.system_prompt,
			tool_allowlist   = EXCLUDED.tool_allowlist,
			defer_tool_names = EXCLUDED.defer_tool_names,
			model            = EXCLUDED.model,
			version          = EXCLUDED.version,
			updated_at       = now(),
			updated_by       = EXCLUDED.updated_by
	`, string(kind), orgID, prompt, allowlistJSON, deferJSON, modelArg, newVersion, actor); err != nil {
		return ArchetypeConfig{}, fmt.Errorf("archetype writer: upsert config: %w", err)
	}

	// Build `after` JSON from the just-written values + server-set
	// updated_at/updated_by (read back so the audit log captures the
	// canonical row, not a synthesised copy).
	var written ArchetypeConfig
	if err := tx.QueryRowContext(ctx, `
		SELECT archetype_kind, org_id, system_prompt, tool_allowlist, defer_tool_names, model, version, updated_at, updated_by
		FROM archetype_configurations WHERE archetype_kind = $1 AND org_id = $2
	`, string(kind), orgID).Scan(
		&written.Kind, &written.OrgID, &written.SystemPrompt, &allowlistJSON, &deferJSON,
		&modelArg, &written.Version, &written.UpdatedAt, &written.UpdatedBy,
	); err != nil {
		return ArchetypeConfig{}, fmt.Errorf("archetype writer: read back row: %w", err)
	}
	if err := json.Unmarshal(allowlistJSON, &written.ToolAllowlist); err != nil {
		return ArchetypeConfig{}, fmt.Errorf("archetype writer: decode written allowlist: %w", err)
	}
	if err := json.Unmarshal(deferJSON, &written.DeferToolNames); err != nil {
		return ArchetypeConfig{}, fmt.Errorf("archetype writer: decode written defer: %w", err)
	}
	if modelArg != nil {
		s, ok := modelArg.(string)
		if ok {
			written.Model = &s
		}
	}

	afterJSON, err := json.Marshal(written)
	if err != nil {
		return ArchetypeConfig{}, fmt.Errorf("archetype writer: marshal after: %w", err)
	}

	// Append the audit log row.
	var beforeArg any
	if beforeJSON != nil {
		beforeArg = beforeJSON
	}
	if _, err := tx.ExecContext(ctx, `
		INSERT INTO archetype_configurations_log
			(archetype_kind, org_id, actor, before, after)
		VALUES ($1, $2, $3, $4::jsonb, $5::jsonb)
	`, string(kind), orgID, actor, beforeArg, afterJSON); err != nil {
		return ArchetypeConfig{}, fmt.Errorf("archetype writer: append log: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return ArchetypeConfig{}, fmt.Errorf("archetype writer: commit: %w", err)
	}
	committed = true
	return written, nil
}
