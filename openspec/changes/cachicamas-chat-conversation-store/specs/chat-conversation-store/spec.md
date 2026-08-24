# Spec — Chat conversation store (`chat-conversation-store`)

> **Change**: `cachicamas-chat-conversation-store` · **CH-06** (Wave 1, 7 of 12) of [doc 0005](../../../docs/architecture/milestones/0005-cachicamas-chat-archetype-task-graph.md#ch-06--define-the-conversation-store-port-and-its-in-memory-adapter) (`0005:711-722`) · leaves CH-06.1 `[leaf]` (`0005:724-751`) · CH-06.2 `[leaf]` (`0005:753-778`)
> **Closes**: R-04 (define your own ports), R-16 (durable state independent of transport), register row 3 (home-directory sessions are not a layer rule), research finding 5 (the stream is not the store of record)
> **Status**: **new capability**, promoted verbatim to `openspec/specs/chat-conversation-store/spec.md` at archive. Per D-5, no promoted spec is amended.
> **Depends on**: CH-02 (`e3c717a4`). **Blocks**: CH-07 (`cachicamas-chat-store-adapter`), CH-08 (`cachicamas-chat-resume-in-browser`).
> **Format**: Given/When/Then + RFC 2119 per `openspec/config.yaml` `rules.specs`. Every scenario is independently verifiable — the Gherkin `S-CCS-NNN` rows are transcribed verbatim from `0005:728-775`; the `S-CCS-NNN` micro-test rows are direct sub-tests of `MemoryConversationStore` per the proposal's WU-1.
> **IDs**: requirements `R-CCS-NNN`, scenarios `S-CCS-NNN`, non-functional `NFR-CCS-NNN`. Append-only.
> **Prefix verification**: `CCS` was verified collision-free across `openspec/specs/`, `openspec/changes/` and `docs/architecture/milestones/0005-…md` (proposal D-8).
> **Evidence gate**: `cd backend/agent && make test` (with `-race`) green, uncached; `make lint`; `make build/chat`. Frontend suite **not** re-run — CH-06 writes no `frontend/` file (gate's mechanical form: `git diff --stat e3c717a4 HEAD -- frontend/` empty). A `(cached)` result is not evidence.
>
> **Note on length.** The `sdd-spec` skill sets a 650-word budget. This spec exceeds it deliberately, on the recorded precedent of `chat-archetype-contract/spec.md:16` and `chat-package-boundary/spec.md:13`: ten requirements, six Gherkin scenarios transcribed verbatim (per the skill's "MUST be independently verifiable" rule, scenario wording cannot compress without losing traceability to `0005:728-775`), and four NFRs with mechanical gates do not compress without dropping content `openspec/config.yaml` requires be independently verifiable.

## Purpose

A conversation survives only as long as its caller; every turn is lost when the SSE connection closes (research finding 5, `0005:85`). This capability fixes the defect by introducing the port `ConversationStore`, owned by the chat archetype, and satisfying it first with an adapter that needs no database. The port shape — `Append` + `Load` + `ErrConversationNotFound` — is the seam on which CH-07's postgres adapter and CH-08.1's browser reload both land without changing any caller (`0005:788`).

The acceptance criterion is observable end-to-end (`0005:719`): drive a conversation through two turns, reload through the port, drive a third turn — the third turn's chat-completion request carries both earlier exchanges in original order. The Gherkin at `0005:728-775` is the binding contract; the spec's `S-CCS-001`…`S-CCS-006` scenario names are the sub-test names in `chat/store_test.go`.

## Coverage — the charter and the proposal, traced

| Charter clause / decision | Requirements | Scenarios |
|---|---|---|
| Charter `0005:719` acceptance (2 turns → reload → 3rd turn carries both) | `R-CCS-006` | `S-CCS-004` |
| D-1 `ParticipantID` required | `R-CCS-001` | `S-CCS-006`, `S-CCS-009` |
| D-2 `InitialHistory` is the reload seam | `R-CCS-001`, `R-CCS-006` | `S-CCS-004` |
| D-3 port + adapter in one file | `R-CCS-002` | all micro-tests |
| D-4 stdlib-only imports | `R-CCS-002`, `NFR-CCS-002` | (gated by import-boundary test) |
| D-5 new capability, not an amendment | (whole spec) | (whole spec) |
| D-6 append before `inFlight = false` | `R-CCS-008` | `S-CCS-004`, `S-CCS-006` |
| D-7 `Exchange` carries 8 fields round-trippable to `*agent.History` | `R-CCS-004`, `R-CCS-005`, `R-CCS-006`, `R-CCS-009` | `S-CCS-002`, `S-CCS-003`, `S-CCS-004`, `S-CCS-005` |
| D-8 `make test` (race-clean), not raw `go test` | `NFR-CCS-001` | (whole spec) |

---

## Requirements

### R-CCS-001 — Conversation durability is a port the archetype owns

The system MUST expose all conversation persistence through the `ConversationStore` port. `Conversation.NewConversation(Config)` MUST accept the store via `Config.Store`, the participant via `Config.ParticipantID` (required, D-1 — `NewConversation` rejects empty with `ErrEmptyParticipantID`), and the reload seed via `Config.InitialHistory` (optional, D-2 — `nil` falls back to `agent.NewHistory()`; non-nil is passed directly to the harness, validated by `agent.NewSeededHistory` at `agent/history.go:268`). The store is the only durability surface callers reach.

The composition root at `cmd/chat/main.go:164-166` is the **only** place that constructs a `MemoryConversationStore` and consults `Load(participantID)` on first construction (D-8, locked fork). The factory closure at `registry.go:101` already takes `participantID`; CH-06 does not change the registry signature.

#### Scenarios

- **S-CCS-001** — every exchange is appended in order (Gherkin verbatim, `0005:731-735`)
  - Given a conversation driven through two turns
  - When its record is read through the port
  - Then the record carries both exchanges in the order they occurred
  - And no earlier entry was rewritten by a later one

- **S-CCS-006** — a reload of an unknown conversation is refused, not invented (Gherkin verbatim, `0005:770-774`)
  - Given an identity no conversation was ever recorded under
  - When a reload is attempted through the port
  - Then it is refused as not found
  - And no empty conversation is created as a side effect

### R-CCS-002 — The in-memory adapter needs no database

`MemoryConversationStore` MUST be stdlib-only (`errors`, `sync`) plus `agent` (`NewSeededHistory`) and `ai` (`FailureCategory`, `FinishReason`) — D-4. The import-boundary check at `backend/agent/src/agent/import_boundary_test.go:931-1008` admits the new file unchanged. CH-07's postgres adapter implements the **same two methods** of the same port (R-CCS-010); v1 carries no production dependency on any database driver.

#### Scenarios

- **S-CCS-007** — Append persists the exchange in arrival order
  - Given an in-memory adapter and a participant id `p`
  - When `Append(p, exchangeA)` is called and then `Append(p, exchangeB)` is called
  - Then `Load(p)` returns `[exchangeA, exchangeB]` in arrival order

- **S-CCS-008** — Load returns the slice in insertion order
  - Given an in-memory adapter with three appended exchanges `e1`, `e2`, `e3` for the same participant
  - When `Load(participantID)` is called
  - Then the returned slice is `[e1, e2, e3]` in insertion order

### R-CCS-003 — Recording is append-only

Each `Conversation.Send` call MUST produce exactly ONE exchange record. The in-memory adapter stores the slice in arrival order; no method rewrites a prior entry. The terminal wire event site at `chat/projection.go:67-72` is the **only** append point (one per `Send`, not a stream-level tap) — locked fork.

#### Scenario: S-CCS-009 — Load of an unknown conversation returns `ErrConversationNotFound`

- Given an in-memory adapter
- When `Load("never-existed")` is called
- Then the result is `(nil, chat.ErrConversationNotFound)`
- And a follow-up `Append("another-id", exchange)` returns `nil` and `Load("another-id")` returns one entry — the miss did not mutate the map.

### R-CCS-004 — A cancelled turn carries what it produced

When `runEnd.Outcome() == agent.RunOutcomeInterrupted` (the same discriminator `terminalWireEvent` owns at `projection.go:87`), the exchange record MUST carry `Partial: true`, `TerminalKind: TerminalKindCancelled`, and the accumulated `AssistantText` built from `MessageDelta.Fragment()` (accumulated alongside `msgIndex` at `projection.go:49`). The wire's cancellation discriminator is `FinishReason`'s ABSENCE (`projection.go:90-93`), which round-trips losslessly through the `Exchange.FinishReason == nil` form.

#### Scenario: S-CCS-002 — a cancelled turn is recorded with what it produced (Gherkin verbatim, `0005:737-741`)

- Given a turn cancelled after producing partial assistant text
- When the record is read
- Then the partial text is present
- And the turn is marked as ended by cancellation rather than by completion

### R-CCS-005 — A failed turn is marked failed and later turns append after it

When `runEnd.Outcome() == agent.RunOutcomeFailed`, the exchange record MUST carry `TerminalKind: TerminalKindFailed` and `FailureCategory` populated from `runEnd.Failure().Category()` (D-7). The zero value of `FailureCategory` is legitimate on non-failed turns and is round-trip-safe.

#### Scenario: S-CCS-003 — a failed turn is recorded as failed (Gherkin verbatim, `0005:743-747`)

- Given a turn terminated by a typed provider error
- When the record is read
- Then the turn is present and marked failed
- And a later turn on the same conversation still appends after it

### R-CCS-006 — Reload rebuilds the harness's `*agent.History` from the record

`ExchangesToHistory([]Exchange) (*agent.History, error)` MUST round-trip through `agent.NewSeededHistory` (`agent/history.go:268`). Each exchange becomes one user `ai.Message` (the prompt) plus one assistant `ai.Message` (the accumulated text); the assistant message's `MessageID` (when present) is set on the message metadata so reload-driven runs preserve the wire-side IDs. `Config.InitialHistory` carries the rebuilt history into `NewConversation`.

#### Scenario: S-CCS-004 — a reloaded conversation continues the same transcript (Gherkin verbatim, `0005:760-763`)

- Given a conversation driven through two turns and then reloaded through the port
- When a third turn is driven on the reloaded conversation
- Then the request carries both earlier exchanges in their original order

### R-CCS-007 — An unknown conversation's reload is refused, not invented

`Load(participantID)` MUST return `(nil, ErrConversationNotFound)` on miss and MUST NOT mutate the map. The composition root at `cmd/chat/main.go:164-166` MUST map the error to `[]Exchange{}` and call `chat.ExchangesToHistory`; the resulting conversation starts fresh — same shape as a participant with no turns yet.

### R-CCS-008 — The append happens BEFORE the `inFlight` clear, not after

The terminal wire event site at `chat/projection.go:67-72` MUST call `c.store.Append(c.participantID, exchange)` between `out <- terminalWireEvent(...)` (line 68) and `c.mu.Lock(); c.inFlight = false` (lines 70-71). The store's own internal mutex is taken inside `Append`; the `Conversation.mu` is taken after — the mutex ordering is preserved. A subscriber that receives the terminal wire event and fires a fast `GET …/events` or CH-08.1 reload MUST see the just-finished exchange already persisted.

### R-CCS-009 — An identifier minted during a turn survives reload

The wire's `MessageStart.MessageID().String()` values (projected at `projection.go:45`) MUST be recorded into `Exchange.MessageIDs`. On reload, the assistant message's `ai.MessagePart` MUST carry the same `MessageID` metadata; the reloaded conversation's third-turn request bears the IDs forward (D-7).

#### Scenario: S-CCS-005 — an identifier minted during a turn survives reload (Gherkin verbatim, `0005:765-768`)

- Given a turn whose exchange carries an identifier minted while it ran
- When the conversation is reloaded through the port
- Then that identifier is present and unchanged

### R-CCS-010 — The port is a closed, two-method interface

`ConversationStore` MUST declare exactly `Append(participantID string, exchange Exchange) error` and `Load(participantID string) ([]Exchange, error)`. Adding a method is a semantic break; CH-07's postgres adapter implements the same two methods against a real database, and CH-08.2 widens the port to a list (out of scope here) by extending, not replacing, this declaration.

---

## Non-functional requirements

### NFR-CCS-001 — Race-free under `go test -race`

All `MemoryConversationStore` operations MUST be guarded by the same `sync.Mutex` covering both `Append` and `Load`. Verified by `cd backend/agent && make test` (the project's Makefile target, which runs `go test -race -v ./...` per `openspec/AGENTS.md` § 1) — D-8.

### NFR-CCS-002 — Stdlib-only imports for the port and its v1 adapter

No new top-level Go dependency. The new file's imports are exactly `errors`, `sync`, `github.com/cachicamas/backend/agent/src/agent`, and `github.com/cachicamas/backend/agent/src/ai`. The repo's existing import-boundary test (`backend/agent/src/agent/import_boundary_test.go:931-1008`) admits unchanged — D-4.

### NFR-CCS-003 — Substrate preservation

No file under `backend/agent/src/agent/` is modified. The ten-file substrate list (`event_descriptor.go`, `stream_check.go`, `failure.go`, `sequence.go`, `event.go`, `go.mod`, `go.sum`, `Makefile`, `.golangci.yml`, `import_boundary_test.go`) is byte-unchanged. NFR-TLS-003.

### NFR-CCS-004 — Sub-millisecond in-memory operations at v1's request rate

Coarse `sync.Mutex` is sufficient; no `sync.RWMutex` complexity (locked fork). `Load` returns a defensive copy of the slice so a caller-side mutation cannot corrupt the in-memory state.

---

## Acceptance — proposal success criteria mapped to scenarios

| Acceptance criterion | Evidence |
|---|---|
| All 6 Gherkin scenarios pass; round-trip 2 turns → reload → 3rd-turn request carries both earlier exchanges in order | `S-CCS-001`, `S-CCS-004`, `S-CCS-005`, `S-CCS-006` |
| Cancelled / failed / later-turn tail behaviour | `S-CCS-002`, `S-CCS-003` |
| In-memory adapter micro-tests | `S-CCS-007`, `S-CCS-008`, `S-CCS-009` |
| Composition-root consult lands in the factory closure | `R-CCS-001`, `S-CCS-006` |
| Lock-free on a fast follow-up reload | `R-CCS-008`, `S-CCS-006` |
| Port surface closed; CH-07 cannot silently widen | `R-CCS-010` |

## Explicit non-requirements

| Not required here | Owner / reason |
|---|---|
| A database, Postgres adapter, migrations | CH-07 (`cachicamas-chat-store-adapter`) — `0005:721` |
| Browser resume surface, `/api/agent/conversations/:id/reload` | CH-08.1 (`cachicamas-chat-resume-in-browser`) — `0005:778` |
| Listing multiple conversations per participant | CH-08.2 widens the port to a list |
| Modifying any file under `backend/agent/src/agent/` | NFR-TLS-003 — substrate preservation |
| Modifying `openspec/specs/chat-archetype-contract/spec.md` or `openspec/specs/chat-package-boundary/spec.md` | D-5 — this is a new capability, not an amendment |
| Widening `backend/agent/src/agent/import_boundary_test.go`'s check 6 allowlist | D-4 — stdlib-only + `agent` + `ai` admits unchanged |
| Touching `frontend/**` | CH-05 (merged) wired the page; CH-08.1 mounts the reload endpoint |
| A second constructor `NewConversationWithHistory` | D-2 — `Config.InitialHistory` keeps the choice in the factory closure |

## Cross-references

- Doc 0005 § CH-06 charter (`docs/architecture/milestones/0005-cachicamas-chat-archetype-task-graph.md:711-722`) and the two leaves (`0005:724-751`, `0005:753-778`).
- The Gherkin feature files at `0005:728-748` (CH-06.1 record) and `0005:757-774` (CH-06.2 reload) are the authoritative scenario wording — the `S-CCS-NNN` scenario lines in this spec are transcribed verbatim.
- `chat/projection.go:67-72` — the terminal wire event site that owns the append point (D-6).
- `chat/registry.go:18`, `:69-107` — the factory signature and `GetOrCreate` plumb `participantID` through; CH-06 does not change the registry.
- `cmd/chat/main.go:164-166` — the composition-root factory closure that consults the store (D-8 locked fork).
- `backend/agent/src/agent/history.go:268` — `agent.NewSeededHistory`, the door `ExchangesToHistory` uses.
- `backend/agent/src/agent/import_boundary_test.go:931-1008` — check 6's allowlist admits the new file unchanged (D-4).
- `openspec/AGENTS.md` "Substrate preservation in `backend/agent` (NFR-TLS-003)" — the ten-file substrate list that CH-06 leaves byte-unchanged.
