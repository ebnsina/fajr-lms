-- name: CreateSCORMPackage :one
INSERT INTO scorm_packages (tenant_id, lesson_id, title, entry_href, version, mastery,
                            file_count, bytes, grade_item_id)
VALUES (@tenant_id, @lesson_id, @title, @entry_href, @version, @mastery,
        @file_count, @bytes, @grade_item_id)
RETURNING *;

-- name: SCORMPackageForLesson :one
SELECT * FROM scorm_packages WHERE lesson_id = @lesson_id;

-- name: DeleteSCORMPackage :execrows
DELETE FROM scorm_packages WHERE lesson_id = @lesson_id;

-- name: AddSCORMFile :exec
INSERT INTO scorm_files (package_id, path, content_type, body)
VALUES (@package_id, @path, @content_type, @body)
ON CONFLICT (package_id, path) DO UPDATE SET content_type = excluded.content_type, body = excluded.body;

-- name: SCORMFile :one
SELECT content_type, body FROM scorm_files WHERE package_id = @package_id AND path = @path;

-- name: SCORMState :one
SELECT * FROM scorm_states WHERE package_id = @package_id AND user_id = @user_id;

-- name: SaveSCORMState :one
INSERT INTO scorm_states (package_id, user_id, tenant_id, cmi, lesson_status, score_raw,
                          suspend_data, location, total_time_s)
VALUES (@package_id, @user_id, @tenant_id, @cmi, @lesson_status, @score_raw,
        @suspend_data, @location, @total_time_s)
ON CONFLICT (package_id, user_id) DO UPDATE SET
  cmi = excluded.cmi,
  lesson_status = excluded.lesson_status,
  score_raw = excluded.score_raw,
  suspend_data = excluded.suspend_data,
  location = excluded.location,
  total_time_s = greatest(scorm_states.total_time_s, excluded.total_time_s)
RETURNING *;

-- name: ListSCORMStates :many
SELECT sqlc.embed(s), u.full_name
FROM scorm_states s JOIN users u ON u.id = s.user_id
WHERE s.package_id = @package_id
ORDER BY u.full_name;
