-- name: ListCustomers :many
SELECT * FROM customers_v
WHERE (sqlc.arg(segment)::text = '' OR segment = sqlc.arg(segment)::text)
  AND (sqlc.arg(q)::text = '' OR (name || ' ' || email || ' ' || city) ILIKE '%' || sqlc.arg(q)::text || '%')
ORDER BY ltv_cents DESC, id;

-- name: GetCustomer :one
SELECT * FROM customers_v WHERE id = $1;

-- name: CustomerSegments :many
SELECT segment, count(*)::int AS n, coalesce(sum(ltv_cents), 0)::bigint AS ltv_cents
FROM customers_v GROUP BY segment;

-- name: ListCustomerNotes :many
SELECT * FROM customer_notes WHERE customer_id = ANY(sqlc.arg(customer_ids)::bigint[]) ORDER BY created_at DESC;

-- name: AddCustomerNote :one
INSERT INTO customer_notes (customer_id, author, body) VALUES ($1, $2, $3) RETURNING *;

-- name: DeleteCustomerNote :one
DELETE FROM customer_notes WHERE id = sqlc.arg(id) AND customer_id = sqlc.arg(customer_id) RETURNING *;

-- name: CreateCustomer :one
INSERT INTO customers (name, email, phone, country, address_line1, city, postal_code, tags, accepts_marketing, source)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
RETURNING id;
