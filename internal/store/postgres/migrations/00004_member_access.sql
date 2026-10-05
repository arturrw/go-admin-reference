-- +goose Up
-- Per-member exceptions to the role's permissions, managed by the owner:
-- effective = role permissions + granted - revoked.
ALTER TABLE members
    ADD COLUMN granted text[] NOT NULL DEFAULT '{}',
    ADD COLUMN revoked text[] NOT NULL DEFAULT '{}';

-- +goose Down
ALTER TABLE members DROP COLUMN revoked, DROP COLUMN granted;
