-- name: RecordDemoLead :one
SELECT record_demo_lead(@full_name, @email, @phone, @organisation, @role, @learners, @runs, @note);

-- name: MarkTenantDemo :exec
UPDATE tenants SET demo = true, institution = @institution WHERE id = @id;

-- name: CountTenantCourses :one
SELECT count(*) FROM courses;
