-- name: CreatePortfolioItem :one
INSERT INTO portfolio_items (artist_id, image_url, caption, styles, placement, healed_photo)
VALUES ($1, $2, $3, $4, $5, $6)
RETURNING *;

-- name: GetPortfolioItem :one
SELECT * FROM portfolio_items WHERE id = $1;

-- name: ListPortfolioItemsByArtist :many
SELECT * FROM portfolio_items
WHERE artist_id = $1
ORDER BY created_at DESC
LIMIT $2 OFFSET $3;

-- name: UpdatePortfolioItem :one
UPDATE portfolio_items
SET caption      = COALESCE(sqlc.narg('caption')::text, caption),
    styles       = COALESCE(sqlc.narg('styles')::text[], styles),
    placement    = COALESCE(sqlc.narg('placement')::text, placement),
    healed_photo = COALESCE(sqlc.narg('healed_photo')::boolean, healed_photo)
WHERE id = @id
RETURNING *;

-- name: DeletePortfolioItem :exec
DELETE FROM portfolio_items WHERE id = $1;

-- name: CreateFlashDesign :one
INSERT INTO flash_designs (
    artist_id, title, description, image_url, styles, placement,
    size_inches, price_cents, currency, repeatable
)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
RETURNING *;

-- name: GetFlashDesign :one
SELECT * FROM flash_designs WHERE id = $1;

-- name: ListFlashDesigns :many
SELECT * FROM flash_designs
WHERE (sqlc.narg('status')::text IS NULL OR status = sqlc.narg('status'))
  AND (cardinality(@styles::text[]) = 0 OR styles && @styles::text[])
  AND (sqlc.narg('artist_id')::uuid IS NULL OR artist_id = sqlc.narg('artist_id'))
ORDER BY created_at DESC
LIMIT @lim
OFFSET @off;

-- name: UpdateFlashDesign :one
UPDATE flash_designs
SET title       = COALESCE(sqlc.narg('title')::text, title),
    description = COALESCE(sqlc.narg('description')::text, description),
    image_url   = COALESCE(sqlc.narg('image_url')::text, image_url),
    styles      = COALESCE(sqlc.narg('styles')::text[], styles),
    placement   = COALESCE(sqlc.narg('placement')::text, placement),
    size_inches = COALESCE(sqlc.narg('size_inches')::numeric, size_inches),
    price_cents = COALESCE(sqlc.narg('price_cents')::bigint, price_cents),
    currency    = COALESCE(sqlc.narg('currency')::text, currency),
    repeatable  = COALESCE(sqlc.narg('repeatable')::boolean, repeatable),
    updated_at  = now()
WHERE id = @id
RETURNING *;

-- name: DeleteFlashDesign :exec
DELETE FROM flash_designs WHERE id = $1;

-- name: ReserveFlashDesign :one
UPDATE flash_designs
SET status         = 'reserved',
    reserved_by    = @user_id,
    reserved_until = @reserved_until,
    updated_at     = now()
WHERE id = @id
  AND status = 'available'
RETURNING *;

-- name: ReleaseFlashReservation :one
UPDATE flash_designs
SET status         = 'available',
    reserved_by    = NULL,
    reserved_until = NULL,
    updated_at     = now()
WHERE id = $1
  AND status = 'reserved'
RETURNING *;

-- name: ClaimFlashDesign :one
UPDATE flash_designs
SET status     = 'claimed',
    updated_at = now()
WHERE id = $1
  AND status IN ('available', 'reserved')
RETURNING *;
