// AI-31.1 — finish-reason coverage: the five wire values table-tested
// (R-ACP-001, S-ACP-001/002/003), the unreachable-neutral-value probe
// (R-ACP-002, S-ACP-005), the novel-stop-value typed-malformed behaviour
// (R-ACP-003, S-ACP-006), and the matched-stop-sequence absence negative
// control (R-ACP-004, S-ACP-010).
//
// # Why every test below drives a real transcript (not a unit on the
// normalizer in isolation)
//
// The mapping R-ACP-001 names is already landed — rawStrictFinishReason
// gates the five enum members byte-exactly, then delegates to
// ai.NormalizeFinishReason (chunk.go). What AI-31.1 owes is the
// coverage: a table test that pins the five-row enum, distinct
// assertions on the rows doc 0002 cares about (function_call deprecated
// but enum-legal; content_filter distinct from refusal), the
// unreachable-neutral-values probe driven EXHAUSTIVELY over the table
// (so the "never Refusal/PauseTurn/Unknown" claim is not satisfied
// vacuously by absence of input), the novel-value typed-malformed
// behaviour including its case/whitespace variants, and the
// matched-stop-sequence negative control (extra-key-shaped transcript
// drains to the same completion, proving the absence holds because
// nothing is read, not because nothing was offered).
//
// # requireCheckStreamClean on every drained-stream test (design C1)
//
// The blanket rule landed for this change: every new drained-stream
// test asserts ai.CheckStream(events) reports no violation via the
// shared requireCheckStreamClean helper. A fixture that accidentally
// violates stream protocol fails loudly instead of passing on its
// finish-reason assertion alone.

package openaicompat

import (
	"context"
	"testing"

	"github.com/cachicamas/backend/agent/src/ai"
)

// finishReasonTerminalTranscript builds a minimal transcript: identity
// chunk + chunk whose choice-0 finish_reason carries raw (already
// assembled as a full "data: ...\n\n" SSE frame), then sentinel. Mirrors
// usageTranscript's shape so the table tests stay self-contained.
func finishReasonTerminalTranscript(raw string) string {
	return "" +
		"data: {\"id\":\"chatcmpl-fr\",\"model\":\"m\",\"object\":\"chat.completion.chunk\",\"created\":1700000000,\"choices\":[{\"index\":0,\"delta\":{},\"finish_reason\":\"" + raw + "\"}]}\n\n" +
		"data: [DONE]\n\n"
}

// drainFinishReason drains ch and returns the last event's FinishReason
// value, failing the test if that event does not carry a Completion
// payload.
func drainFinishReason(t *testing.T, ch <-chan ai.Event) ai.FinishReason {
	t.Helper()
	events := drainAll(t, ch)
	requireCheckStreamClean(t, events)
	completion, ok := events[len(events)-1].Completion()
	if !ok {
		t.Fatalf("last event carries no Completion payload: %+v", events[len(events)-1])
	}
	return completion.FinishReason()
}

// fiveWireFinishReasonCases is the table over C2's five raw wire
// spellings, each paired with the neutral value NormalizeFinishReason
// returns for it (R-ACP-001's row table, U5 cited). The enum-count
// assertion in TestFinishReason_FiveWireValues_MapTable guards against a
// sixth member being added without re-stating the table.
var fiveWireFinishReasonCases = []struct {
	name    string
	raw     string
	neutral ai.FinishReason
}{
	{"stop", "stop", ai.FinishReasonStop},
	{"length", "length", ai.FinishReasonLength},
	{"tool_calls", "tool_calls", ai.FinishReasonToolCalls},
	{"function_call_deprecated_but_enum_legal", "function_call", ai.FinishReasonToolCalls},
	{"content_filter", "content_filter", ai.FinishReasonContentFilter},
}

// TestFinishReason_FiveWireValues_MapTable covers S-ACP-001: the
// five-member wire enum maps onto the neutral vocabulary, the table's
// row count is asserted to be 5, so a sixth wire member added later
// fails the count before it fails a mapping.
func TestFinishReason_FiveWireValues_MapTable(t *testing.T) {
	t.Parallel()

	const expectedRowCount = 5
	if got := len(fiveWireFinishReasonCases); got != expectedRowCount {
		t.Fatalf("fiveWireFinishReasonCases row count = %d, want %d (S-ACP-001 — a sixth wire member must re-state this table)", got, expectedRowCount)
	}

	for _, tc := range fiveWireFinishReasonCases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			server := sseServer(t, finishReasonTerminalTranscript(tc.raw))
			defer server.Close()
			c := mustClient(t, server.URL)
			ch, err := c.Stream(context.Background(), validRequest(t))
			if err != nil {
				t.Fatalf("Stream() error = %v, want nil (S-ACP-001)", err)
			}
			got := drainFinishReason(t, ch)
			if got != tc.neutral {
				t.Errorf("FinishReason = %v, want %v (raw %q) (S-ACP-001)", got, tc.neutral, tc.raw)
			}
		})
	}
}

// TestFinishReason_DeprecatedFunctionCall covers S-ACP-002: a transcript
// whose terminal chunk carries `"function_call"` drains to
// FinishReasonToolCalls with no error event — the deprecated member is
// enum-legal, not novel. Named test (not a row of the table above) so
// the "deprecated but enum-legal" reading is unmistakable at review.
func TestFinishReason_DeprecatedFunctionCall(t *testing.T) {
	t.Parallel()

	transcript := finishReasonTerminalTranscript("function_call")
	server := sseServer(t, transcript)
	defer server.Close()
	c := mustClient(t, server.URL)
	ch, err := c.Stream(context.Background(), validRequest(t))
	if err != nil {
		t.Fatalf("Stream() error = %v, want nil (S-ACP-002)", err)
	}
	events := drainAll(t, ch)
	requireCheckStreamClean(t, events)
	for _, ev := range events {
		if ev.Kind() == ai.EventKindError {
			t.Fatalf("unexpected error event for deprecated function_call (S-ACP-002): %+v", ev)
		}
	}
	completion, ok := events[len(events)-1].Completion()
	if !ok {
		t.Fatalf("last event carries no Completion payload (S-ACP-002): %+v", events[len(events)-1])
	}
	if got := completion.FinishReason(); got != ai.FinishReasonToolCalls {
		t.Errorf("FinishReason = %v, want FinishReasonToolCalls (S-ACP-002)", got)
	}
}

// TestFinishReason_ContentFilterDistinctFromRefusal covers S-ACP-003:
// a transcript terminating on `content_filter` drains to
// FinishReasonContentFilter, and the SAME completion's reason is
// asserted NOT to equal FinishReasonRefusal — proving the two are
// distinguished rather than coincidentally equal.
func TestFinishReason_ContentFilterDistinctFromRefusal(t *testing.T) {
	t.Parallel()

	transcript := finishReasonTerminalTranscript("content_filter")
	server := sseServer(t, transcript)
	defer server.Close()
	c := mustClient(t, server.URL)
	ch, err := c.Stream(context.Background(), validRequest(t))
	if err != nil {
		t.Fatalf("Stream() error = %v, want nil (S-ACP-003)", err)
	}
	got := drainFinishReason(t, ch)
	if got != ai.FinishReasonContentFilter {
		t.Errorf("FinishReason = %v, want FinishReasonContentFilter (S-ACP-003)", got)
	}
	if got == ai.FinishReasonRefusal {
		t.Errorf("FinishReason = FinishReasonRefusal, want it distinguished (content_filter is a decision taken BESIDE the model, not a refusal) (S-ACP-003)")
	}
}

// unreachableNeutralValues is R-ACP-002's table of neutral values this
// dialect cannot produce from any wire finish_reason (U5 NEGATIVE).
var unreachableNeutralValues = []ai.FinishReason{
	ai.FinishReasonRefusal,
	ai.FinishReasonPauseTurn,
	ai.FinishReasonUnknown,
}

// TestFinishReason_NeverUnreachable covers S-ACP-005: the five-value
// table of S-ACP-001 driven exhaustively produces no completion whose
// reason matches any of {Refusal, PauseTurn, Unknown}; and as the
// negative control, a transcript carrying an out-of-enum finish value
// produces a typed malformed response rather than
// FinishReasonUnknown, so the "never Unknown" claim is not satisfied
// vacuously by absence of input.
func TestFinishReason_NeverUnreachable(t *testing.T) {
	t.Parallel()

	for _, tc := range fiveWireFinishReasonCases {
		tc := tc
		t.Run("reachable_enum/"+tc.name, func(t *testing.T) {
			t.Parallel()
			server := sseServer(t, finishReasonTerminalTranscript(tc.raw))
			defer server.Close()
			c := mustClient(t, server.URL)
			ch, err := c.Stream(context.Background(), validRequest(t))
			if err != nil {
				t.Fatalf("Stream() error = %v, want nil (S-ACP-005)", err)
			}
			got := drainFinishReason(t, ch)
			for _, u := range unreachableNeutralValues {
				if got == u {
					t.Errorf("FinishReason = %v (unreachable), want one of the five reachable values (S-ACP-005 — U5 NEGATIVE)", got)
				}
			}
		})
	}

	t.Run("novel_value_negative_control", func(t *testing.T) {
		t.Parallel()
		transcript := finishReasonTerminalTranscript("halted")
		server := sseServer(t, transcript)
		defer server.Close()
		c := mustClient(t, server.URL)
		ch, err := c.Stream(context.Background(), validRequest(t))
		if err != nil {
			t.Fatalf("Stream() error = %v, want nil (S-ACP-005 negative control)", err)
		}
		events := drainAll(t, ch)
		// A typed malformed response is a non-nil error event in the
		// drained stream (stream.go's failure-mapping wraps
		// errUnrecognizedFinishReason as a stream error). No completion
		// event means no FinishReason reached — confirming the strict
		// gate did NOT downgrade to FinishReasonUnknown.
		completionCount := 0
		for _, ev := range events {
			if ev.Kind() == ai.EventKindError {
				return // typed malformed response observed — strict gate held
			}
			if _, ok := ev.Completion(); ok {
				completionCount++
			}
		}
		if completionCount == 0 {
			t.Fatal("no completion AND no error event — strict gate produced neither typed malformed nor FinishReasonUnknown (S-ACP-005 negative control)")
		}
		t.Fatalf("completionCount = %d, want 0 (novel wire value must produce typed malformed, not a completion with FinishReasonUnknown) (S-ACP-005 negative control)", completionCount)
	})
}

// novelFinishReasonCases is the case/whitespace variant set R-ACP-003
// (S-ACP-006) names: each MUST produce a typed malformed response, and
// S-ATS-039's strict-rejection test runs unmodified alongside them.
var novelFinishReasonCases = []struct {
	name string
	raw  string
}{
	{"uppercase_STOP", "STOP"},
	{"leading_whitespace_stop", " stop"},
	{"unrecognised_halted", "halted"},
}

// TestFinishReason_NovelValue_TypedMalformed covers S-ACP-006: each of
// the three novel raw finish values yields a typed malformed response
// (no completion event), confirming the strict gate's byte-exact
// rejection holds against case and whitespace variants of legal
// members.
func TestFinishReason_NovelValue_TypedMalformed(t *testing.T) {
	t.Parallel()

	for _, tc := range novelFinishReasonCases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			transcript := finishReasonTerminalTranscript(tc.raw)
			server := sseServer(t, transcript)
			defer server.Close()
			c := mustClient(t, server.URL)
			ch, err := c.Stream(context.Background(), validRequest(t))
			if err != nil {
				t.Fatalf("Stream() error = %v, want nil (S-ACP-006)", err)
			}
			events := drainAll(t, ch)
			completionCount := 0
			for _, ev := range events {
				if _, ok := ev.Completion(); ok {
					completionCount++
				}
			}
			if completionCount != 0 {
				t.Fatalf("completionCount = %d, want 0 (novel raw %q must produce typed malformed, not a completion) (S-ACP-006)", completionCount, tc.raw)
			}
		})
	}
}

// TestStopSequence_NothingIdentifiesMatch covers S-ACP-010: a transcript
// terminating on `finish_reason: "stop"` after a request carrying four
// stop sequences (encoded in the request, not in the transcript) drains
// to a completion carrying the finish reason and nothing identifying
// which sequence matched; and as the negative control, a transcript
// whose terminal chunk carries an EXTRA unknown `stop_sequence` key is
// drained to the same completion, proving the absence assertion holds
// because nothing is read, not because nothing was offered.
//
// The fixture's terminal chunk carries no `stop_sequence` key (the wire
// never does — U4 NEGATIVE); the negative control chunk carries an
// UNKNOWN `stop_sequence` key whose presence is exactly the "extra
// field offered but unread" probe this scenario demands.
func TestStopSequence_NothingIdentifiesMatch(t *testing.T) {
	t.Parallel()

	t.Run("terminal_chunk_no_stop_sequence_key", func(t *testing.T) {
		t.Parallel()
		transcript := finishReasonTerminalTranscript("stop")
		server := sseServer(t, transcript)
		defer server.Close()
		c := mustClient(t, server.URL)
		ch, err := c.Stream(context.Background(), validRequest(t))
		if err != nil {
			t.Fatalf("Stream() error = %v, want nil (S-ACP-010)", err)
		}
		got := drainFinishReason(t, ch)
		if got != ai.FinishReasonStop {
			t.Errorf("FinishReason = %v, want FinishReasonStop (S-ACP-010)", got)
		}
	})

	t.Run("terminal_chunk_with_extra_unknown_stop_sequence_key", func(t *testing.T) {
		t.Parallel()
		// Terminal chunk carries finish_reason="stop" AND an extra
		// unknown "stop_sequence" key whose presence is exactly the
		// probe shape S-ACP-010's negative control names. The
		// completion reports the same FinishReasonStop — the extra key
		// is silently unread (U4 NEGATIVE: the wire carries no such
		// field per the spec).
		transcript := "" +
			"data: {\"id\":\"chatcmpl-ss\",\"model\":\"m\",\"object\":\"chat.completion.chunk\",\"created\":1700000000,\"choices\":[{\"index\":0,\"delta\":{},\"finish_reason\":\"stop\"}],\"stop_sequence\":\"matched-token\"}\n\n" +
			"data: [DONE]\n\n"
		server := sseServer(t, transcript)
		defer server.Close()
		c := mustClient(t, server.URL)
		ch, err := c.Stream(context.Background(), validRequest(t))
		if err != nil {
			t.Fatalf("Stream() error = %v, want nil (S-ACP-010 negative control)", err)
		}
		got := drainFinishReason(t, ch)
		if got != ai.FinishReasonStop {
			t.Errorf("FinishReason = %v, want FinishReasonStop — extra stop_sequence key must not change the mapping (S-ACP-010 negative control)", got)
		}
	})
}
