-- +goose Up
-- +goose StatementBegin
-- The client needs a tenant's slug before it can scope any request to it, and
-- that lookup happens before a scope exists.
CREATE VIEW user_tenants WITH (security_invoker = false) AS
  SELECT m.user_id, m.role, t.id, t.slug, t.name, t.kind, t.status, t.default_dir, t.locale, t.currency
  FROM memberships m JOIN tenants t ON t.id = m.tenant_id
  WHERE m.status = 'active' AND t.status = 'active';

GRANT SELECT ON user_tenants TO fajr_app;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP VIEW IF EXISTS user_tenants;
-- +goose StatementEnd
