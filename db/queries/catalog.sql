-- name: CreateCourse :one
INSERT INTO courses (tenant_id, slug, title, summary, dir, visibility, price_minor, currency, created_by)
VALUES (@tenant_id, @slug, @title, @summary, @dir, @visibility, @price_minor, @currency, @created_by)
RETURNING *;

-- name: GetCourse :one
SELECT * FROM courses WHERE id = @id;

-- name: GetCourseBySlug :one
SELECT * FROM courses WHERE slug = @slug;

-- name: ListCourses :many
SELECT * FROM courses
WHERE (sqlc.narg('status_filter')::publish_status IS NULL OR status = sqlc.narg('status_filter'))
ORDER BY created_at DESC
LIMIT @page_limit OFFSET @page_offset;

-- name: CountCourses :one
SELECT count(*) FROM courses
WHERE (sqlc.narg('status_filter')::publish_status IS NULL OR status = sqlc.narg('status_filter'));

-- name: UpdateCourse :one
UPDATE courses SET
  title       = coalesce(sqlc.narg('title'), title),
  summary     = coalesce(sqlc.narg('summary'), summary),
  dir         = coalesce(sqlc.narg('dir'), dir),
  visibility  = coalesce(sqlc.narg('visibility'), visibility),
  price_minor = coalesce(sqlc.narg('price_minor'), price_minor)
WHERE id = @id
RETURNING *;

-- name: SetCourseStatus :one
UPDATE courses SET
  status = @status,
  published_at = CASE WHEN @status::publish_status = 'published' THEN coalesce(published_at, now()) ELSE published_at END
WHERE id = @id
RETURNING *;

-- name: DeleteCourse :execrows
DELETE FROM courses WHERE id = @id;

-- name: CreateModule :one
INSERT INTO modules (tenant_id, course_id, title, position)
VALUES (@tenant_id, @course_id, @title, coalesce((SELECT max(position) + 1024 FROM modules WHERE course_id = @course_id), 1024))
RETURNING *;

-- name: ListModules :many
SELECT * FROM modules WHERE course_id = @course_id ORDER BY position, created_at;

-- name: MoveModule :one
UPDATE modules SET position = @position WHERE id = @id RETURNING *;

-- name: DeleteModule :execrows
DELETE FROM modules WHERE id = @id;

-- name: CreateLesson :one
INSERT INTO lessons (tenant_id, module_id, title, kind, body, dir, duration_s, is_preview, position)
VALUES (@tenant_id, @module_id, @title, @kind, @body, @dir, @duration_s, @is_preview,
        coalesce((SELECT max(position) + 1024 FROM lessons WHERE module_id = @module_id), 1024))
RETURNING *;

-- name: ListLessonsForCourse :many
SELECT l.* FROM lessons l JOIN modules m ON m.id = l.module_id
WHERE m.course_id = @course_id
ORDER BY m.position, l.position, l.created_at;

-- name: GetLesson :one
SELECT * FROM lessons WHERE id = @id;

-- name: UpdateLesson :one
UPDATE lessons SET
  title      = coalesce(sqlc.narg('title'), title),
  body       = coalesce(sqlc.narg('body'), body),
  kind       = coalesce(sqlc.narg('kind'), kind),
  dir        = coalesce(sqlc.narg('dir'), dir),
  duration_s = coalesce(sqlc.narg('duration_s'), duration_s),
  is_preview = coalesce(sqlc.narg('is_preview'), is_preview),
  status     = coalesce(sqlc.narg('status'), status)
WHERE id = @id
RETURNING *;

-- name: MoveLesson :one
UPDATE lessons SET module_id = @module_id, position = @position WHERE id = @id RETURNING *;

-- name: DeleteLesson :execrows
DELETE FROM lessons WHERE id = @id;

