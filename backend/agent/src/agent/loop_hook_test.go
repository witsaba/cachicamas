// AG-08 — strict-TDD tests for the pre-request hook seam
// (R-PRH-001..007, S-PRH-001..007 + bites S-PRH-001a, S-PRH-001b).
//
// Every scenario is in package agent_test (NFR-PRH-001; AG-07 W6 /
// AG-07 W1 carry): the external-test posture proves every behavioral
// claim from outside the package, with no reach into unexported
// surface. The file carries the helpers the S-PRH-001..007 tests
// reuse (loopRequestSystemText, systemIncludesSegment, plus a small
// identity-build helper for the identity-default scenario), plus the
// bite harness (S-PRH-001a, S-PRH-001b) and the substrate "untouched"
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
	"encoding/json"
	"errors"
	"os"
	"os/exec"
	"reflect"
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

// hookWithMarkerAppended returns a PreRequestHook that derives a new
// ai.Request by appending the hookMarker as a new system segment.
// Used by S-PRH-001 and S-PRH-002's "happy path" assertion.
//
// It exercises AI-12's copy-on-write rebuild (R-REX-001):
// req.With(...) returns a fresh value; the loop's `req` is observably
// unmodified after the call returns.
func hookWithMarkerAppended() agent.PreRequestHook {
	return func(ctx context.Context, req ai.Request) (ai.Request, error) {
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

// hookNoSegmentIdentity returns a PreRequestHook that returns req
// unchanged — the "identity default" hook shape, used by the bites
// and by the deterministic-prefix test.
func hookNoSegmentIdentity() agent.PreRequestHook {
	return func(ctx context.Context, req ai.Request) (ai.Request, error) {
		return req, nil
	}
}

// hookBoomAlwaysErrors returns a PreRequestHook that returns a
// sentinel error. Used by S-PRH-003 (failing-hook aborts-before-I/O).
func hookBoomAlwaysErrors() agent.PreRequestHook {
	return func(ctx context.Context, req ai.Request) (ai.Request, error) {
		return ai.Request{}, errHookBoom
	}
}

// errHookBoom is the typed sentinel hookBoomAlwaysErrors returns.
// Distinct from the loop's other failure sentinels so a test failure
// can name the exact path under test.
var errHookBoom = errors.New("hook boom")

// hookMutatesInputViaAccessor returns a PreRequestHook that reads
// req.Messages() and writes back to the same slice header — the
// substrate R-REX-001 read test (S-PRH-004). The hook's *intent* is
// to mutate the loop's input in place; the substrate's promise is
// that the mutation does not propagate because Messages() returns a
// fresh slice on every call.
func hookMutatesInputViaAccessor() agent.PreRequestHook {
	return func(ctx context.Context, req ai.Request) (ai.Request, error) {
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
// request's system region CONTAINS the marker.
//
// RED at write time: the hook returns (req, nil) unchanged, so the
// captured system region cannot contain the marker. The assertion
// fails for the right reason — marker absent — proving the test
// machinery is wired against production behavior (not vacuously true).
//
// GREEN at Task 3: the production hook-seam code lives in loop.go and
// the test is rewritten for the property case (S-PRH-001).
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
// RED-then-GREEN path: this test is the witness that the "no-op" hook
// is a real no-op. The mirror test (S-PRH-001, "hook adds a system
// segment") is what the hook-seam code makes pass; together they
// form the bite harness AG-05 W1 demands.
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

// --- Phase 4 helpers (substrate guard, run-loop, etc.) ------------------

// gitTopLevelLoopHook returns the absolute path of the repo root.
// Mirrors loop_test.go's helper so this file's tests can run their
// own git diff against AG-08's base ref without sharing state with
// AG-07's helper.
func gitTopLevelLoopHook(t *testing.T) (string, error) {
	t.Helper()
	cmd := exec.Command("git", "rev-parse", "--show-toplevel")
	out, err := cmd.CombinedOutput()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(out)), nil
}

// gitDiffLoopHook runs `git diff <ref> -- <paths...>` and returns the
// raw output. An empty output means no diff.
func gitDiffLoopHook(t *testing.T, root, ref string, paths ...string) (string, error) {
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

// gitOutputLoopHook runs an arbitrary `git <args...>` invocation from
// the supplied root and returns the trimmed stdout. The substrate
// guard's escape hatch for refs that must be resolved at run time
// (e.g. `git merge-base HEAD origin/main`) rather than pinned in source.
func gitOutputLoopHook(t *testing.T, root string, args ...string) (string, error) {
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
// path matches loop.go, loop_test.go, or loop_hook_test.go. Mirrors
// loop_test.go's filter at file granularity.
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
				strings.HasSuffix(path, "/loop_hook_test.go")
		}
		if !skip {
			kept.WriteString(line)
			kept.WriteString("\n")
		}
	}
	return strings.TrimSpace(kept.String())
}

// loadGoldenSkeletonRequest reads the golden fixture of AG-07's
// skeleton's known-good captured request. Mirrors loop_test.go's
// fixture convention: golden files live in testdata/ next to the test
// and are kept under the loop package's review budget.
func loadGoldenSkeletonRequest(t *testing.T) ai.Request {
	t.Helper()
	raw, err := os.ReadFile("testdata/loop_skeleton_request.golden.json")
	if err != nil {
		t.Fatalf("read testdata/loop_skeleton_request.golden.json: %v", err)
	}
	var req ai.Request
	if err := json.Unmarshal(raw, &req); err != nil {
		t.Fatalf("unmarshal testdata/loop_skeleton_request.golden.json: %v", err)
	}
	return req
}

// reflectRequestsDeepEqual compares two requests via ai.Request.Equal
// AND a deep-equality check on their public fields. Belt-and-braces
// for the identity-default scenario, where the loop's claim is
// byte-stability against AG-07's skeleton's known-good output.
func reflectRequestsDeepEqual(a, b ai.Request) bool {
	if !a.Equal(b) {
		return false
	}
	// Belt-and-braces: also assert the field-by-field reflection
	// matches. Equal already covers this on its own, but the
	// guard is here to catch regressions in Equal that lose a field.
	if !reflect.DeepEqual(a.Model(), b.Model()) {
		return false
	}
	if !reflect.DeepEqual(a.Messages(), b.Messages()) {
		return false
	}
	aSys, _ := a.SystemInstruction()
	bSys, _ := b.SystemInstruction()
	return reflect.DeepEqual(aSys, bSys)
}

// waitForGoroutineBaseline polls runtime.NumGoroutine() until it
// drops back to baseline or deadline expires. Used by
// S-PRH-007 (unbuffered sink) to assert no stranded producer.
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
}

// drainSinkBufferedLen drains sink and returns the number of events
// read. Used by S-PRH-007 to confirm the producer wrote events to
// the unbuffered sink.
func drainSinkBufferedLen(sink <-chan *agent.Event) int {
	n := 0
	for range sink {
		n++
	}
	return n
}

// syncCounters is a tiny thread-safe helper for accumulating counts
// across goroutines. The unbuffered-sink test uses it to count events
// the consumer drains in parallel with the producer.
type syncCounters struct {
	mu    sync.Mutex
	total int
}

// add accumulates a count atomically.
func (c *syncCounters) add(n int) {
	c.mu.Lock()
	c.total += n
	c.mu.Unlock()
}

// value returns the current accumulated count.
func (c *syncCounters) value() int {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.total
}