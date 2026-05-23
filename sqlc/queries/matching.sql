-- name: CreateMatchRequest :one
INSERT INTO match_requests (
    client_id, description, reference_urls, styles, placement,
    size_inches, budget_min_cents, budget_max_cents, city, country, expires_at
)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
RETURNING *;

-- name: GetMatchRequestByID :one
SELECT * FROM match_requests WHERE id = $1;

-- name: ListMatchRequestsByClient :many
SELECT * FROM match_requests
WHERE client_id = @client_id
  AND (sqlc.narg('status')::text IS NULL OR status = sqlc.narg('status'))
ORDER BY created_at DESC
LIMIT @lim
OFFSET @off;

-- name: UpdateMatchRequestStatus :exec
UPDATE match_requests
SET status = $2
WHERE id = $1;

-- name: ExpireOpenMatchRequests :execrows
UPDATE match_requests
SET status = 'expired'
WHERE status = 'open'
  AND expires_at IS NOT NULL
  AND expires_at < now();

-- name: InsertMatch :one
INSERT INTO matches (request_id, artist_id, score)
VALUES ($1, $2, $3)
ON CONFLICT (request_id, artist_id) DO UPDATE
SET score      = EXCLUDED.score,
    updated_at = now()
RETURNING *;

-- name: GetMatchByID :one
SELECT * FROM matches WHERE id = $1;

-- name: ListMatchesByRequest :many
SELECT * FROM matches
WHERE request_id = $1
ORDER BY score DESC, created_at ASC;

-- name: ListLeadsForArtist :many
SELECT
    sqlc.embed(matches),
    mr.client_id      AS request_client_id,
    mr.description    AS request_description,
    mr.styles         AS request_styles,
    mr.placement      AS request_placement,
    mr.budget_min_cents AS request_budget_min_cents,
    mr.budget_max_cents AS request_budget_max_cents,
    mr.city           AS request_city,
    mr.country        AS request_country
FROM matches
JOIN match_requests mr ON mr.id = matches.request_id
WHERE matches.artist_id = @artist_id
  AND (sqlc.narg('status')::text IS NULL OR matches.status = sqlc.narg('status'))
ORDER BY matches.created_at DESC
LIMIT @lim
OFFSET @off;

-- name: UpdateMatchStatus :one
UPDATE matches
SET status     = $2,
    updated_at = now()
WHERE id = $1
RETURNING *;
