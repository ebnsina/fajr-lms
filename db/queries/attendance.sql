-- name: CreateClassSession :one
INSERT INTO class_sessions (tenant_id, course_id, title, location, starts_at, ends_at, created_by)
VALUES (@tenant_id, @course_id, @title, @location, @starts_at, @ends_at, @created_by)
RETURNING *;

-- name: GetClassSession :one
SELECT * FROM class_sessions WHERE id = @id;

-- name: ListClassSessions :many
SELECT * FROM class_sessions WHERE course_id = @course_id
ORDER BY starts_at DESC
LIMIT @page_limit OFFSET @page_offset;

-- name: MarkAttendance :one
INSERT INTO attendance (tenant_id, session_id, enrollment_id, status, note, marked_by)
VALUES (@tenant_id, @session_id, @enrollment_id, @status, @note, @marked_by)
ON CONFLICT (session_id, enrollment_id) DO UPDATE
SET status = excluded.status, note = excluded.note, marked_by = excluded.marked_by, marked_at = now()
RETURNING *;

-- The roll for one session: every enrolled learner, marked or not.
-- name: SessionRoll :many
SELECT e.id AS enrollment_id, e.user_id, u.full_name, a.status, a.note, a.marked_at
FROM enrollments e
JOIN users u ON u.id = e.user_id
LEFT JOIN attendance a ON a.enrollment_id = e.id AND a.session_id = @session_id
WHERE e.course_id = @course_id AND e.status <> 'cancelled'
ORDER BY u.full_name;

-- name: AttendanceSummary :one
SELECT
  count(*) FILTER (WHERE a.status = 'present') AS present,
  count(*) FILTER (WHERE a.status = 'late')    AS late,
  count(*) FILTER (WHERE a.status = 'absent')  AS absent,
  count(*) FILTER (WHERE a.status = 'excused') AS excused
FROM attendance a JOIN class_sessions s ON s.id = a.session_id
WHERE s.course_id = @course_id AND a.enrollment_id = @enrollment_id;

-- name: MyAttendance :many
SELECT sqlc.embed(a), s.title, s.starts_at
FROM attendance a JOIN class_sessions s ON s.id = a.session_id
WHERE a.enrollment_id = @enrollment_id
ORDER BY s.starts_at DESC
LIMIT @page_limit OFFSET @page_offset;

-- name: AddGuardian :one
INSERT INTO guardianships (tenant_id, guardian_id, student_id, relation)
VALUES (@tenant_id, @guardian_id, @student_id, @relation)
ON CONFLICT (tenant_id, guardian_id, student_id) DO UPDATE SET relation = excluded.relation
RETURNING *;

-- name: RemoveGuardian :execrows
DELETE FROM guardianships WHERE guardian_id = @guardian_id AND student_id = @student_id;

-- name: GuardiansOf :many
SELECT guardian_id, relation FROM guardianships WHERE student_id = @student_id;

-- name: ListGuardianships :many
SELECT g.*, gu.full_name AS guardian_name, st.full_name AS student_name
FROM guardianships g
JOIN users gu ON gu.id = g.guardian_id
JOIN users st ON st.id = g.student_id
ORDER BY st.full_name;
