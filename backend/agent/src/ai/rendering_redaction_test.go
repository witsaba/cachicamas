// The V-FAIL-13 rendering posture, closed over the four carrier types
// that lacked it: Message, Tool, ToolSet and ToolChoice.
//
// Every other payload-bearing exported type in this package defines
// String()/GoString() naming structure only, because fmt's printValue
// skips handleMethods for a value reached through an unexported field and
// falls back to reflection — %v on a bare struct with unexported fields
// prints their contents. Message (which holds the conversation), Tool
// (name, description and schema), ToolSet and ToolChoice (a tool name)
// were the four types without renderings, so fmt.Sprintf("%v", msg)
// reproduced prompt text verbatim. These tests plant sentinels in every
// content-carrying field and assert no fmt verb reproduces one.
package ai_test

import (
	"fmt"
	"strings"
	"testing"

	"github.com/cachicamas/backend/agent/src/ai"
)

// renderingVerbs is every fmt verb family with a distinct rendering path:
// %v and %s go through String, %+v prints field names on reflection
// fallback, %#v goes through GoString.
var renderingVerbs = []string{"%v", "%+v", "%s", "%#v"}

func assertSentinelFreeRendering(t *testing.T, label string, value any, sentinels ...string) {
	t.Helper()
	for _, verb := range renderingVerbs {
		rendered := fmt.Sprintf(verb, value)
		if rendered == "" {
			t.Errorf("fmt.Sprintf(%q, %s) = \"\", want a non-empty structural label", verb, label)
		}
		for _, sentinel := range sentinels {
			if strings.Contains(rendered, sentinel) {
				t.Errorf("fmt.Sprintf(%q, %s) = %q reproduces the planted sentinel %q — content must never reach a rendering (V-FAIL-13)",
					verb, label, rendered, sentinel)
			}
		}
	}
}

func TestMessage_Renderings_NameStructureNeverContent(t *testing.T) {
	t.Parallel()

	const sentinel = "SENTINEL-message-prompt-text-b7e02"
	part, err := ai.NewText(sentinel)
	if err != nil {
		t.Fatalf("ai.NewText() error = %v, want nil", err)
	}
	msg, err := ai.NewMessage(ai.RoleUser, part)
	if err != nil {
		t.Fatalf("ai.NewMessage() error = %v, want nil", err)
	}

	assertSentinelFreeRendering(t, "Message", msg, sentinel)

	if got := msg.String(); !strings.Contains(got, "message(") {
		t.Errorf("Message.String() = %q, want the package's structural label form", got)
	}
}

func TestTool_Renderings_NameStructureNeverContent(t *testing.T) {
	t.Parallel()

	const (
		nameSentinel   = "sentinel_tool_name_b7e02"
		descSentinel   = "SENTINEL-tool-description-b7e02"
		schemaSentinel = "SENTINEL-tool-schema-b7e02"
	)
	tool, err := ai.NewTool(nameSentinel, descSentinel, []byte(`{"type":"object","marker":"`+schemaSentinel+`"}`))
	if err != nil {
		t.Fatalf("ai.NewTool() error = %v, want nil", err)
	}

	// The tool NAME is excluded too: R-AMR-017 already forbids a request
	// rendering from reproducing "a tool's name or schema", and the
	// value-level rendering keeps the same line.
	assertSentinelFreeRendering(t, "Tool", tool, nameSentinel, descSentinel, schemaSentinel)

	set, err := ai.NewToolSet(tool)
	if err != nil {
		t.Fatalf("ai.NewToolSet() error = %v, want nil", err)
	}
	assertSentinelFreeRendering(t, "ToolSet", set, nameSentinel, descSentinel, schemaSentinel)
}

func TestToolChoice_Renderings_NameModeNeverToolName(t *testing.T) {
	t.Parallel()

	const nameSentinel = "sentinel_choice_tool_b7e02"
	named, err := ai.NewNamedToolChoice(nameSentinel)
	if err != nil {
		t.Fatalf("ai.NewNamedToolChoice() error = %v, want nil", err)
	}

	assertSentinelFreeRendering(t, "ToolChoice", named, nameSentinel)

	if got := named.String(); !strings.Contains(got, "toolchoice(") {
		t.Errorf("ToolChoice.String() = %q, want the package's structural label form", got)
	}
}
