-- +goose Up
-- +goose StatementBegin

-- One OpenID Connect provider per school: the account a learner or a teacher
-- already has, instead of a code sent to their phone.
CREATE TABLE identity_providers (
  id              uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  tenant_id       uuid NOT NULL UNIQUE REFERENCES tenants(id) ON DELETE CASCADE,
  label           text NOT NULL DEFAULT 'Your school account' CHECK (length(btrim(label)) BETWEEN 1 AND 60),
  issuer          text NOT NULL CHECK (issuer ~ '^https://' AND length(issuer) <= 300),
  client_id       text NOT NULL CHECK (length(btrim(client_id)) BETWEEN 1 AND 300),
  client_secret   text NOT NULL CHECK (length(client_secret) <= 500),
  -- Empty means any address the provider vouches for.
  allowed_domains text[] NOT NULL DEFAULT '{}',
  join_role       member_role NOT NULL DEFAULT 'student',
  auto_join       boolean NOT NULL DEFAULT true,
  enabled         boolean NOT NULL DEFAULT true,
  created_at      timestamptz NOT NULL DEFAULT now(),
  updated_at      timestamptz NOT NULL DEFAULT now()
);

-- A login in flight. The state is the key, and it carries what the callback
-- needs to finish: which school, and the values only this browser knows.
CREATE TABLE sso_logins (
  state        text PRIMARY KEY CHECK (length(state) BETWEEN 20 AND 100),
  provider_id  uuid NOT NULL REFERENCES identity_providers(id) ON DELETE CASCADE,
  nonce        text NOT NULL CHECK (length(nonce) BETWEEN 20 AND 100),
  verifier     text NOT NULL CHECK (length(verifier) BETWEEN 43 AND 128),
  redirect_uri text NOT NULL CHECK (length(redirect_uri) <= 500),
  expires_at   timestamptz NOT NULL,
  created_at   timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX sso_logins_expiry_idx ON sso_logins (expires_at);

-- The account at the provider, so a person keeps their place here even if the
-- school changes the address on their account.
CREATE TABLE sso_identities (
  provider_id uuid NOT NULL REFERENCES identity_providers(id) ON DELETE CASCADE,
  subject     text NOT NULL CHECK (length(subject) BETWEEN 1 AND 255),
  user_id     uuid NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  created_at  timestamptz NOT NULL DEFAULT now(),
  PRIMARY KEY (provider_id, subject)
);

CREATE TRIGGER identity_providers_touch BEFORE UPDATE ON identity_providers
  FOR EACH ROW EXECUTE FUNCTION touch_updated_at();

ALTER TABLE identity_providers ENABLE ROW LEVEL SECURITY;
ALTER TABLE identity_providers FORCE ROW LEVEL SECURITY;
CREATE POLICY tenant_isolation ON identity_providers
  USING (tenant_id = current_tenant_id()) WITH CHECK (tenant_id = current_tenant_id());

-- Signing in happens before there is a tenant to scope to, so these two are
-- reached through named functions rather than left open.
ALTER TABLE sso_logins ENABLE ROW LEVEL SECURITY;
ALTER TABLE sso_logins FORCE ROW LEVEL SECURITY;
ALTER TABLE sso_identities ENABLE ROW LEVEL SECURITY;
ALTER TABLE sso_identities FORCE ROW LEVEL SECURITY;

CREATE FUNCTION sso_provider_for(p_slug text) RETURNS SETOF identity_providers
LANGUAGE sql STABLE SECURITY DEFINER SET search_path = public AS $$
  SELECT p.* FROM identity_providers p JOIN tenants t ON t.id = p.tenant_id
  WHERE t.slug = lower(p_slug) AND p.enabled AND t.status = 'active'
$$;

CREATE FUNCTION sso_provider_by_id(p_id uuid) RETURNS SETOF identity_providers
LANGUAGE sql STABLE SECURITY DEFINER SET search_path = public AS $$
  SELECT * FROM identity_providers WHERE id = p_id AND enabled
$$;

CREATE FUNCTION sso_login_start(p_state text, p_provider_id uuid, p_nonce text,
                                p_verifier text, p_redirect_uri text, p_ttl_s int)
RETURNS SETOF sso_logins
LANGUAGE sql VOLATILE SECURITY DEFINER SET search_path = public AS $$
  DELETE FROM sso_logins WHERE expires_at < now();
  INSERT INTO sso_logins (state, provider_id, nonce, verifier, redirect_uri, expires_at)
  VALUES (p_state, p_provider_id, p_nonce, p_verifier, p_redirect_uri, now() + make_interval(secs => p_ttl_s))
  RETURNING *;
$$;

-- A state is good once: taking it is what spends it.
CREATE FUNCTION sso_login_take(p_state text) RETURNS SETOF sso_logins
LANGUAGE sql VOLATILE SECURITY DEFINER SET search_path = public AS $$
  DELETE FROM sso_logins WHERE state = p_state AND expires_at > now() RETURNING *;
$$;

CREATE FUNCTION sso_user_for(p_provider_id uuid, p_subject text) RETURNS SETOF users
LANGUAGE sql STABLE SECURITY DEFINER SET search_path = public AS $$
  SELECT u.* FROM sso_identities i JOIN users u ON u.id = i.user_id
  WHERE i.provider_id = p_provider_id AND i.subject = p_subject
$$;

CREATE FUNCTION sso_link_user(p_provider_id uuid, p_subject text, p_user_id uuid) RETURNS void
LANGUAGE sql VOLATILE SECURITY DEFINER SET search_path = public AS $$
  INSERT INTO sso_identities (provider_id, subject, user_id)
  VALUES (p_provider_id, p_subject, p_user_id)
  ON CONFLICT (provider_id, subject) DO UPDATE SET user_id = excluded.user_id;
$$;

-- Joining a school on first sign-in, at the role the school chose.
CREATE FUNCTION sso_join(p_tenant_id uuid, p_user_id uuid, p_role member_role) RETURNS void
LANGUAGE sql VOLATILE SECURITY DEFINER SET search_path = public AS $$
  INSERT INTO memberships (tenant_id, user_id, role, status)
  VALUES (p_tenant_id, p_user_id, p_role, 'active')
  ON CONFLICT (tenant_id, user_id) DO NOTHING;
$$;

GRANT SELECT, INSERT, UPDATE, DELETE ON identity_providers TO fajr_app;

REVOKE ALL ON FUNCTION sso_provider_for(text) FROM PUBLIC;
REVOKE ALL ON FUNCTION sso_provider_by_id(uuid) FROM PUBLIC;
REVOKE ALL ON FUNCTION sso_login_start(text, uuid, text, text, text, int) FROM PUBLIC;
REVOKE ALL ON FUNCTION sso_login_take(text) FROM PUBLIC;
REVOKE ALL ON FUNCTION sso_user_for(uuid, text) FROM PUBLIC;
REVOKE ALL ON FUNCTION sso_link_user(uuid, text, uuid) FROM PUBLIC;
REVOKE ALL ON FUNCTION sso_join(uuid, uuid, member_role) FROM PUBLIC;

GRANT EXECUTE ON FUNCTION sso_provider_for(text) TO fajr_app;
GRANT EXECUTE ON FUNCTION sso_provider_by_id(uuid) TO fajr_app;
GRANT EXECUTE ON FUNCTION sso_login_start(text, uuid, text, text, text, int) TO fajr_app;
GRANT EXECUTE ON FUNCTION sso_login_take(text) TO fajr_app;
GRANT EXECUTE ON FUNCTION sso_user_for(uuid, text) TO fajr_app;
GRANT EXECUTE ON FUNCTION sso_link_user(uuid, text, uuid) TO fajr_app;
GRANT EXECUTE ON FUNCTION sso_join(uuid, uuid, member_role) TO fajr_app;

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP FUNCTION IF EXISTS sso_join(uuid, uuid, member_role);
DROP FUNCTION IF EXISTS sso_link_user(uuid, text, uuid);
DROP FUNCTION IF EXISTS sso_user_for(uuid, text);
DROP FUNCTION IF EXISTS sso_login_take(text);
DROP FUNCTION IF EXISTS sso_login_start(text, uuid, text, text, text, int);
DROP FUNCTION IF EXISTS sso_provider_by_id(uuid);
DROP FUNCTION IF EXISTS sso_provider_for(text);
DROP TABLE IF EXISTS sso_identities;
DROP TABLE IF EXISTS sso_logins;
DROP TABLE IF EXISTS identity_providers;
-- +goose StatementEnd
