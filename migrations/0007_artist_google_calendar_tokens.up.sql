-- Google Calendar OAuth tokens for the artist's connected account.
-- Stored encrypted at rest (AES-256-GCM) since they grant ongoing API access;
-- NULL = calendar not connected. google_calendar_email (added in 0006) remains
-- the "connected?" signal surfaced to the client.
ALTER TABLE artist_settings
    ADD COLUMN google_calendar_access_token  TEXT,
    ADD COLUMN google_calendar_refresh_token TEXT,
    ADD COLUMN google_calendar_token_expiry  TIMESTAMPTZ;
