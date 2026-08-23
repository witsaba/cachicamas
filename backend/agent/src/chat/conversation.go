// CH-02.1/.2/.3 — the archetype's first behaviour. Conversation owns one
// agent.Harness and one *agent.History, reused across successive turns
// (R-CCP-001, D0): one browser turn is exactly one Harness.Run call.

package chat

import (
	"context"
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

// Send drives one browser turn — exactly one Harness.Run call (R-CCP-001) —
// and returns the projected wire event stream. The stream carries exactly
// one terminal event and is then closed exactly once (R-CCP-006).
//
// Two goroutines drive one turn: the runner calls Harness.Run over an
// unbuffered sink (real backpressure — a buffered channel would fake it),
// then hands its own return values to the projector over a buffered
// result channel; the projector ranges the sink, projecting events, and
// completes the terminal wire event once both the held run_end payload and
// the runner's result are available.
func (c *Conversation) Send(ctx context.Context, prompt string) (<-chan WireEvent, error) {
	part, err := ai.NewText(prompt)
	if err != nil {
		return nil, err
	}
	msg, err := ai.NewMessage(ai.RoleUser, part)
	if err != nil {
		return nil, err
	}

	c.mu.Lock()
	c.inFlight = true
	c.mu.Unlock()

	sink := make(chan *agent.Event)
	result := make(chan runResult, 1)
	out := make(chan WireEvent)

	go func() {
		_, finish, runErr := c.harness.Run(ctx, msg, sink)
		result <- runResult{finish: finish, err: runErr}
	}()

	go c.project(ctx, sink, result, out)

	return out, nil
}

// CancelOutcome is Cancel's typed result (R-CCP-006).
type CancelOutcome int

const (
	// CancelRequested reports that a turn was in flight and its
	// cancellation was requested.
	CancelRequested CancelOutcome = iota

	// CancelNoOp reports that no turn was in flight; nothing was touched.
	CancelNoOp
)

// String renders the outcome for a diagnostic reader.
func (o CancelOutcome) String() string {
	switch o {
	case CancelRequested:
		return "requested"
	case CancelNoOp:
		return "no-op"
	default:
		return "cancel-outcome(invalid)"
	}
}

// Cancel requests cancellation of the in-flight turn, if any (R-CCP-006). If
// no turn is in flight, it reports CancelNoOp without touching the harness —
// a distinguishable, non-error outcome, never a typed rejection. Otherwise
// it calls Harness.Interrupt, never Harness.Shutdown: Shutdown latches a
// terminal, one-way refusal, which is not what "cancel this turn" means for
// a conversation that must accept the next prompt.
func (c *Conversation) Cancel() CancelOutcome {
	c.mu.Lock()
	defer c.mu.Unlock()
	if !c.inFlight {
		return CancelNoOp
	}
	c.harness.Interrupt()
	return CancelRequested
}
