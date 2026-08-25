// CH-02.1/.2/.3 — the archetype's first behaviour. Conversation owns one
// agent.Harness and one *agent.History, reused across successive turns
// (R-CCP-001, D0): one browser turn is exactly one Harness.Run call.
//
// CH-06 — extends the conversation with conversation-store durability
// (R-CCS-001, D-1, D-2): Config gains Store (required), ParticipantID
// (required), and InitialHistory (optional seed). The terminal wire
// event site at projection.go calls c.store.Append before clearing
// inFlight (R-CCS-008, D-6).

package chat

import (
	"context"
	"errors"
	"log/slog"
	"strings"
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

// Config configures a Conversation (design AD-2 + CH-06 D-1, D-2 +
// CH-09.1 D-1 + R-CTS-001 + CH-10.1 D-1 + R-CPM-001).
type Config struct {
	// Provider is the model provider every turn streams from. Required.
	Provider ai.ModelProvider

	// Logger receives one structured record per unmapped Layer 2 event
	// (R-CCP-004, D4). Nil defaults to slog.Default().
	Logger *slog.Logger

	// Store is the conversation-store port this Conversation persists
	// every completed turn into (R-CCS-001, D-1). Required: nil is
	// rejected by NewConversation with ErrNilStore. The store is
	// addressed at the terminal wire event site in projection.go by
	// the participant id below (D-6).
	Store ConversationStore

	// ParticipantID is the store's addressable key — the identity the
	// reload path consults (R-CCS-001, D-1). Required: an empty or
	// whitespace-only value is rejected by NewConversation with
	// ErrEmptyParticipantID. A defaulted UUID would be a stable
	// per-process identity, catastrophic in production; the participant
	// id must come from the HTTP layer (Registry.GetOrCreate passes it
	// through to the factory).
	ParticipantID string

	// InitialHistory is the reload seam (R-CCS-006, D-2). Non-nil is
	// passed directly to the harness (the door at
	// agent.NewSeededHistory validates the message slice). Nil falls
	// back to agent.NewHistory(), today's default.
	InitialHistory *agent.History

	// ToolSource is the chat-owned port for tool resolution (CH-09.1,
	// D-1, R-CTS-001). The conversation adapts it back to Layer 2's
	// agent.Registry for harness.Tools (loop.go:112). Required: nil
	// is rejected by NewConversation with ErrNilToolSource (typed,
	// never panic). Constructed once at the composition root via
	// chat.FromAgentRegistry(agent.NewMapRegistry(...)); see
	// cmd/chat/main.go for the production wire.
	ToolSource ToolSource

	// PermissionPolicy is the chat-owned port for the permission
	// policy (CH-10.1, D-1, R-CPM-001). The conversation injects it
	// into the harness's Turn.PermissionPolicy so Layer 2's gate
	// (runPermissionGate, AG-10) consults it on every scheduled
	// tool call. Required: nil is rejected by NewConversation with
	// ErrNilPermissionPolicy (typed, never panic). Constructed once
	// at the composition root via
	// chat.NewDefaultPermissionPolicy(deferToolNames); see
	// cmd/chat/main.go for the production wire.
	PermissionPolicy PermissionPolicy

	// Scheduler is the wake handle the HTTP reverse-channel
	// handler reaches to unblock a parked gate (CH-10.1, R-CPM-004,
	// D-11). Optional: nil leaves the harness's Scheduler field
	// at zero, which Harness.Run lazily initializes to a fresh
	// *agent.Scheduler per Run (per-Run construction at
	// agent/harness.go:446-449). Tests inject a real Scheduler
	// so the HTTP handler's WakeParked call has a non-nil
	// receiver; production callers leave this nil and rely on
	// the per-Run lazy default.
	Scheduler *agent.Scheduler
}

// Conversation owns one agent.Harness and one *agent.History, reused across
// successive turns (R-CCP-001, D0). It drives one browser turn per
// Harness.Run call and projects Layer 2's event stream onto WireEvent
// values. CH-06 widens this with the store + participant id the
// projector addresses at the terminal wire event site (R-CCS-008,
// D-6).
type Conversation struct {
	harness *agent.Harness
	logger  *slog.Logger

	store         ConversationStore
	participantID string

	mu       sync.Mutex
	inFlight bool
}

// NewConversation constructs a Conversation. cfg.Provider is required;
// cfg.Store is required (R-CCS-001); cfg.ParticipantID is required and
// non-whitespace (D-1); cfg.ToolSource is required (R-CTS-001,
// CH-09.1); cfg.PermissionPolicy is required (R-CPM-001, CH-10.1);
// every other harness seam — including RetryAttempts, left
// unset so defaultRetryAttempts applies (R-CCP-009) — is left at
// CH-00's recorded v1 answer. Harness.Shutdown is never called by
// this package (R-CCP-001): it latches a terminal, one-way refusal,
// which is not what "cancel this turn" means for a conversation that
// must accept the next prompt.
func NewConversation(cfg Config) (*Conversation, error) {
	if cfg.Provider == nil {
		return nil, ErrNilProvider
	}
	if cfg.Store == nil {
		return nil, ErrNilStore
	}
	if strings.TrimSpace(cfg.ParticipantID) == "" {
		return nil, ErrEmptyParticipantID
	}
	if cfg.ToolSource == nil {
		return nil, ErrNilToolSource
	}
	if cfg.PermissionPolicy == nil {
		return nil, ErrNilPermissionPolicy
	}
	logger := cfg.Logger
	if logger == nil {
		logger = slog.Default()
	}
	history := cfg.InitialHistory
	if history == nil {
		history = agent.NewHistory()
	}
	// chat.ToolSource wraps agent.Registry; we hand it to the harness
	// as the executor-side registry. chat.ToolSource's method set is
	// byte-identical to agent.Registry's, so the dynamic type
	// (agentRegistryAdapter or an in-test stub) satisfies both
	// interfaces and Go's interface assignment accepts the value
	// without an explicit conversion.
	//
	// CH-10.1 (D-1, R-CPM-001): chat.PermissionPolicy is a type alias
	// of agent.PermissionPolicy; we hand the same value to
	// harness.Turn.PermissionPolicy. The harness threads it into
	// Schedule via the per-turn Continuation's field (CH-09.1 D-1
	// precedent + CH-10.1 widening).
	//
	// CH-10.1 (D-11): the harness's Scheduler field is the
	// caller-owned wake handle. nil leaves Harness.Run's per-Run
	// lazy default (agent/harness.go:446-449); a non-nil value
	// (test injection) threads through Conversation.Scheduler().
	return &Conversation{
		harness: &agent.Harness{
			Provider:  cfg.Provider,
			System:    SystemPrompt,
			History:   history,
			Scheduler: cfg.Scheduler,
			Turn: agent.TurnOptions{
				Tools:            cfg.ToolSource,
				PermissionPolicy: cfg.PermissionPolicy,
			},
		},
		logger:        logger,
		store:         cfg.Store,
		participantID: cfg.ParticipantID,
	}, nil
}

// Send drives one browser turn — exactly one Harness.Run call (R-CCP-001) —
// and returns the projected wire event stream. The stream carries exactly
// one terminal event and is then closed exactly once (R-CHS-006 — was
// R-CCP-006 before CH-03.1 widened it).
//
// Two goroutines drive one turn: the runner calls Harness.Run over an
// unbuffered sink (real backpressure — a buffered channel would fake it),
// then hands its own return values to the projector over a buffered
// result channel; the projector ranges the sink, projecting events, and
// completes the terminal wire event once both the held run_end payload and
// the runner's result are available.
//
// The projected wire stream is BUFFERED (capacity wireChannelBuffer,
// small enough to bound memory but large enough that a POST can return
// 200 before the SSE subscriber attaches — R-CHS-001.a's invariant "the
// turn is driving before the response is written" is preserved because
// the projector goroutine writes into the buffered channel; if no
// subscriber ever arrives, only the buffer's worth of events sit in the
// channel until the terminal close lands).
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

	// Fix C (observability): record the chat turn's start so an
	// operator can correlate the projected wire stream with the
	// upstream provider's request, the participant's reload surface,
	// and the failure logs (when the run fails). Without this log
	// `docker logs cachicamas-agent-chat` is silent until something
	// goes wrong, and even then the failure-only log line leaves no
	// breadcrumb back to the originating turn. The participant id is
	// already on the Conversation struct; the prompt size is the
	// only new attribute — kept off the body itself to avoid leaking
	// user input through the structured logs (the prompt is also
	// persisted via buildTerminalExchange, where it belongs).
	c.logger.LogAttrs(ctx, slog.LevelInfo, "chat turn started",
		slog.String("participant_id", c.participantID),
		slog.Int("prompt_bytes", len(prompt)),
	)

	sink := make(chan *agent.Event)
	result := make(chan runResult, 1)
	out := make(chan WireEvent, wireChannelBuffer)

	go func() {
		_, finish, runErr := c.harness.Run(ctx, msg, sink)
		result <- runResult{finish: finish, err: runErr}
	}()

	// The projector receives the prompt verbatim so it can stamp the
	// resulting Exchange's PromptText field (D-7, R-CCS-004 / R-CCS-006).
	go c.project(ctx, prompt, sink, result, out)

	return out, nil
}

// wireChannelBuffer sizes the projected wire channel. A small buffer
// (capacity 8) lets a POST return 200 immediately while the projector
// concurrently writes message.start / message.delta / message.end /
// turn.end without blocking — the SSE subscriber can attach slightly
// later and still see the ordered frames (S-CHS-002.a). Memory cost
// is bounded: the buffer stores, at most, the events from a single
// short turn; for a long turn, the projector blocks on the SSE
// subscriber's read pace, which is the backpressure the HTTP path
// requires.
const wireChannelBuffer = 8

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

// IsInFlight reports whether a turn is currently driving through this
// Conversation. The read happens under c.mu so callers outside the
// package — chiefly chat.Registry.GetOrCreate, which needs to decide
// whether a fresh POST may proceed or must be refused with a 409
// (S-CHS-001.c) — do not race the writer in chat/projection.go's
// terminal-event site. Returning the bool (not a copy of the struct)
// is the durable property; the field itself can change shape in a
// future refactor without breaking the call site.
func (c *Conversation) IsInFlight() bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.inFlight
}

// Scheduler returns the harness-owned *agent.Scheduler the HTTP
// reverse-channel handler reaches to wake a parked permission gate
// (R-CPM-001, D-11, R-CPM-004). Method-only — no stored field on
// Conversation (D-11 option (b)); the harness's own Scheduler field
// at `agent/harness.go:81` is the single source of truth. The
// harness constructs a default Scheduler per Run when none was
// injected (per-Run lazy construction at `agent/harness.go:446-449`),
// so even a chat composition that doesn't inject a scheduler reaches
// a live *agent.Scheduler through this accessor.
//
// The returned pointer is the same value Layer 2's gate wrote to
// (when Schedule ran); WakeParked closes the parked channel for the
// callID whose decision has just been recorded. Chat never mutates
// the scheduler; this method is a pure pass-through.
func (c *Conversation) Scheduler() *agent.Scheduler {
	return c.harness.Scheduler
}

// ParticipantIDForTest returns the participantID the Conversation
// was constructed with. Test-only surface (the production path
// reads participantID via the registry's per-conversation lookup;
// production code MUST NOT call this). Exposed so the CH-10 HTTP
// test helpers can register the test conversation with the
// chat-package Registry.
func (c *Conversation) ParticipantIDForTest() string {
	return c.participantID
}
