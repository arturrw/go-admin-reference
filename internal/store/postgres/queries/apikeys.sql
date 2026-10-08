-- name: ListAPIKeys :many
SELECT * FROM api_keys WHERE revoked_at IS NULL ORDER BY created_at DESC, id DESC;

-- name: CountAPIKeys :one
SELECT count(*)::int FROM api_keys;

-- name: CreateAPIKey :one
INSERT INTO api_keys (name, scope, last4, token_hash, created_by)
VALUES ($1, $2, $3, $4, $5)
RETURNING *;

-- name: APIKeyByHash :one
SELECT * FROM api_keys WHERE token_hash = $1 AND revoked_at IS NULL;

-- name: TouchAPIKey :exec
UPDATE api_keys SET last_used_at = now() WHERE id = $1;

-- name: RevokeAPIKey :one
UPDATE api_keys SET revoked_at = now() WHERE id = $1 AND revoked_at IS NULL RETURNING *;