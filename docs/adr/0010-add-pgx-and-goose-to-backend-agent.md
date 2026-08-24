# ADR 0010 — Add `pgx/v5` and `pressly/goose/v3` as top-level deps in `backend/agent`

> Status: **Accepted** (2026-08-24). Implements the dependency admission `openspec/AGENTS.md`
> Hard Rule 5 and `openspec/config.yaml` `apply.no-new-top-level-deps-without-an-ADR` require.
> Implements: CH-07 (`cachicamas-chat-store-adapter`) — closes R-08.
> Companion: this ADR is merged in the same PR as the first production import that requires
> either dep, satisfying the "merged in this PR" form of Hard Rule 5 (proposal `obs-a55f0c15be013cff`,
> task T-02 in the apply-phase task list).

---

## Resolved TOC

- [Context](#context)
- [Decision](#decision)
- [Alternatives considered](#alternatives-considered)
- [Consequences](#consequences)
- [References](#references)

---

## Context

CH-07 (`cachicamas-chat-store-adapter`) needs a real database adapter behind the
`chat.ConversationStore` port the chat archetype owns
([`openspec/specs/chat-conversation-store/spec.md`](../../../openspec/specs/chat-conversation-store/spec.md)).
CH-06 shipped the closed two-method port and an in-memory adapter; CH-07 swaps in the real
postgres adapter behind the same port, owns its tables per [ADR 0009 § D6](./0009-redefine-cachicamas-as-a-multiplayer-agentic-system.md#d6--each-business-system-owns-its-own-tables),
and proves the swap was a swap with a file-tree guard.

Two top-level Go dependencies are required:

1. **A Postgres driver.** The driver must support `database/sql` so the existing Layer 3 code
   keeps the stdlib surface (per [ADR 0005 § D1 row 2](../0005-promote-agent-stack-to-own-module.md#d1--dependency-rule-v2),
   Layer 3 reaches Layer 2 through ports, not vendor SDKs). The DBA precedent at
   `backend/database_administrator/src/migration/postgres/driver.go:30-33` blank-imports
   `_ "github.com/jackc/pgx/v5/stdlib"` for this exact purpose; the precedent is in production.
2. **A migration runner.** The runner must run forward-only SQL migrations from an embedded
   virtual filesystem, with a bookkeeping table the chat archetype owns (NFR-CCS-005 +
   NFR-CCS-006 require the `chat_` table-namespace prefix and the forward-only allowlist).
   `github.com/pressly/goose/v3` is the project standard — `backend/database_administrator`
   already uses it, the devops story is identical, no second tool to learn.

The `backend/agent` module currently has neither dependency. The driver's absence means the
adapter cannot dial; the runner's absence means the migrations cannot be applied. CH-07 needs
both; the `apply.no-new-top-level-deps-without-an-ADR` rule requires this ADR.

---

## Decision

Admit as new top-level deps in `backend/agent`:

| Module | Pin | Why |
|---|---|---|
| `github.com/jackc/pgx/v5` | `v5.10.0` | Postgres driver (matches `backend/database_administrator/go.mod`) |
| `github.com/jackc/pgx/v5/stdlib` | (same module as above; blank-imported) | Registers `"pgx"` as a `database/sql` driver so `sql.Open("pgx", dsn)` works |
| `github.com/pressly/goose/v3` | `v3.27.1` | Migration runner (matches `backend/database_administrator/go.mod`) |

The pins mirror `backend/database_administrator/go.mod` byte-for-byte so the project's
Postgres story stays on one driver version and one goose version across all consumers.

---

## Alternatives considered

1. **`lib/pq` (older driver).** Rejected.
   * In maintenance mode — losing the pgx binary protocol, no batch-mode improvements.
   * Splits the driver story with DBA (DBA already uses pgx; chat would carry `lib/pq`).
   * Same ADR requirement, worse fit.

2. **Defer to the Database Administrator archetype (call it over MCP).** Rejected on timing.
   * [ADR 0009 § D5](./0009-redefine-cachicamas-as-a-multiplayer-agentic-system.md#d5--the-database-administrator-archetype-is-planned)
     records the DBA archetype as **PLANNED, NOT BUILT**. There is no HTTP/MCP surface today
     to call.
   * Calling DBA over MCP for migrations would couple chat's startup path to a service that
     does not exist. The chat binary must boot today, against its own embedded SQL, with
     its own bookkeeping table.
   * The chat archetype owning its migrations is required by ADR 0009 § D6 regardless —
     *each business system owns its own tables* — so even once DBA exists, chat still owns
     `chat_*` and the DBA archetype is the broker for cross-archetype migrations, not the
     holder of chat's own.

3. **A shared module extracting pgx for both `backend/agent` and `backend/database_administrator`.**
   Rejected as a refactor too big for this change.
   * No precedent. `go.work` is editor-ergonomics only — modules do not import each other
     (ADR 0005 § D1 row 2).
   * Creating a third module to host pgx adds an import edge every other module would have
     to reason about; the cost exceeds the benefit for the one new dep consumer.
   * A separate ADR would record this if it is ever pursued; this ADR does not authorise it.

4. **`pgxpool` directly without `database/sql`.** Rejected for v1.
   * DBA's stack is stdlib-shaped (`sql.Open("pgx", dsn)` at
     `backend/database_administrator/src/migration/postgres/driver.go:171`); CH-07 mirrors
     that to keep the devops story identical — one DSN, one driver, one OTel SDK.
   * `pgxpool`'s trace hooks are a richer integration CH-07 does not need at v1. The
     proposal's R-7 explicitly defers `pgxpool`'s own hooks. CH-08+ can swap the dialer
     behind the same constructor signature.

5. **Hand-roll a migration runner (no `pressly/goose/v3`).** Rejected.
   * Goose is the project's standard runner — DBA already uses it, the operator's mental
     model is unified, the bookkeeping-table name is the only per-archetype variation.
   * A hand-rolled runner must independently implement: lexicographic ordering,
     idempotent re-run, advisory-lock-or-equivalent for the multi-replica case, and the
     rollback bookkeeping. None of these are CH-07's deliverable.

---

## Consequences

### Positive

- **Mirrors the proven DBA precedent.** `pgx/v5` + `pressly/goose/v3` is already a
  production pair in `backend/database_administrator`. Operators reading this ADR will
  recognise the move.
- **Self-contained chat package.** Chat owns its tables, its migrations, its bookkeeping
  table (`chat_schema_migrations`, distinct from DBA's `schema_migrations`), and its
  migration runner. No cross-module dependency at runtime.
- **Closed two-method port stays closed.** The postgres adapter implements the same
  `Append` and `Load` methods the in-memory adapter already satisfies
  (R-CCS-010, R-CCS-011). No caller reaches around the port.
- **Forward-only migrations are enforced at provider construction time.** The runner
  refuses any migration whose every line does not match the
  `CREATE TABLE | CREATE INDEX | CREATE SEQUENCE | COMMENT | INSERT` allowlist (NFR-CCS-006).
  This replaces the `pressly/goose/v3/lock` advisory-lock pattern DBA uses — chat does
  not need `lock` for v1 because the chat binary's migrator runs before the listener
  binds and before any conversation can call `Append`. Single-process startup means no
  two replicas can race the migration.

### Negative / risk

- **Two new top-level deps.** `pgx/v5` and its `stdlib` subpackage plus `pressly/goose/v3`
  appear in `backend/agent/go.mod` and `go.sum`. Both pin to the versions in DBA's
  `go.mod`; a future version bump on either side needs to be a paired change.
- **The chat-owned migration runner is intentionally simpler than DBA's.** No advisory
  lock. The safety guarantee is the forward-only allowlist, not the lock. If CH-07.1's
  acceptance scenario is ever exercised against two chat replicas starting simultaneously,
  the `chat_schema_migrations` row will be inserted twice if both pass the allowlist check
  — postgres will reject the second insert as a unique-key violation, and one of the
  replicas will fail to boot. **Operator discipline** (start one replica at a time during
  migrations) is the documented workaround for v1; the DBA-style lock is recorded as a
  deferred capability, not a v1 requirement.
- **Allowlist widening in `backend/agent/src/agent/import_boundary_test.go`** (NFR-CCS-005's
  allowlist amendment clause). The chat archetype allowlist at lines 931–971 admits
  `github.com/jackc/pgx/v5/stdlib` and `github.com/pressly/goose/v3`. This is the **only**
  change to the ten-file substrate list under NFR-TLS-003 — per agent-package-scaffold
  R-AGP-003's same-commit rule, the widening lands with the first production import that
  requires it.

### Neutral

- **Rollback is a code revert.** After revert, `cmd/chat/main.go:165` falls back to
  `chat.NewMemoryConversationStore()`; the migration is forward-only so the `chat_*`
  tables can be left in place with no harm; DBA's `schema_migrations` is unaffected.
  Re-deploying CH-07 after a revert picks up where it left off.
- **`pgxpool` is not admitted in v1.** It is recorded as a deferred capability (out of
  CH-07 scope). The constructor returns a closer for `*sql.DB`; CH-08+ can swap the
  dialer without changing the constructor's signature.

---

## References

- [ADR 0009 § D5](./0009-redefine-cachicamas-as-a-multiplayer-agentic-system.md#d5--the-database-administrator-archetype-is-planned),
  [§ D6](./0009-redefine-cachicamas-as-a-multiplayer-agentic-system.md#d6--each-business-system-owns-its-own-tables) — the
  constraints this ADR operates under.
- [ADR 0005 § D1](../0005-promote-agent-stack-to-own-module.md#d1--dependency-rule-v2) row 2 — Layer 2 / Layer 3 reach
  Layer 1 through ports, not vendor SDKs; the stdlib shape this ADR preserves.
- `openspec/AGENTS.md` "Hard rules" item 5 — "New top-level dependency ⇒ ADR first".
- `openspec/config.yaml` `apply.no-new-top-level-deps-without-an-ADR` — the gate this
  ADR satisfies.
- Doc 0005 § CH-07 charter (`docs/architecture/milestones/0005-cachicamas-chat-archetype-task-graph.md:786-791`)
  — the deliverable this ADR enables.
- DBA precedent: `backend/database_administrator/src/migration/postgres/driver.go:30-33`
  (blank-imports `_ "github.com/jackc/pgx/v5/stdlib"`), `backend/database_administrator/src/migration/runner.go:25`
  (imports `github.com/pressly/goose/v3`). The precedent this ADR mirrors verbatim.
- CH-07 proposal: Engram `obs-a55f0c15be013cff` (`sdd/cachicamas-chat-store-adapter/proposal`).
- CH-07 spec: Engram `obs-d4555102442f5955` (`sdd/cachicamas-chat-store-adapter/spec`).
- CH-07 design: Engram `obs-38d8b29d883ba8d8` (`sdd/cachicamas-chat-store-adapter/design`).
- CH-07 tasks: Engram `obs-d439027cce4bf5ac` (`sdd/cachicamas-chat-store-adapter/tasks`).