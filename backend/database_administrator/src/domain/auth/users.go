// Package auth contains the domain types for the google-auth-bootstrap slice.
//
// users.go defines the User value object + the UserRepository port used
// by the bootstrap service. The UserStatus enum is the canonical
// implementation of spec R-DB-006; the DB column accepts any string and
// trusts the Go layer to validate.
//
// CRITICAL invariants (locked at design §2.2 + spec §4 / §5):
//   - User.Status is validated via NewUserStatus so the bootstrap
//     service cannot persist a value outside the four locked states.
//     (S-DB-050)
//   - User.CreatedAt is set ONLY at construction time. The repository
//     UPDATE statements NEVER include the created_at column; this is
//     the application-layer half of S-DB-002. A DB trigger
//     (migration 2026XXXXXX_user_immutable_created_at.sql) provides
//     the second half at the database level.
//   - Email is the main identifier (lowercase-normalized); google_sub
//     is the provider-specific secondary key. Lookups MUST key by
//     google_sub first, per design D4 / spec R-BE-002.
package auth

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"
)

// UserStatus is the locked state-machine vocabulary for auth.users.status
// (R-DB-006 / S-DB-050). The DB column accepts any string and trusts
// the Go layer; future phases may add a CHECK constraint, but for now
// the Go validation is the gate.
type UserStatus string

// Locked UserStatus values. Adding a new value here is a spec change
// (R-DB-006 / design §0 AD-1); removing one breaks persisted data.
const (
	UserStatusRegistered UserStatus = "registered"
	UserStatusActive     UserStatus = "active"
	UserStatusInactive   UserStatus = "inactive"
	UserStatusBlocked    UserStatus = "blocked"
)

// knownUserStatuses is the closed set of accepted values. Kept private
// so callers MUST go through NewUserStatus — the closure is the
// spec-mandated guarantee that an unknown value cannot reach the DB.
var knownUserStatuses = map[UserStatus]struct{}{
	UserStatusRegistered: {},
	UserStatusActive:     {},
	UserStatusInactive:   {},
	UserStatusBlocked:    {},
}

// NewUserStatus parses a raw string (typically a DB column value or a
// JSON payload field) and returns a UserStatus. Unknown values yield
// an error so the bootstrap handler can map to HTTP 500 (corrupt DB)
// or HTTP 400 (bad request payload).
func NewUserStatus(raw string) (UserStatus, error) {
	s := UserStatus(raw)
	if _, ok := knownUserStatuses[s]; !ok {
		return "", fmt.Errorf("auth.NewUserStatus: %q is not a known user status (want one of registered|active|inactive|blocked)", raw)
	}
	return s, nil
}

// String satisfies fmt.Stringer so JSON marshalling + slog output use
// the canonical lowercase form. Returning the underlying string keeps
// the value JSON-stable across releases.
func (s UserStatus) String() string {
	return string(s)
}

// User is the canonical domain value for an auth.users row. The fields
// mirror the SQL schema; the JSON tags let the bootstrap handler
// serialise the lookup response without a separate DTO.
type User struct {
	ID            int64      `json:"id"`
	Email         string     `json:"email"`
	EmailVerified bool       `json:"email_verified"`
	GoogleSub     string     `json:"google_sub"`
	Name          string     `json:"name"`
	PictureURL    string     `json:"picture_url"`
	Status        UserStatus `json:"status"`
	LastLoginAt   *time.Time `json:"last_login_at,omitempty"`
	CreatedAt     time.Time  `json:"created_at"`
	UpdatedAt     time.Time  `json:"updated_at"`
}

// NewUser is the canonical constructor for an in-memory User. It
// applies the application-layer defaults: status=registered, email
// lowercased (per spec R-BE-002), no name/picture until the bootstrap
// UPDATE step fills them in.
//
// The CreatedAt timestamp is left at the zero value so the repository
// INSERT path can use the column's DEFAULT now(). The bootstrap
// service reads the DB-generated value back via RETURNING.
func NewUser(email, googleSub string) *User {
	return &User{
		Email:     strings.ToLower(email),
		GoogleSub: googleSub,
		Status:    UserStatusRegistered,
	}
}

// Validate enforces the construction-time invariants the bootstrap
// handler relies on: a User without an email or google_sub cannot be
// persisted (the DB NOT NULL constraints would reject it, but failing
// here yields a cleaner 400 response than a 500 from a pgx error).
func (u *User) Validate() error {
	if u == nil {
		return errors.New("auth.User.Validate: nil receiver")
	}
	if u.Email == "" {
		return errors.New("auth.User.Validate: email is required")
	}
	if u.GoogleSub == "" {
		return errors.New("auth.User.Validate: google_sub is required")
	}
	return nil
}

// IsActive reports whether the user is in the `active` state — the
// only state the bootstrap handler treats as "session cookie
// authorised". inactive and blocked are terminal (no session cookie);
// registered is the pre-organization-creation state that the
// bootstrap service promotes to active in the same transaction.
func (u *User) IsActive() bool {
	return u != nil && u.Status == UserStatusActive
}

// Querier is the narrow interface satisfied by both *sql.DB and
// *sql.Tx. The bootstrap service uses Querier on every repository
// call so a single transaction (R-BE-002 / S-BE-020) can compose
// user + organization + login_audit writes atomically. Methods that
// require a transaction (InsertRegistered → RETURNING, ExecContext)
// are split per-method on the concrete repository struct.
type Querier interface {
	QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row
	ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error)
}

// UserRepository is the application-layer port the bootstrap service
// depends on. The Postgres adapter in
// infrastructure/postgres/user_repo.go is the only known
// implementation in this slice; tests may use a fake.
//
// Each method accepts a Querier (interface satisfied by both
// *sql.DB and *sql.Tx) so the bootstrap service can run them inside a
// single transaction (R-BE-002 atomicity).
type UserRepository interface {
	// FindByGoogleSub returns the live (deleted_at IS NULL) user with
	// the given google_sub, or (nil, nil) when no such row exists.
	// Returns an error only for DB-level failures.
	FindByGoogleSub(ctx context.Context, q Querier, googleSub string) (*User, error)

	// FindByID returns the live user with the given id, or (nil, nil)
	// when no such row exists.
	FindByID(ctx context.Context, q Querier, id int64) (*User, error)

	// InsertRegistered persists a freshly-built User with the
	// `registered` status. Returns the persisted row (including the
	// DB-generated id + timestamps). Must be idempotent on
	// google_sub: a duplicate insert returns the existing row, not
	// an error (spec R-BOOTSTRAP-1).
	InsertRegistered(ctx context.Context, q Querier, u *User) (*User, error)

	// UpdateLoginFields advances the mutable login fields
	// (name / picture_url / email_verified / last_login_at /
	// updated_at). MUST NOT touch created_at (S-DB-002 — enforced at
	// the repo layer by the explicit column list below; the DB
	// trigger prevents accidental writes from any other path).
	UpdateLoginFields(ctx context.Context, q Querier, id int64, u *User) error

	// PromoteToActive transitions a user from `registered` to
	// `active`. The WHERE clause carries `status = 'registered'` so
	// a re-call on an already-active user is a no-op (idempotent),
	// matching spec R-BE-002.
	PromoteToActive(ctx context.Context, q Querier, id int64) error
}