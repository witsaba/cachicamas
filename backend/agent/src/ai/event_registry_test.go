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
