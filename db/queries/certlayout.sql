-- name: GetCertificateLayout :one
SELECT tenant_id, fields, background_type, (background IS NOT NULL)::boolean AS has_background, updated_at
FROM certificate_layouts WHERE tenant_id = @tenant_id;

-- name: SaveCertificateFields :one
INSERT INTO certificate_layouts (tenant_id, fields) VALUES (@tenant_id, @fields)
ON CONFLICT (tenant_id) DO UPDATE SET fields = excluded.fields
RETURNING tenant_id, fields, background_type, (background IS NOT NULL)::boolean AS has_background, updated_at;

-- name: SaveCertificateBackground :exec
INSERT INTO certificate_layouts (tenant_id, background, background_type)
VALUES (@tenant_id, @background, @background_type)
ON CONFLICT (tenant_id) DO UPDATE
SET background = excluded.background, background_type = excluded.background_type;

-- name: ClearCertificateBackground :exec
UPDATE certificate_layouts SET background = NULL, background_type = '' WHERE tenant_id = @tenant_id;

-- name: PublicCertificateLayout :one
SELECT fields, background_type, (background IS NOT NULL)::boolean AS has_background
FROM certificate_layout_for(@tenant_id);

-- name: CertificateBackground :one
SELECT background, background_type FROM certificate_layout_for(@tenant_id);

-- name: CertificateTenant :one
SELECT * FROM certificate_tenant(@serial);

-- name: OwnCertificateBackground :one
SELECT background, background_type FROM certificate_layouts WHERE tenant_id = @tenant_id;
