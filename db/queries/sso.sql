-- name: SetIdentityProvider :one
INSERT INTO identity_providers (tenant_id, label, issuer, client_id, client_secret,
                                allowed_domains, join_role, auto_join, enabled)
VALUES (@tenant_id, @label, @issuer, @client_id, @client_secret,
        @allowed_domains, @join_role, @auto_join, @enabled)
ON CONFLICT (tenant_id) DO UPDATE SET
  label = excluded.label,
  issuer = excluded.issuer,
  client_id = excluded.client_id,
  client_secret = CASE WHEN excluded.client_secret = '' THEN identity_providers.client_secret
                       ELSE excluded.client_secret END,
  allowed_domains = excluded.allowed_domains,
  join_role = excluded.join_role,
  auto_join = excluded.auto_join,
  enabled = excluded.enabled
RETURNING *;

-- name: GetIdentityProvider :one
SELECT * FROM identity_providers WHERE tenant_id = @tenant_id;

-- name: DeleteIdentityProvider :execrows
DELETE FROM identity_providers WHERE tenant_id = @tenant_id;

-- name: SSOProviderFor :one
SELECT * FROM sso_provider_for(@slug);

-- name: SSOProviderByID :one
SELECT * FROM sso_provider_by_id(@id);

-- name: StartSSOLogin :one
SELECT * FROM sso_login_start(@state, @provider_id, @nonce, @verifier, @redirect_uri, @ttl_s);

-- name: TakeSSOLogin :one
SELECT * FROM sso_login_take(@state);

-- name: SSOUser :one
SELECT * FROM sso_user_for(@provider_id, @subject);

-- name: LinkSSOUser :exec
SELECT sso_link_user(@provider_id, @subject, @user_id);

-- name: JoinBySSO :exec
SELECT sso_join(@tenant_id, @user_id, @join_role);
