package ai_test

import (
	"bytes"
	"encoding/json"
	"errors"
	"os"
	"reflect"
	"strings"
	"testing"
	"unsafe"

	"github.com/cachicamas/backend/database_administrator/src/tools/agent/ai"
)

// T-AI15-001
func TestNewToolCallStartPayload_Matrix(t *testing.T) {
	cases := []struct {
		name, callID, toolName string
		wantErr                error
	}{
		{"valid", "call-A", "weather", nil},
		{"whitespace-padded values", " call-A ", " weather ", nil},
		{"empty call ID", "", "weather", ai.ErrEmptyToolResultCallID},
		{"whitespace call ID", "\t\n", "weather", ai.ErrEmptyToolResultCallID},
		{"long call ID", strings.Repeat("a", ai.MaxToolResultCallIDLength+1), "weather", ai.ErrToolResultCallIDTooLong},
		{"empty name", "call-A", "", ai.ErrEmptyToolCallName},
		{"whitespace name", "call-A", " \t", ai.ErrEmptyToolCallName},
		{"long name", "call-A", strings.Repeat("n", ai.MaxToolNameLength+1), ai.ErrToolCallNameTooLong},
		{"control name", "call-A", "weather\n", ai.ErrInvalidToolCallName},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			p, err := ai.NewToolCallStartPayload(tc.callID, tc.toolName)
			if !errors.Is(err, tc.wantErr) {
				t.Fatalf("error = %v, want %v", err, tc.wantErr)
			}
			if tc.wantErr != nil {
				if p != (ai.ToolCallStartPayload{}) {
					t.Errorf("payload = %+v, want zero", p)
				}
				return
			}
			if p.CallID() != tc.callID || p.Name() != tc.toolName || p.Kind() != ai.EventKindToolCallStart {
				t.Errorf("payload accessors = (%q,%q,%q), want (%q,%q,%q)", p.CallID(), p.Name(), p.Kind(), tc.callID, tc.toolName, ai.EventKindToolCallStart)
			}
			if err := p.Validate(); err != nil {
				t.Errorf("Validate() = %v, want nil", err)
			}
		})
	}
}

// T-AI15-002
func TestNewToolCallDeltaPayload_Matrix(t *testing.T) {
	cases := []struct {
		name, callID string
		delta        json.RawMessage
		wantErr      error
	}{
		{"valid fragment", "call-A", json.RawMessage(`{"city":"`), nil},
		{"single byte", "call-A", json.RawMessage(`{`), nil},
		{"empty call ID", "", json.RawMessage(`{`), ai.ErrEmptyToolResultCallID},
		{"whitespace call ID", "  ", json.RawMessage(`{`), ai.ErrEmptyToolResultCallID},
		{"long call ID", strings.Repeat("a", ai.MaxToolResultCallIDLength+1), json.RawMessage(`{`), ai.ErrToolResultCallIDTooLong},
		{"nil delta", "call-A", nil, ai.ErrMalformedToolCallArguments},
		{"empty delta", "call-A", json.RawMessage{}, ai.ErrMalformedToolCallArguments},
		{"oversize delta", "call-A", json.RawMessage(strings.Repeat("x", ai.MaxToolCallDeltaLength+1)), ai.ErrMalformedToolCallArguments},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			p, err := ai.NewToolCallDeltaPayload(tc.callID, tc.delta)
			if !errors.Is(err, tc.wantErr) {
				t.Fatalf("error = %v, want %v", err, tc.wantErr)
			}
			if tc.wantErr != nil {
				if !reflect.ValueOf(p).IsZero() {
					t.Errorf("payload = %+v, want zero", p)
				}
				return
			}
			if p.CallID() != tc.callID || !bytes.Equal(p.Delta(), tc.delta) || p.Kind() != ai.EventKindToolCallDelta {
				t.Errorf("delta payload did not preserve inputs or kind")
			}
			if err := p.Validate(); err != nil {
				t.Errorf("Validate() = %v, want nil", err)
			}
		})
	}
}

// T-AI15-003
func TestNewToolCallEndPayload_Matrix(t *testing.T) {
	valid := json.RawMessage(`{"city":"Tokyo"}`)
	oversize := json.RawMessage(`"` + strings.Repeat("x", ai.MaxToolCallArgumentsLength) + `"`)
	cases := []struct {
		name, callID, toolName string
		arguments              json.RawMessage
		wantErr                error
	}{
		{"valid", "call-A", "weather", valid, nil},
		{"empty call ID", "", "weather", valid, ai.ErrEmptyToolResultCallID},
		{"whitespace call ID", " \t", "weather", valid, ai.ErrEmptyToolResultCallID},
		{"long call ID", strings.Repeat("a", ai.MaxToolResultCallIDLength+1), "weather", valid, ai.ErrToolResultCallIDTooLong},
		{"empty name", "call-A", "", valid, ai.ErrEmptyToolCallName},
		{"whitespace name", "call-A", "\n", valid, ai.ErrEmptyToolCallName},
		{"long name", "call-A", strings.Repeat("n", ai.MaxToolNameLength+1), valid, ai.ErrToolCallNameTooLong},
		{"control name", "call-A", "weather\x00", valid, ai.ErrInvalidToolCallName},
		{"nil arguments", "call-A", "weather", nil, ai.ErrMalformedToolCallArguments},
		{"empty arguments", "call-A", "weather", json.RawMessage{}, ai.ErrMalformedToolCallArguments},
		{"invalid arguments", "call-A", "weather", json.RawMessage(`{"city":`), ai.ErrMalformedToolCallArguments},
		{"oversize arguments", "call-A", "weather", oversize, ai.ErrToolCallArgumentsTooLong},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			p, err := ai.NewToolCallEndPayload(tc.callID, tc.toolName, tc.arguments)
			if !errors.Is(err, tc.wantErr) {
				t.Fatalf("error = %v, want %v", err, tc.wantErr)
			}
			if tc.wantErr != nil {
				if !reflect.ValueOf(p).IsZero() {
					t.Errorf("payload = %+v, want zero", p)
				}
				return
			}
			if p.CallID() != tc.callID || p.Name() != tc.toolName || !bytes.Equal(p.Arguments(), tc.arguments) || p.Kind() != ai.EventKindToolCallEnd {
				t.Errorf("end payload did not preserve inputs or kind")
			}
			if err := p.Validate(); err != nil {
				t.Errorf("Validate() = %v, want nil", err)
			}
		})
	}
}

// T-AI15-004
func TestCallIDCorrelation_Interleaved(t *testing.T) {
	inputs := []struct {
		callID string
		delta  json.RawMessage
	}{{"A", json.RawMessage(`{"a":`)}, {"A", json.RawMessage(`1`)}, {"B", json.RawMessage(`{"b":`)}, {"A", json.RawMessage(`}`)}, {"B", json.RawMessage(`2`)}, {"B", json.RawMessage(`}`)}}
	grouped := map[string][][]byte{}
	for _, input := range inputs {
		p, err := ai.NewToolCallDeltaPayload(input.callID, input.delta)
		if err != nil {
			t.Fatalf("NewToolCallDeltaPayload(%q) = %v", input.callID, err)
		}
		grouped[p.CallID()] = append(grouped[p.CallID()], p.Delta())
	}
	if got := bytes.Join(grouped["A"], nil); !bytes.Equal(got, []byte(`{"a":1}`)) {
		t.Errorf("A = %q, want %q", got, `{"a":1}`)
	}
	if got := bytes.Join(grouped["B"], nil); !bytes.Equal(got, []byte(`{"b":2}`)) {
		t.Errorf("B = %q, want %q", got, `{"b":2}`)
	}
}

// T-AI15-005
func TestConcatenationInvariant_SingleCallID(t *testing.T) {
	fragments := []json.RawMessage{json.RawMessage(`{"city"`), json.RawMessage(`:"Tokyo"`), json.RawMessage(`}`)}
	var deltas [][]byte
	for _, fragment := range fragments {
		p, err := ai.NewToolCallDeltaPayload("call-A", fragment)
		if err != nil {
			t.Fatalf("NewToolCallDeltaPayload = %v", err)
		}
		deltas = append(deltas, p.Delta())
	}
	assembled := bytes.Join(deltas, nil)
	end, err := ai.NewToolCallEndPayload("call-A", "weather", assembled)
	if err != nil {
		t.Fatalf("NewToolCallEndPayload = %v", err)
	}
	if !bytes.Equal(assembled, end.Arguments()) {
		t.Errorf("assembled = %q, End.Arguments = %q", assembled, end.Arguments())
	}
	var parsed any
	if err := json.Unmarshal(assembled, &parsed); err != nil {
		t.Errorf("assembled JSON = %v", err)
	}
}

// T-AI15-006
func TestToolCallEventPayloads_InterfaceIsSealed(t *testing.T) {
	start, _ := ai.NewToolCallStartPayload("A", "weather")
	delta, _ := ai.NewToolCallDeltaPayload("A", json.RawMessage(`{`))
	end, _ := ai.NewToolCallEndPayload("A", "weather", json.RawMessage(`{}`))
	for _, tc := range []struct {
		name string
		val  any
		kind ai.EventKind
	}{{"ToolCallStartPayload", start, ai.EventKindToolCallStart}, {"ToolCallDeltaPayload", delta, ai.EventKindToolCallDelta}, {"ToolCallEndPayload", end, ai.EventKindToolCallEnd}} {
		t.Run(tc.name, func(t *testing.T) {
			typ := reflect.TypeOf(tc.val)
			if got := reflect.ValueOf(tc.val).MethodByName("Kind").Call(nil)[0].Interface().(ai.EventKind); got != tc.kind {
				t.Errorf("Kind = %q, want %q", got, tc.kind)
			}
			for i := 0; i < typ.NumField(); i++ {
				if typ.Field(i).PkgPath == "" {
					t.Errorf("field %s is exported", typ.Field(i).Name)
				}
			}
			for i := 0; i < typ.NumMethod(); i++ {
				name := typ.Method(i).Name
				if name == "AiPayload" || name == "MarshalJSON" || name == "UnmarshalJSON" || name == "MarshalText" || name == "UnmarshalText" || name == "Clone" || strings.HasPrefix(name, "With") {
					t.Errorf("forbidden exported method %s.%s", typ.Name(), name)
				}
			}
		})
	}
}

// T-AI15-007
func TestSentinels_ReuseAndDistinctness(t *testing.T) {
	sentinels := []error{ai.ErrEmptyToolCallName, ai.ErrToolCallNameTooLong, ai.ErrInvalidToolCallName, ai.ErrMalformedToolCallArguments, ai.ErrToolCallArgumentsTooLong, ai.ErrEmptyToolResultCallID, ai.ErrToolResultCallIDTooLong}
	for i, a := range sentinels {
		for j, b := range sentinels {
			if i < j && (a == b || errors.Is(a, b) || errors.Is(b, a)) {
				t.Errorf("sentinel collision: %v and %v", a, b)
			}
		}
	}
	_, startErr := ai.NewToolCallStartPayload("A", "")
	_, endErr := ai.NewToolCallEndPayload("A", "", json.RawMessage(`{}`))
	for _, err := range []error{startErr, endErr} {
		if !errors.Is(err, ai.ErrEmptyToolCallName) || err.Error() != ai.ErrEmptyToolCallName.Error() {
			t.Errorf("error = %v, want verbatim ErrEmptyToolCallName", err)
		}
	}
}

// T-AI15-008
func TestNewToolCallEvent_RoundTrip(t *testing.T) {
	start, err := ai.NewToolCallStartEvent("A", "weather")
	if err != nil {
		t.Fatal(err)
	}
	delta, err := ai.NewToolCallDeltaEvent("A", json.RawMessage(`{}`))
	if err != nil {
		t.Fatal(err)
	}
	end, err := ai.NewToolCallEndEvent("A", "weather", json.RawMessage(`{}`))
	if err != nil {
		t.Fatal(err)
	}
	if start.Sequence >= delta.Sequence || delta.Sequence >= end.Sequence {
		t.Errorf("sequences = %d,%d,%d, want monotonic", start.Sequence, delta.Sequence, end.Sequence)
	}
	if p, ok := ai.AsToolCallStart(start); !ok || p.CallID() != "A" {
		t.Errorf("AsToolCallStart = (%+v,%v)", p, ok)
	}
	if p, ok := ai.AsToolCallDelta(delta); !ok || !bytes.Equal(p.Delta(), []byte(`{}`)) {
		t.Errorf("AsToolCallDelta = (%+v,%v)", p, ok)
	}
	if p, ok := ai.AsToolCallEnd(end); !ok || !bytes.Equal(p.Arguments(), []byte(`{}`)) {
		t.Errorf("AsToolCallEnd = (%+v,%v)", p, ok)
	}
	for _, ev := range []ai.Event{start, delta, end} {
		if err := ev.Validate(); err != nil {
			t.Errorf("Event.Validate() = %v", err)
		}
	}
	zero, err := ai.NewToolCallDeltaEvent("A", nil)
	if !errors.Is(err, ai.ErrMalformedToolCallArguments) || zero != (ai.Event{}) {
		t.Errorf("invalid event = (%+v,%v), want zero + ErrMalformedToolCallArguments", zero, err)
	}
}

// T-AI15-009
func TestAsToolCall_Helpers_KindParity(t *testing.T) {
	start, _ := ai.NewToolCallStartEvent("A", "weather")
	delta, _ := ai.NewToolCallDeltaEvent("A", json.RawMessage(`{`))
	end, _ := ai.NewToolCallEndEvent("A", "weather", json.RawMessage(`{}`))
	if _, ok := ai.AsToolCallStart(start); !ok {
		t.Error("AsToolCallStart rejected right kind")
	}
	if _, ok := ai.AsToolCallDelta(delta); !ok {
		t.Error("AsToolCallDelta rejected right kind")
	}
	if _, ok := ai.AsToolCallEnd(end); !ok {
		t.Error("AsToolCallEnd rejected right kind")
	}
	for _, ev := range []ai.Event{delta, end} {
		if _, ok := ai.AsToolCallStart(ev); ok {
			t.Errorf("AsToolCallStart accepted %q", ev.Kind)
		}
	}
	for _, ev := range []ai.Event{start, end} {
		if _, ok := ai.AsToolCallDelta(ev); ok {
			t.Errorf("AsToolCallDelta accepted %q", ev.Kind)
		}
	}
	for _, ev := range []ai.Event{start, delta} {
		if _, ok := ai.AsToolCallEnd(ev); ok {
			t.Errorf("AsToolCallEnd accepted %q", ev.Kind)
		}
	}
}

// T-AI15-010
func TestValidate_IdempotencyAndPurity(t *testing.T) {
	cases := []struct {
		name     string
		validate func() error
		want     error
	}{
		{"valid start", func() error { return toolCallStartPayloadOf("A", "weather").Validate() }, nil},
		{"invalid start", func() error { return toolCallStartPayloadOf("", "weather").Validate() }, ai.ErrEmptyToolResultCallID},
		{"valid delta", func() error { return toolCallDeltaPayloadOf("A", json.RawMessage(`{`)).Validate() }, nil},
		{"invalid delta", func() error { return toolCallDeltaPayloadOf("A", nil).Validate() }, ai.ErrMalformedToolCallArguments},
		{"valid end", func() error { return toolCallEndPayloadOf("A", "weather", json.RawMessage(`{}`)).Validate() }, nil},
		{"invalid end", func() error { return toolCallEndPayloadOf("A", "weather", nil).Validate() }, ai.ErrMalformedToolCallArguments},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			for i := 0; i < 2; i++ {
				if err := tc.validate(); !errors.Is(err, tc.want) {
					t.Errorf("Validate #%d = %v, want %v", i+1, err, tc.want)
				}
			}
		})
	}
}

// T-AI15-011
func TestAccessors_ReturnConstructorInputsVerbatim(t *testing.T) {
	callID, name := `call\x41`, "name_αβγ"
	start, _ := ai.NewToolCallStartPayload(callID, name)
	if start.CallID() != callID || start.Name() != name {
		t.Error("start accessors changed input")
	}
	deltaBytes := json.RawMessage{0xF0, 0x9F, 0x98}
	delta, _ := ai.NewToolCallDeltaPayload(callID, deltaBytes)
	if delta.CallID() != callID || !bytes.Equal(delta.Delta(), deltaBytes) {
		t.Error("delta accessors changed input")
	}
	args := json.RawMessage(` { "x" : 1 } `)
	end, _ := ai.NewToolCallEndPayload(callID, name, args)
	if end.CallID() != callID || end.Name() != name || !bytes.Equal(end.Arguments(), args) {
		t.Error("end accessors changed input")
	}
}

// T-AI15-012
func TestDocGoParagraph_Guard(t *testing.T) {
	body, err := os.ReadFile("doc.go")
	if err != nil {
		t.Fatal(err)
	}
	doc := string(body)
	if !strings.Contains(doc, "// AI-15 paragraph") {
		t.Skip("AI-15 package paragraph is added by the dedicated DOCS commit")
	}
	for _, phrase := range []string{"ToolCallStartPayload", "ToolCallDeltaPayload", "ToolCallEndPayload", "MaxToolCallDeltaLength", "toolcall_event.go"} {
		if !strings.Contains(doc, phrase) {
			t.Errorf("doc.go missing %q", phrase)
		}
	}
	if trimmed := bytes.TrimSpace(body); len(trimmed) == 0 || trimmed[len(trimmed)-1] != '.' {
		t.Error("doc.go last non-whitespace byte must be period")
	}
}

// T-AI15-013
func TestToolCallEventGo_ImportsOnlyStdlib(t *testing.T) {
	body, err := os.ReadFile("toolcall_event.go")
	if err != nil {
		t.Skipf("toolcall_event.go absent in RED: %v", err)
	}
	for _, forbidden := range []string{"github.com/", "golang.org/", "cachicamas_agent", "cachicamas_coding", "unicode/utf8"} {
		if bytes.Contains(body, []byte(forbidden)) {
			t.Errorf("toolcall_event.go contains forbidden dependency %q", forbidden)
		}
	}
}

// T-AI15-014
func TestOrphanDelta_Accepted(t *testing.T) {
	p, err := ai.NewToolCallDeltaPayload("never-started", json.RawMessage(`{"x":`))
	if err != nil || p.CallID() != "never-started" {
		t.Errorf("orphan delta = (%+v,%v), want accepted", p, err)
	}
	for _, forbidden := range []string{"IsOrphan", "FirstForCall", "StreamContext", "RequiresPriorStart"} {
		if _, ok := reflect.TypeOf(p).MethodByName(forbidden); ok {
			t.Errorf("unexpected Layer-2 method %s", forbidden)
		}
	}
}

// T-AI15-015
func TestEndPayload_AllEmptyFieldsLiteral(t *testing.T) {
	if err := (ai.ToolCallEndPayload{}).Validate(); !errors.Is(err, ai.ErrEmptyToolResultCallID) {
		t.Errorf("zero Validate = %v", err)
	}
	if err := toolCallEndPayloadOf("", "", nil).Validate(); !errors.Is(err, ai.ErrEmptyToolResultCallID) {
		t.Errorf("unsafe zero Validate = %v", err)
	}
	defer func() {
		if recover() == nil {
			t.Error("nil pointer Validate did not panic")
		}
	}()
	var p *ai.ToolCallEndPayload
	_ = p.Validate()
}

// T-AI15-016
func TestEndToEndReconstruction_InterleavedMultiCall(t *testing.T) {
	starts := []struct{ id, name string }{{"A", "weather"}, {"B", "search"}}
	for _, s := range starts {
		if _, err := ai.NewToolCallStartEvent(s.id, s.name); err != nil {
			t.Fatal(err)
		}
	}
	arrivals := []struct {
		id    string
		delta json.RawMessage
	}{{"A", json.RawMessage(`{"city":`)}, {"B", json.RawMessage(`{"query":`)}, {"A", json.RawMessage(`"Tokyo"}`)}, {"B", json.RawMessage(`"TDD"}`)}}
	grouped := map[string][][]byte{}
	for _, arrival := range arrivals {
		ev, err := ai.NewToolCallDeltaEvent(arrival.id, arrival.delta)
		if err != nil {
			t.Fatal(err)
		}
		p, ok := ai.AsToolCallDelta(ev)
		if !ok {
			t.Fatalf("AsToolCallDelta(%q) = false", arrival.id)
		}
		grouped[p.CallID()] = append(grouped[p.CallID()], p.Delta())
	}
	for _, s := range starts {
		assembled := bytes.Join(grouped[s.id], nil)
		ev, err := ai.NewToolCallEndEvent(s.id, s.name, assembled)
		if err != nil {
			t.Fatal(err)
		}
		end, ok := ai.AsToolCallEnd(ev)
		if !ok || !bytes.Equal(end.Arguments(), assembled) {
			t.Errorf("end %s did not reconstruct", s.id)
		}
	}
	oneShot, err := ai.NewToolCallEndPayload("C", "one-shot", json.RawMessage(`{}`))
	if err != nil || !bytes.Equal(oneShot.Arguments(), []byte(`{}`)) {
		t.Errorf("one-shot end = (%+v,%v)", oneShot, err)
	}
}

// T-AI15-017
func TestMaxToolCallDeltaLength_ExactBoundary(t *testing.T) {
	if ai.MaxToolCallDeltaLength != 1<<16 {
		t.Errorf("constant = %d, want %d", ai.MaxToolCallDeltaLength, 1<<16)
	}
	if _, err := ai.NewToolCallDeltaPayload("A", json.RawMessage(strings.Repeat("x", ai.MaxToolCallDeltaLength))); err != nil {
		t.Errorf("exact max rejected: %v", err)
	}
	if _, err := ai.NewToolCallDeltaPayload("A", json.RawMessage(strings.Repeat("x", ai.MaxToolCallDeltaLength+1))); !errors.Is(err, ai.ErrMalformedToolCallArguments) {
		t.Errorf("max+1 error = %v", err)
	}
}

// T-AI15-018
func TestMaxToolResultCallIDLength_Reuse(t *testing.T) {
	if _, err := ai.NewToolCallStartPayload(strings.Repeat("a", ai.MaxToolResultCallIDLength), "weather"); err != nil {
		t.Errorf("512-byte callID rejected: %v", err)
	}
	if _, err := ai.NewToolCallStartPayload(strings.Repeat("a", ai.MaxToolResultCallIDLength+1), "weather"); !errors.Is(err, ai.ErrToolResultCallIDTooLong) {
		t.Errorf("513-byte error = %v", err)
	}
	body, err := os.ReadFile("toolcall_event.go")
	if err == nil && bytes.Contains(body, []byte("MaxToolCallIDLength")) {
		t.Error("toolcall_event.go must reuse MaxToolResultCallIDLength")
	}
}

// T-AI15-019
func TestDeltaPayload_NoUTF8BoundaryCheck(t *testing.T) {
	raw := json.RawMessage{0xF0, 0x9F, 0x98}
	p, err := ai.NewToolCallDeltaPayload("A", raw)
	if err != nil || !bytes.Equal(p.Delta(), raw) || p.Validate() != nil {
		t.Errorf("raw high bytes = (%x,%v), want accepted", p.Delta(), err)
	}
	for _, fragment := range []json.RawMessage{json.RawMessage(`{"emoji":"\uD83D`), json.RawMessage(`\uDE00"}`)} {
		p, err := ai.NewToolCallDeltaPayload("A", fragment)
		if err != nil || p.Validate() != nil {
			t.Errorf("escaped fragment %q rejected: %v", fragment, err)
		}
	}
}

type startPayloadLayout struct{ callID, name string }
type deltaPayloadLayout struct {
	callID string
	delta  json.RawMessage
}
type endPayloadLayout struct {
	callID, name string
	arguments    json.RawMessage
}

func toolCallStartPayloadOf(callID, name string) ai.ToolCallStartPayload {
	layout := startPayloadLayout{callID, name}
	return *(*ai.ToolCallStartPayload)(unsafe.Pointer(&layout))
}
func toolCallDeltaPayloadOf(callID string, delta json.RawMessage) ai.ToolCallDeltaPayload {
	layout := deltaPayloadLayout{callID, delta}
	return *(*ai.ToolCallDeltaPayload)(unsafe.Pointer(&layout))
}
func toolCallEndPayloadOf(callID, name string, arguments json.RawMessage) ai.ToolCallEndPayload {
	layout := endPayloadLayout{callID, name, arguments}
	return *(*ai.ToolCallEndPayload)(unsafe.Pointer(&layout))
}
