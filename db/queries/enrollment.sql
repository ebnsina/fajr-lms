-- name: EnrollUser :one
INSERT INTO enrollments (tenant_id, course_id, user_id, source)
VALUES (@tenant_id, @course_id, @user_id, @source)
ON CONFLICT (course_id, user_id) DO UPDATE
SET status = 'active', completed_at = NULL
RETURNING *;

-- name: GetEnrollment :one
SELECT * FROM enrollments WHERE course_id = @course_id AND user_id = @user_id;

-- name: CancelEnrollment :execrows
UPDATE enrollments SET status = 'cancelled' WHERE id = @id AND status <> 'cancelled';

-- name: ListMyEnrollments :many
SELECT sqlc.embed(e), c.slug, c.title, c.dir, c.status AS course_status
FROM enrollments e JOIN courses c ON c.id = e.course_id
WHERE e.user_id = @user_id AND e.status <> 'cancelled'
ORDER BY e.created_at DESC
LIMIT @page_limit OFFSET @page_offset;

-- Roster with a completion count, so the UI never asks per learner.
-- name: ListCourseRoster :many
SELECT
  sqlc.embed(e),
  u.full_name,
  count(p.lesson_id) FILTER (WHERE p.state = 'completed') AS lessons_completed
FROM enrollments e
JOIN users u ON u.id = e.user_id
LEFT JOIN lesson_progress p ON p.enrollment_id = e.id
WHERE e.course_id = @course_id AND e.status <> 'cancelled'
GROUP BY e.id, u.full_name
ORDER BY u.full_name
LIMIT @page_limit OFFSET @page_offset;

-- name: CountPublishedLessons :one
SELECT count(*) FROM lessons l
JOIN modules m ON m.id = l.module_id
WHERE m.course_id = @course_id AND l.status = 'published';

-- Merge rules for offline sync: never rewind a resume point, never un-complete.
-- name: RecordProgress :one
INSERT INTO lesson_progress (tenant_id, enrollment_id, lesson_id, state, position_s, completed_at)
VALUES (@tenant_id, @enrollment_id, @lesson_id,
        CASE WHEN @completed::boolean THEN 'completed'::progress_state ELSE 'in_progress'::progress_state END,
        @position_s,
        CASE WHEN @completed::boolean THEN now() ELSE NULL END)
ON CONFLICT (enrollment_id, lesson_id) DO UPDATE SET
  position_s   = GREATEST(lesson_progress.position_s, excluded.position_s),
  state        = CASE WHEN lesson_progress.state = 'completed' THEN 'completed'::progress_state ELSE excluded.state END,
  completed_at = coalesce(lesson_progress.completed_at, excluded.completed_at),
  updated_at   = now()
RETURNING *;

-- name: ListProgressForEnrollment :many
SELECT * FROM lesson_progress WHERE enrollment_id = @enrollment_id;

-- name: CountCompletedLessons :one
SELECT count(*) FROM lesson_progress WHERE enrollment_id = @enrollment_id AND state = 'completed';

-- name: CompleteEnrollment :one
UPDATE enrollments SET status = 'completed', completed_at = coalesce(completed_at, now())
WHERE id = @id
RETURNING *;

-- name: LessonCourse :one
SELECT m.course_id FROM lessons l JOIN modules m ON m.id = l.module_id WHERE l.id = @lesson_id;

-- name: GetEnrollmentByID :one
SELECT * FROM enrollments WHERE id = @id;
