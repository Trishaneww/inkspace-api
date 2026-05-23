CREATE TABLE conversations (
    id              UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    client_id       UUID        NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    artist_id       UUID        NOT NULL REFERENCES artists (id) ON DELETE CASCADE,
    booking_id      UUID        REFERENCES bookings (id) ON DELETE SET NULL,
    last_message_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_conversations_client_id ON conversations (client_id, last_message_at DESC);
CREATE INDEX idx_conversations_artist_id ON conversations (artist_id, last_message_at DESC);

-- One general conversation per (client, artist) when no booking is attached.
CREATE UNIQUE INDEX idx_conversations_unique_no_booking
    ON conversations (client_id, artist_id)
    WHERE booking_id IS NULL;

-- One conversation per (client, artist, booking) when a booking is attached.
CREATE UNIQUE INDEX idx_conversations_unique_with_booking
    ON conversations (client_id, artist_id, booking_id)
    WHERE booking_id IS NOT NULL;

CREATE TABLE messages (
    id              UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    conversation_id UUID        NOT NULL REFERENCES conversations (id) ON DELETE CASCADE,
    sender_id       UUID        NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    body            TEXT        NOT NULL CHECK (length(body) > 0),
    attachments     TEXT[]      NOT NULL DEFAULT '{}',
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_messages_conversation_id ON messages (conversation_id, created_at);

CREATE TABLE conversation_read_cursors (
    conversation_id UUID        NOT NULL REFERENCES conversations (id) ON DELETE CASCADE,
    user_id         UUID        NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    last_read_at    TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (conversation_id, user_id)
);
