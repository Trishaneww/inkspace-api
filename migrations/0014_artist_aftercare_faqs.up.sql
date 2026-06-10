ALTER TABLE artist_settings
    ADD COLUMN aftercare TEXT  NOT NULL DEFAULT '',
    ADD COLUMN faqs      JSONB NOT NULL DEFAULT '[]';
