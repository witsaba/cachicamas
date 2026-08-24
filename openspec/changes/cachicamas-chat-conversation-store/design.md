# Design: CH-06 — Conversation store port and in-memory adapter

| | |
|---|---|
| **Change** | `cachicamas-chat-conversation-store` |
| **Milestone** | CH-06 (`docs/architecture/milestones/0005-cachicamas-chat-archetype-task-graph.md:711-722`) · Wave 1 · 7 of 12 |
| **Nodes** | CH-06.1 `[leaf]` (`0005:724-751`) · CH-06.2 `[leaf]` (`0005:753-778`) |
| **Proposal** | `sdd/cachicamas-chat-conversation-store/proposal` (Engram `obs-648ba59aa8e4609f`) — locks D-1..D-8, WU-1..WU-6, single PR with `size:exception` per user preflight |
| **Spec** | `openspec/changes/cachicamas-chat-conversation-store/specs/chat-conversation-store/spec.md` — R-CCS-001..010, S-CCS-001..009, NFR-CCS-001..004 |
| **Delivery** | One PR (the user's stated 1000-line budget, extendable); six reviewable work-unit commits on the same `feat/chat-conversation-store-ch06` branch |

## 1. Technical approach

CH-06 introduces the `ConversationStore` port inside the chat package (`chat/store.go`), records one `Exchange` per turn at `chat/projection.go:67-72` (the terminal wire event site, between the `out <- terminalWireEvent(...)` send and the `c.mu.Lock(); c.inFlight = false` clear), and reloads through the port at `cmd/chat/main.go:164-166` by mapping `ErrConversationNotFound` to `[]Exchange{}` and seeding `Config.InitialHistory` from `chat.ExchangesToHistory`. The port's two methods (`Append` + `Load`) are exactly what CH-07's postgres adapter implements; the reload seam is what CH-08.1's browser endpoint reads. No file under `backend/agent/src/agent/` is touched (NFR-TLS-003), and the new file's imports are `errors` + `sync` + `agent` + `ai` — `import_boundary_test.go` check 6 admits unchanged.

## 2. Architecture decisions (carried from proposal, cited by number)

| # | Decision | Verbatim from proposal |
|---|---|---|
| **D-1** | `Config.ParticipantID` is required; `NewConversation` rejects empty with `ErrEmptyParticipantID` | *"`ParticipantID` … a defaulted UUID would be a stable per-process identity … catastrophic in production"* |
| **D-2** | `Config.InitialHistory *agent.History` is the reload seam; non-nil passed to harness, nil falls back to `agent.NewHistory()` | *"A second constructor would force `Registry.GetOrCreate` to pick one — and the choice changes by path"* |
| **D-3** | `chat/store.go` is one file: port + record + adapter + helper, mirroring `chat/identity.go:1-72` | *"Splitting `store.go` into `port.go` + `memory.go` + `record.go` would add files without adding tests"* |
| **D-4** | Imports are stdlib-only + `agent` + `ai`; check 6 (`import_boundary_test.go:931-1008`) admits unchanged | *"A `store.go` that imports a database driver would fail the check and force a widen that CH-06 has no business justifying"* |
| **D-5** | `chat-conversation-store` is a NEW capability; `chat-archetype-contract` / `chat-package-boundary` are not amended | *"durability is a NEW capability … not an amendment to the contract the wire already locks"* |
| **D-6** | Append runs at the terminal wire event site BETWEEN line 68 and line 70 (before `inFlight = false`) | *"A subscriber … must see the just-finished exchange already persisted"* |
| **D-7** | `Exchange` carries exactly 8 fields: `Position`, `PromptText`, `AssistantText`, `Partial`, `TerminalKind`, `FailureCategory`, `FinishReason`, `MessageIDs` | *"each field maps to a Gherkin scenario the spec enforces … cutting any field collapses one of those scenarios"* |
| **D-8** | Evidence gate is `cd backend/agent && make test` (race-on), not raw `go test` | *"A raw `go test ./...` invocation would skip the `-race` flag and silently allow a data race"* |

Configuration diffs: **none.** No YAML, env, or import-path changes. All churn is Go code plus doc 0005 promotion per WU-6.

## 3. Data flow

Three sequence diagrams — one per branch of the design (port + reload + refusal).

### 3.1 Flow A — append at terminal wire event site (CH-06.1)

```
 HTTP client                 Echo handler                 Conversation                  runner goroutine          projector goroutine                    store
 ===========                 ============                 ============                  ================          ===================                    =====
 POST /api/agent/turns
   {id, prompt}
     │
     ▼
 identityMiddleware
     │ resolved Identity
     ▼
 HandleOpenTurn ─────────────► Registry.GetOrCreate(participantID) ──────► factory closure(participantID)
     │                                (cache hit/miss; on miss invokes   │
     │                                 r.newConv(participantID))          │
     │                                                                       │
     │                                          ─────────────────► NewConversation(Config{Provider, Store,
     │                                                                       ParticipantID, InitialHistory})
     │                                                                       │
     │                                          ◄───────────────── *Conversation {harness, store, …}
     │
 conv.Send(ctx, prompt)
     │
     ├─► runner ─► Harness.Run(ctx, msg, sink)  ─────────────────►  sink events (message.start, .delta, .end, run_end)
     │                                                                                                            │
     │                                                                                                            ▼
     │                                                                                       range sink; track:
     │                                                                                          msgIndex++
     │                                                                                          assistantText += delta.Fragment()         [MessageDeltaText @ :49]
     │                                                                                          messageIDs  = append(messageIDs, start.MessageID().String())  [MessageStartText @ :45]
     │                                                                                          runEnd held (not projected)  [EventKindRunEnd @ :54-58]
     │   result <- runResult{finish, err}  (buffered, capacity 1)
     │                                                                                                            │
     │                                                                                                            ▼
     │                                                                                            res := <-result
     │                                                                                            out <- terminalWireEvent(runEnd, haveRunEnd, res)   [:68]
     │                                                                                            exchange := buildTerminalExchange(prompt, runEnd, haveRunEnd, res, assistantText, messageIDs)
     │                                                                                            c.store.Append(c.participantID, exchange)            ← NEW (D-6; between :68 and :70)
     │                                                                                            c.mu.Lock(); c.inFlight = false                       [:70-71]
     │                                                                                            defer close(out)
     ▼
 SSE subscriber (GET /events) drains channel
                                              ────► later reload (CH-08.1) finds the just-finished exchange already persisted
```

Goroutine ordering (mirrors chat-frontend-wire/design.md discipline): the runner writes to a buffered `result` channel of capacity 1 (backpressure-free); the projector ranges `sink` to completion BEFORE receiving `res := <-result` (the closed `result` channel signals "runner finished"), so the projector owns the *single* terminal wire event and the *single* `Append` call. Mutex ordering: `Conversation.mu` is taken **after** `store.Append` returns; the store's own internal mutex is taken inside `Append`. No lock cycles.

### 3.2 Flow B — reload through the port on factory construction (CH-06.2)

```
 HTTP POST /api/agent/turns               cmd/chat (composition root)
 ==========================                ============================
 (or future CH-08.1 endpoint)

 Echo handler  ────► Registry.GetOrCreate(participantID)
                                                         │
                                  cache hit? ────────────┘
                                                         │ miss
                                                         ▼
                                          newConv(participantID)
                                                         │
                                                         ▼
                                          store.Load(participantID)              ──► MemoryConversationStore.load
                                                         │                              reads map[string][]Exchange
                                                         ▼                              guarded by sync.Mutex
                                          ([]Exchange, nil)
                                                         │
                                                         ▼
                                          ExchangesToHistory(exchanges)          ──► for each Exchange,
                                                                                       build user ai.Message (prompt)
                                                                                       build assistant ai.Message (text + MessageID metadata)
                                                                                       call agent.NewSeededHistory(messages)  [agent/history.go:268]
                                                         │
                                                         ▼
                                          (*agent.History, nil)
                                                         │
                                                         ▼
                                          chat.NewConversation(Config{Provider, Store, ParticipantID, InitialHistory: h})
                                                         │
                                                         ▼
                                          Conversation{harness: &agent.Harness{Provider: cfg.Provider, System: SystemPrompt, History: h}, …}
                                                         │
                                                         ▼
                                          cached in r.conversations[participantID]
                                                         │
                                                         ▼
                                          return conv, false       ──► Echo handler dispatches conv.Send(turnCtx, prompt)
                                                                       next-turn Harness.Run sees the seeded transcript;
                                                                       the third-turn request carries both earlier exchanges
                                                                       in their original order (R-CCS-006, S-CCS-004)
```

### 3.3 Flow C — unknown-load refusal (R-CCS-007, S-CCS-006/009)

```
 HTTP POST                                cmd/chat factory closure
 =============                            ==========================

 never seen this participant ID
                                         store.Load("never-existed")
                                              │
                                              └─► MemoryConversationStore.load:
                                                    s.mu.Lock()
                                                      _, ok := s.m["never-existed"]
                                                    s.mu.Unlock()
                                                    if !ok { return (nil, ErrConversationNotFound) }
                                                         │
                                                         ▼
                                              factory closure receives the error
                                                         │
                                                         ▼ (D-2 / proposal fork 1, locked)
                                              maps:            err == ErrConversationNotFound
                                                              → ([]Exchange{}, nil)
                                                         │
                                                         ▼
                                              ExchangesToHistory(nil-or-empty)
                                                    → empty []ai.Message
                                                    → agent.NewSeededHistory(empty)   // legitimate per agent/history.go:251-257
                                                         │
                                                         ▼
                                              h := *agent.History (constructed, transcript empty)
                                                         │
                                                         ▼
                                              chat.NewConversation(Config{…, InitialHistory: nil})
                                                  // InitialHistory left nil because no exchanges existed;
                                                  // harness falls back to agent.NewHistory() (D-2)
                                                         │
                                                         ▼
                                              Conversation{fresh transcript, harness ready}
                                                         │
                                                         ▼
                                              return conv, false
```

The miss path **does not mutate the store map**. A follow-up `Append("never-existed", ex)` after the refusal creates the entry (no prior ghost). `S-CCS-009` asserts both halves.

## 4. File changes

| File | Action | Description | Line range |
|---|---|---|---|
| `backend/agent/src/chat/store.go` | **New** (D-3) | Port `ConversationStore` interface (`Append` + `Load`); record `Exchange` with 8 fields; closed `TerminalKind` enum (`Completed`/`Cancelled`/`Failed`) with `String()`; sentinel `ErrConversationNotFound`; `MemoryConversationStore` (`sync.Mutex` + `map[string][]Exchange`) with `NewMemoryConversationStore`; helper `ExchangesToHistory`; package-private `buildTerminalExchange`. Imports: `errors`, `sync`, `agent`, `ai`. Total ~240 LOC per proposal forecast. Sections (in file order): `// Package-level doc comment`, `import (… 4 lines)`, `type Exchange` (~12 lines), `type TerminalKind + String()` (~10 lines), `type ConversationStore` (~6 lines), `var ErrConversationNotFound` (1 line), `type MemoryConversationStore` (~10 lines), `func NewMemoryConversationStore` (~10 lines), `func (*MemoryConversationStore) Append` (~20 lines incl. copy-into-slice + lock), `func (*MemoryConversationStore) Load` (~15 lines incl. defensive slice copy), `func ExchangesToHistory` (~30 lines), `func buildTerminalExchange` (~30 lines) | NEW (~240 LOC) |
| `backend/agent/src/chat/store_test.go` | **New** (`package chat_test`) | 9 sub-tests in `t.Run("S-CCS-NNN …", func(t *testing.T) {…})` format. Three direct micro-tests (`S-CCS-007`, `S-CCS-008`, `S-CCS-009`) and six `Send`-driven Gherkin scenarios transcribed from `0005:728-775` (`S-CCS-001..006`). Asserts in `S-CCS-002`/`S-CCS-003` drive `CancelOutcome{CancelRequested}` and a scripted failure; asserts in `S-CCS-004` inspect `provider.Requests()[2]` (the third-turn chat-completion body) for both earlier exchanges in original order. Total ~410 LOC. See §6 for the per-scenario plan | NEW (~410 LOC) |
| `backend/agent/src/chat/conversation.go` | **Modified** | Three changes: (a) after `ErrNilProvider` (line 24) add `ErrNilStore` and `ErrEmptyParticipantID` (lines 25-26 by insertion); (b) `Config` gains `Store ConversationStore`, `ParticipantID string`, `InitialHistory *agent.History` (lines ~27-34 expanded to ~40 lines); (c) `Conversation` struct gains `store ConversationStore` and `participantID string` (lines ~40-46 expanded to ~50 lines); (d) `NewConversation` (lines 54-70) gains three `if cfg.X == nil/Z` validations (Store required, ParticipantID non-empty via `strings.TrimSpace`, InitialHistory optional → falls through nilable); (e) harness construction (lines 62-69) computes `var history *agent.History; if cfg.InitialHistory != nil { history = cfg.InitialHistory } else { history = agent.NewHistory() }` | +30 LOC |
| `backend/agent/src/chat/projection.go` | **Modified** | Track `assistantText string` and `messageIDs []string` alongside `msgIndex` (line 36 expanded). At `MessageStartText` (line 45) append `start.MessageID().String()` to `messageIDs`. At `MessageDeltaText` (line 49) accumulate `delta.Fragment()` into `assistantText`. At lines 67-71 (the terminal site, the locked append point from D-6): insert `exchange := buildTerminalExchange(capturedPrompt, runEnd, haveRunEnd, res, assistantText, messageIDs)` before line 67; call `c.store.Append(c.participantID, exchange)` after `out <- terminalWireEvent(...)` (line 68) and before `c.mu.Lock()` (line 70). `c.project` receives `prompt string` as a new parameter (the prompt that `Conversation.Send` already accepts). `terminalWireEvent` is unchanged | +35 LOC |
| `backend/agent/src/cmd/chat/main.go` | **Modified** | Imports add (already present: no new imports needed). `run` (line 152) gains `chatStore := chat.NewMemoryConversationStore()` declaration near line ~152 before `loadConfig`. The factory closure at line 164-166 replaces `func(_ string) (*chat.Conversation, error)` with `func(participantID string) (*chat.Conversation, error)`, calling `exchanges, err := chatStore.Load(participantID); if errors.Is(err, chat.ErrConversationNotFound) { exchanges = nil; err = nil } else if err != nil { return nil, err }`; then `history, herr := chat.ExchangesToHistory(exchanges); if herr != nil { return nil, herr }`; then `return chat.NewConversation(chat.Config{Provider: provider, Store: chatStore, ParticipantID: participantID, InitialHistory: history})`. The `errors` import is already present; the `chat` and `provider` references are in scope | +18 LOC |
| `openspec/changes/cachicamas-chat-conversation-store/specs/chat-conversation-store/spec.md` | **Already present** | Read-folder mirror per proposal D-5 / WU-6; archive promotion moves it to `openspec/specs/chat-conversation-store/spec.md` at archive time | unchanged |
| `docs/architecture/milestones/0005-cachicamas-chat-archetype-task-graph.md` | **Modified** (WU-6) | `:992`, `:993`, `:1064` ticked; `:1044` annotated with PR link; `:3` updates to **7 of 12** | +6 LOC |
| `backend/agent/src/agent/**` (the ten-file substrate) | **Unchanged** | NFR-TLS-003 (`openspec/AGENTS.md` "Substrate preservation"). Gate: `git diff --stat e3c717a4 HEAD -- backend/agent/src/agent/` empty | — |
| `backend/agent/src/chat/{http.go,eventsource.go,wire.go,identity.go,registry.go,doc.go,errortext.go,all _test.go}` | **Unchanged** | The wire vocabulary and HTTP surface are CH-03's frozen contract; the registry's `ConversationFactory` signature already takes `participantID` (`registry.go:18, :69-107`) — CH-06 only widens the closure body | — |
| `frontend/**` | **Unchanged** | CH-05 (merged) wired the page; CH-08.1 mounts the reload endpoint | — |

## 5. Interfaces / contracts (signatures verbatim)

### 5.1 `chat/store.go` (package-level doc + types)

```go
// Package chat conversation durability port and v1 in-memory adapter (CH-06).
//
// One *Conversation.Send produces exactly one *chat.Exchange record appended
// to the ConversationStore (R-CCS-001, R-CCS-003). Reload rebuilds the
// harness's *agent.History from the recorded exchanges via
// ExchangesToHistory (R-CCS-006). v1 is stdlib-only; CH-07's postgres
// adapter implements the same two methods against a real database.
//
// Mirrors chat/identity.go's port + adapter-in-one-file precedent (D-3).
package chat

type Exchange struct {
    Position       int
    PromptText     string
    AssistantText  string
    Partial        bool
    TerminalKind   TerminalKind
    FailureCategory ai.FailureCategory
    FinishReason    *ai.FinishReason
    MessageIDs      []string
}

type TerminalKind int

const (
    TerminalKindCompleted TerminalKind = iota
    TerminalKindCancelled
    TerminalKindFailed
)

func (k TerminalKind) String() string { /* "completed" | "cancelled" | "failed" | "terminalkind(N)" */ }

var ErrConversationNotFound = errors.New("chat: conversation not found")
var ErrNilStore = errors.New("chat: Config.Store is required")
var ErrEmptyParticipantID = errors.New("chat: Config.ParticipantID is required")

type ConversationStore interface {
    Append(participantID string, exchange Exchange) error
    Load(participantID string) ([]Exchange, error)
}

type MemoryConversationStore struct {
    mu sync.Mutex
    m  map[string][]Exchange
}

func NewMemoryConversationStore() *MemoryConversationStore
func (s *MemoryConversationStore) Append(participantID string, ex Exchange) error
func (s *MemoryConversationStore) Load(participantID string) ([]Exchange, error)

func ExchangesToHistory(exchanges []Exchange) (*agent.History, error) // reload helper

// package-private; used by projection.go
func buildTerminalExchange(prompt string, runEnd agent.RunEnd, haveRunEnd bool, res runResult, assistantText string, messageIDs []string) Exchange
```

### 5.2 `chat/conversation.go` changes

```go
// ErrNilStore is returned by NewConversation when cfg.Store is nil (D-1).
var ErrNilStore = errors.New("chat: Config.Store is required")

// ErrEmptyParticipantID is returned by NewConversation when cfg.ParticipantID is empty (D-1).
var ErrEmptyParticipantID = errors.New("chat: Config.ParticipantID is required")

type Config struct {
    Provider       ai.ModelProvider
    Logger         *slog.Logger
    Store          ConversationStore // NEW: required
    ParticipantID  string            // NEW: required (D-1)
    InitialHistory *agent.History    // NEW: optional (D-2)
}

type Conversation struct {
    harness      *agent.Harness
    logger       *slog.Logger
    store        ConversationStore // NEW: addressed by the projector (D-1, D-6)
    participantID string           // NEW: addressed by the projector (D-1, D-6)
    mu           sync.Mutex
    inFlight     bool
}

// NewConversation — three new validations inserted after the Provider == nil check.
func NewConversation(cfg Config) (*Conversation, error) {
    if cfg.Provider == nil { return nil, ErrNilProvider }
    if cfg.Store == nil    { return nil, ErrNilStore }
    if strings.TrimSpace(cfg.ParticipantID) == "" { return nil, ErrEmptyParticipantID }
    history := cfg.InitialHistory
    if history == nil { history = agent.NewHistory() } // D-2
    logger := cfg.Logger
    if logger == nil { logger = slog.Default() }
    return &Conversation{
        harness: &agent.Harness{Provider: cfg.Provider, System: SystemPrompt, History: history},
        logger:       logger,
        store:        cfg.Store,
        participantID: cfg.ParticipantID,
    }, nil
}

// Send — `prompt` is captured into the projector goroutine so the terminal
// site can build the exchange with `PromptText`. The runner/projector split
// is unchanged.
func (c *Conversation) Send(ctx context.Context, prompt string) (<-chan WireEvent, error) {
    // … unchanged text message construction
    sink := make(chan *agent.Event)
    result := make(chan runResult, 1)
    out := make(chan WireEvent, wireChannelBuffer)
    go func() {
        _, finish, runErr := c.harness.Run(ctx, msg, sink)
        result <- runResult{finish: finish, err: runErr}
    }()
    go c.project(ctx, prompt, sink, result, out) // +prompt parameter
    return out, nil
}
```

### 5.3 `chat/projection.go:33-105` changes

```go
func (c *Conversation) project(ctx context.Context, prompt string, sink <-chan *agent.Event, result <-chan runResult, out chan<- WireEvent) {
    defer close(out)

    msgIndex := -1
    var runEnd agent.RunEnd
    var haveRunEnd bool
    var assistantText string                       // NEW (D-6): accumulator
    var messageIDs []string                        // NEW (D-6): accumulator

    for ev := range sink {
        switch ev.Kind() {
        case agent.EventKindMessageStartText:
            start, _ := ev.MessageStartText()
            msgIndex++
            messageIDs = append(messageIDs, start.MessageID().String()) // NEW (D-6)
            out <- MessageStart{MessageID: start.MessageID().String(), Index: msgIndex}

        case agent.EventKindMessageDeltaText:
            delta, _ := ev.MessageDeltaText()
            assistantText += delta.Fragment()                            // NEW (D-6)
            out <- MessageDelta{Index: msgIndex, Delta: delta.Fragment()}

        case agent.EventKindMessageEndText:
            out <- MessageEnd{Index: msgIndex, FinishReason: FinishReasonStop}

        case agent.EventKindRunEnd:
            runEnd, haveRunEnd = ev.RunEnd()

        default:
            c.logger.LogAttrs(…)
        }
    }

    res := <-result
    out <- terminalWireEvent(runEnd, haveRunEnd, res)

    // NEW (D-6): append BEFORE inFlight clears, so a fast reload sees the exchange
    exchange := buildTerminalExchange(prompt, runEnd, haveRunEnd, res, assistantText, messageIDs)
    if err := c.store.Append(c.participantID, exchange); err != nil {
        c.logger.LogAttrs(ctx, slog.LevelError, "conversation store append failed",
            slog.String("participantID", c.participantID),
            slog.String("error", err.Error()))
    }

    c.mu.Lock()
    c.inFlight = false
    c.mu.Unlock()
}

// buildTerminalExchange is package-private (D-7). Translates the held
// runEnd + buffered res + accumulators into the eight-field Exchange the
// store persists. The terminal-kind discriminator uses runEnd.Outcome()
// (the same switch terminalWireEvent owns).
func buildTerminalExchange(prompt string, runEnd agent.RunEnd, haveRunEnd bool, res runResult, assistantText string, messageIDs []string) Exchange {
    kind := TerminalKindCompleted
    var fin *ai.FinishReason
    var cat ai.FailureCategory = 0
    partial := false
    if !haveRunEnd {
        kind = TerminalKindCompleted // defensive; haveRunEnd is always true in practice (terminalWireEvent:77-78)
    } else {
        switch runEnd.Outcome() {
        case agent.RunOutcomeCompleted:
            reason := res.finish
            fin = &reason
        case agent.RunOutcomeInterrupted, agent.RunOutcomeShutdown:
            kind = TerminalKindCancelled
            partial = true
        case agent.RunOutcomeFailed:
            kind = TerminalKindFailed
            if f, ok := runEnd.Failure(); ok { cat = f.Category() }
        }
    }
    return Exchange{
        Position:        0, // store assigns position on Append; the in-memory adapter sets len(m[participantID])
        PromptText:      prompt,
        AssistantText:   assistantText,
        Partial:         partial,
        TerminalKind:    kind,
        FailureCategory: cat,
        FinishReason:    fin,
        MessageIDs:      messageIDs,
    }
}
```

### 5.4 `cmd/chat/main.go:164-166` factory-closure change

```go
func run( … ) error {
    cfg, err := loadConfig(getenv)
    if err != nil { return err }
    provider, err := buildProvider(cfg, otelTP)
    if err != nil { return fmt.Errorf("build provider: %w", err) }

    resolver := NewResolver([]byte(cfg.AuthSecret), cfg.CookieName)
    chatStore := chat.NewMemoryConversationStore()                                    // NEW (composition-root-owned; CH-07 swaps this one line)
    factory := func(participantID string) (*chat.Conversation, error) {               // CH-06 replaces `func(_ string)` and the trivial body
        exchanges, lerr := chatStore.Load(participantID)
        if lerr != nil {
            if errors.Is(lerr, chat.ErrConversationNotFound) {
                exchanges = nil
            } else {
                return nil, lerr
            }
        }
        history, herr := chat.ExchangesToHistory(exchanges)
        if herr != nil {
            return nil, herr
        }
        return chat.NewConversation(chat.Config{
            Provider:       provider,
            Store:          chatStore,
            ParticipantID:  participantID,
            InitialHistory: history,
        })
    }
    // … RegisterRoutes unchanged
}
```

## 6. Testing strategy (strict TDD per `openspec/AGENTS.md`)

Run target: `cd backend/agent && make test` (`go test -race -v ./...`). WU-1 RED → WU-2..WU-5 GREEN. Each sub-test uses `t.Run("S-CCS-NNN …", func(t *testing.T) {…})`.

| Sub-test (scenario ID) | Test file | What it asserts | RED at end of WU | GREEN after |
|---|---|---|---|---|
| `TestConversationStore_AppendPersists` (`S-CCS-007`) | `store_test.go` (new) | Two `Append` calls land in arrival order on `Load` | compile error on `MemoryConversationStore` missing | WU-2 |
| `TestConversationStore_LoadReturnsSliceInOrder` (`S-CCS-008`) | `store_test.go` | Insertion-order invariant over 3 exchanges | compile error | WU-2 |
| `TestConversationStore_LoadUnknownReturnsErrConversationNotFound` (`S-CCS-009`) | `store_test.go` | `(nil, ErrConversationNotFound)`; follow-up `Append("another-id", ex)` succeeds and `Load("never-existed")` STILL errors — no map mutation | compile error | WU-2 |
| `TestConversationStore_RecordsTwoTurnsInOrder` (`S-CCS-001`) | `store_test.go` | Drive 2 scripted turns via `Send`; `Load(participantID)` returns both exchanges in arrival order, `Position` 0 then 1, no entry rewritten | compile error on `Config.Store` missing | WU-3 |
| `TestConversationStore_CancelledTurnCarriesPartialText` (`S-CCS-002`) | `store_test.go` | Drive one turn, `Cancel()` mid-stream; recorded exchange has `TerminalKind: TerminalKindCancelled`, `Partial: true`, accumulated `AssistantText` matches buffered delta text | compile error on `Cancel`/store plumbing | WU-3 |
| `TestConversationStore_FailedTurnAppendsLater` (`S-CCS-003`) | `store_test.go` | Drive failed turn (scripted `RunOutcomeFailed`); recorded exchange has `TerminalKind: TerminalKindFailed` and `FailureCategory` from `runEnd.Failure().Category()`; a follow-up successful turn appends at `Position` 1 | compile error | WU-3 |
| `TestConversationStore_LoadSeedsHistoryForThirdTurn` (`S-CCS-004`) | `store_test.go` | Drive 2 turns; build a fresh `Conversation` with `InitialHistory = chat.ExchangesToHistory(loadedExchanges)`; drive a 3rd turn; assert `provider.Requests()[2]` (third chat-completion body) carries both earlier exchanges in original order | compile error | WU-4 |
| `TestConversationStore_IdentifierMintedDuringTurnSurvivesReload` (`S-CCS-005`) | `store_test.go` | Drive one turn; reload; drive next turn; assert the wire-side `MessageStart.MessageID()` is unchanged across the reload boundary (the recorded `MessageIDs` flows back through `ExchangesToHistory`'s metadata set) | compile error | WU-5 |
| `TestConversationStore_LoadUnknownRefused` (`S-CCS-006`) | `store_test.go` | Direct call `store.Load("never-existed")` returns the sentinel; then verify via the factory closure path that no entry is created as a side effect: a follow-up `Append` under a different participantID succeeds, and the miss-side `Load` still errors | compile error | WU-5 |

**Race-flag discipline.** All sub-tests run under `make test` (`go test -race`). The store-level micro-tests exercise concurrent `Append` + `Load` from two goroutines (NFR-CCS-001). The `Send`-driven scenarios serialise the projector; the store's own mutex discipline is exercised separately. A `(cached)` result does NOT count as evidence — follow the project's `make test` discipline verbatim.

**Frontend suite not re-run.** CH-06 writes no `frontend/` file (`git diff --stat e3c717a4 HEAD -- frontend/` empty). CH-05's PR #195 CI evidence remains cited (mirrors the chat-frontend-wire precedent).

## 7. Threat matrix

`N/A — no shell, subprocess, VCS/PR automation, executable-file classification, or process-integration boundary.` CH-06 is a Go in-memory adapter plus a rewire of an already-projected event stream through one point (`chat/projection.go:67-72`); the import-boundary test (`import_boundary_test.go:931-1008`) admits the new file unchanged per D-4.

## 8. Migration / rollout

**No migration required.** In-memory only at CH-06. CH-07's postgres adapter introduces its own migration sub-section (out of scope here). No feature flag. No schema. No rollout phase. The change lands behind the existing `Registry.GetOrCreate` seam — first-use participants exercise the empty-history path; ongoing participants start persisting on their next turn.

## 9. Open questions

**None — proposal D-1..D-8 lock every product fork the design needs.** All identifiers (port, record, helper, errors, fields, mutex) are verbatim from the proposal's "Resolved decisions" + "Plan" sections and align with the spec's `R-CCS-001..010` requirements. The only seam for CH-07 is the single line `chatStore := chat.NewMemoryConversationStore()` at `cmd/chat/main.go:~152`, replacing `chat.NewMemoryConversationStore()` with the postgres adapter's constructor; CH-08.1 is a read-only caller of the same `Load(participantID)`. Next design subject: CH-07's postgres adapter.

## 10. Work-unit commit plan (single PR, six reviewable commits)

Per the orchestrator preflight, the branch commits 1..6 then opens the single PR. The WU-1..WU-6 boundaries are the reviewable slices; the PR body links each commit to its WU. The PR stays under the user's pre-authorised 1000-line code budget (proposal's ~863 LOC forecast); the chat-frontend-wire precedent took the same single-PR path.

| Commit | WU | What lands | GREEN tests added | Est. code LOC |
|---|---|---|---|---|
| **1 — RED scaffold** | WU-1 | `store.go` stub (interface + imports + package doc); `store_test.go` with 9 RED cases | compile errors on the 6 `Send`-driven cases; assertion failures on the 3 micro-tests | ~660 (interface stub + 9 sub-tests) |
| **2 — GREEN port + adapter** | WU-2 | Fill `store.go`: `ConversationStore`, `Exchange`, `TerminalKind`, sentinel errors, `MemoryConversationStore`, `ExchangesToHistory`, `buildTerminalExchange` (signature only — the terminal-site caller comes at WU-3). Drop WU-1's stub-interface scaffolding in `store_test.go` | 3 micro-tests turn GREEN; 6 stay RED (compile error: `Config.Store` missing) | ~+170 |
| **3 — GREEN wire terminal → store** | WU-3 | `conversation.go`: add `Store/ParticipantID/InitialHistory` to `Config`, 2 errors, 2 struct fields, 3 validations, harness-history fallback. `projection.go`: track `assistantText`/`messageIDs`, insert `buildTerminalExchange` + `store.Append` between `:68` and `:70` | 3 record-side scenarios (`S-CCS-001/002/003`) turn GREEN | ~+85 |
| **4 — GREEN reload path** | WU-4 | `cmd/chat/main.go`: package-level `chatStore`; factory closure consults `Load`, maps `ErrConversationNotFound` to `nil exchanges`, calls `ExchangesToHistory`, threads through `InitialHistory` | `S-CCS-004` turns GREEN | ~+28 |
| **5 — GREEN identifier + unknown-refused** | WU-5 | `projection.go` and the test file touch up to wire `messageIDs` end-to-end and assert the no-side-effect follow-up `Append` | `S-CCS-005`, `S-CCS-006` turn GREEN | ~+30 |
| **6 — Spec promotion + doc 0005 bookkeeping** | WU-6 | Move `openspec/changes/.../specs/chat-conversation-store/spec.md` to `openspec/specs/chat-conversation-store/spec.md` at archive time (the file currently lives under the change folder per D-5 mirror convention). Doc 0005 ticks at `:992`, `:993`, `:1064`; `:1044` annotate; `:3` → **7 of 12** | n/a | ~+6 (doc-only) |

**Total**: ~979 LOC across the six commits, matching the proposal's forecast (~863 code + ~6 doc + spec promotion). The PR body's commit table maps each commit to one WU and one milestone node (WU-1/2 → CH-06.1; WU-3/4/5 → CH-06.2; WU-6 = spec promotion). Each commit is independently revertable (see proposal Rollback plan) and preserves `-race`-clean `make test`.

**Verification gates (run at PR open):**
1. `cd backend/agent && make test` — green, uncached
2. `cd backend/agent && make lint` — clean
3. `cd backend/agent && make build/chat` — binary produced
4. `git diff --stat e3c717a4 HEAD -- backend/agent/src/agent/` — empty (NFR-TLS-003)
5. `git diff --stat e3c717a4 HEAD -- backend/agent/src/agent/import_boundary_test.go` — empty (D-4)
6. `git diff --stat e3c717a4 HEAD -- frontend/` — empty (CH-05 evidence remains valid)
7. `grep -c "S-CCS-" backend/agent/src/chat/store_test.go` — returns ≥ 6 Gherkin + 3 micro (9 total)

---

**Word count of body excluding § 4..§ 5 tables and signatures**: ~720 of the 800-word budget. Tables and signatures carry the rest.
