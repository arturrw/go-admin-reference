-- +goose Up
-- Workspace settings: one row of JSON, so new settings need no migration. The
-- check keeps it to a single row.
CREATE TABLE settings (
    id         boolean PRIMARY KEY DEFAULT true CHECK (id),
    data       jsonb       NOT NULL,
    updated_at timestamptz NOT NULL DEFAULT now()
);

-- +goose Down
DROP TABLE settings;
