ALTER TABLE booking_requests DROP CONSTRAINT booking_requests_ai_label_check;
ALTER TABLE booking_requests ADD CONSTRAINT booking_requests_ai_label_check
    CHECK (ai_label IS NULL OR ai_label IN ('book', 'needs_info', 'pass'));
