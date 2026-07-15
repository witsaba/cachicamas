// Package domain contains the core business types of the database
// administrator service. This file defines the prompts aggregate for
// the 2026-07-15-prompt-storage-table change:
//
//   - Prompt                (one row of the `prompt` table; the current
//     definitive version)
//   - PromptRevision        (one row of `prompt_revision`; an append-only
//     snapshot of a body+description at a point in
//     time)
//   - CreatePromptInput,
//     UpdatePromptInput     (handler-facing payloads)
//   - PromptRepository      (the hexagonal port the application layer
//     uses to read and persist Prompt rows)
//   - PromptRevisionRepository
//   - validation rules (slug regex, length caps)
//   - GoneError             (the prompt-specific 410 case: a soft-deleted
//     prompt that still exists in the DB)
//
// Per design §4, the handler maps each error's Code() to the locked HTTP
// envelope; the application layer propagates errors as-is (do not wrap
// further — the handler uses errors.As). Prompt feature reuses the
// existing ConflictError / NotFoundError / ValidationError types from
// organization.go for the 409 / 404 / 400 cases (wire codes
// `conflict`, `not_found`, `validation`); the new GoneError covers the
// 410 case, which no existing type handled.
//
// See domain/prompt_test.go for the locked validation rules + error
// code contract (TDD discipline: the test file landed before this one).
package domain

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"regexp"
	"time"
)

// ---------------------------------------------------------------------------
// Constants (spec INV-1, INV-2, §6.4–6.7).
// ---------------------------------------------------------------------------

const (
	// MaxPromptDescriptionLen caps the human-readable description at
	// 280 chars. Mirrors the CHECK constraint on `prompt.description`.
	MaxPromptDescriptionLen = 280

	// MaxPromptBodyLen caps the markdown body at 512 KB. Mirrors the
	// CHECK constraint on `prompt.body` and `prompt_revision.body`.
	MaxPromptBodyLen = 524288

	// DefaultListLimit is the default page size for List.
	DefaultListLimit = 50

	// MaxListLimit is the hard cap on List's page size; values above
	// this are clamped to MaxListLimit.
	MaxListLimit = 200
)

// promptSlugRegex matches the locked slug pattern: 2-100 chars total,
// must start and end with [a-z0-9], inner chars may include hyphens.
// This is the server-side rule; the SQL CHECK on `prompt.slug` is the
// same regex. Single source of truth. The name is qualified with the
// `prompt` prefix to avoid collision with the slugRegex in
// organization.go (different patterns, different features).
var promptSlugRegex = regexp.MustCompile(`^[a-z0-9][a-z0-9-]{0,98}[a-z0-9]$`)

// ---------------------------------------------------------------------------
// Locked user-facing error messages for the prompts feature. The
// handler returns the message verbatim inside the error envelope; a
// typo in any of these will fail a contract test. Codes at the wire
// level are the existing generic codes from organization.go
// (validation, conflict, not_found); this file only adds GoneError
// for the 410 case.
// ---------------------------------------------------------------------------

const (
	// CodePromptDeleted is the wire code returned when an operation
	// targets a soft-deleted prompt. The handler maps GoneError to
	// HTTP 410. The existing vocabulary in organization.go has no
	// 410 case, so this code is new for the prompts feature.
	CodePromptDeleted = "prompt_deleted"

	// MsgPromptNotFound is the user-facing message when no active
	// prompt matches the requested slug or id.
	MsgPromptNotFound = "Prompt not found."

	// MsgPromptRevisionNotFound is the user-facing message when the
	// requested revision number does not exist for the prompt.
	MsgPromptRevisionNotFound = "Prompt revision not found."

	// MsgPromptSlugTaken is the user-facing message when the slug
	// collides with another active prompt.
	MsgPromptSlugTaken = "This prompt slug is already taken. Try another."

	// MsgPromptDeleted is the user-facing message when an operation
	// targets a soft-deleted prompt.
	MsgPromptDeleted = "This prompt has been deleted and cannot be modified."

	// MsgPromptSlugInvalid is the user-facing message for slug
	// regex failure.
	MsgPromptSlugInvalid = "Slug must be 2-100 chars: lowercase letters, digits, and hyphens; must start and end with a letter or digit."

	// MsgPromptDescriptionInvalid is the user-facing message for
	// description length failure.
	MsgPromptDescriptionInvalid = "Prompt description must be 1-280 characters."

	// MsgPromptBodyTooLarge is the user-facing message for body
	// length failure.
	MsgPromptBodyTooLarge = "Prompt body must be 1-524288 characters."
)

// ---------------------------------------------------------------------------
// Entity types.
// ---------------------------------------------------------------------------

// Prompt is the domain entity for one row of the `prompt` table
// (DDL: migration/sql/20260715120000_prompts.sql). The struct mirrors
// the DDL column-for-column; the `db` tags tell the pgx adapter which
// column each field reads from, the `json` tags control wire
// serialization.
//
// DeletedAt is a *time.Time without `omitempty` because the locked
// contract is that a soft-NULL (deleted) is a distinct state from
// "never deleted" (zero time); the API consumer can distinguish them.
type Prompt struct {
	ID          int64      `db:"id"          json:"id"`
	Description string     `db:"description" json:"description"`
	Slug        string     `db:"slug"        json:"slug"`
	Body        string     `db:"body"        json:"body"`
	DeletedAt   *time.Time `db:"deleted_at"  json:"deleted_at"`
	CreatedAt   time.Time  `db:"created_at"  json:"created_at"`
	UpdatedAt   time.Time  `db:"updated_at"  json:"updated_at"`
}

// PromptRevision is the domain entity for one row of the
// `prompt_revision` table. The struct snapshots the prompt body and
// description at the moment of a change; it is append-only.
//
// RevisionNumber is monotonic per prompt_id and starts at 1 for the
// initial create.
type PromptRevision struct {
	ID             int64     `db:"id"              json:"id"`
	PromptID       int64     `db:"prompt_id"       json:"prompt_id"`
	RevisionNumber int       `db:"revision_number" json:"revision_number"`
	Description    string    `db:"description"     json:"description"`
	Body           string    `db:"body"            json:"body"`
	ChangeNote     *string   `db:"change_note"     json:"change_note"`
	CreatedBy      *string   `db:"created_by"      json:"created_by"`
	CreatedAt      time.Time `db:"created_at"      json:"created_at"`
}

// CreatePromptInput is the handler-facing payload for the POST
// /prompts use case.
type CreatePromptInput struct {
	Slug        string
	Description string
	Body        string
}

// UpdatePromptInput is the handler-facing payload for the PATCH
// /prompts/:slug use case. At least one of Description or Body MUST
// be non-nil; the service layer validates this.
type UpdatePromptInput struct {
	Description *string
	Body        *string
}

// ---------------------------------------------------------------------------
// Validation rules (spec S-PR-10..19).
//
// The validators return *ValidationError with a Fields map keyed by
// the wire field name. The existing handler maps ValidationError to
// 400 with code "validation" (from organization.go's CodeValidation).
// ---------------------------------------------------------------------------

// ValidateSlug returns a *ValidationError if slug does not match the
// locked regex `^[a-z0-9][a-z0-9-]{0,98}[a-z0-9]$`.
func ValidateSlug(slug string) error {
	if !promptSlugRegex.MatchString(slug) {
		return &ValidationError{
			Fields: map[string]string{"slug": MsgPromptSlugInvalid},
		}
	}
	return nil
}

// ValidateDescription returns a *ValidationError if description is
// empty or longer than MaxPromptDescriptionLen.
func ValidateDescription(desc string) error {
	n := len([]rune(desc))
	if n < 1 || n > MaxPromptDescriptionLen {
		return &ValidationError{
			Fields: map[string]string{"description": MsgPromptDescriptionInvalid},
		}
	}
	return nil
}

// ValidateBody returns a *ValidationError if body is empty or longer
// than MaxPromptBodyLen.
func ValidateBody(body string) error {
	n := len([]rune(body))
	if n < 1 || n > MaxPromptBodyLen {
		return &ValidationError{
			Fields: map[string]string{"body": MsgPromptBodyTooLarge},
		}
	}
	return nil
}

// ---------------------------------------------------------------------------
// GoneError — the prompt-specific 410 case.
//
// The row exists in the DB but is soft-deleted. Reads hide it (so they
// return NotFoundError, 404); updates and restores must reject (so
// they return GoneError, 410). Reusing NotFoundError would conflate
// the two and force the handler to infer 410 from message text; a
// dedicated type keeps the HTTP mapping deterministic and easy to
// test with errors.As.
// ---------------------------------------------------------------------------

// GoneError signals that a row exists but is in a terminal state that
// disallows the requested operation (e.g., soft-deleted prompt). The
// handler maps GoneError to HTTP 410 with the locked envelope.
type GoneError struct {
	Slug  string
	Cause error
}

func (e *GoneError) Error() string {
	if e.Slug == "" {
		return MsgPromptDeleted
	}
	return fmt.Sprintf("prompt slug=%q: %s", e.Slug, MsgPromptDeleted)
}

func (e *GoneError) Code() string { return CodePromptDeleted }

func (e *GoneError) Unwrap() error { return e.Cause }

// NewPromptDeleted returns a *GoneError for a soft-deleted prompt
// that the caller tried to modify or restore.
func NewPromptDeleted(slug string) *GoneError {
	return &GoneError{Slug: slug}
}

// AsPromptDeleted is a convenience that wraps errors.As for the
// prompt-specific GoneError, mirroring the pattern used elsewhere in
// the codebase to keep the handler's type switch readable.
func AsPromptDeleted(err error) (*GoneError, bool) {
	var gone *GoneError
	if errors.As(err, &gone) {
		return gone, true
	}
	return nil, false
}

// ---------------------------------------------------------------------------
// Repository ports (hexagonal boundary). Implementations live under
// src/infrastructure/postgres/. The interfaces are deliberately
// small so a fake is trivial to write in tests.
// ---------------------------------------------------------------------------

// sqlExecutor is satisfied by *sql.DB and *sql.Tx, so the service can
// pass either to the repo depending on whether the operation runs in
// a transaction. Defining it in the domain package keeps pgx out of
// the domain (spec S-PR-X5).
type sqlExecutor interface {
	ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error)
	QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error)
	QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row
}

// PromptRepository is the port for reading and persisting Prompt
// rows. The application layer depends only on this interface.
//
// Contract:
//   - Insert(ctx, db, p) MUST set p.ID, p.CreatedAt, p.UpdatedAt from
//     the DB clock (DEFAULT now()) and return the populated p.
//   - SelectBySlug(ctx, db, slug) returns *NotFoundError with Code
//     CodeNotFound when no active row matches.
//   - SelectByID(ctx, db, id) returns *NotFoundError with Code
//     CodeNotFound when no row matches.
//   - SelectList(ctx, db, limit, offset) returns active rows only
//     (deleted_at IS NULL) ordered by updated_at DESC.
//   - UpdateBody(ctx, db, id, body, description) updates body,
//     description, and updated_at; the DB clock owns updated_at.
//   - SoftDelete(ctx, db, id) sets deleted_at = now(); idempotent.
//   - LockAndLoad(ctx, db, id) issues SELECT … FOR UPDATE; the caller
//     MUST be inside a transaction.
//   - MaxRevisionNumber(ctx, db, promptID) returns
//     COALESCE(MAX(revision_number), 0) for the given prompt; used by
//     the service under FOR UPDATE to assign the next revision.
//   - All methods honour the context.
type PromptRepository interface {
	Insert(ctx context.Context, db sqlExecutor, p *Prompt) error
	SelectBySlug(ctx context.Context, db sqlExecutor, slug string) (*Prompt, error)
	SelectByID(ctx context.Context, db sqlExecutor, id int64) (*Prompt, error)
	SelectList(ctx context.Context, db sqlExecutor, limit, offset int) ([]*Prompt, error)
	UpdateBody(ctx context.Context, db sqlExecutor, id int64, body, description string) error
	SoftDelete(ctx context.Context, db sqlExecutor, id int64) error
	LockAndLoad(ctx context.Context, db sqlExecutor, id int64) (*Prompt, error)
	MaxRevisionNumber(ctx context.Context, db sqlExecutor, promptID int64) (int, error)
}

// PromptRevisionRepository is the port for reading and persisting
// PromptRevision rows.
//
// Contract:
//   - Insert(ctx, db, r) MUST set r.ID and r.CreatedAt from the DB
//     clock.
//   - SelectLatestForPrompt(ctx, db, promptID) returns the highest
//     revision_number for the prompt, or *NotFoundError with Code
//     CodeNotFound if none exist (which means the invariant "every
//     prompt has at least one revision" was violated at the DB level).
//   - SelectByPromptAndNumber(ctx, db, promptID, n) returns the
//     specific revision, or *NotFoundError with Code CodeNotFound.
//   - SelectListByPrompt(ctx, db, promptID) returns all revisions
//     ordered by revision_number DESC.
//   - All methods honour the context.
type PromptRevisionRepository interface {
	Insert(ctx context.Context, db sqlExecutor, r *PromptRevision) error
	SelectLatestForPrompt(ctx context.Context, db sqlExecutor, promptID int64) (*PromptRevision, error)
	SelectByPromptAndNumber(ctx context.Context, db sqlExecutor, promptID int64, n int) (*PromptRevision, error)
	SelectListByPrompt(ctx context.Context, db sqlExecutor, promptID int64) ([]*PromptRevision, error)
}
