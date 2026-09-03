-- name: AwardPoints :exec
-- Nothing happens if this thing was already paid for.
INSERT INTO point_awards (tenant_id, user_id, kind, ref_id, points)
VALUES (@tenant_id, @user_id, @kind, @ref_id, @points)
ON CONFLICT (tenant_id, user_id, kind, ref_id) DO NOTHING;

-- name: MyPoints :one
SELECT coalesce(sum(points), 0)::bigint AS points FROM point_awards WHERE user_id = @user_id;

-- name: Leaderboard :many
SELECT p.user_id, u.full_name, sum(p.points)::bigint AS points
FROM point_awards p JOIN users u ON u.id = p.user_id
WHERE p.created_at >= @since
GROUP BY p.user_id, u.full_name
ORDER BY points DESC, u.full_name
LIMIT @page_limit;

-- name: SetPointsOn :one
UPDATE tenants SET points_on = @points_on WHERE id = @id RETURNING *;

-- name: CreateBadge :one
INSERT INTO badges (tenant_id, name, description, emoji, threshold)
VALUES (@tenant_id, @name, @description, @emoji, @threshold)
RETURNING *;

-- name: ListBadges :many
SELECT b.*, (SELECT count(*) FROM badge_awards a WHERE a.badge_id = b.id) AS earned_by
FROM badges b ORDER BY b.threshold;

-- name: DeleteBadge :execrows
DELETE FROM badges WHERE id = @id;

-- name: AwardBadge :exec
INSERT INTO badge_awards (badge_id, user_id, tenant_id)
VALUES (@badge_id, @user_id, @tenant_id)
ON CONFLICT DO NOTHING;

-- name: MyBadges :many
SELECT b.*, a.awarded_at
FROM badge_awards a JOIN badges b ON b.id = a.badge_id
WHERE a.user_id = @user_id
ORDER BY a.awarded_at DESC;

-- name: BadgesToAward :many
-- The badges this person has passed the mark for and does not yet hold.
SELECT b.* FROM badges b
WHERE b.threshold <= @points
  AND NOT EXISTS (SELECT 1 FROM badge_awards a WHERE a.badge_id = b.id AND a.user_id = @user_id);
