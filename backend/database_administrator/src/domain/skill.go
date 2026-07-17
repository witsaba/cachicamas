// Package domain — Skill aggregate for the 2026-07-17-skills-foundational
// change. Mirrors the prompts aggregate (prompt.go) shape 1:1 with three
// deltas, per design §3.1:
//
//   1. Name regex matches the agentskills.io spec, NOT the prompts spec.
//      Per engram obs #1964 the correct pattern is
//      `^[a-z0-9]+(-[a-z0-9]+)*$`, length 1..64 (the prompts regex
//      `^[a-z0-9][a-z0-9-]{0,98}[a-z0-9]$` would reject one-char names
//      and accept invalid consecutive hyphens).
//
//   2. Description cap is 1024 chars, NOT 280 (agentskills.io spec;
//      engram obs #1962 decision #4).
//
//   3. Body MUST begin with YAML frontmatter; `ParseFrontmatter` parses
//      the metadata block and `LockStepCheck` enforces consistency
//      between the parsed `name`/`description` and the request fields.
//      The parser is `gopkg.in/yaml.v3` (added to go.mod in PR1a per
//      engram obs #1963).
//
// This file lands as the GREEN companion of skill_test.go. The pattern
// reuses ValidationError / ConflictError / NotFoundError / CodeServer
// from organization.go and adds the new GoneError type for the 410 case
// (the locked wire code is `skill_deleted`, per engram obs #1959 / #1967).
//
// The domain layer has no infrastructure imports — spec S-SK-029
// (engram obs #1967) is enforced by `domain/imports_test.go` and by the
// scanner at the bottom of skill_test.go.
package domain

import (
	"bytes"
	"context"
	"regexp"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

// ---------------------------------------------------------------------------
// Constants (spec INV-1, INV-2; engram obs #1962 decisions #1, #4, #7).
// ---------------------------------------------------------------------------

const (
	// MinSkillNameLen is the lower bound on the skill name (1 char
	// per agentskills.io spec — overrides the prompts-style 2-char
	// minimum).
	MinSkillNameLen = 1

	// MaxSkillNameLen is the upper bound on the skill name (64 chars
	// per agentskills.io spec — overrides the prompts-style 100-char
	// maximum).
	MaxSkillNameLen = 64

	// MaxSkillDescriptionLen is the upper bound on skill description
	// (1024 chars per agentskills.io spec — overrides the prompts
	// 280 cap).
	MaxSkillDescriptionLen = 1024

	// MaxSkillBodyLen is the upper bound on the skill markdown body
	// (512 KB — same as prompts).
	MaxSkillBodyLen = 524288
)

// skillNameRegex matches the agentskills.io name constraint: one or
// more lowercase-alphanumeric groups separated by single hyphens, no
// leading/trailing/consecutive hyphens. Identical to the SQL CHECK
// constraint in 20260717120000_skills.sql — single source of truth.
var skillNameRegex = regexp.MustCompile(`^[a-z0-9]+(-[a-z0-9]+)*$`)

// ---------------------------------------------------------------------------
// ValidateSkillName (spec S-SK-024..027 + S-SK-001..003).
//
// Length check first (cheap), then regex shape, then reserved-word
// substring check (case-insensitive). Returns *ValidationError with a
// Fields map keyed by the wire name "name". The handler maps this to
// HTTP 400 with code "validation" via the existing envelope.
// ---------------------------------------------------------------------------

// ValidateSkillBody returns a *ValidationError if body is empty or
// longer than MaxSkillBodyLen (524288 bytes — same as prompts).
// Counts RUNES, not bytes, matching the description validator.
func ValidateSkillBody(body string) error {
	n := len([]rune(body))
	if n < 1 || n > MaxSkillBodyLen {
		return &ValidationError{
			Fields: map[string]string{"body": MsgSkillBodyTooLarge},
		}
	}
	return nil
}

// ValidateSkillDescription returns a *ValidationError if description
// is empty or longer than MaxSkillDescriptionLen (1024 chars per
// agentskills.io). Counts RUNES, not bytes, so multi-byte unicode is
// counted faithfully.
func ValidateSkillDescription(desc string) error {
	n := len([]rune(desc))
	if n < 1 || n > MaxSkillDescriptionLen {
		return &ValidationError{
			Fields: map[string]string{"description": MsgSkillDescriptionInvalid},
		}
	}
	return nil
}

// ValidateSkillName returns a *ValidationError if name does not match
// the agentskills.io pattern (^[a-z0-9]+(-[a-z0-9]+)*$, length 1..64)
// or contains a reserved substring (case-insensitive: "anthropic",
// "claude"). Returns nil when the name is acceptable.
func ValidateSkillName(name string) error {
	if n := len(name); n < MinSkillNameLen || n > MaxSkillNameLen {
		return &ValidationError{
			Fields: map[string]string{"name": MsgSkillNameLength},
		}
	}
	if !skillNameRegex.MatchString(name) {
		return &ValidationError{
			Fields: map[string]string{"name": MsgSkillNameFormat},
		}
	}
	// Reserved-word check (case-insensitive substring). The strings
	// "anthropic" and "claude" are reserved per agentskills.io; we
	// compare against the lowercased name to defeat case obfuscation
	// like "aNthropic" or "cLaude-helper".
	lowered := strings.ToLower(name)
	for _, reserved := range skillReservedSubstrings {
		if strings.Contains(lowered, reserved) {
			return &ValidationError{
				Fields: map[string]string{"name": MsgSkillNameReserved},
			}
		}
	}
	return nil
}

// skillReservedSubstrings is the closed set of substrings that
// agentskills.io treats as reserved (case-insensitive). The list is
// small and unlikely to grow; extending it would require a review of
// every existing skill name in production.
var skillReservedSubstrings = []string{"anthropic", "claude"}

// ---------------------------------------------------------------------------
// Locked wire vocabulary for the skills feature (spec §7).
//
// CodeValidation, CodeNotFound, CodeConflict, CodeServer are reused
// from organization.go (no new 4xx/5xx type definitions needed). The
// new vocabulary items introduced by Skills are:
//
//   - CodeSkillDeleted = "skill_deleted"  — locked code for the 410
//     Gone response (mirrors CodePromptDeleted).
//   - ResourceSkill    = "skill"          — used with NotFoundError to
//     identify which entity was missing.
// ---------------------------------------------------------------------------

// CodeSkillDeleted is the wire code returned when an operation
// targets a soft-deleted skill. The handler maps GoneError to HTTP
// 410. No existing code in organization.go covers 410, so this code
// is new for the skills feature.
const CodeSkillDeleted = "skill_deleted"

// ResourceSkill is the value used in NotFoundError.Resource for
// "skill" lookups; the value is also reused by the prompts feature
// (ResourcePrompt) so the handler can tell which entity type a
// not-found error refers to.
const ResourceSkill = "skill"

// MsgSkillNotFound is the user-facing message when no active skill
// matches the requested name.
const MsgSkillNotFound = "Skill not found."

// MsgSkillRevisionNotFound is the user-facing message when the
// requested revision number does not exist for the skill.
const MsgSkillRevisionNotFound = "Skill revision not found."

// MsgSkillDeleted is the user-facing message when an operation
// targets a soft-deleted skill.
const MsgSkillDeleted = "This skill has been deleted and cannot be modified."

// MsgSkillConflict is the user-facing message when the name collides
// with another active skill.
const MsgSkillConflict = "This skill name is already taken. Try another."

// ---------------------------------------------------------------------------
// SkillGoneError — the skills-specific 410 case.
//
// Mirror of prompts.GoneError. The row exists but is soft-deleted;
// reads hide it (returns NotFoundError → 404); updates and restores
// reject with GoneError → 410. Reusing NotFoundError would conflate
// the two and force the handler to infer 410 from message text; a
// dedicated type keeps the HTTP mapping deterministic.
// ---------------------------------------------------------------------------

// SkillGoneError signals that a skill row exists but is in a
// terminal state that disallows the requested operation (e.g.,
// soft-deleted). The handler maps it to HTTP 410 with code
// `skill_deleted`.
type SkillGoneError struct {
	Name  string
	Cause error
}

func (e *SkillGoneError) Error() string {
	if e.Name == "" {
		return MsgSkillDeleted
	}
	return "skill name=" + e.Name + ": " + MsgSkillDeleted
}

// Code returns the locked wire code for HTTP 410 mapping
// ("skill_deleted"). Handler maps to 410 via the AppError interface.
func (e *SkillGoneError) Code() string { return CodeSkillDeleted }

func (e *SkillGoneError) Unwrap() error { return e.Cause }

// NewSkillDeleted returns a *SkillGoneError for a soft-deleted skill
// that the caller tried to modify or restore.
func NewSkillDeleted(name string) *SkillGoneError {
	return &SkillGoneError{Name: name}
}

// placeholder message constants — wired in tasks 1.2 and 1.3 below.
// Keeping them at the top so the locked vocabulary stays together.
// MsgSkillNameLength is overridden in GREEN task 1.2 with the spec text;
// the current value is the placeholder so the test for 1.1 passes.
const (
	MsgSkillNameLength         = "Skill name must be 1-64 characters."
	MsgSkillNameFormat         = "Skill name must be lowercase letters, digits, and single hyphens; cannot start or end with a hyphen."
	MsgSkillNameReserved       = "Skill name cannot contain \"anthropic\" or \"claude\"."
	MsgSkillDescriptionInvalid = "Skill description must be 1-1024 characters."
	MsgSkillBodyTooLarge       = "Skill body must be 1-524288 characters."

	// Frontmatter parser messages (spec SCN-3.5, SCN-3.6).
	MsgSkillFrontmatterMissing       = "Skill body must start with YAML frontmatter (---)."
	MsgSkillFrontmatterUnterminated  = "Skill body frontmatter is missing the closing --- fence."
	MsgSkillFrontmatterMalformed     = "Skill body frontmatter is not valid YAML."
	MsgSkillFrontmatterNameMissing   = "Skill body frontmatter is missing the required 'name' key."
	MsgSkillFrontmatterDescMissing   = "Skill body frontmatter is missing the required 'description' key."
	MsgSkillFrontmatterNameNotString = "Skill body frontmatter 'name' must be a string."
	MsgSkillFrontmatterDescNotString = "Skill body frontmatter 'description' must be a string."

	// Lock-step (spec S-SK-035, S-SK-036).
	MsgSkillNameLockStep        = "Skill body frontmatter 'name' must match the URL slug."
	MsgSkillDescriptionLockStep = "Skill body frontmatter 'description' must match the request description."
)

// ---------------------------------------------------------------------------
// Entity types (spec INV-1, INV-2; mirrors prompt.go shape).
// ---------------------------------------------------------------------------

// Skill is the domain entity for one row of the `skill` table (DDL:
// migration/sql/20260717120000_skills.sql). The struct mirrors the
// DDL column-for-column; the `db` tags tell the pgx adapter which
// column each field reads from, the `json` tags control wire
// serialization.
type Skill struct {
	ID          int64      `db:"id"          json:"id"`
	Name        string     `db:"name"        json:"name"`
	Description string     `db:"description" json:"description"`
	Body        string     `db:"body"        json:"body"`
	DeletedAt   *time.Time `db:"deleted_at"  json:"deleted_at"`
	CreatedAt   time.Time  `db:"created_at"  json:"created_at"`
	UpdatedAt   time.Time  `db:"updated_at"  json:"updated_at"`
}

// SkillRevision is the domain entity for one row of the
// `skill_revision` table. Append-only: never UPDATE, never DELETE
// except via CASCADE when the parent skill is hard-deleted.
type SkillRevision struct {
	ID             int64     `db:"id"              json:"id"`
	SkillID        int64     `db:"skill_id"        json:"skill_id"`
	RevisionNumber int       `db:"revision_number" json:"revision_number"`
	Description    string    `db:"description"     json:"description"`
	Body           string    `db:"body"            json:"body"`
	ChangeNote     *string   `db:"change_note"     json:"change_note"`
	CreatedAt      time.Time `db:"created_at"      json:"created_at"`
}

// ---------------------------------------------------------------------------
// Frontmatter parsing (spec S-SK-031..035 — ANTI-DRIFT GATE).
//

// Frontmatter is the parsed subset of a SKILL.md YAML metadata block.
// Only the two required scalars are exported; everything else in the
// frontmatter is silently ignored.
type Frontmatter struct {
	Name        string
	Description string
}

// skillFrontmatterFenceOpen is the leading line that every SKILL.md
// body MUST start with. We check for the literal "---\n" sequence at
// byte offset 0 (no leading whitespace, no BOM).
const skillFrontmatterFenceOpen = "---\n"

// skillFrontmatterFenceClose is the closing fence that ends the
// frontmatter block. Always followed by '\n' (the line after the
// closing fence starts the markdown body, which may be empty).
const skillFrontmatterFenceClose = "\n---\n"

// ParseFrontmatter reads the YAML metadata block at the top of a
// SKILL.md body and returns the two required scalars
// (name, description). On any malformed input it returns a
// *ValidationError with a precise `fields.body` message; the function
// never panics.
//
// Returns a *ValidationError when:
//   - the body does not start with "---\n" (MsgSkillFrontmatterMissing)
//   - the closing fence is missing (MsgSkillFrontmatterUnterminated)
//   - the YAML fails to parse (MsgSkillFrontmatterMalformed)
//   - the `name` key is absent or not a string (MsgSkillFrontmatterNameMissing/NotString)
//   - the `description` key is absent or not a string (MsgSkillFrontmatterDescMissing/NotString)
func ParseFrontmatter(body string) (Frontmatter, error) {
	// Normalize CRLF -> LF so Windows-authored files don't fail the
	// fence checks below. We replace "\r\n" first to avoid leaving
	// stray "\r" in the YAML block (which the YAML parser happens to
	// tolerate but is brittle).
	normalized := strings.ReplaceAll(body, "\r\n", "\n")
	if !strings.HasPrefix(normalized, skillFrontmatterFenceOpen) {
		return Frontmatter{}, &ValidationError{
			Fields: map[string]string{"body": MsgSkillFrontmatterMissing},
		}
	}
	rest := normalized[len(skillFrontmatterFenceOpen):]
	// Locate the closing fence. We accept both the canonical
	// "\n---\n" (with a trailing newline into the markdown body) and
	// the "\n---" at end-of-file (no markdown after the fence). This
	// keeps metadata-only skills valid.
	var closeIdx int
	if i := strings.Index(rest, skillFrontmatterFenceClose); i >= 0 {
		closeIdx = i
	} else if strings.HasSuffix(rest, "\n---") {
		closeIdx = len(rest) - len("\n---")
	} else {
		return Frontmatter{}, &ValidationError{
			Fields: map[string]string{"body": MsgSkillFrontmatterUnterminated},
		}
	}
	yamlBlock := rest[:closeIdx]	

	var raw struct {
		Name        interface{} `yaml:"name"`
		Description interface{} `yaml:"description"`
	}
	if err := yaml.Unmarshal([]byte(yamlBlock), &raw); err != nil {
		return Frontmatter{}, &ValidationError{
			Fields: map[string]string{"body": MsgSkillFrontmatterMalformed},
		}
	}

	nameStr, ok := raw.Name.(string)
	if !ok {
		if raw.Name == nil {
			return Frontmatter{}, &ValidationError{
				Fields: map[string]string{"body": MsgSkillFrontmatterNameMissing},
			}
		}
		return Frontmatter{}, &ValidationError{
			Fields: map[string]string{"body": MsgSkillFrontmatterNameNotString},
		}
	}
	descStr, ok := raw.Description.(string)
	if !ok {
		if raw.Description == nil {
			return Frontmatter{}, &ValidationError{
				Fields: map[string]string{"body": MsgSkillFrontmatterDescMissing},
			}
		}
		return Frontmatter{}, &ValidationError{
			Fields: map[string]string{"body": MsgSkillFrontmatterDescNotString},
		}
	}

	return Frontmatter{Name: nameStr, Description: descStr}, nil
}

// Compile-time assertion that bytes is used (kept around in case
// future frontmatter parsing wants to read large bodies without
// allocating a string — currently unused).
var _ = bytes.NewBufferString

// LockStepCheck enforces frontmatter-vs-request consistency
// (spec S-SK-035/036, scenarios SCN-3.5). TrimSpace is applied to
// both sides so trailing newlines from copy-paste don't break
// legitimate edits (design risk R7).
//
// Returns a *ValidationError populated for whichever side failed:
//   - fields.name          when slug != fm.Name
//   - fields.description   when reqDescription != fm.Description
func LockStepCheck(slug, reqDescription string, fm Frontmatter) error {
	fields := map[string]string{}
	if strings.TrimSpace(slug) != strings.TrimSpace(fm.Name) {
		fields["name"] = MsgSkillNameLockStep
	}
	if strings.TrimSpace(reqDescription) != strings.TrimSpace(fm.Description) {
		fields["description"] = MsgSkillDescriptionLockStep
	}
	if len(fields) == 0 {
		return nil
	}
	return &ValidationError{Fields: fields}
}

// ---------------------------------------------------------------------------
// Skill list / detail DTOs (used by the application + HTTP layers).
//
// SkillDetail is what the handler returns from
// GET /skills/:name + POST /skills + PATCH /skills/:name + restore.
// It carries `current_revision` so the frontend never has to derive
// it (anti-drift gate from obs #1959 #2 — the prompts gotcha where
// `v{undefined}` rendered because the field was undefined at
// runtime).
//
// SkillListItem is the trimmed shape returned from GET /skills.
// SkillRevisionItem is one entry in GET /skills/:name/revisions.
// ---------------------------------------------------------------------------

// SkillDetail is the response shape for a single skill (current state
// + current revision number, joined at the DB level).
type SkillDetail struct {
	Skill
	CurrentRevision int `db:"current_revision" json:"current_revision"`
}

// SkillListItem is the trimmed shape for GET /skills (list view).
type SkillListItem struct {
	Name            string    `db:"name"             json:"name"`
	Description     string    `db:"description"      json:"description"`
	CurrentRevision int       `db:"current_revision" json:"current_revision"`
	UpdatedAt       time.Time `db:"updated_at"       json:"updated_at"`
}

// SkillRevisionItem is one entry in the GET /skills/:name/revisions
// list. The body is intentionally omitted (it can be large); callers
// that need a specific revision use the application service.
type SkillRevisionItem struct {
	RevisionNumber int       `db:"revision_number" json:"revision_number"`
	Description    string    `db:"description"     json:"description"`
	ChangeNote     *string   `db:"change_note"     json:"change_note"`
	CreatedAt      time.Time `db:"created_at"      json:"created_at"`
}

// ---------------------------------------------------------------------------
// Repository ports (hexagonal boundary). Implementations live under
// src/infrastructure/postgres/skills/ (PR1b). Interfaces are small so
// fakes are trivial in tests.
// ---------------------------------------------------------------------------

// sqlExecutor is satisfied by *sql.DB and *sql.Tx, so the service
// can pass either to the repo depending on whether the operation
// runs in a transaction. Defining it in the domain package keeps pgx
// out of the domain (spec SCN-6.2 — see also imports_test.go).
//
// SkillRepository uses this same SQLExecutor type as the prompts
// port (declared in prompt.go); defining it again here would
// duplicate the interface so we reuse the same name on the same
// underlying contract.

// SkillRepository is the port for reading and persisting Skill rows.
// The application layer depends only on this interface.
//
// Contract (implemented in PR1b):
//
//   - Insert(ctx, db, s) MUST set s.ID, s.CreatedAt, s.UpdatedAt from
//     the DB clock and return the populated entity.
//   - SelectBySlug(ctx, db, name) returns *NotFoundError with Code
//     CodeNotFound when no active row matches.
//   - SelectBySlugAny returns the row regardless of deleted_at;
//     callers inspect the DeletedAt pointer themselves.
//   - SelectByID returns *NotFoundError with Code CodeNotFound when
//     no row matches.
//   - List(ctx, db, limit) returns active rows ordered by updated_at
//     DESC; limit is clamped at the application layer (default 50,
//     hard cap 200).
//   - ListWithCurrentRevision(ctx, db, limit) returns SkillListItem
//     rows joined with MAX(revision_number).
//   - UpdateBody(ctx, db, id, description, body) updates fields and
//     updated_at from the DB clock.
//   - SoftDelete(ctx, db, id) sets deleted_at = now(); idempotent.
//   - LockAndLoad issues SELECT … FOR UPDATE; caller MUST be inside
//     a transaction.
//   - MaxRevisionNumber(ctx, db, skillID) returns
//     COALESCE(MAX(revision_number), 0) under the FOR UPDATE lock.
type SkillRepository interface {
	Insert(ctx context.Context, db sqlExecutor, s *Skill) error
	SelectBySlug(ctx context.Context, db sqlExecutor, name string) (*Skill, error)
	SelectBySlugAny(ctx context.Context, db sqlExecutor, name string) (*Skill, error)
	SelectByID(ctx context.Context, db sqlExecutor, id int64) (*Skill, error)
	List(ctx context.Context, db sqlExecutor, limit int) ([]*Skill, error)
	ListWithCurrentRevision(ctx context.Context, db sqlExecutor, limit int) ([]*SkillListItem, error)
	UpdateBody(ctx context.Context, db sqlExecutor, id int64, description, body string) error
	SoftDelete(ctx context.Context, db sqlExecutor, id int64) error
	LockAndLoad(ctx context.Context, db sqlExecutor, id int64) (*Skill, error)
	MaxRevisionNumber(ctx context.Context, db sqlExecutor, skillID int64) (int, error)
}

// SkillRevisionRepository is the port for reading and persisting
// SkillRevision rows.
//
// Contract (implemented in PR1b):
//
//   - Insert(ctx, db, r) MUST set r.ID, r.CreatedAt from the DB clock.
//   - SelectBySkillAndNumber(ctx, db, skillID, n) returns the
//     specific revision; *NotFoundError with Code CodeNotFound when
//     missing.
//   - ListBySkillID(ctx, db, skillID) returns rows ordered by
//     revision_number DESC.
type SkillRevisionRepository interface {
	Insert(ctx context.Context, db sqlExecutor, r *SkillRevision) error
	SelectBySkillAndNumber(ctx context.Context, db sqlExecutor, skillID int64, n int) (*SkillRevision, error)
	ListBySkillID(ctx context.Context, db sqlExecutor, skillID int64) ([]*SkillRevision, error)
}
