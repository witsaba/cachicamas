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

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
	"go.opentelemetry.io/otel/trace/noop"

	"github.com/cachicamas/backend/agent/src/agent"
	"github.com/cachicamas/backend/agent/src/ai"
	"github.com/cachicamas/backend/agent/src/archetype"
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

	// TracerProvider is the OpenTelemetry tracing API tracer
	// provider (ADR 0005 § D3, adr:240) this Conversation threads
	// into Layer 2's harness (fix/chat-stack-wiring, Gap B).
	// Optional: nil leaves the harness's TracerProvider field at
	// zero, which Harness.Run and Harness.Compact resolve to the
	// tracing API's own no-op provider (agent/observability.go:
	// tracerFromHarness). Production wiring at cmd/chat/main.go
	// passes the same TracerProvider installOTelSDK registered
	// globally, so L2's invoke_agent + turn + execute_tool +
	// compact spans land in the OTLP pipeline alongside L1's
	// openaicompat request span and the HTTP server span the Echo
	// middleware opens upstream.
	TracerProvider trace.TracerProvider

	// AssistantConfigLoader is the Layer 3 archetype storage handle
	// this Conversation uses for the version-aware system-prompt
	// rebuild contract (REQ-CCVP-001/002/003, design AD-3).
	// Optional: nil leaves the Conversation with the chat v1 literal
	// (`SystemPrompt`) and disables the rebuild mechanism — the
	// historical v1 behaviour. Set by the chat composition root
	// (cmd/chat/main.go) to the same `archetype.Loader` the GET
	// handler uses.
	AssistantConfigLoader archetype.Loader
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

	// cfgTracer is the OpenTelemetry TracerProvider this Conversation
	// received via Config.TracerProvider (Step 2, fix/chat-stack-wiring).
	// Held as a struct field rather than re-derived from c.harness
	// because the archetype-layer chat.turn span is opened by this
	// package's own Send, not by the harness — it is independent of
	// whatever tracer the harness's L2 spans derive (Layer 2's spans
	// derive from h.TracerProvider; the archetype-layer span derives
	// from cfgTracer; in production both are the same value, but the
	// separation keeps the seam explicit). conversationTracerProvider
	// (defined below) wraps this with a no-op fallback.
	cfgTracer trace.TracerProvider

	// assistantConfigTracker drives the version-aware rebuild contract
	// (REQ-CCVP-001/002/003, design AD-3). Nil when no AssistantConfigLoader
	// was supplied in Config — the rebuild is then a no-op and the
	// Conversation keeps the chat v1 literal `SystemPrompt`.
	assistantConfigTracker *archetype.VersionTracker

	mu       sync.Mutex
	inFlight bool
}

// Chat-side tracer scope — mirrors agent.observability.go's own
// "github.com/cachicamas/backend/agent/src/agent" pattern. Every
// span this file opens carries this scope name so a Jaeger search
// can filter chat-archetype instrumentation from Layer 2/Layer 1.
const tracerScope = "github.com/cachicamas/backend/agent/src/chat"

// chatTurnSpanName is the constant span name for the archetype-layer
// span Conversation.Send opens around one browser turn (Step 6, ADR
// 0005 § D3 widening). Cardinality 1, mirrors openaicompat's own
// "chat" span name convention.
const chatTurnSpanName = "chat.turn"

// Attribute keys the chat.turn span carries (ADR 0005 § D3, the
// archetype-layer extension). Each key is a literal constant
// spelled exactly once here and referenced from every recording
// site — never assembled at a call site, mirroring
// agent/observability.go and ai/openaicompat/trace.go's own
// attribute-key discipline.
const (
	turnIDKey        = "cachicamas.turn.id"
	participantIDKey = "cachicamas.participant.id"
	promptBytesKey   = "cachicamas.chat.prompt_bytes"
)

// conversationTracerProvider resolves cfg.TracerProvider through the
// API's own no-op fallback (R-AOB-009's Layer 1 precedent, restated
// for the archetype layer) so a zero-value Config stays inert.
func conversationTracerProvider(tp trace.TracerProvider) trace.TracerProvider {
	if tp == nil {
		return noop.NewTracerProvider()
	}
	return tp
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
	conv := &Conversation{
		harness: &agent.Harness{
			Provider:       cfg.Provider,
			System:         SystemPrompt,
			History:        history,
			Scheduler:      cfg.Scheduler,
			TracerProvider: conversationTracerProvider(cfg.TracerProvider),
			Turn: agent.TurnOptions{
				Tools:            cfg.ToolSource,
				PermissionPolicy: cfg.PermissionPolicy,
			},
		},
		logger:        logger,
		store:         cfg.Store,
		participantID: cfg.ParticipantID,
		cfgTracer:     cfg.TracerProvider,
	}

	// CH-12 (cachicamas-assistant-configuration-ui, REQ-CCVP-001/002/003,
	// design AD-3): if a Loader is wired, build the version tracker.
	// The applyPrompt callback mutates the harness's System field,
	// which Harness.Run reads on every turn. The tracker's initial
	// LoadByKindAndOrg call seeds both the recorded version AND the
	// harness.System prompt (a fresh harness is created above with
	// System = SystemPrompt; if the Loader has a different prompt
	// at version 1, the apply fires during NewVersionTracker to
	// keep the harness in sync).
	if cfg.AssistantConfigLoader != nil {
		tracker, trackerErr := archetype.NewVersionTracker(
			context.Background(),
			cfg.AssistantConfigLoader,
			archetype.KindChat,
			cfg.ParticipantID,
			func(newPrompt string) {
				conv.harness.System = newPrompt
			},
		)
		if trackerErr != nil {
			// A transient Loader hiccup at construction time must
			// NOT fail NewConversation — the Conversation is
			// usable; the next Send's ReloadAssistantConfig will
			// retry. Log so the operator sees the issue.
			logger.LogAttrs(context.Background(), slog.LevelWarn, "assistant config initial load failed",
				slog.String("participant_id", cfg.ParticipantID),
				slog.String("error", trackerErr.Error()),
			)
		}
		conv.assistantConfigTracker = tracker
	}

	return conv, nil
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

	// CH-12 (cachicamas-assistant-configuration-ui, REQ-CCVP-002):
	// at the Send boundary, consult the archetype loader and rebuild
	// the system prompt if a newer version is persisted. NO-OP when
	// no Loader was wired (chat v1 default stays in place). The
	// rebuild happens BEFORE Harness.Run so the new prompt is the
	// one the harness streams this turn. A transient Loader error is
	// logged and the existing prompt is kept — a turn must not be
	// aborted by a config-read hiccup (REQ-CCVP-003 in-flight guarantee).
	if err := c.ReloadAssistantConfig(ctx); err != nil {
		c.logger.LogAttrs(ctx, slog.LevelWarn, "assistant config reload failed (keeping prior prompt)",
			slog.String("participant_id", c.participantID),
			slog.String("error", err.Error()),
		)
	}

	// Open the archetype-layer "chat.turn" span (Step 6, ADR 0005 §
	// D3 widening). Every span L2's Harness.Run and L1's openaicompat
	// request emit becomes a child of this one in Jaeger, giving the
	// operator a single trace_id per browser turn to read end-to-end.
	// The finalizer runs as a `defer` here — never at the individual
	// return sites below — so every exit path, including the rare
	// validation rejections above that already returned early, lands
	// the span exactly once (the L2 harness already follows this
	// R-AGO-007 discipline; the archetype layer mirrors it).
	turnCtx, turnSpan := conversationTracerProvider(c.cfgTracer).Tracer(tracerScope).Start(ctx, chatTurnSpanName,
		trace.WithSpanKind(trace.SpanKindInternal),
		trace.WithAttributes(
			attribute.String(turnIDKey, ""),
			attribute.String(participantIDKey, c.participantID),
			attribute.Int(promptBytesKey, len(prompt)),
		),
	)
	defer turnSpan.End()

	sink := make(chan *agent.Event)
	result := make(chan runResult, 1)
	out := make(chan WireEvent, wireChannelBuffer)

	go func() {
		_, finish, runErr := c.harness.Run(turnCtx, msg, sink)
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

// ReloadAssistantConfig consults the archetype.Loader and, on version
// mismatch with the Conversation's recorded version, rebuilds the
// harness system prompt. No-op when the Conversation was constructed
// without an AssistantConfigLoader (i.e. the chat v1 literal stays in
// place). Returns the Loader error (if any) so the caller can decide
// how to surface it — Send calls this at the Send boundary and logs
// the error rather than failing the turn (a transient Loader outage
// must not tear down an in-flight turn).
//
// Safe to call from any goroutine; the underlying VersionTracker is
// internally serialised.
func (c *Conversation) ReloadAssistantConfig(ctx context.Context) error {
	if c.assistantConfigTracker == nil {
		return nil
	}
	if err := c.assistantConfigTracker.Reload(ctx); err != nil {
		return err
	}
	return nil
}

// LoadedAssistantConfigVersion returns the version the archetype
// tracker last saw from the Loader. Returns 0 when no Loader was wired
// (the Conversation is on the chat v1 default).
//
// Exposed for tests + observability; production callers should rely
// on the Send-boundary rebuild, not poll this.
func (c *Conversation) LoadedAssistantConfigVersion() int {
	if c.assistantConfigTracker == nil {
		return 0
	}
	return c.assistantConfigTracker.RecordedVersion()
}

// SystemPromptForTest returns the harness.System string the
// Conversation is currently driving turns with. Test-only surface
// (the production path reads System via the harness during Run).
func (c *Conversation) SystemPromptForTest() string {
	return c.harness.System
}
