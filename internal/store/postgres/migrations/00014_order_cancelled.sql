-- +goose Up
-- A new order status: staff can cancel an order that is still pending. A
-- cancelled order is not revenue, so the customer metrics leave it out.
ALTER TABLE orders DROP CONSTRAINT orders_status_check;
ALTER TABLE orders ADD CONSTRAINT orders_status_check
    CHECK (status IN ('pending', 'paid', 'shipped', 'delivered', 'refunded', 'failed', 'cancelled'));
ALTER TABLE order_events DROP CONSTRAINT order_events_status_check;
ALTER TABLE order_events ADD CONSTRAINT order_events_status_check
    CHECK (status IN ('pending', 'paid', 'shipped', 'delivered', 'refunded', 'failed', 'cancelled'));

DROP VIEW customers_v;
CREATE VIEW customers_v AS
SELECT c.*,
       coalesce(s.orders, 0)::int        AS orders,
       coalesce(s.ltv_cents, 0)::bigint  AS ltv_cents,
       s.last_order_at::timestamptz      AS last_order_at,
       CASE
           WHEN s.last_order_at IS NOT NULL AND now() - s.last_order_at > interval '45 days' THEN 'At risk'
           WHEN coalesce(s.ltv_cents, 0) >= 150000 OR coalesce(s.orders, 0) >= 8 THEN 'VIP'
           WHEN coalesce(s.orders, 0) <= 1 OR now() - c.created_at < interval '30 days' THEN 'New'
           ELSE 'Regular'
       END::text AS segment
FROM customers c
LEFT JOIN LATERAL (
    SELECT count(*) FILTER (WHERE o.status NOT IN ('refunded', 'failed', 'cancelled'))        AS orders,
           sum(o.total_cents) FILTER (WHERE o.status NOT IN ('refunded', 'failed', 'cancelled')) AS ltv_cents,
           max(o.placed_at)                                                                 AS last_order_at
    FROM orders o
    WHERE o.customer_id = c.id
) s ON true;

-- +goose Down
DROP VIEW customers_v;
CREATE VIEW customers_v AS
SELECT c.*,
       coalesce(s.orders, 0)::int        AS orders,
       coalesce(s.ltv_cents, 0)::bigint  AS ltv_cents,
       s.last_order_at::timestamptz      AS last_order_at,
       CASE
           WHEN s.last_order_at IS NOT NULL AND now() - s.last_order_at > interval '45 days' THEN 'At risk'
           WHEN coalesce(s.ltv_cents, 0) >= 150000 OR coalesce(s.orders, 0) >= 8 THEN 'VIP'
           WHEN coalesce(s.orders, 0) <= 1 OR now() - c.created_at < interval '30 days' THEN 'New'
           ELSE 'Regular'
       END::text AS segment
FROM customers c
LEFT JOIN LATERAL (
    SELECT count(*) FILTER (WHERE o.status NOT IN ('refunded', 'failed'))                     AS orders,
           sum(o.total_cents) FILTER (WHERE o.status NOT IN ('refunded', 'failed'))           AS ltv_cents,
           max(o.placed_at)                                                                 AS last_order_at
    FROM orders o
    WHERE o.customer_id = c.id
) s ON true;
UPDATE orders SET status = 'failed' WHERE status = 'cancelled';
UPDATE order_events SET status = 'failed' WHERE status = 'cancelled';
ALTER TABLE order_events DROP CONSTRAINT order_events_status_check;
ALTER TABLE order_events ADD CONSTRAINT order_events_status_check
    CHECK (status IN ('pending', 'paid', 'shipped', 'delivered', 'refunded', 'failed'));
ALTER TABLE orders DROP CONSTRAINT orders_status_check;
ALTER TABLE orders ADD CONSTRAINT orders_status_check
    CHECK (status IN ('pending', 'paid', 'shipped', 'delivered', 'refunded', 'failed'));
