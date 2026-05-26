-- name: CreatePhoneVerification :one
INSERT INTO phone_verifications (user_id, phone, code_hash, expires_at)
VALUES ($1, $2, $3, $4)
RETURNING *;

-- name: GetActivePhoneVerification :one
SELECT * FROM phone_verifications
WHERE id = $1
  AND consumed_at IS NULL
  AND expires_at > now();

-- name: IncrementPhoneVerificationAttempts :exec
UPDATE phone_verifications
SET attempts = attempts + 1
WHERE id = $1;

-- name: RefreshPhoneVerificationCode :exec
-- Used when the user requests a resend. Rotates the code on the same
-- verification record (preserving its id, so the frontend's stored
-- verificationId keeps working) and resets the attempt counter.
UPDATE phone_verifications
SET code_hash = $2,
    expires_at = $3,
    attempts = 0
WHERE id = $1 AND consumed_at IS NULL;

-- name: ConsumePhoneVerification :exec
UPDATE phone_verifications
SET consumed_at = now()
WHERE id = $1 AND consumed_at IS NULL;

-- name: RevokeActivePhoneVerificationsForUser :exec
UPDATE phone_verifications
SET consumed_at = now()
WHERE user_id = $1 AND consumed_at IS NULL;

-- name: DeleteExpiredPhoneVerifications :execrows
DELETE FROM phone_verifications
WHERE expires_at < now() - INTERVAL '24 hours';
