import { kyClient } from '@/lib/ky';

/**
 * Timing that governs when COSMO follows up.
 *
 * Every field is optional: an omitted field means "use the system default",
 * which is what lets an admin change one number without freezing the rest of
 * the cadence at whatever the defaults were the day they saved.
 */
/**
 * Whether an AI-written reply may be sent without a person reading it first.
 *
 * `intents` is an allowlist — there is no "all intents" option, and the server
 * refuses to store intents that can never be auto-sent (a decline, an opt-out,
 * or a reply it could not classify).
 */
export interface AutoReplyPolicy {
  enabled?: boolean;
  intents?: string[];
  min_confidence?: number;
  daily_cap?: number;
  /** What each intent's reply says, keyed like `intents`. */
  contents?: Record<string, AutoReplyContent>;
}

/**
 * What one intent's automatic reply says.
 *
 * `template` sends `body` as written, with merge fields filled in; an empty
 * subject keeps "Re: <their subject>". `ai` keeps the AI-written draft but adds
 * `guidance` to the instructions it follows. Content changes what is sent,
 * never whether it may be sent.
 */
export interface AutoReplyContent {
  mode: 'template' | 'ai';
  subject?: string;
  body?: string;
  guidance?: string;
}

export interface AutoReplyView {
  enabled: boolean;
  intents: string[];
  min_confidence: number;
  daily_cap: number;
  summary: string;
  contents: Record<string, AutoReplyContent>;
}

export interface OutreachSettings {
  auto_reply?: AutoReplyPolicy;
  no_reply_hours?: number;
  follow_up1_min_days?: number;
  follow_up1_max_days?: number;
  follow_up2_min_days?: number;
  follow_up2_max_days?: number;
  meeting_confirm_min_days?: number;
  re_engage_threshold_days?: number;
  max_followups?: number;
}

/** The numbers actually in force, after defaults are filled in. */
export interface EffectiveOutreachConfig {
  no_reply_hours: number;
  follow_up1_min_days: number;
  follow_up1_max_days: number;
  follow_up2_min_days: number;
  follow_up2_max_days: number;
  meeting_confirm_min_days: number;
  re_engage_threshold_days: number;
  max_followups: number;
}

export interface OutreachSettingsPayload {
  settings: OutreachSettings;
  effective: EffectiveOutreachConfig;
  defaults: EffectiveOutreachConfig;
  summary: string;
  auto_reply: AutoReplyView;
  /** Served by the server so the form cannot offer an intent it would refuse. */
  auto_reply_selectable_intents: string[];
  /** Merge fields a fixed reply may use, e.g. `first_name` for {{first_name}}. */
  auto_reply_merge_tags: string[];
}

const path = (orgId: string) => `v2/organizations/${orgId}/outreach-settings`;

const OutreachSettingsApi = {
  get: async (orgId = 'me') => {
    const res = await kyClient.get<{ data: OutreachSettingsPayload }>(
      path(orgId)
    );
    return (await res.json()).data;
  },
  update: async (settings: OutreachSettings, orgId = 'me') => {
    const res = await kyClient.put<{ data: OutreachSettingsPayload }>(
      path(orgId),
      { json: settings }
    );
    return (await res.json()).data;
  },
};

export default OutreachSettingsApi;
