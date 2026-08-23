// CH-02.1/.2/.3 — the archetype's first behaviour. Conversation owns one
// agent.Harness and one *agent.History, reused across successive turns
// (R-CCP-001, D0): one browser turn is exactly one Harness.Run call.
package chat

import (
	"errors"
	"log/slog"
	"sync"

	"github.com/cachicamas/backend/agent/src/agent"
	"github.com/cachicamas/backend/agent/src/ai"
)

// SystemPrompt is the archetype's v1 system prompt (R-CCP-002, D13). This is
// v1 placeholder text a later milestone replaces, not a considered product
// prompt: CH-00 § 4 assigned its content to CH-02, and D13 pinned this
// literal as "one minimal neutral sentence".
const SystemPrompt = "You are the cachicamas chat assistant; answer the participant in plain, well-formatted text."

// ErrNilProvider is returned by NewConversation when cfg.Provider is nil.
var ErrNilProvider = errors.New("chat: Config.Provider is required")

// Config configures a Conversation (design AD-2).
type Config struct {
	// Provider is the model provider every turn streams from. Required.
	Provider ai.ModelProvider

	// Logger receives one structured record per unmapped Layer 2 event
	// (R-CCP-004, D4). Nil defaults to slog.Default().
	Logger *slog.Logger
}

// Conversation owns one agent.Harness and one *agent.History, reused across
// successive turns (R-CCP-001, D0). It drives one browser turn per
// Harness.Run call and projects Layer 2's event stream onto WireEvent
// values.
type Conversation struct {
	harness *agent.Harness
	logger  *slog.Logger

	mu       sync.Mutex
	inFlight bool
}

// NewConversation constructs a Conversation. cfg.Provider is required; every
// other harness seam — including RetryAttempts, left unset so
// defaultRetryAttempts applies (R-CCP-009) — is left at CH-00's recorded v1
// answer. Harness.Shutdown is never called by this package (R-CCP-001): it
// latches a terminal, one-way refusal, which is not what "cancel this turn"
// means for a conversation that must accept the next prompt.
func NewConversation(cfg Config) (*Conversation, error) {
	if cfg.Provider == nil {
		return nil, ErrNilProvider
	}
	logger := cfg.Logger
	if logger == nil {
		logger = slog.Default()
	}
	return &Conversation{
		harness: &agent.Harness{
			Provider: cfg.Provider,
			System:   SystemPrompt,
			History:  agent.NewHistory(),
		},
		logger: logger,
	}, nil
}
