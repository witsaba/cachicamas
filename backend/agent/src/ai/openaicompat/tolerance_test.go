// AI-28.4 — unknown and delta-less tolerance: R-ATS-017 (unknown frame
// types and undeclared delta fields skipped without corrupting adjacent
// accumulation, S-ATS-063…067), R-ATS-018 (delta-less/content-less shapes
// normalize cleanly, S-ATS-068…070) and R-ATS-019 (keep-alive frames
// interleaved anywhere are inert, S-ATS-071…073).
//
// # Which scenarios need new production code, and which do not
//
// R-ATS-017's SSE-event-type clause and its object-discriminator clause
// are genuinely new: before this slice, run()'s frame loop did not
// inspect frame.Event at all, and wireChunk carried no Object field, so
// both a non-default-event-type frame and an object-mismatched frame
// would have been decoded and applied as if ordinary. S-ATS-063/066 below
// use ADVERSARIAL fixtures (an unwanted "INTRUDER" content string riding
// the frame that must be skipped) specifically so the test would catch a
// missing skip rather than passing vacuously against an empty-choices
// no-op frame.
//
// R-ATS-017's undeclared-field clause (S-ATS-064/065/067), R-ATS-018
// (S-ATS-068/069) and R-ATS-019 (S-ATS-071…073) are already satisfied by
// existing code: Go's encoding/json silently ignores unknown struct
// fields by default (no DisallowUnknownFields anywhere in this package),
// contentText's existing present/absent trichotomy already normalizes a
// content-free or empty-string stream correctly (S-ATS-068 is the same
// shape as slice 2's own S-ATS-031; S-ATS-069 is new integration-level
// coverage of a single chunk carrying content:"" AND finish_reason
// simultaneously), and SSE comment lines are already invisible at the
// decoder level (never dispatched as a Frame at all, so run()'s loop never
// even sees them — a decoder.go, AI-27, frozen-file property). Where a
// test in this half passed on first execution, non-vacuity is proven
// either by a staged, reverted mutation or, for the keep-alive cases, by
// the fixture's own comparative (plain-vs-keepalive twin) design — see
// this milestone's apply-progress record.

package openaicompat

import (
	"context"
	"testing"

	"github.com/cachicamas/backend/agent/src/ai"
)

// ---------------------------------------------------------------------
// R-ATS-017 — unknown frame types / undeclared delta fields skipped
// without corrupting adjacent accumulation (S-ATS-063…067).
// ---------------------------------------------------------------------

// TestTolerance_UnrecognisedSSEEventType_SkippedBetweenTwoContentChunks
// covers S-ATS-063: a frame with a non-default SSE event type — carrying
// an adversarial content string of its own — is skipped entirely, never
// decoded or applied, leaving the two surrounding deltas untouched.
func TestTolerance_UnrecognisedSSEEventType_SkippedBetweenTwoContentChunks(t *testing.T) {
	t.Parallel()

	transcript := "" +
		"data: {\"id\":\"chatcmpl-t\",\"model\":\"m\",\"object\":\"chat.completion.chunk\",\"choices\":[{\"index\":0,\"delta\":{\"role\":\"assistant\",\"content\":\"alpha\"},\"finish_reason\":null}]}\n\n" +
		"event: ping\ndata: {\"id\":\"chatcmpl-t\",\"model\":\"m\",\"object\":\"chat.completion.chunk\",\"choices\":[{\"index\":0,\"delta\":{\"content\":\"INTRUDER\"},\"finish_reason\":null}]}\n\n" +
		"data: {\"id\":\"chatcmpl-t\",\"model\":\"m\",\"object\":\"chat.completion.chunk\",\"choices\":[{\"index\":0,\"delta\":{\"content\":\"omega\"},\"finish_reason\":null}]}\n\n" +
		"data: {\"id\":\"chatcmpl-t\",\"model\":\"m\",\"object\":\"chat.completion.chunk\",\"choices\":[{\"index\":0,\"delta\":{},\"finish_reason\":\"stop\"}]}\n\n" +
		"data: [DONE]\n\n"
	server := sseServer(t, transcript)
	defer server.Close()
	c := mustClient(t, server.URL)
	ch, err := c.Stream(context.Background(), validRequest(t))
	if err != nil {
		t.Fatalf("Stream() error = %v, want nil (S-ATS-063)", err)
	}
	events := drainAll(t, ch)

	var deltas []string
	for _, ev := range events {
		if d, ok := ev.TextDelta(); ok {
			deltas = append(deltas, d.Delta())
		}
	}
	want := []string{"alpha", "omega"}
	if len(deltas) != len(want) {
		t.Fatalf("deltas = %q, want %q — the unrecognised-event-type frame must be skipped, not applied (S-ATS-063)", deltas, want)
	}
	for i := range want {
		if deltas[i] != want[i] {
			t.Errorf("delta[%d] = %q, want %q (S-ATS-063)", i, deltas[i], want[i])
		}
	}
	for _, ev := range events {
		if ev.Kind() == ai.EventKindError {
			t.Fatalf("unexpected error event, want the unrecognised event type silently skipped (S-ATS-063): %+v", ev)
		}
	}
}

// TestTolerance_UndeclaredObfuscationField_ContentStillMapped covers
// S-ATS-064: a delta carrying the documented, undeclared obfuscation field
// (C7) alongside content is accepted, with content mapped normally.
func TestTolerance_UndeclaredObfuscationField_ContentStillMapped(t *testing.T) {
	t.Parallel()

	transcript := "" +
		"data: {\"id\":\"chatcmpl-o\",\"model\":\"m\",\"object\":\"chat.completion.chunk\",\"choices\":[{\"index\":0,\"delta\":{\"content\":\"bravo\",\"obfuscation\":\"xxxxxx\"},\"finish_reason\":null}]}\n\n" +
		"data: {\"id\":\"chatcmpl-o\",\"model\":\"m\",\"object\":\"chat.completion.chunk\",\"choices\":[{\"index\":0,\"delta\":{},\"finish_reason\":\"stop\"}]}\n\n" +
		"data: [DONE]\n\n"
	server := sseServer(t, transcript)
	defer server.Close()
	c := mustClient(t, server.URL)
	ch, err := c.Stream(context.Background(), validRequest(t))
	if err != nil {
		t.Fatalf("Stream() error = %v, want nil (S-ATS-064)", err)
	}
	events := drainAll(t, ch)
	var deltas []string
	for _, ev := range events {
		if d, ok := ev.TextDelta(); ok {
			deltas = append(deltas, d.Delta())
		}
		if ev.Kind() == ai.EventKindError {
			t.Fatalf("unexpected error event (S-ATS-064): %+v", ev)
		}
	}
	if len(deltas) != 1 || deltas[0] != "bravo" {
		t.Errorf("deltas = %q, want exactly [\"bravo\"] (S-ATS-064)", deltas)
	}
}

// TestTolerance_InventedSiblingFields_ContentStillMapped covers S-ATS-065:
// a delta carrying two entirely invented sibling fields alongside content
// is accepted, with content mapped normally and no failure.
func TestTolerance_InventedSiblingFields_ContentStillMapped(t *testing.T) {
	t.Parallel()

	transcript := "" +
		"data: {\"id\":\"chatcmpl-i\",\"model\":\"m\",\"object\":\"chat.completion.chunk\",\"choices\":[{\"index\":0,\"delta\":{\"content\":\"charlie\",\"made_up_field_one\":42,\"made_up_field_two\":{\"nested\":true}},\"finish_reason\":null}]}\n\n" +
		"data: {\"id\":\"chatcmpl-i\",\"model\":\"m\",\"object\":\"chat.completion.chunk\",\"choices\":[{\"index\":0,\"delta\":{},\"finish_reason\":\"stop\"}]}\n\n" +
		"data: [DONE]\n\n"
	server := sseServer(t, transcript)
	defer server.Close()
	c := mustClient(t, server.URL)
	ch, err := c.Stream(context.Background(), validRequest(t))
	if err != nil {
		t.Fatalf("Stream() error = %v, want nil (S-ATS-065)", err)
	}
	events := drainAll(t, ch)
	var deltas []string
	for _, ev := range events {
		if d, ok := ev.TextDelta(); ok {
			deltas = append(deltas, d.Delta())
		}
		if ev.Kind() == ai.EventKindError {
			t.Fatalf("unexpected error event (S-ATS-065): %+v", ev)
		}
	}
	if len(deltas) != 1 || deltas[0] != "charlie" {
		t.Errorf("deltas = %q, want exactly [\"charlie\"] (S-ATS-065)", deltas)
	}
}

// TestTolerance_NonChunkObjectDiscriminator_SkippedBetweenTwoContentChunks
// covers S-ATS-066: a frame whose JSON object carries "object":
// "chat.completion" — not the chunk discriminator — and an adversarial
// content string of its own is skipped entirely, leaving the two
// surrounding deltas unchanged.
func TestTolerance_NonChunkObjectDiscriminator_SkippedBetweenTwoContentChunks(t *testing.T) {
	t.Parallel()

	transcript := "" +
		"data: {\"id\":\"chatcmpl-n\",\"model\":\"m\",\"object\":\"chat.completion.chunk\",\"choices\":[{\"index\":0,\"delta\":{\"role\":\"assistant\",\"content\":\"uno\"},\"finish_reason\":null}]}\n\n" +
		"data: {\"object\":\"chat.completion\",\"id\":\"not-a-chunk\",\"model\":\"m\",\"choices\":[{\"index\":0,\"delta\":{\"content\":\"INTRUDER\"},\"finish_reason\":null}]}\n\n" +
		"data: {\"id\":\"chatcmpl-n\",\"model\":\"m\",\"object\":\"chat.completion.chunk\",\"choices\":[{\"index\":0,\"delta\":{\"content\":\"dos\"},\"finish_reason\":null}]}\n\n" +
		"data: {\"id\":\"chatcmpl-n\",\"model\":\"m\",\"object\":\"chat.completion.chunk\",\"choices\":[{\"index\":0,\"delta\":{},\"finish_reason\":\"stop\"}]}\n\n" +
		"data: [DONE]\n\n"
	server := sseServer(t, transcript)
	defer server.Close()
	c := mustClient(t, server.URL)
	ch, err := c.Stream(context.Background(), validRequest(t))
	if err != nil {
		t.Fatalf("Stream() error = %v, want nil (S-ATS-066)", err)
	}
	events := drainAll(t, ch)
	var deltas []string
	for _, ev := range events {
		if d, ok := ev.TextDelta(); ok {
			deltas = append(deltas, d.Delta())
		}
		if ev.Kind() == ai.EventKindError {
			t.Fatalf("unexpected error event, want the non-chunk object skipped (S-ATS-066): %+v", ev)
		}
	}
	want := []string{"uno", "dos"}
	if len(deltas) != len(want) {
		t.Fatalf("deltas = %q, want %q — the non-chunk-object frame must be skipped, not applied (S-ATS-066)", deltas, want)
	}
	for i := range want {
		if deltas[i] != want[i] {
			t.Errorf("delta[%d] = %q, want %q (S-ATS-066)", i, deltas[i], want[i])
		}
	}
}

// TestTolerance_RefusalWithNoContent_NoTextDeltaAndFollowingContentUnaffected
// covers S-ATS-067: a delta carrying only refusal (C7), no content, emits
// no text delta, and the following content chunk's delta is unaffected —
// refusal mapping is not this milestone's.
func TestTolerance_RefusalWithNoContent_NoTextDeltaAndFollowingContentUnaffected(t *testing.T) {
	t.Parallel()

	transcript := "" +
		"data: {\"id\":\"chatcmpl-r\",\"model\":\"m\",\"object\":\"chat.completion.chunk\",\"choices\":[{\"index\":0,\"delta\":{\"refusal\":\"no\"},\"finish_reason\":null}]}\n\n" +
		"data: {\"id\":\"chatcmpl-r\",\"model\":\"m\",\"object\":\"chat.completion.chunk\",\"choices\":[{\"index\":0,\"delta\":{\"content\":\"after\"},\"finish_reason\":null}]}\n\n" +
		"data: {\"id\":\"chatcmpl-r\",\"model\":\"m\",\"object\":\"chat.completion.chunk\",\"choices\":[{\"index\":0,\"delta\":{},\"finish_reason\":\"stop\"}]}\n\n" +
		"data: [DONE]\n\n"
	server := sseServer(t, transcript)
	defer server.Close()
	c := mustClient(t, server.URL)
	ch, err := c.Stream(context.Background(), validRequest(t))
	if err != nil {
		t.Fatalf("Stream() error = %v, want nil (S-ATS-067)", err)
	}
	events := drainAll(t, ch)
	var deltas []string
	for _, ev := range events {
		if d, ok := ev.TextDelta(); ok {
			deltas = append(deltas, d.Delta())
		}
	}
	if len(deltas) != 1 || deltas[0] != "after" {
		t.Errorf("deltas = %q, want exactly [\"after\"] — refusal mapping is not this milestone's (S-ATS-067)", deltas)
	}
}

// ---------------------------------------------------------------------
// R-ATS-018 — delta-less and content-less shapes normalize cleanly
// (S-ATS-068…070).
// ---------------------------------------------------------------------

// TestTolerance_NoContentAnywhere_NormalizesToResponseStartThenCompletion
// covers S-ATS-068: a role-only opening chunk and a terminal chunk with no
// content string anywhere normalizes to exactly ResponseStart, Completion.
func TestTolerance_NoContentAnywhere_NormalizesToResponseStartThenCompletion(t *testing.T) {
	t.Parallel()

	transcript := "" +
		"data: {\"id\":\"chatcmpl-e\",\"model\":\"m\",\"object\":\"chat.completion.chunk\",\"choices\":[{\"index\":0,\"delta\":{\"role\":\"assistant\"},\"finish_reason\":null}]}\n\n" +
		"data: {\"id\":\"chatcmpl-e\",\"model\":\"m\",\"object\":\"chat.completion.chunk\",\"choices\":[{\"index\":0,\"delta\":{},\"finish_reason\":\"stop\"}]}\n\n" +
		"data: [DONE]\n\n"
	server := sseServer(t, transcript)
	defer server.Close()
	c := mustClient(t, server.URL)
	ch, err := c.Stream(context.Background(), validRequest(t))
	if err != nil {
		t.Fatalf("Stream() error = %v, want nil (S-ATS-068)", err)
	}
	events := drainAll(t, ch)
	want := []ai.EventKind{ai.EventKindResponseStart, ai.EventKindCompletion}
	if len(events) != len(want) {
		t.Fatalf("drained kinds = %v, want %v (S-ATS-068)", kindsOf(events), want)
	}
	for i := range want {
		if events[i].Kind() != want[i] {
			t.Errorf("events[%d].Kind() = %v, want %v (S-ATS-068)", i, events[i].Kind(), want[i])
		}
	}
	if report := ai.CheckStream(events); report.Violation() != nil {
		t.Errorf("CheckStream violation = %v, want nil (S-ATS-068)", report.Violation())
	}
}

// TestTolerance_EmptyStringContent_MintsOneEmptyDelta covers S-ATS-069: a
// single chunk carrying content:"" AND the terminal finish_reason
// simultaneously mints a block containing exactly one empty delta.
func TestTolerance_EmptyStringContent_MintsOneEmptyDelta(t *testing.T) {
	t.Parallel()

	transcript := "" +
		"data: {\"id\":\"chatcmpl-z\",\"model\":\"m\",\"object\":\"chat.completion.chunk\",\"choices\":[{\"index\":0,\"delta\":{\"content\":\"\"},\"finish_reason\":\"stop\"}]}\n\n" +
		"data: [DONE]\n\n"
	server := sseServer(t, transcript)
	defer server.Close()
	c := mustClient(t, server.URL)
	ch, err := c.Stream(context.Background(), validRequest(t))
	if err != nil {
		t.Fatalf("Stream() error = %v, want nil (S-ATS-069)", err)
	}
	events := drainAll(t, ch)
	want := []ai.EventKind{
		ai.EventKindResponseStart, ai.EventKindTextBlockStart,
		ai.EventKindTextDelta, ai.EventKindTextBlockEnd, ai.EventKindCompletion,
	}
	if len(events) != len(want) {
		t.Fatalf("drained kinds = %v, want %v (S-ATS-069)", kindsOf(events), want)
	}
	for i := range want {
		if events[i].Kind() != want[i] {
			t.Errorf("events[%d].Kind() = %v, want %v (S-ATS-069)", i, events[i].Kind(), want[i])
		}
	}
	d, ok := events[2].TextDelta()
	if !ok {
		t.Fatal("events[2] carries no TextDelta payload (S-ATS-069)")
	}
	if d.Delta() != "" {
		t.Errorf("Delta() = %q, want empty string (S-ATS-069)", d.Delta())
	}
}

// S-ATS-070 is [inspection], not [test]: discharged by a reviewer reading
// the shipped mapping source (chunk.go, stream_state.go) and confirming no
// code path enforces a minimum delta count, minimum block count or
// non-empty-text requirement anywhere — see tasks.md's own evidence log.

// ---------------------------------------------------------------------
// R-ATS-019 — keep-alive frames interleaved anywhere are inert
// (S-ATS-071…073).
// ---------------------------------------------------------------------

// TestKeepAlive_BeforeEveryChunk_IdenticalEventSequence covers S-ATS-071:
// interleaving a keep-alive comment before every chunk produces an
// event sequence identical in kind, order, sequence number and payload to
// the keep-alive-free twin.
func TestKeepAlive_BeforeEveryChunk_IdenticalEventSequence(t *testing.T) {
	t.Parallel()

	plain := "" +
		"data: {\"id\":\"chatcmpl-k\",\"model\":\"m\",\"object\":\"chat.completion.chunk\",\"choices\":[{\"index\":0,\"delta\":{\"role\":\"assistant\",\"content\":\"one\"},\"finish_reason\":null}]}\n\n" +
		"data: {\"id\":\"chatcmpl-k\",\"model\":\"m\",\"object\":\"chat.completion.chunk\",\"choices\":[{\"index\":0,\"delta\":{\"content\":\"two\"},\"finish_reason\":null}]}\n\n" +
		"data: {\"id\":\"chatcmpl-k\",\"model\":\"m\",\"object\":\"chat.completion.chunk\",\"choices\":[{\"index\":0,\"delta\":{},\"finish_reason\":\"stop\"}]}\n\n" +
		"data: [DONE]\n\n"
	withKeepAlives := "" +
		": keep-alive\n\n" +
		"data: {\"id\":\"chatcmpl-k\",\"model\":\"m\",\"object\":\"chat.completion.chunk\",\"choices\":[{\"index\":0,\"delta\":{\"role\":\"assistant\",\"content\":\"one\"},\"finish_reason\":null}]}\n\n" +
		": keep-alive\n\n" +
		"data: {\"id\":\"chatcmpl-k\",\"model\":\"m\",\"object\":\"chat.completion.chunk\",\"choices\":[{\"index\":0,\"delta\":{\"content\":\"two\"},\"finish_reason\":null}]}\n\n" +
		": keep-alive\n\n" +
		"data: {\"id\":\"chatcmpl-k\",\"model\":\"m\",\"object\":\"chat.completion.chunk\",\"choices\":[{\"index\":0,\"delta\":{},\"finish_reason\":\"stop\"}]}\n\n" +
		": keep-alive\n\n" +
		"data: [DONE]\n\n"

	drain := func(transcript string) []ai.Event {
		server := sseServer(t, transcript)
		defer server.Close()
		c := mustClient(t, server.URL)
		ch, err := c.Stream(context.Background(), validRequest(t))
		if err != nil {
			t.Fatalf("Stream() error = %v, want nil (S-ATS-071)", err)
		}
		return drainAll(t, ch)
	}

	plainEvents := drain(plain)
	keepAliveEvents := drain(withKeepAlives)

	if len(plainEvents) != len(keepAliveEvents) {
		t.Fatalf("event counts differ: plain=%d keepAlive=%d, want equal (S-ATS-071)", len(plainEvents), len(keepAliveEvents))
	}
	for i := range plainEvents {
		if plainEvents[i].Kind() != keepAliveEvents[i].Kind() {
			t.Errorf("kind[%d] differs: plain=%v keepAlive=%v (S-ATS-071)", i, plainEvents[i].Kind(), keepAliveEvents[i].Kind())
		}
		if plainEvents[i].Sequence() != keepAliveEvents[i].Sequence() {
			t.Errorf("sequence[%d] differs: plain=%d keepAlive=%d (S-ATS-071)", i, plainEvents[i].Sequence(), keepAliveEvents[i].Sequence())
		}
		pd, pok := plainEvents[i].TextDelta()
		kd, kok := keepAliveEvents[i].TextDelta()
		if pok != kok {
			t.Errorf("TextDelta presence[%d] differs: plain=%v keepAlive=%v (S-ATS-071)", i, pok, kok)
		}
		if pok && kok && pd.Delta() != kd.Delta() {
			t.Errorf("delta[%d] differs: plain=%q keepAlive=%q (S-ATS-071)", i, pd.Delta(), kd.Delta())
		}
	}
}

// TestKeepAlive_BetweenTwoDeltasOfOneBlock_ConcatenationByteIdentical
// covers S-ATS-072: a keep-alive between two deltas of one block leaves
// the concatenated deltas byte-identical to the keep-alive-free form.
func TestKeepAlive_BetweenTwoDeltasOfOneBlock_ConcatenationByteIdentical(t *testing.T) {
	t.Parallel()

	transcript := "" +
		"data: {\"id\":\"chatcmpl-m\",\"model\":\"m\",\"object\":\"chat.completion.chunk\",\"choices\":[{\"index\":0,\"delta\":{\"content\":\"foo\"},\"finish_reason\":null}]}\n\n" +
		": keep-alive between deltas\n\n" +
		"data: {\"id\":\"chatcmpl-m\",\"model\":\"m\",\"object\":\"chat.completion.chunk\",\"choices\":[{\"index\":0,\"delta\":{\"content\":\"bar\"},\"finish_reason\":null}]}\n\n" +
		"data: {\"id\":\"chatcmpl-m\",\"model\":\"m\",\"object\":\"chat.completion.chunk\",\"choices\":[{\"index\":0,\"delta\":{},\"finish_reason\":\"stop\"}]}\n\n" +
		"data: [DONE]\n\n"
	server := sseServer(t, transcript)
	defer server.Close()
	c := mustClient(t, server.URL)
	ch, err := c.Stream(context.Background(), validRequest(t))
	if err != nil {
		t.Fatalf("Stream() error = %v, want nil (S-ATS-072)", err)
	}
	events := drainAll(t, ch)
	var got string
	for _, ev := range events {
		if d, ok := ev.TextDelta(); ok {
			got += d.Delta()
		}
	}
	if got != "foobar" {
		t.Errorf("concatenated = %q, want %q (S-ATS-072)", got, "foobar")
	}
}

// TestKeepAlive_AfterTerminalChunkBeforeSentinel_TerminatesCleanly covers
// S-ATS-073: a keep-alive between the terminal chunk and the sentinel
// leaves the stream terminating cleanly, terminal event unchanged.
func TestKeepAlive_AfterTerminalChunkBeforeSentinel_TerminatesCleanly(t *testing.T) {
	t.Parallel()

	transcript := "" +
		"data: {\"id\":\"chatcmpl-p\",\"model\":\"m\",\"object\":\"chat.completion.chunk\",\"choices\":[{\"index\":0,\"delta\":{},\"finish_reason\":\"stop\"}]}\n\n" +
		": keep-alive after terminal, before sentinel\n\n" +
		"data: [DONE]\n\n"
	server := sseServer(t, transcript)
	defer server.Close()
	c := mustClient(t, server.URL)
	ch, err := c.Stream(context.Background(), validRequest(t))
	if err != nil {
		t.Fatalf("Stream() error = %v, want nil (S-ATS-073)", err)
	}
	events := drainAll(t, ch)
	if len(events) == 0 {
		t.Fatal("drained zero events (S-ATS-073)")
	}
	last := events[len(events)-1]
	if last.Kind() != ai.EventKindCompletion {
		t.Fatalf("last event kind = %v, want EventKindCompletion (S-ATS-073)", last.Kind())
	}
	for _, ev := range events {
		if ev.Kind() == ai.EventKindError {
			t.Fatalf("unexpected error event (S-ATS-073): %+v", ev)
		}
	}
}
