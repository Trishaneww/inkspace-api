ALTER TABLE artist_settings
    ADD COLUMN google_calendar_access_token  TEXT,
    ADD COLUMN google_calendar_refresh_token TEXT,
    ADD COLUMN google_calendar_token_expiry  TIMESTAMPTZ;
