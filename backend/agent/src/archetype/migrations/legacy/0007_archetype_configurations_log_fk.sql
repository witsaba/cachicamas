-- Migration 0007 — archetype_configurations_log gains archetype_slug FK
-- (cachicamas-archetype-system-foundation, PR-1, T-07)
--
-- Mirrors the PK reshape from 0006 on the audit log table. Every
-- prior row had archetype_kind='chat' (the only kind at the time);
-- backfill populates archetype_slug='assistant' for those rows. The
-- DEFAULT clause catches future inserts during the column-add phase
-- (Postgres ADD COLUMN NOT NULL DEFAULT is metadata-only in PG 11+).
--
-- Steps:
--   1. ADD COLUMN archetype_slug text NOT NULL DEFAULT 'assistant'
--   2. Defensive UPDATE — catches any NULL that slipped past DEFAULT.
--   3. ADD CONSTRAINT archetype_configurations_log_slug_fkey FOREIGN
--      KEY (archetype_slug) REFERENCES archetypes(slug) ON DELETE
--      RESTRICT.
--   4. CREATE INDEX idx_archetype_configurations_log_slug_org_created
--      ON archetype_configurations_log (archetype_slug, org_id, created_at DESC)
--      — replaces the original kind-prefixed index for query plans that
--      filter by slug. The old kind-index is left in place for backwards
--      compatibility with any historical query plan.
--
-- Forward-only additive (per the chat composition root's allowlist;
-- CREATE TABLE/INDEX/INSERT/ALTER ADD COLUMN/COMMENT ON are all
-- allowed).

-- +goose Up

ALTER TABLE archetype_configurations_log
    ADD COLUMN archetype_slug text NOT NULL DEFAULT 'assistant';

UPDATE archetype_configurations_log
    SET archetype_slug = 'assistant'
    WHERE archetype_slug IS NULL;

ALTER TABLE archetype_configurations_log
    ADD CONSTRAINT archetype_configurations_log_slug_fkey
        FOREIGN KEY (archetype_slug)
        REFERENCES archetypes(slug)
        ON DELETE RESTRICT;

CREATE INDEX idx_archetype_configurations_log_slug_org_created
    ON archetype_configurations_log (archetype_slug, org_id, created_at DESC);
