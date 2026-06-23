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
	"errors"
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
	spanNameCreate = "organization.create"
	spanNameList   = "organization.list"
	spanNameGet    = "organization.get"
)

// Locked HTTP route strings for span attributes.
const (
	httpRouteList  = "/organizations"
	httpRouteGet   = "/organizations/:id"
	httpRoutePost  = "/organizations"
)

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

// List returns every organization, ordered by (created_at ASC,
// id ASC). On success the span carries organization.count so a
// Jaeger query can chart list-size without a separate metric.
func (s *OrganizationService) List(ctx context.Context) ([]domain.Organization, error) {
	ctx, span := s.tracer.Start(ctx, spanNameList)
	defer span.End()

	setHTTPRouteAttrs(span, "GET", httpRouteList)

	orgs, err := s.repo.SelectAll(ctx)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return nil, err
	}

	span.SetAttributes(attribute.Int("organization.count", len(orgs)))
	setHTTPStatus(span, 200)
	s.logger.InfoContext(ctx, "organization.list ok",
		slog.Int("organization.count", len(orgs)),
	)
	return orgs, nil
}

// Get returns a single organization by id. The id is a path
// param, so the span always carries organization.id (whether or
// not the row exists) — this is what spec §3.4 says.
func (s *OrganizationService) Get(ctx context.Context, id int64) (*domain.Organization, error) {
	ctx, span := s.tracer.Start(ctx, spanNameGet)
	defer span.End()

	setHTTPRouteAttrs(span, "GET", httpRouteGet)
	span.SetAttributes(attribute.Int64("organization.id", id))

	org, err := s.repo.SelectByID(ctx, id)
	if err != nil {
		var nerr *domain.NotFoundError
		if errors.As(err, &nerr) {
			setHTTPStatus(span, 404)
		} else {
			span.RecordError(err)
			span.SetStatus(codes.Error, err.Error())
		}
		return nil, err
	}

	setHTTPStatus(span, 200)
	s.logger.InfoContext(ctx, "organization.get ok",
		slog.Int64("organization.id", org.ID),
	)
	return org, nil
}

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

// Compile-time check that the service struct exposes the three
// use cases the handler needs. If a method is renamed, the build
// breaks here.
var _ interface {
	Create(ctx context.Context, in domain.CreateOrganizationInput) (*domain.Organization, error)
	List(ctx context.Context) ([]domain.Organization, error)
	Get(ctx context.Context, id int64) (*domain.Organization, error)
} = (*OrganizationService)(nil)
