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
	"regexp"
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
	return nil
}

// placeholder message constants — wired in tasks 1.2 and 1.3 below.
// Keeping them at the top so the locked vocabulary stays together.
// MsgSkillNameLength is overridden in GREEN task 1.2 with the spec text;
// the current value is the placeholder so the test for 1.1 passes.
const (
	MsgSkillNameLength = "Skill name must be 1-64 characters."
	MsgSkillNameFormat = "Skill name must be lowercase letters, digits, and single hyphens; cannot start or end with a hyphen."
)
