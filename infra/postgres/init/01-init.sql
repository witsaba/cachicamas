-- ============================================================================
-- Cachicamas — PostgreSQL initialization
--
-- Runs automatically the FIRST TIME the postgres container starts, against the
-- database named by POSTGRES_DB in the environment. Files are executed in
-- alphabetical order, so prefix with NN- to control order.
--
-- Roles provisioned here:
--
--   postgres  — superuser. Created automatically by the official image from
--               POSTGRES_USER env. Never used by the application.
--
--   queen     — DBA role for the platform. NOT a superuser. Has CREATEROLE,
--               CREATEDB and REPLICATION so it can:
--                 • create / drop other roles (e.g. `cachicamas_app`, `wiki`,
--                   or any future per-context user)
--                 • create new databases / schemas for new bounded contexts
--                 • GRANT / REVOKE on any object it owns
--                 • run logical backups / read replicas
--               Today the microservices also connect as queen because no
--               other role exists yet. Tomorrow: run a migration as queen
--               that creates a least-privilege role (e.g. cachicamas_app),
--               then switch the service's connection string.
--
-- Why "queen"? cachicamas = a Polybia wasp species native to the Llanos
-- Orientales; the queen leads the colony. This role leads the database.
--
-- Reference: https://github.com/docker-library/postgres
-- ============================================================================

-- ---------------------------------------------------------------------------
-- Extensions
-- ---------------------------------------------------------------------------
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";
CREATE EXTENSION IF NOT EXISTS "pgcrypto";
CREATE EXTENSION IF NOT EXISTS "citext";

-- ---------------------------------------------------------------------------
-- Read the queen password from the container environment.
--
-- The official postgres entrypoint runs /docker-entrypoint-initdb.d/*.sql
-- via `psql -f`, so the psql `\getenv` command (available since psql 15)
-- is available here. Anything the docker-compose env: block defines is
-- reachable by name. Falls back to an obvious dev default if the var is
-- missing (change in any real env).
-- ---------------------------------------------------------------------------
\getenv queen_password QUEEN_PASSWORD
SELECT set_config('app.queen_password', coalesce(:'queen_password', 'changeme-queen'), false);

-- ---------------------------------------------------------------------------
-- DBA role — `queen`
--
-- CREATEROLE   — create / alter / drop other roles.
-- CREATEDB     — create new databases for future bounded contexts.
-- REPLICATION  — required for logical backups / read replicas.
-- NOSUPERUSER  — explicit. Cannot read other users' data without GRANTs,
--                cannot tamper with the catalog bypassing ACLs.
-- NOINHERIT    — explicit. Future per-context roles granted to queen do NOT
--                leak to anyone inheriting queen's privileges.
--
-- Idempotent: re-running this file does NOT clobber an existing queen.
-- ---------------------------------------------------------------------------
DO $$
BEGIN
    IF NOT EXISTS (SELECT 1 FROM pg_roles WHERE rolname = 'queen') THEN
        EXECUTE format(
            'CREATE ROLE queen WITH LOGIN NOSUPERUSER NOCREATEDB NOCREATEROLE NOINHERIT PASSWORD %L',
            current_setting('app.queen_password')
        );
    END IF;
END
$$;

-- Apply the DBA attributes after creation (cleaner than mixing WITH clauses).
ALTER ROLE queen WITH CREATEROLE CREATEDB REPLICATION;

-- ---------------------------------------------------------------------------
-- Database-level settings — must run AFTER queen exists.
--
-- Ordering rationale:
--   1. The queen role must exist before we can transfer ownership of
--      `cachicamas_pg` to her (else the ALTER DATABASE fails with
--      "role \"queen\" does not exist").
--   2. The timezone pin to 'UTC' must be in place BEFORE the migration
--      runner connects as queen for the first time, so the first
--      tstamp written to public.schema_migrations is already in UTC
--      (no follow-up ALTER DATABASE will retroactively re-stamp rows).
--
-- These statements live in init.sql (not in a goose migration) on
-- purpose — see spec R-DBMIG-011. Running them as the cluster
-- superuser during initdb is the only context where both succeed.
--
-- The DO $$ block is required because Postgres' `ALTER DATABASE name`
-- syntax does NOT accept `current_database()` as the target — only a
-- literal identifier. Wrapping in DO + EXECUTE lets us resolve the
-- name at runtime while still running as superuser.
-- ---------------------------------------------------------------------------
DO $$
BEGIN
    EXECUTE format('ALTER DATABASE %I OWNER TO queen', current_database());
    EXECUTE format('ALTER DATABASE %I SET timezone = %L', current_database(), 'UTC');
END
$$;

-- ---------------------------------------------------------------------------
-- Schemas (logical separation by bounded context)
--
-- Owned by queen so future per-context roles can be granted narrowly:
--   GRANT USAGE ON SCHEMA <name> TO <role>;
--   GRANT ALL  ON SCHEMA <name> TO <role>;
-- ---------------------------------------------------------------------------
CREATE SCHEMA IF NOT EXISTS catalog;
CREATE SCHEMA IF NOT EXISTS identity;
CREATE SCHEMA IF NOT EXISTS observability;

ALTER SCHEMA catalog       OWNER TO queen;
ALTER SCHEMA identity      OWNER TO queen;
ALTER SCHEMA observability OWNER TO queen;

COMMENT ON SCHEMA catalog       IS 'Product catalog, categories, brands, SKUs';
COMMENT ON SCHEMA identity      IS 'Users, roles, sessions, audit log';
COMMENT ON SCHEMA observability IS 'Operational state (health, migrations log)';

-- ---------------------------------------------------------------------------
-- Migrations bookkeeping (used by pressly/goose v3.27.1 as queen)
--
-- Goose v3 writes rows with column names `id`, `version_id`, `is_applied`,
-- and `tstamp`; the legacy v2 shape (`version TEXT PK`, `applied_at`,
-- `description`) is incompatible with v3 and would crash the first boot
-- with `column "version_id" of relation "schema_migrations" does not
-- exist`. We provision the v3 shape directly here so the table is
-- usable on the very first container start.
--
-- See design §9 and spec R-DBMIG-070. Index on version_id keeps the
-- `SELECT MAX(version_id)` lookup fast as the history grows.
-- ---------------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS public.schema_migrations (
    id         BIGSERIAL    PRIMARY KEY,
    version_id BIGINT       NOT NULL UNIQUE,
    is_applied BOOLEAN      NOT NULL,
    tstamp     TIMESTAMPTZ  NOT NULL DEFAULT now()
);

-- Transfer ownership of the bookkeeping table to queen. Without this,
-- the table owner is the cluster superuser (the POSTGRES_USER from
-- docker-compose, `cachicamas`), and Postgres 15+ does NOT auto-grant
-- DML to the database owner on objects the superuser created. The
-- runner connects as queen, so it must own (or at least have explicit
-- grants on) this table — otherwise the first boot fails with
-- `ERROR: permission denied for table schema_migrations (SQLSTATE 42501)`.
--
-- We use ALTER TABLE ... OWNER TO (rather than GRANT ALL) so queen is
-- the future grantor: when a least-privilege role (e.g. cachicamas_app)
-- is provisioned later, queen can grant narrow DML on this table
-- without needing superuser. See spec R-DBMIG-070.
ALTER TABLE public.schema_migrations OWNER TO queen;

-- Seed the bookkeeping table with a zero-version row.
--
-- Goose v3.27.1's Up() path checks `ListMigrations` and refuses to
-- start if the table is empty (error: "missing zero version migration").
-- Normally goose inserts this row itself when it CREATEs the table on
-- first boot. We pre-create the table here so the runner doesn't need
-- CREATE permission on the public schema at runtime — but that means
-- we must also pre-seed the zero row, otherwise the runner crashes.
--
-- The row is marked `is_applied = false` per goose's convention: the
-- zero row is a marker, not a real applied migration. Goose's
-- UpVersions() will skip past it and apply every real migration whose
-- version_id > 0.
--
-- ON CONFLICT (version_id) DO NOTHING makes the script idempotent:
-- re-running init.sql (e.g. after a partial failure) does not duplicate
-- the row. The version_id column has a unique index (created above)
-- which provides the conflict target.
INSERT INTO public.schema_migrations (version_id, is_applied, tstamp)
    VALUES (0, false, now())
    ON CONFLICT (version_id) DO NOTHING;

COMMENT ON TABLE public.schema_migrations IS
    'Append-only log of applied migrations (goose v3 schema: id, version_id, is_applied, tstamp)';

-- ---------------------------------------------------------------------------
-- sync_job table CRUD grant (2026-07-08-workspace-sync-clone PR-1)
--
-- The `sync_job` table is created by the goose migration
-- `20260708120200_sync_job.sql` (owned by queen). The database_administrator
-- service connects as queen and needs CRUD on this table to read the
-- latest sync_job for the card and to insert / update jobs as the
-- user creates a workspace or clicks Sync. The GRANT is conditional on
-- the table existing (the migration may not have run yet on a fresh
-- cluster) — DO $$ so the init script remains idempotent.
-- ---------------------------------------------------------------------------
DO $$
BEGIN
    IF EXISTS (
        SELECT 1 FROM information_schema.tables
         WHERE table_schema = 'public' AND table_name = 'sync_job'
    ) THEN
        EXECUTE 'GRANT SELECT, INSERT, UPDATE, DELETE ON public.sync_job TO queen';
    END IF;
END
$$;

-- ---------------------------------------------------------------------------
-- Recipe for creating a future application role (e.g. cachicamas_app).
-- Run this as queen once you want least-privilege runtime users:
--
--   CREATE ROLE cachicamas_app WITH LOGIN NOSUPERUSER NOCREATEDB
--                                NOCREATEROLE NOINHERIT
--                                PASSWORD '...';
--   GRANT CONNECT ON DATABASE cachicamas TO cachicamas_app;
--   GRANT USAGE   ON SCHEMA public, catalog, identity, observability
--                   TO cachicamas_app;
--   GRANT CREATE  ON SCHEMA public     TO cachicamas_app;
--   ALTER DEFAULT PRIVILEGES FOR ROLE queen IN SCHEMA public
--       GRANT SELECT, INSERT, UPDATE, DELETE ON TABLES    TO cachicamas_app;
--   ALTER DEFAULT PRIVILEGES FOR ROLE queen IN SCHEMA public
--       GRANT USAGE, SELECT            ON SEQUENCES TO cachicamas_app;
-- ---------------------------------------------------------------------------