// Package domain_test contains the TDD-locked unit tests for the
// skills feature. Every assertion below corresponds to a spec scenario
// in engram obs #1967 (sdd/cachicamas-skills-foundational/spec).
//
// This file MUST land before the production code it tests (RED step
// in strict TDD). The production file is domain/skill.go.
//
// Naming convention: each test function asserts one behavioural rule
// with a `_Accepts_` or `_Rejects_` infix that mirrors the spec
// scenarios (e.g. S-SK-024..036). The cases stay deterministic
// because the domain layer has zero infrastructure dependencies.
package domain_test

import (
	"strings"
	"testing"

	"github.com/cachicamas/backend/database_administrator/src/domain"
)

// ---------------------------------------------------------------------------
// ValidateName (spec S-SK-024..027, S-SK-001..003).
//
// Regex: ^[a-z0-9]+(-[a-z0-9]+)*$  (agentskills.io spec).
// Length: 1..64 characters.
// Reserved substrings (case-insensitive): "anthropic", "claude".
// ---------------------------------------------------------------------------

func TestSkillNameRegex_AcceptsValidNames(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name string
		in   string
	}{
		{"1char", "a"},
		{"64chars", strings.Repeat("a", 64)},
		{"alphanumerics", "pdfcleanup2026"},
		{"singleHyphenMiddle", "pdf-cleanup"},
		{"manyHyphens", "pdf-cleanup-2026-v1"},
		{"pureDigits", "12345"},
	}
	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			if err := domain.ValidateSkillName(tc.in); err != nil {
				t.Fatalf("expected nil for %q, got %v", tc.in, err)
			}
		})
	}
}
