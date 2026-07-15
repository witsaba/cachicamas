// Package application — the PromptService use case facade for the
// 2026-07-15-prompt-storage-table change (PR 3 of 4).
//
// PromptService composes the prompt + prompt_revision repos to
// implement the seven use cases the HTTP handler will call:
//   - Create      (POST   /prompts)
//   - Update      (PATCH  /prompts/:slug)
//   - Restore     (POST   /prompts/:slug/revisions/:n/restore)
//   - SoftDelete  (DELETE /prompts/:slug)
//   - GetBySlug   (GET    /prompts/:slug)
//   - List        (GET    /prompts)
//   - ListRevisions (GET  /prompts/:slug/revisions)
//
// The service is the single concurrency gate for Update / Restore /
// SoftDelete: it acquires a row-level FOR UPDATE lock on the prompt
// row before reading it, which serializes concurrent writers on the
// same prompt. Two goroutines that try to Update the same prompt
// run sequentially; revision numbers are monotonic (spec INV-4,
// S-PR-20, S-PR-21).
package application

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log/slog"

	"github.com/cachicamas/backend/database_administrator/src/domain"
)

// PromptService is the use case facade for prompt operations.
type PromptService struct {
	promptRepo domain.PromptRepository
	revRepo    domain.PromptRevisionRepository
	db         *sql.DB
	logger     *slog.Logger
}

// NewPromptService constructs a PromptService. db is the connection
// pool used for single-statement reads (GetBySlug, List, etc.) and
// also the source of new transactions for multi-statement writes.
func NewPromptService(
	promptRepo domain.PromptRepository,
	revRepo domain.PromptRevisionRepository,
	db *sql.DB,
	logger *slog.Logger,
) *PromptService {
	if logger == nil {
		logger = slog.Default()
	}
	return &PromptService{
		promptRepo: promptRepo,
		revRepo:    revRepo,
		db:         db,
		logger:     logger,
	}
}

// ---------------------------------------------------------------------------
// Create
// ---------------------------------------------------------------------------

// Create validates the input, opens a TX, inserts the prompt row +
// revision 1 atomically, and returns the hydrated prompt + revision.
// On unique-violation of the partial slug index, returns
// *domain.ConflictError (mapped to 409 by the handler).
func (s *PromptService) Create(ctx context.Context, in domain.CreatePromptInput) (*domain.Prompt, *domain.PromptRevision, error) {
	if err := domain.ValidateSlug(in.Slug); err != nil {
		return nil, nil, err
	}
	if err := domain.ValidateDescription(in.Description); err != nil {
		return nil, nil, err
	}
	if err := domain.ValidateBody(in.Body); err != nil {
		return nil, nil, err
	}

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, nil, fmt.Errorf("PromptService.Create: BeginTx: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	p := &domain.Prompt{
		Description: in.Description,
		Slug:        in.Slug,
		Body:        in.Body,
	}
	if err := s.promptRepo.Insert(ctx, tx, p); err != nil {
		return nil, nil, err
	}

	rev := &domain.PromptRevision{
		PromptID:       p.ID,
		RevisionNumber: 1,
		Description:    p.Description,
		Body:           p.Body,
	}
	if err := s.revRepo.Insert(ctx, tx, rev); err != nil {
		return nil, nil, fmt.Errorf("PromptService.Create: revision Insert: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return nil, nil, fmt.Errorf("PromptService.Create: Commit: %w", err)
	}
	return p, rev, nil
}

// ---------------------------------------------------------------------------
// Update
// ---------------------------------------------------------------------------

// Update is the PATCH /prompts/:slug use case. At least one of
// in.Description or in.Body MUST be non-nil. The service acquires a
// FOR UPDATE lock on the prompt row, computes next = max+1, inserts
// a snapshot revision, and updates the prompt. Two concurrent calls
// serialize on the lock and produce sequential revision numbers
// (S-PR-21). On a soft-deleted prompt, returns *domain.GoneError (410).
func (s *PromptService) Update(ctx context.Context, slug string, in domain.UpdatePromptInput) (*domain.Prompt, *domain.PromptRevision, error) {
	if in.Description == nil && in.Body == nil {
		return nil, nil, &domain.ValidationError{
			Fields: map[string]string{"body": "at least one of description or body must be provided"},
		}
	}
	if in.Description != nil {
		if err := domain.ValidateDescription(*in.Description); err != nil {
			return nil, nil, err
		}
	}
	if in.Body != nil {
		if err := domain.ValidateBody(*in.Body); err != nil {
			return nil, nil, err
		}
	}

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, nil, fmt.Errorf("PromptService.Update: BeginTx: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	// Step 1: Locate the prompt by slug (returns row even if soft-
	// deleted so we can distinguish 404 from 410 per spec S-PR-8).
	existing, err := s.promptRepo.SelectBySlugAny(ctx, tx, slug)
	if err != nil {
		return nil, nil, err
	}
	if existing.DeletedAt != nil {
		return nil, nil, domain.NewPromptDeleted(slug)
	}
	// Step 2: Lock and load (this is the concurrency gate).
	locked, err := s.promptRepo.LockAndLoad(ctx, tx, existing.ID)
	if err != nil {
		return nil, nil, err
	}
	if locked.DeletedAt != nil {
		return nil, nil, domain.NewPromptDeleted(slug)
	}

	// Step 3: Compute the next revision number under the lock.
	maxRev, err := s.promptRepo.MaxRevisionNumber(ctx, tx, existing.ID)
	if err != nil {
		return nil, nil, fmt.Errorf("PromptService.Update: MaxRevisionNumber: %w", err)
	}
	newRev := maxRev + 1

	// Step 3: Decide which fields to update on the prompt.
	newDescription := locked.Description
	if in.Description != nil {
		newDescription = *in.Description
	}
	newBody := locked.Body
	if in.Body != nil {
		newBody = *in.Body
	}

	// Step 4: Insert the snapshot revision.
	rev := &domain.PromptRevision{
		PromptID:       existing.ID,
		RevisionNumber: newRev,
		Description:    newDescription,
		Body:           newBody,
	}
	if err := s.revRepo.Insert(ctx, tx, rev); err != nil {
		return nil, nil, fmt.Errorf("PromptService.Update: revision Insert: %w", err)
	}

	// Step 5: Update the prompt body/description. The DB owns
	// updated_at via DEFAULT now().
	if err := s.promptRepo.UpdateBody(ctx, tx, existing.ID, newBody, newDescription); err != nil {
		return nil, nil, err
	}

	if err := tx.Commit(); err != nil {
		return nil, nil, fmt.Errorf("PromptService.Update: Commit: %w", err)
	}

	// Re-read for the response (the in-memory copy is stale).
	updated, err := s.promptRepo.SelectByID(ctx, s.db, existing.ID)
	if err != nil {
		return nil, nil, fmt.Errorf("PromptService.Update: re-read: %w", err)
	}
	return updated, rev, nil
}

// ---------------------------------------------------------------------------
// Restore
// ---------------------------------------------------------------------------

// Restore implements POST /prompts/:slug/revisions/:n/restore. Reads
// the historical revision n, inserts it as the NEW latest revision
// (so history is preserved per spec S-PR-4), and updates the prompt
// to match. The change_note is set to "restored from revision N".
// On a soft-deleted prompt, returns *domain.GoneError (410,
// S-PR-5).
func (s *PromptService) Restore(ctx context.Context, slug string, n int) (*domain.Prompt, *domain.PromptRevision, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, nil, fmt.Errorf("PromptService.Restore: BeginTx: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	// Step 1: Locate the prompt by slug (returns row even if soft-
	// deleted so we can distinguish 404 from 410 per spec S-PR-5).
	existing, err := s.promptRepo.SelectBySlugAny(ctx, tx, slug)
	if err != nil {
		return nil, nil, err
	}
	if existing.DeletedAt != nil {
		return nil, nil, domain.NewPromptDeleted(slug)
	}
	// Step 2: Lock it.
	locked, err := s.promptRepo.LockAndLoad(ctx, tx, existing.ID)
	if err != nil {
		return nil, nil, err
	}
	if locked.DeletedAt != nil {
		return nil, nil, domain.NewPromptDeleted(slug)
	}
	// Step 3: Read the historical revision.
	historical, err := s.revRepo.SelectByPromptAndNumber(ctx, tx, existing.ID, n)
	if err != nil {
		return nil, nil, err
	}
	// Step 4: Compute next revision.
	maxRev, err := s.promptRepo.MaxRevisionNumber(ctx, tx, existing.ID)
	if err != nil {
		return nil, nil, fmt.Errorf("PromptService.Restore: MaxRevisionNumber: %w", err)
	}
	newRev := maxRev + 1
	note := fmt.Sprintf("restored from revision %d", n)

	// Step 5: Insert a new revision copying historical body+description.
	rev := &domain.PromptRevision{
		PromptID:       existing.ID,
		RevisionNumber: newRev,
		Description:    historical.Description,
		Body:           historical.Body,
		ChangeNote:     &note,
	}
	if err := s.revRepo.Insert(ctx, tx, rev); err != nil {
		return nil, nil, fmt.Errorf("PromptService.Restore: revision Insert: %w", err)
	}
	// Step 6: Update the prompt to match the restored revision.
	if err := s.promptRepo.UpdateBody(ctx, tx, existing.ID, historical.Body, historical.Description); err != nil {
		return nil, nil, err
	}

	if err := tx.Commit(); err != nil {
		return nil, nil, fmt.Errorf("PromptService.Restore: Commit: %w", err)
	}

	updated, err := s.promptRepo.SelectByID(ctx, s.db, existing.ID)
	if err != nil {
		return nil, nil, fmt.Errorf("PromptService.Restore: re-read: %w", err)
	}
	return updated, rev, nil
}

// ---------------------------------------------------------------------------
// SoftDelete
// ---------------------------------------------------------------------------

// SoftDelete sets deleted_at = now() on the prompt. Idempotent:
// calling it twice returns nil on both calls (spec S-PR-6 idempotent
// semantics).
func (s *PromptService) SoftDelete(ctx context.Context, slug string) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("PromptService.SoftDelete: BeginTx: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	existing, err := s.promptRepo.SelectBySlug(ctx, tx, slug)
	if err != nil {
		// NotFound is acceptable: the prompt was already deleted
		// (or never existed). To honor "idempotent 204", we treat
		// both as success.
		var nerr *domain.NotFoundError
		if errors.As(err, &nerr) {
			return tx.Commit()
		}
		return err
	}
	if err := s.promptRepo.SoftDelete(ctx, tx, existing.ID); err != nil {
		return err
	}
	return tx.Commit()
}

// ---------------------------------------------------------------------------
// Reads (no transaction needed)
// ---------------------------------------------------------------------------

// GetBySlug returns the current prompt row for the slug.
func (s *PromptService) GetBySlug(ctx context.Context, slug string) (*domain.Prompt, error) {
	return s.promptRepo.SelectBySlug(ctx, s.db, slug)
}

// List returns active prompts ordered by updated_at DESC. The
// service clamps limit to MaxListLimit (200) before reaching the
// repo and defaults limit to DefaultListLimit (50) when the caller
// passes 0 or negative.
func (s *PromptService) List(ctx context.Context, limit, offset int) ([]*domain.Prompt, error) {
	if limit <= 0 {
		limit = domain.DefaultListLimit
	}
	if limit > domain.MaxListLimit {
		limit = domain.MaxListLimit
	}
	if offset < 0 {
		offset = 0
	}
	return s.promptRepo.SelectList(ctx, s.db, limit, offset)
}

// ListRevisions returns the revision history of the prompt, newest
// first. Returns *domain.NotFoundError if the prompt does not exist
// or is soft-deleted.
func (s *PromptService) ListRevisions(ctx context.Context, slug string) ([]*domain.PromptRevision, error) {
	p, err := s.promptRepo.SelectBySlug(ctx, s.db, slug)
	if err != nil {
		return nil, err
	}
	return s.revRepo.SelectListByPrompt(ctx, s.db, p.ID)
}