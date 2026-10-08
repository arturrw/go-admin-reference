-- +goose Up
-- Units sold, revenue and the sparkline are read from orders now, so the
-- stored copies (which could only drift from them) go away.
ALTER TABLE products DROP COLUMN sold_30d, DROP COLUMN trend;

-- +goose Down
ALTER TABLE products ADD COLUMN sold_30d integer NOT NULL DEFAULT 0, ADD COLUMN trend integer[] NOT NULL DEFAULT '{}';
