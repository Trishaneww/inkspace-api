CREATE TABLE users (
    id                UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    email             CITEXT      NOT NULL UNIQUE,
    password_hash     TEXT        NOT NULL,
    role              TEXT        NOT NULL CHECK (role IN ('artist', 'studio_admin', 'user')),
    email_verified_at TIMESTAMPTZ,
    created_at        TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at        TIMESTAMPTZ NOT NULL DEFAULT now(),
    first_name        TEXT,
    last_name         TEXT,
    phone             TEXT        NOT NULL,
    phone_verified_at TIMESTAMPTZ,

    username          CITEXT,
    avatar_url        TEXT,
    instagram_url     TEXT,

    marketing_opt_in_at     TIMESTAMPTZ,
    marketing_opt_in_source TEXT,

    -- How the account authenticates. An email belongs to exactly one provider;
    -- a second provider claiming the same email is rejected at sign-in.
    auth_provider     TEXT        NOT NULL DEFAULT 'password'
                      CHECK (auth_provider IN ('password', 'google', 'microsoft'))
);

CREATE INDEX idx_users_role ON users (role);

CREATE UNIQUE INDEX idx_users_phone_unique ON users (phone);

CREATE UNIQUE INDEX idx_users_username_unique
    ON users (username)
    WHERE username IS NOT NULL;

CREATE TABLE refresh_tokens (
    id          UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id     UUID        NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    token_hash  TEXT        NOT NULL UNIQUE,
    expires_at  TIMESTAMPTZ NOT NULL,
    revoked_at  TIMESTAMPTZ,
    user_agent  TEXT,
    ip_address  INET,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_refresh_tokens_user_id ON refresh_tokens (user_id);
CREATE INDEX idx_refresh_tokens_expires_at ON refresh_tokens (expires_at);
