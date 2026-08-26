-- Migration 0005 — chat_assistant_config
--
-- One configuration row per org. Stores the chat archetype's
-- user-editable knobs: system prompt, tool allowlist (names only;
-- implementations stay in code per ADR 0006), defer set (subset of
-- allowlist), optional informational model field, and a monotonic
-- version int that the runtime reads on every Send to decide whether
-- to rebuild the prompt (REQ-CCVP-001/002).
--
-- Persistence is the API layer's responsibility only. The Loader does
-- NOT auto-write on absent row (REQ-CACS-003): an absent read returns
-- safe defaults in-memory; the first PUT is what creates the row.
--
-- The audit log of writes lives in 0006_assistant_config_log.sql
-- (separate table; added in PR-3 of cachicamas-assistant-configuration-ui).
--
-- Forward-only. Rollback is `DROP TABLE chat_assistant_config`.

CREATE TABLE IF NOT EXISTS chat_assistant_config (
    org_id           text        PRIMARY KEY,
    system_prompt    text        NOT NULL,
    tool_allowlist   jsonb       NOT NULL,
    defer_tool_names jsonb       NOT NULL,
    model            text        NULL,
    version          int         NOT NULL DEFAULT 1,
    updated_at       timestamptz NOT NULL DEFAULT now(),
    updated_by       text        NOT NULL DEFAULT ''
);
