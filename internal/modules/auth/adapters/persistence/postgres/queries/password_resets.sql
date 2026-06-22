-- name: CreatePasswordResetRequest :exec
INSERT INTO password_reset_requests (
  id,
  user_id,
  email,
  otp_hash,
  reset_token_hash,
  attempts,
  expires_at,
  reset_token_expires_at,
  used_at,
  created_at,
  updated_at
)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11);

-- name: LatestActivePasswordResetByEmail :one
SELECT
  id,
  user_id,
  email,
  otp_hash,
  reset_token_hash,
  attempts,
  expires_at,
  reset_token_expires_at,
  used_at,
  created_at,
  updated_at
FROM password_reset_requests
WHERE email = $1
  AND used_at IS NULL
  AND expires_at > $2
ORDER BY created_at DESC
LIMIT 1;

-- name: GetPasswordResetByResetTokenHash :one
SELECT
  id,
  user_id,
  email,
  otp_hash,
  reset_token_hash,
  attempts,
  expires_at,
  reset_token_expires_at,
  used_at,
  created_at,
  updated_at
FROM password_reset_requests
WHERE reset_token_hash = $1
  AND used_at IS NULL
  AND reset_token_expires_at > $2
LIMIT 1;

-- name: IncrementPasswordResetAttempts :exec
UPDATE password_reset_requests
SET
  attempts = attempts + 1,
  updated_at = $1
WHERE id = $2;

-- name: SetPasswordResetToken :execrows
UPDATE password_reset_requests
SET
  reset_token_hash = $1,
  reset_token_expires_at = $2,
  updated_at = $3
WHERE id = $4
  AND used_at IS NULL;

-- name: MarkPasswordResetUsed :execrows
UPDATE password_reset_requests
SET
  used_at = $1,
  updated_at = $1
WHERE id = $2
  AND used_at IS NULL;
