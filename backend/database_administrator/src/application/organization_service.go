// Package application contains the use cases of the database
// administrator service. This file implements the organization
// use cases: Create, List, Get. Each use case opens an OTel
// span (locked names from spec §3.4) and emits a structured
// slog line on success or failure.
//
// Hexagonal boundary (design §4):
//
//   - This file imports domain (the port) and the stdlib
//     observability stack (slog, OTel trace). It does NOT import
//     pressly/goose or jackc/pgx.
//   - The pgx-backed adapter lives in src/infrastructure/postgres/.
//   - main.go wires the adapter to this service via the
//     domain.OrganizationRepository port.
package application

import (
	"context"
	"fmt"
	"log/slog"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
	"go.opentelemetry.io/otel/trace/noop"

	"github.com/cachicamas/backend/database_administrator/src/domain"
)

// Locked OTel span names (spec §3.4). Centralised so any future
// caller that wants to record these spans uses the same string.
const (
	spanNameCreate     = "organization.create"
	spanNameSetupState = "organization.setup_state"
)

// Locked HTTP route strings for span attributes.
const (
	httpRoutePost        = "/organizations"
	httpRouteSetupState  = "/setup-state"
)

// SetupState is the wire shape returned by GetSetupState. JSON tag
// matches the backend endpoint contract (R-OW-005 / S-OW-040):
//
//	{ "hasOrganization": <bool> }
type SetupState struct {
	HasOrganization bool `json:"hasOrganization"`
}

// OrganizationService is the use case facade for organization
// CRUD. It is the ONLY caller of domain.OrganizationRepository in
// the application layer; main.go is the composition root that
// wires the port to a concrete adapter.
type OrganizationService struct {
	repo   domain.OrganizationRepository
	logger *slog.Logger
	tracer trace.Tracer
}

// NewOrganizationService constructs an OrganizationService. The
// repo is the hexagonal port; production code wires it to a
// pgx-backed adapter (see src/infrastructure/postgres/organization_repo.go),
// tests wire it to an in-memory fake. tracer is the OTel Tracer
// used to open the locked span names; production code wires it
// via otel.Tracer("database_administrator"), tests wire it via
// an in-memory recorder.
func NewOrganizationService(repo domain.OrganizationRepository, logger *slog.Logger, tracer trace.Tracer) *OrganizationService {
	if logger == nil {
		logger = slog.Default()
	}
	if tracer == nil {
		tracer = noop.NewTracerProvider().Tracer("application/organization_service")
	}
	return &OrganizationService{
		repo:   repo,
		logger: logger,
		tracer: tracer,
	}
}

// Create validates the input, builds a *domain.Organization with
// is_active=true, and delegates persistence to the repo. On
// success the returned *Organization carries the DB-assigned id
// and the Postgres-set timestamps.
//
// OTel attributes (spec §3.4):
//
//   - http.method      = "POST"
//   - http.route       = "/organizations"
//   - http.status_code = 201 (success)
//   - organization.id  = <int64> (after successful insert)
//
// On validation failure the function returns immediately — no
// span, no repo call, no DB round-trip. (Validation errors are
// caller errors that do not need to be traced as a request; the
// HTTP layer logs the bad input.)
//
// On a unique-violation from the repo, the *ConflictError
// propagates as-is so the handler can map to HTTP 409.
func (s *OrganizationService) Create(ctx context.Context, in domain.CreateOrganizationInput) (*domain.Organization, error) {
	// Validation runs BEFORE the span so the request never appears
	// in the trace. The handler returns 400 to the client without
	// the validation logic having touched a single downstream
	// system.
	if err := domain.Validate(in); err != nil {
		return nil, err
	}

	ctx, span := s.tracer.Start(ctx, spanNameCreate)
	defer span.End()

	setHTTPRouteAttrs(span, "POST", httpRoutePost)

	org := &domain.Organization{
		ShortName:      in.ShortName,
		FullName:       in.FullName,
		Identification: in.Identification,
		IsActive:       true,
		Email:          in.Email,
		Phone:          in.Phone,
	}

	persisted, err := s.repo.Insert(ctx, org)
	if err != nil {
		// Propagate domain errors (ConflictError, NotFoundError,
		// InternalError) unchanged so the handler can errors.As
		// them and map to the locked HTTP envelope.
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return nil, err
	}

	span.SetAttributes(attribute.Int64("organization.id", persisted.ID))
	setHTTPStatus(span, 201)
	s.logger.InfoContext(ctx, "organization.create ok",
		slog.Int64("organization.id", persisted.ID),
		slog.String("identification", persisted.Identification),
	)
	return persisted, nil
}

// List and Get methods were removed in the 2026-07-06 ownboarding
// change. The /organizations list and get-by-id endpoints are gone;
// only POST /organizations (ownboarding submit) and GET /setup-state
// remain. The corresponding spanNameList, spanNameGet, httpRouteList,
// and httpRouteGet constants are gone too.

// ---------------------------------------------------------------------------
// Helpers — kept unexported and small so the three use cases
// don't duplicate attribute-setting code.
// ---------------------------------------------------------------------------

// setHTTPRouteAttrs sets the always-emitted HTTP method and route
// attributes on a span. The caller is responsible for any
// conditional attributes (e.g. organization.id) and for setting
// http.status_code after the operation completes via
// setHTTPStatus.
func setHTTPRouteAttrs(span trace.Span, method, route string) {
	span.SetAttributes(
		attribute.String("http.method", method),
		attribute.String("http.route", route),
	)
}

// setHTTPStatus records the HTTP status code on the span. Called
// after the operation completes so the value reflects the actual
// outcome (201/200/404/etc.), not the request-time guess.
func setHTTPStatus(span trace.Span, status int) {
	span.SetAttributes(attribute.Int("http.status_code", status))
}

// GetSetupState returns the install-level "is there at least one
// organization?" boolean. The ownboarding gate (frontend
// requireOwnboarding helper) reads this to decide whether to redirect
// the user to /ownboarding or let them land on /home.
//
// Why a service method instead of calling the repo from the handler:
// keeps the hexagonal boundary (handler depends on application, not
// on the repo adapter) and gives us a single place to add SPAN
// observability + slog logging for the setup-state check.
func (s *OrganizationService) GetSetupState(ctx context.Context) (SetupState, error) {
	ctx, span := s.tracer.Start(ctx, spanNameSetupState)
	defer span.End()

	setHTTPRouteAttrs(span, "GET", httpRouteSetupState)

	exists, err := s.repo.HasOrganization(ctx)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return SetupState{}, fmt.Errorf("get setup state: %w", err)
	}

	setHTTPStatus(span, 200)
	s.logger.InfoContext(ctx, "organization.setup_state ok",
		slog.Bool("has_organization", exists),
	)
	return SetupState{HasOrganization: exists}, nil
}

// Compile-time check that the service struct exposes the two
// use cases the handler needs. If a method is renamed, the build
// breaks here.
var _ interface {
	Create(ctx context.Context, in domain.CreateOrganizationInput) (*domain.Organization, error)
	GetSetupState(ctx context.Context) (SetupState, error)
} = (*OrganizationService)(nil)
