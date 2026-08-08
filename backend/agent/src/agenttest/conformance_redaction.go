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

	"github.com/cachicamas/backend/agent/src/agenttest/sweep"
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

// sentinelDenyList builds the one-entry deny list scanForSentinel and
// scanTextForSentinel hand to [sweep.Scan] (AI-36.1, design.md AD-1): the
// substring-match mechanism now delegates to the module's one shared
// sweep implementation, while this file keeps its own names, its own
// event-walking logic and its own exact message text (a behavior-
// preserving refactor, S-CNF-080).
func sentinelDenyList(sentinel string) []sweep.Entry {
	return []sweep.Entry{{Vector: "sentinel", Needle: []byte(sentinel)}}
}

// verboseRenderingVerbs are the formatting verbs R-CNF-027 item 1 names
// beyond the default/string form scanForSentinel already checked via
// summarize/Error(): the extended (%+v) and Go-syntax (%#v) verbs
// (AI-36.1, WU-2). %v is included too so the sweep covers the verb a
// caller's own fmt.Sprintf/Println would actually dispatch through,
// rather than only the accessor methods summarize/Error already call
// directly.
var verboseRenderingVerbs = []string{"%v", "%+v", "%#v"}

// scanForSentinel walks events, rendering each through the in-package
// summarize (stream_kit_diff.go), the event's own verbose and Go-syntax
// renderings, and Error() plus every formatting verb on any error
// payload, and reports whether the sentinel appears anywhere and — when
// it does — a message naming where, without reprinting the sentinel
// itself (R-CNF-013/R-CNF-027's own requirement). Pure: no testing.TB
// involved, so this package's own tests can prove detection fires against
// a hand-built leaking event without needing a real failing subtest to
// observe it.
func scanForSentinel(events []ai.Event, sentinel string) (leaked bool, whereMsg string) {
	deny := sentinelDenyList(sentinel)
	for i, ev := range events {
		if _, found := sweep.Scan([]byte(summarize(ev)), deny); found {
			return true, fmt.Sprintf("agenttest: redaction: event[%d] (kind=%v)'s summarize() rendering leaked the planted sentinel (R-CNF-013)", i, ev.Kind())
		}
		// R-CNF-027 item 1: the event's own verbose/Go-syntax renderings —
		// event metadata is exactly what these renderings expose (kind,
		// sequence and, for any type whose default formatting is not
		// overridden, its exported fields).
		for _, verb := range verboseRenderingVerbs {
			if _, found := sweep.Scan([]byte(fmt.Sprintf(verb, ev)), deny); found {
				return true, fmt.Sprintf("agenttest: redaction: event[%d] (kind=%v) rendered via %s leaked the planted sentinel (R-CNF-027)", i, ev.Kind(), verb)
			}
		}
		if payload, ok := ev.ErrorPayload(); ok {
			if _, found := sweep.Scan([]byte(payload.Error()), deny); found {
				return true, fmt.Sprintf("agenttest: redaction: event[%d] (kind=%v)'s Failure.Error() leaked the planted sentinel (R-CNF-013)", i, ev.Kind())
			}
			for _, verb := range verboseRenderingVerbs {
				if _, found := sweep.Scan([]byte(fmt.Sprintf(verb, payload)), deny); found {
					return true, fmt.Sprintf("agenttest: redaction: event[%d] (kind=%v)'s error payload rendered via %s leaked the planted sentinel (R-CNF-027)", i, ev.Kind(), verb)
				}
			}
		}
	}
	return false, ""
}

// scanTextForSentinel reports whether text (an already-rendered diagnostic
// corpus, e.g. a captured failure message) contains the sentinel, without
// reprinting either.
func scanTextForSentinel(text, sentinel string) bool {
	_, found := sweep.Scan([]byte(text), sentinelDenyList(sentinel))
	return found
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

// redactionCase proves R-CNF-013/R-CNF-027. It orchestrates two focused
// sweeps (AI-36.1, WU-2): redactionSweepAllCategories, over every
// registered failure category, and redactionSweepDivergenceReport, over
// the suite's own diff/summary output — not only the subject's own
// output.
func redactionCase(t *testing.T, f Factory) {
	t.Helper()
	redactionSweepAllCategories(t, f)
	redactionSweepDivergenceReport(t, f)
}

// contentSentinelSuffix turns a credential-shaped sentinel into a second,
// distinct sentinel standing in for "a sentinel content body" (R-CNF-027
// item 2, S-CNF-078): a separate, independently-detectable value planted
// alongside the credential stand-in, never reused byte-for-byte, so a
// sweep that only happened to catch the credential's shape cannot pass
// vacuously for the content-body claim.
func contentSentinelFor(credentialSentinel string) string {
	return credentialSentinel + "-CONTENT-BODY"
}

// redactionSweepAllCategories widens R-CNF-013's single-category proof to
// R-CNF-027's full requirement: every registered failure category
// (ai.FailureCategories(), "every failure path the suite can trigger"),
// two distinct planted sentinels (a credential stand-in and a
// content-body stand-in, both folded into the failure's Cause — an
// opaque, freeform field — and the credential stand-in additionally into
// RequestID), and scanForSentinel's now-widened sweep (summarize, the
// event's own verbose/Go-syntax renderings, and the error payload's own
// four verbs). Each deny list is proven live via sweep.SelfTest before it
// scans the real corpus (AD-2, S-CNF-079).
func redactionSweepAllCategories(t *testing.T, f Factory) {
	t.Helper()
	credentialSentinel := sentinelOf(f)
	contentSentinel := contentSentinelFor(credentialSentinel)

	if err := sweep.SelfTest(sentinelDenyList(credentialSentinel)); err != nil {
		t.Fatalf("agenttest: redaction: sweep.SelfTest(credential sentinel) = %v, want nil (AD-2 positive control)", err)
	}
	if err := sweep.SelfTest(sentinelDenyList(contentSentinel)); err != nil {
		t.Fatalf("agenttest: redaction: sweep.SelfTest(content sentinel) = %v, want nil (AD-2 positive control)", err)
	}

	for _, category := range ai.FailureCategories() {
		t.Run(category.String(), func(t *testing.T) {
			planted := errors.New("upstream failure detail: " + credentialSentinel + " body: " + contentSentinel)
			failure, err := ai.MidStreamFailure(ai.FailureReport{
				Category:  category,
				Cause:     planted,
				RequestID: credentialSentinel,
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

			if leaked, msg := scanForSentinel(rec.Events(), credentialSentinel); leaked {
				t.Error(msg)
			}
			if leaked, msg := scanForSentinel(rec.Events(), contentSentinel); leaked {
				t.Error(msg)
			}
		})
	}
}

// redactionSweepDivergenceReport forces a divergence through
// RequireSameEvents (two differently-categorised terminals, unstamped)
// and scans ITS OWN failure text, replayed against a capturing double
// rather than the real t (S-CNF-033/S-CNF-079) — the suite's own
// diff/summary path, not only the subject's own output. Both sentinels
// are swept, matching redactionSweepAllCategories's two-sentinel plant.
func redactionSweepDivergenceReport(t *testing.T, f Factory) {
	t.Helper()
	credentialSentinel := sentinelOf(f)
	contentSentinel := contentSentinelFor(credentialSentinel)

	planted := errors.New("upstream failure detail: " + credentialSentinel + " body: " + contentSentinel)
	failure, err := ai.MidStreamFailure(ai.FailureReport{
		Category:  ai.FailureCategoryUnavailable,
		Cause:     planted,
		RequestID: credentialSentinel,
	}, false)
	requireConstructed(t, err, "ai.MidStreamFailure")
	terminal, err := ai.ErrorEvent(failure)
	requireConstructed(t, err, "ai.ErrorEvent")

	otherFailure, err := ai.MidStreamFailure(ai.FailureReport{Category: ai.FailureCategoryTimeout}, false)
	requireConstructed(t, err, "ai.MidStreamFailure")
	otherTerminal, err := ai.ErrorEvent(otherFailure)
	requireConstructed(t, err, "ai.ErrorEvent")

	capture := &capturingTB{}
	RequireSameEvents(capture, []ai.Event{terminal}, []ai.Event{otherTerminal})
	if len(capture.messages) == 0 {
		t.Fatal("RequireSameEvents did not report a divergence between two differently-categorised terminals, want one so this case can scan its own failure text")
	}
	if scanTextForSentinel(capture.allText(), credentialSentinel) {
		t.Error("agenttest: redaction: RequireSameEvents's own divergence report leaked the planted credential sentinel, want the suite's own failure output redacted too (R-CNF-013, S-CNF-033)")
	}
	if scanTextForSentinel(capture.allText(), contentSentinel) {
		t.Error("agenttest: redaction: RequireSameEvents's own divergence report leaked the planted content sentinel, want the suite's own failure output redacted too (R-CNF-013, S-CNF-033)")
	}
}
