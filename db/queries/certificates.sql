-- name: IssueCertificate :one
INSERT INTO certificates (tenant_id, course_id, enrollment_id, user_id, serial,
                          recipient_name, course_title, issuer_name, grade_percent, issued_by)
VALUES (@tenant_id, @course_id, @enrollment_id, @user_id, @serial,
        @recipient_name, @course_title, @issuer_name, @grade_percent, @issued_by)
RETURNING *;

-- name: GetCertificate :one
SELECT * FROM certificates WHERE id = @id;

-- name: CertificateForEnrollment :one
SELECT * FROM certificates WHERE enrollment_id = @enrollment_id;

-- name: ListMyCertificates :many
SELECT sqlc.embed(c), co.slug AS course_slug
FROM certificates c JOIN courses co ON co.id = c.course_id
WHERE c.user_id = @user_id
ORDER BY c.issued_at DESC;

-- name: ListCourseCertificates :many
SELECT sqlc.embed(c), u.full_name
FROM certificates c JOIN users u ON u.id = c.user_id
WHERE c.course_id = @course_id
ORDER BY c.issued_at DESC
LIMIT @page_limit OFFSET @page_offset;

-- name: RevokeCertificate :one
UPDATE certificates SET revoked_at = now(), revoked_reason = @revoked_reason
WHERE id = @id AND revoked_at IS NULL
RETURNING *;

-- name: VerifyCertificate :one
SELECT * FROM certificate_verifications WHERE serial = @serial;
