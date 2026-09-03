-- name: CreateOrder :one
INSERT INTO orders (tenant_id, user_id, course_id, provider, amount_minor, currency, reference,
                    coupon_id, list_amount_minor, discount_minor)
VALUES (@tenant_id, @user_id, @course_id, @provider, @amount_minor, @currency, @reference,
        @coupon_id, @list_amount_minor, @discount_minor)
RETURNING *;

-- name: GetOrder :one
SELECT * FROM orders WHERE id = @id;

-- name: GetOrderByReference :one
SELECT * FROM orders WHERE reference = @reference;

-- name: OpenOrderForCourse :one
SELECT * FROM orders
WHERE course_id = @course_id AND user_id = @user_id AND status IN ('pending', 'awaiting_review');

-- name: ListMyOrders :many
SELECT sqlc.embed(o), c.title, c.slug
FROM orders o JOIN courses c ON c.id = o.course_id
WHERE o.user_id = @user_id
ORDER BY o.created_at DESC
LIMIT @page_limit OFFSET @page_offset;

-- name: ListOrdersForReview :many
SELECT sqlc.embed(o), c.title, u.full_name
FROM orders o JOIN courses c ON c.id = o.course_id JOIN users u ON u.id = o.user_id
WHERE o.status = 'awaiting_review'
ORDER BY o.created_at
LIMIT @page_limit OFFSET @page_offset;

-- name: SubmitPaymentProof :one
UPDATE orders SET
  status = 'awaiting_review',
  proof_media_id = @proof_media_id,
  provider_ref = @provider_ref,
  note = @note
WHERE id = @id AND status = 'pending'
RETURNING *;

-- name: SettleOrder :one
UPDATE orders SET
  status = @status,
  reviewed_by = @reviewed_by,
  reviewed_at = now(),
  paid_at = CASE WHEN @status::order_status = 'paid' THEN coalesce(paid_at, now()) ELSE paid_at END
WHERE id = @id AND status IN ('pending', 'awaiting_review')
RETURNING *;

-- name: CancelOrder :one
UPDATE orders SET status = 'cancelled'
WHERE id = @id AND status IN ('pending', 'awaiting_review')
RETURNING *;

-- name: RecordPaymentEvent :one
INSERT INTO payment_events (tenant_id, order_id, provider, event_id, kind, payload)
VALUES (@tenant_id, @order_id, @provider, @event_id, @kind, @payload)
ON CONFLICT (provider, event_id) DO NOTHING
RETURNING *;

-- name: ListOrderEvents :many
SELECT * FROM payment_events WHERE order_id = @order_id ORDER BY created_at;

-- name: RefundOrder :one
-- The sum is checked in the statement, so two refunds racing cannot together
-- hand back more than was paid.
UPDATE orders SET
  refunded_minor = refunded_minor + @amount_minor,
  refund_reason  = @reason,
  refunded_by    = @refunded_by,
  refunded_at    = now(),
  status = CASE WHEN refunded_minor + @amount_minor >= amount_minor THEN 'refunded'::order_status ELSE status END
WHERE id = @id AND status IN ('paid', 'refunded') AND refunded_minor + @amount_minor <= amount_minor
RETURNING *;

-- name: ListPaidOrders :many
SELECT sqlc.embed(o), c.title, u.full_name
FROM orders o JOIN courses c ON c.id = o.course_id JOIN users u ON u.id = o.user_id
WHERE o.status IN ('paid', 'refunded')
ORDER BY o.paid_at DESC NULLS LAST
LIMIT @page_limit OFFSET @page_offset;
