CREATE TABLE open_books (
    id               UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    artist_id        UUID        NOT NULL UNIQUE REFERENCES artists (id) ON DELETE CASCADE,

    slug             CITEXT      NOT NULL UNIQUE,
    scheduling_mode  TEXT        NOT NULL DEFAULT 'artist_scheduled'
                     CHECK (scheduling_mode IN ('artist_scheduled', 'client_scheduled')),

    created_at       TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at       TIMESTAMPTZ NOT NULL DEFAULT now(),

    custom_questions JSONB       NOT NULL DEFAULT '[]',

    theme            TEXT        NOT NULL DEFAULT 'inkspace'
                     CHECK (theme IN ('inkspace', 'noir', 'sand', 'sage', 'midnight', 'navy', 'custom')),
    -- Set only when theme = 'custom': artist-chosen hex palette + optional background image.
    custom_theme         JSONB,
    background_image_key TEXT
);
