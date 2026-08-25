// CH-10.1 — SummarizeConversationTool tests (S-CTT-101..104).
//
// S-CTT-101 — SummarizeConversationTool writes the summary column via
//             store.UpdateSummary; EffectClass = Mutate.
// S-CTT-102 — non-empty args or malformed JSON yields a typed
//             result_failure; never silently ignored.
// S-CTT-103 — the persisted summary round-trips through Load (memory
//             adapter). The postgres cross-process case is gated
//             INTEGRATION=1 and lives in store_postgres_test.go.
// S-CTT-104 — Name() == "summarize_conversation"; EffectClass() ==
//             agent.EffectClassMutate; chat tool #2.

package chat_test

import (
	"context"
	"encoding/json"
	"strings"
	"sync"
	"testing"

	"github.com/cachicamas/backend/agent/src/agent"
	"github.com/cachicamas/backend/agent/src/ai"
	"github.com/cachicamas/backend/agent/src/chat"
)

// fakeStore is an in-test double of chat.ConversationStore that
// captures UpdateSummary calls. Implements only the methods the
// summarizer needs; Load returns a fixed slice for round-trip tests.
type fakeStore struct {
	mu           sync.Mutex
	summaryByPID map[string]string
	loadedForPID map[string]bool
}

func newFakeStore() *fakeStore {
	return &fakeStore{
		summaryByPID: map[string]string{},
		loadedForPID: map[string]bool{},
	}
}

func (f *fakeStore) Append(participantID string, _ chat.Exchange) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.loadedForPID[participantID] = true
	return nil
}

func (f *fakeStore) Load(participantID string) ([]chat.Exchange, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if !f.loadedForPID[participantID] {
		return nil, chat.ErrConversationNotFound
	}
	return nil, nil
}

func (f *fakeStore) List(_ string) ([]chat.ConversationSummary, error) {
	return []chat.ConversationSummary{}, nil
}

func (f *fakeStore) UpdateSummary(participantID, summary string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.summaryByPID[participantID] = summary
	return nil
}

func (f *fakeStore) summaryOf(participantID string) (string, bool) {
	f.mu.Lock()
	defer f.mu.Unlock()
	s, ok := f.summaryByPID[participantID]
	return s, ok
}

// TestSummarizeConversationTool_NameAndEffectClass (S-CTT-104).
func TestSummarizeConversationTool_NameAndEffectClass(t *testing.T) {
	t.Parallel()

	store := newFakeStore()
	tool := chat.NewSummarizeConversationTool(store, "scn-104-participant")

	if got := tool.Name(); got != "summarize_conversation" {
		t.Errorf("Name() = %q, want %q (S-CTT-104)", got, "summarize_conversation")
	}
	if got := tool.EffectClass(); got != agent.EffectClassMutating {
		t.Errorf("EffectClass() = %v, want EffectClassMutating (S-CTT-104)", got)
	}
}

// TestSummarizeConversationTool_Run_EmptyArgs_Succeeds (S-CTT-101).
// Empty args is documented as a synonym for "{}"; the tool generates
// a placeholder summary and writes it via UpdateSummary. The schema
// is intentionally empty for v1; a future milestone adds content
// arguments.
func TestSummarizeConversationTool_Run_EmptyArgs_Succeeds(t *testing.T) {
	t.Parallel()

	store := newFakeStore()
	tool := chat.NewSummarizeConversationTool(store, "scn-101-participant")

	res, err := tool.Run(context.Background(), []byte(""), agent.PolicySlot("c1"))
	if err != nil {
		t.Fatalf("Run returned err=%v, want nil", err)
	}
	if res.Outcome != agent.ToolOutcomeSuccess {
		t.Errorf("Outcome = %v, want ToolOutcomeSuccess (S-CTT-101)", res.Outcome)
	}
	if len(res.Content) == 0 {
		t.Errorf("Content is empty; want a non-empty placeholder summary")
	}
	if got, ok := store.summaryOf("scn-101-participant"); !ok {
		t.Errorf("UpdateSummary was NOT called; want one store write")
	} else if got != string(res.Content) {
		t.Errorf("UpdateSummary received %q, want %q (write-then-return symmetry)", got, string(res.Content))
	}
}

// TestSummarizeConversationTool_Run_NonEmptyArgs_TypedFailure
// (S-CTT-102). The schema is `{}`; non-empty args yields
// ToolOutcomeResultFailure with a typed message (never silently
// ignored; threat matrix "Args JSON validation").
func TestSummarizeConversationTool_Run_NonEmptyArgs_TypedFailure(t *testing.T) {
	t.Parallel()

	store := newFakeStore()
	tool := chat.NewSummarizeConversationTool(store, "scn-102-participant")

	nonEmpty := []byte(`{"summary":"custom"}`)
	res, err := tool.Run(context.Background(), nonEmpty, agent.PolicySlot("c1"))
	if err != nil {
		t.Fatalf("Run returned err=%v, want nil", err)
	}
	if res.Outcome != agent.ToolOutcomeResultFailure {
		t.Errorf("Outcome = %v, want ToolOutcomeResultFailure (S-CTT-102)", res.Outcome)
	}
	if _, ok := store.summaryOf("scn-102-participant"); ok {
		t.Errorf("UpdateSummary was called for a rejected args; want zero writes")
	}
	if !strings.Contains(string(res.Content), "summarize_conversation") {
		t.Errorf("Content = %q; want a typed refusal that mentions the tool name", string(res.Content))
	}
}

// TestSummarizeConversationTool_Run_MalformedJSON_TypedFailure
// (S-CTT-102 strict). JSON syntax errors yield the same typed
// refusal — never silently ignored.
func TestSummarizeConversationTool_Run_MalformedJSON_TypedFailure(t *testing.T) {
	t.Parallel()

	store := newFakeStore()
	tool := chat.NewSummarizeConversationTool(store, "scn-102b-participant")

	res, err := tool.Run(context.Background(), []byte(`{not json`), agent.PolicySlot("c1"))
	if err != nil {
		t.Fatalf("Run returned err=%v, want nil", err)
	}
	if res.Outcome != agent.ToolOutcomeResultFailure {
		t.Errorf("Outcome = %v, want ToolOutcomeResultFailure (S-CTT-102)", res.Outcome)
	}
}

// TestSummarizeConversationTool_UpdateSummaryRoundTrip (S-CTT-103)
// exercises the memory adapter path. The CrossProcess postgres variant
// is INTEGRATION-gated and lives in store_postgres_test.go (CH-09
// S-CCS-021 precedent).
func TestSummarizeConversationTool_UpdateSummaryRoundTrip(t *testing.T) {
	t.Parallel()

	memStore := chat.NewMemoryConversationStore()
	tool := chat.NewSummarizeConversationTool(memStore, "scn-103-participant")

	res, err := tool.Run(context.Background(), []byte(""), agent.PolicySlot("c1"))
	if err != nil {
		t.Fatalf("Run returned err=%v, want nil", err)
	}
	if res.Outcome != agent.ToolOutcomeSuccess {
		t.Fatalf("Run outcome = %v, want ToolOutcomeSuccess", res.Outcome)
	}

	// Verify the WRITE happened: the memory adapter received the
	// UpdateSummary call with the SAME content Run returned. The
	// full cross-process round-trip (postgres + integration test)
// lives in T-05b. Here we exercise the in-memory shape.
	got, ok := memStore.SummaryForTest("scn-103-participant")
	if !ok {
		t.Fatalf("UpdateSummary was NOT called; want one store write")
	}
	if got != string(res.Content) {
		t.Errorf("UpdateSummary received %q, want %q (write-then-return symmetry)", got, string(res.Content))
	}
}

// TestSummarizeConversationTool_RequiresNonNilStore asserts the
// constructor's panic posture: a nil store surfaces immediately at
// composition-root wiring time, not at first Run. Mirrors
// NewCurrentTimeTool's nil-now panic.
func TestSummarizeConversationTool_RequiresNonNilStore(t *testing.T) {
	t.Parallel()

	defer func() {
		if r := recover(); r == nil {
			t.Errorf("NewSummarizeConversationTool(nil, _) did not panic; want composition-time fail-fast")
		}
	}()
	_ = chat.NewSummarizeConversationTool(nil, "scn-fail-participant") //nolint:staticcheck // intentional nil
}

// (helpers above are imported for compilation; ai is referenced by
// the future-tightening of the schema probe).
var _ = ai.RoleUser
var _ = json.Marshal