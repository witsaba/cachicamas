// AI-21.1 — the script vocabulary: what a test tells the fake to do on one
// streaming call, expressed as an ordered list of opaque steps.
//
// Emit is the door onto a Step that sends an event; Hold (AI-21.4) is the
// door onto a Step that blocks the producer at a Gate instead. Together
// they are the whole vocabulary — a Step cannot express a third shape.
package agenttest

import "github.com/cachicamas/backend/agent/src/ai"

// Step is one instruction in a Script: emit one event today; AI-21.4 adds a
// second shape once Gate exists to hold on. It is opaque so a script cannot
// express anything Emit (or, later, Hold) does not build.
type Step struct {
	event ai.Event
}

// Emit returns a Step that, when the producer reaches it, sends ev on the
// stream, in order.
//
// ev MUST already be a validly constructed ai.Event — one of the ai
// package's own constructors is the only door to that. Emit panics
// immediately, naming the defect, when given an event that was never
// constructed (R-AFP-002): a script carrying one is a fixture-authoring
// mistake, decidable the moment the script is written, not a condition a
// caller who reached the stream needs to handle.
func Emit(ev ai.Event) Step {
	if ev.Kind() == 0 {
		panic("agenttest: Emit given an ai.Event that was never constructed (zero Kind) — build it through one of the ai package's own constructors first")
	}
	return Step{event: ev}
}

// Script describes one streaming call's scripted behavior: the steps to run,
// in order, and the returned channel's exact buffer capacity.
type Script struct {
	// Steps is the ordered list of instructions the producer runs.
	Steps []Step

	// Buffer sizes the returned channel's capacity exactly (R-AFP-011): 0
	// means unbuffered.
	Buffer int
}
