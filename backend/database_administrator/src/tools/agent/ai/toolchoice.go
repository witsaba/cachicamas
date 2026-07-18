package ai

// ToolChoice selects whether the model may, must, or must not invoke a tool.
// Vendor-neutral; maps to OpenAI tool_choice, Anthropic tool_choice, and
// OpenRouter's pass-through semantics.
//
// The zero value (ToolChoice("")) is rejected by Request.Validate.
// AI-09 ships exactly three values; future v1.1 may add a specific-tool
// pinning variant via an additive constant or parallel field.
//
// WARNING: ToolChoice must remain a plain typed-string enum with no
// methods beyond its three constants. Do NOT add UnmarshalText or
// MarshalText without extending FR-10 reflection checks.
type ToolChoice string

const (
	// ToolChoiceAuto leaves the tool-call decision to the provider.
	// Default; zero-value analogue after NewGenerationOptions substitution.
	ToolChoiceAuto ToolChoice = "auto"

	// ToolChoiceNone explicitly forbids tool use.
	ToolChoiceNone ToolChoice = "none"

	// ToolChoiceRequired requires at least one tool call.
	// Rejected by Request.Validate when no tools are declared.
	ToolChoiceRequired ToolChoice = "required"
)
