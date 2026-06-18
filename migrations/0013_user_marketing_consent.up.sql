ALTER TABLE users
    ADD COLUMN marketing_opt_in_at     TIMESTAMPTZ,
    ADD COLUMN marketing_opt_in_source TEXT;
