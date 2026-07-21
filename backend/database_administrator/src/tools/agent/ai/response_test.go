// Package-internal test for the sealed ResponseStartPayload / ResponseCompletePayload
// payloads (AI-12). The test file lives in package ai_test so it cannot accidentally
// reach the unexported aiPayload() marker — that un-reachability is the point of
// the TestResponsePayloads_InterfaceIsSealed checks below. AI-12 follows the same
// external-test convention as request_test.go and usage_test.go; AI-11's
// event_test.go deliberately uses package ai because it needs to implement the
// eventPayload interface for negative tests, but AI-12 only checks the seal and
// does not need a sibling implementation.
package ai_test

import (
	"errors"
	"os"
	"reflect"
	"strings"
	"testing"
	"unsafe"

	"github.com/cachicamas/backend/database_administrator/src/tools/agent/ai"
)

// MaxResponseIDLengthTest mirrors the production MaxResponseIDLength constant
// (256 bytes) so boundary tests can use it without leaking the production
// symbol into this test surface. Mirrors the AI-10 spec §"Requirement: Numeric
// validation" boundary-test pattern.
const MaxResponseIDLengthTest = 256

// responseIDOf builds a string of the requested byte length using ASCII filler.
// At 256 the value is exactly the inclusive boundary MaxResponseIDLengthTest.
// At 257 the value is one byte over the boundary and MUST trip the length check.
// The exact bytes do not matter — only that the resulting len(...) is the
// requested count — because the sentinel is on length, not content.
func responseIDOf(n int) string {
	if n <= 0 {
		return ""
	}
	b := make([]byte, n)
	for i := range b {
		b[i] = 'a'
	}
	return string(b)
}

// usageWithCounts builds a Usage with the requested required counts and
// no optional details. For valid counts it delegates to the public
// NewUsage; for invalid counts it bypasses validation via
// rawUsageWithCounts so the matrix can drive NewResponseCompletePayload
// with a Usage whose Validate() fails. Production code never sees a
// Usage with negative counts because every construction goes through
// NewUsage. Test affordance only.
func usageWithCounts(input, output, total int64) ai.Usage {
	u, err := ai.NewUsage(input, output, total, nil, nil, nil)
	if err == nil {
		return u
	}
	return rawUsageWithCounts(input, output, total)
}

// rawUsageWithCounts constructs a Usage with the requested required
// counts, bypassing ai.Usage.Validate. The standard reflect-with-unsafe
// trick cannot set unexported fields (reflect.Value.SetInt enforces the
// export flag), so the well-known memory-layout cast is used: declare
// a local struct with field-for-field identical layout to ai.Usage,
// fill it in, then reinterpret the bytes via unsafe.Pointer. Layout
// MUST match usage.go's Usage declaration order; if Usage ever gains,
// removes, or reorders a field, this helper MUST be updated to match
// or it will silently construct a corrupted Usage. The returned Usage
// is local to the test (every production path goes through NewUsage).
// Standard Go testing pattern (protobuf, gRPC, sealed-value suites).
func rawUsageWithCounts(input, output, total int64) ai.Usage {
	type usageLayout struct {
		inputTokens      int64
		outputTokens     int64
		totalTokens      int64
		cacheReadTokens  *int64
		cacheWriteTokens *int64
		reasoningTokens  *int64
	}
	raw := usageLayout{
		inputTokens:  input,
		outputTokens: output,
		totalTokens:  total,
	}
	return *(*ai.Usage)(unsafe.Pointer(&raw))
}

// ---------------------------------------------------------------------------
// T-AI12-001 — Constructor validation matrix (SCN-001, SCN-002)
// ---------------------------------------------------------------------------

// TestNewResponseStartPayload_ConstructorValidationMatrix is the combined
// boundary + first-failure-order matrix for ResponseID × Model. Each row
// drives the same constructor with one invalid input (or no invalid input
// for happy paths) and asserts the (payload, error) pair. Per SCN-001.
func TestNewResponseStartPayload_ConstructorValidationMatrix(t *testing.T) {
	tooLong := responseIDOf(MaxResponseIDLengthTest + 1)
	maxLen := responseIDOf(MaxResponseIDLengthTest)

	cases := []struct {
		name        string
		responseID  string
		model       string
		wantErr     error // sentinel or wrapping sentinel; nil → success
		wantSuccess bool
	}{
		// Happy paths.
		{name: "minimal ASCII", responseID: "r1", model: "gpt-x", wantSuccess: true},
		{name: "non-trivial ASCII", responseID: "resp-abc-123", model: "claude-opus", wantSuccess: true},
		{name: "256-byte boundary accepted", responseID: maxLen, model: "m", wantSuccess: true},
		// ResponseID validation order: empty → whitespace → length.
		{name: "empty response id", responseID: "", model: "gpt-x", wantErr: ai.ErrEmptyResponseID},
		{name: "whitespace response id", responseID: "   ", model: "gpt-x", wantErr: ai.ErrWhitespaceResponseID},
		{name: "whitespace-only tab/newline response id", responseID: "\t\n", model: "gpt-x", wantErr: ai.ErrWhitespaceResponseID},
		{name: "too-long response id", responseID: tooLong, model: "gpt-x", wantErr: ai.ErrResponseIDTooLong},
		// Model validation order (only after ResponseID passes). First-failure
		// pinning: empty ResponseID must fire before any model check.
		{name: "first failure: empty response id wins over empty model", responseID: "", model: "", wantErr: ai.ErrEmptyResponseID},
		{name: "empty model", responseID: "r1", model: "", wantErr: ai.ErrEmptyResponseModel},
		{name: "whitespace model", responseID: "r1", model: "   ", wantErr: ai.ErrWhitespaceResponseModel},
		{name: "too-long model", responseID: "r1", model: tooLong, wantErr: ai.ErrResponseModelTooLong},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			p, err := ai.NewResponseStartPayload(tc.responseID, tc.model)
			if tc.wantSuccess {
				if err != nil {
					t.Fatalf("NewResponseStartPayload(%q,%q) = %v, want nil",
						tc.responseID, tc.model, err)
				}
				if p.ResponseID() != tc.responseID {
					t.Errorf("ResponseID() = %q, want %q", p.ResponseID(), tc.responseID)
				}
				if p.Model() != tc.model {
					t.Errorf("Model() = %q, want %q", p.Model(), tc.model)
				}
				if got := p.Kind(); got != ai.EventKindResponseStart {
					t.Errorf("Kind() = %q, want %q", got, ai.EventKindResponseStart)
				}
				if err := p.Validate(); err != nil {
					t.Errorf("Validate() = %v, want nil", err)
				}
				return
			}
			if err == nil {
				t.Fatalf("NewResponseStartPayload(%q,%q) = no error, want %v",
					tc.responseID, tc.model, tc.wantErr)
			}
			if !errors.Is(err, tc.wantErr) {
				t.Errorf("error = %v, want errors.Is(_, %v)", err, tc.wantErr)
			}
			// Sentinel value is zero — never a half-constructed payload.
			if p != (ai.ResponseStartPayload{}) {
				t.Errorf("payload = %+v, want zero on error", p)
			}
		})
	}
}

// TestNewResponseCompletePayload_ConstructorValidationMatrix covers the
// ResponseID + FinishReason + Usage validation chain. First-failure order is
// pinned: ResponseID → FinishReason → Usage. Per SCN-002.
func TestNewResponseCompletePayload_ConstructorValidationMatrix(t *testing.T) {
	tooLong := responseIDOf(MaxResponseIDLengthTest + 1)

	cases := []struct {
		name        string
		responseID  string
		finish      ai.FinishReason
		usage       ai.Usage
		wantErr     error
		wantSuccess bool
	}{
		// Happy paths.
		{name: "stop + valid usage", responseID: "r1", finish: ai.FinishReasonStop, usage: usageWithCounts(10, 20, 30), wantSuccess: true},
		{name: "length + zero usage (absent)", responseID: "r1", finish: ai.FinishReasonLength, usage: ai.Usage{}, wantSuccess: true},
		{name: "tool_call + zero usage", responseID: "r1", finish: ai.FinishReasonToolCall, usage: ai.Usage{}, wantSuccess: true},
		// ResponseID first.
		{name: "empty response id", responseID: "", finish: ai.FinishReasonStop, usage: ai.Usage{}, wantErr: ai.ErrEmptyResponseID},
		{name: "whitespace response id", responseID: "   ", finish: ai.FinishReasonStop, usage: ai.Usage{}, wantErr: ai.ErrWhitespaceResponseID},
		{name: "too-long response id", responseID: tooLong, finish: ai.FinishReasonStop, usage: ai.Usage{}, wantErr: ai.ErrResponseIDTooLong},
		// FinishReason next — even a bogus finish must wait for ResponseID to pass.
		{name: "invalid finish reason", responseID: "r1", finish: ai.FinishReason("not-a-real-reason"), usage: ai.Usage{}, wantErr: ai.ErrInvalidFinishReason},
		{name: "first failure: empty response id wins over invalid finish", responseID: "", finish: ai.FinishReason("bogus"), usage: ai.Usage{}, wantErr: ai.ErrEmptyResponseID},
		{name: "first failure: invalid finish wins over bad usage", responseID: "r1", finish: ai.FinishReason("bogus"), usage: usageWithCounts(-1, 0, 0), wantErr: ai.ErrInvalidFinishReason},
		// Usage last — Usage.Validate() error propagates.
		{name: "negative input usage", responseID: "r1", finish: ai.FinishReasonStop, usage: usageWithCounts(-1, 0, 0), wantErr: ai.ErrUsageNegativeInputTokens},
		{name: "negative output usage", responseID: "r1", finish: ai.FinishReasonStop, usage: usageWithCounts(0, -1, 0), wantErr: ai.ErrUsageNegativeOutputTokens},
		{name: "negative total usage", responseID: "r1", finish: ai.FinishReasonStop, usage: usageWithCounts(0, 0, -1), wantErr: ai.ErrUsageNegativeTotalTokens},
		{name: "total less than inputs", responseID: "r1", finish: ai.FinishReasonStop, usage: usageWithCounts(10, 20, 25), wantErr: ai.ErrUsageTotalLessThanInputs},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			p, err := ai.NewResponseCompletePayload(tc.responseID, tc.finish, tc.usage)
			if tc.wantSuccess {
				if err != nil {
					t.Fatalf("NewResponseCompletePayload(%q,%q,usage) = %v, want nil",
						tc.responseID, tc.finish, err)
				}
				if p.ResponseID() != tc.responseID {
					t.Errorf("ResponseID() = %q, want %q", p.ResponseID(), tc.responseID)
				}
				if p.FinishReason() != tc.finish {
					t.Errorf("FinishReason() = %q, want %q", p.FinishReason(), tc.finish)
				}
				if got := p.Usage(); got != tc.usage {
					t.Errorf("Usage() = %+v, want %+v", got, tc.usage)
				}
				if got := p.Kind(); got != ai.EventKindResponseComplete {
					t.Errorf("Kind() = %q, want %q", got, ai.EventKindResponseComplete)
				}
				if err := p.Validate(); err != nil {
					t.Errorf("Validate() = %v, want nil", err)
				}
				return
			}
			if err == nil {
				t.Fatalf("NewResponseCompletePayload(%q,%q,usage) = no error, want %v",
					tc.responseID, tc.finish, tc.wantErr)
			}
			if !errors.Is(err, tc.wantErr) {
				t.Errorf("error = %v, want errors.Is(_, %v)", err, tc.wantErr)
			}
			if p != (ai.ResponseCompletePayload{}) {
				t.Errorf("payload = %+v, want zero on error", p)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// T-AI12-002 — Sentinel distinctness + prefix + non-nil + non-empty (SCN-006)
// ---------------------------------------------------------------------------

// TestResponseSentinels_ArePairwiseDistinctAndPrefixed verifies the six
// AI-12 payload sentinels (three ResponseID + three Model) plus the reused
// AI-10 ErrInvalidFinishReason are all distinct via errors.Is, all start
// with "ai: ", all non-nil, all non-empty. Per SCN-006.
func TestResponseSentinels_ArePairwiseDistinctAndPrefixed(t *testing.T) {
	sentinels := map[string]error{
		"ErrEmptyResponseID":         ai.ErrEmptyResponseID,
		"ErrWhitespaceResponseID":    ai.ErrWhitespaceResponseID,
		"ErrResponseIDTooLong":       ai.ErrResponseIDTooLong,
		"ErrEmptyResponseModel":      ai.ErrEmptyResponseModel,
		"ErrWhitespaceResponseModel": ai.ErrWhitespaceResponseModel,
		"ErrResponseModelTooLong":    ai.ErrResponseModelTooLong,
		// Reused from AI-10 — must remain distinct from AI-12 sentinels.
		"ErrInvalidFinishReason": ai.ErrInvalidFinishReason,
	}
	if len(sentinels) != 7 {
		t.Fatalf("sentinels map has %d entries, want 7 (six AI-12 + one reused AI-10)",
			len(sentinels))
	}
	// Identity + prefix + non-empty.
	for name, e := range sentinels {
		if e == nil {
			t.Errorf("%s must not be nil", name)
			continue
		}
		if !errors.Is(e, e) {
			t.Errorf("%s must be errors.Is-compatible with itself", name)
		}
		if e.Error() == "" {
			t.Errorf("%s must have a non-empty message", name)
		}
		if !strings.HasPrefix(e.Error(), "ai: ") {
			t.Errorf("%s.Error() = %q, must start with %q (Phase B sentinel prefix)",
				name, e.Error(), "ai: ")
		}
	}
	// Pairwise distinct.
	for nameA, a := range sentinels {
		for nameB, b := range sentinels {
			if nameA == nameB {
				continue
			}
			if errors.Is(a, b) {
				t.Errorf("%s and %s must be distinct sentinels", nameA, nameB)
			}
		}
	}
}

// ---------------------------------------------------------------------------
// T-AI12-003 — Reflection: no Marshal/Clone/With + sealed marker parity (SCN-004)
// ---------------------------------------------------------------------------

// TestResponsePayloads_HaveNoMarshalOrCloneOrWithMethods verifies via
// reflection that neither payload type exposes MarshalJSON, UnmarshalJSON,
// MarshalText, UnmarshalText, Clone, or any method starting with "With".
// This is the AI-11 vendor-leak guard applied to AI-12 payloads. Per SCN-004.
func TestResponsePayloads_HaveNoMarshalOrCloneOrWithMethods(t *testing.T) {
	startPayload, err := ai.NewResponseStartPayload("r1", "gpt-x")
	if err != nil {
		t.Fatalf("setup NewResponseStartPayload = %v", err)
	}
	finishPayload, err := ai.NewResponseCompletePayload("r1", ai.FinishReasonStop, ai.Usage{})
	if err != nil {
		t.Fatalf("setup NewResponseCompletePayload = %v", err)
	}

	cases := []struct {
		name  string
		value any
	}{
		{"ResponseStartPayload", startPayload},
		{"ResponseCompletePayload", finishPayload},
	}
	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			typ := reflect.TypeOf(tc.value)
			forbidden := map[string]bool{
				"MarshalJSON":   true,
				"UnmarshalJSON": true,
				"MarshalText":   true,
				"UnmarshalText": true,
				"Clone":         true,
			}
			for i := 0; i < typ.NumMethod(); i++ {
				m := typ.Method(i)
				if forbidden[m.Name] {
					t.Errorf("%s must not define %s (AI-11 vendor-leak guard)",
						typ.Name(), m.Name)
				}
				if strings.HasPrefix(m.Name, "With") {
					t.Errorf("%s.%s must not be a With* method (sealed payload forbids builder-style mutation)",
						typ.Name(), m.Name)
				}
			}
		})
	}
}

// TestResponsePayloads_InterfaceIsSealed verifies the sealed eventPayload
// interface contract from outside the package: the unexported aiPayload()
// marker is unreachable from package ai_test, so the only way for
// ResponseStartPayload / ResponseCompletePayload to satisfy eventPayload
// is to be declared in package ai. The compile-time check that the test
// file even builds is the seal: if these types could be constructed
// outside package ai, the unexported method would have to be exported.
//
// Practically, this test asserts:
//   - The struct has a Kind() EventKind method (the second half of the
//     interface contract).
//   - The struct exposes no exported method whose name begins with
//     "AiPayload" — a structural mirror of "aiPayload cannot be reached
//     from outside the package".
//
// The seal itself (unexported method) is enforced by the Go type system
// at compile time and is implicit in this test file's ability to compile
// while remaining in package ai_test. Per SCN-004.
func TestResponsePayloads_InterfaceIsSealed(t *testing.T) {
	startPayload, err := ai.NewResponseStartPayload("r1", "gpt-x")
	if err != nil {
		t.Fatalf("setup NewResponseStartPayload = %v", err)
	}
	finishPayload, err := ai.NewResponseCompletePayload("r1", ai.FinishReasonStop, ai.Usage{})
	if err != nil {
		t.Fatalf("setup NewResponseCompletePayload = %v", err)
	}

	// The exposed surface must include Kind() EventKind for both payloads.
	for _, tc := range []struct {
		name  string
		value any
		kind  ai.EventKind
	}{
		{"ResponseStartPayload", startPayload, ai.EventKindResponseStart},
		{"ResponseCompletePayload", finishPayload, ai.EventKindResponseComplete},
	} {
		tc := tc
		t.Run(tc.name+"_Kind", func(t *testing.T) {
			v := reflect.ValueOf(tc.value)
			kindMethod := v.MethodByName("Kind")
			if !kindMethod.IsValid() {
				t.Fatalf("%s must expose Kind() method (sealed eventPayload contract)",
					tc.name)
			}
			results := kindMethod.Call(nil)
			if len(results) != 1 {
				t.Fatalf("%s.Kind() returned %d values, want 1", tc.name, len(results))
			}
			got, ok := results[0].Interface().(ai.EventKind)
			if !ok {
				t.Fatalf("%s.Kind() returned %T, want ai.EventKind", tc.name, results[0].Interface())
			}
			if got != tc.kind {
				t.Errorf("%s.Kind() = %q, want %q", tc.name, got, tc.kind)
			}
		})

		// The unexported aiPayload() marker cannot appear under any exported
		// alias. If the seal were broken, an external struct could declare a
		// method named "AiPayload" — we forbid that name too.
		t.Run(tc.name+"_NoAiPayloadAlias", func(t *testing.T) {
			typ := reflect.TypeOf(tc.value)
			for i := 0; i < typ.NumMethod(); i++ {
				if typ.Method(i).Name == "AiPayload" {
					t.Errorf("%s must not expose AiPayload (alias of the unexported aiPayload marker)",
						typ.Name())
				}
			}
		})
	}
}

// ---------------------------------------------------------------------------
// T-AI12-004 — Event constructor round-trip + accessor round-trip (SCN-003)
// ---------------------------------------------------------------------------

// TestNewResponseStartEvent_RoundTripAndKind verifies the producer-side
// event constructor: Kind stamps from payload, payload stored verbatim,
// Sequence follows AI-11's counter, accessor parity via AsResponseStart.
// Per SCN-003.
func TestNewResponseStartEvent_RoundTripAndKind(t *testing.T) {
	ev, err := ai.NewResponseStartEvent("r1", "gpt-x")
	if err != nil {
		t.Fatalf("NewResponseStartEvent(r1, gpt-x) = %v, want nil", err)
	}
	if ev.Kind != ai.EventKindResponseStart {
		t.Errorf("Kind = %q, want %q", ev.Kind, ai.EventKindResponseStart)
	}
	if ev.Sequence < 1 {
		t.Errorf("Sequence = %d, want >= 1 (first event on a fresh producer)", ev.Sequence)
	}

	p, ok := ai.AsResponseStart(ev)
	if !ok {
		t.Fatalf("AsResponseStart returned ok=false; want true (Kind matches)")
	}
	if p.ResponseID() != "r1" {
		t.Errorf("AsResponseStart.ResponseID() = %q, want %q", p.ResponseID(), "r1")
	}
	if p.Model() != "gpt-x" {
		t.Errorf("AsResponseStart.Model() = %q, want %q", p.Model(), "gpt-x")
	}
	if err := ev.Validate(); err != nil {
		t.Errorf("Validate() = %v, want nil (parity guaranteed by NewEvent)", err)
	}
}

// TestNewResponseCompleteEvent_RoundTripAndKind mirrors the start event
// test for the completion event. Per SCN-003.
func TestNewResponseCompleteEvent_RoundTripAndKind(t *testing.T) {
	usage := usageWithCounts(10, 20, 30)
	ev, err := ai.NewResponseCompleteEvent("r1", ai.FinishReasonStop, usage)
	if err != nil {
		t.Fatalf("NewResponseCompleteEvent(r1, stop, usage) = %v, want nil", err)
	}
	if ev.Kind != ai.EventKindResponseComplete {
		t.Errorf("Kind = %q, want %q", ev.Kind, ai.EventKindResponseComplete)
	}

	p, ok := ai.AsResponseComplete(ev)
	if !ok {
		t.Fatalf("AsResponseComplete returned ok=false; want true (Kind matches)")
	}
	if p.ResponseID() != "r1" {
		t.Errorf("AsResponseComplete.ResponseID() = %q, want %q", p.ResponseID(), "r1")
	}
	if p.FinishReason() != ai.FinishReasonStop {
		t.Errorf("AsResponseComplete.FinishReason() = %q, want %q",
			p.FinishReason(), ai.FinishReasonStop)
	}
	if p.Usage() != usage {
		t.Errorf("AsResponseComplete.Usage() = %+v, want %+v", p.Usage(), usage)
	}
	if err := ev.Validate(); err != nil {
		t.Errorf("Validate() = %v, want nil", err)
	}
}

// TestEventConstructors_PropagateValidationError verifies the event
// constructors return the constructor's sentinel verbatim (not wrap it)
// so consumers can errors.Is directly. Per SCN-003 (negative half).
func TestEventConstructors_PropagateValidationError(t *testing.T) {
	_, err := ai.NewResponseStartEvent("", "gpt-x")
	if err == nil {
		t.Fatal("NewResponseStartEvent(empty, gpt-x) = no error, want ErrEmptyResponseID")
	}
	if !errors.Is(err, ai.ErrEmptyResponseID) {
		t.Errorf("error = %v, want errors.Is(_, ErrEmptyResponseID)", err)
	}

	_, err = ai.NewResponseCompleteEvent("", ai.FinishReasonStop, ai.Usage{})
	if err == nil {
		t.Fatal("NewResponseCompleteEvent(empty, stop, Usage{}) = no error, want ErrEmptyResponseID")
	}
	if !errors.Is(err, ai.ErrEmptyResponseID) {
		t.Errorf("error = %v, want errors.Is(_, ErrEmptyResponseID)", err)
	}

	_, err = ai.NewResponseCompleteEvent("r1", ai.FinishReason("bogus"), ai.Usage{})
	if err == nil {
		t.Fatal("NewResponseCompleteEvent(r1, bogus, Usage{}) = no error, want ErrInvalidFinishReason")
	}
	if !errors.Is(err, ai.ErrInvalidFinishReason) {
		t.Errorf("error = %v, want errors.Is(_, ErrInvalidFinishReason)", err)
	}
}

// ---------------------------------------------------------------------------
// T-AI12-005 — IsTerminalKind matrix + AsResponseStart/Complete wrong-kind (SCN-005)
// ---------------------------------------------------------------------------

// TestIsTerminalKind_Matrix verifies IsTerminalKind returns true ONLY for
// the two canonical terminal kinds (ResponseComplete, Error) and false for
// every other reserved kind, the zero value, and an unregistered garbage
// string. Per SCN-005.
func TestIsTerminalKind_Matrix(t *testing.T) {
	cases := []struct {
		kind ai.EventKind
		want bool
	}{
		{ai.EventKindResponseComplete, true},
		{ai.EventKindError, true},
		{ai.EventKindResponseStart, false},
		{ai.EventKindTextStart, false},
		{ai.EventKindTextDelta, false},
		{ai.EventKindTextEnd, false},
		{ai.EventKindReasoningStart, false},
		{ai.EventKindReasoningDelta, false},
		{ai.EventKindReasoningEnd, false},
		{ai.EventKindToolCallStart, false},
		{ai.EventKindToolCallDelta, false},
		{ai.EventKindToolCallEnd, false},
		{ai.EventKind(""), false},
		{ai.EventKind("not.a.kind"), false},
		{ai.EventKind("response.stop"), false}, // not a registered wire kind
	}
	for _, tc := range cases {
		tc := tc
		t.Run(string(tc.kind), func(t *testing.T) {
			if got := ai.IsTerminalKind(tc.kind); got != tc.want {
				t.Errorf("IsTerminalKind(%q) = %v, want %v", tc.kind, got, tc.want)
			}
		})
	}
}

// TestAsResponseStart_AndComplete_WrongKind verifies the accessor parity
// pre-check: AsResponseStart on a Complete-payload event returns
// (ResponseStartPayload{}, false); AsResponseComplete on a Start-payload
// event returns (ResponseCompletePayload{}, false). Per SCN-005.
func TestAsResponseStart_AndComplete_WrongKind(t *testing.T) {
	// Build a Complete event and try AsResponseStart on it.
	completeEv, err := ai.NewResponseCompleteEvent("r1", ai.FinishReasonStop, ai.Usage{})
	if err != nil {
		t.Fatalf("setup NewResponseCompleteEvent = %v", err)
	}
	if got, ok := ai.AsResponseStart(completeEv); ok {
		t.Errorf("AsResponseStart(complete-event) = (%+v, true), want (zero, false)",
			got)
	} else if got != (ai.ResponseStartPayload{}) {
		t.Errorf("AsResponseStart(complete-event) zero payload = %+v, want zero ResponseStartPayload",
			got)
	}

	// Build a Start event and try AsResponseComplete on it.
	startEv, err := ai.NewResponseStartEvent("r1", "gpt-x")
	if err != nil {
		t.Fatalf("setup NewResponseStartEvent = %v", err)
	}
	if got, ok := ai.AsResponseComplete(startEv); ok {
		t.Errorf("AsResponseComplete(start-event) = (%+v, true), want (zero, false)",
			got)
	} else if got != (ai.ResponseCompletePayload{}) {
		t.Errorf("AsResponseComplete(start-event) zero payload = %+v, want zero ResponseCompletePayload",
			got)
	}
}

// ---------------------------------------------------------------------------
// T-AI12-006 — Doc-guard: AI-12 paragraph + producer-scoped rephrase (SCN-007)
// ---------------------------------------------------------------------------

// TestDocGo_AI12ParagraphAndProducerScopedWording verifies doc.go after the
// AI-12 changes: must mention AI-12, ResponseStartPayload, ResponseCompletePayload,
// IsTerminalKind, and the producer-scoped "fresh producer carries sequence 1"
// wording. The stream-scoped "exactly one EventKindResponseStart per stream
// (sequence 1)" wording from AI-11 MUST be rephrased. Per SCN-007.
func TestDocGo_AI12ParagraphAndProducerScopedWording(t *testing.T) {
	body, err := os.ReadFile("doc.go")
	if err != nil {
		t.Fatalf("ReadFile doc.go = %v", err)
	}
	doc := string(body)
	docLower := strings.ToLower(doc)

	mustMention := []string{
		"AI-12",
		"ResponseStartPayload",
		"ResponseCompletePayload",
		"IsTerminalKind",
	}
	for _, phrase := range mustMention {
		if !strings.Contains(doc, phrase) {
			t.Errorf("doc.go must mention %q (AI-12 surface summary)", phrase)
		}
	}

	// Producer-scoped rephrase: AI-11's "Exactly one EventKindResponseStart
	// per stream (sequence 1)" must be replaced by wording that frames the
	// rule per producer. Any of the producer-scoped phrases below satisfies
	// the spec.
	producerScopedPhrases := []string{
		"fresh producer",
		"producer-scoped",
		"per producer",
	}
	matched := false
	for _, phrase := range producerScopedPhrases {
		if strings.Contains(docLower, phrase) {
			matched = true
			break
		}
	}
	if !matched {
		t.Errorf("doc.go must rephrase the AI-11 'sequence 1' rule producer-scoped "+
			"(expected one of: %v)", producerScopedPhrases)
	}
}

// ---------------------------------------------------------------------------
// T-AI12-007 — Import-boundary smoke for response.go (SCN-008)
// ---------------------------------------------------------------------------

// TestResponseGo_ImportsOnlyStdlib verifies response.go imports only the Go
// standard library — no cachicamas imports, no vendor SDKs. This is the
// ADR 0004 Layer 1 purity canary applied to the AI-12 surface. The check
// is source-level (regex over import lines) so it skips gracefully when the
// file is absent (RED phase). Per SCN-008.
func TestResponseGo_ImportsOnlyStdlib(t *testing.T) {
	body, err := os.ReadFile("response.go")
	if err != nil {
		// RED: file does not exist yet — the import-boundary gate is the
		// package-level test in import_boundary_test.go; here we only
		// assert what we can when the file is present.
		t.Skipf("response.go not yet present (RED phase): %v", err)
	}
	src := string(body)
	// Find every quoted import line; flag anything outside the stdlib.
	for _, line := range strings.Split(src, "\n") {
		trimmed := strings.TrimSpace(line)
		if !strings.HasPrefix(trimmed, "\"") {
			continue
		}
		if strings.Contains(trimmed, "github.com/") ||
			strings.Contains(trimmed, "golang.org/") ||
			strings.Contains(trimmed, "cachicamas") {
			t.Errorf("response.go imports a non-stdlib path: %q", trimmed)
		}
	}
}
