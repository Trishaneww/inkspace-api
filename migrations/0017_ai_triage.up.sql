ALTER TABLE booking_requests
    ADD COLUMN ai_status        TEXT        NOT NULL DEFAULT 'pending'
                                CHECK (ai_status IN ('pending', 'ready', 'failed', 'skipped')),
    ADD COLUMN ai_label         TEXT        CHECK (ai_label IS NULL OR ai_label IN ('book', 'needs_info', 'pass')),
    ADD COLUMN ai_summary       TEXT        NOT NULL DEFAULT '',
    -- ai_signals: { "good": ["fits your style", ...], "bad": ["budget unclear", ...] }
    ADD COLUMN ai_signals       JSONB       NOT NULL DEFAULT '{}',
    -- ai_red_flags: ["mentioned they're 17", ...]
    ADD COLUMN ai_red_flags     JSONB       NOT NULL DEFAULT '[]',
    ADD COLUMN ai_reasoning     TEXT        NOT NULL DEFAULT '',
    ADD COLUMN ai_value_cents   BIGINT      CHECK (ai_value_cents IS NULL OR ai_value_cents >= 0),
    ADD COLUMN ai_session_count INTEGER     CHECK (ai_session_count IS NULL OR ai_session_count > 0),
    ADD COLUMN ai_draft_reply   TEXT        NOT NULL DEFAULT '',
    ADD COLUMN ai_updated_at    TIMESTAMPTZ;
