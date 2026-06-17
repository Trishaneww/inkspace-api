ALTER TABLE payment_requests
    ADD COLUMN last_emailed_at TIMESTAMPTZ NOT NULL DEFAULT now();
