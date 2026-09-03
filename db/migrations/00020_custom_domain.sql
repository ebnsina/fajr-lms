-- +goose Up
-- +goose StatementBegin

-- A school can put its public site on its own domain. The token is what the
-- school publishes in DNS to prove the domain is theirs.
ALTER TABLE tenants
  ADD COLUMN custom_domain text UNIQUE
    CHECK (custom_domain IS NULL OR custom_domain ~ '^[a-z0-9]([a-z0-9-]*[a-z0-9])?(\.[a-z0-9]([a-z0-9-]*[a-z0-9])?)+$'),
  ADD COLUMN domain_token text NOT NULL DEFAULT '',
  ADD COLUMN domain_verified_at timestamptz;

DROP VIEW published_pages;
CREATE VIEW published_pages WITH (security_invoker = false) AS
  SELECT p.id, p.slug, p.title, p.description, p.dir, p.blocks, p.nav_label, p.nav_order,
         p.updated_at, t.slug AS tenant_slug, t.name AS tenant_name, t.default_dir AS tenant_dir,
         t.site_theme
  FROM site_pages p JOIN tenants t ON t.id = p.tenant_id
  WHERE p.status = 'published' AND t.status = 'active';

GRANT SELECT ON published_pages TO fajr_app;

-- Resolving a host to a school happens before any tenant scope exists.
CREATE FUNCTION resolve_domain(p_domain text) RETURNS SETOF tenants
LANGUAGE sql STABLE SECURITY DEFINER SET search_path = public AS $$
  SELECT * FROM tenants
  WHERE custom_domain = lower(p_domain) AND domain_verified_at IS NOT NULL AND status = 'active'
$$;

REVOKE ALL ON FUNCTION resolve_domain(text) FROM PUBLIC;
GRANT EXECUTE ON FUNCTION resolve_domain(text) TO fajr_app;

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP FUNCTION IF EXISTS resolve_domain(text);
ALTER TABLE tenants
  DROP COLUMN IF EXISTS custom_domain,
  DROP COLUMN IF EXISTS domain_token,
  DROP COLUMN IF EXISTS domain_verified_at;
-- +goose StatementEnd
