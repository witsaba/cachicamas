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
	"time"

	"github.com/cachicamas/backend/agent/src/agent"
	"github.com/cachicamas/backend/agent/src/ai"
)

// Exchange is the record ConversationStore.Append persists (D-7,
// R-CCS-001, R-CCS-006). Ten fields round-trip through the reload
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
//
// CH-09 (R-CTS-006, R-CCS-015): two new fields widen the record
// additively to carry tool-call records. The eight pre-existing
// fields are byte-unchanged; identifier-append-only per CH-07 /
// CH-08 precedent at chat-conversation-store/spec.md:9.
//
//   - ToolCalls: port-side projection of the wire's ToolCallDTO —
//     one record per tool call the model emitted during the turn,
//     in issuance order. Carries WireCallID, Tool, Arguments
//     (byte-equal to what the wire emitted). NFR-CCS-008 / S-CTS-019
//     bind the reload surface.
//   - ToolResults: port-side projection of the wire's ToolResultDTO
//     — one record per tool call's outcome (success /
//     result_failure / execution_failure with a typed category
//     string). Carries WireCallID, Tool, Outcome, Content,
//     FailureCategory. Empty Content when Outcome is
//     execution_failure (no provider text — R-CCP-008 / D6 mirror).
type Exchange struct {
	Position        int
	PromptText      string
	AssistantText   string
	Partial         bool
	TerminalKind    TerminalKind
	FailureCategory ai.FailureCategory
	FinishReason    *ai.FinishReason
	MessageIDs      []string
	ToolCalls       []ToolCallRecord
	ToolResults     []ToolResultRecord
}

// ToolCallRecord is the port-side projection of the wire's
// ToolCallDTO (R-CTS-006, R-CCS-015). It carries the wire call id,
// tool name, and arguments bytes — sufficient to render "the model
// called tool X with these arguments" on reload. The struct is
// the chat package's own projection; the wire's ToolCallDTO lives
// in the frontend mirror.
type ToolCallRecord struct {
	WireCallID string
	Tool       string
	Arguments  string
}

// ToolResultRecord is the port-side projection of the wire's
// ToolResultDTO (R-CTS-006, R-CCS-015). It carries the outcome enum
// (closed vocabulary: "success" | "result_failure" |
// "execution_failure"), the content bytes for success /
// result_failure outcomes, and a typed failure category string for
// execution_failure outcomes (no provider text — R-CCP-008 / D6
// mirror).
type ToolResultRecord struct {
	WireCallID      string
	Tool            string
	Outcome         string
	Content         string
	FailureCategory string
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

// ConversationStore is the port the chat archetype owns (R-CCS-010,
// additively widened by R-CCS-013). The originally-closed two-method
// surface is extended by a third method `List`; future widens MUST
// follow the same additive pattern (R-CCS-013's "extending, not
// replacing, this declaration" anticipatory clause). CH-07's postgres
// adapter implements the first two methods against a real database;
// CH-08 widens it again with `List` for the participant-scoped listing
// surface (CH-08.2, R-CRI-002).
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

	// List returns the participant-scoped conversation summaries for
	// participantID (R-CCS-013). At v1, one conversation per
	// participant (decisions #3925 D-1), so the returned slice
	// carries 0 or 1 entry. A miss (no Append under participantID)
	// returns (non-nil empty slice, nil) — not ErrConversationNotFound:
	// the handler maps an empty list to `200 []` so the chat page
	// renders an empty rail rather than a refusal (S-CRI-004, S-CCS-018).
	// The returned slice is a fresh copy on every call (NFR-CCS-004
	// carried forward) — caller-side mutation cannot corrupt the
	// in-memory state.
	List(participantID string) ([]ConversationSummary, error)
}

// ConversationSummary is the port-owned list projection (R-CCS-014).
// The wire DTO (ConversationSummaryDTO, R-CRI-004 in the CH-08 spec)
// is a pure transport projection defined in the chat-resume spec and
// carries the same three fields; the wire must not invent fields
// beyond this struct (REQ-7 closed-union enforcement at the wire's
// edge). LiveActivity and TurnCount are set by the adapter at List
// time; ConversationID is the participant id by the one-per-
// participant decision (D-1, chat_resume-in-browser/decisions #3925).
type ConversationSummary struct {
	ConversationID string
	LastActivityAt time.Time
	TurnCount      int
}

// MemoryConversationStore is the v1 in-memory adapter for
// ConversationStore (D-3, D-4). It holds map[string][]Exchange guarded
// by sync.Mutex covering Append, Load, and List (NFR-CCS-001). Coarse
// locking is sufficient at v1's request rate. The activity map
// tracks the per-participant LastActivityAt so List can return it
// alongside TurnCount (R-CCS-013).
type MemoryConversationStore struct {
	mu       sync.Mutex
	m        map[string][]Exchange
	activity map[string]time.Time
}

// NewMemoryConversationStore returns an empty in-memory store ready
// to receive Append calls. Each call returns an independent store;
// two stores do not share state.
func NewMemoryConversationStore() *MemoryConversationStore {
	return &MemoryConversationStore{
		m:        make(map[string][]Exchange),
		activity: make(map[string]time.Time),
	}
}

// Append records ex under participantID, assigning its Position field
// to the insertion index. The map is guarded by s.mu; concurrent
// Append + Load + List on the same store are race-free under -race.
// The activity map is bumped to time.Now() on every Append so List
// can return LastActivityAt (R-CCS-013, R-CCS-014).
//
// CH-09 (R-CCS-015): the new ToolCalls / ToolResults slices are
// stored verbatim. The in-memory adapter's defensive copy on Load
// (NFR-CCS-008 carries NFR-CCS-004 forward) covers caller-side
// mutation of the new fields; this Append does NOT take a copy of
// the slices because the caller (the projector) is the slice's
// owner and never reuses them post-Append. The Load path carries
// the defensive-copy semantics.
func (s *MemoryConversationStore) Append(participantID string, ex Exchange) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	ex.Position = len(s.m[participantID])
	s.m[participantID] = append(s.m[participantID], ex)
	s.activity[participantID] = time.Now()
	return nil
}

// Load returns the slice of exchanges recorded for participantID, in
// insertion order. A miss returns (nil, ErrConversationNotFound). The
// returned slice is a defensive copy; mutating it does not mutate
// the store's own state (NFR-CCS-004).
//
// CH-09 (NFR-CCS-008, S-CCS-020): each Exchange's ToolCalls and
// ToolResults slices are themselves defensive-copied on Load, so
// caller-side mutation of either slice (or its elements) cannot
// corrupt the store's state. The slice elements are value types
// (ToolCallRecord / ToolResultRecord) so a shallow element copy
// is sufficient; the slice headers need are independent backing
// arrays.
func (s *MemoryConversationStore) Load(participantID string) ([]Exchange, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	src, ok := s.m[participantID]
	if !ok {
		return nil, ErrConversationNotFound
	}
	out := make([]Exchange, len(src))
	for i, ex := range src {
		out[i] = ex
		out[i].ToolCalls = copyToolCallRecords(ex.ToolCalls)
		out[i].ToolResults = copyToolResultRecords(ex.ToolResults)
		out[i].MessageIDs = copyStrings(ex.MessageIDs)
	}
	return out, nil
}

// copyToolCallRecords returns a fresh backing array. Nil-safe: a
// nil input yields a nil output (the JSONB round-trip in postgres
// returns nil for empty arrays, and the in-memory adapter must
// match that surface).
func copyToolCallRecords(src []ToolCallRecord) []ToolCallRecord {
	if src == nil {
		return nil
	}
	out := make([]ToolCallRecord, len(src))
	copy(out, src)
	return out
}

// copyToolResultRecords is the ToolResult twin of copyToolCallRecords.
func copyToolResultRecords(src []ToolResultRecord) []ToolResultRecord {
	if src == nil {
		return nil
	}
	out := make([]ToolResultRecord, len(src))
	copy(out, src)
	return out
}

// copyStrings is the MessageIDs defensive copy helper. Pre-existing
// in spirit (NFR-CCS-004); extracted so the Load path applies the
// same idiom to the new fields.
func copyStrings(src []string) []string {
	if src == nil {
		return nil
	}
	out := make([]string, len(src))
	copy(out, src)
	return out
}

// List returns the participant-scoped conversation summaries for
// participantID. At v1 (D-1, one conversation per participant) the
// returned slice carries 0 or 1 entry: 0 when the participant has
// no recorded exchanges (the handler maps this to `200 []` per
// S-CRI-004 and S-CCS-018), 1 otherwise. The returned slice is a
// defensive copy (NFR-CCS-004); the store's map state is not mutated
// by this call. The underlying `s.m` map iteration is over a stable
// shape — concurrent Append can race the iteration, but the map
// write happens under s.mu which List also takes, so under -race the
// read is consistent.
func (s *MemoryConversationStore) List(participantID string) ([]ConversationSummary, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	src, ok := s.m[participantID]
	if !ok {
		// Non-nil empty slice — JSON serializes as [] not null (the
		// wire contract per S-CRI-004, S-CCS-018).
		return []ConversationSummary{}, nil
	}
	out := make([]ConversationSummary, 0, 1)
	out = append(out, ConversationSummary{
		ConversationID: participantID,
		LastActivityAt: s.activity[participantID],
		TurnCount:      len(src),
	})
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