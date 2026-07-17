-- +goose Up
-- +goose StatementBegin
--
-- 2026-07-17-skills-foundational
-- Lifts ADR-SK-001..009 (design #1968 §3.3).
--
CREATE TABLE IF NOT EXISTS skill (
    id          BIGSERIAL    PRIMARY KEY,
    name        TEXT         NOT NULL
                 CHECK (name ~ '^[a-z0-9]+(-[a-z0-9]+)*$' AND length(name) BETWEEN 1 AND 64),
    description TEXT         NOT NULL CHECK (length(description) BETWEEN 1 AND 1024),
    body        TEXT         NOT NULL CHECK (length(body) BETWEEN 1 AND 524288),
    deleted_at  TIMESTAMPTZ  NULL,
    created_at  TIMESTAMPTZ  NOT NULL DEFAULT now(),
    updated_at  TIMESTAMPTZ  NOT NULL DEFAULT now()
);

ALTER TABLE skill OWNER TO queen;

CREATE TABLE IF NOT EXISTS skill_revision (
    id              BIGSERIAL    PRIMARY KEY,
    skill_id        BIGINT       NOT NULL REFERENCES skill(id) ON DELETE CASCADE,
    revision_number INT          NOT NULL CHECK (revision_number > 0),
    description     TEXT         NOT NULL CHECK (length(description) BETWEEN 1 AND 1024),
    body            TEXT         NOT NULL CHECK (length(body) BETWEEN 1 AND 524288),
    change_note     TEXT         NULL,
    created_at      TIMESTAMPTZ  NOT NULL DEFAULT now(),
    CONSTRAINT skill_revision_unique UNIQUE (skill_id, revision_number)
);

ALTER TABLE skill_revision OWNER TO queen;

COMMENT ON TABLE skill IS
    'Current definitive row of an Agent Skill (SKILL.md). Always reflects the latest version. Soft-delete via deleted_at reuses the name.';

COMMENT ON COLUMN skill.body IS
    'SKILL.md file content (YAML frontmatter + markdown body). Validated 1..524288 bytes by CHECK constraint. Frontmatter MUST contain name and description matching the row.';

COMMENT ON COLUMN skill.name IS
    'URL-friendly slug + agentskills.io name. Format: ^[a-z0-9]+(-[a-z0-9]+)*$, length 1..64. Reserved words (anthropic, claude) rejected at domain layer. Unique among active rows.';

COMMENT ON TABLE skill_revision IS
    'Append-only history of changes to `skill`. INSERT-only; never UPDATE, never DELETE except via CASCADE when the parent skill is hard-deleted.';

COMMENT ON COLUMN skill_revision.revision_number IS
    'Monotonic per skill_id. Strictly increasing positive integer. Assigned by the application under a FOR UPDATE row lock on the parent skill.';

-- Name uniqueness scoped to active rows (ADR-SK-003).
CREATE UNIQUE INDEX IF NOT EXISTS skill_slug_active_uidx
    ON skill(name) WHERE deleted_at IS NULL;

-- List ordering by recency (used by GET /skills).
CREATE INDEX IF NOT EXISTS skill_updated_at_idx
    ON skill(updated_at DESC);

-- Revision lookup supports: latest first, exact (skill_id, n).
CREATE INDEX IF NOT EXISTS skill_revision_skill_id_idx
    ON skill_revision(skill_id, revision_number DESC);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
-- Forward-only migration per openspec/AGENTS.md.
-- +goose StatementEnd
