-- name: CreateOpenBook :one
INSERT INTO open_books (artist_id, slug, scheduling_mode)
VALUES ($1, $2, $3)
RETURNING *;

-- name: GetOpenBookByArtist :one
SELECT * FROM open_books WHERE artist_id = $1;

-- name: GetOpenBookBySlug :one
SELECT * FROM open_books WHERE slug = $1;

-- name: UpdateOpenBook :one
UPDATE open_books
SET slug             = COALESCE(sqlc.narg('slug')::citext, slug),
    scheduling_mode  = COALESCE(sqlc.narg('scheduling_mode')::text, scheduling_mode),
    custom_questions = COALESCE(sqlc.narg('custom_questions')::jsonb, custom_questions),
    theme            = COALESCE(sqlc.narg('theme')::text, theme),
    updated_at       = now()
WHERE artist_id = @artist_id
RETURNING *;
