-- name: CreateTopic :one
INSERT INTO topics (tenant_id, name, slug) VALUES (@tenant_id, @name, @slug) RETURNING *;

-- name: ListTopics :many
SELECT t.*, (SELECT count(*) FROM course_topics ct WHERE ct.topic_id = t.id) AS courses
FROM topics t ORDER BY t.name;

-- name: DeleteTopic :execrows
DELETE FROM topics WHERE id = @id;

-- name: TagCourse :exec
INSERT INTO course_topics (course_id, topic_id, tenant_id)
VALUES (@course_id, @topic_id, @tenant_id)
ON CONFLICT DO NOTHING;

-- name: UntagCourse :execrows
DELETE FROM course_topics WHERE course_id = @course_id AND topic_id = @topic_id;

-- name: TopicsOfCourse :many
SELECT t.* FROM course_topics ct JOIN topics t ON t.id = ct.topic_id
WHERE ct.course_id = @course_id ORDER BY t.name;

-- name: CreateCollection :one
INSERT INTO collections (tenant_id, kind, slug, title, summary, dir, price_minor, currency)
VALUES (@tenant_id, @kind, @slug, @title, @summary, @dir, @price_minor, @currency)
RETURNING *;

-- name: ListCollections :many
SELECT c.*, (SELECT count(*) FROM collection_courses cc WHERE cc.collection_id = c.id) AS courses
FROM collections c
WHERE (sqlc.narg('kind')::collection_kind IS NULL OR c.kind = sqlc.narg('kind')::collection_kind)
ORDER BY c.title;

-- name: GetCollection :one
SELECT * FROM collections WHERE slug = @slug;

-- name: UpdateCollection :one
UPDATE collections SET
  title       = coalesce(sqlc.narg('title'), title),
  summary     = coalesce(sqlc.narg('summary'), summary),
  status      = coalesce(sqlc.narg('status'), status),
  price_minor = coalesce(sqlc.narg('price_minor'), price_minor)
WHERE id = @id
RETURNING *;

-- name: DeleteCollection :execrows
DELETE FROM collections WHERE id = @id;

-- name: AddCourseToCollection :exec
INSERT INTO collection_courses (collection_id, course_id, tenant_id, position)
VALUES (@collection_id, @course_id, @tenant_id,
        coalesce((SELECT max(position) + 1 FROM collection_courses WHERE collection_id = @collection_id), 1))
ON CONFLICT DO NOTHING;

-- name: RemoveCourseFromCollection :execrows
DELETE FROM collection_courses WHERE collection_id = @collection_id AND course_id = @course_id;

-- name: CollectionCourses :many
SELECT sqlc.embed(c), cc.position
FROM collection_courses cc JOIN courses c ON c.id = cc.course_id
WHERE cc.collection_id = @collection_id
ORDER BY cc.position;

-- name: MoveCourseInCollection :execrows
UPDATE collection_courses SET position = @position
WHERE collection_id = @collection_id AND course_id = @course_id;
