// Package domain contains the core business types of the database
// administrator service. It has no dependencies on frameworks,
// transport, or infrastructure.
//
// This file defines the Organization entity, the validation
// contract for CreateOrganizationInput, the OrganizationRepository
// port, and the four typed error values used by every layer above
// this one. Per design §4, the handler maps each error's Code() to
// the locked HTTP envelope; the application layer propagates the
// error as-is (do not wrap further — the handler uses errors.As).
package domain

import (
	"context"
	"net/mail"
	"regexp"
	"time"
)

// Organization is the domain entity for one row of the
// `organization` table (DDL: migration/sql/*_orgs_and_projects.sql).
// The struct mirrors the DDL column-for-column; the `db` tags tell
// the pgx adapter which column each field reads from, the `json`
// tags control wire serialization.
//
// Per locked decision #7, ID is BIGSERIAL end-to-end — same type in
// the DDL, in the Go struct, and on the wire (int64). No UUID
// mapping.
//
// Optional fields (shortname, email, phone) use a *string without
// `json:",omitempty"` so a nil pointer serializes as JSON `null`,
// not as an absent key. The spec §2.1 contract is that consumers
// can distinguish "not provided" (null) from "empty string" ("").
type Organization struct {
	ID             int64     `db:"id"             json:"id"`
	ShortName      *string   `db:"shortname"      json:"shortname"`
	FullName       string    `db:"full_name"      json:"full_name"`
	Identification string    `db:"identification" json:"identification"`
	IsActive       bool      `db:"is_active"      json:"is_active"`
	Email          *string   `db:"email"          json:"email"`
	Phone          *string   `db:"phone"          json:"phone"`
	CreatedAt      time.Time `db:"created_at"     json:"created_at"`
	UpdatedAt      time.Time `db:"updated_at"     json:"updated_at"`
}

// CreateOrganizationInput is the handler-facing payload for the
// POST /organizations use case. The handler decodes JSON or
// form-encoded bodies into this shape, then hands it to
// application.OrganizationService.Create. The validation rules
// below are the single source of truth on the server side; the
// frontend Zod schema mirrors them byte-for-byte.
type CreateOrganizationInput struct {
	FullName       string
	Identification string
	ShortName      *string
	Email          *string
	Phone          *string
}

// OrganizationRepository is the hexagonal port the application
// layer uses to read and persist Organization aggregates.
// Implementations live under src/infrastructure/postgres/. The
// interface is deliberately small so a fake is trivial to write
// in tests (see application/organization_service_test.go).
//
// Contract:
//
//   - Insert(ctx, org) MUST set org.ID, org.IsActive=true, and
//     org.CreatedAt / org.UpdatedAt from the database's own
//     clock (Postgres DEFAULT now()) and return the populated
//     org. The application layer never sets timestamps itself.
//   - SelectAll(ctx) returns rows ordered by (created_at ASC, id
//     ASC) so the result is deterministic across calls.
//   - SelectByID(ctx, id) returns *NotFoundError when no row
//     matches. The handler maps NotFoundError to HTTP 404.
//   - All three methods honour the context.
type OrganizationRepository interface {
	Insert(ctx context.Context, o *Organization) (*Organization, error)
	SelectAll(ctx context.Context) ([]Organization, error)
	SelectByID(ctx context.Context, id int64) (*Organization, error)
	HasOrganization(ctx context.Context) (bool, error)
}

// ---------------------------------------------------------------------------
// Locked error message vocabulary (spec §4). Single source of truth
// for every error response. A typo in one of these constants will
// fail a test; do not duplicate these strings in the handler.
// ---------------------------------------------------------------------------

const (
	// MsgNameRequired is the field-level error message for missing full_name.
	MsgNameRequired = "Name is required."
	// MsgNameLength is the field-level error message for full_name length.
	MsgNameLength = "Name must be 3–120 characters."
	// MsgSlugRequired is the field-level error message for missing identification (slug).
	MsgSlugRequired = "Slug is required."
	// MsgSlugFormat is the field-level error message for identification (slug) format.
	MsgSlugFormat = "Slug must be 3–60 characters, lowercase letters, digits, and hyphens; must start and end with a letter or digit."
	// MsgShortnameLength is the field-level error message for shortname length.
	MsgShortnameLength = "Short name must be 40 characters or fewer."
	// MsgEmailFormat is the field-level error message for email RFC 5322 validity.
	MsgEmailFormat = "Email is not a valid email address."
	// MsgPhoneFormat is the field-level error message for phone E.164 format.
	MsgPhoneFormat = "Phone must be in E.164 format (e.g. +14155552671)."

	// MsgConflictSlug is the message shown when an organization identification is already taken.
	MsgConflictSlug = "This slug is already taken. Try another."
	// MsgNotFound is the message shown when an organization does not exist.
	MsgNotFound = "Organization not found."
	// MsgServerFailure is the message shown for an unexpected, non-categorized failure.
	MsgServerFailure = "Something went wrong. Please try again."

	// CodeValidation is the locked vocabulary for field-level validation failures.
	CodeValidation = "validation"
	// CodeConflict is the locked vocabulary for unique-constraint violations.
	CodeConflict = "conflict"
	// CodeNotFound is the locked vocabulary for missing-row responses.
	CodeNotFound = "not_found"
	// CodeServer is the locked vocabulary for unexpected internal failures.
	CodeServer = "server"
)

// ---------------------------------------------------------------------------
// Validation regexes (spec §2.3).
// ---------------------------------------------------------------------------

// slugRegex matches the locked slug pattern: 3-60 chars total,
// must start and end with [a-z0-9], inner chars include hyphens.
// Single source of truth for the server-side slug rule.
var slugRegex = regexp.MustCompile(`^[a-z0-9][a-z0-9-]{1,58}[a-z0-9]$`)

// e164Regex matches the locked phone pattern: a leading `+`, a
// non-zero first digit, then 1-14 more digits.
var e164Regex = regexp.MustCompile(`^\+[1-9]\d{1,14}$`)

// Length caps (spec §2.3).
const (
	fullNameMinLen  = 3
	fullNameMaxLen  = 120
	shortnameMaxLen = 40
	slugMinLen      = 3
	slugMaxLen      = 60
)

// Validate runs every rule from spec §2.3 against the input and
// returns a *ValidationError with the failing fields populated
// when at least one rule fails. When every rule passes, Validate
// returns nil. The function is pure: no I/O, no clock, no logging.
//
// On failure, the returned *ValidationError's Fields map contains
// ONE entry per failing field. The handler reads Fields[fc] to
// render the field-level error envelope. Order of rules does not
// matter — the consumer iterates the map.
func Validate(in CreateOrganizationInput) error {
	fields := map[string]string{}

	fullName := trim(in.FullName)
	if fullName == "" {
		fields["full_name"] = MsgNameRequired
	} else if len(fullName) < fullNameMinLen || len(fullName) > fullNameMaxLen {
		fields["full_name"] = MsgNameLength
	}

	if in.Identification == "" {
		fields["identification"] = MsgSlugRequired
	} else if len(in.Identification) < slugMinLen || len(in.Identification) > slugMaxLen {
		fields["identification"] = MsgSlugFormat
	} else if !slugRegex.MatchString(in.Identification) {
		fields["identification"] = MsgSlugFormat
	}

	if in.ShortName != nil {
		if len(*in.ShortName) > shortnameMaxLen {
			fields["shortname"] = MsgShortnameLength
		}
	}

	if in.Email != nil {
		// The locked contract is RFC 5322 validity; net/mail.ParseAddress
		// is the canonical Go implementation. Empty string is treated
		// as "not provided" (the handler maps the nil-vs-empty-string
		// distinction at the wire boundary; by the time we reach
		// Validate, nil means absent and "" would have already been
		// filtered out by the handler).
		if *in.Email != "" {
			if _, err := mail.ParseAddress(*in.Email); err != nil {
				fields["email"] = MsgEmailFormat
			}
		}
	}

	if in.Phone != nil {
		if *in.Phone != "" {
			if !e164Regex.MatchString(*in.Phone) {
				fields["phone"] = MsgPhoneFormat
			}
		}
	}

	if len(fields) == 0 {
		return nil
	}
	return &ValidationError{Fields: fields}
}

// ---------------------------------------------------------------------------
// Typed errors (spec §3 + design §4).
//
// Each error type implements the AppError interface — a regular
// `error` plus a Code() string method. The handler uses a type
// switch (or errors.As + map) to map Code() to the locked HTTP
// envelope. The application layer propagates these as-is so the
// handler can recover them via errors.As.
// ---------------------------------------------------------------------------

// AppError is the marker interface for every error in this
// package that the handler maps to a specific HTTP envelope.
type AppError interface {
	error
	Code() string
}

// ValidationError signals one or more field-level validation
// failures. The Fields map keys are the wire names (e.g.
// "full_name", "identification") and the values are the locked
// error message strings from spec §4.
type ValidationError struct {
	Fields map[string]string
}

func (e *ValidationError) Error() string {
	return "validation failed"
}

// Code returns the CodeValidation string (handler maps to 422 envelope).
func (e *ValidationError) Code() string { return CodeValidation }

// ConflictError signals a uniqueness violation on a field that
// the database guards (e.g. organization.identification). The
// Cause wraps the original pgx error so slog can log it; the
// user-facing message is the locked vocabulary string.
type ConflictError struct {
	Cause error
}

func (e *ConflictError) Error() string {
	if e.Cause != nil {
		return "conflict: " + e.Cause.Error()
	}
	return "conflict"
}

// Code returns the CodeConflict string (handler maps to 409 envelope).
func (e *ConflictError) Code() string { return CodeConflict }

// NotFoundError signals that a row lookup returned no rows. The
// Resource names which entity was missing (e.g. "organization").
// The handler currently ignores Resource and uses the generic
// 404 envelope, but the field is part of the contract so future
// resources can carry their own locked message.
type NotFoundError struct {
	Resource string
}

func (e *NotFoundError) Error() string { return e.Resource + " not found" }

// Code returns the CodeNotFound string (handler maps to 404 envelope).
func (e *NotFoundError) Code() string { return CodeNotFound }

// InternalError signals an unexpected failure the application
// did not categorize. The Cause is logged via slog; the
// user-facing message is the locked vocabulary string and never
// the underlying error.
type InternalError struct {
	Cause error
}

func (e *InternalError) Error() string {
	if e.Cause != nil {
		return "internal: " + e.Cause.Error()
	}
	return "internal error"
}

// Code returns the CodeServer string (handler maps to 500 envelope).
func (e *InternalError) Code() string { return CodeServer }

// trim removes leading and trailing whitespace from a string.
// Using strings.TrimSpace would add an import; the implementation
// is one line and keeps the file self-contained.
func trim(s string) string {
	start := 0
	end := len(s)
	for start < end {
		c := s[start]
		if c != ' ' && c != '\t' && c != '\n' && c != '\r' {
			break
		}
		start++
	}
	for end > start {
		c := s[end-1]
		if c != ' ' && c != '\t' && c != '\n' && c != '\r' {
			break
		}
		end--
	}
	return s[start:end]
}
