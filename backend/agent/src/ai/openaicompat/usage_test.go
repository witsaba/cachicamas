// AI-28.3 — absent-versus-zero fidelity: R-ATS-015 (usage fields the
// transcript never carried read back absent, not zero, S-ATS-055…058) and
// R-ATS-016 (the usage chunk's presence is asserted, never silently
// accepted as empty, S-ATS-059…062). Same httptest.Server-driven seam as
// stream_text_test.go: this is genuinely new mapping behavior — before
// this slice, buildCompletion() always returned ai.Usage{} unconditionally
// regardless of the transcript, so every test below is a real RED against
// the unmodified slice-1/2/3 code (confirmed by running before writing any
// production code — see this milestone's apply-progress record).

package openaicompat

import (
	"context"
	"testing"

	"github.com/cachicamas/backend/agent/src/ai"
)

// usageTranscript builds a minimal transcript: one identity-establishing
// terminal chunk, then usageChunk verbatim (already a full "data:
// ...\n\n" SSE frame, or "" to omit it entirely), then the sentinel.
func usageTranscript(usageChunk string) string {
	return "" +
		"data: {\"id\":\"chatcmpl-u\",\"model\":\"m\",\"choices\":[{\"index\":0,\"delta\":{},\"finish_reason\":\"stop\"}]}\n\n" +
		usageChunk +
		"data: [DONE]\n\n"
}

// drainLastCompletion drains ch and returns the last event's Completion
// payload, failing the test if that event does not carry one.
func drainLastCompletion(t *testing.T, ch <-chan ai.Event) ai.Completion {
	t.Helper()
	events := drainAll(t, ch)
	if len(events) == 0 {
		t.Fatal("drained zero events, want a completion")
	}
	completion, ok := events[len(events)-1].Completion()
	if !ok {
		t.Fatalf("last event carries no Completion payload: %+v", events[len(events)-1])
	}
	return completion
}

// ---------------------------------------------------------------------
// R-ATS-015 — usage fields the transcript never carried read back absent,
// not zero (S-ATS-055…058).
// ---------------------------------------------------------------------

// TestUsage_OnlyPromptAndCompletionTokens_UnmappedFieldsReadAbsent covers
// S-ATS-055: a usage chunk carrying only prompt_tokens and
// completion_tokens (both present) leaves the fields this milestone never
// maps — CacheRead, CacheWrite, Reasoning — reporting absent.
func TestUsage_OnlyPromptAndCompletionTokens_UnmappedFieldsReadAbsent(t *testing.T) {
	t.Parallel()

	transcript := usageTranscript("data: {\"id\":\"chatcmpl-u\",\"model\":\"m\",\"choices\":[],\"usage\":{\"prompt_tokens\":11,\"completion_tokens\":22,\"total_tokens\":33}}\n\n")
	server := sseServer(t, transcript)
	defer server.Close()
	c := mustClient(t, server.URL)
	ch, err := c.Stream(context.Background(), validRequest(t))
	if err != nil {
		t.Fatalf("Stream() error = %v, want nil (S-ATS-055)", err)
	}
	usage := drainLastCompletion(t, ch).Usage()

	if v, ok := usage.Input.Count(); !ok || v != 11 {
		t.Errorf("Input.Count() = (%d, %v), want (11, true) (S-ATS-055)", v, ok)
	}
	if v, ok := usage.Output.Count(); !ok || v != 22 {
		t.Errorf("Output.Count() = (%d, %v), want (22, true) (S-ATS-055)", v, ok)
	}
	if _, ok := usage.CacheRead.Count(); ok {
		t.Error("CacheRead.Count() ok = true, want false — this milestone never maps it (S-ATS-055)")
	}
	if _, ok := usage.CacheWrite.Count(); ok {
		t.Error("CacheWrite.Count() ok = true, want false — this milestone never maps it (S-ATS-055)")
	}
	if _, ok := usage.Reasoning.Count(); ok {
		t.Error("Reasoning.Count() ok = true, want false — this milestone never maps it (S-ATS-055)")
	}
}

// TestUsage_ExplicitZero_ReportsReportedZeroNotAbsent covers S-ATS-056: an
// explicit 0 for prompt_tokens reports (0, true) — a reported zero,
// distinguishable from the absent case.
func TestUsage_ExplicitZero_ReportsReportedZeroNotAbsent(t *testing.T) {
	t.Parallel()

	transcript := usageTranscript("data: {\"id\":\"chatcmpl-u\",\"model\":\"m\",\"choices\":[],\"usage\":{\"prompt_tokens\":0,\"completion_tokens\":5,\"total_tokens\":5}}\n\n")
	server := sseServer(t, transcript)
	defer server.Close()
	c := mustClient(t, server.URL)
	ch, err := c.Stream(context.Background(), validRequest(t))
	if err != nil {
		t.Fatalf("Stream() error = %v, want nil (S-ATS-056)", err)
	}
	usage := drainLastCompletion(t, ch).Usage()

	v, ok := usage.Input.Count()
	if !ok {
		t.Fatal("Input.Count() ok = false, want true — an explicit 0 is a reported zero, not absent (S-ATS-056)")
	}
	if v != 0 {
		t.Errorf("Input.Count() value = %d, want 0 (S-ATS-056)", v)
	}
}

// TestUsage_OmittedVsExplicitZero_UsageRecordsNotEqual covers S-ATS-057: two
// transcripts identical except one omits prompt_tokens and the other
// reports it as 0 produce usage records that are NOT equal.
func TestUsage_OmittedVsExplicitZero_UsageRecordsNotEqual(t *testing.T) {
	t.Parallel()

	omitted := usageTranscript("data: {\"id\":\"chatcmpl-u\",\"model\":\"m\",\"choices\":[],\"usage\":{\"completion_tokens\":5,\"total_tokens\":5}}\n\n")
	explicitZero := usageTranscript("data: {\"id\":\"chatcmpl-u\",\"model\":\"m\",\"choices\":[],\"usage\":{\"prompt_tokens\":0,\"completion_tokens\":5,\"total_tokens\":5}}\n\n")

	drain := func(transcript string) ai.Usage {
		server := sseServer(t, transcript)
		defer server.Close()
		c := mustClient(t, server.URL)
		ch, err := c.Stream(context.Background(), validRequest(t))
		if err != nil {
			t.Fatalf("Stream() error = %v, want nil (S-ATS-057)", err)
		}
		return drainLastCompletion(t, ch).Usage()
	}

	usageOmitted := drain(omitted)
	usageZero := drain(explicitZero)

	if usageOmitted == usageZero {
		t.Errorf("usage records are equal (%+v), want NOT equal — an omitted field must be distinguishable from an explicit 0 (S-ATS-057)", usageOmitted)
	}
	if _, ok := usageOmitted.Input.Count(); ok {
		t.Error("omitted-field usage reports Input present, want absent (S-ATS-057 premise)")
	}
	if v, ok := usageZero.Input.Count(); !ok || v != 0 {
		t.Errorf("explicit-zero usage Input.Count() = (%d, %v), want (0, true) (S-ATS-057 premise)", v, ok)
	}
}

// TestUsage_NullOnNonFinalChunks_DoesNotOverwritePopulatedUsage covers
// S-ATS-058: content chunks carrying usage:null (C4), followed by a
// populated usage chunk, leave the completion's usage carrying the final
// chunk's reported counts — no field reset by the preceding nulls.
func TestUsage_NullOnNonFinalChunks_DoesNotOverwritePopulatedUsage(t *testing.T) {
	t.Parallel()

	transcript := "" +
		"data: {\"id\":\"chatcmpl-u\",\"model\":\"m\",\"choices\":[{\"index\":0,\"delta\":{\"role\":\"assistant\",\"content\":\"hi\"},\"finish_reason\":null}],\"usage\":null}\n\n" +
		"data: {\"id\":\"chatcmpl-u\",\"model\":\"m\",\"choices\":[{\"index\":0,\"delta\":{},\"finish_reason\":\"stop\"}],\"usage\":null}\n\n" +
		"data: {\"id\":\"chatcmpl-u\",\"model\":\"m\",\"choices\":[],\"usage\":{\"prompt_tokens\":7,\"completion_tokens\":9,\"total_tokens\":16}}\n\n" +
		"data: [DONE]\n\n"
	server := sseServer(t, transcript)
	defer server.Close()
	c := mustClient(t, server.URL)
	ch, err := c.Stream(context.Background(), validRequest(t))
	if err != nil {
		t.Fatalf("Stream() error = %v, want nil (S-ATS-058)", err)
	}
	usage := drainLastCompletion(t, ch).Usage()

	if v, ok := usage.Input.Count(); !ok || v != 7 {
		t.Errorf("Input.Count() = (%d, %v), want (7, true) — preceding usage:null must not reset it (S-ATS-058)", v, ok)
	}
	if v, ok := usage.Output.Count(); !ok || v != 9 {
		t.Errorf("Output.Count() = (%d, %v), want (9, true) (S-ATS-058)", v, ok)
	}
}

// ---------------------------------------------------------------------
// R-ATS-016 — the usage chunk's presence is asserted, never silently
// accepted as empty (S-ATS-059…062).
// ---------------------------------------------------------------------

// usagePresent reports whether u carries at least one present count — the
// "same assertion" S-ATS-059 requires to hold positively and S-ATS-060
// requires to fail when the usage chunk is removed.
func usagePresent(u ai.Usage) bool {
	_, ok := u.Input.Count()
	return ok
}

// TestUsage_PresenceAssertedPositively_AndNegativeControlFails covers
// S-ATS-059 and S-ATS-060 together: the SAME presence assertion holds when
// the usage chunk arrives and fails when it is deliberately removed —
// proving the assertion actually distinguishes a populated usage from a
// missing one, rather than passing vacuously either way.
func TestUsage_PresenceAssertedPositively_AndNegativeControlFails(t *testing.T) {
	t.Parallel()

	withUsage := usageTranscript("data: {\"id\":\"chatcmpl-u\",\"model\":\"m\",\"choices\":[],\"usage\":{\"prompt_tokens\":4,\"completion_tokens\":6,\"total_tokens\":10}}\n\n")
	withoutUsage := usageTranscript("")

	t.Run("usage chunk present: assertion holds (S-ATS-059)", func(t *testing.T) {
		t.Parallel()
		server := sseServer(t, withUsage)
		defer server.Close()
		c := mustClient(t, server.URL)
		ch, err := c.Stream(context.Background(), validRequest(t))
		if err != nil {
			t.Fatalf("Stream() error = %v, want nil", err)
		}
		usage := drainLastCompletion(t, ch).Usage()
		if !usagePresent(usage) {
			t.Fatal("usagePresent() = false, want true — the usage chunk arrived (S-ATS-059)")
		}
	})

	t.Run("usage chunk removed: same assertion fails (S-ATS-060)", func(t *testing.T) {
		t.Parallel()
		server := sseServer(t, withoutUsage)
		defer server.Close()
		c := mustClient(t, server.URL)
		ch, err := c.Stream(context.Background(), validRequest(t))
		if err != nil {
			t.Fatalf("Stream() error = %v, want nil", err)
		}
		usage := drainLastCompletion(t, ch).Usage()
		if usagePresent(usage) {
			t.Fatal("usagePresent() = true even though the usage chunk was removed — S-ATS-059's own check would have passed vacuously, proving nothing (S-ATS-060)")
		}
	})
}

// TestUsage_EmptyChoicesArray_NoTextEventNoProtocolViolation covers
// S-ATS-061: the usage chunk's empty choices array derives no text event,
// no extra response-start, and no protocol-order violation.
func TestUsage_EmptyChoicesArray_NoTextEventNoProtocolViolation(t *testing.T) {
	t.Parallel()

	transcript := usageTranscript("data: {\"id\":\"chatcmpl-u\",\"model\":\"m\",\"choices\":[],\"usage\":{\"prompt_tokens\":1,\"completion_tokens\":2,\"total_tokens\":3}}\n\n")
	server := sseServer(t, transcript)
	defer server.Close()
	c := mustClient(t, server.URL)
	ch, err := c.Stream(context.Background(), validRequest(t))
	if err != nil {
		t.Fatalf("Stream() error = %v, want nil (S-ATS-061)", err)
	}
	events := drainAll(t, ch)

	textCount, responseStartCount := 0, 0
	for _, ev := range events {
		switch ev.Kind() {
		case ai.EventKindTextDelta:
			textCount++
		case ai.EventKindResponseStart:
			responseStartCount++
		}
	}
	if textCount != 0 {
		t.Errorf("text delta count = %d, want 0 (S-ATS-061)", textCount)
	}
	if responseStartCount != 1 {
		t.Errorf("response-start count = %d, want exactly 1 (S-ATS-061)", responseStartCount)
	}
	if report := ai.CheckStream(events); report.Violation() != nil {
		t.Errorf("CheckStream violation = %v, want nil (S-ATS-061)", report.Violation())
	}
}

// TestUsage_NoUsageChunkAtAll_CompletesWithWhollyAbsentUsage covers
// S-ATS-062: a stream that terminates cleanly with the sentinel but
// carries no usage chunk at all still completes, with no failure, and
// every usage field reporting absent.
func TestUsage_NoUsageChunkAtAll_CompletesWithWhollyAbsentUsage(t *testing.T) {
	t.Parallel()

	transcript := usageTranscript("")
	server := sseServer(t, transcript)
	defer server.Close()
	c := mustClient(t, server.URL)
	ch, err := c.Stream(context.Background(), validRequest(t))
	if err != nil {
		t.Fatalf("Stream() error = %v, want nil (S-ATS-062)", err)
	}
	events := drainAll(t, ch)
	if len(events) == 0 {
		t.Fatal("drained zero events (S-ATS-062)")
	}
	for _, ev := range events {
		if ev.Kind() == ai.EventKindError {
			t.Fatalf("unexpected error event, want a clean completion (S-ATS-062): %+v", ev)
		}
	}
	completion, ok := events[len(events)-1].Completion()
	if !ok {
		t.Fatalf("last event carries no Completion payload (S-ATS-062): %+v", events[len(events)-1])
	}
	usage := completion.Usage()
	if _, ok := usage.Input.Count(); ok {
		t.Error("Input.Count() ok = true, want false (S-ATS-062)")
	}
	if _, ok := usage.Output.Count(); ok {
		t.Error("Output.Count() ok = true, want false (S-ATS-062)")
	}
	if _, ok := usage.CacheRead.Count(); ok {
		t.Error("CacheRead.Count() ok = true, want false (S-ATS-062)")
	}
	if _, ok := usage.CacheWrite.Count(); ok {
		t.Error("CacheWrite.Count() ok = true, want false (S-ATS-062)")
	}
	if _, ok := usage.Reasoning.Count(); ok {
		t.Error("Reasoning.Count() ok = true, want false (S-ATS-062)")
	}
}
