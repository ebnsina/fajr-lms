-- +goose Up
-- +goose StatementBegin

-- A tenant's public website: pages built from a list of blocks. Navigation is
-- derived from the pages themselves rather than kept as a separate menu.
CREATE TABLE site_pages (
  id          uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  tenant_id   uuid NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
  slug        text NOT NULL CHECK (slug = '' OR slug ~ '^[a-z0-9][a-z0-9-]{0,62}$'),
  title       text NOT NULL COLLATE human_name CHECK (length(btrim(title)) BETWEEN 1 AND 200),
  description text NOT NULL DEFAULT '' CHECK (length(description) <= 300),
  dir         text_dir NOT NULL DEFAULT 'auto',
  blocks      jsonb NOT NULL DEFAULT '[]' CHECK (jsonb_typeof(blocks) = 'array'),
  status      publish_status NOT NULL DEFAULT 'draft',
  nav_label   text NOT NULL DEFAULT '' CHECK (length(nav_label) <= 40),
  nav_order   int NOT NULL DEFAULT 0,
  updated_by  uuid REFERENCES users(id) ON DELETE SET NULL,
  created_at  timestamptz NOT NULL DEFAULT now(),
  updated_at  timestamptz NOT NULL DEFAULT now(),
  UNIQUE (tenant_id, slug)
);

CREATE INDEX site_pages_nav_idx ON site_pages (tenant_id, nav_order) WHERE nav_label <> '';

CREATE TRIGGER site_pages_touch BEFORE UPDATE ON site_pages FOR EACH ROW EXECUTE FUNCTION touch_updated_at();

ALTER TABLE site_pages ENABLE ROW LEVEL SECURITY;
ALTER TABLE site_pages FORCE ROW LEVEL SECURITY;
CREATE POLICY tenant_isolation ON site_pages USING (tenant_id = current_tenant_id()) WITH CHECK (tenant_id = current_tenant_id());
GRANT SELECT, INSERT, UPDATE, DELETE ON site_pages TO fajr_app;

-- A published page is public by definition, so it is readable without a tenant
-- scope. Drafts never appear here.
CREATE VIEW published_pages WITH (security_invoker = false) AS
  SELECT p.id, p.slug, p.title, p.description, p.dir, p.blocks, p.nav_label, p.nav_order,
         p.updated_at, t.slug AS tenant_slug, t.name AS tenant_name, t.default_dir AS tenant_dir
  FROM site_pages p JOIN tenants t ON t.id = p.tenant_id
  WHERE p.status = 'published' AND t.status = 'active';

GRANT SELECT ON published_pages TO fajr_app;

-- The catalogue a site may advertise: published, publicly visible courses only.
CREATE VIEW published_courses WITH (security_invoker = false) AS
  SELECT c.id, c.slug, c.title, c.summary, c.dir, c.price_minor, c.currency,
         c.published_at, t.slug AS tenant_slug
  FROM courses c JOIN tenants t ON t.id = c.tenant_id
  WHERE c.status = 'published' AND c.visibility = 'public' AND t.status = 'active';

GRANT SELECT ON published_courses TO fajr_app;

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP VIEW IF EXISTS published_courses;
DROP VIEW IF EXISTS published_pages;
DROP TABLE IF EXISTS site_pages;
-- +goose StatementEnd
