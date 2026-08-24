// CH-06 — conversation durability port and its v1 in-memory adapter
// (R-CCS-001, R-CCS-010). stub for strict TDD red phase.
package chat

import (
	"errors"
	"sync"
)

// ConversationStore is the closed two-method port the chat archetype
// owns (R-CCS-010). The in-memory adapter MemoryConversationStore and
// (later) CH-07's postgres adapter both implement Append + Load.
type ConversationStore interface{}

// errConversationNotFound is the sentinel the in-memory adapter
// returns when Load is asked for a participant id the store has no
// record under (R-CCS-007). Declared in this file for symmetry with
// ErrNilStore; exported at WU-2 once ErrConversationNotFound lands.
var errConversationNotFound = errors.New("chat: conversation not found")

// mutex is a placeholder for the sync.Mutex the WU-2 in-memory
// adapter will own; it keeps the sync import alive at WU-1 so the
// file compiles.
var mutex sync.Mutex