ALTER TABLE artist_locations
    DROP CONSTRAINT IF EXISTS artist_locations_closed_requires_timestamp,
    DROP COLUMN closed_at,
    DROP COLUMN status;
