import { InboundLeadForm } from '@/network/client/contact-list';

// Contact readiness status
export type ContactStatus = 'ready' | 'pending';

// Context level for outreach
export type ContextLevel = 'LOW' | 'MEDIUM' | 'HIGH';

// Outreach decision types
export type OutreachDecision = 'INTRO' | 'FOLLOW-UP' | 'NURTURE' | 'HOLD';

// Lifecycle stages
export type LifecycleStage =
  | 'new'
  | 'contacted'
  | 'replied'
  | 'qualified'
  | 'proposal'
  | 'won'
  | 'lost'
  | 'meeting'
  | 'dropped';

// Outreach stage (customer status)
export type OutreachStage =
  | 'COLD'
  | 'NO_REPLY'
  | 'REPLIED'
  | 'POST_MEETING'
  | 'DROPPED';

// Next step options (BD actions)
export type NextStep =
  | 'SEND' // Send initial message
  | 'FOLLOW_UP_1' // Follow-up #1 (Day 4-5)
  | 'FOLLOW_UP_2' // Follow-up #2 (Day 9-12)
  | 'SET_MEETING' // Propose meeting
  | 'FOLLOW_UP_MEETING_1' // Meeting confirmation follow-up #1
  | 'FOLLOW_UP_MEETING_2' // Meeting confirmation follow-up #2
  | 'PREPARE_MEETING' // Prepare meeting materials
  | 'WAIT' // Wait for response / wait for meeting day
  | 'FOLLOW_UP' // Follow-up deal (post-meeting)
  | 'DROP'; // Drop contact

// Business stage (pipeline stage)
export type BusinessStage = 'PRE_SALES' | 'SALES' | 'POST_SALES';

export type Contact = {
  name: string;
  email?: string; // Now stored in profile.email
  phone?: string; // Now stored in profile.phone
  company: string;
  job_title: string;
  address: string;
  city: string;
  country: string;
  state: string;
  inbound_lead_form?: InboundLeadForm;
  tags?: Record<string, string>;
  zip: string;
  id: string;
  user_id: string;
  source: string;
  source_id: string;
  updated_at: string;
  hubspot_id: string | null;
  created_at: string;

  // Added by info - tracks which BD added this contact
  added_by_name?: string;
  added_by_email?: string;

  // Status fields - indicates if contact has all required fields
  status?: ContactStatus;
  missing_fields?: string[];

  // Contact information - LinkedIn URL for LinkedIn source, email for others
  contact_information?: string;

  // Outreach context fields
  industry?: string;
  contact_channel?: string;
  lifecycle_stage?: LifecycleStage;
  outreach_stage?: OutreachStage; // Customer outreach status
  context_level?: ContextLevel;
  outreach_decision?: OutreachDecision;
  scenario?: string;
  message_draft?: string;
  last_outcome?: string;
  next_step?: NextStep;
  followup_count?: number; // Number of follow-ups sent
  meeting?: string;

  // Pipeline stage
  business_stage?: BusinessStage;

  ai_insights?: {
    suspected_pain_points?: Array<{
      pain_point: string;
      confidence?: number;
      evidence?: string[];
      detected_at?: string;
      status?: string;
    }>;
    suspected_goals?: Array<{
      goal: string;
      confidence?: number;
      evidence?: string[];
      detected_at?: string;
      status?: string;
    }>;
    buying_signals?: Array<{
      signal: string;
      strength?: string;
      occurred_at?: string;
      recency_score?: number;
    }>;
    decision_style?: {
      type?: string;
      confidence?: number;
      evidence?: string[];
      key_decision_factors?: string[];
    };
    communication_preferences?: {
      preferred_channel?: string;
      best_time_to_contact?: string;
      confidence?: number;
    };
    anticipated_objections?: Array<{
      objection: string;
      likelihood?: number;
      suggested_response?: string;
    }>;
    research_suggestions?: Array<{
      category?: string;
      data_point: string;
      why_important?: string;
      priority?: 'High' | 'Medium' | 'Low';
      where_to_find?: string;
    }>;
  };
  confirmed_facts?: Record<string, any>;
  profile?: {
    email?: string; // Contact email (moved from top-level field)
    phone?: string; // Contact phone (moved from top-level field)
    linkedin_url?: string; // LinkedIn profile URL
    custom_fields?: Record<
      string,
      {
        value: string;
        source?: string;
        updated_at?: string;
      }
    >;
    research_findings?: Array<{
      category: string;
      field_name: string;
      value: string;
      source?: string;
      added_at: string;
      added_by: string;
      priority?: string;
      why_important?: string;
    }>;
  };
};
