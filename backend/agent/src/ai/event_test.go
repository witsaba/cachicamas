// Tests for AI-14.1 … AI-14.4 — the event envelope, its per-stream sequence
// and its ordering invariants.
//
// The package under test is imported by its module path from the external
// test package ai_test, so every assertion is written against exactly the
// surface a Layer 2 consumer sees — the same convention content_part_test.go
// and every other Layer 1 test file uses.
package ai_test

import (
	"errors"
	"go/ast"
	"go/parser"
	"go/token"
	"reflect"
	"strings"
	"testing"

	"github.com/cachicamas/backend/agent/src/ai"
)

// R-AEE-001 — the kind is derived from the payload and is never stored.
//
// AI-14 ships zero production kinds (R-AEE-006), so the only Event this test
// can construct without the test-only witness (export_test.go, landed later
// in this same milestone) is the zero value. That is still a real instance of
// "derived, not stored": the zero value carries no payload, so it reports the
// zero kind, and nothing about the type lets a caller override that
// independently of the payload. The richer case — a non-zero kind read back
// correctly — is proven again once the witness lands (R-AEE-005).
func TestEvent_Kind_IsDerivedFromPayloadAndNeverStored(t *testing.T) {
	t.Parallel()

	t.Run("a part that carries no payload reports the zero kind, and re-reading agrees", func(t *testing.T) {
		t.Parallel()

		var unconstructed ai.Event
		first := unconstructed.Kind()
		if first != 0 {
			t.Errorf("unconstructed.Kind() = %v, want the zero kind", first)
		}
		if second := unconstructed.Kind(); second != first {
			t.Errorf("re-reading unconstructed.Kind() = %v, want %v (the same value again)", second, first)
		}
	})

	t.Run("Event declares no kind field: every field is unexported", func(t *testing.T) {
		t.Parallel()

		typ := reflect.TypeOf(ai.Event{})
		if typ.NumField() == 0 {
			t.Fatal("ai.Event has no fields at all; the test below would pass vacuously")
		}
		for i := 0; i < typ.NumField(); i++ {
			field := typ.Field(i)
			if field.IsExported() {
				t.Errorf("ai.Event field %q is exported; a settable field could disagree with the payload it names", field.Name)
			}
		}
	})
}

// R-AEE-002 — a payload-less event is rejected with ErrNotInVocabulary.
//
// S-AEE-005 (a payload whose type is not registered) and S-AEE-006 (the AI-04
// sentinel set is unchanged) are deferred to later steps of this same
// milestone: S-AEE-005 needs the test-only witness kind (export_test.go,
// R-AEE-004) to construct a payload the registry does not know, and S-AEE-006
// is the exact claim NFR-AEE-C's mechanical guard makes for the whole
// capability (Phase 6). Both are recorded against this requirement in
// tasks.md at the step they are actually proven.
func TestCheckEmit_PayloadlessEvent_RejectedWithErrNotInVocabulary(t *testing.T) {
	t.Parallel()

	var unconstructed ai.Event

	err := ai.CheckEmit(unconstructed)
	if err == nil {
		t.Fatal("ai.CheckEmit(unconstructed) = nil, want a rejection — a value that skipped construction has no payload, therefore no kind")
	}
	if !errors.Is(err, ai.ErrNotInVocabulary) {
		t.Errorf("ai.CheckEmit(unconstructed) = %v, want errors.Is to match ai.ErrNotInVocabulary", err)
	}
	if got := err.Error(); !strings.Contains(got, "event") {
		t.Errorf("ai.CheckEmit(unconstructed).Error() = %q, want it to name the offending position (\"event\")", got)
	}
}

// R-AEE-003 — the payload contract is sealed against outside implementation.
//
// S-AEE-007 (a type declared in ai_test that attempts to satisfy eventPayload
// fails to compile) is a structural fact proven by construction: an
// unexported interface with unexported methods has no legal implementation
// syntax from another package — content_part.go's partPayload established
// this sealing idiom, and the AST scan below verifies it mechanically rather
// than by review, exactly as content_part_registry_test.go does for its own
// sealed interface. S-AEE-009 (every payload this test package can observe is
// either absent or one this package built) follows from the same fact:
// ai_test has no syntax to construct any eventPayload value of its own.
func TestEventPayload_Contract_IsSealed(t *testing.T) {
	t.Parallel()

	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "event.go", nil, 0)
	if err != nil {
		t.Fatalf("parsing event.go: %v", err)
	}

	iface := findInterfaceType(file, "eventPayload")
	if iface == nil {
		t.Fatal(`event.go declares no "eventPayload" interface type; the guard would pass vacuously`)
	}
	if ast.IsExported("eventPayload") {
		t.Fatal("eventPayload must be unexported so no outside package can name it")
	}
	if len(iface.Methods.List) == 0 {
		t.Fatal("eventPayload declares no methods; the guard would pass vacuously")
	}
	for _, method := range iface.Methods.List {
		for _, name := range method.Names {
			if ast.IsExported(name.Name) {
				t.Errorf("eventPayload method %q is exported; an exported method could be satisfied by a type outside this package, unsealing the contract", name.Name)
			}
		}
	}
}

// R-AEE-005 — an event is readable from an external package without a type
// switch over unexported types.
func TestEvent_ReadableExternally_NoTypeSwitchOverUnexportedTypes(t *testing.T) {
	t.Parallel()

	t.Run("kind and sequence both compile and report the producer's values", func(t *testing.T) {
		t.Parallel()

		event := ai.NewWitnessEvent(1)
		if got := event.Kind(); got != ai.KindTestWitness {
			t.Errorf("event.Kind() = %v, want %v", got, ai.KindTestWitness)
		}
		// The witness constructor does not stamp a sequence — that is the
		// per-stream stamper's job (AI-14.2) — so an unstamped witness event
		// reports the documented sentinel here, which is itself a value a
		// consumer reads with no special casing.
		if got := event.Sequence(); got != 0 {
			t.Errorf("event.Sequence() = %v, want the unstamped sentinel 0", got)
		}
	})

	t.Run("the typed payload is obtained through an exported accessor, no unexported-type assertion", func(t *testing.T) {
		t.Parallel()

		event := ai.NewWitnessEvent(3)
		payload, ok := event.WitnessPayload()
		if !ok {
			t.Fatal("event.WitnessPayload() reported no payload on an event of its own kind")
		}
		if got := payload.BlockIndex(); got != 3 {
			t.Errorf("payload.BlockIndex() = %v, want 3 (the value the constructor was given)", got)
		}
	})

	t.Run("an accessor called for the wrong kind reports failure, not a panic and not a partial value", func(t *testing.T) {
		t.Parallel()

		var wrongKind ai.Event // the zero Event carries no WitnessPayload

		payload, ok := wrongKind.WitnessPayload()
		if ok {
			t.Error("wrongKind.WitnessPayload() reported success on an event that carries no witness payload")
		}
		if payload != (ai.WitnessPayload{}) {
			t.Errorf("wrongKind.WitnessPayload() = %+v on failure, want the zero WitnessPayload", payload)
		}
	})
}

// findInterfaceType returns the *ast.InterfaceType declared under the given
// name in file, or nil if no such type declaration exists. Shared by every
// sealing/registry guard in this test package that needs to read a package
// interface's shape from source rather than trust a claim about it.
func findInterfaceType(file *ast.File, name string) *ast.InterfaceType {
	for _, decl := range file.Decls {
		gen, ok := decl.(*ast.GenDecl)
		if !ok || gen.Tok != token.TYPE {
			continue
		}
		for _, spec := range gen.Specs {
			typed, ok := spec.(*ast.TypeSpec)
			if !ok || typed.Name.Name != name {
				continue
			}
			iface, ok := typed.Type.(*ast.InterfaceType)
			if !ok {
				return nil
			}
			return iface
		}
	}
	return nil
}
