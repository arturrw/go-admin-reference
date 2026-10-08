-- name: ListActivity :many
SELECT * FROM activity
WHERE (sqlc.arg(actor_id)::bigint = 0 OR actor_id = sqlc.arg(actor_id)::bigint)
  AND (sqlc.arg(kind)::text = '' OR kind = sqlc.arg(kind)::text)
  AND (NOT sqlc.arg(exclude_auth)::bool OR kind NOT IN ('auth', 'alert'))
  AND (sqlc.arg(notify_member)::bigint = 0 OR (kind <> 'auth'
       AND (kind <> 'alert' OR (entity = 'member' AND entity_id = sqlc.arg(notify_member)::bigint))
       AND actor_id IS DISTINCT FROM sqlc.arg(notify_member)::bigint))
  AND (sqlc.arg(q)::text = '' OR (actor || ' ' || message) ILIKE '%' || sqlc.arg(q)::text || '%')
ORDER BY at DESC, id DESC
LIMIT sqlc.arg(lim)::int OFFSET sqlc.arg(off)::int;

-- name: CountActivity :one
SELECT count(*)::int FROM activity
WHERE (sqlc.arg(actor_id)::bigint = 0 OR actor_id = sqlc.arg(actor_id)::bigint)
  AND (sqlc.arg(kind)::text = '' OR kind = sqlc.arg(kind)::text)
  AND (NOT sqlc.arg(exclude_auth)::bool OR kind NOT IN ('auth', 'alert'))
  AND (sqlc.arg(notify_member)::bigint = 0 OR (kind <> 'auth'
       AND (kind <> 'alert' OR (entity = 'member' AND entity_id = sqlc.arg(notify_member)::bigint))
       AND actor_id IS DISTINCT FROM sqlc.arg(notify_member)::bigint))
  AND (sqlc.arg(q)::text = '' OR (actor || ' ' || message) ILIKE '%' || sqlc.arg(q)::text || '%');

-- name: AddActivity :one
INSERT INTO activity (kind, actor, actor_id, message, entity, entity_id)
VALUES ($1, $2, sqlc.narg(actor_id), $3, $4, $5)
RETURNING *;