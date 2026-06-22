-- +goose Up
-- +goose StatementBegin
-- witsaba-core-tables migration 3 of 3: task + spec + spec_phase.
--
-- Locks spec T1-T3 (task + FK), S4-S7 (spec + FK + date CHECK),
-- SP1-SP7 (spec_phase + 8 phase values + natural-key UNIQUE +
-- agent-first re-entry support + partial current-state index).
--
-- Cardinality map (per locked decisions A2, A4):
--   - milestone -- 1:N --> task   (synthetic PK + non-PK FK)
--   - task      -- 1:N --> spec   (synthetic PK + non-PK FK)
--   - spec      -- 1:N --> spec_phase  (append-only history child;
--                                       synthetic PK + UNIQUE natural
--                                       key on (spec_id, phase, started_at))
--
-- The 8 phase values locked in proposal Q1 are enforced by the
-- `spec_phase_phase_check` CHECK constraint. The agent-first
-- re-entry pattern (close current, open new) is supported by:
--   - the natural-key UNIQUE `spec_phase_natural_key`
--     (allows re-entering the same phase at a different started_at)
--   - the partial index `idx_spec_phase_current_state`
--     (fast lookup of the currently-open phase per spec)
--   - the nullable `ended_at` column (set on close)
--   - the `notes` column (agent's reasoning for each transition)
--
-- NOTE: The partial UNIQUE index on (spec_id) WHERE ended_at IS NULL
-- is intentionally NOT added in v1 (proposal A2 + design §6.2). DB-level
-- enforcement of "at most one open phase per spec" is deferred to the
-- follow-up change `witsaba-core-tables-append-only-enforcement`.
CREATE TABLE task (
    id              BIGSERIAL    PRIMARY KEY,
    milestone_id    BIGINT       NOT NULL REFERENCES milestone(requirement_id),
    title           TEXT         NOT NULL,
    description     TEXT,
    created_at      TIMESTAMPTZ  NOT NULL DEFAULT now(),
    updated_at      TIMESTAMPTZ  NOT NULL DEFAULT now()
);

CREATE INDEX idx_task_milestone_id ON task(milestone_id);

CREATE TABLE spec (
    id              BIGSERIAL    PRIMARY KEY,
    task_id         BIGINT       NOT NULL REFERENCES task(id),
    content         TEXT         NOT NULL,
    start_date      DATE,
    end_date        DATE,
    created_at      TIMESTAMPTZ  NOT NULL DEFAULT now(),
    updated_at      TIMESTAMPTZ  NOT NULL DEFAULT now(),
    CONSTRAINT spec_dates_valid
        CHECK (end_date IS NULL OR start_date IS NULL OR end_date >= start_date)
);

CREATE INDEX idx_spec_task_id ON spec(task_id);

CREATE TABLE spec_phase (
    id              BIGSERIAL    PRIMARY KEY,
    spec_id         BIGINT       NOT NULL REFERENCES spec(id),
    phase           TEXT         NOT NULL,
    started_at      TIMESTAMPTZ  NOT NULL DEFAULT now(),
    ended_at        TIMESTAMPTZ,
    notes           TEXT,
    created_at      TIMESTAMPTZ  NOT NULL DEFAULT now(),
    updated_at      TIMESTAMPTZ  NOT NULL DEFAULT now(),
    CONSTRAINT spec_phase_phase_check CHECK (phase IN (
        'tdd_red',
        'implementation',
        'tdd_green',
        'verify',
        'pr',
        'technical_ai_review',
        'ai_approved',
        'human_approved'
    )),
    CONSTRAINT spec_phase_natural_key UNIQUE (spec_id, phase, started_at),
    CONSTRAINT spec_phase_dates_valid
        CHECK (ended_at IS NULL OR ended_at >= started_at)
);

COMMENT ON COLUMN spec_phase.notes IS
    'Agent''s reasoning for each phase transition. Required on re-entries (e.g., returning to tdd_red after a technical_ai_review found a gap). The audit trail of (spec_id, phase, started_at, notes, ended_at) is the agent''s primary memory of WHY a transition happened.';

CREATE INDEX idx_spec_phase_spec_id        ON spec_phase(spec_id);
CREATE INDEX idx_spec_phase_current_state  ON spec_phase(spec_id) WHERE ended_at IS NULL;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE spec_phase;
DROP TABLE spec;
DROP TABLE task;
-- +goose StatementEnd
