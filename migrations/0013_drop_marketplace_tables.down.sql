-- Recreate the marketplace tables that 0013 up dropped. Faithful to the
-- previous schemas in deleted migrations 0005-0009. Tables are created
-- in dependency order (parents before children).

-- ===== bookings (was 0005) =====

CREATE TABLE bookings (
    id                   UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    client_id            UUID        NOT NULL REFERENCES users (id) ON DELETE RESTRICT,
    artist_id            UUID        NOT NULL REFERENCES artists (id) ON DELETE RESTRICT,
    status               TEXT        NOT NULL DEFAULT 'inquiry'
                         CHECK (status IN ('inquiry', 'consultation', 'deposit_held',
                                           'scheduled', 'completed', 'cancelled')),
    notes                TEXT        NOT NULL DEFAULT '',
    estimated_hours      NUMERIC(5,2),
    deposit_amount_cents BIGINT      NOT NULL DEFAULT 0 CHECK (deposit_amount_cents >= 0),
    scheduled_for        TIMESTAMPTZ,
    created_at           TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at           TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_bookings_client_id ON bookings (client_id, created_at DESC);
CREATE INDEX idx_bookings_artist_id ON bookings (artist_id, created_at DESC);
CREATE INDEX idx_bookings_status ON bookings (status);
CREATE INDEX idx_bookings_scheduled_for ON bookings (scheduled_for) WHERE scheduled_for IS NOT NULL;

CREATE TABLE booking_status_history (
    id          UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    booking_id  UUID        NOT NULL REFERENCES bookings (id) ON DELETE CASCADE,
    from_status TEXT,
    to_status   TEXT        NOT NULL,
    changed_by  UUID        REFERENCES users (id) ON DELETE SET NULL,
    note        TEXT,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_booking_status_history_booking_id ON booking_status_history (booking_id, created_at);

-- ===== matching (was 0006) =====

CREATE TABLE match_requests (
    id               UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    client_id        UUID        NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    description      TEXT        NOT NULL,
    reference_urls   TEXT[]      NOT NULL DEFAULT '{}',
    styles           TEXT[]      NOT NULL DEFAULT '{}',
    placement        TEXT,
    size_inches      NUMERIC(5,2),
    budget_min_cents BIGINT      CHECK (budget_min_cents IS NULL OR budget_min_cents >= 0),
    budget_max_cents BIGINT      CHECK (budget_max_cents IS NULL OR budget_max_cents >= 0),
    city             TEXT,
    country          TEXT,
    status           TEXT        NOT NULL DEFAULT 'open'
                     CHECK (status IN ('open', 'matched', 'converted', 'expired')),
    expires_at       TIMESTAMPTZ,
    created_at       TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_match_requests_client_id ON match_requests (client_id, created_at DESC);
CREATE INDEX idx_match_requests_status ON match_requests (status);
CREATE INDEX idx_match_requests_styles ON match_requests USING GIN (styles);

CREATE TABLE matches (
    id         UUID             PRIMARY KEY DEFAULT gen_random_uuid(),
    request_id UUID             NOT NULL REFERENCES match_requests (id) ON DELETE CASCADE,
    artist_id  UUID             NOT NULL REFERENCES artists (id) ON DELETE CASCADE,
    score      DOUBLE PRECISION NOT NULL DEFAULT 0,
    status     TEXT             NOT NULL DEFAULT 'pending'
               CHECK (status IN ('pending', 'interested', 'passed')),
    created_at TIMESTAMPTZ      NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ      NOT NULL DEFAULT now(),
    UNIQUE (request_id, artist_id)
);

CREATE INDEX idx_matches_request_id ON matches (request_id);
CREATE INDEX idx_matches_artist_id_status ON matches (artist_id, status);

-- ===== messaging (was 0007) =====

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

CREATE UNIQUE INDEX idx_conversations_unique_no_booking
    ON conversations (client_id, artist_id)
    WHERE booking_id IS NULL;

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

-- ===== payments (was 0008) =====

CREATE TABLE payments (
    id                 UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    booking_id         UUID        NOT NULL REFERENCES bookings (id) ON DELETE RESTRICT,
    payer_id           UUID        NOT NULL REFERENCES users (id) ON DELETE RESTRICT,
    payee_id           UUID        NOT NULL REFERENCES users (id) ON DELETE RESTRICT,
    kind               TEXT        NOT NULL CHECK (kind IN ('deposit', 'balance')),
    amount_cents       BIGINT      NOT NULL CHECK (amount_cents > 0),
    currency           TEXT        NOT NULL CHECK (char_length(currency) = 3),
    status             TEXT        NOT NULL DEFAULT 'pending'
                       CHECK (status IN ('pending', 'succeeded', 'failed', 'refunded')),
    provider           TEXT        NOT NULL,
    provider_intent_id TEXT        NOT NULL,
    created_at         TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at         TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (provider, provider_intent_id)
);

CREATE INDEX idx_payments_booking_id ON payments (booking_id);
CREATE INDEX idx_payments_payer_id ON payments (payer_id, created_at DESC);
CREATE INDEX idx_payments_payee_id ON payments (payee_id, created_at DESC);
CREATE INDEX idx_payments_status ON payments (status);

CREATE TABLE payment_webhook_events (
    id          UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    provider    TEXT        NOT NULL,
    event_id    TEXT        NOT NULL,
    payload     JSONB       NOT NULL,
    processed   BOOLEAN     NOT NULL DEFAULT false,
    received_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (provider, event_id)
);

CREATE INDEX idx_payment_webhook_events_unprocessed
    ON payment_webhook_events (received_at)
    WHERE processed = false;

-- ===== notifications (was 0009) =====

CREATE TABLE notifications (
    id         UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id    UUID        NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    kind       TEXT        NOT NULL,
    title      TEXT        NOT NULL,
    body       TEXT        NOT NULL DEFAULT '',
    data       JSONB       NOT NULL DEFAULT '{}'::jsonb,
    read_at    TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_notifications_user_id_created
    ON notifications (user_id, created_at DESC);

CREATE INDEX idx_notifications_unread
    ON notifications (user_id, created_at DESC)
    WHERE read_at IS NULL;

CREATE TABLE notification_preferences (
    user_id     UUID        PRIMARY KEY REFERENCES users (id) ON DELETE CASCADE,
    enabled     JSONB       NOT NULL DEFAULT '{}'::jsonb,
    quiet_hours JSONB,
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE notification_deliveries (
    id              UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    notification_id UUID        NOT NULL REFERENCES notifications (id) ON DELETE CASCADE,
    channel         TEXT        NOT NULL CHECK (channel IN ('in_app', 'email', 'push')),
    status          TEXT        NOT NULL DEFAULT 'pending'
                    CHECK (status IN ('pending', 'sent', 'failed', 'skipped')),
    error           TEXT,
    attempted_at    TIMESTAMPTZ,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_notification_deliveries_notification_id
    ON notification_deliveries (notification_id);

CREATE INDEX idx_notification_deliveries_pending
    ON notification_deliveries (created_at)
    WHERE status = 'pending';
