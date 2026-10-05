-- name: ListActivity :many
SELECT * FROM activity
WHERE (sqlc.arg(actor_id)::bigint = 0 OR actor_id = sqlc.arg(actor_id)::bigint)
  AND (sqlc.arg(kind)::text = '' OR kind = sqlc.arg(kind)::text)
  AND (NOT sqlc.arg(exclude_auth)::bool OR kind <> 'auth')
  AND (sqlc.arg(q)::text = '' OR (actor || ' ' || message) ILIKE '%' || sqlc.arg(q)::text || '%')
ORDER BY at DESC, id DESC
LIMIT sqlc.arg(lim)::int OFFSET sqlc.arg(off)::int;

-- name: CountActivity :one
SELECT count(*)::int FROM activity
WHERE (sqlc.arg(actor_id)::bigint = 0 OR actor_id = sqlc.arg(actor_id)::bigint)
  AND (sqlc.arg(kind)::text = '' OR kind = sqlc.arg(kind)::text)
  AND (NOT sqlc.arg(exclude_auth)::bool OR kind <> 'auth')
  AND (sqlc.arg(q)::text = '' OR (actor || ' ' || message) ILIKE '%' || sqlc.arg(q)::text || '%');

-- name: AddActivity :one
INSERT INTO activity (kind, actor, actor_id, message, entity, entity_id)
VALUES ($1, $2, sqlc.narg(actor_id), $3, $4, $5)
RETURNING *;