-- name: GetUserByUsername :one
SELECT * FROM users WHERE username = $1;

-- name: UpdateUserProfile :one
UPDATE users
SET first_name    = COALESCE(sqlc.narg('first_name')::text,    first_name),
    last_name     = COALESCE(sqlc.narg('last_name')::text,     last_name),
    username      = COALESCE(sqlc.narg('username')::citext,    username),
    phone         = COALESCE(sqlc.narg('phone')::text,         phone),
    avatar_url    = COALESCE(sqlc.narg('avatar_url')::text,    avatar_url),
    instagram_url = COALESCE(sqlc.narg('instagram_url')::text, instagram_url),
    updated_at    = now()
WHERE id = @id
RETURNING *;

-- name: UpdateUserEmail :one
UPDATE users
SET email      = @email,
    updated_at = now()
WHERE id = @id
RETURNING *;

-- name: EnsureArtistSettings :exec
INSERT INTO artist_settings (artist_id)
VALUES ($1)
ON CONFLICT (artist_id) DO NOTHING;

-- name: GetArtistSettings :one
SELECT * FROM artist_settings WHERE artist_id = $1;

-- name: UpdateArtistSettings :one
UPDATE artist_settings
SET payout_frequency       = COALESCE(sqlc.narg('payout_frequency')::text,       payout_frequency),
    currency               = COALESCE(sqlc.narg('currency')::char(3),            currency),
    platform_fee_payer     = COALESCE(sqlc.narg('platform_fee_payer')::text,     platform_fee_payer),
    accepting_bookings     = COALESCE(sqlc.narg('accepting_bookings')::boolean,  accepting_bookings),
    timezone               = COALESCE(sqlc.narg('timezone')::text,               timezone),
    slot_interval_minutes  = COALESCE(sqlc.narg('slot_interval_minutes')::integer, slot_interval_minutes),
    buffer_minutes         = COALESCE(sqlc.narg('buffer_minutes')::integer,      buffer_minutes),
    min_notice_minutes     = COALESCE(sqlc.narg('min_notice_minutes')::integer,  min_notice_minutes),
    terms_text             = COALESCE(sqlc.narg('terms_text')::text,             terms_text),
    terms_show_on_booking  = COALESCE(sqlc.narg('terms_show_on_booking')::boolean,  terms_show_on_booking),
    terms_show_at_deposit  = COALESCE(sqlc.narg('terms_show_at_deposit')::boolean,  terms_show_at_deposit),
    waiver_required        = COALESCE(sqlc.narg('waiver_required')::boolean,     waiver_required),
    notify_by_email        = COALESCE(sqlc.narg('notify_by_email')::boolean,     notify_by_email),
    notify_by_sms          = COALESCE(sqlc.narg('notify_by_sms')::boolean,       notify_by_sms),
    deposit_refund_policy  = COALESCE(sqlc.narg('deposit_refund_policy')::text,  deposit_refund_policy),
    styles                 = COALESCE(sqlc.narg('styles')::text[],              styles),
    aftercare              = COALESCE(sqlc.narg('aftercare')::text,              aftercare),
    faqs                   = COALESCE(sqlc.narg('faqs')::jsonb,                  faqs),
    monthly_booking_goal   = COALESCE(sqlc.narg('monthly_booking_goal')::integer, monthly_booking_goal),

    -- Nullable fields support explicit clearing via a paired boolean.
    deposit_flat_fee_cents = CASE
                                 WHEN @clear_deposit_flat_fee::boolean THEN NULL
                                 ELSE COALESCE(sqlc.narg('deposit_flat_fee_cents')::bigint, deposit_flat_fee_cents)
                             END,
    max_advance_days       = CASE
                                 WHEN @clear_max_advance_days::boolean THEN NULL
                                 ELSE COALESCE(sqlc.narg('max_advance_days')::integer, max_advance_days)
                             END,
    cancellation_notice_hours = CASE
                                 WHEN @clear_cancellation_notice_hours::boolean THEN NULL
                                 ELSE COALESCE(sqlc.narg('cancellation_notice_hours')::integer, cancellation_notice_hours)
                             END,
    stripe_account_id      = CASE
                                 WHEN @clear_stripe_account::boolean THEN NULL
                                 ELSE COALESCE(sqlc.narg('stripe_account_id')::text, stripe_account_id)
                             END,
    google_calendar_email  = CASE
                                 WHEN @clear_google_calendar::boolean THEN NULL
                                 ELSE COALESCE(sqlc.narg('google_calendar_email')::text, google_calendar_email)
                             END,
    waiver_file_url        = CASE
                                 WHEN @clear_waiver_file::boolean THEN NULL
                                 ELSE COALESCE(sqlc.narg('waiver_file_url')::text, waiver_file_url)
                             END,
    updated_at             = now()
WHERE artist_id = @artist_id
RETURNING *;

-- name: SetStripeAccount :one
UPDATE artist_settings
SET stripe_account_id        = @stripe_account_id::text,
    stripe_charges_enabled   = false,
    stripe_payouts_enabled   = false,
    stripe_details_submitted = false,
    updated_at               = now()
WHERE artist_id = @artist_id
RETURNING *;

-- name: UpdateStripeAccountStatus :one
UPDATE artist_settings
SET stripe_charges_enabled   = @charges_enabled::boolean,
    stripe_payouts_enabled   = @payouts_enabled::boolean,
    stripe_details_submitted = @details_submitted::boolean,
    updated_at               = now()
WHERE artist_id = @artist_id
RETURNING *;

-- name: ClearStripeAccount :one
UPDATE artist_settings
SET stripe_account_id        = NULL,
    stripe_charges_enabled   = false,
    stripe_payouts_enabled   = false,
    stripe_details_submitted = false,
    updated_at               = now()
WHERE artist_id = @artist_id
RETURNING *;

-- name: GetArtistSettingsByStripeAccount :one
SELECT * FROM artist_settings WHERE stripe_account_id = $1;

-- name: SetGoogleCalendarConnection :one
UPDATE artist_settings
SET google_calendar_email         = @google_calendar_email::text,
    google_calendar_access_token  = @access_token::text,
    google_calendar_refresh_token = COALESCE(sqlc.narg('refresh_token')::text, google_calendar_refresh_token),
    google_calendar_token_expiry  = @token_expiry::timestamptz,
    updated_at                    = now()
WHERE artist_id = @artist_id
RETURNING *;

-- name: ClearGoogleCalendarConnection :one
UPDATE artist_settings
SET google_calendar_email         = NULL,
    google_calendar_access_token  = NULL,
    google_calendar_refresh_token = NULL,
    google_calendar_token_expiry  = NULL,
    updated_at                    = now()
WHERE artist_id = @artist_id
RETURNING *;

-- name: ListAvailabilityWindows :many
SELECT *
FROM artist_availability_windows
WHERE artist_id = $1
ORDER BY weekday, start_minute;

-- name: DeleteAllAvailabilityWindows :exec
DELETE FROM artist_availability_windows WHERE artist_id = $1;

-- name: CreateAvailabilityWindow :one
INSERT INTO artist_availability_windows (artist_id, weekday, start_minute, end_minute)
VALUES (@artist_id, @weekday, @start_minute, @end_minute)
RETURNING *;

-- name: ListSessionPresets :many
SELECT *
FROM artist_session_presets
WHERE artist_id = $1
ORDER BY position, created_at;

-- name: CreateSessionPreset :one
INSERT INTO artist_session_presets (artist_id, name, description, approx_duration_minutes, position)
VALUES (@artist_id, @name, @description, @approx_duration_minutes, @position)
RETURNING *;

-- name: UpdateSessionPreset :one
UPDATE artist_session_presets
SET name                    = COALESCE(sqlc.narg('name')::text,                  name),
    description             = COALESCE(sqlc.narg('description')::text,           description),
    approx_duration_minutes = COALESCE(sqlc.narg('approx_duration_minutes')::integer, approx_duration_minutes),
    position                = COALESCE(sqlc.narg('position')::integer,           position)
WHERE id = @id AND artist_id = @artist_id
RETURNING *;

-- name: DeleteSessionPreset :exec
DELETE FROM artist_session_presets
WHERE id = @id AND artist_id = @artist_id;

-- name: ListDaysOff :many
SELECT *
FROM artist_days_off
WHERE artist_id = $1
ORDER BY day;

-- name: AddDayOff :exec
INSERT INTO artist_days_off (artist_id, day)
VALUES (@artist_id, @day)
ON CONFLICT (artist_id, day) DO NOTHING;

-- name: RemoveDayOff :exec
DELETE FROM artist_days_off
WHERE artist_id = @artist_id AND day = @day;

-- name: ListBlocklist :many
SELECT *
FROM artist_blocklist
WHERE artist_id = $1
ORDER BY created_at DESC;

-- name: AddBlocklistEntry :one
INSERT INTO artist_blocklist (artist_id, email, phone, note)
VALUES (@artist_id, sqlc.narg('email')::citext, sqlc.narg('phone')::text, @note)
RETURNING *;

-- name: RemoveBlocklistEntry :exec
DELETE FROM artist_blocklist
WHERE id = @id AND artist_id = @artist_id;
