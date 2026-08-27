-- Migration 0006 — archetype_configurations PK reshape
-- (cachicamas-archetype-system-foundation, PR-1, T-04)
--
-- The one non-additive migration in this change. Replaces the flat
-- archetype_configurations(archetype_kind, org_id) PK with the
-- polymorphic archetype_configurations(archetype_slug, org_id) shape
-- referenced to the new archetypes parent table.
--
-- Steps inside ONE transaction:
--   1. CREATE TABLE archetype_configurations__backup AS SELECT * —
--      pre-migration snapshot. The wrapper (if shipped at
--      migrations/0006_runner.go per design AD-6) leaves this in
--      place on rollback; operators DROP it manually after verifying
--      the new shape.
--   2. ADD COLUMN archetype_slug text — nullable initially so the
--      UPDATE can populate it without violating NOT NULL.
--   3. UPDATE … SET archetype_slug = 'assistant' WHERE archetype_kind = 'chat'
--      — re-key legacy rows. The chat kind was the unique kind today;
--      migration 0003's seed row carries org_id='__default__', which
--      is preserved verbatim through the reshape.
--   4. DROP CONSTRAINT archetype_configurations_pkey.
--   5. DROP COLUMN archetype_kind.
--   6. ALTER COLUMN archetype_slug SET NOT NULL.
--   7. ADD PRIMARY KEY (archetype_slug, org_id).
--   8. ADD CONSTRAINT archetype_configurations_slug_fkey FOREIGN KEY
--      (archetype_slug) REFERENCES archetypes(slug) ON DELETE RESTRICT.
--
-- Transaction semantics: any failure before COMMIT rolls back the
-- entire tx. The __backup table retains the pre-migration shape for
-- manual operator recovery.
--
-- Pre-PR-1 audit (`gentle-ai migrator validate --migration 0006`)
-- gates whether the runner's allowlist accepts this SQL. If refused,
-- migrations/0006_runner.go (T-05) wraps the SQL in a one-shot runner
-- that self-deletes from archetype_schema_migrations on success.

-- +goose Up

BEGIN;

CREATE TABLE archetype_configurations__backup AS
    SELECT * FROM archetype_configurations;

ALTER TABLE archetype_configurations
    ADD COLUMN archetype_slug text;

UPDATE archetype_configurations
    SET archetype_slug = 'assistant'
    WHERE archetype_kind = 'chat';

ALTER TABLE archetype_configurations
    DROP CONSTRAINT archetype_configurations_pkey;

ALTER TABLE archetype_configurations
    DROP COLUMN archetype_kind;

ALTER TABLE archetype_configurations
    ALTER COLUMN archetype_slug SET NOT NULL;

ALTER TABLE archetype_configurations
    ADD PRIMARY KEY (archetype_slug, org_id);

ALTER TABLE archetype_configurations
    ADD CONSTRAINT archetype_configurations_slug_fkey
        FOREIGN KEY (archetype_slug)
        REFERENCES archetypes(slug)
        ON DELETE RESTRICT;

COMMIT;
