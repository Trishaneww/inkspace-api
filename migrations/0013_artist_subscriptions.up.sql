ALTER TABLE artists
    ADD COLUMN stripe_customer_id                TEXT,
    ADD COLUMN stripe_subscription_id            TEXT,
    ADD COLUMN subscription_status               TEXT        NOT NULL DEFAULT 'none'
                                                 CHECK (subscription_status IN (
                                                     'none', 'incomplete', 'incomplete_expired',
                                                     'trialing', 'active', 'past_due',
                                                     'canceled', 'unpaid', 'paused'
                                                 )),
    ADD COLUMN subscription_current_period_end   TIMESTAMPTZ,
    ADD COLUMN subscription_cancel_at_period_end BOOLEAN     NOT NULL DEFAULT false;

CREATE UNIQUE INDEX idx_artists_stripe_customer
    ON artists (stripe_customer_id)
    WHERE stripe_customer_id IS NOT NULL;

CREATE UNIQUE INDEX idx_artists_stripe_subscription
    ON artists (stripe_subscription_id)
    WHERE stripe_subscription_id IS NOT NULL;
