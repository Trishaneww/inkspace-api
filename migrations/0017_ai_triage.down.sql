ALTER TABLE booking_requests
    DROP COLUMN IF EXISTS ai_updated_at,
    DROP COLUMN IF EXISTS ai_draft_reply,
    DROP COLUMN IF EXISTS ai_session_count,
    DROP COLUMN IF EXISTS ai_value_cents,
    DROP COLUMN IF EXISTS ai_reasoning,
    DROP COLUMN IF EXISTS ai_red_flags,
    DROP COLUMN IF EXISTS ai_signals,
    DROP COLUMN IF EXISTS ai_summary,
    DROP COLUMN IF EXISTS ai_label,
    DROP COLUMN IF EXISTS ai_status;
