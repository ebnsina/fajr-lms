-- name: CreateAcademicYear :one
INSERT INTO academic_years (tenant_id, name, starts_on, ends_on)
VALUES (@tenant_id, @name, @starts_on, @ends_on)
RETURNING *;

-- name: ListAcademicYears :many
SELECT * FROM academic_years ORDER BY starts_on DESC;

-- name: CurrentYear :one
SELECT * FROM academic_years WHERE is_current;

-- name: ClearCurrentYear :exec
-- Cleared first and set second, in the same transaction: one statement cannot
-- both drop and take the only-current index without colliding with itself.
UPDATE academic_years SET is_current = false WHERE is_current;

-- name: MakeYearCurrent :one
UPDATE academic_years SET is_current = true WHERE id = @id RETURNING *;

-- name: DeleteAcademicYear :execrows
DELETE FROM academic_years WHERE id = @id;

-- name: CreateTerm :one
INSERT INTO terms (tenant_id, year_id, name, starts_on, ends_on)
VALUES (@tenant_id, @year_id, @name, @starts_on, @ends_on)
RETURNING *;

-- name: ListTerms :many
SELECT * FROM terms WHERE year_id = @year_id ORDER BY starts_on;

-- name: CurrentTerm :one
SELECT * FROM terms WHERE is_current;

-- name: ClearCurrentTerm :exec
UPDATE terms SET is_current = false WHERE is_current;

-- name: MakeTermCurrent :one
UPDATE terms SET is_current = true WHERE id = @id RETURNING *;

-- name: DeleteTerm :execrows
DELETE FROM terms WHERE id = @id;

-- name: CreateClass :one
INSERT INTO classes (tenant_id, name, rank) VALUES (@tenant_id, @name, @rank) RETURNING *;

-- name: ListClasses :many
SELECT * FROM classes ORDER BY rank, name;

-- name: UpdateClass :one
UPDATE classes SET
  name = coalesce(sqlc.narg('name'), name),
  rank = coalesce(sqlc.narg('rank'), rank)
WHERE id = @id
RETURNING *;

-- name: DeleteClass :execrows
DELETE FROM classes WHERE id = @id;

-- name: CreateSection :one
INSERT INTO sections (tenant_id, class_id, name, capacity, teacher_id)
VALUES (@tenant_id, @class_id, @name, @capacity, @teacher_id)
RETURNING *;

-- name: ListSections :many
SELECT sqlc.embed(s), c.name AS class_name, u.full_name AS teacher_name,
       (SELECT count(*) FROM placements p WHERE p.section_id = s.id) AS students
FROM sections s
JOIN classes c ON c.id = s.class_id
LEFT JOIN users u ON u.id = s.teacher_id
ORDER BY c.rank, c.name, s.name;

-- name: UpdateSection :one
UPDATE sections SET
  name       = coalesce(sqlc.narg('name'), name),
  capacity   = CASE WHEN @set_capacity::boolean THEN sqlc.narg('capacity') ELSE capacity END,
  teacher_id = CASE WHEN @set_teacher::boolean THEN sqlc.narg('teacher_id') ELSE teacher_id END
WHERE id = @id
RETURNING *;

-- name: DeleteSection :execrows
DELETE FROM sections WHERE id = @id;

-- name: CreateSubject :one
INSERT INTO subjects (tenant_id, class_id, name, code, dir)
VALUES (@tenant_id, @class_id, @name, @code, @dir)
RETURNING *;

-- name: ListSubjects :many
SELECT sqlc.embed(s), c.name AS class_name
FROM subjects s LEFT JOIN classes c ON c.id = s.class_id
ORDER BY c.rank NULLS FIRST, s.name;

-- name: DeleteSubject :execrows
DELETE FROM subjects WHERE id = @id;

-- name: PlaceStudent :one
INSERT INTO placements (tenant_id, year_id, section_id, user_id, roll_no)
VALUES (@tenant_id, @year_id, @section_id, @user_id, @roll_no)
ON CONFLICT (year_id, user_id) DO UPDATE
SET section_id = excluded.section_id, roll_no = excluded.roll_no
RETURNING *;

-- name: RemovePlacement :execrows
DELETE FROM placements WHERE id = @id;

-- name: ListSectionRoll :many
SELECT sqlc.embed(p), u.full_name, u.phone, u.email
FROM placements p JOIN users u ON u.id = p.user_id
WHERE p.section_id = @section_id
ORDER BY p.roll_no NULLS LAST, u.full_name;

-- name: StudentPlacement :one
SELECT sqlc.embed(p), s.name AS section_name, c.name AS class_name, y.name AS year_name
FROM placements p
JOIN sections s ON s.id = p.section_id
JOIN classes c ON c.id = s.class_id
JOIN academic_years y ON y.id = p.year_id
WHERE p.user_id = @user_id AND y.is_current;
