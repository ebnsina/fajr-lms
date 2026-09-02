-- +goose Up
-- +goose StatementBegin

-- Auth records are keyed by user, not tenant, so RLS does not apply to them.
CREATE TYPE otp_purpose AS ENUM ('login', 'verify_contact');

CREATE TABLE otp_challenges (
  id           uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  destination  text NOT NULL CHECK (length(destination) BETWEEN 3 AND 320),
  purpose      otp_purpose NOT NULL DEFAULT 'login',
  code_hash    bytea NOT NULL,
  attempts     smallint NOT NULL DEFAULT 0 CHECK (attempts >= 0),
  consumed_at  timestamptz,
  expires_at   timestamptz NOT NULL,
  created_at   timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX otp_lookup_idx ON otp_challenges (destination, purpose, created_at DESC);

CREATE TABLE sessions (
  id           uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  user_id      uuid NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  token_hash   bytea NOT NULL UNIQUE,
  user_agent   text NOT NULL DEFAULT '',
  ip           inet,
  last_used_at timestamptz NOT NULL DEFAULT now(),
  revoked_at   timestamptz,
  expires_at   timestamptz NOT NULL,
  created_at   timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX sessions_user_idx ON sessions (user_id) WHERE revoked_at IS NULL;

-- Sessions join users, which is under RLS. The view runs as its owner, so
-- authentication can read the row without a tenant scope existing yet.
CREATE VIEW live_sessions WITH (security_invoker = false) AS
  SELECT s.id AS session_id, s.user_id, s.token_hash, u.full_name, s.expires_at
  FROM sessions s JOIN users u ON u.id = s.user_id
  WHERE s.revoked_at IS NULL AND s.expires_at > now();

GRANT SELECT ON live_sessions TO fajr_app;

GRANT SELECT, INSERT, UPDATE, DELETE ON otp_challenges, sessions TO fajr_app;


-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP VIEW IF EXISTS live_sessions;
DROP TABLE IF EXISTS sessions;
DROP TABLE IF EXISTS otp_challenges;
DROP TYPE IF EXISTS otp_purpose;
-- +goose StatementEnd
