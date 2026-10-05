-- name: RevenueSeries :many
SELECT * FROM (
    SELECT * FROM revenue_daily ORDER BY day DESC LIMIT sqlc.arg(days)::int
) r ORDER BY day;

-- name: OrdersHeatmap :many
SELECT dow, hour, orders FROM orders_heatmap;


-- name: CountMembers :one
SELECT count(*)::int FROM members;
