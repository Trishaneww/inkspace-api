CREATE TABLE artist_settings (
    artist_id              UUID        PRIMARY KEY REFERENCES artists (id) ON DELETE CASCADE,

    -- Personal Info (business side). Structured so a future booking page can
    -- render the studio on a map (geocode by concatenating the parts).
    studio_name            TEXT        NOT NULL DEFAULT '',
    studio_address         TEXT        NOT NULL DEFAULT '',
    studio_city            TEXT        NOT NULL DEFAULT '',
    studio_province        TEXT        NOT NULL DEFAULT '',
    studio_postal_code     TEXT        NOT NULL DEFAULT '',
    studio_country         TEXT        NOT NULL DEFAULT '',

    -- Payments & payouts. stripe_account_id NULL = not yet connected.
    stripe_account_id      TEXT,
    payout_frequency       TEXT        NOT NULL DEFAULT 'weekly'
                           CHECK (payout_frequency IN ('weekly', 'biweekly', 'monthly')),
    currency               CHAR(3)     NOT NULL DEFAULT 'CAD',

    -- Deposits. deposit_flat_fee_cents NULL = artist sets a custom amount per
    -- client when sending the deposit link.
    deposit_flat_fee_cents BIGINT      CHECK (deposit_flat_fee_cents IS NULL OR deposit_flat_fee_cents >= 0),
    platform_fee_payer     TEXT        NOT NULL DEFAULT 'client'
                           CHECK (platform_fee_payer IN ('artist', 'client', 'split')),

    -- Booking preferences
    accepting_bookings     BOOLEAN     NOT NULL DEFAULT true,
    timezone               TEXT        NOT NULL DEFAULT 'America/Toronto',
    -- google_calendar_email NULL = calendar not connected.
    google_calendar_email  TEXT,
    slot_interval_minutes  INTEGER     NOT NULL DEFAULT 60
                           CHECK (slot_interval_minutes IN (15, 30, 60)),
    buffer_minutes         INTEGER     NOT NULL DEFAULT 0  CHECK (buffer_minutes >= 0),
    min_notice_minutes     INTEGER     NOT NULL DEFAULT 0  CHECK (min_notice_minutes >= 0),
    -- max_advance_days NULL = no upper bound on how far out clients can book.
    max_advance_days       INTEGER     CHECK (max_advance_days IS NULL OR max_advance_days > 0),

    -- Terms & consent
    terms_text             TEXT        NOT NULL DEFAULT '',
    terms_show_on_booking  BOOLEAN     NOT NULL DEFAULT false,
    terms_show_at_deposit  BOOLEAN     NOT NULL DEFAULT false,
    waiver_file_url        TEXT,
    waiver_required        BOOLEAN     NOT NULL DEFAULT false,

    -- Notifications & messaging
    notify_by_email        BOOLEAN     NOT NULL DEFAULT true,
    notify_by_sms          BOOLEAN     NOT NULL DEFAULT false,

    created_at             TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at             TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE artist_availability_windows (
    id           UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    artist_id    UUID        NOT NULL REFERENCES artists (id) ON DELETE CASCADE,
    weekday      INTEGER     NOT NULL CHECK (weekday BETWEEN 0 AND 6),   -- 0 = Sunday
    start_minute INTEGER     NOT NULL CHECK (start_minute BETWEEN 0 AND 1439),
    end_minute   INTEGER     NOT NULL CHECK (end_minute   BETWEEN 1 AND 1440),
    CONSTRAINT availability_window_end_after_start CHECK (end_minute > start_minute)
);

CREATE INDEX idx_availability_windows_artist
    ON artist_availability_windows (artist_id, weekday, start_minute);

CREATE TABLE artist_session_presets (
    id                      UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    artist_id               UUID        NOT NULL REFERENCES artists (id) ON DELETE CASCADE,
    name                    TEXT        NOT NULL,
    description             TEXT        NOT NULL DEFAULT '',
    approx_duration_minutes INTEGER     NOT NULL CHECK (approx_duration_minutes > 0),
    position                INTEGER     NOT NULL DEFAULT 0,
    created_at              TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_session_presets_artist
    ON artist_session_presets (artist_id, position);

CREATE TABLE artist_days_off (
    artist_id UUID NOT NULL REFERENCES artists (id) ON DELETE CASCADE,
    day       DATE NOT NULL,
    PRIMARY KEY (artist_id, day)
);

CREATE TABLE artist_blocklist (
    id         UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    artist_id  UUID        NOT NULL REFERENCES artists (id) ON DELETE CASCADE,
    email      CITEXT,
    phone      TEXT,
    note       TEXT        NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT blocklist_requires_email_or_phone CHECK (email IS NOT NULL OR phone IS NOT NULL)
);

CREATE INDEX idx_blocklist_artist ON artist_blocklist (artist_id);
