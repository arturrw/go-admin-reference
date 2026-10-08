-- +goose Up
-- When a member last opened their notifications; newer feed entries are unread.
ALTER TABLE members ADD COLUMN notifications_read_at timestamptz;

-- +goose Down
ALTER TABLE members DROP COLUMN notifications_read_at;