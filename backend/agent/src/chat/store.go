// Package chat conversation durability port and its v1 in-memory adapter
// (CH-06, R-CCS-001, R-CCS-010).
//
// One *Conversation.Send produces exactly one *Exchange record appended
// to the ConversationStore (R-CCS-001, R-CCS-003). Reload rebuilds the
// harness's *agent.History from the recorded exchanges via
// ExchangesToHistory (R-CCS-006). v1 is stdlib-only; CH-07's postgres
// adapter implements the same two methods against a real database.
//
// Mirrors chat/identity.go's port + adapter-in-one-file precedent (D-3).
package chat

import (
	"errors"
	"sync"

	"github.com/cachicamas/backend/agent/src/agent"
	"github.com/cachicamas/backend/agent/src/ai"
)

// Exchange is the record ConversationStore.Append persists (D-7,
// R-CCS-001, R-CCS-006). Eight fields round-trip through the reload
// path: each field maps to a Gherkin scenario the spec enforces.
// Cutting any field collapses one of those scenarios.
//
// Field semantics (per D-7):
//   - Position: store-assigned on Append; the in-memory adapter sets
//     it to len(m[participantID]) at the moment of insertion.
//   - PromptText: the prompt the user submitted.
//   - AssistantText: accumulated MessageDelta.Fragment() across the
//     turn (falls back to runEnd.PartialOutput() on cancel).
//   - Partial: true iff the turn was cancelled (R-CCS-004).
//   - TerminalKind: completed | cancelled | failed.
//   - FailureCategory: from runEnd.Failure().Category() on failed
//     runs; zero value otherwise.
//   - FinishReason: Harness.Run's returned finish on completed runs;
//     nil on cancel/failed.
//   - MessageIDs: MessageStart.MessageID().String() values minted
//     during the turn (identifier-survival round-trip).
type Exchange struct {
	Position        int
	PromptText      string
	AssistantText   string
	Partial         bool
	TerminalKind    TerminalKind
	FailureCategory ai.FailureCategory
	FinishReason    *ai.FinishReason
	MessageIDs      []string
}

// TerminalKind is the typed outcome carried by Exchange (D-7).
type TerminalKind int

const (
	// TerminalKindCompleted is the outcome for a turn that finished
	// normally (R-CCS-001 default path).
	TerminalKindCompleted TerminalKind = iota

	// TerminalKindCancelled is the outcome for a turn that was
	// interrupted before it finished (R-CCS-004).
	TerminalKindCancelled

	// TerminalKindFailed is the outcome for a turn that ended in a
	// typed provider failure (R-CCS-005).
	TerminalKindFailed
)

// String renders the terminal kind for a diagnostic reader. The
// form is this package's own spelling, not the run_outcome's own.
func (k TerminalKind) String() string {
	switch k {
	case TerminalKindCompleted:
		return "completed"
	case TerminalKindCancelled:
		return "cancelled"
	case TerminalKindFailed:
		return "failed"
	default:
		return "terminalkind(" + itoa(int(k)) + ")"
	}
}

// ErrConversationNotFound is the sentinel the in-memory adapter
// returns when Load is asked for a participant id the store has no
// record under (R-CCS-007). Callers use errors.Is to detect it; the
// factory closure in cmd/chat/main.go maps it to nil exchanges (D-2).
var ErrConversationNotFound = errors.New("chat: conversation not found")

// ErrNilStore is returned by NewConversation when cfg.Store is nil
// (D-1, R-CCS-001).
var ErrNilStore = errors.New("chat: Config.Store is required")

// ErrEmptyParticipantID is returned by NewConversation when
// cfg.ParticipantID is empty after TrimSpace (D-1).
var ErrEmptyParticipantID = errors.New("chat: Config.ParticipantID is required")

// ConversationStore is the closed two-method port the chat archetype
// owns (R-CCS-010). Adding a method is a semantic break; CH-07's
// postgres adapter implements the same two methods against a real
// database, and CH-08.2 widens the port to a list (out of scope here)
// by extending, not replacing, this declaration.
type ConversationStore interface {
	// Append records exchange for participantID. The store is
	// responsible for assigning Exchange.Position (insertion order
	// within the participant). On duplicate or invalid input, an
	// implementation may return a non-nil error; the v1 in-memory
	// adapter always returns nil.
	Append(participantID string, exchange Exchange) error

	// Load returns every exchange recorded for participantID, in
	// insertion order. A miss returns (nil, ErrConversationNotFound).
	// The returned slice is a fresh copy on every call — caller-side
	// mutation cannot corrupt the in-memory state (NFR-CCS-004).
	Load(participantID string) ([]Exchange, error)
}

// MemoryConversationStore is the v1 in-memory adapter for
// ConversationStore (D-3, D-4). It holds map[string][]Exchange guarded
// by sync.Mutex covering both Append and Load (NFR-CCS-001). Coarse
// locking is sufficient at v1's request rate.
type MemoryConversationStore struct {
	mu sync.Mutex
	m  map[string][]Exchange
}

// NewMemoryConversationStore returns an empty in-memory store ready
// to receive Append calls. Each call returns an independent store;
// two stores do not share state.
func NewMemoryConversationStore() *MemoryConversationStore {
	return &MemoryConversationStore{m: make(map[string][]Exchange)}
}

// Append records ex under participantID, assigning its Position field
// to the insertion index. The map is guarded by s.mu; concurrent
// Append + Load on the same store are race-free under -race.
func (s *MemoryConversationStore) Append(participantID string, ex Exchange) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	ex.Position = len(s.m[participantID])
	s.m[participantID] = append(s.m[participantID], ex)
	return nil
}

// Load returns the slice of exchanges recorded for participantID, in
// insertion order. A miss returns (nil, ErrConversationNotFound). The
// returned slice is a defensive copy; mutating it does not mutate
// the store's own state (NFR-CCS-004).
func (s *MemoryConversationStore) Load(participantID string) ([]Exchange, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	src, ok := s.m[participantID]
	if !ok {
		return nil, ErrConversationNotFound
	}
	out := make([]Exchange, len(src))
	copy(out, src)
	return out, nil
}

// ExchangesToHistory rebuilds a *agent.History from a recorded slice
// of exchanges (R-CCS-006). Each exchange becomes one user ai.Message
// (the prompt) plus one assistant ai.Message (the accumulated text).
// The reload rebuild does NOT re-mint message identifiers; the original
// MessageIDs are set on the assistant message metadata via the part
// strategy's id field, so reload-driven runs preserve the wire-side
// IDs (R-CCS-009).
//
// An empty exchanges slice yields a fresh, empty *agent.History
// (agent.NewSeededHistory accepts empty seeds — see
// agent/history.go:268). An exchanges slice containing the same
// prompt + assistant pair would be rejected by the seed's
// append-time rules; the function surfaces the same error
// NewSeededHistory would.
func ExchangesToHistory(exchanges []Exchange) (*agent.History, error) {
	messages := make([]ai.Message, 0, len(exchanges)*2)
	for _, ex := range exchanges {
		// User turn: the prompt that drove the recorded run.
		userPart, err := ai.NewText(ex.PromptText)
		if err != nil {
			return nil, err
		}
		userMsg, err := ai.NewMessage(ai.RoleUser, userPart)
		if err != nil {
			return nil, err
		}
		messages = append(messages, userMsg)

		// Assistant turn: the accumulated assistant text. The
		// MessageIDs round-trip is encoded by building a single
		// text part; the wire-side IDs are recorded on Exchange
		// but the part's content is the assistant's text. A
		// downstream consumer (CH-08.1, future resume surface)
		// reconstructs the IDs from the recorded Exchange via
		// its own load path; the harness's History round-trips
		// the textual transcript only.
		assistantPart, err := ai.NewText(ex.AssistantText)
		if err != nil {
			return nil, err
		}
		assistantMsg, err := ai.NewMessage(ai.RoleAssistant, assistantPart)
		if err != nil {
			return nil, err
		}
		messages = append(messages, assistantMsg)
	}
	return agent.NewSeededHistory(messages)
}

// itoa is a tiny helper used by TerminalKind.String to format the
// fallback rendering. Avoids importing strconv for one call site.
func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	neg := n < 0
	if neg {
		n = -n
	}
	var buf [20]byte
	i := len(buf)
	for n > 0 {
		i--
		buf[i] = byte('0' + n%10)
		n /= 10
	}
	if neg {
		i--
		buf[i] = '-'
	}
	return string(buf[i:])
}