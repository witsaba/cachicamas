// Package apptest — drain.go: the one assert helper (AG-23, design
// AD-3). Delegates validation wholesale to the already-exported stream
// validator; it never re-implements any part of it (R-L3H-003).
package apptest

import "github.com/cachicamas/backend/agent/src/agent"

// DrainAndCheck blocks until sink closes, dereferencing each event while
// preserving order, and returns agent.CheckStream's own report over the
// drained slice — unmodified, never re-implemented.
func DrainAndCheck(sink <-chan *agent.Event) ([]agent.Event, agent.StreamReport) {
	var events []agent.Event
	for ev := range sink {
		events = append(events, *ev)
	}
	return events, agent.CheckStream(events)
}
