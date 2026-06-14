ALTER TABLE booking_requests
    DROP CONSTRAINT booking_requests_status_check;

ALTER TABLE booking_requests
    ADD CONSTRAINT booking_requests_status_check
    CHECK (status IN ('pending', 'consultation_requested', 'accepted',
                      'declined', 'expired', 'converted', 'cancelled'));
