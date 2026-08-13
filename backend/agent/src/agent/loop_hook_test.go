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