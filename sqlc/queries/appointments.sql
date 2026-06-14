-- name: CreateAppointment :one
INSERT INTO appointments (
    artist_id, booking_request_id, type, status,
    scheduled_start, duration_minutes, format, scheduling_origin
) VALUES (
    @artist_id, @booking_request_id, @type, @status,
    sqlc.narg('scheduled_start'), @duration_minutes, sqlc.narg('format'), @scheduling_origin
)
RETURNING *;

-- name: GetLatestAppointmentByRequest :one
SELECT *
FROM appointments
WHERE booking_request_id = $1
ORDER BY created_at DESC
LIMIT 1;

-- name: ListLatestAppointmentsByArtist :many
-- The most recent appointment per request, so the inbox list can show the
-- scheduled/proposed state without an N+1 lookup.
SELECT DISTINCT ON (booking_request_id) *
FROM appointments
WHERE artist_id = $1
ORDER BY booking_request_id, created_at DESC;

-- name: UpdateAppointmentSchedule :one
UPDATE appointments
SET scheduled_start  = @scheduled_start::timestamptz,
    duration_minutes = COALESCE(sqlc.narg('duration_minutes')::integer, duration_minutes),
    format           = COALESCE(sqlc.narg('format')::text, format),
    status           = 'scheduled',
    updated_at       = now()
WHERE id = @id AND artist_id = @artist_id
RETURNING *;

-- name: CountOverlappingAppointments :one
-- Counts the artist's live, time-locked appointments whose [start, start+duration)
-- window overlaps the proposed one — used to block double-booking. Proposed
-- client-scheduled slots (NULL start) and cancelled/no-show slots never conflict.
SELECT COUNT(*)
FROM appointments
WHERE artist_id = @artist_id
  AND scheduled_start IS NOT NULL
  AND status NOT IN ('cancelled', 'no_show')
  AND booking_request_id <> @exclude_request_id
  AND tstzrange(scheduled_start, scheduled_start + duration_minutes * interval '1 minute')
      && tstzrange(@window_start::timestamptz, @window_end::timestamptz);
