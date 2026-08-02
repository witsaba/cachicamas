// AI-23.7 — redaction: a planted sentinel appears in no event, no error
// string, and no test-failure output the suite itself produces
// (R-CNF-013).
//
// # Sentinel channel (design.md D4)
//
// The sentinel is planted into a FailureReport's Cause and RequestID —
// never RawLabel, which stream_kit_diff.go's summaryTable renders by
// design (a bounded, sanctioned fragment, AI-19's own safe-metadata
// posture): planting there would fail a sanctioned rendering rather than
// detect an actual leak. Direct precedent: ai/provider_failure_test.go's
// TestFailure_Error_ExcludesTheCauseAndBoundedMetadata_UnwrapStillExposesThem
// (line ~1172) already proves Failure.Error() itself never reproduces
// Cause/RequestID; this file additionally proves the SUITE's own
// renderings (summarize, RequireSameEvents's diff text) hold the same
// line.

package agenttest

import (
	"errors"
	"fmt"
	"strings"
	"testing"

	"github.com/cachicamas/backend/agent/src/ai"
)

func init() {
	registerConformanceCase("redaction/sentinel_absent_from_every_rendering", CapNone, redactionCase)
}

// defaultRedactionSentinel is Factory.Sentinel's fallback when left "" —
// distinctive enough that an accidental substring collision with ordinary
// fixture text is implausible.
const defaultRedactionSentinel = "AGENTTEST-CONFORMANCE-REDACTION-CANARY-4f3c9e"

// sentinelOf returns f's configured redaction canary, or the suite's own
// default when the factory left it unset ("" selects the default).
func sentinelOf(f Factory) string {
	if f.Sentinel != "" {
		return f.Sentinel
	}
	return defaultRedactionSentinel
}

// scanForSentinel walks events, rendering each through the in-package
// summarize (stream_kit_diff.go) plus Error() on any error payload, and
// reports whether the sentinel appears anywhere and — when it does — a
// message naming where, without reprinting the sentinel itself
// (R-CNF-013's own requirement). Pure: no testing.TB involved, so this
// package's own tests can prove detection fires against a hand-built
// leaking event without needing a real failing subtest to observe it.
func scanForSentinel(events []ai.Event, sentinel string) (leaked bool, whereMsg string) {
	for i, ev := range events {
		if strings.Contains(summarize(ev), sentinel) {
			return true, fmt.Sprintf("agenttest: redaction: event[%d] (kind=%v)'s summarize() rendering leaked the planted sentinel (R-CNF-013)", i, ev.Kind())
		}
		if payload, ok := ev.ErrorPayload(); ok {
			if strings.Contains(payload.Error(), sentinel) {
				return true, fmt.Sprintf("agenttest: redaction: event[%d] (kind=%v)'s Failure.Error() leaked the planted sentinel (R-CNF-013)", i, ev.Kind())
			}
		}
	}
	return false, ""
}

// scanTextForSentinel reports whether text (an already-rendered diagnostic
// corpus, e.g. a captured failure message) contains the sentinel, without
// reprinting either.
func scanTextForSentinel(text, sentinel string) bool {
	return strings.Contains(text, sentinel)
}

// capturingTB is an unexported, message-capturing testing.TB double living
// in production code, not a _test.go file: redactionCase needs to replay a
// forced divergence through AI-22's RequireSameEvents and scan its own
// failure text for a leaked sentinel, without that intentional divergence
// propagating to the real *testing.T driving the suite (Go's testing
// package propagates a failed subtest to every ancestor unconditionally —
// the same constraint this milestone's conformance_suite_test.go records
// throughout). It mirrors agenttest_test's own fakeTB
// (stream_kit_record_test.go) in shape and intent; it cannot be that exact
// type, since a non-test file cannot import a symbol a _test.go file
// declares — this is design.md's "promoting fakeTB" read literally, given
// Go's own build boundary, not a copy-paste.
type capturingTB struct {
	testing.TB
	messages []string
}

func (c *capturingTB) Helper() {}

func (c *capturingTB) Fatalf(format string, args ...any) {
	c.messages = append(c.messages, fmt.Sprintf(format, args...))
}

func (c *capturingTB) Fatal(args ...any) {
	c.messages = append(c.messages, fmt.Sprint(args...))
}

// allText concatenates every captured message — the corpus redactionCase
// scans for a leaked sentinel.
func (c *capturingTB) allText() string {
	return strings.Join(c.messages, "\n")
}

// redactionCase proves R-CNF-013. It plants sentinelOf(f) into a
// mid-stream failure's Cause and RequestID (S-CNF-032), scans every
// drained event's rendering and error text for it, then forces a
// divergence through RequireSameEvents against a capturingTB and scans
// that failure text too (S-CNF-033) — the suite's own diff/summary path,
// not only the subject's own output.
func redactionCase(t *testing.T, f Factory) {
	t.Helper()
	sentinel := sentinelOf(f)

	planted := errors.New("upstream failure detail: " + sentinel)
	failure, err := ai.MidStreamFailure(ai.FailureReport{
		Category:  ai.FailureCategoryUnavailable,
		Cause:     planted,
		RequestID: sentinel,
	}, false)
	requireConstructed(t, err, "ai.MidStreamFailure")
	terminal, err := ai.ErrorEvent(failure)
	requireConstructed(t, err, "ai.ErrorEvent")

	script := Script{Steps: []Step{Emit(terminal)}}
	subject := f.New(t, script)
	ch, err := subject.Stream(t.Context(), minimalRequest(t))
	if err != nil {
		t.Fatalf("agenttest: redaction/sentinel_absent_from_every_rendering: Stream returned %v, want no failure", err)
	}
	rec := DrainAndRecord(t, ch, DefaultDrainTimeout)

	if leaked, msg := scanForSentinel(rec.Events(), sentinel); leaked {
		t.Error(msg)
	}

	// Force a divergence through RequireSameEvents (a different category,
	// unstamped) and scan ITS OWN failure text, replayed against a
	// capturing double rather than the real t (S-CNF-033).
	otherFailure, err := ai.MidStreamFailure(ai.FailureReport{Category: ai.FailureCategoryTimeout}, false)
	requireConstructed(t, err, "ai.MidStreamFailure")
	otherTerminal, err := ai.ErrorEvent(otherFailure)
	requireConstructed(t, err, "ai.ErrorEvent")

	capture := &capturingTB{}
	RequireSameEvents(capture, rec.Events(), []ai.Event{otherTerminal})
	if len(capture.messages) == 0 {
		t.Fatal("RequireSameEvents did not report a divergence between two differently-categorised terminals, want one so this case can scan its own failure text")
	}
	if scanTextForSentinel(capture.allText(), sentinel) {
		t.Error("agenttest: redaction: RequireSameEvents's own divergence report leaked the planted sentinel, want the suite's own failure output redacted too (R-CNF-013, S-CNF-033)")
	}
}
