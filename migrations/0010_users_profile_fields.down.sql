ALTER TABLE users DROP CONSTRAINT users_role_check;
ALTER TABLE users
    ADD CONSTRAINT users_role_check
    CHECK (role IN ('client', 'artist', 'admin'));

UPDATE users SET role = 'client' WHERE role = 'user';
UPDATE users SET role = 'admin'  WHERE role = 'studio_admin';

DROP INDEX IF EXISTS idx_users_phone_unique;

ALTER TABLE users
    DROP COLUMN IF EXISTS phone_verified_at,
    DROP COLUMN IF EXISTS phone,
    DROP COLUMN IF EXISTS last_name,
    DROP COLUMN IF EXISTS first_name;
