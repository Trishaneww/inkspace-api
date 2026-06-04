CREATE TABLE flashes (
    id                    UUID         PRIMARY KEY DEFAULT gen_random_uuid(),
    artist_id             UUID         NOT NULL REFERENCES artists (id) ON DELETE CASCADE,

    status                TEXT         NOT NULL DEFAULT 'draft'
                          CHECK (status IN ('draft', 'available', 'claimed', 'archived')),

    title                 TEXT         NOT NULL,
    description           TEXT,

    s3_key                TEXT,
    reference_s3_key      TEXT,

    color_type            TEXT         NOT NULL DEFAULT 'both'
                          CHECK (color_type IN ('black_and_grey', 'color', 'both')),
    styles                TEXT[]       NOT NULL DEFAULT '{}',
    placements            TEXT[]       NOT NULL DEFAULT '{}',

    pricing_mode          TEXT         NOT NULL DEFAULT 'per_size'
                          CHECK (pricing_mode IN ('per_size', 'flat')),
    flat_price_cents      BIGINT       CHECK (flat_price_cents IS NULL OR flat_price_cents > 0),
    flat_duration_minutes INTEGER      CHECK (flat_duration_minutes IS NULL OR flat_duration_minutes > 0),

    deposit_cents         BIGINT       CHECK (deposit_cents IS NULL OR deposit_cents >= 0),
    -- Currency is an artist-level setting (artist_settings.currency); a flash
    -- inherits its artist's currency at read time, so it isn't stored here.

    repeatable            BOOLEAN      NOT NULL DEFAULT false,

    claimed_at            TIMESTAMPTZ,
    claimed_by_booking_id UUID,
    archived_at           TIMESTAMPTZ,
    published_at          TIMESTAMPTZ,

    view_count            BIGINT       NOT NULL DEFAULT 0,
    save_count            BIGINT       NOT NULL DEFAULT 0,

    created_at            TIMESTAMPTZ  NOT NULL DEFAULT now(),
    updated_at            TIMESTAMPTZ  NOT NULL DEFAULT now(),

    CONSTRAINT flashes_claimed_requires_timestamp
        CHECK (status <> 'claimed'  OR claimed_at IS NOT NULL),
    CONSTRAINT flashes_archived_requires_timestamp
        CHECK (status <> 'archived' OR archived_at IS NOT NULL),
    CONSTRAINT flashes_flat_pricing_complete
        CHECK (pricing_mode <> 'flat'
               OR (flat_price_cents IS NOT NULL AND flat_duration_minutes IS NOT NULL))
);

CREATE INDEX idx_flashes_artist_status ON flashes (artist_id, status, created_at DESC);
CREATE INDEX idx_flashes_styles        ON flashes USING GIN (styles);
CREATE INDEX idx_flashes_placements    ON flashes USING GIN (placements);

CREATE TABLE flash_pricing_tiers (
    flash_id         UUID    NOT NULL REFERENCES flashes (id) ON DELETE CASCADE,
    size_code        TEXT    NOT NULL
                     CHECK (size_code IN ('x_small', 'small', 'medium', 'large', 'x_large')),
    duration_minutes INTEGER NOT NULL CHECK (duration_minutes > 0),
    price_cents      BIGINT  NOT NULL CHECK (price_cents > 0),
    PRIMARY KEY (flash_id, size_code)
);
