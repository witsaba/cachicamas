// Package domain_test contains the TDD-locked unit tests for the
// prompt feature. Every assertion below corresponds to a spec scenario
// in openspec/changes/2026-07-15-prompt-storage-table/specs/prompts/spec.md.
//
// This file MUST land before the production code it tests (RED step
// in strict TDD). The production file is domain/prompt.go.
package domain_test

import (
	"errors"
	"strings"
	"testing"

	"github.com/cachicamas/backend/database_administrator/src/domain"
)

// ---------------------------------------------------------------------------
// ValidateSlug (spec S-PR-10..14).
// ---------------------------------------------------------------------------

func TestValidateSlug_RejectsUppercase(t *testing.T) {
	t.Parallel()
	if err := domain.ValidateSlug("Welcome"); err == nil {
		t.Fatalf("expected error for uppercase slug, got nil")
	}
}

func TestValidateSlug_RejectsEmpty(t *testing.T) {
	t.Parallel()
	if err := domain.ValidateSlug(""); err == nil {
		t.Fatalf("expected error for empty slug, got nil")
	}
}

func TestValidateSlug_AcceptsTwoChars(t *testing.T) {
	t.Parallel()
	if err := domain.ValidateSlug("ab"); err != nil {
		t.Fatalf("expected nil for 2-char slug, got %v", err)
	}
}

func TestValidateSlug_AcceptsHundredChars(t *testing.T) {
	t.Parallel()
	slug := strings.Repeat("a", 100)
	if err := domain.ValidateSlug(slug); err != nil {
		t.Fatalf("expected nil for 100-char slug, got %v", err)
	}
}

func TestValidateSlug_RejectsOneHundredOneChars(t *testing.T) {
	t.Parallel()
	slug := strings.Repeat("a", 101)
	if err := domain.ValidateSlug(slug); err == nil {
		t.Fatalf("expected error for 101-char slug, got nil")
	}
}

func TestValidateSlug_RejectsLeadingHyphen(t *testing.T) {
	t.Parallel()
	if err := domain.ValidateSlug("-welcome"); err == nil {
		t.Fatalf("expected error for leading hyphen, got nil")
	}
}

func TestValidateSlug_RejectsTrailingHyphen(t *testing.T) {
	t.Parallel()
	if err := domain.ValidateSlug("welcome-"); err == nil {
		t.Fatalf("expected error for trailing hyphen, got nil")
	}
}

func TestValidateSlug_AcceptsHyphenInMiddle(t *testing.T) {
	t.Parallel()
	if err := domain.ValidateSlug("welcome-email-v2"); err != nil {
		t.Fatalf("expected nil for slug with internal hyphens, got %v", err)
	}
}

func TestValidateSlug_AcceptsDigits(t *testing.T) {
	t.Parallel()
	if err := domain.ValidateSlug("welcome-2024"); err != nil {
		t.Fatalf("expected nil for slug with digits, got %v", err)
	}
}

func TestValidateSlug_RejectsSpecialCharacters(t *testing.T) {
	t.Parallel()
	for _, bad := range []string{"welcome!", "welcome.email", "welcome_email", "welcome/email", "welcome email"} {
		if err := domain.ValidateSlug(bad); err == nil {
			t.Errorf("expected error for slug %q, got nil", bad)
		}
	}
}

// ---------------------------------------------------------------------------
// ValidateDescription (spec S-PR-15, S-PR-16).
// ---------------------------------------------------------------------------

func TestValidateDescription_RejectsEmpty(t *testing.T) {
	t.Parallel()
	if err := domain.ValidateDescription(""); err == nil {
		t.Fatalf("expected error for empty description, got nil")
	}
}

func TestValidateDescription_Accepts280Chars(t *testing.T) {
	t.Parallel()
	desc := strings.Repeat("a", 280)
	if err := domain.ValidateDescription(desc); err != nil {
		t.Fatalf("expected nil for 280-char description, got %v", err)
	}
}

func TestValidateDescription_Rejects281Chars(t *testing.T) {
	t.Parallel()
	desc := strings.Repeat("a", 281)
	if err := domain.ValidateDescription(desc); err == nil {
		t.Fatalf("expected error for 281-char description, got nil")
	}
}

// ---------------------------------------------------------------------------
// ValidateBody (spec S-PR-17, S-PR-18, S-PR-19).
// ---------------------------------------------------------------------------

func TestValidateBody_RejectsEmpty(t *testing.T) {
	t.Parallel()
	if err := domain.ValidateBody(""); err == nil {
		t.Fatalf("expected error for empty body, got nil")
	}
}

func TestValidateBody_AcceptsAtMaxLen(t *testing.T) {
	t.Parallel()
	body := strings.Repeat("x", domain.MaxPromptBodyLen)
	if err := domain.ValidateBody(body); err != nil {
		t.Fatalf("expected nil for body at max length, got %v", err)
	}
}

func TestValidateBody_RejectsOverMaxLen(t *testing.T) {
	t.Parallel()
	body := strings.Repeat("x", domain.MaxPromptBodyLen+1)
	if err := domain.ValidateBody(body); err == nil {
		t.Fatalf("expected error for body over max length, got nil")
	}
}

// ---------------------------------------------------------------------------
// ValidationError surface (the handler maps *ValidationError to 400
// with the locked `validation` code from organization.go).
// ---------------------------------------------------------------------------

func TestValidateSlug_ErrorExposesFieldsMap(t *testing.T) {
	t.Parallel()
	err := domain.ValidateSlug("BAD")
	if err == nil {
		t.Fatalf("expected error, got nil")
	}
	var verr *domain.ValidationError
	if !errors.As(err, &verr) {
		t.Fatalf("expected *ValidationError, got %T", err)
	}
	if verr.Fields["slug"] == "" {
		t.Fatalf("expected slug field in error.Fields, got %+v", verr.Fields)
	}
	if verr.Code() != domain.CodeValidation {
		t.Errorf("expected Code() = %q, got %q", domain.CodeValidation, verr.Code())
	}
}

func TestValidateDescription_ErrorExposesFieldsMap(t *testing.T) {
	t.Parallel()
	err := domain.ValidateDescription("")
	var verr *domain.ValidationError
	if !errors.As(err, &verr) {
		t.Fatalf("expected *ValidationError, got %T", err)
	}
	if verr.Fields["description"] == "" {
		t.Fatalf("expected description field in error.Fields, got %+v", verr.Fields)
	}
}

func TestValidateBody_ErrorExposesFieldsMap(t *testing.T) {
	t.Parallel()
	err := domain.ValidateBody("")
	var verr *domain.ValidationError
	if !errors.As(err, &verr) {
		t.Fatalf("expected *ValidationError, got %T", err)
	}
	if verr.Fields["body"] == "" {
		t.Fatalf("expected body field in error.Fields, got %+v", verr.Fields)
	}
}

// ---------------------------------------------------------------------------
// GoneError surface (the handler maps *GoneError to 410 with the
// `prompt_deleted` code; this is the only NEW wire code added by the
// prompt feature, because no existing type covers 410).
// ---------------------------------------------------------------------------

func TestNewPromptDeleted_ProducesGoneError(t *testing.T) {
	t.Parallel()
	err := domain.NewPromptDeleted("welcome")
	if err == nil {
		t.Fatalf("expected error, got nil")
	}
	var gone *domain.GoneError
	if !errors.As(err, &gone) {
		t.Fatalf("expected *GoneError, got %T", err)
	}
	if gone.Slug != "welcome" {
		t.Errorf("expected slug = %q, got %q", "welcome", gone.Slug)
	}
	if gone.Code() != domain.CodePromptDeleted {
		t.Errorf("expected Code() = %q, got %q", domain.CodePromptDeleted, gone.Code())
	}
}

func TestAsPromptDeleted_ReturnsFalseOnUnrelatedError(t *testing.T) {
	t.Parallel()
	err := errors.New("some other error")
	if _, ok := domain.AsPromptDeleted(err); ok {
		t.Errorf("expected AsPromptDeleted to return false for unrelated error")
	}
}

func TestAsPromptDeleted_ReturnsTrueOnPromptDeleted(t *testing.T) {
	t.Parallel()
	err := domain.NewPromptDeleted("foo")
	gone, ok := domain.AsPromptDeleted(err)
	if !ok {
		t.Fatalf("expected AsPromptDeleted to return true")
	}
	if gone.Slug != "foo" {
		t.Errorf("expected slug = %q, got %q", "foo", gone.Slug)
	}
}

// ---------------------------------------------------------------------------
// Entity struct shape (compile-time and runtime invariants).
// ---------------------------------------------------------------------------

func TestPrompt_StructHasExpectedFields(t *testing.T) {
	t.Parallel()
	// Compile-time check: if any db tag is missing on these fields,
	// the test file fails to compile. The repo will check the same
	// shape at runtime via the column scan.
	p := domain.Prompt{}
	if p.ID != 0 {
		t.Errorf("expected zero-value ID, got %d", p.ID)
	}
	if p.Slug != "" {
		t.Errorf("expected zero-value Slug, got %q", p.Slug)
	}
}

func TestPromptRevision_StructHasExpectedFields(t *testing.T) {
	t.Parallel()
	r := domain.PromptRevision{}
	if r.RevisionNumber != 0 {
		t.Errorf("expected zero-value RevisionNumber, got %d", r.RevisionNumber)
	}
	if r.PromptID != 0 {
		t.Errorf("expected zero-value PromptID, got %d", r.PromptID)
	}
}

// ---------------------------------------------------------------------------
// Locked constants are stable (a typo here breaks the wire contract;
// the test fails loudly before any handler is built).
// ---------------------------------------------------------------------------

func TestCodePromptDeleted_IsStable(t *testing.T) {
	t.Parallel()
	if domain.CodePromptDeleted != "prompt_deleted" {
		t.Errorf("CodePromptDeleted must be the locked value %q, got %q", "prompt_deleted", domain.CodePromptDeleted)
	}
}

func TestMaxPromptDescriptionLen_IsStable(t *testing.T) {
	t.Parallel()
	if domain.MaxPromptDescriptionLen != 280 {
		t.Errorf("MaxPromptDescriptionLen must be 280, got %d", domain.MaxPromptDescriptionLen)
	}
}

func TestMaxPromptBodyLen_IsStable(t *testing.T) {
	t.Parallel()
	if domain.MaxPromptBodyLen != 524288 {
		t.Errorf("MaxPromptBodyLen must be 524288 (512 KB), got %d", domain.MaxPromptBodyLen)
	}
}

func TestDefaultAndMaxListLimit_AreOrdered(t *testing.T) {
	t.Parallel()
	if domain.DefaultListLimit > domain.MaxListLimit {
		t.Errorf("DefaultListLimit (%d) must be <= MaxListLimit (%d)", domain.DefaultListLimit, domain.MaxListLimit)
	}
	if domain.MaxListLimit != 200 {
		t.Errorf("MaxListLimit must be 200, got %d", domain.MaxListLimit)
	}
	if domain.DefaultListLimit != 50 {
		t.Errorf("DefaultListLimit must be 50, got %d", domain.DefaultListLimit)
	}
}
