ALTER TABLE payment_requests
    ADD COLUMN reminder_sent_at TIMESTAMPTZ;

ALTER TABLE appointments
    ADD COLUMN reminder_sent_at TIMESTAMPTZ;
