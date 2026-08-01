// Tests for the AI-14 descriptor skeleton — groundwork for R-AEE-014.
//
// The package under test is imported by its module path from the external
// test package ai_test, matching every other Layer 1 test file's convention.
package ai_test

import (
	"testing"

	"github.com/cachicamas/backend/agent/src/ai"
)

// Groundwork for R-AEE-014 — the descriptor's zero value is "no block role,
// no cardinality restriction, not terminal", the least restrictive reading,
// so a kind registered without stating one does not accidentally join block
// ordering, get capped at one, or end the stream.
func TestEventDescriptor_ZeroValue_RoundTrips(t *testing.T) {
	t.Parallel()

	t.Run("BlockRoleNone and CardinalityAny are the zero values", func(t *testing.T) {
		t.Parallel()

		var role ai.BlockRole
		if role != ai.BlockRoleNone {
			t.Errorf("the zero BlockRole = %v, want BlockRoleNone", role)
		}

		var cardinality ai.Cardinality
		if cardinality != ai.CardinalityAny {
			t.Errorf("the zero Cardinality = %v, want CardinalityAny", cardinality)
		}
	})

	t.Run("the zero EventDescriptor round-trips", func(t *testing.T) {
		t.Parallel()

		var d ai.EventDescriptor
		if d.Role != ai.BlockRoleNone {
			t.Errorf("the zero EventDescriptor.Role = %v, want BlockRoleNone", d.Role)
		}
		if d.Cardinality != ai.CardinalityAny {
			t.Errorf("the zero EventDescriptor.Cardinality = %v, want CardinalityAny", d.Cardinality)
		}
		if d.Terminal {
			t.Error("the zero EventDescriptor.Terminal = true, want false")
		}

		// An explicit literal built from the zero-valued components must equal
		// the zero value: the three fields are the whole of the type's
		// comparable state, so nothing hides an unstated fourth thing.
		explicit := ai.EventDescriptor{Role: ai.BlockRoleNone, Cardinality: ai.CardinalityAny, Terminal: false}
		if explicit != d {
			t.Errorf("an explicit zero-valued EventDescriptor %+v != the zero value %+v", explicit, d)
		}
	})
}
