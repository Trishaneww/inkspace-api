CREATE TABLE conversations (
    id                  UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    artist_id           UUID        NOT NULL REFERENCES artists (id) ON DELETE CASCADE,
    booking_request_id  UUID        NOT NULL UNIQUE REFERENCES booking_requests (id) ON DELETE CASCADE,

    client_name         TEXT        NOT NULL DEFAULT '',
    client_email        CITEXT      NOT NULL,
    -- Magic-link token granting a guest client read/reply access before they
    -- have an account (mirrors booking_requests.schedule_token).
    client_access_token TEXT        NOT NULL UNIQUE,

    status              TEXT        NOT NULL DEFAULT 'open'
                        CHECK (status IN ('open', 'archived')),

    last_message_at     TIMESTAMPTZ,
    artist_last_read_at TIMESTAMPTZ,
    client_last_read_at TIMESTAMPTZ,

    created_at          TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at          TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_conversations_artist
    ON conversations (artist_id, last_message_at DESC NULLS LAST);

CREATE TABLE messages (
    id              UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    conversation_id UUID        NOT NULL REFERENCES conversations (id) ON DELETE CASCADE,

    sender_type     TEXT        NOT NULL CHECK (sender_type IN ('artist', 'client', 'system')),
    sender_user_id  UUID        REFERENCES users (id) ON DELETE SET NULL,

    body            TEXT        NOT NULL,
    ai_drafted      BOOLEAN     NOT NULL DEFAULT false,

    created_at      TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_messages_conversation
    ON messages (conversation_id, created_at);
