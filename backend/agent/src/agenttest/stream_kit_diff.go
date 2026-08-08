// AI-22.2 — readable event diffs: a failed comparison names the first
// divergence precisely instead of dumping two recordings and asking the
// reader to spot the difference (design.md D2).

package agenttest

import (
	"fmt"
	"reflect"
	"testing"

	"github.com/cachicamas/backend/agent/src/ai"
)

// RequireSameEvents fails tb naming the FIRST divergence between got and
// want — the lowest differing index, both sides' kind and sequence — never
// "not equal", never an arbitrary or last divergence, and never a full dump
// of both slices (R-STK-003). Each side's rendering is a bounded payload
// summary, never the payload verbatim (R-STK-004).
//
// Equality is checked per index with [reflect.DeepEqual], never by
// comparing rendered summaries: a summary is deliberately lossy (bounded,
// design.md D2), so comparing by summary would let two genuinely different
// events read as equal. When the recordings differ only in length, the
// report names the index at which the shorter one ended and every kind the
// longer one still holds past that point.
func RequireSameEvents(tb testing.TB, got, want []ai.Event) {
	tb.Helper()

	shorter := min(len(got), len(want))
	for i := range shorter {
		if !reflect.DeepEqual(got[i], want[i]) {
			reportDivergence(tb, i,
				fmt.Sprintf("%s (seq=%d)", summarize(got[i]), got[i].Sequence()),
				fmt.Sprintf("%s (seq=%d)", summarize(want[i]), want[i].Sequence()))
			return
		}
	}

	if len(got) == len(want) {
		return
	}
	if len(got) < len(want) {
		reportDivergence(tb, shorter,
			fmt.Sprintf("ends there (%d event(s) total)", len(got)),
			fmt.Sprintf("still holds %v", kindsOf(want[shorter:])))
		return
	}
	reportDivergence(tb, shorter,
		fmt.Sprintf("still holds %v", kindsOf(got[shorter:])),
		fmt.Sprintf("ends there (%d event(s) total)", len(want)))
}

// reportDivergence is the one call site that turns a located divergence
// into tb's failure — the value-mismatch path and the length-mismatch path
// both funnel through it rather than each formatting their own near-
// duplicate message.
func reportDivergence(tb testing.TB, index int, got, want string) {
	tb.Helper()
	tb.Fatalf("agenttest: RequireSameEvents: first divergence at index %d: got %s, want %s (R-STK-003)", index, got, want)
}

// kindsOf renders a slice of events as their kinds, in order — the
// length-mismatch report's way of naming "what the longer side still
// holds" without dumping the events themselves.
func kindsOf(events []ai.Event) []string {
	out := make([]string, len(events))
	for i, ev := range events {
		out[i] = ev.Kind().String()
	}
	return out
}

// summaryRuneCap bounds every free-form payload fragment this file renders
// — a delta, a tool-call argument fragment, a reasoning token — to at most
// this many runes, with an elision marker past it (R-STK-004). It is never
// applied to a structural field (a block index, a finish reason, a tool
// name or id): those render whole, because they are already bounded by
// their own construction rules and truncating them would hide the actual
// divergence rather than just its byte volume.
const summaryRuneCap = 32

// summaryTable renders one event of each registered kind into a bounded,
// kind-naming diagnostic string (R-STK-004). It is keyed by every member
// [ai.EventKinds] enumerates — TestRequireSameEvents_EveryRegisteredKind_
// UsesASpecificSummaryNotTheGenericFallback (stream_kit_diff_test.go) is
// this table's own exhaustiveness proof, from outside the package: it
// fails, naming the kind, the moment a kind's entry stops rendering
// anything beyond the generic, payload-free event(kind seq=N) fallback
// [summarize] falls back to for anything not in this table.
var summaryTable = map[ai.EventKind]func(ai.Event) string{
	ai.EventKindResponseStart: func(ev ai.Event) string {
		p, _ := ev.ResponseStart()
		// AI-36 (WU-4, S-STK-049): id/model are caller/provider-supplied
		// strings with no length bound of their own — TestRequireSameEvents_BoundHoldsForEveryEventKind
		// proved an over-bound value here rendered whole (%q) before this
		// fix. boundedFragment is now applied to both, matching every
		// other free-form field in this table.
		return fmt.Sprintf("responsestart(id=%s model=%s)", boundedFragment(p.ResponseID()), boundedFragment(p.ServedModel()))
	},
	ai.EventKindCompletion: func(ev ai.Event) string {
		p, _ := ev.Completion()
		return fmt.Sprintf("completion(reason=%v usage=%+v)", p.FinishReason(), p.Usage())
	},
	ai.EventKindReasoningBlockStart: func(ev ai.Event) string {
		p, _ := ev.ReasoningBlockStart()
		return fmt.Sprintf("reasoningblockstart(block=%d redacted=%v)", p.BlockIndex(), p.Redacted())
	},
	ai.EventKindReasoningDelta: func(ev ai.Event) string {
		p, _ := ev.ReasoningDelta()
		return fmt.Sprintf("reasoningdelta(block=%d fragment=%s)", p.BlockIndex(), boundedFragment(string(p.Fragment())))
	},
	ai.EventKindReasoningBlockEnd: func(ev ai.Event) string {
		p, _ := ev.ReasoningBlockEnd()
		token, ok := p.Token()
		if !ok {
			return fmt.Sprintf("reasoningblockend(block=%d token=none)", p.BlockIndex())
		}
		return fmt.Sprintf("reasoningblockend(block=%d token=%s)", p.BlockIndex(), boundedFragment(string(token)))
	},
	ai.EventKindTextBlockStart: func(ev ai.Event) string {
		p, _ := ev.TextBlockStart()
		return fmt.Sprintf("text_block_start(block=%d)", p.Block())
	},
	ai.EventKindTextDelta: func(ev ai.Event) string {
		p, _ := ev.TextDelta()
		return fmt.Sprintf("text_delta(block=%d delta=%s)", p.Block(), boundedFragment(p.Delta()))
	},
	ai.EventKindTextBlockEnd: func(ev ai.Event) string {
		p, _ := ev.TextBlockEnd()
		return fmt.Sprintf("text_block_end(block=%d)", p.Block())
	},
	ai.EventKindToolCallStart: func(ev ai.Event) string {
		p, _ := ev.ToolCallStart()
		// AI-36 (WU-4, S-STK-049): id/name are caller/provider-supplied
		// strings with no length bound of their own — same fix and same
		// proof as EventKindResponseStart above.
		return fmt.Sprintf("tool_call_start(block=%d id=%s name=%s)", p.BlockIndex(), boundedFragment(p.ID()), boundedFragment(p.Name()))
	},
	ai.EventKindToolCallDelta: func(ev ai.Event) string {
		p, _ := ev.ToolCallDelta()
		return fmt.Sprintf("tool_call_delta(block=%d fragment=%s)", p.BlockIndex(), boundedFragment(string(p.Fragment())))
	},
	ai.EventKindToolCallEnd: func(ev ai.Event) string {
		p, _ := ev.ToolCallEnd()
		return fmt.Sprintf("tool_call_end(block=%d arguments=%s)", p.BlockIndex(), boundedFragment(string(p.Arguments())))
	},
	ai.EventKindError: func(ev ai.Event) string {
		p, ok := ev.ErrorPayload()
		if !ok {
			return "error(<no payload>)"
		}
		return fmt.Sprintf("error(category=%v label=%s)", p.Category(), boundedFragment(p.RawLabel()))
	},
}

// summarize renders ev through its kind's registered summaryTable entry, or
// falls back to its bare [ai.EventKind.String] (never blank, even for the
// zero kind, which renders "unset") for a kind this table does not
// register — the zero kind included, which is never a member of
// [ai.EventKinds] and so never reaches this table at all. The caller
// appends the sequence itself (R-STK-003), so this fallback stays
// payload-free rather than duplicating it.
func summarize(ev ai.Event) string {
	if render, ok := summaryTable[ev.Kind()]; ok {
		return render(ev)
	}
	return ev.Kind().String()
}

// boundedFragment renders a free-form byte fragment as its true length plus
// a head capped at summaryRuneCap runes, with a trailing elision marker
// when truncated (R-STK-004). The true length is always the untruncated
// byte count, so a reader can see how much was elided even though the head
// itself never grows past the cap.
func boundedFragment(s string) string {
	runes := []rune(s)
	head := s
	if len(runes) > summaryRuneCap {
		head = string(runes[:summaryRuneCap]) + "…"
	}
	return fmt.Sprintf("len=%d head=%q", len(s), head)
}
