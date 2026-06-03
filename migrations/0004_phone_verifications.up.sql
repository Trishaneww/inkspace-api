CREATE TABLE phone_verifications (
    id          UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id     UUID        NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    phone       TEXT        NOT NULL,
    code_hash   TEXT        NOT NULL,
    attempts    INT         NOT NULL DEFAULT 0,
    expires_at  TIMESTAMPTZ NOT NULL,
    consumed_at TIMESTAMPTZ,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_phone_verifications_user_id
    ON phone_verifications (user_id);

CREATE INDEX idx_phone_verifications_expires_at
    ON phone_verifications (expires_at);
