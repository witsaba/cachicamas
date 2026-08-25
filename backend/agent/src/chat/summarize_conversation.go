// CH-10.1 — chat.SummarizeConversationTool (D-2 G2, R-CPM-001,
// R-CPM-006 companion: the mutation target for ConversationStore.
// UpdateSummary).
//
// The second chat-owned tool (CH-09's CurrentTimeTool is the first).
// EffectClass is EffectClassMutate (CH-09's CurrentTimeTool is
// EffectClassRead; this is the first mutating-class chat tool). The
// mutation target is the `summary` column on `chat_conversations` —
// the 4th ConversationStore port method (R-CCS-017) writes it.
//
// # Construction
//
// NewSummarizeConversationTool(store, participantID) requires a
// non-nil store AND a non-empty participantID. A nil store is a
// composition-time defect that surfaces immediately (panic) so the
// misconfiguration is caught at wiring time, not at first call.
// The composition root constructs one tool per participant because
// the participantID is the UpdateSummary key — sharing one tool
// across participants would need a thread-local key.
//
// # Args schema
//
// `{}` (empty object). The summarizer generates a placeholder
// summary in v1 (the v1 wiring is the seam; a future milestone
// derives content from prompt + assistant text). Non-empty args
// or malformed JSON yields ToolOutcomeResultFailure with a typed
// message — never silently ignored (S-CTT-102).
//
// # PolicySlot
//
// The chat SummarizeConversationTool does NOT consult the
// PolicySlot (chat-owned state, not Layer 2). The participant ID
// is captured at construction time, so the tool's Run is
// self-contained — it does not need to read the per-call seam.

package chat

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/cachicamas/backend/agent/src/agent"
)

// SummarizeConversationTool is the chat archetype's second tool —
// the first mutating-class tool (D-2 G2, R-CPM-001, S-CTT-101..104).
// EffectClass is EffectClassMutate. Writes the conversation summary
// via ConversationStore.UpdateSummary (the 4th port method, R-CCS-017,
// landed in T-05a).
type SummarizeConversationTool struct {
	store         ConversationStore
	participantID string
	now           func() time.Time
}

// NewSummarizeConversationTool constructs the tool with the given
// store + participantID. Both are required — a nil store or empty
// participantID surfaces as a panic at wiring time so a
// misconfigured composition root fails fast, not at first call.
//
// `now` is injectable so tests can drive a fixed clock; production
// callers pass time.Now. Mirrors NewCurrentTimeTool's NowFunc
// pattern (D-2 precedent).
func NewSummarizeConversationTool(store ConversationStore, participantID string) *SummarizeConversationTool {
	if store == nil {
		panic("chat: NewSummarizeConversationTool requires a non-nil ConversationStore")
	}
	if participantID == "" {
		panic("chat: NewSummarizeConversationTool requires a non-empty participantID")
	}
	return &SummarizeConversationTool{
		store:         store,
		participantID: participantID,
		now:           time.Now,
	}
}

// Name returns the tool's identity — the literal "summarize_conversation"
// (R-CPM-001, S-CTT-104).
func (t *SummarizeConversationTool) Name() string { return "summarize_conversation" }

// EffectClass returns EffectClassMutate (S-CTT-104). The
// summarizer writes to chat_conversations.summary — a state
// mutation, not a read.
func (t *SummarizeConversationTool) EffectClass() agent.EffectClass {
	return agent.EffectClassMutating
}

// Run generates a placeholder summary and writes it via
// store.UpdateSummary (S-CTT-101).
//
// Args schema is `{}`; non-empty or malformed JSON yields
// ToolOutcomeResultFailure with a typed message (S-CTT-102).
//
// On the happy path, Result.Content is the placeholder summary text
// and Outcome is ToolOutcomeSuccess. The scheduler fills CallID;
// the tool itself leaves it empty.
func (t *SummarizeConversationTool) Run(_ context.Context, args []byte, _ agent.PolicySlot) (agent.Result, error) {
	// Empty bytes is a documented synonym for "{}" — same convention
	// as CurrentTimeTool (D-2 / S-CTT-001 precedent).
	if len(args) == 0 {
		summary := t.placeholder()
		if err := t.store.UpdateSummary(t.participantID, summary); err != nil {
			return agent.Result{
				Outcome: agent.ToolOutcomeResultFailure,
				Content: []byte(fmt.Sprintf("summarize_conversation: store.UpdateSummary: %v", err)),
			}, nil
		}
		return agent.Result{
			Outcome: agent.ToolOutcomeSuccess,
			Content: []byte(summary),
		}, nil
	}

	// Validate that args is the JSON empty object `{}`. Anything
	// else is a typed refusal — the tool refuses unknown args
	// rather than silently ignoring them (S-CTT-102).
	var probe map[string]json.RawMessage
	if uerr := json.Unmarshal(args, &probe); uerr != nil {
		return agent.Result{
			Outcome: agent.ToolOutcomeResultFailure,
			Content: []byte(fmt.Sprintf("summarize_conversation: args must be a JSON object, got: %v", uerr)),
		}, nil
	}
	if len(probe) != 0 {
		keys := make([]string, 0, len(probe))
		for k := range probe {
			keys = append(keys, k)
		}
		return agent.Result{
			Outcome: agent.ToolOutcomeResultFailure,
			Content: []byte(fmt.Sprintf("summarize_conversation: unexpected args %v (schema is {})", keys)),
		}, nil
	}

	summary := t.placeholder()
	if err := t.store.UpdateSummary(t.participantID, summary); err != nil {
		return agent.Result{
			Outcome: agent.ToolOutcomeResultFailure,
			Content: []byte(fmt.Sprintf("summarize_conversation: store.UpdateSummary: %v", err)),
		}, nil
	}
	return agent.Result{
		Outcome: agent.ToolOutcomeSuccess,
		Content: []byte(summary),
	}, nil
}

// placeholder generates a placeholder summary text. v1 does NOT
// derive content from the conversation; the seam is the product,
// the placeholder is a stand-in. A future milestone replaces this
// with content derived from prompt + assistant text.
//
// The placeholder is a non-empty deterministic string so a future
// derivation can replace it without breaking callers that depend
// on the schema (S-CTT-101: Result.Content is non-empty on success).
func (t *SummarizeConversationTool) placeholder() string {
	return fmt.Sprintf("conversation summarized at %s", t.now().Format(time.RFC3339))
}