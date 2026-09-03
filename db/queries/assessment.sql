-- name: CreateQuiz :one
INSERT INTO quizzes (tenant_id, lesson_id, title, instructions, dir, time_limit_s, max_attempts, pass_percent, shuffle, reveal_answers, draw_count)
VALUES (@tenant_id, @lesson_id, @title, @instructions, @dir, @time_limit_s, @max_attempts, @pass_percent, @shuffle, @reveal_answers, sqlc.narg('draw_count'))
RETURNING *;

-- name: GetQuiz :one
SELECT * FROM quizzes WHERE id = @id;

-- name: GetQuizByLesson :one
SELECT * FROM quizzes WHERE lesson_id = @lesson_id;

-- name: UpdateQuiz :one
UPDATE quizzes SET
  title          = coalesce(sqlc.narg('title'), title),
  instructions   = coalesce(sqlc.narg('instructions'), instructions),
  time_limit_s   = coalesce(sqlc.narg('time_limit_s'), time_limit_s),
  max_attempts   = coalesce(sqlc.narg('max_attempts'), max_attempts),
  pass_percent   = coalesce(sqlc.narg('pass_percent'), pass_percent),
  shuffle        = coalesce(sqlc.narg('shuffle'), shuffle),
  reveal_answers = coalesce(sqlc.narg('reveal_answers'), reveal_answers)
WHERE id = @id
RETURNING *;

-- name: CreateQuestion :one
INSERT INTO questions (tenant_id, quiz_id, kind, prompt, dir, points, explanation, position)
VALUES (@tenant_id, @quiz_id, @kind, @prompt, @dir, @points, @explanation,
        coalesce((SELECT max(position) + 1024 FROM questions WHERE quiz_id = @quiz_id), 1024))
RETURNING *;

-- name: ListQuestions :many
SELECT * FROM questions WHERE quiz_id = @quiz_id ORDER BY position, created_at;

-- name: GetQuestion :one
SELECT * FROM questions WHERE id = @id;

-- name: DeleteQuestion :execrows
DELETE FROM questions WHERE id = @id;

-- name: CreateOption :one
INSERT INTO question_options (tenant_id, question_id, label, is_correct, position)
VALUES (@tenant_id, @question_id, @label, @is_correct,
        coalesce((SELECT max(position) + 1024 FROM question_options WHERE question_id = @question_id), 1024))
RETURNING *;

-- name: ListOptionsForQuiz :many
SELECT o.* FROM question_options o JOIN questions q ON q.id = o.question_id
WHERE q.quiz_id = @quiz_id
ORDER BY q.position, o.position;

-- name: ReplaceOptions :exec
DELETE FROM question_options WHERE question_id = @question_id;

-- name: CountAttempts :one
SELECT count(*) FROM quiz_attempts WHERE quiz_id = @quiz_id AND user_id = @user_id;

-- name: OpenAttempt :one
SELECT * FROM quiz_attempts
WHERE quiz_id = @quiz_id AND user_id = @user_id AND state = 'in_progress';

-- name: StartAttempt :one
INSERT INTO quiz_attempts (tenant_id, quiz_id, enrollment_id, user_id, attempt_no, expires_at, points_possible)
VALUES (@tenant_id, @quiz_id, @enrollment_id, @user_id, @attempt_no,
        CASE WHEN @time_limit_s::integer > 0 THEN now() + make_interval(secs => @time_limit_s::integer) END,
        @points_possible)
RETURNING *;

-- name: GetAttempt :one
SELECT * FROM quiz_attempts WHERE id = @id;

-- name: ListMyAttempts :many
SELECT * FROM quiz_attempts WHERE quiz_id = @quiz_id AND user_id = @user_id ORDER BY attempt_no;

-- name: SaveAnswer :one
INSERT INTO attempt_answers (tenant_id, attempt_id, question_id, option_ids, text_answer, needs_grading)
VALUES (@tenant_id, @attempt_id, @question_id, @option_ids, @text_answer, @needs_grading)
ON CONFLICT (attempt_id, question_id) DO UPDATE SET
  option_ids    = excluded.option_ids,
  text_answer   = excluded.text_answer,
  needs_grading = excluded.needs_grading,
  updated_at    = now()
RETURNING *;

-- name: ListAnswers :many
SELECT * FROM attempt_answers WHERE attempt_id = @attempt_id;

-- name: GradeAnswer :exec
UPDATE attempt_answers SET points_awarded = @points_awarded, needs_grading = @needs_grading
WHERE attempt_id = @attempt_id AND question_id = @question_id;

-- name: MarkAnswer :one
UPDATE attempt_answers SET
  points_awarded = @points_awarded,
  feedback       = @feedback,
  graded_by      = @graded_by,
  needs_grading  = false
WHERE attempt_id = @attempt_id AND question_id = @question_id
RETURNING *;

-- name: ListAttemptsForMarking :many
SELECT sqlc.embed(a), u.full_name, q.title AS quiz_title, l.title AS lesson_title,
       count(ans.question_id) FILTER (WHERE ans.needs_grading) AS pending
FROM quiz_attempts a
JOIN users u ON u.id = a.user_id
JOIN quizzes q ON q.id = a.quiz_id
JOIN lessons l ON l.id = q.lesson_id
LEFT JOIN attempt_answers ans ON ans.attempt_id = a.id
WHERE a.state = 'submitted'
GROUP BY a.id, u.full_name, q.title, l.title
HAVING count(ans.question_id) FILTER (WHERE ans.needs_grading) > 0
ORDER BY a.submitted_at
LIMIT @page_limit OFFSET @page_offset;

-- Everything a marker needs for one attempt, answer key included.
-- name: AttemptSheet :many
SELECT sqlc.embed(q), ans.option_ids, ans.text_answer, ans.points_awarded,
       ans.needs_grading, ans.feedback
FROM questions q
LEFT JOIN attempt_answers ans ON ans.question_id = q.id AND ans.attempt_id = @attempt_id
WHERE q.quiz_id = @quiz_id
ORDER BY q.position;

-- name: SumAwardedPoints :one
SELECT coalesce(sum(points_awarded), 0)::integer AS total,
       count(*) FILTER (WHERE needs_grading) AS pending
FROM attempt_answers WHERE attempt_id = @attempt_id;

-- name: FinalizeAttempt :one
UPDATE quiz_attempts SET state = 'graded', graded_at = now(), points_awarded = @points_awarded
WHERE id = @id AND state = 'submitted'
RETURNING *;

-- name: FinishAttempt :one
UPDATE quiz_attempts SET
  state = @state,
  submitted_at = coalesce(submitted_at, now()),
  graded_at = CASE WHEN @state::attempt_state = 'graded' THEN now() ELSE graded_at END,
  points_awarded = @points_awarded
WHERE id = @id AND state = 'in_progress'
RETURNING *;

-- name: ExpireAttempt :one
UPDATE quiz_attempts SET state = 'expired', submitted_at = coalesce(submitted_at, now())
WHERE id = @id AND state = 'in_progress'
RETURNING *;

-- name: ServeQuestions :exec
INSERT INTO attempt_questions (attempt_id, question_id, tenant_id, position)
SELECT @attempt_id, unnest(@question_ids::uuid[]), @tenant_id, generate_subscripts(@question_ids::uuid[], 1);

-- name: ListServedQuestions :many
SELECT q.* FROM attempt_questions a
JOIN questions q ON q.id = a.question_id
WHERE a.attempt_id = @attempt_id
ORDER BY a.position;
