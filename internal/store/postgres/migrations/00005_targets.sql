-- +goose Up
-- Quarterly revenue goals, set by the owner. Quarters without a row use the
-- default goal.
CREATE TABLE targets (
    quarter    text        PRIMARY KEY, -- e.g. 2026-Q4
    goal_cents bigint      NOT NULL CHECK (goal_cents > 0),
    updated_by text        NOT NULL DEFAULT '',
    updated_at timestamptz NOT NULL DEFAULT now()
);

-- +goose Down
DROP TABLE targets;