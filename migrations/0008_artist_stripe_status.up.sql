-- Stripe Connect onboarding status, mirrored from the connected account so the
-- booking flow can gate on "can actually accept payments" without a live API
-- call on every request. stripe_account_id (added in 0006) stays the
-- "connected?" signal; an account can exist but be mid-onboarding or restricted,
-- so charges_enabled is the real "can take deposits" gate. Kept fresh by the
-- account.updated webhook and the post-onboarding status refresh.
ALTER TABLE artist_settings
    ADD COLUMN stripe_charges_enabled   BOOLEAN NOT NULL DEFAULT false,
    ADD COLUMN stripe_payouts_enabled   BOOLEAN NOT NULL DEFAULT false,
    ADD COLUMN stripe_details_submitted BOOLEAN NOT NULL DEFAULT false;

-- Deposit refund policy. Snapshotted onto each payment request at pay time so
-- it stays as immutable dispute evidence even if the artist changes it later.
-- cancellation_notice_hours only applies to the 'refundable_within_window'
-- policy: the client must cancel at least this many hours before the
-- appointment to be eligible for a deposit refund.
ALTER TABLE artist_settings
    ADD COLUMN deposit_refund_policy TEXT NOT NULL DEFAULT 'non_refundable'
        CHECK (deposit_refund_policy IN ('non_refundable', 'refundable_within_window', 'always_refundable')),
    ADD COLUMN cancellation_notice_hours INTEGER
        CHECK (cancellation_notice_hours IS NULL OR cancellation_notice_hours > 0);

-- Stripe automatic payouts only support 'weekly' and 'monthly' (there is no
-- native biweekly interval), so the schema can no longer promise biweekly.
-- Fold any existing biweekly rows down to weekly before tightening the check.
UPDATE artist_settings SET payout_frequency = 'weekly' WHERE payout_frequency = 'biweekly';
ALTER TABLE artist_settings DROP CONSTRAINT artist_settings_payout_frequency_check;
ALTER TABLE artist_settings ADD CONSTRAINT artist_settings_payout_frequency_check
    CHECK (payout_frequency IN ('weekly', 'monthly'));
