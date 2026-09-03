-- name: CreateCoupon :one
INSERT INTO coupons (tenant_id, code, kind, value, course_id, max_redemptions, starts_at, ends_at, created_by)
VALUES (@tenant_id, @code, @kind, @value, @course_id, @max_redemptions, @starts_at, @ends_at, @created_by)
RETURNING *;

-- name: ListCoupons :many
SELECT * FROM coupons ORDER BY created_at DESC LIMIT @page_limit OFFSET @page_offset;

-- name: CouponByCode :one
SELECT * FROM coupons WHERE code = @code;

-- name: SetCouponActive :one
UPDATE coupons SET active = @active WHERE id = @id RETURNING *;

-- name: DeleteCoupon :execrows
DELETE FROM coupons WHERE id = @id;

-- name: RedeemCoupon :execrows
UPDATE coupons SET redeemed = redeemed + 1
WHERE id = @id AND (max_redemptions IS NULL OR redeemed < max_redemptions);
