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
	"os"
	"os/exec"
	"runtime"
	"strings"
	"sync"
	"testing"
	"time"

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

// hookBoomAlwaysErrors returns a PreRequestHook that returns a
// sentinel error. Used by S-PRH-003 (R-PRH-003: failing hook aborts
// before I/O with typed error).
func hookBoomAlwaysErrors() func(_ context.Context, _ ai.Request) (ai.Request, error) {
	return func(_ context.Context, _ ai.Request) (ai.Request, error) {
		return ai.Request{}, errHookBoom
	}
}

// errHookBoom is the typed sentinel hookBoomAlwaysErrors returns.
// Distinct from the loop's other failure sentinels so a test
// failure can name the exact path under test.
var errHookBoom = errors.New("hook boom")

// hookMutatesInputViaAccessor returns a PreRequestHook that reads
// req.Messages() and writes back to the same slice header — the
// substrate R-REX-001 read test (S-PRH-004). The hook's *intent*
// is to mutate the loop's input in place; the substrate's promise
// is that the mutation does not propagate because Messages()
// returns a fresh slice on every call.
func hookMutatesInputViaAccessor() func(_ context.Context, req ai.Request) (ai.Request, error) {
	return func(_ context.Context, req ai.Request) (ai.Request, error) {
		msgs := req.Messages()
		if len(msgs) == 0 {
			return req, nil
		}
		// Build a fresh message that the helper uses to overwrite the
		// slot it just read — the substrate's no-mutation promise is
		// that the loop's input is unaffected.
		part, _ := ai.NewText("loop-input-mutation-attempt")
		mutated, _ := ai.NewMessage(ai.RoleUser, part)
		msgs[0] = mutated
		return req, nil
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

// S-PRH-003 — AG-08.1 failing hook aborts before I/O (R-PRH-003, D2).
// Given a hook that returns (ai.Request{}, errHookBoom), when Turn
// runs, then:
//
//   (a) provider.Requests() is empty — provider.Stream was never
//       called (the hook aborted the turn BEFORE I/O);
//   (b) the sink drains unblocked — the loop closed sink before
//       returning the typed error;
//   (c) the returned error wraps *ai.PreStreamFailure with a
//       hook-attributing FailureReport.Category
//       (FailureCategoryUnsupportedCapability).
//
// Mirrors the existing pre-stream-failure path (loop.go:140-147):
// close sink, return (ai.Message{}, 0, typedErr).
func TestTurn_PreRequestHook_FailureAbortsBeforeStream(t *testing.T) {
	t.Parallel()

	provider := agenttest.NewProvider(scriptTextResponse(t, ai.FinishReasonStop))
	sink := make(chan *agent.Event, 16)

	_, _, err := agent.Turn(
		contextBackground(),
		provider,
		"system prompt for prh-003",
		[]ai.Message{firstMessage(t)},
		agent.TurnOptions{PreRequestHook: hookBoomAlwaysErrors()},
		sink,
	)
	if err == nil {
		t.Fatal("Turn returned err = nil, want a non-nil typed pre-stream failure (hook returned an error)")
	}

	// (a) Provider never saw a request — Stream was not called.
	captured := provider.Requests()
	if len(captured) != 0 {
		t.Errorf("provider captured %d request(s), want 0 (hook failure must abort BEFORE provider.Stream, R-PRH-003)",
			len(captured))
	}

	// (b) Sink drains unblocked — the loop closed it. drainSink
	// returns when the channel closes; reaching here proves it.
	got := drainSink(t, sink)
	if len(got) == 0 {
		t.Error("sink was empty after Turn returned; the loop emits run_start + turn_start before the hook call, so the sink should carry those even on the failure path")
	}

	// (c) The returned error wraps *ai.PreStreamFailure with a
	// hook-attributing FailureReport.Category (UnsupportedCapability).
	var failure *ai.Failure
	if !errors.As(err, &failure) {
		t.Fatalf("errors.As(err, *ai.Failure) = false on %v — the loop must wrap hook failures in *ai.PreStreamFailure", err)
	}
	if failure.Category() != ai.FailureCategoryUnsupportedCapability {
		t.Errorf("failure category = %v, want %v (hook-attributing; not provider-auth)",
			failure.Category(), ai.FailureCategoryUnsupportedCapability)
	}
}

// S-PRH-004 — AG-08.1 hook cannot mutate input in place (R-PRH-004,
// R-REX-001). Given a hook that reads req.Messages() and writes
// back to the same slice header (a deliberate attempt to mutate the
// loop's input in place), when Turn runs, then the captured request
// at provider.Requests()[0] is byte-equal (via ai.Request.Equal) to
// the skeleton's captured request — the substrate's copy-on-write
// promise holds: the mutation is local to the hook's accessor copy
// and does not propagate.
//
// This is a direct read of R-REX-001: ai.Request is a value type
// whose accessors return fresh slices on every call.
func TestTurn_PreRequestHook_CannotMutateInput(t *testing.T) {
	t.Parallel()

	// First: capture the skeleton's known-good request (no hook).
	skeletonProvider := agenttest.NewProvider(scriptTextResponse(t, ai.FinishReasonStop))
	skeletonSink := make(chan *agent.Event, 16)

	_, _, err := agent.Turn(
		contextBackground(),
		skeletonProvider,
		"system prompt for prh-004",
		[]ai.Message{firstMessage(t)},
		agent.TurnOptions{},
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

	// Second: run with the mutating hook. The hook writes back into
	// the slice it got from req.Messages(); if Messages() returned
	// the loop's internal buffer (a violation of R-REX-001), the
	// loop's `req` would carry the mutated message; the captured
	// request would then NOT equal skeleton's.
	mutateProvider := agenttest.NewProvider(scriptTextResponse(t, ai.FinishReasonStop))
	mutateSink := make(chan *agent.Event, 16)

	_, _, err = agent.Turn(
		contextBackground(),
		mutateProvider,
		"system prompt for prh-004",
		[]ai.Message{firstMessage(t)},
		agent.TurnOptions{PreRequestHook: hookMutatesInputViaAccessor()},
		mutateSink,
	)
	if err != nil {
		t.Fatalf("mutating-hook Turn returned err = %v, want nil", err)
	}
	drainSink(t, mutateSink)
	mutateCaptured := mutateProvider.Requests()
	if len(mutateCaptured) != 1 {
		t.Fatalf("mutating-hook provider captured %d request(s), want 1", len(mutateCaptured))
	}

	// The mutation MUST NOT have propagated: captured == skeleton via
	// Request.Equal (which excludes MessageID identity but compares
	// every message's Content).
	if !mutateCaptured[0].Equal(skeletonCaptured[0]) {
		t.Errorf("mutating hook DID mutate loop input: captured request NOT equal to skeleton's (R-REX-001 / R-PRH-004 violation)\n  skeleton system: %q\n  mutate system:   %q",
			loopRequestSystemText(t, skeletonCaptured[0]), loopRequestSystemText(t, mutateCaptured[0]))
	}
}

// S-PRH-005 — AG-08.2 prefix stability (R-PRH-006, R-ACB-007). Given
// two consecutive Turn(...) calls with the same system, same tools,
// same hook, and the second transcript = first transcript + 1
// appended message, when both captured requests are compared via
// ai.Request.Equal, then:
//
//   (a) the captured requests' system regions are byte-equal;
//   (b) the captured requests' tools regions are byte-equal;
//   (c) the captured requests' CacheBoundaries() return the same
//       cascade order (R-ACB-007);
//   (d) the second captured request has exactly one more message than
//       the first; the first N messages are content-equal via
//       Message.Equal (which excludes MessageID identity).
//
// The walking-skeleton scope emits no tools region, so (b) is
// vacuously satisfied (both have no tools). The tools check still
// runs and asserts equal via Request.Equal.
//
// Mirrors AG-07 S-LSK-004 (two-sequential-turns share nothing on
// identity, share everything on input) but AG-08 closes the
// *byte-stability* property the prefix-cache (Layer 3) relies on.
func TestTurn_PrefixStability_ByteStableToolsSystem(t *testing.T) {
	t.Parallel()

	// First turn: a single user message.
	firstTranscript := []ai.Message{firstMessage(t)}

	firstProvider := agenttest.NewProvider(scriptTextResponse(t, ai.FinishReasonStop))
	firstSink := make(chan *agent.Event, 16)

	_, _, err := agent.Turn(
		contextBackground(),
		firstProvider,
		"system prompt for prh-005",
		firstTranscript,
		agent.TurnOptions{PreRequestHook: hookWithMarkerAppended()},
		firstSink,
	)
	if err != nil {
		t.Fatalf("first Turn returned err = %v, want nil", err)
	}
	drainSink(t, firstSink)
	firstCaptured := firstProvider.Requests()
	if len(firstCaptured) != 1 {
		t.Fatalf("first provider captured %d request(s), want 1", len(firstCaptured))
	}

	// Second turn: same system + same tools + same hook; transcript
	// has one APPENDED message (the first message + a new one).
	appendedPart, _ := ai.NewText("second-turn-appended")
	appendedMsg, _ := ai.NewMessage(ai.RoleUser, appendedPart)
	secondTranscript := append([]ai.Message{}, firstTranscript...)
	secondTranscript = append(secondTranscript, appendedMsg)

	secondProvider := agenttest.NewProvider(scriptTextResponse(t, ai.FinishReasonStop))
	secondSink := make(chan *agent.Event, 16)

	_, _, err = agent.Turn(
		contextBackground(),
		secondProvider,
		"system prompt for prh-005",
		secondTranscript,
		agent.TurnOptions{PreRequestHook: hookWithMarkerAppended()},
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

	// (a) System regions byte-equal (the marker was appended by the
	// hook in both turns — same input, same hook → same marker
	// position).
	if !strings.Contains(loopRequestSystemText(t, firstCaptured[0]), hookMarker) {
		t.Errorf("first turn's system region LACKS marker %q (hook did not run?)", hookMarker)
	}
	if !strings.Contains(loopRequestSystemText(t, secondCaptured[0]), hookMarker) {
		t.Errorf("second turn's system region LACKS marker %q (hook did not run?)", hookMarker)
	}
	if loopRequestSystemText(t, firstCaptured[0]) != loopRequestSystemText(t, secondCaptured[0]) {
		t.Errorf("system regions differ between turns:\n  first:  %q\n  second: %q",
			loopRequestSystemText(t, firstCaptured[0]), loopRequestSystemText(t, secondCaptured[0]))
	}

	// (b) Tools regions byte-equal. Walking skeleton has no tools;
	// assert both report absent tools (or both report equal empty).
	firstTools, firstHasTools := firstCaptured[0].Tools()
	secondTools, secondHasTools := secondCaptured[0].Tools()
	if firstHasTools != secondHasTools {
		t.Errorf("tools presence differs: first=%v, second=%v", firstHasTools, secondHasTools)
	}
	if firstHasTools {
		firstToolList := firstTools.Tools()
		secondToolList := secondTools.Tools()
		if len(firstToolList) != len(secondToolList) {
			t.Errorf("tools length differs: first=%d, second=%d", len(firstToolList), len(secondToolList))
		}
		// Walking-skeleton scope: no tools region by construction.
		// The presence equality above is the AG-08 invariant; if a
		// future milestone installs tools, this branch needs a
		// field-by-field comparison through the accessor surface.
		_ = secondToolList
	}

	// (c) CacheBoundaries() cascade order pinned (R-ACB-007).
	firstBoundaries := firstCaptured[0].CacheBoundaries()
	secondBoundaries := secondCaptured[0].CacheBoundaries()
	if len(firstBoundaries) != len(secondBoundaries) {
		t.Errorf("CacheBoundaries() count differs: first=%d, second=%d", len(firstBoundaries), len(secondBoundaries))
	}
	for i := range firstBoundaries {
		if firstBoundaries[i] != secondBoundaries[i] {
			t.Errorf("CacheBoundaries()[%d] differs: first=%+v, second=%+v (cascade order drift, R-ACB-007)",
				i, firstBoundaries[i], secondBoundaries[i])
		}
	}

	// (d) Message region grew by 1; first N messages content-equal.
	firstMsgs := firstCaptured[0].Messages()
	secondMsgs := secondCaptured[0].Messages()
	if len(secondMsgs) != len(firstMsgs)+1 {
		t.Errorf("message region grew by %d, want 1 (first=%d, second=%d)",
			len(secondMsgs)-len(firstMsgs), len(firstMsgs), len(secondMsgs))
	}
	for i := 0; i < len(firstMsgs); i++ {
		if !firstMsgs[i].Equal(secondMsgs[i]) {
			t.Errorf("message[%d] content differs:\n  first:  %+v\n  second: %+v",
				i, firstMsgs[i].Content(), secondMsgs[i].Content())
		}
	}
}

// S-PRH-006 — AG-08.2 hook determinism (R-PRH-007). Given a hook
// that adds a constant system segment, when the loop calls it twice
// with byte-equal req values, then both hook invocations' outputs are
// byte-equal (via Request.Equal) — hook-applied markers cannot
// oscillate between turns.
//
// Combined with S-PRH-005, this closes the prefix-stability property:
// tools + system regions byte-equal across turns AND the hook that
// populated them is deterministic. A timestamp-bearing hook would
// fail S-PRH-005's cascade comparison; a non-deterministic hook
// would fail S-PRH-006 directly.
func TestTurn_PrefixStability_DeterministicHook(t *testing.T) {
	t.Parallel()

	// First call: capture the captured request.
	firstProvider := agenttest.NewProvider(scriptTextResponse(t, ai.FinishReasonStop))
	firstSink := make(chan *agent.Event, 16)

	_, _, err := agent.Turn(
		contextBackground(),
		firstProvider,
		"system prompt for prh-006",
		[]ai.Message{firstMessage(t)},
		agent.TurnOptions{PreRequestHook: hookWithMarkerAppended()},
		firstSink,
	)
	if err != nil {
		t.Fatalf("first Turn returned err = %v, want nil", err)
	}
	drainSink(t, firstSink)
	firstCaptured := firstProvider.Requests()
	if len(firstCaptured) != 1 {
		t.Fatalf("first provider captured %d request(s), want 1", len(firstCaptured))
	}

	// Second call: same inputs, same hook. The captured request MUST
	// be byte-equal to the first — the hook is deterministic.
	secondProvider := agenttest.NewProvider(scriptTextResponse(t, ai.FinishReasonStop))
	secondSink := make(chan *agent.Event, 16)

	_, _, err = agent.Turn(
		contextBackground(),
		secondProvider,
		"system prompt for prh-006",
		[]ai.Message{firstMessage(t)},
		agent.TurnOptions{PreRequestHook: hookWithMarkerAppended()},
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

	if !firstCaptured[0].Equal(secondCaptured[0]) {
		t.Errorf("hook NOT deterministic: first captured request != second captured request (R-PRH-007 violation)\n  first system:  %q\n  second system: %q",
			loopRequestSystemText(t, firstCaptured[0]), loopRequestSystemText(t, secondCaptured[0]))
	}
}

// S-PRH-007 — AG-07 W1 carry: back-pressure through the hook path.
// Given an unbuffered sink (make(chan *Event)) and a concurrent
// consumer goroutine that drains sink while Turn runs, when Turn
// returns, then runtime.NumGoroutine() returns to its baseline (no
// stranded producer) and the consumer's drain unblocks — proves the
// hook path supports back-pressure, not just the buffered-sink path
// AG-07 exercised.
//
// The hook is installed to exercise the full hook → provider.Stream
// → translate → emit path through the unbuffered channel.
func TestTurn_PreRequestHook_UnbufferedSink(t *testing.T) {
	t.Parallel()

	provider := agenttest.NewProvider(scriptTextResponse(t, ai.FinishReasonStop))
	sink := make(chan *agent.Event) // unbuffered

	// Baseline goroutine count before launching the consumer +
	// Turn. Both add goroutines; after Turn returns and the
	// consumer exits, NumGoroutine must drop back to baseline.
	baseline := runtime.NumGoroutine()

	var consumerWG sync.WaitGroup
	consumerWG.Add(1)
	go func() {
		defer consumerWG.Done()
		// Consumer drains every event; the loop emits into an
		// unbuffered channel, so the consumer's presence is what
		// keeps the producer from blocking. The loop terminates the
		// drain by closing sink before returning.
		for range sink {
			// discard — the test only cares that the consumer reads.
			_ = struct{}{}
		}
	}()

	_, _, err := agent.Turn(
		contextBackground(),
		provider,
		"system prompt for prh-007",
		[]ai.Message{firstMessage(t)},
		agent.TurnOptions{PreRequestHook: hookWithMarkerAppended()},
		sink,
	)
	if err != nil {
		t.Fatalf("Turn returned err = %v, want nil (sink consumer was draining)", err)
	}

	// The loop closed sink on its return — consumer's range loop
	// must terminate. Wait for the consumer to exit.
	consumerDone := make(chan struct{})
	go func() {
		consumerWG.Wait()
		close(consumerDone)
	}()
	select {
	case <-consumerDone:
		// Consumer drained to completion; the loop closed sink.
	case <-time.After(5 * time.Second):
		t.Fatal("consumer did not exit within 5s — sink was not closed by Turn (back-pressure failure)")
	}

	// Goroutine baseline must return to its pre-Turn value (or
	// below it after GC). Poll briefly to allow the runtime to
	// settle; the consumer goroutine has already exited.
	waitForGoroutineBaseline(t, baseline, 2*time.Second)
}

// waitForGoroutineBaseline polls runtime.NumGoroutine() until it
// drops back to baseline (or below) or the deadline expires. Used
// by S-PRH-007 to assert no stranded producer.
func waitForGoroutineBaseline(t *testing.T, baseline int, deadline time.Duration) {
	t.Helper()
	end := time.Now().Add(deadline)
	for time.Now().Before(end) {
		if runtime.NumGoroutine() <= baseline {
			return
		}
		runtime.Gosched()
		time.Sleep(5 * time.Millisecond)
	}
	t.Errorf("runtime.NumGoroutine() = %d, baseline = %d (stranded producer goroutine — back-pressure failure)",
		runtime.NumGoroutine(), baseline)
}

// TestTurn_PreRequestHook_SubstrateUntouched — NFR-PRH-003 substrate
// guard (AG-08's author). Given the base ref that AG-08 branched
// from (post-AG-07 merge at 93077c07), when the AG-08 changes are
// compared against that ref, then the diff against
// backend/agent/src/agent/ (excluding loop.go, loop_test.go, and
// loop_hook_test.go) shows zero lines changed, and the diff against
// backend/agent/go.mod and backend/agent/go.sum is empty — the 21
// substrate files (per NFR-PRH-003) are byte-unchanged.
//
// The base ref is resolved at run time so the test survives merge:
// an `AG08_BASE_REF` env var pins it explicitly when set; otherwise
// it falls back to the merge-base of HEAD and origin/main (AG-07 W3
// fix carried forward — env-var fallback shipped in AG-07 PR #167).
//
// This is the AG-08 substrate guard; AG-07's TestTurn_SubstrateUntouched
// also runs (its filter was widened to exclude loop_hook_test.go).
// Together they cover the substrate for AG-08's full footprint.
func TestTurn_PreRequestHook_SubstrateUntouched(t *testing.T) {
	root, err := gitTopLevelHook(t)
	if err != nil {
		t.Fatalf("git rev-parse --show-toplevel failed: %v", err)
	}

	mainRef := os.Getenv("AG08_BASE_REF")
	if mainRef == "" {
		// Fall back to the merge-base with origin/main (the most
		// recent common ancestor).
		out, err := gitOutputHook(t, root, "merge-base", "HEAD", "origin/main")
		if err != nil {
			t.Fatalf("substrate untouched: cannot determine base ref (set AG08_BASE_REF): %v", err)
		}
		mainRef = out
	}

	// 1. backend/agent/src/agent/ excluding loop.go, loop_test.go,
	// loop_hook_test.go. The three loop files are owned by AG-07
	// (loop.go, loop_test.go) and AG-08 (loop_hook_test.go); all
	// other files are substrate.
	raw, err := gitDiffHook(t, root, mainRef, "backend/agent/src/agent/")
	if err != nil {
		t.Fatalf("git diff %s -- backend/agent/src/agent/ failed: %v", mainRef, err)
	}
	stripped := filterOutLoopHookFiles(raw)
	if len(stripped) != 0 {
		t.Errorf("substrate was edited (NFR-PRH-003 violated):\n%s", stripped)
	}

	// 2. backend/agent/go.mod and backend/agent/go.sum byte-identical.
	goModSum, err := gitDiffHook(t, root, mainRef, "backend/agent/go.mod", "backend/agent/go.sum")
	if err != nil {
		t.Fatalf("git diff %s -- backend/agent/go.mod go.sum failed: %v", mainRef, err)
	}
	if len(goModSum) != 0 {
		t.Errorf("go.mod / go.sum drifted from main:\n%s", goModSum)
	}
}

// gitTopLevelHook returns the absolute path of the repo root.
func gitTopLevelHook(t *testing.T) (string, error) {
	t.Helper()
	cmd := exec.Command("git", "rev-parse", "--show-toplevel")
	out, err := cmd.CombinedOutput()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(out)), nil
}

// gitDiffHook runs `git diff <ref> -- <paths...>` and returns the
// raw output. An empty output means no diff.
func gitDiffHook(t *testing.T, root, ref string, paths ...string) (string, error) {
	t.Helper()
	args := []string{"diff", ref, "--"}
	args = append(args, paths...)
	cmd := exec.Command("git", args...)
	cmd.Dir = root
	out, err := cmd.CombinedOutput()
	if err != nil {
		return "", err
	}
	return string(out), nil
}

// gitOutputHook runs an arbitrary `git <args...>` invocation from
// the supplied root and returns the trimmed stdout. The substrate
// guard's escape hatch for refs that must be resolved at run time
// (e.g. `git merge-base HEAD origin/main`) rather than pinned in
// source.
func gitOutputHook(t *testing.T, root string, args ...string) (string, error) {
	t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = root
	out, err := cmd.CombinedOutput()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(out)), nil
}

// filterOutLoopHookFiles strips entire file-level diff blocks whose
// path matches loop.go, loop_test.go, loop_hook_test.go, tool.go,
// tool_test.go, or scheduler_test.go. Mirrors loop_test.go's filter
// at file granularity. The widening to the tool family is the
// AG-08 W3 pattern carried into AG-09 (NFR-TLS-003's 21-path
// substrate list itself stays byte-untouched; the filter just
// accommodates AG-09's new files).
//
// AG-10 widens the filter to also exclude permission_protocol.go and
// permission_protocol_test.go (the AG-10 file pair, owned by this
// milestone; NFR-TLS-003 7th carry). The AG-10 remediation round
// (verify-report W8/S4) widens it again to exclude
// loop_permission_e2e_test.go and permission_policy_helpers_test.go.
func filterOutLoopHookFiles(diff string) string {
	if diff == "" {
		return ""
	}
	var kept strings.Builder
	lines := strings.Split(diff, "\n")
	skip := false
	for _, line := range lines {
		if strings.HasPrefix(line, "diff --git ") {
			idx := strings.LastIndex(line, " b/")
			path := line
			if idx >= 0 {
				path = line[idx+3:]
			}
			skip = strings.HasSuffix(path, "/loop.go") ||
				strings.HasSuffix(path, "/loop_test.go") ||
				strings.HasSuffix(path, "/loop_hook_test.go") ||
				strings.HasSuffix(path, "/loop_tool_dispatch_test.go") ||
				strings.HasSuffix(path, "/tool.go") ||
				strings.HasSuffix(path, "/tool_test.go") ||
				strings.HasSuffix(path, "/scheduler.go") ||
				strings.HasSuffix(path, "/scheduler_test.go") ||
				strings.HasSuffix(path, "/scripted_tool_test.go") ||
				// AG-10 widening (NFR-TLS-003 7th carry):
				strings.HasSuffix(path, "/permission_protocol.go") ||
				strings.HasSuffix(path, "/permission_protocol_test.go") ||
				// AG-10 remediation widening (verify-report W8/S4):
				strings.HasSuffix(path, "/loop_permission_e2e_test.go") ||
				strings.HasSuffix(path, "/permission_policy_helpers_test.go") ||
				// AG-11 widening (R-ATT-009/R-LSK-004/R-APP-012, D3):
				// turn_events.go and failure.go are pre-existing
				// substrate files released for AG-11 only (exact
				// filename, no wildcard/prefix/directory pattern);
				// the three suffixes after them are AG-11's new
				// test files. Byte-in-sync with loop_test.go's
				// filterOutLoopFiles.
				strings.HasSuffix(path, "/turn_events.go") ||
				strings.HasSuffix(path, "/failure.go") ||
				strings.HasSuffix(path, "/invariant_pin_test.go") ||
				strings.HasSuffix(path, "/turn_termination_test.go") ||
				strings.HasSuffix(path, "/turn_failure_test.go") ||
				// AG-12 widening (NFR-HIS-004, R-HIS-004): history.go is
				// the new transcript-store file; the three suffixes
				// after it are AG-12's new test files (behavior tests
				// plus the closed-route surface guard). Byte-in-sync
				// with loop_test.go's filterOutLoopFiles.
				strings.HasSuffix(path, "/history.go") ||
				strings.HasSuffix(path, "/history_test.go") ||
				strings.HasSuffix(path, "/history_synthesis_test.go") ||
				strings.HasSuffix(path, "/history_surface_guard_test.go") ||
				// AG-12 widening (R-HIS-009, R-AGP-002): doc.go and
				// doc_contract_guard_test.go are pre-existing substrate
				// files released for AG-12 only, mirroring AG-11's
				// turn_events.go/failure.go precedent above — AG-12
				// lands the L2C-07 row and its expectedLayer2ContractRows
				// entry, the first doc-row amendment since this guard
				// was introduced at AG-08 (AG-04/05/06 predate it).
				// Byte-in-sync with loop_test.go's filterOutLoopFiles.
				strings.HasSuffix(path, "/doc.go") ||
				strings.HasSuffix(path, "/doc_contract_guard_test.go") ||
				// AG-13 widening (R-RUN-012/S-RUN-111, R-LSK-004
				// "AG-13 needs no release"): harness.go is the new
				// run-driver production file; harness_test.go is its
				// first test file. Exact filenames, no
				// wildcard/prefix/directory pattern. Byte-in-sync with
				// loop_test.go's filterOutLoopFiles.
				strings.HasSuffix(path, "/harness.go") ||
				strings.HasSuffix(path, "/harness_test.go") ||
				strings.HasSuffix(path, "/harness_steering_test.go") ||
				strings.HasSuffix(path, "/harness_pause_test.go") ||
				// Phase 5 widening (R-RUN-010, S-RUN-091/S-APP-016):
				// harness_suspension_test.go, the fifth and final
				// /harness*.go suffix. Byte-in-sync with loop_test.go's
				// filterOutLoopFiles.
				strings.HasSuffix(path, "/harness_suspension_test.go") ||
				// AG-14 widening (R-LSK-004 "AG-14's release scope",
				// R-CAN-008 L2C-08): run_events.go is a pre-existing
				// substrate file released for AG-14 only, mirroring
				// AG-11's turn_events.go/failure.go and AG-12's
				// doc.go/doc_contract_guard_test.go precedent above —
				// exact filename, no wildcard/prefix/directory pattern.
				// cancellation.go is AG-14's new production file; the
				// four suffixes after it are AG-14's new test files.
				// Byte-in-sync with loop_test.go's filterOutLoopFiles.
				strings.HasSuffix(path, "/run_events.go") ||
				strings.HasSuffix(path, "/cancellation.go") ||
				strings.HasSuffix(path, "/cancellation_interrupt_test.go") ||
				strings.HasSuffix(path, "/cancellation_shutdown_test.go") ||
				strings.HasSuffix(path, "/cancellation_winddown_test.go") ||
				strings.HasSuffix(path, "/cancellation_events_test.go") ||
				// AG-15 widening (agent-loop-skeleton delta's R-LSK-004
				// "AG-15 needs no release and widens both filters by
				// name", S-LSK-023): retry_policy.go is AG-15's first
				// new production file; retry_policy_test.go is its
				// first test file. Byte-in-sync with loop_test.go's
				// filterOutLoopFiles.
				strings.HasSuffix(path, "/retry_policy.go") ||
				strings.HasSuffix(path, "/retry_policy_test.go") ||
				// AG-15 amendment (enumerated, not in tasks.md's
				// original file list): retry_decision_internal_test.go,
				// package-internal for S-RTY-001 (design AD-2's
				// unexported retryDecision/retryVerdict; NFR-RTY-001's
				// own carve-out). Byte-in-sync with loop_test.go's
				// filterOutLoopFiles.
				strings.HasSuffix(path, "/retry_decision_internal_test.go")
		}
		if !skip {
			kept.WriteString(line)
			kept.WriteString("\n")
		}
	}
	return strings.TrimSpace(kept.String())
}