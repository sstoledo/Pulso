-- name: CreateSession :one
INSERT INTO sessions (user_id, token_hash, expires_at)
VALUES ($1, $2, $3)
RETURNING *;

-- name: GetSessionUser :one
SELECT u.id, u.name, u.email, u.created_at, s.id AS session_id, s.expires_at
FROM sessions AS s
JOIN users AS u ON u.id = s.user_id
WHERE s.token_hash = $1
  AND s.revoked_at IS NULL
  AND s.expires_at > now();

-- name: RevokeSession :execrows
UPDATE sessions
SET revoked_at = now()
WHERE token_hash = $1
  AND revoked_at IS NULL;
