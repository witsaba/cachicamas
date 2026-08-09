// AI-23.1 — pluggable skeleton: factory seam, capability marking, empty
// case table, a runner that never hangs (R-CNF-001…R-CNF-004, NFR-CNF-E).
//
// This file is package agenttest (white-box), unlike this directory's other
// _test.go files (package agenttest_test): the suite's own runner logic
// needs direct, registry-independent access — an explicit case list rather
// than the package-level registry conformance_text.go and its siblings
// populate via init() — so a unit test of the runner's mechanics stays
// correct regardless of how many leaves have landed by the time the whole
// milestone's tests run together. Go permits both test-package styles to
// coexist in one directory; this package's fake/kit precedent uses
// agenttest_test throughout because those files prove EXTERNAL readability
// (doc.go's role 1/2) — a property this file does not need to prove.
// design.md leaves the case table's concrete keying mechanism to this
// implementation (spec.md "What this file deliberately leaves to design",
// item 2).
//
// # Why this file favours pure-function assertions over literal subtest
// # failure, and what "scratch-verified" means below
//
// Go's testing package propagates a failed subtest to EVERY ancestor
// *testing.T unconditionally (testing.common.Fail walks c.parent.Fail()
// with no opt-out) — confirmed empirically while writing this file. A test
// that literally drives RunConformance/runConformanceCases into a genuine
// t.Fatalf/t.Errorf to prove "this correctly fails" therefore also fails
// the *testing.T doing the observing, which would make `make test` red for
// the wrong reason (NFR-CNF-F requires a recorded GREEN make test, not a
// permanently red one). Wherever a property can be verified without
// touching a real *testing.T's pass/fail state, this file does that: most
// of the runner's decisions are refactored into pure functions
// (factoryDefect, applyDeclaredAbsences, applyTokenCountingCrossCheck,
// declaredAbsentSkipReason, invokeCaseRecovering, Capability.registered)
// or exercised through requireValidFactory/crossCheckDeclaredOptionalCapabilities,
// which take testing.TB precisely so a capturing double (probeTB, this
// file's own analogue of agenttest_test's fakeTB) can stand in without
// ever touching the real tree. The handful of properties that can only be
// observed by watching a real subtest fail and the run still continue
// (S-CNF-003's failing half, S-CNF-009, S-CNF-011, and two NFR-CNF-E
// extreme inputs) were verified once by literally running the scenario and
// reading the output — the same "apply, observe it fails, record, revert"
// discipline AI-20's own R-AMP-016 signature-guard bite proof uses — and
// tasks.md records that captured output. This file's own committed
// regression coverage for those same properties is the pure-function
// proof plus a same-shape passing scenario, so `make test` stays green
// forever rather than intermittently red by design.

package agenttest

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"strings"
	"testing"

	"github.com/cachicamas/backend/agent/src/ai"
)

// boolPtr is this file's own *bool builder for Factory's declared-capability
// fields — a one-line helper repeated across every test in this file rather
// than imported, since taking the address of a literal is not an expression
// Go allows inline.
func boolPtr(b bool) *bool { return &b }

// probeTB is a minimal testing.TB double capturing Fatal/Fatalf/Error/Errorf
// calls instead of touching the real testing tree — this file's own way of
// testing functions that report through testing.TB without incurring Go's
// unconditional subtest-failure propagation to a real parent test. It
// mirrors agenttest_test's own fakeTB (stream_kit_record_test.go),
// unreachable from this internal-package file, restated here rather than
// imported across the test/production package boundary.
type probeTB struct {
	testing.TB // nil: every method this file's helpers call is overridden below
	failed     bool
	messages   []string
}

func (p *probeTB) Helper() {}

func (p *probeTB) Fatalf(format string, args ...any) {
	p.failed = true
	p.messages = append(p.messages, fmt.Sprintf(format, args...))
}

func (p *probeTB) Errorf(format string, args ...any) {
	p.failed = true
	p.messages = append(p.messages, fmt.Sprintf(format, args...))
}

// lastMessage returns the most recently captured message, or "".
func (p *probeTB) lastMessage() string {
	if len(p.messages) == 0 {
		return ""
	}
	return p.messages[len(p.messages)-1]
}

// --- AI-23.1 Item 1 (R-CNF-001) ---

// S-CNF-001 (via GREEN) — RunConformance against an empty case table runs
// to completion and returns a total record whose entries are all not yet
// exercised, since nothing registered a case against them.
func TestConformanceSkeleton_EmptyCaseTable_RunsToCompletionRecordTotal(t *testing.T) {
	record := runConformanceCases(t, FakeFactory(), nil)

	entries := record.Entries()
	if len(entries) != 9 {
		t.Fatalf("len(record.Entries()) = %d, want 9 (R-CNF-017 totality over both closed lists)", len(entries))
	}
	for _, e := range entries {
		if e.Capability.Optional() {
			continue // FakeFactory declares all three explicitly; covered below.
		}
		if e.Outcome != OutcomeNotExercised {
			t.Errorf("required capability %v outcome = %v, want %v (an empty case table exercises no required case)", e.Capability, e.Outcome, OutcomeNotExercised)
		}
	}
}

// TestRunConformance_PublicEntryPoint_AgainstFakeFactory proves the exported
// door: RunConformance(t, FakeFactory()) compiles and runs against the case
// table this package builds via init() registration.
func TestRunConformance_PublicEntryPoint_AgainstFakeFactory(t *testing.T) {
	record := RunConformance(t, FakeFactory())
	if len(record.Entries()) != 9 {
		t.Fatalf("len(record.Entries()) = %d, want 9", len(record.Entries()))
	}
}

// S-CNF-003 (the "remaining cases still run" half) — several cases in one
// run, none of which fail, all run to completion: the loop does not stop
// after the first case for any structural reason. The "that case fails,
// naming the subject and the scenario" half is proven by
// TestInvokeCaseRecovering_PanicsAreCaughtAndTheirValuePreserved below (the
// pure recovery mechanism runOneCase wraps every case in) plus one-time
// scratch verification recorded in tasks.md: running two registered cases
// where the first factory-panics showed
// `--- FAIL: .../probe/first_case_cannot_be_realised` naming the case and
// the panic value, immediately followed by
// `--- PASS: .../probe/second_case_still_runs`, proving continuation for
// real. That specific scenario is not repeated here because Go
// unconditionally propagates a genuinely failing subtest to this test
// function itself, which would make this file's own `make test` red by
// design rather than by defect.
func TestConformanceSkeleton_SeveralCases_AllRunToCompletion(t *testing.T) {
	var ran [3]bool
	cases := []conformanceCase{
		{name: "probe/case_zero", capability: CapNone, run: func(_ *testing.T, _ Factory) { ran[0] = true }},
		{name: "probe/case_one", capability: CapNone, run: func(_ *testing.T, _ Factory) { ran[1] = true }},
		{name: "probe/case_two", capability: CapNone, run: func(_ *testing.T, _ Factory) { ran[2] = true }},
	}

	runConformanceCases(t, FakeFactory(), cases)

	for i, r := range ran {
		if !r {
			t.Errorf("case %d never ran, want every registered case to run", i)
		}
	}
}

// TestInvokeCaseRecovering_PanicsAreCaughtAndTheirValuePreserved is the pure
// proof of R-CNF-001's core mechanism: a panicking case never escapes
// unrecovered, and its value survives for an attributable message.
func TestInvokeCaseRecovering_PanicsAreCaughtAndTheirValuePreserved(t *testing.T) {
	t.Run("a panicking function is caught, its value preserved", func(t *testing.T) {
		panicked, value := invokeCaseRecovering(func() { panic("boom") })
		if !panicked {
			t.Fatal("panicked = false, want true")
		}
		if value != "boom" {
			t.Errorf("value = %#v, want %q", value, "boom")
		}
	})

	t.Run("a nil-pointer-dereference-shaped panic is caught too (proxy for a nil subject)", func(t *testing.T) {
		panicked, _ := invokeCaseRecovering(func() {
			var subject ai.ModelProvider // nil interface
			_, _ = subject.Stream(t.Context(), ai.Request{})
		})
		if !panicked {
			t.Fatal("panicked = false, want true (a nil ModelProvider's method call panics)")
		}
	})

	t.Run("a function that does not panic reports so, with no value", func(t *testing.T) {
		panicked, value := invokeCaseRecovering(func() {})
		if panicked {
			t.Errorf("panicked = true, want false")
		}
		if value != nil {
			t.Errorf("value = %#v, want nil", value)
		}
	})
}

// --- AI-23.1 Item 2 (R-CNF-002, non-negotiable) ---

// S-CNF-004 — a factory declaring CAP-O-03 not offered records the entry
// absent, never not-exercised, even before any case for it exists.
func TestConformanceSkeleton_DeclaredCacheBoundaryAbsent_RecordsAbsentNeverNotExercised(t *testing.T) {
	f := FakeFactory()
	record := runConformanceCases(t, f, nil)

	entry, ok := record.Entry(CapCacheBoundary)
	if !ok {
		t.Fatal("record.Entry(CapCacheBoundary) reports no entry, want one (R-CNF-017 totality)")
	}
	if entry.Outcome != OutcomeAbsent {
		t.Errorf("CapCacheBoundary outcome = %v, want %v (declared not offered is a conclusion, S-CNF-004)", entry.Outcome, OutcomeAbsent)
	}
}

// S-CNF-005 — a factory declaring CAP-O-02 offered by a provider that does
// not satisfy ai.TokenCounter fails the entry, naming the contradiction.
// Exercised twice: once through the pure decision (applyTokenCountingCrossCheck)
// and once through the testing.TB-substitutable wiring (probeTB), so both
// the decision and its reporting are proven without touching a real
// *testing.T.
func TestConformanceSkeleton_DeclaredTokenCountingOffered_ProviderDoesNotSatisfyIt_EntryFails(t *testing.T) {
	t.Run("pure decision", func(t *testing.T) {
		record := newCapabilityRecord(t.Name())
		applyTokenCountingCrossCheck(&record, false)
		entry, ok := record.Entry(CapTokenCounting)
		if !ok || entry.Outcome != OutcomeFailed {
			t.Errorf("entry = %+v (ok=%v), want Outcome=%v", entry, ok, OutcomeFailed)
		}
	})

	t.Run("wired through crossCheckDeclaredOptionalCapabilities with a probeTB", func(t *testing.T) {
		f := Factory{
			New: func(_ testing.TB, scripts ...Script) ai.ModelProvider {
				return NewProvider(scripts...) // agenttest.Provider never implements ai.TokenCounter
			},
			Reasoning:     boolPtr(false),
			TokenCounting: boolPtr(true), // declared offered — a lie the fake cannot back up
			CacheBoundary: boolPtr(false),
		}
		record := newCapabilityRecord(t.Name())
		probe := &probeTB{}
		crossCheckDeclaredOptionalCapabilities(probe, f, &record)

		if !probe.failed {
			t.Error("probe.failed = false, want true (declaration/discovery contradiction, S-CNF-005)")
		}
		entry, ok := record.Entry(CapTokenCounting)
		if !ok || entry.Outcome != OutcomeFailed {
			t.Errorf("entry = %+v (ok=%v), want Outcome=%v", entry, ok, OutcomeFailed)
		}
		if msg := probe.lastMessage(); msg == "" {
			t.Error("no message captured, want one naming the contradiction")
		}
	})

	t.Run("a satisfying subject never fails the entry", func(t *testing.T) {
		f := Factory{
			New: func(_ testing.TB, scripts ...Script) ai.ModelProvider {
				return tokenCountingStub{Provider: NewProvider(scripts...)}
			},
			Reasoning:     boolPtr(false),
			TokenCounting: boolPtr(true),
			CacheBoundary: boolPtr(false),
		}
		record := newCapabilityRecord(t.Name())
		probe := &probeTB{}
		crossCheckDeclaredOptionalCapabilities(probe, f, &record)

		if probe.failed {
			t.Errorf("probe.failed = true, want false (subject genuinely satisfies ai.TokenCounter): %v", probe.messages)
		}
		entry, ok := record.Entry(CapTokenCounting)
		if !ok || entry.Outcome != OutcomeNotExercised {
			t.Errorf("entry = %+v (ok=%v), want Outcome=%v (the cross-check alone never marks satisfied — only a case can)", entry, ok, OutcomeNotExercised)
		}
	})
}

// tokenCountingStub wraps a Provider and additionally satisfies
// ai.TokenCounter, for this file's own declared/discovery-agree scenario.
type tokenCountingStub struct {
	*Provider
}

func (tokenCountingStub) CountTokens(context.Context, ai.Request) (ai.TokenCount, error) {
	return ai.Tokens(0), nil
}

// S-CNF-006 — a factory that never declares one optional capability fails
// at construction, naming the undeclared capability, rather than producing
// a record with a not-exercised entry. Exercised via factoryDefect's pure
// decision and requireValidFactory's testing.TB-substitutable wiring.
func TestConformanceSkeleton_UndeclaredOptionalCapability_FailsConstructionNamingIt(t *testing.T) {
	cases := []struct {
		name string
		f    Factory
		want Capability
	}{
		{
			name: "nil Reasoning",
			f: Factory{
				New:           func(_ testing.TB, scripts ...Script) ai.ModelProvider { return NewProvider(scripts...) },
				Reasoning:     nil,
				TokenCounting: boolPtr(false),
				CacheBoundary: boolPtr(false),
				Retry:         boolPtr(false),
			},
			want: CapReasoningContent,
		},
		{
			name: "nil TokenCounting",
			f: Factory{
				New:           func(_ testing.TB, scripts ...Script) ai.ModelProvider { return NewProvider(scripts...) },
				Reasoning:     boolPtr(false),
				TokenCounting: nil,
				CacheBoundary: boolPtr(false),
				Retry:         boolPtr(false),
			},
			want: CapTokenCounting,
		},
		{
			name: "nil CacheBoundary",
			f: Factory{
				New:           func(_ testing.TB, scripts ...Script) ai.ModelProvider { return NewProvider(scripts...) },
				Reasoning:     boolPtr(false),
				TokenCounting: boolPtr(false),
				CacheBoundary: nil,
				Retry:         boolPtr(false),
			},
			want: CapCacheBoundary,
		},
		{
			name: "nil Retry",
			f: Factory{
				New:           func(_ testing.TB, scripts ...Script) ai.ModelProvider { return NewProvider(scripts...) },
				Reasoning:     boolPtr(false),
				TokenCounting: boolPtr(false),
				CacheBoundary: boolPtr(false),
				Retry:         nil,
			},
			want: CapRetry,
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			msg := factoryDefect(c.f)
			if msg == "" {
				t.Fatal("factoryDefect returned \"\", want a defect naming the undeclared capability (S-CNF-006)")
			}
			if want := c.want.String(); !contains(msg, want) {
				t.Errorf("factoryDefect message %q does not name %q", msg, want)
			}

			probe := &probeTB{}
			requireValidFactory(probe, c.f)
			if !probe.failed {
				t.Error("requireValidFactory did not fail against an undeclared capability, want it to")
			}
		})
	}
}

// contains reports whether s contains sub — this file's own tiny helper so
// it does not need to import strings for one call site.
func contains(s, sub string) bool {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}

// --- AI-23.1 Item 3 (R-CNF-003) ---

// S-CNF-007 — the usage absent-vs-zero case (AI-23.8's node, exercises
// CAP-R-03 completion metadata) computes as required.
func TestConformanceSkeleton_CompletionMetadataStanding_IsRequired(t *testing.T) {
	if CapCompletionMetadata.Optional() {
		t.Error("CapCompletionMetadata.Optional() = true, want false (CAP-R-03 is required, S-CNF-007)")
	}
	if standingOf(CapCompletionMetadata) != StandingRequired {
		t.Errorf("standingOf(CapCompletionMetadata) = %v, want %v", standingOf(CapCompletionMetadata), StandingRequired)
	}
}

// S-CNF-008 — the exactly-one-terminal and redaction cases exercise no
// listed optional capability; both default to required via CapNone.
func TestConformanceSkeleton_NoListedCapabilityCases_DefaultToRequired(t *testing.T) {
	if CapNone.Optional() {
		t.Error("CapNone.Optional() = true, want false — a case exercising no listed capability defaults to required (S-CNF-008)")
	}
	if standingOf(CapNone) != StandingRequired {
		t.Errorf("standingOf(CapNone) = %v, want %v", standingOf(CapNone), StandingRequired)
	}
}

// S-CNF-009 — a case keyed to a capability outside both closed lists is
// marked required (never optional) and, per runRegisteredCase's own source,
// fails loudly naming it: `conformance_suite.go`'s unregistered-capability
// branch reads
// `"agenttest: conformance case %q is keyed to capability %v, outside both
// the CAP-R and CAP-O closed lists — defaulting to required and failing
// (R-CNF-003, S-CNF-009)"` and runs inside its own t.Run subtest, so one
// mis-keyed case cannot abort the rest of the run. That wiring was
// scratch-verified once (tasks.md records the captured
// `--- FAIL: .../wrapper/probe/unclassified_capability` output naming
// exactly capability(250)); this test carries the permanent, pure proof of
// the decision the wiring reads.
func TestConformanceSkeleton_CapabilityRegistered_UnregisteredValuesRejected(t *testing.T) {
	garbage := Capability(250)
	if garbage.registered() {
		t.Error("Capability(250).registered() = true, want false")
	}
	if garbage.Optional() {
		t.Error("Capability(250).Optional() = true, want false (defaults required, S-CNF-009)")
	}
	if standingOf(garbage) != StandingRequired {
		t.Errorf("standingOf(Capability(250)) = %v, want %v", standingOf(garbage), StandingRequired)
	}

	// The zero value is likewise unregistered — a case literal that forgot
	// to set its capability field is caught the same way, never silently
	// read as CapNone's legitimate "no listed capability" marker.
	var zero Capability
	if zero.registered() {
		t.Error("the zero Capability.registered() = true, want false (a forgotten field must not pass as legitimate)")
	}

	// CapNone and every real capability ARE registered.
	if !CapNone.registered() {
		t.Error("CapNone.registered() = false, want true")
	}
	for _, c := range Capabilities() {
		if !c.registered() {
			t.Errorf("%v.registered() = false, want true", c)
		}
	}
}

// --- AI-23.1 Item 4 (R-CNF-004) ---

// S-CNF-010 — a subject offering no optional capability: the run names
// each skipped case with its capability and the absence reason, and the
// skipped case's own body never runs. A t.Skip does not propagate as a
// failure to its parent (unlike t.Fatal/t.Error — confirmed empirically),
// so this scenario is safe to run for real.
func TestConformanceSkeleton_NoOptionalCapabilityOffered_SkippedCasesReportedNeverSilent(t *testing.T) {
	f := FakeFactory()
	*f.Reasoning = false // this run declares nothing offered, unlike FakeFactory's own default

	var bodyRan bool
	cases := []conformanceCase{
		{name: "reasoning/probe_must_not_run", capability: CapReasoningContent, run: func(t *testing.T, _ Factory) {
			bodyRan = true
			t.Fatal("this case's body must never run when its capability is declared absent")
		}},
	}

	record := runConformanceCases(t, f, cases)
	if bodyRan {
		t.Fatal("the skipped case's body ran, want the runner to skip it before invoking it (R-CNF-004)")
	}
	entry, ok := record.Entry(CapReasoningContent)
	if !ok || entry.Outcome != OutcomeAbsent {
		t.Errorf("CapReasoningContent entry = %+v (ok=%v), want Outcome=%v", entry, ok, OutcomeAbsent)
	}

	skip, reason := declaredAbsentSkipReason(f, CapReasoningContent)
	if !skip || reason == "" {
		t.Errorf("declaredAbsentSkipReason = (%v, %q), want (true, a non-empty reason)", skip, reason)
	}
}

// S-CNF-011 — an attempt to skip a required case fails, naming the
// required case, rather than honouring the skip. declaredAbsentSkipReason
// is the pure gate this property rests on: it is deliberately keyed off
// Capability.Optional() first, so no required capability — including one a
// hypothetical caller tried to mark absent — is ever skippable. The
// runner's own escalation (runRegisteredCase: a case that skips itself
// despite required standing gets `t.Errorf` naming it, per R-CNF-004's
// no-waivers rule) was scratch-verified once; tasks.md records the captured
// output naming `no_listed_capability` and citing S-CNF-011.
func TestConformanceSkeleton_DeclaredAbsentSkipReason_NeverSkipsARequiredCapability(t *testing.T) {
	allFalse := Factory{Reasoning: boolPtr(false), TokenCounting: boolPtr(false), CacheBoundary: boolPtr(false)}
	for _, c := range []Capability{CapStreamingText, CapToolCalls, CapCompletionMetadata, CapCancellation, CapTypedFailures, CapNone} {
		skip, reason := declaredAbsentSkipReason(allFalse, c)
		if skip {
			t.Errorf("declaredAbsentSkipReason(%v) = (true, %q), want (false, \"\") — required capabilities are never skippable (R-CNF-004, S-CNF-011)", c, reason)
		}
	}
}

// --- AI-23.1 Item 5 (NFR-CNF-E) ---

// S-CNF-066 (partial) — a nil factory, an empty case selection, a nil
// subject and an undeclared capability expectation each fail attributably,
// naming the entry point and the offending input, and never panic. The
// nil-New and undeclared-capability branches are proven directly via
// factoryDefect/requireValidFactory+probeTB (safe, real assertions). The
// nil-subject and panicking-case branches are proven by
// TestInvokeCaseRecovering_PanicsAreCaughtAndTheirValuePreserved's pure
// coverage of the same recovery mechanism runOneCase wraps every case
// with, plus one-time scratch verification (tasks.md records
// `conformance_suite.go:...: agenttest: conformance case "probe/nil_subject"
// (capability=no_listed_capability) panicked: runtime error: invalid memory
// address or nil pointer dereference (R-CNF-001)` and the equivalent line
// for a case that panics directly, both attributable and non-crashing).
func TestConformanceSkeleton_ExtremeInputs_FailAttributablyNeverPanic(t *testing.T) {
	t.Run("nil New func", func(t *testing.T) {
		f := Factory{Reasoning: boolPtr(false), TokenCounting: boolPtr(false), CacheBoundary: boolPtr(false)}
		if msg := factoryDefect(f); msg == "" {
			t.Error("factoryDefect(nil New) returned \"\", want a defect")
		}
		probe := &probeTB{}
		requireValidFactory(probe, f)
		if !probe.failed {
			t.Error("requireValidFactory did not fail against a nil New func, want it to (NFR-CNF-E)")
		}
	})

	t.Run("empty case selection", func(t *testing.T) {
		defer requireNoPanicHere(t)
		record := runConformanceCases(t, FakeFactory(), []conformanceCase{})
		if len(record.Entries()) != 9 {
			t.Errorf("an empty case selection must still return a total record; got %d entries, want 9", len(record.Entries()))
		}
	})

	t.Run("undeclared capability expectation", func(t *testing.T) {
		f := Factory{
			New:           func(_ testing.TB, scripts ...Script) ai.ModelProvider { return NewProvider(scripts...) },
			Reasoning:     boolPtr(false),
			TokenCounting: boolPtr(false),
			CacheBoundary: nil,
		}
		probe := &probeTB{}
		requireValidFactory(probe, f)
		if !probe.failed {
			t.Error("requireValidFactory did not fail against an undeclared CacheBoundary, want it to")
		}
	})

	t.Run("nil subject and panicking case: recovery mechanism, pure", func(t *testing.T) {
		defer requireNoPanicHere(t)

		panicked, _ := invokeCaseRecovering(func() {
			var subject ai.ModelProvider
			_, _ = subject.Stream(t.Context(), ai.Request{})
		})
		if !panicked {
			t.Error("a nil subject's Stream call did not panic under invokeCaseRecovering, want it caught")
		}

		panicked, _ = invokeCaseRecovering(panickyRun2)
		if !panicked {
			t.Error("a deliberately panicking case body was not caught")
		}
	})
}

// panickyRun2 always panics — invokeCaseRecovering's own extreme-input
// fixture (distinct name from any fixture a later leaf might add).
func panickyRun2() { panic("agenttest: deliberately panicking (test fixture)") }

// requireNoPanicHere converts a panic on the calling goroutine into a test
// failure naming it, rather than letting it crash the test binary — this
// file's own copy of the pattern fake_queue_test.go's requireNoPanic
// establishes in the sibling agenttest_test package (unreachable from here,
// a different package).
func requireNoPanicHere(t *testing.T) {
	t.Helper()
	if r := recover(); r != nil {
		t.Fatalf("panicked: %v, want no panic escaping RunConformance for this input (NFR-CNF-E)", r)
	}
}

// === AI-23.2 — text and lifecycle cases (R-CNF-005, R-CNF-006) ===

// S-CNF-012, S-CNF-013 — textOrderingCase passes against FakeFactory's real
// scripted subject: order, contiguity and the split-rune reconstruction all
// hold. Safe to run directly (no failure expected).
func TestConformanceText_OrderingCase_PassesAgainstFakeFactory(t *testing.T) {
	textOrderingCase(t, FakeFactory())
}

// S-CNF-015, S-CNF-016 — textEmptyCompletionCase passes: no text content is
// conformant, not a failure or a contiguity violation.
func TestConformanceText_EmptyCompletionCase_PassesAgainstFakeFactory(t *testing.T) {
	textEmptyCompletionCase(t, FakeFactory())
}

// reorderedTextProvider is a minimal ai.ModelProvider whose Stream sends a
// text delta before its own block's start — deliberately violating
// ordering, a shape AI-21's own Provider structurally refuses to script
// (its internal ai.CheckStream pre-check panics on such a script before any
// goroutine starts). This file's own hand-built test double is what lets
// S-CNF-014 exist at all: proving the suite surfaces a real ordering
// violation, never trusting a subject to misbehave through the fake.
type reorderedTextProvider struct{}

func (reorderedTextProvider) Stream(_ context.Context, _ ai.Request) (<-chan ai.Event, error) {
	out := make(chan ai.Event, 4)
	var stamper ai.Stamper

	delta, err := ai.NewTextDelta(1, "oops")
	if err != nil {
		return nil, err
	}
	start, err := ai.NewTextBlockStart(1)
	if err != nil {
		return nil, err
	}
	end, err := ai.NewTextBlockEnd(1)
	if err != nil {
		return nil, err
	}
	completion, err := ai.NewCompletion(ai.FinishReasonStop, ai.Usage{})
	if err != nil {
		return nil, err
	}

	out <- stamper.Stamp(delta) // delta BEFORE its block's own start — the violation
	out <- stamper.Stamp(start)
	out <- stamper.Stamp(end)
	out <- stamper.Stamp(completion)
	close(out)
	return out, nil
}

// S-CNF-014 — a subject whose deltas arrive reordered: RequireValidStream —
// textOrderingCase's only ordering assertion (confirmed by inspection: it
// calls nothing else that could report an ordering problem, satisfying this
// item's own REFACTOR check) — fails, carrying ai.CheckStream's own
// verdict, unmodified. Proven directly against RequireValidStream with a
// probeTB rather than through textOrderingCase(t, badFactory) end to end,
// for the same propagation reason recorded on AI-23.1's Item 1: a genuine
// failure inside a real *testing.T subtree marks every ancestor failed,
// which would make this meta-test's own `make test` red by construction.
// The full, real, end-to-end failure was scratch-verified once (tasks.md
// records the captured output).
func TestConformanceText_ReorderedSubject_RequireValidStreamCarriesCheckStreamVerdict(t *testing.T) {
	subject := reorderedTextProvider{}
	ch, err := subject.Stream(t.Context(), minimalRequest(t))
	if err != nil {
		t.Fatalf("reorderedTextProvider.Stream returned %v, want no failure constructing the scenario itself", err)
	}

	probe := &probeTB{}
	rec := DrainAndRecord(probe, ch, DefaultDrainTimeout)
	if probe.failed {
		t.Fatalf("DrainAndRecord itself failed: %v, want the drain to succeed so RequireValidStream can inspect the ordering", probe.messages)
	}
	RequireValidStream(probe, rec)

	if !probe.failed {
		t.Fatal("RequireValidStream did not fail against a subject with a delta before its block's start, want it to (S-CNF-014)")
	}
	if msg := probe.lastMessage(); !contains(msg, "R-STK-005") {
		t.Errorf("failure message %q does not cite R-STK-005 (RequireValidStream's own citation of ai.CheckStream's verdict)", msg)
	}
}

// NFR-CNF-E (partial, this leaf's own item 3) — the text case's assertion
// path given a zero-length script: attributable failure, never a panic. A
// zero-length script means Stream never emits a terminal event, so
// DrainAndRecord hits its own bounded deadline — proving that path fails
// attributably rather than hanging or panicking is this leaf's own
// contribution to totality.
func TestConformanceText_ZeroLengthScript_FailsAttributablyNeverPanics(t *testing.T) {
	defer requireNoPanicHere(t)

	script := Script{} // zero steps: Stream returns a channel that only ever closes... actually never closes either, since produce() with zero steps runs straight to the deferred close.
	f := FakeFactory()
	subject := f.New(t, script)
	ch, err := subject.Stream(t.Context(), minimalRequest(t))
	if err != nil {
		t.Fatalf("Stream returned %v for a zero-length script, want a usable (if immediately-closing) carrier", err)
	}

	probe := &probeTB{}
	rec := DrainAndRecord(probe, ch, DefaultDrainTimeout)
	if probe.failed {
		t.Fatalf("DrainAndRecord failed against a zero-length script's stream: %v, want a clean, immediate close", probe.messages)
	}
	if got := rec.Len(); got != 0 {
		t.Errorf("rec.Len() = %d, want 0 (a zero-length script emits nothing and closes bare)", got)
	}
}

// === AI-23.3 — tool-call cases (R-CNF-007, R-CNF-008) ===

// S-CNF-017 — two concurrent tool calls whose argument fragments interleave
// each reconstruct exactly, with no cross-contamination.
func TestConformanceToolCall_InterleavedCase_PassesAgainstFakeFactory(t *testing.T) {
	toolCallInterleavedCase(t, FakeFactory())
}

// S-CNF-018 — a zero-delta whole tool call is accepted.
func TestConformanceToolCall_ZeroDeltaCase_PassesAgainstFakeFactory(t *testing.T) {
	toolCallZeroDeltaCase(t, FakeFactory())
}

// S-CNF-019 — two calls to the same tool name carry distinct, ordered
// ordinals.
func TestConformanceToolCall_OrdinalCase_PassesAgainstFakeFactory(t *testing.T) {
	toolCallOrdinalCase(t, FakeFactory())
}

// S-CNF-020 — mixed text and tool-call content survives, terminating on
// the tool-call finish reason.
func TestConformanceToolCall_MixedTextAndToolCase_PassesAgainstFakeFactory(t *testing.T) {
	mixedTextAndToolCallCase(t, FakeFactory())
}

// NFR-CNF-E (partial, this leaf's own item 3) — a tool call with an empty
// (non-nil, zero-length) argument-byte payload: attributable pass, never a
// panic. Empty deltas are unrestricted by tool_call_event.go's own rules
// (R-ATC-005), so this is a genuine pass, proving reconstructToolCalls
// handles a zero-length fragment/argument slice without panicking (nil
// slice append and bytes.Equal against an empty slice are both safe in
// Go, but this proves it end to end rather than by inspection alone).
func TestConformanceToolCall_EmptyArgumentPayload_FailsAttributablyNeverPanics(t *testing.T) {
	defer requireNoPanicHere(t)

	start, err := ai.NewToolCallStart(1, "call-1", "search")
	if err != nil {
		t.Fatalf("ai.NewToolCallStart returned %v, want no failure", err)
	}
	delta, err := ai.NewToolCallDelta(1, []byte{}) // empty, non-nil fragment
	if err != nil {
		t.Fatalf("ai.NewToolCallDelta returned %v, want no failure", err)
	}
	end, err := ai.NewToolCallEnd(1, []byte{}) // empty, non-nil arguments
	if err != nil {
		t.Fatalf("ai.NewToolCallEnd returned %v, want no failure", err)
	}

	script := Script{Steps: []Step{Emit(start), Emit(delta), Emit(end)}}
	subject := FakeFactory().New(t, script)
	ch, err := subject.Stream(t.Context(), minimalRequest(t))
	if err != nil {
		t.Fatalf("Stream returned %v, want no failure", err)
	}
	rec := DrainAndRecord(t, ch, DefaultDrainTimeout)
	RequireValidStream(t, rec)

	calls := reconstructToolCalls(rec.Events())
	call, ok := calls[1]
	if !ok || !call.sawStart || !call.sawEnd {
		t.Fatalf("call = %+v (found=%v), want a complete start+end even with empty payloads", call, ok)
	}
	if len(call.arguments) != 0 {
		t.Errorf("arguments = %q, want empty", call.arguments)
	}
}

// === AI-23.4 — terminal and error cases (R-CNF-009, R-CNF-010) ===

// S-CNF-021 — the normal, pre-stream and mid-stream paths all pass.
func TestConformanceTerminal_ExactlyOneCase_PassesAgainstFakeFactory(t *testing.T) {
	terminalExactlyOneCase(t, FakeFactory())
}

// S-CNF-022 — the partial-output discriminator reports both states
// correctly.
func TestConformanceTerminal_DiscriminatorCase_PassesAgainstFakeFactory(t *testing.T) {
	terminalDiscriminatorCase(t, FakeFactory())
}

// S-CNF-024 — all nine categories are exercised and classified on both
// delivery paths.
func TestConformanceTerminal_FailureCategoryExhaustivenessCase_PassesAgainstFakeFactory(t *testing.T) {
	terminalFailureCategoryExhaustivenessCase(t, FakeFactory())

	if got := len(ai.FailureCategories()); got != 9 {
		t.Errorf("len(ai.FailureCategories()) = %d, want 9 — this suite's own exhaustiveness pin (R-CNF-010)", got)
	}
}

// S-CNF-025 — the exhaustiveness check itself fails, naming the uncovered
// category, when the exercised set is artificially shrunk below the
// enumerator's length. This is the pure, permanent proof that a future
// upstream addition to ai.FailureCategories() would be caught: a real
// tenth category cannot be constructed (the vocabulary is closed), so the
// check's OWN logic is what this test exercises.
func TestRequireFailureCategoryCoverage_ShrunkExercisedSet_NamesTheUncoveredCategory(t *testing.T) {
	all := ai.FailureCategories()
	if len(all) < 2 {
		t.Fatal("ai.FailureCategories() has fewer than 2 members, this test needs at least one to omit")
	}
	missing := all[0]
	exercised := make(map[ai.FailureCategory]bool, len(all)-1)
	for _, c := range all[1:] {
		exercised[c] = true
	}

	err := requireFailureCategoryCoverage(exercised)
	if err == nil {
		t.Fatal("requireFailureCategoryCoverage returned nil against a shrunk set, want an error naming the missing category (S-CNF-025)")
	}
	if !contains(err.Error(), missing.String()) {
		t.Errorf("error %q does not name the missing category %v", err, missing)
	}

	// The companion case: a fully covered set reports no violation.
	full := make(map[ai.FailureCategory]bool, len(all))
	for _, c := range all {
		full[c] = true
	}
	if err := requireFailureCategoryCoverage(full); err != nil {
		t.Errorf("requireFailureCategoryCoverage(full) = %v, want nil", err)
	}
}

// postTerminalEventProvider is a minimal ai.ModelProvider whose Stream
// sends an event after its own terminal completion — deliberately
// violating R-CNF-009's exactly-one-terminal rule, a shape AI-21's own
// Provider structurally refuses to script (S-CNF-023's own reason for
// needing a hand-built double, matching reorderedTextProvider's precedent).
type postTerminalEventProvider struct{}

func (postTerminalEventProvider) Stream(_ context.Context, _ ai.Request) (<-chan ai.Event, error) {
	out := make(chan ai.Event, 2)
	var stamper ai.Stamper

	completion, err := ai.NewCompletion(ai.FinishReasonStop, ai.Usage{})
	if err != nil {
		return nil, err
	}
	start, err := ai.NewTextBlockStart(1)
	if err != nil {
		return nil, err
	}

	out <- stamper.Stamp(completion)
	out <- stamper.Stamp(start) // AFTER the terminal — the violation
	close(out)
	return out, nil
}

// S-CNF-023 — a subject emitting an event after its terminal:
// RequireValidStream — the only ordering assertion terminalExactlyOneCase
// calls, confirmed by inspection — fails, naming the post-terminal event.
// Proven directly against RequireValidStream with a probeTB, for the same
// propagation reason recorded throughout this file.
func TestConformanceTerminal_PostTerminalEvent_RequireValidStreamNamesIt(t *testing.T) {
	subject := postTerminalEventProvider{}
	ch, err := subject.Stream(t.Context(), minimalRequest(t))
	if err != nil {
		t.Fatalf("postTerminalEventProvider.Stream returned %v, want no failure constructing the scenario", err)
	}

	probe := &probeTB{}
	rec := DrainAndRecord(probe, ch, DefaultDrainTimeout)
	if probe.failed {
		t.Fatalf("DrainAndRecord itself failed: %v, want the drain to succeed so RequireValidStream can inspect ordering", probe.messages)
	}
	RequireValidStream(probe, rec)

	if !probe.failed {
		t.Fatal("RequireValidStream did not fail against an event following the terminal, want it to (S-CNF-023)")
	}
}

// NFR-CNF-E (partial, this leaf's own item 3) — a failure category value
// with no attached message (a bare FailureReport{Category: c}, every other
// field at its zero value) never panics on either construction path.
func TestConformanceTerminal_FailureCategoryWithNoAttachedMessage_NeverPanics(t *testing.T) {
	defer requireNoPanicHere(t)

	for _, category := range ai.FailureCategories() {
		if _, err := ai.PreStreamFailure(ai.FailureReport{Category: category}); err != nil {
			t.Errorf("ai.PreStreamFailure(%v) returned %v, want no failure for a bare report", category, err)
		}
		if _, err := ai.MidStreamFailure(ai.FailureReport{Category: category}, false); err != nil {
			t.Errorf("ai.MidStreamFailure(%v) returned %v, want no failure for a bare report", category, err)
		}
	}
}

// === AI-23.5 — cancellation and closure cases (R-CNF-011, R-CNF-012) ===
//
// Neither test below calls t.Parallel(), matching conformance_cancellation.go's
// own R-STK-008 non-negotiable rule — required for RequireNoGoroutineLeak's
// process-wide goroutine count to mean anything.

// S-CNF-026, S-CNF-027 — cancelling mid-consumption closes within the
// bounded drain deadline, repeated 50 times (RequireNoGoroutineLeak's own
// amplitude) with no goroutine growth beyond tolerance.
func TestConformanceCancellation_BoundedCloseCase_PassesAgainstFakeFactory(t *testing.T) {
	cancellationBoundedCloseCase(t, FakeFactory())
}

// S-CNF-029, S-CNF-030 — an abandoned-then-cancelled stream closes bare,
// no terminal invented, repeated with no goroutine growth beyond
// tolerance.
func TestConformanceCancellation_AbandonedThenCancelledCase_PassesAgainstFakeFactory(t *testing.T) {
	cancellationAbandonedThenCancelledCase(t, FakeFactory())
}

// === AI-38 design D4 — requireCancellationTail self-tests (S-CNF-082, S-CNF-083) ===
//
// Hand-built ai.Event doubles, driven directly through probeTB rather than a
// real Stream call — the two tests above already prove the amended helper
// against a genuinely running fake, so this section's job is narrower: pin
// each individually-rejected shape (two terminals, wrong category, an event
// after the terminal) and the two admitted shapes (bare, and one correctly
// placed and categorized terminal), independent of any subject's timing.

// mustCancellationErrorEvent builds one ai.ErrorEvent whose failure carries
// category — this section's own minimal event builder for hand-constructed
// tail shapes.
func mustCancellationErrorEvent(tb testing.TB, category ai.FailureCategory) ai.Event {
	tb.Helper()
	failure, err := ai.MidStreamFailure(ai.FailureReport{Category: category}, false)
	if err != nil {
		tb.Fatalf("ai.MidStreamFailure(%v) returned %v, want no failure", category, err)
	}
	ev, err := ai.ErrorEvent(failure)
	if err != nil {
		tb.Fatalf("ai.ErrorEvent returned %v, want no failure", err)
	}
	return ev
}

// TestRequireCancellationTail_BareClose_Admitted proves the zero-event tail
// (a true bare close) is accepted with no reported failure.
func TestRequireCancellationTail_BareClose_Admitted(t *testing.T) {
	probe := &probeTB{}
	requireCancellationTail(probe, nil)
	if probe.failed {
		t.Errorf("requireCancellationTail(nil) failed = %v, messages = %v, want no failure for a bare close (R-CNF-011/R-CNF-012)", probe.failed, probe.messages)
	}
}

// TestRequireCancellationTail_SingleCancellationTerminal_Admitted proves the
// shape design D4 adds: an ordinary event, then exactly one
// cancellation-category terminal in final position, is accepted.
func TestRequireCancellationTail_SingleCancellationTerminal_Admitted(t *testing.T) {
	probe := &probeTB{}
	end, err := ai.NewTextBlockEnd(1)
	if err != nil {
		t.Fatalf("ai.NewTextBlockEnd returned %v, want no failure", err)
	}
	events := []ai.Event{end, mustCancellationErrorEvent(t, ai.FailureCategoryCancellation)}

	requireCancellationTail(probe, events)
	if probe.failed {
		t.Errorf("requireCancellationTail([TextBlockEnd, cancellation terminal]) failed = %v, messages = %v, want no failure (design D4's amended admission)", probe.failed, probe.messages)
	}
}

// TestRequireCancellationTail_TwoTerminals_Rejected proves S-CNF-082/083's
// "more than one terminal" rejection: two cancellation-category terminals
// still fail, naming the observed count.
func TestRequireCancellationTail_TwoTerminals_Rejected(t *testing.T) {
	probe := &probeTB{}
	events := []ai.Event{
		mustCancellationErrorEvent(t, ai.FailureCategoryCancellation),
		mustCancellationErrorEvent(t, ai.FailureCategoryCancellation),
	}

	requireCancellationTail(probe, events)
	if !probe.failed {
		t.Fatal("requireCancellationTail([2 cancellation terminals]) did not fail, want a rejection naming the count (S-CNF-082/083)")
	}
	if got := probe.lastMessage(); !strings.Contains(got, "2 error terminal") {
		t.Errorf("requireCancellationTail message = %q, want it to name the observed terminal count", got)
	}
}

// TestRequireCancellationTail_WrongCategory_Rejected proves a terminal of
// any non-cancellation category still fails, naming the observed and wanted
// category (S-CNF-082/083).
func TestRequireCancellationTail_WrongCategory_Rejected(t *testing.T) {
	probe := &probeTB{}

	requireCancellationTail(probe, []ai.Event{mustCancellationErrorEvent(t, ai.FailureCategoryTimeout)})
	if !probe.failed {
		t.Fatal("requireCancellationTail([timeout terminal]) did not fail, want a rejection naming the wrong category (S-CNF-082/083)")
	}
	if got := probe.lastMessage(); !strings.Contains(got, "timeout") || !strings.Contains(got, "cancellation") {
		t.Errorf("requireCancellationTail message = %q, want it to name both the observed and wanted category", got)
	}
}

// TestRequireCancellationTail_EventAfterTerminal_Rejected proves any event
// following the admitted terminal still fails, naming that it is not in
// final position (S-CNF-082/083).
func TestRequireCancellationTail_EventAfterTerminal_Rejected(t *testing.T) {
	probe := &probeTB{}
	delta, err := ai.NewTextDelta(1, "leaked-after-terminal")
	if err != nil {
		t.Fatalf("ai.NewTextDelta returned %v, want no failure", err)
	}
	events := []ai.Event{mustCancellationErrorEvent(t, ai.FailureCategoryCancellation), delta}

	requireCancellationTail(probe, events)
	if !probe.failed {
		t.Fatal("requireCancellationTail([cancellation terminal, TextDelta]) did not fail, want a rejection naming the trailing event (S-CNF-082/083)")
	}
	if got := probe.lastMessage(); !strings.Contains(got, "final position") {
		t.Errorf("requireCancellationTail message = %q, want it to say the terminal is not in final position", got)
	}
}

// TestRequireCancellationTail_InventedCompletion_Rejected proves a
// fabricated Completion is rejected regardless of position — unchanged by
// design D4 (R-CNF-011/R-CNF-012).
func TestRequireCancellationTail_InventedCompletion_Rejected(t *testing.T) {
	probe := &probeTB{}
	completion, err := ai.NewCompletion(ai.FinishReasonStop, ai.Usage{})
	if err != nil {
		t.Fatalf("ai.NewCompletion returned %v, want no failure", err)
	}

	requireCancellationTail(probe, []ai.Event{completion})
	if !probe.failed {
		t.Fatal("requireCancellationTail([Completion]) did not fail, want a rejection naming the invented Completion (R-CNF-011/R-CNF-012)")
	}
	if got := probe.lastMessage(); !strings.Contains(got, "Completion") {
		t.Errorf("requireCancellationTail message = %q, want it to name the invented Completion", got)
	}
}

// === AI-23.7 — redaction cases (R-CNF-013) ===

// S-CNF-032, S-CNF-033 — redactionCase passes against FakeFactory: neither
// the planted sentinel's event rendering nor the suite's own forced-
// divergence failure text leaks it.
func TestConformanceRedaction_Case_PassesAgainstFakeFactory(t *testing.T) {
	redactionCase(t, FakeFactory())
}

// S-CNF-034 — a leaking rendering is detected, naming where, without
// reprinting the sentinel itself. RawLabel is summaryTable's own sanctioned
// rendering (design.md D4), so a sentinel planted there — unlike
// Cause/RequestID — legitimately surfaces through summarize(), simulating
// "a subject that does leak" without needing a hand-built ai.ModelProvider.
// Proven directly against the pure scanForSentinel, with zero testing.T
// propagation risk.
func TestScanForSentinel_LeakingRawLabel_DetectsItNamesWhereNeverReprintsIt(t *testing.T) {
	const sentinel = "LEAK-CANARY-b7e2"

	leaky, err := ai.MidStreamFailure(ai.FailureReport{Category: ai.FailureCategoryUnknown, RawLabel: sentinel}, false)
	if err != nil {
		t.Fatalf("ai.MidStreamFailure returned %v, want no failure", err)
	}
	terminal, err := ai.ErrorEvent(leaky)
	if err != nil {
		t.Fatalf("ai.ErrorEvent returned %v, want no failure", err)
	}

	leaked, msg := scanForSentinel([]ai.Event{terminal}, sentinel)
	if !leaked {
		t.Fatal("scanForSentinel did not detect a sentinel planted in RawLabel (summaryTable's own sanctioned rendering), want detection (S-CNF-034)")
	}
	if msg == "" {
		t.Error("scanForSentinel reported leaked=true with an empty message, want one naming where")
	}
	if contains(msg, sentinel) {
		t.Errorf("scanForSentinel's message %q reprints the sentinel itself, want it named only by location (S-CNF-034)", msg)
	}
	if !contains(msg, "event[0]") {
		t.Errorf("message %q does not name the offending event's index", msg)
	}

	// Companion: a genuinely clean event reports no leak.
	clean, err := ai.NewCompletion(ai.FinishReasonStop, ai.Usage{})
	if err != nil {
		t.Fatalf("ai.NewCompletion returned %v, want no failure", err)
	}
	if leaked, _ := scanForSentinel([]ai.Event{clean}, sentinel); leaked {
		t.Error("scanForSentinel reported a leak against a completion event that never carries the sentinel, want none")
	}
}

// NFR-CNF-E (partial, this leaf's own item 2) — an empty-string sentinel
// falls back to the suite default (S-CNF-066 partial), and a sentinel
// containing characters that could break naive string matching (format
// verbs, backslashes, quotes) never panics.
func TestConformanceRedaction_ExtremeSentinels_FailAttributablyNeverPanic(t *testing.T) {
	t.Run("empty string falls back to the default", func(t *testing.T) {
		if got := sentinelOf(Factory{Sentinel: ""}); got != defaultRedactionSentinel {
			t.Errorf("sentinelOf(empty) = %q, want the suite default %q", got, defaultRedactionSentinel)
		}
	})

	t.Run("a sentinel with format verbs, backslashes and quotes never panics", func(t *testing.T) {
		defer requireNoPanicHere(t)

		f := FakeFactory()
		f.Sentinel = `%s%d\n<>&"';--CANARY`
		redactionCase(t, f)
	})
}

// === AI-23.8 — optional-capability cases (R-CNF-014, R-CNF-015, R-CNF-016) ===

// S-CNF-035, S-CNF-036, S-CNF-037 — reasoningWholeBlocksCase passes against
// FakeFactory (which declares Reasoning offered).
func TestConformanceCapabilities_ReasoningCase_PassesAgainstFakeFactory(t *testing.T) {
	reasoningWholeBlocksCase(t, FakeFactory())
}

// S-CNF-038 — a factory declaring reasoning not offered records the entry
// absent and the case's body never runs — the runner's generic
// declared-absent mechanism (AI-23.1, already proven for CapReasoningContent
// by TestConformanceSkeleton_NoOptionalCapabilityOffered_...), reconfirmed
// here against this leaf's own registered case via runConformanceCases.
func TestConformanceCapabilities_ReasoningDeclaredAbsent_SkippedRecordedAbsent(t *testing.T) {
	f := FakeFactory()
	*f.Reasoning = false

	record := runConformanceCases(t, f, []conformanceCase{
		{name: "reasoning/whole_blocks_never_leak_into_text", capability: CapReasoningContent, run: reasoningWholeBlocksCase},
	})
	entry, ok := record.Entry(CapReasoningContent)
	if !ok || entry.Outcome != OutcomeAbsent {
		t.Errorf("CapReasoningContent entry = %+v (ok=%v), want Outcome=%v (S-CNF-038)", entry, ok, OutcomeAbsent)
	}
}

// tokenCountingStubDeclining wraps a Provider, satisfies ai.TokenCounter,
// and always declines to answer — CAP-O-02's "advertised but non-conformant"
// shape (S-CNF-041), distinct from tokenCountingStub (AI-23.1), which
// answers cleanly.
type tokenCountingStubDeclining struct {
	*Provider
}

func (tokenCountingStubDeclining) CountTokens(context.Context, ai.Request) (ai.TokenCount, error) {
	return ai.TokenCount{}, errors.New("agenttest: token_counting stub deliberately declines to answer (test fixture)")
}

// S-CNF-039 — a subject satisfying the token-counting contract answers a
// genuine, present count, and the case asserts it.
func TestConformanceCapabilities_TokenCountingCase_SatisfyingSubject_PassesAndAssertsTheCount(t *testing.T) {
	f := Factory{
		New: func(_ testing.TB, scripts ...Script) ai.ModelProvider {
			return tokenCountingStub{Provider: NewProvider(scripts...)}
		},
		Reasoning:     boolPtr(false),
		TokenCounting: boolPtr(true),
		CacheBoundary: boolPtr(false),
	}
	tokenCountingCase(t, f)
}

// S-CNF-040 — a factory declaring token counting not offered records a
// clean absence: no error, no substituted zero, the entry is absent.
func TestConformanceCapabilities_TokenCountingDeclaredAbsent_CleanAbsence(t *testing.T) {
	f := FakeFactory() // TokenCounting: false, and Provider never satisfies ai.TokenCounter
	record := runConformanceCases(t, f, []conformanceCase{
		{name: "token_counting/asked_of_the_provider_value", capability: CapTokenCounting, run: tokenCountingCase},
	})
	entry, ok := record.Entry(CapTokenCounting)
	if !ok || entry.Outcome != OutcomeAbsent {
		t.Errorf("CapTokenCounting entry = %+v (ok=%v), want Outcome=%v (S-CNF-040)", entry, ok, OutcomeAbsent)
	}
}

// S-CNF-041 — a subject that satisfies the token-counting contract and then
// declines to answer: the underlying mechanism (tokenCounterOf +
// CountTokens) reports the error tokenCountingCase would turn into a
// failure. Proven directly against the ingredients rather than running the
// case with a real Fatal, for the propagation reason recorded throughout
// this file.
func TestConformanceCapabilities_TokenCountingDeclines_ReportsAnErrorNotAbsence(t *testing.T) {
	f := Factory{
		New: func(_ testing.TB, scripts ...Script) ai.ModelProvider {
			return tokenCountingStubDeclining{Provider: NewProvider(scripts...)}
		},
		Reasoning:     boolPtr(false),
		TokenCounting: boolPtr(true),
		CacheBoundary: boolPtr(false),
	}
	counter, ok := tokenCounterOf(t, f)
	if !ok {
		t.Fatal("tokenCounterOf reports the declining stub does not satisfy ai.TokenCounter, want it to (it implements CountTokens)")
	}
	if _, err := counter.CountTokens(t.Context(), minimalRequest(t)); err == nil {
		t.Error("CountTokens returned nil error for a deliberately declining stub, want an error (S-CNF-041)")
	}
}

// NFR-CNF-E (this leaf's own item 4) — the reverse declaration/discovery
// mismatch (declared absent, provider satisfies ai.TokenCounter): the
// cross-check fails the entry rather than trusting the declaration
// silently into an absent record.
func TestConformanceCapabilities_ReverseTokenCountingMismatch_FailsEntryNeverPanics(t *testing.T) {
	defer requireNoPanicHere(t)

	f := Factory{
		New: func(_ testing.TB, scripts ...Script) ai.ModelProvider {
			return tokenCountingStub{Provider: NewProvider(scripts...)} // satisfies ai.TokenCounter
		},
		Reasoning:     boolPtr(false),
		TokenCounting: boolPtr(false), // declared NOT offered — the reverse contradiction
		CacheBoundary: boolPtr(false),
	}
	record := newCapabilityRecord(t.Name())
	probe := &probeTB{}
	crossCheckDeclaredOptionalCapabilities(probe, f, &record)

	if !probe.failed {
		t.Error("probe.failed = false, want true (reverse declaration/discovery contradiction, R-CNF-002)")
	}
	entry, ok := record.Entry(CapTokenCounting)
	if !ok || entry.Outcome != OutcomeFailed {
		t.Errorf("CapTokenCounting entry = %+v (ok=%v), want Outcome=%v", entry, ok, OutcomeFailed)
	}
}

// S-CNF-042 (offered half) — cacheBoundaryHonoringCase passes when driven
// against a factory declaring cache-boundary honoring offered.
func TestConformanceCapabilities_CacheBoundaryHonoringCase_PassesWhenDeclaredOffered(t *testing.T) {
	f := Factory{
		New: func(_ testing.TB, scripts ...Script) ai.ModelProvider {
			return NewProvider(scripts...)
		},
		Reasoning:     boolPtr(false),
		TokenCounting: boolPtr(false),
		CacheBoundary: boolPtr(true),
	}
	cacheBoundaryHonoringCase(t, f)
}

// S-CNF-042 (declared-absent half) — FakeFactory declares CacheBoundary
// not offered; the entry is absent with a reported skip, the runner's own
// generic mechanism, reconfirmed for this leaf's registered case.
func TestConformanceCapabilities_CacheBoundaryDeclaredAbsent_SkippedRecordedAbsent(t *testing.T) {
	f := FakeFactory()
	record := runConformanceCases(t, f, []conformanceCase{
		{name: "cache_boundary/honoring_is_consumer_visible", capability: CapCacheBoundary, run: cacheBoundaryHonoringCase},
	})
	entry, ok := record.Entry(CapCacheBoundary)
	if !ok || entry.Outcome != OutcomeAbsent {
		t.Errorf("CapCacheBoundary entry = %+v (ok=%v), want Outcome=%v (S-CNF-042)", entry, ok, OutcomeAbsent)
	}
}

// S-CNF-043 — all seven finish reasons pass; S-CNF-046 (standing) is
// covered by TestConformanceSkeleton_CompletionMetadataStanding_IsRequired
// (AI-23.1) against the identical CapCompletionMetadata key this case
// registers under (confirmed by grep below, not re-derived here).
func TestConformanceCapabilities_FinishReasonExhaustivenessCase_PassesAgainstFakeFactory(t *testing.T) {
	finishReasonExhaustivenessCase(t, FakeFactory())
}

// S-CNF-044 — the drift guard fires in both directions against an
// artificially shrunk (simulating a removed value) or grown (simulating an
// added, eighth value) hand-list. A real eighth ai.FinishReason cannot be
// constructed — the vocabulary is closed — so this is the pure, permanent
// proof the guard mechanism itself has teeth.
func TestFinishReasonDriftGuardAgainst_ShrunkOrGrownList_FailsInBothDirections(t *testing.T) {
	t.Run("shrunk (simulates a removed value)", func(t *testing.T) {
		shrunk := handListedFinishReasons[:len(handListedFinishReasons)-1]
		if _, matches := finishReasonDriftGuardAgainst(shrunk); matches {
			t.Error("finishReasonDriftGuardAgainst(shrunk) reports matches=true, want false")
		}
	})

	t.Run("grown (simulates an added, eighth value)", func(t *testing.T) {
		grown := make([]ai.FinishReason, len(handListedFinishReasons)+1)
		copy(grown, handListedFinishReasons)
		grown[len(grown)-1] = ai.FinishReasonStop // any valid member repeated — only the COUNT matters to this guard
		if _, matches := finishReasonDriftGuardAgainst(grown); matches {
			t.Error("finishReasonDriftGuardAgainst(grown) reports matches=true, want false")
		}
	})

	t.Run("the real hand-list still matches (companion case)", func(t *testing.T) {
		if _, matches := finishReasonDriftGuardAgainst(handListedFinishReasons); !matches {
			t.Error("finishReasonDriftGuardAgainst(handListedFinishReasons) reports matches=false, want true")
		}
	})
}

// S-CNF-045 — usageAbsentVsZeroCase passes; S-CNF-046 (standing) shares
// TestConformanceSkeleton_CompletionMetadataStanding_IsRequired's coverage,
// same as the finish-reason case above.
func TestConformanceCapabilities_UsageAbsentVsZeroCase_PassesAgainstFakeFactory(t *testing.T) {
	usageAbsentVsZeroCase(t, FakeFactory())
}

// === AI-23.6 — the capability record: totality, verdict, comparison (R-CNF-017, R-CNF-018) ===

// S-CNF-047 — any completed run's record carries exactly nine entries,
// one per capability in the two closed lists, each naming its capability,
// standing and outcome.
func TestCapabilityRecord_Totality_ExactlyNineEntriesEachNamingAllThreeFields(t *testing.T) {
	record := runConformanceCases(t, FakeFactory(), nil)
	entries := record.Entries()
	if len(entries) != 9 {
		t.Fatalf("len(entries) = %d, want 9 (S-CNF-047)", len(entries))
	}
	seen := make(map[Capability]bool, 9)
	for _, e := range entries {
		if seen[e.Capability] {
			t.Errorf("capability %v appears more than once in the record, want exactly one entry each", e.Capability)
		}
		seen[e.Capability] = true
		if e.Capability == 0 {
			t.Error("an entry names the zero Capability value, want a real one")
		}
		if e.Standing != StandingRequired && e.Standing != StandingOptional {
			t.Errorf("capability %v standing = %v, want a real Standing value", e.Capability, e.Standing)
		}
	}
	for _, c := range Capabilities() {
		if !seen[c] {
			t.Errorf("capability %v has no entry, want the record total over Capabilities() (S-CNF-047)", c)
		}
	}
}

// S-CNF-048 — a run whose subject offers no optional capability: the three
// optional entries are absent, none not-exercised.
func TestCapabilityRecord_NoOptionalCapabilityOffered_FourAbsentNoneNotExercised(t *testing.T) {
	f := Factory{
		New:           func(_ testing.TB, scripts ...Script) ai.ModelProvider { return NewProvider(scripts...) },
		Reasoning:     boolPtr(false),
		TokenCounting: boolPtr(false),
		CacheBoundary: boolPtr(false),
		Retry:         boolPtr(false),
	}
	record := runConformanceCases(t, f, nil)
	for _, c := range []Capability{CapReasoningContent, CapTokenCounting, CapCacheBoundary, CapRetry} {
		entry, ok := record.Entry(c)
		if !ok || entry.Outcome != OutcomeAbsent {
			t.Errorf("%v entry = %+v (ok=%v), want Outcome=%v (S-CNF-048)", c, entry, ok, OutcomeAbsent)
		}
	}
}

// S-CNF-049 — a run in which a required capability's cases failed: that
// entry is failed, never absent (absent has no legal reading for a
// required standing). Proven directly against the record's own mutation
// method (recordCaseResult) rather than by running a case designed to
// fail through a real *testing.T — the same propagation reason recorded
// throughout this file: a case that genuinely calls t.Error would mark
// this meta-test itself failed too.
func TestCapabilityRecord_RequiredCapabilityCasesFailed_EntryFailedNeverAbsent(t *testing.T) {
	record := newCapabilityRecord(t.Name())
	record.recordCaseResult(CapToolCalls, false) // simulates: this required capability's case(s) failed

	entry, ok := record.Entry(CapToolCalls)
	if !ok || entry.Outcome != OutcomeFailed {
		t.Errorf("CapToolCalls entry = %+v (ok=%v), want Outcome=%v (S-CNF-049)", entry, ok, OutcomeFailed)
	}
}

// S-CNF-050 — a run cannot record a required capability as optional:
// standing is not the run's to supply. Proven structurally, not by
// runtime validation: Standing is set exactly once, in newCapabilityRecord,
// derived from standingOf(Capability) alone, and no other function in this
// package assigns to a CapabilityRecordEntry's Standing field at all —
// confirmed by grep, cited in tasks.md, since there is no code path left
// to exercise at runtime.
func TestCapabilityRecord_StandingIsNeverSuppliedByARun_StructuralByGrep(t *testing.T) {
	// A living regression proof of the same fact: the standing this suite
	// computes for every capability always equals standingOf(c) — the one
	// function derived purely from Capability.Optional() — regardless of
	// what any case does.
	record := runConformanceCases(t, FakeFactory(), []conformanceCase{
		{name: "tool_calls/probe", capability: CapToolCalls, run: func(_ *testing.T, _ Factory) {}},
	})
	for _, e := range record.Entries() {
		if e.Standing != standingOf(e.Capability) {
			t.Errorf("%v standing = %v, want %v (standingOf, the only source, S-CNF-050)", e.Capability, e.Standing, standingOf(e.Capability))
		}
	}
}

// S-CNF-051 — a published record carries no capability-specific detail,
// model content or credentials: proven by CapabilityRecordEntry's own
// shape — exactly three fields, all closed enums, via reflection, so a
// future field addition is caught here rather than only by review.
func TestCapabilityRecordEntry_ExportedShape_CarriesOnlyCapabilityStandingOutcome(t *testing.T) {
	typ := reflect.TypeOf(CapabilityRecordEntry{})
	if got := typ.NumField(); got != 3 {
		t.Fatalf("CapabilityRecordEntry has %d field(s), want exactly 3 (S-CNF-051)", got)
	}
	wantFields := map[string]bool{"Capability": true, "Standing": true, "Outcome": true}
	for i := range typ.NumField() {
		name := typ.Field(i).Name
		if !wantFields[name] {
			t.Errorf("CapabilityRecordEntry carries unexpected field %q, want only Capability/Standing/Outcome (S-CNF-051)", name)
		}
	}
}

// S-CNF-052 — every required entry satisfied, every optional satisfied or
// absent: pass.
func TestCapabilityRecord_Verdict_AllRequiredSatisfiedOptionalSatisfiedOrAbsent_Pass(t *testing.T) {
	f := Factory{
		New:           func(_ testing.TB, scripts ...Script) ai.ModelProvider { return NewProvider(scripts...) },
		Reasoning:     boolPtr(true),
		TokenCounting: boolPtr(false),
		CacheBoundary: boolPtr(false),
		Retry:         boolPtr(false),
	}
	var cases []conformanceCase
	for _, c := range Capabilities() {
		if c == CapReasoningContent {
			cases = append(cases, conformanceCase{name: "probe/" + c.String(), capability: c, run: func(_ *testing.T, _ Factory) {}})
			continue
		}
		if c.Optional() {
			continue // left declared false: applyDeclaredAbsences marks it absent
		}
		cases = append(cases, conformanceCase{name: "probe/" + c.String(), capability: c, run: func(_ *testing.T, _ Factory) {}})
	}
	record := runConformanceCases(t, f, cases)
	if got := record.Verdict(); got != VerdictPass {
		t.Errorf("Verdict() = %v, want %v (S-CNF-052)", got, VerdictPass)
	}
}

// S-CNF-053 — one not-exercised entry, rest satisfied: inconclusive,
// neither pass nor failure.
func TestCapabilityRecord_Verdict_OneNotExercised_Inconclusive(t *testing.T) {
	f := Factory{
		New:           func(_ testing.TB, scripts ...Script) ai.ModelProvider { return NewProvider(scripts...) },
		Reasoning:     boolPtr(false),
		TokenCounting: boolPtr(false),
		CacheBoundary: boolPtr(false),
		Retry:         boolPtr(false),
	}
	// No cases registered for any required capability: every required
	// entry stays not-exercised; every optional entry is absent.
	record := runConformanceCases(t, f, nil)
	if got := record.Verdict(); got != VerdictInconclusive {
		t.Errorf("Verdict() = %v, want %v (S-CNF-053)", got, VerdictInconclusive)
	}
}

// S-CNF-054 — one failed required entry: failure, and no optional result
// offsets it. Built directly on the record's own mutation methods, for the
// same propagation reason S-CNF-049's test above states — a case designed
// to fail via a real *testing.T would mark this meta-test failed too.
func TestCapabilityRecord_Verdict_OneFailedRequiredEntry_FailureNeverOffset(t *testing.T) {
	record := newCapabilityRecord(t.Name())
	for _, c := range Capabilities() {
		switch {
		case c == CapToolCalls:
			record.recordCaseResult(c, false) // the one required failure
		case c == CapReasoningContent:
			record.recordCaseResult(c, true) // an optional entry satisfied — must not offset the failure above
		case c.Optional():
			record.setOutcome(c, OutcomeAbsent)
		default:
			record.recordCaseResult(c, true) // every other required entry satisfied
		}
	}

	if got := record.Verdict(); got != VerdictFail {
		t.Errorf("Verdict() = %v, want %v — an optional satisfied entry must not offset a required failure (S-CNF-054)", got, VerdictFail)
	}
}

// S-CNF-055 — two records for the same subject differing on one entry:
// the comparison names that entry and the direction of the difference.
func TestCompareCapabilityRecords_OneDifferingEntry_NamesItAndTheDirection(t *testing.T) {
	want := newCapabilityRecord("subject")
	want.setOutcome(CapToolCalls, OutcomeSatisfied)

	got := newCapabilityRecord("subject")
	got.setOutcome(CapToolCalls, OutcomeFailed)

	diffs := CompareCapabilityRecords(got, want)
	if len(diffs) != 1 {
		t.Fatalf("len(diffs) = %d, want 1 (S-CNF-055): %v", len(diffs), diffs)
	}
	d := diffs[0]
	if d.Capability != CapToolCalls {
		t.Errorf("diff capability = %v, want %v", d.Capability, CapToolCalls)
	}
	if d.Got != OutcomeFailed || d.Want != OutcomeSatisfied {
		t.Errorf("diff = (got=%v, want=%v), want (got=%v, want=%v) — the direction of the difference", d.Got, d.Want, OutcomeFailed, OutcomeSatisfied)
	}

	// Companion: identical records compare with no differences.
	if diffs := CompareCapabilityRecords(want, want); len(diffs) != 0 {
		t.Errorf("CompareCapabilityRecords(want, want) = %v, want none", diffs)
	}
}

// S-CNF-056 — AI-21's fake as the first subject, the whole suite run end
// to end: the verdict is a pass and every required entry is satisfied.
// This is the milestone's own capstone proof: every leaf's registered
// cases, run together through the public RunConformance door, against a
// real scripted subject.
func TestRunConformance_FakeFactoryEndToEnd_VerdictPassEveryRequiredSatisfied(t *testing.T) {
	record := RunConformance(t, FakeFactory())

	if got := record.Verdict(); got != VerdictPass {
		t.Fatalf("Verdict() = %v, want %v (S-CNF-056). Entries: %v", got, VerdictPass, record.Entries())
	}
	for _, e := range record.Entries() {
		if e.Standing != StandingRequired {
			continue
		}
		if e.Outcome != OutcomeSatisfied {
			t.Errorf("required capability %v outcome = %v, want %v", e.Capability, e.Outcome, OutcomeSatisfied)
		}
	}
	// FakeFactory declares Reasoning offered, TokenCounting,
	// CacheBoundary, and Retry not — so the record exercises both
	// optional outcomes (design.md's own stated purpose for
	// FakeFactory's declarations).
	if entry, ok := record.Entry(CapReasoningContent); !ok || entry.Outcome != OutcomeSatisfied {
		t.Errorf("CapReasoningContent entry = %+v (ok=%v), want Outcome=%v", entry, ok, OutcomeSatisfied)
	}
	if entry, ok := record.Entry(CapTokenCounting); !ok || entry.Outcome != OutcomeAbsent {
		t.Errorf("CapTokenCounting entry = %+v (ok=%v), want Outcome=%v", entry, ok, OutcomeAbsent)
	}
	if entry, ok := record.Entry(CapCacheBoundary); !ok || entry.Outcome != OutcomeAbsent {
		t.Errorf("CapCacheBoundary entry = %+v (ok=%v), want Outcome=%v", entry, ok, OutcomeAbsent)
	}
	if entry, ok := record.Entry(CapRetry); !ok || entry.Outcome != OutcomeAbsent {
		t.Errorf("CapRetry entry = %+v (ok=%v), want Outcome=%v", entry, ok, OutcomeAbsent)
	}
}

// NFR-CNF-E (this leaf's own item 4) — Verdict() and
// CompareCapabilityRecords given a zero-value, never-constructed
// CapabilityRecord: attributable (never silently a pass), never a panic.
func TestCapabilityRecord_ZeroValue_VerdictNeverSilentlyPassesNeverPanics(t *testing.T) {
	defer requireNoPanicHere(t)

	var zero CapabilityRecord
	if got := zero.Verdict(); got == VerdictPass {
		t.Error("Verdict() on a zero-value CapabilityRecord reports pass, want it to never silently pass an uninitialized record (NFR-CNF-E)")
	}

	diffs := CompareCapabilityRecords(zero, newCapabilityRecord("want"))
	if len(diffs) == 0 {
		t.Error("CompareCapabilityRecords(zero, a real record) reports no differences, want every capability to differ")
	}
}
