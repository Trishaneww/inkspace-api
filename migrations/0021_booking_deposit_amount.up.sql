ALTER TABLE booking_requests
    ADD COLUMN deposit_amount_cents BIGINT
        CHECK (deposit_amount_cents IS NULL OR deposit_amount_cents >= 0);
