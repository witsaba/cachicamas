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