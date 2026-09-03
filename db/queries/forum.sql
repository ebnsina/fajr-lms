-- name: CreateThread :one
INSERT INTO forum_threads (tenant_id, course_id, title, dir, author_id)
VALUES (@tenant_id, @course_id, @title, @dir, @author_id)
RETURNING *;

-- name: ListThreads :many
SELECT sqlc.embed(t), u.full_name AS author_name
FROM forum_threads t LEFT JOIN users u ON u.id = t.author_id
WHERE t.course_id = @course_id
ORDER BY t.pinned DESC, t.last_post_at DESC
LIMIT @page_limit OFFSET @page_offset;

-- name: GetThread :one
SELECT sqlc.embed(t), u.full_name AS author_name, c.slug AS course_slug, c.title AS course_title
FROM forum_threads t
JOIN courses c ON c.id = t.course_id
LEFT JOIN users u ON u.id = t.author_id
WHERE t.id = @id;

-- name: SetThreadFlags :one
UPDATE forum_threads SET
  pinned = coalesce(sqlc.narg('pinned'), pinned),
  locked = coalesce(sqlc.narg('locked'), locked)
WHERE id = @id
RETURNING *;

-- name: DeleteThread :execrows
DELETE FROM forum_threads WHERE id = @id;

-- name: AddPost :one
INSERT INTO forum_posts (tenant_id, thread_id, author_id, body, dir)
VALUES (@tenant_id, @thread_id, @author_id, @body, @dir)
RETURNING *;

-- name: CountReply :exec
-- The thread carries its own count and last post, so a list of threads is one
-- query rather than one per row.
UPDATE forum_threads SET reply_count = reply_count + 1, last_post_at = now() WHERE id = @id;

-- name: ListPosts :many
SELECT sqlc.embed(p), u.full_name AS author_name
FROM forum_posts p LEFT JOIN users u ON u.id = p.author_id
WHERE p.thread_id = @thread_id
ORDER BY p.created_at;

-- name: GetPost :one
SELECT * FROM forum_posts WHERE id = @id;

-- name: EditPost :one
UPDATE forum_posts SET body = @body WHERE id = @id AND removed_at IS NULL RETURNING *;

-- name: RemovePost :one
UPDATE forum_posts SET removed_at = now(), removed_by = @removed_by
WHERE id = @id AND removed_at IS NULL
RETURNING *;
