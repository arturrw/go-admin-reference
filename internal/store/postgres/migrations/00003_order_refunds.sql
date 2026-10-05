-- +goose Up
-- A refund keeps why it happened, who issued it and when, so the order and the
-- customer's purchase history can show it later.
ALTER TABLE orders
    ADD COLUMN refund_reason text        NOT NULL DEFAULT '',
    ADD COLUMN refunded_by   text        NOT NULL DEFAULT '',
    ADD COLUMN refunded_at   timestamptz;

-- +goose Down
ALTER TABLE orders DROP COLUMN refunded_at, DROP COLUMN refunded_by, DROP COLUMN refund_reason;
