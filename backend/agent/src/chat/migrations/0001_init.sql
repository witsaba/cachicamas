-- 0001_init.sql — CH-07's chat archetype migration (NFR-CCS-005, NFR-CCS-006).
--
-- Two tables only. The chat archetype owns these and ONLY these
-- (ADR 0009 § D6: each business system owns its own tables; no
-- archetype writes to another system's schema). Every CREATE TABLE /
-- CREATE INDEX in this migration carries the `chat_` prefix.
--
-- Forward-only (NFR-CCS-006): no DROP, no ALTER, no TRUNCATE, no
-- destructive ops. The runner refuses any migration whose every
-- line does not match the CREATE TABLE / CREATE INDEX /
-- CREATE SEQUENCE / COMMENT / INSERT allowlist. This migration
-- passes the check trivially.
--
-- Recorded at:
--   docs/adr/0010-add-pgx-and-goose-to-backend-agent.md  (ADR for the deps)
--   openspec/specs/chat-conversation-store/spec.md        (NFR-CCS-005/006, R-CCS-011)
--   docs/architecture/milestones/0005-...md              (CH-07 charter)

CREATE TABLE chat_conversations (
    participant_id text PRIMARY KEY,
    created_at     timestamptz NOT NULL DEFAULT now(),
    updated_at     timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE chat_exchanges (
    participant_id    text        NOT NULL REFERENCES chat_conversations(participant_id) ON DELETE CASCADE,
    position          integer     NOT NULL,
    prompt_text       text        NOT NULL,
    assistant_text    text        NOT NULL,
    partial           boolean     NOT NULL,
    terminal_kind     text        NOT NULL CHECK (terminal_kind IN ('completed','cancelled','failed')),
    failure_category  text        NOT NULL DEFAULT '',
    finish_reason     text        NOT NULL DEFAULT '',
    message_ids       jsonb       NOT NULL DEFAULT '[]'::jsonb,
    recorded_at       timestamptz NOT NULL DEFAULT now(),
    PRIMARY KEY (participant_id, position)
);

CREATE INDEX chat_exchanges_participant_recorded_at_idx
    ON chat_exchanges (participant_id, recorded_at DESC);