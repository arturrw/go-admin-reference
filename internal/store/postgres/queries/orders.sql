-- name: ListOrders :many
SELECT o.id, o.status, o.payment, o.total_cents, o.placed_at,
       c.id AS customer_id, c.name AS customer_name, c.email AS customer_email,
       c.country AS customer_country, c.segment AS customer_segment
FROM orders o
JOIN customers_v c ON c.id = o.customer_id
WHERE (sqlc.arg(status)::text = '' OR o.status = sqlc.arg(status)::text)
  AND (sqlc.arg(customer_id)::bigint = 0 OR o.customer_id = sqlc.arg(customer_id)::bigint)
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
  AND (sqlc.arg(q)::text = ''
       OR (c.name || ' ' || c.email) ILIKE '%' || sqlc.arg(q)::text || '%'
       OR ('#' || o.id::text) ILIKE '%' || sqlc.arg(q)::text || '%');

-- name: GetOrder :one
SELECT o.id, o.status, o.payment, o.total_cents, o.placed_at,
       c.id AS customer_id, c.name AS customer_name, c.email AS customer_email,
       c.country AS customer_country, c.segment AS customer_segment
FROM orders o
JOIN customers_v c ON c.id = o.customer_id
WHERE o.id = $1;

-- name: ListOrderItems :many
SELECT * FROM order_items WHERE order_id = ANY(sqlc.arg(order_ids)::bigint[]) ORDER BY order_id, line;

-- name: OrderCounts :many
SELECT status, count(*)::int AS n FROM orders GROUP BY status;

-- name: UpdateOrderStatus :execrows
UPDATE orders SET status = $2 WHERE id = $1;
