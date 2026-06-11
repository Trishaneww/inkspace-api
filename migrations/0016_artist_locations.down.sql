ALTER TABLE booking_requests
    DROP COLUMN location_id;

ALTER TABLE artist_settings
    DROP COLUMN current_location_id,
    ADD COLUMN studio_name        TEXT NOT NULL DEFAULT '',
    ADD COLUMN studio_address     TEXT NOT NULL DEFAULT '',
    ADD COLUMN studio_city        TEXT NOT NULL DEFAULT '',
    ADD COLUMN studio_province    TEXT NOT NULL DEFAULT '',
    ADD COLUMN studio_postal_code TEXT NOT NULL DEFAULT '',
    ADD COLUMN studio_country     TEXT NOT NULL DEFAULT '';

DROP TABLE artist_locations;
