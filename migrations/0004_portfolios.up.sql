CREATE TABLE portfolio_items (
    id           UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    artist_id    UUID        NOT NULL REFERENCES artists (id) ON DELETE CASCADE,
    image_url    TEXT        NOT NULL,
    caption      TEXT        NOT NULL DEFAULT '',
    styles       TEXT[]      NOT NULL DEFAULT '{}',
    placement    TEXT,
    healed_photo BOOLEAN     NOT NULL DEFAULT false,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_portfolio_items_artist_id ON portfolio_items (artist_id, created_at DESC);
CREATE INDEX idx_portfolio_items_styles ON portfolio_items USING GIN (styles);

CREATE TABLE flash_designs (
    id             UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    artist_id      UUID        NOT NULL REFERENCES artists (id) ON DELETE CASCADE,
    title          TEXT        NOT NULL,
    description    TEXT        NOT NULL DEFAULT '',
    image_url      TEXT        NOT NULL,
    styles         TEXT[]      NOT NULL DEFAULT '{}',
    placement      TEXT,
    size_inches    NUMERIC(5,2),
    price_cents    BIGINT      NOT NULL CHECK (price_cents > 0),
    currency       TEXT        NOT NULL CHECK (char_length(currency) = 3),
    repeatable     BOOLEAN     NOT NULL DEFAULT false,
    status         TEXT        NOT NULL DEFAULT 'available'
                   CHECK (status IN ('available', 'reserved', 'claimed', 'retired')),
    reserved_by    UUID        REFERENCES users (id) ON DELETE SET NULL,
    reserved_until TIMESTAMPTZ,
    created_at     TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at     TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_flash_designs_artist_id ON flash_designs (artist_id, created_at DESC);
CREATE INDEX idx_flash_designs_status ON flash_designs (status);
CREATE INDEX idx_flash_designs_styles ON flash_designs USING GIN (styles);
