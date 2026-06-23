// Package domain_test contains the test suite for the organization
// domain entity. This file covers the validation rules locked in
// spec §2.3 + §4 of openspec/changes/organizations-first-front/.
//
// Strict TDD discipline (per openspec/AGENTS.md and
// sdd-init/cachicamas): this file was written BEFORE
// organization.go existed. Running `go test ./src/domain/...` with
// no Validate function in the domain package must fail with
// "undefined: Validate" — that failure IS the RED step.
package domain_test

import (
	"errors"
	"reflect"
	"strings"
	"testing"

	"github.com/cachicamas/backend/database_administrator/src/domain"
)

// Locked error message vocabulary (from spec §4). These constants
// pin the strings a test asserts against; the production code in
// organization.go re-exports them so a typo in one place cannot
// diverge from the spec.
const (
	msgNameRequired        = "Name is required."
	msgNameLength          = "Name must be 3–120 characters."
	msgSlugRequired        = "Slug is required."
	msgSlugFormat          = "Slug must be 3–60 characters, lowercase letters, digits, and hyphens; must start and end with a letter or digit."
	msgShortnameLength     = "Short name must be 40 characters or fewer."
	msgEmailFormat         = "Email is not a valid email address."
	msgPhoneFormat         = "Phone must be in E.164 format (e.g. +14155552671)."
)

// TestValidate_FullName covers spec §2.3 rule 1: full_name is
// required, 3-120 chars after trim. We assert the *ValidationError
// type, its Code(), and the locked message string in the Fields map.
func TestValidate_FullName(t *testing.T) {
	cases := []struct {
		name    string
		full    string
		wantKey string
		wantMsg string
	}{
		{
			name:    "missing full_name",
			full:    "",
			wantKey: "full_name",
			wantMsg: msgNameRequired,
		},
		{
			name:    "whitespace-only full_name is treated as missing after trim",
			full:    "   ",
			wantKey: "full_name",
			wantMsg: msgNameRequired,
		},
		{
			name:    "too short (2 chars after trim)",
			full:    "ab",
			wantKey: "full_name",
			wantMsg: msgNameLength,
		},
		{
			name:    "too long (121 chars after trim)",
			full:    strings.Repeat("a", 121),
			wantKey: "full_name",
			wantMsg: msgNameLength,
		},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			err := domain.Validate(domain.CreateOrganizationInput{
				FullName:       tc.full,
				Identification: "acme",
			})
			if err == nil {
				t.Fatalf("Validate: expected error, got nil")
			}
			var verr *domain.ValidationError
			if !errors.As(err, &verr) {
				t.Fatalf("Validate returned %T, want *ValidationError", err)
			}
			if got := verr.Code(); got != "validation" {
				t.Errorf("Code() = %q, want %q", got, "validation")
			}
			if got := verr.Fields[tc.wantKey]; got != tc.wantMsg {
				t.Errorf("Fields[%q] = %q, want %q", tc.wantKey, got, tc.wantMsg)
			}
		})
	}
}

// TestValidate_Identification covers spec §2.3 rule 2: the slug
// regex `^[a-z0-9][a-z0-9-]{1,58}[a-z0-9]$`. We hit the four locked
// failure cases (uppercase, too short, too long, leading hyphen) and
// the missing-slug case.
func TestValidate_Identification(t *testing.T) {
	cases := []struct {
		name    string
		ident   string
		wantKey string
		wantMsg string
	}{
		{
			name:    "missing identification",
			ident:   "",
			wantKey: "identification",
			wantMsg: msgSlugRequired,
		},
		{
			name:    "uppercase letter rejected",
			ident:   "Acme-Industrial",
			wantKey: "identification",
			wantMsg: msgSlugFormat,
		},
		{
			name:    "too short (2 chars)",
			ident:   "ab",
			wantKey: "identification",
			wantMsg: msgSlugFormat,
		},
		{
			name:    "too long (61 chars)",
			ident:   "a" + strings.Repeat("b", 59) + "c",
			wantKey: "identification",
			wantMsg: msgSlugFormat,
		},
		{
			name:    "leading hyphen rejected",
			ident:   "-acme",
			wantKey: "identification",
			wantMsg: msgSlugFormat,
		},
		{
			name:    "trailing hyphen rejected",
			ident:   "acme-",
			wantKey: "identification",
			wantMsg: msgSlugFormat,
		},
		{
			name:    "embedded underscore rejected",
			ident:   "acme_industrial",
			wantKey: "identification",
			wantMsg: msgSlugFormat,
		},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			err := domain.Validate(domain.CreateOrganizationInput{
				FullName:       "Acme",
				Identification: tc.ident,
			})
			if err == nil {
				t.Fatalf("Validate: expected error, got nil")
			}
			var verr *domain.ValidationError
			if !errors.As(err, &verr) {
				t.Fatalf("Validate returned %T, want *ValidationError", err)
			}
			if got := verr.Code(); got != "validation" {
				t.Errorf("Code() = %q, want %q", got, "validation")
			}
			if got := verr.Fields[tc.wantKey]; got != tc.wantMsg {
				t.Errorf("Fields[%q] = %q, want %q", tc.wantKey, got, tc.wantMsg)
			}
		})
	}
}

// TestValidate_Shortname covers spec §2.3 rule 3: shortname is
// optional, 0-40 chars when present.
func TestValidate_Shortname(t *testing.T) {
	t.Run("41 chars rejected", func(t *testing.T) {
		err := domain.Validate(domain.CreateOrganizationInput{
			FullName:       "Acme",
			Identification: "acme",
			ShortName:      stringPtr(strings.Repeat("s", 41)),
		})
		if err == nil {
			t.Fatalf("Validate: expected error, got nil")
		}
		var verr *domain.ValidationError
		if !errors.As(err, &verr) {
			t.Fatalf("Validate returned %T, want *ValidationError", err)
		}
		if got := verr.Fields["shortname"]; got != msgShortnameLength {
			t.Errorf("Fields[shortname] = %q, want %q", got, msgShortnameLength)
		}
	})

	t.Run("40 chars accepted", func(t *testing.T) {
		err := domain.Validate(domain.CreateOrganizationInput{
			FullName:       "Acme",
			Identification: "acme",
			ShortName:      stringPtr(strings.Repeat("s", 40)),
		})
		if err != nil {
			t.Fatalf("Validate: unexpected error: %v", err)
		}
	})
}

// TestValidate_Email covers spec §2.3 rule 4: email is optional,
// must be RFC 5322 valid when present.
func TestValidate_Email(t *testing.T) {
	t.Run("invalid email rejected", func(t *testing.T) {
		err := domain.Validate(domain.CreateOrganizationInput{
			FullName:       "Acme",
			Identification: "acme",
			Email:          stringPtr("not-an-email"),
		})
		if err == nil {
			t.Fatalf("Validate: expected error, got nil")
		}
		var verr *domain.ValidationError
		if !errors.As(err, &verr) {
			t.Fatalf("Validate returned %T, want *ValidationError", err)
		}
		if got := verr.Fields["email"]; got != msgEmailFormat {
			t.Errorf("Fields[email] = %q, want %q", got, msgEmailFormat)
		}
	})

	t.Run("valid email accepted", func(t *testing.T) {
		err := domain.Validate(domain.CreateOrganizationInput{
			FullName:       "Acme",
			Identification: "acme",
			Email:          stringPtr("hello@acme.example"),
		})
		if err != nil {
			t.Fatalf("Validate: unexpected error: %v", err)
		}
	})
}

// TestValidate_Phone covers spec §2.3 rule 5: phone is optional,
// must match E.164 `^\+[1-9]\d{1,14}$` when present.
func TestValidate_Phone(t *testing.T) {
	t.Run("missing + prefix rejected", func(t *testing.T) {
		err := domain.Validate(domain.CreateOrganizationInput{
			FullName:       "Acme",
			Identification: "acme",
			Phone:          stringPtr("4155552671"),
		})
		if err == nil {
			t.Fatalf("Validate: expected error, got nil")
		}
		var verr *domain.ValidationError
		if !errors.As(err, &verr) {
			t.Fatalf("Validate returned %T, want *ValidationError", err)
		}
		if got := verr.Fields["phone"]; got != msgPhoneFormat {
			t.Errorf("Fields[phone] = %q, want %q", got, msgPhoneFormat)
		}
	})

	t.Run("leading zero after + rejected", func(t *testing.T) {
		err := domain.Validate(domain.CreateOrganizationInput{
			FullName:       "Acme",
			Identification: "acme",
			Phone:          stringPtr("+0123456789"),
		})
		if err == nil {
			t.Fatalf("Validate: expected error, got nil")
		}
		var verr *domain.ValidationError
		if !errors.As(err, &verr) {
			t.Fatalf("Validate returned %T, want *ValidationError", err)
		}
		if got := verr.Fields["phone"]; got != msgPhoneFormat {
			t.Errorf("Fields[phone] = %q, want %q", got, msgPhoneFormat)
		}
	})

	t.Run("valid E.164 accepted", func(t *testing.T) {
		err := domain.Validate(domain.CreateOrganizationInput{
			FullName:       "Acme",
			Identification: "acme",
			Phone:          stringPtr("+14155552671"),
		})
		if err != nil {
			t.Fatalf("Validate: unexpected error: %v", err)
		}
	})
}

// TestValidate_AllOptionalOmitted asserts the happy-path minimum:
// only full_name and identification; everything else nil/empty.
// This is the canonical valid input for B-1.
func TestValidate_AllOptionalOmitted(t *testing.T) {
	err := domain.Validate(domain.CreateOrganizationInput{
		FullName:       "Acme Industrial S.A.",
		Identification: "acme-industrial",
	})
	if err != nil {
		t.Fatalf("Validate: unexpected error: %v", err)
	}
}

// TestValidate_MultipleFieldsError asserts that Validate reports
// ALL failing fields in the Fields map, not just the first one.
// This is what spec §3.1 says: "The fields map contains only the
// failing fields."
func TestValidate_MultipleFieldsError(t *testing.T) {
	err := domain.Validate(domain.CreateOrganizationInput{
		FullName:       "",
		Identification: "AB",
		Email:          stringPtr("nope"),
	})
	if err == nil {
		t.Fatalf("Validate: expected error, got nil")
	}
	var verr *domain.ValidationError
	if !errors.As(err, &verr) {
		t.Fatalf("Validate returned %T, want *ValidationError", err)
	}
	if got := verr.Fields["full_name"]; got != msgNameRequired {
		t.Errorf("Fields[full_name] = %q, want %q", got, msgNameRequired)
	}
	if got := verr.Fields["identification"]; got != msgSlugFormat {
		t.Errorf("Fields[identification] = %q, want %q", got, msgSlugFormat)
	}
	if got := verr.Fields["email"]; got != msgEmailFormat {
		t.Errorf("Fields[email] = %q, want %q", got, msgEmailFormat)
	}
}

// TestOrganization_OptionalFieldsSerializeAsNull locks the spec
// §2.1 contract: optional *string fields must serialize as JSON
// `null` (NOT be omitted) when nil, so consumers can distinguish
// "not provided" from "empty string". This is what the locked
// decision #11 in the apply prompt is about.
func TestOrganization_OptionalFieldsSerializeAsNull(t *testing.T) {
	// 1. Nil optionals on a populated struct (no validation
	// differences — this only exercises the type system).
	empty := domain.Organization{
		ID:             42,
		FullName:       "Acme",
		Identification: "acme",
		IsActive:       true,
	}
	if got := empty.ShortName; got != nil {
		t.Errorf("ShortName = %v, want nil", got)
	}
	if got := empty.Email; got != nil {
		t.Errorf("Email = %v, want nil", got)
	}
	if got := empty.Phone; got != nil {
		t.Errorf("Phone = %v, want nil", got)
	}

	// 2. The struct's JSON tags must NOT carry `omitempty` —
	// otherwise a nil *string is dropped from the wire output
	// instead of becoming JSON null. Inspect the struct field
	// tags directly so a future contributor who adds `omitempty`
	// back gets a red test instead of a silent spec violation.
	if got := jsonTag(t, "ShortName"); got != "shortname" {
		t.Errorf("ShortName json tag = %q, want %q (must NOT include ,omitempty)", got, "shortname")
	}
	if got := jsonTag(t, "Email"); got != "email" {
		t.Errorf("Email json tag = %q, want %q (must NOT include ,omitempty)", got, "email")
	}
	if got := jsonTag(t, "Phone"); got != "phone" {
		t.Errorf("Phone json tag = %q, want %q (must NOT include ,omitempty)", got, "phone")
	}
}

// jsonTag returns the value of the `json` struct tag on a named
// field of domain.Organization, or "" if the tag is absent. Uses
// reflection so a future rename in the struct is caught by the
// test runner, not by a silent string-match in the assertion.
func jsonTag(t *testing.T, fieldName string) string {
	t.Helper()
	v := reflect.TypeOf(domain.Organization{})
	f, ok := v.FieldByName(fieldName)
	if !ok {
		t.Fatalf("Organization has no field %q", fieldName)
	}
	return f.Tag.Get("json")
}

// stringPtr is a tiny helper to take the address of a string literal
// in test fixtures.
func stringPtr(s string) *string { return &s }
