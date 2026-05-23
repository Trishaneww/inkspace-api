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
