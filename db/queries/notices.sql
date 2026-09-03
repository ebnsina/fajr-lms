-- name: SectionAudience :many
SELECT p.user_id AS student_id, g.guardian_id
FROM placements p
JOIN academic_years y ON y.id = p.year_id AND y.is_current
LEFT JOIN guardianships g ON g.student_id = p.user_id AND g.tenant_id = p.tenant_id
WHERE p.section_id = @section_id;

-- name: ClassAudience :many
SELECT p.user_id AS student_id, g.guardian_id
FROM placements p
JOIN sections s ON s.id = p.section_id
JOIN academic_years y ON y.id = p.year_id AND y.is_current
LEFT JOIN guardianships g ON g.student_id = p.user_id AND g.tenant_id = p.tenant_id
WHERE s.class_id = @class_id;

-- name: SchoolAudience :many
SELECT m.user_id AS student_id, g.guardian_id
FROM memberships m
LEFT JOIN guardianships g ON g.student_id = m.user_id AND g.tenant_id = m.tenant_id
WHERE m.role = 'student' AND m.status = 'active';

-- name: MyChildren :many
SELECT g.student_id, g.relation, u.full_name,
       (SELECT s.name FROM placements p
        JOIN sections s ON s.id = p.section_id
        JOIN academic_years y ON y.id = p.year_id AND y.is_current
        WHERE p.user_id = g.student_id) AS section_name,
       (SELECT c.name FROM placements p
        JOIN sections s ON s.id = p.section_id
        JOIN classes c ON c.id = s.class_id
        JOIN academic_years y ON y.id = p.year_id AND y.is_current
        WHERE p.user_id = g.student_id) AS class_name
FROM guardianships g JOIN users u ON u.id = g.student_id
WHERE g.guardian_id = @guardian_id
ORDER BY u.full_name;
