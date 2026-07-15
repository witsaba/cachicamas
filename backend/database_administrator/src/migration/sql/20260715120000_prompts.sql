-- +goose Up
-- +goose StatementBegin
--
-- 2026-07-15-prompt-storage-table
-- Lifts proposal D1, D2, D3, D4, D5 and spec INV-1..4.
--
-- Two tables: `prompt` (current definitive row) + `prompt_revision`
-- (append-only history). Soft-delete on `prompt.deleted_at`. Slug uniqueness
-- is partial over active rows so the slug can be reused after delete.
--
CREATE TABLE IF NOT EXISTS prompt (
    id           BIGSERIAL    PRIMARY KEY,
    description  TEXT         NOT NULL CHECK (length(description) BETWEEN 1 AND 280),
    slug         TEXT         NOT NULL CHECK (slug ~ '^[a-z0-9][a-z0-9-]{0,98}[a-z0-9]$'),
    body         TEXT         NOT NULL CHECK (length(body) BETWEEN 1 AND 524288),
    deleted_at   TIMESTAMPTZ  NULL,
    created_at   TIMESTAMPTZ  NOT NULL DEFAULT now(),
    updated_at   TIMESTAMPTZ  NOT NULL DEFAULT now()
);

ALTER TABLE prompt OWNER TO queen;

CREATE TABLE IF NOT EXISTS prompt_revision (
    id              BIGSERIAL    PRIMARY KEY,
    prompt_id       BIGINT       NOT NULL REFERENCES prompt(id) ON DELETE CASCADE,
    revision_number INT          NOT NULL CHECK (revision_number > 0),
    description     TEXT         NOT NULL CHECK (length(description) BETWEEN 1 AND 280),
    body            TEXT         NOT NULL CHECK (length(body) BETWEEN 1 AND 524288),
    change_note     TEXT         NULL,
    created_by      TEXT         NULL,
    created_at      TIMESTAMPTZ  NOT NULL DEFAULT now(),
    CONSTRAINT prompt_revision_unique UNIQUE (prompt_id, revision_number)
);

ALTER TABLE prompt_revision OWNER TO queen;

COMMENT ON TABLE prompt IS
    'Current definitive row of an LLM prompt. Always reflects the latest version. Soft-delete via deleted_at reuses the slug.';

COMMENT ON COLUMN prompt.body IS
    'Markdown prompt body (utf-8 TEXT). Validated 1..524288 bytes by CHECK constraint.';

COMMENT ON COLUMN prompt.slug IS
    'URL-friendly slug. Format: ^[a-z0-9][a-z0-9-]{0,98}[a-z0-9]$. Unique among active (non-deleted) rows.';

COMMENT ON TABLE prompt_revision IS
    'Append-only history of changes to `prompt`. INSERT-only; never UPDATE, never DELETE except via CASCADE when the parent prompt is hard-deleted.';

COMMENT ON COLUMN prompt_revision.revision_number IS
    'Monotonic per prompt_id. Strictly increasing positive integer. Assigned by the application under a FOR UPDATE row lock on the parent prompt.';

-- Slug uniqueness is scoped to active rows.
CREATE UNIQUE INDEX IF NOT EXISTS prompt_slug_active_uidx
    ON prompt(slug) WHERE deleted_at IS NULL;

-- List ordering by recency (used by GET /prompts).
CREATE INDEX IF NOT EXISTS prompt_updated_at_idx
    ON prompt(updated_at DESC);

-- Revision lookup supports: latest first, exact (prompt_id, n).
CREATE INDEX IF NOT EXISTS prompt_revision_prompt_id_idx
    ON prompt_revision(prompt_id, revision_number DESC);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
-- Forward-only migration per openspec/AGENTS.md.
-- +goose StatementEnd