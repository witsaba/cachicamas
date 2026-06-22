-- +goose Up
-- +goose StatementBegin
-- witsaba-core-tables migration 2 of 3: requirement + requirement_spike + milestone.
--
-- Locks spec R1-R6 (requirement + 256 KiB cap + FKs + nullables),
-- S1-S3 (requirement_spike + multi-spike cardinality), M1-M5
-- (milestone strict inheritance + 1:1 cardinality + date CHECK).
--
-- Cardinality map (per locked decisions A1, A3):
--   - requirement -- 1:N --> requirement_spike  (synthetic PK + non-PK FK)
--   - requirement -- 1:1 --> milestone          (PK = FK = requirement_id)
--
-- The 256 KiB PRD cap is enforced via the `requirement_content_size_cap`
-- CHECK constraint AND documented as a column comment on
-- `requirement.content`. The escape hatch for a future higher cap is
-- documented in design.md §6.1: DROP CONSTRAINT then re-ADD with the
-- new limit.
CREATE TABLE requirement (
    id                     BIGSERIAL    PRIMARY KEY,
    project_id             BIGINT       NOT NULL REFERENCES project(id),
    filename               TEXT         NOT NULL,
    content                TEXT         NOT NULL,
    git_repository_url     TEXT,
    analysis_result        TEXT,
    is_technically_viable  BOOLEAN,
    created_at             TIMESTAMPTZ  NOT NULL DEFAULT now(),
    updated_at             TIMESTAMPTZ  NOT NULL DEFAULT now(),
    CONSTRAINT requirement_content_size_cap CHECK (octet_length(content) <= 262144)
);

COMMENT ON COLUMN requirement.content IS
    'Max PRD size: 256 KiB (262144 bytes), enforced via CHECK constraint.';

CREATE INDEX idx_requirement_project_id ON requirement(project_id);

CREATE TABLE requirement_spike (
    id              BIGSERIAL    PRIMARY KEY,
    requirement_id  BIGINT       NOT NULL REFERENCES requirement(id),
    started_at      DATE,
    ended_at        DATE,
    outcome         TEXT,
    findings        TEXT,
    created_at      TIMESTAMPTZ  NOT NULL DEFAULT now(),
    updated_at      TIMESTAMPTZ  NOT NULL DEFAULT now(),
    CONSTRAINT requirement_spike_dates_valid
        CHECK (ended_at IS NULL OR started_at IS NULL OR ended_at >= started_at)
);

CREATE INDEX idx_requirement_spike_requirement_id ON requirement_spike(requirement_id);

CREATE TABLE milestone (
    requirement_id  BIGINT       PRIMARY KEY REFERENCES requirement(id),
    title           TEXT         NOT NULL,
    description     TEXT,
    start_date      DATE,
    end_date        DATE,
    created_at      TIMESTAMPTZ  NOT NULL DEFAULT now(),
    updated_at      TIMESTAMPTZ  NOT NULL DEFAULT now(),
    CONSTRAINT milestone_dates_valid
        CHECK (end_date IS NULL OR start_date IS NULL OR end_date >= start_date)
);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE milestone;
DROP TABLE requirement_spike;
DROP TABLE requirement;
-- +goose StatementEnd
