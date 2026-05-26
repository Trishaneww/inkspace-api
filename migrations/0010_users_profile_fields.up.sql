-- Add profile fields the frontend collects on signup.
ALTER TABLE users
    ADD COLUMN first_name        TEXT,
    ADD COLUMN last_name         TEXT,
    ADD COLUMN phone             TEXT,
    ADD COLUMN phone_verified_at TIMESTAMPTZ;

-- Sparse unique index on phone (only enforces uniqueness for non-null phones).
CREATE UNIQUE INDEX idx_users_phone_unique
    ON users (phone)
    WHERE phone IS NOT NULL;

-- Re-align role values with the frontend (artist / studio_admin / user).
-- Map any legacy rows so the new CHECK constraint doesn't reject them.
UPDATE users SET role = 'user'         WHERE role = 'client';
UPDATE users SET role = 'studio_admin' WHERE role = 'admin';

ALTER TABLE users DROP CONSTRAINT users_role_check;
ALTER TABLE users
    ADD CONSTRAINT users_role_check
    CHECK (role IN ('artist', 'studio_admin', 'user'));
