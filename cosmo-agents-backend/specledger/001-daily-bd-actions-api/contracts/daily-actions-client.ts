// =============================================================================
// Daily BD Actions - Frontend API Client Contract
// =============================================================================
// This file defines the TypeScript types and API client interface that
// correspond to the daily-actions-api.yaml OpenAPI spec.
//
// NOTE: This is a CONTRACT file, not production code. It will be used as
// the reference for implementing the actual client in:
//   src/network/client/daily-actions.ts
//
// The patterns here follow existing conventions from:
//   src/network/client/outreach.ts
//   src/network/client/intelligence.ts
//   src/models/response.ts
// =============================================================================

import type { ApiResponse } from '@/models/response';
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

/** Lightweight contact summary embedded in each action card */
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

/** Snapshot of contact's outreach state at generation time */
export interface OutreachStateSnapshot {
  conversation_state: ConversationState;
  next_step: NextStep;
  followup_count: number;
  max_followups: number;
}

/** Data specific to outreach and follow-up actions */
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
}

/** Meeting briefing document */
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

/** Data specific to meeting prep actions */
export interface MeetingActionData {
  meeting_id: string;
  meeting_title: string;
  meeting_time: string;
  meeting_duration_minutes: number;
  meeting_channel: string;
  hours_until_meeting: number;
  briefing: MeetingBriefing;
}

/** Data specific to data enrichment actions */
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

/** Data specific to "prospect replied" actions */
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

/** Priority factor breakdown */
export interface PriorityFactor {
  factor: string;
  value: number;
  description: string;
}

// =============================================================================
// MAIN ENTITY: DailyAction
// =============================================================================

/** A single suggested action (the "action card" data model) */
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

  // Type-specific data (only one of these is populated per action)
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

/** Agent's strategic analysis (the narrative in the morning briefing) */
export interface AgentBriefing {
  greeting: string;
  strategic_reasoning: string;
  memory_references: Array<{
    contact_id: string;
    contact_name: string;
    event_summary: string;
    event_timestamp: string;
    relevance: string;
  }>;
  category_counts: Array<{
    category: CategoryId;
    label: string;
    count: number;
    icon: string;
    color: string;
  }>;
}

/** A group of actions sharing the same type/urgency */
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

/** Pipeline-level stats */
export interface PipelineSummary {
  total_active_contacts: number;
  contacts_by_stage: Record<string, number>;
  contacts_by_lifecycle: Record<string, number>;
  response_rate_7d: number;
  meetings_booked_7d: number;
  avg_response_time_hours: number;
}

/** Running tally of action completion */
export interface ActionProgress {
  total: number;
  completed: number;
  skipped: number;
  snoozed: number;
  remaining: number;
  completion_rate: number;
}

/** The full daily actions briefing */
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
// API CLIENT INTERFACE
// =============================================================================
// This interface documents the methods the actual client will expose.
// Implementation will use kyClient from '@/lib/ky' following existing patterns.
// =============================================================================

/** Response for loading more actions in a category (FR-021) */
export interface LoadMoreActionsResponseData {
  actions: DailyAction[];
  has_more: boolean;
  next_offset?: number;
}

/** Request body for chat with BD Agent (FR-023) */
export interface ChatRequest {
  messages: Array<{
    role: 'user' | 'assistant' | 'system';
    content: string;
  }>;
  stream?: boolean;
}

export interface DailyActionsApiClient {
  /**
   * Trigger daily actions generation.
   * Call on page open (morning briefing) or explicit refresh.
   * Returns immediately; listen on SSE or poll GET for completion.
   */
  generate: (
    options?: GenerateRequest
  ) => Promise<ApiResponse<GenerateResponseData>>;

  /**
   * Get today's computed daily actions briefing.
   * Cheap read from cache/DB -- no AI computation.
   * Returns generation_status: "not_generated" if no generation has run.
   */
  getDailyActions: (options?: {
    language?: Language;
    include_completed?: boolean;
  }) => Promise<ApiResponse<DailyActionsBriefing>>;

  /**
   * Load more actions for a specific category (FR-021).
   * The initial briefing returns at most 5 actions per category.
   * This endpoint loads subsequent batches using offset-based pagination.
   * Actions are returned sorted by priority.
   */
  loadMoreActions: (
    categoryId: CategoryId,
    offset: number,
    limit?: number
  ) => Promise<ApiResponse<LoadMoreActionsResponseData>>;

  /**
   * Update a single action's state.
   * Used for: Mark Sent, Skip, Snooze, Complete, Reopen.
   * Also records feedback and advances pipeline state when applicable.
   */
  updateAction: (
    actionId: string,
    request: UpdateActionRequest
  ) => Promise<ApiResponse<UpdateActionResponseData>>;

  /**
   * Get daily progress summary.
   * Used for end-of-day wrap-up or on-demand summary.
   */
  getSummary: (language?: Language) => Promise<ApiResponse<DailySummaryData>>;

  /**
   * Send a message to the BD Agent for conversational interaction (FR-023).
   * Supports: action commands, pipeline queries, strategy questions,
   * analytics requests, contact lookups, pasting prospect replies.
   *
   * Returns a streamed SSE response (text/event-stream) following
   * the Vercel AI SDK data protocol.
   */
  chat: (request: ChatRequest) => Promise<Response>;

  /**
   * Open SSE event stream for real-time updates.
   * Returns an EventSource instance the caller must manage.
   *
   * Events: prospect_replied, meeting_approaching, followup_due,
   * snooze_expired, generation_complete, action_updated.
   *
   * The caller should:
   * 1. Open on page mount
   * 2. Close on page unmount
   * 3. Reconnect automatically on error (EventSource does this by default)
   */
  openEventStream: () => EventSource;
}
