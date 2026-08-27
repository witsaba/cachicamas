-- Migration 0002 — archetype_configurations_log
--
-- Append-only audit of every successful PUT against
-- archetype_configurations. The Writer (archetype.PostgresWriter)
-- inserts one row per accepted WriteConfig call inside the same
-- transaction as the UPSERT — REQ-CACL-002: every successful PUT
-- appends exactly one row; failed or rejected writes MUST NOT append.
--
-- Schema:
--   id           bigserial PK
--   archetype_kind text NOT NULL — discriminator (matches the row's kind)
--   org_id        text NOT NULL — matches the row's org
--   actor         text NOT NULL — the session identity that wrote the row
--   before        jsonb NULL — the prior state (NULL for first write)
--   after         jsonb NOT NULL — the new state, full ArchetypeConfig shape
--   created_at    timestamptz NOT NULL DEFAULT now()
--
-- Why nullable `before`: a first-time PUT (the row didn't exist) has
-- nothing to compare against; the absence is itself meaningful signal
-- ("this org's assistant was just configured for the first time").
--
-- Index `(archetype_kind, org_id, created_at DESC)` is the primary
-- query path: "show me the configuration history for org X". Other
-- shapes (audit by actor, audit by time window) hit this index as a
-- prefix and a parallel scan is acceptable for the expected audit
-- volume.
--
-- Forward-only. Rollback is `DROP TABLE archetype_configurations_log`.
-- (The audit log is itself the durability surface for compliance —
-- rolling back THIS migration is allowed only as part of a wider
-- operator-driven data-retention cycle, never as part of a code
-- rollback.)

-- +goose Up

CREATE TABLE IF NOT EXISTS archetype_configurations_log (
    id             bigserial    PRIMARY KEY,
    archetype_kind text         NOT NULL,
    org_id         text         NOT NULL,
    actor          text         NOT NULL DEFAULT '',
    before         jsonb        NULL,
    after          jsonb        NOT NULL,
    created_at     timestamptz  NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_archetype_configurations_log_kind_org_created
    ON archetype_configurations_log (archetype_kind, org_id, created_at DESC);
