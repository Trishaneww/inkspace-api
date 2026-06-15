CREATE TABLE booking_requests (
    id                       UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    artist_id                UUID        NOT NULL REFERENCES artists (id) ON DELETE CASCADE,
    open_book_id             UUID        NOT NULL REFERENCES open_books (id) ON DELETE CASCADE,

    type                     TEXT        NOT NULL CHECK (type IN ('flash', 'custom')),
    flash_id                 UUID        REFERENCES flashes (id) ON DELETE SET NULL,
    description              TEXT        NOT NULL DEFAULT '',
    reference_image_keys     TEXT[]      NOT NULL DEFAULT '{}',
    placement                TEXT        NOT NULL DEFAULT '',
    approx_size_inches       INTEGER     CHECK (approx_size_inches IS NULL OR approx_size_inches > 0),

    client_availability      JSONB       NOT NULL DEFAULT '[]',

    client_name              TEXT        NOT NULL,
    client_email             CITEXT      NOT NULL,
    client_phone             TEXT        NOT NULL DEFAULT '',

    status                   TEXT        NOT NULL DEFAULT 'pending'
                             CHECK (status IN ('pending', 'consultation_requested', 'accepted',
                                               'declined', 'expired', 'converted', 'cancelled')),
    deposit_status           TEXT        NOT NULL DEFAULT 'not_required'
                             CHECK (deposit_status IN ('not_required', 'pending', 'paid', 'refunded')),
    waiver_status            TEXT        NOT NULL DEFAULT 'not_required'
                             CHECK (waiver_status IN ('not_required', 'pending', 'signed')),

    session_duration_minutes INTEGER     CHECK (session_duration_minutes IS NULL OR session_duration_minutes > 0),
    client_user_id           UUID        REFERENCES users (id) ON DELETE SET NULL,

    created_at               TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at               TIMESTAMPTZ NOT NULL DEFAULT now(),
    decided_at               TIMESTAMPTZ,

    styles          TEXT[]  NOT NULL DEFAULT '{}',
    custom_answers  JSONB   NOT NULL DEFAULT '[]',
    color_type      TEXT    NOT NULL DEFAULT '',
    location_id     UUID    REFERENCES artist_locations (id) ON DELETE SET NULL,
    flash_size_code TEXT    NOT NULL DEFAULT ''
);

CREATE INDEX idx_booking_requests_artist_status
    ON booking_requests (artist_id, status, created_at DESC);
