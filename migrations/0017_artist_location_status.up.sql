-- Guest spots are closed (soft-deleted) rather than removed, so booking
-- requests keep their location reference and we retain a history of where
-- artists have worked.
ALTER TABLE artist_locations
    ADD COLUMN status    TEXT        NOT NULL DEFAULT 'active'
        CHECK (status IN ('active', 'closed')),
    ADD COLUMN closed_at TIMESTAMPTZ;

ALTER TABLE artist_locations
    ADD CONSTRAINT artist_locations_closed_requires_timestamp
        CHECK (status <> 'closed' OR closed_at IS NOT NULL);
