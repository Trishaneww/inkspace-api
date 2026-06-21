ALTER TABLE open_books
    DROP CONSTRAINT open_books_theme_check,
    ADD CONSTRAINT open_books_theme_check
        CHECK (theme IN ('inkspace', 'noir', 'sand', 'sage', 'midnight', 'navy', 'custom'));

ALTER TABLE open_books
    ADD COLUMN custom_theme         JSONB,
    ADD COLUMN background_image_key TEXT;
