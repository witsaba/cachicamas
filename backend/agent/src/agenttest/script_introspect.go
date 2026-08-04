// AI-23.2 extension — R-CNF-024: read-only, vendor-neutral introspection of
// a Script's ordered steps from OUTSIDE the agenttest package. Script.Steps
// is already an exported field; only per-Step reads were missing, and Go
// permits declaring methods on Script/Step from any file in this package —
// so fake_script.go (and every other fake_*.go/stream_kit_*.go file) stays
// byte-identical (NFR-CNF-D).
//
// Both methods return value copies: ai.Event is itself an opaque immutable
// value (unexported payload, no setters of its own), so no pointer, slice
// or backing array of internal state escapes through either method. There
// is deliberately no Gate() accessor — it would expose a mutation door
// (Release) that reconstructing a script's ordered steps does not need —
// and no setter, no rewriting constructor (S-CLA-035). Vendor-neutral:
// imports only ai, names no adapter, encodes no wire format.

package agenttest

import "github.com/cachicamas/backend/agent/src/ai"

// IsHold reports whether s is a Hold step — one that blocks its producer at
// a Gate — rather than an Emit step.
func (s Step) IsHold() bool { return s.isHold }

// Event returns the event s emits, and true — or the zero ai.Event and
// false when s is a Hold step, or the zero Step, and carries no event.
func (s Step) Event() (ai.Event, bool) {
	if s.isHold {
		return ai.Event{}, false
	}
	return s.event, s.event.Kind() != 0
}
