-- +goose Up
-- +goose StatementBegin

CREATE EXTENSION IF NOT EXISTS pgcrypto;
CREATE EXTENSION IF NOT EXISTS pg_trgm;

-- Case-and-accent-insensitive collation for names across Latin, Arabic and Bengali.
CREATE COLLATION IF NOT EXISTS human_name (provider = icu, locale = 'und-u-ks-level2', deterministic = false);

-- Returns the tenant scoping the current transaction, or NULL so policies fail closed.
CREATE FUNCTION current_tenant_id() RETURNS uuid
LANGUAGE sql STABLE PARALLEL SAFE AS $$
  SELECT nullif(current_setting('app.tenant_id', true), '')::uuid
$$;

CREATE TYPE tenant_kind AS ENUM ('institution', 'creator', 'corporate');
CREATE TYPE tenant_status AS ENUM ('active', 'suspended');
CREATE TYPE member_role AS ENUM ('owner', 'admin', 'instructor', 'assistant', 'student', 'parent');
CREATE TYPE member_status AS ENUM ('invited', 'active', 'suspended');
CREATE TYPE text_dir AS ENUM ('auto', 'ltr', 'rtl');

CREATE TABLE tenants (
  id            uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  slug          text NOT NULL UNIQUE CHECK (slug ~ '^[a-z0-9][a-z0-9-]{1,62}$'),
  name          text NOT NULL CHECK (length(btrim(name)) BETWEEN 1 AND 200),
  kind          tenant_kind NOT NULL,
  status        tenant_status NOT NULL DEFAULT 'active',
  default_dir   text_dir NOT NULL DEFAULT 'auto',
  locale        text NOT NULL DEFAULT 'en' CHECK (locale ~ '^[a-z]{2}(-[A-Za-z0-9]{2,8})*$'),
  currency      char(3) NOT NULL DEFAULT 'BDT' CHECK (currency ~ '^[A-Z]{3}$'),
  created_at    timestamptz NOT NULL DEFAULT now(),
  updated_at    timestamptz NOT NULL DEFAULT now()
);

-- Users are global: one person may belong to several tenants with different roles.
CREATE TABLE users (
  id            uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  phone         text UNIQUE CHECK (phone ~ '^\+[1-9][0-9]{6,14}$'),
  email         text UNIQUE CHECK (email = lower(email) AND email LIKE '%_@_%.__%'),
  full_name     text NOT NULL COLLATE human_name CHECK (length(btrim(full_name)) BETWEEN 1 AND 200),
  password_hash text,
  created_at    timestamptz NOT NULL DEFAULT now(),
  updated_at    timestamptz NOT NULL DEFAULT now(),
  CONSTRAINT users_need_an_identifier CHECK (phone IS NOT NULL OR email IS NOT NULL)
);

CREATE TABLE memberships (
  id         uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  tenant_id  uuid NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
  user_id    uuid NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  role       member_role NOT NULL,
  status     member_status NOT NULL DEFAULT 'active',
  created_at timestamptz NOT NULL DEFAULT now(),
  updated_at timestamptz NOT NULL DEFAULT now(),
  UNIQUE (tenant_id, user_id, role)
);

CREATE INDEX memberships_tenant_idx ON memberships (tenant_id);
CREATE INDEX memberships_user_idx ON memberships (user_id);
CREATE INDEX users_name_trgm_idx ON users USING gin (full_name gin_trgm_ops);

-- Tenants: a tenant may be created unscoped, but never enumerated.
ALTER TABLE tenants ENABLE ROW LEVEL SECURITY;
ALTER TABLE tenants FORCE ROW LEVEL SECURITY;
CREATE POLICY tenant_self_read ON tenants FOR SELECT USING (id = current_tenant_id());
CREATE POLICY tenant_self_write ON tenants FOR UPDATE
  USING (id = current_tenant_id()) WITH CHECK (id = current_tenant_id());

-- Users are global, so visibility is derived from a shared membership.
ALTER TABLE users ENABLE ROW LEVEL SECURITY;
ALTER TABLE users FORCE ROW LEVEL SECURITY;
CREATE POLICY user_shares_tenant ON users FOR SELECT USING (
  EXISTS (SELECT 1 FROM memberships m WHERE m.user_id = users.id AND m.tenant_id = current_tenant_id())
);
CREATE POLICY user_update_in_tenant ON users FOR UPDATE USING (
  EXISTS (SELECT 1 FROM memberships m WHERE m.user_id = users.id AND m.tenant_id = current_tenant_id())
);

ALTER TABLE memberships ENABLE ROW LEVEL SECURITY;
ALTER TABLE memberships FORCE ROW LEVEL SECURITY;
CREATE POLICY tenant_isolation ON memberships
  USING (tenant_id = current_tenant_id()) WITH CHECK (tenant_id = current_tenant_id());

-- Provisioning happens before a tenant scope exists, so it runs through one
-- audited function rather than an INSERT policy the app role could misuse.
CREATE FUNCTION provision_tenant(
  p_slug text, p_name text, p_kind tenant_kind,
  p_default_dir text_dir DEFAULT 'auto', p_locale text DEFAULT 'en', p_currency char(3) DEFAULT 'BDT'
) RETURNS tenants
LANGUAGE sql VOLATILE SECURITY DEFINER SET search_path = public AS $$
  INSERT INTO tenants (slug, name, kind, default_dir, locale, currency)
  VALUES (p_slug, p_name, p_kind, p_default_dir, p_locale, p_currency)
  RETURNING *
$$;

CREATE FUNCTION signup_user(p_phone text, p_email text, p_full_name text, p_password_hash text)
RETURNS users
LANGUAGE sql VOLATILE SECURITY DEFINER SET search_path = public AS $$
  INSERT INTO users (phone, email, full_name, password_hash)
  VALUES (nullif(p_phone, ''), nullif(lower(p_email), ''), p_full_name, nullif(p_password_hash, ''))
  RETURNING *
$$;

-- Pre-auth lookups run before a tenant is known, so they are narrow and audited.
CREATE FUNCTION resolve_tenant(p_slug text) RETURNS SETOF tenants
LANGUAGE sql STABLE SECURITY DEFINER SET search_path = public AS $$
  SELECT * FROM tenants WHERE slug = p_slug
$$;

CREATE FUNCTION auth_find_user(p_phone text, p_email text) RETURNS SETOF users
LANGUAGE sql STABLE SECURITY DEFINER SET search_path = public AS $$
  SELECT * FROM users
  WHERE (nullif(p_phone, '') IS NOT NULL AND phone = p_phone)
     OR (nullif(p_email, '') IS NOT NULL AND email = lower(p_email))
  LIMIT 1
$$;

CREATE FUNCTION auth_memberships(p_user_id uuid) RETURNS SETOF memberships
LANGUAGE sql STABLE SECURITY DEFINER SET search_path = public AS $$
  SELECT m.* FROM memberships m JOIN tenants t ON t.id = m.tenant_id
  WHERE m.user_id = p_user_id AND m.status = 'active' AND t.status = 'active'
$$;

GRANT SELECT, UPDATE ON tenants, users TO fajr_app;
GRANT SELECT, INSERT, UPDATE, DELETE ON memberships TO fajr_app;
REVOKE ALL ON FUNCTION provision_tenant(text, text, tenant_kind, text_dir, text, char),
  signup_user(text, text, text, text), resolve_tenant(text),
  auth_find_user(text, text), auth_memberships(uuid) FROM PUBLIC;
GRANT EXECUTE ON FUNCTION provision_tenant(text, text, tenant_kind, text_dir, text, char),
  signup_user(text, text, text, text), resolve_tenant(text),
  auth_find_user(text, text), auth_memberships(uuid) TO fajr_app;

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS memberships;
DROP TABLE IF EXISTS users;
DROP TABLE IF EXISTS tenants;
DROP FUNCTION IF EXISTS signup_user(text, text, text, text);
DROP FUNCTION IF EXISTS provision_tenant(text, text, tenant_kind, text_dir, text, char);
DROP FUNCTION IF EXISTS auth_memberships(uuid);
DROP FUNCTION IF EXISTS auth_find_user(text, text);
DROP FUNCTION IF EXISTS resolve_tenant(text);
DROP FUNCTION IF EXISTS current_tenant_id();
DROP TYPE IF EXISTS text_dir, member_status, member_role, tenant_status, tenant_kind;
DROP COLLATION IF EXISTS human_name;
-- +goose StatementEnd
