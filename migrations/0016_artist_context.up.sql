ALTER TABLE artist_settings
    ADD COLUMN min_session_price_cents BIGINT
        CHECK (min_session_price_cents IS NULL OR min_session_price_cents >= 0),
    ADD COLUMN declined_placements     TEXT[] NOT NULL DEFAULT '{}',
    ADD COLUMN declined_styles         TEXT[] NOT NULL DEFAULT '{}',
    ADD COLUMN work_summary            TEXT   NOT NULL DEFAULT '';
