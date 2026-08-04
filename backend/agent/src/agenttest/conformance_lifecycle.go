// AI-23.2 amendment — the mandated response lifecycle (R-CNF-019…R-CNF-021):
// every started-response conformance case must open its drained window with
// the lifecycle prefix carrying the case's own scripted identity, and every
// exact count this amendment introduces is derived from its own amended
// script, never obtained by incrementing a shipped count.
//
// checkLifecyclePrefix is a pure, directly-callable check — no testing.TB
// involved — precisely so a synthetic, start-less (or identity-mismatched)
// event slice can be handed to it and provably rejected: the permanent
// negative proof R-CNF-020 requires, mirroring
// conformance_terminal.go's own requireFailureCategoryCoverage pure-check
// idiom. requireDrainedKinds is the shared exact-count-plus-positional-kind
// assertion every amended case in this package's derivation table (design.md
// R-CNF-019) uses instead of a hand-rolled loop.
//
// New file only (NFR-CNF-D): fake_*.go and stream_kit_*.go are never
// touched by this amendment.

package agenttest

import (
	"fmt"
	"testing"

	"github.com/cachicamas/backend/agent/src/ai"
)

// checkLifecyclePrefix reports the first reason events does not open with
// the lifecycle prefix carrying the scripted identity, or nil when it does.
// The rules are checked in order: an empty slice, or one whose first event
// is not a response start, fails naming the absent lifecycle event
// (S-CLA-015); a present response start whose ResponseID() or
// ServedModel() differs from the wanted values fails naming the mismatched
// field — proving equality, not mere presence, is what is asserted
// (S-CLA-016).
func checkLifecyclePrefix(events []ai.Event, wantResponseID, wantServedModel string) error {
	if len(events) == 0 || events[0].Kind() != ai.EventKindResponseStart {
		return fmt.Errorf("agenttest: drained window does not open with %v, want the lifecycle prefix as event 0 (R-CNF-020)", ai.EventKindResponseStart)
	}
	start, ok := events[0].ResponseStart()
	if !ok {
		return fmt.Errorf("agenttest: event 0 carries kind %v but no ResponseStart payload, want the lifecycle prefix (R-CNF-020)", events[0].Kind())
	}
	if start.ResponseID() != wantResponseID {
		return fmt.Errorf("agenttest: lifecycle prefix ResponseID() = %q, want %q (R-CNF-020, identity equality)", start.ResponseID(), wantResponseID)
	}
	if start.ServedModel() != wantServedModel {
		return fmt.Errorf("agenttest: lifecycle prefix ServedModel() = %q, want %q (R-CNF-020, identity equality)", start.ServedModel(), wantServedModel)
	}
	return nil
}

// requireDrainedKinds fails tb, naming the discrepancy, unless events has
// exactly len(want) elements whose kinds match want positionally — the one
// derivation shape every amended case's exact drained window uses
// (R-CNF-019), shared rather than reimplemented per case.
func requireDrainedKinds(tb testing.TB, events []ai.Event, want []ai.EventKind) {
	tb.Helper()
	if len(events) != len(want) {
		tb.Fatalf("agenttest: drained %d event(s), want %d %v (R-CNF-019)", len(events), len(want), want)
	}
	for i, ev := range events {
		if ev.Kind() != want[i] {
			tb.Errorf("agenttest: event %d kind = %v, want %v (R-CNF-019)", i, ev.Kind(), want[i])
		}
	}
}
