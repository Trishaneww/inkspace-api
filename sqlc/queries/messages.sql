-- name: CreateMessage :one
INSERT INTO messages (conversation_id, sender_type, sender_user_id, body, ai_drafted)
VALUES ($1, $2, $3, $4, $5)
RETURNING *;

-- name: ListMessages :many
SELECT * FROM messages
WHERE conversation_id = $1
ORDER BY created_at ASC;

-- name: CountPriorUnreadFromSender :one
SELECT COUNT(*)
FROM messages m
WHERE m.conversation_id = @conversation_id
  AND m.sender_type = @sender_type
  AND (@recipient_last_read::timestamptz IS NULL OR m.created_at > @recipient_last_read)
  AND m.created_at < (SELECT t.created_at FROM messages t WHERE t.id = @message_id);
