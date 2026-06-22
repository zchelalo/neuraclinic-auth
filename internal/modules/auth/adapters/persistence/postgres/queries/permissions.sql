-- name: ListPermissions :many
SELECT
  id,
  key,
  description,
  created_at,
  updated_at,
  deleted_at
FROM permissions
WHERE deleted_at IS NULL
ORDER BY key ASC;

-- name: ListRolePermissionEffects :many
SELECT
  permission_key,
  effect
FROM role_permissions
WHERE role_key = $1;

-- name: ListUserPermissionOverrideEffects :many
SELECT
  permission_key,
  effect
FROM user_permission_overrides
WHERE user_id = $1;
