CREATE TABLE appointments (
    id                 UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    artist_id          UUID        NOT NULL REFERENCES artists (id) ON DELETE CASCADE,
    booking_request_id UUID        NOT NULL REFERENCES booking_requests (id) ON DELETE CASCADE,

    type               TEXT        NOT NULL CHECK (type IN ('consultation', 'session')),
    status             TEXT        NOT NULL DEFAULT 'proposed'
                       CHECK (status IN ('proposed', 'scheduled', 'completed', 'cancelled', 'no_show')),

    scheduled_start    TIMESTAMPTZ,
    duration_minutes   INTEGER     NOT NULL CHECK (duration_minutes > 0),
    format             TEXT        CHECK (format IS NULL OR format IN ('in_person', 'online', 'phone')),

    scheduling_origin  TEXT        NOT NULL CHECK (scheduling_origin IN ('artist_set', 'client_booked')),

    created_at         TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at         TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_appointments_artist_start ON appointments (artist_id, scheduled_start);
CREATE INDEX idx_appointments_request ON appointments (booking_request_id);
