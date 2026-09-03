-- name: CreateMembership :one
INSERT INTO memberships (tenant_id, user_id, role)
VALUES (@tenant_id, @user_id, @role)
RETURNING *;

-- name: ListTenantMembers :many
SELECT m.*, u.full_name, u.phone, u.email
FROM memberships m JOIN users u ON u.id = m.user_id
WHERE m.status = 'active'
ORDER BY u.full_name
LIMIT @page_limit OFFSET @page_offset;

-- name: CountTenantMembers :one
SELECT count(*) FROM memberships WHERE status = 'active';

-- name: GetMembership :one
SELECT * FROM memberships WHERE tenant_id = @tenant_id AND user_id = @user_id;

-- name: SetMembershipRole :one
UPDATE memberships SET role = @role
WHERE tenant_id = @tenant_id AND user_id = @user_id
RETURNING *;

-- name: DeleteMembership :execrows
DELETE FROM memberships WHERE tenant_id = @tenant_id AND user_id = @user_id;

-- name: CountOwners :one
SELECT count(*) FROM memberships WHERE role = 'owner' AND status = 'active';
