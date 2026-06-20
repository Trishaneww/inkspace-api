ALTER TABLE artist_settings
    ADD COLUMN monthly_booking_goal INTEGER NOT NULL DEFAULT 20
        CHECK (monthly_booking_goal > 0);
