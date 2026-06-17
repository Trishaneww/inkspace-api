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
