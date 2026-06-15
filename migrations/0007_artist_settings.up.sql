CREATE TABLE artist_settings (
    artist_id              UUID        PRIMARY KEY REFERENCES artists (id) ON DELETE CASCADE,

    stripe_account_id      TEXT,
    payout_frequency       TEXT        NOT NULL DEFAULT 'weekly'
                           CHECK (payout_frequency IN ('weekly', 'monthly')),
    currency               CHAR(3)     NOT NULL DEFAULT 'CAD',

    deposit_flat_fee_cents BIGINT      CHECK (deposit_flat_fee_cents IS NULL OR deposit_flat_fee_cents >= 0),
    platform_fee_payer     TEXT        NOT NULL DEFAULT 'client'
                           CHECK (platform_fee_payer IN ('artist', 'client', 'split')),

    accepting_bookings     BOOLEAN     NOT NULL DEFAULT true,
    timezone               TEXT        NOT NULL DEFAULT 'America/Toronto',
    google_calendar_email  TEXT,
    slot_interval_minutes  INTEGER     NOT NULL DEFAULT 60
                           CHECK (slot_interval_minutes IN (15, 30, 60)),
    buffer_minutes         INTEGER     NOT NULL DEFAULT 0  CHECK (buffer_minutes >= 0),
    min_notice_minutes     INTEGER     NOT NULL DEFAULT 0  CHECK (min_notice_minutes >= 0),
    max_advance_days       INTEGER     CHECK (max_advance_days IS NULL OR max_advance_days > 0),

    terms_text             TEXT        NOT NULL DEFAULT '',
    terms_show_on_booking  BOOLEAN     NOT NULL DEFAULT false,
    terms_show_at_deposit  BOOLEAN     NOT NULL DEFAULT false,
    waiver_file_url        TEXT,
    waiver_required        BOOLEAN     NOT NULL DEFAULT false,

    notify_by_email        BOOLEAN     NOT NULL DEFAULT true,
    notify_by_sms          BOOLEAN     NOT NULL DEFAULT false,

    created_at             TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at             TIMESTAMPTZ NOT NULL DEFAULT now(),

    google_calendar_access_token  TEXT,
    google_calendar_refresh_token TEXT,
    google_calendar_token_expiry  TIMESTAMPTZ,

    stripe_charges_enabled   BOOLEAN NOT NULL DEFAULT false,
    stripe_payouts_enabled   BOOLEAN NOT NULL DEFAULT false,
    stripe_details_submitted BOOLEAN NOT NULL DEFAULT false,

    deposit_refund_policy TEXT NOT NULL DEFAULT 'non_refundable'
        CHECK (deposit_refund_policy IN ('non_refundable', 'refundable_within_window', 'always_refundable')),
    cancellation_notice_hours INTEGER
        CHECK (cancellation_notice_hours IS NULL OR cancellation_notice_hours > 0),

    styles TEXT[] NOT NULL DEFAULT '{}',

    aftercare TEXT  NOT NULL DEFAULT '',
    faqs      JSONB NOT NULL DEFAULT '[]',

    current_location_id UUID REFERENCES artist_locations (id) ON DELETE SET NULL
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
