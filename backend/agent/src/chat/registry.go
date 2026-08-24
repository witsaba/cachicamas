// CH-03.1/.3 — the Conversation registry (R-CHS-001.c, R-CHS-003, D2,
// AD-CHS-3). One *Conversation per participant, guarded by a single
// mutex; a second POST while inFlight=true signals the 409 path with a
// (nil, true, nil) return rather than synthesising a second Conversation.
// The factory is injected at construction so the registry stays ignorant
// of provider wiring — CH-04's composition root constructs the factory
// with its own scripted or production provider.

package chat

import "sync"

// ConversationFactory constructs a fresh *Conversation for a participant
// who has none yet. Errors are surfaced to the HTTP handler as 500
// ("server") envelopes; in practice they should never fire because the
// only documented failure path is ErrNilProvider, which is a
// composition-time defect, not a runtime one.
type ConversationFactory func(participantID string) (*Conversation, error)

// streamEntry is the per-turn state the registry tracks so the GET
// and DELETE handlers can locate the right stream and enforce the
// cross-participant guard (R-CHS-004.b). nil stream = turn already
// terminated; non-nil = turn in flight, drain it.
type streamEntry struct {
	ownerID   string
	stream    <-chan WireEvent
	completed bool
}

// Registry maps participant ids to their *Conversation and turn-ids
// to the per-turn stream Channel. The map is read-mostly at steady
// state and write-only on first use, so a single mutex is sufficient
// — a finer-grained design would add complexity without measurable
// benefit at v1's request rate.
type Registry struct {
	mu           sync.Mutex
	conversations map[string]*Conversation
	streams      map[string]*streamEntry
	newConv      ConversationFactory
}

// NewRegistry constructs a Registry with the given factory. The factory
// is required (its zero value would panic on first use, not return a
// nil Registry) — composition root composition errors fail-fast here.
func NewRegistry(newConv ConversationFactory) *Registry {
	if newConv == nil {
		panic("chat: NewRegistry requires a non-nil ConversationFactory")
	}
	return &Registry{
		conversations: make(map[string]*Conversation),
		streams:       make(map[string]*streamEntry),
		newConv:       newConv,
	}
}

// GetOrCreate returns the *Conversation for participantID, constructing
// one via the factory on first use. The bool result is the "in flight"
// signal the HTTP handler reads as a 409 — true means a second POST
// raced ahead and the request must be refused (R-CHS-001.c).
//
// Two return shapes:
//   - (conv, false)  — proceed; the handler dispatches Send.
//   - (nil, true)    — 409; a turn is already in flight for this participant.
//
// A factory error is a composition-time defect (the only documented
// failure path is ErrNilProvider, which means the composition root wired
// a Conversation with no provider). Such a defect panics here so it
// surfaces in the failing test, never as a silent refusal.
func (r *Registry) GetOrCreate(participantID string) (*Conversation, bool) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if conv, ok := r.conversations[participantID]; ok {
		if conv.inFlight {
			return nil, true
		}
		return conv, false
	}

	conv, err := r.newConv(participantID)
	if err != nil {
		panic("chat: Registry factory error: " + err.Error())
	}
	r.conversations[participantID] = conv
	return conv, false
}

// ConversationByTurnID returns the *Conversation for a participant, the
// only Conversation the registry ever knows about for that participant
// (D2). The turn-id is not consulted because the registry does not
// currently track per-turn identity — CH-03 v1 binds every turn to its
// participant's single Conversation, and the HTTP handler enforces the
// cross-participant guard separately (R-CHS-004.b).
func (r *Registry) ConversationByTurnID(participantID, _ string) (*Conversation, bool) {
	r.mu.Lock()
	defer r.mu.Unlock()
	conv, ok := r.conversations[participantID]
	return conv, ok
}

// CancelByTurnID invokes Conversation.Cancel on the participant's
// Conversation and reports whether cancellation was actually requested
// (true) or no-op'd (false). For an unknown participant the result is
// false with no side effects, mirroring DELETE-on-non-existent-turn's
// 200/204 response (R-CHS-003.b/c).
func (r *Registry) CancelByTurnID(participantID, _ string) bool {
	r.mu.Lock()
	conv, ok := r.conversations[participantID]
	r.mu.Unlock()
	if !ok {
		return false
	}
	return conv.Cancel() == CancelRequested
}

// StoreStream records the just-opened turn's stream so the GET and
// DELETE handlers can find it. Per-turn, keyed by turnID. Called by
// HandleOpenTurn after Conversation.Send returns.
func (r *Registry) StoreStream(ownerID, turnID string, stream <-chan WireEvent) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.streams[turnID] = &streamEntry{ownerID: ownerID, stream: stream}
}

// MarkStreamCompleted marks the turn's stream as terminated but keeps
// the entry so a late GET can still observe completion (S-CHS-002.b).
// The stream field is set to nil to signal "already terminated" without
// removing the entry.
func (r *Registry) MarkStreamCompleted(turnID string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if entry, ok := r.streams[turnID]; ok {
		entry.completed = true
		entry.stream = nil
	}
}

// LoadStream returns the entry for a turn so the GET handler can
// stream it. (stream, ownerID, true) on hit; (nil, "", false) on miss.
// The caller (the GET handler) checks ownerID == caller.ParticpantID()
// for the cross-participant guard (R-CHS-004.b).
func (r *Registry) LoadStream(turnID string) (<-chan WireEvent, string, bool) {
	r.mu.Lock()
	defer r.mu.Unlock()
	entry, ok := r.streams[turnID]
	if !ok {
		return nil, "", false
	}
	return entry.stream, entry.ownerID, true
}

// OwnerOf returns the participantID that owns turnID, if any. Used by
// the DELETE handler for the cross-participant guard.
func (r *Registry) OwnerOf(turnID string) (string, bool) {
	r.mu.Lock()
	defer r.mu.Unlock()
	entry, ok := r.streams[turnID]
	if !ok {
		return "", false
	}
	return entry.ownerID, true
}

// ClearStream removes the stream entry. Called after a DELETE so a
// follow-up POST with the same id is treated as a fresh turn (not
// 403 not_found). Entries are small enough that the cost of leaking
// one on the no-DELETE path is negligible over v1's request rate.
func (r *Registry) ClearStream(turnID string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.streams, turnID)
}

// IsCompleted reports whether the stream has already terminated —
// the GET handler reads this to drive the S-CHS-002.b "already-
// terminated" fast path which returns a single turn.end frame.
func (r *Registry) IsCompleted(turnID string) bool {
	r.mu.Lock()
	defer r.mu.Unlock()
	entry, ok := r.streams[turnID]
	if !ok {
		return false
	}
	return entry.completed
}
