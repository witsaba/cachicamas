-- Migration 0003 — seed chat archetype default row
--
-- The chat archetype is the unique archetype shipped today. Its
-- configuration MUST live in the database from boot, not in memory,
-- so a hot reload of the chat binary preserves operator tuning across
-- restarts and so the audit log captures every change from the very
-- first PUT (vs starting the log empty until the first user action).
--
-- Schema mechanics:
--   - org_id = '__default__' is a SENTINEL — the row is the per-archetype
--     system default, not a per-org row. The Loader recognises the
--     sentinel and rewrites the returned ArchetypeConfig.OrgID to the
--     caller's orgID while leaving the row itself untouched.
--   - ON CONFLICT DO NOTHING — running this migration twice is safe.
--     If an operator wants to refresh the default, they PUT to
--     (`chat`, `__default__`) directly; this migration is the seed,
--     not the source of truth.
--   - tool_allowlist and defer_tool_names are JSONB arrays; the
--     values mirror `archetype.DefaultDeferToolNames` and the chat
--     archetype's known tool names at this migration's authoring
--     time (`current_time`, `summarize_conversation`). When a new
--     tool is registered via `archetype.SetRegisteredToolNames`, the
--     safe-default factory in `archetype/config.go` is updated
--     simultaneously; an operator who wants the system default to
--     pick up the new tool must PUT to `__default__` (which the
--     fallback below the sentinel row does NOT auto-write).
--
-- Loader two-step lookup (in `archetype.PostgresLoader`):
--   1. SELECT ... WHERE archetype_kind = $1 AND org_id = $2 (caller's org)
--   2. If absent, SELECT ... WHERE archetype_kind = $1 AND org_id = '__default__'
--   3. If still absent, return in-memory `archetype.DefaultConfig(...)` —
--      this is the LAST-RESORT fallback for a DB outage (REQ-CACS-003
--      "if the database for any reason fails, MUST to use hardcoded
--      default values").
--
-- Per-org PUT writes `(archetype_kind, caller_org_id)` rows that SHADOW
-- this default — the org's tuning wins, the default stays untouched
-- for other orgs.
--
-- Forward-only. Rollback is `DELETE FROM archetype_configurations
-- WHERE archetype_kind = 'chat' AND org_id = '__default__'`.

INSERT INTO archetype_configurations
    (archetype_kind, org_id, system_prompt, tool_allowlist, defer_tool_names, model, version, updated_at, updated_by)
VALUES
    ('chat',
     '__default__',
     'You are the cachicamas chat assistant; answer the participant in plain, well-formatted text.',
     '["current_time", "summarize_conversation"]'::jsonb,
     '["summarize_conversation"]'::jsonb,
     NULL,
     1,
     now(),
     'seed')
ON CONFLICT (archetype_kind, org_id) DO NOTHING;
