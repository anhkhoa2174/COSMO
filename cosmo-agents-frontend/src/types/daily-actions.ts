import type {
  ConversationState,
  NextStep,
  OutreachScenario,
  ContextLevel,
  InteractionChannel,
  Sentiment,
  FeedbackAction,
  Language,
} from '@/network/client/outreach';

// =============================================================================
// ENUMS & LITERALS
// =============================================================================

export type ActionType =
  | 'outreach'
  | 'followup'
  | 'respond'
  | 'meeting_prep'
  | 'enrich';

export type ActionStatus =
  | 'suggested'
  | 'in_progress'
  | 'completed'
  | 'skipped'
  | 'snoozed'
  | 'deferred';

export type ActionTransition =
  | 'mark_sent'
  | 'skip'
  | 'snooze'
  | 'snooze_custom'
  | 'mark_completed'
  | 'reopen';

export type CategoryId =
  | 'replied'
  | 'followup'
  | 'new_outreach'
  | 'meeting_prep'
  | 'enrichment';

/** FR-007: Category display priority order (1 = highest) */
export const CATEGORY_PRIORITY: Record<CategoryId, number> = {
  meeting_prep: 1,
  replied: 2,
  followup: 3,
  new_outreach: 4,
  enrichment: 5,
};

/** Sort categories by priority order */
export function sortCategoriesByPriority<T extends { id: CategoryId }>(
  categories: T[]
): T[] {
  return [...categories].sort(
    (a, b) => (CATEGORY_PRIORITY[a.id] ?? 99) - (CATEGORY_PRIORITY[b.id] ?? 99)
  );
}

export type GenerationStatus =
  | 'not_generated'
  | 'started'
  | 'generating'
  | 'ready'
  | 'stale';

export type IntentAssessment =
  | 'interested'
  | 'requesting_info'
  | 'scheduling_meeting'
  | 'declining'
  | 'unclear';

export type ContactSource = 'LinkedIn' | 'Apollo' | 'Manual' | 'HubSpot';

export type BDMessageType =
  | 'briefing'
  | 'alert'
  | 'update'
  | 'summary'
  | 'conversational'
  | 'user_message';

export type SSEEventType =
  | 'prospect_replied'
  | 'meeting_approaching'
  | 'followup_due'
  | 'snooze_expired'
  | 'generation_complete'
  | 'action_updated';

// =============================================================================
// CORE DATA TYPES
// =============================================================================

export interface ActionContact {
  id: string;
  name: string;
  email?: string;
  company?: string;
  job_title?: string;
  linkedin_url?: string;
  source: ContactSource;
  status: 'ready' | 'pending';
  outreach_stage: ConversationState;
  lifecycle_stage?: string;
  avatar_url?: string;
}

export interface OutreachStateSnapshot {
  conversation_state: ConversationState;
  next_step: NextStep;
  followup_count: number;
  max_followups: number;
}

export interface OutreachActionData {
  draft_message: string;
  scenario: OutreachScenario;
  context_level: ContextLevel;
  company_context?: string;
  followup_number?: number;
  days_since_last_interaction: number;
  previous_messages_count: number;
  last_sent_date?: string;
  is_final_followup: boolean;
  outreach_state: OutreachStateSnapshot;
  // Agentic fields (006-agentic-daily-actions)
  messaging_strategy?: string;
  strategic_reasoning?: string;
  recommended_channel?: string;
  referenced_knowledge?: string[];
  confidence_level?: 'high' | 'medium' | 'low';
  outcome_pattern_cited?: string;
}

export interface MeetingBriefing {
  prospect_profile_summary: string;
  conversation_summary: {
    touchpoint_count: number;
    duration_days: number;
    tone_assessment: string;
    key_topics: string[];
  };
  pain_points: Array<{
    pain_point: string;
    confidence: number;
    evidence: string[];
  }>;
  suggested_agenda: Array<{
    topic: string;
    duration_minutes: number;
    notes?: string;
  }>;
  discovery_questions: string[];
  recommended_next_steps: string[];
  risk_flags: string[];
}

export interface MeetingActionData {
  meeting_id: string;
  meeting_title: string;
  meeting_time: string;
  meeting_duration_minutes: number;
  meeting_channel: string;
  hours_until_meeting: number;
  briefing: MeetingBriefing;
}

export interface EnrichmentActionData {
  missing_fields: string[];
  quality_impact: string;
  contact_status: 'ready' | 'pending';
  suggested_sources: Array<{
    field: string;
    source: string;
    url?: string;
  }>;
}

export interface RespondActionData {
  reply_preview: string;
  reply_timestamp: string;
  reply_channel: InteractionChannel;
  intent_assessment: IntentAssessment;
  intent_reasoning: string;
  recommended_action: string;
  draft_response?: string;
  conversation_context: {
    total_interactions: number;
    days_in_conversation: number;
    last_outgoing_message_preview?: string;
    key_topics_discussed: string[];
  };
}

export interface PriorityFactor {
  factor: string;
  value: number;
  description: string;
}

// =============================================================================
// MAIN ENTITY: DailyAction
// =============================================================================

export interface DailyAction {
  id: string;
  type: ActionType;
  contact: ActionContact;
  priority: number;
  priority_factors?: PriorityFactor[];
  reasoning: string;
  status: ActionStatus;
  status_changed_at?: string;
  snooze_until?: string;
  outreach_data?: OutreachActionData;
  meeting_data?: MeetingActionData;
  enrichment_data?: EnrichmentActionData;
  respond_data?: RespondActionData;
  created_at: string;
  updated_at: string;
}

// =============================================================================
// BRIEFING STRUCTURE
// =============================================================================

export interface MemoryReference {
  contact_id: string;
  contact_name: string;
  event_summary: string;
  event_timestamp: string;
  relevance: string;
}

export interface CategoryBadge {
  category: CategoryId;
  label: string;
  count: number;
  icon: string;
  color: string;
}

export interface AgentBriefing {
  greeting: string;
  strategic_reasoning: string;
  memory_references: MemoryReference[];
  category_counts: CategoryBadge[];
}

export interface ActionCategory {
  id: CategoryId;
  label: string;
  icon: string;
  color: string;
  description: string;
  actions: DailyAction[];
  total_count: number;
  has_more: boolean;
  next_offset?: number;
}

export interface PipelineSummary {
  total_active_contacts: number;
  contacts_by_stage: Record<string, number>;
  contacts_by_lifecycle: Record<string, number>;
  response_rate_7d: number;
  meetings_booked_7d: number;
  avg_response_time_hours: number;
}

export interface ActionProgress {
  total: number;
  completed: number;
  skipped: number;
  snoozed: number;
  remaining: number;
  completion_rate: number;
}

export interface DailyActionsBriefing {
  generation_id: string;
  generation_status: GenerationStatus;
  generated_at?: string;
  date: string;
  language: Language;
  agent_briefing: AgentBriefing;
  categories: ActionCategory[];
  pipeline_summary: PipelineSummary;
  progress: ActionProgress;
}

// =============================================================================
// MESSAGE PARTS (Discriminated Union)
// =============================================================================

export interface AlertData {
  alert_type: 'prospect_replied' | 'meeting_approaching' | 'snooze_expired';
  contact: ActionContact;
  content: string;
  agent_reasoning: string;
  recommended_action: string;
  action_id?: string;
  timestamp: string;
}

export interface DailySummaryData {
  date: string;
  agent_summary: string;
  progress: ActionProgress;
  breakdown: {
    outreach_sent: number;
    followups_sent: number;
    replies_handled: number;
    meetings_prepped: number;
    contacts_enriched: number;
  };
  outcomes: {
    responses_received_today: number;
    meetings_booked_today: number;
    contacts_dropped: number;
  };
  carry_over: Array<{
    action_id: string;
    type: string;
    contact_name: string;
    reason: string;
  }>;
}

/** A contact row returned from agent search queries (FR-029, FR-030) */
export interface ContactListItem {
  id: string;
  name: string;
  company?: string;
  job_title?: string;
  outreach_stage?: string;
  priority?: number;
  status?: 'ready' | 'pending';
  days_since_last_interaction?: number;
  generated_message?: string;
}

/** Contact list result data for chat query responses */
export interface ContactListData {
  title: string;
  icon?: string;
  contacts: ContactListItem[];
  total_count: number;
}

export type MessagePart =
  | { type: 'text'; text: string }
  | { type: 'agent-reasoning'; text: string; memory_refs?: MemoryReference[] }
  | { type: 'category-badges'; badges: CategoryBadge[] }
  | { type: 'action-category'; category: ActionCategory }
  | { type: 'alert'; alert: AlertData }
  | { type: 'summary'; summary: DailySummaryData }
  | { type: 'pipeline-summary'; pipeline_summary: PipelineSummary }
  | { type: 'contact-list'; contact_list: ContactListData };

export interface BDChatMessage {
  id: string;
  role: 'user' | 'assistant' | 'system';
  parts: MessagePart[];
  message_type: BDMessageType;
  created_at: string;
  metadata?: Record<string, unknown>;
}

// =============================================================================
// REQUEST / RESPONSE TYPES
// =============================================================================

export interface GenerateRequest {
  language?: Language;
  force_refresh?: boolean;
}

export interface GenerateResponseData {
  generation_id: string;
  generation_status: 'started' | 'already_in_progress' | 'cached';
}

export interface UpdateActionRequest {
  transition: ActionTransition;
  content?: string;
  channel?: InteractionChannel;
  skip_reason?: string;
  snooze_until?: string;
  feedback_action?: FeedbackAction;
}

export interface UpdateActionResponseData {
  action: DailyAction;
  contact_state_change?: {
    previous_state: string;
    new_state: string;
    new_next_step: string;
  };
  progress: ActionProgress;
}

// =============================================================================
// SSE EVENT TYPES
// =============================================================================

export interface SSEProspectRepliedEvent {
  event_type: 'prospect_replied';
  event_id: string;
  timestamp: string;
  contact: ActionContact;
  reply_preview: string;
  reply_timestamp: string;
  reply_channel: string;
  intent_assessment: IntentAssessment;
  agent_reasoning: string;
  recommended_action: string;
  draft_response?: string;
  action_id: string;
}

export interface SSEMeetingApproachingEvent {
  event_type: 'meeting_approaching';
  event_id: string;
  timestamp: string;
  meeting_id: string;
  meeting_title: string;
  meeting_time: string;
  contact: ActionContact;
  hours_until: number;
  has_prep: boolean;
  agent_message: string;
  action_id: string;
}

export interface SSEFollowupDueEvent {
  event_type: 'followup_due';
  event_id: string;
  timestamp: string;
  contact: ActionContact;
  followup_number: number;
  days_since: number;
  is_final: boolean;
  agent_message: string;
  action_id: string;
}

export interface SSESnoozeExpiredEvent {
  event_type: 'snooze_expired';
  event_id: string;
  timestamp: string;
  actions: DailyAction[];
  agent_message: string;
}

export interface SSEGenerationCompleteEvent {
  event_type: 'generation_complete';
  event_id: string;
  timestamp: string;
  generation_id: string;
  action_count: number;
}

export interface SSEActionUpdatedEvent {
  event_type: 'action_updated';
  event_id: string;
  timestamp: string;
  action_id: string;
  new_status: ActionStatus;
  updated_by: string;
}

export type DailyActionSSEEvent =
  | SSEProspectRepliedEvent
  | SSEMeetingApproachingEvent
  | SSEFollowupDueEvent
  | SSESnoozeExpiredEvent
  | SSEGenerationCompleteEvent
  | SSEActionUpdatedEvent;

// =============================================================================
// MCP ENDPOINT TYPES (from cosmo-agents-mcp)
// =============================================================================

/** GET /v1/mcp/outcome-metrics */
export interface OutcomeMetrics {
  id?: string;
  user_id: string;
  computed_at?: string;
  period: string;
  total_sent: number;
  total_replied: number;
  total_meetings: number;
  reply_rate_overall: number;
  reply_rate_by_channel?: Record<string, number>;
  reply_rate_by_strategy?: Record<string, number>;
  reply_rate_by_industry?: Record<string, number>;
  reply_rate_by_time_of_day?: Record<string, number>;
  avg_messages_to_meeting?: number;
  top_performing_strategies?: string[];
  message?: string; // Present when no metrics computed yet
}

/** GET /v1/mcp/contacts/pipeline */
export interface PipelineContact {
  contact_id: string;
  name: string;
  email: string;
  company: string;
  job_title: string;
  industry: string;
  source: string;
  outreach_stage: string;
  business_stage: string;
  conversation_state: string;
  next_step: string;
  days_since_contact: number;
  followup_count: number;
  context_level: string;
  message_draft?: string;
  type: string;
}

export interface ContactsPipelineResponse {
  contacts: PipelineContact[];
  total: number;
}

/** GET /v1/mcp/contacts/:contact_id/interactions */
export interface ContactInteraction {
  id: string;
  channel: string;
  direction: string;
  content: string;
  subject?: string;
  sentiment?: string;
  timestamp: string;
}

export interface ContactInteractionsResponse {
  contact_id: string;
  contact_name: string;
  interactions: ContactInteraction[];
  total: number;
}

/** GET /v1/mcp/daily-actions/status */
export interface DailyActionsStatusItem {
  action_id: string;
  contact_id: string;
  contact_name: string;
  company: string;
  action_type: string;
  category_id: string;
  priority: number;
  status: string;
  reasoning: string;
}

export interface DailyActionsStatusResponse {
  generation_id?: string;
  date: string;
  status: string;
  actions: DailyActionsStatusItem[];
  total_actions: number;
  completed: number;
  pending: number;
}

/** POST /v2/contacts/batch (update tags) */
export interface UpdateTagsContact {
  email: string;
  tags: Record<string, string>;
}

// =============================================================================
// FRONTEND-ONLY TYPES (UI State)
// =============================================================================

