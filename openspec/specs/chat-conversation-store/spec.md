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
