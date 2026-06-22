CREATE TABLE payment_requests (
    id                 UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    artist_id          UUID NOT NULL REFERENCES artists (id) ON DELETE CASCADE,
    booking_request_id UUID NOT NULL REFERENCES booking_requests (id) ON DELETE CASCADE,

    type   TEXT NOT NULL CHECK (type IN ('deposit', 'final')),
    status TEXT NOT NULL DEFAULT 'requested'
           CHECK (status IN ('requested', 'processing', 'paid', 'failed', 'canceled', 'expired', 'refunded')),

    currency CHAR(3) NOT NULL,

    amount_cents          BIGINT NOT NULL CHECK (amount_cents > 0),
    platform_fee_cents    BIGINT NOT NULL CHECK (platform_fee_cents >= 0),
    client_charge_cents   BIGINT NOT NULL CHECK (client_charge_cents > 0),
    fee_payer             TEXT   NOT NULL CHECK (fee_payer IN ('artist', 'client', 'split')),
    amount_refunded_cents BIGINT NOT NULL DEFAULT 0 CHECK (amount_refunded_cents >= 0),

    client_email TEXT NOT NULL,
    client_name  TEXT NOT NULL DEFAULT '',
    description  TEXT NOT NULL DEFAULT '',

    public_token               TEXT NOT NULL UNIQUE,
    stripe_checkout_session_id TEXT,
    stripe_payment_intent_id   TEXT,

    expires_at  TIMESTAMPTZ NOT NULL,
    paid_at     TIMESTAMPTZ,
    canceled_at TIMESTAMPTZ,

    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),

    last_emailed_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    reminder_sent_at TIMESTAMPTZ,
    -- Client-chosen session start carried until the deposit is paid (then the
    -- appointment is scheduled at this time).
    scheduled_start  TIMESTAMPTZ
);

CREATE INDEX idx_payment_requests_booking
    ON payment_requests (booking_request_id, created_at DESC);

CREATE INDEX idx_payment_requests_artist
    ON payment_requests (artist_id, status);

CREATE UNIQUE INDEX idx_payment_requests_payment_intent
    ON payment_requests (stripe_payment_intent_id)
    WHERE stripe_payment_intent_id IS NOT NULL;

CREATE UNIQUE INDEX idx_payment_requests_checkout_session
    ON payment_requests (stripe_checkout_session_id)
    WHERE stripe_checkout_session_id IS NOT NULL;

CREATE TABLE stripe_events (
    id          TEXT PRIMARY KEY,
    type        TEXT NOT NULL,
    received_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
