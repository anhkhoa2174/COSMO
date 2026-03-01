import type { Agent } from './agent';
import type { EmailIntent } from './email';

export type CampaignPlaybook =
  | 'revive_dormant_leads'
  | 'upsell_to_existing_customers'
  | 'event_invite'
  | 'content_offering'
  | 'webinar_follow_up'
  | 'click_start_from_scratch'
  | 'custom';

export type CampaignStatus = 'active' | 'paused' | 'draft' | 'scheduled' | 'ended';

export type CampaignAction = 'Let AI reply' | 'Assign to a person' | 'Draft an email';

export type CampaignConfig<T = any> = {
  intent_type: EmailIntent;
  who: CampaignAction;
  payload: T;
};

export type CampaignNotification = {
  // id: string;
  user_id: string;
  // created_at: string;
  // campaign_id: string;
  // is_deleted: boolean;
  // updated_at: string;
  // user: User;
};

export type FormInbound = {
  id: string;
  name: string;
  slug: string;
};

export type Campaign = {
  id: string;
  user_id: string;
  playbook: CampaignPlaybook;
  name: string;
  list_contact_id: string | null;
  organization_id: string;
  schedule: string | null;
  created_at: string;
  updated_at: string;
  inbound_lead_forms: FormInbound[];
  status: CampaignStatus;
  agent?: Agent;
  agent_id: string | null;
  notifications: CampaignNotification[];
  templates: { id: string; send_after: number }[];
  draft_templates: { id: string; intent: EmailIntent }[];
  cmetadata: {
    client?: {
      contact_id: string;
    };
    config: CampaignConfig[];
  };
};
