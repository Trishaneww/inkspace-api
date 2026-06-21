ALTER TABLE appointments
    DROP CONSTRAINT appointments_status_check,
    ADD CONSTRAINT appointments_status_check
        CHECK (status IN ('proposed', 'awaiting_deposit', 'scheduled', 'completed', 'cancelled', 'no_show'));

ALTER TABLE appointments
    ADD COLUMN hold_expires_at TIMESTAMPTZ;

CREATE INDEX idx_appointments_hold_expiry
    ON appointments (hold_expires_at)
    WHERE status = 'awaiting_deposit';
