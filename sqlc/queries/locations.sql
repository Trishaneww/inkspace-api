-- name: CreateArtistLocation :one
INSERT INTO artist_locations (
    artist_id, label, address, city, province, postal_code, country,
    timezone, is_primary, start_date, end_date
) VALUES (
    @artist_id, @label, @address, @city, @province, @postal_code, @country,
    @timezone, @is_primary, sqlc.narg('start_date'), sqlc.narg('end_date')
)
RETURNING *;

-- name: ListArtistLocations :many
-- Active locations only — what the artist can currently work from and what
-- clients can book. Closed guest spots are excluded.
SELECT *
FROM artist_locations
WHERE artist_id = $1 AND status = 'active'
ORDER BY is_primary DESC, start_date ASC NULLS LAST, created_at;

-- name: ListAllArtistLocations :many
-- Every location including closed ones, so a booking request can always
-- resolve the spot it was tagged with even after the artist closes it.
SELECT *
FROM artist_locations
WHERE artist_id = $1
ORDER BY is_primary DESC, start_date ASC NULLS LAST, created_at;

-- name: GetArtistLocation :one
SELECT *
FROM artist_locations
WHERE id = $1 AND artist_id = $2;

-- name: GetPrimaryLocation :one
SELECT *
FROM artist_locations
WHERE artist_id = $1 AND is_primary
LIMIT 1;

-- name: UpdateArtistLocation :one
UPDATE artist_locations
SET label       = COALESCE(sqlc.narg('label')::text,       label),
    address     = COALESCE(sqlc.narg('address')::text,     address),
    city        = COALESCE(sqlc.narg('city')::text,        city),
    province    = COALESCE(sqlc.narg('province')::text,    province),
    postal_code = COALESCE(sqlc.narg('postal_code')::text, postal_code),
    country     = COALESCE(sqlc.narg('country')::text,     country),
    timezone    = COALESCE(sqlc.narg('timezone')::text,    timezone),
    start_date  = CASE
                      WHEN @clear_dates::boolean THEN NULL
                      ELSE COALESCE(sqlc.narg('start_date')::date, start_date)
                  END,
    end_date    = CASE
                      WHEN @clear_dates::boolean THEN NULL
                      ELSE COALESCE(sqlc.narg('end_date')::date, end_date)
                  END,
    updated_at  = now()
WHERE id = @id AND artist_id = @artist_id
RETURNING *;

-- name: CloseArtistLocation :exec
-- Soft-delete a guest spot: mark it closed but keep the row so booking
-- requests retain their location. The home studio can't be closed.
UPDATE artist_locations
SET status     = 'closed',
    closed_at  = now(),
    updated_at = now()
WHERE id = @id AND artist_id = @artist_id AND NOT is_primary AND status = 'active';

-- name: SetCurrentLocation :exec
UPDATE artist_settings
SET current_location_id = sqlc.narg('current_location_id')::uuid,
    updated_at          = now()
WHERE artist_id = @artist_id;
