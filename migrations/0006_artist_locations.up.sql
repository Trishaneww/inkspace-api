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
    -- Guest spots are closed (soft-deleted) rather than removed, so booking
    -- requests keep their location reference and we retain a history of where
    -- artists have worked.
    status       TEXT        NOT NULL DEFAULT 'active'
                 CHECK (status IN ('active', 'closed')),
    closed_at    TIMESTAMPTZ,
    CONSTRAINT artist_location_dates_valid CHECK (
        (start_date IS NULL AND end_date IS NULL)
        OR (start_date IS NOT NULL AND end_date IS NOT NULL AND end_date >= start_date)
    ),
    CONSTRAINT artist_locations_closed_requires_timestamp
        CHECK (status <> 'closed' OR closed_at IS NOT NULL)
);

CREATE INDEX idx_artist_locations_artist ON artist_locations (artist_id);

CREATE UNIQUE INDEX idx_artist_locations_primary
    ON artist_locations (artist_id) WHERE is_primary;
