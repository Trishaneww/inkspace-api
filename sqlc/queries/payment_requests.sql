-- name: CreatePaymentRequest :one
INSERT INTO payment_requests (
    artist_id, booking_request_id, type, currency,
    amount_cents, platform_fee_cents, client_charge_cents, fee_payer,
    client_email, client_name, description, public_token, expires_at
) VALUES (
    @artist_id, @booking_request_id, @type, @currency,
    @amount_cents, @platform_fee_cents, @client_charge_cents, @fee_payer,
    @client_email, @client_name, @description, @public_token, @expires_at
)
RETURNING *;

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
