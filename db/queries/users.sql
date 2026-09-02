-- name: SignupUser :one
SELECT * FROM signup_user(@phone, @email, @full_name, @password_hash);

-- name: GetUser :one
SELECT * FROM users WHERE id = $1;

-- name: FindUserForAuth :one
SELECT * FROM auth_find_user(@phone, @email);

-- name: ListUserMemberships :many
SELECT * FROM auth_memberships(@user_id);
