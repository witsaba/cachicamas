-- Migration 0001 — archetype_configurations
--
-- One configuration row per (archetype_kind, org_id) pair. Stores the
-- runtime-editable knobs every Layer 3 archetype exposes:
--
--   - system_prompt:    the per-archetype system prompt
--   - tool_allowlist:   jsonb array of tool names (implementations stay
--                       in code per ADR 0006; only names toggle)
--   - defer_tool_names: jsonb subset of tool_allowlist requiring
--                       permission approval before each invocation
--   - model:            optional informational field; the actual
--                       model selection remains env-driven (process-wide)
--                       per the locked decision
--   - version:          monotonic int bumped on every successful PUT;
--                       consulted by the per-archetype version-aware
--                       rebuild contract
--   - updated_at/_by:   server-set on every successful PUT
--
-- Why this is named archetype_configurations (not chat_assistant_config):
-- the storage contract is a Layer 3 concern shared by every archetype.
-- Putting "chat" in the table name would couple future archetypes
-- (coding, support, ...) to whichever archetype ships first. The
-- `archetype_kind` column is the discriminator; today's only value is
-- 'chat' (the chat archetype), but the schema supports future kinds
-- additively without migration churn.
--
-- Persistence is the write API layer's responsibility only. The Loader
-- does NOT auto-write on absent row (REQ-CACS-003): an absent read
-- returns safe defaults in memory; the first PUT is what creates the
-- row.
--
-- The audit log of writes ships in 0002_archetype_configurations_log.sql
-- (separate table; added in PR-3 of
-- cachicamas-assistant-configuration-ui).
--
-- Forward-only. Rollback is `DROP TABLE archetype_configurations`.

-- +goose Up

CREATE TABLE IF NOT EXISTS archetype_configurations (
    archetype_kind   text        NOT NULL DEFAULT 'chat',
    org_id           text        NOT NULL,
    system_prompt    text        NOT NULL,
    tool_allowlist   jsonb       NOT NULL,
    defer_tool_names jsonb       NOT NULL,
    model            text        NULL,
    version          int         NOT NULL DEFAULT 1,
    updated_at       timestamptz NOT NULL DEFAULT now(),
    updated_by       text        NOT NULL DEFAULT '',
    PRIMARY KEY (archetype_kind, org_id)
);
