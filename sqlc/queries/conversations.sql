-- name: GetOrCreateConversation :one
INSERT INTO conversations (artist_id, booking_request_id, client_name, client_email, client_access_token)
VALUES ($1, $2, $3, $4, $5)
ON CONFLICT (artist_id, client_email)
DO UPDATE SET updated_at = conversations.updated_at
RETURNING *;

-- name: GetConversationForArtist :one
SELECT * FROM conversations WHERE id = $1 AND artist_id = $2;

-- name: GetConversationForClient :one
SELECT * FROM conversations WHERE id = $1 AND client_email = $2;

-- name: GetConversationByToken :one
SELECT * FROM conversations WHERE client_access_token = $1;

-- name: GetConversationByID :one
SELECT * FROM conversations WHERE id = $1;

-- name: GetBookingRequestPhone :one
SELECT client_phone FROM booking_requests WHERE id = $1;

-- name: ListConversationFlagsForArtist :many
SELECT
    c.id,
    c.client_email,
    EXISTS(
        SELECT 1 FROM messages m
        WHERE m.conversation_id = c.id AND m.sender_type = 'artist'
    )::boolean AS artist_replied
FROM conversations c
WHERE c.artist_id = $1;

-- name: ListConversationsForArtist :many
SELECT
    c.*,
    COALESCE((
        SELECT m.body FROM messages m
        WHERE m.conversation_id = c.id
        ORDER BY m.created_at DESC LIMIT 1
    ), '')::text AS last_message_body,
    COALESCE((
        SELECT m.sender_type = 'client'
               AND (c.artist_last_read_at IS NULL OR m.created_at > c.artist_last_read_at)
        FROM messages m
        WHERE m.conversation_id = c.id
        ORDER BY m.created_at DESC LIMIT 1
    ), false)::boolean AS has_unread
FROM conversations c
WHERE c.artist_id = $1
ORDER BY c.last_message_at DESC NULLS LAST, c.created_at DESC;

-- name: ListConversationsForClient :many
SELECT
    c.*,
    COALESCE(
        NULLIF(TRIM(COALESCE(u.first_name, '') || ' ' || COALESCE(u.last_name, '')), ''),
        u.username,
        ''
    )::text AS artist_name,
    COALESCE((
        SELECT m.body FROM messages m
        WHERE m.conversation_id = c.id
        ORDER BY m.created_at DESC LIMIT 1
    ), '')::text AS last_message_body,
    COALESCE((
        SELECT m.sender_type = 'artist'
               AND (c.client_last_read_at IS NULL OR m.created_at > c.client_last_read_at)
        FROM messages m
        WHERE m.conversation_id = c.id
        ORDER BY m.created_at DESC LIMIT 1
    ), false)::boolean AS has_unread
FROM conversations c
JOIN artists a ON a.id = c.artist_id
JOIN users u ON u.id = a.user_id
WHERE c.client_email = $1
ORDER BY c.last_message_at DESC NULLS LAST, c.created_at DESC;

-- name: TouchConversationAfterSend :exec
UPDATE conversations
SET last_message_at     = now(),
    updated_at          = now(),
    artist_last_read_at = CASE WHEN @sender_type::text = 'artist' THEN now() ELSE artist_last_read_at END,
    client_last_read_at = CASE WHEN @sender_type::text = 'client' THEN now() ELSE client_last_read_at END
WHERE id = @id;

-- name: MarkConversationReadByArtist :exec
UPDATE conversations
SET artist_last_read_at = now(), updated_at = now()
WHERE id = $1 AND artist_id = $2;

-- name: MarkConversationReadByClient :exec
UPDATE conversations
SET client_last_read_at = now(), updated_at = now()
WHERE id = $1;

-- name: GetArtistDisplayName :one
SELECT COALESCE(
    NULLIF(TRIM(COALESCE(u.first_name, '') || ' ' || COALESCE(u.last_name, '')), ''),
    u.username,
    ''
)::text AS display_name
FROM artists a
JOIN users u ON u.id = a.user_id
WHERE a.id = $1;
