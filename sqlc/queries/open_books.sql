-- name: CreateOpenBook :one
INSERT INTO open_books (artist_id, slug, scheduling_mode)
VALUES ($1, $2, $3)
RETURNING *;

-- name: GetOpenBookByArtist :one
SELECT * FROM open_books WHERE artist_id = $1;

-- name: GetOpenBookBySlug :one
SELECT * FROM open_books WHERE slug = $1;
