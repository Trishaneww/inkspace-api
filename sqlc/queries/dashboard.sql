-- name: GetDashboardCounts :one
SELECT
    COUNT(*) FILTER (WHERE status = 'pending')          AS new_inquiries,
    COUNT(*) FILTER (WHERE deposit_status = 'pending')  AS awaiting_deposit,
    COUNT(*) FILTER (WHERE status = 'accepted')         AS scheduled,
    COUNT(*) FILTER (
        WHERE created_at >= date_trunc('month', now())
    )                                                   AS leads_this_month,
    COUNT(*) FILTER (
        WHERE created_at >= date_trunc('month', now()) - interval '1 month'
          AND created_at <  date_trunc('month', now())
    )                                                   AS leads_last_month
FROM booking_requests
WHERE artist_id = $1;

-- name: GetBookingMixForRange :one
SELECT
    COUNT(*) FILTER (
        WHERE type = 'custom'
          AND status IN ('accepted', 'converted')
          AND decided_at >= @current_start
    )                                                   AS current_custom,
    COUNT(*) FILTER (
        WHERE type = 'flash'
          AND status IN ('accepted', 'converted')
          AND decided_at >= @current_start
    )                                                   AS current_flash,
    COUNT(*) FILTER (
        WHERE status IN ('accepted', 'converted')
          AND decided_at >= @prev_start
          AND decided_at <  @current_start
    )                                                   AS prev_bookings
FROM booking_requests
WHERE artist_id = @artist_id;

-- name: GetMonthlyEarnings :many
SELECT
    date_trunc('month', paid_at)::timestamptz                 AS month,
    COALESCE(SUM(client_charge_cents), 0)::bigint             AS collected_cents,
    COALESCE(SUM(client_charge_cents - platform_fee_cents), 0)::bigint AS net_cents
FROM payment_requests
WHERE artist_id = @artist_id
  AND status = 'paid'
  AND paid_at >= @since
GROUP BY 1
ORDER BY 1;

-- name: ListUpcomingAppointments :many
SELECT
    a.id,
    a.type,
    a.scheduled_start,
    a.duration_minutes,
    br.client_name,
    br.client_email,
    br.type AS request_type
FROM appointments a
JOIN booking_requests br ON br.id = a.booking_request_id
WHERE a.artist_id = @artist_id
  AND a.status = 'scheduled'
  AND a.scheduled_start >= now()
ORDER BY a.scheduled_start ASC
LIMIT 6;
