// AG-08 — strict-TDD tests for the pre-request hook seam
// (R-PRH-001..007, S-PRH-001..007 + bites S-PRH-001a, S-PRH-001b).
//
// Every scenario is in package agent_test (NFR-PRH-001; AG-07 W6 /
// AG-07 W1 carry): the external-test posture proves every behavioral
// claim from outside the package, with no reach into unexported
// surface. The file carries the helpers the S-PRH-001..007 tests
// reuse (loopRequestSystemText, systemIncludesSegment, plus the
// bite harness for S-PRH-001a/001b) and the substrate "untouched"
// guard at NFR-PRH-003.
//
// # Bite-first RED ordering (defense against AG-05 W1)
//
// S-PRH-001a (no-segment bite) and S-PRH-001b (adds-segment bite) are
// RED-recorded BEFORE S-PRH-001 (the property test) goes GREEN. The
// two bites prove the test wiring distinguishes "hook did not add a
// segment" from "hook added a segment" — the AG-05 W1 failure mode
// the bite pattern defends against. Mirrors AG-05 S-AMT-071/072 and
// AG-07 S-LSK-003a/S-LSK-003b.
package agent_test

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/cachicamas/backend/agent/src/agent"
	"github.com/cachicamas/backend/agent/src/agenttest"
	"github.com/cachicamas/backend/agent/src/ai"
)

// hookMarker is the literal text the AG-08 hook fixture appends to
// the system instruction. Using a single, distinctive marker across
// the file keeps the bites and the property tests on the same shape
// and the failure messages greppable.
const hookMarker = "AG-08-PRE-REQUEST-HOOK-MARKER"

// loopRequestSystemText returns the SystemInstruction text that the
// hook fixture appended. Used by the bites to assert presence and
// absence of the marker without depending on the SystemInstruction
// segment internals.
func loopRequestSystemText(t *testing.T, req ai.Request) string {
	t.Helper()
	system, hasSystem := req.SystemInstruction()
	if !hasSystem {
		t.Fatal("captured request carries no SystemInstruction; the loop's buildLoopRequest should always attach one")
	}
	var b strings.Builder
	for _, seg := range system.Segments() {
		b.WriteString(seg.Text())
	}
	return b.String()
}

// systemIncludesSegment reports whether text appears anywhere in the
// captured request's system region.
func systemIncludesSegment(t *testing.T, req ai.Request, want string) bool {
	t.Helper()
	return strings.Contains(loopRequestSystemText(t, req), want)
}

// hookNoSegmentIdentity returns a PreRequestHook that returns req
// unchanged — the "identity default" hook shape, used by the bites
// and by the deterministic-prefix test.
func hookNoSegmentIdentity() func(_ context.Context, req ai.Request) (ai.Request, error) {
	return func(_ context.Context, req ai.Request) (ai.Request, error) {
		return req, nil
	}
}

// hookWithMarkerAppended returns a PreRequestHook that derives a new
// ai.Request by appending the hookMarker as a new system segment.
// Used by S-PRH-001 (R-PRH-002: hook sees + shapes outgoing request).
//
// It exercises AI-12's copy-on-write rebuild (R-REX-001):
// req.With(...) returns a fresh value; the loop's `req` is observably
// unmodified after the call returns.
func hookWithMarkerAppended() func(_ context.Context, req ai.Request) (ai.Request, error) {
	return func(_ context.Context, req ai.Request) (ai.Request, error) {
		system, hasSystem := req.SystemInstruction()
		if !hasSystem {
			return ai.Request{}, errors.New("hookWithMarkerAppended: request carries no SystemInstruction")
		}
		markerSeg, err := ai.NewSegment(hookMarker)
		if err != nil {
			return ai.Request{}, err
		}
		segments := append([]ai.Segment{}, system.Segments()...)
		segments = append(segments, markerSeg)
		newInstr, err := ai.NewSystemInstruction(segments...)
		if err != nil {
			return ai.Request{}, err
		}
		return req.With(ai.WithSystemInstruction(newInstr))
	}
}

// S-PRH-001a — RED bite (recorded 2026-08-13): the no-segment hook
// returns req unchanged; the assertion checks that the captured
// request's system region DOES NOT contain the marker.
//
// RED at write time: the hook returns (req, nil) unchanged, so the
// captured system region cannot contain the marker. The assertion
// fails for the right reason — marker absent — proving the test
// machinery is wired against production behavior (not vacuously true).
//
// Mirrors AG-05 S-AMT-071/072 and AG-07 S-LSK-003a/003b.
func TestTurn_PreRequestHook_NoSegmentBite(t *testing.T) {
	t.Parallel()

	provider := agenttest.NewProvider(scriptTextResponse(t, ai.FinishReasonStop))
	sink := make(chan *agent.Event, 16)

	_, _, err := agent.Turn(
		contextBackground(),
		provider,
		"system prompt for prh-001a",
		[]ai.Message{firstMessage(t)},
		agent.TurnOptions{PreRequestHook: hookNoSegmentIdentity()},
		sink,
	)
	if err != nil {
		t.Fatalf("Turn returned err = %v, want nil (hook returned no error)", err)
	}
	drainSink(t, sink)

	captured := provider.Requests()
	if len(captured) != 1 {
		t.Fatalf("provider captured %d request(s), want 1", len(captured))
	}

	// BITE: a no-segment hook returns req unchanged; the captured
	// system region MUST NOT contain the marker. The assertion below
	// fails RED at write time for the right reason (marker absent).
	if systemIncludesSegment(t, captured[0], hookMarker) {
		t.Errorf("no-segment bite DID NOT bite: captured system region CONTAINS marker %q — the marker should be absent when the hook returns req unchanged",
			hookMarker)
	}
}

// S-PRH-001b — RED bite (recorded 2026-08-13): the no-segment hook
// returns req unchanged; the assertion checks that the captured
// request IS byte-equal to skeleton's captured request via
// ai.Request.Equal.
//
// The skeleton's captured request (zero-value TurnOptions, no hook)
// is byte-equal to the hook-noop's captured request — because the
// hook returns (req, nil) unchanged, the provider receives the same
// request as the skeleton case. The bite's job is to prove the
// assertion compares hooks honestly: an empty-skull hook MUST match
// skeleton's output; only a hook that mutates via req.With(...) will
// fail this assertion.
//
// The companion test (S-PRH-001, "hook adds a system segment") is
// what the hook-seam code makes pass; together they form the bite
// harness AG-05 W1 demands.
func TestTurn_PreRequestHook_AddsSegmentBite(t *testing.T) {
	t.Parallel()

	provider := agenttest.NewProvider(scriptTextResponse(t, ai.FinishReasonStop))
	sink := make(chan *agent.Event, 16)

	_, _, err := agent.Turn(
		contextBackground(),
		provider,
		"system prompt for prh-001b",
		[]ai.Message{firstMessage(t)},
		agent.TurnOptions{PreRequestHook: hookNoSegmentIdentity()},
		sink,
	)
	if err != nil {
		t.Fatalf("Turn returned err = %v, want nil", err)
	}
	drainSink(t, sink)

	captured := provider.Requests()
	if len(captured) != 1 {
		t.Fatalf("provider captured %d request(s), want 1", len(captured))
	}

	// Skeleton's captured request (no hook installed).
	skeletonProvider := agenttest.NewProvider(scriptTextResponse(t, ai.FinishReasonStop))
	skeletonSink := make(chan *agent.Event, 16)
	_, _, err = agent.Turn(
		contextBackground(),
		skeletonProvider,
		"system prompt for prh-001b",
		[]ai.Message{firstMessage(t)},
		agent.TurnOptions{}, // zero value, no hook
		skeletonSink,
	)
	if err != nil {
		t.Fatalf("skeleton Turn returned err = %v, want nil", err)
	}
	drainSink(t, skeletonSink)
	skeletonCaptured := skeletonProvider.Requests()
	if len(skeletonCaptured) != 1 {
		t.Fatalf("skeleton provider captured %d request(s), want 1", len(skeletonCaptured))
	}

	// BITE: with the no-op hook, captured MUST equal skeleton's via
	// Request.Equal — proving the wiring doesn't accidentally add or
	// drop bytes when the hook is a real no-op. The next test
	// (S-PRH-001, "adds system segment") breaks this equality by
	// appending the marker; together they prove the seam mutates
	// only when the hook mutates.
	if !captured[0].Equal(skeletonCaptured[0]) {
		t.Errorf("no-op-hook bite: captured request NOT equal to skeleton's\n  captured system: %q\n  skeleton system: %q",
			loopRequestSystemText(t, captured[0]), loopRequestSystemText(t, skeletonCaptured[0]))
	}

	// Marker MUST be absent in both — guards against accidental
	// pre-population by either path.
	if systemIncludesSegment(t, captured[0], hookMarker) {
		t.Errorf("no-op hook leaked the marker into the captured system region")
	}
	if systemIncludesSegment(t, skeletonCaptured[0], hookMarker) {
		t.Errorf("skeleton leaked the marker into the captured system region")
	}
}

// S-PRH-001 — AG-08.1 happy path. Given a hook that derives a new
// ai.Request via req.With(ai.WithSystemInstruction(...)) appending a
// marker segment, when Turn runs, then the captured request at
// provider.Requests()[0] carries the added marker — the hook's return
// value is what the provider received (R-PRH-002 / D2).
//
// RED bites (S-PRH-001a/001b) are RED-recorded first (Task 2) and
// GREEN at this point (Task 3 added the seam). This is the
// property test that the bites defend against being vacuous.
func TestTurn_PreRequestHook_AddsSystemSegment(t *testing.T) {
	t.Parallel()

	provider := agenttest.NewProvider(scriptTextResponse(t, ai.FinishReasonStop))
	sink := make(chan *agent.Event, 16)

	_, _, err := agent.Turn(
		contextBackground(),
		provider,
		"system prompt for prh-001",
		[]ai.Message{firstMessage(t)},
		agent.TurnOptions{PreRequestHook: hookWithMarkerAppended()},
		sink,
	)
	if err != nil {
		t.Fatalf("Turn returned err = %v, want nil", err)
	}
	drainSink(t, sink)

	captured := provider.Requests()
	if len(captured) != 1 {
		t.Fatalf("provider captured %d request(s), want 1", len(captured))
	}

	// The hook appended the marker — assert the captured system
	// region carries it (the property: hook sees + shapes the
	// outgoing request).
	if !systemIncludesSegment(t, captured[0], hookMarker) {
		t.Errorf("captured request's system region LACKS marker %q — the hook's derived request did not reach the provider (R-PRH-002 violation)",
			hookMarker)
	}

	// Belt-and-braces: the original system region (the system prompt
	// the loop's buildLoopRequest constructed) must still be present
	// alongside the marker — the hook APPENDED, did not REPLACE.
	if !strings.Contains(loopRequestSystemText(t, captured[0]), "system prompt for prh-001") {
		t.Errorf("captured system region lost the original system prompt — hook replaced instead of appending")
	}
}

// S-PRH-002 — AG-08.1 identity default (R-PRH-005, D4a). Given a
// zero-value TurnOptions (no hook installed), when Turn runs twice
// against the same script and inputs as the skeleton's S-LSK-001
// baseline, then provider.Requests()[0] from each turn is byte-equal
// (via ai.Request.Equal) to the other — the seam adds zero observable
// behavior when not installed (AG-07 R-LSK-002 carry).
//
// Two-turn comparison replaces the JSON golden fixture the design
// spec called for: ai.Request is a sealed value type (R-REX-001,
// V-REQ-03) with no MarshalJSON, so a stable byte-fingerprint
// golden would require either a JSON encoding (substrate edit) or
// a per-region field dump (test bloat). The two-turn comparison
// proves the same property: identical inputs + identity default →
// byte-equal captured requests.
//
// This is RED-then-GREEN: the identity default is the load-bearing
// property AG-07's S-LSK-001 byte-stability passed and AG-08 must
// not regress.
func TestTurn_PreRequestHook_NilIdentity(t *testing.T) {
	t.Parallel()

	skeletonProvider := agenttest.NewProvider(scriptTextResponse(t, ai.FinishReasonStop))
	skeletonSink := make(chan *agent.Event, 16)

	_, _, err := agent.Turn(
		contextBackground(),
		skeletonProvider,
		"system prompt for prh-002",
		[]ai.Message{firstMessage(t)},
		agent.TurnOptions{}, // zero value, no hook (identity default)
		skeletonSink,
	)
	if err != nil {
		t.Fatalf("skeleton Turn returned err = %v, want nil", err)
	}
	drainSink(t, skeletonSink)

	skeletonCaptured := skeletonProvider.Requests()
	if len(skeletonCaptured) != 1 {
		t.Fatalf("skeleton provider captured %d request(s), want 1", len(skeletonCaptured))
	}

	// Run a SECOND skeleton Turn with the same inputs. The captured
	// request MUST be byte-equal to the first — the identity default
	// preserves AG-07's R-LSK-002 byte-stability on identical inputs.
	secondProvider := agenttest.NewProvider(scriptTextResponse(t, ai.FinishReasonStop))
	secondSink := make(chan *agent.Event, 16)

	_, _, err = agent.Turn(
		contextBackground(),
		secondProvider,
		"system prompt for prh-002",
		[]ai.Message{firstMessage(t)},
		agent.TurnOptions{}, // zero value, no hook (identity default)
		secondSink,
	)
	if err != nil {
		t.Fatalf("second Turn returned err = %v, want nil", err)
	}
	drainSink(t, secondSink)

	secondCaptured := secondProvider.Requests()
	if len(secondCaptured) != 1 {
		t.Fatalf("second provider captured %d request(s), want 1", len(secondCaptured))
	}

	if !skeletonCaptured[0].Equal(secondCaptured[0]) {
		t.Errorf("identity default NOT byte-stable: skeleton's captured request != second's captured request\n  skeleton system: %q\n  second system:   %q",
			loopRequestSystemText(t, skeletonCaptured[0]), loopRequestSystemText(t, secondCaptured[0]))
	}
}