-- name: GetBookingRequestByID :one
SELECT * FROM booking_requests WHERE id = $1;

-- name: SetBookingRequestTriageStatus :exec
UPDATE booking_requests
SET ai_status     = $2,
    ai_updated_at = now()
WHERE id = $1;

-- name: UpdateBookingRequestTriage :exec
UPDATE booking_requests
SET ai_status       = 'ready',
    ai_label        = $2,
    ai_summary      = $3,
    ai_signals      = $4,
    ai_red_flags    = $5,
    ai_reasoning    = $6,
    ai_value_cents  = $7,
    ai_session_count = $8,
    ai_draft_reply  = $9,
    ai_updated_at   = now()
WHERE id = $1;
