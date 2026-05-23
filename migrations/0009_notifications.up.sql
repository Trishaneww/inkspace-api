CREATE TABLE notifications (
    id         UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id    UUID        NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    kind       TEXT        NOT NULL,
    title      TEXT        NOT NULL,
    body       TEXT        NOT NULL DEFAULT '',
    data       JSONB       NOT NULL DEFAULT '{}'::jsonb,
    read_at    TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_notifications_user_id_created
    ON notifications (user_id, created_at DESC);

CREATE INDEX idx_notifications_unread
    ON notifications (user_id, created_at DESC)
    WHERE read_at IS NULL;

CREATE TABLE notification_preferences (
    user_id     UUID        PRIMARY KEY REFERENCES users (id) ON DELETE CASCADE,
    enabled     JSONB       NOT NULL DEFAULT '{}'::jsonb,
    quiet_hours JSONB,
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE notification_deliveries (
    id              UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    notification_id UUID        NOT NULL REFERENCES notifications (id) ON DELETE CASCADE,
    channel         TEXT        NOT NULL CHECK (channel IN ('in_app', 'email', 'push')),
    status          TEXT        NOT NULL DEFAULT 'pending'
                    CHECK (status IN ('pending', 'sent', 'failed', 'skipped')),
    error           TEXT,
    attempted_at    TIMESTAMPTZ,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_notification_deliveries_notification_id
    ON notification_deliveries (notification_id);

CREATE INDEX idx_notification_deliveries_pending
    ON notification_deliveries (created_at)
    WHERE status = 'pending';
