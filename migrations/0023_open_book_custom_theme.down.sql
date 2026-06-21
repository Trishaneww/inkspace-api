ALTER TABLE open_books
    DROP COLUMN IF EXISTS custom_theme,
    DROP COLUMN IF EXISTS background_image_key;

ALTER TABLE open_books
    DROP CONSTRAINT open_books_theme_check,
    ADD CONSTRAINT open_books_theme_check
        CHECK (theme IN ('inkspace', 'noir', 'sand', 'sage', 'midnight', 'navy'));
