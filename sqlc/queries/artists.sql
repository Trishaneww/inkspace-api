-- name: EnsureArtist :exec
INSERT INTO artists (user_id)
VALUES ($1)
ON CONFLICT (user_id) DO NOTHING;

-- name: GetArtistByUserID :one
SELECT * FROM artists WHERE user_id = $1;

-- name: GetArtistByID :one
SELECT * FROM artists WHERE id = $1;

-- name: SetArtistOnboardedAt :exec
UPDATE artists
SET onboarded_at = now(),
    updated_at   = now()
WHERE id = $1;

-- name: GetArtistOnboardedAt :one
SELECT onboarded_at FROM artists WHERE user_id = $1;

-- name: GetArtistByStripeCustomerID :one
SELECT * FROM artists WHERE stripe_customer_id = $1;

-- name: SetArtistStripeCustomerID :exec
UPDATE artists
SET stripe_customer_id = $2,
    updated_at         = now()
WHERE id = $1;

-- name: UpdateArtistSubscription :exec
UPDATE artists
SET stripe_subscription_id            = $2,
    subscription_status               = $3,
    subscription_current_period_end   = $4,
    subscription_cancel_at_period_end = $5,
    updated_at                        = now()
WHERE id = $1;
