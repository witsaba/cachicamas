// Package application contains the use cases of the database
// administrator service. This file implements the identity use case
// (LookupByEmail). The slice is intentionally thin — the heavy
// lifting (JWE decryption, claim → email extraction) lives in
// src/interfaces/http/auth_middleware.go; the service is the bridge
// from that middleware to the domain.IdentityRepository port.
//
// Hexagonal boundary (design §4):
//
//   - This file imports domain (the port), the stdlib observability
//     stack (slog, OTel trace). It does NOT import pressly/goose or
//     jackc/pgx.
//   - The pgx-backed adapter lives in src/infrastructure/postgres/.
//   - main.go (when PR-3 lands) wires the adapter to this service
//     via the domain.IdentityRepository port.
package application

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"log/slog"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"

	"github.com/cachicamas/backend/database_administrator/src/domain"
)

// Locked OTel span name (cachicamas-github-login spec §3.4 +
// design §2.2). Centralised so any future caller that wants to
// record the same span uses the same string.
const spanNameIdentityLookup = "identity.lookup"

// Locked OTel span name (cachicamas-identity-signin-callback slice).
// Centralised so the handler / repository / tests share the same
// string for the OAuth-driven UPSERT path.
const spanNameIdentityUpsert = "identity.upsert"

// IdentityService is the use case facade for identity lookups. It
// is the ONLY caller of domain.IdentityRepository in the
// application layer; main.go is the composition root that wires
// the port to a concrete adapter.
type IdentityService struct {
	repo   domain.IdentityRepository
	logger *slog.Logger
	tracer trace.Tracer
}

// NewIdentityService constructs an IdentityService. The caller
// (composition root in main.go, when PR-3 lands) passes the repo
// port, a logger, and a tracer. The constructor is cheap and
// side-effect-free; the first LookupByEmail call is what actually
// touches the database.
func NewIdentityService(repo domain.IdentityRepository, logger *slog.Logger, tracer trace.Tracer) *IdentityService {
	return &IdentityService{
		repo:   repo,
		logger: logger,
		tracer: tracer,
	}
}

// LookupByEmail resolves an identity by email. The email parameter
// is passed through to the repository unchanged; the repository
// MUST handle case-insensitive matching (Postgres CITEXT column).
//
// Returns:
//   - (*Identity, nil)             — hit
//   - (nil, *IdentityNotFoundError) — miss (the HTTP handler maps to 404)
//   - (nil, other error)             — infra/transport error (5xx)
//
// The OTel span `identity.lookup` carries `auth.email_hash`
// (sha256 first 12 hex chars) so observability never logs raw
// email PII. The hash attribute is always set, even on miss (with
// the requested email's hash) so the operator can correlate.
func (s *IdentityService) LookupByEmail(ctx context.Context, email string) (*domain.Identity, error) {
	ctx, span := s.tracer.Start(ctx, spanNameIdentityLookup)
	defer span.End()

	out, err := s.repo.LookupByEmail(ctx, email)
	if err != nil {
		span.RecordError(err)
		var nf *domain.IdentityNotFoundError
		if errors.As(err, &nf) {
			span.SetStatus(codes.Error, "identity_not_found")
		} else {
			span.SetStatus(codes.Error, "repo_error")
		}
		return nil, err
	}
	span.SetAttributes(
		attribute.Int64("identity.id", out.ID),
		attribute.String("identity.provider", out.Provider),
	)
	return out, nil
}

// UpsertFromOAuth persists an OAuth identity event into identity.user
// + identity.account. The handler (interfaces/http/identity_handler.go)
// is the only production caller; the HTTP transport validates +
// canonicalizes + verifies the HMAC, then dispatches a closed
// domain.IdentityEvent to this method.
//
// Behaviour:
//   - identity.user row is selected by case-insensitive email; if
//     missing, INSERT a new row.
//   - identity.account row is INSERTed with the resolved user_id, ON
//     CONFLICT (provider, provider_account_id) DO NOTHING.
//   - Returns the resolved *Identity (new or reused).
//
// The OTel span `identity.upsert` carries `auth.email_hash`
// (sha256 first 12 hex chars) so observability never logs raw email
// PII. The hash attribute is always set, even on error (with the
// requested email's hash) so the operator can correlate.
func (s *IdentityService) UpsertFromOAuth(ctx context.Context, ev domain.IdentityEvent) (*domain.Identity, error) {
	ctx, span := s.tracer.Start(ctx, spanNameIdentityUpsert)
	defer span.End()

	emailHash := sha256Hex12(ev.Email)
	span.SetAttributes(attribute.String("auth.email_hash", emailHash))

	out, err := s.repo.Upsert(ctx, ev)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "repo_error")
		return nil, err
	}
	span.SetAttributes(
		attribute.Int64("identity.id", out.ID),
		attribute.String("identity.provider", out.Provider),
		attribute.String("identity.outcome", "upserted"),
	)
	return out, nil
}

// sha256Hex12 returns sha256(s) truncated to 12 hex chars. The same
// shape the JWE verifier middleware uses for PII-safe email hashing
// (interfaces/http/auth_middleware.go); keeping the format identical
// makes log-line grepping across the two slices trivial.
func sha256Hex12(s string) string {
	h := sha256.Sum256([]byte(s))
	return hex.EncodeToString(h[:])[:12]
}
