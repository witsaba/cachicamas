// Package chat_test — conversation_version_test.go is the TDD contract
// for the chat-side wiring of the archetype.VersionTracker
// (REQ-CCVP-001/002/003, design AD-3). The tracker itself is tested
// in src/archetype/version_test.go; this file tests the chat
// Conversation's integration.
//
// Four scenarios:
//
//	- Test_Conversation_NewConversation_SetsInitialSystemFromConfig
//	- Test_Conversation_ReloadAssistantConfig_VersionMismatch_UpdatesSystem
//	- Test_Conversation_ReloadAssistantConfig_VersionMatch_NoUpdate
//	- Test_Conversation_ReloadAssistantConfig_NoLoader_NoOp
//
// RED at T-07: chat.Config has no AssistantConfigLoader field;
// chat.NewConversation does not consult a Loader; chat.Conversation
// has no ReloadAssistantConfig method; chat.Conversation has no
// SystemPromptForTest test helper. T-08 adds all four.
package chat_test

import (
	"context"
	"database/sql"
	"sync"
	"testing"

	"github.com/cachicamas/backend/agent/src/agent"
	"github.com/cachicamas/backend/agent/src/agenttest"
	"github.com/cachicamas/backend/agent/src/ai"
	"github.com/cachicamas/backend/agent/src/archetype"
	"github.com/cachicamas/backend/agent/src/chat"
)

// fakeVersionedLoader satisfies archetype.Loader for the test.
type fakeVersionedLoader struct {
	mu     sync.Mutex
	result archetype.ArchetypeConfig
	found  bool
	err    error
}

func (f *fakeVersionedLoader) LoadByKindAndOrg(_ context.Context, _ archetype.ArchetypeKind, _ string) (archetype.ArchetypeConfig, bool, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.result, f.found, f.err
}

func (f *fakeVersionedLoader) WithTx(_ *sql.Tx) archetype.Loader { return f }

func (f *fakeVersionedLoader) withCurrent(cfg archetype.ArchetypeConfig, found bool) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.result = cfg
	f.found = found
}

// scriptedProvider builds a deterministic scripted stream so the
// Conversation can drive one turn. Mirrors conversation_test.go's
// helper but inlined here so the test file stays self-contained.
func scriptedProvider(t *testing.T) *agenttest.Provider {
	t.Helper()
	start, err := ai.NewTextBlockStart(1)
	if err != nil {
		t.Fatalf("ai.NewTextBlockStart: %v", err)
	}
	delta, err := ai.NewTextDelta(1, "hi")
	if err != nil {
		t.Fatalf("ai.NewTextDelta: %v", err)
	}
	end, err := ai.NewTextBlockEnd(1)
	if err != nil {
		t.Fatalf("ai.NewTextBlockEnd: %v", err)
	}
	completion, err := ai.NewCompletion(ai.FinishReasonStop, ai.Usage{})
	if err != nil {
		t.Fatalf("ai.NewCompletion: %v", err)
	}
	script := agenttest.Script{
		Steps: []agenttest.Step{
			agenttest.Emit(start),
			agenttest.Emit(delta),
			agenttest.Emit(end),
			agenttest.Emit(completion),
		},
	}
	return agenttest.NewProvider(script)
}

// Test_Conversation_NewConversation_SetsInitialSystemFromConfig —
// REQ-CCVP-001. Given the Loader returns prompt="initial prompt" at
// version=3, when the Conversation is constructed, then the harness
// system prompt is "initial prompt" and the recorded version is 3.
//
// RED at T-07.
func Test_Conversation_NewConversation_SetsInitialSystemFromConfig(t *testing.T) {
	t.Parallel()

	loader := &fakeVersionedLoader{
		result: archetype.ArchetypeConfig{
			Kind:           archetype.KindChat,
			OrgID:          "test-conv",
			SystemPrompt:   "initial prompt",
			ToolAllowlist:  []string{"current_time"},
			DeferToolNames: []string{"summarize_conversation"},
			Version:        3,
		},
		found: true,
	}

	conv, err := chat.NewConversation(chat.Config{
		Provider:             scriptedProvider(t),
		Store:                chat.NewMemoryConversationStore(),
		ParticipantID:        "test-conv",
		ToolSource:           chat.FromAgentRegistry(agent.NewMapRegistry(nil)),
		PermissionPolicy:     chat.NewDefaultPermissionPolicy(nil),
		AssistantConfigLoader: loader,
	})
	if err != nil {
		t.Fatalf("chat.NewConversation: %v", err)
	}

	if got := conv.LoadedAssistantConfigVersion(); got != 3 {
		t.Errorf("LoadedAssistantConfigVersion = %d, want 3", got)
	}
	if got := conv.SystemPromptForTest(); got != "initial prompt" {
		t.Errorf("SystemPromptForTest = %q, want %q", got, "initial prompt")
	}
}

// Test_Conversation_ReloadAssistantConfig_VersionMismatch_UpdatesSystem
// — REQ-CCVP-002 / Scenario "version mismatch rebuilds the prompt on
// next Send". Given the Conversation was constructed at version=3
// and the Loader now reports version=4 with a new prompt, when
// ReloadAssistantConfig runs, then the harness system prompt is
// replaced and the recorded version bumps to 4.
//
// RED at T-07.
func Test_Conversation_ReloadAssistantConfig_VersionMismatch_UpdatesSystem(t *testing.T) {
	t.Parallel()

	loader := &fakeVersionedLoader{
		result: archetype.ArchetypeConfig{
			Kind:         archetype.KindChat,
			OrgID:        "test-conv",
			SystemPrompt: "initial prompt",
			Version:      3,
		},
		found: true,
	}
	conv, err := chat.NewConversation(chat.Config{
		Provider:             scriptedProvider(t),
		Store:                chat.NewMemoryConversationStore(),
		ParticipantID:        "test-conv",
		ToolSource:           chat.FromAgentRegistry(agent.NewMapRegistry(nil)),
		PermissionPolicy:     chat.NewDefaultPermissionPolicy(nil),
		AssistantConfigLoader: loader,
	})
	if err != nil {
		t.Fatalf("chat.NewConversation: %v", err)
	}

	loader.withCurrent(archetype.ArchetypeConfig{
		Kind:         archetype.KindChat,
		OrgID:        "test-conv",
		SystemPrompt: "second prompt after config change",
		Version:      4,
	}, true)

	if err := conv.ReloadAssistantConfig(context.Background()); err != nil {
		t.Fatalf("ReloadAssistantConfig: %v", err)
	}

	if got := conv.LoadedAssistantConfigVersion(); got != 4 {
		t.Errorf("LoadedAssistantConfigVersion = %d, want 4", got)
	}
	if got := conv.SystemPromptForTest(); got != "second prompt after config change" {
		t.Errorf("SystemPromptForTest = %q, want %q", got, "second prompt after config change")
	}
}

// Test_Conversation_ReloadAssistantConfig_VersionMatch_NoUpdate —
// REQ-CCVP-002 / Scenario "version match does not re-read". Given the
// Conversation holds version=3 and the Loader still reports version=3,
// when ReloadAssistantConfig runs, then the system prompt is unchanged.
//
// RED at T-07.
func Test_Conversation_ReloadAssistantConfig_VersionMatch_NoUpdate(t *testing.T) {
	t.Parallel()

	loader := &fakeVersionedLoader{
		result: archetype.ArchetypeConfig{
			Kind:         archetype.KindChat,
			OrgID:        "test-conv",
			SystemPrompt: "initial prompt",
			Version:      3,
		},
		found: true,
	}
	conv, err := chat.NewConversation(chat.Config{
		Provider:             scriptedProvider(t),
		Store:                chat.NewMemoryConversationStore(),
		ParticipantID:        "test-conv",
		ToolSource:           chat.FromAgentRegistry(agent.NewMapRegistry(nil)),
		PermissionPolicy:     chat.NewDefaultPermissionPolicy(nil),
		AssistantConfigLoader: loader,
	})
	if err != nil {
		t.Fatalf("chat.NewConversation: %v", err)
	}
	if err := conv.ReloadAssistantConfig(context.Background()); err != nil {
		t.Fatalf("ReloadAssistantConfig: %v", err)
	}
	if got := conv.SystemPromptForTest(); got != "initial prompt" {
		t.Errorf("SystemPromptForTest = %q, want %q (no update on match)", got, "initial prompt")
	}
	if got := conv.LoadedAssistantConfigVersion(); got != 3 {
		t.Errorf("LoadedAssistantConfigVersion = %d, want 3", got)
	}
}

// Test_Conversation_ReloadAssistantConfig_NoLoader_NoOp — defence in
// depth: a Conversation constructed without an AssistantConfigLoader
// is inert. Reload returns nil and the system prompt stays at the
// chat v1 default (`chat.SystemPrompt`).
//
// RED at T-07.
func Test_Conversation_ReloadAssistantConfig_NoLoader_NoOp(t *testing.T) {
	t.Parallel()

	conv, err := chat.NewConversation(chat.Config{
		Provider:         scriptedProvider(t),
		Store:            chat.NewMemoryConversationStore(),
		ParticipantID:    "test-conv",
		ToolSource:       chat.FromAgentRegistry(agent.NewMapRegistry(nil)),
		PermissionPolicy: chat.NewDefaultPermissionPolicy(nil),
		// no AssistantConfigLoader
	})
	if err != nil {
		t.Fatalf("chat.NewConversation: %v", err)
	}
	if err := conv.ReloadAssistantConfig(context.Background()); err != nil {
		t.Errorf("ReloadAssistantConfig returned err=%v, want nil", err)
	}
	if got := conv.LoadedAssistantConfigVersion(); got != 0 {
		t.Errorf("LoadedAssistantConfigVersion = %d, want 0 (no loader)", got)
	}
	if got := conv.SystemPromptForTest(); got != chat.SystemPrompt {
		t.Errorf("SystemPromptForTest = %q, want the chat v1 default %q", got, chat.SystemPrompt)
	}
}
