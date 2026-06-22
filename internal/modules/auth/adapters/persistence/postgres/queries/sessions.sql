-- name: CreateSession :exec
INSERT INTO auth_sessions (
  id,
  user_id,
  refresh_jti,
  revoked_at,
  expires_at,
  created_at,
  updated_at
)
VALUES ($1, $2, $3, $4, $5, $6, $6);

-- name: GetSessionByID :one
SELECT
  id,
  user_id,
  refresh_jti,
  revoked_at,
  expires_at,
  created_at
FROM auth_sessions
WHERE id = $1;

-- name: RotateSessionRefresh :execrows
UPDATE auth_sessions
SET
  refresh_jti = $1,
  expires_at = $2,
  updated_at = $3
WHERE id = $4
  AND refresh_jti = $5
  AND revoked_at IS NULL
  AND expires_at > $3;

-- name: RevokeSession :exec
UPDATE auth_sessions
SET
  revoked_at = COALESCE(revoked_at, $1),
  updated_at = $1
WHERE id = $2;

-- name: RevokeSessionsByUserID :exec
UPDATE auth_sessions
SET
  revoked_at = COALESCE(revoked_at, $1),
  updated_at = $1
WHERE user_id = $2
  AND revoked_at IS NULL;
