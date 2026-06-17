-- name: CreateUser :one
INSERT INTO users (email, password_hash, role, first_name, last_name, phone, username, instagram_url)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
RETURNING *;

-- name: GetUserByID :one
SELECT * FROM users WHERE id = $1;

-- name: GetUserByEmail :one
SELECT * FROM users WHERE email = $1;

-- name: GetUserByPhone :one
SELECT * FROM users WHERE phone = $1;

-- name: UpdateUserPassword :exec
UPDATE users
SET password_hash = $2,
    updated_at = now()
WHERE id = $1;

-- name: UpdateUnverifiedUser :one
UPDATE users
SET password_hash = $2,
    role          = $3,
    first_name    = $4,
    last_name     = $5,
    phone         = $6,
    username      = $7,
    instagram_url = $8,
    updated_at    = now()
WHERE id = $1 AND phone_verified_at IS NULL
RETURNING *;

-- name: MarkEmailVerified :exec
UPDATE users
SET email_verified_at = now(),
    updated_at = now()
WHERE id = $1 AND email_verified_at IS NULL;

-- name: MarkPhoneVerified :exec
UPDATE users
SET phone_verified_at = now(),
    updated_at = now()
WHERE id = $1 AND phone_verified_at IS NULL;

-- name: DeleteUser :exec
DELETE FROM users WHERE id = $1;

-- name: CreateRefreshToken :one
INSERT INTO refresh_tokens (user_id, token_hash, expires_at, user_agent, ip_address)
VALUES ($1, $2, $3, $4, $5)
RETURNING *;

-- name: GetActiveRefreshTokenByHash :one
SELECT * FROM refresh_tokens
WHERE token_hash = $1
  AND revoked_at IS NULL
  AND expires_at > now();

-- name: RevokeRefreshToken :exec
UPDATE refresh_tokens
SET revoked_at = now()
WHERE token_hash = $1;

-- name: RevokeAllRefreshTokensForUser :exec
UPDATE refresh_tokens
SET revoked_at = now()
WHERE user_id = $1 AND revoked_at IS NULL;

-- name: DeleteExpiredRefreshTokens :execrows
DELETE FROM refresh_tokens
WHERE expires_at < now() - INTERVAL '7 days';

-- name: SetUserMarketingOptIn :exec
UPDATE users
SET marketing_opt_in_at = now(),
    marketing_opt_in_source = @source,
    updated_at = now()
WHERE id = @id;
