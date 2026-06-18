-- name: CreateBookingRequest :one
INSERT INTO booking_requests (
    artist_id, open_book_id, location_id, type, flash_id, description, reference_image_keys,
    placement, approx_size_inches, color_type, styles, client_availability, custom_answers,
    client_name, client_email, client_phone, status, deposit_status, waiver_status,
    session_duration_minutes, flash_size_code
) VALUES (
    @artist_id, @open_book_id, sqlc.narg('location_id'), @type, sqlc.narg('flash_id'), @description, @reference_image_keys,
    @placement, sqlc.narg('approx_size_inches'), @color_type, @styles, @client_availability, @custom_answers,
    @client_name, @client_email, @client_phone, @status, @deposit_status, @waiver_status,
    sqlc.narg('session_duration_minutes'), @flash_size_code
)
RETURNING *;

-- name: ListBookingRequestsByArtist :many
SELECT *
FROM booking_requests
WHERE artist_id = $1
ORDER BY created_at DESC;

-- name: GetBookingRequest :one
SELECT *
FROM booking_requests
WHERE id = $1 AND artist_id = $2;

-- name: ListBookingRequestsByClientEmail :many
SELECT *
FROM booking_requests
WHERE client_email = $1
ORDER BY created_at DESC;

-- name: DeclineOtherFlashRequests :exec
UPDATE booking_requests
SET status     = 'declined',
    decided_at = now(),
    updated_at = now()
WHERE artist_id = @artist_id
  AND flash_id = @flash_id
  AND status = 'pending'
  AND id <> @exclude_id;

-- name: UpdateBookingRequestStatus :one
UPDATE booking_requests
SET status                   = @status::text,
    session_duration_minutes = COALESCE(sqlc.narg('session_duration_minutes')::integer, session_duration_minutes),
    deposit_status           = COALESCE(sqlc.narg('deposit_status')::text, deposit_status),
    decided_at               = now(),
    updated_at               = now()
WHERE id = @id AND artist_id = @artist_id
RETURNING *;

-- name: ReopenBookingRequest :one
UPDATE booking_requests
SET status         = 'pending',
    deposit_status = 'not_required',
    decided_at     = NULL,
    updated_at     = now()
WHERE id = @id AND artist_id = @artist_id
RETURNING *;

-- name: GetBookingStats :one
SELECT
    COUNT(*) FILTER (WHERE status = 'pending')        AS new_inquiries,
    COUNT(*) FILTER (WHERE deposit_status = 'pending') AS awaiting_deposit,
    COUNT(*) FILTER (
        WHERE status IN ('accepted', 'converted')
          AND decided_at >= date_trunc('month', now())
    )                                                  AS booked_this_month
FROM booking_requests
WHERE artist_id = $1;

-- name: LinkBookingRequestsToClient :exec
UPDATE booking_requests
SET client_user_id = @client_user_id,
    updated_at = now()
WHERE client_email = @client_email
  AND client_user_id IS NULL;
