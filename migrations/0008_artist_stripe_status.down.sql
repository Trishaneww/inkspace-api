ALTER TABLE artist_settings DROP CONSTRAINT artist_settings_payout_frequency_check;
ALTER TABLE artist_settings ADD CONSTRAINT artist_settings_payout_frequency_check
    CHECK (payout_frequency IN ('weekly', 'biweekly', 'monthly'));

ALTER TABLE artist_settings
    DROP COLUMN IF EXISTS cancellation_notice_hours,
    DROP COLUMN IF EXISTS deposit_refund_policy,
    DROP COLUMN IF EXISTS stripe_details_submitted,
    DROP COLUMN IF EXISTS stripe_payouts_enabled,
    DROP COLUMN IF EXISTS stripe_charges_enabled;
