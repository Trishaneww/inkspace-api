ALTER TABLE appointments
    DROP COLUMN IF EXISTS reminder_sent_at;

ALTER TABLE payment_requests
    DROP COLUMN IF EXISTS reminder_sent_at;
