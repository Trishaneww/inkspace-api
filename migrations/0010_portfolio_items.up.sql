CREATE TABLE portfolio_items (
    id                 UUID         PRIMARY KEY DEFAULT gen_random_uuid(),
    artist_id          UUID         NOT NULL REFERENCES artists (id) ON DELETE CASCADE,

    status             TEXT         NOT NULL DEFAULT 'draft'
                       CHECK (status IN ('draft', 'published', 'archived')),

    title              TEXT         NOT NULL,
    description        TEXT,
    completion_date    DATE,

    image_keys         TEXT[]       NOT NULL DEFAULT '{}',

    styles             TEXT[]       NOT NULL DEFAULT '{}',
    placement          TEXT,

    color_type         TEXT         CHECK (color_type IS NULL OR color_type IN ('black_and_grey', 'color')),
    approx_size_inches INTEGER      CHECK (approx_size_inches IS NULL OR approx_size_inches > 0),
    healed             BOOLEAN      NOT NULL DEFAULT false,
    session_count      INTEGER      CHECK (session_count IS NULL OR session_count > 0),
    total_minutes      INTEGER      CHECK (total_minutes IS NULL OR total_minutes > 0),

    archived_at        TIMESTAMPTZ,
    published_at       TIMESTAMPTZ,

    created_at         TIMESTAMPTZ  NOT NULL DEFAULT now(),
    updated_at         TIMESTAMPTZ  NOT NULL DEFAULT now(),

    CONSTRAINT portfolio_items_archived_requires_timestamp
        CHECK (status <> 'archived' OR archived_at IS NOT NULL)
);

CREATE INDEX idx_portfolio_items_artist_status ON portfolio_items (artist_id, status, created_at DESC);
CREATE INDEX idx_portfolio_items_styles        ON portfolio_items USING GIN (styles);
