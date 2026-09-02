-- name: CreateMediaAsset :one
INSERT INTO media_assets (tenant_id, provider, external_ref, state, kind, title, duration_s, content_type, metadata, created_by)
VALUES (@tenant_id, @provider, @external_ref, @state, @kind, @title, @duration_s, @content_type, @metadata, @created_by)
RETURNING *;

-- name: GetMediaAsset :one
SELECT * FROM media_assets WHERE id = @id;

-- name: UpdateMediaState :one
UPDATE media_assets SET
  state        = @state,
  duration_s   = coalesce(sqlc.narg('duration_s'), duration_s),
  byte_size    = coalesce(sqlc.narg('byte_size'), byte_size),
  content_type = coalesce(sqlc.narg('content_type'), content_type),
  error        = coalesce(sqlc.narg('error'), error),
  metadata     = coalesce(sqlc.narg('metadata'), metadata)
WHERE id = @id
RETURNING *;

-- name: AttachMediaToLesson :one
UPDATE lessons SET media_id = @media_id WHERE id = @id RETURNING *;

-- name: RecordMediaDelivery :exec
INSERT INTO media_delivery (tenant_id, day, requests, bytes)
VALUES (@tenant_id, current_date, 1, @bytes)
ON CONFLICT (tenant_id, day) DO UPDATE
SET requests = media_delivery.requests + 1, bytes = media_delivery.bytes + excluded.bytes;

-- name: MediaUsage :many
SELECT * FROM media_delivery WHERE day >= current_date - @days::integer ORDER BY day DESC;
