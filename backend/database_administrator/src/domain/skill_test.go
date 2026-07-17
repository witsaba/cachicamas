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
	"context"
	"os"
	"strings"
	"testing"

	"github.com/cachicamas/backend/database_administrator/src/domain"
)

// ---------------------------------------------------------------------------
// Architectural invariant — domain MUST NOT import pgx (spec SCN-6.2).
//
// This is a stronger scanner than imports_test.go: it walks every .go
// file under domain/ and rejects the literal "github.com/jackc/pgx"
// string in any of them. The companion test in imports_test.go uses
// `go list -deps` to verify transitive imports; this one verifies the
// direct source. Both MUST pass together.
// ---------------------------------------------------------------------------

func TestDomain_DoesNotImportPgx(t *testing.T) {
	t.Parallel()
	entries, err := os.ReadDir(".")
	if err != nil {
		t.Fatalf("ReadDir(domain): %v", err)
	}
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".go") || strings.HasSuffix(e.Name(), "_test.go") {
			continue
		}
		contents, err := os.ReadFile(e.Name())
		if err != nil {
			t.Fatalf("ReadFile(%s): %v", e.Name(), err)
		}
		if strings.Contains(string(contents), "github.com/jackc/pgx") {
			t.Errorf("domain/%s must not import pgx; found import", e.Name())
		}
	}
}

// ---------------------------------------------------------------------------
// Repository ports (hexagonal boundary; implementations land in
// PR1b under src/infrastructure/postgres/skills/).
// ---------------------------------------------------------------------------

func TestSkillRepository_PortInterface(t *testing.T) {
	t.Parallel()
	// Compile-time check: a fake must satisfy the interface. If the
	// interface surface drifts, this file fails to build.
	fake := &fakeSkillRepo{}
	var _ domain.SkillRepository = fake
	if fake == nil {
		t.Fatalf("unexpected nil fake")
	}
}

func TestSkillRevisionRepository_PortInterface(t *testing.T) {
	t.Parallel()
	fake := &fakeSkillRevisionRepo{}
	var _ domain.SkillRevisionRepository = fake
	if fake == nil {
		t.Fatalf("unexpected nil fake")
	}
}

// fakeSkillRepo satisfies domain.SkillRepository at compile time
// (used only for the port interface test). No real methods; add
// stubs here if a future test calls them.
type fakeSkillRepo struct{}

func (*fakeSkillRepo) Insert(context.Context, domain.SQLExecutor, *domain.Skill) error { return nil }
func (*fakeSkillRepo) SelectBySlug(context.Context, domain.SQLExecutor, string) (*domain.Skill, error) {
	return nil, nil
}
func (*fakeSkillRepo) SelectBySlugAny(context.Context, domain.SQLExecutor, string) (*domain.Skill, error) {
	return nil, nil
}
func (*fakeSkillRepo) SelectByID(context.Context, domain.SQLExecutor, int64) (*domain.Skill, error) {
	return nil, nil
}
func (*fakeSkillRepo) List(context.Context, domain.SQLExecutor, int) ([]*domain.Skill, error) {
	return nil, nil
}
func (*fakeSkillRepo) ListWithCurrentRevision(context.Context, domain.SQLExecutor, int) ([]*domain.SkillListItem, error) {
	return nil, nil
}
func (*fakeSkillRepo) UpdateBody(context.Context, domain.SQLExecutor, int64, string, string) error {
	return nil
}
func (*fakeSkillRepo) SoftDelete(context.Context, domain.SQLExecutor, int64) error { return nil }
func (*fakeSkillRepo) LockAndLoad(context.Context, domain.SQLExecutor, int64) (*domain.Skill, error) {
	return nil, nil
}
func (*fakeSkillRepo) MaxRevisionNumber(context.Context, domain.SQLExecutor, int64) (int, error) {
	return 0, nil
}

// fakeSkillRevisionRepo satisfies domain.SkillRevisionRepository at
// compile time (used only for the port interface test).
type fakeSkillRevisionRepo struct{}

func (*fakeSkillRevisionRepo) Insert(context.Context, domain.SQLExecutor, *domain.SkillRevision) error {
	return nil
}
func (*fakeSkillRevisionRepo) SelectBySkillAndNumber(context.Context, domain.SQLExecutor, int64, int) (*domain.SkillRevision, error) {
	return nil, nil
}
func (*fakeSkillRevisionRepo) ListBySkillID(context.Context, domain.SQLExecutor, int64) ([]*domain.SkillRevision, error) {
	return nil, nil
}

// ---------------------------------------------------------------------------
// Error vocabulary (spec §7, design §3.6).
//
// Wire codes are the locked vocabulary the handler maps to HTTP
// statuses (400 / 404 / 409 / 410 / 500). Skill-specific values are
// `skill_deleted` (added here for the 410 case — organization.go has
// no 410 type); the rest come from organization.go.
// ---------------------------------------------------------------------------

func TestSkillValidationError_Fields(t *testing.T) {
	t.Parallel()
	err := &domain.ValidationError{
		Fields: map[string]string{"name": domain.MsgSkillNameLength},
	}
	if err.Code() != domain.CodeValidation {
		t.Errorf("expected Code() = %q, got %q", domain.CodeValidation, err.Code())
	}
	if err.Fields["name"] == "" {
		t.Errorf("expected Fields[name] to be non-empty, got %+v", err.Fields)
	}
}

func TestSkillNotFoundError_ResourceSkill(t *testing.T) {
	t.Parallel()
	err := &domain.NotFoundError{Resource: domain.ResourceSkill}
	if err.Code() != domain.CodeNotFound {
		t.Errorf("expected Code() = %q, got %q", domain.CodeNotFound, err.Code())
	}
	if err.Resource != domain.ResourceSkill {
		t.Errorf("expected Resource = %q, got %q", domain.ResourceSkill, err.Resource)
	}
}

func TestSkillConflictError_CodeIsConflict(t *testing.T) {
	t.Parallel()
	err := &domain.ConflictError{}
	if err.Code() != domain.CodeConflict {
		t.Errorf("expected Code() = %q, got %q", domain.CodeConflict, err.Code())
	}
}

func TestSkillGoneError_CodeIsSkillDeleted(t *testing.T) {
	t.Parallel()
	err := domain.NewSkillDeleted("pdf-cleanup")
	if err.Code() != domain.CodeSkillDeleted {
		t.Errorf("expected Code() = %q, got %q", domain.CodeSkillDeleted, err.Code())
	}
}

// ---------------------------------------------------------------------------
// Entity structs (spec INV-1, INV-2; mirrors `prompt.go` shape).
// ---------------------------------------------------------------------------

func TestSkill_EntityFields(t *testing.T) {
	t.Parallel()
	s := domain.Skill{}
	if s.ID != 0 {
		t.Errorf("expected zero-value ID, got %d", s.ID)
	}
	if s.Name != "" {
		t.Errorf("expected zero-value Name, got %q", s.Name)
	}
	if s.DeletedAt != nil {
		t.Errorf("expected zero-value (nil) DeletedAt, got %v", s.DeletedAt)
	}
}

func TestSkillRevision_EntityFields(t *testing.T) {
	t.Parallel()
	r := domain.SkillRevision{}
	if r.RevisionNumber != 0 {
		t.Errorf("expected zero-value RevisionNumber, got %d", r.RevisionNumber)
	}
	if r.SkillID != 0 {
		t.Errorf("expected zero-value SkillID, got %d", r.SkillID)
	}
	if r.ChangeNote != nil {
		t.Errorf("expected nil ChangeNote, got %v", r.ChangeNote)
	}
}

// ---------------------------------------------------------------------------
// CRLF + empty-body normalization (spec §4.3, design risk R2).
// ---------------------------------------------------------------------------

func TestFrontmatter_HandlesCRLF(t *testing.T) {
	t.Parallel()
	// Windows-authored files emit CRLF between lines. The YAML parser
	// accepts CRLF inside the block, but the fence matching code must
	// also normalize CRLF or the closing fence is never found.
	body := "---\r\nname: pdf-cleanup\r\ndescription: d\r\n---\r\n# Body\r\n"
	fm, err := domain.ParseFrontmatter(body)
	if err != nil {
		t.Fatalf("expected nil for CRLF body, got %v", err)
	}
	if fm.Name != "pdf-cleanup" {
		t.Errorf("expected Name = %q, got %q", "pdf-cleanup", fm.Name)
	}
	if fm.Description != "d" {
		t.Errorf("expected Description = %q, got %q", "d", fm.Description)
	}
}

func TestFrontmatter_HandlesEmptyMarkdownBody(t *testing.T) {
	t.Parallel()
	// Closing fence at end of file with no trailing newline.
	body := "---\nname: pdf-cleanup\ndescription: d\n---"
	fm, err := domain.ParseFrontmatter(body)
	if err != nil {
		t.Fatalf("expected nil for body ending at closing fence, got %v", err)
	}
	if fm.Name != "pdf-cleanup" {
		t.Errorf("expected Name = %q, got %q", "pdf-cleanup", fm.Name)
	}
}

// ---------------------------------------------------------------------------
// LockStepCheck (spec S-SK-035, S-SK-036, S-SK-007/008).
//
// The frontmatter parser returns `name` and `description`. These MUST
// equal the request's `name`/URL slug and `description`; mismatch is
// user error or spoofing (decision #2 in obs #1962). TrimSpace is
// applied to both sides so trailing newlines from copy-paste don't
// break legitimate edits.
// ---------------------------------------------------------------------------

func TestLockStepCheck_RejectsNameMismatch(t *testing.T) {
	t.Parallel()
	fm := domain.Frontmatter{Name: "other-skill", Description: "Same."}
	err := domain.LockStepCheck("pdf-cleanup", "Same.", fm)
	if err == nil {
		t.Fatalf("expected error for name mismatch, got nil")
	}
}

func TestLockStepCheck_RejectsDescriptionMismatch(t *testing.T) {
	t.Parallel()
	fm := domain.Frontmatter{Name: "pdf-cleanup", Description: "From frontmatter."}
	err := domain.LockStepCheck("pdf-cleanup", "From request.", fm)
	if err == nil {
		t.Fatalf("expected error for description mismatch, got nil")
	}
}

func TestLockStepCheck_AcceptsExactMatch(t *testing.T) {
	t.Parallel()
	fm := domain.Frontmatter{Name: "pdf-cleanup", Description: "Same."}
	if err := domain.LockStepCheck("pdf-cleanup", "Same.", fm); err != nil {
		t.Fatalf("expected nil for exact match, got %v", err)
	}
}

func TestLockStepCheck_AcceptsAfterTrim(t *testing.T) {
	t.Parallel()
	fm := domain.Frontmatter{Name: "pdf-cleanup", Description: "Same."}
	// Whitespace on either side is silently trimmed; the comparison
	// is on the trimmed values.
	if err := domain.LockStepCheck("  pdf-cleanup  ", "\tSame.\n", fm); err != nil {
		t.Fatalf("expected nil after trim, got %v", err)
	}
}

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

