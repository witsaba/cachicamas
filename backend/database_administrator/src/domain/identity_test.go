// Package domain_test contains the test suite for the identity
// domain entity. This file locks the public surface of
// domain/identity.go against spec R-BAM-010 / S-BAM-040 (locked field
// list + AppError implementation).
//
// Strict TDD discipline (per openspec/AGENTS.md and
// sdd-init/cachicamas): this file was written BEFORE
// identity.go existed. Running `go test ./src/domain/...` with no
// Identity type must fail with "undefined: domain.Identity" — that
// failure IS the RED step.
package domain_test

import (
	"errors"
	"reflect"
	"testing"

	"github.com/cachicamas/backend/database_administrator/src/domain"
)

// TestIdentity_StructFields locks the field list of domain.Identity
// per spec R-BAM-010. Any future renames or removals will fail this
// test, forcing the change to be intentional.
func TestIdentity_StructFields(t *testing.T) {
	want := []string{"ID", "Email", "Name", "ImageURL", "Provider", "ProviderAccountID"}
	got := identityFieldNames(t, domain.Identity{})
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("Identity field list drift:\n got  = %v\n want = %v", got, want)
	}
}

// TestIdentityNotFoundError_AppError asserts that the "identity not
// found" error type satisfies the domain.AppError interface so the
// HTTP handler can map it to a 404 envelope with code="not_found"
// without importing pgx or any infrastructure package.
func TestIdentityNotFoundError_AppError(t *testing.T) {
	var _ domain.AppError = (*domain.IdentityNotFoundError)(nil)

	e := &domain.IdentityNotFoundError{Email: "ghost@example.com"}
	if got, want := e.Error(), `identity not found: "ghost@example.com"`; got != want {
		t.Errorf("Error():\n got  = %q\n want = %q", got, want)
	}
	if got, want := e.Code(), domain.CodeNotFound; got != want {
		t.Errorf("Code():\n got  = %q\n want = %q", got, want)
	}
}

// TestIdentityRepository_PortShape locks the IdentityRepository
// interface signature. The signature MUST stay exactly as the spec
// dictates so the application/infrastructure layers cannot drift.
func TestIdentityRepository_PortShape(_ *testing.T) {
	// Compile-time assertion: any concrete adapter that does not
	// expose LookupByEmail with this shape will fail to build.
	// (Interface type assertions on nil are cheap; the value here is
	// the documentation.)
	var _ domain.IdentityRepository = nil
}

// TestIdentityRepository_LookupByEmail_IsContextFirst locks the
// argument order (ctx first) per Go conventions. The handler depends
// on this so it can wire a request-bound context with a deadline.
func TestIdentityRepository_LookupByEmail_IsContextFirst(t *testing.T) {
	// Build a closure type whose first parameter is context-shaped
	// (we use the parameter count + the parameter types via reflect).
	// If a future refactor moves ctx to the second position, the
	// reflect comparison below flags it.
	var iface domain.IdentityRepository
	method := reflect.ValueOf(&iface).Elem().Type().Method(0)
	if got, want := method.Name, "LookupByEmail"; got != want {
		t.Errorf("method name: got %q want %q", got, want)
	}
	// Two inputs: ctx, email. Outputs: (*Identity, error).
	if got := method.Type.NumIn(); got != 2 {
		t.Fatalf("method inputs: got %d want 2", got)
	}
	// First input name check via reflection on Fn signature.
	in0 := method.Type.In(0)
	in1 := method.Type.In(1)
	if in0.Kind() != reflect.Interface || in0.String() != "context.Context" {
		t.Errorf("first input must be context.Context; got %s", in0)
	}
	if in1.Kind() != reflect.String {
		t.Errorf("second input must be string (email); got %s", in1)
	}
	if got := method.Type.NumOut(); got != 2 {
		t.Errorf("method outputs: got %d want 2 (*Identity, error)", got)
	}
}

// identityFieldNames extracts the exported field names of T in
// declaration order. It is order-sensitive because the locked field
// list spec R-BAM-010 is order-sensitive.
func identityFieldNames[T any](t *testing.T, _ T) []string {
	t.Helper()
	var zero T
	v := reflect.TypeOf(zero)
	out := make([]string, 0, v.NumField())
	for i := 0; i < v.NumField(); i++ {
		out = append(out, v.Field(i).Name)
	}
	return out
}

// Compile-time guard: the sentinel/typed error that the
// infrastructure layer MUST return on a not-found row must be
// reachable via errors.As. (This is documentation as much as test —
// the lookup code uses errors.As to convert the repo error to the
// domain-level IdentityNotFoundError.)
var _ error = (*domain.IdentityNotFoundError)(nil)
var _ = errors.As
