// Package application — SkillService use case facade for the
// 2026-07-17-skills-foundational change (PR1c of the chained PR set).
//
// SkillService composes the skill + skill_revision repos to implement
// the seven use cases the HTTP handler will call:
//   - Create      (POST   /skills)
//   - Update      (PATCH  /skills/:name)
//   - Restore     (POST   /skills/:name/revisions/:n/restore)
//   - SoftDelete  (DELETE /skills/:name)
//   - GetBySlug   (GET    /skills/:name)
//   - List        (GET    /skills)
//   - ListRevisions (GET  /skills/:name/revisions)
//
// The service is the single concurrency gate for Update / Restore /
// SoftDelete: it acquires a row-level FOR UPDATE lock on the skill
// row before reading it, which serializes concurrent writers on the
// same skill. Two goroutines that try to Update the same skill run
// sequentially; revision numbers are monotonic (spec SCN-4.2).
//
// This file mirrors the structure of prompt_service.go (the proven
// pattern from the prompts feature), with three deltas per design
// ADR-SK-005: frontmatter parsing + lock-step happens at the service
// layer (not the handler), the reserved-word check is integrated into
// ValidateSkillName, and the GoneError carries the skill NAME (not
// slug) per the agentskills.io convention.
package application

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log/slog"

	"github.com/cachicamas/backend/database_administrator/src/domain"
)

// SkillService is the use case facade for skill operations.
type SkillService struct {
	skillRepo domain.SkillRepository
	revRepo   domain.SkillRevisionRepository
	db        *sql.DB
	logger    *slog.Logger
}

// NewSkillService constructs a SkillService. db is the connection pool
// used for single-statement reads (GetBySlug, List, ListRevisions) and
// the source of new transactions for multi-statement writes (Create,
// Update, Restore, SoftDelete).
func NewSkillService(
	skillRepo domain.SkillRepository,
	revRepo domain.SkillRevisionRepository,
	db *sql.DB,
	logger *slog.Logger,
) *SkillService {
	if logger == nil {
		logger = slog.Default()
	}
	return &SkillService{
		skillRepo: skillRepo,
		revRepo:   revRepo,
		db:        db,
		logger:    logger,
	}
}

// ---------------------------------------------------------------------------
// Create (POST /skills)
// ---------------------------------------------------------------------------

// Create validates the input, opens a TX, inserts the skill row +
// revision 1 atomically, and returns the hydrated skill + revision.
// On unique-violation of the partial slug index, returns
// *domain.ConflictError (mapped to 409 by the handler).
func (s *SkillService) Create(ctx context.Context, in domain.CreateSkillInput) (*domain.Skill, *domain.SkillRevision, error) {
	if err := domain.ValidateSkillName(in.Name); err != nil {
		return nil, nil, err
	}
	if err := domain.ValidateSkillDescription(in.Description); err != nil {
		return nil, nil, err
	}
	if err := domain.ValidateSkillBody(in.Body); err != nil {
		return nil, nil, err
	}
	// Frontmatter parsing + lock-step (design ADR-SK-005): the body's
	// YAML frontmatter MUST declare name == slug and description ==
	// request description. ParseFrontmatter normalizes CRLF; LockStepCheck
	// trims whitespace before comparing.
	fm, err := domain.ParseFrontmatter(in.Body)
	if err != nil {
		return nil, nil, err
	}
	if err := domain.LockStepCheck(in.Name, in.Description, fm); err != nil {
		return nil, nil, err
	}

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, nil, fmt.Errorf("SkillService.Create: BeginTx: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	sk := &domain.Skill{
		Name:        in.Name,
		Description: in.Description,
		Body:        in.Body,
	}
	if err := s.skillRepo.Insert(ctx, tx, sk); err != nil {
		return nil, nil, err
	}

	rev := &domain.SkillRevision{
		SkillID:        sk.ID,
		RevisionNumber: 1,
		Description:    sk.Description,
		Body:           sk.Body,
	}
	if err := s.revRepo.Insert(ctx, tx, rev); err != nil {
		return nil, nil, fmt.Errorf("SkillService.Create: revision Insert: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return nil, nil, fmt.Errorf("SkillService.Create: Commit: %w", err)
	}
	return sk, rev, nil
}

// ---------------------------------------------------------------------------
// Update (PATCH /skills/:name)
// ---------------------------------------------------------------------------

// Update applies a partial mutation to an existing skill and appends
// a new revision row reflecting the post-mutation state. At least
// one of in.Description or in.Body MUST be non-nil. Concurrency gate:
// the service acquires a SELECT … FOR UPDATE row lock via the repo
// before reading current state, computes max+1 under the lock, and
// re-reads the skill for the response so updated_at reflects the DB
// clock. On a soft-deleted skill, returns *domain.SkillGoneError
// (410 — spec SCN-3.x).
func (s *SkillService) Update(ctx context.Context, name string, in domain.UpdateSkillInput) (*domain.Skill, *domain.SkillRevision, error) {
	if in.Description == nil && in.Body == nil {
		return nil, nil, &domain.ValidationError{
			Fields: map[string]string{"body": "at least one of description or body must be provided"},
		}
	}
	if in.Description != nil {
		if err := domain.ValidateSkillDescription(*in.Description); err != nil {
			return nil, nil, err
		}
	}
	if in.Body != nil {
		if err := domain.ValidateSkillBody(*in.Body); err != nil {
			return nil, nil, err
		}
	}

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, nil, fmt.Errorf("SkillService.Update: BeginTx: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	// Step 1: Locate by name (any state) so we can distinguish 404 vs 410.
	existing, err := s.skillRepo.SelectBySlugAny(ctx, tx, name)
	if err != nil {
		return nil, nil, err
	}
	if existing.DeletedAt != nil {
		return nil, nil, domain.NewSkillDeleted(name)
	}
	// Step 2: Lock and load (concurrency gate — serializes concurrent
	// writers on the same row).
	locked, err := s.skillRepo.LockAndLoad(ctx, tx, existing.ID)
	if err != nil {
		return nil, nil, err
	}
	if locked.DeletedAt != nil {
		return nil, nil, domain.NewSkillDeleted(name)
	}

	// Step 2.5: If the body changed, parse its frontmatter and check
	// lock-step against the POST-mutation state. When the caller did
	// not provide a new description, infer it from the body's
	// frontmatter (the body is the source of truth for both fields).
	if in.Body != nil {
		fm, err := domain.ParseFrontmatter(*in.Body)
		if err != nil {
			return nil, nil, err
		}
		descForLockStep := fm.Description
		if in.Description != nil {
			descForLockStep = *in.Description
		}
		if err := domain.LockStepCheck(name, descForLockStep, fm); err != nil {
			return nil, nil, err
		}
		// When the caller omitted Description, lift it from the
		// frontmatter so the skill row + revision both reflect the
		// post-mutation state.
		if in.Description == nil {
			inferred := fm.Description
			in.Description = &inferred
		}
	}

	// Step 3: Compute next revision under the lock.
	maxRev, err := s.skillRepo.MaxRevisionNumber(ctx, tx, existing.ID)
	if err != nil {
		return nil, nil, fmt.Errorf("SkillService.Update: MaxRevisionNumber: %w", err)
	}
	newRev := maxRev + 1

	// Step 4: Decide the new field values.
	newDescription := locked.Description
	if in.Description != nil {
		newDescription = *in.Description
	}
	newBody := locked.Body
	if in.Body != nil {
		newBody = *in.Body
	}

	// Step 5: Snapshot revision.
	rev := &domain.SkillRevision{
		SkillID:        existing.ID,
		RevisionNumber: newRev,
		Description:    newDescription,
		Body:           newBody,
	}
	if err := s.revRepo.Insert(ctx, tx, rev); err != nil {
		return nil, nil, fmt.Errorf("SkillService.Update: revision Insert: %w", err)
	}

	// Step 6: Update the skill row.
	if err := s.skillRepo.UpdateBody(ctx, tx, existing.ID, newBody, newDescription); err != nil {
		return nil, nil, err
	}

	if err := tx.Commit(); err != nil {
		return nil, nil, fmt.Errorf("SkillService.Update: Commit: %w", err)
	}

	// Re-read so the response reflects the DB clock + committed state.
	updated, err := s.skillRepo.SelectByID(ctx, s.db, existing.ID)
	if err != nil {
		return nil, nil, fmt.Errorf("SkillService.Update: re-read: %w", err)
	}
	return updated, rev, nil
}

// ---------------------------------------------------------------------------
// Restore (POST /skills/:name/revisions/:n/restore)
// ---------------------------------------------------------------------------

// Restore implements POST /skills/:name/revisions/:n/restore. Reads
// historical revision n, inserts it as the NEW latest revision (so
// history is preserved per spec SCN-1.3), and updates the skill to
// match. The change_note is set to "restored from revision N". On a
// soft-deleted skill, returns *domain.SkillGoneError (410).
func (s *SkillService) Restore(ctx context.Context, name string, n int) (*domain.Skill, *domain.SkillRevision, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, nil, fmt.Errorf("SkillService.Restore: BeginTx: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	existing, err := s.skillRepo.SelectBySlugAny(ctx, tx, name)
	if err != nil {
		return nil, nil, err
	}
	if existing.DeletedAt != nil {
		return nil, nil, domain.NewSkillDeleted(name)
	}
	locked, err := s.skillRepo.LockAndLoad(ctx, tx, existing.ID)
	if err != nil {
		return nil, nil, err
	}
	if locked.DeletedAt != nil {
		return nil, nil, domain.NewSkillDeleted(name)
	}
	historical, err := s.revRepo.SelectBySkillAndNumber(ctx, tx, existing.ID, n)
	if err != nil {
		return nil, nil, err
	}
	maxRev, err := s.skillRepo.MaxRevisionNumber(ctx, tx, existing.ID)
	if err != nil {
		return nil, nil, fmt.Errorf("SkillService.Restore: MaxRevisionNumber: %w", err)
	}
	newRev := maxRev + 1
	note := fmt.Sprintf("restored from revision %d", n)

	rev := &domain.SkillRevision{
		SkillID:        existing.ID,
		RevisionNumber: newRev,
		Description:    historical.Description,
		Body:           historical.Body,
		ChangeNote:     &note,
	}
	if err := s.revRepo.Insert(ctx, tx, rev); err != nil {
		return nil, nil, fmt.Errorf("SkillService.Restore: revision Insert: %w", err)
	}
	if err := s.skillRepo.UpdateBody(ctx, tx, existing.ID, historical.Body, historical.Description); err != nil {
		return nil, nil, err
	}

	if err := tx.Commit(); err != nil {
		return nil, nil, fmt.Errorf("SkillService.Restore: Commit: %w", err)
	}

	updated, err := s.skillRepo.SelectByID(ctx, s.db, existing.ID)
	if err != nil {
		return nil, nil, fmt.Errorf("SkillService.Restore: re-read: %w", err)
	}
	return updated, rev, nil
}

// ---------------------------------------------------------------------------
// SoftDelete (DELETE /skills/:name)
// ---------------------------------------------------------------------------

// SoftDelete sets deleted_at = now() on the skill. Idempotent:
// calling it twice returns nil on both calls (spec SCN-2.3).
func (s *SkillService) SoftDelete(ctx context.Context, name string) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("SkillService.SoftDelete: BeginTx: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	existing, err := s.skillRepo.SelectBySlug(ctx, tx, name)
	if err != nil {
		// NotFound = idempotent success (already deleted or never existed).
		var nerr *domain.NotFoundError
		if errors.As(err, &nerr) {
			return tx.Commit()
		}
		return err
	}
	if err := s.skillRepo.SoftDelete(ctx, tx, existing.ID); err != nil {
		return err
	}
	return tx.Commit()
}

// ---------------------------------------------------------------------------
// Reads (no transaction needed) — listed here for completeness; the
// remaining use cases land in subsequent TDD tasks (3.5..3.12).
// ---------------------------------------------------------------------------

// GetBySlug returns the current skill row for the name. Soft-deleted
// skills return *domain.NotFoundError (404 — spec SCN-5.4).
func (s *SkillService) GetBySlug(ctx context.Context, name string) (*domain.Skill, error) {
	return s.skillRepo.SelectBySlug(ctx, s.db, name)
}

// List returns active skills ordered by updated_at DESC. The service
// clamps limit to domain.MaxListLimit (200) and defaults to
// domain.DefaultListLimit (50) when the caller passes 0 or negative.
func (s *SkillService) List(ctx context.Context, limit int) ([]*domain.Skill, error) {
	if limit <= 0 {
		limit = domain.DefaultListLimit
	}
	if limit > domain.MaxListLimit {
		limit = domain.MaxListLimit
	}
	return s.skillRepo.List(ctx, s.db, limit)
}

// ListRevisions returns the revision history of the skill, newest
// first. Returns *domain.NotFoundError if the skill does not exist or
// is soft-deleted.
func (s *SkillService) ListRevisions(ctx context.Context, name string) ([]*domain.SkillRevision, error) {
	sk, err := s.skillRepo.SelectBySlug(ctx, s.db, name)
	if err != nil {
		return nil, err
	}
	return s.revRepo.ListBySkillID(ctx, s.db, sk.ID)
}

// _ keeps the `errors` import live for tasks 3.7/3.8 that introduce
// the SoftDelete + GoneError branching in the same package. Removing
// this placeholder now would force a churn import dance in the next
// commits; safe to remove once Restore / Update land.
var _ = errors.As