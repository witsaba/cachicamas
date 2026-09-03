// Package auth — organizations_test.go locks the Organization value-
// object contract per spec R-DB-003 / R-BOOTSTRAP-1.
//
// The tests are pure-domain: no DB, no HTTP. The integration tests
// in src/infrastructure/postgres/organization_repo_test.go cover the
// DB-backed behaviour (FK RESTRICT, slug uniqueness, soft-delete
// filtering) under INTEGRATION=1.
package auth

import (
	"testing"
)

// TestOrganization_NewOrganization covers the constructor invariant:
// every newly-built Organization MUST carry a non-empty Name (the
// organization display name) and a non-empty OwnerID (the FK to
// auth.users). Slug is intentionally left empty — the SQL layer
// sets it to 'pyme-<userID>' per design §1.2 step 4.
func TestOrganization_NewOrganization(t *testing.T) {
	o := NewOrganization(42, "Acme Studio")
	if o == nil {
		t.Fatal("NewOrganization: returned nil")
	}
	if o.OwnerID != 42 {
		t.Errorf("NewOrganization().OwnerID = %d, want 42", o.OwnerID)
	}
	if o.Name != "Acme Studio" {
		t.Errorf("NewOrganization().Name = %q, want %q", o.Name, "Acme Studio")
	}
}

// TestOrganization_Validate covers the construction-time invariants
// the bootstrap service relies on: an organization without a name or
// owner_id must be rejected so the SQL layer cannot surface a NOT
// NULL violation as a 500.
func TestOrganization_Validate(t *testing.T) {
	t.Run("empty name", func(t *testing.T) {
		o := NewOrganization(1, "")
		if err := o.Validate(); err == nil {
			t.Fatal("Organization{Name: \"\"}.Validate(): expected error, got nil")
		}
	})
	t.Run("zero owner_id", func(t *testing.T) {
		o := NewOrganization(0, "Acme Studio")
		if err := o.Validate(); err == nil {
			t.Fatal("Organization{OwnerID: 0}.Validate(): expected error, got nil")
		}
	})
	t.Run("valid organization", func(t *testing.T) {
		o := NewOrganization(1, "Acme Studio")
		if err := o.Validate(); err != nil {
			t.Fatalf("Organization{Name, OwnerID populated}.Validate(): unexpected error %v", err)
		}
	})
}

// TestOrganization_DerivePymeName covers the display-name derivation
// rule (design §1.2 step 4): the bootstrap service derives the
// organization name from the email local-part, lowercased, with the
// fallback "pyme" when the local-part is empty.
func TestOrganization_DerivePymeName(t *testing.T) {
	cases := []struct {
		email string
		want  string
	}{
		{"founder@example.com", "founder"},
		{"FOUNDER@example.COM", "founder"},
		{"jane.doe+test@startup.io", "jane.doe+test"},
		{"@boring.com", pymeNameFallback},
		{"", pymeNameFallback},
	}
	for _, c := range cases {
		t.Run(c.email, func(t *testing.T) {
			got := DerivePymeNameFromEmail(c.email)
			if got != c.want {
				t.Errorf("DerivePymeNameFromEmail(%q) = %q, want %q", c.email, got, c.want)
			}
		})
	}
}