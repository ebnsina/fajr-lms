-- name: CreateSitePage :one
INSERT INTO site_pages (tenant_id, slug, title, description, dir, blocks, nav_label, nav_order, updated_by)
VALUES (@tenant_id, @slug, @title, @description, @dir, @blocks, @nav_label, @nav_order, @updated_by)
RETURNING *;

-- name: GetSitePage :one
SELECT * FROM site_pages WHERE id = @id;

-- name: ListSitePages :many
SELECT * FROM site_pages ORDER BY nav_order, title;

-- name: UpdateSitePage :one
UPDATE site_pages SET
  title       = coalesce(sqlc.narg('title'), title),
  description = coalesce(sqlc.narg('description'), description),
  dir         = coalesce(sqlc.narg('dir'), dir),
  blocks      = coalesce(sqlc.narg('blocks'), blocks),
  nav_label   = coalesce(sqlc.narg('nav_label'), nav_label),
  nav_order   = coalesce(sqlc.narg('nav_order'), nav_order),
  updated_by  = @updated_by
WHERE id = @id
RETURNING *;

-- name: SetSitePageStatus :one
UPDATE site_pages SET status = @status, updated_by = @updated_by WHERE id = @id RETURNING *;

-- name: DeleteSitePage :execrows
DELETE FROM site_pages WHERE id = @id;

-- name: GetPublishedPage :one
SELECT * FROM published_pages WHERE tenant_slug = @tenant_slug AND slug = @slug;

-- name: ListSiteNav :many
SELECT slug, nav_label, nav_order FROM published_pages
WHERE tenant_slug = @tenant_slug AND nav_label <> ''
ORDER BY nav_order, nav_label;

-- name: ListPublishedCourses :many
SELECT * FROM published_courses
WHERE tenant_slug = @tenant_slug
ORDER BY published_at DESC NULLS LAST
LIMIT @page_limit;

-- name: SetSiteTheme :one
UPDATE tenants SET site_theme = @site_theme WHERE id = @id RETURNING *;
