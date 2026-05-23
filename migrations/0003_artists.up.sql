CREATE TABLE artists (
    id           UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id      UUID        NOT NULL UNIQUE REFERENCES users (id) ON DELETE CASCADE,
    handle       CITEXT      NOT NULL UNIQUE,
    display_name TEXT        NOT NULL,
    bio          TEXT        NOT NULL DEFAULT '',
    styles       TEXT[]      NOT NULL DEFAULT '{}',
    city         TEXT,
    country      TEXT,
    booking_link TEXT,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at   TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_artists_city ON artists (city);
CREATE INDEX idx_artists_country ON artists (country);
CREATE INDEX idx_artists_styles ON artists USING GIN (styles);

CREATE TABLE artist_availability (
    artist_id      UUID        PRIMARY KEY REFERENCES artists (id) ON DELETE CASCADE,
    accepting_new  BOOLEAN     NOT NULL DEFAULT true,
    weekly_windows JSONB       NOT NULL DEFAULT '[]'::jsonb,
    waitlist_days  INTEGER     NOT NULL DEFAULT 0 CHECK (waitlist_days >= 0),
    updated_at     TIMESTAMPTZ NOT NULL DEFAULT now()
);
