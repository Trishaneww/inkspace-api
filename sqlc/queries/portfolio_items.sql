-- name: CreatePortfolioItem :one
INSERT INTO portfolio_items (
    artist_id,
    status,
    title,
    description,
    completion_date,
    image_keys,
    styles,
    placement,
    color_type,
    approx_size_inches,
    healed,
    session_count,
    total_minutes,
    published_at
)
VALUES (
    @artist_id,
    @status,
    @title,
    sqlc.narg('description')::text,
    sqlc.narg('completion_date')::date,
    @image_keys::text[],
    @styles::text[],
    sqlc.narg('placement')::text,
    sqlc.narg('color_type')::text,
    sqlc.narg('approx_size_inches')::integer,
    @healed,
    sqlc.narg('session_count')::integer,
    sqlc.narg('total_minutes')::integer,
    sqlc.narg('published_at')::timestamptz
)
RETURNING *;

-- name: GetPortfolioItem :one
SELECT * FROM portfolio_items WHERE id = $1;

-- name: ListPortfolioItemsByArtist :many
SELECT *
FROM portfolio_items
WHERE artist_id = @artist_id
  AND (sqlc.narg('status')::text IS NULL OR status = sqlc.narg('status'))
ORDER BY created_at DESC
LIMIT @lim
OFFSET @off;

-- name: CountPortfolioItemsByArtist :one
SELECT
    COUNT(*) FILTER (
        WHERE sqlc.narg('status')::text IS NULL OR status = sqlc.narg('status')
    )                                              AS total,
    COUNT(*) FILTER (WHERE status = 'published')   AS published
FROM portfolio_items
WHERE artist_id = @artist_id;

-- name: UpdatePortfolioItem :one
UPDATE portfolio_items
SET title              = @title,
    description        = sqlc.narg('description')::text,
    completion_date    = sqlc.narg('completion_date')::date,
    image_keys         = @image_keys::text[],
    styles             = @styles::text[],
    placement          = sqlc.narg('placement')::text,
    color_type         = sqlc.narg('color_type')::text,
    approx_size_inches = sqlc.narg('approx_size_inches')::integer,
    healed             = @healed,
    session_count      = sqlc.narg('session_count')::integer,
    total_minutes      = sqlc.narg('total_minutes')::integer,
    updated_at         = now()
WHERE id = @id
RETURNING *;

-- name: PublishPortfolioItem :one
UPDATE portfolio_items
SET status       = 'published',
    published_at = COALESCE(published_at, now()),
    updated_at   = now()
WHERE id = @id
  AND status = 'draft'
RETURNING *;

-- name: ArchivePortfolioItem :one
UPDATE portfolio_items
SET status      = 'archived',
    archived_at = now(),
    updated_at  = now()
WHERE id = @id
  AND status <> 'archived'
RETURNING *;

-- name: UnarchivePortfolioItem :one
UPDATE portfolio_items
SET status      = CASE
                      WHEN published_at IS NOT NULL THEN 'published'
                      ELSE 'draft'
                  END,
    archived_at = NULL,
    updated_at  = now()
WHERE id = @id
  AND status = 'archived'
RETURNING *;

-- name: DeletePortfolioItem :exec
DELETE FROM portfolio_items WHERE id = $1;
