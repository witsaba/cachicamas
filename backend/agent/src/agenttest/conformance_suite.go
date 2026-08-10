// AI-23.1 — the provider conformance suite's pluggable skeleton: Factory,
// Capability, the case table and the runner every other conformance_*.go
// leaf registers cases into (R-CNF-001…R-CNF-004).
//
// # One factory seam, zero copied assertions (R-CNF-001)
//
// [RunConformance] is the suite's one exported entry point: it accepts a
// caller-supplied [Factory], runs the whole registered case table against
// the subject it builds, and returns AI-03 §10's [CapabilityRecord]. The
// caller writes no assertion of its own — every case delegates its
// mechanics to AI-22's stream test kit or to [ai.CheckStream] (NFR-CNF-D).
//
// # The case table (design's own keying-mechanism choice)
//
// Each conformance_*.go leaf registers its cases into this file's
// package-level registry from its own init() function, so a leaf's whole
// contribution — cases and their registration alike — lives in one file and
// reverts with one commit (this milestone's tasks.md "Suggested Work
// Units" rollback boundary). [RunConformance] runs the registry;
// runConformanceCases is the same mechanism parameterized over an explicit
// case list, so this package's own white-box tests can exercise the runner
// against a hand-built list without depending on how many leaves have
// landed.

package agenttest

import (
	"fmt"
	"testing"

	"github.com/cachicamas/backend/agent/src/ai"
)

// Capability identifies one of the nine capabilities in the two closed
// lists — CAP-R-01…05 and CAP-O-01…03 from AI-03, CAP-O-04 added by AI-35
// under AI-03 §13 rule 1 — plus [CapNone] for a case that
// exercises none of them. The zero value is not a member — this package's
// zero-names-nothing idiom, restated: a [conformanceCase] whose author
// forgot to set its capability field is caught as a registration defect
// (R-CNF-003's default-to-required rule, S-CNF-009) rather than silently
// reading as a legitimate structural case.
type Capability uint8

const (
	_ Capability = iota // the zero value names no capability at all

	// CapStreamingText is CAP-R-01: text arrives as delimited blocks,
	// reconstructible exactly.
	CapStreamingText

	// CapToolCalls is CAP-R-02: a tool call arrives with identity, name,
	// exact argument bytes and an observable ordinal.
	CapToolCalls

	// CapCompletionMetadata is CAP-R-03: a normally-finished stream ends
	// carrying a finish reason and a usage record.
	CapCompletionMetadata

	// CapCancellation is CAP-R-04: the caller-owned signal ends the stream
	// within bounded time.
	CapCancellation

	// CapTypedFailures is CAP-R-05: every failure is classifiable through
	// one vocabulary, on both delivery paths, and states whether content
	// preceded it.
	CapTypedFailures

	// CapReasoningContent is CAP-O-01: reasoning content, optional.
	CapReasoningContent

	// CapTokenCounting is CAP-O-02: token counting, optional.
	CapTokenCounting

	// CapCacheBoundary is CAP-O-03: honoring cache-boundary markers,
	// optional.
	CapCacheBoundary

	// CapRetry is CAP-O-04: auto-retry of retryable pre-stream failures,
	// optional.
	CapRetry

	// CapNone marks a conformance case that exercises no capability in
	// either closed list — a structural or lifecycle case (AI-03 §5's
	// closing note: exactly-one-terminal and redaction are both required
	// "by the default", never by a record entry). It is deliberately
	// declared after the nine real capabilities, outside
	// [capabilityFirst]…[capabilityEnd]'s range, so it can never be mistaken
	// for one of them and never receives a capability-record entry
	// (R-CNF-017's totality is over the two closed lists only).
	CapNone

	capabilityFirst = CapStreamingText
	capabilityEnd   = CapRetry + 1
)

// capabilityNames is the stable string form of each capability, named with
// its AI-03 identifier so an attributable failure message is self-citing.
var capabilityNames = [...]string{
	CapStreamingText:      "CAP-R-01(streaming_text)",
	CapToolCalls:          "CAP-R-02(tool_calls)",
	CapCompletionMetadata: "CAP-R-03(completion_metadata)",
	CapCancellation:       "CAP-R-04(cancellation)",
	CapTypedFailures:      "CAP-R-05(typed_failures)",
	CapReasoningContent:   "CAP-O-01(reasoning_content)",
	CapTokenCounting:      "CAP-O-02(token_counting)",
	CapCacheBoundary:      "CAP-O-03(cache_boundary)",
	CapRetry:              "CAP-O-04(retry)",
	CapNone:               "no_listed_capability",
}

// String renders the capability, or "capability(N)" for a value outside
// this vocabulary — a shape no parser accepts.
func (c Capability) String() string {
	if int(c) < len(capabilityNames) && capabilityNames[c] != "" {
		return capabilityNames[c]
	}
	return fmt.Sprintf("capability(%d)", uint8(c))
}

// Capabilities returns the nine-member closed-list union (AI-03's eight
// plus AI-35's CAP-O-04) — CAP-R-01…05
// then CAP-O-01…04, in declaration order — the enumerator [CapabilityRecord]
// totality is built from. [CapNone] is deliberately excluded: it is not a
// member of either closed list. The result is a fresh slice on every call —
// this package's own closed-vocabulary idiom (EventKinds, FailureCategories).
func Capabilities() []Capability {
	out := make([]Capability, 0, int(capabilityEnd-capabilityFirst))
	for c := capabilityFirst; c < capabilityEnd; c++ {
		out = append(out, c)
	}
	return out
}

// Optional reports whether c is one of CAP-O-01…04 — AI-03 §11's marking
// rule as amended for CAP-O-04 (AI-35), applied mechanically: every other
// value, including [CapNone], the
// zero value and a value outside this vocabulary entirely, is required by
// the same default.
func (c Capability) Optional() bool {
	switch c {
	case CapReasoningContent, CapTokenCounting, CapCacheBoundary, CapRetry:
		return true
	default:
		return false
	}
}

// registered reports whether c is a legitimate value this suite recognises:
// one of the nine real capabilities, or [CapNone]. Anything else —
// including the zero value — is a case-registration defect (S-CNF-009).
func (c Capability) registered() bool {
	return c == CapNone || (c >= capabilityFirst && c < capabilityEnd)
}

// standingOf derives a capability's standing purely from Capability.Optional
// (R-CNF-003): never from a case's node structure, never from the run.
func standingOf(c Capability) Standing {
	if c.Optional() {
		return StandingOptional
	}
	return StandingRequired
}

// Factory is the caller-supplied seam AI-23's suite runs against (R-CNF-001,
// R-CNF-002, design.md D2). It carries both halves R-CNF-002 requires
// together: the constructor that turns a suite-authored [Script] into a live
// subject, and that subject's declared optional-capability expectation.
type Factory struct {
	// New builds a fresh subject per case (never shared across cases,
	// R-AFP-003's isolation restated at the suite boundary), scripted with
	// AI-21's own Script/Step/Emit/Hold/Gate vocabulary.
	New func(tb testing.TB, scripts ...Script) ai.ModelProvider

	// Reasoning, TokenCounting, CacheBoundary and Retry are the factory's
	// declared expectation for CAP-O-01, CAP-O-02, CAP-O-03 and CAP-O-04
	// respectively (R-CNF-002). nil means undeclared: the suite fails
	// construction, naming the undeclared capability (S-CNF-006). Non-nil
	// false means declared not offered: the suite records the entry absent
	// and skips its cases, reported rather than silent (R-CNF-004). Non-nil
	// true means declared offered: the suite runs the capability's cases
	// normally, and for CAP-O-02 — the only askable optional contract,
	// R-AMP-017 — additionally cross-checks the declaration against the
	// subject's own ai.TokenCounter discovery (design.md D6): a mismatch in
	// either direction fails the entry rather than trusting either side
	// silently.
	Reasoning, TokenCounting, CacheBoundary, Retry *bool

	// Sentinel is the redaction canary AI-23.7's cases plant into a
	// subject's failure-report fields. "" selects the suite's own default
	// (conformance_redaction.go).
	Sentinel string

	// Dialect declares wire-dialect expressiveness limits this subject's
	// factory asserts (design D5, AI-38): nil means fully expressive — the
	// suite's default reading, and where FakeFactory leaves it — so every
	// existing factory is unaffected by this field's addition. A
	// declaration here is proven by the amended case it narrows
	// (finishReasonExhaustivenessCase, terminalFailureCategoryExhaustivenessCase),
	// never trusted at face value: a value the dialect CAN express, or
	// produces as something other than the declared collapse, still fails
	// (S-CNF-084, S-CNF-087).
	Dialect *DialectConstraints
}

// DialectConstraints declares wire-dialect expressiveness limits a
// subject's Factory MAY assert (design D5, AI-38). The zero value declares
// nothing unreachable and no mid-stream collapse — identical to a nil
// *DialectConstraints for every case that reads it.
type DialectConstraints struct {
	// UnreachableFinishReasons lists finish reasons finishReasonExhaustivenessCase
	// must NOT expect reachable on this subject's normally-finished stream.
	// Each listed value instead asserts the negative: scripting it yields
	// exactly one typed failure terminal and no Completion — unsatisfiable
	// and unviolated (R-CNF-016, S-CNF-043). MUST be a subset of
	// handListedFinishReasons; the drift guard and the hand-list itself
	// are untouched by this field.
	UnreachableFinishReasons []ai.FinishReason

	// MidStreamCategoryCollapse, when non-nil, is the single failure
	// category terminalFailureCategoryExhaustivenessCase's mid-stream half
	// must observe for EVERY scripted category on this subject — a
	// dialect-aware collapse, recorded as such and never read as a skip or
	// as a pass of the original category (R-CNF-010, S-CNF-024). The
	// pre-stream half (pure construction, no wire involved) is unaffected.
	MidStreamCategoryCollapse *ai.FailureCategory
}

// FakeFactory wraps AI-21's [Provider] as a reference subject: New returns a
// fresh scripted [Provider] per case, and all four optional capabilities
// are explicitly declared — Reasoning offered, TokenCounting,
// CacheBoundary, and Retry not — so a full suite run against it exercises
// both the satisfied and the absent optional outcomes (S-CNF-056) and never
// trips S-CNF-006's undeclared-capability construction failure.
func FakeFactory() Factory {
	reasoningOffered, tokenCountingOffered, cacheBoundaryOffered, retryOffered := true, false, false, false
	return Factory{
		New: func(_ testing.TB, scripts ...Script) ai.ModelProvider {
			return NewProvider(scripts...)
		},
		Reasoning:     &reasoningOffered,
		TokenCounting: &tokenCountingOffered,
		CacheBoundary: &cacheBoundaryOffered,
		Retry:         &retryOffered,
	}
}

// conformanceCase is one independently verifiable conformance assertion,
// keyed by the single capability it exercises (AI-03 §11). name is the
// t.Run subtest path other leaves' registrations choose so that, for
// example, `go test -run 'TestConformance/text'` selects exactly one
// leaf's cases — this suite's own concrete answer to spec.md's "what this
// file deliberately leaves to design" item 2.
type conformanceCase struct {
	name       string
	capability Capability
	run        func(t *testing.T, f Factory)
}

// conformanceRegistry is the whole case table: every conformance_*.go leaf
// but this one appends to it from its own init() function. RunConformance
// runs it; this package's white-box tests instead call
// runConformanceCases directly with a hand-built list, so a unit test of
// the runner's own mechanics never depends on how many leaves have landed.
var conformanceRegistry []conformanceCase

// registerConformanceCase appends one case to the suite's case table. It is
// called only from a conformance_*.go leaf's own init() function.
func registerConformanceCase(name string, capability Capability, run func(t *testing.T, f Factory)) {
	conformanceRegistry = append(conformanceRegistry, conformanceCase{name: name, capability: capability, run: run})
}

// RegisterConformanceCase is the exported twin of [registerConformanceCase]
// used by external test packages (notably [openaicompat.conformance_retry])
// to register conformance cases from outside the agenttest tree. The
// unexported sibling stays in place so the package-internal conformance
// leaves keep their self-registration pattern; the exported twin is
// reserved for adapter-specific cases whose test code lives in the
// adapter's own package (which already imports [agenttest] for the
// stream test kit).
func RegisterConformanceCase(name string, capability Capability, run func(t *testing.T, f Factory)) {
	registerConformanceCase(name, capability, run)
}

// RunConformance runs the suite's whole registered case table against the
// subject f builds, and returns AI-03 §10's total capability record
// (R-CNF-001). See the package doc for what "zero copied assertions" means
// in practice, and Factory's own doc for the declared-capability contract
// this call enforces before any case runs.
func RunConformance(t *testing.T, f Factory) CapabilityRecord {
	t.Helper()
	return runConformanceCases(t, f, conformanceRegistry)
}

// runConformanceCases is RunConformance's mechanism, parameterized over an
// explicit case list rather than the package registry (this file's own
// case-table design choice, see the package doc).
func runConformanceCases(t *testing.T, f Factory, cases []conformanceCase) CapabilityRecord {
	t.Helper()

	requireValidFactory(t, f)

	record := newCapabilityRecord(t.Name())
	applyDeclaredAbsences(f, &record)
	crossCheckDeclaredOptionalCapabilities(t, f, &record)

	for _, c := range cases {
		runRegisteredCase(t, f, c, &record)
	}

	return record
}

// factoryDefect reports the first reason f cannot run the suite at all
// (NFR-CNF-E) or leaves an optional capability undeclared (S-CNF-006), or ""
// when f is usable. It is a pure function — no testing.TB involved — so this
// package's own tests can assert on the exact message without driving any
// real subtest (design note: Go's testing package propagates a failed
// subtest to every ancestor unconditionally, which would make a literal
// "does this call correctly fail a *testing.T" test also fail the meta-test
// observing it; keeping the decision pure sidesteps that entirely).
func factoryDefect(f Factory) string {
	if f.New == nil {
		return "agenttest: RunConformance given a Factory with a nil New constructor — want a subject builder (NFR-CNF-E)"
	}
	if f.Reasoning == nil {
		return fmt.Sprintf("agenttest: RunConformance given a Factory that never declared %v (nil *bool) — want an explicit true/false declaration (R-CNF-002, S-CNF-006)", CapReasoningContent)
	}
	if f.TokenCounting == nil {
		return fmt.Sprintf("agenttest: RunConformance given a Factory that never declared %v (nil *bool) — want an explicit true/false declaration (R-CNF-002, S-CNF-006)", CapTokenCounting)
	}
	if f.CacheBoundary == nil {
		return fmt.Sprintf("agenttest: RunConformance given a Factory that never declared %v (nil *bool) — want an explicit true/false declaration (R-CNF-002, S-CNF-006)", CapCacheBoundary)
	}
	if f.Retry == nil {
		return fmt.Sprintf("agenttest: RunConformance given a Factory that never declared %v (nil *bool) — want an explicit true/false declaration (R-CNF-002, S-CNF-006)", CapRetry)
	}
	return ""
}

// requireValidFactory fails tb, naming the specific missing piece, when
// factoryDefect finds one. It takes testing.TB rather than *testing.T
// (matching AI-22's own kit functions, e.g. DrainAndRecord) precisely so a
// test can substitute a capturing double for tb without touching a real
// *testing.T tree. Every real call passes a *testing.T, which satisfies
// testing.TB, so RunConformance's own call site needs no change.
func requireValidFactory(tb testing.TB, f Factory) {
	tb.Helper()
	if msg := factoryDefect(f); msg != "" {
		tb.Fatalf("%s", msg)
	}
}

// FactoryDefectForTest is the exported twin of [factoryDefect], used by
// external test packages (notably [openaicompat.conformance_retry]) that
// need to assert on the exact defect message without driving a real
// *testing.T subtest. Its semantics are identical to the unexported
// sibling; the only difference is the export, reserved for adapter-
// specific conformance cases whose test code lives in another package.
func FactoryDefectForTest(f Factory) string {
	return factoryDefect(f)
}

// declaredOffered reports whether f declares c offered — true only for a
// non-nil, true *bool. Called only after requireValidFactory has already
// confirmed every optional capability's *bool is non-nil, so dereferencing
// here is safe; a capability outside the four optional ones (including
// CapNone) always reports false, which is never consulted for a required
// capability's standing (Optional() gates every caller of this function).
func declaredOffered(f Factory, c Capability) bool {
	switch c {
	case CapReasoningContent:
		return f.Reasoning != nil && *f.Reasoning
	case CapTokenCounting:
		return f.TokenCounting != nil && *f.TokenCounting
	case CapCacheBoundary:
		return f.CacheBoundary != nil && *f.CacheBoundary
	case CapRetry:
		return f.Retry != nil && *f.Retry
	default:
		return false
	}
}

// optionalCapabilities returns CAP-O-01…04 in Capabilities()' order — the
// four-member subset every declared-capability decision iterates over.
func optionalCapabilities() []Capability {
	out := make([]Capability, 0, 4)
	for _, c := range Capabilities() {
		if c.Optional() {
			out = append(out, c)
		}
	}
	return out
}

// applyDeclaredAbsences marks every optional capability the factory
// declares not offered as OutcomeAbsent up front — a conclusion the record
// carries regardless of whether any case for that capability is registered
// yet (S-CNF-004): absence must never wait on a case existing to report it.
func applyDeclaredAbsences(f Factory, record *CapabilityRecord) {
	for _, c := range optionalCapabilities() {
		if !declaredOffered(f, c) {
			record.setOutcome(c, OutcomeAbsent)
		}
	}
}

// applyTokenCountingCrossCheck folds design.md D6's cross-check result into
// record: a subject that does not satisfy ai.TokenCounter despite a
// declared-true Factory.TokenCounting fails the entry (S-CNF-005). Kept pure
// (no testing.TB) so it is directly testable without touching a real
// *testing.T tree.
func applyTokenCountingCrossCheck(record *CapabilityRecord, satisfiesTokenCounter bool) {
	if !satisfiesTokenCounter {
		record.setOutcome(CapTokenCounting, OutcomeFailed)
	}
}

// crossCheckDeclaredOptionalCapabilities runs design.md D6's declaration
// versus askable-discovery cross-check for CAP-O-02, the only capability
// R-AMP-017 makes askable in v1. A declared-true TokenCounting whose subject
// does not satisfy ai.TokenCounter fails the entry immediately, naming the
// contradiction (S-CNF-005), independent of whether any CAP-O-02 case is
// registered yet. CAP-O-01, CAP-O-03 and CAP-O-04 have no askable seam and
// are not checked here; their outcome comes entirely from their own cases running
// (or being skipped, when declared absent). tb is testing.TB for the same
// substitutability reason requireValidFactory takes it.
func crossCheckDeclaredOptionalCapabilities(tb testing.TB, f Factory, record *CapabilityRecord) {
	tb.Helper()
	if f.TokenCounting == nil {
		return // requireValidFactory already failed construction for this case
	}

	if *f.TokenCounting {
		_, satisfies := tokenCounterOf(tb, f)
		applyTokenCountingCrossCheck(record, satisfies)
		if !satisfies {
			tb.Errorf("agenttest: factory declares %v offered (Factory.TokenCounting=true), but the built subject does not satisfy ai.TokenCounter — declaration/discovery contradiction (R-CNF-002, S-CNF-005)", CapTokenCounting)
		}
		return
	}

	// R-CNF-002's own "or the reverse": a factory declaring CAP-O-02 NOT
	// offered whose subject nonetheless satisfies ai.TokenCounter is the
	// same contradiction in the other direction (AI-23.8's own
	// NFR-CNF-E item, S-CNF-066 partial) — never trusted silently into an
	// absent entry applyDeclaredAbsences already recorded.
	if _, satisfies := tokenCounterOf(tb, f); satisfies {
		record.setOutcome(CapTokenCounting, OutcomeFailed)
		tb.Errorf("agenttest: factory declares %v not offered (Factory.TokenCounting=false), but the built subject DOES satisfy ai.TokenCounter — declaration/discovery contradiction (R-CNF-002)", CapTokenCounting)
	}
}

// tokenCounterOf builds a fresh subject via f.New and reports it as an
// ai.TokenCounter, and whether it satisfies that contract at all — the one
// place both this file's construction-time cross-check and AI-23.8's own
// CAP-O-02 case (R-CNF-015) perform the type assertion, so no duplicate
// logic exists between them.
func tokenCounterOf(tb testing.TB, f Factory) (ai.TokenCounter, bool) {
	tb.Helper()
	subject := f.New(tb)
	counter, ok := subject.(ai.TokenCounter)
	return counter, ok
}

// runRegisteredCase runs one case as its own subtest and folds the result
// into record: a case keyed to an unregistered capability fails loudly,
// naming it (S-CNF-009); a case whose capability is optional and declared
// absent is skipped before its body ever runs, reported rather than silent
// (R-CNF-004); every other case runs normally, with its own panic recovered
// and converted into an attributable failure so it never aborts the rest of
// the run (R-CNF-001), and a required-standing case that skips itself is
// rejected — there are no waivers (R-CNF-004).
func runRegisteredCase(t *testing.T, f Factory, c conformanceCase, record *CapabilityRecord) {
	t.Helper()

	if !c.capability.registered() {
		t.Run(c.name, func(t *testing.T) {
			t.Fatalf("agenttest: conformance case %q is keyed to capability %v, outside both the CAP-R and CAP-O closed lists — defaulting to required and failing (R-CNF-003, S-CNF-009)", c.name, c.capability)
		})
		return
	}

	if skip, reason := declaredAbsentSkipReason(f, c.capability); skip {
		t.Run(c.name, func(t *testing.T) {
			t.Skipf("%s", reason)
		})
		record.setOutcome(c.capability, OutcomeAbsent)
		return
	}

	ok, skipped := runOneCase(t, f, c)
	if skipped {
		if !c.capability.Optional() {
			t.Errorf("agenttest: conformance case %q attempted to skip required capability %v — there are no waivers (R-CNF-004, S-CNF-011)", c.name, c.capability)
			record.setOutcome(c.capability, OutcomeFailed)
			return
		}
		record.setOutcome(c.capability, OutcomeAbsent)
		return
	}
	record.recordCaseResult(c.capability, ok)
}

// declaredAbsentSkipReason reports whether c's cases must be skipped
// because the factory declared it not offered, and the message to report
// the skip with. It is deliberately keyed off Capability.Optional() first:
// a required capability is never skippable for any reason (R-CNF-004's
// no-waivers rule), regardless of what a hypothetical caller passed. Pure —
// no testing.TB involved.
func declaredAbsentSkipReason(f Factory, c Capability) (bool, string) {
	if !c.Optional() {
		return false, ""
	}
	if declaredOffered(f, c) {
		return false, ""
	}
	return true, fmt.Sprintf("agenttest: capability %v declared not offered by the factory — case skipped, recorded absent (R-CNF-002, R-CNF-004)", c)
}

// invokeCaseRecovering calls run, recovering any panic rather than letting
// it escape, and reports whether one occurred and its value. It is the pure
// core of R-CNF-001's "one case's panic never aborts the run" — kept free
// of testing.T so this package's own tests can exercise recovery directly,
// without relying on Go's real subtest machinery (which would propagate a
// deliberately-panicking probe's failure to every ancestor test).
func invokeCaseRecovering(run func()) (panicked bool, value any) {
	defer func() {
		if r := recover(); r != nil {
			panicked, value = true, r
		}
	}()
	run()
	return false, nil
}

// runOneCase runs c.run as a t.Run subtest, recovering any panic (via
// invokeCaseRecovering) into an attributable failure naming the case and
// its capability (R-CNF-001) so one case can never abort the rest of the
// run. It reports whether the subtest passed and whether it was skipped —
// Go's testing package considers a skipped subtest passing, so the caller
// reads both results to tell "satisfied" apart from "skipped" (R-CNF-004).
func runOneCase(t *testing.T, f Factory, c conformanceCase) (ok bool, skipped bool) {
	t.Helper()
	ok = t.Run(c.name, func(t *testing.T) {
		t.Helper()
		panicked, value := invokeCaseRecovering(func() { c.run(t, f) })
		skipped = t.Skipped()
		if panicked {
			t.Fatalf("agenttest: conformance case %q (capability=%v) panicked: %v (R-CNF-001)", c.name, c.capability, value)
		}
	})
	return ok, skipped
}

// --- Shared case-authoring helpers (used by every conformance_*.go leaf) ---

// requireConstructed fails tb, naming the constructor, when err is
// non-nil — AI-21's own posture restated: a script may only ever carry an
// event that already came from one of ai's own constructors, so every case
// in this suite that hand-builds a scripted event guards its construction
// the same way rather than re-deriving the check per file. Called
// immediately after each two-result ai.NewXxx call (Go does not allow
// forwarding a multi-valued call alongside another argument in the same
// expression, so this is deliberately two statements at each call site
// rather than a single wrapping call).
func requireConstructed(tb testing.TB, err error, constructor string) {
	tb.Helper()
	if err != nil {
		tb.Fatalf("agenttest: %s returned %v, want no failure", constructor, err)
	}
}

// minimalRequest builds the smallest valid ai.Request a case needs to call
// Stream with — provider_test.go's own mustSimpleRequest precedent,
// restated here since that helper lives in the sibling agenttest_test
// package and is unreachable from this internal-package file.
func minimalRequest(tb testing.TB) ai.Request {
	tb.Helper()
	part, err := ai.NewText("conformance suite probe")
	if err != nil {
		tb.Fatalf("agenttest: ai.NewText returned %v, want no failure", err)
	}
	message, err := ai.NewMessage(ai.RoleUser, part)
	if err != nil {
		tb.Fatalf("agenttest: ai.NewMessage returned %v, want no failure", err)
	}
	request, err := ai.NewRequest("cachicamas-conformance-suite-model", []ai.Message{message})
	if err != nil {
		tb.Fatalf("agenttest: ai.NewRequest returned %v, want no failure", err)
	}
	return request
}
