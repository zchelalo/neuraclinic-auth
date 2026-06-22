CREATE TABLE auth_sessions (
  id uuid PRIMARY KEY,
  user_id uuid NOT NULL,
  refresh_jti uuid UNIQUE NOT NULL,
  revoked_at timestamptz,
  expires_at timestamptz NOT NULL,
  created_at timestamptz NOT NULL DEFAULT now(),
  updated_at timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX idx_auth_sessions_user_active
  ON auth_sessions (user_id, expires_at)
  WHERE revoked_at IS NULL;

CREATE TABLE password_reset_requests (
  id uuid PRIMARY KEY,
  user_id uuid NOT NULL,
  email varchar(254) NOT NULL,
  otp_hash text NOT NULL,
  reset_token_hash text,
  attempts int NOT NULL DEFAULT 0,
  expires_at timestamptz NOT NULL,
  reset_token_expires_at timestamptz,
  used_at timestamptz,
  created_at timestamptz NOT NULL DEFAULT now(),
  updated_at timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX idx_password_reset_requests_user_active
  ON password_reset_requests (user_id, expires_at)
  WHERE used_at IS NULL;

CREATE UNIQUE INDEX idx_password_reset_requests_reset_token_hash
  ON password_reset_requests (reset_token_hash)
  WHERE reset_token_hash IS NOT NULL;

CREATE TABLE permissions (
  id uuid PRIMARY KEY,
  key varchar(80) UNIQUE NOT NULL,
  description text NOT NULL,
  created_at timestamptz NOT NULL DEFAULT now(),
  updated_at timestamptz NOT NULL DEFAULT now(),
  deleted_at timestamptz
);

CREATE TABLE role_permissions (
  role_key varchar(80) NOT NULL,
  permission_key varchar(80) NOT NULL REFERENCES permissions(key) ON DELETE CASCADE,
  effect varchar(20) NOT NULL CHECK (effect IN ('allow', 'deny')),
  PRIMARY KEY (role_key, permission_key)
);

CREATE TABLE user_permission_overrides (
  user_id uuid NOT NULL,
  permission_key varchar(80) NOT NULL REFERENCES permissions(key) ON DELETE CASCADE,
  effect varchar(20) NOT NULL CHECK (effect IN ('allow', 'deny')),
  created_at timestamptz NOT NULL DEFAULT now(),
  updated_at timestamptz NOT NULL DEFAULT now(),
  PRIMARY KEY (user_id, permission_key)
);

INSERT INTO permissions (id, key, description) VALUES
  ('00000000-0000-0000-0000-000000000001', 'PERMISSION_KEY_PATIENT_CREATE', 'Create patients'),
  ('00000000-0000-0000-0000-000000000002', 'PERMISSION_KEY_PATIENT_VIEW', 'View patients'),
  ('00000000-0000-0000-0000-000000000003', 'PERMISSION_KEY_PATIENT_EDIT', 'Edit patients'),
  ('00000000-0000-0000-0000-000000000004', 'PERMISSION_KEY_PATIENT_DELETE', 'Delete patients'),
  ('00000000-0000-0000-0000-000000000010', 'PERMISSION_KEY_APPOINTMENT_CREATE', 'Create appointments'),
  ('00000000-0000-0000-0000-000000000011', 'PERMISSION_KEY_APPOINTMENT_VIEW', 'View appointments'),
  ('00000000-0000-0000-0000-000000000012', 'PERMISSION_KEY_APPOINTMENT_EDIT', 'Edit appointments'),
  ('00000000-0000-0000-0000-000000000020', 'PERMISSION_KEY_NOTE_CREATE', 'Create notes'),
  ('00000000-0000-0000-0000-000000000021', 'PERMISSION_KEY_NOTE_VIEW', 'View notes'),
  ('00000000-0000-0000-0000-000000000022', 'PERMISSION_KEY_NOTE_EDIT', 'Edit notes'),
  ('00000000-0000-0000-0000-000000000023', 'PERMISSION_KEY_NOTE_DELETE', 'Delete notes'),
  ('00000000-0000-0000-0000-000000000030', 'PERMISSION_KEY_USER_CREATE', 'Create users'),
  ('00000000-0000-0000-0000-000000000031', 'PERMISSION_KEY_USER_VIEW', 'View users'),
  ('00000000-0000-0000-0000-000000000032', 'PERMISSION_KEY_USER_EDIT', 'Edit users'),
  ('00000000-0000-0000-0000-000000000033', 'PERMISSION_KEY_USER_DELETE', 'Delete users'),
  ('00000000-0000-0000-0000-000000000040', 'PERMISSION_KEY_ROLE_CREATE', 'Create roles'),
  ('00000000-0000-0000-0000-000000000041', 'PERMISSION_KEY_ROLE_VIEW', 'View roles'),
  ('00000000-0000-0000-0000-000000000042', 'PERMISSION_KEY_ROLE_EDIT', 'Edit roles'),
  ('00000000-0000-0000-0000-000000000050', 'PERMISSION_KEY_PSYCHOLOGIST_CREATE', 'Create psychologists'),
  ('00000000-0000-0000-0000-000000000051', 'PERMISSION_KEY_PSYCHOLOGIST_VIEW', 'View psychologists'),
  ('00000000-0000-0000-0000-000000000052', 'PERMISSION_KEY_PSYCHOLOGIST_EDIT', 'Edit psychologists'),
  ('00000000-0000-0000-0000-000000000060', 'PERMISSION_KEY_ADMIN_CREATE', 'Create admins'),
  ('00000000-0000-0000-0000-000000000061', 'PERMISSION_KEY_ADMIN_VIEW', 'View admins'),
  ('00000000-0000-0000-0000-000000000062', 'PERMISSION_KEY_ADMIN_EDIT', 'Edit admins'),
  ('00000000-0000-0000-0000-000000000070', 'PERMISSION_KEY_FILE_UPLOAD', 'Upload files'),
  ('00000000-0000-0000-0000-000000000071', 'PERMISSION_KEY_FILE_VIEW', 'View files'),
  ('00000000-0000-0000-0000-000000000072', 'PERMISSION_KEY_FILE_DELETE', 'Delete files');

INSERT INTO role_permissions (role_key, permission_key, effect)
SELECT 'ROLE_KEY_ADMIN', key, 'allow'
FROM permissions
WHERE deleted_at IS NULL;

INSERT INTO role_permissions (role_key, permission_key, effect) VALUES
  ('ROLE_KEY_PSYCHOLOGIST', 'PERMISSION_KEY_PATIENT_CREATE', 'allow'),
  ('ROLE_KEY_PSYCHOLOGIST', 'PERMISSION_KEY_PATIENT_VIEW', 'allow'),
  ('ROLE_KEY_PSYCHOLOGIST', 'PERMISSION_KEY_PATIENT_EDIT', 'allow'),
  ('ROLE_KEY_PSYCHOLOGIST', 'PERMISSION_KEY_APPOINTMENT_CREATE', 'allow'),
  ('ROLE_KEY_PSYCHOLOGIST', 'PERMISSION_KEY_APPOINTMENT_VIEW', 'allow'),
  ('ROLE_KEY_PSYCHOLOGIST', 'PERMISSION_KEY_APPOINTMENT_EDIT', 'allow'),
  ('ROLE_KEY_PSYCHOLOGIST', 'PERMISSION_KEY_NOTE_CREATE', 'allow'),
  ('ROLE_KEY_PSYCHOLOGIST', 'PERMISSION_KEY_NOTE_VIEW', 'allow'),
  ('ROLE_KEY_PSYCHOLOGIST', 'PERMISSION_KEY_NOTE_EDIT', 'allow'),
  ('ROLE_KEY_PSYCHOLOGIST', 'PERMISSION_KEY_NOTE_DELETE', 'allow'),
  ('ROLE_KEY_PSYCHOLOGIST', 'PERMISSION_KEY_PSYCHOLOGIST_VIEW', 'allow'),
  ('ROLE_KEY_PSYCHOLOGIST', 'PERMISSION_KEY_FILE_UPLOAD', 'allow'),
  ('ROLE_KEY_PSYCHOLOGIST', 'PERMISSION_KEY_FILE_VIEW', 'allow'),
  ('ROLE_KEY_PSYCHOLOGIST', 'PERMISSION_KEY_FILE_DELETE', 'allow');
