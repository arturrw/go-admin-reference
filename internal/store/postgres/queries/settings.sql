-- name: GetSettings :one
SELECT data FROM settings WHERE id;

-- name: SaveSettings :exec
INSERT INTO settings (id, data) VALUES (true, $1)
ON CONFLICT (id) DO UPDATE SET data = EXCLUDED.data, updated_at = now();