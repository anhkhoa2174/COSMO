-- Follow-up timing was compiled into the binary, so every organisation ran on
-- the same 8-hour no-reply window and the same follow-up spacing. An admin now
-- sets its own cadence; NULL means "use the built-in default", which keeps
-- existing rows behaving exactly as they did before this column existed.
ALTER TABLE organizations
    ADD COLUMN IF NOT EXISTS outreach_settings JSONB;

COMMENT ON COLUMN organizations.outreach_settings IS
    'Per-organisation outreach cadence. Keys: no_reply_hours, follow_up1_min_days, follow_up1_max_days, follow_up2_min_days, follow_up2_max_days, meeting_confirm_min_days, re_engage_threshold_days, max_followups. Absent keys fall back to the service defaults.';
