// Package auth — me.go implements the MeService use case (spec
// R-BE-003 / R-ME-1 / S-BE-030).
//
// The service is the ONLY place that joins user + organization for
// the /internal/me/:user_id endpoint. The HTTP handler is a thin
// wrapper that decodes the path parameter, invokes Get, and maps
// errors to HTTP envelopes (404 for unknown user, 401 for missing
// X-Internal-Secret via the T2.12 middleware).
package auth

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/cachicamas/backend/database_administrator/src/domain/auth"
)

// MeResult is the service's return type. The HTTP handler serialises
// it to {user, organization} per the locked wire contract.
type MeResult struct {
	User         *auth.User         `json:"user"`
	Organization *auth.Organization `json:"organization"`
}

// MeService orchestrates the user + organization lookup for the
// /internal/me endpoint. It depends on the two repository ports
// defined in the domain package + a *sql.DB handle (used as the
// Querier for both repositories; the MeService does not need
// transactional semantics since reads are independent).
type MeService struct {
	db    *sql.DB
	users auth.UserRepository
	orgs  auth.OrganizationRepository
}

// NewMeService wires the two repositories + a shared *sql.DB into a
// MeService. The db handle is used as the Querier for both
// repositories (single-statement reads, no transaction).
func NewMeService(db *sql.DB, users auth.UserRepository, orgs auth.OrganizationRepository) *MeService {
	return &MeService{db: db, users: users, orgs: orgs}
}

// Get loads the user + their organization in two roundtrips. The
// 1:1 MVP tenancy rule means a successful user lookup always has a
// matching organization; if the organization is missing (data
// integrity bug), the service still returns the user with a nil
// organization so the caller can branch.
//
// Errors:
//   - ErrValidation when id is non-positive (handler maps to 400).
//   - *auth.NotFoundError when the user does not exist (handler
//     maps to 404).
//   - Other errors are wrapped and returned as-is (handler maps
//     to 500).
func (s *MeService) Get(ctx context.Context, id int64) (*MeResult, error) {
	if id <= 0 {
		return nil, fmt.Errorf("%w: user_id must be positive (got %d)", ErrValidation, id)
	}
	user, err := s.users.FindByID(ctx, s.db, id)
	if err != nil {
		return nil, fmt.Errorf("auth.MeService.Get: find user: %w", err)
	}
	if user == nil {
		return nil, &auth.NotFoundError{Resource: "auth.users"}
	}
	org, err := s.orgs.FindByOwnerID(ctx, s.db, user.ID)
	if err != nil {
		// Don't fail the whole request for a missing org — return
		// the user with a nil org so the caller can branch.
		return &MeResult{User: user, Organization: nil}, fmt.Errorf("auth.MeService.Get: find org: %w", err)
	}
	return &MeResult{User: user, Organization: org}, nil
}

// Guard: ErrValidation must be reachable via errors.Is even when
// wrapped by fmt.Errorf("…: %w", ErrValidation). The fmt.Errorf
// call above does that automatically; this is a documentation line
// so the unused `errors` import does not get pruned during refactor.
var _ = errors.New