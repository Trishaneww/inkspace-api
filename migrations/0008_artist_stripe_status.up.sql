
ALTER TABLE artist_settings
    ADD COLUMN stripe_charges_enabled   BOOLEAN NOT NULL DEFAULT false,
    ADD COLUMN stripe_payouts_enabled   BOOLEAN NOT NULL DEFAULT false,
    ADD COLUMN stripe_details_submitted BOOLEAN NOT NULL DEFAULT false;

ALTER TABLE artist_settings
    ADD COLUMN deposit_refund_policy TEXT NOT NULL DEFAULT 'non_refundable'
        CHECK (deposit_refund_policy IN ('non_refundable', 'refundable_within_window', 'always_refundable')),
    ADD COLUMN cancellation_notice_hours INTEGER
        CHECK (cancellation_notice_hours IS NULL OR cancellation_notice_hours > 0);

UPDATE artist_settings SET payout_frequency = 'weekly' WHERE payout_frequency = 'biweekly';
ALTER TABLE artist_settings DROP CONSTRAINT artist_settings_payout_frequency_check;
ALTER TABLE artist_settings ADD CONSTRAINT artist_settings_payout_frequency_check
    CHECK (payout_frequency IN ('weekly', 'monthly'));
