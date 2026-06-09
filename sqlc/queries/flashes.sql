-- name: CreateFlash :one
INSERT INTO flashes (
    artist_id,
    status,
    title,
    description,
    s3_key,
    reference_s3_key,
    color_type,
    styles,
    placements,
    pricing_mode,
    flat_price_cents,
    flat_duration_minutes,
    deposit_cents,
    repeatable,
    published_at
)
VALUES (
    @artist_id,
    @status,
    @title,
    sqlc.narg('description')::text,
    sqlc.narg('s3_key')::text,
    sqlc.narg('reference_s3_key')::text,
    @color_type,
    @styles::text[],
    @placements::text[],
    @pricing_mode,
    sqlc.narg('flat_price_cents')::bigint,
    sqlc.narg('flat_duration_minutes')::integer,
    sqlc.narg('deposit_cents')::bigint,
    @repeatable,
    sqlc.narg('published_at')::timestamptz
)
RETURNING *;

-- name: GetFlash :one
SELECT * FROM flashes WHERE id = $1;

-- name: ListFlashesByArtist :many
SELECT *
FROM flashes
WHERE artist_id = @artist_id
  AND (sqlc.narg('status')::text IS NULL OR status = sqlc.narg('status'))
ORDER BY created_at DESC
LIMIT @page_limit
OFFSET @page_offset;

-- name: CountFlashesByArtist :one
SELECT
    COUNT(*) FILTER (
        WHERE sqlc.narg('status')::text IS NULL OR status = sqlc.narg('status')
    )                                                     AS total,
    COUNT(*) FILTER (WHERE status = 'available')          AS available
FROM flashes
WHERE artist_id = @artist_id;

-- name: UpdateFlash :one
UPDATE flashes
SET title                 = COALESCE(sqlc.narg('title')::text,                          title),
    description           = COALESCE(sqlc.narg('description')::text,                    description),
    s3_key                = COALESCE(sqlc.narg('s3_key')::text,                         s3_key),
    reference_s3_key      = CASE
                                WHEN @clear_reference_s3_key::boolean THEN NULL
                                ELSE COALESCE(sqlc.narg('reference_s3_key')::text, reference_s3_key)
                            END,
    color_type            = COALESCE(sqlc.narg('color_type')::text,                     color_type),
    styles                = COALESCE(sqlc.narg('styles')::text[],                       styles),
    placements            = COALESCE(sqlc.narg('placements')::text[],                   placements),
    pricing_mode          = COALESCE(sqlc.narg('pricing_mode')::text,                   pricing_mode),
    flat_price_cents      = COALESCE(sqlc.narg('flat_price_cents')::bigint,             flat_price_cents),
    flat_duration_minutes = COALESCE(sqlc.narg('flat_duration_minutes')::integer,       flat_duration_minutes),
    deposit_cents         = COALESCE(sqlc.narg('deposit_cents')::bigint,                deposit_cents),
    repeatable            = COALESCE(sqlc.narg('repeatable')::boolean,                  repeatable),
    updated_at            = now()
WHERE id = @id
RETURNING *;

-- name: PublishFlash :one
UPDATE flashes
SET status       = 'available',
    published_at = COALESCE(published_at, now()),
    updated_at   = now()
WHERE id = @id
  AND status = 'draft'
RETURNING *;

-- name: ClaimFlash :one
UPDATE flashes
SET status                = 'claimed',
    claimed_at            = now(),
    claimed_by_booking_id = @claimed_by_booking_id,
    updated_at            = now()
WHERE id = @id
  AND status = 'available'
RETURNING *;

-- name: ArchiveFlash :one
UPDATE flashes
SET status      = 'archived',
    archived_at = now(),
    updated_at  = now()
WHERE id = @id
  AND status <> 'archived'
RETURNING *;

-- name: UnarchiveFlash :one
UPDATE flashes
SET status      = CASE
                      WHEN published_at IS NOT NULL THEN 'available'
                      ELSE 'draft'
                  END,
    archived_at = NULL,
    updated_at  = now()
WHERE id = @id
  AND status = 'archived'
RETURNING *;

-- name: DeleteFlash :exec
DELETE FROM flashes WHERE id = $1;

-- name: IncrementFlashViewCount :exec
UPDATE flashes
SET view_count = view_count + 1
WHERE id = $1;

-- name: ListFlashPricingTiers :many
SELECT *
FROM flash_pricing_tiers
WHERE flash_id = $1
ORDER BY
    CASE size_code
        WHEN 'x_small' THEN 1
        WHEN 'small'   THEN 2
        WHEN 'medium'  THEN 3
        WHEN 'large'   THEN 4
        WHEN 'x_large' THEN 5
    END;

-- name: ListFlashPricingTiersForFlashes :many
-- Batched fetch for a page of flashes, avoiding an N+1 of ListFlashPricingTiers.
-- Ordered by flash so callers can group sequentially, then by size.
SELECT *
FROM flash_pricing_tiers
WHERE flash_id = ANY(@flash_ids::uuid[])
ORDER BY
    flash_id,
    CASE size_code
        WHEN 'x_small' THEN 1
        WHEN 'small'   THEN 2
        WHEN 'medium'  THEN 3
        WHEN 'large'   THEN 4
        WHEN 'x_large' THEN 5
    END;

-- name: UpsertFlashPricingTier :one
INSERT INTO flash_pricing_tiers (flash_id, size_code, duration_minutes, price_cents)
VALUES (@flash_id, @size_code, @duration_minutes, @price_cents)
ON CONFLICT (flash_id, size_code) DO UPDATE
SET duration_minutes = EXCLUDED.duration_minutes,
    price_cents      = EXCLUDED.price_cents
RETURNING *;

-- name: DeleteFlashPricingTier :exec
DELETE FROM flash_pricing_tiers
WHERE flash_id = $1 AND size_code = $2;

-- name: DeleteAllFlashPricingTiers :exec
DELETE FROM flash_pricing_tiers WHERE flash_id = $1;
