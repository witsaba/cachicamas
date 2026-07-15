# Explore — `2026-07-15-prompt-storage-table`

> **Change**: `2026-07-15-prompt-storage-table`
> **Project**: `cachicamas`
> **Artifact store**: `openspec` (Engram offline at session start; project reality is `openspec/changes/<name>/`)
> **Strict TDD**: ACTIVE
> **Phase**: explore (read-only; no proposal decisions yet)

---

## 0. User intent (verbatim)

> "I wanna add a new table to the main database in `@backend/database_administrator/`. I wanna add a table to handle the prompts. The table must to have a pk, a short description about the prompt, a slug unique key and a binary column to store the markdown prompt. review in the web and research about the standar industry about how is handled the prompts in a postgres database."

Hard requirements extracted:

| # | Requirement | Source |
| --- | --- | --- |
| R1 | New table in the **main** database (PostgreSQL 18, user `queen`) | User intent + project context |
| R2 | A primary key | User |
| R3 | A short description column | User |
| R4 | A **slug** with a **UNIQUE** constraint | User |
| R5 | A **binary** column to store the markdown prompt | User |
| R6 | Conform to existing project conventions (hexagonal Go layout, goose migrations, TDD) | `openspec/config.yaml` + `AGENTS.md` |
| R7 | Research industry standards first | User |

User ambiguity to surface in proposal:

- **R5 says "binary column".** "Binary" in PostgreSQL terminology specifically means `BYTEA`. Markdown, however, is UTF-8 text. Industry consensus (PostgreSQL wiki, LangChain, LlamaIndex, multiple production schemas) is that prompt bodies go in `TEXT` (or `JSONB` for structured prompts), not `BYTEA`. The proposal must ask the user to confirm which one they actually want and why.

---

## 1. Project recon

### 1.1 Stack (from `openspec/project.md`)

- Go 1.26.3 + Echo v5 (`github.com/labstack/echo/v5`)
- PostgreSQL 18 (`postgres:18-alpine3.24`)
- Hexagonal layout under `backend/database_administrator/src/{cmd,application,domain,interfaces,infrastructure,migration,otel}/`
- Migrations live in `src/migration/sql/` with `+goose Up` / `+goose Down` annotations
- DB user: `queen` (NOSUPERUSER, CREATEROLE, CREATEDB) — provisioned via `infra/postgres/init/01-init.sql`
- Module path: `github.com/cachicamas/backend/database_administrator`

### 1.2 Existing migration patterns (canonical reference)

Two recent migrations informed the existing convention; the new table MUST follow the same shape:

| Migration | Locks |
| --- | --- |
| `20260622120002_tasks_and_specs.sql` | `task`, `spec`, `spec_phase` — synthetic BIGSERIAL PK, FK, CHECK constraints, COMMENT ON TABLE/COLUMN, partial index for current-state lookup |
| `20260708120200_sync_job.sql` | `sync_job` — `CREATE TABLE IF NOT EXISTS`, partial UNIQUE index for single-flight invariant, `ALTER TABLE … OWNER TO queen`, `IF NOT EXISTS` on every object |

Canonical column-shape template (extracted from the two migrations above):

```sql
CREATE TABLE IF NOT EXISTS <name> (
    id          BIGSERIAL    PRIMARY KEY,
    -- domain columns here
    created_at  TIMESTAMPTZ  NOT NULL DEFAULT now(),
    updated_at  TIMESTAMPTZ  NOT NULL DEFAULT now()
);

ALTER TABLE <name> OWNER TO queen;

CREATE INDEX IF NOT EXISTS <name>_<col>_idx ON <name>(<col>);

-- +goose StatementEnd
-- +goose Down
-- +goose StatementBegin
-- Forward-only migration per openspec/AGENTS.md. Down block intentionally empty.
-- +goose StatementEnd
```

### 1.3 Domain/repo patterns (canonical reference)

`src/domain/workspace.go` and `src/infrastructure/postgres/workspace_repo.go` define the hexagonal contract:

- `domain.Workspace` mirrors the DDL column-for-column with `db` + `json` struct tags
- `domain.WorkspaceRepository` is the port the application layer depends on
- The pgx adapter translates `SQLSTATE 23505` → `*domain.ConflictError`, "no rows" → `*domain.NotFoundError`
- Typed errors implement a `Code()` method that the HTTP handler maps to a locked error envelope
- The pgx import surface is restricted to `infrastructure/postgres/` — `domain/` must NOT import `interfaces/` or `otel/`

### 1.4 Tests (canonical reference)

- `make test` → `go test -race -v ./...`
- Domain tests live next to the file (`workspace_test.go`, `organization_test.go`)
- Repo tests live next to the adapter (`workspace_repo_test.go`)
- Handler tests live in `interfaces/http/` next to the handler

### 1.5 Existing tables in scope (sanity check)

The schema currently holds (from migrations): `task`, `spec`, `spec_phase`, `workspace`, `workspace_repository` (dropped 2026-07-08), `sync_job`, `account_token`, `organization`, `project`, `milestone`, `requirement`. **None** of these is a prompt table. The new table is genuinely net-new.

### 1.6 Strict TDD

`openspec/config.yaml` enforces `tdd: true` + `test_command: "go test ./..."`. Every code task in the future `apply` phase starts with a failing test. AGENTS.md repeats this discipline for domain/application/handler/repo layers.

### 1.7 Session preflight (carried from injected block)

- Execution mode: `auto` (phases may run back-to-back)
- Artifact store: `engram` (non-authoritative carve-out active; in practice using `openspec/changes/<name>/` because Engram HTTP server at `127.0.0.1:7437` is unreachable)
- Chained PR strategy: `auto-forecast`
- Review budget: 450 changed lines
- Pause-before-apply gate still applies: if the forecast exceeds the budget or chained PR is recommended, the orchestrator must ask the user.

---

## 2. Web research — industry standards for prompt storage in PostgreSQL

Sources (full URLs in the search-content cache):

| # | Source | Relevant takeaway |
| --- | --- | --- |
| 1 | [PostgreSQL wiki: BinaryFilesInDB](https://wiki.postgresql.org/wiki/BinaryFilesInDB) | BYTEA is for true binary blobs (images, encrypted/compressed data). TEXT is preferred for any text-based content; both go through TOAST transparently when large. |
| 2 | [PostgreSQL docs 18: Binary Data Types](https://www.postgresql.org/docs/current/datatype-binary.html) | BYTEA stores raw bytes with no encoding; TEXT/VARCHAR are server-encoding aware (UTF-8 by default in this project). |
| 3 | [Evan Jones: Postgres large sub-string query performance](https://www.evanjones.ca/postgres-large-string-performance.html) | Substring/regex queries on large TEXT are ~6× slower than HSTORE/JSONB key lookups. BYTEA is faster for raw substring search but loses the ability to do text-aware operations (lowercase, LIKE, regex with locale). |
| 4 | [LangChain: `langchain-postgres` schema management](https://deepwiki.com/langchain-ai/langchain-postgres/5.2-schema-management) | LangChain uses TEXT/VARCHAR for prompt template content; vector tables add a `vector` column for embeddings. |
| 5 | [LlamaIndex: `PostgresChatStore`](https://developers.llamaindex.ai/python/framework-api-reference/storage/chat_store/postgres/) | Stores chat messages as TEXT or JSONB; uses `key`/`value` columns. |
| 6 | [Oracle Developers: Designing Multi-Tenant Agent Memory Schemas for SaaS (2026-07-09)](https://blogs.oracle.com/developers/from-prompt-to-persistence-part-1-designing-multi-tenant-agent-memory-schemas-for-saas) | Production multi-tenant prompt schema: `tenant_id`, `prompt_id`, `name` (unique within tenant), `template TEXT`, `version INT`, `is_active BOOL`, `created_at/updated_at`. |
| 7 | [jusdb.com: Database Schema Design for LLM Applications](https://www.jusdb.com/blog/database-schema-design-llm-applications) | Walk-through of every LLM-related table; prompt templates stored as TEXT with metadata columns. |
| 8 | [Markaicode: Compliance Archiving for LLM Applications](https://markaicode.com/compliance-llm-chat-log-archiving/) | Audit-log pattern uses UUID PK, tenant scoping, TIMESTAMPTZ, TEXT body, plus JSONB for request/response payloads. |
| 9 | [Soft-deleting Postgres rows without losing the URL slug (DEV)](https://dev.to/danielrusnok/soft-deleting-postgres-rows-without-losing-the-url-slug-351k) | Pattern: `slug TEXT NOT NULL` + partial UNIQUE index `WHERE deleted_at IS NULL` to free the slug on soft-delete without a name collision. |
| 10 | [Tim Santeford: Automating URL Slug Generation in PostgreSQL](https://www.timsanteford.com/posts/automating-url-slug-generation-in-postgresql-with-triggers-and-functions/) | `slugify()` plpgsql + `BEFORE INSERT/UPDATE` trigger; CHECK constraint enforces `[a-z0-9-]+`. |
| 11 | [StackOverflow: best way to store a unique URL slug](https://stackoverflow.com/questions/434376/what-is-the-best-way-to-store-a-unique-url-slug) | Community consensus: TEXT column + UNIQUE index + lowercase + hyphen normalization + CHECK or regex constraint. |

### 2.1 Synthesis — body column choice

| Option | Use case | Markdown fit |
| --- | --- | --- |
| `TEXT` (utf-8) | Markdown, HTML, any human-readable text. TOAST handles up to ~1 GB. Best for LIKE/regex/lowercase operations. | ✅ Industry standard. Recommended. |
| `JSONB` | Structured prompts with template variables (`{"system": "...", "user_template": "{{input}}"}`). Best for key/value lookups. | ✅ When the prompt has structure. |
| `BYTEA` | True binary (images, encrypted blobs, compressed payloads). No encoding awareness. Substring search is faster but text ops are gone. | ⚠️ Works (any bytes are valid) but discards the affordances Postgres gives you for utf-8 text. **Not recommended for markdown.** |
| `VARCHAR(n)` | Bounded text. Avoid for variable-length markdown; pick `TEXT` instead. | ❌ No advantage. |

### 2.2 Synthesis — slug

Pattern that survives soft-delete and tenant scoping:

```sql
slug TEXT NOT NULL CHECK (slug ~ '^[a-z0-9][a-z0-9-]{0,98}[a-z0-9]$'),
CREATE UNIQUE INDEX prompt_slug_active_uidx ON prompt(slug) WHERE deleted_at IS NULL;
```

If multi-tenant: include the tenant column in the partial index.

### 2.3 Synthesis — PK + audit

- `BIGSERIAL PRIMARY KEY` (matches the project's existing convention).
- `created_at TIMESTAMPTZ NOT NULL DEFAULT now()` and `updated_at TIMESTAMPTZ NOT NULL DEFAULT now()` (with trigger or app-layer discipline to bump `updated_at`).

---

## 3. Open questions to surface in proposal

These cannot be resolved by the orchestrator alone; they need explicit user input.

| # | Question | Why it matters | Default if unanswered |
| --- | --- | --- | --- |
| Q1 | **Body column type**: `TEXT` (recommended for markdown), `JSONB` (if prompts have structured variables), or `BYTEA` (the user said "binary" — confirm intent)? | Changes the schema, the domain model, and the repo surface. | `TEXT` (industry default, matches R5's actual intent if "binary" was used loosely) |
| Q2 | **Versioning**: include `version INT` + `is_active BOOL` in v1, or keep v1 single-version and defer? | Adds 2 columns and a partial unique index. Affects API shape. | Defer to v1.1 (single-version first; v1.1 adds history) |
| Q3 | **Soft-delete**: yes (with `deleted_at TIMESTAMPTZ NULL` + partial unique on slug) or no? | Project's `workspace` already soft-deletes (`deleted_at TIMESTAMPTZ NULL`). Consistent with the codebase. | Yes (matches existing pattern) |
| Q4 | **Ownership / tenancy**: global (no FK), `organization`-scoped (FK to `organization.id`), or `workspace`-scoped (FK to `workspace.id`)? | Determines FK chain and unique-scope. | Defer to v1.1 if not answerable now |
| Q5 | **Read/write API in v1**: just the table + a typed Go domain + a pgx repo + a minimal handler (`GET /prompts`, `GET /prompts/:slug`)? | Determines the size of the apply phase. | Yes — minimal CRUD (list + get-by-slug + create + update) |

---

## 4. Risks and mitigations

| Risk | Severity | Mitigation |
| --- | --- | --- |
| User wanted `BYTEA` but the orchestrator picked `TEXT` | Medium | Q1 in proposal forces a user answer before spec/design. |
| Table created without an owner → DB user `queen` cannot ALTER later | Low (proven pattern) | `ALTER TABLE … OWNER TO queen;` per existing migration template. |
| Slug collision on soft-delete | Low | Partial unique index `WHERE deleted_at IS NULL` (synthesis §2.2). |
| Forward-only migration breaks revert | Low (project norm) | Empty `Down` block, consistent with `sync_job.sql` and `drop_workspace_repository.sql`. |
| Review budget blown if apply adds handler + repo + domain + tests in one PR | Medium | `tasks.md` will forecast the line count and propose chained PRs (`auto-forecast`) if it exceeds 450 lines. |
| TDD skipped in `apply` | High (project rule) | Each task in `tasks.md` is preceded by a RED step; orchestrator pause gate before `apply`. |
| Engram offline breaks artifact persistence | Resolved | This change uses `openspec/changes/<name>/` (the project norm). Engram carve-out is non-authoritative per orchestrator workflow. |

---

## 5. Out of scope (explicit)

- Vector embeddings of prompts (pgvector) — defer to a separate change.
- Prompt templating / variable substitution engine — pure storage only.
- UI surface for editing prompts — defer to frontend change.
- Multi-tenant scoping — defer unless Q4 answers otherwise.
- Prompt history / audit log — defer to v1.1.
- Encryption-at-rest of body content — Postgres-level TDE, defer.

---

## 6. Next phase

`proposal.md` will:

1. Lock the table name (`prompt`).
2. Surface Q1–Q5 from §3 to the user with default recommendations.
3. Lock the canonical DDL shape (per §1.2) with Q1's answer.
4. List out-of-scope items (mirrors §5).
5. State the rollback plan (empty Down; `DROP TABLE` is manual per project norm).

Until Q1 is answered, `spec.md` and `design.md` will use a placeholder column type marked `<<body column — TBD by Q1>>` so the design is concrete enough to review but honest about the unresolved decision.
