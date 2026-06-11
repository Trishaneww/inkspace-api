UPDATE booking_requests SET client_phone = '' WHERE client_phone IS NULL;

ALTER TABLE booking_requests
    ALTER COLUMN client_phone SET DEFAULT '',
    ALTER COLUMN client_phone SET NOT NULL;
