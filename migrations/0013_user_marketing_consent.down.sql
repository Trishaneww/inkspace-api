ALTER TABLE users
    DROP COLUMN IF EXISTS marketing_opt_in_at,
    DROP COLUMN IF EXISTS marketing_opt_in_source;
