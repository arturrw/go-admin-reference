-- name: RevenueSeries :many
SELECT * FROM (
    SELECT * FROM revenue_daily ORDER BY day DESC LIMIT sqlc.arg(days)::int
) r ORDER BY day;

-- name: OrdersHeatmap :many
SELECT dow, hour, orders FROM orders_heatmap;


-- name: CountMembers :one
SELECT count(*)::int FROM members;

-- name: GetTarget :one
SELECT * FROM targets WHERE quarter = $1;

-- name: UpsertTarget :one
INSERT INTO targets (quarter, goal_cents, updated_by) VALUES ($1, $2, $3)
ON CONFLICT (quarter) DO UPDATE SET goal_cents = EXCLUDED.goal_cents, updated_by = EXCLUDED.updated_by, updated_at = now()
RETURNING *;
