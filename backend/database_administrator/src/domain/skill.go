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
	"regexp"
	"strings"

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
// Frontmatter parsing (spec S-SK-031..035 — ANTI-DRIFT GATE).
//
// Parses the YAML metadata block at the top of a SKILL.md body. The
// parser is intentionally narrow: it accepts only `name` and
// `description` scalars (other keys are permitted and ignored), and
// any deviation from the expected shape surfaces as a *ValidationError
// so the handler maps it to HTTP 400 with `code = "validation"`.
//
// The parser uses gopkg.in/yaml.v3 (added to go.mod in PR1a per
// engram obs #1963) and never executes custom unmarshalers, anchors,
// merge keys, or any tag directive — defense in depth against YAML
// injection (risk R5 in design §7).
// ---------------------------------------------------------------------------

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
	if !strings.HasPrefix(body, skillFrontmatterFenceOpen) {
		return Frontmatter{}, &ValidationError{
			Fields: map[string]string{"body": MsgSkillFrontmatterMissing},
		}
	}
	rest := body[len(skillFrontmatterFenceOpen):]
	// Locate the closing fence from the start of `rest` so the YAML
	// block itself never contains a "\n---\n" sequence (a valid YAML
	// document cannot, but we assert defensively).
	closeIdx := strings.Index(rest, skillFrontmatterFenceClose)
	if closeIdx < 0 {
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
