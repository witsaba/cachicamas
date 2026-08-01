// Tests for R-AEE-004, R-AEE-006 and R-AEE-014 — the event-kind registry is
// exhaustive, ships zero production members, and cannot register a kind
// without a descriptor.
//
// AI-14 registers zero production event kinds (R-AEE-006), so unlike
// content_part_registry_test.go's AI-06.4 guard — which scans production
// source for four declared PartKind constants — this guard's subject is the
// test-only witness kind bridged through export_test.go (D6) plus whatever a
// test dynamically registers through RegisterTestKind. The mechanism is the
// same shape AI-06.4 established: a witness table, cross-checked against the
// kind vocabulary the package actually reports, so a kind reachable by one
// path and not the other is caught rather than assumed away.
package ai_test

import (
	"go/ast"
	"go/parser"
	"go/token"
	"testing"

	"github.com/cachicamas/backend/agent/src/ai"
)

// eventKindWitness is what a registered event kind must supply to prove its
// legs, mirroring content_part_registry_test.go's partKindWitness (AI-06.4).
type eventKindWitness struct {
	// registeredName is this kind's registered name (event.go's eventRegistry
	// entry), stated here rather than read from the registry so the registry
	// is checked rather than trusted.
	registeredName string

	// construct is leg 1: a constructor reachable from another package.
	construct func(ai.BlockIndex) ai.Event

	// read is leg 2: the exported accessor for this kind's payload.
	read func(ai.Event) (ai.WitnessPayload, bool)
}

// eventKindWitnesses is the guard's table. It carries exactly one entry at
// this milestone — AI-14 registers zero production kinds (R-AEE-006) — for
// the test-only witness kind bridged through export_test.go. Leg 3 (a
// validation path) is exercised once the per-stream stamper exists to give a
// witness event a real sequence: see sequence_test.go's R-AEE-010 coverage
// and the leg-3 addition recorded there in tasks.md.
var eventKindWitnesses = map[ai.EventKind]eventKindWitness{
	ai.KindTestWitness: {
		registeredName: "test_witness",
		construct:      ai.NewWitnessEvent,
		read:           func(e ai.Event) (ai.WitnessPayload, bool) { return e.WitnessPayload() },
	},
}

// TestEventKindRegistration_TheTestKindVocabulary_HasConstructorAndAccessor is
// the exhaustiveness guard (R-AEE-004). It fails when a kind ai.AllTestEventKinds
// carries has no witness, when a witness kind is not carried by
// ai.AllTestEventKinds, or when a witness leg does not do what it promises.
func TestEventKindRegistration_TheTestKindVocabulary_HasConstructorAndAccessor(t *testing.T) {
	t.Parallel()

	kinds := ai.AllTestEventKinds()
	if len(kinds) == 0 {
		t.Fatal("ai.AllTestEventKinds() is empty; the exhaustiveness assertion would pass vacuously")
	}

	t.Run("every test event kind has a witness, and every witness a kind", func(t *testing.T) {
		t.Parallel()

		for _, k := range kinds {
			if _, ok := eventKindWitnesses[k]; !ok {
				t.Errorf("%v is carried by ai.AllTestEventKinds() but has no entry in eventKindWitnesses", k)
			}
		}
		for k := range eventKindWitnesses {
			if !containsEventKind(kinds, k) {
				t.Errorf("eventKindWitnesses carries a witness for %v, which ai.AllTestEventKinds() does not enumerate", k)
			}
		}
	})

	for k, witness := range eventKindWitnesses {
		t.Run(witness.registeredName, func(t *testing.T) {
			t.Parallel()

			t.Run("leg 1 — a constructor", func(t *testing.T) {
				t.Parallel()

				event := witness.construct(1)
				if event.Kind() != k {
					t.Fatalf("the witness for %v built an event of kind %v, want %v", witness.registeredName, event.Kind(), k)
				}
			})

			t.Run("leg 2 — a payload accessor", func(t *testing.T) {
				t.Parallel()

				event := witness.construct(1)
				payload, ok := witness.read(event)
				if !ok {
					t.Errorf("the accessor for %v reported no payload on an event of its own kind", witness.registeredName)
				}

				// The accessor must also answer no for the zero Event: the
				// second result distinguishes "not this kind" from "this
				// kind, empty" (content_part_registry_test.go's leg 2
				// pattern, restated for the event envelope).
				if _, ok := witness.read(ai.Event{}); ok {
					t.Errorf("the accessor for %v reported a payload on the zero Event", witness.registeredName)
				}
				_ = payload
			})
		})
	}
}

// R-AEE-006 — AI-14 registers zero production event kinds.
func TestEventKinds_ProductionVocabulary_IsEmpty(t *testing.T) {
	t.Parallel()

	if got := ai.EventKinds(); len(got) != 0 {
		t.Errorf("ai.EventKinds() = %v, want empty — AI-14 ships zero production kinds", got)
	}
}

func containsEventKind(haystack []ai.EventKind, want ai.EventKind) bool {
	for _, k := range haystack {
		if k == want {
			return true
		}
	}
	return false
}

// R-AEE-014, S-AEE-043 — every registered kind has a descriptor with values
// from the stated domains: Role is one of the four declared BlockRole
// constants, Cardinality is one of the two declared Cardinality constants.
func TestEventRegistration_EveryRegisteredKind_HasADescriptorFromStatedDomains(t *testing.T) {
	t.Parallel()

	validRoles := map[ai.BlockRole]bool{
		ai.BlockRoleNone: true, ai.BlockRoleStart: true, ai.BlockRoleDelta: true, ai.BlockRoleEnd: true,
	}
	validCardinalities := map[ai.Cardinality]bool{
		ai.CardinalityAny: true, ai.CardinalityAtMostOne: true,
	}

	kinds := ai.AllTestEventKinds()
	if len(kinds) == 0 {
		t.Fatal("ai.AllTestEventKinds() is empty; the assertion would pass vacuously")
	}
	for _, k := range kinds {
		d, ok := ai.DescriptorOf(k)
		if !ok {
			t.Errorf("ai.DescriptorOf(%v) reported not registered, but %v is carried by ai.AllTestEventKinds()", k, k)
			continue
		}
		if !validRoles[d.Role] {
			t.Errorf("%v's descriptor.Role = %v, want one of the four declared BlockRole constants", k, d.Role)
		}
		if !validCardinalities[d.Cardinality] {
			t.Errorf("%v's descriptor.Cardinality = %v, want one of the two declared Cardinality constants", k, d.Cardinality)
		}
	}
}

// R-AEE-014, S-AEE-044 — a kind structurally cannot register without a
// descriptor: eventRegistration's descriptor field is a concrete
// EventDescriptor value, never a pointer or an interface, so there is no Go
// literal that omits it and produces "no descriptor" rather than the zero
// EventDescriptor — exactly as event_descriptor.go's own file comment states.
// Proven by AST inspection of event.go's declaration, mirroring
// TestEventPayload_Contract_IsSealed's approach (event_test.go).
func TestEventRegistration_DescriptorField_IsAConcreteRequiredType(t *testing.T) {
	t.Parallel()

	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "event.go", nil, 0)
	if err != nil {
		t.Fatalf("parsing event.go: %v", err)
	}

	field := structFieldType(file, "eventRegistration", "descriptor")
	if field == nil {
		t.Fatal(`event.go's "eventRegistration" struct declares no "descriptor" field; the guard would pass vacuously`)
	}
	ident, ok := field.(*ast.Ident)
	if !ok || ident.Name != "EventDescriptor" {
		t.Errorf("eventRegistration.descriptor's type = %T, want the concrete named type EventDescriptor "+
			"(never a pointer or an interface, either of which could be nil/absent)", field)
	}
}

// R-AEE-014, S-AEE-045 — a start|delta|end descriptor's payload exposes a
// readable block index; a none descriptor does not require one.
//
// No t.Parallel() anywhere in this test, including its subtests: it calls
// ai.RegisterTestKind, which export_test.go's own file comment forbids
// running in parallel — eventRegistry is one shared package-level slice, and
// a concurrent truncation from another test would corrupt it.
func TestEventRegistration_BlockScopedDescriptor_ExposesAReadableBlockIndex(t *testing.T) {
	blockRoles := []struct {
		name string
		role ai.BlockRole
	}{
		{"BlockRoleStart", ai.BlockRoleStart},
		{"BlockRoleDelta", ai.BlockRoleDelta},
		{"BlockRoleEnd", ai.BlockRoleEnd},
	}
	for _, tc := range blockRoles {
		t.Run(tc.name, func(t *testing.T) {
			k, cleanup := ai.RegisterTestKind("block_scoped_"+tc.name, ai.EventDescriptor{Role: tc.role})
			t.Cleanup(cleanup)

			event := ai.NewTestEvent(k, 7)
			payload, ok := event.WitnessPayload()
			if !ok {
				t.Fatal("event.WitnessPayload() reported no payload on an event of its own kind")
			}
			if got := payload.BlockIndex(); got != 7 {
				t.Errorf("payload.BlockIndex() = %v, want 7 (the value the constructor was given)", got)
			}
		})
	}

	t.Run("BlockRoleNone does not require one", func(t *testing.T) {
		// KindTestWitness's own descriptor is BlockRoleNone; its events still
		// carry a readable block index (WitnessPayload always exposes one),
		// which is permitted — R-AEE-014 says one is not REQUIRED for a
		// BlockRoleNone kind, not that one is forbidden.
		d, ok := ai.DescriptorOf(ai.KindTestWitness)
		if !ok {
			t.Fatal("ai.DescriptorOf(ai.KindTestWitness) reported not registered")
		}
		if d.Role != ai.BlockRoleNone {
			t.Fatalf("ai.KindTestWitness's descriptor.Role = %v, want BlockRoleNone (test assumption)", d.Role)
		}
		event := ai.NewWitnessEvent(0)
		if _, ok := event.WitnessPayload(); !ok {
			t.Fatal("event.WitnessPayload() reported no payload on an event of its own kind")
		}
	})
}

// structFieldType returns the type expression of a named field within a
// named struct type declared in file, or nil if either does not exist.
func structFieldType(file *ast.File, structName, fieldName string) ast.Expr {
	for _, decl := range file.Decls {
		gen, ok := decl.(*ast.GenDecl)
		if !ok || gen.Tok != token.TYPE {
			continue
		}
		for _, spec := range gen.Specs {
			typed, ok := spec.(*ast.TypeSpec)
			if !ok || typed.Name.Name != structName {
				continue
			}
			st, ok := typed.Type.(*ast.StructType)
			if !ok {
				return nil
			}
			for _, f := range st.Fields.List {
				for _, name := range f.Names {
					if name.Name == fieldName {
						return f.Type
					}
				}
			}
		}
	}
	return nil
}
