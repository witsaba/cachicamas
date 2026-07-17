// Package domain — Skill aggregate for the 2026-07-17-skills-foundational
// change. Mirrors the prompts aggregate (prompt.go) shape 1:1 with three
// deltas, per design §3.1:
//
//  1. Name regex matches the agentskills.io spec, NOT the prompts spec.
//     Per engram obs #1964 the correct pattern is
//     `^[a-z0-9]+(-[a-z0-9]+)*$`, length 1..64 (the prompts regex
//     `^[a-z0-9][a-z0-9-]{0,98}[a-z0-9]$` would reject one-char names
//     and accept invalid consecutive hyphens).
//  2. Description cap is 1024 chars, NOT 280 (agentskills.io spec;
//     engram obs #1962 decision #4).
//  3. Body MUST begin with YAML frontmatter; `ParseFrontmatter` parses
//     the metadata block and `LockStepCheck` enforces consistency
//     between the parsed `name`/`description` and the request fields.
//     The parser is `gopkg.in/yaml.v3` (added to go.mod in PR1a per
//     engram obs #1963).
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
// CodeValidation, CodeNotFound, CodeConflict, CodeServer are reused from
// organization.go. The new items Skills introduce are CodeSkillDeleted
// (410) and ResourceSkill (identifies which entity a NotFoundError is about).
// ---------------------------------------------------------------------------

// CodeSkillDeleted is the wire code for HTTP 410 (skill exists but is
// soft-deleted). New for skills — no existing code in organization.go
// covers 410.
const CodeSkillDeleted = "skill_deleted"

// ResourceSkill is the NotFoundError.Resource value for "skill" lookups
// (mirrors ResourcePrompt).
const ResourceSkill = "skill"

// SkillGoneError signals that a skill row exists but is in a terminal
// state that disallows the requested operation. Mirror of prompts.GoneError;
// dedicated type keeps the 410 mapping deterministic (vs inferring from
// message text). Reads hide the row (→ NotFoundError → 404); updates
// and restores reject with GoneError → 410.
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

func (e *SkillGoneError) Code() string { return CodeSkillDeleted }

func (e *SkillGoneError) Unwrap() error { return e.Cause }

func NewSkillDeleted(name string) *SkillGoneError {
	return &SkillGoneError{Name: name}
}

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

	// User-facing messages.
	MsgSkillNotFound         = "Skill not found."
	MsgSkillRevisionNotFound = "Skill revision not found."
	MsgSkillDeleted          = "This skill has been deleted and cannot be modified."
	MsgSkillConflict         = "This skill name is already taken. Try another."
)

// ---------------------------------------------------------------------------
// Entity types (spec INV-1, INV-2; mirrors prompt.go shape).
// ---------------------------------------------------------------------------

// Skill is one row of the `skill` table (DDL:
// migration/sql/20260717120000_skills.sql). db tags drive the pgx
// scan; json tags drive wire serialization.
type Skill struct {
	ID          int64      `db:"id"          json:"id"`
	Name        string     `db:"name"        json:"name"`
	Description string     `db:"description" json:"description"`
	Body        string     `db:"body"        json:"body"`
	DeletedAt   *time.Time `db:"deleted_at"  json:"deleted_at"`
	CreatedAt   time.Time  `db:"created_at"  json:"created_at"`
	UpdatedAt   time.Time  `db:"updated_at"  json:"updated_at"`
}

// SkillRevision: one row of `skill_revision`. Append-only (never
// UPDATE, never DELETE except via CASCADE when parent is hard-deleted).
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
	// Normalize CRLF → LF so Windows-authored files don't fail fence checks.
	normalized := strings.ReplaceAll(body, "\r\n", "\n")
	if !strings.HasPrefix(normalized, skillFrontmatterFenceOpen) {
		return Frontmatter{}, &ValidationError{Fields: map[string]string{"body": MsgSkillFrontmatterMissing}}
	}
	rest := normalized[len(skillFrontmatterFenceOpen):]
	// Accept canonical "\n---\n" and "\n---" at EOF (no markdown body).
	var closeIdx int
	if i := strings.Index(rest, skillFrontmatterFenceClose); i >= 0 {
		closeIdx = i
	} else if strings.HasSuffix(rest, "\n---") {
		closeIdx = len(rest) - len("\n---")
	} else {
		return Frontmatter{}, &ValidationError{Fields: map[string]string{"body": MsgSkillFrontmatterUnterminated}}
	}
	yamlBlock := rest[:closeIdx]

	var raw struct {
		Name        interface{} `yaml:"name"`
		Description interface{} `yaml:"description"`
	}
	if err := yaml.Unmarshal([]byte(yamlBlock), &raw); err != nil {
		return Frontmatter{}, &ValidationError{Fields: map[string]string{"body": MsgSkillFrontmatterMalformed}}
	}

	nameStr, ok := raw.Name.(string)
	if !ok {
		if raw.Name == nil {
			return Frontmatter{}, &ValidationError{Fields: map[string]string{"body": MsgSkillFrontmatterNameMissing}}
		}
		return Frontmatter{}, &ValidationError{Fields: map[string]string{"body": MsgSkillFrontmatterNameNotString}}
	}
	descStr, ok := raw.Description.(string)
	if !ok {
		if raw.Description == nil {
			return Frontmatter{}, &ValidationError{Fields: map[string]string{"body": MsgSkillFrontmatterDescMissing}}
		}
		return Frontmatter{}, &ValidationError{Fields: map[string]string{"body": MsgSkillFrontmatterDescNotString}}
	}

	return Frontmatter{Name: nameStr, Description: descStr}, nil
}

// LockStepCheck enforces frontmatter-vs-request consistency
// (spec S-SK-035/036). TrimSpace on both sides (design risk R7).
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
// Skill list / detail DTOs.
//
// SkillDetail carries current_revision (anti-drift gate from obs
// #1959 #2 — the prompts gotcha where v{undefined} rendered because
// the field was undefined at runtime).
// ---------------------------------------------------------------------------

// SkillDetail: response shape for a single skill (current state +
// current revision number, joined at the DB level).
type SkillDetail struct {
	Skill
	CurrentRevision int `db:"current_revision" json:"current_revision"`
}

// SkillListItem: trimmed shape for GET /skills.
type SkillListItem struct {
	Name            string    `db:"name"             json:"name"`
	Description     string    `db:"description"      json:"description"`
	CurrentRevision int       `db:"current_revision" json:"current_revision"`
	UpdatedAt       time.Time `db:"updated_at"       json:"updated_at"`
}

// SkillRevisionItem: one entry in GET /skills/:name/revisions. Body
// intentionally omitted (can be large).
type SkillRevisionItem struct {
	RevisionNumber int       `db:"revision_number" json:"revision_number"`
	Description    string    `db:"description"     json:"description"`
	ChangeNote     *string   `db:"change_note"     json:"change_note"`
	CreatedAt      time.Time `db:"created_at"      json:"created_at"`
}

// ---------------------------------------------------------------------------
// Repository ports (hexagonal boundary; implementations land in PR1b
// under src/infrastructure/postgres/skills/). sqlExecutor is the
// shared *sql.DB / *sql.Tx contract reused from prompt.go (spec
// SCN-6.2 keeps pgx out of domain).
// ---------------------------------------------------------------------------

// SkillRepository reads/persists Skill rows. Application layer
// depends only on this interface. Contract (PR1b):
//   - Insert(ctx, db, s): sets s.ID, s.CreatedAt, s.UpdatedAt from DB clock.
//   - SelectBySlug: active-only; *NotFoundError on miss.
//   - SelectBySlugAny: any state; caller inspects DeletedAt.
//   - SelectByID: any state by PK; *NotFoundError on miss.
//   - List: active rows ordered by updated_at DESC.
//   - ListWithCurrentRevision: SkillListItem joined with MAX(revision_number).
//   - UpdateBody: updates fields + updated_at from DB clock.
//   - SoftDelete: sets deleted_at = now(); idempotent.
//   - LockAndLoad: SELECT … FOR UPDATE (caller MUST be in a transaction).
//   - MaxRevisionNumber: COALESCE(MAX(revision_number), 0) under FOR UPDATE.
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

// SkillRevisionRepository: reads/persists SkillRevision rows.
// Contract (PR1b):
//   - Insert: sets r.ID, r.CreatedAt from DB clock.
//   - SelectBySkillAndNumber: *NotFoundError on miss.
//   - ListBySkillID: ordered by revision_number DESC.
type SkillRevisionRepository interface {
	Insert(ctx context.Context, db sqlExecutor, r *SkillRevision) error
	SelectBySkillAndNumber(ctx context.Context, db sqlExecutor, skillID int64, n int) (*SkillRevision, error)
	ListBySkillID(ctx context.Context, db sqlExecutor, skillID int64) ([]*SkillRevision, error)
}
