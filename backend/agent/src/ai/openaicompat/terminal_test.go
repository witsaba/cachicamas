// AI-28.2 — terminal discipline: R-ATS-012 (the terminal sentinel is clean
// termination, never payload, never truncation, S-ATS-044…047) and
// R-ATS-014 (frames arriving after the sentinel and before EOF are ignored,
// S-ATS-052…054). Same httptest.Server-driven seam as stream_text_test.go —
// every scenario here needs the whole Client.Stream → real HTTP → real
// Decoder pipeline, because it exercises the read LOOP's own sentinel
// recognition and post-sentinel behavior (stream.go's run), not the
// mapper's per-chunk mapping in isolation.
//
// truncation_test.go (R-ATS-013) is this node's other half — split into its
// own file because tasks.md's own focused test command names them
// separately ('TestTerminal|TestTruncation').
//
// # Coverage of already-implemented behavior, not a new feature
//
// stream.go's run() already recognises doneSentinel and returns
// immediately on a match (slice 1), which structurally guarantees
// R-ATS-014 for free: nothing after that return is ever reached, in the
// same Feed call or any later one. Every test in this file's R-ATS-012/014
// sections passed on first execution against the unmodified slice-1/2
// code — non-vacuity is proven by a staged, reverted mutation instead of a
// compile-fail RED (this milestone's apply-progress record carries the
// mutation and its output), the same disclosed pattern slice 2 used for
// its own already-satisfied cases.
//
// S-ATS-046's own fixture below models a well-formed sentinel frame — its
// own mandatory blank line present — followed immediately by EOF with no
// further bytes; see the test's own doc comment and spec.md's rev-3
// revision note (slice-6 coordinator ruling) for why this is the
// scenario's sole correct reading, not one of two competing ones.

package openaicompat

import (
	"context"
	"testing"

	"github.com/cachicamas/backend/agent/src/ai"
)

// ---------------------------------------------------------------------
// R-ATS-012 — data: [DONE] is clean termination, never payload, never
// truncation (S-ATS-044…047; C5).
// ---------------------------------------------------------------------

// TestTerminal_TerminalChunkThenDone_EndsCleanlyWithCompletion covers
// S-ATS-044 and S-ATS-045: a transcript ending with a terminal chunk, then
// data: [DONE], then EOF drains to a clean completion with no failure
// event, and no emitted delta carries the sentinel's own literal text.
func TestTerminal_TerminalChunkThenDone_EndsCleanlyWithCompletion(t *testing.T) {
	t.Parallel()

	transcript := "" +
		"data: {\"id\":\"chatcmpl-term\",\"model\":\"m\",\"object\":\"chat.completion.chunk\",\"choices\":[{\"index\":0,\"delta\":{},\"finish_reason\":\"stop\"}]}\n\n" +
		"data: [DONE]\n\n"
	server := sseServer(t, transcript)
	defer server.Close()

	c := mustClient(t, server.URL)
	ch, err := c.Stream(context.Background(), validRequest(t))
	if err != nil {
		t.Fatalf("Stream() error = %v, want nil (S-ATS-044)", err)
	}
	events := drainAll(t, ch)

	if len(events) == 0 {
		t.Fatal("drained zero events, want at least a terminal event (S-ATS-044)")
	}
	last := events[len(events)-1]
	if last.Kind() != ai.EventKindCompletion {
		t.Fatalf("last event kind = %v, want EventKindCompletion (S-ATS-044)", last.Kind())
	}
	for _, ev := range events {
		if ev.Kind() == ai.EventKindError {
			t.Fatalf("unexpected error event %+v, want a clean completion with no failure (S-ATS-044)", ev)
		}
	}

	// S-ATS-045: the sentinel produced no payload — no TextDelta anywhere
	// carries its literal text.
	for _, ev := range events {
		if d, ok := ev.TextDelta(); ok && d.Delta() == doneSentinel {
			t.Errorf("a TextDelta carries the literal %q — the sentinel produced payload (S-ATS-045)", doneSentinel)
		}
	}
}

// TestTerminal_DoneWithNoPaddingBeyondMandatoryBlankLine_SameOutcome covers
// S-ATS-046 (spec.md rev 3). The scenario now reads precisely: the
// sentinel's own SSE-mandatory blank line — the ONE line every S-ATS-04x
// fixture in this file already needs for [DONE] to dispatch as a Frame at
// all — is immediately followed by EOF, with zero further bytes of any
// kind (no keep-alive, no extra terminator, no padding).
//
// # Why the fixture never models an unterminated sentinel frame
//
// An earlier reading of this scenario's rev-1/rev-2 wording ("no trailing
// blank line before EOF") could be taken as the sentinel's own mandatory
// blank line itself missing, leaving the literal bytes "data: [DONE]"
// pending and undispatched at EOF. That shape was empirically probed
// against Decoder.Finish() (a frozen AI-27 file this milestone MUST NOT
// amend, per spec.md's own "binding predecessors, cited by identifier and
// never amended") and confirmed to return ErrTruncated, because the
// sentinel's own frame is never fully dispatched by Decoder.Feed for want
// of its terminating blank line — per WHATWG dispatch, an unterminated
// event is never received at all. That is R-ATS-013's truncation case, not
// this rule's clean termination, and R-ATS-013's own unconditional text
// forbids reading it any other way. spec.md's rev-3 revision note (the
// slice-6 coordinator ruling) corrects the scenario text itself to the
// well-formed-frame-at-exact-EOF reading this fixture already models —
// this test's fixture and assertions are unchanged by that correction;
// only the scenario's own wording was.
func TestTerminal_DoneWithNoPaddingBeyondMandatoryBlankLine_SameOutcome(t *testing.T) {
	t.Parallel()

	transcript := "" +
		"data: {\"id\":\"chatcmpl-pad\",\"model\":\"m\",\"object\":\"chat.completion.chunk\",\"choices\":[{\"index\":0,\"delta\":{},\"finish_reason\":\"stop\"}]}\n\n" +
		"data: [DONE]\n\n"
	server := sseServer(t, transcript)
	defer server.Close()

	c := mustClient(t, server.URL)
	ch, err := c.Stream(context.Background(), validRequest(t))
	if err != nil {
		t.Fatalf("Stream() error = %v, want nil (S-ATS-046)", err)
	}
	events := drainAll(t, ch)
	if len(events) == 0 {
		t.Fatal("drained zero events (S-ATS-046)")
	}
	last := events[len(events)-1]
	if last.Kind() != ai.EventKindCompletion {
		t.Fatalf("last event kind = %v, want EventKindCompletion — outcome must equal the terminated form, sentinel never reported as truncation (S-ATS-046)", last.Kind())
	}
	for _, ev := range events {
		if ev.Kind() == ai.EventKindError {
			t.Fatalf("unexpected error event %+v — the sentinel must never be reported as truncation (S-ATS-046)", ev)
		}
	}
}

// S-ATS-047 is [inspection], not [test]: discharged by a reviewer reading
// the shipped source and fixture set — stream.go's doneSentinel constant
// already cites C5 in its own doc comment, and this file's fixtures above
// carry the exact bytes "data: [DONE]" — see tasks.md's own evidence log.

// ---------------------------------------------------------------------
// R-ATS-014 — frames arriving after the terminal sentinel and before EOF
// are ignored (S-ATS-052…054).
// ---------------------------------------------------------------------

// TestTerminal_WellFormedChunkAfterSentinel_Ignored covers S-ATS-052: a
// well-formed content chunk arriving after data: [DONE] and before EOF
// contributes no text delta, and the stream's last event remains the
// completion determined before the sentinel.
func TestTerminal_WellFormedChunkAfterSentinel_Ignored(t *testing.T) {
	t.Parallel()

	transcript := "" +
		"data: {\"id\":\"chatcmpl-post\",\"model\":\"m\",\"object\":\"chat.completion.chunk\",\"choices\":[{\"index\":0,\"delta\":{},\"finish_reason\":\"stop\"}]}\n\n" +
		"data: [DONE]\n\n" +
		"data: {\"id\":\"chatcmpl-post\",\"model\":\"m\",\"object\":\"chat.completion.chunk\",\"choices\":[{\"index\":0,\"delta\":{\"content\":\"ghost\"},\"finish_reason\":null}]}\n\n"
	server := sseServer(t, transcript)
	defer server.Close()

	c := mustClient(t, server.URL)
	ch, err := c.Stream(context.Background(), validRequest(t))
	if err != nil {
		t.Fatalf("Stream() error = %v, want nil (S-ATS-052)", err)
	}
	events := drainAll(t, ch)
	if len(events) == 0 {
		t.Fatal("drained zero events (S-ATS-052)")
	}
	for _, ev := range events {
		if d, ok := ev.TextDelta(); ok {
			t.Errorf("unexpected TextDelta %q derived from a post-sentinel frame, want none (S-ATS-052)", d.Delta())
		}
	}
	last := events[len(events)-1]
	if last.Kind() != ai.EventKindCompletion {
		t.Fatalf("last event kind = %v, want EventKindCompletion — the terminal event determined before the sentinel must stand (S-ATS-052)", last.Kind())
	}
}

// TestTerminal_MalformedFrameAfterSentinel_StillEndsCleanly covers
// S-ATS-053: a malformed frame arriving after data: [DONE] is neither
// parsed nor surfaced as a failure — the stream still ends cleanly.
func TestTerminal_MalformedFrameAfterSentinel_StillEndsCleanly(t *testing.T) {
	t.Parallel()

	transcript := "" +
		"data: {\"id\":\"chatcmpl-mal\",\"model\":\"m\",\"object\":\"chat.completion.chunk\",\"choices\":[{\"index\":0,\"delta\":{},\"finish_reason\":\"stop\"}]}\n\n" +
		"data: [DONE]\n\n" +
		"data: {this is not valid json at all\n\n"
	server := sseServer(t, transcript)
	defer server.Close()

	c := mustClient(t, server.URL)
	ch, err := c.Stream(context.Background(), validRequest(t))
	if err != nil {
		t.Fatalf("Stream() error = %v, want nil (S-ATS-053)", err)
	}
	events := drainAll(t, ch)
	if len(events) == 0 {
		t.Fatal("drained zero events (S-ATS-053)")
	}
	last := events[len(events)-1]
	if last.Kind() != ai.EventKindCompletion {
		t.Fatalf("last event kind = %v, want EventKindCompletion — a post-sentinel malformed frame must not surface as a failure (S-ATS-053)", last.Kind())
	}
	for _, ev := range events {
		if ev.Kind() == ai.EventKindError {
			t.Fatalf("unexpected error event %+v derived from a post-sentinel malformed frame (S-ATS-053)", ev)
		}
	}
}

// TestTerminal_DuplicateSentinel_ExactlyOneTerminalEvent covers S-ATS-054:
// a second data: [DONE] following the first is itself a post-sentinel
// frame, ignored — exactly one terminal event is ever emitted.
func TestTerminal_DuplicateSentinel_ExactlyOneTerminalEvent(t *testing.T) {
	t.Parallel()

	transcript := "" +
		"data: {\"id\":\"chatcmpl-dup\",\"model\":\"m\",\"object\":\"chat.completion.chunk\",\"choices\":[{\"index\":0,\"delta\":{},\"finish_reason\":\"stop\"}]}\n\n" +
		"data: [DONE]\n\n" +
		"data: [DONE]\n\n"
	server := sseServer(t, transcript)
	defer server.Close()

	c := mustClient(t, server.URL)
	ch, err := c.Stream(context.Background(), validRequest(t))
	if err != nil {
		t.Fatalf("Stream() error = %v, want nil (S-ATS-054)", err)
	}
	events := drainAll(t, ch)

	terminalCount := 0
	for _, ev := range events {
		if ev.Kind() == ai.EventKindCompletion || ev.Kind() == ai.EventKindError {
			terminalCount++
		}
	}
	if terminalCount != 1 {
		t.Fatalf("terminal event count = %d, want exactly 1 (S-ATS-054): %+v", terminalCount, kindsOf(events))
	}
	if report := ai.CheckStream(events); report.Violation() != nil {
		t.Errorf("CheckStream violation = %v, want nil — no duplicate-terminal violation (S-ATS-054)", report.Violation())
	}
}
