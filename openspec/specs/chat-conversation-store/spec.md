# Spec — Chat conversation store (`chat-conversation-store`)

> **Change**: `cachicamas-chat-conversation-store` · **CH-06** (Wave 1, 7 of 12) of [doc 0005](../../../docs/architecture/milestones/0005-cachicamas-chat-archetype-task-graph.md#ch-06--define-the-conversation-store-port-and-its-in-memory-adapter) (`0005:711-722`) · leaves CH-06.1 `[leaf]` (`0005:724-751`) · CH-06.2 `[leaf]` (`0005:753-778`)
> **Closes**: R-04 (define your own ports), R-16 (durable state independent of transport), register row 3 (home-directory sessions are not a layer rule), research finding 5 (the stream is not the store of record)
> **Status**: **new capability**, promoted verbatim to `openspec/specs/chat-conversation-store/spec.md` at archive. Per D-5, no promoted spec is amended.
> **Depends on**: CH-02 (`e3c717a4`). **Blocks:** CH-07 (`cachicamas-chat-store-adapter`), CH-08 (`cachicamas-chat-resume-in-browser`).
> **Format**: Given/When/Then + RFC 2119 per `openspec/config.yaml` `rules.specs`. Every scenario is independently verifiable — the Gherkin `S-CCS-NNN` rows are transcribed verbatim from `0005:728-775`; the `S-CCS-NNN` micro-test rows are direct sub-tests of `MemoryConversationStore` per the proposal's WU-1.
> **IDs**: requirements `R-CCS-NNN`, scenarios `S-CCS-NNN`, non-functional `NFR-CCS-NNN`. Append-only.
> **Prefix verification**: `CCS` was verified collision-free across `openspec/specs/`, `openspec/changes/` and `docs/architecture/milestones/0005-…md` (proposal D-8).
> **Evidence gate**: `cd backend/agent && make test` (with `-race`) green, uncached; `make lint`; `make build/chat`. Frontend suite **not** re-run — CH-06 writes no `frontend/` file (gate's mechanical form: `git diff --stat e3c717a4 HEAD -- frontend/` empty). A `(cached)` result is not evidence.
>
> **Note on length.** The `sdd-spec` skill sets a 650-word budget. This spec exceeds it deliberately, on the recorded precedent of `chat-archetype-contract/spec.md:16` and `chat-package-boundary/spec.md:13`: ten requirements, six Gherkin scenarios transcribed verbatim (per the skill's "MUST be independently verifiable" rule, scenario wording cannot compress without losing traceability to `0005:728-775`), and four NFRs with mechanical gates do not compress without dropping content `openspec/config.yaml` requires be independently verifiable.
>
> **Amended 2026-08-24 (CH-07, `cachicamas-chat-store-adapter`) by the apply executor**: this spec is amended in place by CH-07 to admit a second adapter (`PostgresConversationStore`) behind the closed R-CCS-010 port. The amendment is **additive only** — every R-CCS-001…R-CCS-010 requirement, every S-CCS-001…S-CCS-009 scenario, and every NFR-CCS-001…NFR-CCS-004 non-functional requirement is **byte-unchanged**. New identifiers: **R-CCS-011**, **R-CCS-012**, **NFR-CCS-005**, **NFR-CCS-006**, **S-CCS-010**, **S-CCS-011**, **S-CCS-012**, **S-CCS-013**, **S-CCS-014**. The D-5 "no promoted spec is amended" sentence above is overridden by the CH-07 amendment header which records that this is an additive, identifier-append-only change following the precedent of `agent-package-scaffold` R-AGP-003 / R-AGP-005 ("the amendment lands in the same commit as the first production import that requires it"). The D-5 record is preserved in the **Untemporal-invariant register** at the bottom of this file.
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

---

## Amendments added by CH-07 (2026-08-24)

### R-CCS-011 — A postgres adapter satisfies the closed two-method port

The system MUST expose a `PostgresConversationStore` type that implements `ConversationStore` (R-CCS-010) by providing `Append(participantID string, exchange Exchange) error` and `Load(participantID string) ([]Exchange, error)`. The type lives in `backend/agent/src/chat/store_postgres.go` (D-3 mirrors the in-memory adapter's port + adapter-in-one-file precedent). The two methods MUST produce semantically identical observable behaviour to `MemoryConversationStore` for the same inputs, as witnessed by the CH-06 scenario set running unchanged against both adapters (R-CCS-012, CH-07.2's first Gherkin scenario). The adapter MUST dial Postgres through `database/sql.Open("pgx", dsn)` (the `pgx/v5/stdlib` blank-import that lets `pgx` masquerade as a stdlib driver) and MUST use `pgxpool`-shaped connection acquisition only when explicit pool semantics are required (CH-07 v1 stays on the stdlib `*sql.DB` surface for parity with the existing DBA precedent at `backend/database_administrator/src/migration/postgres/driver.go`).

#### Scenarios

- **S-CCS-010** — a conversation written by one process is read by another (Gherkin verbatim, `0005:800-803`)
  - Given a conversation recorded through the database adapter
  - When it is read back through a separately constructed adapter over the same store
  - Then the record matches what was written, in order

- **S-CCS-013** — the adapter writes only tables this archetype owns (Gherkin verbatim, `0005:805-809`)
  - Given the merged migration
  - When the tables it creates or alters are enumerated
  - Then every one of them belongs to this archetype
  - And no table owned by another system appears

- **S-CCS-014** — two participants' conversations never mix (Gherkin verbatim, `0005:811-814`)
  - Given conversations recorded for two different participants
  - When each participant's conversations are read
  - Then neither set contains an entry belonging to the other

### R-CCS-012 — Cross-process round-trip is the binding acceptance

The CH-06 scenario set (`S-CCS-001` through `S-CCS-009`) MUST pass against both `MemoryConversationStore` and `PostgresConversationStore` with **scenario text unchanged** (CH-07.2's first Gherkin scenario at `0005:827-830`). The test harness MUST use a single scenario-list constant shared by both adapter test invocations — a copy of the scenarios is a fork, not a contract. The `INTEGRATION=1` env gate isolates the cross-process scenario (`S-CCS-010`); the other scenarios run against an in-process `*sql.DB` opened against a `:memory:`-equivalent or an INTEGRATION-1 Postgres, with the test choosing the surface by gate. The guard test (`S-CCS-011`) runs unconditionally (no env gate).

#### Scenarios

- **S-CCS-011** — the port's scenarios run unchanged against both adapters (Gherkin verbatim, `0005:827-830`)
  - Given the conversation store scenarios from CH-06
  - When they run against the in-memory adapter and against the database adapter
  - Then both runs pass with the scenario text unchanged

- **S-CCS-012** — the guard bites when a caller reaches past the port (Gherkin verbatim, `0005:832-835`)
  - Given a scratch file in the archetype naming the database adapter directly instead of the port
  - When the check runs
  - Then it fails naming the file and the bypassed port

### NFR-CCS-005 — Postgres tables belong to this archetype

The merged migration MUST create exactly two tables: `chat_conversations` (the participant id keyed row set, holding `participant_id text PRIMARY KEY, created_at timestamptz NOT NULL DEFAULT now(), updated_at timestamptz NOT NULL DEFAULT now()`) and `chat_exchanges` (one row per `Append`, holding `participant_id text NOT NULL REFERENCES chat_conversations(participant_id) ON DELETE CASCADE, position integer NOT NULL, prompt_text text NOT NULL, assistant_text text NOT NULL, partial boolean NOT NULL, terminal_kind text NOT NULL CHECK (terminal_kind IN ('completed','cancelled','failed')), failure_category text NOT NULL DEFAULT '', finish_reason text NOT NULL DEFAULT '', message_ids jsonb NOT NULL DEFAULT '[]'::jsonb, recorded_at timestamptz NOT NULL DEFAULT now(), PRIMARY KEY (participant_id, position)`). Both table names MUST begin with `chat_`. The migration MUST NOT create or alter any table whose name does not begin with `chat_` (ADR 0009 § D6: *each business system owns its own tables; no archetype writes to another system's schema*). The chat-archetype import-boundary allowlist at `backend/agent/src/agent/import_boundary_test.go:931-1008` MUST be widened in the same commit to admit `github.com/jackc/pgx/v5/stdlib` and `github.com/pressly/goose/v3` (per R-AGP-003's "amendment lands in the same commit as the first production import that requires it" rule, recorded in `chat-package-boundary` `Untemporal-invariant register`).

### NFR-CCS-006 — Migration is forward-only and append-only

The single v1 migration `backend/agent/src/chat/migrations/0001_init.sql` MUST contain only `CREATE TABLE`, `CREATE INDEX`, `CREATE SEQUENCE`, `COMMENT`, and `INSERT` statements (no `DROP`, no `ALTER` of pre-existing tables, no `TRUNCATE`). The migration runner (a chat-package-local `goose.Provider` instance, mirroring `backend/database_administrator/src/migration/runner.go` but bound to `database/sql.DB`) MUST refuse to apply a migration whose every line does not match this allowlist; this is enforced by a static check at provider construction time (a test against a planted defeat migration). Down-migrations are explicitly out of scope — rollback is by code revert (the seam survives) and operator-driven DB inspection, not by an automated reverse path.

## Acceptance — proposal success criteria mapped to scenarios (amended)

| Acceptance criterion | Evidence |
|---|---|
| All 6 Gherkin scenarios pass; round-trip 2 turns → reload → 3rd-turn request carries both earlier exchanges in order | `S-CCS-001`, `S-CCS-004`, `S-CCS-005`, `S-CCS-006` |
| Cancelled / failed / later-turn tail behaviour | `S-CCS-002`, `S-CCS-003` |
| In-memory adapter micro-tests | `S-CCS-007`, `S-CCS-008`, `S-CCS-009` |
| Composition-root consult lands in the factory closure | `R-CCS-001`, `S-CCS-006` |
| Lock-free on a fast follow-up reload | `R-CCS-008`, `S-CCS-006` |
| Port surface closed; CH-07 cannot silently widen | `R-CCS-010` |
| **CH-07.1** cross-process read round-trips | `S-CCS-010` (INTEGRATION=1) |
| **CH-07.1** archetype owns every created table | `S-CCS-013`, `NFR-CCS-005` |
| **CH-07.1** two participants never mix | `S-CCS-014` |
| **CH-07.2** CH-06 scenarios pass against both adapters, unchanged text | `S-CCS-011` |
| **CH-07.2** guard bites with file+port failure message | `S-CCS-012` |
| **CH-07** migration is forward-only and append-only | `NFR-CCS-006` |

## Explicit non-requirements (amended)

| Not required here | Owner / reason |
|---|---|
| A database, Postgres adapter, migrations | ~~CH-07 (`cachicamas-chat-store-adapter`) — `0005:721`~~ **now CH-07** |
| Browser resume surface, `/api/agent/conversations/:id/reload` | CH-08.1 (`cachicamas-chat-resume-in-browser`) — `0005:778` |
| Listing multiple conversations per participant | CH-08.2 widens the port to a list |
| Modifying any file under `backend/agent/src/agent/` | NFR-TLS-003 — substrate preservation (allowlist amendment is the **only** change permitted, per R-AGP-003) |
| Modifying `openspec/specs/chat-archetype-contract/spec.md` or `openspec/specs/chat-package-boundary/spec.md` | D-5 — CH-07 amends `chat-conversation-store` only |
| Widening `backend/agent/src/agent/import_boundary_test.go`'s check 6 allowlist beyond `pgx/v5/stdlib` + `pressly/goose/v3` | NFR-CCS-005 — only the two deps required by R-CCS-011 / NFR-CCS-005 are admitted |
| Touching `frontend/**` | CH-05 (merged) wired the page; CH-08.1 mounts the reload endpoint |
| A second constructor `NewConversationWithHistory` | D-2 — `Config.InitialHistory` keeps the choice in the factory closure |
| **Down-migrations or destructive schema changes** | NFR-CCS-006 — forward-only; rollback by code revert |
| **Connection pooling tuning beyond `pgxpool` defaults** | Out of CH-07 scope; CH-04 owns env vars |
| **Read replicas / write splitting** | Out of scope; v1 single connection string |
| **Conversation ACL enforcement** | CH-08.2 widens the port to a list; ACL is the seam |
| **Tool: deferred-but-related — tool-call history persistence** | Out of scope for chat; conversation tool-call history is the Layer 2 harness's seam, not the chat archetype's |

## Cross-references (amended)

- Doc 0005 § CH-07 charter (`docs/architecture/milestones/0005-cachicamas-chat-archetype-task-graph.md:787-800`) and the two leaves (`0005:793-815`, `0005:820-838`).
- The Gherkin feature files at `0005:797-815` (CH-07.1) and `0005:824-836` (CH-07.2) are the authoritative scenario wording — the `S-CCS-010`…`S-CCS-014` scenario lines in this spec are transcribed verbatim.
- `chat/store.go:102-115` — the closed two-method port `ConversationStore`; CH-07's postgres adapter implements the same two methods.
- `chat/store_postgres.go` (new) — the postgres adapter. Implements the same two methods, dials via `database/sql.Open("pgx", dsn)`, owns the connection lifecycle in its constructor (`NewPostgresConversationStore(ctx, dsn) (*PostgresConversationStore, func() error, error)` — the returned closer must be invoked by the composition root after OTel shutdown).
- `chat/migrations/0001_init.sql` (new) — the single forward-only migration. Two tables only.
- `chat/migrator/` (new) — chat-package-local goose runner, mirroring `backend/database_administrator/src/migration/runner.go`'s `GooseRunner` shape but bound to `database/sql.DB` (not the `pgxpool`-shaped one) and admitting only forward-only migrations (NFR-CCS-006).
- `chat/store_guard_test.go` (new) — CH-07.2 guard. Walks `backend/agent/src/chat/**/*.*.` excluding tests and the adapter's own declaration file, fails the build with a message naming the file and the bypassed port when a non-test file imports `chat.NewPostgresConversationStore` or names `*chat.PostgresConversationStore` outside the adapter's own file.
- `cmd/chat/main.go` — gains a `CACHICAMAS_CHAT_STORE_DSN` env read and an adapter-swap line replacing `chat.NewMemoryConversationStore()` (the seam CH-07 preserved). The factory closure body is unchanged.
- `backend/agent/src/agent/import_boundary_test.go:931-971` — allowlist widens per R-AGP-003 (NFR-CCS-005).
- `docs/adr/0010-add-pgx-and-goose-to-backend-agent.md` — the ADR justifying the new top-level dependencies; same-PR per `openspec/AGENTS.md` Hard Rule 5.

## Untemporal-invariant register (CH-07 addition)

The D-5 record ("no promoted spec is amended") is preserved here as an **absence marker**, the same way `chat-package-boundary` records the analogous invariant. The amendment is additive only — see the CH-07 amendment header at the top of this file. R-AGP-003 / R-AGP-005 are the precedent this entry cites.

---

# Amendments added by CH-08 (2026-08-24) — `cachicamas-chat-resume-in-browser`

> **Change**: `cachicamas-chat-resume-in-browser` · **CH-08** (Wave 2) of [doc 0005](../../../docs/architecture/milestones/0005-cachicamas-chat-archetype-task-graph.md#ch-08--resume-a-conversation-in-the-browser) (`0005:842-899`) · leaves CH-08.1 `[leaf]` (`0005:855-876`) · CH-08.2 `[leaf]` (`0005:878-899`)
> **Amends**: this spec (`chat-conversation-store`) — **additive only**; identical pattern to the CH-07 amendment above. Identifier-append-only per the CH-07 precedent. The widening is **third method on the same declaration**, not a replacement of R-CCS-010's two-method surface.

## ADDED Requirements

### R-CCS-013 — `ConversationStore` widens with a third additive method `List`

The `ConversationStore` interface (R-CCS-010) MUST be extended by adding a third method `List(participantID string) ([]ConversationSummary, error)` to the same declaration. The two existing methods `Append` and `Load` MUST remain byte-unchanged. `MemoryConversationStore` (R-CCS-002) and `PostgresConversationStore` (R-CCS-011) MUST both implement `List`. `MemoryConversationStore.List` MUST iterate the `m` map and return `[]ConversationSummary` (possibly empty) without mutating map state; `PostgresConversationStore.List` MUST execute `SELECT participant_id, updated_at, COALESCE((SELECT MAX(position)+1 FROM chat_exchanges WHERE participant_id = $1), 0) AS turn_count FROM chat_conversations WHERE participant_id = $1 ORDER BY updated_at DESC` against the existing schema. The widening is **additive** — it does not replace R-CCS-010's two-method declaration; future widens MUST follow the same additive pattern.

### R-CCS-014 — `ConversationSummary` is the port's list projection

The system MUST expose a `ConversationSummary` struct carrying `ConversationID string`, `LastActivityAt time.Time`, and `TurnCount int`. `ConversationSummary` is defined in `chat-conversation-store` because it is the port's projection (the rule: the port owns its own projections). The wire DTO `ConversationSummaryDTO` is defined in the resume spec (`cachicamas-chat-resume-in-browser`, R-CRI-004) and is a pure transport projection — the wire MUST NOT invent fields beyond what `ConversationSummary` carries.

### NFR-CCS-007 — `List` is a participant-scoped read under identity middleware

`ConversationStore.List` MUST be a participant-scoped read. Implementations MUST NOT return rows for any other participant even under a corrupted or missing identity. The guard pattern: identity middleware runs the participant through `getIdentity(c)` (R-CHS-004.b shape) before `List` is called; the handler uses the resolved id, not any URL or header value.

### Scenarios

#### Scenario: S-CCS-015 — reloading the page restores the conversation (Gherkin verbatim, `0005:862-866`)

- Given an employee who has completed two turns and reloads the page
- When the page loads
- Then both exchanges are shown in their original order
- And the input accepts a new prompt that continues the same conversation

#### Scenario: S-CCS-016 — a reload during a streaming turn shows what was recorded (Gherkin verbatim, `0005:868-872`)

- Given an employee who reloads while a turn is streaming
- When the page loads
- Then the exchanges recorded before the reload are shown
- And the page does not claim the turn is still streaming

#### Scenario: S-CCS-017 — a participant sees their own conversations and no others (Gherkin verbatim, `0005:885-889`)

- Given two participants who have each held conversations
- When one of them requests their list
- Then the list contains only their own conversations
- And each entry identifies its conversation well enough to open it

#### Scenario: S-CCS-018 — a participant with no conversations gets an empty list (Gherkin verbatim, `0005:891-895`)

- Given an authenticated participant who has never held a conversation
- When they request their list
- Then the list is empty
- And the response is a success rather than a not-found

---

# Untemporal-invariant register (CH-08 addition)

The D-5 / CH-07-2 records above are preserved here as **absence markers**. The CH-08 amendment is additive only — see the CH-08 amendment header at the top of this file. R-CCS-013 widens the port to N+1 methods, not "opens" it; future widens keep the additive pattern.

---

# CH-09 amendment — `cachicamas-chat-tool-source`

> **Change**: `cachicamas-chat-tool-source` · **CH-09** (Wave 3, 10 of 12) of [doc 0005](../../../docs/architecture/milestones/0005-cachicamas-chat-archetype-task-graph.md#ch-09--offer-tools-through-a-tool-source-port) (`0005:923-934`)
> **Amends**: this spec (`chat-conversation-store`) — **additive only**. **R-CCS-001..014 byte-unchanged**; **S-CCS-001..018 byte-unchanged**; **NFR-CCS-001..007 byte-unchanged**. New identifiers: **R-CCS-015**, **R-CCS-016**, **NFR-CCS-008**, **S-CCS-019**, **S-CCS-020**, **S-CCS-021**, **S-CCS-022**. Identifier-append-only per CH-07 / CH-08 precedent (`chat-conversation-store/spec.md:9`, `chat-conversation-store/spec.md:14`). The widening is **two new fields on the existing `Exchange` struct**, not a replacement of R-CCS-010's two-method surface, and not a replacement of R-CCS-007's append-before-`inFlight`-clear ordering.

## ADDED Requirements

### R-CCS-015 — `chat.Exchange` widens with `[]ToolCallRecord` and `[]ToolResultRecord`

The `chat.Exchange` struct (`backend/agent/src/chat/store.go:41-50`) MUST widen additively with two new fields: `ToolCalls []ToolCallRecord` and `ToolResults []ToolResultRecord`. `ToolCallRecord` MUST carry `WireCallID string`, `Tool string`, and `Arguments string`. `ToolResultRecord` MUST carry `WireCallID string`, `Tool string`, `Outcome string` (one of `"success"`, `"result_failure"`, `"execution_failure"` — the closed vocabulary at `backend/agent/src/agent/tool_event.go:227-246`), `Content string`, and `FailureCategory string` (non-empty only when `Outcome == "execution_failure"`, mirroring R-CCP-008 / D6). The eight pre-existing fields MUST remain byte-unchanged (R-CCS-001 / R-CCS-007). The widening is **additive**; future widens follow the same pattern. Source of truth for the field shape: `cachicamas-chat-tool-source/spec.md` `R-CTS-006`.

### R-CCS-016 — Reload replays tool-call records in issuance order; the participant's transcript on reload is byte-equal to the live transcript

A reloaded exchange's `ToolCalls` and `ToolResults` slices MUST carry the records in the same issuance order in which the chat projector (`backend/agent/src/chat/projection.go`) emitted the corresponding `ToolCallStart` / `ToolResult` wire events. The reload surface at `GET /api/agent/conversations/:id` (CH-08.1, R-CRI-001) MUST return these slices intact; the chat page's `useChatStream.reset(entries)` seed MUST render tool entries from them after the assistant said entry (per `cachicamas-chat-tool-source/spec.md` `S-CTS-019`, mirrored as `S-FCL-014`). A re-append preserves them (per R-CCS-001's `Append` contract and `MemoryConversationStore.Append`'s `ex.Position = len(...)` discipline at `store.go:174-181`).

### NFR-CCS-008 — Defensive copy on `Load` extends to the new fields

The defensive-copy semantics from NFR-CCS-004 MUST extend to the two new fields: `MemoryConversationStore.Load` (`store.go:187-197`) MUST return slices whose `ToolCalls` and `ToolResults` are byte-equal to the stored slices AND caller-side mutation of either field MUST NOT corrupt the store's state. `PostgresConversationStore.Load` MUST apply the same defensive-copy discipline when materialising from `chat_tool_calls` + `chat_tool_results` (the sibling tables per `cachicamas-chat-tool-source/spec.md` `R-CTS-006`). Carries NFR-CCS-004 forward; NFR-CCS-006 (forward-only migration) governs the sibling-table migration.

### Scenarios

#### Scenario: S-CCS-019 — a turn with two tool calls round-trips through the store in issuance order (Gherkin verbatim, explore #3952)

- Given a turn whose projection yields two tool calls and two results
- When `Load(participantID)` reads back
- Then each `Exchange.ToolCalls` and `Exchange.ToolResults` carries the records in issuance order
- And a re-append preserves them

#### Scenario: S-CCS-020 — defensive copy on `Load` extends to the new fields (Gherkin verbatim, explore #3952)

- Given an in-memory adapter recording `Exchange{ToolCalls: [c1, c2], ToolResults: [r1, r2]}`
- When `Load(p)` is called
- Then the returned slice's `ToolCalls` / `ToolResults` fields are byte-equal to the input AND the caller's later mutation does not corrupt the store (NFR-CCS-008 carries NFR-CCS-004 forward)

#### Scenario: S-CCS-021 — postgres adapter round-trips tool records across processes (Gherkin verbatim, explore #3952; gated `INTEGRATION=1`)

- Given a postgres adapter recording `Exchange{ToolCalls: [...], ToolResults: [...]}`
- When a separately constructed adapter reads back
- Then both arrays match the input in order (cross-process round-trip, CH-07.1 precedent)

#### Scenario: S-CCS-022 — tool records never leak across participants (Gherkin verbatim, explore #3952)

- Given a recording under participant `p1` with tool calls
- When `Load("p2")` is called
- Then `p2`'s slice contains no tool calls from `p1` (R-CHS-004.b shape preserved)

## Untemporal-invariant register (CH-09 addition)

The D-5 / CH-07-2 / CH-08 records above are preserved here as **absence markers**. The CH-09 amendment is additive only — see the CH-09 amendment header at the top of this file. R-CCS-015 widens `Exchange` with two new fields; R-CCS-016 widens the reload contract; the port's two-method surface (R-CCS-010, extended to N+1 by R-CCS-013) is not replaced. Future widens keep the additive pattern.

---

# CH-10 amendment — `cachicamas-chat-permission`

> **Change**: `cachicamas-chat-permission` · **CH-10** (Wave 3, 11 of 12) of [doc 0005](../../../docs/architecture/milestones/0005-cachicamas-chat-archetype-task-graph.md#ch-10--approve-a-tool-call-from-the-browser) (`0005:936-947`)
> **Amends**: this spec (`chat-conversation-store`) — **additive only**. **R-CCS-001…016 byte-unchanged**; **S-CCS-001…022 byte-unchanged**; **NFR-CCS-001…008 byte-unchanged**. New identifiers: **R-CCS-017**, **R-CCS-018**, **NFR-CCS-009**, **S-CCS-023**, **S-CCS-024**, **S-CCS-025**. Identifier-append-only per CH-07 / CH-08 / CH-09 precedent (`chat-conversation-store/spec.md:9, 14`, plus the three preceding amendment headers at `:14`, `:304-307`, `:361-364`). The widening is **a fourth method on the same port declaration** (R-CCS-017, additive per the R-CCS-013 N+1 precedent) and **one new optional field on `Exchange`** (R-CCS-018, additive per the R-CCS-015 / R-CCS-016 CH-09 precedent). No requirement on the existing port is replaced.
>
> **F-CPM-002/003 alignment** (recorded, design-time fixed in `cachicamas-chat-permission/spec.md` § "Spec defect F-CPM-002/003 resolution"): no spec-text changes to existing requirements R-CCS-001…016 are required. R-CCS-010/011/012/013/014/015/016 carry forward byte-clean. The chat projector suppression rule that closes F-CPM-002/003 lives in `chat/projection.go` and `use-chat-stream.ts` per R-CPM-008; this spec's contribution is the persistence widening that makes the persisted `Exchange` carry the per-`wireCallId` decision records.

## ADDED Requirements

### R-CCS-017 — `ConversationStore` widens with a fourth additive method `UpdateSummary`

The `ConversationStore` interface (R-CCS-010, extended to N+1 by R-CCS-013, extended to N+2 here) MUST be widened by adding a fourth method `UpdateSummary(participantID string, summary string) error` to the same declaration. The three existing methods `Append`, `Load`, and `List` MUST remain byte-unchanged. `MemoryConversationStore` (R-CCS-002) and `PostgresConversationStore` (R-CCS-011) MUST both implement `UpdateSummary`. `MemoryConversationStore.UpdateSummary` MUST look up the conversation by `participantID`, store the `summary` string on the `ConversationSummary` projection (R-CCS-014), and return `chat.ErrConversationNotFound` if no conversation is registered under that id (mirrors `Load`'s not-found shape from R-CCS-001 / S-CCS-006). `PostgresConversationStore.UpdateSummary` MUST execute `UPDATE chat_conversations SET summary = $1, updated_at = NOW() WHERE participant_id = $2` against the existing schema. The forward-only migration `backend/agent/src/chat/migrations/0003_summarize.sql` MUST add `summary TEXT` (nullable) to `chat_conversations` (NFR-CCS-006 affordance; an `ADD COLUMN nullable` is the documented forward-only allowance per `openspec/AGENTS.md` "Substrate preservation in `backend/agent`" paragraph on CH-07's `pgx/v5` admission precedent). The widening is **additive** — it does not replace R-CCS-010's two-method surface, R-CCS-013's three-method surface, or R-CCS-015/016's `Exchange` field widenings; future widens MUST follow the same additive pattern. Source of truth for the method shape: `cachicamas-chat-permission/spec.md` R-CPM-006 / Q1 resolution at decision #3983.

### R-CCS-018 — `chat.Exchange` widens with `[]PermissionDecisionRecord`

The `chat.Exchange` struct (`backend/agent/src/chat/store.go:41-50`, already widened by CH-09 to 12 fields with `ToolCalls`/`ToolResults` per R-CCS-015) MUST widen additively with a new field `PermissionDecisions []PermissionDecisionRecord`. `PermissionDecisionRecord` MUST carry `WireCallID string`, `Tool string`, and `Outcome string`. `Outcome` MUST be one of the chat wire's closed 2-value vocabulary `"allow_once" | "deny"` (the D-12 Layer-2 outcome collapse has already happened at the projector per `cachicamas-chat-permission/spec.md` R-CPM-003; persistence only sees the collapsed form, never Layer 2's 4-value vocabulary). The twelve pre-existing fields MUST remain byte-unchanged. The forward-only sibling table `chat_permission_decisions` MUST be created by `backend/agent/src/chat/migrations/0004_permission_decisions.sql`, keyed by `(participant_id, exchange_position, position)` with FK to `chat_exchanges (participant_id, position) ON DELETE CASCADE` (mirrors `chat/migrations/0002_tool_records.sql:60` template). The migration MUST include a `chat_permission_decisions_lookup_idx` index on `(participant_id, exchange_position)` to keep the per-exchange lookup bounded. `MemoryConversationStore.Append` MUST round-trip the new field; `PostgresConversationStore.Append` MUST INSERT sibling-table rows in the same transaction as the `chat_exchanges` INSERT (mirrors CH-09's `chat_tool_calls`/`chat_tool_results` sibling-table transaction at `store_postgres.go:263-303`). Source of truth for the field shape: `cachicamas-chat-permission/spec.md` R-CPM-006.

### NFR-CCS-009 — Defensive copy on `Load` extends to `PermissionDecisions`

The defensive-copy semantics from NFR-CCS-004 (CH-06) and NFR-CCS-008 (CH-09 carry) MUST extend to the new `PermissionDecisions` field: `MemoryConversationStore.Load` (`backend/agent/src/chat/store.go:249-264`) MUST return exchanges whose `PermissionDecisions` slice is byte-equal to the stored slice AND caller-side mutation of any record in the slice MUST NOT corrupt the store's state. The implementation MUST use a `copyPermissionDecisionRecords` helper in the same idiom as `copyToolCallRecords`/`copyToolResultRecords` from CH-09. `PostgresConversationStore.Load` MUST materialise a fresh slice from the `chat_permission_decisions` sibling-table rows; the returned slice is fresh per call (no aliasing of an internal buffer). Carries NFR-CCS-008 / NFR-CCS-004 forward. Mirrors `cachicamas-chat-permission/spec.md` NFR-CPM-004 / S-CPM-023.

## Scenarios

### S-CCS-023 — `UpdateSummary` round-trips through the store; load returns the summary verbatim (Gherkin verbatim, explore #3985)

- Given an in-memory adapter recording no summary under participant `p`
- When `UpdateSummary(p, "the participant asked about cats")` is called
- And `Load(p)` is called
- Then the returned conversation's `ConversationSummary.Summary` (or the `ConversationStore`-level read for the summary DTO) is byte-equal to `"the participant asked about cats"`
- And a participant `p2` who has never called `UpdateSummary` sees `Load(p2).Summary == ""` (the empty / NULL column path; cross-participant isolation mirrors R-CHS-004.b)

### S-CCS-024 — defensive copy on `Load` extends to `PermissionDecisions` (Gherkin verbatim, explore #3985; mirrors CH-09 S-CCS-020)

- Given an in-memory adapter recording `Exchange{PermissionDecisions: [d1, d2]}` for participant `p` (with `d1.Outcome == "allow_once"`)
- Then `Load(p)` returns the slice byte-equal to `[d1, d2]`
- And when the caller mutates the returned `result[0].Outcome = "deny"`, a subsequent `Load(p)` returns the original `[d1, d2]` with `d1.Outcome == "allow_once"` unchanged — caller-side mutation does NOT corrupt the store (NFR-CCS-009 carries NFR-CCS-008 / NFR-CCS-004 forward)

### S-CCS-025 — permission decisions never leak across participants (Gherkin verbatim, explore #3985; mirrors CH-09 S-CCS-022)

- Given a recording under participant `p1` with `PermissionDecisions: [d1]`
- When `Load("p2")` is called
- Then `p2`'s slice contains no permission decisions from `p1` (R-CHS-004.b shape preserved)
- And the postgres cross-process variant is gated `INTEGRATION=1` (CH-07.1 precedent — `S-CCS-021` mirror)

## Untemporal-invariant register (CH-10 addition)

The D-5 / CH-07-2 / CH-08 / CH-09 records above are preserved here as **absence markers**. The CH-10 amendment is additive only — see the CH-10 amendment header at the top of this section. R-CCS-017 widens the port to N+2 methods (not "opens" it; R-CCS-013's N+1 widening is preserved); R-CCS-018 widens `Exchange` with one new field (CH-09's R-CCS-015/016 two-field widening is preserved). The closed port's two-method surface from R-CCS-010 is **not replaced** — every existing requirement, scenario, and NFR from R-CCS-001…016 carries forward byte-clean. Future widens keep the additive pattern.
