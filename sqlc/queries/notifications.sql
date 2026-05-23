-- name: InsertNotification :one
INSERT INTO notifications (user_id, kind, title, body, data)
VALUES ($1, $2, $3, $4, $5)
RETURNING *;

-- name: GetNotificationByID :one
SELECT * FROM notifications WHERE id = $1;

-- name: ListNotificationsForUser :many
SELECT * FROM notifications
WHERE user_id = @user_id
  AND (@unread_only::boolean = false OR read_at IS NULL)
ORDER BY created_at DESC
LIMIT @lim
OFFSET @off;

-- name: CountUnreadNotifications :one
SELECT count(*)::bigint FROM notifications
WHERE user_id = $1 AND read_at IS NULL;

-- name: MarkNotificationRead :exec
UPDATE notifications
SET read_at = now()
WHERE id = $1 AND read_at IS NULL;

-- name: MarkAllNotificationsRead :execrows
UPDATE notifications
SET read_at = now()
WHERE user_id = $1 AND read_at IS NULL;

-- name: GetNotificationPreferences :one
SELECT * FROM notification_preferences WHERE user_id = $1;

-- name: UpsertNotificationPreferences :one
INSERT INTO notification_preferences (user_id, enabled, quiet_hours)
VALUES ($1, $2, $3)
ON CONFLICT (user_id) DO UPDATE
SET enabled     = EXCLUDED.enabled,
    quiet_hours = EXCLUDED.quiet_hours,
    updated_at  = now()
RETURNING *;

-- name: InsertNotificationDelivery :one
INSERT INTO notification_deliveries (notification_id, channel)
VALUES ($1, $2)
RETURNING *;

-- name: UpdateDeliveryStatus :exec
UPDATE notification_deliveries
SET status       = $2,
    error        = $3,
    attempted_at = now()
WHERE id = $1;

-- name: ListPendingDeliveries :many
SELECT * FROM notification_deliveries
WHERE status = 'pending'
ORDER BY created_at ASC
LIMIT $1;
