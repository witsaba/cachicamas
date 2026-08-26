-- 0002_tool_records.sql — CH-09 sibling tables for tool-call records
-- (R-CTS-006, R-CCS-015, NFR-CCS-006 forward-only, NFR-CCS-008).
--
-- CH-09 widens Exchange with ToolCalls []ToolCallRecord and
-- ToolResults []ToolResultRecord. Rather than ALTER chat_exchanges
-- (which would violate NFR-CCS-006 forward-only), the new fields
-- live in two sibling tables keyed by (chat_exchanges.participant_id,
-- chat_exchanges.position) — the same composite key the existing
-- exchanges table uses:
--
--   - chat_tool_calls: one row per tool call the model emitted
--     during the exchange; carries wire call id, tool name, and
--     arguments bytes (as text). Position is 0-indexed within the
--     exchange so the load order is deterministic.
--
--   - chat_tool_results: one row per tool call's outcome; carries
--     outcome enum ("success" | "result_failure" |
--     "execution_failure"), content text, and a typed failure
--     category string for execution_failure outcomes (no provider
--     text — R-CCP-008 / D6 mirror). Same 0-indexed position.
--
-- Forward-only (NFR-CCS-006): no DROP, no ALTER, no TRUNCATE, no
-- destructive ops. The runner refuses any migration whose every
-- line does not match the CREATE TABLE / CREATE INDEX / INSERT
-- allowlist. This migration passes the check trivially.
--
-- A future "MCP source" column can land here via
-- `ALTER TABLE chat_tool_calls ADD COLUMN source TEXT NOT NULL
-- DEFAULT 'builtin'` — a NEW column on a NEW table, satisfies
-- NFR-CCS-006.

-- +goose Up

CREATE TABLE chat_tool_calls (
    participant_id text NOT NULL,
    exchange_position integer NOT NULL,
    position integer NOT NULL,
    wire_call_id text NOT NULL,
    tool text NOT NULL,
    arguments text NOT NULL DEFAULT '{}',
    PRIMARY KEY (participant_id, exchange_position, position),
    FOREIGN KEY (participant_id, exchange_position)
        REFERENCES chat_exchanges (participant_id, position) ON DELETE CASCADE
);
CREATE INDEX chat_tool_calls_lookup_idx
    ON chat_tool_calls (participant_id, exchange_position);

CREATE TABLE chat_tool_results (
    participant_id text NOT NULL,
    exchange_position integer NOT NULL,
    position integer NOT NULL,
    wire_call_id text NOT NULL,
    tool text NOT NULL,
    outcome text NOT NULL CHECK (outcome IN ('success','result_failure','execution_failure')),
    content text NOT NULL DEFAULT '',
    failure_category text NOT NULL DEFAULT '',
    PRIMARY KEY (participant_id, exchange_position, position),
    FOREIGN KEY (participant_id, exchange_position)
        REFERENCES chat_exchanges (participant_id, position) ON DELETE CASCADE
);
CREATE INDEX chat_tool_results_lookup_idx
    ON chat_tool_results (participant_id, exchange_position);