-- name: CreateAppointment :one
INSERT INTO appointments (
    artist_id, booking_request_id, type, status,
    scheduled_start, duration_minutes, format, scheduling_origin, hold_expires_at
) VALUES (
    @artist_id, @booking_request_id, @type, @status,
    sqlc.narg('scheduled_start'), @duration_minutes, sqlc.narg('format'), @scheduling_origin,
    sqlc.narg('hold_expires_at')
)
RETURNING *;

-- name: SetAppointmentCalendarEvent :exec
UPDATE appointments
SET google_calendar_event_id = sqlc.narg('google_calendar_event_id')::text,
    updated_at               = now()
WHERE id = @id;

-- name: UpdateAppointmentStatus :one
UPDATE appointments
SET status     = @status,
    updated_at = now()
WHERE id = @id AND artist_id = @artist_id
RETURNING *;

-- name: GetLatestAppointmentByRequest :one
SELECT *
FROM appointments
WHERE booking_request_id = $1
ORDER BY (status IN ('scheduled', 'awaiting_deposit', 'proposed')) DESC, created_at DESC
LIMIT 1;

-- name: ListLiveAppointmentsByRequest :many
SELECT *
FROM appointments
WHERE booking_request_id = $1 AND status IN ('scheduled', 'awaiting_deposit', 'proposed')
ORDER BY created_at;

-- name: ListLatestAppointmentsByArtist :many
-- The current appointment per request (live if any, else most recent), so the
-- inbox list can show the scheduled/held/proposed state without an N+1 lookup.
SELECT DISTINCT ON (booking_request_id) *
FROM appointments
WHERE artist_id = $1
ORDER BY booking_request_id, (status IN ('scheduled', 'awaiting_deposit', 'proposed')) DESC, created_at DESC;

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
-- A non-expired deposit hold blocks the slot; an expired hold does not.
SELECT COUNT(*)
FROM appointments
WHERE artist_id = @artist_id
  AND scheduled_start IS NOT NULL
  AND status NOT IN ('cancelled', 'no_show')
  AND (status <> 'awaiting_deposit' OR hold_expires_at > now())
  AND booking_request_id <> @exclude_request_id
  AND tstzrange(scheduled_start, scheduled_start + duration_minutes * interval '1 minute')
      && tstzrange(@window_start::timestamptz, @window_end::timestamptz);

-- name: ListAppointmentsByArtistInRange :many
-- Scheduled (time-locked) appointments for the calendar, joined with their
-- booking request and studio location. Proposed (no start) and cancelled/no-show
-- appointments are excluded — they have no place on a time grid.
SELECT
    a.id,
    a.booking_request_id,
    a.type,
    a.status,
    a.scheduled_start,
    a.duration_minutes,
    a.format,
    br.client_name,
    COALESCE(loc.label, '')   AS location_label,
    COALESCE(loc.address, '') AS location_address,
    COALESCE(loc.city, '')    AS location_city
FROM appointments a
JOIN booking_requests br ON br.id = a.booking_request_id
LEFT JOIN artist_locations loc ON loc.id = br.location_id
WHERE a.artist_id = @artist_id
  AND a.scheduled_start IS NOT NULL
  AND a.status NOT IN ('cancelled', 'no_show', 'awaiting_deposit')
  AND a.scheduled_start >= @range_start::timestamptz
  AND a.scheduled_start <  @range_end::timestamptz
ORDER BY a.scheduled_start;

-- name: ListBusyAppointmentsByArtistInRange :many
SELECT scheduled_start, duration_minutes
FROM appointments
WHERE artist_id = @artist_id
  AND scheduled_start IS NOT NULL
  AND status NOT IN ('cancelled', 'no_show')
  AND (status <> 'awaiting_deposit' OR hold_expires_at > now())
  AND scheduled_start >= @range_start::timestamptz
  AND scheduled_start <  @range_end::timestamptz
ORDER BY scheduled_start;

-- name: HoldAppointmentSchedule :one
UPDATE appointments
SET scheduled_start  = @scheduled_start::timestamptz,
    duration_minutes = COALESCE(sqlc.narg('duration_minutes')::integer, duration_minutes),
    status           = 'awaiting_deposit',
    hold_expires_at  = @hold_expires_at::timestamptz,
    updated_at       = now()
WHERE id = @id AND artist_id = @artist_id
RETURNING *;

-- name: ConfirmHeldAppointment :one
UPDATE appointments
SET status          = 'scheduled',
    hold_expires_at = NULL,
    updated_at      = now()
WHERE booking_request_id = @booking_request_id AND status = 'awaiting_deposit'
RETURNING *;

-- name: ClaimExpiredHolds :many
UPDATE appointments
SET status     = 'cancelled',
    updated_at = now()
WHERE status = 'awaiting_deposit'
  AND hold_expires_at IS NOT NULL
  AND hold_expires_at <= now()
RETURNING id, artist_id, booking_request_id, google_calendar_event_id;
