-- name: CreateArtist :one
INSERT INTO artists (user_id, handle, display_name, bio, styles, city, country, booking_link)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
RETURNING *;

-- name: GetArtistByID :one
SELECT * FROM artists WHERE id = $1;

-- name: GetArtistByHandle :one
SELECT * FROM artists WHERE handle = $1;

-- name: GetArtistByUserID :one
SELECT * FROM artists WHERE user_id = $1;

-- name: ListArtists :many
SELECT * FROM artists
WHERE (sqlc.narg('city')::text IS NULL OR city = sqlc.narg('city'))
  AND (sqlc.narg('country')::text IS NULL OR country = sqlc.narg('country'))
  AND (cardinality(@styles::text[]) = 0 OR styles && @styles::text[])
ORDER BY created_at DESC
LIMIT @lim
OFFSET @off;

-- name: UpdateArtist :one
UPDATE artists
SET display_name = COALESCE(sqlc.narg('display_name')::text, display_name),
    bio          = COALESCE(sqlc.narg('bio')::text, bio),
    styles       = COALESCE(sqlc.narg('styles')::text[], styles),
    city         = COALESCE(sqlc.narg('city')::text, city),
    country      = COALESCE(sqlc.narg('country')::text, country),
    booking_link = COALESCE(sqlc.narg('booking_link')::text, booking_link),
    updated_at   = now()
WHERE id = @id
RETURNING *;

-- name: DeleteArtist :exec
DELETE FROM artists WHERE id = $1;

-- name: UpsertArtistAvailability :one
INSERT INTO artist_availability (artist_id, accepting_new, weekly_windows, waitlist_days)
VALUES ($1, $2, $3, $4)
ON CONFLICT (artist_id) DO UPDATE
SET accepting_new  = EXCLUDED.accepting_new,
    weekly_windows = EXCLUDED.weekly_windows,
    waitlist_days  = EXCLUDED.waitlist_days,
    updated_at     = now()
RETURNING *;

-- name: GetArtistAvailability :one
SELECT * FROM artist_availability WHERE artist_id = $1;
