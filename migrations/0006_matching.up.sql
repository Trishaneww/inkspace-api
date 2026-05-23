CREATE TABLE match_requests (
    id               UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    client_id        UUID        NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    description      TEXT        NOT NULL,
    reference_urls   TEXT[]      NOT NULL DEFAULT '{}',
    styles           TEXT[]      NOT NULL DEFAULT '{}',
    placement        TEXT,
    size_inches      NUMERIC(5,2),
    budget_min_cents BIGINT      CHECK (budget_min_cents IS NULL OR budget_min_cents >= 0),
    budget_max_cents BIGINT      CHECK (budget_max_cents IS NULL OR budget_max_cents >= 0),
    city             TEXT,
    country          TEXT,
    status           TEXT        NOT NULL DEFAULT 'open'
                     CHECK (status IN ('open', 'matched', 'converted', 'expired')),
    expires_at       TIMESTAMPTZ,
    created_at       TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_match_requests_client_id ON match_requests (client_id, created_at DESC);
CREATE INDEX idx_match_requests_status ON match_requests (status);
CREATE INDEX idx_match_requests_styles ON match_requests USING GIN (styles);

CREATE TABLE matches (
    id         UUID             PRIMARY KEY DEFAULT gen_random_uuid(),
    request_id UUID             NOT NULL REFERENCES match_requests (id) ON DELETE CASCADE,
    artist_id  UUID             NOT NULL REFERENCES artists (id) ON DELETE CASCADE,
    score      DOUBLE PRECISION NOT NULL DEFAULT 0,
    status     TEXT             NOT NULL DEFAULT 'pending'
               CHECK (status IN ('pending', 'interested', 'passed')),
    created_at TIMESTAMPTZ      NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ      NOT NULL DEFAULT now(),
    UNIQUE (request_id, artist_id)
);

CREATE INDEX idx_matches_request_id ON matches (request_id);
CREATE INDEX idx_matches_artist_id_status ON matches (artist_id, status);
