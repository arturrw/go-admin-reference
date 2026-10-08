-- name: ListOrders :many
SELECT o.id, o.status, o.payment, o.total_cents, o.placed_at, o.refund_reason, o.refunded_by, o.refunded_at,
       c.id AS customer_id, c.name AS customer_name, c.email AS customer_email,
       c.country AS customer_country, c.segment AS customer_segment
FROM orders o
JOIN customers_v c ON c.id = o.customer_id
WHERE (sqlc.arg(status)::text = '' OR o.status = sqlc.arg(status)::text)
  AND (sqlc.arg(customer_id)::bigint = 0 OR o.customer_id = sqlc.arg(customer_id)::bigint)
  AND (sqlc.narg(placed_from)::timestamptz IS NULL OR o.placed_at >= sqlc.narg(placed_from)::timestamptz)
  AND (sqlc.narg(placed_to)::timestamptz IS NULL OR o.placed_at < sqlc.narg(placed_to)::timestamptz)
  AND (sqlc.arg(q)::text = ''
       OR (c.name || ' ' || c.email) ILIKE '%' || sqlc.arg(q)::text || '%'
       OR ('#' || o.id::text) ILIKE '%' || sqlc.arg(q)::text || '%')
ORDER BY o.placed_at DESC, o.id DESC
LIMIT sqlc.arg(lim)::int OFFSET sqlc.arg(off)::int;

-- name: CountOrders :one
SELECT count(*)::int
FROM orders o
JOIN customers c ON c.id = o.customer_id
WHERE (sqlc.arg(status)::text = '' OR o.status = sqlc.arg(status)::text)
  AND (sqlc.arg(customer_id)::bigint = 0 OR o.customer_id = sqlc.arg(customer_id)::bigint)
  AND (sqlc.narg(placed_from)::timestamptz IS NULL OR o.placed_at >= sqlc.narg(placed_from)::timestamptz)
  AND (sqlc.narg(placed_to)::timestamptz IS NULL OR o.placed_at < sqlc.narg(placed_to)::timestamptz)
  AND (sqlc.arg(q)::text = ''
       OR (c.name || ' ' || c.email) ILIKE '%' || sqlc.arg(q)::text || '%'
       OR ('#' || o.id::text) ILIKE '%' || sqlc.arg(q)::text || '%');

-- name: GetOrder :one
SELECT o.id, o.status, o.payment, o.total_cents, o.placed_at, o.refund_reason, o.refunded_by, o.refunded_at,
       c.id AS customer_id, c.name AS customer_name, c.email AS customer_email,
       c.country AS customer_country, c.segment AS customer_segment
FROM orders o
JOIN customers_v c ON c.id = o.customer_id
WHERE o.id = $1;

-- name: ListOrderItems :many
-- image_url is the product's current cover, so orders follow gallery edits;
-- the snapshot taken at purchase is only used once the product is deleted.
SELECT oi.order_id, oi.line, oi.product_id, oi.name, oi.sku, oi.category, oi.hue, oi.qty, oi.price_cents,
       (CASE WHEN p.id IS NULL THEN oi.image_url
             ELSE coalesce((SELECT pi.url FROM product_images pi
                            WHERE pi.product_id = p.id ORDER BY pi.position LIMIT 1), '')
        END)::text AS image_url
FROM order_items oi
LEFT JOIN products p ON p.id = oi.product_id
WHERE oi.order_id = ANY(sqlc.arg(order_ids)::bigint[])
ORDER BY oi.order_id, oi.line;

-- name: OrderCounts :many
SELECT status, count(*)::int AS n FROM orders GROUP BY status;

-- name: ReserveStock :one
-- Takes qty units off an active product in one step, so stock cannot go
-- negative; no row means missing, not for sale, or not enough stock.
UPDATE products SET stock = stock - sqlc.arg(qty)::int, updated_at = now()
WHERE id = sqlc.arg(id) AND status = 'active' AND stock >= sqlc.arg(qty)::int
RETURNING *;

-- name: CreateOrder :one
INSERT INTO orders (customer_id, status, payment, total_cents)
VALUES ($1, 'pending', $2, $3)
RETURNING id;

-- name: AddOrderItem :exec
INSERT INTO order_items (order_id, line, product_id, name, sku, category, hue, image_url, qty, price_cents)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10);

-- name: UpdateOrderStatus :execrows
-- The refund columns are set with a refund and cleared by any other status.
UPDATE orders
SET status = $2, refund_reason = $3, refunded_by = $4, refunded_at = sqlc.narg(refunded_at)
WHERE id = $1;

-- name: ListOrderEvents :many
SELECT * FROM order_events WHERE order_id = ANY(sqlc.arg(order_ids)::bigint[]) ORDER BY order_id, seq;

-- name: AddOrderEvent :exec
INSERT INTO order_events (order_id, seq, status, at, by)
VALUES (sqlc.arg(order_id), (SELECT coalesce(max(seq), 0) + 1 FROM order_events WHERE order_id = sqlc.arg(order_id)), sqlc.arg(status), sqlc.arg(at), sqlc.arg(by));
