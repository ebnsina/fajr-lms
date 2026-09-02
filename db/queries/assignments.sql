-- name: CreateAssignment :one
INSERT INTO assignments (tenant_id, lesson_id, title, instructions, dir, points, due_at, allow_late, late_penalty, max_files)
VALUES (@tenant_id, @lesson_id, @title, @instructions, @dir, @points, @due_at, @allow_late, @late_penalty, @max_files)
RETURNING *;

-- name: GetAssignment :one
SELECT * FROM assignments WHERE id = @id;

-- name: GetAssignmentByLesson :one
SELECT * FROM assignments WHERE lesson_id = @lesson_id;

-- name: UpdateAssignment :one
UPDATE assignments SET
  title        = coalesce(sqlc.narg('title'), title),
  instructions = coalesce(sqlc.narg('instructions'), instructions),
  due_at       = coalesce(sqlc.narg('due_at'), due_at),
  allow_late   = coalesce(sqlc.narg('allow_late'), allow_late),
  late_penalty = coalesce(sqlc.narg('late_penalty'), late_penalty)
WHERE id = @id
RETURNING *;

-- name: UpsertSubmission :one
INSERT INTO submissions (tenant_id, assignment_id, enrollment_id, user_id, state, body, media_ids, is_late, submitted_at)
VALUES (@tenant_id, @assignment_id, @enrollment_id, @user_id, @state, @body, @media_ids, @is_late,
        CASE WHEN @state::submission_state <> 'draft' THEN now() END)
ON CONFLICT (assignment_id, enrollment_id) DO UPDATE SET
  state        = excluded.state,
  body         = excluded.body,
  media_ids    = excluded.media_ids,
  is_late      = excluded.is_late,
  submitted_at = CASE WHEN excluded.state <> 'draft' THEN coalesce(submissions.submitted_at, now()) END,
  updated_at   = now()
RETURNING *;

-- name: GetSubmission :one
SELECT * FROM submissions WHERE id = @id;

-- name: MySubmission :one
SELECT * FROM submissions WHERE assignment_id = @assignment_id AND user_id = @user_id;

-- name: ListSubmissions :many
SELECT sqlc.embed(s), u.full_name
FROM submissions s JOIN users u ON u.id = s.user_id
WHERE s.assignment_id = @assignment_id AND s.state <> 'draft'
ORDER BY s.submitted_at
LIMIT @page_limit OFFSET @page_offset;

-- name: ListSubmissionsToGrade :many
SELECT sqlc.embed(s), u.full_name, a.title AS assignment_title, a.points, a.due_at
FROM submissions s
JOIN users u ON u.id = s.user_id
JOIN assignments a ON a.id = s.assignment_id
WHERE s.state = 'submitted'
ORDER BY s.submitted_at
LIMIT @page_limit OFFSET @page_offset;

-- name: GradeSubmission :one
UPDATE submissions SET
  state = 'returned', points_awarded = @points_awarded, feedback = @feedback,
  graded_by = @graded_by, graded_at = now()
WHERE id = @id AND state = 'submitted'
RETURNING *;

-- name: BestAssignmentScores :many
SELECT a.id AS assignment_id, s.enrollment_id, s.points_awarded
FROM submissions s JOIN assignments a ON a.id = s.assignment_id
JOIN lessons l ON l.id = a.lesson_id JOIN modules m ON m.id = l.module_id
WHERE m.course_id = @course_id AND s.state = 'returned' AND s.points_awarded IS NOT NULL;

-- name: SubmissionForMarking :one
SELECT sqlc.embed(s), sqlc.embed(a), u.full_name
FROM submissions s
JOIN assignments a ON a.id = s.assignment_id
JOIN users u ON u.id = s.user_id
WHERE s.id = @id;
