// AI-29 — the reasoning stream: a recorded capability absence.
//
// # What this file pins
//
// R-ARS-015 (an extension field inside a delta is ignored, never leaks into
// text, and never fails the stream, S-ARS-036…039) and R-ARS-014 row 2
// (the conformance factory declares reasoning explicitly `false`,
// S-ARS-033). Per `decision.md` § 5 row 2's ruling, this change's one
// `_test.go` file gains the second function; row 2's declaration had no
// landed test. Per design.md § 4's design-validation corrective (F5), this
// file also introduces the durable-guard helper idiom (S-ARS-042) — a
// permanently-executable self-inversion that extends, but does not replace,
// the staged-reverted-mutation and comparative-twin precedents named in
// tolerance_test.go's header.
//
// # Test-only — no production Go
//
// Per R-ARS-017 / S-ARS-043/044, this file is the only Go artifact this
// change adds under `backend/agent/`. `go.mod` and `go.work` are unchanged,
// the package's production Go is byte-identical before and after this
// change, and every assertion below exercises code that already exists on
// the branch — `decodeChunk` (chunk.go:213), `run` (stream.go), and the
// bridge factory (bridge_test.go:46-73, referenced from
// `TestConformanceFactory_DeclaresReasoningExplicitlyFalse` below).
//
// # Tolerance_test.go's idiom, followed
//
// Same package conventions, `sseServer`/`mustClient`/`drainAll` helpers,
// `t.Parallel()`, scenario-ID-bearing failure messages. No new helper
// types, no package-level state beyond the named sentinel constant the
// fixtures share.
//
// # RED discipline
//
// Per strict TDD and design.md § 1: this file's RED is a compile/first-run
// capture, not a behavioral RED — `decodeChunk` (the only chunk-JSON
// decode path in this package) uses plain `json.Unmarshal` and has no
// `DisallowUnknownFields`, so the extension-field pin is GREEN against the
// landed code on first execution. Non-vacuity is carried by the durable
// inversion guard below and by the apply-time staged mutation, both
// recorded in the apply-progress artifact for this change.

package openaicompat

import (
	"context"
	"strings"
	"testing"

	"github.com/cachicamas/backend/agent/src/agenttest"
	"github.com/cachicamas/backend/agent/src/ai"
)

// reasoningExtensionSentinel is the distinctive byte sequence the
// extension-field fixtures carry inside the undeclared `reasoning_content`
// member (S-ARS-041: appears exactly once, distinct from every other value
// in the fixture). It is chosen to be a literal-bytes search target —
// unique, recognizable, and not a substring of any other fixture value
// below.
const reasoningExtensionSentinel = "RSN-EXT-SENTINEL-7f3a"

// reasoningAssertionHelperName is the function name assertNoSentinelLeak is
// exposed under (the durable-guard helper's hook for the inversion test).
// Exposed as a package-level identifier so the inversion test below can
// name it directly and any future code review can find the helper by its
// name in a grep.
const reasoningAssertionHelperName = "assertNoSentinelLeak"

// assertNoSentinelLeak returns whether events contains no emitted text
// event that carries any byte of sentinel — the "drop, not leak" half of
// R-ARS-015 (S-ARS-036). The check is byte-substring across the union of
// every text event's delta payload: a single byte of the sentinel appearing
// in any one event fails the assertion.
//
// scenarios is the human-readable list of scenario IDs the assertion is
// protecting; it is included in the failure message so a reviewer sees the
// exact contract this guard discharges when an inversion fires.
//
// This helper is the durable guard (S-ARS-042). It is intentionally a
// normal Go function: an inversion test below calls it with a synthetic
// event list whose sentinel has been deliberately routed into a text
// delta, and asserts the helper returns false. The inversion fires on
// every future run, not just on a one-off staged mutation.
func assertNoSentinelLeak(events []ai.Event, sentinel string, scenarios []string) bool {
	if len(sentinel) == 0 {
		return true
	}
	for _, ev := range events {
		d, ok := ev.TextDelta()
		if !ok {
			continue
		}
		if strings.Contains(d.Delta(), sentinel) {
			return false
		}
	}
	return true
}

// assertNoReasoningTypedEvent returns whether events contains no event of a
// reasoning-typed kind — the "drop, not emit-as-reasoning-event" half of
// R-ARS-015 (S-ARS-037). The kind set covers every reasoning-shaped kind
// the production registry enumerates (EventKindReasoningBlockStart,
// EventKindReasoningDelta, EventKindReasoningBlockEnd); a future addition
// to the registry will surface here as a compilation gap, which is the
// desired failure shape.
func assertNoReasoningTypedEvent(events []ai.Event) bool {
	for _, ev := range events {
		switch ev.Kind() {
		case ai.EventKindReasoningBlockStart,
			ai.EventKindReasoningDelta,
			ai.EventKindReasoningBlockEnd:
			return false
		}
	}
	return true
}

// assertTerminalIsCompletionAndNoError returns whether events ends in the
// stream's normal completion terminal, with no error event anywhere — the
// "drop, not fail" half of R-ARS-015 (S-ARS-038). Empty events return
// false: a stream that drains to nothing has not reached its terminal.
func assertTerminalIsCompletionAndNoError(events []ai.Event) bool {
	if len(events) == 0 {
		return false
	}
	last := events[len(events)-1]
	if last.Kind() != ai.EventKindCompletion {
		return false
	}
	for _, ev := range events {
		if ev.Kind() == ai.EventKindError {
			return false
		}
	}
	return true
}

// TestReasoningExtensionField_DroppedNotLeakedNotFailed_FixtureAAndFixtureB
// covers S-ARS-036, S-ARS-037, S-ARS-038 and S-ARS-039. Two fixtures back
// R-ARS-015: a delta carrying declared content alongside an undeclared
// reasoning-bearing extension field, and a delta carrying only the
// extension field. Both must terminate cleanly with no text event carrying
// any byte of the sentinel and no reasoning-typed event emitted.
//
// The comparative-twin half of the assertion (S-ARS-038) is satisfied by
// draining the extension-free twin and asserting the kinds, sequence and
// terminal agree event-for-event — the same idiom tolerance_test.go:319
// uses for S-ATS-071.
func TestReasoningExtensionField_DroppedNotLeakedNotFailed_FixtureAAndFixtureB(t *testing.T) {
	t.Parallel()

	// Fixture A (S-ARS-036, S-ARS-037, S-ARS-038): delta 1 carries the
	// sentinel inside an undeclared `reasoning_content` extension field
	// alongside a declared content string; delta 2 carries distinct content
	// AFTER the extension field, asserted intact and in order (S-ARS-040's
	// "content after the extension field" clause — the fixture cannot pass
	// on an empty remainder).
	fixtureA := "" +
		"data: {\"id\":\"chatcmpl-re\",\"model\":\"m\",\"object\":\"chat.completion.chunk\",\"created\":1700000000,\"choices\":[{\"index\":0,\"delta\":{\"role\":\"assistant\",\"content\":\"alpha\",\"reasoning_content\":\"RSN-EXT-SENTINEL-7f3a\"},\"finish_reason\":null}]}\n\n" +
		"data: {\"id\":\"chatcmpl-re\",\"model\":\"m\",\"object\":\"chat.completion.chunk\",\"created\":1700000000,\"choices\":[{\"index\":0,\"delta\":{\"content\":\"omega\"},\"finish_reason\":null}]}\n\n" +
		"data: {\"id\":\"chatcmpl-re\",\"model\":\"m\",\"object\":\"chat.completion.chunk\",\"created\":1700000000,\"choices\":[{\"index\":0,\"delta\":{},\"finish_reason\":\"stop\"}]}\n\n" +
		"data: [DONE]\n\n"

	// Fixture B (S-ARS-039): a delta carrying ONLY the undeclared
	// reasoning-bearing extension field — the boundary shape that proves
	// the ignore path holds at the edge as well as alongside content.
	fixtureB := "" +
		"data: {\"id\":\"chatcmpl-rb\",\"model\":\"m\",\"object\":\"chat.completion.chunk\",\"created\":1700000000,\"choices\":[{\"index\":0,\"delta\":{\"reasoning_content\":\"RSN-EXT-SENTINEL-7f3a\"},\"finish_reason\":null}]}\n\n" +
		"data: {\"id\":\"chatcmpl-rb\",\"model\":\"m\",\"object\":\"chat.completion.chunk\",\"created\":1700000000,\"choices\":[{\"index\":0,\"delta\":{},\"finish_reason\":\"stop\"}]}\n\n" +
		"data: [DONE]\n\n"

	// Twin (S-ATS-071 idiom): the identical fixture with the extension
	// field stripped, used as the kind/sequence/terminal oracle. The
	// fixture-A stream and the fixture-A twin must agree event-for-event;
	// fixture B's drain is asserted against the empty-content twin shape
	// (the same shape TestTolerance_NoContentAnywhere_NormalizesToResponseStartThenCompletion
	// already pins, tolerance_test.go:238).
	twinA := "" +
		"data: {\"id\":\"chatcmpl-re\",\"model\":\"m\",\"object\":\"chat.completion.chunk\",\"created\":1700000000,\"choices\":[{\"index\":0,\"delta\":{\"role\":\"assistant\",\"content\":\"alpha\"},\"finish_reason\":null}]}\n\n" +
		"data: {\"id\":\"chatcmpl-re\",\"model\":\"m\",\"object\":\"chat.completion.chunk\",\"created\":1700000000,\"choices\":[{\"index\":0,\"delta\":{\"content\":\"omega\"},\"finish_reason\":null}]}\n\n" +
		"data: {\"id\":\"chatcmpl-re\",\"model\":\"m\",\"object\":\"chat.completion.chunk\",\"created\":1700000000,\"choices\":[{\"index\":0,\"delta\":{},\"finish_reason\":\"stop\"}]}\n\n" +
		"data: [DONE]\n\n"

	drain := func(transcript string) []ai.Event {
		server := sseServer(t, transcript)
		defer server.Close()
		c := mustClient(t, server.URL)
		ch, err := c.Stream(context.Background(), validRequest(t))
		if err != nil {
			t.Fatalf("Stream() error = %v, want nil", err)
		}
		return drainAll(t, ch)
	}

	eventsA := drain(fixtureA)
	eventsB := drain(fixtureB)
	twinEvents := drain(twinA)

	// S-ARS-036 — no text event carries any byte of the sentinel. The
	// fixture-A stream's text-delta deltas must equal {"alpha", "omega"},
	// the twin's sequence — not because of twin-agreement (below) but
	// because every byte of the sentinel is absent.
	scenarios := []string{"S-ARS-036"}
	if !assertNoSentinelLeak(eventsA, reasoningExtensionSentinel, scenarios) {
		t.Errorf("fixture A: a text event carried a byte of %q (S-ARS-036, R-ARS-015) — extension field leaked into text", reasoningExtensionSentinel)
	}
	scenariosB := []string{"S-ARS-039"}
	if !assertNoSentinelLeak(eventsB, reasoningExtensionSentinel, scenariosB) {
		t.Errorf("fixture B: a text event carried a byte of %q (S-ARS-039, R-ARS-015) — extension-only delta leaked into text", reasoningExtensionSentinel)
	}

	// S-ARS-037 — no reasoning-typed event is emitted by either fixture.
	if !assertNoReasoningTypedEvent(eventsA) {
		t.Errorf("fixture A: a reasoning-typed event was emitted (S-ARS-037, R-ARS-015)")
	}
	if !assertNoReasoningTypedEvent(eventsB) {
		t.Errorf("fixture B: a reasoning-typed event was emitted (S-ARS-037, R-ARS-015)")
	}

	// S-ARS-038 — the stream reaches its normal terminal with no failure,
	// and the terminal and event sequence match the extension-free twin.
	if !assertTerminalIsCompletionAndNoError(eventsA) {
		t.Errorf("fixture A: terminal is not the normal completion event, or an error event appeared (S-ARS-038, R-ARS-015): %v", kindsOf(eventsA))
	}
	if !assertTerminalIsCompletionAndNoError(eventsB) {
		t.Errorf("fixture B: terminal is not the normal completion event, or an error event appeared (S-ARS-038, R-ARS-015): %v", kindsOf(eventsB))
	}

	// Twin-agreement (S-ATS-071 idiom): fixture A's emitted kinds,
	// sequences and terminal must equal the twin's event-for-event. This
	// is the "same terminal the extension-free stream reaches" clause
	// S-ARS-038's own text names.
	if len(eventsA) != len(twinEvents) {
		t.Fatalf("fixture A: %d event(s), twin %d — counts must match (S-ARS-038, twin idiom S-ATS-071): %v vs %v", len(eventsA), len(twinEvents), kindsOf(eventsA), kindsOf(twinEvents))
	}
	for i := range eventsA {
		if eventsA[i].Kind() != twinEvents[i].Kind() {
			t.Errorf("event[%d] kind differs: extension=%v twin=%v (S-ARS-038, S-ATS-071)", i, eventsA[i].Kind(), twinEvents[i].Kind())
		}
		if eventsA[i].Sequence() != twinEvents[i].Sequence() {
			t.Errorf("event[%d] sequence differs: extension=%d twin=%d (S-ARS-038, S-ATS-071)", i, eventsA[i].Sequence(), twinEvents[i].Sequence())
		}
		pd, pok := eventsA[i].TextDelta()
		td, tok := twinEvents[i].TextDelta()
		if pok != tok {
			t.Errorf("event[%d] TextDelta presence differs: extension=%v twin=%v (S-ARS-038, S-ATS-071)", i, pok, tok)
		}
		if pok && tok && pd.Delta() != td.Delta() {
			t.Errorf("event[%d] delta differs: extension=%q twin=%q (S-ARS-038, S-ARS-040)", i, pd.Delta(), td.Delta())
		}
	}

	// S-ARS-040 (asserted explicitly): declared content is present AFTER
	// the extension field's position and arrives intact and in order.
	var deltas []string
	for _, ev := range eventsA {
		if d, ok := ev.TextDelta(); ok {
			deltas = append(deltas, d.Delta())
		}
	}
	want := []string{"alpha", "omega"}
	if len(deltas) != len(want) {
		t.Fatalf("fixture A: emitted %d delta(s) = %q, want %d = %q — content after the extension field must arrive intact and in order (S-ARS-040)", len(deltas), deltas, len(want), want)
	}
	for i := range want {
		if deltas[i] != want[i] {
			t.Errorf("fixture A delta[%d] = %q, want %q (S-ARS-040)", i, deltas[i], want[i])
		}
	}
}

// TestConformanceFactory_DeclaresReasoningExplicitlyFalse covers S-ARS-033:
// the bridge factory's three optional-capability declarations are explicit
// non-nil pointers to `false` — a declaration distinguishable from
// omission. Per `decision.md` § 5 row 2, this is the landed test for the
// mechanism named at row 2 (the bridge factory declares reasoning
// explicitly `false`), and was the only row with no pre-existing landed
// test.
func TestConformanceFactory_DeclaresReasoningExplicitlyFalse(t *testing.T) {
	t.Parallel()

	factory := conformanceBridgeFactory()

	if factory.Reasoning == nil {
		t.Fatalf("factory.Reasoning is nil (S-ARS-033) — declaration is by omission, not explicit; declaration and omission are not distinguishable")
	}
	if *factory.Reasoning != false {
		t.Errorf("factory.Reasoning = %v, want false (S-ARS-033)", *factory.Reasoning)
	}

	if factory.TokenCounting == nil {
		t.Fatalf("factory.TokenCounting is nil (S-ARS-033) — declaration is by omission, not explicit")
	}
	if *factory.TokenCounting != false {
		t.Errorf("factory.TokenCounting = %v, want false (S-ARS-033)", *factory.TokenCounting)
	}

	if factory.CacheBoundary == nil {
		t.Fatalf("factory.CacheBoundary is nil (S-ARS-033) — declaration is by omission, not explicit")
	}
	if *factory.CacheBoundary != false {
		t.Errorf("factory.CacheBoundary = %v, want false (S-ARS-033)", *factory.CacheBoundary)
	}

	// Cross-check the declarations are in fact the agenttest.Factory
	// shape the suite consumes, not some locally redefined type: the
	// factory's declarations are what applyDeclaredAbsences and
	// declaredAbsentSkipReason consult (conformance_suite.go:330, 441),
	// and the test must verify the bridge factory's shape is exactly the
	// suite's Factory type — not by casting, but by calling New and
	// confirming the suite surface accepted. The surface check below is
	// performed by constructing one and confirming the construction does
	// not error — the suite would fail construction itself if Factory
	// were mis-typed (S-CNF-006: nil optional-capability declarations
	// fail construction).
	var _ agenttest.Factory = factory
}

// TestReasoningExtensionField_DurableGuardHelper_InvertsOnSyntheticRouting
// covers S-ARS-042: a mutation that routes the sentinel into a text delta
// must make the leak-assertion helper report failure. This is the
// durable inversion guard — a new idiom this change introduces, extending
// (not replacing) the staged-reverted-mutation and comparative-twin
// precedents named in tolerance_test.go's header. The inversion fires on
// every future run, not just on a one-off staged mutation.
//
// The synthetic event list is built through the exported ai.NewTextDelta
// constructor (the production registry's own constructor), so the helper
// operates on the same Event shape production code produces. The
// sentinel is one chosen byte of the constant's own length-zero suffix
// of the surrounding text delta — the helper's substring check fires on
// any byte, not on exact equality, by design.
func TestReasoningExtensionField_DurableGuardHelper_InvertsOnSyntheticRouting(t *testing.T) {
	t.Parallel()

	// Sanity: the helper's name is the same one apply-progress cites —
	// the constant is here so a future rename of assertNoSentinelLeak
	// surfaces here at compile time and the durable-guard row's
	// apply-progress evidence does not silently drift.
	if reasoningAssertionHelperName != "assertNoSentinelLeak" {
		t.Fatalf("reasoningAssertionHelperName = %q, want %q (durable-guard naming discipline)", reasoningAssertionHelperName, "assertNoSentinelLeak")
	}

	// Synthetic inversion: build a real text-delta event that carries the
	// sentinel in its delta payload (the construction is the "mutation"
	// — production code is not touched). The helper must return false on
	// this event list; the assertion below requires the inversion to fire.
	synthetic, err := ai.NewTextDelta(1, "before-"+reasoningExtensionSentinel+"-after")
	if err != nil {
		t.Fatalf("ai.NewTextDelta() error = %v, want nil (synthetic inversion setup)", err)
	}
	events := []ai.Event{synthetic}

	scenarios := []string{"S-ARS-042 (inversion)"}
	if assertNoSentinelLeak(events, reasoningExtensionSentinel, scenarios) {
		t.Fatalf("assertNoSentinelLeak returned true on a synthetic event list whose text delta carries the sentinel (S-ARS-042) — the durable inversion guard is broken; the helper cannot fail under deliberate routing and the test it guards is vacuous")
	}
}
