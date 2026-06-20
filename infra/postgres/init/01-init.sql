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
-- Migrations bookkeeping (used by a future migration runner as queen)
-- ---------------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS public.schema_migrations (
    version     TEXT        PRIMARY KEY,
    applied_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    description TEXT
);

COMMENT ON TABLE public.schema_migrations IS 'Append-only log of applied migrations';

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