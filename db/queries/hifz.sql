-- name: ListSurahs :many
SELECT * FROM surahs ORDER BY number;

-- name: SurahAyahCount :one
SELECT ayah_count FROM surahs WHERE number = @number;

-- name: RecordHifz :one
INSERT INTO hifz_entries (tenant_id, student_id, teacher_id, on_date, kind,
                          from_surah, from_ayah, to_surah, to_ayah, quality, mistakes, note)
VALUES (@tenant_id, @student_id, @teacher_id, @on_date, @kind,
        @from_surah, @from_ayah, @to_surah, @to_ayah, @quality, @mistakes, @note)
RETURNING *;

-- name: ListHifzForStudent :many
SELECT sqlc.embed(h), s.name_en AS from_name, e.name_en AS to_name, u.full_name AS teacher_name
FROM hifz_entries h
JOIN surahs s ON s.number = h.from_surah
JOIN surahs e ON e.number = h.to_surah
LEFT JOIN users u ON u.id = h.teacher_id
WHERE h.student_id = @student_id
ORDER BY h.on_date DESC, h.created_at DESC
LIMIT @page_limit OFFSET @page_offset;

-- name: HifzOnDate :many
SELECT sqlc.embed(h), u.full_name AS student_name
FROM hifz_entries h JOIN users u ON u.id = h.student_id
WHERE h.on_date = @on_date
ORDER BY u.full_name;

-- name: DeleteHifzEntry :execrows
DELETE FROM hifz_entries WHERE id = @id;

-- name: HifzSummary :one
-- What a student has committed, counted in ayahs the way a teacher counts it:
-- every ayah inside the ranges they have taken as a new lesson.
WITH taken AS (
  SELECT DISTINCT h.from_surah, h.from_ayah, h.to_surah, h.to_ayah
  FROM hifz_entries h
  WHERE h.student_id = @student_id AND h.kind = 'sabaq'
)
SELECT
  coalesce(sum(
    CASE WHEN t.from_surah = t.to_surah THEN t.to_ayah - t.from_ayah + 1
    ELSE (SELECT s.ayah_count FROM surahs s WHERE s.number = t.from_surah) - t.from_ayah + 1
         + t.to_ayah
         + coalesce((SELECT sum(s.ayah_count) FROM surahs s
                     WHERE s.number > t.from_surah AND s.number < t.to_surah), 0)
    END
  ), 0)::bigint AS ayahs,
  count(*)::bigint AS lessons
FROM taken t;

-- name: IsGuardianOf :one
SELECT EXISTS (
  SELECT 1 FROM guardianships WHERE guardian_id = @guardian_id AND student_id = @student_id
) AS is_guardian;
