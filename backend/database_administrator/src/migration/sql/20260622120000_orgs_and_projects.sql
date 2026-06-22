-- +goose Up
-- +goose StatementBegin
-- witsaba-core-tables migration 1 of 3: organization + project.
--
-- Locks spec O1-O4 (organization) and P1-P4 (project). The remaining
-- 6 tables land in files 20260622120001 and 20260622120002; see
-- openspec/changes/witsaba-core-tables/{proposal,spec,design}/ for
-- the full context.
--
-- Cardinality: one organization -> many projects (synthetic PK on
-- project, non-PK FK organization_id). `project.metadata` is the
-- ONLY place JSONB is allowed in the entire witsaba schema
-- (per locked decision).
--
-- Append-mostly convention: `organization.is_active` is the ONLY
-- column in the entire schema that allows in-place UPDATE; the
-- column comment below makes that exception explicit for code
-- review. DB-level enforcement is a follow-up change
-- (`witsaba-core-tables-append-only-enforcement`).
CREATE TABLE organization (
    id              BIGSERIAL    PRIMARY KEY,
    shortname       TEXT,
    full_name       TEXT         NOT NULL,
    identification  TEXT         NOT NULL UNIQUE,
    is_active       BOOLEAN      NOT NULL DEFAULT TRUE,
    email           TEXT,
    phone           TEXT,
    created_at      TIMESTAMPTZ  NOT NULL DEFAULT now(),
    updated_at      TIMESTAMPTZ  NOT NULL DEFAULT now()
);

COMMENT ON COLUMN organization.is_active IS
    'UPDATE-in-place: this column is the ONLY mutation allowed on organization rows.';

CREATE TABLE project (
    id              BIGSERIAL    PRIMARY KEY,
    organization_id BIGINT       NOT NULL REFERENCES organization(id),
    key             TEXT         NOT NULL UNIQUE,
    full_name       TEXT         NOT NULL,
    start_date      DATE,
    end_date        DATE,
    metadata        JSONB,
    created_at      TIMESTAMPTZ  NOT NULL DEFAULT now(),
    updated_at      TIMESTAMPTZ  NOT NULL DEFAULT now()
);

CREATE INDEX idx_project_organization_id ON project(organization_id);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE project;
DROP TABLE organization;
-- +goose StatementEnd
