-- name: CreatePaymentRequest :one
INSERT INTO payment_requests (
    artist_id, booking_request_id, type, currency,
    amount_cents, platform_fee_cents, client_charge_cents, fee_payer,
    client_email, client_name, description, public_token, expires_at,
    scheduled_start
) VALUES (
    @artist_id, @booking_request_id, @type, @currency,
    @amount_cents, @platform_fee_cents, @client_charge_cents, @fee_payer,
    @client_email, @client_name, @description, @public_token, @expires_at,
    sqlc.narg('scheduled_start')
)
RETURNING *;

-- name: SetDepositScheduledStart :exec
UPDATE payment_requests
SET scheduled_start = sqlc.narg('scheduled_start'),
    updated_at      = now()
WHERE booking_request_id = @booking_request_id
  AND type = 'deposit'
  AND status IN ('requested', 'processing');

-- name: GetPaymentRequestByToken :one
SELECT * FROM payment_requests WHERE public_token = $1;

-- name: GetPaymentRequestForArtist :one
SELECT * FROM payment_requests WHERE id = @id AND artist_id = @artist_id;

-- name: MarkPaymentRequestEmailed :one
UPDATE payment_requests
SET last_emailed_at = now(),
    updated_at      = now()
WHERE id = @id AND artist_id = @artist_id
RETURNING *;

-- name: GetPaymentRequestByPaymentIntent :one
SELECT * FROM payment_requests WHERE stripe_payment_intent_id = $1;

-- name: ListPaymentRequestsByArtist :many
SELECT * FROM payment_requests
WHERE artist_id = $1
ORDER BY created_at DESC;

-- name: ListPaymentRequestsByBooking :many
SELECT * FROM payment_requests
WHERE booking_request_id = $1
ORDER BY created_at DESC;

-- name: CountLivePaymentRequests :one
SELECT COUNT(*) FROM payment_requests
WHERE booking_request_id = @booking_request_id
  AND type = @type
  AND status IN ('requested', 'processing', 'paid');

-- name: AttachCheckoutSession :one
UPDATE payment_requests
SET stripe_checkout_session_id = @stripe_checkout_session_id,
    stripe_payment_intent_id   = sqlc.narg('stripe_payment_intent_id'),
    status                     = 'processing',
    updated_at                 = now()
WHERE id = @id
RETURNING *;

-- name: MarkPaymentRequestPaidByPaymentIntent :one
UPDATE payment_requests
SET status     = 'paid',
    paid_at    = now(),
    updated_at = now()
WHERE stripe_payment_intent_id = @stripe_payment_intent_id
  AND status NOT IN ('paid', 'refunded')
RETURNING *;

-- name: MarkPaymentRequestPaidBySession :one
UPDATE payment_requests
SET status                   = 'paid',
    stripe_payment_intent_id = COALESCE(sqlc.narg('stripe_payment_intent_id'), stripe_payment_intent_id),
    paid_at                  = now(),
    updated_at               = now()
WHERE stripe_checkout_session_id = @stripe_checkout_session_id
  AND status NOT IN ('paid', 'refunded')
RETURNING *;

-- name: MarkPaymentRequestFailedByPaymentIntent :one
UPDATE payment_requests
SET status     = 'failed',
    updated_at = now()
WHERE stripe_payment_intent_id = @stripe_payment_intent_id
  AND status NOT IN ('paid', 'refunded')
RETURNING *;

-- name: MarkPaymentRequestExpiredBySession :one
UPDATE payment_requests
SET status     = 'expired',
    updated_at = now()
WHERE stripe_checkout_session_id = @stripe_checkout_session_id
  AND status IN ('requested', 'processing')
RETURNING *;

-- name: RefundPaymentRequestByPaymentIntent :one
UPDATE payment_requests
SET status                = 'refunded',
    amount_refunded_cents = @amount_refunded_cents,
    updated_at            = now()
WHERE stripe_payment_intent_id = @stripe_payment_intent_id
RETURNING *;

-- name: CancelPaymentRequest :one
UPDATE payment_requests
SET status      = 'canceled',
    canceled_at = now(),
    updated_at  = now()
WHERE id = @id AND artist_id = @artist_id
  AND status IN ('requested', 'processing')
RETURNING *;

-- name: SetBookingDepositStatus :exec
UPDATE booking_requests
SET deposit_status = @deposit_status::text,
    updated_at     = now()
WHERE id = @booking_request_id;

-- name: SetBookingDepositAmount :exec
UPDATE booking_requests
SET deposit_amount_cents = sqlc.narg('deposit_amount_cents'),
    updated_at           = now()
WHERE id = @booking_request_id;

-- name: SumPaidDepositsForBooking :one
SELECT COALESCE(SUM(amount_cents), 0)::bigint AS deposit_paid_cents
FROM payment_requests
WHERE booking_request_id = @booking_request_id
  AND type = 'deposit'
  AND status = 'paid';

-- name: ExpireDepositRequestsForBooking :exec
UPDATE payment_requests
SET status     = 'expired',
    updated_at = now()
WHERE booking_request_id = @booking_request_id
  AND type = 'deposit'
  AND status IN ('requested', 'processing');

-- name: GetArtistEarnings :one
SELECT
    COALESCE(SUM(client_charge_cents), 0)::bigint AS collected_cents,
    COALESCE(SUM(platform_fee_cents), 0)::bigint AS fee_cents,
    COUNT(*)::bigint AS paid_count,
    COALESCE(SUM(client_charge_cents) FILTER (
        WHERE paid_at >= date_trunc('month', now())), 0)::bigint AS month_collected_cents,
    COALESCE(SUM(platform_fee_cents) FILTER (
        WHERE paid_at >= date_trunc('month', now())), 0)::bigint AS month_fee_cents,
    COUNT(*) FILTER (
        WHERE paid_at >= date_trunc('month', now()))::bigint AS month_paid_count
FROM payment_requests
WHERE artist_id = @artist_id AND status = 'paid';

-- name: ListRecentPaidPaymentsForArtist :many
SELECT
    pr.id, pr.type, pr.status, pr.currency,
    pr.amount_cents, pr.platform_fee_cents, pr.client_charge_cents,
    pr.paid_at, pr.created_at,
    br.client_name, br.client_email,
    br.type AS request_type, br.placement
FROM payment_requests pr
JOIN booking_requests br ON br.id = pr.booking_request_id
WHERE pr.artist_id = @artist_id AND pr.status = 'paid'
ORDER BY pr.paid_at DESC NULLS LAST
LIMIT 50;

-- name: SeedMarkPaymentPaid :exec
UPDATE payment_requests
SET status = 'paid', paid_at = @paid_at, updated_at = now()
WHERE id = @id;
