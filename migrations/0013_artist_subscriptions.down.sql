DROP INDEX IF EXISTS idx_artists_stripe_subscription;
DROP INDEX IF EXISTS idx_artists_stripe_customer;

ALTER TABLE artists
    DROP COLUMN IF EXISTS subscription_cancel_at_period_end,
    DROP COLUMN IF EXISTS subscription_current_period_end,
    DROP COLUMN IF EXISTS subscription_status,
    DROP COLUMN IF EXISTS stripe_subscription_id,
    DROP COLUMN IF EXISTS stripe_customer_id;
