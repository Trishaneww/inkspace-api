ALTER TABLE artist_settings
    ADD COLUMN styles TEXT[] NOT NULL DEFAULT '{}';

ALTER TABLE open_books
    ADD COLUMN custom_questions JSONB NOT NULL DEFAULT '[]';

ALTER TABLE booking_requests
    ADD COLUMN styles         TEXT[] NOT NULL DEFAULT '{}',
    ADD COLUMN custom_answers JSONB  NOT NULL DEFAULT '[]';
