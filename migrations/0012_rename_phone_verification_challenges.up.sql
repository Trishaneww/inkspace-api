-- Rename `phone_verification_challenges` to the cleaner `phone_verifications`.
-- "Challenge" is borrowed from security/MFA terminology; in our phone-OTP
-- context, "verification" is more idiomatic and matches industry libraries
-- (Firebase, Twilio, Auth0).

ALTER TABLE phone_verification_challenges RENAME TO phone_verifications;
ALTER INDEX idx_phone_challenges_user_id    RENAME TO idx_phone_verifications_user_id;
ALTER INDEX idx_phone_challenges_expires_at RENAME TO idx_phone_verifications_expires_at;
