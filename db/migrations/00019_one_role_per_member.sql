-- +goose Up
-- +goose StatementBegin

-- A person holds one role in a school. The rest of the product already assumes
-- it: the session carries a single role, and the switcher shows one per school.
DELETE FROM memberships m
USING memberships keep
WHERE m.tenant_id = keep.tenant_id
  AND m.user_id = keep.user_id
  AND (keep.created_at, keep.id) < (m.created_at, m.id);

ALTER TABLE memberships DROP CONSTRAINT memberships_tenant_id_user_id_role_key;
ALTER TABLE memberships ADD CONSTRAINT memberships_tenant_id_user_id_key UNIQUE (tenant_id, user_id);

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE memberships DROP CONSTRAINT memberships_tenant_id_user_id_key;
ALTER TABLE memberships ADD CONSTRAINT memberships_tenant_id_user_id_role_key UNIQUE (tenant_id, user_id, role);
-- +goose StatementEnd
