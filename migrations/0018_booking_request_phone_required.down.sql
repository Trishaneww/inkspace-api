ALTER TABLE booking_requests
    ALTER COLUMN client_phone DROP NOT NULL,
    ALTER COLUMN client_phone DROP DEFAULT;
