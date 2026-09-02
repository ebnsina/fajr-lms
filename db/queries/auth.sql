-- name: CountRecentOTPs :one
SELECT count(*) FROM otp_challenges
WHERE destination = @destination AND purpose = @purpose AND created_at > now() - @lookback::interval;

-- name: CreateOTPChallenge :one
INSERT INTO otp_challenges (destination, purpose, code_hash, expires_at)
VALUES (@destination, @purpose, @code_hash, now() + @ttl_interval::interval)
RETURNING *;

-- name: LatestOTPChallenge :one
SELECT * FROM otp_challenges
WHERE destination = @destination AND purpose = @purpose AND consumed_at IS NULL
ORDER BY created_at DESC LIMIT 1;

-- name: RecordOTPAttempt :exec
UPDATE otp_challenges SET attempts = attempts + 1 WHERE id = @id;

-- name: ConsumeOTPChallenge :exec
UPDATE otp_challenges SET consumed_at = now() WHERE id = @id AND consumed_at IS NULL;

-- name: CreateSession :one
INSERT INTO sessions (user_id, token_hash, user_agent, ip, expires_at)
VALUES (@user_id, @token_hash, @user_agent, @ip, now() + @ttl_interval::interval)
RETURNING *;

-- name: GetSessionByToken :one
SELECT session_id, user_id, full_name, expires_at FROM live_sessions WHERE token_hash = @token_hash;

-- name: TouchSession :exec
UPDATE sessions SET last_used_at = now() WHERE id = @id;

-- name: RevokeSession :exec
UPDATE sessions SET revoked_at = now() WHERE id = @id AND revoked_at IS NULL;

-- name: RevokeUserSessions :exec
UPDATE sessions SET revoked_at = now() WHERE user_id = @user_id AND revoked_at IS NULL;

-- name: DeleteExpiredAuthRecords :exec
WITH s AS (DELETE FROM sessions WHERE expires_at < now() - interval '30 days')
DELETE FROM otp_challenges WHERE created_at < now() - interval '7 days';
