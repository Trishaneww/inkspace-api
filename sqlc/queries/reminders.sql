-- name: ClaimDuePaymentReminders :many
UPDATE payment_requests
SET reminder_sent_at = now(),
    updated_at       = now()
WHERE status = 'requested'
  AND reminder_sent_at IS NULL
  AND created_at <= @cutoff
  AND expires_at > now()
RETURNING id, artist_id, type, currency, client_charge_cents,
          client_name, client_email, public_token;

-- name: ClaimDueSessionReminders :many
UPDATE appointments a
SET reminder_sent_at = now(),
    updated_at       = now()
FROM booking_requests br
WHERE a.booking_request_id = br.id
  AND a.status = 'scheduled'
  AND a.type IN ('consultation', 'session')
  AND a.reminder_sent_at IS NULL
  AND a.scheduled_start >= now()
  AND a.scheduled_start <= @window_end
  AND br.client_phone <> ''
RETURNING a.id, a.artist_id, a.type, a.scheduled_start,
          br.client_name, br.client_phone;

-- name: GetArtistContact :one
SELECT u.first_name, u.last_name, u.username, s.timezone
FROM artists ar
JOIN users u ON u.id = ar.user_id
LEFT JOIN artist_settings s ON s.artist_id = ar.id
WHERE ar.id = $1;
