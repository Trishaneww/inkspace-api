CREATE TABLE artist_locations (
    id           UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    artist_id    UUID        NOT NULL REFERENCES artists (id) ON DELETE CASCADE,
    label        TEXT        NOT NULL DEFAULT '',
    address      TEXT        NOT NULL DEFAULT '',
    city         TEXT        NOT NULL DEFAULT '',
    province     TEXT        NOT NULL DEFAULT '',
    postal_code  TEXT        NOT NULL DEFAULT '',
    country      TEXT        NOT NULL DEFAULT '',
    timezone     TEXT        NOT NULL DEFAULT '',
    is_primary   BOOLEAN     NOT NULL DEFAULT false,
    -- Guest spots carry a date range; the home studio leaves both NULL.
    start_date   DATE,
    end_date     DATE,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at   TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT artist_location_dates_valid CHECK (
        (start_date IS NULL AND end_date IS NULL)
        OR (start_date IS NOT NULL AND end_date IS NOT NULL AND end_date >= start_date)
    )
);

CREATE INDEX idx_artist_locations_artist ON artist_locations (artist_id);

CREATE UNIQUE INDEX idx_artist_locations_primary
    ON artist_locations (artist_id) WHERE is_primary;

ALTER TABLE artist_settings
    DROP COLUMN studio_name,
    DROP COLUMN studio_address,
    DROP COLUMN studio_city,
    DROP COLUMN studio_province,
    DROP COLUMN studio_postal_code,
    DROP COLUMN studio_country,
    ADD COLUMN current_location_id UUID REFERENCES artist_locations (id) ON DELETE SET NULL;

ALTER TABLE booking_requests
    ADD COLUMN location_id UUID REFERENCES artist_locations (id) ON DELETE SET NULL;
