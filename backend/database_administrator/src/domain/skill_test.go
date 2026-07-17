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
// ParseFrontmatter (spec S-SK-031..035 — ANTI-DRIFT GATE).
//
// Each frontmatter test below corresponds to a real failure mode the
// parser MUST surface as a *ValidationError. Frontmatter shape is the
// critical gateway: a misread here means the rest of the contract is
// unenforceable (lock-step, reserved-word, etc).
// ---------------------------------------------------------------------------

func TestParseFrontmatter_RejectsMissingFence(t *testing.T) {
	t.Parallel()
	// No leading "---"; body starts with markdown.
	body := "name: pdf-cleanup\ndescription: d\n---\n\n# Markdown only\n"
	if _, err := domain.ParseFrontmatter(body); err == nil {
		t.Fatalf("expected error for missing leading fence, got nil")
	}
}

func TestParseFrontmatter_RejectsMalformedYAML(t *testing.T) {
	t.Parallel()
	// Valid fence markers but unparseable YAML inside.
	body := "---\nname: pdf-cleanup\n  description: [unterminated\n---\n"
	if _, err := domain.ParseFrontmatter(body); err == nil {
		t.Fatalf("expected error for malformed YAML, got nil")
	}
}

func TestParseFrontmatter_RejectsMissingName(t *testing.T) {
	t.Parallel()
	body := "---\ndescription: a description\n---\n"
	if _, err := domain.ParseFrontmatter(body); err == nil {
		t.Fatalf("expected error for missing name key, got nil")
	}
}

func TestParseFrontmatter_RejectsMissingDescription(t *testing.T) {
	t.Parallel()
	body := "---\nname: pdf-cleanup\n---\n"
	if _, err := domain.ParseFrontmatter(body); err == nil {
		t.Fatalf("expected error for missing description key, got nil")
	}
}

func TestParseFrontmatter_AcceptsValidFrontmatter(t *testing.T) {
	t.Parallel()
	body := "---\nname: pdf-cleanup\ndescription: A description.\n---\n\n# Heading\n\nMarkdown body.\n"
	fm, err := domain.ParseFrontmatter(body)
	if err != nil {
		t.Fatalf("expected nil for valid frontmatter, got %v", err)
	}
	if fm.Name != "pdf-cleanup" {
		t.Errorf("expected fm.Name = %q, got %q", "pdf-cleanup", fm.Name)
	}
	if fm.Description != "A description." {
		t.Errorf("expected fm.Description = %q, got %q", "A description.", fm.Description)
	}
}

func TestParseFrontmatter_AcceptsFrontmatterOnlyBody(t *testing.T) {
	t.Parallel()
	// No markdown after the closing fence — the spec allows this for
	// "metadata-only" skills.
	body := "---\nname: pdf-cleanup\ndescription: d\n---\n"
	fm, err := domain.ParseFrontmatter(body)
	if err != nil {
		t.Fatalf("expected nil for frontmatter-only body, got %v", err)
	}
	if fm.Name != "pdf-cleanup" {
		t.Errorf("expected fm.Name = %q, got %q", "pdf-cleanup", fm.Name)
	}
}

// ---------------------------------------------------------------------------
// ValidateSkillBody (spec S-SK-030, S-SK-031, S-SK-003).
// ---------------------------------------------------------------------------

func TestValidateSkillBody_AcceptsAtMaxLen(t *testing.T) {
	t.Parallel()
	b := strings.Repeat("x", domain.MaxSkillBodyLen)
	if err := domain.ValidateSkillBody(b); err != nil {
		t.Fatalf("expected nil for body at max length, got %v", err)
	}
}

func TestValidateSkillBody_RejectsOverMaxLen(t *testing.T) {
	t.Parallel()
	b := strings.Repeat("x", domain.MaxSkillBodyLen+1)
	if err := domain.ValidateSkillBody(b); err == nil {
		t.Fatalf("expected error for body over max length, got nil")
	}
}

func TestValidateSkillBody_RejectsEmpty(t *testing.T) {
	t.Parallel()
	if err := domain.ValidateSkillBody(""); err == nil {
		t.Fatalf("expected error for empty body, got nil")
	}
}

// ---------------------------------------------------------------------------
// ValidateSkillDescription (spec S-SK-028, S-SK-029, S-SK-003).
// ---------------------------------------------------------------------------

func TestValidateSkillDescription_Accepts1024Chars(t *testing.T) {
	t.Parallel()
	d := strings.Repeat("a", 1024)
	if err := domain.ValidateSkillDescription(d); err != nil {
		t.Fatalf("expected nil for 1024-char description, got %v", err)
	}
}

func TestValidateSkillDescription_Rejects1025Chars(t *testing.T) {
	t.Parallel()
	d := strings.Repeat("a", 1025)
	if err := domain.ValidateSkillDescription(d); err == nil {
		t.Fatalf("expected error for 1025-char description, got nil")
	}
}

func TestValidateSkillDescription_Accepts1Char(t *testing.T) {
	t.Parallel()
	if err := domain.ValidateSkillDescription("x"); err != nil {
		t.Fatalf("expected nil for 1-char description, got %v", err)
	}
}

func TestValidateSkillDescription_RejectsEmpty(t *testing.T) {
	t.Parallel()
	if err := domain.ValidateSkillDescription(""); err == nil {
		t.Fatalf("expected error for empty description, got nil")
	}
}

// ---------------------------------------------------------------------------
// ValidateSkillName — reserved-word rejection (spec S-SK-026).
// ---------------------------------------------------------------------------

func TestValidateSkillName_RejectsReservedWords(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name string
		in   string
	}{
		{"anthropicToolkit", "anthropic-toolkit"},
		{"claudeHelper", "claude-helper"},
		{"uppercaseAnthropic", "Anthropic"},
		{"uppercaseClaude", "CLAUDE"},
		{"claudeInMiddle", "my-claude-skill"},
		{"anthropicPrefix", "anthropic-foo"},
	}
	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			if err := domain.ValidateSkillName(tc.in); err == nil {
				t.Fatalf("expected error for reserved name %q, got nil", tc.in)
			}
		})
	}
}

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

func TestSkillNameRegex_RejectsInvalidNames(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name string
		in   string
	}{
		{"uppercase", "FooBar"},
		{"leadingHyphen", "-lead"},
		{"trailingHyphen", "trail-"},
		{"consecutiveHyphens", "foo--bar"},
		{"sixty5chars", strings.Repeat("a", 65)},
		{"empty", ""},
		{"underscore", "foo_bar"},
		{"space", "foo bar"},
		{"dot", "foo.bar"},
	}
	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			if err := domain.ValidateSkillName(tc.in); err == nil {
				t.Fatalf("expected error for %q, got nil", tc.in)
			}
		})
	}
}

