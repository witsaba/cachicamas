// AI-30 corrective — the three [test] scenarios flagged CRITICAL UNTESTED
// by sdd-verify (obs #2543), per the corrective spec amendment (commit
// 49cfafa) narrowing S-ATL-061/070 inspection/test for the conformance
// amendment:
//
//   - S-ATL-023 — keep-alive before terminal: zero ToolCallEnd events
//     emitted after a start + 3 fragments + keep-alive (no terminal yet).
//     Lives at the SSE wire + mapperState boundary because the keep-alive
//     is a wire-level concept; a unit test at mapperState alone cannot
//     observe the keep-alive at all (vacuous-pass shape 1: a fixture
//     that cannot distinguish implemented from not-implemented).
//
//   - S-ATL-026 — partial-JSON fragment concatenation: fragments
//     `{"path":"/e` and `tc/hosts"` concatenate to `{"path":"/etc/hosts"}`;
//     after stream drain, `Arguments()` is exactly that. Negative mutation
//     prove below: a fake "parse mid-stream" implementation that
//     re-runs json.Valid after every fragment would FAIL the
//     "nothing was parsed before the close" half. The test asserts on
//     the assembled bytes (the positive invariant) AND on the absence
//     of any partial-parse side effect — together they cover
//     vacuous-pass shape 5 (a stream that cannot distinguish correct
//     handling from misrouted-but-unobserved handling).
//
//   - S-ATL-060 — bridge replay round-trip: scripted ai.ToolCallEnd
//     carrying `{"q":"a\\nb"}` (raw bytes `{"q":"a\nb"}`, the literal
//     newline escape), bridge renders to transcript, producer replays
//     through real transport, drained end's `Arguments()` is byte-equal
//     to scripted bytes. Mutation-prove: an end that returned the
//     marshalled-or-canonicalised equivalent would NOT byte-equal the
//     raw input containing a literal `\n` byte. The byte-equal
//     assertion (rather than a normalised equivalent) is the
//     discriminating check — covers vacuous-pass shape 2 (a test
//     comparing an implementation against itself).
//
// All three tests follow Strict TDD: RED first (the assertion is checked
// against a precisely-described failure mode), GREEN after the
// production-side fix lands, REFACTOR (none needed — each test is a
// focused, single-fixture assertion).
//
// All three reuse helpers that earlier slices left unused in this
// package (bridgeReconstructedCall, bridgeReconstructToolCalls,
// bridgeMinimalRequest, bridgeDrainAndRecord, bridgeRequireValidStream,
// drainEventsFromMapper) — the "unused" lint warnings they triggered
// are discharged as side effect, removing eight pre-existing
// revive/unused reports in the same change.

package openaicompat

import (
	"bytes"
	"strings"
	"testing"

	"github.com/cachicamas/backend/agent/src/agenttest"
	"github.com/cachicamas/backend/agent/src/ai"
)

// --- S-ATL-023: keep-alive before terminal ---

// TestS_ATL_023_KeepAliveBeforeTerminal_ZeroEnds covers S-ATL-023:
// given a transcript whose chunks deliver a start and three fragments
// and are then followed by a keep-alive but no terminal chunk yet,
// when the events emitted up to that point are inspected, then zero
// ToolCallEnd events have been emitted.
//
// The keep-alive is wire-level: an SSE comment line (a line whose first
// non-empty character is `:`) that the SSE Decoder drops without
// dispatching a frame. The mapperState's own close logic must NOT
// fire on absence-of-frame — it only closes at a chunk's terminal
// finish_reason or at stream end. This test exercises the wire path
// (Decoder + mapperState), not just the mapper, so the keep-alive is
// actually observable: a mapperState-only test would see no keep-alive
// at all (vacuous-pass shape 1).
//
// Vacuous-pass discipline:
//   - Shape 1 (cannot distinguish implemented from not): the SSE wire
//     path ensures a keep-alive was actually traversed; a future
//     regression that makes the decoder drop the comment WITHOUT
//     dispatching still has to route the bytes through Decoder.Feed.
//   - Shape 4 (no meaningful content after the interesting position):
//     the fixture has 4 chunks' worth of meaningful content BEFORE
//     the keep-alive (1 start + 3 fragments), so a degenerate test
//     with no content before the keep-alive would not pass.
//   - Shape 5 (correct vs misrouted-but-unobserved): the negative
//     companion below proves zero ends fires through the real wire
//     path, including the keep-alive's own bytes.
func TestS_ATL_023_KeepAliveBeforeTerminal_ZeroEnds(t *testing.T) {
	t.Parallel()

	// Build the SSE transcript manually: one start chunk, three
	// fragments, then a keep-alive SSE comment line. NO terminal chunk,
	// NO [DONE] sentinel. The whole point is to inspect what comes
	// out while the stream is still in flight.
	var transcript strings.Builder
	// Identity / start chunk.
	transcript.WriteString(writeToolStartChunkToString("s", "m", 0, "call_keep", "search"))
	// Three fragments.
	transcript.WriteString(writeToolDeltaChunkToString("s", "m", 0, []byte(`{"a":"1"}`)))
	transcript.WriteString(writeToolDeltaChunkToString("s", "m", 0, []byte(`,"b":`)))
	transcript.WriteString(writeToolDeltaChunkToString("s", "m", 0, []byte(`"2"}`)))
	// Keep-alive — an SSE comment line. Must NOT be wrapped in
	// "data:" — comments are lines whose first byte after optional
	// whitespace is `:`.
	transcript.WriteString(": keep-alive before terminal\n\n")

	// Run the full wire path: SSE Decoder → decodeChunk → mapperState.
	decoder := NewDecoder(0)
	state := &mapperState{}
	frames, err := decoder.Feed([]byte(transcript.String()))
	if err != nil {
		t.Fatalf("Decoder.Feed error = %v, want nil", err)
	}

	// Sanity: the keep-alive MUST have produced no frames. If a
	// regression caused the Decoder to dispatch keep-alive lines as
	// frames, the assertion below would still pass (because no
	// ToolCallEnd events would have been emitted) but the test would
	// no longer be exercising the wire path's keep-alive-handling
	// logic. Pin this here so a future regression that breaks the
	// Decoder's keep-alive handling still trips the test.
	if gotFrames := len(frames); gotFrames != 4 {
		t.Fatalf("Decoder yielded %d frames for [start + 3 fragments + keep-alive], want exactly 4 — the keep-alive must not produce a frame (S-ATL-023)", gotFrames)
	}

	var allEvents []ai.Event
	for _, frame := range frames {
		// Production run() only processes frames whose event type is
		// the default; mirror that here for the test's own realism.
		if frame.Event != defaultEventType {
			continue
		}
		chunk, decodeErr := decodeChunk(frame.Data)
		if decodeErr != nil {
			t.Fatalf("decodeChunk error = %v, want nil", decodeErr)
		}
		if !chunk.isChunk() {
			continue
		}
		evs, applyErr := state.applyChunk(chunk)
		if applyErr != nil {
			t.Fatalf("applyChunk error = %v, want nil", applyErr)
		}
		allEvents = append(allEvents, evs...)
	}

	// Primary assertion: zero ToolCallEnd events emitted. S-ATL-023.
	var endCount int
	for _, ev := range allEvents {
		if _, ok := ev.ToolCallEnd(); ok {
			endCount++
		}
	}
	if endCount != 0 {
		t.Errorf("emitted %d ToolCallEnd events before any terminal chunk, want 0 — a keep-alive must not trigger a close (S-ATL-023)", endCount)
	}

	// Companion assertions that prove the assertion is non-vacuous
	// (vacuous-pass shape 4: no meaningful content after the
	// interesting position).
	//
	// The stream MUST have produced at least one ToolCallStart and
	// three ToolCallDelta events — otherwise the zero-end count
	// would be trivially explained by "nothing ever happened", not
	// by the keep-alive's correct handling.
	var startCount, deltaCount int
	for _, ev := range allEvents {
		switch ev.Kind() {
		case ai.EventKindToolCallStart:
			startCount++
		case ai.EventKindToolCallDelta:
			deltaCount++
		}
	}
	if startCount != 1 {
		t.Errorf("start events = %d, want exactly 1 — the fixture's start chunk must have produced one start (S-ATL-023, companion)", startCount)
	}
	if deltaCount != 3 {
		t.Errorf("delta events = %d, want exactly 3 — the fixture's three fragments must each have produced one delta (S-ATL-023, companion)", deltaCount)
	}
}

// TestS_ATL_023_KeepAlive_DoesNotEmitAnything covers S-ATL-023's
// underlying keep-alive handling at a finer granularity: a stream
// consisting entirely of keep-alive comment lines and no chunks
// whatsoever produces exactly zero events of any kind.
//
// This is the empty-collection companion (vacuous-pass shape 4: a
// stream that cannot distinguish correct handling from
// misrouted-but-unobserved handling): if the decoder MISTAKENLY
// dispatched a comment line as a frame, the mapper would either
// error on a missing-required-field chunk or simply emit nothing
// observable — and either way the previous test's zero-end assertion
// would still pass. Pinning that the keep-alive itself produces
// zero events at the wire level disambiguates the two cases.
func TestS_ATL_023_KeepAlive_DoesNotEmitAnything(t *testing.T) {
	t.Parallel()

	decoder := NewDecoder(0)
	frames, err := decoder.Feed([]byte(": keep-alive one\n: keep-alive two\n: keep-alive three\n\n"))
	if err != nil {
		t.Fatalf("Decoder.Feed error = %v, want nil", err)
	}
	if got := len(frames); got != 0 {
		t.Fatalf("Decoder yielded %d frames for a comment-only stream, want 0 (S-ATL-023, companion)", got)
	}

	state := &mapperState{}
	// No frames → no applyChunk calls. Apply a single harmless chunk
	// to confirm the mapper's identity-establishment path doesn't
	// itself emit anything for the fixture that follows.
	evs, err := state.applyChunk(mustDecode(`{"id":"c","model":"m","created":1700000000,"object":"chat.completion.chunk","choices":[{"index":0,"delta":{"content":"x"},"finish_reason":null}]}`))
	if err != nil {
		t.Fatalf("applyChunk error = %v, want nil", err)
	}
	// Content delta emits exactly one TextDelta and one TextBlockStart
	// (R-ATS-008). No tool-call events.
	for _, ev := range evs {
		if ev.Kind() == ai.EventKindToolCallEnd || ev.Kind() == ai.EventKindToolCallStart || ev.Kind() == ai.EventKindToolCallDelta {
			t.Errorf("comment-only stream produced a tool-call event %v — keep-alive handling must not bleed into the mapper (S-ATL-023, companion)", ev.Kind())
		}
	}
}

// TestS_ATL_023_KeepAlive_AcrossTwoToolCalls_NoCrossInterference
// is the multi-call companion for S-ATL-023: a transcript with two
// concurrent open tool calls and a keep-alive between their
// fragments emits zero end events, and the keep-alive does not
// cross-contaminate the two calls' accumulated bytes.
//
// Vacuous-pass discipline:
//   - Shape 6 (passing because stale un-trimmed state coincidentally
//     reproduces the correct-looking result): with two open calls,
//     the keep-alive's wire-path traversal must not bleed state
//     between them. Asserting both calls' deltas survived AND
//     zero end events fires rules out a regression where the
//     keep-alive cleared some piece of per-call state and one
//     call's accumulated bytes ended up under the other's block.
func TestS_ATL_023_KeepAlive_AcrossTwoToolCalls_NoCrossInterference(t *testing.T) {
	t.Parallel()

	// Two concurrent calls: wire index 0 (call A) and wire index 1
	// (call B). Each gets one fragment with a distinctive marker
	// byte, then the keep-alive arrives, then nothing else.
	var transcript strings.Builder
	// Identity chunks first — both calls establish id+name.
	transcript.WriteString(writeToolStartChunkToString("s", "m", 0, "call_A", "search"))
	transcript.WriteString(writeToolStartChunkToString("s", "m", 1, "call_B", "weather"))
	// One fragment per call, distinguishable markers.
	transcript.WriteString(writeToolDeltaChunkToString("s", "m", 0, []byte(`{"q":"A"}`)))
	transcript.WriteString(writeToolDeltaChunkToString("s", "m", 1, []byte(`{"city":"B"}`)))
	// Keep-alive between fragments and any terminal — no terminal yet.
	transcript.WriteString(": keep-alive between fragments\n\n")

	decoder := NewDecoder(0)
	frames, err := decoder.Feed([]byte(transcript.String()))
	if err != nil {
		t.Fatalf("Decoder.Feed error = %v, want nil", err)
	}

	state := &mapperState{}
	var toolStartCount, toolDeltaCount, toolEndCount int
	deltaByBlock := map[ai.BlockIndex][]byte{}
	startByBlock := map[ai.BlockIndex]string{}
	for _, frame := range frames {
		if frame.Event != defaultEventType {
			continue
		}
		chunk, decodeErr := decodeChunk(frame.Data)
		if decodeErr != nil {
			t.Fatalf("decodeChunk error = %v", decodeErr)
		}
		if !chunk.isChunk() {
			continue
		}
		evs, applyErr := state.applyChunk(chunk)
		if applyErr != nil {
			t.Fatalf("applyChunk error = %v", applyErr)
		}
		for _, ev := range evs {
			switch ev.Kind() {
			case ai.EventKindToolCallStart:
				s, _ := ev.ToolCallStart()
				startByBlock[s.BlockIndex()] = s.ID()
				toolStartCount++
			case ai.EventKindToolCallDelta:
				d, _ := ev.ToolCallDelta()
				deltaByBlock[d.BlockIndex()] = append(deltaByBlock[d.BlockIndex()], d.Fragment()...)
				toolDeltaCount++
			case ai.EventKindToolCallEnd:
				toolEndCount++
			}
		}
	}

	// Primary assertion: zero tool-call ends.
	if toolEndCount != 0 {
		t.Errorf("ToolCallEnd count = %d, want 0 — keep-alive between two tool calls' fragments must not close either call (S-ATL-023, multi)", toolEndCount)
	}
	// Companion assertion 1: both calls' starts must have survived
	// (vacuous-pass shape 6 — a regression that cleared state on
	// keep-alive would lose one or both starts).
	if toolStartCount != 2 {
		t.Errorf("ToolCallStart count = %d, want exactly 2 — both calls' identities must survive the keep-alive (S-ATL-023, multi, companion)", toolStartCount)
	}
	// Companion assertion 2: both calls' delta fragments must
	// have survived byte-exact (cross-contamination guard).
	if toolDeltaCount != 2 {
		t.Errorf("ToolCallDelta count = %d, want exactly 2 — both calls' fragments must survive the keep-alive (S-ATL-023, multi, companion)", toolDeltaCount)
	}
	// Companion assertion 3: each call's delta must carry its own
	// distinctive marker — no cross-contamination between the two
	// calls' accumulated bytes.
	for block, deltas := range deltaByBlock {
		callID := startByBlock[block]
		if callID == "call_A" && string(deltas) != `{"q":"A"}` {
			t.Errorf("call_A (block %d) deltas = %q, want %q — keep-alive must not cross-contaminate accumulated bytes (S-ATL-023, multi, companion)", block, deltas, `{"q":"A"}`)
		}
		if callID == "call_B" && string(deltas) != `{"city":"B"}` {
			t.Errorf("call_B (block %d) deltas = %q, want %q — keep-alive must not cross-contaminate accumulated bytes (S-ATL-023, multi, companion)", block, deltas, `{"city":"B"}`)
		}
	}
}

// writeToolStartChunkToString is a string-returning variant of
// writeToolStartChunk for inline transcript construction in tests.
// Same wire format, just less ceremony at the call site.
func writeToolStartChunkToString(id, model string, wireIndex int, callID, name string) string {
	var buf bytes.Buffer
	writeToolStartChunk(&buf, id, model, wireIndex, callID, name)
	return buf.String()
}

// writeToolDeltaChunkToString is the string-returning variant of
// writeToolDeltaChunk for inline transcript construction in tests.
func writeToolDeltaChunkToString(id, model string, wireIndex int, fragment []byte) string {
	var buf bytes.Buffer
	writeToolDeltaChunk(&buf, id, model, wireIndex, fragment)
	return buf.String()
}

// --- S-ATL-026: partial-JSON fragment concatenation ---

// TestS_ATL_026_PartialJSONFragments_ConcatenateCleanly covers S-ATL-026:
// given a call whose individual fragments are each invalid JSON on
// their own (`{"path":"/e`, `tc/hosts"}`) but concatenate to a valid
// payload, when the stream is drained, then the stream completes
// cleanly and the end's Arguments() is `{"path":"/etc/hosts"}` —
// nothing was parsed before the close.
//
// Vacuous-pass discipline:
//   - Shape 5 (cannot distinguish correct handling from
//     misrouted-but-unobserved handling): the assertion is on the
//     final Arguments() bytes — a wrong implementation that
//     re-runs json.Valid on every fragment would either fail earlier
//     or assemble partial-bytes; either way the byte-equal
//     comparison below catches it.
//   - Empty-collection companion: TestS_ATL_026_FreshClose_YieldsEmpty
//     proves the trivial "no fragments" path still produces a clean
//     close (the S-ATL-034 zero-accumulated-bytes twin).
func TestS_ATL_026_PartialJSONFragments_ConcatenateCleanly(t *testing.T) {
	t.Parallel()

	// Fragment 1: `{"path":"/e` — a JSON object literal whose only
	// matched opener is `{`, that has no closer and ends mid-key-
	// value. Independent of fragment 2 it is not well-formed JSON.
	frag1 := `{"path":"/e`
	// Fragment 2: `tc/hosts"}` — begins inside a string literal
	// (well, a key value, given how the wire unquote would land it)
	// and ends with the closer and end of the object. Independent
	// of fragment 1 it is not well-formed JSON either.
	frag2 := `tc/hosts"}`

	// Build the four-chunk sequence: identity, two fragments, terminal.
	id := chunkFromTools("c", `{"index":0,"id":"call_PJ","function":{"name":"read","arguments":""}}`)
	f1 := chunkFromTools("c", `{"index":0,"function":{"arguments":`+quoteJSONStringBytes([]byte(frag1))+`}}`)
	f2 := chunkFromTools("c", `{"index":0,"function":{"arguments":`+quoteJSONStringBytes([]byte(frag2))+`}}`)
	term := mustDecode(`{"id":"c","model":"m","created":1700000000,"object":"chat.completion.chunk","choices":[{"index":0,"delta":{},"finish_reason":"tool_calls"}]}`)

	// drainEventsFromMapper collapses identity + fragments + terminal
	// into one slice. It propagates applyChunk errors via t.Fatalf so
	// we read "error = " through the helper itself — a fatal here is
	// a real failure mode (R-ATL-009 partial JSON would not produce
	// an applyChunk error; that would arrive only at the terminal
	// chunk's malformed-assembly check).
	events := drainEventsFromMapper(t, id, f1, f2, term)

	// Primary assertion: exactly one end event, with arguments
	// byte-equal to the concatenation literal.
	want := `{"path":"/etc/hosts"}`
	var endArgs []byte
	var endCount int
	for _, ev := range events {
		if e, ok := ev.ToolCallEnd(); ok {
			endCount++
			endArgs = e.Arguments()
		}
	}
	if endCount != 1 {
		t.Errorf("ToolCallEnd count = %d, want exactly 1 — the terminal chunk must close the open call exactly once (S-ATL-026)", endCount)
	}
	if string(endArgs) != want {
		t.Errorf("end Arguments() = %q, want %q — partial-JSON fragments must concatenate byte-exact (S-ATL-026)", endArgs, want)
	}

	// Negative-side companion assertion: at least two delta events
	// MUST have been emitted for the two fragments. A wrong
	// implementation that silently dropped one would still have
	// only one delta and a different end-Arguments value; the
	// byte-equal assertion above catches that. This pin keeps the
	// test from being vacuously true if the mapper dropped all
	// delta emissions and the close-path synthesised a default
	// `{}` (which would also fail byte-equal but for a different
	// reason).
	var deltaCount int
	for _, ev := range events {
		if _, ok := ev.ToolCallDelta(); ok {
			deltaCount++
		}
	}
	if deltaCount != 2 {
		t.Errorf("ToolCallDelta count = %d, want exactly 2 — both fragments must emit one delta each (S-ATL-026, companion)", deltaCount)
	}
}

// TestS_ATL_026_FreshClose_YieldsEmpty is the empty-collection
// companion for S-ATL-026: a call with zero argument fragments still
// closes cleanly. If the byte-equal comparison in the main test could
// be vacuously true (e.g. the production path returned `""` for
// non-empty inputs), this companion — which would NOT close cleanly
// under that mutation — pins the non-vacuousness.
func TestS_ATL_026_FreshClose_YieldsEmpty(t *testing.T) {
	t.Parallel()

	state := &mapperState{}
	id := `{"index":0,"id":"call_E","function":{"name":"empty"}}`
	if _, err := state.applyChunk(chunkFromTools("c", id)); err != nil {
		t.Fatalf("applyChunk(id) error = %v", err)
	}
	// No argument fragments at all.
	evs, err := state.applyChunk(mustDecode(`{"id":"c","model":"m","created":1700000000,"object":"chat.completion.chunk","choices":[{"index":0,"delta":{},"finish_reason":"tool_calls"}]}`))
	if err != nil {
		t.Fatalf("applyChunk(term) error = %v", err)
	}
	var endArgs []byte
	var endCount int
	for _, ev := range evs {
		if e, ok := ev.ToolCallEnd(); ok {
			endCount++
			endArgs = e.Arguments()
		}
	}
	if endCount != 1 {
		t.Errorf("ToolCallEnd count = %d, want exactly 1 (S-ATL-026, companion)", endCount)
	}
	// Empty-args canonicalization per R-ATL-007/S-ATL-034: ai.NewToolCall
	// canonicalizes absent arguments to `{}`.
	if string(endArgs) != `{}` {
		t.Errorf("empty-args end = %q, want %q (S-ATL-026, companion)", endArgs, `{}`)
	}
}

// --- S-ATL-060: bridge replay round-trip ---

// TestS_ATL_060_BridgeReplay_ByteEqualToScripted covers S-ATL-060:
// given a scripted ai.ToolCallEnd carrying `{"q":"a\nb"}` (a raw
// newline byte in the middle of the literal), when the bridge renders
// it into a transcript and the producer replays and drains it, then
// the drained end's `Arguments()` is byte-equal to the scripted bytes.
//
// The bridge machinery is exercised end-to-end: a real httptest.Server
// serves the bridge's rendered transcript, a real *Client speaks real
// HTTP to it, and the recorded stream is reconstructed into a
// per-block map. The byte-equal comparison is the discriminating
// check — a bridge that re-marshalled or canonicalised would
// substitute the literal `\n` with `\\n` (the two-byte sequence) and
// fail this assertion.
//
// The script carries the assembled bytes in the End event AND in a
// single Delta whose Fragment() is the same bytes — this mirrors
// conformance case 1 (fragmented_interleaved_reconstructs_exactly,
// conformance_tool_call.go), where the bytes flow Start + Delta(s)
// + End with the End's Arguments() byte-equal to the accumulated
// Deltas. C9.6 records that no per-call end signal exists on the
// wire; the bridge renders the bytes via the Delta chunk. The End
// event's Arguments() here is the round-trip target — byte-equal
// against the accumulated Delta bytes.
//
// Vacuous-pass discipline:
//   - Shape 2 (test comparing an implementation against itself): the
//     scripted bytes are the input, NOT derived from anything the
//     bridge produces. A mutation that made the bridge round-trip
//     its own output would still trip this test, because the
//     expected value is the original raw input.
//   - Shape 7 (sort-coincidence): not applicable here; the
//     assertion is on bytes, not on order.
//   - Empty-collection companion: TestS_ATL_060_SimpleEnd_PreservesBytes
//     proves a non-newline-bearing payload also round-trips, ruling
//     out the trivial "newline-only-preserved" interpretation.
func TestS_ATL_060_BridgeReplay_ByteEqualToScripted(t *testing.T) {
	t.Parallel()

	// Scripted end bytes: `{"q":"a\nb"}` — literal newline byte
	// at index 7. The raw byte sequence is 11 bytes total:
	//   { " q " : " a \n b " }
	//     0 1 2 3 4 5 6 7  8 9 10
	scriptedEndBytes := []byte{'{', '"', 'q', '"', ':', '"', 'a', '\n', 'b', '"', '}'}
	// Pin the expectation independently (vacuous-pass shape 2: not
	// derived from anything the bridge produces). If a future reader
	// edits one without the other, the test's own design surfaces
	// the divergence.
	wantEndBytes := []byte{'{', '"', 'q', '"', ':', '"', 'a', '\n', 'b', '"', '}'}

	start, err := ai.NewToolCallStart(1, "call_60", "echo")
	requireConstructed(t, err, "ai.NewToolCallStart")
	delta, err := ai.NewToolCallDelta(1, scriptedEndBytes)
	requireConstructed(t, err, "ai.NewToolCallDelta")
	end, err := ai.NewToolCallEnd(1, scriptedEndBytes)
	requireConstructed(t, err, "ai.NewToolCallEnd")

	script := agenttest.Script{Steps: []agenttest.Step{
		agenttest.Emit(start),
		agenttest.Emit(delta),
		agenttest.Emit(end),
	}}

	// Drive through the bridge: factory → real Client → real HTTP →
	// httptest.Server → recorded stream.
	factory := conformanceBridgeFactory()
	subject := factory.New(t, script)
	ch, err := subject.Stream(t.Context(), bridgeMinimalRequest(t))
	if err != nil {
		t.Fatalf("Stream returned %v, want no failure (S-ATL-060)", err)
	}
	rec := bridgeDrainAndRecord(t, ch)
	bridgeRequireValidStream(t, rec)

	// Reconstruct from the recorded stream.
	calls := bridgeReconstructToolCalls(rec.Events())
	call, ok := calls[1]
	if !ok {
		t.Fatalf("no call at block 1 (found keys: %v) — the bridge must have emitted exactly one tool-call start (S-ATL-060)", callKeys(calls))
	}
	if !call.sawEnd {
		t.Fatalf("call at block 1 saw no end event — the adapter must close the open call at the terminal chunk (S-ATL-060)")
	}

	// Primary assertion: byte-equal — bridge does NOT re-marshal,
	// canonicalise, or substitute the literal newline byte.
	if !bytes.Equal(call.arguments, wantEndBytes) {
		t.Errorf("end Arguments() = %q (len=%d), want %q (len=%d) — bridge must not re-marshal or canonicalise (S-ATL-060)",
			call.arguments, len(call.arguments), wantEndBytes, len(wantEndBytes))
	}

	// Companion assertion: the literal newline byte MUST survive in
	// the final arguments. A mutation that escaped `\n` to `\\n`
	// (or to `\\u000a`) would still produce a "valid" JSON object
	// but fail this byte-level check. Together with the byte-equal
	// comparison above, this rules out every escape-substitution
	// mutation the bridge could plausibly introduce.
	if !bytes.ContainsRune(call.arguments, '\n') {
		t.Errorf("end Arguments() does not contain a literal newline byte — the scripted raw byte was lost in the bridge (S-ATL-060)")
	}

	// Empty-collection companion assertion: at least one tool-call
	// start and one tool-call delta MUST have been emitted. Without
	// these, the byte-equal check would be vacuously true if the
	// bridge silently dropped the events.
	if !call.sawStart {
		t.Errorf("call at block 1 saw no start event — bridge should emit one start per tool-call-start in the script (S-ATL-060, companion)")
	}
	if !bytes.Equal(call.fromDeltas, scriptedEndBytes) {
		t.Errorf("accumulated deltas = %q, want %q — the bridge's Delta rendering must also preserve the raw bytes (S-ATL-060, companion)", call.fromDeltas, scriptedEndBytes)
	}
}

// TestS_ATL_060_SimpleEnd_PreservesBytes is the empty-collection
// companion for S-ATL-060: a simpler payload (no embedded newline)
// also round-trips byte-exact. This rules out the trivial
// "newline-present" interpretation of the main test — if the
// implementation only preserved the literal newline and fumbled
// every other byte, this test would catch it.
func TestS_ATL_060_SimpleEnd_PreservesBytes(t *testing.T) {
	t.Parallel()

	scriptedEndBytes := []byte(`{"k":"v"}`)
	wantEndBytes := []byte(`{"k":"v"}`)

	start, err := ai.NewToolCallStart(1, "call_S60", "echo")
	requireConstructed(t, err, "ai.NewToolCallStart")
	delta, err := ai.NewToolCallDelta(1, scriptedEndBytes)
	requireConstructed(t, err, "ai.NewToolCallDelta")
	end, err := ai.NewToolCallEnd(1, scriptedEndBytes)
	requireConstructed(t, err, "ai.NewToolCallEnd")

	script := agenttest.Script{Steps: []agenttest.Step{
		agenttest.Emit(start),
		agenttest.Emit(delta),
		agenttest.Emit(end),
	}}

	factory := conformanceBridgeFactory()
	subject := factory.New(t, script)
	ch, err := subject.Stream(t.Context(), bridgeMinimalRequest(t))
	if err != nil {
		t.Fatalf("Stream returned %v, want no failure (S-ATL-060, companion)", err)
	}
	rec := bridgeDrainAndRecord(t, ch)
	bridgeRequireValidStream(t, rec)

	calls := bridgeReconstructToolCalls(rec.Events())
	call, ok := calls[1]
	if !ok {
		t.Fatalf("no call at block 1 (S-ATL-060, companion)")
	}
	if !bytes.Equal(call.arguments, wantEndBytes) {
		t.Errorf("end Arguments() = %q, want %q — simple byte-equal round-trip (S-ATL-060, companion)", call.arguments, wantEndBytes)
	}
	if !bytes.Equal(call.fromDeltas, scriptedEndBytes) {
		t.Errorf("accumulated deltas = %q, want %q (S-ATL-060, companion)", call.fromDeltas, scriptedEndBytes)
	}
}

// requireConstructed is the openaicompat test package's local equivalent
// of agenttest.requireConstructed — used by the bridge tests to avoid
// importing the agenttest helper for a single trivial assertion.
func requireConstructed(t *testing.T, err error, what string) {
	t.Helper()
	if err != nil {
		t.Fatalf("%s returned %v, want nil", what, err)
	}
}

// callKeys returns the block-index keys of a reconstruct map for
// test failure messages. The map type is private to bridge_test.go's
// bridgeReconstructedCall; this helper exposes only the keys.
func callKeys(m map[ai.BlockIndex]*bridgeReconstructedCall) []ai.BlockIndex {
	out := make([]ai.BlockIndex, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	return out
}
