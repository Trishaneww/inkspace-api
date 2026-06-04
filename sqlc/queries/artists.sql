-- Provision an artist profile for a user. Idempotent: a no-op if the user
-- already has one, so it's safe to call on activation and as a lazy net.
-- name: EnsureArtist :exec
INSERT INTO artists (user_id)
VALUES ($1)
ON CONFLICT (user_id) DO NOTHING;

-- name: GetArtistByUserID :one
SELECT * FROM artists WHERE user_id = $1;
