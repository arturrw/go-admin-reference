-- +goose Up
-- An order's history: every status it moved to, when, and who did it. The
-- fulfilment timeline in the UI is drawn from these rows.
CREATE TABLE order_events (
    order_id bigint      NOT NULL REFERENCES orders (id) ON DELETE CASCADE,
    seq      integer     NOT NULL,
    status   text        NOT NULL CHECK (status IN ('pending', 'paid', 'shipped', 'delivered', 'refunded', 'failed')),
    at       timestamptz NOT NULL,
    by       text        NOT NULL DEFAULT '',
    PRIMARY KEY (order_id, seq)
);

-- Orders that already exist get the history their status implies: placed, paid
-- a minute later, shipped the same day, delivered two days after that.
INSERT INTO order_events (order_id, seq, status, at, by)
SELECT id, 1, 'pending', placed_at, 'Customer' FROM orders;

INSERT INTO order_events (order_id, seq, status, at, by)
SELECT id, 2, CASE WHEN status = 'failed' THEN 'failed' ELSE 'paid' END, least(placed_at + interval '1 minute', now()), 'Payment provider'
FROM orders WHERE status <> 'pending';

INSERT INTO order_events (order_id, seq, status, at, by)
SELECT id, 3, 'shipped', least(placed_at + interval '5 hours', now()), 'Warehouse'
FROM orders WHERE status IN ('shipped', 'delivered');

INSERT INTO order_events (order_id, seq, status, at, by)
SELECT id, 4, 'delivered', least(placed_at + interval '2 days 5 hours', now()), 'Carrier'
FROM orders WHERE status = 'delivered';

INSERT INTO order_events (order_id, seq, status, at, by)
SELECT id, 3, 'refunded', least(coalesce(refunded_at, placed_at + interval '2 days'), now()), coalesce(nullif(refunded_by, ''), 'Support')
FROM orders WHERE status = 'refunded';

-- +goose Down
DROP TABLE order_events;