// Package auth contains the domain types for the google-auth-bootstrap slice.
//
// users_test.go locks the User value-object contract per spec R-DB-001 /
// R-DB-006 / S-DB-050 and the UserRepository port (spec R-BOOTSTRAP-1).
//
// The tests are pure-domain: no DB, no HTTP, no goroutines. The
// integration tests in src/infrastructure/postgres/user_repo_test.go cover
// the DB-backed behaviour (created_at immutability, ON CONFLICT
// idempotency, etc.) under INTEGRATION=1.
package auth

import (
	"strings"
	"testing"
)

// TestUserStatus_ValidValues covers R-DB-006 / S-DB-050: every locked
// status value must parse via NewStatus without error. Each value is
// asserted individually so a regression that drops one enum case fails
// only the specific assertion, not the whole batch.
func TestUserStatus_ValidValues(t *testing.T) {
	cases := []struct {
		raw    string
		expect UserStatus
	}{
		{raw: "registered", expect: UserStatusRegistered},
		{raw: "active", expect: UserStatusActive},
		{raw: "inactive", expect: UserStatusInactive},
		{raw: "blocked", expect: UserStatusBlocked},
	}
	for _, c := range cases {
		t.Run(c.raw, func(t *testing.T) {
			got, err := NewUserStatus(c.raw)
			if err != nil {
				t.Fatalf("NewUserStatus(%q): unexpected error %v", c.raw, err)
			}
			if got != c.expect {
				t.Errorf("NewUserStatus(%q) = %q, want %q", c.raw, got, c.expect)
			}
		})
	}
}

// TestUserStatus_InvalidValue covers S-DB-050: any string that is not
// one of the four locked values must return a non-nil error. This is
// the gate that the bootstrap service relies on so a typo in a future
// admin tool cannot push `users.status` to a value the Go layer cannot
// round-trip.
func TestUserStatus_InvalidValue(t *testing.T) {
	cases := []string{"", "garbage", "REGISTERED", "registered ", " active"}
	for _, raw := range cases {
		t.Run("raw="+raw, func(t *testing.T) {
			got, err := NewUserStatus(raw)
			if err == nil {
				t.Fatalf("NewUserStatus(%q): expected error, got nil (status=%q)", raw, got)
			}
			if got != "" {
				t.Errorf("NewUserStatus(%q): expected zero-value status, got %q", raw, got)
			}
			// Error message must name the bad input so the admin tool
			// can surface it back to the user.
			if !strings.Contains(err.Error(), raw) && raw != "" {
				t.Errorf("NewUserStatus(%q): error %q does not name the bad input", raw, err.Error())
			}
		})
	}
}

// TestUserStatus_String covers the String() method on UserStatus so the
// JSON marshaller (and the login_audits payload) round-trips each
// locked value back to its canonical string.
func TestUserStatus_String(t *testing.T) {
	cases := []struct {
		s    UserStatus
		want string
	}{
		{UserStatusRegistered, "registered"},
		{UserStatusActive, "active"},
		{UserStatusInactive, "inactive"},
		{UserStatusBlocked, "blocked"},
	}
	for _, c := range cases {
		t.Run(c.want, func(t *testing.T) {
			if got := c.s.String(); got != c.want {
				t.Errorf("UserStatus(%q).String() = %q, want %q", c.s, got, c.want)
			}
		})
	}
}

// TestUser_StatusRegistered_IsNotActive covers the state-machine
// invariant that drives the registered→active transition in the
// bootstrap service. A freshly-registered user MUST NOT report
// IsActive()==true; only an explicitly-promoted user does. This is
// the property R-BOOTSTRAP-1 / R-BE-002 rely on.
func TestUser_StatusRegistered_IsNotActive(t *testing.T) {
	u := &User{Status: UserStatusRegistered}
	if u.IsActive() {
		t.Fatalf("User{Status: registered}.IsActive() = true, want false")
	}
}

// TestUser_StatusActive_IsActive covers the positive case: an
// active user reports IsActive()==true. The bootstrap service uses
// this to decide whether to issue a session cookie.
func TestUser_StatusActive_IsActive(t *testing.T) {
	u := &User{Status: UserStatusActive}
	if !u.IsActive() {
		t.Fatalf("User{Status: active}.IsActive() = false, want true")
	}
}

// TestUser_OtherStatuses_AreNotActive covers the negative cases for
// the remaining locked status values. inactive and blocked must both
// report IsActive()==false so the bootstrap handler treats them as
// terminal (no session cookie issued).
func TestUser_OtherStatuses_AreNotActive(t *testing.T) {
	for _, s := range []UserStatus{UserStatusInactive, UserStatusBlocked} {
		t.Run(string(s), func(t *testing.T) {
			u := &User{Status: s}
			if u.IsActive() {
				t.Fatalf("User{Status: %q}.IsActive() = true, want false", s)
			}
		})
	}
}

// TestUser_NewUser_DefaultsToRegistered covers the constructor
// invariant: a User built without an explicit status must default to
// `registered` (matching the DB column default). This protects against
// a future refactor that drops the default in the constructor.
func TestUser_NewUser_DefaultsToRegistered(t *testing.T) {
	u := NewUser("test@example.com", "google-sub-123")
	if u == nil {
		t.Fatal("NewUser: returned nil")
	}
	if u.Status != UserStatusRegistered {
		t.Errorf("NewUser().Status = %q, want %q", u.Status, UserStatusRegistered)
	}
	if u.Email != "test@example.com" {
		t.Errorf("NewUser().Email = %q, want %q", u.Email, "test@example.com")
	}
	if u.GoogleSub != "google-sub-123" {
		t.Errorf("NewUser().GoogleSub = %q, want %q", u.GoogleSub, "google-sub-123")
	}
}

// TestUser_Validate covers the construction-time validation
// invariant: empty email or empty google_sub must be rejected. The
// bootstrap service relies on this so a malformed request cannot
// create a user row that the resolver cannot later look up.
func TestUser_Validate(t *testing.T) {
	t.Run("empty email", func(t *testing.T) {
		u := NewUser("", "google-sub-1")
		if err := u.Validate(); err == nil {
			t.Fatal("User{Email: \"\"}.Validate(): expected error, got nil")
		}
	})
	t.Run("empty google_sub", func(t *testing.T) {
		u := NewUser("test@example.com", "")
		if err := u.Validate(); err == nil {
			t.Fatal("User{GoogleSub: \"\"}.Validate(): expected error, got nil")
		}
	})
	t.Run("valid user", func(t *testing.T) {
		u := NewUser("test@example.com", "google-sub-1")
		if err := u.Validate(); err != nil {
			t.Fatalf("User{Email, GoogleSub populated}.Validate(): unexpected error %v", err)
		}
	})
}