// AI-38 — the boundary-replay sweep: every eligible conformance transcript
// replayed split at every admitted byte offset, asserting an identical
// drained event sequence to the canonical unsplit replay at the
// integration level (a real *openaicompat.Client over real HTTP, not the
// decoder called directly), plus a canonical anchor against a hard-coded
// expected event list so a uniformly broken harness cannot pass
// vacuously (design D8, R-ACR-005, tasks.md Phase 7 WU8).
//
// AI-27.3's own offset_sweep_test.go already owns the DECODER level
// (openaicompat.NewDecoder.Feed called directly); this sweep proves the
// same offset-robustness claim one layer up, through the real HTTP
// client — T5's own "AI-27.3 already owns the decoder level" rationale.
//
// # Fixture eligibility (S-ACR-016's size-bound clause)
//
// Every transcript this sweep replays MUST be at most
// conformanceSweepMaxBytes, inclusive — checkConformanceSweepBound proves
// this over the curated set below, mirroring openaicompat_test's own
// unexported checkSweepFixtureBound (identical shape, identical inclusive
// bound). fixtures.OpenrouterToolCallStream() is 1073 bytes — 49 over the
// 1024-byte bound — and is therefore NOT in this sweep's candidate set;
// text_stream (921B), with_usage (910B) and reasoning_extension (680B)
// all fit. This mirrors the AI-27.3 precedent exactly: an oversized
// fixture is structurally excluded from the exhaustive sweep rather than
// relying on convention (R-ASD-014's own posture, restated for this
// milestone's own bridge). Excluding tool_call.sse from THIS sweep does
// not remove its OWN coverage: it is still exercised end-to-end by
// TestOpenRouterAdapter_ToolCalls and the unscoped run.
//
// # Two tiers (design D8)
//
// Tier 1 (event-level, exhaustive): for each eligible transcript, every
// offset 0…len(transcript) inclusive is replayed split in two writes
// through a real httptest.Server + real *openaicompat.Client; the
// drained events must equal the canonical (unsplit) replay's, via
// agenttest.RequireSameEvents.
//
// Tier 2 (record-level, sampled {1, len/2, len-1}): design D8's own
// literal choice for this tier — NOT a fallback contingent on tier 1's
// measured runtime (see the runtime-measurement note on
// TestBoundarySweep_EventLevel_EveryOffsetMatchesCanonical below for the
// reconciliation with tasks.md's phrasing). Each of the three relative
// offset strategies is applied GLOBALLY across one full unscoped
// agenttest.RunConformance run — via conformanceBridgeFactorySplitAt, a
// variant of conformanceBridgeFactory whose server splits EVERY
// transcript any case in the suite constructs at that strategy's
// relative offset — and the returned record must still equal
// expectedOpenRouterRecord() every time.

//nolint:revive // underscore in package name per task plan § PR #2 2.1: "package openrouter_conformance"
package openrouter_conformance

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"

	"github.com/cachicamas/backend/agent/src/agenttest"
	"github.com/cachicamas/backend/agent/src/ai"
	"github.com/cachicamas/backend/agent/src/ai/openaicompat"
	"github.com/cachicamas/backend/agent/src/ai/openaicompat/openrouter/conformance/fixtures"
)

// conformanceSweepMaxBytes is this sweep's fixture bound (S-ACR-016):
// inclusive, mirroring openaicompat_test's maxSweepFixtureBytes exactly
// (same value, same rationale — a bound this sweep's own
// checkConformanceSweepBound enforces mechanically, not by convention).
const conformanceSweepMaxBytes = 1024

// conformanceSweepTranscript names one candidate transcript this sweep
// may replay: its name, the raw bytes, and — for at most one entry — a
// hard-coded expected event list anchoring its canonical replay
// (S-ACR-015's vacuity guard). want is nil for every non-anchored entry.
type conformanceSweepTranscript struct {
	name       string
	transcript []byte
	want       []ai.Event
}

// conformanceSweepTranscripts is the curated candidate set (S-ACR-016):
// every fixtures.Openrouter* transcript that fits conformanceSweepMaxBytes.
// fixtures.OpenrouterToolCallStream() (1073B) is deliberately absent —
// see this file's header comment.
func conformanceSweepTranscripts(tb testing.TB) []conformanceSweepTranscript {
	tb.Helper()
	return []conformanceSweepTranscript{
		{
			name:       "text_stream",
			transcript: []byte(fixtures.OpenrouterTextStream()),
			want:       conformanceSweepTextStreamAnchor(tb),
		},
		{
			name:       "with_usage",
			transcript: []byte(fixtures.OpenrouterTextStreamWithUsage()),
		},
		{
			name:       "reasoning_extension",
			transcript: []byte(fixtures.OpenrouterReasoningExtensionStream()),
		},
	}
}

// conformanceSweepRequireConstructed fails tb, naming the constructor,
// when err is non-nil — this file's own copy of the suite's
// requireConstructed idiom (agenttest's own is unexported).
func conformanceSweepRequireConstructed(tb testing.TB, err error, constructor string) {
	tb.Helper()
	if err != nil {
		tb.Fatalf("boundary_sweep_test: %s returned %v, want no failure", constructor, err)
	}
}

// conformanceSweepTextStreamAnchor is text_stream's hard-coded expected
// event list (S-ACR-015): a real decode of fixtures.OpenrouterTextStream()
// dumped once via TestBoundarySweep_DumpCanonicalEvents (temporary,
// removed once this list was pinned) and hand-verified against the
// canonical script text_stream.go documents. Stamped with a fresh
// ai.Stamper so sequence numbers match what a genuinely fresh stream
// produces — the same stamping fake_provider.go's stampSteps performs.
func conformanceSweepTextStreamAnchor(tb testing.TB) []ai.Event {
	tb.Helper()

	responseStart, err := ai.NewResponseStart("chatcmpl-or-text", "openai/gpt-4o")
	conformanceSweepRequireConstructed(tb, err, "ai.NewResponseStart")
	textStart, err := ai.NewTextBlockStart(1)
	conformanceSweepRequireConstructed(tb, err, "ai.NewTextBlockStart")
	alpha, err := ai.NewTextDelta(1, "alpha")
	conformanceSweepRequireConstructed(tb, err, "ai.NewTextDelta")
	beta, err := ai.NewTextDelta(1, " beta")
	conformanceSweepRequireConstructed(tb, err, "ai.NewTextDelta")
	gamma, err := ai.NewTextDelta(1, " gamma")
	conformanceSweepRequireConstructed(tb, err, "ai.NewTextDelta")
	textEnd, err := ai.NewTextBlockEnd(1)
	conformanceSweepRequireConstructed(tb, err, "ai.NewTextBlockEnd")
	completion, err := ai.NewCompletion(ai.FinishReasonStop, ai.Usage{})
	conformanceSweepRequireConstructed(tb, err, "ai.NewCompletion")

	var stamper ai.Stamper
	return []ai.Event{
		stamper.Stamp(responseStart),
		stamper.Stamp(textStart),
		stamper.Stamp(alpha),
		stamper.Stamp(beta),
		stamper.Stamp(gamma),
		stamper.Stamp(textEnd),
		stamper.Stamp(completion),
	}
}

// conformanceSweepServeWhole is a minimal single-shot handler serving
// transcript in one Write call — the canonical (unsplit) reference every
// split replay in this file is compared against.
func conformanceSweepServeWhole(transcript []byte) http.HandlerFunc {
	return func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(transcript)
	}
}

// conformanceSweepServeSplitInOrder returns a handler serving the
// (n+1)th inbound request's response split into two writes —
// transcript[:offsets[n]] then transcript[offsets[n]:], with a real
// Flush between them so the client's TCP read genuinely observes two
// separate reads at the split boundary — in arrival order, one entry of
// offsets consumed per request.
func conformanceSweepServeSplitInOrder(transcript []byte, offsets []int) http.HandlerFunc {
	var mu sync.Mutex
	next := 0
	return func(w http.ResponseWriter, _ *http.Request) {
		mu.Lock()
		idx := next
		next++
		mu.Unlock()

		if idx >= len(offsets) {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		offset := offsets[idx]
		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(transcript[:offset])
		if flusher, ok := w.(http.Flusher); ok {
			flusher.Flush()
		}
		_, _ = w.Write(transcript[offset:])
	}
}

// conformanceSweepClient builds a real *openaicompat.Client pointed at
// server, credential and header shape irrelevant to this sweep (no
// attribution round-tripper needed: this file proves offset robustness,
// not header injection, which bridge_test.go's own factory already
// covers).
func conformanceSweepClient(tb testing.TB, server *httptest.Server) *openaicompat.Client {
	tb.Helper()
	client, err := openaicompat.New(openaicompat.Config{
		Endpoint:   server.URL,
		Credential: openaicompat.NewCredential("boundary-sweep-token"),
		HTTPClient: server.Client(),
	})
	if err != nil {
		tb.Fatalf("openaicompat.New(Config) error = %v, want nil", err)
	}
	return client
}

// conformanceSweepDrain streams one request against client and drains it
// bounded, returning the events observed.
func conformanceSweepDrain(tb testing.TB, client *openaicompat.Client) []ai.Event {
	tb.Helper()
	ch, err := client.Stream(context.Background(), conformanceSweepRequest(tb))
	if err != nil {
		tb.Fatalf("Stream error = %v, want nil", err)
	}
	rec := agenttest.DrainAndRecord(tb, ch, agenttest.DefaultDrainTimeout)
	return rec.Events()
}

// conformanceSweepCanonicalReplay serves transcript whole, from a fresh
// single-shot server, and returns the drained events — the reference
// every split replay is compared against.
func conformanceSweepCanonicalReplay(tb testing.TB, transcript []byte) []ai.Event {
	tb.Helper()
	server := httptest.NewServer(conformanceSweepServeWhole(transcript))
	defer server.Close()
	return conformanceSweepDrain(tb, conformanceSweepClient(tb, server))
}

// conformanceSweepRequest builds the smallest valid ai.Request this
// sweep's real HTTP calls need.
func conformanceSweepRequest(tb testing.TB) ai.Request {
	tb.Helper()
	part, err := ai.NewText("boundary sweep probe")
	conformanceSweepRequireConstructed(tb, err, "ai.NewText")
	message, err := ai.NewMessage(ai.RoleUser, part)
	conformanceSweepRequireConstructed(tb, err, "ai.NewMessage")
	request, err := ai.NewRequest("cachicamas-boundary-sweep-model", []ai.Message{message})
	conformanceSweepRequireConstructed(tb, err, "ai.NewRequest")
	return request
}

// TestBoundarySweep_FixturesWithinSweepBound is S-ACR-016's positive
// clause: every candidate transcript this sweep actually replays is
// within conformanceSweepMaxBytes.
func TestBoundarySweep_FixturesWithinSweepBound(t *testing.T) {
	for _, v := range checkConformanceSweepBound(conformanceSweepTranscripts(t)) {
		t.Error(v)
	}
}

// TestBoundarySweep_SyntheticOverweightFixture_Fails is S-ACR-016's
// negative clause, proven permanently against a synthetic candidate —
// mirrors openaicompat_test's TestOffsetSweep_SyntheticOverweightFixture_Fails
// exactly: the real registry is never touched.
func TestBoundarySweep_SyntheticOverweightFixture_Fails(t *testing.T) {
	const oversizedName = "SYNTHETIC-OVERWEIGHT-FOR-GUARD-PROOF"
	oversized := conformanceSweepTranscript{
		name:       oversizedName,
		transcript: make([]byte, conformanceSweepMaxBytes+1),
	}

	violations := checkConformanceSweepBound([]conformanceSweepTranscript{oversized})
	if len(violations) != 1 {
		t.Fatalf("checkConformanceSweepBound(1025-byte synthetic fixture) = %d violations, want exactly 1", len(violations))
	}
	if violations[0].name != oversizedName {
		t.Errorf("violation names transcript %q, want %q", violations[0].name, oversizedName)
	}
	if violations[0].size != conformanceSweepMaxBytes+1 {
		t.Errorf("violation reports size %d, want %d", violations[0].size, conformanceSweepMaxBytes+1)
	}

	atBound := conformanceSweepTranscript{name: "AT-BOUND", transcript: make([]byte, conformanceSweepMaxBytes)}
	if got := checkConformanceSweepBound([]conformanceSweepTranscript{atBound}); len(got) != 0 {
		t.Errorf("checkConformanceSweepBound(exactly-at-bound fixture) = %v, want no violations — the bound is inclusive", got)
	}
}

// checkConformanceSweepBound is S-ACR-016's structural size guard, a pure
// function over any candidate transcript set — mirrors
// openaicompat_test's unexported checkSweepFixtureBound exactly (same
// shape, same inclusive bound reasoning).
func checkConformanceSweepBound(candidates []conformanceSweepTranscript) []conformanceSweepBoundViolation {
	var violations []conformanceSweepBoundViolation
	for _, tr := range candidates {
		if len(tr.transcript) > conformanceSweepMaxBytes {
			violations = append(violations, conformanceSweepBoundViolation{name: tr.name, size: len(tr.transcript)})
		}
	}
	return violations
}

// conformanceSweepBoundViolation names one candidate transcript whose
// size exceeds conformanceSweepMaxBytes.
type conformanceSweepBoundViolation struct {
	name string
	size int
}

func (v conformanceSweepBoundViolation) String() string {
	return fmt.Sprintf("transcript %q is %d bytes, want <= %d (S-ACR-016)", v.name, v.size, conformanceSweepMaxBytes)
}

// TestBoundarySweep_CanonicalDecodeMatchesHardCodedAnchor is S-ACR-015's
// vacuity guard: at least one transcript's canonical replay is anchored
// against a hard-coded expected event list, not only against itself —
// otherwise every offset in the exhaustive sweep below could pass while
// the canonical decode itself were uniformly wrong.
func TestBoundarySweep_CanonicalDecodeMatchesHardCodedAnchor(t *testing.T) {
	anchored := 0
	for _, tr := range conformanceSweepTranscripts(t) {
		if tr.want == nil {
			continue
		}
		anchored++
		got := conformanceSweepCanonicalReplay(t, tr.transcript)
		agenttest.RequireSameEvents(t, got, tr.want)
	}
	if anchored == 0 {
		t.Fatal("no sweep transcript sets want — the sweep would compare every split replay only against a same-code canonical replay, which cannot detect a uniformly wrong decode (S-ACR-015)")
	}
}

// TestBoundarySweep_EventLevel_EveryOffsetMatchesCanonical is R-ACR-005 /
// S-ACR-014's exhaustive tier: every admitted byte offset (0…len,
// inclusive) of every eligible transcript, replayed split through a real
// httptest.Server + real *openaicompat.Client, drains to the same event
// sequence as the canonical unsplit replay.
//
// # Runtime measurement (T5, tasks.md 7.2)
//
// Measured: `go test -run TestBoundarySweep_EventLevel_EveryOffsetMatchesCanonical
// -race -v` — the full offset cross-product over the three eligible
// transcripts (922+911+681 = 2514 real HTTP round trips) reports
// --- PASS: ... (1.03s) as the test's own measured runtime (2.366s total
// go test wall time including package build/setup) — well under the 30s
// budget, so this tier stays exhaustive, no sampling applied (tasks.md
// 7.2's own instruction: measure first, sample only if the budget is
// exceeded). See boundary_sweep_test.go's own header comment for why
// tier 2 (record-level) is sampled unconditionally instead — design D8's
// own literal choice, not a runtime-contingent fallback of this tier.
func TestBoundarySweep_EventLevel_EveryOffsetMatchesCanonical(t *testing.T) {
	for _, tr := range conformanceSweepTranscripts(t) {
		t.Run(tr.name, func(t *testing.T) {
			canonical := conformanceSweepCanonicalReplay(t, tr.transcript)

			offsets := make([]int, len(tr.transcript)+1)
			for i := range offsets {
				offsets[i] = i
			}
			server := httptest.NewServer(conformanceSweepServeSplitInOrder(tr.transcript, offsets))
			defer server.Close()
			client := conformanceSweepClient(t, server)

			for _, offset := range offsets {
				got := conformanceSweepDrain(t, client)
				agenttest.RequireSameEvents(t, got, canonical)
				if t.Failed() {
					t.Fatalf("transcript %q: mismatch first observed at split offset %d", tr.name, offset)
				}
			}
		})
	}
}

// conformanceSweepOffsetStrategy names one of design D8's three sampled
// relative offsets and how to compute it against any transcript length
// (T5): "1" (the first byte only in the initial write), "len/2" (an even
// split) and "len-1" (everything but the last byte in the initial
// write). Each is a distinct byte count for every transcript any case in
// the unscoped run constructs, not a single global byte offset — a fixed
// byte count would be meaningless across transcripts of different
// lengths.
type conformanceSweepOffsetStrategy struct {
	name string
	fn   func(transcriptLen int) int
}

// conformanceSweepOffsetStrategies is design D8's own literal choice for
// tier 2 (record-level): {1, len/2, len-1}, clamped so every strategy
// stays within [0, n] for any transcript length, including the
// pathological n=0 case (an empty rendered transcript, which no real
// case produces, but the clamp keeps the strategy total rather than
// slicing out of range).
func conformanceSweepOffsetStrategies() []conformanceSweepOffsetStrategy {
	return []conformanceSweepOffsetStrategy{
		{name: "offset=1", fn: func(n int) int {
			if n < 1 {
				return 0
			}
			return 1
		}},
		{name: "offset=len/2", fn: func(n int) int { return n / 2 }},
		{name: "offset=len-1", fn: func(n int) int {
			if n < 1 {
				return 0
			}
			return n - 1
		}},
	}
}

// TestBoundarySweep_RecordLevel_SampledOffsetsMatchExpectation is R-ACR-005
// / S-ACR-014's second tier (design D8, tasks.md Phase 7 WU8 task 7.2):
// for each of the three sampled relative offset strategies, one full
// unscoped agenttest.RunConformance run — via conformanceBridgeFactorySplitAt,
// which splits EVERY transcript any case constructs at that strategy's
// relative offset — must still emit a CapabilityRecord equal to
// expectedOpenRouterRecord() (the same expectation
// TestOpenRouterAdapter_FullConformance asserts unsplit). This is the
// record-level claim T5 demands: not merely identical decoded frames
// (tier 1's claim) but an identical suite VERDICT under adversarial
// fragmentation.
//
// No t.Parallel(): each run drives the cancellation cases' own
// RequireNoGoroutineLeak, which is serial-only (R-STK-008) — identical
// constraint to TestOpenRouterAdapter_FullConformance.
func TestBoundarySweep_RecordLevel_SampledOffsetsMatchExpectation(t *testing.T) {
	for _, strategy := range conformanceSweepOffsetStrategies() {
		t.Run(strategy.name, func(t *testing.T) {
			record := agenttest.RunConformance(t, conformanceBridgeFactorySplitAt(strategy.fn))
			requireOpenRouterCapabilityRecord(t, record)
		})
	}
}
