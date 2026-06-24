ALTER TABLE artist_settings
    DROP COLUMN IF EXISTS work_summary,
    DROP COLUMN IF EXISTS declined_styles,
    DROP COLUMN IF EXISTS declined_placements,
    DROP COLUMN IF EXISTS min_session_price_cents;
