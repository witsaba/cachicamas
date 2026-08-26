// Package chat — assistant_config.go is the chat-owned AssistantConfig
// storage layer (REQ-CACS-001/002/003, design AD-1/AD-2). It defines the
// in-memory `AssistantConfig` value type, the `Loader` port, the
// Postgres-backed implementation, the safe-default factory, and the
// validation sentinels surfaced by the PUT endpoint (REQ-CACAPI-002/003).
//
// The Loader is a read-only port. Writes live in the API layer
// (`putAssistantConfigHandler` in http.go) and route through the
// append-only audit log. The Loader MUST NOT auto-write on absent row —
// REQ-CACS-003 forbids it. Defaults are in-memory only.
package chat

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"
)

// AssistantConfig is the persisted chat archetype configuration for one
// org. Stored as one row per `org_id` in `chat_assistant_config`.
//
// Field provenance:
//   - SystemPrompt: configurable; default is the chat v1 literal at
//     `DefaultSystemPrompt`.
//   - ToolAllowlist: persisted names; toggled via PUT. Implementations
//     stay in code (current_time.go, summarize_conversation.go).
//   - DeferToolNames: subset of ToolAllowlist that requires permission
//     approval before each invocation.
//   - Model: optional informational field; the actual model selection
//     remains env-driven (process-wide).
//   - Version: monotonic; bumped on every successful PUT. Used by the
//     `*Conversation.version` propagation contract (REQ-CCVP-001/002).
//   - UpdatedAt/UpdatedBy: server-set on every successful PUT.
type AssistantConfig struct {
	OrgID          string    `json:"org_id"`
	SystemPrompt   string    `json:"system_prompt"`
	ToolAllowlist  []string  `json:"tool_allowlist"`
	DeferToolNames []string  `json:"defer_tool_names"`
	Model          *string   `json:"model,omitempty"`
	Version        int       `json:"version"`
	UpdatedBy      string    `json:"updated_by,omitempty"`
	UpdatedAt      time.Time `json:"updated_at"`
}

// DefaultSystemPrompt is the chat archetype's v1 system prompt. Kept
// exported so the Loader's safe-default factory can reuse the literal
// without copy-paste drift.
const DefaultSystemPrompt = "You are the cachicamas chat assistant; answer the participant in plain, well-formatted text."

// DefaultDeferToolNames is the safe-default defer set. Tools whose name
// appears here require explicit permission approval before each call.
// `summarize_conversation` mutates chat state, so it is deferred by
// default.
var DefaultDeferToolNames = []string{"summarize_conversation"}

// MaxSystemPromptLength caps the persisted system prompt. Mirrors the
// 4000-char cap on the front-end Configure section (REQ-FACS-002).
const MaxSystemPromptLength = 4000

// Validation sentinels (REQ-CACAPI-002/003). Each is distinct under
// `errors.Is` so the handler can map to specific 400 responses without
// string-matching.
var (
	// ErrSystemPromptEmpty is returned when the prompt body is empty
	// after trimming whitespace.
	ErrSystemPromptEmpty = errors.New("assistant config: system_prompt is empty")
	// ErrSystemPromptTooLong is returned when the prompt exceeds
	// MaxSystemPromptLength.
	ErrSystemPromptTooLong = errors.New("assistant config: system_prompt exceeds maximum length")
	// ErrSystemPromptContainsHTML is returned when the prompt body
	// contains a `<script` or `<iframe` substring (case-insensitive).
	// Defensive guard against stored XSS; the prompt is interpolated
	// verbatim into the harness and would be rendered if the LLM ever
	// echoed it.
	ErrSystemPromptContainsHTML = errors.New("assistant config: system_prompt contains disallowed HTML pattern")
	// ErrToolAllowlistEmpty is returned when tool_allowlist is the empty
	// slice. The Assistant is always allowed to use at least one tool.
	ErrToolAllowlistEmpty = errors.New("assistant config: tool_allowlist must contain at least one tool")
	// ErrUnknownToolName is returned when tool_allowlist contains a name
	// not in the registered tool set. The list of registered names is
	// the `ToolSource.RegisteredNames()` set supplied at composition.
	ErrUnknownToolName = errors.New("assistant config: tool_allowlist contains a name not in the registered tool set")
	// ErrDeferToolNotInAllowlist is returned when defer_tool_names
	// contains a name not present in tool_allowlist. Defer is a subset
	// of allow.
	ErrDeferToolNotInAllowlist = errors.New("assistant config: defer_tool_names must be a subset of tool_allowlist")
)

// Loader is the read port for AssistantConfig. The Postgres adapter is
// the only shipped implementation; in-memory fakes can be added for
// tests by implementing this interface.
//
// LoadByOrg returns a value (not a pointer) so callers can compare the
// returned AssistantConfig to a declared-zero-value safely. The
// safe-default path returns DefaultAssistantConfig(orgID, knownTools)
// by value, never nil; the absent-row signal is the bool second return.
type Loader interface {
	// LoadByOrg returns the persisted config for the given org. When
	// the row is absent, it returns the safe-default config (built from
	// `DefaultAssistantConfig(orgID, knownTools)`), found=false, and
	// MUST NOT auto-write — REQ-CACS-003.
	LoadByOrg(ctx context.Context, orgID string) (AssistantConfig, bool, error)

	// WithTx returns a Loader that issues its read inside the supplied
	// transaction. Used by the FOR SHARE serialisation contract (design
	// AD-2): when caller A holds LoadByOrg inside a tx, concurrent
	// writer B is blocked by Postgres's row-level shared lock until A
	// commits.
	WithTx(tx *sql.Tx) Loader
}

// PostgresLoader is the Postgres-backed implementation of Loader. Holds
// either a *sql.DB (default) or a *sql.Tx (after WithTx). The two are
// mutually exclusive per Loader instance.
type PostgresLoader struct {
	db *sql.DB
	tx *sql.Tx
}

// NewPostgresAssistantConfigLoader returns a Loader that reads from the
// supplied *sql.DB pool. The pool is the chat archetype's migration
// runner pool (CACHICAMAS_CHAT_STORE_DSN).
func NewPostgresAssistantConfigLoader(db *sql.DB) Loader {
	return &PostgresLoader{db: db}
}

// WithTx returns a Loader bound to the supplied transaction. The
// returned Loader's LoadByOrg issues a SELECT ... FOR SHARE so that
// concurrent UPDATE statements on the same row block until the tx
// commits or rolls back.
func (l *PostgresLoader) WithTx(tx *sql.Tx) Loader {
	return &PostgresLoader{tx: tx}
}

// LoadByOrg reads `chat_assistant_config` for the supplied orgID. On a
// tx-bound Loader the query uses `FOR SHARE`; on a db-bound Loader it
// uses a plain `SELECT` (no row lock acquired).
//
// Behaviour matrix:
//   - row present     → returns row, found=true,  err=nil
//   - row absent      → returns safe-default, found=false, err=nil
//   - query/scan fail → returns zero-value, false, err
//
// The Loader MUST NOT auto-write the safe default. Persistence is the
// API layer's responsibility (REQ-CACS-003).
func (l *PostgresLoader) LoadByOrg(ctx context.Context, orgID string) (AssistantConfig, bool, error) {
	if strings.TrimSpace(orgID) == "" {
		return AssistantConfig{}, false, errors.New("assistant config loader: orgID must be non-empty")
	}
	const baseSelect = `SELECT org_id, system_prompt, tool_allowlist, defer_tool_names, model, version, updated_at, updated_by FROM chat_assistant_config WHERE org_id = $1`
	query := baseSelect
	if l.tx != nil {
		query = baseSelect + ` FOR SHARE`
	}
	var (
		row          *sql.Row
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
		row = l.tx.QueryRowContext(ctx, query, orgID)
	} else {
		row = l.db.QueryRowContext(ctx, query, orgID)
	}
	if err := row.Scan(&gotOrgID, &gotPrompt, &rawAllowlist, &rawDefers, &gotModel, &gotVersion, &gotUpdatedAt, &gotUpdatedBy); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			known := knownToolNames(ctx)
			return DefaultAssistantConfig(orgID, known), false, nil
		}
		return AssistantConfig{}, false, fmt.Errorf("assistant config loader: scan row for org=%q: %w", orgID, err)
	}

	var allowlist []string
	if len(rawAllowlist) > 0 {
		if err := json.Unmarshal(rawAllowlist, &allowlist); err != nil {
			return AssistantConfig{}, false, fmt.Errorf("assistant config loader: decode tool_allowlist for org=%q: %w", orgID, err)
		}
	}
	var defers []string
	if len(rawDefers) > 0 {
		if err := json.Unmarshal(rawDefers, &defers); err != nil {
			return AssistantConfig{}, false, fmt.Errorf("assistant config loader: decode defer_tool_names for org=%q: %w", orgID, err)
		}
	}

	cfg := AssistantConfig{
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

// DefaultAssistantConfig is the safe-default factory (REQ-CACS-003,
// design AD-2). Pure: same input → same output, no I/O. The `knownTools`
// argument is the registered tool set supplied by the composition root;
// the defaults only allow tools that actually exist, so the first PUT
// cannot bypass the registry check by listing unknown names.
//
// Returns a value (not a pointer) so callers — including the Loader's
// absent-row path — can return it directly without dereferencing.
func DefaultAssistantConfig(orgID string, knownTools []string) AssistantConfig {
	allowlist := append([]string(nil), knownTools...)
	defers := append([]string(nil), DefaultDeferToolNames...)
	return AssistantConfig{
		OrgID:          orgID,
		SystemPrompt:   DefaultSystemPrompt,
		ToolAllowlist:  allowlist,
		DeferToolNames: defers,
		Model:          nil,
		Version:        1,
		UpdatedBy:      "",
		UpdatedAt:      time.Time{},
	}
}

// knownToolNames returns the registered tool names from the chat
// archetype's tool source. The composition root wires the registry; the
// helper consults a package-level set when set (set by main.go after
// construction) and falls back to the hardcoded current default
// otherwise. The fallback keeps the Loader usable in tests and in any
// composition that does not call SetRegisteredToolNames.
//
// Set via `SetRegisteredToolNames` from cmd/chat/main.go at composition
// time.
var registeredToolNames = func() []string {
	return []string{"current_time", "summarize_conversation"}
}

// SetRegisteredToolNames overrides the package-level set used by
// DefaultAssistantConfig and the Loader's absent-row fallback. Called
// once from cmd/chat/main.go after the ToolSource registry is built.
func SetRegisteredToolNames(names []string) {
	copied := append([]string(nil), names...)
	registeredToolNames = func() []string {
		return copied
	}
}

// knownToolNames is the closure-returning helper used by LoadByOrg.
func knownToolNames(_ context.Context) []string {
	return registeredToolNames()
}
