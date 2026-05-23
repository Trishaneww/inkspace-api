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
