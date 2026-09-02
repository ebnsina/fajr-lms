-- name: CreateNotification :one
INSERT INTO notifications (tenant_id, user_id, kind, title, body, data)
VALUES (@tenant_id, @user_id, @kind, @title, @body, @data)
RETURNING *;

-- name: QueueDelivery :one
INSERT INTO notification_deliveries (notification_id, tenant_id, channel, destination, body)
VALUES (@notification_id, @tenant_id, @channel, @destination, @body)
RETURNING *;

-- name: ListInbox :many
SELECT * FROM notifications
WHERE user_id = @user_id
ORDER BY created_at DESC
LIMIT @page_limit OFFSET @page_offset;

-- name: CountUnread :one
SELECT count(*) FROM notifications WHERE user_id = @user_id AND read_at IS NULL;

-- name: MarkRead :execrows
UPDATE notifications SET read_at = now() WHERE id = @id AND user_id = @user_id AND read_at IS NULL;

-- name: MarkAllRead :execrows
UPDATE notifications SET read_at = now() WHERE user_id = @user_id AND read_at IS NULL;

-- name: ClaimDeliveries :many
SELECT * FROM claim_deliveries(@claim_limit);

-- name: SettleDelivery :exec
SELECT settle_delivery(@delivery_id, @new_state, @failure, @backoff);

-- name: DeliveriesForNotification :many
SELECT * FROM notification_deliveries WHERE notification_id = @notification_id ORDER BY channel;
