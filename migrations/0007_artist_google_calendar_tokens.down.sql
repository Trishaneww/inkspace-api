ALTER TABLE artist_settings
    DROP COLUMN IF EXISTS google_calendar_access_token,
    DROP COLUMN IF EXISTS google_calendar_refresh_token,
    DROP COLUMN IF EXISTS google_calendar_token_expiry;
