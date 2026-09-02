-- +goose Up
-- +goose StatementBegin

-- A school's site is dressed for where it teaches: the Gulf reads right to
-- left and sets Arabic larger, Bengal sets Bengali and runs denser.
ALTER TABLE tenants ADD COLUMN site_theme text NOT NULL DEFAULT 'plain'
  CHECK (site_theme IN ('plain', 'gulf', 'bengal'));

DROP VIEW published_pages;
CREATE VIEW published_pages WITH (security_invoker = false) AS
  SELECT p.id, p.slug, p.title, p.description, p.dir, p.blocks, p.nav_label, p.nav_order,
         p.updated_at, t.slug AS tenant_slug, t.name AS tenant_name, t.default_dir AS tenant_dir,
         t.site_theme
  FROM site_pages p JOIN tenants t ON t.id = p.tenant_id
  WHERE p.status = 'published' AND t.status = 'active';

GRANT SELECT ON published_pages TO fajr_app;

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP VIEW IF EXISTS published_pages;
ALTER TABLE tenants DROP COLUMN IF EXISTS site_theme;
-- +goose StatementEnd
