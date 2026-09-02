-- name: ProvisionTenant :one
SELECT * FROM provision_tenant(@slug, @name, @kind, @default_dir, @locale, @currency);

-- name: GetTenant :one
SELECT * FROM tenants WHERE id = $1;

-- name: ResolveTenant :one
SELECT * FROM resolve_tenant(@slug);
