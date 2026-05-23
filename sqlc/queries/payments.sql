-- name: CreatePayment :one
INSERT INTO payments (
    booking_id, payer_id, payee_id, kind, amount_cents, currency,
    provider, provider_intent_id
)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
RETURNING *;

-- name: GetPaymentByID :one
SELECT * FROM payments WHERE id = $1;

-- name: GetPaymentByProviderIntent :one
SELECT * FROM payments
WHERE provider = $1 AND provider_intent_id = $2;

-- name: ListPaymentsByPayer :many
SELECT * FROM payments
WHERE payer_id = $1
ORDER BY created_at DESC
LIMIT $2 OFFSET $3;

-- name: ListPaymentsByPayee :many
SELECT * FROM payments
WHERE payee_id = $1
ORDER BY created_at DESC
LIMIT $2 OFFSET $3;

-- name: ListPaymentsByBooking :many
SELECT * FROM payments
WHERE booking_id = $1
ORDER BY created_at DESC;

-- name: UpdatePaymentStatus :one
UPDATE payments
SET status     = $2,
    updated_at = now()
WHERE id = $1
RETURNING *;

-- name: InsertWebhookEvent :one
INSERT INTO payment_webhook_events (provider, event_id, payload)
VALUES ($1, $2, $3)
ON CONFLICT (provider, event_id) DO NOTHING
RETURNING *;

-- name: MarkWebhookProcessed :exec
UPDATE payment_webhook_events
SET processed = true
WHERE id = $1;

-- name: ListUnprocessedWebhookEvents :many
SELECT * FROM payment_webhook_events
WHERE processed = false
ORDER BY received_at ASC
LIMIT $1;
