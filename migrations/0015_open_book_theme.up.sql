ALTER TABLE open_books
    ADD COLUMN theme TEXT NOT NULL DEFAULT 'inkspace'
    CHECK (theme IN ('inkspace', 'noir', 'sand', 'sage', 'midnight', 'navy'));
