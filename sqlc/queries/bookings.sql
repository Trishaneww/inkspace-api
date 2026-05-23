-- name: CreateBooking :one
INSERT INTO bookings (client_id, artist_id, notes, estimated_hours, deposit_amount_cents)
VALUES ($1, $2, $3, $4, $5)
RETURNING *;

-- name: GetBookingByID :one
SELECT * FROM bookings WHERE id = $1;

-- name: ListBookingsByClient :many
SELECT * FROM bookings
WHERE client_id = @client_id
  AND (sqlc.narg('status')::text IS NULL OR status = sqlc.narg('status'))
ORDER BY created_at DESC
LIMIT @lim
OFFSET @off;

-- name: ListBookingsByArtist :many
SELECT * FROM bookings
WHERE artist_id = @artist_id
  AND (sqlc.narg('status')::text IS NULL OR status = sqlc.narg('status'))
ORDER BY created_at DESC
LIMIT @lim
OFFSET @off;

-- name: UpdateBookingNotes :one
UPDATE bookings
SET notes                = COALESCE(sqlc.narg('notes')::text, notes),
    estimated_hours      = COALESCE(sqlc.narg('estimated_hours')::numeric, estimated_hours),
    deposit_amount_cents = COALESCE(sqlc.narg('deposit_amount_cents')::bigint, deposit_amount_cents),
    updated_at           = now()
WHERE id = @id
RETURNING *;

-- name: UpdateBookingStatus :one
UPDATE bookings
SET status     = $2,
    updated_at = now()
WHERE id = $1
RETURNING *;

-- name: UpdateBookingSchedule :one
UPDATE bookings
SET scheduled_for = $2,
    updated_at    = now()
WHERE id = $1
RETURNING *;

-- name: DeleteBooking :exec
DELETE FROM bookings WHERE id = $1;

-- name: InsertBookingStatusHistory :one
INSERT INTO booking_status_history (booking_id, from_status, to_status, changed_by, note)
VALUES ($1, $2, $3, $4, $5)
RETURNING *;

-- name: ListBookingStatusHistory :many
SELECT * FROM booking_status_history
WHERE booking_id = $1
ORDER BY created_at ASC;
