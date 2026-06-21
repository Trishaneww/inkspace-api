DROP INDEX IF EXISTS idx_appointments_hold_expiry;

ALTER TABLE appointments
    DROP COLUMN IF EXISTS hold_expires_at;

ALTER TABLE appointments
    DROP CONSTRAINT appointments_status_check,
    ADD CONSTRAINT appointments_status_check
        CHECK (status IN ('proposed', 'scheduled', 'completed', 'cancelled', 'no_show'));
