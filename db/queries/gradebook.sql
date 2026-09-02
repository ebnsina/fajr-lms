-- name: CreateGradeItem :one
INSERT INTO grade_items (tenant_id, course_id, quiz_id, source, title, category, points_possible, weight, position)
VALUES (@tenant_id, @course_id, @quiz_id, @source, @title, @category, @points_possible, @weight,
        coalesce((SELECT max(position) + 1024 FROM grade_items WHERE course_id = @course_id), 1024))
RETURNING *;

-- name: GetGradeItem :one
SELECT * FROM grade_items WHERE id = @id;

-- name: ListGradeItems :many
SELECT * FROM grade_items WHERE course_id = @course_id ORDER BY position, created_at;

-- name: UpdateGradeItem :one
UPDATE grade_items SET
  title    = coalesce(sqlc.narg('title'), title),
  category = coalesce(sqlc.narg('category'), category),
  weight   = coalesce(sqlc.narg('weight'), weight)
WHERE id = @id
RETURNING *;

-- name: DeleteGradeItem :execrows
DELETE FROM grade_items WHERE id = @id AND source = 'manual';

-- name: SetGradeOverride :one
INSERT INTO grade_overrides (tenant_id, grade_item_id, enrollment_id, points, note, set_by)
VALUES (@tenant_id, @grade_item_id, @enrollment_id, @points, @note, @set_by)
ON CONFLICT (grade_item_id, enrollment_id) DO UPDATE
SET points = excluded.points, note = excluded.note, set_by = excluded.set_by, updated_at = now()
RETURNING *;

-- name: ClearGradeOverride :execrows
DELETE FROM grade_overrides WHERE grade_item_id = @grade_item_id AND enrollment_id = @enrollment_id;

-- The best graded attempt per learner per quiz, which is what a grade uses.
-- name: BestQuizScores :many
SELECT DISTINCT ON (a.quiz_id, a.enrollment_id)
  a.quiz_id, a.enrollment_id, a.points_awarded, a.points_possible, a.graded_at
FROM quiz_attempts a
JOIN quizzes q ON q.id = a.quiz_id
JOIN lessons l ON l.id = q.lesson_id
JOIN modules m ON m.id = l.module_id
WHERE m.course_id = @course_id AND a.state = 'graded'
ORDER BY a.quiz_id, a.enrollment_id, a.points_awarded DESC, a.graded_at;

-- name: ListCourseOverrides :many
SELECT o.* FROM grade_overrides o JOIN grade_items i ON i.id = o.grade_item_id
WHERE i.course_id = @course_id;

-- name: ListCourseEnrollments :many
SELECT e.id, e.user_id, u.full_name
FROM enrollments e JOIN users u ON u.id = e.user_id
WHERE e.course_id = @course_id AND e.status <> 'cancelled'
ORDER BY u.full_name;

-- A quiz item is worth whatever its questions add up to.
-- name: SyncQuizItemPoints :exec
UPDATE grade_items i
SET points_possible = GREATEST(1, (SELECT coalesce(sum(q.points), 0)::integer FROM questions q WHERE q.quiz_id = @target_quiz))
WHERE i.quiz_id = @target_quiz;
