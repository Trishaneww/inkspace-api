-- Tracks in-flight phone verification codes.
-- A challenge is created on signup / login (when MFA is required) and consumed
-- when the user submits the correct code, after which a session token is issued.
CREATE TABLE phone_verification_challenges (
    id          UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id     UUID        NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    phone       TEXT        NOT NULL,
    code_hash   TEXT        NOT NULL,
    attempts    INT         NOT NULL DEFAULT 0,
    expires_at  TIMESTAMPTZ NOT NULL,
    consumed_at TIMESTAMPTZ,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_phone_challenges_user_id
    ON phone_verification_challenges (user_id);

CREATE INDEX idx_phone_challenges_expires_at
    ON phone_verification_challenges (expires_at);
