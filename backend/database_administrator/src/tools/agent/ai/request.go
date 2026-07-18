package ai

import (
	"errors"
	"math"
)

// ErrEmptyModel is returned when a Request model identifier is empty.
var ErrEmptyModel = errors.New("ai: empty model identifier")

// ErrWhitespaceModel is returned when a model identifier is whitespace-only.
var ErrWhitespaceModel = errors.New("ai: whitespace-only model identifier")

// ErrInvalidMaxOutputTokens is returned for a negative or oversized token limit.
var ErrInvalidMaxOutputTokens = errors.New("ai: invalid MaxOutputTokens value")

// ErrTemperatureOutOfRange is returned for a non-finite value or one outside [0, 2].
var ErrTemperatureOutOfRange = errors.New("ai: Temperature outside [0.0, 2.0]")

// ErrTopPOutOfRange is returned for a non-finite value or one outside [0, 1].
var ErrTopPOutOfRange = errors.New("ai: TopP outside [0.0, 1.0]")

// ErrTooManyStopSequences is returned when the request has more than eight stops.
var ErrTooManyStopSequences = errors.New("ai: too many stop sequences")

// ErrStopSequenceTooLong is returned when a stop exceeds the byte limit.
var ErrStopSequenceTooLong = errors.New("ai: stop sequence exceeds MaxRequestStopSequenceLength bytes")

// ErrEmptyStopSequence is returned when a stop sequence is empty.
var ErrEmptyStopSequence = errors.New("ai: empty stop sequence")

// ErrWhitespaceStopSequence is returned when a stop sequence is whitespace-only.
var ErrWhitespaceStopSequence = errors.New("ai: whitespace-only stop sequence")

// ErrInvalidToolChoice is returned when ToolChoice is not canonical.
var ErrInvalidToolChoice = errors.New("ai: invalid ToolChoice value")

// ErrToolChoiceRequiresTools is returned when Required has no declared tools.
var ErrToolChoiceRequiresTools = errors.New("ai: ToolChoiceRequired requires at least one tool")

// ErrWhitespaceSystemInstruction is returned for a whitespace-only instruction.
var ErrWhitespaceSystemInstruction = errors.New("ai: whitespace-only system instruction")

// ErrSystemInstructionTooLong is returned when an instruction exceeds the byte limit.
var ErrSystemInstructionTooLong = errors.New("ai: system instruction exceeds MaxRequestSystemInstructionLength bytes")

// MaxRequestMaxOutputTokens is the inclusive output-token ceiling (131,072).
const MaxRequestMaxOutputTokens = 1 << 17

// MaxRequestSystemInstructionLength is the inclusive instruction byte ceiling (64 KiB).
const MaxRequestSystemInstructionLength = 1 << 16

// MaxRequestStopSequences is the inclusive stop-sequence count ceiling.
const MaxRequestStopSequences = 8

// MaxRequestStopSequenceLength is the inclusive byte ceiling for one stop sequence.
const MaxRequestStopSequenceLength = 256

// GenerationOptions carries the v1 vendor-neutral generation parameters.
// The zero value uses provider defaults. Combining Temperature and TopP is
// discouraged but not rejected. Values are immutable; reconstruct to extend.
type GenerationOptions struct {
	systemInstruction string
	maxOutputTokens   int
	temperature       float64
	topP              float64
	stopSequences     []string
	toolChoice        ToolChoice
}

// NewGenerationOptions constructs generation options in field order and
// returns the zero value with the first validation error.
func NewGenerationOptions(systemInstruction string, maxOutputTokens int, temperature, topP float64, stopSequences []string, toolChoice ToolChoice) (GenerationOptions, error) {
	if toolChoice == "" {
		toolChoice = ToolChoiceAuto
	}
	opts := GenerationOptions{
		systemInstruction: systemInstruction,
		maxOutputTokens:   maxOutputTokens,
		temperature:       temperature,
		topP:              topP,
		stopSequences:     stopSequences,
		toolChoice:        toolChoice,
	}
	if err := opts.Validate(); err != nil {
		return GenerationOptions{}, err
	}
	return opts, nil
}

// Validate reports whether every generation option satisfies its field rule.
// Checks run in field order and stop at the first failure.
func (o GenerationOptions) Validate() error {
	if o.systemInstruction != "" && isAllWhitespace(o.systemInstruction) {
		return ErrWhitespaceSystemInstruction
	}
	if len(o.systemInstruction) > MaxRequestSystemInstructionLength {
		return ErrSystemInstructionTooLong
	}
	if o.maxOutputTokens != 0 && (o.maxOutputTokens < 0 || o.maxOutputTokens > MaxRequestMaxOutputTokens) {
		return ErrInvalidMaxOutputTokens
	}
	if math.IsNaN(o.temperature) || math.IsInf(o.temperature, 0) || o.temperature < 0 || o.temperature > 2 {
		return ErrTemperatureOutOfRange
	}
	if math.IsNaN(o.topP) || math.IsInf(o.topP, 0) || o.topP < 0 || o.topP > 1 {
		return ErrTopPOutOfRange
	}
	if len(o.stopSequences) > MaxRequestStopSequences {
		return ErrTooManyStopSequences
	}
	for _, stop := range o.stopSequences {
		if stop == "" {
			return ErrEmptyStopSequence
		}
		if isAllWhitespace(stop) {
			return ErrWhitespaceStopSequence
		}
		if len(stop) > MaxRequestStopSequenceLength {
			return ErrStopSequenceTooLong
		}
	}
	switch o.ToolChoice() {
	case ToolChoiceAuto, ToolChoiceNone, ToolChoiceRequired:
		return nil
	default:
		return ErrInvalidToolChoice
	}
}

// SystemInstruction returns the verbatim system instruction.
func (o GenerationOptions) SystemInstruction() string { return o.systemInstruction }

// MaxOutputTokens returns the requested output-token ceiling; zero is unset.
func (o GenerationOptions) MaxOutputTokens() int { return o.maxOutputTokens }

// Temperature returns the requested sampling temperature; zero is unset.
func (o GenerationOptions) Temperature() float64 { return o.temperature }

// TopP returns the requested nucleus-sampling probability; zero is unset.
func (o GenerationOptions) TopP() float64 { return o.topP }

// StopSequences returns the verbatim stop-sequence slice; nil is unset.
func (o GenerationOptions) StopSequences() []string { return o.stopSequences }

// ToolChoice returns the tool policy, defaulting the zero value to Auto.
func (o GenerationOptions) ToolChoice() ToolChoice {
	if o.toolChoice == "" {
		return ToolChoiceAuto
	}
	return o.toolChoice
}

// Request is the provider-neutral value that Layer 2 hands to Layer 1 to
// start a turn. It bundles model, history, tools, and generation options.
//
// A Request can only be built via NewRequest, which validates every component.
// Layer 2 MUST treat any returned error as a hard failure before network I/O.
// Request values are immutable. To extend a request, build a new Request via
// NewRequest. Validation costs O(len(messages) + len(tools)).
type Request struct {
	model    string
	messages []Message
	tools    []ToolDeclaration
	options  GenerationOptions
}

// NewRequest constructs a Request after validating model, messages, tools,
// then options. It returns a zero Request with the first validation error.
func NewRequest(model string, messages []Message, tools []ToolDeclaration, opts GenerationOptions) (Request, error) {
	r := Request{model: model, messages: messages, tools: tools, options: opts}
	if err := r.Validate(); err != nil {
		return Request{}, err
	}
	return r, nil
}

// Validate re-runs validation over the stored fields in construction order.
// Validation costs O(len(messages) + len(tools)).
func (r Request) Validate() error {
	if r.model == "" {
		return ErrEmptyModel
	}
	if isAllWhitespace(r.model) {
		return ErrWhitespaceModel
	}
	for _, message := range r.messages {
		if err := message.Validate(); err != nil {
			return err
		}
	}
	if err := ValidateTools(r.tools); err != nil {
		return err
	}
	if err := r.options.Validate(); err != nil {
		return err
	}
	if r.options.ToolChoice() == ToolChoiceRequired && len(r.tools) == 0 {
		return ErrToolChoiceRequiresTools
	}
	return nil
}

// Model returns the verbatim model identifier.
func (r Request) Model() string { return r.model }

// Messages returns the verbatim message slice.
func (r Request) Messages() []Message { return r.messages }

// Tools returns the verbatim tool-declaration slice.
func (r Request) Tools() []ToolDeclaration { return r.tools }

// Options returns the verbatim generation options.
func (r Request) Options() GenerationOptions { return r.options }
