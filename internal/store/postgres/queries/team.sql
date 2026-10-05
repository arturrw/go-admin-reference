-- name: ListMembers :many
SELECT * FROM members WHERE (sqlc.arg(role)::text = '' OR role = sqlc.arg(role)::text) ORDER BY id;

-- name: GetMember :one
SELECT * FROM members WHERE id = $1;

-- name: MemberByEmail :one
SELECT * FROM members WHERE lower(email) = lower(sqlc.arg(email)::text);

-- name: TouchMember :exec
UPDATE members SET last_active_at = now() WHERE id = $1;

-- name: CreateMember :one
INSERT INTO members (name, email, role) VALUES ($1, $2, $3) RETURNING *;

-- name: UpdateMember :one
UPDATE members SET name = $2, email = $3, role = $4 WHERE id = $1 RETURNING *;

-- name: SetMemberAccess :one
UPDATE members SET granted = sqlc.arg(granted)::text[], revoked = sqlc.arg(revoked)::text[] WHERE id = sqlc.arg(id) RETURNING *;

-- name: SetMemberStatus :one
UPDATE members SET status = $2 WHERE id = $1 RETURNING *;

-- name: DeleteMember :execrows
DELETE FROM members WHERE id = $1;

-- name: CreateSession :exec
INSERT INTO sessions (token_hash, member_id, expires_at) VALUES ($1, $2, $3);

-- name: TouchSession :one
-- Sliding expiry: returns the member only while the session is still valid.
UPDATE sessions SET expires_at = sqlc.arg(expires_at)
WHERE token_hash = sqlc.arg(token_hash) AND expires_at > now()
RETURNING member_id;

-- name: DeleteSession :exec
DELETE FROM sessions WHERE token_hash = $1;

-- name: DeleteMemberSessions :exec
DELETE FROM sessions WHERE member_id = $1;

-- name: DeleteExpiredSessions :execrows
DELETE FROM sessions WHERE expires_at <= now();
