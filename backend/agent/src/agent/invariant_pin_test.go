// Tests for AG-04.3 — the two invariant pins AG-04 co-owns (R-AEV-007, no
// delta-to-accumulated-payload route; R-AEV-008, typed failure) and the
// ordering-invariants/membership-criterion documentation pin (R-AEV-011,
// S-AEV-102). package agent_test: NFR-AEV-001.
//
// Non-claim, restated per risk 4 (0003:2203): nothing in this file asserts
// that AG-04 alone closes envelope invariant 1 or invariant 4. Invariant 1
// closes jointly with AG-05.1; invariant 4 closes jointly with AG-11.2.
package agent_test

import (
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"strings"
	"testing"

	"github.com/cachicamas/backend/agent/src/agent"
	"github.com/cachicamas/backend/agent/src/ai"
)

// S-AEV-060 — every exported route from a delta kind to a payload is
// enumerated, and none accepts or produces an accumulated-message
// payload; the pin is asserted mechanically. AG-04 registers no delta
// kind at all (its four kinds are exactly the run/turn bracket family, no
// BracketRoleDelta-equivalent concept exists), so the mechanical pin is
// twofold: the registered kind set is exactly the four bracket kinds, and
// no exported declaration names a delta or accumulated-message concept.
func TestConstructionSurface_NoRouteFromDeltaKindToAccumulatedPayload(t *testing.T) {
	t.Parallel()

	kinds := agent.EventKinds()
	want := []agent.EventKind{
		agent.EventKindRunStart, agent.EventKindRunEnd,
		agent.EventKindTurnStart, agent.EventKindTurnEnd,
	}
	if len(kinds) != len(want) {
		t.Fatalf("agent.EventKinds() = %v (%d kinds), want exactly the 4 bracket kinds %v", kinds, len(kinds), want)
	}

	forbidden := []string{"Delta", "Accumulat", "Snapshot", "Message"}
	names := exportedPackageAgentNames(t)
	if len(names) == 0 {
		t.Fatal("exportedPackageAgentNames found zero declarations; the assertion would pass vacuously")
	}
	for _, name := range names {
		for _, f := range forbidden {
			if containsFold(name, f) {
				t.Errorf("exported declaration %q names a delta/accumulated-message concept (%q); AG-04 registers no delta kind and offers no route to one (R-AEV-007)", name, f)
			}
		}
	}
}

// S-AEV-061 — the package documentation states the delta rule: a delta
// carries an index and the new fragment only, and the absence of an
// accumulated-snapshot route is the mechanism AG-05 inherits.
func TestPackageDoc_StatesTheDeltaRule(t *testing.T) {
	t.Parallel()

	doc := packageFileDoc(t, "doc.go")
	if !containsFold(doc, "index and the new fragment only") {
		t.Errorf("doc.go's package documentation does not state that a delta carries an index and the new fragment only:\n%s", doc)
	}
	if !containsFold(doc, "AG-05 inherits") {
		t.Errorf("doc.go's package documentation does not state that the absent accumulated-snapshot route is the mechanism AG-05 inherits:\n%s", doc)
	}
}

// S-AEV-070 — a failure payload's category is reachable as a typed value
// and its cause is reachable as an inspectable error value, through the
// typed-failure surface.
func TestFailure_CategoryAndCause_ReachableAsTypedValues(t *testing.T) {
	t.Parallel()

	cause := os.ErrDeadlineExceeded // any inspectable error value, standing in for a real cause
	inner, err := ai.PreStreamFailure(ai.FailureReport{Category: ai.FailureCategoryTimeout, Cause: cause})
	if err != nil {
		t.Fatalf("ai.PreStreamFailure() error = %v, want nil", err)
	}
	failure, err := agent.NewFailure(inner)
	if err != nil {
		t.Fatalf("agent.NewFailure() error = %v, want nil", err)
	}

	if got := failure.Category(); got != ai.FailureCategoryTimeout {
		t.Errorf("failure.Category() = %v, want ai.FailureCategoryTimeout", got)
	}
	if got := failure.Unwrap(); got == nil {
		t.Error("failure.Unwrap() = nil, want the wrapped cause reachable as an inspectable error value")
	}
}

// S-AEV-071 — a failure constructed on the pre-stream path and one
// constructed on the mid-stream path are distinguishable through the
// typed surface without parsing any text.
func TestFailure_Delivery_DistinguishesPreStreamFromMidStream(t *testing.T) {
	t.Parallel()

	preInner, err := ai.PreStreamFailure(ai.FailureReport{Category: ai.FailureCategoryUnavailable})
	if err != nil {
		t.Fatalf("ai.PreStreamFailure() error = %v, want nil", err)
	}
	pre, err := agent.NewFailure(preInner)
	if err != nil {
		t.Fatalf("agent.NewFailure() error = %v, want nil", err)
	}

	midInner, err := ai.MidStreamFailure(ai.FailureReport{Category: ai.FailureCategoryUnavailable}, false)
	if err != nil {
		t.Fatalf("ai.MidStreamFailure() error = %v, want nil", err)
	}
	mid, err := agent.NewFailure(midInner)
	if err != nil {
		t.Fatalf("agent.NewFailure() error = %v, want nil", err)
	}

	if pre.Delivery() == mid.Delivery() {
		t.Fatalf("pre.Delivery() = %v and mid.Delivery() = %v, want distinguishable values", pre.Delivery(), mid.Delivery())
	}
	if got, want := pre.Delivery(), ai.DeliveryPreStream; got != want {
		t.Errorf("pre.Delivery() = %v, want %v", got, want)
	}
	if got, want := mid.Delivery(), ai.DeliveryMidStream; got != want {
		t.Errorf("mid.Delivery() = %v, want %v", got, want)
	}
}

// S-AEV-072 — every Layer 2 failure category maps to a stated
// ai.FailureCategory member, and the mapping is declared in source, not
// inferred. Layer 2 declares no category vocabulary of its own —
// Failure.Category's return type IS ai.FailureCategory (a thin wrap, per
// design AD-2), so the "mapping" is identity, declared in the accessor's
// own signature. Proven by reflection over the method's return type.
func TestFailure_CategoryMapping_IsIdentityDeclaredInSource(t *testing.T) {
	t.Parallel()

	method, ok := reflect.TypeOf(&agent.Failure{}).MethodByName("Category")
	if !ok {
		t.Fatal(`*agent.Failure has no "Category" method; the mapping this scenario checks would be unreachable`)
	}
	// Method.Type's first result is the receiver for a value obtained via
	// reflect.Type.MethodByName (method expression form): out[0] is *Failure,
	// out[1] is the actual return type.
	if got, want := method.Type.NumOut(), 1; got != want {
		t.Fatalf("Category method has %d return values, want %d", got, want)
	}
	wantType := reflect.TypeOf(ai.FailureCategory(0))
	if got := method.Type.Out(0); got != wantType {
		t.Errorf("Category()'s return type = %v, want %v — every Layer 2 category must map to a stated ai.FailureCategory member, declared in the accessor's own return type", got, wantType)
	}
}

// S-AEV-073 — no test written for this capability asserts on the content
// of a failure message string as the carrier of meaning. Audited by
// scanning every _test.go source file this capability added for the
// literal prefix `ai.Failure.Error()` renders — a hardcoded occurrence of
// it would be the signature of exactly the assertion this scenario
// forbids.
//
// The needle is assembled by concatenation, deliberately, so this
// scanning test's own source never contains the contiguous byte sequence
// it looks for — a self-reference this audit would otherwise
// false-positive on, mirroring ambient_authority_test.go's own uniform,
// never-name-this-file-specifically posture (S-AGP-054).
func TestSuite_NeverAssertsOnFailureMessageStringContent(t *testing.T) {
	t.Parallel()

	needle := "provider" + " failure:" // see file comment: assembled so this file cannot self-match

	entries, err := os.ReadDir(".")
	if err != nil {
		t.Fatalf("os.ReadDir(\".\") error = %v, want nil", err)
	}
	var scanned int
	for _, entry := range entries {
		name := entry.Name()
		if entry.IsDir() || !strings.HasSuffix(name, "_test.go") {
			continue
		}
		scanned++
		contents, err := os.ReadFile(name)
		if err != nil {
			t.Fatalf("os.ReadFile(%q) error = %v, want nil", name, err)
		}
		if strings.Contains(string(contents), needle) {
			t.Errorf("%s contains the literal ai.Failure.Error() rendering prefix %q — a test asserting on failure message-string content, forbidden by R-AEV-008", name, needle)
		}
	}
	if scanned == 0 {
		t.Fatal("no _test.go file found in \".\"; the audit would pass vacuously")
	}
}

// S-AEV-102 — (bite) doc.go's raw bytes contain both the ordering-
// invariants statement (R-AEV-011, 1.5) and the membership-criterion
// statement (R-AEV-011, the L2C-04 row, 3.2). A scratch edit removing or
// contradicting either MUST fail a named test.
//
// This is a literal textual-presence pin, not a new guarded-row
// mechanism: design AD-3 states the four ordering invariants are pinned
// "behaviorally", by the validator's own rejection scenarios, deliberately
// NOT as a guarded doc row — freezing partial-invariant wording as a
// byte-exact contract row would overclaim or churn the guard across later
// milestones. This test satisfies S-AEV-102's literal wording ("a test
// FAILS naming the missing or divergent statement") without reopening
// that decision: it pins the STATEMENTS' presence, not their exact
// byte-for-byte wording the way the L2C row guard pins L2C-04 itself.
func TestDocGo_StatesOrderingInvariantsAndMembershipCriterion(t *testing.T) {
	t.Parallel()

	path := docGoPathForInvariantPin(t)
	contents, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("os.ReadFile(%q) error = %v, want nil", path, err)
	}
	doc := string(contents)

	const orderingSentence = "Every event's ordering is stamped per consumer lane: independent,"
	const membershipSentence = "Stream membership: a fact belongs on the event stream or it does not exist upward"

	if !strings.Contains(doc, orderingSentence) {
		t.Errorf("doc.go is missing the ordering-invariants statement (expected to contain %q)", orderingSentence)
	}
	if !strings.Contains(doc, membershipSentence) {
		t.Errorf("doc.go is missing the membership-criterion statement (expected to contain %q)", membershipSentence)
	}
}

// docGoPathForInvariantPin resolves doc.go relative to this test file's
// own location, mirroring doc_contract_guard_test.go's docGoPath — a
// second, independent resolver rather than an import of that guard's
// unexported helper, so this pin does not depend on that guard's own
// internals.
func docGoPathForInvariantPin(t *testing.T) string {
	t.Helper()
	_, thisFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller(0) failed to report this test file's own path — cannot resolve doc.go")
	}
	return filepath.Join(filepath.Dir(thisFile), "doc.go")
}
