package ai_test

import (
	"encoding/json"
	"errors"
	"math"
	"os"
	"reflect"
	"strings"
	"testing"

	"github.com/cachicamas/backend/database_administrator/src/tools/agent/ai"
)

func TestToolChoiceConstants(t *testing.T) {
	cases := []struct {
		got  ai.ToolChoice
		want string
	}{
		{ai.ToolChoiceAuto, "auto"},
		{ai.ToolChoiceNone, "none"},
		{ai.ToolChoiceRequired, "required"},
	}
	for _, tc := range cases {
		if string(tc.got) != tc.want {
			t.Errorf("ToolChoice = %q, want %q", tc.got, tc.want)
		}
	}
	if got := reflect.TypeOf(ai.ToolChoice("")).NumMethod(); got != 0 {
		t.Errorf("ToolChoice has %d methods, want 0", got)
	}
}

func TestNewRequest_EmptyModel(t *testing.T) {
	req, err := ai.NewRequest("", nil, nil, ai.GenerationOptions{})
	assertErrorIs(t, err, ai.ErrEmptyModel)
	if !reflect.DeepEqual(req, ai.Request{}) {
		t.Errorf("NewRequest error path returned non-zero Request: %#v", req)
	}
}

func TestNewRequest_WhitespaceModel(t *testing.T) {
	for _, model := range []string{"   ", "\t\n", "\u00a0", "\u3000"} {
		t.Run(model, func(t *testing.T) {
			_, err := ai.NewRequest(model, nil, nil, ai.GenerationOptions{})
			assertErrorIs(t, err, ai.ErrWhitespaceModel)
		})
	}
}

func TestNewRequest_MinimalRequest(t *testing.T) {
	req, err := ai.NewRequest("gpt-4o", nil, nil, ai.GenerationOptions{})
	if err != nil {
		t.Fatalf("NewRequest minimal request error = %v, want nil", err)
	}
	if req.Model() != "gpt-4o" {
		t.Errorf("Model() = %q, want gpt-4o", req.Model())
	}
	if err := req.Validate(); err != nil {
		t.Errorf("Validate() = %v, want nil", err)
	}
}

func TestNewRequest_PropagatesMessageError(t *testing.T) {
	_, err := ai.NewRequest("gpt-4o", []ai.Message{{}}, nil, ai.GenerationOptions{})
	assertErrorIs(t, err, ai.ErrInvalidRole)
}

func TestValidate_PropagatesMessageError(t *testing.T) {
	messages := []ai.Message{mustMsg(ai.RoleUser, "message-1", mustTextPart(t, "hello"))}
	req, err := ai.NewRequest("gpt-4o", messages, nil, ai.GenerationOptions{})
	if err != nil {
		t.Fatalf("setup NewRequest error = %v", err)
	}
	messages[0].Content = nil
	assertErrorIs(t, req.Validate(), ai.ErrEmptyContent)
}

func TestNewRequest_DuplicateTools(t *testing.T) {
	t.Run("byte-equal names rejected", func(t *testing.T) {
		tools := []ai.ToolDeclaration{
			mustTool(t, "search", "first", json.RawMessage(`{"type":"object"}`)),
			mustTool(t, "search", "second", json.RawMessage(`{"type":"object"}`)),
		}
		_, err := ai.NewRequest("gpt-4o", nil, tools, ai.GenerationOptions{})
		assertErrorIs(t, err, ai.ErrDuplicateToolName)
	})
	t.Run("whitespace-padded names remain distinct", func(t *testing.T) {
		tools := []ai.ToolDeclaration{
			mustTool(t, "search", "first", json.RawMessage(`{"type":"object"}`)),
			mustTool(t, "search ", "second", json.RawMessage(`{"type":"object"}`)),
		}
		if _, err := ai.NewRequest("gpt-4o", nil, tools, ai.GenerationOptions{}); err != nil {
			t.Errorf("NewRequest distinct tool names error = %v, want nil", err)
		}
	})
}

func TestNewRequest_ToolChoiceRequiresTools(t *testing.T) {
	opts := mustOptions(t, "", 0, 0, 0, nil, ai.ToolChoiceRequired)
	_, err := ai.NewRequest("gpt-4o", nil, nil, opts)
	assertErrorIs(t, err, ai.ErrToolChoiceRequiresTools)
}

func TestValidate_MaxOutputTokensBoundaries(t *testing.T) {
	cases := []struct {
		name string
		in   int
		want error
	}{
		{"negative", -1, ai.ErrInvalidMaxOutputTokens},
		{"unset", 0, nil},
		{"maximum", ai.MaxRequestMaxOutputTokens, nil},
		{"maximum plus one", ai.MaxRequestMaxOutputTokens + 1, ai.ErrInvalidMaxOutputTokens},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := ai.NewGenerationOptions("", tc.in, 0, 0, nil, ai.ToolChoiceAuto)
			assertErrorIs(t, err, tc.want)
		})
	}
}

func TestValidate_TemperatureRange(t *testing.T) {
	cases := []struct {
		name string
		in   float64
		want error
	}{
		{"nan", math.NaN(), ai.ErrTemperatureOutOfRange},
		{"positive infinity", math.Inf(1), ai.ErrTemperatureOutOfRange},
		{"negative infinity", math.Inf(-1), ai.ErrTemperatureOutOfRange},
		{"below minimum", -math.SmallestNonzeroFloat64, ai.ErrTemperatureOutOfRange},
		{"minimum", 0, nil},
		{"maximum", 2, nil},
		{"above maximum", 2.0000001, ai.ErrTemperatureOutOfRange},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := ai.NewGenerationOptions("", 0, tc.in, 0, nil, ai.ToolChoiceAuto)
			assertErrorIs(t, err, tc.want)
		})
	}
}

func TestValidate_TopPRange(t *testing.T) {
	cases := []struct {
		name string
		in   float64
		want error
	}{
		{"nan", math.NaN(), ai.ErrTopPOutOfRange},
		{"infinity", math.Inf(1), ai.ErrTopPOutOfRange},
		{"below minimum", -math.SmallestNonzeroFloat64, ai.ErrTopPOutOfRange},
		{"minimum", 0, nil},
		{"maximum", 1, nil},
		{"above maximum", 1.0000001, ai.ErrTopPOutOfRange},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := ai.NewGenerationOptions("", 0, 0, tc.in, nil, ai.ToolChoiceAuto)
			assertErrorIs(t, err, tc.want)
		})
	}
}

func TestValidate_StopSequences_TooMany(t *testing.T) {
	for _, tc := range []struct {
		name string
		in   []string
		want error
	}{
		{"maximum", makeStops(ai.MaxRequestStopSequences), nil},
		{"maximum plus one", makeStops(ai.MaxRequestStopSequences + 1), ai.ErrTooManyStopSequences},
	} {
		t.Run(tc.name, func(t *testing.T) {
			_, err := ai.NewGenerationOptions("", 0, 0, 0, tc.in, ai.ToolChoiceAuto)
			assertErrorIs(t, err, tc.want)
		})
	}
}

func TestValidate_StopSequences_TooLong(t *testing.T) {
	for _, tc := range []struct {
		name string
		in   string
		want error
	}{
		{"maximum", strings.Repeat("x", ai.MaxRequestStopSequenceLength), nil},
		{"maximum plus one", strings.Repeat("x", ai.MaxRequestStopSequenceLength+1), ai.ErrStopSequenceTooLong},
	} {
		t.Run(tc.name, func(t *testing.T) {
			_, err := ai.NewGenerationOptions("", 0, 0, 0, []string{tc.in}, ai.ToolChoiceAuto)
			assertErrorIs(t, err, tc.want)
		})
	}
}

func TestValidate_StopSequences_Empty(t *testing.T) {
	_, err := ai.NewGenerationOptions("", 0, 0, 0, []string{""}, ai.ToolChoiceAuto)
	assertErrorIs(t, err, ai.ErrEmptyStopSequence)
}

func TestValidate_StopSequences_Whitespace(t *testing.T) {
	_, err := ai.NewGenerationOptions("", 0, 0, 0, []string{"\t \n"}, ai.ToolChoiceAuto)
	assertErrorIs(t, err, ai.ErrWhitespaceStopSequence)
}

func TestValidate_StopSequencesWhitespaceVariants(t *testing.T) {
	for _, stop := range []string{"\t", "\u00a0", "\u3000"} {
		t.Run(stop, func(t *testing.T) {
			_, err := ai.NewGenerationOptions("", 0, 0, 0, []string{stop}, ai.ToolChoiceAuto)
			assertErrorIs(t, err, ai.ErrWhitespaceStopSequence)
		})
	}
}

func TestValidate_InvalidToolChoice(t *testing.T) {
	for _, choice := range []ai.ToolChoice{"garbage", "AUTO", "required "} {
		t.Run(string(choice), func(t *testing.T) {
			_, err := ai.NewGenerationOptions("", 0, 0, 0, nil, choice)
			assertErrorIs(t, err, ai.ErrInvalidToolChoice)
		})
	}
}

func TestValidate_SystemInstruction_Length(t *testing.T) {
	for _, tc := range []struct {
		name string
		in   string
		want error
	}{
		{"maximum", strings.Repeat("s", ai.MaxRequestSystemInstructionLength), nil},
		{"maximum plus one", strings.Repeat("s", ai.MaxRequestSystemInstructionLength+1), ai.ErrSystemInstructionTooLong},
	} {
		t.Run(tc.name, func(t *testing.T) {
			_, err := ai.NewGenerationOptions(tc.in, 0, 0, 0, nil, ai.ToolChoiceAuto)
			assertErrorIs(t, err, tc.want)
		})
	}
}

func TestValidate_SystemInstruction_Whitespace(t *testing.T) {
	_, err := ai.NewGenerationOptions("   ", 0, 0, 0, nil, ai.ToolChoiceAuto)
	assertErrorIs(t, err, ai.ErrWhitespaceSystemInstruction)
}

func TestValidate_SystemInstructionWhitespaceVariants(t *testing.T) {
	for _, instruction := range []string{"\t", "\u00a0", "\u3000"} {
		t.Run(instruction, func(t *testing.T) {
			_, err := ai.NewGenerationOptions(instruction, 0, 0, 0, nil, ai.ToolChoiceAuto)
			assertErrorIs(t, err, ai.ErrWhitespaceSystemInstruction)
		})
	}
}

func TestAccessors_ReturnVerbatimInputs(t *testing.T) {
	stops := []string{"END", "STOP"}
	opts := mustOptions(t, "be concise", 512, 0.7, 0.9, stops, ai.ToolChoiceRequired)
	messages := []ai.Message{mustMsg(ai.RoleUser, "message-1", mustTextPart(t, "hello"))}
	tools := []ai.ToolDeclaration{mustTool(t, "search", "Search data", json.RawMessage(`{"type":"object"}`))}
	req, err := ai.NewRequest("gpt-4o", messages, tools, opts)
	if err != nil {
		t.Fatalf("NewRequest error = %v, want nil", err)
	}
	if req.Model() != "gpt-4o" || !reflect.DeepEqual(req.Messages(), messages) || !reflect.DeepEqual(req.Tools(), tools) || !reflect.DeepEqual(req.Options(), opts) {
		t.Errorf("Request accessors did not preserve inputs")
	}
	if opts.SystemInstruction() != "be concise" || opts.MaxOutputTokens() != 512 || opts.Temperature() != 0.7 || opts.TopP() != 0.9 || !reflect.DeepEqual(opts.StopSequences(), stops) || opts.ToolChoice() != ai.ToolChoiceRequired {
		t.Errorf("GenerationOptions accessors did not preserve inputs")
	}
}

func TestRequest_HasNoMarshalJSON(t *testing.T) {
	assertNoMethods(t, reflect.TypeOf(ai.Request{}), "MarshalJSON", "UnmarshalJSON", "MarshalText", "UnmarshalText")
}

func TestGenerationOptions_HasNoMarshalJSON(t *testing.T) {
	assertNoMethods(t, reflect.TypeOf(ai.GenerationOptions{}), "MarshalJSON", "UnmarshalJSON", "MarshalText", "UnmarshalText")
}

func TestRequest_HasNoClone(t *testing.T) {
	assertNoMethods(t, reflect.TypeOf(ai.Request{}), "Clone")
}

func TestRequest_HasNoSetters(t *testing.T) {
	for _, typ := range []reflect.Type{reflect.TypeOf(ai.Request{}), reflect.TypeOf(ai.GenerationOptions{})} {
		for i := 0; i < typ.NumMethod(); i++ {
			if strings.HasPrefix(typ.Method(i).Name, "With") {
				t.Errorf("%s exposes forbidden setter %s", typ, typ.Method(i).Name)
			}
		}
		for i := 0; i < typ.NumField(); i++ {
			if typ.Field(i).PkgPath == "" {
				t.Errorf("%s field %s is exported", typ, typ.Field(i).Name)
			}
		}
	}
}

func TestDocGo_AI09Paragraph(t *testing.T) {
	data, err := os.ReadFile("doc.go")
	if err != nil {
		t.Fatalf("read doc.go: %v", err)
	}
	src := string(data)
	packageIndex := strings.Index(src, "package ai")
	paragraphIndex := strings.Index(src, "AI-09 paragraph")
	if packageIndex < 0 || paragraphIndex < packageIndex || !strings.Contains(src[paragraphIndex:], "Request") {
		t.Errorf("doc.go must contain the AI-09 Request paragraph after package ai")
	}
}

func TestImportBoundary_StaysStdlib(t *testing.T) {
	for _, name := range []string{"request.go", "request_test.go", "toolchoice.go"} {
		data, err := os.ReadFile(name)
		if err != nil {
			t.Fatalf("read %s: %v", name, err)
		}
		if strings.Contains(string(data), "github.com/cachicamas/backend/database_administrator/src/") && name != "request_test.go" {
			t.Errorf("%s imports a cachicamas package across the Layer 1 boundary", name)
		}
	}
}

func TestSentinels_SelfDistinct(t *testing.T) {
	for _, sentinel := range requestSentinels() {
		if !errors.Is(sentinel, sentinel) {
			t.Errorf("%v must satisfy errors.Is(err, err)", sentinel)
		}
		if !strings.HasPrefix(sentinel.Error(), "ai: ") {
			t.Errorf("sentinel %q lacks ai: prefix", sentinel)
		}
	}
}

func TestSentinels_PairwiseDistinct(t *testing.T) {
	sentinels := requestSentinels()
	if len(sentinels) != 13 {
		t.Fatalf("request sentinel count = %d, want 13", len(sentinels))
	}
	for i := range sentinels {
		for j := range sentinels {
			if i != j && errors.Is(sentinels[i], sentinels[j]) {
				t.Errorf("sentinels %v and %v alias via errors.Is", sentinels[i], sentinels[j])
			}
		}
	}
}

func TestRequest_ReValidateAfterConstruction(t *testing.T) {
	messages := []ai.Message{mustMsg(ai.RoleUser, "message-1", mustTextPart(t, "hello"))}
	req, err := ai.NewRequest("gpt-4o", messages, nil, ai.GenerationOptions{})
	if err != nil {
		t.Fatalf("setup NewRequest error = %v", err)
	}
	messages[0].Role = ""
	assertErrorIs(t, req.Validate(), ai.ErrInvalidRole)
}

func TestNewGenerationOptions_SubstitutesToolChoiceAuto(t *testing.T) {
	opts, err := ai.NewGenerationOptions("", 0, 0, 0, nil, "")
	if err != nil {
		t.Fatalf("NewGenerationOptions zero choice error = %v, want nil", err)
	}
	if opts.ToolChoice() != ai.ToolChoiceAuto {
		t.Errorf("ToolChoice() = %q, want %q", opts.ToolChoice(), ai.ToolChoiceAuto)
	}
}

func TestGenerationOptions_ZeroValueValidates(t *testing.T) {
	var opts ai.GenerationOptions
	if err := opts.Validate(); err != nil {
		t.Errorf("GenerationOptions{}.Validate() = %v, want nil", err)
	}
	if opts.ToolChoice() != ai.ToolChoiceAuto {
		t.Errorf("GenerationOptions{}.ToolChoice() = %q, want %q", opts.ToolChoice(), ai.ToolChoiceAuto)
	}
}

func TestRequest_ValidatesMessagesFirst(t *testing.T) {
	duplicate := mustTool(t, "search", "Search", json.RawMessage(`{"type":"object"}`))
	_, err := ai.NewRequest("gpt-4o", []ai.Message{{}}, []ai.ToolDeclaration{duplicate, duplicate}, ai.GenerationOptions{})
	assertErrorIs(t, err, ai.ErrInvalidRole)
}

func TestRequest_ValidatesToolsAfterMessages(t *testing.T) {
	messages := []ai.Message{mustMsg(ai.RoleUser, "", mustTextPart(t, "hello"))}
	_, err := ai.NewRequest("gpt-4o", messages, []ai.ToolDeclaration{{}}, ai.GenerationOptions{})
	assertErrorIs(t, err, ai.ErrEmptyToolName)
}

func TestRequest_ValidatesOptionsLast(t *testing.T) {
	stops := []string{"STOP"}
	opts := mustOptions(t, "", 0, 0, 0, stops, ai.ToolChoiceRequired)
	stops[0] = ""
	messages := []ai.Message{mustMsg(ai.RoleUser, "", mustTextPart(t, "hello"))}
	_, err := ai.NewRequest("gpt-4o", messages, nil, opts)
	assertErrorIs(t, err, ai.ErrEmptyStopSequence)
}

func TestNewRequest_AllowedCombinations(t *testing.T) {
	tool := mustTool(t, "search", "Search", json.RawMessage(`{"type":"object"}`))
	cases := []struct {
		name  string
		tools []ai.ToolDeclaration
		opts  ai.GenerationOptions
	}{
		{"temperature plus topP", nil, mustOptions(t, "", 0, 1, 0.5, nil, ai.ToolChoiceAuto)},
		{"required with tool", []ai.ToolDeclaration{tool}, mustOptions(t, "", 0, 0, 0, nil, ai.ToolChoiceRequired)},
		{"none with declared tool", []ai.ToolDeclaration{tool}, mustOptions(t, "", 0, 0, 0, nil, ai.ToolChoiceNone)},
		{"auto without tools", nil, mustOptions(t, "", 0, 0, 0, nil, ai.ToolChoiceAuto)},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if _, err := ai.NewRequest("gpt-4o", nil, tc.tools, tc.opts); err != nil {
				t.Errorf("NewRequest allowed combination error = %v, want nil", err)
			}
		})
	}
}

func TestConstants_Values(t *testing.T) {
	if ai.MaxRequestMaxOutputTokens != 1<<17 || ai.MaxRequestSystemInstructionLength != 1<<16 || ai.MaxRequestStopSequences != 8 || ai.MaxRequestStopSequenceLength != 256 {
		t.Errorf("AI-09 request constant values drifted")
	}
}

func TestRequest_NotContentPart(t *testing.T) {
	if method, ok := reflect.TypeOf(ai.Request{}).MethodByName("Kind"); ok {
		t.Errorf("Request has Kind method %s; Request must not implement ContentPart", method.Type)
	}
}

func requestSentinels() []error {
	return []error{
		ai.ErrEmptyModel,
		ai.ErrWhitespaceModel,
		ai.ErrInvalidMaxOutputTokens,
		ai.ErrTemperatureOutOfRange,
		ai.ErrTopPOutOfRange,
		ai.ErrTooManyStopSequences,
		ai.ErrStopSequenceTooLong,
		ai.ErrEmptyStopSequence,
		ai.ErrWhitespaceStopSequence,
		ai.ErrInvalidToolChoice,
		ai.ErrToolChoiceRequiresTools,
		ai.ErrWhitespaceSystemInstruction,
		ai.ErrSystemInstructionTooLong,
	}
}

func assertErrorIs(t *testing.T, got, want error) {
	t.Helper()
	if want == nil {
		if got != nil {
			t.Errorf("error = %v, want nil", got)
		}
		return
	}
	if !errors.Is(got, want) {
		t.Errorf("error = %v, want errors.Is(_, %v)", got, want)
	}
}

func assertNoMethods(t *testing.T, typ reflect.Type, names ...string) {
	t.Helper()
	for _, name := range names {
		if _, ok := typ.MethodByName(name); ok {
			t.Errorf("%s must not expose %s", typ, name)
		}
	}
}

func makeStops(n int) []string {
	stops := make([]string, n)
	for i := range stops {
		stops[i] = "stop"
	}
	return stops
}

func mustOptions(t *testing.T, systemInstruction string, maxOutputTokens int, temperature, topP float64, stopSequences []string, toolChoice ai.ToolChoice) ai.GenerationOptions {
	t.Helper()
	opts, err := ai.NewGenerationOptions(systemInstruction, maxOutputTokens, temperature, topP, stopSequences, toolChoice)
	if err != nil {
		t.Fatalf("NewGenerationOptions setup error = %v", err)
	}
	return opts
}

func mustTool(t *testing.T, name, description string, schema json.RawMessage) ai.ToolDeclaration {
	t.Helper()
	tool, err := ai.NewToolDeclaration(name, description, schema)
	if err != nil {
		t.Fatalf("NewToolDeclaration setup error = %v", err)
	}
	return tool
}

func mustMsg(role ai.Role, id string, parts ...ai.ContentPart) ai.Message {
	return ai.Message{Role: role, ID: id, Content: parts}
}

func mustTextPart(t *testing.T, text string) ai.ContentPart {
	t.Helper()
	part, err := ai.ContentPartFromText(text)
	if err != nil {
		t.Fatalf("ContentPartFromText setup error = %v", err)
	}
	return part
}
