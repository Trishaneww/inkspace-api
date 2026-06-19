ALTER TABLE booking_requests
    ADD COLUMN schedule_token      TEXT UNIQUE,
    ADD COLUMN schedule_emailed_at TIMESTAMPTZ;
