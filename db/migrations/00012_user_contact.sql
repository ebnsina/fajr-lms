-- +goose Up
-- +goose StatementBegin
-- Notifications are sent outside any tenant scope, so the lookup is audited.
CREATE FUNCTION auth_find_user_by_id(p_user_id uuid) RETURNS SETOF users
LANGUAGE sql STABLE SECURITY DEFINER SET search_path = public AS $$
  SELECT * FROM users WHERE id = p_user_id
$$;
REVOKE ALL ON FUNCTION auth_find_user_by_id(uuid) FROM PUBLIC;
GRANT EXECUTE ON FUNCTION auth_find_user_by_id(uuid) TO fajr_app;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP FUNCTION IF EXISTS auth_find_user_by_id(uuid);
-- +goose StatementEnd
