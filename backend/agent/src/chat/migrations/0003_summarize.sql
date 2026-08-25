-- 0003_summarize.sql — CH-10.3 4th port method's mutation target
-- (R-CCS-017, R-CPM-006, NFR-CPM-005).
--
-- CH-10 widens Exchange with a summary column on chat_conversations
-- (the target of SummarizeConversationTool's UpdateSummary call).
-- The migration is forward-only: ADD COLUMN nullable is the
-- documented affordance per AGENTS.md "Substrate preservation in
-- backend/agent" paragraph on CH-07's pgx/v5 admission precedent
-- (NFR-CCS-006 / NFR-CPM-005).
--
-- Forward-only (NFR-CPM-005): no DROP, no ALTER of pre-existing
-- columns. The runner refuses any migration whose every line does
-- not match the ALTER TABLE / CREATE TABLE / CREATE INDEX / INSERT
-- allowlist. This migration passes the check trivially.
--
-- A future widening (e.g. "summary length cap", "summary
-- generation timestamp") can land via a follow-on
-- ALTER TABLE chat_conversations ADD COLUMN ... — a NEW column
-- on the EXISTING table satisfies NFR-CCS-006.

-- +goose Up

ALTER TABLE chat_conversations ADD COLUMN summary TEXT;