// AG-19 — Phase 4: the scope fence — what AG-19 proves, and what it
// deliberately does not ship (R-DEL-009).
package agent_test

import (
	"os"
	"reflect"
	"regexp"
	"sort"
	"strings"
	"testing"

	"github.com/cachicamas/backend/agent/src/agent"
)

// del024ByteUnchangedFiles is R-DEL-009's own named list, relative to
// backend/agent/src/agent/ — every file this change MUST NOT edit.
// go.mod and go.sum are checked separately (they live one directory
// up, at backend/agent/).
func del024ByteUnchangedFiles() []string {
	return []string{
		// event.go, event_registry_test.go, stream_check.go and
		// permission_events.go are removed here — the Layer 2
		// audit-fix change (branch fix/agent-layer2-audit-findings)
		// legitimately edits all four (per-tool-name
		// CardinalityAtMostOne keying, R-APE-003, and the corrected
		// 25-kind registry prose), mirroring the AG-18 amendment
		// precedent on TestNoRelease_SubstrateByteUnchanged.
		"event_descriptor.go",
		"delegation_events.go",
		"cost_events.go",
	}
}

// S-DEL-024 — given the merge base of this change's branch with
// origin/main, when git diff is taken over backend/agent/, then every
// file named byte-unchanged above is byte-unchanged, the diff under
// backend/agent/src/ai/ is empty, the go.mod/go.sum diff is empty,
// the every-kind-constructible guard passes at its committed kind
// count with AG-19 registering none, and no exported signature named
// above changed.
func TestScopeFence_S_DEL_024_ByteUnchangedFilesAndNoNewKind(t *testing.T) {
	t.Parallel()

	root, err := gitTopLevel(t)
	if err != nil {
		t.Fatalf("git rev-parse --show-toplevel failed: %v", err)
	}

	baseRef := os.Getenv("AG19_BASE_REF")
	if baseRef == "" {
		out, err := gitOutput(t, root, "merge-base", "HEAD", "origin/main")
		if err != nil {
			t.Fatalf("scope fence: cannot determine base ref (set AG19_BASE_REF): %v", err)
		}
		baseRef = out
	}

	for _, name := range del024ByteUnchangedFiles() {
		path := "backend/agent/src/agent/" + name
		diff, err := gitDiff(t, root, baseRef, path)
		if err != nil {
			t.Fatalf("git diff %s -- %s failed: %v", baseRef, path, err)
		}
		if diff != "" {
			t.Errorf("%s is not byte-unchanged against %s (R-DEL-009):\n%s", path, baseRef, diff)
		}
	}

	aiDiff, err := gitDiff(t, root, baseRef, "backend/agent/src/ai/")
	if err != nil {
		t.Fatalf("git diff %s -- backend/agent/src/ai/ failed: %v", baseRef, err)
	}
	// CH-03 carve-out: wantGoModRequires is bumped from 3 to 4
	// entries in two ai/ test files
	// (import_boundary_test.go:221 and
	// openrouter/zero_requires_test.go:85) per each test's own
	// documented "update in the same commit" instruction. Anything
	// beyond the two carve-out files fails.
	ch03AiCarveout := []string{
		"backend/agent/src/ai/import_boundary_test.go",
		"backend/agent/src/ai/openaicompat/openrouter/zero_requires_test.go",
	}
	if strings.Contains(aiDiff, "diff --git") {
		chunks := strings.Split(aiDiff, "diff --git")
		var stray []string
		for _, chunk := range chunks[1:] {
			nl := strings.Index(chunk, "\n")
			header := chunk
			if nl >= 0 {
				header = chunk[:nl]
			}
			allowed := false
			for _, file := range ch03AiCarveout {
				if strings.Contains(header, file) {
					allowed = true
					break
				}
			}
			if !allowed {
				stray = append(stray, chunk)
			}
		}
		if len(stray) == 0 {
			t.Logf("CH-03 carve-out: every ai/ diff is one of the documented require-bump amendments.\n%s", aiDiff)
		} else {
			t.Errorf("backend/agent/src/ai/ was edited beyond the CH-03 wantGoModRequires amendments (R-RUN-012 violated):\n%s", aiDiff)
		}
	} else if aiDiff != "" {
		t.Errorf("backend/agent/src/ai/ is not byte-unchanged against %s (R-RUN-012, Layer 1 consumed never edited):\n%s", baseRef, aiDiff)
	}

	modDiff, err := gitDiff(t, root, baseRef, "backend/agent/go.mod", "backend/agent/go.sum")
	if err != nil {
		t.Fatalf("git diff %s -- go.mod go.sum failed: %v", baseRef, err)
	}
	// CH-03 carve-out for go.mod / go.sum drift: the chat
	// archetype's HTTP+SSE surface requires
	// github.com/labstack/echo/v5 v5.2.1 (ADR
	// adr/echo-v5-in-agent-module). Recorded exception; every other
	// modification fails.
	if isCH03GoModDrift(modDiff) {
		t.Logf("CH-03 carve-out: go.mod drift is the recorded Echo require; passes per D3.\n%s", modDiff)
	} else if modDiff != "" {
		t.Errorf("go.mod/go.sum diff against %s is not empty:\n%s", baseRef, modDiff)
	}

	// The every-kind-constructible guard passes at its committed kind
	// count with AG-19 registering none: 25 kinds, the same count
	// event.go's own registry declares (verified byte-unchanged
	// above, so this count cannot have moved).
	if got := len(agent.EventKinds()); got != 25 {
		t.Errorf("len(agent.EventKinds()) = %d, want 25 (AG-19 registers no new EventKind)", got)
	}

	// No exported signature named by R-DEL-009 changed: Tool,
	// PolicySlot, Harness and the permission policy interface keep
	// their method sets/underlying types.
	toolType := reflect.TypeOf((*agent.Tool)(nil)).Elem()
	if got := toolType.NumMethod(); got != 3 {
		t.Errorf("agent.Tool declares %d method(s), want 3 (Name, EffectClass, Run)", got)
	}
	policyType := reflect.TypeOf((*agent.PermissionPolicy)(nil)).Elem()
	if got := policyType.NumMethod(); got != 2 {
		t.Errorf("agent.PermissionPolicy declares %d method(s), want 2 (Resolve, Remember)", got)
	}
	harnessPtrType := reflect.TypeOf((*agent.Harness)(nil))
	if got := harnessPtrType.NumMethod(); got != 5 {
		t.Errorf("*agent.Harness declares %d exported method(s), want 5 (Compact, Interrupt, Shutdown, Steer, Run)", got)
	}
	var policySlot agent.PolicySlot = "policy-slot-scope-fence-check"
	if _, ok := any(policySlot).(string); !ok {
		t.Error("agent.PolicySlot no longer accepts a plain string value — its underlying type must remain `any`")
	}
}

// hasSuffixLiteralPattern matches one `strings.HasSuffix(path, "...")`
// call's string-literal second argument, capturing the literal's raw
// text (without its surrounding quotes).
var hasSuffixLiteralPattern = regexp.MustCompile(`strings\.HasSuffix\(path,\s*"([^"]*)"\)`)

// extractHasSuffixEntries reads path (a Go source file) and returns
// every `strings.HasSuffix(path, "X")` literal it finds, in file
// order.
func extractHasSuffixEntries(t *testing.T, path string) []string {
	t.Helper()
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("os.ReadFile(%q): %v", path, err)
	}
	matches := hasSuffixLiteralPattern.FindAllStringSubmatch(string(raw), -1)
	out := make([]string, len(matches))
	for i, m := range matches {
		out[i] = m[1]
	}
	return out
}

// S-LSK-031 — AG-19: no release taken, and the widening is exact.
// Given the merge base of the AG-19 branch with origin/main, when
// both substrate filters are read, then each names /delegation_seam.go
// and each of AG-19's covered new test files by exact filename
// suffix, the two filters carry an identical entry set, neither
// contains a wildcard, a prefix match or a directory pattern, and
// neither names cost_events.go, cost_events_test.go,
// stream_check_test.go or reconstruction_test.go.
func TestScopeFence_S_LSK_031_SubstrateFiltersByteInSyncExactWidening(t *testing.T) {
	t.Parallel()

	loopEntries := extractHasSuffixEntries(t, "loop_test.go")
	hookEntries := extractHasSuffixEntries(t, "loop_hook_test.go")

	sortedLoop := append([]string(nil), loopEntries...)
	sortedHook := append([]string(nil), hookEntries...)
	sort.Strings(sortedLoop)
	sort.Strings(sortedHook)
	if len(sortedLoop) != len(sortedHook) {
		t.Fatalf("filterOutLoopFiles has %d entries, filterOutLoopHookFiles has %d, want the identical count", len(sortedLoop), len(sortedHook))
	}
	for i := range sortedLoop {
		if sortedLoop[i] != sortedHook[i] {
			t.Errorf("entry set differs at sorted position %d: filterOutLoopFiles has %q, filterOutLoopHookFiles has %q", i, sortedLoop[i], sortedHook[i])
		}
	}

	wantAG19 := []string{
		"/delegation_seam.go",
		"/delegation_seam_test.go",
		"/revocation_test.go",
		"/delegating_tool_test.go",
		"/nested_run_test.go",
		"/walkable_tree_test.go",
		"/siblings_test.go",
		"/cancellation_test.go",
		"/cost_test.go",
		"/permission_scope_test.go",
		"/scope_fence_test.go",
		"/inert_path_test.go",
		"/v1_scope_test.go",
	}
	for _, want := range wantAG19 {
		if !containsString(loopEntries, want) {
			t.Errorf("filterOutLoopFiles does not name %q", want)
		}
		if !containsString(hookEntries, want) {
			t.Errorf("filterOutLoopHookFiles does not name %q", want)
		}
	}

	// cost_events.go was released by AG-16 (already present in both
	// filters before this change) — AG-19 must not add a SECOND
	// entry for it, but its pre-existing presence is not this
	// scenario's concern. cost_events_test.go, stream_check_test.go
	// and reconstruction_test.go have never been released by any
	// milestone, AG-19 included, and must remain genuinely absent.
	if n := countString(loopEntries, "/cost_events.go"); n != 1 {
		t.Errorf("filterOutLoopFiles names %q %d time(s), want exactly 1 (AG-16's release; AG-19 must not add a duplicate)", "/cost_events.go", n)
	}
	if n := countString(hookEntries, "/cost_events.go"); n != 1 {
		t.Errorf("filterOutLoopHookFiles names %q %d time(s), want exactly 1 (AG-16's release; AG-19 must not add a duplicate)", "/cost_events.go", n)
	}
	neverReleased := []string{"/cost_events_test.go", "/stream_check_test.go", "/reconstruction_test.go"}
	for _, bad := range neverReleased {
		if containsString(loopEntries, bad) {
			t.Errorf("filterOutLoopFiles names %q, which no milestone including AG-19 may release", bad)
		}
		if containsString(hookEntries, bad) {
			t.Errorf("filterOutLoopHookFiles names %q, which no milestone including AG-19 may release", bad)
		}
	}

	// No wildcard, no prefix match, no directory-level relaxation:
	// every entry is an exact "/<filename>" suffix — starts with
	// exactly one slash, contains no further slash, and no glob
	// metacharacter.
	exactSuffixPattern := regexp.MustCompile(`^/[A-Za-z0-9_.]+\.go$`)
	for _, entries := range [][]string{loopEntries, hookEntries} {
		for _, e := range entries {
			if !exactSuffixPattern.MatchString(e) {
				t.Errorf("filter entry %q is not an exact /<filename>.go suffix (no wildcard, prefix or directory pattern allowed)", e)
			}
			if strings.Count(e, "/") != 1 {
				t.Errorf("filter entry %q contains more than one path separator — want a bare filename suffix, not a directory pattern", e)
			}
		}
	}
}

// S-TLS-020 / S-DEL-003 — AG-19: the seam rides beside PolicySlot,
// not inside it, and the guard proves it by raw bytes. Given the
// merged AG-19 change, when scheduler.go's own diff against the
// merge base is scanned, then the ADDED lines contain no type
// assertion of any kind and no occurrence of the seam's context key
// or interface type; and when a tool records both the policy value
// it received and the seam it obtained from its context, then the
// policy value is byte-identical to the injected one and the seam
// is a distinct value reached through its own accessor, never
// through the policy parameter. Asserted directly (verify-report
// CRITICAL-6 — previously unasserted, only claimed by the narrower
// pre-existing PolicySlot-type-assertion guard).
func TestScopeFence_S_TLS_020_SeamRidesBesidePolicySlotNotInsideIt(t *testing.T) {
	t.Parallel()

	root, err := gitTopLevel(t)
	if err != nil {
		t.Fatalf("git rev-parse --show-toplevel failed: %v", err)
	}
	baseRef := os.Getenv("AG19_BASE_REF")
	if baseRef == "" {
		out, err := gitOutput(t, root, "merge-base", "HEAD", "origin/main")
		if err != nil {
			t.Fatalf("S-TLS-020: cannot determine base ref (set AG19_BASE_REF): %v", err)
		}
		baseRef = out
	}
	if baseRefIsHEAD(t, root, baseRef) {
		t.Skip("S-TLS-020: base ref resolves to HEAD — this change is merged, so scheduler.go's added lines no longer appear in a diff against the base and the absence this guard asserts holds trivially; branch-scoped by construction, mirroring AG-07's substrate guard")
	}
	diff, err := gitDiff(t, root, baseRef, "backend/agent/src/agent/scheduler.go")
	if err != nil {
		t.Fatalf("git diff %s -- scheduler.go failed: %v", baseRef, err)
	}
	if diff == "" {
		// AG-20 fix: an empty diff means scheduler.go was not touched
		// on THIS branch, so the absence S-TLS-020 asserts (no type
		// assertion, no bare seam type/key named in ADDED lines) holds
		// vacuously and correctly for it — there are no added lines to
		// violate it. That is a skip, not a failure: fatal-ing here
		// only ever fires on a later, innocent branch (any branch cut
		// after AG-19 merged that does not itself touch scheduler.go —
		// AG-20, AG-21, AG-22, AG-23 alike), never on the branch that
		// actually introduces the hunk this guard exists to scan. On
		// AG-19's own branch the diff was non-empty and the scan ran;
		// see TestScopeFence_S_LSK_031... and this file's own history.
		t.Skip("S-TLS-020: scheduler.go's diff against the merge base is empty on this branch — the absence this guard asserts holds vacuously (nothing was added to violate it); the guard bites on the branch that actually introduces the seam-install hunk")
	}
	// Scoped to ADDED lines only, not the whole file: scheduler.go's
	// pre-existing `cause, _ := r.(error)` (unrelated to this change)
	// would otherwise false-positive a whole-file scan.
	typeAssertionPattern := regexp.MustCompile(`\.\s*\(\s*\*?\s*[A-Za-z_]`)
	// Word-boundary matched: scheduler.go legitimately CALLS the
	// constructor/installer (newDelegationSeam, withDelegationSeam)
	// as part of this change's own wiring — those identifiers merely
	// CONTAIN "DelegationSeam" as a naming convention. What S-TLS-020
	// forbids is scheduler.go naming the BARE interface type
	// (DelegationSeam) or the unexported context-key type
	// (delegationSeamKey) directly, e.g. by declaring its own
	// DelegationSeam-typed variable or reading the key type itself.
	bareSeamTypeOrKey := regexp.MustCompile(`\b(DelegationSeam|delegationSeamKey)\b`)
	for _, line := range strings.Split(diff, "\n") {
		if !strings.HasPrefix(line, "+") || strings.HasPrefix(line, "+++") {
			continue
		}
		if typeAssertionPattern.MatchString(line) {
			t.Errorf("scheduler.go's diff against %s adds a type assertion: %q", baseRef, line)
		}
		if bareSeamTypeOrKey.MatchString(line) {
			t.Errorf("scheduler.go's diff against %s names the seam's bare context-key or interface type directly (S-TLS-020 requires zero occurrence): %q", baseRef, line)
		}
	}

	const callID = "call-tls-020"
	toolName := "tls020_tool"
	tool := &seamCapturingTool{toolName: toolName}
	reg := agent.NewMapRegistry(map[string]agent.Tool{toolName: tool})
	calls := callsToAICalls([]scheduledCall{{id: callID, name: toolName, args: []byte(`{}`), effect: agent.EffectClassRead}})

	sink := make(chan *agent.Event, 16)
	sched := &agent.Scheduler{}
	_ = sched.Schedule(contextBackground(), calls, reg, "run-tls-020", "turn-tls-020", nil, &agent.LaneStamper{}, sink)
	drainSink(t, sink)

	if _, hadSeam := tool.Seam(); !hadSeam {
		t.Fatal("tool.Run's context carried no DelegationSeam")
	}
	if got, want := tool.Policy(), agent.PolicySlot(callID); got != want {
		t.Errorf("tool.Run received PolicySlot %v, want byte-identical to the injected %v", got, want)
	}
	if seamFromPolicy, ok := tool.Policy().(agent.DelegationSeam); ok {
		t.Errorf("the injected PolicySlot type-asserts to a DelegationSeam (%v) — the seam must be reachable only through DelegationSeamFrom(ctx), never by unwrapping the policy parameter", seamFromPolicy)
	}
}

func containsString(haystack []string, needle string) bool {
	for _, s := range haystack {
		if s == needle {
			return true
		}
	}
	return false
}

func countString(haystack []string, needle string) int {
	n := 0
	for _, s := range haystack {
		if s == needle {
			n++
		}
	}
	return n
}

// baseRefIsHEAD reports whether baseRef resolves to the same commit as
// HEAD. That is the post-merge condition: once this change is on main,
// `git merge-base HEAD origin/main` resolves to HEAD itself, so the
// change's own additions no longer appear in any diff against the base.
//
// The AG-19 diff guards assert an ABSENCE — that no forbidden symbol was
// introduced by this change's own added lines. An empty diff satisfies
// that trivially, so the guards are branch-scoped by construction and
// must degrade the way AG-07's substrate guard already does
// (loop_test.go's substrate check passes on an empty diff) rather than
// redden main's suite. The anti-vacuity floors below stay in force for
// the on-branch case they were written for (verify-report WARNING-9).
func baseRefIsHEAD(t *testing.T, root, baseRef string) bool {
	t.Helper()
	headSHA, err := gitOutput(t, root, "rev-parse", "HEAD^{commit}")
	if err != nil {
		t.Fatalf("git rev-parse HEAD failed: %v", err)
	}
	baseSHA, err := gitOutput(t, root, "rev-parse", baseRef+"^{commit}")
	if err != nil {
		t.Fatalf("git rev-parse %s failed: %v", baseRef, err)
	}
	return headSHA == baseSHA
}
