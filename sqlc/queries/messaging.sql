-- name: CreateConversation :one
INSERT INTO conversations (client_id, artist_id, booking_id)
VALUES ($1, $2, $3)
RETURNING *;

-- name: GetConversationByID :one
SELECT * FROM conversations WHERE id = $1;

-- name: FindConversationForPair :one
SELECT * FROM conversations
WHERE client_id = @client_id
  AND artist_id = @artist_id
  AND ((@booking_id::uuid IS NULL AND booking_id IS NULL)
       OR booking_id = @booking_id);

-- name: ListConversationsForClient :many
SELECT * FROM conversations
WHERE client_id = $1
ORDER BY last_message_at DESC
LIMIT $2 OFFSET $3;

-- name: ListConversationsForArtist :many
SELECT * FROM conversations
WHERE artist_id = $1
ORDER BY last_message_at DESC
LIMIT $2 OFFSET $3;

-- name: TouchConversationLastMessageAt :exec
UPDATE conversations
SET last_message_at = now()
WHERE id = $1;

-- name: InsertMessage :one
INSERT INTO messages (conversation_id, sender_id, body, attachments)
VALUES ($1, $2, $3, $4)
RETURNING *;

-- name: ListMessages :many
SELECT * FROM messages
WHERE conversation_id = @conversation_id
  AND (sqlc.narg('before')::timestamptz IS NULL OR created_at < sqlc.narg('before'))
ORDER BY created_at DESC
LIMIT @lim;

-- name: GetReadCursor :one
SELECT * FROM conversation_read_cursors
WHERE conversation_id = $1 AND user_id = $2;

-- name: UpsertReadCursor :one
INSERT INTO conversation_read_cursors (conversation_id, user_id, last_read_at)
VALUES ($1, $2, now())
ON CONFLICT (conversation_id, user_id) DO UPDATE
SET last_read_at = EXCLUDED.last_read_at
RETURNING *;
