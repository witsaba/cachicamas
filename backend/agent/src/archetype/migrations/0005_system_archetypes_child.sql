-- Migration 0005 — system_archetypes child table
--
-- Class Table Inheritance child for archetypes with type='system'. The
-- child table holds type-specific columns (bundle_version, is_critical)
-- that have no meaning for future general/owned kinds.
--
-- PK = FK: every system archetype row MUST have a corresponding
-- archetypes row (insert order enforced by application code). The FK
-- is ON DELETE RESTRICT — system catalogue rows are authoritative;
-- "uninstall system" is not a workflow that should silently delete the
-- catalogue.
--
-- Forward-only. Rollback branch: DROP TABLE system_archetypes; the
-- parent archetypes rows remain.

-- +goose Up

CREATE TABLE IF NOT EXISTS system_archetypes (
    slug           text    PRIMARY KEY
                       REFERENCES archetypes(slug) ON DELETE RESTRICT,
    bundle_version text    NOT NULL,
    is_critical    boolean NOT NULL
);
