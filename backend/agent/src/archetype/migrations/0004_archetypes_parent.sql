-- Migration 0004 — archetypes parent table (Class Table Inheritance root)
--
-- Establishes the polymorphic identity table for the multi-archetype
-- catalogue. The new shape replaces the flat
-- archetype_configurations(archetype_kind, org_id) design with a parent
-- archetypes(slug, type, ...) table whose rows are referenced by the
-- per-org override table (archetype_configurations reshape in 0006)
-- and the type-specific child tables (system_archetypes in 0005,
-- future general_archetypes / owned_archetypes).
--
-- Schema:
--   slug         text PRIMARY KEY — the polymorphic identifier; the
--                 frontend AGENTS[0].slug is "assistant" today, mirrored
--                 here so the wire and the DB speak the same identifier.
--   type         text NOT NULL CHECK (type IN ('system','general','owned'))
--                 discriminator for child-table routing; locked at
--                 proposal OQ-3.
--   display_name text NOT NULL
--   tagline      text NOT NULL
--   status       text NOT NULL CHECK (status IN ('active','draft','archived'))
--                 — Loader treats (status='archived' OR archived_at IS NOT NULL)
--                 as the terminal predicate (SD-CASF-012).
--   archived_at  timestamptz NULL — soft-delete signal; separate from
--                 status so reactivation (UPDATE … SET status='active',
--                 archived_at=NULL) preserves history.
--   created_at   timestamptz NOT NULL DEFAULT now()
--   created_by   text NOT NULL
--
-- Forward-only. Rollback branch: DROP TABLE archetypes; on a database
-- with existing archetype_configurations rows, manual operator review
-- is required because the FK target referenced by 0006 is removed.

-- +goose Up

CREATE TABLE IF NOT EXISTS archetypes (
    slug         text        PRIMARY KEY,
    type         text        NOT NULL CHECK (type IN ('system','general','owned')),
    display_name text        NOT NULL,
    tagline      text        NOT NULL,
    status       text        NOT NULL CHECK (status IN ('active','draft','archived')),
    archived_at  timestamptz NULL,
    created_at   timestamptz NOT NULL DEFAULT now(),
    created_by   text        NOT NULL
);
