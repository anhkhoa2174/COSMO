-- Next-step decision engine (report Section 6.7).
--
-- The engine is additive: the cadence keeps writing contacts.next_step and
-- contacts.outreach_stage exactly as before, and the engine writes its own
-- columns beside them. An organisation that never turns the engine on sees no
-- change, and turning it off again leaves the cadence untouched.

-- The latest decision for each contact, denormalised onto the contact so the
-- outreach list can show it without a join.
ALTER TABLE contacts
    ADD COLUMN IF NOT EXISTS next_action             VARCHAR(40),
    ADD COLUMN IF NOT EXISTS next_action_args        JSONB,
    ADD COLUMN IF NOT EXISTS next_action_reason      TEXT,
    ADD COLUMN IF NOT EXISTS next_action_due_at      TIMESTAMPTZ,
    ADD COLUMN IF NOT EXISTS next_action_decision_id UUID;

CREATE INDEX IF NOT EXISTS idx_contacts_next_action_due_at
    ON contacts (next_action_due_at)
    WHERE next_action_due_at IS NOT NULL AND is_deleted = false;

-- Every decision the engine takes, with what it saw and why. The record, not
-- the model's stated reason, is the explanation: it names the rules that
-- removed each action that was not offered.
CREATE TABLE IF NOT EXISTS next_step_decisions (
    id             UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id        UUID NOT NULL,
    contact_id     UUID NOT NULL REFERENCES contacts(id) ON DELETE CASCADE,
    trigger        VARCHAR(20) NOT NULL,           -- reply, timer, cadence, manual
    situation      JSONB NOT NULL DEFAULT '{}',    -- the facts the decision was taken on
    rules_fired    JSONB NOT NULL DEFAULT '[]',    -- [{rule, effect, removed}]
    eligible       JSONB NOT NULL DEFAULT '[]',    -- actions left after the rules
    action         VARCHAR(40) NOT NULL,
    args           JSONB NOT NULL DEFAULT '{}',
    reason         TEXT NOT NULL DEFAULT '',
    selected_by    VARCHAR(20) NOT NULL,           -- rule (one action left), model, fallback
    fallback_cause TEXT NOT NULL DEFAULT '',
    approval       VARCHAR(20) NOT NULL,           -- automatic, approve, confirm, decide, task
    status         VARCHAR(20) NOT NULL,           -- applied, pending_review, approved, rejected
    model          VARCHAR(100) NOT NULL DEFAULT '',
    prompt_version VARCHAR(20) NOT NULL DEFAULT '',
    reviewed_by    UUID,
    reviewed_at    TIMESTAMPTZ,
    created_at     TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_next_step_decisions_contact
    ON next_step_decisions (contact_id, created_at DESC);
